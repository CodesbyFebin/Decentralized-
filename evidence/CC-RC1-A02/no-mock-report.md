# No-mock production gate

Scanned 45 production file(s); exempt (tests, fixtures, explicit demo adapter): src/server/adapters/demo.ts.

| Verdict | Count |
| --- | --- |
| explicit demo-mode handling / label | 12 |
| explicit demo-mode check | 22 |
| timer (polling, abort, UI feedback) | 5 |
| comment | 1 |
| input placeholder text / prop | 20 |
| state comparison / label | 8 |

Production mock leakage: NONE

## All classified hits
- [explicit demo-mode handling / label] server.ts:13 — import { DemoPlatformAdapter } from './src/server/adapters/demo';
- [explicit demo-mode check] server.ts:76 — if (adapter.mode === 'demo') console.log('[command-centre] DEMO REPLAY: every value is SIMULATED and mutations are refused.');
- [timer (polling, abort, UI feedback)] src/components/common/ConnectDialog.tsx:44 — setTimeout(() => setCopied(null), 1500);
- [comment] src/components/common/ui.tsx:356 — /** Honest provenance marker for demo seed data (reality matrix: SIMULATED). */
- [explicit demo-mode handling / label] src/components/common/ui.tsx:356 — /** Honest provenance marker for demo seed data (reality matrix: SIMULATED). */
- [explicit demo-mode handling / label] src/components/common/ui.tsx:357 — export const DemoTag: React.FC<{ mode: 'demo' \| 'live' }> = ({ mode }) =>
- [explicit demo-mode check] src/components/common/ui.tsx:358 — mode === 'demo' ? (
- [explicit demo-mode handling / label] src/components/common/ui.tsx:360 — title="Values come from the demo platform store, not live node agents."
- [explicit demo-mode handling / label] src/components/common/ui.tsx:363 — DEMO DATA
- [input placeholder text / prop] src/components/common/ui.tsx:437 — placeholder: string;
- [input placeholder text / prop] src/components/common/ui.tsx:441 — }> = ({ search, onSearch, placeholder, children, onSort, sortLabel = 'Sort' }) => (
- [input placeholder text / prop] src/components/common/ui.tsx:450 — placeholder={placeholder}
- [input placeholder text / prop] src/components/common/ui.tsx:451 — className="bg-transparent outline-none text-[12px] text-slate-100 placeholder:text-slate-500 w-full"
- [explicit demo-mode check] src/components/layout/AppShell.tsx:36 — const signedIn = mode === 'demo' \|\| !!session?.authenticated;
- [explicit demo-mode check] src/components/layout/AppShell.tsx:58 — {mode === 'demo' && (
- [explicit demo-mode handling / label] src/components/layout/AppShell.tsx:60 — <b className="text-amber-300">DEMO REPLAY</b> — every value is SIMULATED from a recorded dev-cluster view. Mutations are refused. Set PLATFO
- [explicit demo-mode check] src/components/layout/CommandPalette.tsx:37 — const data = useResource<{ nodes: NodeRec[]; apps: AppRec[]; domains: DomainRec[] }>(open && (mode === 'demo' \|\| session?.authenticated) ? '
- [timer (polling, abort, UI feedback)] src/components/layout/CommandPalette.tsx:43 — setTimeout(() => input.current?.focus(), 0);
- [input placeholder text / prop] src/components/layout/CommandPalette.tsx:84 — placeholder="Search pages, hosts, apps, domains — or ask the Copilot"
- [input placeholder text / prop] src/components/layout/CommandPalette.tsx:85 — className="flex-1 bg-transparent outline-none text-[14px] text-white placeholder:text-slate-500"
- [explicit demo-mode check] src/components/layout/Sidebar.tsx:54 — const state = !backend ? 'checking' : backend.reachable ? (capabilities?.mode === 'demo' ? 'demo' : 'ok') : 'down';
- [explicit demo-mode check] src/components/layout/Sidebar.tsx:88 — <Hexagon className={`w-6 h-6 ${state === 'down' ? 'text-rose-400 fill-rose-400/20' : state === 'demo' ? 'text-amber-400 fill-amber-400/20' :
- [explicit demo-mode check] src/components/layout/Sidebar.tsx:94 — : state === 'demo'
- [explicit demo-mode check] src/components/layout/Sidebar.tsx:99 — <span className={`w-1.5 h-1.5 rounded-full ${state === 'down' ? 'bg-rose-400' : state === 'demo' ? 'bg-amber-400' : 'bg-emerald-400'}`} />
- [explicit demo-mode check] src/components/layout/Sidebar.tsx:100 — {state === 'checking' ? 'Checking' : state === 'down' ? 'Unreachable' : state === 'demo' ? 'Simulated' : 'Reachable'}
- [input placeholder text / prop] src/components/layout/SignIn.tsx:54 — placeholder="dhcap1.…"
- [explicit demo-mode check] src/components/layout/TopBar.tsx:28 — const actor = session?.actor ?? (mode === 'demo' ? 'demo viewer' : 'not signed in');
- [explicit demo-mode check] src/components/layout/TopBar.tsx:74 — <span className="text-[11px] text-slate-400">{signedIn ? roleOf(session!.actions) : mode === 'demo' ? 'Simulated' : 'Signed out'}</span>
- [input placeholder text / prop] src/components/pages/AppDetailView.tsx:146 — <input type="number" min={0} max={64} value={scaleTo} onChange={(e) => setScaleTo(e.target.value)} placeholder={String(a.desiredReplicas)} c
- [input placeholder text / prop] src/components/pages/AppDetailView.tsx:149 — <input value={confirmText} onChange={(e) => setConfirmText(e.target.value)} placeholder={`type ${a.name} to stop it`} className="rounded-lg 
- [input placeholder text / prop] src/components/pages/AppsView.tsx:49 — <TableToolbar search={search} onSearch={setSearch} placeholder="Search apps or domains...">
- [input placeholder text / prop] src/components/pages/CopilotView.tsx:87 — <input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder={`type ${a.confirmText} to confirm`} className="mt-2 w-full 
- [input placeholder text / prop] src/components/pages/CopilotView.tsx:208 — <input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Ask about hosts, apps, certificates, storage, the audit ledger 
- [explicit demo-mode check] src/components/pages/DashboardView.tsx:27 — const who = session?.actor?.replace(/^operator:?/, '') \|\| (mode === 'demo' ? 'demo viewer' : 'operator');
- [state comparison / label] src/components/pages/DashboardView.tsx:200 — <li className="flex items-center gap-2.5"><IconTile tone={d.verification.state === 'VERIFIED' ? 'emerald' : 'rose'} size="sm"><FileCheck2 cl
- [explicit demo-mode check] src/components/pages/DeployNewView.tsx:141 — {!allowed && <Note>{mode === 'demo' ? 'The demo replay is read-only.' : 'Applying needs a capability with api.write. You can still compose a
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:147 — <label className={label}>Application name<input className={input} value={f.name} onChange={(e) => set('name', e.target.value)} placeholder="
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:167 — <label className={label}>Image (ref@sha256:…)<input className={`${input} font-mono`} value={f.image} onChange={(e) => set('image', e.target.
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:184 — <label className={label}>Port name<input className={input} value={f.port} onChange={(e) => set('port', e.target.value)} placeholder="http (e
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:188 — <label className={label}>Ingress host<input className={input} value={f.ingressHost} onChange={(e) => set('ingressHost', e.target.value)} pla
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:196 — <input className={`${input} font-mono`} value={e.k} placeholder="NAME" onChange={(x) => set('env', f.env.map((y, j) => (j === i ? { ...y, k:
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:197 — <input className={input} value={e.v} placeholder="value" onChange={(x) => set('env', f.env.map((y, j) => (j === i ? { ...y, v: x.target.valu
- [input placeholder text / prop] src/components/pages/DeployNewView.tsx:209 — <textarea value={yaml} onChange={(e) => setYaml(e.target.value)} rows={22} spellCheck={false} className="w-full rounded-xl bg-[#020814] bord
- [state comparison / label] src/components/pages/EvidenceView.tsx:70 — <IconTile tone={v.state === 'VERIFIED' ? 'emerald' : v.state === 'INVALID' ? 'rose' : 'slate'} size="lg">{v.state === 'INVALID' ? <ShieldX c
- [state comparison / label] src/components/pages/EvidenceView.tsx:89 — <div className="flex items-center gap-2"><span className="font-mono text-[11px] text-slate-500">{m.id}</span><span className="text-[13px] te
- [input placeholder text / prop] src/components/pages/NodesView.tsx:235 — <TableToolbar search={search} onSearch={setSearch} placeholder="Search hosts by name, region, or ID..." onSort={() => setSortDesc(!sortDesc)
- [explicit demo-mode check] src/components/pages/SettingsView.tsx:26 — <div className="flex justify-between"><span className="text-slate-400">Adapter</span><span className="text-slate-100">{mode === 'demo' ? 'de
- [explicit demo-mode check] src/components/pages/TeamView.tsx:33 — <p className="mt-2 text-[12.5px] text-slate-400">{mode === 'demo' ? 'Demo replay: no session is needed and nothing can be changed.' : 'Not s
- [explicit demo-mode check] src/lib/session.tsx:90 — if (mode === 'demo') return action === 'api.read';
- [timer (polling, abort, UI feedback)] src/lib/useResource.ts:63 — if (alive && pollMs > 0 && document.visibilityState !== 'hidden') timer = setTimeout(run, pollMs);
- [timer (polling, abort, UI feedback)] src/lib/useResource.ts:64 — else if (alive && pollMs > 0) timer = setTimeout(run, pollMs * 3);
- [timer (polling, abort, UI feedback)] src/server/adapters/controlPlane.ts:65 — const timer = setTimeout(() => ac.abort(new Error('timeout')), budget);
- [state comparison / label] src/server/adapters/controlPlane.ts:191 — state: v.ok ? 'VERIFIED' : 'INVALID',
- [explicit demo-mode check] src/server/bff.ts:173 — if (config.adapter === 'demo') return;
- [explicit demo-mode handling / label] src/server/config.ts:3 — * demo replay unless ALLOW_DEMO_IN_PRODUCTION=1, and there is no automatic
- [explicit demo-mode handling / label] src/server/config.ts:24 — throw new ConfigError('PLATFORM_ADAPTER is not set. Use PLATFORM_ADAPTER=controlplane with DH_CONTROL_URL, or PLATFORM_ADAPTER=demo for a la
- [explicit demo-mode handling / label] src/server/config.ts:26 — if (raw !== 'controlplane' && raw !== 'demo') throw new ConfigError(`PLATFORM_ADAPTER must be "controlplane" or "demo", got "${raw}".`);
- [explicit demo-mode check] src/server/config.ts:27 — if (raw === 'demo' && production && env.ALLOW_DEMO_IN_PRODUCTION !== '1') {
- [explicit demo-mode handling / label] src/server/config.ts:28 — throw new ConfigError('Refusing to start: PLATFORM_ADAPTER=demo in production. Set ALLOW_DEMO_IN_PRODUCTION=1 only for a labelled showcase.'
- [explicit demo-mode check] src/server/reality/capabilities.ts:19 — const live: TruthState = mode === 'demo' ? 'SIMULATED' : 'LIVE';
- [explicit demo-mode check] src/server/reality/capabilities.ts:20 — const src = mode === 'demo' ? 'demo-replay' : 'control-plane';
- [explicit demo-mode check] src/server/reality/capabilities.ts:30 — nodeOperations: ok ? on('approve, drain, undrain and revoke go through the control plane', mode === 'demo' ? 'SIMULATED' : 'LIVE') : down(in
- [explicit demo-mode handling / label] src/server/reality/mapView.ts:38 — /** LIVE for a real control plane, SIMULATED for the demo replay. */
- [state comparison / label] src/server/reality/mapView.ts:369 — state: !ver ? 'UNKNOWN' : ver.ok ? 'VERIFIED' : 'INVALID',
- [state comparison / label] src/server/reality/mapView.ts:479 — result: ev.verification.state === 'VERIFIED' ? 'PASS' : ev.verification.state === 'INVALID' ? 'FAIL' : 'UNKNOWN',
- [state comparison / label] src/server/reality/mapView.ts:480 — detail: ev.verification.state === 'VERIFIED' ? `${ev.verification.entries} entries hash-chained and verified by ${ev.verification.verifiedBy
- [explicit demo-mode handling / label] src/types/reality.ts:30 — export type AdapterMode = 'controlplane' \| 'demo';
- [state comparison / label] src/types/reality.ts:262 — state: 'VERIFIED' \| 'INVALID' \| 'UNKNOWN';
