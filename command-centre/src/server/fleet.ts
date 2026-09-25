/**
 * Fleet store: the ownership-aware view of infrastructure used by the
 * Nodes & Compute, Storage and Deploy command surfaces.
 *
 * Every summary number the UI shows (counts, totals, region roll-ups,
 * ownership mix, contribution totals) is DERIVED here from the underlying
 * node / storage-node / deployment lists — nothing on those pages is a
 * hardcoded headline figure. In demo mode the lists themselves are seed data.
 */
import {
  FleetNode,
  StorageNode,
  FleetDeployment,
  RegionRollup,
  WorldRegion,
  ExternalNetwork,
  DeployTemplate,
  FleetOverview,
  StorageFleetOverview,
  DeployOverview,
  Ownership,
  BenefitRing,
  ContributionMetric
} from '../types/platform';

const REGION_ANCHORS: Record<WorldRegion, [number, number]> = {
  'North America': [39.5, -98.35],
  'South America': [-14.2, -51.9],
  Europe: [50.1, 9.6],
  Africa: [1.6, 20.9],
  Asia: [22.8, 90.4],
  Oceania: [-25.3, 133.8]
};

const REGION_ORDER: WorldRegion[] = ['North America', 'Europe', 'Asia', 'South America', 'Africa', 'Oceania'];

type Seed = [name: string, region: WorldRegion, country: string, cc: string, lat: number, lng: number];

/** Community / DePIN peers visible to this operator. Only placement + status matter for these. */
const COMMUNITY_SEEDS: Seed[] = [
  ['cm-nyc-01', 'North America', 'United States', 'US', 40.7, -74.0],
  ['cm-yyz-02', 'North America', 'Canada', 'CA', 43.6, -79.4],
  ['cm-sfo-03', 'North America', 'United States', 'US', 37.7, -122.4],
  ['cm-ams-04', 'Europe', 'Netherlands', 'NL', 52.4, 4.9],
  ['cm-par-05', 'Europe', 'France', 'FR', 48.9, 2.35],
  ['cm-mad-06', 'Europe', 'Spain', 'ES', 40.4, -3.7],
  ['cm-waw-07', 'Europe', 'Poland', 'PL', 52.2, 21.0],
  ['cm-hel-08', 'Europe', 'Finland', 'FI', 60.2, 24.9],
  ['cm-sgp-09', 'Asia', 'Singapore', 'SG', 1.35, 103.8],
  ['cm-tyo-10', 'Asia', 'Japan', 'JP', 35.7, 139.7],
  ['cm-gru-11', 'South America', 'Brazil', 'BR', -23.5, -46.6],
  ['cm-nbo-12', 'Africa', 'Kenya', 'KE', -1.3, 36.8]
];

const DEPIN_SEEDS: Seed[] = [
  ['dp-akash-fra', 'Europe', 'Germany', 'DE', 50.1, 8.7],
  ['dp-flux-lon', 'Europe', 'United Kingdom', 'GB', 51.5, -0.1],
  ['dp-akash-bom', 'Asia', 'India', 'IN', 19.1, 72.9],
  ['dp-flux-hkg', 'Asia', 'Hong Kong', 'HK', 22.3, 114.2],
  ['dp-akash-icn', 'Asia', 'South Korea', 'KR', 37.5, 127.0],
  ['dp-flux-del', 'Asia', 'India', 'IN', 28.6, 77.2],
  ['dp-akash-jnb', 'Africa', 'South Africa', 'ZA', -26.2, 28.0],
  ['dp-flux-los', 'Africa', 'Nigeria', 'NG', 6.5, 3.4]
];

function peerNode(seed: Seed, ownership: Ownership, i: number): FleetNode {
  const [name, region, country, countryCode, lat, lng] = seed;
  return {
    id: `${ownership}-${i}`,
    name,
    shortId: `${ownership === 'community' ? 'cm' : 'dp'}_${String(i + 1).padStart(2, '0')}...${(0x3a1f + i * 977).toString(16).slice(-4)}`,
    ownership,
    status: 'Online',
    roles: ownership === 'community' ? ['Community'] : ['DePIN'],
    cpuAllocated: 0,
    cpuTotal: 0,
    memoryAllocatedGb: 0,
    memoryTotalGb: 0,
    storageAllocatedGb: 0,
    storageTotalGb: 0,
    gpuCount: 0,
    gpuAllocated: 0,
    country,
    countryCode,
    region,
    coordinates: [lat, lng],
    uptime: '—',
    workloadsOwner: 0,
    workloadsCommunity: 0,
    contribution: { cpuHours: 0, gpuHours: 0, bandwidthGb: 0, storageTbHours: 0 }
  };
}

export class FleetStore {
  ownedNodes: FleetNode[] = [
    {
      id: 'node_01',
      name: 'homelab-kochi',
      shortId: 'node_01...a3f2',
      ownership: 'owned',
      status: 'Online',
      roles: ['Private Host', 'Edge'],
      cpuAllocated: 4,
      cpuTotal: 12,
      memoryAllocatedGb: 8,
      memoryTotalGb: 32,
      storageAllocatedGb: 428,
      storageTotalGb: 1000,
      gpuCount: 0,
      gpuAllocated: 0,
      country: 'India',
      countryCode: 'IN',
      region: 'Asia',
      coordinates: [9.93, 76.27],
      uptime: '12d 4h',
      workloadsOwner: 3,
      workloadsCommunity: 1,
      contribution: { cpuHours: 1240, gpuHours: 0, bandwidthGb: 320, storageTbHours: 0.9 }
    },
    {
      id: 'node_02',
      name: 'gpu-workstation',
      shortId: 'node_02...f9b1',
      ownership: 'owned',
      status: 'Online',
      roles: ['Compute', 'GPU'],
      cpuAllocated: 8,
      cpuTotal: 16,
      memoryAllocatedGb: 32,
      memoryTotalGb: 64,
      storageAllocatedGb: 120,
      storageTotalGb: 1000,
      gpuModel: 'RTX 4090',
      gpuCount: 1,
      gpuAllocated: 1,
      country: 'India',
      countryCode: 'IN',
      region: 'Asia',
      coordinates: [12.97, 77.59],
      uptime: '7d 2h',
      workloadsOwner: 1,
      workloadsCommunity: 1,
      contribution: { cpuHours: 1020, gpuHours: 412, bandwidthGb: 210, storageTbHours: 0.3 }
    },
    {
      id: 'node_03',
      name: 'vps-eu',
      shortId: 'node_03...7c9d',
      ownership: 'owned',
      status: 'Online',
      roles: ['Edge', 'Storage'],
      cpuAllocated: 2,
      cpuTotal: 4,
      memoryAllocatedGb: 4,
      memoryTotalGb: 8,
      storageAllocatedGb: 120,
      storageTotalGb: 250,
      gpuCount: 0,
      gpuAllocated: 0,
      country: 'Germany',
      countryCode: 'DE',
      region: 'Europe',
      coordinates: [50.11, 8.68],
      uptime: '18d 1h',
      workloadsOwner: 2,
      workloadsCommunity: 1,
      contribution: { cpuHours: 980, gpuHours: 0, bandwidthGb: 420, storageTbHours: 0.6 }
    },
    {
      id: 'node_04',
      name: 'raspberry-pi',
      shortId: 'node_04...81de',
      ownership: 'owned',
      status: 'Offline',
      roles: ['Storage (Idle)'],
      cpuAllocated: 0,
      cpuTotal: 4,
      memoryAllocatedGb: 0.5,
      memoryTotalGb: 4,
      storageAllocatedGb: 64,
      storageTotalGb: 128,
      gpuCount: 0,
      gpuAllocated: 0,
      country: 'India',
      countryCode: 'IN',
      region: 'Asia',
      coordinates: [10.0, 76.3],
      uptime: '2d ago',
      workloadsOwner: 0,
      workloadsCommunity: 0,
      contribution: { cpuHours: 0, gpuHours: 0, bandwidthGb: 0, storageTbHours: 0 }
    }
  ];

  peerNodes: FleetNode[] = [
    ...COMMUNITY_SEEDS.map((s, i) => peerNode(s, 'community', i)),
    ...DEPIN_SEEDS.map((s, i) => peerNode(s, 'depin', i))
  ];

  storageNodes: StorageNode[] = [
    { id: 'st_01', name: 'homelab-kochi', shortId: 'node_01...a3f2', ownership: 'owned', type: 'Local Disk', filesystem: 'EXT4', status: 'Online', capacityTb: 8, usedTb: 3.2, volumes: 3, replicas: 3, integrity: 'Healthy', bandwidthServed: '320 GB', policy: 'Private + Share', region: 'Asia', coordinates: [9.93, 76.27] },
    { id: 'st_02', name: 'nas-truenas', shortId: 'node_02...f7e1', ownership: 'owned', type: 'NAS', filesystem: 'ZFS', status: 'Online', capacityTb: 16, usedTb: 6.1, volumes: 3, replicas: 2, integrity: 'Healthy', bandwidthServed: '1.2 TB', policy: 'Private', region: 'Asia', coordinates: [10.52, 76.21] },
    { id: 'st_03', name: 'vps-eu', shortId: 'node_03...9c4d', ownership: 'owned', type: 'Block Storage', filesystem: 'XFS', status: 'Online', capacityTb: 12, usedTb: 4.8, volumes: 2, replicas: 1, integrity: 'Healthy', bandwidthServed: '980 GB', policy: 'Community', region: 'Europe', coordinates: [50.11, 8.68] },
    { id: 'st_04', name: 'rp4-edge', shortId: 'node_04...8d1e', ownership: 'owned', type: 'SD Card', filesystem: 'EXT4', status: 'Offline', capacityTb: 1, usedTb: 0.62, volumes: 1, replicas: 0, integrity: 'Checking', bandwidthServed: '—', policy: 'Private', region: 'Asia', coordinates: [10.0, 76.3] },
    { id: 'st_05', name: 'crust-node', shortId: 'node_05...2a9b', ownership: 'depin', type: 'DePIN (Crust)', filesystem: 'Crust', status: 'Online', capacityTb: 20, usedTb: 9.0, volumes: 4, replicas: 5, integrity: 'Healthy', bandwidthServed: '2.1 TB', policy: 'DePIN (Crust)', region: 'Europe', coordinates: [52.52, 13.4] },
    { id: 'st_06', name: 'cm-ams-vault', shortId: 'cm_04...b71c', ownership: 'community', type: 'Community Vault', filesystem: 'ZFS', status: 'Online', capacityTb: 4, usedTb: 1.8, volumes: 2, replicas: 3, integrity: 'Healthy', bandwidthServed: '640 GB', policy: 'Community', region: 'Europe', coordinates: [52.37, 4.9] },
    { id: 'st_07', name: 'cm-nyc-vault', shortId: 'cm_01...4e2a', ownership: 'community', type: 'Community Vault', filesystem: 'EXT4', status: 'Online', capacityTb: 3.4, usedTb: 1.2, volumes: 2, replicas: 2, integrity: 'Healthy', bandwidthServed: '410 GB', policy: 'Community', region: 'North America', coordinates: [40.71, -74.0] },
    { id: 'st_08', name: 'storj-gru', shortId: 'dp_03...c55f', ownership: 'depin', type: 'DePIN (Storj)', filesystem: 'Storj', status: 'Online', capacityTb: 2, usedTb: 1.0, volumes: 1, replicas: 3, integrity: 'Healthy', bandwidthServed: '220 GB', policy: 'DePIN (Storj)', region: 'South America', coordinates: [-23.55, -46.63] },
    { id: 'st_09', name: 'sia-host-jnb', shortId: 'dp_07...9e03', ownership: 'depin', type: 'DePIN (Sia)', filesystem: 'Sia', status: 'Offline', capacityTb: 2, usedTb: 0.38, volumes: 1, replicas: 1, integrity: 'Checking', bandwidthServed: '—', policy: 'DePIN (Sia)', region: 'Africa', coordinates: [-26.2, 28.05] }
  ];

  deployments: FleetDeployment[] = [
    { id: 'dep_01', name: 'agentswarm-web', shortId: 'web_01...3f2a', status: 'Running', source: 'GitHub', sourceRef: 'main', environment: 'Production', replicasObserved: 3, replicasDesired: 3, placement: ['owned', 'community', 'edge'], endpoint: 'https://agentswarm.d.host', region: 'Asia', lastDeployed: '2h ago', commit: '7c8a91f' },
    { id: 'dep_02', name: 'api-service', shortId: 'api_02...9c1d', status: 'Deploying', source: 'GitLab', sourceRef: 'develop', environment: 'Preview', replicasObserved: 2, replicasDesired: 3, placement: ['owned', 'community'], endpoint: 'https://api-pr-42.d.host', region: 'Europe', lastDeployed: '12m ago', commit: '4d2c6e3' },
    { id: 'dep_03', name: 'ewastekochi', shortId: 'web_03...7e4b', status: 'Running', source: 'Docker', sourceRef: 'v1.2.0', environment: 'Production', replicasObserved: 2, replicasDesired: 2, placement: ['owned', 'community', 'edge'], endpoint: 'https://ewastekochi.in', region: 'Asia', lastDeployed: '1d ago', commit: 'a9f3d21' },
    { id: 'dep_04', name: 'rag-copilot', shortId: 'svc_04...6b8e', status: 'Degraded', source: 'GitHub', sourceRef: 'main', environment: 'Production', replicasObserved: 1, replicasDesired: 2, placement: ['edge', 'community', 'depin'], endpoint: 'https://rag.d.host', region: 'Europe', lastDeployed: '3h ago', commit: 'c2e9f44' },
    { id: 'dep_05', name: 'codingagent-web', shortId: 'web_05...1a7c', status: 'Running', source: 'GitHub', sourceRef: 'main', environment: 'Production', replicasObserved: 3, replicasDesired: 3, placement: ['owned', 'edge'], endpoint: 'https://codingagent.in', region: 'North America', lastDeployed: '3d ago', commit: 'e41b09a' },
    { id: 'dep_06', name: 'bestaiagent-web', shortId: 'web_06...8f30', status: 'Running', source: 'GitHub', sourceRef: 'main', environment: 'Production', replicasObserved: 2, replicasDesired: 2, placement: ['owned', 'community'], endpoint: 'https://bestaiagent.in', region: 'Europe', lastDeployed: '5d ago', commit: '19cd7e2' },
    { id: 'dep_07', name: 'dh-docs', shortId: 'web_07...c0d4', status: 'Running', source: 'Template', sourceRef: 'astro', environment: 'Production', replicasObserved: 3, replicasDesired: 3, placement: ['owned', 'edge'], endpoint: 'https://docs.decentralized.host', region: 'North America', lastDeployed: '1w ago', commit: '5b7a0c1' },
    { id: 'dep_08', name: 'llm-inference', shortId: 'gpu_08...2e9a', status: 'Running', source: 'Docker', sourceRef: 'vllm:0.6', environment: 'Preview', replicasObserved: 2, replicasDesired: 2, placement: ['owned', 'depin'], endpoint: 'https://infer-pr-7.d.host', region: 'Asia', lastDeployed: '6h ago', commit: 'f0a2b8d' },
    { id: 'dep_09', name: 'status-page', shortId: 'web_09...5d11', status: 'Running', source: 'Upload', sourceRef: 'status.zip', environment: 'Preview', replicasObserved: 3, replicasDesired: 3, placement: ['owned', 'community'], endpoint: 'https://status-pr-3.d.host', region: 'Europe', lastDeployed: '2d ago', commit: '9e3c7b2' },
    { id: 'dep_10', name: 'wp-blog', shortId: 'web_10...a4f7', status: 'Running', source: 'Template', sourceRef: 'wordpress', environment: 'Preview', replicasObserved: 2, replicasDesired: 2, placement: ['community', 'owned'], endpoint: 'https://blog-pr-1.d.host', region: 'South America', lastDeployed: '4d ago', commit: '3a8d5e0' },
    { id: 'dep_11', name: 'edge-cache-af', shortId: 'edg_11...7b2c', status: 'Running', source: 'GitHub', sourceRef: 'main', environment: 'Production', replicasObserved: 2, replicasDesired: 2, placement: ['community', 'depin'], endpoint: 'https://af.edge.d.host', region: 'Africa', lastDeployed: '6d ago', commit: 'b6e2f19' },
    { id: 'dep_12', name: 'legacy-landing', shortId: 'web_12...0e6f', status: 'Stopped', source: 'GitHub', sourceRef: 'legacy', environment: 'Production', replicasObserved: 0, replicasDesired: 0, placement: ['owned'], endpoint: 'https://old.d.host', region: 'Europe', lastDeployed: '3w ago', commit: '0d4e8a7' }
  ];

  computeNetworks: ExternalNetwork[] = [
    { id: 'dh', name: 'Decentralized.Host', description: 'Your private + community network', kind: 'compute', status: 'Active', accent: '#248BFF', glyph: 'cube' },
    { id: 'akash', name: 'Akash Network', description: 'Decentralized cloud compute', kind: 'compute', status: 'Ready to Deploy', accent: '#EF4444', glyph: 'akash' },
    { id: 'golem', name: 'Golem Network', description: 'Distributed compute (CPU/GPU)', kind: 'compute', status: 'Ready to Install', accent: '#5965FF', glyph: 'golem' },
    { id: 'flux', name: 'Flux Network', description: 'Decentralized cloud infrastructure', kind: 'compute', status: 'Active', accent: '#38BDF8', glyph: 'flux' },
    { id: 'cloud', name: 'External Cloud', description: 'Connect AWS, GCP, Azure, etc.', kind: 'compute', status: 'Not Configured', accent: '#94A3B8', glyph: 'cloud' }
  ];

  storageNetworks: ExternalNetwork[] = [
    { id: 'filecoin', name: 'Filecoin Network', description: 'Decentralized storage network', kind: 'storage', status: 'Ready to Install', accent: '#248BFF', glyph: 'filecoin' },
    { id: 'arweave', name: 'Arweave Network', description: 'Permanent data storage', kind: 'storage', status: 'Not Installed', accent: '#E2E8F0', glyph: 'arweave' },
    { id: 'storj', name: 'Storj Network', description: 'Distributed cloud storage', kind: 'storage', status: 'Active', accent: '#2563EB', glyph: 'storj' },
    { id: 'sia', name: 'Sia Network', description: 'Decentralized storage platform', kind: 'storage', status: 'Eligibility Check', accent: '#10D981', glyph: 'sia' },
    { id: 'ipfs', name: 'IPFS (Pinning Service)', description: 'Distributed content addressing', kind: 'storage', status: 'Active', accent: '#20DDF7', glyph: 'ipfs' },
    { id: 'crust', name: 'Crust Network', description: 'Web3 storage infrastructure', kind: 'storage', status: 'Active', accent: '#F59E0B', glyph: 'crust' }
  ];

  templates: DeployTemplate[] = [
    { id: 'nextjs', name: 'Next.js', description: 'Full-stack web application', glyph: 'N', accent: '#F8FAFC' },
    { id: 'astro', name: 'Astro', description: 'Static site with edge support', glyph: 'A', accent: '#A855F7' },
    { id: 'fastapi', name: 'FastAPI', description: 'Python API service', glyph: 'bolt', accent: '#10D981' },
    { id: 'postgres', name: 'PostgreSQL', description: 'Database with persistent storage', glyph: 'db', accent: '#38BDF8' },
    { id: 'wordpress', name: 'WordPress', description: 'Blog or CMS', glyph: 'W', accent: '#E2E8F0' },
    { id: 'inference', name: 'AI Inference', description: 'GPU workload template', glyph: 'cube', accent: '#A855F7' }
  ];

  /** Trend deltas are period-over-period comparisons the telemetry pipeline would supply. */
  trends = {
    cpuHours: 18, gpuHours: 26, storageTbHours: 12, bandwidth: 32,
    storageContributed: 28, bandwidthServed: 18, storageWorkloads: 40,
    requests: 18, deployBandwidth: 32
  };

  allNodes(): FleetNode[] {
    return [...this.ownedNodes, ...this.peerNodes];
  }

  regionRollups(): RegionRollup[] {
    const rollups = new Map<WorldRegion, RegionRollup>();
    for (const r of REGION_ORDER) {
      rollups.set(r, { region: r, coordinates: REGION_ANCHORS[r], owned: 0, community: 0, depin: 0, offline: 0, storageTb: 0, storageNodes: 0, deployments: 0 });
    }
    for (const n of this.allNodes()) {
      const r = rollups.get(n.region)!;
      if (n.status === 'Offline') r.offline++;
      if (n.ownership === 'owned') r.owned++;
      else if (n.ownership === 'community') r.community++;
      else if (n.ownership === 'depin') r.depin++;
    }
    for (const s of this.storageNodes) {
      const r = rollups.get(s.region)!;
      r.storageNodes++;
      r.storageTb = Math.round((r.storageTb + s.capacityTb) * 10) / 10;
    }
    for (const d of this.deployments) {
      if (d.status !== 'Stopped') rollups.get(d.region)!.deployments++;
    }
    return REGION_ORDER.map((r) => rollups.get(r)!);
  }

  fleetOverview(): FleetOverview {
    const owned = this.ownedNodes;
    const sum = (f: (n: FleetNode) => number) => owned.reduce((a, n) => a + f(n), 0);
    const cpuHours = sum((n) => n.contribution.cpuHours);
    const gpuHours = sum((n) => n.contribution.gpuHours);
    const bandwidthGb = sum((n) => n.contribution.bandwidthGb);
    const storageTbHours = sum((n) => n.contribution.storageTbHours);

    const contribution: ContributionMetric[] = [
      { id: 'cpu', label: 'CPU Contribution', value: `${cpuHours.toLocaleString()} CPU-hours`, deltaPercent: this.trends.cpuHours, tone: 'emerald' },
      { id: 'gpu', label: 'GPU Contribution', value: `${gpuHours.toLocaleString()} GPU-hours`, deltaPercent: this.trends.gpuHours, tone: 'violet' },
      { id: 'storage', label: 'Storage Contribution', value: `${storageTbHours.toFixed(1)} TB-hours`, deltaPercent: this.trends.storageTbHours, tone: 'blue' },
      { id: 'bandwidth', label: 'Bandwidth Contribution', value: `${bandwidthGb.toLocaleString()} GB`, deltaPercent: this.trends.bandwidth, tone: 'amber' }
    ];

    // Benefits are derived from current placement of owner workloads + storage.
    const ownerWl = sum((n) => n.workloadsOwner);
    const allWl = sum((n) => n.workloadsOwner + n.workloadsCommunity);
    const ownedStorage = this.storageNodes.filter((s) => s.ownership === 'owned');
    const privateShare = ownedStorage.filter((s) => s.policy.startsWith('Private')).length / Math.max(ownedStorage.length, 1);

    const benefits: BenefitRing[] = [
      { id: 'onHw', label: 'Workloads on Your Hardware', value: pct(ownerWl, allWl), unit: '%', tone: 'cyan' },
      { id: 'extDep', label: 'External Dependencies', value: pct(this.deployments.filter((d) => d.placement.includes('depin')).length, this.deployments.length), unit: '%', tone: 'violet' },
      { id: 'private', label: 'Private Storage', value: Math.round(privateShare * 100), unit: '%', tone: 'emerald' },
      { id: 'indep', label: 'Independent Nodes', value: owned.length, unit: '', tone: 'blue' }
    ];

    return {
      mode: 'demo',
      observedAt: new Date().toISOString(),
      nodes: this.allNodes(),
      regions: this.regionRollups(),
      contribution,
      networks: this.computeNetworks.map((n) =>
        n.id === 'dh' ? { ...n, nodes: owned.filter((o) => o.status !== 'Offline').length + 1 } : n
      ),
      benefits
    };
  }

  storageOverview(): StorageFleetOverview {
    const nodes = this.storageNodes;
    const online = nodes.filter((n) => n.status === 'Online');
    const used = nodes.reduce((a, n) => a + n.usedTb, 0);
    const replicated = online.filter((n) => n.replicas > 1).reduce((a, n) => a + n.usedTb, 0);
    const owned = nodes.filter((n) => n.ownership === 'owned');
    const ownedUsed = owned.reduce((a, n) => a + n.usedTb, 0);

    const integrityChecks = 12942;
    const benefits: BenefitRing[] = [
      { id: 'onHw', label: 'Data on Your Hardware', value: pct(ownedUsed, used), unit: '%', tone: 'cyan' },
      { id: 'extDep', label: 'External Dependency', value: 100 - pct(ownedUsed, used), unit: '%', tone: 'violet' },
      { id: 'protected', label: 'Replicated & Protected', value: pct(replicated, used), unit: '%', tone: 'emerald' },
      { id: 'indep', label: 'Independent Storage Nodes', value: owned.length + nodes.filter((n) => n.ownership === 'community').length, unit: '', tone: 'blue' }
    ];

    return {
      mode: 'demo',
      observedAt: new Date().toISOString(),
      storageNodes: nodes,
      regions: this.regionRollups(),
      contribution: [
        { id: 'contrib', label: 'Storage Contributed', value: '12.6 TB-hours', deltaPercent: this.trends.storageContributed, tone: 'emerald' },
        { id: 'bw', label: 'Bandwidth Served', value: '4.8 TB', deltaPercent: this.trends.bandwidthServed, tone: 'violet' },
        { id: 'wl', label: 'Active Storage Workloads', value: String(online.filter((n) => n.policy !== 'Private').length + 1), deltaPercent: this.trends.storageWorkloads, tone: 'blue' }
      ],
      networks: this.storageNetworks,
      benefits,
      replicatedTb: round1(replicated),
      protectedPercent: pct(replicated, used),
      integrityChecks,
      integrityPassedPercent: nodes.some((n) => n.integrity === 'Degraded') ? 99 : 100
    };
  }

  deployOverview(): DeployOverview {
    const weights: Record<Ownership, number> = { owned: 0, community: 0, depin: 0, edge: 0, cloud: 0 };
    for (const d of this.deployments) {
      if (!d.replicasObserved) continue;
      // Primary placement carries half the replicas; the rest are spread over secondary targets.
      const [primary, ...rest] = d.placement;
      weights[primary] += d.replicasObserved * (rest.length ? 0.5 : 1);
      for (const o of rest) weights[o] += (d.replicasObserved * 0.5) / rest.length;
    }
    // Edge nodes in this fleet are operator-owned PoPs, so they count as "your hardware".
    weights.owned += weights.edge;
    weights.edge = 0;
    const total = Object.values(weights).reduce((a, b) => a + b, 0) || 1;
    const mix = (['owned', 'community', 'depin', 'cloud'] as Ownership[]).map((o) => ({ ownership: o, percent: Math.round((weights[o] / total) * 100) }));
    // Keep the donut summing to 100 after rounding.
    mix[0].percent += 100 - mix.reduce((a, m) => a + m.percent, 0);

    return {
      mode: 'demo',
      observedAt: new Date().toISOString(),
      deployments: this.deployments,
      regions: this.regionRollups(),
      templates: this.templates,
      networks: this.computeNetworks.map((n) => (n.id === 'dh' ? { ...n, nodes: this.ownedNodes.length + 4 } : n)),
      ownershipMix: mix,
      successRatePercent: 98.4,
      avgDeploySeconds: 134,
      deploysThisWeek: 3,
      traffic24h: { requests: 1_200_000, requestsDeltaPercent: this.trends.requests, bandwidthGb: 264, bandwidthDeltaPercent: this.trends.deployBandwidth }
    };
  }
}

function pct(part: number, whole: number): number {
  return whole > 0 ? Math.round((part / whole) * 100) : 0;
}

function round1(n: number): number {
  return Math.round(n * 10) / 10;
}

export const fleetStore = new FleetStore();
