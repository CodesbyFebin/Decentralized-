/**
 * Pure mapping from the control plane's console projection (GET /api/v1/view,
 * pkg/control/views.go) to the Command Centre's normalized model.
 *
 * Rules:
 *  - never invent a value: missing observations become null / UNKNOWN;
 *  - health and freshness follow what the control plane computed (its fresh
 *    window and LostAfter threshold), not UI constants;
 *  - desired, admitted and observed stay separate.
 */
import type {
  AppPhase,
  AppRec,
  ArtifactRec,
  AuditEntryRec,
  CertRec,
  CertState,
  ClusterRec,
  ControlResult,
  DomainRec,
  EvidenceBundle,
  Freshness,
  Metric,
  NodeHealth,
  NodeLifecycle,
  NodeRec,
  Overview,
  RealitySnapshot,
  ReplicaRec,
  SecurityControl,
  TruthState,
  VolumeRec
} from '../../types/reality';
import type { CPView, CPNode, CPApp } from './cpTypes';

export interface MapOptions {
  now: number;
  /** LIVE for a real control plane, SIMULATED for the demo replay. */
  state: Extract<TruthState, 'LIVE' | 'SIMULATED'>;
  source: string;
  /** Operator-configured map positions keyed by node region. */
  regionLocations?: Record<string, [number, number]>;
  /** Transport facts the BFF knows about its control-plane connection. */
  transport?: { apiTls: boolean; unauthenticatedRejected: boolean | null };
  /** Certificates are EXPIRING when fewer than this many ms remain. */
  expiringWithinMs?: number;
}

const DAY = 86_400_000;

export function mapFreshness(f: string | undefined | null): Freshness {
  if (f === 'FRESH') return 'LIVE';
  if (f === 'STALE') return 'STALE';
  return 'UNKNOWN';
}

/** Node health mirrors the control plane: lost > stale > fresh; no observation is UNKNOWN. */
export function deriveNodeHealth(n: Pick<CPNode, 'status' | 'health' | 'lastObs'>): { health: NodeHealth; reason: string } {
  const fresh = n.lastObs?.freshness;
  if (n.status === 'revoked') return { health: 'OFFLINE', reason: 'identity revoked' };
  if (n.health === 'lost') return { health: 'OFFLINE', reason: 'control plane marked the host lost (no signed observation within LostAfter)' };
  if (!fresh || fresh === 'NONE') return { health: 'UNKNOWN', reason: 'no signed observation received yet' };
  if (fresh === 'STALE') return { health: 'DEGRADED', reason: `last observation ${Math.round((n.lastObs?.ageMs ?? 0) / 1000)}s old (outside the fresh window)` };
  return { health: 'HEALTHY', reason: 'fresh signed observation' };
}

export function deriveLifecycle(status: string): NodeLifecycle {
  switch (status) {
    case 'pending':
      return 'PENDING_APPROVAL';
    case 'draining':
      return 'DRAINING';
    case 'revoked':
      return 'REVOKED';
    case 'ready':
    case 'lost':
      return 'ACTIVE';
    default:
      return 'UNKNOWN';
  }
}

export function mapCertState(backend: string, notAfter: number | null, now: number, expiringWithinMs: number): CertState {
  switch (backend) {
    case 'FAILED':
      return 'FAILED';
    case 'EXPIRED':
      return 'EXPIRED';
    case 'REQUESTED':
    case 'PENDING':
      return 'PENDING';
    case 'ISSUED':
    case 'RENEWING':
      if (notAfter === null) return 'UNKNOWN';
      if (notAfter <= now) return 'EXPIRED';
      if (notAfter - now < expiringWithinMs) return 'EXPIRING';
      return 'VALID';
    default:
      return 'UNKNOWN';
  }
}

function digestOf(image: string): string | null {
  const at = image.indexOf('@');
  return at >= 0 ? image.slice(at + 1) : null;
}

function nz(v: number | undefined | null): number | null {
  return v ? v : null;
}

function mapNode(n: CPNode, opts: MapOptions): NodeRec {
  const { health, reason } = deriveNodeHealth(n);
  const obs = n.lastObs;
  const peers = n.mesh?.peers ?? [];
  const recent = opts.now - 3 * 60_000;
  const loc = opts.regionLocations?.[n.region];
  const edge = n.edge;
  return {
    id: n.id,
    name: n.name,
    status: n.status,
    lifecycle: deriveLifecycle(n.status),
    health,
    healthReason: reason,
    identity: n.identity,
    admission: { newWork: n.newAdmission, existingWork: n.existing },
    observation: {
      freshness: mapFreshness(obs?.freshness),
      observedAt: nz(obs?.receivedAt),
      ageMs: obs && obs.freshness !== 'NONE' ? obs.ageMs ?? null : null,
      seq: obs?.seq ?? 0,
      source: obs?.source ? `host-observation (${obs.source})` : 'host-observation',
      evidence: n.evidence || null
    },
    region: n.region ?? '',
    zone: n.zone ?? '',
    host: n.host ?? '',
    os: n.os ?? '',
    arch: n.arch ?? '',
    roles: n.roles ?? [],
    tiers: n.tiers ?? [],
    isEdge: (n.roles ?? []).includes('edge') || !!edge,
    declared: { cpuMilli: n.cpuMilli ?? 0, memBytes: n.memBytes ?? 0 },
    facts: n.facts
      ? {
          kernel: n.facts.kernel ?? '',
          cpus: n.facts.cpus ?? 0,
          memBytes: n.facts.memBytes ?? 0,
          runtimes: n.facts.runtimes ?? [],
          docker: n.facts.docker ?? '',
          udp443: !!n.facts.udp443,
          clockSkewMs: n.facts.clockSkewMs ?? 0,
          probes: (n.facts.probes ?? []).map((p) => ({ name: p.name, ok: !!p.ok, detail: p.detail ?? '' }))
        }
      : null,
    storage: n.storage
      ? {
          capacityBytes: n.storage.capacityBytes ?? 0,
          freeBytes: n.storage.freeBytes ?? 0,
          usedBytes: n.storage.usedBytes ?? 0,
          quotaBytes: n.storage.quotaBytes ?? 0,
          chunks: n.storage.chunks ?? 0,
          corrupt: n.storage.corrupt ?? 0
        }
      : null,
    mesh: n.mesh
      ? {
          meshIp: n.mesh.meshIp ?? n.meshIp ?? '',
          device: n.mesh.device ?? '',
          peers: peers.length,
          peersAlive: peers.filter((p) => p.gossip === 'alive').length,
          handshakesRecent: peers.filter((p) => (p.lastHandshake ?? 0) >= recent).length
        }
      : null,
    edgeObs: edge ? { requests: edge.requests ?? 0, errors: edge.errors ?? 0, routes: (edge.routes ?? []).length } : null,
    workloads: obs && obs.freshness !== 'NONE' ? n.workloads ?? 0 : null,
    policy: n.policy ?? null,
    keys: (n.keys ?? []).map((k) => ({ pub: k.pub, revoked: !!k.revoked, reason: k.reason ?? '' })),
    ledger: n.ledger?.hash ? { seq: n.ledger.seq, hash: n.ledger.hash } : null,
    mode: n.mode ?? '',
    modeDetail: n.modeDetail ?? '',
    joinedAt: nz(n.joinedAt),
    approvedAt: nz(n.approvedAt),
    revokedAt: nz(n.revokedAt),
    location: loc ? { lat: loc[0], lng: loc[1], state: 'CONFIGURED' } : null
  };
}

export function deriveAppPhase(a: Pick<AppRec, 'deleted' | 'desiredReplicas' | 'admitted' | 'observedRunning' | 'healthyReplicas'> & { refused: number; unknown: number }): { phase: AppPhase; reason: string } {
  if (a.deleted) return { phase: 'DELETED', reason: 'application deleted' };
  if (a.desiredReplicas === 0) return { phase: 'STOPPED', reason: 'scaled to 0 replicas' };
  if (a.refused > 0) return { phase: 'REFUSED', reason: `${a.refused} replica(s) refused by host policy` };
  if (a.unknown === a.desiredReplicas) return { phase: 'UNKNOWN', reason: 'no replica has been observed yet' };
  if (a.observedRunning >= a.desiredReplicas && a.healthyReplicas >= a.desiredReplicas) return { phase: 'READY', reason: 'all desired replicas observed running and passing health checks' };
  if (a.admitted < a.desiredReplicas || a.observedRunning < a.desiredReplicas) return { phase: 'CONVERGING', reason: `${a.observedRunning}/${a.desiredReplicas} observed running` };
  return { phase: 'DEGRADED', reason: `${a.healthyReplicas}/${a.desiredReplicas} replicas passing health checks` };
}

function mapApp(a: CPApp): AppRec {
  const replicas: ReplicaRec[] = (a.rows ?? []).map((r) => ({
    replica: r.replica,
    assignment: r.assignment,
    node: r.node,
    nodeName: r.nodeName,
    desired: r.desired,
    desiredGen: r.desiredGen,
    admitted: r.admitted,
    admittedGen: r.admittedGen,
    observed: r.observed,
    observedGen: r.observedGen,
    code: r.code ?? '',
    reason: r.reason ?? '',
    checks: (r.checks ?? []).map((c) => ({ name: c.name, ok: !!c.ok, detail: c.detail ?? '' })),
    health: r.health ? { ok: !!r.health.ok, checkedAt: r.health.checkedAt, latencyUs: r.health.latencyUs ?? 0, detail: r.health.detail ?? '' } : null,
    restarts: r.restarts ?? 0,
    startedAt: nz(r.startedAt),
    freshness: mapFreshness(r.freshness),
    observedAt: nz(r.observedAt),
    evidence: r.evidence || null
  }));
  // Only replicas the control plane currently wants running count, and only
  // when their observation is fresh: a stale RUNNING is a last-known state,
  // not proof (a lost host keeps its old row with desired STOPPED).
  const wanted = replicas.filter((r) => r.desired === 'RUNNING');
  const observedRunning = wanted.filter((r) => r.observed === 'RUNNING' && r.freshness === 'LIVE').length;
  const healthyReplicas = wanted.filter((r) => r.observed === 'RUNNING' && r.freshness === 'LIVE' && (r.health ? r.health.ok : false)).length;
  const refused = wanted.filter((r) => r.admitted === 'REFUSED').length;
  const unknown = wanted.filter((r) => r.observed === 'UNKNOWN' || r.freshness === 'UNKNOWN').length;
  const spec = a.manifest?.spec ?? {};
  const base = {
    deleted: !!a.deleted,
    desiredReplicas: a.replicas ?? 0,
    admitted: a.admitted ?? 0,
    observedRunning,
    healthyReplicas
  };
  const { phase, reason } = deriveAppPhase({ ...base, refused, unknown });
  return {
    name: a.name,
    generation: a.generation,
    hash: a.hash,
    image: a.image,
    imageDigest: digestOf(a.image ?? ''),
    runtime: a.runtime || 'process',
    ...base,
    drift: a.drift ?? 0,
    phase,
    phaseReason: reason,
    replicas,
    ingress: (a.ingress ?? []).map((i) => ({ host: i.host, port: i.port, tls: i.tls ?? '' })),
    history: a.history ?? [],
    resources: { cpu: spec.resources?.cpu ?? '', mem: spec.resources?.mem ?? '' },
    // Only variable NAMES leave the BFF; values can hold secrets.
    envNames: Object.keys(spec.env ?? {}),
    volumes: (spec.volumes ?? []).map((v) => v.name),
    federated: !!a.federation
  };
}

function metric(value: number | null, state: TruthState, source: string, detail?: string): Metric {
  return { value, state, source, detail };
}

export function mapView(v: CPView, opts: MapOptions): RealitySnapshot {
  const now = opts.now;
  const live = opts.state; // LIVE or SIMULATED
  const derived: TruthState = opts.state === 'SIMULATED' ? 'SIMULATED' : 'DERIVED';
  const expiring = opts.expiringWithinMs ?? 14 * DAY;
  const src = opts.source;

  const nodes = (v.nodes ?? []).map((n) => mapNode(n, opts));
  const apps = (v.apps ?? []).filter((a) => !a.deleted).map(mapApp);

  const volumes: VolumeRec[] = (v.volumes ?? []).map((x) => {
    const committed = x.committedRef;
    return {
      id: x.id,
      app: x.app,
      name: x.name,
      replica: x.replica,
      sizeBytes: x.sizeBytes ?? 0,
      durabilityReplicas: x.durability?.replicas ?? 0,
      erasure: x.durability?.erasure ?? 'none',
      state: x.state ?? 'UNKNOWN',
      detail: x.detail ?? '',
      members: x.members ?? [],
      memberNames: x.memberNames ?? [],
      verified: x.verified ?? 0,
      committed: committed ? { id: committed.id, root: committed.root, ts: committed.ts, chunks: committed.chunks ?? 0, bytes: committed.bytes ?? 0 } : null,
      snapshots: (x.snapshots ?? []).length
    };
  });

  const artifacts: ArtifactRec[] = (v.artifacts ?? []).map((a) => ({
    digest: a.digest,
    name: a.name,
    bytes: a.bytes ?? 0,
    chunks: a.chunks ?? 0,
    attested: !!a.attested,
    uploaded: nz(a.uploaded),
    holders: a.holderNames ?? []
  }));

  // Certificates are observed by edge hosts; keep the freshest per host name.
  const certificates: CertRec[] = [];
  const certByHost = new Map<string, CertRec>();
  const tlsModeByHost = new Map<string, string>();
  for (const s of v.edge?.services ?? []) tlsModeByHost.set(s.host, s.tls ?? '');
  for (const e of v.edge?.edges ?? []) {
    for (const c of e.obs?.certs ?? []) {
      const rec: CertRec = {
        host: c.host,
        names: c.names ?? [],
        state: mapCertState(c.state, nz(c.notAfter), now, expiring),
        backendState: c.state,
        issuer: c.issuer ?? '',
        serial: c.serial ?? '',
        notBefore: nz(c.notBefore),
        notAfter: nz(c.notAfter),
        fingerprint: c.fingerprint ?? '',
        detail: c.detail ?? '',
        observedBy: e.name,
        tlsMode: tlsModeByHost.get(c.host) ?? ''
      };
      const prev = certByHost.get(c.host);
      if (!prev || (rec.notAfter ?? 0) > (prev.notAfter ?? 0)) certByHost.set(c.host, rec);
    }
  }
  certificates.push(...certByHost.values());

  const domains: DomainRec[] = (v.edge?.services ?? []).map((s) => {
    const routes = (v.edge?.edges ?? []).flatMap((e) => (e.obs?.routes ?? []).filter((r) => r.host === s.host).map((r) => ({ r, fresh: e.freshness })));
    const eps = routes.flatMap((x) => x.r.endpoints ?? []);
    const routing = eps.filter((ep) => ep.state === 'routing').length;
    const anyFresh = routes.some((x) => x.fresh === 'FRESH');
    return {
      host: s.host,
      app: s.app,
      desired: { app: s.app, port: s.port, tls: s.tls ?? '' },
      routing: {
        state: routes.length === 0 ? 'NOT ROUTED' : anyFresh ? 'LIVE' : 'STALE',
        endpoints: (s.endpoints ?? []).length,
        routingEndpoints: routing,
        detail: routes.length === 0 ? 'no edge host reports a route for this name' : `${routing} endpoint(s) routing at ${routes.length} edge route table(s)`
      },
      tls: certByHost.get(s.host) ?? null,
      dns: { value: null, state: 'UNAVAILABLE', source: 'none', detail: 'The control plane does not resolve public DNS; point the name at an edge host yourself.' }
    };
  });

  const audit = v.audit;
  const ver = audit?.verification;
  const entries: AuditEntryRec[] = (audit?.entries ?? []).map((e) => ({
    seq: e.seq,
    ts: e.ts,
    actor: e.actor,
    source: e.source,
    action: e.action,
    resource: e.resource,
    generation: e.generation ?? 0,
    detail: e.detail ?? '',
    evidence: e.evidence || null,
    hash: e.hash,
    prev: e.prev
  }));
  const evidence: EvidenceBundle = {
    verification: {
      state: !ver ? 'UNKNOWN' : ver.ok ? 'VERIFIED' : 'INVALID',
      entries: ver?.entries ?? 0,
      head: ver?.head ?? '',
      checkpoints: ver?.checkpoints ?? 0,
      verifiedAt: nz(ver?.verifiedAt),
      verifiedBy: ver?.verifiedBy ?? '',
      breakDetail: ver?.break ? JSON.stringify(ver.break) : ver?.checkpointBreak ? JSON.stringify(ver.checkpointBreak) : null
    },
    head: audit?.head ?? 0,
    entries: entries.reverse(),
    milestones: (v.milestones ?? []).map((m) => ({ id: m.id, title: m.title, state: m.state, basis: m.basis, evidence: m.evidence ?? [], gaps: m.gaps ?? [] })),
    artifacts,
    chaos: (v.chaos ?? []).map((c) => ({ id: c.id, scenario: c.scenario, verdict: c.verdict, received: c.received, signer: c.signer, evidence: c.evidence }))
  };

  const count = (h: NodeHealth) => nodes.filter((n) => n.health === h).length;
  const observedNodes = nodes.filter((n) => n.storage);
  const edgeNodes = nodes.filter((n) => n.edgeObs);
  const ov: NonNullable<CPView['overview']> = v.overview ?? {};
  const cluster: ClusterRec = {
    name: v.cluster?.name ?? '',
    root: v.cluster?.root ?? '',
    quorum: v.cluster?.quorum ?? 'UNKNOWN',
    ha: v.cluster?.ha ?? 'UNKNOWN',
    durability: v.cluster?.durability ?? 'UNKNOWN',
    frozen: !!v.cluster?.frozen,
    members: (v.cluster?.members ?? []).map((m) => ({ id: m.id, apiAddr: m.apiAddr, suffrage: m.suffrage, inRaft: !!m.inRaft, leader: !!m.leader })),
    servedBy: { member: v.servedBy?.member ?? '', state: v.servedBy?.state ?? '', leader: v.servedBy?.leader ?? '', stale: !!v.servedBy?.stale }
  };

  const unavailable = (detail: string): Metric => metric(null, 'UNAVAILABLE', 'none', detail);
  const overview: Overview = {
    generatedAt: v.generatedAt,
    provenance: { state: live, source: src, observedAt: v.generatedAt, ageMs: now - v.generatedAt },
    cluster,
    metrics: {
      nodesTotal: metric(nodes.length, derived, 'nodes'),
      nodesHealthy: metric(count('HEALTHY'), derived, 'nodes.lastObs'),
      nodesDegraded: metric(count('DEGRADED'), derived, 'nodes.lastObs'),
      nodesOffline: metric(count('OFFLINE'), derived, 'nodes.health'),
      nodesUnknown: metric(count('UNKNOWN'), derived, 'nodes.lastObs'),
      applications: metric(apps.length, derived, 'apps'),
      replicasDesired: metric(ov.desired ?? null, live, 'overview.desired'),
      replicasAdmitted: metric(ov.admitted ?? null, live, 'overview.admitted'),
      replicasObserved: metric(ov.observed ?? null, live, 'overview.observed'),
      replicasRefused: metric(ov.refused ?? null, live, 'overview.refused'),
      drift: metric(ov.drift ?? null, live, 'overview.drift'),
      volumes: metric(volumes.length, derived, 'volumes'),
      volumesDegraded: metric(ov.volumesDegraded ?? null, live, 'overview.volumesDegraded'),
      storageUsedBytes: observedNodes.length
        ? metric(observedNodes.reduce((a, n) => a + (n.storage?.usedBytes ?? 0), 0), derived, 'nodes.storage', `sum over ${observedNodes.length} observing host(s)`)
        : unavailable('no host reports storage'),
      storageCapacityBytes: observedNodes.length
        ? metric(observedNodes.reduce((a, n) => a + (n.storage?.quotaBytes ?? 0), 0), derived, 'nodes.storage.quotaBytes', 'sum of host storage quotas')
        : unavailable('no host reports storage'),
      domains: metric(domains.length, derived, 'edge.services'),
      certificates: edgeNodes.length ? metric(certificates.length, derived, 'edge.certs') : unavailable('no edge host is reporting'),
      edgeRequests: edgeNodes.length
        ? metric(edgeNodes.reduce((a, n) => a + (n.edgeObs?.requests ?? 0), 0), derived, 'edge.requests', 'requests since each edge process started')
        : unavailable('no edge host is reporting'),
      edgeErrors: edgeNodes.length ? metric(edgeNodes.reduce((a, n) => a + (n.edgeObs?.errors ?? 0), 0), derived, 'edge.errors') : unavailable('no edge host is reporting'),
      cpuDeclaredMilli: metric(nodes.reduce((a, n) => a + n.declared.cpuMilli, 0), 'CONFIGURED', 'nodes.enroll.cpuMilli', 'capacity declared at enrolment'),
      memDeclaredBytes: metric(nodes.reduce((a, n) => a + n.declared.memBytes, 0), 'CONFIGURED', 'nodes.enroll.memBytes', 'capacity declared at enrolment'),
      visitors30d: unavailable('no visitor analytics subsystem exists'),
      bandwidth: unavailable('edge hosts do not report byte counters')
    },
    incidents: ov.incidents ?? [],
    evidenceAgeMs: ov.evidenceAgeMs ?? null
  };

  return {
    overview,
    nodes,
    apps,
    volumes,
    artifacts,
    domains,
    certificates,
    evidence,
    security: deriveSecurity(nodes, certificates, cluster, evidence, opts, derived),
    diagnostics: (v.diagnostics ?? []).map((d) => ({ subject: d.subject, item: d.item, value: d.value, basis: d.basis, detail: d.detail ?? '' }))
  };
}

/** `none` is the result when no host satisfies the control: FAIL for invariants, WARN for recommendations. */
function allOrNone<T>(xs: T[], pred: (x: T) => boolean, none: ControlResult = 'FAIL'): ControlResult {
  if (xs.length === 0) return 'UNKNOWN';
  const n = xs.filter(pred).length;
  return n === xs.length ? 'PASS' : n === 0 ? none : 'WARN';
}

export function deriveSecurity(
  nodes: NodeRec[],
  certs: CertRec[],
  cluster: ClusterRec,
  ev: EvidenceBundle,
  opts: MapOptions,
  basis: TruthState
): SecurityControl[] {
  const withPolicy = nodes.filter((n) => n.policy && n.lifecycle !== 'REVOKED');
  const pol = (k: string) => (n: NodeRec) => (n.policy as Record<string, unknown>)[k];
  const voters = cluster.members.filter((m) => m.suffrage === 'Voter').length;
  const certResult: ControlResult =
    certs.length === 0 ? 'UNKNOWN' : certs.some((c) => c.state === 'FAILED' || c.state === 'EXPIRED') ? 'FAIL' : certs.some((c) => c.state !== 'VALID') ? 'WARN' : 'PASS';
  const skews = nodes.filter((n) => n.facts);
  const t = opts.transport;
  const controls: SecurityControl[] = [
    {
      id: 'audit-chain',
      title: 'Audit ledger integrity',
      result: ev.verification.state === 'VERIFIED' ? 'PASS' : ev.verification.state === 'INVALID' ? 'FAIL' : 'UNKNOWN',
      detail: ev.verification.state === 'VERIFIED' ? `${ev.verification.entries} entries hash-chained and verified by ${ev.verification.verifiedBy.slice(0, 12)}` : ev.verification.breakDetail ?? 'no verification result',
      basis
    },
    {
      id: 'authn',
      title: 'Operator authentication',
      result: t?.unauthenticatedRejected === true ? 'PASS' : t?.unauthenticatedRejected === false ? 'FAIL' : 'UNKNOWN',
      detail:
        t?.unauthenticatedRejected === true
          ? 'control plane rejected an unauthenticated probe; every operator route needs a root-anchored capability'
          : t?.unauthenticatedRejected === false
            ? 'control plane answered an unauthenticated probe'
            : 'not probed',
      basis: t?.unauthenticatedRejected == null ? 'UNKNOWN' : basis
    },
    {
      id: 'api-tls',
      title: 'Control-plane API transport',
      result: t ? (t.apiTls ? 'PASS' : 'WARN') : 'UNKNOWN',
      detail: t ? (t.apiTls ? 'HTTPS with the cluster CA' : 'plain HTTP (dev cluster without --tls); do not expose this outside loopback') : 'unknown',
      basis: 'CONFIGURED'
    },
    {
      id: 'ha',
      title: 'Control-plane fault tolerance',
      result: voters >= 3 ? 'PASS' : voters > 0 ? 'WARN' : 'UNKNOWN',
      detail: cluster.ha,
      basis
    },
    { id: 'digest', title: 'Image digest pinning', result: allOrNone(withPolicy, (n) => !!pol('denyImagesWithoutDigest')(n)), detail: 'host policy denyImagesWithoutDigest', basis },
    { id: 'signature', title: 'Artifact signature required', result: allOrNone(withPolicy, (n) => !!pol('requireImageSignature')(n), 'WARN'), detail: 'host policy requireImageSignature (recommended; artifacts can still be attested by the root)', basis },
    { id: 'exec', title: 'Remote exec disabled', result: allOrNone(withPolicy, (n) => !pol('allowExec')(n)), detail: 'host policy allowExec', basis },
    {
      id: 'clock',
      title: 'Host clock skew within policy',
      result: allOrNone(skews, (n) => Math.abs(n.facts!.clockSkewMs) < Number((n.policy as Record<string, unknown> | null)?.maxClockSkewMs ?? 30000)),
      detail: skews.length ? `max |skew| ${Math.max(...skews.map((n) => Math.abs(n.facts!.clockSkewMs)))}ms` : 'no host facts',
      basis
    },
    { id: 'certs', title: 'TLS certificates', result: certResult, detail: certs.length ? `${certs.filter((c) => c.state === 'VALID').length}/${certs.length} valid (observed at edge)` : 'no certificate observations', basis },
    { id: 'frozen', title: 'Control plane not frozen', result: cluster.frozen ? 'WARN' : 'PASS', detail: cluster.frozen ? 'writes are frozen' : 'accepting signed changes', basis },
    { id: 'waf', title: 'Web application firewall', result: 'UNAVAILABLE', detail: 'no WAF exists in the edge', basis: 'UNAVAILABLE' },
    { id: 'ddos', title: 'DDoS telemetry', result: 'UNAVAILABLE', detail: 'no DDoS telemetry feed exists', basis: 'UNAVAILABLE' }
  ];
  return controls;
}
