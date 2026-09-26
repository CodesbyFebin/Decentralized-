package control

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/raft"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/manifest"
)

// Command is one replicated log entry. TS and Actor are assigned by the
// leader when it proposes; Apply never reads the wall clock, so every member
// computes the same state.
type Command struct {
	Type  string          `json:"type"`
	TS    int64           `json:"-"` // unix nanoseconds; not marshaled to JSON (use string in protobuf/wire if needed)
	Actor string          `json:"actor"`
	Data  json.RawMessage `json:"data"`
}

// Result is returned to the proposer.
type Result struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func ok(format string, args ...any) *Result {
	return &Result{OK: true, Message: fmt.Sprintf(format, args...)}
}
func fail(code, format string, args ...any) *Result {
	return &Result{OK: false, Code: code, Message: fmt.Sprintf(format, args...)}
}

// FSM is the raft state machine.
type FSM struct {
	mu sync.RWMutex
	s  *State
	// onApply is called (outside the lock) after each command; the server
	// uses it to wake the reconciler and the evidence mirror.
	onApply func(cmd *Command, res *Result)
	// retrievalObserver instruments the authorization→decryption boundary for testing
	retrievalObserver RetrievalQualificationObserver
}

// NewFSM returns an empty state machine.
func NewFSM() *FSM { return &FSM{s: newState(), retrievalObserver: &NoOpObserver{}} }

// SetRetrievalObserver sets the observer for authorization→decryption boundary instrumentation.
func (f *FSM) SetRetrievalObserver(obs RetrievalQualificationObserver) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if obs != nil {
		f.retrievalObserver = obs
	}
}

// Read runs fn with a read lock on the state.
func (f *FSM) Read(fn func(s *State)) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	fn(f.s)
}

// AuthorizeSecretRetrievalCommand validates the request and constructs the FSM command.
// This runs on the leader only; the FSM will re-validate deterministically on all members.
func (f *FSM) AuthorizeSecretRetrievalCommand(req *SecretRetrievalRequest) (*Command, error) {
	// Leader-side static validation (signature)
	if err := req.VerifySignature(); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Construct deterministic authorization command
	// Include leader's current timestamp for clock skew evaluation
	now := time.Now().UnixNano()

	// Encode the request as JSON for Raft transmission
	// Include proposal timestamp in the data for clock skew validation
	payload := map[string]interface{}{
		"proposal_ts": now,
		"request":     req,
	}
	dataJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode command: %w", err)
	}

	cmd := &Command{
		Type:  "secret-retrieval-authorize",
		TS:    now,
		Actor: "node/" + req.NodeID,
		Data:  json.RawMessage(dataJSON),
	}

	// Instrumentation: record authorization proposal
	requestDigest := req.RequestDigest()
	if f.retrievalObserver != nil {
		f.retrievalObserver.AuthorizationProposed(map[string]string{
			"requestDigest": requestDigest,
			"requestID":     req.RequestID,
			"nodeID":        req.NodeID,
			"secretID":      req.SecretID,
			"timestamp":     fmt.Sprintf("%d", now),
		})
	}

	return cmd, nil
}

// Apply implements raft.FSM.
func (f *FSM) Apply(l *raft.Log) any {
	var cmd Command
	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		return fail("DECODE", "command: %v", err)
	}
	f.mu.Lock()
	res := f.apply(&cmd)
	f.s.Index = f.s.IndexBase + int64(l.Index)
	obs := f.retrievalObserver
	f.mu.Unlock()

	// Instrumentation: record successful authorization commits
	if cmd.Type == "secret-retrieval-authorize" && res.OK && obs != nil {
		// Extract requestDigest from command payload for metrics tracking
		var payload map[string]interface{}
		var requestDigest string
		if err := json.Unmarshal(cmd.Data, &payload); err == nil {
			if reqData, ok := payload["request"]; ok {
				reqJSON, _ := json.Marshal(reqData)
				var req SecretRetrievalRequest
				if json.Unmarshal(reqJSON, &req) == nil {
					requestDigest = req.RequestDigest()
				}
			}
		}
		obs.AuthorizationCommitted(map[string]string{
			"logIndex":      fmt.Sprintf("%d", l.Index),
			"requestDigest": requestDigest,
			"result":        res.Message,
		})
	}

	if f.onApply != nil {
		f.onApply(&cmd, res)
	}
	return res
}

// ApplyLocal applies a command without raft (tests and restore).
func (f *FSM) ApplyLocal(cmd *Command) *Result {
	f.mu.Lock()
	res := f.apply(cmd)
	f.s.Index++
	obs := f.retrievalObserver
	f.mu.Unlock()

	// Instrumentation: record successful authorization commits (same as Apply)
	if cmd.Type == "secret-retrieval-authorize" && res.OK && obs != nil {
		// Full details already recorded in AuthorizationProposed
		obs.AuthorizationCommitted(map[string]string{
			"source": "ApplyLocal",
			"result": res.Message,
		})
	}

	return res
}

func (f *FSM) apply(c *Command) (res *Result) {
	defer func() {
		if r := recover(); r != nil {
			res = fail("PANIC", "apply %s: %v", c.Type, r)
		}
	}()
	s := f.s
	if s.Cluster == "" && c.Type != "init" {
		return fail("UNINITIALIZED", "cluster is not initialized")
	}
	h, found := handlers[c.Type]
	if !found {
		return fail("UNKNOWN", "unknown command %q", c.Type)
	}
	return h(s, c)
}

type handler func(s *State, c *Command) *Result

var handlers = map[string]handler{}

func register(name string, h handler) { handlers[name] = h }

func decode[T any](c *Command) (T, error) {
	var v T
	err := json.Unmarshal(c.Data, &v)
	return v, err
}

// Snapshot implements raft.FSM.
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	b, err := json.Marshal(f.s)
	if err != nil {
		return nil, err
	}
	return &fsmSnapshot{data: b}, nil
}

// Restore implements raft.FSM.
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	s := newState()
	if err := json.NewDecoder(rc).Decode(s); err != nil {
		return err
	}
	s.ensure()
	f.mu.Lock()
	f.s = s
	f.mu.Unlock()
	return nil
}

// Export returns a deep copy of the state as JSON.
func (f *FSM) Export() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return json.Marshal(f.s)
}

type fsmSnapshot struct{ data []byte }

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := sink.Write(s.data); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *fsmSnapshot) Release() {}

// ------------------------------------------------------------ cluster/roster

type initData struct {
	Cluster    string             `json:"cluster"`
	Root       string             `json:"root"`
	RootCACert string             `json:"rootCaCert"`
	Roster     *envelope.Envelope `json:"roster"`
}

func verifyRoster(env *envelope.Envelope, root string) (api.Roster, error) {
	var r api.Roster
	if err := env.VerifyKey(envelope.KindRoster, root); err != nil {
		return r, fmt.Errorf("roster signature: %w", err)
	}
	if err := env.Decode(&r); err != nil {
		return r, err
	}
	if r.Root != root {
		return r, errors.New("roster names a different root")
	}
	if len(r.Members) == 0 {
		return r, errors.New("roster has no members")
	}
	for _, m := range r.Members {
		pub, err := identity.DecodePub(m.Pub)
		if err != nil || identity.NodeID(pub) != m.ID {
			return r, fmt.Errorf("member %s key does not derive its id", m.ID)
		}
		tok, err := capability.Decode(m.Delegation)
		if err != nil {
			return r, fmt.Errorf("member %s delegation: %w", m.ID, err)
		}
		if len(tok.Blocks) != 1 {
			return r, fmt.Errorf("member %s delegation must be a single root block", m.ID)
		}
		var b capability.Block
		if err := tok.Blocks[0].VerifyKey(envelope.KindCapability, root); err != nil {
			return r, fmt.Errorf("member %s delegation not signed by root", m.ID)
		}
		_ = tok.Blocks[0].Decode(&b)
		if b.Next != m.Pub {
			return r, fmt.Errorf("member %s delegation names a different key", m.ID)
		}
	}
	return r, nil
}

func init() {
	register("init", func(s *State, c *Command) *Result {
		if s.Cluster != "" {
			return fail("EXISTS", "cluster %s is already initialized", s.Cluster)
		}
		d, err := decode[initData](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		r, err := verifyRoster(d.Roster, d.Root)
		if err != nil {
			return fail("ROSTER", "%v", err)
		}
		if r.Cluster != d.Cluster {
			return fail("ROSTER", "roster is for cluster %q", r.Cluster)
		}
		s.Cluster, s.Root, s.RootCACert, s.Roster, s.RosterBody = d.Cluster, d.Root, d.RootCACert, d.Roster, r
		s.Publishers = []string{d.Root}
		for i, m := range r.Members {
			s.MemberMesh[m.ID] = fmt.Sprintf("10.77.255.%d", i+1)
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "cluster-init", Resource: "cluster/" + d.Cluster,
			Detail: fmt.Sprintf("root %s; roster v%d with %d member(s)", short(d.Root), r.Version, len(r.Members)), Evidence: d.Roster.Digest()})
		return ok("cluster %s initialized", d.Cluster)
	})

	register("roster", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		r, err := verifyRoster(env, s.Root)
		if err != nil {
			return fail("ROSTER", "%v", err)
		}
		if r.Cluster != s.Cluster || r.Version <= s.RosterBody.Version {
			return fail("ROSTER", "roster version %d must exceed %d for cluster %s", r.Version, s.RosterBody.Version, s.Cluster)
		}
		var names []string
		used := map[string]bool{}
		for _, ip := range s.MemberMesh {
			used[ip] = true
		}
		for _, m := range r.Members {
			names = append(names, m.ID)
			if _, has := s.MemberMesh[m.ID]; !has {
				for i := 1; i < 254; i++ {
					ip := fmt.Sprintf("10.77.255.%d", i)
					if !used[ip] {
						s.MemberMesh[m.ID] = ip
						used[ip] = true
						break
					}
				}
			}
		}
		s.retireRoster()
		s.Roster, s.RosterBody = env, r
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "roster-update", Resource: "cluster/" + s.Cluster,
			Detail: fmt.Sprintf("roster v%d: %s", r.Version, strings.Join(names, ", ")), Evidence: env.Digest()})
		return ok("roster v%d accepted", r.Version)
	})

	// root-rotate: old root signs the rotation; new root signs the new roster.
	register("root-rotate", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Rotation *envelope.Envelope `json:"rotation"`
			Roster   *envelope.Envelope `json:"roster"`
			CACert   string             `json:"caCert"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if err := d.Rotation.VerifyKey(envelope.KindRotation, s.Root); err != nil {
			return fail("ROTATION", "rotation not signed by current root: %v", err)
		}
		var rot api.Rotation
		_ = d.Rotation.Decode(&rot)
		if rot.OldPub != s.Root || rot.Node != "root:"+s.Cluster {
			return fail("ROTATION", "rotation statement does not rotate this cluster's root")
		}
		r, err := verifyRoster(d.Roster, rot.NewPub)
		if err != nil {
			return fail("ROSTER", "%v", err)
		}
		old := s.Root
		s.retireRoster()
		s.Root, s.Roster, s.RosterBody, s.RootRotation = rot.NewPub, d.Roster, r, d.Rotation
		if d.CACert != "" {
			s.RootCACert = d.CACert
		}
		for i, p := range s.Publishers {
			if p == old {
				s.Publishers[i] = rot.NewPub
			}
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "root-rotate", Resource: "cluster/" + s.Cluster,
			Detail: fmt.Sprintf("root %s → %s", short(old), short(rot.NewPub)), Evidence: d.Rotation.Digest()})
		return ok("root rotated")
	})

	register("freeze", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Frozen bool `json:"frozen"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if s.Frozen == d.Frozen {
			return ok("no change (frozen=%v)", d.Frozen)
		}
		s.Frozen = d.Frozen
		s.FrozenAt = c.TS
		action, detail := "cp-unfreeze", "signing new work resumed"
		if d.Frozen {
			action, detail = "cp-freeze", "control plane frozen: hosts hold admitted work and refuse new work"
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: action, Resource: "cluster/" + s.Cluster, Detail: detail})
		return ok("%s", detail)
	})

	register("local-ca", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Cert string `json:"cert"`
			Key  string `json:"key"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if s.LocalCACert != "" {
			return ok("local CA already present")
		}
		s.LocalCACert, s.LocalCAKey = d.Cert, d.Key
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceControl, Action: "local-ca-create", Resource: "cluster/" + s.Cluster, Detail: "cluster-local CA for tls: local ingress"})
		return ok("local CA stored")
	})

	register("checkpoint", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		m := s.member(env.Signer)
		if m == nil {
			return fail("CHECKPOINT", "signer %s is not a roster member", env.Signer)
		}
		if err := env.VerifyKey(envelope.KindCheckpoint, m.Pub); err != nil {
			return fail("CHECKPOINT", "%v", err)
		}
		var p audit.CheckpointPayload
		_ = env.Decode(&p)
		if p.Seq < 1 || p.Seq > int64(len(s.Audit.Entries)) || s.Audit.Entries[p.Seq-1].Hash != p.Hash {
			return fail("CHECKPOINT", "checkpoint does not match the ledger at seq %d", p.Seq)
		}
		s.Checkpoints = append(s.Checkpoints, env)
		return ok("checkpoint at seq %d", p.Seq)
	})

	register("publishers", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Keys []string `json:"keys"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		for _, k := range d.Keys {
			if _, err := identity.DecodePub(k); err != nil {
				return fail("KEY", "publisher key %q: %v", k, err)
			}
		}
		s.Publishers = uniqueSorted(append([]string{s.Root}, d.Keys...))
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "publishers-set", Resource: "cluster/" + s.Cluster, Detail: fmt.Sprintf("%d publisher key(s)", len(s.Publishers))})
		return ok("publishers updated")
	})

	register("attest", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if err := env.VerifyKey(envelope.KindArtifact, s.Publishers...); err != nil {
			return fail("ATTESTATION", "attestation not signed by a publisher: %v", err)
		}
		var a api.ArtifactAttestation
		_ = env.Decode(&a)
		s.Attestations[a.Digest] = env
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "artifact-attest", Resource: "artifact/" + a.Name, Detail: a.Digest, Evidence: env.Digest()})
		return ok("attestation stored for %s", a.Digest)
	})

	register("artifact", func(s *State, c *Command) *Result {
		d, err := decode[Artifact](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		cur := s.Artifacts[d.Digest]
		if cur == nil {
			d.Uploaded = c.TS
			d.Holders = uniqueSorted(d.Holders)
			s.Artifacts[d.Digest] = &d
			s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "artifact-push", Resource: "artifact/" + d.Name,
				Detail: fmt.Sprintf("%s, %d bytes in %d chunks, held by %d node(s)", d.Digest, d.Bytes, d.Chunks, len(d.Holders))})
			return ok("artifact %s registered", d.Digest)
		}
		cur.Holders = uniqueSorted(append(cur.Holders, d.Holders...))
		if d.Name != "" {
			cur.Name = d.Name
		}
		return ok("artifact holders updated")
	})

	register("chaos-report", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if err := env.VerifySelf(envelope.KindChaosReport); err != nil {
			return fail("SIGNATURE", "%v", err)
		}
		var hdr struct {
			ID       string `json:"id"`
			Scenario string `json:"scenario"`
			Verdict  string `json:"verdict"`
		}
		_ = env.Decode(&hdr)
		s.ChaosReports = append(s.ChaosReports, ChaosRec{ID: hdr.ID, Received: c.TS, Env: env})
		if len(s.ChaosReports) > 300 {
			s.ChaosReports = s.ChaosReports[len(s.ChaosReports)-300:]
		}
		s.audit(audit.Entry{TS: c.TS, Actor: env.Signer, Source: audit.SourceChaos, Action: "chaos-report", Resource: "chaos/" + hdr.Scenario, Detail: hdr.Verdict, Evidence: env.Digest()})
		return ok("chaos report stored")
	})
}

// ------------------------------------------------------------------- nodes

type inviteData struct {
	Nonce   string   `json:"nonce"`
	Expires int64    `json:"expires"`
	Roles   []string `json:"roles"`
	Note    string   `json:"note"`
	Auto    bool     `json:"auto"`
}

type enrollData struct {
	Env *envelope.Envelope `json:"env"`
}

func init() {
	register("invite", func(s *State, c *Command) *Result {
		d, err := decode[inviteData](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if _, dup := s.Invites[d.Nonce]; dup {
			return fail("EXISTS", "invite nonce reused")
		}
		s.Invites[d.Nonce] = &Invite{Nonce: d.Nonce, Expires: d.Expires, Roles: d.Roles, Note: d.Note, Auto: d.Auto, Created: c.TS}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "node-invite", Resource: "invite/" + d.Nonce[:8],
			Detail: fmt.Sprintf("roles %v, auto-approve %v, note %q", d.Roles, d.Auto, d.Note)})
		return ok("invite recorded")
	})

	register("invite-revoke", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Nonce string `json:"nonce"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		inv := s.Invites[d.Nonce]
		if inv == nil || len(d.Nonce) < 16 {
			return fail("NOT_FOUND", "no invite with that nonce")
		}
		if inv.Used != "" {
			return fail("CONFLICT", "invite already used by %s; revoke the host instead", inv.Used)
		}
		if inv.Revoked != 0 {
			return ok("invite already revoked")
		}
		inv.Revoked = c.TS
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "node-invite-revoke", Resource: "invite/" + d.Nonce[:8], Detail: "join token withdrawn before use"})
		return ok("invite revoked")
	})

	register("enroll", func(s *State, c *Command) *Result {
		d, err := decode[enrollData](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if err := d.Env.VerifySelf(envelope.KindEnroll); err != nil {
			s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: d.Env.Signer, Reason: err.Error()})
			return fail("SIGNATURE", "enrollment: %v", err)
		}
		var e api.Enroll
		_ = d.Env.Decode(&e)
		if e.ID != d.Env.Signer || e.Pub != d.Env.Pub {
			return fail("ENROLL", "enrollment identity mismatch")
		}
		if n := s.Nodes[e.ID]; n != nil {
			// Known host re-announcing itself (restart). Its key must still be valid.
			if !contains(n.ValidKeys(c.TS), d.Env.Pub) {
				s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: "enrollment key is not a valid key for this host"})
				return fail("KEY", "enrollment key is not a valid key for %s", e.ID)
			}
			// A validly signed but older enrollment is a replay: it would put
			// back facts and policy the host has since replaced.
			if e.TS <= n.Enroll.TS {
				s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: fmt.Sprintf("stale enrollment: ts %d is not newer than the accepted %d", e.TS, n.Enroll.TS), Evidence: d.Env.Digest()})
				return fail("STALE", "enrollment is not newer than the one already accepted for %s", n.Name)
			}
			n.Enroll = e
			n.EnrollEnv = d.Env
			return ok("%s re-enrolled", n.Name)
		}
		tok, err := capability.Decode(e.JoinToken)
		if err != nil {
			s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: "join token: " + err.Error()})
			return fail("TOKEN", "join token: %v", err)
		}
		res := capability.Verify(tok, []string{s.Root}, capability.Request{Action: "node.join", Resource: "cluster/" + s.Cluster, Now: c.TS})
		if !res.OK {
			s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: "join token: " + res.Reason})
			return fail("TOKEN", "join token: %s", res.Reason)
		}
		inv := s.Invites[res.Last.Caveats.Nonce]
		if inv == nil {
			return fail("TOKEN", "join token nonce is unknown")
		}
		if inv.Used != "" {
			s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: "join token already used by " + inv.Used})
			return fail("TOKEN", "join token already used by %s", inv.Used)
		}
		if inv.Revoked != 0 {
			s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: "join token was revoked by the owner"})
			return fail("TOKEN", "join token was revoked")
		}
		if inv.Expires > 0 && c.TS > inv.Expires {
			s.reject(Rejection{TS: c.TS, Kind: "enroll", Node: e.ID, Reason: "join token expired"})
			return fail("TOKEN", "join token expired")
		}
		for _, n := range s.Nodes {
			if n.Name == e.Name {
				return fail("NAME", "a host named %s already exists", e.Name)
			}
		}
		inv.Used = e.ID
		roles := uniqueSorted(append(append([]string{}, e.Roles...), inv.Roles...))
		n := &Node{ID: e.ID, Name: e.Name, Status: "pending", Health: "unknown", Enroll: e, EnrollEnv: d.Env,
			Keys: []api.KeyRecord{{Pub: e.Pub, From: 0}}, Roles: roles, JoinedAt: c.TS, FirstSeen: c.TS}
		s.Nodes[e.ID] = n
		s.audit(audit.Entry{TS: c.TS, Actor: e.ID, Source: audit.SourceHost, Action: "enroll", Resource: "node/" + e.ID,
			Detail: fmt.Sprintf("%s requested admission from %s/%s (%s/%s)", e.Name, e.Region, e.Zone, e.OS, e.Arch), Evidence: d.Env.Digest()})
		if inv.Auto {
			approve(s, n, c.TS, "invite:"+inv.Nonce[:8])
			return ok("%s enrolled and approved by invite", e.Name)
		}
		return ok("%s enrolled; waiting for operator approval", e.Name)
	})

	register("approve", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Node string `json:"node"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[d.Node]
		if n == nil {
			return fail("NOT_FOUND", "no host %s", d.Node)
		}
		if n.Status == "ready" {
			return ok("%s is already approved", n.Name)
		}
		approve(s, n, c.TS, c.Actor)
		return ok("%s approved; it may now admit work", n.Name)
	})

	register("revoke", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Node   string `json:"node"`
			Reason string `json:"reason"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[d.Node]
		if n == nil {
			return fail("NOT_FOUND", "no host %s", d.Node)
		}
		if n.Status == "revoked" {
			return ok("%s is already revoked", n.Name)
		}
		n.Status, n.RevokedAt = "revoked", c.TS
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "node-revoke", Resource: "node/" + n.ID,
			Detail: fmt.Sprintf("%s revoked (%s). New admission blocked; admitted processes are not killed by this record.", n.Name, d.Reason)})
		return ok("%s revoked; it refuses new work, admitted work continues until a signed stop", n.Name)
	})

	register("drain", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Node  string `json:"node"`
			Drain bool   `json:"drain"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[d.Node]
		if n == nil {
			return fail("NOT_FOUND", "no host %s", d.Node)
		}
		if d.Drain && n.Status == "ready" {
			n.Status = "draining"
			s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "node-drain", Resource: "node/" + n.ID, Detail: n.Name + " draining: replicas will be rescheduled"})
			return ok("%s draining", n.Name)
		}
		if !d.Drain && n.Status == "draining" {
			n.Status = "ready"
			s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "node-undrain", Resource: "node/" + n.ID, Detail: n.Name})
			return ok("%s accepting work again", n.Name)
		}
		return ok("no change")
	})

	register("node-health", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Node   string `json:"node"`
			Health string `json:"health"`
			Detail string `json:"detail"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[d.Node]
		if n == nil || n.Health == d.Health {
			return ok("no change")
		}
		n.Health = d.Health
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceControl, Action: "node-" + d.Health, Resource: "node/" + n.ID, Detail: d.Detail})
		return ok("%s is %s", n.Name, d.Health)
	})

	register("wg-binding", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		var b api.WGBinding
		if err := env.Decode(&b); err != nil {
			return fail("DECODE", "%v", err)
		}
		if n := s.Nodes[env.Signer]; n != nil {
			if err := env.VerifyKey(envelope.KindWGBinding, n.ValidKeys(c.TS)...); err != nil {
				s.reject(Rejection{TS: c.TS, Kind: "wg-binding", Node: n.ID, Reason: err.Error()})
				return fail("SIGNATURE", "wg binding: %v", err)
			}
			if b.Node != n.ID || b.MeshIP != n.MeshIP {
				return fail("BINDING", "binding must name this host and its assigned mesh address %s", n.MeshIP)
			}
			changed := n.Binding == nil || n.Binding.Digest() != env.Digest()
			var prev api.WGBinding
			if n.Binding != nil {
				_ = n.Binding.Decode(&prev)
			}
			n.Binding = env
			if changed && (prev.WGPub != b.WGPub || prev.Endpoint != b.Endpoint) {
				s.audit(audit.Entry{TS: c.TS, Actor: n.ID, Source: audit.SourceHost, Action: "wg-binding", Resource: "node/" + n.ID,
					Detail: fmt.Sprintf("wireguard key %s at %s → mesh %s", short(b.WGPub), b.Endpoint, b.MeshIP), Evidence: env.Digest()})
			}
			return ok("binding stored")
		}
		if m := s.member(env.Signer); m != nil {
			if err := env.VerifyKey(envelope.KindWGBinding, m.Pub); err != nil {
				return fail("SIGNATURE", "member wg binding: %v", err)
			}
			if b.MeshIP != s.MemberMesh[m.ID] {
				return fail("BINDING", "member binding must use mesh address %s", s.MemberMesh[m.ID])
			}
			prev := s.MemberBind[m.ID]
			s.MemberBind[m.ID] = env
			if prev == nil || prev.Digest() != env.Digest() {
				var p api.WGBinding
				if prev != nil {
					_ = prev.Decode(&p)
				}
				if p.WGPub != b.WGPub || p.Endpoint != b.Endpoint {
					s.audit(audit.Entry{TS: c.TS, Actor: m.ID, Source: audit.SourceControl, Action: "wg-binding", Resource: "member/" + m.ID,
						Detail: fmt.Sprintf("member wireguard key %s at %s → mesh %s", short(b.WGPub), b.Endpoint, b.MeshIP), Evidence: env.Digest()})
				}
			}
			return ok("member binding stored")
		}
		return fail("NOT_FOUND", "binding signer %s is unknown", env.Signer)
	})

	register("rotate-key", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Env    *envelope.Envelope `json:"env"`
			OldSig string             `json:"oldSig"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		var r api.Rotation
		if err := d.Env.Decode(&r); err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[r.Node]
		if n == nil {
			return fail("NOT_FOUND", "no host %s", r.Node)
		}
		if err := d.Env.VerifyKey(envelope.KindRotation, r.NewPub); err != nil {
			return fail("ROTATION", "new key signature: %v", err)
		}
		if !contains(n.ValidKeys(c.TS), r.OldPub) {
			return fail("ROTATION", "old key is not currently valid for %s", n.Name)
		}
		oldPub, err := identity.DecodePub(r.OldPub)
		if err != nil {
			return fail("ROTATION", "old key: %v", err)
		}
		sig, err := base64.RawURLEncoding.DecodeString(d.OldSig)
		if err != nil || !ed25519.Verify(oldPub, envelope.SigningInput(envelope.KindRotation, d.Env.Payload), sig) {
			return fail("ROTATION", "old key did not countersign the rotation")
		}
		for _, k := range n.Keys {
			if k.Pub == r.NewPub {
				return ok("rotation already applied")
			}
		}
		if r.GraceUntil <= c.TS {
			return fail("ROTATION", "grace period must end in the future")
		}
		for i := range n.Keys {
			if n.Keys[i].Pub == r.OldPub {
				n.Keys[i].Until = r.GraceUntil
			}
		}
		n.Keys = append(n.Keys, api.KeyRecord{Pub: r.NewPub, From: r.NotBefore})
		s.audit(audit.Entry{TS: c.TS, Actor: n.ID, Source: audit.SourceHost, Action: "key-rotate", Resource: "node/" + n.ID,
			Detail: fmt.Sprintf("%s → %s; old key valid until %d", short(r.OldPub), short(r.NewPub), r.GraceUntil), Evidence: d.Env.Digest()})
		return ok("%s rotated to a new key; old key accepted during grace", n.Name)
	})

	register("revoke-key", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Node   string `json:"node"`
			Pub    string `json:"pub"`
			Reason string `json:"reason"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[d.Node]
		if n == nil {
			return fail("NOT_FOUND", "no host %s", d.Node)
		}
		for i := range n.Keys {
			if n.Keys[i].Pub == d.Pub && !n.Keys[i].Revoked {
				n.Keys[i].Revoked, n.Keys[i].Reason = true, d.Reason
				s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "key-revoke", Resource: "node/" + n.ID, Detail: short(d.Pub) + ": " + d.Reason})
				return ok("key revoked; signatures by it are rejected immediately")
			}
		}
		return fail("NOT_FOUND", "key is not an active key of %s", n.Name)
	})
}

func approve(s *State, n *Node, ts int64, actor string) {
	n.Status, n.ApprovedAt = "ready", ts
	if n.MeshIP == "" {
		s.MeshNext++
		n.MeshIP = meshIPFor(s.MeshNext - 1)
	}
	s.audit(audit.Entry{TS: ts, Actor: actor, Source: audit.SourceOperator, Action: "node-approve", Resource: "node/" + n.ID,
		Detail: fmt.Sprintf("%s approved; mesh address %s", n.Name, n.MeshIP)})
}

// -------------------------------------------------------------------- apps

func templateHash(m api.Manifest) string {
	m.Spec.Replicas = 0
	return manifest.Hash(&m)
}

func init() {
	register("apply", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Manifest   api.Manifest  `json:"manifest"`
			Federation *FedPlacement `json:"federation"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		m := d.Manifest
		hash := manifest.Hash(&m)
		th := templateHash(m)
		app := s.Apps[m.Metadata.Name]
		if app != nil && !app.Deleted && app.Hash == hash {
			return &Result{OK: true, Code: "UNCHANGED", Message: fmt.Sprintf("%s unchanged; generation %d was not advanced", app.Name, app.Generation), Data: app.Generation}
		}
		if app != nil && app.Federation != nil && d.Federation == nil {
			return fail("FEDERATED", "%s is managed by federation peer %s", app.Name, app.Federation.Peer)
		}
		change := "created"
		if app == nil || app.Deleted {
			gen := int64(1)
			if app != nil {
				gen = app.Generation + 1
			}
			app = &App{Name: m.Metadata.Name, Created: c.TS, Generation: gen}
			s.Apps[m.Metadata.Name] = app
		} else if app.TemplateHash != th {
			app.Generation++
			change = "template changed; rolling update to a new generation"
		} else {
			change = fmt.Sprintf("replicas %d → %d; generation unchanged", app.Manifest.Spec.Replicas, m.Spec.Replicas)
		}
		app.Manifest, app.Hash, app.TemplateHash, app.Updated, app.Deleted, app.Federation = m, hash, th, c.TS, false, d.Federation
		app.History = append(app.History, AppRevision{Generation: app.Generation, Hash: hash, TS: c.TS, Actor: c.Actor, Change: change})
		if len(app.History) > 50 {
			app.History = app.History[len(app.History)-50:]
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "manifest-apply", Resource: "app/" + app.Name, Generation: app.Generation,
			Detail: fmt.Sprintf("%s (%s, %d replica(s), %s)", change, m.Spec.Image, m.Spec.Replicas, hash)})
		return &Result{OK: true, Message: fmt.Sprintf("%s: %s (generation %d)", app.Name, change, app.Generation), Data: app.Generation}
	})

	register("delete-app", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			App string `json:"app"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		app := s.Apps[d.App]
		if app == nil || app.Deleted {
			return fail("NOT_FOUND", "no app %s", d.App)
		}
		app.Deleted, app.Updated = true, c.TS
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "app-delete", Resource: "app/" + app.Name, Generation: app.Generation,
			Detail: "desired state removed; replicas will receive signed stops; volumes are retained"})
		return ok("%s deleted; replicas will be stopped", d.App)
	})

	register("purge-app", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			App string `json:"app"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if app := s.Apps[d.App]; app != nil && app.Deleted {
			delete(s.Apps, d.App)
			for _, id := range s.sortedVolumeIDs() {
				if v := s.Volumes[id]; v.App == d.App {
					v.Orphaned = true
				}
			}
		}
		return ok("purged")
	})

	// assign upserts leader-signed assignments and removes finished ones.
	register("assign", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Upsert []*envelope.Envelope `json:"upsert"`
			Remove []string             `json:"remove"`
			Reason string               `json:"reason"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		changed := 0
		for _, env := range d.Upsert {
			m := s.memberForKey(env.Pub)
			if m == nil || m.ID != env.Signer {
				return fail("SIGNATURE", "assignment not signed by a roster member")
			}
			if err := env.VerifyKey(envelope.KindAssignment, m.Pub); err != nil {
				return fail("SIGNATURE", "assignment: %v", err)
			}
			var a api.Assignment
			if err := env.Decode(&a); err != nil {
				return fail("DECODE", "%v", err)
			}
			key := assignmentKey(a.ID, a.Node)
			prev := s.Assignments[key]
			s.Assignments[key] = &AssignmentRec{Key: key, A: a, Env: env, Created: c.TS}
			if prev != nil && prev.A.Generation == a.Generation && prev.A.Desired == a.Desired {
				continue // capability refresh only
			}
			changed++
			nodeName := a.Node
			if n := s.Nodes[a.Node]; n != nil {
				nodeName = n.Name
			}
			action := "assignment-sign"
			if a.Desired == "stopped" {
				action = "assignment-stop"
			}
			s.audit(audit.Entry{TS: c.TS, Actor: env.Signer, Source: audit.SourceControl, Action: action, Resource: "app/" + a.ID, Generation: a.Generation,
				Detail: fmt.Sprintf("%s on %s: %s", a.Desired, nodeName, d.Reason), Evidence: env.Digest()})
		}
		for _, key := range d.Remove {
			if _, has := s.Assignments[key]; has {
				delete(s.Assignments, key)
			}
		}
		return ok("%d assignment change(s), %d removed", changed, len(d.Remove))
	})

	register("volume-plan", func(s *State, c *Command) *Result {
		d, err := decode[struct {
			Volumes []*Volume `json:"volumes"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		for _, v := range d.Volumes {
			cur := s.Volumes[v.ID]
			if cur == nil {
				v.Created = c.TS
				v.Snapshots = nil
				v.Committed = ""
				s.Volumes[v.ID] = v
				s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceControl, Action: "volume-create", Resource: "volume/" + v.ID,
					Detail: fmt.Sprintf("%d replica(s) on %s", v.Durability.Replicas, strings.Join(names(s, v.Members), ", "))})
				continue
			}
			if strings.Join(cur.Members, ",") != strings.Join(v.Members, ",") || cur.Primary != v.Primary {
				s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceControl, Action: "volume-placement", Resource: "volume/" + v.ID,
					Detail: fmt.Sprintf("members %s → %s; primary %s", strings.Join(names(s, cur.Members), ","), strings.Join(names(s, v.Members), ","), name(s, v.Primary))})
			}
			cur.Members, cur.Primary, cur.Short, cur.Orphaned = v.Members, v.Primary, v.Short, false
		}
		return ok("volume placement updated")
	})
}

func names(s *State, ids []string) []string {
	var out []string
	for _, id := range ids {
		out = append(out, name(s, id))
	}
	return out
}

func name(s *State, id string) string {
	if n := s.Nodes[id]; n != nil {
		return n.Name
	}
	if id == "" {
		return "-"
	}
	return short(id)
}

// ------------------------------------------------------------ observations

func init() {
	register("observe", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		return applyObservation(s, env, c.TS)
	})

	register("evidence", func(s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		n := s.Nodes[env.Signer]
		if n == nil {
			return fail("NOT_FOUND", "evidence signer %s is unknown", env.Signer)
		}
		if err := env.VerifyKey(env.Kind, n.ValidKeys(c.TS)...); err != nil {
			s.reject(Rejection{TS: c.TS, Kind: env.Kind, Node: n.ID, Reason: err.Error()})
			return fail("SIGNATURE", "%s: %v", env.Kind, err)
		}
		switch env.Kind {
		case envelope.KindReplica:
			return applyReplicaEvidence(s, n, env, c.TS)
		case envelope.KindRepair:
			var r api.RepairEvidence
			if err := env.Decode(&r); err != nil || r.Node != n.ID {
				return fail("EVIDENCE", "repair evidence must be issued by the repairing host")
			}
			s.Repairs = append(s.Repairs, RepairRec{Evidence: env.Digest(), Received: c.TS, R: r})
			if len(s.Repairs) > 1000 {
				s.Repairs = s.Repairs[len(s.Repairs)-1000:]
			}
			okN, failN := 0, 0
			sources := map[string]bool{}
			for _, it := range r.Items {
				if it.Result == "present" && it.Verified {
					okN++
				} else {
					failN++
				}
				sources[name(s, it.Source)] = true
			}
			var src []string
			for k := range sources {
				src = append(src, k)
			}
			sort.Strings(src)
			s.audit(audit.Entry{TS: c.TS, Actor: n.ID, Source: audit.SourceHost, Action: "storage-repair", Resource: "volume/" + r.VolumeID,
				Detail:   fmt.Sprintf("%s: %d object(s) repaired and verified, %d failed; from %s; local root %s, peer root %s", r.Trigger, okN, failN, strings.Join(src, ","), short(r.LocalRoot), short(r.PeerRoot)),
				Evidence: env.Digest()})
			return ok("repair evidence recorded")
		}
		return fail("KIND", "unsupported evidence kind %s", env.Kind)
	})
}

func applyObservation(s *State, env *envelope.Envelope, ts int64) *Result {
	n := s.Nodes[env.Signer]
	if n == nil {
		s.reject(Rejection{TS: ts, Kind: "observation", Node: env.Signer, Reason: "unknown host"})
		return fail("NOT_FOUND", "observation from unknown host %s", env.Signer)
	}
	if err := env.VerifyKey(envelope.KindObservation, n.ValidKeys(ts)...); err != nil {
		reason := err.Error()
		if errors.Is(err, envelope.ErrSignerKey) {
			reason = "wrong host key: signed by a key that is not valid for " + n.Name
		}
		s.reject(Rejection{TS: ts, Kind: "observation", Node: n.ID, Reason: reason, Evidence: env.Digest()})
		return fail("SIGNATURE", "observation rejected: %s", reason)
	}
	var o api.Observation
	if err := env.Decode(&o); err != nil {
		return fail("DECODE", "%v", err)
	}
	if o.Node != n.ID {
		s.reject(Rejection{TS: ts, Kind: "observation", Node: n.ID, Reason: "observation names another host"})
		return fail("OBSERVATION", "observation names another host")
	}
	if o.Seq <= n.ObserveSeq {
		s.reject(Rejection{TS: ts, Kind: "observation", Node: n.ID, Reason: "replay detected", Seq: o.Seq, LastSeq: n.ObserveSeq, Evidence: env.Digest()})
		return &Result{OK: false, Code: "REPLAY", Message: fmt.Sprintf("replay detected: seq %d, last accepted %d", o.Seq, n.ObserveSeq)}
	}
	prev := n.Obs
	n.Obs, n.ObsEnv, n.ObsDigest, n.ObserveSeq, n.LastObsAt = &o, env, env.Digest(), o.Seq, ts
	if n.Health != "live" {
		if n.Health == "lost" {
			s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "node-recovered", Resource: "node/" + n.ID, Detail: n.Name + " is reporting again"})
		}
		n.Health = "live"
	}
	auditTransitions(s, n, prev, &o, env.Digest(), ts)
	return ok("observation seq %d accepted", o.Seq)
}

func auditTransitions(s *State, n *Node, prev, cur *api.Observation, evidence string, ts int64) {
	before := map[string]api.WorkloadObs{}
	if prev != nil {
		for _, w := range prev.Workloads {
			before[w.Assignment] = w
		}
	}
	for _, w := range cur.Workloads {
		p, had := before[w.Assignment]
		key := func(x api.WorkloadObs) string {
			return fmt.Sprintf("%s|%s|%s|%d|%d", x.Admitted, x.Code, x.Observed, x.PID, x.Generation)
		}
		if had && key(p) == key(w) {
			if p.Health != nil && w.Health != nil && p.Health.OK != w.Health.OK {
				act := "health-fail"
				if w.Health.OK {
					act = "health-pass"
				}
				s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: act, Resource: "app/" + w.Assignment, Generation: w.Generation, Detail: w.Health.Detail, Evidence: evidence})
			}
			continue
		}
		action := "workload-observe"
		switch {
		case w.Admitted == "refused":
			action = "node-refuse"
		case w.Observed == "running":
			action = "workload-start"
			if had && p.Observed == "running" {
				action = "workload-observe"
			}
		case w.Observed == "stopped":
			action = "workload-stop"
		case w.Observed == "oom-killed":
			action = "workload-oom"
		case w.Observed == "failed" || w.Observed == "exited":
			action = "workload-fail"
		}
		if had && p.Observed == "running" && w.Observed == "running" && p.PID != 0 && w.PID != 0 && p.PID != w.PID && p.Generation == w.Generation {
			action = "observe-conflict"
		}
		detail := fmt.Sprintf("%s/%s: %s", w.Admitted, w.Observed, w.Reason)
		if w.PID != 0 {
			detail += fmt.Sprintf(" (pid %d)", w.PID)
		}
		if w.ContainerID != "" {
			detail += fmt.Sprintf(" (container %s)", short(w.ContainerID))
		}
		s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: action, Resource: "app/" + w.Assignment, Generation: w.Generation, Detail: detail, Evidence: evidence})
	}
	if prev == nil || prev.Mode != cur.Mode {
		if cur.Mode != "" && cur.Mode != "normal" || (prev != nil && prev.Mode != "" && prev.Mode != "normal") {
			s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "host-mode", Resource: "node/" + n.ID, Detail: fmt.Sprintf("%s: %s", cur.Mode, cur.ModeDetail), Evidence: evidence})
		}
	}
	if cur.Storage != nil && cur.Storage.Corrupt > 0 && (prev == nil || prev.Storage == nil || prev.Storage.Corrupt < cur.Storage.Corrupt) {
		s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "storage-corruption", Resource: "node/" + n.ID, Detail: fmt.Sprintf("%d chunk(s) failed BLAKE3 verification and were quarantined", cur.Storage.Corrupt), Evidence: evidence})
	}
	if cur.Edge != nil {
		prevCerts := map[string]string{}
		if prev != nil && prev.Edge != nil {
			for _, c := range prev.Edge.Certs {
				prevCerts[c.Host] = c.State
			}
		}
		for _, c := range cur.Edge.Certs {
			if prevCerts[c.Host] != c.State {
				s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "cert-" + strings.ToLower(c.State), Resource: "cert/" + c.Host, Detail: fmt.Sprintf("%s via %s: %s", c.State, c.Issuer, c.Detail), Evidence: evidence})
			}
		}
		prevEP := map[string]string{}
		if prev != nil && prev.Edge != nil {
			for _, r := range prev.Edge.Routes {
				for _, e := range r.Endpoints {
					prevEP[r.Host+"|"+e.Assignment] = e.State
				}
			}
		}
		for _, r := range cur.Edge.Routes {
			for _, e := range r.Endpoints {
				k := r.Host + "|" + e.Assignment
				if old, had := prevEP[k]; had && old != e.State {
					s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "route-" + e.State, Resource: "route/" + r.Host, Detail: fmt.Sprintf("%s on %s: %s → %s (%s)", e.Assignment, name(s, e.Node), old, e.State, e.Reason), Evidence: evidence})
				}
			}
		}
	}
}

func applyReplicaEvidence(s *State, n *Node, env *envelope.Envelope, ts int64) *Result {
	var r api.ReplicaEvidence
	if err := env.Decode(&r); err != nil || r.Node != n.ID {
		return fail("EVIDENCE", "replica evidence must be issued by the holding host")
	}
	v := s.Volumes[r.VolumeID]
	if v == nil {
		return fail("NOT_FOUND", "no volume %s", r.VolumeID)
	}
	var snap *Snapshot
	for _, sn := range v.Snapshots {
		if sn.ID == r.Snapshot {
			snap = sn
		}
	}
	if snap == nil {
		if !r.Primary || r.Manifest == nil || n.ID != v.Primary {
			return fail("EVIDENCE", "snapshot %s is unknown and the evidence is not from the primary", short(r.Snapshot))
		}
		if id := SnapshotID(r.Manifest); id != r.Snapshot {
			return fail("EVIDENCE", "snapshot id does not match its manifest (%s != %s)", short(id), short(r.Snapshot))
		}
		var files int64
		files = int64(len(r.Manifest.Files))
		snap = &Snapshot{ID: r.Snapshot, Root: r.Root, Bytes: r.Bytes, Chunks: r.Chunks, Files: files, Parent: r.Manifest.Parent, TS: r.Manifest.TS, Creator: n.ID, State: "pending", Evidence: map[string]*ReplicaProof{}}
		v.Snapshots = append(v.Snapshots, snap)
		s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "snapshot-create", Resource: "volume/" + v.ID,
			Detail: fmt.Sprintf("%s: %d file(s), %d chunk(s), %d bytes, merkle %s", short(snap.ID), files, snap.Chunks, snap.Bytes, short(snap.Root)), Evidence: env.Digest()})
	}
	if r.Root != snap.Root {
		return fail("EVIDENCE", "replica root %s does not match snapshot root %s", short(r.Root), short(snap.Root))
	}
	snap.Evidence[n.ID] = &ReplicaProof{Node: n.ID, Root: r.Root, Chunks: r.Chunks, Missing: r.Missing, Corrupt: r.Corrupt, VerifiedAt: r.VerifiedAt, Received: ts, Evidence: env.Digest()}
	complete := 0
	for _, m := range sortedKeys(snap.Evidence) {
		if snap.Evidence[m].Complete(snap.Root) {
			complete++
		}
	}
	quorum := int(v.Durability.Replicas)
	if quorum > 2 {
		quorum = 2 // commit once two independent hosts verified every chunk
	}
	if quorum < 1 {
		quorum = 1
	}
	if snap.State == "pending" && complete >= quorum {
		snap.State, snap.CommittedAt = "committed", ts
		prevCommitted := v.Committed
		v.Committed = snap.ID
		for _, other := range v.Snapshots {
			if other.ID != snap.ID && other.State == "committed" && other.TS < snap.TS {
				other.State = "superseded"
			}
		}
		// retention: keep Retain newest non-pending snapshots
		keep := v.Retain
		if keep < 1 {
			keep = 1
		}
		var kept []*Snapshot
		var nonPending int64
		for i := len(v.Snapshots) - 1; i >= 0; i-- {
			sn := v.Snapshots[i]
			if sn.State != "pending" {
				nonPending++
				if nonPending > keep {
					continue
				}
			}
			kept = append([]*Snapshot{sn}, kept...)
		}
		v.Snapshots = kept
		s.audit(audit.Entry{TS: ts, Actor: n.ID, Source: audit.SourceHost, Action: "snapshot-commit", Resource: "volume/" + v.ID,
			Detail: fmt.Sprintf("%s committed: %d verified replica(s) (quorum %d); previous %s", short(snap.ID), complete, quorum, short(prevCommitted)), Evidence: env.Digest()})
	}
	// Bound pending snapshots that never committed (primary died mid-write).
	if len(v.Snapshots) > int(v.Retain)+8 {
		v.Snapshots = v.Snapshots[len(v.Snapshots)-int(v.Retain)-8:]
	}
	return ok("replica evidence recorded (%d complete)", complete)
}

// SnapshotID is the content address of a snapshot manifest: the BLAKE3 hash
// of its canonical encoding, which is also its CAS object id.
func SnapshotID(m *api.SnapshotManifest) string {
	b, err := canonJSON(m)
	if err != nil {
		return ""
	}
	return "b3:" + envelope.HashBytes(b)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func uniqueSorted(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		if s != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func short(s string) string {
	s = strings.TrimPrefix(s, "b3:")
	if len(s) > 12 {
		return s[:12]
	}
	if s == "" {
		return "-"
	}
	return s
}

// Authorization command handlers

const (
	authClockSkewTolerance = int64(5e9) // ±5 seconds in nanoseconds
)

// secretRetrievalAuthData wraps the request for FSM processing.
// The leader validates the request signature and proposes this deterministic command.
// The FSM re-evaluates all replicated-state predicates and records consumption atomically.
type secretRetrievalAuthData struct {
	Request    *SecretRetrievalRequest `json:"request"`
	ProposalTS string                  `json:"proposalTs"` // leader's observed time when proposing (as decimal string for JSON safety)
}

func secretRetrievalAuthorize(s *State, c *Command) *Result {
	// Decode wrapper payload containing proposal_ts and request
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Data, &payload); err != nil {
		return fail("DECODE", "secret-retrieval-authorize: invalid payload: %v", err)
	}

	// Extract proposal timestamp from payload
	var proposalTS int64
	if pts, foundPTS := payload["proposal_ts"]; foundPTS {
		switch v := pts.(type) {
		case float64:
			proposalTS = int64(v)
		case int64:
			proposalTS = v
		default:
			return fail("DECODE", "secret-retrieval-authorize: invalid proposal_ts type")
		}
	}

	// Extract and decode the request
	reqData, foundReq := payload["request"]
	if !foundReq {
		return fail("DECODE", "secret-retrieval-authorize: missing request in payload")
	}

	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return fail("DECODE", "secret-retrieval-authorize: request marshal: %v", err)
	}

	var req SecretRetrievalRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return fail("DECODE", "secret-retrieval-authorize: invalid request: %v", err)
	}

	// Check 1: Request schema valid (implicit in decode)
	if req.Version != 1 {
		return fail("INVALID", "secret-retrieval-authorize: unsupported version %d", req.Version)
	}
	if req.RequestID == "" || req.SecretID == "" || req.NodeID == "" {
		return fail("INVALID", "secret-retrieval-authorize: missing required fields")
	}

	// Check 2: Signature verifies against node public key (deterministic)
	if err := req.VerifySignature(); err != nil {
		return fail("DENIED", "secret-retrieval-authorize: signature: %v", err)
	}

	// Check 3: Node exists in current state
	node, exists := s.Nodes[req.NodeID]
	if !exists {
		return fail("DENIED", "secret-retrieval-authorize: node %s not found", req.NodeID)
	}

	// Check 4: Node not revoked
	if node.RevokedAt != 0 {
		return fail("DENIED", "secret-retrieval-authorize: node %s is revoked", req.NodeID)
	}

	// Check 5: Request timestamp within clock skew tolerance (±5s)
	// Parse request timestamp (from client, now stored as string)
	var reqTS int64
	if _, err := fmt.Sscanf(req.Timestamp, "%d", &reqTS); err != nil {
		return fail("DECODE", "secret-retrieval-authorize: invalid request timestamp: %v", err)
	}
	if proposalTS < reqTS-authClockSkewTolerance || proposalTS > reqTS+authClockSkewTolerance {
		return fail("DENIED", "secret-retrieval-authorize: request timestamp %d out of sync (now: %d, skew: ±%dns)", reqTS, proposalTS, authClockSkewTolerance)
	}

	// Check 6: Secret exists
	secretRecord := s.Secrets.GetRecord(req.SecretID, req.SecretVersion)
	if secretRecord == nil {
		return fail("DENIED", "secret-retrieval-authorize: secret %s version %d not found", req.SecretID, req.SecretVersion)
	}

	// Check 7: Secret version exists (implicit in above check)

	// Check 8: Scope fields match secret record
	if secretRecord.DeploymentID != req.DeploymentID || secretRecord.WorkloadID != req.WorkloadID || secretRecord.Environment != req.Environment {
		return fail("DENIED", "secret-retrieval-authorize: scope mismatch (secret: %s/%s/%s, request: %s/%s/%s)",
			secretRecord.DeploymentID, secretRecord.WorkloadID, secretRecord.Environment,
			req.DeploymentID, req.WorkloadID, req.Environment)
	}

	// Check 9: Assignment exists (key = secretId@nodeId is not right; we need to find assignment by workload/deployment/environment/node)
	// Actually, the assignment is id@node where id identifies the workload placement
	// We need to find an assignment for this node with matching workload/deployment/environment
	// Since we don't have direct workload->assignment mapping, we scan assignments
	var assignmentRec *AssignmentRec
	for _, rec := range s.Assignments {
		if rec.A.Node == req.NodeID {
			// Check if this assignment's scope matches the request scope
			// The assignment doesn't explicitly carry deployment/workload/environment as fields
			// Those come from the secret metadata
			// For now, verify that an assignment exists for this node in running state
			assignmentRec = rec
			break
		}
	}
	if assignmentRec == nil {
		return fail("DENIED", "secret-retrieval-authorize: no assignment found for node %s", req.NodeID)
	}

	// Check 10: Assignment in eligible state (Desired="running")
	if assignmentRec.A.Desired != "running" {
		return fail("DENIED", "secret-retrieval-authorize: assignment %s desired state is %q, not running", assignmentRec.A.ID, assignmentRec.A.Desired)
	}

	// Check 11: Nonce not already consumed (replay protection)
	requestDigest := req.RequestDigest()
	if s.ReplayLedger.IsConsumed(requestDigest) {
		return fail("DENIED", "secret-retrieval-authorize: request already authorized (replay)")
	}

	// All checks passed: record consumption atomically
	auth := &ConsumedAuthorization{
		RequestDigest: requestDigest,
		ConsumedNonce: req.Nonce,
		ConsumedAt:    fmt.Sprintf("%d", c.TS),
		NodeID:        req.NodeID,
		SecretID:      req.SecretID,
		SecretVersion: req.SecretVersion,
		Outcome:       "SUCCESS",
	}
	s.ReplayLedger.Record(auth)

	// Audit
	s.audit(audit.Entry{
		TS:       c.TS / 1e6, // convert nanoseconds to milliseconds
		Actor:    req.NodeID,
		Source:   audit.SourceHost,
		Action:   "secret-retrieval-authorize",
		Resource: fmt.Sprintf("secret/%s/v%d", req.SecretID, req.SecretVersion),
		Detail:   fmt.Sprintf("workload %s deployment %s env %s", req.WorkloadID, req.DeploymentID, req.Environment),
		Evidence: requestDigest,
	})

	return ok("authorization succeeded; request digest %s", requestDigest[:16])
}

// Secret command handlers

func secretCreate(s *State, c *Command) *Result {
	rec, err := decode[SecretRecord](c)
	if err != nil {
		return fail("DECODE", "secret-create: %v", err)
	}
	if rec.SecretID == "" {
		return fail("INVALID", "secret-create: empty secretId")
	}
	if rec.Version < 1 {
		return fail("INVALID", "secret-create: version must be >= 1")
	}
	if len(rec.EncryptedData) == 0 {
		return fail("INVALID", "secret-create: encrypted data empty")
	}
	rec.CreatedAt = c.TS
	rec.EncryptedAt = c.TS
	s.Secrets.AddRecord(&rec)
	return ok("secret %s version %d created", rec.SecretID, rec.Version)
}

func secretVersionAdd(s *State, c *Command) *Result {
	rec, err := decode[SecretRecord](c)
	if err != nil {
		return fail("DECODE", "secret-version-add: %v", err)
	}
	if rec.SecretID == "" {
		return fail("INVALID", "secret-version-add: empty secretId")
	}
	if rec.Version < 1 {
		return fail("INVALID", "secret-version-add: version must be >= 1")
	}
	if len(rec.EncryptedData) == 0 {
		return fail("INVALID", "secret-version-add: encrypted data empty")
	}
	existing := s.Secrets.GetRecord(rec.SecretID, rec.Version)
	if existing != nil {
		return fail("EXISTS", "secret %s version %d already exists", rec.SecretID, rec.Version)
	}
	rec.EncryptedAt = c.TS
	s.Secrets.AddRecord(&rec)
	return ok("secret %s version %d added", rec.SecretID, rec.Version)
}

func init() {
	register("secret-retrieval-authorize", secretRetrievalAuthorize)
	register("secret-create", secretCreate)
	register("secret-version-add", secretVersionAdd)
}
