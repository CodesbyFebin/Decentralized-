/**
 * Production adapter: the Decentralized.Host control plane (dh-control).
 *
 * - forwards the caller's capability as the bearer token; the control plane
 *   verifies it against the cluster root and enforces the action
 *   (api.read / api.write / api.admin) and records the actor;
 * - never falls back to other data when the backend fails;
 * - tries the configured members in order so a single member outage does not
 *   blind the console (followers relay the leader's view or flag staleness).
 */
import { createHash } from 'node:crypto';
import type { AuditEntryRec, LedgerVerification, NodeOperation, RealitySnapshot } from '../../types/reality';
import { mapView } from '../reality/mapView';
import { isCPView } from '../reality/cpTypes';
import { AdapterError, type BackendHealth, type OperationResult, type PlatformAdapter, type RequestCtx } from './types';

export interface ControlPlaneConfig {
  endpoints: string[];
  timeoutMs: number;
  regionLocations?: Record<string, [number, number]>;
  /** Optional CA bundle is handled by NODE_EXTRA_CA_CERTS; nothing to do here. */
}

const PATHLIKE = /(\/[\w.-]+){2,}/g;

/** Backend messages go to the browser only for client errors and with paths stripped. */
export function sanitizeMessage(msg: string): string {
  return msg.replace(PATHLIKE, '<path>').slice(0, 400);
}

export class ControlPlaneAdapter implements PlatformAdapter {
  readonly mode = 'controlplane' as const;
  readonly source: string;
  private inflight = new Map<string, { at: number; p: Promise<RealitySnapshot> }>();
  private authProbe: { at: number; value: boolean | null } | null = null;

  constructor(private cfg: ControlPlaneConfig) {
    if (!cfg.endpoints.length) throw new Error('DH_CONTROL_URL must list at least one control-plane endpoint');
    this.source = `control-plane:${cfg.endpoints[0]}`;
  }

  private get apiTls(): boolean {
    return this.cfg.endpoints.every((e) => e.startsWith('https://'));
  }

  /** Set when every member failed at the transport level; calls fail fast until it expires. */
  private down: { until: number; err: AdapterError } | null = null;

  /**
   * One HTTP exchange with failover across members on transport errors only.
   * The whole exchange shares one deadline so a frozen member (accepts TCP,
   * never answers) cannot multiply the wait by the number of members.
   */
  private async call(path: string, init: { method?: string; token?: string | null; body?: string; contentType?: string; requestId: string; signal?: AbortSignal; timeoutMs?: number }): Promise<{ status: number; json: unknown; text: string }> {
    if (this.down && Date.now() < this.down.until) throw this.down.err;
    const deadline = Date.now() + (init.timeoutMs ?? this.cfg.timeoutMs);
    let lastErr: unknown = null;
    const eps = this.cfg.endpoints;
    for (let i = 0; i < eps.length; i++) {
      const base = eps[i];
      const remaining = deadline - Date.now();
      if (remaining <= 0) break;
      const budget = Math.max(Math.min(1000, remaining), Math.floor(remaining / (eps.length - i)));
      const ac = new AbortController();
      const timer = setTimeout(() => ac.abort(new Error('timeout')), budget);
      const onAbort = () => ac.abort(init.signal?.reason);
      init.signal?.addEventListener('abort', onAbort, { once: true });
      try {
        const headers: Record<string, string> = { 'X-Request-Id': init.requestId, Accept: 'application/json' };
        if (init.token) headers.Authorization = `Bearer ${init.token}`;
        if (init.contentType) headers['Content-Type'] = init.contentType;
        const res = await fetch(base + path, { method: init.method ?? 'GET', headers, body: init.body, signal: ac.signal });
        const text = await res.text();
        let json: unknown = null;
        try {
          json = text ? JSON.parse(text) : null;
        } catch {
          json = null;
        }
        this.down = null;
        return { status: res.status, json, text };
      } catch (err) {
        lastErr = err;
        if (init.signal?.aborted) break;
      } finally {
        clearTimeout(timer);
        init.signal?.removeEventListener('abort', onAbort);
      }
    }
    if (init.signal?.aborted) throw new AdapterError('BACKEND_UNAVAILABLE', 'Request cancelled.', 499);
    const timedOut = lastErr === null || (lastErr instanceof Error && (lastErr.message === 'timeout' || (lastErr as { cause?: Error }).cause?.message === 'timeout' || lastErr.name === 'TimeoutError' || lastErr.name === 'AbortError'));
    const err = timedOut
      ? new AdapterError('BACKEND_TIMEOUT', `No control-plane member answered within ${init.timeoutMs ?? this.cfg.timeoutMs}ms.`, 504, { endpoints: eps.length })
      : new AdapterError('BACKEND_UNAVAILABLE', 'Control plane is unreachable.', 503, { endpoints: eps.length });
    // Report unavailability quickly for a short window instead of re-waiting on every call.
    this.down = { until: Date.now() + 3000, err };
    throw err;
  }

  /** Map a control-plane HTTP answer to data or a normalized error. */
  private expectOk(r: { status: number; json: unknown }): unknown {
    const body = (r.json ?? {}) as { message?: string; code?: string; ok?: boolean };
    const msg = typeof body.message === 'string' ? body.message : '';
    if (r.status === 401) {
      if (/not granted/.test(msg)) throw new AdapterError('PERMISSION_DENIED', sanitizeMessage(msg), 403);
      throw new AdapterError('UNAUTHENTICATED', sanitizeMessage(msg || 'authentication required'), 401);
    }
    if (r.status === 404) throw new AdapterError('NOT_FOUND', sanitizeMessage(msg || 'not found'), 404);
    if (r.status === 400 || r.status === 422) throw new AdapterError('VALIDATION_FAILED', sanitizeMessage(msg || 'invalid request'), 422);
    if (r.status === 409) {
      if (body.code === 'NOT_FOUND') throw new AdapterError('NOT_FOUND', sanitizeMessage(msg), 404);
      throw new AdapterError('CONFLICT', sanitizeMessage(msg || 'conflict'), 409, body.code ? { backendCode: body.code } : undefined);
    }
    if (r.status >= 500) throw new AdapterError('BACKEND_ERROR', 'The control plane reported an internal error.', 502, { status: r.status });
    if (r.status < 200 || r.status >= 300) throw new AdapterError('BACKEND_ERROR', `Unexpected control-plane status ${r.status}.`, 502);
    return r.json;
  }

  async health(signal?: AbortSignal): Promise<BackendHealth> {
    try {
      const r = await this.call('/api/v1/health', { requestId: 'health', signal, timeoutMs: Math.min(this.cfg.timeoutMs, 2500) });
      const h = (r.json ?? {}) as Record<string, unknown>;
      if (r.status !== 200 || typeof h.state !== 'string') return { reachable: false, detail: `health endpoint answered ${r.status}` };
      return {
        reachable: true,
        detail: `member ${String(h.member).slice(0, 12)} is ${h.state}`,
        cluster: h.cluster as string | undefined,
        leader: h.leader as string | undefined,
        member: h.member as string | undefined,
        state: h.state as string
      };
    } catch (e) {
      return { reachable: false, detail: e instanceof AdapterError ? e.message : 'control plane unreachable' };
    }
  }

  async probeUnauthenticatedRejected(signal?: AbortSignal): Promise<boolean | null> {
    if (this.authProbe && Date.now() - this.authProbe.at < 60_000) return this.authProbe.value;
    let value: boolean | null = null;
    try {
      const r = await this.call('/api/v1/view', { requestId: 'authn-probe', signal, timeoutMs: Math.min(this.cfg.timeoutMs, 2500) });
      value = r.status === 401;
    } catch {
      value = null;
    }
    this.authProbe = { at: Date.now(), value };
    return value;
  }

  async snapshot(ctx: RequestCtx): Promise<RealitySnapshot> {
    if (!ctx.token) throw new AdapterError('UNAUTHENTICATED', 'Sign in with an operator capability.', 401);
    // De-duplicate concurrent reads by the same principal for a short window.
    const key = createHash('sha256').update(ctx.token).digest('hex');
    const hit = this.inflight.get(key);
    if (hit && Date.now() - hit.at < 1500) return hit.p;
    const p = (async () => {
      const [r, unauth] = await Promise.all([
        this.call('/api/v1/view', { token: ctx.token, requestId: ctx.requestId, signal: ctx.signal }),
        this.probeUnauthenticatedRejected(ctx.signal)
      ]);
      const json = this.expectOk(r);
      if (!isCPView(json)) throw new AdapterError('BACKEND_MALFORMED', 'Control plane returned a payload that is not a view.', 502);
      return mapView(json, {
        now: Date.now(),
        state: 'LIVE',
        source: `${this.source}/api/v1/view`,
        regionLocations: this.cfg.regionLocations,
        transport: { apiTls: this.apiTls, unauthenticatedRejected: unauth }
      });
    })();
    this.inflight.set(key, { at: Date.now(), p });
    p.catch(() => this.inflight.delete(key));
    return p;
  }

  async auditPage(ctx: RequestCtx, from: number, limit: number): Promise<{ head: number; entries: AuditEntryRec[] }> {
    const r = await this.call(`/api/v1/audit?from=${Math.max(0, from)}&limit=${Math.min(Math.max(limit, 1), 500)}`, { token: ctx.token, requestId: ctx.requestId, signal: ctx.signal });
    const body = this.expectOk(r) as { head?: number; entries?: AuditEntryRec[] | null };
    if (!body || typeof body.head !== 'number') throw new AdapterError('BACKEND_MALFORMED', 'Malformed audit page.', 502);
    return {
      head: body.head,
      entries: (body.entries ?? []).map((e) => ({ ...e, generation: e.generation ?? 0, detail: e.detail ?? '', evidence: e.evidence || null }))
    };
  }

  async verifyAudit(ctx: RequestCtx): Promise<LedgerVerification> {
    const r = await this.call('/api/v1/audit/verify', { token: ctx.token, requestId: ctx.requestId, signal: ctx.signal });
    const v = this.expectOk(r) as { ok?: boolean; entries?: number; head?: string; checkpoints?: number; verifiedAt?: number; verifiedBy?: string; break?: unknown; checkpointBreak?: unknown };
    if (!v || typeof v.ok !== 'boolean') throw new AdapterError('BACKEND_MALFORMED', 'Malformed verification result.', 502);
    return {
      state: v.ok ? 'VERIFIED' : 'INVALID',
      entries: v.entries ?? 0,
      head: v.head ?? '',
      checkpoints: v.checkpoints ?? 0,
      verifiedAt: v.verifiedAt ?? null,
      verifiedBy: v.verifiedBy ?? '',
      breakDetail: v.break ? JSON.stringify(v.break) : v.checkpointBreak ? JSON.stringify(v.checkpointBreak) : null
    };
  }

  async nodeLogs(ctx: RequestCtx, nodeId: string, assignment?: string, tail = 200): Promise<string[]> {
    const q = new URLSearchParams({ tail: String(Math.min(Math.max(tail, 1), 2000)) });
    if (assignment) q.set('assignment', assignment);
    const r = await this.call(`/api/v1/nodes/${encodeURIComponent(nodeId)}/logs?${q}`, { token: ctx.token, requestId: ctx.requestId, signal: ctx.signal });
    if (r.status === 200 && r.json === null) {
      // The endpoint answers plain text; an unreadable log is reported, not hidden.
      if (/^no logs:/.test(r.text)) throw new AdapterError('NOT_FOUND', 'The host has no readable log for this target.', 404);
      return r.text.split('\n').filter((l, i, a) => l !== '' || i < a.length - 1).slice(-2000).map(sanitizeMessage);
    }
    const body = this.expectOk(r);
    if (Array.isArray(body)) return body.map((x) => sanitizeMessage(String(x)));
    if (body && typeof body === 'object' && Array.isArray((body as { lines?: unknown }).lines)) return ((body as { lines: unknown[] }).lines).map((x) => sanitizeMessage(String(x)));
    throw new AdapterError('BACKEND_MALFORMED', 'Malformed log response.', 502);
  }

  private async mutate(path: string, ctx: RequestCtx, body?: unknown, contentType = 'application/json'): Promise<OperationResult> {
    if (!ctx.token) throw new AdapterError('UNAUTHENTICATED', 'Sign in with an operator capability.', 401);
    const r = await this.call(path, {
      method: 'POST',
      token: ctx.token,
      requestId: ctx.requestId,
      signal: ctx.signal,
      contentType,
      body: body === undefined ? undefined : typeof body === 'string' ? body : JSON.stringify(body)
    });
    const res = this.expectOk(r) as { ok?: boolean; message?: string; code?: string; data?: unknown };
    if (!res || typeof res.ok !== 'boolean') throw new AdapterError('BACKEND_MALFORMED', 'Malformed operation result.', 502);
    return { ok: res.ok, code: res.code ?? null, message: sanitizeMessage(res.message ?? ''), data: res.data };
  }

  nodeOperation(ctx: RequestCtx, nodeId: string, op: NodeOperation, reason?: string): Promise<OperationResult> {
    const id = encodeURIComponent(nodeId);
    switch (op) {
      case 'APPROVE':
        return this.mutate(`/api/v1/nodes/${id}/approve`, ctx);
      case 'DRAIN':
        return this.mutate(`/api/v1/nodes/${id}/drain`, ctx);
      case 'UNDRAIN':
        return this.mutate(`/api/v1/nodes/${id}/undrain`, ctx);
      case 'REVOKE':
        return this.mutate(`/api/v1/nodes/${id}/revoke`, ctx, { reason: reason || 'operator revocation from Command Centre' });
    }
  }

  applyManifest(ctx: RequestCtx, yaml: string): Promise<OperationResult> {
    return this.mutate('/api/v1/apply', ctx, { yaml });
  }

  scaleApp(ctx: RequestCtx, app: string, replicas: number): Promise<OperationResult> {
    return this.mutate(`/api/v1/apps/${encodeURIComponent(app)}/scale`, ctx, { replicas });
  }

  deleteApp(ctx: RequestCtx, app: string): Promise<OperationResult> {
    return this.mutate(`/api/v1/apps/${encodeURIComponent(app)}/delete`, ctx);
  }
}
