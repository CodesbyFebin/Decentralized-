import React from 'react';
import { Activity, AlertTriangle, Gauge } from 'lucide-react';
import type { Freshness, Metric, NodeRec, Overview } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, KpiTile, PanelHeader } from '../common/ui';
import { Gate, Unavailable, FreshnessPill, since, metricText, Note } from '../common/states';

interface Payload {
  edges: { id: string; name: string; requests: number; errors: number; routes: number; freshness: Freshness; observedAt: number | null }[];
  replicaHealth: { app: string; assignment: string; node: string; latencyUs: number; ok: boolean; checkedAt: number }[];
  hosts: { name: string; freshness: Freshness; storage: NodeRec['storage']; mesh: NodeRec['mesh']; workloads: number | null; declared: NodeRec['declared'] }[];
  metrics: Overview['metrics'] & Record<string, Metric>;
}

export default function AnalyticsView() {
  const res = useResource<Payload>('/analytics');
  return (
    <div className="space-y-4 pt-2">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Analytics</h1>
        <p className="text-[13px] text-slate-400">Infrastructure telemetry that hosts actually report. Visitor analytics are a separate subsystem that does not exist yet.</p>
      </div>
      <Gate res={res}>
        {(d) => {
          const lat = d.replicaHealth.map((r) => r.latencyUs);
          const max = Math.max(...lat, 1);
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
