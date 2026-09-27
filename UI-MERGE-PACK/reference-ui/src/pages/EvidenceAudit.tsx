import React, { useState } from 'react';
import {
  Lock,
  FileCheck2,
  Shield,
  CheckCircle2,
  Copy,
  Check,
  Search,
  ExternalLink,
  Layers,
  Terminal,
  Activity
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const EvidenceAudit: React.FC = () => {
  const [copiedRoot, setCopiedRoot] = useState<string | null>(null);

  const evidenceRoots = [
    {
      id: 'root-01',
      digest: '0x7c9b81f9a2e0bc419d8213aa672901ce45bb9012',
      workload: 'agentswarm-web (web_01...3f2a)',
      node: 'homelab-kochi',
      raftHeight: 482910,
      timestamp: '12m ago',
      metrics: '1,240 CPU·s, 320 GB bandwidth',
      signatures: 'ed25519:node_01...a3f2 (Valid)',
      status: 'VERIFIED',
    },
    {
      id: 'root-02',
      digest: '0xfa3921bce092781a54cc9921ef00182ba71891ce',
      workload: 'llama-inference-worker (lease-9821)',
      node: 'gpu-workstation',
      raftHeight: 482894,
      timestamp: '1h ago',
      metrics: '64,800 CPU·s, 412 GPU·s, 210 GB bandwidth',
      signatures: 'ed25519:node_02...f9b1 (Valid)',
      status: 'VERIFIED',
    },
    {
      id: 'root-03',
      digest: '0x88bb7710adfe91823bc0192eab55219088cc11ee',
      workload: 'matrix-synapse-edge (lease-9822)',
      node: 'homelab-kochi',
      raftHeight: 482810,
      timestamp: '3h ago',
      metrics: '259,200 CPU·s, 420 GB bandwidth',
      signatures: 'ed25519:node_01...a3f2 (Valid)',
      status: 'VERIFIED',
    },
  ];

  const handleCopy = (digest: string) => {
    navigator.clipboard.writeText(digest);
    setCopiedRoot(digest);
    setTimeout(() => setCopiedRoot(null), 2500);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <Lock className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
            Evidence Network & Merkle Ledger (Rule #19)
          </span>
        </div>
        <h1 className="text-2xl font-bold text-white tracking-tight">
          Tamper-Evident Cryptographic Audit
        </h1>
        <p className="text-xs text-slate-300 mt-1 max-w-2xl leading-relaxed">
          Every deployment artifact, runtime execution window, and metering sample produces an immutable hash chain.
          Periodic Merkle roots are committed to Raft quorum with optional Web3 anchoring.
        </p>

        {/* Chain visualization */}
        <div className="mt-5 p-3.5 rounded-2xl bg-black/50 border border-blue-500/20 font-mono text-[11px] text-cyan-300 flex flex-wrap items-center gap-2">
          <span>SOURCE DIGEST</span>
          <span className="text-slate-500">→</span>
          <span>BUILD ATTESTATION</span>
          <span className="text-slate-500">→</span>
          <span>ARTIFACT DIGEST</span>
          <span className="text-slate-500">→</span>
          <span>LEASE COMMITMENT</span>
          <span className="text-slate-500">→</span>
          <span>SIGNED USAGE RECORDS</span>
          <span className="text-slate-500">→</span>
          <span className="text-emerald-400 font-bold">MERKLE ROOT</span>
        </div>
      </div>

      {/* Merkle Roots Table */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl space-y-4">
        <h3 className="text-sm font-bold text-white font-mono flex items-center justify-between">
          <span>Attested Merkle Roots (Raft Consensus Height #482,910)</span>
          <span className="text-xs text-emerald-400 font-normal">Quorum: 5/5 Verified</span>
        </h3>

        <div className="space-y-3 font-mono text-xs">
          {evidenceRoots.map((root) => (
            <div
              key={root.id}
              className="p-4 rounded-2xl bg-slate-900/80 border border-slate-800 hover:border-blue-500/40 transition-all space-y-2.5"
            >
              <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 pb-2 border-b border-slate-800">
                <div className="flex items-center gap-2">
                  <span className="text-cyan-300 font-bold">{root.digest}</span>
                  <button
                    onClick={() => handleCopy(root.digest)}
                    className="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-white"
                  >
                    {copiedRoot === root.digest ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
                <span className="px-2 py-0.5 rounded-full text-[10px] bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                  ✓ {root.status}
                </span>
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-[11px] text-slate-300">
                <div>
                  <span className="text-slate-500 block">Workload</span>
                  <span className="text-white">{root.workload}</span>
                </div>
                <div>
                  <span className="text-slate-500 block">Execution Node</span>
                  <span className="text-slate-200">{root.node}</span>
                </div>
                <div>
                  <span className="text-slate-500 block">Metered Volume</span>
                  <span className="text-purple-300">{root.metrics}</span>
                </div>
                <div>
                  <span className="text-slate-500 block">Node Signature</span>
                  <span className="text-emerald-400">{root.signatures}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
