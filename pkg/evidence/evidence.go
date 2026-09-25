// Package evidence records production-validation runs so that a promoted
// claim can be reproduced by someone else (docs/BLUEPRINT.md §12).
//
// A validation record captures:
//   - the environment: OS, kernel, architecture, CPU, memory, filesystem,
//     network interfaces, container or virtualization, and toolchain;
//   - the exact source, as a digest of the tree;
//   - binary digests;
//   - every command run, with start and end times, exit codes and a BLAKE3
//     hash of its log;
//   - the claim the run is meant to support.
//
// The record is signed (envelope kind "validation-record"), and `dh evidence
// verify` re-hashes every file.
//
// The environment is reported as measured. A container is reported as a
// container, never as a machine.
package evidence

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"decentralized.host/pkg/envelope"
)

// KindRecord is the signing domain of validation records.
const KindRecord = "validation-record"

// Env is the measured environment of one machine.
type Env struct {
	Hostname       string   `json:"hostname"`
	OS             string   `json:"os"`
	Arch           string   `json:"arch"`
	Kernel         string   `json:"kernel"`
	Distro         string   `json:"distro"`
	CPUModel       string   `json:"cpuModel"`
	CPUs           int64    `json:"cpus"`
	MemBytes       int64    `json:"memBytes"`
	DataDir        string   `json:"dataDir"`
	FSType         string   `json:"fsType"`
	DiskBytes      int64    `json:"diskBytes"`
	DiskFreeBytes  int64    `json:"diskFreeBytes"`
	Container      string   `json:"container"`      // docker | podman | "" (none detected)
	Virtualization string   `json:"virtualization"` // e.g. kvm, "" if not detected
	Addresses      []string `json:"addresses"`
	GoVersion      string   `json:"goVersion"`
	Python         string   `json:"python"`
	Docker         string   `json:"docker"`
	Notes          []string `json:"notes"` // anything the collector could not determine
}

// CollectEnv measures the current machine. dataDir selects the filesystem
// to report.
func CollectEnv(dataDir string) Env {
	e := Env{OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: int64(runtime.NumCPU()), GoVersion: goVersion(), DataDir: dataDir}
	e.Hostname, _ = os.Hostname()
	e.Kernel = run("uname", "-srvm")
	e.Python = run("python3", "--version")
	e.Docker = run("docker", "version", "--format", "{{.Server.Version}}")
	platformEnv(&e)
	if ifs, err := net.Interfaces(); err == nil {
		for _, i := range ifs {
			addrs, _ := i.Addrs()
			for _, a := range addrs {
				e.Addresses = append(e.Addresses, i.Name+" "+a.String())
			}
		}
	}
	if e.CPUModel == "" {
		e.Notes = append(e.Notes, "CPU model not determined")
	}
	if e.MemBytes == 0 {
		e.Notes = append(e.Notes, "memory size not determined")
	}
	if e.FSType == "" {
		e.Notes = append(e.Notes, "filesystem type not determined")
	}
	return e
}

func goVersion() string {
	if v := run("go", "version"); v != "" {
		return v
	}
	return runtime.Version() + " (binary)"
}

func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}

func readFirst(path, prefix string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), prefix) {
			return strings.TrimSpace(strings.TrimPrefix(sc.Text(), prefix))
		}
	}
	return ""
}

// ------------------------------------------------------------ source

// DigestVersion names the source-digest algorithm. Records store it next to
// the digest so a verifier recomputes with the same rules.
//
// dh-src-digest/1 excluded directories named bin, evidence and so on at any
// depth, which silently dropped the source package pkg/evidence (found when
// a bundle of REF-MAC-A01's tree failed to build). Version 2 excludes
// build and evidence output only at the root.
const DigestVersion = "dh-src-digest/2"

var sourceExt = map[string]bool{".go": true, ".py": true, ".js": true, ".css": true, ".html": true, ".md": true, ".json": true, ".mod": true, ".sum": true, ".sh": true, ".yaml": true, ".yml": true, ".txt": true}

// rootOutputs are build output, evidence and local cluster state, excluded
// only at the root of the tree.
var rootOutputs = map[string]bool{"bin": true, "evidence": true, "chaos-reports": true, "devcluster": true, "tools/bin": true}

// caches are excluded at any depth.
var caches = map[string]bool{".git": true, "__pycache__": true, "node_modules": true}

// SourceEntry is one path covered by the source digest.
type SourceEntry struct {
	Path   string // relative, "/" separators
	Link   string // symlink target, or ""
	Exec   bool
	Hash   string // b3 hex of content (regular files)
	OSPath string
}

// SourceFiles lists exactly the paths the digest covers, in digest order.
func SourceFiles(root string) ([]SourceEntry, error) {
	var out []SourceEntry
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && (caches[d.Name()] || rootOutputs[rel]) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			out = append(out, SourceEntry{Path: rel, Link: filepath.ToSlash(target), OSPath: p})
			return nil
		}
		if !d.Type().IsRegular() || (!sourceExt[filepath.Ext(p)] && d.Name() != "Makefile") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, SourceEntry{Path: rel, Exec: info.Mode().Perm()&0o111 != 0, Hash: envelope.HashBytes(b), OSPath: p})
		return nil
	})
	return out, err
}

func (e SourceEntry) line() string {
	if e.Link != "" {
		return "l\x00" + e.Path + "\x00" + e.Link
	}
	mode := "-"
	if e.Exec {
		mode = "x"
	}
	return "f\x00" + e.Path + "\x00" + mode + "\x00" + e.Hash
}

// SourceDigest identifies a source tree (algorithm dh-src-digest/2):
//
//  1. Walk the tree. Skip .git, __pycache__ and node_modules at any depth,
//     and bin, evidence, chaos-reports, devcluster and tools/bin at the root
//     only.
//  2. Keep regular files whose extension is in sourceExt, plus files named
//     Makefile, and every symbolic link (links are never followed).
//  3. For each kept path, relative to the root with "/" separators, emit
//     one line:
//     "f" NUL path NUL mode NUL b3hex(content) for a regular file, where mode
//     is "x" if any execute bit is set and "-" otherwise, or
//     "l" NUL path NUL target for a symbolic link.
//  4. Sort the lines bytewise, join them with "\n", and return
//     "b3:" + hex(BLAKE3-256(joined)).
//
// Empty directories, timestamps, ownership and all other mode bits are
// ignored, so the same tree copied to another machine gives the same digest.
func SourceDigest(root string) (string, int, error) {
	entries, err := SourceFiles(root)
	if err != nil {
		return "", 0, err
	}
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = e.line()
	}
	sort.Strings(lines)
	return "b3:" + envelope.HashBytes([]byte(strings.Join(lines, "\n"))), len(lines), nil
}

// ------------------------------------------------------------ steps

// Step is one executed command.
type Step struct {
	Name     string   `json:"name"`
	Command  []string `json:"command"`
	Start    int64    `json:"start"`
	End      int64    `json:"end"`
	Duration int64    `json:"durationMs"`
	Exit     int64    `json:"exit"`
	Log      string   `json:"log"`     // path relative to the evidence dir
	LogHash  string   `json:"logHash"` // b3 of the log
	Attempts int64    `json:"attempts"`
}

// ReadSteps loads dir/steps.jsonl.
func ReadSteps(dir string) ([]Step, error) {
	f, err := os.Open(filepath.Join(dir, "steps.jsonl"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Step
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var s Step
		if err := json.Unmarshal(sc.Bytes(), &s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, sc.Err()
}

// AppendStep records a step.
func AppendStep(dir string, s Step) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "steps.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// ------------------------------------------------------------ record

// File is one artifact in the evidence directory.
type File struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Hash  string `json:"hash"`
}

// Outcomes of a validation run. Verification (signature and artifact
// hashes) is separate and computed by Verify, never stored.
const (
	OutcomePass         = "PASS"
	OutcomeFail         = "FAIL"
	OutcomeIncomplete   = "INCOMPLETE"
	OutcomeInfraFailure = "INFRA_FAILURE"
)

// SchemaVersion of validation records.
const SchemaVersion = "dh-validation/2"

// Record is the signed validation record.
type Record struct {
	SchemaVersion   string            `json:"schemaVersion"`
	EvidenceID      string            `json:"evidenceId"`       // e.g. PV1-S1-A03, REF-MAC-A01
	AttemptID       string            `json:"attemptId"`        // e.g. A03
	ParentEvidence  string            `json:"parentEvidenceId"` // the previous attempt of this stage, or ""
	Stage           string            `json:"stage"`
	Claim           string            `json:"claim"`
	Scope           string            `json:"scope"`
	Limitations     []string          `json:"limitations"`
	ScopeExclusions []string          `json:"scopeExclusions"`
	DigestVersion   string            `json:"digestVersion"`
	SourceDigest    string            `json:"sourceDigest"`
	SourceFiles     int64             `json:"sourceFiles"`
	Commit          string            `json:"commit"` // "" when not under version control
	Binaries        map[string]string `json:"binaries"`
	Machines        []Env             `json:"machines"`
	Required        []string          `json:"requiredSteps"`
	Steps           []Step            `json:"steps"`
	Files           []File            `json:"files"`
	Outcome         string            `json:"outcome"`
	OutcomeReason   string            `json:"outcomeReason"`
	Start           int64             `json:"start"`
	End             int64             `json:"end"`
}

// DecideOutcome derives the outcome from the recorded steps. infra, if set,
// records that the infrastructure prevented a meaningful run (the operator
// states why). It overrides only runs that did not otherwise complete.
func DecideOutcome(required []string, steps []Step, infra string) (string, string) {
	ran := map[string]Step{}
	for _, s := range steps {
		ran[s.Name] = s
	}
	var missing, failed []string
	for _, r := range required {
		if _, ok := ran[r]; !ok {
			missing = append(missing, r)
		}
	}
	for _, s := range steps {
		if s.Exit != 0 {
			failed = append(failed, s.Name)
		}
	}
	switch {
	case infra != "" && (len(missing) > 0 || len(steps) == 0):
		return OutcomeInfraFailure, infra
	case len(failed) > 0:
		return OutcomeFail, "failed step(s): " + strings.Join(failed, ", ")
	case len(missing) > 0:
		return OutcomeIncomplete, "required step(s) not run: " + strings.Join(missing, ", ")
	case len(steps) == 0:
		return OutcomeIncomplete, "no steps recorded"
	}
	return OutcomePass, "every required step ran and exited 0"
}

// HashFiles lists every file under dir except the record itself.
func HashFiles(dir string) ([]File, error) {
	var out []File
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "record.json" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, File{Path: filepath.ToSlash(rel), Bytes: int64(len(b)), Hash: "b3:" + envelope.HashBytes(b)})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

// Verify checks a record's files against dir and returns problems found.
func Verify(r Record, dir string) []string {
	var problems []string
	for _, f := range r.Files {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path)))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", f.Path, err))
			continue
		}
		if got := "b3:" + envelope.HashBytes(b); got != f.Hash {
			problems = append(problems, fmt.Sprintf("%s: hash %s, record says %s", f.Path, got, f.Hash))
		}
	}
	return problems
}
