import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'

interface NetworkNode {
  id: string
  name: string
  region: string
  status: 'connected' | 'disconnected' | 'degraded'
  latency: number
  bandwidth: number
  uptime: number
}

interface BGPRoute {
  id: string
  destination: string
  nextHop: string
  asPath: string
  status: 'active' | 'inactive'
  prefixLength: number
}

interface PeeringConnection {
  id: string
  peer: string
  peerType: 'direct' | 'exchange' | 'cdn'
  status: 'active' | 'standby' | 'down'
  bandwidth: number
  packetsExchanged: number
}

interface TrafficMetric {
  time: string
  inbound: number
  outbound: number
  total: number
}

interface NetworkIssue {
  id: string
  type: 'congestion' | 'latency' | 'packet_loss' | 'down'
  severity: 'critical' | 'warning' | 'info'
  description: string
  affectedNodes: number
  duration: string
}

const generateMockNetworkNodes = (): NetworkNode[] => [
  { id: 'node-us-east', name: 'US East Region', region: 'us-east-1', status: 'connected', latency: 8, bandwidth: 8.5, uptime: 99.97 },
  { id: 'node-us-west', name: 'US West Region', region: 'us-west-2', status: 'connected', latency: 12, bandwidth: 7.2, uptime: 99.95 },
  { id: 'node-eu-west', name: 'EU West Region', region: 'eu-west-1', status: 'connected', latency: 24, bandwidth: 6.8, uptime: 99.92 },
  { id: 'node-ap-south', name: 'APAC Region', region: 'ap-south-1', status: 'degraded', latency: 42, bandwidth: 4.2, uptime: 99.87 },
]

const generateMockBGPRoutes = (): BGPRoute[] => [
  { id: 'bgp-001', destination: '10.0.0.0/8', nextHop: '192.168.1.1', asPath: 'AS64512 AS64513', status: 'active', prefixLength: 8 },
  { id: 'bgp-002', destination: '172.16.0.0/12', nextHop: '192.168.1.2', asPath: 'AS64514 AS64515', status: 'active', prefixLength: 12 },
  { id: 'bgp-003', destination: '192.168.0.0/16', nextHop: '192.168.1.3', asPath: 'AS64516', status: 'active', prefixLength: 16 },
  { id: 'bgp-004', destination: '203.0.113.0/24', nextHop: '192.168.1.4', asPath: 'AS64517 AS64518', status: 'inactive', prefixLength: 24 },
]

const generateMockPeeringConnections = (): PeeringConnection[] => [
  { id: 'peer-001', peer: 'AWS Direct Connect', peerType: 'direct', status: 'active', bandwidth: 10.0, packetsExchanged: 4280000000 },
  { id: 'peer-002', peer: 'Google Cloud Interconnect', peerType: 'direct', status: 'active', bandwidth: 10.0, packetsExchanged: 2180000000 },
  { id: 'peer-003', peer: 'Equinix IX NY', peerType: 'exchange', status: 'active', bandwidth: 100.0, packetsExchanged: 8920000000 },
  { id: 'peer-004', peer: 'Cloudflare CDN', peerType: 'cdn', status: 'standby', bandwidth: 50.0, packetsExchanged: 1420000000 },
  { id: 'peer-005', peer: 'Level3 Transit', peerType: 'direct', status: 'down', bandwidth: 5.0, packetsExchanged: 0 },
]

const generateMockTrafficMetrics = (): TrafficMetric[] => [
  { time: '00:00', inbound: 450, outbound: 420, total: 870 },
  { time: '04:00', inbound: 280, outbound: 260, total: 540 },
  { time: '08:00', inbound: 820, outbound: 780, total: 1600 },
  { time: '12:00', inbound: 1200, outbound: 1100, total: 2300 },
  { time: '16:00', inbound: 1480, outbound: 1350, total: 2830 },
  { time: '20:00', inbound: 920, outbound: 850, total: 1770 },
  { time: '23:59', inbound: 640, outbound: 580, total: 1220 },
]

const generateMockNetworkIssues = (): NetworkIssue[] => [
  { id: 'issue-001', type: 'latency', severity: 'warning', description: 'Elevated latency detected in APAC region', affectedNodes: 2, duration: '18 minutes' },
  { id: 'issue-002', type: 'congestion', severity: 'info', description: 'High traffic on Equinix IX connection', affectedNodes: 1, duration: 'Ongoing' },
]

const generateMockProtocolDistribution = () => [
  { name: 'IPv4', value: 72, color: '#00D9FF' },
  { name: 'IPv6', value: 18, color: '#7C3AED' },
  { name: 'Other', value: 10, color: '#EC4899' },
]

export default function NetworkPage() {
  const [networkNodes, setNetworkNodes] = useState<NetworkNode[]>([])
  const [bgpRoutes, setBGPRoutes] = useState<BGPRoute[]>([])
  const [peeringConnections, setPeeringConnections] = useState<PeeringConnection[]>([])
  const [trafficMetrics, setTrafficMetrics] = useState<TrafficMetric[]>([])
  const [networkIssues, setNetworkIssues] = useState<NetworkIssue[]>([])
  const [protocolDistribution, setProtocolDistribution] = useState<any[]>([])
  const [expandedRoute, setExpandedRoute] = useState<string | null>(null)

  useEffect(() => {
    setNetworkNodes(generateMockNetworkNodes())
    setBGPRoutes(generateMockBGPRoutes())
    setPeeringConnections(generateMockPeeringConnections())
    setTrafficMetrics(generateMockTrafficMetrics())
    setNetworkIssues(generateMockNetworkIssues())
    setProtocolDistribution(generateMockProtocolDistribution())
  }, [])

  const totalBandwidth = networkNodes.reduce((sum, n) => sum + n.bandwidth, 0)
  const avgLatency = (networkNodes.reduce((sum, n) => sum + n.latency, 0) / networkNodes.length).toFixed(1)
  const connectedNodes = networkNodes.filter(n => n.status === 'connected').length
  const totalTraffic = trafficMetrics[trafficMetrics.length - 1]?.total || 0

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
      case 'active':
        return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'degraded':
      case 'standby':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'disconnected':
      case 'down':
      case 'inactive':
        return 'bg-red-500/20 text-red-400 border-red-500/50'
      default:
        return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'warning':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'info':
        return 'bg-blue-500/20 text-blue-400 border-blue-500/50'
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
            <h1 className="text-4xl font-bold text-white mb-2">Network Management</h1>
            <p className="text-neutral-400">Monitor network topology, BGP routes, peering connections, and traffic</p>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-5 gap-4">
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Connected Nodes</div>
              <div className="text-3xl font-bold text-white mb-1">{connectedNodes}/{networkNodes.length}</div>
              <div className="text-xs text-green-400">All regions operational</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Total Bandwidth</div>
              <div className="text-3xl font-bold text-white mb-1">{totalBandwidth.toFixed(1)} Gbps</div>
              <div className="text-xs text-neutral-400">Aggregate capacity</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Avg Latency</div>
              <div className="text-3xl font-bold text-white mb-1">{avgLatency}ms</div>
              <div className="text-xs text-green-400">Within targets</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Active BGP Routes</div>
              <div className="text-3xl font-bold text-white mb-1">{bgpRoutes.filter(r => r.status === 'active').length}</div>
              <div className="text-xs text-neutral-400">Production routes</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Current Traffic</div>
              <div className="text-3xl font-bold text-white mb-1">{totalTraffic} Mbps</div>
              <div className="text-xs text-neutral-400">Peak capacity</div>
            </Card>
          </div>

          {/* Network Topology */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Network Topology - Regional Nodes</h2>
            <div className="space-y-3">
              {networkNodes.map((node) => (
                <div key={node.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className="w-3 h-3 rounded-full bg-green-500" />
                      <div>
                        <div className="text-white font-semibold">{node.name}</div>
                        <div className="text-neutral-400 text-sm">{node.region}</div>
                      </div>
                    </div>
                    <Badge className={getStatusColor(node.status)}>
                      {node.status.charAt(0).toUpperCase() + node.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Latency</div>
                      <div className="text-white font-semibold">{node.latency}ms</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Bandwidth</div>
                      <div className="text-white font-semibold">{node.bandwidth} Gbps</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Uptime</div>
                      <div className="text-white font-semibold">{node.uptime}%</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Traffic Analysis */}
          <div className="grid grid-cols-2 gap-4">
            <Card>
              <h2 className="text-xl font-bold text-white mb-6">24-Hour Traffic</h2>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={trafficMetrics}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                  <XAxis dataKey="time" stroke="#999" />
                  <YAxis stroke="#999" />
                  <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                  <Legend />
                  <Line type="monotone" dataKey="inbound" stroke="#00D9FF" dot={false} name="Inbound" />
                  <Line type="monotone" dataKey="outbound" stroke="#7C3AED" dot={false} name="Outbound" />
                </LineChart>
              </ResponsiveContainer>
            </Card>

            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Protocol Distribution</h2>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie data={protocolDistribution} cx="50%" cy="50%" outerRadius={80} dataKey="value">
                    {protocolDistribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </div>

          {/* BGP Routes */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">BGP Routes</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Destination</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Next Hop</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">AS Path</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Status</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {bgpRoutes.map((route) => (
                    <React.Fragment key={route.id}>
                      <tr className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white font-mono text-sm">{route.destination}</td>
                        <td className="py-3 px-4 text-neutral-300 font-mono text-sm">{route.nextHop}</td>
                        <td className="py-3 px-4 text-neutral-300 font-mono text-sm">{route.asPath}</td>
                        <td className="py-3 px-4">
                          <Badge className={getStatusColor(route.status)}>
                            {route.status.charAt(0).toUpperCase() + route.status.slice(1)}
                          </Badge>
                        </td>
                        <td className="py-3 px-4">
                          <button
                            onClick={() => setExpandedRoute(expandedRoute === route.id ? null : route.id)}
                            className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors"
                          >
                            {expandedRoute === route.id ? 'Hide' : 'Details'}
                          </button>
                        </td>
                      </tr>
                      {expandedRoute === route.id && (
                        <tr className="border-b border-neutral-800 bg-neutral-900/30">
                          <td colSpan={5} className="py-4 px-4">
                            <div className="grid grid-cols-3 gap-4 text-sm">
                              <div>
                                <div className="text-neutral-400 mb-1">Prefix Length</div>
                                <div className="text-white">{route.prefixLength}</div>
                              </div>
                              <div>
                                <div className="text-neutral-400 mb-1">Route Status</div>
                                <div className="text-white">{route.status}</div>
                              </div>
                              <div className="text-right">
                                <button className="px-3 py-1.5 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 text-sm font-medium transition-colors">
                                  Withdraw Route
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

          {/* Peering Connections */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Peering Connections</h2>
            <div className="grid grid-cols-1 gap-4">
              {peeringConnections.map((peer) => (
                <div key={peer.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex items-center justify-between mb-3">
                    <div>
                      <div className="text-white font-semibold">{peer.peer}</div>
                      <div className="text-neutral-400 text-sm capitalize">{peer.peerType} Peering</div>
                    </div>
                    <Badge className={getStatusColor(peer.status)}>
                      {peer.status.charAt(0).toUpperCase() + peer.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Bandwidth</div>
                      <div className="text-white font-semibold">{peer.bandwidth} Gbps</div>
                    </div>
                    <div>
                      <div className="text-neutral-400 text-xs mb-1">Packets Exchanged</div>
                      <div className="text-white font-semibold">{(peer.packetsExchanged / 1000000000).toFixed(2)}B</div>
                    </div>
                    <div className="text-right">
                      <button className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors">
                        View Stats
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Network Issues & Alerts */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Network Issues & Alerts</h2>
            {networkIssues.length > 0 ? (
              <div className="space-y-3">
                {networkIssues.map((issue) => (
                  <div key={issue.id} className={`p-4 rounded-lg border ${getSeverityColor(issue.severity)}`}>
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <div className="font-semibold capitalize">{issue.type.replace('_', ' ')}</div>
                        <div className="text-sm mt-1">{issue.description}</div>
                      </div>
                      <Badge className={getSeverityColor(issue.severity)}>
                        {issue.severity.charAt(0).toUpperCase() + issue.severity.slice(1)}
                      </Badge>
                    </div>
                    <div className="flex gap-4 text-xs text-neutral-400 mt-3">
                      <div>Affected Nodes: {issue.affectedNodes}</div>
                      <div>Duration: {issue.duration}</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-neutral-400">
                No active network issues detected
              </div>
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
