import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface ServiceStatus {
  name: string
  status: 'operational' | 'degraded' | 'outage'
  uptime: number
  responseTime: number
  lastIncident?: string
}

interface StatusEvent {
  id: string
  timestamp: string
  title: string
  description: string
  status: 'investigating' | 'identified' | 'monitoring' | 'resolved'
  affectedServices: string[]
  duration?: number
  impact: 'none' | 'minor' | 'major' | 'critical'
}

interface SLAMetric {
  service: string
  current: number
  monthly: number
  quarterly: number
  target: number
}

interface MaintenanceWindow {
  id: string
  title: string
  description: string
  scheduledStart: string
  estimatedDuration: number
  affectedServices: string[]
  status: 'scheduled' | 'in-progress' | 'completed'
}

interface StatusStats {
  operationalServices: number
  totalServices: number
  avgUptime: number
  slaCompliance: number
  avgResponseTime: number
}

const Status: React.FC = () => {
  const [services, setServices] = useState<ServiceStatus[]>([])
  const [events, setEvents] = useState<StatusEvent[]>([])
  const [slaMetrics, setSLAMetrics] = useState<SLAMetric[]>([])
  const [maintenance, setMaintenance] = useState<MaintenanceWindow[]>([])
  const [stats, setStats] = useState<StatusStats>({
    operationalServices: 0,
    totalServices: 0,
    avgUptime: 0,
    slaCompliance: 0,
    avgResponseTime: 0,
  })
  const [loading, setLoading] = useState(true)
  const [expandedEvent, setExpandedEvent] = useState<string | null>(null)
  const [timeRange, setTimeRange] = useState<'24h' | '7d' | '30d'>('24h')

  const generateMockServices = (): ServiceStatus[] => {
    const now = Date.now()
    return [
      {
        name: 'API Gateway',
        status: 'operational',
        uptime: 99.98,
        responseTime: 45,
        lastIncident: new Date(now - 604800000).toISOString(),
      },
      {
        name: 'Database Cluster',
        status: 'operational',
        uptime: 99.95,
        responseTime: 12,
        lastIncident: new Date(now - 1209600000).toISOString(),
      },
      {
        name: 'Cache Layer',
        status: 'degraded',
        uptime: 98.50,
        responseTime: 125,
        lastIncident: new Date(now - 3600000).toISOString(),
      },
      {
        name: 'Storage System',
        status: 'operational',
        uptime: 99.99,
        responseTime: 78,
        lastIncident: new Date(now - 2592000000).toISOString(),
      },
      {
        name: 'CDN Network',
        status: 'operational',
        uptime: 99.97,
        responseTime: 22,
        lastIncident: new Date(now - 1814400000).toISOString(),
      },
      {
        name: 'Monitoring Service',
        status: 'operational',
        uptime: 99.92,
        responseTime: 34,
        lastIncident: new Date(now - 604800000).toISOString(),
      },
    ]
  }

  const generateMockEvents = (): StatusEvent[] => {
    const now = Date.now()
    return [
      {
        id: 'event-1',
        timestamp: new Date(now - 3600000).toISOString(),
        title: 'Cache Layer Degradation Detected',
        description: 'Elevated latency observed on cache layer. Traffic being rerouted. Investigation ongoing.',
        status: 'monitoring',
        affectedServices: ['Cache Layer', 'API Gateway'],
        duration: 45,
        impact: 'minor',
      },
      {
        id: 'event-2',
        timestamp: new Date(now - 86400000).toISOString(),
        title: 'Scheduled Database Maintenance',
        description: 'Routine database optimization and index rebuilding completed successfully.',
        status: 'resolved',
        affectedServices: ['Database Cluster'],
        duration: 120,
        impact: 'none',
      },
      {
        id: 'event-3',
        timestamp: new Date(now - 604800000).toISOString(),
        title: 'Brief Network Connectivity Issue',
        description: 'Temporary network interruption affecting CDN nodes in US-West region resolved.',
        status: 'resolved',
        affectedServices: ['CDN Network'],
        duration: 15,
        impact: 'minor',
      },
      {
        id: 'event-4',
        timestamp: new Date(now - 1209600000).toISOString(),
        title: 'Database Performance Degradation',
        description: 'Query optimizer issue identified and patched. Performance returned to normal.',
        status: 'resolved',
        affectedServices: ['Database Cluster', 'API Gateway'],
        duration: 240,
        impact: 'major',
      },
      {
        id: 'event-5',
        timestamp: new Date(now - 2592000000).toISOString(),
        title: 'Storage System Rebalancing',
        description: 'Routine data rebalancing across storage nodes completed.',
        status: 'resolved',
        affectedServices: ['Storage System'],
        duration: 180,
        impact: 'none',
      },
    ]
  }

  const generateMockSLAMetrics = (): SLAMetric[] => [
    {
      service: 'API Gateway',
      current: 99.98,
      monthly: 99.94,
      quarterly: 99.96,
      target: 99.9,
    },
    {
      service: 'Database Cluster',
      current: 99.95,
      monthly: 99.93,
      quarterly: 99.94,
      target: 99.9,
    },
    {
      service: 'Cache Layer',
      current: 98.50,
      monthly: 98.75,
      quarterly: 98.80,
      target: 99.5,
    },
    {
      service: 'Storage System',
      current: 99.99,
      monthly: 99.98,
      quarterly: 99.97,
      target: 99.9,
    },
  ]

  const generateMockMaintenance = (): MaintenanceWindow[] => {
    const now = Date.now()
    return [
      {
        id: 'maint-1',
        title: 'Network Upgrade - US-East',
        description: 'Upgrading network infrastructure in US-East region for improved redundancy',
        scheduledStart: new Date(now + 604800000).toISOString(),
        estimatedDuration: 120,
        affectedServices: ['API Gateway', 'CDN Network'],
        status: 'scheduled',
      },
      {
        id: 'maint-2',
        title: 'Database Firmware Update',
        description: 'Critical firmware update for database cluster systems',
        scheduledStart: new Date(now + 1209600000).toISOString(),
        estimatedDuration: 90,
        affectedServices: ['Database Cluster'],
        status: 'scheduled',
      },
      {
        id: 'maint-3',
        title: 'TLS Certificate Renewal',
        description: 'Automatic TLS certificate renewal and deployment across all services',
        scheduledStart: new Date(now - 86400000).toISOString(),
        estimatedDuration: 30,
        affectedServices: ['API Gateway', 'CDN Network', 'Monitoring Service'],
        status: 'completed',
      },
    ]
  }

  useEffect(() => {
    const loadStatusData = async () => {
      try {
        setLoading(true)
        const mockServices = generateMockServices()
        const mockEvents = generateMockEvents()
        const mockSLA = generateMockSLAMetrics()
        const mockMaint = generateMockMaintenance()

        setServices(mockServices)
        setEvents(mockEvents)
        setSLAMetrics(mockSLA)
        setMaintenance(mockMaint)

        const operationalCount = mockServices.filter((s) => s.status === 'operational').length
        const avgUptime = mockServices.reduce((sum, s) => sum + s.uptime, 0) / mockServices.length
        const avgResponseTime = mockServices.reduce((sum, s) => sum + s.responseTime, 0) / mockServices.length
        const slaComp = mockSLA.reduce((sum, s) => sum + (s.current >= s.target ? 1 : 0), 0) / mockSLA.length

        setStats({
          operationalServices: operationalCount,
          totalServices: mockServices.length,
          avgUptime: Math.round(avgUptime * 100) / 100,
          slaCompliance: Math.round(slaComp * 100),
          avgResponseTime: Math.round(avgResponseTime),
        })
      } catch (err) {
        console.error('Failed to load status data:', err)
      } finally {
        setLoading(false)
      }
    }

    loadStatusData()
  }, [])

  if (loading) {
    return (
      <AppLayout title="Status" subtitle="System status and incident history">
        <Loading message="Loading status data..." />
      </AppLayout>
    )
  }

  const resolvedEvents = events.filter((e) => e.status === 'resolved')
  const activeEvents = events.filter((e) => e.status !== 'resolved')

  return (
    <AppLayout
      title="Status"
      subtitle={`${stats.operationalServices}/${stats.totalServices} operational • ${stats.avgUptime}% uptime • ${stats.slaCompliance}% SLA compliance`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Operational</p>
              <span className="text-3xl font-bold text-success-500">{stats.operationalServices}</span>
              <p className="text-xs text-neutral-500 mt-2">of {stats.totalServices} services</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Avg Uptime</p>
              <span className="text-3xl font-bold text-primary-500">{stats.avgUptime}%</span>
              <p className="text-xs text-neutral-500 mt-2">current period</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">SLA Compliance</p>
              <span className="text-3xl font-bold text-success-500">{stats.slaCompliance}%</span>
              <p className="text-xs text-neutral-500 mt-2">services compliant</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Avg Response</p>
              <span className="text-3xl font-bold text-info-500">{stats.avgResponseTime}ms</span>
              <p className="text-xs text-neutral-500 mt-2">latency</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Active Incidents</p>
              <span className="text-3xl font-bold text-warning-500">{activeEvents.length}</span>
              <p className="text-xs text-neutral-500 mt-2">being monitored</p>
            </div>
          </Card>
        </div>

        {/* Service Status */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Service Status</h3>
          </div>
          <div className="divide-y divide-neutral-700">
            {services.map((service) => (
              <div key={service.name} className="p-6 hover:bg-neutral-800/20 transition">
                <div className="flex justify-between items-center mb-3">
                  <div className="flex items-center gap-3">
                    <div
                      className="w-3 h-3 rounded-full"
                      style={{
                        backgroundColor:
                          service.status === 'operational'
                            ? '#10b981'
                            : service.status === 'degraded'
                              ? '#f59e0b'
                              : '#ef4444',
                      }}
                    />
                    <div>
                      <p className="text-white font-medium">{service.name}</p>
                      <p className="text-xs text-neutral-400">
                        {service.responseTime}ms • Uptime: {service.uptime}%
                      </p>
                    </div>
                  </div>
                  <Badge
                    status={
                      service.status === 'operational'
                        ? 'success'
                        : service.status === 'degraded'
                          ? 'warning'
                          : 'error'
                    }
                  >
                    {service.status.charAt(0).toUpperCase() + service.status.slice(1)}
                  </Badge>
                </div>
                <div className="bg-neutral-700 rounded-full h-2">
                  <div
                    className={`${
                      service.status === 'operational'
                        ? 'bg-success-500'
                        : service.status === 'degraded'
                          ? 'bg-warning-500'
                          : 'bg-error-500'
                    } h-2 rounded-full`}
                    style={{ width: `${service.uptime}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Status Events / Incidents */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">
              Incident History ({activeEvents.length} active, {resolvedEvents.length} resolved)
            </h3>
          </div>
          <div className="divide-y divide-neutral-700">
            {events.length === 0 ? (
              <div className="p-6 text-center text-neutral-400">
                No incidents recorded
              </div>
            ) : (
              events.map((event) => (
                <div
                  key={event.id}
                  className="p-6 hover:bg-neutral-800/20 transition cursor-pointer"
                  onClick={() => setExpandedEvent(expandedEvent === event.id ? null : event.id)}
                >
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge
                          status={
                            event.impact === 'critical'
                              ? 'error'
                              : event.impact === 'major'
                                ? 'warning'
                                : event.impact === 'minor'
                                  ? 'info'
                                  : 'success'
                          }
                        >
                          {event.impact.charAt(0).toUpperCase() + event.impact.slice(1)}
                        </Badge>
                        <Badge
                          status={
                            event.status === 'resolved'
                              ? 'success'
                              : event.status === 'identified'
                                ? 'warning'
                                : 'info'
                          }
                        >
                          {event.status.charAt(0).toUpperCase() + event.status.slice(1)}
                        </Badge>
                        <span className="text-white font-medium">{event.title}</span>
                      </div>
                      <p className="text-sm text-neutral-400 mb-2">{event.description}</p>
                      <div className="flex flex-wrap gap-2 mb-2">
                        {event.affectedServices.map((svc) => (
                          <span key={svc} className="px-2 py-1 bg-neutral-700/50 rounded text-xs text-neutral-300">
                            {svc}
                          </span>
                        ))}
                      </div>
                      <p className="text-xs text-neutral-500">
                        {new Date(event.timestamp).toLocaleString()}
                        {event.duration && ` • Duration: ${event.duration} minutes`}
                      </p>
                    </div>
                  </div>

                  {expandedEvent === event.id && (
                    <div className="mt-4 p-4 bg-neutral-800/30 rounded-lg border-l-2 border-info-500">
                      <p className="text-sm text-neutral-300">
                        <strong>Full Description:</strong> {event.description}
                      </p>
                      {event.duration && (
                        <p className="text-sm text-neutral-400 mt-2">
                          <strong>Duration:</strong> {event.duration} minutes
                        </p>
                      )}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </Card>

        {/* SLA Metrics */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">SLA Performance</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Service</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Current</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Monthly</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Quarterly</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Target</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {slaMetrics.map((metric) => (
                  <tr key={metric.service} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{metric.service}</td>
                    <td className="px-6 py-3 text-neutral-300 font-mono text-xs">{metric.current}%</td>
                    <td className="px-6 py-3 text-neutral-300 font-mono text-xs">{metric.monthly}%</td>
                    <td className="px-6 py-3 text-neutral-300 font-mono text-xs">{metric.quarterly}%</td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">{metric.target}%</td>
                    <td className="px-6 py-3">
                      <Badge status={metric.current >= metric.target ? 'success' : 'warning'}>
                        {metric.current >= metric.target ? 'Compliant' : 'Below Target'}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Scheduled Maintenance */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Scheduled Maintenance</h3>
          </div>
          <div className="divide-y divide-neutral-700">
            {maintenance.length === 0 ? (
              <div className="p-6 text-center text-neutral-400">
                No scheduled maintenance
              </div>
            ) : (
              maintenance.map((window) => (
                <div key={window.id} className="p-6 hover:bg-neutral-800/20 transition">
                  <div className="flex justify-between items-start gap-4 mb-3">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge
                          status={
                            window.status === 'scheduled'
                              ? 'info'
                              : window.status === 'in-progress'
                                ? 'warning'
                                : 'success'
                          }
                        >
                          {window.status.charAt(0).toUpperCase() + window.status.slice(1)}
                        </Badge>
                        <span className="text-white font-medium">{window.title}</span>
                      </div>
                      <p className="text-sm text-neutral-400 mb-2">{window.description}</p>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4 text-xs mb-3">
                    <div>
                      <p className="text-neutral-400 mb-1">Scheduled Start</p>
                      <p className="text-neutral-200">{new Date(window.scheduledStart).toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-neutral-400 mb-1">Estimated Duration</p>
                      <p className="text-neutral-200">{window.estimatedDuration} minutes</p>
                    </div>
                  </div>
                  <div>
                    <p className="text-xs text-neutral-400 mb-2">Affected Services:</p>
                    <div className="flex flex-wrap gap-2">
                      {window.affectedServices.map((svc) => (
                        <span key={svc} className="px-2 py-1 bg-neutral-700/50 rounded text-xs text-neutral-300">
                          {svc}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Status
