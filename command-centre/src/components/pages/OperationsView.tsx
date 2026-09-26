import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { Clock, Loader, PauseCircle, CheckCircle2, XCircle, MinusCircle } from 'lucide-react';
import type { OperationRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { Glass, FilterChips, TONE, type Tone } from '../common/ui';
import { Gate, Note, TruthTag, fmtTime, since } from '../common/states';

const STATUS: Record<OperationRec['status'], { tone: Tone; icon: React.ReactNode }> = {
  QUEUED: { tone: 'slate', icon: <Clock className="w-4 h-4" aria-hidden /> },
  RUNNING: { tone: 'blue', icon: <Loader className="w-4 h-4" aria-hidden /> },
  WAITING: { tone: 'amber', icon: <PauseCircle className="w-4 h-4" aria-hidden /> },
  SUCCEEDED: { tone: 'emerald', icon: <CheckCircle2 className="w-4 h-4" aria-hidden /> },
  FAILED: { tone: 'rose', icon: <XCircle className="w-4 h-4" aria-hidden /> },
  CANCELLED: { tone: 'slate', icon: <MinusCircle className="w-4 h-4" aria-hidden /> }
};

const StatusBadge: React.FC<{ s: OperationRec['status'] }> = ({ s }) => {
  const c = TONE[STATUS[s].tone];
  return (
    <span className="inline-flex items-center gap-1 px-2 py-[2px] rounded-md text-[11.5px] font-bold border" style={{ color: c, borderColor: `${c}40`, background: `${c}14` }}>
      {STATUS[s].icon}
      {s === 'CANCELLED' ? 'SUPERSEDED' : s}
    </span>
  );
};

type Chip = 'active' | 'all' | 'FAILED';

export default function OperationsView() {
  const res = useResource<{ operations: OperationRec[] }>('/operations', { pollMs: 5_000 });
  const [chip, setChip] = useState<Chip>('active');
  return (
    <div className="space-y-4 pt-2">
      <div>
        <div className="flex items-center gap-2">
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Operations</h1>
          <TruthTag state="DERIVED" title="derived from control-plane state; the platform has no Operation resource" />
        </div>
        <p className="text-[13px] text-slate-400 max-w-3xl">Work that takes time after the control plane commits it: deployments converging, hosts draining, volumes replicating. The platform commits changes synchronously and has no operation record, so each item here is read from current state, with its source. No percentages are estimated.</p>
      </div>
      <Gate res={res}>
        {(d, stale) => {
          const active = d.operations.filter((o) => o.status === 'RUNNING' || o.status === 'QUEUED' || o.status === 'WAITING');
          const rows = chip === 'active' ? active : chip === 'FAILED' ? d.operations.filter((o) => o.status === 'FAILED') : d.operations;
          return (
            <>
              <FilterChips<Chip>
                chips={[
                  { id: 'active', label: 'In progress', count: active.length, tone: 'blue' },
                  { id: 'FAILED', label: 'Failed', count: d.operations.filter((o) => o.status === 'FAILED').length, tone: 'rose' },
                  { id: 'all', label: 'All recent', count: d.operations.length }
                ]}
                active={chip}
                onChange={setChip}
              />
              {rows.length === 0 ? (
                <Glass className="p-6 text-center text-[13px] text-slate-400">{chip === 'active' ? 'Nothing is in progress.' : chip === 'FAILED' ? 'No failed operation in the current state.' : 'No operations.'}</Glass>
              ) : (
                <ul className="space-y-3">
                  {rows.map((o) => (
                    <li key={o.id}>
                      <Glass className={`p-4 ${o.status === 'FAILED' ? 'border border-rose-400/35' : ''}`}>
                        <div className="flex flex-wrap items-center gap-2">
                          <StatusBadge s={o.status} />
                          {stale && <span className="text-[11px] font-semibold text-amber-300">LAST OBSERVED, NOT CURRENT</span>}
                          <Link to={o.targetHref} className="text-[14px] font-semibold text-slate-100 hover:text-cyan-300">{o.title}</Link>
                          <span className="text-[11px] text-slate-500 font-mono">{o.id}</span>
                        </div>
                        <div className="mt-1 text-[12px] text-slate-400">
                          {o.detail} · actor {o.actor ?? 'unknown'} · started {o.startedAt ? <span title={fmtTime(o.startedAt)}>{since(o.startedAt)}</span> : 'unknown'} · updated {o.updatedAt ? <span title={fmtTime(o.updatedAt)}>{since(o.updatedAt)}</span> : 'unknown'}
                        </div>
                        {o.error && <div className="mt-1 text-[12.5px] text-rose-200">Error: {o.error}</div>}
                        <ol className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11.5px]">
                          {o.steps.map((s) => (
                            <li key={s.label} className={s.state === 'FAILED' ? 'text-rose-300' : s.state === 'PASSED' ? 'text-emerald-300' : s.state === 'RUNNING' ? 'text-blue-300' : 'text-slate-500'}>
                              {s.state === 'PASSED' ? '✓' : s.state === 'FAILED' ? '✗' : s.state === 'RUNNING' ? '…' : '·'} {s.label} <span className="text-slate-500">({s.detail})</span>
                            </li>
                          ))}
                        </ol>
                        <div className="mt-1 text-[10.5px] text-slate-500">source: {o.source}</div>
                      </Glass>
                    </li>
                  ))}
                </ul>
              )}
              <Note>Operations the control plane refuses never start and are not listed here; see Activity → Refused. Adapter installation and other long-running kinds do not exist yet.</Note>
            </>
          );
        }}
      </Gate>
    </div>
  );
}
