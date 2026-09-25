package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/storage"
)

func (a *Agent) volumeDir(vid string) string {
	return filepath.Join(a.cfg.DataDir, "volumes", safeName(vid), "data")
}

func (a *Agent) volState(vid string) *VolState {
	a.mu.Lock()
	defer a.mu.Unlock()
	vs := a.st.Volumes[vid]
	if vs == nil {
		vs = &VolState{Evidence: map[string]int64{}}
		a.st.Volumes[vid] = vs
	}
	if vs.Evidence == nil {
		vs.Evidence = map[string]int64{}
	}
	return vs
}

// prepareVolume returns the host directory for a volume, restoring the
// committed snapshot first when the replica moved here.
func (a *Agent) prepareVolume(v api.AssignedVolume) (string, error) {
	dir := a.volumeDir(v.VolumeID)
	vs := a.volState(v.VolumeID)
	if v.Restore != "" && vs.Restored != v.Restore && vs.LastSnapshot != v.Restore {
		duty := a.duty(v.VolumeID)
		var members []string
		if duty != nil {
			members = duty.Members
		}
		if err := a.ensureSnapshot(v.VolumeID, v.Restore, members, "restore"); err != nil {
			return "", fmt.Errorf("restore %s: %w", short(v.Restore), err)
		}
		if err := storage.Restore(a.cas, v.Restore, dir); err != nil {
			return "", err
		}
		key := ""
		if m, err := storage.LoadManifest(a.cas, v.Restore); err == nil {
			key = contentKey(m)
		}
		a.mu.Lock()
		vs.Restored, vs.LastSnapshot, vs.LastRoot = v.Restore, v.Restore, key
		a.mu.Unlock()
		a.record("volume-restore", "volume/"+v.VolumeID, 0, fmt.Sprintf("restored committed snapshot %s from verified chunks before start", short(v.Restore)))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *Agent) duty(vid string) *api.VolumeDuty {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.bundle == nil {
		return nil
	}
	for i := range a.bundle.Volumes {
		if a.bundle.Volumes[i].VolumeID == vid {
			d := a.bundle.Volumes[i]
			return &d
		}
	}
	return nil
}

// snapshotPresence lists which of a snapshot's objects this host holds.
func (a *Agent) snapshotPresence(snap string) ([]string, []string) {
	if snap == "" {
		return nil, nil
	}
	m, err := storage.LoadManifest(a.cas, snap)
	if err != nil {
		return nil, nil
	}
	need := storage.ChunkSet(snap, m)
	var present []string
	for _, id := range need {
		if a.cas.Has(id) {
			present = append(present, id)
		}
	}
	return need, present
}

func (a *Agent) storageLoop(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	lastGC := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.storeMu.Lock()
		a.storageTick()
		if time.Since(lastGC) > time.Minute {
			lastGC = time.Now()
			a.gc()
		}
		a.storeMu.Unlock()
	}
}

func (a *Agent) storageTick() {
	a.mu.RLock()
	b := a.bundle
	a.mu.RUnlock()
	if b == nil || a.dev == nil {
		return
	}
	for _, duty := range b.Volumes {
		if duty.Primary == a.st.NodeID {
			a.maybeSnapshot(duty)
		}
		a.antiEntropy(duty)
	}
}

// contentKey identifies volume content independent of the snapshot's
// timestamp and parent.
func contentKey(m *api.SnapshotManifest) string {
	b, _ := canon.Marshal(m.Files)
	return envelope.HashBytes(b)
}

func (a *Agent) maybeSnapshot(duty api.VolumeDuty) {
	a.mu.RLock()
	var av *api.AssignedVolume
	running := false
	for _, ad := range a.st.Admitted {
		if ad.Stopped {
			continue
		}
		for i := range ad.Volumes {
			if ad.Volumes[i].VolumeID == duty.VolumeID {
				v := ad.Volumes[i]
				av = &v
				running = ad.LastState == "running"
			}
		}
	}
	a.mu.RUnlock()
	if av == nil || !running {
		return
	}
	vs := a.volState(duty.VolumeID)
	every := av.SnapshotEvery
	if every < 1000 {
		every = 30_000
	}
	if a.now()-vs.LastSnapshotAt < every {
		return
	}
	dir := a.volumeDir(duty.VolumeID)
	if _, err := os.Stat(dir); err != nil {
		return
	}
	res, err := storage.Snapshot(a.cas, dir, duty.VolumeID, vs.LastSnapshot, a.now())
	a.mu.Lock()
	vs.LastSnapshotAt = a.now()
	a.mu.Unlock()
	if err != nil {
		a.record("snapshot-fail", "volume/"+duty.VolumeID, 0, err.Error())
		return
	}
	key := contentKey(res.Manifest)
	if key == vs.LastRoot && vs.LastSnapshot != "" {
		// Unchanged since the last snapshot; the manifest object we just
		// wrote is unreferenced and will be collected.
		return
	}
	a.mu.Lock()
	vs.LastSnapshot, vs.LastRoot = res.ID, key
	a.mu.Unlock()
	a.record("snapshot-create", "volume/"+duty.VolumeID, 0, fmt.Sprintf("%s: %d file(s), %d object(s), %d bytes, merkle %s", short(res.ID), len(res.Manifest.Files), res.Chunks, res.Bytes, short(res.Root)))
	a.sendReplicaEvidence(duty.VolumeID, res.ID, res.Root, res.Chunks, res.Bytes, 0, 0, true, res.Manifest)
	// Push to the other members now (write path); anti-entropy also converges.
	for _, m := range duty.Members {
		if m == a.st.NodeID {
			continue
		}
		a.pushSnapshot(m, duty.VolumeID, res.ID, res.ChunkIDs)
	}
}

func (a *Agent) pushSnapshot(member, vid, snap string, ids []string) {
	ip := a.meshIPOf(member)
	pc := a.peerClient(30 * time.Second)
	if ip == "" || pc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// Manifest first, so the member can compute what it lacks.
	if data, err := a.cas.Get(snap); err == nil {
		_ = pc.PutChunk(ctx, ip, snap, data)
	}
	rep, err := pc.Buckets(ctx, ip, vid, snap)
	theirs := map[string]bool{}
	if err == nil {
		mine := storage.Buckets(ids)
		for i := range mine {
			if mine[i] == rep.Buckets[i] {
				for _, id := range ids {
					if storage.BucketOf(id) == i {
						theirs[id] = true
					}
				}
				continue
			}
			list, _ := pc.Bucket(ctx, ip, vid, snap, i)
			for _, id := range list {
				theirs[id] = true
			}
		}
	}
	for _, id := range ids {
		if theirs[id] {
			continue
		}
		data, err := a.cas.Get(id)
		if err != nil {
			continue
		}
		if err := pc.PutChunk(ctx, ip, id, data); err != nil {
			return
		}
	}
}

// ensureSnapshot makes every object of snap present locally, repairing from
// peers with Merkle bucket comparison. It records repair evidence.
func (a *Agent) ensureSnapshot(vid, snap string, members []string, trigger string) error {
	if _, err := storage.LoadManifest(a.cas, snap); err != nil {
		if _, err := a.fetchObject(snap, members); err != nil {
			return fmt.Errorf("manifest unavailable: %w", err)
		}
	}
	m, err := storage.LoadManifest(a.cas, snap)
	if err != nil {
		return err
	}
	need := storage.ChunkSet(snap, m)
	_, missing, corrupt := storage.Check(a.cas, need)
	if len(missing)+len(corrupt) == 0 {
		return nil
	}
	want := map[string]string{}
	for _, id := range missing {
		want[id] = "missing"
	}
	for _, id := range corrupt {
		want[id] = "corrupt"
	}
	localRoot := storage.MerkleRoot(presentOf(need, want))
	pc := a.peerClient(30 * time.Second)
	if pc == nil {
		return fmt.Errorf("%d object(s) missing and no mesh device", len(want))
	}
	opID := randID()
	var items []api.RepairItem
	peerRoot := ""
	for _, member := range members {
		if len(want) == 0 {
			break
		}
		if member == a.st.NodeID {
			continue
		}
		ip := a.meshIPOf(member)
		if ip == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		rep, err := pc.Buckets(ctx, ip, vid, snap)
		if err != nil {
			cancel()
			continue
		}
		if peerRoot == "" {
			peerRoot = rep.Root
		}
		mine := storage.Buckets(presentOf(need, want))
		for i := range mine {
			if mine[i] == rep.Buckets[i] {
				continue
			}
			list, err := pc.Bucket(ctx, ip, vid, snap, i)
			if err != nil {
				continue
			}
			for _, id := range list {
				prev, needed := want[id]
				if !needed {
					continue
				}
				data, err := pc.GetChunk(ctx, ip, id)
				item := api.RepairItem{Object: id, Source: member, Previous: prev}
				switch {
				case err != nil:
					item.Result, item.Detail = "failed", err.Error()
				case storage.ChunkID(data) != id:
					item.Result, item.Detail = "failed", "peer bytes failed BLAKE3 verification"
				default:
					if err := a.cas.PutWithID(id, data); err != nil {
						item.Result, item.Detail = "failed", err.Error()
					} else {
						item.Result, item.Verified = "present", true
						delete(want, id)
					}
				}
				items = append(items, item)
			}
		}
		cancel()
	}
	if len(items) > 0 {
		sort.Slice(items, func(i, j int) bool { return items[i].Object < items[j].Object })
		ev := api.RepairEvidence{OpID: opID, Node: a.st.NodeID, VolumeID: vid, Snapshot: snap, Trigger: trigger, LocalRoot: localRoot, PeerRoot: peerRoot, Items: items, TS: a.now()}
		if env, err := a.sign(envelope.KindRepair, ev); err == nil {
			_ = a.cp.post("/v1/evidence", env, nil)
		}
		okN := 0
		for _, it := range items {
			if it.Verified {
				okN++
			}
		}
		a.record("storage-repair", "volume/"+vid, 0, fmt.Sprintf("%s op %s: %d/%d object(s) repaired and verified for %s", trigger, opID[:8], okN, len(items), short(snap)))
	}
	if len(want) > 0 {
		return fmt.Errorf("%d object(s) still missing after anti-entropy", len(want))
	}
	return nil
}

func presentOf(need []string, want map[string]string) []string {
	var out []string
	for _, id := range need {
		if _, missing := want[id]; !missing {
			out = append(out, id)
		}
	}
	return out
}

func randID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// antiEntropy converges this host's copy of a volume's retained snapshots
// and reports signed replica evidence.
func (a *Agent) antiEntropy(duty api.VolumeDuty) {
	vs := a.volState(duty.VolumeID)
	targets := append([]string{}, duty.Keep...)
	if duty.Committed != nil && !contains(targets, duty.Committed.ID) {
		targets = append(targets, duty.Committed.ID)
	}
	// Newest first: the committed snapshot and anything pending after it.
	for i := len(targets) - 1; i >= 0; i-- {
		snap := targets[i]
		a.mu.RLock()
		last := vs.Evidence[snap]
		a.mu.RUnlock()
		if a.now()-last < a.cfg.AntiEntropyEvery.Milliseconds() {
			continue
		}
		members := append([]string{duty.Primary}, duty.Members...)
		_ = a.ensureSnapshot(duty.VolumeID, snap, members, "anti-entropy") // failures show up as Missing below
		m, merr := storage.LoadManifest(a.cas, snap)
		if merr != nil {
			a.mu.Lock()
			vs.Evidence[snap] = a.now() - a.cfg.AntiEntropyEvery.Milliseconds()*3/4 // retry soon, not every tick
			a.mu.Unlock()
			continue
		}
		need := storage.ChunkSet(snap, m)
		_, missing, corrupt := storage.Check(a.cas, need)
		var bytes int64
		for _, f := range m.Files {
			bytes += f.Size
		}
		// The evidence always names the snapshot root; Missing and Corrupt
		// say whether this copy is complete.
		a.sendReplicaEvidence(duty.VolumeID, snap, storage.MerkleRoot(need), int64(len(need)), bytes, int64(len(missing)), int64(len(corrupt)), false, nil)
		a.mu.Lock()
		vs.Evidence[snap] = a.now()
		a.mu.Unlock()
	}
}

func (a *Agent) sendReplicaEvidence(vid, snap, root string, chunks, bytes, missing, corrupt int64, primary bool, m *api.SnapshotManifest) {
	// Evidence claims a durable copy: flush the device cache first.
	if err := a.cas.Barrier(); err != nil {
		return
	}
	ev := api.ReplicaEvidence{Node: a.st.NodeID, VolumeID: vid, Snapshot: snap, Root: root, Chunks: chunks, Bytes: bytes, VerifiedAt: a.now(), Missing: missing, Corrupt: corrupt, Primary: primary, Manifest: m}
	env, err := a.sign(envelope.KindReplica, ev)
	if err != nil {
		return
	}
	_ = a.cp.post("/v1/evidence", env, nil)
}

// gc deletes objects no retained snapshot or artifact references. Only
// objects older than ten minutes are collected, so in-flight writes are safe.
func (a *Agent) gc() {
	a.mu.RLock()
	b := a.bundle
	a.mu.RUnlock()
	if b == nil {
		return
	}
	keep := map[string]bool{}
	for _, d := range b.Volumes {
		for _, snap := range append(append([]string{}, d.Keep...), a.volState(d.VolumeID).LastSnapshot) {
			if snap == "" {
				continue
			}
			keep[snap] = true
			if m, err := storage.LoadManifest(a.cas, snap); err == nil {
				for _, id := range storage.ChunkSet(snap, m) {
					keep[id] = true
				}
			} else {
				return // an unreadable retained manifest: do not guess, collect nothing
			}
		}
	}
	for _, art := range b.Artifacts {
		keep[art.Manifest] = true
		if data, err := a.cas.Get(art.Manifest); err == nil {
			var m api.ArtifactManifest
			if jsonUnmarshal(data, &m) == nil {
				for _, c := range m.Chunks {
					keep[c] = true
				}
			}
		}
	}
	removed := 0
	for _, id := range a.cas.List() {
		if keep[id] {
			continue
		}
		if age, ok := a.cas.Age(id); !ok || age < 10*time.Minute {
			continue
		}
		if a.cas.Delete(id) == nil {
			removed++
		}
	}
	if removed > 0 {
		a.record("storage-gc", "node/"+a.st.NodeID, 0, fmt.Sprintf("collected %d unreferenced object(s)", removed))
	}
}

func readTail(path string, n int) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n"), nil
}
