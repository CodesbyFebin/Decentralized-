package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/mesh"
	"decentralized.host/pkg/peer"
	"decentralized.host/pkg/storage"
)

func (a *Agent) meshLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	lastRTT := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.syncMesh()
		if time.Since(lastRTT) > 10*time.Second {
			lastRTT = time.Now()
			a.measureRTT()
		}
	}
}

func (a *Agent) syncMesh() {
	if a.cfg.MeshListen == "" {
		return
	}
	a.mu.RLock()
	b := a.bundle
	a.mu.RUnlock()
	if b == nil || b.Node.MeshIP == "" || b.Node.Status == "pending" {
		return
	}
	a.meshMu.Lock()
	if a.dev == nil {
		key, err := mesh.LoadOrCreateKey(filepath.Join(a.cfg.DataDir, "wg.key"))
		if err != nil {
			a.meshMu.Unlock()
			return
		}
		_, portStr, _ := net.SplitHostPort(a.cfg.MeshListen)
		port, _ := strconv.Atoi(portStr)
		dev, err := mesh.Start(key, b.Node.MeshIP, port)
		if err != nil {
			a.meshMu.Unlock()
			a.log.Printf("mesh: %v", err)
			return
		}
		a.dev = dev
		srv, err := a.startPeerServer(dev)
		if err == nil {
			a.peerSrv = srv
		}
		g, err := mesh.StartGossip(dev, a.st.NodeID)
		if err == nil {
			a.gossip = g
		}
		go a.record("mesh-up", "node/"+a.st.NodeID, 0, fmt.Sprintf("wireguard-go device %s on udp %d (key %s)", b.Node.MeshIP, port, short(key.PublicString())))
	}
	dev := a.dev
	var peers []mesh.PeerConfig
	ips := map[string]string{}
	info := map[string]api.Peer{}
	ok := map[string]bool{}
	for _, p := range b.Peers {
		wb, valid := verifyPeerBinding(p)
		info[p.ID] = p
		ok[p.ID] = valid
		if !valid {
			continue // a binding not signed by the peer's own key is never configured
		}
		ips[p.MeshIP] = p.ID
		peers = append(peers, mesh.PeerConfig{Node: p.ID, WGPub: wb.WGPub, Endpoint: wb.Endpoint, MeshIP: wb.MeshIP})
	}
	a.peerIPs, a.peerInfo, a.bindOK = ips, info, ok
	gossip := a.gossip
	a.meshMu.Unlock()
	_ = dev.SetPeers(peers)
	// Publish our own binding if the control plane does not hold the current one.
	adv := a.cfg.MeshAdvertise
	if adv == "" {
		adv = a.cfg.MeshListen
	}
	want := api.WGBinding{Node: a.st.NodeID, WGPub: dev.PublicKey(), Endpoint: adv, MeshIP: b.Node.MeshIP}
	if env := a.bindingToSend(want); env != nil {
		_ = a.cp.post("/v1/binding", env, nil)
	}
	if gossip != nil {
		var addrs []string
		have := map[string]bool{}
		for _, m := range gossip.Members() {
			if m.State == "alive" {
				have[m.Name] = true
			}
		}
		for _, p := range peers {
			if !have[p.Node] && !strings.HasPrefix(info[p.Node].Status, "control") {
				addrs = append(addrs, fmt.Sprintf("%s:%d", p.MeshIP, mesh.GossipPort))
			}
		}
		// One join in flight at a time: a join can take up to the transport
		// timeout, and piling them up every tick worsens a slow host.
		if len(addrs) > 0 && a.joining.CompareAndSwap(false, true) {
			go func() {
				defer a.joining.Store(false)
				gossip.Join(addrs)
			}()
		}
	}
}

// bindingToSend returns the signed binding to publish, or nil. A binding is
// re-signed only when key, endpoint or address change; the identical
// envelope is re-sent every 30s (the control plane ignores exact repeats).
func (a *Agent) bindingToSend(want api.WGBinding) *envelope.Envelope {
	a.meshMu.Lock()
	defer a.meshMu.Unlock()
	key := fmt.Sprintf("%s|%s|%s", want.WGPub, want.Endpoint, want.MeshIP)
	if a.sentBinding == key && a.sentEnv != nil {
		if time.Since(a.sentBindingAt) < 30*time.Second {
			return nil
		}
		a.sentBindingAt = time.Now()
		return a.sentEnv
	}
	want.TS = a.now()
	env, err := a.sign(envelope.KindWGBinding, want)
	if err != nil {
		return nil
	}
	a.sentBinding, a.sentEnv, a.sentBindingAt = key, env, time.Now()
	return env
}

func (a *Agent) measureRTT() {
	pc := a.peerClient(3 * time.Second)
	if pc == nil {
		return
	}
	a.meshMu.RLock()
	targets := map[string]string{}
	for id, p := range a.peerInfo {
		if a.bindOK[id] {
			targets[id] = p.MeshIP
		}
	}
	a.meshMu.RUnlock()
	var wg sync.WaitGroup
	var mu sync.Mutex
	res := map[string][2]int64{}
	for id, ip := range targets {
		wg.Add(1)
		go func(id, ip string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			d, err := pc.Ping(ctx, ip)
			mu.Lock()
			if err == nil {
				res[id] = [2]int64{d.Microseconds(), time.Now().UnixMilli()}
			} else {
				res[id] = [2]int64{-1, 0}
			}
			mu.Unlock()
		}(id, ip)
	}
	wg.Wait()
	a.meshMu.Lock()
	a.rtt = res
	a.meshMu.Unlock()
}

func (a *Agent) meshObs() *api.MeshObs {
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	if a.dev == nil {
		return &api.MeshObs{Device: "none", Gossip: "none", Peers: []api.PeerObs{}}
	}
	m := &api.MeshObs{Device: "wireguard-go/netstack", WGPub: a.dev.PublicKey(), ListenPort: int64(a.dev.ListenPort()), MeshIP: a.dev.IP(), Gossip: "none", Peers: []api.PeerObs{}}
	gstate := map[string]string{}
	if a.gossip != nil {
		m.Gossip = "memberlist"
		for _, g := range a.gossip.Members() {
			gstate[g.Name] = g.State
			// Gossip claims are cross-checked against verified bindings.
			if id := a.peerIPs[g.Addr]; id != "" && id != g.Name {
				gstate[g.Name] = "identity-mismatch"
			}
		}
		m.Members = int64(a.gossip.NumMembers())
	}
	stats := map[string]mesh.PeerStats{}
	for _, s := range a.dev.Stats() {
		stats[s.Node] = s
	}
	for _, id := range sortedKeys(a.peerInfo) {
		p := a.peerInfo[id]
		po := api.PeerObs{Node: id, BindingOK: a.bindOK[id], Gossip: "unknown", RTTUs: -1}
		if s, ok := stats[id]; ok {
			po.WGPub, po.Endpoint, po.RxBytes, po.TxBytes = s.WGPub, s.Endpoint, s.RxBytes, s.TxBytes
			if !s.LastHandshake.IsZero() {
				po.LastHandshake = s.LastHandshake.UnixMilli()
			}
		}
		if g, ok := gstate[id]; ok {
			po.Gossip = g
		} else if strings.HasPrefix(p.Status, "control") {
			po.Gossip = "not-a-gossip-member"
		}
		if r, ok := a.rtt[id]; ok && r[1] > 0 {
			po.RTTUs, po.RTTAt = r[0], r[1]
		}
		m.Peers = append(m.Peers, po)
	}
	return m
}

// ------------------------------------------------------------ peer server

type peerServer struct {
	srv *http.Server
}

func (p *peerServer) close() { _ = p.srv.Close() }

func (a *Agent) peerOf(r *http.Request) (string, api.Peer, bool) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	id := a.peerIPs[host]
	if id == "" {
		return "", api.Peer{}, false
	}
	return id, a.peerInfo[id], true
}

func (a *Agent) startPeerServer(dev *mesh.Device) (*peerServer, error) {
	ln, err := dev.ListenTCP(peer.Port)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	guard := func(cpOnly bool, h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, p, ok := a.peerOf(r)
			if !ok {
				http.Error(w, peer.ErrUnauthorized.Error(), http.StatusForbidden)
				return
			}
			if cpOnly && !contains(p.Roles, "control-plane") {
				http.Error(w, "only control-plane members may call this", http.StatusForbidden)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("GET /peer/v1/ping", guard(false, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"node":%q,"ts":%d}`, a.st.NodeID, a.now())
	}))
	mux.HandleFunc("GET /peer/v1/chunks/{id}", guard(false, func(w http.ResponseWriter, r *http.Request) {
		data, err := a.cas.Get(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(data)
	}))
	mux.HandleFunc("PUT /peer/v1/chunks/{id}", guard(false, func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(io.LimitReader(r.Body, storage.MaxChunk*4+1<<20))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := a.cas.PutWithID(r.PathValue("id"), data); err != nil {
			code := 400
			if err == storage.ErrQuota {
				code = http.StatusInsufficientStorage
			}
			http.Error(w, err.Error(), code)
			return
		}
		w.WriteHeader(204)
	}))
	mux.HandleFunc("GET /peer/v1/volumes/buckets", guard(false, func(w http.ResponseWriter, r *http.Request) {
		ids, present := a.snapshotPresence(r.URL.Query().Get("snapshot"))
		rep := peer.BucketsReply{Root: storage.MerkleRoot(present), Present: int64(len(present)), Buckets: storage.Buckets(present)}
		_ = ids
		writeJSON(w, rep)
	}))
	mux.HandleFunc("GET /peer/v1/volumes/bucket", guard(false, func(w http.ResponseWriter, r *http.Request) {
		_, present := a.snapshotPresence(r.URL.Query().Get("snapshot"))
		b, _ := strconv.Atoi(r.URL.Query().Get("bucket"))
		var out []string
		for _, id := range present {
			if storage.BucketOf(id) == b {
				out = append(out, id)
			}
		}
		writeJSON(w, out)
	}))
	mux.HandleFunc("GET /peer/v1/ledger", guard(true, func(w http.ResponseWriter, r *http.Request) {
		from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 5000 {
			limit = 1000
		}
		entries := a.journal.Entries(from, limit)
		rep := peer.LedgerReply{Entries: entries, Corrupt: a.journal.Corrupt()}
		seq, hash := a.journal.Head()
		if env, err := a.sign(envelope.KindCheckpoint, audit.CheckpointPayload{Ledger: a.st.NodeID, Seq: seq, Hash: hash, TS: a.now()}); err == nil {
			rep.Checkpoint = env
		}
		writeJSON(w, rep)
	}))
	mux.HandleFunc("GET /peer/v1/logs", guard(true, func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("assignment")
		tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
		if tail <= 0 || tail > 5000 {
			tail = 200
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, a.logTail(id, tail))
	}))
	mux.HandleFunc("POST /peer/v1/exec", guard(true, func(w http.ResponseWriter, r *http.Request) {
		var req peer.ExecRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, a.exec(req))
	}))
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	return &peerServer{srv: srv}, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// exec runs a command in a workload only if host policy allows exec and the
// operator presents a capability anchored in the root this host pinned.
func (a *Agent) exec(req peer.ExecRequest) peer.ExecReply {
	refuse := func(why string) peer.ExecReply {
		a.record("exec-refuse", "app/"+req.Assignment, 0, why)
		return peer.ExecReply{ExitCode: -1, Refused: why}
	}
	if !a.pol.AllowExec {
		return refuse("host policy allowExec=false")
	}
	tok, err := capability.Decode(req.Capability)
	if err != nil {
		return refuse("no operator capability: " + err.Error())
	}
	res := capability.Verify(tok, []string{a.st.Root}, capability.Request{Action: "api.admin", Resource: "cluster/" + a.st.Cluster, Now: a.now()})
	if !res.OK {
		return refuse("operator capability: " + res.Reason)
	}
	a.mu.RLock()
	ad := a.st.Admitted[req.Assignment]
	a.mu.RUnlock()
	if ad == nil || ad.Stopped {
		return refuse("no running workload " + req.Assignment)
	}
	code, out, err := a.runtimeFor(ad.Runtime).Exec(ad.Inst, req.Argv, 30*time.Second)
	if err != nil {
		return refuse(err.Error())
	}
	a.record("exec", "app/"+req.Assignment, ad.Generation, fmt.Sprintf("argv %q exit %d (operator key %s)", req.Argv, code, short(res.Signers[len(res.Signers)-1])))
	return peer.ExecReply{ExitCode: code, Output: out}
}

func (a *Agent) logTail(id string, n int) string {
	a.mu.RLock()
	ad := a.st.Admitted[id]
	a.mu.RUnlock()
	if ad != nil && ad.Runtime == "docker" && ad.Inst.ContainerID != "" {
		return a.docker.Logs(ad.Inst, n)
	}
	b, err := readTail(filepath.Join(a.cfg.DataDir, "logs", safeName(id)+".log"), n)
	if err != nil {
		return "no logs: " + err.Error()
	}
	return b
}
