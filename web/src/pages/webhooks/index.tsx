import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface Webhook {
  id: string
  url: string
  events: string[]
  status: 'active' | 'inactive'
  secret: string
  created: string
  lastTriggered?: string
  deliveryCount: number
  failureCount: number
}

interface WebhookDelivery {
  id: string
  webhookId: string
  event: string
  status: 'success' | 'failed' | 'pending'
  statusCode?: number
  deliveredAt: string
  retryCount: number
  responseTime?: number
}

const Webhooks: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([])
  const [loading, setLoading] = useState(true)
  const [showWebhookForm, setShowWebhookForm] = useState(false)
  const [newWebhook, setNewWebhook] = useState({
    url: '',
    events: [] as string[],
  })
  const [selectedWebhook, setSelectedWebhook] = useState<string | null>(null)

  const availableEvents = [
    'alert.created',
    'alert.resolved',
    'deployment.started',
    'deployment.completed',
    'node.joined',
    'node.left',
    'policy.updated',
    'certificate.expiring',
    'storage.full',
    'error.occurred',
  ]

  const generateMockWebhooks = (): Webhook[] => [
    {
      id: 'webhook-1',
      url: 'https://api.example.com/webhooks/alerts',
      events: ['alert.created', 'alert.resolved'],
      status: 'active',
      secret: 'whsec_1a2b3c4d5e6f7g8h9i0j',
      created: '2026-08-15T10:30:00Z',
      lastTriggered: '2026-09-30T08:45:00Z',
      deliveryCount: 542,
      failureCount: 3,
    },
    {
      id: 'webhook-2',
      url: 'https://api.example.com/webhooks/deployments',
      events: ['deployment.started', 'deployment.completed'],
      status: 'active',
      secret: 'whsec_2b3c4d5e6f7g8h9i0j1k',
      created: '2026-09-01T14:20:00Z',
      lastTriggered: '2026-09-28T16:30:00Z',
      deliveryCount: 128,
      failureCount: 1,
    },
    {
      id: 'webhook-3',
      url: 'https://monitoring.example.com/webhook',
      events: ['node.joined', 'node.left', 'policy.updated'],
      status: 'active',
      secret: 'whsec_3c4d5e6f7g8h9i0j1k2l',
      created: '2026-07-20T09:15:00Z',
      lastTriggered: '2026-09-30T07:20:00Z',
      deliveryCount: 1250,
      failureCount: 8,
    },
    {
      id: 'webhook-4',
      url: 'https://legacy.example.com/events',
      events: ['error.occurred'],
      status: 'inactive',
      secret: 'whsec_4d5e6f7g8h9i0j1k2l3m',
      created: '2026-06-10T08:00:00Z',
      deliveryCount: 89,
      failureCount: 5,
    },
  ]

  const generateMockDeliveries = (): WebhookDelivery[] => {
    const now = Date.now()
    return [
      {
        id: 'del-1',
        webhookId: 'webhook-1',
        event: 'alert.created',
        status: 'success',
        statusCode: 200,
        deliveredAt: new Date(now - 300000).toISOString(),
        retryCount: 0,
        responseTime: 145,
      },
      {
        id: 'del-2',
        webhookId: 'webhook-2',
        event: 'deployment.completed',
        status: 'success',
        statusCode: 200,
        deliveredAt: new Date(now - 600000).toISOString(),
        retryCount: 0,
        responseTime: 312,
      },
      {
        id: 'del-3',
        webhookId: 'webhook-1',
        event: 'alert.resolved',
        status: 'success',
        statusCode: 200,
        deliveredAt: new Date(now - 900000).toISOString(),
        retryCount: 1,
        responseTime: 289,
      },
      {
        id: 'del-4',
        webhookId: 'webhook-3',
        event: 'policy.updated',
        status: 'failed',
        statusCode: 500,
        deliveredAt: new Date(now - 1200000).toISOString(),
        retryCount: 3,
        responseTime: 5000,
      },
      {
        id: 'del-5',
        webhookId: 'webhook-1',
        event: 'alert.created',
        status: 'success',
        statusCode: 200,
        deliveredAt: new Date(now - 1800000).toISOString(),
        retryCount: 0,
        responseTime: 156,
      },
      {
        id: 'del-6',
        webhookId: 'webhook-2',
        event: 'deployment.started',
        status: 'success',
        statusCode: 200,
        deliveredAt: new Date(now - 2400000).toISOString(),
        retryCount: 0,
        responseTime: 234,
      },
    ]
  }

  useEffect(() => {
    const loadWebhooks = async () => {
      try {
        setLoading(true)
        const mockWebhooks = generateMockWebhooks()
        const mockDeliveries = generateMockDeliveries()
        setWebhooks(mockWebhooks)
        setDeliveries(mockDeliveries)
      } catch (err) {
        console.error('Failed to load webhooks:', err)
      } finally {
        setLoading(false)
      }
    }

    loadWebhooks()
  }, [])

  const handleCreateWebhook = async () => {
    if (!newWebhook.url.trim() || newWebhook.events.length === 0) return

    try {
      await apiClient.post('/webhooks', {
        url: newWebhook.url,
        events: newWebhook.events,
      })

      const webhook: Webhook = {
        id: `webhook-${Date.now()}`,
        url: newWebhook.url,
        events: newWebhook.events,
        status: 'active',
        secret: `whsec_${Math.random().toString(36).substr(2, 20)}`,
        created: new Date().toISOString(),
        deliveryCount: 0,
        failureCount: 0,
      }

      setWebhooks((prev) => [webhook, ...prev])
      setNewWebhook({ url: '', events: [] })
      setShowWebhookForm(false)
    } catch (err) {
      console.error('Failed to create webhook:', err)
    }
  }

  const handleToggleWebhook = async (webhookId: string) => {
    try {
      const webhook = webhooks.find((w) => w.id === webhookId)
      if (!webhook) return

      await apiClient.patch(`/webhooks/${webhookId}`, {
        status: webhook.status === 'active' ? 'inactive' : 'active',
      })

      setWebhooks((prev) =>
        prev.map((w) =>
          w.id === webhookId
            ? {
                ...w,
                status: w.status === 'active' ? 'inactive' : 'active',
              }
            : w
        )
      )
    } catch (err) {
      console.error('Failed to toggle webhook:', err)
    }
  }

  const handleDeleteWebhook = async (webhookId: string) => {
    try {
      await apiClient.delete(`/webhooks/${webhookId}`)
      setWebhooks((prev) => prev.filter((w) => w.id !== webhookId))
    } catch (err) {
      console.error('Failed to delete webhook:', err)
    }
  }

  const handleTestWebhook = async (webhookId: string) => {
    try {
      await apiClient.post(`/webhooks/${webhookId}/test`, {})
    } catch (err) {
      console.error('Failed to test webhook:', err)
    }
  }

  if (loading) {
    return (
      <AppLayout title="Webhooks" subtitle="Webhook management and event delivery">
        <Loading message="Loading webhooks..." />
      </AppLayout>
    )
  }

  const activeWebhooks = webhooks.filter((w) => w.status === 'active').length
  const totalDeliveries = webhooks.reduce((sum, w) => sum + w.deliveryCount, 0)
  const successRate =
    totalDeliveries > 0
      ? (
          ((totalDeliveries - webhooks.reduce((sum, w) => sum + w.failureCount, 0)) /
            totalDeliveries) *
          100
        ).toFixed(1)
      : '100'

  const selectedWebhookData = webhooks.find((w) => w.id === selectedWebhook)
  const webhookDeliveries = selectedWebhook
    ? deliveries.filter((d) => d.webhookId === selectedWebhook)
    : []

  return (
    <AppLayout
      title="Webhooks"
      subtitle={`${activeWebhooks} active • ${totalDeliveries} total deliveries • ${successRate}% success`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Active Webhooks</p>
              <span className="text-3xl font-bold text-primary-500">{activeWebhooks}</span>
              <p className="text-xs text-neutral-500 mt-2">of {webhooks.length} configured</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Deliveries</p>
              <span className="text-3xl font-bold text-info-500">{totalDeliveries}</span>
              <p className="text-xs text-neutral-500 mt-2">all time</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Success Rate</p>
              <span className="text-3xl font-bold text-success-500">{successRate}%</span>
              <p className="text-xs text-neutral-500 mt-2">delivery success</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Events</p>
              <span className="text-3xl font-bold text-warning-500">{availableEvents.length}</span>
              <p className="text-xs text-neutral-500 mt-2">event types</p>
            </div>
          </Card>
        </div>

        {/* Webhooks Management */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold text-white">Webhooks</h3>
              <Button
                variant="primary"
                size="sm"
                onClick={() => setShowWebhookForm(!showWebhookForm)}
              >
                + New Webhook
              </Button>
            </div>
          </div>

          {showWebhookForm && (
            <div className="p-6 border-b border-neutral-700 bg-neutral-800/30">
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Webhook URL
                  </label>
                  <input
                    type="url"
                    value={newWebhook.url}
                    onChange={(e) => setNewWebhook({ ...newWebhook, url: e.target.value })}
                    placeholder="https://api.example.com/webhook"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Events to Subscribe
                  </label>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                    {availableEvents.map((event) => (
                      <label key={event} className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={newWebhook.events.includes(event)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setNewWebhook({
                                ...newWebhook,
                                events: [...newWebhook.events, event],
                              })
                            } else {
                              setNewWebhook({
                                ...newWebhook,
                                events: newWebhook.events.filter((ev) => ev !== event),
                              })
                            }
                          }}
                          className="rounded"
                        />
                        <span className="text-sm text-neutral-300">{event}</span>
                      </label>
                    ))}
                  </div>
                </div>
                <div className="flex gap-2 justify-end">
                  <Button variant="secondary" size="sm" onClick={() => setShowWebhookForm(false)}>
                    Cancel
                  </Button>
                  <Button variant="primary" size="sm" onClick={handleCreateWebhook}>
                    Create Webhook
                  </Button>
                </div>
              </div>
            </div>
          )}

          <div className="divide-y divide-neutral-700">
            {webhooks.map((webhook) => (
              <div key={webhook.id} className="p-6 hover:bg-neutral-800/20 transition">
                <div className="flex justify-between items-start gap-4 mb-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <p className="text-white font-mono text-sm font-medium">{webhook.url}</p>
                      <Badge status={webhook.status === 'active' ? 'success' : 'default'}>
                        {webhook.status === 'active' ? 'Active' : 'Inactive'}
                      </Badge>
                    </div>
                    <div className="flex flex-wrap gap-1 mb-2">
                      {webhook.events.map((event) => (
                        <span
                          key={event}
                          className="px-2 py-1 bg-neutral-700 text-neutral-300 text-xs rounded"
                        >
                          {event}
                        </span>
                      ))}
                    </div>
                    <div className="text-xs text-neutral-500 space-y-1">
                      <p>
                        Created {new Date(webhook.created).toLocaleDateString()} • Deliveries:{' '}
                        {webhook.deliveryCount} • Failures: {webhook.failureCount}
                      </p>
                      {webhook.lastTriggered && (
                        <p>
                          Last triggered {new Date(webhook.lastTriggered).toLocaleString()}
                        </p>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setSelectedWebhook(webhook.id)}
                  >
                    View Deliveries
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => handleTestWebhook(webhook.id)}
                  >
                    Test
                  </Button>
                  <button
                    onClick={() => handleToggleWebhook(webhook.id)}
                    className="px-3 py-1 text-xs font-medium rounded-lg transition bg-neutral-700 hover:bg-neutral-600 text-neutral-300"
                  >
                    {webhook.status === 'active' ? 'Disable' : 'Enable'}
                  </button>
                  <button
                    onClick={() => handleDeleteWebhook(webhook.id)}
                    className="px-3 py-1 text-xs font-medium rounded-lg transition bg-error-900 hover:bg-error-800 text-error-300"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Webhook Deliveries */}
        {selectedWebhook && selectedWebhookData && (
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <div className="flex justify-between items-center">
                <h3 className="text-lg font-semibold text-white">
                  Deliveries for {selectedWebhookData.url}
                </h3>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setSelectedWebhook(null)}
                >
                  Close
                </Button>
              </div>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="border-b border-neutral-700">
                  <tr>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">Event</th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      Status Code
                    </th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      Response Time
                    </th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">Retries</th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      Delivered At
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-700">
                  {webhookDeliveries.map((delivery) => (
                    <tr key={delivery.id} className="hover:bg-neutral-800/30">
                      <td className="px-6 py-3 text-neutral-100 font-medium text-xs">
                        {delivery.event}
                      </td>
                      <td className="px-6 py-3">
                        <Badge status={delivery.status === 'success' ? 'success' : 'error'}>
                          {delivery.status === 'success' ? 'Success' : 'Failed'}
                        </Badge>
                      </td>
                      <td className="px-6 py-3 text-neutral-400 text-xs">
                        {delivery.statusCode || '-'}
                      </td>
                      <td className="px-6 py-3 text-neutral-400 text-xs">
                        {delivery.responseTime ? `${delivery.responseTime}ms` : '-'}
                      </td>
                      <td className="px-6 py-3 text-neutral-400 text-xs">
                        {delivery.retryCount}
                      </td>
                      <td className="px-6 py-3 text-neutral-500 text-xs">
                        {new Date(delivery.deliveredAt).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {/* Available Events */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Available Events</h3>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 p-6">
            {availableEvents.map((event) => (
              <div key={event} className="px-3 py-2 bg-neutral-800/30 rounded border border-neutral-700">
                <p className="text-white font-mono text-sm">{event}</p>
              </div>
            ))}
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Webhooks
