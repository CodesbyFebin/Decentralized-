import React from 'react';
import {
  CreditCard,
  Coins,
  Shield,
  CheckCircle2,
  TrendingDown,
  ArrowRight,
  Wallet,
  Zap
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

export const Billing: React.FC = () => {
  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="p-6 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl">
        <div className="flex items-center gap-2 mb-1">
          <CreditCard className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-semibold text-cyan-400 font-mono uppercase tracking-wider">
            Settlement Neutral Core (Rule #21)
          </span>
        </div>
        <h1 className="text-2xl font-bold text-white tracking-tight">
          Billing & Resource Settlement
        </h1>
        <p className="text-xs text-slate-300 mt-1 max-w-2xl leading-relaxed">
          Zero token required to run on your own hardware.
          For third-party DePIN leasing, choose your preferred settlement rail: Free, Internal Credits, Fiat USD, or Web3 Stablecoins.
        </p>
      </div>

      {/* Free Self-Hosting Guarantee Card */}
      <div className="p-5 rounded-3xl bg-emerald-950/20 border border-emerald-500/30 flex items-center justify-between">
        <div className="flex items-center gap-3.5">
          <div className="p-2.5 rounded-2xl bg-emerald-500/20 text-emerald-400">
            <CheckCircle2 className="w-6 h-6" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-white">Sovereign Self-Hosting: $0.00 Platform Fee</h3>
            <p className="text-xs text-slate-300">
              You own the hardware; you provide the electricity. The platform charges 0% commission on your private cloud.
            </p>
          </div>
        </div>
        <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
          PERMANENTLY FREE
        </span>
      </div>

      {/* Settlement Channels */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono text-xs">
        <div className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 space-y-3">
          <div className="flex justify-between items-center pb-2 border-b border-slate-800">
            <span className="text-white font-sans font-bold">Credit Balance</span>
            <span className="text-cyan-300 font-bold">$240.00</span>
          </div>
          <p className="text-slate-400 font-sans text-[11px]">
            Used for automatic payment of community edge leases and storage replica verification.
          </p>
          <button className="w-full py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-bold transition-all">
            Add Balance
          </button>
        </div>

        <div className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 space-y-3">
          <div className="flex justify-between items-center pb-2 border-b border-slate-800">
            <span className="text-white font-sans font-bold">DePIN Provider Earnings</span>
            <span className="text-emerald-400 font-bold">+$48.50 / mo</span>
          </div>
          <p className="text-slate-400 font-sans text-[11px]">
            Accrued from contributed GPU/CPU hours on gpu-workstation and homelab-kochi.
          </p>
          <button className="w-full py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 transition-all">
            Withdraw Earnings
          </button>
        </div>

        <div className="p-5 rounded-2xl bg-[#0c1630]/90 border border-blue-500/25 space-y-3">
          <div className="flex justify-between items-center pb-2 border-b border-slate-800">
            <span className="text-white font-sans font-bold">Optional Web3 Wallet</span>
            <span className="text-purple-300 font-bold">Disconnected</span>
          </div>
          <p className="text-slate-400 font-sans text-[11px]">
            Optional. Connect EVM/L2 wallet only if using permissionless on-chain escrow settlement.
          </p>
          <button className="w-full py-2 rounded-xl bg-purple-600/30 hover:bg-purple-600/50 text-purple-200 border border-purple-500/40 transition-all flex items-center justify-center gap-1.5">
            <Wallet className="w-3.5 h-3.5" />
            <span>Connect Web3 Wallet</span>
          </button>
        </div>
      </div>
    </div>
  );
};
