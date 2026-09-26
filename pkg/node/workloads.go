package node

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/policy"
	"decentralized.host/pkg/runtime"
)

const maxRestarts = 5

// reconcileWorkloads applies admission to every assignment in the verified
// bundle and converges the local runtime toward what was admitted.
func (a *Agent) reconcileWorkloads() {
	a.mu.RLock()
	b, trusted, trustWhy, fresh, skew := a.bundle, a.trusted, a.trustWhy, a.fresh, a.skewMs
	a.mu.RUnlock()

	seen := map[string]bool{}
	if b != nil {
		for _, env := range b.Assignments {
			as, sigOK, sigWhy := a.verifyAssignment(b, env)
			if as.Node != a.st.NodeID && as.Node != "" && sigOK {
				// Signed for another host: refuse regardless of anything else.
				a.decide(as, policy.Decision{Code: policy.CodeWrongHost, Reason: "assignment names another host"}, nil)
				continue
			}
			seen[as.ID] = true
			a.handleAssignment(b, env, as, sigOK, sigWhy, trusted, trustWhy, fresh, skew)
		}
	}
	// Admitted work that is no longer in a fresh, trusted desired state is
	// stopped; while the plane is stale or untrusted it is held.
	a.mu.RLock()
	ids := sortedKeys(a.st.Admitted)
	a.mu.RUnlock()
	for _, id := range ids {
		a.mu.RLock()
		ad := a.st.Admitted[id]
		a.mu.RUnlock()
		if ad == nil || seen[id] || ad.Stopped {
			continue
		}
		if trusted && fresh && b != nil && !b.Frozen {
			a.stopWorkload(ad, "no longer in the signed desired state")
			continue
		}
		if a.pol.OfflineAdmission == "stop" && !fresh {
			a.stopWorkload(ad, "control plane stale and host policy offlineAdmission=stop")
			continue
		}
		a.supervise(ad, true)
	}
	// Forget finished records.
	a.mu.Lock()
	var forgotten []string
	for id, ad := range a.st.Admitted {
		if ad.Stopped && !seen[id] {
			delete(a.st.Admitted, id)
			forgotten = append(forgotten, id)
		}
	}
	a.mu.Unlock()
	for _, id := range forgotten {
		a.closeForward(id)
	}
	// Offline policy "stop" also applies to workloads still listed in a stale bundle.
	if a.pol.OfflineAdmission == "stop" && !fresh {
		for _, id := range ids {
			a.mu.RLock()
			ad := a.st.Admitted[id]
			a.mu.RUnlock()
			if ad != nil && !ad.Stopped {
				a.stopWorkload(ad, "control plane stale and host policy offlineAdmission=stop")
			}
		}
	}
}

func (a *Agent) capabilityCheck(root string, as api.Assignment) (bool, string) {
	tok, err := capability.Decode(as.Capability)
	if err != nil {
		return false, err.Error()
	}
	res := capability.Verify(tok, []string{root}, capability.Request{
		Action: "workload.admit", Resource: "app/" + as.ID, Audience: a.st.NodeID, Now: a.now(),
		Generation: as.Generation, Digest: as.Digest, CPUMilli: as.Resources.CPUMilli, MemBytes: as.Resources.MemBytes,
	})
	if !res.OK {
		return false, fmt.Sprintf("%s (block %d)", res.Reason, res.Block)
	}
	return true, fmt.Sprintf("chain of %d block(s) anchored in the pinned root", len(tok.Blocks))
}

func (a *Agent) attested(b *api.Bundle, digest string) (bool, string) {
	if !a.pol.RequireImageSignature {
		return true, "not required by host policy"
	}
	var trustedKeys []string
	for _, k := range a.pol.TrustedPublishers {
		if k == "cluster-root" {
			trustedKeys = append(trustedKeys, a.st.Root)
		} else {
			trustedKeys = append(trustedKeys, k)
		}
	}
	for _, env := range b.Attestations {
		var at api.ArtifactAttestation
		if env.Decode(&at) != nil || at.Digest != digest {
			continue
		}
		if env.VerifyKey(envelope.KindArtifact, trustedKeys...) == nil {
			return true, "attested by trusted publisher " + short(env.Pub)
		}
	}
	return false, "no attestation for " + digest + " from a host-trusted publisher"
}

func (a *Agent) handleAssignment(b *api.Bundle, env *envelope.Envelope, as api.Assignment, sigOK bool, sigWhy string, trusted bool, trustWhy string, fresh bool, skew int64) {
	a.mu.RLock()
	prior := a.st.Admitted[as.ID]
	a.mu.RUnlock()
	if trusted && trustWhy == "" {
		trustWhy = "roster and bundle signer chain to the pinned root; state index not rolled back"
	}
	in := policy.Input{Policy: a.pol, NodeID: a.st.NodeID, Assignment: as, SignatureOK: sigOK, SignatureDetail: sigWhy,
		PlaneTrusted: trusted, PlaneDetail: trustWhy, Fresh: fresh, Frozen: b.Frozen, ClockSkewMs: skew,
		Revoked: contains(b.Revoked, a.st.NodeID) || b.Node.Status == "revoked", LedgerCorrupt: a.journal.Corrupt() != nil}
	in.CapabilityOK, in.CapabilityWhy = a.capabilityCheck(a.st.Root, as)
	in.Attested, in.AttestDetail = a.attested(b, as.Digest)
	in.SandboxOK, in.SandboxDetail = a.sandbox.Available()
	if prior != nil && !prior.Stopped {
		in.PriorGeneration = prior.Generation
		in.PriorRunning = a.runtimeFor(prior.Runtime).Status(prior.Inst).State == "running"
	}
	if prior != nil && prior.Stopped && prior.Generation > in.PriorGeneration {
		in.PriorGeneration = prior.Generation
	}
	a.mu.RLock()
	for id, ad := range a.st.Admitted {
		if id == as.ID || ad.Stopped {
			continue
		}
		in.UsedWorkloads++
		in.UsedCPUMilli += ad.CPUMilli
		in.UsedMemBytes += ad.MemBytes
	}
	a.mu.RUnlock()
	if as.Runtime == "docker" && !a.dockerOK.Load() {
		in.Policy.AllowRuntimes = without(in.Policy.AllowRuntimes, "docker")
	}
	d := policy.Admit(in)
	a.decide(as, d, prior)

	switch {
	case d.Code == policy.CodeStop:
		if prior != nil && !prior.Stopped {
			a.stopWorkload(prior, "signed stop for generation "+fmt.Sprint(as.Generation))
		}
	case d.Hold:
		if prior != nil {
			a.supervise(prior, true)
		}
	case d.Allowed:
		if prior != nil && !prior.Stopped && prior.Generation == as.Generation && (prior.Inst.PID != 0 || prior.Inst.ContainerID != "" || prior.LastState == "failed") {
			a.mu.Lock()
			prior.Volumes = as.Volumes
			a.mu.Unlock()
			a.supervise(prior, false)
			return
		}
		// Fetch and verify the artifact before touching an older generation,
		// so a rolling update never stops working code for a missing binary.
		if as.Runtime == "process" {
			if _, ready := a.artifactReady(b, as.Image, as.Digest); !ready {
				a.noteStarting(as, "admitted; fetching and verifying artifact "+short(as.Digest))
				if prior != nil && !prior.Stopped {
					a.supervise(prior, true)
				}
				return
			}
		}
		raw, _ := json.Marshal(as)
		if prior != nil && !prior.Stopped && prior.Generation < as.Generation {
			if overlapSafe(prior, as) && a.runtimeFor(prior.Runtime).Status(prior.Inst).State == "running" {
				// Make-before-break: the old instance keeps serving until
				// the new one passes its health check.
				a.startWorkload(b, as, raw, prior)
				return
			}
			a.stopWorkload(prior, fmt.Sprintf("replaced by generation %d", as.Generation))
		}
		a.startWorkload(b, as, raw, nil)
	default:
		// Refused. Nothing new starts. A previously admitted instance (an
		// older generation, or one replaced by a stale/forged assignment)
		// keeps running under the admission it already has.
		if prior != nil && !prior.Stopped {
			a.supervise(prior, true)
		}
	}
}

func without(list []string, v string) []string {
	var out []string
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// decide records the decision and journals it when it changes.
func (a *Agent) decide(as api.Assignment, d policy.Decision, prior *Admitted) {
	key := fmt.Sprintf("%d|%v|%v|%s", as.Generation, d.Allowed, d.Hold, d.Code)
	a.mu.Lock()
	a.decisions[as.ID] = &decision{Gen: as.Generation, Allowed: d.Allowed, Hold: d.Hold, Code: d.Code, Reason: d.Reason, Checks: d.Checks, At: a.now()}
	changed := a.st.Decisions[as.ID] != key
	a.st.Decisions[as.ID] = key
	a.mu.Unlock()
	if !changed {
		return
	}
	action := "admission-refuse"
	switch {
	case d.Code == policy.CodeStop:
		action = "admission-stop"
	case d.Hold:
		action = "admission-hold"
	case d.Allowed:
		action = "admission-allow"
	}
	a.record(action, "app/"+as.ID, as.Generation, fmt.Sprintf("%s: %s", d.Code, d.Reason))
}

func (a *Agent) allocMeshPort(id string) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if p, ok := a.st.MeshPorts[id]; ok {
		return p
	}
	p := a.st.NextMesh
	a.st.NextMesh++
	a.st.MeshPorts[id] = p
	return p
}

// effRuntime is the runtime backend for an assignment: the sandbox when an
// isolation profile is set, otherwise the declared runtime.
func effRuntime(as api.Assignment) string {
	if as.Isolation != "" {
		return "sandbox"
	}
	return as.Runtime
}

// overlapSafe reports whether two generations of a replica may run at the
// same time: only without volumes (two writers must never share one) and
// only for workloads that serve a port (otherwise nothing is handed over).
func overlapSafe(prior *Admitted, as api.Assignment) bool {
	return len(prior.Volumes) == 0 && len(as.Volumes) == 0 && len(prior.Ports) > 0 && len(as.Ports) > 0 && prior.Inst.Port > 0
}

// startWorkload starts an admitted assignment. retire, if set, is the older
// generation of the same replica, still serving; it is stopped only after
// the new instance is ready and the mesh forwarder has moved to it.
func (a *Agent) startWorkload(b *api.Bundle, as api.Assignment, raw []byte, retire *Admitted) {
	spec := runtime.Spec{Assignment: as.ID, App: as.App, Node: a.st.NodeID, Generation: as.Generation, Replica: as.Replica,
		Args: as.Command, Env: as.Env, CPUMilli: as.Resources.CPUMilli, MemBytes: as.Resources.MemBytes, WantPort: len(as.Ports) > 0,
		WorkDir: filepath.Join(a.cfg.DataDir, "work", safeName(as.ID)), LogPath: filepath.Join(a.cfg.DataDir, "logs", safeName(as.ID)+".log")}
	_ = os.MkdirAll(filepath.Dir(spec.LogPath), 0o700)
	if len(as.Ports) > 0 {
		spec.ContainerPort = as.Ports[0].Container
		if spec.ContainerPort == 0 {
			spec.ContainerPort = 8080
		}
	}
	for _, v := range as.Volumes {
		dir, err := a.prepareVolume(v)
		if err != nil {
			a.failStart(as, raw, "volume "+v.Name+": "+err.Error())
			return
		}
		spec.Mounts = append(spec.Mounts, runtime.Mount{Name: v.Name, HostPath: dir, Path: v.Mount})
	}
	switch effRuntime(as) {
	case "process", "sandbox":
		path, ready := a.artifactReady(b, as.Image, as.Digest)
		if !ready {
			a.noteStarting(as, "admitted; fetching and verifying artifact "+short(as.Digest))
			return
		}
		spec.Executable = path
		spec.Isolation = as.Isolation
	case "docker":
		spec.Image = as.Image
	}
	rt := a.runtimeFor(effRuntime(as))
	if retire != nil {
		a.meshMu.Lock()
		a.cutover[as.ID] = true
		a.meshMu.Unlock()
		a.mu.Lock()
		if a.st.Retiring == nil {
			a.st.Retiring = map[string]*Retiring{}
		}
		a.st.Retiring[as.ID] = &Retiring{Runtime: retire.Runtime, Inst: retire.Inst, Generation: retire.Generation}
		a.mu.Unlock()
	}
	inst, err := rt.Start(spec)
	if err != nil {
		if retire != nil {
			// The old generation keeps serving; nothing was handed over.
			a.meshMu.Lock()
			delete(a.cutover, as.ID)
			a.meshMu.Unlock()
			a.mu.Lock()
			delete(a.st.Retiring, as.ID)
			a.mu.Unlock()
		}
		a.failStart(as, raw, err.Error())
		return
	}
	ad := &Admitted{ID: as.ID, App: as.App, Replica: as.Replica, Generation: as.Generation, Runtime: effRuntime(as), Image: as.Image, Digest: as.Digest,
		CPUMilli: as.Resources.CPUMilli, MemBytes: as.Resources.MemBytes, Inst: inst, Health: as.Health, Ports: as.Ports, Volumes: as.Volumes,
		LastState: "running", AdmittedAt: a.now(), Assignment: raw}
	a.mu.Lock()
	if prev := a.st.Admitted[as.ID]; prev != nil && prev.Generation == as.Generation {
		ad.Restarts = prev.Restarts
	}
	a.st.Admitted[as.ID] = ad
	// A new instance is not healthy until it passes its own probe; a result
	// from the previous instance must never vouch for it.
	delete(a.health, as.ID)
	delete(a.healthAt, as.ID)
	a.mu.Unlock()
	where := fmt.Sprintf("pid %d", inst.PID)
	if inst.ContainerID != "" {
		where = "container " + short(inst.ContainerID)
	}
	a.record("workload-start", "app/"+as.ID, as.Generation, fmt.Sprintf("%s via %s runtime (%s); %s", as.Image, as.Runtime, where, rt.Limits()))
	switch {
	case retire != nil:
		go a.handover(as, ad, retire)
	case inst.Port > 0:
		a.ensureForward(as.ID, a.allocMeshPort(as.ID), inst.Port)
	}
}

// handover waits for the new instance to answer its health check (or accept
// a connection when it has none), moves the mesh forwarder to it, then stops
// the old generation. Connections already piped to the old instance finish
// on it.
func (a *Agent) handover(as api.Assignment, ad *Admitted, old *Admitted) {
	start := time.Now()
	ready, how := false, ""
	for time.Since(start) < 30*time.Second {
		if ad.Health.HTTP != "" {
			if obs := probeHTTP(fmt.Sprintf("http://127.0.0.1:%d%s", ad.Inst.Port, ad.Health.HTTP), time.Second); obs.OK {
				ready, how = true, "passed its health check ("+obs.Detail+")"
				break
			}
		} else if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", ad.Inst.Port), time.Second); err == nil {
			c.Close()
			ready, how = true, "accepted a connection"
			break
		}
		if st := a.runtimeFor(ad.Runtime).Status(ad.Inst); st.State != "running" {
			how = "exited before becoming ready: " + st.Detail
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready && how == "" {
		how = "did not become ready within 30s"
	}
	a.switchForward(as.ID, a.allocMeshPort(as.ID), ad.Inst.Port)
	// Requests accepted by the old instance get a moment to finish.
	time.Sleep(500 * time.Millisecond)
	_ = a.runtimeFor(old.Runtime).Stop(old.Inst, 5*time.Second)
	a.mu.Lock()
	delete(a.st.Retiring, as.ID)
	a.mu.Unlock()
	a.record("workload-stop", "app/"+as.ID, old.Generation, fmt.Sprintf("replaced by generation %d without a gap: new instance %s after %s; mesh traffic moved, then the old instance stopped",
		as.Generation, how, time.Since(start).Round(time.Millisecond)))
}

// noteStarting records that an admitted assignment is waiting on a
// prerequisite (the observation reports it as "starting").
func (a *Agent) noteStarting(as api.Assignment, why string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if d := a.decisions[as.ID]; d != nil && d.Gen == as.Generation {
		d.Reason = why
	}
}

// artifactReady returns the verified executable path, starting a background
// fetch when it is not yet cached. Fetch failures are journaled and retried.
func (a *Agent) artifactReady(b *api.Bundle, image, digest string) (string, bool) {
	path := filepath.Join(a.cfg.DataDir, "artifacts", strings.TrimPrefix(digest, "b3:"))
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	a.mu.Lock()
	if a.fetching == nil {
		a.fetching = map[string]time.Time{}
	}
	if t, busy := a.fetching[digest]; busy && time.Since(t) < 5*time.Minute {
		a.mu.Unlock()
		return "", false
	}
	a.fetching[digest] = time.Now()
	a.mu.Unlock()
	go func() {
		start := time.Now()
		p, err := a.fetchArtifact(b, image, digest)
		a.mu.Lock()
		delete(a.fetching, digest)
		a.mu.Unlock()
		if err != nil {
			a.record("artifact-fetch-fail", "artifact/"+short(digest), 0, fmt.Sprintf("%v (after %s)", err, time.Since(start).Round(time.Millisecond)))
			return
		}
		_ = p
	}()
	return "", false
}

func (a *Agent) failStart(as api.Assignment, raw []byte, why string) {
	a.mu.Lock()
	ad := a.st.Admitted[as.ID]
	if ad == nil || ad.Generation != as.Generation {
		ad = &Admitted{ID: as.ID, App: as.App, Replica: as.Replica, Generation: as.Generation, Runtime: effRuntime(as), Image: as.Image, Digest: as.Digest,
			CPUMilli: as.Resources.CPUMilli, MemBytes: as.Resources.MemBytes, Health: as.Health, Ports: as.Ports, Volumes: as.Volumes, AdmittedAt: a.now(), Assignment: raw}
		a.st.Admitted[as.ID] = ad
	}
	ad.LastState, ad.LastDetail = "failed", why
	ad.Restarts++
	ad.Backoff = a.now() + backoffMs(ad.Restarts)
	a.mu.Unlock()
	a.record("workload-fail", "app/"+as.ID, as.Generation, why)
}

func backoffMs(n int64) int64 {
	d := int64(1000) << uint(minI(n, 6))
	return d
}

func minI(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// supervise checks a running instance and restarts it (with backoff) if it
// died while still admitted. hold=true means the plane is not fresh: the
// instance is kept, but a dead one is only restarted if it was admitted
// under the same generation (it was), because restarting is not new work.
func (a *Agent) supervise(ad *Admitted, hold bool) {
	if ad.Stopped {
		return
	}
	rt := a.runtimeFor(ad.Runtime)
	st := rt.Status(ad.Inst)
	a.mu.Lock()
	prevState := ad.LastState
	if ad.Inst.PID != 0 || ad.Inst.ContainerID != "" {
		ad.LastState = st.State
		if st.State != "running" {
			ad.LastExit, ad.LastDetail = st.ExitCode, st.Detail
		}
	}
	a.mu.Unlock()
	if st.State == "running" {
		if ad.Inst.Port > 0 {
			a.ensureForward(ad.ID, a.allocMeshPort(ad.ID), ad.Inst.Port)
		}
		return
	}
	if prevState == "running" {
		action := "workload-exit"
		if st.State == "oom-killed" {
			action = "workload-oom"
		}
		a.record(action, "app/"+ad.ID, ad.Generation, st.Detail)
		a.pauseForward(ad.ID)
	}
	if ad.Restarts >= maxRestarts || a.now() < ad.Backoff {
		return
	}
	var as api.Assignment
	if json.Unmarshal(ad.Assignment, &as) != nil {
		return
	}
	a.mu.RLock()
	b := a.bundle
	a.mu.RUnlock()
	if b == nil {
		return
	}
	a.mu.Lock()
	ad.Restarts++
	ad.Backoff = a.now() + backoffMs(ad.Restarts)
	restarts := ad.Restarts
	a.mu.Unlock()
	a.record("workload-restart", "app/"+ad.ID, ad.Generation, fmt.Sprintf("restart %d/%d after %s", restarts, maxRestarts, st.Detail))
	if ad.Runtime == "docker" && ad.Inst.ContainerID != "" {
		_ = a.docker.Stop(ad.Inst, time.Second)
	}
	a.startWorkload(b, as, ad.Assignment, nil)
	a.mu.Lock()
	if cur := a.st.Admitted[ad.ID]; cur != nil {
		cur.Restarts = restarts
	}
	a.mu.Unlock()
}

func (a *Agent) stopWorkload(ad *Admitted, why string) {
	rt := a.runtimeFor(ad.Runtime)
	a.pauseForward(ad.ID)
	if ad.Inst.PID != 0 || ad.Inst.ContainerID != "" {
		_ = rt.Stop(ad.Inst, 5*time.Second)
	}
	// Take a final snapshot of attached volumes so a later placement can restore it.
	a.mu.Lock()
	ad.Stopped, ad.LastState = true, "stopped"
	a.mu.Unlock()
	a.record("workload-stop", "app/"+ad.ID, ad.Generation, why)
}

// checkHealth probes running workloads on their configured interval.
func (a *Agent) checkHealth() {
	a.mu.RLock()
	var list []*Admitted
	for _, ad := range a.st.Admitted {
		if !ad.Stopped && ad.LastState == "running" && ad.Health.HTTP != "" && ad.Inst.Port > 0 {
			list = append(list, ad)
		}
	}
	a.mu.RUnlock()
	for _, ad := range list {
		a.mu.RLock()
		last := a.healthAt[ad.ID]
		a.mu.RUnlock()
		if time.Since(last) < time.Duration(ad.Health.IntervalMs)*time.Millisecond {
			continue
		}
		obs := probeHTTP(fmt.Sprintf("http://127.0.0.1:%d%s", ad.Inst.Port, ad.Health.HTTP), time.Duration(ad.Health.TimeoutMs)*time.Millisecond)
		if v, ok := a.chaos.failHealth.Load(ad.ID); ok && v.(bool) {
			obs.OK, obs.Detail = false, "chaos: health failure injected"
		}
		a.mu.Lock()
		prev := a.health[ad.ID]
		if !obs.OK {
			obs.Consecutive = 1
			if prev != nil && !prev.OK {
				obs.Consecutive = prev.Consecutive + 1
			}
		}
		// Report unhealthy only after the failure threshold, healthy immediately.
		if !obs.OK && obs.Consecutive < ad.Health.FailureThreshold && prev != nil && prev.OK {
			held := *prev
			held.Consecutive = obs.Consecutive
			held.Detail = "probe failed (" + obs.Detail + "); below failure threshold"
			obs = held
		}
		a.health[ad.ID] = &obs
		a.healthAt[ad.ID] = time.Now()
		a.mu.Unlock()
	}
}

func safeName(s string) string {
	return strings.NewReplacer("/", "_", "@", "_", ":", "_").Replace(s)
}
