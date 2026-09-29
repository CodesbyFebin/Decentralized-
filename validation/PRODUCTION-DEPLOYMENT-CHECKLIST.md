# Production Deployment Security Checklist

**Date**: 2026-09-29  
**Audience**: Cluster operators and deployment engineers  
**Requirement Level**: MUST for P1_CORE qualification

---

## Pre-Deployment Configuration

### 1. TLS Configuration [CRITICAL]

**Requirement**: All API communication must be encrypted.

**Configuration**:
```bash
# Control plane members must be deployed with:
dh-control -tls

# This serves the operator/host API over TLS 1.3 with:
# - Root-issued member certificate
# - Mutual TLS (mTLS) for peer connections (Raft)
# - Client certificate validation
```

**Verification**:
```bash
# After deployment, verify TLS is active:
curl -k https://localhost:7700/health  # Should succeed
curl http://localhost:7700/health      # Should fail or redirect
```

**Why**: Without TLS, operator credentials and host observations transmitted in cleartext.
- **Risk**: Network eavesdropping of sensitive data
- **Impact**: Violates P1_CORE gates 17-24 (failure detection over secure transport)

---

### 2. Bootstrap Certificate Pinning [CRITICAL]

**Requirement**: Operators must verify bootstrap certificate fingerprint when joining new members.

**Procedure**:
```bash
# Step 1: Bootstrap code displayed on stdout
dh-control -data /path/to/new-member

# Output:
# member dh1<nodeID> awaiting bootstrap or join; one-time code in /path/to/new-member/bootstrap.code

# Step 2: Examine bootstrap certificate fingerprint
cat /path/to/new-member/bootstrap.code
# Example output:
# dhjoin1.fingerprint=sha256:<64-hex-chars>

# Step 3: Out-of-band verification
# Operator prints this fingerprint over secure channel (phone call, encrypted message)
# Joining operator verifies it matches before proceeding with join

# Step 4: Complete join
dh node join dhjoin1.<token>
```

**Verification**:
```bash
# Confirm member joined with correct identity
dh get nodes | grep dh1<nodeID>
```

**Why**: Bootstrap certificate is self-signed and one-time only. Pinning prevents MITM.
- **Risk**: Unauthorized node could impersonate legitimate node during join
- **Impact**: Violates P1_CORE gate 17 (node topology authenticity)

---

### 3. Key Rotation Setup [HIGH]

**Requirement**: Plan for periodic key rotation (recommend: yearly).

**Procedure**:
```bash
# Step 1: Derive new key on member
dh node rotate-key --grace 5m

# Step 2: Monitor peer acceptance (grace period allows gradual migration)
# During grace period, both old and new keys are valid

# Step 3: After grace, old key is archived
ls identity/identity.key.retired-*

# Step 4: Verify new key is in use
dh get nodes --detail | grep <nodeID>
```

**Audit Trail**:
- Rotation is recorded in Raft audit trail: `identity/rotation-<timestamp>.json`
- Includes old signature (proof that old key authorized the new one)
- Timestamp is immutable (from proposal_ts in Raft)

**Why**: Ed25519 key compromise could allow impersonation of rotated identity.
- **Risk**: Stolen old key could sign malicious bundles (during grace period only)
- **Mitigation**: Grace period is configurable; shorten for high-security environments

---

### 4. Secrets Bootstrap Material [CRITICAL]

**Requirement**: Secrets encryption requires bootstrap material (KEK derivation).

**Procedure**:
```bash
# Step 1: Generate 32 bytes of cryptographically random material
openssl rand -base64 32  # or similar

# Step 2: Encode for environment variable
DECENTRALIZED_KEK_BOOTSTRAP=$(python3 -c "
  import base64, os
  material = os.urandom(32)
  print(base64.urlsafe_b64encode(material).decode().rstrip('='))
")

# Step 3: Provide to control-plane members
# Option A: Environment variable at startup
export DECENTRALIZED_KEK_BOOTSTRAP="<value>"
dh-control ...

# Option B: Kubernetes secret
kubectl create secret generic dh-kek-bootstrap \
  --from-literal=material="$DECENTRALIZED_KEK_BOOTSTRAP" \
  -n dh-system

# Step 4: Never log or display bootstrap material
# Verify it's not in process listing:
ps aux | grep dh-control  # Should NOT show bootstrap value
```

**Verification**:
```bash
# After startup, verify KEK is derived:
dh control status | grep kek_status
# Should show: "Locked: false" (KEK available)
```

**Important**: 
- Bootstrap material is **never persisted** (not in Raft, etcd, or disk)
- Must be provided at startup of each member
- Lost bootstrap means all DEKs become unrecoverable (safe failure mode)

**Why**: KEK protects all Data Encryption Keys (DEKs) for secrets.
- **Risk**: Bootstrap compromise allows decryption of all secrets
- **Mitigation**: Use high-entropy random material (not passwords)

---

### 5. Audit Trail Configuration [HIGH]

**Requirement**: Collect and retain audit trail for all security events.

**Procedure**:
```bash
# Step 1: Enable audit mirror (optional but recommended)
dh-control \
  -postgres "postgresql://user:pass@postgres:5432/dh_audit" \
  ...

# Step 2: Configure audit retention policy
# Default: 90-day retention (configurable via environment variable)
export DH_AUDIT_RETENTION_DAYS=365

# Step 3: Collect audit logs
dh audit tail --format json | tee audit-$(date +%Y%m%d).jsonl

# Step 4: Archive and secure
tar czf audit-$(date +%Y%m%d).tar.gz audit-$(date +%Y%m%d).jsonl
# Copy to secure storage (offline, encrypted)
```

**Audit Events**:
- ✅ Node enrollment (identity, join time)
- ✅ Key rotations (old key, new key, grace period)
- ✅ Signed intent (bundle proposals, rejections)
- ✅ Secret creation and retrieval authorization
- ✅ Policy enforcement (local policy matches, rejections)
- ✅ Observations (signed evidence from nodes)

**Why**: Audit trail is proof of compliance with qualification gates 25-32.
- **Requirement**: All security-relevant events must be recorded
- **Scope**: Covers identity binding, rotation, signed intent, evidence integrity

---

### 6. Certificate Monitoring [MEDIUM]

**Requirement**: Monitor for upcoming certificate expiration.

**Procedure**:
```bash
# Step 1: Check member certificate expiration
dh get nodes --detail | grep -i "cert.*expire\|key.*expire"

# Step 2: Set up automated alerting
# Add to monitoring (Prometheus, DataDog, etc.):
dh_cert_expires_at_timestamp{node_id="dh1<ID>"} <unix-seconds>

# Step 3: Alert thresholds
# Warning: 90 days before expiration
# Critical: 30 days before expiration

# Step 4: Renewal procedure
# Member certificates expire in 5 years; renewal requires leadership
dh control renew-certs
```

**Certificate Details**:
- Root CA: 20-year validity (very unlikely to expire in practice)
- Member certificates: 5-year validity
- Local CA (for ingress): 10-year validity

**Why**: Certificate expiration causes loss of cluster connectivity.
- **Risk**: Unplanned downtime if certificate expires
- **Mitigation**: Proactive renewal 90+ days before expiration

---

### 7. Memory Clearing for Secrets [MEDIUM]

**Requirement**: Plaintext secrets must be cleared from memory after use.

**Implementation Status**: ✅ IMPLEMENTED

**Code Usage**:
```go
// When decrypting secrets, clear immediately after use:
plaintext, _ := DecryptSecret(record, dek)
defer ClearBytes(plaintext)  // Clear after decryption

// For secret delivery to workloads:
// Materialized secrets are in tmpfs (not persisted to disk)
// Cleanup occurs when workload terminates
```

**Verification**:
- Review code for all `DecryptSecret` calls
- Ensure `defer ClearBytes(plaintext)` is used
- Check that secrets are not logged or printed

**Why**: Plaintext in memory could be exposed via memory dump.
- **Risk**: Memory forensics could recover decrypted secrets
- **Mitigation**: Explicit zeroing (this release)

---

### 8. Network Security [MEDIUM]

**Requirement**: Isolate control plane and node communication.

**Configuration**:
```bash
# Step 1: Bind to private networks only
dh-control \
  -api 10.0.0.1:7700 \         # Private network
  -raft 10.0.0.2:7800 \         # Private network
  -mesh 10.0.0.3:51900 ...      # Private network (WireGuard)

# Step 2: Firewall rules
# Only allow operators to access API (7700)
# Only allow peers to access Raft (7800)
# Only allow nodes to access mesh (51900)

# Step 3: Network policy (Kubernetes)
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: dh-control-network-policy
spec:
  podSelector:
    matchLabels:
      app: dh-control
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:  # Only dh-noded pods
        matchLabels:
          app: dh-noded
    ports:
    - protocol: TCP
      port: 7700
```

**Why**: Control plane is trust anchor; network isolation prevents unauthorized access.
- **Risk**: Unauthorized bundle injection, observation tampering
- **Mitigation**: Network-level access control

---

## Deployment Verification

### Checklist

- [ ] **TLS Enabled**: `dh-control -tls` confirmed in running process
- [ ] **Bootstrap Verified**: Certificate fingerprint manually verified before join
- [ ] **Key Rotation Scheduled**: Annual key rotation schedule established
- [ ] **Bootstrap Material Secured**: KEK bootstrap not logged, only env var
- [ ] **Audit Trail Active**: `dh audit tail` returns events
- [ ] **Certificate Monitoring**: Alerts set for 90-day window
- [ ] **Memory Clearing**: Code review confirms `ClearBytes` usage
- [ ] **Network Isolated**: Control plane on private network, firewall rules in place
- [ ] **Documentation Updated**: Runbook covers all security procedures
- [ ] **Incident Plan**: Procedure for key compromise, audit log recovery

### Post-Deployment Tests

```bash
# Verify TLS enforcement
curl http://localhost:7700/health 2>&1 | grep -i "refused\|ssl\|tls"

# Verify audit trail collection
dh audit tail --limit 10

# Verify node identities
dh get nodes | grep "^dh1"

# Verify secret operations
dh secret create test-secret "test-value"
dh secret retrieve test-secret  # Should work
dh secret delete test-secret    # Should work
```

---

## Incident Response

### Key Compromise

**If private key is exposed**:
1. Immediately revoke the node: `dh node revoke <node-id>`
2. Wait for replica election (typically <10s)
3. Generate new key: `dh node rotate-key --grace 0s` (skip grace for emergency)
4. Review audit trail for suspicious activity

### Bootstrap Material Leak

**If DECENTRALIZED_KEK_BOOTSTRAP is exposed**:
1. All DEKs become compromised; plan for secrets re-encryption
2. Rotate bootstrap material: Change env var, restart all control members
3. Re-encrypt all secrets with new KEK

### Audit Trail Loss

**If audit logs are corrupted or lost**:
1. Cannot recover lost events (immutable once written)
2. Deployment loses compliance evidence for those events
3. Incident post-mortem is the recovery document

---

## References

- **Security Audit**: `validation/SECURITY-AUDIT-2026-09-29.md`
- **P1 Qualification**: `validation/p1-core-qualification.sh`
- **Cryptographic Operations**: `pkg/control/crypto.go`
- **Identity Management**: `pkg/identity/identity.go`
- **TLS Configuration**: `pkg/pki/pki.go`
- **Secrets Management**: `pkg/control/secrets.go`

---

**Last Updated**: 2026-09-29  
**Approval**: Pending security review
