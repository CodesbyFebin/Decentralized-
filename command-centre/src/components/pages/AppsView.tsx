import React, { useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Plus, AppWindow, Globe, Layers } from 'lucide-react';
import type { AppRec, DomainRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, IconTile, StatusPill, PrimaryButton, TableToolbar, FilterChips, KpiTile } from '../common/ui';
import { Gate, Empty, shortDigest, since } from '../common/states';

type Chip = 'all' | 'running' | 'converging' | 'degraded' | 'stopped' | 'unknown';

const CHIP: Record<Exclude<Chip, 'all'>, (a: AppRec) => boolean> = {
  running: (a) => a.phase === 'READY',
  converging: (a) => a.phase === 'CONVERGING',
  degraded: (a) => a.phase === 'DEGRADED' || a.phase === 'REFUSED',
  stopped: (a) => a.phase === 'STOPPED' || a.phase === 'DELETED',
  unknown: (a) => a.phase === 'UNKNOWN'
};

export default function AppsView() {
  const res = useResource<{ apps: AppRec[]; domains: DomainRec[] }>('/apps');
  const [chip, setChip] = useState<Chip>('all');
  const [search, setSearch] = useState('');
  const navigate = useNavigate();
  const apps = res.data?.apps ?? [];
  const rows = useMemo(
    () =>
      apps
        .filter((a) => (chip === 'all' ? true : CHIP[chip](a)))
        .filter((a) => {
          const q = search.trim().toLowerCase();
          if (!q) return true;
          return a.name.includes(q) || a.hash.includes(q) || a.ingress.some((i) => i.host.includes(q)) || a.replicas.some((r) => r.nodeName.toLowerCase().includes(q) || r.node.includes(q));
        }),
    [apps, chip, search]
  );

  return (
    <div className="space-y-4 pt-2">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Applications</h1>
          <p className="text-[13px] text-slate-400">Every application with its desired, observed and verified replicas, from the control plane.</p>
        </div>
        <PrimaryButton onClick={() => navigate('/deploy/new')}>
          <Plus className="w-4 h-4" /> New Application
        </PrimaryButton>
      </div>
      <Gate res={res}>
        {(d) =>
          d.apps.length === 0 ? (
            <Empty title="No applications deployed." detail="Deploy a dh/v1 manifest to create one." action={<PrimaryButton onClick={() => navigate('/deploy/new')}><Plus className="w-4 h-4" /> New Deployment</PrimaryButton>} />
          ) : (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <KpiTile tone="cyan" icon={<AppWindow className="w-6 h-6" />} label="Applications" value={d.apps.length} sub={`${d.apps.filter((a) => a.phase === 'READY').length} ready`} />
                <KpiTile tone="blue" icon={<Layers className="w-6 h-6" />} label="Replicas desired" value={d.apps.reduce((a, x) => a + x.desiredReplicas, 0)} />
                <KpiTile tone="emerald" icon={<Layers className="w-6 h-6" />} label="Observed running (fresh)" value={d.apps.reduce((a, x) => a + x.observedRunning, 0)} sub={`${d.apps.reduce((a, x) => a + x.healthyReplicas, 0)} passing health checks`} />
                <KpiTile tone="violet" icon={<Globe className="w-6 h-6" />} label="Domains" value={d.domains.length} />
              </div>
              <Glass className="overflow-hidden">
                <TableToolbar search={search} onSearch={setSearch} placeholder="Search name, manifest hash, domain or node…">
                  <FilterChips<Chip>
                    chips={[
                      { id: 'all', label: 'All', count: apps.length },
                      { id: 'running', label: 'Running', count: apps.filter(CHIP.running).length, tone: 'emerald' },
                      { id: 'converging', label: 'Converging', count: apps.filter(CHIP.converging).length, tone: 'blue' },
                      { id: 'degraded', label: 'Degraded', count: apps.filter(CHIP.degraded).length, tone: 'amber' },
                      { id: 'stopped', label: 'Stopped', count: apps.filter(CHIP.stopped).length },
                      { id: 'unknown', label: 'Unknown', count: apps.filter(CHIP.unknown).length }
                    ]}
                    active={chip}
                    onChange={setChip}
                  />
                </TableToolbar>
                <div className="overflow-x-auto">
                  <table className="dh-table w-full min-w-[860px]">
                    <thead>
                      <tr>
                        <th>Application</th>
                        <th>Phase</th>
                        <th>Desired</th>
                        <th>Observed</th>
                        <th>Verified</th>
                        <th>Placement</th>
                        <th>Artifact</th>
                        <th>Endpoint</th>
                        <th>Updated</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((a) => {
                        const last = a.history[a.history.length - 1];
                        return (
                          <tr key={a.name}>
                            <td>
                              <Link to={`/apps/${encodeURIComponent(a.name)}`} className="flex items-center gap-2.5 group">
                                <IconTile tone="blue" size="sm"><AppWindow className="w-4 h-4" /></IconTile>
                                <div>
                                  <div className="font-semibold text-slate-100 group-hover:text-cyan-300">{a.name}</div>
                                  <div className="text-[11px] text-slate-500">generation {a.generation} · {a.runtime}</div>
                                </div>
                              </Link>
                            </td>
                            <td title={a.phaseReason}><StatusPill status={a.phase} /></td>
                            <td className="tabular-nums">{a.desiredReplicas}</td>
                            <td className="tabular-nums">{a.observedRunning}</td>
                            <td className="tabular-nums" title="running, fresh and passing the declared health check">{a.healthyReplicas}</td>
                            <td className="text-slate-300 text-[12px]">{[...new Set(a.replicas.filter((r) => r.desired === 'RUNNING').map((r) => r.nodeName))].join(', ') || '—'}</td>
                            <td className="font-mono text-[11.5px] text-slate-300" title={a.image}>{shortDigest(a.imageDigest ?? a.image)}</td>
                            <td>{a.ingress.map((i) => <div key={i.host} className="text-sky-300">{i.host}</div>)}{!a.ingress.length && <span className="text-slate-500">—</span>}</td>
                            <td className="text-slate-300">{last ? <>{last.change} · {since(last.ts)}<div className="text-[11px] text-slate-500">{last.actor}</div></> : '—'}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </Glass>
            </>
          )
        }
      </Gate>
    </div>
  );
}
