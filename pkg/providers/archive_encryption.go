package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// ArchiveEncryptor handles encryption and decryption of archives.
type ArchiveEncryptor struct {
	keyID     string
	algorithm string
	key       []byte
}

// NewArchiveEncryptor creates an archive encryptor instance.
func NewArchiveEncryptor(algorithm, keyID string, key []byte) (*ArchiveEncryptor, error) {
	if algorithm == "" {
		return nil, fmt.Errorf("encryption algorithm required")
	}

	if algorithm == "AES256" && len(key) != 32 {
		return nil, fmt.Errorf("AES256 requires 32-byte key, got %d bytes", len(key))
	}

	return &ArchiveEncryptor{
		keyID:     keyID,
		algorithm: algorithm,
		key:       key,
	}, nil
}

// Encrypt encrypts data using the configured algorithm.
func (ae *ArchiveEncryptor) Encrypt(data []byte) ([]byte, error) {
	switch ae.algorithm {
	case "AES256":
		return ae.encryptAES256(data)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", ae.algorithm)
	}
}

// Decrypt decrypts data using the configured algorithm.
func (ae *ArchiveEncryptor) Decrypt(encrypted []byte) ([]byte, error) {
	switch ae.algorithm {
	case "AES256":
		return ae.decryptAES256(encrypted)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", ae.algorithm)
	}
}

// encryptAES256 encrypts data using AES-256-GCM.
func (ae *ArchiveEncryptor) encryptAES256(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(ae.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, data, nil)
	return sealed, nil
}

// decryptAES256 decrypts data using AES-256-GCM.
func (ae *ArchiveEncryptor) decryptAES256(encrypted []byte) ([]byte, error) {
	block, err := aes.NewCipher(ae.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// ArchiveSigner handles signing and verification of archive integrity.
type ArchiveSigner struct {
	signingKey []byte
}

// NewArchiveSigner creates an archive signer instance.
func NewArchiveSigner(signingKey []byte) (*ArchiveSigner, error) {
	if len(signingKey) == 0 {
		return nil, fmt.Errorf("signing key required")
	}

	return &ArchiveSigner{
		signingKey: signingKey,
	}, nil
}

// Sign computes HMAC-SHA256 signature of data.
func (as *ArchiveSigner) Sign(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("data to sign required")
	}

	hash := sha256.New()
	hash.Write(append(as.signingKey, data...))
	signature := hash.Sum(nil)

	return fmt.Sprintf("%x", signature), nil
}

// Verify verifies HMAC-SHA256 signature of data.
func (as *ArchiveSigner) Verify(data []byte, signature string) (bool, error) {
	if len(data) == 0 {
		return false, fmt.Errorf("data to verify required")
	}

	computed, err := as.Sign(data)
	if err != nil {
		return false, err
	}

	return computed == signature, nil
}

// ArchiveManifest represents metadata for an encrypted archive.
type ArchiveManifest struct {
	CampaignID       string `json:"campaign_id"`
	Timestamp        int64  `json:"timestamp"`
	Size             int64  `json:"size"`
	Encryption       string `json:"encryption"`        // "none", "AES256"
	EncryptionKeyID  string `json:"encryption_key_id"`
	Signature        string `json:"signature"`         // HMAC-SHA256
	ContentHash      string `json:"content_hash"`      // SHA256 of plaintext
	BackendProvider  string `json:"backend_provider"`  // "s3", "gcs", "azure"
	StorageLocation  string `json:"storage_location"`  // Full path in storage
	RetentionPolicy  string `json:"retention_policy"`  // Lifecycle policy name
	CreatedAt        int64  `json:"created_at"`
	ExpiresAt        int64  `json:"expires_at"`
}
