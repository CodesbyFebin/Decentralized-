import React, { useState } from 'react';
import {
  X,
  Rocket,
  Github,
  Box,
  UploadCloud,
  FileCode,
  Sparkles,
  Shield,
  Layers,
  ChevronRight,
  Globe,
  CheckCircle2,
  Lock
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

interface NewDeploymentModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const NewDeploymentModal: React.FC<NewDeploymentModalProps> = ({ isOpen, onClose }) => {
  const { addDeployment } = useNetwork();
  const [sourceType, setSourceType] = useState<'github' | 'docker' | 'upload' | 'template' | 'copilot'>('github');
  const [appName, setAppName] = useState('portfolio-site');
  const [repoUrl, setRepoUrl] = useState('https://github.com/kochie/portfolio-site');
  const [branch, setBranch] = useState('main');
  const [imageName, setImageName] = useState('ghcr.io/kochie/portfolio:latest');
  const [replicas, setReplicas] = useState(3);
  const [placementMode, setPlacementMode] = useState<'self' | 'edge' | 'community' | 'depin' | 'hybrid'>('self');
  const [failureDomains, setFailureDomains] = useState<string[]>(['machine', 'operator', 'network']);
  const [step, setStep] = useState<'configure' | 'building' | 'done'>('configure');

  if (!isOpen) return null;

  const handleDeploy = () => {
    setStep('building');
    setTimeout(() => {
      setStep('done');
    }, 2000);
  };

  const handleFinalize = () => {
    addDeployment({
      name: appName,
      source: sourceType === 'docker'
        ? { type: 'Docker', image: imageName, commitSha: '7f91a2e' }
        : { type: 'GitHub', repo: repoUrl.replace('https://github.com/', ''), branch, commitSha: '3c81e9b' },
      environment: 'Production',
      replicas: { desired: replicas, observed: replicas, healthy: replicas },
      placement: placementMode === 'hybrid'
        ? ['self', 'community', 'edge']
        : placementMode === 'edge'
        ? ['self', 'edge']
        : placementMode === 'community'
        ? ['community', 'edge']
        : placementMode === 'depin'
        ? ['depin', 'edge']
        : ['self'],
      domain: `https://${appName}.d.host`,
      failureDomains,
      trustClass: 'PRIVATE',
      ports: [80, 443],
    });
    onClose();
    setStep('configure');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-3xl bg-gradient-to-b from-[#0d1838] to-[#070d1e] border border-blue-500/30 p-6 shadow-2xl custom-scrollbar text-slate-200">
        <button
          onClick={onClose}
          className="absolute top-5 right-5 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-slate-700/50 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="flex items-center gap-3 mb-5">
          <div className="w-10 h-10 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 flex items-center justify-center text-cyan-300 shadow-lg shadow-cyan-500/20">
            <Rocket className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-white tracking-tight">
              Deploy Anywhere. Without Giving Up Control.
            </h2>
            <p className="text-xs text-slate-400">
              Portable deployment contract with supply chain attestation & failure domain awareness.
            </p>
          </div>
        </div>

        {step === 'configure' && (
          <div className="space-y-4">
            {/* Source Selection */}
            <div>
              <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-2">
                1. Select Source
              </label>
              <div className="grid grid-cols-3 gap-2">
                <button
                  onClick={() => setSourceType('github')}
                  className={`p-2.5 rounded-xl border text-xs font-semibold flex items-center gap-2 transition-all ${
                    sourceType === 'github'
                      ? 'bg-blue-600/20 border-cyan-400 text-white shadow-md'
                      : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:text-white'
                  }`}
                >
                  <Github className="w-4 h-4 text-cyan-400" />
                  <span>GitHub</span>
                </button>
                <button
                  onClick={() => setSourceType('docker')}
                  className={`p-2.5 rounded-xl border text-xs font-semibold flex items-center gap-2 transition-all ${
                    sourceType === 'docker'
                      ? 'bg-blue-600/20 border-cyan-400 text-white shadow-md'
                      : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:text-white'
                  }`}
                >
                  <Box className="w-4 h-4 text-purple-400" />
                  <span>OCI Container</span>
                </button>
                <button
                  onClick={() => setSourceType('template')}
                  className={`p-2.5 rounded-xl border text-xs font-semibold flex items-center gap-2 transition-all ${
                    sourceType === 'template'
                      ? 'bg-blue-600/20 border-cyan-400 text-white shadow-md'
                      : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:text-white'
                  }`}
                >
                  <FileCode className="w-4 h-4 text-emerald-400" />
                  <span>Template</span>
                </button>
              </div>
            </div>

            {/* Inputs based on source */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                  Application Name
                </label>
                <input
                  type="text"
                  value={appName}
                  onChange={(e) => setAppName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
                />
              </div>

              <div>
                <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                  Assigned Domain
                </label>
                <input
                  type="text"
                  disabled
                  value={`https://${appName}.d.host`}
                  className="w-full px-3 py-2 rounded-xl bg-slate-950/80 border border-slate-800 text-cyan-400 text-xs font-mono"
                />
              </div>
            </div>

            {sourceType === 'github' ? (
              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2">
                  <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                    Repository URL
                  </label>
                  <input
                    type="text"
                    value={repoUrl}
                    onChange={(e) => setRepoUrl(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
                  />
                </div>
                <div>
                  <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                    Branch
                  </label>
                  <input
                    type="text"
                    value={branch}
                    onChange={(e) => setBranch(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
                  />
                </div>
              </div>
            ) : (
              <div>
                <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1">
                  Docker Image URI
                </label>
                <input
                  type="text"
                  value={imageName}
                  onChange={(e) => setImageName(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-xs font-mono focus:outline-none focus:border-cyan-400"
                />
              </div>
            )}

            {/* Placement Mode Selection */}
            <div>
              <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-2">
                2. Deployment Mode
              </label>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {[
                  { id: 'self', label: 'Self Host', desc: 'Deploy to hardware you control' },
                  { id: 'edge', label: 'Edge Deploy', desc: 'Distribute across edge nodes' },
                  { id: 'hybrid', label: 'Hybrid Deploy', desc: 'Own hardware + Edge cache' },
                  { id: 'community', label: 'Community', desc: 'Verified independent operators' },
                  { id: 'depin', label: 'DePIN Deploy', desc: 'External decentralized compute' },
                ].map((mode) => (
                  <button
                    key={mode.id}
                    onClick={() => setPlacementMode(mode.id as any)}
                    className={`p-2.5 rounded-xl border text-left transition-all ${
                      placementMode === mode.id
                        ? 'bg-blue-600/20 border-cyan-400 text-white'
                        : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:text-white'
                    }`}
                  >
                    <p className="text-xs font-bold">{mode.label}</p>
                    <p className="text-[10px] text-slate-500 mt-0.5 line-clamp-1">{mode.desc}</p>
                  </button>
                ))}
              </div>
            </div>

            {/* Replicas & Failure Domains */}
            <div className="p-3.5 rounded-2xl bg-blue-950/25 border border-blue-500/20 space-y-2.5">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-white">Replicas & Failure Domains</span>
                <span className="text-xs font-mono font-bold text-cyan-400">{replicas} Replicas</span>
              </div>
              <input
                type="range"
                min="1"
                max="5"
                value={replicas}
                onChange={(e) => setReplicas(Number(e.target.value))}
                className="w-full accent-cyan-400"
              />
              <div className="flex items-center gap-2 pt-1">
                <span className="text-[11px] text-slate-400">Diversity Check:</span>
                {['node', 'operator', 'network / ASN'].map((domain) => (
                  <span
                    key={domain}
                    className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-cyan-500/10 text-cyan-300 border border-cyan-500/25"
                  >
                    ✓ {domain}
                  </span>
                ))}
              </div>
            </div>

            {/* Submit */}
            <div className="pt-2 flex items-center justify-between">
              <span className="text-xs text-slate-500">
                Produces immutable cryptographic SBOM digest & signed evidence.
              </span>
              <button
                onClick={handleDeploy}
                className="px-5 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/25 transition-all flex items-center gap-2"
              >
                <span>Trigger Build & Placement</span>
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}

        {step === 'building' && (
          <div className="py-10 text-center space-y-4">
            <div className="w-14 h-14 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 mx-auto flex items-center justify-center text-cyan-400 animate-pulse">
              <Layers className="w-7 h-7 animate-spin" />
            </div>
            <div className="space-y-1">
              <h3 className="text-base font-bold text-white">Executing Sovereign Build Pipeline</h3>
              <p className="text-xs text-slate-400 font-mono">
                SOURCE → RESOLVE → BUILD → TEST → SBOM → HASH → SIGN → ATTEST
              </p>
            </div>
            <div className="max-w-md mx-auto p-3 rounded-xl bg-black/60 border border-slate-800 text-[11px] font-mono text-left text-slate-300 space-y-1">
              <p className="text-emerald-400">✓ Source commit 3c81e9b fetched</p>
              <p className="text-cyan-400">✓ Zero-vulnerability hermetic container compiled</p>
              <p className="text-purple-400">✓ Immutable Artifact Digest: sha256:b36f4c9a8e01...</p>
              <p className="text-slate-400">✓ Placement verified on 3 independent failure domains</p>
            </div>
          </div>
        )}

        {step === 'done' && (
          <div className="py-8 text-center space-y-5">
            <div className="w-14 h-14 rounded-2xl bg-emerald-500/20 border border-emerald-400/40 mx-auto flex items-center justify-center text-emerald-400">
              <CheckCircle2 className="w-8 h-8" />
            </div>
            <div className="space-y-1">
              <h3 className="text-base font-bold text-white">Application '{appName}' Successfully Deployed!</h3>
              <p className="text-xs text-slate-400">
                All {replicas} replicas are active and receiving traffic through distributed Anycast edge ingress.
              </p>
            </div>

            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-blue-500/20 text-xs font-mono space-y-1">
              <p className="text-cyan-300">URL: https://{appName}.d.host</p>
              <p className="text-slate-400">TLS: Auto-provisioned ECC P-384 Certificate Active</p>
            </div>

            <button
              onClick={handleFinalize}
              className="w-full py-3 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-emerald-600 via-teal-600 to-cyan-500 hover:from-emerald-500 hover:to-cyan-400 shadow-lg shadow-emerald-500/25 transition-all"
            >
              Complete & View in Websites & Apps
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
