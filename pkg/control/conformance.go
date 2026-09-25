package control

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"decentralized.host/conformance/vectors"
	"decentralized.host/pkg/api"
	"decentralized.host/pkg/conformance"
)

// The conformance endpoint runs the embedded dh/v1 vectors against this
// member's own protocol code, on request. The result is OBSERVED: it is a run
// that just happened on the serving member, not a stored claim.
// selfConformance runs the vectors once per process: a member's protocol code
// cannot change while it runs.
var selfConformance = sync.OnceValues(func() (*conformance.Report, error) {
	var suite conformance.Suite
	if err := json.Unmarshal(vectors.DHv1, &suite); err != nil {
		return nil, err
	}
	return conformance.Run(&suite, conformance.InProcess{}, "go-reference", conformance.Options{})
})

func conformanceMilestone() MilestoneView {
	m := MilestoneView{ID: "M8", Title: "Protocol conformance", Basis: api.Observed}
	rep, err := selfConformance()
	switch {
	case err != nil:
		m.State, m.Basis = "NOT OBSERVED", api.Unknown
		m.Gaps = []string{"embedded vectors could not run: " + err.Error()}
	case rep.OK(false):
		m.State = "VERIFIED"
		m.Evidence = []string{fmt.Sprintf("this member passes %d/%d dh/v1 vectors", rep.Pass, rep.Pass+rep.Fail)}
	default:
		m.State = "PARTIAL"
		m.Gaps = []string{fmt.Sprintf("this member fails %d of %d dh/v1 vectors", rep.Fail, rep.Pass+rep.Fail)}
	}
	m.Evidence = append(m.Evidence, "other implementations: dh-conformance run -adapter \"<command>\"")
	return m
}

func init() {
	extraRoutes = append(extraRoutes, func(s *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /api/v1/conformance", s.op("api.read", func(w http.ResponseWriter, _ *http.Request, _ *authz) {
			var suite conformance.Suite
			if err := json.Unmarshal(vectors.DHv1, &suite); err != nil {
				writeErr(w, 500, "embedded vectors: %v", err)
				return
			}
			rep, err := conformance.Run(&suite, conformance.InProcess{}, "go-reference on "+short(s.id.ID), conformance.Options{})
			if err != nil {
				writeErr(w, 500, "%v", err)
				return
			}
			writeJSON(w, 200, map[string]any{
				"basis":    api.Observed,
				"member":   s.id.ID,
				"go":       runtime.Version(),
				"vectors":  len(suite.Vectors),
				"ops":      conformance.Ops,
				"report":   rep,
				"external": "independent implementations are tested with: dh-conformance run -adapter \"<command>\" (see docs/protocol/conformance.md)",
			})
		}))
	})
}
