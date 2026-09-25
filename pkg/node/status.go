package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/storage"
)

// Status is the agent's local self-report (loopback only).
type Status struct {
	Node        string               `json:"node"`
	Name        string               `json:"name"`
	Cluster     string               `json:"cluster"`
	Root        string               `json:"root"`
	Mode        string               `json:"mode"`
	ModeDetail  string               `json:"modeDetail"`
	Trusted     bool                 `json:"trusted"`
	TrustWhy    string               `json:"trustWhy"`
	Fresh       bool                 `json:"fresh"`
	SkewMs      int64                `json:"skewMs"`
	LastIndex   int64                `json:"lastIndex"`
	LastIssued  int64                `json:"lastIssued"`
	BundleAgeMs int64                `json:"bundleAgeMs"`
	Seq         int64                `json:"seq"`
	Admitted    map[string]*Admitted `json:"admitted"`
	Decisions   map[string]*decision `json:"decisions"`
	Outbox      int                  `json:"outbox"`
	LastErr     string               `json:"lastErr"`
	MeshIP      string               `json:"meshIp"`
	Ledger      int64                `json:"ledger"`
	LedgerBreak *audit.Break         `json:"ledgerBreak"`
	ClockOffset int64                `json:"clockOffset"`
	Storage     any                  `json:"storage"`
	Edge        any                  `json:"edge"`
	PeerIPs     map[string]string    `json:"peerIps"`
	BindingOK   map[string]bool      `json:"bindingOk"`
	Refused     string               `json:"refused"`
}

// Snapshot returns the current status.
func (a *Agent) Snapshot() Status {
	a.mu.RLock()
	defer a.mu.RUnlock()
	s := Status{Node: a.st.NodeID, Name: a.cfg.Name, Cluster: a.st.Cluster, Root: a.st.Root, Mode: a.mode, ModeDetail: a.modeDetail,
		Trusted: a.trusted, TrustWhy: a.trustWhy, Fresh: a.fresh, SkewMs: a.skewMs, LastIndex: a.st.LastIndex, LastIssued: a.st.LastIssued,
		Seq: a.st.Seq, Admitted: map[string]*Admitted{}, Decisions: map[string]*decision{}, Outbox: a.outbox.size(), LastErr: a.lastErr,
		ClockOffset: a.clockOff.Load(), Storage: a.cas.Stats()}
	if !a.lastBundleAt.IsZero() {
		s.BundleAgeMs = time.Since(a.lastBundleAt).Milliseconds()
	}
	for k, v := range a.st.Admitted {
		cp := *v
		s.Admitted[k] = &cp
	}
	for k, v := range a.decisions {
		cp := *v
		s.Decisions[k] = &cp
	}
	if a.bundle != nil {
		s.MeshIP = a.bundle.Node.MeshIP
	}
	s.Ledger, _ = a.journal.Head()
	s.LedgerBreak = a.journal.Corrupt()
	if a.edge != nil {
		s.Edge = a.edge.observe()
	}
	return s
}

// Status returns the snapshot plus mesh diagnostics. Mesh state is read
// before the main lock (lock order: meshMu is never taken under mu).
func (a *Agent) Status() Status {
	a.meshMu.RLock()
	ips, ok, refused := map[string]string{}, map[string]bool{}, a.refused
	for k, v := range a.peerIPs {
		ips[k] = v
	}
	for k, v := range a.bindOK {
		ok[k] = v
	}
	a.meshMu.RUnlock()
	s := a.Snapshot()
	s.PeerIPs, s.BindingOK, s.Refused = ips, ok, refused
	return s
}

func (a *Agent) startStatus() error {
	ln, err := net.Listen("tcp", a.cfg.StatusListen)
	if err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(a.cfg.DataDir, "status.addr"), []byte(ln.Addr().String()), 0o600)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, a.Status()) })
	mux.HandleFunc("GET /ledger", func(w http.ResponseWriter, r *http.Request) {
		from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
		writeJSON(w, map[string]any{"entries": a.journal.Entries(from, 0), "corrupt": a.journal.Corrupt()})
	})
	if a.cfg.Chaos {
		mux.HandleFunc("POST /chaos", a.handleChaos)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-a.stopc
		_ = srv.Close()
	}()
	go srv.Serve(ln)
	return nil
}

// ChaosRequest injects a fault (only with --chaos).
type ChaosRequest struct {
	Fault      string `json:"fault"` // clock-offset | drop-cp | enospc | corrupt-chunk | fail-health | crash-mid-push
	Value      int64  `json:"value"`
	On         bool   `json:"on"`
	Assignment string `json:"assignment"`
	Object     string `json:"object"`
	Volume     string `json:"volume"`
}

func (a *Agent) handleChaos(w http.ResponseWriter, r *http.Request) {
	var req ChaosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var detail string
	switch req.Fault {
	case "clock-offset":
		a.clockOff.Store(req.Value)
		detail = fmt.Sprintf("host clock offset set to %dms", req.Value)
	case "drop-cp":
		a.chaos.dropCP.Store(req.On)
		detail = fmt.Sprintf("control-plane connectivity dropped=%v", req.On)
	case "enospc":
		a.cas.InjectENOSPC(req.On)
		detail = fmt.Sprintf("storage ENOSPC injection=%v", req.On)
	case "corrupt-chunk":
		ids := a.cas.List()
		target := req.Object
		if target == "" && req.Volume != "" {
			// A data chunk of the volume's committed snapshot on this host.
			if d := a.duty(req.Volume); d != nil && d.Committed != nil {
				if m, err := storage.LoadManifest(a.cas, d.Committed.ID); err == nil {
					for _, f := range m.Files {
						if len(f.Chunks) > 0 {
							target = f.Chunks[len(f.Chunks)/2]
						}
					}
				}
			}
		}
		if target == "" && len(ids) > 0 {
			target = ids[len(ids)/2]
		}
		if err := a.cas.CorruptForTest(target); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		detail = "flipped a byte in " + target
		writeJSON(w, map[string]any{"ok": true, "detail": detail, "object": target})
		a.record("chaos-inject", "node/"+a.st.NodeID, 0, detail)
		return
	case "fail-health":
		a.chaos.failHealth.Store(req.Assignment, req.On)
		detail = fmt.Sprintf("health failure for %s=%v", req.Assignment, req.On)
	default:
		http.Error(w, "unknown fault", 400)
		return
	}
	a.record("chaos-inject", "node/"+a.st.NodeID, 0, detail)
	writeJSON(w, map[string]any{"ok": true, "detail": detail})
}

// ErrNotJoined is returned by offline commands on an unjoined data dir.
var ErrNotJoined = errors.New("host has not joined a cluster")
