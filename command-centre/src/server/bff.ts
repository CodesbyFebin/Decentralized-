/**
 * Command Centre BFF: authentication passthrough, validation, normalization.
 * It is an adapter and presentation boundary, not a second control plane:
 * it holds no operational state of its own.
 */
import express, { type NextFunction, type Request, type Response } from 'express';
import { randomUUID } from 'node:crypto';
import type { AppRec, Envelope, NodeOperation, Provenance, RealitySnapshot } from '../types/reality';
import { AdapterError, type PlatformAdapter, type RequestCtx } from './adapters/types';
import { deriveCapabilities } from './reality/capabilities';
import { attentionItems } from './reality/attention';
import { deriveOperations } from './reality/operations';
import { clearSessionCookie, decodeCapability, describeSession, setSessionCookie, tokenFrom } from './session';
import { answer, dismissPendingAction, rephraseWithModel, takePendingAction, type DocsIndex } from './copilot';
import type { BffConfig } from './config';
import { EvidenceStore, EvidenceStoreError } from './evidenceStore';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const NAME = /^[a-z0-9][a-z0-9-]{0,41}$/;
const NODE_ID = /^[a-z0-9]{8,64}$/;
const NODE_OPS: NodeOperation[] = ['APPROVE', 'DRAIN', 'UNDRAIN', 'REVOKE'];

declare module 'express-serve-static-core' {
  interface Request {
    requestId: string;
    startedAt: number;
  }
}

/** In-process request metrics (no external telemetry, no phone-home). */
export class BffMetrics {
  private routes = new Map<string, { count: number; errors: number; totalMs: number; maxMs: number }>();
  private backend = { count: 0, errors: 0, totalMs: 0, maxMs: 0 };
  record(route: string, ms: number, error: boolean) {
    const r = this.routes.get(route) ?? { count: 0, errors: 0, totalMs: 0, maxMs: 0 };
    r.count++;
    r.totalMs += ms;
    r.maxMs = Math.max(r.maxMs, ms);
    if (error) r.errors++;
    this.routes.set(route, r);
  }
  recordBackend(ms: number, error: boolean) {
    this.backend.count++;
    this.backend.totalMs += ms;
    this.backend.maxMs = Math.max(this.backend.maxMs, ms);
    if (error) this.backend.errors++;
  }
  toJSON() {
    const avg = (x: { count: number; totalMs: number }) => (x.count ? Math.round(x.totalMs / x.count) : 0);
    return {
      backend: { ...this.backend, avgMs: avg(this.backend) },
      routes: Object.fromEntries([...this.routes].map(([k, v]) => [k, { ...v, avgMs: avg(v) }]))
    };
  }
}

export class HttpError extends Error {
  constructor(public status: number, public code: string, message: string, public details?: Record<string, unknown>) {
    super(message);
  }
}

function sendError(res: Response, req: Request, err: unknown) {
  let status = 500;
  let code = 'INTERNAL';
  let message = 'Unexpected error in the Command Centre BFF.';
  let details: Record<string, unknown> | undefined;
  if (err instanceof HttpError || err instanceof AdapterError) {
    status = err.status;
    code = err.code;
    message = err.message;
    details = err.details;
  } else {
    console.error(`[bff] ${req.requestId} unhandled`, err);
  }
  res.status(status).json({ error: { code, message, request_id: req.requestId, ...(details ? { details } : {}) } });
}

type Handler = (req: Request, res: Response) => Promise<unknown>;

export interface BffDeps {
  adapter: PlatformAdapter;
  config: BffConfig;
  docs: DocsIndex | null;
  metrics?: BffMetrics;
  /** Signed validation records; defaults to config.evidenceDir / config.cli. */
  evidence?: EvidenceStore;
}

/** Distinct generations an app has had, oldest first. */
export function generationsOf(app: AppRec): number[] {
  return [...new Set(app.history.map((h) => h.generation))].sort((a, b) => a - b);
}

/** Deployment progress is derived from replica rows for one generation. */
export function deploymentProgress(app: AppRec, generation: number) {
  const rows = app.replicas.filter((r) => r.desiredGen === generation && r.desired === 'RUNNING');
  const desired = app.generation === generation ? app.desiredReplicas : rows.length;
  const admitted = rows.filter((r) => r.admittedGen === generation && r.admitted === 'ADMITTED').length;
  const refused = rows.filter((r) => r.admitted === 'REFUSED');
  const running = rows.filter((r) => r.observedGen === generation && r.observed === 'RUNNING' && r.freshness === 'LIVE').length;
  const failed = rows.filter((r) => ['FAILED', 'OOM-KILLED', 'CRASHED'].includes(r.observed));
  const healthy = rows.filter((r) => r.observedGen === generation && r.observed === 'RUNNING' && r.freshness === 'LIVE' && r.health?.ok).length;
  const superseded = app.generation > generation;
  type St = 'PASSED' | 'RUNNING' | 'PENDING' | 'FAILED';
  const stage = (done: boolean, started: boolean, bad = false): St => (bad ? 'FAILED' : done ? 'PASSED' : started ? 'RUNNING' : 'PENDING');
  const stages = [
    { id: 'ACCEPTED', label: 'Accepted by control plane', state: 'PASSED' as St, detail: `generation ${generation} committed` },
    { id: 'SCHEDULING', label: 'Scheduled to hosts', state: stage(rows.length >= desired && desired > 0, rows.length > 0), detail: `${rows.length}/${desired} assignment(s)` },
    { id: 'ADMISSION', label: 'Admitted by host policy', state: stage(admitted >= desired && desired > 0, admitted > 0, refused.length > 0), detail: refused.length ? `${refused.length} refused: ${refused[0].reason}` : `${admitted}/${desired} admitted` },
    { id: 'RUNTIME', label: 'Observed running', state: stage(running >= desired && desired > 0, running > 0, failed.length > 0), detail: failed.length ? `${failed.length} ${failed[0].observed}: ${failed[0].reason}` : `${running}/${desired} running (fresh)` },
    { id: 'HEALTH', label: 'Health checks passing', state: stage(healthy >= desired && desired > 0, healthy > 0), detail: `${healthy}/${desired} healthy` }
  ];
  let state: string;
  if (superseded) state = 'SUPERSEDED';
  else if (desired === 0) state = 'READY';
  else if (refused.length) state = 'VALIDATION_FAILED';
  else if (failed.length) state = 'DEPLOY_FAILED';
  else if (healthy >= desired) state = 'READY';
  else if (running >= desired) state = 'VERIFYING';
  else if (rows.length) state = 'DEPLOYING';
  else state = 'QUEUED';
  // Scaling appends history without a new generation: the first entry submitted it, the last is the latest change.
  const revs = app.history.filter((h) => h.generation === generation);
  const hist = revs[0];
  const lastRev = revs[revs.length - 1];
  return {
    id: `${app.name}@${generation}`,
    app: app.name,
    generation,
    state,
    stages,
    actor: hist?.actor ?? null,
    change: hist?.change ?? null,
    submittedAt: hist?.ts ?? null,
    revisions: revs.map((h) => ({ ts: h.ts, actor: h.actor, change: h.change, hash: h.hash })),
    updatedAt: lastRev?.ts ?? null,
    image: app.image,
    imageDigest: app.imageDigest,
    hash: lastRev?.hash ?? app.hash
  };
}

export function createBff({ adapter, config, docs, metrics = new BffMetrics(), evidence = new EvidenceStore(config.evidenceDir, config.cli) }: BffDeps) {
  const r = express.Router();
  r.use(express.json({ limit: '256kb' }));

  r.use((req, res, next) => {
    const incoming = req.header('x-request-id');
    req.requestId = incoming && UUID.test(incoming) ? incoming : randomUUID();
    req.startedAt = Date.now();
    res.setHeader('X-Request-Id', req.requestId);
    res.setHeader('Cache-Control', 'no-store');
    res.on('finish', () => metrics.record(`${req.method} ${req.route?.path ?? req.path}`, Date.now() - req.startedAt, res.statusCode >= 500));
    next();
  });

  // CSRF: browsers cannot attach this header cross-site without a CORS
  // preflight, and the BFF never answers CORS.
  r.use((req, _res, next) => {
    if (req.method !== 'GET' && req.method !== 'HEAD' && req.header('x-dh-console') !== '1') {
      return next(new HttpError(403, 'CSRF_REJECTED', 'Mutating requests must come from the console (missing X-DH-Console header).'));
    }
    next();
  });

  const ctx = (req: Request): RequestCtx => ({ token: tokenFrom(req), requestId: req.requestId });

  const wrap = (h: Handler) => async (req: Request, res: Response, next: NextFunction) => {
    try {
      await h(req, res);
    } catch (e) {
      next(e);
    }
  };

  const requireSession = (req: Request) => {
    if (config.adapter === 'demo') return;
    const s = describeSession(tokenFrom(req));
    if (!tokenFrom(req)) throw new HttpError(401, 'UNAUTHENTICATED', 'Sign in with an operator capability.');
    if (!s.authenticated) throw new HttpError(401, 'UNAUTHENTICATED', 'The capability is malformed or expired.');
  };

  const snap = async (req: Request): Promise<RealitySnapshot> => {
    requireSession(req);
    const t0 = Date.now();
    try {
      const s = await adapter.snapshot(ctx(req));
      metrics.recordBackend(Date.now() - t0, false);
      return s;
    } catch (e) {
      metrics.recordBackend(Date.now() - t0, true);
      throw e;
    }
  };

  const env = <T>(req: Request, data: T, provenance: Provenance): Envelope<T> => ({ data, provenance, request_id: req.requestId });

  const mutation = async (req: Request, fn: (c: RequestCtx) => Promise<{ ok: boolean; code: string | null; message: string; data?: unknown }>) => {
    requireSession(req);
    const result = await fn(ctx(req));
    return { ...result, request_id: req.requestId, actor: describeSession(tokenFrom(req)).actor };
  };

  /* ----------------------------------------------------------- settings */

  // Non-secret server configuration, so operators can see what the console is
  // bound to. Tokens, keys and API keys are never included.
  r.get('/settings', wrap(async (req, res) => {
    requireSession(req);
    res.json({
      data: {
        adapter: config.adapter,
        production: config.production,
        controlEndpoints: config.controlEndpoints,
        timeoutMs: config.timeoutMs,
        regionLocations: config.regionLocations ? Object.keys(config.regionLocations) : null,
        secureCookies: config.secureCookies,
        copilotModel: config.copilotModel,
        evidenceDir: config.evidenceDir,
        cliConfigured: !!config.cli,
        session: describeSession(tokenFrom(req))
      },
      request_id: req.requestId
    });
  }));

  /* ------------------------------------------------------------- health */

  r.get('/health', wrap(async (req, res) => {
    const backend = await adapter.health();
    res.json({ bff: 'ok', mode: adapter.mode, backend, request_id: req.requestId });
  }));

  /* ------------------------------------------------------------ session */

  r.get('/session', wrap(async (req, res) => {
    res.json({ data: describeSession(tokenFrom(req)), mode: adapter.mode, request_id: req.requestId });
  }));

  r.post('/session', wrap(async (req, res) => {
    const token = String(req.body?.token ?? '').trim();
    const d = decodeCapability(token);
    if (!d) throw new HttpError(422, 'VALIDATION_FAILED', 'That is not a dhcap1 capability. Mint one with `dh token` or `dh console`.');
    if (d.expiresAt !== null && d.expiresAt <= Date.now()) throw new HttpError(401, 'UNAUTHENTICATED', 'That capability has expired.');
    if (adapter.mode === 'controlplane') {
      // Let the control plane verify the signature chain before we keep it.
      await adapter.auditPage({ token, requestId: req.requestId }, Number.MAX_SAFE_INTEGER - 1, 1);
    }
    setSessionCookie(res, token, d.expiresAt, config.secureCookies);
    res.json({ data: describeSession(token), request_id: req.requestId });
  }));

  r.delete('/session', wrap(async (req, res) => {
    clearSessionCookie(res, config.secureCookies);
    res.json({ data: describeSession(null), request_id: req.requestId });
  }));

  /* ------------------------------------------------------- capabilities */

  r.get('/capabilities', wrap(async (req, res) => {
    const health = await adapter.health();
    let s: RealitySnapshot | null = null;
    let detail = health.detail;
    if (health.reachable && tokenFrom(req)) {
      try {
        s = await adapter.snapshot(ctx(req));
      } catch (e) {
        detail = e instanceof AdapterError ? e.message : detail;
      }
    } else if (health.reachable) {
      detail = 'sign in to discover capabilities';
    }
    res.json({
      data: deriveCapabilities({
        mode: adapter.mode,
        now: Date.now(),
        snapshot: s,
        reachable: health.reachable,
        detail,
        hasRegionLocations: !!config.regionLocations,
        copilotModel: !!config.copilotModel,
        validationRecords: evidence.configured ? `signed validation records from the evidence directory${evidence.canVerify ? '; verified with dh evidence verify' : '; no dh CLI configured, so they cannot be verified here'}` : null
      }),
      request_id: req.requestId
    });
  }));

  /* ----------------------------------------------------------- overview */

  r.get('/overview', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(
      env(req, {
        overview: s.overview,
        nodes: s.nodes,
        apps: s.apps.map(({ replicas, history, ...a }) => ({ ...a, replicaCount: replicas.length, lastChange: history[history.length - 1] ?? null })),
        domains: s.domains,
        certificates: s.certificates,
        security: s.security,
        verification: s.evidence.verification,
        recentEvents: s.evidence.entries.slice(0, 30),
        attention: attentionItems(s),
        volumes: s.volumes.map((v) => ({ id: v.id, app: v.app, name: v.name, state: v.state, members: v.memberNames }))
      }, s.overview.provenance)
    );
  }));

  /* -------------------------------------------------------------- nodes */

  r.get('/nodes', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(env(req, { nodes: s.nodes, metrics: s.overview.metrics, cluster: s.overview.cluster }, s.overview.provenance));
  }));

  r.get('/nodes/:id', wrap(async (req, res) => {
    const s = await snap(req);
    const id = req.params.id;
    const node = s.nodes.find((n) => n.id === id || n.name === id);
    if (!node) throw new HttpError(404, 'NOT_FOUND', `No host "${id}" in the view you can read.`);
    const replicas = s.apps.flatMap((a) => a.replicas.filter((r) => r.node === node.id).map((r) => ({ app: a.name, ...r })));
    const events = s.evidence.entries.filter((e) => e.resource.includes(node.id) || e.actor === node.id || e.resource.includes(node.name)).slice(0, 25);
    const diagnostics = s.diagnostics.filter((d) => d.subject === node.name);
    const volumes = s.volumes.filter((v) => v.members.includes(node.id));
    // Resources the control plane has assigned to this host (desired RUNNING replicas), from each app's manifest.
    const allocated = { cpuMilli: 0, memBytes: 0, replicas: 0, undeclared: 0 };
    for (const a of s.apps) {
      for (const r of a.replicas) {
        if (r.node !== node.id || r.desired !== 'RUNNING') continue;
        allocated.replicas++;
        if (a.resources.cpuMilli === null || a.resources.memBytes === null) allocated.undeclared++;
        allocated.cpuMilli += a.resources.cpuMilli ?? 0;
        allocated.memBytes += a.resources.memBytes ?? 0;
      }
    }
    res.json(env(req, { node, replicas, events, diagnostics, volumes, allocated }, { ...s.overview.provenance, observedAt: node.observation.observedAt, ageMs: node.observation.ageMs }));
  }));

  r.get('/nodes/:id/logs', wrap(async (req, res) => {
    requireSession(req);
    const assignment = typeof req.query.assignment === 'string' ? req.query.assignment : undefined;
    if (assignment && !/^[\w-]+\/r\d+$/.test(assignment)) throw new HttpError(422, 'VALIDATION_FAILED', 'assignment must look like app/r0');
    const tail = Number(req.query.tail ?? 200);
    const lines = await adapter.nodeLogs(ctx(req), req.params.id, assignment, tail);
    res.json({ data: { lines }, request_id: req.requestId });
  }));

  r.post('/nodes/:id/operations', wrap(async (req, res) => {
    const id = req.params.id;
    const type = String(req.body?.type ?? '').toUpperCase() as NodeOperation;
    if (!NODE_ID.test(id)) throw new HttpError(422, 'VALIDATION_FAILED', 'Node id is malformed.');
    if (!NODE_OPS.includes(type)) throw new HttpError(422, 'VALIDATION_FAILED', `type must be one of ${NODE_OPS.join(', ')}.`);
    if (type === 'REVOKE' && req.body?.confirm !== id) {
      throw new HttpError(428, 'CONFIRMATION_REQUIRED', 'Revoking a host identity cannot be undone. Repeat the node id in "confirm".', { expected: 'node id' });
    }
    const out = await mutation(req, (c) => adapter.nodeOperation(c, id, type, typeof req.body?.reason === 'string' ? req.body.reason.slice(0, 200) : undefined));
    res.status(out.ok ? 200 : 409).json({ data: { operation: type, node: id, ...out }, request_id: req.requestId });
  }));

  /* --------------------------------------------------------------- apps */

  r.get('/apps', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(env(req, { apps: s.apps, domains: s.domains }, s.overview.provenance));
  }));

  r.get('/apps/:name', wrap(async (req, res) => {
    const s = await snap(req);
    const app = s.apps.find((a) => a.name === req.params.name);
    if (!app) throw new HttpError(404, 'NOT_FOUND', `No application "${req.params.name}".`);
    res.json(
      env(req, {
        app,
        domains: s.domains.filter((d) => d.app === app.name),
        volumes: s.volumes.filter((v) => v.app === app.name),
        deployments: generationsOf(app).map((g) => deploymentProgress(app, g)).reverse(),
        events: s.evidence.entries.filter((e) => e.resource.includes(`/${app.name}`) || e.resource === `app/${app.name}`).slice(0, 25)
      }, s.overview.provenance)
    );
  }));

  r.post('/apps/:name/scale', wrap(async (req, res) => {
    const name = req.params.name;
    const replicas = Number(req.body?.replicas);
    if (!NAME.test(name)) throw new HttpError(422, 'VALIDATION_FAILED', 'Application name is malformed.');
    if (!Number.isInteger(replicas) || replicas < 0 || replicas > 64) throw new HttpError(422, 'VALIDATION_FAILED', 'replicas must be an integer 0..64.');
    if (replicas === 0 && req.body?.confirm !== name) throw new HttpError(428, 'CONFIRMATION_REQUIRED', 'Scaling to zero stops the application. Repeat its name in "confirm".');
    const out = await mutation(req, (c) => adapter.scaleApp(c, name, replicas));
    res.status(out.ok ? 200 : 409).json({ data: out, request_id: req.requestId });
  }));

  r.post('/apps/:name/delete', wrap(async (req, res) => {
    const name = req.params.name;
    if (!NAME.test(name)) throw new HttpError(422, 'VALIDATION_FAILED', 'Application name is malformed.');
    if (req.body?.confirm !== name) throw new HttpError(428, 'CONFIRMATION_REQUIRED', 'Deleting an application stops every replica. Repeat its name in "confirm".');
    const out = await mutation(req, (c) => adapter.deleteApp(c, name));
    res.status(out.ok ? 200 : 409).json({ data: out, request_id: req.requestId });
  }));

  /* -------------------------------------------------------- deployments */

  r.get('/deployments', wrap(async (req, res) => {
    const s = await snap(req);
    const list = s.apps
      .flatMap((a) => generationsOf(a).map((g) => deploymentProgress(a, g)))
      .sort((x, y) => (y.submittedAt ?? 0) - (x.submittedAt ?? 0));
    res.json(env(req, { deployments: list, artifacts: s.artifacts }, s.overview.provenance));
  }));

  r.get('/deployments/:app/:generation', wrap(async (req, res) => {
    const s = await snap(req);
    const app = s.apps.find((a) => a.name === req.params.app);
    const gen = Number(req.params.generation);
    if (!app || !Number.isInteger(gen) || gen < 1 || gen > app.generation) throw new HttpError(404, 'NOT_FOUND', 'No such deployment.');
    res.json(env(req, { deployment: deploymentProgress(app, gen), replicas: app.replicas.filter((r) => r.desiredGen === gen) }, s.overview.provenance));
  }));

  r.post('/deployments', wrap(async (req, res) => {
    const yaml = String(req.body?.manifest ?? '');
    if (!yaml.trim()) throw new HttpError(422, 'VALIDATION_FAILED', 'manifest (dh/v1 YAML) is required.');
    if (yaml.length > 256_000) throw new HttpError(413, 'VALIDATION_FAILED', 'manifest is too large.');
    const name = /^\s*name:\s*([a-z0-9-]+)\s*$/m.exec(yaml)?.[1] ?? null;
    const out = await mutation(req, (c) => adapter.applyManifest(c, yaml));
    // The control plane returns the committed generation; progress is then read from its view.
    let deployment = null;
    if (out.ok && name && typeof out.data === 'number') {
      const s = await adapter.snapshot(ctx(req));
      const app = s.apps.find((a) => a.name === name);
      if (app) deployment = deploymentProgress(app, out.data);
    }
    res.status(out.ok ? 202 : 409).json({ data: { ...out, app: name, deployment }, request_id: req.requestId });
  }));

  /* ------------------------------------------------------------ storage */

  r.get('/storage', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(
      env(req, {
        volumes: s.volumes,
        artifacts: s.artifacts,
        hosts: s.nodes.map((n) => ({ id: n.id, name: n.name, region: n.region, health: n.health, storage: n.storage, freshness: n.observation.freshness, location: n.location })),
        metrics: { used: s.overview.metrics.storageUsedBytes, capacity: s.overview.metrics.storageCapacityBytes, volumes: s.overview.metrics.volumes, degraded: s.overview.metrics.volumesDegraded }
      }, s.overview.provenance)
    );
  }));

  /* -------------------------------------------------- domains / security */

  r.get('/domains', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(env(req, { domains: s.domains, certificates: s.certificates }, s.overview.provenance));
  }));

  r.get('/security', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(env(req, { controls: s.security, certificates: s.certificates, verification: s.evidence.verification, cluster: s.overview.cluster }, s.overview.provenance));
  }));

  r.get('/analytics', wrap(async (req, res) => {
    const s = await snap(req);
    const edges = s.nodes.filter((n) => n.edgeObs);
    res.json(
      env(req, {
        edges: edges.map((n) => ({ id: n.id, name: n.name, requests: n.edgeObs!.requests, errors: n.edgeObs!.errors, routes: n.edgeObs!.routes, freshness: n.observation.freshness, observedAt: n.observation.observedAt })),
        replicaHealth: s.apps.flatMap((a) => a.replicas.filter((r) => r.health).map((r) => ({ app: a.name, assignment: r.assignment, node: r.nodeName, latencyUs: r.health!.latencyUs, ok: r.health!.ok, checkedAt: r.health!.checkedAt }))),
        hosts: s.nodes.map((n) => ({ name: n.name, freshness: n.observation.freshness, storage: n.storage, mesh: n.mesh, workloads: n.workloads, declared: n.declared })),
        metrics: s.overview.metrics
      }, s.overview.provenance)
    );
  }));

  r.get('/diagnostics', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(env(req, { diagnostics: s.diagnostics, milestones: s.evidence.milestones, incidents: s.overview.incidents }, s.overview.provenance));
  }));

  /* ------------------------------------------------------------ evidence */

  r.get('/evidence', wrap(async (req, res) => {
    const s = await snap(req);
    const { entries, ...rest } = s.evidence;
    res.json(env(req, { ...rest, recent: entries.slice(0, 50) }, s.overview.provenance));
  }));

  r.get('/audit', wrap(async (req, res) => {
    requireSession(req);
    const limit = Math.min(Math.max(Number(req.query.limit ?? 50) || 50, 1), 200);
    const before = req.query.before !== undefined ? Number(req.query.before) : null;
    // The control plane pages ascending from `from`; ask for the window that ends before `before`.
    const probe = await adapter.auditPage(ctx(req), Number.MAX_SAFE_INTEGER - 1, 1);
    const end = before !== null && Number.isFinite(before) ? Math.min(before - 1, probe.head) : probe.head;
    const from = Math.max(0, end - limit);
    const page = await adapter.auditPage(ctx(req), from, limit);
    const entries = page.entries.filter((e) => e.seq <= end).reverse();
    res.json({ data: { head: page.head, entries, nextBefore: from > 0 ? from + 1 : null }, request_id: req.requestId });
  }));

  r.get('/operations', wrap(async (req, res) => {
    const s = await snap(req);
    const byName = new Map(s.apps.map((a) => [a.name, a]));
    const ops = deriveOperations(
      s,
      (name, g) => (byName.has(name) ? deploymentProgress(byName.get(name)!, g) : null),
      (name) => (byName.has(name) ? generationsOf(byName.get(name)!) : [])
    );
    res.json(env(req, { operations: ops }, { ...s.overview.provenance, state: 'DERIVED' }));
  }));

  r.get('/rejections', wrap(async (req, res) => {
    const s = await snap(req);
    res.json(env(req, { rejections: s.rejections }, s.overview.provenance));
  }));

  r.post('/evidence/verify', wrap(async (req, res) => {
    requireSession(req);
    res.json({ data: await adapter.verifyAudit(ctx(req)), request_id: req.requestId });
  }));

  /* ------------------------------------------------------------- copilot */

  r.post('/copilot/query', wrap(async (req, res) => {
    requireSession(req);
    const query = String(req.body?.query ?? '').slice(0, 2000);
    if (!query.trim()) throw new HttpError(422, 'VALIDATION_FAILED', 'query is required.');
    let s: RealitySnapshot | null = null;
    let why: string | null = null;
    try {
      s = await adapter.snapshot(ctx(req));
    } catch (e) {
      if (e instanceof AdapterError && (e.code === 'UNAUTHENTICATED' || e.code === 'PERMISSION_DENIED')) throw e;
      why = e instanceof AdapterError ? e.message : 'the control plane is unavailable';
    }
    const sess = describeSession(tokenFrom(req));
    let out = answer({ query, snapshot: s, unavailableReason: why, docs, token: tokenFrom(req), canWrite: !sess.readOnly && adapter.mode === 'controlplane' });
    if (config.copilotModel === 'gemini' && out.grounded && !out.proposedAction) {
      try {
        out = await rephraseWithModel(query, out);
      } catch {
        /* keep the deterministic, cited answer */
      }
    }
    res.json({ data: out, request_id: req.requestId });
  }));

  r.post('/copilot/actions/:id/approve', wrap(async (req, res) => {
    requireSession(req);
    const token = tokenFrom(req) ?? '';
    const a = takePendingAction(req.params.id, token);
    if (a === 'NOT_FOUND') throw new HttpError(404, 'NOT_FOUND', 'No pending action with that id.');
    if (a === 'WRONG_PRINCIPAL') throw new HttpError(403, 'PERMISSION_DENIED', 'Only the operator who received the proposal can approve it.');
    if (a === 'EXPIRED') throw new HttpError(410, 'EXPIRED', 'The proposal expired; ask again.');
    if (a.confirmText && req.body?.confirm !== a.confirmText) {
      // Put it back so the operator can retry with the confirmation.
      throw new HttpError(428, 'CONFIRMATION_REQUIRED', `Type "${a.confirmText}" to approve this ${a.risk} action.`);
    }
    const out =
      a.kind === 'NODE_OPERATION'
        ? await mutation(req, (c) => adapter.nodeOperation(c, a.params.nodeId!, a.params.op!, 'approved Copilot proposal'))
        : await mutation(req, (c) => adapter.scaleApp(c, a.params.app!, a.params.replicas!));
    res.status(out.ok ? 200 : 409).json({ data: { action: a, result: out }, request_id: req.requestId });
  }));

  r.post('/copilot/actions/:id/dismiss', wrap(async (req, res) => {
    requireSession(req);
    const ok = dismissPendingAction(req.params.id, tokenFrom(req) ?? '');
    if (!ok) throw new HttpError(404, 'NOT_FOUND', 'No pending action with that id.');
    res.json({ data: { dismissed: true }, request_id: req.requestId });
  }));

  /* ------------------------------------------------ validation records */

  const store = async <T>(fn: () => Promise<T>): Promise<T> => {
    try {
      return await fn();
    } catch (e) {
      if (e instanceof EvidenceStoreError) throw new HttpError(e.status, e.code, e.message);
      throw e;
    }
  };

  r.get('/evidence/records', wrap(async (req, res) => {
    requireSession(req);
    const records = await store(() => evidence.list());
    res.json({
      data: { records, verifications: Object.fromEntries(records.map((x) => [x.id, evidence.lastVerification(x.id)])), canVerify: evidence.canVerify },
      provenance: { state: 'LIVE', source: 'evidence-dir', observedAt: Date.now(), ageMs: 0 },
      request_id: req.requestId
    });
  }));

  r.get('/evidence/records/:id', wrap(async (req, res) => {
    requireSession(req);
    const record = await store(() => evidence.get(req.params.id));
    const children = (await store(() => evidence.list())).filter((x) => x.parent === record.id).map((x) => ({ id: x.id, outcome: x.outcome }));
    res.json({ data: { record, verification: evidence.lastVerification(record.id), canVerify: evidence.canVerify, children }, request_id: req.requestId });
  }));

  r.get('/evidence/records/:id/record.json', wrap(async (req, res) => {
    requireSession(req);
    const raw = await store(() => evidence.raw(req.params.id));
    res.setHeader('Content-Type', 'application/json');
    res.setHeader('Content-Disposition', `attachment; filename="${req.params.id}-record.json"`);
    res.send(raw);
  }));

  r.get('/evidence/records/:id/steps/:step/log', wrap(async (req, res) => {
    requireSession(req);
    const log = await store(() => evidence.stepLog(req.params.id, req.params.step));
    res.json({ data: log, request_id: req.requestId });
  }));

  r.post('/evidence/records/:id/verify', wrap(async (req, res) => {
    requireSession(req);
    const v = await store(() => evidence.verify(req.params.id));
    res.json({ data: v, request_id: req.requestId });
  }));

  /* ------------------------------------------------------------ invites */

  r.get('/invites', wrap(async (req, res) => {
    requireSession(req);
    const out = await adapter.listInvites(ctx(req));
    res.json(env(req, out, { state: adapter.mode === 'demo' ? 'SIMULATED' : 'LIVE', source: adapter.source, observedAt: out.serverTime, ageMs: 0 }));
  }));

  r.post('/invites/:nonce/revoke', wrap(async (req, res) => {
    const nonce = req.params.nonce;
    if (!/^[0-9a-f]{16,128}$/.test(nonce)) throw new HttpError(422, 'VALIDATION_FAILED', 'Invite id is malformed.');
    const out = await mutation(req, (c) => adapter.revokeInvite(c, nonce));
    res.status(out.ok ? 200 : 409).json({ data: out, request_id: req.requestId });
  }));

  /* --------------------------------------------------------- operations */

  r.get('/bff/metrics', wrap(async (req, res) => {
    requireSession(req);
    res.json({ data: metrics.toJSON(), request_id: req.requestId });
  }));

  r.use((req, _res, next) => next(new HttpError(404, 'NOT_FOUND', `No API route ${req.method} ${req.path}.`)));

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  r.use((err: unknown, req: Request, res: Response, _next: NextFunction) => {
    if (err && typeof err === 'object' && (err as { type?: string }).type === 'entity.parse.failed') {
      return sendError(res, req, new HttpError(400, 'VALIDATION_FAILED', 'Request body is not valid JSON.'));
    }
    sendError(res, req, err);
  });

  return r;
}
