import React, { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, ShieldAlert, RefreshCw, Download, ChevronDown, ChevronRight, CheckCircle2, XCircle, MinusCircle } from 'lucide-react';
import type { ValidationRecord, ValidationVerification } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/client';
import { Glass, PanelHeader, GhostButton, StatusPill } from '../common/ui';
import { Gate, ErrorState, Note, fmtTime, fmtDuration } from '../common/states';
import { EvidenceSeal, Digest } from '../common/truth';

interface Payload {
  record: ValidationRecord;
  verification: ValidationVerification | null;
  canVerify: boolean;
  children: { id: string; outcome: ValidationRecord['outcome'] }[];
}

const Row: React.FC<{ k: string; children: React.ReactNode }> = ({ k, children }) => (
  <div className="grid grid-cols-[150px_1fr] gap-3 py-1.5 text-[12.5px] border-b border-[rgba(125,190,255,0.06)] last:border-0">
    <dt className="text-slate-400">{k}</dt>
    <dd className="text-slate-100 break-words min-w-0">{children}</dd>
  </div>
);

function StepLog({ id, step }: { id: string; step: string }) {
  const res = useResource<{ text: string; truncated: boolean }>(`/evidence/records/${encodeURIComponent(id)}/steps/${encodeURIComponent(step)}/log`);
  return (
    <Gate res={res}>
      {(d) => (
        <pre className="mt-2 max-h-[360px] overflow-auto rounded-xl bg-[#020611]/95 border border-[rgba(125,190,255,0.12)] p-3 text-[11.5px] leading-relaxed text-slate-200 whitespace-pre-wrap" style={{ fontFamily: 'var(--font-mono)' }} tabIndex={0}>
          {d.truncated && '… (earlier output truncated; the full log is in the record directory)\n'}
          {d.text || '(empty log)'}
        </pre>
      )}
    </Gate>
  );
}

export default function EvidenceDetailView() {
  const { id = '' } = useParams();
  const res = useResource<Payload>(`/evidence/records/${encodeURIComponent(id)}`);
  const [verification, setVerification] = useState<ValidationVerification | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [verr, setVerr] = useState<ApiError | null>(null);
  const [open, setOpen] = useState<string | null>(null);

  const verify = async () => {
    setVerifying(true);
    setVerr(null);
    try {
      const r = await api.post<{ data: ValidationVerification }>(`/evidence/records/${encodeURIComponent(id)}/verify`);
      setVerification(r.data);
    } catch (e) {
      setVerr(e as ApiError);
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="space-y-4 pt-2">
      <Link to="/evidence" className="inline-flex items-center gap-1.5 text-[12.5px] text-slate-400 hover:text-cyan-300">
        <ArrowLeft className="w-3.5 h-3.5" /> Evidence
      </Link>
      <Gate res={res}>
        {(d) => {
          const r = d.record;
          const v = verification ?? d.verification;
          const failed = r.steps.filter((s) => s.exit !== 0);
          const missing = r.requiredSteps.filter((n) => !r.steps.some((s) => s.name === n));
          return (
            <>
              {r.outcome !== 'PASS' && (
                <div role="alert" className="flex items-start gap-3 p-4 rounded-2xl border border-rose-400/40 bg-rose-500/10">
                  <ShieldAlert className="w-6 h-6 text-rose-300 shrink-0" aria-hidden />
                  <div>
                    <div className="text-[15px] font-bold text-rose-100">{r.outcome === 'FAIL' ? 'This record FAILED' : `Outcome: ${r.outcome}`}</div>
                    <div className="text-[13px] text-rose-100/90">{r.outcomeReason || 'no reason recorded'}</div>
                    {failed.length > 0 && <div className="mt-1 text-[12px] text-rose-200">Failed steps: {failed.map((s) => s.name).join(', ')}</div>}
                    {d.children.length > 0 && <div className="mt-1 text-[12px] text-rose-100/80">Later attempt(s): {d.children.map((c) => <Link key={c.id} to={`/evidence/${encodeURIComponent(c.id)}`} className="underline mr-2">{c.id} ({c.outcome})</Link>)}</div>}
                  </div>
                </div>
              )}

              <Glass className="p-5">
                <div className="flex flex-wrap items-start gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h1 className="text-2xl font-extrabold text-white tracking-tight font-mono">{r.id}</h1>
                      <EvidenceSeal outcome={r.outcome} size="md" />
                    </div>
                    <p className="mt-2 text-[13px] text-slate-300 max-w-4xl">{r.claim}</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <GhostButton onClick={verify} disabled={!d.canVerify || verifying} title={d.canVerify ? 'Runs dh evidence verify on the stored record' : 'No dh CLI configured (DH_CLI)'} className="disabled:opacity-40">
                      <RefreshCw className={`w-4 h-4 ${verifying ? 'animate-spin' : ''}`} /> Verify
                    </GhostButton>
                    <a href={`/api/v1/evidence/records/${encodeURIComponent(r.id)}/record.json`} className="inline-flex items-center gap-1.5 px-3 py-2 rounded-xl border border-[rgba(125,190,255,0.18)] text-[13px] text-slate-100 hover:border-cyan-400/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400">
                      <Download className="w-4 h-4" /> Export signed record
                    </a>
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-2 text-[12.5px]">
                  <span className="text-slate-400">Verification here:</span>
                  {v ? (
                    <>
                      <StatusPill status={v.state} tone={v.state === 'VERIFIED' ? 'emerald' : 'rose'} />
                      <span className="text-slate-300 break-all">{v.detail}</span>
                      <span className="text-slate-500">· {v.verifier} · {fmtTime(v.verifiedAt)}</span>
                    </>
                  ) : (
                    <span className="text-slate-400">not run in this console session{d.canVerify ? '' : ' (no dh CLI configured)'}</span>
                  )}
                </div>
                {verr && <div className="mt-3"><ErrorState error={verr} /></div>}
              </Glass>

              <div className="grid lg:grid-cols-2 gap-4 [&>*]:min-w-0">
                <Glass className="p-4">
                  <PanelHeader title="Subject" />
                  <dl className="mt-2">
                    <Row k="Stage">{r.stage} · attempt {r.attempt}</Row>
                    <Row k="Previous attempt">{r.parent ? <Link to={`/evidence/${encodeURIComponent(r.parent)}`} className="text-cyan-300 hover:underline">{r.parent}</Link> : 'none'}</Row>
                    <Row k="Scope">{r.scope}</Row>
                    <Row k="Excluded">{r.exclusions.length ? <ul className="list-disc ml-4">{r.exclusions.map((x) => <li key={x}>{x}</li>)}</ul> : 'nothing stated'}</Row>
                    <Row k="Limitations">{r.limitations.length ? <ul className="list-disc ml-4">{r.limitations.map((x) => <li key={x}>{x}</li>)}</ul> : 'none stated'}</Row>
                    <Row k="Started / ended">{fmtTime(r.startedAt)} → {fmtTime(r.endedAt)}{r.startedAt && r.endedAt ? ` (took ${fmtDuration(r.endedAt - r.startedAt)})` : ''}</Row>
                  </dl>
                </Glass>
                <Glass className="p-4">
                  <PanelHeader title="Binding" subtitle="What this record is about, exactly." />
                  <dl className="mt-2">
                    <Row k="Source digest"><Digest value={r.sourceDigest} chars={24} /> <span className="text-slate-500">({r.digestVersion})</span></Row>
                    <Row k="Commit">{r.commit ? <span className="font-mono">{r.commit}</span> : 'not recorded'}</Row>
                    <Row k="Binaries">{Object.keys(r.binaries).length ? Object.entries(r.binaries).map(([k, h]) => <div key={k} className="flex gap-2"><span className="text-slate-400 w-28 shrink-0">{k}</span><Digest value={h} chars={12} /></div>) : 'none recorded'}</Row>
                    <Row k="Signer">{r.signer}</Row>
                    <Row k="Signer key"><Digest value={r.signerPub} chars={20} /></Row>
                    <Row k="Signature"><Digest value={r.signature} chars={20} /></Row>
                    <Row k="Files hashed">{r.files} · schema {r.schemaVersion}</Row>
                  </dl>
                </Glass>
              </div>

              <Glass className="p-4 overflow-x-auto">
                <PanelHeader title="Gate results" subtitle={`${r.steps.length} step(s) ran · ${r.requiredSteps.length} required`} />
                {missing.length > 0 && <p className="mt-2 text-[12.5px] text-rose-200">Required but not run: {missing.join(', ')}</p>}
                <table className="dh-table w-full min-w-[720px] mt-2">
                  <thead><tr><th>Step</th><th>Result</th><th>Exit</th><th>Attempts</th><th>Duration</th><th>Log</th></tr></thead>
                  <tbody>
                    {r.steps.map((s) => {
                      const ok = s.exit === 0;
                      return (
                        <React.Fragment key={s.name}>
                          <tr className={ok ? undefined : 'bg-rose-500/[0.08]'}>
                            <td className="font-mono text-[12px]">{s.name}{r.requiredSteps.includes(s.name) ? '' : <span className="text-slate-500"> (optional)</span>}</td>
                            <td>{ok ? <span className="inline-flex items-center gap-1 text-emerald-300"><CheckCircle2 className="w-4 h-4" aria-hidden /> passed</span> : s.exit === null ? <span className="inline-flex items-center gap-1 text-slate-400"><MinusCircle className="w-4 h-4" aria-hidden /> did not finish</span> : <span className="inline-flex items-center gap-1 text-rose-300 font-semibold"><XCircle className="w-4 h-4" aria-hidden /> FAILED</span>}</td>
                            <td className="tabular-nums">{s.exit ?? '—'}</td>
                            <td className="tabular-nums">{s.attempts}</td>
                            <td className="tabular-nums">{fmtDuration(s.durationMs)}</td>
                            <td>
                              {s.log ? (
                                <button type="button" onClick={() => setOpen(open === s.name ? null : s.name)} aria-expanded={open === s.name} className="inline-flex items-center gap-1 text-[12px] text-cyan-300 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 rounded">
                                  {open === s.name ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />} {s.log}
                                </button>
                              ) : '—'}
                            </td>
                          </tr>
                          {open === s.name && (
                            <tr>
                              <td colSpan={6}>
                                <div className="text-[11px] text-slate-500 font-mono">$ {s.command.join(' ')} · log hash {s.logHash ?? '—'}</div>
                                <StepLog id={r.id} step={s.name} />
                              </td>
                            </tr>
                          )}
                        </React.Fragment>
                      );
                    })}
                  </tbody>
                </table>
                <Note>Verify offline with <code className="text-cyan-200">dh evidence verify --dir evidence/{r.dir}</code>. It checks the signature and re-hashes every file the record names.</Note>
              </Glass>
            </>
          );
        }}
      </Gate>
    </div>
  );
}
