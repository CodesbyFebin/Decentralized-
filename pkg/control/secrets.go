// Package control persistence for application secrets encryption at rest.
package control

import (
	"encoding/json"
	"time"
)

// Secret represents a named secret with identity and metadata.
type Secret struct {
	ID        string `json:"id"`        // immutable: secret-<random>
	Name      string `json:"name"`      // e.g., app/web/db-password
	CreatedAt int64  `json:"createdAt"` // Unix nanoseconds
}

// SecretVersion represents a versioned plaintext (volatile, never persisted).
type SecretVersion struct {
	SecretID     string `json:"secretId"`
	Version      int32  `json:"version"`     // 1-indexed
	Data         []byte `json:"data"`        // plaintext (memory-only)
	DeploymentID string `json:"deploymentId"`
	WorkloadID   string `json:"workloadId"`
	Environment  string `json:"environment"` // "prod", "staging", etc.
	CreatedAt    int64  `json:"createdAt"`   // Unix ns
	EncryptedAt  int64  `json:"encryptedAt"` // Unix ns
}

// SecretRecord is the encrypted persistence unit (what lives in Raft/BoltDB).
type SecretRecord struct {
	SecretID      string `json:"secretId"`
	Version       int32  `json:"version"`
	Algorithm     string `json:"algorithm"`     // "aes-256-gcm"
	KeyID         string `json:"keyId"`        // identifies which DEK version
	EncryptedData []byte `json:"encryptedData"` // ciphertext
	Nonce         []byte `json:"nonce"`        // 96-bit random
	ClusterID     string `json:"clusterId"`
	DeploymentID  string `json:"deploymentId"`
	WorkloadID    string `json:"workloadId"`
	Environment   string `json:"environment"`
	CreatedAt     int64  `json:"createdAt"`
	EncryptedAt   int64  `json:"encryptedAt"`
}

// SecretsStore holds all persisted secret records, indexed by secretID then version.
type SecretsStore map[string]map[int32]*SecretRecord

// NewSecretsStore initializes an empty secrets store.
func NewSecretsStore() SecretsStore {
	return make(SecretsStore)
}

// AddRecord stores or updates a secret record.
func (s SecretsStore) AddRecord(record *SecretRecord) {
	if _, exists := s[record.SecretID]; !exists {
		s[record.SecretID] = make(map[int32]*SecretRecord)
	}
	s[record.SecretID][record.Version] = record
}

// GetRecord retrieves a specific secret version.
func (s SecretsStore) GetRecord(secretID string, version int32) *SecretRecord {
	if versions, exists := s[secretID]; exists {
		return versions[version]
	}
	return nil
}

// LatestVersion returns the highest version number for a secret, or 0 if none.
func (s SecretsStore) LatestVersion(secretID string) int32 {
	if versions, exists := s[secretID]; exists {
		var maxVersion int32
		for v := range versions {
			if v > maxVersion {
				maxVersion = v
			}
		}
		return maxVersion
	}
	return 0
}

// DeleteSecret removes all versions of a secret (hard delete).
func (s SecretsStore) DeleteSecret(secretID string) {
	delete(s, secretID)
}

// MarshalJSON serializes the secrets store (for Raft/persistence).
func (s SecretsStore) MarshalJSON() ([]byte, error) {
	// Convert map[string]map[int32]*SecretRecord to JSON-serializable form
	// (JSON doesn't support int32 as object keys, so normalize to string keys)
	normalized := make(map[string]map[string]*SecretRecord)
	for secretID, versions := range s {
		normalized[secretID] = make(map[string]*SecretRecord)
		for version, record := range versions {
			normalized[secretID][json.Number(version).String()] = record
		}
	}
	return json.Marshal(normalized)
}

// UnmarshalJSON deserializes the secrets store.
func (s *SecretsStore) UnmarshalJSON(data []byte) error {
	normalized := make(map[string]map[string]*SecretRecord)
	if err := json.Unmarshal(data, &normalized); err != nil {
		return err
	}
	*s = NewSecretsStore()
	for _, versions := range normalized {
		for versionStr, record := range versions {
			var version int32
			if tmpInt64, err := json.Number(versionStr).Int64(); err == nil {
				version = int32(tmpInt64)
			}
			(*s).AddRecord(&SecretRecord{
				SecretID:      record.SecretID,
				Version:       version,
				Algorithm:     record.Algorithm,
				KeyID:         record.KeyID,
				EncryptedData: record.EncryptedData,
				Nonce:         record.Nonce,
				ClusterID:     record.ClusterID,
				DeploymentID:  record.DeploymentID,
				WorkloadID:    record.WorkloadID,
				Environment:   record.Environment,
				CreatedAt:     record.CreatedAt,
				EncryptedAt:   record.EncryptedAt,
			})
		}
	}
	return nil
}

// KekStatus tracks KEK availability and bootstrap state.
type KekStatus struct {
	Locked    bool   // true if bootstrap missing/corrupted
	BootstrapHash string // BLAKE3 hash of bootstrap (never stores bootstrap itself)
	DerivedAt int64  // Unix ns when KEK was last derived
	Error     string // error message if locked
}

// NewKekStatus initializes the KEK status (initially locked until derived).
func NewKekStatus() KekStatus {
	return KekStatus{
		Locked: true,
		Error:  "KEK not yet derived from bootstrap",
	}
}

// SecretCommand is the Raft command for secret operations.
type SecretCommand struct {
	Type      string         `json:"type"`      // "secret-create", "secret-version-add"
	Timestamp int64          `json:"timestamp"` // Unix ns
	Actor     string         `json:"actor"`     // who requested (for audit)
	SecretID  string         `json:"secretId"`
	Record    *SecretRecord  `json:"record"`    // encrypted record
}

// Now returns current Unix nanoseconds (used in tests to mock time).
func Now() int64 {
	return time.Now().UnixNano()
}
