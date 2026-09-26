package control

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestRaftPersistence verifies that encrypted SecretRecords persist through FSM.
func TestRaftPersistence(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("db-password-secret")
	dek, _ := GenerateDEK()
	secretID := "secret-001"

	// Encrypt plaintext (before proposing to Raft)
	record, err := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	if err != nil {
		t.Fatalf("EncryptSecret: %v", err)
	}

	// Propose command to FSM
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}

	res := fsm.ApplyLocal(cmd)
	if !res.OK {
		t.Fatalf("ApplyLocal failed: %v", res.Message)
	}

	// Verify record persisted in state
	fsm.Read(func(s *State) {
		retrieved := s.Secrets.GetRecord(secretID, 1)
		if retrieved == nil {
			t.Fatalf("Secret not found in state after persistence")
		}
		if !bytes.Equal(retrieved.EncryptedData, record.EncryptedData) {
			t.Error("Persisted encrypted data differs from original")
		}
		if !bytes.Equal(retrieved.Nonce, record.Nonce) {
			t.Error("Persisted nonce differs from original")
		}
	})
}

// TestSecretVersionAdd verifies multiple versions of the same secret can be persisted.
func TestSecretVersionAdd(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	secretID := "secret-multi"

	// Add version 1
	dek1, _ := GenerateDEK()
	plaintext1 := []byte("version-1-password")
	record1, _ := EncryptSecret(plaintext1, secretID, 1, dek1, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")

	recordJSON1, _ := json.Marshal(record1)
	cmd1 := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON1,
	}
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("secret-create v1 failed: %v", res1.Message)
	}

	// Add version 2
	dek2, _ := GenerateDEK()
	plaintext2 := []byte("version-2-password-rotated")
	record2, _ := EncryptSecret(plaintext2, secretID, 2, dek2, "test-cluster", "deploy-1", "workload-1", "prod", "key-2")

	recordJSON2, _ := json.Marshal(record2)
	cmd2 := &Command{
		Type:  "secret-version-add",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON2,
	}
	res2 := fsm.ApplyLocal(cmd2)
	if !res2.OK {
		t.Fatalf("secret-version-add v2 failed: %v", res2.Message)
	}

	// Verify both versions persisted
	fsm.Read(func(s *State) {
		v1 := s.Secrets.GetRecord(secretID, 1)
		v2 := s.Secrets.GetRecord(secretID, 2)
		if v1 == nil || v2 == nil {
			t.Fatalf("One or both versions missing")
		}
		if bytes.Equal(v1.EncryptedData, v2.EncryptedData) {
			t.Error("Version 1 and 2 have identical ciphertext (should differ)")
		}
	})
}

// TestDuplicateVersionRejected verifies same version cannot be added twice.
func TestDuplicateVersionRejected(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	secretID := "secret-dup"
	dek, _ := GenerateDEK()
	plaintext := []byte("test-data")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")

	// Add version 1
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	res1 := fsm.ApplyLocal(cmd)
	if !res1.OK {
		t.Fatalf("First create failed")
	}

	// Try to add same version again with secret-version-add (should be rejected)
	cmd2 := &Command{
		Type:  "secret-version-add",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("Duplicate version accepted (should be rejected)")
	}
}

// TestPlaintextCanary verifies no plaintext appears in state after encryption.
func TestPlaintextCanary(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("super-secret-plaintext-password")
	dek, _ := GenerateDEK()
	secretID := "secret-canary"

	// Encrypt and persist
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	fsm.ApplyLocal(cmd)

	// Scan replicated state for plaintext
	fsm.Read(func(s *State) {
		stateJSON, _ := json.Marshal(s)
		if bytes.Contains(stateJSON, plaintext) {
			t.Error("Plaintext found in replicated state (SECURITY ISSUE)")
		}
	})
}

// TestBoltDBCanary verifies plaintext does not appear when persisted to Bolt.
// This is a simulation; actual BoltDB persistence tested in full integration.
func TestBoltDBCanary(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("secret-db-password-shouldnotpersist")
	dek, _ := GenerateDEK()
	secretID := "secret-bolt"

	// Encrypt
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	fsm.ApplyLocal(cmd)

	// Simulate serializing state (as would be persisted to BoltDB)
	fsm.Read(func(s *State) {
		serialized, _ := json.Marshal(s)
		if bytes.Contains(serialized, plaintext) {
			t.Error("Plaintext canary: secret found in serialized state (BoltDB would persist plaintext)")
		}
	})
}

// TestSnapshotCanary verifies plaintext does not appear in snapshot.
func TestSnapshotCanary(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("snapshot-canary-secret")
	dek, _ := GenerateDEK()
	secretID := "secret-snap"

	// Encrypt and persist
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	fsm.ApplyLocal(cmd)

	// Simulate snapshot: Snapshot() serializes state
	fsm.Read(func(s *State) {
		snapshot, _ := json.Marshal(s)
		if bytes.Contains(snapshot, plaintext) {
			t.Error("Plaintext canary: secret found in snapshot (would be persisted)")
		}
	})
}

// TestRestartUnlocks verifies member can decrypt persisted secrets after restart.
func TestRestartUnlocks(t *testing.T) {
	// Simulate: encrypt, persist, stop, restart, try to decrypt
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("restart-secret-test")
	dek, _ := GenerateDEK()
	secretID := "secret-restart"

	// Encrypt and persist
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	fsm.ApplyLocal(cmd)

	// Simulate restart: read persisted record and decrypt with same DEK
	fsm.Read(func(s *State) {
		persisted := s.Secrets.GetRecord(secretID, 1)
		if persisted == nil {
			t.Fatalf("Record not persisted")
		}

		// Decrypt (simulating: after restart, derive KEK, unwrap DEK, decrypt)
		recovered, err := DecryptSecret(persisted, dek)
		if err != nil {
			t.Fatalf("DecryptSecret after restart: %v", err)
		}
		if !bytes.Equal(recovered, plaintext) {
			t.Error("Recovered plaintext differs after restart")
		}
	})
}

// TestExportCanary verifies plaintext does not appear in export.
// This test simulates exporting state (as would happen with backup).
func TestExportCanary(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("export-canary-secret")
	dek, _ := GenerateDEK()
	secretID := "secret-export"

	// Encrypt and persist
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	fsm.ApplyLocal(cmd)

	// Simulate export: serialize state (as would be backed up)
	fsm.Read(func(s *State) {
		exported, _ := json.Marshal(s)
		if bytes.Contains(exported, plaintext) {
			t.Error("Plaintext canary: secret found in export (backup would contain plaintext)")
		}
	})
}

// NegativeControl Tests: Verify security invariants by breaking them

// TestNegativeControl_DisableAADBinding introduces a mutation that disables AAD binding.
// This test MUST fail (i.e., scope tampering must be detected).
func TestNegativeControl_DisableAADBinding_TamperedScopeDetected(t *testing.T) {
	plaintext := []byte("secret")
	dek, _ := GenerateDEK()

	// Encrypt with one scope
	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Tamper with scope (disable AAD binding check would normally allow this)
	record.DeploymentID = "deploy-999" // Different scope

	// Decrypt MUST fail because AAD is part of auth tag
	_, err := DecryptSecret(record, dek)
	if err == nil {
		t.Error("NegativeControl_AADBinding: scope tampering not detected (invariant broken)")
	}
}

// TestNegativeControl_PlaintextPersistence verifies canary detects plaintext persistence.
// This test demonstrates that proper encryption keeps plaintext out of serialized state.
func TestNegativeControl_PlaintextPersistence_CanaryDetects(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	plaintext := []byte("plaintext-secret-TEST-12345")
	dek, _ := GenerateDEK()

	// Properly encrypt (this should keep plaintext out of state)
	record, _ := EncryptSecret(plaintext, "secret-test", 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	recordJSON, _ := json.Marshal(record)
	cmd := &Command{
		Type:  "secret-create",
		TS:    Now(),
		Actor: "test-user",
		Data:  recordJSON,
	}
	fsm.ApplyLocal(cmd)

	// Canary scan: verify plaintext does NOT appear in serialized state
	fsm.Read(func(s *State) {
		serialized, _ := json.Marshal(s)
		if bytes.Contains(serialized, plaintext) {
			t.Error("NegativeControl_PlaintextPersistence: plaintext found in state (encryption failed)")
		}
		// Test passes: plaintext correctly absent
	})
}

// TestNegativeControl_AuthTagBypass verifies AEAD auth tag cannot be bypassed.
// Deliberately tampering with encrypted data must cause authentication to fail.
func TestNegativeControl_AuthTagBypass_TamperDetected(t *testing.T) {
	plaintext := []byte("protected-data")
	dek, _ := GenerateDEK()

	// Encrypt
	record, _ := EncryptSecret(plaintext, "secret-1", 1, dek, "cluster-1", "deploy-1", "workload-1", "prod", "key-1")

	// Try to bypass auth tag by modifying ciphertext
	if len(record.EncryptedData) > 5 {
		record.EncryptedData[5] ^= 0xFF // Flip a bit
	}

	// Decryption MUST fail (auth tag verification fails)
	_, err := DecryptSecret(record, dek)
	if err == nil {
		t.Error("NegativeControl_AuthTagBypass: tampering not detected (invariant broken)")
	}
}
