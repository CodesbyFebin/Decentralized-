# Dev-Cluster v0.1 Evidence Archive

This directory contains validation evidence from Decentralized.Host v0.1 alpha development on a **dev-cluster only** (3+3 loopback topology, single control surface).

## Important Scope Limitations

These files document:
- ✅ Unit test conformance (136/136 dh/v1 test vectors)
- ✅ Local policy enforcement functionality
- ✅ State machine transitions and audit trail recording
- ❌ **NOT** multi-machine failure domain isolation
- ❌ **NOT** real network partition injection
- ❌ **NOT** independent host/filesystem/OS failure boundaries
- ❌ **NOT** production-grade qualification

## What This Means

**All files in this directory represent dev-cluster validation only.** They are not production sign-off or real multi-machine qualification (P1_CORE).

Real multi-machine qualification (v1.0 target) requires:
- ≥2 independent Linux hosts with distinct isolation boundaries
- Real network failure injection (packet loss, latency, partition)
- Storage failure scenarios (disk full, corruption, I/O errors)
- Independent operator domains
- Chaos testing under sustained load

## Files

- `SIGN-OFF-PACKAGE-v1.0.0.md` - Dev-cluster security review (not production)
- `DEPLOYMENT-READINESS-SUMMARY.md` - Dev-cluster deployment notes
- `QUALIFICATION-VERDICT-FINAL.md` - Dev-cluster qualification result
- `RELEASE-DEPLOYMENT-PLAN.md` - Dev-cluster deployment phases
- Artifact tarballs and evidence digests

## See Also

- `docs/ROADMAP.md` - v0.2 and v1.0 goals
- `README.md` - Current v0.1 scope and features
- `.github/workflows/qualification.yml` - Notes on why M7 chaos testing not in CI
