import React, { useState } from 'react';
import { Eye, EyeOff, Plus, Trash2, RotateCw, Lock, AlertCircle, CheckCircle2 } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Unavailable, Note, since } from '../common/states';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/api';

interface Secret {
  id: string;
  name: string;
  createdAt: number;
  rotatedAt: number;
  usedInDeployments: string[];
  lastAccessedAt?: number;
  rotationSchedule?: 'never' | '30days' | '90days' | '365days';
}

interface SecretsData {
  secrets: Secret[];
  rotationPolicy: {
    enforcedSchedule: '30days' | '90days' | '365days' | 'never';
    maxSecretAge: number;
  };
}

export default function SecretsView() {
  const { session, capabilities, can } = useSession();
  const secretsRes = useResource<SecretsData>('/secrets');
  const [createName, setCreateName] = useState('');
  const [createValue, setCreateValue] = useState('');
  const [showValue, setShowValue] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmDeleteName, setConfirmDeleteName] = useState('');
  const [rotatingId, setRotatingId] = useState<string | null>(null);

  const isSelfHosted = capabilities?.backend.reachable === false;

  if (isSelfHosted) {
    return (
      <div className="space-y-4 pt-2 max-w-3xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Secrets</h1>
          <p className="text-[13px] text-slate-400">Self-hosted deployment.</p>
        </div>
        <Glass className="p-5 flex items-start gap-4">
          <IconTile tone="cyan" size="lg"><Lock className="w-6 h-6" /></IconTile>
          <div>
            <div className="text-[15px] font-semibold text-white">Secrets management not configured</div>
            <p className="mt-1 text-[13px] text-slate-300">Self-hosted deployments can use environment variables directly. For production, configure a secrets subsystem.</p>
          </div>
        </Glass>
        <Note>Secrets are encrypted at rest and in transit. Access is logged to the audit ledger. Rotation is enforced per policy.</Note>
      </div>
    );
  }

  const createSecret = async () => {
    if (!can('api.admin')) return;
    if (!createName.trim() || !createValue) {
      setErr('Secret name and value are required');
      return;
    }
    if (!/^[A-Z_][A-Z0-9_]*$/.test(createName)) {
      setErr('Secret name must be uppercase alphanumeric with underscores, starting with a letter or underscore');
      return;
    }

    setBusy(true);
    setErr('');
    setSuccessMsg('');
    try {
      const res = await api('POST', '/secrets', {
        body: { name: createName, value: createValue }
      });
      if (res.ok) {
        setSuccessMsg(`Secret "${createName}" created and encrypted`);
        setCreateName('');
        setCreateValue('');
        setShowValue(false);
        secretsRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to create secret');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error creating secret');
    } finally {
      setBusy(false);
    }
  };

  const deleteSecret = async (id: string, name: string) => {
    if (confirmDeleteName !== name) {
      setDeletingId(id);
      return;
    }
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/secrets/${id}/delete`, { body: {} });
      if (res.ok) {
        setSuccessMsg(`Secret "${name}" deleted`);
        setDeletingId(null);
        setConfirmDeleteName('');
        secretsRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to delete secret');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error deleting secret');
    } finally {
      setBusy(false);
    }
  };

  const rotateSecret = async (id: string, name: string) => {
    if (!can('api.admin')) return;
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/secrets/${id}/rotate`, { body: {} });
      if (res.ok) {
        setSuccessMsg(`Secret "${name}" rotated`);
        setRotatingId(null);
        secretsRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to rotate secret');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error rotating secret');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Secrets</h1>
        <p className="text-[13px] text-slate-400">Encrypted environment variables, API keys, and credentials. All access is audited.</p>
      </div>

      {secretsRes.loading ? (
        <Glass className="p-5"><p className="text-slate-400">Loading secrets...</p></Glass>
      ) : secretsRes.data ? (
        <>
          <Glass className="p-5">
            <PanelHeader icon={<IconTile tone="cyan" size="sm"><Lock className="w-4 h-4" /></IconTile>} title="Rotation policy" />
            <div className="mt-3 grid sm:grid-cols-2 gap-x-6 gap-y-1.5 text-[12.5px]">
              <div className="flex justify-between">
                <span className="text-slate-400">Enforcement</span>
                <span className="text-slate-100">{secretsRes.data.rotationPolicy.enforcedSchedule}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Max age</span>
                <span className="text-slate-100">{(secretsRes.data.rotationPolicy.maxSecretAge / 86400).toFixed(0)} days</span>
              </div>
            </div>
          </Glass>

          <Glass className="p-5">
            <PanelHeader title="Create secret" subtitle="Add a new encrypted secret" />
            {err && <div className="mt-3 text-sm text-rose-300 bg-rose-500/10 p-2 rounded">{err}</div>}
            {successMsg && <div className="mt-3 text-sm text-emerald-300 bg-emerald-500/10 p-2 rounded">{successMsg}</div>}
            {!can('api.admin') ? (
              <p className="mt-3 text-[12.5px] text-slate-400">Only api.admin can create secrets.</p>
            ) : (
              <div className="mt-3 space-y-3">
                <div>
                  <label className="text-[12px] text-slate-400 block mb-2">Secret name (uppercase, alphanumeric + underscores)</label>
                  <input
                    type="text"
                    placeholder="e.g., DATABASE_PASSWORD, API_KEY_PROD"
                    value={createName}
                    onChange={(e) => setCreateName(e.target.value.toUpperCase())}
                    disabled={busy}
                    className="w-full md:w-96 px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                  />
                </div>
                <div>
                  <label className="text-[12px] text-slate-400 block mb-2">Secret value (never logged)</label>
                  <div className="relative">
                    <input
                      type={showValue ? 'text' : 'password'}
                      placeholder="Value is encrypted at rest and in transit"
                      value={createValue}
                      onChange={(e) => setCreateValue(e.target.value)}
                      disabled={busy}
                      className="w-full md:w-96 px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                    />
                    <button
                      onClick={() => setShowValue(!showValue)}
                      className="absolute right-3 top-2.5 text-slate-400 hover:text-slate-300"
                    >
                      {showValue ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>
                <button
                  onClick={createSecret}
                  disabled={busy}
                  className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:bg-slate-700 text-white text-sm rounded font-medium transition"
                >
                  <Plus className="w-4 h-4" /> Create secret
                </button>
              </div>
            )}
          </Glass>

          {secretsRes.data.secrets.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Active secrets" subtitle={`${secretsRes.data.secrets.length} secret${secretsRes.data.secrets.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 space-y-2">
                {secretsRes.data.secrets.map((secret) => (
                  <div key={secret.id} className="flex items-center justify-between p-3 rounded-lg bg-white/3 border border-white/5">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-mono text-[12px] text-cyan-200">{secret.name}</span>
                        {secret.usedInDeployments.length > 0 && (
                          <span className="text-[10px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300">
                            Used in {secret.usedInDeployments.length} deployment{secret.usedInDeployments.length !== 1 ? 's' : ''}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-4 text-[11px] text-slate-400">
                        <span>Created {since(secret.createdAt)}</span>
                        {secret.lastAccessedAt && <span>Last used {since(secret.lastAccessedAt)}</span>}
                      </div>
                      {secret.rotatedAt && (
                        <div className="text-[11px] text-slate-400 mt-1">Rotated {since(secret.rotatedAt)}</div>
                      )}
                    </div>
                    {can('api.admin') && (
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => rotateSecret(secret.id, secret.name)}
                          disabled={busy || rotatingId === secret.id}
                          className="p-1.5 text-slate-400 hover:text-emerald-400 transition disabled:opacity-50"
                          title="Rotate secret"
                        >
                          <RotateCw className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => deleteSecret(secret.id, secret.name)}
                          disabled={busy}
                          className="p-1.5 text-slate-400 hover:text-rose-400 transition disabled:opacity-50"
                          title="Delete secret"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </Glass>
          )}

          {deletingId && (
            <Glass className="p-4 border border-rose-400/20">
              <div className="flex items-start gap-3">
                <AlertCircle className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
                <div className="flex-1">
                  <div className="text-[13px] font-semibold text-rose-300">Confirm deletion</div>
                  <p className="text-[12px] text-slate-300 mt-1">Type the secret name to confirm permanent deletion. This cannot be undone.</p>
                  <div className="mt-3 flex gap-2">
                    <input
                      type="text"
                      placeholder="Enter secret name to confirm"
                      value={confirmDeleteName}
                      onChange={(e) => setConfirmDeleteName(e.target.value)}
                      className="flex-1 px-3 py-2 bg-white/5 border border-white/10 rounded text-[12px] text-slate-100 focus:outline focus:outline-2 focus:outline-rose-400"
                    />
                    <button
                      onClick={() => deleteSecret(deletingId, confirmDeleteName)}
                      disabled={busy || confirmDeleteName !== secretsRes.data?.secrets.find(s => s.id === deletingId)?.name}
                      className="px-3 py-2 bg-rose-600 hover:bg-rose-500 disabled:bg-slate-700 text-white text-[12px] rounded font-medium"
                    >
                      Delete
                    </button>
                    <button
                      onClick={() => {
                        setDeletingId(null);
                        setConfirmDeleteName('');
                      }}
                      className="px-3 py-2 bg-white/10 hover:bg-white/20 text-white text-[12px] rounded"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            </Glass>
          )}
        </>
      ) : null}

      <Note>Secrets are encrypted with AES-256-GCM and stored separately from code. Access is audited per the control plane. Rotation is enforced by policy and tracked in the audit ledger. Secret values are never logged, cached, or stored in browser history.</Note>
    </div>
  );
}
