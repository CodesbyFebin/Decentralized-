import React, { useState } from 'react';
import { Activity, AlertTriangle, Gauge, TrendingUp, Zap, HardDrive, Network, DollarSign, BarChart3 } from 'lucide-react';
import type { Freshness, Metric, NodeRec, Overview } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, KpiTile, PanelHeader, PageTabs } from '../common/ui';
import { Gate, Unavailable, FreshnessPill, since, metricText, Note, fmtBytes } from '../common/states';

type Tab = 'overview' | 'resources' | 'apps' | 'efficiency' | 'failures';

interface Payload {
  edges: { id: string; name: string; requests: number; errors: number; routes: number; freshness: Freshness; observedAt: number | null }[];
  replicaHealth: { app: string; assignment: string; node: string; latencyUs: number; ok: boolean; checkedAt: number }[];
  hosts: { name: string; freshness: Freshness; storage: NodeRec['storage']; mesh: NodeRec['mesh']; workloads: number | null; declared: NodeRec['declared']; cpu?: { used: number; limit: number }; memory?: { used: number; limit: number } }[];
  metrics: Overview['metrics'] & Record<string, Metric>;
  appMetrics?: { app: string; replicas: number; desiredReplicas: number; recentDeployments: number; lastUpdate: number }[];
  nodeEfficiency?: { node: string; packRatio: number; spareCapacity: number; strain: number }[];
  failures?: { service: string; mtbf: number; meanRecoveryTime: number; lastFailure: number; rootCause?: string }[];
  costProjection?: { monthlyUsd: number; perReplica: number; perGb: number };
}

export default function AnalyticsView() {
  const res = useResource<Payload>('/analytics');
  const [tab, setTab] = useState<Tab>('overview');

  return (
    <div className="space-y-4 pt-2">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Analytics</h1>
        <p className="text-[13px] text-slate-400">Infrastructure telemetry that hosts actually report. Visitor analytics are a separate subsystem that does not exist yet.</p>
      </div>

      <PageTabs tabs={[
        { id: 'overview', label: 'Overview' },
        { id: 'resources', label: 'Resources' },
        { id: 'apps', label: 'Applications' },
        { id: 'efficiency', label: 'Node Efficiency' },
        { id: 'failures', label: 'Failures' }
      ] as { id: Tab; label: string }[]} active={tab} onChange={setTab} />

      <Gate res={res}>
        {(d) => {
          const lat = d.replicaHealth.map((r) => r.latencyUs);
          const max = Math.max(...lat, 1);

          if (tab === 'resources') {
            const totalCpu = d.hosts.reduce((a, h) => a + (h.cpu?.limit ?? 0), 0);
            const usedCpu = d.hosts.reduce((a, h) => a + (h.cpu?.used ?? 0), 0);
            const totalMem = d.hosts.reduce((a, h) => a + (h.memory?.limit ?? 0), 0);
            const usedMem = d.hosts.reduce((a, h) => a + (h.memory?.used ?? 0), 0);
            const totalStorage = d.hosts.reduce((a, h) => a + (h.storage?.quotaBytes ?? 0), 0);
            const usedStorage = d.hosts.reduce((a, h) => a + (h.storage?.usedBytes ?? 0), 0);

            return (
              <>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  <KpiTile tone="blue" icon={<Zap className="w-6 h-6" />} label="CPU (7d avg)" value={usedCpu ? `${Math.round((usedCpu/totalCpu)*100)}%` : '—'} sub={`${usedCpu}m / ${totalCpu}m`} />
                  <KpiTile tone="violet" icon={<Zap className="w-6 h-6" />} label="Memory (7d avg)" value={totalMem ? `${Math.round((usedMem/totalMem)*100)}%` : '—'} sub={totalMem ? `${fmtBytes(usedMem)} / ${fmtBytes(totalMem)}` : '—'} />
                  <KpiTile tone="emerald" icon={<HardDrive className="w-6 h-6" />} label="Storage (7d avg)" value={totalStorage ? `${Math.round((usedStorage/totalStorage)*100)}%` : '—'} sub={totalStorage ? `${fmtBytes(usedStorage)} / ${fmtBytes(totalStorage)}` : '—'} />
                  <KpiTile tone="cyan" icon={<Network className="w-6 h-6" />} label="Network (7d total)" value={metricText(d.metrics.networkBytes)} />
                </div>
                <Glass className="p-4">
                  <PanelHeader title="Resource usage by host" />
                  <div className="mt-3 space-y-3">
                    {d.hosts.map((h) => (
                      <div key={h.name}>
                        <div className="flex justify-between text-[12px] mb-1">
                          <span className="text-slate-200">{h.name}</span>
                          <span className="text-slate-400">{h.workloads ?? 0} workloads</span>
                        </div>
                        {h.cpu && <div className="flex gap-2 items-center text-[11px]">
                          <span className="text-slate-400 w-6">CPU</span>
                          <div className="flex-1 h-1.5 rounded bg-white/5"><div className="h-full rounded bg-blue-400" style={{ width: `${Math.min(100, (h.cpu.used/h.cpu.limit)*100)}%` }} /></div>
                          <span className="text-slate-300 tabular-nums">{Math.round((h.cpu.used/h.cpu.limit)*100)}%</span>
                        </div>}
                        {h.memory && <div className="flex gap-2 items-center text-[11px]">
                          <span className="text-slate-400 w-6">Mem</span>
                          <div className="flex-1 h-1.5 rounded bg-white/5"><div className="h-full rounded bg-violet-400" style={{ width: `${Math.min(100, (h.memory.used/h.memory.limit)*100)}%` }} /></div>
                          <span className="text-slate-300 tabular-nums">{Math.round((h.memory.used/h.memory.limit)*100)}%</span>
                        </div>}
                      </div>
                    ))}
                  </div>
                </Glass>
                {d.costProjection && <Glass className="p-4"><PanelHeader title="Cost Projection" /><div className="mt-3 space-y-1 text-[12px]"><div className="flex justify-between"><span className="text-slate-400">Monthly estimate</span><span className="text-slate-100">${d.costProjection.monthlyUsd.toFixed(2)}</span></div><div className="flex justify-between"><span className="text-slate-400">Per replica</span><span className="text-slate-100">${d.costProjection.perReplica.toFixed(4)}</span></div><div className="flex justify-between"><span className="text-slate-400">Per GB</span><span className="text-slate-100">${d.costProjection.perGb.toFixed(4)}</span></div></div></Glass>}
                <Note>Resource usage is a snapshot, not a time series. Trend lines require persistent metrics storage.</Note>
              </>
            );
          }

          if (tab === 'apps' && d.appMetrics) {
            return (
              <>
                <Glass className="overflow-x-auto">
                  <table className="dh-table w-full min-w-[700px]">
                    <thead><tr><th>Application</th><th>Replicas</th><th>Desired</th><th>Recent deployments</th><th>Last update</th></tr></thead>
                    <tbody>
                      {d.appMetrics.map((a) => (
                        <tr key={a.app}>
                          <td className="text-slate-100 font-semibold">{a.app}</td>
                          <td className={`tabular-nums ${a.replicas === a.desiredReplicas ? 'text-emerald-300' : 'text-amber-300'}`}>{a.replicas}</td>
                          <td className="tabular-nums text-slate-400">{a.desiredReplicas}</td>
                          <td className="tabular-nums text-slate-300">{a.recentDeployments}</td>
                          <td className="text-slate-300">{since(a.lastUpdate)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </Glass>
                <Note>Deployment frequency and update lag indicate deployment velocity.</Note>
              </>
            );
          }

          if (tab === 'efficiency' && d.nodeEfficiency) {
            return (
              <>
                <div className="grid grid-cols-3 gap-3">
                  <KpiTile tone="blue" icon={<BarChart3 className="w-6 h-6" />} label="Avg pack ratio" value={`${Math.round(d.nodeEfficiency.reduce((a, n) => a + n.packRatio, 0) / d.nodeEfficiency.length * 100)}%`} sub="utilization vs capacity" />
                  <KpiTile tone="emerald" icon={<TrendingUp className="w-6 h-6" />} label="Avg spare capacity" value={`${Math.round(d.nodeEfficiency.reduce((a, n) => a + n.spareCapacity, 0) / d.nodeEfficiency.length * 100)}%`} />
                  <KpiTile tone="amber" icon={<AlertTriangle className="w-6 h-6" />} label="Avg strain" value={`${Math.round(d.nodeEfficiency.reduce((a, n) => a + n.strain, 0) / d.nodeEfficiency.length * 100)}%`} sub="rescheduling pressure" />
                </div>
                <Glass className="p-4">
                  <PanelHeader title="Node efficiency metrics" />
                  <div className="mt-3 space-y-2">
                    {d.nodeEfficiency.map((n) => (
                      <div key={n.node} className="text-[12px] flex justify-between items-center p-2 rounded bg-white/3">
                        <span className="text-slate-200">{n.node}</span>
                        <div className="flex gap-4 text-slate-400">
                          <span>Pack: {Math.round(n.packRatio * 100)}%</span>
                          <span>Spare: {Math.round(n.spareCapacity * 100)}%</span>
                          <span className={n.strain > 0.8 ? 'text-amber-300' : 'text-emerald-300'}>Strain: {Math.round(n.strain * 100)}%</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </Glass>
              </>
            );
          }

          if (tab === 'failures' && d.failures) {
            return (
              <>
                <Glass className="p-4">
                  <PanelHeader title="Failure tracking" subtitle="Mean Time Between Failures and recovery times." />
                  <div className="mt-3 space-y-3">
                    {d.failures.map((f) => (
                      <div key={f.service} className="p-3 rounded-lg bg-white/3 border border-white/5">
                        <div className="text-[12.5px] font-semibold text-slate-100 mb-2">{f.service}</div>
                        <div className="grid grid-cols-3 gap-4 text-[12px]">
                          <div><span className="text-slate-400">MTBF</span><div className="text-slate-100">{Math.round(f.mtbf / 24)} days</div></div>
                          <div><span className="text-slate-400">Mean recovery</span><div className="text-slate-100">{Math.round(f.meanRecoveryTime / 60)} min</div></div>
                          <div><span className="text-slate-400">Last failure</span><div className="text-slate-100">{since(f.lastFailure)}</div></div>
                        </div>
                        {f.rootCause && <div className="mt-2 text-[11px] text-slate-400">Root cause: {f.rootCause}</div>}
                      </div>
                    ))}
                  </div>
                </Glass>
              </>
            );
          }

          return (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <KpiTile tone="cyan" icon={<Activity className="w-6 h-6" />} label="Edge requests" value={metricText(d.metrics.edgeRequests)} sub="since each edge process started" />
                <KpiTile tone="rose" icon={<AlertTriangle className="w-6 h-6" />} label="Edge errors" value={metricText(d.metrics.edgeErrors)} />
                <KpiTile tone="emerald" icon={<Gauge className="w-6 h-6" />} label="Health checks passing" value={`${d.replicaHealth.filter((r) => r.ok).length} / ${d.replicaHealth.length}`} />
                <KpiTile tone="violet" icon={<Gauge className="w-6 h-6" />} label="Median probe latency" value={lat.length ? `${Math.round([...lat].sort((a, b) => a - b)[Math.floor(lat.length / 2)] / 1000 * 10) / 10} ms` : '—'} />
              </div>
              <div className="grid lg:grid-cols-2 gap-4">
                <Glass className="p-4">
                  <PanelHeader title="Health-check latency by replica" subtitle="Latest probe per replica (not a time series)." />
                  <ul className="mt-3 space-y-2">
                    {d.replicaHealth.map((r) => (
                      <li key={r.assignment + r.node}>
                        <div className="flex justify-between text-[12px]"><span className="text-slate-200">{r.assignment} <span className="text-slate-500">on {r.node}</span></span><span className={r.ok ? 'text-emerald-300' : 'text-rose-300'}>{(r.latencyUs / 1000).toFixed(2)} ms · {since(r.checkedAt)}</span></div>
                        <div className="h-1.5 mt-1 rounded-full bg-white/[0.05]"><div className={`h-full rounded-full ${r.ok ? 'bg-cyan-400' : 'bg-rose-400'}`} style={{ width: `${Math.max(2, (r.latencyUs / max) * 100)}%` }} /></div>
                      </li>
                    ))}
                    {!d.replicaHealth.length && <li className="text-[12.5px] text-slate-400">No health checks configured.</li>}
                  </ul>
                </Glass>
                <Glass className="p-4">
                  <PanelHeader title="Edge hosts" />
                  <ul className="mt-3 space-y-2 text-[12.5px]">
                    {d.edges.map((e) => (
                      <li key={e.id} className="flex items-center justify-between gap-2">
                        <span className="text-slate-100">{e.name}</span>
                        <span className="text-slate-400 tabular-nums">{e.requests} req · {e.errors} err · {e.routes} routes</span>
                        <FreshnessPill f={e.freshness} />
                      </li>
                    ))}
                    {!d.edges.length && <li className="text-slate-400">No edge host is reporting.</li>}
                  </ul>
                </Glass>
              </div>
              <div className="grid md:grid-cols-2 gap-3">
                <Unavailable title="Visitor analytics" detail={d.metrics.visitors30d.detail ?? 'not measured'} />
                <Unavailable title="Bandwidth" detail={d.metrics.bandwidth.detail ?? 'not measured'} />
              </div>
              <Note>No metric history is stored, so nothing here is drawn as a trend line.</Note>
            </>
          );
        }}
      </Gate>
    </div>
  );
}
