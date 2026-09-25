import React, { useState } from 'react';
import { Search, Bell, Sun, Moon, ChevronDown, ShieldCheck, Check, User, LogOut, Box, Sparkles } from 'lucide-react';
import { PlatformCapabilities } from '../../types/platform';
import { CapabilityBadge } from '../common/CapabilityBadge';

interface Props {
  onOpenCommandPalette: () => void;
  capabilities?: PlatformCapabilities;
  unreadNotificationsCount?: number;
  onOpenNotifications?: () => void;
  onNavigateHome?: () => void;
}

export const TopBar: React.FC<Props> = ({
  onOpenCommandPalette,
  unreadNotificationsCount = 1,
  onOpenNotifications,
  onNavigateHome
}) => {
  const [themeDark, setThemeDark] = useState(true);
  const [profileOpen, setProfileOpen] = useState(false);

  return (
    <header className="sticky top-0 z-20 bg-[rgba(6,16,36,0.72)] backdrop-blur-2xl border-b border-[rgba(125,190,255,0.14)] px-5 py-3 flex flex-col gap-2.5 shadow-[0_4px_24px_rgba(2,6,23,0.5)]">
      {/* Main Top Command Row */}
      <div className="flex items-center justify-between gap-4">
        {/* Left: Brand Header with Luminous 3D Cube Logo & Tagline */}
        <div
          onClick={onNavigateHome}
          className="flex items-center gap-3 cursor-pointer select-none group"
        >
          <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-cyan-400 via-blue-600 to-violet-600 p-0.5 shadow-[0_0_16px_rgba(32,221,247,0.35)] flex items-center justify-center transition-transform group-hover:scale-105">
            <div className="w-full h-full bg-[#050D1E] rounded-[10px] flex items-center justify-center">
              <Box className="w-5 h-5 text-cyan-400 animate-pulse-glow" />
            </div>
          </div>
          <div className="flex flex-col">
            <span className="font-extrabold text-sm sm:text-base tracking-tight text-white flex items-center">
              Decentralized<span className="text-cyan-400">.Host</span>
            </span>
            <span className="text-[9px] uppercase tracking-widest text-slate-400 font-mono font-semibold">
              Your Data. Your Rules.
            </span>
          </div>
        </div>

        {/* Center: Omnibar with ⌘ K indicator */}
        <div
          onClick={onOpenCommandPalette}
          className="flex-1 max-w-xl hidden md:flex items-center gap-3 px-4 py-2 rounded-2xl bg-[rgba(10,24,50,0.65)] border border-[rgba(125,190,255,0.16)] hover:border-[rgba(125,190,255,0.35)] text-slate-400 text-xs cursor-pointer transition-all shadow-inner group"
        >
          <Search className="w-4 h-4 text-cyan-400/80 group-hover:text-cyan-400 transition-colors" />
          <span className="flex-1 text-slate-400 font-sans truncate">
            Search apps, nodes, domains, logs, or ask the AI copilot...
          </span>
          <div className="flex items-center gap-1 px-2 py-0.5 rounded-lg bg-[rgba(18,38,76,0.7)] border border-[rgba(125,190,255,0.2)] font-mono text-[10px] text-cyan-300">
            <span>⌘</span>
            <span>K</span>
          </div>
        </div>

        {/* Right Cluster: Theme, Notifications, User Badge */}
        <div className="flex items-center gap-3">
          {/* Day / Night Toggle */}
          <button
            onClick={() => setThemeDark(!themeDark)}
            className="p-2 rounded-xl bg-[rgba(10,24,50,0.6)] border border-[rgba(125,190,255,0.16)] hover:border-[rgba(125,190,255,0.3)] text-slate-400 hover:text-slate-200 transition-all cursor-pointer"
            title="Toggle theme"
          >
            {themeDark ? <Moon className="w-4 h-4 text-cyan-400" /> : <Sun className="w-4 h-4 text-amber-400" />}
          </button>

          {/* Notifications */}
          <button
            onClick={onOpenNotifications}
            className="relative p-2 rounded-xl bg-[rgba(10,24,50,0.6)] border border-[rgba(125,190,255,0.16)] hover:border-[rgba(125,190,255,0.3)] text-slate-400 hover:text-slate-200 transition-all cursor-pointer"
            title="Recent platform alerts"
          >
            <Bell className="w-4 h-4 text-slate-300" />
            {unreadNotificationsCount > 0 && (
              <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-rose-500 animate-ping" />
            )}
            {unreadNotificationsCount > 0 && (
              <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-rose-500 shadow-[0_0_8px_#f43f5e]" />
            )}
          </button>

          {/* User profile dropdown matching design.md */}
          <div className="relative">
            <button
              onClick={() => setProfileOpen(!profileOpen)}
              className="flex items-center gap-2.5 pl-2 pr-3 py-1.5 rounded-2xl bg-[rgba(10,24,50,0.65)] border border-[rgba(125,190,255,0.16)] hover:border-[rgba(125,190,255,0.32)] transition-all cursor-pointer"
            >
              <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-cyan-500 via-blue-600 to-indigo-600 flex items-center justify-center font-bold text-white text-xs shadow-[0_0_12px_rgba(32,221,247,0.4)]">
                F
              </div>
              <div className="flex flex-col text-left leading-tight hidden sm:flex">
                <span className="text-xs font-semibold text-white">Febin Francis</span>
                <span className="text-[10px] text-cyan-400/90 font-mono">Owner</span>
              </div>
              <ChevronDown className="w-3.5 h-3.5 text-slate-400 ml-1" />
            </button>

            {profileOpen && (
              <div className="absolute right-0 mt-2 w-60 rounded-2xl alien-glass-floating border border-[rgba(125,190,255,0.25)] shadow-2xl p-2 z-50 text-xs">
                <div className="px-3 py-2 border-b border-[rgba(125,190,255,0.12)]">
                  <div className="font-semibold text-white">Febin Francis</div>
                  <div className="text-[11px] text-slate-400 font-mono">febin@decentralized.host</div>
                  <div className="mt-1.5 flex items-center gap-1.5 text-[10px] text-emerald-400 font-mono">
                    <Check className="w-3 h-3" />
                    <span>Cryptographic Identity Verified</span>
                  </div>
                </div>

                <div className="py-1">
                  <button
                    onClick={() => setProfileOpen(false)}
                    className="w-full text-left px-3 py-2 text-slate-300 hover:bg-white/[0.06] rounded-xl flex items-center gap-2"
                  >
                    <User className="w-3.5 h-3.5 text-slate-400" />
                    <span>Profile & Keyrings</span>
                  </button>
                  <button
                    onClick={() => setProfileOpen(false)}
                    className="w-full text-left px-3 py-2 text-slate-300 hover:bg-white/[0.06] rounded-xl flex items-center gap-2"
                  >
                    <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
                    <span>Audit & Permissions</span>
                  </button>
                  <button
                    onClick={() => setProfileOpen(false)}
                    className="w-full text-left px-3 py-2 text-rose-400 hover:bg-rose-950/30 rounded-xl flex items-center gap-2"
                  >
                    <LogOut className="w-3.5 h-3.5" />
                    <span>Lock Session</span>
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Operational State Strip: Desired / Observed / Evidence */}
      <div className="flex flex-wrap items-center justify-between gap-2 px-3.5 py-1.5 rounded-xl bg-[rgba(8,18,38,0.85)] border border-[rgba(125,190,255,0.12)] text-[11px] font-mono text-slate-400">
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <div className="flex items-center gap-1.5 text-emerald-400 font-semibold">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_8px_#10D981]" />
            <span>CONTROL PLANE: ACTIVE</span>
          </div>

          <span className="text-slate-600 hidden sm:inline">|</span>

          <div className="flex items-center gap-1.5">
            <span className="text-slate-500 uppercase text-[10px]">Desired:</span>
            <span className="text-slate-300 font-medium">10 Nodes · 12 Apps · 3× Quorum</span>
          </div>

          <span className="text-slate-600 hidden sm:inline">|</span>

          <div className="flex items-center gap-1.5">
            <span className="text-slate-500 uppercase text-[10px]">Observed:</span>
            <span className="text-emerald-400 font-semibold">8 Online</span>
            <span className="text-slate-600">/</span>
            <span className="text-amber-400 font-semibold">1 Degraded</span>
            <span className="text-slate-600">/</span>
            <span className="text-slate-400">1 Offline</span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-cyan-300">
            <ShieldCheck className="w-3.5 h-3.5 text-cyan-400" />
            <span className="hidden lg:inline">EVIDENCE:</span>
            <span>QUAL-DH-2026-0925-A1</span>
            <span className="text-emerald-400 font-semibold">(VERIFIED)</span>
          </div>
          <span className="text-slate-600 hidden sm:inline">|</span>
          <CapabilityBadge state="LIVE" />
        </div>
      </div>
    </header>
  );
};
