import React from 'react';
import { ChevronRight, MapPinOff } from 'lucide-react';
import { HoloGlobe, GlobeArc, GlobeMarker } from './HoloGlobe';
import type { NodeHealth, NodeRec, TruthState } from '../../types/reality';
import { TONE, Tone, RegionCallout } from './ui';

/** Main column + right rail, the layout shared by the Deploy / Storage / Nodes surfaces. */
export const SurfaceLayout: React.FC<{ rail: React.ReactNode; children: React.ReactNode }> = ({ rail, children }) => (
  <div className="grid grid-cols-1 2xl:grid-cols-[minmax(0,1fr)_296px] xl:grid-cols-[minmax(0,1fr)_280px] gap-4">
    <div className="min-w-0 space-y-4">{children}</div>
    <aside className="min-w-0 space-y-4">{rail}</aside>
  </div>
);

/**
 * Spatial hero: gradient headline, the live topology globe bleeding off the
 * top-right, and an optional overlay. When `frozen` the globe stops moving.
 */
export const SurfaceHero: React.FC<{
  eyebrow: React.ReactNode;
  title: React.ReactNode;
  description: React.ReactNode;
  actions: React.ReactNode;
  markers: GlobeMarker[];
  arcs: GlobeArc[];
  focusLng?: number;
  overlay?: React.ReactNode;
  frozen?: boolean;
}> = ({ eyebrow, title, description, actions, markers, arcs, focusLng = 40, overlay, frozen }) => (
  <section className="relative min-h-[320px] lg:min-h-[300px] [overflow-x:clip] md:overflow-visible">
    <HoloGlobe
      markers={markers}
      arcs={arcs}
      focusLng={focusLng}
      tilt={20}
      speed={2.2}
      center={[0.56, 0.5]}
      radius={0.44}
      resolution={1.25}
      frozen={frozen}
      className={`dh-fade-edges !absolute right-[-8px] top-[-64px] w-full md:w-[64%] h-[440px] opacity-40 md:opacity-100 pointer-events-auto ${frozen ? 'grayscale' : ''}`}
    />
    <div className="absolute inset-y-0 left-0 w-full md:w-[60%] bg-[linear-gradient(90deg,rgba(2,7,17,0.85)_40%,transparent)] pointer-events-none" />
    {overlay}
    <div className="relative pt-3 pointer-events-none">
      <div className="pointer-events-auto w-fit">{eyebrow}</div>
      <h1 className="mt-3 max-w-[600px] text-[38px] sm:text-[44px] leading-[1.04] font-extrabold tracking-[-0.03em] text-white">{title}</h1>
      <p className="mt-4 text-[14.5px] leading-relaxed text-slate-300 max-w-[450px]">{description}</p>
      <div className="mt-5 flex flex-wrap gap-2.5 pointer-events-auto w-fit max-w-full">{actions}</div>
    </div>
  </section>
);

export const Eyebrow: React.FC<{ label: string; state?: TruthState | null }> = ({ label, state }) => (
  <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-lg bg-white/[0.04] border border-[rgba(125,190,255,0.16)] text-[12.5px] text-slate-200">
    <ChevronRight className="w-3.5 h-3.5 text-cyan-300" />
    <span className="font-medium">{label}</span>
    {state && (
      <>
        <span className="text-slate-600">·</span>
        <span
          className={`inline-flex items-center gap-1 font-semibold text-[11px] tracking-wider ${state === 'LIVE' ? 'text-emerald-400' : state === 'SIMULATED' ? 'text-amber-300' : 'text-slate-400'}`}
        >
          <span className={`w-1.5 h-1.5 rounded-full ${state === 'LIVE' ? 'bg-emerald-400 animate-pulse' : state === 'SIMULATED' ? 'bg-amber-400' : 'bg-slate-500'}`} />
          {state}
        </span>
      </>
    )}
  </div>
);

const HEALTH_TONE: Record<NodeHealth, Tone> = { HEALTHY: 'emerald', DEGRADED: 'amber', OFFLINE: 'rose', UNKNOWN: 'slate' };

export interface RegionGroup {
  region: string;
  nodes: NodeRec[];
  lat: number;
  lng: number;
  worst: NodeHealth;
}

/** Group hosts by enrolled region; only regions with an operator-configured position are placeable. */
export function regionGroups(nodes: NodeRec[]): { placed: RegionGroup[]; unplaced: string[] } {
  const by = new Map<string, NodeRec[]>();
  for (const n of nodes) by.set(n.region || '(no region)', [...(by.get(n.region || '(no region)') ?? []), n]);
  const order: NodeHealth[] = ['OFFLINE', 'DEGRADED', 'UNKNOWN', 'HEALTHY'];
  const placed: RegionGroup[] = [];
  const unplaced: string[] = [];
  for (const [region, ns] of by) {
    const loc = ns.find((n) => n.location)?.location;
    if (!loc) {
      unplaced.push(region);
      continue;
    }
    const worst = order.find((h) => ns.some((n) => n.health === h)) ?? 'UNKNOWN';
    placed.push({ region, nodes: ns, lat: loc.lat, lng: loc.lng, worst });
  }
  return { placed, unplaced };
}

export function groupMarkers(groups: RegionGroup[], lines: (g: RegionGroup) => React.ReactNode[], onClick?: (g: RegionGroup) => void): GlobeMarker[] {
  return groups.map((g) => {
    const tone = HEALTH_TONE[g.worst];
    return {
      id: g.region,
      lat: g.lat,
      lng: g.lng,
      color: TONE[tone],
      kind: 'cube' as const,
      callout: <RegionCallout tone={tone} title={g.region} lines={lines(g)} onClick={onClick ? () => onClick(g) : undefined} />
    };
  });
}

/** Arcs only between regions that both have a healthy host: they represent live mesh paths. */
export function meshArcs(groups: RegionGroup[]): GlobeArc[] {
  const live = groups.filter((g) => g.nodes.some((n) => n.health === 'HEALTHY'));
  const colors = [TONE.cyan, TONE.violet, TONE.blue, '#F472B6'];
  const arcs: GlobeArc[] = [];
  for (let i = 0; i < live.length; i++) for (let j = i + 1; j < live.length; j++) arcs.push({ from: [live[i].lat, live[i].lng], to: [live[j].lat, live[j].lng], color: colors[(i + j) % colors.length] });
  return arcs;
}

export const NoGeoNote: React.FC<{ unplaced: string[] }> = ({ unplaced }) =>
  unplaced.length ? (
    <div className="flex items-start gap-2 text-[11.5px] text-slate-400">
      <MapPinOff className="w-3.5 h-3.5 mt-0.5 shrink-0" />
      <span>
        {unplaced.length} region(s) not on the map ({unplaced.slice(0, 4).join(', ')}{unplaced.length > 4 ? '…' : ''}): hosts do not report coordinates. Set <code className="text-cyan-200">DH_REGION_LOCATIONS</code> to place them.
      </span>
    </div>
  ) : null;
