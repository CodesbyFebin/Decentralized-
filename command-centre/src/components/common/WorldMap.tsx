import React, { useEffect, useMemo, useState } from 'react';
import { Globe, ArrowUpRight } from 'lucide-react';
import { api } from '../../lib/api';
import { NodeInfo } from '../../types/platform';
import { HoloGlobe, GlobeArc, GlobeMarker } from './HoloGlobe';

interface Props {
  selectedRegion?: string;
  onSelectRegion?: (region: string) => void;
  heightClass?: string;
  onViewAllNodes?: () => void;
  /** Explicit counts override the ones derived from node observations. */
  onlineCount?: number;
  degradedCount?: number;
  offlineCount?: number;
  unknownCount?: number;
  showRegions?: boolean;
  showStorageLabels?: boolean;
}

const STATUS_COLOR: Record<NodeInfo['status'], string> = { Online: '#10D981', Degraded: '#FBBF24', Offline: '#FB4F64' };
const REGION_COLOR = ['#20DDF7', '#248BFF', '#F59E0B', '#A855F7', '#10D981', '#F472B6', '#38BDF8'];

/**
 * Global infrastructure card: holographic globe with one marker per observed node,
 * region callouts derived from node placement, and a live status strip.
 */
export const WorldMap: React.FC<Props> = ({
  onSelectRegion,
  heightClass = 'h-[360px]',
  onViewAllNodes,
  onlineCount,
  degradedCount,
  offlineCount,
  unknownCount,
  showRegions = true,
  showStorageLabels
}) => {
  const [nodes, setNodes] = useState<NodeInfo[] | null>(null);

  useEffect(() => {
    api
      .getNodes()
      .then((r) => setNodes(r.data))
      .catch(() => setNodes([]));
  }, []);

  const { markers, arcs, counts } = useMemo(() => {
    const list = nodes ?? [];
    const byRegion = new Map<string, NodeInfo[]>();
    list.forEach((n) => byRegion.set(n.region, [...(byRegion.get(n.region) ?? []), n]));
    const regions = [...byRegion.entries()];

    const markers: GlobeMarker[] = list.map((n) => ({
      id: n.id,
      lat: n.coordinates[0],
      lng: n.coordinates[1],
      color: STATUS_COLOR[n.status],
      size: 0.8
    }));

    if (showRegions) {
      regions.forEach(([region, ns], i) => {
        const lat = ns.reduce((a, n) => a + n.coordinates[0], 0) / ns.length;
        const lng = ns.reduce((a, n) => a + n.coordinates[1], 0) / ns.length;
        const color = REGION_COLOR[i % REGION_COLOR.length];
        const storageTb = ns.reduce((a, n) => a + n.diskTotalGb, 0) / 1000;
        markers.push({
          id: `region-${region}`,
          lat,
          lng,
          color,
          kind: 'cube',
          size: 1.1,
          callout: (
            <button
              onClick={() => onSelectRegion?.(region)}
              className="flex items-center gap-2 pl-2 pr-3 py-1.5 rounded-full bg-[rgba(6,16,40,0.82)] backdrop-blur-md border text-left whitespace-nowrap hover:scale-[1.04] transition-transform"
              style={{ borderColor: `${color}66`, boxShadow: `0 0 14px ${color}33` }}
            >
              <span className="w-2 h-2 rounded-full" style={{ background: color, boxShadow: `0 0 6px ${color}` }} />
              <span className="text-[11.5px] font-semibold text-white">{region}</span>
              <span className="text-[10.5px] text-slate-400">
                {showStorageLabels ? `${storageTb.toFixed(1)} TB` : `${ns.length} node${ns.length === 1 ? '' : 's'}`}
              </span>
            </button>
          )
        });
      });
    }

    const arcs: GlobeArc[] = [];
    for (let i = 0; i < list.length; i++) {
      const j = (i * 3 + 1) % list.length;
      if (i !== j) arcs.push({ from: list[i].coordinates, to: list[j].coordinates, color: REGION_COLOR[i % REGION_COLOR.length] });
    }

    return {
      markers,
      arcs,
      counts: {
        online: list.filter((n) => n.status === 'Online').length,
        degraded: list.filter((n) => n.status === 'Degraded').length,
        offline: list.filter((n) => n.status === 'Offline').length
      }
    };
  }, [nodes, showRegions, showStorageLabels, onSelectRegion]);

  const strip = [
    { label: 'Online', value: onlineCount ?? counts.online, color: '#10D981' },
    { label: 'Degraded', value: degradedCount ?? counts.degraded, color: '#FBBF24' },
    { label: 'Offline', value: offlineCount ?? counts.offline, color: '#FB4F64' },
    { label: 'Unknown', value: unknownCount ?? 0, color: '#64748B' }
  ];

  return (
    <div className={`relative w-full rounded-2xl dh-glass overflow-hidden ${heightClass}`}>
      <HoloGlobe markers={markers} arcs={arcs} focusLng={40} tilt={20} speed={2} center={[0.47, 0.58]} radius={0.44} resolution={1.3} safeRight={120} className="absolute inset-0" />

      <div className="relative z-10 flex items-start justify-between p-4 pointer-events-none">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-xl bg-blue-600/20 border border-cyan-400/30 flex items-center justify-center text-cyan-400 shadow-[0_0_12px_rgba(32,221,247,0.3)]">
            <Globe className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-white tracking-tight">Global Infrastructure</h3>
            <p className="text-[11px] text-slate-400">{nodes === null ? 'Loading node observations…' : `${nodes.length} nodes across ${new Set(nodes.map((n) => n.region)).size} regions`}</p>
          </div>
        </div>
        {onViewAllNodes && (
          <button onClick={onViewAllNodes} className="pointer-events-auto flex items-center gap-1 text-[11.5px] font-semibold text-cyan-300 hover:text-cyan-200">
            View All Nodes <ArrowUpRight className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      <ul className="absolute right-4 top-16 z-10 space-y-2 pointer-events-none">
        {strip.map((s) => (
          <li key={s.label} className="flex items-center justify-between gap-5 text-[11.5px]">
            <span className="flex items-center gap-2 text-slate-300">
              <span className="w-2 h-2 rounded-full" style={{ background: s.color, boxShadow: `0 0 6px ${s.color}` }} />
              {s.label}
            </span>
            <span className="font-semibold text-white tabular-nums">{s.value}</span>
          </li>
        ))}
      </ul>
    </div>
  );
};
