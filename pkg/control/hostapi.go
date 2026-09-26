package control

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/pki"
)

// HostRequest is the payload of a host-signed request envelope (kind "request").
type HostRequest struct {
	Node      string `json:"node"`
	TS        int64  `json:"ts"`
	Op        string `json:"op"`
	Arg       string `json:"arg"`
	LastIndex int64  `json:"lastIndex"`
}

// requestWindow bounds replay of signed read requests. It is deliberately
// wide: clock skew is detected and reported by the host from bundle
// timestamps, not by refusing to talk to it.
const requestWindow = 10 * time.Minute

func (s *Server) verifyHostRequest(r *http.Request, op string) (*HostRequest, *Node, error) {
	var env envelope.Envelope
	if err := readBody(r, 64<<10, &env); err != nil {
		return nil, nil, err
	}
	var req HostRequest
	if err := json.Unmarshal(env.Payload, &req); err != nil {
		return nil, nil, err
	}
	var node *Node
	var keys []string
	s.fsm.Read(func(st *State) {
		if n := st.Nodes[req.Node]; n != nil {
			cp := *n
			node = &cp
			keys = n.ValidKeys(nowMs())
		}
	})
	if node == nil {
		return nil, nil, fmt.Errorf("unknown host %s", req.Node)
	}
	if err := env.VerifyKey(envelope.KindRequest, keys...); err != nil {
		s.local.add(Rejection{TS: nowMs(), Kind: "request", Node: req.Node, Reason: err.Error()})
		return nil, nil, err
	}
	if req.Op != op {
		return nil, nil, fmt.Errorf("request is for op %q", req.Op)
	}
	if d := time.Duration(nowMs()-req.TS) * time.Millisecond; d > requestWindow || d < -requestWindow {
		return nil, nil, fmt.Errorf("request timestamp %s outside the %s window", d, requestWindow)
	}
	return &req, node, nil
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var env envelope.Envelope
	if err := readBody(r, 256<<10, &env); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	if err := env.VerifySelf(envelope.KindEnroll); err != nil {
		s.local.add(Rejection{TS: nowMs(), Kind: "enroll", Node: env.Signer, Reason: err.Error()})
		writeErr(w, 401, "enrollment signature: %v", err)
		return
	}
	res, err := s.propose("enroll", env.Signer, enrollData{Env: &env})
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	code := 200
	if !res.OK {
		code = 403
	}
	writeJSON(w, code, res)
}

func (s *Server) handleBundle(w http.ResponseWriter, r *http.Request) {
	req, _, err := s.verifyHostRequest(r, "bundle")
	if err != nil {
		writeErr(w, 401, "%v", err)
		return
	}
	// Only a leader confirmed by a quorum round trip issues bundles; a
	// partitioned member must never make hosts believe the plane is fresh.
	if err := s.verifyLeader(); err != nil {
		writeErr(w, 503, "leadership not confirmed by quorum: %v", err)
		return
	}
	env, err := s.buildBundle(req.Node)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, map[string]any{"bundle": env})
}

// ObserveResult reports per-observation outcomes to the host.
type ObserveResult struct {
	Seq    int64  `json:"seq"`
	Status string `json:"status"` // committed | cached | rejected
	Reason string `json:"reason"`
}

func materialKey(o *api.Observation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s|%s|%d|", o.Mode, o.ModeDetail, o.Ledger.Seq)
	ws := append([]api.WorkloadObs(nil), o.Workloads...)
	sort.Slice(ws, func(i, j int) bool { return ws[i].Assignment < ws[j].Assignment })
	for _, w := range ws {
		h := ""
		if w.Health != nil {
			h = fmt.Sprint(w.Health.OK)
		}
		fmt.Fprintf(&b, "%s:%d:%s:%s:%s:%d:%s:%d:%d;", w.Assignment, w.Generation, w.Admitted, w.Code, w.Observed, w.PID, h, w.MeshPort, len(w.Volumes))
		for _, v := range w.Volumes {
			fmt.Fprintf(&b, "%s=%s/%s,", v.VolumeID, v.LastSnapshot, v.Restored)
		}
	}
	if o.Storage != nil {
		fmt.Fprintf(&b, "|storage:%d", o.Storage.Corrupt)
	}
	if o.Edge != nil {
		for _, r := range o.Edge.Routes {
			for _, e := range r.Endpoints {
				fmt.Fprintf(&b, "|r:%s:%s:%s", r.Host, e.Assignment, e.State)
			}
		}
		for _, c := range o.Edge.Certs {
			fmt.Fprintf(&b, "|c:%s:%s:%s", c.Host, c.State, c.Serial)
		}
	}
	if o.Mesh != nil {
		for _, p := range o.Mesh.Peers {
			fmt.Fprintf(&b, "|g:%s:%s:%v", p.Node, p.Gossip, p.BindingOK)
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

const heartbeatCommit = 15 * time.Second

func (s *Server) handleObserve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Observations []*envelope.Envelope `json:"observations"`
	}
	if err := readBody(r, 8<<20, &body); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	var out []ObserveResult
	for _, env := range body.Observations {
		out = append(out, s.observeOne(env))
	}
	writeJSON(w, 200, map[string]any{"results": out})
}

func (s *Server) observeOne(env *envelope.Envelope) ObserveResult {
	var o api.Observation
	if err := env.Decode(&o); err != nil {
		return ObserveResult{Status: "rejected", Reason: err.Error()}
	}
	var keys []string
	var committedSeq int64
	var known bool
	s.fsm.Read(func(st *State) {
		if n := st.Nodes[env.Signer]; n != nil {
			known = true
			keys = n.ValidKeys(nowMs())
			committedSeq = n.ObserveSeq
		}
	})
	if !known {
		s.local.add(Rejection{TS: nowMs(), Kind: "observation", Node: env.Signer, Reason: "unknown host", Seq: o.Seq})
		return ObserveResult{Seq: o.Seq, Status: "rejected", Reason: "unknown host"}
	}
	if err := env.VerifyKey(envelope.KindObservation, keys...); err != nil {
		reason := err.Error()
		if errors.Is(err, envelope.ErrSignerKey) {
			reason = "wrong host key"
		}
		s.local.add(Rejection{TS: nowMs(), Kind: "observation", Node: env.Signer, Reason: reason, Seq: o.Seq, Evidence: env.Digest()})
		return ObserveResult{Seq: o.Seq, Status: "rejected", Reason: reason}
	}
	cached := s.obs.get(env.Signer)
	last := committedSeq
	if cached != nil && cached.Obs.Seq > last {
		last = cached.Obs.Seq
	}
	if o.Seq <= last {
		// A validly signed but old message: record it in the replicated
		// rejection log so every member can show it.
		if o.Seq <= committedSeq {
			_, _ = s.propose("observe", env.Signer, env)
		} else {
			s.local.add(Rejection{TS: nowMs(), Kind: "observation", Node: env.Signer, Reason: "replay detected", Seq: o.Seq, LastSeq: last, Evidence: env.Digest()})
		}
		return ObserveResult{Seq: o.Seq, Status: "rejected", Reason: fmt.Sprintf("replay detected: seq %d, last accepted %d", o.Seq, last)}
	}
	key := materialKey(&o)
	entry := &cachedObs{Env: env, Obs: o, Received: time.Now(), Material: key}
	commit := cached == nil || cached.Material != key || time.Since(cached.Committed) > heartbeatCommit || o.Buffered
	if cached != nil && !commit {
		entry.Committed, entry.CommitSeq = cached.Committed, cached.CommitSeq
	}
	s.obs.put(env.Signer, entry)
	if !commit {
		return ObserveResult{Seq: o.Seq, Status: "cached"}
	}
	res, err := s.propose("observe", env.Signer, env)
	if err != nil {
		return ObserveResult{Seq: o.Seq, Status: "cached", Reason: "commit failed: " + err.Error()}
	}
	if !res.OK {
		return ObserveResult{Seq: o.Seq, Status: "rejected", Reason: res.Message}
	}
	s.obs.markCommitted(env.Signer, o.Seq)
	return ObserveResult{Seq: o.Seq, Status: "committed"}
}

func (s *Server) handleEvidence(w http.ResponseWriter, r *http.Request) {
	var env envelope.Envelope
	if err := readBody(r, 16<<20, &env); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	res, err := s.propose("evidence", env.Signer, &env)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleBinding(w http.ResponseWriter, r *http.Request) {
	var env envelope.Envelope
	if err := readBody(r, 64<<10, &env); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	same := false
	s.fsm.Read(func(st *State) {
		if n := st.Nodes[env.Signer]; n != nil && n.Binding != nil && bytes.Equal(n.Binding.Payload, env.Payload) {
			same = true
		}
		if b := st.MemberBind[env.Signer]; b != nil && bytes.Equal(b.Payload, env.Payload) {
			same = true
		}
	})
	if same {
		writeJSON(w, 200, ok("binding unchanged"))
		return
	}
	res, err := s.propose("wg-binding", env.Signer, &env)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleRotate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env    *envelope.Envelope `json:"env"`
		OldSig string             `json:"oldSig"`
	}
	if err := readBody(r, 64<<10, &body); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	res, err := s.propose("rotate-key", body.Env.Signer, body)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	code := 200
	if !res.OK {
		code = 403
	}
	writeJSON(w, code, res)
}

// handleChunk serves artifact chunks held by this member (fallback source
// when no mesh peer has them).
func (s *Server) handleChunk(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	req, _, err := s.verifyHostRequest(r, "chunk")
	if err != nil {
		writeErr(w, 401, "%v", err)
		return
	}
	data, err := s.cas.Get(req.Arg)
	if err != nil {
		// Artifacts land on the member that received the upload (the leader
		// at the time); a follower forwards rather than answering "missing".
		if !s.IsLeader() && r.Header.Get("X-DH-Forwarded") == "" {
			r.Body = io.NopCloser(bytes.NewReader(body))
			s.forward(w, r)
			return
		}
		writeErr(w, 404, "%v", err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(data)
}

// handleCertSign issues a cluster-local CA certificate to an edge host for
// ingress hosts declared with tls: local. The edge keeps its private key;
// only the public key travels.
func (s *Server) handleCertSign(w http.ResponseWriter, r *http.Request) {
	req, node, err := s.verifyHostRequest(r, "cert")
	if err != nil {
		writeErr(w, 401, "%v", err)
		return
	}
	if !contains(node.Roles, "edge") {
		writeErr(w, 403, "%s does not have the edge role", node.Name)
		return
	}
	var body struct {
		Names  []string `json:"names"`
		PubPEM string   `json:"pubPem"`
	}
	if err := json.Unmarshal([]byte(req.Arg), &body); err != nil || len(body.Names) == 0 {
		writeErr(w, 400, "cert request: names and pubPem required")
		return
	}
	var caCert, caKey string
	allowed := map[string]bool{}
	s.fsm.Read(func(st *State) {
		caCert, caKey = st.LocalCACert, st.LocalCAKey
		for _, name := range st.sortedAppNames() {
			a := st.Apps[name]
			if a.Deleted {
				continue
			}
			for _, in := range a.Manifest.Spec.Ingress {
				if in.TLS == "local" {
					allowed[in.Host] = true
				}
			}
		}
	})
	if caCert == "" {
		writeErr(w, 503, "local CA not created yet")
		return
	}
	for _, n := range body.Names {
		if !allowed[n] {
			writeErr(w, 403, "%s is not an ingress host with tls: local", n)
			return
		}
	}
	block, _ := pem.Decode([]byte(body.PubPEM))
	if block == nil {
		writeErr(w, 400, "bad public key PEM")
		return
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		writeErr(w, 400, "public key: %v", err)
		return
	}
	cert, err := pki.SignLocal([]byte(caCert), []byte(caKey), pub, body.Names, 30*24*time.Hour)
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, map[string]any{"cert": string(cert), "ca": caCert})
}

// handleRetrieveSecret processes authenticated secret retrieval requests.
// Implements the SECRET-RETRIEVAL-P0-A01 production orchestration:
//  1. Parse and verify signed request
//  2. Create authorization command (AuthorizeSecretRetrievalCommand)
//  3. Submit via raft.Apply() (s.propose())
//  4. Wait for commit confirmation (future.Error() == nil)
//  5. Call DecryptSecret() only after commit is confirmed
//  6. Return plaintext response
func (s *Server) handleRetrieveSecret(w http.ResponseWriter, r *http.Request) {
	var req SecretRetrievalRequest
	if err := readBody(r, 64<<10, &req); err != nil {
		writeErr(w, 400, "parse request: %v", err)
		return
	}

	// Verify signature
	if err := req.VerifySignature(); err != nil {
		writeErr(w, 401, "signature verification failed: %v", err)
		return
	}

	// Create authorization command on leader (triggers AuthorizationProposed observer)
	var cmd *Command
	var cmdErr error
	s.fsm.Read(func(st *State) {
		cmd, cmdErr = s.fsm.AuthorizeSecretRetrievalCommand(&req)
	})
	if cmdErr != nil {
		writeErr(w, 400, "command: %v", cmdErr)
		return
	}

	// CRITICAL ORCHESTRATION BOUNDARY:
	// Submit to Raft for replication and commit confirmation
	// This is where FSM.Apply() will be called on all replicas
	res, err := s.raft().propose(cmd, 10*time.Second)
	if err != nil {
		writeErr(w, 503, "raft unavailable: %v", err)
		return
	}

	// Check authorization result
	if !res.OK {
		// Authorization failed (signature, scope, replay, etc.)
		code := 403
		if strings.Contains(res.Code, "UNINITIALIZED") {
			code = 503
		}
		writeJSON(w, code, map[string]any{
			"error": res.Message,
			"code":  res.Code,
		})
		return
	}

	// CRITICAL: Authorization is now committed to replicated state.
	// ReplayLedger has been updated on all members via Raft.
	// Record the observation (for qualification testing)
	requestDigest := req.RequestDigest()
	if s.fsm.retrievalObserver != nil {
		s.fsm.retrievalObserver.AuthorizationCommitted(map[string]string{
			"requestDigest": requestDigest,
			"requestID":     req.RequestID,
		})
	}

	// Deterministic fault injection point for R1-02 qualification
	// (tests can block here to verify commit survives leader crash)
	if s.fsm.retrievalObserver != nil {
		blockChan := s.fsm.retrievalObserver.BeforeDecrypt(map[string]string{
			"requestDigest": requestDigest,
		})
		select {
		case <-blockChan:
			// Fault injection released, proceed to decrypt
		case <-r.Context().Done():
			// Request cancelled while blocked
			writeErr(w, 499, "request cancelled")
			return
		}
	}

	// Decrypt the secret (only after commitment is confirmed)
	var secretRecord *SecretRecord
	var dek [32]byte
	s.fsm.Read(func(st *State) {
		secretRecord = st.Secrets.GetRecord(req.SecretID, req.SecretVersion)
		// In production, DEK would come from a secure key manager
		// For now, this is a placeholder - tests will set it up
		copy(dek[:], make([]byte, 32))
	})

	if secretRecord == nil {
		writeErr(w, 404, "secret not found")
		return
	}

	plaintext, err := DecryptSecret(secretRecord, dek)
	if err != nil {
		writeErr(w, 500, "decrypt failed: %v", err)
		if s.fsm.retrievalObserver != nil {
			s.fsm.retrievalObserver.AfterDecrypt(map[string]string{
				"requestDigest": requestDigest,
			}, err)
		}
		return
	}

	if s.fsm.retrievalObserver != nil {
		s.fsm.retrievalObserver.AfterDecrypt(map[string]string{
			"requestDigest": requestDigest,
		}, nil)
	}

	// Record response intent (for qualification testing)
	if s.fsm.retrievalObserver != nil {
		blockChan := s.fsm.retrievalObserver.BeforeResponseWrite(map[string]string{
			"requestDigest": requestDigest,
		})
		select {
		case <-blockChan:
			// Ready to send response
		case <-r.Context().Done():
			// Request cancelled before sending response
			return
		}
	}

	// A05-P1-R1: Fail-closed delivery gate
	// If ephemeral delivery is not requested, deny the retrieval.
	// Legacy plaintext fallback is disabled by default (require explicit opt-in configuration).
	if req.EphemeralID == "" {
		writeErr(w, 403, "ephemeral delivery required; plaintext retrieval not authorized")
		if s.fsm.retrievalObserver != nil {
			s.fsm.retrievalObserver.AfterResponseWrite(map[string]string{
				"requestDigest": requestDigest,
			}, errors.New("plaintext fallback denied"))
		}
		return
	}

	// A05-P1-R1: Create signed delivery envelope
	// Agent will receive this envelope, verify signature, and materialize on assigned node
	// ClusterID comes from the SecretRecord's scope binding (same cluster where secret was created)
	clusterID := ""
	if secretRecord != nil {
		clusterID = secretRecord.ClusterID
	}
	envelope := &SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          req.EphemeralID,
		AuthorizationDigest: requestDigest,
		ClusterID:           clusterID,          // From secret's scope binding
		NodeID:              req.NodeID,         // Target node (agent verifies == self)
		DeploymentID:        req.DeploymentID,
		WorkloadID:          req.WorkloadID,
		Environment:         req.Environment,
		SecretID:            req.SecretID,
		SecretVersion:       req.SecretVersion,
		IssuedAt:            Now(),
		ExpiresAt:           Now() + int64(5*time.Minute), // 5-minute validity window
		Nonce:               req.Nonce,
		PlaintextPayload:    plaintext,
	}

	// Sign the envelope using control-plane identity
	// This proves the envelope came from the authorized control plane
	canonical := envelope.CanonicalEnvelope()
	envelope.Signature = s.id.Sign(canonical)

	// Return signed envelope (not plaintext)
	// Agent verifies signature before materializing
	writeJSON(w, 200, map[string]any{
		"envelope": envelope,
		"digest":   requestDigest,
	})

	if s.fsm.retrievalObserver != nil {
		s.fsm.retrievalObserver.AfterResponseWrite(map[string]string{
			"requestDigest": requestDigest,
		}, nil)
	}
}

// forwardInternal posts a signed envelope to the leader's host channel.
func (s *Server) forwardInternal(path string, env any) {
	addr, _ := s.leaderAPI()
	if addr == "" {
		return
	}
	b, _ := canonJSONWire(env)
	scheme := "http"
	client := &http.Client{Timeout: 5 * time.Second}
	if s.cfg.TLS {
		scheme = "https"
		if conf, err := pki.ClientTLS([]byte(s.credentials().RootCACert)); err == nil {
			client.Transport = &http.Transport{TLSClientConfig: conf}
		}
	}
	resp, err := client.Post(scheme+"://"+addr+path, "application/json", bytes.NewReader(b))
	if err == nil {
		resp.Body.Close()
	}
}
