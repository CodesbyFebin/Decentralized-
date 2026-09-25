// Package identity implements dh/v1 Ed25519 identities and dh1 node IDs.
//
// A node ID is "dh1" followed by the first 26 characters of the lowercase,
// unpadded RFC 4648 base32 encoding of BLAKE3-256(raw 32-byte public key).
// The ID is derived from the genesis key and stays stable across key
// rotation; later keys are bound to it by signed rotation statements.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zeebo/blake3"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// IDPattern matches a syntactically valid dh1 identifier.
var IDPattern = regexp.MustCompile(`^dh1[a-z2-7]{26}$`)

// Identity is an Ed25519 key pair with its derived dh1 ID.
type Identity struct {
	ID   string
	Pub  ed25519.PublicKey
	Priv ed25519.PrivateKey
}

// NodeID derives the dh1 identifier for a raw Ed25519 public key.
func NodeID(pub ed25519.PublicKey) string {
	sum := blake3.Sum256(pub)
	return "dh1" + strings.ToLower(b32.EncodeToString(sum[:]))[:26]
}

// EncodePub encodes a public key for the wire: base64url without padding.
func EncodePub(pub ed25519.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(pub)
}

// DecodePub parses a wire-encoded public key.
func DecodePub(s string) (ed25519.PublicKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("identity: public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("identity: public key has %d bytes, want 32", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// Generate creates a fresh identity.
func Generate() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Identity{ID: NodeID(pub), Pub: pub, Priv: priv}, nil
}

// FromSeed builds a deterministic identity (tests and conformance vectors).
func FromSeed(seed []byte) *Identity {
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return &Identity{ID: NodeID(pub), Pub: pub, Priv: priv}
}

// PubString is the wire encoding of this identity's public key.
func (i *Identity) PubString() string { return EncodePub(i.Pub) }

// Sign signs msg with the private key.
func (i *Identity) Sign(msg []byte) []byte { return ed25519.Sign(i.Priv, msg) }

const (
	keyFile = "identity.key"
	pubFile = "identity.pub"
)

// LoadOrCreate loads the identity stored in dir, creating one if absent.
// The private key file is written with mode 0600 and is never served.
func LoadOrCreate(dir string) (*Identity, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	id, err := Load(dir)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	id, err = Generate()
	if err != nil {
		return nil, err
	}
	if err := Save(dir, id); err != nil {
		return nil, err
	}
	return id, nil
}

// Load reads an identity from dir.
func Load(dir string) (*Identity, error) {
	keyPath := filepath.Join(dir, keyFile)
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(keyPath); err == nil && st.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(keyPath, 0o600); err != nil {
			return nil, fmt.Errorf("identity: tighten key permissions: %w", err)
		}
	}
	return ParsePrivatePEM(raw)
}

// ParsePrivatePEM parses a PKCS#8 PEM Ed25519 private key.
func ParsePrivatePEM(raw []byte) (*Identity, error) {
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("identity: key file is not a PKCS#8 PEM private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("identity: parse key: %w", err)
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("identity: key is not Ed25519")
	}
	pub := priv.Public().(ed25519.PublicKey)
	return &Identity{ID: NodeID(pub), Pub: pub, Priv: priv}, nil
}

// PrivatePEM encodes the private key as PKCS#8 PEM.
func (i *Identity) PrivatePEM() []byte {
	der, err := x509.MarshalPKCS8PrivateKey(i.Priv)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// Save writes the identity to dir (key 0600, public key 0644).
func Save(dir string, id *Identity) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(dir, keyFile), id.PrivatePEM(), 0o600); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, pubFile), []byte(id.PubString()+"\n"), 0o644)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
