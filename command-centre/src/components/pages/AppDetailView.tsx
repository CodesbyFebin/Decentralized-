import React, { useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, AppWindow, Check, X, Scaling, Trash2 } from 'lucide-react';
import type { AppRec, AuditEntryRec, DomainRec, VolumeRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError, type MutationResult } from '../../lib/client';
import { Glass, PageTabs, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Gate, FreshnessPill, shortDigest, since, fmtTime, fmtBytes, ErrorState, Empty, Note } from '../common/states';

export interface DeploymentProgress {
  id: string;
  app: string;
  generation: number;
  state: string;
  stages: { id: string; label: string; state: 'PASSED' | 'RUNNING' | 'PENDING' | 'FAILED'; detail: string }[];
  actor: string | null;
  change: string | null;
  submittedAt: number | null;
  revisions: { ts: number; actor: string; change: string; hash: string }[];
  updatedAt: number | null;
  image: string;
  imageDigest: string | null;
  hash: string;
}

interface Payload {
  app: AppRec;
  domains: DomainRec[];
  volumes: VolumeRec[];
  deployments: DeploymentProgress[];
  events: AuditEntryRec[];
}

type Tab = 'overview' | 'replicas' | 'deployments' | 'domains' | 'storage' | 'environment' | 'events';

/** Decentralization dimensions for one app, never a single score. */
export function Decentralization({ app }: { app: AppRec }) {
  const running = app.replicas.filter((r) => r.desired === 'RUNNING');
  const hosts = new Set(running.map((r) => r.node));
  const hostShare = running.length ? Math.max(...[...hosts].map((h) => running.filter((r) => r.node === h).length)) / running.length : 0;
  return (
    <Glass className="p-4">
      <PanelHeader title="Placement independence" subtitle="Dimensions the platform can observe. No single 'decentralization score'." />
      <ul className="mt-3 grid grid-cols-2 gap-2 text-[12.5px]">
        <li className="flex justify-between"><span className="text-slate-400">Desired replicas</span><span className="text-slate-100">{app.desiredReplicas}</span></li>
        <li className="flex justify-between"><span className="text-slate-400">Distinct hosts</span><span className="text-slate-100">{hosts.size}</span></li>
        <li className="flex justify-between"><span className="text-slate-400">Largest host share</span><span className="text-slate-100">{running.length ? `${Math.round(hostShare * 100)}%` : '—'}</span></li>
        <li className="flex justify-between"><span className="text-slate-400">Independent operators</span><span className="text-slate-100">1 (your cluster root)</span></li>
        <li className="flex justify-between"><span className="text-slate-400">Federated replicas</span><span className="text-slate-100">{app.federated ? 'yes' : '0'}</span></li>
        <li className="flex justify-between"><span className="text-slate-400">Central-cloud replicas</span><span className="text-slate-100">unknown</span></li>
      </ul>
      <Note>All hosts are enrolled under one cluster root, so they belong to one operator. Operator independence requires federation with another cluster.</Note>
    </Glass>
  );
}

export const StageList: React.FC<{ stages: DeploymentProgress['stages'] }> = ({ stages }) => (
  <ol className="space-y-2">
    {stages.map((s) => (
      <li key={s.id} className="flex items-center gap-3">
        <span className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-bold border ${s.state === 'PASSED' ? 'bg-emerald-500/20 border-emerald-400/50 text-emerald-300' : s.state === 'FAILED' ? 'bg-rose-500/20 border-rose-400/50 text-rose-300' : s.state === 'RUNNING' ? 'bg-blue-500/20 border-blue-400/50 text-blue-300 animate-pulse' : 'bg-white/[0.03] border-slate-600 text-slate-500'}`}>
          {s.state === 'PASSED' ? <Check className="w-3.5 h-3.5" /> : s.state === 'FAILED' ? <X className="w-3.5 h-3.5" /> : '·'}
        </span>
        <span className="text-[13px] text-slate-100 w-56">{s.label}</span>
        <StatusPill status={s.state} dot={false} tone={s.state === 'PASSED' ? 'emerald' : s.state === 'FAILED' ? 'rose' : s.state === 'RUNNING' ? 'blue' : 'slate'} />
        <span className="text-[12px] text-slate-400">{s.detail}</span>
      </li>
    ))}
  </ol>
);

export default function AppDetailView() {
  const { name = '' } = useParams();
  const res = useResource<Payload>(`/apps/${encodeURIComponent(name)}`);
  const [params, setParams] = useSearchParams();
  const tab = (params.get('tab') as Tab) || 'overview';
  const { can, mode } = useSession();
  const navigate = useNavigate();
  const [scaleTo, setScaleTo] = useState<string>('');
  const [confirmText, setConfirmText] = useState('');
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<MutationResult | null>(null);
  const [err, setErr] = useState<ApiError | null>(null);
  const [deleting, setDeleting] = useState(false);

  const mutate = async (path: string, body: unknown) => {
    setBusy(true);
    setErr(null);
    setResult(null);
    try {
      const r = await api.post<{ data: MutationResult }>(path, body);
      setResult(r.data);
      res.refresh();
      return r.data;
    } catch (e) {
      setErr(e as ApiError);
      return null;
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 pt-2">
      <Link to="/apps" className="inline-flex items-center gap-1.5 text-[12.5px] text-slate-400 hover:text-cyan-300">
        <ArrowLeft className="w-3.5 h-3.5" /> Websites & Apps
      </Link>
      <Gate res={res}>
        {(d, stale) => {
          const a = d.app;
          const canWrite = can('api.write') && !stale && mode === 'controlplane';
          const n = Number(scaleTo);
          return (
            <>
              <Glass className="p-5">
                <div className="flex flex-wrap items-start gap-4">
                  <IconTile tone="blue" size="lg"><AppWindow className="w-6 h-6" /></IconTile>
                  <div className="flex-1 min-w-0">
                    <h1 className="text-2xl font-extrabold text-white tracking-tight">{a.name}</h1>
                    <div className="mt-1 flex flex-wrap gap-2 items-center">
                      <StatusPill status={a.phase} />
                      <span className="text-[12px] text-slate-400">{a.phaseReason}</span>
                    </div>
                    <div className="mt-2 text-[12px] text-slate-400">
                      generation {a.generation} · {a.runtime} · <span className="font-mono" title={a.image}>{a.image.split('@')[0]}@{shortDigest(a.imageDigest)}</span>
                    </div>
                  </div>
                  <div className="grid grid-cols-4 gap-2 text-center">
                    {[
                      ['Desired', a.desiredReplicas],
                      ['Admitted', a.admitted],
                      ['Observed', a.observedRunning],
                      ['Healthy', a.healthyReplicas]
                    ].map(([k, v]) => (
                      <div key={k} className="px-3 py-2 rounded-xl bg-white/[0.03] border border-[rgba(125,190,255,0.12)]">
                        <div className="text-[20px] font-bold text-white tabular-nums">{v}</div>
                        <div className="text-[10.5px] text-slate-400">{k}</div>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="mt-4 flex flex-wrap items-end gap-2">
                  <label className="text-[12px] text-slate-300">
                    Scale to
                    <input type="number" min={0} max={64} value={scaleTo} onChange={(e) => setScaleTo(e.target.value)} placeholder={String(a.desiredReplicas)} className="ml-2 w-20 rounded-lg bg-black/30 border border-[rgba(125,190,255,0.2)] px-2 py-1.5 text-[12.5px]" />
                  </label>
                  {n === 0 && scaleTo !== '' && (
                    <input value={confirmText} onChange={(e) => setConfirmText(e.target.value)} placeholder={`type ${a.name} to stop it`} className="rounded-lg bg-black/30 border border-amber-400/40 px-2 py-1.5 text-[12.5px]" />
                  )}
                  <GhostButton
                    disabled={!canWrite || busy || scaleTo === '' || !Number.isInteger(n) || n < 0 || n > 64 || n === a.desiredReplicas || (n === 0 && confirmText !== a.name)}
                    title={canWrite ? undefined : 'needs api.write on fresh data'}
                    onClick={() => mutate(`/apps/${encodeURIComponent(a.name)}/scale`, { replicas: n, confirm: confirmText })}
                    className="disabled:opacity-40"
                  >
                    <Scaling className="w-4 h-4" /> Apply
                  </GhostButton>
                  <GhostButton disabled={!canWrite || busy} onClick={() => setDeleting(true)} className="!border-rose-400/40 !text-rose-200 disabled:opacity-40 ml-auto">
                    <Trash2 className="w-4 h-4" /> Delete
                  </GhostButton>
                </div>
                {deleting && (
                  <div className="mt-3 rounded-xl border border-rose-400/30 bg-rose-500/10 p-3" role="alertdialog">
                    <div className="text-[13px] text-rose-100">Delete {a.name}? Every replica is stopped. Type the name to confirm.</div>
                    <div className="mt-2 flex gap-2">
                      <input value={confirmText} onChange={(e) => setConfirmText(e.target.value)} className="rounded-lg bg-black/40 border border-rose-400/40 px-2 py-1.5 text-[12.5px]" />
                      <GhostButton
                        disabled={confirmText !== a.name || busy}
                        onClick={async () => {
                          const r = await mutate(`/apps/${encodeURIComponent(a.name)}/delete`, { confirm: confirmText });
                          if (r?.ok) navigate('/apps');
                        }}
                        className="!border-rose-400/50 disabled:opacity-40"
                      >
                        Delete application
                      </GhostButton>
                      <GhostButton onClick={() => setDeleting(false)}>Cancel</GhostButton>
                    </div>
                  </div>
                )}
                {result && (
                  <div className={`mt-3 rounded-xl p-3 text-[12.5px] border ${result.ok ? 'border-emerald-400/30 bg-emerald-500/10 text-emerald-100' : 'border-rose-400/30 bg-rose-500/10 text-rose-100'}`} role="status">
                    Control plane {result.ok ? 'committed' : 'refused'}: {result.message || result.code} <span className="font-mono text-[10.5px] opacity-70">· request {result.request_id}</span>
                  </div>
                )}
                {err && <div className="mt-3"><ErrorState error={err} /></div>}
              </Glass>

              <PageTabs<Tab>
                tabs={[
                  { id: 'overview', label: 'Overview' },
                  { id: 'replicas', label: `Replicas (${a.replicas.length})` },
                  { id: 'deployments', label: `Deployments (${d.deployments.length})` },
                  { id: 'domains', label: 'Domains' },
                  { id: 'storage', label: 'Storage' },
                  { id: 'environment', label: 'Environment' },
                  { id: 'events', label: 'Events' }
                ]}
                active={tab}
                onChange={(t) => setParams((p) => (t === 'overview' ? (p.delete('tab'), p) : (p.set('tab', t), p)), { replace: true })}
              />

              {tab === 'overview' && (
                <div className="grid lg:grid-cols-2 gap-4">
                  <Glass className="p-4">
                    <PanelHeader title="Latest deployment" subtitle={d.deployments[0] ? `generation ${d.deployments[0].generation} · ${d.deployments[0].state}` : undefined} />
                    <div className="mt-3">{d.deployments[0] ? <StageList stages={d.deployments[0].stages} /> : <p className="text-slate-400 text-[12.5px]">No history.</p>}</div>
                  </Glass>
                  <Decentralization app={a} />
                </div>
              )}

              {tab === 'replicas' && (
                <div className="space-y-3">
                  {a.replicas.map((r) => (
                    <Glass key={r.assignment + r.node} className="p-4">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-semibold text-white">{r.assignment}</span>
                        <span className="text-slate-500 text-[12px]">on</span>
                        <Link to={`/nodes/${r.node}`} className="text-cyan-300 hover:underline text-[13px]">{r.nodeName}</Link>
                        <span className="text-[11.5px] text-slate-500">desired gen {r.desiredGen} · admitted gen {r.admittedGen} · observed gen {r.observedGen}</span>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-2 items-center text-[12px]">
                        <span className="text-slate-400">Desired</span><StatusPill status={r.desired} dot={false} />
                        <span className="text-slate-400">Admitted</span><StatusPill status={r.admitted} dot={false} />
                        <span className="text-slate-400">Observed</span><StatusPill status={r.observed} dot={false} />
                        <FreshnessPill f={r.freshness} ageMs={r.observedAt ? Date.now() - r.observedAt : null} />
                        {r.health && <span className="text-slate-300">health: {r.health.ok ? 'passing' : 'failing'} ({r.health.detail})</span>}
                        {r.restarts > 0 && <span className="text-amber-300">{r.restarts} restart(s)</span>}
                      </div>
                      <div className="mt-2 text-[12px] text-slate-400">{r.code}: {r.reason}</div>
                      {r.checks.length > 0 && (
                        <details className="mt-2">
                          <summary className="text-[12px] text-cyan-300 cursor-pointer">Host admission checks ({r.checks.filter((c) => c.ok).length}/{r.checks.length} passed)</summary>
                          <ul className="mt-2 grid md:grid-cols-2 gap-1">
                            {r.checks.map((c) => (
                              <li key={c.name} className="flex gap-2 text-[11.5px]">
                                {c.ok ? <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0" /> : <X className="w-3.5 h-3.5 text-rose-400 shrink-0" />}
                                <span className="text-slate-300 font-semibold">{c.name}</span>
                                <span className="text-slate-400">{c.detail}</span>
                              </li>
                            ))}
                          </ul>
                        </details>
                      )}
                      {r.evidence && <div className="mt-2 text-[11px] text-slate-500 font-mono">observation evidence {shortDigest(r.evidence)}</div>}
                    </Glass>
                  ))}
                </div>
              )}

              {tab === 'deployments' && (
                <Glass className="overflow-x-auto">
                  <table className="dh-table w-full min-w-[720px]">
                    <thead><tr><th>Generation</th><th>State</th><th>Change</th><th>Actor</th><th>Submitted</th><th>Manifest hash</th></tr></thead>
                    <tbody>
                      {d.deployments.map((x) => (
                        <tr key={x.id}>
                          <td><Link className="text-cyan-300 hover:underline" to={`/deploy/${encodeURIComponent(x.app)}/${x.generation}`}>#{x.generation}</Link></td>
                          <td><StatusPill status={x.state} /></td>
                          <td className="text-slate-300 whitespace-normal">{x.revisions.map((r) => r.change).join(' · ') || '—'}</td>
                          <td className="text-slate-300">{x.actor ?? '—'}</td>
                          <td className="text-slate-300">{fmtTime(x.submittedAt)}</td>
                          <td className="font-mono text-[11.5px] text-slate-400">{shortDigest(x.hash)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </Glass>
              )}

              {tab === 'domains' &&
                (d.domains.length ? (
                  <Glass className="p-4 space-y-2">
                    {d.domains.map((x) => (
                      <div key={x.host} className="flex flex-wrap items-center gap-3 text-[12.5px]">
                        <span className="text-sky-300 font-semibold">{x.host}</span>
                        <span className="text-slate-400">routing</span><StatusPill status={x.routing.state === 'NOT ROUTED' ? 'UNKNOWN' : x.routing.state} />
                        <span className="text-slate-400">TLS</span><StatusPill status={x.tls?.state ?? 'UNKNOWN'} />
                        <span className="text-slate-500">{x.routing.detail}</span>
                      </div>
                    ))}
                  </Glass>
                ) : (
                  <Empty title="No ingress configured" detail="Add spec.ingress to the manifest to route a host name through the edge." />
                ))}

              {tab === 'storage' &&
                (d.volumes.length ? (
                  <Glass className="p-4 space-y-2">
                    {d.volumes.map((v) => (
                      <div key={v.id} className="flex flex-wrap items-center gap-3 text-[12.5px]">
                        <span className="text-slate-100 font-semibold">{v.id}</span>
                        <StatusPill status={v.state} />
                        <span className="text-slate-400">{fmtBytes(v.sizeBytes)} · {v.verified}/{v.durabilityReplicas} verified · on {v.memberNames.join(', ')}</span>
                      </div>
                    ))}
                  </Glass>
                ) : (
                  <Empty title="No volumes" />
                ))}

              {tab === 'environment' && (
                <Glass className="p-4">
                  <PanelHeader title="Environment variables" subtitle="Names only. Values never leave the BFF." />
                  <div className="mt-2 flex flex-wrap gap-2">
                    {a.envNames.length ? a.envNames.map((k) => <code key={k} className="px-2 py-1 rounded-md bg-white/[0.04] border border-[rgba(125,190,255,0.14)] text-[12px] text-cyan-200">{k}</code>) : <span className="text-slate-400 text-[12.5px]">none</span>}
                  </div>
                  <div className="mt-4 text-[12.5px] text-slate-300">Resources: cpu {a.resources.cpu || '—'} · mem {a.resources.mem || '—'}</div>
                  <Note>The platform has no separate secrets store yet; manifest env values are stored in the control plane state. Do not put secrets in them.</Note>
                </Glass>
              )}

              {tab === 'events' &&
                (d.events.length ? (
                  <Glass className="p-4">
                    {d.events.map((e) => (
                      <div key={e.seq} className="py-2 border-b border-[rgba(125,190,255,0.06)] last:border-0">
                        <div className="text-[12.5px] text-slate-100"><span className="font-mono text-slate-500">#{e.seq}</span> {e.action} <span className="text-slate-400">{e.resource}</span></div>
                        <div className="text-[11.5px] text-slate-400">{e.detail} · {e.actor} · {since(e.ts)}</div>
                      </div>
                    ))}
                  </Glass>
                ) : (
                  <Empty title="No recent events for this application" />
                ))}
            </>
          );
        }}
      </Gate>
    </div>
  );
}
