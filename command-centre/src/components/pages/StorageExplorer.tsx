import React, { useEffect, useState } from 'react';
import {
  HardDrive,
  Upload,
  Plus,
  Search,
  CheckCircle2,
  ShieldCheck,
  FileCode,
  FileArchive,
  FileText,
  Folder,
  Trash2,
  RefreshCw,
  X,
  ExternalLink,
  Layers,
  Database
} from 'lucide-react';
import { StorageObject, StorageBucket } from '../../types/platform';
import { api } from '../../lib/api';
import { MetricCard } from '../common/MetricCard';
import { WorldMap } from '../common/WorldMap';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
  /** Hide the standalone page chrome when hosted inside the Storage command surface. */
  embedded?: boolean;
  openUpload?: boolean;
}

export const StorageExplorer: React.FC<Props> = ({ onNavigate, embedded, openUpload }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState<'files' | 'replication' | 'access' | 'lifecycle'>('files');
  const [showUploadModal, setShowUploadModal] = useState(!!openUpload);
  const [uploadFileName, setUploadFileName] = useState('');
  const [uploadBucket, setUploadBucket] = useState('website-assets');
  const [uploading, setUploading] = useState(false);
  const [inspectFile, setInspectFile] = useState<StorageObject | null>(null);

  const loadStorage = async () => {
    try {
      setLoading(true);
      const res = await api.getStorage();
      setData(res);
    } catch (err) {
      console.error('Failed to load storage:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadStorage();
  }, []);

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!uploadFileName.trim()) return;

    try {
      setUploading(true);
      const res = await api.uploadStorageFile({
        name: uploadFileName.trim(),
        bucket: uploadBucket,
        type: uploadFileName.endsWith('.zip') ? 'ZIP' : uploadFileName.endsWith('.sql') ? 'SQL' : uploadFileName.endsWith('.pdf') ? 'PDF' : 'Binary'
      });

      setData((prev: any) => ({
        ...prev,
        objects: [res.data, ...prev.objects]
      }));
      setShowUploadModal(false);
      setUploadFileName('');
    } catch (err: any) {
      alert(`Upload failed: ${err.message}`);
    } finally {
      setUploading(false);
    }
  };

  const handleVerify = async (id: string) => {
    try {
      const res = await api.verifyStorageFile(id);
      alert(`Verification passed!\nSHA-256 CID: ${res.verifiedCid}\nReplicas verified: ${res.replicasVerified}/3`);
    } catch (err: any) {
      alert(`Verification failed: ${err.message}`);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Delete this storage object from all replicated mesh nodes?')) return;
    try {
      await api.deleteStorageFile(id);
      setData((prev: any) => ({
        ...prev,
        objects: prev.objects.filter((o: any) => o.id !== id)
      }));
      if (inspectFile?.id === id) setInspectFile(null);
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  };

  const filteredObjects = (data?.objects || []).filter((obj: StorageObject) =>
    obj.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    obj.bucket.toLowerCase().includes(searchTerm.toLowerCase()) ||
    obj.sha256Cid.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {embedded ? (
        <div className="flex justify-end">
          <button
            onClick={() => setShowUploadModal(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-btn-spatial text-white text-xs font-semibold shadow-lg shadow-blue-600/30"
          >
            <Upload className="w-4 h-4" />
            <span>Upload Files</span>
          </button>
        </div>
      ) : (
      <>
      {/* Header matching Storage.png */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Storage Subsystem</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">Distributed Storage</h1>
          <p className="text-xs text-slate-400">
            Store, manage and access your data across a global network of independent nodes.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowUploadModal(true)}
            className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 transition-all"
          >
            <Upload className="w-4 h-4" />
            <span>Upload Files</span>
          </button>
        </div>
      </div>

      {/* Top 4 Metric Cards matching Storage.png */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          icon={<HardDrive className="w-5 h-5 text-blue-400" />}
          label="Total Storage"
          value="12.4 TB"
          change="+18%"
          trendColor="blue"
          capability="LIVE"
          provenance="quorum/disk-blocks"
        />
        <MetricCard
          icon={<FileText className="w-5 h-5 text-purple-400" />}
          label="Total Files"
          value="1.2M"
          change="+24%"
          trendColor="purple"
          capability="LIVE"
          provenance="ipfs/merkle-index"
        />
        <MetricCard
          icon={<Layers className="w-5 h-5 text-cyan-400" />}
          label="Replicated Copies"
          value="3x"
          subValue="Geo-distributed"
          trendColor="cyan"
          capability="LIVE"
          provenance="raft/consensus"
        />
        <MetricCard
          icon={<Database className="w-5 h-5 text-emerald-400" />}
          label="Storage Cost"
          value="$28.40"
          change="-32%"
          trendColor="green"
          capability="CONFIGURED"
          provenance="node-operator/credits"
        />
      </div>

      {/* Middle Row: Global Storage Network + Storage Distribution & Health */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Global Map (7 cols) */}
        <div className="lg:col-span-7 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-sm font-bold text-white">Global Storage Network</h2>
              <p className="text-xs text-slate-400">
                Your data is automatically distributed across multiple independent nodes worldwide.
              </p>
            </div>
            <CapabilityBadge state="LIVE" />
          </div>

          <WorldMap heightClass="h-[300px]" showStorageLabels={true} showRegions={true} />
        </div>

        {/* Center: Storage Distribution Donut + Replication Status (3 cols) */}
        <div className="lg:col-span-3 space-y-4">
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-4 space-y-3">
            <h2 className="text-sm font-bold text-white">Storage Distribution</h2>
            <div className="flex items-center justify-center py-1">
              <div className="relative w-24 h-24 flex items-center justify-center">
                <svg className="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                  <path className="text-emerald-500" strokeDasharray="26, 100" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path className="text-blue-500" strokeDasharray="22, 100" strokeDashoffset="-26" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path className="text-purple-500" strokeDasharray="25, 100" strokeDashoffset="-48" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path className="text-cyan-500" strokeDasharray="11, 100" strokeDashoffset="-73" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                </svg>
                <div className="absolute flex flex-col items-center">
                  <span className="text-sm font-bold font-mono text-white">12.4 TB</span>
                  <span className="text-[8px] text-slate-400 font-mono">Total</span>
                </div>
              </div>
            </div>

            <div className="space-y-1 text-[11px] font-mono">
              <div className="flex justify-between text-slate-300">
                <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-emerald-400" /> North America</span>
                <span>26%</span>
              </div>
              <div className="flex justify-between text-slate-300">
                <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-blue-400" /> Europe</span>
                <span>22%</span>
              </div>
              <div className="flex justify-between text-slate-300">
                <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-purple-400" /> Asia</span>
                <span>25%</span>
              </div>
              <div className="flex justify-between text-slate-300">
                <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-cyan-400" /> Australia</span>
                <span>11%</span>
              </div>
            </div>
          </div>

          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-4 flex items-center justify-between">
            <div>
              <div className="text-xs text-slate-400 font-mono">Replication Status</div>
              <div className="text-lg font-bold font-mono text-white mt-0.5">3× Average</div>
              <div className="text-[10px] text-slate-400">All files stored across multiple regions</div>
            </div>
            <div className="w-8 h-8 rounded-full bg-emerald-950 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
              <CheckCircle2 className="w-4 h-4" />
            </div>
          </div>
        </div>

        {/* Right: Storage Health & Recent Events (2 cols) */}
        <div className="lg:col-span-2 space-y-4">
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-4 space-y-2">
            <h2 className="text-xs font-bold text-white">Storage Health</h2>
            <div className="flex items-center justify-between font-mono text-xs">
              <span className="text-emerald-400 font-bold text-xl">98%</span>
              <span className="text-[10px] text-slate-400">Healthy</span>
            </div>
            <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
              <div className="bg-emerald-500 h-full rounded-full" style={{ width: '98%' }} />
            </div>
          </div>

          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-4 space-y-3 font-mono text-[11px]">
            <h2 className="text-xs font-bold text-white font-sans">Recent Storage Events</h2>
            <div className="space-y-2">
              <div>
                <div className="text-slate-200">File uploaded</div>
                <div className="text-slate-500 text-[10px]">website-assets/hero.jpg · 2m ago</div>
              </div>
              <div>
                <div className="text-slate-200">Replication completed</div>
                <div className="text-slate-500 text-[10px]">backups/db-2026.sql · 8m ago</div>
              </div>
              <div>
                <div className="text-slate-200">New bucket created</div>
                <div className="text-slate-500 text-[10px]">media · 23m ago</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      </>
      )}

      {/* Files & Buckets Table matching Storage.png */}
      <div className="space-y-4">
        {/* Table Filter Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 p-3 rounded-2xl bg-[#0D1527] border border-slate-800">
          <div className="flex items-center gap-2">
            {['Files & Buckets', 'Replication', 'Access Control', 'Lifecycle'].map((tab) => (
              <button
                key={tab}
                className={`px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
                  tab === 'Files & Buckets'
                    ? 'bg-blue-600 text-white shadow-md'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
            <Search className="w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search files, folders, or CID..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-transparent text-white focus:outline-none placeholder-slate-500 w-56 font-mono"
            />
          </div>
        </div>

        {/* Table */}
        <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 overflow-hidden shadow-xl">
          <table className="w-full text-left border-collapse text-xs">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-950/60 text-slate-400 font-mono text-[11px] uppercase">
                <th className="py-3.5 px-4 font-semibold">Name</th>
                <th className="py-3.5 px-4 font-semibold">Type</th>
                <th className="py-3.5 px-4 font-semibold">Size</th>
                <th className="py-3.5 px-4 font-semibold">Replicas</th>
                <th className="py-3.5 px-4 font-semibold">Location</th>
                <th className="py-3.5 px-4 font-semibold">Modified</th>
                <th className="py-3.5 px-4 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-mono">
              {filteredObjects.map((obj: StorageObject) => (
                <tr
                  key={obj.id}
                  className="hover:bg-slate-800/30 transition-colors group cursor-pointer"
                  onClick={() => setInspectFile(obj)}
                >
                  {/* Name */}
                  <td className="py-4 px-4">
                    <div className="flex items-center gap-3">
                      <div className="w-7 h-7 rounded-lg bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-cyan-400">
                        {obj.type === 'Folder' ? (
                          <Folder className="w-4 h-4 text-amber-400" />
                        ) : obj.type === 'ZIP' ? (
                          <FileArchive className="w-4 h-4 text-purple-400" />
                        ) : obj.type === 'SQL' ? (
                          <Database className="w-4 h-4 text-emerald-400" />
                        ) : (
                          <FileText className="w-4 h-4 text-blue-400" />
                        )}
                      </div>
                      <div>
                        <div className="font-bold text-white flex items-center gap-1.5">
                          <span>{obj.name}</span>
                        </div>
                        <div className="text-[10px] text-slate-500 truncate max-w-[200px]" title={obj.sha256Cid}>
                          CID: {obj.sha256Cid.slice(0, 16)}...
                        </div>
                      </div>
                    </div>
                  </td>

                  {/* Type */}
                  <td className="py-4 px-4 text-slate-300">{obj.type}</td>

                  {/* Size */}
                  <td className="py-4 px-4 text-white font-bold">{obj.sizeFormatted}</td>

                  {/* Replicas */}
                  <td className="py-4 px-4">
                    <span className="px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-500/30 text-[11px]">
                      {obj.replicasObserved}×
                    </span>
                  </td>

                  {/* Location */}
                  <td className="py-4 px-4 text-slate-400">{obj.locationSummary}</td>

                  {/* Modified */}
                  <td className="py-4 px-4 text-slate-400">{obj.modified}</td>

                  {/* Actions */}
                  <td className="py-4 px-4 text-right">
                    <div className="flex items-center justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                      <button
                        onClick={() => handleVerify(obj.id)}
                        className="px-2.5 py-1 rounded-lg bg-cyan-950/60 border border-cyan-500/40 text-cyan-300 hover:bg-cyan-900/60 text-[11px]"
                      >
                        Verify CID
                      </button>
                      <button
                        onClick={() => handleDelete(obj.id)}
                        className="p-1 rounded-lg hover:bg-rose-950 text-slate-500 hover:text-rose-400"
                        title="Delete file"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Upload File Modal */}
      {showUploadModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4">
          <form
            onSubmit={handleUpload}
            className="w-full max-w-md rounded-2xl bg-[#0B1120] border border-slate-700 p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-white flex items-center gap-2">
                <Upload className="w-4 h-4 text-cyan-400" />
                Upload to Mesh Storage
              </h3>
              <button
                type="button"
                onClick={() => setShowUploadModal(false)}
                className="text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">File Name or Artifact</label>
              <input
                type="text"
                required
                placeholder="e.g. bundle-v3.zip or report.pdf"
                value={uploadFileName}
                onChange={(e) => setUploadFileName(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Target Bucket</label>
              <select
                value={uploadBucket}
                onChange={(e) => setUploadBucket(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
              >
                <option value="website-assets">website-assets</option>
                <option value="backups">backups</option>
                <option value="media">media</option>
              </select>
            </div>

            <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 text-[11px] font-mono text-slate-400 space-y-1">
              <div>• Content-addressing: Calculates SHA-256 Merkle CID</div>
              <div>• Quorum replication: 3× copies geo-placed across nodes</div>
            </div>

            <div className="flex justify-end gap-2 pt-3 border-t border-slate-800">
              <button
                type="button"
                onClick={() => setShowUploadModal(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={uploading}
                className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold shadow-lg shadow-blue-600/30 flex items-center gap-1.5"
              >
                {uploading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Upload className="w-3.5 h-3.5" />}
                <span>Upload & Replicate</span>
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
