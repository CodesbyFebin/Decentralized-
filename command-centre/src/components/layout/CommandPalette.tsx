import React, { useState, useEffect } from 'react';
import {
  Search,
  Globe,
  Server,
  HardDrive,
  AtSign,
  ShieldCheck,
  Sparkles,
  Rocket,
  PlusCircle,
  FileCheck2,
  X
} from 'lucide-react';
import { NavRoute } from './Sidebar';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onNavigate: (route: NavRoute, entityId?: string) => void;
}

export const CommandPalette: React.FC<Props> = ({ isOpen, onClose, onNavigate }) => {
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        if (isOpen) onClose();
        else onClose(); // parent handles toggle
      }
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const quickCommands = [
    { id: 'cmd-deploy', label: 'Deploy Application', icon: <Rocket className="w-4 h-4 text-cyan-400" />, route: 'deploy' as NavRoute, category: 'Actions' },
    { id: 'cmd-domain', label: 'Register / Add Domain', icon: <AtSign className="w-4 h-4 text-blue-400" />, route: 'domains' as NavRoute, category: 'Actions' },
    { id: 'cmd-upload', label: 'Upload Storage File', icon: <HardDrive className="w-4 h-4 text-purple-400" />, route: 'storage' as NavRoute, category: 'Actions' },
    { id: 'cmd-copilot', label: 'Ask RAG Copilot', icon: <Sparkles className="w-4 h-4 text-cyan-300" />, route: 'copilot' as NavRoute, category: 'Actions' },
    { id: 'cmd-evidence', label: 'Inspect Sealed Evidence Ledger', icon: <FileCheck2 className="w-4 h-4 text-emerald-400" />, route: 'evidence' as NavRoute, category: 'Actions' }
  ];

  const entities = [
    { id: 'app-ewastekochi', label: 'ewastekochi.com', type: 'Application (Static Site)', route: 'apps' as NavRoute, icon: <Globe className="w-4 h-4 text-emerald-400" /> },
    { id: 'app-bestaiagent', label: 'bestaiagent.in', type: 'Application (Web App)', route: 'apps' as NavRoute, icon: <Globe className="w-4 h-4 text-emerald-400" /> },
    { id: 'app-agentswarm', label: 'agentswarm.in', type: 'Application (Web App)', route: 'apps' as NavRoute, icon: <Globe className="w-4 h-4 text-emerald-400" /> },
    { id: 'app-agentswarm-portal', label: 'app.agentswarm.in', type: 'Application (Web App)', route: 'apps' as NavRoute, icon: <Globe className="w-4 h-4 text-emerald-400" /> },
    { id: 'node-sa-east-1', label: 'sa-east-1 (São Paulo, Brazil)', type: 'Node (Degraded)', route: 'nodes' as NavRoute, icon: <Server className="w-4 h-4 text-amber-400" /> },
    { id: 'node-us-east-1', label: 'us-east-1 (New York, USA)', type: 'Node (Online)', route: 'nodes' as NavRoute, icon: <Server className="w-4 h-4 text-emerald-400" /> },
    { id: 'dom-web3', label: 'myapp.web3', type: 'Domain (Web3 / ENS)', route: 'domains' as NavRoute, icon: <AtSign className="w-4 h-4 text-purple-400" /> },
    { id: 'stor-backups', label: 'backups/db-backup.sql', type: 'Storage Object (SQL)', route: 'storage' as NavRoute, icon: <HardDrive className="w-4 h-4 text-cyan-400" /> },
    { id: 'cert-codingagent', label: 'codingagent.in (SSL Expiring Soon)', type: 'Certificate (Let\'s Encrypt)', route: 'security' as NavRoute, icon: <ShieldCheck className="w-4 h-4 text-amber-400" /> }
  ];

  const filteredCommands = quickCommands.filter((c) =>
    c.label.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const filteredEntities = entities.filter(
    (e) =>
      e.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.type.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-start justify-center pt-24 px-4">
      <div className="w-full max-w-2xl rounded-2xl bg-[#0B1120] border border-slate-700/80 shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Search Input Box */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-slate-800">
          <Search className="w-5 h-5 text-slate-400" />
          <input
            autoFocus
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Type a command, application, node, domain, or ask Copilot..."
            className="flex-1 bg-transparent text-white text-sm focus:outline-none placeholder-slate-500"
          />
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Results List */}
        <div className="max-h-96 overflow-y-auto p-3 space-y-4">
          {/* Quick Actions */}
          {filteredCommands.length > 0 && (
            <div>
              <div className="text-[10px] font-mono uppercase tracking-wider text-slate-500 px-3 py-1">
                Platform Actions
              </div>
              <div className="space-y-1 mt-1">
                {filteredCommands.map((cmd) => (
                  <button
                    key={cmd.id}
                    onClick={() => {
                      onNavigate(cmd.route);
                      onClose();
                    }}
                    className="w-full flex items-center justify-between px-3 py-2.5 rounded-xl hover:bg-slate-800/80 text-left text-sm text-slate-200 transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-1.5 rounded-lg bg-slate-900 border border-slate-800 group-hover:border-slate-700">
                        {cmd.icon}
                      </div>
                      <span className="font-medium text-white">{cmd.label}</span>
                    </div>
                    <span className="text-[11px] font-mono text-slate-500 group-hover:text-slate-300">
                      Jump →
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Resources */}
          {filteredEntities.length > 0 && (
            <div>
              <div className="text-[10px] font-mono uppercase tracking-wider text-slate-500 px-3 py-1">
                Infrastructure Resources
              </div>
              <div className="space-y-1 mt-1">
                {filteredEntities.map((ent) => (
                  <button
                    key={ent.id}
                    onClick={() => {
                      onNavigate(ent.route, ent.id);
                      onClose();
                    }}
                    className="w-full flex items-center justify-between px-3 py-2.5 rounded-xl hover:bg-slate-800/80 text-left text-sm text-slate-200 transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-1.5 rounded-lg bg-slate-900 border border-slate-800 group-hover:border-slate-700">
                        {ent.icon}
                      </div>
                      <div className="flex flex-col">
                        <span className="font-medium text-white">{ent.label}</span>
                        <span className="text-[11px] text-slate-400 font-mono">{ent.type}</span>
                      </div>
                    </div>
                    <span className="text-[11px] font-mono text-slate-500 group-hover:text-cyan-400">
                      Inspect →
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {filteredCommands.length === 0 && filteredEntities.length === 0 && (
            <div className="py-8 text-center text-sm text-slate-400">
              No matching resources found for "{searchTerm}".
            </div>
          )}
        </div>

        {/* Footer info */}
        <div className="px-5 py-2.5 bg-slate-950/60 border-t border-slate-800/80 flex items-center justify-between text-[11px] font-mono text-slate-500">
          <span>Navigate with [↑][↓] · Select [Enter] · Close [Esc]</span>
          <span className="text-cyan-400">Truthfulness Contract Active</span>
        </div>
      </div>
    </div>
  );
};
