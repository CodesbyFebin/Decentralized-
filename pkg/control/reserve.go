package control

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"

	"decentralized/pkg/identity"
	"decentralized/pkg/providers"
)

// OwnerReserve represents proof that a user owns/controls hardware
type OwnerReserve struct {
	// Identity information
	OwnerID       string         `json:"owner_id"`        // User identity
	OwnerIdentity ed25519.PublicKey `json:"owner_identity"` // Public key proof

	// Hardware identification
	HardwareID    string                  `json:"hardware_id"`     // Unique hardware fingerprint
	HostID        string                  `json:"host_id"`         // Unique host identifier
	Hostname      string                  `json:"hostname"`        // System hostname

	// Hardware characteristics (immutable once registered)
	CPUCores      int                     `json:"cpu_cores"`
	CPUModel      string                  `json:"cpu_model"`
	MemoryBytes   int64                   `json:"memory_bytes"`
	StorageBytes  int64                   `json:"storage_bytes"`
	GPUTypes      []string                `json:"gpu_types"`

	// Proof of control
	ReserveToken  string                  `json:"reserve_token"`    // Signed proof of ownership
	ReserveSignature string                `json:"reserve_signature"` // Ed25519 signature
	TokenIssued   time.Time               `json:"token_issued"`
	TokenExpiry   time.Time               `json:"token_expiry"`

	// Audit trail
	VerifiedAt    time.Time               `json:"verified_at"`
	VerificationMethod string              `json:"verification_method"` // "hardware_attestation", "control_plane_auth", etc.
	Evidence      *providers.Evidence     `json:"evidence"`

	// Status
	Status        string                  `json:"status"`           // "verified", "expired", "revoked", "pending"
	RevokedAt     *time.Time              `json:"revoked_at,omitempty"`
	RevokeReason  string                  `json:"revoke_reason,omitempty"`
}

// ReserveValidator manages owner verification
type ReserveValidator struct {
	maxTokenLifetime time.Duration
	signer           ed25519.PrivateKey
	verifier         ed25519.PublicKey
}

// NewReserveValidator creates a new owner-reserve validator
func NewReserveValidator(signingKey ed25519.PrivateKey) *ReserveValidator {
	return &ReserveValidator{
		maxTokenLifetime: 24 * time.Hour,
		signer:           signingKey,
		verifier:         signingKey.Public().(ed25519.PublicKey),
	}
}

// VerifyOwnership performs a multi-stage verification that the user owns the hardware
func (rv *ReserveValidator) VerifyOwnership(
	ctx context.Context,
	ownerID string,
	ownerPubKey ed25519.PublicKey,
	hostID string,
	hostname string,
	hwProfile interface{}, // Hardware profile from prober
) (*OwnerReserve, error) {

	reserve := &OwnerReserve{
		OwnerID:    ownerID,
		OwnerIdentity: ownerPubKey,
		HostID:     hostID,
		Hostname:   hostname,
		Status:     "pending",
	}

	// Stage 1: Verify control via challenge-response (simplified for v0.1)
	// In production, this would involve:
	// - Issuing a cryptographic challenge
	// - Requiring proof of execution on the target hardware
	// - Verifying hardware attestation (if available)
	// - Checking file system permissions
	// - Running control-plane auth check

	if err := rv.verifyControlPlaneAuth(ctx, ownerID, hostID); err != nil {
		return nil, fmt.Errorf("control plane auth failed: %w", err)
	}

	// Stage 2: Generate reserve token
	token, sig, err := rv.generateReserveToken(ownerID, hostID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reserve token: %w", err)
	}

	reserve.ReserveToken = token
	reserve.ReserveSignature = sig
	reserve.TokenIssued = time.Now()
	reserve.TokenExpiry = time.Now().Add(rv.maxTokenLifetime)

	// Stage 3: Record verification
	reserve.VerifiedAt = time.Now()
	reserve.VerificationMethod = "control_plane_auth"
	reserve.Status = "verified"

	// Stage 4: Create evidence (cryptographic proof)
	reserve.Evidence = &providers.Evidence{
		Signature:   sig,
		SignedBy:    hex.EncodeToString(rv.verifier),
		Timestamp:   time.Now(),
		ContentHash: hashOwnerReserve(reserve),
		Metadata: map[string]string{
			"owner_id":    ownerID,
			"host_id":     hostID,
			"verification_method": "control_plane_auth",
		},
	}

	return reserve, nil
}

// VerifyAndExtendReserve checks if a reserve token is still valid and extends it if needed
func (rv *ReserveValidator) VerifyAndExtendReserve(
	ctx context.Context,
	reserve *OwnerReserve,
	extensionDays int,
) error {

	// Check if token is expired
	if time.Now().After(reserve.TokenExpiry) {
		reserve.Status = "expired"
		return fmt.Errorf("reserve token expired at %s", reserve.TokenExpiry)
	}

	// Check if revoked
	if reserve.Status == "revoked" {
		return fmt.Errorf("reserve token revoked: %s", reserve.RevokeReason)
	}

	// If expiring soon, extend it
	if time.Until(reserve.TokenExpiry) < 72*time.Hour {
		reserve.TokenExpiry = time.Now().Add(time.Duration(extensionDays) * 24 * time.Hour)
		reserve.VerifiedAt = time.Now()
	}

	return nil
}

// RevokeReserve marks a reserve token as revoked
func (rv *ReserveValidator) RevokeReserve(
	ctx context.Context,
	reserve *OwnerReserve,
	reason string,
) error {

	reserve.Status = "revoked"
	now := time.Now()
	reserve.RevokedAt = &now
	reserve.RevokeReason = reason

	return nil
}

// verifyControlPlaneAuth checks if the user is authenticated to the control plane for this host
func (rv *ReserveValidator) verifyControlPlaneAuth(ctx context.Context, ownerID string, hostID string) error {
	// In production, this would:
	// 1. Call the control plane to verify the user is authorized for this host
	// 2. Check RBAC policies
	// 3. Verify host trust status

	// For v0.1, we do a basic check
	if ownerID == "" || hostID == "" {
		return fmt.Errorf("missing owner_id or host_id")
	}

	// Placeholder: In real implementation, call control plane API
	// cpClient := getControlPlaneClient()
	// return cpClient.AuthorizeHostAccess(ctx, ownerID, hostID)

	return nil
}

// generateReserveToken creates a signed token proving ownership
func (rv *ReserveValidator) generateReserveToken(ownerID string, hostID string) (string, string, error) {
	// Create a deterministic message
	message := fmt.Sprintf("RESERVE:%s:%s:%d",
		ownerID,
		hostID,
		time.Now().Unix(),
	)

	// Sign the message
	signature := ed25519.Sign(rv.signer, []byte(message))
	signatureHex := hex.EncodeToString(signature)

	return message, signatureHex, nil
}

// hashOwnerReserve creates a BLAKE3 hash of the reserve (placeholder for v0.1)
func hashOwnerReserve(reserve *OwnerReserve) string {
	// In production, use BLAKE3
	// For v0.1, use simple string hash
	return fmt.Sprintf("hash:%s:%s:%d",
		reserve.OwnerID,
		reserve.HostID,
		reserve.VerifiedAt.Unix(),
	)
}

// ValidReserveForDeployment checks if a reserve is valid for deploying workloads
func (reserve *OwnerReserve) ValidForDeployment() (bool, []string) {
	var issues []string

	if reserve.Status != "verified" {
		issues = append(issues, fmt.Sprintf("reserve status is %s, not verified", reserve.Status))
	}

	if time.Now().After(reserve.TokenExpiry) {
		issues = append(issues, "reserve token has expired")
	}

	if time.Now().Before(reserve.TokenIssued) {
		issues = append(issues, "reserve token not yet valid")
	}

	if len(issues) > 0 {
		return false, issues
	}

	return true, nil
}

// CapabilitiesFromReserve extracts capability declarations from a validated reserve
func CapabilitiesFromReserve(reserve *OwnerReserve) []providers.Capability {
	var caps []providers.Capability

	if reserve.CPUCores > 0 {
		caps = append(caps, providers.Capability{
			Name:  fmt.Sprintf("cpu-%d-core", reserve.CPUCores),
			Level: "full",
			Description: fmt.Sprintf("%d-core CPU (%s)",
				reserve.CPUCores, reserve.CPUModel),
			Evidence: reserve.Evidence,
		})
	}

	if reserve.MemoryBytes > 0 {
		gbTotal := reserve.MemoryBytes / (1024 * 1024 * 1024)
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("memory-%dgb", gbTotal),
			Level:       "full",
			Description: fmt.Sprintf("%d GB RAM", gbTotal),
			Evidence:    reserve.Evidence,
		})
	}

	if reserve.StorageBytes > 0 {
		gbTotal := reserve.StorageBytes / (1024 * 1024 * 1024)
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("storage-%dgb", gbTotal),
			Level:       "full",
			Description: fmt.Sprintf("%d GB storage", gbTotal),
			Evidence:    reserve.Evidence,
		})
	}

	for _, gpuType := range reserve.GPUTypes {
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("gpu-%s", gpuType),
			Level:       "full",
			Description: fmt.Sprintf("%s GPU", gpuType),
			Evidence:    reserve.Evidence,
		})
	}

	return caps
}
