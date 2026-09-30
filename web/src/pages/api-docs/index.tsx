import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'

interface APIEndpoint {
  id: string
  path: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  description: string
  authentication: 'bearer' | 'api-key' | 'none'
  rateLimit: number
  deprecated: boolean
  category: string
  parameters?: string
  requestBody?: string
  responseBody?: string
  status: 'stable' | 'beta' | 'deprecated'
}

interface APIKey {
  id: string
  name: string
  prefix: string
  lastUsed: string
  created: string
  expiresAt: string | null
  scopes: string[]
  status: 'active' | 'revoked' | 'expired'
}

interface RateLimitBucket {
  endpoint: string
  limit: number
  window: string
  current: number
  resetAt: string
}

interface APIMetric {
  time: string
  requests: number
  errors: number
  latency: number
}

const generateMockAPIEndpoints = (): APIEndpoint[] => [
  {
    id: 'api-001',
    path: '/api/v1/health',
    method: 'GET',
    description: 'System health check endpoint',
    authentication: 'none',
    rateLimit: 1000,
    deprecated: false,
    category: 'System',
    status: 'stable',
    parameters: 'none',
    requestBody: 'none',
    responseBody: '{ status: string, timestamp: string, version: string }',
  },
  {
    id: 'api-002',
    path: '/api/v1/nodes',
    method: 'GET',
    description: 'List all compute nodes',
    authentication: 'bearer',
    rateLimit: 500,
    deprecated: false,
    category: 'Nodes',
    status: 'stable',
    parameters: 'limit, offset, status',
    requestBody: 'none',
    responseBody: '{ nodes: Node[], total: number, pagination: {} }',
  },
  {
    id: 'api-003',
    path: '/api/v1/nodes',
    method: 'POST',
    description: 'Create a new compute node',
    authentication: 'bearer',
    rateLimit: 100,
    deprecated: false,
    category: 'Nodes',
    status: 'stable',
    parameters: 'none',
    requestBody: '{ name: string, type: string, cpu: number, memory: number }',
    responseBody: '{ node: Node, id: string }',
  },
  {
    id: 'api-004',
    path: '/api/v1/deployments',
    method: 'GET',
    description: 'List all deployments',
    authentication: 'bearer',
    rateLimit: 300,
    deprecated: false,
    category: 'Deployments',
    status: 'stable',
    parameters: 'limit, offset, status',
    requestBody: 'none',
    responseBody: '{ deployments: Deployment[], total: number }',
  },
  {
    id: 'api-005',
    path: '/api/v1/deployments',
    method: 'POST',
    description: 'Create a new deployment',
    authentication: 'bearer',
    rateLimit: 50,
    deprecated: false,
    category: 'Deployments',
    status: 'stable',
    parameters: 'none',
    requestBody: '{ name: string, version: string, nodes: string[] }',
    responseBody: '{ deployment: Deployment, id: string }',
  },
  {
    id: 'api-006',
    path: '/api/v1/storage/buckets',
    method: 'GET',
    description: 'List storage buckets',
    authentication: 'bearer',
    rateLimit: 200,
    deprecated: false,
    category: 'Storage',
    status: 'stable',
    parameters: 'limit, offset',
    requestBody: 'none',
    responseBody: '{ buckets: Bucket[], total: number }',
  },
  {
    id: 'api-007',
    path: '/api/v1/analytics/metrics',
    method: 'GET',
    description: 'Get system metrics',
    authentication: 'bearer',
    rateLimit: 1000,
    deprecated: false,
    category: 'Analytics',
    status: 'stable',
    parameters: 'start, end, interval',
    requestBody: 'none',
    responseBody: '{ metrics: Metric[], summary: {} }',
  },
  {
    id: 'api-008',
    path: '/api/v1/webhooks',
    method: 'POST',
    description: 'Register webhook (DEPRECATED)',
    authentication: 'bearer',
    rateLimit: 100,
    deprecated: true,
    category: 'Webhooks',
    status: 'deprecated',
    parameters: 'none',
    requestBody: '{ url: string, events: string[] }',
    responseBody: '{ webhook: Webhook, id: string }',
  },
]

const generateMockAPIKeys = (): APIKey[] => [
  {
    id: 'key-001',
    name: 'Production API Key',
    prefix: 'dh_prod_',
    lastUsed: '2026-09-30T10:22:15Z',
    created: '2026-01-15T08:30:00Z',
    expiresAt: null,
    scopes: ['read:all', 'write:deployments', 'write:nodes'],
    status: 'active',
  },
  {
    id: 'key-002',
    name: 'Development Key',
    prefix: 'dh_dev_',
    lastUsed: '2026-09-30T09:45:22Z',
    created: '2026-03-01T14:20:00Z',
    expiresAt: '2026-12-31T23:59:59Z',
    scopes: ['read:all', 'write:all'],
    status: 'active',
  },
  {
    id: 'key-003',
    name: 'Integration Key (CI/CD)',
    prefix: 'dh_ci_',
    lastUsed: '2026-09-29T22:15:00Z',
    created: '2026-06-20T10:00:00Z',
    expiresAt: '2026-12-20T23:59:59Z',
    scopes: ['read:all', 'write:deployments'],
    status: 'active',
  },
  {
    id: 'key-004',
    name: 'Old Staging Key',
    prefix: 'dh_stg_',
    lastUsed: '2026-08-15T14:30:00Z',
    created: '2025-06-01T09:00:00Z',
    expiresAt: '2026-06-01T23:59:59Z',
    scopes: ['read:all'],
    status: 'expired',
  },
]

const generateMockRateLimits = (): RateLimitBucket[] => [
  { endpoint: 'GET /api/v1/health', limit: 1000, window: '1 minute', current: 247, resetAt: '2026-09-30T10:31:00Z' },
  { endpoint: 'GET /api/v1/nodes', limit: 500, window: '1 minute', current: 186, resetAt: '2026-09-30T10:31:00Z' },
  { endpoint: 'POST /api/v1/nodes', limit: 100, window: '1 minute', current: 8, resetAt: '2026-09-30T10:31:00Z' },
  { endpoint: 'GET /api/v1/deployments', limit: 300, window: '1 minute', current: 142, resetAt: '2026-09-30T10:31:00Z' },
  { endpoint: 'POST /api/v1/deployments', limit: 50, window: '1 minute', current: 3, resetAt: '2026-09-30T10:31:00Z' },
]

const generateMockAPIMetrics = () => [
  { time: '00:00', requests: 4200, errors: 8, latency: 142 },
  { time: '04:00', requests: 2400, errors: 3, latency: 98 },
  { time: '08:00', requests: 8900, errors: 18, latency: 256 },
  { time: '12:00', requests: 12400, errors: 31, latency: 384 },
  { time: '16:00', requests: 14800, errors: 42, latency: 512 },
  { time: '20:00', requests: 9600, errors: 19, latency: 276 },
  { time: '23:59', requests: 5200, errors: 9, latency: 158 },
]

export default function APIDocsPage() {
  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([])
  const [apiKeys, setAPIKeys] = useState<APIKey[]>([])
  const [rateLimits, setRateLimits] = useState<RateLimitBucket[]>([])
  const [selectedCategory, setSelectedCategory] = useState<string>('all')
  const [expandedEndpoint, setExpandedEndpoint] = useState<string | null>(null)
  const [expandedKey, setExpandedKey] = useState<string | null>(null)

  useEffect(() => {
    setEndpoints(generateMockAPIEndpoints())
    setAPIKeys(generateMockAPIKeys())
    setRateLimits(generateMockRateLimits())
  }, [])

  const categories = ['all', ...new Set(endpoints.map(e => e.category))]
  const filteredEndpoints = selectedCategory === 'all' ? endpoints : endpoints.filter(e => e.category === selectedCategory)
  const totalRequests = generateMockAPIMetrics().reduce((sum, m) => sum + m.requests, 0)
  const totalErrors = generateMockAPIMetrics().reduce((sum, m) => sum + m.errors, 0)
  const errorRate = ((totalErrors / totalRequests) * 100).toFixed(2)
  const activeKeys = apiKeys.filter(k => k.status === 'active').length

  const getMethodColor = (method: string) => {
    switch (method) {
      case 'GET':
        return 'bg-blue-500/20 text-blue-400 border-blue-500/50'
      case 'POST':
        return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'PUT':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'DELETE':
        return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'PATCH':
        return 'bg-purple-500/20 text-purple-400 border-purple-500/50'
      default:
        return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  const getAuthColor = (auth: string) => {
    switch (auth) {
      case 'bearer':
        return 'text-blue-400'
      case 'api-key':
        return 'text-purple-400'
      case 'none':
        return 'text-green-400'
      default:
        return 'text-neutral-400'
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'expired':
        return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'revoked':
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
            <h1 className="text-4xl font-bold text-white mb-2">API Documentation</h1>
            <p className="text-neutral-400">API reference, authentication, rate limits, and developer resources</p>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-5 gap-4">
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">API Endpoints</div>
              <div className="text-3xl font-bold text-white mb-1">{endpoints.length}</div>
              <div className="text-xs text-neutral-400">{endpoints.filter(e => !e.deprecated).length} active</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Total Requests (24h)</div>
              <div className="text-3xl font-bold text-white mb-1">{(totalRequests / 1000).toFixed(0)}K</div>
              <div className="text-xs text-neutral-400">Peak capacity</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Error Rate</div>
              <div className="text-3xl font-bold text-white mb-1">{errorRate}%</div>
              <div className="text-xs text-green-400">Within SLA</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Active API Keys</div>
              <div className="text-3xl font-bold text-white mb-1">{activeKeys}</div>
              <div className="text-xs text-neutral-400">Total: {apiKeys.length}</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">API Version</div>
              <div className="text-3xl font-bold text-white mb-1">v1</div>
              <div className="text-xs text-neutral-400">Production</div>
            </Card>
          </div>

          {/* Authentication Info */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-4">Authentication</h2>
            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                <div className="text-white font-semibold mb-2">Bearer Token</div>
                <div className="text-neutral-400 text-sm mb-3">
                  Include in Authorization header:
                </div>
                <div className="bg-neutral-950 p-2 rounded text-primary-400 font-mono text-xs">
                  Authorization: Bearer {'<token>'}
                </div>
              </div>
              <div className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                <div className="text-white font-semibold mb-2">API Key</div>
                <div className="text-neutral-400 text-sm mb-3">
                  Include in request header:
                </div>
                <div className="bg-neutral-950 p-2 rounded text-primary-400 font-mono text-xs">
                  X-API-Key: {'<api_key>'}
                </div>
              </div>
            </div>
          </Card>

          {/* Endpoints */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-4">API Endpoints</h2>
            <div className="flex gap-2 mb-6 flex-wrap">
              {categories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setSelectedCategory(cat)}
                  className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
                    selectedCategory === cat
                      ? 'bg-primary-500/30 text-primary-400 border border-primary-500/50'
                      : 'bg-neutral-800 text-neutral-300 hover:bg-neutral-700'
                  }`}
                >
                  {cat.charAt(0).toUpperCase() + cat.slice(1)}
                </button>
              ))}
            </div>
            <div className="space-y-3">
              {filteredEndpoints.map((endpoint) => (
                <div key={endpoint.id} className="border border-neutral-700 rounded-lg overflow-hidden">
                  <div
                    onClick={() => setExpandedEndpoint(expandedEndpoint === endpoint.id ? null : endpoint.id)}
                    className="p-4 bg-neutral-900 hover:bg-neutral-800/50 cursor-pointer transition-colors flex items-center justify-between"
                  >
                    <div className="flex items-center gap-4 flex-1">
                      <Badge className={getMethodColor(endpoint.method)}>
                        {endpoint.method}
                      </Badge>
                      <div className="flex-1">
                        <div className="text-white font-mono font-semibold">{endpoint.path}</div>
                        <div className="text-neutral-400 text-sm">{endpoint.description}</div>
                      </div>
                      {endpoint.deprecated && (
                        <Badge className="bg-red-500/20 text-red-400 border-red-500/50">
                          DEPRECATED
                        </Badge>
                      )}
                    </div>
                    <div className="text-neutral-400">{expandedEndpoint === endpoint.id ? '−' : '+'}</div>
                  </div>
                  {expandedEndpoint === endpoint.id && (
                    <div className="p-4 bg-neutral-950 border-t border-neutral-700 space-y-4">
                      <div className="grid grid-cols-3 gap-4">
                        <div>
                          <div className="text-neutral-400 text-xs font-medium mb-1">Status</div>
                          <Badge className={endpoint.status === 'stable' ? 'bg-green-500/20 text-green-400 border-green-500/50' : endpoint.status === 'beta' ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50' : 'bg-red-500/20 text-red-400 border-red-500/50'}>
                            {endpoint.status.charAt(0).toUpperCase() + endpoint.status.slice(1)}
                          </Badge>
                        </div>
                        <div>
                          <div className="text-neutral-400 text-xs font-medium mb-1">Authentication</div>
                          <div className={`font-mono text-sm font-semibold ${getAuthColor(endpoint.authentication)}`}>
                            {endpoint.authentication.replace('-', ' ')}
                          </div>
                        </div>
                        <div>
                          <div className="text-neutral-400 text-xs font-medium mb-1">Rate Limit</div>
                          <div className="text-white font-semibold">{endpoint.rateLimit}/min</div>
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <div className="text-neutral-400 text-xs font-medium mb-2">Request Parameters</div>
                          <div className="text-white font-mono text-sm bg-neutral-900 p-2 rounded">
                            {endpoint.parameters}
                          </div>
                        </div>
                        <div>
                          <div className="text-neutral-400 text-xs font-medium mb-2">Request Body</div>
                          <div className="text-white font-mono text-sm bg-neutral-900 p-2 rounded">
                            {endpoint.requestBody}
                          </div>
                        </div>
                      </div>
                      <div>
                        <div className="text-neutral-400 text-xs font-medium mb-2">Response Body</div>
                        <div className="text-white font-mono text-sm bg-neutral-900 p-2 rounded">
                          {endpoint.responseBody}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </Card>

          {/* API Keys */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">API Keys</h2>
            <div className="space-y-3">
              {apiKeys.map((key) => (
                <div key={key.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <div className="text-white font-semibold">{key.name}</div>
                      <div className="text-neutral-400 text-sm font-mono">{key.prefix}••••••••••</div>
                    </div>
                    <Badge className={getStatusColor(key.status)}>
                      {key.status.charAt(0).toUpperCase() + key.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-4 gap-4 text-sm">
                    <div>
                      <div className="text-neutral-400 mb-1">Created</div>
                      <div className="text-white">{new Date(key.created).toLocaleDateString()}</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 mb-1">Last Used</div>
                      <div className="text-white">{new Date(key.lastUsed).toLocaleDateString()}</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 mb-1">Expires</div>
                      <div className="text-white">{key.expiresAt ? new Date(key.expiresAt).toLocaleDateString() : 'Never'}</div>
                    </div>
                    <div className="text-right">
                      <button className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors">
                        Revoke
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            <button className="mt-6 px-4 py-2 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 font-medium transition-colors">
              Generate New API Key
            </button>
          </Card>

          {/* Rate Limits */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Rate Limits</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Endpoint</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Limit</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Window</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Current</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Resets</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Utilization</th>
                  </tr>
                </thead>
                <tbody>
                  {rateLimits.map((rl) => {
                    const utilization = (rl.current / rl.limit) * 100
                    return (
                      <tr key={rl.endpoint} className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white font-mono text-sm">{rl.endpoint}</td>
                        <td className="py-3 px-4 text-neutral-300">{rl.limit}</td>
                        <td className="py-3 px-4 text-neutral-300">{rl.window}</td>
                        <td className="py-3 px-4 text-white font-semibold">{rl.current}</td>
                        <td className="py-3 px-4 text-neutral-300 text-sm">{new Date(rl.resetAt).toLocaleTimeString()}</td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            <div className="w-24 bg-neutral-800 rounded-full h-2">
                              <div
                                className={`h-2 rounded-full ${
                                  utilization > 80
                                    ? 'bg-red-500'
                                    : utilization > 50
                                    ? 'bg-yellow-500'
                                    : 'bg-green-500'
                                }`}
                                style={{ width: `${utilization}%` }}
                              />
                            </div>
                            <span className="text-neutral-300 text-sm">{utilization.toFixed(0)}%</span>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
