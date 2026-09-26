import React, { useState } from 'react';
import { LogOut, AlertTriangle } from 'lucide-react';
import type { CapabilityKey, Session } from '../../types/reality';
import { useSession } from '../../lib/session';
import { useResource } from '../../lib/useResource';
import { Glass, PanelHeader, GhostButton } from '../common/ui';
import { Gate, TruthTag, fmtTime } from '../common/states';

interface Metrics {
  backend: { count: number; errors: number; avgMs: number; maxMs: number };
  routes: Record<string, { count: number; errors: number; avgMs: number; maxMs: number }>;
}

interface ServerSettings {
  adapter: 'controlplane' | 'demo';
  production: boolean;
  controlEndpoints: string[];
  timeoutMs: number;
  regionLocations: string[] | null;
  secureCookies: boolean;
  copilotModel: string | null;
  evidenceDir: string | null;
  cliConfigured: boolean;
  session: Session;
}

interface Setting {
  name: string;
  value: React.ReactNode;
  source: string;
  description: string;
  impact: string;
}

const SECTIONS = ['General', 'Organization', 'Control Plane', 'Node Defaults', 'Deployment Defaults', 'Networking', 'Security', 'Evidence', 'Notifications', 'Integrations', 'Advanced'] as const;
type Section = (typeof SECTIONS)[number];

const SettingTable: React.FC<{ rows: Setting[] }> = ({ rows }) => (
  <div className="overflow-x-auto">
    <table className="dh-table w-full min-w-[720px]">
      <thead><tr><th className="w-[22%]">Setting</th><th className="w-[22%]">Current value</th><th>Source</th><th>What it affects</th></tr></thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.name}>
            <td><div className="text-slate-100">{r.name}</div><div className="text-[11px] text-slate-500 whitespace-normal">{r.description}</div></td>
            <td className="text-slate-200 whitespace-normal break-all">{r.value}</td>
            <td className="font-mono text-[11px] text-slate-400 whitespace-normal">{r.source}</td>
            <td className="text-slate-400 text-[12px] whitespace-normal">{r.impact}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const None: React.FC<{ children: React.ReactNode }> = ({ children }) => <p className="text-[12.5px] text-slate-400">{children}</p>;

export default function SettingsView() {
  const { capabilities, mode, session, signOut } = useSession();
  const res = useResource<ServerSettings>('/settings');
  const metrics = useResource<Metrics>('/bff/metrics', { pollMs: 15_000 });
  const [section, setSection] = useState<Section>('General');

  return (
    <div className="space-y-4 pt-2 max-w-6xl">
      <div>
        <h1 className="text-2xl font-bold text-slate-100 tracking-tight">Settings</h1>
        <p className="text-[13px] text-slate-400">Where each setting lives and what it changes. The console is configured with environment variables on its server and the platform with the dh CLI and each host's own policy; nothing here is edited from the browser.</p>
      </div>
      <div className="grid md:grid-cols-[200px_minmax(0,1fr)] gap-4">
        <nav aria-label="Settings sections" className="md:sticky md:top-2 self-start">
          <ul className="flex md:flex-col gap-1 overflow-x-auto">
            {SECTIONS.map((s) => (
              <li key={s}>
                <button
                  type="button"
                  onClick={() => setSection(s)}
                  aria-current={section === s ? 'page' : undefined}
                  className={`w-full text-left whitespace-nowrap px-3 py-1.5 rounded-lg text-[13px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-cyan-400 ${section === s ? 'bg-white/[0.06] text-slate-100' : 'text-slate-400 hover:text-slate-200'}`}
                >
                  {s}
                </button>
              </li>
            ))}
          </ul>
        </nav>
        <Glass className="p-4 space-y-3">
          <Gate res={res}>
            {(c) => {
              const s = c.session;
              const tables: Record<Section, React.ReactNode> = {
                General: (
                  <SettingTable
                    rows={[
                      { name: 'Data source', value: mode === 'demo' ? 'demo replay (SIMULATED)' : 'control plane', source: 'PLATFORM_ADAPTER', description: 'Where every value in the console comes from.', impact: 'demo is refused in production unless ALLOW_DEMO_IN_PRODUCTION=1; there is no fallback between the two.' },
                      { name: 'Production mode', value: c.production ? 'on' : 'off', source: 'NODE_ENV', description: 'Production hardening of the console server.', impact: 'enables the content security policy and secure cookies by default.' },
                      { name: 'Signed in as', value: s.actor ?? 'not signed in', source: 'your capability (verified by the control plane)', description: 'The actor recorded in the audit ledger for your changes.', impact: `actions: ${s.actions.join(', ') || 'none'}${s.expiresAt ? ` · expires ${fmtTime(s.expiresAt)}` : ''}` }
                    ]}
                  />
                ),
                Organization: <None>There is no organization or team directory: operators are identified by capabilities signed by the cluster root. See Team.</None>,
                'Control Plane': (
                  <SettingTable
                    rows={[
                      { name: 'Members', value: c.controlEndpoints.join(', ') || '—', source: 'DH_CONTROL_URL', description: 'Control-plane API endpoints the console calls, in failover order.', impact: 'if every member is unreachable the console shows CONTROL PLANE UNREACHABLE.' },
                      { name: 'Request timeout', value: `${c.timeoutMs} ms`, source: 'DH_CONTROL_TIMEOUT_MS', description: 'Deadline shared across failover attempts for one request.', impact: 'how quickly the console reports an unreachable control plane.' },
                      { name: 'Reachability', value: capabilities?.backend.reachable ? 'reachable' : 'unreachable', source: 'health probe', description: capabilities?.backend.detail ?? '', impact: `checked ${fmtTime(capabilities?.backend.checkedAt)}; leader ${capabilities?.backend.leader?.slice(0, 16) ?? 'unknown'}` }
                    ]}
                  />
                ),
                'Node Defaults': <None>There are no cluster-wide node defaults. Each host sets its own name, region, declared capacity (--cpu, --mem) and admission policy (policy.yaml) when dh-noded starts; the control plane can read that policy but not change it.</None>,
                'Deployment Defaults': <None>There are no deployment defaults. Every value comes from the dh/v1 manifest you apply; images must be pinned by digest.</None>,
                Networking: (
                  <SettingTable
                    rows={[
                      { name: 'Region map positions', value: c.regionLocations ? c.regionLocations.join(', ') : 'not configured', source: 'DH_REGION_LOCATIONS', description: 'Operator-supplied coordinates per region. Hosts do not report locations.', impact: 'without it the globe shows no hosts and a list is shown instead. Positions are labelled CONFIGURED.' },
                      { name: 'Mesh', value: 'WireGuard per host', source: 'dh-noded --mesh / --mesh-advertise', description: 'Set on each host.', impact: 'private traffic between replicas and to the edge.' }
                    ]}
                  />
                ),
                Security: (
                  <SettingTable
                    rows={[
                      { name: 'Session cookie', value: c.secureCookies ? 'Secure, HttpOnly, SameSite=Strict' : 'HttpOnly, SameSite=Strict (not Secure)', source: 'COOKIE_SECURE / NODE_ENV', description: 'How your capability is kept in the browser.', impact: 'without Secure the cookie can travel over plain HTTP; use HTTPS in front of the console.' },
                      { name: 'Mutations', value: 'require the X-DH-Console header', source: 'console server', description: 'Protection against cross-site requests.', impact: 'requests from other sites are refused before reaching the control plane.' },
                      { name: 'Authorization', value: 'enforced by the control plane', source: 'capability actions (api.read, api.write, api.admin)', description: 'The console hides controls you cannot use, but never relies on that.', impact: 'a read-only capability gets 403 from the control plane for any change.' }
                    ]}
                  />
                ),
                Evidence: (
                  <SettingTable
                    rows={[
                      { name: 'Validation records', value: c.evidenceDir ?? 'not configured', source: 'DH_EVIDENCE_DIR', description: 'Directory of records sealed with dh evidence seal.', impact: 'without it the Evidence page shows the audit ledger only.' },
                      { name: 'Verifier', value: c.cliConfigured ? 'dh evidence verify' : 'not configured', source: 'DH_CLI', description: 'The dh binary used to verify records on request.', impact: 'without it records are listed but cannot be verified from the console.' }
                    ]}
                  />
                ),
                Notifications: <None>Notifications are not implemented. Attention items are shown on the Dashboard when you look.</None>,
                Integrations: (
                  <SettingTable
                    rows={[
                      { name: 'Copilot model', value: c.copilotModel ?? 'none (deterministic retrieval)', source: 'COPILOT_MODEL + GEMINI_API_KEY', description: 'Optional model for rephrasing Copilot answers.', impact: 'answers are grounded in the same data either way; the key never reaches the browser.' },
                      { name: 'External DePIN networks', value: <TruthTag state="UNAVAILABLE" />, source: '—', description: 'No adapter exists.', impact: 'nothing is installed on hosts.' }
                    ]}
                  />
                ),
                Advanced: (
                  <>
                    <div className="overflow-x-auto">
                      <table className="dh-table w-full min-w-[720px]">
                        <thead><tr><th>Capability</th><th>State</th><th>Source</th><th>Detail</th></tr></thead>
                        <tbody>
                          {capabilities &&
                            (Object.keys(capabilities.items) as CapabilityKey[]).map((k) => (
                              <tr key={k}>
                                <td className="text-slate-100 font-mono text-[12px]">{k}</td>
                                <td><TruthTag state={capabilities.items[k].state} /></td>
                                <td className="text-slate-400">{capabilities.items[k].source}</td>
                                <td className="text-slate-300 whitespace-normal">{capabilities.items[k].detail}</td>
                              </tr>
                            ))}
                        </tbody>
                      </table>
                    </div>
                    <Gate res={metrics}>
                      {(m) => (
                        <div className="mt-2 text-[12px] text-slate-400">
                          Console server counters since start: {m.backend.count} control-plane calls · {m.backend.errors} errors · avg {m.backend.avgMs} ms · max {m.backend.maxMs} ms. Nothing is sent anywhere.
                        </div>
                      )}
                    </Gate>
                  </>
                )
              };
              return (
                <>
                  <PanelHeader title={section} />
                  {tables[section]}
                </>
              );
            }}
          </Gate>
        </Glass>
      </div>

      <Glass className="p-4 border border-rose-400/20">
        <PanelHeader icon={<AlertTriangle className="w-4 h-4 text-rose-300" />} title="Danger zone" subtitle="Each of these is done with the dh CLI on an operator machine that holds the keys; the console cannot do them." />
        <ul className="mt-3 space-y-2 text-[12.5px]">
          <li className="flex flex-wrap gap-2"><span className="text-slate-200 w-44">Rotate the cluster root</span><code className="text-cyan-200">dh cp rotate-root</code><span className="text-slate-400">old root signs the rotation; hosts follow it</span></li>
          <li className="flex flex-wrap gap-2"><span className="text-slate-200 w-44">Rotate a host key</span><code className="text-cyan-200">dh-noded rotate-key</code><span className="text-slate-400">on the host; emergency: <code>dh node revoke-key HOST --pub KEY</code></span></li>
          <li className="flex flex-wrap gap-2"><span className="text-slate-200 w-44">Revoke sessions</span><span className="text-slate-400">capabilities cannot be revoked individually yet; they expire (dh console --ttl). Rotating the root invalidates all of them.</span></li>
          <li className="flex flex-wrap gap-2"><span className="text-slate-200 w-44">Freeze the cluster</span><code className="text-cyan-200">dh freeze</code><span className="text-slate-400">hosts hold admitted work and refuse new work</span></li>
        </ul>
        {session?.authenticated && (
          <GhostButton onClick={signOut} className="mt-4 !border-rose-400/40 !text-rose-200">
            <LogOut className="w-4 h-4" /> Sign out of this browser
          </GhostButton>
        )}
      </Glass>
    </div>
  );
}
