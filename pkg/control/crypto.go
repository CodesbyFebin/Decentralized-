// Cryptographic operations for secrets encryption at rest.
package control

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/zeebo/blake3"
)

const (
	// AES-256 key size and GCM nonce size
	dexKeySize   = 32      // 256 bits
	gcmNonceSize = 12      // 96 bits (standard GCM)
	aesGCMTag    = "aes-256-gcm"
	aadVersion   = "v1"    // AAD encoding version
)

// CanonicalAAD constructs length-prefixed AAD with all scope and algorithm fields.
// Format: len(field1):field1:|len(field2):field2:|...|version(8-byte big-endian)
// This prevents ambiguity from field delimiters and ensures all scope is bound.
// Fields (in order): algorithm | keyId | clusterId | secretId | deploymentId | workloadId | environment | version
func CanonicalAAD(secretID, algorithm, keyID, clusterID, deploymentID, workloadID, environment string, version int32) []byte {
	var buf bytes.Buffer
	fields := []string{
		algorithm,     // algorithm (e.g., "aes-256-gcm")
		keyID,         // keyId (deployment-specific key identifier)
		clusterID,     // clusterId (cluster identity)
		secretID,      // secretId (secret identity)
		deploymentID,  // deploymentId (deployment scoping)
		workloadID,    // workloadId (workload scoping)
		environment,   // environment (e.g., "prod", "staging")
	}
	for _, f := range fields {
		fmt.Fprintf(&buf, "%d:%s:", len(f), f)
	}
	// Append version as 8-byte big-endian integer
	versionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(versionBytes, uint64(version))
	buf.Write(versionBytes)
	return buf.Bytes()
}

// GenerateDEK creates a cryptographically random 32-byte DEK for per-secret-version encryption.
func GenerateDEK() ([32]byte, error) {
	var dek [32]byte
	if _, err := rand.Read(dek[:]); err != nil {
		return dek, fmt.Errorf("generate DEK: %w", err)
	}
	return dek, nil
}

// GenerateNonce creates a random 96-bit nonce for GCM.
//
// GCM nonce uniqueness requirement: A given (key, nonce) pair MUST NEVER be reused.
// With random 96-bit nonces and a unique DEK per secret version, collision risk is negligible:
// - Each secret version gets an independent random DEK
// - Nonce is random 96-bit (~79 bits of entropy after accounting for birthday bound)
// - With 2^80 encryptions per DEK, collision probability approaches 2^-80
//
// Practical bound: Per secret version, ~2^62 encryptions before reaching acceptable risk threshold.
// In practice, secret versions rotate frequently; this bound is never approached.
//
// See: https://csrc.nist.gov/publications/detail/sp/800-38d/final
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, gcmNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return nonce, nil
}

// DeriveKEK derives a key-encryption key from operator-provided bootstrap material.
//
// Bootstrap MUST be high-entropy random material (32 bytes of cryptographically random data).
// MUST NOT be a human-readable passphrase or password; those require a password KDF (e.g., Argon2).
//
// Bootstrap is never persisted in Raft/BoltDB; it must be provided externally at startup
// (e.g., from environment variable DECENTRALIZED_KEK_BOOTSTRAP).
//
// Derivation: KEK = BLAKE3(bootstrap || "v1" || clusterID)[:32]
// - "v1" ties derivation to AAD version
// - clusterID prevents cross-cluster key reuse
// - BLAKE3 provides cryptographic strength and speed
//
// Never call if bootstrap is empty; fail-closed before startup proceeds.
func DeriveKEK(bootstrap string, clusterID string) [32]byte {
	h := blake3.New()
	h.Write([]byte(bootstrap))
	h.Write([]byte(aadVersion))
	h.Write([]byte(clusterID))
	var kek [32]byte
	copy(kek[:], h.Sum(nil)[:32])
	return kek
}

// EncryptSecret encrypts a plaintext secret using DEK with AES-256-GCM.
// Returns a SecretRecord with ciphertext, nonce, and authenticated metadata (AAD).
func EncryptSecret(
	plaintext []byte,
	secretID string,
	version int32,
	dek [32]byte,
	clusterID string,
	deploymentID string,
	workloadID string,
	environment string,
	keyID string,
) (*SecretRecord, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("encrypt secret: plaintext empty")
	}

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(dek[:])
	if err != nil {
		return nil, fmt.Errorf("encrypt secret: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encrypt secret: create GCM: %w", err)
	}

	// Generate random nonce
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("encrypt secret: %w", err)
	}

	// Construct canonical AAD (scope and algorithm binding)
	aad := CanonicalAAD(secretID, aesGCMTag, keyID, clusterID, deploymentID, workloadID, environment, version)

	// Encrypt with AEAD
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)

	now := Now()
	return &SecretRecord{
		SecretID:      secretID,
		Version:       version,
		Algorithm:     aesGCMTag,
		KeyID:         keyID,
		EncryptedData: ciphertext,
		Nonce:         nonce,
		ClusterID:     clusterID,
		DeploymentID:  deploymentID,
		WorkloadID:    workloadID,
		Environment:   environment,
		CreatedAt:     now,
		EncryptedAt:   now,
	}, nil
}

// DecryptSecret recovers plaintext from a SecretRecord using DEK.
// Verifies AAD scope binding; fails if tampering is detected.
func DecryptSecret(record *SecretRecord, dek [32]byte) ([]byte, error) {
	if record == nil {
		return nil, errors.New("decrypt secret: record nil")
	}
	if len(record.EncryptedData) == 0 {
		return nil, errors.New("decrypt secret: encrypted data empty")
	}
	if len(record.Nonce) != gcmNonceSize {
		return nil, fmt.Errorf("decrypt secret: invalid nonce size %d", len(record.Nonce))
	}
	if record.Algorithm != aesGCMTag {
		return nil, fmt.Errorf("decrypt secret: unsupported algorithm %s", record.Algorithm)
	}

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(dek[:])
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: create GCM: %w", err)
	}

	// Reconstruct canonical AAD (must match encryption exactly)
	aad := CanonicalAAD(record.SecretID, record.Algorithm, record.KeyID, record.ClusterID, record.DeploymentID, record.WorkloadID, record.Environment, record.Version)

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, record.Nonce, record.EncryptedData, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: AEAD verification failed (tampering detected): %w", err)
	}

	return plaintext, nil
}

// WrapDEK encrypts a DEK using KEK for cluster replication.
// Returns wrapped DEK with authentication tag.
func WrapDEK(dek [32]byte, kek [32]byte, clusterID string) ([]byte, error) {
	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(kek[:])
	if err != nil {
		return nil, fmt.Errorf("wrap DEK: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("wrap DEK: create GCM: %w", err)
	}

	// Generate random nonce for DEK wrapping
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("wrap DEK: %w", err)
	}

	// Use clusterID as AAD for DEK wrapping
	aad := []byte(clusterID)

	// Wrap DEK with authenticated encryption
	wrapped := gcm.Seal(nil, nonce, dek[:], aad)

	// Prepend nonce to wrapped data (nonce must be readable for unwrapping)
	wrappedWithNonce := make([]byte, len(nonce)+len(wrapped))
	copy(wrappedWithNonce, nonce)
	copy(wrappedWithNonce[len(nonce):], wrapped)

	return wrappedWithNonce, nil
}

// UnwrapDEK decrypts a wrapped DEK using KEK.
// Fails if tampering is detected or if wrapped data is malformed.
func UnwrapDEK(wrappedDEK []byte, kek [32]byte, clusterID string) ([32]byte, error) {
	var dek [32]byte

	if len(wrappedDEK) < gcmNonceSize {
		return dek, errors.New("unwrap DEK: wrapped data too short")
	}

	// Extract nonce from prepended data
	nonce := wrappedDEK[:gcmNonceSize]
	ciphertext := wrappedDEK[gcmNonceSize:]

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(kek[:])
	if err != nil {
		return dek, fmt.Errorf("unwrap DEK: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return dek, fmt.Errorf("unwrap DEK: create GCM: %w", err)
	}

	// Decrypt with authentication
	aad := []byte(clusterID)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return dek, fmt.Errorf("unwrap DEK: AEAD verification failed (tampering detected): %w", err)
	}

	if len(plaintext) != dexKeySize {
		return dek, fmt.Errorf("unwrap DEK: invalid plaintext size %d", len(plaintext))
	}

	copy(dek[:], plaintext)
	return dek, nil
}

// EncodeBootstrap base64url-encodes bootstrap material (for env var passing).
func EncodeBootstrap(material []byte) string {
	return base64.RawURLEncoding.EncodeToString(material)
}

// DecodeBootstrap base64url-decodes bootstrap material.
func DecodeBootstrap(encoded string) ([]byte, error) {
	material, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode bootstrap: %w", err)
	}
	if len(material) != dexKeySize {
		return nil, fmt.Errorf("decode bootstrap: invalid size %d (expected %d)", len(material), dexKeySize)
	}
	return material, nil
}

// BootstrapHash returns a BLAKE3 hash of bootstrap material (never stores plaintext).
func BootstrapHash(bootstrap string) string {
	h := blake3.New()
	h.Write([]byte(bootstrap))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil)[:16])
}
