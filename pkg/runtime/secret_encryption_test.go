package runtime

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
)

// TestSecretsKeystore_KEKGeneration verifies KEK generation and management.
func TestSecretsKeystore_KEKGeneration(t *testing.T) {
	ks := NewSecretsKeystore()

	// Generate KEK
	kek, err := ks.GenerateKEK("KEK-1")
	if err != nil {
		t.Fatalf("failed to generate KEK: %v", err)
	}

	if kek.ID != "KEK-1" {
		t.Errorf("KEK ID mismatch: %s", kek.ID)
	}

	if len(kek.KeyData) != 32 {
		t.Errorf("KEK key material length: got %d, want 32 bytes", len(kek.KeyData))
	}

	if kek.Version != 1 {
		t.Errorf("KEK version: got %d, want 1", kek.Version)
	}

	// Set as active
	if err := ks.SetActiveKEK(kek); err != nil {
		t.Fatalf("failed to set active KEK: %v", err)
	}

	if ks.activeKEK != kek {
		t.Error("active KEK not set correctly")
	}

	// Verify cached
	if cached, ok := ks.kekCache["KEK-1"]; !ok || cached != kek {
		t.Error("KEK not cached")
	}
}

// TestSecretsKeystore_DEKGeneration verifies DEK generation with active KEK.
func TestSecretsKeystore_DEKGeneration(t *testing.T) {
	ks := NewSecretsKeystore()

	// Must have active KEK
	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// Generate DEK
	dek, err := ks.GenerateDEK("sec_deploy_name_hash", 1)
	if err != nil {
		t.Fatalf("failed to generate DEK: %v", err)
	}

	if dek.SecretID != "sec_deploy_name_hash" {
		t.Errorf("DEK secret ID mismatch")
	}

	if dek.Version != 1 {
		t.Errorf("DEK version: got %d, want 1", dek.Version)
	}

	if len(dek.KeyData) != 32 {
		t.Errorf("DEK key material length: got %d, want 32 bytes", len(dek.KeyData))
	}
}

// TestSecretsKeystore_DEKWrapUnwrap verifies DEK wrapping and unwrapping.
func TestSecretsKeystore_DEKWrapUnwrap(t *testing.T) {
	ks := NewSecretsKeystore()

	// Setup KEK
	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// Generate and wrap DEK
	dek, _ := ks.GenerateDEK("sec_test_secret", 1)
	dekPlaintext := make([]byte, len(dek.KeyData))
	copy(dekPlaintext, dek.KeyData)

	wrapped, err := ks.WrapDEK(dek)
	if err != nil {
		t.Fatalf("failed to wrap DEK: %v", err)
	}

	// Verify wrapped is base64-encoded
	if _, err := base64.StdEncoding.DecodeString(wrapped); err != nil {
		t.Errorf("wrapped DEK not valid base64: %v", err)
	}

	// Unwrap DEK (use secretID and version, not DEK ID)
	dekUnwrapped, err := ks.UnwrapDEK("KEK-1", wrapped, dek.SecretID, dek.Version)
	if err != nil {
		t.Fatalf("failed to unwrap DEK: %v", err)
	}

	// Verify DEK matches
	if !bytes.Equal(dekUnwrapped, dekPlaintext) {
		t.Error("unwrapped DEK does not match original")
	}
}

// TestSecretsKeystore_SecretEncryptDecrypt verifies end-to-end secret encryption/decryption.
func TestSecretsKeystore_SecretEncryptDecrypt(t *testing.T) {
	ks := NewSecretsKeystore()

	// Setup KEK
	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// Encrypt secret
	plaintext := []byte("super-secret-api-key-12345")
	wrapped, err := ks.EncryptSecret(
		"sec_deploy_name_hash",
		1,
		plaintext,
		"deploy-123",
		"api-worker",
		"production",
	)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}

	if wrapped.SecretID != "sec_deploy_name_hash" {
		t.Errorf("secret ID mismatch")
	}

	if wrapped.Lifecycle != "ACTIVE" {
		t.Errorf("lifecycle: got %s, want ACTIVE", wrapped.Lifecycle)
	}

	// Decrypt secret
	decrypted, err := ks.DecryptSecret(wrapped)
	if err != nil {
		t.Fatalf("failed to decrypt secret: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted secret does not match original")
	}
}

// TestSecretsKeystore_AADAuthentication verifies AEAD authentication with context binding.
// Changing context (deployment, workload, environment) must fail authentication.
func TestSecretsKeystore_AADAuthentication(t *testing.T) {
	ks := NewSecretsKeystore()

	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	plaintext := []byte("secret-value")
	wrapped, _ := ks.EncryptSecret(
		"sec_deploy_name_hash",
		1,
		plaintext,
		"deploy-123",
		"api-worker",
		"production",
	)

	// Test: Tamper with wrapped secret (modify ciphertext)
	ciphertext, _ := base64.StdEncoding.DecodeString(wrapped.Ciphertext)
	if len(ciphertext) > 0 {
		ciphertext[0] ^= 0xFF // flip bits
		wrapped.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	}

	_, err := ks.DecryptSecret(wrapped)
	if err == nil {
		t.Error("expected authentication failure on tampered ciphertext")
	}

	// Test: Tamper with AAD (modify deployment ID in context)
	wrapped2, _ := ks.EncryptSecret(
		"sec_deploy_name_hash",
		1,
		plaintext,
		"deploy-123",
		"api-worker",
		"production",
	)
	wrapped2.AAD = base64.StdEncoding.EncodeToString(
		[]byte(`{"secretId":"sec_deploy_name_hash","version":1,"deploymentId":"deploy-XXX","workloadId":"api-worker","environment":"production","keyId":"KEK-1"}`),
	)

	_, err = ks.DecryptSecret(wrapped2)
	if err == nil {
		t.Error("expected authentication failure on tampered AAD (deployment ID)")
	}
}

// TestSecretsKeystore_PlaintextNeverExposed verifies secrets are never exposed in plaintext during operations.
// Gate requirement: SEC-P0-A01-A03 / ADR 0011 §A
func TestSecretsKeystore_PlaintextNeverExposed(t *testing.T) {
	ks := NewSecretsKeystore()

	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	plaintext := []byte("super-secret-password")

	// Encrypt
	wrapped, _ := ks.EncryptSecret(
		"sec_deploy_name_hash",
		1,
		plaintext,
		"deploy-123",
		"api-worker",
		"production",
	)

	// Verify plaintext not in wrapped struct fields
	checkNotInString := func(s string, secret []byte) {
		if bytes.Contains([]byte(s), secret) {
			t.Errorf("plaintext found in string: %s", s)
		}
	}

	checkNotInString(wrapped.Ciphertext, plaintext)
	checkNotInString(wrapped.WrappedDEK, plaintext)
	checkNotInString(wrapped.Nonce, plaintext)
	checkNotInString(wrapped.AAD, plaintext)

	// Verify plaintext not in KeyData
	if bytes.Contains(kek.KeyData, plaintext) {
		t.Error("plaintext found in KEK")
	}
}

// TestSecretsKeystore_NoActiveKEK verifies fail-closed when KEK is unavailable.
func TestSecretsKeystore_NoActiveKEK(t *testing.T) {
	ks := NewSecretsKeystore() // No KEK set

	// Must fail
	_, err := ks.GenerateDEK("sec_test", 1)
	if err == nil {
		t.Error("expected error when generating DEK without active KEK")
	}

	_, err = ks.EncryptSecret("sec_test", 1, []byte("secret"), "deploy", "workload", "prod")
	if err == nil {
		t.Error("expected error when encrypting without active KEK")
	}
}

// TestSecretsKeystore_LifecycleStates verifies secret lifecycle tracking.
func TestSecretsKeystore_LifecycleStates(t *testing.T) {
	ks := NewSecretsKeystore()

	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// Encrypt creates ACTIVE secret
	wrapped, _ := ks.EncryptSecret(
		"sec_deploy_name_hash",
		1,
		[]byte("secret"),
		"deploy",
		"workload",
		"prod",
	)

	if wrapped.Lifecycle != "ACTIVE" {
		t.Errorf("initial lifecycle: got %s, want ACTIVE", wrapped.Lifecycle)
	}

	// Decrypt works for ACTIVE
	_, err := ks.DecryptSecret(wrapped)
	if err != nil {
		t.Errorf("decrypt failed for ACTIVE secret: %v", err)
	}

	// Decrypt fails for non-ACTIVE
	wrapped.Lifecycle = "REVOKED"
	_, err = ks.DecryptSecret(wrapped)
	if err == nil {
		t.Error("decrypt should fail for REVOKED secret")
	}

	wrapped.Lifecycle = "SUPERSEDED"
	_, err = ks.DecryptSecret(wrapped)
	if err == nil {
		t.Error("decrypt should fail for SUPERSEDED secret (not available for retrieval)")
	}

	wrapped.Lifecycle = "DELETED"
	_, err = ks.DecryptSecret(wrapped)
	if err == nil {
		t.Error("decrypt should fail for DELETED secret")
	}
}

// TestSecretsKeystore_ConcurrentOperations verifies thread-safety under concurrent encryption/decryption.
func TestSecretsKeystore_ConcurrentOperations(t *testing.T) {
	ks := NewSecretsKeystore()

	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// Spawn multiple goroutines encrypting different secrets
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()

			secretID := fmt.Sprintf("sec_test_%d", idx)
			plaintext := []byte(fmt.Sprintf("secret-value-%d", idx))

			wrapped, err := ks.EncryptSecret(secretID, 1, plaintext, "deploy", "workload", "prod")
			if err != nil {
				t.Errorf("encrypt failed for %s: %v", secretID, err)
				return
			}

			decrypted, err := ks.DecryptSecret(wrapped)
			if err != nil {
				t.Errorf("decrypt failed for %s: %v", secretID, err)
				return
			}

			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("decrypted mismatch for secret %s", secretID)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSecretsKeystore_InvalidKEKID verifies error handling for unknown KEK.
func TestSecretsKeystore_InvalidKEKID(t *testing.T) {
	ks := NewSecretsKeystore()

	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// Create a wrapped secret
	wrapped, _ := ks.EncryptSecret("sec_test", 1, []byte("secret"), "deploy", "workload", "prod")

	// Try to decrypt with wrong KEK ID
	wrapped.KeyID = "KEK-UNKNOWN"
	_, err := ks.DecryptSecret(wrapped)
	if err == nil {
		t.Error("expected error for unknown KEK ID")
	}
}

// TestSecretsKeystore_LargeSecrets verifies encryption works for large values.
func TestSecretsKeystore_LargeSecrets(t *testing.T) {
	ks := NewSecretsKeystore()

	kek, _ := ks.GenerateKEK("KEK-1")
	ks.SetActiveKEK(kek)

	// 10 MB secret
	largeSecret := make([]byte, 10*1024*1024)
	for i := range largeSecret {
		largeSecret[i] = byte(i % 256)
	}

	wrapped, err := ks.EncryptSecret("sec_large", 1, largeSecret, "deploy", "workload", "prod")
	if err != nil {
		t.Fatalf("failed to encrypt large secret: %v", err)
	}

	decrypted, err := ks.DecryptSecret(wrapped)
	if err != nil {
		t.Fatalf("failed to decrypt large secret: %v", err)
	}

	if !bytes.Equal(decrypted, largeSecret) {
		t.Error("large secret decryption mismatch")
	}
}
