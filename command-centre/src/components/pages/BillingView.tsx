import React from 'react';
import { Receipt } from 'lucide-react';
import { Glass, IconTile } from '../common/ui';
import { Unavailable, Note } from '../common/states';

/** Self-hosted deployments have nothing to bill. No invoices, cards or charges are shown. */
export default function BillingView() {
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
