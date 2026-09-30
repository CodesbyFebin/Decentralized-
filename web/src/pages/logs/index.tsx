import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'

interface LogEntry {
  id: string
  timestamp: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  service: string
  message: string
  metadata?: Record<string, any>
}

const generateMockLogs = (): LogEntry[] => [
  { id: '1', timestamp: '2026-09-30T10:35:22Z', level: 'ERROR', service: 'API Server', message: 'Database connection timeout after 5s', metadata: { duration: 5000, host: 'db-001' } },
  { id: '2', timestamp: '2026-09-30T10:34:15Z', level: 'WARN', service: 'Cache', message: 'High memory usage detected', metadata: { memory: '85%' } },
  { id: '3', timestamp: '2026-09-30T10:33:40Z', level: 'INFO', service: 'Deployment', message: 'Deployment completed successfully', metadata: { version: 'v2.5.1', nodes: 12 } },
  { id: '4', timestamp: '2026-09-30T10:32:08Z', level: 'DEBUG', service: 'Scheduler', message: 'Task queued for execution', metadata: { taskId: 'task-8234' } },
  { id: '5', timestamp: '2026-09-30T10:30:45Z', level: 'ERROR', service: 'Storage', message: 'Failed to write to bucket', metadata: { bucket: 'app-data', size: '2.5GB' } },
]

export default function LogsPage() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [filter, setFilter] = useState<string>('all')

  useEffect(() => {
    setLogs(generateMockLogs())
  }, [])

  const filteredLogs = filter === 'all' ? logs : logs.filter(l => l.level === filter)
  const errorCount = logs.filter(l => l.level === 'ERROR').length
  const warnCount = logs.filter(l => l.level === 'WARN').length

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'ERROR': return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'WARN': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'INFO': return 'bg-blue-500/20 text-blue-400 border-blue-500/50'
      case 'DEBUG': return 'bg-neutral-700/20 text-neutral-400 border-neutral-600/50'
      default: return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Logs & Events</h1>
            <p className="text-neutral-400">View system logs and events</p>
          </div>

          <div className="grid grid-cols-5 gap-4">
            <Card><div className="text-neutral-400 text-sm mb-2">Total Logs</div><div className="text-3xl font-bold text-white">{logs.length}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Errors</div><div className="text-3xl font-bold text-red-400">{errorCount}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Warnings</div><div className="text-3xl font-bold text-yellow-400">{warnCount}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Info</div><div className="text-3xl font-bold text-blue-400">{logs.filter(l => l.level === 'INFO').length}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Debug</div><div className="text-3xl font-bold text-neutral-400">{logs.filter(l => l.level === 'DEBUG').length}</div></Card>
          </div>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">System Logs</h2>
            <div className="mb-4 flex gap-2">
              {['all', 'ERROR', 'WARN', 'INFO', 'DEBUG'].map(level => (
                <button
                  key={level}
                  onClick={() => setFilter(level)}
                  className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
                    filter === level
                      ? 'bg-primary-500/30 text-primary-400 border border-primary-500/50'
                      : 'bg-neutral-800 text-neutral-300 hover:bg-neutral-700'
                  }`}
                >
                  {level}
                </button>
              ))}
            </div>
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {filteredLogs.map((log) => (
                <div key={log.id} className="p-3 bg-neutral-900 rounded border border-neutral-700">
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Badge className={getLevelColor(log.level)}>{log.level}</Badge>
                      <span className="text-neutral-400 text-sm font-mono">{log.service}</span>
                    </div>
                    <span className="text-neutral-500 text-xs">{new Date(log.timestamp).toLocaleTimeString()}</span>
                  </div>
                  <div className="text-white text-sm">{log.message}</div>
                  {log.metadata && (
                    <div className="text-xs text-neutral-500 mt-2 font-mono">
                      {JSON.stringify(log.metadata)}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
