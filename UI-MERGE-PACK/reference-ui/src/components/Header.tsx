import React, { useState } from 'react';
import {
  Search,
  Command,
  Bell,
  Sun,
  Moon,
  ChevronDown,
  User,
  Shield,
  Key,
  LogOut,
  ExternalLink,
  CheckCircle2,
  AlertTriangle
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Header: React.FC = () => {
  const {
    setIsCommandPaletteOpen,
    setCurrentTab,
    clusterStats,
    activities
  } = useNetwork();

  const [showNotifications, setShowNotifications] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(true);

  return (
    <header className="h-16 px-6 bg-[#070d1e]/85 backdrop-blur-xl border-b border-blue-500/20 flex items-center justify-between sticky top-0 z-20">
      {/* Search Bar matching screenshots */}
      <div className="flex-1 max-w-2xl">
        <button
          onClick={() => setIsCommandPaletteOpen(true)}
          className="w-full flex items-center justify-between px-3.5 py-2 rounded-xl bg-slate-900/70 hover:bg-slate-900 border border-blue-500/25 hover:border-cyan-400/50 text-slate-400 hover:text-slate-200 transition-all text-sm group shadow-inner"
        >
          <div className="flex items-center gap-2.5 truncate">
            <Search className="w-4 h-4 text-cyan-400 group-hover:text-cyan-300" />
            <span className="truncate text-xs md:text-sm">
              Search apps, nodes, domains, logs, or ask the AI copilot...
            </span>
          </div>
          <div className="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-slate-800 border border-slate-700 text-[11px] font-mono text-slate-400">
            <Command className="w-3 h-3" />
            <span>K</span>
          </div>
        </button>
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-3 ml-4">
        {/* Dark/Light toggle */}
        <button
          onClick={() => setIsDarkMode(!isDarkMode)}
          className="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-transparent hover:border-slate-700 transition-colors"
          title="Toggle appearance"
        >
          {isDarkMode ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4 text-indigo-300" />}
        </button>

        {/* Notifications Bell */}
        <div className="relative">
          <button
            onClick={() => setShowNotifications(!showNotifications)}
            className="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-transparent hover:border-slate-700 transition-colors relative"
            title="Notifications"
          >
            <Bell className="w-4 h-4" />
            <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-cyan-400 ring-2 ring-[#070d1e] animate-pulse" />
          </button>

          {showNotifications && (
            <div className="absolute right-0 mt-2 w-80 rounded-2xl bg-[#091124] border border-blue-500/30 shadow-2xl p-3 z-50 animate-in fade-in zoom-in-95 duration-150">
              <div className="flex items-center justify-between pb-2 border-b border-slate-800">
                <span className="text-xs font-bold text-white uppercase tracking-wider">
                  Cluster Notifications
                </span>
                <span className="text-[10px] font-mono text-cyan-400 bg-cyan-500/10 px-1.5 py-0.5 rounded">
                  2 unread
                </span>
              </div>
              <div className="space-y-2 mt-2 max-h-72 overflow-y-auto custom-scrollbar">
                <div className="p-2 rounded-xl bg-slate-900/80 border border-amber-500/30 flex items-start gap-2.5">
                  <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                  <div className="text-xs">
                    <p className="font-semibold text-slate-200">Node raspberry-pi Offline</p>
                    <p className="text-[11px] text-slate-400">Heartbeat expired 48h ago. Replicas safely protected on 2 nodes.</p>
                    <span className="text-[10px] text-slate-500">2 days ago</span>
                  </div>
                </div>

                <div className="p-2 rounded-xl bg-slate-900/80 border border-emerald-500/30 flex items-start gap-2.5">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                  <div className="text-xs">
                    <p className="font-semibold text-slate-200">Raft Consensus Sealed</p>
                    <p className="text-[11px] text-slate-400">Signed metering evidence root 0x7c9b committed to state ledger.</p>
                    <span className="text-[10px] text-slate-500">6 hours ago</span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* User Profile Pill matching screenshots: "F / Febin Francis / Owner" */}
        <div className="relative">
          <button
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="flex items-center gap-2.5 pl-2 pr-3 py-1.5 rounded-xl bg-slate-900/70 hover:bg-slate-800 border border-blue-500/20 hover:border-cyan-500/40 transition-all text-left"
          >
            <div className="w-7 h-7 rounded-lg bg-gradient-to-tr from-blue-600 via-indigo-600 to-purple-500 flex items-center justify-center text-white font-bold text-xs shadow-md shadow-indigo-600/30">
              F
            </div>
            <div className="hidden sm:flex flex-col">
              <span className="text-xs font-semibold text-slate-200 leading-tight">
                Febin Francis
              </span>
              <span className="text-[10px] text-cyan-400/90 leading-none">
                Owner
              </span>
            </div>
            <ChevronDown className="w-3.5 h-3.5 text-slate-400 ml-0.5" />
          </button>

          {showUserMenu && (
            <div className="absolute right-0 mt-2 w-56 rounded-2xl bg-[#091124] border border-blue-500/30 shadow-2xl p-2 z-50 animate-in fade-in zoom-in-95 duration-150">
              <div className="px-3 py-2 border-b border-slate-800">
                <p className="text-xs font-bold text-white">Febin Francis</p>
                <p className="text-[11px] text-slate-400 truncate">kochiewaste@gmail.com</p>
                <div className="mt-1 flex items-center gap-1.5">
                  <span className="w-2 h-2 rounded-full bg-emerald-400" />
                  <span className="text-[10px] text-emerald-400 font-mono">Sovereign Cluster Key Active</span>
                </div>
              </div>

              <div className="py-1">
                <button
                  onClick={() => {
                    setCurrentTab('security');
                    setShowUserMenu(false);
                  }}
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs text-slate-300 hover:text-white hover:bg-slate-800 transition-colors"
                >
                  <Key className="w-3.5 h-3.5 text-cyan-400" />
                  <span>Hardware Keys & Identity</span>
                </button>
                <button
                  onClick={() => {
                    setCurrentTab('team');
                    setShowUserMenu(false);
                  }}
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs text-slate-300 hover:text-white hover:bg-slate-800 transition-colors"
                >
                  <User className="w-3.5 h-3.5 text-purple-400" />
                  <span>Team & Access Control</span>
                </button>
                <button
                  onClick={() => {
                    setCurrentTab('settings');
                    setShowUserMenu(false);
                  }}
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs text-slate-300 hover:text-white hover:bg-slate-800 transition-colors"
                >
                  <Shield className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Cluster Settings</span>
                </button>
              </div>

              <div className="pt-1 border-t border-slate-800">
                <button
                  onClick={() => setShowUserMenu(false)}
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs text-rose-400 hover:bg-rose-500/10 transition-colors"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span>Lock Session</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
