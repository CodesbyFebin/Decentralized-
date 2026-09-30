import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface QualificationCampaign {
  id: string
  name: string
  status: 'in_progress' | 'passed' | 'failed'
  version: string
  startedAt: string
  completedAt: string | null
  phase: number
  totalPhases: number
  backend: string
  nodeCount: number
}

interface VerificationResult {
  id: string
  campaignId: string
  testNumber: number
  name: string
  status: 'passed' | 'failed' | 'skipped'
  duration: number
  timestamp: string
  assertion: string
}

interface EvidenceArtifact {
  id: string
  campaignId: string
  type: 'source' | 'evidence' | 'signature'
  digest: string
  size: number
  created: string
  signer?: string
}

const Evidence: React.FC = () => {
  const [campaigns, setCampaigns] = useState<QualificationCampaign[]>([])
  const [selectedCampaign, setSelectedCampaign] = useState<QualificationCampaign | null>(null)
  const [verificationResults, setVerificationResults] = useState<VerificationResult[]>([])
  const [artifacts, setArtifacts] = useState<EvidenceArtifact[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const generateMockCampaigns = (): QualificationCampaign[] => [
    {
      id: 'camp-1',
      name: 'P1_CORE Qualification Campaign',
      status: 'passed',
      version: 'dh/v1.0.0',
      startedAt: '2026-09-15T08:00:00Z',
      completedAt: '2026-09-15T09:45:00Z',
      phase: 8,
      totalPhases: 8,
      backend: 'QEMU/KVM',
      nodeCount: 3,
    },
    {
      id: 'camp-2',
      name: 'Chaos Framework M7 Validation',
      status: 'passed',
      version: 'dh/v1.0.0',
      startedAt: '2026-09-20T10:30:00Z',
      completedAt: '2026-09-20T13:20:00Z',
      phase: 7,
      totalPhases: 7,
      backend: 'Kubernetes',
      nodeCount: 5,
    },
    {
      id: 'camp-3',
      name: 'P1_MULTIPHYSICAL Qualification',
      status: 'in_progress',
      version: 'dh/v1.0.0',
      startedAt: '2026-09-25T14:00:00Z',
      completedAt: null,
      phase: 4,
      totalPhases: 8,
      backend: 'Physical Hosts',
      nodeCount: 6,
    },
  ]

  const generateMockVerificationResults = (campaignId: string): VerificationResult[] => {
    const baseTests = [
      { name: 'Signed intent acquisition and validation', assertion: 'Ed25519 signatures verified' },
      { name: 'Local policy matching with audit logging', assertion: 'Policy gates enforced per-host' },
      { name: 'BLAKE3 CAS with Merkle anti-entropy', assertion: 'Content addressing validated' },
      { name: 'Userspace WireGuard with signed bindings', assertion: 'Key bindings verified' },
      { name: 'Raft + mTLS control plane', assertion: 'Consensus achieved across nodes' },
      { name: 'ACME TLS integration', assertion: 'Certificate provisioning verified' },
      { name: 'Failure detection and reconciliation', assertion: 'Node failures detected within SLA' },
      { name: 'State machine verification', assertion: 'State transitions validated' },
    ]

    return baseTests.map((test, i) => ({
      id: `result-${i}`,
      campaignId,
      testNumber: i + 1,
      name: test.name,
      status: Math.random() > 0.05 ? 'passed' : 'failed',
      duration: Math.floor(Math.random() * 5000) + 500,
      timestamp: new Date(Date.now() - Math.random() * 3600000).toISOString(),
      assertion: test.assertion,
    }))
  }

  const generateMockArtifacts = (campaignId: string): EvidenceArtifact[] => [
    {
      id: `art-1-${campaignId}`,
      campaignId,
      type: 'source',
      digest: 'blake3:a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0',
      size: 2147483648,
      created: '2026-09-15T08:00:00Z',
    },
    {
      id: `art-2-${campaignId}`,
      campaignId,
      type: 'evidence',
      digest: 'blake3:x9y8z7w6v5u4t3s2r1q0p9o8n7m6l5k4j3i2h1g0f9e8d7c6b5a4z3y2x1w0',
      size: 536870912,
      created: '2026-09-15T09:00:00Z',
    },
    {
      id: `art-3-${campaignId}`,
      campaignId,
      type: 'signature',
      digest: 'blake3:sig_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d',
      size: 64,
      created: '2026-09-15T09:45:00Z',
      signer: 'Decentralized.Host Authority',
    },
  ]

  useEffect(() => {
    const loadEvidence = async () => {
      try {
        setLoading(true)
        const mockCampaigns = generateMockCampaigns()
        setCampaigns(mockCampaigns)
        setSelectedCampaign(mockCampaigns[0])
        setVerificationResults(generateMockVerificationResults(mockCampaigns[0].id))
        setArtifacts(generateMockArtifacts(mockCampaigns[0].id))
        setError(null)
      } catch (err) {
        setError('Failed to load evidence data')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadEvidence()
  }, [])

  const handleSelectCampaign = (campaign: QualificationCampaign) => {
    setSelectedCampaign(campaign)
    setVerificationResults(generateMockVerificationResults(campaign.id))
    setArtifacts(generateMockArtifacts(campaign.id))
  }

  if (error) {
    return (
      <AppLayout title="Evidence" subtitle="Qualification and verification records">
        <ErrorState
          title="Failed to Load Evidence"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Evidence" subtitle="Qualification and verification records">
        <Loading message="Loading evidence data..." />
      </AppLayout>
    )
  }

  const passedCampaigns = campaigns.filter((c) => c.status === 'passed').length
  const passedResults = verificationResults.filter((r) => r.status === 'passed').length

  return (
    <AppLayout
      title="Evidence"
      subtitle={`${passedCampaigns} passed campaigns • ${passedResults}/${verificationResults.length} tests passed`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Campaigns</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{campaigns.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">{passedCampaigns} passed</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Test Pass Rate</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">
                  {passedResults > 0 ? Math.round((passedResults / verificationResults.length) * 100) : 0}%
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">
                {passedResults} of {verificationResults.length}
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Evidence Artifacts</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">{artifacts.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">
                {(artifacts.reduce((sum, a) => sum + a.size, 0) / 1024 / 1024 / 1024).toFixed(1)} GB total
              </p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Verification Status</p>
              <div className="flex items-end gap-2">
                <span className="text-2xl font-bold text-success-500">✓ Valid</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">All signatures verified</p>
            </div>
          </Card>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Campaigns List */}
          <Card variant="glass">
            <div className="p-6 border-b border-neutral-700">
              <h3 className="text-lg font-semibold text-white">Qualification Campaigns</h3>
            </div>
            <div className="divide-y divide-neutral-700 max-h-96 overflow-y-auto">
              {campaigns.map((campaign) => (
                <button
                  key={campaign.id}
                  onClick={() => handleSelectCampaign(campaign)}
                  className={`w-full text-left p-4 transition ${
                    selectedCampaign?.id === campaign.id
                      ? 'bg-primary-500/20 border-l-2 border-primary-500'
                      : 'hover:bg-neutral-800/30'
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1">
                      <p className="font-medium text-neutral-100 text-sm">{campaign.name}</p>
                      <p className="text-xs text-neutral-500 mt-1">Phase {campaign.phase}/{campaign.totalPhases}</p>
                    </div>
                    <Badge
                      status={
                        campaign.status === 'passed'
                          ? 'active'
                          : campaign.status === 'in_progress'
                            ? 'pending'
                            : 'error'
                      }
                      size="sm"
                    />
                  </div>
                  <p className="text-xs text-neutral-400 mt-2">{campaign.backend}</p>
                </button>
              ))}
            </div>
          </Card>

          {/* Selected Campaign Details */}
          {selectedCampaign && (
            <div className="lg:col-span-2 space-y-6">
              {/* Campaign Info */}
              <Card variant="glass">
                <div className="p-6 border-b border-neutral-700">
                  <h3 className="text-lg font-semibold text-white mb-4">
                    {selectedCampaign.name}
                  </h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-xs text-neutral-400 mb-1">Status</p>
                      <Badge
                        status={
                          selectedCampaign.status === 'passed'
                            ? 'active'
                            : selectedCampaign.status === 'in_progress'
                              ? 'pending'
                              : 'error'
                        }
                        size="sm"
                      />
                    </div>
                    <div>
                      <p className="text-xs text-neutral-400 mb-1">Backend</p>
                      <p className="text-sm font-medium text-neutral-200">
                        {selectedCampaign.backend}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-neutral-400 mb-1">Nodes</p>
                      <p className="text-sm font-medium text-neutral-200">
                        {selectedCampaign.nodeCount}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-neutral-400 mb-1">Version</p>
                      <p className="text-sm font-medium text-neutral-200">
                        {selectedCampaign.version}
                      </p>
                    </div>
                  </div>
                </div>
                <div className="p-6 space-y-2">
                  <p className="text-xs text-neutral-400">Started: {new Date(selectedCampaign.startedAt).toLocaleString()}</p>
                  {selectedCampaign.completedAt && (
                    <p className="text-xs text-neutral-400">
                      Completed: {new Date(selectedCampaign.completedAt).toLocaleString()}
                    </p>
                  )}
                  <div className="pt-2">
                    <p className="text-xs text-neutral-400 mb-2">
                      Phase Progress: {selectedCampaign.phase}/{selectedCampaign.totalPhases}
                    </p>
                    <div className="w-full bg-neutral-700 rounded-full h-2">
                      <div
                        className="bg-primary-500 h-2 rounded-full transition-all"
                        style={{
                          width: `${(selectedCampaign.phase / selectedCampaign.totalPhases) * 100}%`,
                        }}
                      />
                    </div>
                  </div>
                </div>
              </Card>

              {/* Verification Results */}
              <Card variant="glass">
                <div className="p-6 border-b border-neutral-700">
                  <h3 className="text-lg font-semibold text-white">Test Results</h3>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="border-b border-neutral-700">
                      <tr>
                        <th className="px-6 py-3 text-left text-neutral-400 font-medium">#</th>
                        <th className="px-6 py-3 text-left text-neutral-400 font-medium">Test Name</th>
                        <th className="px-6 py-3 text-left text-neutral-400 font-medium">Assertion</th>
                        <th className="px-6 py-3 text-left text-neutral-400 font-medium">Duration</th>
                        <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-neutral-700">
                      {verificationResults.slice(0, 8).map((result) => (
                        <tr key={result.id} className="hover:bg-neutral-800/30">
                          <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                            {result.testNumber}
                          </td>
                          <td className="px-6 py-3 text-neutral-100 text-xs font-medium max-w-xs truncate">
                            {result.name}
                          </td>
                          <td className="px-6 py-3 text-neutral-400 text-xs">
                            {result.assertion}
                          </td>
                          <td className="px-6 py-3 text-neutral-400 text-xs font-mono">
                            {result.duration}ms
                          </td>
                          <td className="px-6 py-3">
                            <Badge
                              status={
                                result.status === 'passed'
                                  ? 'active'
                                  : result.status === 'skipped'
                                    ? 'pending'
                                    : 'error'
                              }
                              size="sm"
                            />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Card>

              {/* Evidence Artifacts */}
              <Card variant="glass">
                <div className="p-6 border-b border-neutral-700">
                  <h3 className="text-lg font-semibold text-white">Evidence Artifacts</h3>
                </div>
                <div className="divide-y divide-neutral-700">
                  {artifacts.map((artifact) => (
                    <div key={artifact.id} className="p-6 space-y-3">
                      <div className="flex items-start justify-between">
                        <div>
                          <p className="text-sm font-medium text-neutral-100 capitalize">
                            {artifact.type} Artifact
                          </p>
                          <p className="text-xs text-neutral-500 mt-1">
                            {(artifact.size / 1024 / 1024).toFixed(1)} MB
                          </p>
                        </div>
                        {artifact.signer && (
                          <Badge
                            status="active"
                            size="sm"
                          />
                        )}
                      </div>
                      <div className="bg-neutral-800 rounded p-3">
                        <p className="text-xs font-mono text-neutral-300 break-all">
                          {artifact.digest}
                        </p>
                      </div>
                      <div className="flex items-center justify-between text-xs">
                        <p className="text-neutral-500">
                          Created: {new Date(artifact.created).toLocaleString()}
                        </p>
                        {artifact.signer && (
                          <p className="text-primary-400">✓ {artifact.signer}</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </Card>
            </div>
          )}
        </div>
      </div>
    </AppLayout>
  )
}

export default Evidence
