import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface Certificate {
  id: string
  domain: string
  issuer: string
  issuedDate: string
  expiryDate: string
  status: 'active' | 'expiring_soon' | 'expired'
  fingerprint: string
}

interface SecurityPolicy {
  id: string
  name: string
  type: 'firewall' | 'rate_limit' | 'encryption' | 'authentication'
  status: 'active' | 'inactive'
  created: string
  lastModified: string
  description: string
}

const Security: React.FC = () => {
  const [certificates, setCertificates] = useState<Certificate[]>([])
  const [policies, setPolicies] = useState<SecurityPolicy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddCert, setShowAddCert] = useState(false)
  const [showAddPolicy, setShowAddPolicy] = useState(false)
  const [newCertDomain, setNewCertDomain] = useState('')
  const [newPolicyName, setNewPolicyName] = useState('')
  const [newPolicyType, setNewPolicyType] = useState<'firewall' | 'rate_limit' | 'encryption' | 'authentication'>('firewall')

  const generateMockCertificates = (): Certificate[] => {
    const now = new Date()
    return [
      {
        id: 'cert-1',
        domain: 'decentralized.host',
        issuer: 'Let\'s Encrypt',
        issuedDate: '2026-09-01T00:00:00Z',
        expiryDate: '2026-12-01T00:00:00Z',
        status: 'active',
        fingerprint: 'a3:f2:e1:d4:c5:b6:a7:98',
      },
      {
        id: 'cert-2',
        domain: 'api.decentralized.host',
        issuer: 'Let\'s Encrypt',
        issuedDate: '2026-08-15T00:00:00Z',
        expiryDate: '2026-11-15T00:00:00Z',
        status: 'active',
        fingerprint: 'b4:e3:f2:a1:d5:c6:b7:09',
      },
      {
        id: 'cert-3',
        domain: 'dashboard.decentralized.host',
        issuer: 'DigiCert',
        issuedDate: '2026-07-20T00:00:00Z',
        expiryDate: '2026-10-20T00:00:00Z',
        status: 'expiring_soon',
        fingerprint: 'c5:f4:a3:b2:e6:d7:c8:1a',
      },
      {
        id: 'cert-4',
        domain: 'legacy.decentralized.host',
        issuer: 'GlobalSign',
        issuedDate: '2025-09-01T00:00:00Z',
        expiryDate: '2026-08-15T00:00:00Z',
        status: 'expired',
        fingerprint: 'd6:a5:b4:c3:f7:e8:d9:2b',
      },
    ]
  }

  const generateMockPolicies = (): SecurityPolicy[] => [
    {
      id: 'policy-1',
      name: 'DDoS Protection',
      type: 'firewall',
      status: 'active',
      created: '2026-09-01T10:00:00Z',
      lastModified: '2026-09-28T14:30:00Z',
      description: 'Rate limiting and geo-blocking for DDoS mitigation',
    },
    {
      id: 'policy-2',
      name: 'API Rate Limiting',
      type: 'rate_limit',
      status: 'active',
      created: '2026-08-15T08:00:00Z',
      lastModified: '2026-09-25T11:15:00Z',
      description: '1000 requests per minute per IP, burst to 2000',
    },
    {
      id: 'policy-3',
      name: 'TLS 1.3 Enforcement',
      type: 'encryption',
      status: 'active',
      created: '2026-07-01T09:00:00Z',
      lastModified: '2026-09-30T09:00:00Z',
      description: 'Enforce TLS 1.3 minimum, disable older protocols',
    },
    {
      id: 'policy-4',
      name: 'OAuth 2.0 Authentication',
      type: 'authentication',
      status: 'inactive',
      created: '2026-06-10T12:00:00Z',
      lastModified: '2026-09-20T16:45:00Z',
      description: 'OAuth 2.0 with JWT tokens for API authentication',
    },
  ]

  useEffect(() => {
    const loadSecurity = async () => {
      try {
        setLoading(true)
        const mockCerts = generateMockCertificates()
        const mockPolicies = generateMockPolicies()
        setCertificates(mockCerts)
        setPolicies(mockPolicies)
        setError(null)
      } catch (err) {
        setError('Failed to load security data')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadSecurity()
  }, [])

  if (error) {
    return (
      <AppLayout title="Security" subtitle="SSL certificates and security policies">
        <ErrorState
          title="Failed to Load Security Data"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Security" subtitle="SSL certificates and security policies">
        <Loading message="Loading security data..." />
      </AppLayout>
    )
  }

  const activeCerts = certificates.filter((c) => c.status === 'active').length
  const expiringCerts = certificates.filter((c) => c.status === 'expiring_soon').length
  const activePolicies = policies.filter((p) => p.status === 'active').length
  const complianceScore = Math.round((activeCerts / certificates.length) * 100)

  const handleAddCertificate = async () => {
    if (!newCertDomain.trim()) return
    try {
      await apiClient.post('/security/certificates', { domain: newCertDomain })
      setNewCertDomain('')
      setShowAddCert(false)
      const mockCerts = generateMockCertificates()
      setCertificates(mockCerts)
    } catch (err) {
      console.error('Failed to add certificate:', err)
    }
  }

  const handleAddPolicy = async () => {
    if (!newPolicyName.trim()) return
    try {
      await apiClient.post('/security/policies', { name: newPolicyName, type: newPolicyType })
      setNewPolicyName('')
      setNewPolicyType('firewall')
      setShowAddPolicy(false)
      const mockPolicies = generateMockPolicies()
      setPolicies(mockPolicies)
    } catch (err) {
      console.error('Failed to add policy:', err)
    }
  }

  const handleRenewCertificate = async (certId: string) => {
    try {
      await apiClient.post(`/security/certificates/${certId}/renew`, {})
    } catch (err) {
      console.error('Failed to renew certificate:', err)
    }
  }

  const handleTogglePolicy = async (policyId: string, newStatus: 'active' | 'inactive') => {
    try {
      await apiClient.patch(`/security/policies/${policyId}`, { status: newStatus })
    } catch (err) {
      console.error('Failed to toggle policy:', err)
    }
  }

  return (
    <AppLayout
      title="Security"
      subtitle={`${activeCerts} active certificates • ${activePolicies} active policies`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Active Certificates</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{activeCerts}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">of {certificates.length} total</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Security Policies</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">{activePolicies}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">active policies</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Compliance Score</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">{complianceScore}%</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">Certificate coverage</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Renewal Due Soon</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-warning-500">{expiringCerts}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">within 30 days</p>
            </div>
          </Card>
        </div>

        {/* Certificates */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
            <h3 className="text-lg font-semibold text-white">Certificates ({certificates.length})</h3>
            <Button
              variant="primary"
              size="md"
              onClick={() => setShowAddCert(!showAddCert)}
            >
              + Add Certificate
            </Button>
          </div>

          {showAddCert && (
            <div className="p-6 border-b border-neutral-700 bg-neutral-800/50 space-y-3">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Domain
                </label>
                <input
                  type="text"
                  value={newCertDomain}
                  onChange={(e) => setNewCertDomain(e.target.value)}
                  placeholder="e.g., api.example.com"
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>
              <div className="flex gap-3">
                <Button
                  variant="primary"
                  size="md"
                  onClick={handleAddCertificate}
                  disabled={!newCertDomain.trim()}
                  className="flex-1"
                >
                  Request Certificate
                </Button>
                <Button
                  variant="ghost"
                  size="md"
                  onClick={() => setShowAddCert(false)}
                  className="flex-1"
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Domain</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Issuer</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Issued</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Expires</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {certificates.map((cert) => (
                  <tr key={cert.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{cert.domain}</td>
                    <td className="px-6 py-3 text-neutral-400">{cert.issuer}</td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                      {new Date(cert.issuedDate).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                      {new Date(cert.expiryDate).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-3">
                      <Badge
                        status={
                          cert.status === 'active'
                            ? 'active'
                            : cert.status === 'expiring_soon'
                              ? 'pending'
                              : 'error'
                        }
                        size="sm"
                      />
                    </td>
                    <td className="px-6 py-3 flex gap-2">
                      {(cert.status === 'active' || cert.status === 'expiring_soon') && (
                        <button
                          onClick={() => handleRenewCertificate(cert.id)}
                          className="text-xs px-2 py-1 bg-primary-600 hover:bg-primary-500 rounded text-white"
                        >
                          Renew
                        </button>
                      )}
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        View
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Security Policies */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
            <h3 className="text-lg font-semibold text-white">Security Policies ({policies.length})</h3>
            <Button
              variant="primary"
              size="md"
              onClick={() => setShowAddPolicy(!showAddPolicy)}
            >
              + Add Policy
            </Button>
          </div>

          {showAddPolicy && (
            <div className="p-6 border-b border-neutral-700 bg-neutral-800/50 space-y-3">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Policy Name
                </label>
                <input
                  type="text"
                  value={newPolicyName}
                  onChange={(e) => setNewPolicyName(e.target.value)}
                  placeholder="e.g., WAF Rules"
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Policy Type
                </label>
                <select
                  value={newPolicyType}
                  onChange={(e) => setNewPolicyType(e.target.value as any)}
                  className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="firewall">Firewall</option>
                  <option value="rate_limit">Rate Limiting</option>
                  <option value="encryption">Encryption</option>
                  <option value="authentication">Authentication</option>
                </select>
              </div>
              <div className="flex gap-3">
                <Button
                  variant="primary"
                  size="md"
                  onClick={handleAddPolicy}
                  disabled={!newPolicyName.trim()}
                  className="flex-1"
                >
                  Create Policy
                </Button>
                <Button
                  variant="ghost"
                  size="md"
                  onClick={() => setShowAddPolicy(false)}
                  className="flex-1"
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-neutral-700">
                <tr>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Name</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Type</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Description</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Modified</th>
                  <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-700">
                {policies.map((policy) => (
                  <tr key={policy.id} className="hover:bg-neutral-800/30">
                    <td className="px-6 py-3 text-neutral-100 font-medium">{policy.name}</td>
                    <td className="px-6 py-3 text-neutral-400">
                      <span className="px-2 py-1 bg-neutral-700 rounded text-xs capitalize">
                        {policy.type.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="px-6 py-3 text-neutral-400 text-xs max-w-xs truncate">
                      {policy.description}
                    </td>
                    <td className="px-6 py-3">
                      <Badge
                        status={policy.status === 'active' ? 'active' : 'pending'}
                        size="sm"
                      />
                    </td>
                    <td className="px-6 py-3 text-neutral-400 font-mono text-xs">
                      {new Date(policy.lastModified).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-3 flex gap-2">
                      <button
                        onClick={() => handleTogglePolicy(policy.id, policy.status === 'active' ? 'inactive' : 'active')}
                        className={`text-xs px-2 py-1 rounded ${
                          policy.status === 'active'
                            ? 'bg-error-600 hover:bg-error-500 text-white'
                            : 'bg-success-600 hover:bg-success-500 text-white'
                        }`}
                      >
                        {policy.status === 'active' ? 'Disable' : 'Enable'}
                      </button>
                      <button className="text-xs px-2 py-1 bg-neutral-700 hover:bg-neutral-600 rounded text-neutral-300">
                        Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Security
