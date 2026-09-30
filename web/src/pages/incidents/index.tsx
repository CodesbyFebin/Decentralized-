import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface Incident {
  id: string
  title: string
  severity: 'critical' | 'high' | 'medium' | 'low'
  status: 'open' | 'investigating' | 'monitoring' | 'resolved' | 'closed'
  description: string
  createdAt: string
  resolvedAt?: string
  commander: string
  impactedSystems: string[]
  affectedUsers?: number
  timeToDetect?: number
  timeToResolve?: number
}

interface IncidentStats {
  open: number
  investigating: number
  monitoring: number
  resolved: number
  mttr: number
  critical: number
}

const Incidents: React.FC = () => {
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [stats, setStats] = useState<IncidentStats>({
    open: 0,
    investigating: 0,
    monitoring: 0,
    resolved: 0,
    mttr: 0,
    critical: 0,
  })
  const [loading, setLoading] = useState(true)
  const [selectedStatus, setSelectedStatus] = useState<string>('all')
  const [selectedSeverity, setSelectedSeverity] = useState<string>('all')
  const [showNewForm, setShowNewForm] = useState(false)
  const [expandedIncident, setExpandedIncident] = useState<string | null>(null)
  const [newIncident, setNewIncident] = useState({
    title: '',
    severity: 'high' as const,
    description: '',
  })

  const generateMockIncidents = (): Incident[] => {
    const now = Date.now()
    return [
      {
        id: 'inc-1',
        title: 'Database Connection Pool Exhaustion',
        severity: 'critical',
        status: 'investigating',
        description: 'API endpoints experiencing timeout errors due to connection pool limits',
        createdAt: new Date(now - 1800000).toISOString(),
        commander: 'alice@example.com',
        impactedSystems: ['API', 'Database', 'Web Dashboard'],
        affectedUsers: 450,
        timeToDetect: 5,
      },
      {
        id: 'inc-2',
        title: 'Certificate Expiration Warning',
        severity: 'high',
        status: 'open',
        description: 'TLS certificate expiring in 7 days, manual renewal required',
        createdAt: new Date(now - 86400000).toISOString(),
        commander: 'bob@example.com',
        impactedSystems: ['Security', 'API Gateway'],
      },
      {
        id: 'inc-3',
        title: 'Disk Space Critical on Storage Node',
        severity: 'high',
        status: 'monitoring',
        description: 'Storage node store-02 has 2% disk free, cleanup in progress',
        createdAt: new Date(now - 3600000).toISOString(),
        resolvedAt: new Date(now - 600000).toISOString(),
        commander: 'charlie@example.com',
        impactedSystems: ['Storage', 'Backups'],
        timeToDetect: 8,
        timeToResolve: 50,
      },
      {
        id: 'inc-4',
        title: 'High Error Rate on /api/deployments',
        severity: 'medium',
        status: 'resolved',
        description: '5.2% error rate detected, root cause identified as rate limiter issue',
        createdAt: new Date(now - 7200000).toISOString(),
        resolvedAt: new Date(now - 3600000).toISOString(),
        commander: 'alice@example.com',
        impactedSystems: ['API', 'Deployments'],
        affectedUsers: 120,
        timeToDetect: 3,
        timeToResolve: 60,
      },
      {
        id: 'inc-5',
        title: 'Backup Job Failure',
        severity: 'medium',
        status: 'closed',
        description: 'Nightly backup failed due to network timeout, retried successfully',
        createdAt: new Date(now - 172800000).toISOString(),
        resolvedAt: new Date(now - 168000000).toISOString(),
        commander: 'bob@example.com',
        impactedSystems: ['Backups', 'Storage'],
        timeToDetect: 45,
        timeToResolve: 120,
      },
      {
        id: 'inc-6',
        title: 'Node Memory Leak Detected',
        severity: 'low',
        status: 'closed',
        description: 'Memory usage gradually increasing on compute node, restarted',
        createdAt: new Date(now - 259200000).toISOString(),
        resolvedAt: new Date(now - 255600000).toISOString(),
        commander: 'charlie@example.com',
        impactedSystems: ['Compute', 'Nodes'],
        timeToDetect: 720,
        timeToResolve: 30,
      },
    ]
  }

  useEffect(() => {
    const loadIncidents = async () => {
      try {
        setLoading(true)
        const mockIncidents = generateMockIncidents()
        setIncidents(mockIncidents)

        const mttrs = mockIncidents
          .filter((i) => i.timeToResolve)
          .map((i) => i.timeToResolve || 0)
        const avgMttr = mttrs.length > 0 ? Math.round(mttrs.reduce((a, b) => a + b) / mttrs.length) : 0

        const incidentStats: IncidentStats = {
          open: mockIncidents.filter((i) => i.status === 'open').length,
          investigating: mockIncidents.filter((i) => i.status === 'investigating').length,
          monitoring: mockIncidents.filter((i) => i.status === 'monitoring').length,
          resolved: mockIncidents.filter((i) => i.status === 'resolved').length,
          mttr: avgMttr,
          critical: mockIncidents.filter((i) => i.severity === 'critical').length,
        }
        setStats(incidentStats)
      } catch (err) {
        console.error('Failed to load incidents:', err)
      } finally {
        setLoading(false)
      }
    }

    loadIncidents()
  }, [])

  const handleCreateIncident = async () => {
    if (!newIncident.title.trim()) return

    try {
      await apiClient.post('/incidents', {
        title: newIncident.title,
        severity: newIncident.severity,
        description: newIncident.description,
      })

      const incident: Incident = {
        id: `inc-${Date.now()}`,
        title: newIncident.title,
        severity: newIncident.severity,
        status: 'open',
        description: newIncident.description,
        createdAt: new Date().toISOString(),
        commander: 'current-user@example.com',
        impactedSystems: [],
      }

      setIncidents((prev) => [incident, ...prev])
      setNewIncident({ title: '', severity: 'high', description: '' })
      setShowNewForm(false)
    } catch (err) {
      console.error('Failed to create incident:', err)
    }
  }

  const handleUpdateStatus = async (incidentId: string, newStatus: Incident['status']) => {
    try {
      await apiClient.patch(`/incidents/${incidentId}`, {
        status: newStatus,
      })

      setIncidents((prev) =>
        prev.map((i) =>
          i.id === incidentId
            ? {
                ...i,
                status: newStatus,
                resolvedAt:
                  newStatus === 'resolved' || newStatus === 'closed'
                    ? new Date().toISOString()
                    : i.resolvedAt,
              }
            : i
        )
      )
    } catch (err) {
      console.error('Failed to update incident:', err)
    }
  }

  if (loading) {
    return (
      <AppLayout title="Incidents" subtitle="Incident management and response tracking">
        <Loading message="Loading incidents..." />
      </AppLayout>
    )
  }

  const filteredIncidents = incidents.filter((i) => {
    const matchesStatus = selectedStatus === 'all' || i.status === selectedStatus
    const matchesSeverity = selectedSeverity === 'all' || i.severity === selectedSeverity
    return matchesStatus && matchesSeverity
  })

  const statuses = Array.from(new Set(incidents.map((i) => i.status)))
  const severities = Array.from(new Set(incidents.map((i) => i.severity)))

  return (
    <AppLayout
      title="Incidents"
      subtitle={`${stats.open + stats.investigating} active • ${stats.critical} critical • ${stats.mttr}min MTTR`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Open</p>
              <span className="text-3xl font-bold text-error-500">{stats.open}</span>
              <p className="text-xs text-neutral-500 mt-2">incidents</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Investigating</p>
              <span className="text-3xl font-bold text-warning-500">{stats.investigating}</span>
              <p className="text-xs text-neutral-500 mt-2">in progress</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Monitoring</p>
              <span className="text-3xl font-bold text-info-500">{stats.monitoring}</span>
              <p className="text-xs text-neutral-500 mt-2">follow-up</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Resolved</p>
              <span className="text-3xl font-bold text-success-500">{stats.resolved}</span>
              <p className="text-xs text-neutral-500 mt-2">this month</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Critical</p>
              <span className="text-3xl font-bold text-error-600">{stats.critical}</span>
              <p className="text-xs text-neutral-500 mt-2">severity</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Avg MTTR</p>
              <span className="text-3xl font-bold text-primary-500">{stats.mttr}</span>
              <p className="text-xs text-neutral-500 mt-2">minutes</p>
            </div>
          </Card>
        </div>

        {/* Incident Management */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-white">Incidents</h3>
              <Button
                variant="primary"
                size="sm"
                onClick={() => setShowNewForm(!showNewForm)}
              >
                + Report Incident
              </Button>
            </div>

            {showNewForm && (
              <div className="bg-neutral-800/30 rounded-lg p-4 space-y-3">
                <input
                  type="text"
                  value={newIncident.title}
                  onChange={(e) => setNewIncident({ ...newIncident, title: e.target.value })}
                  placeholder="Incident title"
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
                <textarea
                  value={newIncident.description}
                  onChange={(e) =>
                    setNewIncident({ ...newIncident, description: e.target.value })
                  }
                  placeholder="Description and context"
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none h-20"
                />
                <select
                  value={newIncident.severity}
                  onChange={(e) =>
                    setNewIncident({
                      ...newIncident,
                      severity: e.target.value as 'critical' | 'high' | 'medium' | 'low',
                    })
                  }
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
                <div className="flex gap-2 justify-end">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setShowNewForm(false)}
                  >
                    Cancel
                  </Button>
                  <Button variant="primary" size="sm" onClick={handleCreateIncident}>
                    Report
                  </Button>
                </div>
              </div>
            )}
          </div>

          {/* Filters */}
          <div className="p-6 border-b border-neutral-700 bg-neutral-800/20">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Status
                </label>
                <select
                  value={selectedStatus}
                  onChange={(e) => setSelectedStatus(e.target.value)}
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white text-sm focus:border-primary-500 focus:outline-none"
                >
                  <option value="all">All Status</option>
                  {statuses.map((status) => (
                    <option key={status} value={status}>
                      {status.charAt(0).toUpperCase() + status.slice(1)}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Severity
                </label>
                <select
                  value={selectedSeverity}
                  onChange={(e) => setSelectedSeverity(e.target.value)}
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white text-sm focus:border-primary-500 focus:outline-none"
                >
                  <option value="all">All Severities</option>
                  {severities.map((sev) => (
                    <option key={sev} value={sev}>
                      {sev.charAt(0).toUpperCase() + sev.slice(1)}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          {/* Incidents List */}
          <div className="divide-y divide-neutral-700">
            {filteredIncidents.length === 0 ? (
              <div className="p-6 text-center text-neutral-400">
                No incidents matching the selected filters
              </div>
            ) : (
              filteredIncidents.map((incident) => (
                <div
                  key={incident.id}
                  className="p-6 hover:bg-neutral-800/20 transition cursor-pointer"
                  onClick={() =>
                    setExpandedIncident(
                      expandedIncident === incident.id ? null : incident.id
                    )
                  }
                >
                  <div className="flex justify-between items-start gap-4 mb-2">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <h4 className="text-white font-medium">{incident.title}</h4>
                        <Badge
                          status={
                            incident.severity === 'critical'
                              ? 'error'
                              : incident.severity === 'high'
                                ? 'warning'
                                : incident.severity === 'medium'
                                  ? 'info'
                                  : 'default'
                          }
                        >
                          {incident.severity.charAt(0).toUpperCase() + incident.severity.slice(1)}
                        </Badge>
                      </div>
                      <p className="text-sm text-neutral-400 mb-2">{incident.description}</p>
                      <div className="flex flex-wrap gap-4 text-xs text-neutral-500">
                        <span>Created {new Date(incident.createdAt).toLocaleString()}</span>
                        <span>Commander: {incident.commander}</span>
                        {incident.affectedUsers && <span>Affected: {incident.affectedUsers} users</span>}
                      </div>
                    </div>
                    <Badge
                      status={
                        incident.status === 'open'
                          ? 'error'
                          : incident.status === 'investigating'
                            ? 'warning'
                            : incident.status === 'monitoring'
                              ? 'info'
                              : 'success'
                      }
                    >
                      {incident.status.charAt(0).toUpperCase() + incident.status.slice(1)}
                    </Badge>
                  </div>

                  {expandedIncident === incident.id && (
                    <div className="mt-4 p-4 bg-neutral-800/30 rounded-lg space-y-3 text-sm">
                      <div>
                        <p className="text-neutral-400 mb-2">Impacted Systems</p>
                        <div className="flex flex-wrap gap-2">
                          {incident.impactedSystems.map((sys) => (
                            <span key={sys} className="px-2 py-1 bg-neutral-700 rounded text-xs">
                              {sys}
                            </span>
                          ))}
                        </div>
                      </div>
                      {incident.timeToDetect && (
                        <div>
                          <p className="text-neutral-400">
                            Time to Detect: <span className="text-neutral-200">{incident.timeToDetect} min</span>
                          </p>
                        </div>
                      )}
                      {incident.timeToResolve && (
                        <div>
                          <p className="text-neutral-400">
                            Time to Resolve: <span className="text-neutral-200">{incident.timeToResolve} min</span>
                          </p>
                        </div>
                      )}
                      <div className="flex gap-2 pt-2">
                        <select
                          value={incident.status}
                          onChange={(e) =>
                            handleUpdateStatus(
                              incident.id,
                              e.target.value as Incident['status']
                            )
                          }
                          onClick={(e) => e.stopPropagation()}
                          className="px-3 py-1 bg-neutral-700 border border-neutral-600 rounded text-xs text-white focus:border-primary-500 focus:outline-none"
                        >
                          <option value="open">Open</option>
                          <option value="investigating">Investigating</option>
                          <option value="monitoring">Monitoring</option>
                          <option value="resolved">Resolved</option>
                          <option value="closed">Closed</option>
                        </select>
                      </div>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Incidents
