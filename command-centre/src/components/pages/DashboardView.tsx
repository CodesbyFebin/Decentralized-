import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Rocket, Server, HardDrive, Globe, Bot, AppWindow, Layers, Activity, AlertTriangle, AlertOctagon, Info, CheckCircle2, Clock, CloudOff, HelpCircle, Radio, FileCheck2, Database, Network, Plus } from 'lucide-react';
import type { AppRec, AttentionItem, AuditEntryRec, CertRec, DomainRec, LedgerVerification, NodeRec, Overview, SecurityControl, ValidationRecord, ValidationVerification } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { HoloGlobe } from '../common/HoloGlobe';
import { regionGroups, groupMarkers, meshArcs, NoGeoNote } from '../common/CommandSurface';
import { Glass, PanelHeader, IconTile, StatusPill, TONE, type Tone } from '../common/ui';
import { Gate, fmtBytes, since, fmtTime, TruthTag, Unavailable } from '../common/states';
import { TruthValue, MetricValue, FreshnessBadge, EvidenceSeal, viewFreshness, type ViewFreshness } from '../common/truth';

type AppSummary = Omit<AppRec, 'replicas' | 'history'> & { replicaCount: number; lastChange: AppRec['history'][number] | null };

interface Payload {
  overview: Overview;
  nodes: NodeRec[];
  apps: AppSummary[];
  domains: DomainRec[];
  certificates: CertRec[];
  security: SecurityControl[];
  verification: LedgerVerification;
  recentEvents: AuditEntryRec[];
  attention: AttentionItem[];
  volumes: { id: string; app: string; name: string; state: string; members: string[] }[];
}

/** Audit actions that are operations someone or something asked for (not routine host mode notes). */
const OPERATION = /^(apply|scale|delete-app|node-|enroll|rotate|revoke|freeze|restore|root-rotate|artifact|attest|volume|federation|invite)/;

const OverviewCard: React.FC<{
  to: string;
  icon: React.ReactNode;
  tone: Tone;
  label: string;
  children: React.ReactNode;
  sub?: React.ReactNode;
}> = ({ to, icon, tone, label, children, sub }) => (
  <Link to={to} className="dh-glass rounded-2xl px-3.5 py-3.5 min-w-0 relative overflow-hidden dh-hover-lift focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400">
    <div className="absolute -left-8 -top-8 w-24 h-24 rounded-full blur-2xl pointer-events-none" style={{ background: `${TONE[tone]}22` }} />
    <div className="relative flex items-start gap-3">
      <IconTile tone={tone} size="md" className="!w-10 !h-10 shrink-0">{icon}</IconTile>
      <div className="min-w-0 flex-1">
        <div className="text-[12px] text-slate-300/90">{label}</div>
        {children}
        {sub && <div className="text-[11px] text-slate-400 mt-1 leading-snug">{sub}</div>}
      </div>
    </div>
  </Link>
);

const SEV: Record<AttentionItem['severity'], { tone: Tone; icon: React.ReactNode; label: string }> = {
  critical: { tone: 'rose', icon: <AlertOctagon className="w-4 h-4" aria-hidden />, label: 'Critical' },
  warning: { tone: 'amber', icon: <AlertTriangle className="w-4 h-4" aria-hidden />, label: 'Warning' },
  info: { tone: 'blue', icon: <Info className="w-4 h-4" aria-hidden />, label: 'Info' }
};

const HEALTH: { key: 'healthy' | 'degraded' | 'offline' | 'unknown'; label: string; tone: Tone; icon: React.ReactNode; metric: keyof Overview['metrics'] }[] = [
  { key: 'healthy', label: 'Healthy (fresh)', tone: 'emerald', icon: <CheckCircle2 className="w-4 h-4" aria-hidden />, metric: 'nodesHealthy' },
  { key: 'degraded', label: 'Stale', tone: 'amber', icon: <Clock className="w-4 h-4" aria-hidden />, metric: 'nodesDegraded' },
  { key: 'offline', label: 'Offline', tone: 'rose', icon: <CloudOff className="w-4 h-4" aria-hidden />, metric: 'nodesOffline' },
  { key: 'unknown', label: 'Unknown', tone: 'slate', icon: <HelpCircle className="w-4 h-4" aria-hidden />, metric: 'nodesUnknown' }
];

function EvidenceSnapshot() {
  const { capabilities } = useSession();
  const res = useResource<{ records: ValidationRecord[]; verifications: Record<string, ValidationVerification | null>; canVerify: boolean }>(
    capabilities?.items.validationRecords?.state === 'LIVE' ? '/evidence/records' : null,
    { pollMs: 60_000 }
  );
  const cap = capabilities?.items.validationRecords;
  if (!cap || cap.state !== 'LIVE') return <Unavailable title="Validation records" detail={cap?.detail ?? 'capability not discovered yet'} />;
  if (!res.data) return <p className="text-[12.5px] text-slate-400 mt-3">{res.error ? `Records unavailable: ${res.error.message}` : 'Reading records…'}</p>;
  const recs = res.data.records.slice(0, 5);
  return (
    <ul className="mt-3 space-y-2">
      {recs.map((r) => {
        const v = res.data!.verifications[r.id];
        return (
          <li key={r.id}>
            <Link to={`/evidence/${encodeURIComponent(r.id)}`} className={`flex items-center gap-2 p-2 rounded-xl border hover:border-cyan-400/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 ${r.outcome === 'FAIL' ? 'border-rose-400/35 bg-rose-500/[0.06]' : 'border-[rgba(125,190,255,0.09)] bg-white/[0.02]'}`}>
              <EvidenceSeal outcome={r.outcome} />
              <span className="font-mono text-[12px] text-slate-100">{r.id}</span>
              <span className="text-[11px] text-slate-400 truncate flex-1">{r.stage}</span>
              <span className="text-[11px] text-slate-400 whitespace-nowrap">{v ? v.state : 'not verified here'} · {since(r.endedAt)}</span>
            </Link>
          </li>
        );
      })}
      {!recs.length && <li className="text-[12.5px] text-slate-400">No validation records in the evidence directory.</li>}
    </ul>
  );
}

export default function DashboardView() {
  const res = useResource<Payload>('/overview');
  const navigate = useNavigate();
  const { can } = useSession();

  return (
    <Gate res={res} loadingRows={4}>
      {(d, stale) => {
        const m = d.overview.metrics;
        const observedAt = res.provenance?.observedAt ?? d.overview.generatedAt;
        const fresh: ViewFreshness = viewFreshness({ pageStale: stale });
        const source = 'control plane';
        const groups = regionGroups(d.nodes);
        const liveApps = d.apps.filter((a) => !a.deleted);
        const deploying = liveApps.filter((a) => a.phase === 'CONVERGING');
        const routing = d.domains.filter((x) => x.routing.state === 'LIVE').length;
        const edges = d.nodes.filter((n) => n.isEdge);
        const meshed = d.nodes.filter((n) => n.mesh && n.mesh.device !== 'none');
        const ops = d.recentEvents.filter((e) => OPERATION.test(e.action)).slice(0, 8);
        const attention: AttentionItem[] = stale
          ? [{ severity: 'critical', kind: 'control-plane', subject: 'control plane', title: 'CONTROL PLANE UNREACHABLE', detail: `last observation ${fmtTime(res.fetchedAt)}; nothing on this page is current`, href: '/settings', source: 'console', observedAt: res.fetchedAt }, ...d.attention]
          : d.attention;
        const summary = stale
          ? 'The control plane is unreachable. Everything below is the last observation, not the current state.'
          : attention.length === 0
            ? `${metricText0(m.nodesHealthy)} of ${metricText0(m.nodesTotal)} hosts are reporting fresh observations and every desired replica is observed.`
            : `${attention.filter((a) => a.severity === 'critical').length} critical and ${attention.filter((a) => a.severity === 'warning').length} warning item(s) need attention.`;
        return (
          <div className="space-y-4 pt-1">
            {/* Hero */}
            <section className="relative overflow-hidden rounded-3xl dh-glass p-6 lg:p-7 min-h-[220px]" aria-labelledby="dash-title">
              <div className="hidden md:block">
                <HoloGlobe
                  markers={groupMarkers(groups.placed, (g) => [`${g.nodes.length} host${g.nodes.length > 1 ? 's' : ''}`], (g) => navigate(`/nodes/${g.nodes[0].id}`))}
                  arcs={meshArcs(groups.placed)}
                  focusLng={groups.placed[0]?.lng ?? 20}
                  tilt={24}
                  speed={2}
                  frozen={stale}
                  center={[0.72, 1.05]}
                  radius={0.95}
                  resolution={1.3}
                  className="!absolute inset-y-0 right-0 w-[58%]"
                />
              </div>
              <div className="relative max-w-[640px]">
                <div className="text-[12px] font-semibold tracking-[0.2em] text-cyan-300/90">DECENTRALIZED.HOST</div>
                <h1 id="dash-title" className="mt-1 text-[32px] sm:text-[40px] font-extrabold tracking-tight text-white leading-tight">YOUR DATA. YOUR RULES.</h1>
                <p className="mt-3 text-[13px] text-slate-400">What is happening across my infrastructure right now?</p>
                <p className="mt-1 text-[15px] text-slate-200">{summary}</p>
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  <FreshnessBadge f={fresh} observedAt={observedAt} />
                  <span className="text-[11px] text-slate-400">source: {d.overview.provenance.source} · served by {d.overview.cluster.servedBy.member || 'unknown member'}</span>
                </div>
                {/* Quick actions */}
                <div className="mt-5 flex flex-wrap gap-2" role="group" aria-label="Quick actions">
                  {can('api.write') && <QuickAction to="/deploy/new" icon={<Rocket className="w-4 h-4" />} label="Deploy Application" />}
                  <QuickAction to="/nodes/add" icon={<Plus className="w-4 h-4" />} label="Add Node" />
                  {can('api.write') && <QuickAction to="/deploy/new" icon={<HardDrive className="w-4 h-4" />} label="Create Volume" title="Volumes are declared in an application manifest (spec.volumes)" />}
                  {can('api.write') && <QuickAction to="/deploy/new" icon={<Globe className="w-4 h-4" />} label="Add Domain" title="Domains are declared in an application manifest (spec.ingress)" />}
                  <QuickAction to="/copilot" icon={<Bot className="w-4 h-4" />} label="Open Copilot" accent />
                </div>
              </div>
            </section>

            {/* 1. Infrastructure overview */}
            <section aria-label="Infrastructure overview" className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
              <OverviewCard to="/nodes" tone="cyan" icon={<Server className="w-5 h-5" />} label="Nodes">
                <MetricValue m={m.nodesTotal} size="lg" freshness={fresh} observedAt={observedAt} />
                <div className="text-[11px] text-slate-400">{metricText0(m.nodesHealthy)} fresh · {metricText0(m.nodesDegraded)} stale · {metricText0(m.nodesOffline)} offline</div>
              </OverviewCard>
              <OverviewCard to="/apps" tone="blue" icon={<AppWindow className="w-5 h-5" />} label="Applications">
                <TruthValue value={liveApps.length} state={m.applications.state} size="lg" freshness={fresh} observedAt={observedAt} source={source} />
                <div className="text-[11px] text-slate-400">{liveApps.filter((a) => a.phase === 'READY').length} ready</div>
              </OverviewCard>
              <OverviewCard to="/apps" tone="emerald" icon={<Layers className="w-5 h-5" />} label="Running workloads">
                <MetricValue m={m.replicasObserved} size="lg" freshness={fresh} observedAt={observedAt} />
                <div className="text-[11px] text-slate-400">observed running, fresh · of {metricText0(m.replicasDesired)} desired</div>
              </OverviewCard>
              <OverviewCard to="/storage" tone="violet" icon={<Database className="w-5 h-5" />} label="Storage used">
                <MetricValue m={m.storageUsedBytes} format={(n) => fmtBytes(n)} size="lg" freshness={fresh} observedAt={observedAt} />
                <div className="text-[11px] text-slate-400">{m.storageCapacityBytes.value !== null ? `of ${fmtBytes(m.storageCapacityBytes.value)} host quota` : m.storageCapacityBytes.detail ?? 'quota not reported'} · {metricText0(m.volumes)} volumes</div>
              </OverviewCard>
              <OverviewCard to="/deploy" tone="amber" icon={<Rocket className="w-5 h-5" />} label="Active deployments">
                <TruthValue value={deploying.length} state="DERIVED" size="lg" freshness={fresh} observedAt={observedAt} source="replica rows" />
                <div className="text-[11px] text-slate-400">applications still converging to their desired generation</div>
              </OverviewCard>
              <OverviewCard to="/domains" tone="cyan" icon={<Globe className="w-5 h-5" />} label="Domains">
                <MetricValue m={m.domains} size="lg" freshness={fresh} observedAt={observedAt} />
                <div className="text-[11px] text-slate-400">{routing} routing at the edge · DNS not observed</div>
              </OverviewCard>
            </section>

            <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)] gap-4">
              {/* 2. Topology */}
              <Glass className="p-4 overflow-hidden">
                <PanelHeader icon={<IconTile tone="cyan" size="sm"><Radio className="w-4 h-4" /></IconTile>} title="Infrastructure topology" subtitle="Only what the control plane reports. Every host here is enrolled under this cluster's root." right={<Link to="/nodes" className="text-[12px] text-cyan-300 hover:underline">Nodes →</Link>} />
                <div className="hidden md:block relative">
                  <HoloGlobe markers={groupMarkers(groups.placed, (g) => g.nodes.map((n) => `${n.name}: ${n.health.toLowerCase()}`))} arcs={meshArcs(groups.placed)} focusLng={groups.placed[0]?.lng ?? 30} tilt={22} speed={1.4} frozen={stale} center={[0.5, 0.6]} radius={0.6} resolution={1.3} className="h-[230px] -mx-4" />
                  <NoGeoNote unplaced={groups.unplaced} />
                </div>
                <table className="dh-table w-full mt-2" aria-label="Topology as a table">
                  <thead><tr><th>Layer</th><th>Observed</th><th>Detail</th></tr></thead>
                  <tbody>
                    <tr><td><span className="inline-flex items-center gap-1.5"><Server className="w-3.5 h-3.5" aria-hidden /> My nodes</span></td><td className="tabular-nums">{d.nodes.length}</td><td className="text-slate-400">{[...new Set(d.nodes.map((n) => n.region || 'no region'))].join(', ') || '—'}</td></tr>
                    <tr><td><span className="inline-flex items-center gap-1.5"><Network className="w-3.5 h-3.5" aria-hidden /> Private mesh</span></td><td className="tabular-nums">{meshed.length}</td><td className="text-slate-400">hosts with a WireGuard device; {meshed.reduce((a, n) => a + (n.mesh?.handshakesRecent ?? 0), 0)} recent handshakes reported</td></tr>
                    <tr><td><span className="inline-flex items-center gap-1.5"><AppWindow className="w-3.5 h-3.5" aria-hidden /> Applications</span></td><td className="tabular-nums">{liveApps.length}</td><td className="text-slate-400">{liveApps.reduce((a, x) => a + x.observedRunning, 0)} replicas observed running</td></tr>
                    <tr><td><span className="inline-flex items-center gap-1.5"><HardDrive className="w-3.5 h-3.5" aria-hidden /> Storage</span></td><td className="tabular-nums">{d.volumes.length}</td><td className="text-slate-400">replicated volumes across {new Set(d.volumes.flatMap((v) => v.members)).size} host(s)</td></tr>
                    <tr><td><span className="inline-flex items-center gap-1.5"><Globe className="w-3.5 h-3.5" aria-hidden /> Edge</span></td><td className="tabular-nums">{edges.length}</td><td className="text-slate-400">{edges.map((n) => n.name).join(', ') || 'no edge host'}</td></tr>
                    <tr className="opacity-70"><td>Community / DePIN capacity</td><td><TruthTag state="PLANNED" /></td><td className="text-slate-500">not implemented; no external capacity is shown</td></tr>
                  </tbody>
                </table>
              </Glass>

              <div className="space-y-4">
                {/* 5. Attention required */}
                <Glass className="p-4">
                  <PanelHeader title="Attention required" subtitle="Conditions the current observation proves, each with its source." />
                  {attention.length === 0 ? (
                    <p className="mt-3 flex items-center gap-2 text-[12.5px] text-emerald-300"><CheckCircle2 className="w-4 h-4" aria-hidden /> Nothing the control plane reports needs attention.</p>
                  ) : (
                    <ul className="mt-3 space-y-2">
                      {attention.slice(0, 8).map((a, i) => {
                        const s = SEV[a.severity];
                        return (
                          <li key={`${a.kind}-${a.subject}-${i}`}>
                            <Link to={a.href} className="flex items-start gap-2.5 p-2 rounded-xl border border-[rgba(125,190,255,0.09)] bg-white/[0.02] hover:border-cyan-400/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400">
                              <span className="mt-0.5 inline-flex items-center gap-1 text-[10.5px] font-bold uppercase" style={{ color: TONE[s.tone] }}>{s.icon}{s.label}</span>
                              <span className="min-w-0 flex-1">
                                <span className="block text-[12.5px] text-slate-100">{a.title}</span>
                                <span className="block text-[11px] text-slate-400 truncate">{a.detail}</span>
                                <span className="block text-[10.5px] text-slate-500">{a.source}{a.observedAt ? ` · ${since(a.observedAt)}` : ''}</span>
                              </span>
                            </Link>
                          </li>
                        );
                      })}
                      {attention.length > 8 && <li className="text-[11.5px] text-slate-400">and {attention.length - 8} more</li>}
                    </ul>
                  )}
                </Glass>

                {/* 3. Health */}
                <Glass className="p-4">
                  <PanelHeader title="Host health" subtitle="Counts from signed observations. No percentage is computed." />
                  <ul className="mt-3 grid grid-cols-2 gap-2">
                    {HEALTH.map((h) => (
                      <li key={h.key} className="flex items-center gap-2 p-2 rounded-xl bg-white/[0.02] border border-[rgba(125,190,255,0.08)]">
                        <span style={{ color: TONE[h.tone] }}>{h.icon}</span>
                        <span className="text-[12px] text-slate-300 flex-1">{h.label}</span>
                        <MetricValue m={m[h.metric]} size="sm" />
                      </li>
                    ))}
                  </ul>
                </Glass>
              </div>
            </div>

            <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
              {/* 4. Recent operations */}
              <Glass className="p-4 xl:col-span-1">
                <PanelHeader title="Recent operations" subtitle="From the hash-chained audit ledger" right={<Link to="/activity" className="text-[12px] text-cyan-300 hover:underline">Activity →</Link>} />
                <ul className="mt-3 space-y-2">
                  {ops.map((e) => (
                    <li key={e.seq} className="flex items-start gap-2.5">
                      <IconTile tone={e.source === 'host' ? 'emerald' : e.source === 'operator' ? 'violet' : 'blue'} size="sm"><Activity className="w-4 h-4" /></IconTile>
                      <div className="min-w-0 flex-1">
                        <div className="text-[12.5px] text-slate-100 truncate">{e.action} <span className="text-slate-400">{e.resource}</span></div>
                        <div className="text-[11px] text-slate-500 truncate">{e.actor} · #{e.seq}</div>
                      </div>
                      <span className="text-[11px] text-slate-500 whitespace-nowrap" title={fmtTime(e.ts)}>{since(e.ts)}</span>
                    </li>
                  ))}
                  {!ops.length && <li className="text-[12.5px] text-slate-400">No operations in the latest {d.recentEvents.length} audit entries.</li>}
                </ul>
              </Glass>

              {/* 6. Evidence snapshot */}
              <Glass className="p-4">
                <PanelHeader title="Evidence snapshot" subtitle="Latest signed validation records; failures stay visible." right={<Link to="/evidence" className="text-[12px] text-cyan-300 hover:underline">Evidence →</Link>} />
                <div className="mt-3 flex items-center gap-2 text-[12px]">
                  <FileCheck2 className="w-4 h-4 text-slate-400" aria-hidden />
                  <span className="text-slate-300">Audit ledger</span>
                  <StatusPill status={d.verification.state} />
                  <span className="text-slate-500">{d.verification.entries} entries · {d.verification.verifiedAt ? `verified ${since(d.verification.verifiedAt)}` : 'not verified'}</span>
                </div>
                <EvidenceSnapshot />
              </Glass>

              <Glass className="p-4">
                <PanelHeader title="Applications" right={<Link to="/apps" className="text-[12px] text-cyan-300 hover:underline">Apps →</Link>} />
                <table className="dh-table w-full mt-2">
                  <thead><tr><th>Name</th><th>Phase</th><th>Observed / desired</th></tr></thead>
                  <tbody>
                    {liveApps.slice(0, 6).map((a) => (
                      <tr key={a.name}>
                        <td><Link to={`/apps/${encodeURIComponent(a.name)}`} className="text-slate-100 hover:text-cyan-300 font-semibold">{a.name}</Link></td>
                        <td title={a.phaseReason}><StatusPill status={a.phase} /></td>
                        <td className="tabular-nums">{a.observedRunning} / {a.desiredReplicas}</td>
                      </tr>
                    ))}
                    {!liveApps.length && <tr><td colSpan={3} className="text-slate-400 text-center py-4">No applications deployed.</td></tr>}
                  </tbody>
                </table>
              </Glass>
            </div>
          </div>
        );
      }}
    </Gate>
  );
}

function metricText0(m: { value: number | null }) {
  return m.value === null ? 'unknown' : m.value.toLocaleString();
}

const QuickAction: React.FC<{ to: string; icon: React.ReactNode; label: string; title?: string; accent?: boolean }> = ({ to, icon, label, title, accent }) => (
  <Link
    to={to}
    title={title}
    className={`inline-flex items-center gap-1.5 px-3 py-2 rounded-xl border text-[13px] font-semibold focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 ${accent ? 'border-cyan-400/50 text-cyan-100 bg-cyan-400/10' : 'border-[rgba(125,190,255,0.18)] text-slate-100 bg-white/[0.03] hover:border-cyan-400/40'}`}
  >
    {icon} {label}
  </Link>
);
