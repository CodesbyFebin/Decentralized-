import React, { useState } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Timeline } from 'recharts'

interface ActionItem {
  id: string
  title: string
  owner: string
  dueDate: string
  status: 'open' | 'in-progress' | 'closed'
  priority: 'high' | 'medium' | 'low'
}

interface TimelineEvent {
  time: string
  title: string
  description: string
}

interface Postmortem {
  id: string
  incident: string
  severity: string
  startTime: string
  endTime: string
  duration: string
  rootCause: string
  affectedServices: string[]
  timeline: TimelineEvent[]
  actionItems: ActionItem[]
  mttr: number
}

const mockPostmortems: Postmortem[] = [
  {
    id: '1',
    incident: 'Database Connection Pool Exhaustion',
    severity: 'P1',
    startTime: '2024-01-10T14:30:00Z',
    endTime: '2024-01-10T15:45:00Z',
    duration: '1h 15m',
    rootCause: 'Application connection leak not releasing idle connections after query timeout',
    affectedServices: ['API Gateway', 'User Service', 'Analytics'],
    timeline: [
      { time: '14:30', title: 'Incident Started', description: 'Alert triggered for high database connection count' },
      { time: '14:35', title: 'Investigation', description: 'On-call engineer confirmed database connection exhaustion' },
      { time: '14:50', title: 'Mitigation', description: 'Restarted affected application instances' },
      { time: '15:00', title: 'Recovery', description: 'Services began returning to normal operation' },
      { time: '15:45', title: 'Resolved', description: 'All services at 100% capacity' },
    ],
    actionItems: [
      { id: '1', title: 'Implement connection timeout validation', owner: 'Sarah Chen', dueDate: '2024-01-20', status: 'in-progress', priority: 'high' },
      { id: '2', title: 'Add circuit breaker for database connections', owner: 'James Lee', dueDate: '2024-01-22', status: 'open', priority: 'high' },
      { id: '3', title: 'Deploy connection pool monitoring', owner: 'Alex Kumar', dueDate: '2024-01-18', status: 'closed', priority: 'medium' },
    ],
    mttr: 75,
  },
  {
    id: '2',
    incident: 'DNS Resolution Timeout',
    severity: 'P2',
    startTime: '2024-01-08T09:15:00Z',
    endTime: '2024-01-08T09:52:00Z',
    duration: '37m',
    rootCause: 'DNS server overload due to increased query volume from new deployment',
    affectedServices: ['Frontend', 'Mobile App'],
    timeline: [
      { time: '09:15', title: 'Incident Started', description: 'Client-side DNS resolution failures reported' },
      { time: '09:20', title: 'Detection', description: 'DNS query response times exceeded threshold' },
      { time: '09:30', title: 'Mitigation', description: 'Increased DNS server capacity and adjusted TTL' },
      { time: '09:52', title: 'Resolved', description: 'Query response times normalized' },
    ],
    actionItems: [
      { id: '4', title: 'Implement DNS query caching', owner: 'Nina Patel', dueDate: '2024-01-17', status: 'open', priority: 'medium' },
      { id: '5', title: 'Pre-deployment DNS load testing', owner: 'Marcus Johnson', dueDate: '2024-01-25', status: 'open', priority: 'low' },
    ],
    mttr: 37,
  },
]

const mttrData = [
  { name: 'P1 Incidents', mttr: 62, avgMttr: 75 },
  { name: 'P2 Incidents', mttr: 47, avgMttr: 45 },
  { name: 'P3 Incidents', mttr: 23, avgMttr: 25 },
]

export default function Postmortems() {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Postmortems</h1>
            <p className="text-neutral-400 mt-1">Incident analysis and root cause documentation</p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Total Incidents</p>
            <p className="text-2xl font-bold text-white mt-2">24</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Avg MTTR</p>
            <p className="text-2xl font-bold text-blue-500 mt-2">48m</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Open Action Items</p>
            <p className="text-2xl font-bold text-yellow-500 mt-2">12</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Closed Action Items</p>
            <p className="text-2xl font-bold text-green-500 mt-2">31</p>
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-white mb-4">MTTR by Severity</h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={mttrData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
              <XAxis dataKey="name" stroke="#737373" />
              <YAxis stroke="#737373" />
              <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px', color: '#e5e5e5' }} />
              <Legend />
              <Bar dataKey="mttr" fill="#3b82f6" name="Current MTTR (min)" />
              <Bar dataKey="avgMttr" fill="#6366f1" name="Target MTTR (min)" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="space-y-6">
          {mockPostmortems.map((pm) => (
            <div key={pm.id} className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
              <div className="flex items-start justify-between cursor-pointer" onClick={() => setExpandedId(expandedId === pm.id ? null : pm.id)}>
                <div className="flex-1">
                  <h3 className="text-lg font-semibold text-white">{pm.incident}</h3>
                  <p className="text-sm text-neutral-400 mt-1">
                    {new Date(pm.startTime).toLocaleDateString()} • Duration: {pm.duration} • MTTR: {pm.mttr}m
                  </p>
                  <div className="flex gap-2 mt-3 flex-wrap">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${pm.severity === 'P1' ? 'bg-red-500/20 text-red-400' : 'bg-yellow-500/20 text-yellow-400'}`}>
                      {pm.severity}
                    </span>
                    {pm.affectedServices.map((service) => (
                      <span key={service} className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs">
                        {service}
                      </span>
                    ))}
                  </div>
                </div>
                <span className="text-2xl text-neutral-400">{expandedId === pm.id ? '−' : '+'}</span>
              </div>

              {expandedId === pm.id && (
                <div className="mt-6 space-y-6 border-t border-neutral-800 pt-6">
                  <div>
                    <h4 className="font-semibold text-white mb-2">Root Cause</h4>
                    <p className="text-neutral-300">{pm.rootCause}</p>
                  </div>

                  <div>
                    <h4 className="font-semibold text-white mb-3">Incident Timeline</h4>
                    <div className="space-y-2">
                      {pm.timeline.map((event, idx) => (
                        <div key={idx} className="flex gap-4">
                          <div className="flex flex-col items-center">
                            <div className="w-3 h-3 rounded-full bg-primary-500 mt-1" />
                            {idx < pm.timeline.length - 1 && <div className="w-0.5 h-12 bg-neutral-700" />}
                          </div>
                          <div>
                            <p className="font-medium text-white">{event.title}</p>
                            <p className="text-sm text-neutral-400">{event.time} - {event.description}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div>
                    <h4 className="font-semibold text-white mb-3">Action Items</h4>
                    <div className="space-y-2">
                      {pm.actionItems.map((item) => (
                        <div key={item.id} className="border border-neutral-800 rounded p-3 flex items-start gap-3">
                          <div className="w-4 h-4 rounded border-2 border-neutral-700 mt-0.5 flex-shrink-0" />
                          <div className="flex-1">
                            <div className="flex items-center justify-between">
                              <p className="font-medium text-white">{item.title}</p>
                              <span className={`px-2 py-1 rounded text-xs font-medium ${item.priority === 'high' ? 'bg-red-500/20 text-red-400' : item.priority === 'medium' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-blue-500/20 text-blue-400'}`}>
                                {item.priority}
                              </span>
                            </div>
                            <p className="text-sm text-neutral-400 mt-1">Owner: {item.owner} | Due: {item.dueDate}</p>
                            <span className={`inline-block px-2 py-1 rounded text-xs mt-2 ${item.status === 'closed' ? 'bg-green-500/20 text-green-400' : item.status === 'in-progress' ? 'bg-blue-500/20 text-blue-400' : 'bg-neutral-800 text-neutral-400'}`}>
                              {item.status}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
