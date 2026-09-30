import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface AuditEntry {
  id: string
  timestamp: string
  actor: string
  action: string
  resource: string
  resourceId: string
  resourceType: 'node' | 'policy' | 'deployment' | 'certificate' | 'key' | 'user' | 'webhook'
  outcome: 'success' | 'failure'
  changes?: Record<string, { before: string; after: string }>
  ipAddress: string
  userAgent: string
}

interface AuditStats {
  totalEntries: number
  successCount: number
  failureCount: number
  uniqueActors: number
  uniqueResources: number
}

const Audit: React.FC = () => {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [stats, setStats] = useState<AuditStats>({
    totalEntries: 0,
    successCount: 0,
    failureCount: 0,
    uniqueActors: 0,
    uniqueResources: 0,
  })
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedAction, setSelectedAction] = useState<string>('all')
  const [selectedOutcome, setSelectedOutcome] = useState<'all' | 'success' | 'failure'>('all')
  const [selectedResourceType, setSelectedResourceType] = useState<string>('all')
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('24h')
  const [expandedEntry, setExpandedEntry] = useState<string | null>(null)

  const generateMockAuditEntries = (): AuditEntry[] => {
    const actions = [
      'create',
      'update',
      'delete',
      'enable',
      'disable',
      'start',
      'stop',
      'approve',
      'deny',
      'revoke',
    ]
    const resourceTypes: Array<'node' | 'policy' | 'deployment' | 'certificate' | 'key' | 'user' | 'webhook'> = [
      'node',
      'policy',
      'deployment',
      'certificate',
      'key',
      'user',
      'webhook',
    ]
    const actors = ['alice@example.com', 'bob@example.com', 'system@internal', 'automation@ci']
    const resources = [
      'prod-master-01',
      'policy-rbac-enforce',
      'deployment-v1.2.3',
      'cert-letsencrypt',
      'api-key-ci-001',
      'user-charlie',
      'webhook-slack',
    ]

    const now = Date.now()
    const entries: AuditEntry[] = []

    for (let i = 0; i < 50; i++) {
      const outcome = Math.random() > 0.05 ? 'success' : 'failure'
      entries.push({
        id: `audit-${i}`,
        timestamp: new Date(now - Math.random() * 86400000 * 30).toISOString(),
        actor: actors[Math.floor(Math.random() * actors.length)],
        action: actions[Math.floor(Math.random() * actions.length)],
        resource: resources[Math.floor(Math.random() * resources.length)],
        resourceId: `${Math.random().toString(36).substr(2, 12)}`,
        resourceType: resourceTypes[Math.floor(Math.random() * resourceTypes.length)],
        outcome: outcome as 'success' | 'failure',
        ipAddress: `192.168.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}`,
        userAgent: 'Mozilla/5.0 (CLI/dh-v1)',
      })
    }

    return entries.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
  }

  useEffect(() => {
    const loadAudit = async () => {
      try {
        setLoading(true)
        const mockEntries = generateMockAuditEntries()
        setEntries(mockEntries)

        const stats: AuditStats = {
          totalEntries: mockEntries.length,
          successCount: mockEntries.filter((e) => e.outcome === 'success').length,
          failureCount: mockEntries.filter((e) => e.outcome === 'failure').length,
          uniqueActors: new Set(mockEntries.map((e) => e.actor)).size,
          uniqueResources: new Set(mockEntries.map((e) => e.resource)).size,
        }
        setStats(stats)
      } catch (err) {
        console.error('Failed to load audit entries:', err)
      } finally {
        setLoading(false)
      }
    }

    loadAudit()
  }, [])

  const filteredEntries = entries.filter((entry) => {
    const matchesSearch =
      searchQuery === '' ||
      entry.actor.toLowerCase().includes(searchQuery.toLowerCase()) ||
      entry.resource.toLowerCase().includes(searchQuery.toLowerCase()) ||
      entry.action.toLowerCase().includes(searchQuery.toLowerCase())

    const matchesAction = selectedAction === 'all' || entry.action === selectedAction
    const matchesOutcome = selectedOutcome === 'all' || entry.outcome === selectedOutcome
    const matchesResourceType =
      selectedResourceType === 'all' || entry.resourceType === selectedResourceType

    return matchesSearch && matchesAction && matchesOutcome && matchesResourceType
  })

  const actions = Array.from(new Set(entries.map((e) => e.action))).sort()
  const resourceTypes = Array.from(new Set(entries.map((e) => e.resourceType))).sort()

  if (loading) {
    return (
      <AppLayout title="Audit Log" subtitle="Comprehensive system audit trail">
        <Loading message="Loading audit entries..." />
      </AppLayout>
    )
  }

  const failureRate = (
    ((stats.failureCount / stats.totalEntries) * 100).toFixed(1)
  )

  return (
    <AppLayout
      title="Audit Log"
      subtitle={`${stats.totalEntries} entries • ${failureRate}% failure rate • ${stats.uniqueActors} actors`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Events</p>
              <span className="text-3xl font-bold text-primary-500">{stats.totalEntries}</span>
              <p className="text-xs text-neutral-500 mt-2">audit entries</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Successful</p>
              <span className="text-3xl font-bold text-success-500">{stats.successCount}</span>
              <p className="text-xs text-neutral-500 mt-2">operations</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Failed</p>
              <span className="text-3xl font-bold text-error-500">{stats.failureCount}</span>
              <p className="text-xs text-neutral-500 mt-2">operations</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Actors</p>
              <span className="text-3xl font-bold text-info-500">{stats.uniqueActors}</span>
              <p className="text-xs text-neutral-500 mt-2">unique users</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Resources</p>
              <span className="text-3xl font-bold text-warning-500">{stats.uniqueResources}</span>
              <p className="text-xs text-neutral-500 mt-2">affected</p>
            </div>
          </Card>
        </div>

        {/* Audit Log Viewer */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Audit Trail</h3>

            <div className="space-y-4">
              {/* Search */}
              <div>
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search by actor, resource, or action..."
                  className="w-full px-4 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>

              {/* Filters */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Action
                  </label>
                  <select
                    value={selectedAction}
                    onChange={(e) => setSelectedAction(e.target.value)}
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="all">All Actions</option>
                    {actions.map((action) => (
                      <option key={action} value={action}>
                        {action.charAt(0).toUpperCase() + action.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Outcome
                  </label>
                  <select
                    value={selectedOutcome}
                    onChange={(e) => setSelectedOutcome(e.target.value as typeof selectedOutcome)}
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="all">All Outcomes</option>
                    <option value="success">Success</option>
                    <option value="failure">Failure</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Resource Type
                  </label>
                  <select
                    value={selectedResourceType}
                    onChange={(e) => setSelectedResourceType(e.target.value)}
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="all">All Types</option>
                    {resourceTypes.map((type) => (
                      <option key={type} value={type}>
                        {type.charAt(0).toUpperCase() + type.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Time Range
                  </label>
                  <select
                    value={timeRange}
                    onChange={(e) => setTimeRange(e.target.value as typeof timeRange)}
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="1h">Last 1 hour</option>
                    <option value="24h">Last 24 hours</option>
                    <option value="7d">Last 7 days</option>
                    <option value="30d">Last 30 days</option>
                  </select>
                </div>
              </div>
            </div>
          </div>

          {/* Audit Table */}
          <div className="overflow-x-auto max-h-[700px] overflow-y-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700 sticky top-0 bg-neutral-900">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Timestamp</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actor</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Action</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Resource</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Outcome</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">IP Address</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {filteredEntries.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-4 text-center text-neutral-500">
                      No audit entries found matching the selected filters
                    </td>
                  </tr>
                ) : (
                  filteredEntries.map((entry) => (
                    <React.Fragment key={entry.id}>
                      <tr
                        className="hover:bg-neutral-800/30 transition cursor-pointer"
                        onClick={() =>
                          setExpandedEntry(expandedEntry === entry.id ? null : entry.id)
                        }
                      >
                        <td className="px-6 py-3 text-neutral-500 whitespace-nowrap text-xs">
                          {new Date(entry.timestamp).toLocaleString()}
                        </td>
                        <td className="px-6 py-3 text-neutral-100 text-xs font-medium">
                          {entry.actor}
                        </td>
                        <td className="px-6 py-3 text-neutral-300 text-xs">
                          {entry.action.charAt(0).toUpperCase() + entry.action.slice(1)}
                        </td>
                        <td className="px-6 py-3 text-neutral-400 text-xs">
                          <div>
                            <p className="font-mono">{entry.resource}</p>
                            <p className="text-neutral-500">
                              {entry.resourceType.charAt(0).toUpperCase() +
                                entry.resourceType.slice(1)}
                            </p>
                          </div>
                        </td>
                        <td className="px-6 py-3">
                          <Badge status={entry.outcome === 'success' ? 'success' : 'error'}>
                            {entry.outcome === 'success' ? 'Success' : 'Failed'}
                          </Badge>
                        </td>
                        <td className="px-6 py-3 text-neutral-500 text-xs font-mono">
                          {entry.ipAddress}
                        </td>
                      </tr>
                      {expandedEntry === entry.id && (
                        <tr className="bg-neutral-800/20 border-b border-neutral-700">
                          <td colSpan={6} className="px-6 py-4">
                            <div className="space-y-3">
                              <div>
                                <p className="text-xs font-medium text-neutral-400 mb-1">
                                  Resource ID
                                </p>
                                <p className="text-sm text-neutral-300 font-mono">
                                  {entry.resourceId}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs font-medium text-neutral-400 mb-1">
                                  User Agent
                                </p>
                                <p className="text-sm text-neutral-400 font-mono">
                                  {entry.userAgent}
                                </p>
                              </div>
                              {entry.changes && (
                                <div>
                                  <p className="text-xs font-medium text-neutral-400 mb-2">
                                    Changes
                                  </p>
                                  <div className="space-y-1 text-xs">
                                    {Object.entries(entry.changes).map(([key, value]) => (
                                      <div key={key} className="flex gap-2">
                                        <span className="text-neutral-500 min-w-24">
                                          {key}:
                                        </span>
                                        <span className="text-error-400">
                                          {value.before}
                                        </span>
                                        <span className="text-neutral-500">→</span>
                                        <span className="text-success-400">
                                          {value.after}
                                        </span>
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <div className="p-6 border-t border-neutral-700 text-center text-neutral-400 text-sm">
            Showing {filteredEntries.length} of {stats.totalEntries} audit entries
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Audit
