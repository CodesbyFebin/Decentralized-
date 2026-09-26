package control

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestGenerateDEK(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatalf("GenerateDEK: %v", err)
	}
	if len(dek) != 32 {
		t.Errorf("DEK size: got %d, want 32", len(dek))
	}

	// Verify randomness: two generations should differ
	dek2, _ := GenerateDEK()
	if dek == dek2 {
		t.Error("DEK randomness: two generations identical (extremely unlikely)")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce: %v", err)
	}
	if len(nonce) != 12 {
		t.Errorf("nonce size: got %d, want 12", len(nonce))
	}

	// Verify randomness
	nonce2, _ := GenerateNonce()
	if bytes.Equal(nonce, nonce2) {
		t.Error("nonce randomness: two generations identical (extremely unlikely)")
	}
}

func TestDeriveKEK(t *testing.T) {
	bootstrap1 := make([]byte, 32)
	bootstrap1[0] = 1
	bootstrap := base64.RawURLEncoding.EncodeToString(bootstrap1)
	clusterID := "cluster-abc123"

	kek := DeriveKEK(bootstrap, clusterID)
	if len(kek) != 32 {
		t.Errorf("KEK size: got %d, want 32", len(kek))
	}

	// Same bootstrap and clusterID should produce same KEK
	kek2 := DeriveKEK(bootstrap, clusterID)
	if kek != kek2 {
		t.Error("KEK derivation: same inputs produced different outputs")
	}

	// Different bootstrap should produce different KEK
	bootstrap2Bytes := make([]byte, 32)
	bootstrap2Bytes[0] = 2
	bootstrap2 := base64.RawURLEncoding.EncodeToString(bootstrap2Bytes)
	kek3 := DeriveKEK(bootstrap2, clusterID)
	if kek == kek3 {
		t.Error("KEK derivation: different bootstraps produced same KEK")
	}

	// Different clusterID should produce different KEK
	kek4 := DeriveKEK(bootstrap, "cluster-xyz789")
	if kek == kek4 {
		t.Error("KEK derivation: different clusterIDs produced same KEK")
	}
}

func TestEncryptDecryptSecret_RoundTrip(t *testing.T) {
	plaintext := []byte("super-secret-database-password")
	dek, _ := GenerateDEK()
	secretID := "secret-abc123"
	version := int32(1)
	clusterID := "cluster-123"
	deploymentID := "deployment-456"
	workloadID := "workload-789"
	environment := "prod"
	keyID := "key-001"

	// Encrypt
	record, err := EncryptSecret(plaintext, secretID, version, dek, clusterID, deploymentID, workloadID, environment, keyID)
	if err != nil {
		t.Fatalf("EncryptSecret: %v", err)
	}

	// Verify record structure
	if record.SecretID != secretID {
		t.Errorf("SecretID mismatch")
	}
	if record.Version != version {
		t.Errorf("Version mismatch")
	}
	if record.Algorithm != aesGCMTag {
		t.Errorf("Algorithm mismatch")
	}
	if len(record.Nonce) != 12 {
		t.Errorf("Nonce size: got %d, want 12", len(record.Nonce))
	}
	if len(record.EncryptedData) == 0 {
		t.Error("Encrypted data empty")
	}

	// Decrypt
	recovered, err := DecryptSecret(record, dek)
	if err != nil {
		t.Fatalf("DecryptSecret: %v", err)
	}

	if !bytes.Equal(plaintext, recovered) {
		t.Errorf("Plaintext mismatch: got %v, want %v", recovered, plaintext)
	}
}

func TestEncryptSecret_WrongDEK_Rejected(t *testing.T) {
	plaintext := []byte("secret-data")
	dek, _ := GenerateDEK()
	wrongDEK, _ := GenerateDEK()

	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Decrypt with wrong DEK should fail
	_, err := DecryptSecret(record, wrongDEK)
	if err == nil {
		t.Error("DecryptSecret with wrong DEK: expected error, got nil")
	}
}

func TestDecryptSecret_CiphertextTamper_Rejected(t *testing.T) {
	plaintext := []byte("secret-data")
	dek, _ := GenerateDEK()

	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Tamper with ciphertext
	if len(record.EncryptedData) > 0 {
		record.EncryptedData[0] ^= 0xFF
	}

	_, err := DecryptSecret(record, dek)
	if err == nil {
		t.Error("DecryptSecret with tampered ciphertext: expected error, got nil")
	}
}

func TestDecryptSecret_NonceTamper_Rejected(t *testing.T) {
	plaintext := []byte("secret-data")
	dek, _ := GenerateDEK()

	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Tamper with nonce
	if len(record.Nonce) > 0 {
		record.Nonce[0] ^= 0xFF
	}

	_, err := DecryptSecret(record, dek)
	if err == nil {
		t.Error("DecryptSecret with tampered nonce: expected error, got nil")
	}
}

func TestDecryptSecret_AADTamper_Rejected(t *testing.T) {
	plaintext := []byte("secret-data")
	dek, _ := GenerateDEK()

	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Tamper with scope binding
	record.ClusterID = "cluster-2"

	_, err := DecryptSecret(record, dek)
	if err == nil {
		t.Error("DecryptSecret with tampered AAD: expected error, got nil")
	}
}

func TestDecryptSecret_VersionTamper_Rejected(t *testing.T) {
	plaintext := []byte("secret-data")
	dek, _ := GenerateDEK()

	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Tamper with version (part of AAD)
	record.Version = 2

	_, err := DecryptSecret(record, dek)
	if err == nil {
		t.Error("DecryptSecret with tampered version: expected error, got nil")
	}
}

func TestWrapUnwrapDEK_RoundTrip(t *testing.T) {
	dek, _ := GenerateDEK()
	kek, _ := GenerateDEK()
	clusterID := "cluster-123"

	// Wrap DEK
	wrapped, err := WrapDEK(dek, kek, clusterID)
	if err != nil {
		t.Fatalf("WrapDEK: %v", err)
	}

	if len(wrapped) == 0 {
		t.Error("Wrapped DEK empty")
	}

	// Unwrap DEK
	recovered, err := UnwrapDEK(wrapped, kek, clusterID)
	if err != nil {
		t.Fatalf("UnwrapDEK: %v", err)
	}

	if dek != recovered {
		t.Error("DEK recovery: wrapped and recovered DEKs differ")
	}
}

func TestUnwrapDEK_WrongKEK_Rejected(t *testing.T) {
	dek, _ := GenerateDEK()
	kek, _ := GenerateDEK()
	wrongKEK, _ := GenerateDEK()
	clusterID := "cluster-123"

	wrapped, _ := WrapDEK(dek, kek, clusterID)

	// Unwrap with wrong KEK should fail
	_, err := UnwrapDEK(wrapped, wrongKEK, clusterID)
	if err == nil {
		t.Error("UnwrapDEK with wrong KEK: expected error, got nil")
	}
}

func TestUnwrapDEK_AADTamper_Rejected(t *testing.T) {
	dek, _ := GenerateDEK()
	kek, _ := GenerateDEK()
	clusterID := "cluster-123"

	wrapped, _ := WrapDEK(dek, kek, clusterID)

	// Unwrap with different clusterID (AAD tamper)
	_, err := UnwrapDEK(wrapped, kek, "cluster-456")
	if err == nil {
		t.Error("UnwrapDEK with tampered AAD: expected error, got nil")
	}
}

func TestUnwrapDEK_WrappedTamper_Rejected(t *testing.T) {
	dek, _ := GenerateDEK()
	kek, _ := GenerateDEK()
	clusterID := "cluster-123"

	wrapped, _ := WrapDEK(dek, kek, clusterID)

	// Tamper with wrapped data (after nonce)
	if len(wrapped) > 12 {
		wrapped[13] ^= 0xFF
	}

	_, err := UnwrapDEK(wrapped, kek, clusterID)
	if err == nil {
		t.Error("UnwrapDEK with tampered wrapped data: expected error, got nil")
	}
}

func TestSecretVersionIndependence(t *testing.T) {
	plaintext := []byte("same-secret-two-versions")
	secretID := "secret-1"
	clusterID := "cluster-1"

	// Encrypt same plaintext with different DEKs (simulating version rollover)
	dek1, _ := GenerateDEK()
	record1, _ := EncryptSecret(plaintext, secretID, 1, dek1, clusterID, "deploy-1", "workload-1", "prod", "key-1")

	dek2, _ := GenerateDEK()
	record2, _ := EncryptSecret(plaintext, secretID, 2, dek2, clusterID, "deploy-1", "workload-1", "prod", "key-2")

	// Ciphertexts must differ (different DEKs and nonces)
	if bytes.Equal(record1.EncryptedData, record2.EncryptedData) {
		t.Error("Version independence: same plaintext produced identical ciphertexts")
	}

	// Both must decrypt correctly with their respective DEKs
	recovered1, err := DecryptSecret(record1, dek1)
	if err != nil || !bytes.Equal(recovered1, plaintext) {
		t.Error("Version 1 decryption failed")
	}

	recovered2, err := DecryptSecret(record2, dek2)
	if err != nil || !bytes.Equal(recovered2, plaintext) {
		t.Error("Version 2 decryption failed")
	}

	// Cross-version decryption must fail
	_, err = DecryptSecret(record1, dek2)
	if err == nil {
		t.Error("Cross-version decryption with dek2 on record1: should fail")
	}

	_, err = DecryptSecret(record2, dek1)
	if err == nil {
		t.Error("Cross-version decryption with dek1 on record2: should fail")
	}
}

func TestEncryptSecret_EmptyPlaintext_Rejected(t *testing.T) {
	dek, _ := GenerateDEK()

	_, err := EncryptSecret([]byte{}, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")
	if err == nil {
		t.Error("EncryptSecret with empty plaintext: expected error, got nil")
	}
}

func TestEncodeDecodeBootstrap(t *testing.T) {
	// Generate random 32-byte bootstrap
	original := make([]byte, 32)
	_, _ = rand.Read(original)

	// Encode
	encoded := EncodeBootstrap(original)
	if len(encoded) == 0 {
		t.Error("EncodeBootstrap: empty result")
	}

	// Decode
	decoded, err := DecodeBootstrap(encoded)
	if err != nil {
		t.Fatalf("DecodeBootstrap: %v", err)
	}

	if !bytes.Equal(original, decoded) {
		t.Error("Bootstrap encode/decode: round-trip failed")
	}
}

func TestDecodeBootstrap_InvalidSize_Rejected(t *testing.T) {
	// Encode a 16-byte value (wrong size)
	bad := make([]byte, 16)
	encoded := EncodeBootstrap(bad)

	_, err := DecodeBootstrap(encoded)
	if err == nil {
		t.Error("DecodeBootstrap with wrong size: expected error, got nil")
	}
}

func TestBootstrapHash(t *testing.T) {
	bootstrap1 := make([]byte, 32)
	bootstrap1[0] = 1
	bootstrap := base64.RawURLEncoding.EncodeToString(bootstrap1)

	hash := BootstrapHash(bootstrap)
	if len(hash) == 0 {
		t.Error("BootstrapHash: empty result")
	}

	// Same bootstrap should produce same hash
	hash2 := BootstrapHash(bootstrap)
	if hash != hash2 {
		t.Error("BootstrapHash: same input produced different hash")
	}

	// Different bootstrap should produce different hash
	bootstrap2Bytes := make([]byte, 32)
	bootstrap2Bytes[0] = 2
	bootstrap2 := base64.RawURLEncoding.EncodeToString(bootstrap2Bytes)
	hash3 := BootstrapHash(bootstrap2)
	if hash == hash3 {
		t.Error("BootstrapHash: different inputs produced same hash")
	}
}

func TestNonceUniqueness(t *testing.T) {
	plaintext := []byte("test-secret")
	dek, _ := GenerateDEK()
	nonces := make(map[string]bool)

	// Encrypt same plaintext 100 times, verify all nonces differ
	for i := 0; i < 100; i++ {
		record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")
		nonceStr := string(record.Nonce)
		if nonces[nonceStr] {
			t.Errorf("Nonce uniqueness: duplicate nonce at iteration %d", i)
			break
		}
		nonces[nonceStr] = true
	}

	if len(nonces) != 100 {
		t.Errorf("Nonce uniqueness: got %d unique nonces, want 100", len(nonces))
	}
}
