package chaos

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/node"
)

// Scenarios is the chaos suite, in the order it runs.
func Scenarios() []Scenario {
	return []Scenario{
		leaderCrash(), cpTotalOutage(), hostCrash(), agentRestart(), journalCorruption(), networkPartition(),
		packetChaos(), clockSkew(), staleGenerationRollout(), replayAndForgery(), revokedHost(), storageReplicaLoss(),
		diskFull(), oomKill(), interruptedDeployment(), interruptedStorageWrite(), postgresOutage(),
	}
}

// Find returns a scenario by id.
func Find(id string) (Scenario, bool) {
	for _, s := range Scenarios() {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}

func replicaOn(v *control.View, app string, replica int64) *control.ReplicaView {
	a := App(v, app)
	if a == nil {
		return nil
	}
	for i := range a.Rows {
		if a.Rows[i].Replica == replica && a.Rows[i].Desired == "RUNNING" {
			return &a.Rows[i]
		}
	}
	return nil
}

func pidAlive(pid int64) bool { return pid > 0 && syscall.Kill(int(pid), 0) == nil }

// ------------------------------------------------------------ control plane

func leaderCrash() Scenario {
	return Scenario{ID: "leader-crash", Title: "Control-plane leader crash",
		Injection: "SIGKILL the raft leader while traffic flows through the edge",
		Invariant: "a new leader is elected, committed writes survive, new writes succeed, and no workload request fails",
		CPs:       3, Hosts: 3, Edges: 1,
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			before, _ := x.C.View()
			leader, err := x.C.Leader()
			if err != nil {
				return err
			}
			tr := x.StartTraffic("web.chaos.test", 20*time.Millisecond)
			time.Sleep(time.Second)
			t0 := time.Now()
			x.Step("SIGKILL leader %s", leader)
			if err := x.C.Kill(leader, syscall.SIGKILL); err != nil {
				return err
			}
			var nl string
			for time.Since(t0) < 20*time.Second && nl == "" {
				if l, err := x.C.Leader(); err == nil && l != leader {
					nl = l
				}
				time.Sleep(50 * time.Millisecond)
			}
			if nl == "" {
				return errors.New("no new leader within 20s")
			}
			x.Step("%s elected leader", nl)
			x.Measure("election_ms", "%d", time.Since(t0).Milliseconds())
			if err := x.C.Op.Do("POST", "/api/v1/apps/web/scale", map[string]int64{"replicas": 3}, nil); err != nil {
				return fmt.Errorf("write after failover: %w", err)
			}
			x.Step("write committed through the new leader")
			time.Sleep(2 * time.Second)
			ok, fail := tr.Stop()
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			after, err := x.C.View()
			if err != nil {
				return err
			}
			x.Measure("audit_head_before", "%d", before.Audit.Head)
			x.Measure("audit_head_after", "%d", after.Audit.Head)
			if after.Audit.Head < before.Audit.Head || !after.Audit.Verification.OK {
				return errors.New("committed audit entries lost or chain broken after failover")
			}
			if err := x.C.Restart(leader); err != nil {
				return err
			}
			x.Step("old leader restarted")
			x.Observed("new leader in %s ms; %d/%d requests succeeded during failover; audit %d → %d verified", x.rep.Measurements["election_ms"], ok, ok+fail, before.Audit.Head, after.Audit.Head)
			if fail > 0 {
				return fmt.Errorf("%d workload requests failed during control-plane failover", fail)
			}
			return nil
		}}
}

func cpTotalOutage() Scenario {
	return Scenario{ID: "cp-total-outage", Title: "Total control-plane outage",
		Injection: "SIGKILL every control-plane member for 15s, then restart",
		Invariant: "hosts enter offline-hold, admitted workloads keep running with the same PIDs, no request fails, buffered observations are delivered after recovery",
		CPs:       1, Hosts: 2, Edges: 1,
		App: strings.Replace(defaultApp, "replicas: 3", "replicas: 2", 1),
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 2, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 2, 30*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			pids := map[string]int64{}
			for _, r := range App(v, "web").Rows {
				pids[r.Assignment] = r.PID
			}
			tr := x.StartTraffic("web.chaos.test", 25*time.Millisecond)
			x.Step("SIGKILL cp-1")
			if err := x.C.Kill("cp-1", syscall.SIGKILL); err != nil {
				return err
			}
			deadline := time.Now().Add(25 * time.Second)
			for {
				st, err := x.C.HostStatus("host-a")
				if err == nil && st.Mode == "offline-hold" {
					x.Step("host-a entered offline-hold: %s", trunc(st.ModeDetail, 90))
					break
				}
				if time.Now().After(deadline) {
					return errors.New("host-a did not enter offline-hold")
				}
				time.Sleep(200 * time.Millisecond)
			}
			time.Sleep(10 * time.Second)
			for _, h := range []string{"host-a", "host-b"} {
				st, err := x.C.HostStatus(h)
				if err != nil {
					return err
				}
				x.Measure(h+"_outbox", "%d", st.Outbox)
				for id, ad := range st.Admitted {
					if ad.Inst.PID != pids[id] || !pidAlive(ad.Inst.PID) {
						return fmt.Errorf("%s workload %s disturbed during outage", h, id)
					}
				}
			}
			x.Step("workloads verified running with unchanged PIDs during the outage")
			if err := x.C.Restart("cp-1"); err != nil {
				return err
			}
			x.Step("cp-1 restarted")
			if err := x.C.WaitFor(40*time.Second, "hosts fresh again", func(v *control.View) bool {
				return Node(v, "host-a").LastObs.Freshness == "FRESH" && Node(v, "host-b").LastObs.Freshness == "FRESH"
			}); err != nil {
				return err
			}
			x.Step("hosts reporting again")
			time.Sleep(2 * time.Second)
			ok, fail := tr.Stop()
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			flushed := false
			for _, h := range []string{"host-a", "host-b"} {
				st, _ := x.C.HostStatus(h)
				if st != nil && st.Outbox == 0 {
					flushed = true
				}
			}
			x.Observed("offline-hold entered, workloads untouched, %d/%d requests OK during outage, buffered observations flushed: %v", ok, ok+fail, flushed)
			if fail > 0 {
				return fmt.Errorf("%d requests failed while the control plane was down", fail)
			}
			if !flushed {
				return errors.New("buffered observations were not delivered")
			}
			return nil
		}}
}

// ------------------------------------------------------------------ hosts

func hostCrash() Scenario {
	return Scenario{ID: "host-crash", Title: "Host crash",
		Injection: "SIGKILL a host agent and all of its workloads (machine loss)",
		Invariant: "the edge stops routing to the dead replica without failed requests, the host is marked lost, and the replica is rescheduled and observed running elsewhere",
		CPs:       1, Hosts: 4, Edges: 1, LostAfter: "6s",
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			r0 := replicaOn(v, "web", 0)
			victim := r0.NodeName
			tr := x.StartTraffic("web.chaos.test", 25*time.Millisecond)
			time.Sleep(time.Second)
			t0 := time.Now()
			killed, err := x.C.KillMachine(victim)
			if err != nil {
				return err
			}
			x.Step("killed %s and %d workload(s)", victim, killed)
			var lostAt, movedAt time.Duration
			if err := x.C.WaitFor(60*time.Second, "replica 0 running on another host", func(v *control.View) bool {
				if lostAt == 0 && Node(v, victim).Health == "lost" {
					lostAt = time.Since(t0)
					x.Step("%s marked lost", victim)
				}
				r := replicaOn(v, "web", 0)
				if r != nil && r.NodeName != victim && r.Observed == "RUNNING" && r.Freshness == "FRESH" {
					movedAt = time.Since(t0)
					x.Step("replica 0 running on %s", r.NodeName)
					return true
				}
				return false
			}); err != nil {
				return err
			}
			time.Sleep(2 * time.Second)
			ok, fail := tr.Stop()
			x.Measure("detect_lost_ms", "%d", lostAt.Milliseconds())
			x.Measure("rescheduled_ms", "%d", movedAt.Milliseconds())
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Observed("%s lost after %s, replica running elsewhere after %s; %d/%d requests OK", victim, lostAt.Round(time.Millisecond), movedAt.Round(time.Millisecond), ok, ok+fail)
			if fail > 0 {
				return fmt.Errorf("%d requests failed while a host died", fail)
			}
			return nil
		}}
}

func agentRestart() Scenario {
	return Scenario{ID: "agent-restart", Title: "Host agent restart",
		Injection: "SIGKILL dh-noded only (workloads keep running) and restart it",
		Invariant: "the restarted agent re-adopts every workload by PID and kernel start time without restarting it",
		CPs:       1, Hosts: 2, App: strings.Replace(defaultApp, "replicas: 3", "replicas: 2", 1),
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 2, 60*time.Second); err != nil {
				return err
			}
			before, _ := x.C.HostStatus("host-a")
			pids := map[string]int64{}
			for id, ad := range before.Admitted {
				pids[id] = ad.Inst.PID
			}
			if err := x.C.Kill("host-a", syscall.SIGKILL); err != nil {
				return err
			}
			x.Step("SIGKILL dh-noded on host-a")
			for id, pid := range pids {
				if !pidAlive(pid) {
					return fmt.Errorf("workload %s died with its agent", id)
				}
			}
			time.Sleep(2 * time.Second)
			if err := x.C.Restart("host-a"); err != nil {
				return err
			}
			x.Step("dh-noded restarted")
			time.Sleep(4 * time.Second)
			after, err := x.C.HostStatus("host-a")
			if err != nil {
				return err
			}
			for id, pid := range pids {
				ad := after.Admitted[id]
				if ad == nil || ad.Inst.PID != pid || ad.Restarts != before.Admitted[id].Restarts {
					return fmt.Errorf("workload %s was restarted or lost (pid %d → %+v)", id, pid, ad)
				}
			}
			res, err := hostLedger(x, "host-a")
			if err != nil {
				return err
			}
			readopt := strings.Contains(res, "workload-readopt")
			x.Observed("%d workload(s) re-adopted with unchanged PIDs; ledger records re-adoption: %v", len(pids), readopt)
			if !readopt {
				return errors.New("host ledger has no re-adoption record")
			}
			return nil
		}}
}

func hostLedger(x *Ctx, host string) (string, error) {
	p := x.C.Proc(host)
	addr, err := os.ReadFile(filepath.Join(p.Data, "status.addr"))
	if err != nil {
		return "", err
	}
	resp, err := http.Get("http://" + strings.TrimSpace(string(addr)) + "/ledger")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), nil
}

func journalCorruption() Scenario {
	return Scenario{ID: "journal-corruption", Title: "Host ledger corruption",
		Injection: "stop a host agent, flip bytes inside one entry of its hash-chained journal, restart it",
		Invariant: "the host detects the named break, enters ledger-corrupt mode, keeps admitted work, refuses new work, and never appends to the broken chain; after the local operator seals the journal (kept byte-for-byte) the host admits work again and its new chain records the break",
		CPs:       1, Hosts: 2, App: strings.Replace(defaultApp, "replicas: 3", "replicas: 1", 1),
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 1, 60*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			host := replicaOn(v, "web", 0).NodeName
			other := "host-a"
			if host == "host-a" {
				other = "host-b"
			}
			st, _ := x.C.HostStatus(other)
			if err := x.C.Kill(other, syscall.SIGTERM); err != nil {
				return err
			}
			jp := filepath.Join(x.C.Proc(other).Data, "journal.jsonl")
			raw, err := os.ReadFile(jp)
			if err != nil {
				return err
			}
			lines := strings.Split(string(raw), "\n")
			target := len(lines) / 2
			lines[target] = strings.Replace(lines[target], `"detail":"`, `"detail":"TAMPERED `, 1)
			if err := os.WriteFile(jp, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
				return err
			}
			x.Step("rewrote journal entry at line %d of %d on %s (ledger head was %d)", target+1, len(lines)-1, other, st.Ledger)
			if err := x.C.Restart(other); err != nil {
				return err
			}
			var brk string
			if err := x.C.WaitFor(20*time.Second, other+" reports ledger-corrupt", func(v *control.View) bool {
				n := Node(v, other)
				brk = n.ModeDetail
				return n.Mode == "ledger-corrupt"
			}); err != nil {
				return err
			}
			x.Step("%s mode ledger-corrupt: %s", other, brk)
			// New work for that host must be refused.
			if err := x.C.Op.Do("POST", "/api/v1/apps/web/scale", map[string]int64{"replicas": 2}, nil); err != nil {
				return err
			}
			var code string
			if err := x.C.WaitFor(20*time.Second, "new replica refused by the corrupt host", func(v *control.View) bool {
				for _, r := range App(v, "web").Rows {
					if r.NodeName == other {
						code = r.Code
						return r.Admitted == "REFUSED" && r.Code == "LEDGER_CORRUPT"
					}
				}
				return false
			}); err != nil {
				return fmt.Errorf("%w (last code %q)", err, code)
			}
			x.Step("new replica refused with %s", code)
			// Recovery: the local operator seals the broken journal.
			if err := x.C.Kill(other, syscall.SIGTERM); err != nil {
				return err
			}
			out, err := exec.Command(filepath.Join(x.C.Opt.Bin, "dh-noded"), "ledger-seal", "--data", x.C.Proc(other).Data,
				"--reason", "chaos: tampered entry inspected").CombinedOutput()
			if err != nil {
				return fmt.Errorf("ledger-seal: %v: %s", err, out)
			}
			x.Step("operator sealed the journal: %s", strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0])
			if err := x.C.Restart(other); err != nil {
				return err
			}
			if err := x.C.WaitFor(40*time.Second, "sealed host admits and runs the refused replica", func(v *control.View) bool {
				for _, r := range App(v, "web").Rows {
					if r.NodeName == other && r.Admitted == "ADMITTED" && r.Observed == "RUNNING" {
						return true
					}
				}
				return false
			}); err != nil {
				return err
			}
			x.Step("%s admits work again on a new, verifying chain", other)
			x.Observed("break named (%s); new work refused with LEDGER_CORRUPT; after an operator seal the host admitted the replica", brk)
			return nil
		}}
}

// ---------------------------------------------------------------- network

func networkPartition() Scenario {
	return Scenario{ID: "network-partition", Title: "Network partition of one host",
		Injection: "drop every WireGuard packet to and from one host at the fault proxies and cut its control-plane channel for 25s, then heal",
		Invariant: "the isolated host keeps its admitted workload, gossip suspects it, the control plane marks it lost and reschedules, no request fails, and after healing the cluster converges to exactly the desired replicas",
		CPs:       1, Hosts: 4, Edges: 1, Proxy: true, LostAfter: "8s",
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			r0 := replicaOn(v, "web", 0)
			victim, pid := r0.NodeName, r0.PID
			tr := x.StartTraffic("web.chaos.test", 25*time.Millisecond)
			t0 := time.Now()
			x.Ctl.Set(Faults{Isolated: map[string]bool{victim: true}})
			if _, err := x.C.Chaos(victim, node.ChaosRequest{Fault: "drop-cp", On: true}); err != nil {
				return err
			}
			x.Step("isolated %s (WireGuard dropped both ways, control-plane channel cut)", victim)
			var suspected, lost, moved time.Duration
			if err := x.C.WaitFor(60*time.Second, "partition detected and replica rescheduled", func(v *control.View) bool {
				for _, n := range v.Nodes {
					if n.Name == victim || n.Mesh == nil {
						continue
					}
					for _, p := range n.Mesh.Peers {
						if p.Node == r0.Node && (p.Gossip == "suspect" || p.Gossip == "dead") && suspected == 0 {
							suspected = time.Since(t0)
							x.Step("%s sees %s as %s in gossip", n.Name, victim, p.Gossip)
						}
					}
				}
				if lost == 0 && Node(v, victim).Health == "lost" {
					lost = time.Since(t0)
					x.Step("control plane marked %s lost", victim)
				}
				r := replicaOn(v, "web", 0)
				if r != nil && r.NodeName != victim && r.Observed == "RUNNING" && r.Freshness == "FRESH" && moved == 0 {
					moved = time.Since(t0)
					x.Step("replica 0 running on %s", r.NodeName)
				}
				return lost > 0 && moved > 0
			}); err != nil {
				return err
			}
			st, _ := x.C.HostStatus(victim)
			held := st != nil && st.Mode == "offline-hold" && pidAlive(pid)
			x.Step("isolated host mode %s, original workload alive: %v", st.Mode, pidAlive(pid))
			if !held {
				return errors.New("isolated host did not hold its admitted workload")
			}
			time.Sleep(time.Until(t0.Add(25 * time.Second)))
			x.Ctl.Set(Faults{})
			_, _ = x.C.Chaos(victim, node.ChaosRequest{Fault: "drop-cp", On: false})
			healed := time.Now()
			x.Step("partition healed")
			if err := x.C.WaitFor(60*time.Second, "converged to exactly 3 running replicas", func(v *control.View) bool {
				running := 0
				for _, r := range App(v, "web").Rows {
					if r.Observed == "RUNNING" && r.Freshness == "FRESH" {
						running++
					}
				}
				return running == 3 && !pidAlive(pid)
			}); err != nil {
				return err
			}
			x.Step("old instance on %s stopped by a signed stop; 3 replicas running", victim)
			ok, fail := tr.Stop()
			x.Measure("gossip_suspect_ms", "%d", suspected.Milliseconds())
			x.Measure("marked_lost_ms", "%d", lost.Milliseconds())
			x.Measure("rescheduled_ms", "%d", moved.Milliseconds())
			x.Measure("converged_after_heal_ms", "%d", time.Since(healed).Milliseconds())
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Measure("proxy_packets", "%v", x.Ctl.Stats())
			x.Observed("gossip suspect %s, lost %s, rescheduled %s; converged after heal; %d/%d requests OK", suspected.Round(time.Millisecond), lost.Round(time.Millisecond), moved.Round(time.Millisecond), ok, ok+fail)
			if fail > 0 {
				return fmt.Errorf("%d requests failed during the partition", fail)
			}
			return nil
		}}
}

func packetChaos() Scenario {
	return Scenario{ID: "packet-chaos", Title: "Packet loss, duplication, reordering and delay",
		Injection: "on every WireGuard path: 5% loss, 10% duplication, 10% reordering, +40ms delay for 20s",
		Invariant: "WireGuard and TCP absorb the faults: no request fails, the measured mesh RTT reflects the injected delay, and it returns to baseline when faults are cleared",
		CPs:       1, Hosts: 3, Edges: 1, Proxy: true,
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			rtt := func() (int64, int) {
				v, _ := x.C.View()
				var sum int64
				n := 0
				for _, nd := range v.Nodes {
					if nd.Mesh == nil {
						continue
					}
					for _, p := range nd.Mesh.Peers {
						if p.RTTUs > 0 && time.Since(time.UnixMilli(p.RTTAt)) < 12*time.Second {
							sum += p.RTTUs
							n++
						}
					}
				}
				if n == 0 {
					return -1, 0
				}
				return sum / int64(n), n
			}
			time.Sleep(12 * time.Second)
			base, bn := rtt()
			x.Measure("baseline_mesh_rtt_us", "%d (mean of %d host-measured samples)", base, bn)
			tr := x.StartTraffic("web.chaos.test", 20*time.Millisecond)
			x.Ctl.Set(Faults{DropPct: 5, DupPct: 10, ReorderPct: 10, Delay: 40 * time.Millisecond})
			x.Step("faults on: loss 5%%, dup 10%%, reorder 10%%, delay 40ms")
			time.Sleep(22 * time.Second)
			during, dn := rtt()
			x.Measure("faulted_mesh_rtt_us", "%d (mean of %d samples)", during, dn)
			x.Ctl.Set(Faults{})
			x.Step("faults cleared")
			time.Sleep(12 * time.Second)
			after, an := rtt()
			x.Measure("recovered_mesh_rtt_us", "%d (mean of %d samples)", after, an)
			ok, fail := tr.Stop()
			p50, p99 := tr.Latency()
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Measure("request_latency_p50_ms", "%.1f", float64(p50.Microseconds())/1000)
			x.Measure("request_latency_p99_ms", "%.1f", float64(p99.Microseconds())/1000)
			x.Measure("proxy_packets", "%v", x.Ctl.Stats())
			x.Observed("RTT %dµs → %dµs under faults → %dµs after; %d/%d requests OK (p50 %s, p99 %s)", base, during, after, ok, ok+fail, p50.Round(time.Millisecond), p99.Round(time.Millisecond))
			if fail > 0 {
				return fmt.Errorf("%d requests failed under packet faults", fail)
			}
			if during <= base+40_000 {
				return fmt.Errorf("measured RTT did not reflect the injected 40ms delay (%dµs vs baseline %dµs)", during, base)
			}
			if after > base+20_000 {
				return fmt.Errorf("RTT did not recover (%dµs)", after)
			}
			return nil
		}}
}

func clockSkew() Scenario {
	return Scenario{ID: "clock-skew", Title: "Host clock skew",
		Injection: "shift one host's clock +120s, then −120s, then restore",
		Invariant: "a skewed host detects it from signed bundle timestamps, holds admitted work, refuses new work (CLOCK_SKEW / stale plane), and returns to normal when the clock is corrected",
		CPs:       1, Hosts: 2, App: strings.Replace(defaultApp, "replicas: 3", "replicas: 1", 1),
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 1, 60*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			holder := replicaOn(v, "web", 0)
			other := "host-a"
			if holder.NodeName == "host-a" {
				other = "host-b"
			}
			for _, off := range []int64{120_000, -120_000} {
				if _, err := x.C.Chaos(holder.NodeName, node.ChaosRequest{Fault: "clock-offset", Value: off}); err != nil {
					return err
				}
				x.Step("%s clock offset %+dms", holder.NodeName, off)
				var detail string
				if err := x.C.WaitFor(20*time.Second, "skew detected", func(v *control.View) bool {
					n := Node(v, holder.NodeName)
					detail = n.ModeDetail
					return n.Mode == "clock-skew"
				}); err != nil {
					return err
				}
				x.Step("detected: %s", trunc(detail, 110))
				if !pidAlive(holder.PID) {
					return errors.New("admitted workload stopped by clock skew")
				}
			}
			if _, err := x.C.Chaos(holder.NodeName, node.ChaosRequest{Fault: "clock-offset", Value: 0}); err != nil {
				return err
			}
			// New work on a skewed host is refused: skew the empty host (its
			// clock behind, so bundles look like they come from the future),
			// then scale so the second replica can only land there.
			if _, err := x.C.Chaos(other, node.ChaosRequest{Fault: "clock-offset", Value: -120_000}); err != nil {
				return err
			}
			x.Step("%s clock offset -120000ms; scaling to 2", other)
			if err := x.C.Op.Do("POST", "/api/v1/apps/web/scale", map[string]int64{"replicas": 2}, nil); err != nil {
				return err
			}
			var code string
			if err := x.C.WaitFor(30*time.Second, "new replica refused by the skewed host", func(v *control.View) bool {
				for _, r := range App(v, "web").Rows {
					if r.NodeName == other {
						code = r.Code
						return r.Admitted == "REFUSED" && r.Code == "CLOCK_SKEW"
					}
				}
				return false
			}); err != nil {
				return fmt.Errorf("%w (last code %q)", err, code)
			}
			x.Step("new work refused on %s with %s", other, code)
			if _, err := x.C.Chaos(other, node.ChaosRequest{Fault: "clock-offset", Value: 0}); err != nil {
				return err
			}
			x.Step("clocks restored")
			if err := x.WaitRunning("web", 2, 40*time.Second); err != nil {
				return err
			}
			if !pidAlive(holder.PID) {
				return errors.New("workload lost across clock skew")
			}
			x.Observed("skew detected in both directions from bundle timestamps; admitted workload (pid %d) held; new work refused with CLOCK_SKEW; both replicas running after correction", holder.PID)
			return nil
		}}
}

func staleGenerationRollout() Scenario {
	return Scenario{ID: "stale-generation", Title: "Rollout while a host is cut off",
		Injection: "cut one host from the control plane, roll the app to a new generation, heal",
		Invariant: "the cut-off host keeps the generation it admitted (never runs an unsigned one), the console shows it as stale, other hosts roll within maxUnavailable, and after healing every replica runs the new generation",
		CPs:       1, Hosts: 3, Edges: 1,
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			cut := replicaOn(v, "web", 1)
			tr := x.StartTraffic("web.chaos.test", 25*time.Millisecond)
			if _, err := x.C.Chaos(cut.NodeName, node.ChaosRequest{Fault: "drop-cp", On: true}); err != nil {
				return err
			}
			x.Step("%s cut from the control plane", cut.NodeName)
			m := strings.Replace(fmt.Sprintf(defaultApp, x.Image), "health:", "env: {MESSAGE: generation-two}\n  health:", 1)
			res, err := x.C.Op.Apply([]byte(m))
			if err != nil || !res.OK {
				return fmt.Errorf("apply: %v %v", err, res.Message)
			}
			x.Step("applied generation 2: %s", res.Message)
			var staleSeen bool
			if err := x.C.WaitFor(60*time.Second, "other replicas at generation 2", func(v *control.View) bool {
				a := App(v, "web")
				n := 0
				for _, r := range a.Rows {
					if r.Desired == "RUNNING" && r.ObservedGen == 2 && r.Observed == "RUNNING" {
						n++
					}
					if r.NodeName == cut.NodeName && strings.Contains(r.Status, "STALE") {
						staleSeen = true
					}
				}
				return n >= 2
			}); err != nil {
				return err
			}
			st, _ := x.C.HostStatus(cut.NodeName)
			for _, ad := range st.Admitted {
				if ad.Generation != 1 || !pidAlive(ad.Inst.PID) {
					return fmt.Errorf("cut-off host changed generation without a signed bundle: %+v", ad)
				}
			}
			x.Step("cut-off host still runs generation 1 (stale evidence shown: %v)", staleSeen)
			_, _ = x.C.Chaos(cut.NodeName, node.ChaosRequest{Fault: "drop-cp", On: false})
			x.Step("healed")
			if err := x.C.WaitFor(60*time.Second, "all replicas at generation 2", func(v *control.View) bool {
				a := App(v, "web")
				n := 0
				for _, r := range a.Rows {
					if r.Desired == "RUNNING" && r.ObservedGen == 2 && r.Observed == "RUNNING" && r.Freshness == "FRESH" {
						n++
					}
				}
				return n == 3
			}); err != nil {
				return err
			}
			ok, fail := tr.Stop()
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Observed("cut-off host held generation 1 until healed; rollout completed; %d/%d requests OK", ok, ok+fail)
			if fail > 0 {
				return fmt.Errorf("%d requests failed during the rollout", fail)
			}
			return nil
		}}
}

func replayAndForgery() Scenario {
	return Scenario{ID: "replay-forgery", Title: "Replayed, forged and tampered observations",
		Injection: "resend a committed host observation; send one signed by a foreign key; send one with a flipped field",
		Invariant: "each is rejected with a specific reason and recorded; the host's real state is unchanged",
		CPs:       1, Hosts: 2, App: strings.Replace(defaultApp, "replicas: 3", "replicas: 2", 1),
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 2, 60*time.Second); err != nil {
				return err
			}
			var st struct {
				Nodes map[string]struct {
					Name   string             `json:"name"`
					ObsEnv *envelope.Envelope `json:"obsEnv"`
				} `json:"nodes"`
			}
			if err := x.C.Op.Do("GET", "/api/v1/state", nil, &st); err != nil {
				return err
			}
			var env *envelope.Envelope
			for _, n := range st.Nodes {
				if n.Name == "host-a" {
					env = n.ObsEnv
				}
			}
			if env == nil {
				return errors.New("no committed observation")
			}
			post := func(e *envelope.Envelope) (string, error) {
				b, _ := jsonWire(map[string]any{"observations": []*envelope.Envelope{e}})
				resp, err := http.Post("http://"+x.C.Procs[0].API+"/v1/observe", "application/json", bytes.NewReader(b))
				if err != nil {
					return "", err
				}
				defer resp.Body.Close()
				rb, _ := io.ReadAll(resp.Body)
				return string(rb), nil
			}
			out1, err := post(env)
			if err != nil {
				return err
			}
			x.Step("replay → %s", trunc(out1, 160))
			var o api.Observation
			_ = env.Decode(&o)
			o.Seq += 1_000_000
			imp, _ := identity.Generate()
			forged, _ := envelope.Sign(imp, env.Signer, envelope.KindObservation, o)
			out2, _ := post(forged)
			x.Step("foreign key → %s", trunc(out2, 160))
			tampered := *env
			tampered.Payload = []byte(strings.Replace(string(env.Payload), `"observed":"running"`, `"observed":"stopped"`, 1))
			out3, _ := post(&tampered)
			x.Step("tampered → %s", trunc(out3, 160))
			if !strings.Contains(out1, "replay detected") || !strings.Contains(out2, "wrong host key") || !strings.Contains(out3, "rejected") {
				return errors.New("an injected message was not rejected with its specific reason")
			}
			v, _ := x.C.View()
			reasons := map[string]bool{}
			for _, r := range v.Rejections {
				reasons[r.Reason] = true
			}
			x.Measure("rejection_reasons", "%v", keys(reasons))
			if App(v, "web").Observed != 2 {
				return errors.New("host state changed by injected messages")
			}
			x.Observed("replay, foreign-key and tampered observations rejected with specific reasons; state unchanged")
			return nil
		}}
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func revokedHost() Scenario {
	return Scenario{ID: "revoked-host", Title: "Host revocation under traffic",
		Injection: "revoke a host that serves a replica while traffic flows",
		Invariant: "the revoked host keeps its admitted workload but is removed from the mesh and the edge routing table, the replica is rescheduled, and no request fails",
		CPs:       1, Hosts: 4, Edges: 1,
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			v, _ := x.C.View()
			r := replicaOn(v, "web", 2)
			tr := x.StartTraffic("web.chaos.test", 25*time.Millisecond)
			if err := x.C.Op.Do("POST", "/api/v1/nodes/"+r.Node+"/revoke", map[string]string{"reason": "chaos"}, nil); err != nil {
				return err
			}
			t0 := time.Now()
			x.Step("revoked %s", r.NodeName)
			if err := x.C.WaitFor(40*time.Second, "replica moved and revoked host out of routing", func(v *control.View) bool {
				nr := replicaOn(v, "web", 2)
				for _, e := range v.Edge.Edges {
					if e.Obs == nil {
						continue
					}
					for _, rt := range e.Obs.Routes {
						for _, ep := range rt.Endpoints {
							if ep.Node == r.Node {
								return false
							}
						}
					}
				}
				return nr != nil && nr.Node != r.Node && nr.Observed == "RUNNING"
			}); err != nil {
				return err
			}
			x.Step("replica 2 running on another host; %s removed from routing", r.NodeName)
			x.Measure("reschedule_ms", "%d", time.Since(t0).Milliseconds())
			mark := tr.ServedBy("web/r2@" + r.Node)
			time.Sleep(3 * time.Second)
			after := tr.ServedBy("web/r2@" + r.Node)
			ok, fail := tr.Stop()
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Measure("served_by_revoked_after_removal", "%d", after-mark)
			x.Observed("revoked host removed from routing and mesh; replica rescheduled; %d/%d requests OK; 0 routed to the revoked host after removal: %v", ok, ok+fail, after == mark)
			if fail > 0 || after != mark {
				return fmt.Errorf("failures %d, requests to revoked host after removal %d", fail, after-mark)
			}
			return nil
		}}
}

// --------------------------------------------------------------- storage

const kvApp = `apiVersion: dh/v1
kind: Application
metadata: {name: kv}
spec:
  replicas: 1
  image: %s
  resources: {cpu: 100m, mem: 64Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  volumes: [{name: data, size: 512Mi, mount: /data, durability: {replicas: 3}, snapshot: {every: 2s, retain: 4}}]
  ingress: [{host: kv.chaos.test, port: http, tls: none}]
  health: {http: /healthz, interval: 1s}
`

func kvPut(x *Ctx, key string, data []byte) error {
	req, _ := http.NewRequest("PUT", "http://"+x.C.Edge+"/kv/"+key, bytes.NewReader(data))
	req.Host = "kv.chaos.test"
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		return fmt.Errorf("put: %s", resp.Status)
	}
	return nil
}

func kvGet(x *Ctx, key string) ([]byte, int, error) {
	req, _ := http.NewRequest("GET", "http://"+x.C.Edge+"/kv/"+key, nil)
	req.Host = "kv.chaos.test"
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func kvVol(v *control.View) *control.VolumeView {
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

func kvReady(x *Ctx, payload []byte) error {
	if err := x.WaitRunning("kv", 1, 60*time.Second); err != nil {
		return err
	}
	if err := x.WaitRouting("kv.chaos.test", 1, 30*time.Second); err != nil {
		return err
	}
	if err := kvPut(x, "alpha", payload); err != nil {
		return err
	}
	return x.C.WaitFor(60*time.Second, "committed snapshot verified on 3 replicas", func(v *control.View) bool {
		vol := kvVol(v)
		return vol != nil && vol.CommittedRef != nil && vol.CommittedRef.Bytes >= int64(len(payload)) && vol.State == "HEALTHY" && vol.Verified >= 3
	})
}

func storageReplicaLoss() Scenario {
	return Scenario{ID: "storage-replica-loss", Title: "Loss of a storage replica",
		Injection: "delete every stored object on one replica host while it runs",
		Invariant: "anti-entropy detects the missing objects by Merkle comparison, repairs them from peers with signed evidence, and the volume returns to HEALTHY with identical data",
		CPs:       1, Hosts: 4, Edges: 1, HostArgs: []string{"--anti-entropy", "3s"}, App: kvApp,
		Run: func(x *Ctx) error {
			payload := make([]byte, 2<<20)
			rand.Read(payload)
			if err := kvReady(x, payload); err != nil {
				return err
			}
			v, _ := x.C.View()
			vol := kvVol(v)
			var member string
			for _, m := range vol.Members {
				if m != vol.Primary {
					member = nodeName(v, m)
				}
			}
			repairsBefore := len(v.Repairs)
			objs := filepath.Join(x.C.Proc(member).Data, "cas", "objects")
			if err := os.RemoveAll(objs); err != nil {
				return err
			}
			os.MkdirAll(objs, 0o700)
			t0 := time.Now()
			x.Step("deleted all stored objects on %s", member)
			var items int
			if err := x.C.WaitFor(60*time.Second, "repair evidence and HEALTHY volume", func(v *control.View) bool {
				items = 0
				for _, r := range v.Repairs[min(repairsBefore, len(v.Repairs)):] {
					if nodeName(v, r.R.Node) == member {
						for _, it := range r.R.Items {
							if it.Verified {
								items++
							}
						}
					}
				}
				vv := kvVol(v)
				return items > 0 && vv.State == "HEALTHY" && vv.Verified >= 3
			}); err != nil {
				return err
			}
			x.Measure("repair_ms", "%d", time.Since(t0).Milliseconds())
			x.Measure("objects_repaired", "%d", items)
			got, code, err := kvGet(x, "alpha")
			if err != nil || code != 200 || !bytes.Equal(got, payload) {
				return fmt.Errorf("data changed after repair: %d %v", code, err)
			}
			x.Observed("%d object(s) repaired on %s with verified evidence in %s ms; data identical", items, member, x.rep.Measurements["repair_ms"])
			return nil
		}}
}

func diskFull() Scenario {
	return Scenario{ID: "disk-full", Title: "Disk full on a replica (injected ENOSPC)",
		Injection: "make every new write on one replica host fail with ENOSPC (storage-layer injection), write new data, then clear it",
		Invariant: "the volume is reported DEGRADED — never falsely HEALTHY — while the replica cannot store data; writes still commit on the other replicas; after the disk frees, anti-entropy restores HEALTHY",
		CPs:       1, Hosts: 4, Edges: 1, HostArgs: []string{"--anti-entropy", "3s"}, App: kvApp,
		Run: func(x *Ctx) error {
			payload := make([]byte, 1<<20)
			rand.Read(payload)
			if err := kvReady(x, payload); err != nil {
				return err
			}
			v, _ := x.C.View()
			vol := kvVol(v)
			var member string
			for _, m := range vol.Members {
				if m != vol.Primary {
					member = nodeName(v, m)
				}
			}
			if _, err := x.C.Chaos(member, node.ChaosRequest{Fault: "enospc", On: true}); err != nil {
				return err
			}
			x.Step("ENOSPC injected on %s", member)
			more := make([]byte, 3<<20)
			rand.Read(more)
			if err := kvPut(x, "beta", more); err != nil {
				return err
			}
			var degraded string
			if err := x.C.WaitFor(60*time.Second, "new snapshot committed while the full replica is DEGRADED", func(v *control.View) bool {
				vv := kvVol(v)
				degraded = vv.State + ": " + vv.Detail
				return vv.CommittedRef != nil && vv.CommittedRef.Bytes >= int64(len(payload)+len(more)) && vv.State == "DEGRADED"
			}); err != nil {
				return fmt.Errorf("%w (last: %s)", err, degraded)
			}
			x.Step("reported %s", degraded)
			if _, err := x.C.Chaos(member, node.ChaosRequest{Fault: "enospc", On: false}); err != nil {
				return err
			}
			x.Step("ENOSPC cleared")
			if err := x.C.WaitFor(60*time.Second, "HEALTHY again", func(v *control.View) bool {
				vv := kvVol(v)
				return vv.State == "HEALTHY" && vv.CommittedRef.Bytes >= int64(len(payload)+len(more))
			}); err != nil {
				return err
			}
			got, code, _ := kvGet(x, "beta")
			if code != 200 || !bytes.Equal(got, more) {
				return errors.New("data written during the fault is not intact")
			}
			x.Observed("volume shown DEGRADED while %s was full (%s); HEALTHY after recovery; data intact", member, degraded)
			return nil
		}}
}

func interruptedStorageWrite() Scenario {
	return Scenario{ID: "interrupted-storage-write", Title: "Host killed during a storage write",
		Injection: "write 24 MiB, then SIGKILL the primary host (agent and workload) while the new snapshot is still pending",
		Invariant: "committed data is intact after recovery, and the interrupted write is either fully committed or absent — never partial",
		CPs:       1, Hosts: 4, Edges: 1, LostAfter: "6s", HostArgs: []string{"--anti-entropy", "3s"}, App: kvApp,
		Run: func(x *Ctx) error {
			payload := make([]byte, 2<<20)
			rand.Read(payload)
			if err := kvReady(x, payload); err != nil {
				return err
			}
			big := make([]byte, 24<<20)
			rand.Read(big)
			if err := kvPut(x, "beta", big); err != nil {
				return err
			}
			var pending bool
			if err := x.C.WaitFor(30*time.Second, "a snapshot carrying the write", func(v *control.View) bool {
				for _, sn := range kvVol(v).Snapshots {
					if sn.Bytes >= int64(len(big)) {
						pending = sn.State == "pending"
						return true
					}
				}
				return false
			}); err != nil {
				return err
			}
			v, _ := x.C.View()
			prim := nodeName(v, kvVol(v).Primary)
			if _, err := x.C.KillMachine(prim); err != nil {
				return err
			}
			x.Step("killed %s while the snapshot was %s", prim, map[bool]string{true: "pending", false: "already committed"}[pending])
			if err := x.C.WaitFor(90*time.Second, "replica recovered elsewhere", func(v *control.View) bool {
				r := replicaOn(v, "kv", 0)
				return r != nil && r.NodeName != prim && r.Observed == "RUNNING"
			}); err != nil {
				return err
			}
			var alpha, beta []byte
			var bc int
			if err := x.C.WaitFor(30*time.Second, "data readable", func(*control.View) bool {
				a, ac, e1 := kvGet(x, "alpha")
				b, c, e2 := kvGet(x, "beta")
				alpha, beta, bc = a, b, c
				return e1 == nil && e2 == nil && ac == 200 && (c == 200 || c == 404)
			}); err != nil {
				return err
			}
			if !bytes.Equal(alpha, payload) {
				return errors.New("committed data corrupted")
			}
			outcome := "absent (rolled back to the committed snapshot)"
			switch {
			case bc == 404:
			case bytes.Equal(beta, big):
				outcome = "fully committed before the kill"
			default:
				return fmt.Errorf("PARTIAL DATA: %d of %d bytes", len(beta), len(big))
			}
			x.Measure("snapshot_pending_at_kill", "%v", pending)
			x.Observed("committed data intact; interrupted write %s", outcome)
			return nil
		}}
}

// --------------------------------------------------------------- runtime

func oomKill() Scenario {
	return Scenario{ID: "oom", Title: "Out-of-memory kill (Docker runtime)",
		Injection: "run a container limited to 32 MiB that allocates without bound",
		Invariant: "the container runtime kills it (OOMKilled=true), the host reports oom-killed from runtime evidence, audits it, and restarts it with backoff",
		CPs:       1, Hosts: 1, Needs: []string{"docker"}, App: "-",
		Run: func(x *Ctx) error {
			img, err := busyboxDigest()
			if err != nil {
				return fmt.Errorf("%w: %v", ErrSkip, err)
			}
			x.Step("using %s", img)
			m := fmt.Sprintf(`apiVersion: dh/v1
kind: Application
metadata: {name: hog}
spec:
  replicas: 1
  image: %s
  command: [sh, -c, "head -c 400m /dev/zero | tail"]
  resources: {cpu: 200m, mem: 32Mi}
  placement: {tiers: [trusted], spread: none, antiAffinity: none}
`, img)
			res, err := x.C.Op.Apply([]byte(m))
			if err != nil || !res.OK {
				return fmt.Errorf("apply: %v %s", err, res.Message)
			}
			var row *control.ReplicaView
			if err := x.C.WaitFor(120*time.Second, "OOM kill observed", func(v *control.View) bool {
				row = replicaOn(v, "hog", 0)
				return row != nil && row.Observed == "OOM-KILLED"
			}); err != nil {
				return err
			}
			x.Step("observed OOM-KILLED on %s: %s", row.NodeName, trunc(row.Reason, 100))
			if err := x.C.WaitFor(60*time.Second, "restart with backoff", func(v *control.View) bool {
				r := replicaOn(v, "hog", 0)
				return r != nil && r.Restarts >= 1
			}); err != nil {
				return err
			}
			v, _ := x.C.View()
			audited := false
			for _, e := range v.Audit.Entries {
				if e.Action == "workload-oom" {
					audited = true
				}
			}
			x.Observed("container OOM-killed by the runtime under a 32 MiB limit; reported and audited: %v; restarted with backoff", audited)
			if !audited {
				return errors.New("OOM not in the audit ledger")
			}
			_ = x.C.Op.Do("POST", "/api/v1/apps/hog/delete", nil, nil)
			return nil
		}}
}

func interruptedDeployment() Scenario {
	return Scenario{ID: "interrupted-deployment", Title: "Leader crash during a rolling update",
		Injection: "apply a new generation and SIGKILL the raft leader as soon as the first replica has rolled",
		Invariant: "the new leader resumes the rollout, it completes, and requests keep succeeding (maxUnavailable respected)",
		CPs:       3, Hosts: 3, Edges: 1,
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 60*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			tr := x.StartTraffic("web.chaos.test", 25*time.Millisecond)
			m := strings.Replace(fmt.Sprintf(defaultApp, x.Image), "health:", "env: {MESSAGE: rollout}\n  health:", 1)
			if res, err := x.C.Op.Apply([]byte(m)); err != nil || !res.OK {
				return fmt.Errorf("apply: %v", err)
			}
			x.Step("generation 2 applied")
			if err := x.C.WaitFor(60*time.Second, "first replica rolled", func(v *control.View) bool {
				for _, r := range App(v, "web").Rows {
					if r.ObservedGen == 2 && r.Observed == "RUNNING" {
						return true
					}
				}
				return false
			}); err != nil {
				return err
			}
			leader, err := x.C.Leader()
			if err != nil {
				return err
			}
			if err := x.C.Kill(leader, syscall.SIGKILL); err != nil {
				return err
			}
			x.Step("SIGKILL leader %s mid-rollout", leader)
			if err := x.C.WaitFor(90*time.Second, "rollout complete", func(v *control.View) bool {
				n := 0
				for _, r := range App(v, "web").Rows {
					if r.Desired == "RUNNING" && r.ObservedGen == 2 && r.Observed == "RUNNING" && r.Freshness == "FRESH" {
						n++
					}
				}
				return n == 3
			}); err != nil {
				return err
			}
			x.Step("all replicas at generation 2")
			time.Sleep(2 * time.Second)
			ok, fail := tr.Stop()
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Observed("rollout resumed by the new leader and completed; %d/%d requests OK", ok, ok+fail)
			if fail > 0 {
				return fmt.Errorf("%d requests failed during the interrupted rollout", fail)
			}
			return nil
		}}
}

func postgresOutage() Scenario {
	return Scenario{ID: "postgres-outage", Title: "Evidence database outage",
		Injection: "stop the Postgres evidence mirror, keep operating, restart it",
		Invariant: "while Postgres is down the console reports JOURNAL ONLY (never DB COMMITTED), writes still commit through raft, and the mirror catches up to DB COMMITTED after recovery",
		CPs:       1, Hosts: 1, Needs: []string{"docker"}, App: "-", Postgres: true,
		Run: func(x *Ctx) error {
			durability := func() string {
				v, err := x.C.View()
				if err != nil {
					return ""
				}
				return v.Cluster.Durability
			}
			if err := x.C.WaitFor(60*time.Second, "DB COMMITTED", func(v *control.View) bool { return v.Cluster.Durability == "DB COMMITTED" }); err != nil {
				return fmt.Errorf("%w (%s)", err, durability())
			}
			x.Step("mirror reports DB COMMITTED")
			if out, err := exec.Command("docker", "stop", "-t", "1", x.PG).CombinedOutput(); err != nil {
				return fmt.Errorf("docker stop: %s", out)
			}
			x.Step("postgres stopped")
			// Writes keep committing through raft.
			var r struct {
				OK bool `json:"ok"`
			}
			for i := 0; i < 3; i++ {
				if err := x.C.Op.Do("POST", "/api/v1/freeze", map[string]bool{"frozen": i%2 == 0}, &r); err != nil || !r.OK {
					return fmt.Errorf("write with the database down: %v", err)
				}
			}
			x.Step("3 writes committed through raft with the database down")
			var d string
			if err := x.C.WaitFor(30*time.Second, "JOURNAL ONLY reported", func(v *control.View) bool {
				d = v.Cluster.Durability
				return strings.HasPrefix(d, "JOURNAL ONLY")
			}); err != nil {
				return err
			}
			x.Step("console: %s", trunc(d, 120))
			if out, err := exec.Command("docker", "start", x.PG).CombinedOutput(); err != nil {
				return fmt.Errorf("docker start: %s", out)
			}
			x.Step("postgres started")
			if err := x.C.WaitFor(90*time.Second, "mirror caught up", func(v *control.View) bool { return v.Cluster.Durability == "DB COMMITTED" }); err != nil {
				return fmt.Errorf("%w (%s)", err, durability())
			}
			v, _ := x.C.View()
			x.Measure("mirror_audit_seq", "%d", v.Cluster.Mirror.AuditSeq)
			x.Measure("audit_head", "%d", v.Audit.Head)
			x.Observed("DB COMMITTED → JOURNAL ONLY while down (writes kept committing) → DB COMMITTED with audit seq %d of %d mirrored", v.Cluster.Mirror.AuditSeq, v.Audit.Head)
			return nil
		}}
}

func jsonWire(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

// busyboxDigest pulls busybox and returns a digest-pinned reference.
func busyboxDigest() (string, error) {
	if out, err := exec.Command("docker", "pull", "-q", "busybox:1.36").CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker pull busybox: %s", strings.TrimSpace(string(out)))
	}
	out, err := exec.Command("docker", "image", "inspect", "--format", "{{index .RepoDigests 0}}", "busybox:1.36").Output()
	if err != nil {
		return "", err
	}
	ref := strings.TrimSpace(string(out))
	if !strings.Contains(ref, "@sha256:") {
		return "", fmt.Errorf("no repo digest for busybox (%q)", ref)
	}
	return ref, nil
}

func dockerVersion() string {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
