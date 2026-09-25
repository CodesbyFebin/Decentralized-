// Package chaos turns resilience claims into executable evidence. Every
// scenario runs a real multi-process cluster, injects a real fault, measures
// an invariant, and emits a signed report. A scenario whose invariant was
// not measured cannot PASS.
package chaos

import (
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Faults configures packet-level faults on the WireGuard path.
type Faults struct {
	Isolated   map[string]bool // node name -> drop everything to/from it
	DropPct    int             // random loss, percent
	DupPct     int             // duplicate packets, percent
	ReorderPct int             // hold a packet back behind the next one, percent
	Delay      time.Duration   // added one-way delay
}

// Controller owns the fault state shared by all proxies.
type Controller struct {
	mu     sync.RWMutex
	f      Faults
	stats  struct{ forwarded, dropped, duplicated, reordered atomic.Int64 }
	byAddr map[string]string // "127.0.0.1:port" -> node name (host WireGuard sockets)
}

// NewController returns an empty controller.
func NewController() *Controller {
	return &Controller{f: Faults{Isolated: map[string]bool{}}, byAddr: map[string]string{}}
}

// Set replaces the fault configuration.
func (c *Controller) Set(f Faults) {
	if f.Isolated == nil {
		f.Isolated = map[string]bool{}
	}
	c.mu.Lock()
	c.f = f
	c.mu.Unlock()
}

// Get returns the current faults.
func (c *Controller) Get() Faults {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.f
}

// Stats returns packet counters.
func (c *Controller) Stats() map[string]int64 {
	return map[string]int64{"forwarded": c.stats.forwarded.Load(), "dropped": c.stats.dropped.Load(),
		"duplicated": c.stats.duplicated.Load(), "reordered": c.stats.reordered.Load()}
}

// Proxy relays UDP for one host's WireGuard socket. Peers dial the proxy
// (it is the host's advertised endpoint); every flow involving the host —
// and, through WireGuard endpoint roaming, flows it initiates — crosses a
// proxy, so faults are applied to real encrypted traffic.
type Proxy struct {
	ctl      *Controller
	node     string
	listen   *net.UDPConn
	upstream *net.UDPAddr
	mu       sync.Mutex
	flows    map[string]*flow
	closed   atomic.Bool
}

type flow struct {
	client *net.UDPAddr
	conn   *net.UDPConn
	last   time.Time
}

// NewProxy listens on listen and forwards to the host's WireGuard socket.
func (c *Controller) NewProxy(node, listen, upstream string) (*Proxy, error) {
	la, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return nil, err
	}
	ua, err := net.ResolveUDPAddr("udp", upstream)
	if err != nil {
		return nil, err
	}
	l, err := net.ListenUDP("udp", la)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.byAddr[ua.String()] = node
	c.mu.Unlock()
	p := &Proxy{ctl: c, node: node, listen: l, upstream: ua, flows: map[string]*flow{}}
	go p.inbound()
	return p, nil
}

// Close stops the proxy.
func (p *Proxy) Close() {
	p.closed.Store(true)
	p.listen.Close()
	p.mu.Lock()
	for _, f := range p.flows {
		f.conn.Close()
	}
	p.mu.Unlock()
}

// decide applies faults to one packet from -> to (node names, "" = unknown).
func (p *Proxy) decide(from string) (drop, dup, hold bool, delay time.Duration) {
	f := p.ctl.Get()
	if f.Isolated[p.node] || (from != "" && f.Isolated[from]) {
		return true, false, false, 0
	}
	if f.DropPct > 0 && rand.Intn(100) < f.DropPct {
		return true, false, false, 0
	}
	dup = f.DupPct > 0 && rand.Intn(100) < f.DupPct
	hold = f.ReorderPct > 0 && rand.Intn(100) < f.ReorderPct
	return false, dup, hold, f.Delay
}

func (p *Proxy) send(conn *net.UDPConn, to *net.UDPAddr, b []byte, dup, hold bool, delay time.Duration) {
	out := func() {
		if to == nil {
			_, _ = conn.Write(b)
		} else {
			_, _ = conn.WriteToUDP(b, to)
		}
		p.ctl.stats.forwarded.Add(1)
	}
	if hold {
		// Reorder: release this packet after the next ones (a short hold).
		p.ctl.stats.reordered.Add(1)
		delay += 3 * time.Millisecond
	}
	if delay > 0 {
		time.AfterFunc(delay, out)
	} else {
		out()
	}
	if dup {
		p.ctl.stats.duplicated.Add(1)
		cp := append([]byte(nil), b...)
		time.AfterFunc(delay+time.Millisecond, func() {
			if to == nil {
				_, _ = conn.Write(cp)
			} else {
				_, _ = conn.WriteToUDP(cp, to)
			}
		})
	}
}

func (p *Proxy) inbound() {
	buf := make([]byte, 65536)
	for {
		n, client, err := p.listen.ReadFromUDP(buf)
		if err != nil {
			if p.closed.Load() {
				return
			}
			continue
		}
		pkt := append([]byte(nil), buf[:n]...)
		p.ctl.mu.RLock()
		from := p.ctl.byAddr[client.String()]
		p.ctl.mu.RUnlock()
		drop, dup, hold, delay := p.decide(from)
		if drop {
			p.ctl.stats.dropped.Add(1)
			continue
		}
		fl := p.flowFor(client)
		if fl == nil {
			continue
		}
		p.send(fl.conn, nil, pkt, dup, hold, delay)
	}
}

func (p *Proxy) flowFor(client *net.UDPAddr) *flow {
	key := client.String()
	p.mu.Lock()
	defer p.mu.Unlock()
	if f := p.flows[key]; f != nil {
		f.last = time.Now()
		return f
	}
	conn, err := net.DialUDP("udp", nil, p.upstream)
	if err != nil {
		return nil
	}
	f := &flow{client: client, conn: conn, last: time.Now()}
	p.flows[key] = f
	p.ctl.mu.RLock()
	from := p.ctl.byAddr[client.String()]
	p.ctl.mu.RUnlock()
	go p.outbound(f, from)
	return f
}

// outbound relays the host's replies back to the client.
func (p *Proxy) outbound(f *flow, clientNode string) {
	buf := make([]byte, 65536)
	for {
		n, err := f.conn.Read(buf)
		if err != nil {
			return
		}
		pkt := append([]byte(nil), buf[:n]...)
		drop, dup, hold, delay := p.decide(clientNode)
		if drop {
			p.ctl.stats.dropped.Add(1)
			continue
		}
		p.send(p.listen, f.client, pkt, dup, hold, delay)
	}
}
