package evidence

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, rel, body string, mode os.FileMode) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func digest(t *testing.T, root string) string {
	t.Helper()
	d, _, err := SourceDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func tree(t *testing.T) string {
	root := t.TempDir()
	write(t, root, "go.mod", "module x\n", 0o644)
	write(t, root, "pkg/a/a.go", "package a\n", 0o644)
	write(t, root, "pkg/devcluster/d.go", "package devcluster\n", 0o644)
	write(t, root, "validation/run.sh", "#!/bin/sh\n", 0o755)
	write(t, root, "Makefile", "all:\n", 0o644)
	return root
}

func TestSourceDigestIsStableAcrossCopies(t *testing.T) {
	a := tree(t)
	b := t.TempDir()
	// cp -p keeps modes; timestamps and ownership may differ and must not matter.
	if out, err := exec.Command("cp", "-Rp", a+"/.", b).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if digest(t, a) != digest(t, b) {
		t.Fatal("identical trees produced different digests")
	}
}

func TestSourceDigestExclusionsAndSensitivity(t *testing.T) {
	root := tree(t)
	base := digest(t, root)
	// Build output, evidence, the default dev-cluster directory and caches are not source.
	write(t, root, "bin/dh", "binary", 0o755)
	write(t, root, "evidence/x/record.json", "{}", 0o644)
	write(t, root, "devcluster/cp-1/state.json", "{}", 0o644)
	write(t, root, "tools/bin/pebble", "binary", 0o755)
	write(t, root, "pkg/a/__pycache__/x.pyc", "x", 0o644)
	write(t, root, "pkg/a/notes.bin", "not a source extension", 0o644)
	if digest(t, root) != base {
		t.Fatal("excluded paths changed the digest")
	}
	// A source package that happens to be named devcluster is source.
	write(t, root, "pkg/devcluster/d.go", "package devcluster // changed\n", 0o644)
	if digest(t, root) == base {
		t.Fatal("pkg/devcluster must be part of the source digest")
	}
	write(t, root, "pkg/devcluster/d.go", "package devcluster\n", 0o644)
	if digest(t, root) != base {
		t.Fatal("restoring content must restore the digest")
	}
	if err := os.Chmod(filepath.Join(root, "validation/run.sh"), 0o644); err != nil {
		t.Fatal(err)
	}
	if digest(t, root) == base {
		t.Fatal("the executable bit is part of the digest")
	}
}

func TestSourceDigestRecordsSymlinksWithoutFollowing(t *testing.T) {
	root := tree(t)
	outside := t.TempDir()
	write(t, outside, "secret.go", "package secret\n", 0o644)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	d1 := digest(t, root)
	write(t, outside, "secret.go", "package secret // changed outside the tree\n", 0o644)
	if digest(t, root) != d1 {
		t.Fatal("a symlink must not be followed")
	}
	os.Remove(filepath.Join(root, "link"))
	if err := os.Symlink("elsewhere", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if digest(t, root) == d1 {
		t.Fatal("a symlink's target is part of the digest")
	}
}

func TestDecideOutcome(t *testing.T) {
	ok := Step{Name: "build", Exit: 0}
	bad := Step{Name: "chaos", Exit: 1}
	cases := []struct {
		name     string
		required []string
		steps    []Step
		infra    string
		want     string
	}{
		{"all required ran", []string{"build"}, []Step{ok}, "", OutcomePass},
		{"a step failed", []string{"build", "chaos"}, []Step{ok, bad}, "", OutcomeFail},
		{"required step missing", []string{"build", "chaos"}, []Step{ok}, "", OutcomeIncomplete},
		{"nothing ran", nil, nil, "", OutcomeIncomplete},
		{"infra stopped the run", []string{"build", "chaos"}, []Step{ok}, "VM stopped", OutcomeInfraFailure},
		{"infra claim cannot mask a completed failure", []string{"build", "chaos"}, []Step{ok, bad}, "VM stopped", OutcomeFail},
	}
	for _, c := range cases {
		if got, why := DecideOutcome(c.required, c.steps, c.infra); got != c.want {
			t.Errorf("%s: got %s (%s), want %s", c.name, got, why, c.want)
		}
	}
}

func TestVerifyDetectsAlteredArtifacts(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "logs/01-build.log", "ok\n", 0o644)
	files, err := HashFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := Record{Files: files}
	if p := Verify(r, dir); len(p) != 0 {
		t.Fatalf("unaltered: %v", p)
	}
	write(t, dir, "logs/01-build.log", "FAILED\n", 0o644)
	if p := Verify(r, dir); len(p) != 1 {
		t.Fatalf("an altered log must be reported, got %v", p)
	}
}

// Regression: dh-src-digest/1 skipped any directory named "evidence" (or
// "bin"), which dropped the source package pkg/evidence from REF-MAC-A01's
// digest. Output directories are excluded only at the root.
func TestSourceDigestExcludesOutputsOnlyAtRoot(t *testing.T) {
	root := tree(t)
	base := digest(t, root)
	write(t, root, "pkg/evidence/e.go", "package evidence\n", 0o644)
	if digest(t, root) == base {
		t.Fatal("pkg/evidence is source and must change the digest")
	}
	withPkg := digest(t, root)
	write(t, root, "evidence/REF/record.json", "{}", 0o644)
	write(t, root, "bin/dh", "binary", 0o755)
	if digest(t, root) != withPkg {
		t.Fatal("root-level evidence and bin are output, not source")
	}
}

func TestPackRoundTripReproducesDigest(t *testing.T) {
	root := tree(t)
	write(t, root, "pkg/evidence/e.go", "package evidence\n", 0o644)
	write(t, root, "evidence/REF/record.json", "{}", 0o644)
	var buf bytes.Buffer
	d, n, err := Pack(root, &buf)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(out, filepath.FromSlash(h.Name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if h.Typeflag == tar.TypeSymlink {
			os.Symlink(h.Linkname, p)
			continue
		}
		b, _ := io.ReadAll(tr)
		os.WriteFile(p, b, os.FileMode(h.Mode))
	}
	got, gn, err := SourceDigest(out)
	if err != nil || got != d || gn != n {
		t.Fatalf("unpacked tree: %s (%d files), packed %s (%d files), %v", got, gn, d, n, err)
	}
	var again bytes.Buffer
	if _, _, err := Pack(root, &again); err != nil || !bytes.Equal(again.Bytes(), mustPack(t, root)) {
		t.Fatal("pack output must be byte-identical across runs")
	}
}

func mustPack(t *testing.T, root string) []byte {
	var b bytes.Buffer
	if _, _, err := Pack(root, &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
