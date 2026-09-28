# Remaining qualification and product phases

These phase briefs come from the supplied remaining-phases pack. Its
`PACK-MANIFEST.sha256` verified before integration. The pack was prepared
against `841da8e`; this directory is an implementation roadmap, **not**
evidence that any phase is qualified, verified, or sealed. Check the current
source and campaign records before changing a phase status.

| Order | Brief | Required observation |
|---|---|---|
| 0 | [P1 closure](phases/00-P1-CLOSE.md) | Three QEMU guests and all decisive local gates |
| 1 | [Failure and recovery](phases/01-P1-FAILURE-A01.md) | Real injections and measured restoration |
| 2 | [Mesh](phases/02-P1-MESH-A01.md) | Signed membership and partition convergence |
| 3 | [Evidence](phases/03-P1-EVIDENCE-A01.md) | Signed records, fresh verification and tamper rejection |
| 4 | [Independent hosts](phases/04-P2-MULTIHOST-A01.md) | At least three independently controlled physical hosts |
| 5 | [Marketplace](phases/05-P2-MARKET-A01.md) | Auditable offers, placement and verified usage |
| 6 | [Settlement](phases/06-P3-ECONOMY-A01.md) | Optional payment adapters bound to evidence |
| 7 | [DePIN adapters](phases/07-P4-DEPIN-ADAPTERS-A01.md) | Attributed external evidence |
| 8 | [Federation](phases/08-P5-FEDERATION-A01.md) | Cross-operator trust and revocation |
| 9 | [Production hardening](phases/09-PROD-HARDEN-A01.md) | Security review, recovery and measured scale |

The [gate catalog](qualification/gates.yaml) and [proposed evidence layout](qualification/evidence-layout.txt)
are planning aids. The repository's current `pkg/evidence` contract and `dh
evidence verify` remain the implementation of signed validation records.
The pack's shell manifest scripts and JSON schema were not installed as an
alternative verifier: they do not enforce the existing signing contract, and
the manifest verifier does not reject unlisted artifacts.

The current `p1-qualification-master.sh` runs only through the HTTP smoke
and workload baseline. It must not report the whole P1 campaign as qualified.
Known WSL evidence showed guest SSH readiness 0/3; the root cause and all
downstream gates remain unobserved from this workspace.

Decisive outcomes are `PASS`, `FAIL`, `BLOCKED`, or `UNKNOWN`. A phase is not
promoted by merging its code or documentation. Three VMs on one host remain
one physical host. `AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED`,
with allocation carved from reservation where applicable; cordon remains
orthogonal to lifecycle state.
