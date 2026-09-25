package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/pki"
)

func init() {
	reg("init", "create a cluster trust root (root key + root CA) in --home", false, cmdInit)
	reg("cp bootstrap", "bootstrap the first control-plane member: --api ADDR --code CODE|--code-file FILE", true, cmdBootstrap)
	reg("cp add-member", "add a control-plane member: --api ADDR --code CODE|--code-file FILE", true, cmdAddMember)
	reg("cp remove-member", "remove a control-plane member: --id MEMBER", true, cmdRemoveMember)
	reg("cp status", "show raft membership, leader and each member's health", true, cmdCPStatus)
	reg("cp backup", "write a signed control-plane backup: --out FILE [--secrets]", true, cmdBackup)
	reg("cp restore", "restore a verified backup into a freshly bootstrapped cluster: FILE", true, cmdRestore)
	reg("cp snapshot", "force a raft snapshot on the leader", true, func(op *Operator, _ []string) error {
		var out map[string]any
		if err := op.Do("POST", "/api/v1/cp/snapshot", nil, &out); err != nil {
			return err
		}
		fmt.Printf("raft snapshot %v at index %v term %v (%v bytes)\n", out["id"], out["index"], out["term"], out["size"])
		return nil
	})
	reg("cp transfer-leadership", "ask the leader to hand leadership to another voter", true, func(op *Operator, _ []string) error {
		var r Result
		if err := op.Do("POST", "/api/v1/cp/transfer-leadership", nil, &r); err != nil {
			return err
		}
		return printResult(r)
	})
	reg("cp rotate-root", "rotate the cluster root key (old root signs the rotation)", true, cmdRotateRoot)
	reg("verify backup", "verify a control-plane backup file offline: FILE", false, func(_ *Operator, args []string) error {
		if len(args) != 1 {
			return errors.New("usage: dh verify backup FILE")
		}
		env, err := readEnvelope(args[0])
		if err != nil {
			return err
		}
		b, st, err := control.VerifyBackup(env)
		if err != nil {
			return err
		}
		fmt.Printf("backup OK: cluster %s index %d, %d audit entries verified, %d apps, %d hosts, signed by %s, secrets included: %v\n",
			b.Cluster, b.Index, len(st.Audit.Entries), len(st.Apps), len(st.Nodes), b.Member, b.Secrets)
		return nil
	})
}

func cmdInit(op *Operator, args []string) error {
	fs := flags("init")
	cluster := fs.String("cluster", "", "cluster name (lowercase)")
	useTLS := fs.Bool("tls", true, "members serve the API over TLS with root-issued certificates (disable only for loopback development)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *cluster == "" {
		return errors.New("--cluster is required")
	}
	dir := op.clusterDir(*cluster)
	if _, err := os.Stat(filepath.Join(dir, "root", "identity.key")); err == nil {
		return fmt.Errorf("cluster %s already exists in %s", *cluster, dir)
	}
	root, err := identity.LoadOrCreate(filepath.Join(dir, "root"))
	if err != nil {
		return err
	}
	ca, err := pki.RootCA(root, *cluster)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "root-ca.pem"), ca, 0o644); err != nil {
		return err
	}
	op.Dir, op.Cfg = dir, Config{Cluster: *cluster, TLS: *useTLS}
	if err := op.save(); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(op.Home, "current"), []byte(*cluster+"\n"), 0o600)
	fmt.Fprintf(Out, "cluster %s: root key %s (%s)\nroot CA %s\nKeep %s private: it is the trust anchor every host pins.\n",
		*cluster, root.PubString(), root.ID, filepath.Join(dir, "root-ca.pem"), filepath.Join(dir, "root", "identity.key"))
	return nil
}

type memberInfo struct {
	ID           string `json:"id"`
	Pub          string `json:"pub"`
	APIAddr      string `json:"apiAddr"`
	RaftAddr     string `json:"raftAddr"`
	Bootstrapped bool   `json:"bootstrapped"`
}

// conn is an HTTP client plus scheme for talking to one member directly.
type conn struct {
	c      *http.Client
	scheme string
}

// conn talks to members that already have root-issued certificates.
func (op *Operator) conn() conn { return conn{c: op.client, scheme: op.scheme} }

// memberConn talks to a member that has no credentials yet. With TLS, its
// certificate is a throwaway pinned by fingerprint: the member writes it to
// bootstrap.fingerprint next to bootstrap.code and prints it in its log.
func (op *Operator) memberConn(fingerprint, codeFile string) (conn, error) {
	if !op.Cfg.TLS {
		return conn{c: &http.Client{Timeout: 30 * time.Second}, scheme: "http"}, nil
	}
	if fingerprint == "" && codeFile != "" {
		if b, err := os.ReadFile(filepath.Join(filepath.Dir(codeFile), "bootstrap.fingerprint")); err == nil {
			fingerprint = strings.TrimSpace(string(b))
		}
	}
	if !strings.HasPrefix(fingerprint, "sha256:") {
		return conn{}, errors.New("this cluster uses TLS: pass --fingerprint sha256:… (the member prints it and writes bootstrap.fingerprint next to bootstrap.code)")
	}
	return conn{c: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: pki.PinnedTLS(fingerprint)}}, scheme: "https"}, nil
}

func (k conn) get(addr, path string, out any) error {
	resp, err := k.c.Get(k.scheme + "://" + addr + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, msgOf(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (k conn) post(addr, path string, body, out any) error {
	b, _ := json.Marshal(body)
	resp, err := k.c.Post(k.scheme+"://"+addr+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, msgOf(data))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func codeFrom(code, file string) (string, error) {
	if code != "" {
		return code, nil
	}
	if file == "" {
		return "", errors.New("--code or --code-file is required (the member writes it to <data>/bootstrap.code)")
	}
	b, err := os.ReadFile(file)
	return strings.TrimSpace(string(b)), err
}

// delegation mints the root → member capability carried in the roster.
func (op *Operator) delegation(memberPub string) (string, error) {
	tok, err := capability.Mint(op.Root, capability.Caveats{
		Actions:   []string{"workload.*", "volume.*", "mesh.*", "edge.*", "federation.*"},
		Resources: []string{"*"},
	}, memberPub, "control-plane member")
	if err != nil {
		return "", err
	}
	return tok.Encode(), nil
}

func (op *Operator) signRoster(version int64, members []api.Member) (*envelope.Envelope, error) {
	return envelope.Sign(op.Root, "", envelope.KindRoster, api.Roster{Cluster: op.Cfg.Cluster, Version: version, Root: op.Root.PubString(), Members: members, Issued: time.Now().UnixMilli()})
}

func (op *Operator) memberCert(m memberInfo) (string, error) {
	pub, err := identity.DecodePub(m.Pub)
	if err != nil {
		return "", err
	}
	cert, err := pki.IssueMember(op.Root, op.CAPEM, pub, []string{m.APIAddr, m.RaftAddr})
	return string(cert), err
}

func cmdBootstrap(op *Operator, args []string) error {
	fs := flags("cp bootstrap")
	addr := fs.String("api", "127.0.0.1:7700", "member API address")
	code := fs.String("code", "", "bootstrap code")
	codeFile := fs.String("code-file", "", "file containing the bootstrap code")
	fingerprint := fs.String("fingerprint", "", "TLS certificate fingerprint the member printed (default: bootstrap.fingerprint next to --code-file)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := codeFrom(*code, *codeFile)
	if err != nil {
		return err
	}
	k, err := op.memberConn(*fingerprint, *codeFile)
	if err != nil {
		return err
	}
	var m memberInfo
	if err := k.get(*addr, "/api/v1/member", &m); err != nil {
		return err
	}
	if m.Bootstrapped {
		return fmt.Errorf("member %s already has credentials", m.ID)
	}
	del, err := op.delegation(m.Pub)
	if err != nil {
		return err
	}
	roster, err := op.signRoster(1, []api.Member{{ID: m.ID, Pub: m.Pub, APIAddr: m.APIAddr, RaftAddr: m.RaftAddr, Delegation: del}})
	if err != nil {
		return err
	}
	cert, err := op.memberCert(m)
	if err != nil {
		return err
	}
	var r Result
	if err := k.post(*addr, "/api/v1/bootstrap", map[string]any{"code": c, "cluster": op.Cfg.Cluster, "root": op.Root.PubString(),
		"rootCaCert": string(op.CAPEM), "memberCert": cert, "roster": roster}, &r); err != nil {
		return err
	}
	op.Cfg.Endpoints = []string{m.APIAddr}
	if err := op.save(); err != nil {
		return err
	}
	fmt.Fprintf(Out, "bootstrapped %s: %s\n", m.ID, r.Message)
	return nil
}

func (op *Operator) currentRoster() (api.Roster, error) {
	var out struct {
		Cluster struct {
			RosterVersion int64 `json:"rosterVersion"`
		} `json:"cluster"`
	}
	_ = out
	var st struct {
		RosterBody api.Roster `json:"rosterBody"`
	}
	if err := op.Do("GET", "/api/v1/state", nil, &st); err != nil {
		return api.Roster{}, err
	}
	return st.RosterBody, nil
}

func cmdAddMember(op *Operator, args []string) error {
	fs := flags("cp add-member")
	addr := fs.String("api", "", "new member API address")
	code := fs.String("code", "", "bootstrap code of the new member")
	codeFile := fs.String("code-file", "", "file containing the code")
	fingerprint := fs.String("fingerprint", "", "TLS certificate fingerprint the new member printed (default: bootstrap.fingerprint next to --code-file)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := codeFrom(*code, *codeFile)
	if err != nil {
		return err
	}
	k, err := op.memberConn(*fingerprint, *codeFile)
	if err != nil {
		return err
	}
	var m memberInfo
	if err := k.get(*addr, "/api/v1/member", &m); err != nil {
		return err
	}
	cur, err := op.currentRoster()
	if err != nil {
		return err
	}
	for _, x := range cur.Members {
		if x.ID == m.ID {
			return fmt.Errorf("%s is already a member", m.ID)
		}
	}
	del, err := op.delegation(m.Pub)
	if err != nil {
		return err
	}
	members := append(cur.Members, api.Member{ID: m.ID, Pub: m.Pub, APIAddr: m.APIAddr, RaftAddr: m.RaftAddr, Delegation: del})
	roster, err := op.signRoster(cur.Version+1, members)
	if err != nil {
		return err
	}
	cert, err := op.memberCert(m)
	if err != nil {
		return err
	}
	if err := k.post(*addr, "/api/v1/join-cluster", map[string]any{"code": c, "cluster": op.Cfg.Cluster, "root": op.Root.PubString(),
		"rootCaCert": string(op.CAPEM), "memberCert": cert}, nil); err != nil {
		return fmt.Errorf("credentials for new member: %w", err)
	}
	var r Result
	if err := op.Do("POST", "/api/v1/roster", roster, &r); err != nil {
		return fmt.Errorf("publish roster: %w", err)
	}
	if err := op.Do("POST", "/api/v1/cp/add-voter", map[string]string{"id": m.ID, "raftAddr": m.RaftAddr}, &r); err != nil {
		return fmt.Errorf("add voter: %w", err)
	}
	op.Cfg.Endpoints = appendUnique(op.Cfg.Endpoints, m.APIAddr)
	if err := op.save(); err != nil {
		return err
	}
	fmt.Fprintf(Out, "member %s added (roster v%d, %d members)\n", m.ID, cur.Version+1, len(members))
	return nil
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func cmdRemoveMember(op *Operator, args []string) error {
	fs := flags("cp remove-member")
	id := fs.String("id", "", "member id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cur, err := op.currentRoster()
	if err != nil {
		return err
	}
	var members []api.Member
	var gone *api.Member
	for i, m := range cur.Members {
		if m.ID == *id {
			gone = &cur.Members[i]
			continue
		}
		members = append(members, m)
	}
	if gone == nil {
		return fmt.Errorf("%s is not a member", *id)
	}
	var r Result
	if err := op.Do("POST", "/api/v1/cp/remove-server", map[string]string{"id": *id}, &r); err != nil {
		return err
	}
	roster, err := op.signRoster(cur.Version+1, members)
	if err != nil {
		return err
	}
	if err := op.Do("POST", "/api/v1/roster", roster, &r); err != nil {
		return err
	}
	var eps []string
	for _, e := range op.Cfg.Endpoints {
		if e != gone.APIAddr {
			eps = append(eps, e)
		}
	}
	op.Cfg.Endpoints = eps
	_ = op.save()
	fmt.Printf("member %s removed; roster v%d\n", *id, cur.Version+1)
	return nil
}

func cmdCPStatus(op *Operator, _ []string) error {
	var info struct {
		Servers []map[string]any `json:"servers"`
		Leader  string           `json:"leader"`
	}
	if err := op.Do("GET", "/api/v1/cp/raft", nil, &info); err != nil {
		return err
	}
	cur, _ := op.currentRoster()
	w := table("MEMBER", "SUFFRAGE", "RAFT", "API", "STATE", "INDEX", "LEADER")
	for _, sv := range info.Servers {
		id := fmt.Sprint(sv["id"])
		api := ""
		for _, m := range cur.Members {
			if m.ID == id {
				api = m.APIAddr
			}
		}
		state, index := "unreachable", "-"
		var h map[string]any
		if api != "" && op.conn().get(api, "/api/v1/health", &h) == nil {
			state, index = fmt.Sprint(h["state"]), fmt.Sprint(h["index"])
		}
		mark := ""
		if id == info.Leader {
			mark = "*"
		}
		fmt.Fprintf(w, "%s\t%v\t%v\t%s\t%s\t%s\t%s\n", id, sv["suffrage"], sv["address"], api, state, index, mark)
	}
	w.Flush()
	fmt.Printf("roster v%d, %d member(s)\n", cur.Version, len(cur.Members))
	return nil
}

func cmdBackup(op *Operator, args []string) error {
	fs := flags("cp backup")
	out := fs.String("out", "dh-backup.json", "output file")
	secrets := fs.Bool("secrets", false, "include the local CA private key (export-controlled)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var raw []byte
	path := "/api/v1/cp/backup"
	if *secrets {
		path += "?secrets=true"
	}
	if err := op.Do("POST", path, nil, &raw); err != nil {
		return err
	}
	var env envelope.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	b, st, err := control.VerifyBackup(&env)
	if err != nil {
		return fmt.Errorf("backup did not verify: %w", err)
	}
	if err := os.WriteFile(*out, raw, 0o600); err != nil {
		return err
	}
	fmt.Printf("wrote %s: index %d, %d audit entries, %d apps, %d hosts, state %s (verified)\n", *out, b.Index, len(st.Audit.Entries), len(st.Apps), len(st.Nodes), short(b.StateHash))
	return nil
}

func readEnvelope(path string) (*envelope.Envelope, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var env envelope.Envelope
	return &env, json.Unmarshal(raw, &env)
}

func cmdRestore(op *Operator, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: dh cp restore FILE")
	}
	env, err := readEnvelope(args[0])
	if err != nil {
		return err
	}
	if _, _, err := control.VerifyBackup(env); err != nil {
		return fmt.Errorf("refusing to restore: %w", err)
	}
	var r Result
	if err := op.Do("POST", "/api/v1/cp/restore", env, &r); err != nil {
		return err
	}
	return printResult(r)
}

func cmdRotateRoot(op *Operator, _ []string) error {
	next, err := identity.Generate()
	if err != nil {
		return err
	}
	cur, err := op.currentRoster()
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	rot, err := envelope.Sign(op.Root, "", envelope.KindRotation, api.Rotation{Node: "root:" + op.Cfg.Cluster, OldPub: op.Root.PubString(), NewPub: next.PubString(), NotBefore: now, TS: now})
	if err != nil {
		return err
	}
	old := op.Root
	op.Root = next
	var members []api.Member
	for _, m := range cur.Members {
		del, err := op.delegation(m.Pub)
		if err != nil {
			return err
		}
		m.Delegation = del
		members = append(members, m)
	}
	roster, err := envelope.Sign(next, "", envelope.KindRoster, api.Roster{Cluster: op.Cfg.Cluster, Version: cur.Version + 1, Root: next.PubString(), Members: members, Issued: now})
	if err != nil {
		return err
	}
	op.Root = old
	var r Result
	if err := op.Do("POST", "/api/v1/root-rotate", map[string]any{"rotation": rot, "roster": roster}, &r); err != nil {
		return err
	}
	if !r.OK {
		return errors.New(r.Message)
	}
	retired := filepath.Join(op.Dir, fmt.Sprintf("root.retired-%d", now))
	if err := os.Rename(filepath.Join(op.Dir, "root"), retired); err != nil {
		return err
	}
	if err := identity.Save(filepath.Join(op.Dir, "root"), next); err != nil {
		return err
	}
	fmt.Printf("root rotated to %s; previous root kept in %s. Hosts follow the rotation because it is signed by the root they pinned.\n", next.PubString(), retired)
	return nil
}
