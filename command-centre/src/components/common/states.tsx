import React from 'react';
import { AlertTriangle, CloudOff, Lock, RefreshCw, Ban, Inbox, Info } from 'lucide-react';
import type { Freshness, Metric, TruthState } from '../../types/reality';
import type { Resource } from '../../lib/useResource';
import { ApiError } from '../../lib/client';
import { Glass, TONE, type Tone } from './ui';

/* ------------------------------------------------------------ formatting */

export const fmtBytes = (n: number | null | undefined): string => {
  if (n === null || n === undefined) return '—';
  const u = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v >= 100 || i === 0 ? Math.round(v) : v.toFixed(1)} ${u[i]}`;
};

export const fmtAge = (ms: number | null | undefined): string => {
  if (ms === null || ms === undefined) return 'never';
  if (ms < 1000) return 'just now';
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.round(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.round(m / 60);
  if (h < 48) return `${h}h ago`;
  return `${Math.round(h / 24)}d ago`;
};

export const since = (ts: number | null | undefined) => (ts ? fmtAge(Date.now() - ts) : 'never');
export const fmtTime = (ts: number | null | undefined) => (ts ? new Date(ts).toLocaleString() : '—');
export const short = (s: string | null | undefined, n = 12) => (!s ? '—' : s.length > n + 1 ? `${s.slice(0, n)}…` : s);
export const shortDigest = (d: string | null | undefined) => (!d ? '—' : d.includes(':') ? `${d.split(':')[0]}:${d.split(':')[1].slice(0, 12)}…` : short(d, 14));

/* --------------------------------------------------------------- truth */

const TRUTH_TONE: Record<TruthState, Tone> = {
  LIVE: 'emerald',
  DERIVED: 'cyan',
  CONFIGURED: 'violet',
  UNAVAILABLE: 'slate',
  PLANNED: 'blue',
  SIMULATED: 'amber',
  UNKNOWN: 'slate'
};

export const TruthTag: React.FC<{ state: TruthState; title?: string }> = ({ state, title }) => {
  const c = TONE[TRUTH_TONE[state]];
  return (
    <span
      title={title}
      className="inline-flex items-center gap-1 px-1.5 py-[1px] rounded text-[9.5px] font-bold tracking-wider border"
      style={{ color: c, borderColor: `${c}40`, background: `${c}14` }}
    >
      {state}
    </span>
  );
};

/** Render a Metric: unmeasured values are an em dash, never zero. */
export function metricText(m: Metric | undefined, fmt: (n: number) => string = (n) => n.toLocaleString()): string {
  if (!m || m.value === null) return '—';
  return fmt(m.value);
}

export const MetricNote: React.FC<{ m: Metric | undefined }> = ({ m }) =>
  !m ? null : m.value === null ? <span className="text-slate-500">{m.detail ?? 'not measured'}</span> : <TruthTag state={m.state} title={`${m.source}${m.detail ? ` — ${m.detail}` : ''}`} />;

const FRESH_TONE: Record<Freshness, Tone> = { LIVE: 'emerald', STALE: 'amber', UNKNOWN: 'slate', UNAVAILABLE: 'rose' };

export const FreshnessPill: React.FC<{ f: Freshness; ageMs?: number | null }> = ({ f, ageMs }) => {
  const c = TONE[FRESH_TONE[f]];
  return (
    <span className="inline-flex items-center gap-1.5 px-2 py-[3px] rounded-md text-[11px] font-semibold whitespace-nowrap border" style={{ color: c, background: `${c}1A`, borderColor: `${c}40` }}>
      <span className="w-1.5 h-1.5 rounded-full" style={{ background: c }} />
      {f === 'LIVE' ? 'FRESH' : f}
      {ageMs !== undefined && ageMs !== null && <span className="font-normal opacity-80">· {fmtAge(ageMs)}</span>}
    </span>
  );
};

/* -------------------------------------------------------------- states */

export const Loading: React.FC<{ label?: string; rows?: number }> = ({ label = 'Loading observations…', rows = 3 }) => (
  <div className="space-y-3" role="status" aria-live="polite">
    <span className="sr-only">{label}</span>
    {Array.from({ length: rows }).map((_, i) => (
      <div key={i} className="h-16 rounded-2xl dh-glass animate-pulse opacity-60" />
    ))}
  </div>
);

export const Empty: React.FC<{ title: string; detail?: string; action?: React.ReactNode }> = ({ title, detail, action }) => (
  <Glass className="p-8 text-center">
    <Inbox className="w-7 h-7 mx-auto text-slate-500" />
    <div className="mt-2 text-[14px] font-semibold text-slate-200">{title}</div>
    {detail && <p className="mt-1 text-[12.5px] text-slate-400 max-w-md mx-auto">{detail}</p>}
    {action && <div className="mt-4 flex justify-center">{action}</div>}
  </Glass>
);

export const Unavailable: React.FC<{ title: string; detail: string; state?: TruthState; children?: React.ReactNode }> = ({ title, detail, state = 'UNAVAILABLE', children }) => (
  <Glass className="p-5 border-dashed">
    <div className="flex items-start gap-3">
      <Ban className="w-5 h-5 text-slate-500 shrink-0 mt-0.5" />
      <div className="min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-[14px] font-semibold text-slate-200">{title}</span>
          <TruthTag state={state} />
        </div>
        <p className="mt-1 text-[12.5px] text-slate-400">{detail}</p>
        {children}
      </div>
    </div>
  </Glass>
);

export const ErrorState: React.FC<{ error: ApiError; onRetry?: () => void }> = ({ error, onRetry }) => {
  const denied = error.code === 'PERMISSION_DENIED' || error.status === 403;
  const down = error.backendDown;
  const Icon = denied ? Lock : down ? CloudOff : AlertTriangle;
  const title = denied ? 'Permission denied' : down ? 'Control plane unreachable' : error.code === 'UNAUTHENTICATED' ? 'Sign-in required' : error.code === 'NOT_FOUND' ? 'Not found' : 'Request failed';
  return (
    <Glass className="p-6" role="alert">
      <div className="flex items-start gap-3">
        <Icon className={`w-6 h-6 shrink-0 ${denied ? 'text-amber-300' : 'text-rose-300'}`} />
        <div className="min-w-0 flex-1">
          <div className="text-[15px] font-semibold text-white">{title}</div>
          <p className="mt-1 text-[13px] text-slate-300 break-words">{error.message}</p>
          <div className="mt-2 text-[11px] text-slate-500 font-mono">
            {error.code}
            {error.requestId && <> · request {error.requestId}</>}
          </div>
        </div>
        {onRetry && !denied && (
          <button onClick={onRetry} className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[rgba(125,190,255,0.2)] text-[12px] text-slate-200 hover:border-cyan-400/50">
            <RefreshCw className="w-3.5 h-3.5" /> Retry
          </button>
        )}
      </div>
    </Glass>
  );
};

export const StaleBanner: React.FC<{ fetchedAt: number | null; error: ApiError | null }> = ({ fetchedAt, error }) => (
  <div role="status" className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-amber-500/10 border border-amber-400/30 text-[12.5px] text-amber-100">
    <CloudOff className="w-4 h-4 text-amber-300 shrink-0" />
    <span>
      <b>{error?.backendDown ? 'CONTROL PLANE UNREACHABLE' : 'REFRESH FAILED'}</b> — showing the last observation from {fmtTime(fetchedAt)} ({since(fetchedAt)}). Nothing below is current.
    </span>
    {error?.requestId && <span className="ml-auto font-mono text-[10.5px] text-amber-200/70">{error.requestId}</span>}
  </div>
);

/**
 * Standard wrapper: loading → error → content. Content from a failed refresh
 * stays visible only desaturated, under a stale banner.
 */
export function Gate<T>({ res, children, loadingRows }: { res: Resource<T>; children: (data: T, stale: boolean) => React.ReactNode; loadingRows?: number }) {
  if (res.data === null) {
    if (res.error) return <ErrorState error={res.error} onRetry={res.refresh} />;
    return <Loading rows={loadingRows} />;
  }
  return (
    <>
      {res.stale && <StaleBanner fetchedAt={res.fetchedAt} error={res.error} />}
      <div className={res.stale ? 'grayscale opacity-60 transition-all' : 'transition-all'} aria-disabled={res.stale || undefined}>
        {children(res.data, res.stale)}
      </div>
    </>
  );
}

export const Note: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <p className="flex items-start gap-1.5 text-[11.5px] text-slate-400">
    <Info className="w-3.5 h-3.5 mt-0.5 shrink-0 text-slate-500" />
    <span>{children}</span>
  </p>
);
