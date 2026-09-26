import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Server, AppWindow, Globe, CornerDownLeft, Bot, LayoutGrid, HardDrive, Rocket, FileCheck2, Terminal } from 'lucide-react';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import type { AppRec, DomainRec, NodeRec, ValidationRecord } from '../../types/reality';

const PAGES = [
  ['Dashboard', '/'],
  ['Applications', '/apps'],
  ['Deploy', '/deploy'],
  ['New deployment', '/deploy/new'],
  ['Nodes & Compute', '/nodes'],
  ['Add node', '/nodes/add'],
  ['Storage', '/storage'],
  ['Domains', '/domains'],
  ['SSL & Security', '/security'],
  ['Analytics', '/analytics'],
  ['Evidence', '/evidence'],
  ['Activity (audit ledger)', '/activity'],
  ['Operations', '/operations'],
  ['RAG Copilot', '/copilot'],
  ['Team & session', '/team'],
  ['Settings', '/settings']
] as const;

interface Item {
  label: string;
  hint: string;
  to: string;
  icon: React.ReactNode;
}

export const CommandPalette: React.FC<{ open: boolean; onClose: () => void }> = ({ open, onClose }) => {
  const navigate = useNavigate();
  const { session, mode, can, capabilities } = useSession();
  const [q, setQ] = useState('');
  const [sel, setSel] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const signedIn = open && (mode === 'demo' || !!session?.authenticated);
  const data = useResource<{ nodes: NodeRec[]; apps: (AppRec & { lastChange: { generation: number } | null })[]; domains: DomainRec[]; volumes: { id: string; app: string; name: string }[] }>(signedIn ? '/overview' : null, { pollMs: 0 });
  const records = useResource<{ records: ValidationRecord[] }>(signedIn && capabilities?.items.validationRecords?.state === 'LIVE' ? '/evidence/records' : null, { pollMs: 0 });

  useEffect(() => {
    if (open) {
      setQ('');
      setSel(0);
      setTimeout(() => input.current?.focus(), 0);
    }
  }, [open]);

  const items = useMemo<Item[]>(() => {
    const all: Item[] = [
      ...PAGES.map(([label, to]) => ({ label, hint: 'page', to, icon: <LayoutGrid className="w-4 h-4" /> })),
      ...(data.data?.nodes ?? []).map((n) => ({ label: n.name, hint: `host · ${n.health.toLowerCase()} · ${n.region}`, to: `/nodes/${n.id}`, icon: <Server className="w-4 h-4" /> })),
      ...(data.data?.apps ?? []).map((a) => ({ label: a.name, hint: `app · ${a.phase.toLowerCase()}`, to: `/apps/${encodeURIComponent(a.name)}`, icon: <AppWindow className="w-4 h-4" /> })),
      ...(data.data?.domains ?? []).map((d) => ({ label: d.host, hint: `domain → ${d.app}`, to: '/domains', icon: <Globe className="w-4 h-4" /> })),
      ...(data.data?.apps ?? []).map((a) => ({ label: `${a.name} generation ${a.generation}`, hint: 'deployment', to: `/deploy/${encodeURIComponent(a.name)}/${a.generation}`, icon: <Rocket className="w-4 h-4" /> })),
      ...(data.data?.apps ?? []).map((a) => ({ label: `Open logs: ${a.name}`, hint: 'action', to: `/apps/${encodeURIComponent(a.name)}?tab=logs`, icon: <Terminal className="w-4 h-4" /> })),
      ...(data.data?.volumes ?? []).map((v) => ({ label: `${v.app}/${v.name}`, hint: 'volume', to: '/storage', icon: <HardDrive className="w-4 h-4" /> })),
      ...(records.data?.records ?? []).map((r) => ({ label: r.id, hint: `evidence · ${r.outcome.toLowerCase()}`, to: `/evidence/${encodeURIComponent(r.id)}`, icon: <FileCheck2 className="w-4 h-4" /> })),
      // Actions only navigate to the page that performs them; nothing runs from search.
      ...(can('api.write') ? [{ label: 'Deploy application', hint: 'action', to: '/deploy/new', icon: <Rocket className="w-4 h-4" /> }] : []),
      { label: 'Add node', hint: 'action', to: '/nodes/add', icon: <Server className="w-4 h-4" /> }
    ];
    const s = q.trim().toLowerCase();
    const hits = s ? all.filter((i) => i.label.toLowerCase().includes(s) || i.hint.includes(s)) : all.slice(0, 12);
    if (s) hits.push({ label: `Ask Copilot: "${q.trim()}"`, hint: 'copilot', to: `/copilot?q=${encodeURIComponent(q.trim())}`, icon: <Bot className="w-4 h-4" /> });
    return hits.slice(0, 14);
  }, [q, data.data, records.data, can]);

  if (!open) return null;
  const go = (i: Item) => {
    onClose();
    navigate(i.to);
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-start justify-center pt-[12vh] px-4" onClick={onClose}>
      <div className="w-full max-w-xl dh-glass-strong rounded-2xl overflow-hidden" onClick={(e) => e.stopPropagation()} role="dialog" aria-label="Command palette">
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[rgba(125,190,255,0.14)]">
          <Search className="w-4 h-4 text-cyan-300" />
          <input
            ref={input}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setSel(0);
            }}
            onKeyDown={(e) => {
              if (e.key === 'Escape') onClose();
              if (e.key === 'ArrowDown') setSel((x) => Math.min(x + 1, items.length - 1));
              if (e.key === 'ArrowUp') setSel((x) => Math.max(x - 1, 0));
              if (e.key === 'Enter' && items[sel]) go(items[sel]);
            }}
            placeholder="Search pages, hosts, apps, domains — or ask the Copilot"
            className="flex-1 bg-transparent outline-none text-[14px] text-white placeholder:text-slate-500"
            aria-label="Search"
          />
        </div>
        <ul className="max-h-[50vh] overflow-y-auto py-1" role="listbox">
          {items.map((i, idx) => (
            <li key={i.to + i.label} role="option" aria-selected={idx === sel}>
              <button onClick={() => go(i)} onMouseEnter={() => setSel(idx)} className={`w-full flex items-center gap-3 px-4 py-2 text-left ${idx === sel ? 'bg-cyan-500/10' : ''}`}>
                <span className="text-cyan-300">{i.icon}</span>
                <span className="text-[13px] text-slate-100">{i.label}</span>
                <span className="text-[11px] text-slate-500">{i.hint}</span>
                {idx === sel && <CornerDownLeft className="w-3.5 h-3.5 text-slate-500 ml-auto" />}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};
