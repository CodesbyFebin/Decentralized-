package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/edge"
)

type edgeRunner interface {
	observe() api.EdgeObs
}

type edgeRole struct {
	e *edge.Edge
}

func (r *edgeRole) observe() api.EdgeObs { return r.e.Observe() }

func (a *Agent) meshDialer() edge.Dialer {
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	if a.dev == nil {
		return nil
	}
	return a.dev.DialContext
}

// startEdge runs the edge role: the proxy, certificate manager, and a loop
// feeding it the control plane's service table.
func (a *Agent) startEdge(ctx context.Context) error {
	local := func(names []string, pubPEM string) (string, error) {
		arg, _ := json.Marshal(map[string]any{"names": names, "pubPem": pubPEM})
		req, err := a.request("cert", string(arg))
		if err != nil {
			return "", err
		}
		var out struct {
			Cert string `json:"cert"`
			CA   string `json:"ca"`
		}
		if err := a.cp.post("/v1/cert", req, &out); err != nil {
			return "", err
		}
		return out.Cert, nil
	}
	cm := edge.NewCertManager(filepath.Join(a.cfg.DataDir, "edge"), local, edge.ACMEConfig{
		Directory: a.cfg.Edge.ACMEDirectory, Email: a.cfg.Edge.ACMEEmail, CACert: a.cfg.Edge.ACMECACert, DNS: a.cfg.Edge.ACMEDNS,
	}, a.log)
	e := edge.New(edge.Config{Node: a.st.NodeID, DataDir: a.cfg.DataDir, HTTPListen: a.cfg.Edge.HTTPListen, HTTPSListen: a.cfg.Edge.HTTPSListen,
		Dial: a.meshDialer, Certs: cm, Logger: a.log})
	if err := e.Start(); err != nil {
		return err
	}
	a.mu.Lock()
	a.edge = &edgeRole{e: e}
	a.mu.Unlock()
	go cm.Run(ctx)
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				e.Close()
				return
			case <-t.C:
			}
			a.mu.RLock()
			b := a.bundle
			a.mu.RUnlock()
			if b != nil {
				e.Update(b.Services)
			}
		}
	}()
	a.record("edge-start", "node/"+a.st.NodeID, 0, fmt.Sprintf("edge proxy http %s https %s; acme %q", a.cfg.Edge.HTTPListen, a.cfg.Edge.HTTPSListen, a.cfg.Edge.ACMEDirectory))
	return nil
}

var _ = net.Dial
