import React from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
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
  Activity,
  ListChecks,
  ArrowRight,
  Hexagon
} from 'lucide-react';
import { CubeIllustration } from '../common/Brand';
import { useSession } from '../../lib/session';

const NAV: { to: string; label: string; icon: React.FC<{ className?: string }> }[] = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/apps', label: 'Applications', icon: AppWindow },
  { to: '/deploy', label: 'Deploy', icon: Boxes },
  { to: '/nodes', label: 'Nodes & Compute', icon: Server },
  { to: '/storage', label: 'Storage', icon: HardDrive },
  { to: '/domains', label: 'Domains', icon: Globe },
  { to: '/security', label: 'SSL & Security', icon: ShieldCheck },
  { to: '/analytics', label: 'Analytics', icon: BarChart3 },
  { to: '/billing', label: 'Billing', icon: Receipt },
  { to: '/team', label: 'Team', icon: Users },
  { to: '/copilot', label: 'RAG Copilot', icon: Bot },
  { to: '/evidence', label: 'Evidence', icon: FileCheck2 },
  { to: '/activity', label: 'Activity', icon: Activity },
  { to: '/operations', label: 'Operations', icon: ListChecks },
  { to: '/settings', label: 'Settings', icon: Settings }
];

/** Page-specific promo card at the foot of the sidebar. */
const HERO: Record<string, { title: string; sub?: string; cta: string; to: string }> = {
  deploy: { title: 'Deploy anywhere. Own the infrastructure.', cta: 'New Deployment', to: '/deploy/new' },
  storage: { title: 'Distributed Storage for a Sovereign Internet.', cta: 'Add Storage', to: '/storage?add=1' },
  nodes: { title: 'Turn your hardware into a global edge node.', cta: 'Add Your Machine', to: '/nodes/add' }
};
const DEFAULT_HERO = { title: 'Decentralized Hosting', sub: 'Distributed. Private. Resilient.', cta: 'Deploy an app', to: '/deploy/new' };

export const Sidebar: React.FC<{ mobile?: boolean; onClose?: () => void }> = ({ mobile, onClose }) => {
  const { capabilities } = useSession();
  const loc = useLocation();
  const navigate = useNavigate();
  const section = loc.pathname.split('/')[1] ?? '';
  const hero = HERO[section] ?? DEFAULT_HERO;
  const backend = capabilities?.backend;
  const state = !backend ? 'checking' : backend.reachable ? (capabilities?.mode === 'demo' ? 'demo' : 'ok') : 'down';

  return (
    <aside
      className={
        mobile
          ? 'fixed inset-y-0 left-0 z-50 flex w-[260px] flex-col gap-3 p-3 bg-[#030a1c]/95 backdrop-blur-xl border-r border-cyan-400/20 overflow-y-auto'
          : 'hidden lg:flex w-[228px] shrink-0 flex-col gap-3 sticky top-[76px] h-[calc(100vh-88px)] pl-4 pb-3 select-none z-20'
      }
      onClick={mobile ? (e) => e.stopPropagation() : undefined}
    >
      <nav className="dh-glass rounded-2xl p-2.5 flex flex-col gap-1 overflow-y-auto min-h-0 flex-1" aria-label="Primary">
        {NAV.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            onClick={onClose}
            className={({ isActive }) =>
              `flex items-center gap-3 w-full px-3.5 py-[7px] rounded-xl text-[14px] font-medium transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-300 ${
                isActive
                  ? 'text-white bg-[linear-gradient(90deg,#1f6dff_0%,#4f46e5_60%,#8b5cf6_100%)] shadow-[0_0_22px_rgba(59,130,246,0.55),inset_0_1px_0_rgba(255,255,255,0.25)] border border-cyan-300/50'
                  : 'text-slate-300 hover:text-white hover:bg-white/[0.05] border border-transparent'
              }`
            }
          >
            <Icon className="w-[19px] h-[19px]" />
            <span className="truncate">{label}</span>
          </NavLink>
        ))}
      </nav>

      <button onClick={() => navigate('/settings')} className="dh-glass rounded-2xl px-3.5 py-2.5 text-left hover:border-emerald-400/40 transition-colors">
        <div className="flex items-center gap-2.5">
          <Hexagon className={`w-6 h-6 ${state === 'down' ? 'text-rose-400 fill-rose-400/20' : state === 'demo' ? 'text-amber-400 fill-amber-400/20' : 'text-emerald-400 fill-emerald-400/20'}`} />
          <span className="text-[13px] font-semibold text-slate-100 whitespace-nowrap">Control Plane</span>
          <span
            className={`ml-auto inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[10.5px] font-semibold border ${
              state === 'down'
                ? 'text-rose-300 bg-rose-500/10 border-rose-400/30'
                : state === 'demo'
                  ? 'text-amber-300 bg-amber-500/10 border-amber-400/30'
                  : 'text-emerald-300 bg-emerald-500/10 border-emerald-400/30'
            }`}
          >
            <span className={`w-1.5 h-1.5 rounded-full ${state === 'down' ? 'bg-rose-400' : state === 'demo' ? 'bg-amber-400' : 'bg-emerald-400'}`} />
            {state === 'checking' ? 'Checking' : state === 'down' ? 'Unreachable' : state === 'demo' ? 'Simulated' : 'Reachable'}
          </span>
        </div>
        <div className={`mt-0.5 text-[11.5px] font-medium truncate ${state === 'down' ? 'text-rose-300' : 'text-slate-400'}`} title={backend?.detail}>
          {backend?.detail ?? 'Probing control plane…'}
        </div>
      </button>

      <div className="relative overflow-hidden rounded-2xl border border-cyan-400/25 bg-[linear-gradient(180deg,rgba(14,30,72,0.85),rgba(6,12,32,0.95))] p-3 pt-1 shadow-[0_0_30px_rgba(36,139,255,0.18)] shrink-0">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_30%,rgba(124,77,255,0.35),transparent_60%)] pointer-events-none" />
        <CubeIllustration className="relative w-full h-[84px]" />
        <div className="relative text-center">
          <div className="text-[14px] font-bold text-white leading-snug">{hero.title}</div>
          {hero.sub && <div className="text-[11px] text-slate-400 mt-0.5">{hero.sub}</div>}
          <button
            onClick={() => {
              onClose?.();
              navigate(hero.to);
            }}
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
