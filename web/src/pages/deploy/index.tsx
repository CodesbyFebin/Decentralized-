import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { LineChart } from '@/components/Charts/LineChart'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface Deployment {
  id: string
  name: string
  version: number
  status: 'pending' | 'admitted' | 'executing' | 'observed' | 'verified'
  progress: number
  targetNodes: number
  deployedNodes: number
  createdAt: string
  completedAt?: string
  signedBy: string
  rolloutStatus: 'not_started' | 'in_progress' | 'completed' | 'failed'
  rolloutProgress: number
}

type DeploymentStatus = 'all' | 'pending' | 'admitted' | 'executing' | 'observed' | 'verified' | 'failed'

const Deploy: React.FC = () => {
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<DeploymentStatus>('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedDeployment, setSelectedDeployment] = useState<Deployment | null>(null)
  const [showDeploymentForm, setShowDeploymentForm] = useState(false)
  const [formStep, setFormStep] = useState<'details' | 'targets' | 'review'>('details')
  const [formData, setFormData] = useState({
    name: '',
    version: '1.0.0',
    targetNodes: 3,
  })

  const generateMockDeployments = (): Deployment[] => [
    {
      id: 'deploy-1',
      name: 'api-service-v2.1.0',
      version: 2,
      status: 'verified',
      progress: 100,
      targetNodes: 5,
      deployedNodes: 5,
      createdAt: '2026-09-28T10:30:00Z',
      completedAt: '2026-09-28T10:45:00Z',
      signedBy: 'ci-pipeline',
      rolloutStatus: 'completed',
      rolloutProgress: 100,
    },
    {
      id: 'deploy-2',
      name: 'web-ui-v3.0.0',
      version: 3,
      status: 'executing',
      progress: 75,
      targetNodes: 3,
      deployedNodes: 2,
      createdAt: '2026-09-30T08:15:00Z',
      signedBy: 'jenkins-bot',
      rolloutStatus: 'in_progress',
      rolloutProgress: 67,
    },
    {
      id: 'deploy-3',
      name: 'worker-nodes-v1.5.2',
      version: 1,
      status: 'observed',
      progress: 90,
      targetNodes: 8,
      deployedNodes: 8,
      createdAt: '2026-09-29T14:20:00Z',
      completedAt: '2026-09-29T14:35:00Z',
      signedBy: 'cd-system',
      rolloutStatus: 'completed',
      rolloutProgress: 100,
    },
    {
      id: 'deploy-4',
      name: 'database-migration-v2.0.1',
      version: 2,
      status: 'admitted',
      progress: 25,
      targetNodes: 1,
      deployedNodes: 0,
      createdAt: '2026-09-30T09:00:00Z',
      signedBy: 'admin-user',
      rolloutStatus: 'not_started',
      rolloutProgress: 0,
    },
    {
      id: 'deploy-5',
      name: 'cache-service-v1.8.0',
      version: 1,
      status: 'verified',
      progress: 100,
      targetNodes: 4,
      deployedNodes: 4,
      createdAt: '2026-09-27T16:45:00Z',
      completedAt: '2026-09-27T17:00:00Z',
      signedBy: 'automation',
      rolloutStatus: 'completed',
      rolloutProgress: 100,
    },
  ]

  const generateRolloutTimeline = () => {
    return Array.from({ length: 12 }, (_, i) => ({
      timestamp: `${i}:00`,
      deployed: Math.floor((i / 12) * 100),
      healthy: Math.max(0, Math.floor((i / 12) * 100) - 10),
    }))
  }

  useEffect(() => {
    const loadDeployments = async () => {
      try {
        setLoading(true)
        const mockDeployments = generateMockDeployments()
        setDeployments(mockDeployments)
        if (mockDeployments.length > 0) {
          setSelectedDeployment(mockDeployments[0])
        }
        setError(null)
      } catch (err) {
        setError('Failed to load deployments')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadDeployments()
  }, [])

  if (error) {
    return (
      <AppLayout title="Deployments" subtitle="Application deployment management">
        <ErrorState
          title="Failed to Load Deployments"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Deployments" subtitle="Application deployment management">
        <Loading message="Loading deployments..." />
      </AppLayout>
    )
  }

  const filteredDeployments = deployments.filter((d) => {
    const statusMatch = statusFilter === 'all' || d.status === statusFilter
    const searchMatch =
      searchTerm === '' ||
      d.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      d.id.toLowerCase().includes(searchTerm.toLowerCase())
    return statusMatch && searchMatch
  })

  const rolloutTimeline = selectedDeployment ? generateRolloutTimeline() : []

  const handleDeploy = async () => {
    try {
      await apiClient.post('/deployments', formData)
      setShowDeploymentForm(false)
      setFormStep('details')
      setFormData({ name: '', version: '1.0.0', targetNodes: 3 })
      const mockDeployments = generateMockDeployments()
      setDeployments(mockDeployments)
    } catch (err) {
      console.error('Failed to create deployment:', err)
    }
  }

  const handleRollback = async (deploymentId: string) => {
    try {
      await apiClient.post(`/deployments/${deploymentId}/rollback`, {})
      const mockDeployments = generateMockDeployments()
      setDeployments(mockDeployments)
    } catch (err) {
      console.error('Failed to rollback deployment:', err)
    }
  }

  const verifiedCount = deployments.filter((d) => d.status === 'verified').length
  const inProgressCount = deployments.filter((d) =>
    ['pending', 'admitted', 'executing', 'observed'].includes(d.status)
  ).length

  return (
    <AppLayout
      title="Deployments"
      subtitle={`${verifiedCount} verified • ${inProgressCount} in progress`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Deployments</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{deployments.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">{verifiedCount} verified</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">In Progress</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">{inProgressCount}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">Active deployments</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Success Rate</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">96%</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">Last 30 days</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Avg. Rollout Time</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-warning-500">4.2</span>
                <span className="text-xs text-neutral-400 mb-1">min</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">Zero-downtime deploys</p>
            </div>
          </Card>
        </div>

        {/* Controls */}
        <div className="flex flex-col gap-3">
          <div className="flex gap-3">
            <Button variant="primary" size="md" onClick={() => setShowDeploymentForm(true)}>
              + New Deployment
            </Button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input
              type="text"
              placeholder="Search deployments..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as DeploymentStatus)}
              className="px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white focus:border-primary-500 focus:outline-none"
            >
              <option value="all">All Status</option>
              <option value="pending">Pending</option>
              <option value="admitted">Admitted</option>
              <option value="executing">Executing</option>
              <option value="observed">Observed</option>
              <option value="verified">Verified</option>
            </select>
          </div>
        </div>

        {/* Deployment Form Modal */}
        {showDeploymentForm && (
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
            <Card variant="glass" className="w-full max-w-md">
              <div className="p-6 border-b border-neutral-700">
                <h2 className="text-xl font-bold text-white">New Deployment</h2>
                <div className="flex gap-1 mt-4">
                  {(['details', 'targets', 'review'] as const).map((s, i) => (
                    <div
                      key={s}
                      className={`h-1 flex-1 rounded-full ${
                        (formStep === 'details' && i <= 0) ||
                        (formStep === 'targets' && i <= 1) ||
                        (formStep === 'review' && i <= 2)
                          ? 'bg-primary-500'
                          : 'bg-neutral-700'
                      }`}
                    />
                  ))}
                </div>
              </div>

              <div className="p-6 space-y-4 min-h-[280px]">
                {formStep === 'details' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-neutral-300 mb-2">
                        Deployment Name
                      </label>
                      <input
                        type="text"
                        value={formData.name}
                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        placeholder="e.g., api-service-v2.0.0"
                        className="w-full px-3 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-neutral-300 mb-2">
                        Version
                      </label>
                      <input
                        type="text"
                        value={formData.version}
                        onChange={(e) => setFormData({ ...formData, version: e.target.value })}
                        placeholder="e.g., 2.0.0"
                        className="w-full px-3 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                      />
                    </div>
                  </>
                )}

                {formStep === 'targets' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-neutral-300 mb-2">
                        Target Nodes: {formData.targetNodes}
                      </label>
                      <input
                        type="range"
                        min="1"
                        max="10"
                        value={formData.targetNodes}
                        onChange={(e) =>
                          setFormData({ ...formData, targetNodes: parseInt(e.target.value) })
                        }
                        className="w-full"
                      />
                    </div>
                    <p className="text-xs text-neutral-400 mt-2">
                      Deployment will roll out to {formData.targetNodes} target node(s) with
                      zero-downtime blue-green strategy.
                    </p>
                  </>
                )}

                {formStep === 'review' && (
                  <div className="space-y-3">
                    <div className="bg-neutral-800/50 rounded-lg p-4 space-y-2">
                      <p className="text-sm">
                        <span className="text-neutral-400">Name:</span>{' '}
                        <span className="text-white font-mono">{formData.name}</span>
                      </p>
                      <p className="text-sm">
                        <span className="text-neutral-400">Version:</span>{' '}
                        <span className="text-white font-mono">{formData.version}</span>
                      </p>
                      <p className="text-sm">
                        <span className="text-neutral-400">Target Nodes:</span>{' '}
                        <span className="text-white font-mono">{formData.targetNodes}</span>
                      </p>
                      <p className="text-sm">
                        <span className="text-neutral-400">Strategy:</span>{' '}
                        <span className="text-white font-mono">Blue-Green (Zero-Downtime)</span>
                      </p>
                    </div>
                  </div>
                )}
              </div>

              <div className="p-6 border-t border-neutral-700 flex gap-3">
                <Button
                  variant="ghost"
                  size="md"
                  onClick={() => {
                    setShowDeploymentForm(false)
                    setFormStep('details')
                  }}
                  className="flex-1"
                >
                  Cancel
                </Button>
                {formStep !== 'details' && (
                  <Button
                    variant="secondary"
                    size="md"
                    onClick={() => {
                      if (formStep === 'targets') setFormStep('details')
                      if (formStep === 'review') setFormStep('targets')
                    }}
                    className="flex-1"
                  >
                    Back
                  </Button>
                )}
                {formStep !== 'review' ? (
                  <Button
                    variant="primary"
                    size="md"
                    onClick={() => {
                      if (formStep === 'details') setFormStep('targets')
                      if (formStep === 'targets') setFormStep('review')
                    }}
                    disabled={formStep === 'details' && !formData.name.trim()}
                    className="flex-1"
                  >
                    Next
                  </Button>
                ) : (
                  <Button
                    variant="primary"
                    size="md"
                    onClick={handleDeploy}
                    className="flex-1"
                  >
                    Deploy
                  </Button>
                )}
              </div>
            </Card>
          </div>
        )}

        {/* Deployments List */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Deployments ({filteredDeployments.length})</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Name</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Progress</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Nodes</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Created</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {filteredDeployments.map((deployment) => (
                  <tr
                    key={deployment.id}
                    onClick={() => setSelectedDeployment(deployment)}
                    className={`cursor-pointer ${
                      selectedDeployment?.id === deployment.id
                        ? 'bg-primary-500/10'
                        : 'hover:bg-neutral-800/30'
                    }`}
                  >
                    <td className="px-6 py-3 text-neutral-100 font-medium">{deployment.name}</td>
                    <td className="px-6 py-3">
                      <Badge
                        status={
                          deployment.status === 'verified'
                            ? 'active'
                            : deployment.status === 'executing'
                              ? 'pending'
                              : deployment.status === 'admitted'
                                ? 'pending'
                                : 'inactive'
                        }
                        size="sm"
                      />
                    </td>
                    <td className="px-6 py-3 text-neutral-400">
                      <div className="w-24 h-2 bg-neutral-700 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary-500"
                          style={{ width: `${deployment.progress}%` }}
                        />
                      </div>
                      <span className="text-xs text-neutral-500">{deployment.progress}%</span>
                    </td>
                    <td className="px-6 py-3 text-neutral-400">
                      {deployment.deployedNodes}/{deployment.targetNodes}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                      {new Date(deployment.createdAt).toLocaleString()}
                    </td>
                    <td className="px-6 py-3 flex gap-2">
                      {deployment.status === 'verified' && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            handleRollback(deployment.id)
                          }}
                          className="text-xs px-2 py-1 bg-warning-600/20 hover:bg-warning-600/40 rounded text-warning-400 border border-warning-600/30"
                        >
                          Rollback
                        </button>
                      )}
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        Details
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Rollout Tracking */}
        {selectedDeployment && (
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">
                Rollout Progress: {selectedDeployment.name}
              </h3>
            </div>
            <div className="p-6 space-y-6">
              {/* Overall Progress */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <p className="text-sm text-neutral-400">Rollout Progress</p>
                  <p className="text-sm font-bold text-primary-400">
                    {selectedDeployment.rolloutProgress}%
                  </p>
                </div>
                <div className="w-full h-3 bg-neutral-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-primary-500 to-primary-400"
                    style={{ width: `${selectedDeployment.rolloutProgress}%` }}
                  />
                </div>
              </div>

              {/* Timeline Chart */}
              <div>
                <p className="text-sm text-neutral-400 mb-3">Deployment Timeline (12h)</p>
                <LineChart
                  data={rolloutTimeline}
                  dataKey="deployed"
                  name="Deployed %"
                  stroke="#00D9FF"
                  height={200}
                  xAxisKey="timestamp"
                />
              </div>

              {/* Deployment State Machine */}
              <div className="space-y-2">
                <p className="text-sm text-neutral-400 mb-3">Deployment State</p>
                <div className="flex items-center justify-between text-xs">
                  {(['DESIRED', 'ADMITTED', 'EXECUTING', 'OBSERVED', 'VERIFIED'] as const).map(
                    (state, i) => {
                      const stateValues = {
                        DESIRED: 'pending',
                        ADMITTED: 'admitted',
                        EXECUTING: 'executing',
                        OBSERVED: 'observed',
                        VERIFIED: 'verified',
                      }
                      const isCompleted =
                        (state === 'DESIRED') ||
                        (state === 'ADMITTED' &&
                          ['admitted', 'executing', 'observed', 'verified'].includes(
                            selectedDeployment.status
                          )) ||
                        (state === 'EXECUTING' &&
                          ['executing', 'observed', 'verified'].includes(selectedDeployment.status)) ||
                        (state === 'OBSERVED' &&
                          ['observed', 'verified'].includes(selectedDeployment.status)) ||
                        (state === 'VERIFIED' && selectedDeployment.status === 'verified')

                      return (
                        <div key={state} className="flex-1 text-center">
                          <div
                            className={`w-3 h-3 rounded-full mx-auto mb-1 ${
                              isCompleted ? 'bg-success-500' : 'bg-neutral-700'
                            }`}
                          />
                          <p className="text-neutral-400">{state}</p>
                        </div>
                      )
                    }
                  )}
                </div>
              </div>
            </div>
          </Card>
        )}
      </div>
    </AppLayout>
  )
}

export default Deploy
