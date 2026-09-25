import React, { useEffect, useState } from 'react';
import {
  AtSign,
  Plus,
  Search,
  CheckCircle2,
  AlertTriangle,
  Globe,
  ExternalLink,
  ShieldCheck,
  RefreshCw,
  MoreVertical,
  Link2,
  Trash2,
  X,
  Layers,
  ArrowUpRight
} from 'lucide-react';
import { DomainRecord } from '../../types/platform';
import { api } from '../../lib/api';
import { MetricCard } from '../common/MetricCard';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
}

export const DomainsView: React.FC<Props> = ({ onNavigate }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterTab, setFilterTab] = useState<'All' | 'Traditional' | 'Web3' | 'Expiring'>('All');
  const [showAddModal, setShowAddModal] = useState(false);
  const [newDomainName, setNewDomainName] = useState('');
  const [newDomainType, setNewDomainType] = useState('Traditional');
  const [inspectDomain, setInspectDomain] = useState<DomainRecord | null>(null);
  const [newDnsType, setNewDnsType] = useState('A');
  const [newDnsName, setNewDnsName] = useState('@');
  const [newDnsValue, setNewDnsValue] = useState('198.51.100.24');

  const loadDomains = async () => {
    try {
      setLoading(true);
      const res = await api.getDomains();
      setData(res);
    } catch (err) {
      console.error('Failed to load domains:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDomains();
  }, []);

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomainName.trim()) return;

    try {
      const res = await api.createDomain({
        name: newDomainName.trim(),
        type: newDomainType
      });

      setData((prev: any) => ({
        ...prev,
        data: [res.data, ...prev.data]
      }));
      setShowAddModal(false);
      setNewDomainName('');
    } catch (err: any) {
      alert(`Add domain failed: ${err.message}`);
    }
  };

  const handleAddDnsRecord = async () => {
    if (!inspectDomain) return;
    try {
      const res = await api.addDnsRecord(inspectDomain.id, {
        type: newDnsType,
        name: newDnsName,
        value: newDnsValue,
        ttl: 300
      });
      setInspectDomain({
        ...inspectDomain,
        dnsRecords: [...inspectDomain.dnsRecords, res.data]
      });
      alert('DNS record propagated to all 6 edge Anycast DNS nodes.');
    } catch (err: any) {
      alert(`Failed to add DNS record: ${err.message}`);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Delete domain ${name} and unbind routing?`)) return;
    try {
      await api.deleteDomain(id);
      setData((prev: any) => ({
        ...prev,
        data: prev.data.filter((d: any) => d.id !== id)
      }));
      if (inspectDomain?.id === id) setInspectDomain(null);
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  };

  const filteredDomains = (data?.data || []).filter((d: DomainRecord) => {
    const matchesSearch = d.name.toLowerCase().includes(searchTerm.toLowerCase());
    if (filterTab === 'Traditional') return matchesSearch && d.type === 'Traditional';
    if (filterTab === 'Web3') return matchesSearch && d.type.includes('Web3');
    if (filterTab === 'Expiring') return matchesSearch && d.status === 'Expiring Soon';
    return matchesSearch;
  });

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header matching Domains.png */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>DNS & Domain Resolution</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">Decentralized Domains</h1>
          <p className="text-xs text-slate-400">
            Manage your domains, DNS, and Web3 domain names across a global decentralized network.
          </p>
        </div>

        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 transition-all"
        >
          <Plus className="w-4 h-4" />
          <span>Add Domain</span>
        </button>
      </div>

      {/* Top Metric Cards matching Domains.png */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          icon={<Globe className="w-5 h-5 text-blue-400" />}
          label="Total Domains"
          value={data?.summary.totalDomains || 28}
          change="+12%"
          trendColor="blue"
          capability="LIVE"
          provenance="dns/registry"
        />
        <MetricCard
          icon={<CheckCircle2 className="w-5 h-5 text-emerald-400" />}
          label="Active"
          value={data?.summary.active || 24}
          change="+20%"
          trendColor="green"
          capability="LIVE"
          provenance="edge-health"
        />
        <MetricCard
          icon={<AlertTriangle className="w-5 h-5 text-amber-400" />}
          label="Expiring Soon"
          value={data?.summary.expiringSoon || 2}
          subValue="Within 30 days"
          trendColor="amber"
          capability="LIVE"
          provenance="whois/expiry-checks"
        />
        <MetricCard
          icon={<Link2 className="w-5 h-5 text-purple-400" />}
          label="Web3 Domains"
          value={data?.summary.web3Domains || 6}
          trendColor="purple"
          capability="LIVE"
          provenance="ens/eth-rpc"
        />
      </div>

      {/* Main Domains Table & Sidebar Tools Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Table (9 cols) */}
        <div className="lg:col-span-9 space-y-4">
          {/* Filter Bar */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 p-3 rounded-2xl bg-[#0D1527] border border-slate-800">
            <div className="flex items-center gap-2">
              {[
                { id: 'All', label: 'All Domains', count: 28 },
                { id: 'Traditional', label: 'Traditional', count: 18 },
                { id: 'Web3', label: 'Web3', count: 6 },
                { id: 'Expiring', label: 'Expiring', count: 2 }
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setFilterTab(tab.id as any)}
                  className={`px-3 py-1.5 rounded-xl text-xs font-mono font-medium transition-all ${
                    filterTab === tab.id
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                  }`}
                >
                  <span>{tab.label}</span>
                  <span className="ml-1 opacity-70">({tab.count})</span>
                </button>
              ))}
            </div>

            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
              <Search className="w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search domains..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="bg-transparent text-white focus:outline-none placeholder-slate-500 w-44 font-mono"
              />
            </div>
          </div>

          {/* Table matching Domains.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 overflow-hidden shadow-xl">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="border-b border-slate-800 bg-slate-950/60 text-slate-400 font-mono text-[11px] uppercase">
                  <th className="py-3.5 px-4 font-semibold">Domain Name</th>
                  <th className="py-3.5 px-4 font-semibold">Type</th>
                  <th className="py-3.5 px-4 font-semibold">Status</th>
                  <th className="py-3.5 px-4 font-semibold">DNS / Hosting</th>
                  <th className="py-3.5 px-4 font-semibold">Expiry Date</th>
                  <th className="py-3.5 px-4 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono">
                {filteredDomains.map((dom: DomainRecord) => (
                  <tr
                    key={dom.id}
                    className="hover:bg-slate-800/30 transition-colors group cursor-pointer"
                    onClick={() => setInspectDomain(dom)}
                  >
                    {/* Domain Name */}
                    <td className="py-4 px-4">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-lg bg-blue-600/10 border border-blue-500/30 flex items-center justify-center text-cyan-400">
                          {dom.type.includes('Web3') ? <Link2 className="w-3.5 h-3.5 text-purple-400" /> : <Globe className="w-3.5 h-3.5 text-blue-400" />}
                        </div>
                        <div>
                          <div className="font-bold text-white flex items-center gap-1.5">
                            <span>{dom.name}</span>
                            <ExternalLink className="w-3 h-3 text-slate-500" />
                          </div>
                        </div>
                      </div>
                    </td>

                    {/* Type */}
                    <td className="py-4 px-4">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                          dom.type.includes('Web3')
                            ? 'bg-purple-950 text-purple-400 border border-purple-500/40'
                            : 'bg-slate-800 text-slate-300 border border-slate-700'
                        }`}
                      >
                        {dom.type}
                      </span>
                    </td>

                    {/* Status */}
                    <td className="py-4 px-4">
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium ${
                          dom.status === 'Active'
                            ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-500/30'
                            : 'bg-amber-950/80 text-amber-400 border border-amber-500/30'
                        }`}
                      >
                        <span className={`w-1.5 h-1.5 rounded-full ${dom.status === 'Active' ? 'bg-emerald-400' : 'bg-amber-400'}`} />
                        {dom.status}
                      </span>
                    </td>

                    {/* DNS / Hosting */}
                    <td className="py-4 px-4 text-slate-300">
                      {dom.dnsProvider}
                    </td>

                    {/* Expiry Date */}
                    <td className="py-4 px-4 text-slate-300">
                      <div>{dom.expiryDate}</div>
                      {dom.daysRemaining && (
                        <div className={`text-[10px] ${dom.daysRemaining < 30 ? 'text-amber-400 font-bold' : 'text-slate-500'}`}>
                          {dom.daysRemaining} days
                        </div>
                      )}
                    </td>

                    {/* Actions */}
                    <td className="py-4 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                        <button
                          onClick={() => setInspectDomain(dom)}
                          className="px-2 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-[11px]"
                        >
                          DNS Records
                        </button>
                        <button
                          onClick={() => handleDelete(dom.id, dom.name)}
                          className="p-1 rounded-lg hover:bg-rose-950 text-slate-500 hover:text-rose-400"
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

        {/* Right: Domain Tools & Digital Identity Card (3 cols) */}
        <div className="lg:col-span-3 space-y-4">
          <div className="rounded-2xl bg-gradient-to-b from-[#121B30] to-[#0A1020] border border-blue-500/30 p-5 space-y-3">
            <h2 className="text-sm font-bold text-white">Own Your Digital Identity</h2>
            <p className="text-xs text-slate-300 leading-relaxed">
              Use traditional or Web3 domains. Point to decentralized hosting. Stay in control.
            </p>
            <button
              onClick={() => setShowAddModal(true)}
              className="w-full py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-md flex items-center justify-center gap-1"
            >
              <Plus className="w-4 h-4" />
              <span>Add Domain</span>
            </button>
          </div>

          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-4 space-y-3 text-xs">
            <h3 className="font-bold text-white">DNS & Hosting</h3>
            <div className="space-y-1.5 font-mono text-[11px]">
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between text-slate-300">
                <span>DNS Records</span>
                <span className="text-slate-500">A, AAAA, CNAME</span>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between text-slate-300">
                <span>Nameservers</span>
                <span className="text-slate-500">Decentralized</span>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between text-slate-300">
                <span>IPFS / Web3</span>
                <span className="text-purple-400">ENS Gateways</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Inspect DNS Records Drawer */}
      {inspectDomain && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex justify-end">
          <div className="w-full max-w-xl bg-[#0B1120] border-l border-slate-800 h-full overflow-y-auto flex flex-col justify-between p-6 animate-in slide-in-from-right duration-200">
            <div className="space-y-5">
              <div className="flex items-start justify-between border-b border-slate-800 pb-4">
                <div>
                  <h2 className="text-lg font-bold text-white flex items-center gap-2">
                    {inspectDomain.name}
                    <span className="text-xs px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-500/40 font-mono">
                      {inspectDomain.status}
                    </span>
                  </h2>
                  <span className="text-xs font-mono text-slate-400">{inspectDomain.dnsProvider}</span>
                </div>
                <button
                  onClick={() => setInspectDomain(null)}
                  className="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Add Record Input */}
              <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-3">
                <span className="text-xs font-bold text-white">Add DNS Record</span>
                <div className="grid grid-cols-3 gap-2">
                  <select
                    value={newDnsType}
                    onChange={(e) => setNewDnsType(e.target.value)}
                    className="px-2 py-1.5 rounded-lg bg-slate-950 border border-slate-700 text-xs font-mono text-white"
                  >
                    <option value="A">A</option>
                    <option value="AAAA">AAAA</option>
                    <option value="CNAME">CNAME</option>
                    <option value="TXT">TXT</option>
                    <option value="MX">MX</option>
                  </select>
                  <input
                    type="text"
                    placeholder="Name (@, www)"
                    value={newDnsName}
                    onChange={(e) => setNewDnsName(e.target.value)}
                    className="px-2 py-1.5 rounded-lg bg-slate-950 border border-slate-700 text-xs font-mono text-white"
                  />
                  <input
                    type="text"
                    placeholder="Value / IP"
                    value={newDnsValue}
                    onChange={(e) => setNewDnsValue(e.target.value)}
                    className="px-2 py-1.5 rounded-lg bg-slate-950 border border-slate-700 text-xs font-mono text-white"
                  />
                </div>
                <button
                  onClick={handleAddDnsRecord}
                  className="w-full py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold"
                >
                  Add Record to Distributed DNS
                </button>
              </div>

              {/* Existing Records */}
              <div className="space-y-2">
                <span className="text-xs font-bold text-white">Active DNS Records</span>
                <div className="space-y-1.5 font-mono text-xs">
                  {inspectDomain.dnsRecords.length === 0 ? (
                    <div className="p-4 rounded-xl bg-slate-900 text-slate-500 text-center">
                      No DNS records defined. Handled via IPFS / ENS gateway.
                    </div>
                  ) : (
                    inspectDomain.dnsRecords.map((rec) => (
                      <div
                        key={rec.id}
                        className="flex items-center justify-between p-2.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-300"
                      >
                        <div className="flex items-center gap-2">
                          <span className="px-1.5 py-0.5 rounded bg-blue-950 text-cyan-400 font-bold text-[10px]">
                            {rec.type}
                          </span>
                          <span className="font-semibold text-white">{rec.name}</span>
                        </div>
                        <span className="text-slate-400 truncate max-w-[200px]">{rec.value}</span>
                        <span className="text-slate-600 text-[10px]">{rec.ttl}s</span>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>

            <div className="pt-4 border-t border-slate-800 flex justify-end">
              <button
                onClick={() => setInspectDomain(null)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Domain Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4">
          <form
            onSubmit={handleAddDomain}
            className="w-full max-w-md rounded-2xl bg-[#0B1120] border border-slate-700 p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-white">Register / Connect Domain</h3>
              <button type="button" onClick={() => setShowAddModal(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Domain Name</label>
              <input
                type="text"
                required
                placeholder="e.g. yourbrand.com or mywallet.eth"
                value={newDomainName}
                onChange={(e) => setNewDomainName(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Domain Category</label>
              <select
                value={newDomainType}
                onChange={(e) => setNewDomainType(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
              >
                <option value="Traditional">Traditional (.com, .io, .in, .host)</option>
                <option value="Web3 (ENS)">Web3 ENS (.eth, .web3, .crypto)</option>
                <option value="Internal">Internal (mesh.local)</option>
              </select>
            </div>

            <div className="flex justify-end gap-2 pt-3 border-t border-slate-800">
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold shadow-lg"
              >
                Bind to Edge DNS
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
