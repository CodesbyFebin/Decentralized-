import React from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import type { ReplicaRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, PanelHeader, StatusPill } from '../common/ui';
import { Gate, FreshnessPill, shortDigest, fmtTime, Note } from '../common/states';
import { StageList, type DeploymentProgress } from './AppDetailView';

/**
 * Deployment progress for one generation. State lives in the control plane,
 * so refreshing the browser resumes exactly where the backend is.
 */
export default function DeploymentView() {
  const { app = '', generation = '' } = useParams();
  const res = useResource<{ deployment: DeploymentProgress; replicas: ReplicaRec[] }>(`/deployments/${encodeURIComponent(app)}/${encodeURIComponent(generation)}`, { pollMs: 2000 });
  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <Link to={`/apps/${encodeURIComponent(app)}?tab=deployments`} className="inline-flex items-center gap-1.5 text-[12.5px] text-slate-400 hover:text-cyan-300">
        <ArrowLeft className="w-3.5 h-3.5" /> {app}
      </Link>
      <Gate res={res}>
        {(d) => (
          <>
            <Glass className="p-5">
              <div className="flex flex-wrap items-center gap-3">
                <h1 className="text-2xl font-extrabold text-white">{d.deployment.app} · generation {d.deployment.generation}</h1>
                <StatusPill status={d.deployment.state} />
              </div>
              <div className="mt-1 text-[12px] text-slate-400">
                {d.deployment.change ?? 'change'} by {d.deployment.actor ?? 'unknown'} at {fmtTime(d.deployment.submittedAt)} · manifest {shortDigest(d.deployment.hash)} · artifact {shortDigest(d.deployment.imageDigest)}
              </div>
              <div className="mt-4">
                <StageList stages={d.deployment.stages} />
              </div>
              <Note>Stages are derived from the replica rows the control plane serves for this generation. A successful deployment is not the same as verified evidence; see the Evidence Ledger.</Note>
            </Glass>
            <Glass className="overflow-x-auto">
              <div className="px-4 pt-4"><PanelHeader title="Replicas for this generation" /></div>
              <table className="dh-table w-full min-w-[720px] mt-2">
                <thead><tr><th>Assignment</th><th>Host</th><th>Admitted</th><th>Observed</th><th>Reason</th><th>Freshness</th></tr></thead>
                <tbody>
                  {d.replicas.map((r) => (
                    <tr key={r.assignment + r.node}>
                      <td>{r.assignment}</td>
                      <td><Link to={`/nodes/${r.node}`} className="text-cyan-300 hover:underline">{r.nodeName}</Link></td>
                      <td><StatusPill status={r.admitted} dot={false} /></td>
                      <td><StatusPill status={r.observed} dot={false} /></td>
                      <td className="text-slate-400 whitespace-normal">{r.reason}</td>
                      <td><FreshnessPill f={r.freshness} /></td>
                    </tr>
                  ))}
                  {!d.replicas.length && <tr><td colSpan={6} className="text-center text-slate-500 py-6">No assignments for this generation yet (QUEUED).</td></tr>}
                </tbody>
              </table>
            </Glass>
          </>
        )}
      </Gate>
    </div>
  );
}
