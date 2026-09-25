package edge

import (
	"testing"
	"time"

	"decentralized.host/pkg/api"
)

func routeWith(eps ...*endpoint) *route {
	r := &route{eps: map[string]*endpoint{}}
	for _, ep := range eps {
		r.eps[epKey(ep.ep)] = ep
	}
	return r
}

func ep(assignment, node, state string) *endpoint {
	return &endpoint{ep: api.Endpoint{Assignment: assignment, Node: node}, state: state}
}

// Regression (REF-MAC-A02, M2 "interrupted write"): with no routing
// endpoint, the edge fell back to a draining endpoint on a dead host and a
// request hung for 30s instead of failing fast. A draining endpoint is a
// last resort only with recent proof of life.
func TestPickDrainingOnlyWithRecentProofOfLife(t *testing.T) {
	e := &Edge{}
	dead := ep("kv/r0", "host-a", "draining") // never seen alive
	if got := e.pick(routeWith(dead), ""); got != nil {
		t.Fatalf("picked a draining endpoint with no proof of life: %+v", got.ep)
	}
	stale := ep("kv/r0", "host-a", "draining")
	stale.lastOK.Store(time.Now().Add(-10 * time.Second).UnixMilli())
	if got := e.pick(routeWith(stale), ""); got != nil {
		t.Fatal("picked a draining endpoint last seen alive 10s ago")
	}
	alive := ep("kv/r0", "host-a", "draining")
	alive.lastOK.Store(time.Now().UnixMilli())
	if got := e.pick(routeWith(alive), ""); got != alive {
		t.Fatal("a draining endpoint seen alive just now should serve rather than refuse")
	}
	routing := ep("kv/r0", "host-b", "routing")
	if got := e.pick(routeWith(alive, routing), ""); got != routing {
		t.Fatal("a routing endpoint always wins over a draining one")
	}
	pending := ep("kv/r0", "host-c", "pending")
	if got := e.pick(routeWith(alive, pending), ""); got != pending {
		t.Fatal("a pending endpoint is preferred over a draining one")
	}
}
