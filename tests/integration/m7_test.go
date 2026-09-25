package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/envelope"
)

const fedPolicyAllow = `allowFederated: true
acceptTiers: [trusted, federated]
`

const fedManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: burst}
spec:
  replicas: %d
  image: %s
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [federated], spread: none, antiAffinity: soft, federation: [beta]}
  ports: [{name: http}]
  health: {http: /healthz, interval: 1s}
`

func dhCLI(t *testing.T, home string, args ...string) string {
	t.Helper()
	out, err := exec.Command(filepath.Join(binDir, "dh"), append([]string{"--home", home}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("dh %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// M7 exit: two independent meshes, signed peering, cross-mesh placement,
// local policy enforcement, and revocation by the grantor.
func TestM7Federation(t *testing.T) {
	alpha := up(t, devcluster.Options{Cluster: "alpha", CPs: 1, Hosts: 1})
	// beta (the grantor) runs TLS: alpha reaches it with the CA in the agreement.
	beta := up(t, devcluster.Options{Cluster: "beta", CPs: 1, Hosts: 2, TLS: true,
		HostArgs: func(i int, name string) []string {
			if name == "host-a" {
				return []string{"--tiers", "trusted,federated"}
			}
			return nil
		},
		Policy: func(name string) string {
			if name == "host-a" {
				return fedPolicyAllow
			}
			return "" // host-b keeps the default: allowFederated false
		}})

	// alpha shares its root; beta grants capacity; alpha accepts.
	root := strings.Fields(dhCLI(t, alpha.Home, "federation", "root"))
	if len(root) != 2 || root[0] != "alpha" {
		t.Fatalf("root: %v", root)
	}
	agreementFile := filepath.Join(t.TempDir(), "agreement.json")
	dhCLI(t, beta.Home, "federation", "grant", "--to", "alpha", "--to-root", root[1], "--tiers", "federated",
		"--max-replicas", "2", "--max-cpu", "300m", "--max-mem", "256Mi", "--ttl", "1h", "--out", agreementFile)
	dhCLI(t, alpha.Home, "federation", "accept", agreementFile)
	var env envelope.Envelope
	raw, _ := os.ReadFile(agreementFile)
	json.Unmarshal(raw, &env)
	digest := env.Digest()

	t.Run("placement over quota is refused by the grantor", func(t *testing.T) {
		image, _, err := alpha.Op.PushArtifact(filepath.Join(binDir, "dh-beacon"), "dh-beacon", true)
		if err != nil {
			t.Fatal(err)
		}
		r, err := alpha.Op.Apply([]byte(strings.Replace(fedManifestN(5), "%s", image, 1)))
		if err != nil || !r.OK {
			t.Fatalf("apply: %v %+v", err, r)
		}
		if err := alpha.WaitFor(30*time.Second, "grantor refusal recorded", func(v *control.View) bool {
			for _, o := range v.Federation.Outbound {
				if o.App == "burst" && !o.Accepted && strings.Contains(o.Message, "at most 2 replica") {
					return true
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
		dhCLI(t, alpha.Home, "federation", "withdraw", "burst")
	})

	t.Run("cross-cluster placement runs under the grantor's own authority and host policy", func(t *testing.T) {
		image, _, _ := alpha.Op.PushArtifact(filepath.Join(binDir, "dh-beacon"), "dh-beacon", true)
		r, err := alpha.Op.Apply([]byte(strings.Replace(fedManifestN(2), "%s", image, 1)))
		if err != nil || !r.OK {
			t.Fatalf("apply: %v %+v", err, r)
		}
		// On beta: the app runs only on the host whose own policy accepts federated work.
		if err := beta.WaitFor(60*time.Second, "federated app running on beta", func(v *control.View) bool {
			a := app(v, "fed-alpha-burst")
			return a != nil && a.Observed >= 1
		}); err != nil {
			t.Fatal(err)
		}
		v, _ := beta.View()
		a := app(v, "fed-alpha-burst")
		if a.Federation == nil || a.Federation.Peer != "alpha" {
			t.Fatalf("placement not marked as federated: %+v", a.Federation)
		}
		for _, r := range a.Rows {
			if r.Observed == "RUNNING" && r.NodeName != "host-a" {
				t.Fatalf("federated replica running on %s, whose policy refuses federated work", r.NodeName)
			}
			for _, c := range r.Checks {
				if c.Name == "capability" && !strings.Contains(c.Detail, "anchored in the pinned root") {
					t.Fatalf("beta host admitted federated work without its own root chain: %s", c.Detail)
				}
			}
		}
		// On alpha: the remote status is signed by beta and verified against beta's root.
		if err := alpha.WaitFor(30*time.Second, "signed remote status on alpha", func(v *control.View) bool {
			for _, o := range v.Federation.Outbound {
				if o.App == "burst" && o.Accepted && o.LastStatus != nil {
					var st control.FedStatus
					o.LastStatus.Decode(&st)
					for _, row := range st.Rows {
						if row.Observed == "RUNNING" && row.Evidence != "" {
							return true
						}
					}
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("revocation by the grantor stops the work there and is visible to the grantee", func(t *testing.T) {
		dhCLI(t, beta.Home, "federation", "revoke", digest, "--reason", "test")
		if err := beta.WaitFor(40*time.Second, "federated replicas stopped on beta", func(v *control.View) bool {
			a := app(v, "fed-alpha-burst")
			if a == nil {
				return true
			}
			return a.Observed == 0
		}); err != nil {
			t.Fatal(err)
		}
		if err := alpha.WaitFor(30*time.Second, "alpha sees the revocation", func(v *control.View) bool {
			for _, o := range v.Federation.Outbound {
				if o.App == "burst" && strings.Contains(o.Message, "revoked") {
					return true
				}
			}
			return false
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func fedManifestN(n int) string {
	return strings.Replace(fedManifest, "replicas: %d", "replicas: "+itoa(n), 1)
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }
