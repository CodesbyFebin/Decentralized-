import React, { useEffect, useState } from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Badge } from '@/components/Badge'
import { Loading } from '@/components/Loading'
import { apiClient } from '@/lib/api'

interface LessonLearned {
  category: 'process' | 'technical' | 'communication'
  description: string
  action?: string
}

interface TimelineEvent {
  timestamp: string
  actor: string
  action: string
  details?: string
}

interface Postmortem {
  id: string
  incidentId: string
  incidentTitle: string
  severity: 'critical' | 'high' | 'medium' | 'low'
  date: string
  status: 'draft' | 'published' | 'archived'
  author: string
  rootCause: string
  contributingFactors: string[]
  lessonsLearned: LessonLearned[]
  timeline: TimelineEvent[]
  teamFeedback: number
  resolutionTime: number
}

interface PostmortemStats {
  total: number
  published: number
  criticalLessons: number
  avgFeedback: number
}

const Postmortems: React.FC = () => {
  const [postmortems, setPostmortems] = useState<Postmortem[]>([])
  const [stats, setStats] = useState<PostmortemStats>({
    total: 0,
    published: 0,
    criticalLessons: 0,
    avgFeedback: 0,
  })
  const [loading, setLoading] = useState(true)
  const [expandedPostmortem, setExpandedPostmortem] = useState<string | null>(null)
  const [selectedStatus, setSelectedStatus] = useState<string>('all')

  const generateMockPostmortems = (): Postmortem[] => {
    const now = Date.now()
    return [
      {
        id: 'pm-1',
        incidentId: 'inc-1',
        incidentTitle: 'Database Connection Pool Exhaustion',
        severity: 'critical',
        date: new Date(now - 604800000).toISOString(),
        status: 'published',
        author: 'alice@example.com',
        rootCause: 'Connection pool max size was insufficient for peak traffic load during deployment window',
        contributingFactors: [
          'No connection pool scaling metrics in alerting',
          'Deployment occurred during peak traffic window',
          'Connection timeout configuration was too high',
        ],
        lessonsLearned: [
          {
            category: 'technical',
            description: 'Connection pool sizing must account for peak load multipliers',
            action: 'Increase pool max from 50 to 150 connections',
          },
          {
            category: 'process',
            description: 'Deployments should avoid peak traffic hours',
            action: 'Update deployment schedule policy to restrict to off-peak windows',
          },
          {
            category: 'communication',
            description: 'On-call runbook for pool exhaustion was unclear',
            action: 'Create dedicated runbook with troubleshooting steps',
          },
        ],
        timeline: [
          { timestamp: new Date(now - 604800000).toISOString(), actor: 'monitoring', action: 'Alert triggered', details: 'Connection pool 95% utilized' },
          { timestamp: new Date(now - 604500000).toISOString(), actor: 'alice@example.com', action: 'Incident declared', details: 'Critical severity' },
          { timestamp: new Date(now - 604200000).toISOString(), actor: 'bob@example.com', action: 'Root cause identified', details: 'Pool max too low' },
          { timestamp: new Date(now - 603900000).toISOString(), actor: 'system', action: 'Mitigation applied', details: 'Increased pool size' },
        ],
        teamFeedback: 4.5,
        resolutionTime: 45,
      },
      {
        id: 'pm-2',
        incidentId: 'inc-2',
        incidentTitle: 'Certificate Expiration Warning',
        severity: 'high',
        date: new Date(now - 1209600000).toISOString(),
        status: 'published',
        author: 'bob@example.com',
        rootCause: 'Certificate renewal automation was disabled after infrastructure migration',
        contributingFactors: [
          'Renewal job was not re-enabled post-migration',
          'No alert for upcoming certificate expirations in new monitoring stack',
          'Manual renewal process documentation was outdated',
        ],
        lessonsLearned: [
          {
            category: 'technical',
            description: 'Automate certificate renewal checks in all environments',
            action: 'Deploy cert-manager with automated renewal to production',
          },
          {
            category: 'process',
            description: 'Post-migration runbooks must include critical service verification',
            action: 'Add certificate status check to migration checklist',
          },
        ],
        timeline: [
          { timestamp: new Date(now - 1209600000).toISOString(), actor: 'monitoring', action: 'Expiration detected', details: '7 days remaining' },
          { timestamp: new Date(now - 1209300000).toISOString(), actor: 'bob@example.com', action: 'Incident opened', details: 'High severity' },
          { timestamp: new Date(now - 1209000000).toISOString(), actor: 'system', action: 'Manual renewal triggered', details: 'Success' },
        ],
        teamFeedback: 3.8,
        resolutionTime: 120,
      },
      {
        id: 'pm-3',
        incidentId: 'inc-3',
        incidentTitle: 'Disk Space Critical on Storage Node',
        severity: 'high',
        date: new Date(now - 1814400000).toISOString(),
        status: 'published',
        author: 'charlie@example.com',
        rootCause: 'Backup retention policy was increasing disk usage faster than cleanup jobs could handle',
        contributingFactors: [
          'Backup retention increased to 90 days without capacity planning',
          'Cleanup job was running sequentially instead of parallel',
          'No predictive disk space alerting threshold',
        ],
        lessonsLearned: [
          {
            category: 'technical',
            description: 'Implement predictive alerting for disk usage trends',
            action: 'Deploy ML-based disk space forecasting with 30-day warnings',
          },
          {
            category: 'process',
            description: 'Capacity planning reviews required for policy changes',
            action: 'Establish policy change review board',
          },
        ],
        timeline: [
          { timestamp: new Date(now - 1814400000).toISOString(), actor: 'monitoring', action: 'Alert triggered', details: '5% disk free' },
          { timestamp: new Date(now - 1814100000).toISOString(), actor: 'charlie@example.com', action: 'Incident declared', details: 'High severity' },
          { timestamp: new Date(now - 1813800000).toISOString(), actor: 'system', action: 'Cleanup parallelized', details: 'Job execution time reduced' },
        ],
        teamFeedback: 4.2,
        resolutionTime: 50,
      },
      {
        id: 'pm-4',
        incidentId: 'inc-4',
        incidentTitle: 'High Error Rate on /api/deployments',
        severity: 'medium',
        date: new Date(now - 2419200000).toISOString(),
        status: 'published',
        author: 'alice@example.com',
        rootCause: 'Rate limiter tokens were being consumed by health check requests',
        contributingFactors: [
          'Health checks were not exempted from rate limiting',
          'Health check frequency was increased without updating rate limit config',
          'No separate rate limit bucket for health checks',
        ],
        lessonsLearned: [
          {
            category: 'technical',
            description: 'Separate rate limit buckets for health checks vs API requests',
            action: 'Implement internal request classification and routing',
          },
        ],
        timeline: [
          { timestamp: new Date(now - 2419200000).toISOString(), actor: 'monitoring', action: 'Error rate spike detected', details: '5.2% error rate' },
          { timestamp: new Date(now - 2419000000).toISOString(), actor: 'alice@example.com', action: 'Investigation started', details: 'Medium severity' },
          { timestamp: new Date(now - 2418700000).toISOString(), actor: 'system', action: 'Rate limiter config updated', details: 'Health checks exempted' },
        ],
        teamFeedback: 4.0,
        resolutionTime: 60,
      },
      {
        id: 'pm-5',
        incidentId: 'inc-5',
        incidentTitle: 'Backup Job Failure',
        severity: 'medium',
        date: new Date(now - 3024000000).toISOString(),
        status: 'draft',
        author: 'bob@example.com',
        rootCause: 'Network timeout during backup to object storage',
        contributingFactors: [
          'No retry logic with exponential backoff',
          'Network buffer timeout was too aggressive',
        ],
        lessonsLearned: [
          {
            category: 'technical',
            description: 'All critical jobs must have robust retry mechanisms',
            action: 'Implement exponential backoff with jitter for backup jobs',
          },
        ],
        timeline: [
          { timestamp: new Date(now - 3024000000).toISOString(), actor: 'system', action: 'Backup job started', details: 'Nightly run' },
          { timestamp: new Date(now - 3023700000).toISOString(), actor: 'system', action: 'Network timeout', details: 'Object storage unreachable' },
          { timestamp: new Date(now - 3023400000).toISOString(), actor: 'system', action: 'Job retried', details: 'Success' },
        ],
        teamFeedback: 3.5,
        resolutionTime: 120,
      },
      {
        id: 'pm-6',
        incidentId: 'inc-6',
        incidentTitle: 'Node Memory Leak Detected',
        severity: 'low',
        date: new Date(now - 3628800000).toISOString(),
        status: 'archived',
        author: 'charlie@example.com',
        rootCause: 'Goroutine not properly released in HTTP request handler',
        contributingFactors: [
          'Memory profiling was not part of regular testing',
          'Load testing did not run for extended periods',
        ],
        lessonsLearned: [
          {
            category: 'technical',
            description: 'Long-running memory profiling must be part of test suite',
            action: 'Add 48-hour memory profiling to CI/CD pipeline',
          },
        ],
        timeline: [
          { timestamp: new Date(now - 3628800000).toISOString(), actor: 'monitoring', action: 'Memory trend detected', details: 'Gradual increase' },
          { timestamp: new Date(now - 3628500000).toISOString(), actor: 'charlie@example.com', action: 'Investigation started', details: 'Low severity' },
          { timestamp: new Date(now - 3628200000).toISOString(), actor: 'system', action: 'Fix deployed', details: 'Goroutine released' },
        ],
        teamFeedback: 4.1,
        resolutionTime: 30,
      },
    ]
  }

  useEffect(() => {
    const loadPostmortems = async () => {
      try {
        setLoading(true)
        const mockPostmortems = generateMockPostmortems()
        setPostmortems(mockPostmortems)

        const allLessons = mockPostmortems.flatMap((p) => p.lessonsLearned)
        const criticalCount = allLessons.filter((l) => l.action).length

        const postmortemStats: PostmortemStats = {
          total: mockPostmortems.length,
          published: mockPostmortems.filter((p) => p.status === 'published').length,
          criticalLessons: criticalCount,
          avgFeedback: Math.round(
            (mockPostmortems.reduce((sum, p) => sum + p.teamFeedback, 0) /
              mockPostmortems.length) *
              10
          ) / 10,
        }
        setStats(postmortemStats)
      } catch (err) {
        console.error('Failed to load postmortems:', err)
      } finally {
        setLoading(false)
      }
    }

    loadPostmortems()
  }, [])

  if (loading) {
    return (
      <AppLayout title="Postmortems" subtitle="Incident postmortem reports and lessons learned">
        <Loading message="Loading postmortems..." />
      </AppLayout>
    )
  }

  const filteredPostmortems = postmortems.filter((p) =>
    selectedStatus === 'all' ? true : p.status === selectedStatus
  )

  return (
    <AppLayout
      title="Postmortems"
      subtitle={`${stats.published} published • ${stats.criticalLessons} action items • ${stats.avgFeedback}/5.0 avg feedback`}
    >
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Total Reports</p>
              <span className="text-3xl font-bold text-primary-500">{stats.total}</span>
              <p className="text-xs text-neutral-500 mt-2">postmortems</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Published</p>
              <span className="text-3xl font-bold text-success-500">{stats.published}</span>
              <p className="text-xs text-neutral-500 mt-2">finalized</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Action Items</p>
              <span className="text-3xl font-bold text-warning-500">{stats.criticalLessons}</span>
              <p className="text-xs text-neutral-500 mt-2">with owners</p>
            </div>
          </Card>

          <Card variant="glass">
            <div className="p-4">
              <p className="text-neutral-400 text-sm mb-2">Team Feedback</p>
              <span className="text-3xl font-bold text-info-500">{stats.avgFeedback}</span>
              <p className="text-xs text-neutral-500 mt-2">/ 5.0 average</p>
            </div>
          </Card>
        </div>

        {/* Postmortems List */}
        <Card variant="glass">
          <div className="p-6 border-b border-neutral-700">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-white">Postmortem Reports</h3>
              <Button variant="primary" size="sm">
                + New Report
              </Button>
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-2">Status</label>
              <select
                value={selectedStatus}
                onChange={(e) => setSelectedStatus(e.target.value)}
                className="w-full md:w-48 px-3 py-2 bg-neutral-700 border border-neutral-600 rounded-lg text-white text-sm focus:border-primary-500 focus:outline-none"
              >
                <option value="all">All Status</option>
                <option value="draft">Draft</option>
                <option value="published">Published</option>
                <option value="archived">Archived</option>
              </select>
            </div>
          </div>

          <div className="divide-y divide-neutral-700">
            {filteredPostmortems.length === 0 ? (
              <div className="p-6 text-center text-neutral-400">
                No postmortems with selected status
              </div>
            ) : (
              filteredPostmortems.map((pm) => (
                <div
                  key={pm.id}
                  className="p-6 hover:bg-neutral-800/20 transition cursor-pointer"
                  onClick={() =>
                    setExpandedPostmortem(expandedPostmortem === pm.id ? null : pm.id)
                  }
                >
                  <div className="flex justify-between items-start gap-4 mb-2">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <h4 className="text-white font-medium">{pm.incidentTitle}</h4>
                        <Badge
                          status={
                            pm.severity === 'critical'
                              ? 'error'
                              : pm.severity === 'high'
                                ? 'warning'
                                : pm.severity === 'medium'
                                  ? 'info'
                                  : 'default'
                          }
                        >
                          {pm.severity.charAt(0).toUpperCase() + pm.severity.slice(1)}
                        </Badge>
                      </div>
                      <div className="flex flex-wrap gap-4 text-xs text-neutral-500">
                        <span>Report Date: {new Date(pm.date).toLocaleDateString()}</span>
                        <span>Author: {pm.author}</span>
                        <span>Lessons Learned: {pm.lessonsLearned.length}</span>
                      </div>
                    </div>
                    <Badge
                      status={
                        pm.status === 'draft'
                          ? 'warning'
                          : pm.status === 'published'
                            ? 'success'
                            : 'default'
                      }
                    >
                      {pm.status.charAt(0).toUpperCase() + pm.status.slice(1)}
                    </Badge>
                  </div>

                  {expandedPostmortem === pm.id && (
                    <div className="mt-4 p-4 bg-neutral-800/30 rounded-lg space-y-4 text-sm">
                      <div>
                        <p className="text-neutral-400 font-medium mb-2">Root Cause</p>
                        <p className="text-neutral-200">{pm.rootCause}</p>
                      </div>

                      <div>
                        <p className="text-neutral-400 font-medium mb-2">Contributing Factors</p>
                        <ul className="list-disc list-inside space-y-1 text-neutral-300">
                          {pm.contributingFactors.map((factor, idx) => (
                            <li key={idx}>{factor}</li>
                          ))}
                        </ul>
                      </div>

                      <div>
                        <p className="text-neutral-400 font-medium mb-2">Lessons Learned</p>
                        <div className="space-y-2">
                          {pm.lessonsLearned.map((lesson, idx) => (
                            <div key={idx} className="p-2 bg-neutral-700/50 rounded">
                              <div className="flex items-center gap-2 mb-1">
                                <span className="px-2 py-0.5 bg-neutral-600 rounded text-xs">
                                  {lesson.category}
                                </span>
                                {lesson.action && (
                                  <span className="text-success-400 text-xs font-medium">
                                    ✓ Action assigned
                                  </span>
                                )}
                              </div>
                              <p className="text-neutral-200 mb-1">{lesson.description}</p>
                              {lesson.action && (
                                <p className="text-success-300 text-xs">Action: {lesson.action}</p>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>

                      <div>
                        <p className="text-neutral-400 font-medium mb-2">Timeline</p>
                        <div className="space-y-1 text-xs">
                          {pm.timeline.map((event, idx) => (
                            <div key={idx} className="flex gap-2">
                              <span className="text-neutral-500 min-w-40">
                                {new Date(event.timestamp).toLocaleTimeString()}
                              </span>
                              <span className="text-neutral-300">
                                <strong>{event.action}</strong> by {event.actor}
                              </span>
                              {event.details && (
                                <span className="text-neutral-500">({event.details})</span>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>

                      <div className="flex gap-4 pt-2 text-xs">
                        <span className="text-neutral-400">
                          Team Feedback: <span className="text-info-400">{pm.teamFeedback}/5.0</span>
                        </span>
                        <span className="text-neutral-400">
                          Resolution Time: <span className="text-warning-400">{pm.resolutionTime} min</span>
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </Card>
      </div>
    </AppLayout>
  )
}

export default Postmortems
