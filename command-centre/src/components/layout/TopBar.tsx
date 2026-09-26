import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Bell, Sun, ChevronDown, ShieldCheck, KeyRound, LogOut, Menu } from 'lucide-react';
import { BrandMark } from '../common/Brand';
import { useSession } from '../../lib/session';
import { fmtTime } from '../common/states';

interface Props {
  onOpenCommandPalette: () => void;
  unread: number;
  onOpenNotifications: () => void;
  onOpenMobileNav: () => void;
}

/** Role label derived from the capability's granted actions. */
export function roleOf(actions: string[]): string {
  if (actions.includes('api.admin')) return 'Admin';
  if (actions.includes('api.write')) return 'Operator';
  if (actions.includes('api.read')) return 'Read-only';
  return 'No access';
}

export const TopBar: React.FC<Props> = ({ onOpenCommandPalette, unread, onOpenNotifications, onOpenMobileNav }) => {
  const [profileOpen, setProfileOpen] = useState(false);
  const { session, mode, signOut } = useSession();
  const navigate = useNavigate();
  const signedIn = !!session?.authenticated;
  const actor = session?.actor ?? (mode === 'demo' ? 'demo viewer' : 'not signed in');
  const initial = (actor.replace(/^operator:/, '')[0] ?? '?').toUpperCase();

  return (
    <header className="sticky top-0 z-30 h-[68px] px-3 sm:px-4 flex items-center gap-2 sm:gap-4 bg-[linear-gradient(180deg,rgba(2,7,17,0.92),rgba(2,7,17,0.6))] backdrop-blur-xl">
      <button onClick={onOpenMobileNav} className="lg:hidden p-2 rounded-xl border border-[rgba(125,190,255,0.2)] text-slate-200" aria-label="Open navigation">
        <Menu className="w-5 h-5" />
      </button>

      <button onClick={() => navigate('/')} className="flex items-center gap-2.5 lg:min-w-[212px] pr-2 shrink-0 group" aria-label="Decentralized.Host home">
        <BrandMark size={42} className="shrink-0 transition-transform group-hover:scale-105" />
        <span className="hidden min-[400px]:flex flex-col items-start leading-none">
          <span className="text-[16px] sm:text-[19px] font-extrabold tracking-tight text-white">Decentralized.Host</span>
          <span className="mt-1 text-[9.5px] font-bold tracking-[0.2em] text-slate-300/80">YOUR DATA. YOUR RULES.</span>
        </span>
      </button>

      <button
        onClick={onOpenCommandPalette}
        className="hidden md:flex flex-1 max-w-[520px] items-center gap-3 px-4 h-11 rounded-2xl bg-[rgba(8,20,44,0.7)] border border-[rgba(125,190,255,0.28)] hover:border-cyan-400/50 text-left transition-colors"
      >
        <Search className="w-[18px] h-[18px] text-slate-200" />
        <span className="flex-1 text-[13px] text-slate-400 truncate">Search apps, nodes, domains, storage, or ask the AI copilot...</span>
        <kbd className="flex items-center gap-1 px-2 py-0.5 rounded-md bg-white/[0.06] border border-white/10 text-[11px] text-slate-300 font-sans">⌘ K</kbd>
      </button>

      <div className="flex items-center gap-2 sm:gap-3 ml-auto">
        <div className="hidden sm:flex items-center gap-1.5 h-9 pl-2 pr-1 rounded-full bg-[rgba(8,20,44,0.7)] border border-[rgba(125,190,255,0.22)]" title="Spatial dark theme (the console ships one theme)">
          <Sun className="w-4 h-4 text-slate-200" />
          <span className="w-6 h-6 rounded-full bg-[radial-gradient(circle_at_35%_30%,#c7d2fe,#6366f1_60%,#312e81)] shadow-[0_0_10px_rgba(99,102,241,0.8)]" />
        </div>

        <button onClick={onOpenNotifications} className="relative p-2 rounded-xl text-slate-200 hover:bg-white/[0.06] transition-colors" aria-label={`Notifications${unread ? ` (${unread} open incidents)` : ''}`}>
          <Bell className="w-[22px] h-[22px]" />
          {unread > 0 && (
            <span className="absolute top-0.5 right-0.5 min-w-4 h-4 px-1 rounded-full bg-rose-500 text-[9.5px] font-bold text-white flex items-center justify-center">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </button>

        <div className="relative">
          <button onClick={() => setProfileOpen(!profileOpen)} className="flex items-center gap-2.5 pl-1 pr-2 py-1 rounded-2xl hover:bg-white/[0.05] transition-colors" aria-expanded={profileOpen}>
            <span className="w-9 h-9 rounded-full bg-[linear-gradient(135deg,#6366f1,#8b5cf6)] flex items-center justify-center font-bold text-white text-[15px]">{initial}</span>
            <span className="hidden sm:flex flex-col items-start leading-tight max-w-[160px]">
              <span className="text-[13.5px] font-semibold text-white truncate max-w-full">{actor}</span>
              <span className="text-[11px] text-slate-400">{signedIn ? roleOf(session!.actions) : mode === 'demo' ? 'Simulated' : 'Signed out'}</span>
            </span>
            <ChevronDown className="hidden sm:block w-4 h-4 text-slate-300" />
          </button>

          {profileOpen && (
            <div className="absolute right-0 mt-2 w-72 rounded-2xl alien-glass-floating p-2 z-50 text-xs">
              <div className="px-3 py-2 border-b border-[rgba(125,190,255,0.12)]">
                <div className="font-semibold text-white break-all">{actor}</div>
                {signedIn && (
                  <>
                    <div className="mt-1 text-[11px] text-slate-400">cluster {session!.cluster ?? '—'} · {session!.actions.join(', ') || 'no actions'}</div>
                    <div className="text-[11px] text-slate-400">expires {fmtTime(session!.expiresAt)}</div>
                    <div className="mt-1.5 flex items-center gap-1.5 text-[10.5px] text-emerald-400">
                      <ShieldCheck className="w-3 h-3" /> Capability accepted by the control plane
                    </div>
                  </>
                )}
              </div>
              <div className="py-1">
                <button
                  onClick={() => {
                    setProfileOpen(false);
                    navigate('/team');
                  }}
                  className="w-full text-left px-3 py-2 text-slate-300 hover:bg-white/[0.06] rounded-xl flex items-center gap-2"
                >
                  <KeyRound className="w-3.5 h-3.5 text-slate-400" /> Session & capabilities
                </button>
                {signedIn && (
                  <button
                    onClick={() => {
                      setProfileOpen(false);
                      signOut();
                    }}
                    className="w-full text-left px-3 py-2 text-rose-400 hover:bg-rose-950/30 rounded-xl flex items-center gap-2"
                  >
                    <LogOut className="w-3.5 h-3.5" /> Sign out
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
