import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { api, ApiError, setUnauthenticatedHandler } from './client';
import type { AdapterMode, Capabilities, Session } from '../types/reality';

interface SessionState {
  session: Session | null;
  mode: AdapterMode | null;
  capabilities: Capabilities | null;
  loading: boolean;
  signIn: (token: string) => Promise<void>;
  signOut: () => Promise<void>;
  refreshCapabilities: () => void;
  can: (action: 'api.read' | 'api.write' | 'api.admin') => boolean;
}

const Ctx = createContext<SessionState | null>(null);

export const useSession = () => {
  const v = useContext(Ctx);
  if (!v) throw new Error('useSession outside SessionProvider');
  return v;
};

export const SessionProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [session, setSession] = useState<Session | null>(null);
  const [mode, setMode] = useState<AdapterMode | null>(null);
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [loading, setLoading] = useState(true);
  const [capTick, setCapTick] = useState(0);

  const load = useCallback(async () => {
    try {
      const r = await api.get<{ data: Session; mode: AdapterMode }>('/session');
      setSession(r.data);
      setMode(r.mode);
    } catch {
      setSession({ authenticated: false, actor: null, actions: [], cluster: null, expiresAt: null, readOnly: true });
    } finally {
      setLoading(false);
    }
  }, []);

  const signIn = useCallback(async (token: string) => {
    const r = await api.post<{ data: Session }>('/session', { token });
    setSession(r.data);
    setCapTick((t) => t + 1);
  }, []);

  const signOut = useCallback(async () => {
    await api.del('/session').catch(() => {});
    setSession({ authenticated: false, actor: null, actions: [], cluster: null, expiresAt: null, readOnly: true });
  }, []);

  useEffect(() => {
    setUnauthenticatedHandler(() => setSession((s) => (s ? { ...s, authenticated: false } : s)));
    // `dh console` prints a URL with the capability in the fragment; exchange it
    // for an HttpOnly cookie and remove it from the address bar.
    const m = /[#&]token=(dhcap1\.[A-Za-z0-9_-]+)/.exec(window.location.hash);
    if (m) {
      history.replaceState(null, '', window.location.pathname + window.location.search);
      signIn(m[1])
        .catch(() => {})
        .finally(load);
    } else {
      load();
    }
  }, [load, signIn]);

  useEffect(() => {
    let alive = true;
    const run = () =>
      api
        .get<{ data: Capabilities }>('/capabilities')
        .then((r) => alive && setCapabilities(r.data))
        .catch((e: ApiError) => {
          if (!alive) return;
          setCapabilities((c) => (c ? { ...c, backend: { ...c.backend, reachable: false, state: 'UNAVAILABLE', detail: e.message, checkedAt: Date.now() } } : c));
        });
    run();
    const t = setInterval(run, 15_000);
    return () => {
      alive = false;
      clearInterval(t);
    };
  }, [capTick, session?.authenticated]);

  const can = useCallback(
    (action: 'api.read' | 'api.write' | 'api.admin') => {
      if (!session?.authenticated) return false;
      if (mode === 'demo') return action === 'api.read';
      return session.actions.includes(action);
    },
    [session, mode]
  );

  return (
    <Ctx.Provider value={{ session, mode, capabilities, loading, signIn, signOut, refreshCapabilities: () => setCapTick((t) => t + 1), can }}>
      {children}
    </Ctx.Provider>
  );
};
