// Package integration runs real multi-process clusters (dh-control Raft
// members, dh-noded hosts over userspace WireGuard, edge hosts) and checks
// the milestone invariants against what the processes actually do.
package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
)

var binDir string
var basePort atomic.Int64

func TestMain(m *testing.M) {
	// Integration tests start real processes; they run by default and are
	// skipped with -short.
	dir, err := os.MkdirTemp("", "dh-bin-")
	if err != nil {
		panic(err)
	}
	root, _ := filepath.Abs("../..")
	cmd := exec.Command("go", "build", "-o", dir+"/", "./cmd/dh", "./cmd/dh-control", "./cmd/dh-noded", "./cmd/dh-beacon")
	cmd.Dir = root
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("build failed:", err)
		os.Exit(1)
	}
	binDir = dir
	basePort.Store(21000)
	// Separate port ranges let several test processes run side by side.
	if b, err := strconv.Atoi(os.Getenv("DH_TEST_BASE")); err == nil && b > 0 {
		basePort.Store(int64(b))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func up(t *testing.T, o devcluster.Options) *devcluster.Cluster {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test (starts real processes); skipped with -short")
	}
	o.Dir = t.TempDir()
	o.Bin = binDir
	o.Base = int(basePort.Add(1000))
	o.Quiet = true
	if o.CPs == 0 {
		o.CPs = 1
	}
	c, err := devcluster.Up(o)
	if c != nil {
		t.Cleanup(func() {
			if t.Failed() {
				if keep := os.Getenv("DH_KEEP"); keep != "" {
					// Goroutine dumps from every process, then keep the logs.
					for _, p := range c.Procs {
						_ = syscall.Kill(p.PID, syscall.SIGQUIT)
					}
					time.Sleep(time.Second)
					_ = exec.Command("cp", "-R", filepath.Join(o.Dir, "logs"), filepath.Join(keep, t.Name())).Run()
				}
				dumpLogs(t, o.Dir)
			}
			c.Down()
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func dumpLogs(t *testing.T, dir string) {
	files, _ := filepath.Glob(filepath.Join(dir, "logs", "*.log"))
	for _, f := range files {
		b, _ := os.ReadFile(f)
		lines := strings.Split(string(b), "\n")
		if len(lines) > 25 {
			lines = lines[len(lines)-25:]
		}
		t.Logf("---- %s ----\n%s", filepath.Base(f), strings.Join(lines, "\n"))
	}
}

func deploy(t *testing.T, c *devcluster.Cluster, manifest string) string {
	t.Helper()
	image, _, err := c.Op.PushArtifact(filepath.Join(binDir, "dh-beacon"), "dh-beacon", true)
	if err != nil {
		t.Fatal(err)
	}
	r, err := c.Op.Apply([]byte(fmt.Sprintf(manifest, image)))
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatal(r.Message)
	}
	return image
}

func app(v *control.View, name string) *control.AppView {
	for i := range v.Apps {
		if v.Apps[i].Name == name {
			return &v.Apps[i]
		}
	}
	return nil
}

func nodeByName(v *control.View, name string) *control.NodeView {
	for i := range v.Nodes {
		if v.Nodes[i].Name == name {
			return &v.Nodes[i]
		}
	}
	return nil
}

// observedRunning counts replicas running at the app's generation with fresh evidence.
func observedRunning(v *control.View, name string) int {
	a := app(v, name)
	if a == nil {
		return 0
	}
	return a.Observed
}

func waitRunning(t *testing.T, c *devcluster.Cluster, name string, n int, timeout time.Duration) *control.View {
	t.Helper()
	var last *control.View
	err := c.WaitFor(timeout, fmt.Sprintf("%d replica(s) of %s observed running", n, name), func(v *control.View) bool {
		last = v
		return observedRunning(v, name) >= n
	})
	if err != nil {
		if last != nil {
			if a := app(last, name); a != nil {
				for _, r := range a.Rows {
					t.Logf("r%d on %s: %s [%s] %s", r.Replica, r.NodeName, r.Status, r.Code, r.Reason)
				}
			}
		}
		t.Fatal(err)
	}
	return last
}

const beaconManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: beacon}
spec:
  replicas: %d
  image: %%s
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  health: {http: /healthz, interval: 1s}
`

func beacon(replicas int) string { return fmt.Sprintf(beaconManifest, replicas) }
