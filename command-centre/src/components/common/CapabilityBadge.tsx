import React from 'react';
import { CapabilityState } from '../../types/platform';

interface Props {
  state: CapabilityState;
  className?: string;
}

export const CapabilityBadge: React.FC<Props> = ({ state, className = '' }) => {
  const configs: Record<CapabilityState, { label: string; bg: string; text: string; dot: string }> = {
    LIVE: {
      label: 'LIVE',
      bg: 'bg-emerald-950/40 border-emerald-500/30',
      text: 'text-emerald-400',
      dot: 'bg-emerald-400'
    },
    DERIVED: {
      label: 'DERIVED',
      bg: 'bg-blue-950/40 border-blue-500/30',
      text: 'text-blue-400',
      dot: 'bg-blue-400'
    },
    CONFIGURED: {
      label: 'CONFIGURED',
      bg: 'bg-purple-950/40 border-purple-500/30',
      text: 'text-purple-400',
      dot: 'bg-purple-400'
    },
    UNAVAILABLE: {
      label: 'UNAVAILABLE',
      bg: 'bg-slate-900/60 border-slate-700/40',
      text: 'text-slate-400',
      dot: 'bg-slate-500'
    },
    PLANNED: {
      label: 'PLANNED',
      bg: 'bg-amber-950/40 border-amber-500/30',
      text: 'text-amber-400',
      dot: 'bg-amber-400'
    },
    SIMULATED: {
      label: 'SIMULATED',
      bg: 'bg-cyan-950/40 border-cyan-500/30',
      text: 'text-cyan-400',
      dot: 'bg-cyan-400'
    },
    UNKNOWN: {
      label: 'UNKNOWN',
      bg: 'bg-rose-950/40 border-rose-500/30',
      text: 'text-rose-400',
      dot: 'bg-rose-400'
    }
  };

  const c = configs[state] || configs.UNKNOWN;

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-mono uppercase tracking-wider border ${c.bg} ${c.text} ${className}`}
      title={`Data Truthfulness State: ${c.label}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot}`} />
      {c.label}
    </span>
  );
};
