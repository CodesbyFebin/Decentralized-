import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

type UserRole = 'admin' | 'operator' | 'viewer' | 'guest'

interface TeamMember {
  id: string
  name: string
  email: string
  role: UserRole
  status: 'active' | 'inactive' | 'pending'
  lastActive: string
  joinedDate: string
  avatar?: string
}

interface Permission {
  id: string
  name: string
  description: string
}

interface RolePermissions {
  role: UserRole
  permissions: Permission[]
}

const Team: React.FC = () => {
  const [members, setMembers] = useState<TeamMember[]>([])
  const [rolePermissions, setRolePermissions] = useState<RolePermissions[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddMember, setShowAddMember] = useState(false)
  const [showPermissions, setShowPermissions] = useState(false)
  const [selectedRole, setSelectedRole] = useState<UserRole>('viewer')
  const [newMemberEmail, setNewMemberEmail] = useState('')
  const [newMemberRole, setNewMemberRole] = useState<UserRole>('viewer')

  const generateMockMembers = (): TeamMember[] => [
    {
      id: 'user-1',
      name: 'Alex Chen',
      email: 'alex@decentralized.host',
      role: 'admin',
      status: 'active',
      lastActive: '2026-09-30T09:45:00Z',
      joinedDate: '2026-01-15T00:00:00Z',
    },
    {
      id: 'user-2',
      name: 'Jordan Smith',
      email: 'jordan@decentralized.host',
      role: 'operator',
      status: 'active',
      lastActive: '2026-09-30T08:20:00Z',
      joinedDate: '2026-02-28T00:00:00Z',
    },
    {
      id: 'user-3',
      name: 'Morgan Lee',
      email: 'morgan@decentralized.host',
      role: 'operator',
      status: 'active',
      lastActive: '2026-09-29T16:30:00Z',
      joinedDate: '2026-03-10T00:00:00Z',
    },
    {
      id: 'user-4',
      name: 'Casey Williams',
      email: 'casey@decentralized.host',
      role: 'viewer',
      status: 'active',
      lastActive: '2026-09-30T07:15:00Z',
      joinedDate: '2026-05-22T00:00:00Z',
    },
    {
      id: 'user-5',
      name: 'Taylor Brown',
      email: 'taylor@decentralized.host',
      role: 'viewer',
      status: 'pending',
      lastActive: '',
      joinedDate: '2026-09-28T10:00:00Z',
    },
  ]

  const generateRolePermissions = (): RolePermissions[] => [
    {
      role: 'admin',
      permissions: [
        { id: 'p-1', name: 'Manage Users', description: 'Add, remove, and edit team members' },
        { id: 'p-2', name: 'Configure Security', description: 'Manage certificates and policies' },
        { id: 'p-3', name: 'Manage Deployments', description: 'Create and manage deployments' },
        { id: 'p-4', name: 'View Analytics', description: 'Access system analytics and metrics' },
        { id: 'p-5', name: 'Manage Nodes', description: 'Add, drain, and restart nodes' },
        { id: 'p-6', name: 'Manage Storage', description: 'Create and restore buckets' },
        { id: 'p-7', name: 'System Settings', description: 'Configure system-wide settings' },
      ],
    },
    {
      role: 'operator',
      permissions: [
        { id: 'p-2', name: 'Configure Security', description: 'Manage certificates and policies' },
        { id: 'p-3', name: 'Manage Deployments', description: 'Create and manage deployments' },
        { id: 'p-4', name: 'View Analytics', description: 'Access system analytics and metrics' },
        { id: 'p-5', name: 'Manage Nodes', description: 'Add, drain, and restart nodes' },
        { id: 'p-6', name: 'Manage Storage', description: 'Create and restore buckets' },
      ],
    },
    {
      role: 'viewer',
      permissions: [
        { id: 'p-4', name: 'View Analytics', description: 'Access system analytics and metrics' },
        { id: 'p-8', name: 'View Deployments', description: 'View deployment status and history' },
        { id: 'p-9', name: 'View Nodes', description: 'View node information and metrics' },
      ],
    },
    {
      role: 'guest',
      permissions: [
        { id: 'p-4', name: 'View Analytics', description: 'Access system analytics and metrics (read-only)' },
      ],
    },
  ]

  useEffect(() => {
    const loadTeam = async () => {
      try {
        setLoading(true)
        const mockMembers = generateMockMembers()
        const mockPermissions = generateRolePermissions()
        setMembers(mockMembers)
        setRolePermissions(mockPermissions)
        setError(null)
      } catch (err) {
        setError('Failed to load team data')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadTeam()
  }, [])

  if (error) {
    return (
      <AppLayout title="Team" subtitle="Team members and permissions">
        <ErrorState
          title="Failed to Load Team Data"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Team" subtitle="Team members and permissions">
        <Loading message="Loading team data..." />
      </AppLayout>
    )
  }

  const admins = members.filter((m) => m.role === 'admin').length
  const activeMembers = members.filter((m) => m.status === 'active').length
  const pendingInvites = members.filter((m) => m.status === 'pending').length

  const handleAddMember = async () => {
    if (!newMemberEmail.trim()) return
    try {
      await apiClient.post('/team/members', { email: newMemberEmail, role: newMemberRole })
      setNewMemberEmail('')
      setNewMemberRole('viewer')
      setShowAddMember(false)
      const mockMembers = generateMockMembers()
      setMembers(mockMembers)
    } catch (err) {
      console.error('Failed to add member:', err)
    }
  }

  const handleChangeRole = async (memberId: string, newRole: UserRole) => {
    try {
      await apiClient.patch(`/team/members/${memberId}`, { role: newRole })
      const mockMembers = generateMockMembers()
      setMembers(mockMembers)
    } catch (err) {
      console.error('Failed to change role:', err)
    }
  }

  const handleRemoveMember = async (memberId: string) => {
    try {
      await apiClient.delete(`/team/members/${memberId}`)
      const mockMembers = generateMockMembers()
      setMembers(mockMembers)
    } catch (err) {
      console.error('Failed to remove member:', err)
    }
  }

  const handleResendInvite = async (memberId: string) => {
    try {
      await apiClient.post(`/team/members/${memberId}/resend-invite`, {})
    } catch (err) {
      console.error('Failed to resend invite:', err)
    }
  }

  const getPermissionsForRole = (role: UserRole): Permission[] => {
    return rolePermissions.find((rp) => rp.role === role)?.permissions || []
  }

  return (
    <AppLayout
      title="Team"
      subtitle={`${activeMembers} active members • ${admins} admin${admins !== 1 ? 's' : ''} • ${pendingInvites} pending`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Members</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{members.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">{activeMembers} active</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Administrators</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">{admins}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">full access</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Operators</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-warning-500">
                  {members.filter((m) => m.role === 'operator').length}
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">deployment access</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Pending Invites</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">{pendingInvites}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">awaiting acceptance</p>
            </div>
          </Card>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Team Members */}
          <div className="lg:col-span-2">
            <Card variant="glass">
              <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
                <h3 className="text-lg font-semibold text-white">Team Members</h3>
                <Button
                  variant="primary"
                  size="md"
                  onClick={() => setShowAddMember(!showAddMember)}
                >
                  + Add Member
                </Button>
              </div>

              {showAddMember && (
                <div className="p-6 border-b border-neutral-700 bg-neutral-800/50 space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-neutral-300 mb-2">
                      Email Address
                    </label>
                    <input
                      type="email"
                      value={newMemberEmail}
                      onChange={(e) => setNewMemberEmail(e.target.value)}
                      placeholder="user@example.com"
                      className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-neutral-300 mb-2">
                      Role
                    </label>
                    <select
                      value={newMemberRole}
                      onChange={(e) => setNewMemberRole(e.target.value as UserRole)}
                      className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                    >
                      <option value="viewer">Viewer</option>
                      <option value="operator">Operator</option>
                      <option value="admin">Administrator</option>
                    </select>
                  </div>
                  <div className="flex gap-3">
                    <Button
                      variant="primary"
                      size="md"
                      onClick={handleAddMember}
                      disabled={!newMemberEmail.trim()}
                      className="flex-1"
                    >
                      Send Invite
                    </Button>
                    <Button
                      variant="ghost"
                      size="md"
                      onClick={() => setShowAddMember(false)}
                      className="flex-1"
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              )}

              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="border-b border-neutral-700">
                    <tr>
                      <th className="px-6 py-3 text-left text-neutral-400 font-medium">Name</th>
                      <th className="px-6 py-3 text-left text-neutral-400 font-medium">Email</th>
                      <th className="px-6 py-3 text-left text-neutral-400 font-medium">Role</th>
                      <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                      <th className="px-6 py-3 text-left text-neutral-400 font-medium">Last Active</th>
                      <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-700">
                    {members.map((member) => (
                      <tr key={member.id} className="hover:bg-neutral-800/30">
                        <td className="px-6 py-3 text-neutral-100 font-medium">{member.name}</td>
                        <td className="px-6 py-3 text-neutral-400 font-mono text-xs">{member.email}</td>
                        <td className="px-6 py-3">
                          <select
                            value={member.role}
                            onChange={(e) => handleChangeRole(member.id, e.target.value as UserRole)}
                            className="px-2 py-1 bg-neutral-700 border border-neutral-600 rounded text-xs text-white capitalize focus:border-primary-500 focus:outline-none"
                          >
                            <option value="guest">Guest</option>
                            <option value="viewer">Viewer</option>
                            <option value="operator">Operator</option>
                            <option value="admin">Admin</option>
                          </select>
                        </td>
                        <td className="px-6 py-3">
                          <Badge
                            status={member.status === 'active' ? 'active' : member.status === 'pending' ? 'pending' : 'error'}
                            size="sm"
                          />
                        </td>
                        <td className="px-6 py-3 text-neutral-400 text-xs">
                          {member.lastActive
                            ? new Date(member.lastActive).toLocaleString()
                            : member.status === 'pending'
                              ? 'Pending'
                              : 'Never'}
                        </td>
                        <td className="px-6 py-3 flex gap-2">
                          {member.status === 'pending' && (
                            <button
                              onClick={() => handleResendInvite(member.id)}
                              className="text-xs px-2 py-1 bg-primary-600 hover:bg-primary-500 rounded text-white"
                            >
                              Resend
                            </button>
                          )}
                          <button
                            onClick={() => handleRemoveMember(member.id)}
                            className="text-xs px-2 py-1 bg-error-600 hover:bg-error-500 rounded text-white"
                          >
                            Remove
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          </div>

          {/* Role-Based Access Control */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-white">RBAC</h3>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowPermissions(!showPermissions)}
              >
                {showPermissions ? 'Hide' : 'Show'}
              </Button>
            </div>

            {showPermissions && (
              <div className="divide-y divide-neutral-700">
                {rolePermissions.map((rp) => (
                  <div key={rp.role} className="p-6">
                    <div className="mb-4">
                      <h4 className="text-sm font-semibold text-neutral-100 capitalize mb-1">
                        {rp.role}
                      </h4>
                      <p className="text-xs text-neutral-400">
                        {rp.permissions.length} permission{rp.permissions.length !== 1 ? 's' : ''}
                      </p>
                    </div>
                    <div className="space-y-2">
                      {rp.permissions.slice(0, 3).map((perm) => (
                        <div key={perm.id} className="flex items-start gap-2">
                          <span className="text-primary-400 mt-1">✓</span>
                          <div>
                            <p className="text-xs font-medium text-neutral-300">{perm.name}</p>
                            <p className="text-xs text-neutral-500">{perm.description}</p>
                          </div>
                        </div>
                      ))}
                      {rp.permissions.length > 3 && (
                        <p className="text-xs text-neutral-400 mt-2">
                          +{rp.permissions.length - 3} more permission{rp.permissions.length - 3 !== 1 ? 's' : ''}
                        </p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}

            {!showPermissions && (
              <div className="p-6 space-y-2">
                <p className="text-sm text-neutral-400">Role permissions define what each team member can do:</p>
                <ul className="text-xs text-neutral-500 space-y-1 mt-3">
                  <li>• <span className="text-primary-400">Admin</span>: Full system access</li>
                  <li>• <span className="text-secondary-400">Operator</span>: Deploy and manage infrastructure</li>
                  <li>• <span className="text-success-400">Viewer</span>: Read-only analytics and dashboards</li>
                  <li>• <span className="text-warning-400">Guest</span>: Limited read-only access</li>
                </ul>
              </div>
            )}
          </Card>
        </div>
      </div>
    </AppLayout>
  )
}

export default Team
