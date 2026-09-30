import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { Button } from '@/components/Button'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { LineChart } from '@/components/Charts/LineChart'
import { NodeCreationWizard, NodeConfig } from '@/components/NodeCreationWizard'
import { apiClient } from '@/lib/api'
import { filterNodes, FilterOptions, SortBy, SortOrder, StatusFilter } from '@/lib/nodeFilters'
import { Node } from '@/types'

interface Container {
  id: string
  name: string
  status: 'running' | 'stopped' | 'error'
  cpu: number
  memory: number
  uptime: string
}

const Nodes: React.FC = () => {
  const [nodes, setNodes] = useState<Node[]>([])
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedNodes, setSelectedNodes] = useState<Set<string>>(new Set())
  const [showWizard, setShowWizard] = useState(false)
  const [filterOptions, setFilterOptions] = useState<FilterOptions>({
    status: 'all',
    sortBy: 'name',
    sortOrder: 'asc',
    search: '',
  })

  const generateNodeMetrics = (hours = 24) => {
    return Array.from({ length: hours }, (_, i) => ({
      timestamp: `${i}:00`,
      cpu: Math.random() * 100,
      memory: Math.random() * 100,
      disk: Math.random() * 100,
    }))
  }

  const generateContainers = (nodeId: string, count = 5): Container[] => {
    return Array.from({ length: count }, (_, i) => ({
      id: `${nodeId}-container-${i}`,
      name: `container-${i}`,
      status: ['running', 'stopped', 'error'][Math.floor(Math.random() * 3)] as any,
      cpu: Math.random() * 50,
      memory: Math.random() * 60,
      uptime: `${Math.floor(Math.random() * 30) + 1}d`,
    }))
  }

  const nodeMetrics = selectedNode ? generateNodeMetrics() : []
  const containers = selectedNode ? generateContainers(selectedNode.id) : []
  const filteredNodes = filterNodes(nodes, filterOptions)

  useEffect(() => {
    const loadNodes = async () => {
      try {
        setLoading(true)
        const res = await apiClient.get<Node[]>('/nodes?limit=50')
        if (res.success && res.data) {
          setNodes(res.data)
          if (res.data.length > 0 && !selectedNode) {
            setSelectedNode(res.data[0])
          }
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

  const handleToggleNode = (nodeId: string) => {
    const newSelected = new Set(selectedNodes)
    if (newSelected.has(nodeId)) {
      newSelected.delete(nodeId)
    } else {
      newSelected.add(nodeId)
    }
    setSelectedNodes(newSelected)
  }

  const handleBatchDrain = async () => {
    if (selectedNodes.size === 0) return
    try {
      for (const nodeId of selectedNodes) {
        await apiClient.post(`/nodes/${nodeId}/drain`, {})
      }
      setSelectedNodes(new Set())
    } catch (err) {
      console.error('Failed to drain nodes:', err)
    }
  }

  const handleBatchRestart = async () => {
    if (selectedNodes.size === 0) return
    try {
      for (const nodeId of selectedNodes) {
        await apiClient.post(`/nodes/${nodeId}/restart`, {})
      }
      setSelectedNodes(new Set())
    } catch (err) {
      console.error('Failed to restart nodes:', err)
    }
  }

  const handleCreateNode = async (config: NodeConfig) => {
    try {
      await apiClient.post('/nodes', config)
      const res = await apiClient.get<Node[]>('/nodes?limit=50')
      if (res.success && res.data) {
        setNodes(res.data)
      }
    } catch (err) {
      console.error('Failed to create node:', err)
    }
  }

  if (error) {
    return (
      <AppLayout title="Nodes & Compute" subtitle="Infrastructure nodes">
        <ErrorState
          title="Failed to Load Nodes"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Nodes & Compute" subtitle="Infrastructure nodes">
        <Loading message="Loading nodes..." />
      </AppLayout>
    )
  }

  return (
    <AppLayout
      title="Nodes & Compute"
      subtitle={`${nodes.length} nodes in cluster • ${selectedNodes.size} selected`}
    >
      <NodeCreationWizard
        isOpen={showWizard}
        onClose={() => setShowWizard(false)}
        onSubmit={handleCreateNode}
      />

      <div className="space-y-6">
        {/* Controls Bar */}
        <div className="flex flex-col gap-4">
          <div className="flex gap-3">
            <Button
              variant="primary"
              size="md"
              onClick={() => setShowWizard(true)}
            >
              + Create Node
            </Button>
            {selectedNodes.size > 0 && (
              <>
                <Button
                  variant="secondary"
                  size="md"
                  onClick={handleBatchDrain}
                >
                  Drain ({selectedNodes.size})
                </Button>
                <Button
                  variant="ghost"
                  size="md"
                  onClick={handleBatchRestart}
                >
                  Restart ({selectedNodes.size})
                </Button>
              </>
            )}
          </div>

          {/* Filter and Sort Controls */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <input
              type="text"
              placeholder="Search nodes..."
              value={filterOptions.search}
              onChange={(e) =>
                setFilterOptions({ ...filterOptions, search: e.target.value })
              }
              className="px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
            />
            <select
              value={filterOptions.status}
              onChange={(e) =>
                setFilterOptions({
                  ...filterOptions,
                  status: e.target.value as StatusFilter,
                })
              }
              className="px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white focus:border-primary-500 focus:outline-none"
            >
              <option value="all">All Status</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
              <option value="error">Error</option>
            </select>
            <select
              value={filterOptions.sortBy}
              onChange={(e) =>
                setFilterOptions({
                  ...filterOptions,
                  sortBy: e.target.value as SortBy,
                })
              }
              className="px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white focus:border-primary-500 focus:outline-none"
            >
              <option value="name">Sort by Name</option>
              <option value="cpu">Sort by CPU</option>
              <option value="memory">Sort by Memory</option>
              <option value="disk">Sort by Disk</option>
              <option value="status">Sort by Status</option>
            </select>
            <button
              onClick={() =>
                setFilterOptions({
                  ...filterOptions,
                  sortOrder: filterOptions.sortOrder === 'asc' ? 'desc' : 'asc',
                })
              }
              className="px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white hover:bg-neutral-700 focus:border-primary-500 focus:outline-none"
            >
              {filterOptions.sortOrder === 'asc' ? '↑' : '↓'}
            </button>
          </div>
        </div>

        {/* Main Content */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Nodes List */}
          <Card variant="glass" className="lg:col-span-1">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">
                Nodes ({filteredNodes.length})
              </h3>
            </div>
            <div className="divide-y divide-neutral-700 max-h-[600px] overflow-y-auto">
              {filteredNodes.map((node) => (
                <div
                  key={node.id}
                  className={`p-4 transition-colors border-l-2 ${
                    selectedNode?.id === node.id
                      ? 'bg-primary-500/20 border-primary-500'
                      : 'border-transparent hover:bg-neutral-800/50'
                  }`}
                >
                  <div className="flex items-start gap-3">
                    <input
                      type="checkbox"
                      checked={selectedNodes.has(node.id)}
                      onChange={() => handleToggleNode(node.id)}
                      className="mt-1 cursor-pointer"
                    />
                    <button
                      onClick={() => setSelectedNode(node)}
                      className="flex-1 text-left"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <p className="font-medium text-neutral-100">{node.name}</p>
                        <Badge status={node.status} size="sm" />
                      </div>
                      <p className="text-xs text-neutral-400">
                        CPU: {node.cpu.percent.toFixed(0)}% • Mem:{' '}
                        {node.memory.percent.toFixed(0)}%
                      </p>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Selected Node Details */}
          <div className="lg:col-span-2 space-y-6">
            {selectedNode && (
              <>
                {/* Node Header */}
                <Card variant="glass">
                  <div className="p-6 border-b border-neutral-700">
                    <div className="flex items-start justify-between">
                      <div>
                        <h2 className="text-2xl font-bold text-white">
                          {selectedNode.name}
                        </h2>
                        <p className="text-neutral-400 mt-1">{selectedNode.id}</p>
                      </div>
                      <Badge status={selectedNode.status} />
                    </div>
                  </div>
                  <div className="p-6 space-y-4">
                    <div className="grid grid-cols-3 gap-4">
                      <div>
                        <p className="text-neutral-400 text-sm font-medium mb-1">Type</p>
                        <p className="text-white font-mono text-sm">
                          {selectedNode.nodeType}
                        </p>
                      </div>
                      <div>
                        <p className="text-neutral-400 text-sm font-medium mb-1">
                          Isolation
                        </p>
                        <p className="text-white font-mono text-sm">
                          {selectedNode.isolationBoundary}
                        </p>
                      </div>
                      <div>
                        <p className="text-neutral-400 text-sm font-medium mb-1">
                          Last Seen
                        </p>
                        <p className="text-white font-mono text-sm">
                          {new Date(selectedNode.lastSeen).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                  </div>
                </Card>

                {/* Resource Metrics Cards */}
                <div className="grid grid-cols-3 gap-4">
                  <Card variant="glass">
                    <div className="p-4">
                      <p className="text-neutral-400 text-sm mb-2">CPU</p>
                      <div className="flex items-end gap-2">
                        <span className="text-2xl font-bold text-primary-500">
                          {selectedNode.cpu.percent.toFixed(0)}%
                        </span>
                        <span className="text-xs text-neutral-400 mb-1">
                          {selectedNode.cpu.used}/{selectedNode.cpu.total} cores
                        </span>
                      </div>
                    </div>
                  </Card>
                  <Card variant="glass">
                    <div className="p-4">
                      <p className="text-neutral-400 text-sm mb-2">Memory</p>
                      <div className="flex items-end gap-2">
                        <span className="text-2xl font-bold text-secondary-500">
                          {selectedNode.memory.percent.toFixed(0)}%
                        </span>
                        <span className="text-xs text-neutral-400 mb-1">
                          {selectedNode.memory.used}/{selectedNode.memory.total} GB
                        </span>
                      </div>
                    </div>
                  </Card>
                  <Card variant="glass">
                    <div className="p-4">
                      <p className="text-neutral-400 text-sm mb-2">Disk</p>
                      <div className="flex items-end gap-2">
                        <span className="text-2xl font-bold text-warning-500">
                          {selectedNode.disk.percent.toFixed(0)}%
                        </span>
                        <span className="text-xs text-neutral-400 mb-1">
                          {selectedNode.disk.used}/{selectedNode.disk.total} GB
                        </span>
                      </div>
                    </div>
                  </Card>
                </div>

                {/* Resource Trends */}
                <Card variant="glass">
                  <div className="p-6 border-b border-neutral-700">
                    <h3 className="text-lg font-semibold text-white">
                      Resource Trends (24h)
                    </h3>
                  </div>
                  <div className="p-6">
                    <LineChart
                      data={nodeMetrics}
                      dataKey="cpu"
                      name="CPU %"
                      height={200}
                      xAxisKey="timestamp"
                    />
                  </div>
                </Card>

                {/* Containers Running */}
                <Card variant="glass">
                  <div className="p-6 border-b border-neutral-700">
                    <h3 className="text-lg font-semibold text-white">
                      Containers ({containers.length})
                    </h3>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead className="border-b border-neutral-700">
                        <tr>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                            Name
                          </th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                            Status
                          </th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                            CPU
                          </th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                            Memory
                          </th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">
                            Uptime
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-neutral-700">
                        {containers.map((container) => (
                          <tr key={container.id} className="hover:bg-neutral-800/30">
                            <td className="px-6 py-3 text-neutral-100 font-mono text-xs">
                              {container.name}
                            </td>
                            <td className="px-6 py-3">
                              <Badge
                                status={
                                  container.status === 'running'
                                    ? 'active'
                                    : container.status === 'error'
                                      ? 'error'
                                      : 'inactive'
                                }
                                size="sm"
                              />
                            </td>
                            <td className="px-6 py-3 text-neutral-400">
                              {container.cpu.toFixed(1)}%
                            </td>
                            <td className="px-6 py-3 text-neutral-400">
                              {container.memory.toFixed(1)}%
                            </td>
                            <td className="px-6 py-3 text-neutral-400">
                              {container.uptime}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Card>

                {/* Tags */}
                {selectedNode.tags && selectedNode.tags.length > 0 && (
                  <Card variant="glass">
                    <div className="p-6 border-b border-neutral-700">
                      <h3 className="text-lg font-semibold text-white">Tags</h3>
                    </div>
                    <div className="p-6 flex flex-wrap gap-2">
                      {selectedNode.tags.map((tag) => (
                        <span
                          key={tag}
                          className="px-3 py-1 bg-primary-500/20 text-primary-300 rounded-full text-sm font-medium border border-primary-500/30"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </Card>
                )}

                {/* Actions */}
                <div className="flex gap-3">
                  <Button variant="primary" size="md" className="flex-1">
                    View Logs
                  </Button>
                  <Button variant="secondary" size="md" className="flex-1">
                    Drain Node
                  </Button>
                  <Button variant="ghost" size="md" className="flex-1">
                    More Actions
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </AppLayout>
  )
}

export default Nodes
