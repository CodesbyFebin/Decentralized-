import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { FileCheck2, RefreshCw, ShieldCheck, ShieldX, Package, Target } from 'lucide-react';
import type { ArtifactRec, LedgerVerification, MilestoneRec, ValidationRecord, ValidationVerification } from '../../types/reality';
import { useSession } from '../../lib/session';
import { EvidenceSeal, Digest } from '../common/truth';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/client';
import { Glass, PanelHeader, StatusPill, IconTile, GhostButton, FilterChips } from '../common/ui';
import { Gate, shortDigest, since, fmtTime, ErrorState, Note, TruthTag, Unavailable } from '../common/states';

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
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Evidence</h1>
        <p className="text-[13px] text-slate-400">What has actually been verified: signed validation records, the hash-chained audit ledger, and signed host and artifact evidence. Failed records stay visible. Nothing here is editable.</p>
      </div>
      <ValidationRecords />
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
                <Note>Verify independently: <code className="text-cyan-200">dh audit verify</code> fetches the ledger and checks the chain and checkpoints locally. Browse entries in <Link to="/activity" className="text-cyan-300 hover:underline">Activity</Link>.</Note>
              </Glass>

              <div className="grid lg:grid-cols-2 gap-4 [&>*]:min-w-0">
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

            </>
          );
        }}
      </Gate>
    </div>
  );
}

type OutcomeChip = 'all' | 'PASS' | 'FAIL' | 'UNKNOWN';

function ValidationRecords() {
  const { capabilities } = useSession();
  const cap = capabilities?.items.validationRecords;
  const res = useResource<{ records: ValidationRecord[]; verifications: Record<string, ValidationVerification | null>; canVerify: boolean }>(cap?.state === 'LIVE' ? '/evidence/records' : null, { pollMs: 30_000 });
  const [chip, setChip] = useState<OutcomeChip>('all');
  const [stage, setStage] = useState('all');
  if (!cap) return null;
  if (cap.state !== 'LIVE') return <Unavailable title="Validation records" detail={`${cap.detail}. Point DH_EVIDENCE_DIR at the repository's evidence/ directory (and DH_CLI at a dh binary to verify).`} />;
  return (
    <Glass className="p-4 overflow-hidden relative" aria-labelledby="records-title">
      <div className="absolute inset-0 pointer-events-none opacity-60" style={{ background: 'radial-gradient(600px 180px at 10% 0%, rgba(34,211,238,0.10), transparent 60%), radial-gradient(500px 160px at 90% 0%, rgba(168,85,247,0.10), transparent 60%)' }} />
      <div className="relative">
        <PanelHeader title="Validation records" subtitle="Signed by the validator key with dh evidence seal. Outcome and verification are as recorded and as re-checked; neither is inferred." right={<TruthTag state="LIVE" title={cap.detail} />} />
        <Gate res={res}>
          {(d) => {
            const stages = [...new Set(d.records.map((r) => r.stage))].sort();
            const bucket = (r: ValidationRecord): OutcomeChip => (r.outcome === 'PASS' ? 'PASS' : r.outcome === 'FAIL' ? 'FAIL' : 'UNKNOWN');
            const rows = d.records.filter((r) => (chip === 'all' || bucket(r) === chip) && (stage === 'all' || r.stage === stage));
            return (
              <>
                <div className="mt-3 flex flex-wrap items-center gap-3">
                  <FilterChips<OutcomeChip>
                    chips={[
                      { id: 'all', label: 'All', count: d.records.length },
                      { id: 'PASS', label: 'Pass', count: d.records.filter((r) => bucket(r) === 'PASS').length, tone: 'emerald' },
                      { id: 'FAIL', label: 'Fail', count: d.records.filter((r) => bucket(r) === 'FAIL').length, tone: 'rose' },
                      { id: 'UNKNOWN', label: 'Unknown / infra', count: d.records.filter((r) => bucket(r) === 'UNKNOWN').length, tone: 'slate' }
                    ]}
                    active={chip}
                    onChange={setChip}
                  />
                  <label className="text-[12px] text-slate-400">Subject
                    <select value={stage} onChange={(e) => setStage(e.target.value)} className="dh-input !py-1 ml-2">
                      <option value="all">all stages</option>
                      {stages.map((x) => <option key={x} value={x}>{x}</option>)}
                    </select>
                  </label>
                </div>
                <div className="overflow-x-auto mt-3">
                  <table className="dh-table w-full min-w-[980px]">
                    <thead><tr><th>Evidence id</th><th>Outcome</th><th>Subject</th><th>Source digest</th><th>Commit</th><th>Signer</th><th>Created</th><th>Verification here</th></tr></thead>
                    <tbody>
                      {rows.map((r) => {
                        const v = d.verifications[r.id];
                        return (
                          <tr key={r.id} className={r.outcome === 'FAIL' ? 'bg-rose-500/[0.07]' : undefined}>
                            <td><Link to={`/evidence/${encodeURIComponent(r.id)}`} className="font-mono text-[12px] text-cyan-300 hover:underline">{r.id}</Link>{r.parent && <div className="text-[10.5px] text-slate-500">after {r.parent}</div>}</td>
                            <td><EvidenceSeal outcome={r.outcome} />{r.outcome === 'FAIL' && <div className="text-[11px] text-rose-200 mt-0.5 max-w-[240px] whitespace-normal">{r.outcomeReason}</div>}</td>
                            <td>{r.stage}</td>
                            <td><Digest value={r.sourceDigest} chars={10} /></td>
                            <td className="font-mono text-[11.5px] text-slate-300">{r.commit ? r.commit.slice(0, 7) : '—'}</td>
                            <td className="font-mono text-[11px] text-slate-400" title={r.signerPub}>{r.signer.slice(0, 12)}…</td>
                            <td className="text-slate-300" title={fmtTime(r.endedAt)}>{since(r.endedAt)}</td>
                            <td>{v ? <StatusPill status={v.state} tone={v.state === 'VERIFIED' ? 'emerald' : 'rose'} /> : <span className="text-[11.5px] text-slate-500">not run</span>}</td>
                          </tr>
                        );
                      })}
                      {!rows.length && <tr><td colSpan={8} className="text-center text-slate-500 py-6">No record matches this filter.</td></tr>}
                    </tbody>
                  </table>
                </div>
                <Note>A PASS proves the claim only for the scope, commit and source digest the record names. Records are signed by the validator key the operator used; trust in that key is separate from the signature check.</Note>
              </>
            );
          }}
        </Gate>
      </div>
    </Glass>
  );
}
