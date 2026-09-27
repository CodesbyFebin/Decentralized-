import React, { useState } from 'react';
import {
  X,
  Server,
  Laptop,
  Home,
  Cloud,
  Cpu,
  Boxes,
  Terminal,
  Copy,
  Check,
  Shield,
  Sliders,
  ChevronRight,
  Sparkles,
  CheckCircle2,
  AlertCircle
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';

interface AddNodeModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AddNodeModal: React.FC<AddNodeModalProps> = ({ isOpen, onClose }) => {
  const { addNode } = useNetwork();
  const [selectedType, setSelectedType] = useState<string>('linux');
  const [nodeName, setNodeName] = useState('home-server-mini');
  const [copied, setCopied] = useState(false);
  const [enrollmentStep, setEnrollmentStep] = useState<'configure' | 'probing' | 'ready'>('configure');
  const [ownerReserveCpu, setOwnerReserveCpu] = useState(2);
  const [ownerReserveRam, setOwnerReserveRam] = useState(4);
  const [totalCpu, setTotalCpu] = useState(8);
  const [totalRam, setTotalRam] = useState(16);
  const [totalStorage, setTotalStorage] = useState(500);

  if (!isOpen) return null;

  const nodeTypes = [
    { id: 'this-pc', label: 'This Computer', desc: 'Install on your current machine', icon: Laptop },
    { id: 'linux', label: 'Linux Server', desc: 'Ubuntu, Debian, Arch, CentOS, etc.', icon: Server },
    { id: 'home-server', label: 'Home Server', desc: 'Your on-premise hardware / homelab', icon: Home },
    { id: 'vps', label: 'VPS', desc: 'DigitalOcean, Hetzner, Linode, AWS EC2', icon: Cloud },
    { id: 'arm', label: 'Raspberry Pi / ARM', desc: 'ARM devices and SBCs (64-bit)', icon: Cpu },
    { id: 'gpu', label: 'GPU Machine', desc: 'NVIDIA RTX, AMD ROCm, Apple Silicon', icon: Boxes },
  ];

  const installCommand = `curl -sSL https://decentralized.host/install.sh | bash -s -- --token=dhp_token_8a92f0c1 --reserve-cpu=${ownerReserveCpu} --reserve-ram=${ownerReserveRam}`;

  const handleCopy = () => {
    navigator.clipboard.writeText(installCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 3000);
  };

  const handleSimulateProbe = () => {
    setEnrollmentStep('probing');
    setTimeout(() => {
      setEnrollmentStep('ready');
    }, 1800);
  };

  const handleCompleteEnrollment = () => {
    addNode({
      name: nodeName,
      cpu: {
        total: totalCpu,
        ownerReserve: ownerReserveCpu,
        marketplaceReserved: 2,
        allocated: 0,
        available: totalCpu - ownerReserveCpu - 2,
      },
      memoryGb: {
        total: totalRam,
        ownerReserve: ownerReserveRam,
        marketplaceReserved: 4,
        allocated: 0,
        available: totalRam - ownerReserveRam - 4,
      },
      storageGb: {
        total: totalStorage,
        ownerReserve: 100,
        marketplaceReserved: 100,
        allocated: 0,
        available: totalStorage - 200,
      },
      roles: selectedType === 'gpu' ? ['Compute', 'GPU'] : ['Private Host', 'Edge'],
      trustClass: 'OWNER',
      isolation: 'rootless',
    });
    onClose();
    setEnrollmentStep('configure');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-3xl bg-gradient-to-b from-[#0d1838] to-[#070d1e] border border-blue-500/30 p-6 shadow-2xl custom-scrollbar text-slate-200">
        {/* Close Button */}
        <button
          onClick={onClose}
          className="absolute top-5 right-5 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/60 border border-slate-700/50 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Title */}
        <div className="flex items-center gap-3 mb-5">
          <div className="w-10 h-10 rounded-2xl bg-cyan-500/20 border border-cyan-400/40 flex items-center justify-center text-cyan-300 shadow-lg shadow-cyan-500/20">
            <Server className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-white tracking-tight">
              Add a Node: Turn any machine into a node
            </h2>
            <p className="text-xs text-slate-400">
              Sovereign enrollment with cryptographic Ed25519 node identity & strict owner reserves.
            </p>
          </div>
        </div>

        {enrollmentStep === 'configure' && (
          <div className="space-y-5">
            {/* Machine Type Selector */}
            <div>
              <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-2">
                1. Select Machine Architecture
              </label>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                {nodeTypes.map((type) => {
                  const Icon = type.icon;
                  const isSelected = selectedType === type.id;
                  return (
                    <button
                      key={type.id}
                      onClick={() => setSelectedType(type.id)}
                      className={`p-3 rounded-2xl text-left border transition-all flex flex-col gap-2 ${
                        isSelected
                          ? 'bg-blue-600/20 border-cyan-400 text-white shadow-lg shadow-cyan-500/10'
                          : 'bg-slate-900/50 border-slate-800 text-slate-400 hover:border-slate-700 hover:text-slate-200'
                      }`}
                    >
                      <Icon className={`w-5 h-5 ${isSelected ? 'text-cyan-400' : 'text-slate-500'}`} />
                      <div>
                        <p className="text-xs font-semibold leading-tight">{type.label}</p>
                        <p className="text-[10px] text-slate-500 mt-0.5 line-clamp-1">{type.desc}</p>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Node Identifier */}
            <div>
              <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider block mb-1.5">
                2. Node Hostname / Label
              </label>
              <input
                type="text"
                value={nodeName}
                onChange={(e) => setNodeName(e.target.value)}
                placeholder="e.g. homelab-ryzen, vps-frankfurt-01"
                className="w-full px-3.5 py-2.5 rounded-xl bg-slate-900/80 border border-blue-500/30 text-white text-sm focus:outline-none focus:border-cyan-400 font-mono"
              />
            </div>

            {/* Owner Reserve Policy Slider (Constitution Requirement) */}
            <div className="p-4 rounded-2xl bg-blue-950/30 border border-blue-500/25 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-cyan-300 uppercase tracking-wider flex items-center gap-1.5">
                  <Shield className="w-3.5 h-3.5" />
                  Owner Reserve Policy (Constitution Invariant)
                </span>
                <span className="text-[10px] font-mono text-cyan-400 bg-cyan-500/10 px-2 py-0.5 rounded-full border border-cyan-500/20">
                  Owner Always Wins
                </span>
              </div>
              <p className="text-[11px] text-slate-400">
                Capacity locked for your exclusive self-hosting. Marketplace and community workloads will never be scheduled on this reserve.
              </p>

              <div className="grid grid-cols-2 gap-4 pt-1">
                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-slate-300">Reserved CPU:</span>
                    <span className="font-mono font-bold text-cyan-300">{ownerReserveCpu} Cores</span>
                  </div>
                  <input
                    type="range"
                    min="1"
                    max={totalCpu - 2}
                    value={ownerReserveCpu}
                    onChange={(e) => setOwnerReserveCpu(Number(e.target.value))}
                    className="w-full accent-cyan-400"
                  />
                </div>

                <div>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-slate-300">Reserved RAM:</span>
                    <span className="font-mono font-bold text-purple-300">{ownerReserveRam} GB</span>
                  </div>
                  <input
                    type="range"
                    min="2"
                    max={totalRam - 4}
                    value={ownerReserveRam}
                    onChange={(e) => setOwnerReserveRam(Number(e.target.value))}
                    className="w-full accent-purple-400"
                  />
                </div>
              </div>
            </div>

            {/* One-Line Install Script */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
                  <Terminal className="w-3.5 h-3.5 text-cyan-400" />
                  3. Run Sovereign dh-agent Command
                </label>
                <span className="text-[11px] text-slate-500">Rootless or root supported</span>
              </div>

              <div className="relative p-3.5 rounded-xl bg-black/70 border border-slate-700/80 font-mono text-xs text-cyan-300 group">
                <p className="break-all pr-12">{installCommand}</p>
                <button
                  onClick={handleCopy}
                  className="absolute top-2.5 right-2.5 p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-all shadow-md"
                  title="Copy command"
                >
                  {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="pt-2 flex items-center justify-between">
              <span className="text-xs text-slate-500">
                Zero external dependencies. Private WireGuard mesh automatically configured.
              </span>
              <button
                onClick={handleSimulateProbe}
                className="px-5 py-2.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 shadow-lg shadow-cyan-500/25 transition-all flex items-center gap-2"
              >
                <span>Verify & Enroll Node</span>
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}

        {enrollmentStep === 'probing' && (
          <div className="py-12 flex flex-col items-center justify-center text-center space-y-4">
            <div className="relative">
              <div className="w-16 h-16 rounded-full border-2 border-cyan-400/20 border-t-cyan-400 animate-spin" />
              <div className="absolute inset-0 flex items-center justify-center">
                <Sparkles className="w-6 h-6 text-cyan-300 animate-pulse" />
              </div>
            </div>

            <div className="space-y-1">
              <h3 className="text-base font-bold text-white">Running Sovereign Hardware Probe...</h3>
              <p className="text-xs text-slate-400 font-mono">
                CHALLENGE_SENT → IDENTITY_VERIFIED → CAPABILITY_PROBED
              </p>
            </div>

            <div className="w-full max-w-sm p-3 rounded-xl bg-slate-900/80 border border-slate-800 font-mono text-[11px] text-left text-slate-400 space-y-1">
              <p className="text-emerald-400">✓ Ed25519 Cryptographic Handshake Established</p>
              <p className="text-cyan-400">✓ Kernel Cgroups v2 & Rootless OCI Runtime Detected</p>
              <p className="text-purple-400">✓ Owner Reserve Policy Committed: {ownerReserveCpu} CPU / {ownerReserveRam} GB RAM</p>
              <p className="text-slate-300">✓ WireGuard Peer 100.64.0.99 Assigned</p>
            </div>
          </div>
        )}

        {enrollmentStep === 'ready' && (
          <div className="py-8 flex flex-col items-center justify-center text-center space-y-5">
            <div className="w-14 h-14 rounded-2xl bg-emerald-500/20 border border-emerald-400/40 flex items-center justify-center text-emerald-400 shadow-xl shadow-emerald-500/20">
              <CheckCircle2 className="w-8 h-8" />
            </div>

            <div className="space-y-1">
              <h3 className="text-base font-bold text-white">
                Node '{nodeName}' Successfully Qualified!
              </h3>
              <p className="text-xs text-slate-400">
                This node is now active in your sovereign resource ledger with status ACTIVE.
              </p>
            </div>

            <div className="w-full p-4 rounded-2xl bg-slate-900/90 border border-blue-500/25 grid grid-cols-3 gap-2 text-center font-mono">
              <div className="p-2 rounded-xl bg-slate-800/50">
                <span className="text-[10px] text-slate-400 block">Total Cores</span>
                <span className="text-sm font-bold text-cyan-300">{totalCpu} CPU</span>
              </div>
              <div className="p-2 rounded-xl bg-slate-800/50">
                <span className="text-[10px] text-slate-400 block">Total Memory</span>
                <span className="text-sm font-bold text-purple-300">{totalRam} GB</span>
              </div>
              <div className="p-2 rounded-xl bg-slate-800/50">
                <span className="text-[10px] text-slate-400 block">Isolated Storage</span>
                <span className="text-sm font-bold text-emerald-300">{totalStorage} GB</span>
              </div>
            </div>

            <button
              onClick={handleCompleteEnrollment}
              className="w-full py-3 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-emerald-600 via-teal-600 to-cyan-500 hover:from-emerald-500 hover:to-cyan-400 shadow-lg shadow-emerald-500/25 transition-all"
            >
              Add Node to Mesh Fleet & Return to Dashboard
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
