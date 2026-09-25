import React, { useState, useEffect, useMemo } from 'react';
import {
  Server,
  Activity,
  Cpu,
  HardDrive,
  Wifi,
  Globe,
  ShieldCheck,
  AlertTriangle,
  CheckCircle2,
  Clock,
  RefreshCw,
  Play,
  Pause,
  Filter,
  Search,
  Eye,
  Terminal,
  Zap,
  Radio,
  Layers,
  BarChart3,
  X,
  ArrowUpRight,
  TrendingUp,
  AlertCircle,
  Sliders,
  ChevronDown
} from 'lucide-react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ReferenceLine
} from 'recharts';

export interface NodeTelemetryMetricPoint {
  timestamp: string;
  cpu: number;
  memory: number;
  networkIn: number;
  networkOut: number;
  latency: number;
}

export interface DecentralizedNodeTelemetry {
  id: string;
  name: string;
  hostName: string;
  peerId: string;
  consensusRole: 'Validator' | 'Relay' | 'Edge Gateway' | 'Storage Replicator' | 'Merkle Witness';
  region: string;
  location: string;
  provider: string;
  ipAddress: string;
  health: 'Healthy' | 'Degraded' | 'Syncing' | 'Critical';
  uptimePercent: number;
  uptimeDays: number;
  lastHeartbeatSeconds: number;
  blockHeight: number;
  syncPercent: number;
  connectedPeers: number;
  maxPeers: number;
  attestationRate: number;
  latencyMs: number;
  temperatureC: number;
  resources: {
    cpuPercent: number;
    cpuCores: number;
    memoryUsedGb: number;
    memoryTotalGb: number;
    diskUsedTb: number;
    diskTotalTb: number;
    diskIops: number;
    bandwidthInMbps: number;
    bandwidthOutMbps: number;
  };
  metricsHistory: NodeTelemetryMetricPoint[];
  logs: { timestamp: string; level: 'info' | 'warn' | 'error'; message: string }[];
}

export const INITIAL_MOCK_NODES: DecentralizedNodeTelemetry[] = [
  {
    id: 'node-us-east-01',
    name: 'US East Core Validator',
    hostName: 'useast-core-01.mesh.net',
    peerId: '12D3KooWSD8f...9mKx1',
    consensusRole: 'Validator',
    region: 'North America',
    location: 'Ashburn, VA, USA',
    provider: 'Equinix Metal',
    ipAddress: '147.75.104.29',
    health: 'Healthy',
    uptimePercent: 99.99,
    uptimeDays: 142,
    lastHeartbeatSeconds: 1,
    blockHeight: 19842109,
    syncPercent: 100,
    connectedPeers: 48,
    maxPeers: 50,
    attestationRate: 99.98,
    latencyMs: 14,
    temperatureC: 43,
    resources: {
      cpuPercent: 38.4,
      cpuCores: 32,
      memoryUsedGb: 46.2,
      memoryTotalGb: 128,
      diskUsedTb: 2.1,
      diskTotalTb: 8.0,
      diskIops: 2450,
      bandwidthInMbps: 684.2,
      bandwidthOutMbps: 840.5
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 34, memory: 45, networkIn: 620, networkOut: 790, latency: 13 },
      { timestamp: '14:32', cpu: 41, memory: 46, networkIn: 710, networkOut: 820, latency: 15 },
      { timestamp: '14:34', cpu: 37, memory: 46, networkIn: 670, networkOut: 810, latency: 14 },
      { timestamp: '14:36', cpu: 39, memory: 46, networkIn: 690, networkOut: 850, latency: 14 },
      { timestamp: '14:38', cpu: 38, memory: 46, networkIn: 684, networkOut: 840, latency: 14 }
    ],
    logs: [
      { timestamp: '14:38:22', level: 'info', message: 'Attestation slot 829141 verified with 0ms delay.' },
      { timestamp: '14:37:05', level: 'info', message: 'Gossip mesh state synchronized with 48 peers.' },
      { timestamp: '14:35:10', level: 'info', message: 'Heartbeat ping broadcasted to gateway pool.' }
    ]
  },
  {
    id: 'node-eu-west-01',
    name: 'Frankfurt Storage Relay',
    hostName: 'eu-fra-storage-01.mesh.net',
    peerId: '12D3KooWLq2...a7Vz9',
    consensusRole: 'Storage Replicator',
    region: 'Europe',
    location: 'Frankfurt, Germany',
    provider: 'Hetzner Dedicated',
    ipAddress: '159.69.21.14',
    health: 'Healthy',
    uptimePercent: 99.98,
    uptimeDays: 98,
    lastHeartbeatSeconds: 2,
    blockHeight: 19842109,
    syncPercent: 100,
    connectedPeers: 42,
    maxPeers: 50,
    attestationRate: 99.94,
    latencyMs: 18,
    temperatureC: 47,
    resources: {
      cpuPercent: 54.2,
      cpuCores: 24,
      memoryUsedGb: 58.7,
      memoryTotalGb: 96,
      diskUsedTb: 5.6,
      diskTotalTb: 12.0,
      diskIops: 4210,
      bandwidthInMbps: 910.4,
      bandwidthOutMbps: 1120.8
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 48, memory: 56, networkIn: 880, networkOut: 1050, latency: 17 },
      { timestamp: '14:32', cpu: 52, memory: 57, networkIn: 940, networkOut: 1100, latency: 18 },
      { timestamp: '14:34', cpu: 58, memory: 59, networkIn: 990, networkOut: 1190, latency: 19 },
      { timestamp: '14:36', cpu: 55, memory: 58, networkIn: 920, networkOut: 1130, latency: 18 },
      { timestamp: '14:38', cpu: 54, memory: 58, networkIn: 910, networkOut: 1120, latency: 18 }
    ],
    logs: [
      { timestamp: '14:38:14', level: 'info', message: 'Replicated shard bafy2bz...8f1 across 5 EU nodes.' },
      { timestamp: '14:36:51', level: 'info', message: 'IPFS DHT block store prune finished: 1.2 GB freed.' }
    ]
  },
  {
    id: 'node-ap-se-01',
    name: 'Singapore Edge Gateway',
    hostName: 'sgp-edge-01.mesh.net',
    peerId: '12D3KooWRp9...k3Yd4',
    consensusRole: 'Edge Gateway',
    region: 'Asia Pacific',
    location: 'Singapore, Jurong East',
    provider: 'Equinix SG1',
    ipAddress: '103.115.194.8',
    health: 'Degraded',
    uptimePercent: 99.82,
    uptimeDays: 34,
    lastHeartbeatSeconds: 4,
    blockHeight: 19842106,
    syncPercent: 99.98,
    connectedPeers: 36,
    maxPeers: 50,
    attestationRate: 98.91,
    latencyMs: 38,
    temperatureC: 56,
    resources: {
      cpuPercent: 82.6,
      cpuCores: 16,
      memoryUsedGb: 54.1,
      memoryTotalGb: 64,
      diskUsedTb: 3.4,
      diskTotalTb: 4.0,
      diskIops: 3890,
      bandwidthInMbps: 1420.0,
      bandwidthOutMbps: 1680.5
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 74, memory: 50, networkIn: 1200, networkOut: 1400, latency: 32 },
      { timestamp: '14:32', cpu: 79, memory: 52, networkIn: 1350, networkOut: 1550, latency: 35 },
      { timestamp: '14:34', cpu: 86, memory: 55, networkIn: 1510, networkOut: 1720, latency: 42 },
      { timestamp: '14:36', cpu: 84, memory: 54, networkIn: 1460, networkOut: 1690, latency: 40 },
      { timestamp: '14:38', cpu: 82, memory: 54, networkIn: 1420, networkOut: 1680, latency: 38 }
    ],
    logs: [
      { timestamp: '14:38:09', level: 'warn', message: 'High ingress traffic surge from APAC edge ingress route.' },
      { timestamp: '14:35:28', level: 'warn', message: 'Memory pressure reached 84.5% threshold. Spilling buffer to zswap.' }
    ]
  },
  {
    id: 'node-us-west-02',
    name: 'Silicon Valley Relay',
    hostName: 'uswest-relay-02.mesh.net',
    peerId: '12D3KooW5Nt...2cV89',
    consensusRole: 'Relay',
    region: 'North America',
    location: 'San Jose, CA, USA',
    provider: 'Latitude.sh',
    ipAddress: '147.28.140.11',
    health: 'Healthy',
    uptimePercent: 99.96,
    uptimeDays: 81,
    lastHeartbeatSeconds: 1,
    blockHeight: 19842109,
    syncPercent: 100,
    connectedPeers: 49,
    maxPeers: 50,
    attestationRate: 99.97,
    latencyMs: 16,
    temperatureC: 41,
    resources: {
      cpuPercent: 29.8,
      cpuCores: 32,
      memoryUsedGb: 38.4,
      memoryTotalGb: 128,
      diskUsedTb: 1.8,
      diskTotalTb: 6.0,
      diskIops: 1820,
      bandwidthInMbps: 540.2,
      bandwidthOutMbps: 610.7
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 28, memory: 37, networkIn: 510, networkOut: 580, latency: 16 },
      { timestamp: '14:32', cpu: 31, memory: 38, networkIn: 550, networkOut: 620, latency: 17 },
      { timestamp: '14:34', cpu: 30, memory: 38, networkIn: 530, networkOut: 600, latency: 16 },
      { timestamp: '14:36', cpu: 29, memory: 38, networkIn: 525, networkOut: 595, latency: 16 },
      { timestamp: '14:38', cpu: 29, memory: 38, networkIn: 540, networkOut: 610, latency: 16 }
    ],
    logs: [
      { timestamp: '14:37:44', level: 'info', message: 'Relayed 1,420 cross-zone gossip packets to APAC relay.' },
      { timestamp: '14:34:12', level: 'info', message: 'Zero packet loss verified on BGP primary route.' }
    ]
  },
  {
    id: 'node-ap-south-01',
    name: 'Mumbai Merkle Witness',
    hostName: 'in-bom-witness-01.mesh.net',
    peerId: '12D3KooWK6u...9bRt2',
    consensusRole: 'Merkle Witness',
    region: 'Asia Pacific',
    location: 'Mumbai, India',
    provider: 'CtrlS Datacenters',
    ipAddress: '115.112.44.18',
    health: 'Healthy',
    uptimePercent: 99.95,
    uptimeDays: 62,
    lastHeartbeatSeconds: 2,
    blockHeight: 19842109,
    syncPercent: 100,
    connectedPeers: 45,
    maxPeers: 50,
    attestationRate: 99.92,
    latencyMs: 24,
    temperatureC: 48,
    resources: {
      cpuPercent: 44.1,
      cpuCores: 16,
      memoryUsedGb: 32.8,
      memoryTotalGb: 64,
      diskUsedTb: 2.9,
      diskTotalTb: 8.0,
      diskIops: 2890,
      bandwidthInMbps: 420.5,
      bandwidthOutMbps: 512.3
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 42, memory: 32, networkIn: 400, networkOut: 490, latency: 23 },
      { timestamp: '14:32', cpu: 45, memory: 33, networkIn: 430, networkOut: 520, latency: 25 },
      { timestamp: '14:34', cpu: 46, memory: 33, networkIn: 440, networkOut: 530, latency: 24 },
      { timestamp: '14:36', cpu: 43, memory: 32, networkIn: 410, networkOut: 505, latency: 24 },
      { timestamp: '14:38', cpu: 44, memory: 32, networkIn: 420, networkOut: 512, latency: 24 }
    ],
    logs: [
      { timestamp: '14:38:01', level: 'info', message: 'Merkle root hash 0x7e29a... committed to decentralized witness ledger.' },
      { timestamp: '14:31:19', level: 'info', message: 'State trie integrity check passed (0 orphaned nodes).' }
    ]
  },
  {
    id: 'node-sa-east-01',
    name: 'São Paulo Storage Node',
    hostName: 'sa-gru-storage-01.mesh.net',
    peerId: '12D3KooWM9k...4pQe7',
    consensusRole: 'Storage Replicator',
    region: 'South America',
    location: 'São Paulo, Brazil',
    provider: 'Ascenty DC',
    ipAddress: '177.189.92.5',
    health: 'Syncing',
    uptimePercent: 99.45,
    uptimeDays: 19,
    lastHeartbeatSeconds: 5,
    blockHeight: 19842095,
    syncPercent: 99.93,
    connectedPeers: 31,
    maxPeers: 50,
    attestationRate: 97.88,
    latencyMs: 62,
    temperatureC: 49,
    resources: {
      cpuPercent: 66.8,
      cpuCores: 16,
      memoryUsedGb: 44.5,
      memoryTotalGb: 64,
      diskUsedTb: 6.2,
      diskTotalTb: 10.0,
      diskIops: 3600,
      bandwidthInMbps: 380.1,
      bandwidthOutMbps: 490.4
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 60, memory: 42, networkIn: 340, networkOut: 450, latency: 60 },
      { timestamp: '14:32', cpu: 64, memory: 43, networkIn: 360, networkOut: 470, latency: 61 },
      { timestamp: '14:34', cpu: 69, memory: 45, networkIn: 400, networkOut: 510, latency: 64 },
      { timestamp: '14:36', cpu: 67, memory: 44, networkIn: 385, networkOut: 495, latency: 63 },
      { timestamp: '14:38', cpu: 66, memory: 44, networkIn: 380, networkOut: 490, latency: 62 }
    ],
    logs: [
      { timestamp: '14:38:19', level: 'info', message: 'Catching up block headers: 14 blocks remaining to tip.' },
      { timestamp: '14:33:41', level: 'warn', message: 'Transatlantic transit hop packet jitter observed (+12ms).' }
    ]
  },
  {
    id: 'node-eu-north-01',
    name: 'Stockholm Edge Validator',
    hostName: 'eu-arn-edge-01.mesh.net',
    peerId: '12D3KooWR1d...8kLn5',
    consensusRole: 'Validator',
    region: 'Europe',
    location: 'Stockholm, Sweden',
    provider: 'OVHcloud Eco-DC',
    ipAddress: '51.89.142.77',
    health: 'Healthy',
    uptimePercent: 99.99,
    uptimeDays: 168,
    lastHeartbeatSeconds: 1,
    blockHeight: 19842109,
    syncPercent: 100,
    connectedPeers: 47,
    maxPeers: 50,
    attestationRate: 99.99,
    latencyMs: 15,
    temperatureC: 37,
    resources: {
      cpuPercent: 32.5,
      cpuCores: 32,
      memoryUsedGb: 41.2,
      memoryTotalGb: 128,
      diskUsedTb: 3.1,
      diskTotalTb: 8.0,
      diskIops: 2180,
      bandwidthInMbps: 720.8,
      bandwidthOutMbps: 880.4
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 30, memory: 40, networkIn: 690, networkOut: 850, latency: 15 },
      { timestamp: '14:32', cpu: 33, memory: 41, networkIn: 710, networkOut: 870, latency: 15 },
      { timestamp: '14:34', cpu: 35, memory: 41, networkIn: 740, networkOut: 900, latency: 16 },
      { timestamp: '14:36', cpu: 33, memory: 41, networkIn: 715, networkOut: 875, latency: 15 },
      { timestamp: '14:38', cpu: 32, memory: 41, networkIn: 720, networkOut: 880, latency: 15 }
    ],
    logs: [
      { timestamp: '14:38:00', level: 'info', message: 'Green-power certified hydro DC node operating at 37°C optimal.' },
      { timestamp: '14:32:10', level: 'info', message: 'Proposer duty completed for slot 829139. Reward claimed.' }
    ]
  },
  {
    id: 'node-me-central-01',
    name: 'Dubai Edge Gateway',
    hostName: 'dxb-edge-01.mesh.net',
    peerId: '12D3KooWT8x...3nMb1',
    consensusRole: 'Edge Gateway',
    region: 'Middle East',
    location: 'Dubai, UAE',
    provider: 'Khazna DC',
    ipAddress: '86.96.201.32',
    health: 'Healthy',
    uptimePercent: 99.92,
    uptimeDays: 45,
    lastHeartbeatSeconds: 2,
    blockHeight: 19842109,
    syncPercent: 100,
    connectedPeers: 39,
    maxPeers: 50,
    attestationRate: 99.88,
    latencyMs: 28,
    temperatureC: 45,
    resources: {
      cpuPercent: 49.3,
      cpuCores: 24,
      memoryUsedGb: 39.0,
      memoryTotalGb: 64,
      diskUsedTb: 2.4,
      diskTotalTb: 6.0,
      diskIops: 2750,
      bandwidthInMbps: 610.0,
      bandwidthOutMbps: 740.2
    },
    metricsHistory: [
      { timestamp: '14:30', cpu: 46, memory: 38, networkIn: 580, networkOut: 710, latency: 27 },
      { timestamp: '14:32', cpu: 51, memory: 39, networkIn: 630, networkOut: 760, latency: 29 },
      { timestamp: '14:34', cpu: 50, memory: 39, networkIn: 620, networkOut: 750, latency: 28 },
      { timestamp: '14:36', cpu: 48, memory: 39, networkIn: 605, networkOut: 735, latency: 28 },
      { timestamp: '14:38', cpu: 49, memory: 39, networkIn: 610, networkOut: 740, latency: 28 }
    ],
    logs: [
      { timestamp: '14:37:12', level: 'info', message: 'Routing MENA edge clients with sub-30ms latency.' },
      { timestamp: '14:30:45', level: 'info', message: 'TLS session cache hit ratio: 98.4%.' }
    ]
  }
];

// Custom Recharts Dark Tooltip Component
const CustomTelemetryTooltip: React.FC<any> = ({ active, payload, label }) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-[#070B14]/95 backdrop-blur-md border border-slate-700 rounded-xl p-3 shadow-2xl font-mono text-xs z-50">
        <div className="text-slate-400 font-bold mb-2 pb-1 border-b border-slate-800 flex items-center justify-between gap-4">
          <span className="flex items-center gap-1.5 text-slate-300">
            <Clock className="w-3 h-3 text-cyan-400" />
            {label}
          </span>
          <span className="flex items-center gap-1 text-[10px] text-emerald-400">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            LIVE
          </span>
        </div>
        <div className="space-y-1.5">
          {payload.map((entry: any, i: number) => {
            const isLatency = entry.name?.toLowerCase().includes('latency') || entry.dataKey === 'latency';
            return (
              <div key={i} className="flex items-center justify-between gap-5">
                <span className="flex items-center gap-1.5 font-medium" style={{ color: entry.stroke || entry.color }}>
                  <span
                    className="w-2 h-2 rounded-full"
                    style={{ backgroundColor: entry.stroke || entry.color }}
                  />
                  {entry.name}:
                </span>
                <span className="font-bold text-white">
                  {entry.value}
                  {isLatency ? ' ms' : '%'}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    );
  }
  return null;
};

// Mini Tooltip for Card Sparkline
const CustomMiniTooltip: React.FC<any> = ({ active, payload, label }) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-[#070B14]/95 border border-slate-700/80 rounded-lg px-2 py-1 shadow-lg font-mono text-[10px] text-white">
        <div className="text-slate-400 text-[9px]">{label}</div>
        {payload.map((p: any, i: number) => (
          <div key={i} className="flex items-center gap-1" style={{ color: p.stroke }}>
            <span>{p.dataKey === 'cpu' ? 'CPU' : 'Mem'}:</span>
            <span className="font-bold">{p.value}%</span>
          </div>
        ))}
      </div>
    );
  }
  return null;
};

export const NodeTelemetryDashboard: React.FC = () => {
  const [nodes, setNodes] = useState<DecentralizedNodeTelemetry[]>(INITIAL_MOCK_NODES);
  const [isLive, setIsLive] = useState(true);
  const [refreshIntervalSec, setRefreshIntervalSec] = useState(3);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedRegion, setSelectedRegion] = useState<string>('All');
  const [selectedHealth, setSelectedHealth] = useState<string>('All');
  const [viewMode, setViewMode] = useState<'cards' | 'table'>('cards');
  const [inspectNode, setInspectNode] = useState<DecentralizedNodeTelemetry | null>(null);
  const [timeRange, setTimeRange] = useState<'5m' | '1h' | '24h' | '7d'>('5m');
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [actionNotice, setActionNotice] = useState<string | null>(null);

  // Recharts interactive controls state
  const [selectedChartNodeId, setSelectedChartNodeId] = useState<string>('all');
  const [showCpuLine, setShowCpuLine] = useState<boolean>(true);
  const [showMemoryLine, setShowMemoryLine] = useState<boolean>(true);
  const [showLatencyLine, setShowLatencyLine] = useState<boolean>(false);

  // Real-time telemetry simulation engine
  useEffect(() => {
    if (!isLive) return;

    const timer = setInterval(() => {
      setNodes((prevNodes) => {
        const now = new Date();
        const timeStr = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`;

        return prevNodes.map((node) => {
          // Subtle natural fluctuation
          const cpuDelta = (Math.random() - 0.48) * 3.5;
          const newCpu = Math.max(12, Math.min(96, Number((node.resources.cpuPercent + cpuDelta).toFixed(1))));

          const memDelta = (Math.random() - 0.49) * 0.4;
          const newMemGb = Math.max(8, Math.min(node.resources.memoryTotalGb * 0.95, Number((node.resources.memoryUsedGb + memDelta).toFixed(1))));

          const bwInDelta = (Math.random() - 0.48) * 25;
          const newBwIn = Math.max(80, Number((node.resources.bandwidthInMbps + bwInDelta).toFixed(1)));

          const bwOutDelta = (Math.random() - 0.48) * 30;
          const newBwOut = Math.max(90, Number((node.resources.bandwidthOutMbps + bwOutDelta).toFixed(1)));

          const latDelta = (Math.random() - 0.5) * 1.5;
          const newLat = Math.max(8, Math.round(node.latencyMs + latDelta));

          // Increment block height occasionally
          const newBlock = Math.random() > 0.6 ? node.blockHeight + 1 : node.blockHeight;

          // New history slice (keep last 12 points for fluid Recharts trends)
          const newHistory = [
            ...node.metricsHistory.slice(-11),
            {
              timestamp: timeStr,
              cpu: newCpu,
              memory: Math.round((newMemGb / node.resources.memoryTotalGb) * 100),
              networkIn: Math.round(newBwIn),
              networkOut: Math.round(newBwOut),
              latency: newLat
            }
          ];

          return {
            ...node,
            lastHeartbeatSeconds: 1,
            blockHeight: newBlock,
            latencyMs: newLat,
            resources: {
              ...node.resources,
              cpuPercent: newCpu,
              memoryUsedGb: newMemGb,
              bandwidthInMbps: newBwIn,
              bandwidthOutMbps: newBwOut
            },
            metricsHistory: newHistory
          };
        });
      });
    }, refreshIntervalSec * 1000);

    return () => clearInterval(timer);
  }, [isLive, refreshIntervalSec]);

  // Keep inspectNode synchronized with live changes
  useEffect(() => {
    if (inspectNode) {
      const updated = nodes.find((n) => n.id === inspectNode.id);
      if (updated) setInspectNode(updated);
    }
  }, [nodes]);

  const handleManualRefresh = () => {
    setIsRefreshing(true);
    setTimeout(() => {
      setIsRefreshing(false);
      setActionNotice('Cluster telemetry state refreshed across all edge relays.');
      setTimeout(() => setActionNotice(null), 3500);
    }, 450);
  };

  const handlePingNode = (nodeName: string) => {
    setActionNotice(`ICMP & Gossip round-trip ping sent to ${nodeName}. Response: 14.2ms (OK).`);
    setTimeout(() => setActionNotice(null), 4000);
  };

  const handleSimulateLoad = (nodeId: string) => {
    setNodes((prev) =>
      prev.map((n) => {
        if (n.id === nodeId) {
          return {
            ...n,
            resources: {
              ...n.resources,
              cpuPercent: Math.min(94, n.resources.cpuPercent + 25)
            },
            logs: [
              {
                timestamp: new Date().toLocaleTimeString(),
                level: 'warn',
                message: 'Synthetic load benchmark executed (+25% CPU stress test).'
              },
              ...n.logs
            ]
          };
        }
        return n;
      })
    );
    setActionNotice(`Synthetic load test triggered on ${nodeId}. Metrics updated in real-time.`);
    setTimeout(() => setActionNotice(null), 3500);
  };

  // Aggregated Cluster Metrics
  const clusterMetrics = useMemo(() => {
    const totalNodes = nodes.length;
    const healthyNodes = nodes.filter((n) => n.health === 'Healthy').length;
    const degradedNodes = nodes.filter((n) => n.health === 'Degraded').length;
    const syncingNodes = nodes.filter((n) => n.health === 'Syncing').length;
    const criticalNodes = nodes.filter((n) => n.health === 'Critical').length;

    const avgCpu = Math.round(nodes.reduce((acc, n) => acc + n.resources.cpuPercent, 0) / totalNodes);
    const totalMemUsedGb = Math.round(nodes.reduce((acc, n) => acc + n.resources.memoryUsedGb, 0));
    const totalMemCapacityGb = Math.round(nodes.reduce((acc, n) => acc + n.resources.memoryTotalGb, 0));
    const avgMemPercent = Math.round((totalMemUsedGb / totalMemCapacityGb) * 100);

    const totalDiskUsedTb = Number(nodes.reduce((acc, n) => acc + n.resources.diskUsedTb, 0).toFixed(1));
    const totalDiskCapacityTb = Number(nodes.reduce((acc, n) => acc + n.resources.diskTotalTb, 0).toFixed(1));

    const totalBwInGbps = Number((nodes.reduce((acc, n) => acc + n.resources.bandwidthInMbps, 0) / 1000).toFixed(2));
    const totalBwOutGbps = Number((nodes.reduce((acc, n) => acc + n.resources.bandwidthOutMbps, 0) / 1000).toFixed(2));

    const avgUptime = Number((nodes.reduce((acc, n) => acc + n.uptimePercent, 0) / totalNodes).toFixed(2));
    const avgAttestation = Number((nodes.reduce((acc, n) => acc + n.attestationRate, 0) / totalNodes).toFixed(2));
    const avgLatency = Math.round(nodes.reduce((acc, n) => acc + n.latencyMs, 0) / totalNodes);

    return {
      totalNodes,
      healthyNodes,
      degradedNodes,
      syncingNodes,
      criticalNodes,
      avgCpu,
      totalMemUsedGb,
      totalMemCapacityGb,
      avgMemPercent,
      totalDiskUsedTb,
      totalDiskCapacityTb,
      totalBwInGbps,
      totalBwOutGbps,
      avgUptime,
      avgAttestation,
      avgLatency
    };
  }, [nodes]);

  // Cluster aggregate telemetry history for Recharts
  const clusterAggregateHistory = useMemo(() => {
    if (!nodes.length) return [];
    const pointsCount = nodes[0].metricsHistory.length;
    const result: NodeTelemetryMetricPoint[] = [];

    for (let i = 0; i < pointsCount; i++) {
      const timestamp = nodes[0].metricsHistory[i]?.timestamp || '';
      let sumCpu = 0;
      let sumMem = 0;
      let sumNetIn = 0;
      let sumNetOut = 0;
      let sumLat = 0;

      nodes.forEach((n) => {
        const pt = n.metricsHistory[i];
        if (pt) {
          sumCpu += pt.cpu;
          sumMem += pt.memory;
          sumNetIn += pt.networkIn;
          sumNetOut += pt.networkOut;
          sumLat += pt.latency;
        }
      });

      result.push({
        timestamp,
        cpu: Number((sumCpu / nodes.length).toFixed(1)),
        memory: Number((sumMem / nodes.length).toFixed(1)),
        networkIn: Math.round(sumNetIn / nodes.length),
        networkOut: Math.round(sumNetOut / nodes.length),
        latency: Math.round(sumLat / nodes.length)
      });
    }

    return result;
  }, [nodes]);

  // Active chart data based on selected node or cluster aggregate
  const activeChartData = useMemo(() => {
    if (selectedChartNodeId === 'all') {
      return clusterAggregateHistory;
    }
    const target = nodes.find((n) => n.id === selectedChartNodeId);
    return target ? target.metricsHistory : clusterAggregateHistory;
  }, [selectedChartNodeId, clusterAggregateHistory, nodes]);

  // Active chart title and focus node metadata
  const activeChartMetadata = useMemo(() => {
    if (selectedChartNodeId === 'all') {
      return {
        title: 'Cluster Aggregate Telemetry',
        subtitle: `Averaged across all ${nodes.length} decentralized nodes in real time`,
        currentCpu: clusterMetrics.avgCpu,
        currentMem: clusterMetrics.avgMemPercent,
        currentLatency: clusterMetrics.avgLatency
      };
    }
    const node = nodes.find((n) => n.id === selectedChartNodeId);
    if (!node) {
      return {
        title: 'Cluster Aggregate Telemetry',
        subtitle: 'Averaged across all decentralized nodes in real time',
        currentCpu: clusterMetrics.avgCpu,
        currentMem: clusterMetrics.avgMemPercent,
        currentLatency: clusterMetrics.avgLatency
      };
    }
    return {
      title: `${node.name} (${node.consensusRole})`,
      subtitle: `${node.location} · ${node.provider} · IP ${node.ipAddress}`,
      currentCpu: node.resources.cpuPercent,
      currentMem: Math.round((node.resources.memoryUsedGb / node.resources.memoryTotalGb) * 100),
      currentLatency: node.latencyMs
    };
  }, [selectedChartNodeId, nodes, clusterMetrics]);

  // Unique filter regions
  const availableRegions = useMemo(() => {
    const set = new Set(nodes.map((n) => n.region));
    return ['All', ...Array.from(set)];
  }, [nodes]);

  // Filtered nodes
  const filteredNodes = useMemo(() => {
    return nodes.filter((n) => {
      const matchesSearch =
        n.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        n.location.toLowerCase().includes(searchTerm.toLowerCase()) ||
        n.provider.toLowerCase().includes(searchTerm.toLowerCase()) ||
        n.ipAddress.includes(searchTerm) ||
        n.consensusRole.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesRegion = selectedRegion === 'All' || n.region === selectedRegion;
      const matchesHealth = selectedHealth === 'All' || n.health === selectedHealth;

      return matchesSearch && matchesRegion && matchesHealth;
    });
  }, [nodes, searchTerm, selectedRegion, selectedHealth]);

  const getHealthBadge = (health: DecentralizedNodeTelemetry['health']) => {
    switch (health) {
      case 'Healthy':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            Healthy
          </span>
        );
      case 'Degraded':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
            Degraded
          </span>
        );
      case 'Syncing':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-spin" />
            Syncing
          </span>
        );
      case 'Critical':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
            <span className="w-1.5 h-1.5 rounded-full bg-rose-400" />
            Critical
          </span>
        );
    }
  };

  return (
    <div className="space-y-6">
      {/* Action Notification Toast */}
      {actionNotice && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center gap-2.5 px-4 py-3 bg-[#0B132B] border border-cyan-500/40 text-cyan-200 text-xs rounded-xl shadow-2xl animate-in fade-in slide-in-from-bottom duration-200">
          <Zap className="w-4 h-4 text-cyan-400 flex-shrink-0 animate-pulse" />
          <span>{actionNotice}</span>
          <button onClick={() => setActionNotice(null)} className="ml-2 text-slate-400 hover:text-white">
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}

      {/* Main Header with Real-Time Controls */}
      <div className="flex flex-col xl:flex-row items-start xl:items-center justify-between gap-4 bg-[#0A0F1D] border border-slate-800/80 rounded-2xl p-5 shadow-xl">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-cyan-400 uppercase tracking-widest">
            <Radio className="w-3.5 h-3.5 animate-pulse" />
            <span>Telemetry Control Plane</span>
            <span className="text-slate-600">·</span>
            <span className="text-slate-400 font-sans normal-case">Decentralized Mesh v2.4</span>
          </div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-3 mt-1">
            <span>Node Telemetry Dashboard</span>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <span className={`w-2 h-2 rounded-full ${isLive ? 'bg-emerald-400 animate-ping' : 'bg-slate-500'}`} />
              {isLive ? `LIVE (${refreshIntervalSec}s)` : 'PAUSED'}
            </span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Real-time peer consensus health, SLA uptime, and hardware resource utilization across distributed nodes.
          </p>
        </div>

        {/* Action Controls */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Time range selector */}
          <div className="flex items-center bg-[#070B14] rounded-xl border border-slate-800 p-0.5">
            {(['5m', '1h', '24h', '7d'] as const).map((range) => (
              <button
                key={range}
                onClick={() => setTimeRange(range)}
                className={`px-2.5 py-1 text-xs font-mono rounded-lg transition-colors ${
                  timeRange === range
                    ? 'bg-blue-600 text-white font-semibold shadow'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                {range}
              </button>
            ))}
          </div>

          {/* Polling Interval Selector */}
          <select
            value={refreshIntervalSec}
            onChange={(e) => setRefreshIntervalSec(Number(e.target.value))}
            className="bg-[#070B14] border border-slate-800 rounded-xl px-2.5 py-1.5 text-xs text-slate-300 font-mono focus:outline-none focus:border-cyan-500 cursor-pointer"
            title="Telemetry polling interval"
          >
            <option value={1}>1s Poll</option>
            <option value={3}>3s Poll</option>
            <option value={5}>5s Poll</option>
          </select>

          {/* Live Stream Toggle */}
          <button
            onClick={() => setIsLive(!isLive)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold transition-all border ${
              isLive
                ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/20'
                : 'bg-amber-500/10 border-amber-500/30 text-amber-400 hover:bg-amber-500/20'
            }`}
          >
            {isLive ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}
            <span>{isLive ? 'Pause Stream' : 'Resume Live'}</span>
          </button>

          {/* Manual Refresh */}
          <button
            onClick={handleManualRefresh}
            disabled={isRefreshing}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-[#0D1527] border border-slate-700 hover:border-slate-600 text-slate-200 transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-cyan-400 ${isRefreshing ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Cluster Health & SLA Overview Strip */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3.5">
        {/* Total & Healthy Nodes */}
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-4 flex flex-col justify-between">
          <div className="flex items-center justify-between text-slate-400 text-xs">
            <span className="flex items-center gap-1.5">
              <Server className="w-3.5 h-3.5 text-cyan-400" />
              Mesh Nodes
            </span>
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
          </div>
          <div className="mt-2">
            <div className="text-2xl font-bold text-white font-mono">{clusterMetrics.totalNodes}</div>
            <div className="text-[11px] text-emerald-400 font-mono mt-0.5">
              {clusterMetrics.healthyNodes} Healthy · {clusterMetrics.degradedNodes} Degraded
            </div>
          </div>
        </div>

        {/* SLA Cluster Uptime */}
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-4 flex flex-col justify-between">
          <div className="flex items-center justify-between text-slate-400 text-xs">
            <span className="flex items-center gap-1.5">
              <Clock className="w-3.5 h-3.5 text-emerald-400" />
              SLA Uptime
            </span>
            <span className="text-[10px] font-mono text-emerald-400 font-bold">30d Target</span>
          </div>
          <div className="mt-2">
            <div className="text-2xl font-bold text-white font-mono">{clusterMetrics.avgUptime}%</div>
            <div className="text-[11px] text-slate-400 font-mono mt-0.5">Zero critical outages</div>
          </div>
        </div>

        {/* Attestation Rate */}
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-4 flex flex-col justify-between">
          <div className="flex items-center justify-between text-slate-400 text-xs">
            <span className="flex items-center gap-1.5">
              <ShieldCheck className="w-3.5 h-3.5 text-blue-400" />
              Attestation
            </span>
            <span className="text-[10px] font-mono text-blue-400">Consensus</span>
          </div>
          <div className="mt-2">
            <div className="text-2xl font-bold text-white font-mono">{clusterMetrics.avgAttestation}%</div>
            <div className="text-[11px] text-blue-400 font-mono mt-0.5">Active epoch #829</div>
          </div>
        </div>

        {/* Cluster CPU Load */}
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-4 flex flex-col justify-between">
          <div className="flex items-center justify-between text-slate-400 text-xs">
            <span className="flex items-center gap-1.5">
              <Cpu className="w-3.5 h-3.5 text-purple-400" />
              Cluster CPU
            </span>
            <span className="text-[10px] font-mono text-slate-400">Avg</span>
          </div>
          <div className="mt-2">
            <div className="text-2xl font-bold text-white font-mono">{clusterMetrics.avgCpu}%</div>
            <div className="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden mt-1.5">
              <div
                className="bg-gradient-to-r from-purple-500 to-indigo-500 h-full rounded-full transition-all duration-500"
                style={{ width: `${clusterMetrics.avgCpu}%` }}
              />
            </div>
          </div>
        </div>

        {/* Cluster Memory Load */}
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-4 flex flex-col justify-between">
          <div className="flex items-center justify-between text-slate-400 text-xs">
            <span className="flex items-center gap-1.5">
              <Layers className="w-3.5 h-3.5 text-teal-400" />
              Memory Pool
            </span>
            <span className="text-[10px] font-mono text-slate-400">{clusterMetrics.avgMemPercent}%</span>
          </div>
          <div className="mt-2">
            <div className="text-xl font-bold text-white font-mono truncate">
              {clusterMetrics.totalMemUsedGb} <span className="text-xs font-normal text-slate-400">/ {clusterMetrics.totalMemCapacityGb} GB</span>
            </div>
            <div className="w-full bg-slate-800 h-1.5 rounded-full overflow-hidden mt-1.5">
              <div
                className="bg-gradient-to-r from-teal-500 to-cyan-400 h-full rounded-full transition-all duration-500"
                style={{ width: `${clusterMetrics.avgMemPercent}%` }}
              />
            </div>
          </div>
        </div>

        {/* Global Mesh Throughput */}
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-4 flex flex-col justify-between">
          <div className="flex items-center justify-between text-slate-400 text-xs">
            <span className="flex items-center gap-1.5">
              <Activity className="w-3.5 h-3.5 text-amber-400" />
              Throughput
            </span>
            <span className="text-[10px] font-mono text-amber-400">{clusterMetrics.avgLatency}ms</span>
          </div>
          <div className="mt-2">
            <div className="text-xl font-bold text-white font-mono">
              {(clusterMetrics.totalBwInGbps + clusterMetrics.totalBwOutGbps).toFixed(2)}{' '}
              <span className="text-xs font-normal text-slate-400">Gbps</span>
            </div>
            <div className="text-[10px] text-slate-400 font-mono mt-0.5 flex items-center justify-between">
              <span>↓ {clusterMetrics.totalBwInGbps} Gbps</span>
              <span>↑ {clusterMetrics.totalBwOutGbps} Gbps</span>
            </div>
          </div>
        </div>
      </div>

      {/* Real-time Recharts Line Chart Visualization Panel */}
      <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl p-5 shadow-xl space-y-4">
        {/* Chart Header with Interactive Selectors & Toggles */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 border-b border-slate-800/80 pb-4">
          <div>
            <div className="flex items-center gap-2">
              <BarChart3 className="w-4 h-4 text-cyan-400" />
              <h2 className="text-base font-bold text-white tracking-tight">
                Real-Time Resource Telemetry Trends
              </h2>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                recharts engine
              </span>
            </div>
            <p className="text-xs text-slate-400 font-mono mt-1">
              {activeChartMetadata.title} — {activeChartMetadata.subtitle}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {/* Node Focus Dropdown */}
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-slate-500 font-mono">Focus:</span>
              <select
                value={selectedChartNodeId}
                onChange={(e) => setSelectedChartNodeId(e.target.value)}
                className="bg-[#070B14] border border-slate-800 rounded-xl px-3 py-1.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-cyan-500 cursor-pointer"
              >
                <option value="all">⚡ Cluster Aggregate (All 8 Nodes)</option>
                <optgroup label="Individual Decentralized Nodes">
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name} ({node.region})
                    </option>
                  ))}
                </optgroup>
              </select>
            </div>

            {/* Metric Line Toggles */}
            <div className="flex items-center gap-1.5 bg-[#070B14] p-1 rounded-xl border border-slate-800">
              <button
                onClick={() => setShowCpuLine(!showCpuLine)}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-mono transition-all ${
                  showCpuLine
                    ? 'bg-purple-600/30 text-purple-300 border border-purple-500/40 font-semibold'
                    : 'text-slate-500 hover:text-slate-300'
                }`}
                title="Toggle CPU Trend Line"
              >
                <span className="w-2 h-2 rounded-full bg-purple-400" />
                <span>CPU ({activeChartMetadata.currentCpu}%)</span>
              </button>

              <button
                onClick={() => setShowMemoryLine(!showMemoryLine)}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-mono transition-all ${
                  showMemoryLine
                    ? 'bg-cyan-600/30 text-cyan-300 border border-cyan-500/40 font-semibold'
                    : 'text-slate-500 hover:text-slate-300'
                }`}
                title="Toggle Memory Trend Line"
              >
                <span className="w-2 h-2 rounded-full bg-cyan-400" />
                <span>RAM ({activeChartMetadata.currentMem}%)</span>
              </button>

              <button
                onClick={() => setShowLatencyLine(!showLatencyLine)}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-mono transition-all ${
                  showLatencyLine
                    ? 'bg-emerald-600/30 text-emerald-300 border border-emerald-500/40 font-semibold'
                    : 'text-slate-500 hover:text-slate-300'
                }`}
                title="Toggle Latency Trend Line"
              >
                <span className="w-2 h-2 rounded-full bg-emerald-400" />
                <span>RTT ({activeChartMetadata.currentLatency}ms)</span>
              </button>
            </div>
          </div>
        </div>

        {/* Live Recharts LineChart */}
        <div className="h-64 w-full pt-1">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={activeChartData} margin={{ top: 10, right: 15, left: -15, bottom: 0 }}>
              <CartesianGrid stroke="#1E293B" strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="timestamp"
                stroke="#64748B"
                tick={{ fontSize: 11, fontFamily: 'monospace' }}
                tickLine={false}
              />
              <YAxis
                domain={[0, 100]}
                stroke="#64748B"
                tick={{ fontSize: 11, fontFamily: 'monospace' }}
                tickLine={false}
                unit="%"
              />
              <Tooltip content={<CustomTelemetryTooltip />} />
              <ReferenceLine
                y={80}
                stroke="#EF4444"
                strokeDasharray="4 4"
                label={{
                  value: 'Warning Threshold 80%',
                  fill: '#EF4444',
                  fontSize: 10,
                  position: 'insideTopRight'
                }}
              />
              {showCpuLine && (
                <Line
                  type="monotone"
                  dataKey="cpu"
                  name="CPU Usage"
                  stroke="#A855F7"
                  strokeWidth={2.5}
                  dot={{ r: 3, fill: '#A855F7' }}
                  activeDot={{ r: 6, fill: '#FFFFFF', stroke: '#A855F7', strokeWidth: 2 }}
                  isAnimationActive={false}
                />
              )}
              {showMemoryLine && (
                <Line
                  type="monotone"
                  dataKey="memory"
                  name="Memory Usage"
                  stroke="#06B6D4"
                  strokeWidth={2.5}
                  dot={{ r: 3, fill: '#06B6D4' }}
                  activeDot={{ r: 6, fill: '#FFFFFF', stroke: '#06B6D4', strokeWidth: 2 }}
                  isAnimationActive={false}
                />
              )}
              {showLatencyLine && (
                <Line
                  type="monotone"
                  dataKey="latency"
                  name="Round-Trip Latency"
                  stroke="#10B981"
                  strokeWidth={2}
                  dot={{ r: 2.5, fill: '#10B981' }}
                  activeDot={{ r: 5, fill: '#FFFFFF', stroke: '#10B981', strokeWidth: 2 }}
                  isAnimationActive={false}
                />
              )}
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Chart Legend & SLA Guardrail Status */}
        <div className="flex flex-wrap items-center justify-between text-xs font-mono text-slate-400 pt-3 border-t border-slate-800/80 gap-3">
          <div className="flex items-center gap-5">
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-0.5 bg-purple-400" />
              <span>CPU (% Load)</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-0.5 bg-cyan-400" />
              <span>Memory (% Allocation)</span>
            </span>
            {showLatencyLine && (
              <span className="flex items-center gap-1.5">
                <span className="w-2.5 h-0.5 bg-emerald-400" />
                <span>Round-Trip Latency (ms)</span>
              </span>
            )}
            <span className="flex items-center gap-1.5 text-rose-400">
              <span className="w-2.5 h-0.5 bg-rose-500 border-dashed" />
              <span>80% Saturation Ceiling</span>
            </span>
          </div>

          <div className="flex items-center gap-2 text-slate-500 text-[11px]">
            <span>Continuous window: last {activeChartData.length} checkpoints</span>
            <span>·</span>
            <span className="text-cyan-400">Auto-scaling balanced</span>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-3 bg-[#0A0F1D] border border-slate-800/80 rounded-2xl p-3">
        <div className="flex flex-1 items-center gap-3 w-full sm:w-auto">
          {/* Search box */}
          <div className="relative flex-1 max-w-sm">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
            <input
              type="text"
              placeholder="Filter by node name, IP, region, role..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-[#070B14] border border-slate-800 rounded-xl pl-9 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500"
            />
            {searchTerm && (
              <button
                onClick={() => setSearchTerm('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-500 hover:text-white"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>

          {/* Region Filter */}
          <div className="flex items-center gap-1.5">
            <Filter className="w-3.5 h-3.5 text-slate-500" />
            <select
              value={selectedRegion}
              onChange={(e) => setSelectedRegion(e.target.value)}
              className="bg-[#070B14] border border-slate-800 rounded-xl px-2.5 py-1.5 text-xs text-slate-300 font-mono focus:outline-none focus:border-cyan-500 cursor-pointer"
            >
              {availableRegions.map((reg) => (
                <option key={reg} value={reg}>
                  Region: {reg}
                </option>
              ))}
            </select>
          </div>

          {/* Health Filter */}
          <select
            value={selectedHealth}
            onChange={(e) => setSelectedHealth(e.target.value)}
            className="bg-[#070B14] border border-slate-800 rounded-xl px-2.5 py-1.5 text-xs text-slate-300 font-mono focus:outline-none focus:border-cyan-500 cursor-pointer"
          >
            <option value="All">All Health Statuses</option>
            <option value="Healthy">Healthy Only</option>
            <option value="Degraded">Degraded Only</option>
            <option value="Syncing">Syncing Only</option>
            <option value="Critical">Critical Only</option>
          </select>
        </div>

        {/* View Mode Toggle */}
        <div className="flex items-center gap-1 bg-[#070B14] border border-slate-800 rounded-xl p-1">
          <button
            onClick={() => setViewMode('cards')}
            className={`px-3 py-1 text-xs rounded-lg font-medium transition-colors ${
              viewMode === 'cards' ? 'bg-blue-600 text-white font-semibold' : 'text-slate-400 hover:text-white'
            }`}
          >
            Cards View
          </button>
          <button
            onClick={() => setViewMode('table')}
            className={`px-3 py-1 text-xs rounded-lg font-medium transition-colors ${
              viewMode === 'table' ? 'bg-blue-600 text-white font-semibold' : 'text-slate-400 hover:text-white'
            }`}
          >
            Telemetry Table
          </button>
        </div>
      </div>

      {/* Main Content: Card View or Table View */}
      {viewMode === 'cards' ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {filteredNodes.map((node) => {
            return (
              <div
                key={node.id}
                className="bg-[#0A0F1D] border border-slate-800/90 hover:border-slate-700 rounded-2xl p-5 shadow-lg flex flex-col justify-between transition-all hover:shadow-cyan-950/20 group"
              >
                <div>
                  {/* Top Bar of Card */}
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-[#0D1527] border border-slate-800 flex items-center justify-center text-cyan-400 group-hover:border-cyan-500/40 transition-colors">
                        <Server className="w-5 h-5" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-bold text-sm text-white">{node.name}</h3>
                        </div>
                        <div className="text-[11px] text-slate-400 font-mono flex items-center gap-1.5 mt-0.5">
                          <Globe className="w-3 h-3 text-slate-500" />
                          <span>{node.location}</span>
                          <span className="text-slate-600">·</span>
                          <span className="text-cyan-400">{node.provider}</span>
                        </div>
                      </div>
                    </div>
                    {getHealthBadge(node.health)}
                  </div>

                  {/* Consensus Role & Uptime stats */}
                  <div className="flex items-center justify-between text-xs font-mono mt-4 pt-3 border-t border-slate-800/80">
                    <div className="flex items-center gap-1.5">
                      <span className="text-[10px] uppercase tracking-wider text-slate-500">Role:</span>
                      <span className="text-slate-200 font-semibold px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-[10px]">
                        {node.consensusRole}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-slate-400">
                      <span>Uptime:</span>
                      <span className="text-emerald-400 font-bold">{node.uptimePercent}%</span>
                      <span className="text-slate-500 text-[10px]">({node.uptimeDays}d)</span>
                    </div>
                  </div>

                  {/* Resource Gauges (CPU, Memory, Disk, Bandwidth) */}
                  <div className="space-y-3 mt-4 bg-[#070B14] p-3.5 rounded-xl border border-slate-800/80">
                    {/* CPU Utilization */}
                    <div>
                      <div className="flex justify-between text-xs font-mono mb-1">
                        <span className="text-slate-400 flex items-center gap-1.5">
                          <Cpu className="w-3 h-3 text-purple-400" />
                          CPU Usage
                        </span>
                        <span className="text-white font-semibold">{node.resources.cpuPercent}%</span>
                      </div>
                      <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full transition-all duration-500 ${
                            node.resources.cpuPercent > 80
                              ? 'bg-rose-500'
                              : node.resources.cpuPercent > 60
                              ? 'bg-amber-400'
                              : 'bg-gradient-to-r from-purple-500 to-indigo-500'
                          }`}
                          style={{ width: `${node.resources.cpuPercent}%` }}
                        />
                      </div>
                    </div>

                    {/* Memory Utilization */}
                    <div>
                      <div className="flex justify-between text-xs font-mono mb-1">
                        <span className="text-slate-400 flex items-center gap-1.5">
                          <Layers className="w-3 h-3 text-cyan-400" />
                          Memory ({Math.round((node.resources.memoryUsedGb / node.resources.memoryTotalGb) * 100)}%)
                        </span>
                        <span className="text-white">
                          {node.resources.memoryUsedGb} / {node.resources.memoryTotalGb} GB
                        </span>
                      </div>
                      <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-cyan-500 to-teal-400 rounded-full transition-all duration-500"
                          style={{
                            width: `${(node.resources.memoryUsedGb / node.resources.memoryTotalGb) * 100}%`
                          }}
                        />
                      </div>
                    </div>

                    {/* Disk Storage & I/O */}
                    <div>
                      <div className="flex justify-between text-xs font-mono mb-1">
                        <span className="text-slate-400 flex items-center gap-1.5">
                          <HardDrive className="w-3 h-3 text-blue-400" />
                          Storage ({Math.round((node.resources.diskUsedTb / node.resources.diskTotalTb) * 100)}%)
                        </span>
                        <span className="text-white">
                          {node.resources.diskUsedTb} / {node.resources.diskTotalTb} TB
                        </span>
                      </div>
                      <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-blue-500 to-indigo-400 rounded-full transition-all duration-500"
                          style={{
                            width: `${(node.resources.diskUsedTb / node.resources.diskTotalTb) * 100}%`
                          }}
                        />
                      </div>
                    </div>
                  </div>

                  {/* Network & Peer Spark Indicators */}
                  <div className="grid grid-cols-3 gap-2 mt-3 text-center">
                    <div className="p-2 rounded-lg bg-[#070B14] border border-slate-800/80">
                      <span className="text-[10px] text-slate-500 block font-mono">PEERS</span>
                      <span className="text-xs font-bold font-mono text-cyan-400">
                        {node.connectedPeers}/{node.maxPeers}
                      </span>
                    </div>
                    <div className="p-2 rounded-lg bg-[#070B14] border border-slate-800/80">
                      <span className="text-[10px] text-slate-500 block font-mono">LATENCY</span>
                      <span className="text-xs font-bold font-mono text-emerald-400">{node.latencyMs}ms</span>
                    </div>
                    <div className="p-2 rounded-lg bg-[#070B14] border border-slate-800/80">
                      <span className="text-[10px] text-slate-500 block font-mono">TEMP</span>
                      <span className="text-xs font-bold font-mono text-slate-300">{node.temperatureC}°C</span>
                    </div>
                  </div>

                  {/* Real-time Recharts Mini Trend Curve */}
                  <div className="mt-3 pt-2.5 border-t border-slate-800/80">
                    <div className="flex items-center justify-between text-[11px] text-slate-400 font-mono mb-1">
                      <span className="flex items-center gap-1">
                        <TrendingUp className="w-3 h-3 text-cyan-400" />
                        <span>Live Curve (Recharts)</span>
                      </span>
                      <span className="text-[10px] text-slate-500">
                        <span className="text-purple-400 font-semibold">{node.resources.cpuPercent}% CPU</span> ·{' '}
                        <span className="text-cyan-400 font-semibold">
                          {Math.round((node.resources.memoryUsedGb / node.resources.memoryTotalGb) * 100)}% RAM
                        </span>
                      </span>
                    </div>
                    <div className="h-11 w-full bg-[#070B14] rounded-lg p-1 border border-slate-800/60">
                      <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={node.metricsHistory} margin={{ top: 2, right: 2, left: 2, bottom: 2 }}>
                          <Tooltip content={<CustomMiniTooltip />} />
                          <Line
                            type="monotone"
                            dataKey="cpu"
                            stroke="#A855F7"
                            strokeWidth={1.8}
                            dot={false}
                            isAnimationActive={false}
                          />
                          <Line
                            type="monotone"
                            dataKey="memory"
                            stroke="#06B6D4"
                            strokeWidth={1.8}
                            dot={false}
                            isAnimationActive={false}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  </div>
                </div>

                {/* Card Footer Actions */}
                <div className="flex items-center justify-between gap-2 mt-4 pt-3 border-t border-slate-800">
                  <span className="text-[10px] font-mono text-slate-500">
                    Ping: {node.lastHeartbeatSeconds}s ago
                  </span>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handlePingNode(node.name)}
                      className="px-2.5 py-1 text-xs font-semibold rounded-lg bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-700 transition-colors"
                    >
                      Ping
                    </button>
                    <button
                      onClick={() => {
                        setSelectedChartNodeId(node.id);
                        setInspectNode(node);
                      }}
                      className="flex items-center gap-1 px-3 py-1 text-xs font-semibold rounded-lg bg-blue-600 hover:bg-blue-500 text-white transition-colors"
                    >
                      <Eye className="w-3 h-3" />
                      <span>Inspect</span>
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        /* High Density Table View */
        <div className="bg-[#0A0F1D] border border-slate-800/90 rounded-2xl overflow-hidden shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#070B14] border-b border-slate-800 text-slate-400 font-mono uppercase text-[10px]">
                <tr>
                  <th className="py-3 px-4">Node / Identity</th>
                  <th className="py-3 px-4">Health</th>
                  <th className="py-3 px-4">Role</th>
                  <th className="py-3 px-4">Uptime</th>
                  <th className="py-3 px-4">CPU %</th>
                  <th className="py-3 px-4">Memory</th>
                  <th className="py-3 px-4">Live Trend (Recharts)</th>
                  <th className="py-3 px-4">Storage</th>
                  <th className="py-3 px-4">Network (In/Out)</th>
                  <th className="py-3 px-4">Peers / Latency</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/80 font-mono">
                {filteredNodes.map((node) => (
                  <tr key={node.id} className="hover:bg-slate-900/50 transition-colors">
                    <td className="py-3 px-4">
                      <div className="font-bold text-white font-sans text-xs">{node.name}</div>
                      <div className="text-[10px] text-slate-400 flex items-center gap-1 mt-0.5">
                        <span>{node.location}</span>
                        <span className="text-slate-600">·</span>
                        <span className="text-cyan-400">{node.ipAddress}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4">{getHealthBadge(node.health)}</td>
                    <td className="py-3 px-4 text-slate-300">
                      <span className="px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-[10px]">
                        {node.consensusRole}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <span className="text-emerald-400 font-bold">{node.uptimePercent}%</span>
                      <div className="text-[10px] text-slate-500">{node.uptimeDays} days streak</div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2">
                        <span
                          className={`font-semibold ${
                            node.resources.cpuPercent > 80
                              ? 'text-rose-400'
                              : node.resources.cpuPercent > 60
                              ? 'text-amber-400'
                              : 'text-slate-200'
                          }`}
                        >
                          {node.resources.cpuPercent}%
                        </span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-slate-300">
                      {node.resources.memoryUsedGb} / {node.resources.memoryTotalGb} GB
                    </td>
                    <td className="py-2 px-4 w-32">
                      <div className="h-8 w-28 bg-[#070B14] rounded p-0.5 border border-slate-800">
                        <ResponsiveContainer width="100%" height="100%">
                          <LineChart data={node.metricsHistory}>
                            <Line
                              type="monotone"
                              dataKey="cpu"
                              stroke="#A855F7"
                              strokeWidth={1.5}
                              dot={false}
                              isAnimationActive={false}
                            />
                            <Line
                              type="monotone"
                              dataKey="memory"
                              stroke="#06B6D4"
                              strokeWidth={1.5}
                              dot={false}
                              isAnimationActive={false}
                            />
                          </LineChart>
                        </ResponsiveContainer>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-slate-300">
                      {node.resources.diskUsedTb} / {node.resources.diskTotalTb} TB
                    </td>
                    <td className="py-3 px-4 text-slate-300 text-[11px]">
                      <div>↓ {node.resources.bandwidthInMbps} Mbps</div>
                      <div className="text-slate-400">↑ {node.resources.bandwidthOutMbps} Mbps</div>
                    </td>
                    <td className="py-3 px-4">
                      <div className="text-cyan-400">{node.connectedPeers} peers</div>
                      <div className="text-[10px] text-emerald-400">{node.latencyMs}ms ping</div>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => handlePingNode(node.name)}
                          className="px-2 py-1 text-[11px] rounded bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-800"
                        >
                          Ping
                        </button>
                        <button
                          onClick={() => {
                            setSelectedChartNodeId(node.id);
                            setInspectNode(node);
                          }}
                          className="px-2.5 py-1 text-[11px] font-semibold rounded bg-blue-600 hover:bg-blue-500 text-white"
                        >
                          Inspect
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Inspect Node Telemetry Modal / Drawer */}
      {inspectNode && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#0A0F1D] border border-slate-800 rounded-3xl w-full max-w-3xl overflow-hidden shadow-2xl animate-in zoom-in-95 duration-200">
            {/* Header */}
            <div className="p-6 border-b border-slate-800 flex items-start justify-between bg-[#070B14]">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-2xl bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-cyan-400">
                  <Server className="w-6 h-6" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-bold text-white">{inspectNode.name}</h2>
                    {getHealthBadge(inspectNode.health)}
                  </div>
                  <div className="text-xs text-slate-400 font-mono mt-0.5">
                    Peer ID: <span className="text-cyan-400">{inspectNode.peerId}</span> · IP: {inspectNode.ipAddress}
                  </div>
                </div>
              </div>
              <button
                onClick={() => setInspectNode(null)}
                className="p-1.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-400 hover:text-white transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="p-6 space-y-6 max-h-[75vh] overflow-y-auto">
              {/* Top Quick Status row */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 font-mono">
                <div className="p-3 rounded-xl bg-[#070B14] border border-slate-800">
                  <span className="text-[10px] text-slate-500 block uppercase">Consensus Role</span>
                  <span className="text-xs font-bold text-white">{inspectNode.consensusRole}</span>
                </div>
                <div className="p-3 rounded-xl bg-[#070B14] border border-slate-800">
                  <span className="text-[10px] text-slate-500 block uppercase">Block Height</span>
                  <span className="text-xs font-bold text-cyan-400">#{inspectNode.blockHeight}</span>
                </div>
                <div className="p-3 rounded-xl bg-[#070B14] border border-slate-800">
                  <span className="text-[10px] text-slate-500 block uppercase">Attestation Rate</span>
                  <span className="text-xs font-bold text-emerald-400">{inspectNode.attestationRate}%</span>
                </div>
                <div className="p-3 rounded-xl bg-[#070B14] border border-slate-800">
                  <span className="text-[10px] text-slate-500 block uppercase">Uptime Streak</span>
                  <span className="text-xs font-bold text-white">{inspectNode.uptimeDays} days ({inspectNode.uptimePercent}%)</span>
                </div>
              </div>

              {/* Dedicated Real-Time Recharts Curve in Node Inspector */}
              <div className="bg-[#070B14] border border-slate-800 rounded-2xl p-4 space-y-3">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                  <h4 className="text-xs font-bold text-white font-mono uppercase tracking-wider flex items-center gap-2">
                    <Activity className="w-4 h-4 text-cyan-400" />
                    Live Telemetry Curve (Recharts)
                  </h4>
                  <div className="flex items-center gap-4 text-xs font-mono">
                    <span className="flex items-center gap-1.5 text-purple-400">
                      <span className="w-2 h-2 rounded-full bg-purple-400" />
                      CPU: <strong className="text-white">{inspectNode.resources.cpuPercent}%</strong>
                    </span>
                    <span className="flex items-center gap-1.5 text-cyan-400">
                      <span className="w-2 h-2 rounded-full bg-cyan-400" />
                      RAM:{' '}
                      <strong className="text-white">
                        {Math.round((inspectNode.resources.memoryUsedGb / inspectNode.resources.memoryTotalGb) * 100)}%
                      </strong>
                    </span>
                    <span className="flex items-center gap-1.5 text-emerald-400">
                      <span className="w-2 h-2 rounded-full bg-emerald-400" />
                      RTT: <strong className="text-white">{inspectNode.latencyMs}ms</strong>
                    </span>
                  </div>
                </div>

                <div className="h-44 w-full pt-1">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={inspectNode.metricsHistory} margin={{ top: 10, right: 15, left: -20, bottom: 0 }}>
                      <CartesianGrid stroke="#1E293B" strokeDasharray="3 3" vertical={false} />
                      <XAxis
                        dataKey="timestamp"
                        stroke="#64748B"
                        tick={{ fontSize: 10, fontFamily: 'monospace' }}
                      />
                      <YAxis
                        domain={[0, 100]}
                        stroke="#64748B"
                        tick={{ fontSize: 10, fontFamily: 'monospace' }}
                        unit="%"
                      />
                      <Tooltip content={<CustomTelemetryTooltip />} />
                      <ReferenceLine
                        y={80}
                        stroke="#EF4444"
                        strokeDasharray="3 3"
                        label={{ value: '80% Critical', fill: '#EF4444', fontSize: 10, position: 'insideTopRight' }}
                      />
                      <Line
                        type="monotone"
                        dataKey="cpu"
                        name="CPU Usage"
                        stroke="#A855F7"
                        strokeWidth={2.5}
                        dot={{ r: 3, fill: '#A855F7' }}
                        activeDot={{ r: 5, fill: '#FFFFFF', stroke: '#A855F7', strokeWidth: 2 }}
                        isAnimationActive={false}
                      />
                      <Line
                        type="monotone"
                        dataKey="memory"
                        name="Memory Usage"
                        stroke="#06B6D4"
                        strokeWidth={2.5}
                        dot={{ r: 3, fill: '#06B6D4' }}
                        activeDot={{ r: 5, fill: '#FFFFFF', stroke: '#06B6D4', strokeWidth: 2 }}
                        isAnimationActive={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* Real-time Hardware Metrics Details */}
              <div className="bg-[#070B14] border border-slate-800 rounded-2xl p-4 space-y-4">
                <h4 className="text-xs font-bold text-white font-mono uppercase tracking-wider flex items-center gap-2">
                  <Zap className="w-4 h-4 text-cyan-400" />
                  Live Hardware Resource Breakdown
                </h4>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {/* CPU Breakdown */}
                  <div className="p-3.5 rounded-xl bg-slate-900/60 border border-slate-800/80 space-y-2">
                    <div className="flex justify-between text-xs font-mono">
                      <span className="text-slate-300">CPU Load ({inspectNode.resources.cpuCores} Cores)</span>
                      <span className="text-white font-bold">{inspectNode.resources.cpuPercent}%</span>
                    </div>
                    <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                      <div
                        className="bg-purple-500 h-full rounded-full transition-all duration-500"
                        style={{ width: `${inspectNode.resources.cpuPercent}%` }}
                      />
                    </div>
                    <div className="flex justify-between text-[11px] font-mono text-slate-500">
                      <span>Temp: {inspectNode.temperatureC}°C</span>
                      <span>Freq: 3.4 GHz</span>
                    </div>
                  </div>

                  {/* Memory Breakdown */}
                  <div className="p-3.5 rounded-xl bg-slate-900/60 border border-slate-800/80 space-y-2">
                    <div className="flex justify-between text-xs font-mono">
                      <span className="text-slate-300">RAM Allocated</span>
                      <span className="text-white font-bold">
                        {inspectNode.resources.memoryUsedGb} / {inspectNode.resources.memoryTotalGb} GB
                      </span>
                    </div>
                    <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                      <div
                        className="bg-cyan-500 h-full rounded-full transition-all duration-500"
                        style={{
                          width: `${(inspectNode.resources.memoryUsedGb / inspectNode.resources.memoryTotalGb) * 100}%`
                        }}
                      />
                    </div>
                    <div className="flex justify-between text-[11px] font-mono text-slate-500">
                      <span>Buffer / Cache: 14.2 GB</span>
                      <span>Swap: 0%</span>
                    </div>
                  </div>

                  {/* Disk Storage Breakdown */}
                  <div className="p-3.5 rounded-xl bg-slate-900/60 border border-slate-800/80 space-y-2">
                    <div className="flex justify-between text-xs font-mono">
                      <span className="text-slate-300">NVMe High-Speed Array</span>
                      <span className="text-white font-bold">
                        {inspectNode.resources.diskUsedTb} / {inspectNode.resources.diskTotalTb} TB
                      </span>
                    </div>
                    <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                      <div
                        className="bg-blue-500 h-full rounded-full transition-all duration-500"
                        style={{
                          width: `${(inspectNode.resources.diskUsedTb / inspectNode.resources.diskTotalTb) * 100}%`
                        }}
                      />
                    </div>
                    <div className="flex justify-between text-[11px] font-mono text-slate-500">
                      <span>Disk IOPS: {inspectNode.resources.diskIops} ops/s</span>
                      <span>Read latency: 0.12ms</span>
                    </div>
                  </div>

                  {/* Network Throughput Breakdown */}
                  <div className="p-3.5 rounded-xl bg-slate-900/60 border border-slate-800/80 space-y-2">
                    <div className="flex justify-between text-xs font-mono">
                      <span className="text-slate-300">Bandwidth (BGP Multi-homed)</span>
                      <span className="text-emerald-400 font-bold">{inspectNode.latencyMs}ms RTT</span>
                    </div>
                    <div className="flex items-center justify-between text-xs font-mono text-slate-300">
                      <span>Ingress: {inspectNode.resources.bandwidthInMbps} Mbps</span>
                      <span>Egress: {inspectNode.resources.bandwidthOutMbps} Mbps</span>
                    </div>
                    <div className="flex justify-between text-[11px] font-mono text-slate-500">
                      <span>Peers: {inspectNode.connectedPeers}/{inspectNode.maxPeers}</span>
                      <span>Packet Drop: 0.00%</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Node Telemetry Logs Stream */}
              <div className="bg-[#070B14] border border-slate-800 rounded-2xl p-4 space-y-2">
                <div className="flex items-center justify-between">
                  <h4 className="text-xs font-bold text-white font-mono uppercase tracking-wider flex items-center gap-2">
                    <Terminal className="w-4 h-4 text-cyan-400" />
                    Live Diagnostic & Event Stream
                  </h4>
                  <span className="text-[10px] font-mono text-slate-500">auto-tail active</span>
                </div>
                <div className="p-3 rounded-xl bg-slate-950 border border-slate-900 space-y-2 max-h-36 overflow-y-auto font-mono text-xs">
                  {inspectNode.logs.map((log, idx) => (
                    <div key={idx} className="flex items-start gap-2">
                      <span className="text-slate-500 text-[10px]">{log.timestamp}</span>
                      <span
                        className={`text-[10px] uppercase font-bold px-1 rounded ${
                          log.level === 'warn'
                            ? 'bg-amber-500/20 text-amber-400'
                            : log.level === 'error'
                            ? 'bg-rose-500/20 text-rose-400'
                            : 'bg-blue-500/20 text-cyan-400'
                        }`}
                      >
                        {log.level}
                      </span>
                      <span className="text-slate-300">{log.message}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Action Buttons in Modal */}
              <div className="flex items-center justify-between gap-3 pt-3 border-t border-slate-800">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleSimulateLoad(inspectNode.id)}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/30 text-amber-300 transition-colors"
                  >
                    <Activity className="w-3.5 h-3.5" />
                    <span>Trigger Load Test (+25% CPU)</span>
                  </button>
                  <button
                    onClick={() => handlePingNode(inspectNode.name)}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-200 transition-colors"
                  >
                    <Wifi className="w-3.5 h-3.5 text-cyan-400" />
                    <span>Ping Gateway</span>
                  </button>
                </div>
                <button
                  onClick={() => setInspectNode(null)}
                  className="px-4 py-1.5 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-white transition-colors"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
export default NodeTelemetryDashboard;
