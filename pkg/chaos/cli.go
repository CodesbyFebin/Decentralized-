package chaos

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"decentralized.host/pkg/cli"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/node"
)

func init() {
	cli.Register("chaos list", "list chaos scenarios with their injection and invariant", cmdList)
	cli.Register("chaos run", "run chaos scenarios on real disposable clusters: [--scenario a,b|all] [--out DIR] [--submit]", cmdRun)
	cli.Register("chaos soak", "long-duration randomized faults on one cluster under traffic: --duration 10m [--out DIR] [--submit]", cmdSoak)
}

func cmdList(_ *cli.Operator, _ []string) error {
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SCENARIO\tTOPOLOGY\tINJECTION\tINVARIANT")
	for _, s := range Scenarios() {
		fmt.Fprintf(w, "%s\t%dcp/%dh/%de\t%s\t%s\n", s.ID, max1(s.CPs), s.Hosts, s.Edges, s.Injection, s.Invariant)
	}
	return w.Flush()
}

func binDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// submitter returns a function that sends signed reports to the operator's
// current cluster, if one is configured.
func submitter(submit bool) func(env *envelope.Envelope) string {
	if !submit {
		return func(*envelope.Envelope) string { return "" }
	}
	home := os.Getenv("DH_HOME")
	if home == "" {
		h, _ := os.UserHomeDir()
		home = filepath.Join(h, ".dh")
	}
	op, err := cli.Load(home, os.Getenv("DH_CLUSTER"))
	if err != nil {
		return func(*envelope.Envelope) string { return "not submitted: " + err.Error() }
	}
	return func(env *envelope.Envelope) string {
		var r map[string]any
		if err := op.Do("POST", "/api/v1/chaos/report", env, &r); err != nil {
			return "submission failed: " + err.Error()
		}
		return "submitted to cluster " + op.Cfg.Cluster
	}
}

func cmdRun(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("chaos run", flag.ContinueOnError)
	which := fs.String("scenario", "all", "comma-separated scenario ids, or all")
	out := fs.String("out", "./chaos-reports", "report directory")
	submit := fs.Bool("submit", false, "submit signed reports to the current cluster (DH_HOME)")
	keep := fs.Bool("keep", false, "keep scenario cluster directories")
	base := fs.Int("base", 30000, "base port for scenario clusters")
	verbose := fs.Bool("v", true, "print timelines")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var list []Scenario
	if *which == "all" {
		list = Scenarios()
	} else {
		for _, id := range strings.Split(*which, ",") {
			s, ok := Find(strings.TrimSpace(id))
			if !ok {
				return fmt.Errorf("unknown scenario %q (see dh chaos list)", id)
			}
			list = append(list, s)
		}
	}
	r, err := NewRunner(RunOptions{Dir: *out, Bin: binDir(), Base: *base, Verbose: *verbose, Keep: *keep})
	if err != nil {
		return err
	}
	send := submitter(*submit)
	counts := map[string]int{}
	for _, s := range list {
		fmt.Printf("▶ %s — %s\n", s.ID, s.Title)
		rep, env := r.Run(s)
		counts[rep.Verdict]++
		note := send(env)
		fmt.Printf("  %s in %s: %s", rep.Verdict, (time.Duration(rep.DurationMs) * time.Millisecond).Round(time.Second), rep.Observed)
		if note != "" {
			fmt.Printf(" [%s]", note)
		}
		fmt.Println()
	}
	fmt.Printf("\n%d PASS, %d FAIL, %d SKIP — signed reports in %s (runner key %s)\n", counts[Pass], counts[Fail], counts[Skip], *out, r.ID().ID)
	if counts[Fail] > 0 {
		return fmt.Errorf("%d scenario(s) failed", counts[Fail])
	}
	return nil
}

// Soak runs randomized faults against one cluster under continuous traffic.
func Soak(r *Runner, d time.Duration, seed int64) (*Report, *envelope.Envelope) {
	rng := rand.New(rand.NewSource(seed))
	s := Scenario{ID: "soak", Title: fmt.Sprintf("Soak: %s of randomized faults", d),
		Injection: "random sequence of: host agent kill+restart, leader kill+restart, host network isolation, packet faults, workload kill, clock skew",
		Invariant: "after every fault the cluster returns to the desired replica count within 90s, the audit chain verifies throughout, and request availability through the edge stays ≥ 99.9%",
		CPs:       3, Hosts: 4, Edges: 1, Proxy: true, LostAfter: "10s",
		Run: func(x *Ctx) error {
			if err := x.WaitRunning("web", 3, 90*time.Second); err != nil {
				return err
			}
			if err := x.WaitRouting("web.chaos.test", 3, 30*time.Second); err != nil {
				return err
			}
			tr := x.StartTraffic("web.chaos.test", 50*time.Millisecond)
			end := time.Now().Add(d)
			faults := 0
			var worst time.Duration
			hosts := []string{"host-a", "host-b", "host-c", "host-d"}
			for time.Now().Before(end) {
				faults++
				kind := rng.Intn(6)
				h := hosts[rng.Intn(len(hosts))]
				switch kind {
				case 0:
					x.Step("fault %d: kill -9 dh-noded on %s, restart after 5s", faults, h)
					_ = x.C.Kill(h, syscall.SIGKILL)
					time.Sleep(5 * time.Second)
					_ = x.C.Restart(h)
				case 1:
					l, err := x.C.Leader()
					if err == nil {
						x.Step("fault %d: kill -9 leader %s, restart after 5s", faults, l)
						_ = x.C.Kill(l, syscall.SIGKILL)
						time.Sleep(5 * time.Second)
						_ = x.C.Restart(l)
					}
				case 2:
					x.Step("fault %d: isolate %s for 20s", faults, h)
					x.Ctl.Set(Faults{Isolated: map[string]bool{h: true}})
					_, _ = x.C.Chaos(h, node.ChaosRequest{Fault: "drop-cp", On: true})
					time.Sleep(20 * time.Second)
					x.Ctl.Set(Faults{})
					_, _ = x.C.Chaos(h, node.ChaosRequest{Fault: "drop-cp", On: false})
				case 3:
					x.Step("fault %d: packet faults (3%% loss, 5%% dup, 5%% reorder, +25ms) for 20s", faults)
					x.Ctl.Set(Faults{DropPct: 3, DupPct: 5, ReorderPct: 5, Delay: 25 * time.Millisecond})
					time.Sleep(20 * time.Second)
					x.Ctl.Set(Faults{})
				case 4:
					st, err := x.C.HostStatus(h)
					if err == nil {
						for id, ad := range st.Admitted {
							if ad.Inst.PID > 0 && !ad.Stopped {
								x.Step("fault %d: SIGKILL workload %s on %s", faults, id, h)
								_ = syscall.Kill(-int(ad.Inst.PID), syscall.SIGKILL)
								break
							}
						}
					}
				case 5:
					x.Step("fault %d: clock skew +90s on %s for 15s", faults, h)
					_, _ = x.C.Chaos(h, node.ChaosRequest{Fault: "clock-offset", Value: 90_000})
					time.Sleep(15 * time.Second)
					_, _ = x.C.Chaos(h, node.ChaosRequest{Fault: "clock-offset", Value: 0})
				}
				t0 := time.Now()
				if err := x.C.WaitFor(90*time.Second, "recovery to 3 observed replicas with a verified audit chain", func(v *control.View) bool {
					return App(v, "web").Observed >= 3 && v.Audit.Verification.OK
				}); err != nil {
					tr.Stop()
					return fmt.Errorf("fault %d did not recover: %w", faults, err)
				}
				rec := time.Since(t0)
				if rec > worst {
					worst = rec
				}
				x.Step("recovered in %s", rec.Round(time.Millisecond))
				time.Sleep(time.Duration(5+rng.Intn(10)) * time.Second)
			}
			ok, fail := tr.Stop()
			avail := 100 * float64(ok) / float64(max64(ok+fail, 1))
			x.Measure("faults", "%d", faults)
			x.Measure("worst_recovery_ms", "%d", worst.Milliseconds())
			x.Measure("requests_ok", "%d", ok)
			x.Measure("requests_failed", "%d", fail)
			x.Measure("availability_pct", "%.3f", avail)
			x.Measure("seed", "%d", seed)
			x.Observed("%d faults over %s; every fault recovered (worst %s); %d/%d requests OK (%.3f%%)", faults, d, worst.Round(time.Millisecond), ok, ok+fail, avail)
			if avail < 99.9 {
				return fmt.Errorf("availability %.3f%% below 99.9%%", avail)
			}
			return nil
		}}
	return r.Run(s)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func cmdSoak(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("chaos soak", flag.ContinueOnError)
	d := fs.Duration("duration", 10*time.Minute, "how long to inject faults")
	out := fs.String("out", "./chaos-reports", "report directory")
	submit := fs.Bool("submit", false, "submit the signed report to the current cluster (DH_HOME)")
	seed := fs.Int64("seed", time.Now().UnixNano(), "random seed (reproducible fault sequence)")
	base := fs.Int("base", 30000, "base port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := NewRunner(RunOptions{Dir: *out, Bin: binDir(), Base: *base, Verbose: true})
	if err != nil {
		return err
	}
	fmt.Printf("▶ soak for %s (seed %d)\n", *d, *seed)
	rep, env := Soak(r, *d, *seed)
	note := submitter(*submit)(env)
	fmt.Printf("%s: %s %s\n", rep.Verdict, rep.Observed, note)
	if rep.Verdict != Pass {
		return errors.New("soak failed")
	}
	return nil
}
