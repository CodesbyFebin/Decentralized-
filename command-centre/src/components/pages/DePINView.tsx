import React, { useState } from 'react';
import { Network, TrendingUp, Award, Users, Activity, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { Glass, PanelHeader, IconTile, KpiTile, StatusPill } from '../common/ui';
import { Gate, Unavailable, Note, since } from '../common/states';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import type { Freshness } from '../../types/reality';

interface DePINData {
  networks: {
    id: string;
    name: string;
    symbol: string;
    status: 'CONNECTED' | 'SYNCING' | 'STALLED' | 'DISCONNECTED';
    providers: number;
    totalStaked: number;
    monthlyRewards: number;
    apy: number;
    freshness: Freshness;
  }[];
  providers: {
    id: string;
    network: string;
    status: 'ACTIVE' | 'INACTIVE' | 'OFFLINE' | 'SLASHED';
    stake: number;
    rewardsEarned: number;
    uptime: number;
    reputation: number;
    joinedAt: number;
  }[];
  rewards: {
    id: string;
    network: string;
    amount: number;
    period: string;
    claimedAt?: number;
    expiresAt: number;
  }[];
  totalRewardsEarned: number;
  monthlyRewardRate: number;
  slashingEvents: {
    id: string;
    provider: string;
    network: string;
    reason: string;
    amount: number;
    timestamp: number;
  }[];
}

export default function DePINView() {
  const { capabilities, can } = useSession();
  const depinRes = useResource<DePINData>('/depin');
  const [selectedNetwork, setSelectedNetwork] = useState<string | null>(null);
  const [expandedProvider, setExpandedProvider] = useState<string | null>(null);

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">DePIN Networks</h1>
        <p className="text-[13px] text-slate-400">Decentralized physical infrastructure network integration, staking and rewards.</p>
      </div>

      <Gate res={depinRes}>
        {(d) => {
          const activeNetworks = d.networks.filter(n => n.status === 'CONNECTED').length;
          const activeProviders = d.providers.filter(p => p.status === 'ACTIVE').length;

          return (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <KpiTile tone="cyan" icon={<Network className="w-6 h-6" />} label="Connected networks" value={activeNetworks} sub={`of ${d.networks.length}`} />
                <KpiTile tone="violet" icon={<Users className="w-6 h-6" />} label="Active providers" value={activeProviders} sub={`of ${d.providers.length}`} />
                <KpiTile tone="emerald" icon={<Award className="w-6 h-6" />} label="Total staked" value={`$${d.networks.reduce((sum, n) => sum + n.totalStaked, 0).toFixed(0)}`} />
                <KpiTile tone="rose" icon={<TrendingUp className="w-6 h-6" />} label="Monthly rewards" value={`$${d.monthlyRewardRate.toFixed(2)}`} sub="estimated" />
              </div>

              {d.networks.length > 0 && (
                <Glass className="p-4">
                  <PanelHeader title="Available DePIN networks" subtitle="Networks you can join and stake in" />
                  <div className="mt-3 space-y-2">
                    {d.networks.map((net) => (
                      <div
                        key={net.id}
                        onClick={() => setSelectedNetwork(selectedNetwork === net.id ? null : net.id)}
                        className="p-3 rounded-lg bg-white/3 border border-white/5 cursor-pointer hover:bg-white/5 transition"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex-1">
                            <div className="font-semibold text-slate-100 flex items-center gap-2">
                              {net.name}
                              <span className="font-mono text-[11px] text-slate-500">({net.symbol})</span>
                              {net.status === 'CONNECTED' && <CheckCircle2 className="w-4 h-4 text-emerald-400" />}
                              {net.status === 'DISCONNECTED' && <AlertTriangle className="w-4 h-4 text-rose-400" />}
                            </div>
                            <div className="text-[12px] text-slate-400 mt-1">
                              {net.providers} providers · APY {net.apy}% · ${net.monthlyRewards}/mo rewards
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="font-mono text-[12px] text-cyan-200">${net.totalStaked.toFixed(0)}</div>
                            <div className="text-[11px] text-slate-400">total staked</div>
                          </div>
                        </div>

                        {selectedNetwork === net.id && (
                          <div className="mt-3 pt-3 border-t border-white/10 text-[12px] text-slate-300 space-y-1">
                            <div>Network ID: <span className="font-mono text-cyan-200">{net.id.slice(0, 16)}...</span></div>
                            <div>Status: <span className={net.status === 'CONNECTED' ? 'text-emerald-300' : 'text-slate-400'}>{net.status}</span></div>
                            {can('api.write') && (
                              <button className="mt-2 px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white text-[11px] rounded font-medium">
                                Stake in this network
                              </button>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </Glass>
              )}

              {d.providers.length > 0 && (
                <Glass className="p-4">
                  <PanelHeader title="Your provider instances" subtitle={`${activeProviders} active, earning rewards`} />
                  <div className="mt-3 space-y-2">
                    {d.providers.map((p) => (
                      <div
                        key={p.id}
                        onClick={() => setExpandedProvider(expandedProvider === p.id ? null : p.id)}
                        className="p-3 rounded-lg bg-white/3 border border-white/5 cursor-pointer hover:bg-white/5 transition"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-mono text-[12px] text-cyan-200 truncate">{p.id.slice(0, 12)}...</span>
                              <span className="text-[11px] px-2 py-1 rounded bg-white/10">{p.network}</span>
                              <StatusPill status={p.status === 'ACTIVE' ? 'ok' : p.status === 'OFFLINE' ? 'error' : 'warning'} dot={false} />
                            </div>
                            <div className="text-[12px] text-slate-400 mt-1">
                              Uptime {Math.round(p.uptime)}% · Reputation {p.reputation}
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="font-semibold text-emerald-300">${p.rewardsEarned.toFixed(2)}</div>
                            <div className="text-[11px] text-slate-400">rewards</div>
                          </div>
                        </div>

                        {expandedProvider === p.id && (
                          <div className="mt-3 pt-3 border-t border-white/10 text-[12px] text-slate-300 space-y-1">
                            <div>Staked: <span className="text-slate-100">${p.stake.toFixed(2)}</span></div>
                            <div>Status: <span className={p.status === 'ACTIVE' ? 'text-emerald-300' : 'text-slate-400'}>{p.status}</span></div>
                            <div>Joined: {since(p.joinedAt)}</div>
                            {p.status === 'ACTIVE' && (
                              <button className="mt-2 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white text-[11px] rounded font-medium">
                                Claim rewards
                              </button>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </Glass>
              )}

              {d.rewards.length > 0 && (
                <Glass className="p-4">
                  <PanelHeader title="Available rewards" subtitle="Unclaimed and pending rewards" />
                  <div className="mt-3 overflow-x-auto">
                    <table className="dh-table w-full min-w-[600px]">
                      <thead><tr><th>Network</th><th>Amount</th><th>Period</th><th>Expires</th><th>Status</th></tr></thead>
                      <tbody>
                        {d.rewards.map((r) => (
                          <tr key={r.id}>
                            <td className="text-slate-100 font-semibold">{r.network}</td>
                            <td className="font-mono text-emerald-300">${r.amount.toFixed(2)}</td>
                            <td className="text-slate-400 text-[12px]">{r.period}</td>
                            <td className="text-slate-400 text-[12px]">{since(r.expiresAt)}</td>
                            <td>{r.claimedAt ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <AlertTriangle className="w-4 h-4 text-amber-400" />}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Glass>
              )}

              {d.slashingEvents.length > 0 && (
                <Glass className="p-4 border border-rose-400/20">
                  <PanelHeader icon={<AlertTriangle className="w-4 h-4 text-rose-300" />} title="Slashing events" subtitle="Penalties applied to your providers" />
                  <div className="mt-3 space-y-2">
                    {d.slashingEvents.slice(0, 5).map((e) => (
                      <div key={e.id} className="p-2 rounded bg-rose-500/10 border border-rose-500/20 text-[12px]">
                        <div className="flex justify-between">
                          <div className="text-rose-200">{e.reason} on {e.network}</div>
                          <div className="text-rose-300">-${e.amount.toFixed(2)}</div>
                        </div>
                        <div className="text-slate-400 text-[11px] mt-1">{since(e.timestamp)}</div>
                      </div>
                    ))}
                  </div>
                </Glass>
              )}
            </>
          );
        }}
      </Gate>

      <Note>DePIN integration allows you to monetize idle hardware by running provider nodes in decentralized networks. Rewards are settled to your billing account automatically.</Note>
    </div>
  );
}
