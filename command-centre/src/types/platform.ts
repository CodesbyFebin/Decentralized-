export type CapabilityState =
  | 'LIVE'
  | 'DERIVED'
  | 'CONFIGURED'
  | 'UNAVAILABLE'
  | 'PLANNED'
  | 'SIMULATED'
  | 'UNKNOWN';

export interface DesiredState<T = any> {
  specification: T;
  configuredAt: string;
  configuredBy: string;
  version: number;
}

export interface ObservedState<T = any> {
  current: T;
  observedAt: string;
  lastHeartbeatAgeSeconds: number;
  healthy: boolean;
  divergenceDetected: boolean;
  divergenceDetails?: string;
}

export interface EvidenceRecord {
  recordId: string;
  timestamp: string;
  sha256Digest: string;
  signerFingerprint: string;
  status: 'VERIFIED' | 'SEALED' | 'FAILED' | 'PENDING';
  gatesPassed: string[];
  gatesTotal: number;
  sourceCommit?: string;
  artifactDigest?: string;
}

export interface PlatformCapabilities {
  nodes: CapabilityState;
  deployments: CapabilityState;
  storage: CapabilityState;
  domains: CapabilityState;
  acme: CapabilityState;
  waf: CapabilityState;
  ddosTelemetry: CapabilityState;
  visitorAnalytics: CapabilityState;
  infraTelemetry: CapabilityState;
  billing: CapabilityState;
  copilot: CapabilityState;
  evidenceLedger: CapabilityState;
}

export interface NodeWorkload {
  id: string;
  name: string;
  type: 'WASM Edge' | 'Docker Container' | 'AI Inference' | 'IPFS Daemon' | 'Ingress Gateway';
  cpuPercent: number;
  memoryMb: number;
  status: 'Running' | 'Degraded' | 'Stopped';
  deployedAt: string;
}

export interface NodeInfo {
  id: string;
  name: string;
  region: string;
  location: string;
  countryCode: string;
  ipAddress: string;
  provider: string;
  role: 'Validator' | 'Edge Gateway' | 'Storage Replicator' | 'GPU Compute Rig' | 'Homelab Node' | 'DePIN Worker';
  status: 'Online' | 'Degraded' | 'Offline';
  desiredState: 'Active' | 'Drained' | 'Cordoned';
  observedState: 'Active' | 'Degraded' | 'Offline' | 'Cordoned' | 'Drained';
  // CPU & Memory
  cpuPercent: number;
  cpuCores: number;
  cpuModel?: string;
  memoryPercent: number;
  memoryUsedGb: number;
  memoryTotalGb: number;
  // Storage
  diskUsedGb: number;
  diskTotalGb: number;
  bandwidthUsedTb: number;
  // GPU Acceleration (if equipped)
  gpuEquipped?: boolean;
  gpuModel?: string;
  gpuCount?: number;
  gpuPercent?: number;
  gpuMemoryUsedGb?: number;
  gpuMemoryTotalGb?: number;
  // Workload Management
  workloadCount: number;
  workloads?: NodeWorkload[];
  // Node Contribution & SLA Metrics
  uptimePercent: number;
  uptimeDays: number;
  lastHeartbeatSecondsAgo: number;
  uptimeScore?: number; // 0-100 SLA score
  proofsVerifiedCount?: number;
  rewardsEarnedCredits?: number;
  supportedDePINTags?: string[]; // e.g. ['Akash', 'Render', 'Filecoin', 'Livepeer']
  hardwareType?: 'Self-Hosted Edge' | 'Bare-Metal' | 'Cloud Edge' | 'Homelab SBC';
  architecture?: 'x86_64' | 'ARM64' | 'Apple Silicon';
  trustFingerprint: string;
  coordinates: [number, number]; // [lat, lng]
  evidence: EvidenceRecord;
}

export interface DePINIntegration {
  id: string;
  name: string;
  symbol: string;
  category: 'Compute' | 'Storage' | 'AI & Rendering' | 'Wireless & Bandwidth';
  description: string;
  status: 'Active' | 'Connected' | 'Ready';
  nodesConnected: number;
  earningsTotal: string;
  icon: string;
  protocolLink: string;
}

export interface SelfHostingBenefit {
  id: string;
  title: string;
  description: string;
  metric: string;
  tag: string;
}

export interface ComputeSummary {
  totalNodes: number;
  selfHostedNodes: number;
  datacenterNodes: number;
  totalCpuCores: number;
  usedCpuCores: number;
  cpuUtilizationPercent: number;
  totalMemoryGb: number;
  usedMemoryGb: number;
  memoryUtilizationPercent: number;
  totalStorageTb: number;
  usedStorageTb: number;
  storageUtilizationPercent: number;
  totalGpus: number;
  activeGpus: number;
  totalGpuVramGb: number;
  totalActiveWorkloads: number;
  depinNetworksConnected: number;
  totalRewardsEarnedDH: number;
  averageUptimeScore: number;
}

export interface Application {
  id: string;
  name: string;
  domain: string;
  type: 'Static Site' | 'Web App' | 'WordPress' | 'Docker App' | 'Custom';
  status: 'Online' | 'Deploying' | 'Degraded' | 'Offline';
  desiredReplicas: number;
  observedHealthyReplicas: number;
  nodesAssigned: string[];
  regionsAssigned: string[];
  version: string;
  lastDeploymentTime: string;
  visitors30d: number;
  pageViews30d: number;
  bandwidthUsedGb: number;
  avgResponseMs: number;
  envVars: { key: string; isSecret: boolean; maskedValue: string }[];
  logs: string[];
  evidence: EvidenceRecord;
}

export interface Deployment {
  id: string;
  appId: string;
  appName: string;
  domain: string;
  sourceType: 'Git' | 'Docker' | 'Upload' | 'Template';
  sourceReference: string;
  status: 'VALIDATING' | 'SCHEDULING' | 'ARTIFACT_TRANSFER' | 'RUNTIME_CREATION' | 'HEALTH_CHECK' | 'ROUTING' | 'VERIFIED' | 'FAILED' | 'CANCELLED';
  currentStage: string;
  stagesCompleted: string[];
  regions: string[];
  createdAt: string;
  completedAt?: string;
  commitHash?: string;
  artifactDigest?: string;
  logs: { timestamp: string; stage: string; message: string; severity: 'info' | 'warn' | 'error' | 'success' }[];
  evidence?: EvidenceRecord;
}

export interface StorageObject {
  id: string;
  name: string;
  bucket: string;
  type: 'Folder' | 'ZIP' | 'SQL' | 'PDF' | 'Image' | 'Binary';
  sizeBytes: number;
  sizeFormatted: string;
  replicasTarget: number;
  replicasObserved: number;
  nodesPlacements: string[];
  locationSummary: string;
  modified: string;
  sha256Cid: string;
  verificationStatus: 'VERIFIED' | 'UNVERIFIED' | 'MISMATCH';
}

export interface StorageBucket {
  id: string;
  name: string;
  totalSizeFormatted: string;
  fileCount: number;
  regions: string[];
  replicationFactor: number;
  createdAt: string;
}

export interface DomainRecord {
  id: string;
  name: string;
  type: 'Traditional' | 'Web3 (ENS)' | 'Internal';
  status: 'Active' | 'Expiring Soon' | 'Pending DNS';
  dnsProvider: string;
  nodesWorldwide: number;
  expiryDate: string;
  daysRemaining: number | null;
  autoRenew: boolean;
  assignedAppId?: string;
  dnsRecords: {
    id: string;
    type: 'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX';
    name: string;
    value: string;
    ttl: number;
  }[];
  web3ContentHash?: string;
}

export interface SSLCertificate {
  id: string;
  domain: string;
  type: 'DV' | 'OV' | 'Wildcard';
  status: 'Valid' | 'Expiring Soon' | 'Expired' | 'Pending';
  issuedBy: "Let's Encrypt" | 'ZeroSSL' | 'Cloudflare';
  issuedDate: string;
  expiryDate: string;
  daysRemaining: number;
  autoRenew: boolean;
  fingerprintSha256: string;
  keyType: string;
}

export interface SecurityEvaluation {
  score: number;
  statusText: 'Excellent' | 'Good' | 'Needs Attention';
  passedChecks: { id: string; title: string; detail: string }[];
  warningChecks: { id: string; title: string; detail: string; recommendation: string }[];
  failedChecks: { id: string; title: string; detail: string; remediation: string }[];
  wafRulesActive: number;
  ddosEdgeStatus: 'Active' | 'Degraded' | 'Offline';
  threatsBlockedCount: number;
  threatTelemetryAvailable: boolean;
  securityHeadersConfigured: boolean;
}

export interface TeamMember {
  id: string;
  name: string;
  email: string;
  role: 'Owner' | 'Admin' | 'Developer' | 'Viewer';
  teams: string[];
  accessLevel: 'Full Access' | 'Deploy & Manage' | 'Read Only';
  status: 'Active' | 'Invited' | 'Suspended';
  lastActive: string;
  avatarInitials: string;
}

export interface TeamPermissionMatrix {
  deployApps: boolean;
  manageDomains: boolean;
  manageStorage: boolean;
  sslAndSecurity: boolean;
  viewAnalytics: boolean;
  drainNodes: boolean;
  manageBilling: boolean;
}

export interface CopilotCitation {
  id: string;
  source: 'documentation' | 'node_observation' | 'deployment_log' | 'evidence_record';
  title: string;
  pathOrId: string;
  snippet?: string;
}

export interface ProposedAction {
  id: string;
  title: string;
  riskLevel: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  description: string;
  impact: string;
  operationPayload: {
    actionType: string;
    targetId: string;
    parameters?: Record<string, any>;
  };
  status: 'PENDING_APPROVAL' | 'APPROVED' | 'EXECUTED' | 'DISMISSED' | 'REJECTED';
}

export interface CopilotMessage {
  id: string;
  sender: 'user' | 'assistant';
  timestamp: string;
  content: string;
  stepGuide?: {
    title: string;
    steps: { number: number; label: string; desc: string }[];
    codeSnippets?: { tabName: string; code: string }[];
    docLinks?: { title: string; path: string }[];
  };
  citations?: CopilotCitation[];
  proposedAction?: ProposedAction;
}

export interface PlatformEvent {
  id: string;
  timestamp: string;
  type: string;
  actor: string;
  resourceType: string;
  resourceId: string;
  severity: 'info' | 'success' | 'warn' | 'error';
  message: string;
}

/* ------------------------------------------------------------------ */
/* Infrastructure ownership model (Nodes & Compute / Storage / Deploy) */
/* ------------------------------------------------------------------ */

/** Who operates a piece of infrastructure. Drives colour + icon everywhere. */
export type Ownership = 'owned' | 'community' | 'depin' | 'edge' | 'cloud';

export type WorldRegion = 'North America' | 'South America' | 'Europe' | 'Africa' | 'Asia' | 'Oceania';

export interface FleetNode {
  id: string;
  name: string;
  shortId: string;
  ownership: Ownership;
  status: 'Online' | 'Degraded' | 'Offline';
  roles: string[];
  cpuAllocated: number;
  cpuTotal: number;
  memoryAllocatedGb: number;
  memoryTotalGb: number;
  storageAllocatedGb: number;
  storageTotalGb: number;
  gpuModel?: string;
  gpuCount: number;
  gpuAllocated: number;
  country: string;
  countryCode: string;
  region: WorldRegion;
  coordinates: [number, number];
  uptime: string;
  workloadsOwner: number;
  workloadsCommunity: number;
  contribution: { cpuHours: number; gpuHours: number; bandwidthGb: number; storageTbHours: number };
}

export interface RegionRollup {
  region: WorldRegion;
  coordinates: [number, number];
  owned: number;
  community: number;
  depin: number;
  offline: number;
  storageTb: number;
  storageNodes: number;
  deployments: number;
}

export interface ExternalNetwork {
  id: string;
  name: string;
  description: string;
  kind: 'compute' | 'storage';
  status: 'Active' | 'Ready to Deploy' | 'Ready to Install' | 'Not Installed' | 'Not Configured' | 'Eligibility Check';
  nodes?: number;
  accent: string;
  glyph: string;
}

export interface ContributionMetric {
  id: string;
  label: string;
  value: string;
  deltaPercent: number;
  tone: 'cyan' | 'violet' | 'blue' | 'amber' | 'emerald';
}

export interface BenefitRing {
  id: string;
  label: string;
  value: number;
  unit: '%' | '';
  tone: 'cyan' | 'violet' | 'emerald' | 'blue';
}

export interface StorageNode {
  id: string;
  name: string;
  shortId: string;
  ownership: Ownership;
  type: string;
  filesystem: string;
  status: 'Online' | 'Offline' | 'Degraded';
  capacityTb: number;
  usedTb: number;
  volumes: number;
  replicas: number;
  integrity: 'Healthy' | 'Checking' | 'Degraded';
  bandwidthServed: string;
  policy: string;
  region: WorldRegion;
  coordinates: [number, number];
}

export interface FleetDeployment {
  id: string;
  name: string;
  shortId: string;
  status: 'Running' | 'Deploying' | 'Degraded' | 'Stopped' | 'Failed';
  source: 'GitHub' | 'GitLab' | 'Docker' | 'Upload' | 'Template';
  sourceRef: string;
  environment: 'Production' | 'Preview';
  replicasObserved: number;
  replicasDesired: number;
  placement: Ownership[];
  endpoint: string;
  region: WorldRegion;
  lastDeployed: string;
  commit: string;
}

export interface DeployTemplate {
  id: string;
  name: string;
  description: string;
  glyph: string;
  accent: string;
}

export interface FleetOverview {
  mode: 'demo' | 'live';
  observedAt: string;
  nodes: FleetNode[];
  regions: RegionRollup[];
  contribution: ContributionMetric[];
  networks: ExternalNetwork[];
  benefits: BenefitRing[];
}

export interface StorageFleetOverview {
  mode: 'demo' | 'live';
  observedAt: string;
  storageNodes: StorageNode[];
  regions: RegionRollup[];
  contribution: ContributionMetric[];
  networks: ExternalNetwork[];
  benefits: BenefitRing[];
  replicatedTb: number;
  protectedPercent: number;
  integrityChecks: number;
  integrityPassedPercent: number;
}

export interface DeployOverview {
  mode: 'demo' | 'live';
  observedAt: string;
  deployments: FleetDeployment[];
  regions: RegionRollup[];
  templates: DeployTemplate[];
  networks: ExternalNetwork[];
  ownershipMix: { ownership: Ownership; percent: number }[];
  successRatePercent: number;
  avgDeploySeconds: number;
  deploysThisWeek: number;
  traffic24h: { requests: number; requestsDeltaPercent: number; bandwidthGb: number; bandwidthDeltaPercent: number };
}
