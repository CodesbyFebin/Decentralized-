package control

import (
	"fmt"
	"sort"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
)

// buildBundle derives and signs the desired-state view for one host.
func (s *Server) buildBundle(nodeID string) (*envelope.Envelope, error) {
	var b api.Bundle
	var err error
	s.fsm.Read(func(st *State) {
		n := st.Nodes[nodeID]
		if n == nil {
			err = fmt.Errorf("unknown host %s", nodeID)
			return
		}
		b = api.Bundle{
			Cluster: st.Cluster, Root: st.Root, Roster: st.Roster, Issuer: s.id.ID, Issued: nowMs(), StateIndex: st.Index,
			Frozen: st.Frozen, RootRotation: st.RootRotation, LocalCA: st.LocalCACert,
			Node:        api.BundleNode{ID: n.ID, Name: n.Name, Status: n.Status, MeshIP: n.MeshIP, Keys: n.Keys, Roles: n.Roles},
			Assignments: []*envelope.Envelope{}, Peers: []api.Peer{}, Volumes: []api.VolumeDuty{}, Services: []api.Service{},
			Revoked: []string{}, RevokedKeys: []string{}, Artifacts: []api.ArtifactLocation{}, Publishers: append([]string{}, st.Publishers...),
			Attestations: []*envelope.Envelope{},
		}
		digests := map[string]bool{}
		for _, rec := range st.assignmentsFor(nodeID) {
			b.Assignments = append(b.Assignments, rec.Env)
			digests[rec.A.Digest] = true
		}
		for _, id := range st.sortedNodeIDs() {
			p := st.Nodes[id]
			if p.Status == "revoked" {
				b.Revoked = append(b.Revoked, p.ID)
			}
			for _, k := range p.Keys {
				if k.Revoked {
					b.RevokedKeys = append(b.RevokedKeys, k.Pub)
				}
			}
			if id == nodeID || p.Status == "revoked" || p.Status == "pending" || p.Binding == nil {
				continue
			}
			b.Peers = append(b.Peers, api.Peer{ID: p.ID, Name: p.Name, MeshIP: p.MeshIP, Status: p.Status, Roles: p.Roles, Keys: p.ValidKeys(nowMs()), Binding: p.Binding})
		}
		for _, m := range st.RosterBody.Members {
			if bind := st.MemberBind[m.ID]; bind != nil {
				b.Peers = append(b.Peers, api.Peer{ID: m.ID, Name: "cp:" + short(m.ID), MeshIP: st.MemberMesh[m.ID], Status: "control-plane", Roles: []string{"control-plane"}, Keys: []string{m.Pub}, Binding: bind})
			}
		}
		for _, vid := range st.sortedVolumeIDs() {
			v := st.Volumes[vid]
			if !contains(v.Members, nodeID) && v.Primary != nodeID {
				continue
			}
			duty := api.VolumeDuty{VolumeID: v.ID, App: v.App, Name: v.Name, Replica: v.Replica, Primary: v.Primary, Members: append([]string{}, v.Members...), Retain: v.Retain, Keep: []string{}}
			for _, sn := range v.Snapshots {
				duty.Keep = append(duty.Keep, sn.ID)
				if sn.ID == v.Committed {
					duty.Committed = &api.SnapshotRef{ID: sn.ID, Root: sn.Root, Bytes: sn.Bytes, Chunks: sn.Chunks}
				}
			}
			b.Volumes = append(b.Volumes, duty)
		}
		// Every host gets the service table: edge hosts route ingress with
		// it, and every host's mesh DNS and service proxies resolve from it.
		b.Services = st.services(s.obs)
		for d := range digests {
			if a := st.Artifacts[d]; a != nil {
				b.Artifacts = append(b.Artifacts, api.ArtifactLocation{Digest: a.Digest, Name: a.Name, Manifest: a.Manifest, Nodes: a.Holders, Bytes: a.Bytes})
			}
			if at := st.Attestations[d]; at != nil {
				b.Attestations = append(b.Attestations, at)
			}
		}
		sort.Slice(b.Artifacts, func(i, j int) bool { return b.Artifacts[i].Digest < b.Artifacts[j].Digest })
	})
	if err != nil {
		return nil, err
	}
	return envelope.Sign(s.id, "", envelope.KindBundle, b)
}

// services builds the edge routing table. An endpoint is listed only when
// the assignment is desired running and its host has reported it running
// in a signed observation; the edge then applies its own health probes.
func (st *State) services(cache *obsCache) []api.Service {
	var out []api.Service
	for _, name := range st.sortedAppNames() {
		app := st.Apps[name]
		if len(app.Manifest.Spec.Ingress) == 0 {
			continue
		}
		for _, in := range app.Manifest.Spec.Ingress {
			svc := api.Service{App: app.Name, Host: in.Host, Port: in.Port, TLS: in.TLS, Issuer: in.Issuer, HealthURL: app.Manifest.Spec.Health.HTTP, Endpoints: []api.Endpoint{}}
			for _, key := range st.sortedAssignmentKeys() {
				rec := st.Assignments[key]
				if rec.A.App != app.Name {
					continue
				}
				n := st.Nodes[rec.A.Node]
				// A revoked host may keep running admitted work, but it is
				// no longer a mesh peer, so it is never a routing target.
				if n == nil || n.Status == "revoked" {
					continue
				}
				w := latestWorkload(st, cache, rec.A.Node, rec.A.ID)
				if w == nil || w.Observed != "running" || w.MeshPort == 0 {
					continue
				}
				draining := rec.A.Desired != "running" || app.Deleted || n.Status == "draining" || w.Generation != rec.A.Generation
				svc.Endpoints = append(svc.Endpoints, api.Endpoint{Assignment: rec.A.ID, Node: n.ID, MeshIP: n.MeshIP, Port: w.MeshPort, Generation: w.Generation, Draining: draining, Observed: w.Observed})
			}
			out = append(out, svc)
		}
	}
	return out
}

// latestWorkload prefers the leader's fresher cached observation over the
// committed one.
func latestWorkload(st *State, cache *obsCache, node, id string) *api.WorkloadObs {
	if cache != nil {
		// The local cache wins only if it is at least as new as the replicated
		// record (an ex-leader's cache goes stale once another member leads).
		if c := cache.get(node); c != nil && (st.Nodes[node] == nil || c.Obs.Seq >= st.Nodes[node].ObserveSeq) {
			for i := range c.Obs.Workloads {
				if c.Obs.Workloads[i].Assignment == id {
					w := c.Obs.Workloads[i]
					return &w
				}
			}
			if n := st.Nodes[node]; n != nil && c.Obs.Seq >= n.ObserveSeq {
				return nil
			}
		}
	}
	return st.workloadObs(node, id)
}
