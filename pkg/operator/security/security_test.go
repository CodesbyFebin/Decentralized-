package security

import (
	"testing"
)

func TestInitiateAudit(t *testing.T) {
	sm := NewSecurityManager()

	audit, err := sm.InitiateAudit("op_001", "dh1abcdefghijklmnopqrstuv")
	if err != nil {
		t.Fatalf("InitiateAudit() error = %v", err)
	}

	if audit.ComplianceLevel != UNVERIFIED {
		t.Errorf("expected UNVERIFIED level, got %s", audit.ComplianceLevel)
	}
}

func TestRecordCheck(t *testing.T) {
	sm := NewSecurityManager()
	operatorID := "op_001"

	sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

	// Record passing check
	err := sm.RecordCheck(operatorID, IDENTITY_VERIFICATION, true, "INFO", "Identity verified", "")
	if err != nil {
		t.Fatalf("RecordCheck() error = %v", err)
	}

	audit, _ := sm.GetAudit(operatorID)
	if !audit.IdentityVerified {
		t.Error("expected IdentityVerified to be true")
	}
}

func TestComplianceLevelProgression(t *testing.T) {
	sm := NewSecurityManager()
	operatorID := "op_001"

	sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

	// Record required checks
	checks := []struct {
		checkType AuditCheckType
		passed    bool
	}{
		{IDENTITY_VERIFICATION, true},
		{KEY_MANAGEMENT, true},
		{NETWORK_SECURITY, true},
		{DATA_ISOLATION, true},
		{TLS_CERTIFICATE_VALIDATION, true},
		{FIREWALL_RULES, true},
		{ENCRYPTION_AT_REST, true},
		{ENCRYPTION_IN_TRANSIT, true},
		{ACCESS_CONTROL, true},
		{AUDIT_LOGGING, true},
	}

	for _, check := range checks {
		sm.RecordCheck(operatorID, check.checkType, check.passed, "INFO", "Passed", "")
	}

	audit, _ := sm.GetAudit(operatorID)
	if audit.ComplianceLevel != CERTIFIED {
		t.Errorf("expected CERTIFIED level, got %s", audit.ComplianceLevel)
	}
}

func TestCertification(t *testing.T) {
	sm := NewSecurityManager()
	operatorID := "op_001"

	sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

	// Try to certify without meeting requirements (should fail)
	err := sm.CertifyOperator(operatorID)
	if err == nil {
		t.Error("expected error certifying non-CERTIFIED operator")
	}

	// Record all required checks
	requiredChecks := []AuditCheckType{
		IDENTITY_VERIFICATION,
		KEY_MANAGEMENT,
		NETWORK_SECURITY,
		DATA_ISOLATION,
	}

	for _, check := range requiredChecks {
		sm.RecordCheck(operatorID, check, true, "INFO", "Passed", "")
	}

	// Record additional checks for CERTIFIED
	additionalChecks := []AuditCheckType{
		TLS_CERTIFICATE_VALIDATION,
		FIREWALL_RULES,
		ENCRYPTION_AT_REST,
		ENCRYPTION_IN_TRANSIT,
	}

	for _, check := range additionalChecks {
		sm.RecordCheck(operatorID, check, true, "INFO", "Passed", "")
	}

	// Now certify should work
	err = sm.CertifyOperator(operatorID)
	if err != nil {
		t.Fatalf("CertifyOperator() error = %v", err)
	}

	audit, _ := sm.GetAudit(operatorID)
	if audit.CertificationTime == nil {
		t.Error("expected CertificationTime to be set")
	}

	if audit.ExpirationTime == nil {
		t.Error("expected ExpirationTime to be set")
	}
}

func TestAuditStatus(t *testing.T) {
	sm := NewSecurityManager()
	operatorID := "op_001"

	sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

	sm.RecordCheck(operatorID, IDENTITY_VERIFICATION, true, "INFO", "Passed", "")
	sm.RecordCheck(operatorID, KEY_MANAGEMENT, false, "HIGH", "Failed", "Fix key storage")

	status, err := sm.GetStatus(operatorID)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.PassedChecks != 1 {
		t.Errorf("expected 1 passed check, got %d", status.PassedChecks)
	}

	if status.FailedChecks != 1 {
		t.Errorf("expected 1 failed check, got %d", status.FailedChecks)
	}
}

func TestSecuritySummary(t *testing.T) {
	sm := NewSecurityManager()

	// Create multiple audits with different compliance levels
	for i := 0; i < 3; i++ {
		operatorID := "op_00" + string(rune('1'+i))
		sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

		// Vary compliance levels by recording different checks
		if i == 0 {
			// CERTIFIED
			checks := []AuditCheckType{
				IDENTITY_VERIFICATION, KEY_MANAGEMENT, NETWORK_SECURITY, DATA_ISOLATION,
				TLS_CERTIFICATE_VALIDATION, FIREWALL_RULES, ENCRYPTION_AT_REST, ENCRYPTION_IN_TRANSIT,
			}
			for _, check := range checks {
				sm.RecordCheck(operatorID, check, true, "INFO", "Passed", "")
			}
		} else if i == 1 {
			// STANDARD (2 required checks)
			sm.RecordCheck(operatorID, IDENTITY_VERIFICATION, true, "INFO", "Passed", "")
			sm.RecordCheck(operatorID, KEY_MANAGEMENT, true, "INFO", "Passed", "")
		} else {
			// UNVERIFIED (no checks)
		}
	}

	summary := sm.GetSecuritySummary()
	if summary.TotalAudits != 3 {
		t.Errorf("expected 3 audits, got %d", summary.TotalAudits)
	}

	if summary.CertifiedOperators != 1 {
		t.Errorf("expected 1 certified, got %d", summary.CertifiedOperators)
	}
}

func TestListAudits(t *testing.T) {
	sm := NewSecurityManager()

	count := 10
	for i := 0; i < count; i++ {
		operatorID := "op_" + string(rune('0'+i))
		sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")
	}

	audits := sm.ListAudits()
	if len(audits) != count {
		t.Errorf("expected %d audits, got %d", count, len(audits))
	}
}

func TestRenewCertification(t *testing.T) {
	sm := NewSecurityManager()
	operatorID := "op_001"

	sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

	// Record all required checks
	requiredChecks := []AuditCheckType{
		IDENTITY_VERIFICATION, KEY_MANAGEMENT, NETWORK_SECURITY, DATA_ISOLATION,
		TLS_CERTIFICATE_VALIDATION, FIREWALL_RULES, ENCRYPTION_AT_REST, ENCRYPTION_IN_TRANSIT,
	}

	for _, check := range requiredChecks {
		sm.RecordCheck(operatorID, check, true, "INFO", "Passed", "")
	}

	// Certify
	sm.CertifyOperator(operatorID)

	audit, _ := sm.GetAudit(operatorID)
	oldExpiration := audit.ExpirationTime

	// Try to renew before expiration (should fail)
	err := sm.RenewCertification(operatorID)
	if err == nil {
		t.Error("expected error renewing before expiration")
	}

	// Move expiration to past
	pastTime := sm.audits[operatorID].ExpirationTime
	*pastTime = pastTime.AddDate(-1, 0, 0)

	// Now renew should work
	err = sm.RenewCertification(operatorID)
	if err != nil {
		t.Fatalf("RenewCertification() error = %v", err)
	}

	audit, _ = sm.GetAudit(operatorID)
	if audit.ExpirationTime.Before(*oldExpiration) {
		t.Error("expected new expiration to be later than old")
	}
}

func TestFindingsSeverity(t *testing.T) {
	sm := NewSecurityManager()
	operatorID := "op_001"

	sm.InitiateAudit(operatorID, "dh1abcdefghijklmnopqrstuv")

	// Record checks with different severities
	sm.RecordCheck(operatorID, IDENTITY_VERIFICATION, false, "CRITICAL", "Critical finding", "Fix immediately")
	sm.RecordCheck(operatorID, KEY_MANAGEMENT, false, "HIGH", "High finding", "Fix soon")

	summary := sm.GetSecuritySummary()
	if summary.CriticalFindings != 1 {
		t.Errorf("expected 1 critical finding, got %d", summary.CriticalFindings)
	}

	if summary.HighFindings != 1 {
		t.Errorf("expected 1 high finding, got %d", summary.HighFindings)
	}
}
