export type OperationalState = 'LIVE' | 'DERIVED' | 'CONFIGURED' | 'UNAVAILABLE' | 'PLANNED' | 'SIMULATED' | 'UNKNOWN';
export type Freshness = 'FRESH' | 'STALE' | 'EXPIRED' | 'UNREACHABLE';

export interface TruthEnvelope<T> {
  value: T | null;
  source: string;
  observedAt: string | null;
  freshness: Freshness;
  state: OperationalState;
}

export type TrustClass = 'OWNER' | 'ORGANIZATION' | 'COMMUNITY' | 'MARKETPLACE' | 'EXTERNAL_DEPIN';
export type IsolationLevel = 'rootless' | 'strict_seccomp' | 'gVisor' | 'Firecracker' | 'UNAVAILABLE';

export interface NodeResourceAllocation {
  total: number;
  ownerReserve: number;
  marketplaceReserved: number;
  allocated: number;
  available: number;
}

export interface NodeItem {
  id: string;
  name: string;
  shortId: string;
  status: 'Online' | 'Degraded' | 'Offline' | 'Unknown';
  cordoned: boolean;
  drained: boolean;
  roles: string[];
  cpu: NodeResourceAllocation;
  memoryGb: NodeResourceAllocation;
  storageGb: NodeResourceAllocation;
  gpu?: {
    count: number;
    model: string;
    vramGb: number;
    allocated: number;
    available: number;
  };
  location: {
    country: string;
    region: string;
    code: string;
    flag: string;
    lat: number;
    lng: number;
  };
  uptime: string;
  workloads: {
    total: number;
    owner: number;
    community: number;
  };
  contribution: {
    cpuh: number;
    gpuh: number;
    storageTbh: number;
    bandwidthGb: number;
  };
  trustClass: TrustClass;
  isolation: IsolationLevel;
  agentVersion: string;
  os: string;
  arch: string;
  kernel: string;
  ip: string;
  lastHeartbeat: string;
  isSelfHosted: boolean;
  isContributing: boolean;
  isDePIN: boolean;
}

export interface StorageNodeItem {
  id: string;
  name: string;
  shortId: string;
  type: string;
  status: 'Online' | 'Offline' | 'Degraded';
  capacityTb: number;
  usedTb: number;
  availableTb: number;
  replicas: number;
  integrity: 'Healthy' | 'Checking...' | 'Degraded';
  bandwidthServed: string;
  activeWorkloads: number;
  integrityChecksPassed: number;
  policy: 'Private' | 'Private + Share' | 'Community' | 'DePIN (Crust)';
  location: string;
  flag: string;
  merkleRoot: string;
  mountPoint: string;
  isSelfHosted: boolean;
  isCommunity: boolean;
  isDePIN: boolean;
}

export interface DeploymentItem {
  id: string;
  name: string;
  shortId: string;
  status: 'Running' | 'Deploying' | 'Degraded' | 'Stopped';
  source: {
    type: 'GitHub' | 'GitLab' | 'Docker' | 'Upload';
    repo?: string;
    branch?: string;
    commitSha: string;
    image?: string;
  };
  environment: 'Production' | 'Preview';
  replicas: {
    desired: number;
    observed: number;
    healthy: number;
  };
  placement: ('self' | 'community' | 'depin' | 'edge')[];
  domain: string;
  traffic30d: string;
  lastDeploy: string;
  failureDomains: string[];
  trustClass: 'PRIVATE' | 'RESTRICTED' | 'UNTRUSTED';
  sbom: {
    verified: boolean;
    digest: string;
    packagesCount: number;
  };
  secretsBound: string[];
  ports: number[];
  yamlConfig: string;
}

export interface ActivityItem {
  id: string;
  title: string;
  target: string;
  status: 'Success' | 'Online' | 'Pending' | 'Warning' | 'Error';
  time: string;
  type: 'deploy' | 'node' | 'ssl' | 'domain' | 'team' | 'security' | 'lease';
  details?: string;
}

export interface DePINOffer {
  id: string;
  nodeId: string;
  nodeName: string;
  provider: string;
  cpuOffered: number;
  ramOfferedGb: number;
  gpuOffered?: string;
  storageOfferedGb: number;
  priceHourly: number;
  currency: string;
  schedule: 'ALWAYS' | 'IDLE_ONLY' | 'SCHEDULED';
  terms: string;
  reputationScore: number;
  status: 'Active' | 'Paused' | 'Leased';
}

export interface DePINLease {
  id: string;
  provider: string;
  consumer: string;
  nodeName: string;
  workloadName: string;
  status: 'PROPOSED' | 'BID' | 'ACCEPTED' | 'ACTIVE' | 'EXPIRING' | 'COMPLETED' | 'TERMINATED' | 'DISPUTED';
  cpuAllocated: number;
  ramAllocatedGb: number;
  pricePerHour: number;
  durationHours: number;
  startedAt: string;
  meteredCpuSec: number;
  meteredRamByteSec: string;
  evidenceRoot: string;
  settlementMethod: 'FREE' | 'ORG_INTERNAL' | 'CREDITS' | 'FIAT' | 'NETWORK_TOKEN';
}

export interface CopilotMessage {
  id: string;
  sender: 'user' | 'copilot';
  text: string;
  timestamp: string;
  mode: 'ASK' | 'DIAGNOSE' | 'PLAN' | 'ACT';
  model?: string;
  thinking?: boolean;
  actionProposal?: {
    type: string;
    title: string;
    details: string;
    executed?: boolean;
  };
}
