import React, { useState } from 'react'
import { Download, Filter, Archive } from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts'

interface EvidenceEntry {
  id: string
  type: 'log' | 'metric' | 'audit' | 'snapshot' | 'compliance'
  source: string
  timestamp: string
  description: string
  tags: string[]
  size: string
  status: 'active' | 'archived' | 'pending'
}

interface EvidenceCollection {
  name: string
  count: number
  lastUpdate: string
  size: string
}

const mockEvidenceEntries: EvidenceEntry[] = [
  { id: '1', type: 'audit', source: 'Admin Panel', timestamp: '2024-01-15T14:32:10Z', description: 'Security policy update by admin@example.com', tags: ['security', 'policy', 'admin'], size: '2.4 KB', status: 'active' },
  { id: '2', type: 'metric', source: 'Monitoring System', timestamp: '2024-01-15T12:15:00Z', description: 'Performance metrics snapshot during peak hours', tags: ['performance', 'metrics', 'production'], size: '156 MB', status: 'active' },
  { id: '3', type: 'log', source: 'Application Logs', timestamp: '2024-01-15T10:45:30Z', description: 'Error logs from incident investigation', tags: ['incident', 'errors', 'investigation'], size: '8.7 MB', status: 'active' },
  { id: '4', type: 'compliance', source: 'Compliance Scan', timestamp: '2024-01-14T22:00:00Z', description: 'ISO 27001 compliance verification results', tags: ['compliance', 'iso27001', 'certification'], size: '3.2 MB', status: 'active' },
  { id: '5', type: 'snapshot', source: 'System Backup', timestamp: '2024-01-14T18:30:00Z', description: 'Full system state snapshot before deployment', tags: ['backup', 'deployment', 'recovery'], size: '12.4 GB', status: 'archived' },
]

const collectionData: EvidenceCollection[] = [
  { name: 'Audit Logs', count: 1240, lastUpdate: '2024-01-15T14:32:00Z', size: '2.8 GB' },
  { name: 'Performance Metrics', count: 8560, lastUpdate: '2024-01-15T14:00:00Z', size: '45.6 GB' },
  { name: 'Security Events', count: 320, lastUpdate: '2024-01-15T13:45:00Z', size: '1.2 GB' },
  { name: 'Compliance Reports', count: 156, lastUpdate: '2024-01-14T22:30:00Z', size: '3.4 GB' },
  { name: 'System Snapshots', count: 42, lastUpdate: '2024-01-14T18:30:00Z', size: '523 GB' },
  { name: 'Backup Archives', count: 720, lastUpdate: '2024-01-15T03:00:00Z', size: '8.2 TB' },
]

const collectionGrowthData = [
  { month: 'Jan', audit: 240, metrics: 1200, security: 45, compliance: 28, snapshots: 8 },
  { month: 'Feb', audit: 310, metrics: 1540, security: 62, compliance: 34, snapshots: 12 },
  { month: 'Mar', audit: 420, metrics: 1890, security: 85, compliance: 41, snapshots: 16 },
  { month: 'Apr', audit: 560, metrics: 2340, security: 112, compliance: 52, snapshots: 22 },
  { month: 'May', audit: 710, metrics: 2950, security: 148, compliance: 68, snapshots: 28 },
  { month: 'Jun', audit: 920, metrics: 3780, security: 195, compliance: 89, snapshots: 36 },
]

export default function Evidence() {
  const [filterType, setFilterType] = useState('all')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [sortBy, setSortBy] = useState('recent')

  const filteredEntries = filterType === 'all' ? mockEvidenceEntries : mockEvidenceEntries.filter((e) => e.type === filterType)

  const sortedEntries = [...filteredEntries].sort((a, b) => {
    if (sortBy === 'recent') return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    if (sortBy === 'oldest') return new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    if (sortBy === 'size') return parseInt(b.size) - parseInt(a.size)
    return 0
  })

  const totalEvidence = collectionData.reduce((sum, c) => sum + c.count, 0)
  const totalSize = '577 GB'

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Evidence</h1>
            <p className="text-neutral-400 mt-1">Compliance and audit evidence collection and management</p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Total Evidence Entries</p>
            <p className="text-2xl font-bold text-white mt-2">{totalEvidence.toLocaleString()}</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Active Collections</p>
            <p className="text-2xl font-bold text-blue-500 mt-2">6</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Total Storage Used</p>
            <p className="text-2xl font-bold text-green-500 mt-2">{totalSize}</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Last Updated</p>
            <p className="text-2xl font-bold text-purple-500 mt-2">2 hours</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Evidence Growth (6 months)</h3>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={collectionGrowthData}>
                <defs>
                  <linearGradient id="colorAudit" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.8} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
                <XAxis dataKey="month" stroke="#737373" />
                <YAxis stroke="#737373" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px', color: '#e5e5e5' }} />
                <Area type="monotone" dataKey="audit" stackId="1" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.6} name="Audit" />
                <Area type="monotone" dataKey="metrics" stackId="1" stroke="#8b5cf6" fill="#8b5cf6" fillOpacity={0.4} name="Metrics" />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Collections by Type</h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={collectionData.slice(0, 5)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
                <XAxis dataKey="name" stroke="#737373" angle={-45} textAnchor="end" height={80} />
                <YAxis stroke="#737373" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px', color: '#e5e5e5' }} />
                <Bar dataKey="count" fill="#10b981" name="Entry Count" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <div className="flex flex-col sm:flex-row gap-4 mb-6 items-start sm:items-center">
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-white flex items-center gap-2 mb-4">
                <Archive size={20} /> Evidence Entries
              </h3>
            </div>
            <div className="flex gap-2">
              <select value={filterType} onChange={(e) => setFilterType(e.target.value)} className="px-3 py-2 bg-neutral-800 border border-neutral-700 rounded text-white text-sm">
                <option value="all">All Types</option>
                <option value="audit">Audit</option>
                <option value="metric">Metrics</option>
                <option value="log">Logs</option>
                <option value="compliance">Compliance</option>
                <option value="snapshot">Snapshots</option>
              </select>
              <select value={sortBy} onChange={(e) => setSortBy(e.target.value)} className="px-3 py-2 bg-neutral-800 border border-neutral-700 rounded text-white text-sm">
                <option value="recent">Most Recent</option>
                <option value="oldest">Oldest First</option>
                <option value="size">Largest</option>
              </select>
            </div>
          </div>

          <div className="space-y-3">
            {sortedEntries.map((entry) => (
              <div key={entry.id} className="border border-neutral-800 rounded-lg p-4 hover:border-neutral-700 transition-colors">
                <div className="flex items-start justify-between cursor-pointer" onClick={() => setExpandedId(expandedId === entry.id ? null : entry.id)}>
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${entry.type === 'audit' ? 'bg-blue-500/20 text-blue-400' : entry.type === 'metric' ? 'bg-green-500/20 text-green-400' : entry.type === 'log' ? 'bg-yellow-500/20 text-yellow-400' : entry.type === 'compliance' ? 'bg-purple-500/20 text-purple-400' : 'bg-indigo-500/20 text-indigo-400'}`}>
                        {entry.type}
                      </span>
                      <h4 className="font-medium text-white">{entry.description}</h4>
                    </div>
                    <div className="flex flex-wrap gap-2 mt-2">
                      {entry.tags.map((tag) => (
                        <span key={tag} className="px-2 py-1 bg-neutral-800 text-neutral-300 rounded text-xs">
                          {tag}
                        </span>
                      ))}
                    </div>
                    <p className="text-xs text-neutral-400 mt-2">Source: {entry.source} • {new Date(entry.timestamp).toLocaleString()} • Size: {entry.size}</p>
                  </div>
                  <div className="text-right">
                    <span className={`inline-block px-2 py-1 rounded text-xs font-medium mb-2 ${entry.status === 'active' ? 'bg-green-500/20 text-green-400' : 'bg-gray-500/20 text-gray-400'}`}>
                      {entry.status}
                    </span>
                    <button className="block ml-auto px-3 py-1 text-xs bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded transition-colors">
                      <Download size={14} className="inline mr-1" /> Export
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Evidence Collections</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {collectionData.map((collection, idx) => (
              <div key={idx} className="border border-neutral-800 rounded-lg p-4">
                <p className="font-medium text-white">{collection.name}</p>
                <p className="text-sm text-neutral-400 mt-1">{collection.count.toLocaleString()} entries</p>
                <p className="text-xs text-neutral-400 mt-1">Storage: {collection.size}</p>
                <p className="text-xs text-neutral-500 mt-2">Last updated: {new Date(collection.lastUpdate).toLocaleString()}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
