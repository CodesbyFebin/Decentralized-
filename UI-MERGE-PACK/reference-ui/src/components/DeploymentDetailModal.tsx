import React, { useState } from 'react';
import {
  X,
  Rocket,
  Shield,
  Layers,
  FileCode,
  Activity,
  RotateCw,
  ExternalLink,
  CheckCircle2,
  Lock,
  Globe2,
  Terminal,
  Sliders
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const DeploymentDetailModal: React.FC = () => {
  const {
    deployments,
    selectedDeploymentId,
    setSelectedDeploymentId,
    scaleDeployment,
    reconcileDeployment,
    restartDeployment,
  } = useNetwork();

  const [activeTab, setActiveTab] = useState<'overview' | 'yaml' | 'logs' | 'sbom'>('overview');
  const dep = deployments.find((d) => d.id === selectedDeploymentId);

  if (!dep) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-3xl max-h-[90vh] overflow-y-auto rounded-3xl bg-gradient-to-b from-[#0d1838] to-[#070d1e] border border-blue-500/30 p-6 shadow-2xl custom-scrollbar text-slate-200">
        <button
          onClick={() => setSelectedDeploymentId(null)}
          className="absolute top-5 right-5 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-slate-700/50 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Header */}
        <div className="flex items-center gap-3.5 mb-6 pb-4 border-b border-blue-500/20">
          <div className="w-12 h-12 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 flex items-center justify-center text-cyan-300 shadow-lg shadow-cyan-500/20">
            <Rocket className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold text-white tracking-tight font-mono">{dep.name}</h2>
              <span
                className={`px-2 py-0.5 rounded-full text-[10px] font-mono border ${
                  dep.status === 'Running'
                    ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                    : dep.status === 'Degraded'
                    ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                    : 'bg-blue-500/15 text-cyan-400 border-blue-500/30'
                }`}
              >
                ● {dep.status}
              </span>
              <span className="px-2 py-0.5 rounded-full text-[10px] font-mono bg-purple-500/15 text-purple-300 border border-purple-500/30">
                {dep.environment}
              </span>
            </div>
            <div className="flex items-center gap-3 text-xs text-slate-400 font-mono mt-0.5">
              <a
                href={dep.domain}
                target="_blank"
                rel="noreferrer"
                className="text-cyan-400 hover:underline flex items-center gap-1"
              >
                <span>{dep.domain}</span>
                <ExternalLink className="w-3 h-3" />
              </a>
              <span>•</span>
              <span>Replicas: {dep.replicas.observed} / {dep.replicas.desired}</span>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-2 mb-5 border-b border-slate-800 pb-2">
          {[
            { id: 'overview', label: 'Topology & Placement' },
            { id: 'yaml', label: 'Portable Contract (YAML)' },
            { id: 'sbom', label: 'Supply Chain & SBOM' },
            { id: 'logs', label: 'Container Logs' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition-all ${
                activeTab === tab.id
                  ? 'bg-blue-600/20 text-cyan-300 border border-cyan-500/40'
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
            {/* Quick Actions Bar */}
            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-blue-500/20 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold text-slate-300">Scale Replicas:</span>
                <div className="flex items-center gap-1">
                  {[1, 2, 3, 4, 5].map((count) => (
                    <button
                      key={count}
                      onClick={() => scaleDeployment(dep.id, count)}
                      className={`w-7 h-7 rounded-lg text-xs font-mono font-bold transition-all ${
                        dep.replicas.desired === count
                          ? 'bg-cyan-500 text-black shadow-md shadow-cyan-500/30'
                          : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
                      }`}
                    >
                      {count}
                    </button>
                  ))}
                </div>
              </div>

              <div className="flex items-center gap-2">
                {dep.status === 'Degraded' && (
                  <button
                    onClick={() => reconcileDeployment(dep.id)}
                    className="px-3 py-1.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-amber-600 to-yellow-600 hover:from-amber-500 hover:to-yellow-500 shadow-md transition-all flex items-center gap-1.5"
                  >
                    <span>Reconcile Replicas</span>
                  </button>
                )}
                <button
                  onClick={() => restartDeployment(dep.id)}
                  className="px-3 py-1.5 rounded-xl text-xs font-semibold text-slate-200 bg-slate-800 hover:bg-slate-700 border border-slate-700 transition-all flex items-center gap-1.5"
                >
                  <RotateCw className="w-3.5 h-3.5" />
                  <span>Rolling Restart</span>
                </button>
              </div>
            </div>

            {/* Failure Domains verification */}
            <div className="p-4 rounded-2xl bg-blue-950/25 border border-blue-500/20 space-y-3">
              <span className="text-xs font-bold text-cyan-300 uppercase tracking-wider block">
                Failure Domain Verification (Rule #9: 3 replicas ≠ decentralization)
              </span>
              <p className="text-xs text-slate-300">
                Placement engine strictly scores and isolates pods across distinct operators and network ASNs.
              </p>
              <div className="grid grid-cols-3 gap-2.5 pt-1 text-xs font-mono">
                <div className="p-2.5 rounded-xl bg-slate-900/80 border border-emerald-500/30">
                  <span className="text-slate-400 block text-[10px]">Domain 1: Node Machine</span>
                  <span className="text-emerald-400 font-bold">✓ 3 Independent Nodes</span>
                </div>
                <div className="p-2.5 rounded-xl bg-slate-900/80 border border-emerald-500/30">
                  <span className="text-slate-400 block text-[10px]">Domain 2: Operator</span>
                  <span className="text-emerald-400 font-bold">✓ Febin + Community</span>
                </div>
                <div className="p-2.5 rounded-xl bg-slate-900/80 border border-emerald-500/30">
                  <span className="text-slate-400 block text-[10px]">Domain 3: Network / ASN</span>
                  <span className="text-emerald-400 font-bold">✓ AS55836 + AS24940</span>
                </div>
              </div>
            </div>

            {/* Secrets & Ingress */}
            <div className="grid grid-cols-2 gap-3 text-xs font-mono">
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-1">
                <span className="text-slate-400 text-[10px] uppercase block">Bound Sealed Secrets</span>
                {dep.secretsBound.map((sec) => (
                  <p key={sec} className="text-purple-300 flex items-center gap-1.5">
                    <Lock className="w-3 h-3 text-purple-400" />
                    <span>{sec} (AES-256-GCM)</span>
                  </p>
                ))}
              </div>
              <div className="p-3 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-1">
                <span className="text-slate-400 text-[10px] uppercase block">Anycast Ingress Routing</span>
                <p className="text-cyan-300">Edge WireGuard Overlay: Port 80, 443</p>
                <p className="text-slate-400">ECC TLS Termination: Automated</p>
              </div>
            </div>
          </div>
        )}

        {/* YAML Tab */}
        {activeTab === 'yaml' && (
          <div className="space-y-2">
            <span className="text-xs font-mono text-slate-400 block">
              Portable Deployment Specification (decentralized.host/v1)
            </span>
            <pre className="p-4 rounded-2xl bg-black/80 border border-blue-500/30 text-xs font-mono text-cyan-300 overflow-x-auto custom-scrollbar">
              {dep.yamlConfig}
            </pre>
          </div>
        )}

        {/* SBOM Tab */}
        {activeTab === 'sbom' && (
          <div className="space-y-3 font-mono text-xs">
            <div className="p-3.5 rounded-2xl bg-emerald-950/20 border border-emerald-500/30 flex items-center justify-between">
              <div className="flex items-center gap-2 text-emerald-400">
                <CheckCircle2 className="w-5 h-5" />
                <span className="font-bold">Cryptographically Attested Build Artifact</span>
              </div>
              <span className="text-[10px] text-emerald-300 bg-emerald-500/20 px-2 py-0.5 rounded">
                Verified
              </span>
            </div>
            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-2 text-slate-300">
              <p><span className="text-slate-500">Digest:</span> {dep.sbom.digest}</p>
              <p><span className="text-slate-500">Packages Audited:</span> {dep.sbom.packagesCount} (Zero vulnerabilities)</p>
              <p><span className="text-slate-500">Commit SHA:</span> {dep.source.commitSha}</p>
              <p><span className="text-slate-500">Builder Node:</span> homelab-kochi (Node 01)</p>
            </div>
          </div>
        )}

        {/* Logs Tab */}
        {activeTab === 'logs' && (
          <div className="p-3.5 rounded-2xl bg-black/80 border border-slate-800 font-mono text-[11px] text-slate-300 max-h-72 overflow-y-auto custom-scrollbar space-y-1">
            <p className="text-slate-500">[10:14:02] Container replica #1 running on homelab-kochi</p>
            <p className="text-cyan-400">[10:14:03] Container replica #2 running on vps-eu</p>
            <p className="text-emerald-400">[10:14:05] HTTP GET / 200 OK (1.2ms)</p>
            <p className="text-emerald-400">[10:14:20] Health probe passed on all 3 replicas</p>
          </div>
        )}
      </div>
    </div>
  );
};
