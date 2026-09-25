package control

import (
	"sync"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/scheduler"
)

// obsCache holds the latest verified observation per host on the leader.
// Observations are verified on arrival and only committed to the log when
// something material changed or every heartbeat interval, which keeps the
// replicated log small while the console still sees second-level freshness.
type obsCache struct {
	mu sync.Mutex
	m  map[string]*cachedObs
}

type cachedObs struct {
	Env       *envelope.Envelope
	Obs       api.Observation
	Received  time.Time
	Material  string
	Committed time.Time
	CommitSeq int64
}

func newObsCache() *obsCache { return &obsCache{m: map[string]*cachedObs{}} }

func (c *obsCache) get(node string) *cachedObs {
	c.mu.Lock()
	defer c.mu.Unlock()
	if o := c.m[node]; o != nil {
		cp := *o
		return &cp
	}
	return nil
}

func (c *obsCache) put(node string, o *cachedObs) {
	c.mu.Lock()
	c.m[node] = o
	c.mu.Unlock()
}

func (c *obsCache) markCommitted(node string, seq int64) {
	c.mu.Lock()
	if o := c.m[node]; o != nil {
		o.Committed = time.Now()
		o.CommitSeq = seq
	}
	c.mu.Unlock()
}

// planCache keeps the last scheduler plan per app for explanation.
type planCache struct {
	mu sync.Mutex
	m  map[string]planEntry
}

type planEntry struct {
	Plan scheduler.Plan `json:"plan"`
	At   int64          `json:"at"`
	Note string         `json:"note"`
}

func newPlanCache() *planCache { return &planCache{m: map[string]planEntry{}} }

func (p *planCache) set(app string, plan scheduler.Plan, note string) {
	p.mu.Lock()
	p.m[app] = planEntry{Plan: plan, At: nowMs(), Note: note}
	p.mu.Unlock()
}

func (p *planCache) all() map[string]planEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := map[string]planEntry{}
	for k, v := range p.m {
		out[k] = v
	}
	return out
}

// localRejections records refusals that never reach the log (malformed or
// unauthenticated requests), so the console can show them.
type localRejections struct {
	mu   sync.Mutex
	list []Rejection
}

func (l *localRejections) add(r Rejection) {
	l.mu.Lock()
	l.list = append(l.list, r)
	if len(l.list) > 200 {
		l.list = l.list[len(l.list)-200:]
	}
	l.mu.Unlock()
}

func (l *localRejections) all() []Rejection {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Rejection(nil), l.list...)
}
