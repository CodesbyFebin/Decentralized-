package storage

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
)

func randomBytes(t *testing.T, n int, seed int64) []byte {
	b := make([]byte, n)
	r := mrand.New(mrand.NewSource(seed))
	r.Read(b)
	return b
}

func TestFastCDCBoundsAndDeterminism(t *testing.T) {
	data := randomBytes(t, 4<<20, 1)
	a := ChunkAll(data)
	b := ChunkAll(data)
	if len(a) != len(b) {
		t.Fatal("nondeterministic")
	}
	var total int
	for i, c := range a {
		if c.ID != b[i].ID {
			t.Fatal("nondeterministic id")
		}
		if i < len(a)-1 && (len(c.Data) < MinChunk || len(c.Data) > MaxChunk) {
			t.Fatalf("chunk %d size %d out of bounds", i, len(c.Data))
		}
		total += len(c.Data)
	}
	if total != len(data) {
		t.Fatal("chunks do not cover the input")
	}
	avg := total / len(a)
	if avg < AvgChunk/2 || avg > AvgChunk*2 {
		t.Fatalf("average chunk size %d far from target %d", avg, AvgChunk)
	}
}

// Content-defined chunking: inserting bytes near the start must leave most
// later chunks unchanged (fixed-size chunking would change all of them).
func TestFastCDCShiftResistance(t *testing.T) {
	data := randomBytes(t, 4<<20, 2)
	shifted := append(append(append([]byte{}, data[:1000]...), []byte("INSERTED-BYTES")...), data[1000:]...)
	before := map[string]bool{}
	for _, c := range ChunkAll(data) {
		before[c.ID] = true
	}
	after := ChunkAll(shifted)
	same := 0
	for _, c := range after {
		if before[c.ID] {
			same++
		}
	}
	if float64(same)/float64(len(after)) < 0.8 {
		t.Fatalf("only %d/%d chunks survived a 14-byte insertion", same, len(after))
	}
}

func TestStreamingMatchesWhole(t *testing.T) {
	data := randomBytes(t, 3<<20+123, 3)
	whole := ChunkAll(data)
	ch := NewChunker(iotestReader{bytes.NewReader(data)})
	i := 0
	for {
		c, err := ch.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if c.ID != whole[i].ID || c.Offset != whole[i].Offset {
			t.Fatalf("chunk %d differs between streaming and whole", i)
		}
		i++
	}
	if i != len(whole) {
		t.Fatalf("%d vs %d chunks", i, len(whole))
	}
}

// iotestReader returns short reads to exercise the refill path.
type iotestReader struct{ r io.Reader }

func (r iotestReader) Read(p []byte) (int, error) {
	if len(p) > 7777 {
		p = p[:7777]
	}
	return r.r.Read(p)
}

func TestCASVerifyOnReadAndQuarantine(t *testing.T) {
	c, err := OpenCAS(t.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.Put([]byte("hello sovereign storage"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.Get(id); err != nil || string(got) != "hello sovereign storage" {
		t.Fatal(err)
	}
	if err := c.CorruptForTest(id); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(id); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("corruption served: %v", err)
	}
	if c.Has(id) || c.Stats().Corrupt != 1 {
		t.Fatal("corrupt object not quarantined")
	}
	if err := c.PutWithID(id, []byte("not the content")); err == nil {
		t.Fatal("mismatched content accepted")
	}
}

func TestCASQuotaAndCrashLeftovers(t *testing.T) {
	dir := t.TempDir()
	c, _ := OpenCAS(dir, 100)
	if _, err := c.Put(make([]byte, 60)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Put(append(make([]byte, 60), 1)); !errors.Is(err, ErrQuota) {
		t.Fatalf("quota not enforced: %v", err)
	}
	// A temp file left by a crash is not an object and is cleaned on open.
	os.WriteFile(filepath.Join(dir, "tmp", "put-crash"), []byte("partial"), 0o600)
	c2, _ := OpenCAS(dir, 0)
	if len(c2.List()) != 1 {
		t.Fatal(c2.List())
	}
	if _, err := os.Stat(filepath.Join(dir, "tmp", "put-crash")); !os.IsNotExist(err) {
		t.Fatal("crash leftover not removed")
	}
}

func TestSnapshotRestoreRoundTrip(t *testing.T) {
	c, _ := OpenCAS(t.TempDir(), 0)
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "sub"), 0o700)
	big := make([]byte, 1<<20)
	rand.Read(big)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("alpha"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "big.bin"), big, 0o600)
	s1, err := Snapshot(c, src, "wiki/data/r0", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := Snapshot(c, src, "wiki/data/r0", "", 1)
	if s1.ID != s2.ID || s1.Root != s2.Root {
		t.Fatal("snapshot of unchanged data is not stable")
	}
	dst := filepath.Join(t.TempDir(), "restored")
	if err := Restore(c, s1.ID, dst); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dst, "sub", "big.bin"))
	if !bytes.Equal(got, big) {
		t.Fatal("restored bytes differ")
	}
	m, _ := LoadManifest(c, s1.ID)
	if MerkleRoot(ChunkSet(s1.ID, m)) != s1.Root {
		t.Fatal("root does not match chunk set")
	}
}

func TestMerkleBucketsLocateDifference(t *testing.T) {
	var ids []string
	for i := 0; i < 500; i++ {
		ids = append(ids, ChunkID([]byte{byte(i), byte(i >> 8)}))
	}
	a := Buckets(ids)
	b := Buckets(ids[1:])
	diff := 0
	for i := range a {
		if a[i] != b[i] {
			diff++
			if i != BucketOf(ids[0]) {
				t.Fatal("wrong bucket differs")
			}
		}
	}
	if diff != 1 {
		t.Fatalf("%d buckets differ", diff)
	}
	if MerkleRoot(ids) == MerkleRoot(ids[1:]) || MerkleRoot(ids) != MerkleRoot(append(ids, ids[3])) {
		t.Fatal("merkle root must depend on the set only")
	}
}
