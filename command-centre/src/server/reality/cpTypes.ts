/**
 * Wire types for the parts of GET /api/v1/view the Command Centre reads.
 * Mirrors pkg/control/views.go and pkg/api/types.go; fields are optional where
 * the Go side can omit or null them.
 */

export interface CPObsFreshness {
  seq: number;
  hostTs?: number;
  receivedAt?: number;
  ageMs?: number;
  freshness?: 'FRESH' | 'STALE' | 'NONE';
  source?: string;
}

export interface CPNode {
  id: string;
  name: string;
  status: string;
  health: string;
  identity: string;
  newAdmission: string;
  existing: string;
  keys?: { pub: string; revoked?: boolean; reason?: string }[] | null;
  roles?: string[] | null;
  meshIp?: string;
  region: string;
  zone?: string;
  host?: string;
  os?: string;
  arch?: string;
  tiers?: string[] | null;
  cpuMilli?: number;
  memBytes?: number;
  policy?: Record<string, unknown> | null;
  joinedAt?: number;
  approvedAt?: number;
  revokedAt?: number;
  lastObs?: CPObsFreshness;
  facts?: {
    kernel?: string;
    cpus?: number;
    memBytes?: number;
    docker?: string;
    udp443?: boolean;
    runtimes?: string[] | null;
    probes?: { name: string; ok?: boolean; detail?: string }[] | null;
    clockSkewMs?: number;
    cpuModel?: string;
    physicalCores?: number;
    swapBytes?: number;
    disks?: { name: string; sizeBytes?: number; rotational?: boolean; removable?: boolean; model?: string }[] | null;
    gpus?: { vendor: string; model?: string; vramBytes?: number; driver?: string; source: string }[] | null;
    dataFs?: { path: string; totalBytes?: number; freeBytes?: number } | null;
    unknown?: string[] | null;
  } | null;
  mesh?: {
    device?: string;
    meshIp?: string;
    peers?: { node: string; gossip?: string; lastHandshake?: number; rttUs?: number }[] | null;
  } | null;
  storage?: { capacityBytes?: number; freeBytes?: number; usedBytes?: number; quotaBytes?: number; chunks?: number; corrupt?: number } | null;
  edge?: CPEdgeObs | null;
  ledger?: { seq: number; hash: string };
  mode?: string;
  modeDetail?: string;
  workloads?: number;
  evidence?: string;
}

export interface CPEdgeObs {
  httpAddr?: string;
  httpsAddr?: string;
  routes?: { host: string; app: string; endpoints?: { assignment: string; node: string; state: string; reason?: string; served?: number; failures?: number; latencyUs?: number }[] | null }[] | null;
  certs?: {
    host: string;
    names?: string[] | null;
    state: string;
    issuer?: string;
    serial?: string;
    notBefore?: number;
    notAfter?: number;
    fingerprint?: string;
    detail?: string;
    since?: number;
  }[] | null;
  requests?: number;
  errors?: number;
}

export interface CPReplica {
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
  status?: string;
  code?: string;
  reason?: string;
  checks?: { name: string; ok?: boolean; detail?: string }[] | null;
  health?: { ok?: boolean; checkedAt: number; latencyUs?: number; detail?: string } | null;
  restarts?: number;
  startedAt?: number;
  freshness?: string;
  observedAt?: number;
  evidence?: string;
}

export interface CPApp {
  name: string;
  generation: number;
  hash: string;
  image: string;
  runtime?: string;
  replicas?: number;
  deleted?: boolean;
  desired?: number;
  admitted?: number;
  observed?: number;
  drift?: number;
  rows?: CPReplica[] | null;
  history?: { generation: number; hash: string; ts: number; actor: string; change: string }[] | null;
  manifest?: {
    spec?: {
      resources?: { cpu?: string; mem?: string };
      env?: Record<string, string> | null;
      volumes?: { name: string }[] | null;
    };
  };
  ingress?: { host: string; port: string; tls?: string }[] | null;
  federation?: unknown;
}

export interface CPView {
  generatedAt: number;
  servedBy?: { member: string; state: string; leader: string; index?: number; stale?: boolean };
  cluster?: {
    name: string;
    root: string;
    rosterVersion?: number;
    members?: { id: string; apiAddr: string; suffrage: string; inRaft?: boolean; leader?: boolean }[] | null;
    frozen?: boolean;
    quorum?: string;
    durability?: string;
    ha?: string;
  };
  overview?: {
    hosts?: Record<string, number>;
    hostsFresh?: number;
    desired?: number;
    admitted?: number;
    observed?: number;
    refused?: number;
    drift?: number;
    apps?: number;
    volumes?: number;
    volumesDegraded?: number;
    incidents?: string[] | null;
    evidenceAgeMs?: number;
  };
  nodes?: CPNode[] | null;
  apps?: CPApp[] | null;
  volumes?: {
    id: string;
    app: string;
    name: string;
    replica: number;
    sizeBytes?: number;
    durability?: { replicas?: number; erasure?: string };
    members?: string[] | null;
    memberNames?: string[] | null;
    snapshots?: unknown[] | null;
    committedRef?: { id: string; root: string; ts: number; chunks?: number; bytes?: number } | null;
    verified?: number;
    state?: string;
    detail?: string;
  }[] | null;
  artifacts?: { digest: string; name: string; bytes?: number; chunks?: number; uploaded?: number; holderNames?: string[] | null; attested?: boolean }[] | null;
  edge?: {
    services?: { app: string; host: string; port: string; tls?: string; endpoints?: unknown[] | null }[] | null;
    edges?: { node: string; name: string; obs?: CPEdgeObs | null; freshness?: string }[] | null;
  };
  audit?: {
    head: number;
    headHash?: string;
    entries?: {
      seq: number;
      ts: number;
      actor: string;
      source: string;
      action: string;
      resource: string;
      generation?: number;
      detail?: string;
      evidence?: string;
      prev: string;
      hash: string;
    }[] | null;
    verification?: {
      ok: boolean;
      entries: number;
      head: string;
      break?: unknown;
      checkpoints: number;
      checkpointBreak?: unknown;
      verifiedAt: number;
      verifiedBy: string;
    };
  };
  chaos?: { id: string; scenario: string; verdict: string; received: number; signer: string; evidence: string }[] | null;
  diagnostics?: { subject: string; item: string; value: string; basis: string; detail?: string }[] | null;
  milestones?: { id: string; title: string; state: string; basis: string; evidence?: string[] | null; gaps?: string[] | null }[] | null;
}

/** Minimal structural check before trusting a payload as a view. */
export function isCPView(x: unknown): x is CPView {
  if (!x || typeof x !== 'object') return false;
  const v = x as Record<string, unknown>;
  return typeof v.generatedAt === 'number' && (v.nodes === null || Array.isArray(v.nodes)) && typeof v.cluster === 'object';
}
