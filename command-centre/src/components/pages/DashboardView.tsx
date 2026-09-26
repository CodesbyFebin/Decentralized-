import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Rocket, Globe, Server, FileText, Bot, AppWindow, HardDrive, ShieldCheck, Lock, Activity, ArrowRight, Database, Radio, FileCheck2, Cpu } from 'lucide-react';
import type { AppRec, AuditEntryRec, CertRec, DomainRec, LedgerVerification, NodeRec, Overview, SecurityControl } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { HoloGlobe } from '../common/HoloGlobe';
import { regionGroups, groupMarkers, meshArcs, NoGeoNote } from '../common/CommandSurface';
import { Glass, KpiTile, PanelHeader, IconTile, StatusPill, RingGauge, Legend, GhostButton, Grad } from '../common/ui';
import { Gate, fmtBytes, since, metricText, TruthTag } from '../common/states';

interface Payload {
  overview: Overview;
  nodes: NodeRec[];
  apps: (Omit<AppRec, 'replicas' | 'history'> & { replicaCount: number; lastChange: AppRec['history'][number] | null })[];
  domains: DomainRec[];
  certificates: CertRec[];
  security: SecurityControl[];
  verification: LedgerVerification;
  recentEvents: AuditEntryRec[];
}

export default function DashboardView() {
  const res = useResource<Payload>('/overview');
  const { session, mode } = useSession();
  const navigate = useNavigate();
  const who = session?.actor?.replace(/^operator:?/, '') || (mode === 'demo' ? 'demo viewer' : 'operator');

  return (
    <Gate res={res} loadingRows={4}>
      {(d, stale) => {
        const m = d.overview.metrics;
        const groups = regionGroups(d.nodes);
        const pct = (a: number | null, b: number | null) => (a !== null && b ? Math.round((a / b) * 100) : null);
        const storagePct = pct(m.storageUsedBytes.value, m.storageCapacityBytes.value);
        const measuredCpus = d.nodes.filter((n) => n.facts).reduce((a, n) => a + n.facts!.cpus, 0);
        const validCerts = d.certificates.filter((c) => c.state === 'VALID').length;
        return (
          <div className="space-y-4 pt-1">
            {/* Hero */}
            <section className="relative overflow-hidden rounded-3xl dh-glass p-6 lg:p-7 min-h-[230px]">
              <HoloGlobe
                markers={groupMarkers(groups.placed, (g) => [`${g.nodes.length} node${g.nodes.length > 1 ? 's' : ''}`], (g) => navigate(`/nodes/${g.nodes[0].id}`))}
                arcs={meshArcs(groups.placed)}
                focusLng={groups.placed[0]?.lng ?? 20}
                tilt={24}
                speed={2}
                frozen={stale}
                center={[0.72, 1.05]}
                radius={0.95}
                resolution={1.3}
                className="!absolute inset-y-0 right-0 w-full md:w-[60%] opacity-40 md:opacity-100"
              />
              <div className="relative max-w-[640px]">
                <div className="text-[13px] text-slate-400">{new Date().toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric', year: 'numeric' })}</div>
                <h1 className="mt-1 text-[34px] sm:text-[40px] font-extrabold tracking-tight text-white leading-tight">
                  Welcome back, <Grad>{who}</Grad>
                </h1>
                <p className="mt-1 text-[15px] text-slate-300">
                  {stale
                    ? 'The control plane is unreachable; this is the last observation.'
                    : d.overview.incidents.length
                      ? `${d.overview.incidents.length} open incident(s) need attention.`
                      : m.nodesHealthy.value === m.nodesTotal.value && m.drift.value === 0
                        ? 'Every host is reporting fresh observations and every desired replica is observed.'
                        : `${m.nodesHealthy.value}/${m.nodesTotal.value} hosts fresh · drift ${metricText(m.drift)}.`}
                </p>
                <div className="mt-5 flex flex-wrap gap-2">
                  {[
                    ['Deploy App', '/deploy/new', <Rocket key="r" className="w-4 h-4" />],
                    ['Domains', '/domains', <Globe key="g" className="w-4 h-4" />],
                    ['Manage Nodes', '/nodes', <Server key="s" className="w-4 h-4" />],
                    ['Audit Ledger', '/evidence', <FileText key="f" className="w-4 h-4" />]
                  ].map(([l, to, icon]) => (
                    <GhostButton key={to as string} onClick={() => navigate(to as string)} className="!py-2">
                      {icon} {l}
                    </GhostButton>
                  ))}
                  <GhostButton onClick={() => navigate('/copilot')} className="!py-2 !border-cyan-400/50 !text-cyan-100">
                    <Bot className="w-4 h-4" /> Ask Copilot <ArrowRight className="w-3.5 h-3.5" />
                  </GhostButton>
                </div>
              </div>
              <div className="absolute right-4 top-4 hidden md:block">
                <Glass strong className="px-4 py-3">
                  <div className="text-[12px] text-slate-300">Hosts reporting fresh</div>
                  <div className="text-[22px] font-bold text-white tabular-nums">
                    <span className={m.nodesHealthy.value === m.nodesTotal.value ? 'text-emerald-400' : 'text-amber-300'}>{metricText(m.nodesHealthy)}</span> / {metricText(m.nodesTotal)}
                  </div>
                  <Link to="/nodes" className="text-[12px] text-cyan-300 hover:underline">View nodes →</Link>
                </Glass>
              </div>
            </section>

            {/* KPIs */}
            <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-5 gap-3">
              <KpiTile tone="cyan" icon={<AppWindow className="w-6 h-6" />} label="Applications" value={metricText(m.applications)} sub={`${metricText(m.replicasObserved)}/${metricText(m.replicasDesired)} replicas observed`} onClick={() => navigate('/apps')} />
              <KpiTile tone="emerald" icon={<Server className="w-6 h-6" />} label="Nodes fresh" value={`${metricText(m.nodesHealthy)} / ${metricText(m.nodesTotal)}`} sub={`${metricText(m.nodesDegraded)} stale · ${metricText(m.nodesOffline)} offline`} onClick={() => navigate('/nodes')} />
              <KpiTile tone="violet" icon={<Database className="w-6 h-6" />} label="Storage used" value={fmtBytes(m.storageUsedBytes.value)} sub={m.storageCapacityBytes.value ? `of ${fmtBytes(m.storageCapacityBytes.value)} quota` : m.storageUsedBytes.detail} onClick={() => navigate('/storage')} />
              <KpiTile tone="amber" icon={<Globe className="w-6 h-6" />} label="Domains" value={metricText(m.domains)} sub={`${d.domains.filter((x) => x.routing.state === 'LIVE').length} routing`} onClick={() => navigate('/domains')} />
              <KpiTile tone="blue" icon={<Lock className="w-6 h-6" />} label="TLS certificates" value={metricText(m.certificates)} sub={m.certificates.value === null ? m.certificates.detail : `${validCerts} valid (observed at edge)`} onClick={() => navigate('/security')} />
            </div>

            <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,0.85fr)] gap-4">
              <Glass className="p-4 overflow-hidden">
                <PanelHeader icon={<IconTile tone="cyan" size="sm"><Radio className="w-4 h-4" /></IconTile>} title="Node Network" subtitle="Hosts by region; arcs join regions with a fresh host." right={<Link to="/nodes" className="text-[12px] text-cyan-300 hover:underline">View all nodes →</Link>} />
                <div className="relative">
                  <HoloGlobe markers={groupMarkers(groups.placed, (g) => g.nodes.map((n) => `${n.name}: ${n.health.toLowerCase()}`))} arcs={meshArcs(groups.placed)} focusLng={groups.placed[0]?.lng ?? 30} tilt={22} speed={1.4} frozen={stale} center={[0.42, 0.62]} radius={0.62} resolution={1.3} safeRight={130} className="h-[280px] -mx-4" />
                  <div className="absolute right-0 top-3 w-[130px]">
                    <Legend items={[{ label: 'Healthy', value: metricText(m.nodesHealthy), tone: 'emerald' }, { label: 'Stale', value: metricText(m.nodesDegraded), tone: 'amber' }, { label: 'Offline', value: metricText(m.nodesOffline), tone: 'rose' }, { label: 'Unknown', value: metricText(m.nodesUnknown), tone: 'slate' }]} />
                  </div>
                </div>
                <NoGeoNote unplaced={groups.unplaced} />
              </Glass>

              <Glass className="p-4">
                <PanelHeader title="Resource Usage" subtitle="Point-in-time host observations; no time-series store exists." right={<TruthTag state={stale ? 'UNKNOWN' : 'DERIVED'} />} />
                <div className="mt-4 grid grid-cols-3 gap-3 text-center">
                  <div className="flex flex-col items-center gap-1.5">
                    <RingGauge value={storagePct ?? 0} tone="violet" size={78} stroke={7} label={<span className="text-[15px] font-bold text-white">{storagePct === null ? '—' : `${storagePct}%`}</span>} />
                    <span className="text-[12px] text-slate-300">Storage quota</span>
                    <span className="text-[10.5px] text-slate-500">{fmtBytes(m.storageUsedBytes.value)} / {fmtBytes(m.storageCapacityBytes.value)}</span>
                  </div>
                  <div className="flex flex-col items-center gap-1.5">
                    <RingGauge value={m.replicasDesired.value ? ((m.replicasObserved.value ?? 0) / m.replicasDesired.value) * 100 : 0} tone="emerald" size={78} stroke={7} label={<span className="text-[15px] font-bold text-white">{metricText(m.replicasObserved)}/{metricText(m.replicasDesired)}</span>} />
                    <span className="text-[12px] text-slate-300">Replicas observed</span>
                    <span className="text-[10.5px] text-slate-500">drift {metricText(m.drift)}</span>
                  </div>
                  <div className="flex flex-col items-center gap-1.5">
                    <RingGauge value={100} tone="cyan" size={78} stroke={7} label={<span className="text-[15px] font-bold text-white">{measuredCpus || '—'}</span>} />
                    <span className="text-[12px] text-slate-300">CPUs measured</span>
                    <span className="text-[10.5px] text-slate-500">{(m.cpuDeclaredMilli.value ?? 0) / 1000} declared</span>
                  </div>
                </div>
                <div className="mt-4 rounded-xl bg-white/[0.03] border border-dashed border-[rgba(125,190,255,0.14)] p-3 text-[12px] text-slate-400 flex items-start gap-2">
                  <Cpu className="w-4 h-4 mt-0.5 shrink-0" />
                  CPU and memory utilisation over time is not collected by host agents, so no chart is drawn. Edge counters: {metricText(m.edgeRequests)} requests, {metricText(m.edgeErrors)} errors since edge start.
                </div>
              </Glass>

              <Glass className="p-4 flex flex-col">
                <PanelHeader icon={<IconTile tone="cyan" size="sm"><Bot className="w-4 h-4" /></IconTile>} title="RAG Copilot" subtitle="Answers grounded in what you can read, with citations." />
                <div className="mt-3 space-y-2 flex-1">
                  {[
                    ['Ask a question', 'How is my cluster doing?'],
                    ['Diagnose an issue', "What's wrong right now?"],
                    ['Plan an improvement', 'Show certificates and audit status'],
                    ['Execute an action', 'drain <host> — with your approval']
                  ].map(([t, qx]) => (
                    <button key={t} onClick={() => navigate(`/copilot?q=${encodeURIComponent(qx.includes('<') ? '' : qx)}`)} className="w-full text-left p-2.5 rounded-xl bg-white/[0.025] border border-[rgba(125,190,255,0.09)] hover:border-cyan-400/40">
                      <div className="text-[12.5px] font-semibold text-slate-100">{t}</div>
                      <div className="text-[11px] text-slate-400">{qx}</div>
                    </button>
                  ))}
                </div>
                <button onClick={() => navigate('/copilot')} className="mt-3 w-full py-2.5 rounded-xl bg-btn-spatial text-white text-[13.5px] font-semibold inline-flex items-center justify-center gap-1.5">
                  Start a conversation <ArrowRight className="w-4 h-4" />
                </button>
              </Glass>
            </div>

            <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
              <Glass className="p-4">
                <PanelHeader title="Recent Activity" subtitle="Hash-chained audit ledger" right={<Link to="/evidence" className="text-[12px] text-cyan-300 hover:underline">View all →</Link>} />
                <ul className="mt-3 space-y-2">
                  {d.recentEvents.map((e) => (
                    <li key={e.seq} className="flex items-start gap-2.5">
                      <IconTile tone={e.source === 'host' ? 'emerald' : e.source === 'operator' ? 'violet' : 'blue'} size="sm"><Activity className="w-4 h-4" /></IconTile>
                      <div className="min-w-0 flex-1">
                        <div className="text-[12.5px] text-slate-100 truncate">{e.action} <span className="text-slate-400">{e.resource}</span></div>
                        <div className="text-[11px] text-slate-500 truncate">{e.actor}</div>
                      </div>
                      <span className="text-[11px] text-slate-500 whitespace-nowrap">{since(e.ts)}</span>
                    </li>
                  ))}
                </ul>
              </Glass>
              <Glass className="p-4">
                <PanelHeader title="Applications" right={<Link to="/apps" className="text-[12px] text-cyan-300 hover:underline">View all →</Link>} />
                <table className="dh-table w-full mt-2">
                  <thead><tr><th>Name</th><th>Phase</th><th>Replicas</th><th>Changed</th></tr></thead>
                  <tbody>
                    {d.apps.slice(0, 6).map((a) => (
                      <tr key={a.name}>
                        <td><Link to={`/apps/${encodeURIComponent(a.name)}`} className="text-slate-100 hover:text-cyan-300 font-semibold">{a.name}</Link></td>
                        <td><StatusPill status={a.phase} /></td>
                        <td className="tabular-nums">{a.observedRunning}/{a.desiredReplicas}</td>
                        <td className="text-slate-400">{since(a.lastChange?.ts)}</td>
                      </tr>
                    ))}
                    {!d.apps.length && <tr><td colSpan={4} className="text-slate-500 text-center py-4">No applications</td></tr>}
                  </tbody>
                </table>
                <p className="mt-2 text-[11px] text-slate-500">Visitor traffic: {m.visitors30d.detail}.</p>
              </Glass>
              <Glass className="p-4">
                <PanelHeader title="System Health" right={<Link to="/security" className="text-[12px] text-cyan-300 hover:underline">View all →</Link>} />
                <ul className="mt-3 space-y-2.5 text-[12.5px]">
                  <li className="flex items-center gap-2.5"><IconTile tone="cyan" size="sm"><Server className="w-4 h-4" /></IconTile><div className="flex-1"><div className="text-slate-100">Control plane</div><div className="text-[11px] text-slate-400">{d.overview.cluster.quorum}</div></div></li>
                  <li className="flex items-center gap-2.5"><IconTile tone={d.verification.state === 'VERIFIED' ? 'emerald' : 'rose'} size="sm"><FileCheck2 className="w-4 h-4" /></IconTile><div className="flex-1"><div className="text-slate-100">Audit ledger</div><div className="text-[11px] text-slate-400">{d.verification.state} · {d.verification.entries} entries</div></div></li>
                  <li className="flex items-center gap-2.5"><IconTile tone="violet" size="sm"><HardDrive className="w-4 h-4" /></IconTile><div className="flex-1"><div className="text-slate-100">Storage</div><div className="text-[11px] text-slate-400">{metricText(m.volumes)} volumes · {metricText(m.volumesDegraded)} degraded</div></div></li>
                  <li className="flex items-center gap-2.5"><IconTile tone="blue" size="sm"><Database className="w-4 h-4" /></IconTile><div className="flex-1"><div className="text-slate-100">Durability</div><div className="text-[11px] text-slate-400 break-words">{d.overview.cluster.durability}</div></div></li>
                  <li className="flex items-center gap-2.5"><IconTile tone={d.security.some((c) => c.result === 'FAIL') ? 'rose' : 'emerald'} size="sm"><ShieldCheck className="w-4 h-4" /></IconTile><div className="flex-1"><div className="text-slate-100">Security controls</div><div className="text-[11px] text-slate-400">{d.security.filter((c) => c.result === 'PASS').length} pass · {d.security.filter((c) => c.result === 'WARN').length} warn · {d.security.filter((c) => c.result === 'FAIL').length} fail · {d.security.filter((c) => c.result === 'UNAVAILABLE').length} unavailable</div></div></li>
                </ul>
              </Glass>
            </div>
          </div>
        );
      }}
    </Gate>
  );
}
