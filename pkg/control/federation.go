package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/pki"
)

// Federation lets independent installations cooperate without a global
// authority. The grantor's root signs an agreement naming the grantee's
// root and limits. The grantee's control plane places work by sending a
// request signed by one of its members, whose authority chains to the
// grantee root. The grantor re-signs everything it runs under its own root:
// its hosts never trust, and never see, any key of the grantee.

// FederationView is the console projection of federation state.
type FederationView struct {
	Granted  []*FedAgreementRec `json:"granted"`
	Held     []*FedAgreementRec `json:"held"`
	Inbound  []*FedPlacement    `json:"inbound"`
	Outbound []*FedOutbound     `json:"outbound"`
}

func federationView(st *State) FederationView {
	var v FederationView
	for _, k := range sortedKeys(st.Federation.Granted) {
		v.Granted = append(v.Granted, st.Federation.Granted[k])
	}
	for _, k := range sortedKeys(st.Federation.Held) {
		v.Held = append(v.Held, st.Federation.Held[k])
	}
	for _, k := range sortedKeys(st.Federation.Inbound) {
		v.Inbound = append(v.Inbound, st.Federation.Inbound[k])
	}
	for _, k := range sortedKeys(st.Federation.Outbound) {
		v.Outbound = append(v.Outbound, st.Federation.Outbound[k])
	}
	return v
}

// FedRequest is the payload of a federation-placement envelope.
type FedRequest struct {
	Op         string        `json:"op"` // place | status | withdraw | chunk
	Agreement  string        `json:"agreement"`
	Grantee    string        `json:"grantee"`
	App        string        `json:"app"`
	Manifest   *api.Manifest `json:"manifest"`
	Artifact   *Artifact     `json:"artifact"`
	Object     string        `json:"object"`
	Delegation string        `json:"delegation"` // grantee root → requesting member
	TS         int64         `json:"ts"`
}

// FedStatus is signed by a grantor member (envelope kind federation-status).
type FedStatus struct {
	Agreement string             `json:"agreement"`
	App       string             `json:"app"`
	LocalApp  string             `json:"localApp"`
	Accepted  bool               `json:"accepted"`
	Message   string             `json:"message"`
	Revoked   bool               `json:"revoked"`
	Rows      []FedReplicaRow    `json:"rows"`
	Roster    *envelope.Envelope `json:"roster"` // lets the grantee verify the signer chains to the grantor root
	TS        int64              `json:"ts"`
}

// FedReplicaRow is one remote replica as the grantor observes it.
type FedReplicaRow struct {
	Replica    int64  `json:"replica"`
	Node       string `json:"node"`
	Desired    string `json:"desired"`
	Admitted   string `json:"admitted"`
	Observed   string `json:"observed"`
	Generation int64  `json:"generation"`
	Code       string `json:"code"`
	Evidence   string `json:"evidence"` // digest of the host-signed observation
	Freshness  string `json:"freshness"`
}

func fedLocalName(grantee, app string) string {
	n := "fed-" + grantee + "-" + app
	if len(n) > 42 {
		n = n[:42]
	}
	return strings.TrimRight(n, "-")
}

// ------------------------------------------------------------- FSM commands

func verifyAgreement(env *envelope.Envelope) (FedAgreement, error) {
	var a FedAgreement
	if err := env.Decode(&a); err != nil {
		return a, err
	}
	if err := env.VerifyKey(envelope.KindFedAgreement, a.GrantorRoot); err != nil {
		return a, fmt.Errorf("agreement not signed by the grantor root it names: %w", err)
	}
	if a.Grantor == "" || a.Grantee == "" || a.GranteeRoot == "" {
		return a, errors.New("agreement is incomplete")
	}
	return a, nil
}

func init() {
	// fed-grant: this cluster (as grantor) records an agreement its own root signed.
	register("fed-grant", func(f *FSM, s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		a, err := verifyAgreement(env)
		if err != nil {
			return fail("AGREEMENT", "%v", err)
		}
		if a.GrantorRoot != s.Root || a.Grantor != s.Cluster {
			return fail("AGREEMENT", "a granted agreement must be signed by this cluster's root")
		}
		d := env.Digest()
		s.Federation.Granted[d] = &FedAgreementRec{Digest: d, Env: env, A: a, Received: c.TS}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-grant", Resource: "federation/" + a.Grantee,
			Detail:   fmt.Sprintf("granted %s (root %s): tiers %v, ≤%d replica(s), ≤%dm cpu, ≤%s, until %s", a.Grantee, short(a.GranteeRoot), a.Tiers, a.MaxReplicas, a.MaxCPUMilli, manifest.FormatBytes(a.MaxMemBytes), time.UnixMilli(a.Expires).UTC().Format(time.RFC3339)),
			Evidence: d})
		return &Result{OK: true, Message: "agreement granted to " + a.Grantee, Data: d}
	})

	// fed-hold: this cluster (as grantee) records an agreement a peer granted.
	register("fed-hold", func(f *FSM, s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		a, err := verifyAgreement(env)
		if err != nil {
			return fail("AGREEMENT", "%v", err)
		}
		if a.Grantee != s.Cluster || a.GranteeRoot != s.Root {
			return fail("AGREEMENT", "agreement is for %s (root %s), not this cluster", a.Grantee, short(a.GranteeRoot))
		}
		d := env.Digest()
		s.Federation.Held[d] = &FedAgreementRec{Digest: d, Env: env, A: a, Received: c.TS}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-accept", Resource: "federation/" + a.Grantor,
			Detail: fmt.Sprintf("accepted agreement from %s (root %s pinned by operator acceptance)", a.Grantor, short(a.GrantorRoot)), Evidence: d})
		return &Result{OK: true, Message: "agreement from " + a.Grantor + " accepted", Data: d}
	})

	register("fed-revoke", func(f *FSM, s *State, c *Command) *Result {
		env, err := decode[*envelope.Envelope](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		if err := env.VerifyKey(envelope.KindFedRevocation, s.Root); err != nil {
			return fail("REVOCATION", "revocation must be signed by this cluster's root: %v", err)
		}
		var r struct {
			Agreement string `json:"agreement"`
			Reason    string `json:"reason"`
		}
		_ = env.Decode(&r)
		rec := s.Federation.Granted[r.Agreement]
		if rec == nil {
			return fail("NOT_FOUND", "no granted agreement %s", short(r.Agreement))
		}
		rec.Revoked, rec.RevokedAt, rec.Revocation = true, c.TS, env
		stopped := 0
		for _, name := range sortedKeys(s.Federation.Inbound) {
			in := s.Federation.Inbound[name]
			if in.Agreement == r.Agreement && !in.Withdrawn {
				in.Withdrawn = true
				if app := s.Apps[in.App]; app != nil && !app.Deleted {
					app.Deleted, app.Updated = true, c.TS
					stopped++
				}
			}
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-revoke", Resource: "federation/" + rec.A.Grantee,
			Detail: fmt.Sprintf("agreement %s revoked (%s); %d inbound app(s) stopped by this cluster", short(r.Agreement), r.Reason, stopped), Evidence: env.Digest()})
		return ok("agreement revoked; %d inbound app(s) will receive signed stops", stopped)
	})

	register("fed-inbound", func(f *FSM, s *State, c *Command) *Result {
		d, err := decode[FedPlacement](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		d.Received = c.TS
		s.Federation.Inbound[d.App] = &d
		return ok("inbound placement recorded")
	})

	register("fed-withdraw", func(f *FSM, s *State, c *Command) *Result {
		d, err := decode[struct {
			App string `json:"app"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		in := s.Federation.Inbound[d.App]
		if in == nil {
			return fail("NOT_FOUND", "no inbound app %s", d.App)
		}
		in.Withdrawn = true
		if app := s.Apps[d.App]; app != nil && !app.Deleted {
			app.Deleted, app.Updated = true, c.TS
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-withdraw", Resource: "app/" + d.App, Detail: "withdrawn by " + in.Peer})
		return ok("withdrawn")
	})

	register("fed-outbound", func(f *FSM, s *State, c *Command) *Result {
		d, err := decode[FedOutbound](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		cur := s.Federation.Outbound[d.App]
		if cur == nil {
			d.Requested = c.TS
			s.Federation.Outbound[d.App] = &d
			s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-place-request", Resource: "app/" + d.App,
				Detail: fmt.Sprintf("%d replica(s) requested on %s under agreement %s", d.Replicas, d.Peer, short(d.Agreement))})
			return ok("placement requested on %s", d.Peer)
		}
		changed := cur.Accepted != d.Accepted || cur.Message != d.Message
		cur.Accepted, cur.Message = d.Accepted, d.Message
		if d.LastStatus != nil {
			cur.LastStatus, cur.StatusAt = d.LastStatus, c.TS
		}
		if d.Manifest.Metadata.Name != "" {
			cur.Manifest, cur.Replicas = d.Manifest, d.Replicas
		}
		if changed {
			s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-status", Resource: "app/" + d.App,
				Detail: fmt.Sprintf("%s: accepted=%v %s", d.Peer, d.Accepted, d.Message)})
		}
		return ok("outbound updated")
	})

	register("fed-outbound-delete", func(f *FSM, s *State, c *Command) *Result {
		d, _ := decode[struct {
			App string `json:"app"`
		}](c)
		delete(s.Federation.Outbound, d.App)
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceFederation, Action: "federation-withdraw-request", Resource: "app/" + d.App, Detail: "withdrawn by this cluster"})
		return ok("outbound removed")
	})
}

// ------------------------------------------------------------ operator API

func (s *Server) fedRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/federation/grant", s.opWrite("api.admin", s.passEnvelope("fed-grant")))
	mux.HandleFunc("POST /api/v1/federation/accept", s.opWrite("api.admin", s.passEnvelope("fed-hold")))
	mux.HandleFunc("POST /api/v1/federation/revoke", s.opWrite("api.admin", s.passEnvelope("fed-revoke")))
	mux.HandleFunc("POST /api/v1/federation/withdraw", s.opWrite("api.write", func(w http.ResponseWriter, r *http.Request, a *authz) {
		var body struct {
			App string `json:"app"`
		}
		if err := readBody(r, 4096, &body); err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		res, err := s.propose("fed-outbound-delete", a.Actor, body)
		s.respond(w, res, err)
	}))
	// Peer channel: requests signed by a member of a grantee cluster.
	mux.HandleFunc("POST /fed/v1/request", s.leaderOnly(s.handleFedRequest))
}

// handleFederatedApply records an outbound placement; the leader delivers it.
func (s *Server) handleFederatedApply(w http.ResponseWriter, m *api.Manifest, a *authz) {
	peer := m.Spec.Placement.Federation[0]
	var agreement string
	var why string
	s.fsm.Read(func(st *State) {
		for _, d := range sortedKeys(st.Federation.Held) {
			rec := st.Federation.Held[d]
			if rec.A.Grantor == peer && !rec.Revoked && (rec.A.Expires == 0 || nowMs() < rec.A.Expires) {
				agreement = d
			}
		}
		if agreement == "" {
			why = "no current agreement from " + peer + "; run `dh federation accept` with an agreement " + peer + " granted"
		}
	})
	if agreement == "" {
		writeErr(w, 409, "%s", why)
		return
	}
	res, err := s.propose("fed-outbound", a.Actor, FedOutbound{App: m.Metadata.Name, Peer: peer, Agreement: agreement, Replicas: m.Spec.Replicas, Manifest: *m, Message: "awaiting delivery"})
	s.respond(w, res, err)
	s.wake()
}

// ------------------------------------------------------------- grantor side

// verifyFedRequest authenticates a peer request against a granted agreement.
func (s *Server) verifyFedRequest(env *envelope.Envelope) (*FedRequest, *FedAgreementRec, error) {
	var req FedRequest
	if err := env.Decode(&req); err != nil {
		return nil, nil, err
	}
	var rec *FedAgreementRec
	s.fsm.Read(func(st *State) {
		if r := st.Federation.Granted[req.Agreement]; r != nil {
			cp := *r
			rec = &cp
		}
	})
	if rec == nil {
		return nil, nil, errors.New("unknown agreement")
	}
	if rec.Revoked {
		return &req, rec, errors.New("agreement revoked by the grantor")
	}
	now := nowMs()
	if rec.A.Expires != 0 && now >= rec.A.Expires {
		return &req, rec, errors.New("agreement expired")
	}
	if req.Grantee != rec.A.Grantee {
		return nil, nil, errors.New("request names a different grantee")
	}
	if d := now - req.TS; d > 300_000 || d < -300_000 {
		return nil, nil, errors.New("request timestamp outside the 5 minute window")
	}
	tok, err := capability.Decode(req.Delegation)
	if err != nil {
		return nil, nil, fmt.Errorf("delegation: %w", err)
	}
	vr := capability.Verify(tok, []string{rec.A.GranteeRoot}, capability.Request{Action: "federation." + req.Op, Resource: "federation/" + rec.A.Grantor, Now: now})
	if !vr.OK {
		return nil, nil, fmt.Errorf("delegation does not authorize %s: %s", req.Op, vr.Reason)
	}
	if vr.Last.Next != env.Pub {
		return nil, nil, errors.New("request is not signed by the delegated member key")
	}
	if err := env.VerifyKey(envelope.KindFedPlacement, env.Pub); err != nil {
		return nil, nil, err
	}
	if !contains(rec.A.Actions, "federation."+req.Op) && !contains(rec.A.Actions, "federation.*") {
		return nil, nil, fmt.Errorf("agreement does not allow %s", req.Op)
	}
	return &req, rec, nil
}

func (s *Server) handleFedRequest(w http.ResponseWriter, r *http.Request) {
	var env envelope.Envelope
	if err := readBody(r, 8<<20, &env); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	req, rec, err := s.verifyFedRequest(&env)
	if err != nil {
		code := 403
		if req != nil && rec != nil {
			s.writeFedStatus(w, FedStatus{Agreement: rec.Digest, App: req.App, Accepted: false, Revoked: rec.Revoked, Message: err.Error()})
			return
		}
		s.local.add(Rejection{TS: nowMs(), Kind: "federation", Node: env.Signer, Reason: err.Error()})
		writeErr(w, code, "federation: %v", err)
		return
	}
	switch req.Op {
	case "place":
		s.fedPlace(w, req, rec, &env)
	case "status":
		s.writeFedStatus(w, s.fedStatus(req, rec))
	case "withdraw":
		local := fedLocalName(rec.A.Grantee, req.App)
		res, err := s.propose("fed-withdraw", "federation:"+rec.A.Grantee, map[string]string{"app": local})
		if err != nil {
			writeErr(w, 503, "%v", err)
			return
		}
		s.writeFedStatus(w, FedStatus{Agreement: rec.Digest, App: req.App, LocalApp: local, Accepted: res.OK, Message: res.Message})
	default:
		writeErr(w, 400, "unknown op %q", req.Op)
	}
}

// fedPlace enforces the agreement, then runs the app under this cluster's
// own authority (its own root, scheduler and host policies).
func (s *Server) fedPlace(w http.ResponseWriter, req *FedRequest, rec *FedAgreementRec, env *envelope.Envelope) {
	a := rec.A
	m := req.Manifest
	refuse := func(format string, args ...any) {
		s.writeFedStatus(w, FedStatus{Agreement: rec.Digest, App: req.App, Accepted: false, Message: fmt.Sprintf(format, args...)})
	}
	if m == nil {
		refuse("no manifest")
		return
	}
	spec := m.Spec
	if a.MaxReplicas > 0 && spec.Replicas > a.MaxReplicas {
		refuse("agreement allows at most %d replica(s), requested %d", a.MaxReplicas, spec.Replicas)
		return
	}
	if a.MaxCPUMilli > 0 && spec.Replicas*spec.Resources.CPUMilli > a.MaxCPUMilli {
		refuse("agreement allows %dm cpu in total, requested %dm", a.MaxCPUMilli, spec.Replicas*spec.Resources.CPUMilli)
		return
	}
	if a.MaxMemBytes > 0 && spec.Replicas*spec.Resources.MemBytes > a.MaxMemBytes {
		refuse("agreement allows %s memory in total, requested %s", manifest.FormatBytes(a.MaxMemBytes), manifest.FormatBytes(spec.Replicas*spec.Resources.MemBytes))
		return
	}
	for _, t := range spec.Placement.Tiers {
		if !contains(a.Tiers, t) {
			refuse("tier %q is not allowed by the agreement (allowed %v)", t, a.Tiers)
			return
		}
	}
	if len(a.Runtimes) > 0 && !contains(a.Runtimes, spec.Runtime) {
		refuse("runtime %s is not allowed by the agreement", spec.Runtime)
		return
	}
	if len(spec.Volumes) > 0 || len(spec.Ingress) > 0 {
		refuse("federated placements may not declare volumes or ingress in this release")
		return
	}
	// A process artifact is fetched from the grantee and verified by BLAKE3.
	if spec.Runtime == "process" && req.Artifact != nil {
		if err := s.fetchPeerArtifact(rec, req.Artifact); err != nil {
			refuse("artifact transfer failed: %v", err)
			return
		}
	}
	local := fedLocalName(a.Grantee, req.App)
	lm := *m
	lm.Metadata = api.Metadata{Name: local, Owner: "federation:" + a.Grantee}
	lm.Spec.Placement.Federation = nil
	fp := &FedPlacement{Peer: a.Grantee, PeerRoot: a.GranteeRoot, Agreement: rec.Digest, Request: env.Digest(), RequestEnv: env, App: local, RemoteApp: req.App}
	if _, err := s.propose("fed-inbound", "federation:"+a.Grantee, fp); err != nil {
		refuse("%v", err)
		return
	}
	res, err := s.propose("apply", "federation:"+a.Grantee, map[string]any{"manifest": lm, "federation": fp})
	if err != nil {
		refuse("%v", err)
		return
	}
	st := s.fedStatus(req, rec)
	st.Accepted, st.Message = res.OK, res.Message
	s.writeFedStatus(w, st)
}

func (s *Server) fedStatus(req *FedRequest, rec *FedAgreementRec) FedStatus {
	local := fedLocalName(rec.A.Grantee, req.App)
	out := FedStatus{Agreement: rec.Digest, App: req.App, LocalApp: local, Revoked: rec.Revoked}
	s.fsm.Read(func(st *State) {
		app := st.Apps[local]
		if app == nil {
			out.Message = "no such placement"
			return
		}
		out.Accepted = !app.Deleted
		out.Message = fmt.Sprintf("generation %d", app.Generation)
		if app.Deleted {
			out.Message = "withdrawn"
		}
		av := s.appView(st, app, nowMs())
		for _, r := range av.Rows {
			out.Rows = append(out.Rows, FedReplicaRow{Replica: r.Replica, Node: r.Node, Desired: r.Desired, Admitted: r.Admitted, Observed: r.Observed,
				Generation: r.ObservedGen, Code: r.Code, Evidence: r.Evidence, Freshness: r.Freshness})
		}
	})
	return out
}

func (s *Server) writeFedStatus(w http.ResponseWriter, st FedStatus) {
	st.TS = nowMs()
	s.fsm.Read(func(sst *State) { st.Roster = sst.Roster })
	env, err := envelope.Sign(s.id, "", envelope.KindFedStatus, st)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, env)
}

// fetchPeerArtifact pulls an artifact from the grantee's control plane.
func (s *Server) fetchPeerArtifact(rec *FedAgreementRec, art *Artifact) error {
	if s.cas.Has(art.Manifest) {
		_, _ = s.propose("artifact", "federation:"+rec.A.Grantee, Artifact{Digest: art.Digest, Name: art.Name, Bytes: art.Bytes, Chunks: art.Chunks, Manifest: art.Manifest, Holders: []string{}})
		return nil
	}
	return errors.New("the grantee must push the artifact with the placement (not yet held here)")
}

// ------------------------------------------------------------- grantee side

// federationTick delivers outbound placements and refreshes their status.
func (s *Server) federationTick() {
	type job struct {
		out   FedOutbound
		agree FedAgreementRec
	}
	var jobs []job
	var delegation string
	var cluster string
	var artifacts map[string]*Artifact
	s.fsm.Read(func(st *State) {
		cluster = st.Cluster
		if m := st.member(s.id.ID); m != nil {
			delegation = m.Delegation
		}
		artifacts = map[string]*Artifact{}
		for d, a := range st.Artifacts {
			cp := *a
			artifacts[d] = &cp
		}
		for _, name := range sortedKeys(st.Federation.Outbound) {
			o := st.Federation.Outbound[name]
			if rec := st.Federation.Held[o.Agreement]; rec != nil {
				jobs = append(jobs, job{*o, *rec})
			}
		}
	})
	if delegation == "" {
		return
	}
	for _, j := range jobs {
		if time.Since(time.UnixMilli(j.out.StatusAt)) < 3*time.Second && j.out.Accepted {
			continue
		}
		op := "status"
		if !j.out.Accepted {
			op = "place"
		}
		req := FedRequest{Op: op, Agreement: j.out.Agreement, Grantee: cluster, App: j.out.App, Delegation: delegation, TS: nowMs()}
		if op == "place" {
			m := j.out.Manifest
			req.Manifest = &m
			if a := artifacts[manifest.Digest(m.Spec.Image)]; a != nil {
				req.Artifact = a
				s.pushArtifactToPeer(j.agree, a)
			}
		}
		env, status, err := s.callPeer(j.agree, req)
		upd := FedOutbound{App: j.out.App, Peer: j.out.Peer, Agreement: j.out.Agreement, Replicas: j.out.Replicas}
		switch {
		case err != nil:
			upd.Accepted, upd.Message = j.out.Accepted, "peer unreachable: "+err.Error()
		default:
			upd.Accepted, upd.Message, upd.LastStatus = status.Accepted, status.Message, env
			if status.Revoked {
				upd.Accepted, upd.Message = false, "agreement revoked by "+j.out.Peer+": "+status.Message
			}
		}
		if upd.Accepted != j.out.Accepted || upd.Message != j.out.Message || upd.LastStatus != nil {
			_, _ = s.propose("fed-outbound", s.id.ID, upd)
		}
	}
}

// peerHTTP reaches a grantor's API: over TLS verified against the grantor
// root CA named in the root-signed agreement, or plain HTTP when the grantor
// runs without TLS (every message is signed either way).
func peerHTTP(a FedAgreement) (string, *http.Client, error) {
	if a.GrantorCA == "" {
		return "http", &http.Client{Timeout: 30 * time.Second}, nil
	}
	conf, err := pki.ClientTLS([]byte(a.GrantorCA))
	if err != nil {
		return "", nil, fmt.Errorf("agreement grantor CA: %w", err)
	}
	return "https", &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: conf}}, nil
}

// callPeer sends a signed request to the grantor and verifies its signed answer.
func (s *Server) callPeer(rec FedAgreementRec, req FedRequest) (*envelope.Envelope, *FedStatus, error) {
	env, err := envelope.Sign(s.id, "", envelope.KindFedPlacement, req)
	if err != nil {
		return nil, nil, err
	}
	body, _ := canon.Wire(env)
	var lastErr error
	for _, api := range rec.A.GrantorAPI {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		scheme, client, err := peerHTTP(rec.A)
		if err != nil {
			cancel()
			return nil, nil, err
		}
		hr, _ := http.NewRequestWithContext(ctx, "POST", scheme+"://"+api+"/fed/v1/request", bytes.NewReader(body))
		hr.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(hr)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		cancel()
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s: %s", resp.Status, bytes.TrimSpace(data))
			continue
		}
		var out envelope.Envelope
		if err := json.Unmarshal(data, &out); err != nil {
			lastErr = err
			continue
		}
		st, err := verifyFedStatus(&out, rec.A)
		if err != nil {
			return nil, nil, err
		}
		return &out, st, nil
	}
	return nil, nil, lastErr
}

// verifyFedStatus checks a status is signed by a member of a roster signed
// by the grantor root named in the agreement.
func verifyFedStatus(env *envelope.Envelope, a FedAgreement) (*FedStatus, error) {
	var st FedStatus
	if err := env.Decode(&st); err != nil {
		return nil, err
	}
	if st.Roster == nil || st.Roster.VerifyKey(envelope.KindRoster, a.GrantorRoot) != nil {
		return nil, errors.New("peer status roster is not signed by the grantor root")
	}
	var r api.Roster
	_ = st.Roster.Decode(&r)
	for _, m := range r.Members {
		if m.ID == env.Signer && env.VerifyKey(envelope.KindFedStatus, m.Pub) == nil {
			return &st, nil
		}
	}
	return nil, errors.New("peer status is not signed by a grantor roster member")
}

// pushArtifactToPeer sends artifact objects to the grantor (only objects it
// lacks; every object is re-hashed on receipt).
func (s *Server) pushArtifactToPeer(rec FedAgreementRec, a *Artifact) {
	mb, err := s.cas.Get(a.Manifest)
	if err != nil {
		return
	}
	var m api.ArtifactManifest
	if json.Unmarshal(mb, &m) != nil {
		return
	}
	for _, id := range append([]string{a.Manifest}, m.Chunks...) {
		data, err := s.cas.Get(id)
		if err != nil {
			return
		}
		scheme, client, err := peerHTTP(rec.A)
		if err != nil {
			return
		}
		for _, api := range rec.A.GrantorAPI {
			resp, err := client.Post(scheme+"://"+api+"/fed/v1/object?agreement="+rec.Digest+"&id="+id, "application/octet-stream", bytes.NewReader(data))
			if err == nil {
				resp.Body.Close()
				break
			}
		}
	}
}

func init() {
	// Objects pushed by a grantee are accepted only for a current agreement
	// and only if they hash to their id (content addressing makes the
	// channel self-authenticating for integrity).
	extraRoutes = append(extraRoutes, func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("POST /fed/v1/object", func(w http.ResponseWriter, r *http.Request) {
			d := r.URL.Query().Get("agreement")
			valid := false
			s.fsm.Read(func(st *State) {
				rec := st.Federation.Granted[d]
				valid = rec != nil && !rec.Revoked && (rec.A.Expires == 0 || nowMs() < rec.A.Expires)
			})
			if !valid {
				writeErr(w, 403, "no current agreement")
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
			if err != nil {
				writeErr(w, 400, "%v", err)
				return
			}
			if err := s.cas.PutWithID(r.URL.Query().Get("id"), data); err != nil {
				writeErr(w, 400, "%v", err)
				return
			}
			w.WriteHeader(204)
		})
	})
}

var _ = sort.Strings
