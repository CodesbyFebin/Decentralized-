import React, { useState } from 'react';
import {
  AppWindow,
  Rocket,
  Plus,
  ExternalLink,
  RotateCw,
  Search,
  Filter,
  CheckCircle2,
  AlertTriangle,
  Terminal,
  Shield,
  Layers,
  Globe2
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const WebsitesApps: React.FC = () => {
  const {
    deployments,
    setIsNewDeploymentModalOpen,
    setSelectedDeploymentId,
    restartDeployment,
  } = useNetwork();

  const [searchQuery, setSearchQuery] = useState('');
  const [filterEnv, setFilterEnv] = useState<'all' | 'production' | 'preview'>('all');

  const filtered = deployments.filter((d) => {
    const matches = d.name.toLowerCase().includes(searchQuery.toLowerCase()) || d.domain.toLowerCase().includes(searchQuery.toLowerCase());
    if (!matches) return false;
    if (filterEnv === 'production') return d.environment === 'Production';
    if (filterEnv === 'preview') return d.environment === 'Preview';
    return true;
  });

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <AppWindow className="w-4 h-4 text-cyan-400" />
            <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
              Websites & Applications
            </span>
          </div>
          <h1 className="text-2xl font-bold text-white tracking-tight">
            Sovereign Workload Fleet
          </h1>
          <p className="text-xs text-slate-300 mt-1">
            Declarative deployment contracts, live traffic telemetry, and automatic failure reconciliation.
          </p>
        </div>

        <button
          onClick={() => setIsNewDeploymentModalOpen(true)}
          className="px-4 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/20 transition-all flex items-center gap-1.5"
        >
          <Plus className="w-4 h-4" />
          <span>Deploy New App</span>
        </button>
      </div>

      {/* Grid of Workloads */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filtered.map((app) => (
          <div
            key={app.id}
            onClick={() => setSelectedDeploymentId(app.id)}
            className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 hover:border-cyan-400/50 shadow-xl backdrop-blur-xl transition-all cursor-pointer group flex flex-col justify-between"
          >
            <div>
              <div className="flex items-center justify-between pb-3 border-b border-blue-500/15 mb-3">
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-xl bg-cyan-500/15 border border-cyan-400/30 flex items-center justify-center text-cyan-300">
                    <Rocket className="w-4 h-4" />
                  </div>
                  <div>
                    <h3 className="text-sm font-bold text-white font-mono group-hover:text-cyan-300">
                      {app.name}
                    </h3>
                    <span className="text-[10px] text-slate-400 font-mono">{app.shortId}</span>
                  </div>
                </div>

                <span
                  className={`px-2 py-0.5 rounded-full text-[10px] font-mono border ${
                    app.status === 'Running'
                      ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                      : app.status === 'Degraded'
                      ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                      : 'bg-blue-500/15 text-cyan-400 border-blue-500/30'
                  }`}
                >
                  ● {app.status}
                </span>
              </div>

              <div className="space-y-1.5 text-xs font-mono text-slate-300 mb-4">
                <div className="flex justify-between">
                  <span className="text-slate-400">Environment:</span>
                  <span className="text-purple-300">{app.environment}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Replicas:</span>
                  <span className="text-white">{app.replicas.observed} / {app.replicas.desired}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Traffic (30d):</span>
                  <span className="text-cyan-300">{app.traffic30d} requests</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Source:</span>
                  <span className="text-slate-300">{app.source.type} ({app.source.commitSha})</span>
                </div>
              </div>
            </div>

            <div className="pt-3 border-t border-slate-800 flex items-center justify-between text-xs">
              <a
                href={app.domain}
                target="_blank"
                rel="noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="text-cyan-400 hover:underline flex items-center gap-1 font-mono text-[11px]"
              >
                <span>{app.domain}</span>
                <ExternalLink className="w-3 h-3" />
              </a>

              <button
                onClick={(e) => {
                  e.stopPropagation();
                  restartDeployment(app.id);
                }}
                className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
                title="Rolling restart"
              >
                <RotateCw className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
