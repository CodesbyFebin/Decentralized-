import React, { useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Rocket, FileCode2, Plus, Trash2 } from 'lucide-react';
import type { ArtifactRec } from '../../types/reality';
import { useResource } from '../../lib/useResource';
import { useSession } from '../../lib/session';
import { api, ApiError } from '../../lib/client';
import { Glass, PanelHeader, PrimaryButton, GhostButton } from '../common/ui';
import { ErrorState, Note, shortDigest } from '../common/states';

interface Form {
  name: string;
  runtime: 'process' | 'docker';
  image: string;
  replicas: number;
  cpu: string;
  mem: string;
  tiers: string;
  spread: 'failure-domain' | 'none';
  antiAffinity: 'hard' | 'soft' | 'none';
  port: string;
  containerPort: string;
  ingressHost: string;
  tls: 'local' | 'acme' | 'none';
  healthPath: string;
  env: { k: string; v: string }[];
}

const q = (s: string) => JSON.stringify(s); // JSON strings are valid YAML double-quoted scalars

/** Build a dh/v1 manifest. Only fields the control plane's schema accepts. */
export function toManifest(f: Form): string {
  const lines = [
    'apiVersion: dh/v1',
    'kind: Application',
    'metadata:',
    `  name: ${f.name}`,
    'spec:',
    `  replicas: ${f.replicas}`,
    ...(f.runtime === 'docker' ? ['  runtime: docker'] : []),
    `  image: ${f.image}`,
    ...(f.env.filter((e) => e.k).length ? ['  env:', ...f.env.filter((e) => e.k).map((e) => `    ${e.k}: ${q(e.v)}`)] : []),
    '  resources:',
    `    cpu: ${f.cpu}`,
    `    mem: ${f.mem}`,
    '  placement:',
    `    tiers: [${f.tiers.split(',').map((t) => t.trim()).filter(Boolean).join(', ')}]`,
    `    spread: ${f.spread}`,
    `    antiAffinity: ${f.antiAffinity}`,
    ...(f.port ? ['  ports:', `    - name: ${f.port}`, ...(f.runtime === 'docker' && f.containerPort ? [`      container: ${Number(f.containerPort)}`] : [])] : []),
    ...(f.ingressHost && f.port ? ['  ingress:', `    - host: ${f.ingressHost}`, `      port: ${f.port}`, `      tls: ${f.tls}`] : []),
    ...(f.healthPath && f.port ? ['  health:', `    http: ${f.healthPath}`, '    interval: 5s'] : [])
  ];
  return lines.join('\n') + '\n';
}

export function validateForm(f: Form): string[] {
  const errs: string[] = [];
  if (!/^[a-z0-9][a-z0-9-]{0,41}$/.test(f.name)) errs.push('Name: 1–42 lowercase letters, digits or dashes.');
  if (f.runtime === 'process' && !/^[a-z0-9._-]+@b3:[0-9a-f]{64}$/.test(f.image)) errs.push('Process image must be name@b3:<64 hex> (pick an artifact).');
  if (f.runtime === 'docker' && !/^[\w./:-]+@sha256:[0-9a-f]{64}$/.test(f.image)) errs.push('Docker image must be pinned: ref@sha256:<64 hex>. Tags such as :latest are refused.');
  if (!Number.isInteger(f.replicas) || f.replicas < 1 || f.replicas > 64) errs.push('Replicas: 1–64.');
  if (!/^\d+m?$/.test(f.cpu)) errs.push('CPU like 100m or 1.');
  if (!/^\d+(Ki|Mi|Gi)?$/.test(f.mem)) errs.push('Memory like 64Mi or 1Gi.');
  if (f.ingressHost && !/^[a-z0-9.-]+\.[a-z]{2,}$/.test(f.ingressHost)) errs.push('Ingress host must be a DNS name.');
  if (f.ingressHost && !f.port) errs.push('Ingress needs a named port.');
  for (const e of f.env) if (e.k && !/^[A-Z_][A-Z0-9_]*$/.test(e.k)) errs.push(`Env name "${e.k}" must be UPPER_SNAKE_CASE.`);
  return errs;
}

export default function DeployNewView() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { can, mode } = useSession();
  const arts = useResource<{ artifacts: ArtifactRec[] }>('/deployments', { pollMs: 0 });
  const [mode2, setMode2] = useState<'form' | 'yaml'>(params.get('mode') === 'yaml' ? 'yaml' : 'form');
  const [yaml, setYaml] = useState('');
  const [f, setF] = useState<Form>({
    name: '',
    runtime: params.get('runtime') === 'docker' ? 'docker' : 'process',
    image: '',
    replicas: 2,
    cpu: '100m',
    mem: '64Mi',
    tiers: 'trusted',
    spread: 'failure-domain',
    antiAffinity: 'hard',
    port: 'http',
    containerPort: '8080',
    ingressHost: '',
    tls: 'local',
    healthPath: '/healthz',
    env: []
  });
  const [step, setStep] = useState<'configure' | 'review'>('configure');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const [refused, setRefused] = useState<string | null>(null);

  const manifest = mode2 === 'yaml' ? yaml : toManifest(f);
  const errors = mode2 === 'yaml' ? (yaml.trim() ? [] : ['Paste a dh/v1 manifest.']) : validateForm(f);
  const set = <K extends keyof Form>(k: K, v: Form[K]) => setF((x) => ({ ...x, [k]: v }));
  const allowed = can('api.write') && mode === 'controlplane';
  const artifacts = useMemo(() => arts.data?.artifacts ?? [], [arts.data]);

  const submit = async () => {
    setBusy(true);
    setErr(null);
    setRefused(null);
    try {
      const r = await api.post<{ data: { ok: boolean; message: string; app: string | null; deployment: { app: string; generation: number } | null } }>('/deployments', { manifest });
      if (r.data.deployment) navigate(`/deploy/${encodeURIComponent(r.data.deployment.app)}/${r.data.deployment.generation}`);
      else if (r.data.app) navigate(`/apps/${encodeURIComponent(r.data.app)}`);
    } catch (e) {
      const x = e as ApiError;
      if (x.code === 'CONFLICT' || x.code === 'VALIDATION_FAILED') setRefused(x.message);
      else setErr(x);
    } finally {
      setBusy(false);
    }
  };

  const input = 'w-full rounded-lg bg-black/30 border border-[rgba(125,190,255,0.2)] focus:border-cyan-400/60 outline-none px-2.5 py-2 text-[13px]';
  const label = 'block text-[12px] text-slate-300 space-y-1';

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <Link to="/deploy" className="inline-flex items-center gap-1.5 text-[12.5px] text-slate-400 hover:text-cyan-300">
        <ArrowLeft className="w-3.5 h-3.5" /> Deploy
      </Link>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">New deployment</h1>
          <p className="text-[13px] text-slate-400">Compose a dh/v1 manifest, review the immutable intent, then apply it.</p>
        </div>
        <div className="flex gap-2">
          <GhostButton onClick={() => { setMode2('form'); setStep('configure'); }} className={mode2 === 'form' ? '!border-cyan-400/60' : ''}>Form</GhostButton>
          <GhostButton onClick={() => { setMode2('yaml'); setYaml(yaml || toManifest(f)); setStep('configure'); }} className={mode2 === 'yaml' ? '!border-cyan-400/60' : ''}><FileCode2 className="w-4 h-4" /> YAML</GhostButton>
        </div>
      </div>
      {!allowed && <Note>{mode === 'demo' ? 'The demo replay is read-only.' : 'Applying needs a capability with api.write. You can still compose and review.'}</Note>}

      {step === 'configure' && mode2 === 'form' && (
        <div className="grid lg:grid-cols-2 gap-4">
          <Glass className="p-4 space-y-3">
            <PanelHeader title="Source" subtitle="Only digest-pinned artifacts can be deployed." />
            <label className={label}>Application name<input className={input} value={f.name} onChange={(e) => set('name', e.target.value)} placeholder="my-app" /></label>
            <label className={label}>
              Runtime
              <select className={input} value={f.runtime} onChange={(e) => { set('runtime', e.target.value as Form['runtime']); set('image', ''); }}>
                <option value="process">process (artifact from the cluster CAS)</option>
                <option value="docker">docker (OCI image pinned by sha256)</option>
              </select>
            </label>
            {f.runtime === 'process' ? (
              <label className={label}>
                Artifact
                <select className={input} value={f.image} onChange={(e) => set('image', e.target.value)}>
                  <option value="">Select an artifact…</option>
                  {artifacts.map((a) => (
                    <option key={a.digest} value={`${a.name}@${a.digest}`}>{a.name} · {shortDigest(a.digest)}{a.attested ? ' · attested' : ' · UNSIGNED'}</option>
                  ))}
                </select>
                {!artifacts.length && <span className="text-slate-500">No artifacts in the CAS. Push one with dh artifact push.</span>}
              </label>
            ) : (
              <label className={label}>Image (ref@sha256:…)<input className={`${input} font-mono`} value={f.image} onChange={(e) => set('image', e.target.value.trim())} placeholder="ghcr.io/org/app@sha256:…" /></label>
            )}
            <div className="grid grid-cols-3 gap-2">
              <label className={label}>Replicas<input type="number" min={1} max={64} className={input} value={f.replicas} onChange={(e) => set('replicas', Number(e.target.value))} /></label>
              <label className={label}>CPU<input className={input} value={f.cpu} onChange={(e) => set('cpu', e.target.value)} /></label>
              <label className={label}>Memory<input className={input} value={f.mem} onChange={(e) => set('mem', e.target.value)} /></label>
            </div>
            <Note>The process runtime does not enforce CPU or memory limits; use the docker runtime when limits must be enforced.</Note>
          </Glass>
          <Glass className="p-4 space-y-3">
            <PanelHeader title="Placement & network" />
            <label className={label}>Tiers (comma-separated)<input className={input} value={f.tiers} onChange={(e) => set('tiers', e.target.value)} /></label>
            <div className="grid grid-cols-2 gap-2">
              <label className={label}>Spread<select className={input} value={f.spread} onChange={(e) => set('spread', e.target.value as Form['spread'])}><option>failure-domain</option><option>none</option></select></label>
              <label className={label}>Anti-affinity<select className={input} value={f.antiAffinity} onChange={(e) => set('antiAffinity', e.target.value as Form['antiAffinity'])}><option>hard</option><option>soft</option><option>none</option></select></label>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <label className={label}>Port name<input className={input} value={f.port} onChange={(e) => set('port', e.target.value)} placeholder="http (empty = no port)" /></label>
              {f.runtime === 'docker' && <label className={label}>Container port<input className={input} value={f.containerPort} onChange={(e) => set('containerPort', e.target.value)} /></label>}
            </div>
            <div className="grid grid-cols-2 gap-2">
              <label className={label}>Ingress host<input className={input} value={f.ingressHost} onChange={(e) => set('ingressHost', e.target.value)} placeholder="app.example.com" /></label>
              <label className={label}>TLS<select className={input} value={f.tls} onChange={(e) => set('tls', e.target.value as Form['tls'])}><option value="local">local (cluster CA)</option><option value="acme">acme</option><option value="none">none</option></select></label>
            </div>
            <label className={label}>Health check path<input className={input} value={f.healthPath} onChange={(e) => set('healthPath', e.target.value)} /></label>
            <div>
              <div className="flex items-center justify-between text-[12px] text-slate-300">Environment <button onClick={() => set('env', [...f.env, { k: '', v: '' }])} className="text-cyan-300 inline-flex items-center gap-1"><Plus className="w-3.5 h-3.5" /> add</button></div>
              {f.env.map((e, i) => (
                <div key={i} className="flex gap-2 mt-1.5">
                  <input className={`${input} font-mono`} value={e.k} placeholder="NAME" onChange={(x) => set('env', f.env.map((y, j) => (j === i ? { ...y, k: x.target.value } : y)))} />
                  <input className={input} value={e.v} placeholder="value" onChange={(x) => set('env', f.env.map((y, j) => (j === i ? { ...y, v: x.target.value } : y)))} />
                  <button onClick={() => set('env', f.env.filter((_, j) => j !== i))} aria-label="remove" className="text-slate-400 hover:text-rose-300"><Trash2 className="w-4 h-4" /></button>
                </div>
              ))}
              <Note>Env values are stored in control-plane state; there is no secrets store yet, so do not put secrets here.</Note>
            </div>
          </Glass>
        </div>
      )}

      {step === 'configure' && mode2 === 'yaml' && (
        <Glass className="p-4">
          <textarea value={yaml} onChange={(e) => setYaml(e.target.value)} rows={22} spellCheck={false} className="w-full rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.2)] p-3 font-mono text-[12px] text-cyan-100" placeholder="apiVersion: dh/v1&#10;kind: Application&#10;…" />
        </Glass>
      )}

      {step === 'review' && (
        <Glass className="p-4 space-y-3">
          <PanelHeader title="Review deployment intent" subtitle="This exact manifest is sent to POST /api/v1/apply. The control plane validates it, commits a new generation, and hosts decide admission." />
          <pre className="rounded-xl bg-[#020814] border border-[rgba(125,190,255,0.12)] p-3 text-[12px] text-cyan-100 font-mono overflow-x-auto">{mode2 === 'yaml' ? manifest : manifest.replace(/^(\s+[A-Z_][A-Z0-9_]*: ).*$/gm, '$1"••••"')}</pre>
          {mode2 === 'form' && f.env.length > 0 && <Note>Environment values are masked in this review.</Note>}
        </Glass>
      )}

      {errors.length > 0 && step === 'configure' && (
        <ul className="rounded-xl border border-amber-400/30 bg-amber-500/10 p-3 text-[12.5px] text-amber-100 list-disc list-inside">
          {errors.map((e) => <li key={e}>{e}</li>)}
        </ul>
      )}
      {refused && <div role="alert" className="rounded-xl border border-rose-400/30 bg-rose-500/10 p-3 text-[12.5px] text-rose-100"><b>Control plane refused the manifest:</b> {refused}</div>}
      {err && <ErrorState error={err} />}

      <div className="flex gap-2 justify-end">
        {step === 'review' && <GhostButton onClick={() => setStep('configure')}>Back</GhostButton>}
        {step === 'configure' ? (
          <PrimaryButton disabled={errors.length > 0} onClick={() => setStep('review')}>Review</PrimaryButton>
        ) : (
          <PrimaryButton disabled={!allowed || busy} onClick={submit}>
            <Rocket className="w-4 h-4" /> {busy ? 'Applying…' : 'Apply manifest'}
          </PrimaryButton>
        )}
      </div>
    </div>
  );
}
