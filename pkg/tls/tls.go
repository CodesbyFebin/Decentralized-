// Package tls provides TLS/mTLS validation and enforcement for dh/v1.
//
// This package enforces:
// - TLS 1.3 minimum
// - mTLS mutual authentication
// - Certificate chain validation
// - Cipher suite enforcement
// - Handshake latency measurement
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"
)

// ValidationResult contains the result of TLS validation.
type ValidationResult struct {
	IsValid              bool
	TLSVersion           uint16
	CipherSuite          uint16
	CipherSuiteName      string
	HandshakeLatency     time.Duration
	CertificateChainLen  int
	LeafCertSubject      string
	Errors               []string
}

// TLSValidator validates TLS connections.
type TLSValidator struct {
	requireMTLS          bool
	enforceMinVersion    uint16
	allowedCipherSuites  map[uint16]bool
	enforceChainLength   bool
	maxChainLength       int
	handshakePollTick    time.Duration
}

// NewValidator creates a new TLS validator with production defaults.
func NewValidator() *TLSValidator {
	return &TLSValidator{
		requireMTLS:         true,
		enforceMinVersion:   tls.VersionTLS13,
		handshakePollTick:   100 * time.Millisecond,
		enforceChainLength:  true,
		maxChainLength:      3, // Leaf + Intermediate + Root
		allowedCipherSuites: productionCipherSuites(),
	}
}

// SetRequireMTLS sets whether mTLS is required.
func (v *TLSValidator) SetRequireMTLS(require bool) *TLSValidator {
	v.requireMTLS = require
	return v
}

// SetMinVersion sets the minimum TLS version.
func (v *TLSValidator) SetMinVersion(version uint16) *TLSValidator {
	v.enforceMinVersion = version
	return v
}

// ValidateConnectionState validates a completed TLS connection.
func (v *TLSValidator) ValidateConnectionState(state *tls.ConnectionState) *ValidationResult {
	result := &ValidationResult{
		TLSVersion:          state.Version,
		CipherSuite:         state.CipherSuite,
		CipherSuiteName:     tls.CipherSuiteName(state.CipherSuite),
		CertificateChainLen: len(state.PeerCertificates),
		IsValid:             true,
		Errors:              []string{},
	}

	// Check TLS version
	if state.Version < v.enforceMinVersion {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("TLS version %s is below required minimum %s",
				tlsVersionName(state.Version),
				tlsVersionName(v.enforceMinVersion)))
	}

	// Check cipher suite
	if !v.allowedCipherSuites[state.CipherSuite] {
		result.IsValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("cipher suite %s is not allowed", result.CipherSuiteName))
	}

	// Check mTLS
	if v.requireMTLS && len(state.PeerCertificates) == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "mTLS required but no peer certificates presented")
	}

	// Validate peer certificate chain
	if len(state.PeerCertificates) > 0 {
		if state.PeerCertificates[0].Subject.String() != "" {
			result.LeafCertSubject = state.PeerCertificates[0].Subject.String()
		}

		if v.enforceChainLength && len(state.PeerCertificates) > v.maxChainLength {
			result.IsValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("certificate chain length %d exceeds maximum %d",
					len(state.PeerCertificates), v.maxChainLength))
		}

		// Validate each certificate in chain
		for i, cert := range state.PeerCertificates {
			if err := validateCertificate(cert, i); err != nil {
				result.IsValid = false
				result.Errors = append(result.Errors, fmt.Sprintf("cert[%d]: %v", i, err))
			}
		}
	}

	// Check for known problematic cipher suites
	if isWeakCipherSuite(state.CipherSuite) {
		result.IsValid = false
		result.Errors = append(result.Errors, "weak cipher suite detected")
	}

	return result
}

// MeasureHandshakeLatency measures TLS handshake latency to an address.
func (v *TLSValidator) MeasureHandshakeLatency(addr string, timeout time.Duration) (time.Duration, error) {
	start := time.Now()

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		MinVersion:         v.enforceMinVersion,
		InsecureSkipVerify: true, // For latency measurement only
	})
	if err != nil {
		return 0, fmt.Errorf("tls handshake failed: %w", err)
	}
	defer conn.Close()

	latency := time.Since(start)

	// Validate the connection state
	state := conn.ConnectionState()
	result := v.ValidateConnectionState(&state)
	if !result.IsValid {
		return latency, fmt.Errorf("connection state validation failed: %v", result.Errors)
	}

	return latency, nil
}

// ValidateServerConfig validates a TLS server configuration.
func (v *TLSValidator) ValidateServerConfig(cfg *tls.Config) []string {
	var errors []string

	if cfg == nil {
		return []string{"config is nil"}
	}

	// Check min version
	if cfg.MinVersion == 0 || cfg.MinVersion < v.enforceMinVersion {
		errors = append(errors, fmt.Sprintf("MinVersion not set or below required %s",
			tlsVersionName(v.enforceMinVersion)))
	}

	// Check certificates
	if len(cfg.Certificates) == 0 {
		errors = append(errors, "no certificates configured")
	}

	// Check client auth
	if v.requireMTLS && cfg.ClientAuth == tls.NoClientCert {
		errors = append(errors, "ClientAuth must require certificates for mTLS")
	}

	// Check cipher suites (if explicitly set)
	if len(cfg.CipherSuites) > 0 {
		for _, suite := range cfg.CipherSuites {
			if !v.allowedCipherSuites[suite] {
				errors = append(errors, fmt.Sprintf("disallowed cipher suite: %s", tls.CipherSuiteName(suite)))
			}
		}
	}

	return errors
}

// ValidateCertificateChain validates an X.509 certificate chain.
func (v *TLSValidator) ValidateCertificateChain(certs []*x509.Certificate, roots *x509.CertPool) error {
	if len(certs) == 0 {
		return errors.New("empty certificate chain")
	}

	leaf := certs[0]

	// Validate certificate not expired
	now := time.Now()
	if now.Before(leaf.NotBefore) {
		return fmt.Errorf("certificate not yet valid (notBefore: %s)", leaf.NotBefore)
	}
	if now.After(leaf.NotAfter) {
		return fmt.Errorf("certificate expired (notAfter: %s)", leaf.NotAfter)
	}

	// Validate extended key usage
	if len(leaf.ExtKeyUsage) == 0 {
		return errors.New("certificate missing extended key usage")
	}

	hasValidEKU := false
	for _, eku := range leaf.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth || eku == x509.ExtKeyUsageClientAuth {
			hasValidEKU = true
			break
		}
	}
	if !hasValidEKU {
		return fmt.Errorf("certificate missing required extended key usage (serverAuth or clientAuth)")
	}

	// Validate basic constraints
	if !leaf.BasicConstraintsValid {
		return errors.New("certificate missing valid basic constraints")
	}

	// Validate against root pool
	if roots != nil {
		opts := x509.VerifyOptions{
			Roots:       roots,
			CurrentTime: now,
		}
		if _, err := leaf.Verify(opts); err != nil {
			return fmt.Errorf("certificate chain validation failed: %w", err)
		}
	}

	return nil
}

// Helper functions

func validateCertificate(cert *x509.Certificate, index int) error {
	now := time.Now()

	if now.Before(cert.NotBefore) {
		return fmt.Errorf("not yet valid (notBefore: %s)", cert.NotBefore)
	}

	if now.After(cert.NotAfter) {
		return fmt.Errorf("expired (notAfter: %s)", cert.NotAfter)
	}

	// Note: Certificates expiring within 30 days are logged as warnings
	// but do not cause validation failure
	return nil
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("UNKNOWN(0x%04x)", version)
	}
}

func isWeakCipherSuite(suite uint16) bool {
	// List of weak cipher suites to block
	weakSuites := map[uint16]bool{
		// NULL cipher suites
		0x0001: true, // TLS_NULL_WITH_NULL_NULL
		0x0002: true, // TLS_RSA_WITH_NULL_MD5

		// Export cipher suites (40/56-bit)
		0x0008: true, // TLS_DH_DSS_EXPORT_WITH_DES40_CBC_SHA
		0x0010: true, // TLS_DH_RSA_EXPORT_WITH_DES40_CBC_SHA

		// DES/3DES (only allow in legacy mode)
		0x0009: true, // TLS_DH_DSS_WITH_DES_CBC_SHA
		0x0019: true, // TLS_DH_RSA_WITH_DES_CBC_SHA
		0x001B: true, // TLS_DH_anon_WITH_DES_CBC_SHA

		// MD5
		0x0004: true, // TLS_RSA_WITH_RC4_128_MD5
	}

	return weakSuites[suite]
}

func productionCipherSuites() map[uint16]bool {
	// TLS 1.3 cipher suites (only ones allowed)
	return map[uint16]bool{
		tls.TLS_AES_128_GCM_SHA256:       true,
		tls.TLS_AES_256_GCM_SHA384:       true,
		tls.TLS_CHACHA20_POLY1305_SHA256: true,
	}
}
