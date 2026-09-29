import React, { useRef, useEffect, useState } from 'react';
import { NodeItem } from '../types';

interface GlobeMapProps {
  nodes?: NodeItem[];
  title?: string;
  subtitle?: string;
  activeFilter?: string;
  onFilterChange?: (filter: string) => void;
  showLayerButtons?: boolean;
  filterOptions?: { id: string; label: string; count?: number }[];
  mode?: 'nodes' | 'storage' | 'deploy';
}

export const GlobeMap: React.FC<GlobeMapProps> = ({
  nodes = [],
  title = 'Node Network',
  subtitle = 'Real-time view of your global infrastructure',
  activeFilter = 'all',
  onFilterChange,
  showLayerButtons = true,
  filterOptions = [
    { id: 'all', label: 'All Nodes', count: 8 },
    { id: 'my-nodes', label: 'My Nodes', count: 4 },
    { id: 'community', label: 'Community', count: 3 },
    { id: 'depin', label: 'DePIN', count: 1 },
  ],
  mode = 'nodes',
}) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [selectedNode, setSelectedNode] = useState<any | null>(null);
  const [hoveredRegion, setHoveredRegion] = useState<string | null>(null);

  // Region hubs with coordinates for the map projection
  const regions = [
    { id: 'na', name: 'North America', count: 2, status: 'online', x: 0.26, y: 0.38, color: '#38bdf8' },
    { id: 'eu', name: 'Europe', count: 3, status: 'online', x: 0.52, y: 0.34, color: '#a855f7' },
    { id: 'as', name: 'Asia', count: 2, status: 'online', x: 0.74, y: 0.44, color: '#10b981' },
    { id: 'sa', name: 'South America', count: 1, status: 'degraded', x: 0.35, y: 0.70, color: '#f59e0b' },
    { id: 'af', name: 'Africa', count: 0, status: 'offline', x: 0.53, y: 0.58, color: '#64748b' },
    { id: 'oc', name: 'Oceania', count: 0, status: 'unknown', x: 0.85, y: 0.74, color: '#64748b' },
  ];

  // Canvas animation loop
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;
    let rotation = 0;

    const render = () => {
      rotation += 0.003;
      const width = canvas.width;
      const height = canvas.height;

      ctx.clearRect(0, 0, width, height);

      // Deep space starry dust background
      const centerX = width * 0.5;
      const centerY = height * 0.52;
      const radius = Math.min(width, height) * 0.38;

      // Glow behind globe
      const glowGrad = ctx.createRadialGradient(centerX, centerY, radius * 0.3, centerX, centerY, radius * 1.3);
      glowGrad.addColorStop(0, 'rgba(30, 58, 138, 0.35)');
      glowGrad.addColorStop(0.5, 'rgba(14, 116, 144, 0.15)');
      glowGrad.addColorStop(1, 'rgba(6, 11, 24, 0)');
      ctx.fillStyle = glowGrad;
      ctx.fillRect(0, 0, width, height);

      // Sphere base
      const sphereGrad = ctx.createRadialGradient(
        centerX - radius * 0.3,
        centerY - radius * 0.3,
        radius * 0.1,
        centerX,
        centerY,
        radius
      );
      sphereGrad.addColorStop(0, '#0f2744');
      sphereGrad.addColorStop(0.6, '#081426');
      sphereGrad.addColorStop(1, '#040914');

      ctx.beginPath();
      ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
      ctx.fillStyle = sphereGrad;
      ctx.fill();
      ctx.lineWidth = 1.5;
      ctx.strokeStyle = 'rgba(56, 189, 248, 0.4)';
      ctx.stroke();

      // Atmospheric limb glow
      ctx.save();
      ctx.beginPath();
      ctx.arc(centerX, centerY, radius + 2, 0, Math.PI * 2);
      ctx.strokeStyle = 'rgba(14, 165, 233, 0.25)';
      ctx.lineWidth = 4;
      ctx.stroke();
      ctx.restore();

      // Latitude and Longitude grid rings
      ctx.strokeStyle = 'rgba(56, 189, 248, 0.12)';
      ctx.lineWidth = 0.8;

      for (let lat = -60; lat <= 60; lat += 30) {
        const radLat = (lat * Math.PI) / 180;
        const yOffset = Math.sin(radLat) * radius;
        const rLat = Math.cos(radLat) * radius;
        ctx.beginPath();
        ctx.ellipse(centerX, centerY + yOffset, rLat, rLat * 0.28, 0, 0, Math.PI * 2);
        ctx.stroke();
      }

      // Meridians with rotation
      for (let lng = 0; lng < 360; lng += 45) {
        const radLng = ((lng + rotation * 80) * Math.PI) / 180;
        const xOffset = Math.sin(radLng) * radius;
        ctx.beginPath();
        ctx.ellipse(centerX + xOffset * 0.45, centerY, Math.abs(xOffset * 0.55), radius, 0, 0, Math.PI * 2);
        ctx.stroke();
      }

      // Draw orbital connection arcs between active hubs
      const hubs = [
        { x: centerX - radius * 0.55, y: centerY - radius * 0.25, name: 'North America' },
        { x: centerX + radius * 0.05, y: centerY - radius * 0.35, name: 'Europe' },
        { x: centerX + radius * 0.52, y: centerY - radius * 0.1, name: 'Asia' },
        { x: centerX - radius * 0.35, y: centerY + radius * 0.42, name: 'South America' },
      ];

      // Connecting curves with animated glowing particles
      ctx.lineWidth = 1.2;
      for (let i = 0; i < hubs.length; i++) {
        for (let j = i + 1; j < hubs.length; j++) {
          const h1 = hubs[i];
          const h2 = hubs[j];

          const arcGrad = ctx.createLinearGradient(h1.x, h1.y, h2.x, h2.y);
          arcGrad.addColorStop(0, 'rgba(56, 189, 248, 0.4)');
          arcGrad.addColorStop(0.5, 'rgba(168, 85, 247, 0.6)');
          arcGrad.addColorStop(1, 'rgba(16, 185, 129, 0.4)');

          ctx.strokeStyle = arcGrad;
          ctx.beginPath();
          ctx.moveTo(h1.x, h1.y);
          // quadratic curve through control point elevated above
          const midX = (h1.x + h2.x) / 2;
          const midY = (h1.y + h2.y) / 2 - 40;
          ctx.quadraticCurveTo(midX, midY, h2.x, h2.y);
          ctx.stroke();

          // Particle travel
          const t = (Date.now() / 2000 + (i + j) * 0.25) % 1;
          const px = (1 - t) * (1 - t) * h1.x + 2 * (1 - t) * t * midX + t * t * h2.x;
          const py = (1 - t) * (1 - t) * h1.y + 2 * (1 - t) * t * midY + t * t * h2.y;

          ctx.beginPath();
          ctx.arc(px, py, 2.5, 0, Math.PI * 2);
          ctx.fillStyle = '#00f2fe';
          ctx.shadowColor = '#00f2fe';
          ctx.shadowBlur = 8;
          ctx.fill();
          ctx.shadowBlur = 0;
        }
      }

      // Draw node beacons on globe
      hubs.forEach((hub, idx) => {
        const pulse = (Math.sin(Date.now() / 400 + idx) + 1) / 2;
        const color = idx === 3 ? '#f59e0b' : idx === 1 ? '#c084fc' : '#38bdf8';

        // Outer ripple
        ctx.beginPath();
        ctx.arc(hub.x, hub.y, 8 + pulse * 6, 0, Math.PI * 2);
        ctx.strokeStyle = color;
        ctx.globalAlpha = 0.6 - pulse * 0.4;
        ctx.stroke();
        ctx.globalAlpha = 1;

        // Inner glowing dot
        ctx.beginPath();
        ctx.arc(hub.x, hub.y, 4.5, 0, Math.PI * 2);
        ctx.fillStyle = color;
        ctx.shadowColor = color;
        ctx.shadowBlur = 10;
        ctx.fill();
        ctx.shadowBlur = 0;
      });

      animationFrameId = requestAnimationFrame(render);
    };

    render();

    return () => cancelAnimationFrame(animationFrameId);
  }, []);

  return (
    <div className="relative overflow-hidden rounded-2xl bg-gradient-to-b from-[#0c1630]/90 to-[#070d1e]/90 border border-blue-500/25 p-5 shadow-2xl backdrop-blur-xl">
      {/* Background neon ambient */}
      <div className="absolute top-0 right-0 w-80 h-80 bg-blue-600/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-0 left-0 w-80 h-80 bg-purple-600/10 rounded-full blur-3xl pointer-events-none" />

      {/* Card Header matching screenshots */}
      <div className="relative z-10 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-3 border-b border-blue-500/15">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-base font-bold text-white tracking-tight flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-cyan-400 shadow-[0_0_8px_#22d3ee] animate-pulse" />
              {title}
            </h3>
            <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
              Live Mesh
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">{subtitle}</p>
        </div>

        {/* Layer / Filter buttons matching screenshots */}
        {showLayerButtons && (
          <div className="flex flex-wrap items-center gap-1.5 p-1 rounded-xl bg-slate-900/80 border border-blue-500/20">
            {filterOptions.map((opt) => (
              <button
                key={opt.id}
                onClick={() => onFilterChange && onFilterChange(opt.id)}
                className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 ${
                  activeFilter === opt.id
                    ? 'bg-gradient-to-r from-blue-600 to-cyan-600 text-white shadow-md shadow-blue-500/30'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                }`}
              >
                <span>{opt.label}</span>
                {opt.count !== undefined && (
                  <span className={`text-[10px] px-1.5 rounded-full ${
                    activeFilter === opt.id ? 'bg-black/20 text-white' : 'bg-slate-800 text-slate-400'
                  }`}>
                    {opt.count}
                  </span>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Canvas Area with floating Region badges */}
      <div className="relative w-full h-[320px] md:h-[360px] flex items-center justify-center my-2">
        <canvas
          ref={canvasRef}
          width={800}
          height={400}
          className="w-full h-full max-h-[360px] object-contain"
        />

        {/* Floating Region Badges matching screenshots */}
        {/* North America */}
        <div className="absolute top-6 left-6 md:left-12 group cursor-pointer">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#09142b]/85 border border-cyan-500/40 shadow-lg shadow-cyan-500/10 backdrop-blur-md hover:border-cyan-400 transition-all">
            <span className="w-2 h-2 rounded-full bg-cyan-400 shadow-[0_0_6px_#22d3ee]" />
            <div className="text-left">
              <p className="text-[11px] font-semibold text-slate-200">North America</p>
              <p className="text-[10px] text-cyan-400 font-mono">2 nodes</p>
            </div>
          </div>
        </div>

        {/* Europe */}
        <div className="absolute top-4 right-1/3 group cursor-pointer">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#09142b]/85 border border-purple-500/40 shadow-lg shadow-purple-500/10 backdrop-blur-md hover:border-purple-400 transition-all">
            <span className="w-2 h-2 rounded-full bg-purple-400 shadow-[0_0_6px_#c084fc]" />
            <div className="text-left">
              <p className="text-[11px] font-semibold text-slate-200">Europe</p>
              <p className="text-[10px] text-purple-300 font-mono">3 nodes</p>
            </div>
          </div>
        </div>

        {/* Asia */}
        <div className="absolute top-16 right-6 md:right-16 group cursor-pointer">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#09142b]/85 border border-emerald-500/40 shadow-lg shadow-emerald-500/10 backdrop-blur-md hover:border-emerald-400 transition-all">
            <span className="w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_6px_#34d399]" />
            <div className="text-left">
              <p className="text-[11px] font-semibold text-slate-200">Asia</p>
              <p className="text-[10px] text-emerald-400 font-mono">2 nodes</p>
            </div>
          </div>
        </div>

        {/* South America */}
        <div className="absolute bottom-10 left-1/4 group cursor-pointer">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#09142b]/85 border border-amber-500/40 shadow-lg shadow-amber-500/10 backdrop-blur-md hover:border-amber-400 transition-all">
            <span className="w-2 h-2 rounded-full bg-amber-400 shadow-[0_0_6px_#fbbf24]" />
            <div className="text-left">
              <p className="text-[11px] font-semibold text-slate-200">South America</p>
              <p className="text-[10px] text-amber-300 font-mono">1 node (Degraded)</p>
            </div>
          </div>
        </div>

        {/* Africa */}
        <div className="absolute bottom-8 right-1/3 group cursor-pointer opacity-70">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#09142b]/85 border border-slate-700/60 shadow-lg backdrop-blur-md">
            <span className="w-2 h-2 rounded-full bg-slate-500" />
            <div className="text-left">
              <p className="text-[11px] font-semibold text-slate-300">Africa</p>
              <p className="text-[10px] text-slate-500 font-mono">0 nodes</p>
            </div>
          </div>
        </div>

        {/* Global Network stats overlay on the right matching screenshots */}
        <div className="absolute top-3 right-3 hidden lg:flex flex-col gap-1.5 p-3 rounded-2xl bg-[#060e22]/90 border border-blue-500/30 backdrop-blur-xl shadow-xl">
          <div className="flex items-center justify-between gap-4">
            <span className="text-[11px] text-slate-400 font-medium">Status Breakdown</span>
          </div>
          <div className="space-y-1 text-xs font-mono">
            <div className="flex items-center justify-between gap-3 text-slate-300">
              <span className="flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-emerald-400" /> Online
              </span>
              <span className="font-bold text-white">8</span>
            </div>
            <div className="flex items-center justify-between gap-3 text-slate-300">
              <span className="flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-amber-400" /> Degraded
              </span>
              <span className="font-bold text-amber-400">1</span>
            </div>
            <div className="flex items-center justify-between gap-3 text-slate-300">
              <span className="flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-rose-500" /> Offline
              </span>
              <span className="font-bold text-rose-400">1</span>
            </div>
            <div className="flex items-center justify-between gap-3 text-slate-300">
              <span className="flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-slate-500" /> Unknown
              </span>
              <span className="font-bold text-slate-400">0</span>
            </div>
          </div>
        </div>
      </div>

      {/* Bottom Region Breakdown Pills matching screenshot */}
      <div className="relative z-10 pt-3 border-t border-blue-500/15 flex flex-wrap items-center justify-between gap-2 text-xs">
        <div className="flex flex-wrap items-center gap-2">
          {regions.map((reg) => (
            <div
              key={reg.id}
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-900/60 border ${
                reg.count > 0 ? 'border-blue-500/25 text-slate-200' : 'border-slate-800 text-slate-500'
              }`}
            >
              <span
                className="w-1.5 h-1.5 rounded-full"
                style={{ backgroundColor: reg.color }}
              />
              <span className="font-medium text-[11px]">{reg.name}</span>
              <span className="font-mono text-[10px] text-slate-400">{reg.count} nodes</span>
            </div>
          ))}
        </div>

        <div className="flex items-center gap-2 text-[11px] text-cyan-400 font-mono">
          <span>WireGuard Mesh: Healthy</span>
          <span>•</span>
          <span>Raft Quorum: 5/5</span>
        </div>
      </div>
    </div>
  );
};
