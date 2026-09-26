import { test } from 'node:test';
import assert from 'node:assert/strict';
import { cpSync, existsSync, mkdtempSync, appendFileSync, symlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { EvidenceStore, EvidenceStoreError } from '../../src/server/evidenceStore';

// The repository's own sealed records are the fixtures: nothing here is made up.
const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const evidenceDir = path.join(repo, 'evidence');
const cli = [path.join(repo, 'evidence/NODE-A01-A02/bin/dh'), path.join(repo, 'bin/dh')].find(existsSync) ?? null;

test('records are read as sealed, FAIL included', async () => {
  const store = new EvidenceStore(evidenceDir, null);
  const recs = await store.list();
  const a01 = recs.find((r) => r.id === 'NODE-A01-A01');
  const a02 = recs.find((r) => r.id === 'NODE-A01-A02');
  assert.ok(a01 && a02, 'NODE-A01 records present');
  assert.equal(a01.outcome, 'FAIL');
  assert.match(a01.outcomeReason, /gossip-repeat/);
  assert.equal(a02.outcome, 'PASS');
  assert.equal(a02.parent, 'NODE-A01-A01');
  assert.equal(a02.steps.length, 15);
  assert.ok(a02.sourceDigest?.startsWith('b3:'));
  assert.ok(a02.signer.startsWith('dh1'));
  // Newest first.
  assert.ok((recs[0].endedAt ?? 0) >= (recs[recs.length - 1].endedAt ?? 0));
});

test('ids cannot escape the evidence directory', async () => {
  const store = new EvidenceStore(evidenceDir, null);
  for (const bad of ['..', '../pkg', 'a/b', '', '.hidden/../x']) {
    await assert.rejects(store.get(bad), (e: unknown) => e instanceof EvidenceStoreError && (e.code === 'VALIDATION_FAILED' || e.code === 'NOT_FOUND'));
  }
  const tmp = mkdtempSync(path.join(tmpdir(), 'ev-'));
  symlinkSync(path.join(repo, 'pkg'), path.join(tmp, 'escape'));
  await assert.rejects(new EvidenceStore(tmp, null).get('escape'), (e: unknown) => e instanceof EvidenceStoreError && e.code === 'NOT_FOUND');
});

test('a step log is served only for a step the record names', async () => {
  const store = new EvidenceStore(evidenceDir, null);
  const log = await store.stepLog('NODE-A01-A01', 'gossip-repeat');
  assert.match(log.text, /join contacted 0/);
  await assert.rejects(store.stepLog('NODE-A01-A01', '../../record.json'), (e: unknown) => e instanceof EvidenceStoreError && e.code === 'NOT_FOUND');
});

test('without a CLI, verification is refused rather than assumed', async () => {
  await assert.rejects(new EvidenceStore(evidenceDir, null).verify('NODE-A01-A02'), (e: unknown) => e instanceof EvidenceStoreError && e.code === 'NOT_SUPPORTED');
  assert.equal(new EvidenceStore(evidenceDir, null).lastVerification('NODE-A01-A02'), null);
});

test('verification runs the platform verifier; a tampered copy is UNVERIFIED', { skip: cli ? false : 'no dh binary in this checkout' }, async () => {
  const good = await new EvidenceStore(evidenceDir, cli).verify('NODE-A01-A02');
  assert.equal(good.state, 'VERIFIED');
  const tmp = mkdtempSync(path.join(tmpdir(), 'ev-'));
  cpSync(path.join(evidenceDir, 'CC-RC1-A02'), path.join(tmp, 'CC-RC1-A02'), { recursive: true });
  appendFileSync(path.join(tmp, 'CC-RC1-A02/logs/02-cc-typecheck.log'), 'edited\n');
  const store = new EvidenceStore(tmp, cli);
  const bad = await store.verify('CC-RC1-A02');
  assert.equal(bad.state, 'UNVERIFIED');
  assert.match(bad.detail, /02-cc-typecheck\.log/);
  assert.equal(store.lastVerification('CC-RC1-A02')?.state, 'UNVERIFIED');
});
