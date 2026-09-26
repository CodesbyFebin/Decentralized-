import React, { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Server, Laptop, Home, Cloud, Cpu, CircuitBoard, HelpCircle, KeyRound, Copy, Check, ShieldCheck, XCircle, CheckCircle2, Circle, Loader2 } from 'lucide-react';
import type { InviteRec, NodeRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError, type MutationResult } from '../../lib/client';
import { Glass, PanelHeader, GhostButton, StatusPill, type Tone } from '../common/ui';
import { Gate, Note, ErrorState, fmtTime, since, fmtAge, TruthTag } from '../common/states';

type Kind = 'this' | 'linux' | 'home' | 'vps' | 'arm' | 'gpu' | 'other';

const KINDS: { id: Kind; label: string; icon: React.ReactNode; arch: string; note: string }[] = [
  { id: 'this', label: 'This computer', icon: <Laptop className="w-5 h-5" />, arch: 'amd64', note: 'Runs the agent next to your other work. Keep the host policy strict (policy.yaml) so shared work cannot take what you need.' },
  { id: 'linux', label: 'Linux server', icon: <Server className="w-5 h-5" />, arch: 'amd64', note: 'A dedicated server you control.' },
  { id: 'home', label: 'Home server', icon: <Home className="w-5 h-5" />, arch: 'amd64', note: 'Behind a home router: the mesh needs a reachable UDP endpoint (--mesh-advertise). NAT traversal is not implemented yet.' },
  { id: 'vps', label: 'VPS', icon: <Cloud className="w-5 h-5" />, arch: 'amd64', note: 'Open the WireGuard UDP port (default 51820) in the provider firewall.' },
  { id: 'arm', label: 'Raspberry Pi / ARM', icon: <CircuitBoard className="w-5 h-5" />, arch: 'arm64', note: 'Build for arm64. Measured facts report the board’s real memory and disks.' },
  { id: 'gpu', label: 'GPU machine', icon: <Cpu className="w-5 h-5" />, arch: 'amd64', note: 'GPUs are discovered (sysfs, nvidia-smi) and reported as facts. GPU scheduling is not implemented.' },
  { id: 'other', label: 'Other', icon: <HelpCircle className="w-5 h-5" />, arch: 'amd64', note: 'Any Linux or macOS host that can run a Go binary.' }
];

const FLOW = ['Create invite', 'Install agent', 'Node connects', 'Verify identity', 'Discover hardware', 'Owner approves', 'Active'] as const;

/** How far a used invite's host has come, from the host's own record. */
function flowIndex(inv: InviteRec | null, node: NodeRec | undefined): number {
  if (!inv) return 0;
  if (!node) return inv.state === 'USED' ? 3 : 1;
  if (node.lifecycle === 'ACTIVE' || node.lifecycle === 'DRAINING') return node.health === 'HEALTHY' ? 7 : 6;
  if (node.facts) return 5;
  return 4;
}

const CopyBlock: React.FC<{ label: string; text: string }> = ({ label, text }) => {
  const [ok, setOk] = useState(false);
  return (
    <div className="mt-2">
      <div className="flex items-center justify-between text-[11.5px] text-slate-400">
        <span>{label}</span>
        <button
          type="button"
          onClick={() => navigator.clipboard?.writeText(text).then(() => setOk(true), () => undefined)}
          className="inline-flex items-center gap-1 text-cyan-300 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 rounded"
        >
          {ok ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />} {ok ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className="mt-1 p-3 rounded-xl bg-[#050a14]/90 border border-[rgba(125,190,255,0.12)] text-[12px] text-slate-200 font-mono whitespace-pre-wrap break-all">{text}</pre>
    </div>
  );
};

const INVITE_TONE: Record<InviteRec['state'], Tone> = { ACTIVE: 'blue', USED: 'emerald', REVOKED: 'slate', EXPIRED: 'amber' };

export default function AddNodeView() {
  const { can, session } = useSession();
  const [params] = useSearchParams();
  const [kind, setKind] = useState<Kind>(() => (KINDS.some((x) => x.id === params.get('kind')) ? (params.get('kind') as Kind) : 'linux'));
  const [name, setName] = useState('host-1');
  const [region, setRegion] = useState('');
  const [busy, setBusy] = useState<string | null>(null);
  const [opErr, setOpErr] = useState<ApiError | null>(null);
  const [opMsg, setOpMsg] = useState<string | null>(null);
  const admin = can('api.admin');
  const invites = useResource<{ invites: InviteRec[]; serverTime: number }>(admin ? '/invites' : null, { pollMs: 5_000 });
  const nodes = useResource<{ nodes: NodeRec[] }>('/nodes', { pollMs: 5_000 });
  const k = KINDS.find((x) => x.id === kind)!;
  const safeName = name.replace(/[^a-z0-9-]/gi, '').toLowerCase() || 'host-1';
  const safeRegion = region.replace(/[^a-z0-9-]/gi, '').toLowerCase();

  const operatorCmd = `dh node invite --out ${safeName}.token        # single use, expires in 15 minutes (--ttl to change)`;
  const buildCmd = `GOOS=linux GOARCH=${k.arch} go build -o dh-noded ./cmd/dh-noded   # no signed release packages exist yet`;
  const hostCmd = [
    `./dh-noded --data /var/lib/dh-noded --join-file ${safeName}.token \\`,
    `  --name ${safeName}${safeRegion ? ` --region ${safeRegion}` : ''} \\`,
    `  --mesh 0.0.0.0:51820 --mesh-advertise <public-address>:51820`
  ].join('\n');

  const act = async (id: string, fn: () => Promise<{ data: MutationResult }>) => {
    setBusy(id);
    setOpErr(null);
    setOpMsg(null);
    try {
      const r = await fn();
      setOpMsg(`${r.data.ok ? '✓' : '✗'} ${r.data.message}`);
      invites.refresh();
      nodes.refresh();
    } catch (e) {
      setOpErr(e as ApiError);
    } finally {
      setBusy(null);
    }
  };

  const allNodes = nodes.data?.nodes ?? [];
  const pending = allNodes.filter((n) => n.lifecycle === 'PENDING_APPROVAL');
  const skew = invites.data ? invites.data.serverTime - (invites.fetchedAt ?? Date.now()) : 0;

  return (
    <div className="space-y-4 pt-2">
      <div>
        <div className="text-[12px] text-slate-400"><Link to="/nodes" className="hover:text-cyan-300">Nodes</Link> / Add node</div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Add a node</h1>
        <p className="text-[13px] text-slate-400 max-w-3xl">The node creates and keeps its own private identity. Only its public key reaches the control plane; the console never sees or stores a node key.</p>
      </div>

      <Glass className="p-4">
        <PanelHeader title="1 · What are you adding?" />
        <div className="mt-3 grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-7 gap-2" role="radiogroup" aria-label="Node type">
          {KINDS.map((x) => (
            <button
              key={x.id}
              type="button"
              role="radio"
              aria-checked={kind === x.id}
              onClick={() => setKind(x.id)}
              className={`flex flex-col items-center gap-1.5 p-3 rounded-xl border text-[12.5px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 ${kind === x.id ? 'border-cyan-400/60 bg-cyan-400/10 text-white' : 'border-[rgba(125,190,255,0.12)] text-slate-300 hover:border-cyan-400/30'}`}
            >
              {x.icon}
              {x.label}
            </button>
          ))}
        </div>
        <Note>{k.note}</Note>
      </Glass>

      <Glass className="p-4">
        <PanelHeader icon={<KeyRound className="w-4 h-4 text-cyan-300" />} title="2 · Create an invite (on your operator machine)" subtitle="A join token is signed by the cluster root key. The root key never leaves the operator CLI, so the console cannot create tokens itself." />
        <div className="mt-3 grid sm:grid-cols-2 gap-3 max-w-xl">
          <label className="text-[12px] text-slate-300">Host name
            <input value={name} onChange={(e) => setName(e.target.value)} className="mt-1 w-full dh-input" aria-describedby="name-help" />
          </label>
          <label className="text-[12px] text-slate-300">Region (optional)
            <input value={region} onChange={(e) => setRegion(e.target.value)} placeholder="e.g. eu-west" className="mt-1 w-full dh-input" />
          </label>
        </div>
        <p id="name-help" className="sr-only">Used only to fill in the commands below.</p>
        <CopyBlock label="Operator machine" text={operatorCmd} />
        <PanelHeader title="3 · Install and start the agent (on the new host)" />
        <CopyBlock label={`Build for ${k.arch}`} text={buildCmd} />
        <CopyBlock label="Start with the token" text={hostCmd} />
        <Note>The agent generates its Ed25519 identity in its data directory (mode 0600), signs its enrolment with it, and reports measured hardware in signed observations. Nothing runs on it until you approve it below.</Note>
      </Glass>

      <Glass className="p-4">
        <PanelHeader icon={<ShieldCheck className="w-4 h-4 text-emerald-300" />} title="4 · Approve" subtitle="Hosts that enrolled with a token and wait for the owner." />
        {nodes.error && !nodes.data ? (
          <ErrorState error={nodes.error} onRetry={nodes.refresh} />
        ) : pending.length === 0 ? (
          <p className="mt-3 text-[12.5px] text-slate-400">{nodes.data ? 'No host is waiting for approval.' : 'Loading…'}</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {pending.map((n) => (
              <li key={n.id} className="flex flex-wrap items-center gap-3 p-3 rounded-xl border border-[rgba(125,190,255,0.12)]">
                <Server className="w-4 h-4 text-slate-400" aria-hidden />
                <div className="min-w-0 flex-1">
                  <Link to={`/nodes/${encodeURIComponent(n.id)}`} className="font-semibold text-slate-100 hover:text-cyan-300">{n.name}</Link>
                  <div className="text-[11px] text-slate-400 font-mono">{n.id}</div>
                  <div className="text-[11px] text-slate-400">{n.os}/{n.arch} · {n.facts ? `${n.facts.cpus} CPU, ${n.facts.memBytes === null ? 'memory not measured' : `${Math.round(n.facts.memBytes / 2 ** 30)} GiB measured`}` : 'no observation yet'} · enrolled {since(n.joinedAt)}</div>
                </div>
                {admin ? (
                  <GhostButton disabled={busy === n.id} onClick={() => act(n.id, () => api.post(`/nodes/${encodeURIComponent(n.id)}/operations`, { type: 'APPROVE' }))}>
                    {busy === n.id ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />} Approve
                  </GhostButton>
                ) : (
                  <span className="text-[11.5px] text-slate-400">approval needs api.admin</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </Glass>

      <Glass className="p-4">
        <PanelHeader title="Invites" subtitle="As recorded by the control plane. The token itself is never stored there; each invite is identified by its single-use nonce." right={<TruthTag state="LIVE" />} />
        {opMsg && <p role="status" className="mt-2 text-[12.5px] text-slate-200">{opMsg}</p>}
        {opErr && <div className="mt-2"><ErrorState error={opErr} /></div>}
        {!admin ? (
          <p className="mt-3 text-[12.5px] text-slate-400">Listing and revoking invites needs a capability with <code>api.admin</code>{session?.actor ? ` (signed in as ${session.actor})` : ''}.</p>
        ) : (
          <Gate res={invites}>
            {(d, stale) =>
              d.invites.length === 0 ? (
                <p className="mt-3 text-[12.5px] text-slate-400">No invites have been created.</p>
              ) : (
                <div className="overflow-x-auto mt-2">
                  <table className="dh-table w-full min-w-[760px]">
                    <thead><tr><th>Invite</th><th>State</th><th>Created</th><th>Expires</th><th>Note</th><th>Progress</th><th /></tr></thead>
                    <tbody>
                      {d.invites.map((inv) => {
                        const node = allNodes.find((n) => n.id === inv.usedBy);
                        const idx = flowIndex(inv, node);
                        const left = inv.expiresAt ? inv.expiresAt - (Date.now() + skew) : null;
                        return (
                          <tr key={inv.nonce}>
                            <td className="font-mono text-[11.5px]">{inv.nonce.slice(0, 8)}…{inv.autoApprove && <span className="ml-1 text-[10.5px] text-amber-300">auto-approve</span>}</td>
                            <td><StatusPill status={inv.state} tone={INVITE_TONE[inv.state]} /></td>
                            <td title={fmtTime(inv.createdAt)}>{since(inv.createdAt)}</td>
                            <td title={inv.expiresAt ? fmtTime(inv.expiresAt) : undefined}>{inv.state === 'ACTIVE' && left !== null ? `in ${fmtAge(left)}` : inv.expiresAt ? fmtTime(inv.expiresAt) : '—'}</td>
                            <td className="text-slate-300">{inv.note || '—'}</td>
                            <td>
                              {inv.state === 'USED' ? (
                                <span className="text-[11.5px]">
                                  {node ? <Link to={`/nodes/${encodeURIComponent(node.id)}`} className="text-cyan-300 hover:underline">{node.name}</Link> : <span className="font-mono">{inv.usedBy?.slice(0, 10)}</span>} · {FLOW[Math.min(idx, FLOW.length - 1)]}
                                </span>
                              ) : (
                                <span className="text-[11.5px] text-slate-400">{inv.state === 'ACTIVE' ? 'waiting for a host' : '—'}</span>
                              )}
                            </td>
                            <td>
                              {inv.state === 'ACTIVE' && (
                                <GhostButton disabled={stale || busy === inv.nonce} onClick={() => act(inv.nonce, () => api.post(`/invites/${inv.nonce}/revoke`))} className="!py-1 !text-[12px]">
                                  {busy === inv.nonce ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <XCircle className="w-3.5 h-3.5" />} Revoke
                                </GhostButton>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                  <FlowLegend />
                  <Note>To replace a revoked or expired invite, run <code>dh node invite</code> again; each token admits exactly one host.</Note>
                </div>
              )
            }
          </Gate>
        )}
      </Glass>
    </div>
  );
}

const FlowLegend: React.FC = () => (
  <ol className="mt-3 flex flex-wrap items-center gap-1.5 text-[11px] text-slate-400" aria-label="Enrolment steps">
    {FLOW.map((s, i) => (
      <li key={s} className="inline-flex items-center gap-1">
        <Circle className="w-2.5 h-2.5" aria-hidden />
        {i + 1}. {s}
      </li>
    ))}
  </ol>
);
