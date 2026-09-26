import React, { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Server, Radio, Pause, Play, CheckCircle2, ShieldOff, Terminal, KeyRound } from 'lucide-react';
import type { AuditEntryRec, NodeOperation, NodeRec, ReplicaRec, VolumeRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError, type MutationResult } from '../../lib/client';
import { Glass, PageTabs, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Gate, FreshnessPill, fmtBytes, fmtTime, since, shortDigest, ErrorState, Empty, Note, TruthTag } from '../common/states';

interface Payload {
  node: NodeRec;
  replicas: (ReplicaRec & { app: string })[];
  events: AuditEntryRec[];
  diagnostics: { subject: string; item: string; value: string; basis: string; detail: string }[];
  volumes: VolumeRec[];
}

type Tab = 'overview' | 'workloads' | 'facts' | 'mesh' | 'storage' | 'identity' | 'events' | 'logs';

const Row: React.FC<{ k: string; children: React.ReactNode }> = ({ k, children }) => (
  <div className="flex justify-between gap-4 py-1.5 text-[12.5px] border-b border-[rgba(125,190,255,0.06)] last:border-0">
    <span className="text-slate-400">{k}</span>
    <span className="text-slate-100 text-right break-all">{children}</span>
  </div>
);

export default function NodeDetailView() {
  const { id = '' } = useParams();
  const res = useResource<Payload>(`/nodes/${encodeURIComponent(id)}`);
  const [params, setParams] = useSearchParams();
  const tab = (params.get('tab') as Tab) || 'overview';
  const { can, mode } = useSession();
  const [pending, setPending] = useState<NodeOperation | null>(null);
  const [confirm, setConfirm] = useState<NodeOperation | null>(null);
  const [confirmText, setConfirmText] = useState('');
  const [result, setResult] = useState<MutationResult | null>(null);
  const [opErr, setOpErr] = useState<ApiError | null>(null);

  const run = async (op: NodeOperation, n: NodeRec) => {
    setPending(op);
    setOpErr(null);
    setResult(null);
    try {
      const r = await api.post<{ data: MutationResult }>(`/nodes/${n.id}/operations`, { type: op, ...(op === 'REVOKE' ? { confirm: confirmText } : {}) });
      setResult(r.data);
      setConfirm(null);
      setConfirmText('');
      res.refresh();
    } catch (e) {
      setOpErr(e as ApiError);
    } finally {
      setPending(null);
    }
  };

  return (
    <div className="space-y-4 pt-2">
      <Link to="/nodes" className="inline-flex items-center gap-1.5 text-[12.5px] text-slate-400 hover:text-cyan-300">
        <ArrowLeft className="w-3.5 h-3.5" /> Nodes & Compute
      </Link>
      <Gate res={res}>
        {(d, stale) => {
          const n = d.node;
          const canWrite = can('api.write') && !stale && mode === 'controlplane';
          const canAdmin = can('api.admin') && !stale && mode === 'controlplane';
          const ops: { op: NodeOperation; label: string; icon: React.ReactNode; show: boolean; allowed: boolean; why: string }[] = [
            { op: 'APPROVE', label: 'Approve', icon: <CheckCircle2 className="w-4 h-4" />, show: n.lifecycle === 'PENDING_APPROVAL', allowed: canAdmin, why: 'needs api.admin' },
            { op: 'DRAIN', label: 'Drain', icon: <Pause className="w-4 h-4" />, show: n.lifecycle === 'ACTIVE', allowed: canWrite, why: 'needs api.write' },
            { op: 'UNDRAIN', label: 'Undrain', icon: <Play className="w-4 h-4" />, show: n.lifecycle === 'DRAINING', allowed: canWrite, why: 'needs api.write' },
            { op: 'REVOKE', label: 'Revoke identity', icon: <ShieldOff className="w-4 h-4" />, show: n.lifecycle !== 'REVOKED', allowed: canAdmin, why: 'needs api.admin' }
          ];
          return (
            <>
              <Glass className="p-5">
                <div className="flex flex-wrap items-start gap-4">
                  <IconTile tone={n.isEdge ? 'violet' : 'blue'} size="lg">{n.isEdge ? <Radio className="w-6 h-6" /> : <Server className="w-6 h-6" />}</IconTile>
                  <div className="min-w-0 flex-1">
                    <h1 className="text-2xl font-extrabold text-white tracking-tight">{n.name}</h1>
                    <div className="text-[11.5px] text-slate-500 font-mono break-all">{n.id}</div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <StatusPill status={n.health} />
                      <StatusPill status={n.lifecycle} dot={false} />
                      <FreshnessPill f={n.observation.freshness} ageMs={n.observation.ageMs} />
                      {n.isEdge && <StatusPill status="EDGE" tone="violet" dot={false} />}
                    </div>
                    <p className="mt-2 text-[12.5px] text-slate-400">{n.healthReason}. New work: {n.admission.newWork}; existing work: {n.admission.existingWork}.</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {ops
                      .filter((o) => o.show)
                      .map((o) => (
                        <GhostButton
                          key={o.op}
                          disabled={!o.allowed || pending !== null}
                          title={o.allowed ? undefined : stale ? 'data is stale' : o.why}
                          onClick={() => (o.op === 'REVOKE' || o.op === 'DRAIN' ? setConfirm(o.op) : run(o.op, n))}
                          className={`${o.op === 'REVOKE' ? '!border-rose-400/40 !text-rose-200' : ''} disabled:opacity-40`}
                        >
                          {o.icon} {pending === o.op ? 'Submitting…' : o.label}
                        </GhostButton>
                      ))}
                  </div>
                </div>
                {confirm && (
                  <div className="mt-4 rounded-xl border border-amber-400/30 bg-amber-500/10 p-4" role="alertdialog" aria-label={`Confirm ${confirm}`}>
                    <div className="text-[13px] font-semibold text-amber-100">
                      {confirm === 'REVOKE' ? 'Revoke this host identity? This cannot be undone from the console.' : 'Drain this host? The control plane stops placing work on it and reschedules replicas; it does not kill admitted work.'}
                    </div>
                    {confirm === 'REVOKE' && (
                      <label className="block mt-2 text-[12px] text-amber-100">
                        Type the node id to confirm
                        <input value={confirmText} onChange={(e) => setConfirmText(e.target.value)} className="mt-1 w-full rounded-lg bg-black/40 border border-amber-400/30 px-2 py-1.5 font-mono text-[12px]" />
                      </label>
                    )}
                    <div className="mt-3 flex gap-2">
                      <GhostButton onClick={() => run(confirm, n)} disabled={pending !== null || (confirm === 'REVOKE' && confirmText !== n.id)} className="!border-amber-400/50 disabled:opacity-40">
                        Confirm {confirm.toLowerCase()}
                      </GhostButton>
                      <GhostButton onClick={() => { setConfirm(null); setConfirmText(''); }}>Cancel</GhostButton>
                    </div>
                  </div>
                )}
                {result && (
                  <div className={`mt-3 rounded-xl p-3 text-[12.5px] border ${result.ok ? 'border-emerald-400/30 bg-emerald-500/10 text-emerald-100' : 'border-rose-400/30 bg-rose-500/10 text-rose-100'}`} role="status">
                    Control plane {result.ok ? 'committed' : 'refused'}: {result.message || result.code} <span className="font-mono text-[10.5px] opacity-70">· actor {result.actor} · request {result.request_id}</span>
                    {result.ok && <div className="text-[11.5px] opacity-80 mt-0.5">The new lifecycle appears below once the next view is served; observed effects follow from host observations.</div>}
                  </div>
                )}
                {opErr && <div className="mt-3"><ErrorState error={opErr} /></div>}
              </Glass>

              <PageTabs<Tab>
                tabs={[
                  { id: 'overview', label: 'Overview' },
                  { id: 'workloads', label: `Workloads (${d.replicas.length})` },
                  { id: 'facts', label: 'Facts & probes' },
                  { id: 'mesh', label: 'Mesh' },
                  { id: 'storage', label: 'Storage' },
                  { id: 'identity', label: 'Identity & ledger' },
                  { id: 'events', label: 'Events' },
                  { id: 'logs', label: 'Logs' }
                ]}
                active={tab}
                onChange={(t) => setParams((p) => (t === 'overview' ? (p.delete('tab'), p) : (p.set('tab', t), p)), { replace: true })}
              />

              {tab === 'overview' && (
                <div className="grid md:grid-cols-2 gap-4">
                  <Glass className="p-4">
                    <PanelHeader title="Placement" />
                    <div className="mt-2">
                      <Row k="Region / zone">{n.region || '—'} {n.zone && `/ ${n.zone}`}</Row>
                      <Row k="Host (failure domain)">{n.host || '—'}</Row>
                      <Row k="Tiers">{n.tiers.join(', ') || '—'}</Row>
                      <Row k="Roles">{n.roles.join(', ') || '—'}</Row>
                      <Row k="Map position">{n.location ? <>{n.location.lat}, {n.location.lng} <TruthTag state="CONFIGURED" /></> : 'not configured'}</Row>
                      <Row k="Joined / approved">{fmtTime(n.joinedAt)} / {fmtTime(n.approvedAt)}</Row>
                    </div>
                  </Glass>
                  <Glass className="p-4">
                    <PanelHeader title="Capacity" subtitle="Declared at enrolment vs measured by the host." />
                    <div className="mt-2">
                      <Row k="CPU declared">{n.declared.cpuMilli / 1000} cores <TruthTag state="CONFIGURED" /></Row>
                      <Row k="CPU measured">{n.facts ? `${n.facts.cpus} logical` : '—'}</Row>
                      <Row k="Memory declared">{fmtBytes(n.declared.memBytes)} <TruthTag state="CONFIGURED" /></Row>
                      <Row k="Memory measured">{n.facts?.memBytes != null ? fmtBytes(n.facts.memBytes) : 'not measured'}</Row>
                      <Row k="Workloads">{n.workloads ?? '—'}</Row>
                      <Row k="Mode">{n.mode || '—'} {n.modeDetail}</Row>
                    </div>
                  </Glass>
                  <Glass className="p-4 md:col-span-2">
                    <PanelHeader title="Hardware" subtitle="Measured by the host agent and signed in its observation. Nothing here is declared or inferred." />
                    {n.facts?.unknown === null || !n.facts ? (
                      <p className="text-[12.5px] text-slate-400 mt-2">{n.facts ? 'This host agent predates measured hardware facts; upgrade it to report CPU model, disks and GPUs.' : 'No observation yet.'}</p>
                    ) : (
                      <div className="mt-2 grid sm:grid-cols-2 gap-x-6">
                        <Row k="CPU model">{n.facts.cpuModel ?? 'not measured'}</Row>
                        <Row k="Physical cores">{n.facts.physicalCores ?? 'not measured'}</Row>
                        <Row k="Swap">{n.facts.swapBytes === null ? 'not measured' : fmtBytes(n.facts.swapBytes)}</Row>
                        <Row k="Data filesystem">{n.facts.dataFs ? `${fmtBytes(n.facts.dataFs.freeBytes)} free of ${fmtBytes(n.facts.dataFs.totalBytes)}` : 'not measured'}</Row>
                        <Row k="Disks">
                          {n.facts.disks === null
                            ? 'not measured'
                            : n.facts.disks.length === 0
                              ? 'none visible'
                              : n.facts.disks.map((x) => `${x.name} ${fmtBytes(x.sizeBytes)}${x.rotational ? ' HDD' : ''}${x.removable ? ' removable' : ''}`).join(', ')}
                        </Row>
                        <Row k="GPUs">
                          {n.facts.gpus === null
                            ? 'not measured'
                            : n.facts.gpus.length === 0
                              ? 'none found'
                              : n.facts.gpus.map((g) => `${g.vendor}${g.model ? ` ${g.model}` : ''}${g.vramBytes ? ` ${fmtBytes(g.vramBytes)}` : ''} (${g.source})`).join(', ')}
                        </Row>
                        <Row k="Could not measure">{n.facts.unknown.length ? n.facts.unknown.join(', ') : 'nothing'}</Row>
                      </div>
                    )}
                  </Glass>
                  <Glass className="p-4 md:col-span-2">
                    <PanelHeader title="Host policy" subtitle="Reported by the host; the control plane can read it but never change it." />
                    {n.policy ? (
                      <div className="mt-2 grid sm:grid-cols-2 gap-x-6">
                        {Object.entries(n.policy).map(([k, v]) => (
                          <Row key={k} k={k}>{Array.isArray(v) ? v.join(', ') || '—' : typeof v === 'number' && /Bytes$/.test(k) ? fmtBytes(v) : String(v)}</Row>
                        ))}
                      </div>
                    ) : (
                      <p className="text-[12.5px] text-slate-400 mt-2">No policy reported.</p>
                    )}
                  </Glass>
                </div>
              )}

              {tab === 'workloads' &&
                (d.replicas.length ? (
                  <Glass className="overflow-x-auto">
                    <table className="dh-table w-full min-w-[760px]">
                      <thead><tr><th>Assignment</th><th>Desired</th><th>Admitted</th><th>Observed</th><th>Health</th><th>Freshness</th><th>Logs</th></tr></thead>
                      <tbody>
                        {d.replicas.map((r) => (
                          <tr key={r.assignment}>
                            <td><Link to={`/apps/${encodeURIComponent(r.app)}`} className="text-cyan-300 hover:underline">{r.assignment}</Link> <span className="text-slate-500">gen {r.desiredGen}</span></td>
                            <td><StatusPill status={r.desired} dot={false} /></td>
                            <td title={r.reason}><StatusPill status={r.admitted} dot={false} /></td>
                            <td><StatusPill status={r.observed} dot={false} /></td>
                            <td className="text-slate-300">{r.health ? `${r.health.ok ? 'ok' : 'failing'} · ${r.health.detail}` : '—'}</td>
                            <td><FreshnessPill f={r.freshness} /></td>
                            <td><Link to={`?tab=logs&assignment=${encodeURIComponent(r.assignment)}`} className="text-cyan-300 hover:underline text-[12px]">view</Link></td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </Glass>
                ) : (
                  <Empty title="No assignments on this host" />
                ))}

              {tab === 'facts' && (
                <Glass className="p-4">
                  {n.facts ? (
                    <>
                      <Row k="Kernel">{n.facts.kernel}</Row>
                      <Row k="OS / arch">{n.os} / {n.arch}</Row>
                      <Row k="Runtimes">{n.facts.runtimes.join(', ') || '—'}</Row>
                      <Row k="Docker">{n.facts.docker || 'not available'}</Row>
                      <Row k="UDP 443">{n.facts.udp443 ? 'listening' : 'not listening'}</Row>
                      <Row k="Clock skew">{n.facts.clockSkewMs} ms</Row>
                      <h4 className="mt-4 mb-1 text-[12px] font-semibold text-slate-300">Probes</h4>
                      {n.facts.probes.map((p) => (
                        <Row key={p.name} k={p.name}><StatusPill status={p.ok ? 'AVAILABLE' : 'NOT AVAILABLE'} tone={p.ok ? 'emerald' : 'slate'} dot={false} /> <span className="text-slate-400 text-[11.5px]">{p.detail}</span></Row>
                      ))}
                      {d.diagnostics.length > 0 && <h4 className="mt-4 mb-1 text-[12px] font-semibold text-slate-300">Diagnostics</h4>}
                      {d.diagnostics.map((x) => (
                        <Row key={x.item} k={x.item}>{x.value} <TruthTag state={x.basis === 'OBSERVED' ? 'LIVE' : 'DERIVED'} /></Row>
                      ))}
                    </>
                  ) : (
                    <Empty title="No facts reported" detail="The host has not sent a signed observation yet." />
                  )}
                </Glass>
              )}

              {tab === 'mesh' && (
                <Glass className="p-4">
                  {n.mesh ? (
                    <>
                      <Row k="Mesh IP">{n.mesh.meshIp}</Row>
                      <Row k="Device">{n.mesh.device}</Row>
                      <Row k="Peers (gossip alive)">{n.mesh.peersAlive} / {n.mesh.peers}</Row>
                      <Row k="Handshakes in the last 3 min">{n.mesh.handshakesRecent}</Row>
                    </>
                  ) : (
                    <Empty title="No mesh observation" />
                  )}
                </Glass>
              )}

              {tab === 'storage' && (
                <div className="space-y-3">
                  <Glass className="p-4">
                    {n.storage ? (
                      <>
                        <Row k="Used by volumes">{fmtBytes(n.storage.usedBytes)}</Row>
                        <Row k="Quota">{fmtBytes(n.storage.quotaBytes)}</Row>
                        <Row k="Filesystem free / capacity">{fmtBytes(n.storage.freeBytes)} / {fmtBytes(n.storage.capacityBytes)}</Row>
                        <Row k="Chunks held">{n.storage.chunks}</Row>
                        <Row k="Corrupt chunks">{n.storage.corrupt}</Row>
                      </>
                    ) : (
                      <Empty title="No storage observation" />
                    )}
                  </Glass>
                  {d.volumes.length > 0 && (
                    <Glass className="p-4">
                      <PanelHeader title="Volume replicas on this host" />
                      {d.volumes.map((v) => (
                        <Row key={v.id} k={v.id}><StatusPill status={v.state} /> {v.verified}/{v.durabilityReplicas} verified</Row>
                      ))}
                    </Glass>
                  )}
                </div>
              )}

              {tab === 'identity' && (
                <Glass className="p-4">
                  <Row k="Identity">{n.identity}</Row>
                  <Row k="Host ledger head">{n.ledger ? `#${n.ledger.seq} ${shortDigest(n.ledger.hash)}` : '—'}</Row>
                  <Row k="Last observation evidence">{shortDigest(n.observation.evidence)}</Row>
                  <Row k="Observation seq / source">{n.observation.seq} · {n.observation.source}</Row>
                  <Row k="Revoked at">{fmtTime(n.revokedAt)}</Row>
                  <h4 className="mt-4 mb-1 text-[12px] font-semibold text-slate-300 flex items-center gap-1.5"><KeyRound className="w-3.5 h-3.5" /> Public keys</h4>
                  {n.keys.map((k) => (
                    <Row key={k.pub} k={k.revoked ? 'revoked' : 'active'}><span className="font-mono text-[11px]">{k.pub}</span>{k.reason && <span className="text-slate-400"> — {k.reason}</span>}</Row>
                  ))}
                  <Note>Private keys never leave the host. Verify its ledger independently with <code className="text-cyan-200">dh audit host {n.name}</code>.</Note>
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
                  <Empty title="No audit entries reference this host in the recent window" />
                ))}

              {tab === 'logs' && <LogsPanel nodeId={n.id} replicas={d.replicas} />}
            </>
          );
        }}
      </Gate>
    </div>
  );
}

function LogsPanel({ nodeId, replicas }: { nodeId: string; replicas: ReplicaRec[] }) {
  const [params, setParams] = useSearchParams();
  const assignment = params.get('assignment') ?? replicas[0]?.assignment ?? '';
  const res = useResource<{ lines: string[] }>(assignment ? `/nodes/${nodeId}/logs?assignment=${encodeURIComponent(assignment)}&tail=300` : null, { pollMs: 5000 });
  if (!replicas.length) return <Empty title="No workloads on this host to show logs for" />;
  return (
    <Glass className="p-4 space-y-3">
      <div className="flex items-center gap-2 flex-wrap">
        <Terminal className="w-4 h-4 text-cyan-300" />
        {replicas.map((r) => (
          <button
            key={r.assignment}
            onClick={() => setParams((p) => (p.set('assignment', r.assignment), p), { replace: true })}
            className={`px-2.5 py-1 rounded-lg text-[12px] border ${r.assignment === assignment ? 'border-cyan-400/60 text-white bg-cyan-500/10' : 'border-[rgba(125,190,255,0.16)] text-slate-300'}`}
          >
            {r.assignment}
          </button>
        ))}
      </div>
      <Gate res={res}>
        {(d) => (
          <pre className="max-h-[480px] overflow-auto rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.12)] p-3 text-[11.5px] leading-relaxed text-slate-200 font-mono whitespace-pre-wrap">
            {d.lines.length ? d.lines.join('\n') : '(no output)'}
          </pre>
        )}
      </Gate>
      <Note>Logs are relayed over the WireGuard mesh from the host; the last 300 lines are shown.</Note>
    </Glass>
  );
}
