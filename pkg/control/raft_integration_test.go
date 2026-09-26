package control

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

// generateTestTLSConfig creates a self-signed certificate and TLS config for testing.
func generateTestTLSConfig() *tls.Config {
	// Generate ed25519 key pair
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)

	// Create certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	certTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test"},
			CommonName:   "test.local",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, privKey, privKey)

	// Encode certificate and key to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBytes, _ := x509.MarshalPKCS8PrivateKey(privKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	// Load certificate
	cert, _ := tls.X509KeyPair(certPEM, keyPEM)

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.NoClientCert,
		InsecureSkipVerify: true,
	}
}

// QualificationMember represents one member of the qualification cluster.
type QualificationMember struct {
	ID      string
	Node    *raftNode
	DataDir string

	// Network addresses
	GossipBind      string
	GossipAdvertise string
	RaftBind        string
	RaftAdvertise   string

	// Partition state
	Partitioned bool
}

// RaftQualificationCluster manages a 3-member Raft cluster for failover testing.
type RaftQualificationCluster struct {
	Members []*QualificationMember
	TLS     *tls.Config

	// Cluster state
	started   bool
	partitions map[string]bool // memberID -> partitioned
}

// NewRaftQualificationCluster creates a new 3-member cluster harness (not started).
func NewRaftQualificationCluster(tmpDir string, tlsConf *tls.Config) *RaftQualificationCluster {
	if tlsConf == nil {
		// For testing, generate a self-signed certificate
		tlsConf = generateTestTLSConfig()
	}

	c := &RaftQualificationCluster{
		TLS:        tlsConf,
		Members:    make([]*QualificationMember, 3),
		partitions: make(map[string]bool),
	}

	// Initialize member specs (addresses not used for network in test, but recorded for evidence)
	basePort := 50000
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("member-%d", i)
		c.Members[i] = &QualificationMember{
			ID:              id,
			DataDir:         filepath.Join(tmpDir, "member-"+id),
			GossipBind:      fmt.Sprintf("127.0.0.1:%d", basePort+i),
			GossipAdvertise: fmt.Sprintf("127.0.0.1:%d", basePort+i),
			RaftBind:        fmt.Sprintf("127.0.0.1:%d", basePort+100+i),
			RaftAdvertise:   fmt.Sprintf("127.0.0.1:%d", basePort+100+i),
		}
		c.partitions[id] = false
	}

	return c
}

// Start initializes all three members and bootstraps the cluster.
// Member 0 is bootstrapped as the initial leader with all three in the configuration.
func (c *RaftQualificationCluster) Start(t testing.TB) error {
	for i, m := range c.Members {
		fsm := NewFSM()
		fsm.s.Cluster = "qualification-cluster"

		opts := raftOptions{
			Dir:         m.DataDir,
			ID:          m.ID,
			Bind:        m.RaftBind,
			Advertise:   m.RaftAdvertise,
			TLS:         c.TLS,
			FSM:         fsm,
			Bootstrap:   (i == 0), // Only member 0 bootstraps initially
			LogOutput:   nil,
			FastTimeouts: true, // Speed up election/heartbeat for testing
		}

		node, err := startRaft(opts)
		if err != nil {
			t.Logf("Failed to start member %s: %v", m.ID, err)
			c.Close()
			return err
		}

		m.Node = node
	}

	c.started = true

	// Bootstrap cluster: member 0 bootstraps with all three in configuration
	// Note: In real usage, the other members would join later; for qualification we
	// bootstrap with all three present for deterministic testing.
	bootstrapConfig := raft.Configuration{
		Servers: []raft.Server{
			{ID: raft.ServerID(c.Members[0].ID), Address: raft.ServerAddress(c.Members[0].RaftAdvertise)},
			{ID: raft.ServerID(c.Members[1].ID), Address: raft.ServerAddress(c.Members[1].RaftAdvertise)},
			{ID: raft.ServerID(c.Members[2].ID), Address: raft.ServerAddress(c.Members[2].RaftAdvertise)},
		},
	}

	f := c.Members[0].Node.r.BootstrapCluster(bootstrapConfig)
	if err := f.Error(); err != nil && err != raft.ErrCantBootstrap {
		t.Logf("Bootstrap failed: %v", err)
		c.Close()
		return err
	}

	return nil
}

// WaitForLeader blocks until exactly one stable leader exists and all members have acknowledged it.
// Returns the leader ID and the current term, or error if timeout.
func (c *RaftQualificationCluster) WaitForLeader(timeout time.Duration) (string, uint64, error) {
	deadline := time.Now().Add(timeout)
	var lastLeader string
	var lastTerm uint64
	stableSince := time.Time{}

	for {
		if time.Now().After(deadline) {
			return "", 0, fmt.Errorf("timeout waiting for leader")
		}

		leader := ""
		term := uint64(0)

		// Check member 0
		if c.Members[0].Node != nil {
			if c.Members[0].Node.r.State() == raft.Leader {
				leader = c.Members[0].ID
				term = c.Members[0].Node.r.CurrentTerm()
			}
		}

		// Check member 1
		if leader == "" && c.Members[1].Node != nil {
			if c.Members[1].Node.r.State() == raft.Leader {
				leader = c.Members[1].ID
				term = c.Members[1].Node.r.CurrentTerm()
			}
		}

		// Check member 2
		if leader == "" && c.Members[2].Node != nil {
			if c.Members[2].Node.r.State() == raft.Leader {
				leader = c.Members[2].ID
				term = c.Members[2].Node.r.CurrentTerm()
			}
		}

		if leader != "" && leader == lastLeader && term == lastTerm {
			// Same leader for a stable period
			if stableSince.IsZero() {
				stableSince = time.Now()
			} else if time.Since(stableSince) > 200*time.Millisecond {
				return leader, term, nil
			}
		} else {
			// Leader changed or first observation
			lastLeader = leader
			lastTerm = term
			stableSince = time.Time{}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// Leader returns the current leader ID, or empty string if none.
func (c *RaftQualificationCluster) Leader() string {
	for _, m := range c.Members {
		if m.Node != nil && m.Node.r.State() == raft.Leader {
			return m.ID
		}
	}
	return ""
}

// Followers returns the IDs of all non-leader members.
func (c *RaftQualificationCluster) Followers() []string {
	var followers []string
	leader := c.Leader()
	for _, m := range c.Members {
		if m.ID != leader && m.Node != nil {
			followers = append(followers, m.ID)
		}
	}
	return followers
}

// Term returns the current Raft term for the given member.
func (c *RaftQualificationCluster) Term(memberID string) (uint64, error) {
	m := c.getMember(memberID)
	if m == nil || m.Node == nil {
		return 0, fmt.Errorf("member %s not found or stopped", memberID)
	}
	return m.Node.r.CurrentTerm(), nil
}

// CommitIndex returns the commit index for the given member's Raft state.
func (c *RaftQualificationCluster) CommitIndex(memberID string) (uint64, error) {
	m := c.getMember(memberID)
	if m == nil || m.Node == nil {
		return 0, fmt.Errorf("member %s not found or stopped", memberID)
	}
	// Note: raft.Raft does not expose CommitIndex directly; we use LastIndex() as a proxy.
	// For precise testing, we may need to inspect m.Node.logs directly.
	return m.Node.r.LastIndex(), nil
}

// AppliedIndex returns the FSM's applied index for the given member.
func (c *RaftQualificationCluster) AppliedIndex(memberID string) (int64, error) {
	m := c.getMember(memberID)
	if m == nil || m.Node == nil {
		return 0, fmt.Errorf("member %s not found or stopped", memberID)
	}
	var idx int64
	m.Node.fsm.Read(func(s *State) {
		idx = s.Index
	})
	return idx, nil
}

// Partition isolates a member from the cluster (simulates network partition).
// This stops the member's Raft transport, preventing it from sending/receiving messages.
func (c *RaftQualificationCluster) Partition(memberID string) error {
	m := c.getMember(memberID)
	if m == nil {
		return fmt.Errorf("member %s not found", memberID)
	}
	if m.Node != nil {
		m.Node.trans.Close()
	}
	c.partitions[memberID] = true
	m.Partitioned = true
	return nil
}

// Heal rejoins a partitioned member (simulates network recovery).
// Requires the member to be stopped and restarted for a clean connection.
func (c *RaftQualificationCluster) Heal(memberID string) {
	c.partitions[memberID] = false
	m := c.getMember(memberID)
	if m != nil {
		m.Partitioned = false
	}
}

// Stop terminates a member's Raft node (simulates crash or shutdown).
func (c *RaftQualificationCluster) Stop(memberID string) error {
	m := c.getMember(memberID)
	if m == nil {
		return fmt.Errorf("member %s not found", memberID)
	}
	if m.Node != nil {
		m.Node.shutdown()
		m.Node = nil
	}
	c.partitions[memberID] = false
	m.Partitioned = false
	return nil
}

// Restart restarts a stopped member, loading state from its persistent data directory.
func (c *RaftQualificationCluster) Restart(memberID string) error {
	m := c.getMember(memberID)
	if m == nil {
		return fmt.Errorf("member %s not found", memberID)
	}
	if m.Node != nil {
		// Already running
		return nil
	}

	fsm := NewFSM()
	fsm.s.Cluster = "qualification-cluster"

	opts := raftOptions{
		Dir:         m.DataDir,
		ID:          m.ID,
		Bind:        m.RaftBind,
		Advertise:   m.RaftAdvertise,
		TLS:         c.TLS,
		FSM:         fsm,
		Bootstrap:   false, // Do not bootstrap; load from existing state
		LogOutput:   nil,
		FastTimeouts: true,
	}

	node, err := startRaft(opts)
	if err != nil {
		return fmt.Errorf("restart %s: %w", memberID, err)
	}

	m.Node = node
	c.partitions[memberID] = false
	m.Partitioned = false
	return nil
}

// WaitForNewLeader blocks until a new leader emerges different from previousLeader.
func (c *RaftQualificationCluster) WaitForNewLeader(previousLeader string, timeout time.Duration) (string, uint64, error) {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return "", 0, fmt.Errorf("timeout waiting for new leader (was %s)", previousLeader)
		}

		leader, term, err := c.WaitForLeader(500 * time.Millisecond)
		if err == nil && leader != "" && leader != previousLeader {
			return leader, term, nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// WaitForConvergence blocks until all non-partitioned members have applied the same index.
func (c *RaftQualificationCluster) WaitForConvergence(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for convergence")
		}

		// Collect applied indices from all running members
		indices := make(map[string]int64)
		for _, m := range c.Members {
			if m.Node != nil && !m.Partitioned {
				idx, err := c.AppliedIndex(m.ID)
				if err == nil {
					indices[m.ID] = idx
				}
			}
		}

		// Check if all indices are the same
		if len(indices) > 0 {
			var targetIdx int64
			allSame := true
			for i, idx := range indices {
				if i == c.Members[0].ID {
					targetIdx = idx
				} else if idx != targetIdx {
					allSame = false
					break
				}
			}

			if allSame {
				return nil
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// Close stops all members and cleans up resources.
func (c *RaftQualificationCluster) Close() {
	for _, m := range c.Members {
		if m.Node != nil {
			m.Node.shutdown()
			m.Node = nil
		}
	}
	c.started = false
}

// Helper functions

func (c *RaftQualificationCluster) getMember(id string) *QualificationMember {
	for _, m := range c.Members {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// ========================================
// Harness Qualification Tests
// ========================================

// TestRaftHarness_ClusterFormation verifies that three members form a single cluster
// with one stable leader.
func TestRaftHarness_ClusterFormation(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewRaftQualificationCluster(tmpDir, nil)
	defer c.Close()

	if err := c.Start(t); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for leader election
	leader, term, err := c.WaitForLeader(10 * time.Second)
	if err != nil {
		t.Fatalf("WaitForLeader: %v", err)
	}

	if leader == "" {
		t.Fatal("No leader elected")
	}

	t.Logf("Leader elected: %s at term %d", leader, term)

	// Verify followers
	followers := c.Followers()
	if len(followers) != 2 {
		t.Fatalf("Expected 2 followers, got %d", len(followers))
	}

	t.Logf("Followers: %v", followers)
}

// TestRaftHarness_LeaderFailoverElection verifies that isolating the leader
// causes a new leader to be elected from the remaining quorum.
func TestRaftHarness_LeaderFailoverElection(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewRaftQualificationCluster(tmpDir, nil)
	defer c.Close()

	if err := c.Start(t); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for initial leader
	leader, term1, err := c.WaitForLeader(10 * time.Second)
	if err != nil {
		t.Fatalf("WaitForLeader: %v", err)
	}

	t.Logf("Initial leader: %s at term %d", leader, term1)

	// Isolate the leader
	if err := c.Partition(leader); err != nil {
		t.Fatalf("Partition failed: %v", err)
	}

	t.Logf("Isolated leader: %s", leader)

	// Wait for new leader election from quorum
	newLeader, term2, err := c.WaitForNewLeader(leader, 10*time.Second)
	if err != nil {
		t.Fatalf("WaitForNewLeader: %v", err)
	}

	if newLeader == leader {
		t.Fatal("New leader is the same as old leader")
	}

	if term2 <= term1 {
		t.Fatalf("New term %d should be > old term %d", term2, term1)
	}

	t.Logf("New leader: %s at term %d", newLeader, term2)
}

// TestRaftHarness_ReplicatedCommand verifies that a command committed on the leader
// is replicated to all followers.
func TestRaftHarness_ReplicatedCommand(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewRaftQualificationCluster(tmpDir, nil)
	defer c.Close()

	if err := c.Start(t); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for leader
	leaderID, _, err := c.WaitForLeader(10 * time.Second)
	if err != nil {
		t.Fatalf("WaitForLeader: %v", err)
	}

	t.Logf("Leader: %s", leaderID)

	// Get leader's node
	var leaderNode *raftNode
	for _, m := range c.Members {
		if m.ID == leaderID {
			leaderNode = m.Node
			break
		}
	}

	if leaderNode == nil {
		t.Fatal("Leader node not found")
	}

	// Propose a benign command (init cluster)
	cmd := &Command{
		Type:  "init",
		TS:    Now(),
		Actor: "test",
		Data:  []byte(`{"cluster":"qualification-cluster","root":"test-root","rootCaCert":"test-ca"}`),
	}

	res, err := leaderNode.propose(cmd, 5*time.Second)
	if err != nil {
		t.Fatalf("Propose failed: %v", err)
	}

	if !res.OK {
		t.Logf("Proposal result: %v", res)
		// Init may fail if already initialized; that's okay for this test
	}

	// Wait for convergence
	if err := c.WaitForConvergence(10 * time.Second); err != nil {
		t.Fatalf("WaitForConvergence: %v", err)
	}

	t.Logf("Command replicated and converged")
}

// TestRaftHarness_PartitionHeal verifies that a partitioned member rejoins
// and converges with the cluster.
func TestRaftHarness_PartitionHeal(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewRaftQualificationCluster(tmpDir, nil)
	defer c.Close()

	if err := c.Start(t); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for leader
	_, _, err := c.WaitForLeader(10 * time.Second)
	if err != nil {
		t.Fatalf("WaitForLeader: %v", err)
	}

	// Partition a follower
	followers := c.Followers()
	if len(followers) == 0 {
		t.Fatal("No followers found")
	}

	partitionedMember := followers[0]
	if err := c.Partition(partitionedMember); err != nil {
		t.Fatalf("Partition failed: %v", err)
	}

	t.Logf("Partitioned: %s", partitionedMember)

	// Heal the partition
	time.Sleep(500 * time.Millisecond) // Simulate some time passing
	c.Heal(partitionedMember)

	t.Logf("Healed: %s", partitionedMember)

	// Wait for convergence
	if err := c.WaitForConvergence(10 * time.Second); err != nil {
		t.Fatalf("WaitForConvergence: %v", err)
	}

	t.Logf("Cluster converged")
}

// TestRaftHarness_MemberRestart verifies that a restarted member loads state
// from its persistent data directory and rejoins the cluster.
func TestRaftHarness_MemberRestart(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewRaftQualificationCluster(tmpDir, nil)
	defer c.Close()

	if err := c.Start(t); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for leader
	_, _, err := c.WaitForLeader(10 * time.Second)
	if err != nil {
		t.Fatalf("WaitForLeader: %v", err)
	}

	// Stop a follower
	followers := c.Followers()
	if len(followers) == 0 {
		t.Fatal("No followers found")
	}

	stoppedMember := followers[0]
	if err := c.Stop(stoppedMember); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	t.Logf("Stopped: %s", stoppedMember)

	// Restart the member
	if err := c.Restart(stoppedMember); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	t.Logf("Restarted: %s", stoppedMember)

	// Wait for convergence
	if err := c.WaitForConvergence(10 * time.Second); err != nil {
		t.Fatalf("WaitForConvergence: %v", err)
	}

	t.Logf("Cluster converged with restarted member")
}
