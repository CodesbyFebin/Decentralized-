package control

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"time"

	"github.com/zeebo/blake3"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/peer"
)

var artifactNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

func regexpName(s string) bool { return artifactNameRE.MatchString(s) }

type b3hasher struct{ h *blake3.Hasher }

func newB3() *b3hasher                          { return &b3hasher{h: blake3.New()} }
func (b *b3hasher) Write(p []byte) (int, error) { return b.h.Write(p) }
func (b *b3hasher) hex() string                 { return hex.EncodeToString(b.h.Sum(nil)) }

func init() {
	register("operator-note", func(f *FSM, s *State, c *Command) *Result {
		d, err := decode[struct {
			Action   string `json:"action"`
			Resource string `json:"resource"`
			Detail   string `json:"detail"`
		}](c)
		if err != nil {
			return fail("DECODE", "%v", err)
		}
		s.audit(audit.Entry{TS: c.TS, Actor: c.Actor, Source: audit.SourceOperator, Action: d.Action, Resource: d.Resource, Detail: d.Detail})
		return ok("noted")
	})
}

// distributeArtifacts copies artifacts this member holds to hosts until
// each has at least two verified holders (or every ready host holds it).
// Each receiving host re-hashes every chunk before storing it.
func (s *Server) distributeArtifacts() {
	type job struct {
		a       Artifact
		targets map[string]string // node -> mesh ip
	}
	var jobs []job
	s.fsm.Read(func(st *State) {
		ready := map[string]string{}
		for _, id := range st.sortedNodeIDs() {
			n := st.Nodes[id]
			if usable(n) && n.Binding != nil && !contains(n.Roles, "edge-only") {
				ready[id] = n.MeshIP
			}
		}
		for _, d := range sortedKeys(st.Artifacts) {
			a := st.Artifacts[d]
			want := 2
			if len(ready) < want {
				want = len(ready)
			}
			have := 0
			for _, h := range a.Holders {
				if _, ok := ready[h]; ok {
					have++
				}
			}
			if have >= want || !s.cas.Has(a.Manifest) {
				continue
			}
			j := job{a: *a, targets: map[string]string{}}
			for _, id := range sortedKeys(ready) {
				if !contains(a.Holders, id) && len(j.targets)+have < want {
					j.targets[id] = ready[id]
				}
			}
			jobs = append(jobs, j)
		}
	})
	if len(jobs) == 0 {
		return
	}
	pc, err := s.peerClient(30 * time.Second)
	if err != nil {
		return
	}
	for _, j := range jobs {
		mb, err := s.cas.Get(j.a.Manifest)
		if err != nil {
			continue
		}
		var m api.ArtifactManifest
		if json.Unmarshal(mb, &m) != nil {
			continue
		}
		var holders []string
		for node, ip := range j.targets {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			okAll := pc.PutChunk(ctx, ip, j.a.Manifest, mb) == nil
			for _, cid := range m.Chunks {
				if !okAll {
					break
				}
				data, err := s.cas.Get(cid)
				if err != nil {
					okAll = false
					break
				}
				okAll = pc.PutChunk(ctx, ip, cid, data) == nil
			}
			cancel()
			if okAll {
				holders = append(holders, node)
			}
		}
		if len(holders) > 0 {
			a := j.a
			a.Holders = holders
			_, _ = s.propose("artifact", s.id.ID, a)
		}
	}
}

var _ = peer.Port
