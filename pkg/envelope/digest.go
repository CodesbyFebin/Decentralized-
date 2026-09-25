package envelope

import (
	"encoding/hex"

	"github.com/zeebo/blake3"

	"decentralized.host/pkg/canon"
)

// Digest returns "b3:" + hex(BLAKE3-256(canonical(v))).
func Digest(v any) string {
	return "b3:" + HashBytes(canon.MustMarshal(v))
}

// HashBytes returns hex(BLAKE3-256(b)).
func HashBytes(b []byte) string {
	sum := blake3.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// DomainHash returns "b3:" + hex(BLAKE3-256("decentralized.host/<domain>/v1\n" + canonical(v))).
func DomainHash(domain string, v any) string {
	return "b3:" + HashBytes(SigningInput(domain, canon.MustMarshal(v)))
}
