import React, { useState } from 'react';
import {
  ShieldCheck,
  Key,
  Lock,
  CheckCircle2,
  AlertCircle,
  Eye,
  EyeOff,
  Plus,
  RefreshCw,
  Terminal
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Security: React.FC = () => {
  const { addToast } = useNetwork();
  const [secrets, setSecrets] = useState([
    { name: 'API_SECRET_KEY', boundApps: ['agentswarm-web'], version: 'v2', updated: '2d ago', status: 'Sealed & Active' },
    { name: 'DATABASE_URL', boundApps: ['agentswarm-web', 'ewastekochi'], version: 'v1', updated: '1w ago', status: 'Sealed & Active' },
    { name: 'GEMINI_API_KEY', boundApps: ['rag-copilot'], version: 'v3', updated: '1h ago', status: 'Sealed & Active' },
    { name: 'JWT_SECRET', boundApps: ['api-service'], version: 'v1', updated: '3d ago', status: 'Sealed & Active' },
    { name: 'POSTGRES_PASSWORD', boundApps: ['ewastekochi'], version: 'v1', updated: '1d ago', status: 'Sealed & Active' },
  ]);

  const [newKey, setNewKey] = useState('');
  const [newVal, setNewVal] = useState('');
  const [showAdd, setShowAdd] = useState(false);

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newKey.trim()) return;
    setSecrets((prev) => [
      { name: newKey.toUpperCase(), boundApps: ['General Cluster'], version: 'v1', updated: 'Just now', status: 'Sealed & Active' },
      ...prev,
    ]);
    addToast(`Secret ${newKey.toUpperCase()} encrypted with per-secret DEK and committed to Raft!`, 'success');
    setNewKey('');
    setNewVal('');
    setShowAdd(false);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Lock className="w-4 h-4 text-cyan-400" />
            <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
              Cryptographic Envelopes & Secrets (Rule #14)
            </span>
          </div>
          <h1 className="text-2xl font-bold text-white tracking-tight">
            SSL & Secrets Management
          </h1>
          <p className="text-xs text-slate-300 mt-1 max-w-2xl leading-relaxed">
            ROOT/KEK → per-secret DEK → AES-256-GCM ciphertext committed to Raft consensus.
            Memory-only tmpfs delivery with single-consumption invariant.
          </p>
        </div>

        <button
          onClick={() => setShowAdd(true)}
          className="px-4 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/20 transition-all flex items-center gap-1.5"
        >
          <Plus className="w-4 h-4" />
          <span>Add Sealed Secret</span>
        </button>
      </div>

      {/* Security Invariants Bar */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 font-mono text-xs">
        <div className="p-4 rounded-2xl bg-slate-900/80 border border-blue-500/20">
          <span className="text-[10px] uppercase text-slate-400 block mb-1">Cluster Key Encryption</span>
          <span className="text-sm font-bold text-cyan-300">AES-256-GCM</span>
          <p className="text-[11px] text-slate-400 mt-1 font-sans">Hardware enclave isolated</p>
        </div>
        <div className="p-4 rounded-2xl bg-slate-900/80 border border-blue-500/20">
          <span className="text-[10px] uppercase text-slate-400 block mb-1">Security Invariant</span>
          <span className="text-sm font-bold text-purple-300">Single Consumption</span>
          <p className="text-[11px] text-slate-400 mt-1 font-sans">Lost response ≠ Second authorization</p>
        </div>
        <div className="p-4 rounded-2xl bg-slate-900/80 border border-blue-500/20">
          <span className="text-[10px] uppercase text-slate-400 block mb-1">Runtime Memory</span>
          <span className="text-sm font-bold text-emerald-300">tmpfs / Zero-Disk</span>
          <p className="text-[11px] text-slate-400 mt-1 font-sans">Secrets destroyed on process exit</p>
        </div>
      </div>

      {/* Secrets Table */}
      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
        <div className="overflow-x-auto custom-scrollbar">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                <th className="py-3 px-3">Secret Key</th>
                <th className="py-3 px-2">Bound Workloads</th>
                <th className="py-3 px-2">Version</th>
                <th className="py-3 px-2">Status</th>
                <th className="py-3 px-2">Last Rotated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {secrets.map((s) => (
                <tr key={s.name} className="hover:bg-slate-800/40 transition-colors">
                  <td className="py-3.5 px-3 font-bold text-white flex items-center gap-2">
                    <Key className="w-4 h-4 text-cyan-400" />
                    <span>{s.name}</span>
                  </td>
                  <td className="py-3.5 px-2 text-cyan-300">{s.boundApps.join(', ')}</td>
                  <td className="py-3.5 px-2 text-purple-300">{s.version}</td>
                  <td className="py-3.5 px-2 text-emerald-400 flex items-center gap-1">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    <span>{s.status}</span>
                  </td>
                  <td className="py-3.5 px-2 text-slate-400">{s.updated}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md">
          <form onSubmit={handleAdd} className="w-full max-w-md p-6 rounded-3xl bg-slate-900 border border-blue-500/30 text-white space-y-4 text-xs font-mono">
            <h3 className="text-base font-bold font-sans">Add Sealed Secret</h3>
            <div>
              <label className="block text-slate-400 mb-1">Key Name (e.g. DATABASE_URL)</label>
              <input
                type="text"
                placeholder="DATABASE_URL"
                value={newKey}
                onChange={(e) => setNewKey(e.target.value)}
                className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700 text-white"
              />
            </div>
            <div>
              <label className="block text-slate-400 mb-1">Secret Value (Will be encrypted client-side)</label>
              <input
                type="password"
                placeholder="••••••••••••••••"
                value={newVal}
                onChange={(e) => setNewVal(e.target.value)}
                className="w-full p-2.5 rounded-xl bg-slate-800 border border-slate-700 text-white"
              />
            </div>
            <div className="flex justify-end gap-2 pt-2 font-sans">
              <button
                type="button"
                onClick={() => setShowAdd(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-4 py-2 rounded-xl bg-cyan-600 hover:bg-cyan-500 font-bold text-white"
              >
                Seal & Store Secret
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
