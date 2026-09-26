#!/usr/bin/env node
/**
 * Wave 1 per-page disconnect gate (real processes, real browser).
 *
 * Every Wave 1 route is loaded in its own tab while the cluster is healthy.
 * Then every control-plane member is frozen with SIGSTOP. Each page must:
 * show CONTROL PLANE UNREACHABLE, keep its last observation visible but
 * desaturated (or show an error), and show no FRESH badge. After SIGCONT each
 * page must recover on its own.
 *
 *   CC_URL=http://127.0.0.1:3100 DH_TOKEN_ADMIN=... DEV_CLUSTER_DIR=./devcluster OUT_DIR=./out \
 *     node tests/e2e/wave1-disconnect.mjs
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { execFileSync } from 'node:child_process';

const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || 'playwright');
const CC = process.env.CC_URL || 'http://127.0.0.1:3100';
const TOK = process.env.DH_TOKEN_ADMIN;
const DIR = process.env.DEV_CLUSTER_DIR;
const OUT = process.env.OUT_DIR || '.';
if (!TOK || !DIR) {
  console.error('DH_TOKEN_ADMIN and DEV_CLUSTER_DIR are required');
  process.exit(2);
}
mkdirSync(OUT, { recursive: true });
const pids = JSON.parse(readFileSync(`${DIR}/cluster.json`, 'utf8')).procs.filter((p) => p.kind === 'control').map((p) => String(p.pid));
const signal = (sig) => execFileSync('kill', [`-${sig}`, ...pids]);

const checks = [];
const check = (name, ok, detail = '') => {
  checks.push({ name, ok: !!ok, detail });
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail ? `  (${detail})` : ''}`);
};

const nodes = await (await fetch(`${CC}/api/v1/nodes`, { headers: { Authorization: `Bearer ${TOK}` } })).json();
const nodeId = nodes.data.nodes.find((n) => n.lifecycle === 'ACTIVE').id;
const app = (await (await fetch(`${CC}/api/v1/apps`, { headers: { Authorization: `Bearer ${TOK}` } })).json()).data.apps[0]?.name;

// [route, depends on the control plane]
const ROUTES = [
  ['/', true],
  ['/apps', true],
  [`/apps/${encodeURIComponent(app)}`, true],
  ['/nodes', true],
  [`/nodes/${nodeId}`, true],
  ['/nodes/add', true],
  ['/evidence', true],
  ['/activity?tab=refused', true],
  ['/operations', true],
  ['/settings', false]
];

/** Visible elements whose own text is exactly FRESH (the freshness badge label). */
const freshBadges = (page) =>
  page.evaluate(() => {
    let n = 0;
    for (const el of document.querySelectorAll('span')) {
      const own = [...el.childNodes].filter((c) => c.nodeType === 3).map((c) => c.textContent.trim()).join('');
      if (own === 'FRESH' && el.offsetParent !== null) n++;
    }
    return n;
  });

const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } });
const first = await ctx.newPage();
await first.goto(`${CC}/#token=${TOK}`, { waitUntil: 'networkidle' });
const pages = [];
for (const [route, cp] of ROUTES) {
  const p = route === '/' ? first : await ctx.newPage();
  await p.goto(`${CC}${route}`, { waitUntil: 'networkidle' });
  await p.waitForTimeout(800);
  pages.push({ route, cp, p });
}
let stopped = false;
try {
  for (const { route, p } of pages) {
    const banner = await p.locator('text=CONTROL PLANE UNREACHABLE').count();
    const err = await p.locator('[role="alert"]').count();
    check(`before ${route}: loaded, no outage shown`, banner === 0 && (await p.locator('h1').count()) > 0, `alerts ${err}`);
  }
  check('before /: dashboard shows FRESH', (await freshBadges(first)) > 0);

  signal('STOP');
  stopped = true;
  console.log(`SIGSTOP control-plane members ${pids.join(',')}`);
  await first.waitForTimeout(22_000); // capabilities poll (15 s) plus request deadlines

  for (const { route, cp, p } of pages) {
    const banner = await p.locator('text=CONTROL PLANE UNREACHABLE').count();
    check(`during ${route}: CONTROL PLANE UNREACHABLE shown`, banner > 0);
    if (cp) {
      const grey = await p.locator('.grayscale').count();
      const errorState = await p.locator('text=/unreachable|timed out|BACKEND_/i').count();
      check(`during ${route}: last observation desaturated or error shown`, grey > 0 || errorState > 0, `grayscale ${grey}`);
      const fresh = await freshBadges(p);
      check(`during ${route}: no FRESH badge`, fresh === 0, `${fresh} visible`);
    }
    await p.screenshot({ path: `${OUT}/during-${route.replace(/[^\w]+/g, '_') || 'root'}.png` });
  }

  signal('CONT');
  stopped = false;
  console.log('SIGCONT control-plane members');
  await first.waitForTimeout(24_000);
  for (const { route, cp, p } of pages) {
    const banner = await p.locator('text=CONTROL PLANE UNREACHABLE').count();
    const grey = cp ? await p.locator('.grayscale').count() : 0;
    check(`after ${route}: recovered`, banner === 0 && grey === 0, `banner ${banner}, grayscale ${grey}`);
  }
  check('after /: dashboard shows FRESH again', (await freshBadges(first)) > 0);
} finally {
  if (stopped) signal('CONT');
  await browser.close();
}
writeFileSync(`${OUT}/wave1-disconnect-checks.json`, JSON.stringify(checks, null, 2) + '\n');
const failed = checks.filter((c) => !c.ok).length;
console.log(`${checks.length - failed}/${checks.length} checks passed`);
process.exit(failed ? 1 : 0);
