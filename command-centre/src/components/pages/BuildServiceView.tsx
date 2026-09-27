import React, { useState } from 'react';
import { Plus, Trash2, AlertCircle, CheckCircle2, Clock, GitBranch, Package, FileJson, TrendingUp, ChevronDown, ChevronUp } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill, GhostButton } from '../common/ui';
import { Unavailable, Note, since } from '../common/states';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/api';

interface GitRepository {
  url: string;
  branch: string;
  commit?: string;
  lastFetched?: number;
}

interface DockerBuild {
  id: string;
  gitCommit: string;
  image: string;
  imageDigest?: string;
  status: 'pending' | 'building' | 'completed' | 'failed';
  startedAt: number;
  completedAt?: number;
  duration?: number;
  error?: string;
}

interface BuildAttestation {
  id: string;
  buildId: string;
  format: 'slsa' | 'provenance';
  signer: string;
  signature: string;
  createdAt: number;
  verifiedAt?: number;
}

interface SoftwareBOM {
  id: string;
  buildId: string;
  format: 'cyclonedx' | 'spdx';
  componentCount: number;
  vulnerabilities: number;
  createdAt: number;
  url?: string;
}

interface BuildServiceData {
  repository: GitRepository;
  builds: DockerBuild[];
  attestations: BuildAttestation[];
  sboms: SoftwareBOM[];
}

export default function BuildServiceView() {
  const { session, capabilities, can } = useSession();
  const buildRes = useResource<BuildServiceData>('/builds/service');
  const [expandedBuilds, setExpandedBuilds] = useState<Record<string, boolean>>({});
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createForm, setCreateForm] = useState({
    repoUrl: 'https://github.com/org/repo.git',
    branch: 'main',
    dockerfile: 'Dockerfile',
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const [deletingId, setDeleteingId] = useState<string | null>(null);

  const isSelfHosted = capabilities?.backend.reachable === false;

  if (isSelfHosted) {
    return (
      <div className="space-y-4 pt-2 max-w-3xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Build Service</h1>
          <p className="text-[13px] text-slate-400">Self-hosted deployment.</p>
        </div>
        <Glass className="p-5 flex items-start gap-4">
          <IconTile tone="purple" size="lg">
            <Package className="w-6 h-6" />
          </IconTile>
          <div>
            <div className="text-[15px] font-semibold text-white">Build service not configured</div>
            <p className="mt-1 text-[13px] text-slate-300">Self-hosted deployments use local build workflows. This service is for multi-tenant CI/CD orchestration.</p>
          </div>
        </Glass>
        <Note>Build service automates container image builds from Git commits, generates SBOM for security scanning, and creates cryptographic attestations for supply chain integrity.</Note>
      </div>
    );
  }

  const startBuild = async () => {
    if (!can('api.admin')) return;
    if (!createForm.repoUrl.trim()) {
      setErr('Repository URL is required');
      return;
    }
    if (!createForm.branch.trim()) {
      setErr('Branch name is required');
      return;
    }

    setBusy(true);
    setErr('');
    setSuccessMsg('');
    try {
      const res = await api('POST', '/builds/service/start', {
        body: {
          repository: {
            url: createForm.repoUrl,
            branch: createForm.branch,
          },
          dockerfile: createForm.dockerfile,
        },
      });
      if (res.ok) {
        setSuccessMsg(`Build started for ${createForm.branch}`);
        setCreateForm({ repoUrl: 'https://github.com/org/repo.git', branch: 'main', dockerfile: 'Dockerfile' });
        setShowCreateForm(false);
        buildRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to start build');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error starting build');
    } finally {
      setBusy(false);
    }
  };

  const retryBuild = async (buildId: string) => {
    if (!can('api.admin')) return;
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/builds/service/${buildId}/retry`, { body: {} });
      if (res.ok) {
        setSuccessMsg('Build retry initiated');
        buildRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to retry build');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error retrying build');
    } finally {
      setBusy(false);
    }
  };

  const deleteBuild = async (buildId: string) => {
    if (!can('api.admin')) return;
    if (deletingId !== buildId) {
      setDeleteingId(buildId);
      return;
    }
    setBusy(true);
    setErr('');
    try {
      const res = await api('POST', `/builds/service/${buildId}/delete`, { body: {} });
      if (res.ok) {
        setSuccessMsg('Build deleted');
        setDeleteingId(null);
        buildRes.refresh?.();
      } else if (res.json?.error) {
        setErr(res.json.error.message || 'Failed to delete build');
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'Error deleting build');
    } finally {
      setBusy(false);
    }
  };

  const formatDuration = (ms: number): string => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pending':
        return 'amber';
      case 'building':
        return 'blue';
      case 'completed':
        return 'emerald';
      case 'failed':
        return 'rose';
      default:
        return 'slate';
    }
  };

  const toggleBuild = (id: string) => {
    setExpandedBuilds((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Build Service</h1>
        <p className="text-[13px] text-slate-400">Git-triggered container builds with SBOM and supply chain attestation.</p>
      </div>

      {buildRes.loading ? (
        <Glass className="p-5">
          <p className="text-slate-400">Loading builds...</p>
        </Glass>
      ) : buildRes.data ? (
        <>
          {/* Git Repository */}
          <Glass className="p-5">
            <PanelHeader icon={<IconTile tone="purple" size="sm"><GitBranch className="w-4 h-4" /></IconTile>} title="Git repository" />
            <div className="mt-4 space-y-2">
              <div>
                <div className="text-[12px] text-slate-400 mb-1">Repository URL</div>
                <div className="text-sm font-mono text-cyan-200">{buildRes.data.repository.url}</div>
              </div>
              <div>
                <div className="text-[12px] text-slate-400 mb-1">Branch</div>
                <div className="text-sm font-mono text-cyan-200">{buildRes.data.repository.branch}</div>
              </div>
              {buildRes.data.repository.commit && (
                <div>
                  <div className="text-[12px] text-slate-400 mb-1">Current Commit</div>
                  <div className="text-sm font-mono text-cyan-200">{buildRes.data.repository.commit.substring(0, 12)}</div>
                </div>
              )}
              {buildRes.data.repository.lastFetched && (
                <div className="text-[11px] text-slate-500">
                  Last fetched {since(buildRes.data.repository.lastFetched)}
                </div>
              )}
            </div>
          </Glass>

          {/* Start Build Form */}
          <Glass className="p-5">
            <PanelHeader title="Trigger build" subtitle="Initiate container image build" />
            {err && <div className="mt-3 text-sm text-rose-300 bg-rose-500/10 p-2 rounded">{err}</div>}
            {successMsg && <div className="mt-3 text-sm text-emerald-300 bg-emerald-500/10 p-2 rounded">{successMsg}</div>}
            {!can('api.admin') ? (
              <p className="mt-3 text-[12.5px] text-slate-400">Only api.admin can trigger builds.</p>
            ) : showCreateForm ? (
              <div className="mt-3 space-y-3">
                <div>
                  <label className="text-[12px] text-slate-400 block mb-2">Repository URL</label>
                  <input
                    type="text"
                    placeholder="https://github.com/org/repo.git"
                    value={createForm.repoUrl}
                    onChange={(e) => setCreateForm({ ...createForm, repoUrl: e.target.value })}
                    disabled={busy}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-purple-400"
                  />
                </div>
                <div className="grid sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-[12px] text-slate-400 block mb-2">Branch</label>
                    <input
                      type="text"
                      placeholder="main"
                      value={createForm.branch}
                      onChange={(e) => setCreateForm({ ...createForm, branch: e.target.value })}
                      disabled={busy}
                      className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-purple-400"
                    />
                  </div>
                  <div>
                    <label className="text-[12px] text-slate-400 block mb-2">Dockerfile path</label>
                    <input
                      type="text"
                      placeholder="Dockerfile"
                      value={createForm.dockerfile}
                      onChange={(e) => setCreateForm({ ...createForm, dockerfile: e.target.value })}
                      disabled={busy}
                      className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-slate-100 text-[12px] focus:outline focus:outline-2 focus:outline-purple-400"
                    />
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={startBuild}
                    disabled={busy}
                    className="flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-500 disabled:bg-slate-700 text-white text-sm rounded font-medium transition"
                  >
                    <Plus className="w-4 h-4" /> Start Build
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
                className="mt-3 flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-500 text-white text-sm rounded font-medium transition"
              >
                <Plus className="w-4 h-4" /> New Build
              </button>
            )}
          </Glass>

          {/* Builds List */}
          {buildRes.data.builds.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Recent builds" subtitle={`${buildRes.data.builds.length} build${buildRes.data.builds.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 space-y-2">
                {buildRes.data.builds.map((build) => (
                  <div key={build.id} className="border border-white/5 rounded-lg overflow-hidden">
                    <div
                      className="flex items-center justify-between p-3 bg-white/3 hover:bg-white/5 cursor-pointer"
                      onClick={() => toggleBuild(build.id)}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="font-semibold text-slate-100">Commit {build.gitCommit.substring(0, 8)}</span>
                          <StatusPill status={build.status} label={build.status.charAt(0).toUpperCase() + build.status.slice(1)} />
                        </div>
                        <div className="text-[11px] text-slate-400">
                          Started {since(build.startedAt)}
                          {build.duration && ` · Duration: ${formatDuration(build.duration)}`}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {can('api.admin') && build.status === 'failed' && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              retryBuild(build.id);
                            }}
                            disabled={busy}
                            className="px-2 py-1 text-[11px] bg-amber-600 hover:bg-amber-500 disabled:bg-slate-700 text-white rounded transition"
                          >
                            Retry
                          </button>
                        )}
                        {can('api.admin') && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              deleteBuild(build.id);
                            }}
                            disabled={busy}
                            className="p-1.5 text-slate-400 hover:text-rose-400 transition"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        )}
                        {expandedBuilds[build.id] ? (
                          <ChevronUp className="w-4 h-4 text-slate-400" />
                        ) : (
                          <ChevronDown className="w-4 h-4 text-slate-400" />
                        )}
                      </div>
                    </div>

                    {expandedBuilds[build.id] && (
                      <div className="p-3 bg-white/2 border-t border-white/5 space-y-2 text-[12px]">
                        <div>
                          <span className="text-slate-400">Docker Image</span>
                          <div className="text-slate-100 font-mono mt-1">{build.image}</div>
                        </div>
                        {build.imageDigest && (
                          <div>
                            <span className="text-slate-400">Image Digest</span>
                            <div className="text-slate-100 font-mono mt-1">{build.imageDigest.substring(0, 20)}...</div>
                          </div>
                        )}
                        {build.error && (
                          <div className="p-2 bg-rose-500/10 border border-rose-400/20 rounded text-rose-300">
                            <div className="font-semibold mb-1">Error</div>
                            {build.error}
                          </div>
                        )}
                        {build.completedAt && (
                          <div className="text-emerald-300">Completed {since(build.completedAt)}</div>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </Glass>
          )}

          {/* SBOMs */}
          {buildRes.data.sboms.length > 0 && (
            <Glass className="p-5">
              <PanelHeader icon={<IconTile tone="purple" size="sm"><Package className="w-4 h-4" /></IconTile>} title="Software BOMs" subtitle={`${buildRes.data.sboms.length} SBOM${buildRes.data.sboms.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 space-y-2">
                {buildRes.data.sboms.map((sbom) => (
                  <div key={sbom.id} className="p-3 rounded-lg bg-white/3 border border-white/5">
                    <div className="flex items-center justify-between mb-2">
                      <div>
                        <div className="text-sm font-semibold text-slate-100">{sbom.format.toUpperCase()} ({sbom.componentCount} components)</div>
                        <div className="text-[11px] text-slate-400">Generated {since(sbom.createdAt)}</div>
                      </div>
                      {sbom.vulnerabilities > 0 && (
                        <div className={`text-[11px] font-semibold ${sbom.vulnerabilities > 5 ? 'text-rose-300' : 'text-amber-300'}`}>
                          {sbom.vulnerabilities} vulnerabilities
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </Glass>
          )}

          {/* Attestations */}
          {buildRes.data.attestations.length > 0 && (
            <Glass className="p-5">
              <PanelHeader icon={<IconTile tone="purple" size="sm"><FileJson className="w-4 h-4" /></IconTile>} title="Build Attestations" subtitle={`${buildRes.data.attestations.length} attestation${buildRes.data.attestations.length !== 1 ? 's' : ''}`} />
              <div className="mt-3 space-y-2">
                {buildRes.data.attestations.map((att) => (
                  <div key={att.id} className="p-3 rounded-lg bg-white/3 border border-white/5">
                    <div className="flex items-center justify-between mb-1">
                      <div className="text-sm font-semibold text-slate-100">{att.format.toUpperCase()} Attestation</div>
                      {att.verifiedAt && <span className="text-[10px] text-emerald-300 font-semibold">✓ VERIFIED</span>}
                    </div>
                    <div className="text-[11px] text-slate-400">
                      Signed by {att.signer} · {since(att.createdAt)}
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
              <div className="text-[13px] font-semibold text-rose-300">Delete build?</div>
              <p className="text-[12px] text-slate-300 mt-1">This will remove the build record and associated artifacts. Image in registry will not be deleted.</p>
              <div className="mt-3 flex gap-2">
                <button
                  onClick={() => deleteBuild(deletingId)}
                  disabled={busy}
                  className="px-3 py-2 bg-rose-600 hover:bg-rose-500 disabled:bg-slate-700 text-white text-[12px] rounded font-medium"
                >
                  Delete
                </button>
                <button
                  onClick={() => setDeleteingId(null)}
                  className="px-3 py-2 bg-white/10 hover:bg-white/20 text-white text-[12px] rounded"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </Glass>
      )}

      <Note>Build service automates container image builds from Git commits. Each build generates a Software Bill of Materials (SBOM) for supply chain security and cryptographic attestations for build integrity.</Note>
    </div>
  );
}
