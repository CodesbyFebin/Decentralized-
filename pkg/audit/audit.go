// Package audit implements the dh/v1 hash-chained, append-only ledger.
//
// Each entry commits to its predecessor:
//
//	hash = "b3:" + hex(BLAKE3("decentralized.host/audit/v1\n" + canonical(entry without "hash")))
//
// The first entry's prev is "genesis". Verification is offline and names the
// exact break (sequence gap, previous-hash mismatch, event-hash mismatch,
// checkpoint signature failure) with expected and actual values.
//
// Entries may reference the signed envelope that caused them (Evidence is its
// digest); the chain head is periodically signed by a control-plane member or
// host key (Checkpoint) so a truncated or rewritten ledger is detectable even
// by a verifier that only holds the signer's public key.
package audit

import (
	"encoding/json"
	"fmt"

	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
)

// Genesis is the prev value of the first entry.
const Genesis = "genesis"

// Sources separate authoritative ledgers from projections.
const (
	SourceControl    = "control-plane"
	SourceHost       = "host"
	SourceOperator   = "operator"
	SourceFederation = "federation"
	SourceChaos      = "chaos"
)

// Entry is one ledger event.
type Entry struct {
	Seq        int64  `json:"seq"`
	TS         int64  `json:"ts"` // unix milliseconds, assigned by the writer
	Actor      string `json:"actor"`
	Source     string `json:"source"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	Generation int64  `json:"generation"`
	Detail     string `json:"detail"`
	Evidence   string `json:"evidence"` // digest of the causing signed envelope, or ""
	Prev       string `json:"prev"`
	Hash       string `json:"hash"`
}

type hashable struct {
	Seq        int64  `json:"seq"`
	TS         int64  `json:"ts"`
	Actor      string `json:"actor"`
	Source     string `json:"source"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	Generation int64  `json:"generation"`
	Detail     string `json:"detail"`
	Evidence   string `json:"evidence"`
	Prev       string `json:"prev"`
}

// ComputeHash returns the chain hash of e (ignoring e.Hash).
func ComputeHash(e Entry) string {
	return envelope.DomainHash("audit", hashable{
		Seq: e.Seq, TS: e.TS, Actor: e.Actor, Source: e.Source, Action: e.Action,
		Resource: e.Resource, Generation: e.Generation, Detail: e.Detail, Evidence: e.Evidence, Prev: e.Prev,
	})
}

// Ledger is an in-memory chain. It is not safe for concurrent use.
type Ledger struct {
	Entries []Entry `json:"entries"`
}

// Head returns the last seq and hash.
func (l *Ledger) Head() (int64, string) {
	if len(l.Entries) == 0 {
		return 0, Genesis
	}
	last := l.Entries[len(l.Entries)-1]
	return last.Seq, last.Hash
}

// Append links e to the chain, fills Seq/Prev/Hash and returns the entry.
func (l *Ledger) Append(e Entry) Entry {
	seq, prev := l.Head()
	e.Seq = seq + 1
	e.Prev = prev
	e.Hash = ComputeHash(e)
	l.Entries = append(l.Entries, e)
	return e
}

// Break describes the first verification failure.
type Break struct {
	Seq      int64  `json:"seq"`
	Reason   string `json:"reason"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

func (b *Break) Error() string {
	return fmt.Sprintf("audit: %s at seq %d (expected %s, actual %s)", b.Reason, b.Seq, b.Expected, b.Actual)
}

// Break reasons.
const (
	ReasonSeqGap        = "sequence gap"
	ReasonPrevMismatch  = "previous-hash mismatch"
	ReasonHashMismatch  = "event-hash mismatch"
	ReasonCheckpointSig = "checkpoint signature invalid"
	ReasonCheckpointPos = "checkpoint does not match chain"
	ReasonTruncated     = "ledger truncated below signed checkpoint"
)

// Verify checks the chain from the start. startSeq/startPrev allow verifying
// a suffix whose predecessor is already trusted; pass 0, Genesis for a full
// ledger.
func Verify(entries []Entry, startSeq int64, startPrev string) *Break {
	prevSeq, prev := startSeq, startPrev
	for _, e := range entries {
		if e.Seq != prevSeq+1 {
			return &Break{Seq: e.Seq, Reason: ReasonSeqGap, Expected: fmt.Sprint(prevSeq + 1), Actual: fmt.Sprint(e.Seq)}
		}
		if e.Prev != prev {
			return &Break{Seq: e.Seq, Reason: ReasonPrevMismatch, Expected: prev, Actual: e.Prev}
		}
		if want := ComputeHash(e); want != e.Hash {
			return &Break{Seq: e.Seq, Reason: ReasonHashMismatch, Expected: want, Actual: e.Hash}
		}
		prevSeq, prev = e.Seq, e.Hash
	}
	return nil
}

// CheckpointPayload is what a checkpoint signs.
type CheckpointPayload struct {
	Ledger string `json:"ledger"` // ledger name, e.g. cluster name or host id
	Seq    int64  `json:"seq"`
	Hash   string `json:"hash"`
	TS     int64  `json:"ts"`
}

// Checkpoint is a signed chain head.
type Checkpoint struct {
	Envelope *envelope.Envelope `json:"envelope"`
}

// SignCheckpoint signs the current head of l.
func SignCheckpoint(id *identity.Identity, signer, ledger string, l *Ledger, ts int64) (*envelope.Envelope, error) {
	seq, hash := l.Head()
	return envelope.Sign(id, signer, envelope.KindCheckpoint, CheckpointPayload{Ledger: ledger, Seq: seq, Hash: hash, TS: ts})
}

// VerifyCheckpoints validates signed checkpoints against a chain. keysFor
// returns the allowed wire-encoded public keys for a signer.
func VerifyCheckpoints(entries []Entry, cps []*envelope.Envelope, keysFor func(signer string) []string) *Break {
	bySeq := map[int64]string{}
	var maxSeq int64
	for _, e := range entries {
		bySeq[e.Seq] = e.Hash
		if e.Seq > maxSeq {
			maxSeq = e.Seq
		}
	}
	for _, env := range cps {
		if err := env.VerifyKey(envelope.KindCheckpoint, keysFor(env.Signer)...); err != nil {
			var p CheckpointPayload
			_ = json.Unmarshal(env.Payload, &p)
			return &Break{Seq: p.Seq, Reason: ReasonCheckpointSig, Expected: "signature by " + env.Signer, Actual: err.Error()}
		}
		var p CheckpointPayload
		if err := env.Decode(&p); err != nil {
			return &Break{Reason: ReasonCheckpointSig, Expected: "decodable payload", Actual: err.Error()}
		}
		if p.Seq > maxSeq {
			return &Break{Seq: p.Seq, Reason: ReasonTruncated, Expected: fmt.Sprintf("seq >= %d", p.Seq), Actual: fmt.Sprintf("ledger ends at %d", maxSeq)}
		}
		if p.Seq == 0 {
			continue
		}
		if got := bySeq[p.Seq]; got != p.Hash {
			return &Break{Seq: p.Seq, Reason: ReasonCheckpointPos, Expected: p.Hash, Actual: got}
		}
	}
	return nil
}
