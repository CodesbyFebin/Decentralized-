# Operator Qualification Program

Decentralized.Host dh/v1 requires operators to complete a structured qualification process to join the network. This directory contains the complete guidance and resources for operator onboarding and qualification.

## Documentation Overview

### 1. [TRAINING.md](TRAINING.md)
The Operator Training Program covers:
- **Module 1**: Qualification Requirements (stake, nodes, SLA, security, backup)
- **Module 2**: Tier Progression (BOOTSTRAP → TRUSTED → MASTER)
- **Module 3**: Technical Operations (deployment, scaling, health monitoring)
- **Module 4**: Security & Compliance (key rotation, TLS/mTLS, audit logs)
- **Module 5**: Economics & Settlement (pricing, billing, analytics)
- **Module 6**: Hands-On Labs (5 practical exercises)
- **Module 7**: Certification & Sign-Off

**Duration**: 2-3 weeks for completion
**Target Audience**: New operators, infrastructure teams
**Delivery**: Self-paced online learning + live labs

### 2. [SECURITY-AUDIT.md](SECURITY-AUDIT.md)
The Security Audit Checklist ensures operator infrastructure meets dh/v1 security standards:
- Identity & Authentication (5 items)
- Network Security (6 items)
- Data Protection (5 items)
- Access Control (5 items)
- Operational Security (5 items)
- Compliance & Audit (5 items)
- Deployment Security (5 items)
- Monitoring & Response (5 items)

**Timing**: Phase 3 of onboarding (2-3 weeks)
**Auditor**: External security professional
**Sign-Off**: Required before BOOTSTRAP activation

### 3. [ONBOARDING.md](ONBOARDING.md)
The Operator Onboarding Runbook is a step-by-step guide covering:
- **Phase 1**: Pre-Registration (1-2 weeks) — Training, documentation, infrastructure prep
- **Phase 2**: Registration (1-2 weeks) — Stake deposit, node registration, identity setup
- **Phase 3**: Security Audit (2-3 weeks) — Pre-audit prep, security review, compliance
- **Phase 4**: Operational Readiness (1 week) — SLA baseline, backup/DR testing, health checks
- **Phase 5**: Go-Live (1 day) — Final approval, activation, launch support
- **Phase 6**: Tier Progression (90+ days) — Maintenance of requirements for TRUSTED/MASTER

**Total Timeline**: 7-10 weeks from initial interest to BOOTSTRAP activation
**Critical Path**: Security audit (most time-intensive phase)

## Qualification Requirements by Tier

### BOOTSTRAP (Entry)
- **Stake**: 100M uWork
- **Nodes**: 50+ operational
- **SLA Target**: 95%+ uptime
- **Duration**: 0 days (immediate activation after Phase 5)
- **Reputation Points**: 0-100

### TRUSTED (Intermediate)
- **Stake**: 200M uWork
- **Nodes**: 100+ operational
- **SLA Target**: 98%+ uptime
- **Duration**: 90 days stable operation
- **Reputation Points**: 100-500

### MASTER (Full Capability)
- **Stake**: 500M uWork
- **Nodes**: 250+ operational
- **SLA Target**: 99.5%+ uptime
- **Duration**: 1 year stable operation
- **Reputation Points**: 500+

## Key Operator Obligations

1. **Technical**
   - Maintain registered node count
   - Implement TLS 1.3 + mTLS
   - Execute Ed25519 key rotation (90-day cycle)
   - Maintain 24/7 monitoring and alerting

2. **Financial**
   - Maintain minimum stake for tier
   - Comply with dynamic pricing model
   - Settle lease payments on time
   - Accept SLA-based credits/penalties

3. **Security**
   - Pass initial security audit
   - Complete annual re-audits
   - Report security incidents within 1 hour
   - Maintain 90+ day audit log retention

4. **Operational**
   - Maintain target SLA percentages
   - Respond to alerts within 15 minutes
   - Execute disaster recovery tests quarterly
   - Participate in community updates/calls

## Operator Success Metrics

The system automatically tracks:
- **Uptime**: % of time nodes are healthy and operational
- **Latency**: Response time for workload placement (target: <100ms)
- **Error Rate**: Failed operations / total operations (target: <0.1%)
- **Throughput**: Workloads placed per second (baseline: 1000 tx/sec)
- **Recovery Time**: RTO for failure scenarios (target: <5 min)

Poor performance may trigger:
- SLA credit issuance (to customers)
- Operator notice period for improvement
- Graduated sanctions (suspension, removal)

## Contact & Support

**Onboarding Team**: [operator-onboarding@decentralized.host](mailto:operator-onboarding@decentralized.host)
**Security Audits**: [security@decentralized.host](mailto:security@decentralized.host)
**Operations Support**: [ops@decentralized.host](mailto:ops@decentralized.host)
**Executive Escalation**: [steering@decentralized.host](mailto:steering@decentralized.host)

## Next Steps

1. **New Operator?** Start with [TRAINING.md](TRAINING.md) Module 1
2. **Ready to Register?** Follow the [ONBOARDING.md](ONBOARDING.md) phases
3. **Preparing for Audit?** Review [SECURITY-AUDIT.md](SECURITY-AUDIT.md) checklist
4. **Questions?** Contact the onboarding team above

---

**Last Updated**: 2026-10-03
**dh/v1 Conformance**: Yes
**Status**: Production Ready
