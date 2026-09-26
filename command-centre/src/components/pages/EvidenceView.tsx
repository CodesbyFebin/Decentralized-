import React, { useEffect, useState } from 'react';
import { FileCheck2, RefreshCw, ShieldCheck, ShieldX, Package, Target } from 'lucide-react';
import type { ArtifactRec, AuditEntryRec, LedgerVerification, MilestoneRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/client';
import { Glass, PanelHeader, StatusPill, IconTile, GhostButton } from '../common/ui';
import { Gate, shortDigest, since, fmtTime, ErrorState, Note, TruthTag } from '../common/states';

interface Payload {
  verification: LedgerVerification;
  head: number;
  milestones: MilestoneRec[];
  artifacts: ArtifactRec[];
  chaos: { id: string; scenario: string; verdict: string; received: number; signer: string; evidence: string }[];
}

export default function EvidenceView() {
  const res = useResource<Payload>('/evidence', { pollMs: 15_000 });
  const [verify, setVerify] = useState<LedgerVerification | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [verr, setVerr] = useState<ApiError | null>(null);
  const [entries, setEntries] = useState<AuditEntryRec[]>([]);
  const [nextBefore, setNextBefore] = useState<number | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [lerr, setLerr] = useState<ApiError | null>(null);

  const load = async (before: number | null) => {
    setLoadingMore(true);
    try {
      const r = await api.get<{ data: { entries: AuditEntryRec[]; nextBefore: number | null } }>(`/audit?limit=50${before ? `&before=${before}` : ''}`);
      setEntries((e) => (before ? [...e, ...r.data.entries] : r.data.entries));
      setNextBefore(r.data.nextBefore);
      setLerr(null);
    } catch (e) {
      setLerr(e as ApiError);
    } finally {
      setLoadingMore(false);
    }
  };
  useEffect(() => {
    load(null);
  }, []);

  const reverify = async () => {
    setVerifying(true);
    setVerr(null);
    try {
      const r = await api.post<{ data: LedgerVerification }>('/evidence/verify');
      setVerify(r.data);
    } catch (e) {
      setVerr(e as ApiError);
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="space-y-4 pt-2">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Evidence Ledger</h1>
        <p className="text-[13px] text-slate-400">Append-only, hash-chained audit ledger replicated by Raft, with signed host evidence. Nothing here is editable.</p>
      </div>
      <Gate res={res}>
        {(d) => {
          const v = verify ?? d.verification;
          return (
            <>
              <Glass className="p-5">
                <div className="flex flex-wrap items-start gap-4">
                  <IconTile tone={v.state === 'VERIFIED' ? 'emerald' : v.state === 'INVALID' ? 'rose' : 'slate'} size="lg">{v.state === 'INVALID' ? <ShieldX className="w-6 h-6" /> : <ShieldCheck className="w-6 h-6" />}</IconTile>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2"><span className="text-lg font-bold text-white">Ledger verification</span><StatusPill status={v.state} /></div>
                    <div className="text-[12.5px] text-slate-300 mt-1">{v.entries} entries · {v.checkpoints} signed checkpoint(s) · head <span className="font-mono">{shortDigest(v.head)}</span></div>
                    <div className="text-[12px] text-slate-400">verified {fmtTime(v.verifiedAt)} by {v.verifiedBy || '—'}{verify && ' (re-verified just now)'}</div>
                    {v.breakDetail && <div className="mt-2 text-[12px] text-rose-200 font-mono break-all">{v.breakDetail}</div>}
                  </div>
                  <GhostButton onClick={reverify} disabled={verifying}><RefreshCw className={`w-4 h-4 ${verifying ? 'animate-spin' : ''}`} /> Re-verify now</GhostButton>
                </div>
                {verr && <div className="mt-3"><ErrorState error={verr} /></div>}
                <Note>Verify independently: <code className="text-cyan-200">dh audit verify</code> fetches the ledger and checks the chain and checkpoints locally.</Note>
              </Glass>

              <div className="grid lg:grid-cols-2 gap-4">
                <Glass className="p-4">
                  <PanelHeader icon={<IconTile tone="violet" size="sm"><Target className="w-4 h-4" /></IconTile>} title="Milestone evidence" subtitle="Derived by the control plane from live state; gaps are listed, not hidden." />
                  <ul className="mt-3 space-y-3">
                    {d.milestones.map((m) => (
                      <li key={m.id}>
                        <div className="flex items-center gap-2"><span className="font-mono text-[11px] text-slate-500">{m.id}</span><span className="text-[13px] text-slate-100 font-semibold">{m.title}</span><StatusPill status={m.state} dot={false} tone={m.state === 'VERIFIED' || m.state === 'OPERATIONAL' ? 'emerald' : m.state === 'PARTIAL' ? 'amber' : 'slate'} /><TruthTag state={m.basis === 'OBSERVED' ? 'LIVE' : 'DERIVED'} /></div>
                        <ul className="mt-1 ml-4 list-disc text-[11.5px] text-slate-400">{m.evidence.map((e) => <li key={e}>{e}</li>)}</ul>
                        {m.gaps.length > 0 && <ul className="mt-1 ml-4 list-disc text-[11.5px] text-amber-200">{m.gaps.map((g) => <li key={g}>gap: {g}</li>)}</ul>}
                      </li>
                    ))}
                  </ul>
                </Glass>
                <div className="space-y-4">
                  <Glass className="p-4">
                    <PanelHeader icon={<IconTile tone="cyan" size="sm"><Package className="w-4 h-4" /></IconTile>} title="Artifact attestations" />
                    <ul className="mt-2 space-y-2 text-[12px]">
                      {d.artifacts.map((a) => (
                        <li key={a.digest}><span className="text-slate-100 font-semibold">{a.name}</span> <StatusPill status={a.attested ? 'ATTESTED' : 'UNSIGNED'} tone={a.attested ? 'emerald' : 'amber'} dot={false} /><div className="font-mono text-[10.5px] text-slate-500 break-all">{a.digest}</div></li>
                      ))}
                      {!d.artifacts.length && <li className="text-slate-400">No artifacts.</li>}
                    </ul>
                  </Glass>
                  <Glass className="p-4">
                    <PanelHeader title="Chaos reports" subtitle="Signed reports submitted with dh chaos run --submit." />
                    <ul className="mt-2 space-y-1.5 text-[12px]">
                      {d.chaos.map((c) => <li key={c.id} className="flex justify-between gap-2"><span className="text-slate-200">{c.scenario}</span><StatusPill status={c.verdict} /><span className="text-slate-500">{since(c.received)}</span></li>)}
                      {!d.chaos.length && <li className="text-slate-400">No chaos reports submitted to this cluster.</li>}
                    </ul>
                  </Glass>
                </div>
              </div>

              <Glass className="overflow-x-auto">
                <div className="px-4 pt-4"><PanelHeader icon={<IconTile tone="emerald" size="sm"><FileCheck2 className="w-4 h-4" /></IconTile>} title="Audit ledger" subtitle={`head #${d.head} · newest first · paged from the control plane`} /></div>
                {lerr && <div className="p-4"><ErrorState error={lerr} onRetry={() => load(null)} /></div>}
                <table className="dh-table w-full min-w-[980px] mt-2">
                  <thead><tr><th>#</th><th>Time</th><th>Action</th><th>Resource</th><th>Actor</th><th>Detail</th><th>Evidence</th><th>Hash</th></tr></thead>
                  <tbody>
                    {entries.map((e) => (
                      <tr key={e.seq}>
                        <td className="font-mono text-slate-500">{e.seq}</td>
                        <td className="text-slate-300">{since(e.ts)}</td>
                        <td className="text-slate-100">{e.action}</td>
                        <td className="text-slate-300">{e.resource}</td>
                        <td className="text-slate-400 font-mono text-[11px]">{e.actor.length > 24 ? `${e.actor.slice(0, 22)}…` : e.actor}</td>
                        <td className="text-slate-400 whitespace-normal max-w-[320px]">{e.detail}</td>
                        <td className="font-mono text-[10.5px] text-slate-500">{shortDigest(e.evidence)}</td>
                        <td className="font-mono text-[10.5px] text-slate-500" title={`prev ${e.prev}`}>{shortDigest(e.hash)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <div className="p-3 flex justify-center">
                  {nextBefore ? <GhostButton onClick={() => load(nextBefore)} disabled={loadingMore}>{loadingMore ? 'Loading…' : 'Load older entries'}</GhostButton> : <span className="text-[11.5px] text-slate-500">Beginning of the ledger.</span>}
                </div>
              </Glass>
            </>
          );
        }}
      </Gate>
    </div>
  );
}
