import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Plus, FileCode2, Upload, Github, Container, Sparkles, Home, Globe2, Users, Box, Network, Check, Server, Layers, CheckCircle2, Clock, Package, Cuboid, Layers3, ShieldCheck } from 'lucide-react';
import type { ArtifactRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import type { DeploymentProgress } from './AppDetailView';
import { SurfaceLayout, SurfaceHero } from '../common/CommandSurface';
import { Glass, KpiTile, RailPanel, RailItem, PanelHeader, IconTile, StatusPill, Grad, PrimaryButton, GhostButton, TONE, Tone } from '../common/ui';
import { Gate, Empty, shortDigest, since, fmtBytes, TruthTag } from '../common/states';

interface Payload {
  deployments: DeploymentProgress[];
  artifacts: ArtifactRec[];
}

export default function DeployView() {
  const res = useResource<Payload>('/deployments');
  const { capabilities } = useSession();
  const navigate = useNavigate();
  const caps = capabilities?.items;
  const latestPerApp = new Map<string, DeploymentProgress>();
  for (const d of res.data?.deployments ?? []) if (!latestPerApp.has(d.app)) latestPerApp.set(d.app, d);
  const latest = res.data?.deployments[0];
  const edgeLive = caps?.edgeTraffic.state === 'LIVE' || caps?.edgeTraffic.state === 'SIMULATED';

  const strategies: { id: string; title: string; desc: string; icon: React.ReactNode; tone: Tone; accent2: string; points: string[]; state: 'LIVE' | 'CONFIGURED' | 'UNAVAILABLE' | 'PLANNED' }[] = [
    { id: 'self', title: 'Self Host', desc: 'Replicas on hosts enrolled under your root.', icon: <Home className="w-5 h-5" />, tone: 'cyan', accent2: '#248BFF', points: ['Host-side admission', 'Digest-pinned artifacts', 'Replicated volumes', 'WireGuard mesh'], state: 'LIVE' },
    { id: 'edge', title: 'Edge Deploy', desc: 'Route a host name through edge hosts.', icon: <Globe2 className="w-5 h-5" />, tone: 'blue', accent2: '#5965FF', points: ['spec.ingress', 'Health-gated routing', 'ACME or local-CA TLS', 'Draining & ejection'], state: edgeLive ? 'LIVE' : 'UNAVAILABLE' },
    { id: 'federation', title: 'Federated', desc: 'Delegate replicas to a peer cluster that granted you capacity.', icon: <Users className="w-5 h-5" />, tone: 'violet', accent2: '#C026D3', points: ['Root-signed agreements', 'placement.federation', 'Grantor re-signs', 'Revocable'], state: 'CONFIGURED' },
    { id: 'depin', title: 'DePIN Deploy', desc: 'External decentralized compute networks.', icon: <Box className="w-5 h-5" />, tone: 'slate', accent2: '#475569', points: ['No adapter implemented'], state: 'UNAVAILABLE' },
    { id: 'marketplace', title: 'Marketplace', desc: 'Leases with independent providers.', icon: <Network className="w-5 h-5" />, tone: 'slate', accent2: '#475569', points: ['Offers, bids and leases', 'Usage metering', 'Settlement'], state: 'PLANNED' }
  ];

  const rail = (
    <>
      <RailPanel title="New Deployment" subtitle="Everything ends as a dh/v1 manifest applied to the control plane.">
        <div className="space-y-1.5">
          <RailItem icon={<FileCode2 className="w-5 h-5" />} tone="cyan" title="Compose manifest" subtitle="Pick an artifact, replicas, ingress" onClick={() => navigate('/deploy/new')} />
          <RailItem icon={<Upload className="w-5 h-5" />} tone="blue" title="Paste / upload YAML" subtitle="An existing dh/v1 manifest" onClick={() => navigate('/deploy/new?mode=yaml')} />
          <RailItem icon={<Container className="w-5 h-5" />} tone="blue" title="Deploy container" subtitle="docker runtime, ref@sha256 digest" onClick={() => navigate('/deploy/new?runtime=docker')} />
          <RailItem icon={<Github className="w-5 h-5" />} tone="slate" title="Import from Git" subtitle="No build service yet" trailing={<TruthTag state="PLANNED" />} />
          <RailItem icon={<Sparkles className="w-5 h-5" />} tone="violet" title="Plan with Copilot" subtitle="Scale / drain proposals today" onClick={() => navigate('/copilot')} />
        </div>
      </RailPanel>
      <RailPanel title="Artifacts" subtitle="Content-addressed executables in the cluster CAS." action={<Package className="w-4 h-4 text-slate-400" />}>
        {res.data?.artifacts.length ? (
          <ul className="space-y-2">
            {res.data.artifacts.map((a) => (
              <li key={a.digest} className="text-[12px]">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-slate-100">{a.name}</span>
                  <StatusPill status={a.attested ? 'ATTESTED' : 'UNSIGNED'} tone={a.attested ? 'emerald' : 'amber'} dot={false} />
                </div>
                <div className="font-mono text-[10.5px] text-slate-500 break-all">{a.digest}</div>
                <div className="text-[10.5px] text-slate-500">{fmtBytes(a.bytes)} · {a.chunks} chunks · uploaded {since(a.uploaded)}</div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-[12px] text-slate-400">No artifacts. Push one with <code className="text-cyan-200">dh artifact push FILE --name NAME --sign</code>.</p>
        )}
      </RailPanel>
      <RailPanel title="Developer Experience">
        <ul className="space-y-1.5 text-[12px]">
          {[
            ['Rollback to a generation', 'LIVE', 're-apply an earlier manifest'],
            ['Scale / delete', 'LIVE', 'from the application page'],
            ['Git push auto-deploy', 'PLANNED', 'no build service'],
            ['Preview URLs per branch', 'PLANNED', 'no build service']
          ].map(([t, s, d]) => (
            <li key={t} className="flex items-center justify-between gap-2">
              <span className="text-slate-300">{t}<span className="block text-[10.5px] text-slate-500">{d}</span></span>
              <TruthTag state={s as 'LIVE' | 'PLANNED'} />
            </li>
          ))}
        </ul>
      </RailPanel>
    </>
  );

  const pipeline = (
    <div className="hidden xl:flex absolute right-0 top-[-6px] z-10 items-start gap-3 pointer-events-auto select-none origin-top-right scale-[0.72] min-[1760px]:scale-90 min-[1960px]:scale-100">
      {[
        { label: 'Artifact', icon: <Package className="w-7 h-7" />, detail: ['content-addressed', 'b3 digest'] },
        { label: 'Attest', icon: <ShieldCheck className="w-7 h-7" />, detail: ['root-signed', 'attestation'] },
        { label: 'Schedule', icon: <Cuboid className="w-7 h-7" />, detail: ['signed', 'assignments'] },
        { label: 'Admit', icon: <Layers3 className="w-7 h-7" />, detail: ['host policy', 'checks'] },
        { label: 'Run', icon: <Layers className="w-7 h-7" />, detail: ['observed +', 'health-gated'] }
      ].map((s, i) => (
        <div key={s.label} className="flex items-start gap-3 pt-8">
          {i > 0 && (
            <svg width="26" height="12" className="mt-6 shrink-0" aria-hidden="true">
              <line x1="0" y1="6" x2="20" y2="6" stroke={TONE.cyan} strokeWidth="2" className={res.stale ? '' : 'dh-flow'} />
              <path d="M18 1 L25 6 L18 11" fill="none" stroke={TONE.cyan} strokeWidth="2" />
            </svg>
          )}
          <div className="flex flex-col items-center w-[78px]">
            <span className="text-[12px] font-semibold text-slate-100 mb-1.5">{s.label}</span>
            <IconTile tone="cyan" size="lg" className="!w-14 !h-14 !rounded-2xl">{s.icon}</IconTile>
            <div className="mt-2 text-center text-[10.5px] text-slate-300 leading-tight">{s.detail[0]}<br />{s.detail[1]}</div>
          </div>
        </div>
      ))}
    </div>
  );

  return (
    <SurfaceLayout rail={rail}>
      <SurfaceHero
        eyebrow={
          <div className="inline-flex items-center gap-2 text-[12px] font-semibold tracking-[0.18em] text-slate-200">
            <span className="text-cyan-300">›</span> UNIVERSAL DEPLOY
            {res.provenance && <TruthTag state={res.stale ? 'UNKNOWN' : res.provenance.state} />}
          </div>
        }
        title={
          <span className="xl:text-[40px] min-[1760px]:text-[44px]">
            Deploy <Grad from="#38BDF8" to="#818CF8">anywhere.</Grad>
            <br />
            Without <Grad from="#60A5FA" to="#C084FC">giving up control.</Grad>
          </span>
        }
        description="Apply a digest-pinned manifest. The control plane schedules signed assignments; every host admits or refuses them under its own policy, and progress is what hosts observe."
        actions={
          <>
            <PrimaryButton onClick={() => navigate('/deploy/new')}>
              <Plus className="w-4 h-4" /> New Deployment
            </PrimaryButton>
            <GhostButton onClick={() => navigate('/deploy/new?mode=yaml')}>
              <FileCode2 className="w-4 h-4" /> Apply YAML
            </GhostButton>
          </>
        }
        markers={[]}
        arcs={[]}
        focusLng={30}
        frozen={res.stale}
        overlay={pipeline}
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-5 gap-3">
        {strategies.map((s) => (
          <div key={s.id} className={`dh-feature p-4 text-left ${s.state === 'UNAVAILABLE' || s.state === 'PLANNED' ? 'opacity-60' : ''}`} style={{ ['--dh-accent' as string]: TONE[s.tone], ['--dh-accent-2' as string]: s.accent2 } as React.CSSProperties}>
            <div className="flex items-start gap-2.5">
              <IconTile tone={s.tone}>{s.icon}</IconTile>
              <div className="min-w-0">
                <div className="text-[14.5px] font-bold text-white leading-tight">{s.title}</div>
                <TruthTag state={s.state} />
              </div>
            </div>
            <div className="text-[11.5px] text-slate-400 leading-snug mt-2">{s.desc}</div>
            <ul className="mt-3 space-y-1.5">
              {s.points.map((p) => (
                <li key={p} className="flex items-center gap-2 text-[12px] text-slate-300">
                  <Check className={`w-3.5 h-3.5 shrink-0 ${s.state === 'LIVE' || s.state === 'CONFIGURED' ? 'text-emerald-400' : 'text-slate-600'}`} />
                  {p}
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      <Gate res={res}>
        {(d) => (
          <>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <KpiTile tone="cyan" icon={<Server className="w-6 h-6" />} label="Applications" value={latestPerApp.size} />
              <KpiTile tone="blue" icon={<Layers className="w-6 h-6" />} label="Deployments (generations)" value={d.deployments.length} />
              <KpiTile tone="emerald" icon={<CheckCircle2 className="w-6 h-6" />} label="Current generations READY" value={`${[...latestPerApp.values()].filter((x) => x.state === 'READY').length} / ${latestPerApp.size}`} />
              <KpiTile tone="violet" icon={<Clock className="w-6 h-6" />} label="Last deployment" value={latest ? since(latest.submittedAt) : '—'} sub={latest ? `${latest.app} #${latest.generation} · ${latest.state}` : undefined} />
            </div>
            {d.deployments.length === 0 ? (
              <Empty title="No deployments yet" action={<PrimaryButton onClick={() => navigate('/deploy/new')}><Plus className="w-4 h-4" /> New Deployment</PrimaryButton>} />
            ) : (
              <Glass className="overflow-hidden">
                <div className="px-4 pt-4"><PanelHeader title="Deployments" subtitle="Each generation of each application, with state derived from its replica rows." /></div>
                <div className="overflow-x-auto mt-2">
                  <table className="dh-table w-full min-w-[860px]">
                    <thead><tr><th>Deployment</th><th>State</th><th>Stages</th><th>Artifact</th><th>Change</th><th>Actor</th><th>Submitted</th></tr></thead>
                    <tbody>
                      {d.deployments.map((x) => (
                        <tr key={x.id}>
                          <td><Link to={`/deploy/${encodeURIComponent(x.app)}/${x.generation}`} className="text-cyan-300 hover:underline font-semibold">{x.app} #{x.generation}</Link></td>
                          <td><StatusPill status={x.state} /></td>
                          <td>
                            <div className="flex gap-1" aria-label="stages">
                              {x.stages.map((s) => (
                                <span key={s.id} title={`${s.label}: ${s.state} — ${s.detail}`} className={`w-5 h-1.5 rounded-full ${s.state === 'PASSED' ? 'bg-emerald-400' : s.state === 'FAILED' ? 'bg-rose-400' : s.state === 'RUNNING' ? 'bg-blue-400' : 'bg-slate-700'}`} />
                              ))}
                            </div>
                          </td>
                          <td className="font-mono text-[11.5px] text-slate-300" title={x.image}>{shortDigest(x.imageDigest)}</td>
                          <td className="text-slate-300">{x.change ?? '—'}</td>
                          <td className="text-slate-300">{x.actor ?? '—'}</td>
                          <td className="text-slate-300">{since(x.submittedAt)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Glass>
            )}
          </>
        )}
      </Gate>
    </SurfaceLayout>
  );
}
