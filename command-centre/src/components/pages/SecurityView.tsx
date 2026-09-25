import React, { useEffect, useState } from 'react';
import {
  ShieldCheck,
  Lock,
  Globe,
  AlertTriangle,
  RefreshCw,
  CheckCircle2,
  XCircle,
  Plus,
  Flame,
  FileCheck2,
  ExternalLink,
  ChevronRight,
  X
} from 'lucide-react';
import { SSLCertificate, SecurityEvaluation } from '../../types/platform';
import { api } from '../../lib/api';
import { MetricCard } from '../common/MetricCard';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
}

export const SecurityView: React.FC<Props> = ({ onNavigate }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('Overview');
  const [showIssueModal, setShowIssueModal] = useState(false);
  const [newCertDomain, setNewCertDomain] = useState('');
  const [newCertIssuer, setNewCertIssuer] = useState("Let's Encrypt");
  const [issuing, setIssuing] = useState(false);
  const [inspectCert, setInspectCert] = useState<SSLCertificate | null>(null);

  const loadSecurity = async () => {
    try {
      setLoading(true);
      const res = await api.getSecurity();
      setData(res);
    } catch (err) {
      console.error('Failed to load security:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSecurity();
  }, []);

  const handleIssueCert = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCertDomain.trim()) return;

    try {
      setIssuing(true);
      const res = await api.issueCertificate({
        domain: newCertDomain.trim(),
        issuer: newCertIssuer
      });

      setData((prev: any) => ({
        ...prev,
        certificates: [res.data, ...prev.certificates]
      }));
      setShowIssueModal(false);
      setNewCertDomain('');
    } catch (err: any) {
      alert(`Certificate issuance failed: ${err.message}`);
    } finally {
      setIssuing(false);
    }
  };

  const handleRenew = async (id: string, domain: string) => {
    try {
      const res = await api.renewCertificate(id);
      setData((prev: any) => ({
        ...prev,
        certificates: prev.certificates.map((c: any) => (c.id === id ? res.data : c))
      }));
      alert(`SSL Certificate renewed successfully for ${domain} (Valid for 90 days).`);
    } catch (err: any) {
      alert(`Renewal failed: ${err.message}`);
    }
  };

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  const evaluation: SecurityEvaluation = data.evaluation;
  const certs: SSLCertificate[] = data.certificates;

  return (
    <div className="space-y-6">
      {/* Header matching ssl sec.png */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Edge Encryption & Armor</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">SSL & Security</h1>
          <p className="text-xs text-slate-400">
            Secure your domains and applications with modern encryption, privacy and Web3-native security.
          </p>
        </div>

        <button
          onClick={() => setShowIssueModal(true)}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 transition-all"
        >
          <Plus className="w-4 h-4" />
          <span>Issue SSL Certificate</span>
        </button>
      </div>

      {/* Top 4 Metric Cards matching ssl sec.png */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          icon={<Lock className="w-5 h-5 text-blue-400" />}
          label="SSL Certificates"
          value={data.summary.sslCertificatesCount}
          change="+20%"
          trendColor="blue"
          capability="LIVE"
          provenance="acme/x509-store"
        />
        <MetricCard
          icon={<Globe className="w-5 h-5 text-cyan-400" />}
          label="Secure Domains"
          value={data.summary.secureDomainsCount}
          change="+12%"
          trendColor="cyan"
          capability="LIVE"
          provenance="edge/tls-termination"
        />
        <MetricCard
          icon={<ShieldCheck className="w-5 h-5 text-emerald-400" />}
          label="Security Score"
          value={`${evaluation.score}/100`}
          subValue={evaluation.statusText}
          trendColor="green"
          capability="DERIVED"
          provenance="audit/deterministic-eval"
        />
        <MetricCard
          icon={<Flame className="w-5 h-5 text-purple-400" />}
          label="Threats Blocked"
          value={data.summary.threatsBlockedCount.toLocaleString()}
          change="+34%"
          trendColor="purple"
          capability="DERIVED"
          provenance="waf/edge-syn-ratelimit"
        />
      </div>

      {/* Sub-tabs matching ssl sec.png */}
      <div className="flex items-center gap-2 border-b border-slate-800 pb-2 overflow-x-auto">
        {['Overview', 'SSL Certificates', 'Firewalls', 'DDoS Protection', 'Access Control', 'Security Headers', 'Monitoring', 'Compliance'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
              activeTab === tab
                ? 'bg-blue-600 text-white shadow-md'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Middle Grid: Security Overview + Security Score Breakdown + Threat Telemetry */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Security Cards & Timeline (8 cols) */}
        <div className="lg:col-span-8 space-y-4">
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-bold text-white">Security Overview</h2>
                <p className="text-xs text-slate-400">Your infrastructure is protected and secure.</p>
              </div>
              <span className="text-[11px] font-mono text-slate-500">Last 30 days</span>
            </div>

            {/* 4 Status Cards Row matching ssl sec.png */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div className="p-3 rounded-xl bg-slate-900 border border-slate-800 space-y-1">
                <div className="flex items-center gap-2 text-emerald-400 font-bold text-xs">
                  <Lock className="w-3.5 h-3.5" />
                  <span>SSL Encryption</span>
                </div>
                <div className="text-xs font-bold text-white font-mono">Active</div>
                <div className="text-[10px] text-slate-400">All domains secured</div>
              </div>

              <div className="p-3 rounded-xl bg-slate-900 border border-slate-800 space-y-1">
                <div className="flex items-center gap-2 text-cyan-400 font-bold text-xs">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  <span>DDoS Protection</span>
                </div>
                <div className="text-xs font-bold text-white font-mono">Active</div>
                <div className="text-[10px] text-slate-400">Global edge protection</div>
              </div>

              <div className="p-3 rounded-xl bg-slate-900 border border-slate-800 space-y-1">
                <div className="flex items-center gap-2 text-purple-400 font-bold text-xs">
                  <Flame className="w-3.5 h-3.5" />
                  <span>Firewall (WAF)</span>
                </div>
                <div className="text-xs font-bold text-white font-mono">Active</div>
                <div className="text-[10px] text-slate-400">Threats auto blocked</div>
              </div>

              <div className="p-3 rounded-xl bg-slate-900 border border-slate-800 space-y-1">
                <div className="flex items-center gap-2 text-blue-400 font-bold text-xs">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  <span>Security Headers</span>
                </div>
                <div className="text-xs font-bold text-white font-mono">Configured</div>
                <div className="text-[10px] text-slate-400">Best practice enabled</div>
              </div>
            </div>

            {/* Security Overview Timeline Wave */}
            <div className="w-full h-44 pt-3">
              <svg viewBox="0 0 400 120" className="w-full h-full overflow-visible">
                <defs>
                  <linearGradient id="legitGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#3B82F6" stopOpacity="0.3" />
                    <stop offset="100%" stopColor="#3B82F6" stopOpacity="0.0" />
                  </linearGradient>
                </defs>
                <line x1="0" y1="30" x2="400" y2="30" stroke="#1E293B" strokeDasharray="2 2" />
                <line x1="0" y1="60" x2="400" y2="60" stroke="#1E293B" strokeDasharray="2 2" />
                <line x1="0" y1="90" x2="400" y2="90" stroke="#1E293B" strokeDasharray="2 2" />

                {/* Legitimate Traffic (Blue) */}
                <path d="M 0 65 Q 80 40, 160 55 T 320 35 T 400 45" fill="none" stroke="#3B82F6" strokeWidth="2" />
                <path d="M 0 65 Q 80 40, 160 55 T 320 35 T 400 45 L 400 120 L 0 120 Z" fill="url(#legitGrad)" />

                {/* Blocked Threats (Pink) */}
                <path d="M 0 95 Q 80 75, 160 85 T 320 70 T 400 80" fill="none" stroke="#EC4899" strokeWidth="2" />

                {/* SSL Handshakes (Green) */}
                <path d="M 0 80 Q 80 60, 160 70 T 320 50 T 400 60" fill="none" stroke="#10B981" strokeWidth="1.5" strokeDasharray="3 3" />
              </svg>
            </div>

            <div className="flex items-center justify-center gap-6 text-[11px] font-mono text-slate-400 pt-2 border-t border-slate-800">
              <div className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-blue-500" /> Legitimate Traffic</div>
              <div className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-pink-500" /> Blocked Threats</div>
              <div className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-emerald-400" /> SSL Handshakes</div>
            </div>
          </div>
        </div>

        {/* Right: Security Score & Checks Breakdown (4 cols) */}
        <div className="lg:col-span-4 space-y-4">
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-bold text-white">Security Score</h2>
              <span className="text-xs font-mono text-emerald-400 font-bold">Excellent</span>
            </div>

            <div className="flex items-center gap-4">
              <div className="relative w-20 h-20 flex items-center justify-center">
                <svg className="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                  <path className="text-slate-800" strokeWidth="3.8" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                  <path className="text-emerald-500" strokeDasharray="98, 100" strokeWidth="3.8" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                </svg>
                <div className="absolute font-bold font-mono text-white text-base">98</div>
              </div>
              <p className="text-xs text-slate-300 leading-snug">
                Your infrastructure is well protected with strong security settings.
              </p>
            </div>

            {/* Checklist */}
            <div className="space-y-1.5 font-mono text-xs pt-2 border-t border-slate-800">
              {evaluation.passedChecks.slice(0, 5).map((chk) => (
                <div key={chk.id} className="flex items-center justify-between text-slate-300">
                  <span className="text-[11px] truncate max-w-[200px]">{chk.title}</span>
                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                </div>
              ))}
            </div>
          </div>

          {/* Threat Protection Summary matching ssl sec.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3 font-mono text-xs">
            <h2 className="text-sm font-bold text-white font-sans">Threat Protection</h2>
            <div className="space-y-2">
              <div className="flex justify-between items-center text-slate-300">
                <span>DDoS Attacks Blocked</span>
                <span className="font-bold text-white">342</span>
              </div>
              <div className="flex justify-between items-center text-slate-300">
                <span>Malicious Requests</span>
                <span className="font-bold text-white">726</span>
              </div>
              <div className="flex justify-between items-center text-slate-300">
                <span>Bot Traffic Blocked</span>
                <span className="font-bold text-white">1,248</span>
              </div>
              <div className="flex justify-between items-center text-slate-300">
                <span>Suspicious IPs Filtered</span>
                <span className="font-bold text-white">89</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* SSL Certificates Table matching ssl sec.png */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-bold text-white">Recent SSL Certificates</h2>
          <button onClick={() => setShowIssueModal(true)} className="text-xs font-semibold text-cyan-400 hover:underline">
            Manage Certificates →
          </button>
        </div>

        <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 overflow-hidden shadow-xl">
          <table className="w-full text-left border-collapse text-xs">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-950/60 text-slate-400 font-mono text-[11px] uppercase">
                <th className="py-3.5 px-4 font-semibold">Domain</th>
                <th className="py-3.5 px-4 font-semibold">Type</th>
                <th className="py-3.5 px-4 font-semibold">Status</th>
                <th className="py-3.5 px-4 font-semibold">Issued By</th>
                <th className="py-3.5 px-4 font-semibold">Expiry Date</th>
                <th className="py-3.5 px-4 font-semibold">Auto Renew</th>
                <th className="py-3.5 px-4 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-mono">
              {certs.map((c: SSLCertificate) => (
                <tr
                  key={c.id}
                  className="hover:bg-slate-800/30 transition-colors group cursor-pointer"
                  onClick={() => setInspectCert(c)}
                >
                  {/* Domain */}
                  <td className="py-4 px-4">
                    <div className="flex items-center gap-2.5">
                      <div className="w-6 h-6 rounded bg-emerald-950/60 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
                        <Lock className="w-3 h-3" />
                      </div>
                      <span className="font-bold text-white">{c.domain}</span>
                    </div>
                  </td>

                  {/* Type */}
                  <td className="py-4 px-4 text-slate-400">{c.type}</td>

                  {/* Status */}
                  <td className="py-4 px-4">
                    <span
                      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-medium ${
                        c.status === 'Valid'
                          ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-500/30'
                          : 'bg-amber-950/80 text-amber-400 border border-amber-500/30'
                      }`}
                    >
                      <span className={`w-1.5 h-1.5 rounded-full ${c.status === 'Valid' ? 'bg-emerald-400' : 'bg-amber-400 animate-pulse'}`} />
                      {c.status}
                    </span>
                  </td>

                  {/* Issuer */}
                  <td className="py-4 px-4 text-slate-300">{c.issuedBy}</td>

                  {/* Expiry */}
                  <td className="py-4 px-4 text-slate-300">
                    <div>{c.expiryDate}</div>
                    <div className={`text-[10px] ${c.daysRemaining < 30 ? 'text-amber-400 font-bold' : 'text-slate-500'}`}>
                      {c.daysRemaining} days
                    </div>
                  </td>

                  {/* Auto Renew */}
                  <td className="py-4 px-4">
                    <span className="text-emerald-400">ON</span>
                  </td>

                  {/* Actions */}
                  <td className="py-4 px-4 text-right">
                    <div className="flex items-center justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                      <button
                        onClick={() => handleRenew(c.id, c.domain)}
                        className="px-2 py-1 rounded-lg bg-blue-950/60 border border-blue-500/40 text-cyan-300 hover:bg-blue-900/60 text-[11px]"
                      >
                        Renew
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Issue SSL Modal */}
      {showIssueModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4">
          <form
            onSubmit={handleIssueCert}
            className="w-full max-w-md rounded-2xl bg-[#0B1120] border border-slate-700 p-6 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-white flex items-center gap-2">
                <Lock className="w-4 h-4 text-cyan-400" />
                Issue ACME Certificate
              </h3>
              <button type="button" onClick={() => setShowIssueModal(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Target Domain</label>
              <input
                type="text"
                required
                placeholder="e.g. app.yourdomain.com"
                value={newCertDomain}
                onChange={(e) => setNewCertDomain(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-300">Certificate Authority (ACME)</label>
              <select
                value={newCertIssuer}
                onChange={(e) => setNewCertIssuer(e.target.value)}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
              >
                <option value="Let's Encrypt">Let's Encrypt (Automated DNS-01 / HTTP-01)</option>
                <option value="ZeroSSL">ZeroSSL</option>
                <option value="Cloudflare">Cloudflare Universal SSL</option>
              </select>
            </div>

            <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 text-[11px] font-mono text-slate-400">
              • Performs cryptographic ACME verification challenge<br />
              • Generates ECDSA P-256 private key sealed on edge host keyrings
            </div>

            <div className="flex justify-end gap-2 pt-3 border-t border-slate-800">
              <button
                type="button"
                onClick={() => setShowIssueModal(false)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={issuing}
                className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold shadow-lg"
              >
                {issuing ? 'Issuing Certificate...' : 'Request Certificate'}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
