import React, { useEffect, useMemo, useState } from 'react';
import {
  Plus,
  ServerCog,
  Network,
  Server,
  Cpu,
  MemoryStick,
  HardDrive,
  Gpu,
  Monitor,
  Terminal,
  Home,
  Cloud,
  CircuitBoard,
  Boxes,
  Settings2,
  MoreHorizontal,
  ShieldCheck,
  Globe2,
  Lock,
  Zap,
  RefreshCw,
  Box
} from 'lucide-react';
import { api } from '../../lib/api';
import { FleetOverview, FleetNode, PlatformEvent } from '../../types/platform';
import { NavRoute } from '../layout/Sidebar';
import { HoloGlobe } from '../common/HoloGlobe';
import { SurfaceLayout, SurfaceHero, Eyebrow, regionMarkers, regionArcs } from '../common/CommandSurface';
import {
  Glass,
  KpiTile,
  PageTabs,
  FilterChips,
  TableToolbar,
  RailPanel,
  RailItem,
  NetworkRow,
  BenefitsPanel,
  RegionCallout,
  PanelHeader,
  IconTile,
  StatusPill,
  Delta,
  ViewAll,
  Grad,
  PrimaryButton,
  GhostButton,
  DemoTag,
  Legend,
  TONE,
  Tone
} from '../common/ui';
import { ConnectDialog, ConnectTarget } from '../common/ConnectDialog';
import { NodesOperations, CommandTab } from './NodesOperations';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
  selectedNodeId?: string;
}

type Tab = 'overview' | 'mine' | 'workloads' | 'contributions' | 'depin' | 'mesh' | 'activity';
type Chip = 'all' | 'online' | 'offline' | 'self' | 'contributing' | 'gpu';
type Layer = 'owned' | 'community' | 'depin' | 'workloads';

const TABS: { id: Tab; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'mine', label: 'My Nodes' },
  { id: 'workloads', label: 'Workloads' },
  { id: 'contributions', label: 'Contributions' },
  { id: 'depin', label: 'DePIN Networks' },
  { id: 'mesh', label: 'Mesh Telemetry' },
  { id: 'activity', label: 'Activity' }
];

const LEGACY_TAB: Partial<Record<Tab, CommandTab>> = {
  workloads: 'workloads',
  contributions: 'benefits',
  depin: 'depin',
  mesh: 'mesh'
};

const ADD_TARGETS: (ConnectTarget & { icon: React.ReactNode; tone: Tone })[] = [
  { id: 'this', title: 'This Computer', subtitle: 'Install on your current machine', hostName: 'this-computer', icon: <Monitor className="w-5 h-5" />, tone: 'blue' },
  { id: 'linux', title: 'Linux Server', subtitle: 'Ubuntu, Debian, CentOS, etc.', hostName: 'linux-01', icon: <Terminal className="w-5 h-5" />, tone: 'violet' },
  { id: 'home', title: 'Home Server', subtitle: 'Your on-premise machine', hostName: 'homelab-01', icon: <Home className="w-5 h-5" />, tone: 'emerald' },
  { id: 'vps', title: 'VPS', subtitle: 'DigitalOcean, Hetzner, Linode, etc.', hostName: 'vps-01', edge: true, icon: <Cloud className="w-5 h-5" />, tone: 'slate', note: 'VPS hosts usually have a public address, so they are invited with the edge role and can terminate TLS for your domains.' },
  { id: 'pi', title: 'Raspberry Pi / ARM', subtitle: 'ARM devices and SBCs', hostName: 'raspberry-pi', arch: 'arm64', icon: <CircuitBoard className="w-5 h-5" />, tone: 'violet' },
  { id: 'gpu', title: 'GPU Machine', subtitle: 'NVIDIA, AMD, or Apple Silicon', hostName: 'gpu-01', icon: <Gpu className="w-5 h-5" />, tone: 'emerald', note: 'Run GPU workloads with the docker runtime so device access and resource limits are enforced; the process runtime does not enforce CPU or memory limits.' },
  { id: 'k8s', title: 'Kubernetes Host', subtitle: 'Existing K8s cluster', hostName: 'k8s-node-01', icon: <Boxes className="w-5 h-5" />, tone: 'blue', note: 'dh-noded runs as a host agent next to kubelet. There is no Kubernetes operator; the host admits work under its own policy like any other host.' },
  { id: 'other', title: 'Other', subtitle: 'Manual installation', hostName: 'host-01', icon: <Settings2 className="w-5 h-5" />, tone: 'slate' }
];

const FLAG: Record<string, string> = { IN: '🇮🇳', DE: '🇩🇪', US: '🇺🇸', GB: '🇬🇧', NL: '🇳🇱', SG: '🇸🇬' };

export const NodesView: React.FC<Props> = ({ onNavigate, selectedNodeId }) => {
  const [fleet, setFleet] = useState<FleetOverview | null>(null);
  const [events, setEvents] = useState<PlatformEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>(selectedNodeId && selectedNodeId !== 'add' ? 'mesh' : 'overview');
  const [chip, setChip] = useState<Chip>('all');
  const [layer, setLayer] = useState<Layer>('owned');
  const [search, setSearch] = useState('');
  const [sortDesc, setSortDesc] = useState(false);
  const [connect, setConnect] = useState<ConnectTarget | null>(selectedNodeId === 'add' ? ADD_TARGETS[0] : null);

  const load = () => {
    setError(null);
    api
      .getFleet()
      .then((r) => setFleet(r.data))
      .catch((e) => setError(e.message));
    api
      .getOverview()
      .then((r) => setEvents(r.recentActivity || []))
      .catch(() => {});
  };

  useEffect(load, []);
  useEffect(() => {
    if (selectedNodeId === 'add') setConnect(ADD_TARGETS[0]);
  }, [selectedNodeId]);

  const owned = useMemo(() => fleet?.nodes.filter((n) => n.ownership === 'owned') ?? [], [fleet]);
  const counts = useMemo(() => {
    const all = fleet?.nodes ?? [];
    return {
      owned: owned.length,
      online: owned.filter((n) => n.status === 'Online').length,
      offline: owned.filter((n) => n.status === 'Offline').length,
      community: all.filter((n) => n.ownership === 'community').length,
      depin: all.filter((n) => n.ownership === 'depin').length,
      offlineAll: all.filter((n) => n.status === 'Offline').length,
      contributing: owned.filter((n) => n.workloadsCommunity > 0).length,
      gpu: owned.filter((n) => n.gpuCount > 0).length
    };
  }, [fleet, owned]);

  const totals = useMemo(() => {
    const s = (f: (n: FleetNode) => number) => owned.reduce((a, n) => a + f(n), 0);
    return {
      cpu: s((n) => n.cpuTotal),
      cpuAlloc: s((n) => n.cpuAllocated),
      mem: s((n) => n.memoryTotalGb),
      memAlloc: s((n) => n.memoryAllocatedGb),
      disk: s((n) => n.storageTotalGb),
      diskAlloc: s((n) => n.storageAllocatedGb),
      gpu: s((n) => n.gpuCount),
      gpuAlloc: s((n) => n.gpuAllocated)
    };
  }, [owned]);

  const rows = useMemo(() => {
    let list = owned.filter((n) => {
      if (chip === 'online') return n.status === 'Online';
      if (chip === 'offline') return n.status === 'Offline';
      if (chip === 'contributing') return n.workloadsCommunity > 0;
      if (chip === 'gpu') return n.gpuCount > 0;
      return true;
    });
    const q = search.trim().toLowerCase();
    if (q) list = list.filter((n) => [n.name, n.shortId, n.country, n.region, ...n.roles].some((v) => v.toLowerCase().includes(q)));
    return [...list].sort((a, b) => (sortDesc ? -1 : 1) * a.name.localeCompare(b.name));
  }, [owned, chip, search, sortDesc]);

  if (error && !fleet) {
    return (
      <Glass className="p-8 text-center mt-6">
        <p className="text-rose-300 text-sm">Fleet telemetry unavailable: {error}</p>
        <GhostButton onClick={load} className="mt-4">
          <RefreshCw className="w-4 h-4" /> Retry
        </GhostButton>
      </Glass>
    );
  }

  const regions = fleet?.regions ?? [];
  const heroMarkers = regionMarkers(regions, (r) => {
    const n = r.owned + r.community + r.depin;
    if (!n) return null;
    const tone: Tone = r.region === 'North America' ? 'emerald' : r.region === 'Europe' ? 'blue' : r.region === 'Asia' ? 'blue' : r.region === 'Africa' ? 'emerald' : 'violet';
    return { tone, callout: <RegionCallout tone={tone} title={r.region} lines={[`${n} nodes`]} onClick={() => setTab('mesh')} /> };
  });
  const heroArcs = regionArcs(regions, (r) => r.owned + r.community + r.depin > 0);

  const layerCount = (r: (typeof regions)[number]) =>
    layer === 'owned' ? r.owned : layer === 'community' ? r.community : layer === 'depin' ? r.depin : 0;
  const layerTone: Tone = layer === 'community' ? 'violet' : layer === 'depin' ? 'blue' : layer === 'workloads' ? 'emerald' : 'cyan';
  const mapMarkers = regionMarkers(regions, (r) => {
    if (layer === 'workloads') {
      if (!r.deployments) return null;
      return { tone: layerTone, callout: <RegionCallout tone={layerTone} title={r.region} lines={[`${r.deployments} workloads`]} /> };
    }
    if (!layerCount(r)) return null;
    const lines = [
      r.owned ? <>My Nodes: <b className="text-cyan-300">{r.owned}</b></> : null,
      r.community ? <>Community: <b className="text-violet-300">{r.community}</b></> : null,
      r.depin ? <>DePIN: <b className="text-blue-300">{r.depin}</b></> : null
    ].filter(Boolean) as React.ReactNode[];
    return { tone: layerTone, callout: <RegionCallout tone={layerTone} title={r.region} lines={lines} /> };
  });

  const rail = (
    <>
      <RailPanel
        icon={<IconTile tone="cyan"><Network className="w-5 h-5" /></IconTile>}
        title="Add a Node"
        subtitle="Turn any machine into a node"
      >
        <div className="space-y-1.5">
          {ADD_TARGETS.map((t) => (
            <RailItem key={t.id} icon={t.icon} tone={t.tone} title={t.title} subtitle={t.subtitle} onClick={() => setConnect(t)} />
          ))}
        </div>
      </RailPanel>

      <RailPanel
        icon={<IconTile tone="blue"><Box className="w-5 h-5" /></IconTile>}
        title="DePIN Networks"
        subtitle="Connect to decentralized compute networks."
        action={<ViewAll onClick={() => setTab('depin')} />}
      >
        <div className="space-y-0.5">
          {fleet?.networks.slice(0, 4).map((n) => <NetworkRow key={n.id} network={n} onClick={() => setTab('depin')} />)}
        </div>
      </RailPanel>

      {fleet && (
        <BenefitsPanel
          subtitle="Based on your current configuration."
          rings={fleet.benefits}
          tags={[
            { label: 'Lower Infrastructure Costs', tone: 'emerald', icon: <ShieldCheck className="w-4 h-4" /> },
            { label: 'Full Data Control', tone: 'emerald', icon: <Lock className="w-4 h-4" /> },
            { label: 'Global Availability', tone: 'violet', icon: <Globe2 className="w-4 h-4" /> },
            { label: 'Real Decentralization', tone: 'amber', icon: <Zap className="w-4 h-4" /> }
          ]}
        />
      )}
    </>
  );

  return (
    <>
      <SurfaceLayout rail={rail}>
        <SurfaceHero
          eyebrow={<Eyebrow label="Nodes & Compute" live={fleet?.mode === 'live'} />}
          title={
            <>
              Your hardware.
              <br />
              <Grad>Your network.</Grad>
              <br />
              <Grad from="#C084FC" to="#F472B6">Your workloads.</Grad>
            </>
          }
          description="Connect computers you control and use them to host applications, contribute spare resources, or participate in supported decentralized compute networks."
          actions={
            <>
              <PrimaryButton onClick={() => setConnect(ADD_TARGETS[0])}>
                <Plus className="w-4 h-4" /> Add Your Machine
              </PrimaryButton>
              <GhostButton onClick={() => setConnect(ADD_TARGETS[3])}>
                <ServerCog className="w-4 h-4" /> Connect Remote Server
              </GhostButton>
              <GhostButton onClick={() => setTab('depin')}>
                <Network className="w-4 h-4" /> Explore DePIN Networks
              </GhostButton>
            </>
          }
          markers={heroMarkers}
          arcs={heroArcs}
          focusLng={50}
          overlay={
            fleet && (
              <Glass strong className="hidden lg:block absolute left-[41%] top-0 px-4 py-3 z-10 min-w-[180px]">
                <Legend
                  items={[
                    { label: 'My Nodes', value: counts.owned, tone: 'cyan' },
                    { label: 'Community Nodes', value: counts.community, tone: 'violet' },
                    { label: 'External DePIN', value: counts.depin, tone: 'blue' },
                    { label: 'Offline', value: counts.offlineAll, tone: 'rose' }
                  ]}
                />
              </Glass>
            )
          }
        />

        <PageTabs tabs={TABS} active={tab} onChange={setTab} />

        {tab === 'overview' || tab === 'mine' ? (
          <>
            {tab === 'overview' && (
              <>
                <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-5 gap-3">
                  <KpiTile
                    tone="cyan"
                    icon={<Server className="w-6 h-6" />}
                    label="My Nodes"
                    value={counts.owned}
                    sub={
                      <>
                        <span className="text-emerald-400">{counts.online} online</span>
                        <br />
                        <span className="text-rose-400">{counts.offline} offline</span>
                      </>
                    }
                    spark={[2, 2, 3, 3, 3, 4, 3, 4]}
                    onClick={() => setTab('mine')}
                  />
                  <KpiTile tone="blue" icon={<Cpu className="w-6 h-6" />} label="Total CPU" value={totals.cpu} sub={<>{totals.cpuAlloc} allocated<br />{totals.cpu - totals.cpuAlloc} available</>} spark={[8, 10, 9, 12, 11, 13, 12, 14]} />
                  <KpiTile tone="violet" icon={<MemoryStick className="w-6 h-6" />} label="Total Memory" value={`${totals.mem} GB`} sub={<>{totals.memAlloc} GB allocated<br />{totals.mem - totals.memAlloc} GB available</>} spark={[30, 34, 33, 40, 38, 44, 42, 45]} />
                  <KpiTile tone="amber" icon={<HardDrive className="w-6 h-6" />} label="Total Storage" value={`${(totals.disk / 1000).toFixed(1)} TB`} sub={<>{totals.diskAlloc} GB allocated<br />{((totals.disk - totals.diskAlloc) / 1000).toFixed(1)} TB available</>} spark={[4, 5, 5, 6, 6, 7, 7, 8]} />
                  <KpiTile tone="violet" icon={<Gpu className="w-6 h-6" />} label="Total GPU" value={totals.gpu} sub={<>{totals.gpuAlloc} allocated<br />{totals.gpu - totals.gpuAlloc} available</>} spark={[0, 1, 0, 1, 1, 1, 1, 1]} />
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)] gap-4">
                  <Glass className="p-4 overflow-hidden">
                    <PanelHeader
                      icon={<IconTile tone="cyan" size="sm"><Globe2 className="w-4 h-4" /></IconTile>}
                      title="Global Infrastructure Map"
                      subtitle="Live view of your nodes, community capacity and connected DePIN networks."
                      right={
                        <FilterChips<Layer>
                          chips={[
                            { id: 'owned', label: 'My Nodes', tone: 'cyan' },
                            { id: 'community', label: 'Community', tone: 'violet' },
                            { id: 'depin', label: 'DePIN', tone: 'blue' },
                            { id: 'workloads', label: 'Workloads', tone: 'emerald' }
                          ]}
                          active={layer}
                          onChange={setLayer}
                        />
                      }
                    />
                    <HoloGlobe
                      markers={mapMarkers}
                      arcs={regionArcs(regions, (r) => (layer === 'workloads' ? r.deployments : layerCount(r)) > 0)}
                      focusLng={30}
                      tilt={22}
                      speed={1.5}
                      center={[0.5, 0.64]}
                      radius={0.6}
                      resolution={1.3}
                      className="h-[330px] -mx-4 -mb-4 mt-1"
                    />
                  </Glass>

                  <Glass className="p-4">
                    <PanelHeader
                      title="Network Contribution"
                      right={<span className="px-2.5 py-1 rounded-lg bg-white/[0.04] border border-[rgba(125,190,255,0.14)] text-[11.5px] text-slate-300">Last 30 days</span>}
                    />
                    <div className="mt-3 divide-y divide-[rgba(125,190,255,0.08)]">
                      {fleet?.contribution.map((c) => (
                        <div key={c.id} className="flex items-center gap-3 py-3">
                          <IconTile tone={c.tone}>
                            {c.id === 'cpu' ? <Cpu className="w-5 h-5" /> : c.id === 'gpu' ? <Gpu className="w-5 h-5" /> : c.id === 'storage' ? <HardDrive className="w-5 h-5" /> : <Zap className="w-5 h-5" />}
                          </IconTile>
                          <div className="flex-1 min-w-0">
                            <div className="text-[11.5px] text-slate-400">{c.label}</div>
                            <div className="text-[16px] font-bold text-white tabular-nums">{c.value}</div>
                          </div>
                          <span className="text-[12px]"><Delta value={c.deltaPercent} /></span>
                        </div>
                      ))}
                    </div>
                  </Glass>
                </div>
              </>
            )}

            <Glass className="overflow-hidden">
              <TableToolbar search={search} onSearch={setSearch} placeholder="Search nodes by name, location, or ID..." onSort={() => setSortDesc(!sortDesc)}>
                <FilterChips<Chip>
                  chips={[
                    { id: 'all', label: 'All Nodes', count: counts.owned },
                    { id: 'online', label: 'Online', count: counts.online, tone: 'emerald' },
                    { id: 'offline', label: 'Offline', count: counts.offline, tone: 'rose' },
                    { id: 'self', label: 'Self-Hosted', count: counts.owned },
                    { id: 'contributing', label: 'Contributing', count: counts.contributing },
                    { id: 'gpu', label: 'GPU', count: counts.gpu, tone: 'violet' }
                  ]}
                  active={chip}
                  onChange={setChip}
                />
              </TableToolbar>
              <div className="overflow-x-auto">
                <table className="dh-table w-full min-w-[900px]">
                  <thead>
                    <tr>
                      <th>Node Name</th>
                      <th>Status</th>
                      <th>Roles</th>
                      <th>Resources (CPU / RAM / Storage)</th>
                      <th>Location</th>
                      <th>Uptime</th>
                      <th>Workloads</th>
                      <th>Contribution</th>
                      <th className="text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((n) => (
                      <tr key={n.id}>
                        <td>
                          <div className="flex items-center gap-2.5">
                            <IconTile tone={n.status === 'Offline' ? 'violet' : 'blue'} size="sm">
                              {n.gpuCount ? <Gpu className="w-4 h-4" /> : <Server className="w-4 h-4" />}
                            </IconTile>
                            <div>
                              <div className="font-semibold text-slate-100 whitespace-nowrap">{n.name}</div>
                              <div className="text-[11px] text-slate-500 font-mono">{n.shortId}</div>
                            </div>
                          </div>
                        </td>
                        <td><StatusPill status={n.status} /></td>
                        <td>
                          <div className="flex flex-col gap-1">
                            {n.roles.map((r) => (
                              <span key={r} className="w-fit px-1.5 py-0.5 rounded border border-[rgba(125,190,255,0.18)] bg-white/[0.03] text-[11px] text-slate-300">{r}</span>
                            ))}
                          </div>
                        </td>
                        <td className="tabular-nums text-slate-300">
                          <div className="grid grid-cols-[auto_auto] gap-x-5">
                            <span>{n.cpuAllocated} / {n.cpuTotal} CPU</span>
                            <span>{n.memoryAllocatedGb} / {n.memoryTotalGb} GB</span>
                            <span className="text-slate-400">{n.storageAllocatedGb} / {n.storageTotalGb} GB</span>
                            <span className="text-slate-400">{n.gpuModel ? `${n.gpuCount} × ${n.gpuModel}` : ''}</span>
                          </div>
                        </td>
                        <td>
                          <div className="flex items-center gap-2">
                            <span className="text-base" aria-hidden="true">{FLAG[n.countryCode] ?? '🌐'}</span>
                            <div className="leading-tight">
                              <div className="text-slate-200">{n.country}</div>
                              <div className="text-[11px] text-slate-500">({n.region})</div>
                            </div>
                          </div>
                        </td>
                        <td className="tabular-nums text-slate-300">{n.uptime}</td>
                        <td>
                          <div className="text-slate-100 font-semibold">{n.workloadsOwner + n.workloadsCommunity}</div>
                          {n.workloadsOwner + n.workloadsCommunity > 0 && (
                            <div className="text-[11px] text-slate-500">({n.workloadsOwner} owner / {n.workloadsCommunity} community)</div>
                          )}
                        </td>
                        <td>
                          {n.contribution.cpuHours || n.contribution.gpuHours ? (
                            <div className="flex items-start gap-2">
                              <span className="w-2 h-2 rounded-full bg-emerald-400 mt-1.5 shadow-[0_0_6px_#34d399]" />
                              <div className="leading-tight tabular-nums">
                                <div className="text-slate-200">
                                  {n.contribution.gpuHours ? `${n.contribution.gpuHours} GPUh` : `${n.contribution.cpuHours.toLocaleString()} CPUh`}
                                </div>
                                <div className="text-[11px] text-slate-500">{n.contribution.bandwidthGb} GB</div>
                              </div>
                            </div>
                          ) : (
                            <span className="text-slate-500">—</span>
                          )}
                        </td>
                        <td>
                          <div className="flex justify-end gap-1.5">
                            <button onClick={() => setTab('mesh')} className="p-1.5 rounded-lg border border-[rgba(125,190,255,0.16)] text-slate-300 hover:text-cyan-300 hover:border-cyan-400/40" aria-label={`Open console for ${n.name}`}>
                              <Monitor className="w-4 h-4" />
                            </button>
                            <button onClick={() => setTab('mesh')} className="p-1.5 rounded-lg border border-[rgba(125,190,255,0.16)] text-slate-300 hover:text-cyan-300 hover:border-cyan-400/40" aria-label={`More actions for ${n.name}`}>
                              <MoreHorizontal className="w-4 h-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                    {rows.length === 0 && (
                      <tr>
                        <td colSpan={9} className="text-center text-slate-500 py-8">No nodes match this filter.</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
              {fleet && (
                <div className="flex items-center justify-between px-4 py-2.5 text-[11px] text-slate-500 border-t border-[rgba(125,190,255,0.08)]">
                  <span>Observed {new Date(fleet.observedAt).toLocaleTimeString()}</span>
                  <DemoTag mode={fleet.mode} />
                </div>
              )}
            </Glass>
          </>
        ) : tab === 'activity' ? (
          <Glass className="p-4">
            <PanelHeader title="Node Activity" subtitle="Recent control-plane events across your hosts." />
            <ul className="mt-3 divide-y divide-[rgba(125,190,255,0.08)]">
              {events.map((e) => (
                <li key={e.id} className="flex items-center gap-3 py-2.5">
                  <StatusPill status={e.severity === 'error' ? 'Failed' : e.severity === 'warn' ? 'Degraded' : 'Success'} />
                  <div className="flex-1 min-w-0">
                    <div className="text-[13px] text-slate-100 truncate">{e.message}</div>
                    <div className="text-[11px] text-slate-500">{e.type} · {e.actor}</div>
                  </div>
                  <span className="text-[11px] text-slate-500 whitespace-nowrap">{e.timestamp}</span>
                </li>
              ))}
              {events.length === 0 && <li className="py-6 text-center text-slate-500 text-sm">No recent activity.</li>}
            </ul>
          </Glass>
        ) : (
          <NodesOperations embedded tab={LEGACY_TAB[tab]} onNavigate={onNavigate} selectedNodeId={selectedNodeId !== 'add' ? selectedNodeId : undefined} />
        )}
      </SurfaceLayout>

      <ConnectDialog target={connect} onClose={() => setConnect(null)} />
    </>
  );
};
