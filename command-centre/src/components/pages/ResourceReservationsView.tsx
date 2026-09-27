import React, { useState } from 'react';
import { Plus, Trash2, AlertCircle, CheckCircle2, Clock, Zap, TrendingUp, ChevronDown, ChevronUp } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Unavailable, Note, since } from '../common/states';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/api';

interface Capacity {
  cpuCores: number;
  memoryBytes: number;
  storageBytes: number;
}

interface Reservation {
  id: string;
  nodeId: string;
  name: string;
  status: 'pending' | 'approved' | 'allocated' | 'released';
  requested: Capacity;
  createdAt: number;
  approvedAt?: number;
  allocatedAt?: number;
  releasedAt?: number;
  reason?: string;
}

interface Allocation {
  id: string;
  reservationId: string;
  deploymentId: string;
  allocated: Capacity;
  utilizationCpuCores?: number;
  utilizationMemoryBytes?: number;
  utilizationStorageBytes?: number;
  createdAt: number;
}

interface ResourceReservationsData {
  declaredCapacity: Capacity;
  reservations: Reservation[];
  allocations: Allocation[];
}

export default function ResourceReservationsView() {
  const { session, capabilities, can } = useSession();
  const resourceRes = useResource<ResourceReservationsData>('/resources/reservations');
  const [expandedReservations, setExpandedReservations] = useState<Record<string, boolean>>({});
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: '',
    cpuCores: 1,
    memoryBytes: 1073741824,
    storageBytes: 10737418240,
    reason: '',
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const isSelfHosted = capabilities?.backend.reachable === false;

  if (isSelfHosted) {
    return (
      <div className="space-y-4 pt-2 max-w-3xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Resource Reservations</h1>
          <p className="text-[13px] text-slate-400">Self-hosted deployment.</p>
        </div>
        <Glass className="p-5 flex items-start gap-4">
          <IconTile tone="cyan" size="lg">
            <Zap className="w-6 h-6" />
          </IconTile>
          <div>
            <div className="text-[15px] font-semibold text-white">Resource reservations not configured</div>
            <p className="mt-1 text-[13px] text-slate-300">Self-hosted deployments manage capacity directly. Reservations are for multi-tenant clusters.</p>
          </div>
        </Glass>
        <Note>Reservations enforce capacity limits per deployment, preventing overallocation. Allocations track actual resource usage against reservations.</Note>
      </div>
    );
  }

  const createReservation = async () => {
    if (!can('api.admin')) return;
    if (!createForm.name.trim()) {
      setErr('Reservation name is required');
      return;
    }
    if (createForm.cpuCores < 0.5 || createForm.memoryBytes < 268435456) {
      setErr('Minimum: 0.5 CPU cores and 256 MB memory');
      return;
    }

    setBusy(true);
    setErr('');
    setSuccessMsg('');
    try {
      const res = await api('POST', '/resources/reservations', {
        body: {
          name: createForm.name,
          requested: {
            cpuCores: createForm.cpuCores,
            memoryBytes: createForm.memoryBytes,
            storageBytes: createForm.storageBytes,
          },
          reason: createForm.reason,
        },
      });
      if (res.ok) {
        setSuccessMsg(`Reservation "${createForm.name}" created (pending approval)`);
        setCreateForm({ name: '', cpuCores: 1, memoryBytes: 1073741824, storageBytes: 10737418240, reason: '' });
        setShowCreateForm(false);
        resourceRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to create reservation');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error creating reservation');
    } finally {
      setBusy(false);
    }
  };

  const approveReservation = async (id: string, name: string) => {
    if (!can('api.admin')) return;
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/resources/reservations/${id}/approve`, { body: {} });
      if (res.ok) {
        setSuccessMsg(`Reservation "${name}" approved`);
        resourceRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to approve reservation');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error approving reservation');
    } finally {
      setBusy(false);
    }
  };

  const releaseReservation = async (id: string, name: string) => {
    if (!can('api.admin')) return;
    if (deletingId !== id) {
      setDeletingId(id);
      return;
    }
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/resources/reservations/${id}/release`, { body: {} });
      if (res.ok) {
        setSuccessMsg(`Reservation "${name}" released`);
        setDeletingId(null);
        resourceRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to release reservation');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error releasing reservation');
    } finally {
      setBusy(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(1)} ${units[unitIndex]}`;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pending':
        return 'amber';
      case 'approved':
        return 'blue';
      case 'allocated':
        return 'emerald';
      case 'released':
        return 'slate';
      default:
        return 'slate';
    }
  };

  const toggleReservation = (id: string) => {
    setExpandedReservations((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Resource Reservations</h1>
        <p className="text-[13px] text-slate-400">Capacity planning and allocation enforcement. Prevents oversubscription.</p>
      </div>

      {resourceRes.loading ? (
        <Glass className="p-5">
          <p className="text-slate-400">Loading reservations...</p>
        </Glass>
      ) : resourceRes.data ? (
        <>
          {/* Capacity Overview */}
          <Glass className="p-5">
            <PanelHeader icon={<IconTile tone="cyan" size="sm"><Zap className="w-4 h-4" /></IconTile>} title="Declared capacity" />
            <div className="mt-4 grid sm:grid-cols-3 gap-4">
              <div className="p-3 bg-white/5 rounded">
                <div className="text-[12px] text-slate-400 mb-1">CPU Cores</div>
                <div className="text-xl font-semibold text-cyan-200">{resourceRes.data.declaredCapacity.cpuCores}</div>
              </div>
              <div className="p-3 bg-white/5 rounded">
                <div className="text-[12px] text-slate-400 mb-1">Memory</div>
                <div className="text-xl font-semibold text-cyan-200">{formatBytes(resourceRes.data.declaredCapacity.memoryBytes)}</div>
              </div>
              <div className="p-3 bg-white/5 rounded">
                <div className="text-[12px] text-slate-400 mb-1">Storage</div>
                <div className="text-xl font-semibold text-cyan-200">{formatBytes(resourceRes.data.declaredCapacity.storageBytes)}</div>
              </div>
            </div>
          </Glass>

          {/* Create Reservation Form */}
          <Glass className="p-5">
            <PanelHeader title="Create reservation" subtitle="Request capacity allocation" />
            {err && <div className="mt-3 text-sm text-rose-300 bg-rose-500/10 p-2 rounded">{err}</div>}
            {successMsg && <div className="mt-3 text-sm text-emerald-300 bg-emerald-500/10 p-2 rounded">{successMsg}</div>}
            {!can('api.admin') ? (
              <p className="mt-3 text-[12.5px] text-slate-400">Only api.admin can create reservations.</p>
            ) : showCreateForm ? (
              <div className="mt-3 space-y-3">
                <div>
                  <label className="text-[12px] text-slate-400 block mb-2">Reservation name</label>
                  <input
                    type="text"
                    placeholder="e.g., production-web-tier"
                    value={createForm.name}
                    onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                    disabled={busy}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                  />
                </div>
                <div className="grid sm:grid-cols-3 gap-3">
                  <div>
                    <label className="text-[12px] text-slate-400 block mb-2">CPU cores</label>
                    <input
                      type="number"
                      min="0.5"
                      step="0.5"
                      value={createForm.cpuCores}
                      onChange={(e) => setCreateForm({ ...createForm, cpuCores: parseFloat(e.target.value) })}
                      disabled={busy}
                      className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                    />
                  </div>
                  <div>
                    <label className="text-[12px] text-slate-400 block mb-2">Memory (GB)</label>
                    <input
                      type="number"
                      min="0.25"
                      step="1"
                      value={createForm.memoryBytes / 1073741824}
                      onChange={(e) => setCreateForm({ ...createForm, memoryBytes: parseFloat(e.target.value) * 1073741824 })}
                      disabled={busy}
                      className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                    />
                  </div>
                  <div>
                    <label className="text-[12px] text-slate-400 block mb-2">Storage (GB)</label>
                    <input
                      type="number"
                      min="1"
                      step="10"
                      value={createForm.storageBytes / 1073741824}
                      onChange={(e) => setCreateForm({ ...createForm, storageBytes: parseFloat(e.target.value) * 1073741824 })}
                      disabled={busy}
                      className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400"
                    />
                  </div>
                </div>
                <div>
                  <label className="text-[12px] text-slate-400 block mb-2">Reason (optional)</label>
                  <textarea
                    placeholder="Why this capacity is needed..."
                    value={createForm.reason}
                    onChange={(e) => setCreateForm({ ...createForm, reason: e.target.value })}
                    disabled={busy}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-cyan-400 h-20"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={createReservation}
                    disabled={busy}
                    className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:bg-slate-700 text-white text-sm rounded font-medium transition"
                  >
                    <Plus className="w-4 h-4" /> Create
                  </button>
                  <button
                    onClick={() => setShowCreateForm(false)}
                    className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white text-sm rounded font-medium transition"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <button
                onClick={() => setShowCreateForm(true)}
                className="mt-3 flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm rounded font-medium transition"
              >
                <Plus className="w-4 h-4" /> New reservation
              </button>
            )}
          </Glass>

          {/* Reservations List */}
          {resourceRes.data.reservations.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Active reservations" subtitle={`${resourceRes.data.reservations.length} reservation${resourceRes.data.reservations.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 space-y-2">
                {resourceRes.data.reservations.map((res) => (
                  <div key={res.id} className="border border-white/5 rounded-lg overflow-hidden">
                    <div
                      className="flex items-center justify-between p-3 bg-white/3 hover:bg-white/5 cursor-pointer"
                      onClick={() => toggleReservation(res.id)}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="font-semibold text-slate-100">{res.name}</span>
                          <StatusPill status={res.status} label={res.status.charAt(0).toUpperCase() + res.status.slice(1)} />
                        </div>
                        <div className="text-[11px] text-slate-400">
                          Created {since(res.createdAt)}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {can('api.admin') && res.status === 'pending' && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              approveReservation(res.id, res.name);
                            }}
                            disabled={busy}
                            className="px-2 py-1 text-[11px] bg-emerald-600 hover:bg-emerald-500 disabled:bg-slate-700 text-white rounded transition"
                          >
                            Approve
                          </button>
                        )}
                        {can('api.admin') && res.status !== 'released' && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              releaseReservation(res.id, res.name);
                            }}
                            disabled={busy}
                            className="p-1.5 text-slate-400 hover:text-rose-400 transition"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        )}
                        {expandedReservations[res.id] ? (
                          <ChevronUp className="w-4 h-4 text-slate-400" />
                        ) : (
                          <ChevronDown className="w-4 h-4 text-slate-400" />
                        )}
                      </div>
                    </div>

                    {expandedReservations[res.id] && (
                      <div className="p-3 bg-white/2 border-t border-white/5 space-y-2 text-[12px]">
                        <div className="grid sm:grid-cols-3 gap-3">
                          <div>
                            <span className="text-slate-400">CPU cores</span>
                            <div className="text-slate-100 font-mono">{res.requested.cpuCores}</div>
                          </div>
                          <div>
                            <span className="text-slate-400">Memory</span>
                            <div className="text-slate-100 font-mono">{formatBytes(res.requested.memoryBytes)}</div>
                          </div>
                          <div>
                            <span className="text-slate-400">Storage</span>
                            <div className="text-slate-100 font-mono">{formatBytes(res.requested.storageBytes)}</div>
                          </div>
                        </div>
                        {res.reason && (
                          <div>
                            <span className="text-slate-400">Reason</span>
                            <div className="text-slate-100 mt-1">{res.reason}</div>
                          </div>
                        )}
                        {res.approvedAt && (
                          <div className="text-emerald-300">Approved {since(res.approvedAt)}</div>
                        )}
                        {res.allocatedAt && (
                          <div className="text-emerald-300">Allocated {since(res.allocatedAt)}</div>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </Glass>
          )}

          {/* Allocations */}
          {resourceRes.data.allocations.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Active allocations" subtitle={`${resourceRes.data.allocations.length} allocation${resourceRes.data.allocations.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 space-y-2">
                {resourceRes.data.allocations.map((alloc) => (
                  <div key={alloc.id} className="p-3 rounded-lg bg-white/3 border border-white/5">
                    <div className="flex items-center justify-between mb-2">
                      <div>
                        <div className="text-sm font-semibold text-slate-100">Deployment {alloc.deploymentId}</div>
                        <div className="text-[11px] text-slate-400">Allocated {since(alloc.createdAt)}</div>
                      </div>
                      {alloc.utilizationCpuCores !== undefined && (
                        <div className="text-[11px] text-slate-400">
                          {((alloc.utilizationCpuCores / alloc.allocated.cpuCores) * 100).toFixed(0)}% CPU used
                        </div>
                      )}
                    </div>
                    <div className="text-[11px] text-slate-400 space-y-1">
                      <div>CPU: {alloc.allocated.cpuCores} cores</div>
                      <div>Memory: {formatBytes(alloc.allocated.memoryBytes)}</div>
                      <div>Storage: {formatBytes(alloc.allocated.storageBytes)}</div>
                    </div>
                  </div>
                ))}
              </div>
            </Glass>
          )}
        </>
      ) : null}

      {deletingId && (
        <Glass className="p-4 border border-rose-400/20">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <div className="text-[13px] font-semibold text-rose-300">Release reservation?</div>
              <p className="text-[12px] text-slate-300 mt-1">Releasing this reservation will free up the capacity. Any active deployments will continue but new deployments may use this capacity.</p>
              <div className="mt-3 flex gap-2">
                <button
                  onClick={() => {
                    const reservation = resourceRes.data?.reservations.find((r) => r.id === deletingId);
                    if (reservation) releaseReservation(deletingId, reservation.name);
                  }}
                  disabled={busy}
                  className="px-3 py-2 bg-rose-600 hover:bg-rose-500 disabled:bg-slate-700 text-white text-[12px] rounded font-medium"
                >
                  Release
                </button>
                <button
                  onClick={() => setDeletingId(null)}
                  className="px-3 py-2 bg-white/10 hover:bg-white/20 text-white text-[12px] rounded"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </Glass>
      )}

      <Note>Reservations enforce capacity limits per deployment. Pending reservations require approval before allocation. Allocations track actual resource usage against reserved capacity.</Note>
    </div>
  );
}
