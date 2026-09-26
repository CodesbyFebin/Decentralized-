/**
 * Normalized operational model served by the BFF.
 *
 * Every value the Command Centre renders comes from here. Values carry the
 * truth state they were derived under; nothing in this model is invented by
 * the TypeScript layer. `null` means "not measured" and must never be shown
 * as zero.
 */

export type TruthState = 'LIVE' | 'DERIVED' | 'CONFIGURED' | 'UNAVAILABLE' | 'PLANNED' | 'SIMULATED' | 'UNKNOWN';

/** Freshness of an observation as judged by the control plane. */
export type Freshness = 'LIVE' | 'STALE' | 'UNKNOWN' | 'UNAVAILABLE';

export interface Provenance {
  state: TruthState;
  /** Where the value came from, e.g. "control-plane:/api/v1/view" or "host-observation". */
  source: string;
  observedAt: number | null;
  ageMs: number | null;
}

export interface Metric<T = number> {
  value: T | null;
  state: TruthState;
  source: string;
  detail?: string;
}

export type AdapterMode = 'controlplane' | 'demo';

export type CapabilityKey =
  | 'nodes'
  | 'nodeOperations'
  | 'nodeLogs'
  | 'applications'
  | 'deployments'
  | 'storage'
  | 'domains'
  | 'certificates'
  | 'edgeTraffic'
  | 'visitorAnalytics'
  | 'waf'
  | 'ddosTelemetry'
  | 'evidence'
  | 'audit'
  | 'federation'
  | 'depin'
  | 'billing'
  | 'teamDirectory'
  | 'copilot'
  | 'geolocation';

export interface CapabilityEntry {
  state: TruthState;
  source: string;
  detail: string;
}

export interface Capabilities {
  mode: AdapterMode;
  backend: { reachable: boolean; state: TruthState; detail: string; checkedAt: number; servedBy?: string; leader?: string };
  items: Record<CapabilityKey, CapabilityEntry>;
}

export interface Session {
  authenticated: boolean;
  /** Capability note / signer as recorded by the control plane, e.g. "operator:console". */
  actor: string | null;
  actions: string[];
  cluster: string | null;
  expiresAt: number | null;
  readOnly: boolean;
}

/* ----------------------------------------------------------------- nodes */

export type NodeHealth = 'HEALTHY' | 'DEGRADED' | 'OFFLINE' | 'UNKNOWN';
export type NodeLifecycle = 'PENDING_APPROVAL' | 'ACTIVE' | 'DRAINING' | 'REVOKED' | 'UNKNOWN';

export interface NodeRec {
  id: string;
  name: string;
  /** Raw control-plane status (ready | pending | draining | revoked | lost...). */
  status: string;
  lifecycle: NodeLifecycle;
  health: NodeHealth;
  healthReason: string;
  identity: string;
  admission: { newWork: string; existingWork: string };
  observation: { freshness: Freshness; observedAt: number | null; ageMs: number | null; seq: number; source: string; evidence: string | null };
  region: string;
  zone: string;
  host: string;
  os: string;
  arch: string;
  roles: string[];
  tiers: string[];
  isEdge: boolean;
  declared: { cpuMilli: number; memBytes: number };
  facts: {
    kernel: string;
    cpus: number;
    memBytes: number;
    runtimes: string[];
    docker: string;
    udp443: boolean;
    clockSkewMs: number;
    probes: { name: string; ok: boolean; detail: string }[];
  } | null;
  storage: { capacityBytes: number; freeBytes: number; usedBytes: number; quotaBytes: number; chunks: number; corrupt: number } | null;
  mesh: { meshIp: string; device: string; peers: number; peersAlive: number; handshakesRecent: number } | null;
  edgeObs: { requests: number; errors: number; routes: number } | null;
  workloads: number | null;
  policy: Record<string, unknown> | null;
  keys: { pub: string; revoked: boolean; reason: string }[];
  ledger: { seq: number; hash: string } | null;
  mode: string;
  modeDetail: string;
  joinedAt: number | null;
  approvedAt: number | null;
  revokedAt: number | null;
  /** Operator-configured map position for the region, if any. */
  location: { lat: number; lng: number; state: 'CONFIGURED' } | null;
}

export type NodeOperation = 'APPROVE' | 'DRAIN' | 'UNDRAIN' | 'REVOKE';

/* ------------------------------------------------------------------ apps */

export interface ReplicaRec {
  replica: number;
  assignment: string;
  node: string;
  nodeName: string;
  desired: string;
  desiredGen: number;
  admitted: string;
  admittedGen: number;
  observed: string;
  observedGen: number;
  code: string;
  reason: string;
  checks: { name: string; ok: boolean; detail: string }[];
  health: { ok: boolean; checkedAt: number; latencyUs: number; detail: string } | null;
  restarts: number;
  startedAt: number | null;
  freshness: Freshness;
  observedAt: number | null;
  evidence: string | null;
}

export type AppPhase = 'READY' | 'CONVERGING' | 'DEGRADED' | 'REFUSED' | 'STOPPED' | 'DELETED' | 'UNKNOWN';

export interface AppRec {
  name: string;
  generation: number;
  hash: string;
  image: string;
  imageDigest: string | null;
  runtime: string;
  desiredReplicas: number;
  admitted: number;
  observedRunning: number;
  healthyReplicas: number;
  drift: number;
  phase: AppPhase;
  phaseReason: string;
  deleted: boolean;
  replicas: ReplicaRec[];
  ingress: { host: string; port: string; tls: string }[];
  history: { generation: number; hash: string; ts: number; actor: string; change: string }[];
  resources: { cpu: string; mem: string };
  envNames: string[];
  volumes: string[];
  federated: boolean;
}

export interface ApplyResult {
  ok: boolean;
  code: string | null;
  message: string;
  requestId: string;
}

/* --------------------------------------------------------------- storage */

export interface VolumeRec {
  id: string;
  app: string;
  name: string;
  replica: number;
  sizeBytes: number;
  durabilityReplicas: number;
  erasure: string;
  state: string;
  detail: string;
  members: string[];
  memberNames: string[];
  verified: number;
  committed: { id: string; root: string; ts: number; chunks: number; bytes: number } | null;
  snapshots: number;
}

export interface ArtifactRec {
  digest: string;
  name: string;
  bytes: number;
  chunks: number;
  attested: boolean;
  uploaded: number | null;
  holders: string[];
}

/* ------------------------------------------------------- domains and TLS */

export interface DomainRec {
  host: string;
  app: string;
  /** Desired routing from the manifest. */
  desired: { app: string; port: string; tls: string };
  /** Observed at edge hosts. */
  routing: { state: Freshness | 'NOT ROUTED'; endpoints: number; routingEndpoints: number; detail: string };
  tls: CertRec | null;
  dns: Metric<string>;
}

export type CertState = 'PENDING' | 'VALID' | 'EXPIRING' | 'EXPIRED' | 'FAILED' | 'UNKNOWN';

export interface CertRec {
  host: string;
  names: string[];
  state: CertState;
  backendState: string;
  issuer: string;
  serial: string;
  notBefore: number | null;
  notAfter: number | null;
  fingerprint: string;
  detail: string;
  observedBy: string;
  tlsMode: string;
}

/* --------------------------------------------------------------- evidence */

export interface AuditEntryRec {
  seq: number;
  ts: number;
  actor: string;
  source: string;
  action: string;
  resource: string;
  generation: number;
  detail: string;
  evidence: string | null;
  hash: string;
  prev: string;
}

export interface LedgerVerification {
  state: 'VERIFIED' | 'INVALID' | 'UNKNOWN';
  entries: number;
  head: string;
  checkpoints: number;
  verifiedAt: number | null;
  verifiedBy: string;
  breakDetail: string | null;
}

export interface MilestoneRec {
  id: string;
  title: string;
  state: string;
  basis: string;
  evidence: string[];
  gaps: string[];
}

export interface EvidenceBundle {
  verification: LedgerVerification;
  head: number;
  entries: AuditEntryRec[];
  milestones: MilestoneRec[];
  artifacts: ArtifactRec[];
  chaos: { id: string; scenario: string; verdict: string; received: number; signer: string; evidence: string }[];
}

/* --------------------------------------------------------------- security */

export type ControlResult = 'PASS' | 'WARN' | 'FAIL' | 'UNKNOWN' | 'UNAVAILABLE';

export interface SecurityControl {
  id: string;
  title: string;
  result: ControlResult;
  detail: string;
  basis: TruthState;
}

/* --------------------------------------------------------------- overview */

export interface ClusterRec {
  name: string;
  root: string;
  quorum: string;
  ha: string;
  durability: string;
  frozen: boolean;
  members: { id: string; apiAddr: string; suffrage: string; inRaft: boolean; leader: boolean }[];
  servedBy: { member: string; state: string; leader: string; stale: boolean };
}

export interface Overview {
  generatedAt: number;
  provenance: Provenance;
  cluster: ClusterRec;
  metrics: {
    nodesTotal: Metric;
    nodesHealthy: Metric;
    nodesDegraded: Metric;
    nodesOffline: Metric;
    nodesUnknown: Metric;
    applications: Metric;
    replicasDesired: Metric;
    replicasAdmitted: Metric;
    replicasObserved: Metric;
    replicasRefused: Metric;
    drift: Metric;
    volumes: Metric;
    volumesDegraded: Metric;
    storageUsedBytes: Metric;
    storageCapacityBytes: Metric;
    domains: Metric;
    certificates: Metric;
    edgeRequests: Metric;
    edgeErrors: Metric;
    cpuDeclaredMilli: Metric;
    memDeclaredBytes: Metric;
    visitors30d: Metric;
    bandwidth: Metric;
  };
  incidents: string[];
  evidenceAgeMs: number | null;
}

/** Everything the UI needs from one control-plane view, normalized. */
export interface RealitySnapshot {
  overview: Overview;
  nodes: NodeRec[];
  apps: AppRec[];
  volumes: VolumeRec[];
  artifacts: ArtifactRec[];
  domains: DomainRec[];
  certificates: CertRec[];
  evidence: EvidenceBundle;
  security: SecurityControl[];
  diagnostics: { subject: string; item: string; value: string; basis: string; detail: string }[];
}

/* ----------------------------------------------------------------- errors */

export interface ApiErrorBody {
  error: { code: string; message: string; request_id: string; details?: Record<string, unknown> };
}

export interface Envelope<T> {
  data: T;
  provenance: Provenance;
  request_id: string;
}
