/**
 * Failure injection that needs no cluster: the backend is unreachable,
 * hangs, or answers garbage. The BFF must lose confidence, never invent data.
 */
import { test, after } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import net from 'node:net';
import type { AddressInfo } from 'node:net';
import { ControlPlaneAdapter } from '../../src/server/adapters/controlPlane';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';
import { loadConfig, ConfigError } from '../../src/server/config';
import { startBff } from '../helpers';

// A syntactically valid (unsigned) capability: the BFF forwards it; only the backend can judge it.
const TOKEN = 'dhcap1.' + Buffer.from(JSON.stringify([{ payload: { caveats: { actions: ['api.read', 'api.write'], expires: Date.now() + 3600_000, resources: ['cluster/dev'] }, note: 'test' } }])).toString('base64url');
const servers: { close: () => unknown }[] = [];
after(() => servers.forEach((s) => s.close()));

async function listen(handler: http.RequestListener) {
  const s = http.createServer(handler);
  await new Promise<void>((r) => s.listen(0, '127.0.0.1', () => r()));
  servers.push(s);
  return `http://127.0.0.1:${(s.address() as AddressInfo).port}`;
}

test('backend offline: no data, no demo fallback, capabilities lose confidence, copilot refuses', async () => {
  const bff = await startBff(new ControlPlaneAdapter({ endpoints: ['http://127.0.0.1:1'], timeoutMs: 1000 }));
  try {
    const h = await bff.call('GET', '/health');
    assert.equal(h.json.backend.reachable, false);
    for (const p of ['/nodes', '/overview', '/apps', '/storage', '/evidence', '/analytics', '/security']) {
      const r = await bff.call('GET', p, { token: TOKEN });
      assert.equal(r.status, 503, p);
      assert.equal(r.json.error.code, 'BACKEND_UNAVAILABLE', p);
      assert.equal(r.json.data, undefined, `${p} must not return data`);
    }
    const caps = await bff.call('GET', '/capabilities', { token: TOKEN });
    assert.equal(caps.json.data.backend.reachable, false);
    assert.equal(caps.json.data.items.nodes.state, 'UNKNOWN');
    assert.equal(caps.json.data.items.deployments.state, 'UNKNOWN');
    const c = await bff.call('POST', '/copilot/query', { token: TOKEN, body: { query: 'how many nodes are online?' } });
    assert.equal(c.status, 200);
    assert.equal(c.json.data.state, 'UNKNOWN');
    assert.match(c.json.data.content, /cannot verify current platform state/);
    assert.equal(/\d+ of \d+ host/.test(c.json.data.content), false);
    const d = await bff.call('POST', '/deployments', { token: TOKEN, body: { manifest: 'apiVersion: dh/v1\nkind: Application\nmetadata:\n  name: x\n' } });
    assert.equal(d.status, 503);
    const n = await bff.call('POST', '/nodes/dh1abcdefghijk/operations', { token: TOKEN, body: { type: 'DRAIN' } });
    assert.equal(n.status, 503);
  } finally {
    await bff.close();
  }
});

test('backend timeout is reported as BACKEND_TIMEOUT', async () => {
  const hang = net.createServer(() => {
    /* accept and never answer */
  });
  await new Promise<void>((r) => hang.listen(0, '127.0.0.1', () => r()));
  servers.push(hang);
  const url = `http://127.0.0.1:${(hang.address() as AddressInfo).port}`;
  const bff = await startBff(new ControlPlaneAdapter({ endpoints: [url], timeoutMs: 400 }));
  try {
    const r = await bff.call('GET', '/nodes', { token: TOKEN });
    assert.equal(r.status, 504);
    assert.equal(r.json.error.code, 'BACKEND_TIMEOUT');
  } finally {
    await bff.close();
  }
});

test('malformed backend response is refused, not rendered', async () => {
  const url = await listen((req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.end(req.url?.startsWith('/api/v1/view') ? JSON.stringify({ hello: 'world' }) : '{}');
  });
  const bff = await startBff(new ControlPlaneAdapter({ endpoints: [url], timeoutMs: 1000 }));
  try {
    const r = await bff.call('GET', '/nodes', { token: TOKEN });
    assert.equal(r.status, 502);
    assert.equal(r.json.error.code, 'BACKEND_MALFORMED');
  } finally {
    await bff.close();
  }
});

test('backend 5xx never leaks internals', async () => {
  const url = await listen((_req, res) => {
    res.statusCode = 503;
    res.end(JSON.stringify({ ok: false, message: 'open /var/lib/dh/secret.key: permission denied' }));
  });
  const bff = await startBff(new ControlPlaneAdapter({ endpoints: [url], timeoutMs: 1000 }));
  try {
    const r = await bff.call('GET', '/nodes', { token: TOKEN });
    assert.equal(r.status, 502);
    assert.equal(r.text.includes('/var/lib'), false);
    assert.equal(r.text.includes('secret.key'), false);
  } finally {
    await bff.close();
  }
});

test('failover: a dead first member does not blind the console', async () => {
  const url = await listen((_req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({ member: 'm2', state: 'follower', cluster: 'dev' }));
  });
  const bff = await startBff(new ControlPlaneAdapter({ endpoints: ['http://127.0.0.1:1', url], timeoutMs: 800 }));
  try {
    const h = await bff.call('GET', '/health');
    assert.equal(h.json.backend.reachable, true);
    assert.equal(h.json.backend.state, 'follower');
  } finally {
    await bff.close();
  }
});

test('demo adapter is labelled SIMULATED and refuses every mutation', async () => {
  const bff = await startBff(new DemoPlatformAdapter());
  try {
    const r = await bff.call('GET', '/nodes');
    assert.equal(r.status, 200);
    assert.equal(r.json.provenance.state, 'SIMULATED');
    const m = await bff.call('POST', '/nodes/dh1abcdefghijk/operations', { body: { type: 'DRAIN' } });
    assert.equal(m.status, 409);
    assert.equal(m.json.error.code, 'NOT_SUPPORTED');
    const v = await bff.call('POST', '/evidence/verify', { body: {} });
    assert.equal(v.json.data.state, 'UNKNOWN', 'a replay cannot claim fresh verification');
  } finally {
    await bff.close();
  }
});

test('configuration: no silent adapter choice; demo refused in production', () => {
  assert.throws(() => loadConfig({} as NodeJS.ProcessEnv), ConfigError);
  assert.throws(() => loadConfig({ PLATFORM_ADAPTER: 'python' } as NodeJS.ProcessEnv), ConfigError);
  assert.throws(() => loadConfig({ PLATFORM_ADAPTER: 'controlplane' } as NodeJS.ProcessEnv), /DH_CONTROL_URL/);
  assert.throws(() => loadConfig({ PLATFORM_ADAPTER: 'demo', NODE_ENV: 'production' } as NodeJS.ProcessEnv), /Refusing/);
  assert.equal(loadConfig({ PLATFORM_ADAPTER: 'demo', NODE_ENV: 'production', ALLOW_DEMO_IN_PRODUCTION: '1' } as NodeJS.ProcessEnv).adapter, 'demo');
  assert.throws(() => loadConfig({ PLATFORM_ADAPTER: 'controlplane', DH_CONTROL_URL: 'http://x', DH_REGION_LOCATIONS: '{"a":[1]}' } as NodeJS.ProcessEnv), /lat, lng/);
  const c = loadConfig({ PLATFORM_ADAPTER: 'controlplane', DH_CONTROL_URL: 'http://a:1/, https://b:2', NODE_ENV: 'production' } as NodeJS.ProcessEnv);
  assert.deepEqual(c.controlEndpoints, ['http://a:1', 'https://b:2']);
  assert.equal(c.secureCookies, true);
  assert.equal(c.copilotModel, null, 'no model (no phone-home) unless explicitly enabled');
});

test('request ids are UUIDs; untrusted incoming ids are replaced', async () => {
  const bff = await startBff(new DemoPlatformAdapter());
  try {
    const r = await fetch(`${bff.base}/health`, { headers: { 'X-Request-Id': '<script>' } });
    assert.match(r.headers.get('x-request-id') ?? '', /^[0-9a-f]{8}-[0-9a-f]{4}-/);
    const good = '11111111-2222-4333-8444-555555555555';
    const r2 = await fetch(`${bff.base}/health`, { headers: { 'X-Request-Id': good } });
    assert.equal(r2.headers.get('x-request-id'), good);
  } finally {
    await bff.close();
  }
});
