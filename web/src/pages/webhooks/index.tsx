import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

interface Webhook {
  id: string
  url: string
  events: string[]
  status: 'active' | 'disabled' | 'failed'
  successRate: number
  lastDelivery: string
  deliveries: number
}

const generateMockWebhooks = (): Webhook[] => [
  { id: 'wh-001', url: 'https://api.example.com/webhooks/events', events: ['deployment.completed', 'alert.fired'], status: 'active', successRate: 99.8, lastDelivery: '2026-09-30T10:35Z', deliveries: 1240 },
  { id: 'wh-002', url: 'https://slack.example.com/webhooks/incidents', events: ['incident.created', 'incident.resolved'], status: 'active', successRate: 100, lastDelivery: '2026-09-30T10:32Z', deliveries: 284 },
  { id: 'wh-003', url: 'https://monitoring.example.com/webhooks', events: ['metric.threshold', 'health.check'], status: 'active', successRate: 98.5, lastDelivery: '2026-09-30T10:30Z', deliveries: 5421 },
  { id: 'wh-004', url: 'https://logs.example.com/webhooks/errors', events: ['error.logged'], status: 'failed', successRate: 45, lastDelivery: '2026-09-29T22:15Z', deliveries: 342 },
  { id: 'wh-005', url: 'https://archive.example.com/webhooks', events: ['backup.completed'], status: 'disabled', successRate: 0, lastDelivery: '2026-09-20T14:20Z', deliveries: 156 },
]

const generateMetrics = () => [
  { time: '00:00', delivered: 120, failed: 2 },
  { time: '08:00', delivered: 450, failed: 8 },
  { time: '16:00', delivered: 680, failed: 12 },
  { time: '23:59', delivered: 280, failed: 5 },
]

export default function WebhooksPage() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [metrics, setMetrics] = useState<any[]>([])

  useEffect(() => {
    setWebhooks(generateMockWebhooks())
    setMetrics(generateMetrics())
  }, [])

  const active = webhooks.filter(w => w.status === 'active').length
  const failed = webhooks.filter(w => w.status === 'failed').length
  const totalDeliveries = webhooks.reduce((sum, w) => sum + w.deliveries, 0)
  const avgSuccessRate = (webhooks.reduce((sum, w) => sum + w.successRate, 0) / webhooks.length).toFixed(1)

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'disabled': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'failed': return 'bg-red-500/20 text-red-400 border-red-500/50'
      default: return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Webhooks</h1>
            <p className="text-neutral-400">Configure and monitor event webhooks</p>
          </div>

          <div className="grid grid-cols-5 gap-4">
            <Card><div className="text-neutral-400 text-sm mb-2">Active</div><div className="text-3xl font-bold text-green-400">{active}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Failed</div><div className="text-3xl font-bold text-red-400">{failed}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Total Deliveries</div><div className="text-3xl font-bold text-white">{(totalDeliveries / 1000).toFixed(1)}K</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Avg Success</div><div className="text-3xl font-bold text-green-400">{avgSuccessRate}%</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Webhooks</div><div className="text-3xl font-bold text-white">{webhooks.length}</div></Card>
          </div>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Delivery Success Rate (24h)</h2>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={metrics}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="time" stroke="#999" />
                <YAxis stroke="#999" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                <Legend />
                <Line type="monotone" dataKey="delivered" stroke="#00D9FF" dot={false} name="Delivered" />
                <Line type="monotone" dataKey="failed" stroke="#EF4444" dot={false} name="Failed" />
              </LineChart>
            </ResponsiveContainer>
          </Card>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Configured Webhooks</h2>
            <div className="space-y-3">
              {webhooks.map((wh) => (
                <div key={wh.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex-1">
                      <div className="text-white font-mono text-sm">{wh.url}</div>
                      <div className="text-neutral-400 text-sm mt-1">Events: {wh.events.join(', ')}</div>
                    </div>
                    <Badge className={getStatusColor(wh.status)}>
                      {wh.status.charAt(0).toUpperCase() + wh.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <div className="text-neutral-400 mb-1">Success Rate</div>
                      <div className="text-white">{wh.successRate}%</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 mb-1">Deliveries</div>
                      <div className="text-white">{wh.deliveries.toLocaleString()}</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 mb-1">Last Delivery</div>
                      <div className="text-white">{new Date(wh.lastDelivery).toLocaleTimeString()}</div>
                    </div>
                  </div>
                  <div className="mt-3 flex gap-2">
                    <button className="text-primary-400 hover:text-primary-300 text-sm font-medium">Edit</button>
                    <button className="text-neutral-400 hover:text-neutral-300 text-sm font-medium">{wh.status === 'active' ? 'Disable' : 'Enable'}</button>
                  </div>
                </div>
              ))}
            </div>
            <button className="mt-6 px-4 py-2 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 font-medium transition-colors">
              Add Webhook
            </button>
          </Card>
        </div>
      </div>
    </div>
  )
}
