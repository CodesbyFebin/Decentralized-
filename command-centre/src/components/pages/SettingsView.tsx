import React from 'react';
import { Settings, Activity, LogOut } from 'lucide-react';
import type { CapabilityKey } from '../../types/reality';
import { useSession } from '../../lib/session';
import { useResource } from '../../lib/useResource';
import { Glass, PanelHeader, IconTile, GhostButton } from '../common/ui';
import { Gate, TruthTag, fmtTime } from '../common/states';

interface Metrics {
  backend: { count: number; errors: number; avgMs: number; maxMs: number };
  routes: Record<string, { count: number; errors: number; avgMs: number; maxMs: number }>;
}

export default function SettingsView() {
  const { capabilities, mode, session, signOut } = useSession();
  const metrics = useResource<Metrics>('/bff/metrics', { pollMs: 10_000 });
  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Settings</h1>
        <p className="text-[13px] text-slate-400">What this console is connected to and what it can honestly show. Configuration is set with environment variables on the server, not from the browser.</p>
      </div>
      <Glass className="p-5">
        <PanelHeader icon={<IconTile tone="cyan" size="sm"><Settings className="w-4 h-4" /></IconTile>} title="Connection" />
        <div className="mt-3 grid sm:grid-cols-2 gap-x-6 gap-y-1.5 text-[12.5px]">
          <div className="flex justify-between"><span className="text-slate-400">Adapter</span><span className="text-slate-100">{mode === 'demo' ? 'demo replay (SIMULATED)' : 'control plane'}</span></div>
          <div className="flex justify-between"><span className="text-slate-400">Backend</span><span className={capabilities?.backend.reachable ? 'text-emerald-300' : 'text-rose-300'}>{capabilities?.backend.reachable ? 'reachable' : 'unreachable'}</span></div>
          <div className="flex justify-between gap-3"><span className="text-slate-400">Detail</span><span className="text-slate-100 text-right">{capabilities?.backend.detail ?? '—'}</span></div>
          <div className="flex justify-between"><span className="text-slate-400">Leader</span><span className="text-slate-100 font-mono">{capabilities?.backend.leader?.slice(0, 16) ?? '—'}</span></div>
          <div className="flex justify-between"><span className="text-slate-400">Checked</span><span className="text-slate-100">{fmtTime(capabilities?.backend.checkedAt)}</span></div>
        </div>
        {session?.authenticated && <GhostButton onClick={signOut} className="mt-4 !border-rose-400/40 !text-rose-200"><LogOut className="w-4 h-4" /> Sign out</GhostButton>}
      </Glass>

      <Glass className="overflow-x-auto">
        <div className="px-4 pt-4"><PanelHeader title="Capability matrix" subtitle="Discovered from the connected backend on every refresh." /></div>
        <table className="dh-table w-full min-w-[720px] mt-2">
          <thead><tr><th>Capability</th><th>State</th><th>Source</th><th>Detail</th></tr></thead>
          <tbody>
            {capabilities &&
              (Object.keys(capabilities.items) as CapabilityKey[]).map((k) => (
                <tr key={k}>
                  <td className="text-slate-100 font-mono text-[12px]">{k}</td>
                  <td><TruthTag state={capabilities.items[k].state} /></td>
                  <td className="text-slate-400">{capabilities.items[k].source}</td>
                  <td className="text-slate-300 whitespace-normal">{capabilities.items[k].detail}</td>
                </tr>
              ))}
          </tbody>
        </table>
      </Glass>

      <Glass className="p-4">
        <PanelHeader icon={<IconTile tone="violet" size="sm"><Activity className="w-4 h-4" /></IconTile>} title="BFF observability" subtitle="In-process counters since the server started. Nothing is sent anywhere." />
        <Gate res={metrics}>
          {(m) => (
            <div className="mt-3">
              <div className="text-[12.5px] text-slate-300">Control-plane calls: {m.backend.count} · errors {m.backend.errors} · avg {m.backend.avgMs} ms · max {m.backend.maxMs} ms</div>
              <table className="dh-table w-full mt-2">
                <thead><tr><th>Route</th><th>Requests</th><th>5xx</th><th>Avg ms</th><th>Max ms</th></tr></thead>
                <tbody>
                  {Object.entries(m.routes).sort((a, b) => b[1].count - a[1].count).map(([k, v]) => (
                    <tr key={k}><td className="font-mono text-[11.5px]">{k}</td><td>{v.count}</td><td>{v.errors}</td><td>{v.avgMs}</td><td>{v.maxMs}</td></tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Gate>
      </Glass>
    </div>
  );
}
