// Package stateful provides StatefulSet orchestration for Decentralized.Host.
//
// StatefulSet is an abstraction for stateful workloads that require:
//   - Pod identity: persistent, predictable pod names
//   - Ordered deployment: pods deployed in sequence
//   - Ordered termination: pods terminated in reverse
//   - Persistent storage: volumes bound to pod identity
//   - Network identity: stable DNS names per pod (headless service support)
//
// Each pod is identified by its ordinal (0-based index).
// Pod names are deterministic: <name>-<ordinal>
// Volume claims are bound per pod: <name>-<volume>-<ordinal>
// DNS names resolve to individual pods, not the service.
//
// State machine:
//   DESIRED -> ADMITTED -> EXECUTING -> OBSERVED -> VERIFIED
//
// Each pod tracks its own lifecycle within the statefulset.
package stateful

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"decentralized.host/pkg/api"
)

// Pod represents one member of a StatefulSet with its identity and state.
type Pod struct {
	Name              string            `json:"name"`               // <statefulset>-<ordinal>
	Ordinal           int               `json:"ordinal"`            // 0-based index
	StatefulSetName   string            `json:"statefulset_name"`
	State             string            `json:"state"`              // desired|admitted|executing|observed|verified
	Hostname          string            `json:"hostname"`           // network identity
	RunningOn         string            `json:"running_on"`         // node ID
	VolumeBindings    map[string]string `json:"volume_bindings"`    // volume name -> PVC name
	ReadyAt           time.Time         `json:"ready_at"`
	ObservedGeneration int64            `json:"observed_generation"`
	LastTransition    time.Time         `json:"last_transition"`
	Ready             bool              `json:"ready"`
	Reason            string            `json:"reason"`
}

// Spec defines a StatefulSet's desired state.
type Spec struct {
	Name              string                  `json:"name"`
	Replicas          int32                   `json:"replicas"`
	Selector          map[string]string       `json:"selector"`
	ServiceName       string                  `json:"service_name"`        // headless service
	UpdateStrategy    string                  `json:"update_strategy"`     // RollingUpdate | OnDelete
	PodManagementPolicy string               `json:"pod_management_policy"` // Ordered | Parallel
	Template          api.AppSpec             `json:"template"`
	VolumeClaimTemplates []VolumeClaimTemplate `json:"volume_claim_templates"`
}

// VolumeClaimTemplate defines dynamic volume provisioning per pod.
type VolumeClaimTemplate struct {
	Name      string `json:"name"`
	Size      string `json:"size"`      // e.g., "10Gi"
	AccessMode string `json:"access_mode"` // ReadWriteOnce | ReadOnlyMany | ReadWriteMany
	StorageClass string `json:"storage_class"` // storage class name
}

// Status tracks the StatefulSet's observed state.
type Status struct {
	ObservedGeneration int64         `json:"observed_generation"`
	ReadyReplicas      int32         `json:"ready_replicas"`
	CurrentReplicas    int32         `json:"current_replicas"`
	UpdatedReplicas    int32         `json:"updated_replicas"`
	Replicas           int32         `json:"replicas"`
	Conditions         []Condition   `json:"conditions"`
}

// Condition represents the status of a StatefulSet.
type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"` // True | False | Unknown
	LastUpdateTime     time.Time `json:"last_update_time"`
	LastTransitionTime time.Time `json:"last_transition_time"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// Manager orchestrates StatefulSet lifecycles.
type Manager struct {
	mu           sync.RWMutex
	statefulsets map[string]*Spec
	pods         map[string][]Pod           // statefulset name -> pods
	volumes      map[string][]VolumeBinding // statefulset name -> volume bindings
	nextGen      map[string]int64           // generation tracker
}

// VolumeBinding tracks a volume claim to its pod.
type VolumeBinding struct {
	PVC        string `json:"pvc"`
	PodName    string `json:"pod_name"`
	VolumeName string `json:"volume_name"`
	Size       string `json:"size"`
}

// New creates a new StatefulSet manager.
func New() *Manager {
	return &Manager{
		statefulsets: make(map[string]*Spec),
		pods:         make(map[string][]Pod),
		volumes:      make(map[string][]VolumeBinding),
		nextGen:      make(map[string]int64),
	}
}

// Create creates a new StatefulSet.
func (m *Manager) Create(spec Spec) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.statefulsets[spec.Name]; exists {
		return fmt.Errorf("StatefulSet %s already exists", spec.Name)
	}

	m.statefulsets[spec.Name] = &spec
	m.nextGen[spec.Name] = 1
	m.pods[spec.Name] = make([]Pod, 0, spec.Replicas)
	m.volumes[spec.Name] = make([]VolumeBinding, 0)

	// Initialize pods in DESIRED state
	for i := int32(0); i < spec.Replicas; i++ {
		pod := Pod{
			Name:            fmt.Sprintf("%s-%d", spec.Name, i),
			Ordinal:         int(i),
			StatefulSetName: spec.Name,
			State:           "desired",
			Hostname:        fmt.Sprintf("%s-%d.%s", spec.Name, i, spec.ServiceName),
			VolumeBindings:  make(map[string]string),
		}

		// Create volume claim templates
		for _, vct := range spec.VolumeClaimTemplates {
			pvc := fmt.Sprintf("%s-%s-%d", spec.Name, vct.Name, i)
			pod.VolumeBindings[vct.Name] = pvc

			binding := VolumeBinding{
				PVC:        pvc,
				PodName:    pod.Name,
				VolumeName: vct.Name,
				Size:       vct.Size,
			}
			m.volumes[spec.Name] = append(m.volumes[spec.Name], binding)
		}

		m.pods[spec.Name] = append(m.pods[spec.Name], pod)
	}

	return nil
}

// GetPod retrieves a pod by statefulset and ordinal.
func (m *Manager) GetPod(statefulsetName string, ordinal int) (Pod, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pods, ok := m.pods[statefulsetName]
	if !ok {
		return Pod{}, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	for _, p := range pods {
		if p.Ordinal == ordinal {
			return p, nil
		}
	}

	return Pod{}, fmt.Errorf("Pod ordinal %d not found in %s", ordinal, statefulsetName)
}

// UpdatePodState updates a pod's state and transitions.
func (m *Manager) UpdatePodState(statefulsetName string, ordinal int, newState string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pods, ok := m.pods[statefulsetName]
	if !ok {
		return fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	for i, p := range pods {
		if p.Ordinal == ordinal {
			validTransitions := map[string][]string{
				"desired":   {"admitted"},
				"admitted":  {"executing"},
				"executing": {"observed"},
				"observed":  {"verified"},
				"verified":  {"executing"}, // for rolling updates
			}

			allowed := false
			for _, next := range validTransitions[p.State] {
				if next == newState {
					allowed = true
					break
				}
			}

			if !allowed && p.State != newState {
				return fmt.Errorf("invalid transition %s -> %s", p.State, newState)
			}

			p.State = newState
			p.LastTransition = time.Now()
			p.Reason = reason

			if newState == "observed" {
				p.Ready = true
				p.ReadyAt = time.Now()
			} else if newState == "executing" {
				p.Ready = false
			}

			pods[i] = p
			return nil
		}
	}

	return fmt.Errorf("Pod ordinal %d not found", ordinal)
}

// UpdatePodPlacement assigns a pod to a node.
func (m *Manager) UpdatePodPlacement(statefulsetName string, ordinal int, nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pods, ok := m.pods[statefulsetName]
	if !ok {
		return fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	for i, p := range pods {
		if p.Ordinal == ordinal {
			p.RunningOn = nodeID
			pods[i] = p
			return nil
		}
	}

	return fmt.Errorf("Pod ordinal %d not found", ordinal)
}

// ListPods returns all pods in a StatefulSet.
func (m *Manager) ListPods(statefulsetName string) ([]Pod, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pods, ok := m.pods[statefulsetName]
	if !ok {
		return nil, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	return append([]Pod(nil), pods...), nil
}

// ListVolumes returns all volume bindings for a StatefulSet.
func (m *Manager) ListVolumes(statefulsetName string) ([]VolumeBinding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vols, ok := m.volumes[statefulsetName]
	if !ok {
		return nil, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	return append([]VolumeBinding(nil), vols...), nil
}

// ComputeStatus computes the current status of a StatefulSet.
func (m *Manager) ComputeStatus(statefulsetName string) (Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	spec, ok := m.statefulsets[statefulsetName]
	if !ok {
		return Status{}, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	pods, _ := m.pods[statefulsetName]

	status := Status{
		ObservedGeneration: m.nextGen[statefulsetName],
		Replicas:           int32(len(pods)),
	}

	for _, p := range pods {
		if p.Ready {
			status.ReadyReplicas++
		}
		if p.State == "verified" {
			status.UpdatedReplicas++
		}
		if p.State != "desired" {
			status.CurrentReplicas++
		}
	}

	// Add conditions based on status
	readyCondition := Condition{
		Type:               "Ready",
		Status:             "False",
		LastUpdateTime:     time.Now(),
		LastTransitionTime: time.Now(),
	}

	if status.ReadyReplicas == spec.Replicas {
		readyCondition.Status = "True"
		readyCondition.Message = "all replicas ready"
	} else {
		readyCondition.Message = fmt.Sprintf("%d/%d replicas ready", status.ReadyReplicas, spec.Replicas)
	}

	status.Conditions = append(status.Conditions, readyCondition)

	return status, nil
}

// GetOrderedDeploymentPlan returns pods to deploy in order.
func (m *Manager) GetOrderedDeploymentPlan(statefulsetName string) ([]Pod, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pods, ok := m.pods[statefulsetName]
	if !ok {
		return nil, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	// Return pods sorted by ordinal
	sorted := append([]Pod(nil), pods...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Ordinal < sorted[j].Ordinal
	})

	return sorted, nil
}

// GetOrderedTerminationPlan returns pods to terminate in reverse order.
func (m *Manager) GetOrderedTerminationPlan(statefulsetName string) ([]Pod, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pods, ok := m.pods[statefulsetName]
	if !ok {
		return nil, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	// Return pods sorted by ordinal in reverse
	sorted := append([]Pod(nil), pods...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Ordinal > sorted[j].Ordinal
	})

	return sorted, nil
}

// Delete removes a StatefulSet and all its pods.
func (m *Manager) Delete(statefulsetName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.statefulsets[statefulsetName]; !ok {
		return fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	delete(m.statefulsets, statefulsetName)
	delete(m.pods, statefulsetName)
	delete(m.volumes, statefulsetName)
	delete(m.nextGen, statefulsetName)

	return nil
}

// GetSpec retrieves a StatefulSet spec.
func (m *Manager) GetSpec(statefulsetName string) (Spec, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	spec, ok := m.statefulsets[statefulsetName]
	if !ok {
		return Spec{}, fmt.Errorf("StatefulSet %s not found", statefulsetName)
	}

	return *spec, nil
}

// ReadinessProbe checks if all pods in a StatefulSet are ready.
type ReadinessProbe struct {
	PeriodSeconds    int32  `json:"period_seconds"`
	TimeoutSeconds   int32  `json:"timeout_seconds"`
	InitialDelaySeconds int32 `json:"initial_delay_seconds"`
	SuccessThreshold int32  `json:"success_threshold"`
	FailureThreshold int32  `json:"failure_threshold"`
	HTTPGet          *HTTPGetAction `json:"http_get,omitempty"`
	TCPSocket        *TCPSocketAction `json:"tcp_socket,omitempty"`
	Exec             *ExecAction `json:"exec,omitempty"`
}

// HTTPGetAction defines an HTTP health check.
type HTTPGetAction struct {
	Path   string `json:"path"`
	Port   int32  `json:"port"`
	Scheme string `json:"scheme"` // HTTP | HTTPS
}

// TCPSocketAction defines a TCP health check.
type TCPSocketAction struct {
	Port int32 `json:"port"`
}

// ExecAction defines a command-based health check.
type ExecAction struct {
	Command []string `json:"command"`
}

// WaitCondition specifies how long to wait for a pod to be ready.
type WaitCondition struct {
	ConditionType   string         `json:"condition_type"`   // Ready | Available | etc
	Timeout         time.Duration  `json:"timeout"`
	ReadinessProbe  *ReadinessProbe `json:"readiness_probe,omitempty"`
}
