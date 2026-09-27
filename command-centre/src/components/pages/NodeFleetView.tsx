import React, { useState } from 'react';
import { Server, MapPin, Cpu, HardDrive, AlertCircle, ChevronDown, ChevronUp, Activity } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Unavailable, Note, since } from '../common/states';
import { useResource } from '../../lib/useResource';

interface FleetNode {
  id: string;
  name: string;
  status: 'pending' | 'ready' | 'draining' | 'revoked';
  health: 'live' | 'lost' | 'unknown';
  region: string;
  zone: string;
  host: string;
  tiers: string[];
  arch: string;
  cpuMilli: number;
  memBytes: number;
  diskBytes: number;
  usedCpuMilli: number;
  usedMemBytes: number;
  workloads: number;
  uptimeMs: number;
  joinedAt: number;
  lastObsAt: number;
  obsSeq: number;
  features: string[];
  roles: string[];
}

interface TierSummary {
  nodes: number;
  cpuMilli: number;
  memBytes: number;
  usedCpuMilli: number;
  usedMemBytes: number;
  availCpuMilli: number;
  availMemBytes: number;
  workloads: number;
  readyNodes: number;
}

interface ZoneSummary {
  region: string;
  zone: string;
  nodes: number;
  cpuMilli: number;
  memBytes: number;
  usedCpuMilli: number;
  usedMemBytes: number;
  availCpuMilli: number;
  availMemBytes: number;
  workloads: number;
  readyNodes: number;
}

interface FleetSummary {
  totalNodes: number;
  readyNodes: number;
  drainingNodes: number;
  offlineNodes: number;
  totalCpuMilli: number;
  availCpuMilli: number;
  usedCpuMilli: number;
  totalMemBytes: number;
  availMemBytes: number;
  usedMemBytes: number;
  totalDiskBytes: number;
  totalWorkloads: number;
  byTier: Record<string, TierSummary>;
  byZone: Record<string, ZoneSummary>;
}

export default function NodeFleetView() {
  const { session, capabilities } = useSession();
  const fleetRes = useResource<FleetSummary>('/control/fleet');
  const [expandedZones, setExpandedZones] = useState<Record<string, boolean>>({});
  const [expandedTiers, setExpandedTiers] = useState<Record<string, boolean>>({});

  const isSelfHosted = capabilities?.backend.reachable === false;

  if (isSelfHosted) {
    return (
      <div className="space-y-4 pt-2 max-w-3xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Node Fleet</h1>
          <p className="text-[13px] text-slate-400">Self-hosted deployment.</p>
        </div>
        <Glass className="p-5 flex items-start gap-4">
          <IconTile tone="cyan" size="lg">
            <Server className="w-6 h-6" />
          </IconTile>
          <div>
            <div className="text-[15px] font-semibold text-white">Fleet management not available</div>
            <p className="mt-1 text-[13px] text-slate-300">Self-hosted single-node deployments do not have a distributed fleet. This view is for multi-node clusters.</p>
          </div>
        </Glass>
        <Note>Node Fleet shows capacity inventory, tier classification, failure domain distribution, and resource availability across your cluster.</Note>
      </div>
    );
  }

  if (!fleetRes.data) {
    return (
      <div className="space-y-4 pt-2 max-w-4xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Node Fleet</h1>
        </div>
        <Glass className="p-8 text-center">
          <Activity className="w-8 h-8 animate-spin text-cyan-400 mx-auto mb-3" />
          <p className="text-slate-300">Loading fleet inventory...</p>
        </Glass>
      </div>
    );
  }

  const summary = fleetRes.data;
  const cpuPercent = summary.totalCpuMilli > 0 ? Math.round((summary.usedCpuMilli / summary.totalCpuMilli) * 100) : 0;
  const memPercent = summary.totalMemBytes > 0 ? Math.round((summary.usedMemBytes / summary.totalMemBytes) * 100) : 0;

  return (
    <div className="space-y-6 pt-2">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Node Fleet</h1>
        <p className="text-[13px] text-slate-400 mt-1">Inventory: {summary.totalNodes} nodes across {Object.keys(summary.byZone || {}).length} zones</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Glass className="p-4">
          <div className="text-[11px] text-slate-400 uppercase tracking-wider">Nodes</div>
          <div className="mt-2 text-2xl font-bold text-white">{summary.readyNodes}</div>
          <div className="mt-1 text-[12px] text-slate-400">{summary.totalNodes} total</div>
        </Glass>

        <Glass className="p-4">
          <div className="text-[11px] text-slate-400 uppercase tracking-wider">CPU</div>
          <div className="mt-2 text-2xl font-bold text-white">{cpuPercent}%</div>
          <div className="mt-1 text-[12px] text-slate-400">{Math.round(summary.usedCpuMilli / 1000)} / {Math.round(summary.totalCpuMilli / 1000)}c</div>
        </Glass>

        <Glass className="p-4">
          <div className="text-[11px] text-slate-400 uppercase tracking-wider">Memory</div>
          <div className="mt-2 text-2xl font-bold text-white">{memPercent}%</div>
          <div className="mt-1 text-[12px] text-slate-400">{Math.round(summary.usedMemBytes / 1024 / 1024 / 1024)}G / {Math.round(summary.totalMemBytes / 1024 / 1024 / 1024)}G</div>
        </Glass>

        <Glass className="p-4">
          <div className="text-[11px] text-slate-400 uppercase tracking-wider">Workloads</div>
          <div className="mt-2 text-2xl font-bold text-white">{summary.totalWorkloads}</div>
          <div className="mt-1 text-[12px] text-slate-400">running replicas</div>
        </Glass>
      </div>

      {/* By Tier */}
      {Object.keys(summary.byTier || {}).length > 0 && (
        <Glass className="p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Server className="w-5 h-5 text-purple-400" />
            Capacity by Tier
          </h2>
          <div className="space-y-3">
            {Object.entries(summary.byTier).map(([tier, ts]) => (
              <div key={tier} className="border border-slate-700 rounded-lg p-4">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <div className="font-semibold text-white">{tier}</div>
                    <div className="text-[12px] text-slate-400">{ts.readyNodes} ready / {ts.nodes} total nodes</div>
                  </div>
                  <StatusPill status={ts.readyNodes === ts.nodes ? 'active' : 'partial'} label={`${ts.workloads}w`} />
                </div>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <div className="text-[11px] text-slate-400">CPU</div>
                    <div className="mt-1 font-mono text-[13px] text-cyan-300">{Math.round(ts.usedCpuMilli / 1000)} / {Math.round(ts.cpuMilli / 1000)}c</div>
                  </div>
                  <div>
                    <div className="text-[11px] text-slate-400">Memory</div>
                    <div className="mt-1 font-mono text-[13px] text-cyan-300">{Math.round(ts.usedMemBytes / 1024 / 1024 / 1024)}G / {Math.round(ts.memBytes / 1024 / 1024 / 1024)}G</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </Glass>
      )}

      {/* By Zone */}
      {Object.keys(summary.byZone || {}).length > 0 && (
        <Glass className="p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <MapPin className="w-5 h-5 text-cyan-400" />
            Failure Domains (Zones)
          </h2>
          <div className="space-y-3">
            {Object.entries(summary.byZone).map(([zoneKey, zs]) => (
              <div key={zoneKey} className="border border-slate-700 rounded-lg overflow-hidden">
                <button
                  onClick={() => setExpandedZones(p => ({ ...p, [zoneKey]: !p[zoneKey] }))}
                  className="w-full flex items-center justify-between p-4 hover:bg-slate-800/50 transition"
                >
                  <div className="flex items-center gap-3">
                    <MapPin className="w-4 h-4 text-cyan-400" />
                    <div className="text-left">
                      <div className="font-semibold text-white">{zs.region} / {zs.zone}</div>
                      <div className="text-[12px] text-slate-400">{zs.readyNodes} ready / {zs.nodes} total</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right text-sm">
                      <div className="font-mono text-[12px] text-slate-300">{Math.round(zs.usedCpuMilli / 1000)}c / {Math.round(zs.cpuMilli / 1000)}c</div>
                      <div className="font-mono text-[12px] text-slate-300">{Math.round(zs.usedMemBytes / 1024 / 1024 / 1024)}G / {Math.round(zs.memBytes / 1024 / 1024 / 1024)}G</div>
                    </div>
                    {expandedZones[zoneKey] ? <ChevronUp className="w-5 h-5 text-slate-400" /> : <ChevronDown className="w-5 h-5 text-slate-400" />}
                  </div>
                </button>
                {expandedZones[zoneKey] && (
                  <div className="border-t border-slate-700 bg-slate-900/30 p-4 space-y-2 text-sm">
                    <div className="text-[12px] text-slate-400">Workloads: {zs.workloads} running</div>
                    <div className="text-[12px] text-slate-400">Available: {Math.round(zs.availCpuMilli / 1000)}c CPU, {Math.round(zs.availMemBytes / 1024 / 1024 / 1024)}G memory</div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </Glass>
      )}

      {/* Health Status */}
      {(summary.offlineNodes > 0 || summary.drainingNodes > 0) && (
        <Glass className="p-6 border-l-2 border-yellow-500">
          <h3 className="font-semibold text-white mb-3 flex items-center gap-2">
            <AlertCircle className="w-5 h-5 text-yellow-500" />
            Node Status
          </h3>
          <div className="space-y-2 text-sm">
            {summary.drainingNodes > 0 && (
              <div className="text-slate-300">
                <span className="font-semibold text-yellow-400">{summary.drainingNodes}</span> node{summary.drainingNodes !== 1 ? 's' : ''} draining — workloads being evicted
              </div>
            )}
            {summary.offlineNodes > 0 && (
              <div className="text-slate-300">
                <span className="font-semibold text-red-400">{summary.offlineNodes}</span> node{summary.offlineNodes !== 1 ? 's' : ''} offline — no recent observation
              </div>
            )}
          </div>
        </Glass>
      )}

      <Note>Fleet view shows your cluster inventory organized by tier and failure domain. Capacity utilization guides placement decisions and identifies scaling needs.</Note>
    </div>
  );
}
