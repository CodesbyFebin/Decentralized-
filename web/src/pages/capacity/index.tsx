import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface CapacityMetric {
  name: string
  current: number
  projected30: number
  projected90: number
  threshold: number
  unit: string
  status: 'healthy' | 'warning' | 'critical'
}

interface ForecastTrend {
  metric: string
  historicalUsage: Array<{ date: string; usage: number }>
  projectedUsage: Array<{ date: string; usage: number }>
  growthRate: number
  exhaustionDate?: string
}

interface CapacityAlert {
  id: string
  metric: string
  severity: 'info' | 'warning' | 'critical'
  message: string
  timestamp: string
  recommendation: string
}

interface CapacityStats {
  totalResources: number
  criticalItems: number
  warningItems: number
  avgGrowthRate: number
}

const Capacity: React.FC = () => {
  const [metrics, setMetrics] = useState<CapacityMetric[]>([])
  const [forecasts, setForecasts] = useState<ForecastTrend[]>([])
  const [alerts, setAlerts] = useState<CapacityAlert[]>([])
  const [stats, setStats] = useState<CapacityStats>({
    totalResources: 0,
    criticalItems: 0,
    warningItems: 0,
    avgGrowthRate: 0,
  })
  const [loading, setLoading] = useState(true)
  const [selectedMetric, setSelectedMetric] = useState<string>('cpu')
  const [expandedAlert, setExpandedAlert] = useState<string | null>(null)

  const generateMockMetrics = (): CapacityMetric[] => [
    {
      name: 'CPU Utilization',
      current: 72,
      projected30: 82,
      projected90: 95,
      threshold: 85,
      unit: '%',
      status: 'warning',
    },
    {
      name: 'Memory Usage',
      current: 64,
      projected30: 71,
      projected90: 78,
      threshold: 80,
      unit: '%',
      status: 'healthy',
    },
    {
      name: 'Storage Capacity',
      current: 58,
      projected30: 68,
      projected90: 82,
      threshold: 85,
      unit: '%',
      status: 'warning',
    },
    {
      name: 'Network Bandwidth',
      current: 45,
      projected30: 52,
      projected90: 68,
      threshold: 80,
      unit: 'Gbps',
      status: 'healthy',
    },
    {
      name: 'Database Connections',
      current: 320,
      projected30: 385,
      projected90: 480,
      threshold: 500,
      unit: 'connections',
      status: 'healthy',
    },
    {
      name: 'Node Count',
      current: 12,
      projected30: 14,
      projected90: 18,
      threshold: 25,
      unit: 'nodes',
      status: 'healthy',
    },
  ]

  const generateMockForecasts = (): ForecastTrend[] => {
    const now = Date.now()
    return [
      {
        metric: 'CPU Utilization',
        historicalUsage: [
          { date: new Date(now - 2592000000).toISOString().split('T')[0], usage: 45 },
          { date: new Date(now - 2419200000).toISOString().split('T')[0], usage: 52 },
          { date: new Date(now - 2246400000).toISOString().split('T')[0], usage: 58 },
          { date: new Date(now - 1814400000).toISOString().split('T')[0], usage: 65 },
          { date: new Date(now - 1209600000).toISOString().split('T')[0], usage: 72 },
        ],
        projectedUsage: [
          { date: new Date(now + 2592000000).toISOString().split('T')[0], usage: 82 },
          { date: new Date(now + 7776000000).toISOString().split('T')[0], usage: 95 },
        ],
        growthRate: 6.0,
        exhaustionDate: new Date(now + 5184000000).toISOString().split('T')[0],
      },
      {
        metric: 'Storage Capacity',
        historicalUsage: [
          { date: new Date(now - 2592000000).toISOString().split('T')[0], usage: 38 },
          { date: new Date(now - 2419200000).toISOString().split('T')[0], usage: 42 },
          { date: new Date(now - 2246400000).toISOString().split('T')[0], usage: 48 },
          { date: new Date(now - 1814400000).toISOString().split('T')[0], usage: 53 },
          { date: new Date(now - 1209600000).toISOString().split('T')[0], usage: 58 },
        ],
        projectedUsage: [
          { date: new Date(now + 2592000000).toISOString().split('T')[0], usage: 68 },
          { date: new Date(now + 7776000000).toISOString().split('T')[0], usage: 82 },
        ],
        growthRate: 4.8,
        exhaustionDate: new Date(now + 8640000000).toISOString().split('T')[0],
      },
    ]
  }

  const generateMockAlerts = (): CapacityAlert[] => {
    const now = Date.now()
    return [
      {
        id: 'alert-1',
        metric: 'CPU Utilization',
        severity: 'warning',
        message: 'CPU utilization approaching threshold (currently 72%)',
        timestamp: new Date(now - 3600000).toISOString(),
        recommendation: 'Consider scaling up compute resources or optimizing workload distribution',
      },
      {
        id: 'alert-2',
        metric: 'Storage Capacity',
        severity: 'warning',
        message: 'Storage capacity growing rapidly (58% used)',
        timestamp: new Date(now - 7200000).toISOString(),
        recommendation: 'Plan storage expansion or implement retention policies',
      },
      {
        id: 'alert-3',
        metric: 'Memory Usage',
        severity: 'info',
        message: 'Memory trend shows stable utilization',
        timestamp: new Date(now - 86400000).toISOString(),
        recommendation: 'Continue monitoring; no action required',
      },
      {
        id: 'alert-4',
        metric: 'CPU Utilization',
        severity: 'critical',
        message: 'CPU projected to reach capacity in 68 days',
        timestamp: new Date(now - 172800000).toISOString(),
        recommendation: 'Schedule infrastructure upgrade or implement auto-scaling policies',
      },
      {
        id: 'alert-5',
        metric: 'Database Connections',
        severity: 'info',
        message: 'Connection pool utilization within safe limits',
        timestamp: new Date(now - 259200000).toISOString(),
        recommendation: 'Monitor for sudden spikes during peak traffic',
      },
    ]
  }

  useEffect(() => {
    const loadCapacityData = async () => {
      try {
        setLoading(true)
        const mockMetrics = generateMockMetrics()
        const mockForecasts = generateMockForecasts()
        const mockAlerts = generateMockAlerts()

        setMetrics(mockMetrics)
        setForecasts(mockForecasts)
        setAlerts(mockAlerts)

        const capacityStats: CapacityStats = {
          totalResources: mockMetrics.length,
          criticalItems: mockMetrics.filter((m) => m.status === 'critical').length,
          warningItems: mockMetrics.filter((m) => m.status === 'warning').length,
          avgGrowthRate: mockForecasts.reduce((sum, f) => sum + f.growthRate, 0) / mockForecasts.length,
        }
        setStats(capacityStats)
      } catch (err) {
        console.error('Failed to load capacity data:', err)
      } finally {
        setLoading(false)
      }
    }

    loadCapacityData()
  }, [])

  if (loading) {
    return (
      <AppLayout title="Capacity Planning" subtitle="Resource forecasting and capacity management">
        <Loading message="Loading capacity data..." />
      </AppLayout>
    )
  }

  const criticalAlerts = alerts.filter((a) => a.severity === 'critical')
  const warningAlerts = alerts.filter((a) => a.severity === 'warning')

  return (
    <AppLayout
      title="Capacity Planning"
      subtitle={`${stats.totalResources} resources • ${stats.criticalItems} critical • ${stats.avgGrowthRate.toFixed(1)}% avg growth`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Tracked Resources</p>
              <span className="text-3xl font-bold text-primary-500">{stats.totalResources}</span>
              <p className="text-xs text-neutral-500 mt-2">capacity metrics</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Critical Items</p>
              <span className="text-3xl font-bold text-error-500">{stats.criticalItems}</span>
              <p className="text-xs text-neutral-500 mt-2">require action</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Warnings</p>
              <span className="text-3xl font-bold text-warning-500">{stats.warningItems}</span>
              <p className="text-xs text-neutral-500 mt-2">active alerts</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Avg Growth Rate</p>
              <span className="text-3xl font-bold text-info-500">{stats.avgGrowthRate.toFixed(1)}%</span>
              <p className="text-xs text-neutral-500 mt-2">monthly increase</p>
            </div>
          </Card>
        </div>

        {/* Capacity Metrics */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Resource Capacity</h3>
          </div>
          <div className="p-6 space-y-4">
            {metrics.map((metric) => (
              <div key={metric.name}>
                <div className="flex justify-between items-center mb-2">
                  <div>
                    <p className="text-white font-medium">{metric.name}</p>
                    <p className="text-xs text-neutral-400">
                      Current: {metric.current}{metric.unit} • Threshold: {metric.threshold}{metric.unit}
                    </p>
                  </div>
                  <Badge
                    status={
                      metric.status === 'critical'
                        ? 'error'
                        : metric.status === 'warning'
                          ? 'warning'
                          : 'success'
                    }
                  >
                    {metric.status.charAt(0).toUpperCase() + metric.status.slice(1)}
                  </Badge>
                </div>
                <div className="space-y-1">
                  <div className="flex gap-2 text-xs">
                    <div className="flex-1">
                      <div className="bg-neutral-700 rounded-full h-2 mb-1">
                        <div
                          className="bg-primary-500 h-2 rounded-full"
                          style={{ width: `${Math.min(metric.current, 100)}%` }}
                        />
                      </div>
                      <p className="text-neutral-400">Now</p>
                    </div>
                    <div className="flex-1">
                      <div className="bg-neutral-700 rounded-full h-2 mb-1">
                        <div
                          className="bg-warning-500 h-2 rounded-full"
                          style={{ width: `${Math.min(metric.projected30, 100)}%` }}
                        />
                      </div>
                      <p className="text-neutral-400">30 days</p>
                    </div>
                    <div className="flex-1">
                      <div className="bg-neutral-700 rounded-full h-2 mb-1">
                        <div
                          className={`${
                            metric.projected90 > metric.threshold ? 'bg-error-500' : 'bg-success-500'
                          } h-2 rounded-full`}
                          style={{ width: `${Math.min(metric.projected90, 100)}%` }}
                        />
                      </div>
                      <p className="text-neutral-400">90 days</p>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Forecast Trends */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Usage Forecasts</h3>
          </div>
          <div className="p-6 space-y-6">
            {forecasts.map((forecast) => (
              <div key={forecast.metric}>
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <p className="text-white font-medium">{forecast.metric}</p>
                    <p className="text-xs text-neutral-400">
                      Growth Rate: {forecast.growthRate}% per month
                      {forecast.exhaustionDate && (
                        <>
                          {' • Capacity exhaustion: '}
                          <span className="text-warning-400">{new Date(forecast.exhaustionDate).toLocaleDateString()}</span>
                        </>
                      )}
                    </p>
                  </div>
                </div>
                <div className="bg-neutral-800/30 rounded-lg p-4">
                  <div className="space-y-2 text-xs text-neutral-400">
                    <p>Historical trend (30 days) → Projection (90 days)</p>
                    <div className="flex gap-1 h-16 items-end">
                      {forecast.historicalUsage.map((point, idx) => (
                        <div
                          key={idx}
                          className="flex-1 bg-primary-500/60 rounded-t"
                          style={{ height: `${(point.usage / 100) * 100}%` }}
                          title={`${point.usage}%`}
                        />
                      ))}
                      {forecast.projectedUsage.map((point, idx) => (
                        <div
                          key={idx}
                          className="flex-1 bg-warning-500/60 rounded-t border-l border-neutral-600"
                          style={{ height: `${(point.usage / 100) * 100}%` }}
                          title={`${point.usage}%`}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Capacity Alerts */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">
              Capacity Alerts ({criticalAlerts.length} critical, {warningAlerts.length} warnings)
            </h3>
          </div>
          <div className="divide-y divide-neutral-700">
            {alerts.length === 0 ? (
              <div className="p-6 text-center text-neutral-400">
                No capacity alerts
              </div>
            ) : (
              alerts.map((alert) => (
                <div
                  key={alert.id}
                  className="p-6 hover:bg-neutral-800/20 transition cursor-pointer"
                  onClick={() => setExpandedAlert(expandedAlert === alert.id ? null : alert.id)}
                >
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge
                          status={
                            alert.severity === 'critical'
                              ? 'error'
                              : alert.severity === 'warning'
                                ? 'warning'
                                : 'info'
                          }
                        >
                          {alert.severity.charAt(0).toUpperCase() + alert.severity.slice(1)}
                        </Badge>
                        <span className="text-white font-medium">{alert.metric}</span>
                      </div>
                      <p className="text-sm text-neutral-300 mb-2">{alert.message}</p>
                      <p className="text-xs text-neutral-500">
                        {new Date(alert.timestamp).toLocaleString()}
                      </p>
                    </div>
                  </div>

                  {expandedAlert === alert.id && (
                    <div className="mt-4 p-4 bg-neutral-800/30 rounded-lg border-l-2 border-warning-500">
                      <p className="text-sm font-medium text-neutral-300 mb-2">Recommendation:</p>
                      <p className="text-sm text-neutral-400">{alert.recommendation}</p>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </Card>

        {/* Action Items */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Recommended Actions</h3>
          </div>
          <div className="p-6 space-y-3">
            {metrics
              .filter((m) => m.status !== 'healthy')
              .map((metric) => (
                <div key={metric.name} className="flex items-start gap-3 p-3 bg-neutral-800/30 rounded-lg">
                  <span className="text-warning-400 font-bold mt-1">→</span>
                  <div className="flex-1">
                    <p className="text-white font-medium">{metric.name}</p>
                    <p className="text-xs text-neutral-400 mt-1">
                      {metric.projected90 > metric.threshold
                        ? `Will exceed capacity in ~90 days. Plan ${metric.projected90 - metric.threshold}% additional capacity.`
                        : `Monitor closely. Currently ${metric.current}% of threshold.`}
                    </p>
                  </div>
                  <Button variant="primary" size="sm">
                    Review
                  </Button>
                </div>
              ))}
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Capacity
