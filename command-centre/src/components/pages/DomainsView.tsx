import React from 'react';
import { Link } from 'react-router-dom';
import { Globe } from 'lucide-react';
import type { CertRec, DomainRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, IconTile, StatusPill } from '../common/ui';
import { Gate, Empty, Note, fmtTime, TruthTag } from '../common/states';

export default function DomainsView() {
  const res = useResource<{ domains: DomainRec[]; certificates: CertRec[] }>('/domains');
  return (
    <div className="space-y-4 pt-2">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Domains</h1>
        <p className="text-[13px] text-slate-400">Desired routing from manifests, routing observed at edge hosts, and TLS observed from served certificates — kept separate.</p>
      </div>
      <Gate res={res}>
        {(d) =>
          !d.domains.length ? (
            <Empty title="No domains" detail="Add spec.ingress[].host to an application manifest. Decentralized.Host is not a registrar or DNS provider; point the name at an edge host yourself." />
          ) : (
            <>
              <Glass className="overflow-x-auto">
                <table className="dh-table w-full min-w-[900px]">
                  <thead><tr><th>Domain</th><th>Desired</th><th>Routing (observed)</th><th>TLS (observed)</th><th>DNS</th><th>Expires</th></tr></thead>
                  <tbody>
                    {d.domains.map((x) => (
                      <tr key={x.host}>
                        <td><div className="flex items-center gap-2"><IconTile tone="blue" size="sm"><Globe className="w-4 h-4" /></IconTile><span className="font-semibold text-sky-300">{x.host}</span></div></td>
                        <td><Link to={`/apps/${encodeURIComponent(x.desired.app)}`} className="text-slate-200 hover:text-cyan-300">{x.desired.app}</Link><div className="text-[11px] text-slate-500">port {x.desired.port} · tls {x.desired.tls || 'none'}</div></td>
                        <td><StatusPill status={x.routing.state === 'NOT ROUTED' ? 'UNKNOWN' : x.routing.state} /><div className="text-[11px] text-slate-500 whitespace-normal">{x.routing.detail}</div></td>
                        <td>{x.tls ? <><StatusPill status={x.tls.state} /><div className="text-[11px] text-slate-500">{x.tls.issuer}</div></> : <StatusPill status={x.desired.tls === 'none' ? 'NONE' : 'UNKNOWN'} tone="slate" />}</td>
                        <td><TruthTag state={x.dns.state} title={x.dns.detail} /></td>
                        <td className="text-slate-300">{fmtTime(x.tls?.notAfter)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </Glass>
              <Note>DNS is not observed: the control plane does not resolve public DNS or register names. Routing and TLS come from edge-host observations.</Note>
            </>
          )
        }
      </Gate>
    </div>
  );
}
