package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/envelope"
)

// Export is the vendor-independent exit format (envelope kind "export").
// Secrets are never included: private keys never leave their owners, and
// the local CA key stays in the replicated log.
type Export struct {
	Format       string               `json:"format"`
	Protocol     string               `json:"protocol"`
	Cluster      string               `json:"cluster"`
	Root         string               `json:"root"`
	Roster       *envelope.Envelope   `json:"roster"`
	PastRosters  []*envelope.Envelope `json:"pastRosters"`
	RootRotation *envelope.Envelope   `json:"rootRotation"`
	RootCACert   string               `json:"rootCaCert"`
	LocalCACert  string               `json:"localCaCert"`
	Manifests    []api.Manifest       `json:"manifests"`
	Hosts        []ExportHost         `json:"hosts"`
	Volumes      []*Volume            `json:"volumes"`
	Artifacts    []*Artifact          `json:"artifacts"`
	Publishers   []string             `json:"publishers"`
	Attestations []*envelope.Envelope `json:"attestations"`
	Audit        []audit.Entry        `json:"audit"`
	Checkpoints  []*envelope.Envelope `json:"checkpoints"`
	Federation   Federation           `json:"federation"`
	StateIndex   int64                `json:"stateIndex"`
	Excluded     []string             `json:"excluded"`
	TS           int64                `json:"ts"`
}

type ExportHost struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Status  string             `json:"status"`
	Keys    []api.KeyRecord    `json:"keys"`
	Enroll  *envelope.Envelope `json:"enroll"`
	Binding *envelope.Envelope `json:"binding"`
	Policy  api.PolicySummary  `json:"policy"`
}

func (s *Server) handleExport(w http.ResponseWriter, _ *http.Request, _ *authz) {
	var ex Export
	s.fsm.Read(func(st *State) {
		ex = Export{Format: "dh-export/v1", Protocol: api.ProtocolVersion, Cluster: st.Cluster, Root: st.Root, Roster: st.Roster, PastRosters: st.PastRosters, RootRotation: st.RootRotation, RootCACert: st.RootCACert,
			LocalCACert: st.LocalCACert, Publishers: st.Publishers, Audit: st.Audit.Entries, Checkpoints: st.Checkpoints, Federation: st.Federation,
			StateIndex: st.Index, TS: nowMs(),
			Excluded: []string{"host private keys (never leave hosts)", "control-plane member private keys", "root private key (held by the operator)", "local CA private key", "volume chunk data (use dh volume export)"}}
		for _, n := range st.sortedAppNames() {
			if a := st.Apps[n]; !a.Deleted && a.Federation == nil {
				ex.Manifests = append(ex.Manifests, a.Manifest)
			}
		}
		for _, id := range st.sortedNodeIDs() {
			n := st.Nodes[id]
			ex.Hosts = append(ex.Hosts, ExportHost{ID: n.ID, Name: n.Name, Status: n.Status, Keys: n.Keys, Enroll: n.EnrollEnv, Binding: n.Binding, Policy: n.Enroll.Policy})
		}
		for _, id := range st.sortedVolumeIDs() {
			ex.Volumes = append(ex.Volumes, st.Volumes[id])
		}
		for _, d := range sortedKeys(st.Artifacts) {
			ex.Artifacts = append(ex.Artifacts, st.Artifacts[d])
		}
		for _, d := range sortedKeys(st.Attestations) {
			ex.Attestations = append(ex.Attestations, st.Attestations[d])
		}
	})
	env, err := envelope.Sign(s.id, "", envelope.KindExport, ex)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, env)
}

// VerifyExport checks an export offline: member signature against the
// export's own root-signed roster, the audit chain and its checkpoints.
func VerifyExport(env *envelope.Envelope) (*Export, error) {
	var ex Export
	if err := env.Decode(&ex); err != nil {
		return nil, err
	}
	r, err := verifyRoster(ex.Roster, ex.Root)
	if err != nil {
		return nil, fmt.Errorf("export roster: %w", err)
	}
	var signerOK bool
	for _, m := range r.Members {
		if m.ID == env.Signer && env.VerifyKey(envelope.KindExport, m.Pub) == nil {
			signerOK = true
		}
	}
	if !signerOK {
		return nil, errors.New("export is not signed by a member of its roster")
	}
	if b := audit.Verify(ex.Audit, 0, audit.Genesis); b != nil {
		return nil, b
	}
	keys := HistoricalMemberKeys(ex.Root, ex.RootRotation, ex.Roster, ex.PastRosters)
	if b := audit.VerifyCheckpoints(ex.Audit, ex.Checkpoints, func(s string) []string { return keys[s] }); b != nil {
		return nil, b
	}
	return &ex, nil
}

// HistoricalMemberKeys maps member ids to every key they held in a roster
// signed by the current root or the root it rotated from. Rosters that do
// not verify contribute nothing.
func HistoricalMemberKeys(root string, rotation, current *envelope.Envelope, past []*envelope.Envelope) map[string][]string {
	roots := []string{root}
	if rotation != nil {
		var rot api.Rotation
		if rotation.VerifyKey(envelope.KindRotation, rotation.Pub) == nil && rotation.Decode(&rot) == nil && rot.NewPub == root {
			roots = append(roots, rot.OldPub)
		}
	}
	out := map[string][]string{}
	for _, env := range append([]*envelope.Envelope{current}, past...) {
		if env == nil || env.VerifyKey(envelope.KindRoster, roots...) != nil {
			continue
		}
		var r api.Roster
		if env.Decode(&r) != nil {
			continue
		}
		for _, m := range r.Members {
			out[m.ID] = append(out[m.ID], m.Pub)
		}
	}
	return out
}

// VerifyBackup checks a control-plane backup offline and returns its state.
func VerifyBackup(env *envelope.Envelope) (*Backup, *State, error) {
	var b Backup
	if err := env.Decode(&b); err != nil {
		return nil, nil, err
	}
	if "b3:"+envelope.HashBytes(b.State) != b.StateHash {
		return nil, nil, errors.New("backup state hash mismatch")
	}
	if !canon.IsCanonical(b.State) {
		return nil, nil, errors.New("backup state is not canonical")
	}
	st := newState()
	if err := json.Unmarshal(b.State, st); err != nil {
		return nil, nil, err
	}
	st.ensure()
	m := st.member(env.Signer)
	if m == nil || env.VerifyKey(envelope.KindBackup, m.Pub) != nil {
		return nil, nil, errors.New("backup is not signed by a member of the backed-up roster")
	}
	if _, err := verifyRoster(st.Roster, st.Root); err != nil {
		return nil, nil, fmt.Errorf("backup roster: %w", err)
	}
	if br := audit.Verify(st.Audit.Entries, 0, audit.Genesis); br != nil {
		return nil, nil, br
	}
	return &b, st, nil
}

func init() {
	// restore replaces a freshly initialized state with a verified backup of
	// the same root. The new roster is kept, and the state index continues
	// above the backup's so hosts never see a rollback.
	register("restore", func(f *FSM, s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		b, old, err := VerifyBackup(env)
		if err != nil {
			return fail("BACKUP", "%v", err)
		}
		if old.Root != s.Root && !(s.RootRotation != nil) {
			return fail("BACKUP", "backup root %s does not match this cluster's root", short(old.Root))
		}
		if len(s.Apps) > 0 || len(s.Nodes) > 0 {
			return fail("BACKUP", "restore is only allowed into an empty, freshly bootstrapped cluster")
		}
		roster, rosterBody, memberMesh, memberBind := s.Roster, s.RosterBody, s.MemberMesh, s.MemberBind
		*s = *old
		// The backed-up roster becomes history: its members signed the
		// restored checkpoints. The new roster governs from here on.
		s.retireRoster()
		s.Roster, s.RosterBody = roster, rosterBody
		for id, ip := range memberMesh {
			s.MemberMesh[id] = ip
		}
		for id, bind := range memberBind {
			s.MemberBind[id] = bind
		}
		s.IndexBase = b.Index + 1
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: "cp-restore", Resource: "cluster/" + s.Cluster,
			Detail: fmt.Sprintf("restored from backup at index %d signed by %s (state %s)", b.Index, short(b.Member), short(b.StateHash)), Evidence: env.Digest()})
		return ok("restored %s at index %d", s.Cluster, b.Index)
	})
}
