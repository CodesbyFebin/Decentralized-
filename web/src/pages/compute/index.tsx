import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { AreaChart } from '@/components/Charts/AreaChart'
import { BarChart } from '@/components/Charts/BarChart'
import { LineChart } from '@/components/Charts/LineChart'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'
import { Node } from '@/types'

interface WorkloadData {
  nodeType: string
  count: number
  cpuAllocated: number
  cpuUsed: number
  memoryAllocated: number
  memoryUsed: number
}

const Compute: React.FC = () => {
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const generateTimeSeriesData = (hours = 24) => {
    return Array.from({ length: hours }, (_, i) => ({
      timestamp: `${i}:00`,
      cpu: Math.random() * 80 + 20,
      memory: Math.random() * 75 + 25,
      disk: Math.random() * 70 + 30,
    }))
  }

  const workloadByType: WorkloadData[] = [
    {
      nodeType: 'Master',
      count: 3,
      cpuAllocated: 192,
      cpuUsed: 120,
      memoryAllocated: 384,
      memoryUsed: 240,
    },
    {
      nodeType: 'Worker',
      count: 12,
      cpuAllocated: 480,
      cpuUsed: 360,
      memoryAllocated: 960,
      memoryUsed: 650,
    },
    {
      nodeType: 'Edge',
      count: 8,
      cpuAllocated: 256,
      cpuUsed: 150,
      memoryAllocated: 512,
      memoryUsed: 280,
    },
  ]

  const timeSeriesData = generateTimeSeriesData()

  const chartData = workloadByType.map((w) => ({
    name: w.nodeType,
    allocated: w.cpuAllocated,
    used: w.cpuUsed,
  }))

  const totalAllocatedCPU = workloadByType.reduce((sum, w) => sum + w.cpuAllocated, 0)
  const totalUsedCPU = workloadByType.reduce((sum, w) => sum + w.cpuUsed, 0)
  const totalAllocatedMemory = workloadByType.reduce(
    (sum, w) => sum + w.memoryAllocated,
    0
  )
  const totalUsedMemory = workloadByType.reduce((sum, w) => sum + w.memoryUsed, 0)
  const totalNodes = workloadByType.reduce((sum, w) => sum + w.count, 0)

  useEffect(() => {
    const loadNodes = async () => {
      try {
        setLoading(true)
        const res = await apiClient.get<Node[]>('/nodes?limit=100')
        if (res.success && res.data) {
          setNodes(res.data)
        }
        setError(null)
      } catch (err) {
        setError('Failed to load nodes')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadNodes()
  }, [])

  if (error) {
    return (
      <AppLayout title="Compute Resources" subtitle="Resource allocation and workloads">
        <ErrorState
          title="Failed to Load Compute Data"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Compute Resources" subtitle="Resource allocation and workloads">
        <Loading message="Loading compute resources..." />
      </AppLayout>
    )
  }

  return (
    <AppLayout
      title="Compute Resources"
      subtitle={`${totalNodes} nodes • ${totalAllocatedCPU} CPU cores • ${totalAllocatedMemory} GB memory`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Nodes</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{totalNodes}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">
                {workloadByType.reduce((sum, w) => sum + w.count, 0)} active
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">CPU Utilization</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">
                  {((totalUsedCPU / totalAllocatedCPU) * 100).toFixed(0)}%
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">
                {totalUsedCPU}/{totalAllocatedCPU} cores used
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Memory Utilization</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">
                  {((totalUsedMemory / totalAllocatedMemory) * 100).toFixed(0)}%
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">
                {totalUsedMemory}/{totalAllocatedMemory} GB used
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Utilization Trend</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">68%</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">24h average</p>
            </div>
          </Card>
        </div>

        {/* Distribution by Node Type */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Workload Distribution Table */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">Workload Distribution</h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="border-b border-neutral-700">
                  <tr>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      Type
                    </th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      Count
                    </th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      CPU
                    </th>
                    <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                      Memory
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-700">
                  {workloadByType.map((w) => (
                    <tr key={w.nodeType} className="hover:bg-neutral-800/30">
                      <td className="px-6 py-3 text-neutral-100 font-medium">
                        {w.nodeType}
                      </td>
                      <td className="px-6 py-3 text-neutral-400">{w.count} nodes</td>
                      <td className="px-6 py-3 text-neutral-400">
                        <span className="text-primary-400">
                          {w.cpuUsed}/{w.cpuAllocated}
                        </span>
                        <span className="text-xs text-neutral-500 ml-1">
                          ({((w.cpuUsed / w.cpuAllocated) * 100).toFixed(0)}%)
                        </span>
                      </td>
                      <td className="px-6 py-3 text-neutral-400">
                        <span className="text-secondary-400">
                          {w.memoryUsed}/{w.memoryAllocated} GB
                        </span>
                        <span className="text-xs text-neutral-500 ml-1">
                          ({((w.memoryUsed / w.memoryAllocated) * 100).toFixed(0)}%)
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>

          {/* CPU Allocation Chart */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">CPU Allocation vs Usage</h3>
            </div>
            <div className="p-6">
              <BarChart
                data={chartData}
                dataKeys={[
                  { key: 'allocated', name: 'Allocated', fill: '#7C3AED' },
                  { key: 'used', name: 'Used', fill: '#00D9FF' },
                ]}
                height={250}
                xAxisKey="name"
              />
            </div>
          </Card>
        </div>

        {/* Resource Trends */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* CPU Trend */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">
                Cluster CPU Usage (24h)
              </h3>
            </div>
            <div className="p-6">
              <AreaChart
                data={timeSeriesData}
                dataKey="cpu"
                name="CPU %"
                fill="#00D9FF"
                stroke="#00D9FF"
                height={250}
                xAxisKey="timestamp"
              />
            </div>
          </Card>

          {/* Memory Trend */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">
                Cluster Memory Usage (24h)
              </h3>
            </div>
            <div className="p-6">
              <AreaChart
                data={timeSeriesData}
                dataKey="memory"
                name="Memory %"
                fill="#7C3AED"
                stroke="#7C3AED"
                height={250}
                xAxisKey="timestamp"
              />
            </div>
          </Card>
        </div>

        {/* Node Distribution */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Resource Distribution</h3>
          </div>
          <div className="p-6">
            <LineChart
              data={timeSeriesData}
              dataKey="disk"
              name="Disk %"
              stroke="#F59E0B"
              height={250}
              xAxisKey="timestamp"
            />
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Compute
