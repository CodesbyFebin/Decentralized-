import React, { useState } from 'react'
import { Send, Lightbulb } from 'lucide-react'

interface Message {
  id: string
  type: 'user' | 'assistant'
  content: string
  timestamp: string
}

interface Suggestion {
  id: string
  type: 'optimization' | 'security' | 'performance' | 'cost'
  title: string
  description: string
  impact: string
  effort: 'low' | 'medium' | 'high'
  action: string
}

const mockSuggestions: Suggestion[] = [
  { id: '1', type: 'cost', title: 'Optimize Reserved Instance Usage', description: 'Currently using on-demand instances for non-critical workloads. Reserved instances could save 40% on compute costs.', impact: 'Reduce monthly costs by $2,400', effort: 'low', action: 'Apply' },
  { id: '2', type: 'performance', title: 'Enable Query Result Caching', description: 'Database query patterns show 60% cache-eligible queries. Implementing caching layer could reduce query latency by 45%.', impact: 'Reduce API latency from 250ms to 140ms', effort: 'medium', action: 'Implement' },
  { id: '3', type: 'security', title: 'Enable MFA for Admin Accounts', description: '3 admin accounts are still using single-factor authentication. MFA is recommended for all privileged accounts.', impact: 'Eliminate single-factor auth risk', effort: 'low', action: 'Enable' },
  { id: '4', type: 'optimization', title: 'Scale Down Non-Peak Instances', description: 'Usage patterns show 70% CPU utilization is only reached during 8-10am daily. Auto-scaling rules could optimize resource allocation.', impact: 'Reduce idle capacity waste by 35%', effort: 'medium', action: 'Configure' },
]

export default function Copilot() {
  const [messages, setMessages] = useState<Message[]>([
    { id: '1', type: 'assistant', content: 'Hello! I\'m your infrastructure copilot. I can help you optimize costs, improve security, enhance performance, and provide recommendations based on your system patterns. How can I assist you today?', timestamp: new Date().toISOString() },
  ])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSendMessage = () => {
    if (!input.trim()) return

    const userMessage: Message = {
      id: String(messages.length + 1),
      type: 'user',
      content: input,
      timestamp: new Date().toISOString(),
    }

    setMessages([...messages, userMessage])
    setInput('')
    setLoading(true)

    setTimeout(() => {
      const assistantMessage: Message = {
        id: String(messages.length + 2),
        type: 'assistant',
        content: 'I\'m analyzing your infrastructure based on your query. Based on current metrics, I recommend reviewing your backup retention policy and implementing the suggested cost optimizations. Would you like me to provide more details on any specific area?',
        timestamp: new Date().toISOString(),
      }
      setMessages((prev) => [...prev, assistantMessage])
      setLoading(false)
    }, 1000)
  }

  return (
    <div className="flex-1 overflow-auto">
      <div className="p-8 space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">Copilot</h1>
            <p className="text-neutral-400 mt-1">AI-powered infrastructure insights and recommendations</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6 h-96 overflow-y-auto flex flex-col">
              <div className="flex-1 space-y-4 mb-4">
                {messages.map((msg) => (
                  <div key={msg.id} className={`flex ${msg.type === 'user' ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-xs px-4 py-2 rounded-lg ${msg.type === 'user' ? 'bg-primary-600 text-white' : 'bg-neutral-800 text-neutral-200'}`}>
                      {msg.content}
                    </div>
                  </div>
                ))}
                {loading && (
                  <div className="flex justify-start">
                    <div className="bg-neutral-800 text-neutral-200 px-4 py-2 rounded-lg">
                      <span className="animate-pulse">Analyzing...</span>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="flex gap-2">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
                placeholder="Ask me anything about your infrastructure..."
                className="flex-1 px-4 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:outline-none focus:border-primary-500"
              />
              <button onClick={handleSendMessage} className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-medium transition-colors flex items-center gap-2">
                <Send size={18} />
              </button>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-white flex items-center gap-2">
              <Lightbulb size={20} className="text-yellow-500" />
              Quick Suggestions
            </h3>
            <div className="space-y-3 max-h-96 overflow-y-auto">
              {mockSuggestions.map((suggestion) => (
                <div key={suggestion.id} className="bg-neutral-900 border border-neutral-800 rounded-lg p-3 cursor-pointer hover:border-neutral-700 transition-colors">
                  <div className="flex items-start justify-between gap-2">
                    <span className={`px-2 py-1 rounded text-xs font-medium whitespace-nowrap ${suggestion.type === 'cost' ? 'bg-green-500/20 text-green-400' : suggestion.type === 'security' ? 'bg-red-500/20 text-red-400' : suggestion.type === 'performance' ? 'bg-blue-500/20 text-blue-400' : 'bg-purple-500/20 text-purple-400'}`}>
                      {suggestion.type}
                    </span>
                  </div>
                  <p className="font-medium text-white text-sm mt-2">{suggestion.title}</p>
                  <p className="text-xs text-neutral-400 mt-1">{suggestion.description}</p>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="bg-neutral-900 border border-neutral-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Recommended Actions</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {mockSuggestions.map((suggestion) => (
              <div key={suggestion.id} className="border border-neutral-800 rounded-lg p-4">
                <div className="flex items-start justify-between mb-3">
                  <h4 className="font-medium text-white">{suggestion.title}</h4>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${suggestion.effort === 'low' ? 'bg-green-500/20 text-green-400' : suggestion.effort === 'medium' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-red-500/20 text-red-400'}`}>
                    {suggestion.effort} effort
                  </span>
                </div>
                <p className="text-sm text-neutral-400">{suggestion.description}</p>
                <p className="text-sm text-primary-400 font-medium mt-2">Impact: {suggestion.impact}</p>
                <button className="mt-3 px-3 py-1 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm font-medium transition-colors">
                  {suggestion.action}
                </button>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
