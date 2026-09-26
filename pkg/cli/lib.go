package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/node"
	"decentralized.host/pkg/storage"
)

// Library entry points used by the dev-cluster harness, integration tests
// and the chaos suite. They perform the same operations as the commands.

// Register adds a command from another package (dev, chaos, federation).
func Register(name, help string, run func(op *Operator, args []string) error) {
	reg(name, help, false, run)
}

// Init creates a cluster trust root in home and returns a loaded operator.
// useTLS makes every member serve its API over TLS.
func Init(home, cluster string, useTLS bool) (*Operator, error) {
	op := &Operator{Home: home}
	if err := cmdInit(op, []string{"--cluster", cluster, fmt.Sprintf("--tls=%v", useTLS)}); err != nil {
		return nil, err
	}
	return Load(home, cluster)
}

// Load opens an existing operator home.
func Load(home, cluster string) (*Operator, error) {
	op := &Operator{Home: home}
	return op, op.load(cluster)
}

// Bootstrap bootstraps the first member.
func (op *Operator) Bootstrap(api, code, fingerprint string) error {
	if err := cmdBootstrap(op, []string{"--api", api, "--code", code, "--fingerprint", fingerprint}); err != nil {
		return err
	}
	return op.load(op.Cfg.Cluster)
}

// AddMember adds a control-plane member.
func (op *Operator) AddMember(api, code, fingerprint string) error {
	if err := cmdAddMember(op, []string{"--api", api, "--code", code, "--fingerprint", fingerprint}); err != nil {
		return err
	}
	return op.load(op.Cfg.Cluster)
}

// InviteOpts configures a join token.
type InviteOpts struct {
	Roles []string
	Auto  bool
	TTL   time.Duration
	Note  string
}

// Invite registers an invite and returns the join token.
func (op *Operator) Invite(o InviteOpts) (string, error) {
	if o.TTL == 0 {
		o.TTL = 15 * time.Minute
	}
	if o.Roles == nil {
		o.Roles = []string{}
	}
	nonce := randomNonce()
	exp := time.Now().Add(o.TTL).UnixMilli()
	tok, err := capability.Mint(op.Root, capability.Caveats{Actions: []string{"node.join"}, Resources: []string{"cluster/" + op.Cfg.Cluster}, Expires: exp, Nonce: nonce}, "", "join")
	if err != nil {
		return "", err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/invites", map[string]any{"nonce": nonce, "expires": exp, "roles": o.Roles, "note": o.Note, "auto": o.Auto}, &r); err != nil {
		return "", err
	}
	return node.EncodeJoinToken(node.JoinToken{Cluster: op.Cfg.Cluster, Root: op.Root.PubString(), RootCACert: string(op.CAPEM), Endpoints: op.Cfg.Endpoints, TLS: op.Cfg.TLS, Capability: tok.Encode()}), nil
}

// RevokeInvite withdraws an unused join token (the token or its nonce).
func (op *Operator) RevokeInvite(tokenOrNonce string) (Result, error) {
	var r Result
	nonce, err := inviteNonce(tokenOrNonce)
	if err != nil {
		return r, err
	}
	err = op.Do("POST", "/api/v1/invites/"+nonce+"/revoke", nil, &r)
	return r, err
}

// View fetches the console projection.
func (op *Operator) View() (*control.View, error) { return op.view() }

// Apply applies a manifest.
func (op *Operator) Apply(src []byte) (Result, error) {
	var r Result
	err := op.Do("POST", "/api/v1/apply", src, &r)
	return r, err
}

// PushArtifact uploads an executable and optionally attests it.
func (op *Operator) PushArtifact(path, name string, sign bool) (image, digest string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	var r Result
	if err := op.Do("POST", "/api/v1/artifacts?name="+name, data, &r); err != nil {
		return "", "", err
	}
	d, _ := r.Data.(map[string]any)
	digest = fmt.Sprint(d["digest"])
	if digest != storage.ChunkID(data) {
		return "", "", errors.New("control plane reported a different digest")
	}
	if sign {
		if err := attest(op, digest, name); err != nil {
			return "", "", err
		}
	}
	return fmt.Sprint(d["image"]), digest, nil
}

// Attest signs an artifact attestation with the root key.
func (op *Operator) Attest(digest, name string) error { return attest(op, digest, name) }

// ResolveNode maps a host name to its id.
func (op *Operator) ResolveNode(q string) (string, error) { return op.resolveNode(q) }

// Dir returns the operator's cluster directory.
func (op *Operator) ClusterDir() string { return op.Dir }

// RootPEMPath returns the root CA file path.
func (op *Operator) RootCAPath() string { return filepath.Join(op.Dir, "root-ca.pem") }

// Bearer returns a short-lived operator capability for direct API calls.
func (op *Operator) Bearer() string { return op.bearer() }
