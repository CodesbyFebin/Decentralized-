/**
 * Development-only adapter: replays a view recorded from a real dev cluster
 * (`dh dev up`). Everything it returns is labelled SIMULATED, and it refuses
 * every mutation instead of pretending one happened.
 *
 * It is only constructed when PLATFORM_ADAPTER=demo, and never in production
 * unless ALLOW_DEMO_IN_PRODUCTION=1 (see config.ts). There is no fallback path
 * from the control-plane adapter to this one.
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import type { AuditEntryRec, InviteRec, LedgerVerification, RealitySnapshot } from '../../types/reality';
import { mapView } from '../reality/mapView';
import { isCPView, type CPView } from '../reality/cpTypes';
import { AdapterError, type BackendHealth, type OperationResult, type PlatformAdapter, type RequestCtx } from './types';

const FIXTURE = fileURLToPath(new URL('../demo/fixtures/dev-cluster-view.json', import.meta.url));

export class DemoPlatformAdapter implements PlatformAdapter {
  readonly mode = 'demo' as const;
  readonly source = 'demo-replay:dev-cluster-view.json';
  private view: CPView;

  constructor(private regionLocations?: Record<string, [number, number]>, fixturePath = FIXTURE) {
    const raw = JSON.parse(readFileSync(fixturePath, 'utf8'));
    if (!isCPView(raw)) throw new Error('demo fixture is not a control-plane view');
    this.view = raw;
  }

  async health(): Promise<BackendHealth> {
    return { reachable: true, detail: 'demo replay of a recorded view (SIMULATED)', cluster: this.view.cluster?.name, state: 'demo' };
  }

  async probeUnauthenticatedRejected(): Promise<boolean | null> {
    return null;
  }

  async snapshot(_ctx: RequestCtx): Promise<RealitySnapshot> {
    return mapView(this.view, {
      now: this.view.generatedAt,
      state: 'SIMULATED',
      source: this.source,
      regionLocations: this.regionLocations,
      transport: undefined
    });
  }

  async auditPage(_ctx: RequestCtx, from: number, limit: number): Promise<{ head: number; entries: AuditEntryRec[] }> {
    const s = await this.snapshot(_ctx);
    const asc = [...s.evidence.entries].reverse().filter((e) => e.seq > from).slice(0, limit);
    return { head: s.evidence.head, entries: asc };
  }

  async verifyAudit(_ctx: RequestCtx): Promise<LedgerVerification> {
    // A replay cannot re-verify anything; say so rather than echo "VERIFIED".
    const s = await this.snapshot(_ctx);
    return { ...s.evidence.verification, state: 'UNKNOWN', verifiedAt: null, verifiedBy: '', breakDetail: 'demo replay: the ledger was verified when recorded, not now' };
  }

  async nodeLogs(): Promise<string[]> {
    throw new AdapterError('NOT_SUPPORTED', 'Logs are not available in the demo replay.', 409);
  }

  private refuse(): never {
    throw new AdapterError('NOT_SUPPORTED', 'The demo replay is read-only; connect a control plane to change anything.', 409);
  }

  async nodeOperation(): Promise<OperationResult> {
    this.refuse();
  }
  async applyManifest(): Promise<OperationResult> {
    this.refuse();
  }
  async scaleApp(): Promise<OperationResult> {
    this.refuse();
  }
  async deleteApp(): Promise<OperationResult> {
    this.refuse();
  }
  async listInvites(): Promise<{ invites: InviteRec[]; serverTime: number }> {
    throw new AdapterError('NOT_SUPPORTED', 'Invites are not part of the demo replay.', 409);
  }
  async revokeInvite(): Promise<OperationResult> {
    this.refuse();
  }
}
