// Package policy holds a host's local sovereign policy and the pure
// admission decision.
//
// The policy file lives on the host and is never sent by the control plane.
// The control plane can only read the host-signed summary. Admit is a pure
// function of (policy, assignment, verification results, local state), so
// the same decision logic is exercised by unit tests, the host agent and the
// conformance suite.
package policy

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/manifest"
)

// Policy is the host-local sovereign policy.
type Policy struct {
	Sovereign               bool
	MaxWorkloads            int64
	MaxCPUMilli             int64
	MaxMemBytes             int64
	AcceptTiers             []string
	DenyImagesWithoutDigest bool
	RequireImageSignature   bool
	TrustedPublishers       []string // wire keys; "cluster-root" = the pinned root key
	AllowRuntimes           []string
	AllowIsolation          []string
	AllowFederated          bool
	AllowExec               bool
	OfflineAdmission        string // deny (hold existing, refuse new) | stop (stop everything when offline)
	MaxClockSkewMs          int64
	FreshWindowMs           int64
	StorageQuotaBytes       int64
}

type rawPolicy struct {
	Sovereign               *bool    `yaml:"sovereign"`
	MaxWorkloads            int64    `yaml:"maxWorkloads"`
	MaxCPU                  string   `yaml:"maxCPU"`
	MaxMem                  string   `yaml:"maxMem"`
	AcceptTiers             []string `yaml:"acceptTiers"`
	DenyImagesWithoutDigest *bool    `yaml:"denyImagesWithoutDigest"`
	RequireImageSignature   bool     `yaml:"requireImageSignature"`
	TrustedPublishers       []string `yaml:"trustedPublishers"`
	AllowRuntimes           []string `yaml:"allowRuntimes"`
	AllowIsolation          []string `yaml:"allowIsolation"`
	AllowFederated          bool     `yaml:"allowFederated"`
	AllowExec               bool     `yaml:"allowExec"`
	OfflineAdmission        string   `yaml:"offlineAdmission"`
	MaxClockSkew            string   `yaml:"maxClockSkew"`
	FreshWindow             string   `yaml:"freshWindow"`
	StorageQuota            string   `yaml:"storageQuota"`
}

// Default is the policy written for a new host.
func Default() Policy {
	return Policy{
		Sovereign: true, MaxWorkloads: 40, MaxCPUMilli: 32000, MaxMemBytes: 128 << 30,
		AcceptTiers: []string{"local", "trusted"}, DenyImagesWithoutDigest: true,
		AllowRuntimes: []string{"process", "docker"}, AllowIsolation: []string{"PRIVATE", "RESTRICTED"}, OfflineAdmission: "deny",
		MaxClockSkewMs: 30_000, FreshWindowMs: 10_000, StorageQuotaBytes: 10 << 30,
	}
}

// DefaultYAML is the commented policy file created on first start.
const DefaultYAML = `# Host-local sovereign policy. The control plane cannot change this file.
# Every assignment is checked against it before anything executes.
sovereign: true
maxWorkloads: 40
maxCPU: "32"
maxMem: 128Gi
acceptTiers: [local, trusted]
denyImagesWithoutDigest: true
# When true, only artifacts attested by a trusted publisher key run here.
# "cluster-root" means the root key this host pinned when it joined.
requireImageSignature: false
trustedPublishers: [cluster-root]
allowRuntimes: [process, docker]
allowFederated: false
allowExec: false
# deny: keep admitted work, refuse new work while the control plane is stale.
# stop: also stop admitted work while the control plane is stale.
offlineAdmission: deny
maxClockSkew: 30s
freshWindow: 10s
storageQuota: 10Gi
`

// Parse reads a policy file.
func Parse(src []byte) (Policy, error) {
	var r rawPolicy
	dec := yaml.NewDecoder(bytes.NewReader(src))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return Policy{}, fmt.Errorf("policy: %w", err)
	}
	p := Default()
	if r.Sovereign != nil {
		p.Sovereign = *r.Sovereign
	}
	if r.MaxWorkloads != 0 {
		p.MaxWorkloads = r.MaxWorkloads
	}
	if r.MaxCPU != "" {
		v, err := manifest.ParseCPU(r.MaxCPU)
		if err != nil {
			return Policy{}, fmt.Errorf("policy: maxCPU: %w", err)
		}
		p.MaxCPUMilli = v
	}
	if r.MaxMem != "" {
		v, err := manifest.ParseBytes(r.MaxMem)
		if err != nil {
			return Policy{}, fmt.Errorf("policy: maxMem: %w", err)
		}
		p.MaxMemBytes = v
	}
	if r.AcceptTiers != nil {
		p.AcceptTiers = r.AcceptTiers
	}
	if r.DenyImagesWithoutDigest != nil {
		p.DenyImagesWithoutDigest = *r.DenyImagesWithoutDigest
	}
	p.RequireImageSignature = r.RequireImageSignature
	p.TrustedPublishers = r.TrustedPublishers
	if r.AllowRuntimes != nil {
		p.AllowRuntimes = r.AllowRuntimes
	}
	if r.AllowIsolation != nil {
		p.AllowIsolation = r.AllowIsolation
	}
	p.AllowFederated = r.AllowFederated
	p.AllowExec = r.AllowExec
	if r.OfflineAdmission != "" {
		if r.OfflineAdmission != "deny" && r.OfflineAdmission != "stop" {
			return Policy{}, fmt.Errorf("policy: offlineAdmission must be deny or stop")
		}
		p.OfflineAdmission = r.OfflineAdmission
	}
	for field, dst := range map[string]*int64{r.MaxClockSkew: &p.MaxClockSkewMs, r.FreshWindow: &p.FreshWindowMs} {
		if field == "" {
			continue
		}
		v, err := manifest.ParseDuration(field)
		if err != nil {
			return Policy{}, fmt.Errorf("policy: duration %q: %w", field, err)
		}
		*dst = v
	}
	if r.StorageQuota != "" {
		v, err := manifest.ParseBytes(r.StorageQuota)
		if err != nil {
			return Policy{}, fmt.Errorf("policy: storageQuota: %w", err)
		}
		p.StorageQuotaBytes = v
	}
	sort.Strings(p.AcceptTiers)
	sort.Strings(p.AllowRuntimes)
	sort.Strings(p.AllowIsolation)
	return p, nil
}

// LoadOrCreate reads path, writing DefaultYAML first if it does not exist.
func LoadOrCreate(path string) (Policy, error) {
	src, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(DefaultYAML), 0o600); err != nil {
			return Policy{}, err
		}
		src = []byte(DefaultYAML)
	} else if err != nil {
		return Policy{}, err
	}
	return Parse(src)
}

// Summary is the host-signed, advertised form.
func (p Policy) Summary() api.PolicySummary {
	return api.PolicySummary{
		Sovereign: p.Sovereign, MaxWorkloads: p.MaxWorkloads, MaxCPUMilli: p.MaxCPUMilli, MaxMemBytes: p.MaxMemBytes,
		AcceptTiers: append([]string{}, p.AcceptTiers...), DenyImagesWithoutDigest: p.DenyImagesWithoutDigest,
		RequireImageSignature: p.RequireImageSignature, AllowRuntimes: append([]string{}, p.AllowRuntimes...),
		AllowFederated: p.AllowFederated, AllowExec: p.AllowExec, OfflineAdmission: p.OfflineAdmission,
		MaxClockSkewMs: p.MaxClockSkewMs, StorageQuotaBytes: p.StorageQuotaBytes,
	}
}

// Hash is the content address of the advertised policy.
func (p Policy) Hash() string { return envelope.DomainHash("policy", p.Summary()) }

// ------------------------------------------------------------- admission

// Decision codes. Stable strings: shown in the console and used by the
// conformance suite.
const (
	CodeOK             = "ADMITTED"
	CodeHold           = "HOLD"
	CodeUntrustedPlane = "UNTRUSTED_PLANE"
	CodeSignature      = "ASSIGNMENT_SIGNATURE"
	CodeWrongHost      = "WRONG_HOST"
	CodeCapability     = "CAPABILITY"
	CodeStaleGen       = "STALE_GENERATION"
	CodeRevoked        = "HOST_REVOKED"
	CodeOffline        = "CONTROL_PLANE_STALE"
	CodeFrozen         = "CONTROL_PLANE_FROZEN"
	CodeClockSkew      = "CLOCK_SKEW"
	CodeLedger         = "LEDGER_CORRUPT"
	CodeTier           = "POLICY_TIER"
	CodeRuntime        = "POLICY_RUNTIME"
	CodeIsolation      = "POLICY_ISOLATION"
	CodeDigest         = "POLICY_DIGEST"
	CodeUnsigned       = "POLICY_ARTIFACT_SIGNATURE"
	CodeWorkloads      = "POLICY_WORKLOAD_CAP"
	CodeCPU            = "POLICY_CPU_CAP"
	CodeMem            = "POLICY_MEMORY_CAP"
	CodeFederated      = "POLICY_FEDERATED"
	CodeStop           = "STOP"
)

// Input is everything admission may look at. The caller performs the
// cryptographic checks and passes their results; Admit decides.
type Input struct {
	Policy          Policy
	NodeID          string
	Assignment      api.Assignment
	SignatureOK     bool   // assignment envelope verifies against a roster member
	SignatureDetail string // why not
	CapabilityOK    bool
	CapabilityWhy   string
	PlaneTrusted    bool // bundle signature, roster and pinned root verified
	PlaneDetail     string
	Fresh           bool  // bundle within the fresh window
	Frozen          bool  // bundle says the control plane is frozen
	ClockSkewMs     int64 // bundle issued minus host now (positive = bundle from the future)
	Revoked         bool  // bundle lists this host as revoked
	LedgerCorrupt   bool
	PriorGeneration int64 // highest generation already admitted for this assignment id, 0 = none
	PriorRunning    bool  // an admitted instance of PriorGeneration is running
	Attested        bool  // artifact attested by a trusted publisher
	AttestDetail    string
	SandboxOK       bool   // the host can create sandboxes
	SandboxDetail   string // why not
	// Resources already committed to other admitted workloads.
	UsedWorkloads int64
	UsedCPUMilli  int64
	UsedMemBytes  int64
}

// Decision is the explained outcome.
type Decision struct {
	Allowed bool        `json:"allowed"`
	Hold    bool        `json:"hold"` // keep the prior admitted instance, do not start a new one
	Code    string      `json:"code"`
	Reason  string      `json:"reason"`
	Checks  []api.Check `json:"checks"`
}

// Admit evaluates an assignment. Checks run in a fixed order; the first
// failure decides. Holding a prior admitted instance is only ever allowed
// for the same generation — a stale plane can never start new work.
func Admit(in Input) Decision {
	var d Decision
	a := in.Assignment
	pol := in.Policy
	check := func(name string, ok bool, detail string) bool {
		d.Checks = append(d.Checks, api.Check{Name: name, OK: ok, Detail: detail})
		return ok
	}
	deny := func(code, reason string) Decision {
		d.Allowed, d.Code, d.Reason = false, code, reason
		return d
	}
	hold := func(code, reason string) Decision {
		d.Allowed, d.Hold, d.Code, d.Reason = false, true, code, reason
		return d
	}
	samePrior := in.PriorRunning && in.PriorGeneration == a.Generation && a.Desired == "running"

	if !check("ledger", !in.LedgerCorrupt, "host ledger verifies") {
		if samePrior {
			return hold(CodeLedger, "host ledger failed verification; holding admitted work, refusing new work")
		}
		return deny(CodeLedger, "host ledger failed verification; refusing new work until an operator inspects it")
	}
	if !check("plane", in.PlaneTrusted, in.PlaneDetail) {
		if samePrior {
			return hold(CodeUntrustedPlane, "control-plane bundle is not trusted ("+in.PlaneDetail+"); holding admitted work")
		}
		return deny(CodeUntrustedPlane, "control-plane bundle is not trusted: "+in.PlaneDetail)
	}
	if !check("assignment-signature", in.SignatureOK, in.SignatureDetail) {
		if samePrior {
			return hold(CodeSignature, "replacement assignment is not validly signed; holding admitted work")
		}
		return deny(CodeSignature, "assignment signature: "+in.SignatureDetail)
	}
	if !check("audience", a.Node == in.NodeID, "assignment is for "+a.Node) {
		return deny(CodeWrongHost, "assignment names another host")
	}
	if !check("generation", a.Generation >= in.PriorGeneration, fmt.Sprintf("assignment generation %d, admitted %d", a.Generation, in.PriorGeneration)) {
		return deny(CodeStaleGen, fmt.Sprintf("stale generation %d rejected; generation %d already admitted", a.Generation, in.PriorGeneration))
	}
	// Clock before capability: a skewed host clock makes a valid capability
	// look not-yet-valid or expired; the precise cause is the clock.
	if !check("clock", in.ClockSkewMs <= pol.MaxClockSkewMs, fmt.Sprintf("bundle issued %dms relative to host clock (limit %dms)", in.ClockSkewMs, pol.MaxClockSkewMs)) {
		if samePrior {
			return hold(CodeClockSkew, "bundle timestamp is ahead of the host clock beyond the skew limit; holding admitted work")
		}
		return deny(CodeClockSkew, fmt.Sprintf("bundle is %dms in the future (limit %dms); refusing new work", in.ClockSkewMs, pol.MaxClockSkewMs))
	}
	if !check("capability", in.CapabilityOK, in.CapabilityWhy) {
		if samePrior {
			return hold(CodeCapability, "capability check failed for the replacement ("+in.CapabilityWhy+"); holding admitted work")
		}
		return deny(CodeCapability, "capability: "+in.CapabilityWhy)
	}
	if a.Desired == "stopped" {
		check("desired", true, "desired state is stopped")
		d.Allowed, d.Code, d.Reason = true, CodeStop, "signed stop accepted"
		return d
	}
	if !check("revocation", !in.Revoked, "host approval is current") {
		if samePrior {
			return hold(CodeRevoked, "host approval revoked; admitted work continues, new work is refused")
		}
		return deny(CodeRevoked, "host approval revoked; refusing new work")
	}
	if !check("frozen", !in.Frozen, "control plane is not frozen") {
		if samePrior {
			return hold(CodeFrozen, "control plane frozen; holding admitted work")
		}
		return deny(CodeFrozen, "control plane frozen; refusing new work")
	}
	if !check("fresh", in.Fresh, fmt.Sprintf("bundle within %dms fresh window", pol.FreshWindowMs)) {
		if samePrior {
			return hold(CodeOffline, "control plane stale; holding previously admitted workload")
		}
		return deny(CodeOffline, "control plane stale; host refuses new work")
	}
	if samePrior {
		check("unchanged", true, "same generation already admitted and running")
		d.Allowed, d.Code, d.Reason = true, CodeOK, "already admitted; digest and policy unchanged"
		return d
	}
	if !check("tier", intersects(pol.AcceptTiers, a.Tiers), fmt.Sprintf("host accepts [%s], assignment tiers [%s]", strings.Join(pol.AcceptTiers, ", "), strings.Join(a.Tiers, ", "))) {
		return deny(CodeTier, "trust tier mismatch")
	}
	if !check("runtime", contains(pol.AllowRuntimes, a.Runtime), "runtime "+a.Runtime) {
		return deny(CodeRuntime, "runtime "+a.Runtime+" not allowed by host policy")
	}
	if a.Isolation != "" {
		// Fail closed: an isolation request on a host that cannot sandbox is
		// refused, never run without the boundary.
		if !check("sandbox", in.SandboxOK, in.SandboxDetail) {
			return deny(CodeIsolation, "isolation "+a.Isolation+" requested but the host cannot sandbox: "+in.SandboxDetail)
		}
		if !check("isolation", contains(pol.AllowIsolation, a.Isolation), "isolation "+a.Isolation) {
			return deny(CodeIsolation, "isolation profile "+a.Isolation+" not allowed by host policy")
		}
	}
	digestOK := strings.HasPrefix(a.Digest, "b3:") || strings.HasPrefix(a.Digest, "sha256:")
	digestOK = digestOK && strings.HasSuffix(a.Image, "@"+a.Digest)
	if !check("digest", !pol.DenyImagesWithoutDigest || digestOK, "image "+a.Image) {
		return deny(CodeDigest, "image is not digest-pinned")
	}
	if !check("artifact-signature", !pol.RequireImageSignature || in.Attested, in.AttestDetail) {
		return deny(CodeUnsigned, "artifact is not attested by a trusted publisher: "+in.AttestDetail)
	}
	if !check("federated", a.Federation == nil || pol.AllowFederated, "federated placement") {
		return deny(CodeFederated, "host policy does not accept federated work")
	}
	if !check("workloads", in.UsedWorkloads+1 <= pol.MaxWorkloads, fmt.Sprintf("%d/%d workloads", in.UsedWorkloads+1, pol.MaxWorkloads)) {
		return deny(CodeWorkloads, fmt.Sprintf("workload cap reached (%d/%d)", in.UsedWorkloads, pol.MaxWorkloads))
	}
	if !check("cpu", in.UsedCPUMilli+a.Resources.CPUMilli <= pol.MaxCPUMilli, fmt.Sprintf("%dm of %dm", in.UsedCPUMilli+a.Resources.CPUMilli, pol.MaxCPUMilli)) {
		return deny(CodeCPU, "cpu cap exceeded")
	}
	if !check("memory", in.UsedMemBytes+a.Resources.MemBytes <= pol.MaxMemBytes, fmt.Sprintf("%s of %s", manifest.FormatBytes(in.UsedMemBytes+a.Resources.MemBytes), manifest.FormatBytes(pol.MaxMemBytes))) {
		return deny(CodeMem, "memory cap exceeded")
	}
	d.Allowed, d.Code, d.Reason = true, CodeOK, "signature, capability, generation, digest and local policy allow"
	return d
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func intersects(a, b []string) bool {
	for _, x := range a {
		if contains(b, x) {
			return true
		}
	}
	return false
}
