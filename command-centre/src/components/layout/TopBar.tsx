import React, { useState } from 'react';
import { Search, Bell, Sun, ChevronDown, ShieldCheck, Check, User, LogOut, Menu } from 'lucide-react';
import { PlatformCapabilities } from '../../types/platform';
import { BrandMark } from '../common/Brand';

interface Props {
  onOpenCommandPalette: () => void;
  capabilities?: PlatformCapabilities;
  unreadNotificationsCount?: number;
  onOpenNotifications?: () => void;
  onNavigateHome?: () => void;
  onOpenMobileNav?: () => void;
  onOpenAudit?: () => void;
}

export const TopBar: React.FC<Props> = ({
  onOpenCommandPalette,
  unreadNotificationsCount = 0,
  onOpenNotifications,
  onNavigateHome,
  onOpenMobileNav,
  onOpenAudit
}) => {
  const [profileOpen, setProfileOpen] = useState(false);

  return (
    <header className="sticky top-0 z-30 h-[68px] px-3 sm:px-4 flex items-center gap-2 sm:gap-4 bg-[linear-gradient(180deg,rgba(2,7,17,0.92),rgba(2,7,17,0.6))] backdrop-blur-xl">
      <button
        onClick={onOpenMobileNav}
        className="lg:hidden p-2 rounded-xl border border-[rgba(125,190,255,0.2)] text-slate-200"
        aria-label="Open navigation"
      >
        <Menu className="w-5 h-5" />
      </button>

      {/* Brand */}
      <button onClick={onNavigateHome} className="flex items-center gap-2.5 lg:min-w-[212px] pr-2 shrink-0 group" aria-label="Decentralized.Host home">
        <BrandMark size={42} className="shrink-0 transition-transform group-hover:scale-105" />
        <span className="hidden min-[400px]:flex flex-col items-start leading-none">
          <span className="text-[16px] sm:text-[19px] font-extrabold tracking-tight text-white">
            Decentralized<span className="text-white/95">.Host</span>
          </span>
          <span className="mt-1 text-[9.5px] font-bold tracking-[0.2em] text-slate-300/80">YOUR DATA. YOUR RULES.</span>
        </span>
      </button>

      {/* Omnibar */}
      <button
        onClick={onOpenCommandPalette}
        className="hidden md:flex flex-1 max-w-[520px] items-center gap-3 px-4 h-11 rounded-2xl bg-[rgba(8,20,44,0.7)] border border-[rgba(125,190,255,0.28)] hover:border-cyan-400/50 text-left shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_0_20px_rgba(36,139,255,0.08)] transition-colors"
      >
        <Search className="w-[18px] h-[18px] text-slate-200" />
        <span className="flex-1 text-[13px] text-slate-400 truncate">Search apps, nodes, domains, storage, or ask the AI copilot...</span>
        <kbd className="flex items-center gap-1 px-2 py-0.5 rounded-md bg-white/[0.06] border border-white/10 text-[11px] text-slate-300 font-sans">
          ⌘ K
        </kbd>
      </button>

      <div className="flex items-center gap-2 sm:gap-3 ml-auto">
        {/* Theme indicator (dark-only for now; the control plane ships a single spatial theme) */}
        <div
          className="hidden sm:flex items-center gap-1.5 h-9 pl-2 pr-1 rounded-full bg-[rgba(8,20,44,0.7)] border border-[rgba(125,190,255,0.22)]"
          title="Spatial dark theme"
        >
          <Sun className="w-4 h-4 text-slate-200" />
          <span className="w-6 h-6 rounded-full bg-[radial-gradient(circle_at_35%_30%,#c7d2fe,#6366f1_60%,#312e81)] shadow-[0_0_10px_rgba(99,102,241,0.8)]" />
        </div>

        <button
          onClick={onOpenNotifications}
          className="relative p-2 rounded-xl text-slate-200 hover:bg-white/[0.06] transition-colors"
          aria-label={`Notifications${unreadNotificationsCount ? ` (${unreadNotificationsCount} unread)` : ''}`}
        >
          <Bell className="w-[22px] h-[22px]" />
          {unreadNotificationsCount > 0 && (
            <span className="absolute top-0.5 right-0.5 min-w-4 h-4 px-1 rounded-full bg-rose-500 text-[9.5px] font-bold text-white flex items-center justify-center shadow-[0_0_8px_#f43f5e]">
              {unreadNotificationsCount > 9 ? '9+' : unreadNotificationsCount}
            </span>
          )}
        </button>

        <div className="relative">
          <button
            onClick={() => setProfileOpen(!profileOpen)}
            className="flex items-center gap-2.5 pl-1 pr-2 py-1 rounded-2xl hover:bg-white/[0.05] transition-colors"
            aria-expanded={profileOpen}
          >
            <span className="w-9 h-9 rounded-full bg-[linear-gradient(135deg,#6366f1,#8b5cf6)] flex items-center justify-center font-bold text-white text-[15px] shadow-[0_0_14px_rgba(139,92,246,0.6)]">
              F
            </span>
            <span className="hidden sm:flex flex-col items-start leading-tight">
              <span className="text-[13.5px] font-semibold text-white">Febin Francis</span>
              <span className="text-[11px] text-slate-400">Owner</span>
            </span>
            <ChevronDown className="hidden sm:block w-4 h-4 text-slate-300" />
          </button>

          {profileOpen && (
            <div className="absolute right-0 mt-2 w-60 rounded-2xl alien-glass-floating p-2 z-50 text-xs">
              <div className="px-3 py-2 border-b border-[rgba(125,190,255,0.12)]">
                <div className="font-semibold text-white">Febin Francis</div>
                <div className="text-[11px] text-slate-400">febin@decentralized.host</div>
                <div className="mt-1.5 flex items-center gap-1.5 text-[10px] text-emerald-400">
                  <Check className="w-3 h-3" />
                  <span>Cryptographic Identity Verified</span>
                </div>
              </div>
              <div className="py-1">
                <button onClick={() => setProfileOpen(false)} className="w-full text-left px-3 py-2 text-slate-300 hover:bg-white/[0.06] rounded-xl flex items-center gap-2">
                  <User className="w-3.5 h-3.5 text-slate-400" />
                  <span>Profile & Keyrings</span>
                </button>
                <button
                  onClick={() => {
                    setProfileOpen(false);
                    onOpenAudit?.();
                  }}
                  className="w-full text-left px-3 py-2 text-slate-300 hover:bg-white/[0.06] rounded-xl flex items-center gap-2"
                >
                  <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Audit & Evidence</span>
                </button>
                <button onClick={() => setProfileOpen(false)} className="w-full text-left px-3 py-2 text-rose-400 hover:bg-rose-950/30 rounded-xl flex items-center gap-2">
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
