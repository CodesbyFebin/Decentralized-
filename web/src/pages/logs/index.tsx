import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface LogEntry {
  id: string
  timestamp: string
  level: 'debug' | 'info' | 'warn' | 'error'
  source: string
  message: string
  traceId?: string
  userId?: string
  metadata?: Record<string, string>
}

interface LogStats {
  debug: number
  info: number
  warn: number
  error: number
  total: number
}

const Logs: React.FC = () => {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [stats, setStats] = useState<LogStats>({ debug: 0, info: 0, warn: 0, error: 0, total: 0 })
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedLevel, setSelectedLevel] = useState<'all' | 'debug' | 'info' | 'warn' | 'error'>(
    'all'
  )
  const [selectedSource, setSelectedSource] = useState<string>('all')
  const [timeRange, setTimeRange] = useState<'1h' | '6h' | '24h' | '7d'>('24h')

  const generateMockLogs = (): LogEntry[] => {
    const sources = ['api', 'scheduler', 'storage', 'network', 'auth', 'policy', 'state']
    const messages = [
      'Request processed successfully',
      'Policy evaluation completed',
      'State transition logged',
      'Resource allocation updated',
      'Consensus checkpoint reached',
      'Certificate renewal scheduled',
      'Deployment rollout initiated',
      'Node health check passed',
      'Storage cleanup executed',
      'TLS handshake completed',
      'RPC call received',
      'Admission control evaluated',
      'Intent validation successful',
      'Merkle proof generated',
      'Raft log replicated',
    ]

    const errors = [
      'Connection timeout to peer node',
      'Policy evaluation failed',
      'Storage quota exceeded',
      'Certificate validation failed',
      'Consensus timeout',
      'TLS certificate expired',
      'Node unavailable',
      'RPC call failed',
    ]

    const now = Date.now()
    const logs: LogEntry[] = []

    for (let i = 0; i < 50; i++) {
      const isError = Math.random() > 0.85
      const isWarn = Math.random() > 0.75 && !isError
      const isDebug = Math.random() > 0.5 && !isError && !isWarn

      const level: 'debug' | 'info' | 'warn' | 'error' = isError
        ? 'error'
        : isWarn
          ? 'warn'
          : isDebug
            ? 'debug'
            : 'info'

      const source = sources[Math.floor(Math.random() * sources.length)]
      const message =
        level === 'error'
          ? errors[Math.floor(Math.random() * errors.length)]
          : messages[Math.floor(Math.random() * messages.length)]

      logs.push({
        id: `log-${i}`,
        timestamp: new Date(now - Math.random() * 86400000 * 7).toISOString(),
        level,
        source,
        message,
        traceId: `trace-${Math.random().toString(36).substr(2, 9)}`,
        userId: Math.random() > 0.3 ? `user-${Math.floor(Math.random() * 10)}` : undefined,
        metadata: {
          duration: `${Math.floor(Math.random() * 5000)}ms`,
          region: ['us-east-1', 'eu-west-1', 'ap-southeast-1'][
            Math.floor(Math.random() * 3)
          ],
        },
      })
    }

    return logs.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
  }

  useEffect(() => {
    const loadLogs = async () => {
      try {
        setLoading(true)
        const mockLogs = generateMockLogs()
        setLogs(mockLogs)

        const logStats: LogStats = {
          debug: mockLogs.filter((l) => l.level === 'debug').length,
          info: mockLogs.filter((l) => l.level === 'info').length,
          warn: mockLogs.filter((l) => l.level === 'warn').length,
          error: mockLogs.filter((l) => l.level === 'error').length,
          total: mockLogs.length,
        }
        setStats(logStats)
      } catch (err) {
        console.error('Failed to load logs:', err)
      } finally {
        setLoading(false)
      }
    }

    loadLogs()
  }, [])

  const filteredLogs = logs.filter((log) => {
    const matchesLevel = selectedLevel === 'all' || log.level === selectedLevel
    const matchesSource = selectedSource === 'all' || log.source === selectedSource
    const matchesSearch =
      searchQuery === '' ||
      log.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
      log.source.toLowerCase().includes(searchQuery.toLowerCase()) ||
      log.traceId?.includes(searchQuery)

    return matchesLevel && matchesSource && matchesSearch
  })

  const sources = Array.from(new Set(logs.map((l) => l.source))).sort()

  if (loading) {
    return (
      <AppLayout title="Logs & Diagnostics" subtitle="System and audit logs with search">
        <Loading message="Loading logs..." />
      </AppLayout>
    )
  }

  return (
    <AppLayout
      title="Logs & Diagnostics"
      subtitle={`${stats.total} logs • ${stats.error} errors • ${stats.warn} warnings`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Logs</p>
              <span className="text-3xl font-bold text-primary-500">{stats.total}</span>
              <p className="text-xs text-neutral-500 mt-2">last 24 hours</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Debug</p>
              <span className="text-3xl font-bold text-neutral-400">{stats.debug}</span>
              <p className="text-xs text-neutral-500 mt-2">diagnostic logs</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Info</p>
              <span className="text-3xl font-bold text-info-500">{stats.info}</span>
              <p className="text-xs text-neutral-500 mt-2">informational</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Warnings</p>
              <span className="text-3xl font-bold text-warning-500">{stats.warn}</span>
              <p className="text-xs text-neutral-500 mt-2">warnings</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Errors</p>
              <span className="text-3xl font-bold text-error-500">{stats.error}</span>
              <p className="text-xs text-neutral-500 mt-2">error logs</p>
            </div>
          </Card>
        </div>

        {/* Log Viewer */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white mb-4">Log Viewer</h3>

            <div className="space-y-4">
              {/* Search */}
              <div>
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search logs by message, source, or trace ID..."
                  className="w-full px-4 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>

              {/* Filters */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Log Level
                  </label>
                  <select
                    value={selectedLevel}
                    onChange={(e) =>
                      setSelectedLevel(e.target.value as typeof selectedLevel)
                    }
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="all">All Levels</option>
                    <option value="debug">Debug</option>
                    <option value="info">Info</option>
                    <option value="warn">Warning</option>
                    <option value="error">Error</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Source
                  </label>
                  <select
                    value={selectedSource}
                    onChange={(e) => setSelectedSource(e.target.value)}
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="all">All Sources</option>
                    {sources.map((source) => (
                      <option key={source} value={source}>
                        {source.charAt(0).toUpperCase() + source.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-2">
                    Time Range
                  </label>
                  <select
                    value={timeRange}
                    onChange={(e) => setTimeRange(e.target.value as typeof timeRange)}
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="1h">Last 1 hour</option>
                    <option value="6h">Last 6 hours</option>
                    <option value="24h">Last 24 hours</option>
                    <option value="7d">Last 7 days</option>
                  </select>
                </div>
              </div>
            </div>
          </div>

          {/* Log Table */}
          <div className="overflow-x-auto max-h-[600px] overflow-y-auto">
            <table className="w-full text-sm font-mono">
              <thead className="border-b border-neutral-700 sticky top-0 bg-neutral-900">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Timestamp</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Level</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Source</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Message</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Trace ID</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {filteredLogs.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-4 text-center text-neutral-500">
                      No logs found matching the selected filters
                    </td>
                  </tr>
                ) : (
                  filteredLogs.map((log) => (
                    <tr key={log.id} className="hover:bg-neutral-800/30 transition">
                      <td className="px-6 py-3 text-neutral-500 whitespace-nowrap text-xs">
                        {new Date(log.timestamp).toLocaleString()}
                      </td>
                      <td className="px-6 py-3 whitespace-nowrap">
                        <Badge
                          status={
                            log.level === 'error'
                              ? 'error'
                              : log.level === 'warn'
                                ? 'warning'
                                : log.level === 'debug'
                                  ? 'default'
                                  : 'success'
                          }
                        >
                          {log.level.toUpperCase()}
                        </Badge>
                      </td>
                      <td className="px-6 py-3 text-neutral-300 whitespace-nowrap text-xs">
                        {log.source}
                      </td>
                      <td className="px-6 py-3 text-neutral-400 text-xs max-w-96 truncate">
                        {log.message}
                      </td>
                      <td className="px-6 py-3 text-neutral-500 whitespace-nowrap text-xs">
                        {log.traceId ? (
                          <span className="bg-neutral-800 px-2 py-1 rounded">{log.traceId}</span>
                        ) : (
                          '-'
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <div className="p-6 border-t border-neutral-700 text-center text-neutral-400 text-sm">
            Showing {filteredLogs.length} of {stats.total} logs
          </div>
        </Card>

        {/* Log Statistics by Source */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Logs by Source</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Source</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Debug</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Info</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Warnings</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Errors</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {sources.map((source) => {
                  const sourceLogs = logs.filter((l) => l.source === source)
                  return (
                    <tr key={source} className="hover:bg-neutral-800/30">
                      <td className="px-6 py-3 text-neutral-100 font-medium">
                        {source.charAt(0).toUpperCase() + source.slice(1)}
                      </td>
                      <td className="px-6 py-3 text-neutral-400">
                        {sourceLogs.filter((l) => l.level === 'debug').length}
                      </td>
                      <td className="px-6 py-3 text-neutral-400">
                        {sourceLogs.filter((l) => l.level === 'info').length}
                      </td>
                      <td className="px-6 py-3 text-neutral-400">
                        {sourceLogs.filter((l) => l.level === 'warn').length}
                      </td>
                      <td className="px-6 py-3 text-neutral-400">
                        {sourceLogs.filter((l) => l.level === 'error').length}
                      </td>
                      <td className="px-6 py-3 text-neutral-300 font-medium">{sourceLogs.length}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Logs
