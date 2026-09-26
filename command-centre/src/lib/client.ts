/**
 * Single entry point for talking to the BFF. Components never call fetch
 * directly. Mutations succeed only when the BFF says the control plane
 * committed them.
 */
import type { Envelope, Provenance } from '../types/reality';

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public requestId: string | null,
    public details?: Record<string, unknown>
  ) {
    super(message);
  }
  /** The backend (not the request) is the problem: data should lose confidence. */
  get backendDown(): boolean {
    return ['BACKEND_UNAVAILABLE', 'BACKEND_TIMEOUT', 'NETWORK'].includes(this.code);
  }
}

const BASE = '/api/v1';
let onUnauthenticated: (() => void) | null = null;
export const setUnauthenticatedHandler = (fn: () => void) => {
  onUnauthenticated = fn;
};

async function request<T>(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  let res: Response;
  try {
    res = await fetch(BASE + path, {
      method,
      credentials: 'same-origin',
      signal,
      headers: {
        Accept: 'application/json',
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
        ...(method !== 'GET' ? { 'X-DH-Console': '1' } : {})
      },
      body: body !== undefined ? JSON.stringify(body) : undefined
    });
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e;
    throw new ApiError(0, 'NETWORK', 'The Command Centre server is unreachable.', null);
  }
  let json: unknown = null;
  try {
    json = await res.json();
  } catch {
    /* non-JSON */
  }
  if (!res.ok) {
    const err = (json as { error?: { code?: string; message?: string; request_id?: string; details?: Record<string, unknown> } } | null)?.error;
    const e = new ApiError(res.status, err?.code ?? `HTTP_${res.status}`, err?.message ?? `Request failed (${res.status}).`, err?.request_id ?? res.headers.get('x-request-id'), err?.details);
    if (res.status === 401 && path !== '/session') onUnauthenticated?.();
    throw e;
  }
  return json as T;
}

export const api = {
  get: <T>(path: string, signal?: AbortSignal) => request<T>('GET', path, undefined, signal),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {}),
  del: <T>(path: string) => request<T>('DELETE', path)
};

export type EnvelopeOf<T> = Envelope<T>;
export type { Provenance };

/** Result of a mutation as committed (or refused) by the control plane. */
export interface MutationResult {
  ok: boolean;
  code: string | null;
  message: string;
  request_id: string;
  actor: string | null;
}
