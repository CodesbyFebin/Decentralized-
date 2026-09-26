/**
 * The standard way to render an operational value: the value (or an honest
 * fallback), its truth state, the freshness of the observation behind it,
 * where it came from and when it was observed. Status is always icon + text
 * + colour, never colour alone.
 */
import React from 'react';
import { CheckCircle2, Clock, CloudOff, HelpCircle, TimerOff, XCircle, AlertTriangle, MinusCircle, ShieldCheck, ShieldX } from 'lucide-react';
import type { Freshness, Metric, TruthState } from '../../types/reality';
import { TONE, type Tone } from './ui';
import { TruthTag, fmtAge, fmtTime, StaleContext } from './states';

export type ViewFreshness = 'FRESH' | 'STALE' | 'EXPIRED' | 'UNREACHABLE' | 'UNKNOWN';

/** Freshness of a view: the page's own fetch state wins over the observation's. */
export function viewFreshness(opts: { pageStale?: boolean; observation?: Freshness | null; lost?: boolean }): ViewFreshness {
  if (opts.pageStale) return 'UNREACHABLE';
  if (opts.lost) return 'EXPIRED';
  switch (opts.observation) {
    case 'LIVE':
      return 'FRESH';
    case 'STALE':
      return 'STALE';
    case undefined:
    case null:
      return 'FRESH';
    default:
      return 'UNKNOWN';
  }
}

const FRESH: Record<ViewFreshness, { tone: Tone; icon: React.ReactNode; label: string }> = {
  FRESH: { tone: 'emerald', icon: <CheckCircle2 className="w-3 h-3" aria-hidden />, label: 'FRESH' },
  STALE: { tone: 'amber', icon: <Clock className="w-3 h-3" aria-hidden />, label: 'STALE' },
  EXPIRED: { tone: 'rose', icon: <TimerOff className="w-3 h-3" aria-hidden />, label: 'EXPIRED' },
  UNREACHABLE: { tone: 'rose', icon: <CloudOff className="w-3 h-3" aria-hidden />, label: 'UNREACHABLE' },
  UNKNOWN: { tone: 'slate', icon: <HelpCircle className="w-3 h-3" aria-hidden />, label: 'UNKNOWN' }
};

export const FreshnessBadge: React.FC<{ f: ViewFreshness; observedAt?: number | null; className?: string }> = ({ f: given, observedAt, className = '' }) => {
  // Inside a stale page nothing is current, whatever the observation said when it was fetched.
  const pageStale = React.useContext(StaleContext);
  const f: ViewFreshness = pageStale && (given === 'FRESH' || given === 'STALE') ? 'UNREACHABLE' : given;
  const s = FRESH[f];
  const c = TONE[s.tone];
  return (
    <span
      className={`inline-flex items-center gap-1 px-1.5 py-[1px] rounded text-[10.5px] font-semibold whitespace-nowrap border ${className}`}
      style={{ color: c, borderColor: `${c}40`, background: `${c}14` }}
      title={observedAt ? `observed ${fmtTime(observedAt)}` : undefined}
    >
      {s.icon}
      {s.label}
      {observedAt ? <span className="font-normal opacity-80">· {fmtAge(Date.now() - observedAt)}</span> : null}
    </span>
  );
};

const FALLBACK: Partial<Record<TruthState, string>> = { UNAVAILABLE: 'Unavailable', PLANNED: 'Planned', UNKNOWN: 'Unknown' };

export interface TruthValueProps {
  value: number | string | null | undefined;
  state: TruthState;
  format?: (v: number) => string;
  unit?: string;
  freshness?: ViewFreshness;
  source?: string;
  observedAt?: number | null;
  /** Text for a missing value; defaults by state ("Unknown", "Unavailable", "Planned", "Not measured"). */
  fallback?: string;
  /** Show the truth/freshness/source line under the value. */
  meta?: boolean;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export const TruthValue: React.FC<TruthValueProps> = ({ value, state, format, unit, freshness, source, observedAt, fallback, meta, size = 'md', className = '' }) => {
  const missing = value === null || value === undefined || value === '';
  // A value observed while the page could not refresh is shown, but never as current.
  const text = missing ? (fallback ?? FALLBACK[state] ?? 'Not measured') : typeof value === 'number' ? (format ? format(value) : value.toLocaleString()) : value;
  const sizeCls = size === 'lg' ? 'text-[26px] font-bold' : size === 'sm' ? 'text-[12.5px]' : 'text-[15px] font-semibold';
  const title = [`${state}${freshness ? ` · ${freshness}` : ''}`, source ? `source: ${source}` : '', observedAt ? `observed ${fmtTime(observedAt)}` : ''].filter(Boolean).join('\n');
  return (
    <span className={`inline-flex flex-col ${className}`} title={title}>
      <span className={`${sizeCls} tabular-nums ${missing ? 'text-slate-400 font-medium' : 'text-white'}`}>
        {text}
        {!missing && unit ? <span className="ml-1 text-[0.7em] text-slate-400 font-medium">{unit}</span> : null}
      </span>
      {meta && (
        <span className="mt-1 flex flex-wrap items-center gap-1 text-[10.5px] text-slate-400">
          {state !== 'LIVE' && <TruthTag state={state} />}
          {freshness && <FreshnessBadge f={freshness} observedAt={observedAt} />}
          {source && <span className="truncate max-w-[180px]">{source}</span>}
        </span>
      )}
    </span>
  );
};

/** A Metric from the BFF model as a TruthValue. */
export const MetricValue: React.FC<Omit<TruthValueProps, 'value' | 'state' | 'source'> & { m: Metric | undefined }> = ({ m, ...rest }) =>
  m ? <TruthValue value={m.value} state={m.state} source={m.source} fallback={m.value === null ? (m.detail ?? undefined) : undefined} {...rest} /> : <TruthValue value={null} state="UNKNOWN" {...rest} />;

/* ------------------------------------------------------ desired/observed */

export type Convergence = 'CONVERGED' | 'DEGRADED' | 'CONVERGING' | 'DIVERGED' | 'STOPPED' | 'UNKNOWN';

const CONV: Record<Convergence, { tone: Tone; icon: React.ReactNode }> = {
  CONVERGED: { tone: 'emerald', icon: <CheckCircle2 className="w-4 h-4" aria-hidden /> },
  CONVERGING: { tone: 'blue', icon: <Clock className="w-4 h-4" aria-hidden /> },
  DEGRADED: { tone: 'amber', icon: <AlertTriangle className="w-4 h-4" aria-hidden /> },
  DIVERGED: { tone: 'rose', icon: <XCircle className="w-4 h-4" aria-hidden /> },
  STOPPED: { tone: 'slate', icon: <MinusCircle className="w-4 h-4" aria-hidden /> },
  UNKNOWN: { tone: 'slate', icon: <HelpCircle className="w-4 h-4" aria-hidden /> }
};

export const ConvergenceBadge: React.FC<{ c: Convergence; title?: string }> = ({ c, title }) => {
  const s = CONV[c];
  const col = TONE[s.tone];
  return (
    <span title={title} className="inline-flex items-center gap-1.5 px-2 py-[3px] rounded-md text-[11.5px] font-bold tracking-wide border" style={{ color: col, borderColor: `${col}40`, background: `${col}14` }}>
      {s.icon}
      {c}
    </span>
  );
};

/**
 * Desired, observed and verified side by side. Verified means "backed by a
 * passing check the system actually ran"; it is never inferred from running.
 */
export const StateComparison: React.FC<{
  desired: number | null;
  observed: number | null;
  verified: number | null;
  labels?: [string, string, string];
  notes?: [string, string, string];
  result: Convergence;
  resultReason?: string;
}> = ({ desired, observed, verified, labels = ['Desired', 'Observed', 'Verified'], notes, result, resultReason }) => (
  <div className="rounded-2xl border border-[rgba(125,190,255,0.12)] bg-white/[0.02] p-4">
    <div className="grid grid-cols-3 gap-3">
      {[desired, observed, verified].map((v, i) => (
        <div key={labels[i]} className="min-w-0 break-words">
          <div className="text-[10px] sm:text-[11px] uppercase tracking-wider text-slate-400">{labels[i]}</div>
          <div className="text-[22px] sm:text-[28px] font-bold tabular-nums text-white">{v === null ? <span className="text-slate-500 text-[16px]">Unknown</span> : v}</div>
          {notes && <div className="text-[11px] text-slate-500">{notes[i]}</div>}
        </div>
      ))}
    </div>
    <div className="mt-3 flex flex-wrap items-center gap-2 text-[12px] text-slate-300">
      <span className="text-slate-400">Result</span>
      <ConvergenceBadge c={result} />
      {resultReason && <span className="text-slate-400">{resultReason}</span>}
    </div>
  </div>
);

/* --------------------------------------------------------------- seals */

export const EvidenceSeal: React.FC<{ outcome: 'PASS' | 'FAIL' | 'INFRA_FAILURE' | 'UNKNOWN'; size?: 'sm' | 'md' }> = ({ outcome, size = 'sm' }) => {
  const s =
    outcome === 'PASS'
      ? { tone: 'emerald' as Tone, icon: <ShieldCheck className={size === 'md' ? 'w-4 h-4' : 'w-3.5 h-3.5'} aria-hidden />, label: 'PASS' }
      : outcome === 'FAIL'
        ? { tone: 'rose' as Tone, icon: <ShieldX className={size === 'md' ? 'w-4 h-4' : 'w-3.5 h-3.5'} aria-hidden />, label: 'FAIL' }
        : outcome === 'INFRA_FAILURE'
          ? { tone: 'amber' as Tone, icon: <AlertTriangle className={size === 'md' ? 'w-4 h-4' : 'w-3.5 h-3.5'} aria-hidden />, label: 'INFRA FAILURE' }
          : { tone: 'slate' as Tone, icon: <HelpCircle className={size === 'md' ? 'w-4 h-4' : 'w-3.5 h-3.5'} aria-hidden />, label: 'UNKNOWN' };
  const c = TONE[s.tone];
  return (
    <span className={`inline-flex items-center gap-1 rounded-md font-bold tracking-wide border ${size === 'md' ? 'px-2.5 py-1 text-[12.5px]' : 'px-1.5 py-[2px] text-[11px]'}`} style={{ color: c, borderColor: `${c}55`, background: `${c}18` }}>
      {s.icon}
      {s.label}
    </span>
  );
};

/** A digest with copy; the full value is always available. */
export const Digest: React.FC<{ value: string | null | undefined; chars?: number }> = ({ value, chars = 16 }) => {
  const [copied, setCopied] = React.useState(false);
  if (!value) return <span className="text-slate-500">—</span>;
  const [alg, hex] = value.includes(':') ? value.split(':', 2) : ['', value];
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[11.5px] text-slate-200" title={value}>
      {alg && <span className="text-slate-500">{alg}:</span>}
      {hex.slice(0, chars)}
      {hex.length > chars && '…'}
      <button
        type="button"
        className="text-[10.5px] text-cyan-300 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 rounded"
        onClick={() => {
          navigator.clipboard?.writeText(value).then(
            () => {
              setCopied(true);
              setTimeout(() => setCopied(false), 1500);
            },
            () => undefined
          );
        }}
        aria-label={`Copy ${value}`}
      >
        {copied ? 'copied' : 'copy'}
      </button>
    </span>
  );
};
