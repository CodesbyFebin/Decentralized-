import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { ErrorState } from '@/components/ErrorState'
import { apiClient } from '@/lib/api'

interface DNSRecord {
  id: string
  type: 'A' | 'AAAA' | 'CNAME' | 'MX' | 'TXT' | 'NS' | 'SRV'
  name: string
  value: string
  ttl: number
  status: 'active' | 'pending' | 'error'
}

interface Domain {
  id: string
  name: string
  registrar: string
  status: 'active' | 'inactive' | 'expiring'
  sslStatus: 'secured' | 'unsecured' | 'expiring'
  dnsRecords: DNSRecord[]
  expiryDate: string
  lastChecked: string
  registrationDate: string
  autoRenew: boolean
}

interface WHOISInfo {
  registrar: string
  registrant: string
  expiryDate: string
  nameServers: string[]
  createdDate: string
  updatedDate: string
}

const Domains: React.FC = () => {
  const [domains, setDomains] = useState<Domain[]>([])
  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddDomain, setShowAddDomain] = useState(false)
  const [showDNSEditor, setShowDNSEditor] = useState(false)
  const [showWHOIS, setShowWHOIS] = useState(false)
  const [newDomainName, setNewDomainName] = useState('')
  const [newDNSRecord, setNewDNSRecord] = useState({ type: 'A' as const, name: '', value: '', ttl: 3600 })

  const generateMockDomains = (): Domain[] => [
    {
      id: 'domain-1',
      name: 'decentralized.host',
      registrar: 'Namecheap',
      status: 'active',
      sslStatus: 'secured',
      expiryDate: '2027-09-30T00:00:00Z',
      lastChecked: '2026-09-30T08:30:00Z',
      registrationDate: '2023-09-30T00:00:00Z',
      autoRenew: true,
      dnsRecords: [
        { id: 'dns-1', type: 'A', name: '@', value: '192.168.1.100', ttl: 3600, status: 'active' },
        { id: 'dns-2', type: 'A', name: 'www', value: '192.168.1.100', ttl: 3600, status: 'active' },
        { id: 'dns-3', type: 'CNAME', name: 'api', value: 'decentralized.host', ttl: 3600, status: 'active' },
        { id: 'dns-4', type: 'MX', name: '@', value: '10 mail.decentralized.host', ttl: 1800, status: 'active' },
      ],
    },
    {
      id: 'domain-2',
      name: 'infrastructure.dev',
      registrar: 'GoDaddy',
      status: 'active',
      sslStatus: 'secured',
      expiryDate: '2026-12-15T00:00:00Z',
      lastChecked: '2026-09-29T22:15:00Z',
      registrationDate: '2024-12-15T00:00:00Z',
      autoRenew: false,
      dnsRecords: [
        { id: 'dns-5', type: 'A', name: '@', value: '192.168.2.50', ttl: 3600, status: 'active' },
        { id: 'dns-6', type: 'CNAME', name: 'cdn', value: 'cdn.provider.com', ttl: 3600, status: 'active' },
      ],
    },
    {
      id: 'domain-3',
      name: 'operations.io',
      registrar: 'Vercel Domains',
      status: 'active',
      sslStatus: 'expiring',
      expiryDate: '2026-11-01T00:00:00Z',
      lastChecked: '2026-09-28T16:45:00Z',
      registrationDate: '2024-11-01T00:00:00Z',
      autoRenew: true,
      dnsRecords: [
        { id: 'dns-7', type: 'A', name: '@', value: '192.168.3.25', ttl: 300, status: 'active' },
      ],
    },
  ]

  const generateWHOISInfo = (domain: Domain): WHOISInfo => ({
    registrar: domain.registrar,
    registrant: 'Decentralized Infrastructure Admin',
    expiryDate: domain.expiryDate,
    nameServers: ['ns1.decentralized.host', 'ns2.decentralized.host'],
    createdDate: domain.registrationDate,
    updatedDate: new Date(new Date(domain.registrationDate).getTime() + 30 * 24 * 60 * 60 * 1000).toISOString(),
  })

  useEffect(() => {
    const loadDomains = async () => {
      try {
        setLoading(true)
        const mockDomains = generateMockDomains()
        setDomains(mockDomains)
        setSelectedDomain(mockDomains[0])
        setError(null)
      } catch (err) {
        setError('Failed to load domains')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    loadDomains()
  }, [])

  if (error) {
    return (
      <AppLayout title="Domains" subtitle="DNS and domain management">
        <ErrorState
          title="Failed to Load Domains"
          message={error}
          onRetry={() => window.location.reload()}
        />
      </AppLayout>
    )
  }

  if (loading) {
    return (
      <AppLayout title="Domains" subtitle="DNS and domain management">
        <Loading message="Loading domains..." />
      </AppLayout>
    )
  }

  const activeDomains = domains.filter((d) => d.status === 'active').length
  const securedDomains = domains.filter((d) => d.sslStatus === 'secured').length
  const totalDNSRecords = domains.reduce((sum, d) => sum + d.dnsRecords.length, 0)
  const healthStatus = domains.every((d) => d.status === 'active' && d.sslStatus !== 'unsecured') ? 'healthy' : 'needs_attention'

  const handleAddDomain = async () => {
    if (!newDomainName.trim()) return
    try {
      await apiClient.post('/domains', { name: newDomainName })
      setNewDomainName('')
      setShowAddDomain(false)
      const mockDomains = generateMockDomains()
      setDomains(mockDomains)
    } catch (err) {
      console.error('Failed to add domain:', err)
    }
  }

  const handleAddDNSRecord = async () => {
    if (!selectedDomain || !newDNSRecord.name || !newDNSRecord.value) return
    try {
      await apiClient.post(`/domains/${selectedDomain.id}/dns-records`, newDNSRecord)
      setNewDNSRecord({ type: 'A', name: '', value: '', ttl: 3600 })
      const mockDomains = generateMockDomains()
      setDomains(mockDomains)
      const updated = mockDomains.find((d) => d.id === selectedDomain.id)
      if (updated) setSelectedDomain(updated)
    } catch (err) {
      console.error('Failed to add DNS record:', err)
    }
  }

  const handleDeleteDNSRecord = async (recordId: string) => {
    if (!selectedDomain) return
    try {
      await apiClient.delete(`/domains/${selectedDomain.id}/dns-records/${recordId}`)
      const mockDomains = generateMockDomains()
      setDomains(mockDomains)
      const updated = mockDomains.find((d) => d.id === selectedDomain.id)
      if (updated) setSelectedDomain(updated)
    } catch (err) {
      console.error('Failed to delete DNS record:', err)
    }
  }

  const handleRenewDomain = async (domainId: string) => {
    try {
      await apiClient.post(`/domains/${domainId}/renew`, {})
    } catch (err) {
      console.error('Failed to renew domain:', err)
    }
  }

  const whoisInfo = selectedDomain ? generateWHOISInfo(selectedDomain) : null

  return (
    <AppLayout
      title="Domains"
      subtitle={`${activeDomains} active domains • ${securedDomains} SSL secured`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Domains Managed</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-primary-500">{domains.length}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">{activeDomains} active</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">SSL Status</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500">{securedDomains}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">domains secured</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">DNS Records</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-secondary-500">{totalDNSRecords}</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">across all domains</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Health Status</p>
              <div className="flex items-end gap-2">
                <span className="text-3xl font-bold text-success-500 capitalize">
                  {healthStatus === 'healthy' ? '✓ Good' : '⚠ Check'}
                </span>
              </div>
              <p className="text-xs text-neutral-500 mt-2">all systems nominal</p>
            </div>
          </Card>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Domains List */}
          <div className="lg:col-span-1">
            <Card variant="glass">
              <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
                <h3 className="text-lg font-semibold text-white">Domains</h3>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => setShowAddDomain(!showAddDomain)}
                >
                  + Add
                </Button>
              </div>

              {showAddDomain && (
                <div className="p-4 border-b border-neutral-700 bg-neutral-800/50 space-y-2">
                  <input
                    type="text"
                    value={newDomainName}
                    onChange={(e) => setNewDomainName(e.target.value)}
                    placeholder="example.com"
                    className="w-full px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none text-sm"
                  />
                  <div className="flex gap-2">
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={handleAddDomain}
                      disabled={!newDomainName.trim()}
                      className="flex-1"
                    >
                      Add
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowAddDomain(false)}
                      className="flex-1"
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              )}

              <div className="divide-y divide-neutral-700">
                {domains.map((domain) => (
                  <button
                    key={domain.id}
                    onClick={() => setSelectedDomain(domain)}
                    className={`w-full text-left p-4 transition ${
                      selectedDomain?.id === domain.id
                        ? 'bg-neutral-700 border-l-2 border-primary-500'
                        : 'hover:bg-neutral-800/30'
                    }`}
                  >
                    <p className="font-medium text-neutral-100 text-sm">{domain.name}</p>
                    <div className="flex gap-2 mt-2">
                      <Badge status={domain.status === 'active' ? 'active' : 'pending'} size="sm" />
                      <Badge
                        status={domain.sslStatus === 'secured' ? 'active' : 'error'}
                        size="sm"
                      />
                    </div>
                  </button>
                ))}
              </div>
            </Card>
          </div>

          {/* Domain Details */}
          <div className="lg:col-span-2 space-y-6">
            {selectedDomain && (
              <>
                {/* Domain Info Card */}
                <Card variant="glass">
                  <div className="p-6 border-b border-neutral-700">
                    <div className="flex items-start justify-between">
                      <div>
                        <h3 className="text-2xl font-bold text-white mb-2">{selectedDomain.name}</h3>
                        <p className="text-neutral-400 text-sm">Registered with {selectedDomain.registrar}</p>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          variant="primary"
                          size="sm"
                          onClick={() => setShowWHOIS(!showWHOIS)}
                        >
                          WHOIS
                        </Button>
                        {selectedDomain.status !== 'active' && (
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() => handleRenewDomain(selectedDomain.id)}
                          >
                            Renew
                          </Button>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="p-6 space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Registration Date</p>
                        <p className="text-sm font-medium text-neutral-200">
                          {new Date(selectedDomain.registrationDate).toLocaleDateString()}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Expiry Date</p>
                        <p className="text-sm font-medium text-neutral-200">
                          {new Date(selectedDomain.expiryDate).toLocaleDateString()}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Auto-Renew</p>
                        <p className="text-sm font-medium text-neutral-200">
                          {selectedDomain.autoRenew ? 'Enabled' : 'Disabled'}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Last Health Check</p>
                        <p className="text-sm font-medium text-neutral-200">
                          {new Date(selectedDomain.lastChecked).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                  </div>
                </Card>

                {/* WHOIS Info */}
                {showWHOIS && whoisInfo && (
                  <Card variant="glass">
                    <div className="p-6 border-b border-neutral-700">
                      <h3 className="text-lg font-semibold text-white">WHOIS Information</h3>
                    </div>
                    <div className="p-6 space-y-4">
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Registrar</p>
                        <p className="text-sm font-medium text-neutral-200">{whoisInfo.registrar}</p>
                      </div>
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Registrant</p>
                        <p className="text-sm font-medium text-neutral-200">{whoisInfo.registrant}</p>
                      </div>
                      <div>
                        <p className="text-xs text-neutral-400 mb-1">Name Servers</p>
                        <div className="space-y-1">
                          {whoisInfo.nameServers.map((ns, i) => (
                            <p key={i} className="text-sm font-mono text-neutral-300">
                              {ns}
                            </p>
                          ))}
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-4 pt-2">
                        <div>
                          <p className="text-xs text-neutral-400 mb-1">Created</p>
                          <p className="text-sm font-medium text-neutral-200">
                            {new Date(whoisInfo.createdDate).toLocaleDateString()}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs text-neutral-400 mb-1">Updated</p>
                          <p className="text-sm font-medium text-neutral-200">
                            {new Date(whoisInfo.updatedDate).toLocaleDateString()}
                          </p>
                        </div>
                      </div>
                    </div>
                  </Card>
                )}

                {/* DNS Records */}
                <Card variant="glass">
                  <div className="p-6 border-b border-neutral-700 flex items-center justify-between">
                    <h3 className="text-lg font-semibold text-white">
                      DNS Records ({selectedDomain.dnsRecords.length})
                    </h3>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => setShowDNSEditor(!showDNSEditor)}
                    >
                      + Add Record
                    </Button>
                  </div>

                  {showDNSEditor && (
                    <div className="p-6 border-b border-neutral-700 bg-neutral-800/50 space-y-3">
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-xs font-medium text-neutral-300 mb-1">
                            Type
                          </label>
                          <select
                            value={newDNSRecord.type}
                            onChange={(e) => setNewDNSRecord({ ...newDNSRecord, type: e.target.value as any })}
                            className="w-full px-2 py-1 bg-neutral-700 border border-neutral-600 rounded text-white text-sm focus:border-primary-500 focus:outline-none"
                          >
                            <option>A</option>
                            <option>AAAA</option>
                            <option>CNAME</option>
                            <option>MX</option>
                            <option>TXT</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-neutral-300 mb-1">
                            Name
                          </label>
                          <input
                            type="text"
                            value={newDNSRecord.name}
                            onChange={(e) => setNewDNSRecord({ ...newDNSRecord, name: e.target.value })}
                            placeholder="@"
                            className="w-full px-2 py-1 bg-neutral-700 border border-neutral-600 rounded text-white placeholder-neutral-500 text-sm focus:border-primary-500 focus:outline-none"
                          />
                        </div>
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-neutral-300 mb-1">
                          Value
                        </label>
                        <input
                          type="text"
                          value={newDNSRecord.value}
                          onChange={(e) => setNewDNSRecord({ ...newDNSRecord, value: e.target.value })}
                          placeholder="192.168.1.1"
                          className="w-full px-2 py-1 bg-neutral-700 border border-neutral-600 rounded text-white placeholder-neutral-500 text-sm focus:border-primary-500 focus:outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-neutral-300 mb-1">
                          TTL
                        </label>
                        <input
                          type="number"
                          value={newDNSRecord.ttl}
                          onChange={(e) => setNewDNSRecord({ ...newDNSRecord, ttl: parseInt(e.target.value) })}
                          className="w-full px-2 py-1 bg-neutral-700 border border-neutral-600 rounded text-white text-sm focus:border-primary-500 focus:outline-none"
                        />
                      </div>
                      <div className="flex gap-2">
                        <Button
                          variant="primary"
                          size="sm"
                          onClick={handleAddDNSRecord}
                          disabled={!newDNSRecord.name || !newDNSRecord.value}
                          className="flex-1"
                        >
                          Add
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setShowDNSEditor(false)}
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
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">Type</th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">Name</th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">Value</th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">TTL</th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">Status</th>
                          <th className="px-6 py-3 text-left text-neutral-400 font-medium">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-neutral-700">
                        {selectedDomain.dnsRecords.map((record) => (
                          <tr key={record.id} className="hover:bg-neutral-800/30">
                            <td className="px-6 py-3 text-neutral-100 font-medium">{record.type}</td>
                            <td className="px-6 py-3 text-neutral-400 font-mono">{record.name}</td>
                            <td className="px-6 py-3 text-neutral-400 font-mono text-xs break-all">
                              {record.value}
                            </td>
                            <td className="px-6 py-3 text-neutral-400 text-sm">{record.ttl}</td>
                            <td className="px-6 py-3">
                              <Badge
                                status={record.status === 'active' ? 'active' : 'pending'}
                                size="sm"
                              />
                            </td>
                            <td className="px-6 py-3 flex gap-2">
                              <button
                                onClick={() => handleDeleteDNSRecord(record.id)}
                                className="text-xs px-2 py-1 bg-error-600 hover:bg-error-500 rounded text-white"
                              >
                                Delete
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Card>
              </>
            )}
          </div>
        </div>
      </div>
    </AppLayout>
  )
}

export default Domains
