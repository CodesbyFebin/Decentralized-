package integration

import (
	"crypto/rand"
	"encoding/json"
	"strings"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/node"
)

// M1: two sovereign hosts, signed desired state, admission, signed
// observations, generation and replay protection, revocation, freeze and
// control-plane outage, verified audit.
func TestM1SovereignRuntime(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 2, Chaos: true})
	deploy(t, c, beacon(2))
	v := waitRunning(t, c, "beacon", 2, 60*time.Second)
	a := app(v, "beacon")
	if a.Desired != 2 || a.Admitted != 2 || a.Observed != 2 {
		t.Fatalf("desired/admitted/observed = %d/%d/%d", a.Desired, a.Admitted, a.Observed)
	}
	nodes := map[string]bool{}
	pids := map[string]int64{}
	for _, r := range a.Rows {
		nodes[r.Node] = true
		pids[r.Assignment] = r.PID
		if r.Admitted != "ADMITTED" || r.Observed != "RUNNING" || r.Code != "ADMITTED" || len(r.Checks) < 8 {
			t.Fatalf("replica %d: %+v", r.Replica, r)
		}
	}
	if len(nodes) != 2 {
		t.Fatal("hard anti-affinity: both replicas on one host")
	}

	t.Run("identical desired state does not advance the generation", func(t *testing.T) {
		gen := app(v, "beacon").Generation
		r, err := c.Op.Apply([]byte(strings.Replace(beacon(2), "%s", a.Image, 1)))
		if err != nil {
			t.Fatal(err)
		}
		if r.Code != "UNCHANGED" {
			t.Fatalf("expected UNCHANGED, got %+v", r)
		}
		v2, _ := c.View()
		if app(v2, "beacon").Generation != gen {
			t.Fatal("generation advanced for identical manifest")
		}
	})

	t.Run("replayed observation is rejected with the sequence numbers", func(t *testing.T) {
		env := committedObservation(t, c, "host-a")
		var res struct {
			Results []control.ObserveResult `json:"results"`
		}
		postHost(t, c, "/v1/observe", map[string]any{"observations": []*envelope.Envelope{env}}, &res)
		if len(res.Results) != 1 || res.Results[0].Status != "rejected" || !strings.Contains(res.Results[0].Reason, "replay detected") {
			t.Fatalf("replay not rejected: %+v", res.Results)
		}
	})

	t.Run("observation signed with another key is rejected", func(t *testing.T) {
		env := committedObservation(t, c, "host-a")
		var o api.Observation
		_ = env.Decode(&o)
		o.Seq += 1_000_000
		imp, _ := identity.Generate()
		forged, _ := envelope.Sign(imp, env.Signer, envelope.KindObservation, o)
		var res struct {
			Results []control.ObserveResult `json:"results"`
		}
		postHost(t, c, "/v1/observe", map[string]any{"observations": []*envelope.Envelope{forged}}, &res)
		if res.Results[0].Status != "rejected" || !strings.Contains(res.Results[0].Reason, "wrong host key") {
			t.Fatalf("wrong-key observation accepted: %+v", res.Results)
		}
	})

	t.Run("tampered observation field is rejected", func(t *testing.T) {
		env := committedObservation(t, c, "host-a")
		env.Payload = []byte(strings.Replace(string(env.Payload), `"observed":"running"`, `"observed":"stopped"`, 1))
		var o api.Observation
		_ = json.Unmarshal(env.Payload, &o)
		var res struct {
			Results []control.ObserveResult `json:"results"`
		}
		postHost(t, c, "/v1/observe", map[string]any{"observations": []*envelope.Envelope{env}}, &res)
		if res.Results[0].Status != "rejected" {
			t.Fatalf("tampered observation accepted: %+v", res.Results)
		}
	})

	t.Run("freeze holds admitted work and refuses new work", func(t *testing.T) {
		var r struct{ OK bool }
		if err := c.Op.Do("POST", "/api/v1/freeze", map[string]bool{"frozen": true}, &r); err != nil {
			t.Fatal(err)
		}
		if err := c.WaitFor(20*time.Second, "hosts report frozen-hold", func(v *control.View) bool {
			return nodeByName(v, "host-a").Mode == "frozen-hold" && nodeByName(v, "host-b").Mode == "frozen-hold"
		}); err != nil {
			t.Fatal(err)
		}
		// Scale up while frozen: nothing new is signed.
		if err := c.Op.Do("POST", "/api/v1/apps/beacon/scale", map[string]int64{"replicas": 3}, nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		v, _ := c.View()
		fa := app(v, "beacon")
		if len(fa.Rows) != 2 {
			t.Fatalf("frozen control plane signed new work: %d rows", len(fa.Rows))
		}
		for _, row := range fa.Rows {
			if row.Observed != "RUNNING" || row.PID != pids[row.Assignment] {
				t.Fatalf("admitted workload disturbed by freeze: %+v", row)
			}
		}
		_ = c.Op.Do("POST", "/api/v1/apps/beacon/scale", map[string]int64{"replicas": 2}, nil)
		_ = c.Op.Do("POST", "/api/v1/freeze", map[string]bool{"frozen": false}, nil)
	})

	t.Run("control-plane outage is not workload outage", func(t *testing.T) {
		if err := c.Kill("cp-1", syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(20 * time.Second)
		for {
			st, err := c.HostStatus("host-a")
			if err == nil && st.Mode == "offline-hold" {
				for id, ad := range st.Admitted {
					if ad.Inst.PID != pids[id] || ad.LastState != "running" {
						t.Fatalf("workload %s disturbed: %+v", id, ad)
					}
				}
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("host did not enter offline-hold: %+v %v", st, err)
			}
			time.Sleep(300 * time.Millisecond)
		}
		time.Sleep(3 * time.Second) // let observations buffer
		if err := c.Restart("cp-1"); err != nil {
			t.Fatal(err)
		}
		v := waitRunning(t, c, "beacon", 2, 40*time.Second)
		for _, row := range app(v, "beacon").Rows {
			if row.PID != pids[row.Assignment] {
				t.Fatalf("pid changed across control-plane outage: %d → %d", pids[row.Assignment], row.PID)
			}
		}
		if err := c.WaitFor(15*time.Second, "buffered observations flushed", func(v *control.View) bool {
			for _, e := range v.Audit.Entries {
				if e.Action == "host-mode" && strings.Contains(e.Detail, "offline-hold") {
					return true
				}
			}
			return false
		}); err != nil {
			for _, h := range []string{"host-a", "host-b"} {
				for _, l := range c.HostLedgerTail(h, 12) {
					t.Logf("%s journal %s", h, l)
				}
			}
			t.Fatal(err)
		}
	})

	t.Run("revocation blocks new work without killing admitted work", func(t *testing.T) {
		id, _ := c.Op.ResolveNode("host-b")
		if err := c.Op.Do("POST", "/api/v1/nodes/"+id+"/revoke", map[string]string{"reason": "test"}, nil); err != nil {
			t.Fatal(err)
		}
		if err := c.WaitFor(20*time.Second, "revoked host holds its admitted workload", func(v *control.View) bool {
			n := nodeByName(v, "host-b")
			if n.Identity != "REVOKED" || n.NewAdmission != "BLOCKED" {
				return false
			}
			for _, r := range app(v, "beacon").Rows {
				if r.Node == id && r.Observed == "RUNNING" && r.PID == pids[r.Assignment] {
					return true
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
		// The host itself must decide: hold the admitted generation, refuse new work.
		deadline := time.Now().Add(15 * time.Second)
		for {
			st, _ := c.HostStatus("host-b")
			held := false
			var codes []string
			for id, d := range st.Decisions {
				codes = append(codes, id+"="+d.Code)
				if d.Hold && d.Code == "HOST_REVOKED" {
					held = true
				}
			}
			if held {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("host-b did not record a revocation hold: %v", codes)
			}
			time.Sleep(300 * time.Millisecond)
		}
	})

	t.Run("audit chain verifies and tampering is detected offline", func(t *testing.T) {
		v, _ := c.View()
		if !v.Audit.Verification.OK {
			t.Fatalf("audit does not verify: %+v", v.Audit.Verification)
		}
		var out struct {
			Entries []audit.Entry `json:"entries"`
		}
		if err := c.Op.Do("GET", "/api/v1/audit?limit=5000", nil, &out); err != nil {
			t.Fatal(err)
		}
		if b := audit.Verify(out.Entries, 0, audit.Genesis); b != nil {
			t.Fatal(b)
		}
		out.Entries[3].Detail = "rewritten by an attacker"
		b := audit.Verify(out.Entries, 0, audit.Genesis)
		if b == nil || b.Reason != audit.ReasonHashMismatch || b.Seq != 4 {
			t.Fatalf("tamper not named: %+v", b)
		}
	})

	t.Run("host ledger is authoritative, chained and signed", func(t *testing.T) {
		st, err := c.HostStatus("host-a")
		if err != nil || st.LedgerBreak != nil || st.Ledger < 3 {
			t.Fatalf("host ledger: %+v %v", st, err)
		}
	})
}

func committedObservation(t *testing.T, c *devcluster.Cluster, host string) *envelope.Envelope {
	t.Helper()
	var st struct {
		Nodes map[string]struct {
			Name   string             `json:"name"`
			ObsEnv *envelope.Envelope `json:"obsEnv"`
		} `json:"nodes"`
	}
	if err := c.Op.Do("GET", "/api/v1/state", nil, &st); err != nil {
		t.Fatal(err)
	}
	for _, n := range st.Nodes {
		if n.Name == host && n.ObsEnv != nil {
			return n.ObsEnv
		}
	}
	t.Fatalf("no committed observation for %s", host)
	return nil
}

// postHost posts to the host channel of the first member.
func postHost(t *testing.T, c *devcluster.Cluster, path string, body, out any) {
	t.Helper()
	cl := &hostClient{addr: c.Procs[0].API}
	if err := cl.post(path, body, out); err != nil {
		t.Fatal(err)
	}
}

var _ = node.ChaosRequest{}
var _ = rand.Reader
