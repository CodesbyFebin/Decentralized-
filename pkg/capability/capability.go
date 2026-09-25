// Package capability implements dh/v1 attenuable public-key capabilities.
//
// A token is a chain of signed blocks. Block 0 is signed by a trusted root
// key. Each block names the public key allowed to sign the next block
// (Next). A request is authorized only if it satisfies the caveats of every
// block, so a later block can only narrow authority: attenuation is
// structural, not a convention. Verification needs only the root public key,
// which is why a host can check an assignment while the control plane is
// offline.
//
// Wire form: "dhcap1." + base64url(canonical JSON array of block envelopes).
package capability

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
)

const prefix = "dhcap1."

// Caveats restrict what a block authorizes. Zero values mean "no restriction
// from this block", except Actions and Resources, which must be non-empty.
type Caveats struct {
	Actions     []string `json:"actions"`     // exact or trailing ".*" / "*" wildcard
	Resources   []string `json:"resources"`   // path.Match patterns, e.g. "app/*"
	Audience    string   `json:"audience"`    // dh1 id the request must be for, or ""
	NotBefore   int64    `json:"notBefore"`   // unix ms
	Expires     int64    `json:"expires"`     // unix ms, 0 = none
	Generation  int64    `json:"generation"`  // exact generation, 0 = any
	Digest      string   `json:"digest"`      // exact artifact/image digest, "" = any
	CPUMaxMilli int64    `json:"cpuMaxMilli"` // 0 = unlimited
	MemMaxBytes int64    `json:"memMaxBytes"` // 0 = unlimited
	Nonce       string   `json:"nonce"`       // single-use tokens (join)
}

// Block is the signed unit.
type Block struct {
	Caveats Caveats `json:"caveats"`
	Next    string  `json:"next"` // wire public key that may attenuate; "" = sealed
	Note    string  `json:"note"`
}

// Token is a decoded chain.
type Token struct {
	Blocks []*envelope.Envelope
}

// Request is what the verifier checks against every block.
type Request struct {
	Action     string
	Resource   string
	Audience   string
	Now        int64
	Generation int64
	Digest     string
	CPUMilli   int64
	MemBytes   int64
}

// Mint creates a root-signed token.
func Mint(root *identity.Identity, c Caveats, next, note string) (*Token, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	env, err := envelope.Sign(root, "", envelope.KindCapability, Block{Caveats: c, Next: next, Note: note})
	if err != nil {
		return nil, err
	}
	return &Token{Blocks: []*envelope.Envelope{env}}, nil
}

// Attenuate appends a block signed by holder, which must be the key named by
// the last block's Next.
func (t *Token) Attenuate(holder *identity.Identity, c Caveats, next, note string) (*Token, error) {
	if len(t.Blocks) == 0 {
		return nil, errors.New("capability: empty token")
	}
	var last Block
	if err := t.Blocks[len(t.Blocks)-1].Decode(&last); err != nil {
		return nil, err
	}
	if last.Next == "" {
		return nil, errors.New("capability: token is sealed")
	}
	if last.Next != holder.PubString() {
		return nil, errors.New("capability: holder key is not the delegated next key")
	}
	if err := c.valid(); err != nil {
		return nil, err
	}
	env, err := envelope.Sign(holder, "", envelope.KindCapability, Block{Caveats: c, Next: next, Note: note})
	if err != nil {
		return nil, err
	}
	blocks := append(append([]*envelope.Envelope{}, t.Blocks...), env)
	return &Token{Blocks: blocks}, nil
}

func (c Caveats) valid() error {
	if len(c.Actions) == 0 {
		return errors.New("capability: caveats need at least one action")
	}
	if len(c.Resources) == 0 {
		return errors.New("capability: caveats need at least one resource pattern")
	}
	for _, r := range c.Resources {
		if _, err := path.Match(r, ""); err != nil {
			return fmt.Errorf("capability: resource pattern %q: %w", r, err)
		}
	}
	return nil
}

// Encode returns the wire form.
func (t *Token) Encode() string {
	return prefix + base64.RawURLEncoding.EncodeToString(canon.MustMarshal(t.Blocks))
}

// Decode parses the wire form without verifying it.
func Decode(s string) (*Token, error) {
	if !strings.HasPrefix(s, prefix) {
		return nil, errors.New("capability: missing dhcap1 prefix")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil {
		return nil, fmt.Errorf("capability: %w", err)
	}
	var blocks []*envelope.Envelope
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("capability: %w", err)
	}
	if len(blocks) == 0 || len(blocks) > 16 {
		return nil, errors.New("capability: token must have 1..16 blocks")
	}
	return &Token{Blocks: blocks}, nil
}

// Result explains a verification.
type Result struct {
	OK      bool     `json:"ok"`
	Reason  string   `json:"reason"`
	Block   int      `json:"block"` // index of the failing block, -1 if n/a
	Root    string   `json:"root"`  // root key that anchored the chain
	Signers []string `json:"signers"`
	Last    Block    `json:"-"`
}

// Verify checks the chain against trusted roots and the request.
func Verify(t *Token, roots []string, req Request) Result {
	res := Result{Block: -1}
	if t == nil || len(t.Blocks) == 0 {
		res.Reason = "empty token"
		return res
	}
	expectKey := ""
	for i, env := range t.Blocks {
		var allowed []string
		if i == 0 {
			allowed = roots
		} else {
			allowed = []string{expectKey}
		}
		if err := env.VerifyKey(envelope.KindCapability, allowed...); err != nil {
			res.Block = i
			if i == 0 && errors.Is(err, envelope.ErrSignerKey) {
				res.Reason = "chain is not anchored in a trusted root"
			} else {
				res.Reason = "block signature: " + err.Error()
			}
			return res
		}
		var b Block
		if err := env.Decode(&b); err != nil {
			res.Block, res.Reason = i, "block payload: "+err.Error()
			return res
		}
		if i == 0 {
			res.Root = env.Pub
		}
		res.Signers = append(res.Signers, env.Pub)
		if why := b.Caveats.check(req); why != "" {
			res.Block, res.Reason = i, why
			return res
		}
		if i < len(t.Blocks)-1 && b.Next == "" {
			res.Block, res.Reason = i+1, "block follows a sealed block"
			return res
		}
		expectKey = b.Next
		res.Last = b
	}
	res.OK = true
	return res
}

func (c Caveats) check(r Request) string {
	if !matchAction(c.Actions, r.Action) {
		return fmt.Sprintf("action %q not granted (granted %v)", r.Action, c.Actions)
	}
	if !matchResource(c.Resources, r.Resource) {
		return fmt.Sprintf("resource %q not granted (granted %v)", r.Resource, c.Resources)
	}
	if c.Audience != "" && c.Audience != r.Audience {
		return fmt.Sprintf("audience is %s, request is for %s", c.Audience, r.Audience)
	}
	if c.NotBefore != 0 && r.Now < c.NotBefore {
		return "capability not yet valid"
	}
	if c.Expires != 0 && r.Now >= c.Expires {
		return "capability expired"
	}
	if c.Generation != 0 && c.Generation != r.Generation {
		return fmt.Sprintf("capability is for generation %d, request is generation %d", c.Generation, r.Generation)
	}
	if c.Digest != "" && c.Digest != r.Digest {
		return "capability is bound to a different digest"
	}
	if c.CPUMaxMilli != 0 && r.CPUMilli > c.CPUMaxMilli {
		return fmt.Sprintf("cpu %dm exceeds capability limit %dm", r.CPUMilli, c.CPUMaxMilli)
	}
	if c.MemMaxBytes != 0 && r.MemBytes > c.MemMaxBytes {
		return fmt.Sprintf("memory %d exceeds capability limit %d", r.MemBytes, c.MemMaxBytes)
	}
	return ""
}

func matchAction(granted []string, action string) bool {
	for _, g := range granted {
		switch {
		case g == "*" || g == action:
			return true
		case strings.HasSuffix(g, ".*") && strings.HasPrefix(action, strings.TrimSuffix(g, "*")):
			return true
		}
	}
	return false
}

func matchResource(granted []string, resource string) bool {
	for _, g := range granted {
		if g == "*" || g == resource {
			return true
		}
		if ok, _ := path.Match(g, resource); ok {
			return true
		}
		// "app/*" also covers nested "app/wiki/r0".
		if strings.HasSuffix(g, "/*") && strings.HasPrefix(resource, strings.TrimSuffix(g, "*")) {
			return true
		}
	}
	return false
}

// PubOf is a helper for callers holding a raw key.
func PubOf(k ed25519.PublicKey) string { return identity.EncodePub(k) }
