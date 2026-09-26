/**
 * "Attention required": conditions the snapshot proves, each with the
 * record it came from. Nothing is scored or ranked beyond severity, and a
 * missing observation is reported as missing, not as fine.
 */
import type { AttentionItem, RealitySnapshot } from '../../types/reality';

export type { AttentionItem };

const PRESSURE = 0.85;

export function attentionItems(s: RealitySnapshot): AttentionItem[] {
  const out: AttentionItem[] = [];
  const at = s.overview.provenance.observedAt;
  const add = (i: AttentionItem) => out.push(i);

  if (s.evidence.verification.state !== 'VERIFIED') {
    add({ severity: 'critical', kind: 'audit', subject: 'audit ledger', title: `Audit ledger ${s.evidence.verification.state}`, detail: s.evidence.verification.breakDetail ?? 'the hash chain did not verify', href: '/activity', source: 'control-plane audit verification', observedAt: s.evidence.verification.verifiedAt });
  }
  for (const n of s.nodes) {
    const href = `/nodes/${encodeURIComponent(n.id)}`;
    const src = 'signed host observation';
    if (n.lifecycle === 'REVOKED') continue;
    if (n.lifecycle === 'PENDING_APPROVAL') add({ severity: 'info', kind: 'node-pending', subject: n.name, title: `${n.name} is waiting for approval`, detail: 'enrolled with a join token; it admits no work until an owner approves it', href, source: 'control-plane node record', observedAt: n.joinedAt });
    else if (n.health === 'OFFLINE') add({ severity: 'critical', kind: 'node-offline', subject: n.name, title: `${n.name} is offline`, detail: n.healthReason, href, source: src, observedAt: n.observation.observedAt });
    else if (n.health === 'DEGRADED') add({ severity: 'warning', kind: 'node-stale', subject: n.name, title: `${n.name} observation is stale`, detail: n.healthReason, href, source: src, observedAt: n.observation.observedAt });
    else if (n.health === 'UNKNOWN') add({ severity: 'warning', kind: 'node-unknown', subject: n.name, title: `${n.name} has not reported`, detail: n.healthReason, href, source: src, observedAt: null });
    if (n.storage && n.storage.quotaBytes > 0 && n.storage.usedBytes / n.storage.quotaBytes >= PRESSURE) {
      add({ severity: 'warning', kind: 'storage-pressure', subject: n.name, title: `${n.name} storage at ${Math.round((n.storage.usedBytes / n.storage.quotaBytes) * 100)}% of quota`, detail: 'host storage quota from its own policy', href, source: src, observedAt: n.observation.observedAt });
    }
    if (n.storage && n.storage.corrupt > 0) add({ severity: 'critical', kind: 'storage-corrupt', subject: n.name, title: `${n.name} holds ${n.storage.corrupt} corrupt chunk(s)`, detail: 'found by the host re-hashing stored chunks', href, source: src, observedAt: n.observation.observedAt });
    if (n.mode && n.mode !== 'normal' && n.health !== 'OFFLINE') add({ severity: 'warning', kind: 'node-mode', subject: n.name, title: `${n.name} is in ${n.mode}`, detail: n.modeDetail, href, source: src, observedAt: n.observation.observedAt });
  }
  for (const a of s.apps) {
    if (a.deleted) continue;
    const href = `/apps/${encodeURIComponent(a.name)}`;
    if (a.phase === 'DEGRADED' || a.phase === 'REFUSED') add({ severity: a.phase === 'REFUSED' ? 'critical' : 'warning', kind: 'app', subject: a.name, title: `${a.name} is ${a.phase.toLowerCase()}`, detail: a.phaseReason, href, source: 'replica rows (desired vs observed)', observedAt: at });
  }
  for (const v of s.volumes) {
    if (v.state && !/^(healthy|ok|replicated)$/i.test(v.state)) add({ severity: 'warning', kind: 'volume', subject: v.id, title: `Volume ${v.app}/${v.name} is ${v.state}`, detail: v.detail || `${v.verified}/${v.durabilityReplicas} verified replicas`, href: '/storage', source: 'replica evidence', observedAt: v.committed?.ts ?? null });
  }
  for (const c of s.certificates) {
    if (c.state === 'EXPIRED' || c.state === 'FAILED') add({ severity: 'critical', kind: 'certificate', subject: c.host, title: `Certificate for ${c.host} is ${c.state.toLowerCase()}`, detail: c.detail, href: '/security', source: `observed by ${c.observedBy}`, observedAt: at });
    else if (c.state === 'EXPIRING') add({ severity: 'warning', kind: 'certificate', subject: c.host, title: `Certificate for ${c.host} expires soon`, detail: c.notAfter ? `not after ${new Date(c.notAfter).toISOString()}` : c.detail, href: '/security', source: `observed by ${c.observedBy}`, observedAt: at });
  }
  for (const ctl of s.security) {
    if (ctl.result === 'FAIL') add({ severity: 'warning', kind: 'security', subject: ctl.id, title: ctl.title, detail: ctl.detail, href: '/security', source: ctl.basis, observedAt: at });
  }
  // The control plane's own incident strings, when they add something the above does not name.
  for (const inc of s.overview.incidents) {
    if (!out.some((i) => inc.includes(i.subject))) add({ severity: 'warning', kind: 'incident', subject: 'control plane', title: inc, detail: 'reported by the control plane view', href: '/', source: 'control-plane view', observedAt: at });
  }
  const rank = { critical: 0, warning: 1, info: 2 };
  return out.sort((a, b) => rank[a.severity] - rank[b.severity]);
}
