import React, { useEffect, useState } from 'react';
import {
  CreditCard,
  HardDrive,
  Zap,
  Activity,
  CheckCircle2,
  AlertCircle,
  ShieldCheck,
  RefreshCw,
  ExternalLink,
  Info
} from 'lucide-react';
import { api } from '../../lib/api';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute) => void;
}

export const BillingView: React.FC<Props> = ({ onNavigate }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getBilling().then((res) => {
      setData(res);
      setLoading(false);
    });
  }, []);

  if (loading || !data) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  const { plan, currentUsage, paymentMethod, mode } = data;

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Resource Accounting & Billing</span>
            <span>·</span>
            <CapabilityBadge state="CONFIGURED" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">Billing & Usage</h1>
          <p className="text-xs text-slate-400">
            Transparent resource accounting for your self-hosted sovereign mesh.
          </p>
        </div>

        <div className="px-3 py-1.5 rounded-xl bg-purple-950/60 border border-purple-500/40 text-purple-300 font-mono text-xs">
          Mode: {mode}
        </div>
      </div>

      {/* Honest Truthfulness Banner required by specification */}
      <div className="p-4 rounded-2xl bg-[#0D1527] border border-blue-500/30 flex items-start gap-3 text-xs leading-relaxed">
        <Info className="w-5 h-5 text-cyan-400 flex-shrink-0 mt-0.5" />
        <div>
          <div className="font-bold text-white">Truthfulness Contract: Self-Hosted Sovereign Mesh</div>
          <p className="text-slate-300 mt-0.5">
            This deployment is running in self-hosted autonomous mode. External payment gateways (credit cards/Stripe) are not configured. Resource usage below represents real observed telemetry on your 43 host nodes and local capacity quotas.
          </p>
        </div>
      </div>

      {/* Plan Card */}
      <div className="rounded-2xl bg-[#0D1527] border border-slate-800 p-6 flex flex-col md:flex-row items-start md:items-center justify-between gap-6 shadow-xl">
        <div className="space-y-2">
          <span className="text-xs font-mono uppercase tracking-wider text-cyan-400 font-bold">
            Current Tier
          </span>
          <h2 className="text-2xl font-bold text-white">{plan.name}</h2>
          <p className="text-xs text-slate-400">
            Unlimited validator nodes · 3× quorum storage replication · Free ACME Let's Encrypt certificates.
          </p>
          <div className="text-xs font-mono text-slate-500">
            Billing Cycle: {plan.currentCycleStart} to {plan.currentCycleEnd}
          </div>
        </div>

        <div className="flex flex-col items-end">
          <div className="text-3xl font-extrabold font-mono text-white">
            ${plan.basePriceMonthlyUsd}
            <span className="text-xs font-normal text-slate-400"> / month</span>
          </div>
          <span className="text-[11px] font-mono text-emerald-400 font-semibold mt-1">
            ✓ Operator credits active
          </span>
        </div>
      </div>

      {/* Real Usage Meters */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Storage */}
        <div className="p-5 rounded-2xl bg-[#0D1527] border border-slate-800 space-y-3 font-mono">
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-400 flex items-center gap-1.5">
              <HardDrive className="w-4 h-4 text-blue-400" />
              Storage Meter
            </span>
            <span className="text-xs text-white font-bold">{currentUsage.storageUsedGb} GB</span>
          </div>
          <div className="w-full h-2 bg-slate-900 rounded-full overflow-hidden">
            <div
              className="bg-blue-500 h-full rounded-full"
              style={{ width: `${(currentUsage.storageUsedGb / currentUsage.storageQuotaGb) * 100}%` }}
            />
          </div>
          <div className="flex justify-between text-[11px] text-slate-500">
            <span>Quota: {currentUsage.storageQuotaGb} GB</span>
            <span>{((currentUsage.storageUsedGb / currentUsage.storageQuotaGb) * 100).toFixed(0)}% used</span>
          </div>
        </div>

        {/* Bandwidth */}
        <div className="p-5 rounded-2xl bg-[#0D1527] border border-slate-800 space-y-3 font-mono">
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-400 flex items-center gap-1.5">
              <Activity className="w-4 h-4 text-cyan-400" />
              Bandwidth Meter
            </span>
            <span className="text-xs text-white font-bold">{currentUsage.bandwidthUsedTb} TB</span>
          </div>
          <div className="w-full h-2 bg-slate-900 rounded-full overflow-hidden">
            <div
              className="bg-cyan-500 h-full rounded-full"
              style={{ width: `${(currentUsage.bandwidthUsedTb / currentUsage.bandwidthQuotaTb) * 100}%` }}
            />
          </div>
          <div className="flex justify-between text-[11px] text-slate-500">
            <span>Quota: {currentUsage.bandwidthQuotaTb} TB</span>
            <span>{((currentUsage.bandwidthUsedTb / currentUsage.bandwidthQuotaTb) * 100).toFixed(0)}% used</span>
          </div>
        </div>

        {/* Compute Cores */}
        <div className="p-5 rounded-2xl bg-[#0D1527] border border-slate-800 space-y-3 font-mono">
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-400 flex items-center gap-1.5">
              <Zap className="w-4 h-4 text-purple-400" />
              Compute Cores
            </span>
            <span className="text-xs text-white font-bold">{currentUsage.computeCoresActive} cores</span>
          </div>
          <div className="w-full h-2 bg-slate-900 rounded-full overflow-hidden">
            <div
              className="bg-purple-500 h-full rounded-full"
              style={{ width: `${(currentUsage.computeCoresActive / currentUsage.computeCoresQuota) * 100}%` }}
            />
          </div>
          <div className="flex justify-between text-[11px] text-slate-500">
            <span>Mesh Limit: {currentUsage.computeCoresQuota} cores</span>
            <span>{((currentUsage.computeCoresActive / currentUsage.computeCoresQuota) * 100).toFixed(0)}% used</span>
          </div>
        </div>
      </div>

      {/* Cost Estimate Note */}
      <div className="p-5 rounded-2xl bg-[#0D1527] border border-slate-800 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-bold text-white">Estimated Monthly Infrastructure Cost:</span>
          <span className="text-xl font-bold font-mono text-cyan-400">${currentUsage.estimatedCostUsd}</span>
        </div>
        <div className="text-[11px] font-mono text-slate-400">
          • Calculation: Derived from $0.02 / GB storage + $0.005 / GB egress bandwidth across 43 independent node providers.
        </div>
      </div>
    </div>
  );
};
