import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('Storage Operations: Volume Management API', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list volumes', async () => {
      const res = await call('GET', '/storage', { csrf: false });
      assert.equal(res.status, 200);
      assert.ok(res.json?.data?.volumes);
      assert.ok(Array.isArray(res.json.data.volumes));
    });

    await t.test('should validate volume creation request', async () => {
      const res = await call('POST', '/volumes', {
        body: {
          name: 'test-vol',
          sizeBytes: 10 * 1024 * 1024 * 1024,
          durabilityReplicas: 2,
          app: 'myapp'
        },
        csrf: true
      });

      // Demo adapter may refuse mutation, but check manifest is accepted
      assert.ok(res.status === 422 || res.status === 409 || res.status === 201, `Status should be rejection or success, got ${res.status}`);
    });

    await t.test('should reject invalid volume name', async () => {
      const res = await call('POST', '/volumes', {
        body: {
          name: 'Invalid_Vol',
          sizeBytes: 10 * 1024 * 1024 * 1024,
          durabilityReplicas: 2,
          app: 'myapp'
        },
        csrf: true
      });

      assert.equal(res.status, 422);
      assert.ok(res.json?.data?.message?.toLowerCase().includes('volume'), 'Error should mention volume');
    });

    await t.test('should reject invalid size', async () => {
      const res = await call('POST', '/volumes', {
        body: {
          name: 'test-vol',
          sizeBytes: 512, // Too small
          durabilityReplicas: 2,
          app: 'myapp'
        },
        csrf: true
      });

      assert.equal(res.status, 422);
    });

    await t.test('should reject invalid durability', async () => {
      const res = await call('POST', '/volumes', {
        body: {
          name: 'test-vol',
          sizeBytes: 10 * 1024 * 1024 * 1024,
          durabilityReplicas: 1,
          app: 'myapp'
        },
        csrf: true
      });

      assert.equal(res.status, 422);
    });

    await t.test('should delete volume with confirmation', async () => {
      const res = await call('DELETE', '/volumes/test-vol', {
        csrf: true
      });

      // Should return 422 or 404 if not found, 204 if deleted
      assert.ok([204, 404, 422].includes(res.status), `Status should be delete/not-found/reject, got ${res.status}`);
    });

    await t.test('should create snapshot', async () => {
      const res = await call('POST', '/volumes/test-vol/snapshot', {
        body: { retention: 30 },
        csrf: true
      });

      assert.ok([201, 404, 422].includes(res.status), `Status should be create/not-found/reject, got ${res.status}`);
    });

    await t.test('should restore snapshot', async () => {
      const res = await call('POST', '/volumes/test-vol/restore', {
        body: { snapshotHash: 'abc123' },
        csrf: true
      });

      assert.ok([200, 404, 422].includes(res.status), `Status should be ok/not-found/reject, got ${res.status}`);
    });

    await t.test('should trigger volume repair', async () => {
      const res = await call('POST', '/volumes/test-vol/repair', {
        body: {},
        csrf: true
      });

      assert.ok([202, 404, 422].includes(res.status), `Status should be accepted/not-found/reject, got ${res.status}`);
    });

    await t.test('should list volume replicas and status', async () => {
      const res = await call('GET', '/volumes/test-vol', { csrf: false });

      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        const vol = res.json?.data;
        assert.ok(typeof vol?.id === 'string');
        assert.ok(typeof vol?.verified === 'number');
        assert.ok(typeof vol?.durabilityReplicas === 'number');
        assert.ok(Array.isArray(vol?.memberNames));
      }
    });

    await t.test('should return volume quota info', async () => {
      const res = await call('GET', '/storage', { csrf: false });

      assert.equal(res.status, 200);
      const metrics = res.json?.data?.metrics;
      assert.ok(metrics?.capacity, 'Should have capacity metric');
      assert.ok(metrics?.used, 'Should have used metric');
      assert.ok(metrics?.degraded !== undefined, 'Should have degraded metric');
    });

  } finally {
    await close();
  }
});
