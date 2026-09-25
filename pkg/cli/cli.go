// Package cli implements `dh`, the operator CLI. It holds the cluster root
// key on the operator's machine, mints short-lived capabilities for every
// call, and never depends on the browser.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/pki"
)

// Config is the operator's view of one cluster.
type Config struct {
	Cluster   string   `json:"cluster"`
	Endpoints []string `json:"endpoints"`
	TLS       bool     `json:"tls"`
}

// Operator holds the root key and cluster config.
type Operator struct {
	Home   string
	Dir    string
	Cfg    Config
	Root   *identity.Identity
	CAPEM  []byte
	client *http.Client
	scheme string
}

type command struct {
	name string
	help string
	run  func(op *Operator, args []string) error
	root bool // needs the root key
}

var commands []command

func reg(name, help string, needRoot bool, run func(op *Operator, args []string) error) {
	commands = append(commands, command{name: name, help: help, run: run, root: needRoot})
}

func defaultHome() string {
	if h := os.Getenv("DH_HOME"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dh")
}

// Main runs the CLI.
func Main(argv []string) int {
	global := flag.NewFlagSet("dh", flag.ContinueOnError)
	home := global.String("home", defaultHome(), "operator home (root keys and cluster configs)")
	cluster := global.String("cluster", os.Getenv("DH_CLUSTER"), "cluster name (default: the current cluster in --home)")
	api := global.String("api", os.Getenv("DH_API"), "override control-plane API address")
	global.Usage = usage
	if err := global.Parse(argv); err != nil {
		return 2
	}
	args := global.Args()
	if len(args) == 0 {
		usage()
		return 2
	}
	// Longest matching command name (commands have up to three words).
	var cmd *command
	var rest []string
	for n := 3; n >= 1 && cmd == nil; n-- {
		if len(args) < n {
			continue
		}
		name := strings.Join(args[:n], " ")
		for i := range commands {
			if commands[i].name == name {
				cmd, rest = &commands[i], args[n:]
				break
			}
		}
	}
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "dh: unknown command %q\n\n", strings.Join(args, " "))
		usage()
		return 2
	}
	op := &Operator{Home: *home}
	for _, a := range rest {
		if a == "-h" || a == "-help" || a == "--help" {
			// Help needs no cluster: print the command's summary, then its
			// flags if it defines any.
			fmt.Fprintf(os.Stderr, "dh %s — %s\n", cmd.name, cmd.help)
			func() {
				defer func() { _ = recover() }()
				_ = cmd.run(op, []string{"-h"})
			}()
			return 0
		}
	}
	if cmd.root {
		if err := op.load(*cluster); err != nil {
			fmt.Fprintln(os.Stderr, "dh:", err)
			return 1
		}
		if *api != "" {
			op.Cfg.Endpoints = []string{*api}
		}
	}
	if err := cmd.run(op, rest); err != nil {
		fmt.Fprintln(os.Stderr, "dh:", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `dh — Decentralized.Host operator CLI (protocol dh/v1)

Usage: dh [--home DIR] [--cluster NAME] [--api ADDR] <command> [args]

`)
	sorted := append([]command(nil), commands...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].name < sorted[j].name })
	w := tabwriter.NewWriter(os.Stderr, 2, 4, 2, ' ', 0)
	for _, c := range sorted {
		fmt.Fprintf(w, "  %s\t%s\n", c.name, c.help)
	}
	w.Flush()
}

func (op *Operator) clusterDir(name string) string { return filepath.Join(op.Home, name) }

func (op *Operator) load(cluster string) error {
	if cluster == "" {
		b, err := os.ReadFile(filepath.Join(op.Home, "current"))
		if err != nil {
			return fmt.Errorf("no cluster selected: run `dh init --cluster NAME` or pass --cluster (home %s)", op.Home)
		}
		cluster = strings.TrimSpace(string(b))
	}
	op.Dir = op.clusterDir(cluster)
	b, err := os.ReadFile(filepath.Join(op.Dir, "config.json"))
	if err != nil {
		return fmt.Errorf("cluster %s: %w", cluster, err)
	}
	if err := json.Unmarshal(b, &op.Cfg); err != nil {
		return err
	}
	op.Root, err = identity.Load(filepath.Join(op.Dir, "root"))
	if err != nil {
		return fmt.Errorf("root key: %w", err)
	}
	op.CAPEM, _ = os.ReadFile(filepath.Join(op.Dir, "root-ca.pem"))
	op.client = &http.Client{Timeout: 60 * time.Second}
	op.scheme = "http"
	if op.Cfg.TLS {
		conf, err := pki.ClientTLS(op.CAPEM)
		if err != nil {
			return err
		}
		op.client.Transport = &http.Transport{TLSClientConfig: conf}
		op.scheme = "https"
	}
	return nil
}

func (op *Operator) save() error {
	b, _ := json.MarshalIndent(op.Cfg, "", "  ")
	return os.WriteFile(filepath.Join(op.Dir, "config.json"), b, 0o600)
}

// Token mints an operator capability (root → sealed session).
func (op *Operator) Token(ttl time.Duration, actions []string, note string) string {
	now := time.Now().UnixMilli()
	if note == "" {
		host, _ := os.Hostname()
		note = "cli@" + host
	}
	tok, err := capability.Mint(op.Root, capability.Caveats{Actions: actions, Resources: []string{"cluster/" + op.Cfg.Cluster}, NotBefore: now - 60_000, Expires: now + ttl.Milliseconds()}, "", note)
	if err != nil {
		panic(err)
	}
	return tok.Encode()
}

func (op *Operator) bearer() string {
	return op.Token(5*time.Minute, []string{"api.read", "api.write", "api.admin"}, "")
}

// APIError is a non-2xx API reply.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("%d: %s", e.Code, e.Message) }

// Do calls the API on the first reachable endpoint.
func (op *Operator) Do(method, path string, body any, out any) error {
	if len(op.Cfg.Endpoints) == 0 {
		return errors.New("no control-plane endpoint configured; run `dh cp bootstrap`")
	}
	var payload []byte
	switch b := body.(type) {
	case nil:
	case []byte:
		payload = b
	case string:
		payload = []byte(b)
	default:
		payload, _ = canon.Wire(b)
	}
	var lastErr error
	for _, ep := range op.Cfg.Endpoints {
		req, err := http.NewRequest(method, op.scheme+"://"+ep+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+op.bearer())
		req.Header.Set("Content-Type", "application/json")
		resp, err := op.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 503 || resp.StatusCode == 502 {
			lastErr = &APIError{Code: resp.StatusCode, Message: msgOf(data)}
			continue
		}
		if resp.StatusCode >= 300 {
			return &APIError{Code: resp.StatusCode, Message: msgOf(data)}
		}
		if out == nil {
			return nil
		}
		if raw, ok := out.(*[]byte); ok {
			*raw = data
			return nil
		}
		return json.Unmarshal(data, out)
	}
	return fmt.Errorf("no control-plane member answered: %v", lastErr)
}

func msgOf(data []byte) string {
	var m struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &m) == nil && m.Message != "" {
		return m.Message
	}
	return strings.TrimSpace(string(data))
}

// Result mirrors control.Result.
type Result struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Data    any    `json:"data"`
}

// Out receives informational output; library users (dev clusters, tests,
// the chaos suite) may silence it.
var Out io.Writer = os.Stdout

func printResult(r Result) error {
	if !r.OK {
		return errors.New(r.Message)
	}
	fmt.Fprintln(Out, r.Message)
	return nil
}

func table(headers ...string) *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	return w
}

func short(s string) string {
	s = strings.TrimPrefix(s, "b3:")
	if len(s) > 12 {
		return s[:12]
	}
	if s == "" {
		return "-"
	}
	return s
}

func ago(ms int64) string {
	if ms <= 0 {
		return "never"
	}
	d := time.Since(time.UnixMilli(ms)).Round(time.Second)
	return d.String() + " ago"
}

func flags(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ContinueOnError) }
