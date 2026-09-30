import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'

interface Database {
  id: string
  name: string
  type: 'postgres' | 'mysql' | 'mongodb' | 'redis'
  status: 'healthy' | 'warning' | 'critical'
  size: number
  connections: number
  maxConnections: number
  queryTime: number
  replicationLag: number
  uptime: number
}

interface QueryMetric {
  time: string
  avgTime: number
  maxTime: number
  count: number
}

interface ConnectionPool {
  id: string
  name: string
  type: string
  active: number
  idle: number
  waiting: number
  maxSize: number
}

interface SlowQuery {
  id: string
  query: string
  executionTime: number
  callCount: number
  lastExecuted: string
  status: 'identified' | 'optimizing' | 'optimized'
}

interface ReplicationStatus {
  replicaId: string
  replicaName: string
  status: 'synced' | 'lagging' | 'disconnected'
  lag: number
  lag_bytes: number
  replicationRate: number
}

interface IndexMetric {
  name: string
  size: number
  scanCount: number
  unused: boolean
}

const generateMockDatabases = (): Database[] => [
  {
    id: 'db-001',
    name: 'Production PostgreSQL Primary',
    type: 'postgres',
    status: 'healthy',
    size: 482.5,
    connections: 248,
    maxConnections: 500,
    queryTime: 12.5,
    replicationLag: 0.2,
    uptime: 99.98,
  },
  {
    id: 'db-002',
    name: 'Analytics MySQL',
    type: 'mysql',
    status: 'healthy',
    size: 245.8,
    connections: 156,
    maxConnections: 300,
    queryTime: 8.3,
    replicationLag: 0.5,
    uptime: 99.95,
  },
  {
    id: 'db-003',
    name: 'Document Store MongoDB',
    type: 'mongodb',
    status: 'warning',
    size: 512.2,
    connections: 89,
    maxConnections: 150,
    queryTime: 18.7,
    replicationLag: 1.2,
    uptime: 99.92,
  },
  {
    id: 'db-004',
    name: 'Cache Redis Cluster',
    type: 'redis',
    status: 'healthy',
    size: 42.1,
    connections: 342,
    maxConnections: 1000,
    queryTime: 0.8,
    replicationLag: 0,
    uptime: 99.99,
  },
]

const generateMockQueryMetrics = (): QueryMetric[] => [
  { time: '00:00', avgTime: 12.3, maxTime: 245, count: 1200 },
  { time: '04:00', avgTime: 9.8, maxTime: 180, count: 680 },
  { time: '08:00', avgTime: 14.2, maxTime: 520, count: 2400 },
  { time: '12:00', avgTime: 16.5, maxTime: 890, count: 3200 },
  { time: '16:00', avgTime: 18.3, maxTime: 1240, count: 3800 },
  { time: '20:00', avgTime: 14.8, maxTime: 680, count: 2900 },
  { time: '23:59', avgTime: 11.2, maxTime: 320, count: 1500 },
]

const generateMockConnectionPools = (): ConnectionPool[] => [
  { id: 'pool-001', name: 'App Server Pool', type: 'postgres', active: 45, idle: 55, waiting: 2, maxSize: 100 },
  { id: 'pool-002', name: 'Analytics Pool', type: 'mysql', active: 28, idle: 42, waiting: 0, maxSize: 75 },
  { id: 'pool-003', name: 'Background Workers', type: 'postgres', active: 12, idle: 38, waiting: 1, maxSize: 50 },
  { id: 'pool-004', name: 'Cache Pool', type: 'redis', active: 120, idle: 280, waiting: 0, maxSize: 500 },
]

const generateMockSlowQueries = (): SlowQuery[] => [
  { id: 'sq-001', query: 'SELECT * FROM orders LEFT JOIN customers WHERE date > ?', executionTime: 2340, callCount: 856, lastExecuted: '2026-09-30T10:24:12Z', status: 'optimizing' },
  { id: 'sq-002', query: 'SELECT COUNT(*) FROM events WHERE status = ? GROUP BY type', executionTime: 1820, callCount: 2340, lastExecuted: '2026-09-30T10:25:33Z', status: 'identified' },
  { id: 'sq-003', query: 'UPDATE user_preferences SET last_login = ? WHERE user_id = ?', executionTime: 890, callCount: 4200, lastExecuted: '2026-09-30T10:26:45Z', status: 'optimized' },
]

const generateMockReplicationStatus = (): ReplicationStatus[] => [
  { replicaId: 'replica-001', replicaName: 'PostgreSQL Replica 1', status: 'synced', lag: 0.1, lag_bytes: 2048, replicationRate: 15.2 },
  { replicaId: 'replica-002', replicaName: 'PostgreSQL Replica 2', status: 'lagging', lag: 2.3, lag_bytes: 45056, replicationRate: 12.8 },
  { replicaId: 'replica-003', replicaName: 'MongoDB Replica Set', status: 'synced', lag: 0.05, lag_bytes: 1024, replicationRate: 18.5 },
]

const generateMockIndexMetrics = (): IndexMetric[] => [
  { name: 'idx_users_email', size: 125, scanCount: 4580, unused: false },
  { name: 'idx_orders_customer_id', size: 248, scanCount: 8920, unused: false },
  { name: 'idx_events_timestamp', size: 412, scanCount: 15420, unused: false },
  { name: 'idx_old_logs_date', size: 84, scanCount: 12, unused: true },
  { name: 'idx_archive_status', size: 32, scanCount: 8, unused: true },
]

const generateMockDatabaseSizeDistribution = () => [
  { name: 'Tables', value: 68, color: '#00D9FF' },
  { name: 'Indexes', value: 22, color: '#7C3AED' },
  { name: 'Logs', value: 8, color: '#EC4899' },
  { name: 'Temp', value: 2, color: '#F59E0B' },
]

export default function DatabasePage() {
  const [databases, setDatabases] = useState<Database[]>([])
  const [queryMetrics, setQueryMetrics] = useState<QueryMetric[]>([])
  const [connectionPools, setConnectionPools] = useState<ConnectionPool[]>([])
  const [slowQueries, setSlowQueries] = useState<SlowQuery[]>([])
  const [replicationStatus, setReplicationStatus] = useState<ReplicationStatus[]>([])
  const [indexMetrics, setIndexMetrics] = useState<IndexMetric[]>([])
  const [sizeDistribution, setSizeDistribution] = useState<any[]>([])
  const [expandedDB, setExpandedDB] = useState<string | null>(null)
  const [expandedQuery, setExpandedQuery] = useState<string | null>(null)

  useEffect(() => {
    setDatabases(generateMockDatabases())
    setQueryMetrics(generateMockQueryMetrics())
    setConnectionPools(generateMockConnectionPools())
    setSlowQueries(generateMockSlowQueries())
    setReplicationStatus(generateMockReplicationStatus())
    setIndexMetrics(generateMockIndexMetrics())
    setSizeDistribution(generateMockDatabaseSizeDistribution())
  }, [])

  const totalConnections = connectionPools.reduce((sum, pool) => sum + pool.active + pool.idle, 0)
  const avgQueryTime = (queryMetrics.reduce((sum, m) => sum + m.avgTime, 0) / queryMetrics.length).toFixed(1)
  const totalQueriesPerHour = queryMetrics[queryMetrics.length - 1]?.count || 0
  const totalDatabaseSize = databases.reduce((sum, db) => sum + db.size, 0)

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
      case 'synced':
        return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'warning':
      case 'lagging':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'critical':
      case 'disconnected':
        return 'bg-red-500/20 text-red-400 border-red-500/50'
      default:
        return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          {/* Header */}
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Database Management</h1>
            <p className="text-neutral-400">Monitor database health, query performance, replication, and indexes</p>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-5 gap-4">
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Databases</div>
              <div className="text-3xl font-bold text-white mb-1">{databases.length}</div>
              <div className="text-xs text-green-400">{databases.filter(d => d.status === 'healthy').length} healthy</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Total Size</div>
              <div className="text-3xl font-bold text-white mb-1">{totalDatabaseSize.toFixed(1)} GB</div>
              <div className="text-xs text-neutral-400">All databases</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Avg Query Time</div>
              <div className="text-3xl font-bold text-white mb-1">{avgQueryTime}ms</div>
              <div className="text-xs text-green-400">Within SLA</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Active Connections</div>
              <div className="text-3xl font-bold text-white mb-1">{connectionPools.reduce((sum, p) => sum + p.active, 0)}</div>
              <div className="text-xs text-neutral-400">{totalConnections} total</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Queries/Hour</div>
              <div className="text-3xl font-bold text-white mb-1">{totalQueriesPerHour.toLocaleString()}</div>
              <div className="text-xs text-neutral-400">Last recorded</div>
            </Card>
          </div>

          {/* Query Performance */}
          <div className="grid grid-cols-2 gap-4">
            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Query Performance (24h)</h2>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={queryMetrics}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis dataKey="time" stroke="#999" />
                  <YAxis stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                  <Legend />
                  <Line type="monotone" dataKey="avgTime" stroke="#00D9FF" dot={false} name="Avg Time (ms)" />
                  <Line type="monotone" dataKey="maxTime" stroke="#EF4444" dot={false} name="Max Time (ms)" />
                </LineChart>
              </ResponsiveContainer>
            </Card>

            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Query Volume (24h)</h2>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={queryMetrics}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis dataKey="time" stroke="#999" />
                  <YAxis stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                  <Bar dataKey="count" fill="#7C3AED" name="Query Count" />
                </BarChart>
              </ResponsiveContainer>
            </Card>
          </div>

          {/* Databases */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Database Instances</h2>
            <div className="space-y-3">
              {databases.map((db) => (
                <div key={db.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <div className="text-white font-semibold">{db.name}</div>
                      <div className="text-neutral-400 text-sm capitalize">{db.type}</div>
                    </div>
                    <Badge className={getStatusColor(db.status)}>
                      {db.status.charAt(0).toUpperCase() + db.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-4 gap-4 mb-3">
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Size</div>
                      <div className="text-white font-semibold">{db.size} GB</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Connections</div>
                      <div className="text-white font-semibold">{db.connections}/{db.maxConnections}</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Avg Query Time</div>
                      <div className="text-white font-semibold">{db.queryTime}ms</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Uptime</div>
                      <div className="text-white font-semibold">{db.uptime}%</div>
                    </div>
                  </div>
                  <div className="text-xs text-neutral-400">
                    Replication Lag: {db.replicationLag}ms
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Connection Pools */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Connection Pools</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Pool Name</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Type</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Active</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Idle</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Waiting</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Utilization</th>
                  </tr>
                </thead>
                <tbody>
                  {connectionPools.map((pool) => {
                    const utilization = ((pool.active / pool.maxSize) * 100).toFixed(1)
                    return (
                      <tr key={pool.id} className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white">{pool.name}</td>
                        <td className="py-3 px-4 text-neutral-300 capitalize">{pool.type}</td>
                        <td className="py-3 px-4 text-white font-semibold">{pool.active}</td>
                        <td className="py-3 px-4 text-neutral-300">{pool.idle}</td>
                        <td className="py-3 px-4 text-neutral-300">{pool.waiting}</td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <div className="w-32 bg-neutral-800 rounded-full h-2">
                              <div
                                className="h-2 rounded-full bg-primary-500"
                                style={{ width: `${utilization}%` }}
                              />
                            </div>
                            <span className="text-neutral-300 text-sm">{utilization}%</span>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </Card>

          {/* Slow Queries */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Slow Queries</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Query</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Exec Time (ms)</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Call Count</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Status</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Last Executed</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {slowQueries.map((query) => (
                    <React.Fragment key={query.id}>
                      <tr className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white font-mono text-sm">{query.query.substring(0, 40)}...</td>
                        <td className="py-3 px-4 text-white font-semibold">{query.executionTime}</td>
                        <td className="py-3 px-4 text-neutral-300">{query.callCount.toLocaleString()}</td>
                        <td className="py-3 px-4">
                          <Badge className={query.status === 'optimized' ? 'bg-green-500/20 text-green-400 border-green-500/50' : query.status === 'optimizing' ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50' : 'bg-red-500/20 text-red-400 border-red-500/50'}>
                            {query.status.charAt(0).toUpperCase() + query.status.slice(1)}
                          </Badge>
                        </td>
                        <td className="py-3 px-4 text-neutral-300 text-sm">{new Date(query.lastExecuted).toLocaleTimeString()}</td>
                        <td className="py-3 px-4">
                          <button
                            onClick={() => setExpandedQuery(expandedQuery === query.id ? null : query.id)}
                            className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors"
                          >
                            {expandedQuery === query.id ? 'Hide' : 'Analyze'}
                          </button>
                        </td>
                      </tr>
                      {expandedQuery === query.id && (
                        <tr className="border-b border-neutral-800 bg-neutral-900/30">
                          <td colSpan={6} className="py-4 px-4">
                            <div className="grid grid-cols-2 gap-4 text-sm">
                              <div>
                                <div className="text-neutral-400 mb-2">Full Query</div>
                                <div className="text-white font-mono bg-neutral-950 p-2 rounded">{query.query}</div>
                              </div>
                              <div className="text-right">
                                <button className="px-3 py-1.5 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 text-sm font-medium transition-colors">
                                  Create Index
                                </button>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>

          {/* Replication & Indexes */}
          <div className="grid grid-cols-2 gap-4">
            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Replication Status</h2>
              <div className="space-y-3">
                {replicationStatus.map((replica) => (
                  <div key={replica.replicaId} className="p-3 bg-neutral-900 rounded-lg border border-neutral-700">
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <div className="text-white font-semibold">{replica.replicaName}</div>
                        <div className="text-neutral-400 text-sm">Lag: {replica.lag.toFixed(2)}ms</div>
                      </div>
                      <Badge className={getStatusColor(replica.status)}>
                        {replica.status.charAt(0).toUpperCase() + replica.status.slice(1)}
                      </Badge>
                    </div>
                    <div className="flex justify-between text-xs text-neutral-400">
                      <div>Rate: {replica.replicationRate} MB/s</div>
                      <div>Bytes: {(replica.lag_bytes / 1024).toFixed(2)} KB</div>
                    </div>
                  </div>
                ))}
              </div>
            </Card>

            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Index Statistics</h2>
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {indexMetrics.map((index) => (
                  <div key={index.name} className="p-3 bg-neutral-900 rounded-lg border border-neutral-700">
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <div className={`font-semibold ${index.unused ? 'text-yellow-400' : 'text-white'}`}>
                          {index.name}
                        </div>
                        <div className="text-neutral-400 text-sm">{index.size} MB</div>
                      </div>
                      {index.unused && (
                        <Badge className="bg-yellow-500/20 text-yellow-400 border-yellow-500/50">
                          Unused
                        </Badge>
                      )}
                    </div>
                    <div className="text-xs text-neutral-400">
                      Scans: {index.scanCount.toLocaleString()}
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          </div>

          {/* Database Size Distribution */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Storage Distribution</h2>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie data={sizeDistribution} cx="50%" cy="50%" outerRadius={80} dataKey="value">
                  {sizeDistribution.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </div>
      </div>
    </div>
  )
}
