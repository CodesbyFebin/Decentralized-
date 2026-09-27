package control

import (
	"sort"

	"decentralized.host/pkg/scheduler"
)

// FleetNode is a node as presented in fleet inventory API.
type FleetNode struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Status       string   `json:"status"`       // pending | ready | draining | revoked
	Health       string   `json:"health"`       // live | lost | unknown
	Region       string   `json:"region"`
	Zone         string   `json:"zone"`
	Host         string   `json:"host"`
	Tiers        []string `json:"tiers"`
	Arch         string   `json:"arch"`
	CPUMilli     int64    `json:"cpuMilli"`
	MemBytes     int64    `json:"memBytes"`
	DiskBytes    int64    `json:"diskBytes"`
	UsedCPU      int64    `json:"usedCpuMilli"`
	UsedMem      int64    `json:"usedMemBytes"`
	Workloads    int64    `json:"workloads"`
	UptimeMs     int64    `json:"uptimeMs"`
	JoinedAt     int64    `json:"joinedAt"`
	LastObsAt    int64    `json:"lastObsAt"`
	ObsSeq       int64    `json:"obsSeq"`
	Features     []string `json:"features"`
	Roles        []string `json:"roles"`
}

// FleetSummary aggregates capacity and health across the fleet.
type FleetSummary struct {
	TotalNodes    int64                  `json:"totalNodes"`
	ReadyNodes    int64                  `json:"readyNodes"`
	DrainingNodes int64                  `json:"drainingNodes"`
	OfflineNodes  int64                  `json:"offlineNodes"`
	TotalCPU      int64                  `json:"totalCpuMilli"`
	AvailCPU      int64                  `json:"availCpuMilli"`
	UsedCPU       int64                  `json:"usedCpuMilli"`
	TotalMem      int64                  `json:"totalMemBytes"`
	AvailMem      int64                  `json:"availMemBytes"`
	UsedMem       int64                  `json:"usedMemBytes"`
	TotalDisk     int64                  `json:"totalDiskBytes"`
	TotalWorkload int64                  `json:"totalWorkloads"`
	ByTier        map[string]*TierSummary `json:"byTier"`
	ByZone        map[string]*ZoneSummary `json:"byZone"`
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

// FleetInventory returns all nodes with capacity and health status.
func (s *State) FleetInventory() []*FleetNode {
	var nodes []*FleetNode
	for _, n := range s.Nodes {
		if n.Status == "pending" && n.ApprovedAt == 0 {
			continue // skip unapproved nodes
		}
		fn := &FleetNode{
			ID:        n.ID,
			Name:      n.Name,
			Status:    n.Status,
			Health:    n.Health,
			Region:    n.Enroll.Region,
			Zone:      n.Enroll.Zone,
			Host:      n.Enroll.Host,
			Tiers:     append([]string(nil), n.Enroll.Tiers...),
			Arch:      n.Enroll.Arch,
			CPUMilli:  n.Enroll.CPUMilli,
			MemBytes:  n.Enroll.MemBytes,
			DiskBytes: n.Enroll.DiskBytes,
			JoinedAt:  n.JoinedAt,
			LastObsAt: n.LastObsAt,
			ObsSeq:    n.ObserveSeq,
			Features:  append([]string(nil), n.Enroll.Features...),
			Roles:     append([]string(nil), n.Roles...),
		}
		if n.Obs != nil {
			for _, w := range n.Obs.Workloads {
				if w.Observed == "running" {
					fn.Workloads++
				}
			}
		}
		nodes = append(nodes, fn)
	}
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

// ComputeFleetSummary calculates aggregate fleet metrics.
func (s *State) ComputeFleetSummary() *FleetSummary {
	summary := &FleetSummary{
		ByTier: make(map[string]*TierSummary),
		ByZone: make(map[string]*ZoneSummary),
	}

	for _, fn := range s.FleetInventory() {
		summary.TotalNodes++

		if fn.Status == "ready" && fn.Health == "live" {
			summary.ReadyNodes++
		} else if fn.Status == "draining" {
			summary.DrainingNodes++
		} else if fn.Health == "lost" {
			summary.OfflineNodes++
		}

		summary.TotalCPU += fn.CPUMilli
		summary.UsedCPU += fn.UsedCPU
		summary.TotalMem += fn.MemBytes
		summary.UsedMem += fn.UsedMem
		summary.TotalDisk += fn.DiskBytes
		summary.TotalWorkload += fn.Workloads

		// By Tier
		for _, tier := range fn.Tiers {
			if _, ok := summary.ByTier[tier]; !ok {
				summary.ByTier[tier] = &TierSummary{}
			}
			ts := summary.ByTier[tier]
			ts.Nodes++
			ts.CPUMilli += fn.CPUMilli
			ts.MemBytes += fn.MemBytes
			ts.UsedCPU += fn.UsedCPU
			ts.UsedMem += fn.UsedMem
			ts.Workloads += fn.Workloads
			if fn.Status == "ready" && fn.Health == "live" {
				ts.ReadyNodes++
			}
		}

		// By Zone
		zoneKey := fn.Region + "|" + fn.Zone
		if _, ok := summary.ByZone[zoneKey]; !ok {
			summary.ByZone[zoneKey] = &ZoneSummary{
				Region: fn.Region,
				Zone:   fn.Zone,
			}
		}
		zs := summary.ByZone[zoneKey]
		zs.Nodes++
		zs.CPUMilli += fn.CPUMilli
		zs.MemBytes += fn.MemBytes
		zs.UsedCPU += fn.UsedCPU
		zs.UsedMem += fn.UsedMem
		zs.Workloads += fn.Workloads
		if fn.Status == "ready" && fn.Health == "live" {
			zs.ReadyNodes++
		}
	}

	// Compute available
	summary.AvailCPU = summary.TotalCPU - summary.UsedCPU
	summary.AvailMem = summary.TotalMem - summary.UsedMem

	for _, ts := range summary.ByTier {
		ts.AvailCPU = ts.CPUMilli - ts.UsedCPU
		ts.AvailMem = ts.MemBytes - ts.UsedMem
	}

	for _, zs := range summary.ByZone {
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
