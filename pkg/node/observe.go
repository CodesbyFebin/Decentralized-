package node

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
)

// outbox buffers signed observations while the control plane is
// unreachable and flushes them in order on reconnection.
type outbox struct {
	mu   sync.Mutex
	path string
	list []*envelope.Envelope
}

const outboxCap = 500

func openOutbox(path string) (*outbox, error) {
	o := &outbox{path: path}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return o, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for sc.Scan() {
		var env envelope.Envelope
		if json.Unmarshal(sc.Bytes(), &env) == nil {
			o.list = append(o.list, &env)
		}
	}
	return o, nil
}

func (o *outbox) persist() {
	tmp := o.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}
	w := bufio.NewWriter(f)
	for _, env := range o.list {
		b, _ := json.Marshal(env)
		w.Write(append(b, '\n'))
	}
	w.Flush()
	f.Sync()
	f.Close()
	_ = os.Rename(tmp, o.path)
}

func (o *outbox) add(env *envelope.Envelope) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.list = append(o.list, env)
	if len(o.list) > outboxCap {
		o.list = o.list[len(o.list)-outboxCap:]
	}
	o.persist()
}

func (o *outbox) pending() []*envelope.Envelope {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]*envelope.Envelope(nil), o.list...)
}

func (o *outbox) drop(n int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if n > len(o.list) {
		n = len(o.list)
	}
	o.list = o.list[n:]
	o.persist()
}

func (o *outbox) size() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.list)
}

// buildObservation composes the host's view of itself.
func (a *Agent) buildObservation(buffered bool) api.Observation {
	a.mu.Lock()
	a.st.Seq++
	seq := a.st.Seq
	a.mu.Unlock()
	a.saveState() // persist the sequence high-water mark before it is signed
	a.mu.RLock()
	defer a.mu.RUnlock()
	o := api.Observation{Node: a.st.NodeID, Seq: seq, TS: a.now(), Mode: a.mode, ModeDetail: a.modeDetail, Buffered: buffered, Workloads: []api.WorkloadObs{}}
	if a.bundle != nil {
		o.BundleIndex, o.BundleIssued, o.BundleIssuer = a.bundle.StateIndex, a.bundle.Issued, short(a.bundleEnv.Signer)
	}
	if f, ok := a.facts.Load().(api.Facts); ok {
		o.Facts = f
	}
	o.Facts.ClockSkewMs = a.skewMs
	seen := map[string]bool{}
	ids := sortedKeys(a.decisions)
	for _, id := range sortedKeys(a.st.Admitted) {
		if _, ok := a.decisions[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		w := api.WorkloadObs{Assignment: id, Admitted: "pending", Observed: "unknown", Checks: []api.Check{}, Volumes: []api.VolumeObs{}}
		d := a.decisions[id]
		ad := a.st.Admitted[id]
		if d != nil {
			w.Generation, w.Code, w.Reason, w.Checks = d.Gen, d.Code, d.Reason, d.Checks
			switch {
			case d.Allowed || d.Hold:
				w.Admitted = "allowed"
			default:
				w.Admitted = "refused"
			}
			if d.Hold {
				w.Code = "HOLD"
			}
		}
		if ad != nil {
			w.App, w.Replica, w.Runtime = ad.App, ad.Replica, ad.Runtime
			switch {
			case d == nil:
				w.Generation, w.Admitted = ad.Generation, "allowed"
			case d.Gen != ad.Generation && !ad.Stopped && (!d.Allowed || d.Hold):
				// The assignment on offer was refused (stale, forged or not
				// allowed); report the generation that is actually admitted.
				w.Generation, w.Admitted = ad.Generation, "allowed"
				w.Reason = "generation " + itoa(ad.Generation) + " remains admitted; assignment generation " + itoa(d.Gen) + " refused: " + d.Reason
			}
			switch {
			case ad.Stopped:
				w.Observed = "stopped"
			case ad.LastState == "running":
				w.Observed = "running"
			case ad.LastState == "oom-killed":
				w.Observed = "oom-killed"
			case ad.LastState == "exited":
				w.Observed = "exited"
			case ad.LastState == "failed":
				w.Observed = "failed"
				if w.Reason == "" || d != nil && d.Allowed {
					w.Reason = ad.LastDetail
				}
			default:
				w.Observed = "unknown"
			}
			if !ad.Stopped {
				w.PID, w.ContainerID, w.LocalPort, w.StartedAt = ad.Inst.PID, ad.Inst.ContainerID, ad.Inst.Port, ad.Inst.StartedAt
				if w.Observed == "running" && ad.Inst.Port > 0 {
					w.MeshPort = a.st.MeshPorts[id]
				}
			}
			w.Restarts, w.ExitCode = ad.Restarts, ad.LastExit
			if h := a.health[id]; h != nil && w.Observed == "running" {
				hc := *h
				w.Health = &hc
			}
			for _, v := range ad.Volumes {
				vo := api.VolumeObs{VolumeID: v.VolumeID}
				if vs := a.st.Volumes[v.VolumeID]; vs != nil {
					vo.LastSnapshot, vo.Restored = vs.LastSnapshot, vs.Restored
				}
				w.Volumes = append(w.Volumes, vo)
			}
		} else if d != nil && (d.Allowed && !d.Hold) && d.Code != "STOP" {
			w.Observed = "starting"
		} else if d != nil && d.Code == "STOP" {
			w.Observed = "stopped"
		}
		o.Workloads = append(o.Workloads, w)
	}
	o.Mesh = a.meshObs()
	stats := a.cas.Stats()
	o.Storage = &api.StorageObs{CapacityBytes: diskTotal(a.cfg.DataDir), FreeBytes: diskFree(a.cfg.DataDir), UsedBytes: stats.Used, QuotaBytes: stats.Quota, Chunks: stats.Objects, Corrupt: stats.Corrupt}
	if a.edge != nil {
		e := a.edge.observe()
		o.Edge = &e
	}
	seqL, hash := a.journal.Head()
	o.Ledger = api.LedgerHead{Seq: seqL, Hash: hash}
	return o
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }

// observe signs and sends the current observation, flushing any buffer.
func (a *Agent) observe() {
	if time.Since(a.factsAt) > 30*time.Second {
		go a.probeFacts()
		a.factsAt = time.Now()
	}
	offline := a.chaos.dropCP.Load()
	o := a.buildObservation(false)
	env, err := a.sign(envelope.KindObservation, o)
	if err != nil {
		return
	}
	if offline {
		a.bufferObservation(o)
		return
	}
	pending := a.outbox.pending()
	batch := append(pending, env)
	var res struct {
		Results []struct {
			Seq    int64  `json:"seq"`
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"results"`
	}
	if err := a.cp.post("/v1/observe", map[string]any{"observations": batch}, &res); err != nil {
		a.bufferObservation(o)
		return
	}
	if len(pending) > 0 {
		// Per-observation outcome of the flush, kept as evidence: a
		// buffered observation that was not committed is not durable.
		counts := map[string]int{}
		var notes []string
		for i, r := range res.Results {
			if i >= len(pending) {
				break
			}
			counts[r.Status]++
			if r.Status != "committed" {
				notes = append(notes, fmt.Sprintf("seq %d %s: %s", r.Seq, r.Status, r.Reason))
			}
		}
		detail := fmt.Sprintf("%d buffered observation(s) delivered after reconnection: %d committed, %d cached, %d rejected", len(pending), counts["committed"], counts["cached"], counts["rejected"])
		if len(notes) > 0 {
			detail += " (" + strings.Join(notes, "; ") + ")"
			a.log.Printf("flush: %s", detail)
		}
		a.outbox.drop(len(pending))
		a.record("observations-flushed", "node/"+a.st.NodeID, 0, detail)
	}
	for _, r := range res.Results {
		if r.Status == "rejected" && r.Seq == o.Seq {
			a.mu.Lock()
			a.lastErr = "observation rejected: " + r.Reason
			a.mu.Unlock()
		}
	}
}

// bufferObservation keeps a re-signed copy marked buffered, but only when
// something material changed, so an outage does not fill the buffer with
// identical heartbeats.
func (a *Agent) bufferObservation(o api.Observation) {
	o.Buffered = true
	key := bufferKey(&o)
	a.mu.Lock()
	same := key == a.lastBuffered
	a.lastBuffered = key
	a.mu.Unlock()
	if same {
		return
	}
	env, err := a.sign(envelope.KindObservation, o)
	if err == nil {
		a.outbox.add(env)
	}
}

func bufferKey(o *api.Observation) string {
	b, _ := json.Marshal(struct {
		Mode string
		W    []api.WorkloadObs
	}{o.Mode, stripVolatile(o.Workloads)})
	return string(b)
}

func stripVolatile(ws []api.WorkloadObs) []api.WorkloadObs {
	out := make([]api.WorkloadObs, len(ws))
	for i, w := range ws {
		w.Health = nil
		out[i] = w
	}
	return out
}
