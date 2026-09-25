package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/peer"
)

func init() {
	reg("apply", "apply a dh/v1 manifest: -f FILE", true, cmdApply)
	reg("get apps", "list applications with desired / admitted / observed", true, cmdGetApps)
	reg("describe app", "show every replica: desired, admitted, observed, admission checks: APP", true, cmdDescribeApp)
	reg("scale app", "change replica count: APP --replicas N", true, cmdScale)
	reg("delete app", "remove an application (volumes are retained): APP", true, cmdDelete)
	reg("rollout status app", "wait until every replica runs the current generation: APP [--timeout 2m]", true, cmdRollout)
	reg("logs app", "tail a replica's output over the mesh: APP [--replica 0] [--tail 100]", true, cmdLogs)
	reg("exec app", "run a command in a replica (needs host policy allowExec): APP [--replica 0] -- ARGV...", true, cmdExec)
	reg("explain app", "show the scheduler plan for an app: APP", true, cmdExplain)
}

func cmdApply(op *Operator, args []string) error {
	fs := flags("apply")
	file := fs.String("f", "", "manifest file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	src, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	if _, err := manifest.Parse(src); err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/apply", src, &r); err != nil {
		return err
	}
	return printResult(r)
}

func (op *Operator) view() (*control.View, error) {
	var v control.View
	return &v, op.Do("GET", "/api/v1/view", nil, &v)
}

func cmdGetApps(op *Operator, _ []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	w := table("APP", "GEN", "IMAGE", "RUNTIME", "REPLICAS", "DESIRED", "ADMITTED", "OBSERVED", "DRIFT", "STATE")
	for _, a := range v.Apps {
		state := "active"
		if a.Deleted {
			state = "deleting"
		}
		if a.Federation != nil {
			state = "federated from " + a.Federation.Peer
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\n", a.Name, a.Generation, a.Image, a.Runtime, a.Replicas, a.Desired, a.Admitted, a.Observed, a.Drift, state)
	}
	return w.Flush()
}

func findApp(v *control.View, name string) (*control.AppView, error) {
	for i := range v.Apps {
		if v.Apps[i].Name == name {
			return &v.Apps[i], nil
		}
	}
	return nil, fmt.Errorf("no app %q", name)
}

func cmdDescribeApp(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh describe app APP")
	}
	v, err := op.view()
	if err != nil {
		return err
	}
	a, err := findApp(v, args[0])
	if err != nil {
		return err
	}
	fmt.Printf("App %s  generation %d  %s  image %s (%s)\n", a.Name, a.Generation, short(a.Hash), a.Image, a.Runtime)
	fmt.Printf("DESIRED %d   ADMITTED %d   OBSERVED %d   DRIFT %d\n\n", a.Desired, a.Admitted, a.Observed, a.Drift)
	for _, r := range a.Rows {
		fmt.Printf("Replica %d on %s (%s)\n", r.Replica, r.NodeName, short(r.Node))
		fmt.Printf("  Desired:   %s generation %d\n", r.Desired, r.DesiredGen)
		fmt.Printf("  Admitted:  %s generation %d  [%s] %s\n", r.Admitted, r.AdmittedGen, r.Code, r.Reason)
		fmt.Printf("  Observed:  %s generation %d  pid %d container %s mesh port %d restarts %d\n", r.Observed, r.ObservedGen, r.PID, short(r.ContainerID), r.MeshPort, r.Restarts)
		if r.Health != nil {
			fmt.Printf("  Health:    ok=%v %s (consecutive failures %d)\n", r.Health.OK, r.Health.Detail, r.Health.Consecutive)
		}
		fmt.Printf("  Status:    %s   evidence %s %s (%s)\n", r.Status, r.Freshness, short(r.Evidence), ago(r.ObservedAt))
		for _, c := range r.Checks {
			mark := "✓"
			if !c.OK {
				mark = "✗"
			}
			fmt.Printf("    %s %-20s %s\n", mark, c.Name, c.Detail)
		}
		fmt.Println()
	}
	return nil
}

func cmdScale(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh scale app APP --replicas N")
	}
	fs := flags("scale")
	n := fs.Int64("replicas", -1, "replica count")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/apps/"+args[0]+"/scale", map[string]int64{"replicas": *n}, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdDelete(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh delete app APP")
	}
	var r Result
	if err := op.Do("POST", "/api/v1/apps/"+args[0]+"/delete", nil, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdRollout(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh rollout status app APP")
	}
	fs := flags("rollout")
	timeout := fs.Duration("timeout", 2*time.Minute, "give up after")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	deadline := time.Now().Add(*timeout)
	last := ""
	for {
		v, err := op.view()
		if err != nil {
			return err
		}
		a, err := findApp(v, args[0])
		if err != nil {
			return err
		}
		ready := 0
		for _, r := range a.Rows {
			if r.Desired == "RUNNING" && r.Observed == "RUNNING" && r.ObservedGen == a.Generation && r.Freshness == "FRESH" && (r.Health == nil || r.Health.OK) {
				ready++
			}
		}
		line := fmt.Sprintf("generation %d: %d/%d replicas observed running (admitted %d)", a.Generation, ready, a.Replicas, a.Admitted)
		if line != last {
			fmt.Println(line)
			last = line
		}
		if int64(ready) >= a.Replicas {
			fmt.Printf("rollout of %s generation %d complete\n", a.Name, a.Generation)
			return nil
		}
		if time.Now().After(deadline) {
			for _, r := range a.Rows {
				fmt.Printf("  r%d on %s: %s [%s] %s\n", r.Replica, r.NodeName, r.Status, r.Code, r.Reason)
			}
			return errors.New("rollout did not complete in time")
		}
		time.Sleep(time.Second)
	}
}

func replicaFlags(name string, args []string) (string, int64, *int, []string, error) {
	if len(args) < 1 {
		return "", 0, nil, nil, fmt.Errorf("usage: dh %s APP [--replica N]", name)
	}
	fs := flags(name)
	rep := fs.Int64("replica", 0, "replica number")
	tail := fs.Int("tail", 100, "lines")
	if err := fs.Parse(args[1:]); err != nil {
		return "", 0, nil, nil, err
	}
	return args[0], *rep, tail, fs.Args(), nil
}

func (op *Operator) replicaNode(app string, replica int64) (string, string, error) {
	v, err := op.view()
	if err != nil {
		return "", "", err
	}
	a, err := findApp(v, app)
	if err != nil {
		return "", "", err
	}
	for _, r := range a.Rows {
		if r.Replica == replica && r.Desired == "RUNNING" {
			return r.Node, r.Assignment, nil
		}
	}
	return "", "", fmt.Errorf("replica %d of %s is not desired running", replica, app)
}

func cmdLogs(op *Operator, args []string) error {
	app, rep, tail, _, err := replicaFlags("logs app", args)
	if err != nil {
		return err
	}
	node, id, err := op.replicaNode(app, rep)
	if err != nil {
		return err
	}
	var b []byte
	if err := op.Do("GET", fmt.Sprintf("/api/v1/nodes/%s/logs?assignment=%s&tail=%d", node, id, *tail), nil, &b); err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}

func cmdExec(op *Operator, args []string) error {
	app, rep, _, argv, err := replicaFlags("exec app", args)
	if err != nil {
		return err
	}
	if len(argv) == 0 {
		return errors.New("usage: dh exec app APP [--replica N] -- CMD ARGS...")
	}
	node, id, err := op.replicaNode(app, rep)
	if err != nil {
		return err
	}
	var out peer.ExecReply
	if err := op.Do("POST", "/api/v1/nodes/"+node+"/exec", peer.ExecRequest{Assignment: id, Argv: argv}, &out); err != nil {
		return err
	}
	if out.Refused != "" {
		return errors.New("host refused exec: " + out.Refused)
	}
	fmt.Print(out.Output)
	if out.ExitCode != 0 {
		return fmt.Errorf("exit code %d", out.ExitCode)
	}
	return nil
}

func cmdExplain(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh explain app APP")
	}
	var plans map[string]json.RawMessage
	if err := op.Do("GET", "/api/v1/plans", nil, &plans); err != nil {
		return err
	}
	raw, ok := plans[args[0]]
	if !ok {
		return fmt.Errorf("no plan for %s on the serving member (plans are computed by the leader)", args[0])
	}
	var p struct {
		Plan struct {
			Replicas []struct {
				Replica int64  `json:"replica"`
				Node    string `json:"node"`
				Kept    bool   `json:"kept"`
				Reason  string `json:"reason"`
				Rows    []struct {
					Name   string  `json:"name"`
					OK     bool    `json:"ok"`
					Stage  string  `json:"stage"`
					Reason string  `json:"reason"`
					Score  float64 `json:"score"`
				} `json:"rows"`
			} `json:"replicas"`
		} `json:"plan"`
		Note string `json:"note"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	for _, r := range p.Plan.Replicas {
		target := r.Node
		if target == "" {
			target = "UNSCHEDULABLE"
		}
		fmt.Printf("replica %d → %s (%s)\n", r.Replica, short(target), r.Reason)
		for _, row := range r.Rows {
			if row.OK {
				fmt.Printf("    %-12s pass   score %.4f\n", row.Name, row.Score)
			} else {
				fmt.Printf("    %-12s %-10s %s\n", row.Name, strings.ToUpper(row.Stage), row.Reason)
			}
		}
	}
	if p.Note != "" {
		fmt.Println(p.Note)
	}
	return nil
}
