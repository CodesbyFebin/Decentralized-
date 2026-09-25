package policy

import (
	"testing"

	"decentralized.host/pkg/api"
)

func base() Input {
	return Input{
		Policy: Default(),
		NodeID: "dh1host",
		Assignment: api.Assignment{ID: "wiki/r0", App: "wiki", Node: "dh1host", Generation: 3, Desired: "running",
			Runtime: "process", Image: "dh-beacon@b3:aa", Digest: "b3:aa", Tiers: []string{"trusted"},
			Resources: api.Resources{CPUMilli: 500, MemBytes: 64 << 20}},
		SignatureOK: true, CapabilityOK: true, PlaneTrusted: true, Fresh: true, Attested: true,
	}
}

func TestAdmitHappyPath(t *testing.T) {
	d := Admit(base())
	if !d.Allowed || d.Code != CodeOK {
		t.Fatalf("%+v", d)
	}
}

func TestDenials(t *testing.T) {
	cases := map[string]struct {
		mut  func(*Input)
		code string
	}{
		"forged signature":  {func(i *Input) { i.SignatureOK = false; i.SignatureDetail = "signature does not verify" }, CodeSignature},
		"wrong host":        {func(i *Input) { i.Assignment.Node = "dh1other" }, CodeWrongHost},
		"stale generation":  {func(i *Input) { i.PriorGeneration = 5 }, CodeStaleGen},
		"capability":        {func(i *Input) { i.CapabilityOK = false; i.CapabilityWhy = "capability expired" }, CodeCapability},
		"revoked":           {func(i *Input) { i.Revoked = true }, CodeRevoked},
		"frozen":            {func(i *Input) { i.Frozen = true }, CodeFrozen},
		"stale plane":       {func(i *Input) { i.Fresh = false }, CodeOffline},
		"future bundle":     {func(i *Input) { i.ClockSkewMs = 60_000 }, CodeClockSkew},
		"tier":              {func(i *Input) { i.Assignment.Tiers = []string{"community"} }, CodeTier},
		"runtime":           {func(i *Input) { i.Policy.AllowRuntimes = []string{"docker"} }, CodeRuntime},
		"unpinned":          {func(i *Input) { i.Assignment.Image = "evil:latest"; i.Assignment.Digest = "" }, CodeDigest},
		"unsigned artifact": {func(i *Input) { i.Policy.RequireImageSignature = true; i.Attested = false }, CodeUnsigned},
		"federated":         {func(i *Input) { i.Assignment.Federation = &api.FederationRef{Peer: "b"} }, CodeFederated},
		"workload cap":      {func(i *Input) { i.Policy.MaxWorkloads = 2; i.UsedWorkloads = 2 }, CodeWorkloads},
		"cpu cap":           {func(i *Input) { i.Policy.MaxCPUMilli = 600; i.UsedCPUMilli = 200 }, CodeCPU},
		"memory cap":        {func(i *Input) { i.Policy.MaxMemBytes = 64 << 20; i.UsedMemBytes = 1 }, CodeMem},
		"untrusted plane":   {func(i *Input) { i.PlaneTrusted = false; i.PlaneDetail = "roster not signed by pinned root" }, CodeUntrustedPlane},
		"corrupt ledger":    {func(i *Input) { i.LedgerCorrupt = true }, CodeLedger},
	}
	for name, tc := range cases {
		in := base()
		tc.mut(&in)
		d := Admit(in)
		if d.Allowed || d.Code != tc.code {
			t.Errorf("%s: got %+v", name, d)
		}
		if d.Hold {
			t.Errorf("%s: nothing was admitted before, must not hold", name)
		}
	}
}

// Control-plane outage is not workload outage: the admitted generation is
// held while new work is refused.
func TestHoldSemantics(t *testing.T) {
	for _, mut := range []func(*Input){
		func(i *Input) { i.Fresh = false },
		func(i *Input) { i.Frozen = true },
		func(i *Input) { i.Revoked = true },
		func(i *Input) { i.PlaneTrusted = false },
	} {
		in := base()
		in.PriorGeneration, in.PriorRunning = 3, true
		mut(&in)
		d := Admit(in)
		if d.Allowed || !d.Hold {
			t.Fatalf("admitted work must be held: %+v", d)
		}
		// A new generation under the same condition is refused, not held.
		in.Assignment.Generation = 4
		if d := Admit(in); d.Allowed || d.Hold {
			t.Fatalf("new generation must be refused: %+v", d)
		}
	}
}

func TestStopIsHonoredWhileRevoked(t *testing.T) {
	in := base()
	in.Revoked = true
	in.Assignment.Desired = "stopped"
	if d := Admit(in); !d.Allowed || d.Code != CodeStop {
		t.Fatalf("%+v", d)
	}
}

func TestParseDefaultYAML(t *testing.T) {
	p, err := Parse([]byte(DefaultYAML))
	if err != nil {
		t.Fatal(err)
	}
	if p.MaxCPUMilli != 32000 || p.MaxMemBytes != 128<<30 || p.FreshWindowMs != 10000 || p.OfflineAdmission != "deny" {
		t.Fatalf("%+v", p)
	}
	if _, err := Parse([]byte("nonsense: 1\n")); err == nil {
		t.Fatal("unknown key accepted")
	}
}
