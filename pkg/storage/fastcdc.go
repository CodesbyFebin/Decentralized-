// Package storage implements dh/v1 sovereign storage: BLAKE3 content
// addressing, FastCDC content-defined chunking, an atomic verify-on-read
// local CAS, Merkle roots for anti-entropy, and volume snapshots.
package storage

import (
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/zeebo/blake3"
)

// FastCDC parameters (dh/v1 §7.2). Normalized chunking (level 2) with a
// Gear rolling hash. Masks select the top bits of the hash, which depend on
// the last 64 input bytes; low bits would only depend on the last few.
const (
	MinChunk = 16 << 10
	AvgChunk = 64 << 10
	MaxChunk = 256 << 10
	maskS    = uint64(0xFFFFC00000000000) // top 18 bits; before the average size: harder to cut
	maskL    = uint64(0xFFFC000000000000) // top 14 bits; after the average size: easier to cut
)

// gear is derived, not copied: gear[i] = uint64le(BLAKE3("decentralized.host/fastcdc-gear/v1" || i)[:8]).
var gear = func() [256]uint64 {
	var g [256]uint64
	for i := 0; i < 256; i++ {
		sum := blake3.Sum256(append([]byte("decentralized.host/fastcdc-gear/v1"), byte(i)))
		g[i] = binary.LittleEndian.Uint64(sum[:8])
	}
	return g
}()

// Gear exposes the table for conformance vectors.
func Gear(i byte) uint64 { return gear[i] }

// Cut returns the length of the first chunk of data.
func Cut(data []byte) int {
	n := len(data)
	if n <= MinChunk {
		return n
	}
	if n > MaxChunk {
		n = MaxChunk
	}
	normal := AvgChunk
	if n < normal {
		normal = n
	}
	var h uint64
	i := MinChunk
	for ; i < normal; i++ {
		h = (h << 1) + gear[data[i]]
		if h&maskS == 0 {
			return i + 1
		}
	}
	for ; i < n; i++ {
		h = (h << 1) + gear[data[i]]
		if h&maskL == 0 {
			return i + 1
		}
	}
	return n
}

// Chunk is one content-defined chunk.
type Chunk struct {
	ID     string
	Offset int64
	Data   []byte
}

// ChunkID is "b3:" + hex(BLAKE3-256(data)).
func ChunkID(data []byte) string {
	sum := blake3.Sum256(data)
	return "b3:" + hex.EncodeToString(sum[:])
}

// Chunker splits a stream into content-defined chunks.
type Chunker struct {
	r      io.Reader
	buf    []byte
	start  int
	end    int
	eof    bool
	offset int64
}

// NewChunker wraps r.
func NewChunker(r io.Reader) *Chunker {
	return &Chunker{r: r, buf: make([]byte, 2*MaxChunk)}
}

// Next returns the next chunk or io.EOF. The returned Data is a copy.
func (c *Chunker) Next() (*Chunk, error) {
	if err := c.fill(); err != nil {
		return nil, err
	}
	if c.end == c.start {
		return nil, io.EOF
	}
	n := Cut(c.buf[c.start:c.end])
	data := make([]byte, n)
	copy(data, c.buf[c.start:c.start+n])
	ch := &Chunk{ID: ChunkID(data), Offset: c.offset, Data: data}
	c.start += n
	c.offset += int64(n)
	return ch, nil
}

func (c *Chunker) fill() error {
	if c.eof || c.end-c.start >= MaxChunk {
		return nil
	}
	if c.start > 0 {
		copy(c.buf, c.buf[c.start:c.end])
		c.end -= c.start
		c.start = 0
	}
	for c.end < len(c.buf) && !c.eof {
		n, err := c.r.Read(c.buf[c.end:])
		c.end += n
		if err == io.EOF {
			c.eof = true
		} else if err != nil {
			return err
		}
		if c.end-c.start >= MaxChunk {
			break
		}
	}
	return nil
}

// ChunkAll splits b and returns chunk boundaries (for tests and vectors).
func ChunkAll(b []byte) []Chunk {
	var out []Chunk
	var off int64
	for len(b) > 0 {
		n := Cut(b)
		out = append(out, Chunk{ID: ChunkID(b[:n]), Offset: off, Data: b[:n]})
		b = b[n:]
		off += int64(n)
	}
	return out
}
