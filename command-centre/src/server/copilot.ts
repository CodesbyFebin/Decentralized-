/**
 * RAG Copilot, grounded.
 *
 * Retrieval sources:
 *   - the RealitySnapshot fetched with the CALLER's capability (so anything the
 *     caller may not read never reaches retrieval or a prompt);
 *   - repository documentation (docs/, README.md) on disk.
 * Every factual line carries a citation. When the control plane is unavailable
 * the Copilot says it cannot verify current state instead of guessing.
 * Actions are structured proposals; nothing runs until the same principal
 * approves, and execution goes through the control plane.
 */
import { createHash, randomUUID } from 'node:crypto';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import type { NodeOperation, RealitySnapshot, TruthState } from '../types/reality';

export type CitationType = 'NODE_OBSERVATION' | 'DEPLOYMENT' | 'EVENT' | 'EVIDENCE' | 'DOCUMENTATION' | 'CONFIGURATION' | 'CERTIFICATE' | 'VOLUME';

export interface Citation {
  type: CitationType;
  resourceId: string;
  label: string;
  observedAt: number | null;
  href: string | null;
}

export type Risk = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

export interface ProposedAction {
  id: string;
  kind: 'NODE_OPERATION' | 'SCALE';
  risk: Risk;
  title: string;
  description: string;
  target: string;
  params: { op?: NodeOperation; nodeId?: string; app?: string; replicas?: number };
  /** HIGH / CRITICAL actions need the operator to type this exactly. */
  confirmText: string | null;
  expiresAt: number;
  status: 'PENDING_APPROVAL';
}

export interface CopilotAnswer {
  id: string;
  content: string;
  grounded: boolean;
  state: TruthState;
  citations: Citation[];
  proposedAction: ProposedAction | null;
  model: 'deterministic' | 'gemini';
}

/* ------------------------------------------------------------ docs index */

interface DocChunk {
  path: string;
  heading: string;
  text: string;
  words: Set<string>;
}

const STOP = new Set(['what', 'whats', "what's", 'the', 'and', 'are', 'how', 'why', 'does', 'right', 'now', 'wrong', 'with', 'this', 'that', 'there', 'can', 'you', 'your', 'my', 'for', 'show', 'doing', 'is', 'do']);
const tokenize = (s: string) =>
  new Set(
    s
      .toLowerCase()
      .split(/[^a-z0-9.-]+/)
      .filter((w) => w.length > 2 && !STOP.has(w))
  );

export class DocsIndex {
  private chunks: DocChunk[] = [];
  constructor(private root: string, dirs: string[]) {
    for (const d of dirs) this.walk(join(root, d));
  }
  private walk(p: string) {
    let st;
    try {
      st = statSync(p);
    } catch {
      return;
    }
    if (st.isDirectory()) {
      for (const f of readdirSync(p)) if (!f.startsWith('.')) this.walk(join(p, f));
      return;
    }
    if (!p.endsWith('.md')) return;
    const rel = relative(this.root, p);
    const text = readFileSync(p, 'utf8');
    let heading = rel;
    let buf: string[] = [];
    const flush = () => {
      const body = buf.join('\n').trim();
      if (body) this.chunks.push({ path: rel, heading, text: body, words: tokenize(heading + ' ' + body) });
      buf = [];
    };
    for (const line of text.split('\n')) {
      const m = /^#{1,3}\s+(.*)/.exec(line);
      if (m) {
        flush();
        heading = m[1];
      } else buf.push(line);
    }
    flush();
  }
  get size() {
    return this.chunks.length;
  }
  search(q: string, k = 2): DocChunk[] {
    const qw = tokenize(q);
    return this.chunks
      .map((c) => ({ c, s: [...qw].filter((w) => c.words.has(w)).length + ([...qw].some((w) => c.heading.toLowerCase().includes(w)) ? 2 : 0) }))
      .filter((x) => x.s >= 2)
      .sort((a, b) => b.s - a.s)
      .slice(0, k)
      .map((x) => x.c);
  }
}

/* ------------------------------------------------------- pending actions */

const pending = new Map<string, { action: ProposedAction; principal: string }>();
const principalOf = (token: string) => createHash('sha256').update(token).digest('hex');

export function takePendingAction(id: string, token: string): ProposedAction | 'NOT_FOUND' | 'WRONG_PRINCIPAL' | 'EXPIRED' {
  const p = pending.get(id);
  if (!p) return 'NOT_FOUND';
  if (p.principal !== principalOf(token)) return 'WRONG_PRINCIPAL';
  pending.delete(id);
  if (p.action.expiresAt < Date.now()) return 'EXPIRED';
  return p.action;
}

export function dismissPendingAction(id: string, token: string): boolean {
  const p = pending.get(id);
  if (!p || p.principal !== principalOf(token)) return false;
  pending.delete(id);
  return true;
}

/* ------------------------------------------------------------- answering */

const fmtAge = (ms: number | null) => (ms === null ? 'never observed' : ms < 1000 ? `${ms}ms ago` : `${Math.round(ms / 1000)}s ago`);

export function answer(input: {
  query: string;
  snapshot: RealitySnapshot | null;
  unavailableReason: string | null;
  docs: DocsIndex | null;
  token: string | null;
  canWrite: boolean;
}): CopilotAnswer {
  const q = input.query.trim();
  const ql = q.toLowerCase();
  const s = input.snapshot;
  const cites: Citation[] = [];
  const lines: string[] = [];
  let action: ProposedAction | null = null;

  const nodeCite = (n: RealitySnapshot['nodes'][number]) =>
    cites.push({ type: 'NODE_OBSERVATION', resourceId: n.id, label: `${n.name} observation #${n.observation.seq}`, observedAt: n.observation.observedAt, href: `/nodes/${n.id}` });
  const appCite = (a: RealitySnapshot['apps'][number]) =>
    cites.push({ type: 'DEPLOYMENT', resourceId: a.name, label: `${a.name} generation ${a.generation}`, observedAt: s?.overview.generatedAt ?? null, href: `/apps/${encodeURIComponent(a.name)}` });

  // Action intents first: they need a live snapshot to resolve targets.
  const opMatch = /^(drain|undrain|approve|revoke)\s+(?:(?:node|host)\s+)?([\w.-]+)/.exec(ql);
  const scaleMatch = /scale\s+([\w-]+)\s+(?:to\s+)?(\d{1,2})/.exec(ql);
  const wantsFacts = !!s;

  if (!s && (opMatch || scaleMatch || /node|host|app|deploy|replica|cert|tls|ssl|storage|volume|audit|evidence|incident|health|status|security|wrong|issue/.test(ql))) {
    lines.push(`I cannot verify current platform state: ${(input.unavailableReason ?? 'the control plane is unavailable').replace(/\.$/, '')}. I will not guess.`);
  }

  if (s && opMatch) {
    const [, verb, target] = opMatch;
    const node = s.nodes.find((n) => n.name.toLowerCase() === target || n.id.toLowerCase() === target);
    if (!node) {
      lines.push(`There is no host named "${target}" in the view you can read.`);
    } else if (!input.canWrite) {
      lines.push(`Your capability is read-only, so I cannot propose changes to ${node.name}.`);
      nodeCite(node);
    } else {
      const op = verb.toUpperCase() as NodeOperation;
      const risk: Risk = op === 'REVOKE' ? 'CRITICAL' : op === 'UNDRAIN' ? 'LOW' : 'MEDIUM';
      action = {
        id: randomUUID(),
        kind: 'NODE_OPERATION',
        risk,
        title: `${verb[0].toUpperCase()}${verb.slice(1)} ${node.name}`,
        description:
          op === 'DRAIN'
            ? 'The control plane stops placing new work on the host and moves replicas elsewhere. Existing admitted work is not killed by the control plane.'
            : op === 'UNDRAIN'
              ? 'The host becomes eligible for new placements again.'
              : op === 'APPROVE'
                ? 'The host is admitted to the cluster and may receive assignments.'
                : 'The host identity is revoked; it can no longer receive or report work. This cannot be undone from the console.',
        target: node.name,
        params: { op, nodeId: node.id },
        confirmText: risk === 'CRITICAL' ? node.name : null,
        expiresAt: Date.now() + 10 * 60_000,
        status: 'PENDING_APPROVAL'
      };
      lines.push(`${node.name} is ${node.health} (${node.healthReason}); lifecycle ${node.lifecycle}. I prepared the change below; nothing happens until you approve it.`);
      nodeCite(node);
    }
  } else if (s && scaleMatch) {
    const [, appName, nStr] = scaleMatch;
    const app = s.apps.find((a) => a.name === appName);
    const n = Number(nStr);
    if (!app) lines.push(`There is no application named "${appName}" in the view you can read.`);
    else if (!input.canWrite) {
      lines.push(`Your capability is read-only, so I cannot propose scaling ${app.name}.`);
      appCite(app);
    } else {
      const risk: Risk = n === 0 ? 'HIGH' : n < app.desiredReplicas ? 'MEDIUM' : 'LOW';
      action = {
        id: randomUUID(),
        kind: 'SCALE',
        risk,
        title: `Scale ${app.name} from ${app.desiredReplicas} to ${n} replica(s)`,
        description: 'Applies a new generation with the changed replica count; hosts admit or refuse under their own policy.',
        target: app.name,
        params: { app: app.name, replicas: n },
        confirmText: risk === 'HIGH' ? app.name : null,
        expiresAt: Date.now() + 10 * 60_000,
        status: 'PENDING_APPROVAL'
      };
      lines.push(`${app.name}: desired ${app.desiredReplicas}, observed running ${app.observedRunning}, healthy ${app.healthyReplicas} (${app.phase}).`);
      appCite(app);
    }
  } else if (wantsFacts) {
    const all = /overview|status|summary|everything|how is|how are/.test(ql);
    const diagnose = /wrong|issue|problem|diagnos|incident|fail|degrad|unhealthy|down/.test(ql);
    if (all || diagnose || /node|host|online|offline|heartbeat/.test(ql)) {
      const bad = s.nodes.filter((n) => n.health !== 'HEALTHY');
      const named = s.nodes.filter((n) => ql.includes(n.name.toLowerCase()));
      const list = named.length ? named : diagnose ? bad : s.nodes;
      if (!named.length) {
        const m = s.overview.metrics;
        lines.push(`${m.nodesHealthy.value} of ${m.nodesTotal.value} host(s) have a fresh signed observation; ${m.nodesDegraded.value} stale, ${m.nodesOffline.value} offline, ${m.nodesUnknown.value} never observed.`);
      }
      for (const n of list.slice(0, 6)) {
        lines.push(`• ${n.name}: ${n.health} — ${n.healthReason}; last observation ${fmtAge(n.observation.ageMs)}; ${n.workloads ?? 'unknown'} workload(s).`);
        nodeCite(n);
      }
    }
    if (all || diagnose || /app|deploy|replica|workload/.test(ql)) {
      const list = diagnose ? s.apps.filter((a) => a.phase !== 'READY') : s.apps;
      if (!list.length && diagnose) lines.push('Every application has all desired replicas observed running and passing health checks.');
      for (const a of list.slice(0, 6)) {
        lines.push(`• ${a.name} (generation ${a.generation}): ${a.phase} — desired ${a.desiredReplicas}, admitted ${a.admitted}, observed running ${a.observedRunning}, healthy ${a.healthyReplicas}.`);
        appCite(a);
      }
    }
    if (all || diagnose || /cert|tls|ssl|https/.test(ql)) {
      const list = diagnose ? s.certificates.filter((c) => c.state !== 'VALID') : s.certificates;
      if (!s.certificates.length && !diagnose) lines.push('No edge host reports a certificate, so TLS state is unknown.');
      for (const c of list) {
        lines.push(`• ${c.host}: ${c.state} — issuer ${c.issuer}, expires ${c.notAfter ? new Date(c.notAfter).toISOString() : 'unknown'} (observed by ${c.observedBy}).`);
        cites.push({ type: 'CERTIFICATE', resourceId: c.host, label: `X.509 ${c.fingerprint.slice(0, 19)}…`, observedAt: s.overview.generatedAt, href: '/security' });
      }
    }
    if (all || diagnose || /storage|volume|disk|snapshot|replicat/.test(ql)) {
      const list = diagnose ? s.volumes.filter((v) => v.state !== 'HEALTHY') : s.volumes;
      for (const v of list.slice(0, 6)) {
        lines.push(`• volume ${v.id}: ${v.state}${v.detail ? ` — ${v.detail}` : ''}; ${v.verified}/${v.durabilityReplicas} replica(s) verified the committed snapshot.`);
        cites.push({ type: 'VOLUME', resourceId: v.id, label: `volume ${v.id}`, observedAt: v.committed?.ts ?? null, href: '/storage' });
      }
    }
    if (all || diagnose || /audit|evidence|ledger|verif|tamper/.test(ql)) {
      const ver = s.evidence.verification;
      lines.push(`Audit ledger: ${ver.state} — ${ver.entries} entries, head ${ver.head.slice(0, 18)}…, checked by ${ver.verifiedBy.slice(0, 12) || 'nobody'}.`);
      cites.push({ type: 'EVIDENCE', resourceId: ver.head, label: `ledger verification (${ver.entries} entries)`, observedAt: ver.verifiedAt, href: '/evidence' });
      if (/audit|recent|last|event/.test(ql)) {
        for (const e of s.evidence.entries.slice(0, 4)) {
          lines.push(`• #${e.seq} ${e.action} ${e.resource} by ${e.actor.slice(0, 20)}`);
          cites.push({ type: 'EVENT', resourceId: String(e.seq), label: `audit #${e.seq}`, observedAt: e.ts, href: '/evidence' });
        }
      }
    }
    if (diagnose && s.overview.incidents.length) {
      for (const i of s.overview.incidents) lines.push(`• incident: ${i}`);
      cites.push({ type: 'EVENT', resourceId: 'overview.incidents', label: 'control-plane incidents', observedAt: s.overview.generatedAt, href: '/' });
    }
    if (/security|waf|ddos|firewall/.test(ql)) {
      for (const c of s.security) lines.push(`• ${c.title}: ${c.result} — ${c.detail}`);
      cites.push({ type: 'CONFIGURATION', resourceId: 'security', label: 'derived security controls', observedAt: s.overview.generatedAt, href: '/security' });
    }
  }

  // Documentation retrieval always runs; it is labelled as documentation, not observation.
  const howTo = /\b(how (do|to|can)|explain|runbook|install|join|configure|set ?up|what is|why does)\b/.test(ql);
  const docs = howTo || cites.length === 0 ? input.docs?.search(q) ?? [] : [];
  if (docs.length && !action) {
    for (const d of docs) {
      const excerpt = d.text.replace(/\s+/g, ' ').slice(0, 320);
      lines.push(`From ${d.path} › ${d.heading}: ${excerpt}${d.text.length > 320 ? '…' : ''}`);
      cites.push({ type: 'DOCUMENTATION', resourceId: d.path, label: `${d.path} › ${d.heading}`, observedAt: null, href: null });
    }
  }

  if (action) pending.set(action.id, { action, principal: principalOf(input.token ?? '') });

  const grounded = cites.length > 0;
  if (!lines.length) {
    lines.push("I don't have enough verified platform information to answer that. Try asking about hosts, applications, certificates, storage, the audit ledger, or say \"drain <host>\" / \"scale <app> to <n>\".");
  }
  return {
    id: randomUUID(),
    content: lines.join('\n'),
    grounded,
    state: !s ? 'UNKNOWN' : grounded ? 'DERIVED' : 'UNKNOWN',
    citations: cites,
    proposedAction: action,
    model: 'deterministic'
  };
}

/**
 * Optional model rewrite (COPILOT_MODEL=gemini). The model only rephrases the
 * retrieved, cited facts; it is told to add nothing. Off by default because it
 * sends retrieved platform data to a third party.
 */
export async function rephraseWithModel(query: string, base: CopilotAnswer): Promise<CopilotAnswer> {
  const { GoogleGenAI } = await import('@google/genai');
  const ai = new GoogleGenAI({});
  const prompt = [
    'You are the Decentralized.Host operations assistant.',
    'Answer the operator question using ONLY the facts below. Do not add numbers, hosts, or claims that are not in the facts.',
    'If the facts do not answer the question, say you cannot verify it.',
    `Question: ${query}`,
    'Facts:',
    base.content
  ].join('\n');
  const res = await ai.models.generateContent({ model: process.env.COPILOT_MODEL_NAME || 'gemini-2.5-flash', contents: prompt });
  const text = res.text?.trim();
  return text ? { ...base, content: text, model: 'gemini' } : base;
}
