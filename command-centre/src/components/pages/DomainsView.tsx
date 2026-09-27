import React, { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Globe, Plus, Edit2, RotateCcw, Trash2, Copy, Calendar } from 'lucide-react';
import type { CertRec, DomainRec, AppRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError } from '../../lib/client';
import { Glass, IconTile, StatusPill, PrimaryButton, GhostButton } from '../common/ui';
import { Gate, Empty, Note, fmtTime, TruthTag, ErrorState, shortDigest } from '../common/states';

interface AddDomainForm {
  host: string;
  app: string;
  port: string;
  tlsMode: 'none' | 'local' | 'acme';
}

export default function DomainsView() {
  const { can, mode } = useSession();
  const res = useResource<{ domains: DomainRec[]; certificates: CertRec[] }>('/deployments');
  const apps = useResource<{ apps: AppRec[] }>('/apps');
  const domains = useResource<{ domains: DomainRec[]; certificates: CertRec[] }>('/domains');

  const [showAddDialog, setShowAddDialog] = useState(false);
  const [selectedDomain, setSelectedDomain] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [form, setForm] = useState<AddDomainForm>({
    host: '',
    app: '',
    port: '',
    tlsMode: 'local'
  });

  const allowed = can('api.write') && mode === 'controlplane';
  const appList = useMemo(() => apps.data?.apps ?? [], [apps.data]);

  const addDomain = async () => {
    if (!form.host || !form.app || !form.port) return;
    setBusy(true);
    setErr(null);
    setSuccessMsg(null);
    try {
      await api.post('/domains', {
        host: form.host,
        app: form.app,
        port: form.port,
        tlsMode: form.tlsMode
      });
      setSuccessMsg(`Domain ${form.host} created`);
      setForm({ host: '', app: '', port: '', tlsMode: 'local' });
      setShowAddDialog(false);
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setBusy(false);
    }
  };

  const renewCert = async (host: string) => {
    setBusy(true);
    setErr(null);
    try {
      await api.post(`/domains/${encodeURIComponent(host)}/renew`, {});
      setSuccessMsg(`Certificate renewal triggered for ${host}`);
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setBusy(false);
    }
  };

  const deleteDomain = async (host: string) => {
    if (!confirm(`Delete domain ${host}?`)) return;
    setBusy(true);
    setErr(null);
    try {
      await api.del(`/domains/${encodeURIComponent(host)}`);
      setSuccessMsg(`Domain ${host} removed`);
      setSelectedDomain(null);
    } catch (e) {
      setErr(e as ApiError);
    } finally {
      setBusy(false);
    }
  };

  const input = 'w-full rounded-lg bg-black/30 border border-[rgba(125,190,255,0.2)] focus:border-cyan-400/60 outline-none px-2.5 py-2 text-[13px]';
  const label = 'block text-[12px] text-slate-300 space-y-1';

  return (
    <div className="space-y-4 pt-2 max-w-6xl">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Domains</h1>
          <p className="text-[13px] text-slate-400 mt-1">Routing from manifests, routing observed at edge hosts, and TLS certificates.</p>
        </div>
        {allowed && (
          <PrimaryButton onClick={() => setShowAddDialog(true)} className="flex items-center gap-2">
            <Plus className="w-4 h-4" /> Add domain
          </PrimaryButton>
        )}
      </div>

      {successMsg && (
        <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/10 p-3 text-[12.5px] text-emerald-100">
          {successMsg}
        </div>
      )}
      {err && <ErrorState error={err} />}

      {showAddDialog && (
        <Glass className="p-4 space-y-3">
          <div className="text-lg font-bold text-white">Add domain</div>
          <div className="grid grid-cols-2 gap-3">
            <label className={label}>
              Domain name
              <input
                className={input}
                placeholder="app.example.com"
                value={form.host}
                onChange={(e) => setForm({ ...form, host: e.target.value.trim() })}
                disabled={busy}
              />
            </label>
            <label className={label}>
              Application
              <select
                className={input}
                value={form.app}
                onChange={(e) => setForm({ ...form, app: e.target.value, port: '' })}
                disabled={busy}
              >
                <option value="">Select app...</option>
                {appList.map((a) => <option key={a.name} value={a.name}>{a.name}</option>)}
              </select>
            </label>
            <label className={label}>
              Port name
              <input
                className={input}
                placeholder="http"
                value={form.port}
                onChange={(e) => setForm({ ...form, port: e.target.value.trim() })}
                disabled={busy}
              />
            </label>
            <label className={label}>
              TLS mode
              <select
                className={input}
                value={form.tlsMode}
                onChange={(e) => setForm({ ...form, tlsMode: e.target.value as 'none' | 'local' | 'acme' })}
                disabled={busy}
              >
                <option value="local">Local (cluster CA)</option>
                <option value="acme">ACME (Let's Encrypt)</option>
                <option value="none">None (HTTP only)</option>
              </select>
            </label>
          </div>
          <div className="flex gap-2 justify-end">
            <GhostButton onClick={() => setShowAddDialog(false)} disabled={busy}>Cancel</GhostButton>
            <PrimaryButton onClick={addDomain} disabled={!form.host || !form.app || !form.port || busy}>
              {busy ? 'Creating...' : 'Create domain'}
            </PrimaryButton>
          </div>
        </Glass>
      )}

      <Gate res={domains}>
        {(d) =>
          !d.domains.length ? (
            <Empty
              title="No domains"
              detail="Add spec.ingress[].host to an application manifest to create a domain binding. Decentralized.Host is not a registrar or DNS provider; point the name at an edge host yourself."
            />
          ) : (
            <>
              <Glass className="overflow-x-auto">
                <table className="dh-table w-full min-w-[1200px]">
                  <thead>
                    <tr>
                      <th>Domain</th>
                      <th>Application</th>
                      <th>Routing</th>
                      <th>TLS Status</th>
                      <th>Certificate</th>
                      <th>Expires</th>
                      {allowed && <th>Actions</th>}
                    </tr>
                  </thead>
                  <tbody>
                    {d.domains.map((x) => {
                      const cert = x.tls;
                      const daysLeft = cert?.notAfter ? Math.ceil((cert.notAfter - Date.now()) / 86400000) : null;
                      const expiringSoon = daysLeft !== null && daysLeft < 30;

                      return (
                        <tr key={x.host} className={selectedDomain === x.host ? 'bg-cyan-500/10' : ''}>
                          <td>
                            <div className="flex items-center gap-2">
                              <IconTile tone="blue" size="sm"><Globe className="w-4 h-4" /></IconTile>
                              <div>
                                <div className="font-semibold text-sky-300">{x.host}</div>
                                <button
                                  onClick={() => navigator.clipboard.writeText(x.host)}
                                  className="text-[10px] text-slate-500 hover:text-slate-300 flex items-center gap-1"
                                >
                                  <Copy className="w-3 h-3" /> Copy
                                </button>
                              </div>
                            </div>
                          </td>
                          <td>
                            <Link to={`/apps/${encodeURIComponent(x.desired.app)}`} className="text-slate-200 hover:text-cyan-300">
                              {x.desired.app}
                            </Link>
                            <div className="text-[11px] text-slate-500">port {x.desired.port} · {x.desired.tls}</div>
                          </td>
                          <td>
                            <StatusPill status={x.routing.state === 'NOT ROUTED' ? 'UNKNOWN' : x.routing.state} />
                            <div className="text-[11px] text-slate-500 whitespace-normal">{x.routing.detail}</div>
                          </td>
                          <td>
                            {cert ? (
                              <>
                                <StatusPill status={cert.state} />
                                <div className="text-[11px] text-slate-500">{cert.issuer}</div>
                              </>
                            ) : (
                              <StatusPill status={x.desired.tls === 'none' ? 'NONE' : 'UNKNOWN'} tone="slate" />
                            )}
                          </td>
                          <td>
                            {cert && (
                              <div className="text-[12px] space-y-1">
                                <div className="text-slate-300 font-mono text-[10px]">{shortDigest(cert.fingerprint)}</div>
                                <div className="text-slate-500">Serial: {shortDigest(cert.serial)}</div>
                              </div>
                            )}
                          </td>
                          <td className={expiringSoon ? 'text-amber-300 font-semibold' : 'text-slate-300'}>
                            {cert?.notAfter ? (
                              <div>
                                <div>{fmtTime(cert.notAfter)}</div>
                                {daysLeft !== null && <div className="text-[11px] text-slate-500">{daysLeft} days</div>}
                              </div>
                            ) : (
                              <span className="text-slate-500">—</span>
                            )}
                          </td>
                          {allowed && (
                            <td>
                              <div className="flex items-center gap-1">
                                {cert && daysLeft !== null && daysLeft < 90 && (
                                  <button
                                    onClick={() => renewCert(x.host)}
                                    disabled={busy}
                                    className="text-slate-400 hover:text-cyan-300 p-1"
                                    title="Renew certificate"
                                  >
                                    <RotateCcw className="w-4 h-4" />
                                  </button>
                                )}
                                <button
                                  onClick={() => deleteDomain(x.host)}
                                  disabled={busy}
                                  className="text-slate-400 hover:text-rose-300 p-1"
                                  title="Delete domain"
                                >
                                  <Trash2 className="w-4 h-4" />
                                </button>
                              </div>
                            </td>
                          )}
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </Glass>
              <Note>DNS is not observed: the control plane does not resolve public DNS or register names. Routing and TLS status come from edge-host observations.</Note>
            </>
          )
        }
      </Gate>
    </div>
  );
}
