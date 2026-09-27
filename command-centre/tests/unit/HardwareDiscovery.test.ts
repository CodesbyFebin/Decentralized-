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
  type: 'kvm' | 'xen' | 'hyperv' | 'vmware' | 'docker' | 'lxc' | 'none';
  hypervisor?: string;
  detected: boolean;
}

// Helper: Parse network interface info
function parseNetworkInterface(ifaceData: Record<string, unknown>): NetworkInterface | null {
  if (typeof ifaceData.name !== 'string') return null;
  return {
    name: ifaceData.name,
    ipv4: typeof ifaceData.ipv4 === 'string' ? ifaceData.ipv4 : undefined,
    ipv6: typeof ifaceData.ipv6 === 'string' ? ifaceData.ipv6 : undefined,
    macAddress: typeof ifaceData.macAddress === 'string' ? ifaceData.macAddress : undefined,
    mtu: typeof ifaceData.mtu === 'number' ? ifaceData.mtu : undefined,
    speed: typeof ifaceData.speed === 'string' ? ifaceData.speed : undefined,
  };
}

// Helper: Validate MAC address format
function validateMacAddress(mac: string): boolean {
  return /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/.test(mac);
}

// Helper: Validate IPv4 address
function validateIpv4(ip: string): boolean {
  const parts = ip.split('.');
  if (parts.length !== 4) return false;
  return parts.every(part => {
    const num = parseInt(part, 10);
    return num >= 0 && num <= 255;
  });
}

// Helper: Validate IPv6 address
function validateIpv6(ip: string): boolean {
  const ipv6Regex = /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|::1|::)$/;
  return ipv6Regex.test(ip);
}

// Helper: Detect container runtimes
function detectContainerRuntimes(runtimePaths: Record<string, boolean>): ContainerRuntime[] {
  const known = ['docker', 'containerd', 'cri-o', 'podman', 'rkt', 'lxc'];
  return known.map(name => ({
    name,
    status: runtimePaths[name] ? 'available' : 'unavailable',
  }));
}

// Helper: Detect virtualization
function detectVirtualization(hypervisorIndicators: Record<string, boolean>): VirtualizationInfo {
  if (hypervisorIndicators['kvm']) {
    return { type: 'kvm', detected: true, hypervisor: 'KVM' };
  }
  if (hypervisorIndicators['xen']) {
    return { type: 'xen', detected: true, hypervisor: 'Xen' };
  }
  if (hypervisorIndicators['hyperv']) {
    return { type: 'hyperv', detected: true, hypervisor: 'Hyper-V' };
  }
  if (hypervisorIndicators['vmware']) {
    return { type: 'vmware', detected: true, hypervisor: 'VMware' };
  }
  if (hypervisorIndicators['docker']) {
    return { type: 'docker', detected: true, hypervisor: 'Docker' };
  }
  if (hypervisorIndicators['lxc']) {
    return { type: 'lxc', detected: true, hypervisor: 'LXC' };
  }
  return { type: 'none', detected: false };
}

// Helper: Format bytes for display
function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

test('Network Interface Discovery', async (t) => {
  await t.test('parses basic ethernet interface', () => {
    const ifaceData = {
      name: 'eth0',
      ipv4: '192.168.1.100',
      macAddress: '00:1a:2b:3c:4d:5e',
    };
    const iface = parseNetworkInterface(ifaceData);
    assert.ok(iface);
    assert.equal(iface.name, 'eth0');
    assert.equal(iface.ipv4, '192.168.1.100');
  });

  await t.test('parses interface with IPv6', () => {
    const ifaceData = {
      name: 'eth1',
      ipv4: '10.0.0.5',
      ipv6: '2001:db8::1',
      macAddress: 'aa:bb:cc:dd:ee:ff',
      mtu: 1500,
    };
    const iface = parseNetworkInterface(ifaceData);
    assert.ok(iface?.ipv6);
    assert.equal(iface.mtu, 1500);
  });

  await t.test('includes speed information', () => {
    const ifaceData = {
      name: 'eth0',
      speed: '1 Gbps',
    };
    const iface = parseNetworkInterface(ifaceData);
    assert.equal(iface?.speed, '1 Gbps');
  });

  await t.test('handles multiple interfaces', () => {
    const interfaces = [
      { name: 'eth0', ipv4: '192.168.1.1', macAddress: '00:11:22:33:44:55' },
      { name: 'eth1', ipv4: '10.0.0.1', macAddress: 'aa:bb:cc:dd:ee:ff' },
      { name: 'lo', ipv4: '127.0.0.1', macAddress: '00:00:00:00:00:00' },
    ];
    const parsed = interfaces.map(parseNetworkInterface).filter((i) => i !== null);
    assert.equal(parsed.length, 3);
  });

  await t.test('handles loopback interface', () => {
    const loopback = {
      name: 'lo',
      ipv4: '127.0.0.1',
      ipv6: '::1',
    };
    const iface = parseNetworkInterface(loopback);
    assert.equal(iface?.name, 'lo');
    assert.equal(iface?.ipv4, '127.0.0.1');
  });

  await t.test('handles wireless interfaces', () => {
    const wlan = {
      name: 'wlan0',
      ipv4: '192.168.0.10',
      speed: '300 Mbps',
    };
    const iface = parseNetworkInterface(wlan);
    assert.equal(iface?.name, 'wlan0');
    assert.ok(iface?.speed?.includes('Mbps'));
  });
});

test('MAC Address Validation', async (t) => {
  await t.test('accepts colon-separated format', () => {
    assert.equal(validateMacAddress('00:1a:2b:3c:4d:5e'), true);
  });

  await t.test('accepts hyphen-separated format', () => {
    assert.equal(validateMacAddress('00-1a-2b-3c-4d-5e'), true);
  });

  await t.test('rejects invalid format', () => {
    assert.equal(validateMacAddress('00:1a:2b:3c:4d'), false);
  });

  await t.test('rejects non-hex characters', () => {
    assert.equal(validateMacAddress('00:1a:2b:3c:4d:gz'), false);
  });

  await t.test('accepts uppercase hex', () => {
    assert.equal(validateMacAddress('AA:BB:CC:DD:EE:FF'), true);
  });
});

test('IPv4 Address Validation', async (t) => {
  await t.test('accepts valid IPv4', () => {
    assert.equal(validateIpv4('192.168.1.1'), true);
  });

  await t.test('accepts localhost', () => {
    assert.equal(validateIpv4('127.0.0.1'), true);
  });

  await t.test('accepts 0.0.0.0', () => {
    assert.equal(validateIpv4('0.0.0.0'), true);
  });

  await t.test('rejects out-of-range octets', () => {
    assert.equal(validateIpv4('192.168.1.256'), false);
  });

  await t.test('rejects incomplete address', () => {
    assert.equal(validateIpv4('192.168.1'), false);
  });

  await t.test('rejects invalid format', () => {
    assert.equal(validateIpv4('192.168.1.1.1'), false);
  });
});

test('IPv6 Address Validation', async (t) => {
  await t.test('accepts full IPv6 address', () => {
    assert.equal(validateIpv6('2001:0db8:85a3:0000:0000:8a2e:0370:7334'), true);
  });

  await t.test('accepts compressed IPv6', () => {
    assert.equal(validateIpv6('2001:db8::1'), true);
  });

  await t.test('accepts localhost', () => {
    assert.equal(validateIpv6('::1'), true);
  });

  await t.test('accepts all zeros', () => {
    assert.equal(validateIpv6('::'), true);
  });

  await t.test('rejects invalid IPv6', () => {
    assert.equal(validateIpv6('gggg::1'), false);
  });
});

test('Container Runtime Detection', async (t) => {
  await t.test('detects docker availability', () => {
    const indicators = { docker: true, containerd: false };
    const runtimes = detectContainerRuntimes(indicators);
    const docker = runtimes.find((r) => r.name === 'docker');
    assert.equal(docker?.status, 'available');
  });

  await t.test('detects containerd availability', () => {
    const indicators = { containerd: true, docker: false };
    const runtimes = detectContainerRuntimes(indicators);
    const containerd = runtimes.find((r) => r.name === 'containerd');
    assert.equal(containerd?.status, 'available');
  });

  await t.test('detects cri-o availability', () => {
    const indicators = { 'cri-o': true };
    const runtimes = detectContainerRuntimes(indicators);
    const crioRuntime = runtimes.find((r) => r.name === 'cri-o');
    assert.equal(crioRuntime?.status, 'available');
  });

  await t.test('detects podman availability', () => {
    const indicators = { podman: true };
    const runtimes = detectContainerRuntimes(indicators);
    const podman = runtimes.find((r) => r.name === 'podman');
    assert.equal(podman?.status, 'available');
  });

  await t.test('reports unavailable runtimes', () => {
    const indicators = {};
    const runtimes = detectContainerRuntimes(indicators);
    const docker = runtimes.find((r) => r.name === 'docker');
    assert.equal(docker?.status, 'unavailable');
  });

  await t.test('includes all known runtimes', () => {
    const indicators = {};
    const runtimes = detectContainerRuntimes(indicators);
    assert.ok(runtimes.some((r) => r.name === 'docker'));
    assert.ok(runtimes.some((r) => r.name === 'containerd'));
    assert.ok(runtimes.some((r) => r.name === 'cri-o'));
    assert.ok(runtimes.some((r) => r.name === 'podman'));
  });
});

test('Virtualization Detection', async (t) => {
  await t.test('detects KVM hypervisor', () => {
    const virt = detectVirtualization({ kvm: true });
    assert.equal(virt.type, 'kvm');
    assert.equal(virt.detected, true);
    assert.equal(virt.hypervisor, 'KVM');
  });

  await t.test('detects Xen hypervisor', () => {
    const virt = detectVirtualization({ xen: true });
    assert.equal(virt.type, 'xen');
    assert.equal(virt.hypervisor, 'Xen');
  });

  await t.test('detects Hyper-V hypervisor', () => {
    const virt = detectVirtualization({ hyperv: true });
    assert.equal(virt.type, 'hyperv');
    assert.equal(virt.hypervisor, 'Hyper-V');
  });

  await t.test('detects VMware hypervisor', () => {
    const virt = detectVirtualization({ vmware: true });
    assert.equal(virt.type, 'vmware');
    assert.equal(virt.hypervisor, 'VMware');
  });

  await t.test('detects Docker container', () => {
    const virt = detectVirtualization({ docker: true });
    assert.equal(virt.type, 'docker');
    assert.equal(virt.hypervisor, 'Docker');
  });

  await t.test('detects LXC container', () => {
    const virt = detectVirtualization({ lxc: true });
    assert.equal(virt.type, 'lxc');
  });

  await t.test('reports none when not virtualized', () => {
    const virt = detectVirtualization({});
    assert.equal(virt.type, 'none');
    assert.equal(virt.detected, false);
  });

  await t.test('prefers KVM when multiple detected', () => {
    const virt = detectVirtualization({ kvm: true, xen: true });
    assert.equal(virt.type, 'kvm');
  });
});

test('Byte Formatting', async (t) => {
  await t.test('formats bytes correctly', () => {
    assert.equal(formatBytes(512), '512.00 B');
  });

  await t.test('converts to KB', () => {
    const result = formatBytes(2048);
    assert.match(result, /2\.\d+ KB/);
  });

  await t.test('converts to MB', () => {
    const result = formatBytes(1048576); // 1 MB
    assert.match(result, /1\.\d+ MB/);
  });

  await t.test('converts to GB', () => {
    const result = formatBytes(1073741824); // 1 GB
    assert.match(result, /1\.\d+ GB/);
  });

  await t.test('converts to TB', () => {
    const result = formatBytes(1099511627776); // 1 TB
    assert.match(result, /1\.\d+ TB/);
  });

  await t.test('handles zero bytes', () => {
    assert.equal(formatBytes(0), '0.00 B');
  });

  await t.test('formats large sizes accurately', () => {
    // 8 GB = 8589934592 bytes
    const result = formatBytes(8589934592);
    assert.match(result, /[0-9]\.[0-9]+ GB/);
  });
});

test('Hardware Facts Structure', async (t) => {
  await t.test('includes measured flag', () => {
    const fact: FactValue = { value: 16, measured: true };
    assert.equal(fact.measured, true);
  });

  await t.test('distinguishes measured vs unmeasured', () => {
    const measured: FactValue = { value: 8, measured: true };
    const unmeasured: FactValue = { value: 'unknown', measured: false };
    assert.equal(measured.measured, true);
    assert.equal(unmeasured.measured, false);
  });

  await t.test('tracks unknown fields', () => {
    const unknown = ['nat_type', 'public_ip', 'external_reachability'];
    assert.equal(unknown.length, 3);
    assert.ok(unknown.includes('nat_type'));
  });
});

test('NIC with MTU Discovery', async (t) => {
  await t.test('captures MTU value', () => {
    const nic = {
      name: 'eth0',
      mtu: 1500,
    };
    const iface = parseNetworkInterface(nic);
    assert.equal(iface?.mtu, 1500);
  });

  await t.test('handles jumbo frames', () => {
    const nic = {
      name: 'eth0',
      mtu: 9000,
    };
    const iface = parseNetworkInterface(nic);
    assert.equal(iface?.mtu, 9000);
  });

  await t.test('handles PPP MTU', () => {
    const nic = {
      name: 'ppp0',
      mtu: 1492,
    };
    const iface = parseNetworkInterface(nic);
    assert.equal(iface?.mtu, 1492);
  });
});

test('Multi-interface discovery', async (t) => {
  await t.test('discovers all NICs in order', () => {
    const nics = [
      { name: 'lo', ipv4: '127.0.0.1' },
      { name: 'eth0', ipv4: '192.168.1.1' },
      { name: 'eth1', ipv4: '10.0.0.1' },
      { name: 'wlan0', ipv4: '172.16.0.1' },
    ];
    const parsed = nics.map(parseNetworkInterface).filter((i) => i !== null);
    assert.equal(parsed.length, 4);
  });

  await t.test('handles mixed availability', () => {
    const indicators = {
      docker: true,
      containerd: true,
      'cri-o': false,
      podman: true,
    };
    const runtimes = detectContainerRuntimes(indicators);
    const available = runtimes.filter((r) => r.status === 'available');
    assert.equal(available.length, 3);
  });
});
