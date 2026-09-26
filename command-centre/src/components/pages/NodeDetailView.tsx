import React, { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Server, Radio, Pause, Play, CheckCircle2, ShieldOff, Terminal, KeyRound } from 'lucide-react';
import type { AuditEntryRec, NodeOperation, NodeRec, ReplicaRec, VolumeRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError, type MutationResult } from '../../lib/client';
import { Glass, PageTabs, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { LogViewer } from '../common/LogViewer';
import { Gate, FreshnessPill, fmtBytes, fmtTime, fmtAge, since, shortDigest, ErrorState, Empty, Note, TruthTag, Unavailable } from '../common/states';
import { Digest, FreshnessBadge, viewFreshness } from '../common/truth';

interface Payload {
  node: NodeRec;
  replicas: (ReplicaRec & { app: string })[];
  events: AuditEntryRec[];
  diagnostics: { subject: string; item: string; value: string; basis: string; detail: string }[];
  volumes: VolumeRec[];
  allocated: { cpuMilli: number; memBytes: number; replicas: number; undeclared: number };
}

type Tab = 'overview' | 'workloads' | 'resources' | 'network' | 'contribution' | 'depin' | 'security' | 'logs' | 'evidence' | 'settings';
const LEGACY: Record<string, Tab> = { facts: 'resources', storage: 'resources', mesh: 'network', identity: 'security', events: 'evidence' };

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
  const raw = params.get('tab') ?? 'overview';
  const tab = (LEGACY[raw] ?? raw) as Tab;
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
            { op: 'UNDRAIN', label: 'Resume', icon: <Play className="w-4 h-4" />, show: n.lifecycle === 'DRAINING', allowed: canWrite, why: 'needs api.write' },
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
                  { id: 'resources', label: 'Resources' },
                  { id: 'network', label: 'Network' },
                  { id: 'contribution', label: 'Contribution' },
                  { id: 'depin', label: 'DePIN' },
                  { id: 'security', label: 'Security' },
                  { id: 'logs', label: 'Logs' },
                  { id: 'evidence', label: 'Evidence' },
                  { id: 'settings', label: 'Settings' }
                ]}
                active={tab}
                onChange={(t) => setParams((p) => (t === 'overview' ? (p.delete('tab'), p) : (p.set('tab', t), p)), { replace: true })}
              />

              {tab === 'overview' && (
                <div className="grid md:grid-cols-2 gap-4 [&>*]:min-w-0">
                  <Glass className="p-4">
                    <PanelHeader title="Identity" subtitle="Generated on the host; only the public key is known here." />
                    <div className="mt-2">
                      <Row k="Node id">{n.id}</Row>
                      <Row k="Public key">{n.keys.find((k) => !k.revoked) ? <Digest value={n.keys.find((k) => !k.revoked)!.pub} chars={20} /> : 'no valid key'}</Row>
                      <Row k="Lifecycle"><StatusPill status={n.lifecycle} dot={false} /></Row>
                      <Row k="Freshness"><FreshnessBadge f={viewFreshness({ pageStale: stale, observation: n.observation.freshness, lost: n.health === 'OFFLINE' && n.lifecycle !== 'REVOKED' })} observedAt={n.observation.observedAt} /></Row>
                      <Row k="Agent version"><span className="text-slate-400">not reported by the agent</span></Row>
                      <Row k="OS / kernel / arch">{n.os} · {n.facts?.kernel || '—'} · {n.arch}</Row>
                      <Row k="Uptime">{n.facts?.uptimeSec != null ? fmtAge(n.facts.uptimeSec * 1000) : 'not measured'}</Row>
                    </div>
                  </Glass>
                  <Glass className="p-4">
                    <PanelHeader title="Placement" />
                    <div className="mt-2">
                      <Row k="Region / zone">{n.region || '—'} {n.zone && `/ ${n.zone}`}</Row>
                      <Row k="Host (failure domain)">{n.host || '—'}</Row>
                      <Row k="Operator">this cluster (all hosts share one root)</Row>
                      <Row k="Tiers">{n.tiers.join(', ') || '—'}</Row>
                      <Row k="Roles">{n.roles.join(', ') || '—'}</Row>
                      <Row k="Map position">{n.location ? <>{n.location.lat}, {n.location.lng} <TruthTag state="CONFIGURED" /></> : 'not configured'}</Row>
                      <Row k="Joined / approved">{fmtTime(n.joinedAt)} / {fmtTime(n.approvedAt)}</Row>
                      <Row k="Mode">{n.mode || '—'} {n.modeDetail}</Row>
                    </div>
                  </Glass>
                  <Glass className="p-4 md:col-span-2">
                    <PanelHeader title="Hardware" subtitle="Measured by the host agent and signed in its observation. Nothing here is declared or inferred." />
                    {!n.facts || n.facts.unknown === null ? (
                      <p className="text-[12.5px] text-slate-400 mt-2">{n.facts ? 'This host agent predates measured hardware facts: memory, disks and GPUs show as NOT MEASURED. Upgrade the agent.' : 'No observation yet.'}</p>
                    ) : (
                      <div className="mt-2 grid sm:grid-cols-2 gap-x-6">
                        <Row k="CPU">{n.facts.cpus} logical{n.facts.physicalCores ? ` · ${n.facts.physicalCores} physical` : ''}{n.facts.cpuModel ? ` · ${n.facts.cpuModel}` : ''}</Row>
                        <Row k="Measured RAM">{n.facts.memBytes != null ? fmtBytes(n.facts.memBytes) : 'NOT MEASURED'}</Row>
                        <Row k="Swap">{n.facts.swapBytes === null ? 'NOT MEASURED' : fmtBytes(n.facts.swapBytes)}</Row>
                        <Row k="Filesystem (agent data)">{n.facts.dataFs ? `${fmtBytes(n.facts.dataFs.freeBytes)} free of ${fmtBytes(n.facts.dataFs.totalBytes)}` : 'NOT MEASURED'}</Row>
                        <Row k="Storage devices">
                          {n.facts.disks === null ? 'NOT MEASURED' : n.facts.disks.length === 0 ? 'none visible' : n.facts.disks.map((x) => `${x.name} ${fmtBytes(x.sizeBytes)}${x.rotational ? ' HDD' : ''}${x.removable ? ' removable' : ''}`).join(', ')}
                        </Row>
                        <Row k="GPU">
                          {n.facts.gpus === null ? 'NOT MEASURED' : n.facts.gpus.length === 0 ? 'none found' : n.facts.gpus.map((g) => `${g.vendor}${g.model ? ` ${g.model}` : ''}${g.vramBytes ? ` ${fmtBytes(g.vramBytes)}` : ''} (${g.source})`).join(', ')}
                        </Row>
                        <Row k="Could not measure">{n.facts.unknown.length ? n.facts.unknown.join(', ') : 'nothing'}</Row>
                      </div>
                    )}
                  </Glass>
                </div>
              )}

              {tab === 'workloads' &&
                (d.replicas.length ? (
                  <Glass className="overflow-x-auto">
                    <table className="dh-table w-full min-w-[760px]">
                      <thead><tr><th>Assignment</th><th>Origin</th><th>Desired</th><th>Admitted</th><th>Observed</th><th>Health</th><th>Freshness</th><th>Logs</th></tr></thead>
                      <tbody>
                        {d.replicas.map((r) => (
                          <tr key={r.assignment}>
                            <td><Link to={`/apps/${encodeURIComponent(r.app)}`} className="text-cyan-300 hover:underline">{r.assignment}</Link> <span className="text-slate-500">gen {r.desiredGen}</span></td>
                            <td><StatusPill status="OWNER" tone="cyan" dot={false} /></td>
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

              {tab === 'resources' && (() => {
                // Available is bounded by what was measured and by what the host's policy admits, whichever is smaller.
                const cap = (policy: unknown, measured: number | null) => {
                  const p = typeof policy === 'number' && policy > 0 ? policy : null;
                  if (p === null) return measured;
                  return measured === null ? p : Math.min(p, measured);
                };
                const cpuCeil = cap(n.policy?.maxCpuMilli, n.facts ? n.facts.cpus * 1000 : null);
                const memCeil = cap(n.policy?.maxMemBytes, n.facts?.memBytes ?? null);
                return (
                <div className="grid md:grid-cols-2 gap-4 [&>*]:min-w-0">
                  <Glass className="p-4 md:col-span-2 overflow-x-auto">
                    <PanelHeader title="Resources" subtitle="Each column has its own source. Nothing is filled in to make the row add up." />
                    <table className="dh-table w-full min-w-[680px] mt-2">
                      <thead><tr><th /><th>Total (measured)</th><th>Owner reserved</th><th>Host admits (policy)</th><th>Allocated</th><th>Available</th></tr></thead>
                      <tbody>
                        <tr>
                          <td className="text-slate-300">CPU</td>
                          <td>{n.facts ? `${n.facts.cpus} logical` : 'not measured'}</td>
                          <td className="text-slate-500">no record</td>
                          <td>{typeof n.policy?.maxCpuMilli === 'number' && n.policy.maxCpuMilli > 0 ? <>{Number(n.policy.maxCpuMilli) / 1000} cores <TruthTag state="CONFIGURED" /></> : <span className="text-slate-500">no cap</span>}</td>
                          <td>{d.allocated.cpuMilli / 1000} cores <TruthTag state="DERIVED" /></td>
                          <td>{cpuCeil !== null ? <>{Math.max(0, cpuCeil - d.allocated.cpuMilli) / 1000} cores <TruthTag state="DERIVED" /></> : <span className="text-slate-500">unknown</span>}</td>
                        </tr>
                        <tr>
                          <td className="text-slate-300">Memory</td>
                          <td>{n.facts?.memBytes != null ? fmtBytes(n.facts.memBytes) : 'not measured'}</td>
                          <td className="text-slate-500">no record</td>
                          <td>{typeof n.policy?.maxMemBytes === 'number' && n.policy.maxMemBytes > 0 ? <>{fmtBytes(Number(n.policy.maxMemBytes))} <TruthTag state="CONFIGURED" /></> : <span className="text-slate-500">no cap</span>}</td>
                          <td>{fmtBytes(d.allocated.memBytes)} <TruthTag state="DERIVED" /></td>
                          <td>{memCeil !== null ? <>{fmtBytes(Math.max(0, memCeil - d.allocated.memBytes))} <TruthTag state="DERIVED" /></> : <span className="text-slate-500">unknown</span>}</td>
                        </tr>
                      </tbody>
                    </table>
                    <Note>
                      Allocated is the sum of the manifest requests of the {d.allocated.replicas} replica(s) desired on this host{d.allocated.undeclared ? `; ${d.allocated.undeclared} declare no request and count as 0` : ''}. Available = the smaller of measured total and the policy limit, minus allocated. Declared capacity at enrolment: {n.declared.cpuMilli / 1000} cores, {fmtBytes(n.declared.memBytes)} (CONFIGURED). The platform has no owner-reserve record yet, so none is shown.
                    </Note>
                  </Glass>
                  <Glass className="p-4">
                    <PanelHeader title="Storage" />
                    {n.storage ? (
                      <div className="mt-2">
                        <Row k="Used by volumes">{fmtBytes(n.storage.usedBytes)}</Row>
                        <Row k="Quota (host policy)">{fmtBytes(n.storage.quotaBytes)}</Row>
                        <Row k="Filesystem free / capacity">{fmtBytes(n.storage.freeBytes)} / {fmtBytes(n.storage.capacityBytes)}</Row>
                        <Row k="Chunks held">{n.storage.chunks}</Row>
                        <Row k="Corrupt chunks">{n.storage.corrupt}</Row>
                      </div>
                    ) : (
                      <p className="mt-2 text-[12.5px] text-slate-400">No storage observation.</p>
                    )}
                    {d.volumes.map((v) => (
                      <Row key={v.id} k={v.id}><StatusPill status={v.state} /> {v.verified}/{v.durabilityReplicas} verified</Row>
                    ))}
                  </Glass>
                  <Glass className="p-4">
                    <PanelHeader title="Runtimes & probes" />
                    {n.facts ? (
                      <div className="mt-2">
                        <Row k="Runtimes">{n.facts.runtimes.join(', ') || '—'}</Row>
                        <Row k="Docker">{n.facts.docker || 'not available'}</Row>
                        <Row k="Clock skew">{n.facts.clockSkewMs} ms</Row>
                        {n.facts.probes.map((p) => (
                          <Row key={p.name} k={p.name}><StatusPill status={p.ok ? 'AVAILABLE' : 'NOT AVAILABLE'} tone={p.ok ? 'emerald' : 'slate'} dot={false} /> <span className="text-slate-400 text-[11.5px]">{p.detail}</span></Row>
                        ))}
                        {d.diagnostics.map((x) => (
                          <Row key={x.item} k={x.item}>{x.value} <TruthTag state={x.basis === 'OBSERVED' ? 'LIVE' : 'DERIVED'} /></Row>
                        ))}
                      </div>
                    ) : (
                      <p className="mt-2 text-[12.5px] text-slate-400">No facts reported.</p>
                    )}
                  </Glass>
                </div>
                );
              })()}

              {tab === 'network' && (
                <Glass className="p-4">
                  <PanelHeader title="Network" subtitle="Separate layers, separately observed. A WireGuard handshake does not prove transport reachability or membership." />
                  <div className="mt-2">
                    <Row k="Mesh address">{n.mesh?.meshIp || '—'}</Row>
                    <Row k="WireGuard device">{n.mesh ? n.mesh.device : 'no observation'}</Row>
                    <Row k="WireGuard handshakes (< 3 min)">{n.mesh ? `${n.mesh.handshakesRecent} peer(s)` : '—'}</Row>
                    <Row k="Membership (gossip alive)">{n.mesh ? `${n.mesh.peersAlive} / ${n.mesh.peers}` : '—'}</Row>
                    <Row k="Transport reachability">UNKNOWN (not measured per peer)</Row>
                    <Row k="Edge UDP 443">{n.facts ? (n.facts.udp443 ? 'listening' : 'not listening') : '—'}</Row>
                  </div>
                  <div className="mt-3"><Unavailable title="Interface and NAT discovery" detail="Hosts do not report network interfaces, public reachability or NAT type yet (facts.unknown lists natType)." /></div>
                </Glass>
              )}

              {tab === 'contribution' && (
                <Unavailable title="Contribution policy" state="PLANNED" detail="This host serves its owner only. There is no community or marketplace contribution, and nothing is contributed by default. Capacity can be shared with a peer cluster only through a root-signed federation grant, which the host's own policy (allowFederated) can still refuse." />
              )}

              {tab === 'depin' && <Unavailable title="External DePIN networks" detail="No DePIN adapter is implemented. Nothing is installed on this host and no rewards are reported." />}

              {tab === 'security' && (
                <div className="grid md:grid-cols-2 gap-4 [&>*]:min-w-0">
                  <Glass className="p-4">
                    <PanelHeader title="Identity & keys" />
                    <div className="mt-2">
                      <Row k="Identity">{n.identity}</Row>
                      <Row k="Revoked at">{fmtTime(n.revokedAt)}</Row>
                      <Row k="New work / existing work">{n.admission.newWork} / {n.admission.existingWork}</Row>
                    </div>
                    <h4 className="mt-4 mb-1 text-[12px] font-semibold text-slate-300 flex items-center gap-1.5"><KeyRound className="w-3.5 h-3.5" /> Public keys</h4>
                    {n.keys.map((k) => (
                      <Row key={k.pub} k={k.revoked ? 'revoked' : 'active'}><Digest value={k.pub} chars={20} />{k.reason && <span className="text-slate-400"> — {k.reason}</span>}</Row>
                    ))}
                    <Note>Private keys never leave the host.</Note>
                  </Glass>
                  <Glass className="p-4">
                    <PanelHeader title="Host policy" subtitle="Signed by the host. The control plane can read it but never change it." />
                    {n.policy ? (
                      <div className="mt-2">
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

              {tab === 'evidence' && (
                <Glass className="p-4">
                  <PanelHeader title="Evidence" subtitle="The host's own hash-chained ledger, its signed observations, and audit entries that name it." />
                  <div className="mt-2">
                    <Row k="Host ledger head">{n.ledger ? <>#{n.ledger.seq} <Digest value={n.ledger.hash} /></> : '—'}</Row>
                    <Row k="Last observation">{n.observation.evidence ? <Digest value={n.observation.evidence} /> : '—'} · seq {n.observation.seq} · {n.observation.source}</Row>
                  </div>
                  <h4 className="mt-4 mb-1 text-[12px] font-semibold text-slate-300">Audit entries</h4>
                  {d.events.length ? (
                    d.events.map((e) => (
                      <div key={e.seq} className="py-2 border-b border-[rgba(125,190,255,0.06)] last:border-0">
                        <div className="text-[12.5px] text-slate-100"><span className="font-mono text-slate-500">#{e.seq}</span> {e.action} <span className="text-slate-400">{e.resource}</span></div>
                        <div className="text-[11.5px] text-slate-400">{e.detail} · {e.actor} · {since(e.ts)}</div>
                      </div>
                    ))
                  ) : (
                    <p className="text-[12.5px] text-slate-400">No audit entries reference this host in the recent window.</p>
                  )}
                  <Note>Verify the host ledger independently with <code className="text-cyan-200">dh audit host {n.name}</code>.</Note>
                </Glass>
              )}

              {tab === 'settings' && (
                <Glass className="p-4">
                  <PanelHeader title="Settings" subtitle="Where each setting lives. The console changes only lifecycle (approve, drain, undrain, revoke)." />
                  <div className="mt-2">
                    <Row k="Name, region, zone, roles">set on the host when dh-noded starts (--name, --region, --zone, --roles)</Row>
                    <Row k="Admission policy">the host's own policy.yaml; the control plane cannot change it</Row>
                    <Row k="Declared capacity">--cpu / --mem on the host (CONFIGURED)</Row>
                    <Row k="Key rotation">dh-noded rotate-key on the host; emergency revocation with dh node revoke-key</Row>
                    <Row k="Cordon">not implemented; drain stops new placement and reschedules replicas</Row>
                  </div>
                </Glass>
              )}

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
  if (!replicas.length) return <Empty title="No workloads on this host to show logs for" />;
  return (
    <Glass className="p-4 space-y-3">
      <div className="flex items-center gap-2 flex-wrap" role="tablist" aria-label="Workload">
        {replicas.map((r) => (
          <button
            key={r.assignment}
            role="tab"
            aria-selected={r.assignment === assignment}
            onClick={() => setParams((p) => (p.set('assignment', r.assignment), p), { replace: true })}
            className={`px-2.5 py-1 rounded-lg text-[12px] border focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 ${r.assignment === assignment ? 'border-cyan-400/60 text-white bg-cyan-500/10' : 'border-[rgba(125,190,255,0.16)] text-slate-300'}`}
          >
            {r.assignment}
          </button>
        ))}
      </div>
      {assignment && <LogViewer key={assignment} path={`/nodes/${nodeId}/logs?assignment=${encodeURIComponent(assignment)}`} source={`${assignment}@${nodeId.slice(0, 10)}`} />}
      <Note>Logs are relayed over the WireGuard mesh from the host on request.</Note>
    </Glass>
  );
}
