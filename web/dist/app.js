// Decentralized.Host console shell: session, polling, routing, command
// palette, keyboard navigation and operator actions.
//
// Truthfulness rules this file enforces:
//   - the console never updates state optimistically: after an action it
//     shows the control plane's response and waits for the next view;
//   - when the control plane is unreachable, the last view stays on screen,
//     is visibly marked stale, and its age keeps counting.

import { html, raw, esc, pill, dur, when, fuzzy, shortId } from "./lib.js";
import * as S from "./screens.js";

const SCREENS = [
  ["overview", "Overview", "o", S.overview, "Operate"],
  ["runtime", "Runtime", "r", S.runtime, "Operate"],
  ["hosts", "Hosts", "h", S.hosts, "Operate"],
  ["workloads", "Workloads", "w", S.workloads, "Operate"],
  ["storage", "Storage", "s", S.storage, "Infrastructure"],
  ["mesh", "Mesh", "m", S.mesh, "Infrastructure"],
  ["edge", "Edge", "e", S.edge, "Infrastructure"],
  ["policy", "Policy", "p", S.policy, "Trust"],
  ["audit", "Audit", "a", S.audit, "Trust"],
  ["diagnostics", "Diagnostics", "d", S.diagnostics, "Trust"],
  ["capabilities", "Capabilities", "c", S.capabilities, "Trust"],
  ["federation", "Federation", "f", S.federation, "Trust"],
  ["conformance", "Conformance", "v", S.conformance, "Verify"],
  ["settings", "Settings", ",", S.settings, "Verify"],
];
const byId = new Map(SCREENS.map((s) => [s[0], s]));

const ui = {
  open: new Set(),
  interval: Number(sget("dh.interval")) || 2000,
  reduced: sget("dh.reduced") === "1",
  auditFilter: "",
  auditFull: null,
  auditVerify: null,
  conformance: null,
  lab: {},
};

const st = {
  token: null,
  session: { actions: [], blocks: null, expires: 0 },
  view: null,
  fetchedAt: 0, // browser clock at receipt
  error: null,
  errorAt: 0,
  timer: null,
  route: { screen: "overview", arg: "" },
  lastScreen: null,
};

function sget(k) { try { return localStorage.getItem(k); } catch { return null; } }
function sset(k, v) { try { localStorage.setItem(k, v); } catch { /* private mode */ } }

// ------------------------------------------------------------ session

function b64urlDecode(s) {
  s = s.replace(/-/g, "+").replace(/_/g, "/");
  while (s.length % 4) s += "=";
  const bin = atob(s);
  const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

// Decodes (does not verify) the session capability, to show what it grants
// and to disable buttons it cannot authorize. The server verifies.
function decodeSession(tok) {
  const out = { actions: [], blocks: null, expires: 0 };
  try {
    if (!tok.startsWith("dhcap1.")) return out;
    const blocks = JSON.parse(b64urlDecode(tok.slice(7)));
    out.blocks = blocks;
    const sets = blocks.map((b) => b.payload?.caveats?.actions || []);
    const all = ["api.read", "api.write", "api.admin"];
    out.actions = all.filter((a) => sets.every((set) => set.some((g) => g === "*" || g === a || (g.endsWith(".*") && a.startsWith(g.slice(0, -1))))));
    const exps = blocks.map((b) => b.payload?.caveats?.expires || 0).filter(Boolean);
    out.expires = exps.length ? Math.min(...exps) : 0;
  } catch { /* shown as undecodable */ }
  return out;
}

function loadToken() {
  const m = location.hash.match(/token=([^&]+)/);
  if (m) {
    const tok = decodeURIComponent(m[1]);
    try { sessionStorage.setItem("dh.token", tok); } catch { /* ignore */ }
    history.replaceState(null, "", location.pathname + "#/overview");
    return tok;
  }
  try { return sessionStorage.getItem("dh.token"); } catch { return null; }
}

// ------------------------------------------------------------ API

// A hung control plane must surface as an error, not as a view that stays
// on screen looking current.
const TIMEOUT_MS = 8000;

async function api(path, opts = {}) {
  const ctl = new AbortController();
  const timer = setTimeout(() => ctl.abort(), opts.timeout || TIMEOUT_MS);
  let res;
  try {
    res = await fetch(path, {
      signal: ctl.signal,
      method: opts.method || "GET",
    headers: { Authorization: `Bearer ${st.token}`, ...(opts.body !== undefined ? { "Content-Type": "application/json" } : {}) },
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      cache: "no-store",
      credentials: "omit",
    });
  } catch (e) {
    throw new Error(e.name === "AbortError" ? `no response within ${dur(opts.timeout || TIMEOUT_MS)}` : `unreachable: ${e.message}`);
  } finally {
    clearTimeout(timer);
  }
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { raw: text }; }
  if (!res.ok) {
    const msg = (data && (data.error || data.message)) || text || res.statusText;
    const err = new Error(msg);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

async function poll() {
  clearTimeout(st.timer);
  try {
    const v = await api("/api/v1/view");
    st.view = v;
    st.fetchedAt = Date.now();
    st.error = null;
  } catch (e) {
    if (e.status === 401) {
      st.error = e.message;
      signOut(`The control plane refused this session: ${e.message}`);
      return;
    }
    st.error = e.message || "network error";
    st.errorAt = Date.now();
  }
  render();
  st.timer = setTimeout(poll, ui.interval);
}

// ------------------------------------------------------------ routing

function parseRoute() {
  const h = location.hash.replace(/^#\/?/, "");
  const [screen, ...rest] = h.split("/");
  st.route = { screen: byId.has(screen) ? screen : "overview", arg: decodeURIComponent(rest.join("/") || "") };
}

function go(hash) {
  if (location.hash === hash) render();
  else location.hash = hash;
}

// ------------------------------------------------------------ rendering

const $ = (sel, root = document) => root.querySelector(sel);

function ctx() {
  const v = st.view || {};
  return {
    ui, route: st.route, now: Date.now(), session: st.session,
    nodes: new Map((v.nodes || []).map((n) => [n.id, n])),
    members: new Map((v.cluster?.members || []).map((m) => [m.id, m])),
    can: (a) => st.session.actions.includes(a),
  };
}

function renderShell() {
  $("#app").innerHTML = html`
    <div class="shell">
      <header class="top" role="banner">
        <div class="brand"><svg viewBox="0 0 32 32" aria-hidden="true"><path d="M9 8h7a8 8 0 010 16H9z" fill="none" stroke="#5ad1e6" stroke-width="2.6"/><circle cx="23" cy="9" r="2.4" fill="#3ecf8e"/></svg>
          Decentralized.Host <small id="cluster-name"></small></div>
        <span id="status-chips" style="display:contents"></span>
        <span class="spacer"></span>
        <button class="palette-btn" id="open-palette" aria-label="Open command palette">Search or run a command <span class="kbd">⌘K</span></button>
      </header>
      <nav class="side" id="nav" aria-label="Screens"></nav>
      <main id="main" tabindex="-1"><div class="page-h"><h1 id="title"></h1><span id="title-extra"></span></div><p class="page-sub" id="sub"></p><div class="toolbar" id="toolbar"></div><div class="body" id="body"></div></main>
    </div>
    <div class="toasts" id="toasts" aria-live="polite"></div>`.s;
  $("#open-palette").addEventListener("click", openPalette);
}

function renderNav(v) {
  const refused = v?.overview?.refused || 0;
  const drift = v?.overview?.drift || 0;
  const pending = v?.overview?.hosts?.pending || 0;
  const badge = { runtime: drift ? [drift, "warn"] : null, policy: refused ? [refused, "bad"] : null, hosts: pending ? [pending, ""] : null };
  let last = "";
  const parts = [];
  for (const [id, label, key, , grp] of SCREENS) {
    if (grp !== last) { parts.push(html`<div class="grp">${grp}</div>`); last = grp; }
    const b = badge[id];
    parts.push(html`<a href="#/${id}" class="${st.route.screen === id ? "on" : ""}" ${st.route.screen === id ? raw('aria-current="page"') : ""}>
      <span>${label}</span>${b ? html`<span class="count ${b[1]}">${b[0]}</span>` : html`<span class="k">g ${key}</span>`}</a>`);
  }
  $("#nav").innerHTML = html`${parts}`.s;
}

// A view is stale when the last poll failed or when no new view has arrived
// for several intervals (for example a request still hanging).
function viewStale() {
  if (!st.view) return false;
  return !!st.error || Date.now() - st.fetchedAt > Math.max(3 * ui.interval, 6000);
}

function renderChips(v) {
  const now = Date.now();
  const chips = [];
  const stale = viewStale();
  $("#main")?.classList.toggle("stale", stale);
  if (stale) {
    chips.push(html`<span class="chip bad" title="${st.error || "no new view received"}"><span class="dot"></span>${st.error ? "CONTROL PLANE UNREACHABLE" : "NO FRESH VIEW"} · showing view from ${dur(now - st.fetchedAt)} ago</span>`);
    $("#status-chips").innerHTML = html`${chips}`.s;
    return;
  }
  if (v) {
    const sb = v.servedBy || {};
    chips.push(html`<span class="chip ${sb.state === "leader" ? "good" : "warn"}" title="the member that served this view"><span class="dot"></span>${sb.state || "unknown"} ${shortId(sb.member)}</span>`);
    chips.push(html`<span class="chip" title="age of this view in the browser (refresh every ${dur(ui.interval)})"><span class="dot" style="background:var(--accent)"></span>view ${dur(now - st.fetchedAt)} old</span>`);
  }
  if (v?.cluster?.frozen) chips.push(html`<span class="chip bad"><span class="dot"></span>FROZEN</span>`);
  if (v?.overview) {
    const o = v.overview;
    const total = Object.entries(o.hosts || {}).filter(([k]) => k !== "revoked").reduce((a, [, n]) => a + n, 0);
    chips.push(html`<span class="chip ${o.hostsFresh < total ? "warn" : "good"}" title="hosts with a signed observation in the last 10s"><span class="dot"></span>${o.hostsFresh}/${total} hosts fresh</span>`);
    if (o.drift) chips.push(html`<span class="chip warn"><span class="dot"></span>drift ${o.drift}</span>`);
  }
  $("#status-chips").innerHTML = html`${chips}`.s;
  $("#cluster-name").textContent = v?.cluster?.name ? `· ${v.cluster.name}` : "";
}

function render() {
  if (!st.token) return renderLogin();
  if (!$("#main")) renderShell();
  const v = st.view;
  renderNav(v);
  renderChips(v);
  const [, label, , screen] = byId.get(st.route.screen);
  const main = $("#main");
  main.classList.toggle("stale", viewStale());
  if (st.lastScreen !== st.route.screen + "/" + st.route.arg) {
    st.lastScreen = st.route.screen + "/" + st.route.arg;
    $("#title").textContent = label + (st.route.arg ? ` · ${st.route.arg.startsWith("dh1") ? (ctx().nodes.get(st.route.arg)?.name || shortId(st.route.arg)) : st.route.arg}` : "");
    $("#sub").textContent = screen.sub || "";
    $("#toolbar").innerHTML = screen.toolbar ? screen.toolbar(ctx()).s : "";
    $("#toolbar").style.display = screen.toolbar ? "" : "none";
    main.scrollTop = 0;
    document.title = `${label} · Decentralized.Host`;
  }
  if (!v) {
    $("#body").innerHTML = st.error ? html`<div class="banner bad"><b>NO VIEW</b><span>${st.error}</span></div>`.s : `<div class="boot">Fetching the first view…</div>`;
    return;
  }
  // Do not replace inputs the operator is typing in; render after they leave.
  const active = document.activeElement;
  if (active && $("#body").contains(active) && /^(INPUT|TEXTAREA|SELECT)$/.test(active.tagName)) {
    st.deferred = true;
    return;
  }
  st.deferred = false;
  const scroll = main.scrollTop;
  try {
    $("#body").innerHTML = screen.body(v, ctx()).s;
  } catch (e) {
    console.error(e);
    $("#body").innerHTML = html`<div class="banner bad"><b>RENDER ERROR</b><span>${String(e)}</span></div>`.s;
  }
  main.scrollTop = scroll;
}

function renderLogin(msg) {
  st.lastScreen = null;
  $("#app").innerHTML = html`<section class="panel login">
    <h1>Decentralized.Host console</h1>
    <p class="dim">Paste a session capability (<span class="mono">dhcap1.…</span>). Create one on an operator machine with <span class="mono">dh console</span>, or <span class="mono">dh console --read-only</span>. The token stays in this tab and is sent only to this control plane.</p>
    ${msg ? html`<div class="banner bad"><span>${msg}</span></div>` : ""}
    <textarea class="in" id="token-in" placeholder="dhcap1.…" spellcheck="false" autocomplete="off"></textarea>
    <div class="actions" style="display:flex;justify-content:flex-end;margin-top:10px"><button class="btn primary" id="token-go">Open console</button></div>
  </section>`.s;
  $("#token-go").addEventListener("click", () => {
    const t = $("#token-in").value.trim().replace(/^.*#token=/, "");
    if (!t) return;
    try { sessionStorage.setItem("dh.token", t); } catch { /* ignore */ }
    start(t);
  });
  $("#token-in").focus();
}

function signOut(msg) {
  clearTimeout(st.timer);
  try { sessionStorage.removeItem("dh.token"); } catch { /* ignore */ }
  st.token = null;
  st.view = null;
  renderLogin(msg);
}

// ------------------------------------------------------------ toasts & modals

function toast(title, body, kind = "") {
  const el = document.createElement("div");
  el.className = `toast ${kind}`;
  el.innerHTML = html`<div class="t">${title}</div><div class="small">${body || ""}</div>`.s;
  $("#toasts")?.appendChild(el);
  setTimeout(() => el.remove(), kind === "bad" ? 12000 : 6000);
}

function modal(inner, onMount) {
  closeOverlay();
  const ov = document.createElement("div");
  ov.className = "overlay";
  ov.id = "overlay";
  ov.innerHTML = `<div class="modal" role="dialog" aria-modal="true">${inner.s}</div>`;
  ov.addEventListener("mousedown", (e) => { if (e.target === ov) closeOverlay(); });
  document.body.appendChild(ov);
  onMount?.(ov);
  (ov.querySelector("[autofocus]") || ov.querySelector("button"))?.focus();
  return ov;
}

function closeOverlay() { $("#overlay")?.remove(); }

// confirm shows exactly what will be sent before anything is sent.
function confirmAction({ title, consequence, method = "POST", path, body, typeToConfirm, danger }) {
  return new Promise((resolve) => {
    modal(html`<h3>${title}</h3><p>${consequence}</p>
      <div class="dim small">Request</div><pre class="code">${method} ${path}${body !== undefined ? "\n" + JSON.stringify(body, null, 2) : ""}</pre>
      ${typeToConfirm ? html`<label class="fld" style="margin-top:10px">Type <b class="mono">${typeToConfirm}</b> to confirm<input class="in" id="confirm-in" autocomplete="off" autofocus></label>` : ""}
      <div class="actions"><button class="btn" id="c-no">Cancel</button><button class="btn ${danger ? "danger" : "primary"}" id="c-yes" ${typeToConfirm ? raw("disabled") : ""}>${title}</button></div>`,
    (ov) => {
      const yes = ov.querySelector("#c-yes");
      ov.querySelector("#c-no").onclick = () => { closeOverlay(); resolve(false); };
      yes.onclick = () => { closeOverlay(); resolve(true); };
      const inp = ov.querySelector("#confirm-in");
      if (inp) inp.oninput = () => { yes.disabled = inp.value.trim() !== typeToConfirm; };
    });
  });
}

async function send(desc, path, body, method = "POST") {
  try {
    const res = await api(path, { method, body });
    toast(`${desc}: accepted`, res?.message || "Committed to desired state. Observed state follows when hosts report.", "good");
    poll();
    return res;
  } catch (e) {
    toast(`${desc}: refused`, e.message, "bad");
    return null;
  }
}

// ------------------------------------------------------------ actions

const nodeName = (id) => ctx().nodes.get(id)?.name || shortId(id);

const ACTIONS = {
  async approve(id) {
    if (await confirmAction({ title: "Approve host", consequence: html`Approving ${nodeName(id)} lets the scheduler place work on it. The host still admits work only under its own policy.`, path: `/api/v1/nodes/${id}/approve` }))
      send("Approve", `/api/v1/nodes/${encodeURIComponent(id)}/approve`, {});
  },
  async drain(id) {
    if (await confirmAction({ title: "Drain host", consequence: html`Replicas on ${nodeName(id)} are rescheduled elsewhere. Rolling budgets apply, and the host is not revoked.`, path: `/api/v1/nodes/${id}/drain` }))
      send("Drain", `/api/v1/nodes/${encodeURIComponent(id)}/drain`, {});
  },
  async undrain(id) {
    send("Undrain", `/api/v1/nodes/${encodeURIComponent(id)}/undrain`, {});
  },
  async revoke(id) {
    const name = nodeName(id);
    const body = { reason: "revoked from console" };
    if (await confirmAction({ title: "Revoke host", danger: true, typeToConfirm: name, body, path: `/api/v1/nodes/${id}/revoke`,
      consequence: html`${name} will be marked <b>REVOKED</b>, cut from the mesh and routing, and refused new work. Under its own policy it <b>may continue</b> running work it already admitted. Its replicas are rescheduled.` }))
      send("Revoke", `/api/v1/nodes/${encodeURIComponent(id)}/revoke`, body);
  },
  async ping(id) {
    try {
      const r = await api("/api/v1/mesh/ping", { method: "POST", body: { node: id } });
      modal(html`<h3>Mesh ping · ${nodeName(id)}</h3><p class="dim">Measured just now from the serving control-plane member over the WireGuard mesh.</p><pre class="code">${JSON.stringify(r, null, 2)}</pre><div class="actions"><button class="btn" id="c-no">Close</button></div>`, (ov) => { ov.querySelector("#c-no").onclick = closeOverlay; });
    } catch (e) { toast("Mesh ping failed", e.message, "bad"); }
  },
  async scale(app) {
    const a = (st.view?.apps || []).find((x) => x.name === app);
    modal(html`<h3>Scale ${app}</h3><p class="dim">Changes desired replicas (currently ${a?.replicas}). Hosts admit the new replicas under their own policy.</p>
      <label class="fld">Replicas<input class="in" id="scale-n" type="number" min="0" max="64" value="${a?.replicas ?? 1}" autofocus></label>
      <div class="actions"><button class="btn" id="c-no">Cancel</button><button class="btn primary" id="c-yes">Scale</button></div>`, (ov) => {
      ov.querySelector("#c-no").onclick = closeOverlay;
      ov.querySelector("#c-yes").onclick = () => {
        const n = Number(ov.querySelector("#scale-n").value);
        closeOverlay();
        send(`Scale ${app} to ${n}`, `/api/v1/apps/${encodeURIComponent(app)}/scale`, { replicas: n });
      };
    });
  },
  async delete(app) {
    if (await confirmAction({ title: "Delete application", danger: true, typeToConfirm: app, path: `/api/v1/apps/${app}/delete`,
      consequence: html`Hosts receive signed stops for every replica of <b>${app}</b>. Volumes and their committed snapshots are kept.` }))
      send(`Delete ${app}`, `/api/v1/apps/${encodeURIComponent(app)}/delete`, {});
  },
  async apply() {
    modal(html`<h3>Apply manifest</h3><p class="dim">A dh/v1 Application manifest in YAML. The control plane validates it, computes a plan and signs assignments. Nothing runs until hosts admit it.</p>
      <textarea class="in" id="apply-yaml" spellcheck="false" autofocus placeholder="apiVersion: dh/v1&#10;kind: Application&#10;metadata:&#10;  name: web&#10;spec:&#10;  replicas: 3&#10;  runtime: process&#10;  image: dh-beacon@b3:…"></textarea>
      <div class="actions"><button class="btn" id="c-no">Cancel</button><button class="btn primary" id="c-yes">Apply</button></div>`, (ov) => {
      ov.querySelector("#c-no").onclick = closeOverlay;
      ov.querySelector("#c-yes").onclick = async () => {
        const yaml = ov.querySelector("#apply-yaml").value;
        closeOverlay();
        const r = await send("Apply", "/api/v1/apply", { yaml });
        if (r) toast("Plan", typeof r === "object" ? (r.message || JSON.stringify(r).slice(0, 300)) : String(r));
      };
    });
  },
  async freeze(on) {
    const frozen = on === "1";
    if (await confirmAction({ title: frozen ? "Freeze control plane" : "Unfreeze control plane", danger: frozen, body: { frozen }, path: "/api/v1/freeze",
      consequence: frozen ? "Hosts hold work they already admitted and refuse all new work until you unfreeze. Use this when you suspect the control plane is compromised." : "Hosts resume normal admission under their own policies." }))
      send(frozen ? "Freeze" : "Unfreeze", "/api/v1/freeze", { frozen });
  },
  async invite() {
    toast("Invites are created from the CLI", "Run `dh node invite` on an operator machine. Join tokens carry the root key and a single-use capability, so they are not shown in a browser.");
  },
  async "fed-revoke"(digest) {
    // A revocation is signed by the cluster root key, which never enters a browser.
    modal(html`<h3>Revoke agreement</h3><p>Once revoked, the peer can no longer place work here, and placements under this agreement are stopped.</p>
      <p class="dim">The revocation is signed with the cluster root key, which the console never holds. Run this on the operator machine that holds the root key:</p>
      <pre class="code">dh federation revoke ${digest} --reason "…"</pre><div class="actions"><button class="btn" id="c-no">Close</button></div>`, (ov) => { ov.querySelector("#c-no").onclick = closeOverlay; });
  },
  async "audit-verify"() {
    try {
      ui.auditVerify = await api("/api/v1/audit/verify");
      toast("Audit verified", ui.auditVerify.ok ? `${ui.auditVerify.entries} entries, ${ui.auditVerify.checkpoints} checkpoints: chain intact` : "chain BROKEN — see details", ui.auditVerify.ok ? "good" : "bad");
    } catch (e) { toast("Verification failed", e.message, "bad"); }
    render();
  },
  async "audit-load"() {
    try {
      const r = await api("/api/v1/audit?limit=5000");
      ui.auditFull = r.entries || [];
      toast("Ledger loaded", `${ui.auditFull.length} entries (head #${r.head})`);
    } catch (e) { toast("Load failed", e.message, "bad"); }
    render();
  },
  async conformance() {
    toast("Running dh/v1 vectors", "on the serving member…");
    try { ui.conformance = await api("/api/v1/conformance"); } catch (e) { toast("Conformance run failed", e.message, "bad"); }
    render();
  },
  async "lab-mint"() {
    readLab();
    try {
      const r = await api("/api/v1/lab/mint", { method: "POST", body: labCaveats() });
      Object.assign(ui.lab, { token: r.token, blocks: r.blocks, result: null });
    } catch (e) { toast("Mint failed", e.message, "bad"); }
    render();
  },
  async "lab-attenuate"() {
    readLab();
    try {
      const r = await api("/api/v1/lab/attenuate", { method: "POST", body: { token: ui.lab.token, caveats: labCaveats() } });
      Object.assign(ui.lab, { token: r.token, blocks: r.blocks, result: null });
    } catch (e) { toast("Attenuation refused", e.message, "bad"); }
    render();
  },
  async "lab-verify"() {
    readLab();
    try {
      ui.lab.result = await api("/api/v1/lab/verify", { method: "POST", body: { token: ui.lab.token, request: { Action: ui.lab.reqAction, Resource: ui.lab.reqResource, CPUMilli: Number(ui.lab.reqCpu) || 0 } } });
    } catch (e) { toast("Verify failed", e.message, "bad"); }
    render();
  },
  signout() { signOut(); },
};

function readLab() {
  const val = (id) => $(`#${id}`)?.value ?? "";
  Object.assign(ui.lab, { actions: val("lab-actions"), resources: val("lab-resources"), cpu: val("lab-cpu"), reqAction: val("lab-req-action"), reqResource: val("lab-req-resource"), reqCpu: val("lab-req-cpu") });
}

function labCaveats() {
  const list = (s) => s.split(",").map((x) => x.trim()).filter(Boolean);
  return { actions: list(ui.lab.actions), resources: list(ui.lab.resources), cpuMaxMilli: Number(ui.lab.cpu) || 0 };
}

// ------------------------------------------------------------ command palette

function commands() {
  const v = st.view || {};
  const cmds = SCREENS.map(([id, label, key]) => ({ label: `Go to ${label}`, group: `g ${key}`, run: () => go(`#/${id}`) }));
  for (const n of v.nodes || []) cmds.push({ label: `Host ${n.name}`, group: "host", run: () => go(`#/hosts/${encodeURIComponent(n.id)}`) });
  for (const a of v.apps || []) {
    cmds.push({ label: `Runtime ${a.name}`, group: "app", run: () => go(`#/runtime/${encodeURIComponent(a.name)}`) });
    if (st.session.actions.includes("api.write")) cmds.push({ label: `Scale ${a.name}…`, group: "action", run: () => ACTIONS.scale(a.name) });
  }
  if (st.session.actions.includes("api.write")) {
    cmds.push({ label: "Apply manifest…", group: "action", run: () => ACTIONS.apply() });
    for (const n of v.nodes || []) if (n.identity !== "REVOKED") cmds.push({ label: `Drain ${n.name}…`, group: "action", run: () => ACTIONS.drain(n.id) });
  }
  if (st.session.actions.includes("api.admin")) {
    cmds.push(v.cluster?.frozen
      ? { label: "Unfreeze control plane…", group: "action", run: () => ACTIONS.freeze("0") }
      : { label: "Freeze control plane…", group: "action", run: () => ACTIONS.freeze("1") });
    for (const n of v.nodes || []) if (n.status === "pending") cmds.push({ label: `Approve ${n.name}…`, group: "action", run: () => ACTIONS.approve(n.id) });
  }
  cmds.push({ label: "Verify audit chain now", group: "action", run: () => { go("#/audit"); ACTIONS["audit-verify"](); } });
  cmds.push({ label: "Run conformance vectors", group: "action", run: () => { go("#/conformance"); ACTIONS.conformance(); } });
  cmds.push({ label: "Refresh view", group: "r", run: () => poll() });
  cmds.push({ label: ui.reduced ? "Allow motion" : "Reduce motion", group: "pref", run: () => setReduced(!ui.reduced) });
  cmds.push({ label: "Keyboard shortcuts", group: "?", run: () => help() });
  cmds.push({ label: "Sign out", group: "session", run: () => signOut() });
  return cmds;
}

function openPalette() {
  if (!st.token) return;
  const all = commands();
  let sel = 0;
  let list = all;
  closeOverlay();
  const ov = document.createElement("div");
  ov.className = "overlay";
  ov.id = "overlay";
  ov.innerHTML = `<div class="palette" role="dialog" aria-modal="true" aria-label="Command palette"><input placeholder="Type a command, host or app…" aria-label="Command" autocomplete="off" spellcheck="false"><ul role="listbox"></ul>
    <div class="foot"><span><span class="kbd">↑↓</span> move</span><span><span class="kbd">↵</span> run</span><span><span class="kbd">esc</span> close</span></div></div>`;
  document.body.appendChild(ov);
  const input = ov.querySelector("input");
  const ul = ov.querySelector("ul");
  const draw = () => {
    ul.innerHTML = list.slice(0, 60).map((c, i) => `<li role="option" data-i="${i}" class="${i === sel ? "sel" : ""}" aria-selected="${i === sel}"><span>${esc(c.label)}</span><span class="grp">${esc(c.group)}</span></li>`).join("") || `<li class="dim">no match</li>`;
    ul.querySelector(".sel")?.scrollIntoView({ block: "nearest" });
  };
  const run = (c) => { closeOverlay(); c?.run(); };
  input.addEventListener("input", () => {
    const q = input.value.trim();
    list = q ? all.map((c) => [fuzzy(q, c.label), c]).filter(([s]) => s >= 0).sort((a, b) => b[0] - a[0]).map(([, c]) => c) : all;
    sel = 0;
    draw();
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "ArrowDown") { sel = Math.min(list.length - 1, sel + 1); draw(); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(0, sel - 1); draw(); e.preventDefault(); }
    else if (e.key === "Enter") { run(list[sel]); e.preventDefault(); }
  });
  ul.addEventListener("click", (e) => { const li = e.target.closest("li[data-i]"); if (li) run(list[Number(li.dataset.i)]); });
  ov.addEventListener("mousedown", (e) => { if (e.target === ov) closeOverlay(); });
  draw();
  input.focus();
}

function help() {
  modal(html`<h3>Keyboard</h3><dl class="kv">
    <dt><span class="kbd">⌘K</span> <span class="kbd">/</span></dt><dd>command palette</dd>
    ${SCREENS.map(([, label, key]) => html`<dt><span class="kbd">g</span> <span class="kbd">${key}</span></dt><dd>${label}</dd>`)}
    <dt><span class="kbd">r</span></dt><dd>refresh now</dd><dt><span class="kbd">esc</span></dt><dd>close dialog</dd><dt><span class="kbd">?</span></dt><dd>this help</dd></dl>
    <div class="actions"><button class="btn" id="c-no">Close</button></div>`, (ov) => { ov.querySelector("#c-no").onclick = closeOverlay; });
}

function setReduced(on) {
  ui.reduced = on;
  sset("dh.reduced", on ? "1" : "0");
  document.documentElement.classList.toggle("reduced", on);
  render();
}

// ------------------------------------------------------------ events

let gPending = 0;

function onKey(e) {
  const typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement?.tagName || "");
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") { e.preventDefault(); openPalette(); return; }
  if (e.key === "Escape") { closeOverlay(); return; }
  if (typing || $("#overlay") || e.metaKey || e.ctrlKey || e.altKey) return;
  if (e.key === "/") { e.preventDefault(); openPalette(); return; }
  if (e.key === "?") { help(); return; }
  if (Date.now() - gPending < 1200) {
    gPending = 0;
    const s = SCREENS.find((x) => x[2] === e.key);
    if (s) { e.preventDefault(); go(`#/${s[0]}`); }
    return;
  }
  if (e.key === "g") { gPending = Date.now(); return; }
  if (e.key === "r") poll();
}

function onClick(e) {
  const copy = e.target.closest("[data-copy]");
  if (copy) {
    navigator.clipboard?.writeText(copy.dataset.copy).then(() => toast("Copied", copy.dataset.copy.slice(0, 80)), () => {});
    return;
  }
  const act = e.target.closest("[data-act]");
  if (act && !act.disabled) {
    e.preventDefault();
    ACTIONS[act.dataset.act]?.(act.dataset.arg);
    return;
  }
  if (e.target.closest("a, button, input, select, textarea, summary")) return;
  const tog = e.target.closest("[data-toggle]");
  if (tog) {
    const k = tog.dataset.toggle;
    ui.open.has(k) ? ui.open.delete(k) : ui.open.add(k);
    render();
    return;
  }
  const href = e.target.closest("[data-href]");
  if (href) { go(href.dataset.href); return; }
  const au = e.target.closest("[data-audit]");
  if (au) {
    const seq = Number(au.dataset.audit);
    const entry = (ui.auditFull || st.view?.audit?.entries || []).find((x) => x.seq === seq);
    if (entry) modal(html`<h3>Audit entry #${entry.seq}</h3><p class="dim">hash = H<sub>audit</sub>(entry without hash) · prev links to #${entry.seq - 1}</p><pre class="code">${JSON.stringify(entry, null, 2)}</pre><div class="actions"><button class="btn" id="c-no">Close</button></div>`, (ov) => { ov.querySelector("#c-no").onclick = closeOverlay; });
  }
}

function onInput(e) {
  if (e.target.id === "audit-filter") {
    ui.auditFilter = e.target.value;
    render();
  }
}

function onChange(e) {
  if (e.target.id === "set-interval") {
    ui.interval = Number(e.target.value);
    sset("dh.interval", String(ui.interval));
    poll();
  }
  if (e.target.id === "set-reduced") setReduced(e.target.checked);
}

// ------------------------------------------------------------ start

function start(tok) {
  st.token = tok;
  st.session = decodeSession(tok);
  parseRoute();
  $("#app").innerHTML = "";
  render();
  poll();
}

document.documentElement.classList.toggle("reduced", ui.reduced);
window.addEventListener("hashchange", () => {
  if (/token=/.test(location.hash)) { start(loadToken()); return; }
  parseRoute();
  render();
  $("#main")?.focus({ preventScroll: true });
});
document.addEventListener("keydown", onKey);
document.addEventListener("click", onClick);
document.addEventListener("input", onInput);
document.addEventListener("change", onChange);
document.addEventListener("focusout", () => setTimeout(() => { if (st.deferred) render(); }, 0));
// Keep ages honest between polls.
setInterval(() => { if (st.view) renderChips(st.view); }, 1000);

const tok = loadToken();
if (tok) start(tok);
else renderLogin();
