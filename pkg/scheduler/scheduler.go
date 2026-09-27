// Package scheduler is the deterministic dh/v1 placement engine.
//
// It is a pure function: same inputs, same plan. It never mutates state; the
// reconciler turns a plan into desired assignments, and each host still
// decides whether to admit them.
//
// Placement pipeline per replica:
//   1. Candidate selection: filter nodes by status and eligibility
//   2. Policy enforcement: validate runtime, federation, workload caps
//   3. Trust tier: match declared tiers with node capabilities
//   4. Constraints: architecture, required features
//   5. Capacity: CPU, memory, disk (skipped for stable replicas)
//   6. Anti-affinity: hard (unique node) vs soft (prefer spread)
//   7. Failure-domain spread: distribute across region|zone|host domains
//   8. Scoring: weighted factors (fit, spread, locality, affinity, stability, headroom)
//   9. Tie-breaking: sort by node ID for determinism
//
// Stability: A replica whose current host still passes every filter stays where it is,
// so applying the same desired state twice yields the same placement.
//
// Volume placement: replicas are placed with explicit domain spreading.
// Primary first, then current members from unused domains, then new hosts.
package scheduler

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"decentralized.host/pkg/api"
)

// Node is the scheduler's view of a host.
//
// Placement constraints evaluated in order:
//   - Status: only "ready" hosts are eligible; draining/revoked are ineligible
//   - Tiers: node must advertise at least one tier from Placement.Tiers
//   - Policy: host policy enforces AcceptTiers, AllowRuntimes, MaxWorkloads, and resource caps
//   - Arch: node must match Placement.Arch (if specified)
//   - Features: node must have all Features from Placement (e.g., gpu-nvidia, sse4, avx2)
//   - Capacity: node has sufficient CPUMilli and MemBytes (computed as total-used)
//   - Anti-affinity: hard = one replica per node; soft = prefer spreading
//   - Spread: failure-domain = distribute across region|zone|host domains
//
// Scoring balances multiple factors for optimal placement (not just feasibility).
type Node struct {
	ID        string                // unique node identifier
	Name      string                // human-readable name
	Status    string                // ready | pending | revoked | draining | lost
	Tiers     []string              // tier labels (e.g., compute, storage, standard)
	Region    string                // geographic region for locality and spread
	Zone      string                // availability zone / failure domain
	Host      string                // physical host name for fine-grained domain separation
	Arch      string                // CPU architecture (amd64, arm64, etc.)
	Features  []string              // hardware capabilities (gpu-nvidia, sse4, avx2, etc.)
	CPUMilli  int64                 // total CPU in millicores
	MemBytes  int64                 // total memory in bytes
	DiskFree  int64                 // free disk in bytes, -1 if not measured
	UsedCPU   int64                 // current CPU allocation across all workloads
	UsedMem   int64                 // current memory allocation across all workloads
	Workloads int64                 // count of running workloads (from observations)
	Policy    api.PolicySummary     // node-advertised policy constraints and caps
	UptimeMs  int64                 // milliseconds since last reboot (stability signal)
	Roles     []string              // node roles (e.g., edge-only, dedicated)
}

// DomainKey is the failure-domain identity used for spreading.
func (n Node) DomainKey() string { return n.Region + "|" + n.Zone + "|" + n.Host }

// Row explains one node's evaluation for one replica.
type Row struct {
	Node   string  `json:"node"`
	Name   string  `json:"name"`
	OK     bool    `json:"ok"`
	Stage  string  `json:"stage"`
	Reason string  `json:"reason"`
	Score  float64 `json:"score"`
}

// ReplicaPlan is the decision for one replica.
type ReplicaPlan struct {
	Replica int64  `json:"replica"`
	Node    string `json:"node"` // "" = unschedulable
	Kept    bool   `json:"kept"` // stayed on its current host
	Reason  string `json:"reason"`
	Rows    []Row  `json:"rows"`
}

// Plan is the scheduler output.
type Plan struct {
	App      string        `json:"app"`
	Replicas []ReplicaPlan `json:"replicas"`
}

// Unscheduled returns replicas without a node.
func (p Plan) Unscheduled() []ReplicaPlan {
	var out []ReplicaPlan
	for _, r := range p.Replicas {
		if r.Node == "" {
			out = append(out, r)
		}
	}
	return out
}

// Request is one scheduling problem: place replicas of an app across available nodes.
type Request struct {
	App        string                // app identifier
	Spec       api.AppSpec           // app placement spec with tiers, arch, features, resources, policies
	Nodes      []Node                // available nodes with capacity and policies
	Current    map[int64]string      // current assignment: replica index -> node ID (for stability)
	VolumeHint map[int64][]string    // volume affinity: replica index -> preferred nodes for data locality
	Federated  bool                  // true if placing on behalf of a federation peer (stricter policies)
}

// Schedule computes a placement plan for all replicas.
// Returns a Plan with either a node ID or "" (unschedulable) for each replica.
// Pass 1: Keep replicas on current nodes if still eligible (stability).
// Pass 2: Place remaining replicas using multi-factor scoring, preferring:
//   - Balanced fit (not overloading any single node)
//   - Domain spread (replicas across region|zone|host failure domains)
//   - Region locality (preferred region if specified)
//   - Volume data locality (affinity to volume members)
//   - Host stability (uptime / age)
//   - Headroom (prefer less-utilized nodes)
func Schedule(req Request) Plan {
	nodes := append([]Node(nil), req.Nodes...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	used := map[string][2]int64{}
	count := map[string]int64{}
	domain := map[string]int64{}
	for _, n := range nodes {
		used[n.ID] = [2]int64{n.UsedCPU, n.UsedMem}
		count[n.ID] = n.Workloads
	}
	domains := map[string]bool{}
	for _, n := range nodes {
		if n.Status == "ready" {
			domains[n.DomainKey()] = true
		}
	}
	domainN := int64(len(domains))
	if domainN == 0 {
		domainN = 1
	}
	perApp := map[string]int64{}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	plan := Plan{App: req.App}
	spec := req.Spec

	take := func(n Node) {
		u := used[n.ID]
		used[n.ID] = [2]int64{u[0] + spec.Resources.CPUMilli, u[1] + spec.Resources.MemBytes}
		count[n.ID]++
		perApp[n.ID]++
		domain[n.DomainKey()]++
	}

	// Pass 1: keep replicas whose current host is still valid. Resources the
	// current placement already consumes are part of UsedCPU/UsedMem, so the
	// resource filter is skipped for kept replicas.
	kept := map[int64]bool{}
	for r := int64(0); r < spec.Replicas; r++ {
		cur, ok := req.Current[r]
		if !ok {
			continue
		}
		n, ok := byID[cur]
		if !ok {
			continue
		}
		if stage, why := filter(n, spec, used, perApp, domain, domainN, req.Federated, true); stage == "" {
			perApp[n.ID]++
			domain[n.DomainKey()]++
			kept[r] = true
			plan.Replicas = append(plan.Replicas, ReplicaPlan{Replica: r, Node: n.ID, Kept: true, Reason: "current host still satisfies every constraint"})
		} else {
			_ = why
		}
	}

	for r := int64(0); r < spec.Replicas; r++ {
		if kept[r] {
			continue
		}
		rp := ReplicaPlan{Replica: r}
		type cand struct {
			n     Node
			score float64
		}
		var ranked []cand
		maxDomain := int64(1)
		for _, c := range domain {
			if c > maxDomain {
				maxDomain = c
			}
		}
		for _, n := range nodes {
			stage, why := filter(n, spec, used, perApp, domain, domainN, req.Federated, false)
			if stage != "" {
				rp.Rows = append(rp.Rows, Row{Node: n.ID, Name: n.Name, Stage: stage, Reason: why})
				continue
			}
			s := score(n, spec, used, domain, maxDomain, req.VolumeHint[r])
			rp.Rows = append(rp.Rows, Row{Node: n.ID, Name: n.Name, OK: true, Stage: "score", Reason: "pass", Score: s})
			ranked = append(ranked, cand{n, s})
		}
		sort.SliceStable(ranked, func(i, j int) bool {
			if ranked[i].score != ranked[j].score {
				return ranked[i].score > ranked[j].score
			}
			return ranked[i].n.ID < ranked[j].n.ID
		})
		if len(ranked) == 0 {
			rp.Reason = summarize(rp.Rows)
		} else {
			rp.Node = ranked[0].n.ID
			rp.Reason = fmt.Sprintf("highest score %.4f", ranked[0].score)
			take(ranked[0].n)
		}
		plan.Replicas = append(plan.Replicas, rp)
	}
	sort.Slice(plan.Replicas, func(i, j int) bool { return plan.Replicas[i].Replica < plan.Replicas[j].Replica })
	return plan
}

func filter(n Node, spec api.AppSpec, used map[string][2]int64, perApp, domain map[string]int64, domainN int64, federated, keeping bool) (string, string) {
	if n.Status != "ready" {
		return "status", fmt.Sprintf("%s is %s", n.Name, n.Status)
	}
	if hasRole(n, "edge-only") {
		return "status", n.Name + " is an edge-only host"
	}
	pol := n.Policy
	if len(pol.AllowRuntimes) > 0 && !contains(pol.AllowRuntimes, spec.Runtime) {
		return "policy", fmt.Sprintf("%s policy does not allow runtime %s", n.Name, spec.Runtime)
	}
	if federated && !pol.AllowFederated {
		return "policy", n.Name + " policy does not accept federated work"
	}
	if pol.MaxWorkloads > 0 && !keeping && n.Workloads+perApp[n.ID] >= pol.MaxWorkloads {
		return "policy", fmt.Sprintf("%s workload cap %d reached", n.Name, pol.MaxWorkloads)
	}
	accept := n.Tiers
	if len(pol.AcceptTiers) > 0 {
		accept = intersect(accept, pol.AcceptTiers)
	}
	if !intersects(accept, spec.Placement.Tiers) {
		return "trust", fmt.Sprintf("%s accepts tiers [%s], manifest needs one of [%s]", n.Name, strings.Join(accept, ", "), strings.Join(spec.Placement.Tiers, ", "))
	}
	if len(spec.Placement.Arch) > 0 && !contains(spec.Placement.Arch, n.Arch) {
		return "arch", fmt.Sprintf("%s arch %s not in [%s]", n.Name, n.Arch, strings.Join(spec.Placement.Arch, ", "))
	}
	for _, f := range spec.Placement.Features {
		if !contains(n.Features, f) {
			return "features", fmt.Sprintf("%s lacks feature %s", n.Name, f)
		}
	}
	if !keeping {
		u := used[n.ID]
		if u[0]+spec.Resources.CPUMilli > n.CPUMilli {
			return "resources", fmt.Sprintf("%s cpu exhausted (%dm + %dm > %dm)", n.Name, u[0], spec.Resources.CPUMilli, n.CPUMilli)
		}
		if u[1]+spec.Resources.MemBytes > n.MemBytes {
			return "resources", fmt.Sprintf("%s memory exhausted", n.Name)
		}
		if pol.MaxCPUMilli > 0 && u[0]+spec.Resources.CPUMilli > pol.MaxCPUMilli {
			return "policy", fmt.Sprintf("%s policy cpu cap %dm", n.Name, pol.MaxCPUMilli)
		}
		if pol.MaxMemBytes > 0 && u[1]+spec.Resources.MemBytes > pol.MaxMemBytes {
			return "policy", fmt.Sprintf("%s policy memory cap", n.Name)
		}
	}
	if spec.Placement.AntiAffinity == "hard" && perApp[n.ID] > 0 {
		return "anti-affinity", n.Name + " already hosts a replica (hard anti-affinity)"
	}
	if spec.Placement.Spread == "failure-domain" {
		maxPer := int64(math.Ceil(float64(spec.Replicas) / float64(domainN)))
		if domain[n.DomainKey()] >= maxPer {
			return "failure-domain", fmt.Sprintf("domain %s at spread cap %d", n.DomainKey(), maxPer)
		}
	}
	return "", ""
}

func score(n Node, spec api.AppSpec, used map[string][2]int64, domain map[string]int64, maxDomain int64, volumeNodes []string) float64 {
	u := used[n.ID]
	fit := 0.0
	if free := float64(n.CPUMilli - u[0]); free > 0 {
		fit = 1 - (free-float64(spec.Resources.CPUMilli))/free
	}
	if free := float64(n.MemBytes - u[1]); free > 0 {
		fit = (fit + (1 - (free-float64(spec.Resources.MemBytes))/free)) / 2
	}
	if fit < 0 {
		fit = 0
	}
	spread := 1 - float64(domain[n.DomainKey()])/float64(maxDomain)
	locality := 0.5
	if spec.Placement.PreferRegion != "" {
		if n.Region == spec.Placement.PreferRegion {
			locality = 1
		} else {
			locality = 0.25
		}
	}
	affinity := 0.5
	if contains(volumeNodes, n.ID) {
		affinity = 1
	}
	stability := math.Min(1, math.Log1p(float64(n.UptimeMs)/1000)/math.Log1p(86400*30))
	headroom := 0.5
	if n.CPUMilli > 0 {
		headroom = 1 - float64(u[0])/float64(n.CPUMilli)
	}
	return 0.25*fit + 0.25*spread + 0.2*locality + 0.15*affinity + 0.1*stability + 0.05*headroom
}

func summarize(rows []Row) string {
	if len(rows) == 0 {
		return "no hosts"
	}
	seen := map[string]bool{}
	var parts []string
	for _, r := range rows {
		if !r.OK && !seen[r.Reason] {
			seen[r.Reason] = true
			parts = append(parts, r.Reason)
		}
	}
	return strings.Join(parts, "; ")
}

// VolumeRequest places replicas of one volume.
type VolumeRequest struct {
	Replicas  int64
	SizeBytes int64
	Primary   string   // node running the workload; always a member when eligible
	Current   []string // current members, kept when still eligible
	Tiers     []string
	Nodes     []Node
}

// VolumePlan is the chosen member set with explanations.
type VolumePlan struct {
	Members []string `json:"members"`
	Rows    []Row    `json:"rows"`
	Short   int64    `json:"short"` // replicas that could not be placed
}

// PlaceVolume picks storage replicas: primary first, then current members,
// then new hosts preferring unused failure domains. Two replicas never share
// a failure domain while an unused domain is still available.
func PlaceVolume(req VolumeRequest) VolumePlan {
	nodes := append([]Node(nil), req.Nodes...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	byID := map[string]Node{}
	var plan VolumePlan
	eligible := map[string]bool{}
	for _, n := range nodes {
		byID[n.ID] = n
		why := ""
		switch {
		case n.Status != "ready":
			why = n.Name + " is " + n.Status
		case len(req.Tiers) > 0 && !intersects(n.Tiers, req.Tiers):
			why = n.Name + " tier mismatch"
		case n.DiskFree >= 0 && n.DiskFree < req.SizeBytes:
			why = n.Name + " has insufficient free disk"
		case n.Policy.StorageQuotaBytes > 0 && n.Policy.StorageQuotaBytes < req.SizeBytes:
			why = n.Name + " storage quota below volume size"
		}
		if why == "" {
			eligible[n.ID] = true
			plan.Rows = append(plan.Rows, Row{Node: n.ID, Name: n.Name, OK: true, Stage: "storage", Reason: "eligible"})
		} else {
			plan.Rows = append(plan.Rows, Row{Node: n.ID, Name: n.Name, Stage: "storage", Reason: why})
		}
	}
	usedDomain := map[string]bool{}
	chosen := map[string]bool{}
	add := func(id string) {
		plan.Members = append(plan.Members, id)
		chosen[id] = true
		usedDomain[byID[id].DomainKey()] = true
	}
	if req.Primary != "" && eligible[req.Primary] {
		add(req.Primary)
	}
	for _, id := range req.Current {
		if int64(len(plan.Members)) >= req.Replicas {
			break
		}
		if eligible[id] && !chosen[id] && !usedDomain[byID[id].DomainKey()] {
			add(id)
		}
	}
	for pass := 0; pass < 2 && int64(len(plan.Members)) < req.Replicas; pass++ {
		for _, n := range nodes {
			if int64(len(plan.Members)) >= req.Replicas {
				break
			}
			if !eligible[n.ID] || chosen[n.ID] {
				continue
			}
			if pass == 0 && usedDomain[n.DomainKey()] {
				continue // first pass: new failure domains only
			}
			add(n.ID)
		}
	}
	plan.Short = req.Replicas - int64(len(plan.Members))
	if plan.Short < 0 {
		plan.Short = 0
	}
	return plan
}

func hasRole(n Node, role string) bool { return contains(n.Roles, role) }

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

func intersect(a, b []string) []string {
	var out []string
	for _, x := range a {
		if contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}
