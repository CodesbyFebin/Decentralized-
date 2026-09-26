// Cryptographic operations for secrets encryption at rest.
package control

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/zeebo/blake3"
)

const (
	// AES-256 key size and GCM nonce size
	dexKeySize   = 32      // 256 bits
	gcmNonceSize = 12      // 96 bits (standard GCM)
	aesGCMTag    = "aes-256-gcm"
)

// GenerateDEK creates a cryptographically random 32-byte DEK for per-secret-version encryption.
func GenerateDEK() ([32]byte, error) {
	var dek [32]byte
	if _, err := rand.Read(dek[:]); err != nil {
		return dek, fmt.Errorf("generate DEK: %w", err)
	}
	return dek, nil
}

// GenerateNonce creates a random 96-bit nonce for GCM.
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, gcmNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return nonce, nil
}

// DeriveKEK derives a key-encryption key from operator-provided bootstrap material.
// Never call if bootstrap is empty (will panic).
func DeriveKEK(bootstrap string, clusterID string) [32]byte {
	// KEK = BLAKE3(bootstrap || "v1" || clusterID)[:32]
	h := blake3.New()
	h.Write([]byte(bootstrap))
	h.Write([]byte("v1"))
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

	// Construct AAD (scope binding)
	aad := []byte(fmt.Sprintf("%s|%s|%s|%s|%d",
		clusterID, deploymentID, workloadID, environment, version))

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

	// Reconstruct AAD (must match encryption)
	aad := []byte(fmt.Sprintf("%s|%s|%s|%s|%d",
		record.ClusterID, record.DeploymentID, record.WorkloadID, record.Environment, record.Version))

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
