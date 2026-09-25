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
