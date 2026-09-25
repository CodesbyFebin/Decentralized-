package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/manifest"
)

func init() {
	reg("federation root", "print this cluster's name and root key (share it with a peer that will grant you capacity)", true, func(op *Operator, _ []string) error {
		fmt.Printf("%s %s\n", op.Cfg.Cluster, op.Root.PubString())
		return nil
	})
	reg("federation grant", "grant a peer cluster capacity here: --to NAME --to-root KEY [--tiers federated] [--max-replicas 3] [--max-cpu 1] [--max-mem 1Gi] [--runtimes process,docker] [--ttl 720h] --out FILE", true, cmdFedGrant)
	reg("federation accept", "accept an agreement a peer granted this cluster: FILE", true, func(op *Operator, args []string) error {
		if len(args) != 1 {
			return errors.New("usage: dh federation accept FILE")
		}
		env, err := readEnvelope(args[0])
		if err != nil {
			return err
		}
		var r Result
		if err := op.Do("POST", "/api/v1/federation/accept", env, &r); err != nil {
			return err
		}
		return printResult(r)
	})
	reg("federation revoke", "revoke an agreement this cluster granted: AGREEMENT-DIGEST [--reason R]", true, cmdFedRevoke)
	reg("federation withdraw", "withdraw this cluster's app from a peer: APP", true, func(op *Operator, args []string) error {
		if len(args) != 1 {
			return errors.New("usage: dh federation withdraw APP")
		}
		var r Result
		if err := op.Do("POST", "/api/v1/federation/withdraw", map[string]string{"app": args[0]}, &r); err != nil {
			return err
		}
		return printResult(r)
	})
	reg("federation ls", "list agreements and placements in both directions", true, cmdFedList)
}

func cmdFedGrant(op *Operator, args []string) error {
	fs := flags("federation grant")
	to := fs.String("to", "", "grantee cluster name")
	toRoot := fs.String("to-root", "", "grantee root key")
	tiers := fs.String("tiers", "federated", "workload tiers the grantee may request")
	replicas := fs.Int64("max-replicas", 3, "maximum replicas")
	cpu := fs.String("max-cpu", "1", "maximum total CPU")
	mem := fs.String("max-mem", "1Gi", "maximum total memory")
	runtimes := fs.String("runtimes", "process,docker", "allowed runtimes")
	ttl := fs.Duration("ttl", 30*24*time.Hour, "agreement lifetime")
	trust := fs.String("trust", "partner", "descriptive trust level")
	out := fs.String("out", "", "write the signed agreement to this file (hand it to the grantee)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *to == "" || *toRoot == "" || *out == "" {
		return errors.New("--to, --to-root and --out are required")
	}
	c, err := manifest.ParseCPU(*cpu)
	if err != nil {
		return err
	}
	m, err := manifest.ParseBytes(*mem)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	a := control.FedAgreement{ID: randomNonce(), Grantor: op.Cfg.Cluster, GrantorRoot: op.Root.PubString(), GrantorAPI: op.Cfg.Endpoints,
		GrantorCA: grantorCA(op), Grantee: *to, GranteeRoot: *toRoot, Actions: []string{"federation.place", "federation.status", "federation.withdraw"},
		Tiers: split(*tiers), Runtimes: split(*runtimes), MaxReplicas: *replicas, MaxCPUMilli: c, MaxMemBytes: m,
		NotBefore: now, Expires: now + ttl.Milliseconds(), Trust: *trust, Issued: now}
	env, err := envelope.Sign(op.Root, "", envelope.KindFedAgreement, a)
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/federation/grant", env, &r); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(env, "", "  ")
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(Out, "agreement %s granted to %s; hand %s to its operator (dh federation accept)\n", env.Digest(), *to, *out)
	return nil
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func cmdFedRevoke(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh federation revoke AGREEMENT-DIGEST [--reason R]")
	}
	fs := flags("federation revoke")
	reason := fs.String("reason", "operator revocation", "reason")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	env, err := envelope.Sign(op.Root, "", envelope.KindFedRevocation, map[string]any{"agreement": args[0], "reason": *reason, "ts": time.Now().UnixMilli()})
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/federation/revoke", env, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdFedList(op *Operator, _ []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	f := v.Federation
	w := table("DIRECTION", "PEER", "AGREEMENT", "LIMITS", "STATE")
	for _, g := range f.Granted {
		st := "active"
		if g.Revoked {
			st = "REVOKED"
		}
		fmt.Fprintf(w, "granted to\t%s\t%s\t≤%d replicas, ≤%dm cpu, tiers %v\t%s\n", g.A.Grantee, short(g.Digest), g.A.MaxReplicas, g.A.MaxCPUMilli, g.A.Tiers, st)
	}
	for _, h := range f.Held {
		fmt.Fprintf(w, "held from\t%s\t%s\t≤%d replicas, ≤%dm cpu, tiers %v\tactive\n", h.A.Grantor, short(h.Digest), h.A.MaxReplicas, h.A.MaxCPUMilli, h.A.Tiers)
	}
	w.Flush()
	if len(f.Inbound)+len(f.Outbound) > 0 {
		w = table("PLACEMENT", "APP", "PEER", "STATE")
		for _, in := range f.Inbound {
			st := "running here"
			if in.Withdrawn {
				st = "withdrawn"
			}
			fmt.Fprintf(w, "inbound\t%s (their %s)\t%s\t%s\n", in.App, in.RemoteApp, in.Peer, st)
		}
		for _, o := range f.Outbound {
			fmt.Fprintf(w, "outbound\t%s\t%s\taccepted=%v %s\n", o.App, o.Peer, o.Accepted, o.Message)
		}
		w.Flush()
	}
	return nil
}

// grantorCA is the root CA a peer needs to reach this cluster's API over TLS.
func grantorCA(op *Operator) string {
	if !op.Cfg.TLS {
		return ""
	}
	return string(op.CAPEM)
}
