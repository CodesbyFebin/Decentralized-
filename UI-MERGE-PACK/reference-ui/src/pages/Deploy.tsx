import React, { useState } from 'react';
import {
  Rocket,
  Plus,
  GitBranch,
  Box,
  Layers,
  Shield,
  Activity,
  CheckCircle2,
  Clock,
  Globe2,
  ExternalLink,
  ChevronRight,
  Terminal,
  FileCode,
  Sparkles,
  ArrowRight,
  RotateCw,
  Search,
  Filter
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';
import { GlobeMap } from '../components/GlobeMap';

export const Deploy: React.FC = () => {
  const {
    deployments,
    setIsNewDeploymentModalOpen,
    setSelectedDeploymentId,
    setCurrentTab
  } = useNetwork();

  const [activeTab, setActiveTab] = useState<'all' | 'production' | 'preview' | 'stopped'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const filteredDeployments = deployments.filter((d) => {
    const matchesSearch =
      d.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      d.domain.toLowerCase().includes(searchQuery.toLowerCase());

    if (!matchesSearch) return false;

    if (activeTab === 'production') return d.environment === 'Production';
    if (activeTab === 'preview') return d.environment === 'Preview';
    if (activeTab === 'stopped') return d.status === 'Stopped';

    return true;
  });

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      {/* Hero Header matching screenshot */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 p-6 shadow-2xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-semibold uppercase text-cyan-400 font-mono flex items-center gap-1.5">
            <Rocket className="w-3.5 h-3.5" />
            Universal Deploy
          </span>
          <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
            ● LIVE
          </span>
        </div>

        <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
          Deploy anywhere. <span className="text-cyan-400">Without giving up control.</span>
        </h1>
        <p className="text-xs sm:text-sm text-slate-300 max-w-2xl mt-1.5 leading-relaxed">
          Deploy from Git, containers or artifacts to your hardware, edge nodes, community infrastructure or supported decentralized compute networks.
        </p>

        {/* Buttons */}
        <div className="flex flex-wrap items-center gap-3 mt-4">
          <button
            onClick={() => setIsNewDeploymentModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-blue-600/30 transition-all flex items-center gap-1.5"
          >
            <Plus className="w-4 h-4" />
            <span>New Deployment</span>
          </button>

          <button
            onClick={() => setIsNewDeploymentModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-200 bg-slate-900/80 hover:bg-slate-800 border border-blue-500/30 shadow-md transition-all flex items-center gap-1.5"
          >
            <GitBranch className="w-4 h-4 text-cyan-400" />
            <span>Import Repository</span>
          </button>
        </div>

        {/* Universal Build & Supply Chain Pipeline Preview matching screenshot */}
        <div className="mt-6 pt-4 border-t border-blue-500/20">
          <div className="flex items-center justify-between text-xs font-mono text-slate-400 mb-2">
            <span>Sovereign Supply Chain Pipeline (Rule #8)</span>
            <span className="text-cyan-400">Hermetic & Deterministic</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-3 text-xs font-mono">
            {/* Source */}
            <div className="p-3 rounded-2xl bg-slate-900/80 border border-blue-500/20 flex items-center gap-3">
              <div className="p-2 rounded-xl bg-blue-500/15 text-cyan-400">
                <GitBranch className="w-4 h-4" />
              </div>
              <div>
                <p className="font-bold text-white">Source</p>
                <p className="text-[10px] text-slate-400 font-sans">Git, Docker, or ZIP</p>
              </div>
            </div>

            {/* Build */}
            <div className="p-3 rounded-2xl bg-slate-900/80 border border-blue-500/20 flex items-center gap-3">
              <div className="p-2 rounded-xl bg-purple-500/15 text-purple-400">
                <Box className="w-4 h-4" />
              </div>
              <div>
                <p className="font-bold text-white">Build & SBOM</p>
                <p className="text-[10px] text-slate-400 font-sans">Install, Test, Sign</p>
              </div>
            </div>

            {/* Artifact */}
            <div className="p-3 rounded-2xl bg-slate-900/80 border border-blue-500/20 flex items-center gap-3">
              <div className="p-2 rounded-xl bg-emerald-500/15 text-emerald-400">
                <Shield className="w-4 h-4" />
              </div>
              <div>
                <p className="font-bold text-white">Artifact</p>
                <p className="text-[10px] text-emerald-400 font-sans">b3:6f4c... (Immutable)</p>
              </div>
            </div>

            {/* Deploy */}
            <div className="p-3 rounded-2xl bg-slate-900/80 border border-blue-500/20 flex items-center gap-3">
              <div className="p-2 rounded-xl bg-cyan-500/15 text-cyan-400">
                <Rocket className="w-4 h-4" />
              </div>
              <div>
                <p className="font-bold text-white">Placement</p>
                <p className="text-[10px] text-cyan-300 font-sans">Mesh Replicas Active</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 5 Deployment Modes cards matching screenshot */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3.5">
        {[
          {
            title: 'Self Host',
            desc: 'Deploy to hardware you control',
            points: ['Private mesh', 'Full control', 'Your storage', 'Your networking'],
            badge: 'Sovereign',
            color: 'cyan',
          },
          {
            title: 'Edge Deploy',
            desc: 'Distribute workloads across edge nodes',
            points: ['Global placement', 'Replicas', 'Health routing', 'Edge cache'],
            badge: 'Ultra Low Latency',
            color: 'purple',
          },
          {
            title: 'Community Deploy',
            desc: 'Run on independent operators',
            points: ['Verified operators', 'Resource marketplace', 'Workload isolation', 'Fair usage'],
            badge: 'Distributed',
            color: 'blue',
          },
          {
            title: 'DePIN Deploy',
            desc: 'Deploy through decentralized compute',
            points: ['Provider discovery', 'GPU/CPU selection', 'Native settlement', 'Provenance'],
            badge: 'Permissionless',
            color: 'emerald',
          },
          {
            title: 'Hybrid Deploy',
            desc: 'Combine multiple infrastructure sources',
            points: ['Primary + failover', 'Edge + community', 'Your storage', 'Custom topology'],
            badge: 'Resilient',
            color: 'amber',
          },
        ].map((card) => (
          <div
            key={card.title}
            onClick={() => setIsNewDeploymentModalOpen(true)}
            className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 hover:border-cyan-400/40 shadow-xl backdrop-blur-xl transition-all cursor-pointer group flex flex-col justify-between"
          >
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-blue-500/15 text-cyan-300 border border-blue-500/25">
                  {card.badge}
                </span>
                <ChevronRight className="w-3.5 h-3.5 text-slate-500 group-hover:text-cyan-400 transition-colors" />
              </div>
              <h3 className="text-sm font-bold text-white group-hover:text-cyan-300 transition-colors">
                {card.title}
              </h3>
              <p className="text-[11px] text-slate-400 mt-0.5 mb-3">{card.desc}</p>
            </div>

            <ul className="space-y-1 text-[11px] text-slate-300 font-mono">
              {card.points.map((pt) => (
                <li key={pt} className="flex items-center gap-1.5">
                  <span className="text-cyan-400">✓</span>
                  <span>{pt}</span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      {/* 5 Stats Cards matching screenshot */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-3.5 font-mono">
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Active Deployments</span>
          <span className="text-2xl font-extrabold text-white">12</span>
          <span className="text-[11px] text-cyan-400 block mt-1">↑ 3 this week</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Total Replicas</span>
          <span className="text-2xl font-extrabold text-purple-300">28</span>
          <span className="text-[11px] text-slate-400 block mt-1 font-sans">Across 4 networks</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Successful Deployments</span>
          <span className="text-2xl font-extrabold text-emerald-400">98.4%</span>
          <span className="text-[11px] text-emerald-400 block mt-1">↑ 2.1%</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Avg. Deployment Time</span>
          <span className="text-2xl font-extrabold text-white">2m 14s</span>
          <span className="text-[11px] text-cyan-400 block mt-1">↓ 28%</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl col-span-2 sm:col-span-1">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Global Regions</span>
          <span className="text-2xl font-extrabold text-amber-300">6</span>
          <span className="text-[11px] text-amber-400 block mt-1 font-sans">Multi-ASN Provenance</span>
        </div>
      </div>

      {/* Middle Row: Live Deployment Map + Ownership Gauge & Side Panel matching screenshot */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        <div className="lg:col-span-8 space-y-5">
          <GlobeMap
            title="Live Deployment Map"
            subtitle="Real-time view of your deployments across owned, community and decentralized networks."
            filterOptions={[
              { id: 'all', label: 'All Deployments', count: 12 },
              { id: 'my', label: 'My Nodes', count: 8 },
              { id: 'comm', label: 'Community', count: 3 },
              { id: 'depin', label: 'DePIN', count: 1 },
            ]}
          />

          {/* Infrastructure Ownership Card matching screenshot */}
          <div className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between pb-3 border-b border-blue-500/15 mb-4">
              <h3 className="text-sm font-bold text-white tracking-tight flex items-center gap-2">
                <Shield className="w-4 h-4 text-cyan-400" />
                <span>Infrastructure Ownership</span>
              </h3>
              <span className="text-xs font-mono text-slate-400">All Deployments</span>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 items-center">
              {/* Ownership Breakdown */}
              <div className="flex items-center gap-4">
                <div className="w-20 h-20 rounded-full border-4 border-cyan-400 flex flex-col items-center justify-center font-mono">
                  <span className="text-lg font-bold text-white">67%</span>
                  <span className="text-[8px] text-cyan-400 font-sans">Your Hardware</span>
                </div>
                <div className="space-y-1.5 text-xs font-mono">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-cyan-400" />
                    <span className="text-slate-300">Your Hardware: 67%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-purple-400" />
                    <span className="text-slate-300">Community Nodes: 25%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-400" />
                    <span className="text-slate-300">DePIN Network: 8%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-slate-600" />
                    <span className="text-slate-500">Central Cloud: 0%</span>
                  </div>
                </div>
              </div>

              {/* Traffic metric */}
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-slate-800 font-mono text-xs">
                <span className="text-slate-400 font-sans block mb-1">Deployment Traffic (Last 24h)</span>
                <div className="flex justify-between items-baseline">
                  <div>
                    <span className="text-xl font-bold text-white">1.2M</span>
                    <span className="text-slate-400 text-[11px] ml-1">Requests (↑18%)</span>
                  </div>
                  <div>
                    <span className="text-xl font-bold text-cyan-300">264 GB</span>
                    <span className="text-slate-400 text-[11px] ml-1">Bandwidth (↑32%)</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Right Panel: Deployment Templates, Execution Networks matching screenshot */}
        <div className="lg:col-span-4 space-y-4">
          {/* Deployment Templates */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between pb-2 border-b border-slate-800 mb-2.5">
              <h3 className="text-xs font-bold text-white uppercase tracking-wider">Deployment Templates</h3>
              <button onClick={() => setIsNewDeploymentModalOpen(true)} className="text-[10px] text-cyan-400 hover:underline">
                View All →
              </button>
            </div>

            <div className="space-y-1.5 text-xs">
              {[
                { name: 'Next.js', desc: 'Full-stack web application with SSR', tag: 'Web' },
                { name: 'Astro', desc: 'Static site with edge support', tag: 'Static' },
                { name: 'FastAPI', desc: 'Python API microservice with Uvicorn', tag: 'API' },
                { name: 'PostgreSQL', desc: 'Database with persistent ZFS storage', tag: 'DB' },
                { name: 'WordPress', desc: 'Blog or CMS with MariaDB', tag: 'CMS' },
                { name: 'AI Inference', desc: 'GPU workload template (vLLM / Ollama)', tag: 'AI' },
              ].map((tmpl) => (
                <button
                  key={tmpl.name}
                  onClick={() => setIsNewDeploymentModalOpen(true)}
                  className="w-full flex items-center justify-between p-2 rounded-xl bg-slate-900/60 hover:bg-slate-800/80 border border-slate-800 text-left transition-colors group"
                >
                  <div>
                    <span className="font-bold text-white group-hover:text-cyan-300 font-mono">
                      {tmpl.name}
                    </span>
                    <p className="text-[10px] text-slate-500">{tmpl.desc}</p>
                  </div>
                  <ChevronRight className="w-3.5 h-3.5 text-slate-600 group-hover:text-cyan-400 transition-colors" />
                </button>
              ))}
            </div>
          </div>

          {/* Execution Networks matching screenshot */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between pb-2 border-b border-slate-800 mb-2.5">
              <h3 className="text-xs font-bold text-white uppercase tracking-wider">Execution Networks</h3>
              <button onClick={() => setCurrentTab('depin')} className="text-[10px] text-cyan-400 hover:underline">
                View All →
              </button>
            </div>

            <div className="space-y-2 text-xs font-mono">
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-white font-sans font-semibold">Decentralized.Host</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                  ● Active (8 nodes)
                </span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Akash Network</span>
                <span className="text-[10px] text-amber-400">Ready to Deploy</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Golem Network</span>
                <span className="text-[10px] text-slate-500">Not Installed</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-white font-sans font-semibold">Flux Network</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                  ● Active
                </span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">External Cloud</span>
                <span className="text-[10px] text-slate-500">Not Configured</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Deployments Table matching screenshot */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-4 border-b border-blue-500/15">
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            {[
              { id: 'all', label: `All Deployments (${deployments.length})` },
              { id: 'production', label: `Production (${deployments.filter((d) => d.environment === 'Production').length})` },
              { id: 'preview', label: `Preview (${deployments.filter((d) => d.environment === 'Preview').length})` },
              { id: 'stopped', label: `Stopped (${deployments.filter((d) => d.status === 'Stopped').length})` },
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => setActiveTab(f.id as any)}
                className={`px-3 py-1.5 rounded-xl font-medium transition-all ${
                  activeTab === f.id
                    ? 'bg-blue-600/30 text-cyan-300 border border-cyan-400/40 shadow-md shadow-blue-500/20'
                    : 'text-slate-400 hover:text-white bg-slate-900/60 border border-slate-800'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-2.5 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search deployments..."
              className="w-full pl-9 pr-3 py-2 rounded-xl bg-slate-900/80 border border-slate-800 text-xs text-white placeholder-slate-500 font-mono focus:outline-none focus:border-cyan-400"
            />
          </div>
        </div>

        <div className="overflow-x-auto custom-scrollbar mt-3">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                <th className="py-3 px-3">Name</th>
                <th className="py-3 px-2">Status</th>
                <th className="py-3 px-2">Source</th>
                <th className="py-3 px-2">Environment</th>
                <th className="py-3 px-2">Replicas</th>
                <th className="py-3 px-2">Placement</th>
                <th className="py-3 px-2">Domain / Endpoint</th>
                <th className="py-3 px-2">Last Deployed</th>
                <th className="py-3 px-3 text-right font-sans">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {filteredDeployments.map((d) => (
                <tr
                  key={d.id}
                  className="hover:bg-slate-800/40 transition-colors group cursor-pointer"
                  onClick={() => setSelectedDeploymentId(d.id)}
                >
                  <td className="py-3.5 px-3">
                    <div className="flex items-center gap-2">
                      <div className="p-1.5 rounded-lg bg-blue-500/10 text-cyan-400">
                        <Rocket className="w-4 h-4" />
                      </div>
                      <div>
                        <span className="font-bold text-white group-hover:text-cyan-300">
                          {d.name}
                        </span>
                        <p className="text-[10px] text-slate-500 font-mono">{d.shortId}</p>
                      </div>
                    </div>
                  </td>

                  <td className="py-3.5 px-2">
                    <span
                      className={`px-2 py-0.5 rounded-full text-[10px] border ${
                        d.status === 'Running'
                          ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                          : d.status === 'Degraded'
                          ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                          : 'bg-blue-500/15 text-cyan-400 border-blue-500/30'
                      }`}
                    >
                      ● {d.status}
                    </span>
                  </td>

                  <td className="py-3.5 px-2 text-slate-300">
                    <span className="text-[11px] block">{d.source.type}</span>
                    <span className="text-[10px] text-slate-500 font-mono">
                      {d.source.commitSha}
                    </span>
                  </td>

                  <td className="py-3.5 px-2">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] ${
                        d.environment === 'Production'
                          ? 'bg-purple-500/15 text-purple-300'
                          : 'bg-blue-500/15 text-cyan-300'
                      }`}
                    >
                      {d.environment}
                    </span>
                  </td>

                  <td className="py-3.5 px-2 text-slate-200">
                    {d.replicas.observed} / {d.replicas.desired}
                  </td>

                  <td className="py-3.5 px-2">
                    <div className="flex items-center gap-1">
                      {d.placement.map((p) => (
                        <span
                          key={p}
                          className="px-1.5 py-0.5 rounded text-[10px] bg-slate-800 text-slate-400 uppercase"
                        >
                          {p}
                        </span>
                      ))}
                    </div>
                  </td>

                  <td className="py-3.5 px-2 text-cyan-400 hover:underline">
                    <a href={d.domain} target="_blank" rel="noreferrer" onClick={(e) => e.stopPropagation()}>
                      {d.domain}
                    </a>
                  </td>

                  <td className="py-3.5 px-2 text-slate-400 text-[11px]">
                    {d.lastDeploy}
                  </td>

                  <td className="py-3.5 px-3 text-right" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => setSelectedDeploymentId(d.id)}
                      className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
                      title="Inspect & Logs"
                    >
                      <Terminal className="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
