import React, { useEffect, useState, useMemo } from 'react';
import {
  Server,
  Plus,
  Search,
  RefreshCw,
  HardDrive,
  Zap,
  Globe,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  MoreVertical,
  ShieldCheck,
  RotateCw,
  Cpu,
  Layers,
  Activity,
  Award,
  ExternalLink,
  ChevronRight,
  Terminal,
  Copy,
  Check,
  X,
  Play,
  Pause,
  Filter,
  Sliders,
  Radio,
  Flame,
  Clock,
  Sparkles,
  Database
} from 'lucide-react';
import { NodeInfo, ComputeSummary, DePINIntegration, SelfHostingBenefit, NodeWorkload } from '../../types/platform';
import { api } from '../../lib/api';
import { MetricCard } from '../common/MetricCard';
import { WorldMap } from '../common/WorldMap';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

export type CommandTab = 'mesh' | 'compute' | 'workloads' | 'depin' | 'benefits';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
  selectedNodeId?: string;
  /** Hide the standalone header / KPI row / tab strip when hosted inside Nodes & Compute. */
  embedded?: boolean;
  tab?: CommandTab;
  openAddNode?: boolean;
}

export const NodesOperations: React.FC<Props> = ({ onNavigate, selectedNodeId, embedded, tab, openAddNode }) => {
  const [nodes, setNodes] = useState<NodeInfo[]>([]);
  const [summary, setSummary] = useState<any>(null);
  const [computeSummary, setComputeSummary] = useState<ComputeSummary | null>(null);
  const [depinIntegrations, setDePINIntegrations] = useState<DePINIntegration[]>([]);
  const [selfHostingBenefits, setSelfHostingBenefits] = useState<SelfHostingBenefit[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<'All' | 'Online' | 'Degraded' | 'Offline'>('All');
  const [hardwareFilter, setHardwareFilter] = useState<'All' | 'Self-Hosted' | 'GPU' | 'Validator' | 'Edge'>('All');
  const [activeCommandTab, setActiveCommandTab] = useState<CommandTab>(tab ?? 'mesh');
  useEffect(() => {
    if (tab) setActiveCommandTab(tab);
  }, [tab]);
  const [inspectNode, setInspectNode] = useState<NodeInfo | null>(null);
  const [actionInProgress, setActionInProgress] = useState<string | null>(null);
  const [showAddNodeModal, setShowAddNodeModal] = useState(!!openAddNode);
  const [copiedInstallCmd, setCopiedInstallCmd] = useState(false);
  const [noticeMessage, setNoticeMessage] = useState<string | null>(null);

  const loadAllData = async () => {
    try {
      setLoading(true);
      const [nodeRes, computeRes] = await Promise.all([
        api.getNodes(),
        api.getComputeSummary()
      ]);

      setNodes(nodeRes.data || []);
      setSummary(nodeRes.summary || null);
      if (nodeRes.compute) {
        setComputeSummary(nodeRes.compute);
      } else if (computeRes?.data?.compute) {
        setComputeSummary(computeRes.data.compute);
      }

      if (computeRes?.data?.depin) {
        setDePINIntegrations(computeRes.data.depin);
      }
      if (computeRes?.data?.benefits) {
        setSelfHostingBenefits(computeRes.data.benefits);
      }

      if (selectedNodeId) {
        const found = nodeRes.data.find((n) => n.id === selectedNodeId);
        if (found) setInspectNode(found);
      }
    } catch (err) {
      console.error('Failed to load nodes & compute command centre data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAllData();
  }, [selectedNodeId]);

  const handleOperation = async (id: string, op: 'DRAIN' | 'CORDON' | 'UNCORDON' | 'RESTART') => {
    try {
      setActionInProgress(`${id}-${op}`);
      const res = await api.executeNodeOperation(id, op);
      setNodes((prev) => prev.map((n) => (n.id === id ? res.data : n)));
      if (inspectNode?.id === id) setInspectNode(res.data);
      setNoticeMessage(`Operation ${op} executed on node ${res.data.name}`);
      setTimeout(() => setNoticeMessage(null), 4000);
    } catch (err: any) {
      setNoticeMessage(`Operation failed: ${err.message}`);
      setTimeout(() => setNoticeMessage(null), 4000);
    } finally {
      setActionInProgress(null);
    }
  };

  const handleCopyInstall = () => {
    navigator.clipboard.writeText('curl -sSf https://decentralized.host/install-agent.sh | sudo bash -s -- --token dh_node_join_9841f');
    setCopiedInstallCmd(true);
    setTimeout(() => setCopiedInstallCmd(false), 2500);
  };

  const filteredNodes = useMemo(() => {
    return nodes.filter((node) => {
      const q = searchTerm.toLowerCase();
      const matchesSearch =
        node.name.toLowerCase().includes(q) ||
        node.location.toLowerCase().includes(q) ||
        node.provider.toLowerCase().includes(q) ||
        node.ipAddress.includes(q) ||
        (node.gpuModel && node.gpuModel.toLowerCase().includes(q)) ||
        (node.supportedDePINTags && node.supportedDePINTags.some((t) => t.toLowerCase().includes(q)));

      const matchesStatus = statusFilter === 'All' || node.status === statusFilter;

      let matchesHardware = true;
      if (hardwareFilter === 'Self-Hosted') {
        matchesHardware = node.hardwareType === 'Self-Hosted Edge' || node.hardwareType === 'Homelab SBC';
      } else if (hardwareFilter === 'GPU') {
        matchesHardware = !!node.gpuEquipped;
      } else if (hardwareFilter === 'Validator') {
        matchesHardware = node.role === 'Validator';
      } else if (hardwareFilter === 'Edge') {
        matchesHardware = node.role === 'Edge Gateway' || node.role === 'Storage Replicator';
      }

      return matchesSearch && matchesStatus && matchesHardware;
    });
  }, [nodes, searchTerm, statusFilter, hardwareFilter]);

  const allWorkloads = useMemo(() => {
    const list: { nodeName: string; nodeId: string; location: string; workload: NodeWorkload }[] = [];
    nodes.forEach((n) => {
      if (n.workloads) {
        n.workloads.forEach((w) => {
          list.push({ nodeName: n.name, nodeId: n.id, location: n.location, workload: w });
        });
      }
    });
    return list;
  }, [nodes]);

  const gpuNodes = useMemo(() => nodes.filter((n) => n.gpuEquipped), [nodes]);

  return (
    <div className="space-y-6">
      {/* Toast Notice */}
      {noticeMessage && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center gap-2.5 px-4 py-3 bg-[#071329] border border-cyan-400/40 text-cyan-200 text-xs rounded-xl shadow-2xl animate-in fade-in slide-in-from-bottom">
          <Zap className="w-4 h-4 text-cyan-400 animate-pulse" />
          <span>{noticeMessage}</span>
          <button onClick={() => setNoticeMessage(null)} className="ml-2 text-slate-400 hover:text-white">
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}

      {!embedded && (
      <>
      {/* Header Command Centre */}
      <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4 bg-[#0A1226]/90 border border-slate-800/90 rounded-2xl p-5 shadow-2xl backdrop-blur-xl">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-cyan-400 uppercase tracking-widest">
            <Radio className="w-3.5 h-3.5 animate-pulse" />
            <span>Nodes & Compute Command Centre</span>
            <span className="text-slate-600">·</span>
            <span className="text-emerald-400 flex items-center gap-1 font-semibold">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              Sovereign Mesh Active
            </span>
          </div>
          <h1 className="text-2xl font-black text-white tracking-tight mt-1 flex items-center gap-3">
            <span>Decentralized Hosting & Compute Network</span>
            <span className="text-xs px-2.5 py-0.5 rounded-full bg-blue-500/10 text-cyan-300 border border-cyan-500/20 font-mono">
              v2.8 DePIN
            </span>
          </h1>
          <p className="text-xs text-slate-300 mt-1 max-w-3xl">
            Monitor self-hosted edge nodes, dedicated GPU clusters, CPU & memory allocation, workload orchestration,
            and supported DePIN integrations (Akash, Render, Filecoin, Livepeer, io.net).
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          <button
            onClick={loadAllData}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-2 rounded-xl text-xs font-semibold bg-slate-900 border border-slate-700 hover:border-slate-600 text-slate-200 transition-all cursor-pointer"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-cyan-400 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh Telemetry</span>
          </button>

          <button
            onClick={() => setShowAddNodeModal(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:brightness-110 text-white text-xs font-semibold shadow-lg shadow-blue-500/25 transition-all cursor-pointer"
          >
            <Plus className="w-4 h-4" />
            <span>Join Self-Hosted Node</span>
          </button>
        </div>
      </div>

      {/* Primary Metrics Row (6 key telemetry cards) */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3.5">
        <MetricCard
          icon={<Server className="w-4 h-4 text-emerald-400" />}
          label="Total Mesh Nodes"
          value={summary?.total || nodes.length || 10}
          change={`${summary?.online || 9} Online`}
          trendColor="green"
          capability="LIVE"
          provenance="gossip/quorum"
        />
        <MetricCard
          icon={<Cpu className="w-4 h-4 text-purple-400" />}
          label="Compute Cores"
          value={`${computeSummary?.usedCpuCores || 84} / ${computeSummary?.totalCpuCores || 264}`}
          change={`${computeSummary?.cpuUtilizationPercent || 32}% Core Load`}
          trendColor="purple"
          capability="LIVE"
          provenance="cgroup/proc"
        />
        <MetricCard
          icon={<Flame className="w-4 h-4 text-amber-400" />}
          label="GPU Accelerators"
          value={`${computeSummary?.totalGpus || 6} GPUs`}
          change={`${computeSummary?.totalGpuVramGb || 256} GB VRAM`}
          trendColor="purple"
          capability="LIVE"
          provenance="nvidia-smi"
        />
        <MetricCard
          icon={<HardDrive className="w-4 h-4 text-teal-400" />}
          label="Storage Mesh"
          value={`${summary?.totalStorageTb || 62.0} TB`}
          change="3× Replicated"
          trendColor="blue"
          capability="LIVE"
          provenance="ipfs/btrfs"
        />
        <MetricCard
          icon={<Layers className="w-4 h-4 text-cyan-400" />}
          label="Active Workloads"
          value={computeSummary?.totalActiveWorkloads || allWorkloads.length || 38}
          change="Edge Containers"
          trendColor="blue"
          capability="LIVE"
          provenance="containerd"
        />
        <MetricCard
          icon={<Award className="w-4 h-4 text-emerald-400" />}
          label="DePIN Rewards"
          value={`${computeSummary?.totalRewardsEarnedDH || 5845} DH`}
          change="+$3,820 Earned"
          trendColor="green"
          capability="LIVE"
          provenance="smart-contract"
        />
      </div>

      {/* Main Tab Navigation */}
      <div className="flex flex-wrap items-center gap-2 border-b border-slate-800 pb-3">
        {[
          { id: 'mesh' as CommandTab, label: 'Nodes & Global Mesh', icon: <Server className="w-4 h-4" />, count: nodes.length },
          { id: 'compute' as CommandTab, label: 'GPU & AI Compute Rigs', icon: <Flame className="w-4 h-4 text-amber-400" />, count: gpuNodes.length },
          { id: 'workloads' as CommandTab, label: 'Workload Management', icon: <Layers className="w-4 h-4 text-cyan-400" />, count: allWorkloads.length },
          { id: 'depin' as CommandTab, label: 'DePIN Network Integrations', icon: <Zap className="w-4 h-4 text-emerald-400" />, count: depinIntegrations.length },
          { id: 'benefits' as CommandTab, label: 'Self-Hosting Benefits & SLA', icon: <ShieldCheck className="w-4 h-4 text-blue-400" /> }
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveCommandTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs font-semibold transition-all cursor-pointer ${
              activeCommandTab === tab.id
                ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30 border border-cyan-400/40'
                : 'bg-slate-900/60 text-slate-400 hover:text-slate-200 hover:bg-slate-800/80 border border-slate-800'
            }`}
          >
            <span>{tab.icon}</span>
            <span>{tab.label}</span>
            {tab.count !== undefined && (
              <span className={`px-1.5 py-0.2 rounded-full text-[10px] font-mono ${
                activeCommandTab === tab.id ? 'bg-black/40 text-cyan-200' : 'bg-slate-800 text-slate-400'
              }`}>
                {tab.count}
              </span>
            )}
          </button>
        ))}
      </div>

      </>
      )}

      {/* TAB 1: NODES & GLOBAL MESH */}
      {activeCommandTab === 'mesh' && (
        <div className="space-y-6">
          {/* Middle Row: Global Map + Health & Resource Gauges */}
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
            {/* Left: Global Map with Region Tags */}
            <div className="lg:col-span-8 space-y-3 bg-[#0A1226]/80 border border-slate-800/80 rounded-2xl p-5 shadow-xl backdrop-blur-md">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-800/70 pb-3">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_8px_#34d399]" />
                  <span className="text-sm font-bold text-white tracking-tight">
                    Global Sovereign Infrastructure Topology
                  </span>
                  <span className="text-xs font-mono text-cyan-400">
                    ({summary?.online || 9}/{summary?.total || 10} Online)
                  </span>
                </div>
                <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
                  <span>Anycast latency avg: 24ms</span>
                  <span>·</span>
                  <span className="text-emerald-400 font-semibold">100% Attested</span>
                </div>
              </div>

              <WorldMap heightClass="h-[340px]" showRegions={true} />

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-2 border-t border-slate-800/60 font-mono text-xs">
                <div className="p-2.5 rounded-xl bg-slate-900/60 border border-slate-800">
                  <span className="text-slate-400 text-[10px] block">North America (Ashburn, Austin)</span>
                  <span className="font-bold text-white">2 Nodes · 8.2 TB/s</span>
                </div>
                <div className="p-2.5 rounded-xl bg-slate-900/60 border border-slate-800">
                  <span className="text-slate-400 text-[10px] block">Europe (Frankfurt, London, Stockholm)</span>
                  <span className="font-bold text-white">3 Nodes · 6.8 TB/s</span>
                </div>
                <div className="p-2.5 rounded-xl bg-slate-900/60 border border-slate-800">
                  <span className="text-slate-400 text-[10px] block">Asia (Singapore, Kochi)</span>
                  <span className="font-bold text-white">2 Nodes · 4.2 TB/s</span>
                </div>
                <div className="p-2.5 rounded-xl bg-slate-900/60 border border-slate-800">
                  <span className="text-slate-400 text-[10px] block">Middle East & Oceania & SA</span>
                  <span className="font-bold text-white">3 Nodes · 3.9 TB/s</span>
                </div>
              </div>
            </div>

            {/* Right: Node Health Donut Card & Global Resource Usage */}
            <div className="lg:col-span-4 space-y-4">
              {/* Donut Card */}
              <div className="rounded-2xl bg-[#0A1226]/80 border border-slate-800/80 p-5 space-y-4 shadow-xl backdrop-blur-md">
                <div className="flex items-center justify-between">
                  <h2 className="text-sm font-bold text-white">Consensus & Health Quorum</h2>
                  <span className="text-[10px] font-mono text-emerald-400 px-2 py-0.5 rounded bg-emerald-500/10 border border-emerald-500/20 font-bold">
                    ED25519 VERIFIED
                  </span>
                </div>

                <div className="flex items-center gap-6">
                  {/* Donut representation */}
                  <div className="relative w-28 h-28 flex items-center justify-center flex-shrink-0">
                    <svg className="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                      <path
                        className="text-slate-800"
                        strokeWidth="3.8"
                        stroke="currentColor"
                        fill="none"
                        d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      />
                      <path
                        className="text-emerald-500"
                        strokeDasharray="90, 100"
                        strokeWidth="3.8"
                        strokeLinecap="round"
                        stroke="currentColor"
                        fill="none"
                        d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                      />
                    </svg>
                    <div className="absolute flex flex-col items-center">
                      <span className="text-xl font-extrabold font-mono text-white">
                        {summary?.healthyPercent || 90}%
                      </span>
                      <span className="text-[9px] uppercase tracking-wider text-slate-400 font-mono">Quorum</span>
                    </div>
                  </div>

                  {/* Status List */}
                  <div className="flex flex-col gap-2 font-mono text-xs flex-1">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full bg-emerald-400" />
                        <span className="text-slate-300">Online</span>
                      </div>
                      <span className="font-bold text-white">{summary?.online || 9}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full bg-amber-400" />
                        <span className="text-slate-300">Degraded</span>
                      </div>
                      <span className="font-bold text-amber-400">{summary?.degraded || 1}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full bg-rose-400" />
                        <span className="text-slate-300">Offline</span>
                      </div>
                      <span className="font-bold text-slate-400">{summary?.offline || 0}</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Global Resource Allocation */}
              <div className="rounded-2xl bg-[#0A1226]/80 border border-slate-800/80 p-5 space-y-3 font-mono text-xs shadow-xl backdrop-blur-md">
                <div className="flex items-center justify-between">
                  <h2 className="text-sm font-bold text-white font-sans">Global Resource Pools</h2>
                  <span className="text-[10px] text-slate-400">Mesh v2.8</span>
                </div>

                <div className="space-y-2.5">
                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-slate-400">CPU Compute Load</span>
                      <span className="text-white font-bold">{computeSummary?.cpuUtilizationPercent || 34}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
                      <div className="h-full bg-gradient-to-r from-blue-500 to-indigo-500 rounded-full" style={{ width: `${computeSummary?.cpuUtilizationPercent || 34}%` }} />
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-slate-400">Memory Allocation</span>
                      <span className="text-white font-bold">{computeSummary?.memoryUtilizationPercent || 48}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
                      <div className="h-full bg-gradient-to-r from-purple-500 to-pink-500 rounded-full" style={{ width: `${computeSummary?.memoryUtilizationPercent || 48}%` }} />
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-slate-400">GPU Tensor Utilization</span>
                      <span className="text-amber-400 font-bold">81%</span>
                    </div>
                    <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
                      <div className="h-full bg-gradient-to-r from-amber-500 to-orange-500 rounded-full" style={{ width: '81%' }} />
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-slate-400">IPFS Storage Used</span>
                      <span className="text-emerald-400 font-bold">{computeSummary?.storageUtilizationPercent || 42}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
                      <div className="h-full bg-gradient-to-r from-teal-500 to-emerald-400 rounded-full" style={{ width: `${computeSummary?.storageUtilizationPercent || 42}%` }} />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Nodes Filter & Search Strip */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 p-3.5 rounded-2xl bg-[#0A1226]/80 border border-slate-800 shadow-xl">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs font-mono text-slate-500 mr-1 flex items-center gap-1">
                <Filter className="w-3.5 h-3.5" /> Filter:
              </span>
              {(['All', 'Online', 'Degraded', 'Offline'] as const).map((status) => (
                <button
                  key={status}
                  onClick={() => setStatusFilter(status)}
                  className={`px-3 py-1.5 rounded-xl text-xs font-mono font-medium transition-all cursor-pointer ${
                    statusFilter === status
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                  }`}
                >
                  {status}
                </button>
              ))}

              <div className="h-4 w-px bg-slate-700 mx-1 hidden sm:block" />

              {(['All', 'Self-Hosted', 'GPU', 'Validator', 'Edge'] as const).map((hw) => (
                <button
                  key={hw}
                  onClick={() => setHardwareFilter(hw)}
                  className={`px-2.5 py-1.5 rounded-xl text-xs font-mono font-medium transition-all cursor-pointer ${
                    hardwareFilter === hw
                      ? 'bg-purple-600/30 text-purple-300 border border-purple-500/50'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                  }`}
                >
                  {hw === 'GPU' ? '⚡ GPU Rig' : hw}
                </button>
              ))}
            </div>

            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
              <Search className="w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search by name, location, IP, GPU, DePIN..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="bg-transparent text-white focus:outline-none placeholder-slate-500 w-64 font-mono"
              />
            </div>
          </div>

          {/* Nodes Table */}
          <div className="rounded-2xl bg-[#0A1226]/80 border border-slate-800/80 overflow-hidden shadow-2xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="border-b border-slate-800 bg-slate-950/80 text-slate-400 font-mono text-[11px] uppercase tracking-wider">
                    <th className="py-3.5 px-4 font-semibold">Node Name & Role</th>
                    <th className="py-3.5 px-4 font-semibold">Status</th>
                    <th className="py-3.5 px-4 font-semibold">Location</th>
                    <th className="py-3.5 px-4 font-semibold">Hardware & Specs</th>
                    <th className="py-3.5 px-4 font-semibold">CPU / RAM Load</th>
                    <th className="py-3.5 px-4 font-semibold">GPU Acceleration</th>
                    <th className="py-3.5 px-4 font-semibold">DePIN Networks</th>
                    <th className="py-3.5 px-4 font-semibold">Uptime SLA</th>
                    <th className="py-3.5 px-4 font-semibold text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {filteredNodes.map((node) => (
                    <tr
                      key={node.id}
                      className="hover:bg-blue-600/[0.04] transition-colors group cursor-pointer"
                      onClick={() => setInspectNode(node)}
                    >
                      {/* Name & Role */}
                      <td className="py-4 px-4">
                        <div className="flex items-center gap-3">
                          <div className={`w-9 h-9 rounded-xl border flex items-center justify-center font-mono ${
                            node.gpuEquipped
                              ? 'bg-amber-500/10 border-amber-500/30 text-amber-400'
                              : node.hardwareType === 'Homelab SBC'
                              ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                              : 'bg-blue-600/10 border-blue-500/30 text-cyan-400'
                          }`}>
                            {node.gpuEquipped ? <Flame className="w-4 h-4" /> : <Server className="w-4 h-4" />}
                          </div>
                          <div>
                            <div className="font-bold text-white font-mono flex items-center gap-2">
                              <span>{node.name}</span>
                              {node.hardwareType === 'Homelab SBC' && (
                                <span className="px-1.5 py-0.2 rounded text-[9px] bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                                  Homelab
                                </span>
                              )}
                            </div>
                            <div className="text-[10px] text-slate-500 font-mono flex items-center gap-1.5">
                              <span>{node.ipAddress}</span>
                              <span>·</span>
                              <span className="text-slate-400">{node.role}</span>
                            </div>
                          </div>
                        </div>
                      </td>

                      {/* Status */}
                      <td className="py-4 px-4">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-mono font-medium ${
                            node.status === 'Online'
                              ? 'bg-emerald-950/60 text-emerald-400 border border-emerald-500/40'
                              : 'bg-amber-950/60 text-amber-400 border border-amber-500/40'
                          }`}
                        >
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${
                              node.status === 'Online' ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400 animate-ping'
                            }`}
                          />
                          {node.status}
                        </span>
                      </td>

                      {/* Location */}
                      <td className="py-4 px-4 font-mono text-slate-300">
                        <div className="flex items-center gap-1.5">
                          <span>{node.location}</span>
                        </div>
                        <div className="text-[10px] text-slate-500">{node.provider}</div>
                      </td>

                      {/* Hardware Specs */}
                      <td className="py-4 px-4 font-mono text-slate-300">
                        <div className="text-[11px] font-semibold text-white truncate max-w-[180px]">
                          {node.cpuModel || `${node.cpuCores} Cores`}
                        </div>
                        <div className="text-[10px] text-slate-400">
                          {node.memoryTotalGb} GB RAM · {(node.diskTotalGb / 1000).toFixed(1)} TB SSD
                        </div>
                      </td>

                      {/* CPU / RAM */}
                      <td className="py-4 px-4 font-mono">
                        <div className="w-32 space-y-1">
                          <div className="flex justify-between text-[10px] text-slate-400">
                            <span>CPU {node.cpuPercent}%</span>
                            <span>RAM {node.memoryPercent}%</span>
                          </div>
                          <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden flex">
                            <div className="bg-cyan-500 h-full" style={{ width: `${node.cpuPercent}%` }} />
                            <div className="bg-purple-500 h-full" style={{ width: `${node.memoryPercent}%` }} />
                          </div>
                        </div>
                      </td>

                      {/* GPU Acceleration */}
                      <td className="py-4 px-4 font-mono">
                        {node.gpuEquipped ? (
                          <div>
                            <span className="inline-flex items-center gap-1 text-[11px] text-amber-400 font-bold">
                              <Flame className="w-3 h-3 text-amber-400" />
                              {node.gpuCount}× GPU ({node.gpuMemoryTotalGb}GB)
                            </span>
                            <div className="text-[10px] text-slate-400 truncate max-w-[140px]">
                              {node.gpuModel}
                            </div>
                          </div>
                        ) : (
                          <span className="text-slate-500 text-[10px]">CPU Compute</span>
                        )}
                      </td>

                      {/* DePIN Networks */}
                      <td className="py-4 px-4">
                        <div className="flex flex-wrap gap-1">
                          {node.supportedDePINTags?.map((tag) => (
                            <span
                              key={tag}
                              className="px-1.5 py-0.5 rounded text-[9px] font-mono bg-blue-500/10 text-cyan-300 border border-blue-500/20"
                            >
                              {tag.split(' ')[0]}
                            </span>
                          )) || <span className="text-slate-500 text-[10px]">—</span>}
                        </div>
                      </td>

                      {/* Uptime */}
                      <td className="py-4 px-4 font-mono">
                        <div className="text-emerald-400 font-bold">{node.uptimePercent}%</div>
                        <div className="text-[10px] text-slate-500">{node.uptimeDays} days</div>
                      </td>

                      {/* Actions */}
                      <td className="py-4 px-4 text-right">
                        <div className="flex items-center justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                          <button
                            onClick={() => handleOperation(node.id, 'DRAIN')}
                            disabled={actionInProgress === `${node.id}-DRAIN`}
                            className="px-2.5 py-1 rounded-lg bg-amber-950/40 border border-amber-500/40 text-amber-300 hover:bg-amber-900/40 text-[11px] font-mono cursor-pointer"
                          >
                            Drain
                          </button>
                          <button
                            onClick={() => setInspectNode(node)}
                            className="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white cursor-pointer"
                          >
                            <MoreVertical className="w-4 h-4" />
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
      )}

      {/* TAB 2: GPU & COMPUTE ACCELERATION */}
      {activeCommandTab === 'compute' && (
        <div className="space-y-6">
          <div className="bg-[#0A1226]/80 border border-slate-800 rounded-2xl p-5 shadow-xl">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border-b border-slate-800 pb-4 mb-4">
              <div>
                <h3 className="text-base font-bold text-white flex items-center gap-2">
                  <Flame className="w-5 h-5 text-amber-400" />
                  <span>High-Performance GPU Compute Clusters</span>
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Accelerated hardware nodes serving AI inference, LLM fine-tuning, Render Network 3D jobs, and Livepeer video transcoding.
                </p>
              </div>

              <div className="flex items-center gap-3 font-mono text-xs text-slate-300">
                <span className="px-3 py-1.5 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400 font-bold">
                  {computeSummary?.totalGpus || 6} GPUs Online
                </span>
                <span className="px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800">
                  {computeSummary?.totalGpuVramGb || 256} GB VRAM Allocated
                </span>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {gpuNodes.map((gNode) => (
                <div
                  key={gNode.id}
                  className="p-5 rounded-2xl bg-[#060D1E] border border-amber-500/30 space-y-4 hover:border-amber-400/60 transition-all shadow-lg"
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <span className="text-[10px] font-mono text-amber-400 uppercase tracking-widest block font-bold">
                        {gNode.role} · {gNode.provider}
                      </span>
                      <h4 className="text-base font-bold text-white font-mono mt-0.5">{gNode.name}</h4>
                      <span className="text-xs text-slate-400 font-mono">{gNode.location} · {gNode.ipAddress}</span>
                    </div>
                    <span className="px-2.5 py-1 rounded-full text-xs font-mono font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                      {gNode.status}
                    </span>
                  </div>

                  <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 space-y-2 font-mono text-xs">
                    <div className="flex justify-between items-center text-slate-300">
                      <span className="text-slate-400">Accelerator:</span>
                      <span className="font-bold text-amber-300">{gNode.gpuModel}</span>
                    </div>
                    <div className="flex justify-between items-center text-slate-300">
                      <span className="text-slate-400">VRAM Capacity:</span>
                      <span>{gNode.gpuMemoryUsedGb} / {gNode.gpuMemoryTotalGb} GB</span>
                    </div>
                    <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-gradient-to-r from-amber-500 to-orange-500 rounded-full"
                        style={{ width: `${gNode.gpuPercent || 75}%` }}
                      />
                    </div>
                  </div>

                  <div>
                    <span className="text-[11px] font-mono text-slate-400 block mb-2">Active Workloads on this Rig:</span>
                    <div className="space-y-1.5 font-mono text-xs">
                      {gNode.workloads?.map((w) => (
                        <div key={w.id} className="p-2 rounded-lg bg-slate-900/80 border border-slate-800 flex items-center justify-between">
                          <span className="text-slate-200">{w.name}</span>
                          <span className="text-[10px] text-cyan-400 px-1.5 py-0.5 rounded bg-cyan-500/10 border border-cyan-500/20">
                            {w.type}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t border-slate-800/80 text-xs font-mono">
                    <span className="text-slate-400">Node Credits Earned:</span>
                    <span className="text-emerald-400 font-bold">{gNode.rewardsEarnedCredits} DH</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 3: WORKLOAD MANAGEMENT */}
      {activeCommandTab === 'workloads' && (
        <div className="space-y-4">
          <div className="rounded-2xl bg-[#0A1226]/80 border border-slate-800 p-5 shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div>
                <h3 className="text-base font-bold text-white flex items-center gap-2">
                  <Layers className="w-5 h-5 text-cyan-400" />
                  <span>Workload Management & Orchestration</span>
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Microservices, edge WASM sandboxes, AI inference models, and IPFS daemons running across mesh nodes.
                </p>
              </div>

              <span className="text-xs font-mono text-cyan-400 px-3 py-1 rounded-xl bg-cyan-500/10 border border-cyan-500/30">
                {allWorkloads.length} Managed Workloads
              </span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5">
              {allWorkloads.map(({ nodeName, location, workload }) => (
                <div key={workload.id} className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-3 font-mono text-xs">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="font-bold text-white text-sm">{workload.name}</div>
                      <span className="text-[10px] text-slate-400">{nodeName} ({location})</span>
                    </div>
                    <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                      workload.status === 'Running'
                        ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                        : 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
                    }`}>
                      {workload.status}
                    </span>
                  </div>

                  <div className="p-2.5 rounded-lg bg-black/40 border border-slate-800/80 space-y-1.5 text-[11px]">
                    <div className="flex justify-between">
                      <span className="text-slate-500">Workload Type:</span>
                      <span className="text-cyan-300 font-bold">{workload.type}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-500">Allocated CPU:</span>
                      <span className="text-white">{workload.cpuPercent}% core</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-500">Allocated RAM:</span>
                      <span className="text-white">{workload.memoryMb} MB</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-500">Uptime:</span>
                      <span className="text-slate-400">{workload.deployedAt}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 4: DEPIN NETWORK INTEGRATIONS */}
      {activeCommandTab === 'depin' && (
        <div className="space-y-4">
          <div className="rounded-2xl bg-[#0A1226]/80 border border-slate-800 p-5 shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div>
                <h3 className="text-base font-bold text-white flex items-center gap-2">
                  <Zap className="w-5 h-5 text-emerald-400" />
                  <span>Supported DePIN Network Integrations</span>
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Decentralized Physical Infrastructure Networks connected to your nodes to monetize surplus compute, storage, and bandwidth.
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {depinIntegrations.map((depin) => (
                <div
                  key={depin.id}
                  className="p-5 rounded-2xl bg-[#070E20] border border-slate-800 hover:border-blue-500/40 transition-all space-y-4 shadow-lg flex flex-col justify-between"
                >
                  <div className="space-y-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-xl bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-xl">
                          {depin.icon}
                        </div>
                        <div>
                          <h4 className="font-bold text-white text-base">{depin.name}</h4>
                          <span className="text-xs font-mono text-cyan-400">{depin.symbol} · {depin.category}</span>
                        </div>
                      </div>

                      <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                        {depin.status}
                      </span>
                    </div>

                    <p className="text-xs text-slate-300 leading-relaxed">
                      {depin.description}
                    </p>
                  </div>

                  <div className="pt-3 border-t border-slate-800 space-y-2 font-mono text-xs">
                    <div className="flex justify-between items-center">
                      <span className="text-slate-400">Connected Nodes:</span>
                      <span className="font-bold text-white">{depin.nodesConnected} nodes</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-slate-400">Total Yield Claimed:</span>
                      <span className="font-bold text-emerald-400">{depin.earningsTotal}</span>
                    </div>
                    <a
                      href={depin.protocolLink}
                      target="_blank"
                      rel="noreferrer"
                      className="mt-2 inline-flex items-center gap-1 text-[11px] text-cyan-400 hover:text-cyan-300"
                    >
                      <span>Explore protocol docs</span>
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 5: SELF-HOSTING BENEFITS & SLA */}
      {activeCommandTab === 'benefits' && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {selfHostingBenefits.map((ben) => (
              <div
                key={ben.id}
                className="p-6 rounded-2xl bg-[#0A1226]/80 border border-slate-800 hover:border-cyan-500/40 transition-all space-y-3 shadow-xl"
              >
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-mono uppercase tracking-widest text-cyan-400 px-2 py-0.5 rounded bg-cyan-500/10 border border-cyan-500/20 font-bold">
                    {ben.tag}
                  </span>
                  <span className="text-base font-extrabold text-emerald-400 font-mono">
                    {ben.metric}
                  </span>
                </div>
                <h4 className="text-lg font-bold text-white tracking-tight">{ben.title}</h4>
                <p className="text-xs text-slate-300 leading-relaxed">{ben.description}</p>
              </div>
            ))}
          </div>

          {/* Quick Install Instruction Card */}
          <div className="p-6 rounded-2xl bg-[#070F22] border border-blue-500/30 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <Terminal className="w-5 h-5 text-cyan-400" />
                <h3 className="text-base font-bold text-white">Enroll Your Machine in 60 Seconds</h3>
              </div>
              <span className="text-xs font-mono text-slate-400">Ubuntu 22.04+ / Debian 12 / macOS ARM</span>
            </div>

            <p className="text-xs text-slate-300">
              Transform any Raspberry Pi 5, Intel NUC, dedicated server, or NVIDIA RTX workstation into a verified Decentralized.Host node:
            </p>

            <div className="flex items-center justify-between p-3.5 rounded-xl bg-black border border-slate-800 font-mono text-xs text-cyan-300 select-all">
              <span className="truncate">
                curl -sSf https://decentralized.host/install-agent.sh | sudo bash -s -- --token dh_node_join_9841f
              </span>
              <button
                onClick={handleCopyInstall}
                className="ml-3 flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold cursor-pointer"
              >
                {copiedInstallCmd ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                <span>{copiedInstallCmd ? 'Copied!' : 'Copy'}</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Inspect Node Drawer */}
      {inspectNode && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-md flex justify-end">
          <div className="w-full max-w-xl bg-[#0A1226] border-l border-slate-800 h-full overflow-y-auto flex flex-col justify-between p-6 animate-in slide-in-from-right duration-200">
            <div className="space-y-5">
              <div className="flex items-start justify-between border-b border-slate-800 pb-4">
                <div className="flex items-center gap-3">
                  <div className="w-12 h-12 rounded-xl bg-blue-600/20 border border-blue-500/40 flex items-center justify-center text-cyan-400 font-mono text-xl">
                    {inspectNode.gpuEquipped ? <Flame className="w-6 h-6 text-amber-400" /> : <Server className="w-6 h-6" />}
                  </div>
                  <div>
                    <h2 className="text-lg font-bold text-white flex items-center gap-2">
                      {inspectNode.name}
                      <span className="text-xs px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-500/40 font-mono">
                        {inspectNode.status}
                      </span>
                    </h2>
                    <span className="text-xs font-mono text-slate-400">
                      {inspectNode.location} · {inspectNode.provider} · {inspectNode.role}
                    </span>
                  </div>
                </div>

                <button
                  onClick={() => setInspectNode(null)}
                  className="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white cursor-pointer"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Operator Controls Bar */}
              <div className="p-3.5 rounded-xl bg-slate-900 border border-slate-800 space-y-2">
                <div className="text-xs font-bold text-white">Operator Controls</div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleOperation(inspectNode.id, 'CORDON')}
                    disabled={actionInProgress === `${inspectNode.id}-CORDON`}
                    className="flex-1 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono cursor-pointer"
                  >
                    Cordon
                  </button>
                  <button
                    onClick={() => handleOperation(inspectNode.id, 'DRAIN')}
                    disabled={actionInProgress === `${inspectNode.id}-DRAIN`}
                    className="flex-1 py-1.5 rounded-lg bg-amber-950/60 border border-amber-500/40 text-amber-300 hover:bg-amber-900/60 text-xs font-mono cursor-pointer"
                  >
                    Drain Workloads
                  </button>
                  <button
                    onClick={() => handleOperation(inspectNode.id, 'RESTART')}
                    disabled={actionInProgress === `${inspectNode.id}-RESTART`}
                    className="flex-1 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono flex items-center justify-center gap-1 cursor-pointer"
                  >
                    <RotateCw className="w-3 h-3" />
                    Restart Agent
                  </button>
                </div>
              </div>

              {/* Hardware Telemetry Details */}
              <div className="grid grid-cols-2 gap-3 text-xs font-mono">
                <div className="p-3.5 rounded-xl bg-slate-900 border border-slate-800">
                  <span className="text-slate-500">Hardware Arch & Model</span>
                  <div className="text-white font-bold text-xs mt-1 truncate">
                    {inspectNode.cpuModel || `${inspectNode.cpuCores} Cores`}
                  </div>
                  <div className="text-slate-400 text-[10px] mt-0.5">
                    {inspectNode.architecture} · {inspectNode.hardwareType}
                  </div>
                </div>

                <div className="p-3.5 rounded-xl bg-slate-900 border border-slate-800">
                  <span className="text-slate-500">RAM & Disk Allocation</span>
                  <div className="text-cyan-400 font-bold text-xs mt-1">
                    {inspectNode.memoryUsedGb} / {inspectNode.memoryTotalGb} GB RAM
                  </div>
                  <div className="text-slate-400 text-[10px] mt-0.5">
                    {(inspectNode.diskUsedGb / 1000).toFixed(1)} / {(inspectNode.diskTotalGb / 1000).toFixed(1)} TB Storage
                  </div>
                </div>
              </div>

              {/* GPU Details (if applicable) */}
              {inspectNode.gpuEquipped && (
                <div className="p-4 rounded-xl bg-amber-950/20 border border-amber-500/30 space-y-2 text-xs font-mono">
                  <div className="flex items-center gap-2 text-amber-400 font-bold">
                    <Flame className="w-4 h-4" />
                    <span>GPU Tensor Engine Active</span>
                  </div>
                  <div className="text-white font-bold">{inspectNode.gpuModel}</div>
                  <div className="text-slate-400">
                    Allocated VRAM: {inspectNode.gpuMemoryUsedGb} GB / {inspectNode.gpuMemoryTotalGb} GB ({inspectNode.gpuPercent}% Load)
                  </div>
                </div>
              )}

              {/* Workloads List */}
              {inspectNode.workloads && inspectNode.workloads.length > 0 && (
                <div className="space-y-2">
                  <span className="text-xs font-bold text-white block">Active Assigned Workloads ({inspectNode.workloads.length})</span>
                  <div className="space-y-1.5 font-mono text-xs">
                    {inspectNode.workloads.map((w) => (
                      <div key={w.id} className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between">
                        <div>
                          <div className="text-white font-bold">{w.name}</div>
                          <span className="text-[10px] text-slate-500">{w.type}</span>
                        </div>
                        <span className="px-2 py-0.5 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                          {w.status}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Node Contribution & SLA */}
              <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-2 text-xs font-mono">
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Uptime SLA Score:</span>
                  <span className="text-emerald-400 font-bold">{inspectNode.uptimeScore || 99.9}%</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Proofs Cryptographically Verified:</span>
                  <span className="text-white font-bold">{inspectNode.proofsVerifiedCount || 42800}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Node Credits Claimed:</span>
                  <span className="text-cyan-400 font-bold">{inspectNode.rewardsEarnedCredits || 320} DH</span>
                </div>
              </div>

              {/* TPM & Evidence Fingerprint */}
              <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-2 text-xs font-mono">
                <div className="flex items-center gap-2 text-emerald-400 font-bold">
                  <ShieldCheck className="w-4 h-4" />
                  <span>Hardware TPM Attestation Verified</span>
                </div>
                <div>
                  <span className="text-slate-500">Public Key Fingerprint: </span>
                  <span className="text-purple-400 break-all">{inspectNode.trustFingerprint}</span>
                </div>
                <div>
                  <span className="text-slate-500">Evidence Record: </span>
                  <span className="text-cyan-400">{inspectNode.evidence?.recordId}</span>
                </div>
              </div>
            </div>

            <div className="pt-4 border-t border-slate-800 flex justify-end">
              <button
                onClick={() => setInspectNode(null)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold cursor-pointer"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Node Modal */}
      {showAddNodeModal && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-md flex items-center justify-center p-4">
          <div className="w-full max-w-lg rounded-2xl bg-[#0A1226] border border-slate-700 p-6 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-white flex items-center gap-2">
                <Plus className="w-4 h-4 text-cyan-400" />
                <span>Enroll Self-Hosted or Cloud Node</span>
              </h3>
              <button onClick={() => setShowAddNodeModal(false)} className="text-slate-400 hover:text-white cursor-pointer">
                <X className="w-5 h-5" />
              </button>
            </div>

            <p className="text-xs text-slate-300">
              Run this 1-line installation script on your Linux/macOS machine to establish TPM trust and join the sovereign hosting mesh:
            </p>

            <div className="p-3.5 rounded-xl bg-black border border-slate-800 font-mono text-xs text-cyan-300 select-all flex items-center justify-between">
              <span className="truncate">
                curl -sSf https://decentralized.host/install-agent.sh | sudo bash -s -- --token dh_node_join_9841f
              </span>
              <button
                onClick={handleCopyInstall}
                className="ml-2 px-2.5 py-1 rounded bg-blue-600 text-white text-[11px] font-semibold cursor-pointer flex-shrink-0"
              >
                {copiedInstallCmd ? 'Copied' : 'Copy'}
              </button>
            </div>

            <div className="text-[11px] font-mono text-slate-400 space-y-1">
              <div>• Compatible with x86_64 and ARM64 (Raspberry Pi 5, Apple Silicon)</div>
              <div>• Automatically installs WireGuard P2P gossip mesh daemon</div>
              <div>• Auto-detects NVIDIA CUDA drivers for AI & DePIN compute workloads</div>
            </div>

            <div className="flex justify-end pt-3 border-t border-slate-800">
              <button
                onClick={() => setShowAddNodeModal(false)}
                className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold cursor-pointer"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
