import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { mapView } from '../../src/server/reality/mapView';
import { attentionItems } from '../../src/server/reality/attention';
import type { CPView } from '../../src/server/reality/cpTypes';

const load = (name: string): CPView => JSON.parse(readFileSync(new URL(`../fixtures/${name}`, import.meta.url), 'utf8'));
const snap = (v: CPView) => mapView(v, { now: v.generatedAt, state: 'LIVE', source: 'control-plane:/api/v1/view', transport: { apiTls: false, unauthenticatedRejected: true } });

test('a stale host and a lost host are named, each with its source', () => {
  const stale = attentionItems(snap(load('view-host-stale.json')));
  const s = stale.find((i) => i.kind === 'node-stale');
  assert.ok(s, 'stale host listed');
  assert.equal(s.severity, 'warning');
  assert.equal(s.source, 'signed host observation');
  assert.ok(s.observedAt, 'carries the observation time');

  const lost = attentionItems(snap(load('view-host-lost.json')));
  const l = lost.find((i) => i.kind === 'node-offline');
  assert.ok(l, 'lost host listed');
  assert.equal(l.severity, 'critical');
  assert.equal(lost[0].severity, 'critical', 'critical items first');
});

test('a healthy cluster raises no node alarms', () => {
  const items = attentionItems(snap(load('view-healthy.json')));
  assert.equal(items.filter((i) => i.kind.startsWith('node-') && i.kind !== 'node-mode').length, 0);
});

test('an unverified audit ledger is critical', () => {
  const s = snap(load('view-healthy.json'));
  s.evidence.verification = { ...s.evidence.verification, state: 'INVALID', breakDetail: '{"seq":4}' };
  const items = attentionItems(s);
  assert.equal(items[0].kind, 'audit');
  assert.equal(items[0].severity, 'critical');
});
