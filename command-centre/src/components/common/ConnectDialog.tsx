import React, { useState } from 'react';
import { X, Copy, Check, Terminal, Info } from 'lucide-react';

export interface ConnectTarget {
  id: string;
  title: string;
  subtitle: string;
  /** Host name suggestion used in the generated commands. */
  hostName: string;
  edge?: boolean;
  arch?: 'amd64' | 'arm64';
  note?: string;
  /** Extra step shown after the join (e.g. mount a disk as a storage volume). */
  extra?: { label: string; code?: string };
  /** When set, the backend has no adapter for this target yet: explain instead of showing commands. */
  unavailable?: string;
}

/**
 * Host enrolment instructions. Commands mirror docs/runbooks/install.md:
 * single-use invite → dh-noded --join-file → operator approval.
 */
export const ConnectDialog: React.FC<{ target: ConnectTarget | null; onClose: () => void }> = ({ target, onClose }) => {
  const [copied, setCopied] = useState<number | null>(null);
  if (!target) return null;

  const token = `${target.hostName}.token`;
  const steps: { label: string; code?: string }[] = [
    {
      label: 'On your operator machine, create a single-use join token',
      code: `dh node invite --out ${token}${target.edge ? ' --roles edge' : ''}`
    },
    ...(target.arch === 'arm64'
      ? [{ label: 'Build the host agent for ARM64', code: 'GOOS=linux GOARCH=arm64 make build' }]
      : []),
    { label: `Copy bin/dh-noded and ${token} to the machine, then start the agent` , code: `dh-noded --data /var/lib/dh-noded --join-file ${token} \\\n  --name ${target.hostName} --region <region> --zone a \\\n  --mesh 0.0.0.0:51820 --mesh-advertise <public-host>:51820` },
    { label: 'Approve the pending host (skip if you used --auto)', code: `dh node approve ${target.hostName}` },
    ...(target.extra ? [target.extra] : [])
  ];

  const copy = (i: number, text: string) => {
    navigator.clipboard?.writeText(text.replace(/\\\n\s*/g, '')).catch(() => {});
    setCopied(i);
    setTimeout(() => setCopied(null), 1500);
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4" onClick={onClose} role="dialog" aria-modal="true" aria-label={target.title}>
      <div className="w-full max-w-xl dh-glass-strong rounded-2xl p-5" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-start justify-between gap-3">
          <div>
            <div className="text-[11px] uppercase tracking-widest text-cyan-300 font-semibold">Connect host</div>
            <h3 className="text-lg font-bold text-white mt-0.5">{target.title}</h3>
            <p className="text-[12.5px] text-slate-400">{target.subtitle}</p>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white" aria-label="Close">
            <X className="w-4 h-4" />
          </button>
        </div>

        {target.unavailable ? (
          <div className="mt-4 rounded-xl border border-amber-400/30 bg-amber-500/10 p-3 text-[12.5px] text-amber-100">
            <div className="font-semibold text-amber-300 text-[11px] uppercase tracking-wider mb-1">Not available yet</div>
            {target.unavailable}
          </div>
        ) : (
        <ol className="mt-4 space-y-3">
          {steps.map((s, i) => (
            <li key={i} className="flex gap-3">
              <span className="w-6 h-6 rounded-full bg-blue-600/30 border border-cyan-400/40 text-cyan-200 text-[11px] font-bold flex items-center justify-center shrink-0">
                {i + 1}
              </span>
              <div className="flex-1 min-w-0">
                <div className="text-[12.5px] text-slate-200">{s.label}</div>
                {s.code && (
                  <div className="mt-1.5 flex items-start gap-2 rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.16)] px-3 py-2">
                    <Terminal className="w-3.5 h-3.5 text-cyan-400 mt-0.5 shrink-0" />
                    <pre className="flex-1 text-[11.5px] text-cyan-100 whitespace-pre-wrap break-all font-mono">{s.code}</pre>
                    <button onClick={() => copy(i, s.code!)} className="text-slate-400 hover:text-white shrink-0" aria-label="Copy command">
                      {copied === i ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                    </button>
                  </div>
                )}
              </div>
            </li>
          ))}
        </ol>
        )}

        <div className="mt-4 flex gap-2 rounded-xl bg-white/[0.03] border border-[rgba(125,190,255,0.12)] p-3 text-[11.5px] text-slate-400">
          <Info className="w-4 h-4 text-cyan-300 shrink-0" />
          <span>
            {target.note ??
              'On first start the host writes its own policy.yaml (accepted tiers, runtimes, resource caps). The control plane can read that policy but never change it.'}
          </span>
        </div>
      </div>
    </div>
  );
};
