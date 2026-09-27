import React, { useState } from 'react';
import {
  HardDrive,
  Plus,
  Database,
  Layers,
  Shield,
  Activity,
  CheckCircle2,
  AlertTriangle,
  RotateCw,
  Search,
  ChevronRight,
  Server,
  Cloud,
  Lock,
  Globe2
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';
import { GlobeMap } from '../components/GlobeMap';

export const Storage: React.FC = () => {
  const {
    storageNodes,
    setIsAddStorageModalOpen,
    setCurrentTab
  } = useNetwork();

  const [activeTab, setActiveTab] = useState<'overview' | 'my-storage' | 'volumes' | 'objects' | 'replication' | 'contributions' | 'depin' | 'activity'>('overview');
  const [filterType, setFilterType] = useState<'all' | 'online' | 'offline' | 'self' | 'community' | 'depin'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const filteredStorage = storageNodes.filter((s) => {
    const matchesSearch =
      s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.type.toLowerCase().includes(searchQuery.toLowerCase());

    if (!matchesSearch) return false;

    if (filterType === 'online') return s.status === 'Online';
    if (filterType === 'offline') return s.status === 'Offline';
    if (filterType === 'self') return s.isSelfHosted;
    if (filterType === 'community') return s.isCommunity;
    if (filterType === 'depin') return s.isDePIN;

    return true;
  });

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      {/* Hero Header matching screenshot */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 p-6 shadow-2xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-semibold uppercase text-cyan-400 font-mono flex items-center gap-1.5">
            <HardDrive className="w-3.5 h-3.5" />
            Storage & Data Infrastructure
          </span>
          <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
            ● LIVE
          </span>
        </div>

        <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
          Your disks. <span className="text-cyan-400">Your data.</span>{' '}
          <span className="text-purple-400">Your storage network.</span>
        </h1>
        <p className="text-xs sm:text-sm text-slate-300 max-w-2xl mt-1.5 leading-relaxed">
          Turn storage you control into private, distributed infrastructure. Store your applications and data on your own hardware, replicate across trusted nodes, contribute spare capacity, or connect supported Web3 and DePIN storage networks.
        </p>

        {/* Buttons */}
        <div className="flex flex-wrap items-center gap-3 mt-4">
          <button
            onClick={() => setIsAddStorageModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-blue-600/30 transition-all flex items-center gap-1.5"
          >
            <Plus className="w-4 h-4" />
            <span>Add Storage</span>
          </button>

          <button
            onClick={() => setIsAddStorageModalOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-200 bg-slate-900/80 hover:bg-slate-800 border border-blue-500/30 shadow-md transition-all flex items-center gap-1.5"
          >
            <Database className="w-4 h-4 text-cyan-400" />
            <span>Create Volume</span>
          </button>

          <button
            onClick={() => setCurrentTab('depin')}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-purple-200 bg-purple-950/60 hover:bg-purple-900/60 border border-purple-500/30 shadow-md transition-all flex items-center gap-1.5"
          >
            <Layers className="w-4 h-4 text-purple-400" />
            <span>Explore Storage Networks</span>
          </button>
        </div>

        {/* Navigation Tabs matching screenshot */}
        <div className="flex items-center gap-2 mt-5 pt-3 border-t border-blue-500/20 text-xs overflow-x-auto custom-scrollbar">
          {[
            { id: 'overview', label: 'Overview' },
            { id: 'my-storage', label: 'My Storage' },
            { id: 'volumes', label: 'Volumes' },
            { id: 'objects', label: 'Objects & Artifacts' },
            { id: 'replication', label: 'Replication' },
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

      {/* 6 Stat Cards matching screenshot */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3.5 font-mono">
        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Total Capacity</span>
          <span className="text-xl font-extrabold text-white">68.4 TB</span>
          <span className="text-[11px] text-cyan-400 block mt-1">↑ +12%</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Used Storage</span>
          <span className="text-xl font-extrabold text-purple-300">28.1 TB</span>
          <span className="text-[11px] text-purple-400 block mt-1">41% used</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Available Storage</span>
          <span className="text-xl font-extrabold text-emerald-300">40.3 TB</span>
          <span className="text-[11px] text-emerald-400 block mt-1">59% available</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Storage Nodes</span>
          <span className="text-xl font-extrabold text-white">9</span>
          <span className="text-[11px] text-slate-400 block mt-1">
            <span className="text-emerald-400">7 online</span> • <span className="text-rose-400">2 offline</span>
          </span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Replicated Data</span>
          <span className="text-xl font-extrabold text-blue-300">24.8 TB</span>
          <span className="text-[11px] text-blue-400 block mt-1">89% protected</span>
        </div>

        <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
          <span className="text-xs font-medium text-slate-400 font-sans block mb-1">Integrity Status</span>
          <span className="text-xl font-extrabold text-emerald-400">100%</span>
          <span className="text-[10px] text-emerald-300 block mt-1 font-sans">All replicas healthy</span>
        </div>
      </div>

      {/* Middle Row: Distributed Storage Map + Storage Contribution & Side Panel matching screenshot */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
        <div className="lg:col-span-8 space-y-5">
          <GlobeMap
            title="Distributed Storage Map"
            subtitle="Global view of your storage nodes, replicas and data placement."
            showLayerButtons={true}
            filterOptions={[
              { id: 'all', label: 'My Storage', count: 4 },
              { id: 'comm', label: 'Community', count: 7 },
              { id: 'depin', label: 'DePIN Nodes', count: 3 },
            ]}
          />

          {/* Storage Contribution Card matching screenshot */}
          <div className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between mb-4 pb-2 border-b border-blue-500/15">
              <h3 className="text-sm font-bold text-white tracking-tight flex items-center gap-2">
                <Activity className="w-4 h-4 text-cyan-400" />
                <span>Storage Contribution</span>
              </h3>
              <span className="text-xs font-mono text-slate-400 bg-slate-900 px-2 py-0.5 rounded-lg border border-slate-800">
                Last 30 days
              </span>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 font-mono">
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">Storage Contributed</span>
                <span className="text-lg font-bold text-cyan-300">12.6 TB-h</span>
                <span className="text-xs text-cyan-400 block mt-0.5">↑ 28%</span>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">Bandwidth Served</span>
                <span className="text-lg font-bold text-purple-300">4.8 TB</span>
                <span className="text-xs text-purple-400 block mt-0.5">↑ 18%</span>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">Active Workloads</span>
                <span className="text-lg font-bold text-emerald-300">7 Active</span>
                <span className="text-xs text-emerald-400 block mt-0.5">↑ 40%</span>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-[11px] text-slate-400 font-sans block mb-1">Integrity Checks</span>
                <span className="text-lg font-bold text-amber-300">12,942</span>
                <span className="text-xs text-emerald-400 block mt-0.5 font-sans">100% passed</span>
              </div>
            </div>
          </div>
        </div>

        {/* Right Panel: Add Storage Sources, Supported Storage Networks, Self-Hosting Benefits matching screenshot */}
        <div className="lg:col-span-4 space-y-4">
          {/* Add Storage */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <h3 className="text-sm font-bold text-white tracking-tight mb-2 flex items-center justify-between">
              <span>Add Storage</span>
              <span className="text-[10px] text-cyan-400 font-normal">Connect disks or networks</span>
            </h3>

            <div className="space-y-1.5 text-xs">
              {[
                { label: 'Local Disk', desc: 'Add storage from this machine', icon: HardDrive },
                { label: 'NAS / Network Storage', desc: 'Synology, TrueNAS, QNAP, etc.', icon: Database },
                { label: 'Linux Server', desc: 'Ubuntu, Debian, CentOS, etc.', icon: Server },
                { label: 'S3 Compatible Storage', desc: 'MinIO, Ceph, Wasabi, Cloudflare R2', icon: Cloud },
                { label: 'Dedicated Storage Node', desc: 'Convert a server into storage node', icon: HardDrive },
                { label: 'External DePIN Storage', desc: 'Web3 storage networks', icon: Layers },
              ].map((item) => {
                const Icon = item.icon;
                return (
                  <button
                    key={item.label}
                    onClick={() => setIsAddStorageModalOpen(true)}
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

          {/* Supported Storage Networks matching screenshot */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl">
            <div className="flex items-center justify-between pb-2 border-b border-slate-800 mb-2.5">
              <h3 className="text-xs font-bold text-white uppercase tracking-wider">Supported Storage Networks</h3>
              <button onClick={() => setCurrentTab('depin')} className="text-[10px] text-cyan-400 hover:underline">
                View All →
              </button>
            </div>

            <div className="space-y-2 text-xs font-mono">
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Filecoin Network</span>
                <span className="text-[10px] text-amber-400">Ready to Install</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Arweave Network</span>
                <span className="text-[10px] text-slate-500">Not Installed</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-white font-sans font-semibold">Storj Network</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                  ● Active
                </span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Sia Network</span>
                <span className="text-[10px] text-blue-400">Eligibility Check</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-white font-sans font-semibold">IPFS (Pinning Service)</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                  ● Active
                </span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-xl bg-slate-900/60 border border-slate-800">
                <span className="text-slate-300 font-sans">Crust Network</span>
                <span className="text-[10px] text-slate-500">Not Installed</span>
              </div>
            </div>
          </div>

          {/* Self-Hosting Benefits matching screenshot */}
          <div className="p-4 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl text-center">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider mb-1">
              Self-Hosting Benefits
            </h3>
            <p className="text-[10px] text-slate-400 mb-3">Real infrastructure ownership outcomes</p>

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
                <span className="text-sm font-extrabold text-emerald-300">89%</span>
                <span className="text-[9px] text-slate-400 block font-sans">Replicated</span>
              </div>
              <div className="p-2 rounded-xl bg-slate-900 border border-amber-500/30">
                <span className="text-sm font-extrabold text-amber-300">6</span>
                <span className="text-[9px] text-slate-400 block font-sans">Storage Nodes</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Storage Table matching screenshot */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 pb-4 border-b border-blue-500/15">
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            {[
              { id: 'all', label: `All Storage (${storageNodes.length})` },
              { id: 'online', label: `Online (${storageNodes.filter((s) => s.status === 'Online').length})` },
              { id: 'offline', label: `Offline (${storageNodes.filter((s) => s.status === 'Offline').length})` },
              { id: 'self', label: `Self-Hosted (${storageNodes.filter((s) => s.isSelfHosted).length})` },
              { id: 'community', label: `Community (${storageNodes.filter((s) => s.isCommunity).length})` },
              { id: 'depin', label: `DePIN (${storageNodes.filter((s) => s.isDePIN).length})` },
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => setFilterType(f.id as any)}
                className={`px-3 py-1.5 rounded-xl font-medium transition-all ${
                  filterType === f.id
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
              placeholder="Search storage nodes, volumes..."
              className="w-full pl-9 pr-3 py-2 rounded-xl bg-slate-900/80 border border-slate-800 text-xs text-white placeholder-slate-500 font-mono focus:outline-none focus:border-cyan-400"
            />
          </div>
        </div>

        <div className="overflow-x-auto custom-scrollbar mt-3">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                <th className="py-3 px-3">Node / Storage</th>
                <th className="py-3 px-2">Type</th>
                <th className="py-3 px-2">Status</th>
                <th className="py-3 px-2">Capacity</th>
                <th className="py-3 px-2">Used</th>
                <th className="py-3 px-2">Available</th>
                <th className="py-3 px-2">Replicas</th>
                <th className="py-3 px-2">Integrity</th>
                <th className="py-3 px-2">Bandwidth</th>
                <th className="py-3 px-2">Policy</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {filteredStorage.map((s) => (
                <tr key={s.id} className="hover:bg-slate-800/40 transition-colors">
                  <td className="py-3 px-3">
                    <div className="flex items-center gap-2">
                      <div className="p-1.5 rounded-lg bg-blue-500/10 text-cyan-400">
                        <HardDrive className="w-4 h-4" />
                      </div>
                      <div>
                        <span className="font-bold text-white">{s.name}</span>
                        <p className="text-[10px] text-slate-500">{s.shortId}</p>
                      </div>
                    </div>
                  </td>
                  <td className="py-3 px-2 text-slate-300 font-sans">{s.type}</td>
                  <td className="py-3 px-2">
                    <span
                      className={`px-2 py-0.5 rounded-full text-[10px] border ${
                        s.status === 'Online'
                          ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                          : 'bg-rose-500/15 text-rose-400 border-rose-500/30'
                      }`}
                    >
                      ● {s.status}
                    </span>
                  </td>
                  <td className="py-3 px-2 text-white font-bold">{s.capacityTb} TB</td>
                  <td className="py-3 px-2 text-purple-300">{s.usedTb} TB</td>
                  <td className="py-3 px-2 text-emerald-300">{s.availableTb} TB</td>
                  <td className="py-3 px-2 text-slate-300">{s.replicas} Replicas</td>
                  <td className="py-3 px-2">
                    <span className="text-emerald-400 flex items-center gap-1">
                      <CheckCircle2 className="w-3.5 h-3.5" /> {s.integrity}
                    </span>
                  </td>
                  <td className="py-3 px-2 text-slate-300">{s.bandwidthServed}</td>
                  <td className="py-3 px-2 text-cyan-300 font-sans">{s.policy}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
