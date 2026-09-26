#!/usr/bin/env node
/**
 * Backend-disconnect qualification (real processes, real browser).
 *
 * Freezes every control-plane member of a `dh dev up` cluster with SIGSTOP
 * while a signed-in browser shows the console, checks that the console loses
 * confidence instead of inventing continuity, then resumes the members and
 * checks recovery.
 *
 *   CC_URL=http://127.0.0.1:3100 DH_TOKEN_ADMIN=... DEV_CLUSTER_DIR=./devcluster \
 *   EDGE_URL=http://127.0.0.1:18103 EDGE_HOST=web.dev.test OUT_DIR=./out node tests/e2e/disconnect.mjs
 *
 * Needs Playwright (set PLAYWRIGHT_MODULE to its path if it is not resolvable).
 * Exit code 0 only if every check passes.
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import http from 'node:http';

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const CC = process.env.CC_URL || 'http://127.0.0.1:3100';
const TOK = process.env.DH_TOKEN_ADMIN;
const DIR = process.env.DEV_CLUSTER_DIR;
const EDGE = process.env.EDGE_URL || 'http://127.0.0.1:18103';
const EDGE_HOST = process.env.EDGE_HOST || 'web.dev.test';
const OUT = process.env.OUT_DIR || '.';
if (!TOK || !DIR) {
  console.error('DH_TOKEN_ADMIN and DEV_CLUSTER_DIR are required');
  process.exit(2);
}
mkdirSync(OUT, { recursive: true });
const pids = JSON.parse(readFileSync(`${DIR}/cluster.json`, 'utf8')).procs.filter((p) => p.kind === 'control').map((p) => String(p.pid));

const checks = [];
const check = (name, ok, detail = '') => {
  checks.push({ name, ok: !!ok, detail });
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail ? `  (${detail})` : ''}`);
};
const api = async (method, path, body) => {
  try {
    const r = await fetch(`${CC}/api/v1${path}`, {
      method,
      headers: { Authorization: `Bearer ${TOK}`, 'Content-Type': 'application/json', 'X-DH-Console': '1' },
      body: body ? JSON.stringify(body) : undefined,
      signal: AbortSignal.timeout(15000)
    });
    return { status: r.status, json: await r.json().catch(() => null) };
  } catch (e) {
    return { status: 0, json: null, err: e.message };
  }
};
// fetch() drops a custom Host header, so the edge probe uses node:http, which sends it.
const edge = () =>
  new Promise((resolve) => {
    const u = new URL(EDGE);
    const req = http.request({ host: u.hostname, port: u.port, path: '/', headers: { Host: EDGE_HOST, 'X-DH-No-Redirect': '1' }, timeout: 5000 }, (res) => {
      res.resume();
      resolve(res.statusCode);
    });
    req.on('timeout', () => req.destroy());
    req.on('error', () => resolve(0));
    req.end();
  });
const signal = (sig) => execFileSync('kill', [`-${sig}`, ...pids]);

const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1536, height: 1024 } })).newPage();
let stopped = false;
try {
  await page.goto(`${CC}/#token=${TOK}`, { waitUntil: 'networkidle' });
  await page.goto(`${CC}/nodes`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: `${OUT}/disconnect-1-before.png` });
  check('before: nodes API answers', (await api('GET', '/nodes')).status === 200);
  check('before: edge serves the workload', (await edge()) === 200);

  signal('STOP');
  stopped = true;
  console.log(`SIGSTOP control-plane members ${pids.join(',')}`);
  await page.waitForTimeout(12000);

  check('during: existing workload still served by the edge', (await edge()) === 200);
  const n = await api('GET', '/nodes');
  check('during: nodes API returns an error, not data', n.status >= 500 && !n.json?.data, `${n.status} ${n.json?.error?.code ?? ''}`);
  const c = await api('POST', '/copilot/query', { query: 'how many nodes are online?' });
  check('during: copilot refuses to claim current state', c.json?.data?.state === 'UNKNOWN' && /cannot verify/.test(c.json?.data?.content ?? ''), (c.json?.data?.content ?? '').split('\n')[0]);
  const d = await api('POST', '/deployments', { manifest: 'apiVersion: dh/v1\nkind: Application\nmetadata:\n  name: dc-test\n' });
  check('during: deployment cannot complete', d.status >= 500, `${d.status} ${d.json?.error?.code ?? ''}`);
  const caps = await api('GET', '/capabilities');
  check('during: capabilities lose confidence', caps.json?.data?.backend?.reachable === false && caps.json?.data?.items?.nodes?.state === 'UNKNOWN');
  check('during: UI shows CONTROL PLANE UNREACHABLE', (await page.locator('text=CONTROL PLANE UNREACHABLE').count()) > 0);
  check('during: UI desaturates last observations', (await page.locator('.grayscale').count()) > 0);
  await page.screenshot({ path: `${OUT}/disconnect-2-during.png` });

  signal('CONT');
  stopped = false;
  console.log('SIGCONT control-plane members');
  let recovered = false;
  for (let i = 0; i < 30 && !recovered; i++) {
    await page.waitForTimeout(2000);
    const r = await api('GET', '/nodes');
    recovered = r.status === 200 && r.json.data.nodes.every((x) => x.health === 'HEALTHY');
  }
  check('after: all hosts healthy again', recovered);
  await page.waitForTimeout(6000);
  check('after: UI banner cleared', (await page.locator('text=CONTROL PLANE UNREACHABLE').count()) === 0);
  check('after: nothing left desaturated', (await page.locator('.grayscale').count()) === 0);
  await page.screenshot({ path: `${OUT}/disconnect-3-after.png` });
} finally {
  if (stopped) signal('CONT');
  await browser.close();
}
writeFileSync(`${OUT}/disconnect-checks.json`, JSON.stringify(checks, null, 2) + '\n');
const failed = checks.filter((c) => !c.ok).length;
console.log(`${checks.length - failed}/${checks.length} checks passed`);
process.exit(failed ? 1 : 0);
