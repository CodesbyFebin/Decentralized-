/**
 * Long-running operations, derived. The control plane commits each change
 * synchronously and has no Operation resource, so what is "in progress" is
 * read from state: an application converging to a generation, a host being
 * drained, a volume replicating. Every item names where it came from.
 */
import type { OperationRec, RealitySnapshot } from '../../types/reality';

type Progress = { id: string; app: string; generation: number; state: string; stages: { id: string; label: string; state: string; detail: string }[]; actor: string | null; submittedAt: number | null; updatedAt: number | null };

const DEPLOY_STATUS: Record<string, OperationRec['status']> = {
  QUEUED: 'QUEUED',
  DEPLOYING: 'RUNNING',
  VERIFYING: 'RUNNING',
  READY: 'SUCCEEDED',
  VALIDATION_FAILED: 'FAILED',
  DEPLOY_FAILED: 'FAILED',
  SUPERSEDED: 'CANCELLED'
};

export function deriveOperations(s: RealitySnapshot, progress: (appName: string, generation: number) => Progress | null, generations: (appName: string) => number[], perApp = 3): OperationRec[] {
  const ops: OperationRec[] = [];
  for (const a of s.apps) {
    if (a.deleted) continue;
    for (const g of generations(a.name).slice(-perApp)) {
      const p = progress(a.name, g);
      if (!p) continue;
      const failed = p.stages.find((x) => x.state === 'FAILED');
      ops.push({
        id: `deploy:${a.name}@${g}`,
        kind: 'deployment',
        target: a.name,
        targetHref: `/deploy/${encodeURIComponent(a.name)}/${g}`,
        title: `Deploy ${a.name} generation ${g}`,
        actor: p.actor,
        startedAt: p.submittedAt,
        updatedAt: p.updatedAt,
        status: DEPLOY_STATUS[p.state] ?? 'RUNNING',
        detail: p.state,
        steps: p.stages.map((x) => ({ label: x.label, state: x.state, detail: x.detail })),
        error: failed ? failed.detail : null,
        source: 'replica rows for this generation'
      });
    }
  }
  for (const n of s.nodes) {
    if (n.lifecycle !== 'DRAINING') continue;
    const remaining = s.apps.flatMap((a) => a.replicas).filter((r) => r.node === n.id && r.desired === 'RUNNING').length;
    const req = s.evidence.entries.find((e) => e.action === 'node-drain' && e.resource === `node/${n.id}`);
    ops.push({
      id: `drain:${n.id}`,
      kind: 'drain',
      target: n.name,
      targetHref: `/nodes/${encodeURIComponent(n.id)}`,
      title: `Drain ${n.name}`,
      actor: req?.actor ?? null,
      startedAt: req?.ts ?? null,
      updatedAt: n.observation.observedAt,
      status: remaining > 0 ? 'RUNNING' : 'WAITING',
      detail: remaining > 0 ? `${remaining} replica(s) still desired on this host` : 'no replicas left; waiting for resume or revoke',
      steps: [
        { label: 'Drain requested', state: 'PASSED', detail: req ? `audit #${req.seq}` : 'request not in the recent audit window' },
        { label: 'Replicas rescheduled elsewhere', state: remaining > 0 ? 'RUNNING' : 'PASSED', detail: `${remaining} remaining` }
      ],
      error: null,
      source: 'host lifecycle + desired assignments'
    });
  }
  for (const v of s.volumes) {
    if (!v.state || /^(healthy|ok|replicated)$/i.test(v.state)) continue;
    ops.push({
      id: `replication:${v.id}`,
      kind: 'replication',
      target: `${v.app}/${v.name}`,
      targetHref: '/storage',
      title: `Replicate volume ${v.app}/${v.name}`,
      actor: 'control plane',
      startedAt: null,
      updatedAt: v.committed?.ts ?? null,
      status: 'RUNNING',
      detail: v.detail || v.state,
      steps: [{ label: 'Verified replicas', state: v.verified >= v.durabilityReplicas ? 'PASSED' : 'RUNNING', detail: `${v.verified}/${v.durabilityReplicas}` }],
      error: null,
      source: 'replica evidence'
    });
  }
  const rank: Record<OperationRec['status'], number> = { RUNNING: 0, QUEUED: 1, WAITING: 2, FAILED: 3, SUCCEEDED: 4, CANCELLED: 5 };
  return ops.sort((a, b) => rank[a.status] - rank[b.status] || (b.updatedAt ?? 0) - (a.updatedAt ?? 0));
}
