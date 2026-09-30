import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

interface Incident {
  id: string
  title: string
  severity: 'P1' | 'P2' | 'P3' | 'P4'
  status: 'active' | 'resolved' | 'investigating'
  started: string
  resolved?: string
  affectedServices: string[]
  duration: string
  mttr: string
}

const generateMockIncidents = (): Incident[] => [
  { id: 'inc-001', title: 'Database Availability Degradation', severity: 'P1', status: 'investigating', started: '2026-09-30T10:15Z', affectedServices: ['API', 'Dashboard'], duration: '20m', mttr: '-' },
  { id: 'inc-002', title: 'High Latency on API Endpoints', severity: 'P2', status: 'active', started: '2026-09-30T09:45Z', affectedServices: ['API'], duration: '50m', mttr: '-' },
  { id: 'inc-003', title: 'Cache Cluster Failover', severity: 'P2', status: 'resolved', started: '2026-09-29T22:30Z', resolved: '2026-09-29T22:45Z', affectedServices: ['Cache'], duration: '15m', mttr: '15m' },
  { id: 'inc-004', title: 'Deployment Pipeline Failure', severity: 'P3', status: 'resolved', started: '2026-09-29T14:20Z', resolved: '2026-09-29T14:35Z', affectedServices: ['CI/CD'], duration: '15m', mttr: '15m' },
  { id: 'inc-005', title: 'Certificate Renewal Issue', severity: 'P3', status: 'resolved', started: '2026-09-28T18:00Z', resolved: '2026-09-28T18:30Z', affectedServices: ['Security'], duration: '30m', mttr: '30m' },
]

const generateMetrics = () => [
  { name: 'P1', count: 12 },
  { name: 'P2', count: 24 },
  { name: 'P3', count: 38 },
  { name: 'P4', count: 15 },
]

export default function IncidentsPage() {
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [metrics, setMetrics] = useState<any[]>([])

  useEffect(() => {
    setIncidents(generateMockIncidents())
    setMetrics(generateMetrics())
  }, [])

  const active = incidents.filter(i => i.status === 'active' || i.status === 'investigating').length
  const resolved = incidents.filter(i => i.status === 'resolved').length
  const avgMTTR = '24 minutes'

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'P1': return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'P2': return 'bg-orange-500/20 text-orange-400 border-orange-500/50'
      case 'P3': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'P4': return 'bg-blue-500/20 text-blue-400 border-blue-500/50'
      default: return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Incidents</h1>
            <p className="text-neutral-400">Track and manage system incidents</p>
          </div>

          <div className="grid grid-cols-5 gap-4">
            <Card><div className="text-neutral-400 text-sm mb-2">Active</div><div className="text-3xl font-bold text-red-400">{active}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Resolved (30d)</div><div className="text-3xl font-bold text-green-400">{resolved}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Avg MTTR</div><div className="text-3xl font-bold text-white">{avgMTTR}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Total (30d)</div><div className="text-3xl font-bold text-white">{incidents.length}</div></Card>
            <Card><div className="text-neutral-400 text-sm mb-2">Severity</div><div className="text-3xl font-bold text-yellow-400">Mixed</div></Card>
          </div>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Incidents by Severity</h2>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={metrics}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="name" stroke="#999" />
                <YAxis stroke="#999" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                <Bar dataKey="count" fill="#00D9FF" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Recent Incidents</h2>
            <div className="space-y-3">
              {incidents.map((incident) => (
                <div key={incident.id} className="p-4 bg-neutral-900 rounded-lg border border-neutral-700">
                  <div className="flex justify-between items-start mb-2">
                    <div className="flex items-center gap-2">
                      <Badge className={getSeverityColor(incident.severity)}>{incident.severity}</Badge>
                      <div className="text-white font-semibold">{incident.title}</div>
                    </div>
                    <Badge className={incident.status === 'active' || incident.status === 'investigating' ? 'bg-red-500/20 text-red-400 border-red-500/50' : 'bg-green-500/20 text-green-400 border-green-500/50'}>
                      {incident.status.charAt(0).toUpperCase() + incident.status.slice(1)}
                    </Badge>
                  </div>
                  <div className="text-sm text-neutral-400 mb-2">{incident.affectedServices.join(', ')}</div>
                  <div className="grid grid-cols-3 gap-4 text-xs text-neutral-400">
                    <div>Started: {new Date(incident.started).toLocaleString()}</div>
                    <div>Duration: {incident.duration}</div>
                    <div>MTTR: {incident.mttr}</div>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
