import React from 'react';
import {
  LayoutDashboard,
  Globe,
  Rocket,
  Server,
  HardDrive,
  AtSign,
  ShieldCheck,
  BarChart3,
  CreditCard,
  Users,
  Settings,
  Sparkles,
  FileCheck2,
  Box,
  ChevronRight
} from 'lucide-react';

export type NavRoute =
  | 'dashboard'
  | 'apps'
  | 'deploy'
  | 'nodes'
  | 'storage'
  | 'domains'
  | 'security'
  | 'analytics'
  | 'billing'
  | 'team'
  | 'settings'
  | 'copilot'
  | 'evidence';

interface Props {
  currentRoute: NavRoute;
  onRouteChange: (route: NavRoute) => void;
  collapsed?: boolean;
}

export const Sidebar: React.FC<Props> = ({
  currentRoute,
  onRouteChange,
  collapsed = false
}) => {
  const navItems = [
    { id: 'dashboard' as NavRoute, label: 'Dashboard', icon: <LayoutDashboard className="w-4 h-4" /> },
    { id: 'apps' as NavRoute, label: 'Websites & Apps', icon: <Globe className="w-4 h-4" /> },
    { id: 'deploy' as NavRoute, label: 'Deploy', icon: <Rocket className="w-4 h-4" /> },
    { id: 'nodes' as NavRoute, label: 'Nodes', icon: <Server className="w-4 h-4" /> },
    { id: 'storage' as NavRoute, label: 'Storage', icon: <HardDrive className="w-4 h-4" /> },
    { id: 'domains' as NavRoute, label: 'Domains', icon: <AtSign className="w-4 h-4" /> },
    { id: 'security' as NavRoute, label: 'SSL & Security', icon: <ShieldCheck className="w-4 h-4" /> },
    { id: 'analytics' as NavRoute, label: 'Analytics', icon: <BarChart3 className="w-4 h-4" /> },
    { id: 'billing' as NavRoute, label: 'Billing', icon: <CreditCard className="w-4 h-4" /> },
    { id: 'team' as NavRoute, label: 'Team', icon: <Users className="w-4 h-4" /> },
    { id: 'copilot' as NavRoute, label: 'RAG Copilot', icon: <Sparkles className="w-4 h-4 text-cyan-400" /> },
    { id: 'evidence' as NavRoute, label: 'Evidence Ledger', icon: <FileCheck2 className="w-4 h-4 text-emerald-400" /> },
    { id: 'settings' as NavRoute, label: 'Settings', icon: <Settings className="w-4 h-4" /> }
  ];

  return (
    <aside className="w-64 flex-shrink-0 bg-[rgba(6,16,36,0.7)] backdrop-blur-2xl border-r border-[rgba(125,190,255,0.14)] flex flex-col justify-between h-screen sticky top-0 p-4 select-none z-30 shadow-[4px_0_24px_rgba(2,6,23,0.6)]">
      <div className="flex flex-col gap-6">
        {/* Brand Logo Header matching mm.png */}
        <div className="flex items-center gap-3 px-2 py-1">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-blue-600 via-cyan-500 to-indigo-500 p-0.5 shadow-lg shadow-blue-500/25 flex items-center justify-center">
            <div className="w-full h-full bg-[#050D1E] rounded-[10px] flex items-center justify-center">
              <Box className="w-5 h-5 text-cyan-400 animate-pulse-glow" />
            </div>
          </div>
          <div className="flex flex-col">
            <span className="font-bold text-base tracking-tight text-white flex items-center gap-1.5">
              Decentralized<span className="text-cyan-400">.Host</span>
            </span>
            <span className="text-[9px] uppercase tracking-widest text-slate-400 font-mono font-semibold">
              Your Data. Your Rules.
            </span>
          </div>
        </div>

        {/* Navigation Items */}
        <nav className="flex flex-col gap-1.5 overflow-y-auto max-h-[calc(100vh-320px)] pr-1">
          {navItems.map((item) => {
            const isActive = currentRoute === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onRouteChange(item.id)}
                className={`flex items-center justify-between w-full px-3.5 py-2.5 rounded-2xl text-xs font-semibold transition-all ${
                  isActive
                    ? 'bg-gradient-to-r from-blue-600/90 to-indigo-600/80 text-white shadow-[0_0_20px_rgba(36,139,255,0.35)] border border-cyan-400/40'
                    : 'text-slate-400 hover:text-slate-100 hover:bg-white/[0.04] border border-transparent'
                }`}
              >
                <div className="flex items-center gap-3">
                  <span className={isActive ? 'text-white' : 'text-slate-400'}>{item.icon}</span>
                  <span>{item.label}</span>
                </div>
                {item.id === 'copilot' && (
                  <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-cyan-500/20 text-cyan-300 border border-cyan-500/30">
                    AI
                  </span>
                )}
                {item.id === 'evidence' && (
                  <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                    SEAL
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      </div>

      {/* System Status and Upgrade Plan Cards matching mm.png */}
      <div className="flex flex-col gap-2.5 mt-auto pt-3">
        {/* System Status Pill Card */}
        <div
          onClick={() => onRouteChange('nodes')}
          className="flex items-center justify-between px-3.5 py-2.5 rounded-2xl bg-[rgba(10,24,50,0.55)] border border-[rgba(125,190,255,0.16)] hover:border-emerald-500/40 cursor-pointer transition-all group shadow-md"
        >
          <div className="flex items-center gap-2.5">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_8px_#34d399]" />
            <div className="flex flex-col text-left">
              <span className="text-[10px] text-slate-400 font-mono leading-none">System Status</span>
              <span className="text-xs font-semibold text-emerald-400 mt-0.5">All Systems Operational</span>
            </div>
          </div>
          <ChevronRight className="w-3.5 h-3.5 text-slate-500 group-hover:text-emerald-400 transition-colors" />
        </div>

        {/* Upgrade Plan Card with Spatial Alien Backdrop */}
        <div className="relative overflow-hidden rounded-2xl bg-gradient-to-b from-[rgba(14,30,64,0.7)] to-[rgba(7,16,36,0.85)] border border-[rgba(36,139,255,0.3)] p-4 shadow-xl">
          <div className="absolute top-0 right-0 w-28 h-28 bg-blue-500/15 rounded-full blur-2xl pointer-events-none" />
          <div className="flex items-center gap-3 mb-2">
            <div className="w-8 h-8 rounded-xl bg-blue-600/20 border border-cyan-400/30 flex items-center justify-center shadow-[0_0_12px_rgba(32,221,247,0.3)]">
              <Box className="w-4 h-4 text-cyan-400" />
            </div>
            <div>
              <div className="text-xs font-bold text-white tracking-tight">Decentralized Hosting</div>
              <div className="text-[10px] text-slate-400 font-mono">Distributed. Private. Resilient.</div>
            </div>
          </div>
          <button
            onClick={() => onRouteChange('billing')}
            className="w-full mt-2 py-2 px-3 rounded-xl bg-btn-spatial hover:brightness-110 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 flex items-center justify-center gap-1.5 transition-all cursor-pointer"
          >
            <span>Upgrade Plan</span>
            <ChevronRight className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </aside>
  );
};
