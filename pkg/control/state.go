package control

import (
	"fmt"
	"sort"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/envelope"
)

// State is the replicated control-plane document. Everything in it is
// desired state or committed evidence; it is mutated only by FSM.Apply.
type State struct {
	Cluster      string                        `json:"cluster"`
	Root         string                        `json:"root"`
	RootCACert   string                        `json:"rootCaCert"`
	Roster       *envelope.Envelope            `json:"roster"`
	RosterBody   api.Roster                    `json:"rosterBody"`
	PastRosters  []*envelope.Envelope          `json:"pastRosters"` // root-signed rosters that were replaced (historical checkpoint keys)
	RootRotation *envelope.Envelope            `json:"rootRotation"`
	Frozen       bool                          `json:"frozen"`
	FrozenAt     int64                         `json:"frozenAt"`
	Nodes        map[string]*Node              `json:"nodes"`
	Apps         map[string]*App               `json:"apps"`
	Assignments  map[string]*AssignmentRec     `json:"assignments"`
	Volumes      map[string]*Volume            `json:"volumes"`
	Artifacts    map[string]*Artifact          `json:"artifacts"`
	Attestations map[string]*envelope.Envelope `json:"attestations"`
	Publishers   []string                      `json:"publishers"`
	Invites      map[string]*Invite            `json:"invites"`
	Audit        audit.Ledger                  `json:"audit"`
	Checkpoints  []*envelope.Envelope          `json:"checkpoints"`
	Repairs      []RepairRec                   `json:"repairs"`
	Federation   Federation                    `json:"federation"`
	ChaosReports []ChaosRec                    `json:"chaosReports"`
	MemberMesh   map[string]string             `json:"memberMesh"` // member id -> mesh ip
	MemberBind   map[string]*envelope.Envelope `json:"memberBind"`
	MeshNext     int64                         `json:"meshNext"`
	LocalCACert  string                        `json:"localCaCert"`
	LocalCAKey   string                        `json:"localCaKey"` // secret: never exported without --include-secrets
	Index        int64                         `json:"index"`
	IndexBase    int64                         `json:"indexBase"` // added to raft indexes after a restore so bundles never go backwards
	Rejections   []Rejection                   `json:"rejections"`
	Mirror       MirrorState                   `json:"mirror"`
}

// Node is a host as the control plane knows it.
type Node struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Status     string             `json:"status"` // pending | ready | draining | revoked
	Health     string             `json:"health"` // live | lost | unknown
	Keys       []api.KeyRecord    `json:"keys"`
	Enroll     api.Enroll         `json:"enroll"`
	EnrollEnv  *envelope.Envelope `json:"enrollEnv"`
	MeshIP     string             `json:"meshIp"`
	Binding    *envelope.Envelope `json:"binding"`
	Roles      []string           `json:"roles"`
	JoinedAt   int64              `json:"joinedAt"`
	ApprovedAt int64              `json:"approvedAt"`
	RevokedAt  int64              `json:"revokedAt"`
	ObserveSeq int64              `json:"observeSeq"`
	LastObsAt  int64              `json:"lastObsAt"` // control-plane receive time
	Obs        *api.Observation   `json:"obs"`
	ObsEnv     *envelope.Envelope `json:"obsEnv"`
	ObsDigest  string             `json:"obsDigest"`
	FirstSeen  int64              `json:"firstSeen"`
}

// ValidKeys returns wire keys valid at ts.
func (n *Node) ValidKeys(ts int64) []string {
	var out []string
	for _, k := range n.Keys {
		if k.Revoked {
			continue
		}
		if k.From != 0 && ts < k.From {
			continue
		}
		if k.Until != 0 && ts >= k.Until {
			continue
		}
		out = append(out, k.Pub)
	}
	return out
}

// App is a desired application.
type App struct {
	Name         string        `json:"name"`
	Manifest     api.Manifest  `json:"manifest"`
	Hash         string        `json:"hash"`
	TemplateHash string        `json:"templateHash"`
	Generation   int64         `json:"generation"`
	Created      int64         `json:"created"`
	Updated      int64         `json:"updated"`
	Deleted      bool          `json:"deleted"`
	History      []AppRevision `json:"history"`
	Federation   *FedPlacement `json:"federation"` // set when placed on behalf of a peer cluster
}

type AppRevision struct {
	Generation int64  `json:"generation"`
	Hash       string `json:"hash"`
	TS         int64  `json:"ts"`
	Actor      string `json:"actor"`
	Change     string `json:"change"`
}

// AssignmentRec is a signed assignment. Key = id@node.
type AssignmentRec struct {
	Key     string             `json:"key"`
	A       api.Assignment     `json:"a"`
	Env     *envelope.Envelope `json:"env"`
	Created int64              `json:"created"`
}

func assignmentKey(id, node string) string { return id + "@" + node }

// Volume is one replicated per-replica volume.
type Volume struct {
	ID         string         `json:"id"`
	App        string         `json:"app"`
	Name       string         `json:"name"`
	Replica    int64          `json:"replica"`
	SizeBytes  int64          `json:"sizeBytes"`
	Durability api.Durability `json:"durability"`
	Retain     int64          `json:"retain"`
	Tiers      []string       `json:"tiers"`
	Primary    string         `json:"primary"`
	Members    []string       `json:"members"`
	Short      int64          `json:"short"`
	Snapshots  []*Snapshot    `json:"snapshots"`
	Committed  string         `json:"committed"`
	Created    int64          `json:"created"`
	Orphaned   bool           `json:"orphaned"`
}

// Snapshot is one volume snapshot and the replica evidence for it.
type Snapshot struct {
	ID          string                   `json:"id"`
	Root        string                   `json:"root"`
	Bytes       int64                    `json:"bytes"`
	Chunks      int64                    `json:"chunks"`
	Files       int64                    `json:"files"`
	Parent      string                   `json:"parent"`
	TS          int64                    `json:"ts"`
	Creator     string                   `json:"creator"`
	State       string                   `json:"state"` // pending | committed | superseded
	Evidence    map[string]*ReplicaProof `json:"evidence"`
	CommittedAt int64                    `json:"committedAt"`
}

type ReplicaProof struct {
	Node       string `json:"node"`
	Root       string `json:"root"`
	Chunks     int64  `json:"chunks"`
	Missing    int64  `json:"missing"`
	Corrupt    int64  `json:"corrupt"`
	VerifiedAt int64  `json:"verifiedAt"`
	Received   int64  `json:"received"`
	Evidence   string `json:"evidence"`
}

// Complete reports whether the proof shows a full, verified copy.
func (p *ReplicaProof) Complete(root string) bool {
	return p != nil && p.Missing == 0 && p.Corrupt == 0 && p.Root == root
}

// Artifact is a content-addressed executable known to the cluster.
type Artifact struct {
	Digest   string   `json:"digest"`
	Name     string   `json:"name"`
	Bytes    int64    `json:"bytes"`
	Chunks   int64    `json:"chunks"`
	Manifest string   `json:"manifest"` // CAS id of the chunk list
	Holders  []string `json:"holders"`
	Uploaded int64    `json:"uploaded"`
}

type Invite struct {
	Nonce   string   `json:"nonce"`
	Expires int64    `json:"expires"`
	Roles   []string `json:"roles"`
	Note    string   `json:"note"`
	Auto    bool     `json:"auto"`
	Used    string   `json:"used"` // node id that consumed it
	Created int64    `json:"created"`
	Revoked int64    `json:"revoked,omitempty"` // when the owner withdrew it; a revoked invite admits nobody
}

type RepairRec struct {
	Evidence string             `json:"evidence"`
	Received int64              `json:"received"`
	R        api.RepairEvidence `json:"r"`
}

type ChaosRec struct {
	ID       string             `json:"id"`
	Received int64              `json:"received"`
	Env      *envelope.Envelope `json:"env"`
}

// Rejection records a refused protocol message (bounded ring).
type Rejection struct {
	TS       int64  `json:"ts"`
	Kind     string `json:"kind"`
	Node     string `json:"node"`
	Reason   string `json:"reason"`
	Seq      int64  `json:"seq"`
	LastSeq  int64  `json:"lastSeq"`
	Evidence string `json:"evidence"`
}

// MirrorState reports the optional Postgres evidence mirror.
type MirrorState struct {
	Enabled bool `json:"enabled"`
}

func newState() *State {
	return &State{
		Nodes:        map[string]*Node{},
		Apps:         map[string]*App{},
		Assignments:  map[string]*AssignmentRec{},
		Volumes:      map[string]*Volume{},
		Artifacts:    map[string]*Artifact{},
		Attestations: map[string]*envelope.Envelope{},
		Invites:      map[string]*Invite{},
		MemberMesh:   map[string]string{},
		MemberBind:   map[string]*envelope.Envelope{},
		Federation:   newFederation(),
	}
}

// ensure fills nil maps after decoding an older snapshot.
func (s *State) ensure() {
	if s.Nodes == nil {
		s.Nodes = map[string]*Node{}
	}
	if s.Apps == nil {
		s.Apps = map[string]*App{}
	}
	if s.Assignments == nil {
		s.Assignments = map[string]*AssignmentRec{}
	}
	if s.Volumes == nil {
		s.Volumes = map[string]*Volume{}
	}
	if s.Artifacts == nil {
		s.Artifacts = map[string]*Artifact{}
	}
	if s.Attestations == nil {
		s.Attestations = map[string]*envelope.Envelope{}
	}
	if s.Invites == nil {
		s.Invites = map[string]*Invite{}
	}
	if s.MemberMesh == nil {
		s.MemberMesh = map[string]string{}
	}
	if s.MemberBind == nil {
		s.MemberBind = map[string]*envelope.Envelope{}
	}
	s.Federation.ensure()
}

// member returns the roster member with id.
func (s *State) member(id string) *api.Member {
	for i := range s.RosterBody.Members {
		if s.RosterBody.Members[i].ID == id {
			return &s.RosterBody.Members[i]
		}
	}
	return nil
}

// CheckpointKeys returns every key a member id held in any root-signed
// roster this cluster has had (current or replaced). Historical checkpoints
// stay verifiable after membership changes, rotations and restores.
func (s *State) CheckpointKeys(id string) []string {
	var out []string
	for _, env := range append([]*envelope.Envelope{s.Roster}, s.PastRosters...) {
		if env == nil {
			continue
		}
		var r api.Roster
		if env.Decode(&r) != nil {
			continue
		}
		for _, m := range r.Members {
			if m.ID == id {
				out = append(out, m.Pub)
			}
		}
	}
	return out
}

func (s *State) retireRoster() {
	if s.Roster != nil {
		s.PastRosters = append(s.PastRosters, s.Roster)
	}
}

func (s *State) memberKeys() []string {
	var out []string
	for _, m := range s.RosterBody.Members {
		out = append(out, m.Pub)
	}
	return out
}

func (s *State) memberForKey(pub string) *api.Member {
	for i := range s.RosterBody.Members {
		if s.RosterBody.Members[i].Pub == pub {
			return &s.RosterBody.Members[i]
		}
	}
	return nil
}

// sortedNodeIDs gives a deterministic iteration order.
func (s *State) sortedNodeIDs() []string {
	ids := make([]string, 0, len(s.Nodes))
	for id := range s.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *State) sortedAppNames() []string {
	names := make([]string, 0, len(s.Apps))
	for n := range s.Apps {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (s *State) sortedAssignmentKeys() []string {
	keys := make([]string, 0, len(s.Assignments))
	for k := range s.Assignments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *State) sortedVolumeIDs() []string {
	ids := make([]string, 0, len(s.Volumes))
	for id := range s.Volumes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// assignmentsFor returns records for a node in key order.
func (s *State) assignmentsFor(node string) []*AssignmentRec {
	var out []*AssignmentRec
	for _, k := range s.sortedAssignmentKeys() {
		if r := s.Assignments[k]; r.A.Node == node {
			out = append(out, r)
		}
	}
	return out
}

// workloadObs returns the committed observation of an assignment on a node.
func (s *State) workloadObs(node, id string) *api.WorkloadObs {
	n := s.Nodes[node]
	if n == nil || n.Obs == nil {
		return nil
	}
	for i := range n.Obs.Workloads {
		if n.Obs.Workloads[i].Assignment == id {
			return &n.Obs.Workloads[i]
		}
	}
	return nil
}

func (s *State) audit(e audit.Entry) audit.Entry { return s.Audit.Append(e) }

func (s *State) reject(r Rejection) {
	s.Rejections = append(s.Rejections, r)
	if len(s.Rejections) > 500 {
		s.Rejections = s.Rejections[len(s.Rejections)-500:]
	}
}

// meshIP allocates addresses from 10.77.0.0/16 (hosts from .0.1 upward,
// control-plane members from .255.1 upward).
func meshIPFor(n int64) string { return fmt.Sprintf("10.77.%d.%d", n/254, n%254+1) }
