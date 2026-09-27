import React from 'react';
import {
  Rocket,
  Globe2,
  Server,
  FileText,
  Bot,
  ArrowUpRight,
  TrendingUp,
  HardDrive,
  ShieldCheck,
  CheckCircle2,
  Clock,
  Sparkles,
  ExternalLink,
  ChevronRight,
  MoreVertical,
  Activity,
  Layers
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';
import { GlobeMap } from '../components/GlobeMap';

export const Dashboard: React.FC = () => {
  const {
    clusterStats,
    nodes,
    deployments,
    activities,
    setCurrentTab,
    setIsNewDeploymentModalOpen,
    setIsAddNodeModalOpen,
    setSelectedNodeId,
    setSelectedDeploymentId,
  } = useNetwork();

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      {/* Welcome Banner matching screenshot */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 p-6 shadow-2xl backdrop-blur-xl">
        {/* Glow accents */}
        <div className="absolute top-0 right-1/4 w-96 h-96 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute -bottom-10 right-0 w-80 h-80 bg-purple-600/10 rounded-full blur-3xl pointer-events-none" />

        <div className="relative z-10 flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4">
          <div>
            <p className="text-xs font-semibold text-cyan-400 tracking-wider font-mono">
              Wednesday, Sep 24, 2026
            </p>
            <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight mt-1 flex items-center gap-2">
              Welcome back, Febin <span className="inline-block animate-bounce">👋</span>
            </h1>
            <p className="text-xs sm:text-sm text-slate-300 mt-1">
              Your decentralized infrastructure is running smoothly.
            </p>
          </div>

          {/* Quick Action Pills matching screenshot */}
          <div className="flex flex-wrap items-center gap-2.5">
            <button
              onClick={() => setIsNewDeploymentModalOpen(true)}
              className="px-3.5 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-blue-600/30 transition-all flex items-center gap-1.5"
            >
              <Rocket className="w-3.5 h-3.5" />
              <span>Deploy App</span>
            </button>

            <button
              onClick={() => setCurrentTab('domains')}
              className="px-3.5 py-2 rounded-xl text-xs font-semibold text-purple-200 bg-purple-950/60 hover:bg-purple-900/60 border border-purple-500/30 shadow-md transition-all flex items-center gap-1.5"
            >
              <Globe2 className="w-3.5 h-3.5 text-purple-400" />
              <span>Add Domain</span>
            </button>

            <button
              onClick={() => setIsAddNodeModalOpen(true)}
              className="px-3.5 py-2 rounded-xl text-xs font-semibold text-cyan-200 bg-cyan-950/60 hover:bg-cyan-900/60 border border-cyan-500/30 shadow-md transition-all flex items-center gap-1.5"
            >
              <Server className="w-3.5 h-3.5 text-cyan-400" />
              <span>Manage Nodes</span>
            </button>

            <button
              onClick={() => setCurrentTab('analytics')}
              className="px-3.5 py-2 rounded-xl text-xs font-semibold text-amber-200 bg-amber-950/50 hover:bg-amber-900/50 border border-amber-500/30 shadow-md transition-all flex items-center gap-1.5"
            >
              <FileText className="w-3.5 h-3.5 text-amber-400" />
              <span>View Logs</span>
            </button>

            <button
              onClick={() => setCurrentTab('rag-copilot')}
              className="px-3.5 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 border border-purple-400/40 shadow-lg shadow-purple-600/30 transition-all flex items-center gap-1.5 group"
            >
              <Sparkles className="w-3.5 h-3.5 text-cyan-300 animate-pulse" />
              <span>Ask Copilot</span>
              <ChevronRight className="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" />
            </button>
          </div>
        </div>
      </div>

      {/* 5 Stat Cards matching screenshot */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3.5">
        {/* Applications */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl relative overflow-hidden group hover:border-cyan-400/40 transition-all">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400">Applications</span>
            <div className="p-2 rounded-xl bg-cyan-500/15 text-cyan-400">
              <Rocket className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-2xl font-extrabold text-white font-mono">{deployments.length}</span>
            <span className="text-xs font-bold text-cyan-400 font-mono flex items-center">
              ↑ +20%
            </span>
          </div>
          {/* Sparkline curve */}
          <div className="h-6 w-full mt-2">
            <svg viewBox="0 0 100 24" className="w-full h-full text-cyan-400" fill="none">
              <path d="M0 20 Q 25 18, 50 10 T 100 4" stroke="currentColor" strokeWidth="2" />
            </svg>
          </div>
        </div>

        {/* Nodes Online */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl relative overflow-hidden group hover:border-emerald-400/40 transition-all">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400">Nodes Online</span>
            <div className="p-2 rounded-xl bg-emerald-500/15 text-emerald-400">
              <Server className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-2xl font-extrabold text-white font-mono">
              {nodes.filter((n) => n.status === 'Online').length} <span className="text-sm text-slate-400 font-normal">/ {nodes.length}</span>
            </span>
            <span className="text-xs font-bold text-emerald-400 font-mono flex items-center">
              ↑ +12%
            </span>
          </div>
          <div className="h-6 w-full mt-2">
            <svg viewBox="0 0 100 24" className="w-full h-full text-emerald-400" fill="none">
              <path d="M0 18 Q 30 15, 60 8 T 100 3" stroke="currentColor" strokeWidth="2" />
            </svg>
          </div>
        </div>

        {/* Storage Used */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl relative overflow-hidden group hover:border-purple-400/40 transition-all">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400">Storage Used</span>
            <div className="p-2 rounded-xl bg-purple-500/15 text-purple-400">
              <HardDrive className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-2xl font-extrabold text-white font-mono">428 GB</span>
            <span className="text-xs font-bold text-purple-400 font-mono flex items-center">
              ↑ +18%
            </span>
          </div>
          <div className="h-6 w-full mt-2">
            <svg viewBox="0 0 100 24" className="w-full h-full text-purple-400" fill="none">
              <path d="M0 22 Q 35 16, 70 12 T 100 6" stroke="currentColor" strokeWidth="2" />
            </svg>
          </div>
        </div>

        {/* Domains */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl relative overflow-hidden group hover:border-amber-400/40 transition-all">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400">Domains</span>
            <div className="p-2 rounded-xl bg-amber-500/15 text-amber-400">
              <Globe2 className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-2xl font-extrabold text-white font-mono">8</span>
            <span className="text-xs font-bold text-amber-400 font-mono flex items-center">
              ↑ +14%
            </span>
          </div>
          <div className="h-6 w-full mt-2">
            <svg viewBox="0 0 100 24" className="w-full h-full text-amber-400" fill="none">
              <path d="M0 16 Q 40 14, 75 9 T 100 5" stroke="currentColor" strokeWidth="2" />
            </svg>
          </div>
        </div>

        {/* SSL Certificates */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl relative overflow-hidden group hover:border-cyan-400/40 transition-all col-span-2 sm:col-span-1">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400">SSL Certificates</span>
            <div className="p-2 rounded-xl bg-blue-500/15 text-blue-400">
              <ShieldCheck className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-2xl font-extrabold text-white font-mono">6</span>
            <span className="text-xs font-bold text-cyan-400 font-mono flex items-center">
              ↑ +33%
            </span>
          </div>
          <div className="h-6 w-full mt-2">
            <svg viewBox="0 0 100 24" className="w-full h-full text-blue-400" fill="none">
              <path d="M0 19 Q 20 17, 60 11 T 100 4" stroke="currentColor" strokeWidth="2" />
            </svg>
          </div>
        </div>
      </div>

      {/* Middle Row: Globe Map + Resource Usage + RAG Copilot card matching screenshots */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        {/* Left: Holographic Node Network Globe (7 cols) */}
        <div className="lg:col-span-7">
          <GlobeMap
            title="Node Network"
            subtitle="Real-time view of your global infrastructure"
            showLayerButtons={false}
          />
        </div>

        {/* Right Stack: Resource Usage & RAG Copilot (5 cols) */}
        <div className="lg:col-span-5 space-y-5">
          {/* Resource Usage with Radial Meters & Wave Chart matching screenshot */}
          <div className="p-5 rounded-2xl bg-gradient-to-b from-[#0c1630]/90 to-[#070d1e]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
            <div className="flex items-center justify-between mb-4 pb-2 border-b border-blue-500/15">
              <h3 className="text-sm font-bold text-white tracking-tight flex items-center gap-2">
                <Activity className="w-4 h-4 text-cyan-400" />
                <span>Resource Usage</span>
              </h3>
              <span className="text-xs font-mono text-slate-400 bg-slate-900 px-2 py-0.5 rounded-lg border border-slate-800">
                Last 24 hours
              </span>
            </div>

            {/* 4 Circular / Radial progress meters matching screenshot */}
            <div className="grid grid-cols-4 gap-2 text-center my-3">
              {/* CPU */}
              <div className="flex flex-col items-center">
                <div className="relative w-14 h-14 flex items-center justify-center">
                  <svg className="w-full h-full -rotate-90" viewBox="0 0 36 36">
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#1e293b"
                      strokeWidth="3.2"
                    />
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#38bdf8"
                      strokeWidth="3.2"
                      strokeDasharray="28, 100"
                    />
                  </svg>
                  <span className="absolute text-xs font-bold font-mono text-white">28%</span>
                </div>
                <span className="text-[11px] font-semibold text-slate-300 mt-1">CPU</span>
              </div>

              {/* Memory */}
              <div className="flex flex-col items-center">
                <div className="relative w-14 h-14 flex items-center justify-center">
                  <svg className="w-full h-full -rotate-90" viewBox="0 0 36 36">
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#1e293b"
                      strokeWidth="3.2"
                    />
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#c084fc"
                      strokeWidth="3.2"
                      strokeDasharray="46, 100"
                    />
                  </svg>
                  <span className="absolute text-xs font-bold font-mono text-white">46%</span>
                </div>
                <span className="text-[11px] font-semibold text-slate-300 mt-1">Memory</span>
              </div>

              {/* Storage */}
              <div className="flex flex-col items-center">
                <div className="relative w-14 h-14 flex items-center justify-center">
                  <svg className="w-full h-full -rotate-90" viewBox="0 0 36 36">
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#1e293b"
                      strokeWidth="3.2"
                    />
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#fbbf24"
                      strokeWidth="3.2"
                      strokeDasharray="42, 100"
                    />
                  </svg>
                  <span className="absolute text-xs font-bold font-mono text-white">42%</span>
                </div>
                <span className="text-[11px] font-semibold text-slate-300 mt-1">Storage</span>
              </div>

              {/* Network */}
              <div className="flex flex-col items-center">
                <div className="relative w-14 h-14 flex items-center justify-center">
                  <svg className="w-full h-full -rotate-90" viewBox="0 0 36 36">
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#1e293b"
                      strokeWidth="3.2"
                    />
                    <path
                      d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      fill="none"
                      stroke="#34d399"
                      strokeWidth="3.2"
                      strokeDasharray="52, 100"
                    />
                  </svg>
                  <span className="absolute text-xs font-bold font-mono text-white">52%</span>
                </div>
                <span className="text-[11px] font-semibold text-slate-300 mt-1">Network</span>
              </div>
            </div>

            {/* Glowing wave chart matching screenshot */}
            <div className="relative h-28 w-full mt-4 pt-2">
              <svg viewBox="0 0 400 90" className="w-full h-full overflow-visible">
                <defs>
                  <linearGradient id="wave-grad-cyan" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#38bdf8" stopOpacity="0.4" />
                    <stop offset="100%" stopColor="#38bdf8" stopOpacity="0.0" />
                  </linearGradient>
                  <linearGradient id="wave-grad-purple" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#c084fc" stopOpacity="0.3" />
                    <stop offset="100%" stopColor="#c084fc" stopOpacity="0.0" />
                  </linearGradient>
                </defs>
                {/* Horizontal reference grid */}
                <line x1="0" y1="80" x2="400" y2="80" stroke="rgba(255,255,255,0.06)" />
                <line x1="0" y1="40" x2="400" y2="40" stroke="rgba(255,255,255,0.06)" />

                {/* Purple fill & line */}
                <path
                  d="M 0 65 Q 80 40, 150 70 T 300 35 T 400 60 L 400 85 L 0 85 Z"
                  fill="url(#wave-grad-purple)"
                />
                <path
                  d="M 0 65 Q 80 40, 150 70 T 300 35 T 400 60"
                  fill="none"
                  stroke="#c084fc"
                  strokeWidth="2.5"
                />

                {/* Cyan fill & line */}
                <path
                  d="M 0 75 Q 90 20, 180 55 T 320 25 T 400 45 L 400 85 L 0 85 Z"
                  fill="url(#wave-grad-cyan)"
                />
                <path
                  d="M 0 75 Q 90 20, 180 55 T 320 25 T 400 45"
                  fill="none"
                  stroke="#38bdf8"
                  strokeWidth="2.5"
                />
              </svg>
              <div className="flex justify-between text-[10px] text-slate-500 font-mono mt-1 px-1">
                <span>00:00</span>
                <span>06:00</span>
                <span>12:00</span>
                <span>18:00</span>
                <span>24:00</span>
              </div>
            </div>
          </div>

          {/* RAG Copilot Assistant Card matching screenshot */}
          <div className="p-5 rounded-2xl bg-gradient-to-b from-[#101b3d]/90 via-[#0a142c]/90 to-[#070d1e] border border-purple-500/30 shadow-2xl backdrop-blur-xl relative overflow-hidden">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <div className="relative w-10 h-10 rounded-2xl bg-gradient-to-tr from-purple-600 via-indigo-600 to-cyan-400 p-[1px] shadow-lg shadow-purple-500/30">
                  <div className="w-full h-full bg-[#091124] rounded-2xl flex items-center justify-center">
                    <Bot className="w-5 h-5 text-cyan-300 animate-pulse" />
                  </div>
                </div>
                <div>
                  <h3 className="text-sm font-bold text-white tracking-tight flex items-center gap-1.5">
                    RAG Copilot
                    <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-300 border border-purple-500/30">
                      Gemini 3
                    </span>
                  </h3>
                  <p className="text-[11px] text-slate-400">Your infrastructure assistant</p>
                </div>
              </div>
              <ChevronRight className="w-4 h-4 text-purple-400" />
            </div>

            {/* 4 Action Prompts matching screenshot */}
            <div className="space-y-1.5 my-3 text-xs">
              <button
                onClick={() => setCurrentTab('rag-copilot')}
                className="w-full flex items-center gap-2.5 p-2 rounded-xl bg-slate-900/60 hover:bg-slate-800/80 border border-slate-800 text-left transition-colors"
              >
                <div className="p-1 rounded-lg bg-blue-500/15 text-cyan-400">
                  <Sparkles className="w-3.5 h-3.5" />
                </div>
                <div>
                  <p className="font-semibold text-slate-200">Ask a question</p>
                  <p className="text-[10px] text-slate-400">Get insights from your platform</p>
                </div>
              </button>

              <button
                onClick={() => setCurrentTab('rag-copilot')}
                className="w-full flex items-center gap-2.5 p-2 rounded-xl bg-slate-900/60 hover:bg-slate-800/80 border border-slate-800 text-left transition-colors"
              >
                <div className="p-1 rounded-lg bg-purple-500/15 text-purple-400">
                  <Activity className="w-3.5 h-3.5" />
                </div>
                <div>
                  <p className="font-semibold text-slate-200">Diagnose an issue</p>
                  <p className="text-[10px] text-slate-400">Analyze logs and events with High Thinking</p>
                </div>
              </button>

              <button
                onClick={() => setCurrentTab('rag-copilot')}
                className="w-full flex items-center gap-2.5 p-2 rounded-xl bg-slate-900/60 hover:bg-slate-800/80 border border-slate-800 text-left transition-colors"
              >
                <div className="p-1 rounded-lg bg-emerald-500/15 text-emerald-400">
                  <Layers className="w-3.5 h-3.5" />
                </div>
                <div>
                  <p className="font-semibold text-slate-200">Plan an improvement</p>
                  <p className="text-[10px] text-slate-400">Multi-domain placement recommendations</p>
                </div>
              </button>
            </div>

            <button
              onClick={() => setCurrentTab('rag-copilot')}
              className="w-full py-2.5 px-4 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-purple-600 via-indigo-600 to-cyan-500 hover:from-purple-500 hover:to-cyan-400 shadow-lg shadow-purple-600/30 transition-all flex items-center justify-center gap-1.5"
            >
              <span>Start a conversation</span>
              <ChevronRight className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>

      {/* Bottom Row: Recent Activity (4 cols) + Top Applications (5 cols) + System Health (3 cols) matching screenshot */}
      <div className="grid grid-cols-1 md:grid-cols-12 gap-5">
        {/* Recent Activity (4 cols) */}
        <div className="md:col-span-4 p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between pb-3 border-b border-blue-500/15 mb-3">
            <h3 className="text-sm font-bold text-white tracking-tight">Recent Activity</h3>
            <span className="text-[11px] font-medium text-cyan-400 hover:underline cursor-pointer">
              View All →
            </span>
          </div>

          <div className="space-y-3">
            {activities.slice(0, 5).map((act) => (
              <div key={act.id} className="flex items-start justify-between gap-2 text-xs">
                <div className="flex items-start gap-2.5">
                  <div className="p-1.5 rounded-lg bg-blue-500/10 text-cyan-400 mt-0.5">
                    {act.type === 'deploy' ? (
                      <Rocket className="w-3.5 h-3.5" />
                    ) : act.type === 'node' ? (
                      <Server className="w-3.5 h-3.5" />
                    ) : act.type === 'ssl' ? (
                      <ShieldCheck className="w-3.5 h-3.5 text-purple-400" />
                    ) : (
                      <Globe2 className="w-3.5 h-3.5 text-amber-400" />
                    )}
                  </div>
                  <div>
                    <p className="font-semibold text-slate-200">{act.title}</p>
                    <p className="text-[11px] text-slate-400 font-mono">{act.target}</p>
                  </div>
                </div>

                <div className="text-right">
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                    ● {act.status}
                  </span>
                  <p className="text-[10px] text-slate-500 mt-1">{act.time}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Top Applications (5 cols) */}
        <div className="md:col-span-5 p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between pb-3 border-b border-blue-500/15 mb-3">
            <h3 className="text-sm font-bold text-white tracking-tight">Top Applications</h3>
            <button
              onClick={() => setCurrentTab('websites')}
              className="text-[11px] font-medium text-cyan-400 hover:underline"
            >
              View All →
            </button>
          </div>

          <div className="space-y-2">
            {deployments.slice(0, 5).map((app) => (
              <div
                key={app.id}
                onClick={() => setSelectedDeploymentId(app.id)}
                className="flex items-center justify-between p-2 rounded-xl hover:bg-slate-800/50 cursor-pointer transition-colors group"
              >
                <div className="flex items-center gap-2.5">
                  <div className="w-7 h-7 rounded-lg bg-indigo-500/20 border border-indigo-400/30 flex items-center justify-center text-cyan-300">
                    <Rocket className="w-3.5 h-3.5" />
                  </div>
                  <div>
                    <p className="text-xs font-bold text-white group-hover:text-cyan-300 font-mono">
                      {app.name}
                    </p>
                    <p className="text-[10px] text-slate-400 font-mono">{app.domain}</p>
                  </div>
                </div>

                <div className="flex items-center gap-3 text-xs font-mono">
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                    ● {app.status}
                  </span>
                  <span className="text-slate-300">{app.traffic30d}</span>
                  <span className="text-slate-500 hidden sm:inline">{app.lastDeploy}</span>
                  <MoreVertical className="w-3.5 h-3.5 text-slate-600 group-hover:text-slate-300" />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* System Health (3 cols) */}
        <div className="md:col-span-3 p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between pb-3 border-b border-blue-500/15 mb-3">
            <h3 className="text-sm font-bold text-white tracking-tight">System Health</h3>
            <span className="text-[11px] font-medium text-cyan-400 hover:underline cursor-pointer">
              View All →
            </span>
          </div>

          <div className="space-y-3 text-xs font-mono">
            <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
              <span className="text-slate-300">API Server</span>
              <span className="text-emerald-400 flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> Healthy
              </span>
            </div>

            <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
              <span className="text-slate-300">Database</span>
              <span className="text-emerald-400 flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> Healthy
              </span>
            </div>

            <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
              <span className="text-slate-300">Node Agents</span>
              <span className="text-cyan-400">
                8 / 10 Online
              </span>
            </div>

            <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
              <span className="text-slate-300">Storage Mesh</span>
              <span className="text-emerald-400 flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> Healthy
              </span>
            </div>

            <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
              <span className="text-slate-300">Security</span>
              <span className="text-emerald-400">0 Active Threats</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
