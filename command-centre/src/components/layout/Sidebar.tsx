import React, { useEffect, useState } from 'react';
import {
  LayoutDashboard,
  AppWindow,
  Boxes,
  Server,
  HardDrive,
  Globe,
  ShieldCheck,
  BarChart3,
  Receipt,
  Users,
  Bot,
  Settings,
  FileCheck2,
  ArrowRight,
  Hexagon
} from 'lucide-react';
import { CubeIllustration } from '../common/Brand';
import { api } from '../../lib/api';

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
  onRouteChange: (route: NavRoute, entityId?: string) => void;
  /** Render as an off-canvas drawer (below the lg breakpoint). */
  mobile?: boolean;
  onClose?: () => void;
}

const NAV: { id: NavRoute; label: string; icon: React.FC<{ className?: string }> }[] = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'apps', label: 'Websites & Apps', icon: AppWindow },
  { id: 'deploy', label: 'Deploy', icon: Boxes },
  { id: 'nodes', label: 'Nodes & Compute', icon: Server },
  { id: 'storage', label: 'Storage', icon: HardDrive },
  { id: 'domains', label: 'Domains', icon: Globe },
  { id: 'security', label: 'SSL & Security', icon: ShieldCheck },
  { id: 'analytics', label: 'Analytics', icon: BarChart3 },
  { id: 'billing', label: 'Billing', icon: Receipt },
  { id: 'team', label: 'Team', icon: Users },
  { id: 'copilot', label: 'RAG Copilot', icon: Bot },
  { id: 'evidence', label: 'Evidence Ledger', icon: FileCheck2 },
  { id: 'settings', label: 'Settings', icon: Settings }
];

/** Page-specific promo card at the foot of the sidebar, as in the reference screens. */
const HERO: Partial<Record<NavRoute, { title: string; sub?: string; cta: string; to: [NavRoute, string?] }>> = {
  deploy: { title: 'Deploy anywhere. Own the infrastructure.', cta: 'New Deployment', to: ['deploy', 'new'] },
  storage: { title: 'Distributed Storage for a Sovereign Internet.', cta: 'Add Storage', to: ['storage', 'add'] },
  nodes: { title: 'Turn your hardware into a global edge node.', cta: 'Add Your Machine', to: ['nodes', 'add'] }
};
const DEFAULT_HERO = { title: 'Decentralized Hosting', sub: 'Distributed. Private. Resilient.', cta: 'Upgrade Plan', to: ['billing'] as [NavRoute] };

export const Sidebar: React.FC<Props> = ({ currentRoute, onRouteChange, mobile, onClose }) => {
  const [health, setHealth] = useState<{ healthy: boolean; mode: string } | null>(null);

  useEffect(() => {
    api
      .getHealth()
      .then((r) => setHealth({ healthy: r.health.healthy, mode: r.health.mode }))
      .catch(() => setHealth({ healthy: false, mode: 'unreachable' }));
  }, []);

  const hero = HERO[currentRoute] ?? DEFAULT_HERO;
  const healthy = health?.healthy;

  return (
    <aside
      className={
        mobile
          ? 'fixed inset-y-0 left-0 z-50 flex w-[260px] flex-col gap-3 p-3 bg-[#030a1c]/95 backdrop-blur-xl border-r border-cyan-400/20 overflow-y-auto'
          : 'hidden lg:flex w-[228px] shrink-0 flex-col gap-3 sticky top-[76px] h-[calc(100vh-88px)] pl-4 pb-3 select-none z-20'
      }
      onClick={mobile ? (e) => e.target === e.currentTarget && onClose?.() : undefined}
    >
      <nav className="dh-glass rounded-2xl p-2.5 flex flex-col gap-1 overflow-y-auto min-h-0 flex-1" aria-label="Primary">
        {NAV.map(({ id, label, icon: Icon }) => {
          const active = currentRoute === id;
          return (
            <button
              key={id}
              onClick={() => onRouteChange(id)}
              aria-current={active ? 'page' : undefined}
              className={`flex items-center gap-3 w-full px-3.5 py-[7px] rounded-xl text-[14px] font-medium transition-all ${
                active
                  ? 'text-white bg-[linear-gradient(90deg,#1f6dff_0%,#4f46e5_60%,#8b5cf6_100%)] shadow-[0_0_22px_rgba(59,130,246,0.55),inset_0_1px_0_rgba(255,255,255,0.25)] border border-cyan-300/50'
                  : 'text-slate-300 hover:text-white hover:bg-white/[0.05] border border-transparent'
              }`}
            >
              <Icon className={`w-[19px] h-[19px] ${active ? 'text-white' : 'text-slate-300'}`} />
              <span className="truncate">{label}</span>
            </button>
          );
        })}
      </nav>

      <button
        onClick={() => onRouteChange('nodes')}
        className="dh-glass rounded-2xl px-3.5 py-2.5 text-left hover:border-emerald-400/40 transition-colors"
      >
        <div className="flex items-center gap-2.5">
          <Hexagon
            className={`w-6 h-6 ${healthy === false ? 'text-rose-400 fill-rose-400/20' : 'text-emerald-400 fill-emerald-400/20'} drop-shadow-[0_0_6px_currentColor]`}
          />
          <span className="text-[13px] font-semibold text-slate-100 whitespace-nowrap">Control Plane</span>
          <span
            className={`ml-auto inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[10.5px] font-semibold border ${
              healthy === false
                ? 'text-rose-300 bg-rose-500/10 border-rose-400/30'
                : 'text-emerald-300 bg-emerald-500/10 border-emerald-400/30'
            }`}
          >
            <span className={`w-1.5 h-1.5 rounded-full ${healthy === false ? 'bg-rose-400' : 'bg-emerald-400 animate-pulse'}`} />
            {health === null ? 'Checking' : healthy ? 'Healthy' : 'Unreachable'}
          </span>
        </div>
        <div className={`mt-1 text-[11.5px] font-medium ${healthy === false ? 'text-rose-300' : 'text-emerald-400'}`}>
          {health === null ? 'Probing control plane…' : healthy ? 'All Systems Operational' : 'Control plane not responding'}
        </div>
        {health?.mode === 'demo' && <div className="mt-0.5 text-[10px] text-amber-300/80">Demo platform store</div>}
      </button>

      <div className="relative overflow-hidden rounded-2xl border border-cyan-400/25 bg-[linear-gradient(180deg,rgba(14,30,72,0.85),rgba(6,12,32,0.95))] p-3 pt-1 shadow-[0_0_30px_rgba(36,139,255,0.18)] shrink-0">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_30%,rgba(124,77,255,0.35),transparent_60%)] pointer-events-none" />
        <CubeIllustration className="relative w-full h-[84px]" />
        <div className="relative text-center">
          <div className="text-[14px] font-bold text-white leading-snug">{hero.title}</div>
          {'sub' in hero && hero.sub && <div className="text-[11px] text-slate-400 mt-0.5">{hero.sub}</div>}
          <button
            onClick={() => onRouteChange(hero.to[0], hero.to[1])}
            className="mt-2.5 w-full py-2.5 rounded-xl bg-btn-spatial text-white text-[13.5px] font-semibold shadow-[0_0_20px_rgba(36,139,255,0.5)] border border-white/20 inline-flex items-center justify-center gap-1.5"
          >
            {hero.cta}
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    </aside>
  );
};
