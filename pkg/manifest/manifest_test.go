package manifest

import (
	"strings"
	"testing"
)

const wiki = `
apiVersion: dh/v1
kind: Application
metadata:
  name: wiki
  owner: did:dh:z6Mk
spec:
  replicas: 3
  image: dh-beacon@b3:` + "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262" + `
  placement:
    tiers: [trusted, community]
    spread: failure-domain
    antiAffinity: hard
  resources:
    cpu: 500m
    mem: 512Mi
  ports:
    - name: http
  volumes:
    - name: data
      size: 10Gi
      durability:
        replicas: 3
        erasure: none
      mount: /var/wiki
  ingress:
    - host: wiki.example.com
      port: http
      tls: acme
  health:
    http: /healthz
    interval: 10s
`

func TestParseNormalizes(t *testing.T) {
	m, err := Parse([]byte(wiki))
	if err != nil {
		t.Fatal(err)
	}
	if m.Spec.Runtime != "process" || m.Spec.Resources.CPUMilli != 500 || m.Spec.Resources.MemBytes != 512<<20 {
		t.Fatalf("%+v", m.Spec)
	}
	if m.Spec.Placement.Tiers[0] != "community" {
		t.Fatal("tiers must be sorted for a stable hash")
	}
	if m.Spec.Volumes[0].SizeBytes != 10<<30 || m.Spec.Health.IntervalMs != 10000 {
		t.Fatal("quantities")
	}
}

func TestHashIsSpellingIndependent(t *testing.T) {
	a, _ := Parse([]byte(wiki))
	alt := strings.Replace(strings.Replace(wiki, "cpu: 500m", "cpu: \"0.5\"", 1), "tiers: [trusted, community]", "tiers: [community, trusted]", 1)
	b, err := Parse([]byte(alt))
	if err != nil {
		t.Fatal(err)
	}
	if Hash(a) != Hash(b) {
		t.Fatal("equivalent manifests hash differently")
	}
	c, _ := Parse([]byte(strings.Replace(wiki, "replicas: 3\n  image", "replicas: 4\n  image", 1)))
	if Hash(a) == Hash(c) {
		t.Fatal("different manifests hash the same")
	}
}

func TestRejectsUnpinnedAndUnimplemented(t *testing.T) {
	bad := []struct{ from, to, want string }{
		{"dh-beacon@b3:af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262", "ghcr.io/example/wiki:latest", "digest-pinned"},
		{"erasure: none", "erasure: rs-4-2", "erasure coding is not implemented"},
		{"port: http", "port: grpc", "does not name a declared port"},
		{"name: wiki", "name: Wiki_App", "metadata.name"},
	}
	for _, tc := range bad {
		_, err := Parse([]byte(strings.Replace(wiki, tc.from, tc.to, 1)))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v", tc.want, err)
		}
	}
	if _, err := Parse([]byte(wiki + "  bogusField: 1\n")); err == nil {
		t.Error("unknown fields must be rejected")
	}
}
