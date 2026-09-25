package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"decentralized.host/pkg/envelope"
)

// Journal is a file-backed ledger: one JSON entry per line, fsynced on every
// append. It is the host-local durability boundary. A journal that fails
// verification on open is reported, never silently repaired.
type Journal struct {
	mu      sync.Mutex
	path    string
	ledger  Ledger
	corrupt *Break
}

// OpenJournal loads (or creates) the journal at path and verifies it.
func OpenJournal(path string) (*Journal, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	j := &Journal{path: path}
	f, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if f != nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		line := 0
		for sc.Scan() {
			line++
			if len(sc.Bytes()) == 0 {
				continue
			}
			var e Entry
			if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
				j.corrupt = &Break{Seq: int64(line), Reason: "undecodable journal line", Expected: "JSON entry", Actual: err.Error()}
				break
			}
			j.ledger.Entries = append(j.ledger.Entries, e)
		}
		f.Close()
		if err := sc.Err(); err != nil && j.corrupt == nil {
			j.corrupt = &Break{Reason: "journal read error", Actual: err.Error()}
		}
	}
	if j.corrupt == nil {
		j.corrupt = Verify(j.ledger.Entries, 0, Genesis)
	}
	return j, nil
}

// Corrupt returns the verification break found at open, if any.
func (j *Journal) Corrupt() *Break {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.corrupt
}

// Append adds an entry durably. A corrupt journal refuses appends: extending
// a broken chain would hide the break.
func (j *Journal) Append(e Entry) (Entry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.corrupt != nil {
		return Entry{}, fmt.Errorf("journal is corrupt (%s at seq %d); refusing to append", j.corrupt.Reason, j.corrupt.Seq)
	}
	e = j.ledger.Append(e)
	line, err := json.Marshal(e)
	if err != nil {
		return Entry{}, err
	}
	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		j.ledger.Entries = j.ledger.Entries[:len(j.ledger.Entries)-1]
		return Entry{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		j.ledger.Entries = j.ledger.Entries[:len(j.ledger.Entries)-1]
		return Entry{}, err
	}
	if err := f.Sync(); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Entries returns a copy of entries with seq > from (up to limit; 0 = all).
func (j *Journal) Entries(from int64, limit int) []Entry {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []Entry
	for _, e := range j.ledger.Entries {
		if e.Seq <= from {
			continue
		}
		out = append(out, e)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// Head returns the chain head.
func (j *Journal) Head() (int64, string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.ledger.Head()
}

// Path returns the file path.
func (j *Journal) Path() string { return j.path }

// SealJournal is the operator's recovery path for a corrupt journal. The
// corrupt file is kept byte-for-byte (renamed, never edited) and a new chain
// starts whose first entry names the sealed file, its BLAKE3 hash, the exact
// break and the operator's reason. Nothing is repaired silently: the gap is
// itself recorded evidence. Sealing a journal that verifies is refused.
func SealJournal(path, actor, reason string, ts int64) (sealedPath string, first Entry, err error) {
	if reason == "" {
		return "", Entry{}, errors.New("a reason is required")
	}
	j, err := OpenJournal(path)
	if err != nil {
		return "", Entry{}, err
	}
	br := j.Corrupt()
	if br == nil {
		return "", Entry{}, errors.New("journal verifies; there is nothing to seal")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", Entry{}, err
	}
	sealedPath = fmt.Sprintf("%s.sealed-%d", path, ts)
	if err := os.Rename(path, sealedPath); err != nil {
		return "", Entry{}, err
	}
	nj, err := OpenJournal(path)
	if err != nil {
		return sealedPath, Entry{}, err
	}
	sum := "b3:" + hashHex(raw)
	first, err = nj.Append(Entry{TS: ts, Actor: actor, Source: SourceHost, Action: "ledger-sealed", Resource: "ledger/host",
		Detail: fmt.Sprintf("previous journal sealed as %s (%s, %d bytes): %s at seq %d (expected %s, actual %s); operator reason: %s",
			filepath.Base(sealedPath), sum, len(raw), br.Reason, br.Seq, br.Expected, br.Actual, reason),
		Evidence: sum})
	return sealedPath, first, err
}

func hashHex(b []byte) string { return envelope.HashBytes(b) }
