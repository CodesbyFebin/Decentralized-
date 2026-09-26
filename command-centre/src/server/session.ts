/**
 * Operator sessions.
 *
 * Identity is the control plane's own signed capability (dhcap1…), minted with
 * `dh token` / `dh console`. The BFF keeps it in an HttpOnly, SameSite=Strict
 * cookie and forwards it as the bearer on every control-plane call. The BFF
 * decodes it only to DISPLAY who is signed in; the control plane verifies the
 * signature chain against the cluster root and enforces actions.
 */
import type { Request, Response } from 'express';
import type { Session } from '../types/reality';

export const COOKIE = 'dh_cap';
const PREFIX = 'dhcap1.';

interface Block {
  payload?: { caveats?: { actions?: string[]; expires?: number; resources?: string[] }; note?: string };
  signer?: string;
}

/** Structural decode for display. Returns null for anything that is not a dhcap1 token. */
export function decodeCapability(token: string): { actions: string[]; expiresAt: number | null; note: string; cluster: string | null } | null {
  if (!token.startsWith(PREFIX) || token.length > 16_384) return null;
  let blocks: Block[];
  try {
    blocks = JSON.parse(Buffer.from(token.slice(PREFIX.length), 'base64url').toString('utf8'));
  } catch {
    return null;
  }
  if (!Array.isArray(blocks) || blocks.length === 0 || blocks.length > 16) return null;
  // Attenuation only narrows: effective actions are the intersection across blocks.
  let actions: string[] | null = null;
  let expiresAt: number | null = null;
  let cluster: string | null = null;
  let note = '';
  for (const b of blocks) {
    const cav = b?.payload?.caveats ?? {};
    if (Array.isArray(cav.actions) && cav.actions.length) actions = actions ? actions.filter((a) => cav.actions!.includes(a)) : [...cav.actions];
    if (typeof cav.expires === 'number' && cav.expires > 0) expiresAt = expiresAt === null ? cav.expires : Math.min(expiresAt, cav.expires);
    const res = cav.resources?.find((r) => r.startsWith('cluster/'));
    if (res) cluster = res.slice('cluster/'.length);
    if (b?.payload?.note) note = b.payload.note;
  }
  return { actions: actions ?? [], expiresAt, note, cluster };
}

function parseCookies(header: string | undefined): Record<string, string> {
  const out: Record<string, string> = {};
  if (!header) return out;
  for (const part of header.split(';')) {
    const i = part.indexOf('=');
    if (i > 0) out[part.slice(0, i).trim()] = decodeURIComponent(part.slice(i + 1).trim());
  }
  return out;
}

/** Bearer header (API clients, tests) or the session cookie (browser). */
export function tokenFrom(req: Request): string | null {
  const h = req.header('authorization');
  if (h?.startsWith('Bearer ')) return h.slice(7).trim() || null;
  return parseCookies(req.header('cookie'))[COOKIE] || null;
}

export function describeSession(token: string | null): Session {
  const d = token ? decodeCapability(token) : null;
  if (!d) return { authenticated: false, actor: null, actions: [], cluster: null, expiresAt: null, readOnly: true };
  const expired = d.expiresAt !== null && d.expiresAt <= Date.now();
  return {
    authenticated: !expired,
    actor: d.note ? `operator:${d.note}` : 'operator',
    actions: d.actions,
    cluster: d.cluster,
    expiresAt: d.expiresAt,
    readOnly: !d.actions.includes('api.write') && !d.actions.includes('api.admin')
  };
}

export function setSessionCookie(res: Response, token: string, expiresAt: number | null, secure: boolean): void {
  const maxAge = expiresAt ? Math.max(0, Math.floor((expiresAt - Date.now()) / 1000)) : 12 * 3600;
  const parts = [`${COOKIE}=${encodeURIComponent(token)}`, 'Path=/api', 'HttpOnly', 'SameSite=Strict', `Max-Age=${maxAge}`];
  if (secure) parts.push('Secure');
  res.setHeader('Set-Cookie', parts.join('; '));
}

export function clearSessionCookie(res: Response, secure: boolean): void {
  res.setHeader('Set-Cookie', `${COOKIE}=; Path=/api; HttpOnly; SameSite=Strict; Max-Age=0${secure ? '; Secure' : ''}`);
}
