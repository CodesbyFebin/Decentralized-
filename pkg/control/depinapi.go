package control

import (
	"encoding/json"
	"net/http"
	"time"

	"decentralized.host/pkg/providers"
	"decentralized.host/pkg/system"
)

// HardwareProfileResponse is what the API returns for node hardware
type HardwareProfileResponse struct {
	NodeID        string                 `json:"node_id"`
	Hostname      string                 `json:"hostname"`
	OS            string                 `json:"os"`
	Arch          string                 `json:"arch"`
	CPU           system.CPUInfo         `json:"cpu"`
	Memory        system.MemoryInfo      `json:"memory"`
	Disk          system.DiskInfo        `json:"disk"`
	GPU           []system.GPUInfo       `json:"gpu"`
	Network       []system.NetworkInfo   `json:"network"`
	ProbeTime     time.Time              `json:"probe_time"`
	LastObserved  time.Time              `json:"last_observed"`
	Source        string                 `json:"source"` // "live" or "cached"
}

// CompatibilityResponse is what the API returns for compatibility check
type CompatibilityResponse struct {
	NodeID       string                       `json:"node_id"`
	ProfileName  string                       `json:"profile_name"`
	Score        float64                      `json:"score"`
	Level        string                       `json:"level"`
	Components   providers.CompatibilityScore `json:"components"`
	Qualified    bool                         `json:"qualified"`
	Issues       []string                     `json:"issues"`
	EligibleDeploy bool                       `json:"eligible_deploy"`
}

// OwnerReserveResponse is what the API returns for reserve status
type OwnerReserveResponse struct {
	NodeID      string    `json:"node_id"`
	OwnerID     string    `json:"owner_id"`
	Status      string    `json:"status"` // "verified", "pending", "expired", "revoked"
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	VerifiedAt  time.Time `json:"verified_at,omitempty"`
	TokenSigner string    `json:"token_signer,omitempty"`
}

// DePINProfileInfo describes a hardware requirement profile
type DePINProfileInfo struct {
	Name        string `json:"name"`
	CPU         int    `json:"cpu_cores"`
	Memory      int64  `json:"memory_gb"`
	Storage     int64  `json:"storage_gb"`
	GPU         int64  `json:"gpu_vram_gb"`
	Description string `json:"description"`
}

// handleDePINProfiles lists available hardware profiles
func (s *Server) handleDePINProfiles(w http.ResponseWriter, _ *http.Request, _ *authz) {
	profiles := []DePINProfileInfo{
		{
			Name:        "compute-light",
			CPU:         2,
			Memory:      4,
			Storage:     50,
			GPU:         0,
			Description: "Light compute workloads: CI/CD runners, small services",
		},
		{
			Name:        "compute-standard",
			CPU:         4,
			Memory:      8,
			Storage:     100,
			GPU:         0,
			Description: "Standard compute: web services, APIs, microservices",
		},
		{
			Name:        "compute-heavy",
			CPU:         8,
			Memory:      16,
			Storage:     500,
			GPU:         6,
			Description: "Heavy compute with optional GPU: ML training, video processing",
		},
		{
			Name:        "gpu-optimized",
			CPU:         16,
			Memory:      32,
			Storage:     1024,
			GPU:         24,
			Description: "GPU-intensive: LLM inference, image generation, CUDA workloads",
		},
		{
			Name:        "storage-node",
			CPU:         2,
			Memory:      8,
			Storage:     5120,
			GPU:         0,
			Description: "Storage-optimized: distributed storage, object archival",
		},
	}
	writeJSON(w, 200, map[string]any{"profiles": profiles})
}

// handleNodeHardware returns the hardware profile for a node
func (s *Server) handleNodeHardware(w http.ResponseWriter, r *http.Request, _ *authz) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		writeErr(w, 400, "missing node id")
		return
	}

	var node *Node
	s.fsm.Read(func(st *State) {
		node = st.Nodes[nodeID]
	})
	if node == nil {
		writeErr(w, 404, "node not found")
		return
	}

	// For now, return cached hardware from enrollment
	// In Phase 4a, we'll add a background monitor that keeps this fresh
	if node.Hardware == nil {
		writeErr(w, 404, "hardware profile not yet available; probe pending")
		return
	}

	resp := HardwareProfileResponse{
		NodeID:      nodeID,
		Hostname:    node.Hardware.Hostname,
		OS:          node.Hardware.OS,
		Arch:        node.Hardware.Arch,
		CPU:         node.Hardware.CPU,
		Memory:      node.Hardware.Memory,
		Disk:        node.Hardware.Disk,
		GPU:         node.Hardware.GPU,
		Network:     node.Hardware.Network,
		ProbeTime:   node.Hardware.Probed,
		LastObserved: time.Now(),
		Source:      "cached",
	}
	writeJSON(w, 200, resp)
}

// handleNodeCompatibility checks node hardware against a profile
func (s *Server) handleNodeCompatibility(w http.ResponseWriter, r *http.Request, _ *authz) {
	nodeID := r.PathValue("id")
	profileName := r.URL.Query().Get("profile")

	if nodeID == "" {
		writeErr(w, 400, "missing node id")
		return
	}
	if profileName == "" {
		writeErr(w, 400, "missing profile query parameter")
		return
	}

	var node *Node
	s.fsm.Read(func(st *State) {
		node = st.Nodes[nodeID]
	})
	if node == nil {
		writeErr(w, 404, "node not found")
		return
	}

	if node.Hardware == nil {
		writeErr(w, 404, "hardware profile not yet available")
		return
	}

	// Get profile requirements
	reqs := profileRequirements(profileName)
	if reqs == nil {
		writeErr(w, 404, "profile not found")
		return
	}

	// Convert hardware profile to simple format
	simpleHW := &providers.SimpleHardwareProfile{
		CPUCores:     node.Hardware.CPU.Cores,
		CPUModel:     node.Hardware.CPU.Model,
		MemoryBytes:  node.Hardware.Memory.TotalBytes,
		StorageBytes: node.Hardware.Disk.TotalBytes,
		NetworkCount: len(node.Hardware.Network),
	}
	// Extract GPU types
	for _, gpu := range node.Hardware.GPU {
		simpleHW.GPUs = append(simpleHW.GPUs, gpu.Type)
	}

	// Score compatibility
	checker := providers.NewCompatibilityChecker()
	score := checker.CheckCompatibility(r.Context(), simpleHW, reqs)

	// Determine eligibility: requires GOOD (0.85+) or better and verified reserve
	eligible := score.Overall >= 0.85
	if eligible && node.OwnerReserve != nil {
		eligible = node.OwnerReserve.Status == "verified"
	}

	resp := CompatibilityResponse{
		NodeID:         nodeID,
		ProfileName:    profileName,
		Score:          score.Overall,
		Level:          score.GetCompatibilityLevel(),
		Components:     *score,
		Qualified:      score.Overall >= 0.85,
		Issues:         score.Issues,
		EligibleDeploy: eligible,
	}
	writeJSON(w, 200, resp)
}

// handleNodeReserve returns owner reserve status
func (s *Server) handleNodeReserve(w http.ResponseWriter, r *http.Request, _ *authz) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		writeErr(w, 400, "missing node id")
		return
	}

	var node *Node
	s.fsm.Read(func(st *State) {
		node = st.Nodes[nodeID]
	})
	if node == nil {
		writeErr(w, 404, "node not found")
		return
	}

	if node.OwnerReserve == nil {
		// No reserve yet
		writeJSON(w, 200, map[string]string{"status": "not_reserved"})
		return
	}

	resp := OwnerReserveResponse{
		NodeID:     nodeID,
		OwnerID:    node.OwnerReserve.OwnerID,
		Status:     node.OwnerReserve.Status,
		VerifiedAt: node.OwnerReserve.VerifiedAt,
	}
	if !node.OwnerReserve.ExpiresAt.IsZero() {
		resp.ExpiresAt = node.OwnerReserve.ExpiresAt
	}
	writeJSON(w, 200, resp)
}

// profileRequirements maps profile names to hardware requirements
func profileRequirements(name string) *providers.HardwareRequirement {
	switch name {
	case "compute-light":
		return &providers.HardwareRequirement{
			MinCPUCores:  2,
			MinMemoryGB:  4,
			MinStorageGB: 50,
		}
	case "compute-standard":
		return &providers.HardwareRequirement{
			MinCPUCores:  4,
			MinMemoryGB:  8,
			MinStorageGB: 100,
		}
	case "compute-heavy":
		return &providers.HardwareRequirement{
			MinCPUCores:    8,
			MinMemoryGB:    16,
			MinStorageGB:   500,
			RequiredGPU:    "nvidia",
			MinGPUMemoryGB: 6,
		}
	case "gpu-optimized":
		return &providers.HardwareRequirement{
			MinCPUCores:    16,
			MinMemoryGB:    32,
			MinStorageGB:   1000,
			RequiredGPU:    "nvidia",
			MinGPUMemoryGB: 24,
		}
	case "storage-node":
		return &providers.HardwareRequirement{
			MinCPUCores:  2,
			MinMemoryGB:  8,
			MinStorageGB: 5120,
		}
	}
	return nil
}

// OwnerReserveRequest is the payload for creating/verifying an owner reserve
type OwnerReserveRequest struct {
	OwnerID     string `json:"owner_id"`
	Host        string `json:"host"`
	TokenSigner string `json:"token_signer"`
}

// handleCreateReserve creates or updates an owner reserve
func (s *Server) handleCreateReserve(w http.ResponseWriter, r *http.Request, a *authz) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		writeErr(w, 400, "missing node id")
		return
	}

	var req OwnerReserveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid request: %v", err)
		return
	}

	// Propose reserve creation to FSM
	res, err := s.propose("create-reserve", a.Actor, map[string]any{
		"node":          nodeID,
		"owner_id":      req.OwnerID,
		"host":          req.Host,
		"token_signer":  req.TokenSigner,
	})
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}

	code := 200
	if !res.OK {
		code = http.StatusConflict
	}
	writeJSON(w, code, res)
}
