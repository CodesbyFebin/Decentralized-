import type { AdapterMode, AuditEntryRec, InviteRec, LedgerVerification, NodeOperation, RealitySnapshot } from '../../types/reality';

/** Per-request context: the caller's own capability is forwarded, never a BFF-held credential. */
export interface RequestCtx {
  token: string | null;
  requestId: string;
  signal?: AbortSignal;
}

export type AdapterErrorCode =
  | 'UNAUTHENTICATED'
  | 'PERMISSION_DENIED'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'VALIDATION_FAILED'
  | 'BACKEND_UNAVAILABLE'
  | 'BACKEND_TIMEOUT'
  | 'BACKEND_MALFORMED'
  | 'BACKEND_ERROR'
  | 'NOT_SUPPORTED';

export class AdapterError extends Error {
  constructor(
    public code: AdapterErrorCode,
    message: string,
    public status: number,
    public details?: Record<string, unknown>
  ) {
    super(message);
  }
}

/** Outcome of a mutation as decided by the control plane (Raft-committed or refused). */
export interface OperationResult {
  ok: boolean;
  code: string | null;
  message: string;
  /** Command-specific result, e.g. the committed generation for an apply. */
  data?: unknown;
}

export interface BackendHealth {
  reachable: boolean;
  detail: string;
  cluster?: string;
  leader?: string;
  member?: string;
  state?: string;
}

export interface PlatformAdapter {
  readonly mode: AdapterMode;
  /** Source label used in provenance, e.g. "control-plane:http://127.0.0.1:17701". */
  readonly source: string;
  health(signal?: AbortSignal): Promise<BackendHealth>;
  /** true when the backend rejects an unauthenticated operator call (it must). */
  probeUnauthenticatedRejected(signal?: AbortSignal): Promise<boolean | null>;
  snapshot(ctx: RequestCtx): Promise<RealitySnapshot>;
  auditPage(ctx: RequestCtx, from: number, limit: number): Promise<{ head: number; entries: AuditEntryRec[] }>;
  verifyAudit(ctx: RequestCtx): Promise<LedgerVerification>;
  nodeLogs(ctx: RequestCtx, nodeId: string, assignment?: string, tail?: number): Promise<string[]>;
  nodeOperation(ctx: RequestCtx, nodeId: string, op: NodeOperation, reason?: string): Promise<OperationResult>;
  applyManifest(ctx: RequestCtx, yaml: string): Promise<OperationResult>;
  scaleApp(ctx: RequestCtx, app: string, replicas: number): Promise<OperationResult>;
  deleteApp(ctx: RequestCtx, app: string): Promise<OperationResult>;
  /** Join invites (admin). The root-signed join token is never held by the platform. */
  listInvites(ctx: RequestCtx): Promise<{ invites: InviteRec[]; serverTime: number }>;
  revokeInvite(ctx: RequestCtx, nonce: string): Promise<OperationResult>;
}
