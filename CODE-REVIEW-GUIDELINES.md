# Code Review Guidelines for Steering Committee

**Prepared:** 2026-10-04  
**Purpose:** Standardize PR #68 review process across 5 review tracks  
**Timeline:** Oct 4-8 (code review active), Oct 9 (approval decision)  
**Success Criteria:** All 5 phases approved, zero critical findings left unresolved

---

## Review Organization

### 5 Parallel Review Tracks (One Per Phase)

**Track 1: Phase 6A - Scheduler Scaling (O(1) Topology)**
- Primary Reviewer: Architecture Lead
- Secondary Reviewers: 2x Backend Engineers
- Files: `pkg/scheduler/`, `pkg/topology/`, `cmd/scheduler/`
- Focus: Algorithm correctness, scale testing, edge cases
- Estimated Review Time: 12-15 hours

**Track 2: Phase 6B - Workload Management (GPU, StatefulSets, Storage)**
- Primary Reviewer: Workload Lead
- Secondary Reviewers: 2x Backend Engineers  
- Files: `pkg/workload/`, `pkg/placement/`, `pkg/storage/`
- Focus: Workload lifecycle, storage guarantees, GPU scheduling logic
- Estimated Review Time: 10-12 hours

**Track 3: Phase 6C - Governance (RBAC, Audit Logging)**
- Primary Reviewer: Security Lead
- Secondary Reviewers: 1x Backend, 1x Compliance Officer
- Files: `pkg/rbac/`, `pkg/audit/`, `pkg/policy/`
- Focus: Permission models, audit trail integrity, compliance
- Estimated Review Time: 10-12 hours

**Track 4: Phase 6D - Observability (Prometheus, OpenTelemetry, Alerting)**
- Primary Reviewer: DevOps Lead
- Secondary Reviewers: 2x DevOps Engineers
- Files: `pkg/metrics/`, `pkg/tracing/`, `pkg/alerting/`
- Focus: Metrics accuracy, trace completeness, alert logic
- Estimated Review Time: 8-10 hours

**Track 5: Phase 6E - Marketplace (Pricing, Lease, Settlement)**
- Primary Reviewer: Finance Lead
- Secondary Reviewers: 1x Backend, 1x Finance Analyst
- Files: `pkg/marketplace/`, `pkg/settlement/`, `pkg/pricing/`
- Focus: Pricing logic, settlement correctness, billing accuracy
- Estimated Review Time: 8-10 hours

---

## Review Criteria & Checklist

### Architecture & Design (10 points)

- [ ] **Distributed Systems Principles**
  - [ ] State machine explicitly defined (DESIRED → ADMITTED → EXECUTING → OBSERVED → VERIFIED)
  - [ ] Consensus mechanism documented (Raft, quorum rules)
  - [ ] Failure scenarios considered (node crash, network partition, Byzantine)
  - [ ] Eventual consistency guarantees stated (if applicable)
  - Scoring: 0 = No consideration, 1 = Partially addressed, 2 = Well-designed

- [ ] **API Design**
  - [ ] REST/gRPC endpoints clearly defined
  - [ ] Request/response schemas documented
  - [ ] Error handling standardized (error codes, messages)
  - [ ] Backwards compatibility maintained (if breaking change, documented)
  - Scoring: 0 = No API doc, 1 = Basic docs, 2 = Comprehensive

- [ ] **Scalability**
  - [ ] Algorithm complexity analyzed (O(n), O(n²), etc.)
  - [ ] Tested at target scale (450 nodes for 6A, etc.)
  - [ ] Load balancing strategy documented
  - [ ] Bottleneck mitigation planned
  - Scoring: 0 = Untested at scale, 1 = Tested small scale, 2 = Production-scale tested

### Code Quality (10 points)

- [ ] **Correctness**
  - [ ] Logic verified against specification
  - [ ] Edge cases handled (empty input, nil pointers, max values)
  - [ ] Race conditions eliminated (Go race detector passes)
  - [ ] Resource cleanup guaranteed (defer, context cancellation)
  - Scoring: 0 = Known bugs, 1 = Mostly correct, 2 = Correct + edge cases

- [ ] **Testing**
  - [ ] Unit tests: >90% coverage (go test -cover)
  - [ ] Integration tests: Real cluster scenarios (3+ nodes)
  - [ ] Chaos tests: Failure scenarios (network down, node crash)
  - [ ] Performance tests: Latency + throughput benchmarks
  - Scoring: 0 = No tests, 1 = Unit only, 2 = Comprehensive

- [ ] **Code Style**
  - [ ] Follows Go conventions (gofmt, golint pass)
  - [ ] Comments explain WHY (not WHAT)
  - [ ] Variable names descriptive
  - [ ] Functions <50 lines (prefer composition)
  - Scoring: 0 = Style violations, 1 = Mostly clean, 2 = Clean + well-commented

### Production Hardening (10 points)

- [ ] **TLS & Cryptography**
  - [ ] TLS 1.3 minimum (no fallback to 1.2)
  - [ ] mTLS mutual authentication verified
  - [ ] Certificate pinning implemented
  - [ ] Ed25519 key rotation working (90-day enforcement)
  - Scoring: 0 = No TLS, 1 = TLS but weak, 2 = TLS 1.3 + mTLS + rotation

- [ ] **Audit & Compliance**
  - [ ] All policy decisions logged
  - [ ] BLAKE3 audit chain tamper-proof (verified)
  - [ ] Sensitive data redacted from logs
  - [ ] Audit retention policy enforced (90+ days)
  - Scoring: 0 = No audit logs, 1 = Logs present, 2 = Complete audit trail

- [ ] **Input Validation & Security**
  - [ ] All user input validated (whitelist approach)
  - [ ] SQL injection impossible (parameterized queries)
  - [ ] Command injection impossible (no shell execution)
  - [ ] Rate limiting enforced (token bucket)
  - Scoring: 0 = No validation, 1 = Some validation, 2 = Complete + hardened

### Testing & Coverage (10 points)

- [ ] **Unit Test Coverage**
  - [ ] Overall coverage >90% (go tool cover -html)
  - [ ] Critical paths >95% (scheduler, RBAC, settlement)
  - [ ] Untested lines documented (intentional or bug?)
  - [ ] Coverage not just line-based (path coverage via mutation testing)
  - Scoring: 0 = <70%, 1 = 70-90%, 2 = >90%

- [ ] **Integration Tests**
  - [ ] Multi-node cluster scenarios
  - [ ] Cross-phase integration tested (6A + 6B together)
  - [ ] Failover & recovery verified
  - [ ] Latency & throughput measured
  - Scoring: 0 = No integration tests, 1 = Basic scenarios, 2 = Comprehensive

- [ ] **Chaos & Resilience**
  - [ ] Network partition handled (split-brain prevented)
  - [ ] Node crash recovery tested
  - [ ] Disk/memory exhaustion handled gracefully
  - [ ] Cascading failure prevention verified
  - Scoring: 0 = No chaos tests, 1 = Single failure, 2 = Multiple + cascading

### Documentation (5 points)

- [ ] **Code Documentation**
  - [ ] Exported functions have doc comments
  - [ ] Complex logic explained (WHY, not WHAT)
  - [ ] Types documented (struct fields, interfaces)
  - Scoring: 0 = No docs, 1 = Some docs, 2 = Complete

- [ ] **User-Facing Documentation**
  - [ ] Operator guide updated (if API change)
  - [ ] Configuration documented (all options)
  - [ ] Runbook provided (if operational change)
  - Scoring: 0 = No user docs, 1 = Incomplete, 2 = Complete

---

## Review Process

### Week 1: Kickoff & Initial Review (Oct 4-5)
1. **Assign Reviewers** (Oct 4 EOD)
   - Each track assigned primary + 2 secondary reviewers
   - Reviewers clone repo, set up dev environment
   - Distribute CODE-REVIEW-CHECKLIST.md to each reviewer

2. **Review Kick-off Meeting** (Oct 4, 5pm UTC)
   - 30-minute overview by PR author (Phase 6 architecture)
   - Q&A on review scope and timeline
   - Set expectations: Code review active Oct 4-8, decision Oct 9

3. **Initial Code Review** (Oct 5-6, 2 days)
   - Primary reviewer: Initial pass through code + tests
   - Identify major issues (blockers, security, correctness)
   - Post preliminary findings (GitHub PR comments)

### Week 2: Parallel Review & Discussion (Oct 6-8)
1. **Secondary Reviewer Pass** (Oct 6-7, 2 days)
   - Review same code independently
   - Identify additional issues (edge cases, style, performance)
   - Cross-check primary reviewer findings (agreement or disagreement)

2. **Author Response & Fixes** (Oct 7-8)
   - PR author responds to each finding
   - Provides clarification or commits fixes
   - Push new commits as required

3. **Reviewer Verification** (Oct 8)
   - Reviewers verify fixes address findings
   - Mark findings as resolved
   - Final sign-off per track

### Week 3: Final Decision (Oct 9)
1. **Consensus Meeting** (Oct 9, 2pm UTC)
   - Each track primary reviewer presents findings summary
   - Steering committee discusses and votes
   - Decision: Approve (green light to merge) or Request Changes

2. **Merge or Iterate** (Oct 9 EOD)
   - If approved: PR merged to main branch
   - If changes requested: PR author addresses, new review cycle starts
   - Target: No more than 1 additional cycle

---

## Comment & Finding Templates

### For Blocking Finding (Red Circle - Must Fix)
```
🔴 **BLOCKER:** [Category] 
Severity: CRITICAL

Issue: [Specific problem with code]
Impact: [Why this breaks the system]
Fix: [Recommended solution]
Example: [Code snippet showing correct approach]

Verify: [How to test the fix]
```

### For Optional Finding (Yellow Circle - Nice to Have)
```
🟡 **SUGGESTION:** [Category]
Severity: NICE-TO-HAVE

Issue: [Improvement opportunity]
Rationale: [Why this would be better]
Alternative: [If approach disputed]

This can be addressed in post-launch optimization (Phase 7).
```

### For Question/Clarification
```
❓ **QUESTION:** [Category]

Question: [What I don't understand]
Context: [Related code/spec section]

Please clarify: [What info would help]
```

---

## Review Acceptance Criteria

### Phase Approval Requirements
- [ ] **All Blocking Findings Resolved** (zero red circles in final review)
- [ ] **90%+ Code Coverage** (go tool cover shows >90%)
- [ ] **All Integration Tests Passing** (no flaky tests)
- [ ] **Security Audit Items Addressed** (if pre-identified in SECURITY-AUDIT-RFP)
- [ ] **Performance Benchmarks Met** (latency, throughput targets)
- [ ] **2/3 Reviewers Sign-Off** (majority approval per track)

### PR Approval Gate (Oct 9)
- [ ] **All 5 Tracks Approved**
  - Track 1 (6A): Architecture Lead + 2 Engineers
  - Track 2 (6B): Workload Lead + 2 Engineers
  - Track 3 (6C): Security Lead + 1 Backend + 1 Compliance
  - Track 4 (6D): DevOps Lead + 2 DevOps
  - Track 5 (6E): Finance Lead + 1 Backend + 1 Analyst

- [ ] **Steering Committee Consensus** (unanimous or 4/5+ majority)

---

## Reviewer Responsibility Guidelines

### What Reviewers Must Do
1. **Read the code** (don't just scan diffs)
2. **Run tests locally** (verify coverage, no flakiness)
3. **Check specifications** (does code match spec?)
4. **Think about failures** (what could go wrong?)
5. **Document findings** (clear, actionable comments)
6. **Be respectful** (code review is about improvement, not blame)

### What Reviewers Must NOT Do
1. **Don't approve without reading** (skim is not enough)
2. **Don't require style-only changes** (linters handle that)
3. **Don't approve correctness issues** (even if minor)
4. **Don't comment on taste/preference** (use team standards, not personal preference)
5. **Don't delay unnecessarily** (review within 24 hours of assignment)

### Reviewer Time Commitment
- Primary Reviewer: 12-15 hours over 5 days
- Secondary Reviewers: 8-10 hours each over 5 days
- Total per track: 28-35 hours

---

## Escalation Path

### If Reviewer & Author Disagree
1. **Try to resolve directly** (discussion in GitHub thread)
2. **Ask secondary reviewer** (tie-breaker perspective)
3. **Escalate to tech lead** (if still unresolved)
4. **Steering committee decides** (final authority)

### If Finding is Disputed
- Author can request review by different reviewer
- Disputed finding marked as "deferred" (can revisit post-launch)
- Blocking findings cannot be deferred (must resolve or not ship)

---

## Success Metrics

### Code Review Completion
- [ ] All code reviewed by Oct 8 (within timeline)
- [ ] <2 findings per 100 lines (reasonable feedback density)
- [ ] <5% disputed findings (good reviewer-author alignment)
- [ ] Zero critical findings post-approval (thorough review)

### Review Quality
- [ ] 100% blocking findings resolved
- [ ] >90% code coverage verified
- [ ] All integration tests passing
- [ ] Zero production bugs attributable to unreviewed code (30-day period)

---

**Code Review Guidelines - Decentralized.Host PR #68**  
**Prepared:** 2026-10-04  
**Status:** Ready for Oct 4 code review kickoff  
**Timeline:** Oct 4-9 (5-day review window)
