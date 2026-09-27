import React from 'react';
import {
  LayoutDashboard,
  AppWindow,
  Rocket,
  Server,
  HardDrive,
  Globe2,
  ShieldCheck,
  BarChart3,
  CreditCard,
  Users,
  Bot,
  Settings,
  ChevronRight,
  Boxes,
  Lock,
  Layers,
  Sparkles
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Sidebar: React.FC = () => {
  const { currentTab, setCurrentTab } = useNetwork();

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'websites', label: 'Websites & Apps', icon: AppWindow },
    { id: 'deploy', label: 'Deploy', icon: Rocket },
    { id: 'nodes', label: 'Nodes & Compute', icon: Server },
    { id: 'storage', label: 'Storage', icon: HardDrive },
    { id: 'depin', label: 'DePIN Marketplace', icon: Boxes, badge: 'P2' },
    { id: 'evidence', label: 'Evidence & Audit', icon: Lock, badge: 'Merkle' },
    { id: 'domains', label: 'Domains', icon: Globe2 },
    { id: 'security', label: 'SSL & Security', icon: ShieldCheck },
    { id: 'analytics', label: 'Analytics', icon: BarChart3 },
    { id: 'billing', label: 'Billing & Settlement', icon: CreditCard },
    { id: 'team', label: 'Team', icon: Users },
    { id: 'rag-copilot', label: 'RAG Copilot', icon: Bot, isAi: true },
    { id: 'settings', label: 'Settings', icon: Settings },
  ];

  return (
    <aside className="w-64 shrink-0 flex flex-col bg-[#070d1e]/90 border-r border-blue-500/20 backdrop-blur-xl h-screen sticky top-0 select-none z-30">
      {/* Brand Logo Header */}
      <div className="p-4 flex items-center gap-3 border-b border-blue-500/15">
        <div className="relative flex items-center justify-center w-10 h-10 rounded-xl bg-gradient-to-br from-cyan-500 via-indigo-600 to-purple-600 p-[1px] shadow-lg shadow-cyan-500/20">
          <div className="w-full h-full bg-[#060b18] rounded-xl flex items-center justify-center overflow-hidden">
            {/* Glowing 3D isometric cube icon */}
            <div className="relative w-6 h-6">
              <svg viewBox="0 0 24 24" className="w-full h-full text-cyan-400 drop-shadow-[0_0_8px_rgba(34,211,238,0.8)]" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 2L2 7l10 5 10-5-10-5z" stroke="url(#logo-grad-1)" fill="rgba(6,182,212,0.2)" />
                <path d="M2 17l10 5 10-5" stroke="url(#logo-grad-2)" fill="none" />
                <path d="M2 12l10 5 10-5" stroke="url(#logo-grad-3)" fill="none" />
                <defs>
                  <linearGradient id="logo-grad-1" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#38bdf8" />
                    <stop offset="100%" stopColor="#818cf8" />
                  </linearGradient>
                  <linearGradient id="logo-grad-2" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#818cf8" />
                    <stop offset="100%" stopColor="#c084fc" />
                  </linearGradient>
                  <linearGradient id="logo-grad-3" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stopColor="#38bdf8" />
                    <stop offset="100%" stopColor="#a855f7" />
                  </linearGradient>
                </defs>
              </svg>
            </div>
          </div>
        </div>
        <div className="flex flex-col">
          <div className="flex items-center gap-1.5">
            <span className="font-extrabold tracking-tight text-white text-[15px] font-mono">
              Decentralized<span className="text-cyan-400">.Host</span>
            </span>
          </div>
          <span className="text-[9px] font-semibold tracking-widest text-cyan-300/70 uppercase">
            YOUR DATA. YOUR RULES.
          </span>
        </div>
      </div>

      {/* Navigation Links */}
      <nav className="flex-1 overflow-y-auto px-3 py-3 space-y-1 custom-scrollbar">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setCurrentTab(item.id)}
              className={`w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-sm font-medium transition-all group relative ${
                isActive
                  ? 'bg-gradient-to-r from-blue-600/30 via-indigo-600/25 to-purple-600/20 text-cyan-300 border border-cyan-500/40 shadow-md shadow-cyan-500/10'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800/40'
              }`}
            >
              <div className="flex items-center gap-3">
                <div
                  className={`p-1.5 rounded-lg transition-colors ${
                    isActive
                      ? 'bg-cyan-500/20 text-cyan-300 shadow-sm shadow-cyan-400/30'
                      : 'text-slate-400 group-hover:text-cyan-400 group-hover:bg-slate-800/60'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                </div>
                <span className="truncate">{item.label}</span>
              </div>

              {item.isAi ? (
                <div className="flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-purple-500/20 text-purple-300 text-[10px] font-mono border border-purple-500/30 animate-pulse">
                  <Sparkles className="w-2.5 h-2.5" />
                  <span>AI</span>
                </div>
              ) : item.badge ? (
                <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-blue-500/15 text-blue-300 border border-blue-500/25">
                  {item.badge}
                </span>
              ) : null}
            </button>
          );
        })}
      </nav>

      {/* System Status Indicator */}
      <div className="px-3 py-2 border-t border-blue-500/15">
        <button
          onClick={() => setCurrentTab('analytics')}
          className="w-full flex items-center justify-between p-2.5 rounded-xl bg-slate-900/60 hover:bg-slate-800/60 border border-emerald-500/25 transition-colors group text-left"
        >
          <div className="flex items-center gap-2.5">
            <span className="relative flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500 shadow-[0_0_8px_#10b981]"></span>
            </span>
            <div className="flex flex-col">
              <span className="text-[10px] uppercase font-semibold tracking-wider text-slate-400">
                System Status
              </span>
              <span className="text-xs font-medium text-emerald-400">
                All Systems Operational
              </span>
            </div>
          </div>
          <ChevronRight className="w-3.5 h-3.5 text-slate-500 group-hover:text-emerald-400 transition-colors" />
        </button>
      </div>

      {/* Cosmic Promo Card */}
      <div className="p-3">
        <div className="relative overflow-hidden rounded-2xl bg-gradient-to-b from-indigo-950/80 via-purple-950/50 to-[#070d1e] border border-indigo-500/30 p-3.5 shadow-xl">
          <div className="absolute -top-10 -right-10 w-28 h-28 bg-cyan-500/20 rounded-full blur-2xl pointer-events-none" />
          <div className="absolute -bottom-10 -left-10 w-28 h-28 bg-purple-500/20 rounded-full blur-2xl pointer-events-none" />

          {/* 3D Cube Glow Illustration */}
          <div className="flex justify-center mb-2.5">
            <div className="w-14 h-14 rounded-xl bg-gradient-to-tr from-cyan-500/30 to-purple-600/30 flex items-center justify-center border border-cyan-400/40 shadow-[0_0_20px_rgba(6,182,212,0.3)]">
              <Layers className="w-7 h-7 text-cyan-300 drop-shadow-[0_0_8px_rgba(56,189,248,0.8)]" />
            </div>
          </div>

          <div className="text-center">
            <h4 className="text-xs font-bold text-white tracking-wide">
              Decentralized Hosting
            </h4>
            <p className="text-[11px] text-cyan-300/80 mt-0.5">
              Distributed. Private. Resilient.
            </p>
            <button
              onClick={() => setCurrentTab('billing')}
              className="mt-3 w-full py-1.5 px-3 rounded-lg text-xs font-semibold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 shadow-lg shadow-indigo-600/30 transition-all flex items-center justify-center gap-1.5"
            >
              <span>Upgrade Plan</span>
              <ChevronRight className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
      </div>
    </aside>
  );
};
