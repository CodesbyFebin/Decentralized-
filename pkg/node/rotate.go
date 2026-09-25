package node

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
)

// RotateKey moves this host to a new Ed25519 key. The rotation statement is
// signed by the new key (envelope) and countersigned by the old key, so
// neither a stolen new key nor the control plane alone can rotate a host.
// The old key stays valid for the grace period so peers never see a gap;
// the stable dh1 id (derived from the genesis key) does not change.
func RotateKey(dataDir string, grace time.Duration) (string, error) {
	cfg := Config{DataDir: dataDir}
	a := &Agent{cfg: cfg}
	if err := a.loadState(); err != nil {
		return "", err
	}
	if a.st.Root == "" {
		return "", ErrNotJoined
	}
	idDir := filepath.Join(dataDir, "identity")
	old, err := identity.Load(idDir)
	if err != nil {
		return "", err
	}
	next, err := identity.Generate()
	if err != nil {
		return "", err
	}
	now := time.Now().UnixMilli()
	rot := api.Rotation{Node: a.st.NodeID, OldPub: old.PubString(), NewPub: next.PubString(), NotBefore: now - 1000, GraceUntil: now + grace.Milliseconds(), TS: now}
	env, err := envelope.Sign(next, a.st.NodeID, envelope.KindRotation, rot)
	if err != nil {
		return "", err
	}
	oldSig := base64.RawURLEncoding.EncodeToString(old.Sign(envelope.SigningInput(envelope.KindRotation, env.Payload)))
	cp, err := newCPClient(a.st.Endpoints, a.st.TLS, a.st.RootCACert)
	if err != nil {
		return "", err
	}
	var res struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	if err := cp.post("/v1/rotate", map[string]any{"env": env, "oldSig": oldSig}, &res); err != nil {
		return "", err
	}
	if !res.OK {
		return "", fmt.Errorf("rotation refused: %s", res.Message)
	}
	// Keep the old key alongside for the grace period, then install the new one.
	if err := os.WriteFile(filepath.Join(idDir, fmt.Sprintf("identity.key.retired-%d", now)), old.PrivatePEM(), 0o600); err != nil {
		return "", err
	}
	if err := identity.Save(idDir, next); err != nil {
		return "", err
	}
	// The running agent is the only ledger writer; it journals the reload.
	// The signed statement is kept next to the keys as evidence.
	rec, _ := json.MarshalIndent(map[string]any{"rotation": env, "oldSig": oldSig}, "", "  ")
	if err := os.WriteFile(filepath.Join(idDir, fmt.Sprintf("rotation-%d.json", now)), rec, 0o600); err != nil {
		return "", err
	}
	return next.PubString(), nil
}
