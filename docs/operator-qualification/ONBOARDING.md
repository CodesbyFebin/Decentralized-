# Operator Onboarding Runbook

## Phase 1: Pre-Registration (1-2 weeks)

### 1.1 Initial Contact
- [ ] Operator submits intent-to-register via dh-cli
- [ ] System assigns unique operator ID (op-<uuid>)
- [ ] Onboarding team sends welcome packet and timeline
- [ ] Schedule kick-off call to review requirements

### 1.2 Documentation & Training
- [ ] Operator completes online training modules (Sec 2.3)
- [ ] Operator reviews dh/v1 specification
- [ ] Operator completes hands-on labs (Module 6)
- [ ] Operator team assigned point of contact (POC)

### 1.3 Infrastructure Readiness
- [ ] Operator provisions 50+ nodes (BOOTSTRAP requirement)
- [ ] Operator configures TLS/mTLS certificates
- [ ] Operator sets up monitoring and alerting
- [ ] Operator performs first backup/restore test

## Phase 2: Registration (1-2 weeks)

### 2.1 Stake Deposit
- [ ] Operator deposits 100M uWork to stake wallet
- [ ] Deposit confirmed on-chain with 1-day lock
- [ ] Operator registers stake via `dh-cli operator register-stake`
- [ ] System confirms stake in BOOTSTRAP tier

### 2.2 Node Registration
- [ ] Operator collects hardware inventory (CPUs, GPUs, memory)
- [ ] Operator collects network topology (bandwidth, latency to regions)
- [ ] Operator registers each node via `dh-cli node register`
- [ ] System validates node connectivity and health
- [ ] Node registration must reach 50+ for tier progression
- [ ] System assigns node IDs and generates keys

### 2.3 Identity Configuration
- [ ] Operator generates Ed25519 key pair
- [ ] Operator securely stores private key
- [ ] Operator submits public key to system
- [ ] System binds operator identity to registered nodes
- [ ] Operator configures key rotation schedule (90-day cycle)

## Phase 3: Security Audit (2-3 weeks)

### 3.1 Pre-Audit Preparation
- [ ] Operator prepares infrastructure for audit
- [ ] Operator ensures full audit log availability
- [ ] Operator enables all security monitoring
- [ ] Operator briefs team on security posture

### 3.2 Security Review
- [ ] External auditor performs security assessment
- [ ] Auditor reviews all checklist items (Sec 2.2)
- [ ] Auditor conducts penetration testing (if requested)
- [ ] Auditor provides findings report
- [ ] Operator remediates critical/high issues
- [ ] Auditor performs follow-up verification

### 3.3 Compliance Verification
- [ ] Operator confirms all remediation complete
- [ ] Operator signs compliance statement
- [ ] Auditor issues security certification
- [ ] System records certification on-chain

## Phase 4: Operational Readiness (1 week)

### 4.1 SLA Baseline
- [ ] System monitors operator for 7 days (baseline period)
- [ ] System measures uptime, latency, error rate
- [ ] System verifies 95%+ uptime minimum
- [ ] System records baseline metrics

### 4.2 Backup & Disaster Recovery
- [ ] Operator performs full backup of all nodes
- [ ] Operator simulates disaster recovery scenario
- [ ] Operator measures RTO (target: <5 minutes)
- [ ] Operator measures RPO (target: <1 minute)
- [ ] Operator verifies backup restore procedure

### 4.3 Health Check
- [ ] System runs end-to-end health checks
- [ ] Operator confirms all nodes healthy
- [ ] Operator confirms all monitoring working
- [ ] Operator confirms alert notifications working

## Phase 5: Go-Live (1 day)

### 5.1 Final Approval
- [ ] Compliance team confirms all requirements met
- [ ] Onboarding POC performs final review call
- [ ] System generates activation transaction
- [ ] Operator confirms go-live authorization

### 5.2 Activation
- [ ] System activates operator in BOOTSTRAP tier
- [ ] Operator receives initial workload assignments
- [ ] System enables operator dashboard and APIs
- [ ] Operator receives API credentials and auth tokens

### 5.3 Launch Support
- [ ] Live support team monitored for 48 hours
- [ ] Daily standup calls for first week
- [ ] Incident response procedures tested
- [ ] Operator given escalation contacts

## Phase 6: Tier Progression (90+ days)

### 6.1 TRUSTED Tier Requirements
- [ ] Operator maintains 98%+ uptime for 90 days
- [ ] Operator expands to 100+ nodes
- [ ] Operator increases stake to 200M uWork
- [ ] Operator passes security re-audit

### 6.2 MASTER Tier Requirements  
- [ ] Operator maintains 99.5%+ uptime for 1 year
- [ ] Operator operates 250+ nodes
- [ ] Operator increases stake to 500M uWork
- [ ] Operator passes comprehensive security audit
- [ ] Operator demonstrates cross-region capability

## Appendix: Automation

### Automated Checks
```bash
# Verify operator registration
dh-cli operator status <operator-id>

# Check node health
dh-cli node list --operator <operator-id>
dh-cli node health <node-id>

# Monitor SLA compliance
dh-cli operator sla-status <operator-id>

# View audit log
dh-cli audit log --operator <operator-id>
```

### Escalation Path
1. **Tier 1**: Onboarding POC (email/Slack)
2. **Tier 2**: Operations team lead (Slack @ops)
3. **Tier 3**: Security team (security@decentralized.host)
4. **Tier 4**: Executive steering committee (for policy exceptions)

