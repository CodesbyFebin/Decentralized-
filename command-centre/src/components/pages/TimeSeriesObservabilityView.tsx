import React, { useState } from 'react';
import { TrendingUp, TrendingDown, Clock, Activity, AlertCircle, CheckCircle2, Zap, BarChart3, ChevronDown, ChevronUp } from 'lucide-react';
import { useSession } from '../../lib/session';
import { Glass, PanelHeader, IconTile, StatusPill } from '../common/ui';
import { Unavailable, Note, since } from '../common/states';
import { useResource } from '../../lib/useResource';
import { api, ApiError } from '../../lib/api';

interface MetricPoint {
  timestamp: number;
  value: number;
}

interface MetricSeries {
  name: string;
  unit: string;
  points: MetricPoint[];
  min: number;
  max: number;
  avg: number;
  trend: 'up' | 'down' | 'stable';
}

interface Alert {
  id: string;
  metric: string;
  severity: 'info' | 'warning' | 'critical';
  threshold: number;
  value: number;
  message: string;
  triggeredAt: number;
  resolvedAt?: number;
}

interface TimeRange {
  start: number;
  end: number;
  label: string;
}

interface TimeSeriesObservabilityData {
  timeRange: TimeRange;
  metrics: MetricSeries[];
  alerts: Alert[];
  summary: {
    uptime: number;
    avgLatency: number;
    errorRate: number;
  };
}

export default function TimeSeriesObservabilityView() {
  const { session, capabilities, can } = useSession();
  const tsRes = useResource<TimeSeriesObservabilityData>('/observability/timeseries');
  const [expandedMetrics, setExpandedMetrics] = useState<Record<string, boolean>>({});
  const [selectedTimeRange, setSelectedTimeRange] = useState('1h');
  const [alertFilter, setAlertFilter] = useState<'all' | 'open' | 'resolved'>('all');

  const isSelfHosted = capabilities?.backend.reachable === false;

  const timeRanges: Record<string, number> = {
    '1h': 3600000,
    '6h': 21600000,
    '24h': 86400000,
    '7d': 604800000,
    '30d': 2592000000,
  };

  const toggleMetric = (name: string) => {
    setExpandedMetrics((prev) => ({ ...prev, [name]: !prev[name] }));
  };

  const formatMetricValue = (value: number, unit: string): string => {
    if (unit === 'percent') return `${value.toFixed(2)}%`;
    if (unit === 'ms') return `${value.toFixed(1)}ms`;
    if (unit === 'bytes') {
      const units = ['B', 'KB', 'MB', 'GB'];
      let size = value;
      let unitIndex = 0;
      while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024;
        unitIndex++;
      }
      return `${size.toFixed(1)} ${units[unitIndex]}`;
    }
    if (unit === 'requests/s') return `${value.toFixed(1)} req/s`;
    return `${value.toFixed(2)} ${unit}`;
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up':
        return <TrendingUp className="w-4 h-4 text-rose-400" />;
      case 'down':
        return <TrendingDown className="w-4 h-4 text-emerald-400" />;
      case 'stable':
        return <Zap className="w-4 h-4 text-blue-400" />;
      default:
        return null;
    }
  };

  const getAlertSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'rose';
      case 'warning':
        return 'amber';
      case 'info':
        return 'blue';
      default:
        return 'slate';
    }
  };

  const formatUptime = (percent: number): string => {
    return `${percent.toFixed(3)}%`;
  };

  if (isSelfHosted) {
    return (
      <div className="space-y-4 pt-2 max-w-3xl">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Time-Series Observability</h1>
          <p className="text-[13px] text-slate-400">Self-hosted deployment.</p>
        </div>
        <Glass className="p-5 flex items-start gap-4">
          <IconTile tone="teal" size="lg">
            <BarChart3 className="w-6 h-6" />
          </IconTile>
          <div>
            <div className="text-[15px] font-semibold text-white">Observability not configured</div>
            <p className="mt-1 text-[13px] text-slate-300">Self-hosted deployments use local monitoring. This service is for multi-tenant metrics aggregation and trend analysis.</p>
          </div>
        </Glass>
        <Note>Time-series observability tracks metrics over time, detects trends, and triggers alerts based on thresholds for proactive incident management.</Note>
      </div>
    );
  }

  return (
    <div className="space-y-4 pt-2 max-w-5xl">
      <div>
        <h1 className="text-3xl font-extrabold text-white tracking-tight">Time-Series Observability</h1>
        <p className="text-[13px] text-slate-400">Historical metrics, trend analysis, and alert tracking.</p>
      </div>

      {tsRes.loading ? (
        <Glass className="p-5">
          <p className="text-slate-400">Loading observability data...</p>
        </Glass>
      ) : tsRes.data ? (
        <>
          {/* Time Range Selector */}
          <Glass className="p-5">
            <PanelHeader title="Time range" subtitle={`Last ${selectedTimeRange}`} />
            <div className="mt-4 flex gap-2 flex-wrap">
              {Object.keys(timeRanges).map((range) => (
                <button
                  key={range}
                  onClick={() => setSelectedTimeRange(range)}
                  className={`px-3 py-2 text-sm rounded font-medium transition ${
                    selectedTimeRange === range
                      ? 'bg-teal-600 text-white'
                      : 'bg-white/10 hover:bg-white/20 text-slate-200'
                  }`}
                >
                  {range}
                </button>
              ))}
            </div>
          </Glass>

          {/* Summary */}
          <Glass className="p-5">
            <PanelHeader icon={<IconTile tone="teal" size="sm"><Activity className="w-4 h-4" /></IconTile>} title="Summary" />
            <div className="mt-4 grid sm:grid-cols-3 gap-4">
              <div className="p-3 bg-white/5 rounded">
                <div className="text-[12px] text-slate-400 mb-1">Uptime</div>
                <div className="text-xl font-semibold text-emerald-200">{formatUptime(tsRes.data.summary.uptime)}</div>
              </div>
              <div className="p-3 bg-white/5 rounded">
                <div className="text-[12px] text-slate-400 mb-1">Avg Latency</div>
                <div className="text-xl font-semibold text-cyan-200">{formatMetricValue(tsRes.data.summary.avgLatency, 'ms')}</div>
              </div>
              <div className="p-3 bg-white/5 rounded">
                <div className="text-[12px] text-slate-400 mb-1">Error Rate</div>
                <div className="text-xl font-semibold text-rose-200">{formatMetricValue(tsRes.data.summary.errorRate, 'percent')}</div>
              </div>
            </div>
          </Glass>

          {/* Metrics */}
          {tsRes.data.metrics.length > 0 && (
            <Glass className="p-5">
              <PanelHeader title="Metrics" subtitle={`${tsRes.data.metrics.length} metric${tsRes.data.metrics.length !== 1 ? 's' : ''} tracked`} />
              <div className="mt-3 space-y-2">
                {tsRes.data.metrics.map((metric) => (
                  <div key={metric.name} className="border border-white/5 rounded-lg overflow-hidden">
                    <div
                      className="flex items-center justify-between p-3 bg-white/3 hover:bg-white/5 cursor-pointer"
                      onClick={() => toggleMetric(metric.name)}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="font-semibold text-slate-100">{metric.name}</span>
                          {getTrendIcon(metric.trend)}
                        </div>
                        <div className="text-[11px] text-slate-400">
                          Current: {formatMetricValue(metric.points[metric.points.length - 1]?.value || 0, metric.unit)} ·
                          Avg: {formatMetricValue(metric.avg, metric.unit)}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="text-right text-[11px]">
                          <div className="text-emerald-300">{formatMetricValue(metric.min, metric.unit)}</div>
                          <div className="text-slate-400">{formatMetricValue(metric.max, metric.unit)}</div>
                        </div>
                        {expandedMetrics[metric.name] ? (
                          <ChevronUp className="w-4 h-4 text-slate-400" />
                        ) : (
                          <ChevronDown className="w-4 h-4 text-slate-400" />
                        )}
                      </div>
                    </div>

                    {expandedMetrics[metric.name] && (
                      <div className="p-3 bg-white/2 border-t border-white/5 space-y-2 text-[12px]">
                        <div className="grid sm:grid-cols-4 gap-2">
                          <div>
                            <span className="text-slate-400">Min</span>
                            <div className="text-slate-100 font-mono">{formatMetricValue(metric.min, metric.unit)}</div>
                          </div>
                          <div>
                            <span className="text-slate-400">Max</span>
                            <div className="text-slate-100 font-mono">{formatMetricValue(metric.max, metric.unit)}</div>
                          </div>
                          <div>
                            <span className="text-slate-400">Avg</span>
                            <div className="text-slate-100 font-mono">{formatMetricValue(metric.avg, metric.unit)}</div>
                          </div>
                          <div>
                            <span className="text-slate-400">Points</span>
                            <div className="text-slate-100 font-mono">{metric.points.length}</div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </Glass>
          )}

          {/* Alerts */}
          {tsRes.data.alerts.length > 0 && (
            <Glass className="p-5">
              <div className="flex items-center justify-between mb-4">
                <PanelHeader title="Alerts" subtitle={`${tsRes.data.alerts.length} alert${tsRes.data.alerts.length !== 1 ? 's' : ''}`} />
                <div className="flex gap-2">
                  {(['all', 'open', 'resolved'] as const).map((filter) => (
                    <button
                      key={filter}
                      onClick={() => setAlertFilter(filter)}
                      className={`px-2 py-1 text-[11px] rounded font-medium transition ${
                        alertFilter === filter ? 'bg-teal-600/30 text-teal-200' : 'bg-white/5 text-slate-400'
                      }`}
                    >
                      {filter}
                    </button>
                  ))}
                </div>
              </div>
              <div className="space-y-2">
                {tsRes.data.alerts
                  .filter((alert) => {
                    if (alertFilter === 'open') return !alert.resolvedAt;
                    if (alertFilter === 'resolved') return alert.resolvedAt;
                    return true;
                  })
                  .map((alert) => (
                    <div
                      key={alert.id}
                      className={`p-3 rounded-lg border ${
                        alert.resolvedAt
                          ? 'bg-emerald-500/5 border-emerald-400/20'
                          : 'bg-amber-500/5 border-amber-400/20'
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        {alert.resolvedAt ? (
                          <CheckCircle2 className="w-5 h-5 text-emerald-400 flex-shrink-0 mt-0.5" />
                        ) : (
                          <AlertCircle className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
                        )}
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-semibold text-slate-100">{alert.metric}</span>
                            <StatusPill
                              status={alert.severity}
                              label={alert.severity.charAt(0).toUpperCase() + alert.severity.slice(1)}
                            />
                          </div>
                          <p className="text-[12px] text-slate-300 mb-1">{alert.message}</p>
                          <div className="text-[11px] text-slate-400">
                            Threshold: {alert.threshold} · Current: {alert.value} ·
                            {alert.resolvedAt ? ` Resolved ${since(alert.resolvedAt)}` : ` Triggered ${since(alert.triggeredAt)}`}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
              </div>
            </Glass>
          )}
        </>
      ) : null}

      <Note>Time-series observability tracks metrics over time, detects trends and anomalies, and triggers alerts based on configurable thresholds for proactive monitoring and incident response.</Note>
    </div>
  );
}
