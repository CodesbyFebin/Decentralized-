import React, { useState, useRef, useEffect } from 'react';
import {
  Bot,
  Sparkles,
  Send,
  Cpu,
  Shield,
  Activity,
  Layers,
  ArrowRight,
  CheckCircle2,
  AlertTriangle,
  RotateCw,
  Terminal,
  Zap,
  Sliders,
  BrainCircuit
} from 'lucide-react';
import { useNetwork } from '../context/NetworkContext';
import { CopilotMessage } from '../types';

export const RagCopilot: React.FC = () => {
  const {
    nodes,
    deployments,
    storageNodes,
    reconcileDeployment,
    cordonNode,
    addToast,
  } = useNetwork();

  const [messages, setMessages] = useState<CopilotMessage[]>([
    {
      id: 'msg-1',
      sender: 'copilot',
      text: `### Welcome to Decentralized.Host Infrastructure Intelligence\n\nI am your sovereign cluster copilot, grounded in the **Master Blueprint constitution**:\n- **Node ownership is the foundation** — your machine, your rules.\n- **Desired state is not observed state** — 3 replicas on 1 machine is not decentralization.\n- **Owner Reserve always wins** over third-party marketplace demand.\n\nSelect a mode below or ask me any question about your nodes, deployments, or placement plans.`,
      timestamp: 'Just now',
      mode: 'ASK',
      model: 'gemini-3.5-flash',
    },
  ]);

  const [input, setInput] = useState('');
  const [activeMode, setActiveMode] = useState<'ASK' | 'DIAGNOSE' | 'PLAN' | 'ACT'>('ASK');
  const [selectedModel, setSelectedModel] = useState<'gemini-3.1-pro-preview' | 'gemini-3.5-flash' | 'gemini-3.1-flash-lite'>('gemini-3.1-pro-preview');
  const [isHighThinking, setIsHighThinking] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  const handleSendMessage = async (promptText?: string) => {
    const query = promptText || input;
    if (!query.trim() || isLoading) return;

    const userMessage: CopilotMessage = {
      id: `user-${Date.now()}`,
      sender: 'user',
      text: query,
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      mode: activeMode,
    };

    setMessages((prev) => [...prev, userMessage]);
    if (!promptText) setInput('');
    setIsLoading(true);

    try {
      const response = await fetch('/api/copilot/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt: query,
          mode: activeMode,
          model: selectedModel,
          thinking: isHighThinking,
          context: {
            nodesOnline: `${nodes.filter((n) => n.status === 'Online').length}/${nodes.length}`,
            appCount: deployments.length,
            storageUsed: '428 GB / 2.4 TB',
            alerts: 'Node raspberry-pi offline (2d ago), App rag-copilot degraded (1/2 replicas)',
          },
        }),
      });

      const data = await response.json();

      let actionProposal;
      if (activeMode === 'ACT' || query.toLowerCase().includes('reconcile') || query.toLowerCase().includes('cordon')) {
        actionProposal = {
          type: 'RECONCILE_REPLICA',
          title: 'Schedule Replacement Replica on gpu-workstation',
          details: 'Migrates missing replica of rag-copilot from raspberry-pi (OFFLINE) to gpu-workstation (32GB RAM free). Retains 4 core owner reserve.',
          executed: false,
        };
      }

      const copilotResponse: CopilotMessage = {
        id: `copilot-${Date.now()}`,
        sender: 'copilot',
        text: data.text || 'Diagnostic completed with no further actions needed.',
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        mode: activeMode,
        model: data.model || selectedModel,
        thinking: isHighThinking,
        actionProposal,
      };

      setMessages((prev) => [...prev, copilotResponse]);
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        {
          id: `err-${Date.now()}`,
          sender: 'copilot',
          text: `Error connecting to Gemini API: ${err.message || 'Unknown network error'}. Operating in autonomous local mode.`,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          mode: activeMode,
        },
      ]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleExecuteProposal = (msgId: string) => {
    reconcileDeployment('dep-04');
    setMessages((prev) =>
      prev.map((m) => {
        if (m.id === msgId && m.actionProposal) {
          return {
            ...m,
            actionProposal: { ...m.actionProposal, executed: true },
          };
        }
        return m;
      })
    );
    addToast('Action executed: Replacement replica running on gpu-workstation!', 'success');
  };

  const presetQueries = [
    { mode: 'DIAGNOSE', label: 'Why is rag-copilot degraded?', prompt: 'Analyze why application rag-copilot is degraded and inspect node failures.' },
    { mode: 'PLAN', label: 'Plan replica migration for offline node', prompt: 'Formulate a zero-downtime placement plan to migrate replicas from raspberry-pi.' },
    { mode: 'ASK', label: 'Audit resource ledger availability', prompt: 'Calculate the total unreserved capacity across all 4 owner nodes according to the Master Blueprint.' },
    { mode: 'ACT', label: 'Auto-reconcile degraded workloads', prompt: 'Propose and execute an atomic reconciliation action for degraded workloads.' },
  ];

  return (
    <div className="h-[calc(100vh-6.5rem)] flex flex-col space-y-4 animate-in fade-in duration-200">
      {/* Top Banner / Controls */}
      <div className="p-4 rounded-3xl bg-gradient-to-r from-[#0c183a] via-[#111f4d] to-[#0a122c] border border-blue-500/25 shadow-xl backdrop-blur-xl flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="relative w-10 h-10 rounded-2xl bg-gradient-to-tr from-purple-600 via-indigo-600 to-cyan-400 p-[1px] shadow-lg shadow-purple-500/30">
            <div className="w-full h-full bg-[#091124] rounded-2xl flex items-center justify-center">
              <Bot className="w-5 h-5 text-cyan-300 animate-pulse" />
            </div>
          </div>
          <div>
            <h2 className="text-base font-bold text-white tracking-tight flex items-center gap-2">
              RAG Copilot: Infrastructure Intelligence
              <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-purple-500/20 text-purple-300 border border-purple-500/30">
                Grounded in Cluster Telemetry
              </span>
            </h2>
            <p className="text-xs text-slate-400">
              Autonomous reasoning & safe policy-evaluated execution across nodes, storage, and DePIN.
            </p>
          </div>
        </div>

        {/* Mode Selector & Model Config */}
        <div className="flex flex-wrap items-center gap-2">
          {/* Operational Modes matching spec */}
          <div className="flex items-center gap-1 p-1 rounded-xl bg-slate-900/80 border border-slate-800 text-xs">
            {(['ASK', 'DIAGNOSE', 'PLAN', 'ACT'] as const).map((m) => (
              <button
                key={m}
                onClick={() => {
                  setActiveMode(m);
                  if (m === 'PLAN' || m === 'DIAGNOSE') {
                    setSelectedModel('gemini-3.1-pro-preview');
                    setIsHighThinking(true);
                  }
                }}
                className={`px-2.5 py-1 rounded-lg font-semibold transition-all ${
                  activeMode === m
                    ? m === 'ACT'
                      ? 'bg-rose-600 text-white shadow-md'
                      : m === 'PLAN'
                      ? 'bg-purple-600 text-white shadow-md'
                      : m === 'DIAGNOSE'
                      ? 'bg-amber-600 text-white shadow-md'
                      : 'bg-cyan-600 text-white shadow-md'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                {m}
              </button>
            ))}
          </div>

          {/* High Thinking Toggle (Required feature) */}
          <button
            onClick={() => {
              setIsHighThinking(!isHighThinking);
              if (!isHighThinking) setSelectedModel('gemini-3.1-pro-preview');
            }}
            className={`px-3 py-1.5 rounded-xl text-xs font-mono font-bold border transition-all flex items-center gap-1.5 ${
              isHighThinking
                ? 'bg-purple-600/30 text-purple-300 border-purple-500/60 shadow-lg shadow-purple-600/20'
                : 'bg-slate-900/60 text-slate-400 border-slate-800 hover:text-slate-200'
            }`}
            title="Enable High Thinking Mode (gemini-3.1-pro-preview with ThinkingLevel.HIGH)"
          >
            <BrainCircuit className={`w-3.5 h-3.5 ${isHighThinking ? 'text-purple-300 animate-pulse' : 'text-slate-500'}`} />
            <span>High Thinking</span>
            {isHighThinking && <span className="w-1.5 h-1.5 rounded-full bg-purple-400" />}
          </button>

          {/* Model Selector */}
          <select
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value as any)}
            className="px-2.5 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs font-mono text-cyan-300 focus:outline-none"
          >
            <option value="gemini-3.1-pro-preview">gemini-3.1-pro-preview (Complex & Thinking)</option>
            <option value="gemini-3.5-flash">gemini-3.5-flash (General)</option>
            <option value="gemini-3.1-flash-lite">gemini-3.1-flash-lite (Fast)</option>
          </select>
        </div>
      </div>

      {/* Messages Feed */}
      <div className="flex-1 overflow-y-auto p-4 rounded-3xl bg-[#091124]/90 border border-blue-500/25 shadow-2xl backdrop-blur-xl custom-scrollbar space-y-4">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex flex-col ${msg.sender === 'user' ? 'items-end' : 'items-start'}`}
          >
            <div
              className={`max-w-3xl rounded-2xl p-4 shadow-xl text-xs ${
                msg.sender === 'user'
                  ? 'bg-gradient-to-r from-blue-600 to-cyan-600 text-white font-medium'
                  : 'bg-slate-900/90 border border-blue-500/30 text-slate-200'
              }`}
            >
              {/* Header on copilot response */}
              {msg.sender === 'copilot' && (
                <div className="flex items-center justify-between pb-2 mb-2 border-b border-slate-800 text-[11px] font-mono text-slate-400">
                  <div className="flex items-center gap-2">
                    <span className="text-cyan-400 font-bold flex items-center gap-1">
                      <Sparkles className="w-3 h-3" /> Copilot
                    </span>
                    <span>•</span>
                    <span className="text-purple-300">{msg.model}</span>
                    {msg.thinking && (
                      <span className="px-1.5 py-0.2 rounded bg-purple-500/20 text-purple-200 text-[10px] border border-purple-500/30">
                        ThinkingLevel.HIGH
                      </span>
                    )}
                  </div>
                  <span className="text-[10px] text-slate-500">{msg.timestamp}</span>
                </div>
              )}

              {/* Message text formatted with markdown parsing simulation */}
              <div className="prose prose-invert prose-xs max-w-none space-y-2 whitespace-pre-wrap font-sans leading-relaxed">
                {msg.text}
              </div>

              {/* Executable Action Proposal card if present (ACT Mode) */}
              {msg.actionProposal && (
                <div className="mt-4 p-3.5 rounded-2xl bg-blue-950/40 border border-cyan-400/40 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-cyan-300 uppercase tracking-wider flex items-center gap-1.5">
                      <Zap className="w-3.5 h-3.5 text-cyan-400" />
                      Policy-Evaluated Action Prepared
                    </span>
                    <span className="text-[10px] font-mono text-purple-300 bg-purple-500/20 px-2 py-0.5 rounded">
                      Operator Approval Required
                    </span>
                  </div>

                  <p className="text-xs font-semibold text-white">{msg.actionProposal.title}</p>
                  <p className="text-[11px] text-slate-300 font-mono">{msg.actionProposal.details}</p>

                  <div className="pt-2 flex justify-end">
                    {msg.actionProposal.executed ? (
                      <span className="text-xs font-bold text-emerald-400 flex items-center gap-1.5 font-mono">
                        <CheckCircle2 className="w-4 h-4" /> Executed & Sealed
                      </span>
                    ) : (
                      <button
                        onClick={() => handleExecuteProposal(msg.id)}
                        className="px-4 py-2 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-emerald-600 to-cyan-500 hover:from-emerald-500 hover:to-cyan-400 shadow-lg shadow-emerald-500/25 transition-all flex items-center gap-1.5"
                      >
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        <span>Approve & Execute Action</span>
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}

        {isLoading && (
          <div className="flex items-center gap-3 p-4 rounded-2xl bg-slate-900/80 border border-purple-500/30 text-xs text-purple-300 font-mono w-fit animate-pulse">
            <BrainCircuit className="w-4 h-4 text-purple-400 animate-spin" />
            <span>Reasoning over cluster telemetry with {selectedModel}...</span>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Suggested Quick Prompts */}
      <div className="flex items-center gap-2 overflow-x-auto pb-1 text-xs custom-scrollbar">
        {presetQueries.map((pq) => (
          <button
            key={pq.label}
            onClick={() => {
              setActiveMode(pq.mode as any);
              handleSendMessage(pq.prompt);
            }}
            className="shrink-0 px-3 py-1.5 rounded-xl bg-slate-900/70 hover:bg-slate-800 border border-slate-800 text-slate-300 hover:text-cyan-300 transition-colors flex items-center gap-1.5"
          >
            <Sparkles className="w-3 h-3 text-cyan-400" />
            <span>{pq.label}</span>
          </button>
        ))}
      </div>

      {/* Input Form */}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleSendMessage();
        }}
        className="flex items-center gap-2"
      >
        <div className="relative flex-1">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={`Ask in ${activeMode} mode (e.g. "Why is node raspberry-pi offline?", "Plan migration for app-1")...`}
            className="w-full px-4 py-3 rounded-2xl bg-slate-900/90 border border-blue-500/30 text-white placeholder-slate-500 text-xs sm:text-sm font-mono focus:outline-none focus:border-cyan-400 shadow-inner"
          />
        </div>

        <button
          type="submit"
          disabled={!input.trim() || isLoading}
          className="px-5 py-3 rounded-2xl font-bold text-xs text-white bg-gradient-to-r from-blue-600 via-indigo-600 to-cyan-500 hover:from-blue-500 hover:to-cyan-400 disabled:opacity-50 shadow-lg shadow-cyan-500/20 transition-all flex items-center gap-2"
        >
          <span>Submit</span>
          <Send className="w-3.5 h-3.5" />
        </button>
      </form>
    </div>
  );
};
