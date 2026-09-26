import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { mapView } from '../../src/server/reality/mapView';
import { deriveOperations } from '../../src/server/reality/operations';
import { deploymentProgress, generationsOf } from '../../src/server/bff';
import type { CPView } from '../../src/server/reality/cpTypes';

const load = (name: string): CPView => JSON.parse(readFileSync(new URL(`../fixtures/${name}`, import.meta.url), 'utf8'));
const snap = (v: CPView) => mapView(v, { now: v.generatedAt, state: 'LIVE', source: 'control-plane:/api/v1/view', transport: { apiTls: false, unauthenticatedRejected: true } });
const ops = (s: ReturnType<typeof snap>) => {
  const by = new Map(s.apps.map((a) => [a.name, a]));
  return deriveOperations(s, (n, g) => deploymentProgress(by.get(n)!, g), (n) => generationsOf(by.get(n)!));
};

test('a converged deployment is an operation that SUCCEEDED, with its source', () => {
  const o = ops(snap(load('view-healthy.json')));
  const d = o.find((x) => x.kind === 'deployment' && x.target === 'web');
  assert.ok(d);
  assert.equal(d.status, 'SUCCEEDED');
  assert.equal(d.source, 'replica rows for this generation');
  assert.ok(d.steps.length >= 4);
});

test('a draining host with replicas left is RUNNING, and WAITING once empty', () => {
  const s = snap(load('view-healthy.json'));
  const host = s.nodes.find((n) => s.apps.some((a) => a.replicas.some((r) => r.node === n.id && r.desired === 'RUNNING')))!;
  host.lifecycle = 'DRAINING';
  const running = ops(s).find((x) => x.kind === 'drain');
  assert.equal(running?.status, 'RUNNING');
  for (const a of s.apps) a.replicas = a.replicas.filter((r) => r.node !== host.id);
  assert.equal(ops(s).find((x) => x.kind === 'drain')?.status, 'WAITING');
});

test('no operation is invented when nothing is in progress', () => {
  const o = ops(snap(load('view-healthy.json')));
  assert.equal(o.filter((x) => x.status === 'RUNNING' || x.status === 'QUEUED').length, 0);
  assert.ok(o.every((x) => x.kind === 'deployment'));
});
