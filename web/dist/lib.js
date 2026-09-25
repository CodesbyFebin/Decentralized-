// Rendering helpers. Every interpolated value is escaped unless it is the
// output of html``: host names, reasons and details come from hosts and are
// untrusted.

export class Raw {
  constructor(s) { this.s = s; }
  toString() { return this.s; }
}
export const raw = (s) => new Raw(s);

const ESC = { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" };
export const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ESC[c]);

function part(v) {
  if (v instanceof Raw) return v.s;
  if (Array.isArray(v)) return v.map(part).join("");
  if (v === null || v === undefined || v === false) return "";
  return esc(v);
}

export function html(strings, ...vals) {
  let out = strings[0];
  for (let i = 0; i < vals.length; i++) out += part(vals[i]) + strings[i + 1];
  return new Raw(out);
}

// ------------------------------------------------------------ truth basis

const BASIS = {
  OBSERVED: ["obs", "OBS", "Observed: measured by a host and reported in a signed observation"],
  DERIVED: ["der", "DER", "Derived: computed from observed values"],
  CONFIGURED: ["cfg", "CFG", "Configured: stated by policy or manifest — not a measurement"],
  PLANNED: ["pln", "PLN", "Planned: desired state that has not been observed yet"],
  UNKNOWN: ["unk", "UNK", "Unknown: not measured, or the measurement is stale"],
};

export function basis(b, note) {
  const [cls, label, title] = BASIS[b] || BASIS.UNKNOWN;
  return html`<span class="b b-${cls}" title="${note ? title + " — " + note : title}">${label}</span>`;
}

// A value with its basis. Missing values render as UNKNOWN — never as 0.
export function fact(value, b, note) {
  if (value === null || value === undefined || value === "") return unknown(note);
  return html`${value}${basis(b, note)}`;
}

export function unknown(note, text = "UNKNOWN") {
  return html`<span class="unk-txt" title="${note || "not measured"}">${text}</span>${basis("UNKNOWN", note)}`;
}

// ------------------------------------------------------------ state pills

const GOOD = new Set(["RUNNING", "ADMITTED", "HEALTHY", "FRESH", "ISSUED", "ROUTING", "ACTIVE", "ALLOWED", "RUNS", "READY", "OK", "VERIFIED",
  "OPERATIONAL", "LIVE", "COMMITTED", "LEADER", "PASS", "ALIVE", "NORMAL", "PRESENT", "ACCEPTED", "DB COMMITTED", "RAFT COMMITTED", "FOLLOWER"]);
const WARN = new Set(["STALE", "DEGRADED", "HELD", "HOLD", "PENDING", "DRAINING", "RENEWING", "REQUESTED", "STARTING", "PARTIAL", "SUSPECT",
  "OFFLINE-HOLD", "FROZEN-HOLD", "BUFFERED", "UNCOMMITTED", "JOURNAL ONLY", "DRAINED", "CANDIDATE", "BLOCKED", "MAY CONTINUE", "SUPERSEDED", "EXITED", "STOPPED", "SKIP"]);
const BAD = new Set(["FAILED", "REFUSED", "REVOKED", "EXPIRED", "EJECTED", "OOM-KILLED", "DEAD", "CORRUPT", "LEDGER-CORRUPT", "CLOCK-SKEW", "UNTRUSTED-PLANE",
  "FAIL", "BROKEN", "REJECTED", "MISSING", "LOST", "UNREACHABLE", "WITHDRAWN", "FROZEN"]);
const UNK = new Set(["UNKNOWN", "NONE", "NO EVIDENCE", "NOT OBSERVED", "NOT MEASURED", "NEVER"]);

export function tone(s) {
  const k = String(s || "").toUpperCase();
  if (!k || UNK.has(k)) return "unk";
  if (GOOD.has(k)) return "good";
  if (BAD.has(k)) return "bad";
  if (WARN.has(k)) return "warn";
  if (k.startsWith("RAFT COMMITTED") || k.startsWith("RAFT ")) return "good";
  if (k.startsWith("NOT ")) return "unk";
  return "neutral";
}

export function pill(s, title) {
  const label = s || "UNKNOWN";
  return html`<span class="pill ${tone(label)}" title="${title || ""}">${label}</span>`;
}

// ------------------------------------------------------------ formatting

export function bytes(n) {
  if (n === null || n === undefined) return "—";
  if (n < 0) return "—";
  const u = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v >= 100 || i === 0 ? Math.round(v) : v.toFixed(1)} ${u[i]}`;
}

export const cpu = (m) => (m === null || m === undefined ? "—" : m % 1000 === 0 ? `${m / 1000} cores` : `${m}m`);

export function dur(ms) {
  if (ms === null || ms === undefined || ms < 0) return "—";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s < 10 ? s.toFixed(1) : Math.round(s)}s`;
  const m = s / 60;
  if (m < 60) return `${Math.floor(m)}m ${Math.round(s % 60)}s`;
  const h = m / 60;
  if (h < 48) return `${Math.floor(h)}h ${Math.round(m % 60)}m`;
  return `${Math.floor(h / 24)}d ${Math.round(h % 24)}h`;
}

export const us = (u) => (u === null || u === undefined || u < 0 ? null : u < 1000 ? `${u}µs` : `${(u / 1000).toFixed(u < 10000 ? 2 : 1)}ms`);

// Ages are computed against the browser clock; they are labelled as such.
export function ago(ts, now = Date.now()) {
  if (!ts || ts <= 0) return null;
  const d = now - ts;
  if (d < -2000) return `in ${dur(-d)}`;
  return `${dur(Math.max(0, d))} ago`;
}

export function when(ts) {
  if (!ts || ts <= 0) return "—";
  const d = new Date(ts);
  const p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

export const shortId = (id) => (!id ? "" : id.length > 16 ? id.slice(0, 12) + "…" : id);
export const shortHash = (h) => (!h ? "" : h.startsWith("b3:") || h.startsWith("sha256:") ? h.slice(0, h.indexOf(":") + 11) + "…" : h.length > 18 ? h.slice(0, 14) + "…" : h);

export function hash(h, label) {
  if (!h) return html`<span class="faint">—</span>`;
  return html`<span class="hash copy" data-copy="${h}" title="${h} (click to copy)">${label || shortHash(h)}</span>`;
}

export function idLink(id, name, route) {
  if (!id) return html`<span class="faint">—</span>`;
  const label = name || shortId(id);
  if (route) return html`<a class="id" href="${route}" title="${id}">${label}</a>`;
  return html`<span class="id copy" data-copy="${id}" title="${id} (click to copy)">${label}</span>`;
}

export function checks(list) {
  if (!list || !list.length) return html`<span class="dim">no checks recorded</span>`;
  return html`<ul class="checks">${list.map((c) => html`<li><span class="${c.ok ? "ok" : "no"}">${c.ok ? "✓" : "✗"}</span><span class="nm">${c.name}</span><span class="${c.ok ? "dim" : ""}">${c.detail || ""}</span></li>`)}</ul>`;
}

export function kv(rows) {
  return html`<dl class="kv">${rows.filter(Boolean).map(([k, v]) => html`<dt>${k}</dt><dd>${v}</dd>`)}</dl>`;
}

export function table(cols, rows, opts = {}) {
  if (!rows.length) return html`<div class="empty">${opts.empty || "nothing to show"}</div>`;
  return html`<div class="tbl-wrap"><table class="t"><thead><tr>${cols.map((c) => html`<th class="${c.num ? "num" : ""}">${c.h}</th>`)}</tr></thead><tbody>${rows}</tbody></table></div>`;
}

export function ratioBar(used, total) {
  if (!total || total <= 0 || used === null || used === undefined || used < 0) return "";
  const f = Math.min(1, used / total);
  const col = f > 0.9 ? "var(--bad)" : f > 0.75 ? "var(--warn)" : "var(--accent-2)";
  return html`<div class="bar"><svg viewBox="0 0 100 6" preserveAspectRatio="none"><rect width="${(f * 100).toFixed(2)}" height="6" fill="${col}"/></svg></div>`;
}

export function panel(title, body, extra) {
  return html`<section class="panel"><header><h2>${title}</h2>${extra || ""}</header>${body}</section>`;
}

export function counter(label, value, b, sub, cls = "") {
  return html`<div class="counter ${cls}"><div class="lbl"><span>${label}</span>${b ? basis(b) : ""}</div><div class="num">${value === null || value === undefined ? html`<span class="unk-txt">UNKNOWN</span>` : value}</div>${sub ? html`<div class="sub">${sub}</div>` : ""}</div>`;
}

// Subsequence fuzzy score for the command palette (higher is better; -1 = no match).
export function fuzzy(q, s) {
  q = q.toLowerCase();
  s = s.toLowerCase();
  if (!q) return 0;
  const idx = s.indexOf(q);
  if (idx >= 0) return 1000 - idx - s.length / 100;
  let i = 0;
  let score = 0;
  let last = -1;
  for (let j = 0; j < s.length && i < q.length; j++) {
    if (s[j] === q[i]) {
      score += last === j - 1 ? 5 : 1;
      last = j;
      i++;
    }
  }
  return i === q.length ? score : -1;
}
