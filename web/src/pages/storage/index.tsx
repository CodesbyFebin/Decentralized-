import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { BarChart } from '@/components/Charts/BarChart'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface StorageBucket {
  id: string
  name: string
  size: number
  objects: number
  status: 'active' | 'readonly' | 'error'
  created: string
  lastModified: string
  replicationStatus: 'synced' | 'syncing' | 'error'
}

interface Backup {
  id: string
  bucketId: string
  bucketName: string
  size: number
  timestamp: string
  type: 'full' | 'incremental'
  status: 'completed' | 'in_progress' | 'failed'
}

const Storage: React.FC = () => {
  const [buckets, setBuckets] = useState<StorageBucket[]>([])
  const [backups, setBackups] = useState<Backup[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreateBucket, setShowCreateBucket] = useState(false)
  const [newBucketName, setNewBucketName] = useState('')

  const generateMockBuckets = (): StorageBucket[] => [
    {
      id: 'bucket-1',
      name: 'app-data',
      size: 156.8,
      objects: 2840,
      status: 'active',
      created: '2026-09-01T10:30:00Z',
      lastModified: '2026-09-30T08:45:00Z',
      replicationStatus: 'synced',
    },
    {
      id: 'bucket-2',
      name: 'backups',
      size: 512.3,
      objects: 156,
      status: 'active',
      created: '2026-08-15T14:20:00Z',
      lastModified: '2026-09-30T06:15:00Z',
      replicationStatus: 'synced',
    },
    {
      id: 'bucket-3',
      name: 'logs',
      size: 89.5,
      objects: 12450,
      status: 'active',
      created: '2026-09-20T09:00:00Z',
      lastModified: '2026-09-30T09:55:00Z',
      replicationStatus: 'syncing',
    },
    {
      id: 'bucket-4',
      name: 'media',
      size: 2340.1,
      objects: 580,
      status: 'active',
      created: '2026-07-10T16:45:00Z',
      lastModified: '2026-09-30T07:20:00Z',
      replicationStatus: 'synced',
    },
  ]

  const generateMockBackups = (): Backup[] => [
    {
      id: 'backup-1',
      bucketId: 'bucket-1',
      bucketName: 'app-data',
      size: 156.2,
      timestamp: '2026-09-30T00:00:00Z',
      type: 'full',
      status: 'completed',
    },
    {
      id: 'backup-2',
      bucketId: 'bucket-1',
      bucketName: 'app-data',
      size: 12.4,
      timestamp: '2026-09-29T00:00:00Z',
      type: 'incremental',
      status: 'completed',
    },
    {
      id: 'backup-3',
      bucketId: 'bucket-2',
      bucketName: 'backups',
      size: 510.1,
      timestamp: '2026-09-28T00:00:00Z',
      type: 'full',
      status: 'completed',
    },
    {
      id: 'backup-4',
      bucketId: 'bucket-4',
      bucketName: 'media',
      size: 2200.5,
      timestamp: '2026-09-30T02:00:00Z',
      type: 'full',
      status: 'in_progress',
    },
  ]

  useEffect(() => {
    const loadStorage = async () => {
      try {
        setLoading(true)
        const mockBuckets = generateMockBuckets()
        const mockBackups = generateMockBackups()
        setBuckets(mockBuckets)
        setBackups(mockBackups)
        setError(null)
      } catch (err) {
        setError('Failed to load storage data')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadStorage()
  }, [])

  if (error) {
    return (
      <AppLayout title="Storage" subtitle="Distributed storage management">
        <ErrorState
          title="Failed to Load Storage"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Storage" subtitle="Distributed storage management">
        <Loading message="Loading storage data..." />
      </AppLayout>
    )
  }

  const totalSize = buckets.reduce((sum, b) => sum + b.size, 0)
  const totalObjects = buckets.reduce((sum, b) => sum + b.objects, 0)
  const activeBuckets = buckets.filter((b) => b.status === 'active').length
  const syncedBuckets = buckets.filter((b) => b.replicationStatus === 'synced').length

  const chartData = buckets.map((b) => ({
    name: b.name,
    size: parseFloat(b.size.toFixed(1)),
    objects: b.objects,
  }))

  const handleCreateBucket = async () => {
    if (!newBucketName.trim()) return
    try {
      await apiClient.post('/storage/buckets', { name: newBucketName })
      setNewBucketName('')
      setShowCreateBucket(false)
      const mockBuckets = generateMockBuckets()
      setBuckets(mockBuckets)
    } catch (err) {
      console.error('Failed to create bucket:', err)
    }
  }

  const handleRestoreBackup = async (backupId: string) => {
    try {
      await apiClient.post(`/storage/backups/${backupId}/restore`, {})
    } catch (err) {
      console.error('Failed to restore backup:', err)
    }
  }

  return (
    <AppLayout
      title="Storage"
      subtitle={`${activeBuckets} active buckets • ${(totalSize).toFixed(1)} GB total`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Buckets</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{buckets.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">{activeBuckets} active</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Size</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">
                  {totalSize.toFixed(1)}
                </span>
                <span className="text-xs text-neutral-400 mb-1">GB</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">Across all buckets</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Objects</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-warning-500">{totalObjects.toLocaleString()}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">In all buckets</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Replication</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">{syncedBuckets}/{buckets.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">Buckets synced</p>
            </div>
          </Card>
        </div>

        {/* Bucket Management */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
            <h3 className="text-lg font-semibold text-white">Buckets ({buckets.length})</h3>
            <Button
              variant="primary"
              size="md"
              onClick={() => setShowCreateBucket(!showCreateBucket)}
            >
              + Create Bucket
            </Button>
          </div>

          {showCreateBucket && (
            <div className="p-6 border-b border-neutral-700 bg-neutral-800/50 space-y-3">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Bucket Name
                </label>
                <input
                  type="text"
                  value={newBucketName}
                  onChange={(e) => setNewBucketName(e.target.value)}
                  placeholder="e.g., my-app-bucket"
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>
              <div className="flex gap-3">
                <Button
                  variant="primary"
                  size="md"
                  onClick={handleCreateBucket}
                  disabled={!newBucketName.trim()}
                  className="flex-1"
                >
                  Create
                </Button>
                <Button
                  variant="ghost"
                  size="md"
                  onClick={() => setShowCreateBucket(false)}
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
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Size</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Objects</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Replication</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {buckets.map((bucket) => (
                  <tr key={bucket.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{bucket.name}</td>
                    <td className="px-6 py-3 text-neutral-400">{bucket.size.toFixed(1)} GB</td>
                    <td className="px-6 py-3 text-neutral-400">
                      {bucket.objects.toLocaleString()}
                    </td>
                    <td className="px-6 py-3">
                      <Badge
                        status={bucket.status === 'active' ? 'active' : 'error'}
                        size="sm"
                      />
                    </td>
                    <td className="px-6 py-3">
                      <Badge
                        status={
                          bucket.replicationStatus === 'synced'
                            ? 'active'
                            : bucket.replicationStatus === 'syncing'
                              ? 'pending'
                              : 'error'
                        }
                        size="sm"
                      />
                    </td>
                    <td className="px-6 py-3 flex gap-2">
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        Backup
                      </button>
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Storage Distribution Chart */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Storage Distribution by Bucket</h3>
          </div>
          <div className="p-6">
            <BarChart
              data={chartData}
              dataKeys={[{ key: 'size', name: 'Size (GB)', fill: '#00D9FF' }]}
              height={250}
              xAxisKey="name"
            />
          </div>
        </Card>

        {/* Backups */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Backups ({backups.length})</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Bucket</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Date</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Size</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Type</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {backups.map((backup) => (
                  <tr key={backup.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{backup.bucketName}</td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                      {new Date(backup.timestamp).toLocaleString()}
                    </td>
                    <td className="px-6 py-3 text-neutral-400">{backup.size.toFixed(1)} GB</td>
                    <td className="px-6 py-3 text-neutral-400">
                      <span className="px-2 py-1 bg-neutral-700 rounded text-xs">
                        {backup.type === 'full' ? 'Full' : 'Incremental'}
                      </span>
                    </td>
                    <td className="px-6 py-3">
                      <Badge
                        status={
                          backup.status === 'completed'
                            ? 'active'
                            : backup.status === 'in_progress'
                              ? 'pending'
                              : 'error'
                        }
                        size="sm"
                      />
                    </td>
                    <td className="px-6 py-3 flex gap-2">
                      {backup.status === 'completed' && (
                        <button
                          onClick={() => handleRestoreBackup(backup.id)}
                          className="text-xs px-2 py-1 bg-primary-600 hover:bg-primary-500 rounded text-white"
                        >
                          Restore
                        </button>
                      )}
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        Details
                      </button>
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

export default Storage
