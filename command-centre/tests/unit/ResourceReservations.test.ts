import { test } from 'node:test';
import assert from 'node:assert';

interface Capacity {
  cpuCores: number;
  memoryBytes: number;
  storageBytes: number;
}

interface Reservation {
  id: string;
  nodeId: string;
  name: string;
  status: 'pending' | 'approved' | 'allocated' | 'released';
  requested: Capacity;
  createdAt: number;
  approvedAt?: number;
  allocatedAt?: number;
  releasedAt?: number;
  reason?: string;
}

interface Allocation {
  id: string;
  reservationId: string;
  deploymentId: string;
  allocated: Capacity;
  utilizationCpuCores?: number;
  utilizationMemoryBytes?: number;
  utilizationStorageBytes?: number;
  createdAt: number;
}

interface ResourceReservationsData {
  declaredCapacity: Capacity;
  reservations: Reservation[];
  allocations: Allocation[];
}

function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(1)} ${units[unitIndex]}`;
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'pending':
      return 'amber';
    case 'approved':
      return 'blue';
    case 'allocated':
      return 'emerald';
    case 'released':
      return 'slate';
    default:
      return 'slate';
  }
}

function validateCapacity(capacity: Capacity): boolean {
  if (typeof capacity !== 'object' || !capacity) return false;
  return (
    typeof capacity.cpuCores === 'number' &&
    typeof capacity.memoryBytes === 'number' &&
    typeof capacity.storageBytes === 'number' &&
    capacity.cpuCores >= 0 &&
    capacity.memoryBytes >= 0 &&
    capacity.storageBytes >= 0
  );
}

function validateReservationName(name: string): boolean {
  return typeof name === 'string' && name.trim().length > 0 && name.trim().length <= 256;
}

function validateReservationMinimums(cpuCores: number, memoryBytes: number): boolean {
  return cpuCores >= 0.5 && memoryBytes >= 268435456;
}

function calculateReservationAge(createdAt: number): number {
  return Date.now() - createdAt;
}

function calculateUtilizationPercent(utilized: number, allocated: number): number {
  if (allocated === 0) return 0;
  return Math.min(100, (utilized / allocated) * 100);
}

function validateResourceReservationsData(data: unknown): boolean {
  if (typeof data !== 'object' || !data) return false;
  const d = data as Record<string, unknown>;

  if (!d.declaredCapacity || !validateCapacity(d.declaredCapacity as Capacity)) return false;

  if (!Array.isArray(d.reservations)) return false;
  for (const res of d.reservations) {
    if (typeof res !== 'object' || !res) return false;
    const r = res as Record<string, unknown>;
    if (typeof r.id !== 'string' || typeof r.name !== 'string') return false;
    if (!['pending', 'approved', 'allocated', 'released'].includes(r.status as string)) return false;
    if (!validateCapacity(r.requested as Capacity)) return false;
  }

  if (!Array.isArray(d.allocations)) return false;
  for (const alloc of d.allocations) {
    if (typeof alloc !== 'object' || !alloc) return false;
    const a = alloc as Record<string, unknown>;
    if (typeof a.id !== 'string' || typeof a.deploymentId !== 'string') return false;
    if (!validateCapacity(a.allocated as Capacity)) return false;
  }

  return true;
}

test('Capacity Validation', async (t) => {
  await t.test('validates valid capacity object', () => {
    const capacity: Capacity = {
      cpuCores: 8,
      memoryBytes: 17179869184,
      storageBytes: 1099511627776,
    };
    assert.ok(validateCapacity(capacity));
  });

  await t.test('rejects null capacity', () => {
    assert.equal(validateCapacity(null as unknown), false);
  });

  await t.test('rejects missing cpuCores', () => {
    const capacity = { memoryBytes: 17179869184, storageBytes: 1099511627776 };
    assert.equal(validateCapacity(capacity), false);
  });

  await t.test('rejects negative cpuCores', () => {
    const capacity: Capacity = { cpuCores: -1, memoryBytes: 17179869184, storageBytes: 1099511627776 };
    assert.equal(validateCapacity(capacity), false);
  });

  await t.test('accepts zero cpuCores', () => {
    const capacity: Capacity = { cpuCores: 0, memoryBytes: 0, storageBytes: 0 };
    assert.ok(validateCapacity(capacity));
  });

  await t.test('rejects non-number values', () => {
    const capacity = { cpuCores: 'eight', memoryBytes: 17179869184, storageBytes: 1099511627776 };
    assert.equal(validateCapacity(capacity), false);
  });
});

test('Byte Formatting', async (t) => {
  await t.test('formats bytes to B', () => {
    assert.equal(formatBytes(512), '512.0 B');
  });

  await t.test('formats to KB', () => {
    assert.equal(formatBytes(2048), '2.0 KB');
  });

  await t.test('formats to MB', () => {
    assert.equal(formatBytes(10485760), '10.0 MB');
  });

  await t.test('formats to GB', () => {
    assert.equal(formatBytes(1073741824), '1.0 GB');
  });

  await t.test('formats to TB', () => {
    assert.equal(formatBytes(1099511627776), '1.0 TB');
  });

  await t.test('formats 16 GB memory', () => {
    assert.equal(formatBytes(17179869184), '16.0 GB');
  });

  await t.test('formats fractional sizes', () => {
    assert.equal(formatBytes(1572864), '1.5 MB');
  });

  await t.test('caps at TB', () => {
    const huge = 1099511627776 * 100;
    const formatted = formatBytes(huge);
    assert.ok(formatted.includes('TB'));
  });
});

test('Reservation Name Validation', async (t) => {
  await t.test('validates normal name', () => {
    assert.ok(validateReservationName('production-web-tier'));
  });

  await t.test('rejects empty name', () => {
    assert.equal(validateReservationName(''), false);
  });

  await t.test('rejects whitespace-only name', () => {
    assert.equal(validateReservationName('   '), false);
  });

  await t.test('accepts name with numbers', () => {
    assert.ok(validateReservationName('tier1-production'));
  });

  await t.test('accepts name with special characters', () => {
    assert.ok(validateReservationName('prod_db-cluster-001'));
  });

  await t.test('rejects non-string', () => {
    assert.equal(validateReservationName(null as unknown), false);
  });

  await t.test('accepts max length name', () => {
    const maxName = 'a'.repeat(256);
    assert.ok(validateReservationName(maxName));
  });

  await t.test('rejects over-max length name', () => {
    const overMaxName = 'a'.repeat(257);
    assert.equal(validateReservationName(overMaxName), false);
  });
});

test('Capacity Minimums', async (t) => {
  await t.test('accepts minimum cpu cores (0.5)', () => {
    assert.ok(validateReservationMinimums(0.5, 268435456));
  });

  await t.test('rejects below minimum cpu cores', () => {
    assert.equal(validateReservationMinimums(0.25, 268435456), false);
  });

  await t.test('accepts minimum memory (256 MB)', () => {
    assert.ok(validateReservationMinimums(1, 268435456));
  });

  await t.test('rejects below minimum memory', () => {
    assert.equal(validateReservationMinimums(1, 134217728), false);
  });

  await t.test('accepts generous capacities', () => {
    assert.ok(validateReservationMinimums(16, 34359738368));
  });

  await t.test('rejects negative cpu', () => {
    assert.equal(validateReservationMinimums(-1, 268435456), false);
  });
});

test('Reservation Status Transitions', async (t) => {
  await t.test('initial status is pending', () => {
    const reservation: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'web-tier',
      status: 'pending',
      requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
    };
    assert.equal(reservation.status, 'pending');
  });

  await t.test('can transition pending to approved', () => {
    const reservation: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'web-tier',
      status: 'approved',
      requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
      approvedAt: Date.now(),
    };
    assert.equal(reservation.status, 'approved');
    assert.ok(reservation.approvedAt);
  });

  await t.test('can transition approved to allocated', () => {
    const reservation: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'web-tier',
      status: 'allocated',
      requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
      approvedAt: Date.now(),
      allocatedAt: Date.now(),
    };
    assert.equal(reservation.status, 'allocated');
    assert.ok(reservation.allocatedAt);
  });

  await t.test('can transition to released', () => {
    const reservation: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'web-tier',
      status: 'released',
      requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
      releasedAt: Date.now(),
    };
    assert.equal(reservation.status, 'released');
    assert.ok(reservation.releasedAt);
  });

  await t.test('all status values are valid', () => {
    const validStatuses: Array<'pending' | 'approved' | 'allocated' | 'released'> = ['pending', 'approved', 'allocated', 'released'];
    for (const status of validStatuses) {
      const res: Reservation = {
        id: 'res-001',
        nodeId: 'node-001',
        name: 'test',
        status,
        requested: { cpuCores: 1, memoryBytes: 268435456, storageBytes: 1073741824 },
        createdAt: Date.now(),
      };
      assert.equal(res.status, status);
    }
  });
});

test('Reservation Age Calculation', async (t) => {
  await t.test('calculates current age', () => {
    const now = Date.now();
    const oneHourAgo = now - 3600000;
    const age = calculateReservationAge(oneHourAgo);
    assert.ok(age >= 3600000 && age < 3601000);
  });

  await t.test('handles very recent reservation', () => {
    const now = Date.now();
    const age = calculateReservationAge(now);
    assert.ok(age >= 0 && age < 100);
  });

  await t.test('handles old reservation', () => {
    const now = Date.now();
    const thirtyDaysAgo = now - 2592000000;
    const age = calculateReservationAge(thirtyDaysAgo);
    assert.ok(age >= 2592000000);
  });
});

test('Status Color Mapping', async (t) => {
  await t.test('pending maps to amber', () => {
    assert.equal(getStatusColor('pending'), 'amber');
  });

  await t.test('approved maps to blue', () => {
    assert.equal(getStatusColor('approved'), 'blue');
  });

  await t.test('allocated maps to emerald', () => {
    assert.equal(getStatusColor('allocated'), 'emerald');
  });

  await t.test('released maps to slate', () => {
    assert.equal(getStatusColor('released'), 'slate');
  });

  await t.test('unknown status maps to slate', () => {
    assert.equal(getStatusColor('unknown'), 'slate');
  });
});

test('Allocation Utilization Calculation', async (t) => {
  await t.test('calculates cpu utilization percent', () => {
    const percent = calculateUtilizationPercent(4, 8);
    assert.equal(percent, 50);
  });

  await t.test('calculates full utilization', () => {
    const percent = calculateUtilizationPercent(8, 8);
    assert.equal(percent, 100);
  });

  await t.test('calculates partial utilization', () => {
    const percent = calculateUtilizationPercent(2, 8);
    assert.equal(percent, 25);
  });

  await t.test('handles zero allocated', () => {
    const percent = calculateUtilizationPercent(0, 0);
    assert.equal(percent, 0);
  });

  await t.test('caps at 100 percent', () => {
    const percent = calculateUtilizationPercent(10, 8);
    assert.equal(percent, 100);
  });

  await t.test('handles fractional values', () => {
    const percent = calculateUtilizationPercent(1.5, 4);
    assert.equal(percent, 37.5);
  });
});

test('Allocation Structure', async (t) => {
  await t.test('allocation includes deployment id', () => {
    const allocation: Allocation = {
      id: 'alloc-001',
      reservationId: 'res-001',
      deploymentId: 'deploy-web-001',
      allocated: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
    };
    assert.equal(allocation.deploymentId, 'deploy-web-001');
  });

  await t.test('allocation tracks utilization', () => {
    const allocation: Allocation = {
      id: 'alloc-001',
      reservationId: 'res-001',
      deploymentId: 'deploy-web-001',
      allocated: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      utilizationCpuCores: 2,
      utilizationMemoryBytes: 4294967296,
      utilizationStorageBytes: 53687091200,
      createdAt: Date.now(),
    };
    assert.equal(allocation.utilizationCpuCores, 2);
    assert.equal(allocation.utilizationMemoryBytes, 4294967296);
  });

  await t.test('allocation utilization is optional', () => {
    const allocation: Allocation = {
      id: 'alloc-001',
      reservationId: 'res-001',
      deploymentId: 'deploy-web-001',
      allocated: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
    };
    assert.equal(allocation.utilizationCpuCores, undefined);
  });
});

test('Resource Reservations Data Validation', async (t) => {
  await t.test('validates complete data structure', () => {
    const data: ResourceReservationsData = {
      declaredCapacity: { cpuCores: 32, memoryBytes: 137438953472, storageBytes: 10995116277760 },
      reservations: [
        {
          id: 'res-001',
          nodeId: 'node-001',
          name: 'web-tier',
          status: 'approved',
          requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
          createdAt: Date.now(),
        },
      ],
      allocations: [
        {
          id: 'alloc-001',
          reservationId: 'res-001',
          deploymentId: 'deploy-001',
          allocated: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
          createdAt: Date.now(),
        },
      ],
    };
    assert.ok(validateResourceReservationsData(data));
  });

  await t.test('validates empty reservations list', () => {
    const data: ResourceReservationsData = {
      declaredCapacity: { cpuCores: 32, memoryBytes: 137438953472, storageBytes: 10995116277760 },
      reservations: [],
      allocations: [],
    };
    assert.ok(validateResourceReservationsData(data));
  });

  await t.test('rejects missing declared capacity', () => {
    const data = { reservations: [], allocations: [] };
    assert.equal(validateResourceReservationsData(data), false);
  });

  await t.test('rejects invalid reservation in list', () => {
    const data = {
      declaredCapacity: { cpuCores: 32, memoryBytes: 137438953472, storageBytes: 10995116277760 },
      reservations: [{ id: 'res-001' }],
      allocations: [],
    };
    assert.equal(validateResourceReservationsData(data), false);
  });
});

test('Reservation with Reason', async (t) => {
  await t.test('includes reason when provided', () => {
    const reservation: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'web-tier',
      status: 'pending',
      requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
      reason: 'Black Friday spike expected',
    };
    assert.equal(reservation.reason, 'Black Friday spike expected');
  });

  await t.test('reason is optional', () => {
    const reservation: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'web-tier',
      status: 'pending',
      requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
      createdAt: Date.now(),
    };
    assert.equal(reservation.reason, undefined);
  });
});

test('Multi-Reservation Scenarios', async (t) => {
  await t.test('handles multiple reservations in sequence', () => {
    const reservations: Reservation[] = [
      {
        id: 'res-001',
        nodeId: 'node-001',
        name: 'web-tier',
        status: 'approved',
        requested: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
        createdAt: Date.now(),
      },
      {
        id: 'res-002',
        nodeId: 'node-001',
        name: 'db-tier',
        status: 'pending',
        requested: { cpuCores: 8, memoryBytes: 34359738368, storageBytes: 1099511627776 },
        createdAt: Date.now(),
      },
    ];
    assert.equal(reservations.length, 2);
    assert.equal(reservations[0].status, 'approved');
    assert.equal(reservations[1].status, 'pending');
  });

  await t.test('tracks multiple allocations', () => {
    const allocations: Allocation[] = [
      {
        id: 'alloc-001',
        reservationId: 'res-001',
        deploymentId: 'deploy-web-001',
        allocated: { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 },
        createdAt: Date.now(),
      },
      {
        id: 'alloc-002',
        reservationId: 'res-002',
        deploymentId: 'deploy-db-001',
        allocated: { cpuCores: 8, memoryBytes: 34359738368, storageBytes: 1099511627776 },
        createdAt: Date.now(),
      },
    ];
    assert.equal(allocations.length, 2);
  });

  await t.test('handles heterogeneous reservations', () => {
    const res1: Reservation = {
      id: 'res-001',
      nodeId: 'node-001',
      name: 'small',
      status: 'pending',
      requested: { cpuCores: 1, memoryBytes: 1073741824, storageBytes: 10737418240 },
      createdAt: Date.now(),
    };
    const res2: Reservation = {
      id: 'res-002',
      nodeId: 'node-001',
      name: 'large',
      status: 'pending',
      requested: { cpuCores: 32, memoryBytes: 137438953472, storageBytes: 10995116277760 },
      createdAt: Date.now(),
    };
    assert.notEqual(res1.requested.cpuCores, res2.requested.cpuCores);
  });
});

test('Capacity Planning Scenarios', async (t) => {
  await t.test('calculates remaining capacity after single reservation', () => {
    const declared: Capacity = { cpuCores: 32, memoryBytes: 137438953472, storageBytes: 10995116277760 };
    const reserved: Capacity = { cpuCores: 8, memoryBytes: 34359738368, storageBytes: 1099511627776 };
    const remaining: Capacity = {
      cpuCores: declared.cpuCores - reserved.cpuCores,
      memoryBytes: declared.memoryBytes - reserved.memoryBytes,
      storageBytes: declared.storageBytes - reserved.storageBytes,
    };
    assert.equal(remaining.cpuCores, 24);
  });

  await t.test('accumulates multiple reservations', () => {
    const declared: Capacity = { cpuCores: 32, memoryBytes: 137438953472, storageBytes: 10995116277760 };
    const res1 = { cpuCores: 4, memoryBytes: 8589934592, storageBytes: 107374182400 };
    const res2 = { cpuCores: 8, memoryBytes: 34359738368, storageBytes: 1099511627776 };
    const totalReserved: Capacity = {
      cpuCores: res1.cpuCores + res2.cpuCores,
      memoryBytes: res1.memoryBytes + res2.memoryBytes,
      storageBytes: res1.storageBytes + res2.storageBytes,
    };
    assert.equal(totalReserved.cpuCores, 12);
  });
});
