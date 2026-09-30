import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { Button } from '@/components/Button'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { LineChart } from '@/components/Charts/LineChart'
import { apiClient } from '@/lib/api'
import { Node } from '@/types'

const Nodes: React.FC = () => {
  const [nodes, setNodes] = useState<Node[]>([])
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Generate mock metrics for selected node
  const generateNodeMetrics = (hours = 24) => {
    return Array.from({ length: hours }, (_, i) => ({
      timestamp: `${i}:00`,
      cpu: Math.random() * 100,
      memory: Math.random() * 100,
      disk: Math.random() * 100,
    }))
  }

  const nodeMetrics = selectedNode ? generateNodeMetrics() : []

  useEffect(() => {
    const loadNodes = async () => {
      try {
        setLoading(true)
        const res = await apiClient.get<Node[]>('/nodes?limit=20')
        if (res.success && res.data) {
          setNodes(res.data)
          if (res.data.length > 0) {
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
      subtitle={`${nodes.length} nodes in cluster`}
    >
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Nodes List */}
        <Card variant="glass" className="lg:col-span-1">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">All Nodes</h3>
          </div>
          <div className="divide-y divide-neutral-700 max-h-[600px] overflow-y-auto">
            {nodes.map((node) => (
              <button
                key={node.id}
                onClick={() => setSelectedNode(node)}
                className={`w-full text-left p-4 transition-colors ${
                  selectedNode?.id === node.id
                    ? 'bg-primary-500/20 border-l-2 border-primary-500'
                    : 'hover:bg-neutral-800/50'
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <p className="font-medium text-neutral-100">{node.name}</p>
                  <Badge status={node.status} size="sm" />
                </div>
                <p className="text-xs text-neutral-400">
                  CPU: {node.cpu.percent.toFixed(0)}% • Mem: {node.memory.percent.toFixed(0)}%
                </p>
              </button>
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
                      <h2 className="text-2xl font-bold text-white">{selectedNode.name}</h2>
                      <p className="text-neutral-400 mt-1">{selectedNode.id}</p>
                    </div>
                    <Badge status={selectedNode.status} />
                  </div>
                </div>
                <div className="p-6 space-y-4">
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <p className="text-neutral-400 text-sm font-medium mb-1">Type</p>
                      <p className="text-white font-mono text-sm">{selectedNode.nodeType}</p>
                    </div>
                    <div>
                      <p className="text-neutral-400 text-sm font-medium mb-1">Isolation</p>
                      <p className="text-white font-mono text-sm">{selectedNode.isolationBoundary}</p>
                    </div>
                    <div>
                      <p className="text-neutral-400 text-sm font-medium mb-1">Last Seen</p>
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
                  <h3 className="text-lg font-semibold text-white">Resource Trends (24h)</h3>
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
    </AppLayout>
  )
}

export default Nodes
