// Package edge is the dh/v1 L7 edge: TLS termination, health-gated routing
// to workload replicas across the WireGuard mesh, graceful draining, and
// certificate lifecycle (cluster-local CA or ACME).
//
// Routing truth has two sources and both must agree: the control plane lists
// an endpoint only after the owning host signed an observation that it is
// running, and the edge itself probes the endpoint over the tunnel. An
// endpoint that fails probes is ejected; one the control plane marks as
// draining receives no new requests while in-flight requests complete.
package edge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"decentralized.host/pkg/api"
)

// Dialer reaches mesh addresses (nil until the mesh is up).
type Dialer func(ctx context.Context, network, addr string) (net.Conn, error)

// Config configures an edge.
type Config struct {
	Node        string
	DataDir     string
	HTTPListen  string
	HTTPSListen string
	Dial        func() Dialer
	Certs       *CertManager
	Logger      *log.Logger
	ProbeEvery  time.Duration
}

type endpoint struct {
	ep          api.Endpoint
	state       string // pending | routing | draining | ejected
	reason      string
	since       int64
	inFlight    atomic.Int64
	served      atomic.Int64
	failures    atomic.Int64
	latencyUs   atomic.Int64
	lastOK      atomic.Int64 // unix ms of the last successful probe or response
	consecutive int
}

type route struct {
	svc api.Service
	eps map[string]*endpoint // key assignment@node
	rr  atomic.Uint64
}

// Edge is a running edge proxy.
type Edge struct {
	cfg      Config
	mu       sync.RWMutex
	routes   map[string]*route // by host
	requests atomic.Int64
	errors   atomic.Int64
	http     *http.Server
	https    *http.Server
	probeNow chan struct{}
	fast     *http.Transport
	slow     *http.Transport
	stop     chan struct{}
}

// New creates an edge.
func New(cfg Config) *Edge {
	if cfg.ProbeEvery == 0 {
		cfg.ProbeEvery = time.Second
	}
	e := &Edge{cfg: cfg, routes: map[string]*route{}, stop: make(chan struct{}), probeNow: make(chan struct{}, 1)}
	// Shared upstream transports: connections through the tunnel are pooled
	// and reused, and idle ones are closed.
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := cfg.Dial()
		if d == nil {
			return nil, errors.New("edge mesh is not up")
		}
		// A dead or partitioned peer black-holes packets; a live peer
		// answers a TCP handshake in milliseconds. Fail fast and retry.
		ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		defer cancel()
		return d(ctx, network, addr)
	}
	e.fast = &http.Transport{DialContext: dial, ResponseHeaderTimeout: 2 * time.Second, MaxIdleConnsPerHost: 16, IdleConnTimeout: 30 * time.Second}
	e.slow = &http.Transport{DialContext: dial, ResponseHeaderTimeout: 30 * time.Second, MaxIdleConnsPerHost: 16, IdleConnTimeout: 30 * time.Second}
	return e
}

// Start opens the listeners and the probe loop.
func (e *Edge) Start() error {
	if e.cfg.HTTPListen != "" {
		ln, err := net.Listen("tcp", e.cfg.HTTPListen)
		if err != nil {
			return err
		}
		e.http = &http.Server{Handler: http.HandlerFunc(e.serveHTTP), ReadHeaderTimeout: 10 * time.Second}
		go e.http.Serve(ln)
	}
	if e.cfg.HTTPSListen != "" {
		ln, err := net.Listen("tcp", e.cfg.HTTPSListen)
		if err != nil {
			return err
		}
		conf := &tls.Config{MinVersion: tls.VersionTLS12, NextProtos: []string{"h2", "http/1.1"}, GetCertificate: e.cfg.Certs.GetCertificate}
		e.https = &http.Server{Handler: http.HandlerFunc(e.serveProxy), ReadHeaderTimeout: 10 * time.Second, TLSConfig: conf}
		go e.https.Serve(tls.NewListener(ln, conf))
	}
	go e.probeLoop()
	return nil
}

// Close stops the edge.
func (e *Edge) Close() {
	close(e.stop)
	if e.http != nil {
		_ = e.http.Close()
	}
	if e.https != nil {
		_ = e.https.Close()
	}
}

func epKey(ep api.Endpoint) string { return ep.Assignment + "@" + ep.Node }

// Update applies the control plane's routing table.
func (e *Edge) Update(services []api.Service) {
	now := time.Now().UnixMilli()
	e.mu.Lock()
	defer e.mu.Unlock()
	kick := false
	defer func() {
		if kick {
			select {
			case e.probeNow <- struct{}{}:
			default:
			}
		}
	}()
	seen := map[string]bool{}
	for _, svc := range services {
		seen[svc.Host] = true
		r := e.routes[svc.Host]
		if r == nil {
			r = &route{eps: map[string]*endpoint{}}
			e.routes[svc.Host] = r
		}
		r.svc = svc
		want := map[string]bool{}
		for _, ep := range svc.Endpoints {
			k := epKey(ep)
			want[k] = true
			cur := r.eps[k]
			if cur == nil {
				cur = &endpoint{ep: ep, state: "pending", reason: "listed by control plane; awaiting first edge probe", since: now}
				r.eps[k] = cur
				kick = true
			}
			cur.ep = ep
			if ep.Draining && cur.state != "draining" {
				cur.state, cur.reason, cur.since = "draining", "control plane marked the replica draining; no new requests", now
			}
			if !ep.Draining && cur.state == "draining" {
				cur.state, cur.reason, cur.since = "pending", "no longer draining; awaiting probe", now
				kick = true
			}
		}
		for k, ep := range r.eps {
			if !want[k] {
				if ep.inFlight.Load() == 0 {
					delete(r.eps, k)
				} else if ep.state != "draining" {
					ep.state, ep.reason, ep.since = "draining", "removed from routing table; finishing in-flight requests", now
				}
			}
		}
	}
	for h := range e.routes {
		if !seen[h] {
			delete(e.routes, h)
		}
	}
	if e.cfg.Certs != nil {
		e.cfg.Certs.Want(services)
	}
}

func (e *Edge) probeLoop() {
	t := time.NewTicker(e.cfg.ProbeEvery)
	defer t.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-t.C:
		case <-e.probeNow:
		}
		e.probeAll()
	}
}

func (e *Edge) probeAll() {
	dial := e.cfg.Dial()
	if dial == nil {
		return
	}
	type job struct {
		r  *route
		ep *endpoint
	}
	var jobs []job
	e.mu.RLock()
	for _, r := range e.routes {
		for _, ep := range r.eps {
			if ep.state != "draining" {
				jobs = append(jobs, job{r, ep})
			}
		}
	}
	e.mu.RUnlock()
	client := &http.Client{Timeout: 900 * time.Millisecond, Transport: &http.Transport{DialContext: dial, DisableKeepAlives: true}}
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			path := j.r.svc.HealthURL
			if path == "" {
				path = "/"
			}
			start := time.Now()
			resp, err := client.Get(fmt.Sprintf("http://%s:%d%s", j.ep.ep.MeshIP, j.ep.ep.Port, path))
			ok := err == nil && resp.StatusCode < 500
			detail := ""
			if err != nil {
				detail = err.Error()
			} else {
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				resp.Body.Close()
				detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
			now := time.Now().UnixMilli()
			if ok {
				j.ep.lastOK.Store(now)
			}
			e.mu.Lock()
			defer e.mu.Unlock()
			if j.ep.state == "draining" {
				return
			}
			if ok {
				j.ep.latencyUs.Store(time.Since(start).Microseconds())
				j.ep.consecutive = 0
				if j.ep.state != "routing" {
					j.ep.state, j.ep.reason, j.ep.since = "routing", "edge probe over WireGuard passed ("+detail+")", now
				}
				return
			}
			j.ep.consecutive++
			if j.ep.consecutive >= 2 && j.ep.state != "ejected" {
				j.ep.state, j.ep.reason, j.ep.since = "ejected", fmt.Sprintf("%d consecutive edge probes failed: %s", j.ep.consecutive, detail), now
			}
		}(j)
	}
	wg.Wait()
}

// serveHTTP answers ACME HTTP-01 challenges and otherwise proxies (or
// redirects to HTTPS when a certificate is issued for the host).
func (e *Edge) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") && e.cfg.Certs != nil {
		if body, ok := e.cfg.Certs.ChallengeResponse(strings.TrimPrefix(r.URL.Path, "/.well-known/acme-challenge/")); ok {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, body)
			return
		}
		http.NotFound(w, r)
		return
	}
	host := hostOnly(r.Host)
	if e.https != nil && e.cfg.Certs != nil && e.cfg.Certs.Issued(host) && r.Header.Get("X-DH-No-Redirect") == "" {
		_, port, _ := net.SplitHostPort(e.cfg.HTTPSListen)
		target := url.URL{Scheme: "https", Host: net.JoinHostPort(host, port), Path: r.URL.Path, RawQuery: r.URL.RawQuery}
		http.Redirect(w, r, target.String(), http.StatusPermanentRedirect)
		return
	}
	e.serveProxy(w, r)
}

func hostOnly(h string) string {
	if host, _, err := net.SplitHostPort(h); err == nil {
		return strings.ToLower(host)
	}
	return strings.ToLower(h)
}

func (e *Edge) lookup(host string) *route {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if r := e.routes[host]; r != nil {
		return r
	}
	// wildcard ingress: *.apps.example.test
	if i := strings.IndexByte(host, '.'); i > 0 {
		if r := e.routes["*"+host[i:]]; r != nil {
			return r
		}
	}
	return nil
}

// drainingFallbackMs bounds how recently a draining endpoint must have been
// seen alive to be used as a last resort.
const drainingFallbackMs = 3000

func (e *Edge) pick(r *route, exclude string) *endpoint {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var ready []*endpoint
	for k, ep := range r.eps {
		if ep.state == "routing" && k != exclude {
			ready = append(ready, ep)
		}
	}
	if len(ready) == 0 {
		// Last resort: endpoints the control plane lists as running that
		// the edge has not probed yet (for example a replica that just
		// finished a rolling update). A failure there is retried; it is
		// better than refusing while a probe is due.
		for k, ep := range r.eps {
			if ep.state == "pending" && k != exclude {
				ready = append(ready, ep)
			}
		}
	}
	if len(ready) == 0 {
		// Still nothing: a draining endpoint that the edge saw alive in the
		// last few seconds (for example the old generation during a rolling
		// update) beats refusing the request. A draining endpoint without
		// recent proof of life may be a dead host: sending to it would hang
		// the request instead of failing it fast (REF-MAC-A02 / M2).
		now := time.Now().UnixMilli()
		for k, ep := range r.eps {
			if ep.state == "draining" && k != exclude && now-ep.lastOK.Load() < drainingFallbackMs {
				ready = append(ready, ep)
			}
		}
	}
	if len(ready) == 0 {
		return nil
	}
	sort.Slice(ready, func(i, j int) bool { return epKey(ready[i].ep) < epKey(ready[j].ep) })
	return ready[int(r.rr.Add(1)-1)%len(ready)]
}

func (e *Edge) serveProxy(w http.ResponseWriter, r *http.Request) {
	e.requests.Add(1)
	w.Header().Set("X-DH-Edge", e.cfg.Node)
	route := e.lookup(hostOnly(r.Host))
	if route == nil {
		e.errors.Add(1)
		http.Error(w, "no ingress route for "+hostOnly(r.Host), http.StatusNotFound)
		return
	}
	dial := e.cfg.Dial()
	if dial == nil {
		e.errors.Add(1)
		http.Error(w, "edge mesh is not up", http.StatusServiceUnavailable)
		return
	}
	idempotent := r.Method == http.MethodGet || r.Method == http.MethodHead
	exclude := ""
	for attempt := 0; attempt < 2; attempt++ {
		ep := e.pick(route, exclude)
		if ep == nil {
			e.errors.Add(1)
			w.Header().Set("Retry-After", "1")
			http.Error(w, "no healthy endpoint for "+route.svc.Host+" (all replicas ejected, draining or unobserved)", http.StatusServiceUnavailable)
			return
		}
		ep.inFlight.Add(1)
		start := time.Now()
		failed := false
		target := &url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", ep.ep.MeshIP, ep.ep.Port)}
		proxy := httputil.NewSingleHostReverseProxy(target)
		// A replica that accepts a request but never answers (hung process)
		// is only detectable by time. Idempotent first attempts get a short
		// first-byte budget and are retried elsewhere; others wait longer.
		proxy.Transport = e.slow
		if idempotent && attempt == 0 {
			proxy.Transport = e.fast
		}
		rec := &retryWriter{ResponseWriter: w}
		proxy.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, err error) {
			failed = true
			rec.err = err
		}
		proxy.ModifyResponse = func(resp *http.Response) error {
			resp.Header.Set("X-DH-Upstream", ep.ep.Assignment+"@"+ep.ep.Node)
			return nil
		}
		if idempotent && attempt == 0 {
			rec.buffer = true
		}
		proxy.ServeHTTP(rec, r)
		ep.inFlight.Add(-1)
		if !failed {
			ep.lastOK.Store(time.Now().UnixMilli())
			ep.served.Add(1)
			ep.latencyUs.Store(time.Since(start).Microseconds())
			rec.flush()
			return
		}
		ep.failures.Add(1)
		e.mu.Lock()
		ep.consecutive++
		if ep.state == "routing" {
			ep.state, ep.reason, ep.since = "ejected", "request failed: "+rec.err.Error(), time.Now().UnixMilli()
			// Pooled keep-alive connections may point at the dead replica.
			e.fast.CloseIdleConnections()
			e.slow.CloseIdleConnections()
		}
		e.mu.Unlock()
		if !idempotent {
			e.errors.Add(1)
			http.Error(w, "upstream failed: "+rec.err.Error(), http.StatusBadGateway)
			return
		}
		exclude = epKey(ep.ep)
	}
	e.errors.Add(1)
	http.Error(w, "all attempted upstreams failed", http.StatusBadGateway)
}

// retryWriter lets the first attempt of an idempotent request fail without
// having written anything, so the edge can retry another replica.
type retryWriter struct {
	http.ResponseWriter
	buffer  bool
	code    int
	headers bool
	err     error
	body    []byte
}

func (r *retryWriter) WriteHeader(code int) {
	if r.buffer {
		r.code = code
		r.headers = true
		return
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *retryWriter) Write(b []byte) (int, error) {
	if r.buffer {
		if !r.headers {
			r.code, r.headers = 200, true
		}
		r.body = append(r.body, b...)
		if len(r.body) > 1<<20 {
			r.flush()
		}
		return len(b), nil
	}
	return r.ResponseWriter.Write(b)
}

func (r *retryWriter) flush() {
	if !r.buffer {
		return
	}
	r.buffer = false
	if r.headers {
		r.ResponseWriter.WriteHeader(r.code)
	}
	if len(r.body) > 0 {
		_, _ = r.ResponseWriter.Write(r.body)
	}
	r.body = nil
}

func (r *retryWriter) Flush() {
	r.flush()
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Observe returns the edge's measured state.
func (e *Edge) Observe() api.EdgeObs {
	o := api.EdgeObs{HTTPAddr: e.cfg.HTTPListen, HTTPSAddr: e.cfg.HTTPSListen, Requests: e.requests.Load(), Errors: e.errors.Load(), Routes: []api.RouteObs{}, Certs: []api.CertObs{}}
	e.mu.RLock()
	hosts := make([]string, 0, len(e.routes))
	for h := range e.routes {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	for _, h := range hosts {
		r := e.routes[h]
		ro := api.RouteObs{Host: h, App: r.svc.App, Endpoints: []api.EndpointObs{}}
		keys := make([]string, 0, len(r.eps))
		for k := range r.eps {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ep := r.eps[k]
			ro.Endpoints = append(ro.Endpoints, api.EndpointObs{Assignment: ep.ep.Assignment, Node: ep.ep.Node, State: ep.state, Reason: ep.reason,
				InFlight: ep.inFlight.Load(), Served: ep.served.Load(), Failures: ep.failures.Load(), LatencyUs: ep.latencyUs.Load(), Since: ep.since})
		}
		o.Routes = append(o.Routes, ro)
	}
	e.mu.RUnlock()
	if e.cfg.Certs != nil {
		o.Certs = e.cfg.Certs.Observe()
	}
	return o
}

// ErrNoRoute is returned when a host has no ingress.
var ErrNoRoute = errors.New("edge: no route")
