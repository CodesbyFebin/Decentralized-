package scheduler

import (
	"fmt"
	"reflect"
	"testing"

	"decentralized.host/pkg/api"
)

func nodes(n int) []Node {
	var out []Node
	for i := 0; i < n; i++ {
		out = append(out, Node{
			ID: fmt.Sprintf("dh1node%02d", i), Name: fmt.Sprintf("host-%c", 'a'+i), Status: "ready",
			Tiers: []string{"trusted"}, Region: fmt.Sprintf("cell-%d", i%3), Host: fmt.Sprintf("h%d", i),
			Arch: "amd64", CPUMilli: 2000, MemBytes: 2 << 30, DiskFree: 100 << 30,
			Policy: api.PolicySummary{AcceptTiers: []string{"trusted", "local"}, AllowRuntimes: []string{"process", "docker"}},
		})
	}
	return out
}

func spec(replicas int64) api.AppSpec {
	return api.AppSpec{Replicas: replicas, Runtime: "process", Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
		Placement: api.Placement{Tiers: []string{"trusted"}, Spread: "failure-domain", AntiAffinity: "hard"}}
}

func TestDeterministic(t *testing.T) {
	a := Schedule(Request{App: "wiki", Spec: spec(3), Nodes: nodes(5)})
	b := Schedule(Request{App: "wiki", Spec: spec(3), Nodes: nodes(5)})
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same input, different plan")
	}
	if len(a.Unscheduled()) != 0 {
		t.Fatal(a.Unscheduled())
	}
	seen := map[string]bool{}
	for _, r := range a.Replicas {
		if seen[r.Node] {
			t.Fatal("hard anti-affinity violated")
		}
		seen[r.Node] = true
	}
}

func TestStabilityKeepsValidPlacement(t *testing.T) {
	ns := nodes(5)
	cur := map[int64]string{0: ns[4].ID, 1: ns[3].ID}
	p := Schedule(Request{App: "wiki", Spec: spec(3), Nodes: ns, Current: cur})
	if p.Replicas[0].Node != ns[4].ID || !p.Replicas[0].Kept || p.Replicas[1].Node != ns[3].ID {
		t.Fatalf("valid placement moved: %+v", p.Replicas)
	}
}

func TestMovesOffRevokedHost(t *testing.T) {
	ns := nodes(3)
	ns[0].Status = "revoked"
	p := Schedule(Request{App: "wiki", Spec: spec(2), Nodes: ns, Current: map[int64]string{0: ns[0].ID}})
	for _, r := range p.Replicas {
		if r.Node == ns[0].ID {
			t.Fatal("replica kept on revoked host")
		}
	}
}

func TestTrustTierAndPolicyFilters(t *testing.T) {
	ns := nodes(2)
	ns[0].Policy.AcceptTiers = []string{"local"}
	ns[1].Policy.AllowRuntimes = []string{"docker"}
	p := Schedule(Request{App: "x", Spec: spec(1), Nodes: ns})
	if p.Replicas[0].Node != "" {
		t.Fatal("placed despite filters")
	}
	stages := map[string]bool{}
	for _, r := range p.Replicas[0].Rows {
		stages[r.Stage] = true
	}
	if !stages["trust"] || !stages["policy"] {
		t.Fatalf("explanations missing: %+v", p.Replicas[0].Rows)
	}
}

func TestVolumeSpreadAcrossDomains(t *testing.T) {
	ns := nodes(6) // regions cell-0,1,2 repeated
	p := PlaceVolume(VolumeRequest{Replicas: 3, SizeBytes: 1 << 30, Primary: ns[0].ID, Nodes: ns})
	if len(p.Members) != 3 || p.Members[0] != ns[0].ID {
		t.Fatal(p.Members)
	}
	domains := map[string]bool{}
	byID := map[string]Node{}
	for _, n := range ns {
		byID[n.ID] = n
	}
	for _, m := range p.Members {
		k := byID[m].Region
		if domains[k] {
			t.Fatalf("two replicas in region %s while others were free", k)
		}
		domains[k] = true
	}
}

func TestVolumeShortWhenNotEnoughHosts(t *testing.T) {
	ns := nodes(2)
	ns[1].DiskFree = 10
	p := PlaceVolume(VolumeRequest{Replicas: 3, SizeBytes: 1 << 30, Nodes: ns})
	if p.Short != 2 || len(p.Members) != 1 {
		t.Fatalf("%+v", p)
	}
}

func TestZoneAwareSpread(t *testing.T) {
	// 6 nodes across 3 zones, 2 per zone, same host name in zone
	// This tests that spread respects region|zone|host domains
	var ns []Node
	for zone := 0; zone < 3; zone++ {
		for i := 0; i < 2; i++ {
			ns = append(ns, Node{
				ID:       fmt.Sprintf("node%d%d", zone, i),
				Name:     fmt.Sprintf("host-z%d-%d", zone, i),
				Status:   "ready",
				Tiers:    []string{"compute"},
				Region:   "us-west",
				Zone:     fmt.Sprintf("us-west-2%c", 'a'+byte(zone)),
				Host:     fmt.Sprintf("h%d", zone),
				Arch:     "amd64",
				CPUMilli: 4000,
				MemBytes: 8 << 30,
				Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
			})
		}
	}

	// Request 3 replicas with failure-domain spread
	// With 3 zones and 3 replicas, should get one per zone
	req := Request{
		App: "web",
		Spec: api.AppSpec{
			Replicas: 3,
			Runtime:  "process",
			Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
			Placement: api.Placement{Tiers: []string{"compute"}, Spread: "failure-domain"},
		},
		Nodes: ns,
	}
	p := Schedule(req)

	if len(p.Unscheduled()) != 0 {
		t.Fatalf("failed to schedule: %+v", p.Unscheduled())
	}

	// Check spread across different domain keys
	domainMap := map[string]bool{}
	for _, r := range p.Replicas {
		for _, n := range ns {
			if n.ID == r.Node {
				dk := n.DomainKey()
				if domainMap[dk] {
					t.Fatalf("two replicas in same failure domain %s", dk)
				}
				domainMap[dk] = true
			}
		}
	}
	if len(domainMap) != 3 {
		t.Fatalf("expected spread across 3 failure domains, got %d", len(domainMap))
	}
}

func TestTierPreference(t *testing.T) {
	// Mixed tier cluster: 3 compute nodes, 2 storage nodes
	var ns []Node
	for i := 0; i < 3; i++ {
		ns = append(ns, Node{
			ID:       fmt.Sprintf("compute%d", i),
			Name:     fmt.Sprintf("compute-host-%d", i),
			Status:   "ready",
			Tiers:    []string{"compute", "standard"},
			Region:   "us-west",
			Zone:     "us-west-2a",
			Arch:     "amd64",
			CPUMilli: 8000,
			MemBytes: 16 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"compute", "standard"}},
		})
	}
	for i := 0; i < 2; i++ {
		ns = append(ns, Node{
			ID:       fmt.Sprintf("storage%d", i),
			Name:     fmt.Sprintf("storage-host-%d", i),
			Status:   "ready",
			Tiers:    []string{"storage"},
			Region:   "us-west",
			Zone:     "us-west-2b",
			Arch:     "amd64",
			CPUMilli: 2000,
			MemBytes: 4 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"storage"}},
		})
	}

	req := Request{
		App: "app",
		Spec: api.AppSpec{
			Replicas: 3,
			Runtime:  "process",
			Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
			Placement: api.Placement{Tiers: []string{"compute"}},
		},
		Nodes: ns,
	}
	p := Schedule(req)

	if len(p.Unscheduled()) != 0 {
		t.Fatalf("failed to schedule on compute tier: %+v", p.Unscheduled())
	}

	for _, r := range p.Replicas {
		if !contains([]string{"compute0", "compute1", "compute2"}, r.Node) {
			t.Fatalf("replica placed on wrong tier: %s", r.Node)
		}
	}
}

func TestSoftAntiAffinity(t *testing.T) {
	// 4 nodes in same zone
	var ns []Node
	for i := 0; i < 4; i++ {
		ns = append(ns, Node{
			ID:       fmt.Sprintf("node%d", i),
			Name:     fmt.Sprintf("host-%d", i),
			Status:   "ready",
			Tiers:    []string{"compute"},
			Region:   "us-west",
			Zone:     "us-west-2a",
			Host:     fmt.Sprintf("h%d", i),
			Arch:     "amd64",
			CPUMilli: 4000,
			MemBytes: 8 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
		})
	}

	// Soft anti-affinity allows multiple replicas on same host
	req := Request{
		App: "web",
		Spec: api.AppSpec{
			Replicas: 2,
			Runtime:  "process",
			Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
			Placement: api.Placement{Tiers: []string{"compute"}, AntiAffinity: "soft"},
		},
		Nodes: ns,
	}
	p := Schedule(req)

	if len(p.Unscheduled()) != 0 {
		t.Fatalf("failed to schedule with soft anti-affinity: %+v", p.Unscheduled())
	}

	// With soft anti-affinity, replicas CAN be on same node
	// but scheduler should prefer spreading if possible
	nodeMap := map[string]int{}
	for _, r := range p.Replicas {
		nodeMap[r.Node]++
	}
	if len(nodeMap) < 2 {
		t.Logf("soft anti-affinity: replicas colocated on fewer nodes than available")
	}
}

func TestPreferRegion(t *testing.T) {
	// 6 nodes: 3 in us-west, 3 in us-east
	var ns []Node
	for region, reg := range []string{"us-west", "us-east"} {
		for i := 0; i < 3; i++ {
			ns = append(ns, Node{
				ID:       fmt.Sprintf("%s-node%d", reg, i),
				Name:     fmt.Sprintf("%s-host-%d", reg, i),
				Status:   "ready",
				Tiers:    []string{"compute"},
				Region:   reg,
				Zone:     fmt.Sprintf("%s-2a", reg),
				Host:     fmt.Sprintf("h%d", region*3+i),
				Arch:     "amd64",
				CPUMilli: 4000,
				MemBytes: 8 << 30,
				Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
			})
		}
	}

	req := Request{
		App: "app",
		Spec: api.AppSpec{
			Replicas: 2,
			Runtime:  "process",
			Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
			Placement: api.Placement{
				Tiers:         []string{"compute"},
				PreferRegion:  "us-west",
				AntiAffinity:  "hard",
				Spread:        "failure-domain",
			},
		},
		Nodes: ns,
	}
	p := Schedule(req)

	if len(p.Unscheduled()) != 0 {
		t.Fatalf("failed to schedule with region preference: %+v", p.Unscheduled())
	}

	// Both replicas should prefer us-west region
	for _, r := range p.Replicas {
		for _, n := range ns {
			if n.ID == r.Node && n.Region != "us-west" {
				t.Fatalf("replica placed outside preferred region: %s", r.Node)
			}
		}
	}
}

func TestArchRequirement(t *testing.T) {
	// Cluster with mixed architectures
	ns := []Node{
		{
			ID:       "amd64-1",
			Name:     "host-x86",
			Status:   "ready",
			Tiers:    []string{"compute"},
			Region:   "us-west",
			Zone:     "us-west-2a",
			Arch:     "amd64",
			CPUMilli: 4000,
			MemBytes: 8 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
		},
		{
			ID:       "arm64-1",
			Name:     "host-arm",
			Status:   "ready",
			Tiers:    []string{"compute"},
			Region:   "us-west",
			Zone:     "us-west-2a",
			Arch:     "arm64",
			CPUMilli: 4000,
			MemBytes: 8 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
		},
	}

	req := Request{
		App: "app",
		Spec: api.AppSpec{
			Replicas: 1,
			Runtime:  "process",
			Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
			Placement: api.Placement{
				Tiers: []string{"compute"},
				Arch:  []string{"arm64"},
			},
		},
		Nodes: ns,
	}
	p := Schedule(req)

	if len(p.Unscheduled()) != 0 {
		t.Fatalf("failed to schedule on required architecture")
	}

	if p.Replicas[0].Node != "arm64-1" {
		t.Fatalf("wrong architecture selected: %s", p.Replicas[0].Node)
	}
}

func TestFeatureRequirements(t *testing.T) {
	// Cluster with selective features
	ns := []Node{
		{
			ID:       "gpu-node",
			Name:     "host-gpu",
			Status:   "ready",
			Tiers:    []string{"compute"},
			Region:   "us-west",
			Zone:     "us-west-2a",
			Arch:     "amd64",
			Features: []string{"gpu-nvidia", "sse4"},
			CPUMilli: 16000,
			MemBytes: 32 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
		},
		{
			ID:       "cpu-only",
			Name:     "host-cpu",
			Status:   "ready",
			Tiers:    []string{"compute"},
			Region:   "us-west",
			Zone:     "us-west-2a",
			Arch:     "amd64",
			Features: []string{"sse4"},
			CPUMilli: 4000,
			MemBytes: 8 << 30,
			Policy:   api.PolicySummary{AcceptTiers: []string{"compute"}},
		},
	}

	req := Request{
		App: "ml-app",
		Spec: api.AppSpec{
			Replicas: 1,
			Runtime:  "process",
			Resources: api.Resources{CPUMilli: 500, MemBytes: 256 << 20},
			Placement: api.Placement{
				Tiers:    []string{"compute"},
				Features: []string{"gpu-nvidia"},
			},
		},
		Nodes: ns,
	}
	p := Schedule(req)

	if len(p.Unscheduled()) != 0 {
		t.Fatalf("failed to schedule with feature requirement")
	}

	if p.Replicas[0].Node != "gpu-node" {
		t.Fatalf("wrong node selected for feature requirement: %s", p.Replicas[0].Node)
	}
}

func TestVolumeDomainSpread(t *testing.T) {
	// 6 nodes across 3 zones for volume replication
	// Each zone has 2 nodes with the same host name (single host per zone for failure domain)
	var ns []Node
	for zone := 0; zone < 3; zone++ {
		for i := 0; i < 2; i++ {
			ns = append(ns, Node{
				ID:       fmt.Sprintf("node%d%d", zone, i),
				Name:     fmt.Sprintf("host-z%d-%d", zone, i),
				Status:   "ready",
				Tiers:    []string{"storage"},
				Region:   "us-west",
				Zone:     fmt.Sprintf("us-west-2%c", 'a'+byte(zone)),
				Host:     fmt.Sprintf("h%d", zone), // same host per zone
				DiskFree: 1000 << 30,
				Policy:   api.PolicySummary{AcceptTiers: []string{"storage"}},
			})
		}
	}

	p := PlaceVolume(VolumeRequest{
		Replicas:  3,
		SizeBytes: 100 << 30,
		Nodes:     ns,
		Tiers:     []string{"storage"},
	})

	if p.Short != 0 {
		t.Fatalf("unable to place all volume replicas: short=%d", p.Short)
	}

	// Verify spread across failure domains (region|zone|host)
	domainMap := map[string]bool{}
	for _, mid := range p.Members {
		for _, n := range ns {
			if n.ID == mid {
				dk := n.DomainKey()
				if domainMap[dk] {
					t.Fatalf("two volume replicas in same failure domain %s", dk)
				}
				domainMap[dk] = true
				break
			}
		}
	}
	if len(domainMap) != 3 {
		t.Fatalf("volume replicas not spread across 3 failure domains: got %d", len(domainMap))
	}
}
