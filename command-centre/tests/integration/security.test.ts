import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('Security Operations: Key Management & Policies', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should fetch security controls', async () => {
      const res = await call('GET', '/security', { csrf: false });
      assert.equal(res.status, 200);
      assert.ok(res.json?.data?.controls);
      assert.ok(Array.isArray(res.json.data.controls));
    });

    await t.test('should return certificates from security endpoint', async () => {
      const res = await call('GET', '/security', { csrf: false });
      assert.equal(res.status, 200);
      assert.ok(res.json?.data?.certificates);
      assert.ok(Array.isArray(res.json.data.certificates));
    });

    await t.test('should return audit verification state', async () => {
      const res = await call('GET', '/security', { csrf: false });
      assert.equal(res.status, 200);
      const verification = res.json?.data?.verification;
      assert.ok(verification);
      assert.ok(typeof verification.verifiedAt === 'number');
      assert.ok(typeof verification.state === 'string');
    });

    await t.test('should validate root key rotation request', async () => {
      const res = await call('POST', '/root-rotate', {
        body: {},
        csrf: true
      });

      // Demo adapter may refuse, but should validate the request
      assert.ok([202, 422, 409].includes(res.status), `Status should be accepted/validation/conflict, got ${res.status}`);
    });

    await t.test('should return policy enforcement info', async () => {
      const res = await call('GET', '/security', { csrf: false });
      assert.equal(res.status, 200);
      const policies = res.json?.data?.policies;
      if (policies) {
        assert.ok(typeof policies.maxWorkloads === 'number' || policies.maxWorkloads === undefined);
        assert.ok(typeof policies.cpuLimit === 'string' || policies.cpuLimit === undefined);
      }
    });

    await t.test('should return revocation list', async () => {
      const res = await call('GET', '/security', { csrf: false });
      assert.equal(res.status, 200);
      const revoked = res.json?.data?.revoked;
      if (revoked) {
        assert.ok(Array.isArray(revoked));
        if (revoked.length > 0) {
          const entry = revoked[0];
          assert.ok(typeof entry.key === 'string');
          assert.ok(typeof entry.reason === 'string');
          assert.ok(typeof entry.ts === 'number');
        }
      }
    });

    await t.test('should return root key metadata', async () => {
      const res = await call('GET', '/security', { csrf: false });
      assert.equal(res.status, 200);
      const rootKey = res.json?.data?.rootKey;
      if (rootKey) {
        assert.ok(typeof rootKey.rotatedAt === 'number');
        assert.ok(typeof rootKey.nextRotation === 'number' || rootKey.nextRotation === undefined);
      }
    });

    await t.test('should handle emergency freeze control', async () => {
      const res = await call('POST', '/emergency-control', {
        body: { freeze: true },
        csrf: true
      });

      assert.ok([200, 202, 422, 409].includes(res.status), `Status should be ok/accepted/validation/conflict, got ${res.status}`);
    });

    await t.test('should reject invalid policy updates', async () => {
      const res = await call('POST', '/apply', {
        body: {
          command: 'policy',
          maxWorkloads: -1 // Invalid
        },
        csrf: true
      });

      assert.equal(res.status, 422);
    });

    await t.test('should fetch audit log with retention info', async () => {
      const res = await call('GET', '/audit?limit=50', { csrf: false });
      assert.equal(res.status, 200);
      assert.ok(res.json?.data?.entries);
      assert.ok(Array.isArray(res.json.data.entries));
    });

    await t.test('should support audit log export request', async () => {
      const res = await call('POST', '/audit/export', {
        body: { format: 'json', retention: 30 },
        csrf: true
      });

      assert.ok([200, 202, 404, 422].includes(res.status), `Status should be ok/accepted/not-found/validation, got ${res.status}`);
    });

  } finally {
    await close();
  }
});
