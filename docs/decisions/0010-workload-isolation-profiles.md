# 0010 — Workload isolation profiles

**Status:** accepted

## Context

Wave 1 ran workloads with the `process` runtime (a direct child of the agent,
no isolation) or `docker`. To run a workload the owner trusts less than their
own code — a community or, later, a marketplace workload — the host needs a
kernel boundary around it, and the boundary a host is willing to provide must
be explicit and enforced, not assumed.

## Decision

Three named isolation profiles, set per service in the manifest
(`spec.isolation`) and carried on the assignment. They select the sandbox
runtime (`pkg/runtime/sandbox`), which runs a BLAKE3-verified artifact:

- **PRIVATE** — the owner's own workload. New user, mount, PID, IPC and UTS
  namespaces; a pivoted read-only root holding only the system directories,
  the artifact and declared volumes; a private `/tmp` and `/dev`; a seccomp
  filter; an emptied capability set with `no_new_privs`; cgroup limits when
  given. Shares the host network (it binds its own port).
- **RESTRICTED** — a less-trusted workload. PRIVATE plus a private network
  namespace (loopback only; its one declared port is relayed by the agent
  through `setns`), and mandatory memory and PID limits.
- **UNTRUSTED** — arbitrary code from a marketplace. **Refused.** Namespaces
  and seccomp share the host kernel, which is not a boundary against a kernel
  exploit. UNTRUSTED becomes available only when a qualified hostile-code
  boundary (a microVM such as Firecracker or Kata, or gVisor) is implemented
  and tested. It is never silently downgraded to a container.

Enforcement:

- The manifest validator accepts `PRIVATE`/`RESTRICTED`, refuses `UNTRUSTED`
  with the reason, and rejects isolation on the `docker` runtime.
- Host admission (`pkg/policy`) **fails closed**: an isolation request on a
  host that cannot sandbox is denied (`POLICY_ISOLATION`), never run without
  the boundary. A host's `policy.yaml` can further restrict which profiles it
  accepts (`allowIsolation`), and the control plane cannot change that.
- The mount policy validates every mount before start: sources must be under
  an agent-managed root, targets may not cover system paths.

The seccomp filter is built in-house as classic BPF from per-architecture
syscall numbers (amd64, arm64), so there is no new dependency; UNTRUSTED aside,
this is the isolation ADR 0007's spec draft called for.

## Consequences

- A host without user namespaces, cgroups or root cannot offer isolated
  workloads; it reports so, and such workloads are refused there.
- The workload runs as an unprivileged uid (1000) when a subordinate uid
  range (`/etc/subuid`) is configured, otherwise as namespace-root mapped to
  the agent's uid with every capability dropped — still confined, but a real
  production host should configure a subuid range for the extra layer.
- No hostile multi-tenant execution is claimed until UNTRUSTED exists; the
  marketplace (P2) cannot admit arbitrary code before then.

## Evidence

`RUNTIME-P0-A01` (`pkg/runtime/sandbox` tests: escape, memory, PID, network,
mount policy, UNTRUSTED refusal) and the end-to-end
`tests/integration/runtime_p0_test.go` (a RESTRICTED workload runs and passes
its health check through the relay; UNTRUSTED refused; a host policy that
forbids the profile refuses the workload).
