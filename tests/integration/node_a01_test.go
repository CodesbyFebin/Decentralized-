package integration

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/cli"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/node"
)

// NODE-A01: the sovereign node foundation, checked adversarially against
// real processes. One host joins with an invite that needs the owner's
// approval; every step of its enrollment, identity, facts, freshness and
// revocation is exercised, and every forgery or replay must be refused.
//
// Command-level rejections on the host (replayed, stale, wrong signer, wrong
// target, expired, revoked) are exercised in-process by
// pkg/node.TestCompromisedControlPlaneCannotCommandHost, which can sign
// bundles as a compromised member; this test covers the host↔plane wire.
func TestNodeA01SovereignNode(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 0})
	cp := c.Proc("cp-1")
	cluster := c.Op.Cfg.Cluster

	tok, err := c.Op.Invite(cli.InviteOpts{Note: "node-a01"}) // no auto-approval; default TTL
	if err != nil {
		t.Fatal(err)
	}
	jt, _ := node.DecodeJoinToken(tok)
	joinCap, _ := capability.Decode(jt.Capability)
	var blk capability.Block
	_ = joinCap.Blocks[0].Decode(&blk)
	if ttl := time.Until(time.UnixMilli(blk.Caveats.Expires)); ttl > 16*time.Minute || ttl <= 0 {
		t.Fatalf("default join token lifetime is %s, want at most 15m", ttl)
	}

	const declared = "1Gi"
	if _, err := c.StartHostWithToken(0, "node-a01", tok, "--mem", declared); err != nil {
		t.Fatal(err)
	}
	host := c.Proc("node-a01")
	var id string

	t.Run("identity is generated on the host and the host waits for approval", func(t *testing.T) {
		keyPath := filepath.Join(host.Data, "identity", "identity.key")
		deadline := time.Now().Add(20 * time.Second)
		for {
			if _, err := os.Stat(keyPath); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("no identity key on the host")
			}
			time.Sleep(200 * time.Millisecond)
		}
		st, _ := os.Stat(keyPath)
		if st.Mode().Perm() != 0o600 {
			t.Fatalf("identity key mode %v, want 0600", st.Mode().Perm())
		}
		me, err := identity.Load(filepath.Join(host.Data, "identity"))
		if err != nil {
			t.Fatal(err)
		}
		id = me.ID
		if err := c.WaitFor(20*time.Second, "host enrolled as pending", func(v *control.View) bool {
			n := nodeByName(v, "node-a01")
			return n != nil && n.ID == id && n.Status == "pending"
		}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		v, _ := c.View()
		if n := nodeByName(v, "node-a01"); n.Status != "pending" || n.ApprovedAt != 0 {
			t.Fatalf("host became %s without the owner's approval", n.Status)
		}
	})
	if id == "" {
		t.FailNow()
	}

	t.Run("the private key never reaches the control plane", func(t *testing.T) {
		me, _ := identity.Load(filepath.Join(host.Data, "identity"))
		seed := me.Priv.Seed()
		pem := string(me.PrivatePEM())
		pemBody := strings.Join(strings.Split(strings.TrimSpace(pem), "\n")[1:2], "")
		needles := map[string][]byte{
			"raw seed":        seed,
			"raw private key": me.Priv,
			"hex seed":        []byte(hex.EncodeToString(seed)),
			"base64 seed":     []byte(base64.StdEncoding.EncodeToString(seed)),
			"base64url seed":  []byte(base64.RawURLEncoding.EncodeToString(seed)),
			"base64 key":      []byte(base64.StdEncoding.EncodeToString(me.Priv)),
			"PEM body":        []byte(pemBody),
		}
		var haystacks []string
		scanned := 0
		_ = filepath.WalkDir(cp.Data, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			scanned++
			for what, n := range needles {
				if bytes.Contains(b, n) {
					t.Errorf("%s found in control-plane file %s", what, p)
				}
			}
			return nil
		})
		for _, path := range []string{"/api/v1/state", "/api/v1/view", "/api/v1/audit?limit=5000"} {
			var raw json.RawMessage
			if err := c.Op.Do("GET", path, nil, &raw); err != nil {
				t.Fatal(err)
			}
			haystacks = append(haystacks, string(raw))
		}
		logs, _ := os.ReadFile(cp.Log)
		haystacks = append(haystacks, string(logs))
		for _, h := range haystacks {
			for what, n := range needles {
				if strings.Contains(h, string(n)) {
					t.Errorf("%s found in control-plane API output or log", what)
				}
			}
		}
		if scanned < 3 {
			t.Fatalf("scanned only %d control-plane files", scanned)
		}
		t.Logf("scanned %d control-plane files, 3 API documents and the log for %d encodings of the host key", scanned, len(needles))
	})

	enroll := func(signer *identity.Identity, e api.Enroll) (int, string) {
		env, err := envelope.Sign(signer, "", envelope.KindEnroll, e)
		if err != nil {
			t.Fatal(err)
		}
		return postRaw(t, cp.API, "/v1/enroll", env)
	}
	newEnroll := func(who *identity.Identity, name, token string) api.Enroll {
		return api.Enroll{ID: who.ID, Name: name, Pub: who.PubString(), OS: goruntime.GOOS, Arch: goruntime.GOARCH, Tiers: []string{"trusted"},
			Roles: []string{}, Features: []string{}, JoinToken: token, TS: time.Now().UnixMilli()}
	}
	refused := func(t *testing.T, code int, body, want string) {
		t.Helper()
		if code < 400 || !strings.Contains(body, want) {
			t.Fatalf("expected refusal containing %q, got %d %s", want, code, body)
		}
	}

	t.Run("the join token admits exactly one identity", func(t *testing.T) {
		thief, _ := identity.Generate()
		code, body := enroll(thief, newEnroll(thief, "thief", jt.Capability))
		refused(t, code, body, "already used")
	})

	t.Run("a join token not signed by the cluster root is refused", func(t *testing.T) {
		fake, _ := identity.Generate()
		forged, _ := capability.Mint(fake, capability.Caveats{Actions: []string{"node.join"}, Resources: []string{"cluster/" + cluster},
			Expires: time.Now().Add(time.Hour).UnixMilli(), Nonce: randomHex(16)}, "", "join")
		who, _ := identity.Generate()
		code, body := enroll(who, newEnroll(who, "forger", forged.Encode()))
		refused(t, code, body, "join token")
	})

	t.Run("an expired join token is refused", func(t *testing.T) {
		short, err := c.Op.Invite(cli.InviteOpts{TTL: 300 * time.Millisecond, Note: "expires"})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		sj, _ := node.DecodeJoinToken(short)
		who, _ := identity.Generate()
		code, body := enroll(who, newEnroll(who, "late", sj.Capability))
		refused(t, code, body, "expired")
	})

	t.Run("a join token the owner revoked is refused", func(t *testing.T) {
		tok2, err := c.Op.Invite(cli.InviteOpts{Note: "to revoke"})
		if err != nil {
			t.Fatal(err)
		}
		r, err := c.Op.RevokeInvite(tok2)
		if err != nil || !r.OK {
			t.Fatalf("revoke invite: %+v %v", r, err)
		}
		j2, _ := node.DecodeJoinToken(tok2)
		who, _ := identity.Generate()
		code, body := enroll(who, newEnroll(who, "withdrawn", j2.Capability))
		refused(t, code, body, "revoked")
		// And a used token cannot be "revoked" to hide the host it admitted.
		if r, err := c.Op.RevokeInvite(tok); err == nil && r.OK {
			t.Fatal("revoking a consumed invite succeeded; the host it admitted must be revoked instead")
		}
	})

	t.Run("an enrollment signed by a key other than the enrolling identity is refused", func(t *testing.T) {
		who, _ := identity.Generate()
		other, _ := identity.Generate()
		e := newEnroll(who, "mismatch", jt.Capability)
		env, _ := envelope.Sign(other, "", envelope.KindEnroll, e)
		code, body := postRaw(t, cp.API, "/v1/enroll", env)
		refused(t, code, body, "mismatch")
	})

	t.Run("a replayed enrollment cannot roll the host's record back", func(t *testing.T) {
		var st struct {
			Nodes map[string]struct {
				EnrollEnv *envelope.Envelope `json:"enrollEnv"`
			} `json:"nodes"`
		}
		if err := c.Op.Do("GET", "/api/v1/state", nil, &st); err != nil {
			t.Fatal(err)
		}
		env := st.Nodes[id].EnrollEnv
		if env == nil {
			t.Fatal("no stored enrollment")
		}
		code, body := postRaw(t, cp.API, "/v1/enroll", env)
		refused(t, code, body, "not newer")
	})

	t.Run("facts are measured, not the declared capacity", func(t *testing.T) {
		var n *control.NodeView
		if err := c.WaitFor(20*time.Second, "facts observed", func(v *control.View) bool {
			n = nodeByName(v, "node-a01")
			return n != nil && n.Facts != nil
		}); err != nil {
			t.Fatal(err)
		}
		if n.MemBytes != 1<<30 {
			t.Fatalf("declared memory %d, want 1Gi", n.MemBytes)
		}
		f := n.Facts
		if goruntime.GOOS == "linux" {
			if want := memTotal(t); f.MemBytes != want {
				t.Fatalf("facts.memBytes %d, /proc/meminfo says %d", f.MemBytes, want)
			}
			if f.MemBytes == n.MemBytes {
				t.Fatal("facts.memBytes equals the declared capacity; suspicious")
			}
		}
		if f.CPUs != int64(goruntime.NumCPU()) || f.DataFS == nil || f.DataFS.TotalBytes <= 0 {
			t.Fatalf("facts: %+v", f)
		}
		natUnknown := false
		for _, u := range f.Unknown {
			natUnknown = natUnknown || u == "natType"
		}
		if !natUnknown {
			t.Fatalf("NAT type is not measured but not listed as unknown: %v", f.Unknown)
		}
		t.Logf("facts: cpus=%d model=%q cores=%d mem=%d swap=%d disks=%d gpus=%d unknown=%v", f.CPUs, f.CPUModel, f.PhysicalCores, f.MemBytes, f.SwapBytes, len(f.Disks), len(f.GPUs), f.Unknown)
	})

	t.Run("owner approval makes the host ACTIVE and FRESH", func(t *testing.T) {
		if err := c.Op.Do("POST", "/api/v1/nodes/"+id+"/approve", nil, nil); err != nil {
			t.Fatal(err)
		}
		if err := c.WaitFor(20*time.Second, "host ready and fresh", func(v *control.View) bool {
			n := nodeByName(v, "node-a01")
			return n.Status == "ready" && n.Identity == "ACTIVE" && n.LastObs.Freshness == "FRESH" && n.Health == "live"
		}); err != nil {
			t.Fatal(err)
		}
		st, err := c.HostStatus("node-a01")
		if err != nil || !st.Trusted || !st.Fresh || st.Mode != "normal" {
			t.Fatalf("host's own view: %+v %v", st, err)
		}
	})

	t.Run("heartbeats are authenticated: replay and foreign keys are refused", func(t *testing.T) {
		env := committedObservation(t, c, "node-a01")
		var res struct {
			Results []control.ObserveResult `json:"results"`
		}
		postHost(t, c, "/v1/observe", map[string]any{"observations": []*envelope.Envelope{env}}, &res)
		if res.Results[0].Status != "rejected" || !strings.Contains(res.Results[0].Reason, "replay") {
			t.Fatalf("replayed heartbeat: %+v", res.Results)
		}
		var o api.Observation
		_ = env.Decode(&o)
		o.Seq += 1_000_000
		imp, _ := identity.Generate()
		forged, _ := envelope.Sign(imp, env.Signer, envelope.KindObservation, o)
		postHost(t, c, "/v1/observe", map[string]any{"observations": []*envelope.Envelope{forged}}, &res)
		if res.Results[0].Status != "rejected" || !strings.Contains(res.Results[0].Reason, "wrong host key") {
			t.Fatalf("forged heartbeat: %+v", res.Results)
		}
	})

	t.Run("coordinator disconnect: the host stops claiming freshness, then reconciles", func(t *testing.T) {
		if err := pause(c, "cp-1", syscall.SIGSTOP); err != nil {
			t.Fatal(err)
		}
		resumed := false
		defer func() {
			if !resumed {
				_ = pause(c, "cp-1", syscall.SIGCONT)
			}
		}()
		if err := waitHost(c, "node-a01", 25*time.Second, func(s *node.Status) bool { return s.Mode == "offline-hold" && !s.Fresh }); err != nil {
			t.Fatal(err)
		}
		_ = pause(c, "cp-1", syscall.SIGCONT)
		resumed = true
		if err := waitHost(c, "node-a01", 25*time.Second, func(s *node.Status) bool { return s.Mode == "normal" && s.Fresh && s.Trusted }); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("host disconnect: the plane goes STALE then lost, never a fabricated live state", func(t *testing.T) {
		if err := pause(c, "node-a01", syscall.SIGSTOP); err != nil {
			t.Fatal(err)
		}
		resumed := false
		defer func() {
			if !resumed {
				_ = pause(c, "node-a01", syscall.SIGCONT)
			}
		}()
		sawStale, sawLost := false, false
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) && !sawLost {
			v, err := c.View()
			if err == nil {
				n := nodeByName(v, "node-a01")
				if n.LastObs.Freshness == "STALE" {
					sawStale = true
				}
				switch {
				case n.Health == "lost":
					sawLost = true
					if n.LastObs.Freshness == "FRESH" {
						t.Fatal("lost host reported with FRESH evidence")
					}
				case n.LastObs.Freshness == "FRESH" && n.LastObs.AgeMs >= 10_000:
					t.Fatalf("FRESH claimed for evidence %dms old", n.LastObs.AgeMs)
				}
			}
			time.Sleep(300 * time.Millisecond)
		}
		if !sawStale || !sawLost {
			t.Fatalf("stale=%v lost=%v", sawStale, sawLost)
		}
		_ = pause(c, "node-a01", syscall.SIGCONT)
		resumed = true
		if err := c.WaitFor(30*time.Second, "host live and fresh again", func(v *control.View) bool {
			n := nodeByName(v, "node-a01")
			return n.Health == "live" && n.LastObs.Freshness == "FRESH"
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("audit tampering is detected", func(t *testing.T) {
		var out struct {
			Entries []audit.Entry `json:"entries"`
		}
		if err := c.Op.Do("GET", "/api/v1/audit?limit=5000", nil, &out); err != nil {
			t.Fatal(err)
		}
		if b := audit.Verify(out.Entries, 0, audit.Genesis); b != nil {
			t.Fatal(b)
		}
		var actions []string
		for _, e := range out.Entries {
			actions = append(actions, e.Action)
		}
		all := strings.Join(actions, ",")
		for _, want := range []string{"node-invite", "enroll", "node-approve", "node-invite-revoke"} {
			if !strings.Contains(all, want) {
				t.Fatalf("audit lacks %s: %s", want, all)
			}
		}
		out.Entries[1].Detail = "rewritten"
		if b := audit.Verify(out.Entries, 0, audit.Genesis); b == nil || b.Reason != audit.ReasonHashMismatch {
			t.Fatalf("tamper not detected: %+v", b)
		}
	})

	t.Run("host ledger tampering is detected by the host itself", func(t *testing.T) {
		if err := c.Kill("node-a01", syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(15 * time.Second)
		for c.Alive("node-a01") && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		jp := filepath.Join(host.Data, "journal.jsonl")
		b, err := os.ReadFile(jp)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if len(lines) < 3 {
			t.Fatalf("host ledger has %d entries", len(lines))
		}
		var e map[string]any
		if err := json.Unmarshal([]byte(lines[1]), &e); err != nil {
			t.Fatal(err)
		}
		e["detail"] = "rewritten on disk"
		nb, _ := json.Marshal(e)
		lines[1] = string(nb)
		if err := os.WriteFile(jp, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := c.Restart("node-a01"); err != nil {
			t.Fatal(err)
		}
		if err := waitHost(c, "node-a01", 20*time.Second, func(s *node.Status) bool { return s.LedgerBreak != nil && s.Mode == "ledger-corrupt" }); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a revoked key is refused on every host channel", func(t *testing.T) {
		me, _ := identity.Load(filepath.Join(host.Data, "identity"))
		if err := c.Op.Do("POST", "/api/v1/nodes/"+id+"/revoke-key", map[string]string{"pub": me.PubString(), "reason": "node-a01"}, nil); err != nil {
			t.Fatal(err)
		}
		// A heartbeat signed with the revoked key, newer than anything accepted.
		env := committedObservation(t, c, "node-a01")
		var o api.Observation
		_ = env.Decode(&o)
		o.Seq += 2_000_000
		signed, _ := envelope.Sign(me, id, envelope.KindObservation, o)
		var res struct {
			Results []control.ObserveResult `json:"results"`
		}
		postHost(t, c, "/v1/observe", map[string]any{"observations": []*envelope.Envelope{signed}}, &res)
		if res.Results[0].Status != "rejected" {
			t.Fatalf("heartbeat with a revoked key accepted: %+v", res.Results)
		}
		// A bundle request signed with it.
		req, _ := envelope.Sign(me, id, envelope.KindRequest, control.HostRequest{Node: id, TS: time.Now().UnixMilli(), Op: "bundle"})
		code, body := postRaw(t, cp.API, "/v1/bundle", req)
		if code != 401 {
			t.Fatalf("bundle request with a revoked key: %d %s", code, body)
		}
		if err := c.WaitFor(20*time.Second, "host marked revoked or lost", func(v *control.View) bool {
			n := nodeByName(v, "node-a01")
			return n.LastObs.Freshness != "FRESH"
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func postRaw(t *testing.T, addr, path string, body any) (int, string) {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post("http://"+addr+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data)
}

// pause sends SIGSTOP or SIGCONT; devcluster's Kill waits for an exit.
func pause(c *devcluster.Cluster, name string, sig syscall.Signal) error {
	p := c.Proc(name)
	if p == nil {
		return fmt.Errorf("no process %s", name)
	}
	return syscall.Kill(p.PID, sig)
}

func waitHost(c *devcluster.Cluster, name string, timeout time.Duration, cond func(*node.Status) bool) error {
	deadline := time.Now().Add(timeout)
	var last *node.Status
	for time.Now().Before(deadline) {
		if s, err := c.HostStatus(name); err == nil {
			last = s
			if cond(s) {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if last != nil {
		return fmt.Errorf("%s: condition not met; mode=%s fresh=%v trusted=%v detail=%s", name, last.Mode, last.Fresh, last.Trusted, last.ModeDetail)
	}
	return fmt.Errorf("%s: status never readable", name)
}

func memTotal(t *testing.T) int64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "MemTotal:") {
			n, _ := strconv.ParseInt(strings.Fields(l)[1], 10, 64)
			return n * 1024
		}
	}
	t.Fatal("no MemTotal")
	return 0
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = cryptorand.Read(b)
	return hex.EncodeToString(b)
}
