import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

interface Alert {
  id: string
  title: string
  description: string
  severity: 'critical' | 'warning' | 'info'
  source: string
  status: 'firing' | 'resolved' | 'acknowledged'
  created: string
  affectedResources: string[]
}

const generateMockAlerts = (): Alert[] => [
  { id: '1', title: 'High CPU Usage', description: 'CPU > 85%', severity: 'critical', source: 'CPU Monitor', status: 'firing', created: '2026-09-30T10:15Z', affectedResources: ['node-us-east'] },
  { id: '2', title: 'Replication Lag', description: 'DB lag > 2s', severity: 'warning', source: 'Database', status: 'firing', created: '2026-09-30T10:22Z', affectedResources: ['db-replica-2'] },
  { id: '3', title: 'API Errors High', description: 'Error rate 2%+', severity: 'warning', source: 'API', status: 'acknowledged', created: '2026-09-30T09:45Z', affectedResources: ['api-server-1'] },
  { id: '4', title: 'Disk Space Low', description: 'Available < 10%', severity: 'warning', source: 'Storage', status: 'firing', created: '2026-09-30T08:30Z', affectedResources: ['storage-node-3'] },
  { id: '5', title: 'Cert Expiring', description: 'Expires in 7 days', severity: 'info', source: 'Security', status: 'resolved', created: '2026-09-28T12:00Z', affectedResources: ['api.example.com'] },
]

const generateMetrics = () => [
  { time: '00:00', critical: 2, warning: 8, info: 12 },
  { time: '08:00', critical: 3, warning: 12, info: 18 },
  { time: '16:00', critical: 4, warning: 18, info: 25 },
  { time: '23:59', critical: 1, warning: 6, info: 10 },
]

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [metrics, setMetrics] = useState<any[]>([])

  useEffect(() => {
    setAlerts(generateMockAlerts())
    setMetrics(generateMetrics())
  }, [])

  const firing = alerts.filter(a => a.status === 'firing').length
  const critical = alerts.filter(a => a.severity === 'critical').length

  const getSeverityColor = (sev: string) => sev === 'critical' ? 'bg-red-500/20 text-red-400 border-red-500/50' : sev === 'warning' ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50' : 'bg-blue-500/20 text-blue-400 border-blue-500/50'

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Alerts & Notifications</h1>
            <p className="text-neutral-400">Monitor and manage system alerts</p>
          </div>

          <div className="grid grid-cols-5 gap-4">
            <Card><div className="text-neutral-400 text-sm mb-2">Firing</div><div className="text-3xl font-bold text-red-400">{firing}</div><div className="text-xs text-neutral-400">Active alerts</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Critical</div><div className="text-3xl font-bold text-red-400">{critical}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Warning</div><div className="text-3xl font-bold text-yellow-400">{alerts.filter(a => a.severity === 'warning').length}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Acknowledged</div><div className="text-3xl font-bold text-white">{alerts.filter(a => a.status === 'acknowledged').length}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Resolved</div><div className="text-3xl font-bold text-green-400">{alerts.filter(a => a.status === 'resolved').length}</div></Card>
          </div>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Alert Trend (24h)</h2>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={metrics}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="time" stroke="#999" />
                <YAxis stroke="#999" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                <Legend />
                <Bar dataKey="critical" stackId="a" fill="#EF4444" />
                <Bar dataKey="warning" stackId="a" fill="#F59E0B" />
                <Bar dataKey="info" stackId="a" fill="#3B82F6" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Active Alerts</h2>
            <div className="space-y-3">
              {alerts.filter(a => a.status !== 'resolved').map((alert) => (
                <div key={alert.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex justify-between items-start mb-2">
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <Badge className={getSeverityColor(alert.severity)}>{alert.severity}</Badge>
                        <div className="text-white font-semibold">{alert.title}</div>
                      </div>
                      <div className="text-neutral-400 text-sm">{alert.description}</div>
                    </div>
                    <Badge className={alert.status === 'firing' ? 'bg-red-500/20 text-red-400 border-red-500/50' : 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'}>{alert.status}</Badge>
                  </div>
                  <div className="text-xs text-neutral-400">{alert.source} • {alert.affectedResources.join(', ')}</div>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
