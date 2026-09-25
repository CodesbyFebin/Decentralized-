// Package peer defines the host peer protocol that runs inside the
// WireGuard mesh (dh/v1 §9). A request's mesh source address identifies the
// sender (WireGuard cryptokey routing), so the host authorizes by the
// bundle's address-to-identity map. Content is still verified end to end:
// chunks by BLAKE3, ledgers by hash chain and signature.
package peer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/envelope"
)

// Port is the peer API port on every mesh address.
const Port = 7800

// Client calls a host's peer API through the mesh.
type Client struct {
	HTTP *http.Client
}

func base(meshIP string) string { return fmt.Sprintf("http://%s:%d", meshIP, Port) }

func (c *Client) do(ctx context.Context, method, u string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("peer %s: %s: %s", u, resp.Status, bytes.TrimSpace(b))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if raw, ok := out.(*[]byte); ok {
		*raw, err = io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		return err
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Ping measures a round trip through the tunnel.
func (c *Client) Ping(ctx context.Context, meshIP string) (time.Duration, error) {
	start := time.Now()
	var out map[string]any
	if err := c.do(ctx, "GET", base(meshIP)+"/peer/v1/ping", nil, &out); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// GetChunk fetches a chunk; the caller must verify its hash.
func (c *Client) GetChunk(ctx context.Context, meshIP, id string) ([]byte, error) {
	var b []byte
	err := c.do(ctx, "GET", base(meshIP)+"/peer/v1/chunks/"+url.PathEscape(id), nil, &b)
	return b, err
}

// PutChunk pushes a chunk; the receiver verifies its hash before storing.
func (c *Client) PutChunk(ctx context.Context, meshIP, id string, data []byte) error {
	return c.do(ctx, "PUT", base(meshIP)+"/peer/v1/chunks/"+url.PathEscape(id), data, nil)
}

// Buckets returns the 256 bucket roots of the chunks a peer holds for a snapshot.
func (c *Client) Buckets(ctx context.Context, meshIP, volume, snapshot string) (*BucketsReply, error) {
	var out BucketsReply
	u := base(meshIP) + "/peer/v1/volumes/buckets?volume=" + url.QueryEscape(volume) + "&snapshot=" + url.QueryEscape(snapshot)
	return &out, c.do(ctx, "GET", u, nil, &out)
}

// Bucket lists the chunk ids a peer holds in one bucket of a snapshot.
func (c *Client) Bucket(ctx context.Context, meshIP, volume, snapshot string, bucket int) ([]string, error) {
	var out []string
	u := base(meshIP) + "/peer/v1/volumes/bucket?volume=" + url.QueryEscape(volume) + "&snapshot=" + url.QueryEscape(snapshot) + "&bucket=" + strconv.Itoa(bucket)
	return out, c.do(ctx, "GET", u, nil, &out)
}

// BucketsReply carries a peer's view of one snapshot.
type BucketsReply struct {
	Root    string      `json:"root"`
	Present int64       `json:"present"`
	Buckets [256]string `json:"buckets"`
}

// LedgerReply is a host ledger segment with a signed head.
type LedgerReply struct {
	Entries    []audit.Entry      `json:"entries"`
	Checkpoint *envelope.Envelope `json:"checkpoint"`
	Corrupt    *audit.Break       `json:"corrupt"`
}

// Ledger fetches host ledger entries after seq from.
func (c *Client) Ledger(ctx context.Context, meshIP string, from int64, limit int) (*LedgerReply, error) {
	var out LedgerReply
	u := fmt.Sprintf("%s/peer/v1/ledger?from=%d&limit=%d", base(meshIP), from, limit)
	return &out, c.do(ctx, "GET", u, nil, &out)
}

// Logs fetches the tail of a workload's output.
func (c *Client) Logs(ctx context.Context, meshIP, assignment string, tail int) ([]byte, error) {
	var b []byte
	u := fmt.Sprintf("%s/peer/v1/logs?assignment=%s&tail=%d", base(meshIP), url.QueryEscape(assignment), tail)
	return b, c.do(ctx, "GET", u, nil, &b)
}

// ExecRequest asks a host to run a command inside a workload. The host
// requires an operator capability anchored in its pinned root and its own
// policy must allow exec; the control plane only relays it.
type ExecRequest struct {
	Assignment string   `json:"assignment"`
	Argv       []string `json:"argv"`
	Capability string   `json:"capability"`
}

// ExecReply is the command result.
type ExecReply struct {
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
	Refused  string `json:"refused"`
}

// Exec relays an exec request.
func (c *Client) Exec(ctx context.Context, meshIP string, req ExecRequest) (*ExecReply, error) {
	b, _ := json.Marshal(req)
	var out ExecReply
	err := c.do(ctx, "POST", base(meshIP)+"/peer/v1/exec", b, &out)
	if err != nil && out.Refused == "" {
		return nil, err
	}
	return &out, nil
}

// ErrUnauthorized is returned to peers the host cannot map to an identity.
var ErrUnauthorized = errors.New("peer: mesh source is not an authorized peer")
