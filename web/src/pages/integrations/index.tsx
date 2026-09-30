import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'

interface Integration {
  id: string
  name: string
  category: string
  status: 'connected' | 'disconnected' | 'error'
  lastSync: string
  events: number
  configuration: string
}

const generateMockIntegrations = (): Integration[] => [
  { id: 'int-001', name: 'Slack', category: 'Notifications', status: 'connected', lastSync: '2026-09-30T10:35Z', events: 1245, configuration: 'Webhook configured' },
  { id: 'int-002', name: 'PagerDuty', category: 'Incident Management', status: 'connected', lastSync: '2026-09-30T10:30Z', events: 284, configuration: 'API key active' },
  { id: 'int-003', name: 'Datadog', category: 'Monitoring', status: 'connected', lastSync: '2026-09-30T10:32Z', events: 5421, configuration: 'Agent running' },
  { id: 'int-004', name: 'GitHub', category: 'VCS', status: 'connected', lastSync: '2026-09-30T09:45Z', events: 342, configuration: 'OAuth connected' },
  { id: 'int-005', name: 'Stripe', category: 'Billing', status: 'error', lastSync: '2026-09-29T18:22Z', events: 0, configuration: 'API key expired' },
  { id: 'int-006', name: 'Splunk', category: 'Analytics', status: 'disconnected', lastSync: '2026-09-28T14:15Z', events: 0, configuration: 'Pending configuration' },
]

export default function IntegrationsPage() {
  const [integrations, setIntegrations] = useState<Integration[]>([])

  useEffect(() => {
    setIntegrations(generateMockIntegrations())
  }, [])

  const connected = integrations.filter(i => i.status === 'connected').length
  const errors = integrations.filter(i => i.status === 'error').length
  const totalEvents = integrations.reduce((sum, i) => sum + i.events, 0)

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected': return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'disconnected': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'error': return 'bg-red-500/20 text-red-400 border-red-500/50'
      default: return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Integrations</h1>
            <p className="text-neutral-400">Manage third-party integrations</p>
          </div>

          <div className="grid grid-cols-5 gap-4">
            <Card><div className="text-neutral-400 text-sm mb-2">Connected</div><div className="text-3xl font-bold text-green-400">{connected}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Errors</div><div className="text-3xl font-bold text-red-400">{errors}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Total Events</div><div className="text-3xl font-bold text-white">{(totalEvents / 1000).toFixed(1)}K</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Integrations</div><div className="text-3xl font-bold text-white">{integrations.length}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Categories</div><div className="text-3xl font-bold text-white">6</div></Card>
          </div>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Available Integrations</h2>
            <div className="space-y-3">
              {integrations.map((int) => (
                <div key={int.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <div className="text-white font-semibold">{int.name}</div>
                      <div className="text-neutral-400 text-sm">{int.category}</div>
                    </div>
                    <Badge className={getStatusColor(int.status)}>
                      {int.status.charAt(0).toUpperCase() + int.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-3 gap-4 text-sm mb-3">
                    <div>
                      <div className="text-neutral-400 mb-1">Configuration</div>
                      <div className="text-white">{int.configuration}</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 mb-1">Events</div>
                      <div className="text-white">{int.events.toLocaleString()}</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 mb-1">Last Sync</div>
                      <div className="text-white">{new Date(int.lastSync).toLocaleTimeString()}</div>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors">
                      {int.status === 'connected' ? 'Configure' : 'Connect'}
                    </button>
                    <button className="text-neutral-400 hover:text-neutral-300 text-sm font-medium transition-colors">
                      {int.status === 'connected' ? 'Disconnect' : 'Remove'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
