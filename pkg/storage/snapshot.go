package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/canon"
)

// SnapshotResult describes a snapshot written to the CAS.
type SnapshotResult struct {
	ID       string
	Manifest *api.SnapshotManifest
	Root     string
	Bytes    int64
	Chunks   int64
	ChunkIDs []string
}

// Snapshot chunks every regular file under dir into the CAS and stores the
// canonical manifest itself as a CAS object. Its id is the snapshot id.
func Snapshot(c *CAS, dir, volumeID, parent string, ts int64) (*SnapshotResult, error) {
	m := &api.SnapshotManifest{VolumeID: volumeID, Parent: parent, TS: ts, Files: []api.SnapshotFile{}}
	var all []string
	var total int64
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil // directories are implied; symlinks and devices are not captured
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		ch := NewChunker(f)
		file := api.SnapshotFile{Path: filepath.ToSlash(rel), Mode: int64(info.Mode().Perm()), Chunks: []string{}}
		for {
			chunk, err := ch.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if err := c.PutWithID(chunk.ID, chunk.Data); err != nil {
				return err
			}
			file.Chunks = append(file.Chunks, chunk.ID)
			file.Size += int64(len(chunk.Data))
			all = append(all, chunk.ID)
		}
		total += file.Size
		m.Files = append(m.Files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	body, err := canon.Marshal(m)
	if err != nil {
		return nil, err
	}
	id, err := c.Put(body)
	if err != nil {
		return nil, err
	}
	if err := c.Barrier(); err != nil {
		return nil, fmt.Errorf("snapshot durability barrier: %w", err)
	}
	ids := append(uniq(all), id)
	return &SnapshotResult{ID: id, Manifest: m, Root: MerkleRoot(ids), Bytes: total, Chunks: int64(len(uniq(ids))), ChunkIDs: uniq(ids)}, nil
}

// LoadManifest reads and verifies a snapshot manifest from the CAS.
func LoadManifest(c *CAS, id string) (*api.SnapshotManifest, error) {
	body, err := c.Get(id)
	if err != nil {
		return nil, err
	}
	if !canon.IsCanonical(body) {
		return nil, errors.New("snapshot manifest is not canonical")
	}
	var m api.SnapshotManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ChunkSet returns every object a snapshot needs, including the manifest.
func ChunkSet(id string, m *api.SnapshotManifest) []string {
	ids := []string{id}
	for _, f := range m.Files {
		ids = append(ids, f.Chunks...)
	}
	return uniq(ids)
}

// Restore materializes a snapshot into dir atomically: files are written to
// a sibling staging directory which then replaces dir.
func Restore(c *CAS, id, dir string) error {
	m, err := LoadManifest(c, id)
	if err != nil {
		return fmt.Errorf("restore %s: manifest: %w", id, err)
	}
	stage := dir + ".restore-tmp"
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o700); err != nil {
		return err
	}
	for _, f := range m.Files {
		clean := filepath.Clean(filepath.FromSlash(f.Path))
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("restore: unsafe path %q", f.Path)
		}
		dst := filepath.Join(stage, clean)
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(f.Mode&0o777)|0o600)
		if err != nil {
			return err
		}
		for _, cid := range f.Chunks {
			data, err := c.Get(cid)
			if err != nil {
				out.Close()
				return fmt.Errorf("restore %s: chunk %s: %w", f.Path, cid, err)
			}
			if _, err := out.Write(data); err != nil {
				out.Close()
				return err
			}
		}
		if err := out.Sync(); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
	old := dir + ".restore-old"
	_ = os.RemoveAll(old)
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, old); err != nil {
			return err
		}
	}
	if err := os.Rename(stage, dir); err != nil {
		_ = os.Rename(old, dir)
		return err
	}
	return os.RemoveAll(old)
}

// Check verifies which of a snapshot's objects are present and intact.
func Check(c *CAS, ids []string) (present []string, missing []string, corrupt []string) {
	for _, id := range ids {
		ok, bad := c.Verify(id)
		switch {
		case ok:
			present = append(present, id)
		case bad:
			corrupt = append(corrupt, id)
		default:
			missing = append(missing, id)
		}
	}
	return
}

// PutStream chunks a stream into the CAS and returns the ordered chunk ids
// and total size (used for artifacts).
func PutStream(c *CAS, r io.Reader) ([]string, int64, error) {
	ch := NewChunker(r)
	var ids []string
	var n int64
	for {
		chunk, err := ch.Next()
		if err == io.EOF {
			return ids, n, nil
		}
		if err != nil {
			return nil, 0, err
		}
		if err := c.PutWithID(chunk.ID, chunk.Data); err != nil {
			return nil, 0, err
		}
		ids = append(ids, chunk.ID)
		n += int64(len(chunk.Data))
	}
}
