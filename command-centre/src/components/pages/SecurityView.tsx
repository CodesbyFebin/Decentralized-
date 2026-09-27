import React, { useState } from 'react';
import { ShieldCheck, ShieldAlert, ShieldX, ShieldQuestion, Ban, Lock, RotateCcw, AlertTriangle } from 'lucide-react';
import type { CertRec, ClusterRec, LedgerVerification, SecurityControl } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError } from '../../lib/client';
import { Glass, PanelHeader, StatusPill, IconTile, PrimaryButton, GhostButton } from '../common/ui';
import { Gate, TruthTag, fmtTime, Note, since, ErrorState } from '../common/states';

const ICON = { PASS: ShieldCheck, WARN: ShieldAlert, FAIL: ShieldX, UNKNOWN: ShieldQuestion, UNAVAILABLE: Ban } as const;
const TONE = { PASS: 'emerald', WARN: 'amber', FAIL: 'rose', UNKNOWN: 'slate', UNAVAILABLE: 'slate' } as const;

export default function SecurityView() {
  const { can, mode } = useSession();
  const res = useResource<{ controls: SecurityControl[]; certificates: CertRec[]; verification: LedgerVerification; cluster: ClusterRec; rootKey?: { rotatedAt: number; nextRotation?: number }; policies?: { maxWorkloads?: number; cpuLimit?: string; memLimit?: string; acceptedTiers?: string[] }; revoked?: Array<{ key: string; reason: string; ts: number }> }>('/security', { pollMs: 10_000 });

  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const allowed = can('api.write') && mode === 'controlplane';

  const rotateRootKey = async () => {
    if (!confirm('Rotate root key? This will require re-enrollment of all hosts.')) return;
    setBusy(true);
    setErr(null);
    setSuccessMsg(null);
    try {
      await api.post('/root-rotate', {});
      setSuccessMsg('Root key rotation initiated. Hosts will re-enroll.');
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setBusy(false);
    }
  };

  const toggleClusterFreeze = async (freeze: boolean) => {
    setBusy(true);
    setErr(null);
    try {
      await api.post('/emergency-control', { freeze });
      setSuccessMsg(freeze ? 'Cluster frozen. No new replicas will deploy.' : 'Cluster unfrozen. Operations resume.');
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 pt-2">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">SSL & Security</h1>
          <p className="text-[13px] text-slate-400">Concrete controls derived from what hosts and the control plane report. There is no aggregate security score.</p>
        </div>
        {allowed && (
          <div className="flex gap-2">
            <PrimaryButton onClick={() => rotateRootKey()} disabled={busy} className="flex items-center gap-2">
              <RotateCcw className="w-4 h-4" /> Rotate Root Key
            </PrimaryButton>
          </div>
        )}
      </div>

      {successMsg && (
        <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/10 p-3 text-[12.5px] text-emerald-100">
          {successMsg}
        </div>
      )}
      {err && <ErrorState error={err} />}

      <Gate res={res}>
        {(d) => (
          <>
            <div className="grid sm:grid-cols-5 gap-2">
              {(['PASS', 'WARN', 'FAIL', 'UNKNOWN', 'UNAVAILABLE'] as const).map((k) => (
                <Glass key={k} className="p-3 text-center">
                  <div className="text-[22px] font-bold text-white tabular-nums">{d.controls.filter((c) => c.result === k).length}</div>
                  <StatusPill status={k} tone={TONE[k]} />
                </Glass>
              ))}
            </div>
            <Glass className="p-4">
              <PanelHeader title="Controls" subtitle="Each result shows its basis; click through to the resource for detail." />
              <ul className="mt-3 divide-y divide-[rgba(125,190,255,0.08)]">
                {d.controls.map((c) => {
                  const Icon = ICON[c.result];
                  return (
                    <li key={c.id} className="py-2.5 flex items-start gap-3">
                      <IconTile tone={TONE[c.result]} size="sm"><Icon className="w-4 h-4" /></IconTile>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-[13px] font-semibold text-slate-100">{c.title}</span>
                          <StatusPill status={c.result} tone={TONE[c.result]} dot={false} />
                          <TruthTag state={c.basis} />
                        </div>
                        <div className="text-[12px] text-slate-400 break-words">{c.detail}</div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            </Glass>
            <Glass className="overflow-x-auto">
              <div className="px-4 pt-4"><PanelHeader icon={<IconTile tone="blue" size="sm"><Lock className="w-4 h-4" /></IconTile>} title="Certificates" subtitle="X.509 metadata observed by edge hosts from the certificates they serve." /></div>
              {d.certificates.length ? (
                <table className="dh-table w-full min-w-[860px] mt-2">
                  <thead><tr><th>Host</th><th>State</th><th>Issuer</th><th>Valid from</th><th>Expires</th><th>Fingerprint</th><th>Observed by</th></tr></thead>
                  <tbody>
                    {d.certificates.map((c) => (
                      <tr key={c.host}>
                        <td className="text-sky-300 font-semibold">{c.host}<div className="text-[11px] text-slate-500">mode {c.tlsMode || '—'}</div></td>
                        <td title={c.detail}><StatusPill status={c.state} /><div className="text-[10.5px] text-slate-500">edge: {c.backendState}</div></td>
                        <td className="text-slate-300 whitespace-normal">{c.issuer}</td>
                        <td className="text-slate-300">{fmtTime(c.notBefore)}</td>
                        <td className="text-slate-300">{fmtTime(c.notAfter)}</td>
                        <td className="font-mono text-[10.5px] text-slate-400 whitespace-normal break-all">{c.fingerprint}</td>
                        <td className="text-slate-300">{c.observedBy}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="p-4 text-[12.5px] text-slate-400">No edge host reports a certificate.</p>
              )}
            </Glass>
            {d.rootKey && (
              <Glass className="p-4">
                <PanelHeader title="Root Key Management" />
                <div className="mt-3 space-y-2 text-[13px]">
                  <div className="flex justify-between"><span className="text-slate-400">Current key rotated</span><span className="text-slate-100">{fmtTime(d.rootKey.rotatedAt)}</span></div>
                  {d.rootKey.nextRotation && <div className="flex justify-between"><span className="text-slate-400">Next rotation due</span><span className="text-slate-100">{fmtTime(d.rootKey.nextRotation)}</span></div>}
                  <div className="flex justify-between"><span className="text-slate-400">Rotation interval</span><span className="text-slate-100">90 days</span></div>
                  {allowed && <PrimaryButton onClick={() => rotateRootKey()} disabled={busy} className="mt-2 text-[12px] px-3 py-1">Rotate now</PrimaryButton>}
                </div>
              </Glass>
            )}

            {d.policies && (
              <Glass className="p-4">
                <PanelHeader title="Policy Enforcement" />
                <div className="mt-3 space-y-2 text-[13px]">
                  {d.policies.maxWorkloads && <div className="flex justify-between"><span className="text-slate-400">Max workloads per tier</span><span className="text-slate-100">{d.policies.maxWorkloads}</span></div>}
                  {d.policies.cpuLimit && <div className="flex justify-between"><span className="text-slate-400">CPU limit (per workload)</span><span className="text-slate-100">{d.policies.cpuLimit}</span></div>}
                  {d.policies.memLimit && <div className="flex justify-between"><span className="text-slate-400">Memory limit (per workload)</span><span className="text-slate-100">{d.policies.memLimit}</span></div>}
                  {d.policies.acceptedTiers && <div className="flex justify-between"><span className="text-slate-400">Accepted tiers</span><span className="text-slate-100">{d.policies.acceptedTiers.join(', ')}</span></div>}
                </div>
              </Glass>
            )}

            {d.revoked && d.revoked.length > 0 && (
              <Glass className="overflow-x-auto">
                <div className="px-4 pt-4"><PanelHeader title="Revocation List" subtitle="Signed keys and timestamps that are no longer trusted." /></div>
                <table className="dh-table w-full min-w-[700px] mt-2">
                  <thead><tr><th>Key (short)</th><th>Reason</th><th>Revoked at</th></tr></thead>
                  <tbody>
                    {d.revoked.map((r) => (
                      <tr key={r.key}>
                        <td className="font-mono text-[11px] text-slate-300">{r.key.slice(0, 12)}</td>
                        <td className="text-slate-300">{r.reason}</td>
                        <td className="text-slate-300">{fmtTime(r.ts)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </Glass>
            )}

            {allowed && (
              <Glass className="p-4 border border-amber-400/30 bg-amber-500/5">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="w-5 h-5 text-amber-300 flex-shrink-0 mt-0.5" />
                  <div className="flex-1">
                    <PanelHeader title="Emergency Controls" subtitle="Freeze the cluster to prevent new deployments while you investigate." />
                    <div className="mt-3 flex gap-2">
                      <PrimaryButton onClick={() => toggleClusterFreeze(true)} disabled={busy}>Freeze Cluster</PrimaryButton>
                      <GhostButton onClick={() => toggleClusterFreeze(false)} disabled={busy}>Unfreeze Cluster</GhostButton>
                    </div>
                  </div>
                </div>
              </Glass>
            )}

            <Note>Audit ledger last verified {since(d.verification.verifiedAt)} by {d.verification.verifiedBy.slice(0, 12) || '—'}: {d.verification.state}. WAF and DDoS telemetry are shown as unavailable because no such subsystem exists.</Note>
          </>
        )}
      </Gate>
    </div>
  );
}
