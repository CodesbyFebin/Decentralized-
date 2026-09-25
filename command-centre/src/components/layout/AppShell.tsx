import React, { useState, useEffect } from 'react';
import { Sidebar, NavRoute } from './Sidebar';
import { TopBar } from './TopBar';
import { CommandPalette } from './CommandPalette';
import { DashboardView } from '../pages/DashboardView';
import { WebsitesAppsView } from '../pages/WebsitesAppsView';
import { DeployView } from '../pages/DeployView';
import { NodesView } from '../pages/NodesView';
import { StorageView } from '../pages/StorageView';
import { DomainsView } from '../pages/DomainsView';
import { SecurityView } from '../pages/SecurityView';
import { AnalyticsView } from '../pages/AnalyticsView';
import { BillingView } from '../pages/BillingView';
import { TeamView } from '../pages/TeamView';
import { SettingsView } from '../pages/SettingsView';
import { CopilotView } from '../pages/CopilotView';
import { EvidenceView } from '../pages/EvidenceView';
import { api } from '../../lib/api';
import { PlatformCapabilities, PlatformEvent } from '../../types/platform';
import { X, Bell, ExternalLink } from 'lucide-react';

export const AppShell: React.FC = () => {
  // Real URL routing: parse initial route from window.location.pathname
  const getRouteFromPath = (): NavRoute => {
    const p = window.location.pathname.replace(/^\//, '').split('/')[0];
    const validRoutes: NavRoute[] = [
      'dashboard', 'apps', 'deploy', 'nodes', 'storage', 'domains',
      'security', 'analytics', 'billing', 'team', 'settings', 'copilot', 'evidence'
    ];
    return validRoutes.includes(p as NavRoute) ? (p as NavRoute) : 'dashboard';
  };

  const [currentRoute, setCurrentRoute] = useState<NavRoute>(getRouteFromPath());
  const [selectedEntityId, setSelectedEntityId] = useState<string | undefined>(() => {
    const parts = window.location.pathname.replace(/^\//, '').split('/');
    return parts.length > 1 ? parts[1] : undefined;
  });
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [capabilities, setCapabilities] = useState<PlatformCapabilities | undefined>();
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [recentEvents, setRecentEvents] = useState<PlatformEvent[]>([]);

  useEffect(() => {
    api.getCapabilities().then((res) => setCapabilities(res.capabilities)).catch(() => {});
    api.getOverview().then((res) => setRecentEvents(res.recentActivity || [])).catch(() => {});

    // Listen to browser back/forward navigation
    const handlePopState = () => {
      setCurrentRoute(getRouteFromPath());
      const parts = window.location.pathname.replace(/^\//, '').split('/');
      setSelectedEntityId(parts.length > 1 ? parts[1] : undefined);
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const handleNavigate = (route: NavRoute, entityId?: string) => {
    setCurrentRoute(route);
    setSelectedEntityId(entityId);
    const newPath = entityId ? `/${route}/${entityId}` : route === 'dashboard' ? '/' : `/${route}`;
    if (window.location.pathname !== newPath) {
      window.history.pushState(null, '', newPath);
    }
    window.scrollTo({ top: 0, behavior: 'smooth' });
    setMobileNavOpen(false);
  };

  return (
    <div className="min-h-screen spatial-cosmos-bg text-slate-100 font-sans selection:bg-cyan-500 selection:text-black relative">
      {/* Deep-space backdrop: star field + nebula glow (Layer 0 in design.md) */}
      <div className="fixed inset-0 stars-field opacity-70 pointer-events-none -z-10" />
      <div className="fixed top-[-160px] left-[25%] w-[700px] h-[500px] bg-blue-600/15 rounded-full blur-[160px] pointer-events-none -z-10" />
      <div className="fixed top-[20%] right-[-160px] w-[600px] h-[600px] bg-violet-600/15 rounded-full blur-[170px] pointer-events-none -z-10" />
      <div className="fixed bottom-[-200px] left-[-100px] w-[600px] h-[500px] bg-cyan-500/10 rounded-full blur-[160px] pointer-events-none -z-10" />

      <TopBar
        onOpenCommandPalette={() => setCommandPaletteOpen(true)}
        capabilities={capabilities}
        unreadNotificationsCount={recentEvents.filter((e) => e.severity === 'warn' || e.severity === 'error').length || Math.min(recentEvents.length, 2)}
        onOpenNotifications={() => setNotificationsOpen(true)}
        onNavigateHome={() => handleNavigate('dashboard')}
        onOpenMobileNav={() => setMobileNavOpen(true)}
        onOpenAudit={() => handleNavigate('evidence')}
      />

      <div className="flex">
        <Sidebar currentRoute={currentRoute} onRouteChange={handleNavigate} />
        {mobileNavOpen && (
          <div className="lg:hidden fixed inset-0 z-40 bg-black/60 backdrop-blur-sm" onClick={() => setMobileNavOpen(false)}>
            <Sidebar
              mobile
              currentRoute={currentRoute}
              onRouteChange={(r, id) => {
                setMobileNavOpen(false);
                handleNavigate(r, id);
              }}
              onClose={() => setMobileNavOpen(false)}
            />
          </div>
        )}

        {/* Dynamic Route View */}
        <main className="flex-1 min-w-0 px-4 lg:px-5 pb-8 pt-1">
          {currentRoute === 'dashboard' && <DashboardView onNavigate={handleNavigate} />}
          {currentRoute === 'apps' && (
            <WebsitesAppsView onNavigate={handleNavigate} selectedAppId={selectedEntityId} />
          )}
          {currentRoute === 'deploy' && <DeployView onNavigate={handleNavigate} entityId={selectedEntityId} />}
          {currentRoute === 'nodes' && (
            <NodesView onNavigate={handleNavigate} selectedNodeId={selectedEntityId} />
          )}
          {currentRoute === 'storage' && <StorageView onNavigate={handleNavigate} entityId={selectedEntityId} />}
          {currentRoute === 'domains' && <DomainsView onNavigate={handleNavigate} />}
          {currentRoute === 'security' && <SecurityView onNavigate={handleNavigate} />}
          {currentRoute === 'analytics' && <AnalyticsView onNavigate={handleNavigate} />}
          {currentRoute === 'billing' && <BillingView onNavigate={handleNavigate} />}
          {currentRoute === 'team' && <TeamView onNavigate={handleNavigate} />}
          {currentRoute === 'settings' && <SettingsView onNavigate={handleNavigate} />}
          {currentRoute === 'copilot' && <CopilotView onNavigate={handleNavigate} />}
          {currentRoute === 'evidence' && <EvidenceView />}
        </main>
      </div>

      {/* Global Command Palette (Cmd + K) */}
      <CommandPalette
        isOpen={commandPaletteOpen}
        onClose={() => setCommandPaletteOpen(false)}
        onNavigate={handleNavigate}
      />

      {/* Notifications Drawer */}
      {notificationsOpen && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex justify-end">
          <div className="w-full max-w-md bg-[#0B1120] border-l border-slate-800 h-full p-6 flex flex-col justify-between animate-in slide-in-from-right duration-200">
            <div>
              <div className="flex items-center justify-between border-b border-slate-800 pb-3">
                <div className="flex items-center gap-2">
                  <Bell className="w-4 h-4 text-cyan-400" />
                  <h3 className="font-bold text-white text-sm">Recent Platform Alerts</h3>
                </div>
                <button
                  onClick={() => setNotificationsOpen(false)}
                  className="p-1 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="mt-4 space-y-3 font-mono text-xs overflow-y-auto max-h-[calc(100vh-140px)]">
                {recentEvents.map((evt) => (
                  <div key={evt.id} className="p-3 rounded-xl bg-slate-900 border border-slate-800 space-y-1">
                    <div className="flex justify-between text-slate-400 text-[10px]">
                      <span>{evt.type}</span>
                      <span>{evt.timestamp}</span>
                    </div>
                    <div className="text-white font-sans text-xs">{evt.message}</div>
                    <div className="text-slate-500 text-[10px]">Actor: {evt.actor}</div>
                  </div>
                ))}
              </div>
            </div>

            <div className="pt-3 border-t border-slate-800 flex justify-end">
              <button
                onClick={() => setNotificationsOpen(false)}
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
