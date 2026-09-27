import React, { useState } from 'react';
import {
  Boxes,
  Plus,
  Coins,
  Shield,
  Activity,
  CheckCircle2,
  FileCheck2,
  Lock,
  ArrowRight,
  TrendingUp,
  Cpu,
  Layers,
  Search,
  Scale
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const DePINMarketplace: React.FC = () => {
  const { offers, leases, createDePINOffer, nodes } = useNetwork();
  const [activeTab, setActiveTab] = useState<'offers' | 'leases' | 'metering' | 'disputes'>('offers');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedNodeId, setSelectedNodeId] = useState(nodes[0]?.id || 'node-01');
  const [cpuAmount, setCpuAmount] = useState(2);
  const [ramAmount, setRamAmount] = useState(4);
  const [price, setPrice] = useState(0.12);

  const handlePublishOffer = (e: React.FormEvent) => {
    e.preventDefault();
    const targetNode = nodes.find((n) => n.id === selectedNodeId);
    createDePINOffer({
      nodeId: selectedNodeId,
      nodeName: targetNode?.name || 'homelab-kochi',
      cpuOffered: cpuAmount,
      ramOfferedGb: ramAmount,
      priceHourly: price,
      terms: 'Strict gVisor sandbox. WireGuard ingress. 99.9% availability SLA.',
    });
    setShowCreateModal(false);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      {/* Header */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 p-6 shadow-2xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-semibold uppercase text-cyan-400 font-mono flex items-center gap-1.5">
            <Boxes className="w-3.5 h-3.5" />
            First-Party DePIN Marketplace (P2 Milestone)
          </span>
          <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-purple-500/20 text-purple-300 border border-purple-500/30">
            Settlement Neutral
          </span>
        </div>

        <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
          Sovereign Capacity Exchange
        </h1>
        <p className="text-xs sm:text-sm text-slate-300 max-w-2xl mt-1.5 leading-relaxed">
          Contribute spare hardware to the decentralized mesh or lease independent compute.
          All usage is signed by node agents and cryptographically reconciled. Zero blockchain token required.
        </p>

        <div className="flex flex-wrap items-center gap-3 mt-4">
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-blue-600/30 transition-all flex items-center gap-1.5"
          >
            <Plus className="w-4 h-4" />
            <span>Publish Capacity Offer</span>
          </button>
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-2 mt-5 pt-3 border-t border-blue-500/20 text-xs">
          {[
            { id: 'offers', label: `Active Capacity Offers (${offers.length})` },
            { id: 'leases', label: `Current Leases (${leases.length})` },
            { id: 'metering', label: 'Signed Metering Evidence' },
            { id: 'disputes', label: 'Disputes & SLA' },
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

      {/* Offers Tab */}
      {activeTab === 'offers' && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {offers.map((off) => (
            <div
              key={off.id}
              className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl backdrop-blur-xl space-y-3 font-mono"
            >
              <div className="flex items-center justify-between pb-2 border-b border-slate-800 text-xs">
                <span className="font-bold text-white font-sans">{off.nodeName}</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400">
                  ● {off.status}
                </span>
              </div>

              <div className="space-y-1.5 text-xs text-slate-300">
                <div className="flex justify-between">
                  <span className="text-slate-400">Provider:</span>
                  <span>{off.provider}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">CPU Offered:</span>
                  <span className="text-cyan-300 font-bold">{off.cpuOffered} Cores</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">RAM Offered:</span>
                  <span className="text-purple-300 font-bold">{off.ramOfferedGb} GB</span>
                </div>
                {off.gpuOffered && (
                  <div className="flex justify-between">
                    <span className="text-slate-400">GPU:</span>
                    <span className="text-amber-300">{off.gpuOffered}</span>
                  </div>
                )}
                <div className="flex justify-between">
                  <span className="text-slate-400">Reputation:</span>
                  <span className="text-emerald-400">{off.reputationScore}%</span>
                </div>
                <div className="flex justify-between pt-2 border-t border-slate-800">
                  <span className="text-slate-400">Price:</span>
                  <span className="text-white font-bold">${off.priceHourly}/hr</span>
                </div>
              </div>

              <p className="text-[11px] text-slate-400 font-sans italic">{off.terms}</p>
            </div>
          ))}
        </div>
      )}

      {/* Leases Tab */}
      {activeTab === 'leases' && (
        <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
          <div className="overflow-x-auto custom-scrollbar">
            <table className="w-full text-left text-xs font-mono">
              <thead>
                <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                  <th className="py-3 px-3">Lease ID</th>
                  <th className="py-3 px-2">Workload</th>
                  <th className="py-3 px-2">Node</th>
                  <th className="py-3 px-2">Consumer</th>
                  <th className="py-3 px-2">Status</th>
                  <th className="py-3 px-2">Resources</th>
                  <th className="py-3 px-2">Metered Usage</th>
                  <th className="py-3 px-2">Evidence Root</th>
                  <th className="py-3 px-2">Settlement</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {leases.map((ls) => (
                  <tr key={ls.id} className="hover:bg-slate-800/40 transition-colors">
                    <td className="py-3.5 px-3 text-cyan-400 font-bold">{ls.id}</td>
                    <td className="py-3.5 px-2 text-white">{ls.workloadName}</td>
                    <td className="py-3.5 px-2 text-slate-300">{ls.nodeName}</td>
                    <td className="py-3.5 px-2 text-slate-400">{ls.consumer}</td>
                    <td className="py-3.5 px-2">
                      <span className="px-2 py-0.5 rounded-full text-[10px] bg-emerald-500/15 text-emerald-400">
                        ● {ls.status}
                      </span>
                    </td>
                    <td className="py-3.5 px-2 text-slate-300">
                      {ls.cpuAllocated} CPU / {ls.ramAllocatedGb} GB
                    </td>
                    <td className="py-3.5 px-2 text-slate-300">
                      {ls.meteredCpuSec} CPU·s
                    </td>
                    <td className="py-3.5 px-2 text-purple-300">{ls.evidenceRoot}</td>
                    <td className="py-3.5 px-2 text-amber-300">{ls.settlementMethod}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Metering Tab */}
      {activeTab === 'metering' && (
        <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl space-y-4">
          <h3 className="text-sm font-bold text-white font-mono">
            Cryptographic Usage Evidence Chain (Rule #19)
          </h3>
          <p className="text-xs text-slate-400 font-sans">
            Every CPU sample, memory byte-second, and network packet is signed by dh-agent on host hardware.
          </p>
          <div className="p-4 rounded-2xl bg-black/70 border border-slate-800 font-mono text-xs text-slate-300 space-y-2">
            <p className="text-cyan-400">ROOT_KEK → SESSION_DEK → ED25519_NODE_KEY</p>
            <p className="text-emerald-400">0x7c9b81f9a... Merkle Root: 1,420,800 CPU-seconds committed</p>
            <p className="text-purple-400">Consensus Height: Raft Index #482,910 (Verified by 5/5 peers)</p>
          </div>
        </div>
      )}

      {/* Create Offer Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md">
          <form onSubmit={handlePublishOffer} className="w-full max-w-md p-6 rounded-3xl bg-slate-900 border border-blue-500/30 text-white space-y-4 text-xs font-mono">
            <h3 className="text-base font-bold font-sans">Publish Capacity Offer</h3>

            <div>
              <label className="block text-slate-400 mb-1">Select Contributing Node</label>
              <select
                value={selectedNodeId}
                onChange={(e) => setSelectedNodeId(e.target.value)}
                className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700 text-white"
              >
                {nodes.map((n) => (
                  <option key={n.id} value={n.id}>{n.name} ({n.cpu.available} CPU avail)</option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-slate-400 mb-1">CPU Cores</label>
                <input
                  type="number"
                  min="1"
                  max="16"
                  value={cpuAmount}
                  onChange={(e) => setCpuAmount(Number(e.target.value))}
                  className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700"
                />
              </div>
              <div>
                <label className="block text-slate-400 mb-1">RAM (GB)</label>
                <input
                  type="number"
                  min="1"
                  max="64"
                  value={ramAmount}
                  onChange={(e) => setRamAmount(Number(e.target.value))}
                  className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700"
                />
              </div>
            </div>

            <div>
              <label className="block text-slate-400 mb-1">Hourly Price ($USD)</label>
              <input
                type="number"
                step="0.01"
                value={price}
                onChange={(e) => setPrice(Number(e.target.value))}
                className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700"
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 font-sans"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-4 py-2 rounded-xl bg-cyan-600 hover:bg-cyan-500 font-bold font-sans text-white"
              >
                Publish Offer
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
