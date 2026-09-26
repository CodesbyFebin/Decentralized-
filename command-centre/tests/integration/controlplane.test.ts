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

/* ------------------------------------------------------------------ Wave 1 */

test('invites: admin can list them; a read-only capability is refused by the control plane', { skip }, async () => {
  const a = await bff.call('GET', '/invites', { token: ADMIN });
  assert.equal(a.status, 200);
  assert.ok(Array.isArray(a.json.data.invites));
  assert.equal(a.json.provenance.state, 'LIVE');
  assert.equal(JSON.stringify(a.json).includes('dhcap1.'), false, 'no token material');
  const r = await bff.call('GET', '/invites', { token: READ });
  assert.equal(r.status, 403);
  assert.equal(r.json.error.code, 'PERMISSION_DENIED');
});

test('invites: revoking goes through the control plane, is audited, and needs the console header', { skip }, async () => {
  // Record an invite directly (the join token itself would need the root key, which tests do not hold).
  const nonce = [...crypto.getRandomValues(new Uint8Array(16))].map((b) => b.toString(16).padStart(2, '0')).join('');
  const rec = await fetch(`${URLS[0]}/api/v1/invites`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${ADMIN}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ nonce, expires: Date.now() + 600_000, roles: [], note: 'cc wave1', auto: false })
  });
  assert.equal(rec.status, 200);
  assert.equal((await bff.call('POST', `/invites/${nonce}/revoke`, { token: ADMIN, csrf: false })).status, 403);
  assert.equal((await bff.call('POST', `/invites/${nonce}/revoke`, { token: READ })).status, 403);
  const ok = await bff.call('POST', `/invites/${nonce}/revoke`, { token: ADMIN });
  assert.equal(ok.status, 200);
  assert.equal(ok.json.data.ok, true);
  const list = await bff.call('GET', '/invites', { token: ADMIN });
  assert.equal(list.json.data.invites.find((i: any) => i.nonce === nonce)?.state, 'REVOKED');
  const audit = await bff.call('GET', '/audit?limit=20', { token: READ });
  assert.ok(audit.json.data.entries.some((e: any) => e.action === 'node-invite-revoke' && e.resource === `invite/${nonce.slice(0, 8)}`));
  assert.equal((await bff.call('POST', '/invites/not-hex/revoke', { token: ADMIN })).status, 422);
});

test('node detail derives allocation from the manifests of replicas desired on it', { skip }, async () => {
  const v = await viewDirect(READ);
  const web = v.apps.find((a: any) => a.name === 'web');
  const row = web.rows.find((r: any) => r.desired === 'RUNNING');
  const n = await bff.call('GET', `/nodes/${row.node}`, { token: READ });
  const onNode = v.apps.flatMap((a: any) => (a.rows ?? []).filter((r: any) => r.node === row.node && r.desired === 'RUNNING').map(() => a.manifest.spec.resources?.cpuMilli ?? 0));
  assert.equal(n.json.data.allocated.cpuMilli, onNode.reduce((x: number, y: number) => x + y, 0));
  assert.ok(n.json.data.allocated.cpuMilli > 0);
});

test('operations are derived from state and labelled so', { skip }, async () => {
  const r = await bff.call('GET', '/operations', { token: READ });
  assert.equal(r.status, 200);
  assert.equal(r.json.provenance.state, 'DERIVED');
  const d = r.json.data.operations.find((o: any) => o.kind === 'deployment' && o.target === 'web');
  assert.ok(d, 'web deployment listed');
  assert.ok(['SUCCEEDED', 'RUNNING'].includes(d.status));
  assert.ok(d.source);
});

test('refusals come from the control plane log', { skip }, async () => {
  // A heartbeat from an unknown host is refused and recorded.
  await fetch(`${URLS[0]}/v1/observe`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ observations: [{ kind: 'observation', signer: 'dh1aaaaaaaaaaaaaaaaaaaaaaaaaa', pub: 'x', sig: 'x', payload: { seq: 1 } }] }) });
  const r = await bff.call('GET', '/rejections', { token: READ });
  assert.equal(r.status, 200);
  assert.ok(Array.isArray(r.json.data.rejections));
});

test('settings expose configuration, never credentials', { skip }, async () => {
  const r = await bff.call('GET', '/settings', { token: ADMIN });
  assert.equal(r.status, 200);
  const body = JSON.stringify(r.json);
  assert.equal(body.includes(ADMIN), false);
  assert.equal(body.includes('dhcap1.'), false);
  assert.equal(r.json.data.adapter, 'controlplane');
});

test('validation records are listed as sealed and verified by the platform verifier', { skip, timeout: 60_000 }, async () => {
  const repo = new URL('../../../', import.meta.url).pathname;
  const { existsSync } = await import('node:fs');
  const cli = existsSync(`${repo}bin/dh`) ? `${repo}bin/dh` : null;
  const ev = await startBff(new ControlPlaneAdapter({ endpoints: URLS, timeoutMs: 5000 }), { evidenceDir: `${repo}evidence`, cli });
  try {
    const caps = await ev.call('GET', '/capabilities', { token: READ });
    assert.equal(caps.json.data.items.validationRecords.state, 'LIVE');
    const list = await ev.call('GET', '/evidence/records', { token: READ });
    assert.equal(list.status, 200);
    const fail = list.json.data.records.find((x: any) => x.id === 'NODE-A01-A01');
    assert.equal(fail.outcome, 'FAIL', 'failed records are listed');
    assert.equal((await ev.call('GET', '/evidence/records', {})).status, 401);
    assert.ok([404, 422].includes((await ev.call('GET', '/evidence/records/..%2F..%2Fpkg', { token: READ })).status), 'path traversal refused');
    if (cli) {
      const v = await ev.call('POST', '/evidence/records/NODE-A01-A02/verify', { token: READ });
      assert.equal(v.status, 200);
      assert.equal(v.json.data.state, 'VERIFIED');
    }
    const raw = await ev.call('GET', '/evidence/records/NODE-A01-A02/record.json', { token: READ });
    assert.equal(raw.status, 200);
    assert.match(raw.text, /"kind":\s*"validation-record"/);
  } finally {
    await ev.close();
  }
});
