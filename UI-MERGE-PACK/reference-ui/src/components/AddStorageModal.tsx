import React, { useState } from 'react';
import {
  X,
  HardDrive,
  Database,
  Server,
  Cloud,
  Layers,
  Shield,
  ChevronRight,
  CheckCircle2,
  FolderTree
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

interface AddStorageModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AddStorageModal: React.FC<AddStorageModalProps> = ({ isOpen, onClose }) => {
  const { addStorageNode } = useNetwork();
  const [storageType, setStorageType] = useState('nas');
  const [poolName, setPoolName] = useState('pool-nas-backup');
  const [capacityTb, setCapacityTb] = useState(8);
  const [policy, setPolicy] = useState<'Private' | 'Private + Share' | 'Community' | 'DePIN (Crust)'>('Private');
  const [mountPath, setMountPath] = useState('/mnt/truenas/dh-volumes');

  if (!isOpen) return null;

  const handleCreate = () => {
    addStorageNode({
      name: poolName,
      type: storageType === 'nas'
        ? 'NAS (ZFS RAID-Z2)'
        : storageType === 's3'
        ? 'S3 Compatible Storage'
        : storageType === 'depin'
        ? 'DePIN (Crust / Storj)'
        : 'Local Disk (EXT4)',
      capacityTb,
      policy,
      mountPoint: mountPath,
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-xl max-h-[90vh] overflow-y-auto rounded-3xl bg-gradient-to-b from-[#0d1838] to-[#070d1e] border border-blue-500/30 p-6 shadow-2xl custom-scrollbar text-slate-200">
        <button
          onClick={onClose}
          className="absolute top-5 right-5 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-slate-700/50 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-5">
          <div className="w-10 h-10 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 flex items-center justify-center text-cyan-300 shadow-lg shadow-cyan-500/20">
            <HardDrive className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-white tracking-tight">
              Add Storage: Expand Your Sovereign Mesh
            </h2>
            <p className="text-xs text-slate-400">
              Turn disks you control into private, distributed infrastructure with Merkle integrity checks.
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-2">
              1. Storage Backend Type
            </label>
            <div className="grid grid-cols-2 gap-2.5">
              {[
                { id: 'local', label: 'Local Disk (EXT4/Btrfs)', icon: HardDrive, desc: 'Direct host NVMe/SSD' },
                { id: 'nas', label: 'NAS / Network Storage', icon: Database, desc: 'Synology, TrueNAS, QNAP ZFS' },
                { id: 's3', label: 'S3 Compatible Pool', icon: Cloud, desc: 'MinIO, Ceph, Wasabi, Storj' },
                { id: 'depin', label: 'DePIN Storage Network', icon: Layers, desc: 'Filecoin, Arweave, Crust' },
              ].map((item) => {
                const Icon = item.icon;
                const isSelected = storageType === item.id;
                return (
                  <button
                    key={item.id}
                    onClick={() => setStorageType(item.id)}
                    className={`p-3 rounded-2xl border text-left flex flex-col gap-1.5 transition-all ${
                      isSelected
                        ? 'bg-blue-600/20 border-cyan-400 text-white shadow-md'
                        : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:text-white'
                    }`}
                  >
                    <Icon className={`w-4 h-4 ${isSelected ? 'text-cyan-400' : 'text-slate-500'}`} />
                    <div>
                      <p className="text-xs font-semibold">{item.label}</p>
                      <p className="text-[10px] text-slate-500 line-clamp-1">{item.desc}</p>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                Volume Pool Name
              </label>
              <input
                type="text"
                value={poolName}
                onChange={(e) => setPoolName(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                Capacity (TB)
              </label>
              <input
                type="number"
                value={capacityTb}
                onChange={(e) => setCapacityTb(Number(e.target.value))}
                className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
              />
            </div>
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
              Mount Point / Path
            </label>
            <input
              type="text"
              value={mountPath}
              onChange={(e) => setMountPath(e.target.value)}
              className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1.5">
              Sharing & Contribution Policy
            </label>
            <div className="grid grid-cols-2 gap-2">
              {[
                { id: 'Private', desc: 'Only your own workloads' },
                { id: 'Private + Share', desc: 'Allow trusted peer replication' },
                { id: 'Community', desc: 'Participate in community cache' },
                { id: 'DePIN (Crust)', desc: 'Earn settlement via proof of storage' },
              ].map((p) => (
                <button
                  key={p.id}
                  onClick={() => setPolicy(p.id as any)}
                  className={`p-2.5 rounded-xl border text-left text-xs transition-all ${
                    policy === p.id
                      ? 'bg-blue-600/20 border-cyan-400 text-white'
                      : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:text-white'
                  }`}
                >
                  <p className="font-semibold">{p.id}</p>
                  <p className="text-[10px] text-slate-500">{p.desc}</p>
                </button>
              ))}
            </div>
          </div>

          <div className="pt-2 flex items-center justify-between">
            <span className="text-xs text-slate-500">
              Integrity guaranteed by Merkle tree block hashing.
            </span>
            <button
              onClick={handleCreate}
              className="px-5 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/25 transition-all flex items-center gap-2"
            >
              <span>Mount Volume</span>
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
