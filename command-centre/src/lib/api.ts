import {
  PlatformCapabilities,
  Application,
  Deployment,
  NodeInfo,
  StorageObject,
  StorageBucket,
  DomainRecord,
  SSLCertificate,
  SecurityEvaluation,
  TeamMember,
  TeamPermissionMatrix,
  EvidenceRecord,
  PlatformEvent,
  CopilotMessage,
  ComputeSummary,
  DePINIntegration,
  SelfHostingBenefit,
  FleetOverview,
  StorageFleetOverview,
  DeployOverview
} from '../types/platform';

const API_BASE = '/api/v1';

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers
    },
    ...options
  });

  if (!res.ok) {
    let errorDetail = 'API request failed';
    try {
      const errJson = await res.json();
      errorDetail = errJson?.error?.message || errorDetail;
    } catch {}
    throw new Error(errorDetail);
  }

  return res.json();
}

export const api = {
  getHealth: () =>
    request<{ status: string; health: { healthy: boolean; mode: string; message: string } }>('/health'),

  getCapabilities: () =>
    request<{ capabilities: PlatformCapabilities }>('/capabilities'),

  getOverview: () =>
    request<{
      platformStatus: {
        state: string;
        onlineNodesCount: number;
        totalNodesCount: number;
        uptimePercent: number;
        avgLatencyMs: number;
      };
      metrics: {
        websitesAndApps: number;
        domains: number;
        sslCertificates: number;
        storageUsedGb: number;
        storageTotalGb: number;
        totalVisitors30d: number;
        bandwidthUsedTb: number;
        bandwidthTotalTb: number;
        cpuPercent: number;
        cpuCoresUsed: number;
        cpuCoresTotal: number;
        memoryPercent: number;
        memoryUsedGb: number;
        memoryTotalGb: number;
      };
      quickActions: { id: string; label: string; description: string; route: string }[];
      recentActivity: PlatformEvent[];
      topApps: { id: string; name: string; domain: string; visitors: number; status: string }[];
    }>('/overview'),

  getApps: () => request<{ data: Application[] }>('/apps'),
  createApp: (data: Partial<Application>) =>
    request<{ data: Application }>('/apps', {
      method: 'POST',
      body: JSON.stringify(data)
    }),
  deleteApp: (id: string) =>
    request<{ message: string; id: string }>(`/apps/${id}`, { method: 'DELETE' }),

  getDeployments: () => request<{ data: Deployment[] }>('/deployments'),
  createDeployment: (data: any) =>
    request<{ data: Deployment }>('/deployments', {
      method: 'POST',
      body: JSON.stringify(data)
    }),

  getNodes: () =>
    request<{
      summary: {
        total: number;
        online: number;
        degraded: number;
        offline: number;
        regions: number;
        totalStorageTb: number;
        usedStorageTb?: number;
        totalBandwidthTb: number;
        healthyPercent: number;
        totalGpus?: number;
        activeGpus?: number;
        totalGpuVramGb?: number;
        totalCpuCores?: number;
        totalMemoryGb?: number;
        depinNetworksConnected?: number;
        totalRewardsEarnedDH?: number;
        averageUptimeScore?: number;
      };
      compute?: ComputeSummary;
      data: NodeInfo[];
    }>('/nodes'),

  getComputeSummary: () =>
    request<{
      status: string;
      data: {
        compute: ComputeSummary;
        depin: DePINIntegration[];
        benefits: SelfHostingBenefit[];
      };
    }>('/compute'),

  getFleet: () => request<{ status: string; data: FleetOverview }>('/fleet'),

  getStorageFleet: () => request<{ status: string; data: StorageFleetOverview }>('/storage/fleet'),

  getDeployOverview: () => request<{ status: string; data: DeployOverview }>('/deploy/overview'),

  getDePINIntegrations: () =>
    request<{ status: string; data: DePINIntegration[] }>('/depin'),

  getSelfHostingBenefits: () =>
    request<{ status: string; data: SelfHostingBenefit[] }>('/benefits'),

  executeNodeOperation: (id: string, operation: 'DRAIN' | 'CORDON' | 'UNCORDON' | 'RESTART') =>
    request<{ data: NodeInfo }>(`/nodes/${id}/operations`, {
      method: 'POST',
      body: JSON.stringify({ operation })
    }),

  getStorage: () =>
    request<{
      summary: {
        totalStorageTb: number;
        totalFilesCount: number;
        replicatedCopies: string;
        storageCostUsd: number;
        storageHealthPercent: number;
        regionalDistribution: { region: string; percent: number; storageTb: number; nodes: number }[];
      };
      buckets: StorageBucket[];
      objects: StorageObject[];
    }>('/storage'),

  uploadStorageFile: (data: { name: string; sizeBytes?: number; bucket?: string; type?: string }) =>
    request<{ data: StorageObject }>('/storage/upload', {
      method: 'POST',
      body: JSON.stringify(data)
    }),

  verifyStorageFile: (id: string) =>
    request<{ verifiedCid: string; replicasVerified: number }>(`/storage/${id}/verify`, {
      method: 'POST'
    }),

  deleteStorageFile: (id: string) =>
    request<{ message: string }>(`/storage/${id}`, { method: 'DELETE' }),

  getDomains: () =>
    request<{
      summary: {
        totalDomains: number;
        active: number;
        expiringSoon: number;
        web3Domains: number;
      };
      data: DomainRecord[];
    }>('/domains'),

  createDomain: (data: { name: string; type?: string }) =>
    request<{ data: DomainRecord }>('/domains', {
      method: 'POST',
      body: JSON.stringify(data)
    }),

  addDnsRecord: (domainId: string, record: { type: string; name: string; value: string; ttl?: number }) =>
    request<{ data: any }>(`/domains/${domainId}/records`, {
      method: 'POST',
      body: JSON.stringify(record)
    }),

  deleteDomain: (id: string) =>
    request<{ message: string }>(`/domains/${id}`, { method: 'DELETE' }),

  getSecurity: () =>
    request<{
      summary: {
        sslCertificatesCount: number;
        secureDomainsCount: number;
        securityScore: number;
        threatsBlockedCount: number;
      };
      evaluation: SecurityEvaluation;
      certificates: SSLCertificate[];
    }>('/security'),

  issueCertificate: (data: { domain: string; issuer?: string }) =>
    request<{ data: SSLCertificate }>('/certificates/issue', {
      method: 'POST',
      body: JSON.stringify(data)
    }),

  renewCertificate: (id: string) =>
    request<{ data: SSLCertificate }>(`/certificates/${id}/renew`, {
      method: 'POST'
    }),

  getAnalytics: (range: string = '30d') =>
    request<any>(`/analytics?range=${range}`),

  getBilling: () => request<any>('/billing'),

  getTeam: () =>
    request<{
      summary: {
        totalMembers: number;
        activeMembers: number;
        teamsCount: number;
        pendingInvitesCount: number;
      };
      members: TeamMember[];
      permissions: TeamPermissionMatrix;
      inviteLink: string;
    }>('/team'),

  inviteTeamMember: (data: { email: string; role: string; team?: string }) =>
    request<{ data: TeamMember; inviteLink: string }>('/team/invite', {
      method: 'POST',
      body: JSON.stringify(data)
    }),

  updateTeamPermissions: (matrix: Partial<TeamPermissionMatrix>) =>
    request<{ data: TeamPermissionMatrix }>('/team/permissions', {
      method: 'PATCH',
      body: JSON.stringify(matrix)
    }),

  getSettings: () => request<{ data: any }>('/settings'),
  saveSettings: (settings: any) =>
    request<{ data: any }>('/settings', {
      method: 'POST',
      body: JSON.stringify(settings)
    }),
  createApiKey: (name: string) =>
    request<{ data: any; secret: string; warning: string }>('/settings/api-keys', {
      method: 'POST',
      body: JSON.stringify({ name })
    }),

  getEvidenceLedger: () =>
    request<{ ledgerAuthority: string; records: EvidenceRecord[] }>('/evidence'),

  queryCopilot: (query: string, usePlatformContext: boolean = true) =>
    request<{ data: CopilotMessage }>('/copilot/query', {
      method: 'POST',
      body: JSON.stringify({ query, usePlatformContext })
    }),

  approveCopilotAction: (actionId: string) =>
    request<{ status: string; message: string }>(`/copilot/actions/${actionId}/approve`, {
      method: 'POST'
    }),

  dismissCopilotAction: (actionId: string) =>
    request<{ status: string; message: string }>(`/copilot/actions/${actionId}/dismiss`, {
      method: 'POST'
    })
};
