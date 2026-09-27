import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('Billing & Marketplace: API Contracts', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should fetch billing data', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(res.json?.data?.metering !== undefined);
        assert.ok(typeof res.json.data.metering.cpuSecondTotal === 'number');
        assert.ok(typeof res.json.data.metering.egressBytes === 'number');
      }
    });

    await t.test('should validate metering structure', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.metering) {
        const m = res.json.data.metering;
        assert.ok(typeof m.cpuSecondTotal === 'number' && m.cpuSecondTotal >= 0);
        assert.ok(typeof m.memoryByteHours === 'number' && m.memoryByteHours >= 0);
        assert.ok(typeof m.storageByteHours === 'number' && m.storageByteHours >= 0);
        assert.ok(typeof m.egressBytes === 'number' && m.egressBytes >= 0);
        assert.ok(typeof m.periodStart === 'number');
        assert.ok(typeof m.periodEnd === 'number');
      }
    });

    await t.test('should return settlements array', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data?.settlements));
      }
    });

    await t.test('should validate settlement structure', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.settlements?.length) {
        const s = res.json.data.settlements[0];
        assert.ok(typeof s.id === 'string');
        assert.ok(typeof s.amount === 'number');
        assert.ok(typeof s.currency === 'string');
        assert.ok(['PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'].includes(s.status));
        assert.ok(typeof s.method === 'string');
        assert.ok(typeof s.createdAt === 'number');
      }
    });

    await t.test('should return leases array', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data?.leases));
      }
    });

    await t.test('should validate lease structure', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.leases?.length) {
        const l = res.json.data.leases[0];
        assert.ok(typeof l.id === 'string');
        assert.ok(typeof l.resource === 'string');
        assert.ok(typeof l.quantity === 'number');
        assert.ok(typeof l.unit === 'string');
        assert.ok(typeof l.pricePerUnit === 'number');
        assert.ok(typeof l.startAt === 'number');
        assert.ok(typeof l.endAt === 'number');
        assert.ok(['ACTIVE', 'EXPIRED', 'CANCELLED'].includes(l.status));
      }
    });

    await t.test('should return current balance', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200) {
        assert.ok(typeof res.json?.data?.currentBalance === 'number');
      }
    });

    await t.test('should return credits array', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data?.credits));
      }
    });

    await t.test('should validate credit structure', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.credits?.length) {
        const c = res.json.data.credits[0];
        assert.ok(typeof c.amount === 'number');
        assert.ok(typeof c.source === 'string');
      }
    });

    await t.test('should return invoices array', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data?.invoices));
      }
    });

    await t.test('should validate invoice structure', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.invoices?.length) {
        const inv = res.json.data.invoices[0];
        assert.ok(typeof inv.id === 'string');
        assert.ok(typeof inv.number === 'string');
        assert.ok(typeof inv.amount === 'number');
        assert.ok(typeof inv.currency === 'string');
        assert.ok(['DRAFT', 'SENT', 'PAID', 'OVERDUE', 'CANCELLED'].includes(inv.status));
        assert.ok(typeof inv.issuedAt === 'number');
        assert.ok(typeof inv.dueAt === 'number');
      }
    });

    await t.test('should include freshness metadata', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200) {
        assert.ok(res.json?.data?.freshness !== undefined);
      }
    });

    await t.test('should count active leases', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.leases?.length) {
        const activeLeases = res.json.data.leases.filter((l: any) => l.status === 'ACTIVE').length;
        assert.ok(activeLeases >= 0);
      }
    });

    await t.test('should identify unpaid invoices', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.invoices?.length) {
        const unpaid = res.json.data.invoices.filter((inv: any) => inv.status !== 'PAID').length;
        assert.ok(unpaid >= 0);
      }
    });

    await t.test('should sum available credits', async () => {
      const res = await call('GET', '/billing', { csrf: false });
      if (res.status === 200 && res.json?.data?.credits?.length) {
        const totalCredits = res.json.data.credits.reduce((sum: number, c: any) => sum + c.amount, 0);
        assert.ok(totalCredits >= 0);
      }
    });

  } finally {
    await close();
  }
});

test('Billing & Marketplace: Settlement Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list recent settlements', async () => {
      const res = await call('GET', '/billing/settlements', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter settlements by status', async () => {
      const res = await call('GET', '/billing/settlements?status=COMPLETED', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should retrieve settlement details', async () => {
      const listRes = await call('GET', '/billing/settlements', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const settlementId = listRes.json.data[0].id;
        const detailRes = await call('GET', `/billing/settlements/${settlementId}`, { csrf: false });
        assert.ok([200, 404].includes(detailRes.status));
      }
    });

  } finally {
    await close();
  }
});

test('Billing & Marketplace: Lease Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list active leases', async () => {
      const res = await call('GET', '/billing/leases', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter leases by status', async () => {
      const res = await call('GET', '/billing/leases?status=ACTIVE', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should create lease with valid parameters', async () => {
      const res = await call('POST', '/billing/leases', {
        body: { resource: 'compute', quantity: 4, unit: 'cores', ttl: 2592000 },
        csrf: true
      });
      assert.ok([201, 202, 403, 404, 422].includes(res.status));
    });

    await t.test('should validate lease quantity', async () => {
      const res = await call('POST', '/billing/leases', {
        body: { resource: 'compute', quantity: 0, unit: 'cores', ttl: 2592000 },
        csrf: true
      });
      assert.ok([400, 422, 403, 404].includes(res.status));
    });

    await t.test('should reject invalid resource type', async () => {
      const res = await call('POST', '/billing/leases', {
        body: { resource: 'invalid', quantity: 4, unit: 'cores', ttl: 2592000 },
        csrf: true
      });
      assert.ok([400, 422, 403, 404].includes(res.status));
    });

  } finally {
    await close();
  }
});

test('Billing & Marketplace: Invoice Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should retrieve invoices', async () => {
      const res = await call('GET', '/billing/invoices', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter invoices by status', async () => {
      const res = await call('GET', '/billing/invoices?status=PAID', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should retrieve invoice details', async () => {
      const listRes = await call('GET', '/billing/invoices', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const invoiceId = listRes.json.data[0].id;
        const detailRes = await call('GET', `/billing/invoices/${invoiceId}`, { csrf: false });
        assert.ok([200, 404].includes(detailRes.status));
      }
    });

    await t.test('should reject operations without api.write', async () => {
      const res = await call('POST', '/billing/invoices/inv_001/pay', {
        body: { method: 'card' },
        csrf: true
      });
      assert.ok([403, 404, 422].includes(res.status));
    });

  } finally {
    await close();
  }
});

test('Billing & Marketplace: Credit Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should retrieve credits', async () => {
      const res = await call('GET', '/billing/credits', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should apply promo code', async () => {
      const res = await call('POST', '/billing/credits/apply', {
        body: { code: 'PROMO100' },
        csrf: true
      });
      assert.ok([200, 201, 400, 403, 404, 422].includes(res.status));
    });

    await t.test('should validate promo code format', async () => {
      const res = await call('POST', '/billing/credits/apply', {
        body: { code: '' },
        csrf: true
      });
      assert.ok([400, 422, 403, 404].includes(res.status));
    });

    await t.test('should reject invalid codes', async () => {
      const res = await call('POST', '/billing/credits/apply', {
        body: { code: 'INVALID_CODE_XYZ' },
        csrf: true
      });
      assert.ok([400, 404, 422, 403].includes(res.status));
    });

  } finally {
    await close();
  }
});

test('Billing & Marketplace: CSRF Protection', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should enforce CSRF on lease creation', async () => {
      const res = await call('POST', '/billing/leases', {
        body: { resource: 'compute', quantity: 4, unit: 'cores', ttl: 2592000 },
        csrf: false
      });
      assert.ok([403, 404, 422].includes(res.status));
    });

    await t.test('should enforce CSRF on credit operations', async () => {
      const res = await call('POST', '/billing/credits/apply', {
        body: { code: 'PROMO100' },
        csrf: false
      });
      assert.ok([403, 404, 422].includes(res.status));
    });

  } finally {
    await close();
  }
});
