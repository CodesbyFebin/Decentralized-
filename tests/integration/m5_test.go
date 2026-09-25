package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/envelope"
)

const haManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: web}
spec:
  replicas: 3
  image: %s
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  ingress: [{host: ha.test, port: http, tls: none}]
  health: {http: /healthz, interval: 1s}
`

// traffic sends requests through the edge until stopped and counts results.
type traffic struct {
	ok, fail atomic.Int64
	stop     chan struct{}
	wg       sync.WaitGroup
}

func startTraffic(edge, host string) *traffic {
	tr := &traffic{stop: make(chan struct{})}
	tr.wg.Add(1)
	go func() {
		defer tr.wg.Done()
		cl := &http.Client{Timeout: 3 * time.Second}
		for {
			select {
			case <-tr.stop:
				return
			default:
			}
			req, _ := http.NewRequest("GET", "http://"+edge+"/", nil)
			req.Host = host
			resp, err := cl.Do(req)
			if err == nil && resp.StatusCode == 200 {
				tr.ok.Add(1)
			} else {
				tr.fail.Add(1)
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return tr
}

func (tr *traffic) finish() (int64, int64) {
	close(tr.stop)
	tr.wg.Wait()
	return tr.ok.Load(), tr.fail.Load()
}

// M5 exit: three members, leader loss without workload interruption, no
// lost committed writes, snapshot/backup and restore.
func TestM5HighlyAvailableControlPlane(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 3, Hosts: 3, Edges: 1, LostAfter: "20s"})
	deploy(t, c, haManifest)
	waitRunning(t, c, "web", 3, 60*time.Second)
	if err := c.WaitFor(20*time.Second, "edge routes all replicas", func(v *control.View) bool {
		n := 0
		for _, s := range routeStates(edgeObs(v, "edge-1"), "ha.test") {
			if s == "routing" {
				n++
			}
		}
		return n == 3
	}); err != nil {
		t.Fatal(err)
	}
	v, _ := c.View()
	if v.Cluster.HA != "RAFT 3 VOTERS (tolerates 1 failure(s))" {
		t.Fatalf("HA: %s", v.Cluster.HA)
	}

	t.Run("every member serves the leader's view, the console and conformance", func(t *testing.T) {
		get := func(url string, out any) http.Header {
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+c.Op.Bearer())
			resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("%s: HTTP %d", url, resp.StatusCode)
			}
			if out != nil {
				if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
					t.Fatal(err)
				}
			}
			return resp.Header
		}
		for _, p := range c.Procs {
			if p.Kind != "control" {
				continue
			}
			// Followers relay the leader's view so observed state is as fresh
			// as the leader's, not as fresh as the last commit.
			var mv control.View
			h := get("http://"+p.API+"/api/v1/view", &mv)
			if mv.ServedBy.State != "leader" || mv.Overview.Observed != 3 {
				t.Fatalf("%s served a %s view with %d observed (relayed by %q)", p.Name, mv.ServedBy.State, mv.Overview.Observed, h.Get("X-DH-Relayed-By"))
			}
			if mv.ServedBy.LastContact < 0 {
				t.Fatalf("lastContact %d: zero time leaked", mv.ServedBy.LastContact)
			}
			// The console: no third-party origins, always revalidated.
			resp, err := http.Get("http://" + p.API + "/")
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if csp := resp.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") || strings.Contains(csp, "https:") {
				t.Fatalf("console CSP: %q", csp)
			}
			if resp.Header.Get("Cache-Control") != "no-cache" {
				t.Fatalf("console Cache-Control: %q", resp.Header.Get("Cache-Control"))
			}
		}
		var conf struct {
			Vectors int `json:"vectors"`
			Report  struct {
				Pass, Fail int
			} `json:"report"`
		}
		get("http://"+c.Procs[0].API+"/api/v1/conformance", &conf)
		if conf.Vectors < 100 || conf.Report.Pass != conf.Vectors || conf.Report.Fail != 0 {
			t.Fatalf("conformance on member: %+v", conf)
		}
		for _, m := range v.Milestones {
			if m.ID == "M8" && m.State != "VERIFIED" {
				t.Fatalf("M8 milestone %s: %v", m.State, m.Gaps)
			}
		}
	})

	t.Run("leader loss: new leader, no lost writes, no workload interruption", func(t *testing.T) {
		// A committed write just before the failure.
		if err := c.Op.Do("POST", "/api/v1/apps/web/scale", map[string]int64{"replicas": 3}, nil); err != nil {
			t.Fatal(err)
		}
		before, _ := c.View()
		headBefore := before.Audit.Head
		leader, err := c.Leader()
		if err != nil {
			t.Fatal(err)
		}
		tr := startTraffic(c.Edge, "ha.test")
		time.Sleep(time.Second)
		killedAt := time.Now()
		if err := c.Kill(leader, syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		var newLeader string
		for time.Since(killedAt) < 20*time.Second {
			if l, err := c.Leader(); err == nil && l != leader {
				newLeader = l
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		elected := time.Since(killedAt)
		if newLeader == "" {
			t.Fatal("no new leader within 20s")
		}
		// A write through the new leader.
		var r struct {
			OK      bool   `json:"ok"`
			Message string `json:"message"`
		}
		if err := c.Op.Do("POST", "/api/v1/freeze", map[string]bool{"frozen": false}, &r); err != nil {
			t.Fatalf("write after failover: %v", err)
		}
		time.Sleep(3 * time.Second)
		ok, fail := tr.finish()
		t.Logf("leader %s killed; %s elected in %s; %d requests through the edge during failover, %d failed", leader, newLeader, elected.Round(time.Millisecond), ok, fail)
		if fail > 0 {
			t.Fatalf("workload traffic interrupted by control-plane leader loss: %d failures", fail)
		}
		after, err := c.View()
		if err != nil {
			t.Fatal(err)
		}
		if after.Audit.Head < headBefore || !after.Audit.Verification.OK {
			t.Fatalf("committed writes lost or chain broken: head %d → %d, verify %+v", headBefore, after.Audit.Head, after.Audit.Verification)
		}
		// The killed member rejoins and catches up.
		if err := c.Restart(leader); err != nil {
			t.Fatal(err)
		}
		p := c.Proc(leader)
		deadline := time.Now().Add(30 * time.Second)
		for {
			var h map[string]any
			resp, err := http.Get("http://" + p.API + "/api/v1/health")
			if err == nil {
				json.NewDecoder(resp.Body).Decode(&h)
				resp.Body.Close()
				if h["state"] == "follower" && int64(h["index"].(float64)) >= int64(after.ServedBy.Index) {
					break
				}
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s did not rejoin as a caught-up follower: %v", leader, h)
			}
			time.Sleep(200 * time.Millisecond)
		}
	})

	t.Run("backup verifies offline; total control-plane loss is restored without stopping workloads", func(t *testing.T) {
		var raw json.RawMessage
		if err := c.Op.Do("POST", "/api/v1/cp/backup", nil, &raw); err != nil {
			t.Fatal(err)
		}
		var env envelope.Envelope
		json.Unmarshal(raw, &env)
		b, st, err := control.VerifyBackup(&env)
		if err != nil {
			t.Fatal(err)
		}
		if st.Apps["web"] == nil || audit.Verify(st.Audit.Entries, 0, audit.Genesis) != nil {
			t.Fatal("backup content")
		}
		v, _ := c.View()
		pids := map[string]int64{}
		for _, r := range app(v, "web").Rows {
			pids[r.Assignment] = r.PID
		}
		// Lose every control-plane member and its data.
		for i := 1; i <= 3; i++ {
			name := fmt.Sprintf("cp-%d", i)
			c.Kill(name, syscall.SIGKILL)
			os.RemoveAll(c.Proc(name).Data)
		}
		tr := startTraffic(c.Edge, "ha.test")
		time.Sleep(2 * time.Second)
		// A fresh member at cp-1's address, bootstrapped with the same root.
		if err := c.Restart("cp-1"); err != nil {
			t.Fatal(err)
		}
		p := c.Proc("cp-1")
		code := ""
		for i := 0; i < 100 && code == ""; i++ {
			bb, _ := os.ReadFile(filepath.Join(p.Data, "bootstrap.code"))
			code = string(bb)
			time.Sleep(100 * time.Millisecond)
		}
		c.Op.Cfg.Endpoints = []string{p.API}
		if err := c.Op.Bootstrap(p.API, trimNL(code), ""); err != nil {
			t.Fatal(err)
		}
		var res struct {
			OK      bool   `json:"ok"`
			Message string `json:"message"`
		}
		if err := c.Op.Do("POST", "/api/v1/cp/restore", &env, &res); err != nil || !res.OK {
			t.Fatalf("restore: %v %+v", err, res)
		}
		if err := c.WaitFor(60*time.Second, "hosts reconnect to the restored plane and report the same workloads", func(v *control.View) bool {
			a := app(v, "web")
			if a == nil || a.Observed < 3 {
				return false
			}
			for _, r := range a.Rows {
				if r.PID != pids[r.Assignment] {
					return false
				}
			}
			return v.ServedBy.Index > b.Index
		}); err != nil {
			t.Fatal(err)
		}
		ok, fail := tr.finish()
		t.Logf("restored from backup at index %d; %d requests during total control-plane loss and restore, %d failed", b.Index, ok, fail)
		if fail > 0 {
			t.Fatalf("workloads interrupted by control-plane loss: %d failures", fail)
		}
		v, _ = c.View()
		found := false
		for _, e := range v.Audit.Entries {
			if e.Action == "cp-restore" {
				found = true
			}
		}
		if !found || !v.Audit.Verification.OK {
			t.Fatal("restore not recorded in a verifying ledger")
		}
	})
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
