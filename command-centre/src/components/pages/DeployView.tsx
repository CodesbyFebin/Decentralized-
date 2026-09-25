import React, { useEffect, useMemo, useState } from 'react';
import {
  Plus,
  Github,
  Gitlab,
  Container,
  Upload,
  LayoutTemplate,
  Sparkles,
  Home,
  Globe2,
  Users,
  Box,
  Network,
  Check,
  Server,
  Layers,
  CheckCircle2,
  Clock,
  Monitor,
  MoreHorizontal,
  GitBranch,
  Link2,
  Undo2,
  Settings2,
  FileText,
  Database,
  Zap,
  RefreshCw,
  ArrowDown,
  Cuboid,
  Layers3,
  Download
} from 'lucide-react';
import { api } from '../../lib/api';
import { DeployOverview, FleetOverview, FleetDeployment, Ownership } from '../../types/platform';
import { NavRoute } from '../layout/Sidebar';
import { HoloGlobe } from '../common/HoloGlobe';
import { SurfaceLayout, SurfaceHero, regionMarkers, regionArcs } from '../common/CommandSurface';
import {
  Glass,
  KpiTile,
  FilterChips,
  TableToolbar,
  RailPanel,
  RailItem,
  NetworkRow,
  RegionCallout,
  PanelHeader,
  IconTile,
  StatusPill,
  ViewAll,
  Grad,
  PrimaryButton,
  GhostButton,
  DemoTag,
  Donut,
  MiniBars,
  OwnershipBadges,
  OWNERSHIP_META,
  TONE,
  Tone
} from '../common/ui';
import { DeployWizard } from './DeployWizard';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
  entityId?: string;
}

type Chip = 'all' | 'production' | 'preview' | 'stopped';
type Layer = 'all' | Ownership;

const STRATEGIES: { id: string; title: string; desc: string; icon: React.ReactNode; tone: Tone; accent2: string; points: string[] }[] = [
  { id: 'self-host', title: 'Self Host', desc: 'Deploy to hardware you control.', icon: <Home className="w-6 h-6" />, tone: 'cyan', accent2: '#248BFF', points: ['Private mesh', 'Full control', 'Your storage', 'Your networking'] },
  { id: 'edge', title: 'Edge Deploy', desc: 'Distribute workloads across edge nodes.', icon: <Globe2 className="w-6 h-6" />, tone: 'blue', accent2: '#5965FF', points: ['Global placement', 'Replicas', 'Health routing', 'Edge cache'] },
  { id: 'community', title: 'Community Deploy', desc: 'Run on independent operators.', icon: <Users className="w-6 h-6" />, tone: 'violet', accent2: '#C026D3', points: ['Verified operators', 'Resource marketplace', 'Workload isolation', 'Fair resource usage'] },
  { id: 'depin', title: 'DePIN Deploy', desc: 'Deploy through decentralized compute networks.', icon: <Box className="w-6 h-6" />, tone: 'blue', accent2: '#20DDF7', points: ['Provider discovery', 'GPU/CPU selection', 'Network-native settlement', 'External-network provenance'] },
  { id: 'hybrid', title: 'Hybrid Deploy', desc: 'Combine multiple infrastructure sources.', icon: <Network className="w-6 h-6" />, tone: 'violet', accent2: '#20DDF7', points: ['Primary + failover', 'Edge + community', 'Your storage', 'Custom topology'] }
];

const SOURCES: { id: string; title: string; subtitle: string; icon: React.ReactNode; tone: Tone }[] = [
  { id: 'github', title: 'Import from GitHub', subtitle: 'Connect and deploy a repository', icon: <Github className="w-5 h-5" />, tone: 'slate' },
  { id: 'container', title: 'Deploy Container', subtitle: 'Use a Docker/OCI image', icon: <Container className="w-5 h-5" />, tone: 'blue' },
  { id: 'upload', title: 'Upload Project', subtitle: 'Deploy from a local directory or archive', icon: <Upload className="w-5 h-5" />, tone: 'slate' },
  { id: 'template', title: 'Use Template', subtitle: 'Start with a pre-configured template', icon: <LayoutTemplate className="w-5 h-5" />, tone: 'violet' },
  { id: 'copilot', title: 'Deploy with Copilot', subtitle: 'Describe and deploy with AI assistance', icon: <Sparkles className="w-5 h-5" />, tone: 'blue' }
];

const SOURCE_ICON: Record<FleetDeployment['source'], React.ReactNode> = {
  GitHub: <Github className="w-4 h-4 text-slate-100" />,
  GitLab: <Gitlab className="w-4 h-4 text-orange-400" />,
  Docker: <Container className="w-4 h-4 text-sky-400" />,
  Upload: <Upload className="w-4 h-4 text-slate-300" />,
  Template: <LayoutTemplate className="w-4 h-4 text-violet-300" />
};

const fmtDuration = (s: number) => `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`;
const fmtCount = (n: number) => (n >= 1e6 ? `${(n / 1e6).toFixed(1)}M` : n >= 1e3 ? `${(n / 1e3).toFixed(1)}K` : String(n));

export const DeployView: React.FC<Props> = ({ onNavigate, entityId }) => {
  const [data, setData] = useState<DeployOverview | null>(null);
  const [fleet, setFleet] = useState<FleetOverview | null>(null);
  const [digest, setDigest] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [chip, setChip] = useState<Chip>('all');
  const [layer, setLayer] = useState<Layer>('all');
  const [search, setSearch] = useState('');
  const [sortDesc, setSortDesc] = useState(false);

  const load = () => {
    setError(null);
    api.getDeployOverview().then((r) => setData(r.data)).catch((e) => setError(e.message));
    api.getFleet().then((r) => setFleet(r.data)).catch(() => {});
    api
      .getDeployments()
      .then((r) => setDigest(r.data.find((d) => d.artifactDigest)?.artifactDigest ?? null))
      .catch(() => {});
  };
  useEffect(load, []);

  const deployments = data?.deployments ?? [];
  const counts = useMemo(
    () => ({
      all: deployments.length,
      production: deployments.filter((d) => d.environment === 'Production' && d.status !== 'Stopped').length,
      preview: deployments.filter((d) => d.environment === 'Preview' && d.status !== 'Stopped').length,
      stopped: deployments.filter((d) => d.status === 'Stopped').length,
      active: deployments.filter((d) => d.status !== 'Stopped').length,
      replicas: deployments.reduce((a, d) => a + d.replicasObserved, 0),
      networks: new Set(deployments.flatMap((d) => d.placement)).size,
      regions: new Set(deployments.filter((d) => d.status !== 'Stopped').map((d) => d.region)).size
    }),
    [deployments]
  );

  const rows = useMemo(() => {
    let list = deployments.filter((d) =>
      chip === 'all' ? true : chip === 'stopped' ? d.status === 'Stopped' : d.status !== 'Stopped' && d.environment.toLowerCase() === chip
    );
    const q = search.trim().toLowerCase();
    if (q) list = list.filter((d) => [d.name, d.shortId, d.endpoint, d.sourceRef, d.commit].some((v) => v.toLowerCase().includes(q)));
    return [...list].sort((a, b) => (sortDesc ? -1 : 1) * a.name.localeCompare(b.name));
  }, [deployments, chip, search, sortDesc]);

  if (entityId?.startsWith('new')) {
    const preset = entityId.split(':')[1];
    return <DeployWizard onNavigate={onNavigate} initialSource={preset} />;
  }

  if (error && !data) {
    return (
      <Glass className="p-8 text-center mt-6">
        <p className="text-rose-300 text-sm">Deployment engine unavailable: {error}</p>
        <GhostButton onClick={load} className="mt-4">
          <RefreshCw className="w-4 h-4" /> Retry
        </GhostButton>
      </Glass>
    );
  }

  const startDeploy = (preset?: string) => onNavigate('deploy', preset ? `new:${preset}` : 'new');
  const regions = data?.regions ?? [];
  const owned = fleet?.nodes.filter((n) => n.ownership === 'owned' && n.status === 'Online').length;
  const community = fleet?.nodes.filter((n) => n.ownership === 'community').length;
  const depinProviders = data?.networks.filter((n) => n.id !== 'dh' && n.id !== 'cloud' && n.status === 'Active').length;

  // Region → deployments, optionally narrowed to one placement type.
  const regionDeployments = (region: string) =>
    deployments.filter((d) => d.status !== 'Stopped' && d.region === region && (layer === 'all' || d.placement.includes(layer))).length;
  const mapTone = (region: string): Tone =>
    ({ 'North America': 'cyan', Europe: 'violet', Asia: 'blue', 'South America': 'rose', Africa: 'amber', Oceania: 'emerald' } as Record<string, Tone>)[region] ?? 'cyan';
  const mapMarkers = regionMarkers(regions, (r) => {
    const n = regionDeployments(r.region);
    if (!n) return null;
    const tone = mapTone(r.region);
    return { tone, callout: <RegionCallout tone={tone} title={r.region} lines={[`${n} deployment${n > 1 ? 's' : ''}`]} /> };
  });

  const pipeline = (
    <div className="hidden xl:flex absolute right-0 top-[-6px] z-10 items-start gap-3 pointer-events-auto select-none origin-top-right scale-[0.72] min-[1760px]:scale-90 min-[1960px]:scale-100">
      {/* Source */}
      <Glass strong className="px-3 py-2.5 flex flex-col items-center gap-3">
        <span className="text-[11px] font-semibold text-slate-200">Source</span>
        <Github className="w-5 h-5 text-white" />
        <Gitlab className="w-5 h-5 text-orange-400" />
        <Database className="w-5 h-5 text-blue-400" />
        <Container className="w-5 h-5 text-sky-400" />
        <FileText className="w-5 h-5 text-slate-300" />
      </Glass>
      {[
        { label: 'Build', icon: <Cuboid className="w-7 h-7" />, detail: (
          <ul className="mt-2 space-y-0.5">
            {['Install', 'Build', 'Test', 'Package', 'Sign'].map((s) => (
              <li key={s} className="flex items-center gap-1.5 text-[10.5px] text-slate-300"><Check className="w-3 h-3 text-emerald-400" />{s}</li>
            ))}
          </ul>
        ) },
        { label: 'Artifact', icon: <Layers3 className="w-7 h-7" />, detail: (
          <div className="mt-2 text-center">
            <div className="text-[10.5px] text-slate-300 leading-tight">Immutable<br />&amp; Verified</div>
            <div className="mt-1.5 px-1.5 py-0.5 rounded-md bg-black/40 border border-white/10 text-[10px] font-mono text-cyan-200" title={digest ?? undefined}>
              {(digest ?? 'b3:pending').replace(/^sha256:/, 'b3:').slice(0, 9)}…
            </div>
          </div>
        ) },
        { label: 'Deploy', icon: <Layers className="w-7 h-7" />, detail: null }
      ].map((s) => (
        <div key={s.label} className="flex items-start gap-3 pt-8">
          <svg width="26" height="12" className="mt-6 shrink-0" aria-hidden="true">
            <line x1="0" y1="6" x2="20" y2="6" stroke={TONE.cyan} strokeWidth="2" className="dh-flow" />
            <path d="M18 1 L25 6 L18 11" fill="none" stroke={TONE.cyan} strokeWidth="2" />
          </svg>
          <div className="flex flex-col items-center w-[74px]">
            <span className="text-[12px] font-semibold text-slate-100 mb-1.5">{s.label}</span>
            <IconTile tone="cyan" size="lg" className="!w-14 !h-14 !rounded-2xl">{s.icon}</IconTile>
            {s.detail}
          </div>
        </div>
      ))}
      <svg width="34" height="210" className="shrink-0 mt-2" aria-hidden="true">
        {[22, 76, 130, 184].map((y) => (
          <path key={y} d={`M0 88 C 18 88, 14 ${y}, 30 ${y}`} fill="none" stroke={TONE.cyan} strokeWidth="1.6" className="dh-flow" />
        ))}
      </svg>
      <div className="flex flex-col gap-2 pt-0">
        {[
          { t: 'My Nodes', s: owned !== undefined ? `${owned} nodes` : '—', tone: 'cyan' as Tone, icon: <Home className="w-4 h-4" /> },
          { t: 'Community', s: community !== undefined ? `${community} nodes` : '—', tone: 'violet' as Tone, icon: <Users className="w-4 h-4" /> },
          { t: 'DePIN', s: depinProviders !== undefined ? `${depinProviders} provider${depinProviders === 1 ? '' : 's'}` : '—', tone: 'blue' as Tone, icon: <Box className="w-4 h-4" /> },
          { t: 'Edge', s: `${counts.regions} regions`, tone: 'blue' as Tone, icon: <Globe2 className="w-4 h-4" /> }
        ].map((x) => (
          <Glass strong key={x.t} className="flex items-center gap-2.5 pl-2 pr-4 py-2 min-w-[132px]" style={{ borderColor: `${TONE[x.tone]}66` }}>
            <IconTile tone={x.tone} size="sm">{x.icon}</IconTile>
            <div className="leading-tight">
              <div className="text-[12px] font-bold text-white">{x.t}</div>
              <div className="text-[10.5px] text-slate-300">{x.s}</div>
            </div>
          </Glass>
        ))}
      </div>
    </div>
  );

  const mix = data?.ownershipMix ?? [];
  const mainOwned = mix.find((m) => m.ownership === 'owned')?.percent ?? 0;

  const rail = (
    <>
      <RailPanel title="New Deployment" subtitle="Choose a source to deploy your application.">
        <div className="space-y-1.5">
          {SOURCES.map((s) => (
            <RailItem
              key={s.id}
              icon={s.icon}
              tone={s.tone}
              title={s.title}
              subtitle={s.subtitle}
              onClick={() => (s.id === 'copilot' ? onNavigate('copilot') : startDeploy(s.id))}
            />
          ))}
        </div>
      </RailPanel>

      <RailPanel title="Deployment Templates" action={<ViewAll onClick={() => startDeploy('template')} />}>
        <div className="space-y-0.5 -mt-1">
          {data?.templates.map((t) => (
            <button key={t.id} onClick={() => startDeploy(t.id)} className="w-full flex items-center gap-3 py-2 px-1 rounded-xl hover:bg-white/[0.035] text-left group">
              <span
                className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0 font-extrabold text-[15px] border"
                style={{ color: t.accent, borderColor: `${t.accent}50`, background: `${t.accent}14` }}
                aria-hidden="true"
              >
                {t.glyph === 'bolt' ? <Zap className="w-4 h-4" /> : t.glyph === 'db' ? <Database className="w-4 h-4" /> : t.glyph === 'cube' ? <Box className="w-4 h-4" /> : t.glyph}
              </span>
              <div className="flex-1 min-w-0">
                <div className="text-[12.5px] font-semibold text-slate-100">{t.name}</div>
                <div className="text-[10.5px] text-slate-400 truncate">{t.description}</div>
              </div>
              <span className="text-slate-500 group-hover:text-cyan-300">›</span>
            </button>
          ))}
        </div>
      </RailPanel>

      <RailPanel title="Execution Networks" action={<ViewAll onClick={() => onNavigate('nodes')} />}>
        <div className="space-y-0.5 -mt-1">{data?.networks.map((n) => <NetworkRow key={n.id} network={n} onClick={() => onNavigate('nodes')} />)}</div>
      </RailPanel>

      <RailPanel title="Developer Experience">
        <div className="grid grid-cols-4 gap-1.5">
          {[
            { t: 'Git Push', s: 'Auto Deploy', icon: <GitBranch className="w-5 h-5" />, tone: 'rose' as Tone },
            { t: 'Preview URLs', s: 'Per Branch', icon: <Link2 className="w-5 h-5" />, tone: 'violet' as Tone },
            { t: 'Rollbacks', s: 'One Click', icon: <Undo2 className="w-5 h-5" />, tone: 'amber' as Tone },
            { t: 'Environment', s: 'Management', icon: <Settings2 className="w-5 h-5" />, tone: 'rose' as Tone }
          ].map((x) => (
            <div key={x.t} className="flex flex-col items-center text-center gap-1.5 p-2 rounded-xl bg-white/[0.03] border border-[rgba(125,190,255,0.1)]">
              <IconTile tone={x.tone} size="sm">{x.icon}</IconTile>
              <div className="text-[10px] leading-tight text-slate-200 font-semibold">{x.t}</div>
              <div className="text-[9.5px] leading-tight text-slate-400 -mt-1">{x.s}</div>
            </div>
          ))}
        </div>
      </RailPanel>
    </>
  );

  return (
    <SurfaceLayout rail={rail}>
      <SurfaceHero
        eyebrow={
          <div className="inline-flex items-center gap-2 text-[12px] font-semibold tracking-[0.18em] text-slate-200">
            <span className="text-cyan-300">›</span> UNIVERSAL DEPLOY
            <span className="text-slate-600">·</span>
            <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md border text-[10.5px] ${data?.mode === 'live' ? 'text-emerald-300 border-emerald-400/30 bg-emerald-500/10' : 'text-amber-300 border-amber-400/30 bg-amber-500/10'}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${data?.mode === 'live' ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400'}`} />
              {data?.mode === 'live' ? 'LIVE' : 'DEMO'}
            </span>
          </div>
        }
        title={
          <>
            <span className="xl:text-[40px] min-[1760px]:text-[44px]">
              Deploy <Grad from="#38BDF8" to="#818CF8">anywhere.</Grad>
              <br />
              Without <Grad from="#60A5FA" to="#C084FC">giving up control.</Grad>
            </span>
          </>
        }
        description="Deploy from Git, containers or artifacts to your hardware, edge nodes, community infrastructure or supported decentralized compute networks."
        actions={
          <>
            <PrimaryButton onClick={() => startDeploy()}>
              <Plus className="w-4 h-4" /> New Deployment
            </PrimaryButton>
            <GhostButton onClick={() => startDeploy('github')}>
              <Download className="w-4 h-4" /> Import Repository
            </GhostButton>
          </>
        }
        markers={[]}
        arcs={regionArcs(regions, (r) => r.deployments > 0)}
        focusLng={30}
        overlay={pipeline}
      />

      {/* Deployment strategies */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-5 gap-3">
        {STRATEGIES.map((s) => (
          <button
            key={s.id}
            onClick={() => startDeploy(s.id)}
            className="dh-feature dh-hover-lift p-4 text-left"
            style={{ ['--dh-accent' as string]: TONE[s.tone], ['--dh-accent-2' as string]: s.accent2 } as React.CSSProperties}
          >
            <div className="flex items-start gap-2.5">
              <IconTile tone={s.tone}>{s.icon}</IconTile>
              <div className="min-w-0">
                <div className="text-[14.5px] font-bold text-white leading-tight">{s.title}</div>
                <div className="text-[11.5px] text-slate-400 leading-snug mt-0.5">{s.desc}</div>
              </div>
            </div>
            <ul className="mt-3.5 space-y-1.5">
              {s.points.map((p) => (
                <li key={p} className="flex items-center gap-2 text-[12px] text-slate-300">
                  <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                  {p}
                </li>
              ))}
            </ul>
          </button>
        ))}
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-3 2xl:grid-cols-5 gap-3">
        <KpiTile tone="cyan" icon={<Server className="w-6 h-6" />} label="Active Deployments" value={counts.active} sub={<span className="text-emerald-400">↑ {data?.deploysThisWeek ?? 0} this week</span>} spark={[6, 7, 7, 9, 8, 10, 11, 12]} />
        <KpiTile tone="blue" icon={<Layers className="w-6 h-6" />} label="Total Replicas" value={counts.replicas} sub={`Across ${counts.networks} networks`} spark={[18, 19, 22, 21, 24, 25, 27, 28]} />
        <KpiTile tone="emerald" icon={<CheckCircle2 className="w-6 h-6" />} label="Successful Deployments" value={`${data?.successRatePercent ?? 0}%`} sub={<span className="text-emerald-400">↑ 2.1%</span>} spark={[95, 96, 96, 97, 96, 98, 98, 98.4]} />
        <KpiTile tone="blue" icon={<Clock className="w-6 h-6" />} label="Avg. Deployment Time" value={data ? fmtDuration(data.avgDeploySeconds) : '—'} sub={<span className="text-emerald-400 inline-flex items-center"><ArrowDown className="w-3 h-3" /> 28%</span>} spark={[200, 190, 185, 170, 160, 150, 140, 134]} />
        <KpiTile tone="violet" icon={<Globe2 className="w-6 h-6" />} label="Global Regions" value={counts.regions} trailing={<div className="absolute right-3 bottom-3"><MiniBars values={[1, 2, 3, 4, 5]} tone="violet" /></div>} />
      </div>

      {/* Map + ownership */}
      <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)] gap-4">
        <Glass className="p-4 overflow-hidden">
          <PanelHeader
            icon={<IconTile tone="cyan" size="sm"><Globe2 className="w-4 h-4" /></IconTile>}
            title="Live Deployment Map"
            subtitle="Real-time view of your deployments across owned, community and decentralized networks."
            right={
              <FilterChips<Layer>
                chips={[
                  { id: 'all', label: 'All Deployments', tone: 'cyan' },
                  { id: 'owned', label: 'My Nodes', tone: 'blue' },
                  { id: 'community', label: 'Community', tone: 'violet' },
                  { id: 'depin', label: 'DePIN', tone: 'blue' },
                  { id: 'edge', label: 'Edge', tone: 'emerald' }
                ]}
                active={layer}
                onChange={setLayer}
              />
            }
          />
          <HoloGlobe
            markers={mapMarkers}
            arcs={regionArcs(regions, (r) => regionDeployments(r.region) > 0)}
            focusLng={20}
            tilt={22}
            speed={1.5}
            center={[0.5, 0.66]}
            radius={0.62}
            resolution={1.3}
            className="h-[300px] -mx-4 -mb-4 mt-1"
          />
        </Glass>

        <div className="grid gap-4">
          <Glass className="p-4">
            <PanelHeader title="Infrastructure Ownership" right={<span className="px-2 py-1 rounded-lg bg-white/[0.04] border border-[rgba(125,190,255,0.14)] text-[11px] text-slate-300">All Deployments</span>} />
            <div className="mt-3 flex items-center gap-5">
              <Donut size={116} stroke={13} segments={mix.map((m) => ({ value: m.percent, tone: OWNERSHIP_META[m.ownership].tone }))}>
                <div className="text-[22px] font-extrabold text-white leading-none tabular-nums">{mainOwned}%</div>
                <div className="text-[10px] text-slate-400 mt-1">Your Hardware</div>
              </Donut>
              <ul className="flex-1 space-y-2">
                {mix.map((m) => (
                  <li key={m.ownership} className="flex items-center justify-between text-[12px]">
                    <span className="flex items-center gap-2 text-slate-300">
                      <span className="w-2 h-2 rounded-full" style={{ background: TONE[OWNERSHIP_META[m.ownership].tone] }} />
                      {OWNERSHIP_META[m.ownership].label}
                    </span>
                    <span className="font-semibold text-slate-100 tabular-nums">{m.percent}%</span>
                  </li>
                ))}
              </ul>
            </div>
          </Glass>
          <Glass className="p-4">
            <PanelHeader title="Deployment Traffic (Last 24h)" />
            <div className="mt-3 grid grid-cols-2 gap-3">
              {data && [
                { v: fmtCount(data.traffic24h.requests), l: 'Requests', d: data.traffic24h.requestsDeltaPercent, tone: 'cyan' as Tone },
                { v: `${data.traffic24h.bandwidthGb} GB`, l: 'Bandwidth', d: data.traffic24h.bandwidthDeltaPercent, tone: 'violet' as Tone }
              ].map((x) => (
                <div key={x.l} className="flex items-stretch gap-2.5">
                  <span className="w-1 rounded-full" style={{ background: TONE[x.tone], boxShadow: `0 0 8px ${TONE[x.tone]}` }} />
                  <div>
                    <div className="text-[20px] font-extrabold text-white tabular-nums leading-tight">{x.v}</div>
                    <div className="text-[11px] text-slate-400 flex items-center gap-2">
                      {x.l} <span className="text-emerald-400 font-semibold">↑ {x.d}%</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Glass>
        </div>
      </div>

      {/* Deployments table */}
      <Glass className="overflow-hidden">
        <TableToolbar search={search} onSearch={setSearch} placeholder="Search deployments..." onSort={() => setSortDesc(!sortDesc)}>
          <FilterChips<Chip>
            chips={[
              { id: 'all', label: 'All Deployments', count: counts.all },
              { id: 'production', label: 'Production', count: counts.production, tone: 'emerald' },
              { id: 'preview', label: 'Preview', count: counts.preview, tone: 'blue' },
              { id: 'stopped', label: 'Stopped', count: counts.stopped, tone: 'slate' }
            ]}
            active={chip}
            onChange={setChip}
          />
        </TableToolbar>
        <div className="overflow-x-auto">
          <table className="dh-table w-full min-w-[900px]">
            <thead>
              <tr>
                <th>Name</th>
                <th>Status</th>
                <th>Source</th>
                <th>Environment</th>
                <th>Replicas</th>
                <th>Placement</th>
                <th>Domain / Endpoint</th>
                <th>Last Deployed</th>
                <th className="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((d) => (
                <tr key={d.id}>
                  <td>
                    <div className="flex items-center gap-2.5">
                      <IconTile tone={d.status === 'Degraded' ? 'emerald' : 'blue'} size="sm"><Server className="w-4 h-4" /></IconTile>
                      <div>
                        <div className="font-semibold text-slate-100">{d.name}</div>
                        <div className="text-[11px] text-slate-500 font-mono">{d.shortId}</div>
                      </div>
                    </div>
                  </td>
                  <td><StatusPill status={d.status} /></td>
                  <td>
                    <div className="flex items-center gap-2">
                      {SOURCE_ICON[d.source]}
                      <div className="leading-tight">
                        <div className="text-slate-200">{d.source}</div>
                        <div className="text-[11px] text-slate-500 font-mono">{d.sourceRef}</div>
                      </div>
                    </div>
                  </td>
                  <td><StatusPill status={d.environment} tone={d.environment === 'Production' ? 'emerald' : 'blue'} dot={d.environment === 'Preview'} /></td>
                  <td className={`tabular-nums ${d.replicasObserved < d.replicasDesired ? 'text-amber-300' : 'text-slate-200'}`} title="observed / desired">
                    {d.replicasObserved} / {d.replicasDesired}
                  </td>
                  <td><OwnershipBadges placement={d.placement} /></td>
                  <td>
                    <a href={d.endpoint} target="_blank" rel="noreferrer" className="text-sky-300 hover:text-sky-200 underline decoration-sky-400/40 underline-offset-2">{d.endpoint.replace(/^https?:\/\//, '')}</a>
                  </td>
                  <td>
                    <div className="text-slate-200">{d.lastDeployed}</div>
                    <div className="text-[11px] text-slate-500 font-mono">{d.commit}</div>
                  </td>
                  <td>
                    <div className="flex justify-end gap-1.5">
                      <button onClick={() => onNavigate('apps')} className="p-1.5 rounded-lg border border-[rgba(125,190,255,0.16)] text-slate-300 hover:text-cyan-300 hover:border-cyan-400/40" aria-label={`Open ${d.name}`}>
                        <Monitor className="w-4 h-4" />
                      </button>
                      <button onClick={() => onNavigate('apps')} className="p-1.5 rounded-lg border border-[rgba(125,190,255,0.16)] text-slate-300 hover:text-cyan-300 hover:border-cyan-400/40" aria-label={`More actions for ${d.name}`}>
                        <MoreHorizontal className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={9} className="text-center text-slate-500 py-8">No deployments match this filter.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {data && (
          <div className="flex items-center justify-between px-4 py-2.5 text-[11px] text-slate-500 border-t border-[rgba(125,190,255,0.08)]">
            <span>Replicas show observed / desired. Observed {new Date(data.observedAt).toLocaleTimeString()}</span>
            <DemoTag mode={data.mode} />
          </div>
        )}
      </Glass>
    </SurfaceLayout>
  );
};
