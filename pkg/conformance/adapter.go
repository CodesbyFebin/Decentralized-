// Package conformance implements the dh/v1 conformance suite: deterministic
// test vectors, the adapter protocol an implementation speaks to be tested,
// the Go reference adapter, and the runner (docs/protocol/conformance.md).
//
// An adapter is any executable that reads one JSON request per line on stdin
// and writes one JSON response per line on stdout:
//
//	→ {"id":"canon/…","op":"canon","input":{…}}
//	← {"id":"canon/…","ok":true,"output":{…}}   or   {"id":…,"ok":false,"error":"<code>"}
//
// The runner compares canonical(output) with the vector's expectation, so
// field order and whitespace in responses do not matter.
package conformance

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/storage"
)

// Error codes adapters return. They are part of the protocol.
const (
	CodeReject       = "reject"        // input has no canonical form
	CodeKindMismatch = "kind-mismatch" // envelope kind differs from the expected domain
	CodeNotCanonical = "not-canonical" // envelope payload has no canonical form
	CodeBadKey       = "bad-key"       // embedded public key malformed
	CodeBadSignature = "bad-signature" // signature does not verify
	CodeSignerKey    = "signer-key"    // key is not allowed for the signer
	CodeBadInput     = "bad-input"     // the adapter could not parse the request
	CodeUnsupported  = "unsupported"   // op not implemented by this adapter
)

// Ops lists every operation in protocol order.
var Ops = []string{"canon", "identity", "digest", "domain-hash", "sign", "verify", "audit-hash", "audit-verify", "capability-mint", "capability-verify", "chunk", "merkle"}

// Error is an adapter failure with a protocol code.
type Error struct{ Code string }

func (e *Error) Error() string { return e.Code }

func fail(code string) error { return &Error{Code: code} }

// Handle is the Go reference adapter.
func Handle(op string, in json.RawMessage) (any, error) {
	switch op {
	case "canon":
		var r struct {
			JSON []byte `json:"json_b64"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		c, err := canon.Canonicalize(r.JSON)
		if err != nil {
			return nil, fail(CodeReject)
		}
		return map[string]any{"canonical_b64": c}, nil

	case "identity":
		var r struct {
			Seed string `json:"seed"`
			Pub  string `json:"pub"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		if r.Seed != "" {
			id, err := seedIdentity(r.Seed)
			if err != nil {
				return nil, err
			}
			return map[string]any{"pub": id.PubString(), "id": id.ID}, nil
		}
		pub, err := identity.DecodePub(r.Pub)
		if err != nil {
			return nil, fail(CodeBadKey)
		}
		return map[string]any{"pub": r.Pub, "id": identity.NodeID(pub)}, nil

	case "digest", "domain-hash":
		var r struct {
			Domain string          `json:"domain"`
			Value  json.RawMessage `json:"value"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		c, err := canon.Canonicalize(r.Value)
		if err != nil {
			return nil, fail(CodeReject)
		}
		if op == "digest" {
			return map[string]any{"hash": "b3:" + envelope.HashBytes(c)}, nil
		}
		return map[string]any{"hash": "b3:" + envelope.HashBytes(envelope.SigningInput(r.Domain, c))}, nil

	case "sign":
		var r struct {
			Seed    string          `json:"seed"`
			Signer  string          `json:"signer"`
			Kind    string          `json:"kind"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		id, err := seedIdentity(r.Seed)
		if err != nil {
			return nil, err
		}
		c, err := canon.Canonicalize(r.Payload)
		if err != nil {
			return nil, fail(CodeReject)
		}
		env, err := envelope.Sign(id, r.Signer, r.Kind, json.RawMessage(c))
		if err != nil {
			return nil, fail(CodeReject)
		}
		return map[string]any{"envelope": env, "signing_input_b64": envelope.SigningInput(r.Kind, c)}, nil

	case "verify":
		var r struct {
			Envelope []byte   `json:"envelope_b64"`
			Kind     string   `json:"kind"`
			Self     bool     `json:"self"`
			Allowed  []string `json:"allowed"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		var env envelope.Envelope
		if json.Unmarshal(r.Envelope, &env) != nil {
			return nil, fail(CodeBadInput)
		}
		var err error
		if r.Self {
			err = env.VerifySelf(r.Kind)
		} else {
			err = env.VerifyKey(r.Kind, r.Allowed...)
		}
		if err != nil {
			return map[string]any{"valid": false, "error": verifyCode(err)}, nil
		}
		return map[string]any{"valid": true, "error": ""}, nil

	case "audit-hash":
		var e audit.Entry
		if json.Unmarshal(in, &e) != nil {
			return nil, fail(CodeBadInput)
		}
		return map[string]any{"hash": audit.ComputeHash(e)}, nil

	case "audit-verify":
		var r struct {
			Entries     []audit.Entry        `json:"entries"`
			StartSeq    int64                `json:"startSeq"`
			StartPrev   string               `json:"startPrev"`
			Checkpoints []*envelope.Envelope `json:"checkpoints"`
			Keys        map[string][]string  `json:"keys"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		br := audit.Verify(r.Entries, r.StartSeq, r.StartPrev)
		if br == nil && len(r.Checkpoints) > 0 {
			br = audit.VerifyCheckpoints(r.Entries, r.Checkpoints, func(s string) []string { return r.Keys[s] })
		}
		if br == nil {
			return map[string]any{"break": nil}, nil
		}
		return map[string]any{"break": map[string]any{"seq": br.Seq, "reason": br.Reason}}, nil

	case "capability-mint":
		var r struct {
			Blocks []struct {
				Seed    string             `json:"seed"`
				Caveats capability.Caveats `json:"caveats"`
				Next    string             `json:"next"`
				Note    string             `json:"note"`
			} `json:"blocks"`
		}
		if json.Unmarshal(in, &r) != nil || len(r.Blocks) == 0 {
			return nil, fail(CodeBadInput)
		}
		var tok capability.Token
		for _, b := range r.Blocks {
			id, err := seedIdentity(b.Seed)
			if err != nil {
				return nil, err
			}
			env, err := envelope.Sign(id, "", envelope.KindCapability, capability.Block{Caveats: b.Caveats, Next: b.Next, Note: b.Note})
			if err != nil {
				return nil, fail(CodeReject)
			}
			tok.Blocks = append(tok.Blocks, env)
		}
		return map[string]any{"token": tok.Encode()}, nil

	case "capability-verify":
		var r struct {
			Token   string   `json:"token"`
			Roots   []string `json:"roots"`
			Request struct {
				Action     string `json:"action"`
				Resource   string `json:"resource"`
				Audience   string `json:"audience"`
				Now        int64  `json:"now"`
				Generation int64  `json:"generation"`
				Digest     string `json:"digest"`
				CPUMilli   int64  `json:"cpuMilli"`
				MemBytes   int64  `json:"memBytes"`
			} `json:"request"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		tok, err := capability.Decode(r.Token)
		if err != nil {
			return map[string]any{"ok": false, "block": -1}, nil
		}
		q := r.Request
		res := capability.Verify(tok, r.Roots, capability.Request{Action: q.Action, Resource: q.Resource, Audience: q.Audience,
			Now: q.Now, Generation: q.Generation, Digest: q.Digest, CPUMilli: q.CPUMilli, MemBytes: q.MemBytes})
		return map[string]any{"ok": res.OK, "block": res.Block}, nil

	case "chunk":
		var r struct {
			Data []byte `json:"data_b64"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		chunks := []map[string]any{}
		for _, c := range storage.ChunkAll(r.Data) {
			chunks = append(chunks, map[string]any{"offset": c.Offset, "length": len(c.Data), "id": c.ID})
		}
		return map[string]any{"chunks": chunks}, nil

	case "merkle":
		var r struct {
			IDs []string `json:"ids"`
		}
		if json.Unmarshal(in, &r) != nil {
			return nil, fail(CodeBadInput)
		}
		return map[string]any{"root": storage.MerkleRoot(r.IDs)}, nil
	}
	return nil, fail(CodeUnsupported)
}

func seedIdentity(seed string) (*identity.Identity, error) {
	b, err := hex.DecodeString(seed)
	if err != nil || len(b) != ed25519.SeedSize {
		return nil, fail(CodeBadInput)
	}
	return identity.FromSeed(b), nil
}

func verifyCode(err error) string {
	switch {
	case errors.Is(err, envelope.ErrKind):
		return CodeKindMismatch
	case errors.Is(err, envelope.ErrNotCanonical):
		return CodeNotCanonical
	case errors.Is(err, envelope.ErrBadKey):
		return CodeBadKey
	case errors.Is(err, envelope.ErrBadSig):
		return CodeBadSignature
	case errors.Is(err, envelope.ErrSignerKey):
		return CodeSignerKey
	}
	return "error: " + err.Error()
}

// request/response are the adapter wire messages.
type request struct {
	ID    string          `json:"id"`
	Op    string          `json:"op"`
	Input json.RawMessage `json:"input"`
}

type response struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Output json.RawMessage `json:"output,omitempty"`
	Error  string          `json:"error,omitempty"`
}

func handleLine(line []byte) response {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return response{OK: false, Error: CodeBadInput}
	}
	out, err := Handle(req.Op, req.Input)
	if err != nil {
		var e *Error
		if errors.As(err, &e) {
			return response{ID: req.ID, Error: e.Code}
		}
		return response{ID: req.ID, Error: fmt.Sprint(err)}
	}
	b, err := canon.Wire(out)
	if err != nil {
		return response{ID: req.ID, Error: err.Error()}
	}
	return response{ID: req.ID, OK: true, Output: b}
}
