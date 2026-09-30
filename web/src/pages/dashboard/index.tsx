import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { StatDisplay } from '@/components/StatDisplay'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { AreaChart } from '@/components/Charts/AreaChart'
import { LineChart } from '@/components/Charts/LineChart'
import { apiClient } from '@/lib/api'
import { withAuth } from '@/lib/withAuth'
import { DashboardMetrics, Activity, Node } from '@/types'

const Dashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null)
  const [activities, setActivities] = useState<Activity[]>([])
  const [recentNodes, setRecentNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Generate mock time-series data for charts
  const generateTimeSeriesData = (hours = 24) => {
    return Array.from({ length: hours }, (_, i) => ({
      timestamp: `${i}:00`,
      cpu: Math.random() * 100,
      memory: Math.random() * 100,
      requests: Math.floor(Math.random() * 10000),
      latency: Math.random() * 500,
    }))
  }

  const timeSeriesData = generateTimeSeriesData()

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
        <ErrorState
          title="Failed to Load Dashboard"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Dashboard" subtitle="System overview and recent activity">
        <Loading message="Loading dashboard data..." />
      </AppLayout>
    )
  }

  return (
    <AppLayout title="Dashboard" subtitle="System overview and real-time metrics">
      {/* Key Metrics Grid - Responsive */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
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

      {/* Charts Section - Responsive */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* CPU Usage Over Time */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">CPU Usage (24h)</h3>
          </div>
          <div className="p-6">
            <AreaChart
              data={timeSeriesData}
              dataKey="cpu"
              name="CPU %"
              height={250}
              xAxisKey="timestamp"
            />
          </div>
        </Card>

        {/* Memory Usage Over Time */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Memory Usage (24h)</h3>
          </div>
          <div className="p-6">
            <AreaChart
              data={timeSeriesData}
              dataKey="memory"
              name="Memory %"
              fill="url(#colorGradient2)"
              stroke="#7c3aed"
              height={250}
              xAxisKey="timestamp"
            />
          </div>
        </Card>
      </div>

      {/* System Stats - Responsive */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
        {/* Request Rate */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Request Rate (24h)</h3>
          </div>
          <div className="p-6">
            <LineChart
              data={timeSeriesData}
              dataKey="requests"
              name="Requests"
              height={200}
              xAxisKey="timestamp"
            />
          </div>
        </Card>

        {/* Latency */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Latency (24h)</h3>
          </div>
          <div className="p-6">
            <LineChart
              data={timeSeriesData}
              dataKey="latency"
              name="Latency (ms)"
              stroke="#f59e0b"
              height={200}
              xAxisKey="timestamp"
            />
          </div>
        </Card>

        {/* System Health */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-xl font-semibold text-white">System Health</h3>
          </div>
          <div className="p-6 flex flex-col items-center justify-center min-h-[200px]">
            <div className="w-20 h-20 rounded-full bg-gradient-to-br from-success-500/20 to-primary-500/20 flex items-center justify-center mb-4">
              <span className="text-4xl">✓</span>
            </div>
            <p className="text-success-500 font-semibold">All Systems Operational</p>
            <p className="text-neutral-400 text-sm mt-2">No alerts active</p>
          </div>
        </Card>
      </div>

      {/* Recent Nodes */}
      <Card variant="glass" className="mb-8">
        <div className="p-6 border-b border-neutral-700 flex justify-between items-center">
          <h3 className="text-xl font-semibold text-white">Active Nodes</h3>
          <Badge status="active" label={`${recentNodes.length} Active`} />
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-neutral-700 bg-neutral-800/30">
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
                  Disk
                </th>
                <th className="px-6 py-4 text-left text-sm font-semibold text-neutral-400">
                  Type
                </th>
              </tr>
            </thead>
            <tbody>
              {recentNodes.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-neutral-400">
                    No nodes available
                  </td>
                </tr>
              ) : (
                recentNodes.map((node) => (
                  <tr key={node.id} className="border-b border-neutral-700 hover:bg-neutral-800/50 transition-colors">
                    <td className="px-6 py-4 text-neutral-100 font-medium">{node.name}</td>
                    <td className="px-6 py-4">
                      <Badge status={node.status} size="sm" />
                    </td>
                    <td className="px-6 py-4 text-neutral-300">{node.cpu.percent.toFixed(1)}%</td>
                    <td className="px-6 py-4 text-neutral-300">
                      {node.memory.percent.toFixed(1)}%
                    </td>
                    <td className="px-6 py-4 text-neutral-300">{node.disk.percent.toFixed(1)}%</td>
                    <td className="px-6 py-4 text-neutral-400 text-sm">{node.nodeType}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Recent Activity - Timeline */}
      <Card variant="glass">
        <div className="p-6 border-b border-neutral-700">
          <h3 className="text-xl font-semibold text-white">Recent Activity</h3>
        </div>
        <div className="p-6">
          {activities.length === 0 ? (
            <div className="text-center py-8 text-neutral-400">No recent activity</div>
          ) : (
            <div className="space-y-4">
              {activities.map((activity, index) => (
                <div
                  key={activity.id}
                  className="flex items-start gap-4 p-4 bg-neutral-800/30 rounded-lg hover:bg-neutral-800/50 transition-colors"
                >
                  {/* Timeline dot */}
                  <div className="flex flex-col items-center mt-1">
                    <div
                      className={`w-3 h-3 rounded-full ${
                        activity.status === 'success' ? 'bg-success-500' : 'bg-error-500'
                      }`}
                    />
                    {index < activities.length - 1 && (
                      <div className="w-0.5 h-8 bg-neutral-700 my-2" />
                    )}
                  </div>

                  {/* Activity content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-4 mb-1">
                      <p className="text-neutral-100 font-medium truncate">{activity.action}</p>
                      <Badge
                        status={activity.status === 'success' ? 'active' : 'error'}
                        size="sm"
                      />
                    </div>
                    <p className="text-neutral-400 text-sm">
                      <span className="font-medium">{activity.actor}</span>
                      {' • '}
                      {new Date(activity.timestamp).toLocaleString()}
                    </p>
                    {activity.details && (
                      <p className="text-neutral-500 text-xs mt-2">{activity.details}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </Card>
    </AppLayout>
  )
}

export default withAuth(Dashboard)
