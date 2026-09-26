package cli

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/node"
)

func init() {
	reg("node invite", "create a single-use join token: [--roles edge] [--auto] [--ttl 15m] [--out FILE]", true, cmdInvite)
	reg("node invite-revoke", "withdraw an unused join token: TOKEN|TOKEN-FILE|NONCE", true, cmdInviteRevoke)
	reg("node join", "print how to join a host with a token (runs on the host): TOKEN", false, cmdJoinHelp)
	reg("node approve", "approve a pending host: HOST", true, nodeAction("approve"))
	reg("node revoke", "revoke a host (blocks new work; admitted work may continue): HOST [--reason R]", true, cmdRevoke)
	reg("node drain", "drain a host (replicas are rescheduled): HOST", true, nodeAction("drain"))
	reg("node undrain", "return a drained host to service: HOST", true, nodeAction("undrain"))
	reg("node revoke-key", "emergency-revoke one host key: HOST --pub KEY [--reason R]", true, cmdRevokeKey)
	reg("get nodes", "list hosts", true, cmdGetNodes)
	reg("describe node", "show a host: identity, policy, observation, mesh, storage", true, cmdDescribeNode)
}

func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func cmdInvite(op *Operator, args []string) error {
	fs := flags("node invite")
	roles := fs.String("roles", "", "roles granted to the host (edge)")
	auto := fs.Bool("auto", false, "approve automatically when the host enrolls")
	ttl := fs.Duration("ttl", 15*time.Minute, "token lifetime (keep it short: the token admits whoever presents it first)")
	note := fs.String("note", "", "note recorded with the invite")
	out := fs.String("out", "", "write the token to a file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	nonce := randomNonce()
	exp := time.Now().Add(*ttl).UnixMilli()
	tok, err := capability.Mint(op.Root, capability.Caveats{Actions: []string{"node.join"}, Resources: []string{"cluster/" + op.Cfg.Cluster}, Expires: exp, Nonce: nonce}, "", "join")
	if err != nil {
		return err
	}
	var rl []string
	for _, r := range strings.Split(*roles, ",") {
		if r = strings.TrimSpace(r); r != "" {
			rl = append(rl, r)
		}
	}
	if rl == nil {
		rl = []string{}
	}
	var r Result
	if err := op.Do("POST", "/api/v1/invites", map[string]any{"nonce": nonce, "expires": exp, "roles": rl, "note": *note, "auto": *auto}, &r); err != nil {
		return err
	}
	join := node.EncodeJoinToken(node.JoinToken{Cluster: op.Cfg.Cluster, Root: op.Root.PubString(), RootCACert: string(op.CAPEM), Endpoints: op.Cfg.Endpoints, TLS: op.Cfg.TLS, Capability: tok.Encode()})
	if *out != "" {
		if err := os.WriteFile(*out, []byte(join+"\n"), 0o600); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "join token written to %s (single use, expires %s)\n", *out, time.UnixMilli(exp).Format(time.RFC3339))
		return nil
	}
	fmt.Println(join)
	fmt.Fprintf(os.Stderr, "single-use join token; expires %s. On the host: dh-noded --data DIR --name NAME --join-file TOKENFILE\n", time.UnixMilli(exp).Format(time.RFC3339))
	return nil
}

func cmdInviteRevoke(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh node invite-revoke TOKEN|NONCE")
	}
	nonce, err := inviteNonce(args[0])
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/invites/"+nonce+"/revoke", nil, &r); err != nil {
		return err
	}
	fmt.Println(r.Message)
	return nil
}

// inviteNonce accepts a join token (dhjoin1.…), a file holding one (as
// written by dh node invite --out), or the bare nonce.
func inviteNonce(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "dhjoin1.") {
		if b, err := os.ReadFile(s); err == nil {
			s = strings.TrimSpace(string(b))
		}
	}
	if !strings.HasPrefix(s, "dhjoin1.") {
		if len(s) < 16 || strings.Trim(s, "0123456789abcdef") != "" {
			return "", errors.New("not a join token, token file or invite nonce")
		}
		return s, nil
	}
	t, err := node.DecodeJoinToken(s)
	if err != nil {
		return "", err
	}
	c, err := capability.Decode(t.Capability)
	if err != nil {
		return "", err
	}
	var last capability.Block
	if len(c.Blocks) == 0 || c.Blocks[len(c.Blocks)-1].Decode(&last) != nil || last.Caveats.Nonce == "" {
		return "", errors.New("join token carries no invite nonce")
	}
	return last.Caveats.Nonce, nil
}

func cmdJoinHelp(_ *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh node join TOKEN")
	}
	t, err := node.DecodeJoinToken(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("token for cluster %s, root %s, control plane %v\nrun on the host:\n  dh-noded --data /var/lib/dh --name $(hostname) --join %s\n", t.Cluster, t.Root, t.Endpoints, args[0])
	return nil
}

// resolveNode maps a name or id prefix to a host id.
func (op *Operator) resolveNode(q string) (string, error) {
	var v control.View
	if err := op.Do("GET", "/api/v1/view", nil, &v); err != nil {
		return "", err
	}
	var hits []string
	for _, n := range v.Nodes {
		if n.ID == q || n.Name == q {
			return n.ID, nil
		}
		if strings.HasPrefix(n.ID, q) {
			hits = append(hits, n.ID)
		}
	}
	if len(hits) == 1 {
		return hits[0], nil
	}
	return "", fmt.Errorf("no unique host matches %q", q)
}

func nodeAction(action string) func(op *Operator, args []string) error {
	return func(op *Operator, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("usage: dh node %s HOST", action)
		}
		id, err := op.resolveNode(args[0])
		if err != nil {
			return err
		}
		var r Result
		if err := op.Do("POST", "/api/v1/nodes/"+id+"/"+action, nil, &r); err != nil {
			return err
		}
		return printResult(r)
	}
}

func cmdRevoke(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh node revoke HOST [--reason R]")
	}
	fs := flags("node revoke")
	reason := fs.String("reason", "operator revocation", "reason")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	id, err := op.resolveNode(args[0])
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/nodes/"+id+"/revoke", map[string]string{"reason": *reason}, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdRevokeKey(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh node revoke-key HOST --pub KEY")
	}
	fs := flags("node revoke-key")
	pub := fs.String("pub", "", "key to revoke")
	reason := fs.String("reason", "emergency key revocation", "reason")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	id, err := op.resolveNode(args[0])
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/nodes/"+id+"/revoke-key", map[string]string{"pub": *pub, "reason": *reason}, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdGetNodes(op *Operator, _ []string) error {
	var v control.View
	if err := op.Do("GET", "/api/v1/view", nil, &v); err != nil {
		return err
	}
	w := table("NAME", "ID", "STATUS", "HEALTH", "DOMAIN", "MESH", "ROLES", "WORKLOADS", "LAST OBSERVATION", "MODE")
	for _, n := range v.Nodes {
		roles := strings.Join(n.Roles, ",")
		if roles == "" {
			roles = "-"
		}
		obs := fmt.Sprintf("%s seq %d (%s)", n.LastObs.Freshness, n.LastObs.Seq, ago(n.LastObs.ReceivedAt))
		mode := n.Mode
		if mode == "" {
			mode = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s/%s\t%s\t%s\t%d\t%s\t%s\n", n.Name, n.ID, n.Status, n.Health, n.Region, n.Host, orDash(n.MeshIP), roles, n.Workloads, obs, mode)
	}
	return w.Flush()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func cmdDescribeNode(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh describe node HOST")
	}
	id, err := op.resolveNode(args[0])
	if err != nil {
		return err
	}
	var v control.View
	if err := op.Do("GET", "/api/v1/view", nil, &v); err != nil {
		return err
	}
	for _, n := range v.Nodes {
		if n.ID != id {
			continue
		}
		fmt.Printf("Host %s (%s)\n", n.Name, n.ID)
		fmt.Printf("  Identity: %s   New admission: %s   Existing runtime: %s\n", n.Identity, n.NewAdmission, n.Existing)
		fmt.Printf("  Status %s, health %s, domain %s/%s/%s, %s/%s, tiers %v, roles %v\n", n.Status, n.Health, n.Region, n.Zone, n.Host, n.OS, n.Arch, n.Tiers, n.Roles)
		for _, k := range n.Keys {
			state := "valid"
			if k.Revoked {
				state = "REVOKED: " + k.Reason
			} else if k.Until != 0 {
				state = "valid until " + time.UnixMilli(k.Until).Format(time.RFC3339)
			}
			fmt.Printf("  Key %s  %s\n", k.Pub, state)
		}
		p := n.Policy
		fmt.Printf("  Sovereign policy (host-signed): sovereign=%v maxWorkloads=%d maxCPU=%dm maxMem=%s acceptTiers=%v runtimes=%v requireSignature=%v federated=%v exec=%v offline=%s skew=%dms\n",
			p.Sovereign, p.MaxWorkloads, p.MaxCPUMilli, manifest.FormatBytes(p.MaxMemBytes), p.AcceptTiers, p.AllowRuntimes, p.RequireImageSignature, p.AllowFederated, p.AllowExec, p.OfflineAdmission, p.MaxClockSkewMs)
		fmt.Printf("  Last signed observation: seq %d, %s, %s (source %s, evidence %s)\n", n.LastObs.Seq, n.LastObs.Freshness, ago(n.LastObs.ReceivedAt), n.LastObs.Source, short(n.Evidence))
		if n.Mode != "" {
			fmt.Printf("  Mode: %s %s\n", n.Mode, n.ModeDetail)
		}
		fmt.Printf("  Ledger head: seq %d %s\n", n.Ledger.Seq, short(n.Ledger.Hash))
		if n.Mesh != nil {
			fmt.Printf("  Mesh: %s %s wg %s gossip %s (%d members)\n", n.Mesh.Device, n.Mesh.MeshIP, short(n.Mesh.WGPub), n.Mesh.Gossip, n.Mesh.Members)
			for _, pr := range n.Mesh.Peers {
				lat := "NOT MEASURED"
				if pr.RTTUs >= 0 && pr.RTTAt > 0 {
					lat = fmt.Sprintf("%.2fms", float64(pr.RTTUs)/1000)
				}
				fmt.Printf("    peer %s handshake %s rx %d tx %d gossip %s rtt %s binding-ok %v\n", short(pr.Node), ago(pr.LastHandshake), pr.RxBytes, pr.TxBytes, pr.Gossip, lat, pr.BindingOK)
			}
		}
		if n.Storage != nil {
			fmt.Printf("  Storage: %d chunks, used %s, quota %s, free %s, corrupt %d\n", n.Storage.Chunks, manifest.FormatBytes(n.Storage.UsedBytes), manifest.FormatBytes(n.Storage.QuotaBytes), manifest.FormatBytes(n.Storage.FreeBytes), n.Storage.Corrupt)
		}
		if n.Facts != nil {
			fmt.Printf("  Facts: docker=%q udp443=%v runtimes=%v clockSkew=%dms\n", n.Facts.Docker, n.Facts.UDP443, n.Facts.Runtimes, n.Facts.ClockSkewMs)
		}
		return nil
	}
	return fmt.Errorf("host %s not found", id)
}
