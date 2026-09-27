import React, { useState } from 'react';
import {
  Server,
  Plus,
  Radio,
  Boxes,
  Cpu,
  HardDrive,
  Activity,
  Layers,
  Shield,
  Laptop,
  Home,
  Cloud,
  ChevronRight,
  Sliders,
  Terminal,
  Search,
  Filter,
  ArrowUpDown,
  MoreVertical,
  CheckCircle2,
  AlertTriangle,
  RotateCw,
  ExternalLink
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';
import { GlobeMap } from '../components/GlobeMap';

export const NodesCompute: React.FC = () => {
  const {
    nodes,
    setIsAddNodeModalOpen,
    setSelectedNodeId,
    cordonNode,
    drainNode,
    killSwitchNode,
    setCurrentTab
  } = useNetwork();

  const [activeTab, setActiveTab] = useState<'overview' | 'my-nodes' | 'workloads' | 'contributions' | 'depin' | 'activity'>('overview');
  const [nodeFilter, setNodeFilter] = useState<'all' | 'online' | 'offline' | 'self' | 'contributing' | 'depin'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [mapLayer, setMapLayer] = useState<'all' | 'my-nodes' | 'community' | 'depin'>('all');

  // Filter nodes
  const filteredNodes = nodes.filter((node) => {
    const matchesSearch =
      node.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      node.location.country.toLowerCase().includes(searchQuery.toLowerCase()) ||
      node.shortId.toLowerCase().includes(searchQuery.toLowerCase());

    if (!matchesSearch) return false;

    if (nodeFilter === 'online') return node.status === 'Online';
    if (nodeFilter === 'offline') return node.status === 'Offline';
    if (nodeFilter === 'self') return node.isSelfHosted;
    if (nodeFilter === 'contributing') return node.isContributing;
    if (nodeFilter === 'depin') return node.isDePIN;

    return true;
  });

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      {/* Hero Header matching screenshot */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 p-6 shadow-2xl backdrop-blur-xl">
        <div className="flex items-center gap-2 text-cyan-400 text-xs font-semibold uppercase tracking-wider mb-1">
          <Server className="w-3.5 h-3.5" />
          <span>Nodes & Compute</span>
        </div>

        <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
          Your hardware. <span className="text-purple-400">Your network.</span>{' '}
          <span className="text-cyan-400">Your workloads.</span>
        </h1>
        <p className="text-xs sm:text-sm text-slate-300 max-w-2xl mt-1.5 leading-relaxed">
          Connect computers you control and use them to host applications, contribute spare resources, or participate in supported decentralized compute networks.
        </p>

        {/* Buttons */}
        <div className="flex flex-wrap items-center gap-3 mt-4">
          <button
            onClick={() => setIsAddNodeModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-blue-600/30 transition-all flex items-center gap-1.5"
          >
            <Plus className="w-4 h-4" />
            <span>Add Your Machine</span>
          </button>

          <button
            onClick={() => setIsAddNodeModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-200 bg-slate-900/80 hover:bg-slate-800 border border-blue-500/30 shadow-md transition-all flex items-center gap-1.5"
          >
            <Radio className="w-4 h-4 text-cyan-400" />
            <span>Connect Remote Server</span>
          </button>

          <button
            onClick={() => setCurrentTab('depin')}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-purple-200 bg-purple-950/60 hover:bg-purple-900/60 border border-purple-500/30 shadow-md transition-all flex items-center gap-1.5"
          >
            <Boxes className="w-4 h-4 text-purple-400" />
            <span>Explore DePIN Networks</span>
          </button>
        </div>

        {/* Navigation Tabs matching screenshot */}
        <div className="flex items-center gap-2 mt-5 pt-3 border-t border-blue-500/20 text-xs overflow-x-auto custom-scrollbar">
          {[
            { id: 'overview', label: 'Overview' },
            { id: 'my-nodes', label: 'My Nodes' },
            { id: 'workloads', label: 'Workloads' },
            { id: 'contributions', label: 'Contributions' },
            { id: 'depin', label: 'DePIN Networks' },
            { id: 'activity', label: 'Activity' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`px-3 py-1.5 rounded-xl font-medium transition-all ${
                activeTab === tab.id
                  ? 'bg-blue-600/30 text-cyan-300 border border-cyan-400/40'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* 5 Stat Cards matching screenshot */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3.5 font-mono">
        {/* My Nodes */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400 font-sans">My Nodes</span>
            <div className="p-1.5 rounded-lg bg-cyan-500/15 text-cyan-400">
              <Server className="w-4 h-4" />
            </div>
          </div>
          <span className="text-2xl font-extrabold text-white">4</span>
          <p className="text-[11px] text-slate-400 mt-1">
            <span className="text-emerald-400 font-bold">3 online</span> • <span className="text-rose-400">1 offline</span>
          </p>
        </div>

        {/* Total CPU */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400 font-sans">Total CPU</span>
            <div className="p-1.5 rounded-lg bg-blue-500/15 text-blue-400">
              <Cpu className="w-4 h-4" />
            </div>
          </div>
          <span className="text-2xl font-extrabold text-white">32 Cores</span>
          <p className="text-[11px] text-slate-400 mt-1">
            <span>14 allocated</span> • <span className="text-cyan-400 font-bold">18 available</span>
          </p>
        </div>

        {/* Total Memory */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400 font-sans">Total Memory</span>
            <div className="p-1.5 rounded-lg bg-purple-500/15 text-purple-400">
              <Layers className="w-4 h-4" />
            </div>
          </div>
          <span className="text-2xl font-extrabold text-white">128 GB</span>
          <p className="text-[11px] text-slate-400 mt-1">
            <span>48 allocated</span> • <span className="text-purple-400 font-bold">80 available</span>
          </p>
        </div>

        {/* Total Storage */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400 font-sans">Total Storage</span>
            <div className="p-1.5 rounded-lg bg-emerald-500/15 text-emerald-400">
              <HardDrive className="w-4 h-4" />
            </div>
          </div>
          <span className="text-2xl font-extrabold text-white">2.4 TB</span>
          <p className="text-[11px] text-slate-400 mt-1">
            <span>612 GB alloc</span> • <span className="text-emerald-400 font-bold">1.8 TB avail</span>
          </p>
        </div>

        {/* Total GPU */}
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl col-span-2 sm:col-span-1">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-slate-400 font-sans">Total GPU</span>
            <div className="p-1.5 rounded-lg bg-amber-500/15 text-amber-400">
              <Boxes className="w-4 h-4" />
            </div>
          </div>
          <span className="text-2xl font-extrabold text-white">2</span>
          <p className="text-[11px] text-slate-400 mt-1">
            <span>1 allocated</span> • <span className="text-amber-400 font-bold">1 available</span>
          </p>
        </div>
      </div>

      {/* Middle Row: Global Map + Right Panel (Add a Node + DePIN + Self-Hosting Benefits) matching screenshot */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        {/* Left: Global Infrastructure Map (8 cols) */}
        <div className="lg:col-span-8 space-y-5">
          <GlobeMap
            title="Global Infrastructure Map"
            subtitle="Live view of your nodes, community capacity and connected DePIN networks."
            activeFilter={mapLayer}
            onFilterChange={(f) => setMapLayer(f as any)}
            filterOptions={[
              { id: 'all', label: 'All Layers', count: 8 },
              { id: 'my-nodes', label: 'My Nodes', count: 4 },
              { id: 'community', label: 'Community', count: 3 },
              { id: 'depin', label: 'DePIN', count: 1 },
            ]}
          />

          {/* Network Contribution Card matching screenshot */}
          <div className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between mb-4 pb-2 border-b border-blue-500/15">
              <h3 className="text-sm font-bold text-white tracking-tight flex items-center gap-2">
                <Activity className="w-4 h-4 text-cyan-400" />
                <span>Network Contribution</span>
              </h3>
              <span className="text-xs font-mono text-slate-400 bg-slate-900 px-2 py-0.5 rounded-lg border border-slate-800">
                Last 30 days
              </span>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 font-mono">
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">CPU Contribution</span>
                <span className="text-lg font-bold text-cyan-300">3,240 CPU-h</span>
                <span className="text-xs text-cyan-400 block mt-0.5">↑ 18%</span>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">GPU Contribution</span>
                <span className="text-lg font-bold text-purple-300">412 GPU-h</span>
                <span className="text-xs text-purple-400 block mt-0.5">↑ 26%</span>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">Storage Contribution</span>
                <span className="text-lg font-bold text-emerald-300">1.8 TB-h</span>
                <span className="text-xs text-emerald-400 block mt-0.5">↑ 12%</span>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">Bandwidth</span>
                <span className="text-lg font-bold text-amber-300">950 GB</span>
                <span className="text-xs text-amber-400 block mt-0.5">↑ 32%</span>
              </div>
            </div>
          </div>
        </div>

        {/* Right Panel: Add a Node, DePIN Networks, Self-Hosting Benefits (4 cols) matching screenshot */}
        <div className="lg:col-span-4 space-y-4">
          {/* Add a Node quick list */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <h3 className="text-sm font-bold text-white tracking-tight mb-2 flex items-center justify-between">
              <span>Add a Node</span>
              <span className="text-[10px] text-cyan-400 font-normal">Turn any machine into a node</span>
            </h3>

            <div className="space-y-1.5 text-xs">
              {[
                { label: 'This Computer', desc: 'Install on your current machine', icon: Laptop },
                { label: 'Linux Server', desc: 'Ubuntu, Debian, CentOS, etc.', icon: Server },
                { label: 'Home Server', desc: 'Your on-premise hardware', icon: Home },
                { label: 'VPS', desc: 'DigitalOcean, Hetzner, Linode', icon: Cloud },
                { label: 'Raspberry Pi / ARM', desc: 'ARM devices and SBCs', icon: Cpu },
                { label: 'GPU Machine', desc: 'NVIDIA, AMD, Apple Silicon', icon: Boxes },
              ].map((item) => {
                const Icon = item.icon;
                return (
                  <button
                    key={item.label}
                    onClick={() => setIsAddNodeModalOpen(true)}
                    className="w-full flex items-center justify-between p-2 rounded-xl bg-slate-900/50 hover:bg-slate-800/80 border border-slate-800 text-left transition-colors group"
                  >
                    <div className="flex items-center gap-2.5">
                      <Icon className="w-4 h-4 text-cyan-400" />
                      <div>
                        <p className="font-semibold text-slate-200">{item.label}</p>
                        <p className="text-[10px] text-slate-500">{item.desc}</p>
                      </div>
                    </div>
                    <ChevronRight className="w-3.5 h-3.5 text-slate-600 group-hover:text-cyan-400 transition-colors" />
                  </button>
                );
              })}
            </div>
          </div>

          {/* DePIN Networks connectors matching screenshot */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between pb-2 border-b border-slate-800 mb-2.5">
              <h3 className="text-xs font-bold text-white uppercase tracking-wider">DePIN Networks</h3>
              <button
                onClick={() => setCurrentTab('depin')}
                className="text-[10px] text-cyan-400 hover:underline"
              >
                View All →
              </button>
            </div>

            <div className="space-y-2 text-xs font-mono">
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-white font-sans font-semibold">Decentralized.Host</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                  ● Active (4 nodes)
                </span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Akash Network</span>
                <span className="text-[10px] text-slate-500">Not Installed</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Golem Network</span>
                <span className="text-[10px] text-amber-400">Ready to Install</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Flux Network</span>
                <span className="text-[10px] text-slate-500">Not Installed</span>
              </div>
            </div>
          </div>

          {/* Self-Hosting Benefits Donut matching screenshot */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl text-center">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider mb-1">
              Self-Hosting Benefits
            </h3>
            <p className="text-[10px] text-slate-400 mb-3">Based on your current configuration</p>

            <div className="grid grid-cols-4 gap-1.5 font-mono mb-3">
              <div className="p-2 rounded-xl bg-slate-900 border border-cyan-500/30">
                <span className="text-sm font-extrabold text-cyan-300">82%</span>
                <span className="text-[9px] text-slate-400 block font-sans">Your Hardware</span>
              </div>
              <div className="p-2 rounded-xl bg-slate-900 border border-purple-500/30">
                <span className="text-sm font-extrabold text-purple-300">18%</span>
                <span className="text-[9px] text-slate-400 block font-sans">External Dep</span>
              </div>
              <div className="p-2 rounded-xl bg-slate-900 border border-emerald-500/30">
                <span className="text-sm font-extrabold text-emerald-300">100%</span>
                <span className="text-[9px] text-slate-400 block font-sans">Private Storage</span>
              </div>
              <div className="p-2 rounded-xl bg-slate-900 border border-amber-500/30">
                <span className="text-sm font-extrabold text-amber-300">4</span>
                <span className="text-[9px] text-slate-400 block font-sans">Indep. Nodes</span>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-1.5 text-[10px] font-medium text-slate-300">
              <div className="p-1.5 rounded-lg bg-slate-900/60 border border-slate-800 flex items-center justify-center gap-1">
                <span>🛡 Lower Costs</span>
              </div>
              <div className="p-1.5 rounded-lg bg-slate-900/60 border border-slate-800 flex items-center justify-center gap-1">
                <span>🔐 Full Data Control</span>
              </div>
              <div className="p-1.5 rounded-lg bg-slate-900/60 border border-slate-800 flex items-center justify-center gap-1">
                <span>🌐 Global Availability</span>
              </div>
              <div className="p-1.5 rounded-lg bg-slate-900/60 border border-slate-800 flex items-center justify-center gap-1">
                <span>⚡ Real Decentralization</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Node Table matching screenshot */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
        {/* Table Filters & Search */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-4 border-b border-blue-500/15">
          {/* Filter Pills */}
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            {[
              { id: 'all', label: `All Nodes (${nodes.length})` },
              { id: 'online', label: `Online (${nodes.filter((n) => n.status === 'Online').length})` },
              { id: 'offline', label: `Offline (${nodes.filter((n) => n.status === 'Offline').length})` },
              { id: 'self', label: `Self-Hosted (${nodes.filter((n) => n.isSelfHosted).length})` },
              { id: 'contributing', label: `Contributing (${nodes.filter((n) => n.isContributing).length})` },
              { id: 'depin', label: `DePIN (${nodes.filter((n) => n.isDePIN).length})` },
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => setNodeFilter(f.id as any)}
                className={`px-3 py-1.5 rounded-xl font-medium transition-all ${
                  nodeFilter === f.id
                    ? 'bg-blue-600/30 text-cyan-300 border border-cyan-400/40 shadow-md shadow-blue-500/20'
                    : 'text-slate-400 hover:text-white bg-slate-900/60 border border-slate-800'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-2.5 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search nodes by name, ID..."
              className="w-full pl-9 pr-3 py-2 rounded-xl bg-slate-900/80 border border-slate-800 text-xs text-white placeholder-slate-500 font-mono focus:outline-none focus:border-cyan-400"
            />
          </div>
        </div>

        {/* Table Content */}
        <div className="overflow-x-auto custom-scrollbar mt-3">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                <th className="py-3 px-3">Node Name</th>
                <th className="py-3 px-2">Status</th>
                <th className="py-3 px-2">Roles</th>
                <th className="py-3 px-2">Resources (CPU / RAM / Storage)</th>
                <th className="py-3 px-2">Location</th>
                <th className="py-3 px-2">Uptime</th>
                <th className="py-3 px-2">Workloads</th>
                <th className="py-3 px-2">Contribution</th>
                <th className="py-3 px-3 text-right font-sans">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {filteredNodes.map((n) => (
                <tr
                  key={n.id}
                  className="hover:bg-slate-800/40 transition-colors group cursor-pointer"
                  onClick={() => setSelectedNodeId(n.id)}
                >
                  {/* Name */}
                  <td className="py-3.5 px-3">
                    <div className="flex items-center gap-2">
                      <div className="p-1.5 rounded-lg bg-blue-500/10 text-cyan-400">
                        <Server className="w-4 h-4" />
                      </div>
                      <div>
                        <span className="font-bold text-white group-hover:text-cyan-300">
                          {n.name}
                        </span>
                        <p className="text-[10px] text-slate-500 font-mono">{n.shortId}</p>
                      </div>
                    </div>
                  </td>

                  {/* Status */}
                  <td className="py-3.5 px-2">
                    <div className="flex flex-col gap-1 items-start">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] border ${
                          n.status === 'Online'
                            ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                            : n.status === 'Degraded'
                            ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                            : 'bg-rose-500/15 text-rose-400 border-rose-500/30'
                        }`}
                      >
                        ● {n.status}
                      </span>
                      {n.cordoned && (
                        <span className="px-1.5 py-0.5 rounded text-[9px] bg-amber-500/15 text-amber-400 border border-amber-500/25">
                          Cordoned
                        </span>
                      )}
                    </div>
                  </td>

                  {/* Roles */}
                  <td className="py-3.5 px-2">
                    <div className="flex flex-wrap gap-1">
                      {n.roles.map((r) => (
                        <span
                          key={r}
                          className="px-2 py-0.5 rounded-md bg-slate-800 text-[10px] text-slate-300 border border-slate-700/60"
                        >
                          {r}
                        </span>
                      ))}
                    </div>
                  </td>

                  {/* Resources */}
                  <td className="py-3.5 px-2">
                    <div className="text-[11px] space-y-0.5">
                      <p className="text-slate-300">
                        {n.cpu.allocated} / {n.cpu.total} CPU
                      </p>
                      <p className="text-purple-300">
                        {n.memoryGb.allocated} / {n.memoryGb.total} GB
                      </p>
                      <p className="text-emerald-300">
                        {n.storageGb.allocated} / {n.storageGb.total} GB
                      </p>
                      {n.gpu && (
                        <p className="text-amber-300 text-[10px]">
                          1x {n.gpu.model.split(' ')[1]} ({n.gpu.allocated}/{n.gpu.count})
                        </p>
                      )}
                    </div>
                  </td>

                  {/* Location */}
                  <td className="py-3.5 px-2 text-slate-300">
                    <span className="mr-1">{n.location.flag}</span>
                    <span>{n.location.country}</span>
                    <span className="text-slate-500 block text-[10px]">({n.location.region})</span>
                  </td>

                  {/* Uptime */}
                  <td className="py-3.5 px-2 text-slate-400 text-[11px]">
                    {n.uptime}
                  </td>

                  {/* Workloads */}
                  <td className="py-3.5 px-2">
                    <span className="font-bold text-white">{n.workloads.total}</span>
                    <span className="text-[10px] text-slate-500 block">
                      ({n.workloads.owner} owner / {n.workloads.community} comm)
                    </span>
                  </td>

                  {/* Contribution */}
                  <td className="py-3.5 px-2 text-[11px] text-slate-300">
                    <p>{n.contribution.cpuh > 0 ? `${n.contribution.cpuh} CPU·h` : '-'}</p>
                    <p className="text-slate-500 text-[10px]">{n.contribution.bandwidthGb > 0 ? `${n.contribution.bandwidthGb} GB` : ''}</p>
                  </td>

                  {/* Actions */}
                  <td className="py-3.5 px-3 text-right" onClick={(e) => e.stopPropagation()}>
                    <div className="flex items-center justify-end gap-1.5">
                      <button
                        onClick={() => setSelectedNodeId(n.id)}
                        className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
                        title="Node Details & Terminal"
                      >
                        <Terminal className="w-3.5 h-3.5" />
                      </button>

                      <button
                        onClick={() => cordonNode(n.id)}
                        className={`p-1.5 rounded-lg transition-colors ${
                          n.cordoned
                            ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                            : 'bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-amber-400'
                        }`}
                        title={n.cordoned ? 'Uncordon node' : 'Cordon node'}
                      >
                        <Sliders className="w-3.5 h-3.5" />
                      </button>

                      <button
                        onClick={() => killSwitchNode(n.id)}
                        className="p-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/25 text-rose-400 border border-rose-500/20 transition-colors"
                        title="Sovereign Kill Switch"
                      >
                        <Shield className="w-3.5 h-3.5" />
                      </button>
                    </div>
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
