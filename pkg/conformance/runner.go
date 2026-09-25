package conformance

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"time"

	"decentralized.host/pkg/canon"
)

// Serve runs the Go reference adapter over the line protocol.
func Serve(r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	bw := bufio.NewWriter(w)
	for sc.Scan() {
		if len(bytes.TrimSpace(sc.Bytes())) == 0 {
			continue
		}
		b, _ := canon.Wire(handleLine(sc.Bytes()))
		bw.Write(b)
		bw.WriteByte('\n')
		if err := bw.Flush(); err != nil {
			return err
		}
	}
	return sc.Err()
}

// Load reads a vector file.
func Load(path string) (*Suite, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Suite
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if s.Protocol != Protocol {
		return nil, fmt.Errorf("vector file is for %q, not %q", s.Protocol, Protocol)
	}
	return &s, nil
}

// Result is the outcome of one vector.
type Result struct {
	ID       string `json:"id"`
	Op       string `json:"op"`
	Pass     bool   `json:"pass"`
	Skipped  bool   `json:"skipped,omitempty"`
	Expected string `json:"expected,omitempty"`
	Got      string `json:"got,omitempty"`
	Micros   int64  `json:"micros"`
}

// OpSummary counts results for one op.
type OpSummary struct {
	Pass, Fail, Skip int
}

// Report is a full run.
type Report struct {
	Protocol     string                `json:"protocol"`
	SuiteVersion int                   `json:"suiteVersion"`
	Adapter      string                `json:"adapter"`
	Started      int64                 `json:"started"`
	Millis       int64                 `json:"millis"`
	Pass         int                   `json:"pass"`
	Fail         int                   `json:"fail"`
	Skip         int                   `json:"skip"`
	ByOp         map[string]*OpSummary `json:"byOp"`
	Results      []Result              `json:"results"`
}

// OK reports whether every vector passed (skips count as failures unless
// allowSkip).
func (r *Report) OK(allowSkip bool) bool { return r.Fail == 0 && (allowSkip || r.Skip == 0) }

// Transport sends one request line and returns one response line.
type Transport interface {
	RoundTrip(ctx context.Context, line []byte) ([]byte, error)
	Close() error
}

// InProcess routes requests to the Go reference adapter through the same
// JSON encoding an external adapter would see.
type InProcess struct{}

func (InProcess) RoundTrip(_ context.Context, line []byte) ([]byte, error) {
	return canon.Wire(handleLine(line))
}
func (InProcess) Close() error { return nil }

// Process is an external adapter speaking the line protocol on stdio.
type Process struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

// StartProcess launches command through /bin/sh -c.
func StartProcess(command string) (*Process, error) {
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Process{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 1<<20)}, nil
}

func (p *Process) RoundTrip(ctx context.Context, line []byte) ([]byte, error) {
	type res struct {
		b   []byte
		err error
	}
	ch := make(chan res, 1)
	go func() {
		if _, err := p.in.Write(append(line, '\n')); err != nil {
			ch <- res{nil, err}
			return
		}
		b, err := p.out.ReadBytes('\n')
		ch <- res{b, err}
	}()
	select {
	case r := <-ch:
		return r.b, r.err
	case <-ctx.Done():
		_ = p.cmd.Process.Kill()
		return nil, errors.New("adapter timed out")
	}
}

func (p *Process) Close() error {
	p.in.Close()
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		return <-done
	}
}

// Options select what to run.
type Options struct {
	Ops     map[string]bool // empty = all
	Timeout time.Duration   // per vector
}

// Run executes every selected vector against t.
func Run(s *Suite, t Transport, name string, opt Options) (*Report, error) {
	if opt.Timeout == 0 {
		opt.Timeout = 60 * time.Second
	}
	rep := &Report{Protocol: s.Protocol, SuiteVersion: s.Version, Adapter: name, Started: time.Now().UnixMilli(), ByOp: map[string]*OpSummary{}}
	start := time.Now()
	for _, v := range s.Vectors {
		if len(opt.Ops) > 0 && !opt.Ops[v.Op] {
			continue
		}
		sum := rep.ByOp[v.Op]
		if sum == nil {
			sum = &OpSummary{}
			rep.ByOp[v.Op] = sum
		}
		res := runOne(t, v, opt.Timeout)
		switch {
		case res.Skipped:
			rep.Skip++
			sum.Skip++
		case res.Pass:
			rep.Pass++
			sum.Pass++
		default:
			rep.Fail++
			sum.Fail++
		}
		rep.Results = append(rep.Results, res)
	}
	rep.Millis = time.Since(start).Milliseconds()
	return rep, nil
}

func runOne(t Transport, v Vector, timeout time.Duration) Result {
	res := Result{ID: v.ID, Op: v.Op}
	input, err := Materialize(v.Op, v.Input)
	if err != nil {
		res.Got = "runner: " + err.Error()
		return res
	}
	line, _ := canon.Wire(request{ID: v.ID, Op: v.Op, Input: input})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	t0 := time.Now()
	raw, err := t.RoundTrip(ctx, line)
	res.Micros = time.Since(t0).Microseconds()
	want, _ := canon.Canonicalize(v.Expect)
	res.Expected = string(want)
	if err != nil {
		res.Got = "transport: " + err.Error()
		return res
	}
	var resp struct {
		ID     string          `json:"id"`
		OK     bool            `json:"ok"`
		Output json.RawMessage `json:"output"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		res.Got = "unparseable response: " + string(bytes.TrimSpace(raw))
		return res
	}
	if resp.ID != v.ID {
		res.Got = fmt.Sprintf("response id %q for request %q", resp.ID, v.ID)
		return res
	}
	var got any
	if resp.OK {
		got = map[string]any{"output": resp.Output}
	} else {
		if resp.Error == CodeUnsupported {
			res.Skipped = true
			res.Got = "unsupported"
			return res
		}
		got = map[string]any{"error": resp.Error}
	}
	gb, err := canon.Marshal(got)
	if err != nil {
		res.Got = "response has no canonical form: " + err.Error()
		return res
	}
	res.Got = string(gb)
	res.Pass = bytes.Equal(gb, want)
	if res.Pass {
		res.Expected, res.Got = "", ""
	}
	return res
}

// Print writes a human summary.
func (r *Report) Print(w io.Writer, verbose bool) {
	ops := make([]string, 0, len(r.ByOp))
	for op := range r.ByOp {
		ops = append(ops, op)
	}
	sort.SliceStable(ops, func(i, j int) bool { return opIndex(ops[i]) < opIndex(ops[j]) })
	fmt.Fprintf(w, "%s conformance (suite v%d) against %s\n", r.Protocol, r.SuiteVersion, r.Adapter)
	for _, op := range ops {
		s := r.ByOp[op]
		mark := "PASS"
		if s.Fail > 0 {
			mark = "FAIL"
		} else if s.Skip > 0 {
			mark = "SKIP"
		}
		fmt.Fprintf(w, "  %-4s %-18s %3d pass  %3d fail  %3d skip\n", mark, op, s.Pass, s.Fail, s.Skip)
	}
	for _, res := range r.Results {
		if res.Pass || (res.Skipped && !verbose) {
			continue
		}
		fmt.Fprintf(w, "\n  ✗ %s\n      expected %s\n      got      %s\n", res.ID, clip(res.Expected), clip(res.Got))
	}
	fmt.Fprintf(w, "\n%d pass, %d fail, %d skip in %dms\n", r.Pass, r.Fail, r.Skip, r.Millis)
}

func opIndex(op string) int {
	for i, o := range Ops {
		if o == op {
			return i
		}
	}
	return len(Ops)
}

func clip(s string) string {
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}
