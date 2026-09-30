import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface Integration {
  id: string
  name: string
  service: 'slack' | 'email' | 'pagerduty' | 'webhook' | 'github' | 'datadog'
  status: 'connected' | 'disconnected' | 'error'
  connectedAt: string
  lastUsed?: string
  testStatus?: 'passed' | 'failed'
  lastTest?: string
}

interface AvailableIntegration {
  id: string
  name: string
  service: string
  icon: string
  description: string
  isConnected: boolean
}

const Integrations: React.FC = () => {
  const [integrations, setIntegrations] = useState<Integration[]>([])
  const [available, setAvailable] = useState<AvailableIntegration[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedIntegration, setSelectedIntegration] = useState<string | null>(null)
  const [showConfigForm, setShowConfigForm] = useState(false)
  const [configData, setConfigData] = useState({
    apiKey: '',
    webhookUrl: '',
    channelId: '',
    email: '',
  })

  const generateMockIntegrations = (): Integration[] => [
    {
      id: 'int-1',
      name: 'Slack Notifications',
      service: 'slack',
      status: 'connected',
      connectedAt: '2026-08-10T14:30:00Z',
      lastUsed: '2026-09-30T08:45:00Z',
      testStatus: 'passed',
      lastTest: '2026-09-30T08:30:00Z',
    },
    {
      id: 'int-2',
      name: 'PagerDuty Alerts',
      service: 'pagerduty',
      status: 'connected',
      connectedAt: '2026-08-15T10:00:00Z',
      lastUsed: '2026-09-29T22:15:00Z',
      testStatus: 'passed',
      lastTest: '2026-09-29T22:00:00Z',
    },
    {
      id: 'int-3',
      name: 'Email Notifications',
      service: 'email',
      status: 'connected',
      connectedAt: '2026-07-20T09:15:00Z',
      lastUsed: '2026-09-30T07:20:00Z',
      testStatus: 'passed',
      lastTest: '2026-09-30T07:00:00Z',
    },
    {
      id: 'int-4',
      name: 'GitHub Integration',
      service: 'github',
      status: 'connected',
      connectedAt: '2026-09-01T11:45:00Z',
      lastUsed: '2026-09-28T16:30:00Z',
      testStatus: 'failed',
      lastTest: '2026-09-28T16:20:00Z',
    },
    {
      id: 'int-5',
      name: 'Datadog Monitoring',
      service: 'datadog',
      status: 'disconnected',
      connectedAt: '2026-07-10T08:00:00Z',
    },
  ]

  const generateAvailableIntegrations = (): AvailableIntegration[] => [
    {
      id: 'avail-1',
      name: 'Slack',
      service: 'slack',
      icon: '💬',
      description: 'Send alerts and notifications to Slack channels',
      isConnected: true,
    },
    {
      id: 'avail-2',
      name: 'PagerDuty',
      service: 'pagerduty',
      icon: '📞',
      description: 'Incident response and on-call management',
      isConnected: true,
    },
    {
      id: 'avail-3',
      name: 'Email',
      service: 'email',
      icon: '📧',
      description: 'Send email notifications and reports',
      isConnected: true,
    },
    {
      id: 'avail-4',
      name: 'GitHub',
      service: 'github',
      icon: '🐙',
      description: 'Repository integration and deployment tracking',
      isConnected: true,
    },
    {
      id: 'avail-5',
      name: 'Datadog',
      service: 'datadog',
      icon: '📊',
      description: 'Forward metrics and logs to Datadog',
      isConnected: false,
    },
    {
      id: 'avail-6',
      name: 'Prometheus',
      service: 'prometheus',
      icon: '⚡',
      description: 'Metrics scraping and alerting',
      isConnected: false,
    },
  ]

  useEffect(() => {
    const loadIntegrations = async () => {
      try {
        setLoading(true)
        const mockIntegrations = generateMockIntegrations()
        const mockAvailable = generateAvailableIntegrations()
        setIntegrations(mockIntegrations)
        setAvailable(mockAvailable)
      } catch (err) {
        console.error('Failed to load integrations:', err)
      } finally {
        setLoading(false)
      }
    }

    loadIntegrations()
  }, [])

  const handleTestIntegration = async (integrationId: string) => {
    try {
      await apiClient.post(`/integrations/${integrationId}/test`, {})
      setIntegrations((prev) =>
        prev.map((i) =>
          i.id === integrationId
            ? { ...i, testStatus: 'passed', lastTest: new Date().toISOString() }
            : i
        )
      )
    } catch (err) {
      console.error('Test failed:', err)
      setIntegrations((prev) =>
        prev.map((i) =>
          i.id === integrationId
            ? { ...i, testStatus: 'failed', lastTest: new Date().toISOString() }
            : i
        )
      )
    }
  }

  const handleDisconnect = async (integrationId: string) => {
    try {
      await apiClient.delete(`/integrations/${integrationId}`)
      setIntegrations((prev) => prev.filter((i) => i.id !== integrationId))
    } catch (err) {
      console.error('Failed to disconnect:', err)
    }
  }

  const handleConnect = async () => {
    if (!selectedIntegration) return

    try {
      await apiClient.post('/integrations', {
        service: selectedIntegration,
        config: configData,
      })

      const newIntegration: Integration = {
        id: `int-${Date.now()}`,
        name: available.find((a) => a.service === selectedIntegration)?.name || 'New Integration',
        service: selectedIntegration as Integration['service'],
        status: 'connected',
        connectedAt: new Date().toISOString(),
      }

      setIntegrations((prev) => [newIntegration, ...prev])
      setShowConfigForm(false)
      setSelectedIntegration(null)
      setConfigData({ apiKey: '', webhookUrl: '', channelId: '', email: '' })
    } catch (err) {
      console.error('Failed to connect integration:', err)
    }
  }

  if (loading) {
    return (
      <AppLayout title="Integrations" subtitle="Third-party service integrations">
        <Loading message="Loading integrations..." />
      </AppLayout>
    )
  }

  const connected = integrations.filter((i) => i.status === 'connected').length
  const passedTests = integrations.filter((i) => i.testStatus === 'passed').length

  return (
    <AppLayout
      title="Integrations"
      subtitle={`${connected} connected • ${passedTests} passing tests`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Connected</p>
              <span className="text-3xl font-bold text-success-500">{connected}</span>
              <p className="text-xs text-neutral-500 mt-2">integrations active</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Tests Passing</p>
              <span className="text-3xl font-bold text-primary-500">{passedTests}</span>
              <p className="text-xs text-neutral-500 mt-2">connectivity verified</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Integrations</p>
              <span className="text-3xl font-bold text-info-500">{integrations.length}</span>
              <p className="text-xs text-neutral-500 mt-2">configured</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Available</p>
              <span className="text-3xl font-bold text-warning-500">{available.length}</span>
              <p className="text-xs text-neutral-500 mt-2">services available</p>
            </div>
          </Card>
        </div>

        {/* Connected Integrations */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Connected Integrations</h3>
          </div>

          {integrations.length === 0 ? (
            <div className="p-6 text-center text-neutral-400">
              No integrations connected yet. Browse available integrations below.
            </div>
          ) : (
            <div className="divide-y divide-neutral-700">
              {integrations.map((integration) => (
                <div key={integration.id} className="p-6 hover:bg-neutral-800/20 transition">
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <span className="text-2xl">
                          {available.find((a) => a.service === integration.service)?.icon}
                        </span>
                        <div>
                          <p className="text-white font-medium">{integration.name}</p>
                          <p className="text-xs text-neutral-500">
                            {integration.service.charAt(0).toUpperCase() +
                              integration.service.slice(1)}
                          </p>
                        </div>
                      </div>
                      <div className="flex flex-wrap gap-4 mt-3 text-xs text-neutral-400">
                        <span>
                          Connected {new Date(integration.connectedAt).toLocaleDateString()}
                        </span>
                        {integration.lastUsed && (
                          <span>
                            Last used{' '}
                            {new Date(integration.lastUsed).toLocaleTimeString()}
                          </span>
                        )}
                        {integration.testStatus && (
                          <span className={integration.testStatus === 'passed' ? 'text-success-400' : 'text-error-400'}>
                            Test {integration.testStatus}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Badge status={integration.status === 'connected' ? 'success' : 'error'}>
                        {integration.status === 'connected' ? 'Connected' : 'Disconnected'}
                      </Badge>
                      {integration.testStatus && (
                        <Badge
                          status={integration.testStatus === 'passed' ? 'success' : 'error'}
                        >
                          Test {integration.testStatus === 'passed' ? '✓' : '✗'}
                        </Badge>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2 mt-4">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => handleTestIntegration(integration.id)}
                    >
                      Test Connection
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => handleDisconnect(integration.id)}
                    >
                      Disconnect
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>

        {/* Available Integrations */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Available Integrations</h3>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 p-6">
            {available.map((integration) => {
              const isConnected = integrations.some((i) => i.service === integration.service)

              return (
                <div
                  key={integration.id}
                  className="p-4 bg-neutral-800/30 rounded-lg border border-neutral-700 hover:border-primary-500/50 transition"
                >
                  <div className="flex items-start gap-3 mb-3">
                    <span className="text-3xl">{integration.icon}</span>
                    <div>
                      <p className="text-white font-medium">{integration.name}</p>
                      <p className="text-xs text-neutral-500">{integration.service}</p>
                    </div>
                  </div>
                  <p className="text-sm text-neutral-400 mb-4">{integration.description}</p>
                  {isConnected ? (
                    <Badge status="success">Connected</Badge>
                  ) : (
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => {
                        setSelectedIntegration(integration.service)
                        setShowConfigForm(true)
                      }}
                      className="w-full"
                    >
                      Connect
                    </Button>
                  )}
                </div>
              )
            })}
          </div>
        </Card>

        {/* Configuration Form Modal */}
        {showConfigForm && selectedIntegration && (
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">
                Configure{' '}
                {available.find((a) => a.service === selectedIntegration)?.name}
              </h3>
            </div>
            <div className="p-6 bg-neutral-800/30 space-y-4">
              {['slack', 'pagerduty', 'datadog', 'prometheus'].includes(
                selectedIntegration
              ) && (
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    API Key
                  </label>
                  <input
                    type="password"
                    value={configData.apiKey}
                    onChange={(e) =>
                      setConfigData({ ...configData, apiKey: e.target.value })
                    }
                    placeholder="Enter API key"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
              )}

              {selectedIntegration === 'slack' && (
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Channel ID
                  </label>
                  <input
                    type="text"
                    value={configData.channelId}
                    onChange={(e) =>
                      setConfigData({ ...configData, channelId: e.target.value })
                    }
                    placeholder="e.g., C123456789"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
              )}

              {selectedIntegration === 'email' && (
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Email Address
                  </label>
                  <input
                    type="email"
                    value={configData.email}
                    onChange={(e) =>
                      setConfigData({ ...configData, email: e.target.value })
                    }
                    placeholder="notifications@example.com"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
              )}

              {selectedIntegration === 'webhook' && (
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Webhook URL
                  </label>
                  <input
                    type="url"
                    value={configData.webhookUrl}
                    onChange={(e) =>
                      setConfigData({ ...configData, webhookUrl: e.target.value })
                    }
                    placeholder="https://example.com/webhook"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
              )}

              <div className="flex gap-2 justify-end pt-4">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    setShowConfigForm(false)
                    setSelectedIntegration(null)
                  }}
                >
                  Cancel
                </Button>
                <Button variant="primary" size="sm" onClick={handleConnect}>
                  Connect Integration
                </Button>
              </div>
            </div>
          </Card>
        )}
      </div>
    </AppLayout>
  )
}

export default Integrations
