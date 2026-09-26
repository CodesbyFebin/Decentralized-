/**
 * Signed validation records (`dh evidence seal`) read from an evidence
 * directory, and verified with the platform's own verifier
 * (`dh evidence verify`). Records are files the operator produced; nothing
 * here derives or rewrites an outcome. FAIL records are listed like any
 * other.
 */
import { execFile } from 'node:child_process';
import { readFile, readdir, realpath, stat } from 'node:fs/promises';
import path from 'node:path';
import type { ValidationRecord, ValidationVerification } from '../types/reality';

export const RECORD_ID = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const MAX_LOG_BYTES = 256 * 1024;

export class EvidenceStoreError extends Error {
  constructor(
    readonly code: 'NOT_FOUND' | 'NOT_SUPPORTED' | 'VALIDATION_FAILED' | 'BACKEND_ERROR',
    message: string,
    readonly status: number
  ) {
    super(message);
  }
}

type Raw = Record<string, unknown>;
const str = (v: unknown) => (typeof v === 'string' ? v : '');
const num = (v: unknown) => (typeof v === 'number' && Number.isFinite(v) ? v : null);
const strs = (v: unknown) => (Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []);

export function parseRecord(dirName: string, raw: unknown): ValidationRecord | null {
  if (!raw || typeof raw !== 'object') return null;
  const env = raw as Raw;
  const p = env.payload as Raw | undefined;
  if (!p || typeof p !== 'object' || env.kind !== 'validation-record') return null;
  const outcome = str(p.outcome);
  return {
    id: str(p.evidenceId) || dirName,
    dir: dirName,
    stage: str(p.stage),
    attempt: str(p.attemptId),
    parent: str(p.parentEvidenceId) || null,
    outcome: outcome === 'PASS' || outcome === 'FAIL' || outcome === 'INFRA_FAILURE' ? outcome : 'UNKNOWN',
    outcomeReason: str(p.outcomeReason),
    claim: str(p.claim),
    scope: str(p.scope),
    exclusions: strs(p.scopeExclusions),
    limitations: strs(p.limitations),
    commit: str(p.commit) || null,
    sourceDigest: str(p.sourceDigest) || null,
    digestVersion: str(p.digestVersion),
    schemaVersion: str(p.schemaVersion),
    signer: str(env.signer),
    signerPub: str(env.pub),
    signature: str(env.sig),
    startedAt: num(p.start),
    endedAt: num(p.end),
    requiredSteps: strs(p.requiredSteps),
    steps: (Array.isArray(p.steps) ? (p.steps as Raw[]) : []).map((s) => ({
      name: str(s.name),
      exit: num(s.exit),
      attempts: num(s.attempts) ?? 0,
      durationMs: num(s.durationMs),
      command: strs(s.command),
      log: str(s.log) || null,
      logHash: str(s.logHash) || null
    })),
    binaries: Object.fromEntries(Object.entries((p.binaries as Raw) ?? {}).map(([k, v]) => [k, str(v)])),
    files: Array.isArray(p.files) ? p.files.length : 0
  };
}

export class EvidenceStore {
  private verifications = new Map<string, ValidationVerification>();

  constructor(
    readonly dir: string | null,
    private readonly cli: string | null
  ) {}

  get configured() {
    return !!this.dir;
  }

  get canVerify() {
    return !!this.dir && !!this.cli;
  }

  private need(): string {
    if (!this.dir) throw new EvidenceStoreError('NOT_SUPPORTED', 'No evidence directory is configured (DH_EVIDENCE_DIR).', 409);
    return this.dir;
  }

  /** Resolves a record directory, refusing anything outside the evidence root. */
  private async recordDir(id: string): Promise<string> {
    const root = this.need();
    if (!RECORD_ID.test(id)) throw new EvidenceStoreError('VALIDATION_FAILED', 'Malformed evidence id.', 422);
    const rootReal = await realpath(root);
    let real: string;
    try {
      real = await realpath(path.join(rootReal, id));
    } catch {
      throw new EvidenceStoreError('NOT_FOUND', `No evidence record "${id}".`, 404);
    }
    if (path.dirname(real) !== rootReal) throw new EvidenceStoreError('NOT_FOUND', `No evidence record "${id}".`, 404);
    return real;
  }

  async list(): Promise<ValidationRecord[]> {
    const root = this.need();
    let names: string[];
    try {
      names = (await readdir(root, { withFileTypes: true })).filter((d) => d.isDirectory() && RECORD_ID.test(d.name)).map((d) => d.name);
    } catch {
      throw new EvidenceStoreError('BACKEND_ERROR', 'The evidence directory cannot be read.', 503);
    }
    const out: ValidationRecord[] = [];
    for (const name of names) {
      try {
        const rec = parseRecord(name, JSON.parse(await readFile(path.join(root, name, 'record.json'), 'utf8')));
        if (rec) out.push(rec);
      } catch {
        // A directory without a record (for example an infrastructure failure
        // that produced none) is not a record; INDEX.md describes it.
      }
    }
    return out.sort((a, b) => (b.endedAt ?? 0) - (a.endedAt ?? 0));
  }

  async get(id: string): Promise<ValidationRecord> {
    const dir = await this.recordDir(id);
    let raw: unknown;
    try {
      raw = JSON.parse(await readFile(path.join(dir, 'record.json'), 'utf8'));
    } catch {
      throw new EvidenceStoreError('NOT_FOUND', `No evidence record "${id}".`, 404);
    }
    const rec = parseRecord(path.basename(dir), raw);
    if (!rec) throw new EvidenceStoreError('NOT_FOUND', `"${id}" is not a validation record.`, 404);
    return rec;
  }

  /** The signed record exactly as stored (for export). */
  async raw(id: string): Promise<string> {
    const dir = await this.recordDir(id);
    try {
      return await readFile(path.join(dir, 'record.json'), 'utf8');
    } catch {
      throw new EvidenceStoreError('NOT_FOUND', `No evidence record "${id}".`, 404);
    }
  }

  lastVerification(id: string): ValidationVerification | null {
    return this.verifications.get(id) ?? null;
  }

  /** A step's log, only for a step named in the record. */
  async stepLog(id: string, step: string): Promise<{ text: string; truncated: boolean }> {
    const rec = await this.get(id);
    const s = rec.steps.find((x) => x.name === step);
    if (!s?.log) throw new EvidenceStoreError('NOT_FOUND', `No log for step "${step}".`, 404);
    const dir = await this.recordDir(id);
    const file = path.resolve(dir, s.log);
    if (!file.startsWith(dir + path.sep)) throw new EvidenceStoreError('NOT_FOUND', 'Log path is outside the record.', 404);
    const size = (await stat(file)).size;
    const buf = await readFile(file);
    return { text: buf.subarray(Math.max(0, buf.length - MAX_LOG_BYTES)).toString('utf8'), truncated: size > MAX_LOG_BYTES };
  }

  /** Runs the platform verifier; the result is whatever it reports. */
  async verify(id: string): Promise<ValidationVerification> {
    if (!this.cli) throw new EvidenceStoreError('NOT_SUPPORTED', 'No dh CLI is configured (DH_CLI), so records cannot be verified here.', 409);
    const dir = await this.recordDir(id);
    const res = await new Promise<{ code: number | null; out: string }>((resolve) => {
      execFile(this.cli!, ['evidence', 'verify', '--dir', dir], { timeout: 120_000, maxBuffer: 1 << 20 }, (err, stdout, stderr) => {
        // A non-zero exit sets err.code to the exit status; a spawn failure sets a string.
        const code = !err ? 0 : typeof (err as { code?: unknown }).code === 'number' ? (err as { code: number }).code : null;
        resolve({ code, out: `${stdout}\n${stderr}` });
      });
    });
    const line = res.out.split('\n').find((l) => l.startsWith('verification:')) ?? '';
    const detail = line.replace(/^verification:\s*/, '').trim();
    const v: ValidationVerification = {
      state: res.code === 0 && /^VERIFIED\b/.test(detail) ? 'VERIFIED' : res.code === 1 && /^UNVERIFIED\b/.test(detail) ? 'UNVERIFIED' : 'ERROR',
      detail: detail || (res.code === null ? 'verifier did not run' : `verifier exited ${res.code}`),
      verifiedAt: Date.now(),
      verifier: `${path.basename(this.cli)} evidence verify`
    };
    this.verifications.set(id, v);
    return v;
  }
}
