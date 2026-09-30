import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

interface BackupJob {
  id: string
  name: string
  resource: string
  type: 'full' | 'incremental' | 'differential'
  size: number
  status: 'completed' | 'running' | 'failed' | 'scheduled'
  lastRun: string
  nextRun: string
  duration: number
  retentionDays: number
  successRate: number
}

interface BackupSchedule {
  id: string
  resource: string
  frequency: 'hourly' | 'daily' | 'weekly' | 'monthly'
  time: string
  timezone: string
  enabled: boolean
  lastExecution: string
}

interface RecoveryPoint {
  id: string
  timestamp: string
  resource: string
  size: number
  type: 'full' | 'incremental'
  status: 'available' | 'archived' | 'expired'
  rpo: number
  rto: number
}

interface BackupStatistic {
  date: string
  succeeded: number
  failed: number
  dataSize: number
}

interface StorageBreakdown {
  category: string
  size: number
  percentage: number
}

const generateMockBackupJobs = (): BackupJob[] => [
  {
    id: 'job-001',
    name: 'Database Full Backup',
    resource: 'PostgreSQL Primary',
    type: 'full',
    size: 482.5,
    status: 'completed',
    lastRun: '2026-09-30T02:00:00Z',
    nextRun: '2026-10-01T02:00:00Z',
    duration: 45,
    retentionDays: 30,
    successRate: 100,
  },
  {
    id: 'job-002',
    name: 'Application Data Backup',
    resource: 'App Storage Bucket',
    type: 'incremental',
    size: 156.8,
    status: 'completed',
    lastRun: '2026-09-30T06:00:00Z',
    nextRun: '2026-09-30T12:00:00Z',
    duration: 18,
    retentionDays: 14,
    successRate: 100,
  },
  {
    id: 'job-003',
    name: 'Log Archive',
    resource: 'Log Storage',
    type: 'incremental',
    size: 92.3,
    status: 'running',
    lastRun: '2026-09-30T00:00:00Z',
    nextRun: '2026-09-30T12:00:00Z',
    duration: 12,
    retentionDays: 90,
    successRate: 100,
  },
  {
    id: 'job-004',
    name: 'Configuration Backup',
    resource: 'System Configuration',
    type: 'full',
    size: 8.7,
    status: 'completed',
    lastRun: '2026-09-30T01:00:00Z',
    nextRun: '2026-10-01T01:00:00Z',
    duration: 5,
    retentionDays: 365,
    successRate: 100,
  },
  {
    id: 'job-005',
    name: 'Cache Data Backup',
    resource: 'Redis Cluster',
    type: 'differential',
    size: 34.2,
    status: 'failed',
    lastRun: '2026-09-29T22:00:00Z',
    nextRun: '2026-09-30T22:00:00Z',
    duration: 8,
    retentionDays: 7,
    successRate: 92,
  },
]

const generateMockRecoveryPoints = (): RecoveryPoint[] => [
  { id: 'rp-001', timestamp: '2026-09-30T02:00:00Z', resource: 'PostgreSQL Primary', size: 482.5, type: 'full', status: 'available', rpo: 6, rto: 30 },
  { id: 'rp-002', timestamp: '2026-09-29T20:00:00Z', resource: 'PostgreSQL Primary', size: 125.3, type: 'incremental', status: 'available', rpo: 6, rto: 30 },
  { id: 'rp-003', timestamp: '2026-09-29T14:00:00Z', resource: 'PostgreSQL Primary', size: 98.7, type: 'incremental', status: 'available', rpo: 6, rto: 30 },
  { id: 'rp-004', timestamp: '2026-09-30T06:00:00Z', resource: 'App Storage Bucket', size: 156.8, type: 'incremental', status: 'available', rpo: 12, rto: 60 },
  { id: 'rp-005', timestamp: '2026-09-29T06:00:00Z', resource: 'App Storage Bucket', size: 142.1, type: 'incremental', status: 'available', rpo: 12, rto: 60 },
]

const generateMockBackupStats = (): BackupStatistic[] => [
  { date: 'Sep 24', succeeded: 8, failed: 0, dataSize: 742 },
  { date: 'Sep 25', succeeded: 8, failed: 1, dataSize: 758 },
  { date: 'Sep 26', succeeded: 8, failed: 0, dataSize: 775 },
  { date: 'Sep 27', succeeded: 8, failed: 0, dataSize: 792 },
  { date: 'Sep 28', succeeded: 8, failed: 0, dataSize: 810 },
  { date: 'Sep 29', succeeded: 8, failed: 1, dataSize: 829 },
  { date: 'Sep 30', succeeded: 7, failed: 1, dataSize: 848 },
]

const generateMockStorageBreakdown = (): StorageBreakdown[] => [
  { category: 'Database Backups', size: 1425, percentage: 42 },
  { category: 'Application Data', size: 890, percentage: 26 },
  { category: 'Log Archives', size: 720, percentage: 21 },
  { category: 'Configuration', size: 245, percentage: 7 },
  { category: 'Cache Data', size: 140, percentage: 4 },
]

export default function BackupsPage() {
  const [backupJobs, setBackupJobs] = useState<BackupJob[]>([])
  const [recoveryPoints, setRecoveryPoints] = useState<RecoveryPoint[]>([])
  const [backupStats, setBackupStats] = useState<BackupStatistic[]>([])
  const [storageBreakdown, setStorageBreakdown] = useState<StorageBreakdown[]>([])
  const [expandedJob, setExpandedJob] = useState<string | null>(null)
  const [expandedRP, setExpandedRP] = useState<string | null>(null)

  useEffect(() => {
    setBackupJobs(generateMockBackupJobs())
    setRecoveryPoints(generateMockRecoveryPoints())
    setBackupStats(generateMockBackupStats())
    setStorageBreakdown(generateMockStorageBreakdown())
  }, [])

  const totalBackupSize = backupJobs.reduce((sum, job) => sum + job.size, 0)
  const successRate = (
    backupJobs.reduce((sum, job) => sum + job.successRate, 0) / backupJobs.length
  ).toFixed(1)
  const runningJobs = backupJobs.filter(j => j.status === 'running').length
  const failedJobs = backupJobs.filter(j => j.status === 'failed').length
  const totalRecoveryPoints = recoveryPoints.length

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
      case 'available':
        return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'running':
        return 'bg-blue-500/20 text-blue-400 border-blue-500/50'
      case 'failed':
      case 'expired':
        return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'archived':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'scheduled':
        return 'bg-purple-500/20 text-purple-400 border-purple-500/50'
      default:
        return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'full':
        return 'text-blue-400'
      case 'incremental':
        return 'text-green-400'
      case 'differential':
        return 'text-purple-400'
      default:
        return 'text-neutral-400'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          {/* Header */}
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Backup Management</h1>
            <p className="text-neutral-400">Manage backup jobs, recovery points, and backup strategy</p>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-5 gap-4">
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Backup Jobs</div>
              <div className="text-3xl font-bold text-white mb-1">{backupJobs.length}</div>
              <div className="text-xs text-neutral-400">{runningJobs} running, {failedJobs} failed</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Total Backup Size</div>
              <div className="text-3xl font-bold text-white mb-1">{totalBackupSize.toFixed(1)} GB</div>
              <div className="text-xs text-green-400">↑ 2.3% this week</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Success Rate</div>
              <div className="text-3xl font-bold text-white mb-1">{successRate}%</div>
              <div className="text-xs text-neutral-400">7-day average</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Recovery Points</div>
              <div className="text-3xl font-bold text-white mb-1">{totalRecoveryPoints}</div>
              <div className="text-xs text-neutral-400">Available for restore</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Avg Recovery Time</div>
              <div className="text-3xl font-bold text-white mb-1">48 min</div>
              <div className="text-xs text-neutral-400">RTO across resources</div>
            </Card>
          </div>

          {/* Backup Trend */}
          <div className="grid grid-cols-2 gap-4">
            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Backup Jobs Status (7 Days)</h2>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={backupStats}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis dataKey="date" stroke="#999" />
                  <YAxis stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                  <Legend />
                  <Bar dataKey="succeeded" stackId="a" fill="#00D9FF" name="Successful" />
                  <Bar dataKey="failed" stackId="a" fill="#EF4444" name="Failed" />
                </BarChart>
              </ResponsiveContainer>
            </Card>

            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Backup Data Size Trend</h2>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={backupStats}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis dataKey="date" stroke="#999" />
                  <YAxis stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                  <Line type="monotone" dataKey="dataSize" stroke="#7C3AED" dot={false} name="Total Size (GB)" />
                </LineChart>
              </ResponsiveContainer>
            </Card>
          </div>

          {/* Backup Jobs */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Backup Jobs</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Job Name</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Resource</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Type</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Size</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Status</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Last Run</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Next Run</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {backupJobs.map((job) => (
                    <React.Fragment key={job.id}>
                      <tr className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white">{job.name}</td>
                        <td className="py-3 px-4 text-neutral-300">{job.resource}</td>
                        <td className="py-3 px-4">
                          <span className={`capitalize font-semibold ${getTypeColor(job.type)}`}>{job.type}</span>
                        </td>
                        <td className="py-3 px-4 text-white font-semibold">{job.size} GB</td>
                        <td className="py-3 px-4">
                          <Badge className={getStatusColor(job.status)}>
                            {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
                          </Badge>
                        </td>
                        <td className="py-3 px-4 text-neutral-300 text-sm">{new Date(job.lastRun).toLocaleDateString()}</td>
                        <td className="py-3 px-4 text-neutral-300 text-sm">{new Date(job.nextRun).toLocaleDateString()}</td>
                        <td className="py-3 px-4">
                          <button
                            onClick={() => setExpandedJob(expandedJob === job.id ? null : job.id)}
                            className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors"
                          >
                            {expandedJob === job.id ? 'Hide' : 'Details'}
                          </button>
                        </td>
                      </tr>
                      {expandedJob === job.id && (
                        <tr className="border-b border-neutral-800 bg-neutral-900/30">
                          <td colSpan={8} className="py-4 px-4">
                            <div className="grid grid-cols-4 gap-4 text-sm">
                              <div>
                                <div className="text-neutral-400 mb-1">Duration</div>
                                <div className="text-white">{job.duration} minutes</div>
                              </div>
                              <div>
                                <div className="text-neutral-400 mb-1">Retention</div>
                                <div className="text-white">{job.retentionDays} days</div>
                              </div>
                              <div>
                                <div className="text-neutral-400 mb-1">Success Rate</div>
                                <div className="text-white">{job.successRate}%</div>
                              </div>
                              <div className="text-right">
                                <button className="px-3 py-1.5 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 text-sm font-medium transition-colors">
                                  Run Now
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

          {/* Recovery Points */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Recovery Points</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Timestamp</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Resource</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Type</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Size</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Status</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">RPO</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">RTO</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {recoveryPoints.map((rp) => (
                    <React.Fragment key={rp.id}>
                      <tr className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white text-sm">{new Date(rp.timestamp).toLocaleString()}</td>
                        <td className="py-3 px-4 text-neutral-300">{rp.resource}</td>
                        <td className="py-3 px-4">
                          <span className={`capitalize font-semibold ${getTypeColor(rp.type)}`}>{rp.type}</span>
                        </td>
                        <td className="py-3 px-4 text-white font-semibold">{rp.size} GB</td>
                        <td className="py-3 px-4">
                          <Badge className={getStatusColor(rp.status)}>
                            {rp.status.charAt(0).toUpperCase() + rp.status.slice(1)}
                          </Badge>
                        </td>
                        <td className="py-3 px-4 text-neutral-300">{rp.rpo}h</td>
                        <td className="py-3 px-4 text-neutral-300">{rp.rto}m</td>
                        <td className="py-3 px-4">
                          <button
                            onClick={() => setExpandedRP(expandedRP === rp.id ? null : rp.id)}
                            className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors"
                          >
                            {expandedRP === rp.id ? 'Hide' : 'Restore'}
                          </button>
                        </td>
                      </tr>
                      {expandedRP === rp.id && (
                        <tr className="border-b border-neutral-800 bg-neutral-900/30">
                          <td colSpan={8} className="py-4 px-4">
                            <div className="flex justify-between items-center text-sm">
                              <div>
                                <div className="text-neutral-400 mb-1">Verify Recovery Point Integrity?</div>
                                <div className="text-white">This recovery point is available for restoration</div>
                              </div>
                              <div className="flex gap-2">
                                <button className="px-3 py-1.5 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 font-medium transition-colors">
                                  Verify
                                </button>
                                <button className="px-3 py-1.5 bg-green-500/20 text-green-400 rounded hover:bg-green-500/30 font-medium transition-colors">
                                  Restore
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

          {/* Storage Breakdown */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Backup Storage Breakdown</h2>
            <div className="space-y-3">
              {storageBreakdown.map((category) => (
                <div key={category.category}>
                  <div className="flex justify-between mb-2">
                    <span className="text-neutral-300">{category.category}</span>
                    <div className="flex gap-4">
                      <span className="text-white font-semibold">{category.size} GB</span>
                      <span className="text-primary-400">{category.percentage}%</span>
                    </div>
                  </div>
                  <div className="w-full bg-neutral-800 rounded-full h-2">
                    <div
                      className="h-2 rounded-full bg-primary-500"
                      style={{ width: `${category.percentage}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
            <div className="mt-6 p-4 bg-neutral-900 rounded-lg border border-neutral-700">
              <div className="flex justify-between items-center">
                <div>
                  <div className="text-neutral-400 text-sm mb-1">Total Storage Used</div>
                  <div className="text-2xl font-bold text-white">
                    {storageBreakdown.reduce((sum, cat) => sum + cat.size, 0)} GB
                  </div>
                </div>
                <button className="px-4 py-2 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 font-medium transition-colors">
                  Optimize Storage
                </button>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
