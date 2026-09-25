package control

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/storage"
)

// handleVolumeExport streams a committed snapshot as a tar archive. The
// control plane holds no volume data: it fetches the manifest and chunks
// from replica hosts over the mesh and verifies every object's BLAKE3 hash
// before writing it out.
func (s *Server) handleVolumeExport(w http.ResponseWriter, r *http.Request, a *authz) {
	vid := r.URL.Query().Get("volume")
	snap := r.URL.Query().Get("snapshot")
	var holders []string
	s.fsm.Read(func(st *State) {
		v := st.Volumes[vid]
		if v == nil {
			return
		}
		if snap == "" {
			snap = v.Committed
		}
		for _, sn := range v.Snapshots {
			if sn.ID != snap {
				continue
			}
			for _, node := range sortedKeys(sn.Evidence) {
				if sn.Evidence[node].Complete(sn.Root) {
					if n := st.Nodes[node]; n != nil && n.Status != "revoked" {
						holders = append(holders, n.MeshIP)
					}
				}
			}
		}
	})
	if snap == "" || len(holders) == 0 {
		writeErr(w, 404, "no committed snapshot with verified replicas for volume %q", vid)
		return
	}
	pc, err := s.peerClient(30 * time.Second)
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}
	get := func(id string) ([]byte, error) {
		var lastErr error
		for _, ip := range holders {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			data, err := pc.GetChunk(ctx, ip, id)
			cancel()
			if err != nil {
				lastErr = err
				continue
			}
			if storage.ChunkID(data) != id {
				lastErr = fmt.Errorf("holder %s served corrupt bytes for %s", ip, id)
				continue
			}
			return data, nil
		}
		return nil, lastErr
	}
	mb, err := get(snap)
	if err != nil {
		writeErr(w, 502, "manifest: %v", err)
		return
	}
	if !canon.IsCanonical(mb) {
		writeErr(w, 502, "manifest is not canonical")
		return
	}
	var m api.SnapshotManifest
	if err := json.Unmarshal(mb, &m); err != nil {
		writeErr(w, 502, "%v", err)
		return
	}
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("X-DH-Snapshot", snap)
	tw := tar.NewWriter(w)
	defer tw.Close()
	_ = tw.WriteHeader(&tar.Header{Name: ".dh-snapshot.json", Mode: 0o644, Size: int64(len(mb)), ModTime: time.UnixMilli(m.TS)})
	_, _ = tw.Write(mb)
	for _, f := range m.Files {
		if err := tw.WriteHeader(&tar.Header{Name: f.Path, Mode: f.Mode, Size: f.Size, ModTime: time.UnixMilli(m.TS)}); err != nil {
			return
		}
		for _, cid := range f.Chunks {
			data, err := get(cid)
			if err != nil {
				return // truncated archive: the client sees a tar error, never silently partial data
			}
			if _, err := tw.Write(data); err != nil {
				return
			}
		}
	}
	_, _ = s.propose("operator-note", a.Actor, map[string]string{"action": "volume-export", "resource": "volume/" + vid, "detail": fmt.Sprintf("snapshot %s exported (%d files)", short(snap), len(m.Files))})
}
