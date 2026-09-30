import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { LineChart } from '@/components/Charts/LineChart'
import { AreaChart } from '@/components/Charts/AreaChart'
import { BarChart } from '@/components/Charts/BarChart'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface TimeSeriesData {
  timestamp: string
  requests?: number
  errors?: number
  errorRate?: number
  latency?: number
  p95?: number
  p99?: number
}

interface EndpointMetrics {
  id: string
  endpoint: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  requests: number
  avgLatency: number
  maxLatency: number
  errorCount: number
  errorRate: number
}

interface StatusCode {
  code: number
  count: number
  percentage: number
}

const Analytics: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timeSeriesData, setTimeSeriesData] = useState<TimeSeriesData[]>([])
  const [endpoints, setEndpoints] = useState<EndpointMetrics[]>([])
  const [statusCodes, setStatusCodes] = useState<StatusCode[]>([])

  const generateTimeSeriesData = (hours = 24): TimeSeriesData[] => {
    return Array.from({ length: hours }, (_, i) => {
      const baseRequests = 2000 + Math.random() * 3000
      const errorRate = 2 + Math.random() * 5
      return {
        timestamp: `${i}:00`,
        requests: Math.floor(baseRequests),
        errorRate: parseFloat((errorRate + (Math.random() * 2 - 1)).toFixed(1)),
        errors: Math.floor((baseRequests * errorRate) / 100),
        p95: Math.floor(80 + Math.random() * 150),
        p99: Math.floor(150 + Math.random() * 250),
      }
    })
  }

  const generateEndpointMetrics = (): EndpointMetrics[] => [
    {
      id: 'ep-1',
      endpoint: '/api/v1/nodes',
      method: 'GET',
      requests: 12450,
      avgLatency: 45,
      maxLatency: 320,
      errorCount: 12,
      errorRate: 0.1,
    },
    {
      id: 'ep-2',
      endpoint: '/api/v1/deployments',
      method: 'POST',
      requests: 3200,
      avgLatency: 120,
      maxLatency: 850,
      errorCount: 32,
      errorRate: 1.0,
    },
    {
      id: 'ep-3',
      endpoint: '/api/v1/storage/buckets',
      method: 'GET',
      requests: 8920,
      avgLatency: 65,
      maxLatency: 440,
      errorCount: 25,
      errorRate: 0.3,
    },
    {
      id: 'ep-4',
      endpoint: '/api/v1/health',
      method: 'GET',
      requests: 86400,
      avgLatency: 8,
      maxLatency: 50,
      errorCount: 0,
      errorRate: 0.0,
    },
    {
      id: 'ep-5',
      endpoint: '/api/v1/domains',
      method: 'GET',
      requests: 4560,
      avgLatency: 38,
      maxLatency: 280,
      errorCount: 18,
      errorRate: 0.4,
    },
  ]

  const generateStatusCodes = (): StatusCode[] => {
    const total = 115130
    return [
      { code: 200, count: 112015, percentage: 97.3 },
      { code: 201, count: 1520, percentage: 1.3 },
      { code: 400, count: 800, percentage: 0.7 },
      { code: 404, count: 500, percentage: 0.4 },
      { code: 500, count: 295, percentage: 0.3 },
    ]
  }

  useEffect(() => {
    const loadAnalytics = async () => {
      try {
        setLoading(true)
        const series = generateTimeSeriesData()
        const eps = generateEndpointMetrics()
        const codes = generateStatusCodes()
        setTimeSeriesData(series)
        setEndpoints(eps)
        setStatusCodes(codes)
        setError(null)
      } catch (err) {
        setError('Failed to load analytics data')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadAnalytics()
  }, [])

  if (error) {
    return (
      <AppLayout title="Analytics" subtitle="System metrics and analytics">
        <ErrorState
          title="Failed to Load Analytics"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Analytics" subtitle="System metrics and analytics">
        <Loading message="Loading analytics data..." />
      </AppLayout>
    )
  }

  const totalRequests = timeSeriesData.reduce((sum, d) => sum + (d.requests || 0), 0)
  const avgErrorRate = (timeSeriesData.reduce((sum, d) => sum + (d.errorRate || 0), 0) / timeSeriesData.length).toFixed(1)
  const avgLatency = Math.floor(
    timeSeriesData.reduce((sum, d) => sum + (d.p95 || 0), 0) / timeSeriesData.length
  )
  const uptime = 99.98

  const statusChartData = statusCodes.map((s) => ({
    name: `${s.code}`,
    count: s.count,
  }))

  return (
    <AppLayout
      title="Analytics"
      subtitle={`${(totalRequests / 1000).toFixed(1)}K requests • ${avgErrorRate}% error rate • ${uptime}% uptime`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Requests (24h)</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">
                  {(totalRequests / 1000).toFixed(1)}K
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">peak at 23:00</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Error Rate</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-error-500">{avgErrorRate}%</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">24h average</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">P95 Latency</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">{avgLatency}</span>
                <span className="text-xs text-neutral-400 mb-1">ms</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">95th percentile</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">System Uptime</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">{uptime}%</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">30-day average</p>
            </div>
          </Card>
        </div>

        {/* Traffic & Error Trends */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Request Traffic */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">Request Traffic (24h)</h3>
            </div>
            <div className="p-6">
              <LineChart
                data={timeSeriesData}
                dataKey="requests"
                name="Requests"
                stroke="#00D9FF"
                height={250}
                xAxisKey="timestamp"
              />
            </div>
          </Card>

          {/* Error Rate Trend */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">Error Rate Trend (24h)</h3>
            </div>
            <div className="p-6">
              <AreaChart
                data={timeSeriesData}
                dataKey="errorRate"
                name="Error %"
                fill="#F87171"
                stroke="#F87171"
                height={250}
                xAxisKey="timestamp"
              />
            </div>
          </Card>
        </div>

        {/* Latency Distribution */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* P95/P99 Latency */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">Latency Percentiles (24h)</h3>
            </div>
            <div className="p-6">
              <LineChart
                data={timeSeriesData}
                dataKey="p99"
                name="P99 Latency"
                stroke="#7C3AED"
                height={250}
                xAxisKey="timestamp"
              />
            </div>
          </Card>

          {/* HTTP Status Codes */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">Response Status Codes</h3>
            </div>
            <div className="p-6">
              <BarChart
                data={statusChartData}
                dataKeys={[{ key: 'count', name: 'Count', fill: '#00D9FF' }]}
                height={250}
                xAxisKey="name"
              />
            </div>
          </Card>
        </div>

        {/* Top Endpoints */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Top Endpoints</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Endpoint</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Method</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Requests</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Avg Latency</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Max Latency</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Errors</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Error Rate</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {endpoints.map((ep) => (
                  <tr key={ep.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-mono text-xs">{ep.endpoint}</td>
                    <td className="px-6 py-3 text-neutral-400">
                      <span className="px-2 py-1 bg-neutral-700 rounded text-xs font-medium">
                        {ep.method}
                      </span>
                    </td>
                    <td className="px-6 py-3 text-neutral-400 font-medium">
                      {ep.requests.toLocaleString()}
                    </td>
                    <td className="px-6 py-3 text-neutral-400">
                      <span className="text-primary-400">{ep.avgLatency}ms</span>
                    </td>
                    <td className="px-6 py-3 text-neutral-400">
                      <span className="text-warning-400">{ep.maxLatency}ms</span>
                    </td>
                    <td className="px-6 py-3 text-neutral-400">{ep.errorCount}</td>
                    <td className="px-6 py-3 text-neutral-400">
                      <span className={ep.errorRate > 0.5 ? 'text-error-400' : 'text-success-400'}>
                        {ep.errorRate.toFixed(2)}%
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Analytics
