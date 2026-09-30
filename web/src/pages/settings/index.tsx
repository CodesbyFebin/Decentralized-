import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface APIKey {
  id: string
  name: string
  prefix: string
  created: string
  lastUsed: string | null
  scopes: string[]
  status: 'active' | 'revoked'
}

interface SystemConfig {
  version: string
  environment: 'production' | 'staging' | 'development'
  region: string
  backupFrequency: 'hourly' | 'daily' | 'weekly'
  certificateProvider: string
  tlsVersion: string
  logLevel: 'debug' | 'info' | 'warn' | 'error'
}

interface NotificationSettings {
  email: boolean
  slack: boolean
  deploymentAlerts: boolean
  errorAlerts: boolean
  maintenanceNotices: boolean
}

const Settings: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [systemConfig, setSystemConfig] = useState<SystemConfig | null>(null)
  const [notifications, setNotifications] = useState<NotificationSettings>({
    email: true,
    slack: false,
    deploymentAlerts: true,
    errorAlerts: true,
    maintenanceNotices: true,
  })
  const [showCreateKey, setShowCreateKey] = useState(false)
  const [newKeyName, setNewKeyName] = useState('')
  const [newKeyScopes, setNewKeyScopes] = useState<string[]>(['read:deployments'])

  const generateMockAPIKeys = (): APIKey[] => [
    {
      id: 'key-1',
      name: 'CI/CD Pipeline',
      prefix: 'dh_live_abc123def456',
      created: '2026-08-15T10:30:00Z',
      lastUsed: '2026-09-30T09:45:00Z',
      scopes: ['write:deployments', 'read:nodes', 'read:analytics'],
      status: 'active',
    },
    {
      id: 'key-2',
      name: 'Monitoring Integration',
      prefix: 'dh_live_xyz789uvw012',
      created: '2026-07-20T14:20:00Z',
      lastUsed: '2026-09-30T09:30:00Z',
      scopes: ['read:analytics', 'read:nodes'],
      status: 'active',
    },
    {
      id: 'key-3',
      name: 'Old Development Key',
      prefix: 'dh_live_old123456789',
      created: '2026-03-10T08:00:00Z',
      lastUsed: '2026-06-15T16:45:00Z',
      scopes: ['read:all'],
      status: 'active',
    },
    {
      id: 'key-4',
      name: 'Deprecated API Access',
      prefix: 'dh_live_deprecated123',
      created: '2026-01-05T12:00:00Z',
      lastUsed: null,
      scopes: ['read:all', 'write:all'],
      status: 'revoked',
    },
  ]

  const generateMockSystemConfig = (): SystemConfig => ({
    version: '1.0.0-rc.1',
    environment: 'production',
    region: 'us-east-1',
    backupFrequency: 'daily',
    certificateProvider: 'Let\'s Encrypt',
    tlsVersion: '1.3',
    logLevel: 'info',
  })

  useEffect(() => {
    const loadSettings = async () => {
      try {
        setLoading(true)
        const keys = generateMockAPIKeys()
        const config = generateMockSystemConfig()
        setApiKeys(keys)
        setSystemConfig(config)
        setError(null)
      } catch (err) {
        setError('Failed to load settings')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadSettings()
  }, [])

  if (error) {
    return (
      <AppLayout title="Settings" subtitle="Configuration and preferences">
        <ErrorState
          title="Failed to Load Settings"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Settings" subtitle="Configuration and preferences">
        <Loading message="Loading settings..." />
      </AppLayout>
    )
  }

  const activeKeys = apiKeys.filter((k) => k.status === 'active').length
  const recentlyUsedKeys = apiKeys.filter((k) => k.lastUsed).length

  const handleCreateAPIKey = async () => {
    if (!newKeyName.trim()) return
    try {
      await apiClient.post('/settings/api-keys', { name: newKeyName, scopes: newKeyScopes })
      setNewKeyName('')
      setNewKeyScopes(['read:deployments'])
      setShowCreateKey(false)
      const keys = generateMockAPIKeys()
      setApiKeys(keys)
    } catch (err) {
      console.error('Failed to create API key:', err)
    }
  }

  const handleRevokeKey = async (keyId: string) => {
    try {
      await apiClient.post(`/settings/api-keys/${keyId}/revoke`, {})
      const keys = generateMockAPIKeys()
      setApiKeys(keys)
    } catch (err) {
      console.error('Failed to revoke key:', err)
    }
  }

  const handleSaveNotifications = async () => {
    try {
      await apiClient.patch('/settings/notifications', notifications)
    } catch (err) {
      console.error('Failed to save notifications:', err)
    }
  }

  const handleUpdateConfig = async (key: keyof SystemConfig, value: any) => {
    if (!systemConfig) return
    try {
      const updated = { ...systemConfig, [key]: value }
      await apiClient.patch('/settings/system', { [key]: value })
      setSystemConfig(updated)
    } catch (err) {
      console.error('Failed to update config:', err)
    }
  }

  return (
    <AppLayout
      title="Settings"
      subtitle={`${activeKeys} active API keys • ${systemConfig?.environment} environment`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">API Keys Active</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{activeKeys}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">of {apiKeys.length} total</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">System Version</p>
              <div className="flex items-end gap-2">
                <span className="text-2xl font-bold text-secondary-500">
                  {systemConfig?.version}
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">production ready</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Environment</p>
              <div className="flex items-end gap-2">
                <span className="text-2xl font-bold text-success-500 capitalize">
                  {systemConfig?.environment}
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">
                {systemConfig?.region}
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Backup Frequency</p>
              <div className="flex items-end gap-2">
                <span className="text-2xl font-bold text-warning-500 capitalize">
                  {systemConfig?.backupFrequency}
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">automated</p>
            </div>
          </Card>
        </div>

        {/* System Configuration */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">System Configuration</h3>
          </div>
          <div className="p-6 space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  TLS Version
                </label>
                <select
                  value={systemConfig?.tlsVersion || '1.3'}
                  onChange={(e) => handleUpdateConfig('tlsVersion', e.target.value)}
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="1.2">TLS 1.2</option>
                  <option value="1.3">TLS 1.3</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Log Level
                </label>
                <select
                  value={systemConfig?.logLevel || 'info'}
                  onChange={(e) => handleUpdateConfig('logLevel', e.target.value)}
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="debug">Debug</option>
                  <option value="info">Info</option>
                  <option value="warn">Warn</option>
                  <option value="error">Error</option>
                </select>
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Backup Frequency
                </label>
                <select
                  value={systemConfig?.backupFrequency || 'daily'}
                  onChange={(e) => handleUpdateConfig('backupFrequency', e.target.value)}
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="hourly">Hourly</option>
                  <option value="daily">Daily</option>
                  <option value="weekly">Weekly</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Certificate Provider
                </label>
                <input
                  type="text"
                  value={systemConfig?.certificateProvider || ''}
                  disabled
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-neutral-400"
                />
              </div>
            </div>
          </div>
        </Card>

        {/* API Keys */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
            <h3 className="text-lg font-semibold text-white">API Keys</h3>
            <Button
              variant="primary"
              size="md"
              onClick={() => setShowCreateKey(!showCreateKey)}
            >
              + Create Key
            </Button>
          </div>

          {showCreateKey && (
            <div className="p-6 border-b border-neutral-700 bg-neutral-800/50 space-y-3">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Key Name
                </label>
                <input
                  type="text"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  placeholder="e.g., CI/CD Pipeline"
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Scopes
                </label>
                <div className="space-y-2">
                  {['read:analytics', 'read:deployments', 'write:deployments', 'read:nodes', 'write:nodes'].map((scope) => (
                    <label key={scope} className="flex items-center gap-2 text-sm text-neutral-300">
                      <input
                        type="checkbox"
                        checked={newKeyScopes.includes(scope)}
                        onChange={(e) =>
                          e.target.checked
                            ? setNewKeyScopes([...newKeyScopes, scope])
                            : setNewKeyScopes(newKeyScopes.filter((s) => s !== scope))
                        }
                        className="rounded"
                      />
                      {scope}
                    </label>
                  ))}
                </div>
              </div>
              <div className="flex gap-3">
                <Button
                  variant="primary"
                  size="md"
                  onClick={handleCreateAPIKey}
                  disabled={!newKeyName.trim()}
                  className="flex-1"
                >
                  Create API Key
                </Button>
                <Button
                  variant="ghost"
                  size="md"
                  onClick={() => setShowCreateKey(false)}
                  className="flex-1"
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Name</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Prefix</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Scopes</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Created</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Last Used</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {apiKeys.map((key) => (
                  <tr key={key.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{key.name}</td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">{key.prefix}</td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">
                      {key.scopes.length > 0 ? (
                        <span className="text-primary-400">{key.scopes.length} scope{key.scopes.length !== 1 ? 's' : ''}</span>
                      ) : (
                        'No scopes'
                      )}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                      {new Date(key.created).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">
                      {key.lastUsed ? new Date(key.lastUsed).toLocaleString() : '—'}
                    </td>
                    <td className="px-6 py-3">
                      <Badge status={key.status === 'active' ? 'active' : 'error'} size="sm" />
                    </td>
                    <td className="px-6 py-3 flex gap-2">
                      {key.status === 'active' && (
                        <button
                          onClick={() => handleRevokeKey(key.id)}
                          className="text-xs px-2 py-1 bg-error-600 hover:bg-error-500 rounded text-white"
                        >
                          Revoke
                        </button>
                      )}
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        Copy
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Notification Settings */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Notification Preferences</h3>
          </div>
          <div className="p-6 space-y-4">
            <div className="space-y-3">
              {[
                { key: 'email', label: 'Email Notifications' },
                { key: 'slack', label: 'Slack Notifications' },
                { key: 'deploymentAlerts', label: 'Deployment Alerts' },
                { key: 'errorAlerts', label: 'Error Alerts' },
                { key: 'maintenanceNotices', label: 'Maintenance Notices' },
              ].map((item) => (
                <label key={item.key} className="flex items-center gap-3 text-sm text-neutral-300 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={notifications[item.key as keyof NotificationSettings]}
                    onChange={(e) =>
                      setNotifications({
                        ...notifications,
                        [item.key]: e.target.checked,
                      })
                    }
                    className="w-4 h-4 rounded border-neutral-600 bg-neutral-700"
                  />
                  {item.label}
                </label>
              ))}
            </div>
            <Button
              variant="primary"
              size="md"
              onClick={handleSaveNotifications}
              className="w-full md:w-auto"
            >
              Save Preferences
            </Button>
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Settings
