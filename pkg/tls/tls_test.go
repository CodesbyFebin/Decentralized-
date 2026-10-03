package tls

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"
)

func TestValidatorCreation(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}

	if v.enforceMinVersion != tls.VersionTLS13 {
		t.Errorf("enforceMinVersion = %d, want %d", v.enforceMinVersion, tls.VersionTLS13)
	}

	if !v.requireMTLS {
		t.Error("requireMTLS should be true by default")
	}
}

func TestValidatorChaining(t *testing.T) {
	v := NewValidator().
		SetRequireMTLS(false).
		SetMinVersion(tls.VersionTLS13)

	if v.requireMTLS {
		t.Error("SetRequireMTLS(false) failed")
	}
}

func TestValidateConnectionState_ValidTLS13(t *testing.T) {
	v := NewValidator()
	state := &tls.ConnectionState{
		Version:     tls.VersionTLS13,
		CipherSuite: tls.TLS_AES_256_GCM_SHA384,
		PeerCertificates: []*x509.Certificate{
			{
				Subject:                pkix.Name{CommonName: "test.example.com"},
				NotBefore:              time.Now().Add(-time.Hour),
				NotAfter:               time.Now().Add(24 * time.Hour),
				ExtKeyUsage:            []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				BasicConstraintsValid:  true,
			},
		},
	}

	result := v.ValidateConnectionState(state)
	if !result.IsValid {
		t.Errorf("validation failed: %v", result.Errors)
	}

	if result.TLSVersion != tls.VersionTLS13 {
		t.Errorf("TLSVersion = %d, want %d", result.TLSVersion, tls.VersionTLS13)
	}

	if result.CipherSuite != tls.TLS_AES_256_GCM_SHA384 {
		t.Errorf("CipherSuite = %d, want %d", result.CipherSuite, tls.TLS_AES_256_GCM_SHA384)
	}
}

func TestValidateConnectionState_InvalidTLSVersion(t *testing.T) {
	v := NewValidator()
	state := &tls.ConnectionState{
		Version:     tls.VersionTLS12,
		CipherSuite: tls.TLS_AES_256_GCM_SHA384,
	}

	result := v.ValidateConnectionState(state)
	if result.IsValid {
		t.Error("should reject TLS 1.2 when 1.3 required")
	}

	if len(result.Errors) == 0 {
		t.Error("should have error messages")
	}
}

func TestValidateConnectionState_NoCerts_MTLSRequired(t *testing.T) {
	v := NewValidator().SetRequireMTLS(true)
	state := &tls.ConnectionState{
		Version:          tls.VersionTLS13,
		CipherSuite:      tls.TLS_AES_256_GCM_SHA384,
		PeerCertificates: nil,
	}

	result := v.ValidateConnectionState(state)
	if result.IsValid {
		t.Error("should require peer certificates for mTLS")
	}
}

func TestValidateConnectionState_BadCipherSuite(t *testing.T) {
	v := NewValidator()
	state := &tls.ConnectionState{
		Version:     tls.VersionTLS13,
		CipherSuite: 0x0001, // Invalid weak suite
	}

	result := v.ValidateConnectionState(state)
	if result.IsValid {
		t.Error("should reject invalid cipher suite")
	}
}

func TestValidateServerConfig_Valid(t *testing.T) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAnyClientCert,
		Certificates: []tls.Certificate{
			{},
		},
	}

	v := NewValidator()
	errs := v.ValidateServerConfig(cfg)
	if len(errs) > 0 {
		t.Errorf("valid config should have no errors: %v", errs)
	}
}

func TestValidateServerConfig_NoMinVersion(t *testing.T) {
	cfg := &tls.Config{
		ClientAuth: tls.RequireAnyClientCert,
		Certificates: []tls.Certificate{
			{},
		},
	}

	v := NewValidator()
	errs := v.ValidateServerConfig(cfg)
	if len(errs) == 0 {
		t.Error("should error when MinVersion not set")
	}
}

func TestValidateServerConfig_NoCerts(t *testing.T) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	v := NewValidator()
	errs := v.ValidateServerConfig(cfg)
	if len(errs) == 0 {
		t.Error("should error when no certificates configured")
	}
}

func TestValidateServerConfig_NoClientAuth_MTLSRequired(t *testing.T) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.NoClientCert,
		Certificates: []tls.Certificate{
			{},
		},
	}

	v := NewValidator().SetRequireMTLS(true)
	errs := v.ValidateServerConfig(cfg)
	if len(errs) == 0 {
		t.Error("should error when ClientAuth not set for mTLS")
	}
}

func TestTLSVersionName(t *testing.T) {
	tests := []struct {
		version uint16
		name    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0x9999, "UNKNOWN(0x9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tlsVersionName(tt.version)
			if got != tt.name {
				t.Errorf("tlsVersionName(%d) = %s, want %s", tt.version, got, tt.name)
			}
		})
	}
}

func TestIsWeakCipherSuite(t *testing.T) {
	tests := []struct {
		suite  uint16
		isWeak bool
	}{
		{0x0001, true},                           // NULL_NULL
		{tls.TLS_AES_256_GCM_SHA384, false},      // Strong
		{tls.TLS_AES_128_GCM_SHA256, false},      // Strong
		{tls.TLS_CHACHA20_POLY1305_SHA256, false}, // Strong
	}

	for _, tt := range tests {
		t.Run(tls.CipherSuiteName(tt.suite), func(t *testing.T) {
			got := isWeakCipherSuite(tt.suite)
			if got != tt.isWeak {
				t.Errorf("isWeakCipherSuite(%d) = %v, want %v", tt.suite, got, tt.isWeak)
			}
		})
	}
}

func TestProductionCipherSuites(t *testing.T) {
	suites := productionCipherSuites()

	required := []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	for _, suite := range required {
		if !suites[suite] {
			t.Errorf("missing required cipher suite: %s", tls.CipherSuiteName(suite))
		}
	}

	if len(suites) != len(required) {
		t.Errorf("expected exactly %d cipher suites, got %d", len(required), len(suites))
	}
}
