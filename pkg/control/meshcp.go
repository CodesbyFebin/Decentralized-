package control

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/mesh"
	"decentralized.host/pkg/peer"
)

// meshClient is the member's WireGuard presence. Control-plane members join
// the mesh as non-workload peers so they can reach host agents (logs, host
// ledgers, artifact distribution) through the authenticated tunnel.
type meshClient struct {
	dev     *mesh.Device
	ip      string
	bound   string
	lastErr string
}

func (s *Server) meshHTTP(timeout time.Duration) (*http.Client, error) {
	s.meshMu.RLock()
	defer s.meshMu.RUnlock()
	if s.meshCli.dev == nil {
		return nil, fmt.Errorf("this control-plane member has no mesh device (%s)", s.meshCli.lastErr)
	}
	return s.meshCli.dev.HTTPClient(timeout), nil
}

func (s *Server) stopMesh() {
	s.meshMu.Lock()
	defer s.meshMu.Unlock()
	if s.meshCli.dev != nil {
		s.meshCli.dev.Close()
		s.meshCli.dev = nil
	}
}

// ensureMesh starts the device once the member has a mesh address, keeps its
// signed binding current, and configures every approved host as a peer.
func (s *Server) ensureMesh() {
	if s.cfg.MeshListen == "" {
		return
	}
	var ip string
	var peers []mesh.PeerConfig
	var bound *envelope.Envelope
	s.fsm.Read(func(st *State) {
		ip = st.MemberMesh[s.id.ID]
		bound = st.MemberBind[s.id.ID]
		for _, id := range st.sortedNodeIDs() {
			n := st.Nodes[id]
			if n.Status == "revoked" || n.Binding == nil {
				continue
			}
			var b api.WGBinding
			if n.Binding.Decode(&b) == nil {
				peers = append(peers, mesh.PeerConfig{Node: n.ID, WGPub: b.WGPub, Endpoint: b.Endpoint, MeshIP: b.MeshIP})
			}
		}
	})
	if ip == "" {
		return
	}
	s.meshMu.Lock()
	if s.meshCli.dev == nil || s.meshCli.ip != ip {
		if s.meshCli.dev != nil {
			s.meshCli.dev.Close()
		}
		key, err := mesh.LoadOrCreateKey(filepath.Join(s.cfg.DataDir, "wg.key"))
		if err == nil {
			_, portStr, _ := net.SplitHostPort(s.cfg.MeshListen)
			port, _ := strconv.Atoi(portStr)
			var dev *mesh.Device
			dev, err = mesh.Start(key, ip, port)
			if err == nil {
				s.meshCli = meshClient{dev: dev, ip: ip}
				// Answer peer pings so hosts can measure RTT to members.
				if ln, lerr := dev.ListenTCP(peer.Port); lerr == nil {
					mux := http.NewServeMux()
					mux.HandleFunc("GET /peer/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, `{"member":%q,"ts":%d}`, s.id.ID, nowMs())
					})
					go (&http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}).Serve(ln)
				}
			}
		}
		if err != nil {
			s.meshCli.lastErr = err.Error()
			s.meshMu.Unlock()
			return
		}
	}
	dev := s.meshCli.dev
	s.meshMu.Unlock()
	if err := dev.SetPeers(peers); err != nil {
		s.meshMu.Lock()
		s.meshCli.lastErr = err.Error()
		s.meshMu.Unlock()
	}
	adv := s.cfg.MeshAdvertise
	if adv == "" {
		adv = s.cfg.MeshListen
	}
	want := api.WGBinding{Node: s.id.ID, WGPub: dev.PublicKey(), Endpoint: adv, MeshIP: ip}
	if bound != nil {
		var have api.WGBinding
		if bound.Decode(&have) == nil && have.WGPub == want.WGPub && have.Endpoint == want.Endpoint && have.MeshIP == want.MeshIP {
			return
		}
	}
	want.TS = nowMs()
	env, err := envelope.Sign(s.id, "", envelope.KindWGBinding, want)
	if err != nil {
		return
	}
	if s.IsLeader() {
		_, _ = s.propose("wg-binding", s.id.ID, env)
	} else {
		s.forwardInternal("/v1/binding", env)
	}
}
