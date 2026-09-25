// Command dh-conformance generates the dh/v1 test vectors and runs them
// against any implementation that speaks the adapter protocol.
//
//	dh-conformance gen   [-out conformance/vectors/dh-v1.json]
//	dh-conformance check [-vectors …]            regenerate and compare (drift guard)
//	dh-conformance run   -adapter "<command>" [-vectors …] [-ops canon,sign] [-report out.json]
//	dh-conformance run   -self                   the Go reference, in process
//	dh-conformance adapter                       serve the Go reference over stdio
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/conformance"
)

const defaultVectors = "conformance/vectors/dh-v1.json"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "gen":
		err = gen(os.Args[2:])
	case "check":
		err = check(os.Args[2:])
	case "run":
		err = run(os.Args[2:])
	case "adapter":
		err = conformance.Serve(os.Stdin, os.Stdout)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "dh-conformance:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: dh-conformance gen|check|run|adapter [flags]")
	os.Exit(2)
}

func encodeSuite(s *conformance.Suite) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gen(args []string) error {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	out := fs.String("out", defaultVectors, "vector file to write")
	fs.Parse(args)
	s, err := conformance.Generate()
	if err != nil {
		return err
	}
	b, err := encodeSuite(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %d vectors to %s\n", len(s.Vectors), *out)
	return nil
}

func check(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	path := fs.String("vectors", defaultVectors, "vector file")
	fs.Parse(args)
	s, err := conformance.Generate()
	if err != nil {
		return err
	}
	want, _ := encodeSuite(s)
	have, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	if !bytes.Equal(want, have) {
		return fmt.Errorf("%s differs from what the reference implementation generates; run dh-conformance gen and review the diff", *path)
	}
	fmt.Printf("%s: %d vectors match the reference implementation\n", *path, len(s.Vectors))
	return nil
}

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	path := fs.String("vectors", defaultVectors, "vector file")
	adapter := fs.String("adapter", "", "adapter command (run with /bin/sh -c)")
	self := fs.Bool("self", false, "run the Go reference adapter in process")
	ops := fs.String("ops", "", "comma-separated ops (default all)")
	report := fs.String("report", "", "write a JSON report here")
	timeout := fs.Duration("timeout", 60*time.Second, "per-vector timeout")
	allowSkip := fs.Bool("allow-skip", false, "unsupported ops do not fail the run")
	verbose := fs.Bool("v", false, "list skipped vectors")
	fs.Parse(args)
	s, err := conformance.Load(*path)
	if err != nil {
		return err
	}
	var t conformance.Transport
	name := *adapter
	switch {
	case *self:
		t, name = conformance.InProcess{}, "go-reference (in process)"
	case *adapter != "":
		p, err := conformance.StartProcess(*adapter)
		if err != nil {
			return err
		}
		t = p
	default:
		return fmt.Errorf("give -adapter \"<command>\" or -self")
	}
	opt := conformance.Options{Timeout: *timeout}
	if *ops != "" {
		opt.Ops = map[string]bool{}
		for _, o := range strings.Split(*ops, ",") {
			opt.Ops[strings.TrimSpace(o)] = true
		}
	}
	rep, err := conformance.Run(s, t, name, opt)
	cerr := t.Close()
	if err != nil {
		return err
	}
	rep.Print(os.Stdout, *verbose)
	if *report != "" {
		b, _ := canon.Wire(rep)
		if err := os.WriteFile(*report, b, 0o644); err != nil {
			return err
		}
	}
	if cerr != nil && rep.Fail == 0 {
		fmt.Fprintln(os.Stderr, "adapter exit:", cerr)
	}
	if !rep.OK(*allowSkip) {
		os.Exit(1)
	}
	return nil
}
