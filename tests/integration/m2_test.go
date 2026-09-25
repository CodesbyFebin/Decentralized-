package integration

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/node"
)

const kvManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: kv}
spec:
  replicas: 1
  image: %s
  resources: {cpu: 100m, mem: 64Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  volumes:
    - name: data
      size: 512Mi
      mount: /data
      durability: {replicas: 3}
      snapshot: {every: 2s, retain: 4}
  ingress: [{host: kv.test, port: http, tls: none}]
  health: {http: /healthz, interval: 1s}
`

func edgeDo(c *devcluster.Cluster, method, path string, body []byte) (int, []byte, string, error) {
	req, _ := http.NewRequest(method, "http://"+c.Edge+path, bytes.NewReader(body))
	req.Host = "kv.test"
	cl := &http.Client{Timeout: 30 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return 0, nil, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, resp.Header.Get("X-DH-Upstream"), nil
}

func kvVolume(v *control.View) *control.VolumeView {
	for i := range v.Volumes {
		if v.Volumes[i].ID == "kv/data/r0" {
			return &v.Volumes[i]
		}
	}
	return nil
}

func nodeName(v *control.View, id string) string {
	for _, n := range v.Nodes {
		if n.ID == id {
			return n.Name
		}
	}
	return id
}

// M2 exit: a host dies, and the remaining valid replicas hold recoverable
// data; the replica is rescheduled and restored from the committed snapshot.
func TestM2StorageSurvivesHostLoss(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 4, Edges: 1, Chaos: true, LostAfter: "6s",
		HostArgs: func(int, string) []string { return []string{"--anti-entropy", "3s"} }})
	deploy(t, c, kvManifest)
	waitRunning(t, c, "kv", 1, 60*time.Second)

	payload := make([]byte, 3<<20) // several FastCDC chunks
	rand.Read(payload)
	if err := c.WaitFor(20*time.Second, "edge routes kv", func(*control.View) bool {
		code, _, _, err := edgeDo(c, "GET", "/healthz", nil)
		return err == nil && code == 200
	}); err != nil {
		t.Fatal(err)
	}
	code, _, _, err := edgeDo(c, "PUT", "/kv/alpha", payload)
	if err != nil || code != 201 {
		t.Fatalf("write: %d %v", code, err)
	}

	var committed string
	if err := c.WaitFor(60*time.Second, "a committed snapshot holding the write, verified on 3 replicas", func(v *control.View) bool {
		vol := kvVolume(v)
		if vol == nil || vol.CommittedRef == nil || vol.CommittedRef.Bytes < int64(len(payload)) {
			return false
		}
		committed = vol.Committed
		return vol.State == "HEALTHY" && vol.Verified >= 3
	}); err != nil {
		t.Fatal(err)
	}
	v, _ := c.View()
	vol := kvVolume(v)
	primary := nodeName(v, vol.Primary)
	t.Logf("committed %s on %v (primary %s)", committed[:15], vol.MemberNames, primary)

	killed, err := c.KillMachine(primary)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("killed %s and %d workload(s)", primary, killed)

	var newPrimary string
	if err := c.WaitFor(90*time.Second, "replica rescheduled, restored and running elsewhere", func(v *control.View) bool {
		a := app(v, "kv")
		if a == nil || a.Observed < 1 {
			return false
		}
		for _, r := range a.Rows {
			if r.Desired == "RUNNING" && r.Observed == "RUNNING" && r.NodeName != primary {
				newPrimary = r.NodeName
				for _, vo := range r.Volumes {
					if vo.Restored == committed {
						return true
					}
				}
			}
		}
		return false
	}); err != nil {
		t.Fatal(err)
	}
	var got []byte
	if err := c.WaitFor(30*time.Second, "restored data readable through the edge", func(*control.View) bool {
		code, b, up, err := edgeDo(c, "GET", "/kv/alpha", nil)
		got = b
		return err == nil && code == 200 && strings.Contains(up, "kv/r0") && len(b) == len(payload)
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("restored bytes differ from what was written")
	}
	t.Logf("replica restored on %s from %s; %d bytes identical", newPrimary, committed[:15], len(got))

	v, _ = c.View()
	actions := map[string]bool{}
	for _, e := range v.Audit.Entries {
		actions[e.Action] = true
	}
	for _, want := range []string{"snapshot-commit", "node-lost", "volume-placement"} {
		if !actions[want] {
			t.Errorf("audit lacks %s", want)
		}
	}

	t.Run("interrupted write never corrupts committed data", func(t *testing.T) {
		v, _ := c.View()
		before := kvVolume(v).Committed
		big := make([]byte, 24<<20)
		rand.Read(big)
		code, _, _, err := edgeDo(c, "PUT", "/kv/beta", big)
		if err != nil || code != 201 {
			t.Fatalf("write: %d %v", code, err)
		}
		// Kill the primary the moment it reports a snapshot that carries the
		// new write (normally still pending: being replicated).
		var pendingSeen bool
		if err := c.WaitFor(30*time.Second, "a snapshot carrying the new write", func(v *control.View) bool {
			for _, sn := range kvVolume(v).Snapshots {
				if sn.Bytes >= int64(len(big)) {
					pendingSeen = sn.State == "pending"
					return true
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
		_ = before
		v, _ = c.View()
		vol := kvVolume(v)
		atKill := vol.Committed
		prim := nodeName(v, vol.Primary)
		if _, err := c.KillMachine(prim); err != nil {
			t.Fatal(err)
		}
		t.Logf("killed %s mid-write (pending snapshot seen: %v; committed at kill %s)", prim, pendingSeen, atKill[:15])
		if err := c.WaitFor(120*time.Second, "replica recovered on another host", func(v *control.View) bool {
			a := app(v, "kv")
			for _, r := range a.Rows {
				if r.Desired == "RUNNING" && r.Observed == "RUNNING" && r.NodeName != prim && r.NodeName != primary {
					return true
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
		// Committed data (alpha) must be intact; beta is either fully there
		// (its snapshot committed before the kill) or absent — never partial.
		var alpha, beta []byte
		var betaCode int
		var last string
		if err := c.WaitFor(30*time.Second, "data readable after recovery", func(*control.View) bool {
			c1, a1, _, e1 := edgeDo(c, "GET", "/kv/alpha", nil)
			c2, b2, _, e2 := edgeDo(c, "GET", "/kv/beta", nil)
			alpha, beta, betaCode = a1, b2, c2
			last = fmt.Sprintf("alpha %d %v %q; beta %d %v", c1, e1, clip(a1), c2, e2)
			return e1 == nil && e2 == nil && c1 == 200 && (c2 == 200 || c2 == 404)
		}); err != nil {
			// Diagnostics: what the edge returned, how it routes, and what
			// the recovering host did.
			t.Logf("last reads: %s", last)
			if v, verr := c.View(); verr == nil {
				for _, e := range v.Edge.Edges {
					if e.Obs != nil {
						for _, r := range e.Obs.Routes {
							for _, ep := range r.Endpoints {
								t.Logf("edge %s route %s: %s on %s: %s (%s)", e.Name, r.Host, ep.Assignment, nodeName(v, ep.Node), ep.State, ep.Reason)
							}
						}
					}
				}
				for _, r := range app(v, "kv").Rows {
					t.Logf("kv %s on %s: %s [%s] %s", r.Assignment, r.NodeName, r.Status, r.Code, r.Reason)
					if r.NodeName != prim && r.Observed == "RUNNING" {
						for _, l := range c.HostLedgerTail(r.NodeName, 15) {
							t.Logf("%s journal %s", r.NodeName, l)
						}
					}
				}
			}
			t.Fatal(err)
		}
		if !bytes.Equal(alpha, payload) {
			t.Fatal("committed data corrupted by an interrupted write")
		}
		switch {
		case betaCode == 404:
			t.Logf("interrupted write was not committed; committed state preserved (outcome: rolled back to %s)", atKill[:15])
		case bytes.Equal(beta, big):
			t.Logf("write committed before the kill; replicas hold all %d bytes (outcome: committed)", len(big))
		default:
			t.Fatalf("PARTIAL DATA: beta has %d bytes, want 0 or %d", len(beta), len(big))
		}
	})

	t.Run("anti-entropy repairs a corrupted replica chunk with evidence", func(t *testing.T) {
		v, _ := c.View()
		vol := kvVolume(v)
		var member string
		for _, m := range vol.Members {
			if m != vol.Primary {
				member = nodeName(v, m)
			}
		}
		if member == "" || !c.Alive(member) {
			t.Skip("no live non-primary member")
		}
		before := len(v.Repairs)
		out, err := c.Chaos(member, node.ChaosRequest{Fault: "corrupt-chunk", Volume: "kv/data/r0"})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("corrupted %v on %s", out["object"], member)
		if err := c.WaitFor(60*time.Second, "repair evidence from the corrupted host", func(v *control.View) bool {
			for _, r := range v.Repairs[min(before, len(v.Repairs)):] {
				if nodeName(v, r.R.Node) == member {
					for _, it := range r.R.Items {
						// "missing" when a peer read quarantined it first.
						if it.Object == fmt.Sprint(out["object"]) && it.Verified && (it.Previous == "corrupt" || it.Previous == "missing") {
							return true
						}
					}
				}
			}
			return false
		}); err != nil {
			// The corrupted object may belong to a snapshot already
			// superseded and collected; then quarantine is the evidence.
			st, _ := c.HostStatus(member)
			t.Fatalf("%v (host storage %+v)", err, st.Storage)
		}
	})
}

func clip(b []byte) string {
	if len(b) > 120 {
		return string(b[:120]) + "…"
	}
	return string(b)
}
