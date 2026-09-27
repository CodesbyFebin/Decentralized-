import React, { useState } from 'react';
import { Receipt, TrendingUp, DollarSign, Zap, Database, Globe, AlertCircle, CheckCircle2 } from 'lucide-react';
import { Glass, PanelHeader, IconTile, KpiTile } from '../common/ui';
import { Gate, Unavailable, Note, fmtBytes, since } from '../common/states';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import type { Freshness, Metric } from '../../types/reality';

interface BillingData {
  metering: {
    cpuSecondTotal: number;
    memoryByteHours: number;
    storageByteHours: number;
    egressBytes: number;
    periodStart: number;
    periodEnd: number;
  };
  settlements: {
    id: string;
    amount: number;
    currency: string;
    status: 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED';
    method: string;
    createdAt: number;
    completedAt?: number;
  }[];
  leases: {
    id: string;
    resource: string;
    quantity: number;
    unit: string;
    pricePerUnit: number;
    startAt: number;
    endAt: number;
    status: 'ACTIVE' | 'EXPIRED' | 'CANCELLED';
  }[];
  currentBalance: number;
  credits: { amount: number; source: string; expiresAt?: number }[];
  invoices: {
    id: string;
    number: string;
    amount: number;
    currency: string;
    status: 'DRAFT' | 'SENT' | 'PAID' | 'OVERDUE' | 'CANCELLED';
    issuedAt: number;
    dueAt: number;
  }[];
  freshness: Freshness;
}

export default function BillingView() {
  const { capabilities, can } = useSession();
  const billingRes = useResource<BillingData>('/billing');
  const [selectedInvoice, setSelectedInvoice] = useState<string | null>(null);

  const isSelfHosted = capabilities?.backend.reachable === false;

  if (isSelfHosted) {
    return (
      <div className="space-y-4 pt-2 max-w-3xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Billing</h1>
          <p className="text-[13px] text-slate-400">Self-hosted deployment.</p>
        </div>
        <Glass className="p-5 flex items-start gap-4">
          <IconTile tone="cyan" size="lg"><Receipt className="w-6 h-6" /></IconTile>
          <div>
            <div className="text-[15px] font-semibold text-white">External billing is not configured</div>
            <p className="mt-1 text-[13px] text-slate-300">You run this cluster on hardware you control. There is no subscription, invoice, payment method or wallet involved, and none is required to self-host.</p>
          </div>
        </Glass>
        <div className="grid md:grid-cols-2 gap-3">
          <Unavailable title="Usage metering" state="PLANNED" detail="CPU-seconds, byte-hours and egress are not metered yet; there is no usage to settle." />
          <Unavailable title="Settlement" state="PLANNED" detail="Marketplace settlement (credits, fiat or on-chain) is a later phase and is optional by design." />
        </div>
        <Note>When metering exists, this page will show usage records with their source and signature before any settlement rail.</Note>
      </div>
    );
  }

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Billing & Marketplace</h1>
        <p className="text-[13px] text-slate-400">Usage metering, settlement, leases and invoice management.</p>
      </div>

      <Gate res={billingRes}>
        {(d) => (
          <>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <KpiTile tone="cyan" icon={<Zap className="w-6 h-6" />} label="CPU used" value={Math.round(d.metering.cpuSecondTotal / 3600)} sub="hours this period" />
              <KpiTile tone="violet" icon={<Database className="w-6 h-6" />} label="Storage used" value={fmtBytes(d.metering.storageByteHours / (30 * 86400))} sub="avg per day" />
              <KpiTile tone="emerald" icon={<Globe className="w-6 h-6" />} label="Egress" value={fmtBytes(d.metering.egressBytes)} sub="this period" />
              <KpiTile tone="rose" icon={<DollarSign className="w-6 h-6" />} label="Current balance" value={`$${d.currentBalance.toFixed(2)}`} sub="account credit" />
            </div>

            {d.leases.length > 0 && (
              <Glass className="p-4">
                <PanelHeader title="Active leases" subtitle={`${d.leases.filter(l => l.status === 'ACTIVE').length} resource leases`} />
                <div className="mt-3 overflow-x-auto">
                  <table className="dh-table w-full min-w-[700px]">
                    <thead><tr><th>Resource</th><th>Quantity</th><th>Price per unit</th><th>Status</th><th>Expires</th></tr></thead>
                    <tbody>
                      {d.leases.map((l) => (
                        <tr key={l.id}>
                          <td className="text-slate-100">{l.resource}</td>
                          <td className="tabular-nums text-slate-300">{l.quantity} {l.unit}</td>
                          <td className="tabular-nums text-slate-300">${l.pricePerUnit.toFixed(4)}</td>
                          <td className={l.status === 'ACTIVE' ? 'text-emerald-300' : 'text-slate-400'}>{l.status}</td>
                          <td className="text-slate-400 text-[12px]">{since(l.endAt)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Glass>
            )}

            {d.settlements.length > 0 && (
              <Glass className="p-4">
                <PanelHeader title="Recent settlements" subtitle="Completed and pending transactions" />
                <div className="mt-3 space-y-2">
                  {d.settlements.slice(0, 5).map((s) => (
                    <div key={s.id} className="flex items-center justify-between p-3 rounded-lg bg-white/3 border border-white/5">
                      <div className="flex-1">
                        <div className="font-mono text-[12px] text-cyan-200">{s.id.slice(0, 12)}...</div>
                        <div className="text-[11px] text-slate-400 mt-1">{s.method} · {since(s.createdAt)}</div>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="text-right">
                          <div className="font-semibold text-slate-100">${s.amount.toFixed(2)}</div>
                          <div className="text-[11px] text-slate-400">{s.currency}</div>
                        </div>
                        {s.status === 'COMPLETED' && <CheckCircle2 className="w-4 h-4 text-emerald-400" />}
                        {s.status === 'PENDING' && <AlertCircle className="w-4 h-4 text-amber-400" />}
                        {s.status === 'FAILED' && <AlertCircle className="w-4 h-4 text-rose-400" />}
                      </div>
                    </div>
                  ))}
                </div>
              </Glass>
            )}

            {d.credits.length > 0 && (
              <Glass className="p-4">
                <PanelHeader title="Credits" subtitle={`$${d.credits.reduce((sum, c) => sum + c.amount, 0).toFixed(2)} total`} />
                <div className="mt-3 space-y-2">
                  {d.credits.map((c, i) => (
                    <div key={i} className="flex justify-between items-center p-2 rounded bg-emerald-500/10 border border-emerald-500/20">
                      <div>
                        <div className="text-[12px] text-emerald-300">{c.source}</div>
                        {c.expiresAt && <div className="text-[11px] text-slate-400">expires {since(c.expiresAt)}</div>}
                      </div>
                      <div className="text-emerald-300 font-semibold">${c.amount.toFixed(2)}</div>
                    </div>
                  ))}
                </div>
              </Glass>
            )}

            {d.invoices.length > 0 && (
              <Glass className="p-4">
                <PanelHeader title="Invoices" subtitle={`${d.invoices.filter(i => i.status !== 'PAID').length} unpaid`} />
                <div className="mt-3 overflow-x-auto">
                  <table className="dh-table w-full min-w-[700px]">
                    <thead><tr><th>Invoice</th><th>Amount</th><th>Status</th><th>Issued</th><th>Due</th></tr></thead>
                    <tbody>
                      {d.invoices.map((inv) => (
                        <tr key={inv.id} className={inv.status === 'OVERDUE' ? 'bg-rose-500/5' : ''}>
                          <td className="font-mono text-cyan-200 text-[12px] cursor-pointer hover:text-cyan-100" onClick={() => setSelectedInvoice(inv.id)}>{inv.number}</td>
                          <td className="tabular-nums text-slate-300">${inv.amount.toFixed(2)}</td>
                          <td className={inv.status === 'PAID' ? 'text-emerald-300' : inv.status === 'OVERDUE' ? 'text-rose-300' : 'text-slate-400'}>{inv.status}</td>
                          <td className="text-slate-400 text-[12px]">{since(inv.issuedAt)}</td>
                          <td className="text-slate-400 text-[12px]">{since(inv.dueAt)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Glass>
            )}
          </>
        )}
      </Gate>

      <Note>Metering includes CPU-seconds, memory-byte-hours, storage-byte-hours and egress. Settlement methods support fiat transfers, blockchain tokens and marketplace credits.</Note>
    </div>
  );
}
