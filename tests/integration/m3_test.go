package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"decentralized.host/pkg/cli"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/envelope"
)

func hostObsPub(t *testing.T, c *devcluster.Cluster, host string) string {
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
			return n.ObsEnv.Pub
		}
	}
	return ""
}

// M3: WireGuard mesh with signed bindings, SWIM gossip, capability-bound
// assignments, key rotation with grace, emergency key revocation, revoked
// hosts cut from the mesh, and root rotation followed by hosts.
func TestM3TrustAndMesh(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 3, Chaos: true})
	deploy(t, c, beacon(3))
	v := waitRunning(t, c, "beacon", 3, 60*time.Second)
	pids := map[string]int64{}
	for _, r := range app(v, "beacon").Rows {
		pids[r.Assignment] = r.PID
		found := false
		for _, ch := range r.Checks {
			if ch.Name == "capability" && ch.OK && strings.Contains(ch.Detail, "anchored in the pinned root") {
				found = true
			}
		}
		if !found {
			t.Fatalf("replica %d admitted without a root-anchored capability check: %+v", r.Replica, r.Checks)
		}
	}

	if err := c.WaitFor(30*time.Second, "full WireGuard mesh with handshakes, verified bindings, gossip and measured RTT", func(v *control.View) bool {
		for _, n := range v.Nodes {
			if n.Mesh == nil || n.Mesh.Device != "wireguard-go/netstack" || n.Mesh.Members != 3 || len(n.Mesh.Peers) != 3 {
				return false
			}
			for _, p := range n.Mesh.Peers {
				if !p.BindingOK || p.LastHandshake == 0 || p.RTTAt == 0 {
					return false
				}
			}
		}
		return true
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("key rotation keeps the identity and the workloads", func(t *testing.T) {
		oldPub := hostObsPub(t, c, "host-a")
		out, err := exec.Command(filepath.Join(binDir, "dh-noded"), "rotate-key", "--data", c.Proc("host-a").Data, "--grace", "1h").CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		var newPub string
		if err := c.WaitFor(20*time.Second, "host-a observations signed by the rotated key", func(v *control.View) bool {
			newPub = hostObsPub(t, c, "host-a")
			n := nodeByName(v, "host-a")
			return newPub != oldPub && len(n.Keys) == 2 && n.LastObs.Freshness == "FRESH"
		}); err != nil {
			t.Fatal(err)
		}
		v, _ := c.View()
		n := nodeByName(v, "host-a")
		if n.Keys[0].Until == 0 || n.Keys[1].Pub != newPub {
			t.Fatalf("key records: %+v", n.Keys)
		}
		for _, r := range app(v, "beacon").Rows {
			if r.PID != pids[r.Assignment] {
				t.Fatalf("workload restarted by key rotation: %+v", r)
			}
		}
	})

	t.Run("revoked host is cut from the mesh but keeps admitted work", func(t *testing.T) {
		id, _ := c.Op.ResolveNode("host-c")
		if err := c.Op.Do("POST", "/api/v1/nodes/"+id+"/revoke", map[string]string{"reason": "compromised"}, nil); err != nil {
			t.Fatal(err)
		}
		// Regression (reference run, suite 2): the member that commits a
		// revocation must cut the host from its own mesh at once, not on
		// its next periodic mesh pass.
		time.Sleep(300 * time.Millisecond)
		var early map[string]any
		_ = c.Op.Do("POST", "/api/v1/mesh/ping", map[string]string{"node": id}, &early)
		if s, ok := early["samplesUs"].([]any); ok && len(s) > 0 {
			t.Fatalf("300ms after the committed revocation the control plane still reaches the host: %v", early)
		}
		if err := c.WaitFor(20*time.Second, "other hosts drop host-c as a WireGuard peer", func(v *control.View) bool {
			for _, name := range []string{"host-a", "host-b"} {
				for _, p := range nodeByName(v, name).Mesh.Peers {
					if p.Node == id {
						return false
					}
				}
			}
			return true
		}); err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		_ = c.Op.Do("POST", "/api/v1/mesh/ping", map[string]string{"node": id}, &out)
		if s, ok := out["samplesUs"].([]any); ok && len(s) > 0 {
			t.Fatalf("control plane still reaches the revoked host over the mesh: %v", out)
		}
		st, err := c.HostStatus("host-c")
		if err != nil {
			t.Fatal(err)
		}
		for id, ad := range st.Admitted {
			if ad.LastState != "running" || ad.Inst.PID != pids[id] {
				t.Fatalf("revocation killed admitted work: %+v", ad)
			}
		}
	})

	t.Run("emergency key revocation rejects the host's signatures", func(t *testing.T) {
		id, _ := c.Op.ResolveNode("host-b")
		pub := hostObsPub(t, c, "host-b")
		if err := c.Op.Do("POST", "/api/v1/nodes/"+id+"/revoke-key", map[string]string{"pub": pub, "reason": "key leaked"}, nil); err != nil {
			t.Fatal(err)
		}
		if err := c.WaitFor(20*time.Second, "signatures by the revoked key rejected", func(v *control.View) bool {
			for _, r := range v.Rejections {
				if r.Node == id && strings.Contains(r.Reason, "wrong host key") {
					return nodeByName(v, "host-b").LastObs.Freshness == "STALE"
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("root rotation is followed by hosts that pinned the old root", func(t *testing.T) {
		out, err := exec.Command(filepath.Join(binDir, "dh"), "--home", c.Home, "cp", "rotate-root").CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		op, err := cli.Load(c.Home, c.Opt.Cluster)
		if err != nil {
			t.Fatal(err)
		}
		c.Op = op
		deadline := time.Now().Add(30 * time.Second)
		for {
			st, err := c.HostStatus("host-a")
			if err == nil && st.Root == op.Root.PubString() && st.Trusted && st.Fresh {
				held := false
				for _, d := range st.Decisions {
					if d.Hold {
						held = true
					}
				}
				if !held {
					break
				}
			}
			if time.Now().After(deadline) {
				t.Fatalf("host-a did not follow the root rotation: %+v %v", st, err)
			}
			time.Sleep(300 * time.Millisecond)
		}
		v, _ := c.View()
		if a := app(v, "beacon"); a.Observed < 1 {
			t.Fatal("workloads lost across root rotation")
		}
	})
}
