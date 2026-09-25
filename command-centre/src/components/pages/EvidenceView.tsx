import React, { useEffect, useState } from 'react';
import {
  FileCheck2,
  ShieldCheck,
  Fingerprint,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Lock,
  GitCommit,
  ExternalLink
} from 'lucide-react';
import { EvidenceRecord } from '../../types/platform';
import { api } from '../../lib/api';
import { CapabilityBadge } from '../common/CapabilityBadge';

export const EvidenceView: React.FC = () => {
  const [records, setRecords] = useState<EvidenceRecord[]>([]);
  const [authority, setAuthority] = useState<string>('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getEvidenceLedger().then((res) => {
      setRecords(res.records);
      setAuthority(res.ledgerAuthority);
      setLoading(false);
    });
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Cryptographic Trust Plane</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">
            Evidence & Qualification Ledger
          </h1>
          <p className="text-xs text-slate-400">
            Immutable records proving runtime source code integrity, test verification gates, and signer attestation.
          </p>
        </div>

        <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-emerald-950/60 border border-emerald-500/40 text-emerald-400 font-mono text-xs">
          <ShieldCheck className="w-4 h-4" />
          <span>Ledger Status: Sealed & Immutable</span>
        </div>
      </div>

      {/* Ledger Authority Banner */}
      <div className="p-4 rounded-2xl bg-[#0D1527] border border-blue-500/30 flex items-start gap-3 text-xs font-mono text-slate-300">
        <Lock className="w-5 h-5 text-cyan-400 flex-shrink-0 mt-0.5" />
        <div>
          <div className="font-bold text-white">Authority: {authority}</div>
          <p className="text-slate-400 mt-1 leading-relaxed">
            Every software release and host node enrollment in Decentralized.Host is signed with an ED25519 root identity and hashed with SHA-256. Historical evidence records cannot be deleted, mutated, or forged.
          </p>
        </div>
      </div>

      {/* Records List */}
      <div className="space-y-4">
        {records.map((rec) => (
          <div
            key={rec.recordId}
            className="rounded-2xl bg-[#0D1527] border border-slate-800 p-6 space-y-4 font-mono text-xs shadow-xl"
          >
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 border-b border-slate-800 pb-3">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-lg bg-emerald-950/80 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
                  <FileCheck2 className="w-4 h-4" />
                </div>
                <div>
                  <span className="font-bold text-white text-sm font-sans">{rec.recordId}</span>
                  <div className="text-[10px] text-slate-500">{rec.timestamp}</div>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <span className="px-2.5 py-1 rounded bg-emerald-950 text-emerald-400 border border-emerald-500/40 text-xs font-bold">
                  ✓ {rec.status}
                </span>
                <span className="text-slate-400 text-[11px]">
                  {rec.gatesPassed.length} / {rec.gatesTotal} Gates Passed
                </span>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-1">
                <span className="text-slate-500 uppercase text-[10px]">SHA-256 Release Digest</span>
                <div className="p-2 rounded-lg bg-slate-900 border border-slate-800 text-cyan-400 break-all select-all">
                  {rec.sha256Digest}
                </div>
              </div>

              <div className="space-y-1">
                <span className="text-slate-500 uppercase text-[10px]">Attesting Signer Fingerprint</span>
                <div className="p-2 rounded-lg bg-slate-900 border border-slate-800 text-purple-400 break-all select-all">
                  {rec.signerFingerprint}
                </div>
              </div>
            </div>

            {rec.sourceCommit && (
              <div className="flex items-center gap-2 text-slate-400 pt-1">
                <GitCommit className="w-4 h-4 text-slate-500" />
                <span>Source Revision: {rec.sourceCommit}</span>
              </div>
            )}

            <div className="pt-2 border-t border-slate-800/80">
              <span className="text-[11px] font-bold text-slate-400">Verification Gates Passed:</span>
              <div className="flex flex-wrap gap-2 mt-2">
                {rec.gatesPassed.map((gate) => (
                  <span
                    key={gate}
                    className="px-2.5 py-1 rounded bg-slate-900 text-emerald-400 border border-emerald-500/30 text-[11px]"
                  >
                    ✓ {gate}
                  </span>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
