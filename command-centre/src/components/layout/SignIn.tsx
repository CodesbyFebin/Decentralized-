import React, { useState } from 'react';
import { KeyRound, Terminal } from 'lucide-react';
import { useSession } from '../../lib/session';
import { ApiError } from '../../lib/client';
import { Glass, PrimaryButton } from '../common/ui';
import { BrandMark } from '../common/Brand';

/**
 * Operators authenticate with a capability minted by the cluster root
 * (`dh token` / `dh console`). The control plane verifies it; nothing here
 * grants access on its own.
 */
export const SignIn: React.FC = () => {
  const { signIn, capabilities } = useSession();
  const [token, setToken] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      await signIn(token.trim());
    } catch (x) {
      setErr(x instanceof ApiError ? x : new ApiError(0, 'NETWORK', String(x), null));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="max-w-xl mx-auto mt-10">
      <Glass strong className="p-6">
        <div className="flex items-center gap-3">
          <BrandMark size={44} />
          <div>
            <h1 className="text-xl font-bold text-white">Sign in to the Command Centre</h1>
            <p className="text-[12.5px] text-slate-400">Use an operator capability signed by your cluster root.</p>
          </div>
        </div>

        <form onSubmit={submit} className="mt-5 space-y-3">
          <label className="block text-[12px] text-slate-300" htmlFor="cap">
            Capability token
          </label>
          <textarea
            id="cap"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            rows={4}
            spellCheck={false}
            autoComplete="off"
            placeholder="dhcap1.…"
            className="w-full rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.2)] focus:border-cyan-400/60 outline-none p-3 font-mono text-[11.5px] text-cyan-100 break-all"
          />
          {err && (
            <div role="alert" className="rounded-xl border border-rose-400/30 bg-rose-500/10 p-3 text-[12.5px] text-rose-100">
              {err.message}
              {err.requestId && <div className="mt-1 font-mono text-[10.5px] text-rose-200/70">request {err.requestId}</div>}
            </div>
          )}
          <PrimaryButton type="submit" disabled={busy || !token.trim()} className="w-full">
            <KeyRound className="w-4 h-4" /> {busy ? 'Verifying with the control plane…' : 'Sign in'}
          </PrimaryButton>
        </form>

        <div className="mt-5 rounded-xl bg-white/[0.03] border border-[rgba(125,190,255,0.12)] p-3 text-[12px] text-slate-300 space-y-1.5">
          <div className="flex items-center gap-2 font-semibold text-slate-200">
            <Terminal className="w-4 h-4 text-cyan-300" /> Mint a capability
          </div>
          <code className="block font-mono text-[11.5px] text-cyan-200">dh token --ttl 12h</code>
          <code className="block font-mono text-[11.5px] text-cyan-200">dh token --read-only --ttl 12h</code>
          <p className="text-slate-400">
            Or open the URL printed by <code className="text-cyan-200">dh console</code>; its <code>#token=</code> fragment signs you in and is removed from the address bar. The token is kept in an HttpOnly cookie, never in page storage.
          </p>
        </div>
        {capabilities && (
          <p className="mt-3 text-[11.5px] text-slate-500">
            Control plane: {capabilities.backend.reachable ? 'reachable' : 'unreachable'} — {capabilities.backend.detail}
          </p>
        )}
      </Glass>
    </div>
  );
};
