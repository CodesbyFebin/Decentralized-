import React, { useEffect, useState } from 'react';
import {
  BarChart3,
  Users,
  FileText,
  Clock,
  ShieldCheck,
  HardDrive,
  Download,
  Globe,
  RefreshCw,
  Activity,
  ArrowUpRight,
  ArrowDownRight
} from 'lucide-react';
import { api } from '../../lib/api';
import { MetricCard } from '../common/MetricCard';
import { WorldMap } from '../common/WorldMap';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
}

export const AnalyticsView: React.FC<Props> = ({ onNavigate }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState('30D');

  const loadAnalytics = async () => {
    try {
      setLoading(true);
      const res = await api.getAnalytics(range);
      setData(res);
    } catch (err) {
      console.error('Failed to load analytics:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAnalytics();
  }, [range]);

  const handleExport = () => {
    const jsonStr = JSON.stringify(data, null, 2);
    const blob = new Blob([jsonStr], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `decentralized-host-analytics-${range}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  const { overview, realtime, regionalTraffic, deviceTypes, trafficSources, nodePerformance } = data;

  return (
    <div className="space-y-6">
      {/* Header matching Analystic.png */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Telemetry & Ingress Analytics</span>
            <span>·</span>
            <CapabilityBadge state="DERIVED" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">Analytics & Insights</h1>
          <p className="text-xs text-slate-400">
            Real-time analytics for your decentralized websites and applications.
          </p>
        </div>

        {/* Time filters & Export matching Analystic.png */}
        <div className="flex items-center gap-2 bg-[#0D1527] border border-slate-800 p-1.5 rounded-2xl">
          {['24H', '7D', '30D', '90D', '1Y'].map((t) => (
            <button
              key={t}
              onClick={() => setRange(t)}
              className={`px-3 py-1.5 rounded-xl text-xs font-mono font-medium transition-all ${
                range === t ? 'bg-blue-600 text-white shadow-md' : 'text-slate-400 hover:text-white'
              }`}
            >
              {t}
            </button>
          ))}
          <button
            onClick={handleExport}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono transition-all ml-1"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export</span>
          </button>
        </div>
      </div>

      {/* Top 6 Metric Cards matching Analystic.png */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <MetricCard
          icon={<Users className="w-5 h-5 text-blue-400" />}
          label="Total Visitors"
          value="128.4K"
          change="+24%"
          trendColor="blue"
          capability="DERIVED"
          provenance="edge/unique-ip"
        />
        <MetricCard
          icon={<FileText className="w-5 h-5 text-purple-400" />}
          label="Page Views"
          value="412.7K"
          change="+32%"
          trendColor="purple"
          capability="DERIVED"
          provenance="ingress/http-requests"
        />
        <MetricCard
          icon={<Users className="w-5 h-5 text-cyan-400" />}
          label="Unique Users"
          value="96.1K"
          change="+27%"
          trendColor="cyan"
          capability="DERIVED"
          provenance="cookie-free-hash"
        />
        <MetricCard
          icon={<Clock className="w-5 h-5 text-emerald-400" />}
          label="Avg. Response Time"
          value="284 ms"
          change="-18%"
          trendColor="green"
          capability="LIVE"
          provenance="edge/synthetic-p95"
        />
        <MetricCard
          icon={<ShieldCheck className="w-5 h-5 text-purple-400" />}
          label="Uptime"
          value="99.97%"
          change="+0.02%"
          trendColor="purple"
          capability="LIVE"
          provenance="probes/30s-interval"
        />
        <MetricCard
          icon={<HardDrive className="w-5 h-5 text-blue-400" />}
          label="Bandwidth Usage"
          value="2.4 TB"
          change="+19%"
          trendColor="blue"
          capability="LIVE"
          provenance="bgp/router-interface"
        />
      </div>

      {/* Middle Row: Global Traffic Map (8 cols) + Real-time Activity (4 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Global Traffic */}
        <div className="lg:col-span-8 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-sm font-bold text-white">Global Traffic</h2>
              <p className="text-xs text-slate-400">Visitors to your sites and apps across decentralized nodes worldwide.</p>
            </div>
            <CapabilityBadge state="DERIVED" />
          </div>

          <WorldMap heightClass="h-[320px]" showRegions={true} />
        </div>

        {/* Right: Real-time Activity matching Analystic.png */}
        <div className="lg:col-span-4 rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-bold text-white">Real-time Activity</h2>
            <span className="flex items-center gap-1.5 text-emerald-400 text-xs font-mono">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping" />
              Live
            </span>
          </div>

          <div>
            <div className="text-3xl font-extrabold font-mono text-white tracking-tight">1,482</div>
            <div className="text-xs text-slate-400 font-mono">active users right now</div>
          </div>

          {/* Mini Realtime Bar Chart */}
          <div className="flex items-end justify-between h-16 pt-2 border-b border-slate-800 pb-2">
            {[45, 60, 35, 70, 85, 40, 95, 100, 65, 80, 55, 90, 75, 88].map((val, idx) => (
              <div
                key={idx}
                className="w-2 rounded-t bg-gradient-to-t from-blue-600 to-cyan-400 hover:opacity-100 transition-all"
                style={{ height: `${val}%` }}
              />
            ))}
          </div>

          {/* Top Active Pages matching Analystic.png */}
          <div className="space-y-2 pt-1">
            <span className="text-xs font-bold text-slate-400">Top Active Pages</span>
            <div className="space-y-1.5 font-mono text-xs">
              {realtime.topPages.map((pg: any) => (
                <div key={pg.path} className="flex items-center justify-between">
                  <span className="text-slate-300 font-semibold">{pg.path}</span>
                  <div className="flex items-center gap-2">
                    <div className="w-24 h-1.5 bg-slate-900 rounded-full overflow-hidden">
                      <div className="bg-blue-500 h-full rounded-full" style={{ width: `${(pg.active / 320) * 100}%` }} />
                    </div>
                    <span className="text-slate-400 text-[11px] w-8 text-right">{pg.active}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Bottom Grid: Traffic & Performance + Device Types + Traffic Sources + Node Performance */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Device Types Donut (4 cols) */}
        <div className="lg:col-span-4 rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
          <h2 className="text-sm font-bold text-white">Device Types</h2>
          <div className="flex items-center justify-center py-2">
            <div className="relative w-28 h-28 flex items-center justify-center">
              <svg className="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                <path className="text-blue-500" strokeDasharray="54, 100" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path className="text-purple-500" strokeDasharray="38, 100" strokeDashoffset="-54" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path className="text-cyan-500" strokeDasharray="6, 100" strokeDashoffset="-92" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path className="text-slate-600" strokeDasharray="2, 100" strokeDashoffset="-98" strokeWidth="4" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
              </svg>
              <div className="absolute font-mono text-center">
                <span className="text-sm font-bold text-white">128.4K</span>
                <span className="block text-[8px] text-slate-500 uppercase">Total</span>
              </div>
            </div>
          </div>

          <div className="space-y-1.5 font-mono text-xs">
            <div className="flex justify-between text-slate-300">
              <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-blue-500" /> Desktop</span>
              <span>{deviceTypes.desktop}%</span>
            </div>
            <div className="flex justify-between text-slate-300">
              <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-purple-500" /> Mobile</span>
              <span>{deviceTypes.mobile}%</span>
            </div>
            <div className="flex justify-between text-slate-300">
              <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-cyan-400" /> Tablet</span>
              <span>{deviceTypes.tablet}%</span>
            </div>
            <div className="flex justify-between text-slate-300">
              <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-slate-500" /> Other</span>
              <span>{deviceTypes.other}%</span>
            </div>
          </div>
        </div>

        {/* Traffic Sources (4 cols) */}
        <div className="lg:col-span-4 rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
          <h2 className="text-sm font-bold text-white">Traffic Sources</h2>
          <div className="space-y-3 font-mono text-xs pt-1">
            {trafficSources.map((src: any) => (
              <div key={src.source} className="space-y-1">
                <div className="flex justify-between text-slate-300">
                  <span>{src.source}</span>
                  <span className="text-white font-bold">{src.percent}%</span>
                </div>
                <div className="w-full h-1.5 bg-slate-900 rounded-full overflow-hidden">
                  <div className="bg-gradient-to-r from-blue-500 to-cyan-400 h-full rounded-full" style={{ width: `${src.percent}%` }} />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Node Performance Table (4 cols) matching Analystic.png */}
        <div className="lg:col-span-4 rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-bold text-white">Node Performance</h2>
            <button onClick={() => onNavigate('nodes')} className="text-xs font-semibold text-cyan-400 hover:underline">
              View All Nodes →
            </button>
          </div>

          <div className="space-y-2.5 font-mono text-xs">
            {nodePerformance.slice(0, 5).map((np: any) => (
              <div key={np.node} className="p-2 rounded-xl bg-slate-900/60 border border-slate-800 space-y-1">
                <div className="flex justify-between items-center">
                  <span className="font-bold text-white">{np.node}</span>
                  <span className="text-emerald-400 font-semibold">{np.uptime}%</span>
                </div>
                <div className="flex justify-between text-[11px] text-slate-400">
                  <span>{np.location}</span>
                  <span className="text-cyan-400">{np.responseMs} ms (load {np.load}%)</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
