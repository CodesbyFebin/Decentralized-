import React, { useState } from 'react';
import { KeyRound, Terminal, Check, X, Plus, Clock, Trash2, Copy, AlertTriangle } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Unavailable, fmtTime, Note, since } from '../common/states';
import { roleOf } from '../layout/TopBar';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/api';

const ACTIONS: { id: string; grants: string[] }[] = [
  { id: 'api.read', grants: ['view hosts, apps, storage, domains, evidence', 'read audit ledger and logs', 'use the Copilot (read-only)'] },
  { id: 'api.write', grants: ['apply manifests (deploy, scale, delete)', 'drain / undrain hosts', 'approve Copilot actions of that kind'] },
  { id: 'api.admin', grants: ['approve and revoke hosts', 'create join invites', 'federation, roster and backups'] }
];

interface TeamData {
  members: { actor: string; role: string; joinedAt: number; lastSeen: number }[];
  invitations: { nonce: string; role: string; createdAt: number; expiresAt: number; state: 'ACTIVE' | 'USED' | 'EXPIRED' | 'REVOKED' }[];
}

export default function TeamView() {
  const { session, mode, capabilities, can } = useSession();
  const actions = session?.actions ?? [];
  const teamRes = useResource<TeamData>('/team');
  const [inviteRole, setInviteRole] = useState<'api.read' | 'api.write' | 'api.admin'>('api.read');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const [revokingNonce, setRevokingNonce] = useState<string | null>(null);
  const [confirmedNonce, setConfirmedNonce] = useState('');

  const createInvite = async () => {
    if (!can('api.admin')) return;
    setBusy(true);
    setErr('');
    setSuccessMsg('');
    try {
      const res = await api('POST', '/team/invitations', {
        body: { role: inviteRole, ttl: 86400 }
      });
      if (res.ok) {
        setSuccessMsg(`Invitation created for ${inviteRole} · expires in 24h`);
        setInviteRole('api.read');
        teamRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to create invitation');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error creating invitation');
    } finally {
      setBusy(false);
    }
  };

  const revokeInvitation = async (nonce: string) => {
    if (confirmedNonce !== nonce) {
      setRevokingNonce(nonce);
      return;
    }
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/team/invitations/${nonce}/revoke`, { body: {} });
      if (res.ok) {
        setSuccessMsg('Invitation revoked');
        setRevokingNonce(null);
        setConfirmedNonce('');
        teamRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to revoke invitation');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error revoking invitation');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Team & access</h1>
        <p className="text-[13px] text-slate-400">Operators are identified by signed capabilities chained to your cluster root. The control plane enforces them on every request.</p>
      </div>
      <Glass className="p-5">
        <PanelHeader icon={<IconTile tone="cyan" size="sm"><KeyRound className="w-4 h-4" /></IconTile>} title="This session" />
        {session?.authenticated ? (
          <div className="mt-3 grid sm:grid-cols-2 gap-x-6 gap-y-1.5 text-[12.5px]">
            <div className="flex justify-between"><span className="text-slate-400">Actor (as audited)</span><span className="text-slate-100">{session.actor}</span></div>
            <div className="flex justify-between"><span className="text-slate-400">Role</span><StatusPill status={roleOf(actions)} dot={false} tone="cyan" /></div>
            <div className="flex justify-between"><span className="text-slate-400">Cluster</span><span className="text-slate-100">{session.cluster ?? '—'}</span></div>
            <div className="flex justify-between"><span className="text-slate-400">Expires</span><span className="text-slate-100">{fmtTime(session.expiresAt)}</span></div>
          </div>
        ) : (
          <p className="mt-2 text-[12.5px] text-slate-400">{mode === 'demo' ? 'Demo replay: no session is needed and nothing can be changed.' : 'Not signed in.'}</p>
        )}
        <table className="dh-table w-full mt-4">
          <thead><tr><th>Action</th><th>Granted</th><th>Allows</th></tr></thead>
          <tbody>
            {ACTIONS.map((a) => (
              <tr key={a.id}>
                <td className="font-mono text-cyan-200">{a.id}</td>
                <td>{actions.includes(a.id) ? <Check className="w-4 h-4 text-emerald-400" /> : <X className="w-4 h-4 text-slate-600" />}</td>
                <td className="text-slate-300 whitespace-normal">{a.grants.join(' · ')}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Glass>

      {teamRes.loading ? (
        <Glass className="p-5"><p className="text-slate-400">Loading team data...</p></Glass>
      ) : teamRes.data ? (
        <>
          {teamRes.data.members.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Team members" subtitle={`${teamRes.data.members.length} active operator${teamRes.data.members.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 overflow-x-auto">
                <table className="dh-table w-full min-w-[600px]">
                  <thead><tr><th>Actor</th><th>Role</th><th>Joined</th><th>Last seen</th></tr></thead>
                  <tbody>
                    {teamRes.data.members.map((m) => (
                      <tr key={m.actor}>
                        <td className="text-slate-100 font-mono text-[12px]">{m.actor}</td>
                        <td><StatusPill status={m.role === 'api.admin' ? 'admin' : m.role === 'api.write' ? 'operator' : 'viewer'} dot={false} /></td>
                        <td className="text-slate-400 text-[12px]">{since(m.joinedAt)}</td>
                        <td className="text-slate-400 text-[12px]">{since(m.lastSeen)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Glass>
          )}

          <Glass className="p-5">
            <PanelHeader title="Create invitation" subtitle="Mint a time-limited, role-scoped capability token" />
            {err && <div className="mt-3 text-sm text-rose-300 bg-rose-500/10 p-2 rounded">{err}</div>}
            {successMsg && <div className="mt-3 text-sm text-emerald-300 bg-emerald-500/10 p-2 rounded">{successMsg}</div>}
            {!can('api.admin') ? (
              <p className="mt-3 text-[12.5px] text-slate-400">Only api.admin can create invitations.</p>
            ) : (
              <div className="mt-3 space-y-3">
                <div>
                  <label className="text-[12px] text-slate-400 block mb-2">Role</label>
                  <select
                    value={inviteRole}
                    onChange={(e) => setInviteRole(e.target.value as any)}
                    disabled={busy}
                    className="w-full md:w-48 px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                  >
                    <option value="api.read">Viewer (read-only)</option>
                    <option value="api.write">Operator (read + write)</option>
                    <option value="api.admin">Admin (full access)</option>
                  </select>
                </div>
                <button
                  onClick={createInvite}
                  disabled={busy}
                  className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:bg-slate-700 text-white text-sm rounded font-medium transition"
                >
                  <Plus className="w-4 h-4" /> Create invitation
                </button>
              </div>
            )}
          </Glass>

          {teamRes.data.invitations.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Pending invitations" subtitle={`${teamRes.data.invitations.filter(i => i.state === 'ACTIVE').length} active`} />
              <div className="mt-3 space-y-2">
                {teamRes.data.invitations.map((inv) => (
                  <div key={inv.nonce} className="flex items-center justify-between p-3 rounded-lg bg-white/3 border border-white/5">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <div className="font-mono text-[11px] text-cyan-200 truncate">{inv.nonce.slice(0, 12)}...</div>
                        <span className="text-[11px] px-2 py-1 rounded bg-white/10 text-slate-300">{inv.role === 'api.admin' ? 'admin' : inv.role === 'api.write' ? 'operator' : 'viewer'}</span>
                        {inv.state === 'ACTIVE' && <span className="text-[11px] px-2 py-1 rounded bg-emerald-500/20 text-emerald-300">active</span>}
                        {inv.state === 'USED' && <span className="text-[11px] px-2 py-1 rounded bg-emerald-500/20 text-emerald-300">used</span>}
                        {inv.state === 'EXPIRED' && <span className="text-[11px] px-2 py-1 rounded bg-slate-500/20 text-slate-400">expired</span>}
                        {inv.state === 'REVOKED' && <span className="text-[11px] px-2 py-1 rounded bg-rose-500/20 text-rose-300">revoked</span>}
                      </div>
                      <div className="flex items-center gap-4 text-[11px] text-slate-400">
                        <span className="flex items-center gap-1"><Clock className="w-3 h-3" />expires {since(inv.expiresAt)}</span>
                      </div>
                    </div>
                    {can('api.admin') && inv.state === 'ACTIVE' && (
                      revokingNonce === inv.nonce ? (
                        <div className="flex items-center gap-2">
                          <input
                            type="text"
                            placeholder="confirm nonce"
                            value={confirmedNonce}
                            onChange={(e) => setConfirmedNonce(e.target.value)}
                            className="w-24 px-2 py-1 bg-white/5 border border-white/10 rounded text-[11px] text-slate-100"
                          />
                          <button
                            onClick={() => revokeInvitation(inv.nonce)}
                            disabled={busy || confirmedNonce !== inv.nonce}
                            className="px-2 py-1 bg-rose-600 hover:bg-rose-500 disabled:bg-slate-700 text-white text-[11px] rounded"
                          >
                            Confirm
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setRevokingNonce(inv.nonce)}
                          className="p-1.5 text-slate-400 hover:text-rose-300 transition"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )
                    )}
                  </div>
                ))}
              </div>
            </Glass>
          )}
        </>
      ) : null}

      <Glass className="p-5">
        <PanelHeader icon={<IconTile tone="violet" size="sm"><Terminal className="w-4 h-4" /></IconTile>} title="Mint a capability token" subtitle="Create scoped, expiring tokens with the operator CLI" />
        <pre className="mt-3 rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.12)] p-3 text-[12px] text-cyan-100 font-mono">{`dh token --read-only --ttl 12h     # viewer
dh token --ttl 8h                  # operator (read, write, admin)
dh console --read-only --ttl 12h   # a console URL that signs in directly`}</pre>
        <Note>The console can create and revoke invitations; capabilities themselves cannot be individually revoked. They expire on schedule, or all expire when you rotate the root key (dh root-rotate).</Note>
      </Glass>
    </div>
  );
}
