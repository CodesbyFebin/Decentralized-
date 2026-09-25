package control

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"decentralized.host/pkg/canon"
)

func canonJSON(v any) ([]byte, error) { return canon.Marshal(v) }

func canonJSONWire(v any) ([]byte, error) { return canon.Wire(v) }

func nowMs() int64 { return time.Now().UnixMilli() }

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// unixMilliOrZero maps the zero time (for example raft's last contact on the
// leader, which never contacts itself) to 0 instead of year 1.
func unixMilliOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
