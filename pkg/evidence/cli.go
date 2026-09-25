package evidence

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/cli"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
)

func init() {
	cli.Register("evidence digest", "print the source-tree digest a validation record pins: [--src DIR]", cmdDigest)
	cli.Register("evidence pack", "write exactly the digested source tree as a deterministic .tar.gz: --out FILE [--src DIR]", cmdPack)
	cli.Register("evidence env", "print this machine's measured environment: [--data DIR]", cmdEnv)
	cli.Register("evidence run", "run and record one validation step: --dir EVID --name STEP [--retries N] -- COMMAND...", cmdRun)
	cli.Register("evidence seal", "sign a validation record over an evidence dir: --dir EVID --id EVIDENCE-ID --stage ID --claim TEXT --scope TEXT --require step,... [--parent ID] [--limitation T ...] [--exclude T ...] [--infra-failure REASON] [--machine env.json ...]", cmdSeal)
	cli.Register("evidence verify", "verify a validation record and re-hash its files: --dir EVID", cmdVerify)
}

func cmdDigest(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("evidence digest", flag.ContinueOnError)
	src := fs.String("src", ".", "source tree")
	if err := fs.Parse(args); err != nil {
		return err
	}
	d, n, err := SourceDigest(*src)
	if err != nil {
		return err
	}
	fmt.Printf("%s (%d files)\n", d, n)
	return nil
}

func cmdPack(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("evidence pack", flag.ContinueOnError)
	src := fs.String("src", ".", "source tree")
	out := fs.String("out", "", "output .tar.gz")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("--out is required")
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	d, n, err := Pack(*src, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s: %d files, source %s (%s)\n", *out, n, d, DigestVersion)
	return nil
}

func cmdEnv(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("evidence env", flag.ContinueOnError)
	data := fs.String("data", ".", "directory whose filesystem is reported")
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(CollectEnv(*data), "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdRun(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("evidence run", flag.ContinueOnError)
	dir := fs.String("dir", "", "evidence directory")
	name := fs.String("name", "", "step name")
	retries := fs.Int("retries", 0, "retries on failure (every attempt is logged and counted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cmd := fs.Args()
	if *dir == "" || *name == "" || len(cmd) == 0 {
		return errors.New("usage: dh evidence run --dir EVID --name STEP -- COMMAND...")
	}
	if err := os.MkdirAll(filepath.Join(*dir, "logs"), 0o755); err != nil {
		return err
	}
	steps, _ := ReadSteps(*dir)
	logRel := filepath.Join("logs", fmt.Sprintf("%02d-%s.log", len(steps)+1, safe(*name)))
	logf, err := os.Create(filepath.Join(*dir, logRel))
	if err != nil {
		return err
	}
	st := Step{Name: *name, Command: cmd, Start: time.Now().UnixMilli(), Log: filepath.ToSlash(logRel)}
	for attempt := 0; attempt <= *retries; attempt++ {
		st.Attempts++
		fmt.Fprintf(logf, "=== attempt %d at %s: %s\n", attempt+1, time.Now().UTC().Format(time.RFC3339), strings.Join(cmd, " "))
		c := exec.Command(cmd[0], cmd[1:]...)
		w := io.MultiWriter(logf, os.Stdout)
		c.Stdout, c.Stderr = w, w
		err = c.Run()
		st.Exit = 0
		if err != nil {
			st.Exit = 1
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				st.Exit = int64(ee.ExitCode())
			}
			fmt.Fprintf(logf, "=== exit %d: %v\n", st.Exit, err)
		}
		if st.Exit == 0 {
			break
		}
	}
	st.End = time.Now().UnixMilli()
	st.Duration = st.End - st.Start
	logf.Close()
	b, _ := os.ReadFile(filepath.Join(*dir, logRel))
	st.LogHash = "b3:" + envelope.HashBytes(b)
	if err := AppendStep(*dir, st); err != nil {
		return err
	}
	fmt.Printf("evidence: step %q exit %d after %d attempt(s) in %s\n", st.Name, st.Exit, st.Attempts, time.Duration(st.End-st.Start)*time.Millisecond)
	if st.Exit != 0 {
		return fmt.Errorf("step %q failed", st.Name)
	}
	return nil
}

func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(s))
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

func cmdSeal(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("evidence seal", flag.ContinueOnError)
	dir := fs.String("dir", "", "evidence directory")
	id := fs.String("id", "", "evidence id, e.g. PV1-S1-A03 or REF-MAC-A01")
	parent := fs.String("parent", "", "evidence id of the previous attempt of this stage")
	stage := fs.String("stage", "", "validation stage, e.g. PV1-S1")
	claim := fs.String("claim", "", "the claim this evidence supports")
	scope := fs.String("scope", "", "the environment the evidence covers, stated plainly")
	require := fs.String("require", "", "comma-separated step names that must run for a PASS")
	infra := fs.String("infra-failure", "", "the infrastructure prevented a meaningful run: why")
	src := fs.String("src", ".", "source tree")
	bin := fs.String("bin", "", "directory with the binaries under test (default: next to dh)")
	key := fs.String("key", "", "validator key directory (default: <dir>/../validator)")
	var machines, limitations, exclusions multi
	fs.Var(&machines, "machine", "env.json of another machine in the run (repeatable)")
	fs.Var(&limitations, "limitation", "a known limitation of this evidence (repeatable)")
	fs.Var(&exclusions, "exclude", "something deliberately out of scope (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *id == "" || *stage == "" || *claim == "" || *scope == "" {
		return errors.New("--dir, --id, --stage, --claim and --scope are required")
	}
	if _, err := os.Stat(filepath.Join(*dir, "record.json")); err == nil {
		return errors.New("this evidence directory is already sealed; a new attempt needs a new directory and id")
	}
	r := Record{SchemaVersion: SchemaVersion, EvidenceID: *id, ParentEvidence: *parent, Stage: *stage, Claim: *claim, Scope: *scope,
		Limitations: limitations, ScopeExclusions: exclusions, DigestVersion: DigestVersion, Binaries: map[string]string{}}
	if i := strings.LastIndex(*id, "-"); i >= 0 {
		r.AttemptID = (*id)[i+1:]
	}
	for _, s := range strings.Split(*require, ",") {
		if s = strings.TrimSpace(s); s != "" {
			r.Required = append(r.Required, s)
		}
	}
	var err error
	var n int
	if r.SourceDigest, n, err = SourceDigest(*src); err != nil {
		return err
	}
	r.SourceFiles = int64(n)
	r.Commit = run("git", "-C", *src, "rev-parse", "HEAD")
	if *bin == "" {
		exe, _ := os.Executable()
		*bin = filepath.Dir(exe)
	}
	for _, b := range []string{"dh", "dh-control", "dh-noded", "dh-conformance"} {
		if data, err := os.ReadFile(filepath.Join(*bin, b)); err == nil {
			r.Binaries[b] = "b3:" + envelope.HashBytes(data)
		}
	}
	envPath := filepath.Join(*dir, "env.json")
	if _, err := os.Stat(envPath); err != nil {
		b, _ := json.MarshalIndent(CollectEnv(*dir), "", "  ")
		if err := os.WriteFile(envPath, b, 0o644); err != nil {
			return err
		}
	}
	for _, p := range append([]string{envPath}, machines...) {
		var e Env
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &e); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		r.Machines = append(r.Machines, e)
	}
	if r.Steps, err = ReadSteps(*dir); err != nil {
		return err
	}
	r.Outcome, r.OutcomeReason = DecideOutcome(r.Required, r.Steps, *infra)
	for i, s := range r.Steps {
		if i == 0 || s.Start < r.Start {
			r.Start = s.Start
		}
		if s.End > r.End {
			r.End = s.End
		}
	}
	if r.Files, err = HashFiles(*dir); err != nil {
		return err
	}
	if *key == "" {
		*key = filepath.Join(filepath.Dir(filepath.Clean(*dir)), "validator")
	}
	signer, err := identity.LoadOrCreate(*key)
	if err != nil {
		return err
	}
	env, err := envelope.Sign(signer, "", KindRecord, r)
	if err != nil {
		return err
	}
	b, _ := canon.Wire(env)
	if err := os.WriteFile(filepath.Join(*dir, "record.json"), b, 0o444); err != nil {
		return err
	}
	fmt.Printf("%s %s (%s): %d step(s), %d file(s), source %s, signed by %s\n  scope: %s\n", r.EvidenceID, r.Outcome, r.OutcomeReason, len(r.Steps), len(r.Files), r.SourceDigest[:19], signer.ID, r.Scope)
	return nil
}

func cmdVerify(_ *cli.Operator, args []string) error {
	fs := flag.NewFlagSet("evidence verify", flag.ContinueOnError)
	dir := fs.String("dir", "", "evidence directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(*dir, "record.json"))
	if err != nil {
		return err
	}
	var env envelope.Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return err
	}
	var r Record
	verification, why := "VERIFIED", ""
	switch {
	case env.VerifySelf(KindRecord) != nil:
		verification, why = "UNVERIFIED", "record signature does not verify"
	case env.Decode(&r) != nil:
		verification, why = "UNVERIFIED", "record payload does not decode"
	default:
		if r.SchemaVersion == "" {
			// Schema 1 records (before outcomes) carry "format" and "passed".
			var legacy struct {
				Format string `json:"format"`
				Passed bool   `json:"passed"`
			}
			_ = env.Decode(&legacy)
			r.SchemaVersion, r.DigestVersion = legacy.Format+" (legacy)", "dh-src-digest/0"
			r.Outcome, r.OutcomeReason = OutcomeFail, "legacy record: passed=false"
			if legacy.Passed {
				r.Outcome, r.OutcomeReason = OutcomePass, "legacy record: passed=true (no required-step list; treat with care)"
			}
		}
		if p := Verify(r, *dir); len(p) > 0 {
			verification, why = "UNVERIFIED", fmt.Sprintf("%d file(s) do not match the record: %s", len(p), strings.Join(p, "; "))
		}
	}
	fmt.Printf("evidence:     %s (%s, stage %s, parent %q)\noutcome:      %s — %s\nverification: %s %s\nsigner:       %s\nsource:       %s (%s)\nscope:        %s\n",
		r.EvidenceID, r.SchemaVersion, r.Stage, r.ParentEvidence, r.Outcome, r.OutcomeReason, verification, why, env.Signer, r.SourceDigest, r.DigestVersion, r.Scope)
	for _, l := range r.Limitations {
		fmt.Printf("limitation:   %s\n", l)
	}
	for _, x := range r.ScopeExclusions {
		fmt.Printf("excluded:     %s\n", x)
	}
	if verification != "VERIFIED" {
		return errors.New("evidence is UNVERIFIED")
	}
	return nil
}
