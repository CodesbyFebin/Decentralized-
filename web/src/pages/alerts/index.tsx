import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface AlertRule {
  id: string
  name: string
  condition: string
  threshold: number
  severity: 'critical' | 'high' | 'medium' | 'low'
  enabled: boolean
  actions: string[]
  created: string
  lastTriggered?: string
}

interface Alert {
  id: string
  ruleId: string
  ruleName: string
  severity: 'critical' | 'high' | 'medium' | 'low'
  message: string
  status: 'active' | 'resolved'
  triggeredAt: string
  resolvedAt?: string
  acknowledgedBy?: string
}

const Alerts: React.FC = () => {
  const [rules, setRules] = useState<AlertRule[]>([])
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [showRuleForm, setShowRuleForm] = useState(false)
  const [newRule, setNewRule] = useState({ name: '', condition: '', threshold: 0, severity: 'high' as const })

  const generateMockRules = (): AlertRule[] => [
    {
      id: 'rule-1',
      name: 'High CPU Usage',
      condition: 'cpu_utilization > 85%',
      threshold: 85,
      severity: 'high',
      enabled: true,
      actions: ['Email', 'Slack', 'PagerDuty'],
      created: '2026-08-15T10:30:00Z',
      lastTriggered: '2026-09-30T08:45:00Z',
    },
    {
      id: 'rule-2',
      name: 'Memory Pressure',
      condition: 'memory_utilization > 80%',
      threshold: 80,
      severity: 'high',
      enabled: true,
      actions: ['Email', 'Slack'],
      created: '2026-08-20T14:20:00Z',
      lastTriggered: '2026-09-29T22:15:00Z',
    },
    {
      id: 'rule-3',
      name: 'Disk Space Critical',
      condition: 'disk_free < 5%',
      threshold: 5,
      severity: 'critical',
      enabled: true,
      actions: ['Email', 'PagerDuty', 'Webhook'],
      created: '2026-08-10T09:00:00Z',
      lastTriggered: '2026-09-28T16:30:00Z',
    },
    {
      id: 'rule-4',
      name: 'High Error Rate',
      condition: 'error_rate > 5%',
      threshold: 5,
      severity: 'medium',
      enabled: true,
      actions: ['Email', 'Slack'],
      created: '2026-09-01T11:45:00Z',
      lastTriggered: '2026-09-30T07:20:00Z',
    },
    {
      id: 'rule-5',
      name: 'Node Offline',
      condition: 'node_status == offline',
      threshold: 0,
      severity: 'critical',
      enabled: true,
      actions: ['PagerDuty', 'Email'],
      created: '2026-07-25T08:15:00Z',
      lastTriggered: '2026-09-27T19:45:00Z',
    },
  ]

  const generateMockAlerts = (): Alert[] => [
    {
      id: 'alert-1',
      ruleId: 'rule-1',
      ruleName: 'High CPU Usage',
      severity: 'high',
      message: 'Node prod-worker-03 CPU utilization exceeded 85% threshold',
      status: 'active',
      triggeredAt: '2026-09-30T08:45:00Z',
    },
    {
      id: 'alert-2',
      ruleId: 'rule-4',
      ruleName: 'High Error Rate',
      severity: 'medium',
      message: '/api/deployments endpoint showing 5.2% error rate',
      status: 'active',
      triggeredAt: '2026-09-30T07:20:00Z',
    },
    {
      id: 'alert-3',
      ruleId: 'rule-2',
      ruleName: 'Memory Pressure',
      severity: 'high',
      message: 'Node prod-master-02 memory utilization at 82%',
      status: 'resolved',
      triggeredAt: '2026-09-29T22:15:00Z',
      resolvedAt: '2026-09-30T01:30:00Z',
      acknowledgedBy: 'alice@example.com',
    },
    {
      id: 'alert-4',
      ruleId: 'rule-3',
      ruleName: 'Disk Space Critical',
      severity: 'critical',
      message: 'Storage node store-01 disk free space at 4.2%',
      status: 'active',
      triggeredAt: '2026-09-28T16:30:00Z',
    },
    {
      id: 'alert-5',
      ruleId: 'rule-1',
      ruleName: 'High CPU Usage',
      severity: 'high',
      message: 'Node prod-worker-01 CPU utilization exceeded 85% threshold',
      status: 'resolved',
      triggeredAt: '2026-09-28T14:20:00Z',
      resolvedAt: '2026-09-28T15:45:00Z',
      acknowledgedBy: 'bob@example.com',
    },
  ]

  useEffect(() => {
    const loadAlerts = async () => {
      try {
        setLoading(true)
        const mockRules = generateMockRules()
        const mockAlerts = generateMockAlerts()
        setRules(mockRules)
        setAlerts(mockAlerts)
      } catch (err) {
        console.error('Failed to load alerts:', err)
      } finally {
        setLoading(false)
      }
    }

    loadAlerts()
  }, [])

  const handleCreateRule = async () => {
    if (!newRule.name.trim()) return

    try {
      await apiClient.post('/alerts/rules', {
        name: newRule.name,
        condition: newRule.condition,
        threshold: newRule.threshold,
        severity: newRule.severity,
      })

      const rule: AlertRule = {
        id: `rule-${Date.now()}`,
        name: newRule.name,
        condition: newRule.condition,
        threshold: newRule.threshold,
        severity: newRule.severity,
        enabled: true,
        actions: ['Email'],
        created: new Date().toISOString(),
      }

      setRules((prev) => [rule, ...prev])
      setNewRule({ name: '', condition: '', threshold: 0, severity: 'high' })
      setShowRuleForm(false)
    } catch (err) {
      console.error('Failed to create rule:', err)
    }
  }

  const handleToggleRule = async (ruleId: string) => {
    try {
      const rule = rules.find((r) => r.id === ruleId)
      if (!rule) return

      await apiClient.patch(`/alerts/rules/${ruleId}`, {
        enabled: !rule.enabled,
      })

      setRules((prev) =>
        prev.map((r) => (r.id === ruleId ? { ...r, enabled: !r.enabled } : r))
      )
    } catch (err) {
      console.error('Failed to toggle rule:', err)
    }
  }

  const handleAcknowledgeAlert = async (alertId: string) => {
    try {
      await apiClient.post(`/alerts/${alertId}/acknowledge`, {})
      setAlerts((prev) =>
        prev.map((a) =>
          a.id === alertId ? { ...a, acknowledgedBy: 'current-user@example.com' } : a
        )
      )
    } catch (err) {
      console.error('Failed to acknowledge alert:', err)
    }
  }

  if (loading) {
    return (
      <AppLayout title="Alerts & Monitoring" subtitle="Alert rules and notification management">
        <Loading message="Loading alerts..." />
      </AppLayout>
    )
  }

  const activeAlerts = alerts.filter((a) => a.status === 'active')
  const criticalAlerts = activeAlerts.filter((a) => a.severity === 'critical')
  const resolved24h = alerts.filter(
    (a) =>
      a.status === 'resolved' &&
      new Date(a.resolvedAt || '').getTime() > Date.now() - 86400000
  ).length

  return (
    <AppLayout
      title="Alerts & Monitoring"
      subtitle={`${activeAlerts.length} active • ${criticalAlerts.length} critical`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Active Alerts</p>
              <span className="text-3xl font-bold text-warning-500">{activeAlerts.length}</span>
              <p className="text-xs text-neutral-500 mt-2">requiring attention</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Critical</p>
              <span className="text-3xl font-bold text-error-500">{criticalAlerts.length}</span>
              <p className="text-xs text-neutral-500 mt-2">severity alerts</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Alert Rules</p>
              <span className="text-3xl font-bold text-primary-500">{rules.length}</span>
              <p className="text-xs text-neutral-500 mt-2">
                {rules.filter((r) => r.enabled).length} enabled
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Resolved (24h)</p>
              <span className="text-3xl font-bold text-success-500">{resolved24h}</span>
              <p className="text-xs text-neutral-500 mt-2">alerts resolved</p>
            </div>
          </Card>
        </div>

        {/* Alert Rules */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold text-white">Alert Rules</h3>
              <Button
                variant="primary"
                size="sm"
                onClick={() => setShowRuleForm(!showRuleForm)}
              >
                + New Rule
              </Button>
            </div>
          </div>

          {showRuleForm && (
            <div className="p-6 border-b border-neutral-700 bg-neutral-800/30">
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Rule Name
                  </label>
                  <input
                    type="text"
                    value={newRule.name}
                    onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
                    placeholder="e.g., High CPU Usage"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Condition
                  </label>
                  <input
                    type="text"
                    value={newRule.condition}
                    onChange={(e) => setNewRule({ ...newRule, condition: e.target.value })}
                    placeholder="e.g., cpu_utilization > 85%"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-neutral-300 mb-2">
                      Threshold
                    </label>
                    <input
                      type="number"
                      value={newRule.threshold}
                      onChange={(e) =>
                        setNewRule({ ...newRule, threshold: parseInt(e.target.value) || 0 })
                      }
                      className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-neutral-300 mb-2">
                      Severity
                    </label>
                    <select
                      value={newRule.severity}
                      onChange={(e) =>
                        setNewRule({
                          ...newRule,
                          severity: e.target.value as 'critical' | 'high' | 'medium' | 'low',
                        })
                      }
                      className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                    >
                      <option value="critical">Critical</option>
                      <option value="high">High</option>
                      <option value="medium">Medium</option>
                      <option value="low">Low</option>
                    </select>
                  </div>
                </div>
                <div className="flex gap-2 justify-end">
                  <Button variant="secondary" size="sm" onClick={() => setShowRuleForm(false)}>
                    Cancel
                  </Button>
                  <Button variant="primary" size="sm" onClick={handleCreateRule}>
                    Create Rule
                  </Button>
                </div>
              </div>
            </div>
          )}

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Rule Name</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Condition</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Severity</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Last Triggered</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {rules.map((rule) => (
                  <tr key={rule.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{rule.name}</td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">{rule.condition}</td>
                    <td className="px-6 py-3">
                      <Badge
                        status={
                          rule.severity === 'critical'
                            ? 'error'
                            : rule.severity === 'high'
                              ? 'warning'
                              : 'info'
                        }
                      >
                        {rule.severity.charAt(0).toUpperCase() + rule.severity.slice(1)}
                      </Badge>
                    </td>
                    <td className="px-6 py-3">
                      <button
                        onClick={() => handleToggleRule(rule.id)}
                        className={`px-3 py-1 rounded-full text-xs font-medium transition ${
                          rule.enabled
                            ? 'bg-success-500/20 text-success-300'
                            : 'bg-neutral-700 text-neutral-400'
                        }`}
                      >
                        {rule.enabled ? 'Enabled' : 'Disabled'}
                      </button>
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">
                      {rule.actions.join(', ')}
                    </td>
                    <td className="px-6 py-3 text-neutral-500 text-xs">
                      {rule.lastTriggered
                        ? new Date(rule.lastTriggered).toLocaleString()
                        : 'Never'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Active Alerts */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Active Alerts</h3>
          </div>
          <div className="divide-y divide-neutral-700">
            {activeAlerts.length === 0 ? (
              <div className="p-6 text-center text-neutral-400">No active alerts</div>
            ) : (
              activeAlerts.map((alert) => (
                <div key={alert.id} className="p-6 hover:bg-neutral-800/20 transition">
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge status={alert.severity === 'critical' ? 'error' : 'warning'}>
                          {alert.severity.toUpperCase()}
                        </Badge>
                        <span className="text-white font-medium">{alert.ruleName}</span>
                      </div>
                      <p className="text-neutral-400 text-sm mb-2">{alert.message}</p>
                      <p className="text-neutral-500 text-xs">
                        Triggered {new Date(alert.triggeredAt).toLocaleString()}
                      </p>
                    </div>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => handleAcknowledgeAlert(alert.id)}
                    >
                      Acknowledge
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        </Card>

        {/* Alert History */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Alert History</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Alert</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Severity</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Triggered</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Resolved</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Acknowledged By</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {alerts.map((alert) => (
                  <tr key={alert.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{alert.ruleName}</td>
                    <td className="px-6 py-3">
                      <Badge status={alert.severity === 'critical' ? 'error' : 'warning'}>
                        {alert.severity.charAt(0).toUpperCase() + alert.severity.slice(1)}
                      </Badge>
                    </td>
                    <td className="px-6 py-3">
                      <Badge status={alert.status === 'active' ? 'warning' : 'success'}>
                        {alert.status === 'active' ? 'Active' : 'Resolved'}
                      </Badge>
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">
                      {new Date(alert.triggeredAt).toLocaleString()}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">
                      {alert.resolvedAt ? new Date(alert.resolvedAt).toLocaleString() : '-'}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">
                      {alert.acknowledgedBy || '-'}
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

export default Alerts
