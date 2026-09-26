/**
 * Integration tests against a REAL control plane (`dh dev up`).
 *
 *   DH_CONTROL_URL=http://127.0.0.1:17701,http://127.0.0.1:17702 \
 *   DH_TOKEN_ADMIN=$(dh token --ttl 2h) DH_TOKEN_READ=$(dh token --read-only --ttl 2h) \
 *   npm run test:integration
 *
 * Skipped (not passed) when those variables are absent.
 */
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { ControlPlaneAdapter } from '../../src/server/adapters/controlPlane';
import { startBff, waitFor } from '../helpers';

const URLS = (process.env.DH_CONTROL_URL ?? '').split(',').filter(Boolean);
const ADMIN = process.env.DH_TOKEN_ADMIN ?? '';
const READ = process.env.DH_TOKEN_READ ?? '';
const skip = !URLS.length || !ADMIN || !READ ? 'DH_CONTROL_URL / DH_TOKEN_ADMIN / DH_TOKEN_READ not set' : false;
const APP = `ccq${Date.now().toString(36).slice(-5)}`;

let bff: Awaited<ReturnType<typeof startBff>>;
let hostB = '';
let digestImage = '';

before(async () => {
  if (skip) return;
  bff = await startBff(new ControlPlaneAdapter({ endpoints: URLS, timeoutMs: 5000 }));
});
after(async () => {
  if (skip) return;
  // Leave the cluster as we found it.
  await bff.call('POST', `/apps/${APP}/delete`, { token: ADMIN, body: { confirm: APP } }).catch(() => {});
  if (hostB) await bff.call('POST', `/nodes/${hostB}/operations`, { token: ADMIN, body: { type: 'UNDRAIN' } }).catch(() => {});
  await bff.close();
});

async function viewDirect(token: string) {
  const r = await fetch(`${URLS[0]}/api/v1/view`, { headers: { Authorization: `Bearer ${token}` } });
  return r.json() as Promise<any>;
}

test('health reports the real control plane', { skip }, async () => {
  const r = await bff.call('GET', '/health');
  assert.equal(r.status, 200);
  assert.equal(r.json.backend.reachable, true);
  assert.equal(r.json.mode, 'controlplane');
  assert.match(r.headers.get('x-request-id') ?? '', /^[0-9a-f-]{36}$/);
});

test('unauthenticated and malformed credentials are rejected', { skip }, async () => {
  const a = await bff.call('GET', '/nodes');
  assert.equal(a.status, 401);
  assert.equal(a.json.error.code, 'UNAUTHENTICATED');
  assert.ok(a.json.error.request_id);
  const b = await bff.call('GET', '/nodes', { token: 'dhcap1.garbage' });
  assert.equal(b.status, 401);
  const c = await bff.call('POST', '/session', { body: { token: 'not-a-cap' } });
  assert.equal(c.status, 422);
});

test('node count and health match the control plane exactly', { skip }, async () => {
  const [r, v] = await Promise.all([bff.call('GET', '/nodes', { token: READ }), viewDirect(READ)]);
  assert.equal(r.status, 200);
  assert.equal(r.json.data.nodes.length, v.nodes.length);
  for (const n of v.nodes) {
    const m = r.json.data.nodes.find((x: any) => x.id === n.id);
    assert.ok(m, `node ${n.name} missing`);
    const expected = n.health === 'lost' ? 'OFFLINE' : n.lastObs.freshness === 'FRESH' ? 'HEALTHY' : n.lastObs.freshness === 'STALE' ? 'DEGRADED' : 'UNKNOWN';
    assert.equal(m.health, expected, `${n.name}`);
  }
  assert.equal(r.json.provenance.state, 'LIVE');
  hostB = r.json.data.nodes.find((n: any) => n.name === 'host-b')?.id ?? '';
});

test('node detail is deep-linkable by id and by name', { skip }, async () => {
  const byId = await bff.call('GET', `/nodes/${hostB}`, { token: READ });
  const byName = await bff.call('GET', '/nodes/host-b', { token: READ });
  assert.equal(byId.status, 200);
  assert.equal(byName.json.data.node.id, hostB);
  const missing = await bff.call('GET', '/nodes/nope', { token: READ });
  assert.equal(missing.status, 404);
});

test('read-only capability cannot mutate (enforced by the control plane)', { skip }, async () => {
  const r = await bff.call('POST', `/nodes/${hostB}/operations`, { token: READ, body: { type: 'DRAIN' } });
  assert.equal(r.status, 403);
  assert.equal(r.json.error.code, 'PERMISSION_DENIED');
  assert.match(r.json.error.message, /not granted/);
});

test('mutations without the console header are rejected (CSRF)', { skip }, async () => {
  const r = await bff.call('POST', `/nodes/${hostB}/operations`, { token: ADMIN, body: { type: 'DRAIN' }, csrf: false });
  assert.equal(r.status, 403);
  assert.equal(r.json.error.code, 'CSRF_REJECTED');
});

test('destructive operations require confirmation', { skip }, async () => {
  const r = await bff.call('POST', `/nodes/${hostB}/operations`, { token: ADMIN, body: { type: 'REVOKE' } });
  assert.equal(r.status, 428);
  assert.equal(r.json.error.code, 'CONFIRMATION_REQUIRED');
  const d = await bff.call('POST', '/apps/web/delete', { token: ADMIN, body: {} });
  assert.equal(d.status, 428);
});

test('invalid manifests are refused by the control plane, not accepted', { skip }, async () => {
  const r = await bff.call('POST', '/deployments', { token: ADMIN, body: { manifest: `apiVersion: dh/v1\nkind: Application\nmetadata:\n  name: ${APP}\nspec:\n  replicas: 1\n  image: nginx:latest\n` } });
  assert.equal(r.status, 422);
  assert.equal(r.json.error.code, 'VALIDATION_FAILED');
});

test('deployment lifecycle: accepted, progresses from real observations, reaches READY', { skip, timeout: 120_000 }, async () => {
  const web = await bff.call('GET', '/apps/web', { token: READ });
  digestImage = web.json.data.app.image;
  assert.match(digestImage, /@b3:[0-9a-f]{64}$/);
  const manifest = [
    'apiVersion: dh/v1',
    'kind: Application',
    'metadata:',
    `  name: ${APP}`,
    'spec:',
    '  replicas: 2',
    `  image: ${digestImage}`,
    '  resources:',
    '    cpu: 50m',
    '    mem: 32Mi',
    '  placement:',
    '    tiers: [trusted]',
    '    spread: failure-domain',
    '    antiAffinity: hard',
    '  ports:',
    '    - name: http',
    '  health:',
    '    http: /healthz',
    '    interval: 2s',
    ''
  ].join('\n');
  const r = await bff.call('POST', '/deployments', { token: ADMIN, body: { manifest } });
  assert.equal(r.status, 202, JSON.stringify(r.json));
  assert.equal(r.json.data.ok, true);
  const gen = r.json.data.deployment.generation;
  assert.ok(['QUEUED', 'DEPLOYING', 'VERIFYING'].includes(r.json.data.deployment.state), `initial state ${r.json.data.deployment.state} must not be READY/VERIFIED`);
  const ready = await waitFor(async () => {
    const p = await bff.call('GET', `/deployments/${APP}/${gen}`, { token: READ });
    return p.json?.data?.deployment?.state === 'READY' ? p.json.data : null;
  }, 90_000, 1500);
  assert.ok(ready.deployment.stages.every((s: any) => s.state === 'PASSED'));
  assert.equal(ready.replicas.length, 2);
  // Evidence is separate from deployment success: the deployment carries no "VERIFIED" evidence claim.
  assert.equal(JSON.stringify(ready.deployment).includes('VERIFIED'), false);
});

test('the actor recorded in the audit ledger comes from the capability, not the UI', { skip }, async () => {
  const r = await bff.call('GET', '/audit?limit=100', { token: READ });
  const e = r.json.data.entries.find((x: any) => x.action === 'manifest-apply' && x.resource.includes(APP));
  assert.ok(e, 'manifest-apply entry for the test app');
  assert.match(e.actor, /^operator/);
  assert.equal(/Febin/i.test(e.actor), false);
});

test('scaling to zero needs confirmation; scaling up goes through the control plane', { skip, timeout: 60_000 }, async () => {
  const z = await bff.call('POST', `/apps/${APP}/scale`, { token: ADMIN, body: { replicas: 0 } });
  assert.equal(z.status, 428);
  const s = await bff.call('POST', `/apps/${APP}/scale`, { token: ADMIN, body: { replicas: 3 } });
  assert.equal(s.status, 200);
  const app = await waitFor(async () => {
    const a = await bff.call('GET', `/apps/${APP}`, { token: READ });
    return a.json.data.app.desiredReplicas === 3 ? a.json.data : null;
  }, 20_000);
  // Control-plane semantics: replica changes keep the generation and append a revision.
  assert.equal(app.app.generation, 1);
  assert.equal(app.deployments.length, 1, 'one deployment per generation, not per history entry');
  assert.ok(app.deployments[0].revisions.some((r: any) => /replicas 2 → 3/.test(r.change)));
});

test('drain and undrain round-trip through the control plane', { skip, timeout: 60_000 }, async () => {
  const d = await bff.call('POST', `/nodes/${hostB}/operations`, { token: ADMIN, body: { type: 'DRAIN' } });
  assert.equal(d.status, 200, JSON.stringify(d.json));
  assert.equal(d.json.data.ok, true);
  await waitFor(async () => (await bff.call('GET', `/nodes/${hostB}`, { token: READ })).json.data.node.lifecycle === 'DRAINING', 15_000);
  const u = await bff.call('POST', `/nodes/${hostB}/operations`, { token: ADMIN, body: { type: 'UNDRAIN' } });
  assert.equal(u.json.data.ok, true);
  await waitFor(async () => (await bff.call('GET', `/nodes/${hostB}`, { token: READ })).json.data.node.lifecycle === 'ACTIVE', 15_000);
});

test('operations on unknown nodes are reported as NOT_FOUND', { skip }, async () => {
  const r = await bff.call('POST', '/nodes/dh1doesnotexistxxxxxxxxxxxxx/operations', { token: ADMIN, body: { type: 'DRAIN' } });
  assert.equal(r.status, 404);
});

test('Copilot answers are grounded and cited; read-only principals get no action proposals', { skip }, async () => {
  const a = await bff.call('POST', '/copilot/query', { token: READ, body: { query: 'How are my hosts doing?' } });
  assert.equal(a.status, 200);
  assert.equal(a.json.data.grounded, true);
  assert.ok(a.json.data.citations.some((c: any) => c.type === 'NODE_OBSERVATION'));
  const v = await viewDirect(READ);
  assert.ok(a.json.data.content.includes(`of ${v.nodes.length} host(s)`), a.json.data.content);
  const b = await bff.call('POST', '/copilot/query', { token: READ, body: { query: 'drain host-b' } });
  assert.equal(b.json.data.proposedAction, null);
  assert.match(b.json.data.content, /read-only/);
});

test('Copilot actions require approval by the same principal and execute via the control plane', { skip, timeout: 60_000 }, async () => {
  const p = await bff.call('POST', '/copilot/query', { token: ADMIN, body: { query: 'drain host-b' } });
  const act = p.json.data.proposedAction;
  assert.ok(act);
  assert.equal(act.status, 'PENDING_APPROVAL');
  // A different principal cannot approve it.
  const other = await bff.call('POST', `/copilot/actions/${act.id}/approve`, { token: READ, body: {} });
  assert.equal(other.status, 403);
  const ok = await bff.call('POST', `/copilot/actions/${act.id}/approve`, { token: ADMIN, body: {} });
  assert.equal(ok.status, 200, JSON.stringify(ok.json));
  assert.equal(ok.json.data.result.ok, true);
  const again = await bff.call('POST', `/copilot/actions/${act.id}/approve`, { token: ADMIN, body: {} });
  assert.equal(again.status, 404, 'an approved proposal cannot be replayed');
  await bff.call('POST', `/nodes/${hostB}/operations`, { token: ADMIN, body: { type: 'UNDRAIN' } });
});

test('Copilot CRITICAL actions need typed confirmation', { skip }, async () => {
  const p = await bff.call('POST', '/copilot/query', { token: ADMIN, body: { query: 'revoke host-b' } });
  const act = p.json.data.proposedAction;
  assert.equal(act.risk, 'CRITICAL');
  const r = await bff.call('POST', `/copilot/actions/${act.id}/approve`, { token: ADMIN, body: { confirm: 'wrong' } });
  assert.equal(r.status, 428);
  // Consumed by the failed attempt: dismissing now reports it gone, nothing was revoked.
  const n = await bff.call('GET', `/nodes/${hostB}`, { token: READ });
  assert.notEqual(n.json.data.node.lifecycle, 'REVOKED');
});

test('ledger re-verification is performed by the control plane', { skip }, async () => {
  const r = await bff.call('POST', '/evidence/verify', { token: READ, body: {} });
  assert.equal(r.status, 200);
  assert.equal(r.json.data.state, 'VERIFIED');
  assert.ok(r.json.data.entries > 0);
});

test('backend error text is sanitized before it reaches the browser', { skip }, async () => {
  const r = await bff.call('GET', `/nodes/${hostB}/logs?assignment=nope/r0`, { token: READ });
  const body = JSON.stringify(r.json ?? r.text);
  assert.equal(/\/tmp\/|\/home\/|\/var\//.test(body), false, body);
});

test('secrets (env values) never leave the BFF', { skip }, async () => {
  const r = await bff.call('GET', '/apps/web', { token: READ });
  assert.ok(r.json.data.app.envNames.includes('MESSAGE'));
  assert.equal(JSON.stringify(r.json).includes('hello from a sovereign host'), false);
});
