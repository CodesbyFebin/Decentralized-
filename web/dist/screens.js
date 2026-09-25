// Screens. Each screen renders from the control plane's /api/v1/view
// projection; nothing here invents state. Absent evidence renders as
// UNKNOWN, and every value carries its truth basis.

import {
  html, raw, basis, fact, unknown, pill, tone, bytes, cpu, dur, us, ago, when, shortId, hash, idLink,
  checks, kv, table, ratioBar, panel, counter,
} from "./lib.js";

const FRESH_MS = 10000;

// ------------------------------------------------------------ helpers

function hostLink(ctx, id) {
  if (!id) return html`<span class="faint">—</span>`;
  const n = ctx.nodes.get(id);
  if (n) return idLink(id, n.name || shortId(id), `#/hosts/${encodeURIComponent(id)}`);
  const m = ctx.members.get(id);
  if (m) return idLink(id, `cp:${shortId(id)}`);
  return idLink(id);
}

function freshness(f, ageMs) {
  if (!f || f === "NONE") return pill("NONE", "no signed observation received");
  return html`${pill(f)} <span class="dim small">${ageMs !== undefined && ageMs !== null ? dur(ageMs) + " old" : ""}</span>`;
}

function gen(n) {
  return n ? html`<span class="mono">g${n}</span>` : html`<span class="faint">—</span>`;
}

function emptyPanel(title, msg) {
  return panel(title, html`<div class="empty">${msg}</div>`);
}

function actionBtn(ctx, label, act, arg, cls = "", need = "api.write") {
  const ok = ctx.can(need);
  return html`<button class="btn sm ${cls}" data-act="${act}" data-arg="${arg || ""}" ${ok ? "" : raw("disabled")} title="${ok ? "" : "session capability lacks " + need}">${label}</button>`;
}

// ------------------------------------------------------------ overview

export const overview = {
  title: "Overview",
  sub: "Separate counts, never one health score. Desired state is what operators asked for, admitted is what hosts accepted under their own policy, and observed is what hosts measured and signed.",
  body(v, ctx) {
    const o = v.overview || {};
    const c = v.cluster || {};
    const hostsTotal = Object.values(o.hosts || {}).reduce((a, b) => a + b, 0);
    const revoked = (o.hosts || {}).revoked || 0;
    const stale = (v.nodes || []).filter((n) => n.lastObs?.freshness !== "FRESH" && n.identity !== "REVOKED");
    const driftCls = o.drift > 0 ? "warn" : "";
    return html`
      ${c.frozen ? html`<div class="banner bad"><b>CONTROL PLANE FROZEN</b><span>since ${when(c.frozenAt)}. Hosts hold admitted work and refuse new work.</span></div>` : ""}
      ${v.servedBy?.stale ? html`<div class="banner warn"><b>FOLLOWER VIEW</b><span>This view was served by a follower and may lag the leader.</span></div>` : ""}
      <div class="counters c5">
        ${counter("Desired", o.desired, "CONFIGURED", "replicas asked for")}
        ${counter("Admitted", o.admitted, "OBSERVED", "accepted by host policy")}
        ${counter("Observed", o.observed, "OBSERVED", "running, per signed observation")}
        ${counter("Refused", o.refused, "OBSERVED", "refused by a host", o.refused ? "bad" : "")}
        ${counter("Drift", o.drift, "DERIVED", "desired ≠ observed", driftCls)}
      </div>
      <div class="grid g3" style="margin-top:14px">
        ${panel("Hosts", html`
          ${kv([
            ["Enrolled", fact(hostsTotal, "CONFIGURED", "control-plane registry")],
            ...Object.entries(o.hosts || {}).sort().map(([k, n]) => [k, html`${pill(k.toUpperCase())} <span class="mono">${n}</span>`]),
            ["Fresh evidence", fact(`${o.hostsFresh ?? 0} / ${hostsTotal - revoked}`, "OBSERVED", "hosts with a signed observation in the last 10s")],
            ["Oldest fresh obs.", o.evidenceAgeMs ? fact(dur(o.evidenceAgeMs), "OBSERVED") : unknown("no fresh observation")],
          ])}
          ${stale.length ? html`<div class="banner warn" style="margin:10px 0 0"><b>${stale.length} without fresh evidence</b><span>${stale.map((n) => n.name).join(", ")}: their state is shown as last reported.</span></div>` : ""}
        `)}
        ${panel("Control plane", kv([
          ["Served by", html`${hostLink(ctx, v.servedBy?.member)} ${pill((v.servedBy?.state || "unknown").toUpperCase())}`],
          ["Leader", v.servedBy?.leader ? hostLink(ctx, v.servedBy.leader) : unknown("no leader known")],
          ["Quorum", fact(c.quorum, "OBSERVED", "raft state on the serving member")],
          ["Availability", fact(c.ha, "DERIVED")],
          ["Durability", html`${pill(c.durability?.split(" (")[0] || "UNKNOWN")} <span class="dim small">${c.durability?.includes("(") ? c.durability.slice(c.durability.indexOf("(")) : ""}</span>`],
          ["Raft index", fact(v.servedBy?.index, "OBSERVED")],
          ["Roster", fact(`v${c.rosterVersion} · ${c.members?.length || 0} member(s)`, "CONFIGURED", "root-signed")],
          ["Freeze", html`${c.frozen ? pill("FROZEN") : pill("NORMAL")} ${actionBtn(ctx, c.frozen ? "Unfreeze…" : "Freeze…", "freeze", c.frozen ? "0" : "1", c.frozen ? "" : "danger", "api.admin")}`],
        ]))}
        ${panel("Apps & storage", kv([
          ["Applications", fact(o.apps, "CONFIGURED")],
          ["Volumes", fact(o.volumes, "CONFIGURED")],
          ["Degraded volumes", o.volumesDegraded ? html`<span class="mono" style="color:var(--warn)">${o.volumesDegraded}</span>${basis("DERIVED", "from replica evidence")}` : fact(0, "DERIVED", "from replica evidence")],
          ["Artifacts", fact(v.artifacts?.length || 0, "CONFIGURED")],
          ["Protocol rejections", fact(v.rejections?.length || 0, "OBSERVED", "replayed, forged or tampered messages refused")],
          ["Audit head", html`<span class="mono">#${v.audit?.head}</span> ${v.audit?.verification?.ok ? pill("VERIFIED") : pill("BROKEN")}`],
        ]))}
      </div>
      ${o.incidents?.length ? panel("Active incidents", html`<ul class="timeline">${o.incidents.map((i) => html`<li><span class="when">now</span><span class="mk warn"></span><span class="what">${i}</span></li>`)}</ul>`) : ""}
      ${panel("Milestones", html`<div class="milestones">${(v.milestones || []).map((m) => html`
        <div class="ms"><div class="id2">${m.id}</div><div class="tt">${m.title}</div>${pill(m.state)}${basis(m.basis)}
          ${m.evidence?.length ? html`<ul>${m.evidence.map((e) => html`<li>${e}</li>`)}</ul>` : ""}
          ${m.gaps?.length ? html`<ul class="gaps">${m.gaps.map((e) => html`<li>${e}</li>`)}</ul>` : ""}</div>`)}</div>`,
        html`<span class="dim small">derived from live evidence, not a checklist</span>`)}
      ${panel("Recent audit", auditTimeline(ctx, (v.audit?.entries || []).slice(-12).reverse()), html`<a href="#/audit" class="small">full ledger →</a>`)}
    `;
  },
};

function auditTimeline(ctx, entries) {
  if (!entries.length) return html`<div class="empty">no audit entries</div>`;
  return html`<ul class="timeline">${entries.map((e) => {
    const t = /refuse|reject|revok|fail|corrupt|break/i.test(e.action) ? "bad" : /hold|freeze|drain|stale/i.test(e.action) ? "warn" : /allow|admit|commit|approve|verified/i.test(e.action) ? "good" : "";
    return html`<li><span class="when" title="${when(e.ts)}">${ago(e.ts, ctx.now)?.replace(" ago", "")}</span><span class="mk ${t}"></span>
      <span class="what"><b>${e.action}</b> <span class="dim">${e.resource}</span> <span class="faint small">#${e.seq} · ${e.source}</span><br><span class="small">${e.detail}</span></span></li>`;
  })}</ul>`;
}

// ------------------------------------------------------------ runtime

export const runtime = {
  title: "Runtime",
  sub: "Every replica shows its desired, admitted and observed state separately. If they disagree, that is drift, and a row shows how the host decided.",
  body(v, ctx) {
    const apps = (v.apps || []).filter((a) => !ctx.route.arg || a.name === ctx.route.arg);
    if (!apps.length) return emptyPanel("Applications", ctx.route.arg ? `no application named ${ctx.route.arg}` : "no applications — apply a manifest from Workloads");
    return html`${apps.map((a) => appRuntime(a, ctx))}`;
  },
};

function appRuntime(a, ctx) {
  const rows = (a.rows || []).map((r) => {
    const key = `rt:${a.name}:${r.assignment}:${r.node}`;
    const open = ctx.ui.open.has(key);
    const staleObs = r.freshness !== "FRESH";
    const status = r.status || "UNKNOWN";
    return html`
      <tr class="click ${open ? "open" : ""}" data-toggle="${key}">
        <td class="mono">${r.assignment}</td>
        <td>${hostLink(ctx, r.node)}</td>
        <td class="nowrap">${pill(r.desired, "desired")} ${gen(r.desiredGen)}</td>
        <td class="nowrap">${pill(r.admitted)} ${gen(r.admittedGen)}</td>
        <td class="nowrap">${staleObs && r.observed !== "UNKNOWN" ? pill("STALE", "observation is not fresh; last reported: " + r.observed) : pill(r.observed)} ${gen(r.observedGen)}</td>
        <td>${pill(status)}${r.code && r.code !== "ADMITTED" ? html` <span class="mono small dim">${r.code}</span>` : ""}</td>
        <td class="nowrap">${r.health ? html`${pill(r.health.ok ? "OK" : "FAILED", r.health.detail)} <span class="dim small">${us(r.health.latencyUs) || ""}</span>` : unknown("no health probe result")}</td>
        <td class="num mono">${r.restarts ?? "—"}</td>
        <td class="nowrap">${freshness(r.freshness, r.observedAt ? ctx.now - r.observedAt : null)}</td>
      </tr>
      ${open ? html`<tr class="detail"><td colspan="9">
        <div class="grid g2">
          <div>
            <div class="dim small" style="margin-bottom:6px">Admission decision by the host (${r.code || "no decision reported"})</div>
            <div style="margin-bottom:8px">${r.reason || html`<span class="dim">no reason reported</span>`}</div>
            ${checks(r.checks)}
          </div>
          <div>${kv([
            ["Runtime", r.containerId ? html`container <span class="mono">${r.containerId.slice(0, 12)}</span>` : r.pid ? html`pid <span class="mono">${r.pid}</span>` : unknown("no process reported")],
            ["Started", r.startedAt ? html`${when(r.startedAt)} <span class="dim">(${ago(r.startedAt, ctx.now)})</span>` : unknown()],
            ["Mesh port", r.meshPort ? html`<span class="mono">${r.meshPort}</span>` : unknown()],
            ["Health detail", r.health ? r.health.detail : unknown()],
            ["Observed at", r.observedAt ? html`${when(r.observedAt)} ${basis("OBSERVED")}` : unknown("no observation")],
            ["Evidence", hash(r.evidence)],
            r.federation ? ["Federated for", html`${r.federation.peer} ${hash(r.federation.agreement)}`] : null,
            ...(r.volumes || []).map((vo) => ["Volume " + vo.volumeId, html`last snapshot ${hash(vo.lastSnapshot)}${vo.restored ? html` · restored ${hash(vo.restored)}` : ""}`]),
          ])}</div>
        </div></td></tr>` : ""}`;
  });
  const drift = a.drift || 0;
  return html`
    <section class="panel">
      <header><h2><a href="#/workloads/${encodeURIComponent(a.name)}">${a.name}</a> · generation ${a.generation} · ${a.runtime}</h2>
        <span class="dim small mono" title="${a.image}">${shortId(a.image?.split("@")[0] || "")}@${hash(a.image?.split("@")[1])}</span></header>
      ${a.deleted ? html`<div class="banner warn"><b>DELETED</b><span>Replicas stop as hosts receive the signed stop.</span></div>` : ""}
      <div class="counters">
        ${counter("Desired", a.desired, "CONFIGURED", `${a.replicas} replica(s) in manifest`)}
        ${counter("Admitted", a.admitted, "OBSERVED", "host policy accepted")}
        ${counter("Observed", a.observed, "OBSERVED", "running at current generation")}
        ${counter("Drift", drift, "DERIVED", drift ? "replicas not yet converged" : "converged", drift ? "warn" : "")}
      </div>
      <div style="margin-top:12px">${table(
        [{ h: "Replica" }, { h: "Host" }, { h: "Desired" }, { h: "Admitted" }, { h: "Observed" }, { h: "Status" }, { h: "Health" }, { h: "Restarts", num: true }, { h: "Evidence" }],
        rows, { empty: "no replicas scheduled" })}</div>
    </section>`;
}

// ------------------------------------------------------------ hosts

export const hosts = {
  title: "Hosts",
  sub: "Revoking a host is not the same as losing it. A revoked host has identity REVOKED and its new admissions are BLOCKED, while work it already admitted MAY CONTINUE under its own policy.",
  body(v, ctx) {
    if (ctx.route.arg) return hostDetail(v, ctx, ctx.route.arg);
    const rows = (v.nodes || []).map((n) => html`
      <tr class="click" data-href="#/hosts/${encodeURIComponent(n.id)}">
        <td><a href="#/hosts/${encodeURIComponent(n.id)}">${n.name}</a><div class="id small">${shortId(n.id)} · ${n.meshIp || "no mesh IP"}</div></td>
        <td>${pill(n.identity)}</td>
        <td>${pill(n.status?.toUpperCase())}</td>
        <td>${pill(n.newAdmission)}</td>
        <td>${pill(n.existing)}</td>
        <td>${n.lastObs?.freshness === "FRESH" ? pill(n.mode === "normal" ? "NORMAL" : n.mode?.toUpperCase(), n.modeDetail) : unknown("no fresh observation", "UNKNOWN")}</td>
        <td class="small">${(n.tiers || []).join(", ")}${(n.roles || []).length ? html`<div class="dim">${n.roles.join(", ")}</div>` : ""}</td>
        <td class="nowrap small">${cpu(n.cpuMilli)} · ${bytes(n.memBytes)}${basis("CONFIGURED", "declared by the host at enrollment")}
          <div>${n.storage ? html`${bytes(n.storage.freeBytes)} disk free${basis("OBSERVED")}` : unknown("no storage observation")}</div></td>
        <td class="num mono">${n.workloads}</td>
        <td class="nowrap">${n.lastObs?.seq ? html`<span class="mono">#${n.lastObs.seq}</span> ` : ""}${freshness(n.lastObs?.freshness, n.lastObs?.ageMs)}</td>
      </tr>`);
    return html`${panel("Hosts", table(
      [{ h: "Host" }, { h: "Identity" }, { h: "Status" }, { h: "New admission" }, { h: "Existing runtime" }, { h: "Mode" }, { h: "Tiers / roles" }, { h: "Capacity" }, { h: "Wkld", num: true }, { h: "Last observation" }],
      rows, { empty: "no hosts have enrolled — create an invite with `dh node invite`" }),
      actionBtn(ctx, "Invite a host…", "invite", "", "", "api.admin"))}
      ${panel("Control-plane members", table(
        [{ h: "Member" }, { h: "Role" }, { h: "Suffrage" }, { h: "API" }, { h: "Raft" }, { h: "Mesh IP" }, { h: "WG binding" }],
        (v.cluster?.members || []).map((m) => html`<tr><td>${idLink(m.id)}</td><td>${m.leader ? pill("LEADER") : m.inRaft ? pill("FOLLOWER") : pill("NOT IN RAFT")}</td>
          <td class="small">${m.suffrage || "—"}</td><td class="mono small">${m.apiAddr}</td><td class="mono small">${m.raftAddr}</td><td class="mono small">${m.meshIp || "—"}</td>
          <td>${m.binding ? pill("OK") : pill("NONE", "no signed WireGuard binding")}</td></tr>`)))}`;
  },
};

function hostDetail(v, ctx, id) {
  const n = (v.nodes || []).find((x) => x.id === id);
  if (!n) return emptyPanel("Host", `no host with id ${id}`);
  const fresh = n.lastObs?.freshness === "FRESH";
  const p = n.policy || {};
  const f = n.facts;
  const workloads = [];
  for (const a of v.apps || []) for (const r of a.rows || []) if (r.node === n.id) workloads.push({ a, r });
  return html`
    <div class="toolbar"><a href="#/hosts" class="btn sm">← Hosts</a>
      ${n.status === "pending" ? actionBtn(ctx, "Approve", "approve", n.id, "primary", "api.admin") : ""}
      ${n.status === "draining" || n.status === "drained" ? actionBtn(ctx, "Undrain", "undrain", n.id) : n.identity !== "REVOKED" ? actionBtn(ctx, "Drain", "drain", n.id) : ""}
      ${actionBtn(ctx, "Mesh ping", "ping", n.id, "", "api.read")}
      ${n.identity !== "REVOKED" ? actionBtn(ctx, "Revoke…", "revoke", n.id, "danger", "api.admin") : ""}
    </div>
    ${!fresh ? html`<div class="banner warn"><b>STALE EVIDENCE</b><span>The last signed observation is ${n.lastObs?.ageMs ? dur(n.lastObs.ageMs) + " old" : "missing"}. Everything below is as last reported.</span></div>` : ""}
    ${n.mode && n.mode !== "normal" ? html`<div class="banner ${n.mode === "ledger-corrupt" || n.mode === "untrusted-plane" ? "bad" : "warn"}"><b>${n.mode.toUpperCase()}</b><span>${n.modeDetail}</span></div>` : ""}
    <div class="grid g3">
      ${panel("Identity", kv([
        ["Name", html`<b>${n.name}</b>`],
        ["dh1 id", idLink(n.id, n.id)],
        ["Identity", pill(n.identity)],
        ["New admission", pill(n.newAdmission)],
        ["Existing runtime", pill(n.existing)],
        ["Status", pill(n.status?.toUpperCase())],
        ["Joined", when(n.joinedAt)],
        ["Approved", when(n.approvedAt)],
        n.revokedAt ? ["Revoked", when(n.revokedAt)] : null,
        ["Keys", html`${(n.keys || []).map((k) => html`<div><span class="mono small" title="${k.pub}">${k.pub.slice(0, 16)}…</span> ${k.revoked ? pill("REVOKED", k.reason) : k.until ? pill("GRACE", "valid until " + when(k.until)) : pill("ACTIVE")}</div>`)}`],
      ]))}
      ${panel("Last signed observation", kv([
        ["Freshness", freshness(n.lastObs?.freshness, n.lastObs?.ageMs)],
        ["Sequence", n.lastObs?.seq ? html`<span class="mono">#${n.lastObs.seq}</span> <span class="dim small">(committed #${n.lastObs.committedSeq})</span>` : unknown()],
        ["Host timestamp", when(n.lastObs?.hostTs)],
        ["Received", when(n.lastObs?.receivedAt)],
        ["Source", n.lastObs?.source || "—"],
        ["Buffered", n.lastObs?.buffered ? pill("BUFFERED", "queued while the control plane was unreachable") : html`<span class="dim">no</span>`],
        ["Mode", n.mode ? html`${pill(n.mode.toUpperCase())}` : unknown()],
        ["Host ledger", n.ledger?.seq ? html`<span class="mono">#${n.ledger.seq}</span> ${hash(n.ledger.hash)}` : unknown()],
        ["Evidence", hash(n.evidence)],
      ]))}
      ${panel("Placement", kv([
        ["Region / zone", `${n.region || "—"} / ${n.zone || "—"}`],
        ["Address", html`<span class="mono">${n.host || "—"}</span>`],
        ["Mesh IP", html`<span class="mono">${n.meshIp || "—"}</span>`],
        ["Tiers", (n.tiers || []).join(", ") || "—"],
        ["Roles", (n.roles || []).join(", ") || "—"],
        ["Declared capacity", html`${cpu(n.cpuMilli)} · ${bytes(n.memBytes)}${basis("CONFIGURED", "stated by the host at enrollment")}`],
        ["Workloads", fact(n.workloads, "OBSERVED")],
      ]))}
    </div>
    ${panel("Sovereign policy", html`<p class="dim small" style="margin:0 0 10px">Signed by the host itself. The control plane can read it but cannot change it.</p>${policyKV(p)}`, basis(n.policyBasis || "CONFIGURED", "the host's own signed statement"))}
    <div class="grid g2">
      ${panel("Measured facts", f ? kv([
        ["OS / arch", `${f.os} / ${f.arch}`],
        ["Kernel", f.kernel || "—"],
        ["CPUs", fact(f.cpus, "OBSERVED")],
        ["Memory", fact(bytes(f.memBytes), "OBSERVED")],
        ["Docker", f.docker ? fact(f.docker, "OBSERVED") : html`<span class="unk-txt">NOT AVAILABLE</span>${basis("OBSERVED")}`],
        ["UDP 443", f.udp443 ? fact("LISTENING", "OBSERVED") : html`<span class="unk-txt">NOT LISTENING</span>${basis("OBSERVED")}`],
        ["Runtimes", fact((f.runtimes || []).join(", "), "OBSERVED")],
        ["Clock skew", fact(`${f.clockSkewMs} ms`, "OBSERVED", "host clock minus bundle issue time")],
        ...(f.probes || []).map((pr) => [pr.name, pr.ok ? html`${pr.detail}${basis("OBSERVED")}` : html`<span class="unk-txt">${pr.detail.toUpperCase()}</span>${basis("OBSERVED")}`]),
      ]) : html`<div class="empty">no facts reported</div>`)}
      ${panel("Storage", n.storage ? html`${kv([
        ["Capacity", fact(bytes(n.storage.capacityBytes), "OBSERVED")],
        ["Free", fact(bytes(n.storage.freeBytes), "OBSERVED")],
        ["Used by CAS", fact(bytes(n.storage.usedBytes), "OBSERVED")],
        ["Quota", fact(bytes(n.storage.quotaBytes), "CONFIGURED")],
        ["Chunks", fact(n.storage.chunks, "OBSERVED")],
        ["Integrity failures", n.storage.corrupt ? html`<span style="color:var(--bad)" class="mono">${n.storage.corrupt}</span>${basis("OBSERVED")}` : fact(0, "OBSERVED")],
        ["Merkle root", n.storage.merkleRoot ? hash(n.storage.merkleRoot) : html`<span class="dim small">not reported</span>`],
      ])}${ratioBar(n.storage.usedBytes, n.storage.quotaBytes)}` : html`<div class="empty">no storage observation</div>`)}
    </div>
    ${panel("Mesh", meshHostBody(n, ctx))}
    ${panel("Workloads on this host", table([{ h: "Replica" }, { h: "Admitted" }, { h: "Observed" }, { h: "Code" }, { h: "Reason" }],
      workloads.map(({ r }) => html`<tr><td class="mono"><a href="#/runtime/${encodeURIComponent(r.assignment.split("/")[0])}">${r.assignment}</a></td>
        <td>${pill(r.admitted)} ${gen(r.admittedGen)}</td><td>${pill(r.observed)} ${gen(r.observedGen)}</td><td class="mono small">${r.code}</td><td class="small">${r.reason}</td></tr>`),
      { empty: "no replicas assigned to this host" }))}`;
}

function policyKV(p) {
  return kv([
    ["Sovereign", p.sovereign ? pill("YES") : pill("NO")],
    ["Accepted tiers", (p.acceptTiers || []).join(", ") || "none"],
    ["Runtimes", (p.allowRuntimes || []).join(", ") || "none"],
    ["Caps", `${p.maxWorkloads} workloads · ${cpu(p.maxCpuMilli)} · ${bytes(p.maxMemBytes)}`],
    ["Digest-pinned images", p.denyImagesWithoutDigest ? "required" : "not required"],
    ["Artifact signature", p.requireImageSignature ? "required" : "not required"],
    ["Federated work", p.allowFederated ? "accepted" : "refused"],
    ["Remote exec", p.allowExec ? "allowed" : "refused"],
    ["When control plane is stale", p.offlineAdmission === "stop" ? "stop admitted work" : "hold admitted work, refuse new"],
    ["Max clock skew", `${p.maxClockSkewMs} ms`],
    ["Storage quota", bytes(p.storageQuotaBytes)],
  ]);
}

function meshHostBody(n, ctx) {
  const m = n.mesh;
  if (!m) return html`<div class="empty">no mesh observation</div>`;
  return html`${kv([
    ["Device", fact(m.device, "OBSERVED")],
    ["WireGuard key", html`<span class="mono small">${m.wgPub}</span>`],
    ["Listen port", html`<span class="mono">${m.listenPort}</span>`],
    ["Gossip", html`${m.gossip} · ${m.members} member(s) ${basis("OBSERVED")}`],
  ])}
  <div style="margin-top:10px">${peerTable(m.peers || [], ctx)}</div>`;
}

function peerTable(peers, ctx) {
  return table(
    [{ h: "Peer" }, { h: "Binding" }, { h: "Endpoint" }, { h: "Handshake" }, { h: "RTT" }, { h: "Gossip" }, { h: "Rx", num: true }, { h: "Tx", num: true }],
    peers.map((p) => {
      const hs = p.lastHandshake ? ctx.now - p.lastHandshake : null;
      return html`<tr><td>${hostLink(ctx, p.node)}</td>
        <td>${p.bindingOk ? pill("OK", "signed wg-binding verifies") : pill("FAILED", "no verifying signed binding")}</td>
        <td class="mono small">${p.endpoint || "—"}</td>
        <td class="nowrap">${p.lastHandshake ? html`${ago(p.lastHandshake, ctx.now)}${hs > 180000 ? html` ${pill("OLD")}` : ""}${basis("OBSERVED")}` : html`<span class="unk-txt">NEVER</span>${basis("OBSERVED")}`}</td>
        <td class="nowrap">${p.rttUs >= 0 && p.rttAt ? html`${us(p.rttUs)}${basis("OBSERVED", "measured over the mesh " + ago(p.rttAt, ctx.now))}` : html`<span class="unk-txt">NOT MEASURED</span>`}</td>
        <td>${pill(p.gossip === "not-a-gossip-member" ? "N/A" : p.gossip?.toUpperCase(), p.gossip)}</td>
        <td class="num mono small">${bytes(p.rxBytes)}</td><td class="num mono small">${bytes(p.txBytes)}</td></tr>`;
    }), { empty: "no peers configured" });
}

// ------------------------------------------------------------ workloads

export const workloads = {
  title: "Workloads",
  sub: "Applications as desired: manifests, revisions and scheduler plans. Changing desired state creates signed intent. Nothing is shown as running until a host observes it.",
  toolbar(ctx) {
    return html`${actionBtn(ctx, "Apply manifest…", "apply", "", "primary")}`;
  },
  body(v, ctx) {
    const apps = (v.apps || []).filter((a) => !ctx.route.arg || a.name === ctx.route.arg);
    if (!apps.length) return emptyPanel("Applications", ctx.route.arg ? `no application named ${ctx.route.arg}` : "no applications");
    return html`${apps.map((a) => {
      const s = a.manifest?.spec || {};
      const plan = a.plan?.plan;
      return html`<section class="panel">
        <header><h2>${a.name} · generation ${a.generation}</h2>
          <span>${a.federation ? pill("FEDERATED IN", "placed here by " + a.federation.peer) : ""}${a.outbound ? pill("FEDERATED OUT", "placed on " + a.outbound.peer) : ""}
          ${actionBtn(ctx, "Scale…", "scale", a.name)} ${actionBtn(ctx, "Delete…", "delete", a.name, "danger")} <a class="btn sm" href="#/runtime/${encodeURIComponent(a.name)}">Runtime →</a></span></header>
        <div class="grid g3">
          <div>${kv([
            ["Image", html`<span class="mono small">${a.image}</span>`],
            ["Runtime", a.runtime],
            ["Replicas", fact(a.replicas, "CONFIGURED")],
            ["Resources", `${cpu(s.resources?.cpuMilli)} · ${bytes(s.resources?.memBytes)} per replica`],
            ["Placement", `tiers ${(s.placement?.tiers || []).join(", ") || "any"} · spread ${s.placement?.spread || "—"} · anti-affinity ${s.placement?.antiAffinity || "—"}`],
            ["Health", s.health?.http ? `${s.health.http} every ${dur(s.health.intervalMs)}` : "none"],
            ["Update", `${s.update?.strategy || "rolling"}, max unavailable ${s.update?.maxUnavailable ?? 1}`],
            ["Manifest hash", hash(a.hash)],
          ])}</div>
          <div>
            <div class="dim small" style="margin-bottom:6px">Ingress</div>
            ${(a.ingress || []).length ? html`${a.ingress.map((i) => html`<div><span class="mono">${i.host}</span> → ${i.port} · TLS ${i.tls}</div>`)}` : html`<span class="dim">none</span>`}
            <div class="dim small" style="margin:10px 0 6px">Volumes</div>
            ${(s.volumes || []).length ? html`${s.volumes.map((vo) => html`<div><span class="mono">${vo.name}</span> ${bytes(vo.sizeBytes)} at ${vo.mount} · ×${vo.durability?.replicas}</div>`)}` : html`<span class="dim">none</span>`}
          </div>
          <div>
            <div class="dim small" style="margin-bottom:6px">Revisions</div>
            <ul class="timeline">${(a.history || []).slice().reverse().slice(0, 8).map((h) => html`<li><span class="when">g${h.generation}</span><span class="mk"></span><span class="what">${h.change}<br><span class="faint small">${when(h.ts)} · ${h.actor}</span></span></li>`)}</ul>
          </div>
        </div>
        ${plan ? html`<details class="raw"><summary>Scheduler plan (${a.plan.at ? ago(a.plan.at, ctx.now) : ""}) ${basis("PLANNED")}</summary>
          ${table([{ h: "Replica" }, { h: "Host" }, { h: "Decision" }], (plan.replicas || []).map((r) => html`<tr><td class="mono">r${r.replica}</td><td>${r.node ? hostLink(ctx, r.node) : unknown("unplaced", "UNPLACED")}</td><td class="small">${r.kept ? pill("KEPT") : ""} ${r.reason}</td></tr>`))}</details>` : ""}
        <details class="raw"><summary>Normalized manifest</summary><pre class="code">${JSON.stringify(a.manifest, null, 2)}</pre></details>
      </section>`;
    })}`;
  },
};

// ------------------------------------------------------------ storage

export const storage = {
  title: "Storage",
  sub: "Replica counts come from signed replica evidence, not from configuration. A snapshot is committed only when a quorum of hosts has proven it holds every chunk, verified.",
  body(v, ctx) {
    const reporting = (v.nodes || []).filter((n) => n.storage && n.identity !== "REVOKED");
    const silent = (v.nodes || []).filter((n) => !n.storage && n.identity !== "REVOKED");
    const sum = (k) => reporting.reduce((a, n) => a + (n.storage[k] || 0), 0);
    const corrupt = sum("corrupt");
    const vols = v.volumes || [];
    const degraded = vols.filter((x) => x.state !== "HEALTHY");
    return html`
      <div class="counters c5">
        ${counter("Capacity", bytes(sum("capacityBytes")), "DERIVED", `sum of ${reporting.length} reporting host(s)`)}
        ${counter("Free", bytes(sum("freeBytes")), "DERIVED", silent.length ? `${silent.length} host(s) not reporting` : "all hosts reporting")}
        ${counter("CAS used", bytes(sum("usedBytes")), "DERIVED", `${sum("chunks")} chunk copies`)}
        ${counter("Integrity failures", corrupt, "OBSERVED", "quarantined objects", corrupt ? "bad" : "")}
        ${counter("Degraded volumes", degraded.length, "DERIVED", `${vols.length} volume(s)`, degraded.length ? "warn" : "")}
      </div>
      ${panel("Volumes", table(
        [{ h: "Volume" }, { h: "State" }, { h: "Verified replicas" }, { h: "Committed snapshot" }, { h: "Size" }, { h: "Members" }, { h: "Detail" }],
        vols.map((vo) => {
          const key = `vol:${vo.id}`;
          const open = ctx.ui.open.has(key);
          const c = vo.committedRef;
          return html`<tr class="click ${open ? "open" : ""}" data-toggle="${key}"><td class="mono">${vo.id}</td><td>${pill(vo.state)}</td>
            <td class="mono">${vo.verified} / ${vo.durability?.replicas}${basis("OBSERVED", "hosts with signed replica evidence for the committed snapshot")}</td>
            <td>${c ? html`${hash(c.id)} <span class="dim small">${ago(c.committedAt, ctx.now) || ""}</span>` : unknown("no committed snapshot", "NONE")}</td>
            <td class="small nowrap">${c ? `${bytes(c.bytes)} · ${c.chunks} chunk(s)` : "—"}</td>
            <td class="small">${(vo.members || []).map((m) => hostLink(ctx, m)).reduce((acc, x, i) => html`${acc}${i ? ", " : ""}${x}`, "")}</td>
            <td class="small">${vo.detail}</td></tr>
            ${open ? html`<tr class="detail"><td colspan="7">${snapshotDetail(vo, ctx)}</td></tr>` : ""}`;
        }), { empty: "no volumes — declare one under spec.volumes" }))}
      <div class="grid g2">
        ${panel("Host stores", table([{ h: "Host" }, { h: "Free" }, { h: "Used / quota" }, { h: "Chunks", num: true }, { h: "Corrupt", num: true }],
          (v.nodes || []).map((n) => html`<tr><td>${hostLink(ctx, n.id)}</td>
            ${n.storage ? html`<td class="small">${bytes(n.storage.freeBytes)} of ${bytes(n.storage.capacityBytes)}</td>
              <td class="small">${bytes(n.storage.usedBytes)} / ${bytes(n.storage.quotaBytes)}${ratioBar(n.storage.usedBytes, n.storage.quotaBytes)}</td>
              <td class="num mono">${n.storage.chunks}</td><td class="num mono" style="${n.storage.corrupt ? "color:var(--bad)" : ""}">${n.storage.corrupt}</td>`
              : html`<td colspan="4">${unknown("no storage observation")}</td>`}</tr>`)))}
        ${panel("Repairs (anti-entropy)", repairs(v.repairs || [], ctx))}
      </div>
      ${panel("Artifacts", table([{ h: "Name" }, { h: "Digest" }, { h: "Size" }, { h: "Holders" }, { h: "Attested" }],
        (v.artifacts || []).map((a) => html`<tr><td>${a.name}</td><td>${hash(a.digest)}</td><td class="small">${bytes(a.bytes)} · ${a.chunks} chunks</td>
          <td class="small">${(a.holderNames || []).join(", ") || html`<span class="dim">control plane only</span>`}</td><td>${a.attested ? pill("ATTESTED") : pill("UNSIGNED")}</td></tr>`),
        { empty: "no artifacts — push one with `dh artifact push`" }))}`;
  },
};

function snapshotDetail(vo, ctx) {
  const snaps = (vo.snapshots || []).slice().reverse().slice(0, 6);
  if (!snaps.length) return html`<span class="dim">no snapshots yet</span>`;
  return html`${snaps.map((s) => html`<div style="margin-bottom:10px"><div>${pill(s.state?.toUpperCase())} ${hash(s.id)} <span class="dim small">root ${hash(s.root)} · ${bytes(s.bytes)} · ${s.chunks} chunks · ${s.files} files · by ${shortId(s.creator)} · ${when(s.ts)}</span></div>
    ${table([{ h: "Replica host" }, { h: "Chunks" }, { h: "Missing" }, { h: "Corrupt" }, { h: "Verified" }, { h: "Evidence" }],
      Object.values(s.evidence || {}).map((e) => html`<tr><td>${hostLink(ctx, e.node)}</td><td class="mono">${e.chunks}</td>
        <td class="mono" style="${e.missing ? "color:var(--warn)" : ""}">${e.missing}</td><td class="mono" style="${e.corrupt ? "color:var(--bad)" : ""}">${e.corrupt}</td>
        <td class="small">${ago(e.verifiedAt, ctx.now)}${basis("OBSERVED")}</td><td>${hash(e.evidence)}</td></tr>`), { empty: "no replica evidence" })}</div>`)}`;
}

function repairs(list, ctx) {
  if (!list.length) return html`<div class="empty">no repairs recorded</div>`;
  return html`<ul class="timeline">${list.slice().reverse().slice(0, 12).map((r) => {
    const items = r.r.items || [];
    const ok = items.every((i) => i.verified);
    return html`<li><span class="when">${ago(r.received, ctx.now)?.replace(" ago", "")}</span><span class="mk ${ok ? "good" : "bad"}"></span>
      <span class="what"><b>${r.r.trigger}</b> ${hostLink(ctx, r.r.node)} <span class="dim">${r.r.volumeId || ""}</span><br>
      <span class="small">${items.length} object(s): ${items.filter((i) => i.previous === "missing").length} missing, ${items.filter((i) => i.previous === "corrupt").length} corrupt → ${ok ? "all verified" : "some failed"}</span> ${hash(r.evidence)}</span></li>`;
  })}</ul>`;
}

// ------------------------------------------------------------ mesh

export const mesh = {
  title: "Mesh",
  sub: "What each host reports about its WireGuard peers. A line is solid only when a handshake was observed in the last 3 minutes. Latency shows only where it was measured.",
  body(v, ctx) {
    const nodes = (v.nodes || []).filter((n) => n.identity !== "REVOKED");
    return html`
      ${panel("Topology", topology(v, ctx), html`<span class="dim small">static layout · no animation implies state</span>`)}
      ${nodes.map((n) => panel(html`${n.name} · ${n.meshIp || "no mesh IP"} ${n.lastObs?.freshness !== "FRESH" ? pill("STALE") : ""}`, meshHostBody(n, ctx)))}
      ${(v.nodes || []).some((n) => n.identity === "REVOKED") ? panel("Revoked hosts", html`<p class="dim">Revoked hosts are removed from every peer set and from routing: ${(v.nodes || []).filter((n) => n.identity === "REVOKED").map((n) => n.name).join(", ")}</p>`) : ""}`;
  },
};

function topology(v, ctx) {
  const nodes = (v.nodes || []).filter((n) => n.identity !== "REVOKED");
  const members = v.cluster?.members || [];
  const all = [...members.map((m) => ({ id: m.id, name: "cp:" + shortId(m.id).slice(3, 9), ip: m.meshIp, cp: true })), ...nodes.map((n) => ({ id: n.id, name: n.name, ip: n.meshIp, stale: n.lastObs?.freshness !== "FRESH" }))];
  if (!all.length) return html`<div class="empty">no mesh members</div>`;
  const W = 760, H = 380, cx = W / 2, cy = H / 2, R = Math.min(W, H) / 2 - 50;
  const pos = new Map();
  all.forEach((n, i) => {
    const a = (i / all.length) * Math.PI * 2 - Math.PI / 2;
    pos.set(n.id, [cx + R * Math.cos(a) * 1.45, cy + R * Math.sin(a)]);
  });
  const seen = new Set();
  const lines = [];
  for (const n of nodes) {
    for (const p of n.mesh?.peers || []) {
      const key = [n.id, p.node].sort().join("|");
      if (seen.has(key) || !pos.has(p.node)) continue;
      seen.add(key);
      const [x1, y1] = pos.get(n.id);
      const [x2, y2] = pos.get(p.node);
      const age = p.lastHandshake ? ctx.now - p.lastHandshake : null;
      const cls = age === null ? "never" : age <= 180000 && p.bindingOk ? "up" : "old";
      lines.push(raw(`<line class="${cls}" x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}"><title>${esc2(n.name)} ↔ ${esc2(p.node)}: ${age === null ? "no handshake observed" : "handshake " + dur(age) + " ago"}${p.rttUs >= 0 ? ", RTT " + us(p.rttUs) : ", RTT not measured"}</title></line>`));
    }
  }
  const circles = all.map((n) => {
    const [x, y] = pos.get(n.id);
    return raw(`<g class="n ${n.cp ? "cp" : ""} ${n.stale ? "stale" : ""}"><circle cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="17"/><text x="${x.toFixed(1)}" y="${(y + 31).toFixed(1)}">${esc2(n.name)}</text><text class="ip" x="${x.toFixed(1)}" y="${(y + 43).toFixed(1)}">${esc2(n.ip || "")}</text></g>`);
  });
  return html`<svg class="topo" viewBox="0 0 ${W} ${H}" role="img" aria-label="Mesh topology from host-reported WireGuard handshakes">${lines}${circles}</svg>
    <div class="legend"><span><i style="border-color:rgba(62,207,142,.7)"></i>handshake ≤ 3 min and binding verified</span><span><i style="border-color:rgba(242,169,59,.7);border-top-style:dashed"></i>older handshake or unverified binding</span><span><i style="border-color:rgba(242,95,110,.7);border-top-style:dotted"></i>no handshake observed</span><span>dashed circle = host evidence stale · edges are drawn from host reports only</span></div>`;
}

const esc2 = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

// ------------------------------------------------------------ edge

export const edge = {
  title: "Edge",
  sub: "The routing table the control plane publishes, next to what each edge host reports: endpoint states from its own probes over the mesh, plus certificate states.",
  body(v, ctx) {
    const svc = v.edge?.services || [];
    const edges = v.edge?.edges || [];
    return html`
      ${panel("Services (desired routing)", table([{ h: "Host" }, { h: "App" }, { h: "TLS" }, { h: "Health" }, { h: "Endpoints" }],
        svc.map((s) => html`<tr><td class="mono">${s.host}</td><td><a href="#/runtime/${encodeURIComponent(s.app)}">${s.app}</a></td><td>${s.tls}${s.issuer ? " · " + s.issuer : ""}</td><td class="mono small">${s.healthUrl || "—"}</td>
          <td class="small">${(s.endpoints || []).map((e) => html`<div>${e.assignment} @ ${hostLink(ctx, e.node)} <span class="mono dim">${e.meshIp}:${e.port}</span> ${gen(e.generation)} ${e.draining ? pill("DRAINING") : ""} ${pill((e.observed || "unknown").toUpperCase(), "last signed observation")}</div>`)}</td></tr>`),
        { empty: "no ingress declared" }), html`${basis("CONFIGURED")}`)}
      ${edges.length ? edges.map((e) => edgeHost(e, ctx)) : emptyPanel("Edge hosts", "no host has the edge role")}`;
  },
};

function edgeHost(e, ctx) {
  const o = e.obs;
  if (!o) return panel(e.name, html`<div class="empty">no edge observation from this host</div>`);
  return html`<section class="panel"><header><h2>${e.name} · ${o.httpAddr}${o.httpsAddr ? " · " + o.httpsAddr : ""}</h2><span>${pill(e.freshness)} <span class="dim small">${o.requests} requests · ${o.errors} errors ${basis("OBSERVED")}</span></span></header>
    ${e.freshness !== "FRESH" ? html`<div class="banner warn"><b>STALE</b><span>These states are as last reported by the edge.</span></div>` : ""}
    ${table([{ h: "Route" }, { h: "Endpoint" }, { h: "State" }, { h: "Reason" }, { h: "In flight", num: true }, { h: "Served", num: true }, { h: "Failures", num: true }, { h: "Latency" }, { h: "Since" }],
      (o.routes || []).flatMap((r) => (r.endpoints || []).map((p, i) => html`<tr><td class="mono">${i === 0 ? r.host : ""}</td><td>${p.assignment} @ ${hostLink(ctx, p.node)}</td><td>${pill(p.state?.toUpperCase())}</td>
        <td class="small">${p.reason}</td><td class="num mono">${p.inFlight}</td><td class="num mono">${p.served}</td><td class="num mono">${p.failures}</td>
        <td class="small">${p.latencyUs > 0 ? us(p.latencyUs) : html`<span class="unk-txt">NOT MEASURED</span>`}</td><td class="small">${ago(p.since, ctx.now) || "—"}</td></tr>`)),
      { empty: "no routes" })}
    <div style="margin-top:12px">${table([{ h: "Certificate" }, { h: "State" }, { h: "Issuer" }, { h: "Valid" }, { h: "Detail" }],
      (o.certs || []).map((c) => html`<tr><td class="mono">${(c.names || [c.host]).join(", ")}</td><td>${pill(c.state)}</td><td class="small">${c.issuer || "—"}</td>
        <td class="small nowrap">${c.notAfter ? html`until ${when(c.notAfter)}` : "—"}</td><td class="small">${c.detail}<div class="hash">${c.fingerprint || ""}</div></td></tr>`),
      { empty: "no certificates" })}</div></section>`;
}

// ------------------------------------------------------------ policy

const CODES = [
  ["LEDGER_CORRUPT", "host ledger fails verification", true], ["UNTRUSTED_PLANE", "bundle not anchored in the pinned root", true],
  ["ASSIGNMENT_SIGNATURE", "assignment not signed by a roster member", true], ["WRONG_HOST", "assignment names another host", false],
  ["STALE_GENERATION", "older generation than one already admitted", false], ["CLOCK_SKEW", "bundle timestamp beyond the skew limit", true],
  ["CAPABILITY", "capability chain does not authorize this admission", true], ["HOST_REVOKED", "host approval revoked", true],
  ["CONTROL_PLANE_FROZEN", "operator froze the control plane", true], ["CONTROL_PLANE_STALE", "bundle older than the freshness window", true],
  ["POLICY_TIER", "trust tier not accepted", false], ["POLICY_RUNTIME", "runtime not allowed", false], ["POLICY_DIGEST", "image not digest-pinned", false],
  ["POLICY_ARTIFACT_SIGNATURE", "artifact not attested by a trusted publisher", false], ["POLICY_FEDERATED", "federated work not accepted", false],
  ["POLICY_WORKLOAD_CAP", "workload cap", false], ["POLICY_CPU_CAP", "CPU cap", false], ["POLICY_MEMORY_CAP", "memory cap", false],
];

export const policy = {
  title: "Policy",
  sub: "Each host's own sovereign policy, as the host signed it, and every admission it refused. Refusals come from the host, not from the control plane.",
  body(v, ctx) {
    const nodes = v.nodes || [];
    const refusals = [];
    for (const a of v.apps || []) for (const r of a.rows || []) if (r.admitted === "REFUSED" || r.admitted === "HELD") refusals.push(r);
    const fields = [
      ["Accepted tiers", (p) => (p.acceptTiers || []).join(", ")], ["Runtimes", (p) => (p.allowRuntimes || []).join(", ")],
      ["Max workloads", (p) => p.maxWorkloads], ["Max CPU", (p) => cpu(p.maxCpuMilli)], ["Max memory", (p) => bytes(p.maxMemBytes)],
      ["Digest-pinned", (p) => (p.denyImagesWithoutDigest ? "required" : "—")], ["Artifact signature", (p) => (p.requireImageSignature ? "required" : "—")],
      ["Federated", (p) => (p.allowFederated ? "accept" : "refuse")], ["Exec", (p) => (p.allowExec ? "allow" : "refuse")],
      ["Offline", (p) => p.offlineAdmission], ["Clock skew", (p) => `${p.maxClockSkewMs} ms`], ["Storage quota", (p) => bytes(p.storageQuotaBytes)],
    ];
    return html`
      ${panel("Host policies", table([{ h: "" }, ...nodes.map((n) => ({ h: n.name }))],
        fields.map(([label, fn]) => html`<tr><td class="dim nowrap">${label}</td>${nodes.map((n) => html`<td class="small">${fn(n.policy || {})}</td>`)}</tr>`),
        { empty: "no hosts" }), html`<span class="dim small">host-signed ${basis("OBSERVED", "the host's own signed statement of its policy")}</span>`)}
      ${panel("Refused and held admissions", table([{ h: "Replica" }, { h: "Host" }, { h: "Decision" }, { h: "Code" }, { h: "Reason" }],
        refusals.map((r) => html`<tr><td class="mono">${r.assignment} ${gen(r.desiredGen)}</td><td>${hostLink(ctx, r.node)}</td><td>${pill(r.admitted)}</td><td class="mono small">${r.code}</td><td class="small">${r.reason}</td></tr>`),
        { empty: "no host is refusing or holding any assignment" }))}
      <div class="grid g2">
        ${panel("Protocol rejections", rejections(v.rejections || [], ctx))}
        ${panel("Admission order (dh/v1 §9.3)", html`<p class="dim small" style="margin-top:0">Checks run in this order and the first failure decides. For codes marked hold, a host keeps running work it already admitted at the same generation and starts nothing new.</p>
          ${table([{ h: "Code" }, { h: "Meaning" }, { h: "Holds" }], CODES.map(([c, m, h]) => html`<tr><td class="mono small">${c}</td><td class="small">${m}</td><td>${h ? "hold" : ""}</td></tr>`))}`)}
      </div>`;
  },
};

function rejections(list, ctx) {
  if (!list.length) return html`<div class="empty">no replayed, forged or tampered messages rejected</div>`;
  return html`<ul class="timeline">${list.slice().reverse().slice(0, 30).map((r) => html`<li><span class="when">${ago(r.ts, ctx.now)?.replace(" ago", "")}</span><span class="mk bad"></span>
    <span class="what"><b>${r.kind}</b> from ${hostLink(ctx, r.node)} ${r.seq ? html`<span class="mono small">seq ${r.seq}${r.lastSeq ? ` (last ${r.lastSeq})` : ""}</span>` : ""}<br><span class="small">${r.reason}</span> ${hash(r.evidence)}</span></li>`)}</ul>`;
}

// ------------------------------------------------------------ audit

export const audit = {
  title: "Audit",
  sub: "The control-plane ledger. Every entry commits to the previous hash, and signed checkpoints make truncation detectable. The chain is verified on the serving member, and you can re-verify it on demand.",
  toolbar(ctx) {
    return html`<input class="in" id="audit-filter" placeholder="filter: action, resource, actor, detail" value="${ctx.ui.auditFilter || ""}" style="width:320px">
      <button class="btn sm" data-act="audit-verify" data-arg="">Verify now</button>
      <button class="btn sm" data-act="audit-load" data-arg="">Load full ledger</button>`;
  },
  body(v, ctx) {
    const a = v.audit || {};
    const ver = ctx.ui.auditVerify || a.verification || {};
    const all = ctx.ui.auditFull || a.entries || [];
    const q = (ctx.ui.auditFilter || "").toLowerCase();
    const entries = all.filter((e) => !q || `${e.action} ${e.resource} ${e.actor} ${e.detail} ${e.source}`.toLowerCase().includes(q)).slice().reverse();
    return html`
      <div class="grid g3">
        ${panel("Chain", kv([
          ["Head", html`<span class="mono">#${a.head}</span> ${hash(a.headHash)}`],
          ["Verification", ver.ok ? pill("VERIFIED") : pill("BROKEN")],
          ["Entries checked", fact(ver.entries, "OBSERVED")],
          ["Verified", html`${when(ver.verifiedAt)} by ${hostLink(ctx, ver.verifiedBy)}`],
          ver.break ? ["Break", html`<span style="color:var(--bad)">${ver.break.reason} at seq ${ver.break.seq}</span><div class="small dim">expected ${ver.break.expected}<br>actual ${ver.break.actual}</div>`] : null,
        ]))}
        ${panel("Checkpoints", kv([
          ["Signed checkpoints", fact(a.checkpoints, "OBSERVED")],
          ["Latest at seq", a.lastCheckpoint ? html`<span class="mono">#${a.lastCheckpoint}</span>` : unknown("no checkpoint yet")],
          ["Checkpoint check", ver.checkpointBreak ? html`<span style="color:var(--bad)">${ver.checkpointBreak.reason}</span>` : pill("VERIFIED")],
        ]))}
        ${panel("Showing", kv([
          ["Source", ctx.ui.auditFull ? `full ledger (${ctx.ui.auditFull.length} entries)` : `latest ${(a.entries || []).length} entries from the view`],
          ["Matching filter", `${entries.length}`],
          ["Verify offline", html`<span class="mono small">dh audit verify</span>`],
        ]))}
      </div>
      ${panel("Entries", table([{ h: "Seq", num: true }, { h: "Time" }, { h: "Source" }, { h: "Action" }, { h: "Resource" }, { h: "Gen" }, { h: "Detail" }, { h: "Hash" }],
        entries.slice(0, 400).map((e) => html`<tr class="click" data-audit="${e.seq}"><td class="num mono">${e.seq}</td><td class="small nowrap">${when(e.ts)}</td><td class="small">${e.source}</td>
          <td class="mono small nowrap">${e.action}</td><td class="small">${e.resource}</td><td class="mono small">${e.generation || ""}</td><td class="small">${e.detail}</td><td>${hash(e.hash)}</td></tr>`),
        { empty: q ? "no entries match the filter" : "no entries" }))}`;
  },
};

// ------------------------------------------------------------ diagnostics

export const diagnostics = {
  title: "Diagnostics",
  sub: "Negative facts are shown on purpose: what is not installed, not listening or not measured. This is where the system's knowledge ends.",
  body(v, ctx) {
    const groups = new Map();
    for (const d of v.diagnostics || []) {
      if (!groups.has(d.subject)) groups.set(d.subject, []);
      groups.get(d.subject).push(d);
    }
    const neg = (d) => /^NOT |^NONE|UNKNOWN|MISSING|FAIL|DISABLED|UNAVAILABLE|NO /i.test(d.value);
    return html`<div class="grid g2">${[...groups.entries()].map(([subject, items]) => panel(subject, html`<dl class="kv">${items.map((d) => html`
      <dt>${d.item}</dt><dd>${neg(d) ? html`<span class="unk-txt">${d.value}</span>` : html`<span>${d.value}</span>`}${basis(d.basis)} <span class="dim small">${d.detail}</span></dd>`)}</dl>`))}</div>
      ${!groups.size ? emptyPanel("Diagnostics", "no diagnostics reported") : ""}`;
  },
};

// ------------------------------------------------------------ capabilities

export const capabilities = {
  title: "Capabilities",
  sub: "Authority is a chain of signed, attenuable blocks anchored in the cluster root. This page decodes your session capability locally. The control plane verifies it on every request.",
  body(v, ctx) {
    const s = ctx.session;
    return html`
      ${panel("This session", s.blocks ? html`
        ${kv([
          ["Effective actions", html`${s.actions.map((a) => html`<span class="pill neutral">${a}</span> `)}`],
          ["Expires", s.expires ? html`${when(s.expires)} <span class="dim">(${ago(s.expires, ctx.now)})</span>` : "no expiry"],
          ["Chain", `${s.blocks.length} block(s)`],
          ["Root anchor", html`${s.blocks[0]?.pub === v.cluster?.root ? pill("CLUSTER ROOT") : pill("NOT THE CLUSTER ROOT")} <span class="mono small">${shortId(s.blocks[0]?.pub)}</span>`],
        ])}
        <div style="margin-top:12px">${table([{ h: "#" }, { h: "Signer key" }, { h: "Actions" }, { h: "Resources" }, { h: "Window" }, { h: "Next" }, { h: "Note" }],
          s.blocks.map((b, i) => { const c = b.payload?.caveats || {}; return html`<tr><td class="mono">${i}</td><td class="mono small">${shortId(b.pub)}</td><td class="small">${(c.actions || []).join(", ")}</td>
            <td class="small">${(c.resources || []).join(", ")}</td><td class="small nowrap">${c.notBefore ? when(c.notBefore) : "—"} → ${c.expires ? when(c.expires) : "∞"}</td>
            <td class="small">${b.payload?.next ? html`<span class="mono">${shortId(b.payload.next)}</span>` : "sealed"}</td><td class="small">${b.payload?.note}</td></tr>`; }))}</div>`
        : html`<div class="empty">session capability could not be decoded</div>`)}
      <section class="panel"><header><h2>Capability lab</h2></header>
        <div class="banner lab"><b>LAB</b><span>Lab tokens are signed by an ephemeral lab root that no host trusts. They cannot admit real work. Use them to see how attenuation narrows authority.</span></div>
        <div class="grid g2">
          <div>
            <label class="fld">Actions (comma separated)<input class="in" id="lab-actions" value="${ctx.ui.lab?.actions || "workload.*"}"></label>
            <label class="fld" style="margin-top:8px">Resources<input class="in" id="lab-resources" value="${ctx.ui.lab?.resources || "app/*"}"></label>
            <label class="fld" style="margin-top:8px">CPU max (millicores, 0 = none)<input class="in" id="lab-cpu" value="${ctx.ui.lab?.cpu || "0"}"></label>
            <div class="toolbar" style="margin-top:10px"><button class="btn sm primary" data-act="lab-mint">Mint lab token</button><button class="btn sm" data-act="lab-attenuate" ${ctx.ui.lab?.token ? "" : raw("disabled")}>Attenuate with these caveats</button></div>
            <hr style="border:0;border-top:1px solid var(--line);margin:12px 0">
            <label class="fld">Request action<input class="in" id="lab-req-action" value="${ctx.ui.lab?.reqAction || "workload.admit"}"></label>
            <label class="fld" style="margin-top:8px">Request resource<input class="in" id="lab-req-resource" value="${ctx.ui.lab?.reqResource || "app/web/r0"}"></label>
            <label class="fld" style="margin-top:8px">Request CPU (millicores)<input class="in" id="lab-req-cpu" value="${ctx.ui.lab?.reqCpu || "250"}"></label>
            <div class="toolbar" style="margin-top:10px"><button class="btn sm" data-act="lab-verify" ${ctx.ui.lab?.token ? "" : raw("disabled")}>Verify request</button></div>
          </div>
          <div>
            ${ctx.ui.lab?.token ? html`<div class="dim small">Current lab token · ${ctx.ui.lab.blocks} block(s)</div><pre class="code">${ctx.ui.lab.token}</pre>` : html`<div class="empty">mint a lab token to begin</div>`}
            ${ctx.ui.lab?.result ? html`<div style="margin-top:10px">${kv([
              ["Against lab root", ctx.ui.lab.result.lab.ok ? pill("AUTHORIZED") : html`${pill("REFUSED")} <span class="small">block ${ctx.ui.lab.result.lab.block}: ${ctx.ui.lab.result.lab.reason}</span>`],
              ["Against cluster root", ctx.ui.lab.result.againstClusterRoot.ok ? pill("AUTHORIZED") : html`${pill("REFUSED")} <span class="small">${ctx.ui.lab.result.againstClusterRoot.reason}</span>`],
            ])}</div>` : ""}
          </div>
        </div>
      </section>`;
  },
};

// ------------------------------------------------------------ federation

export const federation = {
  title: "Federation",
  sub: "Explicit, revocable trust between independent clusters. No global authority is involved. Each agreement is signed by the granting cluster's root, and hosts still apply their own allowFederated policy.",
  body(v, ctx) {
    const f = v.federation || {};
    const agr = (list, dir) => table([{ h: "Peer" }, { h: "Agreement" }, { h: "Limits" }, { h: "Valid" }, { h: "State" }, { h: "" }],
      (list || []).map((r) => html`<tr><td>${dir === "granted" ? r.a.grantee : r.a.grantor}<div class="mono small dim">${shortId(dir === "granted" ? r.a.granteeRoot : r.a.grantorRoot)}</div></td>
        <td>${hash(r.digest)}</td><td class="small">${r.a.maxReplicas} replicas · ${cpu(r.a.maxCpuMilli)} · ${bytes(r.a.maxMemBytes)}<div class="dim">tiers ${(r.a.tiers || []).join(", ")} · ${(r.a.runtimes || []).join(", ")}</div></td>
        <td class="small nowrap">${when(r.a.notBefore)} →<br>${when(r.a.expires)}</td><td>${r.revoked ? pill("REVOKED", "at " + when(r.revokedAt)) : r.a.expires && r.a.expires < ctx.now ? pill("EXPIRED") : pill("ACTIVE")}</td>
        <td>${dir === "granted" && !r.revoked ? actionBtn(ctx, "Revoke…", "fed-revoke", r.digest, "danger", "api.admin") : ""}</td></tr>`),
      { empty: dir === "granted" ? "this cluster has not granted capacity to any peer" : "no peer has granted this cluster capacity" });
    return html`<div class="grid g2">
      ${panel("Granted (peers may place work here)", agr(f.granted, "granted"))}
      ${panel("Held (this cluster may place work on peers)", agr(f.held, "held"))}
      ${panel("Inbound placements", table([{ h: "App" }, { h: "Peer" }, { h: "Remote app" }, { h: "Received" }, { h: "State" }],
        (f.inbound || []).map((p) => html`<tr><td><a href="#/runtime/${encodeURIComponent(p.app)}">${p.app}</a></td><td>${p.peer}</td><td>${p.remoteApp}</td><td class="small">${when(p.received)}</td><td>${p.withdrawn ? pill("WITHDRAWN") : pill("ACTIVE")}</td></tr>`),
        { empty: "no inbound placements" }))}
      ${panel("Outbound placements", table([{ h: "App" }, { h: "Peer" }, { h: "Replicas" }, { h: "Accepted" }, { h: "Last status" }],
        (f.outbound || []).map((p) => html`<tr><td>${p.app}</td><td>${p.peer}</td><td class="mono">${p.replicas}</td><td>${p.accepted ? pill("ACCEPTED") : pill("PENDING", p.message)}</td><td class="small">${p.statusAt ? html`${ago(p.statusAt, ctx.now)} ${basis("OBSERVED", "signed status from the peer")}` : unknown("no status from the peer yet")}</td></tr>`),
        { empty: "no outbound placements" }))}
    </div>`;
  },
};

// ------------------------------------------------------------ conformance

export const conformance = {
  title: "Conformance",
  sub: "Protocol conformance, chaos evidence and milestone state. The conformance run happens on the serving member each time you press Run. Chaos reports are signed by the runner that produced them.",
  toolbar() {
    return html`<button class="btn sm primary" data-act="conformance">Run dh/v1 vectors on the serving member</button>`;
  },
  body(v, ctx) {
    const c = ctx.ui.conformance;
    const rep = c?.report;
    return html`
      ${panel("dh/v1 conformance", c ? html`
        <div class="counters">
          ${counter("Vectors", c.vectors, "OBSERVED", `run on ${shortId(c.member)} · ${c.go}`)}
          ${counter("Pass", rep.pass, "OBSERVED", "", rep.pass === c.vectors ? "good" : "")}
          ${counter("Fail", rep.fail, "OBSERVED", "", rep.fail ? "bad" : "")}
          ${counter("Time", dur(rep.millis), "OBSERVED", when(rep.started))}
        </div>
        <div style="margin-top:12px">${table([{ h: "Operation" }, { h: "Pass", num: true }, { h: "Fail", num: true }, { h: "Result" }],
          (c.ops || []).filter((op) => rep.byOp?.[op]).map((op) => { const s = rep.byOp[op]; return html`<tr><td class="mono">${op}</td><td class="num mono">${s.Pass}</td><td class="num mono">${s.Fail}</td><td>${s.Fail ? pill("FAIL") : pill("PASS")}</td></tr>`; }))}</div>
        ${(rep.results || []).filter((r) => !r.pass).length ? html`<pre class="code">${rep.results.filter((r) => !r.pass).map((r) => `${r.id}\n  expected ${r.expected}\n  got      ${r.got}`).join("\n")}</pre>` : ""}
        <p class="dim small">${c.external}</p>` : html`<div class="empty">not run in this session. The result is only shown after it actually runs.</div>`)}
      ${panel("Chaos reports", table([{ h: "Scenario" }, { h: "Verdict" }, { h: "Received" }, { h: "Signer" }, { h: "Evidence" }],
        (v.chaos || []).slice().reverse().map((r) => html`<tr><td>${r.scenario}</td><td>${pill(r.verdict)}</td><td class="small">${when(r.received)}</td><td class="mono small">${shortId(r.signer)}</td><td>${hash(r.evidence)}</td></tr>`),
        { empty: "no signed chaos reports submitted — run `dh chaos run --submit`" }))}
      ${panel("Milestones", html`<div class="milestones">${(v.milestones || []).map((m) => html`<div class="ms"><div class="id2">${m.id}</div><div class="tt">${m.title}</div>${pill(m.state)}${basis(m.basis)}
        ${m.evidence?.length ? html`<ul>${m.evidence.map((e) => html`<li>${e}</li>`)}</ul>` : ""}${m.gaps?.length ? html`<ul class="gaps">${m.gaps.map((e) => html`<li>${e}</li>`)}</ul>` : ""}</div>`)}</div>`)}`;
  },
};

// ------------------------------------------------------------ settings

export const settings = {
  title: "Settings",
  sub: "These preferences stay in this browser tab. The console sends no telemetry and loads nothing from third parties.",
  body(v, ctx) {
    return html`<div class="grid g2">
      ${panel("Session", kv([
        ["Cluster", html`<b>${v.cluster?.name}</b>`],
        ["Root key", html`<span class="mono small">${v.cluster?.root}</span>`],
        ["Capability actions", ctx.session.actions.join(", ") || "none"],
        ["Expires", ctx.session.expires ? `${when(ctx.session.expires)} (${ago(ctx.session.expires, ctx.now)})` : "no expiry"],
        ["Token storage", "sessionStorage of this tab only; removed from the URL on load"],
        ["", html`<button class="btn sm danger" data-act="signout">Sign out</button>`],
      ]))}
      ${panel("Display", html`
        <label class="fld">Refresh interval
          <select class="in" id="set-interval">${[1000, 2000, 5000, 10000, 30000].map((ms) => html`<option value="${ms}" ${ctx.ui.interval === ms ? raw("selected") : ""}>${dur(ms)}</option>`)}</select></label>
        <label class="fld" style="margin-top:10px;display:flex;gap:8px;align-items:center"><input type="checkbox" id="set-reduced" ${ctx.ui.reduced ? raw("checked") : ""}> Reduce motion (always on when the OS asks for it)</label>
        <p class="dim small">Keyboard: <span class="kbd">⌘K</span>/<span class="kbd">Ctrl K</span> or <span class="kbd">/</span> opens the command palette. <span class="kbd">g</span> then a letter jumps between screens. <span class="kbd">r</span> refreshes, and <span class="kbd">?</span> lists every shortcut.</p>`)}
      ${panel("About", kv([
        ["Protocol", "dh/v1"],
        ["View served by", html`${hostLink(ctx, v.servedBy?.member)} (${v.servedBy?.state})`],
        ["View generated", `${when(v.generatedAt)} (control-plane clock)`],
        ["Telemetry", "none"],
      ]))}
    </div>`;
  },
};
