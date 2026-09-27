package control

import (
	"testing"

	"decentralized.host/pkg/api"
)

func TestFleetInventory_Empty(t *testing.T) {
	s := &State{Nodes: make(map[string]*Node)}
	inv := s.FleetInventory()
	if len(inv) != 0 {
		t.Fatalf("expected empty inventory, got %d nodes", len(inv))
	}
}

func TestFleetInventory_SingleNode(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:        "node-001",
				Name:      "host-a",
				Status:    "ready",
				Health:    "live",
				JoinedAt:  1000,
				ApprovedAt: 1100,
				Enroll: api.Enroll{
					Region:   "us-west",
					Zone:     "us-west-2a",
					Host:     "host-a",
					Tiers:    []string{"compute", "standard"},
					Arch:     "amd64",
					CPUMilli: 8000,
					MemBytes: 16 * 1024 * 1024 * 1024,
					DiskBytes: 100 * 1024 * 1024 * 1024,
					Features: []string{"sse4", "avx2"},
				},
			},
		},
	}
	inv := s.FleetInventory()
	if len(inv) != 1 {
		t.Fatalf("expected 1 node, got %d", len(inv))
	}
	n := inv[0]
	if n.ID != "node-001" || n.Region != "us-west" || n.Status != "ready" {
		t.Fatalf("node data mismatch: %+v", n)
	}
}

func TestFleetInventory_ExcludesUnapprovedNodes(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:     "node-001",
				Name:   "host-a",
				Status: "pending",
				// no ApprovedAt
				JoinedAt: 1000,
				Enroll:   api.Enroll{Region: "us-west", Zone: "us-west-2a", Host: "host-a"},
			},
			"node-002": {
				ID:        "node-002",
				Name:      "host-b",
				Status:    "ready",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-west", Zone: "us-west-2a", Host: "host-b"},
			},
		},
	}
	inv := s.FleetInventory()
	if len(inv) != 1 {
		t.Fatalf("expected 1 approved node, got %d", len(inv))
	}
	if inv[0].ID != "node-002" {
		t.Fatalf("wrong node returned")
	}
}

func TestFleetInventory_SortedByRegionZoneName(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-003": {
				ID:        "node-003",
				Name:      "host-c",
				Status:    "ready",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-east", Zone: "us-east-1a", Host: "host-c"},
			},
			"node-001": {
				ID:        "node-001",
				Name:      "host-a",
				Status:    "ready",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-west", Zone: "us-west-2a", Host: "host-a"},
			},
			"node-002": {
				ID:        "node-002",
				Name:      "host-b",
				Status:    "ready",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-west", Zone: "us-west-2a", Host: "host-b"},
			},
		},
	}
	inv := s.FleetInventory()
	if inv[0].ID != "node-003" || inv[1].ID != "node-001" || inv[2].ID != "node-002" {
		t.Fatalf("wrong sort order: %+v", inv)
	}
}

func TestComputeFleetSummary_Empty(t *testing.T) {
	s := &State{Nodes: make(map[string]*Node)}
	summary := s.ComputeFleetSummary()
	if summary.TotalNodes != 0 {
		t.Fatalf("expected 0 nodes, got %d", summary.TotalNodes)
	}
}

func TestComputeFleetSummary_Aggregation(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:        "node-001",
				Name:      "host-a",
				Status:    "ready",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll: api.Enroll{
					Region:   "us-west",
					Zone:     "us-west-2a",
					Host:     "host-a",
					Tiers:    []string{"compute"},
					CPUMilli: 8000,
					MemBytes: 16 * 1024 * 1024 * 1024,
				},
			},
			"node-002": {
				ID:        "node-002",
				Name:      "host-b",
				Status:    "ready",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll: api.Enroll{
					Region:   "us-west",
					Zone:     "us-west-2b",
					Host:     "host-b",
					Tiers:    []string{"storage"},
					CPUMilli: 4000,
					MemBytes: 8 * 1024 * 1024 * 1024,
				},
			},
		},
	}
	summary := s.ComputeFleetSummary()
	if summary.TotalNodes != 2 {
		t.Fatalf("expected 2 nodes, got %d", summary.TotalNodes)
	}
	if summary.TotalCPU != 12000 {
		t.Fatalf("expected 12000 CPU, got %d", summary.TotalCPU)
	}
	if summary.ReadyNodes != 2 {
		t.Fatalf("expected 2 ready nodes, got %d", summary.ReadyNodes)
	}
}

func TestComputeFleetSummary_ByTier(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:        "node-001",
				Status:    "ready",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll: api.Enroll{
					Region:   "us-west",
					Zone:     "us-west-2a",
					Tiers:    []string{"compute", "standard"},
					CPUMilli: 8000,
					MemBytes: 16 * 1024 * 1024 * 1024,
				},
			},
		},
	}
	summary := s.ComputeFleetSummary()
	if ts, ok := summary.ByTier["compute"]; !ok || ts.Nodes != 1 {
		t.Fatalf("compute tier not aggregated")
	}
	if ts, ok := summary.ByTier["standard"]; !ok || ts.Nodes != 1 {
		t.Fatalf("standard tier not aggregated")
	}
}

func TestComputeFleetSummary_ByZone(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:        "node-001",
				Status:    "ready",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll: api.Enroll{
					Region:   "us-west",
					Zone:     "us-west-2a",
					CPUMilli: 8000,
				},
			},
		},
	}
	summary := s.ComputeFleetSummary()
	zoneKey := "us-west|us-west-2a"
	if zs, ok := summary.ByZone[zoneKey]; !ok || zs.Nodes != 1 {
		t.Fatalf("zone not aggregated")
	}
}

func TestComputeFleetSummary_HealthStatus(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:        "node-001",
				Status:    "ready",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-west", Zone: "us-west-2a"},
			},
			"node-002": {
				ID:        "node-002",
				Status:    "ready",
				Health:    "lost",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-west", Zone: "us-west-2a"},
			},
			"node-003": {
				ID:        "node-003",
				Status:    "draining",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll:    api.Enroll{Region: "us-west", Zone: "us-west-2a"},
			},
		},
	}
	summary := s.ComputeFleetSummary()
	if summary.TotalNodes != 3 {
		t.Fatalf("expected 3 total nodes")
	}
	if summary.ReadyNodes != 1 {
		t.Fatalf("expected 1 ready node, got %d", summary.ReadyNodes)
	}
	if summary.OfflineNodes != 1 {
		t.Fatalf("expected 1 offline node, got %d", summary.OfflineNodes)
	}
	if summary.DrainingNodes != 1 {
		t.Fatalf("expected 1 draining node, got %d", summary.DrainingNodes)
	}
}

func TestComputeFleetSummary_AvailableCapacity(t *testing.T) {
	s := &State{
		Nodes: map[string]*Node{
			"node-001": {
				ID:        "node-001",
				Status:    "ready",
				Health:    "live",
				ApprovedAt: 1100,
				Enroll: api.Enroll{
					CPUMilli: 8000,
					MemBytes: 16 * 1024 * 1024 * 1024,
				},
				Obs: &api.Observation{
					Workloads: []api.WorkloadObs{
						{Observed: "running"},
						{Observed: "running"},
					},
				},
			},
		},
	}
	summary := s.ComputeFleetSummary()
	if summary.TotalCPU == 0 {
		t.Fatalf("TotalCPU should be > 0")
	}
	// AvailCPU == TotalCPU initially (no usage tracked from observations yet)
	if summary.AvailCPU != summary.TotalCPU {
		t.Fatalf("AvailCPU should equal TotalCPU before usage tracking implemented")
	}
}
