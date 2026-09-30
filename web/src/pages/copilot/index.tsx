import React, { useEffect, useState, useRef } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: string
  sources?: string[]
}

interface ChatSession {
  id: string
  title: string
  created: string
  messageCount: number
}

const Copilot: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([])
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null)
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [thinking, setThinking] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const suggestedQueries = [
    'What is the current system health status?',
    'How to optimize node resource allocation?',
    'Show recent deployment history and status',
    'Explain the RBAC permission model',
    'How to create and manage API keys?',
  ]

  const generateMockSessions = (): ChatSession[] => [
    {
      id: 'session-1',
      title: 'Node Performance Troubleshooting',
      created: '2026-09-30T08:00:00Z',
      messageCount: 12,
    },
    {
      id: 'session-2',
      title: 'Deployment Strategy Discussion',
      created: '2026-09-28T14:30:00Z',
      messageCount: 8,
    },
    {
      id: 'session-3',
      title: 'Security Configuration Review',
      created: '2026-09-25T10:15:00Z',
      messageCount: 15,
    },
  ]

  const generateMockMessages = (): Message[] => [
    {
      id: 'msg-1',
      role: 'user',
      content: 'What is the current status of the cluster?',
      timestamp: new Date(Date.now() - 300000).toISOString(),
    },
    {
      id: 'msg-2',
      role: 'assistant',
      content: 'The cluster is operating normally with 23 active nodes. Current metrics:\n\n• CPU Utilization: 67.9% (630 cores used / 928 allocated)\n• Memory: 63.0% (1170 GB / 1856 GB)\n• Uptime: 99.98%\n\nAll three master nodes are healthy, 12 worker nodes are operational, and 8 edge nodes are active. No alerts or warnings at this time.',
      timestamp: new Date(Date.now() - 290000).toISOString(),
      sources: ['Dashboard', 'Compute Resources', 'System Health'],
    },
    {
      id: 'msg-3',
      role: 'user',
      content: 'How can I improve memory utilization efficiency?',
      timestamp: new Date(Date.now() - 180000).toISOString(),
    },
    {
      id: 'msg-4',
      role: 'assistant',
      content: 'Based on current metrics, here are recommendations to improve memory utilization:\n\n1. **Container Optimization**: Review container memory limits on worker nodes. Some containers may be over-provisioned.\n\n2. **Pod Consolidation**: Consider consolidating underutilized pods to free up allocated memory blocks.\n\n3. **Cache Optimization**: Implement intelligent caching strategies on API endpoints to reduce memory pressure.\n\n4. **Scaling Strategy**: The current 63% utilization suggests room for workload consolidation before scaling up.\n\nWould you like me to generate a detailed optimization report?',
      timestamp: new Date(Date.now() - 170000).toISOString(),
      sources: ['Compute Resources', 'Performance Best Practices'],
    },
  ]

  useEffect(() => {
    const initializeCopilot = () => {
      const mockSessions = generateMockSessions()
      setSessions(mockSessions)
      setCurrentSessionId(mockSessions[0].id)
      setMessages(generateMockMessages())
    }

    initializeCopilot()
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSendMessage = async (query: string) => {
    if (!query.trim()) return

    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: query,
      timestamp: new Date().toISOString(),
    }

    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setThinking(true)

    try {
      await apiClient.post('/copilot/chat', {
        sessionId: currentSessionId,
        message: query,
      })

      // Simulate response delay
      await new Promise((resolve) => setTimeout(resolve, 1500))

      const assistantMessage: Message = {
        id: `msg-${Date.now() + 1}`,
        role: 'assistant',
        content: `I understand your question about "${query}". Based on the system documentation and current metrics, here are some relevant insights...\n\nThis is a simulated response. In production, this would be powered by RAG (Retrieval-Augmented Generation) to provide accurate, context-aware answers based on your infrastructure data.`,
        timestamp: new Date().toISOString(),
        sources: ['System Documentation', 'Recent Analytics', 'Configuration'],
      }

      setMessages((prev) => [...prev, assistantMessage])
    } catch (err) {
      console.error('Failed to send message:', err)
    } finally {
      setThinking(false)
    }
  }

  const handleNewSession = () => {
    const newSession: ChatSession = {
      id: `session-${Date.now()}`,
      title: 'New Conversation',
      created: new Date().toISOString(),
      messageCount: 0,
    }
    setSessions((prev) => [newSession, ...prev])
    setCurrentSessionId(newSession.id)
    setMessages([])
  }

  return (
    <AppLayout
      title="RAG Copilot"
      subtitle="AI-powered assistant with infrastructure knowledge"
    >
      <div className="flex gap-6 h-[calc(100vh-200px)]">
        {/* Sidebar with Sessions */}
        <Card variant="glass" className="w-80 flex flex-col">
          <div className="p-6 border-b border-neutral-700">
            <Button
              variant="primary"
              size="md"
              onClick={handleNewSession}
              className="w-full"
            >
              + New Chat
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-2">
            <p className="text-xs text-neutral-400 px-2 mb-3">Recent Conversations</p>
            {sessions.map((session) => (
              <button
                key={session.id}
                onClick={() => setCurrentSessionId(session.id)}
                className={`w-full text-left px-4 py-3 rounded-lg transition ${
                  currentSessionId === session.id
                    ? 'bg-primary-500/20 border border-primary-500'
                    : 'hover:bg-neutral-800/30'
                }`}
              >
                <p className="text-sm font-medium text-neutral-100 truncate">{session.title}</p>
                <p className="text-xs text-neutral-500 mt-1">{session.messageCount} messages</p>
              </button>
            ))}
          </div>
        </Card>

        {/* Chat Area */}
        <div className="flex-1 flex flex-col gap-6">
          <Card variant="glass" className="flex-1 flex flex-col">
            {messages.length === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center p-12 space-y-6">
                <div className="text-center">
                  <h2 className="text-2xl font-bold text-neutral-100 mb-2">
                    How can I help you today?
                  </h2>
                  <p className="text-neutral-400">
                    Ask about your infrastructure, deployments, security, or system configuration
                  </p>
                </div>
                <div className="grid grid-cols-1 gap-2 w-full max-w-2xl">
                  {suggestedQueries.map((query, i) => (
                    <button
                      key={i}
                      onClick={() => handleSendMessage(query)}
                      className="text-left px-4 py-3 bg-neutral-800 hover:bg-neutral-700 rounded-lg text-sm text-neutral-300 transition"
                    >
                      {query}
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <>
                <div className="flex-1 overflow-y-auto p-6 space-y-4">
                  {messages.map((message) => (
                    <div
                      key={message.id}
                      className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                    >
                      <div
                        className={`max-w-2xl px-4 py-3 rounded-lg ${
                          message.role === 'user'
                            ? 'bg-primary-600 text-white'
                            : 'bg-neutral-800 text-neutral-100'
                        }`}
                      >
                        <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                        {message.sources && (
                          <div className="mt-2 flex flex-wrap gap-1">
                            {message.sources.map((source, i) => (
                              <span key={i} className="text-xs opacity-75 bg-neutral-700 px-2 py-1 rounded">
                                {source}
                              </span>
                            ))}
                          </div>
                        )}
                        <p className={`text-xs mt-1 ${
                          message.role === 'user' ? 'opacity-75' : 'text-neutral-500'
                        }`}>
                          {new Date(message.timestamp).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                  ))}
                  {thinking && (
                    <div className="flex justify-start">
                      <div className="bg-neutral-800 text-neutral-300 px-4 py-3 rounded-lg">
                        <div className="flex gap-2 items-center">
                          <div className="flex gap-1">
                            <div className="w-2 h-2 bg-primary-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                            <div className="w-2 h-2 bg-primary-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                            <div className="w-2 h-2 bg-primary-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                          </div>
                          <span className="text-sm">Thinking...</span>
                        </div>
                      </div>
                    </div>
                  )}
                  <div ref={messagesEndRef} />
                </div>

                {/* Input Area */}
                <div className="border-t border-neutral-700 p-6">
                  <div className="flex gap-3">
                    <input
                      type="text"
                      value={input}
                      onChange={(e) => setInput(e.target.value)}
                      onKeyPress={(e) => e.key === 'Enter' && handleSendMessage(input)}
                      placeholder="Ask anything about your infrastructure..."
                      disabled={thinking}
                      className="flex-1 px-4 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none disabled:opacity-50"
                    />
                    <Button
                      variant="primary"
                      size="md"
                      onClick={() => handleSendMessage(input)}
                      disabled={!input.trim() || thinking}
                    >
                      Send
                    </Button>
                  </div>
                </div>
              </>
            )}
          </Card>
        </div>
      </div>
    </AppLayout>
  )
}

export default Copilot
