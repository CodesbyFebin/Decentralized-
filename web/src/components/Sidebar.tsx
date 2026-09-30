import React, { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/router'

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: '📊' },
  { name: 'Nodes', href: '/nodes', icon: '🖥️' },
  { name: 'Compute', href: '/compute', icon: '⚡' },
  { name: 'Storage', href: '/storage', icon: '💾' },
  { name: 'Deployments', href: '/deploy', icon: '🚀' },
  { name: 'Domains', href: '/domains', icon: '🌐' },
  { name: 'Security', href: '/security', icon: '🔒' },
  { name: 'Analytics', href: '/analytics', icon: '📈' },
  { name: 'Alerts', href: '/alerts', icon: '🔔' },
  { name: 'Logs', href: '/logs', icon: '📝' },
  { name: 'Integrations', href: '/integrations', icon: '🔗' },
  { name: 'Webhooks', href: '/webhooks', icon: '🪝' },
  { name: 'Audit', href: '/audit', icon: '📖' },
  { name: 'Compliance', href: '/compliance', icon: '✅' },
  { name: 'Team', href: '/team', icon: '👥' },
  { name: 'Settings', href: '/settings', icon: '⚙️' },
  { name: 'Copilot', href: '/copilot', icon: '🤖' },
  { name: 'Evidence', href: '/evidence', icon: '📋' },
]

export const Sidebar: React.FC = () => {
  const router = useRouter()
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div
      className={`fixed left-0 top-0 h-screen bg-neutral-950 border-r border-neutral-700 transition-all duration-300 ${
        collapsed ? 'w-20' : 'w-64'
      }`}
    >
      <div className="p-4 border-b border-neutral-700">
        <div className="flex items-center justify-between">
          {!collapsed && (
            <h1 className="text-xl font-bold text-primary-500">Decentralized</h1>
          )}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="p-2 hover:bg-neutral-800 rounded-lg transition-colors"
          >
            {collapsed ? '→' : '←'}
          </button>
        </div>
      </div>

      <nav className="p-4 space-y-2">
        {navigation.map((item) => {
          const isActive = router.pathname === item.href
          return (
            <Link key={item.href} href={item.href}>
              <a
                className={`flex items-center gap-3 px-4 py-2.5 rounded-lg transition-colors ${
                  isActive
                    ? 'bg-primary-500/20 text-primary-500 border border-primary-500/50'
                    : 'text-neutral-400 hover:bg-neutral-800 hover:text-neutral-200'
                }`}
              >
                <span className="text-lg">{item.icon}</span>
                {!collapsed && <span className="font-medium">{item.name}</span>}
              </a>
            </Link>
          )
        })}
      </nav>

      <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-neutral-700">
        <button className="w-full px-4 py-2 bg-neutral-800 hover:bg-neutral-700 rounded-lg text-neutral-300 text-sm font-medium transition-colors">
          {!collapsed && 'Logout'}
        </button>
      </div>
    </div>
  )
}
