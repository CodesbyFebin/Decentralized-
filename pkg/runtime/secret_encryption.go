package runtime

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// KeyEncryptionKey (KEK) encrypts Data Encryption Keys.
// KEK is replicated in control-plane state (wrapped or encrypted).
// Never exposed as plaintext outside control plane.
type KeyEncryptionKey struct {
	ID      string `json:"id"`           // "KEK-1", "KEK-2", etc.
	KeyData []byte `json:"-"`            // AES key material (256 bits)
	Version int64  `json:"version"`      // incremented on rotation
	Active  bool   `json:"active"`       // only one KEK active at a time
}

// DataEncryptionKey (DEK) encrypts individual secrets.
// DEK is derived from KEK; stored wrapped in Raft.
// Never exposed as plaintext.
type DataEncryptionKey struct {
	ID       string `json:"id"`      // "DEK-sec_deploy_name_hash-v1"
	SecretID string `json:"secretId"`
	Version  int64  `json:"version"`
	KeyData  []byte `json:"-"` // AES key material (256 bits)
}

// WrappedSecret holds ciphertext and encryption metadata for Raft storage.
type WrappedSecret struct {
	SecretID    string `json:"secretId"`
	Version     int64  `json:"version"`
	Algorithm   string `json:"algorithm"` // "AES-256-GCM"
	KeyID       string `json:"keyId"`     // KEK ID that wraps the DEK
	WrappedDEK  string `json:"wrappedDek"`  // base64(KEK_encrypt(DEK))
	Nonce       string `json:"nonce"`       // base64(96-bit random nonce)
	Ciphertext  string `json:"ciphertext"`  // base64(AES-GCM(plaintext, AAD, nonce))
	AAD         string `json:"aad"`         // base64(canonical JSON AAD)
	CreatedAt   int64  `json:"createdAt"`
	Lifecycle   string `json:"lifecycle"` // ACTIVE, SUPERSEDED, REVOKED, DELETED
}

// SecretsKeystore manages KEK and DEK generation, wrapping, and unwrapping.
type SecretsKeystore struct {
	activeKEK *KeyEncryptionKey
	kekCache  map[string]*KeyEncryptionKey // by ID
}

// NewSecretsKeystore creates a keystore.
func NewSecretsKeystore() *SecretsKeystore {
	return &SecretsKeystore{
		kekCache: make(map[string]*KeyEncryptionKey),
	}
}

// GenerateKEK creates a new Key Encryption Key (256-bit AES).
// Call during cluster initialization or key rotation.
// Result must be stored wrapped in State or provided by operator.
func (sk *SecretsKeystore) GenerateKEK(id string) (*KeyEncryptionKey, error) {
	keyMaterial := make([]byte, 32) // 256 bits
	if _, err := rand.Read(keyMaterial); err != nil {
		return nil, fmt.Errorf("failed to generate KEK key material: %w", err)
	}

	kek := &KeyEncryptionKey{
		ID:      id,
		KeyData: keyMaterial,
		Version: 1,
		Active:  false,
	}
	return kek, nil
}

// SetActiveKEK marks a KEK as active and caches it for wrap/unwrap.
func (sk *SecretsKeystore) SetActiveKEK(kek *KeyEncryptionKey) error {
	if kek.KeyData == nil || len(kek.KeyData) != 32 {
		return fmt.Errorf("invalid KEK key material length: %d", len(kek.KeyData))
	}
	sk.activeKEK = kek
	sk.kekCache[kek.ID] = kek
	return nil
}

// GenerateDEK creates a new Data Encryption Key (256-bit AES) for a secret.
func (sk *SecretsKeystore) GenerateDEK(secretID string, version int64) (*DataEncryptionKey, error) {
	if sk.activeKEK == nil {
		return nil, fmt.Errorf("no active KEK available for DEK generation")
	}

	keyMaterial := make([]byte, 32) // 256 bits
	if _, err := rand.Read(keyMaterial); err != nil {
		return nil, fmt.Errorf("failed to generate DEK key material: %w", err)
	}

	dekID := fmt.Sprintf("DEK-%s-v%d", secretID, version)
	dek := &DataEncryptionKey{
		ID:       dekID,
		SecretID: secretID,
		Version:  version,
		KeyData:  keyMaterial,
	}
	return dek, nil
}

// WrapDEK encrypts a DEK using the active KEK.
// Returns base64-encoded wrapped DEK.
// AAD format: secretId+version (must match UnwrapDEK exactly)
func (sk *SecretsKeystore) WrapDEK(dek *DataEncryptionKey) (string, error) {
	if sk.activeKEK == nil {
		return "", fmt.Errorf("no active KEK for DEK wrapping")
	}

	block, err := aes.NewCipher(sk.activeKEK.KeyData)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// AAD: secretId+version to bind wrapping to specific secret version
	// Format must match exactly in UnwrapDEK
	aad := []byte(fmt.Sprintf(`secretId=%s,version=%d`, dek.SecretID, dek.Version))
	ciphertext := gcm.Seal(nonce, nonce, dek.KeyData, aad)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// UnwrapDEK decrypts a wrapped DEK using the appropriate KEK.
// kekID specifies which KEK was used for wrapping.
// wrappedBase64 is the base64-encoded ciphertext.
// secretID and version must match the original wrapping (used in AAD).
func (sk *SecretsKeystore) UnwrapDEK(kekID string, wrappedBase64 string, secretID string, version int64) ([]byte, error) {
	kek, exists := sk.kekCache[kekID]
	if !exists {
		return nil, fmt.Errorf("KEK not found: %s", kekID)
	}

	wrapped, err := base64.StdEncoding.DecodeString(wrappedBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode wrapped DEK: %w", err)
	}

	block, err := aes.NewCipher(kek.KeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(wrapped) < nonceSize {
		return nil, fmt.Errorf("wrapped DEK too short")
	}

	nonce := wrapped[:nonceSize]
	ciphertext := wrapped[nonceSize:]
	// AAD format must match WrapDEK exactly
	aad := []byte(fmt.Sprintf(`secretId=%s,version=%d`, secretID, version))

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap DEK (authentication failed): %w", err)
	}

	return plaintext, nil
}

// EncryptSecret encrypts a secret plaintext using a DEK with AEAD.
// Returns WrappedSecret with ciphertext, nonce, and metadata for Raft storage.
func (sk *SecretsKeystore) EncryptSecret(secretID string, version int64, plaintext []byte,
	deploymentID, workloadID, environment string) (*WrappedSecret, error) {

	if sk.activeKEK == nil {
		return nil, fmt.Errorf("no active KEK for secret encryption")
	}

	// Generate DEK for this secret version
	dek, err := sk.GenerateDEK(secretID, version)
	if err != nil {
		return nil, err
	}

	// Wrap DEK with active KEK
	wrappedDEK, err := sk.WrapDEK(dek)
	if err != nil {
		return nil, err
	}

	// Encrypt secret with DEK using AEAD (AES-256-GCM)
	block, err := aes.NewCipher(dek.KeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher for secret: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM for secret: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// AAD: includes all context to prevent context switching attacks
	aad := []byte(fmt.Sprintf(
		`{"secretId":"%s","version":%d,"deploymentId":"%s","workloadId":"%s","environment":"%s","keyId":"%s"}`,
		secretID, version, deploymentID, workloadID, environment, sk.activeKEK.ID,
	))

	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)

	wrapped := &WrappedSecret{
		SecretID:   secretID,
		Version:    version,
		Algorithm:  "AES-256-GCM",
		KeyID:      sk.activeKEK.ID,
		WrappedDEK: wrappedDEK,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		AAD:        base64.StdEncoding.EncodeToString(aad),
		CreatedAt:  currentTimeMs(),
		Lifecycle:  "ACTIVE",
	}

	return wrapped, nil
}

// DecryptSecret decrypts a wrapped secret from Raft state.
// Returns plaintext secret value or error if authentication fails.
func (sk *SecretsKeystore) DecryptSecret(wrapped *WrappedSecret) ([]byte, error) {
	if wrapped.Lifecycle != "ACTIVE" {
		return nil, fmt.Errorf("secret not in ACTIVE state: %s", wrapped.Lifecycle)
	}

	// Unwrap DEK using the specified KEK
	dekPlaintext, err := sk.UnwrapDEK(wrapped.KeyID, wrapped.WrappedDEK, wrapped.SecretID, wrapped.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap DEK: %w", err)
	}

	// Recreate DEK for decryption
	dek := &DataEncryptionKey{
		ID:       fmt.Sprintf("DEK-%s-v%d", wrapped.SecretID, wrapped.Version),
		SecretID: wrapped.SecretID,
		Version:  wrapped.Version,
		KeyData:  dekPlaintext,
	}

	// Decrypt ciphertext
	block, err := aes.NewCipher(dek.KeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher for decryption: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM for decryption: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(wrapped.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(wrapped.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	aad, err := base64.StdEncoding.DecodeString(wrapped.AAD)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AAD: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret (authentication failed): %w", err)
	}

	return plaintext, nil
}

// currentTimeMs returns current time in milliseconds since epoch.
func currentTimeMs() int64 {
	return 0 // Placeholder; implement with time.Now().UnixMilli() in real code
}
