package control

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/raft"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/manifest"
)

// View is the console projection. Every value is derived from replicated
// state or signed evidence; values carry their truth basis.
type View struct {
	GeneratedAt int64            `json:"generatedAt"`
	ServedBy    ServedBy         `json:"servedBy"`
	Cluster     ClusterView      `json:"cluster"`
	Overview    OverviewView     `json:"overview"`
	Nodes       []NodeView       `json:"nodes"`
	Apps        []AppView        `json:"apps"`
	Volumes     []VolumeView     `json:"volumes"`
	Artifacts   []ArtifactView   `json:"artifacts"`
	Edge        EdgeView         `json:"edge"`
	Audit       AuditView        `json:"audit"`
	Rejections  []Rejection      `json:"rejections"`
	Repairs     []RepairRec      `json:"repairs"`
	Federation  FederationView   `json:"federation"`
	Chaos       []ChaosSummary   `json:"chaos"`
	Diagnostics []DiagnosticView `json:"diagnostics"`
	Milestones  []MilestoneView  `json:"milestones"`
}

type ServedBy struct {
	Member      string `json:"member"`
	State       string `json:"state"`
	Leader      string `json:"leader"`
	Index       int64  `json:"index"`
	LastContact int64  `json:"lastContact"`
	Stale       bool   `json:"stale"` // follower view may lag the leader
}

type MemberView struct {
	ID       string `json:"id"`
	APIAddr  string `json:"apiAddr"`
	RaftAddr string `json:"raftAddr"`
	MeshIP   string `json:"meshIp"`
	Suffrage string `json:"suffrage"`
	InRaft   bool   `json:"inRaft"`
	Leader   bool   `json:"leader"`
	Binding  bool   `json:"binding"`
}

type ClusterView struct {
	Name          string       `json:"name"`
	Root          string       `json:"root"`
	RosterVersion int64        `json:"rosterVersion"`
	Members       []MemberView `json:"members"`
	Frozen        bool         `json:"frozen"`
	FrozenAt      int64        `json:"frozenAt"`
	Quorum        string       `json:"quorum"` // OBSERVED from raft on the serving member
	Mirror        MirrorStatus `json:"mirror"`
	Durability    string       `json:"durability"` // RAFT COMMITTED | DB COMMITTED | JOURNAL ONLY
	LocalCA       bool         `json:"localCa"`
	Publishers    []string     `json:"publishers"`
	HA            string       `json:"ha"`
}

type OverviewView struct {
	Hosts           map[string]int `json:"hosts"`
	HostsFresh      int            `json:"hostsFresh"`
	Desired         int            `json:"desired"`
	Admitted        int            `json:"admitted"`
	Observed        int            `json:"observed"`
	Refused         int            `json:"refused"`
	Drift           int            `json:"drift"`
	Apps            int            `json:"apps"`
	Volumes         int            `json:"volumes"`
	VolumesDegraded int            `json:"volumesDegraded"`
	Incidents       []string       `json:"incidents"`
	EvidenceAgeMs   int64          `json:"evidenceAgeMs"` // oldest fresh host observation
}

type NodeView struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Health       string            `json:"health"`
	Identity     string            `json:"identity"` // ACTIVE | REVOKED
	NewAdmission string            `json:"newAdmission"`
	Existing     string            `json:"existing"`
	Keys         []api.KeyRecord   `json:"keys"`
	Roles        []string          `json:"roles"`
	MeshIP       string            `json:"meshIp"`
	Region       string            `json:"region"`
	Zone         string            `json:"zone"`
	Host         string            `json:"host"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	Tiers        []string          `json:"tiers"`
	CPUMilli     int64             `json:"cpuMilli"`
	MemBytes     int64             `json:"memBytes"`
	Policy       api.PolicySummary `json:"policy"`
	PolicyBasis  api.Basis         `json:"policyBasis"`
	JoinedAt     int64             `json:"joinedAt"`
	ApprovedAt   int64             `json:"approvedAt"`
	RevokedAt    int64             `json:"revokedAt"`
	LastObs      ObsFreshness      `json:"lastObs"`
	Facts        *api.Facts        `json:"facts"`
	Mesh         *api.MeshObs      `json:"mesh"`
	Storage      *api.StorageObs   `json:"storage"`
	Edge         *api.EdgeObs      `json:"edge"`
	Ledger       api.LedgerHead    `json:"ledger"`
	Mode         string            `json:"mode"`
	ModeDetail   string            `json:"modeDetail"`
	Workloads    int               `json:"workloads"`
	WGBinding    *api.WGBinding    `json:"wgBinding"`
	Evidence     string            `json:"evidence"`
}

type ObsFreshness struct {
	Seq        int64  `json:"seq"`
	HostTS     int64  `json:"hostTs"`
	ReceivedAt int64  `json:"receivedAt"`
	AgeMs      int64  `json:"ageMs"`
	Freshness  string `json:"freshness"` // FRESH | STALE | NONE
	Committed  int64  `json:"committedSeq"`
	Buffered   bool   `json:"buffered"`
	Source     string `json:"source"` // leader-cache | replicated-log
}

type ReplicaView struct {
	Replica     int64              `json:"replica"`
	Assignment  string             `json:"assignment"`
	Node        string             `json:"node"`
	NodeName    string             `json:"nodeName"`
	Desired     string             `json:"desired"`
	DesiredGen  int64              `json:"desiredGen"`
	Admitted    string             `json:"admitted"` // ADMITTED | REFUSED | HELD | PENDING | UNKNOWN
	AdmittedGen int64              `json:"admittedGen"`
	Observed    string             `json:"observed"` // RUNNING | STOPPED | FAILED | OOM-KILLED | UNKNOWN ...
	ObservedGen int64              `json:"observedGen"`
	Status      string             `json:"status"`
	Code        string             `json:"code"`
	Reason      string             `json:"reason"`
	Checks      []api.Check        `json:"checks"`
	PID         int64              `json:"pid"`
	ContainerID string             `json:"containerId"`
	MeshPort    int64              `json:"meshPort"`
	Health      *api.HealthObs     `json:"health"`
	Restarts    int64              `json:"restarts"`
	StartedAt   int64              `json:"startedAt"`
	Freshness   string             `json:"freshness"`
	ObservedAt  int64              `json:"observedAt"`
	Evidence    string             `json:"evidence"`
	Federation  *api.FederationRef `json:"federation"`
	Volumes     []api.VolumeObs    `json:"volumes"`
}

type AppView struct {
	Name       string        `json:"name"`
	Generation int64         `json:"generation"`
	Hash       string        `json:"hash"`
	Image      string        `json:"image"`
	Runtime    string        `json:"runtime"`
	Replicas   int64         `json:"replicas"`
	Deleted    bool          `json:"deleted"`
	Desired    int           `json:"desired"`
	Admitted   int           `json:"admitted"`
	Observed   int           `json:"observed"`
	Drift      int           `json:"drift"`
	Rows       []ReplicaView `json:"rows"`
	Plan       *planEntry    `json:"plan"`
	History    []AppRevision `json:"history"`
	Manifest   api.Manifest  `json:"manifest"`
	Ingress    []api.Ingress `json:"ingress"`
	Federation *FedPlacement `json:"federation"`
	Outbound   *FedOutbound  `json:"outbound"`
}

type VolumeView struct {
	*Volume
	CommittedRef *Snapshot `json:"committedRef"`
	Verified     int       `json:"verified"`
	State        string    `json:"state"` // HEALTHY | DEGRADED | UNCOMMITTED | NO EVIDENCE
	Detail       string    `json:"detail"`
	MemberNames  []string  `json:"memberNames"`
}

type ArtifactView struct {
	Artifact
	HolderNames []string `json:"holderNames"`
	Attested    bool     `json:"attested"`
	Local       bool     `json:"local"`
}

type EdgeView struct {
	Services []api.Service  `json:"services"`
	Edges    []EdgeHostView `json:"edges"`
}

type EdgeHostView struct {
	Node      string       `json:"node"`
	Name      string       `json:"name"`
	Obs       *api.EdgeObs `json:"obs"`
	Freshness string       `json:"freshness"`
}

type AuditView struct {
	Head           int64             `json:"head"`
	HeadHash       string            `json:"headHash"`
	Entries        []audit.Entry     `json:"entries"`
	Verification   AuditVerification `json:"verification"`
	Checkpoints    int               `json:"checkpoints"`
	LastCheckpoint int64             `json:"lastCheckpoint"`
}

type ChaosSummary struct {
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	Verdict  string `json:"verdict"`
	Received int64  `json:"received"`
	Signer   string `json:"signer"`
	Evidence string `json:"evidence"`
	Report   any    `json:"report"`
}

type DiagnosticView struct {
	Subject string    `json:"subject"`
	Item    string    `json:"item"`
	Value   string    `json:"value"`
	Basis   api.Basis `json:"basis"`
	Detail  string    `json:"detail"`
}

type MilestoneView struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	State    string    `json:"state"` // VERIFIED | OPERATIONAL | PARTIAL | NOT OBSERVED
	Basis    api.Basis `json:"basis"`
	Evidence []string  `json:"evidence"`
	Gaps     []string  `json:"gaps"`
}

const freshWindow = 10 * time.Second

// handleView serves the console projection. A follower only sees the
// observations the leader has committed so far, so it relays the leader's
// view when the leader answers quickly, and otherwise serves its own view
// (ServedBy.State says follower, and the console flags it).
func (s *Server) handleView(w http.ResponseWriter, r *http.Request, _ *authz) {
	if rn := s.raft(); rn != nil && rn.r.State() != raft.Leader && r.Header.Get("X-DH-Forwarded") == "" {
		if body, ok := s.leaderGet(r, "/api/v1/view", 2*time.Second); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-DH-Relayed-By", s.id.ID)
			_, _ = w.Write(body)
			return
		}
	}
	writeJSON(w, 200, s.buildView())
}

func (s *Server) buildView() *View {
	v := &View{GeneratedAt: nowMs()}
	rn := s.raft()
	v.ServedBy.Member = s.id.ID
	raftServers := map[string]string{}
	if rn != nil {
		v.ServedBy.State = strings.ToLower(rn.r.State().String())
		_, lid := rn.r.LeaderWithID()
		v.ServedBy.Leader = string(lid)
		v.ServedBy.LastContact = unixMilliOrZero(rn.r.LastContact())
		v.ServedBy.Stale = rn.r.State() != raft.Leader
		if cfg := rn.r.GetConfiguration(); cfg.Error() == nil {
			for _, sv := range cfg.Configuration().Servers {
				raftServers[string(sv.ID)] = sv.Suffrage.String()
			}
		}
	}
	verification := s.verifyLedger()
	now := nowMs()
	s.fsm.Read(func(st *State) {
		v.ServedBy.Index = st.Index
		c := &v.Cluster
		c.Name, c.Root, c.RosterVersion, c.Frozen, c.FrozenAt = st.Cluster, st.Root, st.RosterBody.Version, st.Frozen, st.FrozenAt
		c.LocalCA, c.Publishers = st.LocalCACert != "", st.Publishers
		voters := 0
		for _, m := range st.RosterBody.Members {
			suf, in := raftServers[m.ID]
			if suf == "Voter" {
				voters++
			}
			c.Members = append(c.Members, MemberView{ID: m.ID, APIAddr: m.APIAddr, RaftAddr: m.RaftAddr, MeshIP: st.MemberMesh[m.ID], Suffrage: suf, InRaft: in, Leader: m.ID == v.ServedBy.Leader, Binding: st.MemberBind[m.ID] != nil})
		}
		switch {
		case rn == nil:
			c.Quorum = "NOT BOOTSTRAPPED"
		case v.ServedBy.Leader == "":
			c.Quorum = "NO LEADER"
		default:
			c.Quorum = fmt.Sprintf("LEADER %s · %d voter(s)", short(v.ServedBy.Leader), voters)
		}
		c.HA = "SINGLE MEMBER (not highly available)"
		if voters >= 3 {
			c.HA = fmt.Sprintf("RAFT %d VOTERS (tolerates %d failure(s))", voters, (voters-1)/2)
		} else if voters == 2 {
			c.HA = "RAFT 2 VOTERS (no failure tolerance)"
		}
		c.Mirror = s.mirror.status()
		head, _ := st.Audit.Head()
		switch {
		case !c.Mirror.Enabled:
			c.Durability = "RAFT COMMITTED (no database mirror configured)"
		case c.Mirror.Connected && c.Mirror.AuditSeq >= head:
			c.Durability = "DB COMMITTED"
		default:
			c.Durability = fmt.Sprintf("JOURNAL ONLY (database mirror at seq %d of %d: %s)", c.Mirror.AuditSeq, head, c.Mirror.LastError)
		}

		ov := &v.Overview
		ov.Hosts = map[string]int{}
		for _, id := range st.sortedNodeIDs() {
			n := st.Nodes[id]
			nv := s.nodeView(st, n, now)
			ov.Hosts[n.Status]++
			if nv.LastObs.Freshness == "FRESH" {
				ov.HostsFresh++
				if nv.LastObs.AgeMs > ov.EvidenceAgeMs {
					ov.EvidenceAgeMs = nv.LastObs.AgeMs
				}
			}
			if n.Health == "lost" {
				ov.Incidents = append(ov.Incidents, fmt.Sprintf("host %s lost: no signed observation for %s", n.Name, time.Duration(now-nv.LastObs.ReceivedAt)*time.Millisecond/time.Second*time.Second))
			}
			if nv.Mode != "" && nv.Mode != "normal" {
				ov.Incidents = append(ov.Incidents, fmt.Sprintf("host %s in %s mode: %s", n.Name, nv.Mode, nv.ModeDetail))
			}
			v.Nodes = append(v.Nodes, nv)
		}
		for _, name := range st.sortedAppNames() {
			av := s.appView(st, st.Apps[name], now)
			ov.Apps++
			ov.Desired += av.Desired
			ov.Admitted += av.Admitted
			ov.Observed += av.Observed
			ov.Drift += av.Drift
			for _, r := range av.Rows {
				if r.Admitted == "REFUSED" {
					ov.Refused++
				}
			}
			v.Apps = append(v.Apps, av)
		}
		for _, id := range st.sortedVolumeIDs() {
			vv := volumeView(st, st.Volumes[id])
			ov.Volumes++
			if vv.State != "HEALTHY" {
				ov.VolumesDegraded++
			}
			v.Volumes = append(v.Volumes, vv)
		}
		for _, d := range sortedKeys(st.Artifacts) {
			a := st.Artifacts[d]
			v.Artifacts = append(v.Artifacts, ArtifactView{Artifact: *a, HolderNames: names(st, a.Holders), Attested: st.Attestations[d] != nil, Local: s.cas.Has(a.Manifest)})
		}
		v.Edge.Services = st.services(s.obs)
		for _, nv := range v.Nodes {
			if nv.Edge != nil || contains(nv.Roles, "edge") {
				v.Edge.Edges = append(v.Edge.Edges, EdgeHostView{Node: nv.ID, Name: nv.Name, Obs: nv.Edge, Freshness: nv.LastObs.Freshness})
			}
		}
		a := &v.Audit
		a.Head, a.HeadHash = st.Audit.Head()
		start := len(st.Audit.Entries) - 300
		if start < 0 {
			start = 0
		}
		a.Entries = append([]audit.Entry(nil), st.Audit.Entries[start:]...)
		a.Checkpoints = len(st.Checkpoints)
		if n := len(st.Checkpoints); n > 0 {
			var p audit.CheckpointPayload
			_ = st.Checkpoints[n-1].Decode(&p)
			a.LastCheckpoint = p.Seq
		}
		a.Verification = verification
		v.Rejections = append(append([]Rejection(nil), st.Rejections...), s.local.all()...)
		sort.Slice(v.Rejections, func(i, j int) bool { return v.Rejections[i].TS > v.Rejections[j].TS })
		if len(v.Rejections) > 100 {
			v.Rejections = v.Rejections[:100]
		}
		rs := st.Repairs
		if len(rs) > 60 {
			rs = rs[len(rs)-60:]
		}
		v.Repairs = append([]RepairRec(nil), rs...)
		v.Federation = federationView(st)
		for i := len(st.ChaosReports) - 1; i >= 0 && len(v.Chaos) < 100; i-- {
			cr := st.ChaosReports[i]
			var hdr struct {
				Scenario string `json:"scenario"`
				Verdict  string `json:"verdict"`
			}
			_ = cr.Env.Decode(&hdr)
			var rep any
			_ = cr.Env.Decode(&rep)
			v.Chaos = append(v.Chaos, ChaosSummary{ID: cr.ID, Scenario: hdr.Scenario, Verdict: hdr.Verdict, Received: cr.Received, Signer: cr.Env.Signer, Evidence: cr.Env.Digest(), Report: rep})
		}
		v.Diagnostics = diagnostics(st, v, s)
		v.Milestones = milestones(st, v)
	})
	if !verification.OK {
		v.Overview.Incidents = append(v.Overview.Incidents, "audit ledger verification failed")
	}
	if v.Cluster.Frozen {
		v.Overview.Incidents = append(v.Overview.Incidents, "control plane frozen: no new work is signed; admitted work continues")
	}
	return v
}

func (s *Server) nodeView(st *State, n *Node, now int64) NodeView {
	nv := NodeView{ID: n.ID, Name: n.Name, Status: n.Status, Health: n.Health, Keys: n.Keys, Roles: n.Roles, MeshIP: n.MeshIP,
		Region: n.Enroll.Region, Zone: n.Enroll.Zone, Host: n.Enroll.Host, OS: n.Enroll.OS, Arch: n.Enroll.Arch, Tiers: n.Enroll.Tiers,
		CPUMilli: n.Enroll.CPUMilli, MemBytes: n.Enroll.MemBytes, Policy: n.Enroll.Policy, PolicyBasis: api.Observed,
		JoinedAt: n.JoinedAt, ApprovedAt: n.ApprovedAt, RevokedAt: n.RevokedAt}
	nv.Identity = "ACTIVE"
	nv.NewAdmission = "ALLOWED"
	nv.Existing = "RUNS"
	switch n.Status {
	case "revoked":
		nv.Identity, nv.NewAdmission, nv.Existing = "REVOKED", "BLOCKED", "MAY CONTINUE"
	case "pending":
		nv.NewAdmission = "BLOCKED (awaiting approval)"
	case "draining":
		nv.NewAdmission = "BLOCKED (draining)"
	}
	obs, env := n.Obs, n.ObsEnv
	nv.LastObs = ObsFreshness{Seq: n.ObserveSeq, ReceivedAt: n.LastObsAt, Committed: n.ObserveSeq, Source: "replicated-log"}
	if c := s.obs.get(n.ID); c != nil && c.Obs.Seq >= n.ObserveSeq {
		obs, env = &c.Obs, c.Env
		nv.LastObs.Seq, nv.LastObs.ReceivedAt, nv.LastObs.Source = c.Obs.Seq, c.Received.UnixMilli(), "leader-cache"
	}
	if obs == nil {
		nv.LastObs.Freshness = "NONE"
		return nv
	}
	nv.LastObs.HostTS, nv.LastObs.Buffered = obs.TS, obs.Buffered
	nv.LastObs.AgeMs = now - nv.LastObs.ReceivedAt
	nv.LastObs.Freshness = "STALE"
	if nv.LastObs.AgeMs < freshWindow.Milliseconds() {
		nv.LastObs.Freshness = "FRESH"
	}
	f := obs.Facts
	nv.Facts, nv.Mesh, nv.Storage, nv.Edge, nv.Ledger, nv.Mode, nv.ModeDetail = &f, obs.Mesh, obs.Storage, obs.Edge, obs.Ledger, obs.Mode, obs.ModeDetail
	nv.Workloads = len(obs.Workloads)
	nv.Evidence = env.Digest()
	if n.Binding != nil {
		var b api.WGBinding
		if n.Binding.Decode(&b) == nil {
			nv.WGBinding = &b
		}
	}
	return nv
}

func (s *Server) appView(st *State, app *App, now int64) AppView {
	av := AppView{Name: app.Name, Generation: app.Generation, Hash: app.Hash, Image: app.Manifest.Spec.Image, Runtime: app.Manifest.Spec.Runtime,
		Replicas: app.Manifest.Spec.Replicas, Deleted: app.Deleted, History: app.History, Manifest: app.Manifest, Ingress: app.Manifest.Spec.Ingress,
		Federation: app.Federation, Outbound: st.Federation.Outbound[app.Name]}
	if pl, ok := s.plans.all()[app.Name]; ok {
		av.Plan = &pl
	}
	for _, key := range st.sortedAssignmentKeys() {
		rec := st.Assignments[key]
		if rec.A.App != app.Name {
			continue
		}
		row := ReplicaView{Replica: rec.A.Replica, Assignment: rec.A.ID, Node: rec.A.Node, NodeName: name(st, rec.A.Node), Desired: strings.ToUpper(rec.A.Desired),
			DesiredGen: rec.A.Generation, Admitted: "UNKNOWN", Observed: "UNKNOWN", Freshness: "NONE", Federation: rec.A.Federation}
		n := st.Nodes[rec.A.Node]
		w := latestWorkload(st, s.obs, rec.A.Node, rec.A.ID)
		if n != nil {
			age := now - n.LastObsAt
			if c := s.obs.get(n.ID); c != nil && c.Obs.Seq >= n.ObserveSeq {
				age = now - c.Received.UnixMilli()
				row.ObservedAt = c.Received.UnixMilli()
				row.Evidence = c.Env.Digest()
			} else {
				row.ObservedAt = n.LastObsAt
				row.Evidence = n.ObsDigest
			}
			if n.Obs != nil || s.obs.get(n.ID) != nil {
				row.Freshness = "STALE"
				if age < freshWindow.Milliseconds() {
					row.Freshness = "FRESH"
				}
			}
		}
		if w != nil {
			row.AdmittedGen, row.ObservedGen = w.Generation, w.Generation
			row.Code, row.Reason, row.Checks = w.Code, w.Reason, w.Checks
			row.PID, row.ContainerID, row.MeshPort, row.Health, row.Restarts, row.StartedAt, row.Volumes = w.PID, w.ContainerID, w.MeshPort, w.Health, w.Restarts, w.StartedAt, w.Volumes
			switch w.Admitted {
			case "allowed":
				row.Admitted = "ADMITTED"
				if w.Code == "HOLD" || strings.Contains(w.Reason, "holding") {
					row.Admitted = "HELD"
				}
			case "refused":
				row.Admitted = "REFUSED"
			default:
				row.Admitted = "PENDING"
			}
			row.Observed = strings.ToUpper(w.Observed)
		}
		switch {
		case w == nil:
			row.Status = "AWAITING HOST"
		case w.Admitted == "refused" && w.Generation == rec.A.Generation:
			row.Status = "DENIED BY HOST"
		case w.Generation < rec.A.Generation && w.Observed == "running":
			row.Status = "STALE OBSERVATION"
		case w.Generation < rec.A.Generation:
			row.Status = "PENDING ADMISSION"
		case rec.A.Desired == "stopped" && w.Observed == "running":
			row.Status = "STOPPING"
		case rec.A.Desired == "stopped":
			row.Status = "STOPPED"
		case w.Observed == "running" && w.Health != nil && !w.Health.OK:
			row.Status = "RUNNING · HEALTH FAILING"
		case w.Observed == "running" && row.Admitted == "HELD":
			row.Status = "RUNNING · HELD"
		case w.Observed == "running":
			row.Status = "RUNNING"
		default:
			row.Status = strings.ToUpper(w.Observed)
		}
		if row.Freshness == "STALE" && w != nil {
			row.Status += " · EVIDENCE STALE"
		}
		if rec.A.Desired == "running" {
			av.Desired++
			if w != nil && w.Admitted == "allowed" && w.Generation == rec.A.Generation {
				av.Admitted++
			}
			if w != nil && w.Observed == "running" && w.Generation == rec.A.Generation && row.Freshness == "FRESH" {
				av.Observed++
			}
		}
		av.Rows = append(av.Rows, row)
	}
	sort.Slice(av.Rows, func(i, j int) bool {
		if av.Rows[i].Replica != av.Rows[j].Replica {
			return av.Rows[i].Replica < av.Rows[j].Replica
		}
		return av.Rows[i].Desired > av.Rows[j].Desired
	})
	if !app.Deleted {
		av.Drift = int(app.Manifest.Spec.Replicas) - av.Observed
		if av.Drift < 0 {
			av.Drift = -av.Drift
		}
	}
	return av
}

func volumeView(st *State, v *Volume) VolumeView {
	vv := VolumeView{Volume: v, MemberNames: names(st, v.Members)}
	for _, sn := range v.Snapshots {
		if sn.ID == v.Committed {
			vv.CommittedRef = sn
		}
	}
	switch {
	case len(v.Snapshots) == 0:
		vv.State, vv.Detail = "NO EVIDENCE", "no snapshot has been reported yet"
	case vv.CommittedRef == nil:
		vv.State, vv.Detail = "UNCOMMITTED", "snapshots exist but none reached write quorum"
	default:
		for _, m := range v.Members {
			if vv.CommittedRef.Evidence[m].Complete(vv.CommittedRef.Root) {
				vv.Verified++
			}
		}
		if int64(vv.Verified) >= v.Durability.Replicas {
			vv.State = "HEALTHY"
			vv.Detail = fmt.Sprintf("%d/%d replicas hold verified evidence for %s", vv.Verified, v.Durability.Replicas, short(v.Committed))
		} else {
			vv.State = "DEGRADED"
			vv.Detail = fmt.Sprintf("%d/%d replicas verified; anti-entropy is repairing", vv.Verified, v.Durability.Replicas)
		}
	}
	if v.Short > 0 {
		vv.Detail += fmt.Sprintf("; %d replica(s) cannot be placed (not enough eligible failure domains)", v.Short)
	}
	return vv
}

// diagnostics lists negative facts explicitly: what is not measured, not
// listening, not installed, or not implemented.
func diagnostics(st *State, v *View, s *Server) []DiagnosticView {
	var out []DiagnosticView
	add := func(subject, item, value string, basis api.Basis, detail string) {
		out = append(out, DiagnosticView{Subject: subject, Item: item, Value: value, Basis: basis, Detail: detail})
	}
	for _, n := range v.Nodes {
		if n.Facts == nil {
			add(n.Name, "facts", "UNKNOWN", api.Unknown, "no signed observation yet")
			continue
		}
		udp := "NOT LISTENING"
		if n.Facts.UDP443 {
			udp = "LISTENING"
		}
		add(n.Name, "UDP 443", udp, api.Observed, "measured by the host")
		docker := "NOT AVAILABLE"
		if n.Facts.Docker != "" {
			docker = n.Facts.Docker
		}
		add(n.Name, "Docker", docker, api.Observed, "container runtime probe")
		for _, p := range n.Facts.Probes {
			val := "NOT AVAILABLE"
			if p.OK {
				val = "AVAILABLE"
			}
			add(n.Name, p.Name, val, api.Observed, p.Detail)
		}
		if n.Mesh == nil || n.Mesh.Device == "" || n.Mesh.Device == "none" {
			add(n.Name, "WireGuard", "NOT RUNNING", api.Observed, "host reported no mesh device")
		} else {
			for _, p := range n.Mesh.Peers {
				lat := "NOT MEASURED"
				if p.RTTUs >= 0 && p.RTTAt > 0 {
					lat = fmt.Sprintf("%.2f ms", float64(p.RTTUs)/1000)
				}
				add(n.Name, "latency → "+name(st, p.Node), lat, api.Observed, "HTTP round trip over WireGuard measured by the host")
			}
		}
	}
	add("cluster", "HTTP/3 (QUIC)", "NOT IMPLEMENTED", api.Planned, "the edge serves HTTP/1.1 and HTTP/2 over TLS")
	add("cluster", "Erasure coding", "NOT IMPLEMENTED", api.Planned, "volumes use full replicas; manifests requesting erasure are rejected")
	add("cluster", "gVisor / Firecracker isolation", "NOT AVAILABLE", api.Planned, "runtimes are subprocess and Docker; stronger sandboxes need Linux hosts")
	add("cluster", "Kernel WireGuard", "NOT USED", api.Configured, "the mesh uses wireguard-go over a userspace netstack (no root needed)")
	if !v.Cluster.Mirror.Enabled {
		add("cluster", "Postgres mirror", "NOT CONFIGURED", api.Configured, "raft log and snapshots are the durability boundary")
	}
	return out
}

func milestones(st *State, v *View) []MilestoneView {
	ms := []MilestoneView{}
	readyHosts := v.Overview.Hosts["ready"]
	m1 := MilestoneView{ID: "M1", Title: "Sovereign runtime", Basis: api.Derived}
	if readyHosts >= 2 && v.Overview.Observed >= 1 && v.Audit.Verification.OK {
		m1.State = "OPERATIONAL"
		m1.Evidence = []string{fmt.Sprintf("%d approved hosts", readyHosts), fmt.Sprintf("%d replica(s) observed running from signed observations", v.Overview.Observed), "audit chain verified now"}
	} else {
		m1.State = "NOT OBSERVED"
		m1.Gaps = []string{"needs ≥2 approved hosts, a running observed workload and a verified audit chain"}
	}
	ms = append(ms, m1)

	m2 := MilestoneView{ID: "M2", Title: "Sovereign storage", Basis: api.Derived}
	healthy := 0
	for _, vv := range v.Volumes {
		if vv.State == "HEALTHY" {
			healthy++
		}
	}
	repairs := len(st.Repairs)
	switch {
	case healthy > 0:
		m2.State = "OPERATIONAL"
		m2.Evidence = []string{fmt.Sprintf("%d volume(s) with committed, replica-verified snapshots", healthy), fmt.Sprintf("%d repair operation(s) recorded", repairs)}
	case len(v.Volumes) > 0:
		m2.State = "PARTIAL"
		m2.Gaps = []string{"volumes exist but none has full replica evidence yet"}
	default:
		m2.State = "NOT OBSERVED"
		m2.Gaps = []string{"no volume has been created"}
	}
	ms = append(ms, m2)

	m3 := MilestoneView{ID: "M3", Title: "Trust + mesh", Basis: api.Observed}
	handshakes := 0
	for _, n := range v.Nodes {
		if n.Mesh == nil {
			continue
		}
		for _, p := range n.Mesh.Peers {
			if p.LastHandshake > 0 && v.GeneratedAt-p.LastHandshake < 180_000 {
				handshakes++
			}
		}
	}
	rotations := 0
	for _, n := range st.Nodes {
		if len(n.Keys) > 1 {
			rotations++
		}
	}
	if handshakes > 0 {
		m3.State = "OPERATIONAL"
		m3.Evidence = []string{fmt.Sprintf("%d WireGuard peer handshake(s) within 3 minutes (host-reported)", handshakes), fmt.Sprintf("%d host(s) with rotated keys", rotations), "per-assignment capabilities verified by hosts"}
	} else {
		m3.State = "NOT OBSERVED"
		m3.Gaps = []string{"no host has reported a WireGuard handshake"}
	}
	ms = append(ms, m3)

	m4 := MilestoneView{ID: "M4", Title: "Edge + TLS", Basis: api.Observed}
	issued, routing := 0, 0
	for _, e := range v.Edge.Edges {
		if e.Obs == nil {
			continue
		}
		for _, c := range e.Obs.Certs {
			if c.State == "ISSUED" {
				issued++
			}
		}
		for _, r := range e.Obs.Routes {
			for _, ep := range r.Endpoints {
				if ep.State == "routing" {
					routing++
				}
			}
		}
	}
	if routing > 0 {
		m4.State = "OPERATIONAL"
		m4.Evidence = []string{fmt.Sprintf("%d endpoint(s) routing on edge hosts", routing), fmt.Sprintf("%d certificate(s) ISSUED", issued)}
	} else {
		m4.State = "NOT OBSERVED"
		m4.Gaps = []string{"no edge host reports a routing endpoint"}
	}
	ms = append(ms, m4)

	m5 := MilestoneView{ID: "M5", Title: "HA control plane", Basis: api.Observed}
	voters := 0
	for _, m := range v.Cluster.Members {
		if m.Suffrage == "Voter" {
			voters++
		}
	}
	if voters >= 3 {
		m5.State = "OPERATIONAL"
		m5.Evidence = []string{v.Cluster.HA, v.Cluster.Quorum}
	} else {
		m5.State = "NOT OBSERVED"
		m5.Gaps = []string{fmt.Sprintf("%d voter(s); HA needs 3 or 5", voters)}
	}
	ms = append(ms, m5)

	m6 := MilestoneView{ID: "M6", Title: "Chaos + resilience", Basis: api.Observed}
	pass, failN := 0, 0
	for _, c := range v.Chaos {
		if c.Verdict == "PASS" {
			pass++
		} else {
			failN++
		}
	}
	if pass > 0 {
		m6.State = "OPERATIONAL"
		m6.Evidence = []string{fmt.Sprintf("%d signed chaos report(s) PASS, %d not PASS", pass, failN)}
	} else {
		m6.State = "NOT OBSERVED"
		m6.Gaps = []string{"no signed chaos report has been submitted to this cluster"}
	}
	ms = append(ms, m6)

	m7 := MilestoneView{ID: "M7", Title: "Federation", Basis: api.Observed}
	if len(st.Federation.Granted)+len(st.Federation.Held) > 0 {
		m7.State = "OPERATIONAL"
		m7.Evidence = []string{fmt.Sprintf("%d agreement(s) granted, %d held, %d inbound placement(s), %d outbound", len(st.Federation.Granted), len(st.Federation.Held), len(st.Federation.Inbound), len(st.Federation.Outbound))}
	} else {
		m7.State = "NOT OBSERVED"
		m7.Gaps = []string{"no federation agreement"}
	}
	ms = append(ms, m7)
	ms = append(ms, conformanceMilestone())
	return ms
}

var _ = manifest.FormatBytes
