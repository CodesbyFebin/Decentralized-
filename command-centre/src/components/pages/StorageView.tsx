import React, { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Plus, Database, HardDrive, Server, Archive, ShieldCheck, Monitor, Terminal, Settings2, Box, Copy, Layers, Package } from 'lucide-react';
import type { ArtifactRec, Freshness, Metric, NodeHealth, NodeRec, VolumeRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { SurfaceLayout, SurfaceHero, Eyebrow, regionGroups, groupMarkers, meshArcs } from '../common/CommandSurface';
import { Glass, KpiTile, PageTabs, RailPanel, RailItem, NetworkRow, PanelHeader, IconTile, StatusPill, Grad, PrimaryButton, GhostButton, Tone } from '../common/ui';
import { Gate, fmtBytes, since, shortDigest, Unavailable, Empty, FreshnessPill, metricText, Note } from '../common/states';
import { ConnectDialog, ConnectTarget } from '../common/ConnectDialog';

type Tab = 'overview' | 'mine' | 'volumes' | 'objects' | 'replication' | 'contributions' | 'depin' | 'activity';

const STORAGE_NOTE =
  'Volumes are content-addressed (BLAKE3, FastCDC chunks) and replicated across hosts; snapshots commit at quorum 2 and corrupted chunks are repaired from peers. Erasure coding is not implemented.';

const ADD_STORAGE: (ConnectTarget & { icon: React.ReactNode; tone: Tone })[] = [
  { id: 'local', title: 'Local Disk', subtitle: 'Add storage from this machine', hostName: 'this-computer', note: STORAGE_NOTE, icon: <Monitor className="w-5 h-5" />, tone: 'blue' },
  { id: 'nas', title: 'NAS / Network Storage', subtitle: 'Synology, TrueNAS, QNAP, etc.', hostName: 'nas-01', note: STORAGE_NOTE, icon: <Server className="w-5 h-5" />, tone: 'blue' },
  { id: 'linux', title: 'Linux Server', subtitle: 'Ubuntu, Debian, CentOS, etc.', hostName: 'storage-01', note: STORAGE_NOTE, icon: <Terminal className="w-5 h-5" />, tone: 'violet' },
  { id: 's3', title: 'S3 Compatible Storage', subtitle: 'MinIO, Ceph, Wasabi, Cloudflare R2', hostName: 's3', unavailable: 'The storage layer replicates volumes between hosts you enrol; there is no S3 backend adapter. Run a host on the machine that holds the disks instead.', icon: <Database className="w-5 h-5" />, tone: 'emerald' },
  { id: 'dedicated', title: 'Dedicated Storage Node', subtitle: 'Convert a server into a storage node', hostName: 'vault-01', note: STORAGE_NOTE, icon: <Archive className="w-5 h-5" />, tone: 'violet' },
  { id: 'depin', title: 'External DePIN Storage', subtitle: 'Web3 storage networks', hostName: 'depin', unavailable: 'No DePIN storage adapter is implemented. Data stays on hosts you enrol.', icon: <Box className="w-5 h-5" />, tone: 'blue' },
  { id: 'other', title: 'Other', subtitle: 'Manual installation', hostName: 'host-01', note: STORAGE_NOTE, icon: <Settings2 className="w-5 h-5" />, tone: 'slate' }
];

interface Payload {
  volumes: VolumeRec[];
  artifacts: ArtifactRec[];
  hosts: { id: string; name: string; region: string; health: NodeHealth; storage: NodeRec['storage']; freshness: Freshness; location: NodeRec['location'] }[];
  metrics: { used: Metric; capacity: Metric; volumes: Metric; degraded: Metric };
}

export default function StorageView() {
  const res = useResource<Payload>('/storage');
  const [params, setParams] = useSearchParams();
  const tab = (params.get('tab') as Tab) || 'overview';
  const setTab = (t: Tab) => setParams((p) => (t === 'overview' ? (p.delete('tab'), p) : (p.set('tab', t), p)), { replace: true });
  const [connect, setConnect] = useState<ConnectTarget | null>(params.get('add') ? ADD_STORAGE[0] : null);

  const hosts = res.data?.hosts ?? [];
  const groups = regionGroups(hosts.map((h) => ({ ...h, location: h.location } as unknown as NodeRec)));

  const rail = (
    <>
      <RailPanel icon={<IconTile tone="cyan"><Database className="w-5 h-5" /></IconTile>} title="Add Storage" subtitle="Storage joins as a host that holds volume replicas.">
        <div className="space-y-1.5">
          {ADD_STORAGE.map((t) => <RailItem key={t.id} icon={t.icon} tone={t.tone} title={t.title} subtitle={t.subtitle} onClick={() => setConnect(t)} />)}
        </div>
      </RailPanel>
      <RailPanel icon={<IconTile tone="blue"><Box className="w-5 h-5" /></IconTile>} title="Storage Networks">
        <div className="space-y-0.5">
          <NetworkRow network={{ id: 'dh', name: 'Decentralized.Host CAS', description: 'BLAKE3 content-addressed, replicated', status: 'Active', state: res.provenance?.state ?? 'UNKNOWN', accent: '#20DDF7', glyph: 'cube' }} />
          {['Filecoin', 'Arweave', 'Storj', 'Sia', 'IPFS pinning', 'Crust'].map((n) => (
            <NetworkRow key={n} network={{ id: n, name: n, description: 'no adapter implemented', status: 'Not Installed', state: 'UNAVAILABLE', accent: '#64748B', glyph: n[0] }} onClick={() => setTab('depin')} />
          ))}
        </div>
      </RailPanel>
    </>
  );

  return (
    <>
      <SurfaceLayout rail={rail}>
        <SurfaceHero
          eyebrow={<Eyebrow label="Storage & Data Infrastructure" state={res.stale ? 'UNKNOWN' : res.provenance?.state} />}
          title={<>Your disks. <Grad from="#38BDF8" to="#818CF8">Your data.</Grad><br />Your <Grad from="#A855F7" to="#F472B6">storage network.</Grad></>}
          description="Volumes live on hosts you control, replicated with content-addressed chunks and verified snapshots. Nothing is stored on a third party."
          actions={
            <>
              <PrimaryButton onClick={() => setConnect(ADD_STORAGE[0])}><Plus className="w-4 h-4" /> Add Storage</PrimaryButton>
              <GhostButton onClick={() => setTab('volumes')}><Layers className="w-4 h-4" /> Volumes</GhostButton>
              <GhostButton onClick={() => setTab('objects')}><Package className="w-4 h-4" /> Artifacts</GhostButton>
            </>
          }
          markers={groupMarkers(groups.placed, (g) => [`${g.nodes.length} host(s)`, fmtBytes(hosts.filter((h) => h.region === g.region).reduce((a, h) => a + (h.storage?.usedBytes ?? 0), 0)) + ' used'])}
          arcs={meshArcs(groups.placed)}
          focusLng={groups.placed[0]?.lng ?? 10}
          frozen={res.stale}
        />
        <PageTabs tabs={[
          { id: 'overview', label: 'Overview' }, { id: 'mine', label: 'My Storage' }, { id: 'volumes', label: 'Volumes' }, { id: 'objects', label: 'Objects & Artifacts' },
          { id: 'replication', label: 'Replication' }, { id: 'contributions', label: 'Contributions' }, { id: 'depin', label: 'DePIN Networks' }, { id: 'activity', label: 'Activity' }
        ] as { id: Tab; label: string }[]} active={tab} onChange={setTab} />

        <Gate res={res}>
          {(d) => {
            const snapshotsVerified = d.volumes.filter((v) => v.committed && v.verified >= v.durabilityReplicas).length;
            const hostTable = (
              <Glass className="overflow-x-auto">
                <table className="dh-table w-full min-w-[760px]">
                  <thead><tr><th>Host</th><th>Observation</th><th>Used by volumes</th><th>Quota</th><th>Filesystem free</th><th>Chunks</th><th>Corrupt</th></tr></thead>
                  <tbody>
                    {d.hosts.map((h) => (
                      <tr key={h.id}>
                        <td><Link to={`/nodes/${h.id}?tab=storage`} className="text-slate-100 font-semibold hover:text-cyan-300">{h.name}</Link><div className="text-[11px] text-slate-500">{h.region}</div></td>
                        <td><FreshnessPill f={h.freshness} /></td>
                        <td className="tabular-nums">{fmtBytes(h.storage?.usedBytes)}</td>
                        <td className="tabular-nums">{fmtBytes(h.storage?.quotaBytes)}</td>
                        <td className="tabular-nums">{h.storage ? `${fmtBytes(h.storage.freeBytes)} / ${fmtBytes(h.storage.capacityBytes)}` : '—'}</td>
                        <td className="tabular-nums">{h.storage?.chunks ?? '—'}</td>
                        <td className={`tabular-nums ${h.storage?.corrupt ? 'text-rose-300' : ''}`}>{h.storage?.corrupt ?? '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </Glass>
            );
            const volumeTable = d.volumes.length ? (
              <Glass className="overflow-x-auto">
                <table className="dh-table w-full min-w-[860px]">
                  <thead><tr><th>Volume</th><th>State</th><th>Size</th><th>Durability</th><th>Verified replicas</th><th>Members</th><th>Committed snapshot</th></tr></thead>
                  <tbody>
                    {d.volumes.map((v) => (
                      <tr key={v.id}>
                        <td><Link to={`/apps/${encodeURIComponent(v.app)}?tab=storage`} className="text-slate-100 font-semibold hover:text-cyan-300">{v.id}</Link></td>
                        <td title={v.detail}><StatusPill status={v.state} /></td>
                        <td className="tabular-nums">{fmtBytes(v.sizeBytes)}</td>
                        <td>{v.durabilityReplicas}× · erasure {v.erasure}</td>
                        <td className="tabular-nums">{v.verified} / {v.durabilityReplicas}</td>
                        <td className="text-slate-300">{v.memberNames.join(', ')}</td>
                        <td className="font-mono text-[11px] text-slate-400">{v.committed ? `${shortDigest(v.committed.root)} · ${since(v.committed.ts)}` : 'none'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </Glass>
            ) : <Empty title="No volumes" detail="Declare spec.volumes in a manifest to create replicated volumes." />;

            if (tab === 'depin') return <div className="grid md:grid-cols-2 gap-3">{['Filecoin', 'Arweave', 'Storj', 'Sia', 'IPFS pinning', 'Crust'].map((n) => <Unavailable key={n} title={n} detail="No adapter exists. Nothing is installed and no storage is contributed to this network." />)}</div>;
            if (tab === 'contributions') return <Unavailable title="Storage contribution" state="PLANNED" detail="Hosts store replicas only for applications in their own cluster. Offering storage to other operators and metering it is not implemented." />;
            if (tab === 'mine') return hostTable;
            if (tab === 'volumes' || tab === 'replication') return <>{volumeTable}{tab === 'replication' && <Note>{STORAGE_NOTE}</Note>}</>;
            if (tab === 'objects')
              return d.artifacts.length ? (
                <Glass className="overflow-x-auto">
                  <table className="dh-table w-full min-w-[760px]">
                    <thead><tr><th>Artifact</th><th>Digest</th><th>Size</th><th>Chunks</th><th>Attestation</th><th>Uploaded</th></tr></thead>
                    <tbody>{d.artifacts.map((a) => <tr key={a.digest}><td className="font-semibold text-slate-100">{a.name}</td><td className="font-mono text-[11px] text-slate-300 break-all whitespace-normal">{a.digest}</td><td>{fmtBytes(a.bytes)}</td><td>{a.chunks}</td><td><StatusPill status={a.attested ? 'ATTESTED' : 'UNSIGNED'} tone={a.attested ? 'emerald' : 'amber'} dot={false} /></td><td>{since(a.uploaded)}</td></tr>)}</tbody>
                  </table>
                </Glass>
              ) : <Empty title="No artifacts" detail="dh artifact push FILE --name NAME --sign" />;
            if (tab === 'activity') return <StorageActivity />;
            return (
              <>
                <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
                  <KpiTile tone="cyan" icon={<Database className="w-6 h-6" />} label="Quota (hosts)" value={fmtBytes(d.metrics.capacity.value)} sub={d.metrics.capacity.detail} />
                  <KpiTile tone="violet" icon={<HardDrive className="w-6 h-6" />} label="Used by volumes" value={fmtBytes(d.metrics.used.value)} />
                  <KpiTile tone="blue" icon={<Layers className="w-6 h-6" />} label="Volumes" value={metricText(d.metrics.volumes)} sub={`${metricText(d.metrics.degraded)} degraded`} />
                  <KpiTile tone="emerald" icon={<Copy className="w-6 h-6" />} label="Fully verified" value={`${snapshotsVerified} / ${d.volumes.length}`} sub="committed snapshot verified on every replica" />
                  <KpiTile tone="blue" icon={<Server className="w-6 h-6" />} label="Storage hosts" value={d.hosts.filter((h) => h.storage).length} sub={`${d.hosts.filter((h) => h.freshness === 'LIVE').length} fresh`} />
                  <KpiTile tone="amber" icon={<ShieldCheck className="w-6 h-6" />} label="Corrupt chunks" value={d.hosts.some((h) => h.storage) ? d.hosts.reduce((a, h) => a + (h.storage?.corrupt ?? 0), 0) : '—'} sub="reported by hosts" />
                </div>
                <div className="grid lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] gap-4">
                  <div className="space-y-2">
                    <PanelHeader title="Volumes" subtitle="Replica verification of the committed snapshot." />
                    {volumeTable}
                  </div>
                  <div className="space-y-2">
                    <PanelHeader title="Hosts" />
                    {hostTable}
                  </div>
                </div>
              </>
            );
          }}
        </Gate>
      </SurfaceLayout>
      <ConnectDialog target={connect} onClose={() => { setConnect(null); if (params.get('add')) setParams((p) => (p.delete('add'), p), { replace: true }); }} />
    </>
  );
}

function StorageActivity() {
  const res = useResource<{ entries: import('../../types/reality').AuditEntryRec[] }>('/audit?limit=150', { pollMs: 10_000 });
  return (
    <Gate res={res}>
      {(d) => {
        const rows = d.entries.filter((e) => e.resource.startsWith('volume/') || e.resource.startsWith('artifact/'));
        return rows.length ? (
          <Glass className="p-4">
            {rows.map((e) => (
              <div key={e.seq} className="py-2 border-b border-[rgba(125,190,255,0.06)] last:border-0">
                <div className="text-[12.5px] text-slate-100"><span className="font-mono text-slate-500">#{e.seq}</span> {e.action} <span className="text-slate-400">{e.resource}</span></div>
                <div className="text-[11.5px] text-slate-400">{e.detail} · {e.actor} · {since(e.ts)}</div>
              </div>
            ))}
          </Glass>
        ) : <Empty title="No storage activity in the recent audit window" />;
      }}
    </Gate>
  );
}
