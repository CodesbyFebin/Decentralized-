package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistentNodeState represents node state on disk
type PersistentNodeState struct {
	NodeID         string    `json:"nodeId"`
	State          string    `json:"state"`
	DesiredState   string    `json:"desiredState"`
	ObservedAt     int64     `json:"observedAt"`
	LastTransition int64     `json:"lastTransition"`
	SourceID       string    `json:"sourceId"`
	Freshness      string    `json:"freshness"`
	Cordoned       bool      `json:"cordoned"`
	DrainTarget    int       `json:"drainTarget"`
	LastHeartbeat  int64     `json:"lastHeartbeat"`
	Reason         string    `json:"reason"`
	Generation     int64     `json:"generation"`
	SavedAt        time.Time `json:"savedAt"`
}

// NodeStateStore persists node state to disk
type NodeStateStore struct {
	rootPath string
	mu       sync.RWMutex
	cache    map[string]*PersistentNodeState
}

// NewNodeStateStore creates a persistent node state store
func NewNodeStateStore(rootPath string) (*NodeStateStore, error) {
	if rootPath == "" {
		rootPath = "/var/lib/decentralized/node-states"
	}

	if err := os.MkdirAll(rootPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %v", err)
	}

	store := &NodeStateStore{
		rootPath: rootPath,
		cache:    make(map[string]*PersistentNodeState),
	}

	if err := store.loadAllStates(); err != nil {
		return nil, fmt.Errorf("failed to load existing states: %v", err)
	}

	return store, nil
}

// SaveNodeState persists a node state to disk
func (nss *NodeStateStore) SaveNodeState(nodeID string, state *PersistentNodeState) error {
	nss.mu.Lock()
	defer nss.mu.Unlock()

	state.SavedAt = time.Now()
	nss.cache[nodeID] = state

	statePath := nss.getNodeStatePath(nodeID)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal node state: %v", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write node state: %v", err)
	}

	return nil
}

// LoadNodeState loads node state from disk
func (nss *NodeStateStore) LoadNodeState(nodeID string) (*PersistentNodeState, error) {
	nss.mu.RLock()
	if cached, ok := nss.cache[nodeID]; ok {
		nss.mu.RUnlock()
		return cached, nil
	}
	nss.mu.RUnlock()

	statePath := nss.getNodeStatePath(nodeID)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("node state not found: %s", nodeID)
		}
		return nil, fmt.Errorf("failed to read node state: %v", err)
	}

	var state PersistentNodeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node state: %v", err)
	}

	nss.mu.Lock()
	nss.cache[nodeID] = &state
	nss.mu.Unlock()

	return &state, nil
}

// LoadAllNodeStates returns all persisted node states
func (nss *NodeStateStore) LoadAllNodeStates() (map[string]*PersistentNodeState, error) {
	nss.mu.RLock()
	defer nss.mu.RUnlock()

	result := make(map[string]*PersistentNodeState)
	for nodeID, state := range nss.cache {
		result[nodeID] = state
	}
	return result, nil
}

// DeleteNodeState removes persisted state
func (nss *NodeStateStore) DeleteNodeState(nodeID string) error {
	nss.mu.Lock()
	defer nss.mu.Unlock()

	delete(nss.cache, nodeID)
	statePath := nss.getNodeStatePath(nodeID)
	return os.Remove(statePath)
}

// loadAllStates loads all states from disk into cache
func (nss *NodeStateStore) loadAllStates() error {
	entries, err := os.ReadDir(nss.rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		statePath := filepath.Join(nss.rootPath, entry.Name())
		data, err := os.ReadFile(statePath)
		if err != nil {
			continue
		}

		var state PersistentNodeState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		nss.cache[state.NodeID] = &state
	}

	return nil
}

// getNodeStatePath returns the file path for a node state
func (nss *NodeStateStore) getNodeStatePath(nodeID string) string {
	return filepath.Join(nss.rootPath, nodeID+".json")
}

// NodeStateReconciliation tracks DESIRED vs OBSERVED state
type NodeStateReconciliation struct {
	NodeID              string `json:"nodeId"`
	DesiredState        string `json:"desiredState"`
	ObservedState       string `json:"observedState"`
	Divergent           bool   `json:"divergent"`
	LastReconciliation  int64  `json:"lastReconciliation"`
	ReconciliationCount int    `json:"reconciliationCount"`
	Reason              string `json:"reason"`
}

// ReconciliationStore tracks reconciliation of desired vs observed state
type ReconciliationStore struct {
	rootPath string
	mu       sync.RWMutex
	cache    map[string]*NodeStateReconciliation
}

// NewReconciliationStore creates a new reconciliation store
func NewReconciliationStore(rootPath string) (*ReconciliationStore, error) {
	if rootPath == "" {
		rootPath = "/var/lib/decentralized/reconciliation"
	}

	if err := os.MkdirAll(rootPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create reconciliation directory: %v", err)
	}

	return &ReconciliationStore{
		rootPath: rootPath,
		cache:    make(map[string]*NodeStateReconciliation),
	}, nil
}

// RecordReconciliation saves reconciliation attempt
func (rs *ReconciliationStore) RecordReconciliation(nodeID, desired, observed string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	recon := &NodeStateReconciliation{
		NodeID:             nodeID,
		DesiredState:       desired,
		ObservedState:      observed,
		Divergent:          desired != observed,
		LastReconciliation: time.Now().UnixNano(),
	}

	if existing, ok := rs.cache[nodeID]; ok {
		recon.ReconciliationCount = existing.ReconciliationCount + 1
	}

	rs.cache[nodeID] = recon

	reconPath := filepath.Join(rs.rootPath, nodeID+"-reconciliation.json")
	data, err := json.MarshalIndent(recon, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(reconPath, data, 0600)
}

// GetReconciliationStatus returns current reconciliation state
func (rs *ReconciliationStore) GetReconciliationStatus(nodeID string) (*NodeStateReconciliation, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if recon, ok := rs.cache[nodeID]; ok {
		return recon, nil
	}

	return nil, fmt.Errorf("no reconciliation record for node: %s", nodeID)
}
