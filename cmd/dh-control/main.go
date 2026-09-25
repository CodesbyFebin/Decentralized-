// dh-control is a Decentralized.Host control-plane member: a Raft replica
// of the desired-state authority, the host channel, the operator API and the
// web console.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/web"
)

func main() {
	var cfg control.Config
	flag.StringVar(&cfg.DataDir, "data", "./dh-control-data", "data directory (identity, raft log, snapshots)")
	flag.StringVar(&cfg.APIListen, "api", "127.0.0.1:7700", "operator/host API and console listen address")
	flag.StringVar(&cfg.APIAdvertise, "api-advertise", "", "API address other members and hosts use (default: --api)")
	flag.StringVar(&cfg.RaftListen, "raft", "127.0.0.1:7800", "raft (mutual TLS) listen address")
	flag.StringVar(&cfg.RaftAdvertise, "raft-advertise", "", "raft address peers use (default: --raft)")
	flag.BoolVar(&cfg.TLS, "tls", false, "serve the API over TLS with the root-issued member certificate")
	flag.StringVar(&cfg.MeshListen, "mesh", "", "UDP address for this member's WireGuard device, e.g. 0.0.0.0:51900 (empty: no mesh)")
	flag.StringVar(&cfg.MeshAdvertise, "mesh-advertise", "", "WireGuard endpoint hosts dial (default: --mesh)")
	flag.DurationVar(&cfg.LostAfter, "lost-after", 30*time.Second, "mark a host lost after this long without a signed observation")
	flag.StringVar(&cfg.PostgresURL, "postgres", os.Getenv("DH_POSTGRES_URL"), "optional Postgres URL for the evidence mirror")
	flag.BoolVar(&cfg.FastRaft, "fast-raft", false, "shorter raft timeouts (tests and local clusters)")
	noWeb := flag.Bool("no-console", false, "do not serve the web console")
	flag.Parse()
	cfg.Logger = log.New(os.Stderr, "dh-control ", log.LstdFlags|log.Lmsgprefix)
	if !*noWeb {
		cfg.Web = web.FS()
		// Console development: serve files from disk instead of the embedded build.
		if dir := os.Getenv("DH_CONSOLE_DIR"); dir != "" {
			cfg.Web = os.DirFS(dir)
		}
	}
	s, err := control.New(cfg)
	if err != nil {
		cfg.Logger.Fatal(err)
	}
	if code := s.BootstrapCode(); code != "" {
		fmt.Fprintf(os.Stderr, "member %s awaiting bootstrap or join; one-time code in %s/bootstrap.code\n", s.ID().ID, cfg.DataDir)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := s.Run(ctx); err != nil {
		cfg.Logger.Fatal(err)
	}
}
