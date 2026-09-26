import { useCallback, useEffect, useRef, useState } from 'react';
import { api, ApiError } from './client';
import type { Envelope, Provenance } from '../types/reality';

export interface Resource<T> {
  data: T | null;
  provenance: Provenance | null;
  error: ApiError | null;
  loading: boolean;
  /** Data is being shown after a failed refresh: it is the last observation, not current. */
  stale: boolean;
  fetchedAt: number | null;
  refresh: () => void;
}

/* Share in-flight GETs between components mounting at the same time. */
const inflight = new Map<string, Promise<unknown>>();

function shared<T>(path: string, signal: AbortSignal): Promise<T> {
  const hit = inflight.get(path) as Promise<T> | undefined;
  if (hit) return hit;
  const p = api.get<T>(path, signal).finally(() => inflight.delete(path));
  inflight.set(path, p);
  return p;
}

/**
 * Poll a BFF resource. Keeps the last good payload on failure but marks it
 * stale so views can drop confidence instead of freezing green.
 */
export function useResource<T>(path: string | null, opts: { pollMs?: number } = {}): Resource<T> {
  const { pollMs = 5000 } = opts;
  const [state, setState] = useState<Omit<Resource<T>, 'refresh'>>({ data: null, provenance: null, error: null, loading: !!path, stale: false, fetchedAt: null });
  const [tick, setTick] = useState(0);
  const refresh = useCallback(() => setTick((t) => t + 1), []);
  const pathRef = useRef(path);
  pathRef.current = path;

  useEffect(() => {
    if (!path) return;
    let alive = true;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let ac: AbortController | null = null;

    const run = async () => {
      ac = new AbortController();
      try {
        const res = await shared<Envelope<T> | { data: T; request_id: string }>(path, ac.signal);
        if (!alive) return;
        setState({
          data: res.data,
          provenance: 'provenance' in res ? res.provenance : null,
          error: null,
          loading: false,
          stale: false,
          fetchedAt: Date.now()
        });
      } catch (e) {
        if (!alive || (e as Error).name === 'AbortError') return;
        const err = e instanceof ApiError ? e : new ApiError(0, 'NETWORK', String(e), null);
        setState((s) => ({ ...s, error: err, loading: false, stale: s.data !== null }));
      }
      if (alive && pollMs > 0 && document.visibilityState !== 'hidden') timer = setTimeout(run, pollMs);
      else if (alive && pollMs > 0) timer = setTimeout(run, pollMs * 3);
    };
    setState((s) => ({ ...s, loading: s.data === null }));
    run();
    return () => {
      alive = false;
      if (timer) clearTimeout(timer);
      ac?.abort();
    };
  }, [path, pollMs, tick]);

  return { ...state, refresh };
}
