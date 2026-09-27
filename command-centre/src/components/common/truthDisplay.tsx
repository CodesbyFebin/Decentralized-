import React from 'react';
import { AlertCircle, Eye, EyeOff } from 'lucide-react';
import type { TruthState } from '../../types/reality';

/**
 * Truth Envelope: carries operational value with source/freshness/state metadata
 * Ensures UNKNOWN is never rendered as zero, UNAVAILABLE never as healthy, STALE never as LIVE
 */
export interface TruthEnvelope<T> {
  value: T | null;
  source?: string;
  observedAt?: string;
  freshness: 'LIVE' | 'STALE' | 'UNAVAILABLE' | 'UNKNOWN';
  state?: TruthState;
}

/** TruthValue: display operational values with freshness/source semantics */
export const TruthValue: React.FC<{
  label: string;
  envelope: TruthEnvelope<any>;
  format?: (v: any) => string;
  className?: string;
}> = ({ label, envelope, format = (v) => String(v), className = '' }) => {
  const isMissing = envelope.value === null;
  const displayValue = isMissing
    ? envelope.freshness === 'UNAVAILABLE'
      ? 'Unavailable'
      : 'Unknown'
    : format(envelope.value);

  const freshnessTone =
    envelope.freshness === 'LIVE'
      ? 'text-emerald-400'
      : envelope.freshness === 'STALE'
        ? 'text-amber-400'
        : 'text-slate-400';

  return (
    <div className={`py-1.5 text-[12.5px] border-b border-[rgba(125,190,255,0.06)] last:border-0 ${className}`}>
      <div className="flex justify-between gap-4 items-center">
        <span className="text-slate-400">{label}</span>
        <div className="flex items-center gap-1.5 text-right">
          <span className="text-slate-100 break-all">{displayValue}</span>
          <div className="flex items-center gap-1">
            {envelope.source && <span className="text-[10px] text-slate-500 font-mono">{envelope.source}</span>}
            <span className={`text-[10px] font-mono ${freshnessTone}`}>{envelope.freshness}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

/** ResourceDimension: represents MODEL A constraint for a resource (CPU, memory, etc.) */
export interface ResourceDimension {
  total: TruthEnvelope<number>;
  ownerReserve: TruthEnvelope<number>;
  reserved: TruthEnvelope<number>;
  allocated: TruthEnvelope<number>;
  unit: string;
}

/**
 * ResourceLedger: enforces MODEL A invariant
 * AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
 * Allocation consumes reservation (reserved decreases as allocated increases)
 */
export const calculateResourceAvailable = (dim: ResourceDimension): TruthEnvelope<number> => {
  if (
    dim.total.value === null ||
    dim.ownerReserve.value === null ||
    dim.reserved.value === null ||
    dim.allocated.value === null
  ) {
    return {
      value: null,
      freshness: 'UNKNOWN',
      source: 'calculation'
    };
  }

  const available = dim.total.value - dim.ownerReserve.value - dim.reserved.value - dim.allocated.value;
  const freshness =
    dim.total.freshness === 'LIVE' &&
    dim.ownerReserve.freshness === 'LIVE' &&
    dim.reserved.freshness === 'LIVE' &&
    dim.allocated.freshness === 'LIVE'
      ? 'LIVE'
      : 'STALE';

  return {
    value: Math.max(0, available),
    freshness,
    source: 'calculation'
  };
};

/** ResourceLedgerCard: display MODEL A constraint with visual breakdown */
export const ResourceLedgerCard: React.FC<{
  label: string;
  dimension: ResourceDimension;
  className?: string;
}> = ({ label, dimension, className = '' }) => {
  const available = calculateResourceAvailable(dimension);
  const total = dimension.total.value ?? 0;

  const pctOwner = total > 0 ? ((dimension.ownerReserve.value ?? 0) / total) * 100 : 0;
  const pctReserved = total > 0 ? ((dimension.reserved.value ?? 0) / total) * 100 : 0;
  const pctAllocated = total > 0 ? ((dimension.allocated.value ?? 0) / total) * 100 : 0;
  const pctAvailable = total > 0 ? ((available.value ?? 0) / total) * 100 : 0;

  return (
    <div className={`p-4 rounded-2xl bg-slate-900/60 border border-blue-500/20 ${className}`}>
      <div className="mb-3 flex justify-between items-center">
        <div>
          <div className="text-[11px] uppercase font-mono text-slate-400">{label}</div>
          <div className="text-sm font-bold text-white font-mono mt-1">
            {available.value ?? '?'} {dimension.unit} available
          </div>
        </div>
        <div className={`text-[10px] font-mono ${available.freshness === 'LIVE' ? 'text-emerald-400' : 'text-slate-400'}`}>
          {available.freshness}
        </div>
      </div>

      {/* Visual constraint breakdown */}
      <div className="mb-3 flex h-6 rounded-lg overflow-hidden bg-slate-800/50 border border-slate-700/30">
        {pctOwner > 0 && (
          <div
            className="bg-rose-600/70 hover:bg-rose-500/70 transition-colors flex items-center justify-center text-[9px] font-bold text-white"
            style={{ width: `${pctOwner}%` }}
            title={`Owner Reserve: ${dimension.ownerReserve.value} ${dimension.unit}`}
          >
            {pctOwner > 5 && 'Owner'}
          </div>
        )}
        {pctReserved > 0 && (
          <div
            className="bg-amber-600/70 hover:bg-amber-500/70 transition-colors flex items-center justify-center text-[9px] font-bold text-white"
            style={{ width: `${pctReserved}%` }}
            title={`Reserved: ${dimension.reserved.value} ${dimension.unit}`}
          >
            {pctReserved > 5 && 'Reserved'}
          </div>
        )}
        {pctAllocated > 0 && (
          <div
            className="bg-blue-600/70 hover:bg-blue-500/70 transition-colors flex items-center justify-center text-[9px] font-bold text-white"
            style={{ width: `${pctAllocated}%` }}
            title={`Allocated: ${dimension.allocated.value} ${dimension.unit}`}
          >
            {pctAllocated > 5 && 'Allocated'}
          </div>
        )}
        {pctAvailable > 0 && (
          <div
            className="bg-emerald-600/70 hover:bg-emerald-500/70 transition-colors flex items-center justify-center text-[9px] font-bold text-white"
            style={{ width: `${pctAvailable}%` }}
            title={`Available: ${available.value} ${dimension.unit}`}
          >
            {pctAvailable > 5 && 'Available'}
          </div>
        )}
      </div>

      {/* Constraint breakdown details */}
      <div className="space-y-1.5 text-[11px]">
        <div className="flex justify-between">
          <span className="text-slate-400">Total</span>
          <span className="text-slate-100 font-mono">{dimension.total.value ?? '?'} {dimension.unit}</span>
        </div>
        <div className="flex justify-between text-rose-400">
          <span>Owner Reserve</span>
          <span className="font-mono">{dimension.ownerReserve.value ?? '?'} {dimension.unit}</span>
        </div>
        <div className="flex justify-between text-amber-400">
          <span>Reserved</span>
          <span className="font-mono">{dimension.reserved.value ?? '?'} {dimension.unit}</span>
        </div>
        <div className="flex justify-between text-blue-400">
          <span>Allocated</span>
          <span className="font-mono">{dimension.allocated.value ?? '?'} {dimension.unit}</span>
        </div>
        <div className="flex justify-between pt-1.5 border-t border-slate-700/30 text-emerald-400 font-bold">
          <span>Available</span>
          <span className="font-mono">{available.value ?? '?'} {dimension.unit}</span>
        </div>
      </div>

      {/* Freshness indicator */}
      {available.freshness !== 'LIVE' && (
        <div className="mt-3 p-2 rounded-lg bg-slate-800/50 border border-slate-700/30 flex items-center gap-2 text-[10px] text-slate-400">
          <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
          <span>Values are {available.freshness.toLowerCase()}; refresh for current state</span>
        </div>
      )}
    </div>
  );
};

/** CordonCard: display cordon state orthogonal to lifecycle */
export const CordonCard: React.FC<{
  cordoned: boolean;
  lifecycle: string;
  onCordon?: () => void;
  onUncordon?: () => void;
  disabled?: boolean;
  className?: string;
}> = ({ cordoned, lifecycle, onCordon, onUncordon, disabled, className = '' }) => {
  const canModify = !disabled && (lifecycle === 'ACTIVE' || lifecycle === 'DRAINING');

  return (
    <div className={`p-4 rounded-2xl bg-slate-900/60 border ${cordoned ? 'border-amber-500/30' : 'border-blue-500/20'} ${className}`}>
      <div className="flex items-start justify-between">
        <div>
          <div className="text-[11px] uppercase font-mono text-slate-400">Admission Control</div>
          <div className="mt-2 flex items-center gap-2">
            {cordoned ? (
              <span className="px-2 py-1 rounded-lg bg-amber-500/20 text-amber-300 text-[11px] font-mono border border-amber-500/30">
                CORDONED
              </span>
            ) : (
              <span className="px-2 py-1 rounded-lg bg-emerald-500/20 text-emerald-300 text-[11px] font-mono border border-emerald-500/30">
                OPEN
              </span>
            )}
            <span className="text-[11px] text-slate-400">
              {cordoned
                ? 'New workloads rejected; existing workloads continue'
                : 'New workloads accepted; existing workloads running'}
            </span>
          </div>
        </div>

        {canModify && (
          <button
            onClick={cordoned ? onUncordon : onCordon}
            disabled={disabled}
            className={`px-3 py-1.5 rounded-lg text-[11px] font-semibold transition-all ${
              cordoned
                ? 'bg-emerald-600/20 text-emerald-300 hover:bg-emerald-600/40 border border-emerald-500/30 disabled:opacity-50'
                : 'bg-amber-600/20 text-amber-300 hover:bg-amber-600/40 border border-amber-500/30 disabled:opacity-50'
            }`}
          >
            {cordoned ? 'Uncordon' : 'Cordon'}
          </button>
        )}
      </div>

      <div className="mt-3 p-2 rounded-lg bg-slate-800/50 border border-slate-700/30 text-[10px] text-slate-400">
        <strong>Note:</strong> Cordon is orthogonal to lifecycle state. {lifecycle === 'REVOKED' && 'This node is revoked and cannot receive workloads regardless of cordon state.'}
      </div>
    </div>
  );
};
