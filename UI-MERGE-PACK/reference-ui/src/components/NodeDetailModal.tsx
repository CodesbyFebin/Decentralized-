import React, { useState } from 'react';
import {
  X,
  Server,
  Shield,
  ShieldAlert,
  Cpu,
  Layers,
  HardDrive,
  Activity,
  Terminal,
  Clock,
  Radio,
  Sliders,
  AlertTriangle,
  CheckCircle2,
  Lock,
  Boxes
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const NodeDetailModal: React.FC = () => {
  const {
    nodes,
    selectedNodeId,
    setSelectedNodeId,
    cordonNode,
    uncordonNode,
    drainNode,
    killSwitchNode,
    updateOwnerReserve,
  } = useNetwork();

  const [activeTab, setActiveTab] = useState<'overview' | 'ledger' | 'logs' | 'killswitch'>('overview');
  const node = nodes.find((n) => n.id === selectedNodeId);

  if (!node) return null;

  const [editReserveCpu, setEditReserveCpu] = useState(node.cpu.ownerReserve);
  const [editReserveRam, setEditReserveRam] = useState(node.memoryGb.ownerReserve);

  const handleSaveReserves = () => {
    updateOwnerReserve(node.id, editReserveCpu, editReserveRam);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-3xl max-h-[90vh] overflow-y-auto rounded-3xl bg-gradient-to-b from-[#0d1838] to-[#070d1e] border border-blue-500/30 p-6 shadow-2xl custom-scrollbar text-slate-200">
        {/* Close Button */}
        <button
          onClick={() => setSelectedNodeId(null)}
          className="absolute top-5 right-5 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-slate-700/50 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Node Header */}
        <div className="flex items-center gap-3.5 mb-6 pb-4 border-b border-blue-500/20">
          <div className="w-12 h-12 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 flex items-center justify-center text-cyan-300 shadow-lg shadow-cyan-500/20">
            <Server className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold text-white tracking-tight font-mono">{node.name}</h2>
              <span
                className={`px-2 py-0.5 rounded-full text-[10px] font-mono border ${
                  node.status === 'Online'
                    ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                    : 'bg-rose-500/15 text-rose-400 border-rose-500/30'
                }`}
              >
                ● {node.status}
              </span>
              {node.cordoned && (
                <span className="px-2 py-0.5 rounded-full text-[10px] font-mono bg-amber-500/15 text-amber-400 border border-amber-500/30">
                  CORDONED
                </span>
              )}
            </div>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              ID: {node.shortId} • {node.location.flag} {node.location.country} ({node.location.region}) • Agent {node.agentVersion}
            </p>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="flex items-center gap-2 mb-5 border-b border-slate-800 pb-2">
          {[
            { id: 'overview', label: 'Hardware & OS' },
            { id: 'ledger', label: 'Resource Ledger' },
            { id: 'logs', label: 'Telemetry & Logs' },
            { id: 'killswitch', label: 'Owner Kill Switch', isDanger: true },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-all ${
                activeTab === tab.id
                  ? tab.isDanger
                    ? 'bg-rose-600/20 text-rose-300 border border-rose-500/40'
                    : 'bg-blue-600/20 text-cyan-300 border border-cyan-500/40'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Overview Tab */}
        {activeTab === 'overview' && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-blue-500/20">
                <span className="text-[10px] uppercase font-mono text-slate-400 block mb-1">Architecture</span>
                <span className="text-sm font-bold text-white font-mono">{node.arch}</span>
              </div>
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-blue-500/20">
                <span className="text-[10px] uppercase font-mono text-slate-400 block mb-1">Operating System</span>
                <span className="text-sm font-bold text-white font-mono">{node.os}</span>
              </div>
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-blue-500/20">
                <span className="text-[10px] uppercase font-mono text-slate-400 block mb-1">Kernel</span>
                <span className="text-xs font-bold text-slate-300 font-mono truncate">{node.kernel}</span>
              </div>
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-blue-500/20">
                <span className="text-[10px] uppercase font-mono text-slate-400 block mb-1">Runtime Isolation</span>
                <span className="text-sm font-bold text-cyan-400 font-mono">{node.isolation}</span>
              </div>
            </div>

            <div className="p-4 rounded-2xl bg-slate-900/80 border border-blue-500/25 space-y-3 font-mono text-xs">
              <div className="flex items-center justify-between pb-2 border-b border-slate-800">
                <span className="text-slate-400">Cryptographic Node Identity</span>
                <span className="text-cyan-300">ed25519:3a9f018e4bc0... (Verified)</span>
              </div>
              <div className="flex items-center justify-between pb-2 border-b border-slate-800">
                <span className="text-slate-400">WireGuard Mesh IP</span>
                <span className="text-white">{node.ip}</span>
              </div>
              <div className="flex items-center justify-between pb-2 border-b border-slate-800">
                <span className="text-slate-400">Heartbeat Provenance</span>
                <span className="text-emerald-400">{node.lastHeartbeat} (Raft Quorum Validated)</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-slate-400">Workload Trust Class</span>
                <span className="text-purple-300 font-semibold">{node.trustClass}</span>
              </div>
            </div>

            {node.gpu && (
              <div className="p-3.5 rounded-2xl bg-gradient-to-r from-purple-950/40 to-slate-900/80 border border-purple-500/30 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-xl bg-purple-500/20 text-purple-300">
                    <Boxes className="w-5 h-5" />
                  </div>
                  <div>
                    <p className="text-xs font-bold text-white">{node.gpu.model}</p>
                    <p className="text-[11px] text-purple-300 font-mono">{node.gpu.vramGb} GB VRAM • CUDA / ROCm Qualified</p>
                  </div>
                </div>
                <span className="text-xs font-mono px-2.5 py-1 rounded-full bg-purple-500/20 text-purple-200 border border-purple-500/30">
                  {node.gpu.available} / {node.gpu.count} Available
                </span>
              </div>
            )}
          </div>
        )}

        {/* Resource Ledger Tab (Consumptive Model from Constitution) */}
        {activeTab === 'ledger' && (
          <div className="space-y-4">
            <div className="p-3.5 rounded-2xl bg-blue-950/30 border border-blue-500/25">
              <span className="text-xs font-mono font-bold text-cyan-300 block mb-1">
                Resource Ledger Equation (Master Blueprint Invariant):
              </span>
              <p className="text-xs text-slate-300 font-mono">
                AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
              </p>
            </div>

            <div className="grid grid-cols-3 gap-3">
              {/* CPU Breakdown */}
              <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-blue-500/20 space-y-2">
                <div className="flex justify-between items-center text-xs">
                  <span className="font-bold text-white">CPU Allocation</span>
                  <span className="font-mono text-cyan-400">{node.cpu.total} Cores</span>
                </div>
                <div className="space-y-1 text-[11px] font-mono text-slate-400">
                  <div className="flex justify-between"><span>Owner Reserve:</span><span className="text-cyan-300 font-bold">{node.cpu.ownerReserve}</span></div>
                  <div className="flex justify-between"><span>Marketplace Reserved:</span><span>{node.cpu.marketplaceReserved}</span></div>
                  <div className="flex justify-between"><span>Active Allocated:</span><span className="text-purple-300">{node.cpu.allocated}</span></div>
                  <div className="flex justify-between pt-1 border-t border-slate-800 text-emerald-400 font-bold">
                    <span>Available:</span><span>{node.cpu.available}</span>
                  </div>
                </div>
              </div>

              {/* Memory Breakdown */}
              <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-blue-500/20 space-y-2">
                <div className="flex justify-between items-center text-xs">
                  <span className="font-bold text-white">RAM Allocation</span>
                  <span className="font-mono text-purple-400">{node.memoryGb.total} GB</span>
                </div>
                <div className="space-y-1 text-[11px] font-mono text-slate-400">
                  <div className="flex justify-between"><span>Owner Reserve:</span><span className="text-purple-300 font-bold">{node.memoryGb.ownerReserve} GB</span></div>
                  <div className="flex justify-between"><span>Marketplace Reserved:</span><span>{node.memoryGb.marketplaceReserved} GB</span></div>
                  <div className="flex justify-between"><span>Active Allocated:</span><span className="text-cyan-300">{node.memoryGb.allocated} GB</span></div>
                  <div className="flex justify-between pt-1 border-t border-slate-800 text-emerald-400 font-bold">
                    <span>Available:</span><span>{node.memoryGb.available} GB</span>
                  </div>
                </div>
              </div>

              {/* Storage Breakdown */}
              <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-blue-500/20 space-y-2">
                <div className="flex justify-between items-center text-xs">
                  <span className="font-bold text-white">Storage Allocation</span>
                  <span className="font-mono text-emerald-400">{node.storageGb.total} GB</span>
                </div>
                <div className="space-y-1 text-[11px] font-mono text-slate-400">
                  <div className="flex justify-between"><span>Owner Reserve:</span><span className="text-emerald-300 font-bold">{node.storageGb.ownerReserve} GB</span></div>
                  <div className="flex justify-between"><span>Marketplace Reserved:</span><span>{node.storageGb.marketplaceReserved} GB</span></div>
                  <div className="flex justify-between"><span>Active Allocated:</span><span className="text-cyan-300">{node.storageGb.allocated} GB</span></div>
                  <div className="flex justify-between pt-1 border-t border-slate-800 text-emerald-400 font-bold">
                    <span>Available:</span><span>{node.storageGb.available} GB</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Adjust Reserves */}
            <div className="p-4 rounded-2xl bg-slate-900/90 border border-blue-500/25 space-y-3">
              <span className="text-xs font-bold text-white block">Adjust Sovereign Owner Reserves</span>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="flex justify-between text-xs mb-1 font-mono">
                    <span className="text-slate-400">CPU Reserve:</span>
                    <span className="text-cyan-300 font-bold">{editReserveCpu} Cores</span>
                  </div>
                  <input
                    type="range"
                    min="1"
                    max={node.cpu.total - 1}
                    value={editReserveCpu}
                    onChange={(e) => setEditReserveCpu(Number(e.target.value))}
                    className="w-full accent-cyan-400"
                  />
                </div>
                <div>
                  <div className="flex justify-between text-xs mb-1 font-mono">
                    <span className="text-slate-400">RAM Reserve:</span>
                    <span className="text-purple-300 font-bold">{editReserveRam} GB</span>
                  </div>
                  <input
                    type="range"
                    min="1"
                    max={node.memoryGb.total - 2}
                    value={editReserveRam}
                    onChange={(e) => setEditReserveRam(Number(e.target.value))}
                    className="w-full accent-purple-400"
                  />
                </div>
              </div>
              <div className="flex justify-end pt-2">
                <button
                  onClick={handleSaveReserves}
                  className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-blue-600 hover:bg-blue-500 transition-all shadow-md"
                >
                  Commit Reserve Policy
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Logs Tab */}
        {activeTab === 'logs' && (
          <div className="space-y-3">
            <div className="flex items-center justify-between text-xs font-mono text-slate-400">
              <span>dh-agent daemon logs (live stream)</span>
              <span className="flex items-center gap-1.5 text-emerald-400">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping" />
                Listening on WireGuard
              </span>
            </div>
            <div className="p-3.5 rounded-2xl bg-black/80 border border-slate-800 font-mono text-[11px] text-slate-300 max-h-72 overflow-y-auto custom-scrollbar space-y-1">
              <p className="text-slate-500">[10:14:12] dh-agent starting in rootless cgroup mode...</p>
              <p className="text-cyan-400">[10:14:13] WireGuard overlay interface dh0 active (100.64.0.12)</p>
              <p className="text-emerald-400">[10:14:14] Raft challenge received from coordinator, signed with ed25519</p>
              <p className="text-slate-300">[10:14:18] Hardware probe complete: 12 cores, 32GB RAM, KVM enabled</p>
              <p className="text-purple-400">[10:14:22] Signed metering record committed (0.42 TB·h storage, 1240 CPU·s)</p>
              <p className="text-emerald-400">[10:14:35] Workload agentswarm-web replica healthy on port 3000</p>
            </div>
          </div>
        )}

        {/* Sovereign Kill Switch Tab */}
        {activeTab === 'killswitch' && (
          <div className="space-y-4">
            <div className="p-4 rounded-2xl bg-rose-950/30 border border-rose-500/30 space-y-2">
              <div className="flex items-center gap-2 text-rose-400 font-bold text-sm">
                <AlertTriangle className="w-4 h-4" />
                <span>Sovereign Owner Authority (Rule #5 of Master Blueprint)</span>
              </div>
              <p className="text-xs text-slate-300">
                A real decentralized platform gives the hardware owner final local authority.
                Engaging this action cordons the node, evicts third-party marketplace workloads, closes all public ingress,
                and stops external DePIN activity.
              </p>
              <p className="text-[11px] text-amber-300 font-mono">
                * Invariant: Will NOT destroy your own self-hosted workloads.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3 pt-2">
              <button
                onClick={() => cordonNode(node.id)}
                className="p-3.5 rounded-2xl bg-slate-900 border border-amber-500/40 hover:bg-amber-950/20 text-left transition-all group"
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-bold text-amber-400">
                    {node.cordoned ? 'Uncordon Node' : 'Cordon Node'}
                  </span>
                  <Sliders className="w-4 h-4 text-amber-400" />
                </div>
                <p className="text-[11px] text-slate-400">
                  {node.cordoned
                    ? 'Allow the scheduler to place new workloads again.'
                    : 'Refuse new workload placements while keeping existing active.'}
                </p>
              </button>

              <button
                onClick={() => drainNode(node.id)}
                className="p-3.5 rounded-2xl bg-slate-900 border border-blue-500/40 hover:bg-blue-950/20 text-left transition-all group"
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-bold text-cyan-400">Drain Workloads</span>
                  <Activity className="w-4 h-4 text-cyan-400" />
                </div>
                <p className="text-[11px] text-slate-400">
                  Gracefully evict contributed/marketplace workloads and signal peer reconcilers.
                </p>
              </button>
            </div>

            <button
              onClick={() => killSwitchNode(node.id)}
              className="w-full py-3.5 px-4 rounded-2xl bg-rose-600/20 hover:bg-rose-600/30 border border-rose-500 text-rose-300 hover:text-white font-bold text-xs uppercase tracking-wider transition-all flex items-center justify-center gap-2 shadow-lg shadow-rose-900/30"
            >
              <ShieldAlert className="w-5 h-5 text-rose-400" />
              <span>ENGAGE SOVEREIGN KILL SWITCH (ISOLATE MACHINE)</span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
