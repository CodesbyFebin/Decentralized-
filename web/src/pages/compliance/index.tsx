import React, { useState } from 'react'
import { CheckCircle, AlertCircle, XCircle } from 'lucide-react'
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts'

interface ComplianceControl {
  id: string
  name: string
  framework: string
  status: 'compliant' | 'non-compliant' | 'in-progress'
  lastCheck: string
  evidence: string
  percentage: number
}

interface ComplianceFramework {
  name: string
  controls: number
  compliant: number
  percentage: number
}

const mockControls: ComplianceControl[] = [
  { id: '1', name: 'Access Control Policy', framework: 'ISO 27001', status: 'compliant', lastCheck: '2024-01-15T10:30:00Z', evidence: 'config-audit-2024-01-15', percentage: 100 },
  { id: '2', name: 'Data Encryption', framework: 'PCI-DSS', status: 'compliant', lastCheck: '2024-01-14T15:45:00Z', evidence: 'crypto-audit-2024-01-14', percentage: 100 },
  { id: '3', name: 'Incident Response Plan', framework: 'NIST 800-53', status: 'compliant', lastCheck: '2024-01-10T09:20:00Z', evidence: 'incident-plan-v3.2', percentage: 100 },
  { id: '4', name: 'Backup Retention', framework: 'GDPR', status: 'in-progress', lastCheck: '2024-01-12T14:00:00Z', evidence: 'backup-review-2024-01', percentage: 85 },
  { id: '5', name: 'Audit Logging', framework: 'SOC 2', status: 'compliant', lastCheck: '2024-01-15T11:15:00Z', evidence: 'audit-config-verified', percentage: 100 },
]

const frameworkData: ComplianceFramework[] = [
  { name: 'ISO 27001', controls: 14, compliant: 14, percentage: 100 },
  { name: 'PCI-DSS', controls: 12, compliant: 11, percentage: 92 },
  { name: 'GDPR', controls: 8, compliant: 6, percentage: 75 },
  { name: 'NIST 800-53', controls: 20, compliant: 19, percentage: 95 },
  { name: 'SOC 2', controls: 10, compliant: 10, percentage: 100 },
]

const complianceStatusData = [
  { name: 'Compliant', value: 60, fill: '#10b981' },
  { name: 'In Progress', value: 10, fill: '#f59e0b' },
  { name: 'Non-Compliant', value: 5, fill: '#ef4444' },
]

export default function Compliance() {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const overallCompliance = Math.round((60 / 75) * 100)

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Compliance Dashboard</h1>
            <p className="text-neutral-400 mt-1">Monitor regulatory compliance across frameworks and controls</p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Overall Compliance</p>
            <p className="text-2xl font-bold text-green-500 mt-2">{overallCompliance}%</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Compliant Controls</p>
            <p className="text-2xl font-bold text-green-500 mt-2">60</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">In Progress</p>
            <p className="text-2xl font-bold text-yellow-500 mt-2">10</p>
          </div>
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-4">
            <p className="text-neutral-400 text-sm">Non-Compliant</p>
            <p className="text-2xl font-bold text-red-500 mt-2">5</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Compliance Status</h3>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie data={complianceStatusData} cx="50%" cy="50%" labelLine={false} label={({ name, value }) => `${name} (${value})`} outerRadius={100} fill="#8884d8" dataKey="value">
                  {complianceStatusData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.fill} />
                  ))}
                </Pie>
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px' }} />
              </PieChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-4">Framework Compliance</h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={frameworkData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
                <XAxis dataKey="name" stroke="#737373" angle={-45} textAnchor="end" height={80} />
                <YAxis stroke="#737373" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #404040', borderRadius: '8px', color: '#e5e5e5' }} />
                <Bar dataKey="percentage" fill="#10b981" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-white mb-6">Control Status</h3>
          <div className="space-y-3">
            {mockControls.map((control) => (
              <div key={control.id} className="border border-neutral-800 rounded-lg p-4 hover:border-neutral-700 transition-colors">
                <div className="flex items-start gap-4">
                  <div className="mt-1">
                    {control.status === 'compliant' && <CheckCircle className="w-5 h-5 text-green-500" />}
                    {control.status === 'in-progress' && <AlertCircle className="w-5 h-5 text-yellow-500" />}
                    {control.status === 'non-compliant' && <XCircle className="w-5 h-5 text-red-500" />}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="font-semibold text-white">{control.name}</h4>
                        <p className="text-sm text-neutral-400 mt-1">{control.framework}</p>
                      </div>
                      <span className={`px-3 py-1 rounded text-xs font-medium ${control.status === 'compliant' ? 'bg-green-500/20 text-green-400' : control.status === 'in-progress' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-red-500/20 text-red-400'}`}>
                        {control.status}
                      </span>
                    </div>
                    <div className="mt-3 w-full bg-neutral-800 rounded-full h-2">
                      <div className={`h-2 rounded-full ${control.percentage === 100 ? 'bg-green-500' : 'bg-yellow-500'}`} style={{ width: `${control.percentage}%` }} />
                    </div>
                    <p className="text-xs text-neutral-400 mt-2">Last Check: {new Date(control.lastCheck).toLocaleString()} | Evidence: {control.evidence}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Compliance Reports</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {[{ name: 'Annual Compliance Report', date: '2024-01-15', status: 'ready' }, { name: 'PCI-DSS Assessment', date: '2024-01-10', status: 'in-progress' }, { name: 'GDPR Data Processing Audit', date: '2024-01-01', status: 'ready' }].map((report, idx) => (
              <div key={idx} className="border border-neutral-800 rounded-lg p-4">
                <p className="font-medium text-white">{report.name}</p>
                <p className="text-sm text-neutral-400 mt-1">Generated: {report.date}</p>
                <span className={`inline-block px-2 py-1 rounded text-xs font-medium mt-3 ${report.status === 'ready' ? 'bg-green-500/20 text-green-400' : 'bg-yellow-500/20 text-yellow-400'}`}>
                  {report.status === 'ready' ? 'Ready' : 'In Progress'}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
