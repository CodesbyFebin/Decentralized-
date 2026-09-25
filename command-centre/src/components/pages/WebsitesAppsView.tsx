import React, { useEffect, useState } from 'react';
import {
  Globe,
  Rocket,
  Search,
  Filter,
  RefreshCw,
  Plus,
  ExternalLink,
  ShieldCheck,
  Server,
  Activity,
  Terminal,
  Lock,
  Trash2,
  X,
  FileCheck2
} from 'lucide-react';
import { Application } from '../../types/platform';
import { api } from '../../lib/api';
import { DesiredObservedEvidenceBadge } from '../common/DesiredObservedEvidenceBadge';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
  selectedAppId?: string;
}

export const WebsitesAppsView: React.FC<Props> = ({ onNavigate, selectedAppId }) => {
  const [apps, setApps] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeFilter, setActiveFilter] = useState('All');
  const [inspectApp, setInspectApp] = useState<Application | null>(null);
  const [inspectTab, setInspectTab] = useState<'overview' | 'logs' | 'env' | 'evidence'>('overview');

  const loadApps = async () => {
    try {
      setLoading(true);
      const res = await api.getApps();
      setApps(res.data);
      if (selectedAppId) {
        const found = res.data.find((a) => a.id === selectedAppId);
        if (found) setInspectApp(found);
      }
    } catch (err) {
      console.error('Failed to load apps:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadApps();
  }, [selectedAppId]);

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Are you sure you want to remove ${name} from the decentralized mesh?`)) return;
    try {
      await api.deleteApp(id);
      setApps(apps.filter((a) => a.id !== id));
      if (inspectApp?.id === id) setInspectApp(null);
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  };

  const filteredApps = apps.filter((app) => {
    const matchesSearch =
      app.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      app.domain.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesFilter = activeFilter === 'All' || app.type === activeFilter;
    return matchesSearch && matchesFilter;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Websites & Applications</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">
            Applications & Workloads
          </h1>
          <p className="text-xs text-slate-400">
            Inspect observed runtime replicas, anycast routing, and cryptographic release evidence.
          </p>
        </div>

        <button
          onClick={() => onNavigate('deploy')}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 transition-all"
        >
          <Plus className="w-4 h-4" />
          <span>Deploy New App</span>
        </button>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 p-3 rounded-2xl bg-[#0D1527] border border-slate-800">
        <div className="flex items-center gap-2 flex-1 max-w-md px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
          <Search className="w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search applications or domains..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="bg-transparent flex-1 text-white focus:outline-none placeholder-slate-500"
          />
        </div>

        <div className="flex items-center gap-1.5 overflow-x-auto">
          {['All', 'Web App', 'Static Site', 'Docker App'].map((filter) => (
            <button
              key={filter}
              onClick={() => setActiveFilter(filter)}
              className={`px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
                activeFilter === filter
                  ? 'bg-blue-600 text-white shadow-md'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
              }`}
            >
              {filter}
            </button>
          ))}
        </div>
      </div>

      {/* Apps Table */}
      <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 overflow-hidden shadow-xl">
        <table className="w-full text-left border-collapse text-xs">
          <thead>
            <tr className="border-b border-slate-800 bg-slate-950/60 text-slate-400 font-mono text-[11px] uppercase">
              <th className="py-3.5 px-4 font-semibold">Application & Domain</th>
              <th className="py-3.5 px-4 font-semibold">Type</th>
              <th className="py-3.5 px-4 font-semibold">Desired vs Observed</th>
              <th className="py-3.5 px-4 font-semibold">Placement Nodes</th>
              <th className="py-3.5 px-4 font-semibold">Visitors (30d)</th>
              <th className="py-3.5 px-4 font-semibold text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60">
            {filteredApps.map((app) => (
              <tr
                key={app.id}
                className="hover:bg-slate-800/30 transition-colors group cursor-pointer"
                onClick={() => setInspectApp(app)}
              >
                {/* Application Name & Domain */}
                <td className="py-4 px-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-cyan-400 font-mono font-bold">
                      <Globe className="w-4 h-4" />
                    </div>
                    <div>
                      <div className="font-bold text-white flex items-center gap-1.5">
                        <span>{app.name}</span>
                        <span className="text-[10px] font-mono text-slate-500 font-normal">({app.version})</span>
                      </div>
                      <div className="text-[11px] text-cyan-400 font-mono flex items-center gap-1 hover:underline">
                        <span>{app.domain}</span>
                        <ExternalLink className="w-3 h-3 text-slate-500" />
                      </div>
                    </div>
                  </div>
                </td>

                {/* Type */}
                <td className="py-4 px-4">
                  <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-slate-800 border border-slate-700 text-slate-300">
                    {app.type}
                  </span>
                </td>

                {/* Desired vs Observed Badge */}
                <td className="py-4 px-4">
                  <DesiredObservedEvidenceBadge
                    desired={`${app.desiredReplicas} replicas`}
                    observed={`${app.observedHealthyReplicas} healthy`}
                    evidenceDigest={app.evidence?.sha256Digest}
                    isHealthy={app.observedHealthyReplicas >= app.desiredReplicas}
                  />
                </td>

                {/* Placement Nodes */}
                <td className="py-4 px-4 font-mono text-slate-400">
                  <div className="flex items-center gap-1">
                    <Server className="w-3.5 h-3.5 text-slate-500" />
                    <span>{app.nodesAssigned.length} hosts</span>
                    <span className="text-slate-600">({app.regionsAssigned.join(', ')})</span>
                  </div>
                </td>

                {/* Visitors */}
                <td className="py-4 px-4 font-mono text-white font-semibold">
                  {(app.visitors30d / 1000).toFixed(1)}K
                </td>

                {/* Actions */}
                <td className="py-4 px-4 text-right">
                  <div className="flex items-center justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => setInspectApp(app)}
                      className="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium"
                    >
                      Inspect
                    </button>
                    <button
                      onClick={() => handleDelete(app.id, app.name)}
                      className="p-1 rounded-lg hover:bg-rose-950/40 text-slate-500 hover:text-rose-400"
                      title="Delete Application"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Inspect App Drawer / Modal */}
      {inspectApp && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex justify-end">
          <div className="w-full max-w-2xl bg-[#0B1120] border-l border-slate-800 h-full overflow-y-auto flex flex-col justify-between p-6 animate-in slide-in-from-right duration-200">
            <div>
              {/* Drawer Header */}
              <div className="flex items-start justify-between pb-4 border-b border-slate-800">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-blue-600/20 border border-blue-500/40 flex items-center justify-center text-cyan-400">
                    <Globe className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-lg font-bold text-white flex items-center gap-2">
                      {inspectApp.name}
                      <span className="text-xs px-2 py-0.5 rounded bg-emerald-950/60 text-emerald-400 border border-emerald-500/40">
                        {inspectApp.status}
                      </span>
                    </h2>
                    <span className="text-xs font-mono text-cyan-400">{inspectApp.domain}</span>
                  </div>
                </div>

                <button
                  onClick={() => setInspectApp(null)}
                  className="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Drawer Tabs */}
              <div className="flex items-center gap-2 mt-4 border-b border-slate-800 pb-2">
                {[
                  { id: 'overview', label: 'Overview' },
                  { id: 'logs', label: 'Runtime Logs' },
                  { id: 'env', label: 'Environment' },
                  { id: 'evidence', label: 'Cryptographic Evidence' }
                ].map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setInspectTab(tab.id as any)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      inspectTab === tab.id
                        ? 'bg-blue-600 text-white'
                        : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>

              {/* Tab Contents */}
              <div className="mt-5 space-y-4">
                {inspectTab === 'overview' && (
                  <div className="space-y-4">
                    <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-3">
                      <div className="text-xs font-bold text-white">Desired vs Observed Quorum</div>
                      <DesiredObservedEvidenceBadge
                        desired={`${inspectApp.desiredReplicas} replicas requested`}
                        observed={`${inspectApp.observedHealthyReplicas} verified active`}
                        evidenceDigest={inspectApp.evidence?.sha256Digest}
                        isHealthy={inspectApp.observedHealthyReplicas >= inspectApp.desiredReplicas}
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-3 text-xs font-mono">
                      <div className="p-3 rounded-xl bg-slate-900 border border-slate-800">
                        <span className="text-slate-500">Latency (P95)</span>
                        <div className="text-white font-bold text-sm mt-0.5">{inspectApp.avgResponseMs} ms</div>
                      </div>
                      <div className="p-3 rounded-xl bg-slate-900 border border-slate-800">
                        <span className="text-slate-500">30d Bandwidth</span>
                        <div className="text-white font-bold text-sm mt-0.5">{inspectApp.bandwidthUsedGb} GB</div>
                      </div>
                    </div>

                    <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-2">
                      <span className="text-xs font-bold text-white">Assigned Host Nodes</span>
                      <div className="space-y-1.5">
                        {inspectApp.nodesAssigned.map((node) => (
                          <div
                            key={node}
                            className="flex items-center justify-between p-2 rounded-lg bg-slate-950/70 border border-slate-800/60 text-xs font-mono"
                          >
                            <span className="text-cyan-400 font-semibold">{node}</span>
                            <span className="text-emerald-400 flex items-center gap-1">
                              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                              Healthy
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}

                {inspectTab === 'logs' && (
                  <div className="rounded-xl bg-black border border-slate-800 p-4 font-mono text-xs text-emerald-400 space-y-1 max-h-96 overflow-y-auto">
                    {inspectApp.logs.map((log, idx) => (
                      <div key={idx} className="leading-relaxed">
                        {log}
                      </div>
                    ))}
                    <div className="text-slate-600 pt-2 text-[10px]">-- End of log stream --</div>
                  </div>
                )}

                {inspectTab === 'env' && (
                  <div className="space-y-3">
                    <div className="text-xs text-slate-400">
                      Secret values are write-only and encrypted on host keyrings.
                    </div>
                    {inspectApp.envVars.length === 0 ? (
                      <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 text-xs text-slate-500 text-center">
                        No environment variables defined for this application.
                      </div>
                    ) : (
                      inspectApp.envVars.map((env) => (
                        <div
                          key={env.key}
                          className="flex items-center justify-between p-3 rounded-xl bg-slate-900 border border-slate-800 text-xs font-mono"
                        >
                          <span className="text-white font-semibold">{env.key}</span>
                          <span className="text-slate-400">{env.maskedValue}</span>
                        </div>
                      ))
                    )}
                  </div>
                )}

                {inspectTab === 'evidence' && (
                  <div className="space-y-4">
                    <div className="p-4 rounded-xl bg-slate-900/90 border border-emerald-500/30 space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-white flex items-center gap-2">
                          <FileCheck2 className="w-4 h-4 text-emerald-400" />
                          Sealed Evidence Record
                        </span>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-500/40">
                          {inspectApp.evidence.status}
                        </span>
                      </div>

                      <div className="space-y-1.5 text-xs font-mono">
                        <div>
                          <span className="text-slate-500">Record ID: </span>
                          <span className="text-white">{inspectApp.evidence.recordId}</span>
                        </div>
                        <div>
                          <span className="text-slate-500">SHA-256 Digest: </span>
                          <span className="text-cyan-400 break-all">{inspectApp.evidence.sha256Digest}</span>
                        </div>
                        <div>
                          <span className="text-slate-500">Signer Fingerprint: </span>
                          <span className="text-purple-400">{inspectApp.evidence.signerFingerprint}</span>
                        </div>
                      </div>

                      <div className="pt-2 border-t border-slate-800">
                        <span className="text-[11px] font-bold text-slate-400">Passed Verification Gates:</span>
                        <div className="flex flex-wrap gap-1.5 mt-1.5">
                          {inspectApp.evidence.gatesPassed.map((gate) => (
                            <span
                              key={gate}
                              className="px-2 py-0.5 rounded bg-slate-950 text-emerald-400 border border-emerald-500/30 text-[10px] font-mono"
                            >
                              ✓ {gate}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="pt-4 border-t border-slate-800 flex justify-end">
              <button
                onClick={() => setInspectApp(null)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
