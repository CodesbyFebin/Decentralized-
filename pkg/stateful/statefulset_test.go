package stateful

import (
	"testing"
	"time"

	"decentralized.host/pkg/api"
)

func TestStatefulSetCreate(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "cassandra",
		Replicas:    3,
		ServiceName: "cassandra-headless",
		UpdateStrategy: "RollingUpdate",
		PodManagementPolicy: "Ordered",
		Template: api.AppSpec{
			Image: "cassandra:4.0",
		},
		VolumeClaimTemplates: []VolumeClaimTemplate{
			{
				Name:         "data",
				Size:         "10Gi",
				AccessMode:   "ReadWriteOnce",
				StorageClass: "fast-ssd",
			},
		},
	}

	err := mgr.Create(spec)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify pods were created
	pods, err := mgr.ListPods("cassandra")
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}
	if len(pods) != 3 {
		t.Errorf("expected 3 pods, got %d", len(pods))
	}

	// Verify pod names
	for i := 0; i < 3; i++ {
		pod := pods[i]
		expectedName := "cassandra-" + string(rune('0'+i))
		if pod.Name != expectedName {
			t.Errorf("pod %d: expected name %s, got %s", i, expectedName, pod.Name)
		}
		if pod.Ordinal != i {
			t.Errorf("pod %d: expected ordinal %d, got %d", i, i, pod.Ordinal)
		}
		if pod.State != "desired" {
			t.Errorf("pod %d: expected state desired, got %s", i, pod.State)
		}
	}

	// Verify volume bindings
	vols, err := mgr.ListVolumes("cassandra")
	if err != nil {
		t.Fatalf("ListVolumes failed: %v", err)
	}
	if len(vols) != 3 {
		t.Errorf("expected 3 volumes, got %d", len(vols))
	}

	// Check volume naming
	for i, vol := range vols {
		expectedPVC := "cassandra-data-" + string(rune('0'+i))
		if vol.PVC != expectedPVC {
			t.Errorf("volume %d: expected PVC %s, got %s", i, expectedPVC, vol.PVC)
		}
	}

	// Try to create duplicate (should fail)
	err = mgr.Create(spec)
	if err == nil {
		t.Error("expected duplicate create to fail")
	}
}

func TestPodStateTransitions(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "redis",
		Replicas:    2,
		ServiceName: "redis",
	}
	mgr.Create(spec)

	// Get initial pod
	pod, err := mgr.GetPod("redis", 0)
	if err != nil {
		t.Fatalf("GetPod failed: %v", err)
	}
	if pod.State != "desired" {
		t.Errorf("expected state desired, got %s", pod.State)
	}

	// Transition: desired -> admitted
	err = mgr.UpdatePodState("redis", 0, "admitted", "pod admitted by policy")
	if err != nil {
		t.Fatalf("UpdatePodState failed: %v", err)
	}

	pod, _ = mgr.GetPod("redis", 0)
	if pod.State != "admitted" {
		t.Errorf("expected state admitted, got %s", pod.State)
	}

	// Transition: admitted -> executing
	err = mgr.UpdatePodState("redis", 0, "executing", "pod executing")
	if err != nil {
		t.Fatalf("UpdatePodState failed: %v", err)
	}

	// Transition: executing -> observed
	err = mgr.UpdatePodState("redis", 0, "observed", "pod observed as running")
	if err != nil {
		t.Fatalf("UpdatePodState failed: %v", err)
	}

	pod, _ = mgr.GetPod("redis", 0)
	if pod.State != "observed" {
		t.Errorf("expected state observed, got %s", pod.State)
	}
	if !pod.Ready {
		t.Error("expected pod to be ready")
	}

	// Transition: observed -> verified
	err = mgr.UpdatePodState("redis", 0, "verified", "pod verified healthy")
	if err != nil {
		t.Fatalf("UpdatePodState failed: %v", err)
	}

	// Invalid transition (skip a state)
	err = mgr.UpdatePodState("redis", 1, "verified", "invalid")
	if err == nil {
		t.Error("expected invalid transition to fail")
	}
}

func TestOrderedDeployment(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "postgres",
		Replicas:    5,
		ServiceName: "postgres",
	}
	mgr.Create(spec)

	plan, err := mgr.GetOrderedDeploymentPlan("postgres")
	if err != nil {
		t.Fatalf("GetOrderedDeploymentPlan failed: %v", err)
	}

	if len(plan) != 5 {
		t.Errorf("expected 5 pods, got %d", len(plan))
	}

	// Verify ordinal order
	for i := 0; i < 5; i++ {
		if plan[i].Ordinal != i {
			t.Errorf("pod %d: expected ordinal %d, got %d", i, i, plan[i].Ordinal)
		}
	}
}

func TestOrderedTermination(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "mysql",
		Replicas:    4,
		ServiceName: "mysql",
	}
	mgr.Create(spec)

	plan, err := mgr.GetOrderedTerminationPlan("mysql")
	if err != nil {
		t.Fatalf("GetOrderedTerminationPlan failed: %v", err)
	}

	if len(plan) != 4 {
		t.Errorf("expected 4 pods, got %d", len(plan))
	}

	// Verify reverse ordinal order
	expectedOrder := []int{3, 2, 1, 0}
	for i, expected := range expectedOrder {
		if plan[i].Ordinal != expected {
			t.Errorf("position %d: expected ordinal %d, got %d", i, expected, plan[i].Ordinal)
		}
	}
}

func TestComputeStatus(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "etcd",
		Replicas:    3,
		ServiceName: "etcd",
	}
	mgr.Create(spec)

	// Initially, all pods are in desired state
	status, err := mgr.ComputeStatus("etcd")
	if err != nil {
		t.Fatalf("ComputeStatus failed: %v", err)
	}

	if status.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", status.Replicas)
	}
	if status.ReadyReplicas != 0 {
		t.Errorf("expected 0 ready replicas, got %d", status.ReadyReplicas)
	}

	// Transition pods to ready
	for i := 0; i < 3; i++ {
		mgr.UpdatePodState("etcd", i, "admitted", "")
		mgr.UpdatePodState("etcd", i, "executing", "")
		mgr.UpdatePodState("etcd", i, "observed", "")
	}

	status, _ = mgr.ComputeStatus("etcd")
	if status.ReadyReplicas != 3 {
		t.Errorf("expected 3 ready replicas, got %d", status.ReadyReplicas)
	}

	// Check Ready condition
	var readyCondition *Condition
	for i := range status.Conditions {
		if status.Conditions[i].Type == "Ready" {
			readyCondition = &status.Conditions[i]
			break
		}
	}

	if readyCondition == nil {
		t.Fatal("Ready condition not found")
	}
	if readyCondition.Status != "True" {
		t.Errorf("expected Ready status True, got %s", readyCondition.Status)
	}
}

func TestPodPlacement(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "kafka",
		Replicas:    3,
		ServiceName: "kafka",
	}
	mgr.Create(spec)

	// Place pods on specific nodes
	nodes := []string{"node-1", "node-2", "node-3"}
	for i, node := range nodes {
		err := mgr.UpdatePodPlacement("kafka", i, node)
		if err != nil {
			t.Fatalf("UpdatePodPlacement failed: %v", err)
		}
	}

	// Verify placement
	pods, _ := mgr.ListPods("kafka")
	for i, pod := range pods {
		if pod.RunningOn != nodes[i] {
			t.Errorf("pod %d: expected node %s, got %s", i, nodes[i], pod.RunningOn)
		}
	}
}

func TestDelete(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "mongodb",
		Replicas:    2,
		ServiceName: "mongodb",
	}
	mgr.Create(spec)

	// Verify it exists
	_, err := mgr.GetSpec("mongodb")
	if err != nil {
		t.Fatalf("GetSpec failed: %v", err)
	}

	// Delete it
	err = mgr.Delete("mongodb")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = mgr.GetSpec("mongodb")
	if err == nil {
		t.Error("expected GetSpec to fail after delete")
	}

	// Try to get pods (should fail)
	_, err = mgr.ListPods("mongodb")
	if err == nil {
		t.Error("expected ListPods to fail after delete")
	}
}

func TestMultipleStatefulSets(t *testing.T) {
	mgr := New()

	specs := []Spec{
		{Name: "app1", Replicas: 2, ServiceName: "app1"},
		{Name: "app2", Replicas: 3, ServiceName: "app2"},
		{Name: "app3", Replicas: 1, ServiceName: "app3"},
	}

	for _, spec := range specs {
		if err := mgr.Create(spec); err != nil {
			t.Fatalf("Create %s failed: %v", spec.Name, err)
		}
	}

	// Verify each has correct pod count
	testCases := []struct {
		name     string
		expected int32
	}{
		{"app1", 2},
		{"app2", 3},
		{"app3", 1},
	}

	for _, tc := range testCases {
		pods, err := mgr.ListPods(tc.name)
		if err != nil {
			t.Fatalf("ListPods %s failed: %v", tc.name, err)
		}
		if int32(len(pods)) != tc.expected {
			t.Errorf("%s: expected %d pods, got %d", tc.name, tc.expected, len(pods))
		}
	}

	// Update one statefulset
	mgr.UpdatePodState("app2", 0, "admitted", "")
	mgr.UpdatePodState("app2", 0, "executing", "")
	mgr.UpdatePodState("app2", 0, "observed", "")

	status, _ := mgr.ComputeStatus("app2")
	if status.ReadyReplicas != 1 {
		t.Errorf("app2: expected 1 ready replica, got %d", status.ReadyReplicas)
	}
}

func TestHostnamePodIdentity(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "zookeeper",
		Replicas:    3,
		ServiceName: "zk-headless",
	}
	mgr.Create(spec)

	pods, _ := mgr.ListPods("zookeeper")

	expectedHostnames := []string{
		"zookeeper-0.zk-headless",
		"zookeeper-1.zk-headless",
		"zookeeper-2.zk-headless",
	}

	for i, pod := range pods {
		if pod.Hostname != expectedHostnames[i] {
			t.Errorf("pod %d: expected hostname %s, got %s", i, expectedHostnames[i], pod.Hostname)
		}
	}
}

func TestReadinessTransitionTiming(t *testing.T) {
	mgr := New()

	spec := Spec{
		Name:        "prometheus",
		Replicas:    1,
		ServiceName: "prometheus",
	}
	mgr.Create(spec)

	before := time.Now()
	mgr.UpdatePodState("prometheus", 0, "admitted", "")
	mgr.UpdatePodState("prometheus", 0, "executing", "")
	mgr.UpdatePodState("prometheus", 0, "observed", "")
	after := time.Now()

	pod, _ := mgr.GetPod("prometheus", 0)

	if !pod.Ready {
		t.Error("expected pod to be ready")
	}
	if pod.ReadyAt.Before(before) || pod.ReadyAt.After(after) {
		t.Errorf("ReadyAt time not within expected range")
	}
}
