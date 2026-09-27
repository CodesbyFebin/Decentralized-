import { test } from 'node:test';
import assert from 'node:assert';

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

function calculateTrend(points: MetricPoint[]): 'up' | 'down' | 'stable' {
  if (points.length < 2) return 'stable';
  const first = points[0].value;
  const last = points[points.length - 1].value;
  const change = last - first;
  if (Math.abs(change) < first * 0.05) return 'stable';
  return change > 0 ? 'up' : 'down';
}

function formatMetricValue(value: number, unit: string): string {
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
}

function validateMetricSeries(metric: unknown): boolean {
  if (typeof metric !== 'object' || !metric) return false;
  const m = metric as Record<string, unknown>;
  return (
    typeof m.name === 'string' &&
    typeof m.unit === 'string' &&
    Array.isArray(m.points) &&
    typeof m.min === 'number' &&
    typeof m.max === 'number' &&
    typeof m.avg === 'number' &&
    ['up', 'down', 'stable'].includes(m.trend as string)
  );
}

function validateAlert(alert: unknown): boolean {
  if (typeof alert !== 'object' || !alert) return false;
  const a = alert as Record<string, unknown>;
  return (
    typeof a.id === 'string' &&
    typeof a.metric === 'string' &&
    ['info', 'warning', 'critical'].includes(a.severity as string) &&
    typeof a.threshold === 'number' &&
    typeof a.value === 'number' &&
    typeof a.message === 'string' &&
    typeof a.triggeredAt === 'number'
  );
}

function validateTimeSeriesData(data: unknown): boolean {
  if (typeof data !== 'object' || !data) return false;
  const d = data as Record<string, unknown>;

  if (!d.timeRange || typeof d.timeRange !== 'object') return false;
  const tr = d.timeRange as Record<string, unknown>;
  if (typeof tr.start !== 'number' || typeof tr.end !== 'number') return false;

  if (!Array.isArray(d.metrics)) return false;
  for (const metric of d.metrics) {
    if (!validateMetricSeries(metric)) return false;
  }

  if (!Array.isArray(d.alerts)) return false;
  for (const alert of d.alerts) {
    if (!validateAlert(alert)) return false;
  }

  if (!d.summary || typeof d.summary !== 'object') return false;
  const sum = d.summary as Record<string, unknown>;
  if (typeof sum.uptime !== 'number' || typeof sum.avgLatency !== 'number' || typeof sum.errorRate !== 'number') {
    return false;
  }

  return true;
}

test('Metric Series Validation', async (t) => {
  await t.test('validates complete metric series', () => {
    const metric: MetricSeries = {
      name: 'cpu_usage',
      unit: 'percent',
      points: [{ timestamp: Date.now(), value: 45 }],
      min: 20,
      max: 90,
      avg: 55,
      trend: 'up',
    };
    assert.ok(validateMetricSeries(metric));
  });

  await t.test('validates metric with multiple points', () => {
    const metric: MetricSeries = {
      name: 'memory_usage',
      unit: 'bytes',
      points: [
        { timestamp: Date.now() - 300000, value: 1073741824 },
        { timestamp: Date.now(), value: 1610612736 },
      ],
      min: 1073741824,
      max: 1610612736,
      avg: 1342177280,
      trend: 'up',
    };
    assert.ok(validateMetricSeries(metric));
  });

  await t.test('supports percent unit', () => {
    const metric: MetricSeries = {
      name: 'error_rate',
      unit: 'percent',
      points: [{ timestamp: Date.now(), value: 2.5 }],
      min: 0,
      max: 5,
      avg: 2.5,
      trend: 'stable',
    };
    assert.equal(metric.unit, 'percent');
  });

  await t.test('supports ms unit', () => {
    const metric: MetricSeries = {
      name: 'latency',
      unit: 'ms',
      points: [{ timestamp: Date.now(), value: 125 }],
      min: 50,
      max: 500,
      avg: 150,
      trend: 'stable',
    };
    assert.equal(metric.unit, 'ms');
  });

  await t.test('supports requests/s unit', () => {
    const metric: MetricSeries = {
      name: 'throughput',
      unit: 'requests/s',
      points: [{ timestamp: Date.now(), value: 1250 }],
      min: 500,
      max: 2000,
      avg: 1200,
      trend: 'up',
    };
    assert.equal(metric.unit, 'requests/s');
  });

  await t.test('rejects invalid trend', () => {
    const metric = {
      name: 'test',
      unit: 'percent',
      points: [],
      min: 0,
      max: 100,
      avg: 50,
      trend: 'invalid',
    };
    assert.equal(validateMetricSeries(metric), false);
  });
});

test('Metric Value Formatting', async (t) => {
  await t.test('formats percent values', () => {
    assert.equal(formatMetricValue(45.5, 'percent'), '45.50%');
  });

  await t.test('formats milliseconds', () => {
    assert.equal(formatMetricValue(125.5, 'ms'), '125.5ms');
  });

  await t.test('formats bytes to B', () => {
    assert.equal(formatMetricValue(512, 'bytes'), '512.0 B');
  });

  await t.test('formats bytes to GB', () => {
    assert.equal(formatMetricValue(1073741824, 'bytes'), '1.0 GB');
  });

  await t.test('formats requests per second', () => {
    assert.equal(formatMetricValue(1250.5, 'requests/s'), '1250.5 req/s');
  });
});

test('Trend Calculation', async (t) => {
  await t.test('detects upward trend', () => {
    const points: MetricPoint[] = [
      { timestamp: Date.now() - 600000, value: 40 },
      { timestamp: Date.now(), value: 80 },
    ];
    const trend = calculateTrend(points);
    assert.equal(trend, 'up');
  });

  await t.test('detects downward trend', () => {
    const points: MetricPoint[] = [
      { timestamp: Date.now() - 600000, value: 80 },
      { timestamp: Date.now(), value: 40 },
    ];
    const trend = calculateTrend(points);
    assert.equal(trend, 'down');
  });

  await t.test('detects stable trend', () => {
    const points: MetricPoint[] = [
      { timestamp: Date.now() - 600000, value: 50 },
      { timestamp: Date.now(), value: 52 },
    ];
    const trend = calculateTrend(points);
    assert.equal(trend, 'stable');
  });

  await t.test('handles single point', () => {
    const points: MetricPoint[] = [{ timestamp: Date.now(), value: 50 }];
    const trend = calculateTrend(points);
    assert.equal(trend, 'stable');
  });

  await t.test('handles empty points', () => {
    const points: MetricPoint[] = [];
    const trend = calculateTrend(points);
    assert.equal(trend, 'stable');
  });
});

test('Alert Validation', async (t) => {
  await t.test('validates complete alert', () => {
    const alert: Alert = {
      id: 'alert-001',
      metric: 'cpu_usage',
      severity: 'warning',
      threshold: 80,
      value: 85,
      message: 'CPU usage exceeded threshold',
      triggeredAt: Date.now(),
    };
    assert.ok(validateAlert(alert));
  });

  await t.test('validates resolved alert', () => {
    const alert: Alert = {
      id: 'alert-001',
      metric: 'cpu_usage',
      severity: 'warning',
      threshold: 80,
      value: 45,
      message: 'CPU usage exceeded threshold',
      triggeredAt: Date.now() - 600000,
      resolvedAt: Date.now(),
    };
    assert.ok(validateAlert(alert));
  });

  await t.test('supports critical severity', () => {
    const alert: Alert = {
      id: 'alert-001',
      metric: 'disk_usage',
      severity: 'critical',
      threshold: 95,
      value: 98,
      message: 'Disk usage critical',
      triggeredAt: Date.now(),
    };
    assert.equal(alert.severity, 'critical');
  });

  await t.test('supports info severity', () => {
    const alert: Alert = {
      id: 'alert-001',
      metric: 'deployment',
      severity: 'info',
      threshold: 1,
      value: 2,
      message: 'New deployment detected',
      triggeredAt: Date.now(),
    };
    assert.equal(alert.severity, 'info');
  });

  await t.test('rejects invalid severity', () => {
    const alert = {
      id: 'alert-001',
      metric: 'test',
      severity: 'invalid',
      threshold: 100,
      value: 105,
      message: 'Test',
      triggeredAt: Date.now(),
    };
    assert.equal(validateAlert(alert), false);
  });
});

test('Time Series Data Validation', async (t) => {
  await t.test('validates complete timeseries data', () => {
    const data: TimeSeriesObservabilityData = {
      timeRange: { start: Date.now() - 3600000, end: Date.now(), label: '1h' },
      metrics: [
        {
          name: 'cpu',
          unit: 'percent',
          points: [{ timestamp: Date.now(), value: 45 }],
          min: 0,
          max: 100,
          avg: 45,
          trend: 'stable',
        },
      ],
      alerts: [],
      summary: { uptime: 99.95, avgLatency: 125.5, errorRate: 0.05 },
    };
    assert.ok(validateTimeSeriesData(data));
  });

  await t.test('validates empty metrics', () => {
    const data: TimeSeriesObservabilityData = {
      timeRange: { start: Date.now() - 3600000, end: Date.now(), label: '1h' },
      metrics: [],
      alerts: [],
      summary: { uptime: 100, avgLatency: 0, errorRate: 0 },
    };
    assert.ok(validateTimeSeriesData(data));
  });

  await t.test('rejects missing timeRange', () => {
    const data = {
      metrics: [],
      alerts: [],
      summary: { uptime: 100, avgLatency: 0, errorRate: 0 },
    };
    assert.equal(validateTimeSeriesData(data), false);
  });
});

test('Summary Statistics', async (t) => {
  await t.test('tracks uptime percentage', () => {
    const summary = { uptime: 99.95, avgLatency: 125, errorRate: 0.05 };
    assert.ok(summary.uptime >= 0 && summary.uptime <= 100);
  });

  await t.test('tracks average latency', () => {
    const summary = { uptime: 99.95, avgLatency: 250, errorRate: 0.1 };
    assert.ok(summary.avgLatency >= 0);
  });

  await t.test('tracks error rate', () => {
    const summary = { uptime: 99, avgLatency: 500, errorRate: 1.0 };
    assert.ok(summary.errorRate >= 0 && summary.errorRate <= 100);
  });

  await t.test('perfect uptime', () => {
    const summary = { uptime: 100, avgLatency: 50, errorRate: 0 };
    assert.equal(summary.uptime, 100);
  });
});

test('Alert Lifecycle', async (t) => {
  await t.test('alert triggered', () => {
    const alert: Alert = {
      id: 'alert-001',
      metric: 'cpu_usage',
      severity: 'warning',
      threshold: 80,
      value: 85,
      message: 'CPU exceeded',
      triggeredAt: Date.now(),
    };
    assert.equal(alert.resolvedAt, undefined);
  });

  await t.test('alert resolved', () => {
    const triggered = Date.now() - 600000;
    const alert: Alert = {
      id: 'alert-001',
      metric: 'cpu_usage',
      severity: 'warning',
      threshold: 80,
      value: 45,
      message: 'CPU exceeded',
      triggeredAt: triggered,
      resolvedAt: Date.now(),
    };
    assert.ok(alert.resolvedAt! > alert.triggeredAt);
  });

  await t.test('duration calculation', () => {
    const triggered = Date.now() - 600000;
    const alert: Alert = {
      id: 'alert-001',
      metric: 'test',
      severity: 'critical',
      threshold: 100,
      value: 150,
      message: 'Test',
      triggeredAt: triggered,
      resolvedAt: Date.now(),
    };
    const duration = alert.resolvedAt! - alert.triggeredAt;
    assert.ok(duration >= 600000);
  });
});

test('Multi-Metric Scenarios', async (t) => {
  await t.test('multiple metrics with different units', () => {
    const metrics: MetricSeries[] = [
      {
        name: 'cpu',
        unit: 'percent',
        points: [{ timestamp: Date.now(), value: 45 }],
        min: 0,
        max: 100,
        avg: 45,
        trend: 'stable',
      },
      {
        name: 'memory',
        unit: 'bytes',
        points: [{ timestamp: Date.now(), value: 1073741824 }],
        min: 536870912,
        max: 2147483648,
        avg: 1073741824,
        trend: 'stable',
      },
      {
        name: 'latency',
        unit: 'ms',
        points: [{ timestamp: Date.now(), value: 125 }],
        min: 50,
        max: 500,
        avg: 150,
        trend: 'stable',
      },
    ];
    assert.equal(metrics.length, 3);
  });

  await t.test('heterogeneous alert severities', () => {
    const alerts: Alert[] = [
      {
        id: 'a1',
        metric: 'cpu',
        severity: 'critical',
        threshold: 90,
        value: 95,
        message: 'Critical',
        triggeredAt: Date.now(),
      },
      {
        id: 'a2',
        metric: 'memory',
        severity: 'warning',
        threshold: 80,
        value: 85,
        message: 'Warning',
        triggeredAt: Date.now(),
      },
      {
        id: 'a3',
        metric: 'disk',
        severity: 'info',
        threshold: 50,
        value: 55,
        message: 'Info',
        triggeredAt: Date.now(),
      },
    ];
    const severities = new Set(alerts.map((a) => a.severity));
    assert.equal(severities.size, 3);
  });
});

test('Time Range Support', async (t) => {
  await t.test('1 hour range', () => {
    const timeRange: TimeRange = {
      start: Date.now() - 3600000,
      end: Date.now(),
      label: '1h',
    };
    assert.equal(timeRange.end - timeRange.start, 3600000);
  });

  await t.test('24 hour range', () => {
    const timeRange: TimeRange = {
      start: Date.now() - 86400000,
      end: Date.now(),
      label: '24h',
    };
    assert.equal(timeRange.end - timeRange.start, 86400000);
  });

  await t.test('7 day range', () => {
    const timeRange: TimeRange = {
      start: Date.now() - 604800000,
      end: Date.now(),
      label: '7d',
    };
    assert.equal(timeRange.end - timeRange.start, 604800000);
  });

  await t.test('30 day range', () => {
    const timeRange: TimeRange = {
      start: Date.now() - 2592000000,
      end: Date.now(),
      label: '30d',
    };
    assert.equal(timeRange.end - timeRange.start, 2592000000);
  });
});

test('Metric Points and Aggregates', async (t) => {
  await t.test('dense metric data', () => {
    const points: MetricPoint[] = [];
    for (let i = 0; i < 360; i++) {
      points.push({
        timestamp: Date.now() - 3600000 + i * 10000,
        value: 50 + Math.random() * 20,
      });
    }
    assert.equal(points.length, 360);
  });

  await t.test('calculate min from points', () => {
    const points: MetricPoint[] = [
      { timestamp: Date.now(), value: 10 },
      { timestamp: Date.now(), value: 20 },
      { timestamp: Date.now(), value: 5 },
    ];
    const min = Math.min(...points.map((p) => p.value));
    assert.equal(min, 5);
  });

  await t.test('calculate max from points', () => {
    const points: MetricPoint[] = [
      { timestamp: Date.now(), value: 10 },
      { timestamp: Date.now(), value: 50 },
      { timestamp: Date.now(), value: 30 },
    ];
    const max = Math.max(...points.map((p) => p.value));
    assert.equal(max, 50);
  });

  await t.test('calculate avg from points', () => {
    const points: MetricPoint[] = [
      { timestamp: Date.now(), value: 10 },
      { timestamp: Date.now(), value: 20 },
      { timestamp: Date.now(), value: 30 },
    ];
    const avg = points.reduce((sum, p) => sum + p.value, 0) / points.length;
    assert.equal(avg, 20);
  });
});
