package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"decentralized.host/pkg/canon"
	"decentralized.host/pkg/pki"
)

// cpClient talks to control-plane members, failing over between them.
type cpClient struct {
	mu        sync.Mutex
	endpoints []string
	scheme    string
	http      *http.Client
	last      int
}

func newCPClient(endpoints []string, useTLS bool, caPEM string) (*cpClient, error) {
	c := &cpClient{endpoints: endpoints, scheme: "http", http: &http.Client{Timeout: 5 * time.Second}}
	if useTLS {
		conf, err := pki.ClientTLS([]byte(caPEM))
		if err != nil {
			return nil, err
		}
		c.scheme = "https"
		c.http.Transport = &http.Transport{TLSClientConfig: conf}
	}
	return c, nil
}

func (c *cpClient) setEndpoints(eps []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(eps) > 0 {
		c.endpoints = eps
		if c.last >= len(eps) {
			c.last = 0
		}
	}
}

// ErrUnreachable means no member answered.
var ErrUnreachable = errors.New("control plane unreachable")

// HTTPError is a non-2xx reply.
type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("control plane %d: %s", e.Code, e.Message) }

// post sends body to path on the last good member, then the others.
func (c *cpClient) post(path string, body any, out any) error {
	b, err := canon.Wire(body)
	if err != nil {
		return err
	}
	c.mu.Lock()
	eps := append([]string(nil), c.endpoints...)
	start := c.last
	c.mu.Unlock()
	var lastErr error = ErrUnreachable
	for i := 0; i < len(eps); i++ {
		idx := (start + i) % len(eps)
		resp, err := c.http.Post(c.scheme+"://"+eps[idx]+path, "application/json", bytes.NewReader(b))
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrUnreachable, err)
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
			var m struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(data, &m)
			lastErr = fmt.Errorf("%w: %s", ErrUnreachable, m.Message)
			continue
		}
		c.mu.Lock()
		c.last = idx
		c.mu.Unlock()
		if resp.StatusCode >= 300 {
			var m struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(data, &m)
			return &HTTPError{Code: resp.StatusCode, Message: m.Message}
		}
		if out != nil {
			if raw, ok := out.(*[]byte); ok {
				*raw = data
				return nil
			}
			return json.Unmarshal(data, out)
		}
		return nil
	}
	return lastErr
}
