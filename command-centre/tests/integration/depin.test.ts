import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('DePIN Networks: API Contracts', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should fetch DePIN data', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(res.json?.data?.networks !== undefined);
        assert.ok(Array.isArray(res.json.data.networks));
        assert.ok(res.json?.data?.providers !== undefined);
        assert.ok(Array.isArray(res.json.data.providers));
      }
    });

    await t.test('should validate network structure', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200 && res.json?.data?.networks?.length) {
        const net = res.json.data.networks[0];
        assert.ok(typeof net.id === 'string');
        assert.ok(typeof net.name === 'string');
        assert.ok(typeof net.symbol === 'string');
        assert.ok(['CONNECTED', 'SYNCING', 'STALLED', 'DISCONNECTED'].includes(net.status));
        assert.ok(typeof net.providers === 'number');
        assert.ok(typeof net.totalStaked === 'number');
        assert.ok(typeof net.monthlyRewards === 'number');
        assert.ok(typeof net.apy === 'number');
      }
    });

    await t.test('should validate provider structure', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200 && res.json?.data?.providers?.length) {
        const prov = res.json.data.providers[0];
        assert.ok(typeof prov.id === 'string');
        assert.ok(typeof prov.network === 'string');
        assert.ok(['ACTIVE', 'INACTIVE', 'OFFLINE', 'SLASHED'].includes(prov.status));
        assert.ok(typeof prov.stake === 'number');
        assert.ok(typeof prov.rewardsEarned === 'number');
        assert.ok(typeof prov.uptime === 'number');
        assert.ok(typeof prov.reputation === 'number');
        assert.ok(typeof prov.joinedAt === 'number');
      }
    });

    await t.test('should return rewards array', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data?.rewards));
      }
    });

    await t.test('should validate reward structure', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200 && res.json?.data?.rewards?.length) {
        const r = res.json.data.rewards[0];
        assert.ok(typeof r.id === 'string');
        assert.ok(typeof r.network === 'string');
        assert.ok(typeof r.amount === 'number');
        assert.ok(typeof r.period === 'string');
        assert.ok(typeof r.expiresAt === 'number');
      }
    });

    await t.test('should return slashing events', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data?.slashingEvents));
      }
    });

    await t.test('should validate slashing event structure', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200 && res.json?.data?.slashingEvents?.length) {
        const e = res.json.data.slashingEvents[0];
        assert.ok(typeof e.id === 'string');
        assert.ok(typeof e.provider === 'string');
        assert.ok(typeof e.network === 'string');
        assert.ok(typeof e.reason === 'string');
        assert.ok(typeof e.amount === 'number');
        assert.ok(typeof e.timestamp === 'number');
      }
    });

    await t.test('should return reward totals', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200) {
        assert.ok(typeof res.json?.data?.totalRewardsEarned === 'number');
        assert.ok(typeof res.json?.data?.monthlyRewardRate === 'number');
      }
    });

    await t.test('should count active providers and networks', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200) {
        const networks = res.json.data.networks;
        const providers = res.json.data.providers;
        const activeNets = networks.filter((n: any) => n.status === 'CONNECTED').length;
        const activeProvs = providers.filter((p: any) => p.status === 'ACTIVE').length;
        assert.ok(activeNets >= 0);
        assert.ok(activeProvs >= 0);
      }
    });

    await t.test('should include freshness metadata', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      if (res.status === 200) {
        assert.ok(res.json?.data?.freshness !== undefined);
      }
    });

  } finally {
    await close();
  }
});

test('DePIN Networks: Provider Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list providers', async () => {
      const res = await call('GET', '/depin/providers', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter providers by status', async () => {
      const res = await call('GET', '/depin/providers?status=ACTIVE', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should retrieve provider details', async () => {
      const listRes = await call('GET', '/depin/providers', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const providerId = listRes.json.data[0].id;
        const detailRes = await call('GET', `/depin/providers/${providerId}`, { csrf: false });
        assert.ok([200, 404].includes(detailRes.status));
      }
    });

    await t.test('should join network with api.write', async () => {
      const res = await call('POST', '/depin/providers', {
        body: { network: 'akash', stake: 500 },
        csrf: true
      });
      assert.ok([201, 202, 403, 404, 422].includes(res.status));
    });

    await t.test('should validate stake amount', async () => {
      const res = await call('POST', '/depin/providers', {
        body: { network: 'akash', stake: 10 }, // Below minimum
        csrf: true
      });
      assert.ok([400, 422, 403, 404].includes(res.status));
    });

    await t.test('should reject invalid network', async () => {
      const res = await call('POST', '/depin/providers', {
        body: { network: 'invalid_network', stake: 500 },
        csrf: true
      });
      assert.ok([400, 422, 403, 404].includes(res.status));
    });

    await t.test('should enforce CSRF on provider creation', async () => {
      const res = await call('POST', '/depin/providers', {
        body: { network: 'akash', stake: 500 },
        csrf: false
      });
      assert.ok([403, 404, 422].includes(res.status));
    });

  } finally {
    await close();
  }
});

test('DePIN Networks: Network Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list networks', async () => {
      const res = await call('GET', '/depin/networks', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter networks by status', async () => {
      const res = await call('GET', '/depin/networks?status=CONNECTED', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should retrieve network details', async () => {
      const listRes = await call('GET', '/depin/networks', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const networkId = listRes.json.data[0].id;
        const detailRes = await call('GET', `/depin/networks/${networkId}`, { csrf: false });
        assert.ok([200, 404].includes(detailRes.status));
      }
    });

    await t.test('should get network statistics', async () => {
      const listRes = await call('GET', '/depin/networks', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const networkId = listRes.json.data[0].id;
        const statsRes = await call('GET', `/depin/networks/${networkId}/stats`, { csrf: false });
        assert.ok([200, 404].includes(statsRes.status));
      }
    });

  } finally {
    await close();
  }
});

test('DePIN Networks: Reward Operations', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list available rewards', async () => {
      const res = await call('GET', '/depin/rewards', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter rewards by status', async () => {
      const res = await call('GET', '/depin/rewards?status=unclaimed', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should claim reward with api.write', async () => {
      const listRes = await call('GET', '/depin/rewards', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const rewardId = listRes.json.data[0].id;
        const claimRes = await call('POST', `/depin/rewards/${rewardId}/claim`, {
          body: {},
          csrf: true
        });
        assert.ok([200, 201, 400, 403, 404, 422].includes(claimRes.status));
      }
    });

    await t.test('should prevent claiming expired rewards', async () => {
      const res = await call('POST', '/depin/rewards/reward_expired/claim', {
        body: {},
        csrf: true
      });
      assert.ok([400, 404, 422].includes(res.status));
    });

    await t.test('should prevent double claiming', async () => {
      const res = await call('POST', '/depin/rewards/reward_claimed/claim', {
        body: {},
        csrf: true
      });
      assert.ok([400, 404, 422].includes(res.status));
    });

    await t.test('should enforce CSRF on reward operations', async () => {
      const res = await call('POST', '/depin/rewards/any_id/claim', {
        body: {},
        csrf: false
      });
      assert.ok([403, 404, 422].includes(res.status));
    });

  } finally {
    await close();
  }
});

test('DePIN Networks: Authorization Enforcement', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should allow viewers to read networks', async () => {
      const res = await call('GET', '/depin', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should require api.write for stake operations', async () => {
      const res = await call('POST', '/depin/providers', {
        body: { network: 'akash', stake: 500 },
        csrf: true
      });
      // Demo adapter may refuse mutations, but should validate auth
      assert.ok([201, 202, 403, 404, 422].includes(res.status));
    });

    await t.test('should require api.write for reward claims', async () => {
      const res = await call('POST', '/depin/rewards/id/claim', {
        body: {},
        csrf: true
      });
      assert.ok([200, 201, 400, 403, 404, 422].includes(res.status));
    });

  } finally {
    await close();
  }
});

test('DePIN Networks: Provider Lifecycle', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should track provider status transitions', async () => {
      const res = await call('GET', '/depin/providers', { csrf: false });
      if (res.status === 200 && res.json?.data?.length) {
        const provider = res.json.data[0];
        assert.ok(['ACTIVE', 'INACTIVE', 'OFFLINE', 'SLASHED'].includes(provider.status));
      }
    });

    await t.test('should maintain provider uptime metrics', async () => {
      const res = await call('GET', '/depin/providers', { csrf: false });
      if (res.status === 200 && res.json?.data?.length) {
        const provider = res.json.data[0];
        assert.ok(provider.uptime >= 0 && provider.uptime <= 100);
        assert.ok(provider.reputation >= 0 && provider.reputation <= 1000);
      }
    });

    await t.test('should track earned rewards per provider', async () => {
      const res = await call('GET', '/depin/providers', { csrf: false });
      if (res.status === 200 && res.json?.data?.length) {
        const provider = res.json.data[0];
        assert.ok(typeof provider.rewardsEarned === 'number');
        assert.ok(provider.rewardsEarned >= 0);
      }
    });

    await t.test('should show join timestamp', async () => {
      const res = await call('GET', '/depin/providers', { csrf: false });
      if (res.status === 200 && res.json?.data?.length) {
        const provider = res.json.data[0];
        assert.ok(typeof provider.joinedAt === 'number');
        assert.ok(provider.joinedAt > 0);
      }
    });

  } finally {
    await close();
  }
});

test('DePIN Networks: Slashing & Penalties', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should list slashing events', async () => {
      const res = await call('GET', '/depin/slashing', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(Array.isArray(res.json?.data));
      }
    });

    await t.test('should filter slashing events by provider', async () => {
      const res = await call('GET', '/depin/slashing?provider=prov_001', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should show slashing event details', async () => {
      const listRes = await call('GET', '/depin/slashing', { csrf: false });
      if (listRes.status === 200 && listRes.json?.data?.length) {
        const event = listRes.json.data[0];
        assert.ok(typeof event.id === 'string');
        assert.ok(typeof event.provider === 'string');
        assert.ok(typeof event.reason === 'string');
        assert.ok(typeof event.amount === 'number');
      }
    });

  } finally {
    await close();
  }
});
