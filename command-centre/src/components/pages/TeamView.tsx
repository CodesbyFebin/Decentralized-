import React, { useEffect, useState } from 'react';
import {
  Users,
  UserPlus,
  Copy,
  Check,
  Search,
  CheckCircle2,
  Shield,
  Layers,
  Clock,
  MoreVertical,
  RefreshCw,
  X
} from 'lucide-react';
import { TeamMember, TeamPermissionMatrix } from '../../types/platform';
import { api } from '../../lib/api';
import { MetricCard } from '../common/MetricCard';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute) => void;
}

export const TeamView: React.FC<Props> = ({ onNavigate }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [roleFilter, setRoleFilter] = useState('All Roles');
  const [copiedLink, setCopiedLink] = useState(false);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('Developer');

  const loadTeam = async () => {
    try {
      setLoading(true);
      const res = await api.getTeam();
      setData(res);
    } catch (err) {
      console.error('Failed to load team:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTeam();
  }, []);

  const handleCopy = () => {
    if (!data?.inviteLink) return;
    navigator.clipboard.writeText(data.inviteLink);
    setCopiedLink(true);
    setTimeout(() => setCopiedLink(false), 2000);
  };

  const handleTogglePermission = async (permKey: keyof TeamPermissionMatrix) => {
    if (!data?.permissions) return;
    const updated = {
      ...data.permissions,
      [permKey]: !data.permissions[permKey]
    };
    setData({ ...data, permissions: updated });
    await api.updateTeamPermissions(updated);
  };

  const handleInviteSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail.trim()) return;

    try {
      const res = await api.inviteTeamMember({
        email: inviteEmail.trim(),
        role: inviteRole
      });

      setData((prev: any) => ({
        ...prev,
        members: [...prev.members, res.data]
      }));
      setShowInviteModal(false);
      setInviteEmail('');
      alert(`Invitation sent to ${inviteEmail}. Access token signed on authority keyring.`);
    } catch (err: any) {
      alert(`Invite failed: ${err.message}`);
    }
  };

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  const { summary, members, permissions, inviteLink } = data;

  const filteredMembers = members.filter((m: TeamMember) => {
    const matchesSearch =
      m.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      m.email.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesRole = roleFilter === 'All Roles' || m.role === roleFilter;
    return matchesSearch && matchesRole;
  });

  return (
    <div className="space-y-6">
      {/* Header matching teams.png */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Identity & Access Management (IAM)</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">Teams & Collaboration</h1>
          <p className="text-xs text-slate-400">
            Manage your team, roles, permissions and collaborate on your decentralized infrastructure.
          </p>
        </div>

        <button
          onClick={() => setShowInviteModal(true)}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 transition-all"
        >
          <UserPlus className="w-4 h-4" />
          <span>Invite Member</span>
        </button>
      </div>

      {/* 4 Metric Cards matching teams.png */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          icon={<Users className="w-5 h-5 text-blue-400" />}
          label="Total Members"
          value={summary.totalMembers}
          change="+33%"
          trendColor="blue"
          capability="LIVE"
          provenance="iam/org-users"
        />
        <MetricCard
          icon={<CheckCircle2 className="w-5 h-5 text-emerald-400" />}
          label="Active Members"
          value={summary.activeMembers}
          change="+25%"
          trendColor="green"
          capability="LIVE"
          provenance="active-sessions"
        />
        <MetricCard
          icon={<Layers className="w-5 h-5 text-purple-400" />}
          label="Teams"
          value={summary.teamsCount}
          change="+100%"
          trendColor="purple"
          capability="LIVE"
          provenance="rbac/team-units"
        />
        <MetricCard
          icon={<Clock className="w-5 h-5 text-amber-400" />}
          label="Pending Invites"
          value={summary.pendingInvitesCount}
          subValue="Invitations waiting"
          trendColor="amber"
          capability="LIVE"
          provenance="signed-invites"
        />
      </div>

      {/* Main Grid: Members Table (8 cols) + Right Context Sidebar (4 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Table and Permissions (8 cols) */}
        <div className="lg:col-span-8 space-y-6">
          {/* Table Header Filter Bar */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 p-3 rounded-2xl bg-[#0D1527] border border-slate-800">
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold text-white px-2">Members</span>
            </div>

            <div className="flex items-center gap-2">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
                <Search className="w-4 h-4 text-slate-400" />
                <input
                  type="text"
                  placeholder="Search members..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="bg-transparent text-white focus:outline-none placeholder-slate-500 w-40 font-mono"
                />
              </div>

              <select
                value={roleFilter}
                onChange={(e) => setRoleFilter(e.target.value)}
                className="px-2.5 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs font-mono text-white focus:outline-none"
              >
                <option value="All Roles">All Roles</option>
                <option value="Owner">Owner</option>
                <option value="Admin">Admin</option>
                <option value="Developer">Developer</option>
                <option value="Viewer">Viewer</option>
              </select>
            </div>
          </div>

          {/* Members Table matching teams.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 overflow-hidden shadow-xl">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="border-b border-slate-800 bg-slate-950/60 text-slate-400 font-mono text-[11px] uppercase">
                  <th className="py-3.5 px-4 font-semibold">Member</th>
                  <th className="py-3.5 px-4 font-semibold">Role</th>
                  <th className="py-3.5 px-4 font-semibold">Teams</th>
                  <th className="py-3.5 px-4 font-semibold">Access Level</th>
                  <th className="py-3.5 px-4 font-semibold">Status</th>
                  <th className="py-3.5 px-4 font-semibold">Last Active</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono">
                {filteredMembers.map((m: TeamMember) => (
                  <tr key={m.id} className="hover:bg-slate-800/30 transition-colors">
                    {/* Member */}
                    <td className="py-4 px-4">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-blue-600 to-cyan-500 flex items-center justify-center font-bold text-white text-xs">
                          {m.avatarInitials}
                        </div>
                        <div>
                          <div className="font-bold text-white font-sans">{m.name}</div>
                          <div className="text-[11px] text-slate-400">{m.email}</div>
                        </div>
                      </div>
                    </td>

                    {/* Role */}
                    <td className="py-4 px-4">
                      <span
                        className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded text-[11px] font-semibold ${
                          m.role === 'Owner'
                            ? 'bg-amber-950 text-amber-400 border border-amber-500/40'
                            : m.role === 'Admin'
                            ? 'bg-cyan-950 text-cyan-400 border border-cyan-500/40'
                            : m.role === 'Developer'
                            ? 'bg-purple-950 text-purple-400 border border-purple-500/40'
                            : 'bg-slate-800 text-slate-300'
                        }`}
                      >
                        {m.role}
                      </span>
                    </td>

                    {/* Teams */}
                    <td className="py-4 px-4 text-slate-300">
                      {m.teams.join(', ')}
                    </td>

                    {/* Access Level */}
                    <td className="py-4 px-4 text-slate-400">{m.accessLevel}</td>

                    {/* Status */}
                    <td className="py-4 px-4">
                      <span className="flex items-center gap-1.5 text-emerald-400 font-medium">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                        {m.status}
                      </span>
                    </td>

                    {/* Last Active */}
                    <td className="py-4 px-4 text-slate-400">{m.lastActive}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Permissions Overview Toggles matching teams.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
            <h2 className="text-sm font-bold text-white">Permissions Overview</h2>
            <p className="text-xs text-slate-400">Control what your team members can access across the control plane.</p>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
              {[
                { key: 'deployApps', label: 'Deploy Apps', desc: 'Allow deploying and managing applications' },
                { key: 'manageDomains', label: 'Manage Domains', desc: 'Add, edit and transfer domains' },
                { key: 'manageStorage', label: 'Manage Storage', desc: 'Create and manage storage buckets' },
                { key: 'sslAndSecurity', label: 'SSL & Security', desc: 'Manage SSL certificates and security settings' },
                { key: 'viewAnalytics', label: 'View Analytics', desc: 'Access analytics and usage reports' }
              ].map((perm) => (
                <div key={perm.key} className="flex items-center justify-between p-3 rounded-xl bg-slate-900 border border-slate-800">
                  <div className="flex flex-col">
                    <span className="text-xs font-semibold text-white">{perm.label}</span>
                    <span className="text-[10px] text-slate-400">{perm.desc}</span>
                  </div>
                  <button
                    onClick={() => handleTogglePermission(perm.key as keyof TeamPermissionMatrix)}
                    className={`w-9 h-5 rounded-full transition-colors relative ${
                      permissions[perm.key as keyof TeamPermissionMatrix] ? 'bg-blue-600' : 'bg-slate-800'
                    }`}
                  >
                    <span
                      className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                        permissions[perm.key as keyof TeamPermissionMatrix] ? 'right-0.5' : 'left-0.5'
                      }`}
                    />
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Right: Invitation Link + Team Roles & Activity (4 cols) matching teams.png */}
        <div className="lg:col-span-4 space-y-5">
          {/* Share Invitation Link Card matching teams.png */}
          <div className="rounded-2xl bg-gradient-to-b from-[#121B30] to-[#0A1020] border border-blue-500/30 p-5 space-y-3 shadow-lg">
            <h2 className="text-sm font-bold text-white">Share Invitation Link</h2>
            <p className="text-xs text-slate-300">
              Let others join your organization with a secure signed link.
            </p>

            <div className="flex items-center gap-2 p-2 rounded-xl bg-slate-950 border border-slate-800 font-mono text-xs text-slate-300">
              <span className="truncate flex-1">{inviteLink}</span>
              <button
                onClick={handleCopy}
                className="p-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200"
                title="Copy link"
              >
                {copiedLink ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
          </div>

          {/* Team Roles Summary matching teams.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3 font-mono text-xs">
            <h3 className="font-bold text-white font-sans text-sm">Team Roles</h3>
            <div className="space-y-2">
              <div className="flex justify-between items-center p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-amber-400 font-bold">Owner</span>
                <span className="text-white font-bold">1</span>
              </div>
              <div className="flex justify-between items-center p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-cyan-400 font-bold">Admin</span>
                <span className="text-white font-bold">2</span>
              </div>
              <div className="flex justify-between items-center p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-purple-400 font-bold">Developer</span>
                <span className="text-white font-bold">5</span>
              </div>
              <div className="flex justify-between items-center p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-slate-400 font-bold">Viewer</span>
                <span className="text-white font-bold">4</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Invite Member Modal */}
      {showInviteModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4">
          <form
            onSubmit={handleInviteSubmit}
            className="w-full max-w-md rounded-2xl bg-[#0B1120] border border-slate-700 p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-white flex items-center gap-2">
                <UserPlus className="w-4 h-4 text-cyan-400" />
                Invite Team Member
              </h3>
              <button type="button" onClick={() => setShowInviteModal(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Email Address</label>
              <input
                type="email"
                required
                placeholder="colleague@decentralized.host"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Role</label>
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
              >
                <option value="Admin">Admin (Manage team and resources)</option>
                <option value="Developer">Developer (Deploy and manage resources)</option>
                <option value="Viewer">Viewer (Read-only access)</option>
              </select>
            </div>

            <div className="flex justify-end gap-2 pt-3 border-t border-slate-800">
              <button
                type="button"
                onClick={() => setShowInviteModal(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold shadow-lg"
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
