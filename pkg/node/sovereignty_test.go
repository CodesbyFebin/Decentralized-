package node

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/storage"
)

// fakeCP is a control plane controlled by the test. It holds a genuine,
// root-delegated member key — i.e. it models a compromised control plane —
// and serves whatever bundle the test crafts.
type fakeCP struct {
	t      *testing.T
	root   *identity.Identity
	member *identity.Identity
	roster *envelope.Envelope
	deleg  string
	cas    *storage.CAS
	mu     sync.Mutex
	bundle *envelope.Envelope
	obs    []api.Observation
	srv    *httptest.Server
}

func newFakeCP(t *testing.T) *fakeCP {
	f := &fakeCP{t: t}
	f.root, _ = identity.Generate()
	f.member, _ = identity.Generate()
	tok, _ := capability.Mint(f.root, capability.Caveats{Actions: []string{"workload.*"}, Resources: []string{"*"}}, f.member.PubString(), "member")
	f.deleg = tok.Encode()
	f.roster = envelope.MustSign(f.root, envelope.KindRoster, api.Roster{Cluster: "t", Version: 1, Root: f.root.PubString(),
		Members: []api.Member{{ID: f.member.ID, Pub: f.member.PubString(), Delegation: f.deleg}}})
	f.cas, _ = storage.OpenCAS(t.TempDir(), 0)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/enroll", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"ok":true,"message":"enrolled"}`)) })
	mux.HandleFunc("POST /v1/bundle", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"bundle": f.bundle})
	})
	mux.HandleFunc("POST /v1/observe", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Observations []*envelope.Envelope `json:"observations"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		for _, env := range body.Observations {
			var o api.Observation
			env.Decode(&o)
			f.obs = append(f.obs, o)
		}
		f.mu.Unlock()
		w.Write([]byte(`{"results":[]}`))
	})
	mux.HandleFunc("POST /v1/chunk", func(w http.ResponseWriter, r *http.Request) {
		var env envelope.Envelope
		json.NewDecoder(r.Body).Decode(&env)
		var req hostRequest
		env.Decode(&req)
		data, err := f.cas.Get(req.Arg)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		w.Write(data)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{}`)) })
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// artifact stores an executable and returns its location record.
func (f *fakeCP) artifact(path, name string) (string, api.ArtifactLocation) {
	data, err := os.ReadFile(path)
	if err != nil {
		f.t.Fatal(err)
	}
	ids, n, _ := storage.PutStream(f.cas, strings.NewReader(string(data)))
	digest := storage.ChunkID(data)
	mb := canon.MustMarshal(api.ArtifactManifest{Digest: digest, Name: name, Bytes: n, Chunks: ids})
	mid, _ := f.cas.Put(mb)
	return name + "@" + digest, api.ArtifactLocation{Digest: digest, Name: name, Manifest: mid, Bytes: n}
}

type bundleOpts struct {
	signer    *identity.Identity
	roster    *envelope.Envelope
	index     int64
	assign    []*envelope.Envelope
	artifacts []api.ArtifactLocation
	attest    []*envelope.Envelope
	revoked   []string
}

func (f *fakeCP) serve(node string, o bundleOpts) {
	if o.signer == nil {
		o.signer = f.member
	}
	if o.roster == nil {
		o.roster = f.roster
	}
	b := api.Bundle{Cluster: "t", Root: f.root.PubString(), Roster: o.roster, Issuer: o.signer.ID, Issued: time.Now().UnixMilli(), StateIndex: o.index,
		Node: api.BundleNode{ID: node, Status: "ready"}, Assignments: o.assign, Artifacts: o.artifacts, Attestations: o.attest, Revoked: o.revoked,
		Peers: []api.Peer{}, Volumes: []api.VolumeDuty{}, Services: []api.Service{}, RevokedKeys: []string{}, Publishers: []string{f.root.PubString()}}
	env, err := envelope.Sign(o.signer, "", envelope.KindBundle, b)
	if err != nil {
		f.t.Fatal(err)
	}
	f.mu.Lock()
	f.bundle = env
	f.mu.Unlock()
}

type assignOpts struct {
	id, node, image, digest, runtime string
	gen                              int64
	cpu                              int64
	tiers                            []string
	capDigest                        string
	capCPU                           int64
	selfAnchored                     bool
	signer                           *identity.Identity
	desired                          string
}

func (f *fakeCP) assignment(o assignOpts) *envelope.Envelope {
	if o.runtime == "" {
		o.runtime = "process"
	}
	if o.cpu == 0 {
		o.cpu = 100
	}
	if o.tiers == nil {
		o.tiers = []string{"trusted"}
	}
	if o.capDigest == "" {
		o.capDigest = o.digest
	}
	if o.capCPU == 0 {
		o.capCPU = o.cpu
	}
	if o.desired == "" {
		o.desired = "running"
	}
	if o.signer == nil {
		o.signer = f.member
	}
	now := time.Now().UnixMilli()
	cav := capability.Caveats{Actions: []string{"workload.admit"}, Resources: []string{"app/" + o.id}, Audience: o.node, Generation: o.gen,
		Digest: o.capDigest, CPUMaxMilli: o.capCPU, MemMaxBytes: 64 << 20, Expires: now + 3600_000}
	var tok *capability.Token
	if o.selfAnchored {
		tok, _ = capability.Mint(f.member, cav, "", "self-anchored")
	} else {
		del, _ := capability.Decode(f.deleg)
		tok, _ = del.Attenuate(f.member, cav, "", "assignment")
	}
	a := api.Assignment{ID: o.id, App: strings.Split(o.id, "/")[0], Node: o.node, Generation: o.gen, Desired: o.desired, Runtime: o.runtime,
		Image: o.image, Digest: o.digest, Command: []string{"60"}, Env: map[string]string{}, Resources: api.Resources{CPUMilli: o.cpu, MemBytes: 32 << 20},
		Ports: []api.Port{}, Volumes: []api.AssignedVolume{}, Tiers: o.tiers, Capability: tok.Encode(), Issued: now}
	return envelope.MustSign(o.signer, envelope.KindAssignment, a)
}

func newHost(t *testing.T, f *fakeCP, policyYAML string) *Agent {
	dir := t.TempDir()
	if policyYAML != "" {
		os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(policyYAML), 0o600)
	}
	join := EncodeJoinToken(JoinToken{Cluster: "t", Root: f.root.PubString(), Endpoints: []string{strings.TrimPrefix(f.srv.URL, "http://")}, Capability: "dhcap1.unused"})
	a, err := New(Config{DataDir: dir, JoinToken: join, Name: "h", CPUMilli: 4000, MemBytes: 4 << 30, Logger: log.New(io.Discard, "", 0)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, ad := range a.st.Admitted {
			if ad.Inst.PID > 0 {
				syscall.Kill(-int(ad.Inst.PID), syscall.SIGKILL)
			}
		}
	})
	return a
}

// settle runs ticks until the artifact fetch (asynchronous) completes.
func settle(a *Agent, n int) {
	for i := 0; i < n; i++ {
		a.tick()
		time.Sleep(100 * time.Millisecond)
	}
}

func decisionOf(a *Agent, id string) *decision {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.decisions[id]
}

const strictPolicy = `requireImageSignature: true
trustedPublishers: [cluster-root]
allowRuntimes: [process]
acceptTiers: [trusted]
`

// Invariant 1: a compromised control plane (valid member key) cannot make a
// sovereign host run what its root, its capability chain or its own policy
// does not allow.
func TestCompromisedControlPlaneCannotCommandHost(t *testing.T) {
	f := newFakeCP(t)
	a := newHost(t, f, strictPolicy)
	me := a.st.NodeID
	scripts := t.TempDir()
	os.WriteFile(filepath.Join(scripts, "sleeper"), []byte("#!/bin/sh\nexec sleep \"$1\"\n"), 0o755)
	os.WriteFile(filepath.Join(scripts, "shell"), []byte("#!/bin/sh\necho pwned > /tmp/dh-pwned\nsleep 60\n"), 0o755)
	sleepImg, sleepLoc := f.artifact(filepath.Join(scripts, "sleeper"), "sleep")
	shImg, shLoc := f.artifact(filepath.Join(scripts, "shell"), "sh")
	sleepDigest, shDigest := sleepLoc.Digest, shLoc.Digest
	attested := envelope.MustSign(f.root, envelope.KindArtifact, api.ArtifactAttestation{Digest: sleepDigest, Name: "sleep"})

	// Legitimate: root-attested artifact, capability chained to the root.
	good := f.assignment(assignOpts{id: "ok/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 3})
	f.serve(me, bundleOpts{index: 10, assign: []*envelope.Envelope{good}, artifacts: []api.ArtifactLocation{sleepLoc, shLoc}, attest: []*envelope.Envelope{attested}})
	settle(a, 25)
	d := decisionOf(a, "ok/r0")
	if d == nil || !d.Allowed || a.st.Admitted["ok/r0"] == nil || a.st.Admitted["ok/r0"].Inst.PID == 0 {
		for _, e := range a.journal.Entries(0, 0) {
			t.Logf("ledger %d %s %s %s", e.Seq, e.Action, e.Resource, e.Detail)
		}
		t.Fatalf("legitimate assignment not running: %+v", d)
	}
	pid := a.st.Admitted["ok/r0"].Inst.PID

	attacks := []struct {
		name string
		env  *envelope.Envelope
		code string
	}{
		{"shell artifact not attested by the root (forged shell)", f.assignment(assignOpts{id: "shell/r0", node: me, image: shImg, digest: shDigest, gen: 1}), "POLICY_ARTIFACT_SIGNATURE"},
		{"assignment for another host", f.assignment(assignOpts{id: "other/r0", node: "dh1aaaaaaaaaaaaaaaaaaaaaaaaaa", image: sleepImg, digest: sleepDigest, gen: 1}), "WRONG_HOST"},
		{"capability not anchored in the root", f.assignment(assignOpts{id: "self/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 1, selfAnchored: true}), "CAPABILITY"},
		{"image swapped under a capability for another digest", f.assignment(assignOpts{id: "swap/r0", node: me, image: shImg, digest: shDigest, capDigest: sleepDigest, gen: 1}), "CAPABILITY"},
		{"resources above the capability limit", f.assignment(assignOpts{id: "big/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 1, cpu: 900, capCPU: 100}), "CAPABILITY"},
		{"runtime not allowed by host policy", f.assignment(assignOpts{id: "dock/r0", node: me, image: "evil@sha256:" + strings.Repeat("a", 64), digest: "sha256:" + strings.Repeat("a", 64), runtime: "docker", gen: 1}), "POLICY_RUNTIME"},
		{"trust tier not accepted", f.assignment(assignOpts{id: "tier/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 1, tiers: []string{"community"}}), "POLICY_TIER"},
		{"assignment signed by a non-member key", f.assignment(assignOpts{id: "imp/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 1, signer: f.root}), "ASSIGNMENT_SIGNATURE"},
	}
	envs := []*envelope.Envelope{good}
	for _, at := range attacks {
		envs = append(envs, at.env)
	}
	f.serve(me, bundleOpts{index: 11, assign: envs, artifacts: []api.ArtifactLocation{sleepLoc, shLoc}, attest: []*envelope.Envelope{attested}})
	settle(a, 5)
	for _, at := range attacks {
		var as api.Assignment
		at.env.Decode(&as)
		d := decisionOf(a, as.ID)
		if d == nil || d.Allowed || d.Code != at.code {
			t.Errorf("%s: decision %+v, want %s", at.name, d, at.code)
		}
		if ad := a.st.Admitted[as.ID]; ad != nil && ad.Inst.PID != 0 {
			t.Errorf("%s: PROCESS STARTED (pid %d)", at.name, ad.Inst.PID)
		}
	}

	if _, err := os.Stat("/tmp/dh-pwned"); err == nil {
		os.Remove("/tmp/dh-pwned")
		t.Fatal("the forged shell artifact executed")
	}

	t.Run("stale generation is rejected and the running instance is untouched", func(t *testing.T) {
		stale := f.assignment(assignOpts{id: "ok/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 2})
		f.serve(me, bundleOpts{index: 12, assign: []*envelope.Envelope{stale}, artifacts: []api.ArtifactLocation{sleepLoc}, attest: []*envelope.Envelope{attested}})
		settle(a, 3)
		if d := decisionOf(a, "ok/r0"); d == nil || d.Code != "STALE_GENERATION" {
			t.Fatalf("stale generation: %+v", d)
		}
		if a.st.Admitted["ok/r0"].Inst.PID != pid || a.proc.Status(a.st.Admitted["ok/r0"].Inst).State != "running" {
			t.Fatal("running instance disturbed by a stale assignment")
		}
	})

	t.Run("bundle rollback is rejected", func(t *testing.T) {
		f.serve(me, bundleOpts{index: 5, assign: []*envelope.Envelope{good}, artifacts: []api.ArtifactLocation{sleepLoc}, attest: []*envelope.Envelope{attested}})
		a.syncBundle()
		a.mu.RLock()
		trusted, why := a.trusted, a.trustWhy
		a.mu.RUnlock()
		if trusted || !strings.Contains(why, "rollback") {
			t.Fatalf("rollback accepted: trusted=%v %s", trusted, why)
		}
	})

	t.Run("impostor control plane key is not trusted; admitted work is held", func(t *testing.T) {
		imp, _ := identity.Generate()
		f.serve(me, bundleOpts{index: 20, signer: imp, assign: []*envelope.Envelope{good}, artifacts: []api.ArtifactLocation{sleepLoc}, attest: []*envelope.Envelope{attested}})
		settle(a, 2)
		a.mu.RLock()
		trusted, why := a.trusted, a.trustWhy
		a.mu.RUnlock()
		if trusted || !strings.Contains(why, "not in the root-signed roster") {
			t.Fatalf("impostor bundle trusted: %v %s", trusted, why)
		}
		if a.proc.Status(a.st.Admitted["ok/r0"].Inst).State != "running" {
			t.Fatal("admitted work stopped by an untrusted bundle")
		}
	})

	t.Run("roster not signed by the pinned root cannot enlarge the trust set", func(t *testing.T) {
		evilRoot, _ := identity.Generate()
		evil := envelope.MustSign(evilRoot, envelope.KindRoster, api.Roster{Cluster: "t", Version: 9, Root: evilRoot.PubString(), Members: []api.Member{{ID: f.member.ID, Pub: f.member.PubString()}}})
		f.serve(me, bundleOpts{index: 21, roster: evil, assign: []*envelope.Envelope{good}})
		a.syncBundle()
		a.mu.RLock()
		trusted, why := a.trusted, a.trustWhy
		a.mu.RUnlock()
		if trusted || !strings.Contains(why, "roster is not signed by the pinned root") {
			t.Fatalf("forged roster trusted: %v %s", trusted, why)
		}
	})

	t.Run("revocation holds admitted work and refuses new work", func(t *testing.T) {
		newWork := f.assignment(assignOpts{id: "new/r0", node: me, image: sleepImg, digest: sleepDigest, gen: 1})
		f.serve(me, bundleOpts{index: 30, assign: []*envelope.Envelope{good, newWork}, revoked: []string{me}, artifacts: []api.ArtifactLocation{sleepLoc}, attest: []*envelope.Envelope{attested}})
		settle(a, 3)
		if d := decisionOf(a, "ok/r0"); d == nil || !d.Hold || d.Code != "HOST_REVOKED" {
			t.Fatalf("admitted work not held under revocation: %+v", d)
		}
		if d := decisionOf(a, "new/r0"); d == nil || d.Allowed || d.Code != "HOST_REVOKED" {
			t.Fatalf("new work admitted while revoked: %+v", d)
		}
		if a.proc.Status(a.st.Admitted["ok/r0"].Inst).State != "running" {
			t.Fatal("revocation killed admitted work")
		}
	})

	// Every refusal is in the host's own hash-chained ledger.
	refusals := 0
	for _, e := range a.journal.Entries(0, 0) {
		if e.Action == "admission-refuse" {
			refusals++
		}
	}
	if refusals < len(attacks) {
		t.Fatalf("host ledger recorded %d refusals, want >= %d", refusals, len(attacks))
	}
}
