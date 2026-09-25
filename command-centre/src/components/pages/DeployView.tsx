import React, { useState } from 'react';
import {
  Rocket,
  CheckCircle2,
  Globe,
  Server,
  FileCode,
  ShieldCheck,
  FolderArchive,
  ArrowRight,
  ArrowLeft,
  RefreshCw,
  Plus,
  Trash2,
  Lock,
  Layers,
  Sparkles,
  Info
} from 'lucide-react';
import { WorldMap } from '../common/WorldMap';
import { api } from '../../lib/api';
import { NavRoute } from '../layout/Sidebar';
import { CapabilityBadge } from '../common/CapabilityBadge';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
}

export const DeployView: React.FC<Props> = ({ onNavigate }) => {
  const [step, setStep] = useState<1 | 2 | 3>(2); // Start on Configure like screenshot
  const [domain, setDomain] = useState('yourdomain.com');
  const [appType, setAppType] = useState<'Static Site' | 'Web App' | 'WordPress' | 'Docker App' | 'Custom'>('Static Site');
  const [buildCommand, setBuildCommand] = useState('npm run build');
  const [outputDir, setOutputDir] = useState('dist');
  const [envVars, setEnvVars] = useState<{ key: string; value: string; isSecret: boolean }[]>([
    { key: 'VITE_API_URL', value: 'https://api.yourdomain.com', isSecret: false }
  ]);
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');

  // Selected regions
  const [selectedRegions, setSelectedRegions] = useState<Record<string, boolean>>({
    'North America': true,
    Europe: true,
    Asia: true,
    'South America': true,
    Africa: true,
    Australia: true
  });

  // Advanced options
  const [autoSSL, setAutoSSL] = useState(true);
  const [globalCDN, setGlobalCDN] = useState(true);
  const [autoBackup, setAutoBackup] = useState(true);
  const [ddosProtection, setDdosProtection] = useState(true);
  const [highAvailability, setHighAvailability] = useState(true);

  // Deploying modal state
  const [isDeploying, setIsDeploying] = useState(false);
  const [deploymentResult, setDeploymentResult] = useState<any>(null);
  const [activeDeployStage, setActiveDeployStage] = useState('VALIDATING');

  const totalSelectedRegionsCount = Object.values(selectedRegions).filter(Boolean).length;

  const toggleRegion = (region: string) => {
    setSelectedRegions((prev) => ({ ...prev, [region]: !prev[region] }));
  };

  const handleAddEnv = () => {
    if (!newKey.trim()) return;
    setEnvVars([...envVars, { key: newKey.trim(), value: newValue.trim(), isSecret: false }]);
    setNewKey('');
    setNewValue('');
  };

  const handleLaunch = async () => {
    setIsDeploying(true);
    setActiveDeployStage('VALIDATING');

    try {
      const regionsList = Object.keys(selectedRegions).filter((k) => selectedRegions[k]);

      // Stage progression simulation matching master prompt
      setTimeout(() => setActiveDeployStage('SCHEDULING'), 800);
      setTimeout(() => setActiveDeployStage('ARTIFACT_TRANSFER'), 1600);
      setTimeout(() => setActiveDeployStage('RUNTIME_CREATION'), 2400);
      setTimeout(() => setActiveDeployStage('HEALTH_CHECK'), 3200);
      setTimeout(() => setActiveDeployStage('ROUTING'), 4000);

      const res = await api.createDeployment({
        appName: domain.replace('.com', '').replace(/[^a-z0-9]/g, '-'),
        domain,
        sourceType: 'Git',
        sourceReference: 'main',
        regions: regionsList
      });

      setTimeout(() => {
        setActiveDeployStage('VERIFIED');
        setDeploymentResult(res.data);
      }, 4800);
    } catch (err: any) {
      alert(`Deployment failed: ${err.message}`);
      setIsDeploying(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Top Header & Multi-Step Wizard Header matching deploy.png */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="text-xs font-mono text-slate-400 flex items-center gap-2">
            <span onClick={() => onNavigate('apps')} className="hover:text-white cursor-pointer">
              Websites & Apps
            </span>
            <span>&gt;</span>
            <span className="text-cyan-400">Deploy</span>
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">
            Deploy to Decentralized Network
          </h1>
          <p className="text-xs text-slate-400">
            Deploy your website or application across multiple independent nodes worldwide.
          </p>
        </div>

        {/* Step Indicator 1-2-3 matching deploy.png */}
        <div className="flex items-center gap-2 bg-[#0D1527] border border-slate-800 p-1.5 rounded-2xl">
          {/* Step 1: Source */}
          <button
            onClick={() => setStep(1)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
              step === 1 ? 'bg-blue-600 text-white shadow-md' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <span className="w-5 h-5 rounded-full bg-slate-800 text-[11px] font-bold flex items-center justify-center">
              1
            </span>
            <div className="flex flex-col text-left">
              <span className="font-semibold leading-tight">Source</span>
              <span className="text-[9px] text-slate-400 font-mono">Your files</span>
            </div>
          </button>

          <span className="text-slate-600 font-mono text-xs">→</span>

          {/* Step 2: Configure */}
          <button
            onClick={() => setStep(2)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
              step === 2 ? 'bg-blue-600 text-white shadow-md' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <span className="w-5 h-5 rounded-full bg-slate-800 text-[11px] font-bold flex items-center justify-center">
              2
            </span>
            <div className="flex flex-col text-left">
              <span className="font-semibold leading-tight">Configure</span>
              <span className="text-[9px] text-slate-400 font-mono">Domain & settings</span>
            </div>
          </button>

          <span className="text-slate-600 font-mono text-xs">→</span>

          {/* Step 3: Deploy */}
          <button
            onClick={() => setStep(3)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-medium transition-all ${
              step === 3 ? 'bg-blue-600 text-white shadow-md' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <span className="w-5 h-5 rounded-full bg-slate-800 text-[11px] font-bold flex items-center justify-center">
              3
            </span>
            <div className="flex flex-col text-left">
              <span className="font-semibold leading-tight">Deploy</span>
              <span className="text-[9px] text-slate-400 font-mono">Review & launch</span>
            </div>
          </button>
        </div>
      </div>

      {/* STEP 1: SOURCE SELECTION */}
      {step === 1 && (
        <div className="max-w-4xl mx-auto space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div
              onClick={() => setStep(2)}
              className="p-5 rounded-2xl bg-[#0D1527] border border-blue-500/50 hover:border-blue-400 cursor-pointer transition-all space-y-3 group"
            >
              <div className="w-10 h-10 rounded-xl bg-blue-600/20 text-cyan-400 flex items-center justify-center">
                <Globe className="w-5 h-5" />
              </div>
              <div className="font-bold text-white text-sm">Git Repository (Recommended)</div>
              <p className="text-xs text-slate-400">
                Connect GitHub or GitLab repository with automated cryptographic signing on commit.
              </p>
              <CapabilityBadge state="LIVE" />
            </div>

            <div
              onClick={() => setStep(2)}
              className="p-5 rounded-2xl bg-[#0D1527] border border-slate-800 hover:border-slate-700 cursor-pointer transition-all space-y-3"
            >
              <div className="w-10 h-10 rounded-xl bg-purple-600/20 text-purple-400 flex items-center justify-center">
                <Layers className="w-5 h-5" />
              </div>
              <div className="font-bold text-white text-sm">Docker Container Image</div>
              <p className="text-xs text-slate-400">
                Pull pre-built OCI container image with digest verification.
              </p>
              <CapabilityBadge state="LIVE" />
            </div>

            <div
              onClick={() => setStep(2)}
              className="p-5 rounded-2xl bg-[#0D1527] border border-slate-800 hover:border-slate-700 cursor-pointer transition-all space-y-3"
            >
              <div className="w-10 h-10 rounded-xl bg-emerald-600/20 text-emerald-400 flex items-center justify-center">
                <FolderArchive className="w-5 h-5" />
              </div>
              <div className="font-bold text-white text-sm">Archive File Upload</div>
              <p className="text-xs text-slate-400">
                Upload .zip or tarball with path-traversal and hash validation.
              </p>
              <CapabilityBadge state="LIVE" />
            </div>
          </div>

          <div className="flex justify-end">
            <button
              onClick={() => setStep(2)}
              className="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs flex items-center gap-2"
            >
              <span>Continue to Configure</span>
              <ArrowRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      {/* STEP 2: CONFIGURE (Matches deploy.png exactly) */}
      {step === 2 && (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Left Column: Domain & Application Settings (4 cols) */}
          <div className="lg:col-span-4 space-y-5">
            <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
              <h2 className="text-sm font-bold text-white">Domain & Application Settings</h2>

              {/* Domain Input */}
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-300">Domain</label>
                <div className="relative">
                  <input
                    type="text"
                    value={domain}
                    onChange={(e) => setDomain(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-900 border border-slate-700/80 text-white text-xs font-mono focus:outline-none focus:border-cyan-400 pr-9"
                  />
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 absolute right-3 top-3" />
                </div>
                <div className="text-[11px] text-slate-400">
                  Use an existing domain or we can assign a temporary one.{' '}
                  <span className="text-cyan-400 hover:underline cursor-pointer" onClick={() => setDomain('mesh-app-9481.decentralized.host')}>
                    Use a temporary domain
                  </span>
                </div>
              </div>

              {/* Application Type */}
              <div className="space-y-2">
                <label className="text-xs font-semibold text-slate-300">Application Type</label>
                <div className="grid grid-cols-2 gap-2">
                  {[
                    { id: 'Static Site', label: 'Static Site', desc: 'HTML, CSS, JS' },
                    { id: 'Web App', label: 'Web App', desc: 'Node.js, Python, etc.' },
                    { id: 'WordPress', label: 'WordPress', desc: 'Latest & secure' },
                    { id: 'Docker App', label: 'Docker App', desc: 'Deploy from image' },
                    { id: 'Custom', label: 'Custom', desc: 'Advanced setup' }
                  ].map((type) => (
                    <button
                      key={type.id}
                      type="button"
                      onClick={() => setAppType(type.id as any)}
                      className={`p-3 rounded-xl border text-left transition-all ${
                        appType === type.id
                          ? 'bg-blue-600/20 border-blue-500 shadow-md ring-1 ring-blue-500'
                          : 'bg-slate-900/60 border-slate-800 hover:border-slate-700'
                      }`}
                    >
                      <div className="text-xs font-bold text-white">{type.label}</div>
                      <div className="text-[10px] text-slate-400 font-mono mt-0.5">{type.desc}</div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Build Settings (Optional) */}
              <div className="space-y-3 pt-3 border-t border-slate-800">
                <div className="flex items-center justify-between">
                  <label className="text-xs font-semibold text-slate-300">Build Settings (Optional)</label>
                  <button
                    onClick={() => {
                      setBuildCommand('npm run build');
                      setOutputDir('dist');
                    }}
                    className="text-[11px] font-mono text-cyan-400 hover:underline flex items-center gap-1"
                  >
                    <RefreshCw className="w-3 h-3" />
                    Auto-detect
                  </button>
                </div>

                <div className="space-y-1">
                  <span className="text-[11px] text-slate-400 font-mono">Build Command</span>
                  <input
                    type="text"
                    value={buildCommand}
                    onChange={(e) => setBuildCommand(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
                  />
                </div>

                <div className="space-y-1">
                  <span className="text-[11px] text-slate-400 font-mono">Output Directory</span>
                  <input
                    type="text"
                    value={outputDir}
                    onChange={(e) => setOutputDir(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
                  />
                </div>
              </div>

              {/* Environment Variables */}
              <div className="space-y-2 pt-3 border-t border-slate-800">
                <label className="text-xs font-semibold text-slate-300">Environment Variables (Optional)</label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder="e.g. API_URL"
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value)}
                    className="flex-1 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono text-white focus:outline-none"
                  />
                  <input
                    type="text"
                    placeholder="e.g. https://api..."
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                    className="flex-1 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono text-white focus:outline-none"
                  />
                  <button
                    onClick={handleAddEnv}
                    className="p-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white"
                  >
                    <Plus className="w-3.5 h-3.5" />
                  </button>
                </div>

                {envVars.map((env, i) => (
                  <div key={i} className="flex items-center justify-between p-2 rounded-lg bg-slate-900/60 border border-slate-800 text-xs font-mono">
                    <span className="text-cyan-400">{env.key}</span>
                    <span className="text-slate-400 truncate max-w-[140px]">{env.value}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Center Column: Global Node Selection (5 cols) */}
          <div className="lg:col-span-5 space-y-4">
            <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-bold text-white">Global Node Selection</h2>
                  <p className="text-xs text-slate-400">
                    Select regions to deploy your website. Your data will be distributed across independent nodes.
                  </p>
                </div>
                <div className="px-2.5 py-1 rounded-full bg-emerald-950/80 border border-emerald-500/40 text-[11px] font-mono text-emerald-400 font-semibold">
                  {totalSelectedRegionsCount} Regions Selected
                </div>
              </div>

              {/* Map */}
              <WorldMap
                heightClass="h-[240px]"
                showRegions={true}
                onSelectRegion={(reg) => toggleRegion(reg)}
              />

              {/* Region Cards Grid (matching deploy.png) */}
              <div className="grid grid-cols-3 gap-2.5 pt-2">
                {[
                  { name: 'North America', nodes: '8/8 nodes', color: '#10B981' },
                  { name: 'Europe', nodes: '12/12 nodes', color: '#3B82F6' },
                  { name: 'Asia', nodes: '10/10 nodes', color: '#A855F7' },
                  { name: 'South America', nodes: '4/4 nodes', color: '#06B6D4' },
                  { name: 'Africa', nodes: '3/3 nodes', color: '#10B981' },
                  { name: 'Australia', nodes: '6/6 nodes', color: '#F59E0B' }
                ].map((reg) => {
                  const isChecked = selectedRegions[reg.name] ?? false;
                  return (
                    <div
                      key={reg.name}
                      onClick={() => toggleRegion(reg.name)}
                      className={`p-3 rounded-xl border cursor-pointer transition-all ${
                        isChecked
                          ? 'bg-slate-900/90 border-blue-500 shadow-md'
                          : 'bg-slate-950/40 border-slate-800 opacity-60'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-white">{reg.name}</span>
                        <input
                          type="checkbox"
                          checked={isChecked}
                          onChange={() => {}}
                          className="rounded text-blue-600 focus:ring-0"
                        />
                      </div>
                      <div className="text-[10px] font-mono text-slate-400 mt-1">{reg.nodes}</div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Bottom step buttons matching deploy.png */}
            <div className="flex items-center justify-between pt-2">
              <button
                onClick={() => setStep(1)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold flex items-center gap-1.5"
              >
                <ArrowLeft className="w-3.5 h-3.5" />
                <span>Back</span>
              </button>
              <div className="text-xs font-mono text-slate-500">Step 2 of 3</div>
              <button
                onClick={() => setStep(3)}
                className="px-5 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/30 flex items-center gap-2"
              >
                <span>Continue to Review</span>
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Right Column: Deployment Summary & Advanced Options (3 cols) */}
          <div className="lg:col-span-3 space-y-5">
            {/* Deployment Summary */}
            <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-bold text-white">Deployment Summary</h2>
                <button onClick={() => setStep(1)} className="text-[11px] font-semibold text-cyan-400 hover:underline">
                  Edit
                </button>
              </div>

              <div className="p-3 rounded-xl bg-slate-900 border border-slate-800">
                <div className="text-xs font-bold text-white">My Website</div>
                <div className="text-[10px] text-slate-400 font-mono">Static Site (HTML)</div>
              </div>

              <div className="space-y-2.5 text-xs font-mono">
                <div className="flex justify-between">
                  <span className="text-slate-400">Domain</span>
                  <span className="text-cyan-400 font-semibold">{domain}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Regions</span>
                  <span className="text-white">{totalSelectedRegionsCount} regions (43 nodes)</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Estimated Size</span>
                  <span className="text-white">24.8 MB</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Build Command</span>
                  <span className="text-slate-300">{buildCommand}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Output Directory</span>
                  <span className="text-slate-300">{outputDir}</span>
                </div>
              </div>
            </div>

            {/* Advanced Options Toggles matching deploy.png */}
            <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3.5">
              <h2 className="text-sm font-bold text-white">Advanced Options</h2>

              {[
                { label: 'Auto SSL', desc: 'Free SSL certificate for all nodes', val: autoSSL, set: setAutoSSL },
                { label: 'Global CDN', desc: 'Enable distributed edge delivery', val: globalCDN, set: setGlobalCDN },
                { label: 'Auto Backup', desc: 'Daily backups across multiple nodes', val: autoBackup, set: setAutoBackup },
                { label: 'DDoS Protection', desc: 'Built-in network protection', val: ddosProtection, set: setDdosProtection },
                { label: 'High Availability', desc: 'Replicate across independent nodes', val: highAvailability, set: setHighAvailability }
              ].map((opt) => (
                <div key={opt.label} className="flex items-center justify-between">
                  <div className="flex flex-col">
                    <span className="text-xs font-semibold text-white">{opt.label}</span>
                    <span className="text-[10px] text-slate-400">{opt.desc}</span>
                  </div>
                  <button
                    onClick={() => opt.set(!opt.val)}
                    className={`w-9 h-5 rounded-full transition-colors relative ${
                      opt.val ? 'bg-blue-600' : 'bg-slate-800'
                    }`}
                  >
                    <span
                      className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                        opt.val ? 'right-0.5' : 'left-0.5'
                      }`}
                    />
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* STEP 3: REVIEW & LAUNCH */}
      {step === 3 && (
        <div className="max-w-3xl mx-auto space-y-6">
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800 p-6 space-y-5">
            <div className="flex items-center justify-between border-b border-slate-800 pb-4">
              <div>
                <h2 className="text-lg font-bold text-white">Immutable Deployment Review</h2>
                <p className="text-xs text-slate-400">
                  Verify the deployment parameters before mutating the decentralized mesh.
                </p>
              </div>
              <CapabilityBadge state="LIVE" />
            </div>

            <div className="space-y-3 font-mono text-xs">
              <div className="flex justify-between p-3 rounded-xl bg-slate-900 border border-slate-800">
                <span className="text-slate-400">Target Domain:</span>
                <span className="text-cyan-400 font-bold">{domain}</span>
              </div>
              <div className="flex justify-between p-3 rounded-xl bg-slate-900 border border-slate-800">
                <span className="text-slate-400">Application Type:</span>
                <span className="text-white">{appType}</span>
              </div>
              <div className="flex justify-between p-3 rounded-xl bg-slate-900 border border-slate-800">
                <span className="text-slate-400">Allocated Placement:</span>
                <span className="text-emerald-400 font-bold">{totalSelectedRegionsCount} Regions (43 Nodes)</span>
              </div>
              <div className="flex justify-between p-3 rounded-xl bg-slate-900 border border-slate-800">
                <span className="text-slate-400">Replication Quorum:</span>
                <span className="text-purple-400">3× Geo-Redundant</span>
              </div>
              <div className="flex justify-between p-3 rounded-xl bg-slate-900 border border-slate-800">
                <span className="text-slate-400">Auto SSL / ACME:</span>
                <span className="text-emerald-400">Let's Encrypt TLS 1.3</span>
              </div>
            </div>

            <div className="flex items-center justify-between pt-4 border-t border-slate-800">
              <button
                onClick={() => setStep(2)}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
              >
                Back to Configure
              </button>

              <button
                onClick={handleLaunch}
                className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:opacity-90 text-white text-xs font-bold shadow-xl shadow-blue-600/30 flex items-center gap-2"
              >
                <Rocket className="w-4 h-4" />
                <span>Launch Deployment to Mesh</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Deployment Live Progress Modal */}
      {isDeploying && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4">
          <div className="w-full max-w-xl rounded-2xl bg-[#0B1120] border border-slate-700 shadow-2xl p-6 space-y-5 animate-in fade-in zoom-in-95">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-lg bg-blue-600/20 text-cyan-400 flex items-center justify-center">
                  <Rocket className="w-4 h-4 animate-pulse" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-white">Deploying to Mesh</h3>
                  <span className="text-[11px] font-mono text-cyan-400">{domain}</span>
                </div>
              </div>

              {deploymentResult && (
                <span className="px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-500/40 text-xs font-mono">
                  VERIFIED READY
                </span>
              )}
            </div>

            {/* Stages Timeline */}
            <div className="space-y-2 font-mono text-xs">
              {[
                { stage: 'VALIDATING', label: '1. Validating package signature & dependencies' },
                { stage: 'SCHEDULING', label: '2. Scheduling 43 validator nodes across 6 regions' },
                { stage: 'ARTIFACT_TRANSFER', label: '3. Distributing IPFS Merkle blocks (3× quorum)' },
                { stage: 'RUNTIME_CREATION', label: '4. Provisioning micro-VM runtimes' },
                { stage: 'HEALTH_CHECK', label: '5. Running synthetic health & latency checks' },
                { stage: 'ROUTING', label: '6. Binding Anycast edge DNS & ACME SSL' },
                { stage: 'VERIFIED', label: '7. Sealed into Cryptographic Evidence Ledger' }
              ].map((item, idx) => {
                const stages = [
                  'VALIDATING',
                  'SCHEDULING',
                  'ARTIFACT_TRANSFER',
                  'RUNTIME_CREATION',
                  'HEALTH_CHECK',
                  'ROUTING',
                  'VERIFIED'
                ];
                const currentIndex = stages.indexOf(activeDeployStage);
                const isPassed = currentIndex >= idx;
                const isCurrent = activeDeployStage === item.stage && activeDeployStage !== 'VERIFIED';

                return (
                  <div
                    key={item.stage}
                    className={`flex items-center justify-between p-2 rounded-lg border transition-all ${
                      isPassed
                        ? 'bg-slate-900 border-emerald-500/40 text-slate-200'
                        : 'bg-slate-950/40 border-slate-900 text-slate-600'
                    }`}
                  >
                    <span className="flex items-center gap-2">
                      {isPassed ? (
                        <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                      ) : isCurrent ? (
                        <RefreshCw className="w-3.5 h-3.5 animate-spin text-cyan-400" />
                      ) : (
                        <span className="w-3.5 h-3.5 rounded-full border border-slate-700 inline-block" />
                      )}
                      <span>{item.label}</span>
                    </span>
                    <span className="text-[10px] text-slate-400">
                      {isPassed ? 'PASSED' : isCurrent ? 'RUNNING' : 'PENDING'}
                    </span>
                  </div>
                );
              })}
            </div>

            {deploymentResult && (
              <div className="pt-3 border-t border-slate-800 flex items-center justify-between">
                <div className="text-[11px] font-mono text-slate-400">
                  Evidence ID: <span className="text-cyan-400">{deploymentResult.evidence.recordId}</span>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      setIsDeploying(false);
                      onNavigate('apps');
                    }}
                    className="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs"
                  >
                    View in Applications
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
