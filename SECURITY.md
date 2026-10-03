# Security Policy

## Reporting Security Vulnerabilities

**Do not file public issues for security vulnerabilities.** Instead, please email [security@decentralized-host.dev](mailto:codesbyfebin@gmail.com?subject=Security%20Vulnerability) with:

- **Description:** Clear explanation of the vulnerability
- **Reproduction:** Steps to reproduce the issue
- **Impact:** Potential damage or exposure
- **Proposed fix:** Your suggested solution (if available)

**We will:**
1. Acknowledge your report within 48 hours
2. Investigate and verify the vulnerability
3. Work with you to develop and test a fix
4. Coordinate public disclosure with you
5. Credit you in the security advisory (if you'd like)

## Security Features

### Cryptographic Assurance
- **Signed intent:** All work assignments use Ed25519 signatures
- **Signed observations:** Hosts report state with cryptographic proof
- **Hash-chained audit trail:** Tamper detection at every boundary
- **BLAKE3 content addressing:** Hash-based storage integrity

### Isolation & Control
- **Per-host admission control:** No centralized overrides
- **Local policy enforcement:** Each host decides autonomously
- **Revocation support:** Immediate effect (no grace period exploits)
- **Key rotation:** Automatic and operator-triggered

### Encrypted Channels
- **Control-plane TLS:** Mutual mTLS for all APIs
- **Mesh WireGuard:** Peer-to-peer encryption, userspace implementation
- **No unencrypted communication:** TLS enforced from first start

### Audit & Compliance
- **Complete audit trail:** Every decision is logged and signed
- **Local verification:** Operators can verify ledger integrity offline
- **No telemetry:** No data leaves the cluster
- **Tamper detection:** Hash-chained logs prevent retroactive modification

## Known Limitations

### By Design
- **Byzantine hosts:** System doesn't defend against compromised hosts (they can refuse work)
- **Byzantine operators:** Federation assumes trust between operators
- **Timing attacks:** Cryptographic paths are checked carefully but haven't been formally analyzed
- **Kernel exploits:** Userspace WireGuard is not protected from kernel-level attacks

### Not Implemented Yet
- **Formal verification:** Protocol has not been formally verified
- **Differential privacy:** Topology and workload distribution may leak information
- **Quantum-resistant crypto:** Uses standard elliptic curve cryptography
- **Hardware security modules:** Keys are stored in plaintext on disk (encrypt with host filesystem)

## Security Best Practices

### For Operators

1. **Secure bootstrap:**
   - Generate root key on isolated machine
   - Manually verify control-plane certificate fingerprints
   - Use `dh init --acme` for TLS with ACME

2. **Protect secrets:**
   - Encrypt `/etc/dh` partition at rest
   - Use restricted file permissions (mode 0700)
   - Rotate host keys regularly
   - Monitor audit logs for anomalies

3. **Network security:**
   - Isolate control-plane to trusted network
   - Firewall WireGuard port (default 51820)
   - Use VPN or air-gap for production clusters
   - Monitor gossip for unexpected peers

4. **Incident response:**
   - Maintain signed backups (test restore regularly)
   - Revoke compromised certificates immediately
   - Audit trail is tamper-proof (check integrity)
   - Plan for leader loss and total control-plane failure

### For Developers

1. **Input validation:**
   - Validate all network inputs
   - Bounds-check offsets and lengths
   - Use strong types (not `interface{}`)
   - Parse untrusted data carefully

2. **Cryptographic hygiene:**
   - Use `crypto/sha256` and `blake3` appropriately
   - Constant-time comparison for signatures: `subtle.ConstantTimeCompare`
   - No hardcoded secrets in source
   - Use random nonces and salts

3. **Concurrency safety:**
   - Avoid data races (`make race` in CI)
   - Use channels for synchronization
   - Document shared state clearly
   - Test with `go test -race`

4. **Deployment security:**
   - Sign binaries and container images
   - Use staged rollouts
   - Monitor control-plane for unexpected behavior
   - Log all administrative actions

## Vulnerability Response Timeline

| When | Action |
|------|--------|
| Day 0 | Initial report and acknowledgment |
| Day 1-2 | Verification and impact assessment |
| Day 3-7 | Fix development and testing |
| Day 7-14 | Beta release or scheduled release window |
| Release day | Public security advisory and CVE assignment |
| Day 30+ | Post-mortem and analysis (if applicable) |

## CVE Assignments

Security vulnerabilities discovered in Decentralized.Host are assigned CVE identifiers. See [GitHub Security Advisories](https://github.com/CodesbyFebin/Decentralized-/security/advisories) for the full list.

## Security Audits

- **M1–M8 testing:** Each milestone includes comprehensive multi-process tests
- **P1 qualification:** 10/10 gates verify robustness under 17 chaos scenarios
- **Conformance:** 136 test vectors against independent Python implementation
- **Formal audit:** Not yet performed (planned for production release)

## Updates & Patches

### Staying Secure
- Subscribe to [GitHub release notifications](https://github.com/CodesbyFebin/Decentralized-/releases)
- Monitor security advisories: https://github.com/CodesbyFebin/Decentralized-/security
- Enable automatic updates in your deployment
- Test updates in staging before production

### Patching Timeline
- Critical vulnerabilities: Emergency release within 48 hours
- High-severity: Next scheduled release (within 2 weeks)
- Medium/Low: Regular release cycle (monthly or as part of feature releases)

## Questions?

- **Security issues:** [Email us](mailto:codesbyfebin@gmail.com?subject=Security%20Inquiry)
- **General questions:** [GitHub Discussions](https://github.com/CodesbyFebin/Decentralized-/discussions)
- **Runbooks:** [docs/runbooks/](docs/runbooks/)

---

**Thank you for helping us keep Decentralized.Host secure!**
