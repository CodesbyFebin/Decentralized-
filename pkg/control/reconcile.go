package control

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/pki"
	"decentralized.host/pkg/scheduler"
)

// CapabilityTTL is the lifetime of a per-assignment capability. The
// reconciler re-signs assignments at half-life; a host that loses the
// control plane keeps holding admitted work past expiry (hold semantics).
const CapabilityTTL = 24 * time.Hour

func (s *Server) reconcileLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	var last time.Time
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
		case <-s.kick:
			// A committed change (a revocation, a new binding) reaches this
			// member's own mesh immediately rather than on the next 2s pass:
			// the member that revokes a host must not keep it as a peer.
			s.ensureMesh()
		}
		if time.Since(last) < 250*time.Millisecond {
			continue
		}
		last = time.Now()
		if !s.IsLeader() {
			continue
		}
		if err := s.reconcileOnce(); err != nil {
			s.log.Printf("reconcile: %v", err)
		}
	}
}

type reconcilePlan struct {
	upserts []*envelope.Envelope
	removes []string
	reasons []string
	volumes []*Volume
	purge   []string
}

// usable reports whether a host may receive new work.
func usable(n *Node) bool { return n != nil && n.Status == "ready" && n.Health != "lost" }

func (s *Server) schedulerNodes(st *State, excludeApp string) map[string]*scheduler.Node {
	out := map[string]*scheduler.Node{}
	for _, id := range st.sortedNodeIDs() {
		n := st.Nodes[id]
		status := n.Status
		if status == "ready" && n.Health == "lost" {
			status = "lost"
		}
		sn := &scheduler.Node{ID: n.ID, Name: n.Name, Status: status, Tiers: n.Enroll.Tiers, Region: n.Enroll.Region, Zone: n.Enroll.Zone, Host: n.Enroll.Host,
			Arch: n.Enroll.Arch, Features: n.Enroll.Features, CPUMilli: n.Enroll.CPUMilli, MemBytes: n.Enroll.MemBytes, DiskFree: -1,
			Policy: n.Enroll.Policy, Roles: n.Roles, UptimeMs: nowMs() - n.JoinedAt}
		if n.Obs != nil && n.Obs.Storage != nil {
			sn.DiskFree = n.Obs.Storage.FreeBytes
			if q := n.Obs.Storage.QuotaBytes; q > 0 && q-n.Obs.Storage.UsedBytes < sn.DiskFree {
				sn.DiskFree = q - n.Obs.Storage.UsedBytes
			}
		}
		out[id] = sn
	}
	for _, key := range st.sortedAssignmentKeys() {
		rec := st.Assignments[key]
		if rec.A.Desired != "running" || rec.A.App == excludeApp {
			continue
		}
		if sn := out[rec.A.Node]; sn != nil {
			sn.UsedCPU += rec.A.Resources.CPUMilli
			sn.UsedMem += rec.A.Resources.MemBytes
			sn.Workloads++
		}
	}
	return out
}

func (s *Server) reconcileOnce() error {
	var p reconcilePlan
	var err error
	s.fsm.Read(func(st *State) {
		if st.Frozen || st.Cluster == "" {
			return
		}
		m := st.member(s.id.ID)
		if m == nil {
			err = fmt.Errorf("member %s is not in the roster; refusing to sign", s.id.ID)
			return
		}
		for _, name := range st.sortedAppNames() {
			if e := s.reconcileApp(st, st.Apps[name], m.Delegation, &p); e != nil {
				err = e
				return
			}
		}
		// Assignments whose app no longer exists at all.
		for _, key := range st.sortedAssignmentKeys() {
			rec := st.Assignments[key]
			if st.Apps[rec.A.App] == nil {
				p.removes = append(p.removes, key)
			}
		}
	})
	if err != nil {
		return err
	}
	if len(p.volumes) > 0 {
		if _, err := s.propose("volume-plan", "reconciler", map[string]any{"volumes": p.volumes}); err != nil {
			return err
		}
	}
	if len(p.upserts) > 0 || len(p.removes) > 0 {
		reason := strings.Join(uniqueSorted(p.reasons), "; ")
		if len(reason) > 400 {
			reason = reason[:400] + "…"
		}
		res, err := s.propose("assign", "reconciler", map[string]any{"upsert": p.upserts, "remove": p.removes, "reason": reason})
		if err != nil {
			return err
		}
		if !res.OK {
			return fmt.Errorf("assign: %s", res.Message)
		}
	}
	for _, app := range p.purge {
		_, _ = s.propose("purge-app", "reconciler", map[string]any{"app": app})
	}
	return nil
}

func finished(st *State, rec *AssignmentRec) bool {
	n := st.Nodes[rec.A.Node]
	if n == nil {
		return true
	}
	if n.Health == "lost" && nowMs()-n.LastObsAt > int64(10*time.Minute/time.Millisecond) {
		return true
	}
	if n.Obs == nil {
		return n.Status == "pending"
	}
	w := st.workloadObs(rec.A.Node, rec.A.ID)
	if w == nil {
		return n.Obs.TS > rec.A.Issued || n.LastObsAt > rec.Created
	}
	if w.Admitted == "refused" {
		return true
	}
	switch w.Observed {
	case "stopped", "exited", "failed", "oom-killed":
		return true
	}
	return false
}

// replicaReady: running at the record's generation, and healthy if a health
// check is configured.
func replicaReady(st *State, cache *obsCache, rec *AssignmentRec) bool {
	w := latestWorkload(st, cache, rec.A.Node, rec.A.ID)
	if w == nil || w.Observed != "running" || w.Generation != rec.A.Generation || w.Admitted != "allowed" {
		return false
	}
	if rec.A.Health.HTTP != "" && (w.Health == nil || !w.Health.OK) {
		return false
	}
	return true
}

func (s *Server) reconcileApp(st *State, app *App, delegation string, p *reconcilePlan) error {
	var recs []*AssignmentRec
	for _, key := range st.sortedAssignmentKeys() {
		if rec := st.Assignments[key]; rec.A.App == app.Name {
			recs = append(recs, rec)
		}
	}
	stop := func(rec *AssignmentRec, why string) error {
		env, err := s.signAssignment(st, app, rec.A, "stopped", rec.A.Generation, rec.A.Volumes, delegation)
		if err != nil {
			return err
		}
		p.upserts = append(p.upserts, env)
		p.reasons = append(p.reasons, why)
		return nil
	}
	if app.Deleted {
		for _, rec := range recs {
			if rec.A.Desired == "running" {
				if err := stop(rec, app.Name+" deleted"); err != nil {
					return err
				}
			} else if finished(st, rec) {
				p.removes = append(p.removes, rec.Key)
			}
		}
		if len(recs) == 0 {
			p.purge = append(p.purge, app.Name)
		}
		return nil
	}
	spec := app.Manifest.Spec
	nodes := s.schedulerNodes(st, app.Name)
	// Hosts whose local policy refused this generation are excluded from new
	// placement: a host saying no is final for that generation.
	refused := map[string]string{}
	for _, rec := range recs {
		if w := latestWorkload(st, s.obs, rec.A.Node, rec.A.ID); w != nil && w.Admitted == "refused" && w.Generation == rec.A.Generation && rec.A.Generation == app.Generation && strings.HasPrefix(w.Code, "POLICY_") {
			refused[rec.A.Node] = w.Code
		}
	}
	current := map[int64]string{}
	byReplica := map[int64][]*AssignmentRec{}
	for _, rec := range recs {
		byReplica[rec.A.Replica] = append(byReplica[rec.A.Replica], rec)
		if rec.A.Desired != "running" || refused[rec.A.Node] != "" {
			continue
		}
		if cur, has := current[rec.A.Replica]; has {
			// Two running records (a move in progress): prefer the newer placement.
			if old := st.Assignments[assignmentKey(rec.A.ID, cur)]; old != nil && old.Created >= rec.Created {
				continue
			}
		}
		current[rec.A.Replica] = rec.A.Node
	}
	var list []scheduler.Node
	for _, id := range st.sortedNodeIDs() {
		sn := *nodes[id]
		if code := refused[id]; code != "" {
			sn.Status = "refused (" + code + ")"
		}
		list = append(list, sn)
	}
	// Prefer hosts that already hold a replica's volume data: a move then
	// restores from local, already-verified chunks.
	hints := map[int64][]string{}
	for _, v := range spec.Volumes {
		for r := int64(0); r < spec.Replicas; r++ {
			if vol := st.Volumes[fmt.Sprintf("%s/%s/r%d", app.Name, v.Name, r)]; vol != nil {
				hints[r] = append(hints[r], vol.Members...)
			}
		}
	}
	plan := scheduler.Schedule(scheduler.Request{App: app.Name, Spec: spec, Nodes: list, Current: current, VolumeHint: hints, Federated: app.Federation != nil})
	note := ""
	if u := plan.Unscheduled(); len(u) > 0 {
		note = fmt.Sprintf("%d replica(s) unschedulable", len(u))
	}
	s.plans.set(app.Name, plan, note)

	// Rolling-update budget.
	notReady := int64(0)
	for r := int64(0); r < spec.Replicas; r++ {
		ready := false
		for _, rec := range byReplica[r] {
			if rec.A.Desired == "running" && replicaReady(st, s.obs, rec) {
				ready = true
			}
		}
		if !ready {
			notReady++
		}
	}
	budget := spec.Update.MaxUnavailable - notReady
	now := nowMs()
	for _, rp := range plan.Replicas {
		if rp.Node == "" {
			continue
		}
		vols, err := s.planVolumes(st, app, rp.Replica, rp.Node, nodes, p)
		if err != nil {
			return err
		}
		id := fmt.Sprintf("%s/r%d", app.Name, rp.Replica)
		rec := st.Assignments[assignmentKey(id, rp.Node)]
		base := api.Assignment{ID: id, App: app.Name, Replica: rp.Replica, Node: rp.Node}
		switch {
		case rec == nil || rec.A.Desired != "running":
			env, err := s.signAssignment(st, app, base, "running", app.Generation, vols, delegation)
			if err != nil {
				return err
			}
			p.upserts = append(p.upserts, env)
			p.reasons = append(p.reasons, fmt.Sprintf("placed %s on %s", id, name(st, rp.Node)))
		case rec.A.Generation < app.Generation:
			if budget <= 0 {
				p.reasons = append(p.reasons, "rolling update waiting for maxUnavailable budget")
				continue
			}
			budget--
			env, err := s.signAssignment(st, app, base, "running", app.Generation, vols, delegation)
			if err != nil {
				return err
			}
			p.upserts = append(p.upserts, env)
			p.reasons = append(p.reasons, fmt.Sprintf("rolling %s to generation %d", id, app.Generation))
		case now-rec.A.Issued > int64(CapabilityTTL/time.Millisecond)/2 || volumesChanged(rec.A.Volumes, vols) || !anchoredIn(rec.A.Capability, st.Root) || rec.Env.Signer != s.id.ID && st.member(rec.Env.Signer) == nil:
			env, err := s.signAssignment(st, app, base, "running", rec.A.Generation, vols, delegation)
			if err != nil {
				return err
			}
			p.upserts = append(p.upserts, env)
		}
		// Retire other running placements of this replica (moves).
		newReady := rec != nil && rec.A.Desired == "running" && replicaReady(st, s.obs, rec)
		for _, other := range byReplica[rp.Replica] {
			if other.A.Node == rp.Node || other.A.Desired != "running" {
				continue
			}
			on := st.Nodes[other.A.Node]
			gone := on == nil || on.Health == "lost" || refused[other.A.Node] != ""
			if gone || newReady {
				why := fmt.Sprintf("moved %s from %s to %s", id, name(st, other.A.Node), name(st, rp.Node))
				if err := stop(other, why); err != nil {
					return err
				}
			}
		}
	}
	for _, rec := range recs {
		if rec.A.Desired == "running" && rec.A.Replica >= spec.Replicas {
			if err := stop(rec, fmt.Sprintf("scaled %s to %d", app.Name, spec.Replicas)); err != nil {
				return err
			}
		}
		if rec.A.Desired == "stopped" && finished(st, rec) {
			p.removes = append(p.removes, rec.Key)
		}
	}
	return nil
}

// anchoredIn reports whether a capability's first block is signed by root
// (false after a root rotation, so the assignment is re-signed).
func anchoredIn(tok, root string) bool {
	t, err := capability.Decode(tok)
	if err != nil || len(t.Blocks) == 0 {
		return false
	}
	return t.Blocks[0].Pub == root
}

func volumesChanged(a, b []api.AssignedVolume) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}

// planVolumes places storage replicas for one app replica and returns the
// volume attachments for its assignment.
func (s *Server) planVolumes(st *State, app *App, replica int64, node string, nodes map[string]*scheduler.Node, p *reconcilePlan) ([]api.AssignedVolume, error) {
	var out []api.AssignedVolume
	var list []scheduler.Node
	for _, id := range st.sortedNodeIDs() {
		list = append(list, *nodes[id])
	}
	for _, v := range app.Manifest.Spec.Volumes {
		vid := fmt.Sprintf("%s/%s/r%d", app.Name, v.Name, replica)
		cur := st.Volumes[vid]
		var curMembers []string
		if cur != nil {
			curMembers = cur.Members
		}
		vp := scheduler.PlaceVolume(scheduler.VolumeRequest{Replicas: v.Durability.Replicas, SizeBytes: v.SizeBytes, Primary: node, Current: curMembers, Tiers: app.Manifest.Spec.Placement.Tiers, Nodes: list})
		members := vp.Members
		if !contains(members, node) {
			members = append([]string{node}, members...)
		}
		if cur == nil || strings.Join(cur.Members, ",") != strings.Join(members, ",") || cur.Primary != node || cur.Short != vp.Short {
			p.volumes = append(p.volumes, &Volume{ID: vid, App: app.Name, Name: v.Name, Replica: replica, SizeBytes: v.SizeBytes, Durability: v.Durability,
				Retain: v.RetainSnapshots, Tiers: app.Manifest.Spec.Placement.Tiers, Primary: node, Members: members, Short: vp.Short})
		}
		restore := ""
		if cur != nil && cur.Committed != "" && cur.Primary != "" && cur.Primary != node {
			restore = cur.Committed
		}
		// Keep the restore point stable once signed, so the host restores once.
		if rec := st.Assignments[assignmentKey(fmt.Sprintf("%s/r%d", app.Name, replica), node)]; rec != nil {
			for _, av := range rec.A.Volumes {
				if av.VolumeID == vid && av.Restore != "" {
					restore = av.Restore
				}
			}
		}
		out = append(out, api.AssignedVolume{Name: v.Name, VolumeID: vid, Mount: v.Mount, SnapshotEvery: v.SnapshotEveryMs, Restore: restore})
	}
	return out, nil
}

// signAssignment builds, authorizes and signs one assignment.
func (s *Server) signAssignment(st *State, app *App, base api.Assignment, desired string, gen int64, vols []api.AssignedVolume, delegation string) (*envelope.Envelope, error) {
	spec := app.Manifest.Spec
	now := nowMs()
	a := base
	a.Generation, a.Desired = gen, desired
	a.Runtime, a.Image, a.Digest = spec.Runtime, spec.Image, manifest.Digest(spec.Image)
	a.Command, a.Env, a.Resources, a.Ports, a.Health = spec.Command, spec.Env, spec.Resources, spec.Ports, spec.Health
	a.Volumes, a.Tiers, a.ManifestHash, a.Issued = vols, spec.Placement.Tiers, app.Hash, now
	if a.Volumes == nil {
		a.Volumes = []api.AssignedVolume{}
	}
	if f := app.Federation; f != nil {
		a.Federation = &api.FederationRef{Peer: f.Peer, PeerRoot: f.PeerRoot, Agreement: f.Agreement, Placement: f.Request}
	}
	tok, err := capability.Decode(delegation)
	if err != nil {
		return nil, fmt.Errorf("member delegation: %w", err)
	}
	tok, err = tok.Attenuate(s.id, capability.Caveats{
		Actions: []string{"workload.admit"}, Resources: []string{"app/" + a.ID}, Audience: a.Node, Generation: gen, Digest: a.Digest,
		CPUMaxMilli: spec.Resources.CPUMilli, MemMaxBytes: spec.Resources.MemBytes, NotBefore: now - 60_000, Expires: now + int64(CapabilityTTL/time.Millisecond),
	}, "", "assignment "+a.ID)
	if err != nil {
		return nil, err
	}
	a.Capability = tok.Encode()
	return envelope.Sign(s.id, "", envelope.KindAssignment, a)
}

// leaderHousekeeping: host liveness, the local CA, and audit checkpoints.
func (s *Server) leaderHousekeeping() {
	var lost []string
	var needCA bool
	var cpDue bool
	var head int64
	var cluster string
	s.fsm.Read(func(st *State) {
		cluster = st.Cluster
		if cluster == "" {
			return
		}
		needCA = st.LocalCACert == ""
		for _, id := range st.sortedNodeIDs() {
			n := st.Nodes[id]
			if n.Health != "live" || (n.Status != "ready" && n.Status != "draining") {
				continue
			}
			last := n.LastObsAt
			if c := s.obs.get(id); c != nil && c.Received.UnixMilli() > last {
				last = c.Received.UnixMilli()
			}
			if nowMs()-last > s.cfg.LostAfter.Milliseconds() {
				lost = append(lost, id)
			}
		}
		head, _ = st.Audit.Head()
		var lastCP int64
		if n := len(st.Checkpoints); n > 0 {
			var p audit.CheckpointPayload
			_ = st.Checkpoints[n-1].Decode(&p)
			lastCP = p.Seq
		}
		cpDue = head-lastCP >= 25 || (head > lastCP && s.sinceCheckpoint() > time.Minute)
	})
	if cluster == "" {
		return
	}
	for _, id := range lost {
		_, _ = s.propose("node-health", "reconciler", map[string]any{"node": id, "health": "lost",
			"detail": fmt.Sprintf("no signed observation for %s; replicas will be rescheduled, admitted work on the host is not assumed dead", s.cfg.LostAfter)})
	}
	if needCA {
		cert, key, err := pki.NewLocalCA(cluster)
		if err == nil {
			_, _ = s.propose("local-ca", s.id.ID, map[string]string{"cert": string(cert), "key": string(key)})
		}
	}
	if cpDue {
		var env *envelope.Envelope
		s.fsm.Read(func(st *State) {
			env, _ = audit.SignCheckpoint(s.id, "", st.Cluster, &st.Audit, nowMs())
		})
		if env != nil {
			if res, err := s.propose("checkpoint", s.id.ID, env); err == nil && res.OK {
				s.markCheckpoint()
			}
		}
	}
	s.distributeArtifacts()
	s.federationTick()
}

func (s *Server) sinceCheckpoint() time.Duration {
	if s.lastCP.IsZero() {
		return time.Hour
	}
	return time.Since(s.lastCP)
}

func (s *Server) markCheckpoint() { s.lastCP = time.Now() }

// sortStrings is a tiny helper used by views.
func sortStrings(v []string) []string { sort.Strings(v); return v }
