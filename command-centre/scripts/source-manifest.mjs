#!/usr/bin/env node
/**
 * Hash manifest of the Command Centre source. dh-src-digest/2 does not cover
 * .ts/.tsx files, so qualification records include this manifest as a hashed
 * artifact to bind the TypeScript source they tested.
 */
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join, relative } from 'node:path';
const ROOT = new URL('..', import.meta.url).pathname;
const SKIP = new Set(['node_modules', 'dist', '.git']);
const out = [];
(function walk(d) {
  for (const f of readdirSync(d).sort()) {
    if (SKIP.has(f)) continue;
    const p = join(d, f);
    if (statSync(p).isDirectory()) walk(p);
    else out.push(`${createHash('sha256').update(readFileSync(p)).digest('hex')}  ${relative(ROOT, p)}`);
  }
})(ROOT);
const body = out.join('\n') + '\n';
const root = createHash('sha256').update(body).digest('hex');
const text = `# command-centre source manifest (sha256 per file; root = sha256 of this list)\nroot sha256:${root}\n${body}`;
if (process.argv[2]) writeFileSync(process.argv[2], text);
console.log(`sha256:${root} over ${out.length} files`);
