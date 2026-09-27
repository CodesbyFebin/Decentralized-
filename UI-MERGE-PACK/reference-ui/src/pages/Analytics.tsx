import React from 'react';
import { BarChart3, TrendingUp, Activity, Cpu, HardDrive, Globe2, Radio } from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Analytics: React.FC = () => {
  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <BarChart3 className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
            Telemetry & Metrics
          </span>
        </div>
        <h1 className="text-2xl font-bold text-white tracking-tight">
          Cluster Analytics & Performance
        </h1>
        <p className="text-xs text-slate-300 mt-1">
          Real-time metrics aggregated across edge nodes, WireGuard transit tunnels, and storage pools.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 font-mono">
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25">
          <span className="text-xs text-slate-400 font-sans block mb-1">Total Requests (30d)</span>
          <span className="text-2xl font-extrabold text-white">1,248,910</span>
          <span className="text-xs text-emerald-400 block mt-1">↑ 18.4% vs last month</span>
        </div>
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25">
          <span className="text-xs text-slate-400 font-sans block mb-1">Avg Global Edge Latency</span>
          <span className="text-2xl font-extrabold text-cyan-300">18.2 ms</span>
          <span className="text-xs text-cyan-400 block mt-1">Anycast optimized</span>
        </div>
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25">
          <span className="text-xs text-slate-400 font-sans block mb-1">Bandwidth Served</span>
          <span className="text-2xl font-extrabold text-purple-300">4.8 TB</span>
          <span className="text-xs text-purple-400 block mt-1">Zero cloud egress fee</span>
        </div>
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25">
          <span className="text-xs text-slate-400 font-sans block mb-1">SLA Uptime Ratio</span>
          <span className="text-2xl font-extrabold text-emerald-400">99.98%</span>
          <span className="text-xs text-emerald-400 block mt-1">Failure-domain resilient</span>
        </div>
      </div>

      {/* Latency by region */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl space-y-4">
        <h3 className="text-sm font-bold text-white font-mono">Global Edge Ingress Latency</h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 font-mono text-xs">
          <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800">
            <span className="text-slate-400 font-sans block mb-1">Asia (Kochi, Tokyo, Singapore)</span>
            <span className="text-base font-bold text-cyan-300">12ms • AS55836</span>
          </div>
          <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800">
            <span className="text-slate-400 font-sans block mb-1">Europe (Frankfurt, Stockholm)</span>
            <span className="text-base font-bold text-purple-300">22ms • AS24940</span>
          </div>
          <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800">
            <span className="text-slate-400 font-sans block mb-1">North America (San Francisco, NYC)</span>
            <span className="text-base font-bold text-emerald-300">28ms • AS13335</span>
          </div>
        </div>
      </div>
    </div>
  );
};
