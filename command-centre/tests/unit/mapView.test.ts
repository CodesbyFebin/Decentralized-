import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { mapView, deriveNodeHealth, mapCertState, deriveAppPhase, measuredHardware } from '../../src/server/reality/mapView';
import { deriveCapabilities } from '../../src/server/reality/capabilities';
import { isCPView, type CPView } from '../../src/server/reality/cpTypes';

const load = (name: string): CPView => JSON.parse(readFileSync(new URL(`../fixtures/${name}`, import.meta.url), 'utf8'));
const healthy = load('view-healthy.json');
const stale = load('view-host-stale.json');
const lost = load('view-host-lost.json');
const opts = (v: CPView) => ({ now: v.generatedAt, state: 'LIVE' as const, source: 'control-plane:/api/v1/view', transport: { apiTls: false, unauthenticatedRejected: true } });

test('fixtures are real control-plane views', () => {
  for (const v of [healthy, stale, lost]) assert.ok(isCPView(v));
  assert.equal(isCPView({ nodes: [] }), false);
  assert.equal(isCPView(null), false);
});

test('node counts come from the node list, not constants', () => {
  const s = mapView(healthy, opts(healthy));
  assert.equal(s.nodes.length, healthy.nodes!.length);
  assert.equal(s.overview.metrics.nodesTotal.value, healthy.nodes!.length);
  assert.equal(s.overview.metrics.nodesHealthy.value, 4);
  assert.equal(s.overview.metrics.nodesHealthy.state, 'DERIVED');
});

test('stale observation degrades the node instead of staying healthy', () => {
  const s = mapView(stale, opts(stale));
  const c = s.nodes.find((n) => n.name === 'host-c')!;
  assert.equal(c.health, 'DEGRADED');
  assert.equal(c.observation.freshness, 'STALE');
  assert.ok((c.observation.ageMs ?? 0) > 10_000);
  assert.equal(s.overview.metrics.nodesHealthy.value, 3);
  assert.equal(s.overview.metrics.nodesDegraded.value, 1);
});

test('a lost host is OFFLINE and its stale replica is not counted as running', () => {
  const s = mapView(lost, opts(lost));
  const c = s.nodes.find((n) => n.name === 'host-c')!;
  assert.equal(c.health, 'OFFLINE');
  const web = s.apps.find((a) => a.name === 'web')!;
  assert.equal(web.replicas.length, 4, 'the lost row is kept, not hidden');
  assert.equal(web.desiredReplicas, 3);
  assert.equal(web.observedRunning, 3, 'rescheduled replica counts, the stale one does not');
  assert.ok(s.overview.incidents.some((i) => i.includes('host-c')));
});

test('stale replica keeps desired, admitted and observed separate', () => {
  const s = mapView(stale, opts(stale));
  const web = s.apps.find((a) => a.name === 'web')!;
  const row = web.replicas.find((r) => r.nodeName === 'host-c')!;
  assert.equal(row.desired, 'RUNNING');
  assert.equal(row.admitted, 'ADMITTED');
  assert.equal(row.observed, 'RUNNING');
  assert.equal(row.freshness, 'STALE');
  assert.equal(web.observedRunning, 2);
  assert.equal(web.phase, 'CONVERGING');
});

test('health derivation', () => {
  assert.equal(deriveNodeHealth({ status: 'ready', health: 'live', lastObs: { seq: 0, freshness: 'NONE' } }).health, 'UNKNOWN');
  assert.equal(deriveNodeHealth({ status: 'ready', health: 'live', lastObs: { seq: 1, freshness: 'FRESH' } }).health, 'HEALTHY');
  assert.equal(deriveNodeHealth({ status: 'ready', health: 'lost', lastObs: { seq: 1, freshness: 'FRESH' } }).health, 'OFFLINE');
  assert.equal(deriveNodeHealth({ status: 'revoked', health: 'live', lastObs: { seq: 1, freshness: 'FRESH' } }).health, 'OFFLINE');
});

test('app phase never claims READY without observation', () => {
  const base = { deleted: false, desiredReplicas: 3, admitted: 3, observedRunning: 0, healthyReplicas: 0, refused: 0, unknown: 3 };
  assert.equal(deriveAppPhase(base).phase, 'UNKNOWN');
  assert.equal(deriveAppPhase({ ...base, unknown: 0, observedRunning: 3, healthyReplicas: 3 }).phase, 'READY');
  assert.equal(deriveAppPhase({ ...base, unknown: 0, observedRunning: 3, healthyReplicas: 2 }).phase, 'DEGRADED');
  assert.equal(deriveAppPhase({ ...base, refused: 1 }).phase, 'REFUSED');
});

test('certificate state is computed from the observed X.509 validity', () => {
  const now = Date.UTC(2026, 8, 25);
  const day = 86_400_000;
  assert.equal(mapCertState('ISSUED', now + 60 * day, now, 14 * day), 'VALID');
  assert.equal(mapCertState('ISSUED', now + 3 * day, now, 14 * day), 'EXPIRING');
  assert.equal(mapCertState('ISSUED', now - 1, now, 14 * day), 'EXPIRED');
  assert.equal(mapCertState('ISSUED', null, now, 14 * day), 'UNKNOWN');
  assert.equal(mapCertState('PENDING', null, now, 14 * day), 'PENDING');
  assert.equal(mapCertState('FAILED', null, now, 14 * day), 'FAILED');
  assert.equal(mapCertState('whatever', null, now, 14 * day), 'UNKNOWN');
  const s = mapView(healthy, opts(healthy));
  assert.equal(s.certificates.length, 1);
  assert.equal(s.certificates[0].state, 'VALID');
  assert.match(s.certificates[0].fingerprint, /^sha256:/);
});

test('unmeasured metrics are null + UNAVAILABLE, never zero', () => {
  const s = mapView(healthy, opts(healthy));
  assert.equal(s.overview.metrics.visitors30d.value, null);
  assert.equal(s.overview.metrics.visitors30d.state, 'UNAVAILABLE');
  assert.equal(s.overview.metrics.bandwidth.value, null);
  const noEdge = { ...healthy, nodes: healthy.nodes!.map((n) => ({ ...n, edge: null })) };
  const s2 = mapView(noEdge, opts(noEdge));
  assert.equal(s2.overview.metrics.edgeRequests.value, null);
  assert.equal(s2.overview.metrics.edgeRequests.state, 'UNAVAILABLE');
});

test('no environment variable values leave the mapper', () => {
  const s = mapView(healthy, opts(healthy));
  const json = JSON.stringify(s);
  assert.ok(s.apps[0].envNames.includes('MESSAGE'));
  assert.ok(!json.includes('hello from a sovereign host'), 'env value leaked');
});

test('domains keep desired routing, observed routing and DNS separate', () => {
  const s = mapView(healthy, opts(healthy));
  const d = s.domains.find((x) => x.host === 'web.dev.test')!;
  assert.equal(d.desired.app, 'web');
  assert.equal(d.routing.state, 'LIVE');
  assert.equal(d.dns.state, 'UNAVAILABLE');
  assert.equal(d.dns.value, null);
});

test('evidence verification comes from the ledger verification result', () => {
  const s = mapView(healthy, opts(healthy));
  assert.equal(s.evidence.verification.state, 'VERIFIED');
  const broken = { ...healthy, audit: { ...healthy.audit!, verification: { ...healthy.audit!.verification!, ok: false, break: { seq: 3 } } } };
  const s2 = mapView(broken, opts(broken));
  assert.equal(s2.evidence.verification.state, 'INVALID');
  assert.equal(s2.security.find((c) => c.id === 'audit-chain')!.result, 'FAIL');
  const none = { ...healthy, audit: undefined };
  assert.equal(mapView(none, opts(none)).evidence.verification.state, 'UNKNOWN');
});

test('WAF and DDoS are UNAVAILABLE, not a vanity score', () => {
  const s = mapView(healthy, opts(healthy));
  assert.equal(s.security.find((c) => c.id === 'waf')!.result, 'UNAVAILABLE');
  assert.equal(s.security.find((c) => c.id === 'ddos')!.result, 'UNAVAILABLE');
  assert.ok(!('score' in (s as unknown as Record<string, unknown>)));
});

test('demo replay is labelled SIMULATED end to end', () => {
  const s = mapView(healthy, { ...opts(healthy), state: 'SIMULATED', source: 'demo-replay' });
  assert.equal(s.overview.provenance.state, 'SIMULATED');
  assert.equal(s.overview.metrics.nodesTotal.state, 'SIMULATED');
  const caps = deriveCapabilities({ mode: 'demo', now: 0, snapshot: s, reachable: true, detail: 'demo', hasRegionLocations: false, copilotModel: false });
  assert.equal(caps.items.nodes.state, 'SIMULATED');
});

test('capabilities lose confidence when the backend is unreachable', () => {
  const caps = deriveCapabilities({ mode: 'controlplane', now: 0, snapshot: null, reachable: false, detail: 'connect ECONNREFUSED', hasRegionLocations: false, copilotModel: false });
  assert.equal(caps.backend.state, 'UNAVAILABLE');
  for (const k of ['nodes', 'applications', 'deployments', 'evidence', 'storage'] as const) assert.equal(caps.items[k].state, 'UNKNOWN');
  assert.equal(caps.items.billing.state, 'UNAVAILABLE');
  assert.equal(caps.items.geolocation.state, 'UNAVAILABLE');
});

test('region positions are CONFIGURED, never inferred', () => {
  const s = mapView(healthy, { ...opts(healthy), regionLocations: { 'cell-a': [10, 20] } });
  const a = s.nodes.find((n) => n.region === 'cell-a');
  assert.deepEqual(a?.location, { lat: 10, lng: 20, state: 'CONFIGURED' });
  assert.equal(s.nodes.find((n) => n.region !== 'cell-a')!.location, null);
});

test('hardware: an agent without the unknown list is not trusted for measured memory', () => {
  const old = measuredHardware({ cpus: 4, memBytes: 2 << 30 });
  assert.equal(old.memBytes, null);
  assert.equal(old.unknown, null);
  assert.equal(old.disks, null);
  const now = measuredHardware({ cpus: 4, memBytes: 16 * 2 ** 30, cpuModel: 'X', physicalCores: 4, swapBytes: 0, disks: [], gpus: [], unknown: ['gpus', 'natType'] });
  assert.equal(now.memBytes, 16 * 2 ** 30);
  assert.equal(now.swapBytes, 0);
  assert.deepEqual(now.disks, []);
  assert.equal(now.gpus, null, 'gpus listed as unknown must not render as "none"');
  const unmeasured = measuredHardware({ cpus: 4, memBytes: 0, unknown: ['memBytes'] });
  assert.equal(unmeasured.memBytes, null);
});

test('app resources are the manifest requests (cpuMilli, memBytes), not strings', () => {
  const s = mapView(healthy, opts(healthy));
  const web = s.apps.find((a) => a.name === 'web')!;
  const raw = (healthy.apps ?? []).find((a) => a.name === 'web')!.manifest!.spec!.resources!;
  assert.equal(web.resources.cpuMilli, raw.cpuMilli);
  assert.equal(web.resources.memBytes, raw.memBytes);
  assert.ok(web.resources.cpuMilli! > 0);
});
