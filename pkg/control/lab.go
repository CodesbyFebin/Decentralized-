package control

import (
	"net/http"
	"sync"
	"time"

	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/identity"
)

// The capability lab lets an operator experiment with attenuation in the
// console. It uses a separate, ephemeral LAB root that no host pins, so a
// lab token can never admit a real workload. Every response says so.
type lab struct {
	once sync.Once
	root *identity.Identity
	hold *identity.Identity
}

var theLab lab

func (l *lab) init() {
	l.once.Do(func() {
		l.root, _ = identity.Generate()
		l.hold, _ = identity.Generate()
	})
}

const labBanner = "LAB CAPABILITY — signed by an ephemeral lab root that no host trusts; it cannot admit real work"

func (s *Server) labRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/lab/mint", s.op("api.read", func(w http.ResponseWriter, r *http.Request, _ *authz) {
		theLab.init()
		var c capability.Caveats
		if err := readBody(r, 16<<10, &c); err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		if c.Expires == 0 {
			c.Expires = nowMs() + int64(time.Hour/time.Millisecond)
		}
		tok, err := capability.Mint(theLab.root, c, theLab.hold.PubString(), "lab")
		if err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		writeJSON(w, 200, map[string]any{"token": tok.Encode(), "labRoot": theLab.root.PubString(), "banner": labBanner, "blocks": len(tok.Blocks)})
	}))
	mux.HandleFunc("POST /api/v1/lab/attenuate", s.op("api.read", func(w http.ResponseWriter, r *http.Request, _ *authz) {
		theLab.init()
		var body struct {
			Token   string             `json:"token"`
			Caveats capability.Caveats `json:"caveats"`
		}
		if err := readBody(r, 64<<10, &body); err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		tok, err := capability.Decode(body.Token)
		if err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		next, err := tok.Attenuate(theLab.hold, body.Caveats, theLab.hold.PubString(), "lab attenuation")
		if err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		writeJSON(w, 200, map[string]any{"token": next.Encode(), "banner": labBanner, "blocks": len(next.Blocks)})
	}))
	mux.HandleFunc("POST /api/v1/lab/verify", s.op("api.read", func(w http.ResponseWriter, r *http.Request, _ *authz) {
		theLab.init()
		var body struct {
			Token   string             `json:"token"`
			Request capability.Request `json:"request"`
		}
		if err := readBody(r, 64<<10, &body); err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		tok, err := capability.Decode(body.Token)
		if err != nil {
			writeErr(w, 400, "%v", err)
			return
		}
		if body.Request.Now == 0 {
			body.Request.Now = nowMs()
		}
		lab := capability.Verify(tok, []string{theLab.root.PubString()}, body.Request)
		var root string
		s.fsm.Read(func(st *State) { root = st.Root })
		real := capability.Verify(tok, []string{root}, body.Request)
		writeJSON(w, 200, map[string]any{"lab": lab, "againstClusterRoot": real, "banner": labBanner})
	}))
}
