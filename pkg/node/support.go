package node

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/peer"
	"decentralized.host/pkg/storage"
)

func goos() string { return goruntime.GOOS }

func kernelVersion() string {
	out, err := exec.Command("uname", "-sr").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func diskTotal(dir string) int64 {
	var st syscall.Statfs_t
	if syscall.Statfs(dir, &st) != nil {
		return 0
	}
	return int64(st.Blocks) * int64(st.Bsize)
}

func diskFree(dir string) int64 {
	var st syscall.Statfs_t
	if syscall.Statfs(dir, &st) != nil {
		return -1
	}
	return int64(st.Bavail) * int64(st.Bsize)
}

// probeTools reports the presence of tools the blueprint asks about. A
// missing tool is reported as missing, not hidden.
func probeTools() []api.Check {
	var out []api.Check
	check := func(name string, args ...string) {
		c := exec.Command(args[0], args[1:]...)
		b, err := c.CombinedOutput()
		if err != nil {
			out = append(out, api.Check{Name: name, OK: false, Detail: "not installed or not runnable"})
			return
		}
		first := strings.SplitN(strings.TrimSpace(string(b)), "\n", 2)[0]
		out = append(out, api.Check{Name: name, OK: true, Detail: first})
	}
	check("PHP", "php", "-v")
	if _, err := exec.LookPath("php"); err == nil {
		b, _ := exec.Command("php", "-m").Output()
		has := false
		for _, l := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(strings.ToLower(l)) == "redis" {
				has = true
			}
		}
		out = append(out, api.Check{Name: "phpredis", OK: has, Detail: map[bool]string{true: "redis module loaded", false: "redis module not loaded"}[has]})
	} else {
		out = append(out, api.Check{Name: "phpredis", OK: false, Detail: "PHP not installed"})
	}
	if _, err := exec.LookPath("runsc"); err == nil {
		out = append(out, api.Check{Name: "gVisor (runsc)", OK: true, Detail: "binary present (not wired as a runtime)"})
	} else {
		out = append(out, api.Check{Name: "gVisor (runsc)", OK: false, Detail: "not installed"})
	}
	if _, err := os.Stat("/dev/kvm"); err == nil {
		out = append(out, api.Check{Name: "KVM (Firecracker)", OK: true, Detail: "/dev/kvm present (Firecracker not wired as a runtime)"})
	} else {
		out = append(out, api.Check{Name: "KVM (Firecracker)", OK: false, Detail: "/dev/kvm not present"})
	}
	return out
}

// probeHTTP measures one health probe.
func probeHTTP(url string, timeout time.Duration) api.HealthObs {
	if timeout <= 0 {
		timeout = time.Second
	}
	start := time.Now()
	c := &http.Client{Timeout: timeout}
	resp, err := c.Get(url)
	obs := api.HealthObs{CheckedAt: time.Now().UnixMilli(), LatencyUs: time.Since(start).Microseconds()}
	if err != nil {
		obs.Detail = err.Error()
		return obs
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
	obs.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
	obs.Detail = fmt.Sprintf("HTTP %d in %dµs", resp.StatusCode, obs.LatencyUs)
	return obs
}

// ------------------------------------------------------------- artifacts

// fetchArtifact returns the path of a verified executable for digest,
// fetching it from mesh peers (or the control plane) if needed.
func (a *Agent) fetchArtifact(b *api.Bundle, image, digest string) (string, error) {
	if !strings.HasPrefix(digest, "b3:") {
		return "", fmt.Errorf("process artifacts are BLAKE3 addressed, got %q", digest)
	}
	dir := filepath.Join(a.cfg.DataDir, "artifacts")
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, strings.TrimPrefix(digest, "b3:"))
	if data, err := os.ReadFile(path); err == nil {
		if storage.ChunkID(data) == digest {
			return path, nil
		}
		_ = os.Remove(path)
	}
	var loc *api.ArtifactLocation
	for i := range b.Artifacts {
		if b.Artifacts[i].Digest == digest {
			loc = &b.Artifacts[i]
		}
	}
	if loc == nil {
		return "", fmt.Errorf("control plane has no location for artifact %s", digest)
	}
	mb, err := a.fetchObject(loc.Manifest, loc.Nodes)
	if err != nil {
		return "", fmt.Errorf("artifact manifest: %w", err)
	}
	var m api.ArtifactManifest
	if err := json.Unmarshal(mb, &m); err != nil || m.Digest != digest {
		return "", errors.New("artifact manifest does not describe the requested digest")
	}
	var buf bytes.Buffer
	start := time.Now()
	sources := map[string]int{}
	var slowest time.Duration
	for _, cid := range m.Chunks {
		t0 := time.Now()
		data, src, err := a.fetchObjectFrom(cid, loc.Nodes)
		if err != nil {
			return "", fmt.Errorf("chunk %s: %w", short(cid), err)
		}
		if d := time.Since(t0); d > slowest {
			slowest = d
		}
		sources[src]++
		buf.Write(data)
	}
	elapsed := time.Since(start).Round(time.Millisecond)
	_ = slowest
	if got := storage.ChunkID(buf.Bytes()); got != digest {
		return "", fmt.Errorf("assembled artifact hashes to %s, not %s", got, digest)
	}
	if err := a.cas.Barrier(); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o500); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	a.record("artifact-verified", "artifact/"+short(digest), 0, fmt.Sprintf("%s: %d bytes in %d chunk(s) fetched in %s (slowest chunk %s) from %v, BLAKE3 verified", image, buf.Len(), len(m.Chunks), elapsed, slowest.Round(time.Millisecond), sources))
	return path, nil
}

// fetchObject gets a CAS object locally, from mesh peers, or from the
// control plane, verifying its hash in every case.
func (a *Agent) fetchObject(id string, holders []string) ([]byte, error) {
	data, _, err := a.fetchObjectFrom(id, holders)
	return data, err
}

const holderBackoff = 30 * time.Second

// holderDown reports a holder that failed a fetch within holderBackoff.
func (a *Agent) holderDown(node string) bool {
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	return time.Since(a.badHolder[node]) < holderBackoff
}

func (a *Agent) markHolderDown(node string) {
	a.meshMu.Lock()
	defer a.meshMu.Unlock()
	a.badHolder[node] = time.Now()
}

// fetchObjectFrom also reports where the bytes came from. Holders that just
// failed are skipped for holderBackoff; the control plane is the last resort.
func (a *Agent) fetchObjectFrom(id string, holders []string) ([]byte, string, error) {
	if data, err := a.cas.Get(id); err == nil {
		return data, "local", nil
	}
	var lastErr error = errors.New("no holder reachable")
	if pc := a.peerClient(8 * time.Second); pc != nil {
		for _, h := range holders {
			ip := a.meshIPOf(h)
			if ip == "" || h == a.st.NodeID || a.holderDown(h) {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			data, err := pc.GetChunk(ctx, ip, id)
			cancel()
			if err != nil {
				// A dead or partitioned holder would otherwise cost the full
				// timeout on every one of an artifact's chunks.
				a.markHolderDown(h)
				lastErr = err
				continue
			}
			if storage.ChunkID(data) != id {
				lastErr = fmt.Errorf("peer %s served bytes that do not hash to %s", short(h), short(id))
				continue
			}
			_ = a.cas.PutWithID(id, data)
			return data, "peer:" + short(h), nil
		}
	}
	t0 := time.Now()
	req, err := a.request("chunk", id)
	if err != nil {
		return nil, "", err
	}
	t1 := time.Now()
	var data []byte
	if err := a.cp.post("/v1/chunk", req, &data); err != nil {
		return nil, "", fmt.Errorf("%v; control plane: %w", lastErr, err)
	}
	if os.Getenv("DH_DEBUG_FETCH") != "" {
		a.log.Printf("fetch %s: sign %s, post %s", short(id), t1.Sub(t0), time.Since(t1))
	}
	if storage.ChunkID(data) != id {
		return nil, "", errors.New("control plane served bytes with the wrong hash")
	}
	_ = a.cas.PutWithID(id, data)
	return data, "control-plane", nil
}

// ------------------------------------------------------------- forwarding

// forwarder exposes a workload's loopback port on the mesh address, so edge
// hosts reach it only through the WireGuard tunnel.
type forwarder struct {
	ln    net.Listener
	local atomic.Int64 // current loopback target; 0 = no running instance
	mesh  int64
	once  sync.Once
}

func (f *forwarder) close() { f.once.Do(func() { f.ln.Close() }) }

// ensureForward exposes a workload on its mesh port. The listener lives as
// long as the assignment: a restart or rolling replacement only retargets it
// (re-listening on the same userspace-netstack port right after closing it
// is not reliable).
func (a *Agent) ensureForward(id string, meshPort, localPort int64) {
	a.meshMu.Lock()
	defer a.meshMu.Unlock()
	if a.cutover[id] {
		return // a handover owns this forwarder until the new instance is ready
	}
	a.ensureForwardLocked(id, meshPort, localPort)
}

// switchForward ends a handover: traffic moves to the new instance.
func (a *Agent) switchForward(id string, meshPort, localPort int64) {
	a.meshMu.Lock()
	defer a.meshMu.Unlock()
	delete(a.cutover, id)
	a.ensureForwardLocked(id, meshPort, localPort)
}

func (a *Agent) ensureForwardLocked(id string, meshPort, localPort int64) {
	if f := a.forwards[id]; f != nil && f.mesh == meshPort {
		f.local.Store(localPort)
		return
	} else if f != nil {
		f.close()
		delete(a.forwards, id)
	}
	if a.dev == nil {
		return
	}
	ln, err := a.dev.ListenTCP(int(meshPort))
	if err != nil {
		return
	}
	f := &forwarder{ln: ln, mesh: meshPort}
	f.local.Store(localPort)
	a.forwards[id] = f
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			if !a.allowServiceClient(c.RemoteAddr()) {
				a.meshMu.Lock()
				a.refused = fmt.Sprintf("%s refused service connection from %s at %s", id, c.RemoteAddr(), time.Now().Format(time.RFC3339))
				a.meshMu.Unlock()
				c.Close()
				continue
			}
			target := f.local.Load()
			if target == 0 {
				c.Close() // no running instance: the edge sees a failure and retries elsewhere
				continue
			}
			go pipe(c, fmt.Sprintf("127.0.0.1:%d", target))
		}
	}()
}

// pauseForward keeps the mesh listener but stops sending traffic to a
// stopped or exited instance.
func (a *Agent) pauseForward(id string) {
	a.meshMu.Lock()
	defer a.meshMu.Unlock()
	if f := a.forwards[id]; f != nil {
		f.local.Store(0)
	}
}

// closeForward releases the mesh port when the assignment is forgotten.
func (a *Agent) closeForward(id string) {
	a.meshMu.Lock()
	defer a.meshMu.Unlock()
	if f := a.forwards[id]; f != nil {
		f.close()
		delete(a.forwards, id)
	}
}

// allowServiceClient: only edge hosts and control-plane members may reach
// workload ports over the mesh.
func (a *Agent) allowServiceClient(addr net.Addr) bool {
	host, _, _ := net.SplitHostPort(addr.String())
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	// An edge host probing and proxying to a replica it runs itself dials
	// its own mesh address; that client is this host, not a peer.
	if a.dev != nil && host == a.dev.IP() {
		return contains(a.cfg.Roles, "edge")
	}
	id := a.peerIPs[host]
	if id == "" {
		return false
	}
	p := a.peerInfo[id]
	return contains(p.Roles, "edge") || contains(p.Roles, "control-plane")
}

func pipe(c net.Conn, target string) {
	defer c.Close()
	u, err := net.DialTimeout("tcp", target, 3*time.Second)
	if err != nil {
		return
	}
	defer u.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(u, c); done <- struct{}{} }()
	go func() { _, _ = io.Copy(c, u); done <- struct{}{} }()
	<-done
}

func (a *Agent) meshIPOf(node string) string {
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	if p, ok := a.peerInfo[node]; ok && a.bindOK[node] {
		return p.MeshIP
	}
	return ""
}

func (a *Agent) peerClient(timeout time.Duration) *peer.Client {
	a.meshMu.RLock()
	defer a.meshMu.RUnlock()
	if a.dev == nil {
		return nil
	}
	return &peer.Client{HTTP: a.dev.HTTPClient(timeout)}
}
