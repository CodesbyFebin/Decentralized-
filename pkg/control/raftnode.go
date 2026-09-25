package control

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// tlsStream is a raft.StreamLayer over mutually authenticated TLS. Both
// sides present root-issued member certificates; the verifier also checks
// the peer key is in the current roster.
type tlsStream struct {
	ln   net.Listener
	adv  net.Addr
	conf *tls.Config
}

func newTLSStream(bind, advertise string, conf *tls.Config) (*tlsStream, error) {
	ln, err := tls.Listen("tcp", bind, conf)
	if err != nil {
		return nil, err
	}
	adv, err := net.ResolveTCPAddr("tcp", advertise)
	if err != nil {
		ln.Close()
		return nil, err
	}
	return &tlsStream{ln: ln, adv: adv, conf: conf}, nil
}

func (t *tlsStream) Accept() (net.Conn, error) { return t.ln.Accept() }
func (t *tlsStream) Close() error              { return t.ln.Close() }
func (t *tlsStream) Addr() net.Addr            { return t.adv }
func (t *tlsStream) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	return tls.DialWithDialer(d, "tcp", string(addr), t.conf)
}

// raftNode bundles the raft instance and its stores.
type raftNode struct {
	r     *raft.Raft
	trans *raft.NetworkTransport
	logs  *raftboltdb.BoltStore
}

type raftOptions struct {
	Dir          string
	ID           string
	Bind         string
	Advertise    string
	TLS          *tls.Config
	FSM          *FSM
	Bootstrap    bool
	LogOutput    io.Writer
	FastTimeouts bool
}

func startRaft(o raftOptions) (*raftNode, error) {
	if err := os.MkdirAll(o.Dir, 0o700); err != nil {
		return nil, err
	}
	cfg := raft.DefaultConfig()
	cfg.LocalID = raft.ServerID(o.ID)
	cfg.HeartbeatTimeout = 1000 * time.Millisecond
	cfg.ElectionTimeout = 1000 * time.Millisecond
	cfg.LeaderLeaseTimeout = 500 * time.Millisecond
	cfg.CommitTimeout = 20 * time.Millisecond
	cfg.SnapshotThreshold = 4096
	cfg.SnapshotInterval = 60 * time.Second
	cfg.TrailingLogs = 8192
	if o.FastTimeouts {
		cfg.HeartbeatTimeout = 500 * time.Millisecond
		cfg.ElectionTimeout = 500 * time.Millisecond
		cfg.LeaderLeaseTimeout = 250 * time.Millisecond
	}
	out := o.LogOutput
	if out == nil {
		out = io.Discard
	}
	cfg.Logger = hclog.New(&hclog.LoggerOptions{Name: "raft", Level: hclog.Warn, Output: out})

	store, err := raftboltdb.New(raftboltdb.Options{Path: filepath.Join(o.Dir, "raft.db")})
	if err != nil {
		return nil, fmt.Errorf("raft log store: %w", err)
	}
	snaps, err := raft.NewFileSnapshotStore(o.Dir, 3, out)
	if err != nil {
		store.Close()
		return nil, err
	}
	stream, err := newTLSStream(o.Bind, o.Advertise, o.TLS)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("raft listen %s: %w", o.Bind, err)
	}
	trans := raft.NewNetworkTransport(stream, 3, 5*time.Second, out)
	r, err := raft.NewRaft(cfg, o.FSM, store, store, snaps, trans)
	if err != nil {
		trans.Close()
		store.Close()
		return nil, err
	}
	if o.Bootstrap {
		has, err := raft.HasExistingState(store, store, snaps)
		if err != nil {
			return nil, err
		}
		if !has {
			f := r.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{ID: cfg.LocalID, Address: raft.ServerAddress(o.Advertise)}}})
			if err := f.Error(); err != nil && !errors.Is(err, raft.ErrCantBootstrap) {
				return nil, err
			}
		}
	}
	return &raftNode{r: r, trans: trans, logs: store}, nil
}

func (n *raftNode) shutdown() {
	if n == nil {
		return
	}
	_ = n.r.Shutdown().Error()
	_ = n.trans.Close()
	_ = n.logs.Close()
}

// propose submits a command and waits for it to be applied.
func (n *raftNode) propose(cmd *Command, timeout time.Duration) (*Result, error) {
	if n == nil {
		return nil, errors.New("control plane is not bootstrapped")
	}
	b, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	f := n.r.Apply(b, timeout)
	if err := f.Error(); err != nil {
		return nil, err
	}
	res, _ := f.Response().(*Result)
	if res == nil {
		return nil, errors.New("apply returned no result")
	}
	return res, nil
}
