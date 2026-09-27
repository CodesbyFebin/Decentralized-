export type OperationalState = 'LIVE'|'DERIVED'|'CONFIGURED'|'UNAVAILABLE'|'PLANNED'|'SIMULATED'|'UNKNOWN';
export type Freshness = 'FRESH'|'STALE'|'EXPIRED'|'UNREACHABLE'|'UNKNOWN';
export interface TruthEnvelope<T> { value: T|null; source: string|null; observedAt: string|null; freshness: Freshness; state: OperationalState; }
export type GateStatus = 'PASS'|'FAIL'|'BLOCKED'|'UNKNOWN';
export type CapabilityMaturity = 'PLANNED'|'IMPLEMENTED'|'TESTED'|'QUALIFIED'|'VERIFIED'|'SEALED'|'RELEASED';
export interface ResourceDimension { total: TruthEnvelope<number>; ownerReserve: TruthEnvelope<number>; reserved: TruthEnvelope<number>; allocated: TruthEnvelope<number>; available: TruthEnvelope<number>; unit: string; }
export interface FailureDomains { node?: TruthEnvelope<string>; operator?: TruthEnvelope<string>; networkAsn?: TruthEnvelope<string>; region?: TruthEnvelope<string>; storageBackend?: TruthEnvelope<string>; power?: TruthEnvelope<string>; }
