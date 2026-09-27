package control

import (
	"fmt"
	"sort"
	"time"

	"decentralized.host/pkg/scheduler"
)

// Freshness describes the staleness of a node's last observation.
type Freshness string

const (
	FreshmentFresh       Freshness = "fresh"       // < 60s
	FreshnessStale       Freshness = "stale"       // 60s - 5m
	FreshnessExpired     Freshness = "expired"     // > 5m
	FreshnessUnreachable Freshness = "unreachable" // never heard from
)

// CapabilityTruth tracks declared vs observed hardware capabilities.
type CapabilityTruth struct {
	Name      string   `json:"name"`      // capability name (e.g., "gpu-nvidia", "sse4")
	Declared  bool     `json:"declared"`  // declared at enrollment
	Observed  *bool    `json:"observed"`  // observed in facts, nil if unknown
	Probed    string   `json:"probed"`    // probe result if measured
	Evidence  string   `json:"evidence"`  // source (enrollment | facts | probes)
	Mismatch  bool     `json:"mismatch"`  // declared != observed
}

// ResourceCapacity models resource accounts per constraint hierarchy.
type ResourceCapacity struct {
	CPUMilli      int64  `json:"cpuMilli"`      // total CPU in millicores
	MemBytes      int64  `json:"memBytes"`      // total memory in bytes
	DiskBytes     int64  `json:"diskBytes"`     // total disk free in bytes
	OwnerReserve  int64  `json:"ownerReserve"`  // reserved for cluster ownership
	Reserved      int64  `json:"reserved"`      // reserved for future allocation
	Allocated     int64  `json:"allocated"`     // actively allocated to workloads
	Available     int64  `json:"available"`     // available = total - owner - reserved - allocated
}

// SchedulerEligibility tracks whether a node can accept new placements.
type SchedulerEligibility struct {
	Eligible       bool     `json:"eligible"`
	RejectionCode  string   `json:"rejectionCode"`  // if not eligible, why
	RejectionReason string  `json:"rejectionReason"`
}

// PlacementRejectionCode enumerates reasons a node cannot accept placements.
type PlacementRejectionCode string

const (
	RejectNodeNotReady       PlacementRejectionCode = "NODE_NOT_READY"
	RejectNodeCordoned       PlacementRejectionCode = "NODE_CORDONED"
	RejectNodeDraining       PlacementRejectionCode = "NODE_DRAINING"
	RejectNodeStale          PlacementRejectionCode = "NODE_STALE"
	RejectNodeExpired        PlacementRejectionCode = "NODE_EXPIRED"
	RejectNodeUnreachable    PlacementRejectionCode = "NODE_UNREACHABLE"
	RejectInsufficientCPU    PlacementRejectionCode = "INSUFFICIENT_CPU"
	RejectInsufficientMem    PlacementRejectionCode = "INSUFFICIENT_MEMORY"
	RejectInsufficientDisk   PlacementRejectionCode = "INSUFFICIENT_DISK"
	RejectCapabilityMissing  PlacementRejectionCode = "CAPABILITY_MISSING"
	RejectCapabilityMismatch PlacementRejectionCode = "CAPABILITY_MISMATCH"
	RejectPolicyDenied       PlacementRejectionCode = "POLICY_DENIED"
	RejectTierMismatch       PlacementRejectionCode = "TIER_MISMATCH"
)

// FleetNode is a node as presented in fleet inventory API.
// It integrates identity, lifecycle, capability, resource, and observational state.
type FleetNode struct {
	// Identity and lifecycle
	ID         string `json:"id"`         // canonical node identity
	Name       string `json:"name"`       // human-readable name
	Status     string `json:"status"`     // pending | ready | draining | revoked
	Health     string `json:"health"`     // live | lost | unknown (deprecated; use Freshness)
	Cordoned   bool   `json:"cordoned"`   // operator-initiated placement hold

	// Failure domains
	Region  string `json:"region"`  // geographic region
	Zone    string `json:"zone"`    // availability zone / failure domain
	Host    string `json:"host"`    // physical host name

	// Declared capabilities (from enrollment)
	Tiers       []string `json:"tiers"`
	Arch        string   `json:"arch"`
	Features    []string `json:"features"`
	Roles       []string `json:"roles"`

	// Observed hardware (from facts)
	ObservedArch     string         `json:"observedArch"`
	ObservedMemBytes int64          `json:"observedMemBytes"`
	ObservedCPUs     int64          `json:"observedCpus"`
	ObservedDiskFree int64          `json:"observedDiskFree"`
	ObservedUptime   int64          `json:"observedUptime"`

	// Resource capacity model
	CapacityTotal      int64 `json:"capacityTotal"`      // CPUMilli: enrollment-declared
	CapacityReserved   int64 `json:"capacityReserved"`   // CPUMilli: system/owner reserve
	CapacityAllocated  int64 `json:"capacityAllocated"`  // CPUMilli: active workload allocation
	CapacityAvailable  int64 `json:"capacityAvailable"`  // CPUMilli: available for scheduling
	MemCapacityTotal   int64 `json:"memCapacityTotal"`   // MemBytes: enrollment-declared
	MemCapacityReserved int64 `json:"memCapacityReserved"` // MemBytes: system/owner reserve
	MemCapacityAllocated int64 `json:"memCapacityAllocated"` // MemBytes: active workload allocation
	MemCapacityAvailable int64 `json:"memCapacityAvailable"` // MemBytes: available for scheduling

	// Observations
	Freshness       Freshness `json:"freshness"`       // FRESH | STALE | EXPIRED | UNREACHABLE
	LastObsAt       int64     `json:"lastObsAt"`       // control-plane receive time of last obs
	ObsSeq          int64     `json:"obsSeq"`          // observation sequence number
	Workloads       int64     `json:"workloads"`       // count of running workloads

	// Lifecycle timestamps
	JoinedAt      int64 `json:"joinedAt"`
	ApprovedAt    int64 `json:"approvedAt"`
	RevokedAt     int64 `json:"revokedAt"`

	// Capability truth
	Capabilities []CapabilityTruth `json:"capabilities"`

	// Scheduler eligibility
	SchedulerEligible SchedulerEligibility `json:"schedulerEligible"`
}

// FleetSummary aggregates capacity and health across the fleet.
type FleetSummary struct {
	Timestamp        int64                  `json:"timestamp"`
	TotalNodes       int64                  `json:"totalNodes"`       // all nodes in cluster
	ApprovedNodes    int64                  `json:"approvedNodes"`    // approved nodes (lifecycle eligible)
	ReadyNodes       int64                  `json:"readyNodes"`       // ready and fresh
	DrainingNodes    int64                  `json:"drainingNodes"`    // in graceful shutdown
	RevokedNodes     int64                  `json:"revokedNodes"`     // permanently removed
	StaleNodes       int64                  `json:"staleNodes"`       // no recent observation
	CordonedNodes    int64                  `json:"cordonedNodes"`    // operator hold
	EligibleNodes    int64                  `json:"eligibleNodes"`    // ready for placement

	TotalCPU         int64                  `json:"totalCpuMilli"`
	ReservedCPU      int64                  `json:"reservedCpuMilli"` // owner/system reserve
	AllocatedCPU     int64                  `json:"allocatedCpuMilli"`
	AvailCPU         int64                  `json:"availCpuMilli"`

	TotalMem         int64                  `json:"totalMemBytes"`
	ReservedMem      int64                  `json:"reservedMemBytes"`
	AllocatedMem     int64                  `json:"allocatedMemBytes"`
	AvailMem         int64                  `json:"availMemBytes"`

	TotalDisk        int64                  `json:"totalDiskBytes"`
	TotalWorkload    int64                  `json:"totalWorkloads"`

	ByTier           map[string]*TierSummary `json:"byTier"`
	ByZone           map[string]*ZoneSummary `json:"byZone"`
}

// TierSummary aggregates one tier.
type TierSummary struct {
	Nodes        int64 `json:"nodes"`
	CPUMilli     int64 `json:"cpuMilli"`
	MemBytes     int64 `json:"memBytes"`
	UsedCPU      int64 `json:"usedCpuMilli"`
	UsedMem      int64 `json:"usedMemBytes"`
	AvailCPU     int64 `json:"availCpuMilli"`
	AvailMem     int64 `json:"availMemBytes"`
	Workloads    int64 `json:"workloads"`
	ReadyNodes   int64 `json:"readyNodes"`
}

// ZoneSummary aggregates one failure domain (zone).
type ZoneSummary struct {
	Region       string `json:"region"`
	Zone         string `json:"zone"`
	Nodes        int64  `json:"nodes"`
	CPUMilli     int64  `json:"cpuMilli"`
	MemBytes     int64  `json:"memBytes"`
	UsedCPU      int64  `json:"usedCpuMilli"`
	UsedMem      int64  `json:"usedMemBytes"`
	AvailCPU     int64  `json:"availCpuMilli"`
	AvailMem     int64  `json:"availMemBytes"`
	Workloads    int64  `json:"workloads"`
	ReadyNodes   int64  `json:"readyNodes"`
}

// computeFreshness determines observation staleness.
func computeFreshness(lastObsAt int64, now int64) Freshness {
	if lastObsAt == 0 {
		return FreshnessUnreachable
	}
	elapsedMs := now - lastObsAt
	switch {
	case elapsedMs < 60*1000: // < 60s
		return FreshmentFresh
	case elapsedMs < 5*60*1000: // < 5m
		return FreshnessStale
	case elapsedMs < 1*60*60*1000: // < 1h
		return FreshnessExpired
	default:
		return FreshnessUnreachable
	}
}

// determineSchedulerEligibility checks if a node can accept new workloads.
func (fn *FleetNode) determineSchedulerEligibility() {
	fn.SchedulerEligible.Eligible = true
	fn.SchedulerEligible.RejectionCode = ""
	fn.SchedulerEligible.RejectionReason = ""

	// Lifecycle check
	if fn.Status != "ready" {
		fn.SchedulerEligible.Eligible = false
		if fn.Status == "draining" {
			fn.SchedulerEligible.RejectionCode = string(RejectNodeDraining)
			fn.SchedulerEligible.RejectionReason = "node is draining; no new placements"
		} else if fn.Status == "revoked" {
			fn.SchedulerEligible.Eligible = false // revoked nodes never eligible
		} else {
			fn.SchedulerEligible.RejectionCode = string(RejectNodeNotReady)
			fn.SchedulerEligible.RejectionReason = fmt.Sprintf("node status is %s", fn.Status)
		}
		return
	}

	// Cordoned check
	if fn.Cordoned {
		fn.SchedulerEligible.Eligible = false
		fn.SchedulerEligible.RejectionCode = string(RejectNodeCordoned)
		fn.SchedulerEligible.RejectionReason = "node is cordoned by operator"
		return
	}

	// Freshness check
	switch fn.Freshness {
	case FreshnessUnreachable:
		fn.SchedulerEligible.Eligible = false
		fn.SchedulerEligible.RejectionCode = string(RejectNodeUnreachable)
		fn.SchedulerEligible.RejectionReason = "no recent observation from node"
	case FreshnessExpired:
		fn.SchedulerEligible.Eligible = false
		fn.SchedulerEligible.RejectionCode = string(RejectNodeExpired)
		fn.SchedulerEligible.RejectionReason = "observation expired (> 1h)"
	}
}

// FleetInventory returns all approved nodes with comprehensive state including observations, capabilities, and eligibility.
// Unapproved nodes (pending status with no ApprovedAt) are excluded as they haven't completed identity verification.
func (s *State) FleetInventory() []*FleetNode {
	now := time.Now().UnixMilli()
	var nodes []*FleetNode

	for _, n := range s.Nodes {
		// Exclude unapproved pending nodes from fleet inventory
		if n.Status == "pending" && n.ApprovedAt == 0 {
			continue
		}

		fn := &FleetNode{
			ID:         n.ID,
			Name:       n.Name,
			Status:     n.Status,
			Health:     n.Health,
			Region:     n.Enroll.Region,
			Zone:       n.Enroll.Zone,
			Host:       n.Enroll.Host,
			Tiers:      append([]string(nil), n.Enroll.Tiers...),
			Arch:       n.Enroll.Arch,
			Features:   append([]string(nil), n.Enroll.Features...),
			Roles:      append([]string(nil), n.Roles...),

			CapacityTotal:     n.Enroll.CPUMilli,
			MemCapacityTotal:  n.Enroll.MemBytes,

			JoinedAt:   n.JoinedAt,
			ApprovedAt: n.ApprovedAt,
			RevokedAt:  n.RevokedAt,
			LastObsAt:  n.LastObsAt,
			ObsSeq:     n.ObserveSeq,
		}

		// Compute available capacity: total - reserved - allocated
		fn.CapacityAvailable = fn.CapacityTotal - fn.CapacityReserved - fn.CapacityAllocated
		fn.MemCapacityAvailable = fn.MemCapacityTotal - fn.MemCapacityReserved - fn.MemCapacityAllocated

		// Compute freshness
		fn.Freshness = computeFreshness(n.LastObsAt, now)

		// Extract observed hardware facts
		if n.Obs != nil {
			fn.ObservedArch = n.Obs.Facts.Arch
			fn.ObservedMemBytes = n.Obs.Facts.MemBytes
			fn.ObservedCPUs = n.Obs.Facts.CPUs
			if n.Obs.Facts.DataFS != nil {
				fn.ObservedDiskFree = n.Obs.Facts.DataFS.FreeBytes
			}
			fn.ObservedUptime = n.Obs.Facts.UptimeSec * 1000 // convert to ms

			// Count running workloads
			for _, w := range n.Obs.Workloads {
				if w.Observed == "running" {
					fn.Workloads++
				}
			}
		}

		// Determine scheduler eligibility based on comprehensive state
		fn.determineSchedulerEligibility()

		nodes = append(nodes, fn)
	}

	// Sort by region, zone, name
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Region != nodes[j].Region {
			return nodes[i].Region < nodes[j].Region
		}
		if nodes[i].Zone != nodes[j].Zone {
			return nodes[i].Zone < nodes[j].Zone
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes
}

// ComputeFleetSummary calculates aggregate fleet metrics with proper resource accounting.
func (s *State) ComputeFleetSummary() *FleetSummary {
	summary := &FleetSummary{
		Timestamp: time.Now().UnixMilli(),
		ByTier:    make(map[string]*TierSummary),
		ByZone:    make(map[string]*ZoneSummary),
	}

	for _, fn := range s.FleetInventory() {
		summary.TotalNodes++

		// Lifecycle and freshness-based status counting
		if fn.ApprovedAt > 0 {
			summary.ApprovedNodes++
		}
		if fn.Status == "ready" && fn.Freshness == FreshmentFresh {
			summary.ReadyNodes++
		}
		if fn.Status == "draining" {
			summary.DrainingNodes++
		}
		if fn.Status == "revoked" {
			summary.RevokedNodes++
		}
		if fn.Freshness == FreshnessStale || fn.Freshness == FreshnessExpired {
			summary.StaleNodes++
		}
		if fn.Cordoned {
			summary.CordonedNodes++
		}
		if fn.SchedulerEligible.Eligible {
			summary.EligibleNodes++
		}

		// CPU accounting: total = reserved + allocated + available
		summary.TotalCPU += fn.CapacityTotal
		summary.ReservedCPU += fn.CapacityReserved
		summary.AllocatedCPU += fn.CapacityAllocated
		summary.AvailCPU += fn.CapacityAvailable

		// Memory accounting
		summary.TotalMem += fn.MemCapacityTotal
		summary.ReservedMem += fn.MemCapacityReserved
		summary.AllocatedMem += fn.MemCapacityAllocated
		summary.AvailMem += fn.MemCapacityAvailable

		// Disk and workloads
		summary.TotalDisk += fn.ObservedDiskFree
		summary.TotalWorkload += fn.Workloads

		// By Tier aggregation
		for _, tier := range fn.Tiers {
			if _, ok := summary.ByTier[tier]; !ok {
				summary.ByTier[tier] = &TierSummary{}
			}
			ts := summary.ByTier[tier]
			ts.Nodes++
			ts.CPUMilli += fn.CapacityTotal
			ts.MemBytes += fn.MemCapacityTotal
			ts.UsedCPU += fn.CapacityAllocated
			ts.UsedMem += fn.MemCapacityAllocated
			ts.Workloads += fn.Workloads
			if fn.Status == "ready" && fn.Freshness == FreshmentFresh {
				ts.ReadyNodes++
			}
			ts.AvailCPU = ts.CPUMilli - ts.UsedCPU
			ts.AvailMem = ts.MemBytes - ts.UsedMem
		}

		// By Zone aggregation
		zoneKey := fn.Region + "|" + fn.Zone
		if _, ok := summary.ByZone[zoneKey]; !ok {
			summary.ByZone[zoneKey] = &ZoneSummary{
				Region: fn.Region,
				Zone:   fn.Zone,
			}
		}
		zs := summary.ByZone[zoneKey]
		zs.Nodes++
		zs.CPUMilli += fn.CapacityTotal
		zs.MemBytes += fn.MemCapacityTotal
		zs.UsedCPU += fn.CapacityAllocated
		zs.UsedMem += fn.MemCapacityAllocated
		zs.Workloads += fn.Workloads
		if fn.Status == "ready" && fn.Freshness == FreshmentFresh {
			zs.ReadyNodes++
		}
		zs.AvailCPU = zs.CPUMilli - zs.UsedCPU
		zs.AvailMem = zs.MemBytes - zs.UsedMem
	}

	return summary
}

// NodeToSchedulerNode converts control state to scheduler node.
func (n *Node) ToSchedulerNode() scheduler.Node {
	sn := scheduler.Node{
		ID:       n.ID,
		Name:     n.Name,
		Status:   n.Status,
		Tiers:    append([]string(nil), n.Enroll.Tiers...),
		Region:   n.Enroll.Region,
		Zone:     n.Enroll.Zone,
		Host:     n.Enroll.Host,
		Arch:     n.Enroll.Arch,
		Features: append([]string(nil), n.Enroll.Features...),
		CPUMilli: n.Enroll.CPUMilli,
		MemBytes: n.Enroll.MemBytes,
		Policy:   n.Enroll.Policy,
		UptimeMs: 0, // will be computed from FirstSeen
	}

	if n.Obs != nil {
		for _, w := range n.Obs.Workloads {
			if w.Observed == "running" {
				// estimate usage from assignment (simplified)
				sn.UsedCPU += 100  // placeholder
				sn.UsedMem += 1000 // placeholder
			}
		}
		sn.Workloads = int64(len(n.Obs.Workloads))
		if n.Obs.Storage != nil {
			sn.DiskFree = n.Obs.Storage.FreeBytes
		}
	}

	return sn
}
