package evidence

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"sort"
	"time"
)

// Pack writes a deterministic .tar.gz of exactly the files SourceDigest
// covers (sorted paths, zero timestamps, modes reduced to 0755/0644), so a
// validation machine can be given precisely the tree the digest names.
func Pack(root string, w io.Writer) (digest string, files int, err error) {
	entries, err := SourceFiles(root)
	if err != nil {
		return "", 0, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	zw, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
	zw.ModTime = time.Unix(0, 0)
	tw := tar.NewWriter(zw)
	epoch := time.Unix(0, 0)
	for _, e := range entries {
		h := &tar.Header{Name: e.Path, ModTime: epoch, Format: tar.FormatPAX}
		if e.Link != "" {
			h.Typeflag, h.Linkname, h.Mode = tar.TypeSymlink, e.Link, 0o777
			if err := tw.WriteHeader(h); err != nil {
				return "", 0, err
			}
			continue
		}
		b, err := os.ReadFile(e.OSPath)
		if err != nil {
			return "", 0, err
		}
		h.Typeflag, h.Size, h.Mode = tar.TypeReg, int64(len(b)), 0o644
		if e.Exec {
			h.Mode = 0o755
		}
		if err := tw.WriteHeader(h); err != nil {
			return "", 0, err
		}
		if _, err := tw.Write(b); err != nil {
			return "", 0, err
		}
	}
	if err := tw.Close(); err != nil {
		return "", 0, err
	}
	if err := zw.Close(); err != nil {
		return "", 0, err
	}
	digest, files, err = SourceDigest(root)
	return digest, files, err
}
