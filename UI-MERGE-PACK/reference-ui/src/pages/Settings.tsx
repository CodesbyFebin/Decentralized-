import React, { useState } from 'react';
import { Settings as SettingsIcon, Shield, Download, Upload, Radio, Server, CheckCircle2, Lock } from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Settings: React.FC = () => {
  const { nodes, deployments, storageNodes, addToast } = useNetwork();
  const [federationUrl, setFederationUrl] = useState('dhp://federation.decentralized.host/v1');

  const handleExportBackup = () => {
    const data = {
      version: 'decentralized.host/v1',
      exportedAt: new Date().toISOString(),
      nodesCount: nodes.length,
      deploymentsCount: deployments.length,
      storagePoolsCount: storageNodes.length,
      state: { nodes, deployments, storageNodes },
    };
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `decentralized-host-cluster-backup-${Date.now()}.json`;
    a.click();
    addToast('Complete sovereign cluster state exported!', 'success');
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <SettingsIcon className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
            Sovereign Control Plane
          </span>
        </div>
        <h1 className="text-2xl font-bold text-white tracking-tight">
          Cluster Settings & Portability
        </h1>
        <p className="text-xs text-slate-300 mt-1">
          Zero vendor lock-in. Export complete state, configure replaceable coordinators, and manage federation.
        </p>
      </div>

      {/* Backup and Restore */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl space-y-3">
        <h3 className="text-sm font-bold text-white font-mono flex items-center gap-2">
          <Download className="w-4 h-4 text-cyan-400" />
          <span>Full Cluster State Export & Restore (Zero Lock-In)</span>
        </h3>
        <p className="text-xs text-slate-400">
          Download your complete cluster topology, portable YAML deployment manifests, and cryptographic node bindings.
        </p>
        <div className="pt-2 flex items-center gap-3">
          <button
            onClick={handleExportBackup}
            className="px-4 py-2.5 rounded-xl text-xs font-bold text-white bg-blue-600 hover:bg-blue-500 transition-all flex items-center gap-2 shadow-md"
          >
            <Download className="w-4 h-4" />
            <span>Export Snapshot (JSON)</span>
          </button>
        </div>
      </div>

      {/* Federation Settings (Rule #29) */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-xl space-y-3 font-mono text-xs">
        <h3 className="text-sm font-bold text-white font-mono flex items-center gap-2 font-sans">
          <Radio className="w-4 h-4 text-purple-400" />
          <span>Federation & Replaceable Coordinator (Rule #29)</span>
        </h3>
        <p className="text-slate-400 font-sans">
          A truly decentralized network never depends on one coordinator. Configure federated protocol peers.
        </p>
        <div className="pt-2">
          <label className="text-slate-400 block mb-1">Primary Federation Peer URI</label>
          <input
            type="text"
            value={federationUrl}
            onChange={(e) => setFederationUrl(e.target.value)}
            className="w-full max-w-xl p-2.5 rounded-xl bg-slate-900 border border-slate-800 text-cyan-300 focus:outline-none focus:border-cyan-400"
          />
        </div>
      </div>

      {/* Master Blueprint Constitution Summary */}
      <div className="p-5 rounded-3xl bg-blue-950/20 border border-blue-500/25 space-y-2 text-xs font-mono">
        <span className="text-xs font-bold text-cyan-400 uppercase tracking-wider block">
          Final Architectural Constitution
        </span>
        <ul className="space-y-1 text-slate-300 text-[11px] list-disc pl-4">
          <li>USER-OWNED HARDWARE IS THE FOUNDATION.</li>
          <li>NODE OWNER IS THE FINAL LOCAL AUTHORITY.</li>
          <li>SELF-HOSTING DOES NOT REQUIRE A TOKEN OR A WALLET.</li>
          <li>OWNER RESERVE ALWAYS WINS OVER MARKETPLACE DEMAND.</li>
          <li>3 REPLICAS ON 1 MACHINE ≠ DECENTRALIZATION.</li>
          <li>BLOCKCHAIN IS COORDINATION INFRASTRUCTURE, NOT HOSTING INFRASTRUCTURE.</li>
        </ul>
      </div>
    </div>
  );
};
