import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { LineChart } from '@/components/Charts/LineChart'
import { apiClient } from '@/lib/api'

interface ComplianceCheck {
  id: string
  name: string
  framework: 'SOC2' | 'ISO27001' | 'HIPAA' | 'GDPR' | 'PCI-DSS'
  status: 'compliant' | 'partial' | 'non-compliant'
  score: number
  lastAudit: string
  nextAudit: string
  findings: number
  criticalFindings: number
}

interface ComplianceReport {
  date: string
  score: number
  checks: number
  passed: number
  failed: number
}

interface ComplianceEvidence {
  id: string
  checkId: string
  checkName: string
  evidenceType: string
  status: 'verified' | 'pending' | 'failed'
  collectedAt: string
  expiresAt: string
}

const Compliance: React.FC = () => {
  const [checks, setChecks] = useState<ComplianceCheck[]>([])
  const [reports, setReports] = useState<ComplianceReport[]>([])
  const [evidence, setEvidence] = useState<ComplianceEvidence[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedFramework, setSelectedFramework] = useState<string>('all')
  const [selectedReport, setSelectedReport] = useState<ComplianceReport | null>(null)

  const generateMockComplianceChecks = (): ComplianceCheck[] => [
    {
      id: 'check-1',
      name: 'Encryption at Rest',
      framework: 'SOC2',
      status: 'compliant',
      score: 100,
      lastAudit: '2026-09-30T08:00:00Z',
      nextAudit: '2026-10-30T08:00:00Z',
      findings: 0,
      criticalFindings: 0,
    },
    {
      id: 'check-2',
      name: 'Access Control Policy',
      framework: 'ISO27001',
      status: 'compliant',
      score: 95,
      lastAudit: '2026-09-29T14:30:00Z',
      nextAudit: '2026-10-29T14:30:00Z',
      findings: 1,
      criticalFindings: 0,
    },
    {
      id: 'check-3',
      name: 'Audit Logging',
      framework: 'HIPAA',
      status: 'compliant',
      score: 98,
      lastAudit: '2026-09-28T10:15:00Z',
      nextAudit: '2026-10-28T10:15:00Z',
      findings: 0,
      criticalFindings: 0,
    },
    {
      id: 'check-4',
      name: 'Data Processing Agreement',
      framework: 'GDPR',
      status: 'partial',
      score: 75,
      lastAudit: '2026-09-27T11:45:00Z',
      nextAudit: '2026-10-27T11:45:00Z',
      findings: 3,
      criticalFindings: 1,
    },
    {
      id: 'check-5',
      name: 'Encryption in Transit',
      framework: 'PCI-DSS',
      status: 'compliant',
      score: 100,
      lastAudit: '2026-09-26T09:20:00Z',
      nextAudit: '2026-10-26T09:20:00Z',
      findings: 0,
      criticalFindings: 0,
    },
    {
      id: 'check-6',
      name: 'Incident Response Plan',
      framework: 'SOC2',
      status: 'partial',
      score: 80,
      lastAudit: '2026-09-25T15:00:00Z',
      nextAudit: '2026-10-25T15:00:00Z',
      findings: 2,
      criticalFindings: 0,
    },
  ]

  const generateMockReports = (): ComplianceReport[] => {
    const reports: ComplianceReport[] = []
    for (let i = 30; i >= 0; i--) {
      const date = new Date(Date.now() - i * 86400000).toISOString().split('T')[0]
      reports.push({
        date,
        score: 85 + Math.random() * 12,
        checks: 6,
        passed: 5 + Math.floor(Math.random() * 2),
        failed: Math.floor(Math.random() * 2),
      })
    }
    return reports
  }

  const generateMockEvidence = (): ComplianceEvidence[] => [
    {
      id: 'evid-1',
      checkId: 'check-1',
      checkName: 'Encryption at Rest',
      evidenceType: 'Configuration Audit',
      status: 'verified',
      collectedAt: '2026-09-30T08:00:00Z',
      expiresAt: '2026-10-30T08:00:00Z',
    },
    {
      id: 'evid-2',
      checkId: 'check-2',
      checkName: 'Access Control Policy',
      evidenceType: 'Policy Document',
      status: 'verified',
      collectedAt: '2026-09-29T14:30:00Z',
      expiresAt: '2026-10-29T14:30:00Z',
    },
    {
      id: 'evid-3',
      checkId: 'check-3',
      checkName: 'Audit Logging',
      evidenceType: 'Log Verification',
      status: 'verified',
      collectedAt: '2026-09-28T10:15:00Z',
      expiresAt: '2026-10-28T10:15:00Z',
    },
    {
      id: 'evid-4',
      checkId: 'check-4',
      checkName: 'Data Processing Agreement',
      evidenceType: 'Legal Document',
      status: 'pending',
      collectedAt: '2026-09-27T11:45:00Z',
      expiresAt: '2026-10-27T11:45:00Z',
    },
    {
      id: 'evid-5',
      checkId: 'check-5',
      checkName: 'Encryption in Transit',
      evidenceType: 'TLS Certificate',
      status: 'verified',
      collectedAt: '2026-09-26T09:20:00Z',
      expiresAt: '2026-10-26T09:20:00Z',
    },
  ]

  useEffect(() => {
    const loadCompliance = async () => {
      try {
        setLoading(true)
        const mockChecks = generateMockComplianceChecks()
        const mockReports = generateMockReports()
        const mockEvidence = generateMockEvidence()
        setChecks(mockChecks)
        setReports(mockReports)
        setEvidence(mockEvidence)
      } catch (err) {
        console.error('Failed to load compliance data:', err)
      } finally {
        setLoading(false)
      }
    }

    loadCompliance()
  }, [])

  const filteredChecks =
    selectedFramework === 'all' ? checks : checks.filter((c) => c.framework === selectedFramework)

  const overallScore = Math.round(
    checks.reduce((sum, c) => sum + c.score, 0) / checks.length
  )
  const compliantChecks = checks.filter((c) => c.status === 'compliant').length
  const totalCriticalFindings = checks.reduce((sum, c) => sum + c.criticalFindings, 0)

  const chartData = reports.map((r) => ({
    date: r.date.split('-').slice(1).join('-'),
    score: Math.round(r.score * 10) / 10,
  }))

  if (loading) {
    return (
      <AppLayout title="Compliance" subtitle="Compliance tracking and audit reports">
        <Loading message="Loading compliance data..." />
      </AppLayout>
    )
  }

  return (
    <AppLayout
      title="Compliance"
      subtitle={`${overallScore}% overall score • ${compliantChecks}/${checks.length} frameworks compliant`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Overall Score</p>
              <span className="text-3xl font-bold text-primary-500">{overallScore}%</span>
              <p className="text-xs text-neutral-500 mt-2">compliance rating</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Compliant Checks</p>
              <span className="text-3xl font-bold text-success-500">
                {compliantChecks}/{checks.length}
              </span>
              <p className="text-xs text-neutral-500 mt-2">frameworks</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Critical Findings</p>
              <span className="text-3xl font-bold text-error-500">{totalCriticalFindings}</span>
              <p className="text-xs text-neutral-500 mt-2">issues</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Evidence</p>
              <span className="text-3xl font-bold text-info-500">{evidence.length}</span>
              <p className="text-xs text-neutral-500 mt-2">artifacts</p>
            </div>
          </Card>
        </div>

        {/* Compliance Score Trend */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Compliance Score Trend (30d)</h3>
          </div>
          <div className="p-6">
            <LineChart
              data={chartData}
              dataKey="score"
              name="Compliance Score"
              stroke="#00D9FF"
              height={250}
              xAxisKey="date"
            />
          </div>
        </Card>

        {/* Compliance Checks */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-white">Compliance Checks</h3>
              <select
                value={selectedFramework}
                onChange={(e) => setSelectedFramework(e.target.value)}
                className="px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white text-sm focus:border-primary-500 focus:outline-none"
              >
                <option value="all">All Frameworks</option>
                <option value="SOC2">SOC2</option>
                <option value="ISO27001">ISO 27001</option>
                <option value="HIPAA">HIPAA</option>
                <option value="GDPR">GDPR</option>
                <option value="PCI-DSS">PCI-DSS</option>
              </select>
            </div>
          </div>

          <div className="divide-y divide-neutral-700">
            {filteredChecks.map((check) => (
              <div key={check.id} className="p-6 hover:bg-neutral-800/20 transition">
                <div className="flex justify-between items-start gap-4 mb-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <p className="text-white font-medium text-base">{check.name}</p>
                      <Badge
                        status={
                          check.status === 'compliant'
                            ? 'success'
                            : check.status === 'partial'
                              ? 'warning'
                              : 'error'
                        }
                      >
                        {check.status === 'compliant'
                          ? 'Compliant'
                          : check.status === 'partial'
                            ? 'Partial'
                            : 'Non-Compliant'}
                      </Badge>
                    </div>
                    <p className="text-xs text-neutral-500 mb-3">
                      Framework: <span className="text-neutral-400">{check.framework}</span>
                    </p>

                    {/* Score Bar */}
                    <div className="mb-3">
                      <div className="flex justify-between mb-1">
                        <span className="text-xs text-neutral-400">Compliance Score</span>
                        <span className="text-sm font-medium text-white">{check.score}%</span>
                      </div>
                      <div className="w-full bg-neutral-700 rounded-full h-2">
                        <div
                          className={`h-2 rounded-full transition-all ${
                            check.score >= 90
                              ? 'bg-success-500'
                              : check.score >= 75
                                ? 'bg-warning-500'
                                : 'bg-error-500'
                          }`}
                          style={{ width: `${check.score}%` }}
                        />
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-4 text-xs text-neutral-500">
                      <span>Last Audit: {new Date(check.lastAudit).toLocaleDateString()}</span>
                      <span>Next Audit: {new Date(check.nextAudit).toLocaleDateString()}</span>
                      <span className="text-warning-400">
                        {check.findings} finding{check.findings !== 1 ? 's' : ''}
                      </span>
                      {check.criticalFindings > 0 && (
                        <span className="text-error-400">
                          {check.criticalFindings} critical
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Evidence Collection */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Evidence Artifacts</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Check</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Evidence Type</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Collected</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Expires</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {evidence.map((evid) => (
                  <tr key={evid.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium text-sm">
                      {evid.checkName}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs">{evid.evidenceType}</td>
                    <td className="px-6 py-3">
                      <Badge
                        status={
                          evid.status === 'verified'
                            ? 'success'
                            : evid.status === 'pending'
                              ? 'warning'
                              : 'error'
                        }
                      >
                        {evid.status === 'verified'
                          ? 'Verified'
                          : evid.status === 'pending'
                            ? 'Pending'
                            : 'Failed'}
                      </Badge>
                    </td>
                    <td className="px-6 py-3 text-neutral-500 text-xs">
                      {new Date(evid.collectedAt).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-3 text-neutral-500 text-xs">
                      {new Date(evid.expiresAt).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Compliance Reports */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <h3 className="text-lg font-semibold text-white">Recent Reports</h3>
          </div>
          <div className="divide-y divide-neutral-700">
            {reports.slice(-7).reverse().map((report, idx) => (
              <button
                key={idx}
                onClick={() => setSelectedReport(report)}
                className="w-full text-left p-6 hover:bg-neutral-800/20 transition flex justify-between items-center"
              >
                <div>
                  <p className="text-white font-medium">{report.date}</p>
                  <p className="text-xs text-neutral-500 mt-1">
                    {report.passed}/{report.checks} checks passed
                  </p>
                </div>
                <div className="text-right">
                  <p className={`text-2xl font-bold ${
                    report.score >= 90
                      ? 'text-success-500'
                      : report.score >= 75
                        ? 'text-warning-500'
                        : 'text-error-500'
                  }`}>
                    {Math.round(report.score)}%
                  </p>
                </div>
              </button>
            ))}
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Compliance
