import {
  PlatformCapabilities,
  NodeInfo,
  Application,
  Deployment,
  StorageObject,
  StorageBucket,
  DomainRecord,
  SSLCertificate,
  SecurityEvaluation,
  TeamMember,
  TeamPermissionMatrix,
  EvidenceRecord,
  PlatformEvent,
  ProposedAction,
  DePINIntegration,
  SelfHostingBenefit,
  ComputeSummary
} from '../types/platform';

/**
 * DemoPlatformStore: Fallback in-memory data store for Decentralized.Host
 * Acts as an authoritative, self-contained demonstration and fallback provider
 * when the external Python backend is not active or reachable.
 * Contains realistic telemetry for edge nodes, GPU rigs, self-hosted servers,
 * workloads, DePIN network integrations, and cryptographic evidence records.
 */
export class DemoPlatformStore {
  capabilities: PlatformCapabilities = {
    nodes: 'LIVE',
    deployments: 'LIVE',
    storage: 'LIVE',
    domains: 'LIVE',
    acme: 'LIVE',
    waf: 'LIVE',
    ddosTelemetry: 'DERIVED',
    visitorAnalytics: 'DERIVED',
    infraTelemetry: 'LIVE',
    billing: 'CONFIGURED',
    copilot: 'LIVE',
    evidenceLedger: 'LIVE'
  };

  depinIntegrations: DePINIntegration[] = [
    {
      id: 'depin-akash',
      name: 'Akash Network',
      symbol: 'AKT',
      category: 'Compute',
      description: 'Decentralized cloud compute marketplace leasing containerized CPU & GPU workloads.',
      status: 'Active',
      nodesConnected: 6,
      earningsTotal: '1,420 AKT (~$4,680)',
      icon: '⚡',
      protocolLink: 'https://akash.network'
    },
    {
      id: 'depin-render',
      name: 'Render Network',
      symbol: 'RENDER',
      category: 'AI & Rendering',
      description: 'Distributed GPU rendering and generative AI tensor compute engine.',
      status: 'Active',
      nodesConnected: 3,
      earningsTotal: '380 RENDER (~$2,240)',
      icon: '🎨',
      protocolLink: 'https://render.x.io'
    },
    {
      id: 'depin-filecoin',
      name: 'Filecoin / IPFS',
      symbol: 'FIL',
      category: 'Storage',
      description: 'Verifiable cryptographic content-addressed storage and decentralized retrieval market.',
      status: 'Active',
      nodesConnected: 7,
      earningsTotal: '284 FIL (~$1,190)',
      icon: '📦',
      protocolLink: 'https://filecoin.io'
    },
    {
      id: 'depin-livepeer',
      name: 'Livepeer',
      symbol: 'LPT',
      category: 'Compute',
      description: 'Open video infrastructure protocol routing live transcoding through edge node GPUs.',
      status: 'Connected',
      nodesConnected: 2,
      earningsTotal: '92 LPT (~$870)',
      icon: '📹',
      protocolLink: 'https://livepeer.org'
    },
    {
      id: 'depin-ionet',
      name: 'io.net Cloud',
      symbol: 'IO',
      category: 'AI & Rendering',
      description: 'Decentralized physical infrastructure network aggregating enterprise GPUs for ML clustering.',
      status: 'Active',
      nodesConnected: 4,
      earningsTotal: '412 IO (~$1,850)',
      icon: '🧠',
      protocolLink: 'https://io.net'
    },
    {
      id: 'depin-helium',
      name: 'Helium Network',
      symbol: 'HNT',
      category: 'Wireless & Bandwidth',
      description: 'Decentralized wireless IoT and 5G edge routing network.',
      status: 'Ready',
      nodesConnected: 1,
      earningsTotal: '45 HNT (~$270)',
      icon: '📡',
      protocolLink: 'https://helium.com'
    }
  ];

  selfHostingBenefits: SelfHostingBenefit[] = [
    {
      id: 'ben-lockin',
      title: 'Zero Cloud Monopolist Lock-In',
      description: 'Deploy anywhere: run workloads on bare-metal, edge Raspberry Pi, homelab servers, or hybrid hyperscalers with 100% data portability.',
      metric: '100% Sovereign',
      tag: 'Data Sovereignty'
    },
    {
      id: 'ben-costs',
      title: '70% Lower Compute Costs',
      description: 'Eliminate proprietary egress markups, overpriced managed databases, and cloud hyper-scaling taxes through peer-to-peer resource sharing.',
      metric: '72% Average Savings',
      tag: 'Cost Efficiency'
    },
    {
      id: 'ben-yields',
      title: 'Earn Passive Compute & DePIN Yields',
      description: 'Monetize idle CPU cores, unallocated storage blocks, and RTX/H100 GPU compute cycles by serving real global mesh traffic.',
      metric: '$840+/mo per Rig',
      tag: 'Network Rewards'
    },
    {
      id: 'ben-resilience',
      title: 'Censorship-Resistant Anycast Ingress',
      description: 'Traffic automatically reroutes across peer edge relays with no single point of failure, backed by cryptographic ED25519 attestations.',
      metric: '99.99% Uptime',
      tag: 'High Availability'
    }
  ];

  evidenceLedger: EvidenceRecord[] = [
    {
      recordId: 'QUAL-DH-2026-0925-A1',
      timestamp: '2026-09-25T13:40:12Z',
      sha256Digest: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
      signerFingerprint: 'ED25519:8f:9a:12:44:bc:39:aa:7e:90:54:1c:10:ab:44:98:22',
      status: 'VERIFIED',
      gatesPassed: [
        'GATE_A_BUILD',
        'GATE_B_TESTS',
        'GATE_C_API_TRUTH',
        'GATE_D_SECURITY_RBAC',
        'GATE_E_FAIL_CLOSED',
        'GATE_H_COPILOT_GROUNDING',
        'GATE_I_ACTION_SAFETY'
      ],
      gatesTotal: 7,
      sourceCommit: 'git:7b92f4c'
    },
    {
      recordId: 'QUAL-DH-2026-0924-B8',
      timestamp: '2026-09-24T18:22:45Z',
      sha256Digest: '5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8',
      signerFingerprint: 'ED25519:8f:9a:12:44:bc:39:aa:7e:90:54:1c:10:ab:44:98:22',
      status: 'SEALED',
      gatesPassed: [
        'GATE_A_BUILD',
        'GATE_B_TESTS',
        'GATE_C_API_TRUTH',
        'GATE_D_SECURITY_RBAC'
      ],
      gatesTotal: 4,
      sourceCommit: 'git:4a12e81'
    }
  ];

  nodes: NodeInfo[] = [
    {
      id: 'node-us-east-1',
      name: 'Ashburn Core Validator',
      region: 'North America',
      location: 'Ashburn, VA, USA',
      countryCode: 'US',
      ipAddress: '198.51.100.24',
      provider: 'Equinix Metal',
      role: 'Validator',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 34,
      cpuCores: 32,
      cpuModel: 'AMD EPYC 9354 32-Core',
      memoryPercent: 42,
      memoryUsedGb: 53.7,
      memoryTotalGb: 128,
      diskUsedGb: 2100,
      diskTotalGb: 8000,
      bandwidthUsedTb: 2.1,
      gpuEquipped: false,
      workloadCount: 8,
      workloads: [
        { id: 'wl-1', name: 'app.agentswarm.in', type: 'Docker Container', cpuPercent: 8, memoryMb: 1024, status: 'Running', deployedAt: '2h ago' },
        { id: 'wl-2', name: 'decentralized.host-ingress', type: 'Ingress Gateway', cpuPercent: 12, memoryMb: 2048, status: 'Running', deployedAt: '5d ago' },
        { id: 'wl-3', name: 'merkle-consensus-attestor', type: 'WASM Edge', cpuPercent: 6, memoryMb: 512, status: 'Running', deployedAt: '12d ago' },
        { id: 'wl-4', name: 'ipfs-pin-service', type: 'IPFS Daemon', cpuPercent: 5, memoryMb: 4096, status: 'Running', deployedAt: '30d ago' }
      ],
      uptimePercent: 99.99,
      uptimeDays: 142,
      uptimeScore: 99.9,
      proofsVerifiedCount: 48920,
      rewardsEarnedCredits: 384.5,
      supportedDePINTags: ['Akash Network', 'Filecoin / IPFS'],
      hardwareType: 'Bare-Metal',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 1,
      trustFingerprint: 'SHA256:7Bf4...e901',
      coordinates: [39.0438, -77.4874],
      evidence: {
        recordId: 'EV-NODE-USE1-901',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef0',
        signerFingerprint: 'NODE-PUBKEY:us-east-1',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'TPM_ATTESTATION', 'NET_LATENCY'],
        gatesTotal: 3
      }
    },
    {
      id: 'node-gpu-lon-01',
      name: 'London GPU Cluster Rig',
      region: 'Europe',
      location: 'London, UK',
      countryCode: 'GB',
      ipAddress: '198.51.100.82',
      provider: 'Self-Hosted Rig',
      role: 'GPU Compute Rig',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 48,
      cpuCores: 24,
      cpuModel: 'Intel Core i9-14900KS 24-Core',
      memoryPercent: 64,
      memoryUsedGb: 81.9,
      memoryTotalGb: 128,
      diskUsedGb: 3400,
      diskTotalGb: 8000,
      bandwidthUsedTb: 3.8,
      gpuEquipped: true,
      gpuModel: 'NVIDIA GeForce RTX 4090 (24GB VRAM) x 4',
      gpuCount: 4,
      gpuPercent: 78,
      gpuMemoryUsedGb: 74.8,
      gpuMemoryTotalGb: 96,
      workloadCount: 6,
      workloads: [
        { id: 'wl-gpu-1', name: 'deepseek-coder-7b-inference', type: 'AI Inference', cpuPercent: 18, memoryMb: 16384, status: 'Running', deployedAt: '1d ago' },
        { id: 'wl-gpu-2', name: 'render-network-worker', type: 'AI Inference', cpuPercent: 22, memoryMb: 24576, status: 'Running', deployedAt: '3d ago' },
        { id: 'wl-gpu-3', name: 'livepeer-video-transcoder', type: 'Docker Container', cpuPercent: 8, memoryMb: 8192, status: 'Running', deployedAt: '5d ago' }
      ],
      uptimePercent: 99.98,
      uptimeDays: 89,
      uptimeScore: 99.8,
      proofsVerifiedCount: 74120,
      rewardsEarnedCredits: 1280.0,
      supportedDePINTags: ['Render Network', 'io.net Cloud', 'Livepeer', 'Akash Network'],
      hardwareType: 'Self-Hosted Edge',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 2,
      trustFingerprint: 'SHA256:4d71...ba19',
      coordinates: [51.5074, -0.1278],
      evidence: {
        recordId: 'EV-NODE-GPULON-01',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: '7192a0e38459b109c1935817a39158c30981e49183491ca9138e9183401c918a',
        signerFingerprint: 'NODE-PUBKEY:gpu-lon-01',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'GPU_BENCHMARK', 'CUDA_VERIFIED'],
        gatesTotal: 3
      }
    },
    {
      id: 'node-homelab-kochi',
      name: 'Kochi Homelab Sovereign Edge',
      region: 'Asia',
      location: 'Kochi, Kerala, India',
      countryCode: 'IN',
      ipAddress: '203.0.113.14',
      provider: 'Homelab SBC (NVMe Edge)',
      role: 'Homelab Node',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 22,
      cpuCores: 8,
      cpuModel: 'Raspberry Pi 5 + NVMe M.2 (Broadcom BCM2712)',
      memoryPercent: 38,
      memoryUsedGb: 3.1,
      memoryTotalGb: 8,
      diskUsedGb: 380,
      diskTotalGb: 1000,
      bandwidthUsedTb: 0.8,
      gpuEquipped: false,
      workloadCount: 4,
      workloads: [
        { id: 'wl-hl-1', name: 'ewastekochi.com-edge-cache', type: 'WASM Edge', cpuPercent: 5, memoryMb: 256, status: 'Running', deployedAt: '10d ago' },
        { id: 'wl-hl-2', name: 'ipfs-local-retrieval-peer', type: 'IPFS Daemon', cpuPercent: 8, memoryMb: 1024, status: 'Running', deployedAt: '18d ago' },
        { id: 'wl-hl-3', name: 'dns-edge-challenge-responder', type: 'WASM Edge', cpuPercent: 3, memoryMb: 128, status: 'Running', deployedAt: '25d ago' }
      ],
      uptimePercent: 99.96,
      uptimeDays: 74,
      uptimeScore: 99.7,
      proofsVerifiedCount: 29810,
      rewardsEarnedCredits: 198.4,
      supportedDePINTags: ['Filecoin / IPFS', 'Helium Network'],
      hardwareType: 'Homelab SBC',
      architecture: 'ARM64',
      lastHeartbeatSecondsAgo: 3,
      trustFingerprint: 'SHA256:99ff...kochi01',
      coordinates: [9.9312, 76.2673],
      evidence: {
        recordId: 'EV-NODE-KOCHI-01',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'd198304918e3094819a384918c30918e4091830491830491834918304918c309',
        signerFingerprint: 'NODE-PUBKEY:homelab-kochi',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'TPM_ATTESTATION'],
        gatesTotal: 2
      }
    },
    {
      id: 'node-eu-central-1',
      name: 'Frankfurt Storage Relay',
      region: 'Europe',
      location: 'Frankfurt, Germany',
      countryCode: 'DE',
      ipAddress: '203.0.113.88',
      provider: 'Hetzner Dedicated',
      role: 'Storage Replicator',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 31,
      cpuCores: 24,
      cpuModel: 'AMD Ryzen 9 7950X3D',
      memoryPercent: 54,
      memoryUsedGb: 34.5,
      memoryTotalGb: 64,
      diskUsedGb: 5800,
      diskTotalGb: 12000,
      bandwidthUsedTb: 2.9,
      gpuEquipped: false,
      workloadCount: 7,
      workloads: [
        { id: 'wl-fra-1', name: 'ipfs-geo-sharding-replicator', type: 'IPFS Daemon', cpuPercent: 14, memoryMb: 8192, status: 'Running', deployedAt: '40d ago' },
        { id: 'wl-fra-2', name: 'codingagent.in-runner', type: 'Docker Container', cpuPercent: 9, memoryMb: 2048, status: 'Running', deployedAt: '2d ago' }
      ],
      uptimePercent: 99.98,
      uptimeDays: 98,
      uptimeScore: 99.9,
      proofsVerifiedCount: 61840,
      rewardsEarnedCredits: 512.0,
      supportedDePINTags: ['Filecoin / IPFS', 'Akash Network'],
      hardwareType: 'Bare-Metal',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 2,
      trustFingerprint: 'SHA256:91Aa...42c1',
      coordinates: [50.1109, 8.6821],
      evidence: {
        recordId: 'EV-NODE-EUC1-902',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef01',
        signerFingerprint: 'NODE-PUBKEY:eu-central-1',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'TPM_ATTESTATION', 'NET_LATENCY'],
        gatesTotal: 3
      }
    },
    {
      id: 'node-ap-southeast-1',
      name: 'Singapore Edge Gateway',
      region: 'Asia',
      location: 'Singapore, Jurong East',
      countryCode: 'SG',
      ipAddress: '198.51.100.104',
      provider: 'DigitalOcean Edge PoP',
      role: 'Edge Gateway',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 36,
      cpuCores: 16,
      cpuModel: 'Intel Xeon Platinum 8488C',
      memoryPercent: 44,
      memoryUsedGb: 28.1,
      memoryTotalGb: 64,
      diskUsedGb: 1950,
      diskTotalGb: 4000,
      bandwidthUsedTb: 3.4,
      gpuEquipped: false,
      workloadCount: 5,
      workloads: [
        { id: 'wl-sg-1', name: 'edge-anycast-tls-proxy', type: 'Ingress Gateway', cpuPercent: 16, memoryMb: 4096, status: 'Running', deployedAt: '14d ago' },
        { id: 'wl-sg-2', name: 'apac-routing-relay', type: 'WASM Edge', cpuPercent: 8, memoryMb: 1024, status: 'Running', deployedAt: '20d ago' }
      ],
      uptimePercent: 99.99,
      uptimeDays: 87,
      uptimeScore: 99.9,
      proofsVerifiedCount: 52190,
      rewardsEarnedCredits: 420.0,
      supportedDePINTags: ['Akash Network'],
      hardwareType: 'Cloud Edge',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 4,
      trustFingerprint: 'SHA256:c023...fd89',
      coordinates: [1.3521, 103.8198],
      evidence: {
        recordId: 'EV-NODE-APSE1-903',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef012',
        signerFingerprint: 'NODE-PUBKEY:ap-southeast-1',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'TPM_ATTESTATION'],
        gatesTotal: 2
      }
    },
    {
      id: 'node-gpu-austin-01',
      name: 'Austin DePIN AI Supernode',
      region: 'North America',
      location: 'Austin, TX, USA',
      countryCode: 'US',
      ipAddress: '198.51.100.155',
      provider: 'Self-Hosted AI Lab',
      role: 'DePIN Worker',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 52,
      cpuCores: 64,
      cpuModel: 'AMD EPYC 9554 64-Core',
      memoryPercent: 58,
      memoryUsedGb: 148.5,
      memoryTotalGb: 256,
      diskUsedGb: 4600,
      diskTotalGb: 16000,
      bandwidthUsedTb: 5.2,
      gpuEquipped: true,
      gpuModel: 'NVIDIA H100 80GB SXM5 x 2',
      gpuCount: 2,
      gpuPercent: 84,
      gpuMemoryUsedGb: 134.4,
      gpuMemoryTotalGb: 160,
      workloadCount: 4,
      workloads: [
        { id: 'wl-atx-1', name: 'llama-3.3-70b-tensor-slice', type: 'AI Inference', cpuPercent: 32, memoryMb: 65536, status: 'Running', deployedAt: '6h ago' },
        { id: 'wl-atx-2', name: 'ionet-mesh-cluster-agent', type: 'Docker Container', cpuPercent: 12, memoryMb: 16384, status: 'Running', deployedAt: '4d ago' }
      ],
      uptimePercent: 99.97,
      uptimeDays: 54,
      uptimeScore: 99.8,
      proofsVerifiedCount: 98400,
      rewardsEarnedCredits: 2150.0,
      supportedDePINTags: ['io.net Cloud', 'Render Network', 'Akash Network'],
      hardwareType: 'Self-Hosted Edge',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 2,
      trustFingerprint: 'SHA256:h100...austin88',
      coordinates: [30.2672, -97.7431],
      evidence: {
        recordId: 'EV-NODE-ATX-H100',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'e491830491830491830491830491830491830491830491830491830491830491',
        signerFingerprint: 'NODE-PUBKEY:atx-h100',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'TPM_ATTESTATION', 'H100_ATTESTATION'],
        gatesTotal: 3
      }
    },
    {
      id: 'node-eu-north-1',
      name: 'Stockholm Eco Hydro-Edge',
      region: 'Europe',
      location: 'Stockholm, Sweden',
      countryCode: 'SE',
      ipAddress: '203.0.113.79',
      provider: 'OVHcloud Eco-DC',
      role: 'Validator',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 24,
      cpuCores: 32,
      cpuModel: 'AMD EPYC 7543 32-Core',
      memoryPercent: 32,
      memoryUsedGb: 41.0,
      memoryTotalGb: 128,
      diskUsedGb: 1400,
      diskTotalGb: 6000,
      bandwidthUsedTb: 1.6,
      gpuEquipped: false,
      workloadCount: 4,
      workloads: [
        { id: 'wl-arn-1', name: 'green-compute-attestation', type: 'WASM Edge', cpuPercent: 6, memoryMb: 512, status: 'Running', deployedAt: '30d ago' }
      ],
      uptimePercent: 99.99,
      uptimeDays: 168,
      uptimeScore: 100,
      proofsVerifiedCount: 88200,
      rewardsEarnedCredits: 620.0,
      supportedDePINTags: ['Akash Network', 'Filecoin / IPFS'],
      hardwareType: 'Bare-Metal',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 1,
      trustFingerprint: 'SHA256:ef20...119b',
      coordinates: [59.3293, 18.0686],
      evidence: {
        recordId: 'EV-NODE-EUN1-909',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: '3456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012',
        signerFingerprint: 'NODE-PUBKEY:eu-north-1',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'TPM_ATTESTATION'],
        gatesTotal: 2
      }
    },
    {
      id: 'node-me-dxb-1',
      name: 'Dubai Edge Gateway',
      region: 'Middle East',
      location: 'Dubai, UAE',
      countryCode: 'AE',
      ipAddress: '203.0.113.142',
      provider: 'Khazna DC',
      role: 'Edge Gateway',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 33,
      cpuCores: 24,
      cpuModel: 'Intel Xeon Gold 6338',
      memoryPercent: 44,
      memoryUsedGb: 28.2,
      memoryTotalGb: 64,
      diskUsedGb: 1100,
      diskTotalGb: 4000,
      bandwidthUsedTb: 1.8,
      gpuEquipped: false,
      workloadCount: 3,
      workloads: [
        { id: 'wl-dxb-1', name: 'mena-tls-edge-terminator', type: 'Ingress Gateway', cpuPercent: 14, memoryMb: 2048, status: 'Running', deployedAt: '12d ago' }
      ],
      uptimePercent: 99.96,
      uptimeDays: 78,
      uptimeScore: 99.7,
      proofsVerifiedCount: 31400,
      rewardsEarnedCredits: 310.0,
      supportedDePINTags: ['Akash Network'],
      hardwareType: 'Cloud Edge',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 3,
      trustFingerprint: 'SHA256:77bb...ee12',
      coordinates: [25.2048, 55.2708],
      evidence: {
        recordId: 'EV-NODE-MED1-910',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: '456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123',
        signerFingerprint: 'NODE-PUBKEY:me-central-1',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY', 'NET_LATENCY'],
        gatesTotal: 2
      }
    },
    {
      id: 'node-sa-east-1',
      name: 'São Paulo Storage Node',
      region: 'South America',
      location: 'São Paulo, Brazil',
      countryCode: 'BR',
      ipAddress: '198.51.100.198',
      provider: 'Vultr Bare-Metal',
      role: 'Storage Replicator',
      status: 'Degraded',
      desiredState: 'Active',
      observedState: 'Degraded',
      cpuPercent: 76,
      cpuCores: 16,
      cpuModel: 'AMD EPYC 7302P',
      memoryPercent: 82,
      memoryUsedGb: 52.5,
      memoryTotalGb: 64,
      diskUsedGb: 3600,
      diskTotalGb: 4000,
      bandwidthUsedTb: 1.4,
      gpuEquipped: false,
      workloadCount: 2,
      workloads: [
        { id: 'wl-gru-1', name: 'sa-ipfs-peer', type: 'IPFS Daemon', cpuPercent: 38, memoryMb: 16384, status: 'Degraded', deployedAt: '20d ago' }
      ],
      uptimePercent: 99.21,
      uptimeDays: 3,
      uptimeScore: 92.5,
      proofsVerifiedCount: 14100,
      rewardsEarnedCredits: 140.0,
      supportedDePINTags: ['Filecoin / IPFS'],
      hardwareType: 'Cloud Edge',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 28,
      trustFingerprint: 'SHA256:77bc...ff30',
      coordinates: [-23.5505, -46.6333],
      evidence: {
        recordId: 'EV-NODE-SAE1-905',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'e5f67890123456789abcdef0123456789abcdef0123456789abcdef01234',
        signerFingerprint: 'NODE-PUBKEY:sa-east-1',
        status: 'SEALED',
        gatesPassed: ['HARDWARE_INTEGRITY'],
        gatesTotal: 2
      }
    },
    {
      id: 'node-au-syd-1',
      name: 'Sydney Oceanic Edge',
      region: 'Oceania',
      location: 'Sydney, Australia',
      countryCode: 'AU',
      ipAddress: '203.0.113.210',
      provider: 'Equinix SY3',
      role: 'Edge Gateway',
      status: 'Online',
      desiredState: 'Active',
      observedState: 'Active',
      cpuPercent: 28,
      cpuCores: 16,
      cpuModel: 'Intel Xeon E-2388G',
      memoryPercent: 39,
      memoryUsedGb: 25.0,
      memoryTotalGb: 64,
      diskUsedGb: 1200,
      diskTotalGb: 3000,
      bandwidthUsedTb: 1.1,
      gpuEquipped: false,
      workloadCount: 3,
      workloads: [
        { id: 'wl-syd-1', name: 'oceania-anycast-node', type: 'Ingress Gateway', cpuPercent: 11, memoryMb: 2048, status: 'Running', deployedAt: '19d ago' }
      ],
      uptimePercent: 99.98,
      uptimeDays: 45,
      uptimeScore: 99.8,
      proofsVerifiedCount: 22800,
      rewardsEarnedCredits: 260.0,
      supportedDePINTags: ['Akash Network'],
      hardwareType: 'Cloud Edge',
      architecture: 'x86_64',
      lastHeartbeatSecondsAgo: 2,
      trustFingerprint: 'SHA256:1a8d...99cc',
      coordinates: [-33.8688, 151.2093],
      evidence: {
        recordId: 'EV-NODE-AUS1-906',
        timestamp: '2026-09-25T14:10:00Z',
        sha256Digest: 'f67890123456789abcdef0123456789abcdef0123456789abcdef012345',
        signerFingerprint: 'NODE-PUBKEY:au-syd-1',
        status: 'VERIFIED',
        gatesPassed: ['HARDWARE_INTEGRITY'],
        gatesTotal: 2
      }
    }
  ];

  applications: Application[] = [
    {
      id: 'app-agentswarm',
      name: 'agentswarm.in',
      domain: 'agentswarm.in',
      type: 'Web App',
      status: 'Online',
      desiredReplicas: 3,
      observedHealthyReplicas: 3,
      nodesAssigned: ['node-us-east-1', 'node-eu-central-1', 'node-ap-southeast-1'],
      regionsAssigned: ['North America', 'Europe', 'Asia'],
      version: 'v3.1.0',
      lastDeploymentTime: '2026-09-25T12:00:00Z',
      visitors30d: 28400,
      pageViews30d: 89000,
      bandwidthUsedGb: 540,
      avgResponseMs: 145,
      envVars: [
        { key: 'PORT', isSecret: false, maskedValue: '3000' },
        { key: 'GEMINI_API_KEY', isSecret: true, maskedValue: '••••••••••••••••' }
      ],
      logs: [
        '[2026-09-25 14:15:00 UTC] [deploy] Deployment v3.1.0 active and running across 3 geographic zones',
        '[2026-09-25 14:12:30 UTC] [health] Probes OK across 3 regions (Ashburn, Frankfurt, Singapore)'
      ],
      evidence: {
        recordId: 'EV-APP-ASW-01',
        timestamp: '2026-09-25T12:00:00Z',
        sha256Digest: '5a819b183401c918a7192a0e38459b109c1935817a39158c30981e49183491ca',
        signerFingerprint: 'DEPLOY-KEY:agentswarm-ci',
        status: 'VERIFIED',
        gatesPassed: ['BUNDLE_INTEGRITY', 'TESTS_PASSED'],
        gatesTotal: 2
      }
    },
    {
      id: 'app-ewastekochi',
      name: 'ewastekochi.com',
      domain: 'ewastekochi.com',
      type: 'Static Site',
      status: 'Online',
      desiredReplicas: 3,
      observedHealthyReplicas: 3,
      nodesAssigned: ['node-homelab-kochi', 'node-eu-central-1', 'node-ap-southeast-1'],
      regionsAssigned: ['Asia', 'Europe'],
      version: 'v2.4.1',
      lastDeploymentTime: '2026-09-25T10:14:00Z',
      visitors30d: 42100,
      pageViews30d: 138700,
      bandwidthUsedGb: 720,
      avgResponseMs: 180,
      envVars: [
        { key: 'VITE_APP_ENV', isSecret: false, maskedValue: 'production' }
      ],
      logs: [
        '[2026-09-25 14:10:02 UTC] [routing] Edge ingress healthy across self-hosted and cloud nodes',
        '[2026-09-25 14:08:45 UTC] [acme] Certificate renewal check: 498 days remaining'
      ],
      evidence: {
        recordId: 'EV-APP-EWK-01',
        timestamp: '2026-09-25T10:14:00Z',
        sha256Digest: 'c8f3918a09f3e498c8a14917a151b75249a37e1903e1451fca82d398f12a9c11',
        signerFingerprint: 'DEPLOY-KEY:fe-ewaste',
        status: 'VERIFIED',
        gatesPassed: ['BUNDLE_INTEGRITY', 'STATIC_SECURITY_SCAN', 'TLS_PROVENANCE'],
        gatesTotal: 3
      }
    },
    {
      id: 'app-codingagent',
      name: 'codingagent.in',
      domain: 'codingagent.in',
      type: 'Docker App',
      status: 'Online',
      desiredReplicas: 2,
      observedHealthyReplicas: 2,
      nodesAssigned: ['node-us-east-1', 'node-gpu-lon-01'],
      regionsAssigned: ['North America', 'Europe'],
      version: 'v1.4.2',
      lastDeploymentTime: '2026-09-24T16:00:00Z',
      visitors30d: 12900,
      pageViews30d: 30400,
      bandwidthUsedGb: 220,
      avgResponseMs: 290,
      envVars: [],
      logs: [
        '[2026-09-25 13:30:00 UTC] [docker] Container sha256:719f... healthy on 2 hosts (GPU acceleration enabled)',
        '[2026-09-25 13:00:00 UTC] [acme] Warning: Certificate expires in 24 days'
      ],
      evidence: {
        recordId: 'EV-APP-CDA-05',
        timestamp: '2026-09-24T16:00:00Z',
        sha256Digest: 'e491830491830491830491830491830491830491830491830491830491830491',
        signerFingerprint: 'DEPLOY-KEY:codingagent',
        status: 'VERIFIED',
        gatesPassed: ['CONTAINER_SCAN', 'HEALTH_GATE'],
        gatesTotal: 2
      }
    },
    {
      id: 'app-agentswarm-portal',
      name: 'app.agentswarm.in',
      domain: 'app.agentswarm.in',
      type: 'Web App',
      status: 'Online',
      desiredReplicas: 4,
      observedHealthyReplicas: 4,
      nodesAssigned: ['node-us-east-1', 'node-eu-central-1', 'node-ap-southeast-1', 'node-au-syd-1'],
      regionsAssigned: ['North America', 'Europe', 'Asia', 'Oceania'],
      version: 'v3.0.2',
      lastDeploymentTime: '2026-09-25T14:12:00Z',
      visitors30d: 18700,
      pageViews30d: 66300,
      bandwidthUsedGb: 380,
      avgResponseMs: 210,
      envVars: [
        { key: 'SWARM_ORCHESTRATOR', isSecret: false, maskedValue: 'mesh-v2' }
      ],
      logs: [
        '[2026-09-25 14:12:05 UTC] [deploy] Deployment v3.0.2 verified healthy across 4 replicas',
        '[2026-09-25 14:12:01 UTC] [routing] New version promoted to 100% traffic'
      ],
      evidence: {
        recordId: 'EV-APP-ASW-03',
        timestamp: '2026-09-25T14:12:00Z',
        sha256Digest: 'd198304918e3094819a384918c30918e4091830491830491834918304918c309',
        signerFingerprint: 'DEPLOY-KEY:swarm',
        status: 'VERIFIED',
        gatesPassed: ['SMOKE_TESTS', 'HEALTH_GATE', 'TLS_VERIFIED'],
        gatesTotal: 3
      }
    },
    {
      id: 'app-bestaiagent',
      name: 'bestaiagent.in',
      domain: 'bestaiagent.in',
      type: 'Web App',
      status: 'Online',
      desiredReplicas: 3,
      observedHealthyReplicas: 3,
      nodesAssigned: ['node-gpu-lon-01', 'node-gpu-austin-01', 'node-ap-southeast-1'],
      regionsAssigned: ['Europe', 'North America', 'Asia'],
      version: 'v1.8.0',
      lastDeploymentTime: '2026-09-25T08:30:00Z',
      visitors30d: 28400,
      pageViews30d: 92100,
      bandwidthUsedGb: 480,
      avgResponseMs: 195,
      envVars: [],
      logs: [
        '[2026-09-25 14:11:00 UTC] [runtime] GPU Tensor cores active for query inference',
        '[2026-09-25 14:02:18 UTC] [health] /api/health HTTP 200 OK (18ms)'
      ],
      evidence: {
        recordId: 'EV-APP-BAI-02',
        timestamp: '2026-09-25T08:30:00Z',
        sha256Digest: '7192a0e38459b109c1935817a39158c30981e49183491ca9138e9183401c918a',
        signerFingerprint: 'DEPLOY-KEY:ai-agents',
        status: 'VERIFIED',
        gatesPassed: ['CONTAINER_SCAN', 'HEALTH_GATE', 'TLS_VERIFIED'],
        gatesTotal: 3
      }
    }
  ];

  deployments: Deployment[] = [
    {
      id: 'dep-9481',
      appId: 'app-agentswarm-portal',
      appName: 'app.agentswarm.in',
      domain: 'app.agentswarm.in',
      sourceType: 'Git',
      sourceReference: 'github.com/agentswarm/web:main@7b92f4c',
      status: 'VERIFIED',
      currentStage: 'VERIFIED',
      stagesCompleted: [
        'VALIDATING',
        'SCHEDULING',
        'ARTIFACT_TRANSFER',
        'RUNTIME_CREATION',
        'HEALTH_CHECK',
        'ROUTING',
        'VERIFIED'
      ],
      regions: ['North America', 'Europe', 'Asia', 'Oceania'],
      createdAt: '2026-09-25T14:07:00Z',
      completedAt: '2026-09-25T14:12:00Z',
      commitHash: '7b92f4c2810a9f',
      artifactDigest: 'sha256:c8a9183491830491834091830491830491830491830491830491830491830491',
      logs: [
        { timestamp: '14:07:01', stage: 'VALIDATING', message: 'Validating Git repository signature and dependencies', severity: 'info' },
        { timestamp: '14:07:45', stage: 'SCHEDULING', message: 'Assigned 4 target nodes: Ashburn, Frankfurt, Singapore, Sydney', severity: 'info' },
        { timestamp: '14:08:30', stage: 'ARTIFACT_TRANSFER', message: 'Replicated bundle digests across 4 hosts (CID verified)', severity: 'info' },
        { timestamp: '14:09:50', stage: 'RUNTIME_CREATION', message: 'Containers initialized with zero-downtime blue/green', severity: 'success' },
        { timestamp: '14:11:10', stage: 'HEALTH_CHECK', message: 'Heartbeat & synthetic probing passed (4/4 healthy)', severity: 'success' },
        { timestamp: '14:11:40', stage: 'ROUTING', message: 'Edge DNS updated with anycast weights', severity: 'success' },
        { timestamp: '14:12:00', stage: 'VERIFIED', message: 'Deployment sealed into cryptographic evidence ledger', severity: 'success' }
      ],
      evidence: {
        recordId: 'EV-DEP-9481',
        timestamp: '2026-09-25T14:12:00Z',
        sha256Digest: 'c8a9183491830491834091830491830491830491830491830491830491830491',
        signerFingerprint: 'DEPLOY-KEY:swarm',
        status: 'VERIFIED',
        gatesPassed: ['GATE_A_BUILD', 'GATE_B_TESTS', 'GATE_C_API_TRUTH'],
        gatesTotal: 3
      }
    }
  ];

  storageBuckets: StorageBucket[] = [
    {
      id: 'bucket-website-assets',
      name: 'website-assets',
      totalSizeFormatted: '248 GB',
      fileCount: 4210,
      regions: ['North America', 'Europe', 'Asia', 'South America', 'Africa', 'Oceania'],
      replicationFactor: 3,
      createdAt: '2026-08-10'
    },
    {
      id: 'bucket-backups',
      name: 'backups',
      totalSizeFormatted: '1.2 TB',
      fileCount: 340,
      regions: ['North America', 'Europe', 'Asia', 'South America', 'Africa', 'Oceania'],
      replicationFactor: 3,
      createdAt: '2026-08-01'
    },
    {
      id: 'bucket-media',
      name: 'media',
      totalSizeFormatted: '682 GB',
      fileCount: 8900,
      regions: ['North America', 'Europe', 'Asia', 'Oceania'],
      replicationFactor: 3,
      createdAt: '2026-08-15'
    }
  ];

  storageObjects: StorageObject[] = [
    {
      id: 'obj-website-assets',
      name: 'website-assets',
      bucket: 'website-assets',
      type: 'Folder',
      sizeBytes: 266287972352,
      sizeFormatted: '248 GB',
      replicasTarget: 3,
      replicasObserved: 3,
      nodesPlacements: ['node-us-east-1', 'node-eu-central-1', 'node-ap-southeast-1'],
      locationSummary: 'Global (6 regions)',
      modified: 'Sep 22, 2026 14:22',
      sha256Cid: 'QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco',
      verificationStatus: 'VERIFIED'
    },
    {
      id: 'obj-backups',
      name: 'backups',
      bucket: 'backups',
      type: 'Folder',
      sizeBytes: 1319413953331,
      sizeFormatted: '1.2 TB',
      replicasTarget: 3,
      replicasObserved: 3,
      nodesPlacements: ['node-us-east-1', 'node-eu-central-1', 'node-au-syd-1'],
      locationSummary: 'Global (6 regions)',
      modified: 'Sep 21, 2026 03:11',
      sha256Cid: 'QmZtmD2qt8fJpq3CLDHvdzsKfNs5CXjP3nnGoQD1ndGCDJ',
      verificationStatus: 'VERIFIED'
    },
    {
      id: 'obj-media',
      name: 'media',
      bucket: 'media',
      type: 'Folder',
      sizeBytes: 732302344192,
      sizeFormatted: '682 GB',
      replicasTarget: 3,
      replicasObserved: 3,
      nodesPlacements: ['node-us-east-1', 'node-homelab-kochi', 'node-ap-southeast-1'],
      locationSummary: 'Global (5 regions)',
      modified: 'Sep 19, 2026 18:45',
      sha256Cid: 'QmbWqxBEKC3P8tTXBSFL123b56789abcdef0123456789a',
      verificationStatus: 'VERIFIED'
    }
  ];

  domains: DomainRecord[] = [
    {
      id: 'dom-dh',
      name: 'decentralized.host',
      type: 'Traditional',
      status: 'Active',
      dnsProvider: 'Distributed DNS · 6 nodes worldwide',
      nodesWorldwide: 6,
      expiryDate: 'Feb 12, 2027',
      daysRemaining: 523,
      autoRenew: true,
      assignedAppId: 'app-decentralized-host',
      dnsRecords: [
        { id: 'rec-1', type: 'A', name: '@', value: '198.51.100.24', ttl: 300 },
        { id: 'rec-2', type: 'CNAME', name: 'www', value: 'decentralized.host', ttl: 300 }
      ]
    },
    {
      id: 'dom-myapp-web3',
      name: 'myapp.web3',
      type: 'Web3 (ENS)',
      status: 'Active',
      dnsProvider: 'IPFS + Gateway · 4 nodes worldwide',
      nodesWorldwide: 4,
      expiryDate: '—',
      daysRemaining: null,
      autoRenew: false,
      web3ContentHash: 'ipfs://QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco',
      dnsRecords: []
    },
    {
      id: 'dom-ewastekochi',
      name: 'ewastekochi.com',
      type: 'Traditional',
      status: 'Active',
      dnsProvider: 'Edge Network · 6 nodes worldwide',
      nodesWorldwide: 6,
      expiryDate: 'Jan 18, 2027',
      daysRemaining: 498,
      autoRenew: true,
      assignedAppId: 'app-ewastekochi',
      dnsRecords: [
        { id: 'rec-5', type: 'A', name: '@', value: '198.51.100.24', ttl: 300 },
        { id: 'rec-6', type: 'CNAME', name: 'www', value: 'ewastekochi.com', ttl: 300 }
      ]
    },
    {
      id: 'dom-codingagent',
      name: 'codingagent.in',
      type: 'Traditional',
      status: 'Expiring Soon',
      dnsProvider: 'Edge Network · 4 nodes worldwide',
      nodesWorldwide: 4,
      expiryDate: 'Oct 10, 2026',
      daysRemaining: 24,
      autoRenew: true,
      assignedAppId: 'app-codingagent',
      dnsRecords: [
        { id: 'rec-7', type: 'A', name: '@', value: '198.51.100.82', ttl: 300 }
      ]
    },
    {
      id: 'dom-bestaiagent',
      name: 'bestaiagent.in',
      type: 'Traditional',
      status: 'Active',
      dnsProvider: 'Global CDN · 6 nodes worldwide',
      nodesWorldwide: 6,
      expiryDate: 'Mar 2, 2027',
      daysRemaining: 541,
      autoRenew: true,
      assignedAppId: 'app-bestaiagent',
      dnsRecords: [
        { id: 'rec-8', type: 'A', name: '@', value: '198.51.100.155', ttl: 300 }
      ]
    },
    {
      id: 'dom-agentswarm',
      name: 'app.agentswarm.in',
      type: 'Traditional',
      status: 'Active',
      dnsProvider: 'Edge Network · 4 nodes worldwide',
      nodesWorldwide: 4,
      expiryDate: 'Dec 22, 2026',
      daysRemaining: 431,
      autoRenew: true,
      assignedAppId: 'app-agentswarm-portal',
      dnsRecords: [
        { id: 'rec-10', type: 'A', name: '@', value: '203.0.113.88', ttl: 300 }
      ]
    }
  ];

  certificates: SSLCertificate[] = [
    {
      id: 'cert-1',
      domain: 'decentralized.host',
      type: 'DV',
      status: 'Valid',
      issuedBy: "Let's Encrypt",
      issuedDate: '2026-05-04',
      expiryDate: 'Nov 4, 2026',
      daysRemaining: 423,
      autoRenew: true,
      fingerprintSha256: '7E:90:3A:44:B1:92:0C:88:51:24:D3:9F:88:AA:10:4E',
      keyType: 'ECDSA P-256'
    },
    {
      id: 'cert-2',
      domain: 'app.agentswarm.in',
      type: 'DV',
      status: 'Valid',
      issuedBy: "Let's Encrypt",
      issuedDate: '2026-07-18',
      expiryDate: 'Jan 18, 2027',
      daysRemaining: 498,
      autoRenew: true,
      fingerprintSha256: '4A:22:9C:10:E1:83:90:54:1C:88:7B:A2:33:41:99:FF',
      keyType: 'ECDSA P-256'
    },
    {
      id: 'cert-3',
      domain: 'ewastekochi.com',
      type: 'DV',
      status: 'Valid',
      issuedBy: 'ZeroSSL',
      issuedDate: '2026-09-02',
      expiryDate: 'Mar 2, 2027',
      daysRemaining: 541,
      autoRenew: true,
      fingerprintSha256: '99:1C:44:E2:08:91:AA:55:12:44:CC:89:12:00:81:AE',
      keyType: 'RSA 2048'
    },
    {
      id: 'cert-4',
      domain: 'codingagent.in',
      type: 'DV',
      status: 'Expiring Soon',
      issuedBy: "Let's Encrypt",
      issuedDate: '2026-06-10',
      expiryDate: 'Oct 10, 2026',
      daysRemaining: 24,
      autoRenew: true,
      fingerprintSha256: '12:FF:33:AA:99:88:77:66:55:44:33:22:11:00:EE:DD',
      keyType: 'ECDSA P-256'
    },
    {
      id: 'cert-5',
      domain: 'bestaiagent.in',
      type: 'DV',
      status: 'Valid',
      issuedBy: 'Cloudflare',
      issuedDate: '2026-08-22',
      expiryDate: 'Dec 22, 2026',
      daysRemaining: 431,
      autoRenew: true,
      fingerprintSha256: '88:44:AA:90:22:33:11:55:77:99:BB:DD:FF:00:12:34',
      keyType: 'ECDSA P-256'
    }
  ];

  securityEvaluation: SecurityEvaluation = {
    score: 98,
    statusText: 'Excellent',
    passedChecks: [
      { id: 'chk-tls', title: 'SSL/TLS Encryption Enforced', detail: 'All ingress routes terminate on TLS 1.3 with HSTS enabled.' },
      { id: 'chk-ddos', title: 'DDoS Protection Edge Filters', detail: 'SYN flood & layer 7 HTTP flood rate-limiting active on all edge nodes.' },
      { id: 'chk-waf', title: 'Firewall (WAF) Rules Active', detail: 'OWASP Top 10 ruleset active; SQLi, XSS, and path-traversal blocked.' },
      { id: 'chk-headers', title: 'Security Headers Configured', detail: 'Content-Security-Policy, X-Frame-Options, X-Content-Type-Options active.' },
      { id: 'chk-malware', title: 'Malware & Binary Scan', detail: 'Artifact upload signatures verified against known signature databases.' },
      { id: 'chk-secrets', title: 'Zero Secrets In Client Bundles', detail: 'Static code analyzer verified no credentials leaked in front-facing artifacts.' },
      { id: 'chk-audit', title: 'Immutable Audit Logging', detail: 'Audit trail signed and appended to immutable qualification log.' }
    ],
    warningChecks: [
      {
        id: 'chk-exp-cert',
        title: 'SSL Certificate Expiring in 24 Days',
        detail: 'Domain codingagent.in certificate expires on Oct 10, 2026.',
        recommendation: 'Trigger ACME renewal now or verify automated DNS TXT challenge responder.'
      },
      {
        id: 'chk-node-load',
        title: 'Node sa-east-1 High Disk & Memory Load',
        detail: 'Observed disk usage at 90% (3.6 TB / 4.0 TB) and RAM at 82%.',
        recommendation: 'Cordon node or drain non-critical workloads to prevent OOM/degradation.'
      }
    ],
    failedChecks: [],
    wafRulesActive: 18,
    ddosEdgeStatus: 'Active',
    threatsBlockedCount: 1482,
    threatTelemetryAvailable: true,
    securityHeadersConfigured: true
  };

  teamMembers: TeamMember[] = [
    {
      id: 'usr-1',
      name: 'Febin Francis',
      email: 'febin@decentralized.host',
      role: 'Owner',
      teams: ['All Teams', '+2'],
      accessLevel: 'Full Access',
      status: 'Active',
      lastActive: 'Now',
      avatarInitials: 'F'
    },
    {
      id: 'usr-2',
      name: 'Chithra Mathew',
      email: 'chithra@decentralized.host',
      role: 'Admin',
      teams: ['Infrastructure', '+1'],
      accessLevel: 'Full Access',
      status: 'Active',
      lastActive: '5m ago',
      avatarInitials: 'C'
    },
    {
      id: 'usr-3',
      name: 'Akhil R',
      email: 'akhil@decentralized.host',
      role: 'Developer',
      teams: ['Development'],
      accessLevel: 'Deploy & Manage',
      status: 'Active',
      lastActive: '12m ago',
      avatarInitials: 'A'
    }
  ];

  permissionMatrix: TeamPermissionMatrix = {
    deployApps: true,
    manageDomains: true,
    manageStorage: true,
    sslAndSecurity: true,
    viewAnalytics: true,
    drainNodes: false,
    manageBilling: false
  };

  events: PlatformEvent[] = [
    {
      id: 'evt-101',
      timestamp: '5m ago',
      type: 'deployment.completed',
      actor: 'system/deploy',
      resourceType: 'Application',
      resourceId: 'app.agentswarm.in',
      severity: 'success',
      message: 'Deployment completed across 4 geo-replicated edge hosts'
    },
    {
      id: 'evt-102',
      timestamp: '12m ago',
      type: 'node.joined',
      actor: 'mesh/gossip',
      resourceType: 'Node',
      resourceId: 'homelab-kochi',
      severity: 'info',
      message: 'Kochi Homelab Sovereign Edge attested TPM and enrolled into mesh'
    },
    {
      id: 'evt-103',
      timestamp: '1h ago',
      type: 'depin.yield_claimed',
      actor: 'system/rewards',
      resourceType: 'Node',
      resourceId: 'node-gpu-lon-01',
      severity: 'success',
      message: 'Render Network cluster claimed 380 RENDER tokens'
    },
    {
      id: 'evt-104',
      timestamp: '2h ago',
      type: 'ssl.renewed',
      actor: 'acme/letsencrypt',
      resourceType: 'SSL',
      resourceId: 'ewastekochi.com',
      severity: 'success',
      message: 'SSL certificate renewed automatically'
    }
  ];

  settings = {
    organizationName: 'Decentralized.Host',
    timezone: '(GMT+05:30) Asia/Kolkata',
    language: 'English (Default)',
    dateFormat: 'Sep 24, 2026',
    currency: 'USD - US Dollar ($)',
    theme: 'Dark',
    accentColor: '#3B82F6',
    density: 'Comfortable (Default)',
    account: {
      fullName: 'Febin Francis',
      email: 'febin@decentralized.host',
      phone: '+91 98765 43210',
      jobTitle: 'Owner',
      company: 'Decentralized.Host'
    },
    infrastructure: {
      defaultRegion: 'Auto (Best Performance)',
      defaultStorageClass: 'Balanced (Recommended)',
      autoScaling: 'Enabled',
      backupPolicy: 'Daily (7 days retention)'
    },
    notifications: {
      deploymentUpdates: true,
      nodeAlerts: true,
      billingInvoices: true,
      securityAlerts: true,
      productUpdates: false,
      marketing: false
    },
    integrations: {
      github: { connected: true, account: 'decentralized-host' },
      cloudflare: { connected: false },
      discord: { connected: true, channel: '#devops-alerts' },
      slack: { connected: false }
    },
    apiKeys: [
      { id: 'key-1', name: 'Production CI/CD Runner', prefix: 'dh_live_981a', createdAt: '2026-08-14', lastUsed: '5m ago' },
      { id: 'key-2', name: 'Node Operator Agent', prefix: 'dh_live_44f1', createdAt: '2026-09-01', lastUsed: '2s ago' }
    ],
    advanced: {
      betaFeatures: true,
      usageAnalytics: true,
      debugMode: false
    }
  };

  pendingActions: Map<string, ProposedAction> = new Map();

  // Helper method to compute dynamic compute telemetry
  getComputeSummary(): ComputeSummary {
    const totalNodes = this.nodes.length;
    const selfHostedNodes = this.nodes.filter(
      (n) => n.hardwareType === 'Self-Hosted Edge' || n.hardwareType === 'Homelab SBC'
    ).length;
    const datacenterNodes = totalNodes - selfHostedNodes;

    const totalCpuCores = this.nodes.reduce((acc, n) => acc + (n.cpuCores || 16), 0);
    const usedCpuCores = Number(
      this.nodes
        .reduce((acc, n) => acc + ((n.cpuCores || 16) * n.cpuPercent) / 100, 0)
        .toFixed(1)
    );
    const cpuUtilizationPercent = totalCpuCores > 0 ? Math.round((usedCpuCores / totalCpuCores) * 100) : 0;

    const totalMemoryGb = this.nodes.reduce((acc, n) => acc + (n.memoryTotalGb || 64), 0);
    const usedMemoryGb = Number(
      this.nodes.reduce((acc, n) => acc + (n.memoryUsedGb || 32), 0).toFixed(1)
    );
    const memoryUtilizationPercent = totalMemoryGb > 0 ? Math.round((usedMemoryGb / totalMemoryGb) * 100) : 0;

    const totalStorageTb = Number(
      (this.nodes.reduce((acc, n) => acc + (n.diskTotalGb || 2000), 0) / 1000).toFixed(1)
    );
    const usedStorageTb = Number(
      (this.nodes.reduce((acc, n) => acc + (n.diskUsedGb || 1000), 0) / 1000).toFixed(1)
    );
    const storageUtilizationPercent = totalStorageTb > 0 ? Math.round((usedStorageTb / totalStorageTb) * 100) : 0;

    const gpuNodes = this.nodes.filter((n) => n.gpuEquipped);
    const totalGpus = gpuNodes.reduce((acc, n) => acc + (n.gpuCount || 0), 0);
    const activeGpus = gpuNodes.filter((n) => n.status === 'Online').reduce((acc, n) => acc + (n.gpuCount || 0), 0);
    const totalGpuVramGb = gpuNodes.reduce((acc, n) => acc + (n.gpuMemoryTotalGb || 0), 0);

    const totalActiveWorkloads = this.nodes.reduce((acc, n) => acc + (n.workloadCount || 0), 0);
    const depinNetworksConnected = this.depinIntegrations.filter((d) => d.status === 'Active').length;
    const totalRewardsEarnedDH = Number(
      this.nodes.reduce((acc, n) => acc + (n.rewardsEarnedCredits || 0), 0).toFixed(1)
    );
    const averageUptimeScore = Number(
      (this.nodes.reduce((acc, n) => acc + (n.uptimeScore || 99), 0) / (totalNodes || 1)).toFixed(2)
    );

    return {
      totalNodes,
      selfHostedNodes,
      datacenterNodes,
      totalCpuCores,
      usedCpuCores,
      cpuUtilizationPercent,
      totalMemoryGb,
      usedMemoryGb,
      memoryUtilizationPercent,
      totalStorageTb,
      usedStorageTb,
      storageUtilizationPercent,
      totalGpus,
      activeGpus,
      totalGpuVramGb,
      totalActiveWorkloads,
      depinNetworksConnected,
      totalRewardsEarnedDH,
      averageUptimeScore
    };
  }
}

// Export both names for backwards compatibility
export const DemoPlatformStoreClass = DemoPlatformStore;
export const PlatformStore = DemoPlatformStore;
export const platformStore = new DemoPlatformStore();
