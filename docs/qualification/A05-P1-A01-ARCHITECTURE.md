# A05-P1-A01 PHASE 1 — EPHEMERAL SECRET DELIVERY ARCHITECTURE

**Date:** 2026-09-26T14:30:00Z  
**Phase:** Architecture & Design  
**Scope:** SecretMaterializer component for ephemeral tmpfs allocation and lifecycle management

---

## 1. PHASE 1 OBJECTIVES

Phase 1 delivers the smallest missing production primitive: **SecretMaterializer**, integrated into the existing runtime/sandbox path.

### 1.1 Component Scope

**What SecretMaterializer does:**
- Allocates ephemeral tmpfs mount on control-plane response to secret retrieval request
- Tracks ephemeral lifecycle (create → workload-start → workload-run → workload-stop → cleanup)
- Guarantees cleanup via timeout mechanism (fallback if agent crashes)
- Prevents cross-workload visibility through namespace isolation
- Integrates with runtime.Spec and sandbox engine without major architectural changes

**What SecretMaterializer does NOT do (Phase 2+):**
- Lifecycle management service (that's Phase 2)
- Control-plane FSM state tracking (Phase 2)
- Complete integration with RESTRICTED isolation profiles (Phase 3)
- Adversarial escape prevention testing (Phase 4)

### 1.2 Integration Architecture

```
Agent Request (signed, with ephemeralID)
  ↓
Control Plane handleRetrieveSecret()
  ├─ Verify authorization (SEC-P0-A01 ✓)
  ├─ Decrypt secret (A03 ✓)
  ├─ [NEW] Allocate ephemeral tmpfs
  ├─ [NEW] Write plaintext to ephemeral
  └─ Return ephemeralPath (not plaintext)
  ↓
Agent receives ephemeralPath
  ├─ Pass to runtime.Spec.EphemeralMounts
  ├─ Call rt.Start(spec)
  └─ Runtime mounts ephemeral into namespace
  ↓
Workload reads from /run/secrets/...
  ↓
On workload stop:
  ├─ Runtime unmounts ephemeral
  ├─ Agent verifies cleanup
  └─ Report to control plane
```

---

## 2. PHASE 1 DELIVERABLES

### 2.1 Code Changes (Minimal, Focused)

#### 2.1.1 Runtime Spec Extension

**File:** `pkg/runtime/runtime.go`

```go
type Mount struct {
    Name       string // "ephemeral_<id>" or persistent mount name
    HostPath   string // /var/run/secrets/ASSIGNMENT/ephemeral_...
    Path       string // /run/secrets/... (container path)
    ReadOnly   bool
    Ephemeral  bool   // [NEW] Marks as ephemeral tmpfs
}

type Spec struct {
    // ... existing fields ...
    EphemeralMounts []Mount // [NEW] Separate tracking for ephemeral vs. persistent
    EphemeralID     string  // [NEW] UUID for this allocation
}
```

#### 2.1.2 SecretMaterializer Component

**File:** `pkg/runtime/secret_materializer.go` (NEW)

```go
package runtime

import (
    "context"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "time"
)

// Materializer handles ephemeral tmpfs allocation and lifecycle
type Materializer struct {
    basePath string // /var/run/secrets
    timeout  time.Duration
}

// AllocateEphemeral creates tmpfs mount and writes secret
// Returns: path on host, error
func (m *Materializer) AllocateEphemeral(ctx context.Context, 
    assignmentID, ephemeralID string, 
    plaintext []byte, 
    size int64) (hostPath string, err error) {
    
    // 1. Create assignment directory if needed
    assignDir := filepath.Join(m.basePath, assignmentID)
    if err := os.MkdirAll(assignDir, 0755); err != nil {
        return "", fmt.Errorf("mkdir assignment dir: %w", err)
    }
    
    // 2. Create ephemeral file path (never reused)
    ts := time.Now().Unix()
    ephemeralFile := filepath.Join(assignDir, 
        fmt.Sprintf("ephemeral_%s_%d", ephemeralID, ts))
    
    // 3. Create tmpfs mount at this path
    // (Implementation deferred to sandbox engine)
    // For now: create regular file, Phase 2 adds tmpfs mount
    
    if err := ioutil.WriteFile(ephemeralFile, plaintext, 0400); err != nil {
        return "", fmt.Errorf("write ephemeral file: %w", err)
    }
    
    return ephemeralFile, nil
}

// ReleaseEphemeral removes ephemeral file and verifies cleanup
func (m *Materializer) ReleaseEphemeral(ctx context.Context, 
    hostPath string) error {
    
    if err := os.Remove(hostPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("remove ephemeral: %w", err)
    }
    
    // Verify removal (stat should return ENOENT)
    if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
        return fmt.Errorf("ephemeral still exists after removal: %s", hostPath)
    }
    
    return nil
}
```

#### 2.1.3 Sandbox Runtime Integration

**File:** `pkg/runtime/sandbox_runtime.go`

Add ephemeral mount support to SandboxRuntime.Start():

```go
// In SandboxRuntime.Start():
// 1. Pass spec.EphemeralMounts to sandbox config
// 2. Sandbox engine binds ephemeral at container path (read-only)
// 3. Track ephemeral ID in Instance for cleanup

// New Instance field:
type Instance struct {
    // ... existing fields ...
    EphemeralID string // For cleanup tracking
}
```

#### 2.1.4 Control Plane Secret Retrieval Enhancement

**File:** `pkg/control/hostapi.go`

Extend handleRetrieveSecret to allocate ephemeral:

```go
// At line 403, after DecryptSecret():
// [NEW] Instead of returning plaintext in HTTP response:

// 1. Get ephemeralID from request
// 2. Call materializer.AllocateEphemeral(plaintext)
// 3. Return ephemeralPath in response
// 4. Schedule cleanup timer (control-plane side fallback)

// Request structure (extend SecretRetrievalRequest):
type SecretRetrievalRequest struct {
    // ... existing fields (signature, scope, etc.) ...
    EphemeralID     string // [NEW] UUID from agent
    EphemeralPath   string // [NEW] Requested path on host
}

// Response structure (modify SecretRetrievalResponse):
type SecretRetrievalResponse struct {
    // OLD: "plaintext": ...  ← REMOVED
    EphemeralPath   string // [NEW] Path to ephemeral tmpfs
    EphemeralID     string
    Digest          string // Request signature for audit
}
```

### 2.2 Test Coverage (Phase 1 Validation)

#### 2.2.1 Unit Tests

- `TestMaterializer_AllocateEphemeral` — tmpfs creation, file permissions, isolation
- `TestMaterializer_ReleaseEphemeral` — cleanup verification, re-allocation safety
- `TestMaterializer_Concurrent` — multiple concurrent allocations, no cross-contamination

#### 2.2.2 Integration Tests

- `TestA05_SecretRetrievalReturnsEphemeralPath` — Control plane responds with path, not plaintext
- `TestA05_EphemeralMountedInNamespace` — Workload can read from /run/secrets/...
- `TestA05_CleanupAfterStop` — Ephemeral removed after workload stops

#### 2.2.3 Isolation Verification

- `TestA05_SiblingWorkloadsIsolated` — Two workloads get different ephemeral paths, can't read each other's secrets
- `TestA05_EphemeralNotInEnvironment` — Plaintext not leaked via $ENV
- `TestA05_EphemeralNotInProcessListing` — Plaintext not visible in /proc/*/environ

---

## 3. LIFECYCLE MANAGEMENT DESIGN

### 3.1 Ephemeral Lifecycle States

```
CREATED
  ├─ File written to /var/run/secrets/ASSIGNMENT/ephemeral_*
  ├─ Permissions: 0400 (root:root)
  └─ Deadline: now + 5min (control-plane timeout)
  ↓
MOUNTED (in workload namespace)
  ├─ Bind-mounted at /run/secrets/ASSIGNMENT/ephemeral_* (read-only)
  ├─ Visible only inside workload namespace
  └─ Accessible only to workload user
  ↓
CLEANUP_SCHEDULED (on workload stop or timeout)
  ├─ Unmount from namespace
  ├─ Remove file from host
  └─ Report status to control plane
  ↓
CLEANED
  └─ Ephemeral verified removed, no recovery possible
```

### 3.2 Lifecycle Ownership

**Agent owns workload process:**
- Calls rt.Start() with ephemeral paths
- Detects workload exit via rt.Status()
- Calls rt.Stop() to trigger cleanup

**Runtime engine owns mount lifecycle:**
- Mounts ephemeral at container start
- Unmounts ephemeral at container stop
- Reports mount/unmount success to agent

**Control plane owns timeout fallback:**
- Tracks ephemeral deadline (created + 5min)
- Background goroutine checks deadlines
- Removes stale ephemeral if agent fails

### 3.3 Pre-Start / Post-Start Hooks

**Agent-side (in startWorkload):**

```go
// PRE-START: Before rt.Start()
// 1. Request ephemeral allocation from control plane
// 2. Receive ephemeralID + hostPath
// 3. Add to spec.EphemeralMounts

// POST-START: After rt.Start()
// 1. Verify Instance created
// 2. Store ephemeralID in Instance for cleanup tracking
// 3. (Phase 2: ping control plane to confirm mounted)
```

**Runtime-side (in sandbox engine):**

```go
// IN-START: During rt.Start()
// 1. Create tmpfs at /var/run/secrets/ASSIGNMENT/ephemeral_*
// 2. Write plaintext to file (control plane provided)
// 3. chmod 0400
// 4. Bind-mount into namespace at /run/secrets/...
// 5. Mount as read-only (MS_REMOUNT | MS_RDONLY)
```

---

## 4. PLAINTEXT HANDLING & SECURITY

### 4.1 Plaintext Surfaces (Phase 1 Scope)

**ELIMINATED (no longer plaintext in HTTP):**
- ✓ HTTP Response (control plane → agent)

**EXISTING (pre-A05):**
- HTTP request (agent → control plane) — signed, uses TLS
- Control plane memory during decryption — unavoidable, SEC-P0 accepts
- Agent logs — avoid logging plaintext (Phase 2 audit)

**NEW (Phase 1):**
- Ephemeral file on host `/var/run/secrets/...` — tmpfs mount, isolated
- In-container view of `/run/secrets/...` — only accessible to workload namespace

### 4.2 Canary Tests (Phase 1 Verification)

```go
TestA05_EphemeralNotInHostLogs() 
  // Verify plaintext NOT in agent/control-plane logs

TestA05_EphemeralNotInHTTPResponse()
  // Verify response contains only ephemeralPath, not plaintext

TestA05_EphemeralNotInProcessEnvironment()
  // Verify plaintext NOT visible via /proc/PID/environ

TestA05_EphemeralNotAccessibleOutsideNamespace()
  // Host can't read /run/secrets/* (it's in container namespace)
```

---

## 5. PHASE 1 IMPLEMENTATION PLAN

### 5.1 Order of Implementation

1. **runtime.go changes** — Add Mount.Ephemeral, Spec.EphemeralMounts, Spec.EphemeralID
2. **secret_materializer.go** — New component, AllocateEphemeral + ReleaseEphemeral
3. **sandbox_runtime.go** — Pass ephemeral mounts to sandbox config
4. **sandbox_linux.go** — Handle ephemeral mount binding in buildRoot() or post-start hook
5. **hostapi.go changes** — Return ephemeralPath instead of plaintext, call materializer
6. **Test suite** — Unit + integration + isolation verification

### 5.2 Commit Structure

- Commit 1: `runtime: Add ephemeral mount types`
- Commit 2: `runtime: Implement SecretMaterializer component`
- Commit 3: `sandbox: Integrate ephemeral mounts into sandbox runtime`
- Commit 4: `control: Modify secret retrieval to return ephemeral path`
- Commit 5: `test: Add A05 Phase 1 validation suite`

### 5.3 Testing Before Merge

```bash
# Unit tests
go test ./pkg/runtime -v -run TestMaterializer

# Integration tests  
go test ./pkg/control -v -run TestA05_SecretRetrieval
go test ./pkg/node -v -run TestA05_Ephemeral

# Sandbox tests
go test ./pkg/runtime/sandbox -v -run TestA05

# Race detector
go test -race ./pkg/runtime ./pkg/control ./pkg/node
```

---

## 6. KNOWN CONSTRAINTS & DEFERMENTS

### 6.1 Phase 1 Constraints

1. **tmpfs mount implementation deferred** — For now create regular files, Phase 2 adds actual tmpfs mounts
   - Reason: tmpfs creation requires root/unshare, defer to sandbox engine enhancement
   
2. **FSM state tracking deferred** — Control plane doesn't yet persist ephemeral state
   - Reason: Requires FSM schema change, defer to Phase 2 (depends on control plane design)
   
3. **Cleanup timeout mechanism deferred** — No background goroutine yet for timeout-based cleanup
   - Reason: Phase 1 focuses on per-workload lifecycle, Phase 2 adds control plane fallback

4. **Agent-control plane cleanup handshake deferred** — No explicit cleanup signal API yet
   - Reason: Phase 1 assumes workload stop → cleanup, Phase 2 adds explicit signals

### 6.2 Phase 1 Success Criteria

✓ Plaintext no longer in HTTP response (ephemeralPath returned instead)  
✓ Ephemeral file created at control-plane specified path  
✓ Ephemeral mounted read-only in workload namespace  
✓ Ephemeral unmounted and removed on workload stop  
✓ Sibling workloads isolated (different ephemeral paths, can't cross-read)  
✓ No plaintext in agent/control-plane logs  
✓ All Phase 1 unit/integration tests pass  

### 6.3 Phase 1 Known Gaps (Not Blockers)

- ✗ Cleanup timeout if agent crashes (Phase 2: control plane fallback)
- ✗ FSM state persistence (Phase 2: control plane tracking)
- ✗ Explicit cleanup handshake (Phase 2: lifecycle API)
- ✗ Audit logging of ephemeral lifecycle (Phase 2: audit trail)

---

## 7. ARCHITECTURE DECISION RATIONALE

### Why SecretMaterializer (not global secret service)?

✓ Minimal change to existing runtime/sandbox path  
✓ Decouples secret delivery from lifecycle service  
✓ Ephemeral is instance-local, not service-global  
✓ Testable in isolation before full lifecycle integration  

### Why ephemeral files (not process memory map)?

✓ Workload can read via standard file I/O  
✓ Kernel manages mount namespace isolation  
✓ Cleanup guaranteed by filesystem unmount  
✓ No need for IPC/shared memory coordination  

### Why control plane allocates (not agent)?

✓ Control plane owns secret decryption (SEC-P0)  
✓ Agent never sees plaintext  
✓ Audit trail tied to authorization decision  
✓ Simplifies agent code (no materializer logic)  

---

## 8. NEXT STEPS

### Immediate (Phase 1)

1. Implement SecretMaterializer component
2. Extend runtime.Spec with EphemeralMounts
3. Modify hostapi.go to allocate ephemeral and return path
4. Extend sandbox runtime to bind ephemeral in namespace
5. Implement Phase 1 test suite
6. Validate all 5 success criteria

### Following (Phase 2)

- FSM state tracking for ephemeral mounts
- Cleanup timeout mechanism (control plane background goroutine)
- Explicit cleanup handshake API (agent → control plane)
- Audit trail for ephemeral lifecycle events

### Following (Phase 3+)

- RESTRICTED isolation profile enhancements
- Network namespace for ephemeral delivery
- Adversarial escape prevention testing
- Full P0-SOVEREIGN integration

---

**Phase 1 Status:** READY FOR IMPLEMENTATION  
**Depends On:** Phase 0 discovery (complete ✓)  
**Blocks:** Phase 2 (FSM state tracking)  
**Critical Path:** SecretMaterializer → sandbox integration → test validation → merge

