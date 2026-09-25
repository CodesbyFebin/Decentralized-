import React, { useState } from 'react';
import {
  Sparkles,
  Brain,
  Send,
  Copy,
  Check,
  FileText,
  AlertTriangle,
  Server,
  ShieldCheck,
  Layers,
  ArrowRight,
  ExternalLink,
  RefreshCw,
  Lock,
  ThumbsUp,
  ThumbsDown,
  Info
} from 'lucide-react';
import { api } from '../../lib/api';
import { CopilotMessage, ProposedAction } from '../../types/platform';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute, entityId?: string) => void;
}

export const CopilotView: React.FC<Props> = ({ onNavigate }) => {
  const [usePlatformContext, setUsePlatformContext] = useState(true);
  const [inputQuery, setInputQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [copiedCodeTab, setCopiedCodeTab] = useState<string | null>(null);
  const [activeCodeTabs, setActiveCodeTabs] = useState<Record<string, string>>({});

  // Seed conversation matching copiolt RAG.png
  const [messages, setMessages] = useState<CopilotMessage[]>([
    {
      id: 'msg-init-1',
      sender: 'assistant',
      timestamp: '10:24 AM',
      content: "Hi Febin! 👋\nI'm your RAG Copilot. I can help you with:\n\n• Deploy an application with verified best practices\n• Troubleshoot issues across your 43 mesh nodes\n• Set up custom Anycast domains and ACME certificates\n• Analyze usage, storage quorum, and cost projections\n• Propose safe, auditable operational remediations\n\nAsk me anything about your hosting infrastructure or pick a quick prompt."
    },
    {
      id: 'msg-user-1',
      sender: 'user',
      timestamp: '10:25 AM',
      content: 'Show me how to deploy a Next.js app and connect a custom domain with SSL.'
    },
    {
      id: 'msg-guide-1',
      sender: 'assistant',
      timestamp: '10:25 AM',
      content: "I'll help you deploy a Next.js app and connect a custom domain with SSL.\n\nHere's a step-by-step guide based on your platform and best practices:",
      stepGuide: {
        title: 'Deploy Next.js App with Custom Domain & SSL',
        steps: [
          { number: 1, label: 'Prepare App', desc: 'Build your Next.js project' },
          { number: 2, label: 'Deploy', desc: 'Upload or connect Git repo' },
          { number: 3, label: 'Configure Domain', desc: 'Point DNS to your app' },
          { number: 4, label: 'Enable SSL', desc: 'Automatic certificate provisioning' },
          { number: 5, label: 'Verify', desc: 'Test and monitor' }
        ],
        codeSnippets: [
          {
            tabName: 'Using Git (Recommended)',
            code: `# 1. Connect your GitHub repository\n# Go to Deploy > New Deployment > Connect GitHub\n# Select your repository and set build command:\nnpm run build\n# Set output directory:\n.next\n# 2. Configure environment variables (if needed)\n# 3. Deploy and wait for build to complete`
          },
          {
            tabName: 'Using Docker',
            code: `# Multi-stage Docker container\ndocker build -t registry.decentralized.host/my-next:v1 .\ndocker push registry.decentralized.host/my-next:v1\n# Deploy container via Command Centre`
          }
        ],
        docLinks: [
          { title: 'Deploy Next.js Application', path: 'platform/docs/nextjs-deploy' },
          { title: 'Custom Domain Setup', path: 'platform/docs/custom-domain' },
          { title: 'SSL Certificate Guide', path: 'platform/docs/ssl-certificates' }
        ]
      },
      citations: [
        {
          id: 'cit-doc-1',
          source: 'documentation',
          title: 'Deploy Next.js Application on Mesh',
          pathOrId: 'platform/docs/nextjs-deploy',
          snippet: 'Anycast ingress automatically proxies root domains to the nearest healthy validator micro-VM.'
        },
        {
          id: 'cit-obs-1',
          source: 'node_observation',
          title: 'Mesh Scheduler Quorum',
          pathOrId: 'cluster/topology',
          snippet: 'Requires 3+ healthy validator nodes in distinct regions for quorum.'
        }
      ]
    }
  ]);

  const handleSend = async (queryText?: string) => {
    const textToSend = queryText || inputQuery;
    if (!textToSend.trim() || loading) return;

    const userMsg: CopilotMessage = {
      id: `usr-${Date.now()}`,
      sender: 'user',
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      content: textToSend
    };

    setMessages((prev) => [...prev, userMsg]);
    setInputQuery('');
    setLoading(true);

    try {
      const res = await api.queryCopilot(textToSend, usePlatformContext);
      setMessages((prev) => [...prev, res.data]);
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        {
          id: `err-${Date.now()}`,
          sender: 'assistant',
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          content: `I could not complete that query: ${err.message}. Please check that backend services are active.`
        }
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleApproveAction = async (actionId: string) => {
    try {
      const res = await api.approveCopilotAction(actionId);
      alert(`Action executed: ${res.message}`);
      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.proposedAction?.id === actionId) {
            return {
              ...msg,
              proposedAction: { ...msg.proposedAction, status: 'EXECUTED' }
            };
          }
          return msg;
        })
      );
    } catch (err: any) {
      alert(`Execution failed: ${err.message}`);
    }
  };

  const handleDismissAction = async (actionId: string) => {
    try {
      await api.dismissCopilotAction(actionId);
      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.proposedAction?.id === actionId) {
            return {
              ...msg,
              proposedAction: { ...msg.proposedAction, status: 'DISMISSED' }
            };
          }
          return msg;
        })
      );
    } catch (err: any) {
      alert(`Dismiss failed: ${err.message}`);
    }
  };

  const handleCopyCode = (code: string, tabName: string) => {
    navigator.clipboard.writeText(code);
    setCopiedCodeTab(tabName);
    setTimeout(() => setCopiedCodeTab(null), 2000);
  };

  const quickPrompts = [
    'Check the health of my websites',
    'Show me my bandwidth usage',
    'How do I backup my data?',
    'Set up a subdomain for my app',
    'Troubleshoot deployment error and degraded nodes',
    'Suggest performance improvements'
  ];

  return (
    <div className="space-y-6">
      {/* Top Banner matching copiolt RAG.png */}
      <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4 p-5 rounded-2xl bg-gradient-to-r from-[#0E172C] via-[#0B1222] to-[#0A0F1D] border border-cyan-500/30 shadow-xl">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-cyan-400">
            <Sparkles className="w-4 h-4" />
            <span>RAG Infrastructure Engine</span>
            <span>·</span>
            <CapabilityBadge state="LIVE" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">
            RAG Agent — Copilot Assistant
          </h1>
          <p className="text-xs text-slate-300">
            Your AI assistant with knowledge of your infrastructure, documentation, and best practices.
          </p>
        </div>

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2.5 px-3.5 py-2 rounded-xl bg-slate-900/80 border border-slate-700 text-xs">
            <Brain className="w-5 h-5 text-cyan-400" />
            <div className="flex flex-col">
              <span className="font-bold text-white leading-tight">Powered by RAG</span>
              <span className="text-[10px] text-slate-400">Contextual answers from verified docs</span>
            </div>
          </div>

          <div className="flex items-center gap-2 px-3 py-2 rounded-xl bg-slate-900/80 border border-slate-700 text-xs">
            <span className="text-slate-300 font-medium">Use platform context</span>
            <button
              onClick={() => setUsePlatformContext(!usePlatformContext)}
              className={`w-9 h-5 rounded-full transition-colors relative ${
                usePlatformContext ? 'bg-emerald-500' : 'bg-slate-700'
              }`}
            >
              <span
                className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                  usePlatformContext ? 'right-0.5' : 'left-0.5'
                }`}
              />
            </button>
          </div>
        </div>
      </div>

      {/* Feature tags row matching copiolt RAG.png */}
      <div className="flex items-center gap-2 overflow-x-auto pb-1 text-xs font-mono">
        <span className="px-3 py-1 rounded-xl bg-blue-950/60 border border-blue-500/30 text-blue-300 flex items-center gap-1.5 whitespace-nowrap">
          <FileText className="w-3.5 h-3.5" /> Your Data & Docs
        </span>
        <span className="px-3 py-1 rounded-xl bg-emerald-950/60 border border-emerald-500/30 text-emerald-300 flex items-center gap-1.5 whitespace-nowrap">
          <Server className="w-3.5 h-3.5" /> Real-time Platform Context
        </span>
        <span className="px-3 py-1 rounded-xl bg-amber-950/60 border border-amber-500/30 text-amber-300 flex items-center gap-1.5 whitespace-nowrap">
          <Lock className="w-3.5 h-3.5" /> Secure & Private
        </span>
        <span className="px-3 py-1 rounded-xl bg-purple-950/60 border border-purple-500/30 text-purple-300 flex items-center gap-1.5 whitespace-nowrap">
          <ShieldCheck className="w-3.5 h-3.5" /> Deploy & Automate
        </span>
      </div>

      {/* Main Copilot Layout: Chat Area (8 cols) + Right Context Sidebar (4 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Chat Stream (8 cols) */}
        <div className="lg:col-span-8 flex flex-col justify-between h-[640px] rounded-2xl bg-[#0B1120] border border-slate-800 p-5 shadow-2xl">
          {/* Messages Scroll Area */}
          <div className="flex-1 overflow-y-auto space-y-5 pr-2">
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={`flex gap-3 ${msg.sender === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                {msg.sender === 'assistant' && (
                  <div className="w-8 h-8 rounded-xl bg-gradient-to-tr from-blue-600 to-cyan-500 flex items-center justify-center text-white flex-shrink-0 mt-0.5">
                    <Sparkles className="w-4 h-4" />
                  </div>
                )}

                <div
                  className={`max-w-[85%] rounded-2xl p-4 text-xs leading-relaxed space-y-3 ${
                    msg.sender === 'user'
                      ? 'bg-blue-600 text-white rounded-tr-none'
                      : 'bg-slate-900/90 border border-slate-800 text-slate-200 rounded-tl-none'
                  }`}
                >
                  <div className="flex items-center justify-between text-[10px] text-slate-400 font-mono pb-1 border-b border-slate-800/60">
                    <span className="font-semibold text-slate-300">
                      {msg.sender === 'user' ? 'Febin Francis' : 'Decentralized.Host Copilot (AI)'}
                    </span>
                    <span>{msg.timestamp}</span>
                  </div>

                  {/* Main text content */}
                  <div className="whitespace-pre-line">{msg.content}</div>

                  {/* Step Guide Card matching copiolt RAG.png */}
                  {msg.stepGuide && (
                    <div className="p-4 rounded-xl bg-[#070B14] border border-blue-500/30 space-y-4">
                      <div className="text-xs font-bold text-white font-sans">
                        {msg.stepGuide.title}
                      </div>

                      {/* Circles Timeline */}
                      <div className="grid grid-cols-5 gap-2 text-center font-mono">
                        {msg.stepGuide.steps.map((st) => (
                          <div key={st.number} className="flex flex-col items-center">
                            <div className="w-7 h-7 rounded-full bg-blue-600 text-white font-bold flex items-center justify-center text-xs mb-1">
                              {st.number}
                            </div>
                            <span className="text-[10px] font-semibold text-white leading-tight">
                              {st.label}
                            </span>
                            <span className="text-[8px] text-slate-400">{st.desc}</span>
                          </div>
                        ))}
                      </div>

                      {/* Code Snippets with tab selector */}
                      {msg.stepGuide.codeSnippets && (
                        <div className="rounded-xl bg-black border border-slate-800 overflow-hidden">
                          <div className="flex items-center justify-between px-3 py-1.5 bg-slate-950 border-b border-slate-800">
                            <div className="flex gap-2">
                              {msg.stepGuide.codeSnippets.map((snip) => (
                                <button
                                  key={snip.tabName}
                                  onClick={() =>
                                    setActiveCodeTabs((prev) => ({
                                      ...prev,
                                      [msg.id]: snip.tabName
                                    }))
                                  }
                                  className={`text-[10px] font-mono px-2 py-0.5 rounded ${
                                    (activeCodeTabs[msg.id] || msg.stepGuide?.codeSnippets?.[0].tabName) === snip.tabName
                                      ? 'bg-blue-600 text-white'
                                      : 'text-slate-400 hover:text-white'
                                  }`}
                                >
                                  {snip.tabName}
                                </button>
                              ))}
                            </div>

                            <button
                              onClick={() => {
                                const currentTab = activeCodeTabs[msg.id] || msg.stepGuide?.codeSnippets?.[0].tabName;
                                const snip = msg.stepGuide?.codeSnippets?.find((s) => s.tabName === currentTab);
                                if (snip) handleCopyCode(snip.code, snip.tabName);
                              }}
                              className="text-[10px] text-slate-400 hover:text-white flex items-center gap-1 font-mono"
                            >
                              {copiedCodeTab ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                              <span>{copiedCodeTab ? 'Copied!' : 'Copy'}</span>
                            </button>
                          </div>

                          <pre className="p-3 text-[11px] font-mono text-cyan-300 overflow-x-auto leading-relaxed">
                            {
                              msg.stepGuide.codeSnippets.find(
                                (s) => s.tabName === (activeCodeTabs[msg.id] || msg.stepGuide?.codeSnippets?.[0].tabName)
                              )?.code
                            }
                          </pre>
                        </div>
                      )}

                      {/* Doc links */}
                      {msg.stepGuide.docLinks && (
                        <div className="pt-2 border-t border-slate-800">
                          <span className="text-[10px] font-mono text-slate-400 uppercase tracking-wider">
                            Relevant Documentation:
                          </span>
                          <div className="flex flex-wrap gap-2 mt-1">
                            {msg.stepGuide.docLinks.map((doc) => (
                              <div
                                key={doc.title}
                                className="px-2 py-1 rounded bg-slate-900 border border-slate-800 text-[10px] font-mono text-cyan-400 flex items-center gap-1 hover:underline cursor-pointer"
                              >
                                <FileText className="w-3 h-3" />
                                <span>{doc.title}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Grounded Citations */}
                  {msg.citations && msg.citations.length > 0 && (
                    <div className="p-3 rounded-xl bg-slate-950/70 border border-cyan-500/20 space-y-1.5 font-mono text-[11px]">
                      <div className="text-[10px] uppercase text-cyan-400 font-bold flex items-center gap-1">
                        <ShieldCheck className="w-3.5 h-3.5" />
                        <span>Grounded Provenance Citations ({msg.citations.length})</span>
                      </div>
                      {msg.citations.map((c) => (
                        <div key={c.id} className="text-slate-300 pl-2 border-l border-cyan-500/40">
                          <span className="text-cyan-300 font-semibold">{c.title}</span>: {c.snippet}
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Structured Action Proposal matching Master Prompt Rule */}
                  {msg.proposedAction && (
                    <div
                      className={`p-4 rounded-xl border space-y-3 font-mono ${
                        msg.proposedAction.status === 'EXECUTED'
                          ? 'bg-emerald-950/40 border-emerald-500/40 text-emerald-200'
                          : msg.proposedAction.status === 'DISMISSED'
                          ? 'bg-slate-950 border-slate-800 opacity-60'
                          : 'bg-amber-950/40 border-amber-500/50 text-amber-200'
                      }`}
                    >
                      <div className="flex items-center justify-between font-sans">
                        <div className="flex items-center gap-2">
                          <AlertTriangle className="w-4 h-4 text-amber-400" />
                          <span className="font-bold text-xs text-white">
                            Proposed Remediation Operation ({msg.proposedAction.riskLevel} RISK)
                          </span>
                        </div>
                        <span className="text-[10px] px-2 py-0.5 rounded bg-black border border-slate-700">
                          {msg.proposedAction.status}
                        </span>
                      </div>

                      <div className="text-xs font-semibold text-white">{msg.proposedAction.title}</div>
                      <p className="text-[11px] text-slate-300 leading-snug">{msg.proposedAction.description}</p>
                      <div className="text-[10px] text-cyan-400 font-mono">Impact: {msg.proposedAction.impact}</div>

                      {msg.proposedAction.status === 'PENDING_APPROVAL' && (
                        <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-800/80">
                          <button
                            onClick={() => handleDismissAction(msg.proposedAction!.id)}
                            className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-sans"
                          >
                            Dismiss
                          </button>
                          <button
                            onClick={() => handleApproveAction(msg.proposedAction!.id)}
                            className="px-4 py-1.5 rounded-lg bg-gradient-to-r from-amber-600 to-orange-600 hover:opacity-90 text-white font-bold text-xs font-sans shadow-lg flex items-center gap-1.5"
                          >
                            <ShieldCheck className="w-3.5 h-3.5" />
                            <span>Approve & Execute Operation</span>
                          </button>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {msg.sender === 'user' && (
                  <div className="w-8 h-8 rounded-xl bg-blue-700 flex items-center justify-center font-bold text-white text-xs flex-shrink-0 mt-0.5">
                    F
                  </div>
                )}
              </div>
            ))}

            {loading && (
              <div className="flex items-center gap-2 text-xs font-mono text-cyan-400 py-2">
                <RefreshCw className="w-4 h-4 animate-spin" />
                <span>Retrieving verified documentation and node observations...</span>
              </div>
            )}
          </div>

          {/* Chat Input Bar */}
          <div className="pt-3 border-t border-slate-800">
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSend();
              }}
              className="flex items-center gap-2 p-2 rounded-2xl bg-slate-900 border border-slate-700 shadow-inner"
            >
              <input
                type="text"
                value={inputQuery}
                onChange={(e) => setInputQuery(e.target.value)}
                placeholder='Ask a question or try: "Troubleshoot deployment error and degraded nodes"'
                className="flex-1 bg-transparent text-white text-xs font-sans px-3 focus:outline-none placeholder-slate-500"
              />
              <button
                type="submit"
                disabled={loading || !inputQuery.trim()}
                className="p-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-40 text-white shadow-md transition-all"
              >
                <Send className="w-4 h-4" />
              </button>
            </form>
          </div>
        </div>

        {/* Right Context & Knowledge Sources Sidebar (4 cols) matching copiolt RAG.png */}
        <div className="lg:col-span-4 space-y-5">
          {/* Knowledge Sources matching copiolt RAG.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3 text-xs">
            <div className="flex items-center justify-between">
              <h2 className="font-bold text-white">Knowledge Sources</h2>
              <span className="text-[11px] text-cyan-400 font-mono">4 Sources</span>
            </div>

            <div className="space-y-2 font-mono text-[11px]">
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between">
                <span className="text-slate-300">Platform Documentation</span>
                <span className="text-emerald-400">✓ 1,243 pages</span>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between">
                <span className="text-slate-300">Your Infrastructure</span>
                <span className="text-emerald-400">✓ Real-time</span>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between">
                <span className="text-slate-300">Best Practices</span>
                <span className="text-emerald-400">✓ Verified</span>
              </div>
              <div className="p-2.5 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between">
                <span className="text-slate-300">Community & Support</span>
                <span className="text-cyan-400">✓ Updated</span>
              </div>
            </div>
          </div>

          {/* Quick Prompts matching copiolt RAG.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3">
            <h2 className="text-xs font-bold text-white">Quick Prompts</h2>
            <div className="space-y-1.5">
              {quickPrompts.map((prompt) => (
                <button
                  key={prompt}
                  onClick={() => handleSend(prompt)}
                  className="w-full text-left p-2 rounded-xl bg-slate-900/60 border border-slate-800 hover:border-cyan-500/40 text-xs text-slate-300 hover:text-white transition-all flex items-center justify-between group"
                >
                  <span className="truncate">{prompt}</span>
                  <ArrowRight className="w-3 h-3 text-slate-500 group-hover:text-cyan-400 flex-shrink-0" />
                </button>
              ))}
            </div>
          </div>

          {/* Current Context Card matching copiolt RAG.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-2.5 font-mono text-xs">
            <h2 className="font-bold text-white font-sans text-xs">Current Context</h2>
            <div className="space-y-1.5 text-[11px]">
              <div className="flex justify-between">
                <span className="text-slate-500">Account:</span>
                <span className="text-white">Decentralized.Host (Owner)</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Active Websites:</span>
                <span className="text-cyan-400">12 websites connected</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Active Nodes:</span>
                <span className="text-emerald-400">41 / 43 online</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Default Region:</span>
                <span className="text-white">Asia/Kolkata</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
