// Node types
export interface NodeMetrics {
  total: number
  used: number
  percent: number
}

export interface Node {
  id: string
  name: string
  status: 'active' | 'inactive' | 'error'
  cpu: NodeMetrics
  memory: NodeMetrics
  disk: NodeMetrics
  lastSeen: string
  nodeType: 'qemu-vm' | 'kubernetes' | 'native' | 'container'
  isolationBoundary: 'distinct' | 'same' | 'unknown'
  tags: string[]
}

// Deployment types
export type DeploymentState = 'pending' | 'admitted' | 'executing' | 'observed' | 'verified'

export interface Deployment {
  id: string
  name: string
  status: DeploymentState
  createdAt: string
  updatedAt: string
  desiredState: Record<string, unknown>
  observedState: Record<string, unknown>
  targetNodes: string[]
  progress: {
    current: number
    total: number
  }
  signedBy: string
}

// Metrics types
export interface Metric {
  timestamp: string
  nodeId: string
  metricType: 'cpu' | 'memory' | 'disk' | 'network' | 'requests'
  value: number
  unit: string
}

export interface MetricsSnapshot {
  timestamp: string
  metrics: Metric[]
}

// Dashboard types
export interface DashboardMetrics {
  totalNodes: number
  activeDeployments: number
  storageUsed: number
  successRate: number
  cpuAverage: number
  memoryAverage: number
}

export interface Activity {
  id: string
  timestamp: string
  type: 'deployment' | 'node' | 'security' | 'storage'
  action: string
  actor: string
  status: 'success' | 'failure' | 'pending'
  details: string
}

// Storage types
export interface StorageBucket {
  id: string
  name: string
  size: number
  usedSpace: number
  createdAt: string
  replication: number
}

export interface StorageMetrics {
  totalCapacity: number
  usedCapacity: number
  availableCapacity: number
  buckets: StorageBucket[]
}

// Domain types
export interface DNSRecord {
  name: string
  type: 'A' | 'AAAA' | 'CNAME' | 'MX' | 'TXT' | 'NS'
  value: string
  ttl: number
}

export interface Domain {
  id: string
  name: string
  status: 'active' | 'pending' | 'expired'
  registrar: string
  expiresAt: string
  dnsRecords: DNSRecord[]
}

// Security types
export interface Certificate {
  id: string
  commonName: string
  issuer: string
  issuedAt: string
  expiresAt: string
  fingerprint: string
  status: 'valid' | 'expired' | 'revoked'
}

export interface SecurityPolicy {
  id: string
  name: string
  description: string
  rules: PolicyRule[]
  createdAt: string
  updatedAt: string
}

export interface PolicyRule {
  id: string
  action: 'allow' | 'deny'
  resource: string
  principal: string
  effect: string
}

export interface AuditEntry {
  id: string
  timestamp: string
  actor: string
  action: string
  resource: string
  status: 'success' | 'failure'
  details: Record<string, unknown>
}

// Analytics types
export interface TrafficMetric {
  timestamp: string
  requestCount: number
  errorCount: number
  latencyP50: number
  latencyP95: number
  latencyP99: number
}

export interface ErrorAnalytic {
  errorType: string
  count: number
  percentage: number
  lastOccurred: string
}

// Team types
export interface TeamMember {
  id: string
  email: string
  name: string
  role: 'owner' | 'admin' | 'member' | 'viewer'
  joinedAt: string
  status: 'active' | 'pending'
}

// Evidence types
export interface Evidence {
  id: string
  type: 'qualification' | 'chaos' | 'deployment'
  timestamp: string
  campaignId: string
  nodeId: string
  status: 'pass' | 'fail'
  summary: string
  details: Record<string, unknown>
  signature: string
  signer: string
}

// Settings types
export interface Settings {
  siteName: string
  siteDescription: string
  siteUrl: string
  notificationsEnabled: boolean
  darkMode: boolean
  timezone: string
  language: string
}

export interface ApiKey {
  id: string
  name: string
  prefix: string
  createdAt: string
  expiresAt: string
  lastUsedAt: string
  scopes: string[]
}

export interface Webhook {
  id: string
  url: string
  events: string[]
  active: boolean
  createdAt: string
  lastTriggered: string
}

// Auth types
export interface User {
  id: string
  email: string
  name: string
  role: string
  avatar?: string
  createdAt: string
}

export interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
}
