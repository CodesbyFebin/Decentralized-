package integration

import (
	"io"
	"net/http"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
)

const selfEdgeManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: solo}
spec:
  replicas: 1
  image: %s
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted]}
  ports: [{name: http}]
  ingress: [{host: solo.test, port: http, tls: none}]
  health: {http: /healthz, interval: 1s}
`

// Regression (PV-1.1, Linux): an edge host that also runs a replica must
// route to it. The edge dials its own mesh address, and the forwarder once
// refused that client because the host is not its own peer. With random
// placement this only showed up when a replica landed on the edge.
func TestEdgeRoutesToReplicaOnItself(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 0, Edges: 1})
	deploy(t, c, selfEdgeManifest)
	waitRunning(t, c, "solo", 1, 60*time.Second)
	if err := c.WaitFor(20*time.Second, "edge routes the replica on itself", func(v *control.View) bool {
		return routeStates(edgeObs(v, "edge-1"), "solo.test")["solo/r0"] == "routing"
	}); err != nil {
		v, _ := c.View()
		t.Fatalf("%v; edge states %v", err, routeStates(edgeObs(v, "edge-1"), "solo.test"))
	}
	req, _ := http.NewRequest("GET", "http://"+c.Edge+"/", nil)
	req.Host = "solo.test"
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("request through the edge to its own replica: HTTP %d", resp.StatusCode)
	}
}
