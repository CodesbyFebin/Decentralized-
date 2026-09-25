package control

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mirror copies committed audit entries and the state document into
// Postgres. Raft is the durability boundary for desired state; Postgres is a
// queryable evidence mirror. The console shows "DB COMMITTED" only for
// entries the mirror has actually written, and "JOURNAL ONLY" otherwise.
type mirror struct {
	url  string
	log  *log.Logger
	mu   sync.Mutex
	st   MirrorStatus
	pool *pgxpool.Pool
}

// MirrorStatus is exposed in views.
type MirrorStatus struct {
	Enabled     bool   `json:"enabled"`
	Connected   bool   `json:"connected"`
	AuditSeq    int64  `json:"auditSeq"` // highest audit seq written to Postgres
	StateIndex  int64  `json:"stateIndex"`
	LastOK      int64  `json:"lastOk"`
	LastError   string `json:"lastError"`
	LastErrorAt int64  `json:"lastErrorAt"`
}

func newMirror(url string, l *log.Logger) *mirror {
	return &mirror{url: url, log: l, st: MirrorStatus{Enabled: true}}
}

func (m *mirror) status() MirrorStatus {
	if m == nil {
		return MirrorStatus{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.st
}

func (m *mirror) fail(err error) {
	m.mu.Lock()
	m.st.Connected = false
	m.st.LastError = err.Error()
	m.st.LastErrorAt = nowMs()
	m.mu.Unlock()
}

const mirrorSchema = `
create table if not exists dh_audit (
  seq bigint primary key,
  ts bigint not null,
  actor text not null,
  source text not null,
  action text not null,
  resource text not null,
  generation bigint not null,
  detail text not null,
  evidence text not null,
  prev text not null,
  hash text not null unique
);
create table if not exists dh_state (
  id integer primary key,
  cluster text not null,
  state_index bigint not null,
  body jsonb not null,
  updated_at timestamptz not null default now()
);`

func (m *mirror) run(s *Server) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			if m.pool != nil {
				m.pool.Close()
			}
			return
		case <-t.C:
		}
		if !s.IsLeader() {
			continue
		}
		if err := m.sync(s); err != nil {
			m.fail(err)
			if m.pool != nil {
				m.pool.Close()
				m.pool = nil
			}
		}
	}
}

func (m *mirror) sync(s *Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if m.pool == nil {
		pool, err := pgxpool.New(ctx, m.url)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, mirrorSchema); err != nil {
			pool.Close()
			return err
		}
		m.pool = pool
		var max int64
		_ = pool.QueryRow(ctx, "select coalesce(max(seq),0) from dh_audit").Scan(&max)
		m.mu.Lock()
		m.st.AuditSeq = max
		m.mu.Unlock()
	}
	m.mu.Lock()
	from := m.st.AuditSeq
	m.mu.Unlock()
	var entries []struct {
		seq                                                           int64
		ts, gen                                                       int64
		actor, source, action, resource, detail, evidence, prev, hash string
	}
	var body []byte
	var index int64
	var cluster string
	s.fsm.Read(func(st *State) {
		for _, e := range st.Audit.Entries {
			if e.Seq <= from {
				continue
			}
			entries = append(entries, struct {
				seq                                                           int64
				ts, gen                                                       int64
				actor, source, action, resource, detail, evidence, prev, hash string
			}{e.Seq, e.TS, e.Generation, e.Actor, e.Source, e.Action, e.Resource, e.Detail, e.Evidence, e.Prev, e.Hash})
			if len(entries) >= 500 {
				break
			}
		}
		index, cluster = st.Index, st.Cluster
		slim := *st
		slim.LocalCAKey = "" // secrets never leave the replicated log
		slim.Audit.Entries = nil
		body, _ = json.Marshal(&slim)
	})
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var last int64 = from
	for _, e := range entries {
		if _, err := tx.Exec(ctx, `insert into dh_audit (seq, ts, actor, source, action, resource, generation, detail, evidence, prev, hash)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) on conflict (seq) do nothing`,
			e.seq, e.ts, e.actor, e.source, e.action, e.resource, e.gen, e.detail, e.evidence, e.prev, e.hash); err != nil {
			return err
		}
		last = e.seq
	}
	if _, err := tx.Exec(ctx, `insert into dh_state (id, cluster, state_index, body) values (1,$1,$2,$3)
		on conflict (id) do update set cluster=excluded.cluster, state_index=excluded.state_index, body=excluded.body, updated_at=now()`, cluster, index, body); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.st.Connected, m.st.AuditSeq, m.st.StateIndex, m.st.LastOK, m.st.LastError = true, last, index, nowMs(), ""
	m.mu.Unlock()
	return nil
}
