package integration

import (
	"strings"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
)

// Regression (reference run after PV1-S1-A02: chaos host-crash and
// network-partition, intermittent): a host fetching an artifact tried a dead
// holder first for every chunk and waited the full timeout each time, so a
// rescheduled replica never started. Failed holders are now skipped.
func TestArtifactFetchSkipsDeadHolder(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 1, Hosts: 4, LostAfter: "5s"})
	deploy(t, c, beacon(1))
	waitRunning(t, c, "beacon", 1, 60*time.Second)
	// Holders are recorded asynchronously, once hosts report the artifact's
	// chunks (REF-MAC-A02 read the list before that and saw none).
	var v *control.View
	if err := c.WaitFor(30*time.Second, "artifact holders recorded", func(cur *control.View) bool {
		v = cur
		return len(cur.Artifacts) > 0 && len(cur.Artifacts[0].HolderNames) > 0
	}); err != nil {
		t.Fatal(err)
	}
	dead := v.Artifacts[0].HolderNames[0]
	t.Logf("artifact holders %v; killing %s", v.Artifacts[0].HolderNames, dead)
	if err := c.Kill(dead, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	if err := c.WaitFor(30*time.Second, dead+" marked lost", func(v *control.View) bool {
		n := nodeByName(v, dead)
		return n != nil && strings.Contains(n.Health, "lost")
	}); err != nil {
		t.Fatal(err)
	}
	// Three live hosts, hard anti-affinity: at least one must fetch the
	// artifact while the first-listed holder is dead.
	start := time.Now()
	if err := c.Op.Do("POST", "/api/v1/apps/beacon/scale", map[string]int64{"replicas": 3}, nil); err != nil {
		t.Fatal(err)
	}
	waitRunning(t, c, "beacon", 3, 45*time.Second)
	t.Logf("3 replicas running %s after scaling with holder %s dead", time.Since(start).Round(time.Millisecond), dead)
}
