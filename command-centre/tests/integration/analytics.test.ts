import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('Analytics Operations: Metrics & Telemetry', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should fetch analytics overview', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      assert.ok(res.json?.data?.metrics);
      assert.ok(res.json?.data?.edges);
      assert.ok(res.json?.data?.replicaHealth);
    });

    await t.test('should return resource usage metrics', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const data = res.json?.data;
      assert.ok(Array.isArray(data?.hosts));
      data?.hosts.forEach((h: any) => {
        assert.ok(typeof h.name === 'string');
        if (h.cpu) {
          assert.ok(typeof h.cpu.used === 'number');
          assert.ok(typeof h.cpu.limit === 'number');
        }
      });
    });

    await t.test('should include application metrics', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const appMetrics = res.json?.data?.appMetrics;
      if (appMetrics) {
        assert.ok(Array.isArray(appMetrics));
        appMetrics.forEach((a: any) => {
          assert.ok(typeof a.app === 'string');
          assert.ok(typeof a.replicas === 'number');
          assert.ok(typeof a.desiredReplicas === 'number');
        });
      }
    });

    await t.test('should provide node efficiency data', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const efficiency = res.json?.data?.nodeEfficiency;
      if (efficiency) {
        assert.ok(Array.isArray(efficiency));
        efficiency.forEach((n: any) => {
          assert.ok(typeof n.node === 'string');
          assert.ok(typeof n.packRatio === 'number');
          assert.ok(typeof n.strain === 'number');
        });
      }
    });

    await t.test('should track failure metrics', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const failures = res.json?.data?.failures;
      if (failures) {
        assert.ok(Array.isArray(failures));
        failures.forEach((f: any) => {
          assert.ok(typeof f.service === 'string');
          assert.ok(typeof f.mtbf === 'number');
          assert.ok(typeof f.meanRecoveryTime === 'number');
        });
      }
    });

    await t.test('should provide cost projection', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const cost = res.json?.data?.costProjection;
      if (cost) {
        assert.ok(typeof cost.monthlyUsd === 'number');
        assert.ok(typeof cost.perReplica === 'number');
        assert.ok(typeof cost.perGb === 'number');
      }
    });

    await t.test('should return health check data', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const health = res.json?.data?.replicaHealth;
      assert.ok(Array.isArray(health));
      health.forEach((r: any) => {
        assert.ok(typeof r.assignment === 'string');
        assert.ok(typeof r.latencyUs === 'number');
        assert.ok(typeof r.ok === 'boolean');
      });
    });

    await t.test('should return edge host telemetry', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const edges = res.json?.data?.edges;
      assert.ok(Array.isArray(edges));
      edges.forEach((e: any) => {
        assert.ok(typeof e.name === 'string');
        assert.ok(typeof e.requests === 'number');
        assert.ok(typeof e.errors === 'number');
        assert.ok(typeof e.routes === 'number');
      });
    });

    await t.test('should handle empty analytics gracefully', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      assert.ok(res.json?.data?.metrics);
      // Empty arrays are acceptable
      assert.ok(Array.isArray(res.json?.data?.edges));
    });

    await t.test('should track edge error rates', async () => {
      const res = await call('GET', '/analytics', { csrf: false });
      assert.equal(res.status, 200);
      const edges = res.json?.data?.edges;
      if (edges && edges.length > 0) {
        const edge = edges[0];
        if (edge.requests > 0) {
          const errorRate = (edge.errors / edge.requests) * 100;
          assert.ok(errorRate >= 0 && errorRate <= 100);
        }
      }
    });

  } finally {
    await close();
  }
});
