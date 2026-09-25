import React, { useState } from 'react';
import { Globe, ArrowUpRight, Radio } from 'lucide-react';

interface Props {
  selectedRegion?: string;
  onSelectRegion?: (region: string) => void;
  heightClass?: string;
  onViewAllNodes?: () => void;
  onlineCount?: number;
  degradedCount?: number;
  offlineCount?: number;
  unknownCount?: number;
  showRegions?: boolean;
  showStorageLabels?: boolean;
}

export const WorldMap: React.FC<Props> = ({
  selectedRegion,
  onSelectRegion,
  heightClass = 'h-[360px]',
  onViewAllNodes,
  onlineCount = 8,
  degradedCount = 1,
  offlineCount = 1,
  unknownCount = 0
}) => {
  const [hoveredRegion, setHoveredRegion] = useState<string | null>(null);

  const regionPills = [
    { id: 'na', name: 'North America', nodes: 2, status: 'Online', cx: 28, cy: 36, color: '#20DDF7' },
    { id: 'eu', name: 'Europe', nodes: 3, status: 'Online', cx: 50, cy: 28, color: '#248BFF' },
    { id: 'as', name: 'Asia', nodes: 2, status: 'Online', cx: 74, cy: 38, color: '#F59E0B' },
    { id: 'sa', name: 'South America', nodes: 1, status: 'Degraded', cx: 34, cy: 68, color: '#EF4444' },
    { id: 'af', name: 'Africa', nodes: 0, status: 'Planned', cx: 52, cy: 58, color: '#F43F5E' },
    { id: 'oc', name: 'Oceania', nodes: 0, status: 'Offline', cx: 80, cy: 74, color: '#8B5CF6' }
  ];

  return (
    <div className={`relative w-full rounded-[24px] alien-glass-panel p-5 overflow-hidden flex flex-col justify-between ${heightClass} shadow-2xl`}>
      {/* Background Cosmic Atmosphere & Planetary Rim Light */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[340px] h-[340px] bg-gradient-to-tr from-blue-600/15 via-cyan-500/15 to-violet-600/10 rounded-full blur-3xl pointer-events-none -z-10" />

      {/* Header matching mm.png */}
      <div className="flex items-start justify-between z-10">
        <div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-xl bg-blue-600/20 border border-cyan-400/30 flex items-center justify-center text-cyan-400 shadow-[0_0_12px_rgba(32,221,247,0.3)]">
              <Globe className="w-4 h-4" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-bold text-white tracking-tight">Global Infrastructure</h3>
                <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-mono font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/25">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_6px_#34d399]" />
                  Live
                </span>
              </div>
              <p className="text-[11px] text-slate-400 font-mono mt-0.5">Your nodes, regions and edge locations</p>
            </div>
          </div>
        </div>

        {onViewAllNodes && (
          <button
            onClick={onViewAllNodes}
            className="flex items-center gap-1 text-xs font-semibold text-cyan-400 hover:text-cyan-300 transition-colors cursor-pointer group"
          >
            <span>View All Nodes</span>
            <ArrowUpRight className="w-3.5 h-3.5 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform" />
          </button>
        )}
      </div>

      {/* Holographic 3D Earth Vector Visualization */}
      <div className="relative flex-1 w-full my-2 flex items-center justify-center">
        <svg
          viewBox="0 0 800 380"
          className="w-full h-full max-h-[260px] overflow-visible"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            {/* Atmospheric Glow Gradients */}
            <radialGradient id="globeAura" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="#0B1C3D" stopOpacity="0.8" />
              <stop offset="70%" stopColor="#061226" stopOpacity="0.9" />
              <stop offset="92%" stopColor="#248BFF" stopOpacity="0.4" />
              <stop offset="100%" stopColor="#20DDF7" stopOpacity="0.9" />
            </radialGradient>

            <linearGradient id="orbitalGradient" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#20DDF7" stopOpacity="0.8" />
              <stop offset="50%" stopColor="#248BFF" stopOpacity="0.4" />
              <stop offset="100%" stopColor="#A855F7" stopOpacity="0.8" />
            </linearGradient>

            <linearGradient id="arcCyanViolet" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="#20DDF7" />
              <stop offset="100%" stopColor="#A855F7" />
            </linearGradient>

            <filter id="luminousGlow" x="-20%" y="-20%" width="140%" height="140%">
              <feGaussianBlur stdDeviation="3" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
          </defs>

          {/* Holographic Dimensional Globe Sphere */}
          <g transform="translate(400, 190)">
            {/* Ambient atmospheric outer halo */}
            <circle r="145" fill="none" stroke="rgba(32, 221, 247, 0.15)" strokeWidth="1" />
            <circle r="138" fill="none" stroke="rgba(36, 139, 255, 0.25)" strokeWidth="1.5" />
            <circle r="130" fill="url(#globeAura)" stroke="rgba(32, 221, 247, 0.4)" strokeWidth="1.5" filter="url(#luminousGlow)" />

            {/* Latitude & Longitude Spatial Grid */}
            <ellipse rx="130" ry="45" fill="none" stroke="rgba(125, 190, 255, 0.12)" strokeWidth="1" />
            <ellipse rx="130" ry="90" fill="none" stroke="rgba(125, 190, 255, 0.12)" strokeWidth="1" />
            <ellipse rx="45" ry="130" fill="none" stroke="rgba(125, 190, 255, 0.12)" strokeWidth="1" />
            <ellipse rx="90" ry="130" fill="none" stroke="rgba(125, 190, 255, 0.12)" strokeWidth="1" />
            <line x1="-130" y1="0" x2="130" y2="0" stroke="rgba(125, 190, 255, 0.2)" strokeWidth="1" strokeDasharray="3 3" />
            <line x1="0" y1="-130" x2="0" y2="130" stroke="rgba(125, 190, 255, 0.2)" strokeWidth="1" strokeDasharray="3 3" />

            {/* Orbital Rings around Earth */}
            <ellipse
              rx="180"
              ry="65"
              fill="none"
              stroke="url(#orbitalGradient)"
              strokeWidth="1.5"
              transform="rotate(-24)"
              strokeDasharray="6 4"
              opacity="0.7"
            />

            {/* Stylized Glowing Continents Silhouettes */}
            {/* North America */}
            <path
              d="M -75 -60 Q -50 -75, -20 -55 Q -10 -40, -35 -20 Q -65 -30, -75 -60 Z"
              fill="rgba(32, 221, 247, 0.22)"
              stroke="rgba(32, 221, 247, 0.5)"
              strokeWidth="1"
            />
            {/* South America */}
            <path
              d="M -30 -10 Q -15 0, -20 40 Q -35 60, -45 35 Q -40 10, -30 -10 Z"
              fill="rgba(36, 139, 255, 0.2)"
              stroke="rgba(36, 139, 255, 0.45)"
              strokeWidth="1"
            />
            {/* Europe */}
            <path
              d="M 5 -65 Q 35 -65, 40 -40 Q 20 -35, 10 -45 Q -5 -50, 5 -65 Z"
              fill="rgba(36, 139, 255, 0.25)"
              stroke="rgba(36, 139, 255, 0.6)"
              strokeWidth="1"
            />
            {/* Africa */}
            <path
              d="M 10 -30 Q 35 -25, 30 15 Q 15 45, 0 25 Q 5 -10, 10 -30 Z"
              fill="rgba(168, 85, 247, 0.18)"
              stroke="rgba(168, 85, 247, 0.4)"
              strokeWidth="1"
            />
            {/* Asia */}
            <path
              d="M 45 -55 Q 85 -55, 95 -25 Q 75 0, 50 -15 Q 40 -35, 45 -55 Z"
              fill="rgba(32, 221, 247, 0.25)"
              stroke="rgba(32, 221, 247, 0.55)"
              strokeWidth="1"
            />
            {/* Australia / Oceania */}
            <path
              d="M 70 30 Q 90 25, 95 45 Q 85 60, 68 50 Q 65 35, 70 30 Z"
              fill="rgba(89, 101, 255, 0.2)"
              stroke="rgba(89, 101, 255, 0.4)"
              strokeWidth="1"
            />

            {/* Glowing Luminous Network Arcs */}
            <path
              d="M -45 -45 Q 0 -85, 25 -50"
              fill="none"
              stroke="url(#arcCyanViolet)"
              strokeWidth="2"
              strokeDasharray="4 3"
              filter="url(#luminousGlow)"
            />
            <path
              d="M 25 -50 Q 55 -60, 70 -30"
              fill="none"
              stroke="#20DDF7"
              strokeWidth="1.8"
              strokeDasharray="4 3"
              filter="url(#luminousGlow)"
            />
            <path
              d="M -45 -45 Q -50 -10, -30 20"
              fill="none"
              stroke="#248BFF"
              strokeWidth="1.5"
              strokeDasharray="4 3"
              filter="url(#luminousGlow)"
            />
            <path
              d="M 25 -50 Q 45 10, 75 40"
              fill="none"
              stroke="#A855F7"
              strokeWidth="1.5"
              strokeDasharray="4 3"
              filter="url(#luminousGlow)"
            />

            {/* Glowing Active Node Pins */}
            <circle cx="-45" cy="-45" r="4.5" fill="#20DDF7" filter="url(#luminousGlow)" />
            <circle cx="-45" cy="-45" r="8" fill="none" stroke="#20DDF7" strokeWidth="1" opacity="0.6" className="animate-ping" />

            <circle cx="25" cy="-50" r="4.5" fill="#248BFF" filter="url(#luminousGlow)" />
            <circle cx="70" cy="-30" r="4.5" fill="#F59E0B" filter="url(#luminousGlow)" />
            <circle cx="-30" cy="20" r="4.5" fill="#EF4444" filter="url(#luminousGlow)" />
            <circle cx="75" cy="40" r="3.5" fill="#8B5CF6" opacity="0.5" />
          </g>
        </svg>

        {/* Floating Region Badges matching mm.png */}
        {regionPills.map((pill) => {
          const isHovered = hoveredRegion === pill.id;
          const isSelected = selectedRegion === pill.name;
          return (
            <div
              key={pill.id}
              onClick={() => onSelectRegion?.(pill.name)}
              onMouseEnter={() => setHoveredRegion(pill.id)}
              onMouseLeave={() => setHoveredRegion(null)}
              style={{ left: `${pill.cx}%`, top: `${pill.cy}%` }}
              className={`absolute -translate-x-1/2 -translate-y-1/2 z-20 cursor-pointer transition-all duration-300 ${
                isHovered || isSelected ? 'scale-110' : 'hover:scale-105'
              }`}
            >
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[rgba(8,20,44,0.85)] backdrop-blur-xl border border-[rgba(125,190,255,0.25)] shadow-[0_8px_20px_rgba(2,6,23,0.8)] hover:border-cyan-400/50 transition-colors">
                <span
                  className="w-2.5 h-2.5 rounded-full"
                  style={{
                    backgroundColor: pill.color,
                    boxShadow: `0 0 10px ${pill.color}`
                  }}
                />
                <div className="flex items-center gap-1.5 text-xs font-semibold font-sans">
                  <span className="text-white">{pill.name}</span>
                  <span className="text-[10px] text-slate-400 font-mono font-normal">
                    {pill.nodes} {pill.nodes === 1 ? 'node' : 'nodes'}
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Bottom Status Counter Strip matching mm.png */}
      <div className="flex items-center justify-between pt-3 border-t border-[rgba(125,190,255,0.12)] text-xs font-mono z-10 px-1">
        <div className="flex flex-wrap items-center gap-6">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_8px_#34d399]" />
            <span className="text-slate-300">Online</span>
            <span className="font-bold text-white">{onlineCount}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-amber-400 shadow-[0_0_8px_#fbbf24]" />
            <span className="text-slate-300">Degraded</span>
            <span className="font-bold text-white">{degradedCount}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-rose-500 shadow-[0_0_8px_#f43f5e]" />
            <span className="text-slate-300">Offline</span>
            <span className="font-bold text-white">{offlineCount}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-slate-500" />
            <span className="text-slate-400">Unknown</span>
            <span className="font-bold text-white">{unknownCount}</span>
          </div>
        </div>

        <div className="text-[11px] text-slate-500 font-mono hidden md:block">
          Mesh consensus: Active epoch #829
        </div>
      </div>
    </div>
  );
};
