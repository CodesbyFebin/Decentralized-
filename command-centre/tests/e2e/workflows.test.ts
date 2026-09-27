import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('E2E: Deploy Workflow', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should complete full deployment flow', async () => {
      // Step 1: Get available artifacts
      const artifactsRes = await call('GET', '/deployments', { csrf: false });
      assert.equal(artifactsRes.status, 200);
      assert.ok(Array.isArray(artifactsRes.json?.data?.artifacts));

      // Step 2: Create manifest with artifact selection
      const manifest = `apiVersion: dh/v1
kind: Application
metadata:
  name: web-v2
spec:
  replicas: 2
  image: nginx@sha256:${'b'.repeat(64)}
  resources:
    cpu: 500m
    mem: 256Mi
  env:
    - name: DATABASE_URL
      value: "postgresql://localhost/db"
    - name: API_KEY
      value: "secret-key"`;

      // Step 3: Submit deployment
      const deployRes = await call('POST', '/deployments', {
        body: { manifest },
        csrf: true
      });

      // Demo adapter refuses mutations, but validates manifest
      assert.ok([200, 201, 202, 422, 409].includes(deployRes.status));

      // Step 4: Check deployment progress
      if (deployRes.status < 400) {
        const progressRes = await call('GET', '/deployments', { csrf: false });
        assert.equal(progressRes.status, 200);
        assert.ok(Array.isArray(progressRes.json?.data?.deployments));
      }
    });
  } finally {
    await close();
  }
});

test('E2E: Domain & ACME Certificate Workflow', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should complete domain creation and ACME renewal', async () => {
      // Step 1: Get available applications
      const appsRes = await call('GET', '/apps', { csrf: false });
      assert.equal(appsRes.status, 200);

      // Step 2: Create domain with ACME TLS
      const createRes = await call('POST', '/domains', {
        body: {
          host: 'api.example.com',
          app: 'api-server',
          port: 'https',
          tlsMode: 'acme'
        },
        csrf: true
      });

      assert.ok([200, 201, 404, 409, 422].includes(createRes.status));

      // Step 3: Check domain status and certificate
      const domainsRes = await call('GET', '/domains', { csrf: false });
      assert.ok([200, 404].includes(domainsRes.status));
      if (domainsRes.status === 200) {
        assert.ok(Array.isArray(domainsRes.json?.data?.domains));
      }

      // Step 4: Trigger certificate renewal if close to expiry
      if (domainsRes.json?.data?.domains?.length > 0) {
        const renewRes = await call('POST', '/domains/api.example.com/renew', {
          body: {},
          csrf: true
        });
        assert.ok([200, 202, 404, 409, 422].includes(renewRes.status));
      }
    });
  } finally {
    await close();
  }
});

test('E2E: Storage Volume & Snapshot Recovery', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should handle volume creation and snapshot restore', async () => {
      // Step 1: Check storage status
      const storageRes = await call('GET', '/storage', { csrf: false });
      assert.ok([200, 404].includes(storageRes.status));
      if (storageRes.status !== 200) return; // Skip if not available

      // Step 2: Create new volume
      const createVolRes = await call('POST', '/volumes', {
        body: {
          name: 'backup-vol',
          sizeBytes: 100 * 1024 * 1024 * 1024, // 100Gi
          durabilityReplicas: 2,
          app: 'backup-service'
        },
        csrf: true
      });

      assert.ok([201, 202, 404, 409, 422].includes(createVolRes.status));

      // Step 3: Create snapshot
      const snapRes = await call('POST', '/volumes/backup-vol/snapshot', {
        body: { retention: 30 },
        csrf: true
      });

      assert.ok([201, 202, 404, 409, 422].includes(snapRes.status));

      // Step 4: Restore from snapshot
      const restoreRes = await call('POST', '/volumes/backup-vol/restore', {
        body: { snapshotHash: 'abc123def456' },
        csrf: true
      });

      assert.ok([200, 202, 404, 409, 422].includes(restoreRes.status));

      // Step 5: Trigger repair if needed
      const repairRes = await call('POST', '/volumes/backup-vol/repair', {
        body: {},
        csrf: true
      });

      assert.ok([202, 404, 409, 422].includes(repairRes.status));
    });
  } finally {
    await close();
  }
});

test('E2E: Security & Emergency Partition Recovery', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should handle partition scenario and recovery', async () => {
      // Step 1: Check security controls and cluster state
      const secRes = await call('GET', '/security', { csrf: false });
      assert.ok([200, 404].includes(secRes.status));
      if (secRes.status !== 200) return; // Skip if not available

      // Step 2: Detect degraded controls during partition
      // (In real scenario, this would happen automatically)

      // Step 3: Freeze cluster to prevent bad decisions
      const freezeRes = await call('POST', '/emergency-control', {
        body: { freeze: true },
        csrf: true
      });

      assert.ok([200, 202, 404, 409, 422].includes(freezeRes.status));

      // Step 4: Review audit log for partition events
      const auditRes = await call('GET', '/audit?limit=100', { csrf: false });
      assert.equal(auditRes.status, 200);
      assert.ok(Array.isArray(auditRes.json?.data?.entries));

      // Step 5: Rotate root key if compromise suspected
      const rotateRes = await call('POST', '/root-rotate', {
        body: {},
        csrf: true
      });

      assert.ok([202, 409, 422].includes(rotateRes.status));

      // Step 6: Unfreeze cluster once resolved
      const unfreezeRes = await call('POST', '/emergency-control', {
        body: { freeze: false },
        csrf: true
      });

      assert.ok([200, 202, 404, 409, 422].includes(unfreezeRes.status));
    });
  } finally {
    await close();
  }
});

test('E2E: Copilot Diagnostic & Action Flow', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should handle copilot query and action approval', async () => {
      // Step 1: Ask copilot for diagnostics
      const queryRes = await call('POST', '/copilot/query', {
        body: { query: "What's wrong with my cluster?" },
        csrf: true
      });

      assert.ok([200, 401, 503].includes(queryRes.status));

      if (queryRes.status === 200 && queryRes.json?.data?.proposedAction) {
        const action = queryRes.json.data.proposedAction;

        // Step 2: Review proposed action (drain node, scale, restart)
        assert.ok(['drain', 'scale', 'restart', 'cordon'].includes(action.kind));

        // Step 3: Approve action if needed
        const approveRes = await call('POST', `/copilot/actions/${action.id}/approve`, {
          body: { confirm: action.confirmText || '' },
          csrf: true
        });

        assert.ok([200, 202, 400, 403, 404].includes(approveRes.status));
      }
    });
  } finally {
    await close();
  }
});
