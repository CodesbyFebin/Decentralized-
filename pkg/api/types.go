// Package api defines the dh/v1 wire types shared by the control plane,
// host agents, the CLI and the conformance suite.
//
// Quantities on the wire are integers: CPU in millicores, memory and disk in
// bytes, durations in milliseconds, timestamps in unix milliseconds. Floats
// never appear in signed payloads (canonical JSON forbids them).
package api

import (
	"decentralized.host/pkg/envelope"
)

// ProtocolVersion is the wire protocol implemented by this module.
const ProtocolVersion = "dh/v1"

// ---------------------------------------------------------------- manifests

// Manifest is a normalized dh/v1 Application manifest.
type Manifest struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       AppSpec  `json:"spec"`
}

type Metadata struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type AppSpec struct {
	Replicas  int64             `json:"replicas"`
	Runtime   string            `json:"runtime"`             // process | docker
	Isolation string            `json:"isolation,omitempty"` // "" | PRIVATE | RESTRICTED
	Image     string            `json:"image"`               // name@b3:<hex> (process) or ref@sha256:<hex> (docker)
	Command   []string          `json:"command"`             // arguments; never interpreted by a shell
	Env       map[string]string `json:"env"`
	Resources Resources         `json:"resources"`
	Placement Placement         `json:"placement"`
	Ports     []Port            `json:"ports"`
	Volumes   []VolumeSpec      `json:"volumes"`
	Ingress   []Ingress         `json:"ingress"`
	Health    Health            `json:"health"`
	Update    Update            `json:"update"`
}

type Resources struct {
	CPUMilli int64 `json:"cpuMilli"`
	MemBytes int64 `json:"memBytes"`
}

type Placement struct {
	Tiers        []string `json:"tiers"`
	Spread       string   `json:"spread"`       // failure-domain | none
	AntiAffinity string   `json:"antiAffinity"` // hard | soft | none
	Arch         []string `json:"arch"`
	Features     []string `json:"features"`
	PreferRegion string   `json:"preferRegion"`
	Federation   []string `json:"federation"` // peer cluster names allowed to host replicas
}

type Port struct {
	Name      string `json:"name"`
	Container int64  `json:"container"` // listening port inside a container (docker); process workloads get $PORT
	Protocol  string `json:"protocol"`  // http | tcp
}

type VolumeSpec struct {
	Name            string     `json:"name"`
	SizeBytes       int64      `json:"sizeBytes"`
	Mount           string     `json:"mount"`
	Durability      Durability `json:"durability"`
	SnapshotEveryMs int64      `json:"snapshotEveryMs"`
	RetainSnapshots int64      `json:"retainSnapshots"`
}

type Durability struct {
	Replicas int64  `json:"replicas"`
	Erasure  string `json:"erasure"` // none (erasure coding is not implemented)
}

type Ingress struct {
	Host   string `json:"host"`
	Port   string `json:"port"`   // port name
	TLS    string `json:"tls"`    // acme | local | none
	Issuer string `json:"issuer"` // named ACME issuer, "" = default
}

type Health struct {
	HTTP             string `json:"http"`
	IntervalMs       int64  `json:"intervalMs"`
	TimeoutMs        int64  `json:"timeoutMs"`
	FailureThreshold int64  `json:"failureThreshold"`
}

type Update struct {
	Strategy       string `json:"strategy"` // rolling
	MaxUnavailable int64  `json:"maxUnavailable"`
}

// ------------------------------------------------------------- assignments

// Assignment is signed by a control-plane member (envelope kind
// "assignment"). It is intent, not truth: a host may refuse it.
type Assignment struct {
	ID           string            `json:"id"` // <app>/r<replica>
	App          string            `json:"app"`
	Replica      int64             `json:"replica"`
	Node         string            `json:"node"`
	Generation   int64             `json:"generation"`
	Desired      string            `json:"desired"` // running | stopped
	Runtime      string            `json:"runtime"`
	Isolation    string            `json:"isolation,omitempty"`
	Image        string            `json:"image"`
	Digest       string            `json:"digest"`
	Command      []string          `json:"command"`
	Env          map[string]string `json:"env"`
	Resources    Resources         `json:"resources"`
	Ports        []Port            `json:"ports"`
	Health       Health            `json:"health"`
	Volumes      []AssignedVolume  `json:"volumes"`
	Tiers        []string          `json:"tiers"`
	ManifestHash string            `json:"manifestHash"`
	Capability   string            `json:"capability"`
	Federation   *FederationRef    `json:"federation"`
	Issued       int64             `json:"issued"`
}

// AssignedVolume tells a host where a replica's volume lives and, when the
// replica was moved, which committed snapshot to restore before starting.
type AssignedVolume struct {
	Name          string `json:"name"`
	VolumeID      string `json:"volumeId"`
	Mount         string `json:"mount"`
	SnapshotEvery int64  `json:"snapshotEveryMs"`
	Restore       string `json:"restore"` // committed snapshot id, "" = start empty/keep local
}

// FederationRef marks an assignment placed on behalf of a peer cluster.
type FederationRef struct {
	Peer      string `json:"peer"`      // requesting cluster name
	PeerRoot  string `json:"peerRoot"`  // requesting cluster root key
	Agreement string `json:"agreement"` // agreement digest
	Placement string `json:"placement"` // placement request digest
}

// ------------------------------------------------------------------ roster

// Roster is signed by the cluster root key (envelope kind "roster"). It is
// the only way a control-plane member key becomes trusted by hosts.
type Roster struct {
	Cluster  string   `json:"cluster"`
	Version  int64    `json:"version"`
	Root     string   `json:"root"`
	Members  []Member `json:"members"`
	Issued   int64    `json:"issued"`
	NextRoot string   `json:"nextRoot"` // set during a root rotation
}

type Member struct {
	ID         string `json:"id"`
	Pub        string `json:"pub"`
	APIAddr    string `json:"apiAddr"`
	RaftAddr   string `json:"raftAddr"`
	Delegation string `json:"delegation"` // capability: root → member
}

// ------------------------------------------------------------------ bundle

// Bundle is the signed desired-state view for one host (envelope kind
// "bundle"). StateIndex is the replicated-log index the bundle was derived
// from; hosts reject a bundle whose StateIndex is lower than one they have
// already accepted (rollback protection across leader changes).
type Bundle struct {
	Cluster      string               `json:"cluster"`
	Root         string               `json:"root"`
	Roster       *envelope.Envelope   `json:"roster"`
	Issuer       string               `json:"issuer"`
	Issued       int64                `json:"issued"`
	StateIndex   int64                `json:"stateIndex"`
	Frozen       bool                 `json:"frozen"`
	Node         BundleNode           `json:"node"`
	Assignments  []*envelope.Envelope `json:"assignments"`
	Peers        []Peer               `json:"peers"`
	Volumes      []VolumeDuty         `json:"volumes"`
	Services     []Service            `json:"services"`
	Revoked      []string             `json:"revoked"`
	RevokedKeys  []string             `json:"revokedKeys"`
	Artifacts    []ArtifactLocation   `json:"artifacts"`
	Publishers   []string             `json:"publishers"` // keys that may attest artifacts
	Attestations []*envelope.Envelope `json:"attestations"`
	LocalCA      string               `json:"localCa"` // PEM of the cluster-local CA certificate
	RootRotation *envelope.Envelope   `json:"rootRotation"`
}

type BundleNode struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Status string      `json:"status"`
	MeshIP string      `json:"meshIp"`
	Keys   []KeyRecord `json:"keys"`
	Roles  []string    `json:"roles"`
}

// KeyRecord is one valid signing key for a stable dh1 identity.
type KeyRecord struct {
	Pub     string `json:"pub"`
	From    int64  `json:"from"`
	Until   int64  `json:"until"` // 0 = no expiry
	Revoked bool   `json:"revoked"`
	Reason  string `json:"reason"`
}

type Peer struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	MeshIP  string             `json:"meshIp"`
	Status  string             `json:"status"`
	Roles   []string           `json:"roles"`
	Keys    []string           `json:"keys"`
	Binding *envelope.Envelope `json:"binding"` // wg-binding signed by the peer itself
}

// VolumeDuty tells a host which replicated volumes it must hold.
type VolumeDuty struct {
	VolumeID  string       `json:"volumeId"`
	App       string       `json:"app"`
	Name      string       `json:"name"`
	Replica   int64        `json:"replica"`
	Primary   string       `json:"primary"` // node running the workload replica
	Members   []string     `json:"members"` // nodes that must hold snapshots
	Committed *SnapshotRef `json:"committed"`
	Retain    int64        `json:"retain"`
	Keep      []string     `json:"keep"` // snapshot ids to keep (GC roots)
}

type SnapshotRef struct {
	ID     string `json:"id"`
	Root   string `json:"root"` // Merkle root over chunk ids
	Bytes  int64  `json:"bytes"`
	Chunks int64  `json:"chunks"`
}

type ArtifactLocation struct {
	Digest   string   `json:"digest"`
	Name     string   `json:"name"`
	Manifest string   `json:"manifest"` // CAS id of the canonical ordered chunk list
	Nodes    []string `json:"nodes"`
	Bytes    int64    `json:"bytes"`
}

// ArtifactManifest is stored in the CAS; its id is ArtifactLocation.Manifest.
type ArtifactManifest struct {
	Digest string   `json:"digest"` // b3 of the whole file
	Name   string   `json:"name"`
	Bytes  int64    `json:"bytes"`
	Chunks []string `json:"chunks"`
}

// Service is the edge routing table entry for one ingress host.
type Service struct {
	App       string     `json:"app"`
	Host      string     `json:"host"`
	Port      string     `json:"port"`
	TLS       string     `json:"tls"`
	Issuer    string     `json:"issuer"`
	HealthURL string     `json:"healthUrl"`
	Endpoints []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	Assignment string `json:"assignment"`
	Node       string `json:"node"`
	MeshIP     string `json:"meshIp"`
	Port       int64  `json:"port"`
	Generation int64  `json:"generation"`
	Draining   bool   `json:"draining"`
	Observed   string `json:"observed"` // last signed observation state
}

// ------------------------------------------------------------ enrollment

// Enroll is self-signed by the joining host (envelope kind "enroll").
type Enroll struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Pub        string        `json:"pub"`
	Arch       string        `json:"arch"`
	OS         string        `json:"os"`
	Tiers      []string      `json:"tiers"`
	Region     string        `json:"region"`
	Zone       string        `json:"zone"`
	Host       string        `json:"host"`
	CPUMilli   int64         `json:"cpuMilli"`
	MemBytes   int64         `json:"memBytes"`
	DiskBytes  int64         `json:"diskBytes"`
	Roles      []string      `json:"roles"`
	Features   []string      `json:"features"`
	JoinToken  string        `json:"joinToken"`
	PolicyHash string        `json:"policyHash"`
	Policy     PolicySummary `json:"policy"`
	TS         int64         `json:"ts"`
}

// PolicySummary is the host's own, host-signed statement of its local
// sovereign policy. The control plane can read it; it cannot change it.
type PolicySummary struct {
	Sovereign               bool     `json:"sovereign"`
	MaxWorkloads            int64    `json:"maxWorkloads"`
	MaxCPUMilli             int64    `json:"maxCpuMilli"`
	MaxMemBytes             int64    `json:"maxMemBytes"`
	AcceptTiers             []string `json:"acceptTiers"`
	DenyImagesWithoutDigest bool     `json:"denyImagesWithoutDigest"`
	RequireImageSignature   bool     `json:"requireImageSignature"`
	AllowRuntimes           []string `json:"allowRuntimes"`
	AllowFederated          bool     `json:"allowFederated"`
	AllowExec               bool     `json:"allowExec"`
	OfflineAdmission        string   `json:"offlineAdmission"`
	MaxClockSkewMs          int64    `json:"maxClockSkewMs"`
	StorageQuotaBytes       int64    `json:"storageQuotaBytes"`
}

// WGBinding binds a WireGuard key and endpoint to a dh1 identity (envelope
// kind "wg-binding", signed by the host's own identity key).
type WGBinding struct {
	Node     string `json:"node"`
	WGPub    string `json:"wgPub"` // base64 std (WireGuard convention)
	Endpoint string `json:"endpoint"`
	MeshIP   string `json:"meshIp"`
	TS       int64  `json:"ts"`
}

// Rotation moves a host identity to a new signing key. It is signed twice:
// the outer envelope by the new key, and OldSig by the old key over the same
// canonical payload (envelope kind "rotation").
type Rotation struct {
	Node       string `json:"node"`
	OldPub     string `json:"oldPub"`
	NewPub     string `json:"newPub"`
	NotBefore  int64  `json:"notBefore"`
	GraceUntil int64  `json:"graceUntil"`
	TS         int64  `json:"ts"`
}

// ------------------------------------------------------------ observation

// Observation is signed by the host (envelope kind "observation"). Seq is a
// per-host counter; the control plane rejects any seq it has already seen.
type Observation struct {
	Node         string        `json:"node"`
	Seq          int64         `json:"seq"`
	TS           int64         `json:"ts"`
	Mode         string        `json:"mode"` // normal | offline-hold | frozen-hold | clock-skew | ledger-corrupt | untrusted-plane
	ModeDetail   string        `json:"modeDetail"`
	BundleIndex  int64         `json:"bundleIndex"`
	BundleIssued int64         `json:"bundleIssued"`
	BundleIssuer string        `json:"bundleIssuer"`
	Workloads    []WorkloadObs `json:"workloads"`
	Mesh         *MeshObs      `json:"mesh"`
	Storage      *StorageObs   `json:"storage"`
	Edge         *EdgeObs      `json:"edge"`
	Facts        Facts         `json:"facts"`
	Ledger       LedgerHead    `json:"ledger"`
	Buffered     bool          `json:"buffered"` // queued while the control plane was unreachable
}

type WorkloadObs struct {
	Assignment  string      `json:"assignment"`
	App         string      `json:"app"`
	Replica     int64       `json:"replica"`
	Generation  int64       `json:"generation"`
	Admitted    string      `json:"admitted"` // allowed | refused | pending
	Code        string      `json:"code"`
	Reason      string      `json:"reason"`
	Checks      []Check     `json:"checks"`
	Observed    string      `json:"observed"` // running | starting | failed | stopped | exited | oom-killed | unknown
	Runtime     string      `json:"runtime"`
	PID         int64       `json:"pid"`
	ContainerID string      `json:"containerId"`
	LocalPort   int64       `json:"localPort"`
	MeshPort    int64       `json:"meshPort"`
	StartedAt   int64       `json:"startedAt"`
	Restarts    int64       `json:"restarts"`
	ExitCode    int64       `json:"exitCode"`
	Health      *HealthObs  `json:"health"`
	Volumes     []VolumeObs `json:"volumes"`
}

// Check is one admission rule evaluation, shown verbatim in the console.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type HealthObs struct {
	OK          bool   `json:"ok"`
	CheckedAt   int64  `json:"checkedAt"`
	LatencyUs   int64  `json:"latencyUs"` // measured round trip of the probe
	Detail      string `json:"detail"`
	Consecutive int64  `json:"consecutive"` // consecutive failures
}

type VolumeObs struct {
	VolumeID     string `json:"volumeId"`
	LastSnapshot string `json:"lastSnapshot"`
	Restored     string `json:"restored"`
	Bytes        int64  `json:"bytes"`
}

type MeshObs struct {
	Device     string    `json:"device"` // wireguard-go/netstack | none
	WGPub      string    `json:"wgPub"`
	ListenPort int64     `json:"listenPort"`
	MeshIP     string    `json:"meshIp"`
	Peers      []PeerObs `json:"peers"`
	Gossip     string    `json:"gossip"` // memberlist | none
	Members    int64     `json:"members"`
}

type PeerObs struct {
	Node          string `json:"node"`
	WGPub         string `json:"wgPub"`
	Endpoint      string `json:"endpoint"`
	LastHandshake int64  `json:"lastHandshake"` // unix ms, 0 = never
	RxBytes       int64  `json:"rxBytes"`
	TxBytes       int64  `json:"txBytes"`
	Gossip        string `json:"gossip"` // alive | suspect | dead | unknown
	RTTUs         int64  `json:"rttUs"`  // measured over the mesh; -1 = not measured
	RTTAt         int64  `json:"rttAt"`
	BindingOK     bool   `json:"bindingOk"`
}

type StorageObs struct {
	CapacityBytes int64  `json:"capacityBytes"`
	FreeBytes     int64  `json:"freeBytes"`
	UsedBytes     int64  `json:"usedBytes"`
	QuotaBytes    int64  `json:"quotaBytes"`
	Chunks        int64  `json:"chunks"`
	Corrupt       int64  `json:"corrupt"`
	MerkleRoot    string `json:"merkleRoot"`
}

type EdgeObs struct {
	HTTPAddr  string     `json:"httpAddr"`
	HTTPSAddr string     `json:"httpsAddr"`
	Routes    []RouteObs `json:"routes"`
	Certs     []CertObs  `json:"certs"`
	Requests  int64      `json:"requests"`
	Errors    int64      `json:"errors"`
}

type RouteObs struct {
	Host      string        `json:"host"`
	App       string        `json:"app"`
	Endpoints []EndpointObs `json:"endpoints"`
}

type EndpointObs struct {
	Assignment string `json:"assignment"`
	Node       string `json:"node"`
	State      string `json:"state"` // routing | draining | ejected | pending
	Reason     string `json:"reason"`
	InFlight   int64  `json:"inFlight"`
	Served     int64  `json:"served"`
	Failures   int64  `json:"failures"`
	LatencyUs  int64  `json:"latencyUs"`
	Since      int64  `json:"since"`
}

// CertObs states match the blueprint: REQUESTED PENDING ISSUED RENEWING
// EXPIRED REVOKED FAILED UNKNOWN.
type CertObs struct {
	Host        string   `json:"host"`
	Names       []string `json:"names"`
	State       string   `json:"state"`
	Issuer      string   `json:"issuer"`
	Serial      string   `json:"serial"`
	NotBefore   int64    `json:"notBefore"`
	NotAfter    int64    `json:"notAfter"`
	Fingerprint string   `json:"fingerprint"`
	Detail      string   `json:"detail"`
	Since       int64    `json:"since"`
}

// Facts are negative-by-default measurements: a field is true only when the
// host measured it.
//
// Every field is measured on the host when the observation is built. A value
// the host could not measure is zero or empty AND named in Unknown; declared
// capacity (what the operator offers) is in Enroll, never here.
type Facts struct {
	OS          string   `json:"os"`
	Arch        string   `json:"arch"`
	Kernel      string   `json:"kernel"`
	CPUs        int64    `json:"cpus"`     // logical CPUs visible to the agent
	MemBytes    int64    `json:"memBytes"` // physical memory measured by the OS; 0 when unmeasured
	Docker      string   `json:"docker"`   // version, or "" if unavailable
	UDP443      bool     `json:"udp443"`
	Runtimes    []string `json:"runtimes"`
	Probes      []Check  `json:"probes"`
	ClockSkewMs int64    `json:"clockSkewMs"` // host clock minus bundle issue time at receipt

	CPUModel      string   `json:"cpuModel,omitempty"`
	PhysicalCores int64    `json:"physicalCores,omitempty"`
	SwapBytes     int64    `json:"swapBytes,omitempty"`
	UptimeSec     int64    `json:"uptimeSec,omitempty"`
	Disks         []Disk   `json:"disks,omitempty"`
	DataFS        *FSInfo  `json:"dataFs,omitempty"` // filesystem holding the agent's data directory
	GPUs          []GPU    `json:"gpus,omitempty"`
	Unknown       []string `json:"unknown,omitempty"` // facts that could not be measured on this host
}

// Disk is a block device the kernel exposes (loop and RAM devices excluded).
type Disk struct {
	Name       string `json:"name"`
	SizeBytes  int64  `json:"sizeBytes"`
	Rotational bool   `json:"rotational"`
	Removable  bool   `json:"removable"`
	Model      string `json:"model,omitempty"`
}

// FSInfo describes one mounted filesystem.
type FSInfo struct {
	Path       string `json:"path"`
	TotalBytes int64  `json:"totalBytes"`
	FreeBytes  int64  `json:"freeBytes"`
}

// GPU is a display/compute device found by a named source. Fields the source
// cannot report stay empty; nothing is inferred from the vendor.
type GPU struct {
	Vendor    string `json:"vendor"`
	Model     string `json:"model,omitempty"`
	VRAMBytes int64  `json:"vramBytes,omitempty"`
	Driver    string `json:"driver,omitempty"`
	Source    string `json:"source"` // sysfs | nvidia-smi
}

type LedgerHead struct {
	Seq  int64  `json:"seq"`
	Hash string `json:"hash"`
}

// ---------------------------------------------------------- storage evidence

// ReplicaEvidence proves a host holds a complete, verified snapshot (envelope
// kind "replica-evidence").
type ReplicaEvidence struct {
	Node       string            `json:"node"`
	VolumeID   string            `json:"volumeId"`
	Snapshot   string            `json:"snapshot"`
	Root       string            `json:"root"`
	Chunks     int64             `json:"chunks"`
	Bytes      int64             `json:"bytes"`
	VerifiedAt int64             `json:"verifiedAt"`
	Missing    int64             `json:"missing"`
	Corrupt    int64             `json:"corrupt"`
	Primary    bool              `json:"primary"`
	Manifest   *SnapshotManifest `json:"manifest"` // only sent by the primary when a snapshot is created
}

// SnapshotManifest describes a volume snapshot as a tree of files whose
// contents are chunk lists. ID = "b3:" + BLAKE3 domain hash("snapshot", manifest).
type SnapshotManifest struct {
	VolumeID string         `json:"volumeId"`
	Parent   string         `json:"parent"`
	TS       int64          `json:"ts"`
	Files    []SnapshotFile `json:"files"`
}

type SnapshotFile struct {
	Path   string   `json:"path"`
	Mode   int64    `json:"mode"`
	Size   int64    `json:"size"`
	Chunks []string `json:"chunks"`
}

// RepairEvidence records one anti-entropy repair operation (envelope kind
// "repair-evidence"): every object moved, from where, what state it was in
// before, and whether the result verified.
type RepairEvidence struct {
	OpID      string       `json:"opId"`
	Node      string       `json:"node"` // destination (the repairing host)
	VolumeID  string       `json:"volumeId"`
	Snapshot  string       `json:"snapshot"`
	Trigger   string       `json:"trigger"` // anti-entropy | restore | artifact
	LocalRoot string       `json:"localRoot"`
	PeerRoot  string       `json:"peerRoot"`
	Items     []RepairItem `json:"items"`
	TS        int64        `json:"ts"`
}

type RepairItem struct {
	Object   string `json:"object"`   // chunk id
	Source   string `json:"source"`   // node the bytes came from
	Previous string `json:"previous"` // missing | corrupt
	Result   string `json:"result"`   // present | failed
	Verified bool   `json:"verified"`
	Detail   string `json:"detail"`
}

// ArtifactAttestation is signed by a publisher key (envelope kind
// "artifact-attestation"). Hosts with requireImageSignature only run
// artifacts attested by a key in their trusted publisher set.
type ArtifactAttestation struct {
	Digest string `json:"digest"`
	Name   string `json:"name"`
	TS     int64  `json:"ts"`
}

// ------------------------------------------------------------- truth model

// Basis is the truthfulness class of a displayed value.
type Basis string

const (
	Observed   Basis = "OBSERVED"
	Derived    Basis = "DERIVED"
	Configured Basis = "CONFIGURED"
	Planned    Basis = "PLANNED"
	Unknown    Basis = "UNKNOWN"
)

// Fact is a value plus where it came from.
type Fact struct {
	Value    any    `json:"value"`
	Basis    Basis  `json:"basis"`
	At       int64  `json:"at"`
	Source   string `json:"source"`
	Evidence string `json:"evidence"`
	Note     string `json:"note"`
}
