import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { CheckCircle2, Ban, Search } from 'lucide-react';
import type { AuditEntryRec, RejectionRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/client';
import { Glass, PanelHeader, PageTabs, GhostButton, StatusPill } from '../common/ui';
import { Gate, ErrorState, Note, fmtTime, since, StaleBanner } from '../common/states';
import { Digest } from '../common/truth';

type Tab = 'ledger' | 'refused';

const SOURCES = ['all', 'operator', 'host', 'control', 'chaos', 'federation'] as const;

/** Node and app links for audit resources such as node/dh1…, app/web. */
function ResourceLink({ r }: { r: string }) {
  const [kind, name] = r.split('/', 2);
  if (kind === 'node' && name) return <Link to={`/nodes/${encodeURIComponent(name)}`} className="text-cyan-300 hover:underline">{r}</Link>;
  if (kind === 'app' && name) return <Link to={`/apps/${encodeURIComponent(name)}`} className="text-cyan-300 hover:underline">{r}</Link>;
  return <>{r}</>;
}

export default function ActivityView() {
  const [params, setParams] = useSearchParams();
  const tab = (params.get('tab') as Tab) || 'ledger';
  const [entries, setEntries] = useState<AuditEntryRec[]>([]);
  const [head, setHead] = useState<number | null>(null);
  const [nextBefore, setNextBefore] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const [fetchedAt, setFetchedAt] = useState<number | null>(null);
  const [q, setQ] = useState({ actor: '', action: '', resource: '', source: 'all' as (typeof SOURCES)[number], from: '', to: '' });

  const load = useCallback(async (before: number | null) => {
    setLoading(true);
    try {
      const r = await api.get<{ data: { head: number; entries: AuditEntryRec[]; nextBefore: number | null } }>(`/audit?limit=200${before ? `&before=${before}` : ''}`);
      setEntries((e) => (before ? [...e, ...r.data.entries] : r.data.entries));
      setHead(r.data.head);
      setNextBefore(r.data.nextBefore);
      setErr(null);
      setFetchedAt(Date.now());
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    load(null);
  }, [load]);

  const rejections = useResource<{ rejections: RejectionRec[] }>(tab === 'refused' ? '/rejections' : null, { pollMs: 10_000 });

  const rows = useMemo(() => {
    const from = q.from ? Date.parse(q.from) : null;
    const to = q.to ? Date.parse(q.to) + 86_400_000 : null;
    const has = (v: string, n: string) => !n || v.toLowerCase().includes(n.trim().toLowerCase());
    return entries.filter(
      (e) => has(e.actor, q.actor) && has(e.action, q.action) && has(e.resource, q.resource) && (q.source === 'all' || e.source === q.source) && (from === null || e.ts >= from) && (to === null || e.ts < to)
    );
  }, [entries, q]);

  const set = (k: keyof typeof q) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setQ((x) => ({ ...x, [k]: e.target.value }));

  return (
    <div className="space-y-4 pt-2">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Activity</h1>
        <p className="text-[13px] text-slate-400 max-w-3xl">Every committed change is an entry in the control plane's hash-chained audit ledger, with the actor taken from the signed capability. Refused messages (bad signatures, replays, invalid join tokens) are kept in a separate replicated log.</p>
      </div>
      <PageTabs<Tab>
        tabs={[
          { id: 'ledger', label: `Audit ledger${head !== null ? ` (head #${head})` : ''}` },
          { id: 'refused', label: 'Refused' }
        ]}
        active={tab}
        onChange={(t) => setParams((p) => (t === 'ledger' ? (p.delete('tab'), p) : (p.set('tab', t), p)), { replace: true })}
      />

      {tab === 'ledger' && (
        <>
          {err && entries.length > 0 && <StaleBanner fetchedAt={fetchedAt} error={err} />}
          <Glass className="p-4">
            <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-2" role="search" aria-label="Filter audit entries">
              <label className="text-[11.5px] text-slate-400">Actor<input value={q.actor} onChange={set('actor')} className="dh-input w-full mt-1" placeholder="operator:…, dh1…" /></label>
              <label className="text-[11.5px] text-slate-400">Action<input value={q.action} onChange={set('action')} className="dh-input w-full mt-1" placeholder="apply, node-drain…" /></label>
              <label className="text-[11.5px] text-slate-400">Resource<input value={q.resource} onChange={set('resource')} className="dh-input w-full mt-1" placeholder="app/web, node/…" /></label>
              <label className="text-[11.5px] text-slate-400">Source
                <select value={q.source} onChange={set('source')} className="dh-input w-full mt-1">
                  {SOURCES.map((x) => <option key={x} value={x}>{x}</option>)}
                </select>
              </label>
              <label className="text-[11.5px] text-slate-400">From<input type="date" value={q.from} onChange={set('from')} className="dh-input w-full mt-1" /></label>
              <label className="text-[11.5px] text-slate-400">To<input type="date" value={q.to} onChange={set('to')} className="dh-input w-full mt-1" /></label>
            </div>
            <div className="mt-2 flex items-center gap-2 text-[11.5px] text-slate-400">
              <Search className="w-3.5 h-3.5" aria-hidden />
              {rows.length} of {entries.length} loaded entries match{nextBefore ? '; load older entries to search further back' : ' (the whole ledger is loaded)'}.
            </div>
          </Glass>
          {err && entries.length === 0 ? (
            <ErrorState error={err} onRetry={() => load(null)} />
          ) : (
            <Glass className={`overflow-x-auto ${err ? 'grayscale opacity-60' : ''}`}>
              <table className="dh-table w-full min-w-[1080px]">
                <thead><tr><th>#</th><th>Time</th><th>Actor</th><th>Action</th><th>Resource</th><th>Result</th><th>Detail</th><th>Evidence</th><th>Entry hash</th></tr></thead>
                <tbody>
                  {rows.map((e) => (
                    <tr key={e.seq}>
                      <td className="font-mono text-slate-500">{e.seq}</td>
                      <td className="text-slate-300 whitespace-nowrap" title={fmtTime(e.ts)}>{since(e.ts)}</td>
                      <td className="font-mono text-[11px] text-slate-300" title={e.actor}>{e.actor.length > 26 ? `${e.actor.slice(0, 24)}…` : e.actor}<div className="text-[10.5px] text-slate-500 font-sans">{e.source}</div></td>
                      <td className="text-slate-100">{e.action}</td>
                      <td className="text-slate-300"><ResourceLink r={e.resource} /></td>
                      <td><span className="inline-flex items-center gap-1 text-[11.5px] text-emerald-300"><CheckCircle2 className="w-3.5 h-3.5" aria-hidden /> committed</span></td>
                      <td className="text-slate-400 whitespace-normal break-all max-w-[340px]">{e.detail}</td>
                      <td>{e.evidence ? <Digest value={e.evidence} chars={8} /> : <span className="text-slate-500">—</span>}</td>
                      <td title={`prev ${e.prev}`}><Digest value={e.hash} chars={8} /></td>
                    </tr>
                  ))}
                  {!rows.length && !loading && <tr><td colSpan={9} className="text-center text-slate-500 py-6">{entries.length ? 'No loaded entry matches the filters.' : 'The ledger is empty.'}</td></tr>}
                </tbody>
              </table>
              <div className="p-3 flex justify-center">
                {nextBefore ? <GhostButton onClick={() => load(nextBefore)} disabled={loading}>{loading ? 'Loading…' : 'Load older entries'}</GhostButton> : <span className="text-[11.5px] text-slate-500">{loading ? 'Loading…' : 'Beginning of the ledger.'}</span>}
              </div>
            </Glass>
          )}
          <Note>The ledger records the actor, action, resource and evidence digest of each committed change. It does not record console request ids or operation ids; a refused or failed request never becomes an entry (see Refused). Check the chain with <code className="text-cyan-200">dh audit verify</code> or on the Evidence page.</Note>
        </>
      )}

      {tab === 'refused' && (
        <Gate res={rejections}>
          {(d) =>
            d.rejections.length === 0 ? (
              <Glass className="p-6 text-center text-[13px] text-slate-400">The control plane has refused nothing recently.</Glass>
            ) : (
              <Glass className="overflow-x-auto">
                <table className="dh-table w-full min-w-[860px]">
                  <thead><tr><th>Time</th><th>Kind</th><th>Claimed sender</th><th>Result</th><th>Reason</th><th>Evidence</th></tr></thead>
                  <tbody>
                    {d.rejections.map((x, i) => (
                      <tr key={`${x.ts}-${i}`}>
                        <td className="text-slate-300 whitespace-nowrap" title={fmtTime(x.ts)}>{since(x.ts)}</td>
                        <td><StatusPill status={x.kind.toUpperCase()} tone="slate" dot={false} /></td>
                        <td className="font-mono text-[11px] text-slate-300">{x.node ?? '—'}{x.seq ? <span className="text-slate-500"> · seq {x.seq}</span> : null}</td>
                        <td><span className="inline-flex items-center gap-1 text-[11.5px] text-rose-300"><Ban className="w-3.5 h-3.5" aria-hidden /> refused</span></td>
                        <td className="text-slate-300 whitespace-normal break-all max-w-[420px]">{x.reason}</td>
                        <td>{x.evidence ? <Digest value={x.evidence} chars={8} /> : <span className="text-slate-500">—</span>}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <div className="px-4 py-2.5 text-[11px] text-slate-500 border-t border-[rgba(125,190,255,0.08)]">The newest 100, as held by the control plane. "Claimed sender" is what the message said; a refused message proves nothing about who sent it.</div>
              </Glass>
            )
          }
        </Gate>
      )}
    </div>
  );
}
