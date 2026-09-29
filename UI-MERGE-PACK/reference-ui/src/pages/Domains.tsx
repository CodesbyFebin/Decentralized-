import React, { useState } from 'react';
import {
  Globe2,
  Plus,
  ShieldCheck,
  ExternalLink,
  CheckCircle2,
  Lock,
  ArrowRight,
  Search,
  Server
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Domains: React.FC = () => {
  const [domainList, setDomainList] = useState([
    { domain: 'agentswarm.d.host', target: 'agentswarm-web', ssl: 'Active (ECC P-384)', status: 'Active', anycast: '4 Edge Nodes' },
    { domain: 'ewastekochi.in', target: 'ewastekochi', ssl: 'Active (Custom Cert)', status: 'Active', anycast: '4 Edge Nodes' },
    { domain: 'codingagent.in', target: 'codingagent.in', ssl: 'Active (ECC P-384)', status: 'Active', anycast: '3 Edge Nodes' },
    { domain: 'bestailgent.in', target: 'bestailgent.in', ssl: 'Active (ECC P-384)', status: 'Active', anycast: '3 Edge Nodes' },
    { domain: 'decentralized.host', target: 'decentralized.host', ssl: 'Active (ECC P-384)', status: 'Active', anycast: '4 Edge Nodes' },
    { domain: 'api-pr-42.d.host', target: 'api-service', ssl: 'Active (ECC P-384)', status: 'Active', anycast: '2 Edge Nodes' },
    { domain: 'rag.d.host', target: 'rag-copilot', ssl: 'Active (ECC P-384)', status: 'Active', anycast: '3 Edge Nodes' },
  ]);

  const [newDomain, setNewDomain] = useState('');
  const [showAdd, setShowAdd] = useState(false);
  const { addToast } = useNetwork();

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomain.trim()) return;
    setDomainList((prev) => [
      { domain: newDomain, target: 'agentswarm-web', ssl: 'Active (Auto ACME)', status: 'Active', anycast: '4 Edge Nodes' },
      ...prev,
    ]);
    addToast(`Domain ${newDomain} connected with Anycast edge routing!`, 'success');
    setNewDomain('');
    setShowAdd(false);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Globe2 className="w-4 h-4 text-cyan-400" />
            <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
              DNS & Anycast Ingress
            </span>
          </div>
          <h1 className="text-2xl font-bold text-white tracking-tight">
            Domains & Distributed Edge Routing
          </h1>
          <p className="text-xs text-slate-300 mt-1">
            Map custom domains or sovereign .d.host subdomains with automatic ECC TLS termination.
          </p>
        </div>

        <button
          onClick={() => setShowAdd(true)}
          className="px-4 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/20 transition-all flex items-center gap-1.5"
        >
          <Plus className="w-4 h-4" />
          <span>Add Custom Domain</span>
        </button>
      </div>

      <div className="p-5 rounded-3xl bg-[#0c1630]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl">
        <div className="overflow-x-auto custom-scrollbar">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] font-sans">
                <th className="py-3 px-3">Domain</th>
                <th className="py-3 px-2">Target Workload</th>
                <th className="py-3 px-2">Edge Routing</th>
                <th className="py-3 px-2">TLS / SSL Status</th>
                <th className="py-3 px-2">Routing Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {domainList.map((d) => (
                <tr key={d.domain} className="hover:bg-slate-800/40 transition-colors">
                  <td className="py-3.5 px-3 font-bold text-white flex items-center gap-2">
                    <Globe2 className="w-4 h-4 text-cyan-400" />
                    <span>{d.domain}</span>
                  </td>
                  <td className="py-3.5 px-2 text-cyan-300">{d.target}</td>
                  <td className="py-3.5 px-2 text-purple-300">{d.anycast}</td>
                  <td className="py-3.5 px-2 text-emerald-400 flex items-center gap-1">
                    <ShieldCheck className="w-3.5 h-3.5" />
                    <span>{d.ssl}</span>
                  </td>
                  <td className="py-3.5 px-2">
                    <span className="px-2 py-0.5 rounded-full text-[10px] bg-emerald-500/15 text-emerald-400">
                      ● {d.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md">
          <form onSubmit={handleAdd} className="w-full max-w-md p-6 rounded-3xl bg-slate-900 border border-blue-500/30 text-white space-y-4 text-xs font-mono">
            <h3 className="text-base font-bold font-sans">Connect New Domain</h3>
            <div>
              <label className="block text-slate-400 mb-1">Domain Name</label>
              <input
                type="text"
                placeholder="app.yourdomain.com"
                value={newDomain}
                onChange={(e) => setNewDomain(e.target.value)}
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
                Connect Domain
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
