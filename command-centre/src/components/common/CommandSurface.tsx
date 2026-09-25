import React from 'react';
import { ChevronRight } from 'lucide-react';
import { HoloGlobe, GlobeArc, GlobeMarker } from './HoloGlobe';
import { RegionRollup } from '../../types/platform';
import { TONE, Tone } from './ui';

/** Main column + right rail, the layout shared by the Deploy / Storage / Nodes surfaces. */
export const SurfaceLayout: React.FC<{ rail: React.ReactNode; children: React.ReactNode }> = ({ rail, children }) => (
  <div className="grid grid-cols-1 2xl:grid-cols-[minmax(0,1fr)_296px] xl:grid-cols-[minmax(0,1fr)_280px] gap-4">
    <div className="min-w-0 space-y-4">{children}</div>
    <aside className="min-w-0 space-y-4">{rail}</aside>
  </div>
);

/**
 * Spatial hero: big gradient headline on the left, a live holographic globe bleeding
 * off the top-right with region callouts, and an optional overlay (pipeline, legend).
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
}> = ({ eyebrow, title, description, actions, markers, arcs, focusLng = 40, overlay }) => (
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
      className="dh-fade-edges !absolute right-[-8px] top-[-64px] w-full md:w-[64%] h-[440px] opacity-40 md:opacity-100 pointer-events-auto"
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

export const Eyebrow: React.FC<{ label: string; live?: boolean }> = ({ label, live = true }) => (
  <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-lg bg-white/[0.04] border border-[rgba(125,190,255,0.16)] text-[12.5px] text-slate-200">
    <ChevronRight className="w-3.5 h-3.5 text-cyan-300" />
    <span className="font-medium">{label}</span>
    {live && (
      <>
        <span className="text-slate-600">·</span>
        <span className="inline-flex items-center gap-1 text-emerald-400 font-semibold text-[11px] tracking-wider">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" /> LIVE
        </span>
      </>
    )}
  </div>
);

/** Region anchors → globe markers with callout cards. */
export function regionMarkers(
  regions: RegionRollup[],
  render: (r: RegionRollup) => { tone: Tone; callout: React.ReactNode } | null
): GlobeMarker[] {
  const out: GlobeMarker[] = [];
  for (const r of regions) {
    const res = render(r);
    if (!res) continue;
    out.push({
      id: r.region,
      lat: r.coordinates[0],
      lng: r.coordinates[1],
      color: TONE[res.tone],
      kind: 'cube',
      callout: res.callout
    });
  }
  return out;
}

/** Hub-and-spoke arcs between populated regions, alternating accent colours. */
export function regionArcs(regions: RegionRollup[], populated: (r: RegionRollup) => boolean): GlobeArc[] {
  const pts = regions.filter(populated);
  const colors = [TONE.cyan, TONE.violet, TONE.blue, '#F472B6'];
  const arcs: GlobeArc[] = [];
  for (let i = 0; i < pts.length; i++) {
    for (let j = i + 1; j < pts.length; j++) {
      if (pts.length <= 4 || (i + j) % 2 === 1 || j === i + 1) {
        arcs.push({ from: pts[i].coordinates, to: pts[j].coordinates, color: colors[(i + j) % colors.length] });
      }
    }
  }
  return arcs;
}
