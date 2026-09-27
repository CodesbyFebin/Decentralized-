import React, { useState } from 'react';
import { Cpu, HardDrive, Zap, Network, Container, Server, AlertCircle, ChevronDown, ChevronUp } from 'lucide-react';
import { Glass, PanelHeader, IconTile, StatusPill } from '../common/ui';
import { Unavailable, Note, since } from '../common/states';

interface FactValue {
  value: string | number;
  measured: boolean;
}

interface NetworkInterface {
  name: string;
  ipv4?: string;
  ipv6?: string;
  macAddress?: string;
  mtu?: number;
  speed?: string;
}

interface ContainerRuntime {
  name: string;
  version?: string;
  status: 'available' | 'unavailable';
}

interface VirtualizationInfo {
  type: 'kvm' | 'xen' | 'hyperv' | 'vmware' | 'docker' | 'lxc' | 'none';
  hypervisor?: string;
  detected: boolean;
}

interface HardwareFacts {
  memBytes?: FactValue;
  cpuModel?: FactValue;
  cpuCores?: FactValue;
  cpuThreads?: FactValue;
  swapBytes?: FactValue;
  disks?: Array<{
    device: string;
    sizeBytes: number;
    type: string;
  }>;
  gpus?: Array<{
    name: string;
    memory: number;
    driver?: string;
  }>;
  dataFilesystem?: FactValue;
  uptime?: FactValue;
  nics?: NetworkInterface[];
  containerRuntimes?: ContainerRuntime[];
  virtualization?: VirtualizationInfo;
  unknown?: string[];
}

interface NodeHardwareViewProps {
  facts?: HardwareFacts;
  loading?: boolean;
}

export default function NodeHardwareView({ facts, loading }: NodeHardwareViewProps) {
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    cpu: true,
    memory: true,
    storage: false,
    gpu: false,
    network: false,
    container: false,
    virtualization: false,
  });

  const toggleSection = (section: string) => {
    setExpandedSections((prev) => ({
      ...prev,
      [section]: !prev[section],
    }));
  };

  const formatBytes = (bytes: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  const getMeasuredIcon = (measured: boolean) => {
    return measured ? (
      <span className="text-[10px] text-emerald-400">●</span>
    ) : (
      <span className="text-[10px] text-slate-500">○</span>
    );
  };

  if (loading) {
    return <Glass className="p-5"><p className="text-slate-400">Loading hardware facts...</p></Glass>;
  }

  if (!facts) {
    return <Unavailable title="Hardware facts unavailable" message="Host has not reported hardware information yet." />;
  }

  return (
    <div className="space-y-3 pt-2">
      {/* CPU Section */}
      <Glass className="p-4">
        <div
          className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
          onClick={() => toggleSection('cpu')}
        >
          <div className="flex items-center gap-3">
            <IconTile tone="cyan" size="sm">
              <Cpu className="w-4 h-4" />
            </IconTile>
            <div>
              <div className="text-sm font-semibold text-white">CPU</div>
              <div className="text-[11px] text-slate-400">
                {facts.cpuCores?.value} cores · {facts.cpuThreads?.value} threads
              </div>
            </div>
          </div>
          {expandedSections.cpu ? (
            <ChevronUp className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          )}
        </div>

        {expandedSections.cpu && (
          <div className="mt-3 space-y-2 text-[12px]">
            <div className="flex justify-between items-center">
              <span className="text-slate-400">Model</span>
              <div className="flex items-center gap-2">
                {getMeasuredIcon(facts.cpuModel?.measured ?? false)}
                <span className="text-slate-100">{facts.cpuModel?.value || 'Unknown'}</span>
              </div>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Cores</span>
              <div className="flex items-center gap-2">
                {getMeasuredIcon(facts.cpuCores?.measured ?? false)}
                <span className="text-slate-100">{facts.cpuCores?.value || '—'}</span>
              </div>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Threads</span>
              <div className="flex items-center gap-2">
                {getMeasuredIcon(facts.cpuThreads?.measured ?? false)}
                <span className="text-slate-100">{facts.cpuThreads?.value || '—'}</span>
              </div>
            </div>
          </div>
        )}
      </Glass>

      {/* Memory Section */}
      <Glass className="p-4">
        <div
          className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
          onClick={() => toggleSection('memory')}
        >
          <div className="flex items-center gap-3">
            <IconTile tone="emerald" size="sm">
              <Zap className="w-4 h-4" />
            </IconTile>
            <div>
              <div className="text-sm font-semibold text-white">Memory</div>
              <div className="text-[11px] text-slate-400">
                {formatBytes(Number(facts.memBytes?.value) || 0)} RAM
              </div>
            </div>
          </div>
          {expandedSections.memory ? (
            <ChevronUp className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          )}
        </div>

        {expandedSections.memory && (
          <div className="mt-3 space-y-2 text-[12px]">
            <div className="flex justify-between">
              <span className="text-slate-400">RAM</span>
              <div className="flex items-center gap-2">
                {getMeasuredIcon(facts.memBytes?.measured ?? false)}
                <span className="text-slate-100">{formatBytes(Number(facts.memBytes?.value) || 0)}</span>
              </div>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Swap</span>
              <div className="flex items-center gap-2">
                {getMeasuredIcon(facts.swapBytes?.measured ?? false)}
                <span className="text-slate-100">{formatBytes(Number(facts.swapBytes?.value) || 0)}</span>
              </div>
            </div>
          </div>
        )}
      </Glass>

      {/* Storage Section */}
      {facts.disks && facts.disks.length > 0 && (
        <Glass className="p-4">
          <div
            className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
            onClick={() => toggleSection('storage')}
          >
            <div className="flex items-center gap-3">
              <IconTile tone="blue" size="sm">
                <HardDrive className="w-4 h-4" />
              </IconTile>
              <div>
                <div className="text-sm font-semibold text-white">Storage</div>
                <div className="text-[11px] text-slate-400">{facts.disks.length} disk(s)</div>
              </div>
            </div>
            {expandedSections.storage ? (
              <ChevronUp className="w-4 h-4 text-slate-400" />
            ) : (
              <ChevronDown className="w-4 h-4 text-slate-400" />
            )}
          </div>

          {expandedSections.storage && (
            <div className="mt-3 space-y-2">
              {facts.disks.map((disk) => (
                <div key={disk.device} className="text-[12px] p-2 bg-white/3 rounded">
                  <div className="flex justify-between mb-1">
                    <span className="text-slate-100 font-mono">{disk.device}</span>
                    <span className="text-slate-400">{disk.type}</span>
                  </div>
                  <div className="text-slate-400">{formatBytes(disk.sizeBytes)}</div>
                </div>
              ))}
            </div>
          )}
        </Glass>
      )}

      {/* GPU Section */}
      {facts.gpus && facts.gpus.length > 0 && (
        <Glass className="p-4">
          <div
            className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
            onClick={() => toggleSection('gpu')}
          >
            <div className="flex items-center gap-3">
              <IconTile tone="purple" size="sm">
                <Zap className="w-4 h-4" />
              </IconTile>
              <div>
                <div className="text-sm font-semibold text-white">GPUs</div>
                <div className="text-[11px] text-slate-400">{facts.gpus.length} GPU(s)</div>
              </div>
            </div>
            {expandedSections.gpu ? (
              <ChevronUp className="w-4 h-4 text-slate-400" />
            ) : (
              <ChevronDown className="w-4 h-4 text-slate-400" />
            )}
          </div>

          {expandedSections.gpu && (
            <div className="mt-3 space-y-2">
              {facts.gpus.map((gpu, idx) => (
                <div key={idx} className="text-[12px] p-2 bg-white/3 rounded">
                  <div className="flex justify-between mb-1">
                    <span className="text-slate-100 font-semibold">{gpu.name}</span>
                    {gpu.driver && <span className="text-slate-400 text-[11px]">{gpu.driver}</span>}
                  </div>
                  <div className="text-slate-400">{formatBytes(gpu.memory)}</div>
                </div>
              ))}
            </div>
          )}
        </Glass>
      )}

      {/* Network Interfaces Section */}
      {facts.nics && facts.nics.length > 0 && (
        <Glass className="p-4">
          <div
            className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
            onClick={() => toggleSection('network')}
          >
            <div className="flex items-center gap-3">
              <IconTile tone="indigo" size="sm">
                <Network className="w-4 h-4" />
              </IconTile>
              <div>
                <div className="text-sm font-semibold text-white">Network</div>
                <div className="text-[11px] text-slate-400">{facts.nics.length} interface(s)</div>
              </div>
            </div>
            {expandedSections.network ? (
              <ChevronUp className="w-4 h-4 text-slate-400" />
            ) : (
              <ChevronDown className="w-4 h-4 text-slate-400" />
            )}
          </div>

          {expandedSections.network && (
            <div className="mt-3 space-y-2">
              {facts.nics.map((nic) => (
                <div key={nic.name} className="text-[12px] p-2 bg-white/3 rounded">
                  <div className="flex justify-between mb-1">
                    <span className="text-slate-100 font-mono">{nic.name}</span>
                    {nic.speed && <span className="text-slate-400">{nic.speed}</span>}
                  </div>
                  <div className="space-y-0.5 text-slate-400">
                    {nic.ipv4 && <div>IPv4: {nic.ipv4}</div>}
                    {nic.ipv6 && <div>IPv6: {nic.ipv6}</div>}
                    {nic.macAddress && <div className="font-mono text-[11px]">MAC: {nic.macAddress}</div>}
                    {nic.mtu && <div>MTU: {nic.mtu}</div>}
                  </div>
                </div>
              ))}
            </div>
          )}
        </Glass>
      )}

      {/* Container Runtimes Section */}
      {facts.containerRuntimes && facts.containerRuntimes.length > 0 && (
        <Glass className="p-4">
          <div
            className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
            onClick={() => toggleSection('container')}
          >
            <div className="flex items-center gap-3">
              <IconTile tone="orange" size="sm">
                <Container className="w-4 h-4" />
              </IconTile>
              <div>
                <div className="text-sm font-semibold text-white">Container Runtimes</div>
                <div className="text-[11px] text-slate-400">
                  {facts.containerRuntimes.filter((r) => r.status === 'available').length} available
                </div>
              </div>
            </div>
            {expandedSections.container ? (
              <ChevronUp className="w-4 h-4 text-slate-400" />
            ) : (
              <ChevronDown className="w-4 h-4 text-slate-400" />
            )}
          </div>

          {expandedSections.container && (
            <div className="mt-3 space-y-2">
              {facts.containerRuntimes.map((runtime) => (
                <div key={runtime.name} className="text-[12px] p-2 bg-white/3 rounded flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-slate-100 font-mono">{runtime.name}</span>
                    {runtime.version && <span className="text-slate-400">v{runtime.version}</span>}
                  </div>
                  <StatusPill
                    status={runtime.status === 'available' ? 'active' : 'inactive'}
                    label={runtime.status === 'available' ? 'Ready' : 'Not Found'}
                  />
                </div>
              ))}
            </div>
          )}
        </Glass>
      )}

      {/* Virtualization Section */}
      {facts.virtualization && (
        <Glass className="p-4">
          <div
            className="flex items-center justify-between cursor-pointer hover:bg-white/5 p-2 rounded -mx-2 -my-2"
            onClick={() => toggleSection('virtualization')}
          >
            <div className="flex items-center gap-3">
              <IconTile tone="pink" size="sm">
                <Server className="w-4 h-4" />
              </IconTile>
              <div>
                <div className="text-sm font-semibold text-white">Virtualization</div>
                <div className="text-[11px] text-slate-400">
                  {facts.virtualization.detected ? facts.virtualization.type : 'None detected'}
                </div>
              </div>
            </div>
            {expandedSections.virtualization ? (
              <ChevronUp className="w-4 h-4 text-slate-400" />
            ) : (
              <ChevronDown className="w-4 h-4 text-slate-400" />
            )}
          </div>

          {expandedSections.virtualization && (
            <div className="mt-3 space-y-2 text-[12px]">
              <div className="flex justify-between">
                <span className="text-slate-400">Type</span>
                <span className="text-slate-100 uppercase font-semibold">{facts.virtualization.type}</span>
              </div>
              {facts.virtualization.hypervisor && (
                <div className="flex justify-between">
                  <span className="text-slate-400">Hypervisor</span>
                  <span className="text-slate-100">{facts.virtualization.hypervisor}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-slate-400">Status</span>
                <span className={facts.virtualization.detected ? 'text-emerald-300' : 'text-slate-400'}>
                  {facts.virtualization.detected ? 'Detected' : 'Not detected'}
                </span>
              </div>
            </div>
          )}
        </Glass>
      )}

      {/* Unknown Fields */}
      {facts.unknown && facts.unknown.length > 0 && (
        <Glass className="p-4 border border-amber-400/20">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-4 h-4 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <div className="text-[12px] font-semibold text-amber-300">Unmeasurable fields</div>
              <div className="text-[11px] text-slate-400 mt-1">
                These hardware properties could not be measured:
              </div>
              <div className="text-[11px] text-slate-300 mt-2 font-mono space-y-1">
                {facts.unknown.map((field) => (
                  <div key={field}>— {field}</div>
                ))}
              </div>
            </div>
          </div>
        </Glass>
      )}

      <Note>
        Measured facts are marked with ● (filled circle). Unmeasurable values show ○ (hollow circle) and list in the
        unmeasurable fields section. Hardware discovery runs on enrollment and periodically thereafter.
      </Note>
    </div>
  );
}
