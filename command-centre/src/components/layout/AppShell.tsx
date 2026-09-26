import React, { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { X, Bell, AlertTriangle, CloudOff } from 'lucide-react';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { CommandPalette } from './CommandPalette';
import { SignIn } from './SignIn';
import { useSession } from '../../lib/session';
import { useResource } from '../../lib/useResource';
import type { AuditEntryRec } from '../../types/reality';
import { Loading, since } from '../common/states';

export const AppShell: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { session, mode, loading, capabilities } = useSession();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const location = useLocation();

  useEffect(() => {
    setMobileNavOpen(false);
    window.scrollTo({ top: 0 });
  }, [location.pathname]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen((o) => !o);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const signedIn = mode === 'demo' || !!session?.authenticated;
  const overview = useResource<{ overview: { incidents: string[] }; recentEvents: AuditEntryRec[] }>(signedIn ? '/overview' : null, { pollMs: 15_000 });
  const incidents = overview.data?.overview.incidents ?? [];
  const backendDown = capabilities ? !capabilities.backend.reachable : false;

  return (
    <div className="min-h-screen spatial-cosmos-bg text-slate-100 font-sans selection:bg-cyan-500 selection:text-black relative">
      <a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-2 focus:px-3 focus:py-2 focus:rounded-lg focus:bg-cyan-500 focus:text-black">
        Skip to content
      </a>
      <div className="fixed inset-0 stars-field opacity-70 pointer-events-none -z-10" />
      <div className="fixed top-[-160px] left-[25%] w-[700px] h-[500px] bg-blue-600/15 rounded-full blur-[160px] pointer-events-none -z-10" />
      <div className="fixed top-[20%] right-[-160px] w-[600px] h-[600px] bg-violet-600/15 rounded-full blur-[170px] pointer-events-none -z-10" />
      <div className="fixed bottom-[-200px] left-[-100px] w-[600px] h-[500px] bg-cyan-500/10 rounded-full blur-[160px] pointer-events-none -z-10" />

      <TopBar
        onOpenCommandPalette={() => setPaletteOpen(true)}
        unread={incidents.length}
        onOpenNotifications={() => setNotificationsOpen(true)}
        onOpenMobileNav={() => setMobileNavOpen(true)}
      />

      {mode === 'demo' && (
        <div className="mx-4 mb-2 px-4 py-2 rounded-xl border border-amber-400/40 bg-amber-500/10 text-[12.5px] text-amber-100" role="status">
          <b className="text-amber-300">DEMO REPLAY</b> — every value is SIMULATED from a recorded dev-cluster view. Mutations are refused. Set PLATFORM_ADAPTER=controlplane to connect a real control plane.
        </div>
      )}
      {backendDown && signedIn && (
        <div className="mx-4 mb-2 px-4 py-2 rounded-xl border border-rose-400/40 bg-rose-500/10 text-[12.5px] text-rose-100 flex items-center gap-2" role="alert">
          <CloudOff className="w-4 h-4 text-rose-300" />
          <b>CONTROL PLANE UNREACHABLE</b> — {capabilities?.backend.detail}. Values shown are last observations, not current state.
        </div>
      )}

      <div className="flex">
        <Sidebar />
        {mobileNavOpen && (
          <div className="lg:hidden fixed inset-0 z-40 bg-black/60 backdrop-blur-sm" onClick={() => setMobileNavOpen(false)}>
            <Sidebar mobile onClose={() => setMobileNavOpen(false)} />
          </div>
        )}
        <main id="main" className="flex-1 min-w-0 px-4 lg:px-5 pb-8 pt-1">
          {loading ? <Loading label="Checking session…" /> : signedIn ? children : <SignIn />}
        </main>
      </div>

      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />

      {notificationsOpen && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex justify-end" onClick={() => setNotificationsOpen(false)}>
          <div className="w-full max-w-md bg-[#07101f] border-l border-slate-800 h-full p-5 overflow-y-auto" onClick={(e) => e.stopPropagation()} role="dialog" aria-label="Notifications">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div className="flex items-center gap-2">
                <Bell className="w-4 h-4 text-cyan-400" />
                <h3 className="font-bold text-white text-sm">Incidents & recent audit</h3>
              </div>
              <button onClick={() => setNotificationsOpen(false)} className="p-1 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white" aria-label="Close">
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="mt-4 space-y-2">
              {incidents.length === 0 && <p className="text-[12.5px] text-slate-400">The control plane reports no open incidents.</p>}
              {incidents.map((i) => (
                <div key={i} className="flex gap-2 p-3 rounded-xl bg-rose-500/10 border border-rose-400/30 text-[12.5px] text-rose-100">
                  <AlertTriangle className="w-4 h-4 text-rose-300 shrink-0" />
                  {i}
                </div>
              ))}
            </div>
            <h4 className="mt-5 mb-2 text-[11px] uppercase tracking-wider text-slate-500">Audit ledger</h4>
            <div className="space-y-2">
              {(overview.data?.recentEvents ?? []).map((e) => (
                <div key={e.seq} className="p-3 rounded-xl bg-slate-900/70 border border-slate-800">
                  <div className="flex justify-between text-[10.5px] text-slate-500 font-mono">
                    <span>#{e.seq} {e.action}</span>
                    <span>{since(e.ts)}</span>
                  </div>
                  <div className="text-[12.5px] text-slate-200 mt-0.5 break-words">{e.detail || e.resource}</div>
                  <div className="text-[10.5px] text-slate-500 mt-0.5">actor {e.actor}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
