# Evidence

A capability is promoted only by a signed validation record: evidence that
someone else can re-check. This page defines the record, the source digest,
outcomes, verification and attempt lineage (`pkg/evidence`, `dh evidence …`).

## Commands

| Command | Purpose |
|---|---|
| `dh evidence digest [--src DIR]` | source-tree identity (algorithm below) |
| `dh evidence pack --out FILE [--src DIR]` | a deterministic `.tar.gz` of exactly the digested files; use it to move a tree to a validation machine (never hand-written `tar` excludes) |
| `dh evidence env [--data DIR]` | the measured environment of this machine |
| `dh evidence run --dir EVID --name STEP [--retries N] -- CMD…` | run one step: log it, time it, record exit code and log hash; retries are counted, never hidden |
| `dh evidence seal --dir EVID --id ID --stage S --claim … --scope … --require a,b,…` | derive the outcome and sign the record; a sealed directory can never be sealed again |
| `dh evidence verify --dir EVID` | check the signature and re-hash every artifact; prints the outcome and the verification separately |

## Record (schema `dh-validation/2`, signing domain `validation-record`)

| Field | Meaning |
|---|---|
| `schemaVersion`, `evidenceId`, `attemptId`, `parentEvidenceId` | identity and lineage, e.g. `PV1-S1-A03` with parent `PV1-S1-A02` |
| `stage`, `claim`, `scope` | what is claimed and in which environment, in plain words |
| `limitations`, `scopeExclusions` | what the evidence does not show |
| `digestVersion`, `sourceDigest`, `sourceFiles`, `commit` | exact source (`commit` is empty without version control) |
| `binaries` | BLAKE3 of the binaries under test |
| `machines` | measured environment of every machine: OS, kernel, distribution, architecture, CPU model and count, memory, data filesystem and disk, network addresses, **container** and **virtualization** |
| `requiredSteps`, `steps` | every command with start and end times, duration, exit code, attempts and log hash |
| `files` | every artifact in the evidence directory with its size and BLAKE3 hash |
| `outcome`, `outcomeReason` | the execution outcome, derived mechanically (below) |

**Privacy note:** `machines[].addresses` records real network addresses.
Redact them before publishing if they matter. Redaction changes the hashes,
so publish the redacted copy under a new record.

## Outcome and verification are different facts

**Outcome** (stored, derived by `seal`):
- `PASS`: every required step ran and exited 0.
- `FAIL`: a step failed.
- `INCOMPLETE`: a required step never ran.
- `INFRA_FAILURE`: the operator states the infrastructure prevented the run.
  This is only accepted when steps are missing, so it can never hide a
  completed failure.

**Verification** (never stored, computed by `verify`):
- `VERIFIED`: the signature is valid and every artifact still matches its hash.
- `UNVERIFIED`: either check fails.

`outcome: PASS` in a JSON file is not proof. `PASS` together with
`VERIFIED` is.

## Source digest (`dh-src-digest/2`)

1. Walk the tree. Skip `.git`, `__pycache__` and `node_modules` at any depth,
   and `bin`, `evidence`, `chaos-reports`, `devcluster` and `tools/bin` **at
   the root only**.
2. Keep regular files with extensions `.go .py .js .css .html .md .json
   .mod .sum .sh .yaml .yml .txt`, files named `Makefile`, and every symbolic
   link. Links are recorded, never followed.
3. Emit one line per kept path, relative to the root with `/` separators:
   - `f NUL path NUL mode NUL b3hex(content)` for a file, where `mode` is `x`
     if any execute bit is set and `-` otherwise;
   - `l NUL path NUL target` for a symbolic link.
4. Sort the lines bytewise, join them with `\n`, and return `b3:` +
   BLAKE3-256 of the result.

Timestamps, ownership, other mode bits and empty directories do not count.
The same tree copied to another machine gives the same digest. `dh evidence
pack` round-trips it (both tested in `pkg/evidence`).

**History of the algorithm.**
- `dh-src-digest/0` (PV1-S1-A02): no type or mode fields.
- `dh-src-digest/1` (REF-MAC-A01): skipped any directory named `bin` or
  `evidence` at any depth, so it **silently excluded the source package
  `pkg/evidence`**. This was found when a bundle of REF-MAC-A01's tree failed
  to build.
- Digests from different versions are not comparable.

## Attempts are immutable

- One directory is one attempt. `seal` refuses a directory that is already
  sealed, and the stage script refuses a directory that already has steps.
- Never patch code during an attempt. Fix, compute the new digest, and start
  a new attempt whose `parentEvidenceId` names the failed one.
- Failed, incomplete and infrastructure-failure attempts stay in `evidence/`
  with their history file, for example `evidence/PV1-S1-HISTORY.md`.
