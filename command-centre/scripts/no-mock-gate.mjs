#!/usr/bin/env node
/**
 * No-mock production gate.
 *
 * Scans production source (server.ts, src/**) for patterns that indicate
 * synthetic operational data. Every hit is classified; any FORBIDDEN hit
 * fails the gate. Tests, fixtures and the explicit demo adapter are exempt
 * paths, and they are listed in the report rather than silently skipped.
 */
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, relative } from 'node:path';

const ROOT = new URL('..', import.meta.url).pathname;
const EXEMPT = [/^tests\//, /^src\/server\/demo\/fixtures\//, /^src\/server\/adapters\/demo\.ts$/];

/** [pattern, rule]. A rule returns 'forbidden' or an allow reason for the line. */
const RULES = [
  [/Math\.random/, () => 'forbidden'],
  [/\b99\.98\b|\b128400\b|\b1482\b|\b41 of 43\b|\b43 nodes\b|12\.4 ?TB|\b1\.2M\b(?! requests)/, () => 'forbidden'],
  [/Febin|Francis/i, () => 'forbidden'],
  [/\b(198\.51\.100|203\.0\.113|192\.0\.2)\.\d+/, () => 'forbidden'],
  [/dhcap1\.[A-Za-z0-9_-]{20,}/, () => 'forbidden'],
  [/\bfake\b|\bmock(ed|s)?\b|\bdummy\b|lorem ipsum/i, (l) => (/\/\/|\*|never|not |no /i.test(l) ? 'comment/negation' : 'forbidden')],
  [/\bseed(ed)?\b/i, (l) => (/\/\/|\*/.test(l) ? 'comment' : 'forbidden')],
  [/setTimeout\(/, (l) => (/poll|timer|abort|focus|copied|setCopied|deadline|timeout/i.test(l) ? 'timer (polling, abort, UI feedback)' : 'forbidden')],
  [/['"]VERIFIED['"]/, (l, f) => (/reality\.ts$|mapView\.ts$|client|states|ui\.tsx$|EvidenceView|DashboardView|SecurityView/.test(f) || /state ===|state:|\?/.test(l) ? 'state comparison / label' : 'forbidden')],
  [/\bdemo\b/i, (l, f) => (/mode === 'demo'|mode !== 'demo'|=== 'demo'/.test(l) ? 'explicit demo-mode check' : /config\.ts$|server\.ts$|session\.tsx$|capabilities\.ts$|AppShell|TopBar|Sidebar|TeamView|SettingsView|NodeDetailView|AppDetailView|DeployNewView|ui\.tsx$|mapView\.ts$|reality\.ts$|bff\.ts$/.test(f) ? 'explicit demo-mode handling / label' : 'forbidden')],
  [/placeholder/i, (l) => (/placeholder=|placeholder:|placeholder[,}]|\{placeholder\}/.test(l) ? 'input placeholder text / prop' : 'forbidden')]
];

function walk(dir, out = []) {
  for (const f of readdirSync(dir)) {
    if (f === 'node_modules' || f === 'dist' || f.startsWith('.')) continue;
    const p = join(dir, f);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(ts|tsx|mjs|js)$/.test(f)) out.push(p);
  }
  return out;
}

const files = [join(ROOT, 'server.ts'), ...walk(join(ROOT, 'src'))];
const findings = [];
const exempt = new Set();
for (const abs of files) {
  const f = relative(ROOT, abs);
  if (EXEMPT.some((r) => r.test(f))) {
    exempt.add(f);
    continue;
  }
  readFileSync(abs, 'utf8')
    .split('\n')
    .forEach((line, i) => {
      for (const [re, rule] of RULES) {
        if (re.test(line)) findings.push({ file: f, line: i + 1, pattern: re.source.slice(0, 40), verdict: rule(line, f), text: line.trim().slice(0, 140) });
      }
    });
}

const forbidden = findings.filter((x) => x.verdict === 'forbidden');
const report = [
  `# No-mock production gate`,
  ``,
  `Scanned ${files.length - exempt.size} production file(s); exempt (tests, fixtures, explicit demo adapter): ${[...exempt].join(', ') || 'none'}.`,
  ``,
  `| Verdict | Count |`,
  `| --- | --- |`,
  ...Object.entries(findings.reduce((a, x) => ((a[x.verdict] = (a[x.verdict] ?? 0) + 1), a), {})).map(([k, v]) => `| ${k} | ${v} |`),
  ``,
  forbidden.length ? `## FORBIDDEN (${forbidden.length})` : `Production mock leakage: NONE`,
  ...forbidden.map((x) => `- ${x.file}:${x.line} \`${x.text}\``),
  ``,
  `## All classified hits`,
  ...findings.map((x) => `- [${x.verdict}] ${x.file}:${x.line} — ${x.text.replace(/\|/g, '\\|')}`)
].join('\n');

if (process.argv.includes('--report')) writeFileSync(process.argv[process.argv.indexOf('--report') + 1], report + '\n');
console.log(forbidden.length ? report.split('## All')[0] : `no-mock gate: PASS — ${findings.length} hit(s) classified as allowed, 0 forbidden, ${exempt.size} exempt path(s).`);
process.exit(forbidden.length ? 1 : 0);
