package envelope

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"decentralized.host/pkg/identity"
)

type obs struct {
	Node string `json:"node"`
	Seq  int64  `json:"seq"`
}

func TestSignVerify(t *testing.T) {
	id := identity.FromSeed(make([]byte, 32))
	env := MustSign(id, KindObservation, obs{Node: id.ID, Seq: 4})
	if err := env.VerifySelf(KindObservation); err != nil {
		t.Fatal(err)
	}
	if err := env.VerifyKey(KindObservation, id.PubString()); err != nil {
		t.Fatal(err)
	}
}

func TestDomainSeparation(t *testing.T) {
	id := identity.FromSeed(make([]byte, 32))
	env := MustSign(id, KindObservation, obs{Node: id.ID, Seq: 4})
	env.Kind = KindAssignment
	if err := env.VerifySelf(KindAssignment); !errors.Is(err, ErrBadSig) {
		t.Fatalf("cross-domain signature accepted: %v", err)
	}
}

func TestTamperedFieldRejected(t *testing.T) {
	id := identity.FromSeed(make([]byte, 32))
	env := MustSign(id, KindObservation, obs{Node: id.ID, Seq: 4})
	env.Payload = json.RawMessage(`{"node":"` + id.ID + `","seq":5}`)
	if err := env.VerifySelf(KindObservation); !errors.Is(err, ErrBadSig) {
		t.Fatalf("tampered payload accepted: %v", err)
	}
}

// A re-encoded but equivalent payload verifies (the signature covers the
// canonical form); an ambiguous payload does not.
func TestCanonicalizationOnVerify(t *testing.T) {
	id := identity.FromSeed(make([]byte, 32))
	type msg struct {
		Node   string `json:"node"`
		Detail string `json:"detail"`
	}
	env := MustSign(id, KindObservation, msg{Node: id.ID, Detail: "a <url> & more"})
	// encoding/json HTML-escapes RawMessage content in transit; reproduce
	// that re-encoding byte for byte.
	got := *env
	esc := func(hex string) string { return string([]byte{'\\', 'u', '0', '0'}) + hex }
	got.Payload = json.RawMessage(strings.NewReplacer("<", esc("3c"), ">", esc("3e"), "&", esc("26")).Replace(string(env.Payload)))
	if string(got.Payload) == string(env.Payload) {
		t.Fatal("test did not re-encode the payload")
	}
	if err := got.VerifySelf(KindObservation); err != nil {
		t.Fatalf("equivalent re-encoding rejected: %v", err)
	}
	reordered := MustSign(id, KindObservation, obs{Node: id.ID, Seq: 4})
	reordered.Payload = json.RawMessage(`{"seq":4,"node":"` + id.ID + `"}`)
	if err := reordered.VerifySelf(KindObservation); err != nil {
		t.Fatalf("key order must not matter: %v", err)
	}
	dup := MustSign(id, KindObservation, obs{Node: id.ID, Seq: 4})
	dup.Payload = json.RawMessage(`{"node":"` + id.ID + `","seq":4,"seq":5}`)
	if err := dup.VerifySelf(KindObservation); !errors.Is(err, ErrNotCanonical) {
		t.Fatalf("duplicate keys accepted: %v", err)
	}
}

func TestWrongKeyRejected(t *testing.T) {
	a := identity.FromSeed(make([]byte, 32))
	seed := make([]byte, 32)
	seed[0] = 1
	b := identity.FromSeed(seed)
	// b signs claiming to be a.
	env, _ := Sign(b, a.ID, KindObservation, obs{Node: a.ID, Seq: 1})
	if err := env.VerifySelf(KindObservation); !errors.Is(err, ErrSignerKey) {
		t.Fatalf("impersonation accepted by VerifySelf: %v", err)
	}
	if err := env.VerifyKey(KindObservation, a.PubString()); !errors.Is(err, ErrSignerKey) {
		t.Fatalf("impersonation accepted by VerifyKey: %v", err)
	}
}

func TestNodeIDFormat(t *testing.T) {
	id := identity.FromSeed(make([]byte, 32))
	if !identity.IDPattern.MatchString(id.ID) {
		t.Fatal(id.ID)
	}
}
