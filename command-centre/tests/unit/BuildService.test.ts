import { test } from 'node:test';
import assert from 'node:assert';

interface GitRepository {
  url: string;
  branch: string;
  commit?: string;
  lastFetched?: number;
}

interface DockerBuild {
  id: string;
  gitCommit: string;
  image: string;
  imageDigest?: string;
  status: 'pending' | 'building' | 'completed' | 'failed';
  startedAt: number;
  completedAt?: number;
  duration?: number;
  error?: string;
}

interface BuildAttestation {
  id: string;
  buildId: string;
  format: 'slsa' | 'provenance';
  signer: string;
  signature: string;
  createdAt: number;
  verifiedAt?: number;
}

interface SoftwareBOM {
  id: string;
  buildId: string;
  format: 'cyclonedx' | 'spdx';
  componentCount: number;
  vulnerabilities: number;
  createdAt: number;
  url?: string;
}

interface BuildServiceData {
  repository: GitRepository;
  builds: DockerBuild[];
  attestations: BuildAttestation[];
  sboms: SoftwareBOM[];
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'pending':
      return 'amber';
    case 'building':
      return 'blue';
    case 'completed':
      return 'emerald';
    case 'failed':
      return 'rose';
    default:
      return 'slate';
  }
}

function validateGitRepository(repo: unknown): boolean {
  if (typeof repo !== 'object' || !repo) return false;
  const r = repo as Record<string, unknown>;
  return typeof r.url === 'string' && typeof r.branch === 'string' && r.url.length > 0 && r.branch.length > 0;
}

function validateDockerBuild(build: unknown): boolean {
  if (typeof build !== 'object' || !build) return false;
  const b = build as Record<string, unknown>;
  return (
    typeof b.id === 'string' &&
    typeof b.gitCommit === 'string' &&
    typeof b.image === 'string' &&
    ['pending', 'building', 'completed', 'failed'].includes(b.status as string) &&
    typeof b.startedAt === 'number'
  );
}

function validateBuildAttestation(att: unknown): boolean {
  if (typeof att !== 'object' || !att) return false;
  const a = att as Record<string, unknown>;
  return (
    typeof a.id === 'string' &&
    typeof a.buildId === 'string' &&
    ['slsa', 'provenance'].includes(a.format as string) &&
    typeof a.signer === 'string' &&
    typeof a.signature === 'string' &&
    typeof a.createdAt === 'number'
  );
}

function validateSoftwareBOM(sbom: unknown): boolean {
  if (typeof sbom !== 'object' || !sbom) return false;
  const s = sbom as Record<string, unknown>;
  return (
    typeof s.id === 'string' &&
    typeof s.buildId === 'string' &&
    ['cyclonedx', 'spdx'].includes(s.format as string) &&
    typeof s.componentCount === 'number' &&
    typeof s.vulnerabilities === 'number' &&
    typeof s.createdAt === 'number'
  );
}

function validateBuildServiceData(data: unknown): boolean {
  if (typeof data !== 'object' || !data) return false;
  const d = data as Record<string, unknown>;

  if (!validateGitRepository(d.repository)) return false;
  if (!Array.isArray(d.builds)) return false;
  for (const build of d.builds) {
    if (!validateDockerBuild(build)) return false;
  }
  if (!Array.isArray(d.attestations)) return false;
  for (const att of d.attestations) {
    if (!validateBuildAttestation(att)) return false;
  }
  if (!Array.isArray(d.sboms)) return false;
  for (const sbom of d.sboms) {
    if (!validateSoftwareBOM(sbom)) return false;
  }

  return true;
}

test('Git Repository Validation', async (t) => {
  await t.test('validates valid git repository', () => {
    const repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
    };
    assert.ok(validateGitRepository(repo));
  });

  await t.test('rejects missing url', () => {
    const repo = { branch: 'main' };
    assert.equal(validateGitRepository(repo), false);
  });

  await t.test('rejects missing branch', () => {
    const repo = { url: 'https://github.com/org/repo.git' };
    assert.equal(validateGitRepository(repo), false);
  });

  await t.test('rejects empty url', () => {
    const repo: GitRepository = { url: '', branch: 'main' };
    assert.equal(validateGitRepository(repo), false);
  });

  await t.test('includes optional commit hash', () => {
    const repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
      commit: 'abc123def456',
    };
    assert.ok(repo.commit);
  });

  await t.test('includes optional lastFetched timestamp', () => {
    const repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
      lastFetched: Date.now(),
    };
    assert.ok(repo.lastFetched);
  });
});

test('Docker Build Validation', async (t) => {
  await t.test('validates complete docker build', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123def456',
      image: 'registry.example.com/app:latest',
      status: 'completed',
      startedAt: Date.now() - 120000,
      completedAt: Date.now(),
      duration: 120000,
    };
    assert.ok(validateDockerBuild(build));
  });

  await t.test('validates pending build', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123def456',
      image: 'registry.example.com/app:latest',
      status: 'pending',
      startedAt: Date.now(),
    };
    assert.ok(validateDockerBuild(build));
  });

  await t.test('validates building status', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123def456',
      image: 'registry.example.com/app:latest',
      status: 'building',
      startedAt: Date.now() - 30000,
    };
    assert.ok(validateDockerBuild(build));
  });

  await t.test('validates failed status with error', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123def456',
      image: 'registry.example.com/app:latest',
      status: 'failed',
      startedAt: Date.now() - 60000,
      completedAt: Date.now(),
      error: 'Dockerfile not found',
    };
    assert.ok(validateDockerBuild(build));
  });

  await t.test('rejects invalid status', () => {
    const build = {
      id: 'build-001',
      gitCommit: 'abc123def456',
      image: 'registry.example.com/app:latest',
      status: 'unknown',
      startedAt: Date.now(),
    };
    assert.equal(validateDockerBuild(build), false);
  });

  await t.test('includes optional imageDigest', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123def456',
      image: 'registry.example.com/app:latest',
      imageDigest: 'sha256:abcdef1234567890',
      status: 'completed',
      startedAt: Date.now(),
    };
    assert.ok(build.imageDigest);
  });
});

test('Duration Formatting', async (t) => {
  await t.test('formats milliseconds', () => {
    assert.equal(formatDuration(500), '500ms');
  });

  await t.test('formats seconds', () => {
    assert.equal(formatDuration(5000), '5.0s');
  });

  await t.test('formats minutes', () => {
    assert.equal(formatDuration(120000), '2.0m');
  });

  await t.test('formats fractional minutes', () => {
    assert.equal(formatDuration(90000), '1.5m');
  });

  await t.test('handles exact boundaries', () => {
    const atBoundary = formatDuration(1000);
    assert.ok(atBoundary.includes('s'));
  });
});

test('Status Color Mapping', async (t) => {
  await t.test('pending maps to amber', () => {
    assert.equal(getStatusColor('pending'), 'amber');
  });

  await t.test('building maps to blue', () => {
    assert.equal(getStatusColor('building'), 'blue');
  });

  await t.test('completed maps to emerald', () => {
    assert.equal(getStatusColor('completed'), 'emerald');
  });

  await t.test('failed maps to rose', () => {
    assert.equal(getStatusColor('failed'), 'rose');
  });

  await t.test('unknown status maps to slate', () => {
    assert.equal(getStatusColor('unknown'), 'slate');
  });
});

test('Build Attestation Validation', async (t) => {
  await t.test('validates SLSA attestation', () => {
    const att: BuildAttestation = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'slsa',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    assert.ok(validateBuildAttestation(att));
  });

  await t.test('validates provenance attestation', () => {
    const att: BuildAttestation = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'provenance',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    assert.ok(validateBuildAttestation(att));
  });

  await t.test('includes optional verifiedAt', () => {
    const att: BuildAttestation = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'slsa',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now() - 3600000,
      verifiedAt: Date.now(),
    };
    assert.ok(att.verifiedAt);
  });

  await t.test('rejects invalid format', () => {
    const att = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'invalid',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    assert.equal(validateBuildAttestation(att), false);
  });
});

test('Software BOM Validation', async (t) => {
  await t.test('validates CycloneDX BOM', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'cyclonedx',
      componentCount: 42,
      vulnerabilities: 3,
      createdAt: Date.now(),
    };
    assert.ok(validateSoftwareBOM(sbom));
  });

  await t.test('validates SPDX BOM', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'spdx',
      componentCount: 156,
      vulnerabilities: 0,
      createdAt: Date.now(),
    };
    assert.ok(validateSoftwareBOM(sbom));
  });

  await t.test('tracks vulnerability count', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'cyclonedx',
      componentCount: 50,
      vulnerabilities: 12,
      createdAt: Date.now(),
    };
    assert.equal(sbom.vulnerabilities, 12);
  });

  await t.test('handles zero vulnerabilities', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'cyclonedx',
      componentCount: 30,
      vulnerabilities: 0,
      createdAt: Date.now(),
    };
    assert.equal(sbom.vulnerabilities, 0);
  });

  await t.test('includes optional url', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'cyclonedx',
      componentCount: 42,
      vulnerabilities: 1,
      createdAt: Date.now(),
      url: 'https://sbom-store.example.com/sbom-001.xml',
    };
    assert.ok(sbom.url);
  });
});

test('Build Service Data Validation', async (t) => {
  await t.test('validates complete build service data', () => {
    const data: BuildServiceData = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      builds: [
        {
          id: 'build-001',
          gitCommit: 'abc123',
          image: 'app:latest',
          status: 'completed',
          startedAt: Date.now(),
        },
      ],
      attestations: [
        {
          id: 'att-001',
          buildId: 'build-001',
          format: 'slsa',
          signer: 'builder@example.com',
          signature: 'ey...',
          createdAt: Date.now(),
        },
      ],
      sboms: [
        {
          id: 'sbom-001',
          buildId: 'build-001',
          format: 'cyclonedx',
          componentCount: 42,
          vulnerabilities: 1,
          createdAt: Date.now(),
        },
      ],
    };
    assert.ok(validateBuildServiceData(data));
  });

  await t.test('validates empty builds', () => {
    const data: BuildServiceData = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      builds: [],
      attestations: [],
      sboms: [],
    };
    assert.ok(validateBuildServiceData(data));
  });

  await t.test('rejects invalid repository', () => {
    const data = {
      repository: { url: '' },
      builds: [],
      attestations: [],
      sboms: [],
    };
    assert.equal(validateBuildServiceData(data), false);
  });

  await t.test('rejects invalid build in list', () => {
    const data = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      builds: [{ id: 'build-001' }],
      attestations: [],
      sboms: [],
    };
    assert.equal(validateBuildServiceData(data), false);
  });
});

test('Build Lifecycle', async (t) => {
  await t.test('complete lifecycle: pending → building → completed', () => {
    let build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123',
      image: 'app:latest',
      status: 'pending',
      startedAt: Date.now(),
    };
    assert.equal(build.status, 'pending');

    build = { ...build, status: 'building' };
    assert.equal(build.status, 'building');

    build = {
      ...build,
      status: 'completed',
      completedAt: Date.now(),
      duration: 120000,
      imageDigest: 'sha256:abc123',
    };
    assert.equal(build.status, 'completed');
    assert.ok(build.imageDigest);
  });

  await t.test('failure lifecycle: pending → building → failed', () => {
    let build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123',
      image: 'app:latest',
      status: 'pending',
      startedAt: Date.now(),
    };

    build = { ...build, status: 'building' };
    assert.equal(build.status, 'building');

    build = {
      ...build,
      status: 'failed',
      completedAt: Date.now(),
      error: 'Build failed: command exited with status 1',
    };
    assert.equal(build.status, 'failed');
    assert.ok(build.error);
  });

  await t.test('quick failure: pending → failed', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123',
      image: 'app:latest',
      status: 'failed',
      startedAt: Date.now(),
      completedAt: Date.now(),
      error: 'Dockerfile not found',
    };
    assert.equal(build.status, 'failed');
  });
});

test('Multi-Build Scenarios', async (t) => {
  await t.test('multiple builds from same commit', () => {
    const commit = 'abc123def456';
    const builds: DockerBuild[] = [
      {
        id: 'build-001',
        gitCommit: commit,
        image: 'app:latest',
        status: 'completed',
        startedAt: Date.now(),
      },
      {
        id: 'build-002',
        gitCommit: commit,
        image: 'app:v1.0.0',
        status: 'completed',
        startedAt: Date.now(),
      },
    ];
    assert.equal(builds.filter((b) => b.gitCommit === commit).length, 2);
  });

  await t.test('heterogeneous build statuses', () => {
    const builds: DockerBuild[] = [
      {
        id: 'build-001',
        gitCommit: 'abc123',
        image: 'app:v1.0.0',
        status: 'completed',
        startedAt: Date.now(),
      },
      {
        id: 'build-002',
        gitCommit: 'def456',
        image: 'app:v1.0.1',
        status: 'building',
        startedAt: Date.now(),
      },
      {
        id: 'build-003',
        gitCommit: 'ghi789',
        image: 'app:v1.0.2',
        status: 'failed',
        startedAt: Date.now(),
      },
    ];
    const statuses = new Set(builds.map((b) => b.status));
    assert.equal(statuses.size, 3);
  });

  await t.test('accumulate build attestations and sboms', () => {
    const builds: DockerBuild[] = [
      { id: 'build-001', gitCommit: 'abc', image: 'app:1', status: 'completed', startedAt: Date.now() },
      { id: 'build-002', gitCommit: 'def', image: 'app:2', status: 'completed', startedAt: Date.now() },
      { id: 'build-003', gitCommit: 'ghi', image: 'app:3', status: 'completed', startedAt: Date.now() },
    ];
    const attestations = builds.map((b) => ({
      id: `att-${b.id}`,
      buildId: b.id,
      format: 'slsa' as const,
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    }));
    assert.equal(attestations.length, builds.length);
  });
});

test('Attestation Verification', async (t) => {
  await t.test('attestation without verification', () => {
    const att: BuildAttestation = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'slsa',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    assert.equal(att.verifiedAt, undefined);
  });

  await t.test('attestation with verification timestamp', () => {
    const created = Date.now() - 3600000;
    const att: BuildAttestation = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'slsa',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: created,
      verifiedAt: Date.now(),
    };
    assert.ok(att.verifiedAt! > att.createdAt);
  });

  await t.test('verification tracks signature format', () => {
    const slsaAtt: BuildAttestation = {
      id: 'att-001',
      buildId: 'build-001',
      format: 'slsa',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    const provAtt: BuildAttestation = {
      id: 'att-002',
      buildId: 'build-001',
      format: 'provenance',
      signer: 'builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    assert.notEqual(slsaAtt.format, provAtt.format);
  });
});

test('Component and Vulnerability Tracking', async (t) => {
  await t.test('SBOM with high vulnerability count', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'cyclonedx',
      componentCount: 150,
      vulnerabilities: 18,
      createdAt: Date.now(),
    };
    assert.equal(sbom.vulnerabilities, 18);
  });

  await t.test('SBOM with moderate component count', () => {
    const sbom: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'spdx',
      componentCount: 45,
      vulnerabilities: 2,
      createdAt: Date.now(),
    };
    assert.ok(sbom.componentCount > 0 && sbom.componentCount < 100);
  });

  await t.test('vulnerability severity categorization', () => {
    const critical: SoftwareBOM = {
      id: 'sbom-001',
      buildId: 'build-001',
      format: 'cyclonedx',
      componentCount: 50,
      vulnerabilities: 15,
      createdAt: Date.now(),
    };
    const moderate: SoftwareBOM = {
      id: 'sbom-002',
      buildId: 'build-002',
      format: 'cyclonedx',
      componentCount: 50,
      vulnerabilities: 2,
      createdAt: Date.now(),
    };
    assert.ok(critical.vulnerabilities > moderate.vulnerabilities);
  });
});

test('Git Repository Update Tracking', async (t) => {
  await t.test('repo without lastFetched', () => {
    const repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
    };
    assert.equal(repo.lastFetched, undefined);
  });

  await t.test('repo with lastFetched timestamp', () => {
    const repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
      lastFetched: Date.now(),
    };
    assert.ok(repo.lastFetched);
  });

  await t.test('repo with commit hash', () => {
    const repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
      commit: 'a1b2c3d4e5f6',
    };
    assert.equal(repo.commit?.length, 12);
  });

  await t.test('repo update sequence', () => {
    let repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
    };

    repo = { ...repo, commit: 'abc123', lastFetched: Date.now() };
    assert.ok(repo.commit);
    assert.ok(repo.lastFetched);

    repo = { ...repo, commit: 'def456', lastFetched: Date.now() };
    assert.equal(repo.commit, 'def456');
  });
});
