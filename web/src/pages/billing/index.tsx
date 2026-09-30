import React, { useState, useEffect } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Card } from '@/components/Card'
import { Badge } from '@/components/Badge'
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, AreaChart, Area } from 'recharts'

interface CostMetric {
  name: string
  monthly: number
  trend: number
  color: string
}

interface Invoice {
  id: string
  month: string
  amount: number
  status: 'paid' | 'pending' | 'overdue'
  items: number
  dueDate: string
}

interface BudgetAlert {
  resource: string
  usage: number
  budget: number
  percentage: number
  status: 'warning' | 'critical' | 'safe'
}

interface CostProjection {
  month: string
  projected: number
  actual: number
}

const generateMockCostMetrics = (): CostMetric[] => [
  { name: 'Compute', monthly: 2840, trend: 8.5, color: 'text-blue-400' },
  { name: 'Storage', monthly: 1520, trend: 12.3, color: 'text-green-400' },
  { name: 'Network', monthly: 892, trend: -3.2, color: 'text-purple-400' },
  { name: 'Database', monthly: 1240, trend: 6.1, color: 'text-orange-400' },
  { name: 'Security', monthly: 456, trend: 2.4, color: 'text-red-400' },
]

const generateMockInvoices = (): Invoice[] => [
  { id: 'INV-2026-09', month: 'September 2026', amount: 6948, status: 'paid', items: 12, dueDate: '2026-10-05' },
  { id: 'INV-2026-08', month: 'August 2026', amount: 6524, status: 'paid', items: 12, dueDate: '2026-09-05' },
  { id: 'INV-2026-07', month: 'July 2026', amount: 6182, status: 'paid', items: 12, dueDate: '2026-08-05' },
  { id: 'INV-2026-06', month: 'June 2026', amount: 5840, status: 'paid', items: 12, dueDate: '2026-07-05' },
  { id: 'INV-2026-05', month: 'May 2026', amount: 5320, status: 'paid', items: 11, dueDate: '2026-06-05' },
]

const generateMockBudgetAlerts = (): BudgetAlert[] => [
  { resource: 'Compute Budget', usage: 8400, budget: 9000, percentage: 93, status: 'warning' },
  { resource: 'Storage Budget', usage: 4560, budget: 5000, percentage: 91, status: 'warning' },
  { resource: 'Network Budget', usage: 2680, budget: 3000, percentage: 89, status: 'safe' },
  { resource: 'Database Budget', usage: 3720, budget: 4000, percentage: 93, status: 'warning' },
  { resource: 'Security Budget', usage: 1368, budget: 1500, percentage: 91, status: 'warning' },
]

const generateMockCostProjection = (): CostProjection[] => [
  { month: 'May', actual: 5320, projected: 5400 },
  { month: 'Jun', actual: 5840, projected: 5900 },
  { month: 'Jul', actual: 6182, projected: 6200 },
  { month: 'Aug', actual: 6524, projected: 6600 },
  { month: 'Sep', actual: 6948, projected: 7050 },
  { month: 'Oct', actual: 0, projected: 7200 },
  { month: 'Nov', actual: 0, projected: 7350 },
]

const generateMockCostBreakdown = () => [
  { service: 'Compute', value: 2840, percentage: 41 },
  { service: 'Storage', value: 1520, percentage: 22 },
  { service: 'Network', value: 892, percentage: 13 },
  { service: 'Database', value: 1240, percentage: 18 },
  { service: 'Security', value: 456, percentage: 6 },
]

export default function BillingPage() {
  const [costMetrics, setCostMetrics] = useState<CostMetric[]>([])
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [budgetAlerts, setBudgetAlerts] = useState<BudgetAlert[]>([])
  const [costProjection, setCostProjection] = useState<CostProjection[]>([])
  const [costBreakdown, setCostBreakdown] = useState<any[]>([])
  const [expandedInvoice, setExpandedInvoice] = useState<string | null>(null)

  useEffect(() => {
    setCostMetrics(generateMockCostMetrics())
    setInvoices(generateMockInvoices())
    setBudgetAlerts(generateMockBudgetAlerts())
    setCostProjection(generateMockCostProjection())
    setCostBreakdown(generateMockCostBreakdown())
  }, [])

  const totalMonthlyCost = costMetrics.reduce((sum, m) => sum + m.monthly, 0)
  const avgCost = (costMetrics.reduce((sum, m) => sum + m.monthly, 0) / costMetrics.length).toFixed(2)
  const costTrend = (costMetrics.reduce((sum, m) => sum + m.trend, 0) / costMetrics.length).toFixed(1)
  const budgetUtilization = ((costMetrics.reduce((sum, m) => sum + m.monthly, 0) / 8000) * 100).toFixed(1)

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'paid': return 'bg-green-500/20 text-green-400 border-green-500/50'
      case 'pending': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'overdue': return 'bg-red-500/20 text-red-400 border-red-500/50'
      default: return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  const getAlertColor = (status: string) => {
    switch (status) {
      case 'critical': return 'bg-red-500/20 text-red-400 border-red-500/50'
      case 'warning': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50'
      case 'safe': return 'bg-green-500/20 text-green-400 border-green-500/50'
      default: return 'bg-neutral-700/50 text-neutral-300 border-neutral-600/50'
    }
  }

  return (
    <div className="flex bg-neutral-950 min-h-screen">
      <Sidebar />
      <div className="flex-1 ml-64 p-8">
        <div className="space-y-8">
          {/* Header */}
          <div>
            <h1 className="text-4xl font-bold text-white mb-2">Billing & Cost Management</h1>
            <p className="text-neutral-400">Track infrastructure costs, invoices, and budget utilization</p>
          </div>

          {/* Summary Stats */}
          <div className="grid grid-cols-5 gap-4">
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Total Monthly Cost</div>
              <div className="text-3xl font-bold text-white mb-1">${totalMonthlyCost.toLocaleString()}</div>
              <div className="text-xs text-green-400">↑ 6.4% from last month</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Average Resource Cost</div>
              <div className="text-3xl font-bold text-white mb-1">${avgCost}</div>
              <div className="text-xs text-neutral-400">Per resource category</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Cost Trend</div>
              <div className="text-3xl font-bold text-white mb-1">{costTrend}%</div>
              <div className="text-xs text-yellow-400">3-month average</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Budget Utilization</div>
              <div className="text-3xl font-bold text-white mb-1">{budgetUtilization}%</div>
              <div className="text-xs text-neutral-400">of $8,000 monthly budget</div>
            </Card>
            <Card>
              <div className="text-neutral-400 text-sm font-medium mb-2">Unpaid Invoices</div>
              <div className="text-3xl font-bold text-white mb-1">0</div>
              <div className="text-xs text-green-400">All invoices current</div>
            </Card>
          </div>

          {/* Cost Breakdown */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Cost by Service</h2>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={costBreakdown}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="service" stroke="#999" />
                <YAxis stroke="#999" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                <Bar dataKey="value" fill="#00D9FF" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          {/* Cost Projection */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Cost Projection</h2>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={costProjection}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="month" stroke="#999" />
                <YAxis stroke="#999" />
                <Tooltip contentStyle={{ backgroundColor: '#1a1a1a', border: '1px solid #333' }} />
                <Legend />
                <Area type="monotone" dataKey="actual" stroke="#00D9FF" fill="#00D9FF" fillOpacity={0.3} name="Actual Cost" />
                <Area type="monotone" dataKey="projected" stroke="#7C3AED" fill="#7C3AED" fillOpacity={0.2} name="Projected Cost" />
              </AreaChart>
            </ResponsiveContainer>
          </Card>

          {/* Cost Breakdown by Resource */}
          <div className="grid grid-cols-2 gap-4">
            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Monthly Costs by Resource</h2>
              <div className="space-y-4">
                {costMetrics.map((metric) => (
                  <div key={metric.name}>
                    <div className="flex justify-between mb-2">
                      <span className="text-neutral-300">{metric.name}</span>
                      <div className="flex gap-4">
                        <span className="text-white font-semibold">${metric.monthly.toLocaleString()}</span>
                        <span className={`${metric.trend > 0 ? 'text-red-400' : 'text-green-400'}`}>
                          {metric.trend > 0 ? '↑' : '↓'} {Math.abs(metric.trend)}%
                        </span>
                      </div>
                    </div>
                    <div className="w-full bg-neutral-800 rounded-full h-2">
                      <div
                        className={`h-2 rounded-full ${metric.color.replace('text-', 'bg-')}`}
                        style={{ width: `${(metric.monthly / totalMonthlyCost) * 100}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </Card>

            <Card>
              <h2 className="text-xl font-bold text-white mb-6">Budget Status</h2>
              <div className="space-y-4">
                {budgetAlerts.map((alert) => (
                  <div key={alert.resource} className="p-3 bg-neutral-900 rounded-lg border border-neutral-700">
                    <div className="flex justify-between mb-2">
                      <span className="text-neutral-300">{alert.resource}</span>
                      <span className="text-white font-semibold">{alert.percentage}%</span>
                    </div>
                    <div className="w-full bg-neutral-800 rounded-full h-2 mb-2">
                      <div
                        className={`h-2 rounded-full ${
                          alert.status === 'critical'
                            ? 'bg-red-500'
                            : alert.status === 'warning'
                            ? 'bg-yellow-500'
                            : 'bg-green-500'
                        }`}
                        style={{ width: `${alert.percentage}%` }}
                      />
                    </div>
                    <div className="text-xs text-neutral-400">
                      ${alert.usage.toLocaleString()} of ${alert.budget.toLocaleString()}
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          </div>

          {/* Invoices */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Invoice History</h2>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-neutral-700">
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Invoice ID</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Period</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Amount</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Items</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Status</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Due Date</th>
                    <th className="text-left py-3 px-4 text-neutral-400 font-medium">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {invoices.map((invoice) => (
                    <React.Fragment key={invoice.id}>
                      <tr className="border-b border-neutral-800 hover:bg-neutral-900/50 transition-colors">
                        <td className="py-3 px-4 text-white">{invoice.id}</td>
                        <td className="py-3 px-4 text-neutral-300">{invoice.month}</td>
                        <td className="py-3 px-4 text-white font-semibold">${invoice.amount.toLocaleString()}</td>
                        <td className="py-3 px-4 text-neutral-300">{invoice.items} services</td>
                        <td className="py-3 px-4">
                          <Badge className={getStatusColor(invoice.status)}>
                            {invoice.status.charAt(0).toUpperCase() + invoice.status.slice(1)}
                          </Badge>
                        </td>
                        <td className="py-3 px-4 text-neutral-300">{invoice.dueDate}</td>
                        <td className="py-3 px-4">
                          <button
                            onClick={() => setExpandedInvoice(expandedInvoice === invoice.id ? null : invoice.id)}
                            className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors"
                          >
                            {expandedInvoice === invoice.id ? 'Hide' : 'View'} Details
                          </button>
                        </td>
                      </tr>
                      {expandedInvoice === invoice.id && (
                        <tr className="border-b border-neutral-800 bg-neutral-900/30">
                          <td colSpan={7} className="py-4 px-4">
                            <div className="grid grid-cols-2 gap-4 text-sm">
                              <div>
                                <div className="text-neutral-400 mb-1">Invoice Details</div>
                                <div className="space-y-1 text-neutral-300">
                                  <div>ID: {invoice.id}</div>
                                  <div>Period: {invoice.month}</div>
                                  <div>Total: ${invoice.amount.toLocaleString()}</div>
                                </div>
                              </div>
                              <div className="text-right">
                                <button className="px-3 py-1.5 bg-primary-500/20 text-primary-400 rounded hover:bg-primary-500/30 text-sm font-medium transition-colors">
                                  Download PDF
                                </button>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>

          {/* Payment Method */}
          <Card>
            <h2 className="text-xl font-bold text-white mb-6">Payment Method</h2>
            <div className="bg-neutral-900 rounded-lg p-6 border border-neutral-700">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <div className="text-neutral-400 text-sm mb-2">Primary Card</div>
                  <div className="text-white font-semibold">Visa ending in 4242</div>
                  <div className="text-neutral-400 text-sm">Expires 12/2028</div>
                </div>
                <div className="text-4xl text-primary-400">💳</div>
              </div>
              <button className="px-4 py-2 bg-neutral-800 hover:bg-neutral-700 rounded text-neutral-300 text-sm font-medium transition-colors">
                Update Payment Method
              </button>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
