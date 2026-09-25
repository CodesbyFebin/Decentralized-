package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
)

func sample() *Ledger {
	l := &Ledger{}
	l.Append(Entry{TS: 1, Actor: "op", Source: SourceOperator, Action: "node-approve", Resource: "dh1x", Detail: "a"})
	l.Append(Entry{TS: 2, Actor: "op", Source: SourceOperator, Action: "manifest-apply", Resource: "app/wiki", Generation: 1})
	l.Append(Entry{TS: 3, Actor: "dh1x", Source: SourceHost, Action: "workload-start", Resource: "app/wiki/r0", Generation: 1})
	return l
}

func TestChainVerifies(t *testing.T) {
	l := sample()
	if b := Verify(l.Entries, 0, Genesis); b != nil {
		t.Fatal(b)
	}
}

func TestTamperNamesTheBreak(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(l *Ledger)
		reason string
		seq    int64
	}{
		{"detail rewritten", func(l *Ledger) { l.Entries[1].Detail = "rewritten" }, ReasonHashMismatch, 2},
		{"entry removed", func(l *Ledger) { l.Entries = append(l.Entries[:1], l.Entries[2:]...) }, ReasonSeqGap, 3},
		{"prev forged", func(l *Ledger) {
			l.Entries[2].Prev = "b3:00"
			l.Entries[2].Hash = ComputeHash(l.Entries[2])
		}, ReasonPrevMismatch, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := sample()
			tc.mutate(l)
			b := Verify(l.Entries, 0, Genesis)
			if b == nil || b.Reason != tc.reason || b.Seq != tc.seq {
				t.Fatalf("got %+v want %s at %d", b, tc.reason, tc.seq)
			}
			if b.Expected == b.Actual {
				t.Fatal("break must show differing expected/actual")
			}
		})
	}
}

func TestCheckpointDetectsTruncationAndRewrite(t *testing.T) {
	id := identity.FromSeed(make([]byte, 32))
	l := sample()
	cp, err := SignCheckpoint(id, "", "cluster", l, 10)
	if err != nil {
		t.Fatal(err)
	}
	keys := func(string) []string { return []string{id.PubString()} }
	if b := VerifyCheckpoints(l.Entries, []*envelope.Envelope{cp}, keys); b != nil {
		t.Fatal(b)
	}
	// Truncation: a consistent chain that stops early still verifies as a
	// chain, but not against the signed head.
	short := l.Entries[:2]
	if b := Verify(short, 0, Genesis); b != nil {
		t.Fatal("prefix should verify as a chain")
	}
	if b := VerifyCheckpoints(short, []*envelope.Envelope{cp}, keys); b == nil || b.Reason != ReasonTruncated {
		t.Fatalf("truncation not detected: %+v", b)
	}
	// Full rewrite with recomputed hashes: chain verifies, checkpoint does not.
	rw := sample()
	rw.Entries = nil
	rw.Append(Entry{TS: 1, Actor: "attacker", Action: "node-approve"})
	rw.Append(Entry{TS: 2, Actor: "attacker", Action: "x"})
	rw.Append(Entry{TS: 3, Actor: "attacker", Action: "y"})
	if b := VerifyCheckpoints(rw.Entries, []*envelope.Envelope{cp}, keys); b == nil || b.Reason != ReasonCheckpointPos {
		t.Fatalf("rewrite not detected: %+v", b)
	}
	// Wrong key.
	other := identity.FromSeed([]byte(strings.Repeat("x", 32)))
	if b := VerifyCheckpoints(l.Entries, []*envelope.Envelope{cp}, func(string) []string { return []string{other.PubString()} }); b == nil || b.Reason != ReasonCheckpointSig {
		t.Fatalf("wrong key not detected: %+v", b)
	}
}

func TestJournalDurableAndCorruptionRefusesAppend(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "journal.jsonl")
	j, err := OpenJournal(p)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := j.Append(Entry{TS: int64(i), Actor: "h", Action: "a"}); err != nil {
			t.Fatal(err)
		}
	}
	j2, _ := OpenJournal(p)
	if j2.Corrupt() != nil {
		t.Fatal(j2.Corrupt())
	}
	if seq, _ := j2.Head(); seq != 3 {
		t.Fatalf("head %d", seq)
	}
	raw, _ := os.ReadFile(p)
	raw = []byte(strings.Replace(string(raw), `"action":"a"`, `"action":"b"`, 1))
	os.WriteFile(p, raw, 0o600)
	j3, _ := OpenJournal(p)
	if b := j3.Corrupt(); b == nil || b.Reason != ReasonHashMismatch || b.Seq != 1 {
		t.Fatalf("corruption not detected: %+v", b)
	}
	if _, err := j3.Append(Entry{Actor: "h"}); err == nil {
		t.Fatal("append to corrupt journal must fail")
	}
}

func TestSealJournal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "journal.jsonl")
	j, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := j.Append(Entry{TS: int64(i), Actor: "h", Source: SourceHost, Action: "x", Detail: "original"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := SealJournal(path, "op", "test", 10); err == nil {
		t.Fatal("sealing a verifying journal must be refused")
	}
	raw, _ := os.ReadFile(path)
	tampered := strings.Replace(string(raw), "original", "rewritten", 1)
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	sealed, first, err := SealJournal(path, "op", "disk error investigated", 10)
	if err != nil {
		t.Fatal(err)
	}
	if kept, _ := os.ReadFile(sealed); string(kept) != tampered {
		t.Fatal("sealed journal must be kept byte-for-byte")
	}
	if first.Seq != 1 || first.Action != "ledger-sealed" || !strings.Contains(first.Detail, "event-hash mismatch at seq 1") || first.Evidence != "b3:"+hashHex([]byte(tampered)) {
		t.Fatalf("first entry: %+v", first)
	}
	nj, err := OpenJournal(path)
	if err != nil || nj.Corrupt() != nil {
		t.Fatalf("new journal: %v %v", err, nj.Corrupt())
	}
}
