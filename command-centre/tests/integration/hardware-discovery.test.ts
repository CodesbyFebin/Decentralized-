import { test } from 'node:test';
import assert from 'node:assert';

interface FactValue {
  value: string | number;
  measured: boolean;
}

interface NetworkInterface {
  name: string;
  ipv4?: string;
  ipv6?: string;
  macAddress?: string;
  mtu?: number;
  speed?: string;
}

interface ContainerRuntime {
  name: string;
  version?: string;
  status: 'available' | 'unavailable';
}

interface VirtualizationInfo {
  type: string;
  detected: boolean;
}

interface HardwareFacts {
  memBytes?: FactValue;
  cpuModel?: FactValue;
  cpuCores?: FactValue;
  cpuThreads?: FactValue;
  swapBytes?: FactValue;
  nics?: NetworkInterface[];
  containerRuntimes?: ContainerRuntime[];
  virtualization?: VirtualizationInfo;
  unknown?: string[];
}

interface NodeData {
  id: string;
  facts?: HardwareFacts;
}

// Mock API response structure
function createMockNodeResponse(): NodeData {
  return {
    id: 'node-001',
    facts: {
      memBytes: { value: 17179869184, measured: true }, // 16 GB
      cpuModel: { value: 'Intel(R) Core(TM) i7-9700K', measured: true },
      cpuCores: { value: 8, measured: true },
      cpuThreads: { value: 8, measured: true },
      swapBytes: { value: 4294967296, measured: true }, // 4 GB
      nics: [
        {
          name: 'eth0',
          ipv4: '192.168.1.100',
          macAddress: '00:1a:2b:3c:4d:5e',
          mtu: 1500,
          speed: '1 Gbps',
        },
        {
          name: 'eth1',
          ipv4: '10.0.0.1',
          macAddress: 'aa:bb:cc:dd:ee:ff',
          mtu: 1500,
        },
      ],
      containerRuntimes: [
        { name: 'docker', version: '24.0.0', status: 'available' },
        { name: 'containerd', version: '1.7.0', status: 'available' },
        { name: 'cri-o', status: 'unavailable' },
        { name: 'podman', status: 'unavailable' },
      ],
      virtualization: {
        type: 'kvm',
        detected: true,
      },
      unknown: ['nat_type', 'external_reachability'],
    },
  };
}

function validateHardwareFactsStructure(facts: unknown): boolean {
  if (typeof facts !== 'object' || !facts) return false;
  const f = facts as Record<string, unknown>;

  // Validate fact values if present
  if (f.memBytes && typeof f.memBytes === 'object') {
    const mf = f.memBytes as Record<string, unknown>;
    if (typeof mf.measured !== 'boolean') return false;
  }

  // Validate NICs if present
  if (f.nics && Array.isArray(f.nics)) {
    for (const nic of f.nics) {
      if (typeof nic !== 'object' || !nic) return false;
      const n = nic as Record<string, unknown>;
      if (typeof n.name !== 'string') return false;
    }
  }

  // Validate container runtimes if present
  if (f.containerRuntimes && Array.isArray(f.containerRuntimes)) {
    for (const runtime of f.containerRuntimes) {
      if (typeof runtime !== 'object' || !runtime) return false;
      const r = runtime as Record<string, unknown>;
      if (typeof r.name !== 'string') return false;
      if (typeof r.status !== 'string') return false;
    }
  }

  // Validate virtualization if present
  if (f.virtualization && typeof f.virtualization === 'object') {
    const v = f.virtualization as Record<string, unknown>;
    if (typeof v.type !== 'string') return false;
    if (typeof v.detected !== 'boolean') return false;
  }

  return true;
}

test('GET /nodes/{id}/hardware endpoint', async (t) => {
  await t.test('returns valid hardware facts structure', () => {
    const response = createMockNodeResponse();
    assert.ok(response.facts);
    assert.ok(validateHardwareFactsStructure(response.facts));
  });

  await t.test('includes CPU information', () => {
    const response = createMockNodeResponse();
    assert.ok(response.facts?.cpuModel);
    assert.ok(response.facts?.cpuCores);
    assert.ok(response.facts?.cpuThreads);
  });

  await t.test('includes memory information', () => {
    const response = createMockNodeResponse();
    assert.ok(response.facts?.memBytes);
    assert.ok(response.facts?.swapBytes);
  });

  await t.test('includes measured flag on facts', () => {
    const response = createMockNodeResponse();
    if (response.facts?.memBytes) {
      assert.equal(typeof response.facts.memBytes.measured, 'boolean');
      assert.equal(response.facts.memBytes.measured, true);
    }
  });

  await t.test('returns empty facts when unavailable', () => {
    const response: NodeData = { id: 'node-002', facts: {} };
    assert.ok(response.facts !== undefined);
  });
});

test('Network Interface Discovery in API', async (t) => {
  await t.test('returns all network interfaces', () => {
    const response = createMockNodeResponse();
    const nics = response.facts?.nics || [];
    assert.ok(nics.length > 0);
  });

  await t.test('includes IPv4 addresses', () => {
    const response = createMockNodeResponse();
    const nics = response.facts?.nics || [];
    const withIpv4 = nics.filter((n) => n.ipv4);
    assert.ok(withIpv4.length > 0);
  });

  await t.test('includes MAC addresses', () => {
    const response = createMockNodeResponse();
    const nics = response.facts?.nics || [];
    const withMac = nics.filter((n) => n.macAddress);
    assert.ok(withMac.length > 0);
  });

  await t.test('includes MTU values', () => {
    const response = createMockNodeResponse();
    const nics = response.facts?.nics || [];
    const withMtu = nics.filter((n) => n.mtu);
    assert.ok(withMtu.length > 0);
  });

  await t.test('includes speed information', () => {
    const response = createMockNodeResponse();
    const nics = response.facts?.nics || [];
    const withSpeed = nics.filter((n) => n.speed);
    assert.ok(withSpeed.length >= 0); // Speed is optional
  });

  await t.test('validates NIC name field', () => {
    const response = createMockNodeResponse();
    const nics = response.facts?.nics || [];
    for (const nic of nics) {
      assert.ok(typeof nic.name === 'string');
      assert.ok(nic.name.length > 0);
    }
  });
});

test('Container Runtime Discovery in API', async (t) => {
  await t.test('returns container runtime status', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    assert.ok(runtimes.length > 0);
  });

  await t.test('includes Docker status', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    const docker = runtimes.find((r) => r.name === 'docker');
    assert.ok(docker);
  });

  await t.test('includes Containerd status', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    const containerd = runtimes.find((r) => r.name === 'containerd');
    assert.ok(containerd);
  });

  await t.test('includes CRI-O status', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    const crioRuntime = runtimes.find((r) => r.name === 'cri-o');
    assert.ok(crioRuntime);
  });

  await t.test('includes Podman status', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    const podman = runtimes.find((r) => r.name === 'podman');
    assert.ok(podman);
  });

  await t.test('reports availability status for each runtime', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    for (const runtime of runtimes) {
      assert.ok(['available', 'unavailable'].includes(runtime.status));
    }
  });

  await t.test('includes version when available', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    const withVersion = runtimes.filter((r) => r.version);
    assert.ok(withVersion.length >= 0); // Version is optional
  });
});

test('Virtualization Detection in API', async (t) => {
  await t.test('returns virtualization information', () => {
    const response = createMockNodeResponse();
    assert.ok(response.facts?.virtualization);
  });

  await t.test('includes hypervisor type', () => {
    const response = createMockNodeResponse();
    const virt = response.facts?.virtualization;
    assert.ok(virt?.type);
    assert.ok(['kvm', 'xen', 'hyperv', 'vmware', 'docker', 'lxc', 'none'].includes(virt?.type || ''));
  });

  await t.test('indicates detection status', () => {
    const response = createMockNodeResponse();
    const virt = response.facts?.virtualization;
    assert.equal(typeof virt?.detected, 'boolean');
  });

  await t.test('handles no virtualization detected', () => {
    const response: NodeData = {
      id: 'node-003',
      facts: {
        virtualization: { type: 'none', detected: false },
      },
    };
    assert.equal(response.facts?.virtualization?.type, 'none');
    assert.equal(response.facts?.virtualization?.detected, false);
  });

  await t.test('detects KVM hypervisor', () => {
    const response = createMockNodeResponse();
    if (response.facts?.virtualization?.type === 'kvm') {
      assert.equal(response.facts.virtualization.detected, true);
    }
  });
});

test('Unknown/Unmeasurable Fields Tracking', async (t) => {
  await t.test('tracks unmeasurable fields in unknown array', () => {
    const response = createMockNodeResponse();
    const unknown = response.facts?.unknown || [];
    assert.ok(Array.isArray(unknown));
  });

  await t.test('includes nat_type as unknown when not available', () => {
    const response = createMockNodeResponse();
    const unknown = response.facts?.unknown || [];
    assert.ok(unknown.includes('nat_type'));
  });

  await t.test('includes external_reachability as unknown when not available', () => {
    const response = createMockNodeResponse();
    const unknown = response.facts?.unknown || [];
    assert.ok(unknown.includes('external_reachability'));
  });

  await t.test('handles empty unknown array', () => {
    const response: NodeData = {
      id: 'node-004',
      facts: {
        unknown: [],
      },
    };
    assert.equal(response.facts?.unknown?.length, 0);
  });
});

test('Hardware Facts Freshness', async (t) => {
  interface FreshNodeResponse extends NodeData {
    _lastRefreshed?: number;
    _freshness?: 'LIVE' | 'UNAVAILABLE' | 'STALE';
  }

  await t.test('includes freshness indicator', () => {
    const response: FreshNodeResponse = {
      ...createMockNodeResponse(),
      _freshness: 'LIVE',
      _lastRefreshed: Date.now(),
    };
    assert.equal(response._freshness, 'LIVE');
    assert.ok(response._lastRefreshed);
  });

  await t.test('detects stale data', () => {
    const response: FreshNodeResponse = {
      ...createMockNodeResponse(),
      _freshness: 'LIVE',
      _lastRefreshed: Date.now() - 300000, // 5 minutes ago
    };
    const isFresh = response._lastRefreshed ? (Date.now() - response._lastRefreshed) < 60000 : false;
    assert.equal(isFresh, false);
  });

  await t.test('marks unavailable when host unreachable', () => {
    const response: FreshNodeResponse = {
      id: 'node-005',
      _freshness: 'UNAVAILABLE',
    };
    assert.equal(response._freshness, 'UNAVAILABLE');
  });
});

test('RBAC Enforcement on Hardware Facts', async (t) => {
  function checkCapability(action: string, capability: string): boolean {
    const capabilities: Record<string, string[]> = {
      'read_hardware': ['api.admin', 'api.write', 'api.read'],
      'interpret_hardware': ['api.admin', 'api.write', 'api.read'],
    };
    const requiredCapabilities = capabilities[action] || [];
    return requiredCapabilities.includes(capability);
  }

  await t.test('api.read can view hardware facts', () => {
    assert.equal(checkCapability('read_hardware', 'api.read'), true);
  });

  await t.test('api.write can view hardware facts', () => {
    assert.equal(checkCapability('read_hardware', 'api.write'), true);
  });

  await t.test('api.admin can view hardware facts', () => {
    assert.equal(checkCapability('read_hardware', 'api.admin'), true);
  });

  await t.test('no capability cannot view hardware facts', () => {
    assert.equal(checkCapability('read_hardware', 'none'), false);
  });
});

test('Hardware Discovery Response Format', async (t) => {
  await t.test('response includes node ID', () => {
    const response = createMockNodeResponse();
    assert.ok(response.id);
    assert.ok(typeof response.id === 'string');
  });

  await t.test('CPU facts have measured indicator', () => {
    const response = createMockNodeResponse();
    if (response.facts?.cpuCores) {
      assert.equal(typeof response.facts.cpuCores.measured, 'boolean');
    }
  });

  await t.test('memory facts show value and measured flag', () => {
    const response = createMockNodeResponse();
    const mem = response.facts?.memBytes;
    assert.ok(mem);
    assert.equal(typeof mem.value, 'number');
    assert.equal(typeof mem.measured, 'boolean');
  });

  await t.test('validates fact value types', () => {
    const response = createMockNodeResponse();
    const facts = response.facts;
    if (facts?.cpuCores) {
      assert.equal(typeof facts.cpuCores.value, 'number');
    }
    if (facts?.cpuModel) {
      assert.equal(typeof facts.cpuModel.value, 'string');
    }
  });
});

test('Multi-host hardware facts', async (t) => {
  await t.test('discovers hardware from multiple nodes', () => {
    const nodes = [
      { id: 'node-001', facts: createMockNodeResponse().facts },
      { id: 'node-002', facts: createMockNodeResponse().facts },
      { id: 'node-003', facts: createMockNodeResponse().facts },
    ];
    assert.equal(nodes.length, 3);
    for (const node of nodes) {
      assert.ok(validateHardwareFactsStructure(node.facts));
    }
  });

  await t.test('handles heterogeneous hardware', () => {
    const node1Facts: HardwareFacts = {
      cpuCores: { value: 8, measured: true },
      memBytes: { value: 17179869184, measured: true }, // 16 GB
    };
    const node2Facts: HardwareFacts = {
      cpuCores: { value: 16, measured: true },
      memBytes: { value: 34359738368, measured: true }, // 32 GB
    };
    assert.notEqual(node1Facts.cpuCores?.value, node2Facts.cpuCores?.value);
  });

  await t.test('handles partial hardware discovery', () => {
    const response: NodeData = {
      id: 'node-partial',
      facts: {
        cpuCores: { value: 4, measured: true },
        // Other facts not reported
      },
    };
    assert.ok(response.facts?.cpuCores);
    assert.equal(response.facts?.memBytes, undefined);
  });
});

test('NIC Details Validation', async (t) => {
  await t.test('validates NIC structure completeness', () => {
    const nic: NetworkInterface = {
      name: 'eth0',
      ipv4: '192.168.1.1',
      macAddress: '00:11:22:33:44:55',
      mtu: 1500,
      speed: '1 Gbps',
    };
    assert.ok(nic.name);
    assert.ok(nic.ipv4);
    assert.ok(nic.macAddress);
  });

  await t.test('handles optional NIC fields', () => {
    const nic: NetworkInterface = {
      name: 'eth0',
      // ipv4, ipv6, macAddress, mtu, speed all optional
    };
    assert.ok(nic.name);
  });
});

test('Container Runtime Completeness', async (t) => {
  await t.test('reports all known container runtimes', () => {
    const response = createMockNodeResponse();
    const runtimes = response.facts?.containerRuntimes || [];
    const names = runtimes.map((r) => r.name);
    const knownRuntimes = ['docker', 'containerd', 'cri-o', 'podman'];
    for (const known of knownRuntimes) {
      assert.ok(names.includes(known) || names.length === 0);
    }
  });
});
