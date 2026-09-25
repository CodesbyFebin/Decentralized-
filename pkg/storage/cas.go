package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

var idRE = regexp.MustCompile(`^b3:[a-f0-9]{64}$`)

// Errors.
var (
	ErrNotFound = errors.New("cas: object not found")
	ErrCorrupt  = errors.New("cas: object failed BLAKE3 verification and was quarantined")
	ErrQuota    = errors.New("cas: storage quota exceeded (ENOSPC)")
	ErrBadID    = errors.New("cas: malformed object id")
)

// CAS is an immutable local content-addressed store. Objects are written to
// a temporary file, fsynced and renamed into place, so a crash mid-write
// never leaves a visible partial object. Every read re-hashes the bytes; a
// mismatch quarantines the object and is reported, never returned.
type CAS struct {
	root    string
	quota   int64 // 0 = unlimited
	used    atomic.Int64
	count   atomic.Int64
	corrupt atomic.Int64
	mu      sync.Mutex // serializes deletes against quota accounting
	// FailWrites injects ENOSPC for chaos testing (disk-full scenario).
	failWrites atomic.Bool
}

// OpenCAS opens (or creates) a store under dir.
func OpenCAS(dir string, quota int64) (*CAS, error) {
	for _, sub := range []string{"objects", "tmp", "quarantine"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o700); err != nil {
			return nil, err
		}
	}
	c := &CAS{root: dir, quota: quota}
	// Leftover temporaries are incomplete writes from a crash: remove them.
	tmps, _ := os.ReadDir(filepath.Join(dir, "tmp"))
	for _, t := range tmps {
		_ = os.Remove(filepath.Join(dir, "tmp", t.Name()))
	}
	_ = filepath.WalkDir(filepath.Join(dir, "objects"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			c.used.Add(info.Size())
			c.count.Add(1)
		}
		return nil
	})
	q, _ := os.ReadDir(filepath.Join(dir, "quarantine"))
	c.corrupt.Store(int64(len(q)))
	return c, nil
}

// Close is a no-op kept for symmetry.
func (c *CAS) Close() error { return nil }

// Root returns the store directory.
func (c *CAS) Root() string { return c.root }

// SetQuota changes the quota (0 = unlimited).
func (c *CAS) SetQuota(q int64) { c.mu.Lock(); c.quota = q; c.mu.Unlock() }

// InjectENOSPC makes every new write fail with ErrQuota (chaos).
func (c *CAS) InjectENOSPC(on bool) { c.failWrites.Store(on) }

func (c *CAS) path(id string) (string, error) {
	if !idRE.MatchString(id) {
		return "", ErrBadID
	}
	h := id[3:]
	return filepath.Join(c.root, "objects", h[:2], h[2:4], h), nil
}

// Has reports whether the object file exists (not verified).
func (c *CAS) Has(id string) bool {
	p, err := c.path(id)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Put stores data and returns its id. Storing an existing object is a no-op.
func (c *CAS) Put(data []byte) (string, error) {
	id := ChunkID(data)
	return id, c.PutWithID(id, data)
}

// PutWithID stores data under id after verifying id matches.
func (c *CAS) PutWithID(id string, data []byte) error {
	if ChunkID(data) != id {
		return fmt.Errorf("cas: content does not hash to %s", id)
	}
	p, err := c.path(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	if c.failWrites.Load() {
		return ErrQuota
	}
	c.mu.Lock()
	if c.quota > 0 && c.used.Load()+int64(len(data)) > c.quota {
		c.mu.Unlock()
		return ErrQuota
	}
	c.used.Add(int64(len(data)))
	c.mu.Unlock()
	undo := func() { c.used.Add(-int64(len(data))) }
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		undo()
		return err
	}
	tmp, err := os.CreateTemp(filepath.Join(c.root, "tmp"), "put-*")
	if err != nil {
		undo()
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		undo()
		return err
	}
	// fsync(2) per object; durability through the drive cache is provided by
	// Barrier() at commit points (one device flush covers every earlier write).
	if err := unix.Fsync(int(tmp.Fd())); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		undo()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		undo()
		return err
	}
	if err := os.Chmod(tmpName, 0o400); err != nil {
		os.Remove(tmpName)
		undo()
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		undo()
		return err
	}
	syncDir(filepath.Dir(p))
	c.count.Add(1)
	return nil
}

func syncDir(dir string) {
	if f, err := os.Open(dir); err == nil {
		_ = unix.Fsync(int(f.Fd()))
		f.Close()
	}
}

// Barrier makes every object written so far durable on stable storage. On
// macOS this issues F_FULLFSYNC (os.File.Sync), which flushes the drive's
// volatile cache; on Linux fsync already implies a cache flush. It is called
// before an object set is relied on: an artifact is executed, a snapshot is
// reported, or replica evidence is signed.
func (c *CAS) Barrier() error {
	f, err := os.Open(filepath.Join(c.root, "objects"))
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// Get returns verified bytes. A mismatch quarantines the file.
func (c *CAS) Get(id string) ([]byte, error) {
	p, err := c.path(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ChunkID(data) != id {
		c.quarantine(id, p, int64(len(data)))
		return nil, ErrCorrupt
	}
	return data, nil
}

// Verify re-hashes an object; false means missing or corrupt (quarantined).
func (c *CAS) Verify(id string) (present bool, corrupt bool) {
	_, err := c.Get(id)
	switch {
	case err == nil:
		return true, false
	case errors.Is(err, ErrCorrupt):
		return false, true
	default:
		return false, false
	}
}

func (c *CAS) quarantine(id, p string, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dst := filepath.Join(c.root, "quarantine", id[3:]+"."+fmt.Sprint(time.Now().UnixNano()))
	if err := os.Rename(p, dst); err == nil {
		c.used.Add(-size)
		c.count.Add(-1)
		c.corrupt.Add(1)
	}
}

// Delete removes an object (garbage collection).
func (c *CAS) Delete(id string) error {
	p, err := c.path(id)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	info, err := os.Stat(p)
	if err != nil {
		return nil
	}
	_ = os.Chmod(p, 0o600)
	if err := os.Remove(p); err != nil {
		return err
	}
	c.used.Add(-info.Size())
	c.count.Add(-1)
	return nil
}

// Age returns how long ago an object was written.
func (c *CAS) Age(id string) (time.Duration, bool) {
	p, err := c.path(id)
	if err != nil {
		return 0, false
	}
	st, err := os.Stat(p)
	if err != nil {
		return 0, false
	}
	return time.Since(st.ModTime()), true
}

// List returns all object ids, sorted.
func (c *CAS) List() []string {
	var out []string
	_ = filepath.WalkDir(filepath.Join(c.root, "objects"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if len(name) == 64 && !strings.Contains(name, ".") {
			out = append(out, "b3:"+name)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// Stats summarizes the store.
type Stats struct {
	Objects int64 `json:"objects"`
	Used    int64 `json:"used"`
	Quota   int64 `json:"quota"`
	Corrupt int64 `json:"corrupt"`
}

// Stats returns counters.
func (c *CAS) Stats() Stats {
	c.mu.Lock()
	q := c.quota
	c.mu.Unlock()
	return Stats{Objects: c.count.Load(), Used: c.used.Load(), Quota: q, Corrupt: c.corrupt.Load()}
}

// Corrupt test hook: flip one byte of an object in place (chaos scenario).
func (c *CAS) CorruptForTest(id string) error {
	p, err := c.path(id)
	if err != nil {
		return err
	}
	_ = os.Chmod(p, 0o600)
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		b = []byte{0}
	}
	b[len(b)/2] ^= 0xff
	return os.WriteFile(p, b, 0o400)
}
