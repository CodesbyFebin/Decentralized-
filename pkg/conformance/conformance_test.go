package conformance

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const vectorFile = "../../conformance/vectors/dh-v1.json"

// The committed vectors must be exactly what the reference implementation
// generates: a protocol change shows up as a reviewed vector diff.
func TestVectorsMatchReference(t *testing.T) {
	s, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		t.Fatal(err)
	}
	have, err := os.ReadFile(vectorFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), have) {
		t.Fatal("conformance/vectors/dh-v1.json is stale: run `go run ./cmd/dh-conformance gen` and review the diff")
	}
}

func TestReferenceInProcess(t *testing.T) {
	s, err := Load(vectorFile)
	if err != nil {
		t.Fatal(err)
	}
	rep, _ := Run(s, InProcess{}, "go", Options{})
	if !rep.OK(false) {
		rep.Print(testWriter{t}, true)
		t.Fatal("reference adapter fails its own vectors")
	}
}

// The stdio loop is what external runners talk to.
func TestServeLineProtocol(t *testing.T) {
	in := bytes.NewBufferString(`{"id":"a","op":"merkle","input":{"ids":[]}}` + "\n\n" + `{"id":"b","op":"nope","input":{}}` + "\n")
	var out bytes.Buffer
	if err := Serve(in, &out); err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n"))
	if len(lines) != 2 || !bytes.Contains(lines[0], []byte(`"ok":true`)) || !bytes.Contains(lines[1], []byte(`"error":"unsupported"`)) {
		t.Fatalf("unexpected responses:\n%s", out.String())
	}
}

// The independent Python implementation must agree with every vector.
func TestPythonImplementation(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not installed")
	}
	if testing.Short() {
		t.Skip("slow: pure-Python BLAKE3")
	}
	s, err := Load(vectorFile)
	if err != nil {
		t.Fatal(err)
	}
	adapter, _ := filepath.Abs("../../conformance/python/adapter.py")
	p, err := StartProcess(py + " " + adapter)
	if err != nil {
		t.Fatal(err)
	}
	rep, _ := Run(s, p, "python", Options{})
	p.Close()
	if !rep.OK(false) {
		rep.Print(testWriter{t}, true)
		t.Fatal("Python implementation disagrees with the vectors")
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(b []byte) (int, error) { w.t.Log(string(b)); return len(b), nil }

var _ io.Writer = testWriter{}
