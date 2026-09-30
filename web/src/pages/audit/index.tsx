import React, { useState } from 'react'
import Link from 'next/link'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar } from 'recharts'

interface AuditEntry {
  id: string
  timestamp: string
  actor: string
  action: string
  resource: string
  resourceId: string
  result: 'success' | 'failure'
  ipAddress: string
  details: string
}

interface AuditStats {
  total: number
  byAction: Record<string, number>
  byResult: Record<string, number>
}

const mockAuditEntries: AuditEntry[] = [
  { id: '1', timestamp: '2024-01-15T14:32:10Z', actor: 'admin@example.com', action: 'UPDATE_POLICY', resource: 'security-policy', resourceId: 'pol-456', result: 'success', ipAddress: '203.0.113.45', details: 'Updated firewall rules' },
  { id: '2', timestamp: '2024-01-15T13:21:05Z', actor: 'ops@example.com', action: 'DELETE_RESOURCE', resource: 'vm-instance', resourceId: 'vm-8901', result: 'success', ipAddress: '203.0.113.50', details: 'Removed development instance' },
  { id: '3', timestamp: '2024-01-15T12:45:33Z', actor: 'dev@example.com', action: 'CREATE_ACCESS_KEY', resource: 'api-key', resourceId: 'key-7654', result: 'success', ipAddress: '203.0.113.55', details: 'Generated new API key' },
  { id: '4', timestamp: '2024-01-15T11:20:15Z', actor: 'unknown', action: 'UNAUTHORIZED_ACCESS', resource: 'database', resourceId: 'db-3210', result: 'failure', ipAddress: '198.51.100.89', details: 'Blocked unauthorized access attempt' },
  { id: '5', timestamp: '2024-01-15T10:05:42Z', actor: 'admin@example.com', action: 'MODIFY_CONFIG', resource: 'deployment-config', resourceId: 'cfg-5678', result: 'success', ipAddress: '203.0.113.45', details: 'Updated deployment parameters' },
]

const auditTrendData = [
  { time: '00:00', entries: 12, failures: 2 },
  { time: '04:00', entries: 8, failures: 1 },
  { time: '08:00', entries: 24, failures: 3 },
  { time: '12:00', entries: 35, failures: 2 },
  { time: '16:00', entries: 28, failures: 4 },
  { time: '20:00', entries: 19, failures: 1 },
  { time: '23:59', entries: 15, failures: 2 },
]

const actionBreakdownData = [
  { name: 'UPDATE_POLICY', value: 24 },
  { name: 'CREATE_ACCESS_KEY', value: 18 },
  { name: 'DELETE_RESOURCE', value: 14 },
  { name: 'MODIFY_CONFIG', value: 22 },
  { name: 'OTHER', value: 82 },
]

export default function Audit() {
  const [filter, setFilter] = useState('all')
  const [dateRange, setDateRange] = useState('24h')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const filteredEntries = filter === 'all' ? mockAuditEntries : mockAuditEntries.filter(e => e.result === filter)

  const stats: AuditStats = {
    total: 160,
    byAction: { UPDATE_POLICY: 24, CREATE_ACCESS_KEY: 18, DELETE_RESOURCE: 14, MODIFY_CONFIG: 22 },
    byResult: { success: 155, failure: 5 },
  }

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Audit Log</h1>
            <p className="text-neutral-400 mt-1">Track all administrative actions and access patterns</p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Total Entries (24h)</p>
            <p className="text-2xl font-bold text-white mt-2">160</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Successful Actions</p>
            <p className="text-2xl font-bold text-green-500 mt-2">155</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Failed Actions</p>
            <p className="text-2xl font-bold text-red-500 mt-2">5</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Unique Actors</p>
            <p className="text-2xl font-bold text-blue-500 mt-2">12</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Audit Trend (24h)</h3>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={auditTrendData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
                <XAxis dataKey="time" stroke="#737373" />
                <YAxis stroke="#737373" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px', color: '#e5e5e5' }} />
                <Legend />
                <Line type="monotone" dataKey="entries" stroke="#3b82f6" strokeWidth={2} />
                <Line type="monotone" dataKey="failures" stroke="#ef4444" strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Action Breakdown</h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={actionBreakdownData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
                <XAxis dataKey="name" stroke="#737373" angle={-45} textAnchor="end" height={80} />
                <YAxis stroke="#737373" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px', color: '#e5e5e5' }} />
                <Bar dataKey="value" fill="#8b5cf6" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <div className="flex flex-col sm:flex-row gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-2">Filter by Result</label>
              <select value={filter} onChange={(e) => setFilter(e.target.value)} className="px-3 py-2 bg-neutral-800 border border-neutral-700 rounded text-white">
                <option value="all">All</option>
                <option value="success">Success Only</option>
                <option value="failure">Failures Only</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-2">Date Range</label>
              <select value={dateRange} onChange={(e) => setDateRange(e.target.value)} className="px-3 py-2 bg-neutral-800 border border-neutral-700 rounded text-white">
                <option value="24h">Last 24 Hours</option>
                <option value="7d">Last 7 Days</option>
                <option value="30d">Last 30 Days</option>
              </select>
            </div>
            <div className="flex items-end">
              <button className="px-4 py-2 bg-neutral-800 hover:bg-neutral-700 rounded text-white font-medium transition-colors">
                Export Logs
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-neutral-700">
                  <th className="px-4 py-3 text-left text-neutral-300 font-semibold">Timestamp</th>
                  <th className="px-4 py-3 text-left text-neutral-300 font-semibold">Actor</th>
                  <th className="px-4 py-3 text-left text-neutral-300 font-semibold">Action</th>
                  <th className="px-4 py-3 text-left text-neutral-300 font-semibold">Resource</th>
                  <th className="px-4 py-3 text-left text-neutral-300 font-semibold">Result</th>
                </tr>
              </thead>
              <tbody>
                {filteredEntries.map((entry) => (
                  <React.Fragment key={entry.id}>
                    <tr className="border-b border-neutral-800 hover:bg-neutral-800/50 cursor-pointer" onClick={() => setExpandedId(expandedId === entry.id ? null : entry.id)}>
                      <td className="px-4 py-3 text-neutral-300">{new Date(entry.timestamp).toLocaleString()}</td>
                      <td className="px-4 py-3 text-neutral-300">{entry.actor}</td>
                      <td className="px-4 py-3"><span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-medium">{entry.action}</span></td>
                      <td className="px-4 py-3 text-neutral-400">{entry.resource}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${entry.result === 'success' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
                          {entry.result}
                        </span>
                      </td>
                    </tr>
                    {expandedId === entry.id && (
                      <tr className="bg-neutral-800/30 border-b border-neutral-800">
                        <td colSpan={5} className="px-4 py-4">
                          <div className="space-y-2">
                            <p><span className="text-neutral-400">Resource ID:</span> <span className="text-neutral-300">{entry.resourceId}</span></p>
                            <p><span className="text-neutral-400">IP Address:</span> <span className="text-neutral-300">{entry.ipAddress}</span></p>
                            <p><span className="text-neutral-400">Details:</span> <span className="text-neutral-300">{entry.details}</span></p>
                          </div>
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
