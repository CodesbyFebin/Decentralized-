import React, { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Bot, Send, User, FileText, Server, Rocket, ShieldCheck, Database, Activity, Settings2, Lock, Check, X } from 'lucide-react';
import { api, ApiError, type MutationResult } from '../../lib/client';
import { useSession } from '../../lib/session';
import { Glass, IconTile, StatusPill, GhostButton, PrimaryButton, PanelHeader } from '../common/ui';
import { ErrorState, TruthTag, since } from '../common/states';
import type { TruthState } from '../../types/reality';

interface Citation {
  type: string;
  resourceId: string;
  label: string;
  observedAt: number | null;
  href: string | null;
}
interface Proposed {
  id: string;
  kind: string;
  risk: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  title: string;
  description: string;
  target: string;
  confirmText: string | null;
  expiresAt: number;
}
interface Answer {
  id: string;
  content: string;
  grounded: boolean;
  state: TruthState;
  citations: Citation[];
  proposedAction: Proposed | null;
  model: string;
}
type Msg = { role: 'user'; text: string } | { role: 'assistant'; answer: Answer } | { role: 'error'; error: ApiError };

const CITE_ICON: Record<string, React.ReactNode> = {
  NODE_OBSERVATION: <Server className="w-3 h-3" />,
  DEPLOYMENT: <Rocket className="w-3 h-3" />,
  EVIDENCE: <ShieldCheck className="w-3 h-3" />,
  EVENT: <Activity className="w-3 h-3" />,
  VOLUME: <Database className="w-3 h-3" />,
  CERTIFICATE: <Lock className="w-3 h-3" />,
  CONFIGURATION: <Settings2 className="w-3 h-3" />,
  DOCUMENTATION: <FileText className="w-3 h-3" />
};

const MODES = [
  { id: 'ask', label: 'Ask', example: 'How is my cluster doing?' },
  { id: 'diagnose', label: 'Diagnose', example: "What's wrong right now?" },
  { id: 'plan', label: 'Plan', example: 'How do I join a host?' },
  { id: 'act', label: 'Act', example: 'drain host-b' }
];

function ActionCard({ a, onDone }: { a: Proposed; onDone: (r: MutationResult | null, dismissed?: boolean) => void }) {
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const approve = async () => {
    setBusy(true);
    setErr(null);
    try {
      const r = await api.post<{ data: { result: MutationResult } }>(`/copilot/actions/${a.id}/approve`, { confirm });
      onDone(r.data.result);
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setBusy(false);
    }
  };
  const dismiss = async () => {
    await api.post(`/copilot/actions/${a.id}/dismiss`).catch(() => {});
    onDone(null, true);
  };
  const tone = a.risk === 'CRITICAL' ? 'rose' : a.risk === 'HIGH' ? 'amber' : a.risk === 'MEDIUM' ? 'blue' : 'emerald';
  return (
    <div className="mt-3 rounded-xl border border-cyan-400/30 bg-cyan-500/5 p-3">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-[13px] font-semibold text-white">{a.title}</span>
        <StatusPill status={`${a.risk} RISK`} tone={tone} dot={false} />
        <StatusPill status="PENDING APPROVAL" tone="amber" dot={false} />
      </div>
      <p className="mt-1 text-[12px] text-slate-300">{a.description}</p>
      <p className="text-[11px] text-slate-500">Executed through the control plane with your capability; audited under your actor. Expires {since(a.expiresAt).replace(' ago', '')}.</p>
      {a.confirmText && (
        <input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder={`type ${a.confirmText} to confirm`} className="mt-2 w-full rounded-lg bg-black/30 border border-amber-400/40 px-2 py-1.5 text-[12px] font-mono" />
      )}
      <div className="mt-2 flex gap-2">
        <GhostButton onClick={approve} disabled={busy || (!!a.confirmText && confirm !== a.confirmText)} className="!border-emerald-400/40 disabled:opacity-40"><Check className="w-4 h-4" /> {busy ? 'Executing…' : 'Approve'}</GhostButton>
        <GhostButton onClick={dismiss} disabled={busy}><X className="w-4 h-4" /> Dismiss</GhostButton>
      </div>
      {err && <div className="mt-2"><ErrorState error={err} /></div>}
    </div>
  );
}

export default function CopilotView() {
  const [params] = useSearchParams();
  const [msgs, setMsgs] = useState<Msg[]>([]);
  const [input, setInput] = useState('');
  const [busy, setBusy] = useState(false);
  const [results, setResults] = useState<Record<string, MutationResult | 'dismissed'>>({});
  const { capabilities, session } = useSession();
  const bottom = useRef<HTMLDivElement>(null);
  const asked = useRef(false);

  const ask = async (q: string) => {
    if (!q.trim()) return;
    setMsgs((m) => [...m, { role: 'user', text: q }]);
    setInput('');
    setBusy(true);
    try {
      const r = await api.post<{ data: Answer }>('/copilot/query', { query: q });
      setMsgs((m) => [...m, { role: 'assistant', answer: r.data }]);
    } catch (e) {
      setMsgs((m) => [...m, { role: 'error', error: e as ApiError }]);
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    const q = params.get('q');
    if (q && !asked.current) {
      asked.current = true;
      ask(q);
    }
  }, [params]);
  useEffect(() => bottom.current?.scrollIntoView({ behavior: 'smooth', block: 'end' }), [msgs]);

  const cap = capabilities?.items.copilot;
  return (
    <div className="pt-2 grid xl:grid-cols-[minmax(0,1fr)_300px] gap-4">
      <div className="space-y-3">
        <div className="flex items-center gap-3">
          <IconTile tone="cyan" size="lg"><Bot className="w-6 h-6" /></IconTile>
          <div>
            <h1 className="text-2xl font-extrabold text-white tracking-tight">RAG Copilot</h1>
            <p className="text-[12.5px] text-slate-400">Grounded in the platform state your capability can read, plus the repository runbooks. Unknown stays unknown.</p>
          </div>
          {cap && <span className="ml-auto"><TruthTag state={cap.state} title={cap.detail} /></span>}
        </div>

        <Glass className="p-4 min-h-[420px] space-y-4">
          {msgs.length === 0 && (
            <div className="grid sm:grid-cols-2 gap-2">
              {MODES.map((m) => (
                <button key={m.id} onClick={() => ask(m.example)} className="text-left p-3 rounded-xl bg-white/[0.03] border border-[rgba(125,190,255,0.1)] hover:border-cyan-400/40">
                  <div className="text-[12.5px] font-semibold text-slate-100">{m.label}</div>
                  <div className="text-[11.5px] text-slate-400">“{m.example}”</div>
                </button>
              ))}
            </div>
          )}
          {msgs.map((m, i) =>
            m.role === 'user' ? (
              <div key={i} className="flex gap-2 justify-end">
                <div className="max-w-[80%] rounded-2xl bg-blue-600/30 border border-blue-400/30 px-3.5 py-2 text-[13px] text-white">{m.text}</div>
                <User className="w-5 h-5 text-slate-400 mt-1.5" />
              </div>
            ) : m.role === 'error' ? (
              <ErrorState key={i} error={m.error} />
            ) : (
              <div key={i} className="flex gap-2">
                <Bot className="w-5 h-5 text-cyan-300 mt-1.5 shrink-0" />
                <div className="flex-1 min-w-0 rounded-2xl bg-white/[0.03] border border-[rgba(125,190,255,0.12)] px-3.5 py-2.5">
                  <div className="flex items-center gap-2 mb-1">
                    <TruthTag state={m.answer.state} />
                    {!m.answer.grounded && <span className="text-[11px] text-amber-300">not grounded in platform data</span>}
                    <span className="text-[10.5px] text-slate-500">{m.answer.model}</span>
                  </div>
                  <div className="text-[13px] text-slate-100 whitespace-pre-wrap break-words">{m.answer.content}</div>
                  {m.answer.citations.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {m.answer.citations.map((c, j) => {
                        const body = (
                          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-cyan-500/10 border border-cyan-400/25 text-[10.5px] text-cyan-100" title={`${c.type} · ${c.resourceId}${c.observedAt ? ` · observed ${since(c.observedAt)}` : ''}`}>
                            {CITE_ICON[c.type]} {c.label}
                          </span>
                        );
                        return c.href ? <Link key={j} to={c.href}>{body}</Link> : <span key={j}>{body}</span>;
                      })}
                    </div>
                  )}
                  {m.answer.proposedAction && !results[m.answer.proposedAction.id] && (
                    <ActionCard a={m.answer.proposedAction} onDone={(r, dismissed) => setResults((x) => ({ ...x, [m.answer.proposedAction!.id]: dismissed ? 'dismissed' : (r as MutationResult) }))} />
                  )}
                  {m.answer.proposedAction && results[m.answer.proposedAction.id] && (
                    <div className="mt-2 text-[12px] text-slate-300">
                      {results[m.answer.proposedAction.id] === 'dismissed'
                        ? 'Dismissed. Nothing was executed.'
                        : (() => {
                            const r = results[m.answer.proposedAction!.id] as MutationResult;
                            return <span className={r.ok ? 'text-emerald-300' : 'text-rose-300'}>Control plane {r.ok ? 'committed' : 'refused'}: {r.message || r.code} · request {r.request_id}</span>;
                          })()}
                    </div>
                  )}
                </div>
              </div>
            )
          )}
          {busy && <div className="text-[12px] text-slate-400 animate-pulse">Retrieving from the control plane…</div>}
          <div ref={bottom} />
        </Glass>

        <form onSubmit={(e) => { e.preventDefault(); ask(input); }} className="flex gap-2">
          <input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Ask about hosts, apps, certificates, storage, the audit ledger — or 'drain host-b', 'scale web to 2'" className="flex-1 rounded-xl bg-[rgba(8,20,44,0.7)] border border-[rgba(125,190,255,0.28)] focus:border-cyan-400/60 outline-none px-4 py-3 text-[13.5px]" aria-label="Message" />
          <PrimaryButton type="submit" disabled={busy || !input.trim()}><Send className="w-4 h-4" /></PrimaryButton>
        </form>
      </div>

      <div className="space-y-3">
        <Glass className="p-4">
          <PanelHeader title="How answers are built" />
          <ol className="mt-2 space-y-1.5 text-[12px] text-slate-300 list-decimal list-inside">
            <li>Your capability fetches the view — you only retrieve what you may read.</li>
            <li>Facts are pulled from that view and the repo runbooks.</li>
            <li>Every factual line carries a citation.</li>
            <li>Actions are proposals: approval runs them through the control plane under your identity.</li>
          </ol>
        </Glass>
        <Glass className="p-4 text-[12px] text-slate-300 space-y-1.5">
          <PanelHeader title="Model" />
          <p>{cap?.detail ?? '—'}</p>
          <p className="text-slate-500">Set COPILOT_MODEL=gemini with GEMINI_API_KEY to let a model rephrase the cited facts. That sends retrieved platform data to Google, so it is off by default.</p>
          {session?.readOnly && <p className="text-amber-200">Your capability is read-only: the Copilot will not propose changes.</p>}
        </Glass>
      </div>
    </div>
  );
}
