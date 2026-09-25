package capability

import (
	"testing"

	"decentralized.host/pkg/identity"
)

func seed(b byte) *identity.Identity {
	s := make([]byte, 32)
	s[0] = b
	return identity.FromSeed(s)
}

// root → member → assignment, the shape used for host admission.
func chain(t *testing.T) (root, member, host *identity.Identity, tok *Token) {
	root, member, host = seed(1), seed(2), seed(3)
	del, err := Mint(root, Caveats{Actions: []string{"workload.*", "volume.*"}, Resources: []string{"*"}, Expires: 10_000}, member.PubString(), "roster")
	if err != nil {
		t.Fatal(err)
	}
	tok, err = del.Attenuate(member, Caveats{
		Actions: []string{"workload.admit"}, Resources: []string{"app/wiki/r0"}, Audience: host.ID,
		Generation: 7, Digest: "b3:abc", CPUMaxMilli: 500, MemMaxBytes: 512 << 20, Expires: 5_000,
	}, "", "assignment")
	if err != nil {
		t.Fatal(err)
	}
	return
}

func good(host *identity.Identity) Request {
	return Request{Action: "workload.admit", Resource: "app/wiki/r0", Audience: host.ID, Now: 1_000, Generation: 7, Digest: "b3:abc", CPUMilli: 500, MemBytes: 256 << 20}
}

func TestChainAuthorizes(t *testing.T) {
	root, _, host, tok := chain(t)
	wire := tok.Encode()
	dec, err := Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	if res := Verify(dec, []string{root.PubString()}, good(host)); !res.OK {
		t.Fatal(res.Reason)
	}
}

func TestChainRejections(t *testing.T) {
	root, member, host, tok := chain(t)
	roots := []string{root.PubString()}
	mut := map[string]func(r *Request){
		"wrong audience":   func(r *Request) { r.Audience = member.ID },
		"stale generation": func(r *Request) { r.Generation = 6 },
		"other digest":     func(r *Request) { r.Digest = "b3:evil" },
		"cpu over":         func(r *Request) { r.CPUMilli = 501 },
		"mem over":         func(r *Request) { r.MemBytes = 1 << 30 },
		"expired":          func(r *Request) { r.Now = 5_000 },
		"other resource":   func(r *Request) { r.Resource = "app/billing/r0" },
		"widened action":   func(r *Request) { r.Action = "volume.delete" },
		"exec":             func(r *Request) { r.Action = "exec.shell" },
	}
	for name, m := range mut {
		r := good(host)
		m(&r)
		if res := Verify(tok, roots, r); res.OK {
			t.Errorf("%s: accepted", name)
		}
	}
	if res := Verify(tok, []string{member.PubString()}, good(host)); res.OK || res.Block != 0 {
		t.Errorf("unanchored chain accepted: %+v", res)
	}
}

func TestCannotWidenOrForge(t *testing.T) {
	root, member, host, _ := chain(t)
	del, _ := Mint(root, Caveats{Actions: []string{"workload.admit"}, Resources: []string{"app/wiki/*"}}, member.PubString(), "")
	// The member tries to grant itself exec and every resource.
	wide, err := del.Attenuate(member, Caveats{Actions: []string{"*"}, Resources: []string{"*"}}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	r := good(host)
	r.Action, r.Resource = "exec.shell", "app/billing/r0"
	if res := Verify(wide, []string{root.PubString()}, r); res.OK || res.Block != 0 {
		t.Fatalf("widening accepted: %+v", res)
	}
	// Someone other than the delegated key cannot attenuate.
	if _, err := del.Attenuate(host, Caveats{Actions: []string{"workload.admit"}, Resources: []string{"*"}}, "", ""); err == nil {
		t.Fatal("non-delegated key attenuated")
	}
	// A block signed by a non-delegated key is rejected by the verifier too.
	forged, _ := Mint(host, Caveats{Actions: []string{"*"}, Resources: []string{"*"}}, "", "")
	spliced := &Token{Blocks: append(append(del.Blocks[:0:0], del.Blocks...), forged.Blocks...)}
	if res := Verify(spliced, []string{root.PubString()}, good(host)); res.OK || res.Block != 1 {
		t.Fatalf("spliced block accepted: %+v", res)
	}
}

func TestSealedTokenCannotBeExtended(t *testing.T) {
	root, _, host, tok := chain(t)
	if _, err := tok.Attenuate(host, Caveats{Actions: []string{"x"}, Resources: []string{"*"}}, "", ""); err == nil {
		t.Fatal("sealed token extended")
	}
	_ = root
}
