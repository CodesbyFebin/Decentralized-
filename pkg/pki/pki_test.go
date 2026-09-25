package pki

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"decentralized.host/pkg/identity"
)

func TestBootstrapPinning(t *testing.T) {
	cert, fp, err := BootstrapCert()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "ok") }))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.StartTLS()
	defer srv.Close()

	pinned := &http.Client{Transport: &http.Transport{TLSClientConfig: PinnedTLS(fp)}}
	if resp, err := pinned.Get(srv.URL); err != nil {
		t.Fatalf("pinned fingerprint rejected: %v", err)
	} else {
		resp.Body.Close()
	}
	wrong := &http.Client{Transport: &http.Transport{TLSClientConfig: PinnedTLS("sha256:" + strings.Repeat("0", 64))}}
	if _, err := wrong.Get(srv.URL); err == nil || !strings.Contains(err.Error(), "pinned fingerprint") {
		t.Fatalf("a different certificate must be refused, got %v", err)
	}
}

func TestMemberCertificateVerifiesAgainstRoot(t *testing.T) {
	root, _ := identity.Generate()
	member, _ := identity.Generate()
	ca, err := RootCA(root, "t")
	if err != nil {
		t.Fatal(err)
	}
	pem, err := IssueMember(root, ca, member.Pub, []string{"127.0.0.1:7700"})
	if err != nil {
		t.Fatal(err)
	}
	conf, err := MemberTLS(member, pem, ca, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.TLS = &tls.Config{Certificates: conf.Certificates}
	srv.StartTLS()
	defer srv.Close()
	client, err := ClientTLS(ca)
	if err != nil {
		t.Fatal(err)
	}
	if resp, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: client}}).Get(srv.URL); err != nil {
		t.Fatalf("member certificate (IP SAN from host:port) did not verify: %v", err)
	} else {
		resp.Body.Close()
	}
	other, _ := identity.Generate()
	otherCA, _ := RootCA(other, "other")
	foreign, _ := ClientTLS(otherCA)
	if _, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: foreign}}).Get(srv.URL); err == nil {
		t.Fatal("a member certificate must not verify against another cluster's root")
	}
}
