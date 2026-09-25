import React, { useEffect, useMemo, useState } from 'react';
import {
  Plus,
  Database,
  Network,
  HardDrive,
  Server,
  Archive,
  Layers,
  ShieldCheck,
  Monitor,
  Terminal,
  Settings2,
  Box,
  MoreHorizontal,
  Globe2,
  Lock,
  Zap,
  RefreshCw,
  Activity,
  Upload,
  Copy
} from 'lucide-react';
import { api } from '../../lib/api';
import { StorageFleetOverview, StorageNode, PlatformEvent, Ownership } from '../../types/platform';
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
  OWNERSHIP_META,
  Tone
} from '../common/ui';
import { ConnectDialog, ConnectTarget } from '../common/ConnectDialog';
import { StorageExplorer } from './StorageExplorer';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
  entityId?: string;
}

type Tab = 'overview' | 'mine' | 'volumes' | 'objects' | 'replication' | 'contributions' | 'depin' | 'activity';
type Chip = 'all' | 'online' | 'offline' | 'self' | 'community' | 'depin';
type Layer = 'owned' | 'community' | 'depin' | 'links';

const TABS: { id: Tab; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'mine', label: 'My Storage' },
  { id: 'volumes', label: 'Volumes' },
  { id: 'objects', label: 'Objects & Artifacts' },
  { id: 'replication', label: 'Replication' },
  { id: 'contributions', label: 'Contributions' },
  { id: 'depin', label: 'DePIN Networks' },
  { id: 'activity', label: 'Activity' }
];

const STORAGE_NOTE =
  'Volumes are content-addressed (BLAKE3, FastCDC chunks) and replicated across hosts; snapshots commit at quorum 2 and corrupted chunks are repaired from peers. Erasure coding is not implemented.';

const ADD_STORAGE: (ConnectTarget & { icon: React.ReactNode; tone: Tone })[] = [
  { id: 'local', title: 'Local Disk', subtitle: 'Add storage from this machine', hostName: 'this-computer', note: STORAGE_NOTE, icon: <Monitor className="w-5 h-5" />, tone: 'blue' },
  { id: 'nas', title: 'NAS / Network Storage', subtitle: 'Synology, TrueNAS, QNAP, etc.', hostName: 'nas-01', note: STORAGE_NOTE, icon: <Server className="w-5 h-5" />, tone: 'blue' },
  { id: 'linux', title: 'Linux Server', subtitle: 'Ubuntu, Debian, CentOS, etc.', hostName: 'storage-01', note: STORAGE_NOTE, icon: <Terminal className="w-5 h-5" />, tone: 'violet' },
  {
    id: 's3',
    title: 'S3 Compatible Storage',
    subtitle: 'MinIO, Ceph, Wasabi, Cloudflare R2',
    hostName: 's3',
    unavailable: 'The storage layer replicates volumes between hosts you enrol; there is no S3 backend adapter yet. Run a host on the machine that holds the disks instead.',
    icon: <Database className="w-5 h-5" />,
    tone: 'emerald'
  },
  { id: 'dedicated', title: 'Dedicated Storage Node', subtitle: 'Convert a server into a storage node', hostName: 'vault-01', note: STORAGE_NOTE, icon: <Archive className="w-5 h-5" />, tone: 'violet' },
  {
    id: 'depin',
    title: 'External DePIN Storage',
    subtitle: 'Web3 storage networks',
    hostName: 'depin',
    unavailable: 'DePIN storage networks are listed for planning. Placement onto them is not wired into the control plane yet; data stays on hosts you enrol.',
    icon: <Box className="w-5 h-5" />,
    tone: 'blue'
  },
  { id: 'other', title: 'Other', subtitle: 'Manual installation', hostName: 'host-01', note: STORAGE_NOTE, icon: <Settings2 className="w-5 h-5" />, tone: 'slate' }
];

const tb = (n: number) => `${n % 1 === 0 ? n : n.toFixed(1)} TB`;

export const StorageView: React.FC<Props> = ({ onNavigate, entityId }) => {
  const [data, setData] = useState<StorageFleetOverview | null>(null);
  const [events, setEvents] = useState<PlatformEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>('overview');
  const [chip, setChip] = useState<Chip>('all');
  const [layer, setLayer] = useState<Layer>('owned');
  const [search, setSearch] = useState('');
  const [sortDesc, setSortDesc] = useState(true);
  const [connect, setConnect] = useState<ConnectTarget | null>(entityId === 'add' ? ADD_STORAGE[0] : null);
  const [uploadOpen, setUploadOpen] = useState(false);

  const load = () => {
    setError(null);
    api
      .getStorageFleet()
      .then((r) => setData(r.data))
      .catch((e) => setError(e.message));
    api
      .getOverview()
      .then((r) => setEvents((r.recentActivity || []).filter((e) => /storage|file|bucket|replic/i.test(`${e.type} ${e.resourceType} ${e.message}`))))
      .catch(() => {});
  };
  useEffect(load, []);
  useEffect(() => {
    if (entityId === 'add') setConnect(ADD_STORAGE[0]);
  }, [entityId]);

  const nodes = data?.storageNodes ?? [];
  const stats = useMemo(() => {
    const cap = nodes.reduce((a, n) => a + n.capacityTb, 0);
    const used = nodes.reduce((a, n) => a + n.usedTb, 0);
    const by = (o: Ownership) => nodes.filter((n) => n.ownership === o).length;
    return {
      cap,
      used,
      avail: cap - used,
      online: nodes.filter((n) => n.status === 'Online').length,
      offline: nodes.filter((n) => n.status === 'Offline').length,
      owned: by('owned'),
      community: by('community'),
      depin: by('depin'),
      healthy: nodes.filter((n) => n.status === 'Online').every((n) => n.integrity === 'Healthy')
    };
  }, [nodes]);

  const rows = useMemo(() => {
    let list = nodes.filter((n: StorageNode) => {
      if (chip === 'online') return n.status === 'Online';
      if (chip === 'offline') return n.status === 'Offline';
      if (chip === 'self') return n.ownership === 'owned';
      if (chip === 'community') return n.ownership === 'community';
      if (chip === 'depin') return n.ownership === 'depin';
      return true;
    });
    const q = search.trim().toLowerCase();
    if (q) list = list.filter((n) => [n.name, n.shortId, n.type, n.policy, n.region].some((v) => v.toLowerCase().includes(q)));
    return [...list].sort((a, b) => (sortDesc ? b.capacityTb - a.capacityTb : a.capacityTb - b.capacityTb));
  }, [nodes, chip, search, sortDesc]);

  if (error && !data) {
    return (
      <Glass className="p-8 text-center mt-6">
        <p className="text-rose-300 text-sm">Storage telemetry unavailable: {error}</p>
        <GhostButton onClick={load} className="mt-4">
          <RefreshCw className="w-4 h-4" /> Retry
        </GhostButton>
      </Glass>
    );
  }

  const regions = data?.regions ?? [];
  const heroMarkers = regionMarkers(regions, (r) => {
    if (!r.storageNodes) return null;
    const tone: Tone = r.region === 'Europe' || r.region === 'Asia' ? 'blue' : r.region === 'Africa' ? 'emerald' : r.region === 'South America' ? 'violet' : 'cyan';
    return {
      tone,
      callout: <RegionCallout tone={tone} title={r.region} lines={[`${r.storageNodes} storage node${r.storageNodes > 1 ? 's' : ''}`, tb(r.storageTb)]} icon={<Database className="w-4 h-4" />} />
    };
  });

  // Map layer: regions that hold storage nodes of the selected ownership.
  const regionHas = (region: string, o: Ownership) => nodes.some((n) => n.region === region && n.ownership === o);
  const layerTone: Tone = layer === 'community' ? 'violet' : layer === 'depin' ? 'blue' : 'cyan';
  const mapMarkers = regionMarkers(regions, (r) => {
    if (!r.storageNodes) return null;
    if (layer !== 'links' && !regionHas(r.region, layer)) return null;
    return {
      tone: layerTone,
      callout: <RegionCallout tone={layerTone} title={r.region} lines={[tb(r.storageTb), `${r.storageNodes} node${r.storageNodes > 1 ? 's' : ''}`]} icon={<Database className="w-4 h-4" />} />
    };
  });

  const table = (
    <Glass className="overflow-hidden">
      <TableToolbar search={search} onSearch={setSearch} placeholder="Search storage nodes, volumes, or objects..." onSort={() => setSortDesc(!sortDesc)} sortLabel="Capacity">
        <FilterChips<Chip>
          chips={[
            { id: 'all', label: 'All Storage', count: nodes.length },
            { id: 'online', label: 'Online', count: stats.online, tone: 'emerald' },
            { id: 'offline', label: 'Offline', count: stats.offline, tone: 'rose' },
            { id: 'self', label: 'Self-Hosted', count: stats.owned },
            { id: 'community', label: 'Community', count: stats.community },
            { id: 'depin', label: 'DePIN', count: stats.depin, tone: 'blue' }
          ]}
          active={chip}
          onChange={setChip}
        />
      </TableToolbar>
      <div className="overflow-x-auto">
        <table className="dh-table w-full min-w-[940px]">
          <thead>
            <tr>
              <th>Node / Storage</th>
              <th>Type</th>
              <th>Status</th>
              <th>Capacity</th>
              <th>Used</th>
              <th>Available</th>
              <th>Volumes</th>
              <th>Replicas</th>
              <th>Integrity</th>
              <th>Bandwidth</th>
              <th>Policy</th>
              <th className="text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((n) => {
              const m = OWNERSHIP_META[n.ownership];
              const pctUsed = Math.round((n.usedTb / n.capacityTb) * 100);
              return (
                <tr key={n.id}>
                  <td>
                    <div className="flex items-center gap-2.5">
                      <IconTile tone={n.ownership === 'depin' ? 'amber' : m.tone === 'cyan' ? 'blue' : m.tone} size="sm">
                        <HardDrive className="w-4 h-4" />
                      </IconTile>
                      <div>
                        <div className="font-semibold text-slate-100 whitespace-nowrap">{n.name}</div>
                        <div className="text-[11px] text-slate-500 font-mono">{n.shortId}</div>
                      </div>
                    </div>
                  </td>
                  <td>
                    <div className="text-slate-200 whitespace-nowrap">{n.type}</div>
                    <div className="text-[11px] text-slate-500">({n.filesystem})</div>
                  </td>
                  <td><StatusPill status={n.status} /></td>
                  <td className="tabular-nums text-slate-200">{tb(n.capacityTb)}</td>
                  <td className="tabular-nums">
                    <div className="text-slate-200">{n.usedTb < 1 ? `${Math.round(n.usedTb * 1000)} GB` : tb(n.usedTb)}</div>
                    <div className={`text-[11px] ${pctUsed > 60 ? 'text-rose-400' : 'text-slate-500'}`}>{pctUsed}%</div>
                  </td>
                  <td className="tabular-nums text-slate-300">{n.capacityTb - n.usedTb < 1 ? `${Math.round((n.capacityTb - n.usedTb) * 1000)} GB` : tb(Math.round((n.capacityTb - n.usedTb) * 10) / 10)}</td>
                  <td className="tabular-nums text-slate-300">{n.volumes}</td>
                  <td className="tabular-nums text-slate-300">{n.replicas}</td>
                  <td><StatusPill status={n.integrity === 'Checking' ? 'Checking...' : n.integrity} tone={n.integrity === 'Healthy' ? 'emerald' : 'amber'} /></td>
                  <td className="tabular-nums text-slate-300 whitespace-nowrap">{n.bandwidthServed}</td>
                  <td>
                    <span className="px-2 py-1 rounded-md border border-[rgba(125,190,255,0.16)] bg-white/[0.03] text-[11.5px] text-slate-300 whitespace-nowrap">{n.policy}</span>
                  </td>
                  <td className="text-right">
                    <button onClick={() => setTab('objects')} className="p-1.5 rounded-lg border border-[rgba(125,190,255,0.16)] text-slate-300 hover:text-cyan-300 hover:border-cyan-400/40" aria-label={`Actions for ${n.name}`}>
                      <MoreHorizontal className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              );
            })}
            {rows.length === 0 && (
              <tr>
                <td colSpan={12} className="text-center text-slate-500 py-8">No storage matches this filter.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {data && (
        <div className="flex items-center justify-between px-4 py-2.5 text-[11px] text-slate-500 border-t border-[rgba(125,190,255,0.08)]">
          <span>Observed {new Date(data.observedAt).toLocaleTimeString()}</span>
          <DemoTag mode={data.mode} />
        </div>
      )}
    </Glass>
  );

  const contribution = data && (
    <Glass className="p-4">
      <PanelHeader title="Storage Contribution" right={<span className="px-2.5 py-1 rounded-lg bg-white/[0.04] border border-[rgba(125,190,255,0.14)] text-[11.5px] text-slate-300">Last 30 days</span>} />
      <div className="mt-2 divide-y divide-[rgba(125,190,255,0.08)]">
        {data.contribution.map((c) => (
          <div key={c.id} className="flex items-center gap-3 py-3">
            <IconTile tone={c.tone}>{c.id === 'contrib' ? <Database className="w-5 h-5" /> : c.id === 'bw' ? <Activity className="w-5 h-5" /> : <Layers className="w-5 h-5" />}</IconTile>
            <div className="flex-1 min-w-0">
              <div className="text-[11.5px] text-slate-400">{c.label}</div>
              <div className="text-[18px] font-bold text-white tabular-nums">{c.value}</div>
            </div>
            <span className="text-[12px]"><Delta value={c.deltaPercent} /></span>
          </div>
        ))}
        <div className="flex items-center gap-3 py-3">
          <IconTile tone="amber"><ShieldCheck className="w-5 h-5" /></IconTile>
          <div className="flex-1 min-w-0">
            <div className="text-[11.5px] text-slate-400">Integrity Checks</div>
            <div className="text-[18px] font-bold text-white tabular-nums">{data.integrityChecks.toLocaleString()}</div>
          </div>
          <span className="inline-flex items-center gap-1 text-[12px] font-semibold text-emerald-400">
            <ShieldCheck className="w-3.5 h-3.5" /> {data.integrityPassedPercent}% passed
          </span>
        </div>
      </div>
    </Glass>
  );

  const rail = (
    <>
      <RailPanel icon={<IconTile tone="cyan"><Database className="w-5 h-5" /></IconTile>} title="Add Storage" subtitle="Connect storage from your infrastructure or a decentralized network.">
        <div className="space-y-1.5">
          {ADD_STORAGE.map((t) => (
            <RailItem key={t.id} icon={t.icon} tone={t.tone} title={t.title} subtitle={t.subtitle} onClick={() => setConnect(t)} />
          ))}
        </div>
      </RailPanel>

      <RailPanel
        icon={<IconTile tone="blue"><Box className="w-5 h-5" /></IconTile>}
        title="Storage Networks"
        subtitle="Supported decentralized storage networks."
        action={<ViewAll onClick={() => setTab('depin')} />}
      >
        <div className="space-y-0.5">{data?.networks.map((n) => <NetworkRow key={n.id} network={n} onClick={() => setTab('depin')} />)}</div>
      </RailPanel>

      {data && (
        <BenefitsPanel
          subtitle="Real infrastructure ownership outcomes."
          rings={data.benefits}
          tags={[
            { label: 'Lower Storage Costs', tone: 'emerald', icon: <ShieldCheck className="w-4 h-4" /> },
            { label: 'Full Data Control', tone: 'emerald', icon: <Lock className="w-4 h-4" /> },
            { label: 'Global Availability', tone: 'violet', icon: <Globe2 className="w-4 h-4" /> },
            { label: 'Real Decentralization', tone: 'amber', icon: <Zap className="w-4 h-4" /> }
          ]}
        />
      )}
    </>
  );

  let body: React.ReactNode;
  if (tab === 'overview') {
    body = (
      <>
        <div className="grid grid-cols-2 md:grid-cols-3 2xl:grid-cols-6 gap-3">
          <KpiTile tone="cyan" icon={<Database className="w-6 h-6" />} label="Total Capacity" value={tb(Math.round(stats.cap * 10) / 10)} sub={<Delta value={12} />} spark={[40, 44, 46, 50, 55, 58, 63, 68]} />
          <KpiTile tone="violet" icon={<HardDrive className="w-6 h-6" />} label="Used Storage" value={tb(Math.round(stats.used * 10) / 10)} sub={`${Math.round((stats.used / (stats.cap || 1)) * 100)}% used`} spark={[18, 19, 21, 22, 24, 25, 27, 28]} />
          <KpiTile tone="blue" icon={<Archive className="w-6 h-6" />} label="Available Storage" value={tb(Math.round(stats.avail * 10) / 10)} sub={`${Math.round((stats.avail / (stats.cap || 1)) * 100)}% available`} spark={[30, 32, 31, 35, 36, 38, 39, 40]} />
          <KpiTile
            tone="blue"
            icon={<Server className="w-6 h-6" />}
            label="Storage Nodes"
            value={nodes.length}
            sub={
              <>
                <span className="text-emerald-400">{stats.online} online</span> · <span className="text-rose-400">{stats.offline} offline</span>
              </>
            }
            spark={[5, 6, 6, 7, 7, 8, 8, 9]}
          />
          <KpiTile tone="violet" icon={<Copy className="w-6 h-6" />} label="Replicated Data" value={tb(data?.replicatedTb ?? 0)} sub={`${data?.protectedPercent ?? 0}% protected`} spark={[15, 17, 18, 20, 21, 23, 24, 25]} />
          <KpiTile
            tone="emerald"
            icon={<ShieldCheck className="w-6 h-6" />}
            label="Integrity Status"
            value={`${data?.integrityPassedPercent ?? 0}%`}
            sub={stats.healthy ? 'All online replicas healthy' : 'Some replicas need attention'}
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)] gap-4">
          <Glass className="p-4 overflow-hidden relative">
            <PanelHeader
              icon={<IconTile tone="cyan" size="sm"><Database className="w-4 h-4" /></IconTile>}
              title="Distributed Storage Map"
              subtitle="Global view of your storage nodes, replicas and data placement."
              right={
                <FilterChips<Layer>
                  chips={[
                    { id: 'owned', label: 'My Storage', tone: 'cyan' },
                    { id: 'community', label: 'Community', tone: 'violet' },
                    { id: 'depin', label: 'DePIN', tone: 'blue' },
                    { id: 'links', label: 'Replica Links', tone: 'slate' }
                  ]}
                  active={layer}
                  onChange={setLayer}
                />
              }
            />
            <div className="relative">
              <HoloGlobe
                markers={mapMarkers}
                arcs={regionArcs(regions, (r) => r.storageNodes > 0 && (layer === 'links' || regionHas(r.region, layer as Ownership)))}
                focusLng={20}
                tilt={22}
                speed={1.5}
                center={[0.42, 0.64]}
                radius={0.6}
                resolution={1.3}
                safeRight={150}
                className="h-[320px] -mx-4 -mb-4 mt-1"
              />
              <div className="absolute right-0 top-6 w-[150px] hidden sm:block">
                <Legend
                  items={[
                    { label: 'My Nodes', value: stats.owned, tone: 'cyan' },
                    { label: 'Community Nodes', value: stats.community, tone: 'violet' },
                    { label: 'DePIN Nodes', value: stats.depin, tone: 'blue' },
                    { label: 'Offline', value: stats.offline, tone: 'rose' }
                  ]}
                />
              </div>
            </div>
          </Glass>
          {contribution}
        </div>
        {table}
      </>
    );
  } else if (tab === 'mine') {
    body = table;
  } else if (tab === 'volumes' || tab === 'objects') {
    body = <StorageExplorer embedded onNavigate={onNavigate} openUpload={uploadOpen} key={uploadOpen ? 'u' : 'n'} />;
  } else if (tab === 'replication') {
    body = (
      <Glass className="p-4">
        <PanelHeader title="Replication by Region" subtitle="Capacity and storage nodes per region. Volumes replicate across hosts in different failure domains." />
        <div className="mt-4 space-y-3">
          {regions
            .filter((r) => r.storageNodes)
            .map((r) => (
              <div key={r.region}>
                <div className="flex justify-between text-[12.5px] mb-1">
                  <span className="text-slate-200">{r.region}</span>
                  <span className="text-slate-400 tabular-nums">{tb(r.storageTb)} · {r.storageNodes} node{r.storageNodes > 1 ? 's' : ''}</span>
                </div>
                <div className="h-2 rounded-full bg-white/[0.05] overflow-hidden">
                  <div className="h-full rounded-full bg-[linear-gradient(90deg,#20DDF7,#248BFF,#A855F7)]" style={{ width: `${(r.storageTb / (stats.cap || 1)) * 100}%` }} />
                </div>
              </div>
            ))}
        </div>
      </Glass>
    );
  } else if (tab === 'contributions') {
    body = contribution;
  } else if (tab === 'depin') {
    body = (
      <Glass className="p-4">
        <PanelHeader title="DePIN Storage Networks" subtitle="Networks this control plane can report on. Status reflects local configuration, not on-chain state." />
        <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-2">{data?.networks.map((n) => <NetworkRow key={n.id} network={n} />)}</div>
      </Glass>
    );
  } else {
    body = (
      <Glass className="p-4">
        <PanelHeader title="Storage Activity" subtitle="Recent storage and replication events." />
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
          {events.length === 0 && <li className="py-6 text-center text-slate-500 text-sm">No recent storage activity.</li>}
        </ul>
      </Glass>
    );
  }

  return (
    <>
      <SurfaceLayout rail={rail}>
        <SurfaceHero
          eyebrow={<Eyebrow label="Storage & Data Infrastructure" live={data?.mode === 'live'} />}
          title={
            <>
              Your disks. <Grad from="#38BDF8" to="#818CF8">Your data.</Grad>
              <br />
              Your <Grad from="#A855F7" to="#F472B6">storage network.</Grad>
            </>
          }
          description="Turn storage you control into private, distributed infrastructure. Store your applications and data on your own hardware, replicate across trusted nodes, contribute spare capacity, or connect supported Web3 and DePIN storage networks."
          actions={
            <>
              <PrimaryButton onClick={() => setConnect(ADD_STORAGE[0])}>
                <Plus className="w-4 h-4" /> Add Storage
              </PrimaryButton>
              <GhostButton
                onClick={() => {
                  setTab('volumes');
                  setUploadOpen(true);
                }}
              >
                <Upload className="w-4 h-4" /> Create Volume
              </GhostButton>
              <GhostButton onClick={() => setTab('depin')}>
                <Network className="w-4 h-4" /> Explore Storage Networks
              </GhostButton>
            </>
          }
          markers={heroMarkers}
          arcs={regionArcs(regions, (r) => r.storageNodes > 0)}
          focusLng={10}
        />
        <PageTabs tabs={TABS} active={tab} onChange={(t) => { setTab(t); if (t !== 'volumes') setUploadOpen(false); }} />
        {body}
      </SurfaceLayout>
      <ConnectDialog target={connect} onClose={() => setConnect(null)} />
    </>
  );
};
