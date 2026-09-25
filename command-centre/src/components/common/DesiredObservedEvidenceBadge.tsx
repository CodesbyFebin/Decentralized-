import React from 'react';
import { ShieldCheck, CheckCircle2, AlertTriangle, Fingerprint } from 'lucide-react';

interface Props {
  desired: string | number;
  observed: string | number;
  evidenceDigest?: string;
  isHealthy?: boolean;
  className?: string;
}

export const DesiredObservedEvidenceBadge: React.FC<Props> = ({
  desired,
  observed,
  evidenceDigest,
  isHealthy = true,
  className = ''
}) => {
  return (
    <div
      className={`inline-flex items-center gap-3 px-3 py-1.5 rounded-lg bg-[#0F172A]/90 border border-slate-700/60 text-xs font-mono ${className}`}
      title="Desired vs Observed runtime state with Cryptographic Evidence proof"
    >
      <div className="flex items-center gap-1.5 text-slate-400">
        <span className="text-[10px] uppercase tracking-wider text-slate-500 font-semibold">Desired</span>
        <span className="text-slate-200 font-medium">{desired}</span>
      </div>

      <span className="text-slate-600">·</span>

      <div className="flex items-center gap-1.5">
        <span className="text-[10px] uppercase tracking-wider text-slate-500 font-semibold">Observed</span>
        <div className="flex items-center gap-1">
          {isHealthy ? (
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
          ) : (
            <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
          )}
          <span className={isHealthy ? 'text-emerald-400 font-medium' : 'text-amber-400 font-medium'}>
            {observed}
          </span>
        </div>
      </div>

      {evidenceDigest && (
        <>
          <span className="text-slate-600">·</span>
          <div className="flex items-center gap-1 text-cyan-400" title={`Evidence Digest: ${evidenceDigest}`}>
            <Fingerprint className="w-3.5 h-3.5" />
            <span className="text-[11px] font-mono">{evidenceDigest.slice(0, 8)}...</span>
            <ShieldCheck className="w-3 h-3 text-emerald-400" />
          </div>
        </>
      )}
    </div>
  );
};
