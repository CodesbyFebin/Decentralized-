import React, { useState, useEffect } from 'react';
import {
  Search,
  Command,
  Server,
  AppWindow,
  HardDrive,
  Rocket,
  Shield,
  Bot,
  ArrowRight,
  Boxes,
  Lock,
  Globe2,
  X
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const CommandPalette: React.FC = () => {
  const {
    isCommandPaletteOpen,
    setIsCommandPaletteOpen,
    setCurrentTab,
    setIsAddNodeModalOpen,
    setIsNewDeploymentModalOpen,
    setIsAddStorageModalOpen,
    nodes,
    deployments,
  } = useNetwork();

  const [query, setQuery] = useState('');

  if (!isCommandPaletteOpen) return null;

  const quickActions = [
    {
      id: 'act-deploy',
      title: 'New Deployment',
      subtitle: 'Deploy from GitHub, Docker, or Template',
      icon: Rocket,
      action: () => {
        setIsCommandPaletteOpen(false);
        setIsNewDeploymentModalOpen(true);
      },
    },
    {
      id: 'act-add-node',
      title: 'Add a Node',
      subtitle: 'Enroll Linux machine, Home Server, or VPS',
      icon: Server,
      action: () => {
        setIsCommandPaletteOpen(false);
        setIsAddNodeModalOpen(true);
      },
    },
    {
      id: 'act-add-storage',
      title: 'Add Storage Pool',
      subtitle: 'Mount Local NVMe, NAS ZFS, or DePIN Crust',
      icon: HardDrive,
      action: () => {
        setIsCommandPaletteOpen(false);
        setIsAddStorageModalOpen(true);
      },
    },
    {
      id: 'act-copilot',
      title: 'Ask RAG Copilot',
      subtitle: 'AI Infrastructure Diagnosis & Placement',
      icon: Bot,
      action: () => {
        setIsCommandPaletteOpen(false);
        setCurrentTab('rag-copilot');
      },
    },
    {
      id: 'act-depin',
      title: 'Explore DePIN Marketplace',
      subtitle: 'View provider capacity, leases & signed metering',
      icon: Boxes,
      action: () => {
        setIsCommandPaletteOpen(false);
        setCurrentTab('depin');
      },
    },
  ];

  const filteredApps = deployments.filter((d) =>
    d.name.toLowerCase().includes(query.toLowerCase()) || d.domain.toLowerCase().includes(query.toLowerCase())
  );

  const filteredNodes = nodes.filter((n) =>
    n.name.toLowerCase().includes(query.toLowerCase()) || n.location.country.toLowerCase().includes(query.toLowerCase())
  );

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 p-4 bg-black/80 backdrop-blur-md animate-in fade-in duration-150">
      <div className="relative w-full max-w-2xl rounded-2xl bg-[#091329] border border-blue-500/40 shadow-2xl overflow-hidden">
        {/* Search Input */}
        <div className="flex items-center gap-3 px-4 py-3.5 border-b border-blue-500/20 bg-slate-900/60">
          <Search className="w-5 h-5 text-cyan-400" />
          <input
            type="text"
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Type a command, search nodes, apps, domains, or ask AI..."
            className="flex-1 bg-transparent text-white placeholder-slate-400 text-sm focus:outline-none font-mono"
          />
          <button
            onClick={() => setIsCommandPaletteOpen(false)}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Results Container */}
        <div className="max-h-[60vh] overflow-y-auto p-3 space-y-4 custom-scrollbar text-xs">
          {/* Quick Actions */}
          <div>
            <span className="text-[10px] font-bold text-cyan-400/80 uppercase tracking-wider px-2 block mb-1">
              Quick Actions
            </span>
            <div className="space-y-1">
              {quickActions.map((act) => {
                const Icon = act.icon;
                return (
                  <button
                    key={act.id}
                    onClick={act.action}
                    className="w-full flex items-center justify-between p-2.5 rounded-xl hover:bg-slate-800/80 text-left text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-1.5 rounded-lg bg-blue-500/10 text-cyan-400 group-hover:bg-cyan-500/20">
                        <Icon className="w-4 h-4" />
                      </div>
                      <div>
                        <p className="font-semibold">{act.title}</p>
                        <p className="text-[11px] text-slate-400">{act.subtitle}</p>
                      </div>
                    </div>
                    <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-cyan-400 transition-colors" />
                  </button>
                );
              })}
            </div>
          </div>

          {/* Applications */}
          {filteredApps.length > 0 && (
            <div>
              <span className="text-[10px] font-bold text-slate-400 uppercase tracking-wider px-2 block mb-1">
                Applications ({filteredApps.length})
              </span>
              <div className="space-y-1">
                {filteredApps.slice(0, 4).map((app) => (
                  <button
                    key={app.id}
                    onClick={() => {
                      setIsCommandPaletteOpen(false);
                      setCurrentTab('websites');
                    }}
                    className="w-full flex items-center justify-between p-2 rounded-xl hover:bg-slate-800/80 text-left text-slate-300 group"
                  >
                    <div className="flex items-center gap-2.5">
                      <AppWindow className="w-4 h-4 text-indigo-400" />
                      <span className="font-mono text-white">{app.name}</span>
                      <span className="text-[11px] text-slate-500 font-mono">{app.domain}</span>
                    </div>
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400">
                      {app.status}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Nodes */}
          {filteredNodes.length > 0 && (
            <div>
              <span className="text-[10px] font-bold text-slate-400 uppercase tracking-wider px-2 block mb-1">
                Nodes ({filteredNodes.length})
              </span>
              <div className="space-y-1">
                {filteredNodes.slice(0, 4).map((node) => (
                  <button
                    key={node.id}
                    onClick={() => {
                      setIsCommandPaletteOpen(false);
                      setCurrentTab('nodes');
                    }}
                    className="w-full flex items-center justify-between p-2 rounded-xl hover:bg-slate-800/80 text-left text-slate-300 group"
                  >
                    <div className="flex items-center gap-2.5">
                      <Server className="w-4 h-4 text-cyan-400" />
                      <span className="font-mono text-white">{node.name}</span>
                      <span className="text-[11px] text-slate-500">{node.location.country}</span>
                    </div>
                    <span className={`text-[10px] font-mono px-2 py-0.5 rounded ${
                      node.status === 'Online' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-rose-500/15 text-rose-400'
                    }`}>
                      {node.status}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer shortcuts */}
        <div className="px-4 py-2 border-t border-blue-500/20 bg-slate-950/80 flex items-center justify-between text-[11px] text-slate-400">
          <span>Navigate with arrows • Press ESC to close</span>
          <span className="font-mono text-cyan-400">Decentralized.Host v1.4.2</span>
        </div>
      </div>
    </div>
  );
};
