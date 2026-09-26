package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
)

// RUNTIME-P0-A01, end to end: an application that asks for RESTRICTED
// isolation is deployed through the real control plane and run by the real
// host agent inside the sandbox; UNTRUSTED is refused; a host whose policy
// forbids the profile refuses the workload (fail closed).
const sandboxManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: %s}
spec:
  replicas: 1
  image: %s
  isolation: %s
  resources: {cpu: 200m, mem: 128Mi}
  placement: {tiers: [trusted]}
  ports: [{name: http}]
  health: {http: /healthz, interval: 1s}
`

func TestRuntimeP0Isolation(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 1})
	image, _, err := c.Op.PushArtifact(binDir+"/dh-beacon", "dh-beacon", true)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("UNTRUSTED is refused by the control plane at apply", func(t *testing.T) {
		r, err := c.Op.Apply([]byte(fmt.Sprintf(sandboxManifest, "untrusted", image, "UNTRUSTED")))
		if err == nil && r.OK {
			t.Fatal("UNTRUSTED manifest was accepted")
		}
		msg := r.Message
		if err != nil {
			msg = err.Error()
		}
		if !strings.Contains(strings.ToUpper(msg), "UNTRUSTED") {
			t.Fatalf("refusal did not mention UNTRUSTED: %q", msg)
		}
	})

	t.Run("a RESTRICTED workload runs in the sandbox and passes its health check", func(t *testing.T) {
		r, err := c.Op.Apply([]byte(fmt.Sprintf(sandboxManifest, "walled", image, "RESTRICTED")))
		if err != nil || !r.OK {
			t.Fatalf("apply RESTRICTED: %+v %v", r, err)
		}
		v := waitRunning(t, c, "walled", 1, 90*time.Second)
		app := app(v, "walled")
		row := app.Rows[0]
		if row.Observed != "RUNNING" || row.Code != "ADMITTED" {
			t.Fatalf("replica not admitted+running: %+v", row)
		}
		// Admission ran the sandbox and isolation checks and they passed.
		var sawSandbox, sawIsolation bool
		for _, ch := range row.Checks {
			switch ch.Name {
			case "sandbox":
				sawSandbox = ch.OK
			case "isolation":
				sawIsolation = ch.OK
			}
		}
		if !sawSandbox || !sawIsolation {
			t.Fatalf("sandbox/isolation admission checks not both passed: %+v", row.Checks)
		}
		// The host reports the workload under the sandbox runtime.
		st, err := c.HostStatus("host-a")
		if err != nil {
			t.Fatal(err)
		}
		ad := st.Admitted["walled/r0"]
		if ad == nil || ad.Runtime != "sandbox" {
			t.Fatalf("workload not run under the sandbox runtime: %+v", ad)
		}
		if ad.Inst.Port == 0 {
			t.Fatal("no relay port for the RESTRICTED workload")
		}
		// Health passing is proof the relay reached the sandboxed process.
		if !healthy(t, c, "walled") {
			t.Fatal("RESTRICTED workload never became healthy through the relay")
		}
	})
}

// A host whose policy forbids RESTRICTED refuses the workload rather than
// running it unsandboxed.
func TestRuntimeP0IsolationPolicyRefusal(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 1, Policy: func(string) string {
		return "sovereign: true\nallowRuntimes: [process, docker]\nallowIsolation: []\nacceptTiers: [trusted]\n"
	}})
	image, _, err := c.Op.PushArtifact(binDir+"/dh-beacon", "dh-beacon", true)
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Op.Apply([]byte(fmt.Sprintf(sandboxManifest, "walled", image, "RESTRICTED")))
	if err != nil || !r.OK {
		t.Fatalf("apply: %+v %v", r, err)
	}
	err = c.WaitFor(30*time.Second, "host refuses the RESTRICTED workload", func(v *control.View) bool {
		a := app(v, "walled")
		if a == nil || len(a.Rows) == 0 {
			return false
		}
		return a.Rows[0].Code == "POLICY_ISOLATION"
	})
	if err != nil {
		v, _ := c.View()
		if a := app(v, "walled"); a != nil && len(a.Rows) > 0 {
			t.Fatalf("expected POLICY_ISOLATION, got %s: %s", a.Rows[0].Code, a.Rows[0].Reason)
		}
		t.Fatal(err)
	}
}

func healthy(t *testing.T, c *devcluster.Cluster, name string) bool {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		v, _ := c.View()
		if a := app(v, name); a != nil && len(a.Rows) > 0 {
			h := a.Rows[0].Health
			if h != nil && h.OK {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
