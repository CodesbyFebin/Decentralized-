import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { StatDisplay } from '@/components/StatDisplay'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { apiClient } from '@/lib/api'
import { DashboardMetrics, Activity, Node } from '@/types'

const Dashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null)
  const [activities, setActivities] = useState<Activity[]>([])
  const [recentNodes, setRecentNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true)
        const [metricsRes, activitiesRes, nodesRes] = await Promise.all([
          apiClient.get<DashboardMetrics>('/dashboard/metrics'),
          apiClient.get<Activity[]>('/dashboard/activity?limit=10'),
          apiClient.get<Node[]>('/nodes?limit=5'),
        ])

        if (metricsRes.success && metricsRes.data) {
          setMetrics(metricsRes.data)
        }
        if (activitiesRes.success && activitiesRes.data) {
          setActivities(activitiesRes.data)
        }
        if (nodesRes.success && nodesRes.data) {
          setRecentNodes(nodesRes.data)
        }
        setError(null)
      } catch (err) {
        setError('Failed to load dashboard data')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadData()
    const interval = setInterval(loadData, 30000) // Refresh every 30s
    return () => clearInterval(interval)
  }, [])

  if (error) {
    return (
      <AppLayout title="Dashboard" subtitle="System overview">
        <div className="text-center py-12">
          <p className="text-error-500">{error}</p>
        </div>
      </AppLayout>
    )
  }

  return (
    <AppLayout title="Dashboard" subtitle="System overview and recent activity">
      {/* Metrics Grid */}
      <div className="grid grid-cols-4 gap-6 mb-8">
        <StatDisplay
          label="Total Nodes"
          value={metrics?.totalNodes || 0}
          icon="🖥️"
        />
        <StatDisplay
          label="Active Deployments"
          value={metrics?.activeDeployments || 0}
          icon="🚀"
        />
        <StatDisplay
          label="Storage Used"
          value={metrics?.storageUsed || 0}
          unit="GB"
          icon="💾"
        />
        <StatDisplay
          label="Success Rate"
          value={metrics?.successRate || 0}
          unit="%"
          trend={{ direction: 'up', percent: 2.5 }}
          icon="📈"
        />
      </div>

      <div className="grid grid-cols-3 gap-6">
        {/* System Resources */}
        <Card variant="glass" className="col-span-2">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-xl font-semibold text-white">System Resources</h3>
          </div>
          <div className="p-6">
            {loading ? (
              <div className="text-center py-8">
                <div className="animate-spin-slow">⏳</div>
              </div>
            ) : (
              <div className="space-y-4">
                <div>
                  <div className="flex justify-between mb-2">
                    <span className="text-neutral-400">CPU Usage</span>
                    <span className="text-primary-500 font-medium">
                      {metrics?.cpuAverage.toFixed(1)}%
                    </span>
                  </div>
                  <div className="w-full bg-neutral-700 rounded-full h-2">
                    <div
                      className="bg-gradient-to-r from-primary-500 to-secondary-500 h-2 rounded-full"
                      style={{ width: `${metrics?.cpuAverage || 0}%` }}
                    />
                  </div>
                </div>
                <div>
                  <div className="flex justify-between mb-2">
                    <span className="text-neutral-400">Memory Usage</span>
                    <span className="text-primary-500 font-medium">
                      {metrics?.memoryAverage.toFixed(1)}%
                    </span>
                  </div>
                  <div className="w-full bg-neutral-700 rounded-full h-2">
                    <div
                      className="bg-gradient-to-r from-secondary-500 to-warning-500 h-2 rounded-full"
                      style={{ width: `${metrics?.memoryAverage || 0}%` }}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* System Health */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-xl font-semibold text-white">System Health</h3>
          </div>
          <div className="p-6 text-center">
            <div className="w-20 h-20 mx-auto mb-4 rounded-full bg-gradient-to-br from-success-500/20 to-primary-500/20 flex items-center justify-center">
              <span className="text-4xl">✓</span>
            </div>
            <p className="text-success-500 font-semibold">All Systems Operational</p>
            <p className="text-neutral-400 text-sm mt-2">No alerts active</p>
          </div>
        </Card>
      </div>

      {/* Recent Nodes */}
      <Card variant="glass" className="mt-6">
        <div className="p-6 border-b border-neutral-700">
          <h3 className="text-xl font-semibold text-white">Recent Nodes</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-700">
                <th className="px-6 py-4 text-left text-sm font-semibold text-neutral-400">
                  Name
                </th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-neutral-400">
                  Status
                </th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-neutral-400">
                  CPU
                </th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-neutral-400">
                  Memory
                </th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-neutral-400">
                  Type
                </th>
              </tr>
            </thead>
            <tbody>
              {recentNodes.map((node) => (
                <tr key={node.id} className="border-b border-neutral-700 hover:bg-neutral-800/50">
                  <td className="px-6 py-4 text-neutral-100">{node.name}</td>
                  <td className="px-6 py-4">
                    <Badge status={node.status} />
                  </td>
                  <td className="px-6 py-4 text-neutral-300">{node.cpu.percent.toFixed(1)}%</td>
                  <td className="px-6 py-4 text-neutral-300">
                    {node.memory.percent.toFixed(1)}%
                  </td>
                  <td className="px-6 py-4 text-neutral-400">{node.nodeType}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Recent Activity */}
      <Card variant="glass" className="mt-6">
        <div className="p-6 border-b border-neutral-700">
          <h3 className="text-xl font-semibold text-white">Recent Activity</h3>
        </div>
        <div className="p-6">
          <div className="space-y-4">
            {activities.map((activity) => (
              <div
                key={activity.id}
                className="flex items-center justify-between p-4 bg-neutral-800/30 rounded-lg"
              >
                <div>
                  <p className="text-neutral-100 font-medium">{activity.action}</p>
                  <p className="text-neutral-400 text-sm">
                    {activity.actor} • {new Date(activity.timestamp).toLocaleString()}
                  </p>
                </div>
                <Badge
                  status={activity.status === 'success' ? 'active' : 'error'}
                  label={activity.status}
                />
              </div>
            ))}
          </div>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Dashboard
