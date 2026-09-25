package storage

import (
	"encoding/hex"
	"sort"

	"github.com/zeebo/blake3"
)

// MerkleRoot is the dh/v1 Merkle root over a set of chunk ids (§7.4):
// ids are de-duplicated and sorted; leaf = BLAKE3(0x00 || id);
// node = BLAKE3(0x01 || left || right); an odd node is promoted unchanged;
// the empty set hashes to BLAKE3(0x02).
func MerkleRoot(ids []string) string {
	set := uniq(ids)
	if len(set) == 0 {
		sum := blake3.Sum256([]byte{0x02})
		return "b3:" + hex.EncodeToString(sum[:])
	}
	level := make([][32]byte, len(set))
	for i, id := range set {
		level[i] = blake3.Sum256(append([]byte{0x00}, id...))
	}
	for len(level) > 1 {
		var next [][32]byte
		for i := 0; i < len(level); i += 2 {
			if i+1 == len(level) {
				next = append(next, level[i])
				continue
			}
			buf := make([]byte, 0, 65)
			buf = append(buf, 0x01)
			buf = append(buf, level[i][:]...)
			buf = append(buf, level[i+1][:]...)
			next = append(next, blake3.Sum256(buf))
		}
		level = next
	}
	return "b3:" + hex.EncodeToString(level[0][:])
}

// Buckets partitions ids by the first byte of their hash and returns the
// Merkle root of each of the 256 buckets. Two replicas exchange bucket roots
// and only list the buckets that differ.
func Buckets(ids []string) [256]string {
	var parts [256][]string
	for _, id := range uniq(ids) {
		if len(id) < 5 {
			continue
		}
		b, err := hex.DecodeString(id[3:5])
		if err != nil {
			continue
		}
		parts[b[0]] = append(parts[b[0]], id)
	}
	var out [256]string
	for i := range parts {
		out[i] = MerkleRoot(parts[i])
	}
	return out
}

// BucketOf returns the bucket index of a chunk id.
func BucketOf(id string) int {
	if len(id) < 5 {
		return -1
	}
	b, err := hex.DecodeString(id[3:5])
	if err != nil {
		return -1
	}
	return int(b[0])
}

func uniq(ids []string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
