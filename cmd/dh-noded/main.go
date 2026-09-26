// dh-noded is the sovereign host agent.
//
//	dh-noded --data DIR --join dhjoin1.... --name host-a [flags]   run the agent
//	dh-noded rotate-key --data DIR [--grace 168h]                  rotate this host's signing key
//	dh-noded status --data DIR                                     print the agent's local status
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"
	"time"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/manifest"
	"decentralized.host/pkg/node"
	"decentralized.host/pkg/runtime/sandbox"
)

func main() {
	// If this process is a re-executed sandbox init, this never returns.
	sandbox.MaybeInit()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "rotate-key":
			rotate(os.Args[2:])
			return
		case "status":
			status(os.Args[2:])
			return
		case "ledger-seal":
			ledgerSeal(os.Args[2:])
			return
		}
	}
	var cfg node.Config
	var tiers, roles, features, cpu, mem, joinFile string
	host, _ := os.Hostname()
	flag.StringVar(&cfg.DataDir, "data", "./dh-node-data", "data directory (identity, policy, ledger, storage)")
	flag.StringVar(&cfg.JoinToken, "join", os.Getenv("DH_JOIN"), "join token (dhjoin1...), only needed on first start")
	flag.StringVar(&joinFile, "join-file", "", "read the join token from a file")
	flag.StringVar(&cfg.Name, "name", host, "host name shown to operators")
	flag.StringVar(&cfg.Region, "region", "local", "failure domain: region")
	flag.StringVar(&cfg.Zone, "zone", "", "failure domain: zone")
	flag.StringVar(&cfg.Host, "host", host, "failure domain: physical host")
	flag.StringVar(&tiers, "tiers", "trusted", "trust tiers this host belongs to (comma separated)")
	flag.StringVar(&roles, "roles", "", "roles: edge, edge-only (comma separated)")
	flag.StringVar(&features, "features", "", "advertised features (comma separated)")
	flag.StringVar(&cpu, "cpu", fmt.Sprint(goruntime.NumCPU()), "advertised CPU capacity")
	flag.StringVar(&mem, "mem", "4Gi", "advertised memory capacity")
	flag.StringVar(&cfg.MeshListen, "mesh", "0.0.0.0:51820", "UDP address for the WireGuard device")
	flag.StringVar(&cfg.MeshAdvertise, "mesh-advertise", "", "WireGuard endpoint peers dial (default: --mesh)")
	flag.StringVar(&cfg.StatusListen, "status", "127.0.0.1:0", "local status API (loopback)")
	flag.BoolVar(&cfg.Chaos, "chaos", false, "enable fault injection on the local status API")
	flag.StringVar(&cfg.Edge.HTTPListen, "edge-http", "", "edge role: HTTP listen address (ACME HTTP-01 is served here)")
	flag.StringVar(&cfg.Edge.HTTPSListen, "edge-https", "", "edge role: HTTPS listen address")
	flag.StringVar(&cfg.Edge.ACMEDirectory, "acme-directory", "", "ACME directory URL (e.g. https://acme-v02.api.letsencrypt.org/directory or a Pebble URL)")
	flag.StringVar(&cfg.Edge.ACMEEmail, "acme-email", "", "ACME account contact")
	flag.StringVar(&cfg.Edge.ACMECACert, "acme-ca", "", "PEM file trusted for the ACME directory (Pebble)")
	flag.StringVar(&cfg.Edge.ACMEDNS, "acme-dns", "", "DNS-01 provider for wildcard certificates: challtestsrv=http://host:port")
	flag.DurationVar(&cfg.Tick, "tick", time.Second, "reconciliation interval")
	flag.DurationVar(&cfg.AntiEntropyEvery, "anti-entropy", time.Minute, "re-verify each retained snapshot (and repair from peers) this often")
	flag.Parse()
	if joinFile != "" {
		b, err := os.ReadFile(joinFile)
		if err != nil {
			log.Fatal(err)
		}
		cfg.JoinToken = strings.TrimSpace(string(b))
	}
	cfg.Tiers = split(tiers)
	cfg.Roles = split(roles)
	cfg.Features = split(features)
	var err error
	if cfg.CPUMilli, err = manifest.ParseCPU(cpu); err != nil {
		log.Fatal(err)
	}
	if cfg.MemBytes, err = manifest.ParseBytes(mem); err != nil {
		log.Fatal(err)
	}
	cfg.Logger = log.New(os.Stderr, "dh-noded["+cfg.Name+"] ", log.LstdFlags|log.Lmsgprefix)
	a, err := node.New(cfg)
	if err != nil {
		cfg.Logger.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := a.Run(ctx); err != nil {
		cfg.Logger.Fatal(err)
	}
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func rotate(args []string) {
	fs := flag.NewFlagSet("rotate-key", flag.ExitOnError)
	data := fs.String("data", "./dh-node-data", "data directory")
	grace := fs.Duration("grace", 7*24*time.Hour, "how long the old key stays valid")
	_ = fs.Parse(args)
	pub, err := node.RotateKey(*data, *grace)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("rotated to %s; the old key stays valid for %s. Restart dh-noded to sign with the new key.\n", pub, *grace)
}

func status(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	data := fs.String("data", "./dh-node-data", "data directory")
	_ = fs.Parse(args)
	addr, err := os.ReadFile(filepath.Join(*data, "status.addr"))
	if err != nil {
		log.Fatal("agent status address not found (is dh-noded running with --status?)")
	}
	resp, err := http.Get("http://" + strings.TrimSpace(string(addr)) + "/status")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(os.Stdout, resp.Body)
}

// ledgerSeal is the local operator's recovery path for a corrupt host ledger
// (LEDGER_CORRUPT). It runs on the host only, with the agent stopped.
func ledgerSeal(args []string) {
	fs := flag.NewFlagSet("ledger-seal", flag.ExitOnError)
	data := fs.String("data", "./dh-node-data", "data directory")
	reason := fs.String("reason", "", "why the journal is being sealed (recorded in the new chain)")
	_ = fs.Parse(args)
	if addr, err := os.ReadFile(filepath.Join(*data, "status.addr")); err == nil {
		cl := &http.Client{Timeout: time.Second}
		if resp, err := cl.Get("http://" + strings.TrimSpace(string(addr)) + "/status"); err == nil {
			resp.Body.Close()
			log.Fatal("dh-noded is running on this data directory; stop it before sealing its ledger")
		}
	}
	host, _ := os.Hostname()
	sealed, first, err := audit.SealJournal(filepath.Join(*data, "journal.jsonl"), "local operator@"+host, *reason, time.Now().UnixMilli())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("sealed %s\nnew ledger started: seq %d %s\n  %s\n", sealed, first.Seq, first.Hash, first.Detail)
}
