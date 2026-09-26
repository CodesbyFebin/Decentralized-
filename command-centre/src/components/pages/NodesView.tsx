import React, { useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { Plus, ServerCog, Network, Server, Cpu, MemoryStick, HardDrive, Monitor, Terminal, Home, Cloud, CircuitBoard, Boxes, Settings2, Gpu, Box, Radio, ShieldCheck, Layers } from 'lucide-react';
import type { ClusterRec, NodeRec, Overview } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { HoloGlobe } from '../common/HoloGlobe';
import { SurfaceLayout, SurfaceHero, Eyebrow, regionGroups, groupMarkers, meshArcs, NoGeoNote } from '../common/CommandSurface';
import { Glass, KpiTile, PageTabs, FilterChips, TableToolbar, RailPanel, RailItem, NetworkRow, PanelHeader, IconTile, StatusPill, Grad, PrimaryButton, GhostButton, Legend, ViewAll, Tone } from '../common/ui';
import { Gate, FreshnessPill, fmtBytes, since, TruthTag, Unavailable, Empty, Note } from '../common/states';
import { ConnectDialog, ConnectTarget } from '../common/ConnectDialog';

type Tab = 'overview' | 'mine' | 'workloads' | 'contributions' | 'depin' | 'activity';
type Chip = 'all' | 'healthy' | 'degraded' | 'offline' | 'unknown' | 'edge';

const TABS: { id: Tab; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'mine', label: 'My Nodes' },
  { id: 'workloads', label: 'Workloads' },
  { id: 'contributions', label: 'Contributions' },
  { id: 'depin', label: 'DePIN Networks' },
  { id: 'activity', label: 'Activity' }
];

export const ADD_TARGETS: (ConnectTarget & { icon: React.ReactNode; tone: Tone })[] = [
  { id: 'this', title: 'This Computer', subtitle: 'Install on your current machine', hostName: 'this-computer', icon: <Monitor className="w-5 h-5" />, tone: 'blue' },
  { id: 'linux', title: 'Linux Server', subtitle: 'Ubuntu, Debian, CentOS, etc.', hostName: 'linux-01', icon: <Terminal className="w-5 h-5" />, tone: 'violet' },
  { id: 'home', title: 'Home Server', subtitle: 'Your on-premise machine', hostName: 'homelab-01', icon: <Home className="w-5 h-5" />, tone: 'emerald' },
  { id: 'vps', title: 'VPS', subtitle: 'DigitalOcean, Hetzner, Linode, etc.', hostName: 'vps-01', edge: true, icon: <Cloud className="w-5 h-5" />, tone: 'slate', note: 'VPS hosts usually have a public address, so they are invited with the edge role and can terminate TLS for your domains.' },
  { id: 'pi', title: 'Raspberry Pi / ARM', subtitle: 'ARM devices and SBCs', hostName: 'raspberry-pi', arch: 'arm64', icon: <CircuitBoard className="w-5 h-5" />, tone: 'violet' },
  { id: 'gpu', title: 'GPU Machine', subtitle: 'NVIDIA, AMD, or Apple Silicon', hostName: 'gpu-01', icon: <Gpu className="w-5 h-5" />, tone: 'emerald', note: 'GPU discovery and scheduling are not implemented yet: the host joins as a normal CPU host. Use the docker runtime for enforced resource limits.' },
  { id: 'k8s', title: 'Kubernetes Host', subtitle: 'Existing K8s cluster', hostName: 'k8s-node-01', icon: <Boxes className="w-5 h-5" />, tone: 'blue', note: 'dh-noded runs as a host agent next to kubelet. There is no Kubernetes operator; the host admits work under its own policy like any other host.' },
  { id: 'other', title: 'Other', subtitle: 'Manual installation', hostName: 'host-01', icon: <Settings2 className="w-5 h-5" />, tone: 'slate' }
];

interface NodesPayload {
  nodes: NodeRec[];
  metrics: Overview['metrics'];
  cluster: ClusterRec;
}

export default function NodesView() {
  const res = useResource<NodesPayload>('/nodes');
  const [params, setParams] = useSearchParams();
  const tab = (params.get('tab') as Tab) || 'overview';
  const setTab = (t: Tab) => setParams((p) => (t === 'overview' ? (p.delete('tab'), p) : (p.set('tab', t), p)), { replace: true });
  const [chip, setChip] = useState<Chip>('all');
  const [search, setSearch] = useState('');
  const [sortDesc, setSortDesc] = useState(false);
  const [connect, setConnect] = useState<ConnectTarget | null>(params.get('add') ? ADD_TARGETS[0] : null);
  const { capabilities, can } = useSession();
  const navigate = useNavigate();
  const caps = capabilities?.items;

  const nodes = res.data?.nodes ?? [];
  const groups = useMemo(() => regionGroups(nodes), [nodes]);
  const counts = useMemo(
    () => ({
      healthy: nodes.filter((n) => n.health === 'HEALTHY').length,
      degraded: nodes.filter((n) => n.health === 'DEGRADED').length,
      offline: nodes.filter((n) => n.health === 'OFFLINE').length,
      unknown: nodes.filter((n) => n.health === 'UNKNOWN').length,
      edge: nodes.filter((n) => n.isEdge).length,
      pending: nodes.filter((n) => n.lifecycle === 'PENDING_APPROVAL').length
    }),
    [nodes]
  );
  const rows = useMemo(() => {
    let list = nodes.filter((n) => (chip === 'all' ? true : chip === 'edge' ? n.isEdge : n.health === chip.toUpperCase()));
    const q = search.trim().toLowerCase();
    if (q) list = list.filter((n) => [n.name, n.id, n.region, n.host, ...n.roles].some((v) => v.toLowerCase().includes(q)));
    return [...list].sort((a, b) => (sortDesc ? -1 : 1) * a.name.localeCompare(b.name));
  }, [nodes, chip, search, sortDesc]);

  const observed = nodes.filter((n) => n.facts);
  const memMeasured = observed.filter((n) => n.facts!.memBytes !== null);
  const sum = (f: (n: NodeRec) => number) => observed.reduce((a, n) => a + f(n), 0);
  const stale = res.stale;

  const rail = (
    <>
      <RailPanel icon={<IconTile tone="cyan"><Network className="w-5 h-5" /></IconTile>} title="Add a Node" subtitle="Turn any machine into a node">
        {!can('api.admin') && <Note>Creating join invites needs an admin capability; the commands below still show the flow.</Note>}
        <div className="space-y-1.5 mt-2">
          {ADD_TARGETS.map((t) => (
            <RailItem key={t.id} icon={t.icon} tone={t.tone} title={t.title} subtitle={t.subtitle} onClick={() => setConnect(t)} />
          ))}
        </div>
      </RailPanel>
      <RailPanel icon={<IconTile tone="blue"><Box className="w-5 h-5" /></IconTile>} title="Compute Networks" subtitle="Where workloads can run." action={<ViewAll onClick={() => setTab('depin')} />}>
        <div className="space-y-0.5">
          <NetworkRow network={{ id: 'dh', name: 'Decentralized.Host', description: 'Your hosts under your cluster root', status: capabilities?.backend.reachable ? 'Active' : 'Unreachable', state: caps?.nodes.state ?? 'UNKNOWN', accent: '#248BFF', glyph: 'cube', nodes: nodes.length }} />
          <NetworkRow network={{ id: 'fed', name: 'Federated clusters', description: caps?.federation.detail ?? '', status: caps?.federation.state === 'LIVE' ? 'Available' : 'Not Configured', state: caps?.federation.state ?? 'UNKNOWN', accent: '#A855F7', glyph: 'F' }} onClick={() => setTab('contributions')} />
          <NetworkRow network={{ id: 'depin', name: 'External DePIN', description: 'Akash, Golem, Flux, …', status: 'Not Installed', state: 'UNAVAILABLE', accent: '#94A3B8', glyph: 'D' }} onClick={() => setTab('depin')} />
        </div>
      </RailPanel>
      <RailPanel icon={<IconTile tone="emerald"><ShieldCheck className="w-5 h-5" /></IconTile>} title="Host Sovereignty" subtitle="Policy reported by each host (not settable from here).">
        <ul className="space-y-1.5 text-[12px]">
          {[
            ['Deny images without digest', (n: NodeRec) => !!n.policy?.denyImagesWithoutDigest],
            ['Remote exec disabled', (n: NodeRec) => !n.policy?.allowExec],
            ['Federated work refused', (n: NodeRec) => !n.policy?.allowFederated],
            ['Sovereign admission', (n: NodeRec) => !!n.policy?.sovereign]
          ].map(([label, f]) => {
            const withP = nodes.filter((n) => n.policy);
            const k = withP.filter(f as (n: NodeRec) => boolean).length;
            return (
              <li key={label as string} className="flex justify-between gap-2">
                <span className="text-slate-300">{label as string}</span>
                <span className="tabular-nums text-slate-100">{withP.length ? `${k}/${withP.length}` : '—'}</span>
              </li>
            );
          })}
        </ul>
      </RailPanel>
    </>
  );

  return (
    <>
      <SurfaceLayout rail={rail}>
        <SurfaceHero
          eyebrow={<Eyebrow label="Nodes & Compute" state={stale ? 'UNKNOWN' : res.provenance?.state} />}
          title={
            <>
              Your hardware.
              <br />
              <Grad>Your network.</Grad>
              <br />
              <Grad from="#C084FC" to="#F472B6">Your workloads.</Grad>
            </>
          }
          description="Connect computers you control and use them to host applications. Each host admits work under its own policy and reports signed observations."
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
          markers={groupMarkers(groups.placed, (g) => [`${g.nodes.length} host${g.nodes.length > 1 ? 's' : ''} · ${g.worst.toLowerCase()}`])}
          arcs={meshArcs(groups.placed)}
          focusLng={groups.placed[0]?.lng ?? 40}
          frozen={stale}
          overlay={
            res.data && (
              <Glass strong className={`hidden lg:block absolute left-[41%] top-0 px-4 py-3 z-10 min-w-[190px] ${stale ? 'grayscale opacity-60' : ''}`}>
                {stale && <div className="text-[10.5px] font-bold tracking-wider text-amber-300 mb-1.5">LAST OBSERVED — NOT CURRENT</div>}
                <Legend
                  items={[
                    { label: 'Healthy', value: counts.healthy, tone: 'emerald' },
                    { label: 'Stale', value: counts.degraded, tone: 'amber' },
                    { label: 'Offline', value: counts.offline, tone: 'rose' },
                    { label: 'Never observed', value: counts.unknown, tone: 'slate' }
                  ]}
                />
              </Glass>
            )
          }
        />

        <PageTabs tabs={TABS} active={tab} onChange={setTab} />

        <Gate res={res}>
          {(d) =>
            tab === 'depin' ? (
              <div className="grid gap-3 md:grid-cols-2">
                {['Akash Network', 'Golem Network', 'Flux Network', 'io.net'].map((n) => (
                  <Unavailable key={n} title={n} detail="No adapter for this network is implemented. Nothing is installed on your hosts and no rewards are reported." />
                ))}
              </div>
            ) : tab === 'contributions' ? (
              <div className="space-y-3">
                <Unavailable title="Community contribution & metering" state="PLANNED" detail="Hosts do not offer capacity to other operators, and no usage is metered. Capacity can be granted to a peer cluster through a root-signed federation agreement (dh federation grant)." />
                <Unavailable title="Rewards" detail="No settlement exists, so there are no earnings to show." />
              </div>
            ) : tab === 'workloads' ? (
              <NodeWorkloads nodes={d.nodes} />
            ) : tab === 'activity' ? (
              <NodeActivity />
            ) : (
              <>
                {tab === 'overview' && (
                  <>
                    <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-5 gap-3">
                      <KpiTile tone="cyan" icon={<Server className="w-6 h-6" />} label="Hosts" value={d.nodes.length} sub={<><span className="text-emerald-400">{counts.healthy} fresh</span>{counts.degraded + counts.offline + counts.unknown > 0 && <> · <span className="text-rose-300">{counts.degraded + counts.offline + counts.unknown} not fresh</span></>}</>} />
                      <KpiTile tone="blue" icon={<Cpu className="w-6 h-6" />} label="CPUs (measured)" value={observed.length ? sum((n) => n.facts!.cpus) : '—'} sub={`${(d.metrics.cpuDeclaredMilli.value ?? 0) / 1000} cores declared`} />
                      <KpiTile tone="violet" icon={<MemoryStick className="w-6 h-6" />} label="Memory (measured)" value={memMeasured.length ? fmtBytes(memMeasured.reduce((a, n) => a + (n.facts!.memBytes ?? 0), 0)) : '—'} sub={`${memMeasured.length}/${nodes.length} hosts measured · ${fmtBytes(d.metrics.memDeclaredBytes.value)} declared`} />
                      <KpiTile tone="amber" icon={<HardDrive className="w-6 h-6" />} label="Storage used / quota" value={fmtBytes(d.metrics.storageUsedBytes.value)} sub={`of ${fmtBytes(d.metrics.storageCapacityBytes.value)} quota`} />
                      <KpiTile tone="emerald" icon={<Layers className="w-6 h-6" />} label="Workloads observed" value={nodes.some((n) => n.workloads !== null) ? nodes.reduce((a, n) => a + (n.workloads ?? 0), 0) : '—'} sub="from fresh + stale observations" />
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)] gap-4">
                      <Glass className="p-4 overflow-hidden">
                        <PanelHeader icon={<IconTile tone="cyan" size="sm"><Radio className="w-4 h-4" /></IconTile>} title="Global Infrastructure Map" subtitle="Hosts grouped by enrolled region; arcs join regions with a fresh host." right={<TruthTag state={capabilities?.items.geolocation.state ?? 'UNKNOWN'} title={capabilities?.items.geolocation.detail} />} />
                        <HoloGlobe
                          markers={groupMarkers(groups.placed, (g) => g.nodes.map((n) => `${n.name}: ${n.health.toLowerCase()}`), (g) => g.nodes[0] && navigate(`/nodes/${g.nodes[0].id}`))}
                          arcs={meshArcs(groups.placed)}
                          focusLng={groups.placed[0]?.lng ?? 30}
                          tilt={22}
                          speed={1.5}
                          frozen={stale}
                          center={[0.5, 0.64]}
                          radius={0.6}
                          resolution={1.3}
                          className="h-[300px] -mx-4 mt-1"
                        />
                        <NoGeoNote unplaced={groups.unplaced} />
                      </Glass>
                      <Glass className="p-4">
                        <PanelHeader title="Mesh & Control Plane" subtitle="WireGuard mesh and Raft membership as observed." />
                        <ul className="mt-3 space-y-2.5 text-[12.5px]">
                          <li className="flex justify-between"><span className="text-slate-400">Quorum</span><span className="text-slate-100 text-right">{d.cluster.quorum}</span></li>
                          <li className="flex justify-between"><span className="text-slate-400">Fault tolerance</span><span className="text-slate-100 text-right">{d.cluster.ha}</span></li>
                          <li className="flex justify-between gap-3"><span className="text-slate-400">Durability</span><span className="text-slate-100 text-right">{d.cluster.durability}</span></li>
                          <li className="flex justify-between"><span className="text-slate-400">Served by</span><span className="text-slate-100 font-mono">{d.cluster.servedBy.member.slice(0, 12)} ({d.cluster.servedBy.state})</span></li>
                          <li className="flex justify-between"><span className="text-slate-400">Mesh peers alive</span><span className="text-slate-100 tabular-nums">{nodes.some((n) => n.mesh) ? `${nodes.reduce((a, n) => a + (n.mesh?.peersAlive ?? 0), 0)} / ${nodes.reduce((a, n) => a + (n.mesh?.peers ?? 0), 0)}` : '—'}</span></li>
                          <li className="flex justify-between"><span className="text-slate-400">Handshakes &lt; 3 min</span><span className="text-slate-100 tabular-nums">{nodes.some((n) => n.mesh) ? nodes.reduce((a, n) => a + (n.mesh?.handshakesRecent ?? 0), 0) : '—'}</span></li>
                          <li className="flex justify-between"><span className="text-slate-400">Pending approval</span><span className="text-slate-100 tabular-nums">{counts.pending}</span></li>
                        </ul>
                      </Glass>
                    </div>
                  </>
                )}

                {d.nodes.length === 0 ? (
                  <Empty title="No hosts enrolled" detail="Create a join invite with dh node invite and start dh-noded on a machine you control." action={<PrimaryButton onClick={() => setConnect(ADD_TARGETS[0])}><Plus className="w-4 h-4" /> Add Your Machine</PrimaryButton>} />
                ) : (
                  <Glass className="overflow-hidden">
                    <TableToolbar search={search} onSearch={setSearch} placeholder="Search hosts by name, region, or ID..." onSort={() => setSortDesc(!sortDesc)}>
                      <FilterChips<Chip>
                        chips={[
                          { id: 'all', label: 'All Nodes', count: nodes.length },
                          { id: 'healthy', label: 'Healthy', count: counts.healthy, tone: 'emerald' },
                          { id: 'degraded', label: 'Stale', count: counts.degraded, tone: 'amber' },
                          { id: 'offline', label: 'Offline', count: counts.offline, tone: 'rose' },
                          { id: 'unknown', label: 'Unknown', count: counts.unknown, tone: 'slate' },
                          { id: 'edge', label: 'Edge', count: counts.edge, tone: 'violet' }
                        ]}
                        active={chip}
                        onChange={setChip}
                      />
                    </TableToolbar>
                    <div className="overflow-x-auto">
                      <table className="dh-table w-full min-w-[920px]">
                        <thead>
                          <tr>
                            <th>Node</th>
                            <th>Health</th>
                            <th>Lifecycle</th>
                            <th>Observation</th>
                            <th>Resources (measured)</th>
                            <th>Region</th>
                            <th>Workloads</th>
                            <th>Storage</th>
                          </tr>
                        </thead>
                        <tbody>
                          {rows.map((n) => (
                            <tr key={n.id}>
                              <td>
                                <Link to={`/nodes/${n.id}`} className="flex items-center gap-2.5 group">
                                  <IconTile tone={n.isEdge ? 'violet' : 'blue'} size="sm">{n.isEdge ? <Radio className="w-4 h-4" /> : <Server className="w-4 h-4" />}</IconTile>
                                  <div>
                                    <div className="font-semibold text-slate-100 group-hover:text-cyan-300">{n.name}</div>
                                    <div className="text-[11px] text-slate-500 font-mono">{n.id.slice(0, 14)}…</div>
                                  </div>
                                </Link>
                              </td>
                              <td title={n.healthReason}><StatusPill status={n.health} /></td>
                              <td><StatusPill status={n.lifecycle} dot={false} /></td>
                              <td><FreshnessPill f={n.observation.freshness} ageMs={n.observation.ageMs} /></td>
                              <td className="tabular-nums text-slate-300">{n.facts ? `${n.facts.cpus} CPU · ${n.facts.memBytes === null ? 'memory not measured' : fmtBytes(n.facts.memBytes)} · ${n.arch}` : '—'}</td>
                              <td>
                                <div className="text-slate-200">{n.region || '—'}</div>
                                <div className="text-[11px] text-slate-500">{n.host}</div>
                              </td>
                              <td className="tabular-nums">{n.workloads ?? '—'}</td>
                              <td className="tabular-nums text-slate-300">{n.storage ? `${fmtBytes(n.storage.usedBytes)} / ${fmtBytes(n.storage.quotaBytes)}` : '—'}</td>
                            </tr>
                          ))}
                          {rows.length === 0 && (
                            <tr>
                              <td colSpan={8} className="text-center text-slate-500 py-8">No nodes match this filter.</td>
                            </tr>
                          )}
                        </tbody>
                      </table>
                    </div>
                    <div className="px-4 py-2.5 text-[11px] text-slate-500 border-t border-[rgba(125,190,255,0.08)]">
                      Health follows the control plane: fresh signed observation within its fresh window = healthy; older = stale; no observation within LostAfter = offline. View generated {since(d.cluster ? res.provenance?.observedAt : null)}.
                    </div>
                  </Glass>
                )}
              </>
            )
          }
        </Gate>
      </SurfaceLayout>
      <ConnectDialog target={connect} onClose={() => { setConnect(null); if (params.get('add')) setParams((p) => (p.delete('add'), p), { replace: true }); }} />
    </>
  );
}

function NodeWorkloads({ nodes }: { nodes: NodeRec[] }) {
  const apps = useResource<{ apps: import('../../types/reality').AppRec[] }>('/apps');
  const byNode = new Map(nodes.map((n) => [n.id, n]));
  return (
    <Gate res={apps}>
      {(d) => {
        const rows = d.apps.flatMap((a) => a.replicas.map((r) => ({ app: a.name, ...r })));
        if (!rows.length) return <Empty title="No workloads assigned" detail="Deploy an application to place replicas on your hosts." />;
        return (
          <Glass className="overflow-x-auto">
            <table className="dh-table w-full min-w-[820px]">
              <thead>
                <tr>
                  <th>Assignment</th>
                  <th>Host</th>
                  <th>Desired</th>
                  <th>Admitted</th>
                  <th>Observed</th>
                  <th>Freshness</th>
                  <th>Origin</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.assignment + r.node}>
                    <td><Link className="text-cyan-300 hover:underline" to={`/apps/${encodeURIComponent(r.app)}`}>{r.assignment}</Link> <span className="text-slate-500">gen {r.desiredGen}</span></td>
                    <td><Link className="hover:text-cyan-300" to={`/nodes/${r.node}`}>{r.nodeName}</Link> {byNode.get(r.node)?.health !== 'HEALTHY' && <StatusPill status={byNode.get(r.node)?.health ?? 'UNKNOWN'} />}</td>
                    <td><StatusPill status={r.desired} dot={false} /></td>
                    <td title={r.reason}><StatusPill status={r.admitted} dot={false} /></td>
                    <td><StatusPill status={r.observed} dot={false} /></td>
                    <td><FreshnessPill f={r.freshness} /></td>
                    <td><StatusPill status="OWNER" tone="cyan" dot={false} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Glass>
        );
      }}
    </Gate>
  );
}

function NodeActivity() {
  const res = useResource<{ entries: import('../../types/reality').AuditEntryRec[] }>('/audit?limit=100', { pollMs: 10_000 });
  return (
    <Gate res={res}>
      {(d) => {
        const rows = d.entries.filter((e) => e.resource.startsWith('node/') || e.source === 'host' || /node|host|drain|approve|revoke|enroll/.test(e.action));
        if (!rows.length) return <Empty title="No node activity in the last 100 audit entries" />;
        return (
          <Glass className="p-4">
            <ul className="divide-y divide-[rgba(125,190,255,0.08)]">
              {rows.map((e) => (
                <li key={e.seq} className="py-2.5 flex items-start gap-3">
                  <span className="font-mono text-[11px] text-slate-500 w-12">#{e.seq}</span>
                  <div className="flex-1 min-w-0">
                    <div className="text-[13px] text-slate-100">{e.action} <span className="text-slate-400">{e.resource}</span></div>
                    <div className="text-[11.5px] text-slate-400 break-words">{e.detail}</div>
                    <div className="text-[10.5px] text-slate-500">actor {e.actor}</div>
                  </div>
                  <span className="text-[11px] text-slate-500 whitespace-nowrap">{since(e.ts)}</span>
                </li>
              ))}
            </ul>
          </Glass>
        );
      }}
    </Gate>
  );
}
