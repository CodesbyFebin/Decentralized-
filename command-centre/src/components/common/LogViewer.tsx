/**
 * Operational log viewer. The platform relays a host's recent log lines on
 * request (tail over the mesh); this polls that tail. It is not a push
 * stream, and says so. When a poll fails the lines already shown stay, under
 * a LOG STREAM DISCONNECTED banner.
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Pause, Play, Download, Search, CloudOff, Terminal } from 'lucide-react';
import { api, ApiError } from '../../lib/client';
import { fmtTime, since } from './states';

const LEVELS = ['all', 'error', 'warn', 'info'] as const;
type Level = (typeof LEVELS)[number];
const LEVEL_RE: Record<Exclude<Level, 'all'>, RegExp> = {
  error: /\b(error|err|fatal|panic)\b/i,
  warn: /\b(warn|warning)\b/i,
  info: /\b(info)\b/i
};

export const LogViewer: React.FC<{ path: string; source: string; pollMs?: number; tail?: number }> = ({ path, source, pollMs = 3000, tail = 300 }) => {
  const [lines, setLines] = useState<string[] | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [paused, setPaused] = useState(false);
  const [lastOk, setLastOk] = useState<number | null>(null);
  const [query, setQuery] = useState('');
  const [level, setLevel] = useState<Level>('all');
  const box = useRef<HTMLPreElement>(null);
  const url = `${path}${path.includes('?') ? '&' : '?'}tail=${tail}`;

  const load = useCallback(async () => {
    try {
      const r = await api.get<{ data: { lines: string[] } }>(url);
      setLines(r.data.lines);
      setError(null);
      setLastOk(Date.now());
    } catch (e) {
      setError(e as ApiError);
    }
  }, [url]);

  useEffect(() => {
    setLines(null);
    setError(null);
    load();
  }, [load]);

  useEffect(() => {
    if (paused) return;
    const t = setInterval(load, pollMs);
    return () => clearInterval(t);
  }, [paused, load, pollMs]);

  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (lines ?? []).filter((l) => (!q || l.toLowerCase().includes(q)) && (level === 'all' || LEVEL_RE[level].test(l)));
  }, [lines, query, level]);

  useEffect(() => {
    if (!paused && box.current) box.current.scrollTop = box.current.scrollHeight;
  }, [shown, paused]);

  const download = () => {
    const blob = new Blob([(lines ?? []).join('\n') + '\n'], { type: 'text/plain' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `${source.replace(/[^\w.-]+/g, '_')}.log`;
    a.click();
    URL.revokeObjectURL(a.href);
  };

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2 text-[12px]">
        <Terminal className="w-4 h-4 text-cyan-300" aria-hidden />
        <span className="text-slate-300 font-mono">{source}</span>
        <span className="text-slate-500">· tail {tail}, polled every {Math.round(pollMs / 1000)} s</span>
        <span className="ml-auto" />
        <label className="relative">
          <span className="sr-only">Search logs</span>
          <Search className="w-3.5 h-3.5 absolute left-2 top-1/2 -translate-y-1/2 text-slate-500" aria-hidden />
          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="search" className="dh-input !py-1 !pl-7 w-40" />
        </label>
        <label>
          <span className="sr-only">Severity</span>
          <select value={level} onChange={(e) => setLevel(e.target.value as Level)} className="dh-input !py-1">
            {LEVELS.map((l) => (
              <option key={l} value={l}>{l === 'all' ? 'all lines' : `${l} (text match)`}</option>
            ))}
          </select>
        </label>
        <button type="button" onClick={() => setPaused((p) => !p)} className="inline-flex items-center gap-1 px-2 py-1 rounded-lg border border-[rgba(125,190,255,0.18)] text-slate-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400" aria-pressed={paused}>
          {paused ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />} {paused ? 'Resume' : 'Pause'}
        </button>
        <button type="button" onClick={download} disabled={!lines?.length} className="inline-flex items-center gap-1 px-2 py-1 rounded-lg border border-[rgba(125,190,255,0.18)] text-slate-200 disabled:opacity-40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400">
          <Download className="w-3.5 h-3.5" /> Download
        </button>
      </div>
      {error && (
        <div role="status" className="flex items-center gap-2 px-3 py-2 rounded-xl bg-rose-500/10 border border-rose-400/30 text-[12px] text-rose-100">
          <CloudOff className="w-4 h-4" aria-hidden />
          <b>LOG STREAM DISCONNECTED</b>
          <span className="text-rose-200/80">{error.message}{lastOk ? ` · last lines received ${since(lastOk)} (${fmtTime(lastOk)})` : ''}</span>
        </div>
      )}
      <pre
        ref={box}
        tabIndex={0}
        aria-label={`Log lines from ${source}`}
        className={`max-h-[480px] min-h-[160px] overflow-auto rounded-xl bg-[#020611]/95 border border-[rgba(125,190,255,0.12)] p-3 text-[12px] leading-relaxed text-slate-200 whitespace-pre-wrap focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 ${error ? 'opacity-70' : ''}`}
        style={{ fontFamily: 'var(--font-mono)' }}
      >
        {lines === null ? (error ? '' : 'Loading…') : shown.length ? shown.join('\n') : lines.length ? '(no line matches the filter)' : '(no output)'}
      </pre>
      {lines !== null && <div className="text-[11px] text-slate-500">{shown.length} of {lines.length} line(s){lastOk ? ` · received ${since(lastOk)}` : ''}{paused ? ' · paused' : ''}</div>}
    </div>
  );
};
