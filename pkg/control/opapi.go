package control

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/raft"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/peer"
	"decentralized.host/pkg/storage"
)

func (s *Server) opRoutes(mux *http.ServeMux) {
	read, write, admin := "api.read", "api.write", "api.admin"

	mux.HandleFunc("GET /api/v1/view", s.op(read, s.handleView))
	mux.HandleFunc("GET /api/v1/audit", s.op(read, s.handleAudit))
	mux.HandleFunc("GET /api/v1/audit/verify", s.op(read, s.handleAuditVerify))
	mux.HandleFunc("GET /api/v1/plans", s.op(read, func(w http.ResponseWriter, _ *http.Request, _ *authz) { writeJSON(w, 200, s.plans.all()) }))
	mux.HandleFunc("GET /api/v1/state", s.op(admin, s.handleStateDump))
	mux.HandleFunc("GET /api/v1/export", s.op(admin, s.handleExport))
	mux.HandleFunc("GET /api/v1/nodes/{id}/ledger", s.op(read, s.handleHostLedger))
	mux.HandleFunc("GET /api/v1/nodes/{id}/logs", s.op(read, s.handleHostLogs))
	mux.HandleFunc("POST /api/v1/nodes/{id}/exec", s.op(admin, s.handleHostExec))
	mux.HandleFunc("POST /api/v1/mesh/ping", s.op(read, s.handleMeshPing))
	mux.HandleFunc("GET /api/v1/cp/raft", s.op(read, s.handleRaftInfo))

	mux.HandleFunc("POST /api/v1/apply", s.opWrite(write, s.handleApply))
	mux.HandleFunc("POST /api/v1/apps/{name}/scale", s.opWrite(write, s.handleScale))
	mux.HandleFunc("POST /api/v1/apps/{name}/delete", s.opWrite(write, s.simple("delete-app", func(r *http.Request) any { return map[string]string{"app": r.PathValue("name")} })))
	mux.HandleFunc("POST /api/v1/nodes/{id}/approve", s.opWrite(admin, s.simple("approve", func(r *http.Request) any { return map[string]string{"node": r.PathValue("id")} })))
	mux.HandleFunc("POST /api/v1/nodes/{id}/revoke", s.opWrite(admin, s.handleRevoke))
	mux.HandleFunc("POST /api/v1/nodes/{id}/drain", s.opWrite(write, s.simple("drain", func(r *http.Request) any { return map[string]any{"node": r.PathValue("id"), "drain": true} })))
	mux.HandleFunc("POST /api/v1/nodes/{id}/undrain", s.opWrite(write, s.simple("drain", func(r *http.Request) any { return map[string]any{"node": r.PathValue("id"), "drain": false} })))
	mux.HandleFunc("POST /api/v1/nodes/{id}/revoke-key", s.opWrite(admin, s.handleRevokeKey))
	mux.HandleFunc("POST /api/v1/invites", s.opWrite(admin, s.handleInvite))
	mux.HandleFunc("POST /api/v1/invites/{nonce}/revoke", s.opWrite(admin, s.simple("invite-revoke", func(r *http.Request) any { return map[string]string{"nonce": r.PathValue("nonce")} })))
	mux.HandleFunc("POST /api/v1/freeze", s.opWrite(admin, s.handleFreeze))
	mux.HandleFunc("POST /api/v1/roster", s.opWrite(admin, s.passEnvelope("roster")))
	mux.HandleFunc("POST /api/v1/root-rotate", s.opWrite(admin, s.passJSON("root-rotate")))
	mux.HandleFunc("POST /api/v1/publishers", s.opWrite(admin, s.passJSON("publishers")))
	mux.HandleFunc("POST /api/v1/attest", s.opWrite(write, s.passEnvelope("attest")))
	mux.HandleFunc("POST /api/v1/chaos/report", s.opWrite(write, s.passEnvelope("chaos-report")))
	mux.HandleFunc("POST /api/v1/artifacts", s.opWrite(write, s.handleArtifactUpload))
	mux.HandleFunc("POST /api/v1/cp/add-voter", s.opWrite(admin, s.handleAddVoter))
	mux.HandleFunc("POST /api/v1/cp/remove-server", s.opWrite(admin, s.handleRemoveServer))
	mux.HandleFunc("POST /api/v1/cp/transfer-leadership", s.opWrite(admin, s.handleTransferLeadership))
	mux.HandleFunc("POST /api/v1/cp/backup", s.op(admin, s.handleBackup))
	mux.HandleFunc("POST /api/v1/cp/snapshot", s.opWrite(admin, s.handleRaftSnapshot))
	mux.HandleFunc("POST /api/v1/cp/restore", s.opWrite(admin, s.passEnvelope("restore")))
	mux.HandleFunc("GET /api/v1/volumes/export", s.op(read, s.handleVolumeExport))

	s.labRoutes(mux)
}

func (s *Server) respond(w http.ResponseWriter, res *Result, err error) {
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "%v", err)
		return
	}
	code := 200
	if !res.OK {
		code = http.StatusConflict
	}
	writeJSON(w, code, res)
}

func (s *Server) simple(typ string, data func(r *http.Request) any) func(http.ResponseWriter, *http.Request, *authz) {
	return func(w http.ResponseWriter, r *http.Request, a *authz) {
		res, err := s.propose(typ, a.Actor, data(r))
		s.respond(w, res, err)
	}
}

func (s *Server) passEnvelope(typ string) func(http.ResponseWriter, *http.Request, *authz) {
	return func(w http.ResponseWriter, r *http.Request, a *authz) {
		var env envelope.Envelope
		if err := readBody(r, 4<<20, &env); err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		res, err := s.propose(typ, a.Actor, &env)
		s.respond(w, res, err)
	}
}

func (s *Server) passJSON(typ string) func(http.ResponseWriter, *http.Request, *authz) {
	return func(w http.ResponseWriter, r *http.Request, a *authz) {
		var raw json.RawMessage
		if err := readBody(r, 4<<20, &raw); err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		res, err := s.propose(typ, a.Actor, raw)
		s.respond(w, res, err)
	}
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request, a *authz) {
	src, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	if strings.HasPrefix(strings.TrimSpace(string(src)), "{") {
		var wrapped struct {
			YAML string `json:"yaml"`
		}
		if json.Unmarshal(src, &wrapped) == nil && wrapped.YAML != "" {
			src = []byte(wrapped.YAML)
		}
	}
	m, err := manifest.Parse(src)
	if err != nil {
		writeErr(w, 422, "%v", err)
		return
	}
	var fed bool
	s.fsm.Read(func(st *State) {
		if app := st.Apps[m.Metadata.Name]; app != nil && app.Federation != nil {
			fed = true
		}
	})
	if fed {
		writeErr(w, 409, "%s is a federated placement owned by a peer cluster", m.Metadata.Name)
		return
	}
	if len(m.Spec.Placement.Federation) > 0 {
		s.handleFederatedApply(w, m, a)
		return
	}
	res, err := s.propose("apply", a.Actor, map[string]any{"manifest": m})
	s.respond(w, res, err)
}

func (s *Server) handleScale(w http.ResponseWriter, r *http.Request, a *authz) {
	var body struct {
		Replicas int64 `json:"replicas"`
	}
	if err := readBody(r, 4096, &body); err != nil || body.Replicas < 0 || body.Replicas > 64 {
		writeErr(w, 400, "replicas must be 0..64")
		return
	}
	var m *api.Manifest
	s.fsm.Read(func(st *State) {
		if app := st.Apps[r.PathValue("name")]; app != nil && !app.Deleted {
			cp := app.Manifest
			m = &cp
		}
	})
	if m == nil {
		writeErr(w, 404, "no app %s", r.PathValue("name"))
		return
	}
	m.Spec.Replicas = body.Replicas
	res, err := s.propose("apply", a.Actor, map[string]any{"manifest": m})
	s.respond(w, res, err)
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request, a *authz) {
	var body struct {
		Reason string `json:"reason"`
	}
	_ = readBody(r, 4096, &body)
	if body.Reason == "" {
		body.Reason = "operator revocation"
	}
	res, err := s.propose("revoke", a.Actor, map[string]string{"node": r.PathValue("id"), "reason": body.Reason})
	s.respond(w, res, err)
}

func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request, a *authz) {
	var body struct {
		Pub    string `json:"pub"`
		Reason string `json:"reason"`
	}
	if err := readBody(r, 4096, &body); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	res, err := s.propose("revoke-key", a.Actor, map[string]string{"node": r.PathValue("id"), "pub": body.Pub, "reason": body.Reason})
	s.respond(w, res, err)
}

func (s *Server) handleInvite(w http.ResponseWriter, r *http.Request, a *authz) {
	var body inviteData
	if err := readBody(r, 8192, &body); err != nil || len(body.Nonce) < 16 {
		writeErr(w, 400, "invite needs a nonce of at least 16 characters")
		return
	}
	res, err := s.propose("invite", a.Actor, body)
	s.respond(w, res, err)
}

func (s *Server) handleFreeze(w http.ResponseWriter, r *http.Request, a *authz) {
	var body struct {
		Frozen bool `json:"frozen"`
	}
	if err := readBody(r, 4096, &body); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	res, err := s.propose("freeze", a.Actor, body)
	s.respond(w, res, err)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request, _ *authz) {
	from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	var out struct {
		Cluster     string               `json:"cluster"`
		Entries     []audit.Entry        `json:"entries"`
		Checkpoints []*envelope.Envelope `json:"checkpoints"`
		Members     []api.Member         `json:"members"`
		PastRosters []*envelope.Envelope `json:"pastRosters"`
		Head        int64                `json:"head"`
	}
	s.fsm.Read(func(st *State) {
		out.Cluster = st.Cluster
		out.Head, _ = st.Audit.Head()
		for _, e := range st.Audit.Entries {
			if e.Seq > from {
				out.Entries = append(out.Entries, e)
				if len(out.Entries) >= limit {
					break
				}
			}
		}
		out.Checkpoints = st.Checkpoints
		out.Members = st.RosterBody.Members
		out.PastRosters = st.PastRosters
	})
	writeJSON(w, 200, out)
}

// AuditVerification is the result of a full ledger verification.
type AuditVerification struct {
	OK          bool         `json:"ok"`
	Entries     int          `json:"entries"`
	Head        string       `json:"head"`
	Break       *audit.Break `json:"break"`
	Checkpoints int          `json:"checkpoints"`
	CheckBreak  *audit.Break `json:"checkpointBreak"`
	VerifiedAt  int64        `json:"verifiedAt"`
	VerifiedBy  string       `json:"verifiedBy"`
}

func (s *Server) verifyLedger() AuditVerification {
	var v AuditVerification
	s.fsm.Read(func(st *State) {
		v.Entries = len(st.Audit.Entries)
		_, v.Head = st.Audit.Head()
		v.Break = audit.Verify(st.Audit.Entries, 0, audit.Genesis)
		v.Checkpoints = len(st.Checkpoints)
		v.CheckBreak = audit.VerifyCheckpoints(st.Audit.Entries, st.Checkpoints, st.CheckpointKeys)
	})
	v.OK = v.Break == nil && v.CheckBreak == nil
	v.VerifiedAt, v.VerifiedBy = nowMs(), s.id.ID
	return v
}

func (s *Server) handleAuditVerify(w http.ResponseWriter, _ *http.Request, _ *authz) {
	writeJSON(w, 200, s.verifyLedger())
}

func (s *Server) handleStateDump(w http.ResponseWriter, _ *http.Request, _ *authz) {
	b, err := s.fsm.Export()
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	var st State
	_ = json.Unmarshal(b, &st)
	st.LocalCAKey = "[secret withheld]"
	writeJSON(w, 200, &st)
}

// nodeMeshIP resolves a host id to its mesh address.
func (s *Server) nodeMeshIP(id string) (string, string) {
	var ip, nm string
	s.fsm.Read(func(st *State) {
		if n := st.Nodes[id]; n != nil {
			ip, nm = n.MeshIP, n.Name
		}
		if ip == "" {
			for _, n := range st.Nodes {
				if n.Name == id {
					ip, nm = n.MeshIP, n.Name
				}
			}
		}
	})
	return ip, nm
}

func (s *Server) peerClient(timeout time.Duration) (*peer.Client, error) {
	c, err := s.meshHTTP(timeout)
	if err != nil {
		return nil, err
	}
	return &peer.Client{HTTP: c}, nil
}

func (s *Server) handleHostLedger(w http.ResponseWriter, r *http.Request, _ *authz) {
	ip, _ := s.nodeMeshIP(r.PathValue("id"))
	if ip == "" {
		writeErr(w, 404, "no such host")
		return
	}
	pc, err := s.peerClient(10 * time.Second)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rep, err := pc.Ledger(r.Context(), ip, from, limit)
	if err != nil {
		writeErr(w, 502, "host ledger over mesh: %v", err)
		return
	}
	writeJSON(w, 200, rep)
}

func (s *Server) handleHostLogs(w http.ResponseWriter, r *http.Request, a *authz) {
	ip, name := s.nodeMeshIP(r.PathValue("id"))
	if ip == "" {
		writeErr(w, 404, "no such host")
		return
	}
	pc, err := s.peerClient(10 * time.Second)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	b, err := pc.Logs(r.Context(), ip, r.URL.Query().Get("assignment"), tail)
	if err != nil {
		writeErr(w, 502, "logs from %s over mesh: %v", name, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) handleHostExec(w http.ResponseWriter, r *http.Request, a *authz) {
	ip, _ := s.nodeMeshIP(r.PathValue("id"))
	if ip == "" {
		writeErr(w, 404, "no such host")
		return
	}
	var req peer.ExecRequest
	if err := readBody(r, 64<<10, &req); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	// The host verifies the operator's own capability; the control plane
	// cannot mint one that the host would accept for exec.
	req.Capability = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	pc, err := s.peerClient(30 * time.Second)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	rep, err := pc.Exec(r.Context(), ip, req)
	if err != nil {
		writeErr(w, 502, "%v", err)
		return
	}
	detail := fmt.Sprintf("argv %q → exit %d", req.Argv, rep.ExitCode)
	if rep.Refused != "" {
		detail = fmt.Sprintf("argv %q refused by host: %s", req.Argv, rep.Refused)
	}
	_, _ = s.propose("operator-note", a.Actor, map[string]string{"action": "operator-exec", "resource": "app/" + req.Assignment, "detail": detail})
	writeJSON(w, 200, rep)
}

func (s *Server) handleMeshPing(w http.ResponseWriter, r *http.Request, _ *authz) {
	var body struct {
		Node string `json:"node"`
	}
	_ = readBody(r, 4096, &body)
	ip, name := s.nodeMeshIP(body.Node)
	if ip == "" {
		writeErr(w, 404, "no such host")
		return
	}
	pc, err := s.peerClient(5 * time.Second)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	var samples []int64
	var lastErr string
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		d, err := pc.Ping(ctx, ip)
		cancel()
		if err != nil {
			lastErr = err.Error()
			if len(samples) == 0 {
				break // unreachable: report it now rather than after 5 timeouts
			}
			continue
		}
		samples = append(samples, d.Microseconds())
	}
	writeJSON(w, 200, map[string]any{"node": name, "meshIp": ip, "samplesUs": samples, "error": lastErr, "from": s.id.ID,
		"method": "HTTP GET /peer/v1/ping over WireGuard (userspace), 5 sequential samples"})
}

func (s *Server) handleRaftInfo(w http.ResponseWriter, _ *http.Request, _ *authz) {
	rn := s.raft()
	if rn == nil {
		writeErr(w, 503, "not bootstrapped")
		return
	}
	cfg := rn.r.GetConfiguration()
	var servers []map[string]any
	if cfg.Error() == nil {
		for _, sv := range cfg.Configuration().Servers {
			servers = append(servers, map[string]any{"id": string(sv.ID), "address": string(sv.Address), "suffrage": sv.Suffrage.String()})
		}
	}
	_, lid := rn.r.LeaderWithID()
	writeJSON(w, 200, map[string]any{"servers": servers, "leader": string(lid), "state": rn.r.State().String(), "stats": rn.r.Stats()})
}

func (s *Server) handleAddVoter(w http.ResponseWriter, r *http.Request, a *authz) {
	var body struct {
		ID       string `json:"id"`
		RaftAddr string `json:"raftAddr"`
	}
	if err := readBody(r, 4096, &body); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	inRoster := false
	s.fsm.Read(func(st *State) { inRoster = st.member(body.ID) != nil })
	if !inRoster {
		writeErr(w, 409, "%s is not in the root-signed roster; publish a roster that includes it first", body.ID)
		return
	}
	if err := s.raft().r.AddVoter(raft.ServerID(body.ID), raft.ServerAddress(body.RaftAddr), 0, 30*time.Second).Error(); err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, ok("%s added as a voter", body.ID))
}

func (s *Server) handleRemoveServer(w http.ResponseWriter, r *http.Request, _ *authz) {
	var body struct {
		ID string `json:"id"`
	}
	if err := readBody(r, 4096, &body); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	if err := s.raft().r.RemoveServer(raft.ServerID(body.ID), 0, 30*time.Second).Error(); err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, ok("%s removed from the raft configuration", body.ID))
}

func (s *Server) handleTransferLeadership(w http.ResponseWriter, _ *http.Request, _ *authz) {
	if err := s.raft().r.LeadershipTransfer().Error(); err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, ok("leadership transferred"))
}

func (s *Server) handleRaftSnapshot(w http.ResponseWriter, _ *http.Request, _ *authz) {
	f := s.raft().r.Snapshot()
	if err := f.Error(); err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	meta, _, err := f.Open()
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "index": meta.Index, "term": meta.Term, "size": meta.Size, "id": meta.ID})
}

// Backup is a signed, verifiable copy of the replicated state.
type Backup struct {
	Cluster   string          `json:"cluster"`
	Index     int64           `json:"index"`
	Member    string          `json:"member"`
	TS        int64           `json:"ts"`
	StateHash string          `json:"stateHash"`
	State     json.RawMessage `json:"state"`
	Secrets   bool            `json:"secrets"`
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request, _ *authz) {
	includeSecrets := r.URL.Query().Get("secrets") == "true"
	raw, err := s.fsm.Export()
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	var st State
	_ = json.Unmarshal(raw, &st)
	if !includeSecrets {
		st.LocalCAKey = ""
	}
	body, err := canon.Marshal(&st)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	b := Backup{Cluster: st.Cluster, Index: st.Index, Member: s.id.ID, TS: nowMs(), StateHash: "b3:" + envelope.HashBytes(body), State: body, Secrets: includeSecrets}
	env, err := envelope.Sign(s.id, "", envelope.KindBackup, b)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, env)
}

// handleArtifactUpload chunks an executable into this member's CAS and
// registers it; distribution to hosts happens in housekeeping.
func (s *Server) handleArtifactUpload(w http.ResponseWriter, r *http.Request, a *authz) {
	name := r.URL.Query().Get("name")
	if !regexpName(name) {
		writeErr(w, 400, "artifact name must be lowercase letters, digits, dot, dash or underscore")
		return
	}
	body := io.LimitReader(r.Body, 512<<20)
	h := newB3()
	ids, n, err := storage.PutStream(s.cas, io.TeeReader(body, h))
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	digest := "b3:" + h.hex()
	m := api.ArtifactManifest{Digest: digest, Name: name, Bytes: n, Chunks: ids}
	mb, _ := canon.Marshal(m)
	mid, err := s.cas.Put(mb)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	res, err := s.propose("artifact", a.Actor, Artifact{Digest: digest, Name: name, Bytes: n, Chunks: int64(len(ids)), Manifest: mid, Holders: []string{}})
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	res.Data = map[string]any{"digest": digest, "image": fmt.Sprintf("%s@%s", name, digest), "bytes": n, "chunks": len(ids), "manifest": mid}
	s.wake()
	writeJSON(w, 200, res)
}
