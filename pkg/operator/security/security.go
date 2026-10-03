// Package security implements operator security auditing and compliance verification
// for the Decentralized.Host operator qualification program.
//
// The security audit framework validates operator identity, key management,
// network security, data isolation, and compliance requirements.
package security

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"sync"
	"time"

	"decentralized.host/pkg/identity"
)

// ComplianceLevel represents the security compliance tier
type ComplianceLevel string

const (
	UNVERIFIED ComplianceLevel = "UNVERIFIED"
	BASIC      ComplianceLevel = "BASIC"
	STANDARD   ComplianceLevel = "STANDARD"
	ADVANCED   ComplianceLevel = "ADVANCED"
	CERTIFIED  ComplianceLevel = "CERTIFIED"
)

// AuditCheckType defines types of security checks
type AuditCheckType string

const (
	IDENTITY_VERIFICATION       AuditCheckType = "IDENTITY_VERIFICATION"
	KEY_MANAGEMENT              AuditCheckType = "KEY_MANAGEMENT"
	NETWORK_SECURITY            AuditCheckType = "NETWORK_SECURITY"
	DATA_ISOLATION              AuditCheckType = "DATA_ISOLATION"
	TLS_CERTIFICATE_VALIDATION  AuditCheckType = "TLS_CERTIFICATE_VALIDATION"
	FIREWALL_RULES              AuditCheckType = "FIREWALL_RULES"
	ENCRYPTION_AT_REST          AuditCheckType = "ENCRYPTION_AT_REST"
	ENCRYPTION_IN_TRANSIT       AuditCheckType = "ENCRYPTION_IN_TRANSIT"
	ACCESS_CONTROL              AuditCheckType = "ACCESS_CONTROL"
	AUDIT_LOGGING               AuditCheckType = "AUDIT_LOGGING"
)

// AuditResult represents the result of a security check
type AuditResult struct {
	CheckType    AuditCheckType `json:"check_type"`
	Passed       bool           `json:"passed"`
	Severity     string         `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW, INFO
	Message      string         `json:"message"`
	Details      string         `json:"details,omitempty"`
	CheckedAt    time.Time      `json:"checked_at"`
	Remediation  string         `json:"remediation,omitempty"`
}

// OperatorAudit tracks security audit results for an operator
type OperatorAudit struct {
	OperatorID           string              `json:"operator_id"`
	NodeID               string              `json:"node_id"`
	ComplianceLevel      ComplianceLevel     `json:"compliance_level"`
	IdentityVerified     bool                `json:"identity_verified"`
	KeyManagementOK      bool                `json:"key_management_ok"`
	NetworkSecurityOK    bool                `json:"network_security_ok"`
	DataIsolationOK      bool                `json:"data_isolation_ok"`
	TLSCertificateOK     bool                `json:"tls_certificate_ok"`
	FirewallConfigOK     bool                `json:"firewall_config_ok"`
	EncryptionAtRestOK   bool                `json:"encryption_at_rest_ok"`
	EncryptionInTransitOK bool               `json:"encryption_in_transit_ok"`
	AccessControlOK      bool                `json:"access_control_ok"`
	AuditLoggingOK       bool                `json:"audit_logging_ok"`
	OverallScore         int16               `json:"overall_score"` // 0-100
	AuditResults         []AuditResult       `json:"audit_results"`
	FirstAuditTime       time.Time           `json:"first_audit_time"`
	LastAuditTime        time.Time           `json:"last_audit_time"`
	CertificationTime    *time.Time          `json:"certification_time,omitempty"`
	ExpirationTime       *time.Time          `json:"expiration_time,omitempty"`
}

// SecurityManager manages security audits
type SecurityManager struct {
	mu     sync.RWMutex
	audits map[string]*OperatorAudit
}

// NewSecurityManager creates a new security manager
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		audits: make(map[string]*OperatorAudit),
	}
}

// InitiateAudit creates a new audit record
func (sm *SecurityManager) InitiateAudit(operatorID string, nodeID string) (*OperatorAudit, error) {
	if operatorID == "" || nodeID == "" {
		return nil, errors.New("operator ID and node ID cannot be empty")
	}

	if !identity.IDPattern.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid node ID format: %s", nodeID)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	audit := &OperatorAudit{
		OperatorID:      operatorID,
		NodeID:          nodeID,
		ComplianceLevel: UNVERIFIED,
		FirstAuditTime:  now,
		LastAuditTime:   now,
		AuditResults:    []AuditResult{},
	}

	sm.audits[operatorID] = audit
	return audit, nil
}

// GetAudit retrieves an operator's audit record
func (sm *SecurityManager) GetAudit(operatorID string) (*OperatorAudit, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	audit, exists := sm.audits[operatorID]
	if !exists {
		return nil, fmt.Errorf("no audit found for operator %s", operatorID)
	}
	return audit, nil
}

// RecordCheck records the result of a security check
func (sm *SecurityManager) RecordCheck(operatorID string, checkType AuditCheckType, passed bool, severity string, message string, remediation string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	audit, exists := sm.audits[operatorID]
	if !exists {
		return fmt.Errorf("no audit found for operator %s", operatorID)
	}

	result := AuditResult{
		CheckType:   checkType,
		Passed:      passed,
		Severity:    severity,
		Message:     message,
		CheckedAt:   time.Now(),
		Remediation: remediation,
	}

	audit.AuditResults = append(audit.AuditResults, result)

	// Update corresponding field
	if passed {
		switch checkType {
		case IDENTITY_VERIFICATION:
			audit.IdentityVerified = true
		case KEY_MANAGEMENT:
			audit.KeyManagementOK = true
		case NETWORK_SECURITY:
			audit.NetworkSecurityOK = true
		case DATA_ISOLATION:
			audit.DataIsolationOK = true
		case TLS_CERTIFICATE_VALIDATION:
			audit.TLSCertificateOK = true
		case FIREWALL_RULES:
			audit.FirewallConfigOK = true
		case ENCRYPTION_AT_REST:
			audit.EncryptionAtRestOK = true
		case ENCRYPTION_IN_TRANSIT:
			audit.EncryptionInTransitOK = true
		case ACCESS_CONTROL:
			audit.AccessControlOK = true
		case AUDIT_LOGGING:
			audit.AuditLoggingOK = true
		}
	}

	audit.LastAuditTime = time.Now()
	sm.updateComplianceLevel(audit)

	return nil
}

// updateComplianceLevel updates compliance level based on check results
func (sm *SecurityManager) updateComplianceLevel(audit *OperatorAudit) {
	// Count passed checks (required checks only)
	requiredChecks := []bool{
		audit.IdentityVerified,
		audit.KeyManagementOK,
		audit.NetworkSecurityOK,
		audit.DataIsolationOK,
	}

	passedCount := 0
	for _, passed := range requiredChecks {
		if passed {
			passedCount++
		}
	}

	// Determine compliance level
	switch passedCount {
	case 0:
		audit.ComplianceLevel = UNVERIFIED
		audit.OverallScore = 0
	case 1:
		audit.ComplianceLevel = BASIC
		audit.OverallScore = 25
	case 2:
		audit.ComplianceLevel = STANDARD
		audit.OverallScore = 50
	case 3:
		audit.ComplianceLevel = ADVANCED
		audit.OverallScore = 75
	case 4:
		// Check additional security measures for CERTIFIED
		additionalChecks := []bool{
			audit.TLSCertificateOK,
			audit.FirewallConfigOK,
			audit.EncryptionAtRestOK,
			audit.EncryptionInTransitOK,
			audit.AccessControlOK,
			audit.AuditLoggingOK,
		}

		additionalPassed := 0
		for _, passed := range additionalChecks {
			if passed {
				additionalPassed++
			}
		}

		if additionalPassed >= 4 {
			audit.ComplianceLevel = CERTIFIED
			audit.OverallScore = 100
		} else {
			audit.ComplianceLevel = ADVANCED
			audit.OverallScore = 75 + int16((additionalPassed*20)/6)
		}
	}
}

// VerifyOperatorIdentity verifies operator identity against their Ed25519 key
func (sm *SecurityManager) VerifyOperatorIdentity(operatorID string, pub ed25519.PublicKey, signature []byte, message []byte) error {
	if !ed25519.Verify(pub, message, signature) {
		return errors.New("operator identity verification failed: invalid signature")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	audit, exists := sm.audits[operatorID]
	if !exists {
		return fmt.Errorf("no audit found for operator %s", operatorID)
	}

	audit.IdentityVerified = true
	audit.LastAuditTime = time.Now()
	sm.updateComplianceLevel(audit)

	return nil
}

// CertifyOperator marks an operator as certified after successful audit
func (sm *SecurityManager) CertifyOperator(operatorID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	audit, exists := sm.audits[operatorID]
	if !exists {
		return fmt.Errorf("no audit found for operator %s", operatorID)
	}

	if audit.ComplianceLevel != CERTIFIED {
		return fmt.Errorf("operator must be CERTIFIED to receive security certification, current: %s", audit.ComplianceLevel)
	}

	now := time.Now()
	audit.CertificationTime = &now
	// Certification valid for 1 year
	expiration := now.AddDate(1, 0, 0)
	audit.ExpirationTime = &expiration

	return nil
}

// GetAuditStatus returns audit status for an operator
type AuditStatus struct {
	OperatorID      string
	ComplianceLevel ComplianceLevel
	OverallScore    int16
	PassedChecks    int
	FailedChecks    int
	IsCertified     bool
	ExpiresAt       *time.Time
}

// GetStatus returns audit status
func (sm *SecurityManager) GetStatus(operatorID string) (*AuditStatus, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	audit, exists := sm.audits[operatorID]
	if !exists {
		return nil, fmt.Errorf("no audit found for operator %s", operatorID)
	}

	passedCount := 0
	failedCount := 0
	for _, result := range audit.AuditResults {
		if result.Passed {
			passedCount++
		} else {
			failedCount++
		}
	}

	return &AuditStatus{
		OperatorID:      operatorID,
		ComplianceLevel: audit.ComplianceLevel,
		OverallScore:    audit.OverallScore,
		PassedChecks:    passedCount,
		FailedChecks:    failedCount,
		IsCertified:     audit.CertificationTime != nil,
		ExpiresAt:       audit.ExpirationTime,
	}, nil
}

// ListAudits returns all audit records
func (sm *SecurityManager) ListAudits() []*OperatorAudit {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	audits := make([]*OperatorAudit, 0, len(sm.audits))
	for _, audit := range sm.audits {
		audits = append(audits, audit)
	}
	return audits
}

// SecuritySummary provides aggregate security information
type SecuritySummary struct {
	TotalAudits           int
	CertifiedOperators    int
	AdvancedOperators     int
	StandardOperators     int
	BasicOperators        int
	UnverifiedOperators   int
	AverageComplianceScore int16
	CriticalFindings      int
	HighFindings          int
}

// GetSecuritySummary returns overall security statistics
func (sm *SecurityManager) GetSecuritySummary() SecuritySummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := SecuritySummary{
		TotalAudits: len(sm.audits),
	}

	var totalScore int32
	for _, audit := range sm.audits {
		totalScore += int32(audit.OverallScore)

		switch audit.ComplianceLevel {
		case CERTIFIED:
			summary.CertifiedOperators++
		case ADVANCED:
			summary.AdvancedOperators++
		case STANDARD:
			summary.StandardOperators++
		case BASIC:
			summary.BasicOperators++
		case UNVERIFIED:
			summary.UnverifiedOperators++
		}

		// Count findings by severity
		for _, result := range audit.AuditResults {
			if !result.Passed {
				switch result.Severity {
				case "CRITICAL":
					summary.CriticalFindings++
				case "HIGH":
					summary.HighFindings++
				}
			}
		}
	}

	if summary.TotalAudits > 0 {
		summary.AverageComplianceScore = int16(totalScore / int32(summary.TotalAudits))
	}

	return summary
}

// RenewCertification extends an operator's security certification
func (sm *SecurityManager) RenewCertification(operatorID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	audit, exists := sm.audits[operatorID]
	if !exists {
		return fmt.Errorf("no audit found for operator %s", operatorID)
	}

	if audit.ComplianceLevel != CERTIFIED {
		return fmt.Errorf("only CERTIFIED operators can renew certification")
	}

	if audit.ExpirationTime == nil || time.Now().Before(*audit.ExpirationTime) {
		return errors.New("certification still valid, renewal not yet allowed")
	}

	// Renew for another year
	now := time.Now()
	audit.CertificationTime = &now
	expiration := now.AddDate(1, 0, 0)
	audit.ExpirationTime = &expiration

	return nil
}
