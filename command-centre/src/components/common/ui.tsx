import React from 'react';
import {
  ChevronRight,
  ArrowUp,
  ArrowDown,
  Home,
  Users,
  Box,
  Globe2,
  Cloud,
  Search,
  SlidersHorizontal,
  ArrowUpDown
} from 'lucide-react';
import type { TruthState } from '../../types/reality';

/** A network the console can report on. `state` is the truth state of the integration itself. */
export interface NetworkInfo {
  id: string;
  name: string;
  description: string;
  status: string;
  state: TruthState;
  accent: string;
  glyph: string;
  nodes?: number;
}

/* ------------------------------------------------------------------ */
/* Tokens                                                              */
/* ------------------------------------------------------------------ */

export type Tone = 'cyan' | 'violet' | 'blue' | 'amber' | 'emerald' | 'rose' | 'slate';

export const TONE: Record<Tone, string> = {
  cyan: '#20DDF7',
  violet: '#A855F7',
  blue: '#248BFF',
  amber: '#F59E0B',
  emerald: '#10D981',
  rose: '#FB4F64',
  slate: '#94A3B8'
};



/* ------------------------------------------------------------------ */
/* Surfaces                                                            */
/* ------------------------------------------------------------------ */

export const Glass: React.FC<React.HTMLAttributes<HTMLDivElement> & { strong?: boolean }> = ({
  className = '',
  strong,
  children,
  ...rest
}) => (
  <div
    {...rest}
    className={`${strong ? 'dh-glass-strong' : 'dh-glass'} rounded-2xl ${className}`}
  >
    {children}
  </div>
);

export const IconTile: React.FC<{ tone: Tone; children: React.ReactNode; size?: 'sm' | 'md' | 'lg'; className?: string }> = ({
  tone,
  children,
  size = 'md',
  className = ''
}) => {
  const dims = size === 'lg' ? 'w-12 h-12 rounded-2xl' : size === 'sm' ? 'w-8 h-8 rounded-lg' : 'w-10 h-10 rounded-xl';
  const c = TONE[tone];
  return (
    <div
      className={`${dims} flex items-center justify-center shrink-0 border ${className}`}
      style={{
        color: c,
        background: `linear-gradient(145deg, ${c}2E, ${c}0D)`,
        borderColor: `${c}55`,
        boxShadow: `0 0 18px ${c}33, inset 0 1px 0 rgba(255,255,255,0.12)`
      }}
    >
      {children}
    </div>
  );
};

/* ------------------------------------------------------------------ */
/* Data marks                                                          */
/* ------------------------------------------------------------------ */

export const Sparkline: React.FC<{ data: number[]; tone: Tone; width?: number; height?: number; className?: string }> = ({
  data,
  tone,
  width = 64,
  height = 26,
  className = ''
}) => {
  if (data.length < 2) return null;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const span = max - min || 1;
  const pts = data.map((v, i) => [(i / (data.length - 1)) * width, height - 2 - ((v - min) / span) * (height - 4)]);
  const line = pts.map(([x, y], i) => `${i ? 'L' : 'M'}${x.toFixed(1)},${y.toFixed(1)}`).join(' ');
  const id = `sg-${tone}-${data.length}-${Math.round(data[data.length - 1])}`;
  const c = TONE[tone];
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} className={className} aria-hidden="true">
      <defs>
        <linearGradient id={id} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor={c} stopOpacity="0.35" />
          <stop offset="100%" stopColor={c} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={`${line} L${width},${height} L0,${height} Z`} fill={`url(#${id})`} />
      <path d={line} fill="none" stroke={c} strokeWidth="1.8" strokeLinecap="round" style={{ filter: `drop-shadow(0 0 3px ${c})` }} />
    </svg>
  );
};

/** Five ascending bars, like the "Global Regions" tile. */
export const MiniBars: React.FC<{ values: number[]; tone: Tone }> = ({ values, tone }) => {
  const max = Math.max(...values, 1);
  return (
    <div className="flex items-end gap-1 h-7" aria-hidden="true">
      {values.map((v, i) => (
        <span
          key={i}
          className="w-2 rounded-sm"
          style={{
            height: `${Math.max(12, (v / max) * 100)}%`,
            background: `linear-gradient(180deg, ${TONE[tone]}, ${TONE.violet})`,
            opacity: 0.45 + (i / values.length) * 0.55
          }}
        />
      ))}
    </div>
  );
};

export const RingGauge: React.FC<{
  value: number;
  max?: number;
  tone: Tone;
  size?: number;
  stroke?: number;
  label?: React.ReactNode;
  sublabel?: React.ReactNode;
  track?: string;
}> = ({ value, max = 100, tone, size = 64, stroke = 6, label, sublabel, track = 'rgba(125,190,255,0.12)' }) => {
  const r = (size - stroke) / 2;
  const circ = 2 * Math.PI * r;
  const frac = Math.max(0, Math.min(1, value / max));
  const c = TONE[tone];
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
        <circle cx={size / 2} cy={size / 2} r={r} stroke={track} strokeWidth={stroke} fill="none" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={c}
          strokeWidth={stroke}
          fill="none"
          strokeLinecap="round"
          strokeDasharray={`${circ * frac} ${circ}`}
          style={{ filter: `drop-shadow(0 0 5px ${c})` }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center text-center leading-none">
        {label}
        {sublabel}
      </div>
    </div>
  );
};

/** Multi-segment donut for share-of-total breakdowns. */
export const Donut: React.FC<{
  segments: { value: number; tone: Tone }[];
  size?: number;
  stroke?: number;
  children?: React.ReactNode;
}> = ({ segments, size = 120, stroke = 12, children }) => {
  const r = (size - stroke) / 2;
  const circ = 2 * Math.PI * r;
  const total = segments.reduce((a, s) => a + s.value, 0) || 1;
  let offset = 0;
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" aria-hidden="true">
        <circle cx={size / 2} cy={size / 2} r={r} stroke="rgba(125,190,255,0.10)" strokeWidth={stroke} fill="none" />
        {segments.map((s, i) => {
          const len = (s.value / total) * circ;
          const el = (
            <circle
              key={i}
              cx={size / 2}
              cy={size / 2}
              r={r}
              stroke={TONE[s.tone]}
              strokeWidth={stroke}
              fill="none"
              strokeDasharray={`${Math.max(0, len - 2)} ${circ}`}
              strokeDashoffset={-offset}
              style={{ filter: `drop-shadow(0 0 4px ${TONE[s.tone]})` }}
            />
          );
          offset += len;
          return el;
        })}
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center text-center">{children}</div>
    </div>
  );
};

/* ------------------------------------------------------------------ */
/* KPI tile                                                            */
/* ------------------------------------------------------------------ */

export const KpiTile: React.FC<{
  icon: React.ReactNode;
  tone: Tone;
  label: string;
  value: React.ReactNode;
  sub?: React.ReactNode;
  spark?: number[];
  trailing?: React.ReactNode;
  onClick?: () => void;
}> = ({ icon, tone, label, value, sub, spark, trailing, onClick }) => (
  <Glass
    onClick={onClick}
    className={`@container px-3.5 py-3.5 min-w-0 relative overflow-hidden ${onClick ? 'cursor-pointer dh-hover-lift' : ''}`}
  >
    <div
      className="absolute -left-8 -top-8 w-24 h-24 rounded-full blur-2xl pointer-events-none"
      style={{ background: `${TONE[tone]}22` }}
    />
    <div className="flex items-center gap-3">
    <IconTile tone={tone} size="md" className="!w-11 !h-11 !hidden @[176px]:!flex">
      {icon}
    </IconTile>
    <div className="min-w-0 flex-1 relative z-[1]">
      <div className="text-[12px] text-slate-300/90 leading-snug">{label}</div>
      <div className="text-[22px] leading-tight font-bold text-white tracking-tight tabular-nums whitespace-nowrap">{value}</div>
      {sub && <div className="text-[10.5px] text-slate-400 mt-0.5 leading-snug">{sub}</div>}
    </div>
    </div>
    {spark && <Sparkline data={spark} tone={tone} width={56} height={26} className="absolute right-3 bottom-3 opacity-80" />}
    {trailing}
  </Glass>
);

export const Delta: React.FC<{ value: number; suffix?: string; invert?: boolean }> = ({ value, suffix = '%', invert }) => {
  const up = value >= 0;
  const good = invert ? !up : up;
  return (
    <span className={`inline-flex items-center gap-0.5 font-semibold ${good ? 'text-emerald-400' : 'text-rose-400'}`}>
      {up ? <ArrowUp className="w-3 h-3" /> : <ArrowDown className="w-3 h-3" />}
      {up ? '+' : ''}
      {value}
      {suffix}
    </span>
  );
};

/* ------------------------------------------------------------------ */
/* Status                                                              */
/* ------------------------------------------------------------------ */

const STATUS_TONE: Record<string, Tone> = {
  Online: 'emerald',
  Running: 'emerald',
  Active: 'emerald',
  Healthy: 'emerald',
  Success: 'emerald',
  Deploying: 'blue',
  Preview: 'blue',
  'Ready to Deploy': 'amber',
  'Ready to Install': 'amber',
  'Eligibility Check': 'blue',
  Degraded: 'amber',
  Checking: 'amber',
  Pending: 'amber',
  Offline: 'rose',
  Failed: 'rose',
  'Not Installed': 'rose',
  Stopped: 'slate',
  'Not Configured': 'slate',
  Production: 'emerald',
  HEALTHY: 'emerald',
  READY: 'emerald',
  VALID: 'emerald',
  PASS: 'emerald',
  RUNNING: 'emerald',
  ADMITTED: 'emerald',
  VERIFIED: 'emerald',
  LIVE: 'emerald',
  ACTIVE: 'emerald',
  DEGRADED: 'amber',
  CONVERGING: 'blue',
  DEPLOYING: 'blue',
  VERIFYING: 'blue',
  QUEUED: 'blue',
  PENDING: 'amber',
  PENDING_APPROVAL: 'amber',
  DRAINING: 'amber',
  EXPIRING: 'amber',
  WARN: 'amber',
  STALE: 'amber',
  HELD: 'amber',
  OFFLINE: 'rose',
  FAILED: 'rose',
  FAIL: 'rose',
  REFUSED: 'rose',
  REVOKED: 'rose',
  EXPIRED: 'rose',
  INVALID: 'rose',
  DEPLOY_FAILED: 'rose',
  VALIDATION_FAILED: 'rose',
  UNKNOWN: 'slate',
  UNAVAILABLE: 'slate',
  STOPPED: 'slate',
  SUPERSEDED: 'slate',
  DELETED: 'slate'
};

export const StatusPill: React.FC<{ status: string; tone?: Tone; className?: string; dot?: boolean }> = ({
  status,
  tone,
  className = '',
  dot = true
}) => {
  const t = tone ?? STATUS_TONE[status] ?? 'slate';
  const c = TONE[t];
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-[3px] rounded-md text-[11px] font-semibold whitespace-nowrap border ${className}`}
      style={{ color: c, background: `${c}1A`, borderColor: `${c}40` }}
    >
      {dot && <span className="w-1.5 h-1.5 rounded-full" style={{ background: c, boxShadow: `0 0 6px ${c}` }} />}
      {status}
    </span>
  );
};

export const LiveTag: React.FC<{ label?: string }> = ({ label = 'LIVE' }) => (
  <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[10px] font-bold tracking-wider text-emerald-300 bg-emerald-500/10 border border-emerald-400/30">
    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_6px_#34d399]" />
    {label}
  </span>
);

/** Honest provenance marker for demo seed data (reality matrix: SIMULATED). */
export const DemoTag: React.FC<{ mode: 'demo' | 'live' }> = ({ mode }) =>
  mode === 'demo' ? (
    <span
      title="Values come from the demo platform store, not live node agents."
      className="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-bold tracking-wider text-amber-300 bg-amber-500/10 border border-amber-400/30"
    >
      DEMO DATA
    </span>
  ) : (
    <LiveTag />
  );


/* ------------------------------------------------------------------ */
/* Navigation                                                          */
/* ------------------------------------------------------------------ */

export const PageTabs = <T extends string>({
  tabs,
  active,
  onChange
}: {
  tabs: { id: T; label: string }[];
  active: T;
  onChange: (id: T) => void;
}) => (
  <div role="tablist" className="dh-glass rounded-2xl p-1.5 flex items-center gap-1 overflow-x-auto">
    {tabs.map((t) => (
      <button
        key={t.id}
        role="tab"
        aria-selected={active === t.id}
        onClick={() => onChange(t.id)}
        className={`px-4 py-2 rounded-xl text-[13px] font-semibold whitespace-nowrap transition-all ${
          active === t.id
            ? 'text-white bg-gradient-to-b from-[#1b3f86] to-[#10275a] border border-cyan-400/50 shadow-[0_0_18px_rgba(32,221,247,0.28)]'
            : 'text-slate-400 hover:text-slate-100 border border-transparent'
        }`}
      >
        {t.label}
      </button>
    ))}
  </div>
);

export const FilterChips = <T extends string>({
  chips,
  active,
  onChange
}: {
  chips: { id: T; label: string; count?: number; tone?: Tone }[];
  active: T;
  onChange: (id: T) => void;
}) => (
  <div className="flex items-center gap-1.5 flex-wrap">
    {chips.map((c) => {
      const on = active === c.id;
      const col = c.tone ? TONE[c.tone] : undefined;
      return (
        <button
          key={c.id}
          onClick={() => onChange(c.id)}
          className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[11.5px] font-semibold border transition-all ${
            on
              ? 'bg-blue-600/80 text-white border-cyan-400/50 shadow-[0_0_12px_rgba(36,139,255,0.45)]'
              : 'bg-white/[0.03] text-slate-300 border-[rgba(125,190,255,0.14)] hover:border-[rgba(125,190,255,0.3)]'
          }`}
        >
          {col && <span className="w-2 h-2 rounded-full" style={{ background: col, boxShadow: `0 0 6px ${col}` }} />}
          {c.label}
          {c.count !== undefined && <span className={on ? 'text-cyan-100' : 'text-slate-500'}>({c.count})</span>}
        </button>
      );
    })}
  </div>
);

export const TableToolbar: React.FC<{
  search: string;
  onSearch: (v: string) => void;
  placeholder: string;
  children?: React.ReactNode;
  onSort?: () => void;
  sortLabel?: string;
}> = ({ search, onSearch, placeholder, children, onSort, sortLabel = 'Sort' }) => (
  <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 px-4 pt-4 pb-3">
    {children}
    <div className="flex items-center gap-2 ml-auto">
      <label className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-[rgba(125,190,255,0.14)] focus-within:border-cyan-400/50 w-52">
        <Search className="w-3.5 h-3.5 text-slate-400" />
        <input
          value={search}
          onChange={(e) => onSearch(e.target.value)}
          placeholder={placeholder}
          className="bg-transparent outline-none text-[12px] text-slate-100 placeholder:text-slate-500 w-full"
        />
      </label>
      <span className="hidden sm:inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-[rgba(125,190,255,0.14)] text-[12px] text-slate-300">
        <SlidersHorizontal className="w-3.5 h-3.5" /> Filter
      </span>
      {onSort && (
        <button
          onClick={onSort}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-[rgba(125,190,255,0.14)] hover:border-cyan-400/40 text-[12px] text-slate-300"
        >
          <ArrowUpDown className="w-3.5 h-3.5" /> {sortLabel}
        </button>
      )}
    </div>
  </div>
);

/* ------------------------------------------------------------------ */
/* Right rail                                                          */
/* ------------------------------------------------------------------ */

export const RailPanel: React.FC<{
  icon?: React.ReactNode;
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}> = ({ icon, title, subtitle, action, children, className = '' }) => (
  <Glass className={`p-4 ${className}`}>
    <div className="flex items-start gap-3 mb-3">
      {icon}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-[15px] font-bold text-white tracking-tight whitespace-nowrap">{title}</h3>
          {action}
        </div>
        {subtitle && <p className="text-[12px] text-slate-400 leading-snug mt-0.5">{subtitle}</p>}
      </div>
    </div>
    {children}
  </Glass>
);

export const RailItem: React.FC<{
  icon: React.ReactNode;
  tone: Tone;
  title: string;
  subtitle: string;
  onClick?: () => void;
  trailing?: React.ReactNode;
}> = ({ icon, tone, title, subtitle, onClick, trailing }) => (
  <button
    onClick={onClick}
    className="w-full flex items-center gap-3 p-2.5 rounded-xl bg-white/[0.025] border border-[rgba(125,190,255,0.09)] hover:border-cyan-400/40 hover:bg-white/[0.05] transition-all text-left group"
  >
    <IconTile tone={tone}>{icon}</IconTile>
    <div className="flex-1 min-w-0">
      <div className="text-[13px] font-semibold text-slate-100 truncate">{title}</div>
      <div className="text-[11px] text-slate-400 truncate">{subtitle}</div>
    </div>
    {trailing ?? <ChevronRight className="w-4 h-4 text-slate-500 group-hover:text-cyan-300 transition-colors" />}
  </button>
);

export const ViewAll: React.FC<{ onClick?: () => void; label?: string }> = ({ onClick, label = 'View All' }) => (
  <button onClick={onClick} className="inline-flex items-center gap-0.5 text-[12px] font-semibold text-cyan-300 hover:text-cyan-200">
    {label} <ChevronRight className="w-3.5 h-3.5" />
  </button>
);

export const NetworkGlyph: React.FC<{ network: Pick<NetworkInfo, 'glyph' | 'accent' | 'name'> }> = ({ network }) => {
  const letter = network.name.replace(/[^A-Za-z]/g, '').charAt(0).toUpperCase();
  return (
    <div
      className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0 font-extrabold text-[15px] border"
      style={{
        color: network.accent,
        background: `radial-gradient(circle at 30% 25%, ${network.accent}40, ${network.accent}0F 70%)`,
        borderColor: `${network.accent}55`,
        boxShadow: `0 0 14px ${network.accent}30`
      }}
      aria-hidden="true"
    >
      {network.glyph === 'cube' ? <Box className="w-4.5 h-4.5" /> : network.glyph === 'cloud' ? <Cloud className="w-4.5 h-4.5" /> : letter}
    </div>
  );
};

export const NetworkRow: React.FC<{ network: NetworkInfo; onClick?: () => void }> = ({ network, onClick }) => (
  <button
    onClick={onClick}
    className="w-full flex items-center gap-3 py-2 px-1 rounded-xl hover:bg-white/[0.035] transition-colors text-left"
  >
    <NetworkGlyph network={network} />
    <div className="flex-1 min-w-0">
      <div className="text-[12px] font-semibold text-slate-100 leading-tight truncate">{network.name}</div>
      <div className="text-[10.5px] text-slate-400 leading-snug truncate">{network.description}</div>
    </div>
    <div className="flex flex-col items-end gap-1 shrink-0">
      <StatusPill status={network.status} className="!text-[9.5px] !px-1.5 !gap-1" />
      {network.nodes !== undefined && <span className="text-[10.5px] text-slate-400">{network.nodes} nodes</span>}
    </div>
  </button>
);


/* ------------------------------------------------------------------ */
/* Globe callouts                                                      */
/* ------------------------------------------------------------------ */

export const RegionCallout: React.FC<{ tone: Tone; title: string; lines: React.ReactNode[]; icon?: React.ReactNode; onClick?: () => void }> = ({
  tone,
  title,
  lines,
  icon,
  onClick
}) => (
  <button
    onClick={onClick}
    className="flex items-start gap-2.5 pl-2 pr-3.5 py-2 rounded-xl text-left backdrop-blur-md border whitespace-nowrap hover:scale-[1.03] transition-transform"
    style={{
      background: 'rgba(6,16,40,0.78)',
      borderColor: `${TONE[tone]}66`,
      boxShadow: `0 0 18px ${TONE[tone]}30, inset 0 1px 0 rgba(255,255,255,0.08)`
    }}
  >
    <IconTile tone={tone} size="sm">
      {icon ?? <Box className="w-4 h-4" />}
    </IconTile>
    <div className="leading-tight">
      <div className="text-[12px] font-bold text-white">{title}</div>
      {lines.map((l, i) => (
        <div key={i} className="text-[11px] text-slate-300">
          {l}
        </div>
      ))}
    </div>
  </button>
);

export const Legend: React.FC<{ items: { label: string; value: React.ReactNode; tone: Tone }[] }> = ({ items }) => (
  <ul className="space-y-2">
    {items.map((i) => (
      <li key={i.label} className="flex items-center justify-between gap-4 text-[12px]">
        <span className="flex items-center gap-2 text-slate-300">
          <span className="w-2 h-2 rounded-full" style={{ background: TONE[i.tone], boxShadow: `0 0 6px ${TONE[i.tone]}` }} />
          {i.label}
        </span>
        <span className="font-semibold text-slate-100 tabular-nums">{i.value}</span>
      </li>
    ))}
  </ul>
);

export const PanelHeader: React.FC<{ icon?: React.ReactNode; title: string; subtitle?: string; right?: React.ReactNode }> = ({
  icon,
  title,
  subtitle,
  right
}) => (
  <div className="flex flex-wrap items-start justify-between gap-3">
    <div className="flex items-start gap-3 min-w-0">
      {icon}
      <div className="min-w-0">
        <h3 className="text-[15px] font-bold text-white tracking-tight">{title}</h3>
        {subtitle && <p className="text-[11.5px] text-slate-400 mt-0.5">{subtitle}</p>}
      </div>
    </div>
    {right}
  </div>
);

/** Gradient headline word, as in "Deploy <anywhere>." */
export const Grad: React.FC<{ children: React.ReactNode; from?: string; to?: string }> = ({ children, from = '#20DDF7', to = '#A855F7' }) => (
  <span style={{ background: `linear-gradient(90deg, ${from}, ${to})`, WebkitBackgroundClip: 'text', backgroundClip: 'text', color: 'transparent' }}>
    {children}
  </span>
);

export const PrimaryButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement>> = ({ className = '', children, ...rest }) => (
  <button
    {...rest}
    className={`inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl bg-btn-spatial text-white text-[13.5px] font-semibold whitespace-nowrap shadow-[0_0_24px_rgba(36,139,255,0.45)] border border-white/20 disabled:opacity-60 ${className}`}
  >
    {children}
  </button>
);

export const GhostButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement>> = ({ className = '', children, ...rest }) => (
  <button
    {...rest}
    className={`inline-flex items-center justify-center gap-2 px-3.5 py-2.5 rounded-xl text-slate-100 text-[13px] font-semibold whitespace-nowrap bg-[rgba(10,24,50,0.6)] border border-[rgba(125,190,255,0.22)] hover:border-cyan-400/50 hover:bg-[rgba(16,34,70,0.7)] transition-all ${className}`}
  >
    {children}
  </button>
);
