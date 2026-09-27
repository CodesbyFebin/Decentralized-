import React, { useState } from 'react';
import { Users, Plus, Shield, Key, Mail, CheckCircle2 } from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Team: React.FC = () => {
  const { addToast } = useNetwork();
  const [members, setMembers] = useState([
    { name: 'Febin Francis', email: 'kochiewaste@gmail.com', role: 'Owner', key: 'ed25519:febin_root...a9f1', status: 'Active' },
    { name: 'Chithra Francis', email: 'chithra@decentralized.host', role: 'Operator', key: 'ed25519:chithra...d482', status: 'Pending Invite' },
  ]);

  const [inviteEmail, setInviteEmail] = useState('');
  const [showInvite, setShowInvite] = useState(false);

  const handleInvite = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail) return;
    setMembers((prev) => [
      ...prev,
      { name: inviteEmail.split('@')[0], email: inviteEmail, role: 'Operator', key: 'Pending enrollment...', status: 'Pending Invite' },
    ]);
    addToast(`Invitation sent to ${inviteEmail} with cryptographic enrollment challenge!`, 'success');
    setInviteEmail('');
    setShowInvite(false);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Users className="w-4 h-4 text-cyan-400" />
            <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
              Access Control & Operator Keys
            </span>
          </div>
          <h1 className="text-2xl font-bold text-white tracking-tight">
            Team & Cryptographic Quorum
          </h1>
          <p className="text-xs text-slate-300 mt-1">
            Manage authorized operators. All actions require signed cryptographic credentials.
          </p>
        </div>

        <button
          onClick={() => setShowInvite(true)}
          className="px-4 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/20 transition-all flex items-center gap-1.5"
        >
          <Plus className="w-4 h-4" />
          <span>Invite Member</span>
        </button>
      </div>

      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
        <div className="overflow-x-auto custom-scrollbar">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                <th className="py-3 px-3">Member</th>
                <th className="py-3 px-2">Role</th>
                <th className="py-3 px-2">Public Key Fingerprint</th>
                <th className="py-3 px-2">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {members.map((m) => (
                <tr key={m.email} className="hover:bg-slate-800/40 transition-colors">
                  <td className="py-3.5 px-3">
                    <span className="font-bold text-white block">{m.name}</span>
                    <span className="text-slate-400 text-[11px]">{m.email}</span>
                  </td>
                  <td className="py-3.5 px-2 text-cyan-300">{m.role}</td>
                  <td className="py-3.5 px-2 text-purple-300">{m.key}</td>
                  <td className="py-3.5 px-2">
                    <span className={`px-2 py-0.5 rounded-full text-[10px] ${
                      m.status === 'Active' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-amber-500/15 text-amber-400'
                    }`}>
                      ● {m.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {showInvite && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md">
          <form onSubmit={handleInvite} className="w-full max-w-md p-6 rounded-3xl bg-slate-900 border border-blue-500/30 text-white space-y-4 text-xs font-mono">
            <h3 className="text-base font-bold font-sans">Invite Team Member</h3>
            <div>
              <label className="block text-slate-400 mb-1">Email Address</label>
              <input
                type="email"
                placeholder="colleague@domain.com"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700 text-white"
              />
            </div>
            <div className="flex justify-end gap-2 pt-2 font-sans">
              <button
                type="button"
                onClick={() => setShowInvite(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-4 py-2 rounded-xl bg-cyan-600 hover:bg-cyan-500 font-bold text-white"
              >
                Send Invite
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
