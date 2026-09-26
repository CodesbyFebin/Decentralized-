import React, { useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Plus, AppWindow, Globe, Layers } from 'lucide-react';
import type { AppRec, DomainRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, IconTile, StatusPill, PrimaryButton, TableToolbar, FilterChips, KpiTile } from '../common/ui';
import { Gate, Empty, shortDigest, since } from '../common/states';

type Chip = 'all' | 'READY' | 'CONVERGING' | 'DEGRADED' | 'other';

export default function AppsView() {
  const res = useResource<{ apps: AppRec[]; domains: DomainRec[] }>('/apps');
  const [chip, setChip] = useState<Chip>('all');
  const [search, setSearch] = useState('');
  const navigate = useNavigate();
  const apps = res.data?.apps ?? [];
  const rows = useMemo(
    () =>
      apps
        .filter((a) => (chip === 'all' ? true : chip === 'other' ? !['READY', 'CONVERGING', 'DEGRADED'].includes(a.phase) : a.phase === chip))
        .filter((a) => !search || a.name.includes(search.toLowerCase()) || a.ingress.some((i) => i.host.includes(search.toLowerCase()))),
    [apps, chip, search]
  );

  return (
    <div className="space-y-4 pt-2">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Websites & Apps</h1>
          <p className="text-[13px] text-slate-400">Every application with its desired, admitted and observed replicas.</p>
        </div>
        <PrimaryButton onClick={() => navigate('/deploy/new')}>
          <Plus className="w-4 h-4" /> New Deployment
        </PrimaryButton>
      </div>
      <Gate res={res}>
        {(d) =>
          d.apps.length === 0 ? (
            <Empty title="No applications" detail="Deploy a dh/v1 manifest to create one." action={<PrimaryButton onClick={() => navigate('/deploy/new')}><Plus className="w-4 h-4" /> New Deployment</PrimaryButton>} />
          ) : (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <KpiTile tone="cyan" icon={<AppWindow className="w-6 h-6" />} label="Applications" value={d.apps.length} sub={`${d.apps.filter((a) => a.phase === 'READY').length} ready`} />
                <KpiTile tone="blue" icon={<Layers className="w-6 h-6" />} label="Replicas desired" value={d.apps.reduce((a, x) => a + x.desiredReplicas, 0)} />
                <KpiTile tone="emerald" icon={<Layers className="w-6 h-6" />} label="Observed running (fresh)" value={d.apps.reduce((a, x) => a + x.observedRunning, 0)} sub={`${d.apps.reduce((a, x) => a + x.healthyReplicas, 0)} passing health checks`} />
                <KpiTile tone="violet" icon={<Globe className="w-6 h-6" />} label="Domains" value={d.domains.length} />
              </div>
              <Glass className="overflow-hidden">
                <TableToolbar search={search} onSearch={setSearch} placeholder="Search apps or domains...">
                  <FilterChips<Chip>
                    chips={[
                      { id: 'all', label: 'All', count: apps.length },
                      { id: 'READY', label: 'Ready', count: apps.filter((a) => a.phase === 'READY').length, tone: 'emerald' },
                      { id: 'CONVERGING', label: 'Converging', count: apps.filter((a) => a.phase === 'CONVERGING').length, tone: 'blue' },
                      { id: 'DEGRADED', label: 'Degraded', count: apps.filter((a) => a.phase === 'DEGRADED').length, tone: 'amber' },
                      { id: 'other', label: 'Other', count: apps.filter((a) => !['READY', 'CONVERGING', 'DEGRADED'].includes(a.phase)).length }
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
                        <th>Admitted</th>
                        <th>Observed</th>
                        <th>Healthy</th>
                        <th>Artifact</th>
                        <th>Domains</th>
                        <th>Last change</th>
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
                            <td className="tabular-nums">{a.admitted}</td>
                            <td className="tabular-nums">{a.observedRunning}</td>
                            <td className="tabular-nums">{a.healthyReplicas}</td>
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
