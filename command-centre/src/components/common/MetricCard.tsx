import React from 'react';
import { ArrowUpRight, ArrowDownRight } from 'lucide-react';
import { CapabilityState } from '../../types/platform';
import { CapabilityBadge } from './CapabilityBadge';

interface Props {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  subValue?: string;
  change?: string;
  isPositive?: boolean;
  trendColor?: 'cyan' | 'blue' | 'purple' | 'green' | 'amber';
  capability?: CapabilityState;
  provenance?: string;
  onClick?: () => void;
}

export const MetricCard: React.FC<Props> = ({
  icon,
  label,
  value,
  subValue,
  change,
  isPositive = true,
  trendColor = 'cyan',
  capability = 'LIVE',
  provenance,
  onClick
}) => {
  const colorMap = {
    cyan: { stroke: '#06B6D4', fill: 'rgba(6, 182, 212, 0.15)', text: 'text-cyan-400' },
    blue: { stroke: '#3B82F6', fill: 'rgba(59, 130, 246, 0.15)', text: 'text-blue-400' },
    purple: { stroke: '#A855F7', fill: 'rgba(168, 85, 247, 0.15)', text: 'text-purple-400' },
    green: { stroke: '#10B981', fill: 'rgba(16, 185, 129, 0.15)', text: 'text-emerald-400' },
    amber: { stroke: '#F59E0B', fill: 'rgba(245, 158, 11, 0.15)', text: 'text-amber-400' }
  };

  const currentTheme = colorMap[trendColor];

  return (
    <div
      onClick={onClick}
      className={`relative overflow-hidden rounded-[22px] bg-[rgba(8,20,42,0.65)] backdrop-blur-xl border border-[rgba(125,190,255,0.16)] p-5 hover:border-[rgba(125,190,255,0.35)] transition-all shadow-[0_12px_28px_-8px_rgba(2,6,23,0.7)] group ${
        onClick ? 'cursor-pointer hover:bg-[rgba(12,30,60,0.75)] hover:shadow-[0_0_24px_rgba(36,139,255,0.18)]' : ''
      }`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-2xl bg-[rgba(10,24,52,0.8)] border border-[rgba(125,190,255,0.18)] shadow-inner group-hover:border-cyan-400/40 transition-colors">
            {icon}
          </div>
          <div>
            <div className="text-xs font-medium text-slate-400 font-sans">{label}</div>
            <div className="text-2xl font-bold tabular-nums whitespace-nowrap tracking-tight text-white mt-0.5">
              {value}
            </div>
          </div>
        </div>

        {capability && <CapabilityBadge state={capability} />}
      </div>

      <div className="flex items-end justify-between mt-2 pt-2 border-t border-[rgba(125,190,255,0.10)]">
        <div className="flex items-center gap-1.5">
          {change && (
            <div
              className={`flex items-center text-xs font-semibold font-mono ${
                isPositive ? 'text-emerald-400' : 'text-rose-400'
              }`}
            >
              {isPositive ? (
                <ArrowUpRight className="w-3.5 h-3.5" />
              ) : (
                <ArrowDownRight className="w-3.5 h-3.5" />
              )}
              <span>{change}</span>
            </div>
          )}
          {subValue && <span className="text-xs text-slate-400">{subValue}</span>}
          {provenance && (
            <span className="text-[10px] text-slate-500 font-mono truncate max-w-[100px]" title={provenance}>
              · {provenance}
            </span>
          )}
        </div>

        {/* Mini sparkline curve with glow */}
        <div className="w-20 h-6">
          <svg viewBox="0 0 80 24" className="w-full h-full overflow-visible">
            <defs>
              <linearGradient id={`grad-${label.replace(/\s+/g, '')}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={currentTheme.stroke} stopOpacity="0.4" />
                <stop offset="100%" stopColor={currentTheme.stroke} stopOpacity="0.0" />
              </linearGradient>
            </defs>
            <path
              d="M 0 18 Q 20 6, 40 14 T 80 4"
              fill="none"
              stroke={currentTheme.stroke}
              strokeWidth="2"
            />
            <path
              d="M 0 18 Q 20 6, 40 14 T 80 4 L 80 24 L 0 24 Z"
              fill={`url(#grad-${label.replace(/\s+/g, '')})`}
            />
          </svg>
        </div>
      </div>
    </div>
  );
};
