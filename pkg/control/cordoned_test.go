package control

import (
	"testing"
	"time"

	"decentralized.host/pkg/api"
)

// TestCordonedNodePreventsPlacement verifies cordoned nodes reject new placements.
func TestCordonedNodePreventsPlacement(t *testing.T) {
	s := newState()

	// Create a ready node
	now := time.Now().UnixMilli()
	n := &Node{
		ID:   "node-1",
		Name: "node-1",
		Status: "ready",
		Enroll: api.Enroll{
			CPUMilli:      8000,
			MemBytes:      16000000000,
			Region:        "us-west",
			Zone:          "us-west-1a",
			Host:          "host-1",
			Tiers:         []string{"standard"},
			Arch:          "x86_64",
			Features:      []string{},
		},
		FailureDomain: "us-west-1a",
		ApprovedAt: now - 10000,
		LastObsAt:  now - 1000, // Fresh observation
		Obs: &api.Observation{
			Facts: api.Facts{
				Arch:      "x86_64",
				MemBytes:  16000000000,
				CPUs:      8,
			},
			Workloads: []api.WorkloadObs{},
		},
	}
	s.Nodes["node-1"] = n

	// Verify node is initially ready and eligible
	fleet := s.FleetInventory()
	if len(fleet) != 1 {
		t.Errorf("expected 1 node, got %d", len(fleet))
	}
	fn := fleet[0]
	if fn.Status != "ready" {
		t.Errorf("expected status=ready, got %s", fn.Status)
	}
	if fn.Cordoned {
		t.Error("expected Cordoned=false initially")
	}
	if !fn.SchedulerEligible.Eligible {
		t.Errorf("expected eligible, got rejection: %s", fn.SchedulerEligible.RejectionReason)
	}

	// Cordon the node by setting Cordoned flag
	n.Cordoned = true

	// Verify node is now cordoned
	fleet = s.FleetInventory()
	fn = fleet[0]
	if !fn.Cordoned {
		t.Error("expected Cordoned=true after cordon")
	}
	if fn.Status != "ready" {
		t.Errorf("expected status=ready after cordon, got %s (cordon should not change status)", fn.Status)
	}
	if fn.SchedulerEligible.Eligible {
		t.Error("expected cordoned node to be ineligible for placement")
	}
	if fn.SchedulerEligible.RejectionCode != string(RejectNodeCordoned) {
		t.Errorf("expected rejection code NODE_CORDONED, got %s", fn.SchedulerEligible.RejectionCode)
	}
}

// TestCordonedNodeStaysReadyForExistingWorkloads verifies cordoned nodes keep existing workloads.
func TestCordonedNodeStaysReadyForExistingWorkloads(t *testing.T) {
	s := newState()
	now := time.Now().UnixMilli()

	// Create a ready node with running workload
	n := &Node{
		ID:   "node-1",
		Name: "node-1",
		Status: "ready",
		Enroll: api.Enroll{
			CPUMilli:      8000,
			MemBytes:      16000000000,
			Region:        "us-west",
			Zone:          "us-west-1a",
			Host:          "host-1",
			Tiers:         []string{"standard"},
			Arch:          "x86_64",
			Features:      []string{},
		},
		FailureDomain: "us-west-1a",
		ApprovedAt: now - 10000,
		LastObsAt:  now - 1000,
		Obs: &api.Observation{
			Facts: api.Facts{
				Arch:     "x86_64",
				MemBytes: 16000000000,
				CPUs:     8,
			},
			Workloads: []api.WorkloadObs{
				{
					Assignment: "wl-1",
					Observed:   "running",
				},
			},
		},
	}
	s.Nodes["node-1"] = n

	// Node should be ready and eligible
	fleet := s.FleetInventory()
	fn := fleet[0]
	if fn.Status != "ready" {
		t.Fatalf("expected status=ready, got %s", fn.Status)
	}
	if !fn.SchedulerEligible.Eligible {
		t.Fatal("expected node to be eligible initially")
	}

	// Cordon the node
	n.Cordoned = true

	// Verify node is still ready (cordon doesn't change status, only eligibility)
	fleet = s.FleetInventory()
	fn = fleet[0]
	if fn.Status != "ready" {
		t.Errorf("expected status=ready after cordon, got %s", fn.Status)
	}
	if fn.Cordoned != true {
		t.Error("expected Cordoned=true after cordon")
	}
	// Existing workloads can continue because status is still "ready"
	// Only NEW placements are rejected
	if fn.Workloads != 1 {
		t.Errorf("expected 1 running workload, got %d", fn.Workloads)
	}
}

// TestUncordonNodeRestoresEligibility verifies uncordoning restores eligibility.
func TestUncordonNodeRestoresEligibility(t *testing.T) {
	s := newState()
	now := time.Now().UnixMilli()

	// Create a ready node
	n := &Node{
		ID:   "node-1",
		Name: "node-1",
		Status: "ready",
		Enroll: api.Enroll{
			CPUMilli:      8000,
			MemBytes:      16000000000,
			Region:        "us-west",
			Zone:          "us-west-1a",
			Host:          "host-1",
			Tiers:         []string{"standard"},
			Arch:          "x86_64",
			Features:      []string{},
		},
		FailureDomain: "us-west-1a",
		ApprovedAt: now - 10000,
		LastObsAt:  now - 1000,
		Obs: &api.Observation{
			Facts: api.Facts{
				Arch:     "x86_64",
				MemBytes: 16000000000,
				CPUs:     8,
			},
			Workloads: []api.WorkloadObs{},
		},
	}
	s.Nodes["node-1"] = n

	// Cordon the node
	n.Cordoned = true

	// Verify cordoned
	fleet := s.FleetInventory()
	fn := fleet[0]
	if !fn.Cordoned {
		t.Fatal("expected Cordoned=true after cordon")
	}
	if fn.SchedulerEligible.Eligible {
		t.Fatal("expected ineligible after cordon")
	}

	// Uncordon the node
	n.Cordoned = false

	// Verify uncordoned and eligible again
	fleet = s.FleetInventory()
	fn = fleet[0]
	if fn.Cordoned {
		t.Error("expected Cordoned=false after uncordon")
	}
	if fn.Status != "ready" {
		t.Errorf("expected status=ready, got %s", fn.Status)
	}
	if !fn.SchedulerEligible.Eligible {
		t.Errorf("expected eligible after uncordon, got rejection: %s", fn.SchedulerEligible.RejectionReason)
	}
}

// TestOwnerReserveConstraint verifies available = total - owner - reserved.
func TestOwnerReserveConstraint(t *testing.T) {
	s := newState()
	now := time.Now().UnixMilli()

	// Create node with 8000 CPU millicores
	n := &Node{
		ID:   "node-1",
		Name: "node-1",
		Status: "ready",
		Enroll: api.Enroll{
			CPUMilli:      8000,
			MemBytes:      16000000000,
			Region:        "us-west",
			Zone:          "us-west-1a",
			Host:          "host-1",
			Tiers:         []string{"standard"},
			Arch:          "x86_64",
			Features:      []string{},
		},
		FailureDomain: "us-west-1a",
		ApprovedAt: now - 10000,
		LastObsAt:  now - 1000,
		Obs: &api.Observation{
			Facts: api.Facts{
				Arch:     "x86_64",
				MemBytes: 16000000000,
				CPUs:     8,
			},
			Workloads: []api.WorkloadObs{},
		},
	}
	s.Nodes["node-1"] = n

	// Set owner reserve to 1000 CPU, 1GB memory
	s.ResourceLedger.CapacityByNode["node-1"] = &NodeCapacityModel{
		NodeID:          "node-1",
		OwnerCPU:        1000,
		OwnerMem:        1000000000,
		TotalCPU:        8000,
		TotalMem:        16000000000,
	}

	// Reserve 2000 CPU, 2GB memory
	s.ResourceLedger.Reservations["res-1"] = &ResourceReservation{
		ID:       "res-1",
		NodeID:   "node-1",
		CPUMilli: 2000,
		MemBytes: 2000000000,
	}

	// Verify available = total - owner - reserved = 8000 - 1000 - 2000 = 5000
	fleet := s.FleetInventory()
	fn := fleet[0]
	if fn.CapacityTotal != 8000 {
		t.Errorf("expected total=8000, got %d", fn.CapacityTotal)
	}
	expectedAvailable := int64(5000)
	if fn.CapacityAvailable != expectedAvailable {
		t.Errorf("MODEL A constraint: available = total - owner - reserved = 8000 - 1000 - 2000 = 5000, got %d", fn.CapacityAvailable)
	}
}

// TestCordonedFleetSummary verifies cordoned nodes are counted separately.
func TestCordonedFleetSummary(t *testing.T) {
	s := newState()
	now := time.Now().UnixMilli()

	// Create 2 nodes
	for i := 1; i <= 2; i++ {
		nodeID := "node-" + string(rune('0'+i))
		n := &Node{
			ID:   nodeID,
			Name: nodeID,
			Status: "ready",
			Enroll: api.Enroll{
				CPUMilli:      8000,
				MemBytes:      16000000000,
				Region:        "us-west",
				Zone:          "us-west-1a",
				Host:          "host-" + string(rune('0'+i)),
				Tiers:         []string{"standard"},
				Arch:          "x86_64",
				Features:      []string{},
			},
			FailureDomain: "us-west-1a",
			ApprovedAt: now - 10000,
			LastObsAt:  now - 1000,
			Obs: &api.Observation{
				Facts: api.Facts{
					Arch:     "x86_64",
					MemBytes: 16000000000,
					CPUs:     8,
				},
				Workloads: []api.WorkloadObs{},
			},
		}
		s.Nodes[nodeID] = n
	}

	// Cordon node-2
	s.Nodes["node-2"].Cordoned = true

	// Verify fleet summary counts cordoned nodes
	summary := s.ComputeFleetSummary()
	if summary.TotalNodes != 2 {
		t.Errorf("expected total=2, got %d", summary.TotalNodes)
	}
	if summary.EligibleNodes != 1 {
		t.Errorf("expected eligible=1 (cordoned node excluded), got %d", summary.EligibleNodes)
	}
	if summary.CordonedNodes != 1 {
		t.Errorf("expected cordoned=1, got %d", summary.CordonedNodes)
	}
}

// TestCordonIdempotency verifies cordoning the same node twice is safe.
func TestCordonIdempotency(t *testing.T) {
	s := newState()
	now := time.Now().UnixMilli()

	// Create a ready node
	n := &Node{
		ID:   "node-1",
		Name: "node-1",
		Status: "ready",
		Enroll: api.Enroll{
			CPUMilli:      8000,
			MemBytes:      16000000000,
			Region:        "us-west",
			Zone:          "us-west-1a",
			Host:          "host-1",
			Tiers:         []string{"standard"},
			Arch:          "x86_64",
			Features:      []string{},
		},
		FailureDomain: "us-west-1a",
		ApprovedAt: now - 10000,
		LastObsAt:  now - 1000,
		Obs: &api.Observation{
			Facts: api.Facts{
				Arch:     "x86_64",
				MemBytes: 16000000000,
				CPUs:     8,
			},
			Workloads: []api.WorkloadObs{},
		},
	}
	s.Nodes["node-1"] = n

	// Cordon twice
	n.Cordoned = true
	// Second cordon should be idempotent (no-op)
	n.Cordoned = true

	// Verify node is still cordoned
	fleet := s.FleetInventory()
	fn := fleet[0]
	if !fn.Cordoned {
		t.Error("expected Cordoned=true")
	}
}

