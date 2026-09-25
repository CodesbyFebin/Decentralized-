package cli

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/storage"
)

func init() {
	reg("volume ls", "list volumes with replica evidence", true, cmdVolumes)
	reg("volume export", "download a committed snapshot and verify it: VOLUME --out DIR [--snapshot ID]", true, cmdVolumeExport)
	reg("get volumes", "alias of volume ls", true, cmdVolumes)
	reg("artifact push", "upload an executable into the cluster CAS: FILE --name NAME [--sign]", true, cmdArtifactPush)
	reg("artifact sign", "attest an artifact digest with the root key: DIGEST --name NAME", true, cmdArtifactSign)
	reg("artifact ls", "list artifacts", true, cmdArtifacts)
	reg("mesh status", "show every host's WireGuard device and gossip view", true, cmdMeshStatus)
	reg("mesh peers", "show peer handshakes, traffic and measured RTT: [HOST]", true, cmdMeshPeers)
	reg("mesh ping", "measure RTT from the control plane to a host over the mesh: HOST", true, cmdMeshPing)
	reg("mesh doctor", "check bindings, handshakes and gossip for inconsistencies", true, cmdMeshDoctor)
	reg("audit tail", "show recent audit entries: [-n 30]", true, cmdAuditTail)
	reg("audit verify", "fetch the full ledger and verify chain + checkpoints locally", true, cmdAuditVerify)
	reg("audit host", "fetch and verify a host's own ledger over the mesh: HOST", true, cmdAuditHost)
	reg("freeze", "freeze the control plane: hosts hold admitted work, refuse new work", true, freeze(true))
	reg("unfreeze", "resume signing new work", true, freeze(false))
	reg("export", "write a signed, secret-free export of the installation: --out FILE", true, cmdExport)
	reg("import", "verify an export and re-apply its desired state here: FILE", true, cmdImport)
	reg("verify export", "verify an export file offline: FILE", false, func(_ *Operator, args []string) error {
		if len(args) != 1 {
			return errors.New("usage: dh verify export FILE")
		}
		env, err := readEnvelope(args[0])
		if err != nil {
			return err
		}
		ex, err := control.VerifyExport(env)
		if err != nil {
			return err
		}
		fmt.Printf("export OK: cluster %s, %d manifests, %d hosts, %d volumes, %d audit entries (chain and %d checkpoints verified), signed by %s\n",
			ex.Cluster, len(ex.Manifests), len(ex.Hosts), len(ex.Volumes), len(ex.Audit), len(ex.Checkpoints), env.Signer)
		return nil
	})
	reg("console", "print a console URL with a session capability: [--read-only] [--ttl 12h]", true, cmdConsole)
	reg("token", "print an operator bearer capability: [--read-only] [--ttl 1h]", true, cmdToken)
	reg("publishers set", "set artifact publisher keys (root is always included): KEY...", true, func(op *Operator, args []string) error {
		var r Result
		if err := op.Do("POST", "/api/v1/publishers", map[string]any{"keys": args}, &r); err != nil {
			return err
		}
		return printResult(r)
	})
}

func cmdVolumes(op *Operator, _ []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	w := table("VOLUME", "STATE", "PRIMARY", "MEMBERS", "COMMITTED", "SNAPSHOTS", "BYTES", "DETAIL")
	for _, vol := range v.Volumes {
		var bytes int64
		if vol.CommittedRef != nil {
			bytes = vol.CommittedRef.Bytes
		}
		primary := vol.Primary
		for _, n := range v.Nodes {
			if n.ID == vol.Primary {
				primary = n.Name
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", vol.ID, vol.State, primary, strings.Join(vol.MemberNames, ","), short(vol.Committed), len(vol.Snapshots), manifest.FormatBytes(bytes), vol.Detail)
	}
	return w.Flush()
}

func cmdVolumeExport(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh volume export VOLUME --out DIR")
	}
	fs := flags("volume export")
	out := fs.String("out", "", "directory to write files into")
	snap := fs.String("snapshot", "", "snapshot id (default: committed)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("--out is required")
	}
	var raw []byte
	if err := op.Do("GET", "/api/v1/volumes/export?volume="+url.QueryEscape(args[0])+"&snapshot="+url.QueryEscape(*snap), nil, &raw); err != nil {
		return err
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	var m api.SnapshotManifest
	files := map[string][]byte{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("archive truncated or corrupt: %w", err)
		}
		b, _ := io.ReadAll(tr)
		if h.Name == ".dh-snapshot.json" {
			if err := json.Unmarshal(b, &m); err != nil {
				return err
			}
			continue
		}
		files[h.Name] = b
	}
	// Re-verify every file against the manifest's chunk list locally.
	for _, f := range m.Files {
		b := files[f.Path]
		var got []string
		for _, c := range storage.ChunkAll(b) {
			got = append(got, c.ID)
		}
		if strings.Join(got, ",") != strings.Join(f.Chunks, ",") {
			return fmt.Errorf("%s does not match its manifest chunks", f.Path)
		}
		dst := filepath.Join(*out, filepath.FromSlash(f.Path))
		if !strings.HasPrefix(filepath.Clean(dst), filepath.Clean(*out)) {
			return fmt.Errorf("unsafe path %s", f.Path)
		}
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		if err := os.WriteFile(dst, b, os.FileMode(f.Mode&0o777)|0o600); err != nil {
			return err
		}
	}
	fmt.Printf("exported %d file(s) from %s into %s; every file re-chunked and matched its manifest\n", len(m.Files), args[0], *out)
	return nil
}

func cmdArtifactPush(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh artifact push FILE --name NAME [--sign]")
	}
	fs := flags("artifact push")
	name := fs.String("name", filepath.Base(args[0]), "artifact name")
	sign := fs.Bool("sign", false, "also attest the digest with the root key")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/artifacts?name="+url.QueryEscape(*name), data, &r); err != nil {
		return err
	}
	d, _ := r.Data.(map[string]any)
	digest := fmt.Sprint(d["digest"])
	if want := storage.ChunkID(data); want != digest {
		return fmt.Errorf("control plane reported digest %s, local file is %s", digest, want)
	}
	fmt.Printf("%s@%s (%v bytes, %v chunks)\n", *name, digest, d["bytes"], d["chunks"])
	if *sign {
		return attest(op, digest, *name)
	}
	return nil
}

func attest(op *Operator, digest, name string) error {
	env, err := envelope.Sign(op.Root, "", envelope.KindArtifact, api.ArtifactAttestation{Digest: digest, Name: name, TS: time.Now().UnixMilli()})
	if err != nil {
		return err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/attest", env, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdArtifactSign(op *Operator, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: dh artifact sign DIGEST --name NAME")
	}
	fs := flags("artifact sign")
	name := fs.String("name", "", "artifact name")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	return attest(op, args[0], *name)
}

func cmdArtifacts(op *Operator, _ []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	w := table("NAME", "DIGEST", "BYTES", "CHUNKS", "HOLDERS", "ATTESTED")
	for _, a := range v.Artifacts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%v\n", a.Name, a.Digest, manifest.FormatBytes(a.Bytes), a.Chunks, strings.Join(a.HolderNames, ","), a.Attested)
	}
	return w.Flush()
}

func cmdMeshStatus(op *Operator, _ []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	w := table("HOST", "MESH IP", "DEVICE", "WG KEY", "GOSSIP", "MEMBERS", "PEERS", "HANDSHAKES<3m")
	for _, n := range v.Nodes {
		if n.Mesh == nil {
			fmt.Fprintf(w, "%s\t%s\tUNKNOWN\t-\t-\t-\t-\t-\n", n.Name, orDash(n.MeshIP))
			continue
		}
		hs := 0
		for _, p := range n.Mesh.Peers {
			if p.LastHandshake > 0 && time.Since(time.UnixMilli(p.LastHandshake)) < 3*time.Minute {
				hs++
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\n", n.Name, orDash(n.Mesh.MeshIP), n.Mesh.Device, short(n.Mesh.WGPub), n.Mesh.Gossip, n.Mesh.Members, len(n.Mesh.Peers), hs)
	}
	return w.Flush()
}

func cmdMeshPeers(op *Operator, args []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	names := map[string]string{}
	for _, n := range v.Nodes {
		names[n.ID] = n.Name
	}
	for _, m := range v.Cluster.Members {
		names[m.ID] = "cp:" + short(m.ID)
	}
	w := table("HOST", "PEER", "ENDPOINT", "HANDSHAKE", "RX", "TX", "GOSSIP", "RTT", "BINDING")
	for _, n := range v.Nodes {
		if len(args) == 1 && n.Name != args[0] && n.ID != args[0] {
			continue
		}
		if n.Mesh == nil {
			continue
		}
		for _, p := range n.Mesh.Peers {
			rtt := "NOT MEASURED"
			if p.RTTUs >= 0 && p.RTTAt > 0 {
				rtt = fmt.Sprintf("%.2fms", float64(p.RTTUs)/1000)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%v\n", n.Name, names[p.Node], orDash(p.Endpoint), ago(p.LastHandshake), p.RxBytes, p.TxBytes, p.Gossip, rtt, p.BindingOK)
		}
	}
	return w.Flush()
}

func cmdMeshPing(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh mesh ping HOST")
	}
	id, err := op.resolveNode(args[0])
	if err != nil {
		return err
	}
	var out struct {
		Node      string  `json:"node"`
		MeshIP    string  `json:"meshIp"`
		SamplesUs []int64 `json:"samplesUs"`
		Error     string  `json:"error"`
		From      string  `json:"from"`
		Method    string  `json:"method"`
	}
	if err := op.Do("POST", "/api/v1/mesh/ping", map[string]string{"node": id}, &out); err != nil {
		return err
	}
	fmt.Printf("%s (%s) from control-plane member %s — %s\n", out.Node, out.MeshIP, short(out.From), out.Method)
	if len(out.SamplesUs) == 0 {
		return fmt.Errorf("no reply: %s", out.Error)
	}
	var sum int64
	for i, s := range out.SamplesUs {
		fmt.Printf("  sample %d: %.3f ms\n", i+1, float64(s)/1000)
		sum += s
	}
	sorted := append([]int64(nil), out.SamplesUs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	fmt.Printf("  n=%d min %.3f ms median %.3f ms mean %.3f ms\n", len(sorted), float64(sorted[0])/1000, float64(sorted[len(sorted)/2])/1000, float64(sum)/float64(len(sorted))/1000)
	return nil
}

func cmdMeshDoctor(op *Operator, _ []string) error {
	v, err := op.view()
	if err != nil {
		return err
	}
	problems := 0
	report := func(format string, a ...any) {
		problems++
		fmt.Printf("✗ "+format+"\n", a...)
	}
	for _, n := range v.Nodes {
		if n.Status != "ready" {
			continue
		}
		if n.Mesh == nil || n.Mesh.Device == "none" {
			report("%s: no WireGuard device reported", n.Name)
			continue
		}
		if n.WGBinding == nil {
			report("%s: control plane holds no signed WireGuard binding", n.Name)
		}
		for _, p := range n.Mesh.Peers {
			if !p.BindingOK {
				report("%s: peer %s binding failed verification (not configured)", n.Name, short(p.Node))
			}
			if p.BindingOK && (p.LastHandshake == 0 || time.Since(time.UnixMilli(p.LastHandshake)) > 3*time.Minute) {
				report("%s → %s: no WireGuard handshake in 3 minutes", n.Name, short(p.Node))
			}
			if p.Gossip == "suspect" || p.Gossip == "dead" || p.Gossip == "identity-mismatch" {
				report("%s sees %s as %s in gossip", n.Name, short(p.Node), p.Gossip)
			}
		}
	}
	if problems == 0 {
		fmt.Println("✓ every ready host has a device, a signed binding, recent handshakes with all bound peers, and a clean gossip view")
	}
	return nil
}

func cmdAuditTail(op *Operator, args []string) error {
	fs := flags("audit tail")
	n := fs.Int("n", 30, "entries")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var out struct {
		Entries []audit.Entry `json:"entries"`
		Head    int64         `json:"head"`
	}
	from := int64(0)
	var h struct {
		Head int64 `json:"head"`
	}
	if err := op.Do("GET", "/api/v1/audit?limit=1", nil, &h); err == nil && h.Head > int64(*n) {
		from = h.Head - int64(*n)
	}
	if err := op.Do("GET", fmt.Sprintf("/api/v1/audit?from=%d&limit=%d", from, *n), nil, &out); err != nil {
		return err
	}
	w := table("SEQ", "TIME", "SOURCE", "ACTOR", "ACTION", "RESOURCE", "GEN", "DETAIL")
	for _, e := range out.Entries {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n", e.Seq, time.UnixMilli(e.TS).Format("15:04:05.000"), e.Source, short(e.Actor), e.Action, e.Resource, e.Generation, e.Detail)
	}
	return w.Flush()
}

// cmdAuditVerify verifies the ledger on the operator's machine, trusting
// only the root key: roster from the root, member keys from the roster,
// checkpoints from member keys, entries from the chain.
func cmdAuditVerify(op *Operator, _ []string) error {
	var all []audit.Entry
	var cps, past []*envelope.Envelope
	from := int64(0)
	for {
		var out struct {
			Entries     []audit.Entry        `json:"entries"`
			Checkpoints []*envelope.Envelope `json:"checkpoints"`
			PastRosters []*envelope.Envelope `json:"pastRosters"`
			Head        int64                `json:"head"`
		}
		if err := op.Do("GET", fmt.Sprintf("/api/v1/audit?from=%d&limit=5000", from), nil, &out); err != nil {
			return err
		}
		all = append(all, out.Entries...)
		cps, past = out.Checkpoints, out.PastRosters
		if len(out.Entries) == 0 || out.Entries[len(out.Entries)-1].Seq >= out.Head {
			break
		}
		from = out.Entries[len(out.Entries)-1].Seq
	}
	var st struct {
		Roster       *envelope.Envelope `json:"roster"`
		RootRotation *envelope.Envelope `json:"rootRotation"`
		Root         string             `json:"root"`
	}
	if err := op.Do("GET", "/api/v1/state", nil, &st); err != nil {
		return err
	}
	if st.Root != op.Root.PubString() {
		return fmt.Errorf("cluster root %s is not this operator's root %s", short(st.Root), short(op.Root.PubString()))
	}
	if st.Roster == nil || st.Roster.VerifyKey(envelope.KindRoster, st.Root) != nil {
		return errors.New("roster is not signed by the cluster root")
	}
	keys := control.HistoricalMemberKeys(st.Root, st.RootRotation, st.Roster, past)
	if b := audit.Verify(all, 0, audit.Genesis); b != nil {
		return fmt.Errorf("HASH CHAIN FAILURE: %s at seq %d (expected %s, actual %s)", b.Reason, b.Seq, b.Expected, b.Actual)
	}
	if b := audit.VerifyCheckpoints(all, cps, func(s string) []string { return keys[s] }); b != nil {
		return fmt.Errorf("CHECKPOINT FAILURE: %s at seq %d (expected %s, actual %s)", b.Reason, b.Seq, b.Expected, b.Actual)
	}
	head := "genesis"
	if len(all) > 0 {
		head = all[len(all)-1].Hash
	}
	fmt.Printf("audit verified locally: %d entries, head %s, %d signed checkpoint(s) by roster members\n", len(all), short(head), len(cps))
	return nil
}

func cmdAuditHost(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh audit host HOST")
	}
	id, err := op.resolveNode(args[0])
	if err != nil {
		return err
	}
	var rep struct {
		Entries    []audit.Entry      `json:"entries"`
		Checkpoint *envelope.Envelope `json:"checkpoint"`
		Corrupt    *audit.Break       `json:"corrupt"`
	}
	if err := op.Do("GET", "/api/v1/nodes/"+id+"/ledger?limit=5000", nil, &rep); err != nil {
		return err
	}
	if rep.Corrupt != nil {
		return fmt.Errorf("host reports its own ledger corrupt: %s at seq %d", rep.Corrupt.Reason, rep.Corrupt.Seq)
	}
	if b := audit.Verify(rep.Entries, 0, audit.Genesis); b != nil {
		return fmt.Errorf("HOST LEDGER HASH CHAIN FAILURE: %s at seq %d", b.Reason, b.Seq)
	}
	v, err := op.view()
	if err != nil {
		return err
	}
	var keys []string
	for _, n := range v.Nodes {
		if n.ID == id {
			for _, k := range n.Keys {
				if !k.Revoked {
					keys = append(keys, k.Pub)
				}
			}
		}
	}
	if b := audit.VerifyCheckpoints(rep.Entries, []*envelope.Envelope{rep.Checkpoint}, func(string) []string { return keys }); b != nil {
		return fmt.Errorf("host checkpoint: %s", b.Reason)
	}
	for _, e := range rep.Entries[max(0, len(rep.Entries)-15):] {
		fmt.Printf("%5d %s %-20s %-22s %s\n", e.Seq, time.UnixMilli(e.TS).Format("15:04:05"), e.Action, e.Resource, e.Detail)
	}
	fmt.Printf("host ledger verified: %d entries, head signed by the host's own key\n", len(rep.Entries))
	return nil
}

func freeze(on bool) func(op *Operator, args []string) error {
	return func(op *Operator, _ []string) error {
		var r Result
		if err := op.Do("POST", "/api/v1/freeze", map[string]bool{"frozen": on}, &r); err != nil {
			return err
		}
		return printResult(r)
	}
}

func cmdExport(op *Operator, args []string) error {
	fs := flags("export")
	out := fs.String("out", "dh-export.json", "output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var raw []byte
	if err := op.Do("GET", "/api/v1/export", nil, &raw); err != nil {
		return err
	}
	var env envelope.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	ex, err := control.VerifyExport(&env)
	if err != nil {
		return fmt.Errorf("export did not verify: %w", err)
	}
	if err := os.WriteFile(*out, raw, 0o600); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d manifests, %d hosts, %d volumes, %d artifacts, %d audit entries; excluded: %s\n", *out, len(ex.Manifests), len(ex.Hosts), len(ex.Volumes), len(ex.Artifacts), len(ex.Audit), strings.Join(ex.Excluded, "; "))
	return nil
}

func cmdImport(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh import FILE")
	}
	env, err := readEnvelope(args[0])
	if err != nil {
		return err
	}
	ex, err := control.VerifyExport(env)
	if err != nil {
		return fmt.Errorf("refusing to import: %w", err)
	}
	for _, m := range ex.Manifests {
		b, _ := json.Marshal(m)
		var r Result
		if err := op.Do("POST", "/api/v1/apply", b, &r); err != nil {
			return fmt.Errorf("apply %s: %w", m.Metadata.Name, err)
		}
		fmt.Printf("%s: %s\n", m.Metadata.Name, r.Message)
	}
	fmt.Printf("imported desired state from cluster %s (export verified: %d audit entries). Hosts must join this cluster; artifacts must be pushed again.\n", ex.Cluster, len(ex.Audit))
	return nil
}

func cmdConsole(op *Operator, args []string) error {
	fs := flags("console")
	ro := fs.Bool("read-only", false, "read-only session")
	ttl := fs.Duration("ttl", 12*time.Hour, "session lifetime")
	if err := fs.Parse(args); err != nil {
		return err
	}
	actions := []string{"api.read", "api.write", "api.admin"}
	if *ro {
		actions = []string{"api.read"}
	}
	tok := op.Token(*ttl, actions, "console")
	scheme := "http"
	if op.Cfg.TLS {
		scheme = "https"
	}
	fmt.Printf("%s://%s/#token=%s\n", scheme, op.Cfg.Endpoints[0], tok)
	fmt.Fprintf(os.Stderr, "console session valid for %s (%v). The token stays in the URL fragment and is never sent to a server log.\n", *ttl, actions)
	return nil
}

func cmdToken(op *Operator, args []string) error {
	fs := flags("token")
	ro := fs.Bool("read-only", false, "read-only")
	ttl := fs.Duration("ttl", time.Hour, "lifetime")
	if err := fs.Parse(args); err != nil {
		return err
	}
	actions := []string{"api.read", "api.write", "api.admin"}
	if *ro {
		actions = []string{"api.read"}
	}
	fmt.Println(op.Token(*ttl, actions, "token"))
	return nil
}

var _ = http.MethodGet
