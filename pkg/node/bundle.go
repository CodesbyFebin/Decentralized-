package node

import (
	"errors"
	"fmt"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/manifest"
)

// hostRequest mirrors control.HostRequest (kept separate to avoid an import cycle).
type hostRequest struct {
	Node      string `json:"node"`
	TS        int64  `json:"ts"`
	Op        string `json:"op"`
	Arg       string `json:"arg"`
	LastIndex int64  `json:"lastIndex"`
}

func (a *Agent) request(op, arg string) (*envelope.Envelope, error) {
	a.mu.RLock()
	last := a.st.LastIndex
	a.mu.RUnlock()
	return a.sign(envelope.KindRequest, hostRequest{Node: a.st.NodeID, TS: a.now(), Op: op, Arg: arg, LastIndex: last})
}

func (a *Agent) enrollEnvelope() (*envelope.Envelope, error) {
	caps := a.cfg.CPUMilli
	id := a.key()
	e := api.Enroll{ID: a.st.NodeID, Name: a.cfg.Name, Pub: id.PubString(), Arch: arch(), OS: goos(), Tiers: a.cfg.Tiers,
		Region: a.cfg.Region, Zone: a.cfg.Zone, Host: a.cfg.Host, CPUMilli: caps, MemBytes: a.cfg.MemBytes, DiskBytes: diskTotal(a.cfg.DataDir),
		Roles: a.cfg.Roles, Features: a.cfg.Features, JoinToken: a.st.JoinCap, PolicyHash: a.pol.Hash(), Policy: a.pol.Summary(), TS: a.now()}
	if e.Tiers == nil {
		e.Tiers = []string{"trusted"}
	}
	if e.Roles == nil {
		e.Roles = []string{}
	}
	if e.Features == nil {
		e.Features = []string{}
	}
	// After a key rotation the genesis id no longer derives from the current
	// key, so enrollment (self-certifying) is only sent with the genesis key.
	if identity.NodeID(id.Pub) != a.st.NodeID {
		return nil, errors.New("re-enrollment after key rotation is not needed")
	}
	return envelope.Sign(id, "", envelope.KindEnroll, e)
}

func (a *Agent) enroll() error {
	env, err := a.enrollEnvelope()
	if err != nil {
		return err
	}
	var res struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	if err := a.cp.post("/v1/enroll", env, &res); err != nil {
		return err
	}
	a.mu.Lock()
	first := !a.st.Enrolled
	a.st.Enrolled = true
	a.mu.Unlock()
	if first {
		a.record("enroll", "node/"+a.st.NodeID, 0, res.Message)
	}
	return nil
}

// syncBundle fetches and verifies the signed desired state.
func (a *Agent) syncBundle() {
	if a.chaos.dropCP.Load() {
		a.markStale("control plane unreachable (chaos: connectivity dropped)")
		return
	}
	a.mu.RLock()
	enrolled := a.st.Enrolled
	a.mu.RUnlock()
	if !enrolled {
		if err := a.enroll(); err != nil {
			a.markStale("enrollment: " + err.Error())
			return
		}
	}
	req, err := a.request("bundle", "")
	if err != nil {
		a.markStale(err.Error())
		return
	}
	var out struct {
		Bundle *envelope.Envelope `json:"bundle"`
	}
	if err := a.cp.post("/v1/bundle", req, &out); err != nil {
		var he *HTTPError
		if errors.As(err, &he) && he.Code == 401 {
			// Unknown to this control plane (for example after a restore): re-enroll.
			a.mu.Lock()
			a.st.Enrolled = false
			a.mu.Unlock()
		}
		a.markStale(err.Error())
		return
	}
	b, why := a.verifyBundle(out.Bundle)
	if b == nil {
		a.mu.Lock()
		a.trusted, a.trustWhy, a.fresh = false, why, false
		a.mu.Unlock()
		a.setMode("untrusted-plane", why)
		return
	}
	now := a.now()
	skew := b.Issued - now
	a.mu.Lock()
	a.bundle, a.bundleEnv, a.trusted, a.trustWhy = b, out.Bundle, true, ""
	a.skewMs = skew
	a.lastBundleAt = time.Now()
	a.fresh = now-b.Issued <= a.pol.FreshWindowMs && skew <= a.pol.MaxClockSkewMs
	a.st.LastIndex, a.st.LastIssued = b.StateIndex, b.Issued
	var eps []string
	if b.Roster != nil {
		var r api.Roster
		if b.Roster.Decode(&r) == nil {
			for _, m := range r.Members {
				if m.APIAddr != "" {
					eps = append(eps, m.APIAddr)
				}
			}
		}
	}
	a.mu.Unlock()
	a.cp.setEndpoints(eps)
	switch {
	case skew > a.pol.MaxClockSkewMs:
		a.setMode("clock-skew", fmt.Sprintf("bundle issued %dms ahead of host clock (limit %dms); holding admitted work, refusing new work", skew, a.pol.MaxClockSkewMs))
	case now-b.Issued > a.pol.FreshWindowMs:
		a.setMode("clock-skew", fmt.Sprintf("bundle is %dms old on arrival: host clock ahead or plane slow; holding admitted work", now-b.Issued))
	case b.Frozen:
		a.setMode("frozen-hold", "control plane frozen: holding admitted work, refusing new work")
	case a.journal.Corrupt() != nil:
		br := a.journal.Corrupt()
		a.setMode("ledger-corrupt", fmt.Sprintf("%s at seq %d", br.Reason, br.Seq))
	default:
		a.setMode("normal", "")
	}
}

func (a *Agent) markStale(why string) {
	a.mu.Lock()
	a.lastErr = why
	stale := a.bundle == nil || a.now()-a.bundle.Issued > a.pol.FreshWindowMs
	if stale {
		a.fresh = false
	}
	a.mu.Unlock()
	if stale {
		a.setMode("offline-hold", "control plane unreachable or stale: "+why+"; admitted work continues, new work refused")
	}
}

func (a *Agent) setMode(mode, detail string) {
	a.mu.Lock()
	changed := a.mode != mode
	a.mode, a.modeDetail = mode, detail
	a.st.Mode = mode
	a.mu.Unlock()
	if changed {
		a.record("host-mode", "node/"+a.st.NodeID, 0, fmt.Sprintf("%s: %s", mode, detail))
	}
}

// verifyBundle returns the decoded bundle or why it is not trusted.
func (a *Agent) verifyBundle(env *envelope.Envelope) (*api.Bundle, string) {
	if env == nil {
		return nil, "empty bundle"
	}
	var b api.Bundle
	if err := env.Decode(&b); err != nil {
		return nil, "undecodable bundle: " + err.Error()
	}
	a.mu.RLock()
	root, cluster, lastIndex := a.st.Root, a.st.Cluster, a.st.LastIndex
	a.mu.RUnlock()
	// Root rotation: accepted only when signed by the root this host pinned.
	if b.Root != root {
		if b.RootRotation == nil || b.RootRotation.VerifyKey(envelope.KindRotation, root) != nil {
			return nil, fmt.Sprintf("bundle root %s is not the pinned root %s and no rotation signed by the pinned root was presented", short(b.Root), short(root))
		}
		var rot api.Rotation
		_ = b.RootRotation.Decode(&rot)
		if rot.OldPub != root || rot.NewPub != b.Root {
			return nil, "root rotation statement does not connect the pinned root to the bundle root"
		}
		a.mu.Lock()
		a.st.Root = b.Root
		a.mu.Unlock()
		a.record("root-rotate", "cluster/"+cluster, 0, fmt.Sprintf("pinned root %s → %s (rotation signed by the previously pinned root)", short(root), short(b.Root)))
		root = b.Root
	}
	if b.Cluster != cluster {
		return nil, fmt.Sprintf("bundle is for cluster %q", b.Cluster)
	}
	if b.Roster == nil || b.Roster.VerifyKey(envelope.KindRoster, root) != nil {
		return nil, "roster is not signed by the pinned root"
	}
	var r api.Roster
	if err := b.Roster.Decode(&r); err != nil {
		return nil, "roster: " + err.Error()
	}
	var member *api.Member
	for i := range r.Members {
		if r.Members[i].ID == env.Signer {
			member = &r.Members[i]
		}
	}
	if member == nil {
		return nil, fmt.Sprintf("bundle signer %s is not in the root-signed roster", short(env.Signer))
	}
	if err := env.VerifyKey(envelope.KindBundle, member.Pub); err != nil {
		return nil, "bundle signature: " + err.Error()
	}
	if b.Node.ID != a.st.NodeID {
		return nil, "bundle is for another host"
	}
	if b.StateIndex < lastIndex {
		return nil, fmt.Sprintf("rollback rejected: bundle state index %d is older than accepted index %d", b.StateIndex, lastIndex)
	}
	return &b, ""
}

// verifyAssignment checks an assignment envelope against the verified roster.
func (a *Agent) verifyAssignment(b *api.Bundle, env *envelope.Envelope) (api.Assignment, bool, string) {
	var as api.Assignment
	if err := env.Decode(&as); err != nil {
		return as, false, "undecodable assignment"
	}
	var r api.Roster
	_ = b.Roster.Decode(&r)
	for _, m := range r.Members {
		if m.ID == env.Signer {
			if err := env.VerifyKey(envelope.KindAssignment, m.Pub); err != nil {
				return as, false, err.Error()
			}
			return as, true, "signed by roster member " + short(m.ID)
		}
	}
	return as, false, "assignment signer is not a roster member"
}

// verifyPeerBinding checks a peer's WireGuard binding is signed by the peer's own key.
func verifyPeerBinding(p api.Peer) (api.WGBinding, bool) {
	var wb api.WGBinding
	if p.Binding == nil || p.Binding.Decode(&wb) != nil {
		return wb, false
	}
	if p.Binding.VerifyKey(envelope.KindWGBinding, p.Keys...) != nil {
		return wb, false
	}
	if wb.Node != p.ID || wb.MeshIP != p.MeshIP || p.Binding.Signer != p.ID {
		return wb, false
	}
	return wb, true
}

var _ = manifest.Digest
