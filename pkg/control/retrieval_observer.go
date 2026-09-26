package control

import (
	"sync"
	"time"
)

// RetrievalQualificationObserver instruments the authorization→decryption boundary
// to prove that authorized consumption commits before decrypt and survives failover.
type RetrievalQualificationObserver interface {
	// AuthorizationProposed is called when FSM.AuthorizeSecretRetrievalCommand creates the command.
	// metadata contains: requestDigest, requestID, nodeID, secretID
	AuthorizationProposed(metadata map[string]string)

	// AuthorizationCommitted is called after FSM.Apply completes and the lock is released,
	// confirming the authorization is now in replicated state.
	AuthorizationCommitted(metadata map[string]string)

	// BeforeDecrypt is called just before DecryptSecret() is invoked.
	// Returns a channel that can block the decrypt (for fault injection).
	BeforeDecrypt(metadata map[string]string) <-chan struct{}

	// AfterDecrypt is called after DecryptSecret() completes (on success or error).
	AfterDecrypt(metadata map[string]string, err error)

	// BeforeResponseWrite is called before sending the plaintext response.
	BeforeResponseWrite(metadata map[string]string) <-chan struct{}

	// AfterResponseWrite is called after the response is sent.
	AfterResponseWrite(metadata map[string]string, err error)
}

// NoOpObserver implements RetrievalQualificationObserver with no instrumentation.
type NoOpObserver struct{}

func (o *NoOpObserver) AuthorizationProposed(metadata map[string]string)      {}
func (o *NoOpObserver) AuthorizationCommitted(metadata map[string]string)     {}
func (o *NoOpObserver) BeforeDecrypt(metadata map[string]string) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (o *NoOpObserver) AfterDecrypt(metadata map[string]string, err error)     {}
func (o *NoOpObserver) BeforeResponseWrite(metadata map[string]string) <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (o *NoOpObserver) AfterResponseWrite(metadata map[string]string, err error) {}

// R1TestObserver collects evidence for R1-01 (commit-before-decrypt) qualification.
// Thread-safe via internal mutex.
type R1TestObserver struct {
	mu sync.Mutex

	// Event sequence tracking
	ProposedEvents    []map[string]string
	CommittedEvents   []map[string]string
	DecryptedEvents   []map[string]string
	ResponseSentEvents []map[string]string

	// Counters
	AuthorizationProposedCount  int64
	AuthorizationCommittedCount int64
	BeforeDecryptCount          int64
	AfterDecryptCount           int64
	BeforeResponseWriteCount    int64
	AfterResponseWriteCount     int64

	// Per-request tracking for R1-02B precision
	// Maps requestDigest -> RequestMetrics
	RequestMetrics map[string]*RequestMetrics

	// Fault injection: channels that can be set to block operations
	// If nil, operation proceeds immediately; otherwise blocks until channel closes
	BlockBeforeDecrypt      <-chan struct{}
	BlockBeforeResponseWrite <-chan struct{}

	// Tracking errors
	DecryptErrors []error
}

// RequestMetrics tracks detailed metrics for a specific secret retrieval request.
type RequestMetrics struct {
	RequestDigest               string
	LeaderCommitConfirmations   int64
	DecryptInvocations          int64
	DecryptSuccesses            int64
	DecryptErrors               []error
	ResponseWriteAttempts       int64
	ResponseWriteSuccesses      int64
	AuthorizationCommittedTime  int64 // nanoseconds
	DecryptStartTime            int64 // nanoseconds
	DecryptSuccessTime          int64 // nanoseconds
	ResponseWriteStartTime      int64 // nanoseconds
	ResponseWriteSuccessTime    int64 // nanoseconds
}

func NewR1TestObserver() *R1TestObserver {
	return &R1TestObserver{
		ProposedEvents:     make([]map[string]string, 0),
		CommittedEvents:    make([]map[string]string, 0),
		DecryptedEvents:    make([]map[string]string, 0),
		ResponseSentEvents: make([]map[string]string, 0),
		DecryptErrors:      make([]error, 0),
		RequestMetrics:     make(map[string]*RequestMetrics),
	}
}

func (o *R1TestObserver) AuthorizationProposed(metadata map[string]string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.ProposedEvents = append(o.ProposedEvents, copyMetadata(metadata))
	o.AuthorizationProposedCount++
}

// getOrCreateMetrics returns the RequestMetrics for a digest, creating if needed.
func (o *R1TestObserver) getOrCreateMetrics(requestDigest string) *RequestMetrics {
	if m, ok := o.RequestMetrics[requestDigest]; ok {
		return m
	}
	m := &RequestMetrics{RequestDigest: requestDigest}
	o.RequestMetrics[requestDigest] = m
	return m
}

func (o *R1TestObserver) AuthorizationCommitted(metadata map[string]string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.CommittedEvents = append(o.CommittedEvents, copyMetadata(metadata))
	o.AuthorizationCommittedCount++

	// Track per-request metrics
	if requestDigest, ok := metadata["requestDigest"]; ok {
		m := o.getOrCreateMetrics(requestDigest)
		m.LeaderCommitConfirmations++
		m.AuthorizationCommittedTime = time.Now().UnixNano()
	}
}

func (o *R1TestObserver) BeforeDecrypt(metadata map[string]string) <-chan struct{} {
	o.mu.Lock()
	o.BeforeDecryptCount++
	blockChan := o.BlockBeforeDecrypt

	// Track per-request metrics
	if requestDigest, ok := metadata["requestDigest"]; ok {
		m := o.getOrCreateMetrics(requestDigest)
		m.DecryptInvocations++
		m.DecryptStartTime = time.Now().UnixNano()
	}

	o.mu.Unlock()

	if blockChan != nil {
		// Return a channel that will unblock when either the test's injection channel
		// closes OR we receive a signal to proceed
		return blockChan
	}

	// No blocking - return immediately closed channel
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (o *R1TestObserver) AfterDecrypt(metadata map[string]string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.DecryptedEvents = append(o.DecryptedEvents, copyMetadata(metadata))
	o.AfterDecryptCount++
	if err != nil {
		o.DecryptErrors = append(o.DecryptErrors, err)
	}

	// Track per-request metrics
	if requestDigest, ok := metadata["requestDigest"]; ok {
		m := o.getOrCreateMetrics(requestDigest)
		if err == nil {
			m.DecryptSuccesses++
			m.DecryptSuccessTime = time.Now().UnixNano()
		} else {
			m.DecryptErrors = append(m.DecryptErrors, err)
		}
	}
}

func (o *R1TestObserver) BeforeResponseWrite(metadata map[string]string) <-chan struct{} {
	o.mu.Lock()
	o.BeforeResponseWriteCount++
	blockChan := o.BlockBeforeResponseWrite

	// Track per-request metrics
	if requestDigest, ok := metadata["requestDigest"]; ok {
		m := o.getOrCreateMetrics(requestDigest)
		m.ResponseWriteAttempts++
		m.ResponseWriteStartTime = time.Now().UnixNano()
	}

	o.mu.Unlock()

	if blockChan != nil {
		return blockChan
	}

	ch := make(chan struct{})
	close(ch)
	return ch
}

func (o *R1TestObserver) AfterResponseWrite(metadata map[string]string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.ResponseSentEvents = append(o.ResponseSentEvents, copyMetadata(metadata))
	o.AfterResponseWriteCount++

	// Track per-request metrics
	if requestDigest, ok := metadata["requestDigest"]; ok {
		m := o.getOrCreateMetrics(requestDigest)
		if err == nil {
			m.ResponseWriteSuccesses++
			m.ResponseWriteSuccessTime = time.Now().UnixNano()
		}
	}
}

// GetCommittedCount returns the number of authorizations committed.
func (o *R1TestObserver) GetCommittedCount() int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.AuthorizationCommittedCount
}

// GetDecryptedCount returns the number of decrypts attempted.
func (o *R1TestObserver) GetDecryptedCount() int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.AfterDecryptCount
}

// GetResponseCount returns the number of responses sent.
func (o *R1TestObserver) GetResponseCount() int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.AfterResponseWriteCount
}

// GetEventsSnapshot returns a consistent snapshot of all recorded events.
func (o *R1TestObserver) GetEventsSnapshot() map[string]interface{} {
	o.mu.Lock()
	defer o.mu.Unlock()
	return map[string]interface{}{
		"ProposedEvents":    append([]map[string]string{}, o.ProposedEvents...),
		"CommittedEvents":   append([]map[string]string{}, o.CommittedEvents...),
		"DecryptedEvents":   append([]map[string]string{}, o.DecryptedEvents...),
		"ResponseSentEvents": append([]map[string]string{}, o.ResponseSentEvents...),
		"Counters": map[string]int64{
			"AuthorizationProposedCount":  o.AuthorizationProposedCount,
			"AuthorizationCommittedCount": o.AuthorizationCommittedCount,
			"BeforeDecryptCount":          o.BeforeDecryptCount,
			"AfterDecryptCount":           o.AfterDecryptCount,
			"BeforeResponseWriteCount":    o.BeforeResponseWriteCount,
			"AfterResponseWriteCount":     o.AfterResponseWriteCount,
		},
		"DecryptErrors": o.DecryptErrors,
	}
}

// GetRequestMetrics returns a copy of the RequestMetrics for a specific digest.
// Used by R1-02B to verify exact accounting.
func (o *R1TestObserver) GetRequestMetrics(requestDigest string) *RequestMetrics {
	o.mu.Lock()
	defer o.mu.Unlock()
	m, ok := o.RequestMetrics[requestDigest]
	if !ok {
		return nil
	}
	// Return a copy to prevent external mutation
	copy := *m
	return &copy
}

// GetAllRequestMetrics returns a map of all request metrics.
func (o *R1TestObserver) GetAllRequestMetrics() map[string]*RequestMetrics {
	o.mu.Lock()
	defer o.mu.Unlock()
	result := make(map[string]*RequestMetrics, len(o.RequestMetrics))
	for k, v := range o.RequestMetrics {
		copy := *v
		result[k] = &copy
	}
	return result
}

// copyMetadata returns a shallow copy of the metadata map.
func copyMetadata(m map[string]string) map[string]string {
	copy := make(map[string]string, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
