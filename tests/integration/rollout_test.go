package integration

import (
	"fmt"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
)

const rolloutManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: roll}
spec:
  replicas: 1
  image: %%s
  env: {GENERATION_MARK: "%s"}
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted]}
  ports: [{name: http}]
  ingress: [{host: roll.test, port: http, tls: none}]
  health: {http: /healthz, interval: 1s}
`

// Regression (PV-1.1, Linux chaos interrupted-deployment): a rolling update
// used to stop the old instance before starting the new one, and the edge
// learned "draining" only afterwards, so requests failed. With one replica
// there is nothing to fall back on: only make-before-break serves every
// request.
func TestRollingUpdateOfSingleReplicaDropsNothing(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 1, Edges: 1})
	image := deploy(t, c, fmt.Sprintf(rolloutManifest, "one"))
	waitRunning(t, c, "roll", 1, 60*time.Second)
	if err := c.WaitFor(20*time.Second, "edge routes roll/r0", func(v *control.View) bool {
		return routeStates(edgeObs(v, "edge-1"), "roll.test")["roll/r0"] == "routing"
	}); err != nil {
		t.Fatal(err)
	}
	tr := startTraffic(c.Edge, "roll.test")
	time.Sleep(time.Second)
	r, err := c.Op.Apply([]byte(fmt.Sprintf(fmt.Sprintf(rolloutManifest, "two"), image)))
	if err != nil || !r.OK {
		t.Fatalf("apply generation 2: %v %s", err, r.Message)
	}
	if err := c.WaitFor(60*time.Second, "generation 2 observed running", func(v *control.View) bool {
		a := app(v, "roll")
		return a != nil && a.Generation == 2 && a.Observed == 1
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	ok, fail := tr.finish()
	t.Logf("%d requests during the rolling update, %d failed", ok, fail)
	if fail > 0 {
		t.Fatalf("rolling update of the only replica failed %d request(s)", fail)
	}
}
