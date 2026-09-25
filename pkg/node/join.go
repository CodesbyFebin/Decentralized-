package node

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// JoinToken is what `dh node invite` prints and an operator hands to a new
// host out of band. It pins the cluster root: everything the host later
// accepts must chain to Root.
type JoinToken struct {
	Cluster    string   `json:"cluster"`
	Root       string   `json:"root"`
	RootCACert string   `json:"rootCaCert"`
	Endpoints  []string `json:"endpoints"`
	TLS        bool     `json:"tls"`
	Capability string   `json:"capability"` // dhcap1 root-signed node.join with a single-use nonce
}

const joinPrefix = "dhjoin1."

// EncodeJoinToken serializes a join token.
func EncodeJoinToken(t JoinToken) string {
	b, _ := json.Marshal(t)
	return joinPrefix + base64.RawURLEncoding.EncodeToString(b)
}

// DecodeJoinToken parses a join token.
func DecodeJoinToken(s string) (JoinToken, error) {
	var t JoinToken
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, joinPrefix) {
		return t, errors.New("join token must start with dhjoin1.")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, joinPrefix))
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if t.Cluster == "" || t.Root == "" || len(t.Endpoints) == 0 || t.Capability == "" {
		return t, errors.New("join token is incomplete")
	}
	return t, nil
}
