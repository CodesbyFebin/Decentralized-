// Package envelope implements dh/v1 signed envelopes.
//
// An envelope carries a canonical JSON payload, the signer's dh1 identity,
// the public key that produced the signature, and an Ed25519 signature over
//
//	"decentralized.host/" + kind + "/v1\n" + canonical(payload)
//
// The kind prefix is the signing domain: a signature produced for one kind
// can never verify as another kind.
package envelope

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/identity"
)

// Signing domains used by dh/v1.
const (
	KindEnroll        = "enroll"
	KindAssignment    = "assignment"
	KindBundle        = "bundle"
	KindObservation   = "observation"
	KindFacts         = "facts"
	KindCapability    = "capability"
	KindRotation      = "rotation"
	KindWGBinding     = "wg-binding"
	KindRoster        = "roster"
	KindCheckpoint    = "audit-checkpoint"
	KindLedger        = "host-ledger"
	KindReplica       = "replica-evidence"
	KindRepair        = "repair-evidence"
	KindCert          = "cert-evidence"
	KindRoute         = "route-evidence"
	KindRequest       = "request"
	KindFedOffer      = "federation-offer"
	KindFedAgreement  = "federation-agreement"
	KindFedPlacement  = "federation-placement"
	KindFedStatus     = "federation-status"
	KindFedRevocation = "federation-revocation"
	KindArtifact      = "artifact-attestation"
	KindBackup        = "cp-backup"
	KindExport        = "export"
	KindChaosReport   = "chaos-report"
)

// Envelope is a signed, canonical protocol message.
type Envelope struct {
	Kind    string          `json:"kind"`
	Signer  string          `json:"signer"`
	Pub     string          `json:"pub"`
	Payload json.RawMessage `json:"payload"`
	Sig     string          `json:"sig"`
}

// SigningInput returns the exact bytes that are signed for kind/payload.
func SigningInput(kind string, canonicalPayload []byte) []byte {
	prefix := "decentralized.host/" + kind + "/v1\n"
	out := make([]byte, 0, len(prefix)+len(canonicalPayload))
	out = append(out, prefix...)
	return append(out, canonicalPayload...)
}

// Sign builds an envelope. signer is normally id.ID; a rotated key signs on
// behalf of the stable genesis ID, so the caller may pass a different signer.
func Sign(id *identity.Identity, signer, kind string, payload any) (*Envelope, error) {
	body, err := canon.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if signer == "" {
		signer = id.ID
	}
	sig := ed25519.Sign(id.Priv, SigningInput(kind, body))
	return &Envelope{
		Kind:    kind,
		Signer:  signer,
		Pub:     id.PubString(),
		Payload: body,
		Sig:     base64.RawURLEncoding.EncodeToString(sig),
	}, nil
}

// MustSign panics on encoding errors; use for payloads built in code.
func MustSign(id *identity.Identity, kind string, payload any) *Envelope {
	env, err := Sign(id, "", kind, payload)
	if err != nil {
		panic(err)
	}
	return env
}

// Verification errors are stable strings: the console and the conformance
// suite show them verbatim.
var (
	ErrKind         = errors.New("envelope kind mismatch")
	ErrNotCanonical = errors.New("payload has no canonical form")
	ErrBadKey       = errors.New("public key is malformed")
	ErrBadSig       = errors.New("signature does not verify")
	ErrSignerKey    = errors.New("public key is not a valid key for signer")
)

// VerifySelf checks the signature with the embedded public key and checks
// that the embedded key's dh1 ID equals Signer. Use it for self-certifying
// messages (enrollment, genesis bindings).
func (e *Envelope) VerifySelf(kind string) error {
	pub, err := e.verifyWith(kind)
	if err != nil {
		return err
	}
	if identity.NodeID(pub) != e.Signer {
		return ErrSignerKey
	}
	return nil
}

// VerifyKey checks the signature and that the embedded key is one of the
// allowed keys (wire-encoded) for the signer.
func (e *Envelope) VerifyKey(kind string, allowed ...string) error {
	if _, err := e.verifyWith(kind); err != nil {
		return err
	}
	for _, k := range allowed {
		if k == e.Pub {
			return nil
		}
	}
	return ErrSignerKey
}

func (e *Envelope) verifyWith(kind string) (ed25519.PublicKey, error) {
	if e == nil {
		return nil, errors.New("nil envelope")
	}
	if e.Kind != kind {
		return nil, fmt.Errorf("%w: got %q want %q", ErrKind, e.Kind, kind)
	}
	// The signature covers the canonical form. Transport may re-encode the
	// payload (for example encoding/json escaping "<" as < inside a
	// RawMessage), so verifiers canonicalize first. Payloads that have no
	// canonical form (duplicate keys, floats, invalid UTF-8) are rejected.
	c, err := canon.Canonicalize(e.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotCanonical, err)
	}
	e.Payload = c
	pub, err := identity.DecodePub(e.Pub)
	if err != nil {
		return nil, ErrBadKey
	}
	sig, err := base64.RawURLEncoding.DecodeString(e.Sig)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, ErrBadSig
	}
	if !ed25519.Verify(pub, SigningInput(kind, e.Payload), sig) {
		return nil, ErrBadSig
	}
	return pub, nil
}

// Decode unmarshals the payload into v. Call only after verification.
func (e *Envelope) Decode(v any) error { return json.Unmarshal(e.Payload, v) }

// Digest is the BLAKE3 content address of the envelope's canonical form.
func (e *Envelope) Digest() string { return Digest(e) }
