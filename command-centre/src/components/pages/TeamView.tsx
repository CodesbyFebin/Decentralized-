import React from 'react';
import { KeyRound, Terminal, Check, X } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill } from '../common/ui';
import { Unavailable, fmtTime, Note } from '../common/states';
import { roleOf } from '../layout/TopBar';

const ACTIONS: { id: string; grants: string[] }[] = [
  { id: 'api.read', grants: ['view hosts, apps, storage, domains, evidence', 'read audit ledger and logs', 'use the Copilot (read-only)'] },
  { id: 'api.write', grants: ['apply manifests (deploy, scale, delete)', 'drain / undrain hosts', 'approve Copilot actions of that kind'] },
  { id: 'api.admin', grants: ['approve and revoke hosts', 'create join invites', 'federation, roster and backups'] }
];

export default function TeamView() {
  const { session, mode, capabilities } = useSession();
  const actions = session?.actions ?? [];
  return (
    <div className="space-y-4 pt-2 max-w-4xl">
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
      <Glass className="p-5">
        <PanelHeader icon={<IconTile tone="violet" size="sm"><Terminal className="w-4 h-4" /></IconTile>} title="Give someone access" subtitle="Mint a scoped, expiring capability with the operator CLI and share it privately." />
        <pre className="mt-3 rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.12)] p-3 text-[12px] text-cyan-100 font-mono">{`dh token --read-only --ttl 12h     # viewer
dh token --ttl 8h                  # operator (read, write, admin)
dh console --read-only --ttl 12h   # a console URL that signs in directly`}</pre>
        <Note>Capabilities cannot be listed or revoked individually from the console; they expire. Rotate the root key (dh root-rotate) to invalidate every outstanding capability.</Note>
      </Glass>
      <Unavailable title="Team directory & invitations" detail={capabilities?.items.teamDirectory.detail ?? 'not implemented'} />
    </div>
  );
}
