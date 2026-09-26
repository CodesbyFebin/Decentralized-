import React from 'react';
import { ShieldCheck, ShieldAlert, ShieldX, ShieldQuestion, Ban, Lock } from 'lucide-react';
import type { CertRec, ClusterRec, LedgerVerification, SecurityControl } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, PanelHeader, StatusPill, IconTile } from '../common/ui';
import { Gate, TruthTag, fmtTime, Note, since } from '../common/states';

const ICON = { PASS: ShieldCheck, WARN: ShieldAlert, FAIL: ShieldX, UNKNOWN: ShieldQuestion, UNAVAILABLE: Ban } as const;
const TONE = { PASS: 'emerald', WARN: 'amber', FAIL: 'rose', UNKNOWN: 'slate', UNAVAILABLE: 'slate' } as const;

export default function SecurityView() {
  const res = useResource<{ controls: SecurityControl[]; certificates: CertRec[]; verification: LedgerVerification; cluster: ClusterRec }>('/security', { pollMs: 10_000 });
  return (
    <div className="space-y-4 pt-2">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">SSL & Security</h1>
        <p className="text-[13px] text-slate-400">Concrete controls derived from what hosts and the control plane report. There is no aggregate security score.</p>
      </div>
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
            <Note>Audit ledger last verified {since(d.verification.verifiedAt)} by {d.verification.verifiedBy.slice(0, 12) || '—'}: {d.verification.state}. WAF and DDoS telemetry are shown as unavailable because no such subsystem exists.</Note>
          </>
        )}
      </Gate>
    </div>
  );
}
