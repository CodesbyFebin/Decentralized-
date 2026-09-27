import { test } from 'node:test';
import * as assert from 'node:assert';

test('Storage: Volume Name Validation', async (t) => {
  const validateName = (name: string): boolean => /^[a-z0-9][a-z0-9-]{0,62}$/.test(name);

  await t.test('should accept valid volume names', () => {
    const names = ['data-v1', 'vol', 'my-storage-123', 'a'];
    names.forEach(n => assert.ok(validateName(n), `${n} should be valid`));
  });

  await t.test('should reject invalid volume names', () => {
    const names = ['', 'Data-Vol', 'vol_1', 'vol.data', 'a'.repeat(64)];
    names.forEach(n => assert.ok(!validateName(n), `${n} should be invalid`));
  });

  await t.test('should start with lowercase alphanumeric', () => {
    assert.ok(!validateName('-data'));
    assert.ok(!validateName('_vol'));
    assert.ok(validateName('a-vol'));
  });
});

test('Storage: Volume Size Validation', async (t) => {
  const parseSize = (val: string): number => {
    const match = val.match(/^(\d+)([KMG]i)?$/);
    if (!match) return -1;
    const num = parseInt(match[1], 10);
    const unit = match[2] || '';
    const multipliers: Record<string, number> = { 'Ki': 1024, 'Mi': 1024 * 1024, 'Gi': 1024 * 1024 * 1024 };
    return num * (multipliers[unit] || 1);
  };

  await t.test('should parse common size formats', () => {
    assert.equal(parseSize('100'), 100);
    assert.equal(parseSize('1Ki'), 1024);
    assert.equal(parseSize('512Mi'), 512 * 1024 * 1024);
    assert.equal(parseSize('10Gi'), 10 * 1024 * 1024 * 1024);
  });

  await t.test('should reject invalid formats', () => {
    assert.equal(parseSize('invalid'), -1);
    assert.equal(parseSize('10mb'), -1);
    assert.equal(parseSize('10.5Gi'), -1);
  });

  await t.test('should validate minimum and maximum', () => {
    const minBytes = 1024 * 1024; // 1Mi
    const maxBytes = 1024 * 1024 * 1024 * 1024; // 1Ti
    const sizes = [
      { val: '1Mi', bytes: 1024 * 1024, valid: true },
      { val: '100Gi', bytes: 100 * 1024 * 1024 * 1024, valid: true },
      { val: '512Ki', bytes: 512 * 1024, valid: false }
    ];
    sizes.forEach(s => {
      const parsed = parseSize(s.val);
      if (s.valid) {
        assert.ok(parsed >= minBytes && parsed <= maxBytes, `${s.val} should be in valid range`);
      }
    });
  });
});

test('Storage: Durability Level Validation', async (t) => {
  const validateDurability = (level: number): boolean => Number.isInteger(level) && level >= 2 && level <= 4;

  await t.test('should accept valid durability levels', () => {
    for (const level of [2, 3, 4]) {
      assert.ok(validateDurability(level), `${level} should be valid`);
    }
  });

  await t.test('should reject invalid levels', () => {
    for (const level of [1, 0, 5, -1, 2.5]) {
      assert.ok(!validateDurability(level), `${level} should be invalid`);
    }
  });
});

test('Storage: Snapshot Management', async (t) => {
  await t.test('should validate snapshot creation request', () => {
    const req = {
      volumeId: 'data-v1',
      retention: 30,
      consistency: 'quorum'
    };
    assert.ok(req.volumeId);
    assert.ok(typeof req.retention === 'number');
    assert.equal(req.consistency, 'quorum');
  });

  await t.test('should validate snapshot restore request', () => {
    const req = {
      volumeId: 'data-v1',
      snapshotHash: 'abc123def456',
      targetTime: Date.now()
    };
    assert.ok(req.volumeId);
    assert.ok(req.snapshotHash.length > 0);
    assert.ok(req.targetTime > 0);
  });
});

test('Storage: Replica Status', async (t) => {
  await t.test('should determine health from verified replicas', () => {
    const getHealth = (verified: number, replicas: number): string => {
      if (verified === replicas) return 'HEALTHY';
      if (verified >= Math.ceil(replicas / 2)) return 'DEGRADED';
      return 'CRITICAL';
    };

    assert.equal(getHealth(2, 2), 'HEALTHY');
    assert.equal(getHealth(1, 2), 'DEGRADED');
    assert.equal(getHealth(0, 2), 'CRITICAL');
    assert.equal(getHealth(2, 3), 'DEGRADED');
    assert.equal(getHealth(3, 3), 'HEALTHY');
  });

  await t.test('should identify members needing repair', () => {
    const needsRepair = (memberStates: Record<string, boolean>): string[] => {
      return Object.entries(memberStates)
        .filter(([_, verified]) => !verified)
        .map(([member]) => member);
    };

    const members = { 'host-1': true, 'host-2': false, 'host-3': true };
    assert.deepEqual(needsRepair(members), ['host-2']);
  });
});

test('Storage: Quota Management', async (t) => {
  await t.test('should validate quota limits', () => {
    const validateQuota = (used: number, quota: number): boolean => used >= 0 && quota > used;
    assert.ok(validateQuota(500, 1000));
    assert.ok(!validateQuota(1000, 1000));
    assert.ok(!validateQuota(-1, 1000));
  });

  await t.test('should calculate usage percentage', () => {
    const usagePercent = (used: number, quota: number): number => (used / quota) * 100;
    assert.equal(usagePercent(50, 100), 50);
    const pct = usagePercent(1, 1024);
    assert.ok(Math.abs(pct - 0.0977) < 0.001, `Expected ~0.0977, got ${pct}`);
  });

  await t.test('should warn at thresholds', () => {
    const getTone = (percent: number): string => {
      if (percent >= 90) return 'critical';
      if (percent >= 75) return 'warning';
      return 'healthy';
    };
    assert.equal(getTone(95), 'critical');
    assert.equal(getTone(80), 'warning');
    assert.equal(getTone(50), 'healthy');
  });
});
