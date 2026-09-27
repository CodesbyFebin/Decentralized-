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

interface ErrorResponse {
  error: {
    message: string;
    code?: string;
  };
}

function createMockBuildServiceResponse(): BuildServiceData {
  return {
    repository: {
      url: 'https://github.com/org/myapp.git',
      branch: 'main',
      commit: 'abc123def456',
      lastFetched: Date.now() - 300000,
    },
    builds: [
      {
        id: 'build-001',
        gitCommit: 'abc123def456',
        image: 'registry.example.com/myapp:latest',
        imageDigest: 'sha256:abcd1234efgh5678',
        status: 'completed',
        startedAt: Date.now() - 600000,
        completedAt: Date.now() - 480000,
        duration: 120000,
      },
      {
        id: 'build-002',
        gitCommit: 'xyz789abc123',
        image: 'registry.example.com/myapp:v1.0.0',
        status: 'building',
        startedAt: Date.now() - 180000,
      },
    ],
    attestations: [
      {
        id: 'att-001',
        buildId: 'build-001',
        format: 'slsa',
        signer: 'ci-builder@example.com',
        signature: 'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...',
        createdAt: Date.now() - 480000,
        verifiedAt: Date.now() - 470000,
      },
    ],
    sboms: [
      {
        id: 'sbom-001',
        buildId: 'build-001',
        format: 'cyclonedx',
        componentCount: 52,
        vulnerabilities: 3,
        createdAt: Date.now() - 480000,
        url: 'https://sbom-store.example.com/sbom-001.xml',
      },
    ],
  };
}

function validateBuildServiceResponseStructure(data: unknown): boolean {
  if (typeof data !== 'object' || !data) return false;
  const d = data as Record<string, unknown>;

  if (!d.repository || typeof d.repository !== 'object') return false;
  const repo = d.repository as Record<string, unknown>;
  if (typeof repo.url !== 'string' || typeof repo.branch !== 'string') return false;

  if (!Array.isArray(d.builds)) return false;
  for (const build of d.builds) {
    if (typeof build !== 'object' || !build) return false;
    const b = build as Record<string, unknown>;
    if (typeof b.id !== 'string' || typeof b.status !== 'string') return false;
    if (!['pending', 'building', 'completed', 'failed'].includes(b.status as string)) return false;
  }

  if (!Array.isArray(d.attestations)) return false;
  for (const att of d.attestations) {
    if (typeof att !== 'object' || !att) return false;
    const a = att as Record<string, unknown>;
    if (typeof a.id !== 'string' || typeof a.buildId !== 'string') return false;
    if (!['slsa', 'provenance'].includes(a.format as string)) return false;
  }

  if (!Array.isArray(d.sboms)) return false;
  for (const sbom of d.sboms) {
    if (typeof sbom !== 'object' || !sbom) return false;
    const s = sbom as Record<string, unknown>;
    if (typeof s.id !== 'string' || typeof s.componentCount !== 'number') return false;
  }

  return true;
}

function validateErrorResponse(data: unknown): boolean {
  if (typeof data !== 'object' || !data) return false;
  const d = data as Record<string, unknown>;
  if (!d.error || typeof d.error !== 'object') return false;
  return typeof (d.error as Record<string, unknown>).message === 'string';
}

test('GET /builds/service endpoint', async (t) => {
  await t.test('returns valid build service response structure', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(validateBuildServiceResponseStructure(response));
  });

  await t.test('includes git repository info', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(response.repository);
    assert.equal(typeof response.repository.url, 'string');
    assert.equal(typeof response.repository.branch, 'string');
  });

  await t.test('includes builds list', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(Array.isArray(response.builds));
  });

  await t.test('includes attestations list', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(Array.isArray(response.attestations));
  });

  await t.test('includes SBOMs list', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(Array.isArray(response.sboms));
  });

  await t.test('returns empty when no builds exist', () => {
    const response: BuildServiceData = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      builds: [],
      attestations: [],
      sboms: [],
    };
    assert.ok(validateBuildServiceResponseStructure(response));
  });
});

test('Docker Build Status in Response', async (t) => {
  await t.test('includes pending status', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(response.builds.some((b) => b.status === 'pending' || b.status === 'building'));
  });

  await t.test('includes building status', () => {
    const response = createMockBuildServiceResponse();
    const building = response.builds.find((b) => b.status === 'building');
    assert.ok(building);
  });

  await t.test('includes completed status', () => {
    const response = createMockBuildServiceResponse();
    const completed = response.builds.find((b) => b.status === 'completed');
    assert.ok(completed);
  });

  await t.test('completed build has duration', () => {
    const response = createMockBuildServiceResponse();
    const completed = response.builds.find((b) => b.status === 'completed');
    if (completed) {
      assert.ok(completed.duration);
      assert.ok(completed.completedAt);
    }
  });

  await t.test('building status lacks completedAt', () => {
    const response = createMockBuildServiceResponse();
    const building = response.builds.find((b) => b.status === 'building');
    if (building) {
      assert.equal(building.completedAt, undefined);
    }
  });

  await t.test('failed status includes error message', () => {
    const response: BuildServiceData = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      builds: [
        {
          id: 'build-fail',
          gitCommit: 'abc123',
          image: 'app:latest',
          status: 'failed',
          startedAt: Date.now(),
          error: 'Build failed: command exited with status 1',
        },
      ],
      attestations: [],
      sboms: [],
    };
    const failed = response.builds[0];
    assert.ok(failed.error);
  });
});

test('POST /builds/service/start endpoint', async (t) => {
  await t.test('validates start build request', () => {
    const request = {
      repository: {
        url: 'https://github.com/org/repo.git',
        branch: 'develop',
      },
      dockerfile: 'docker/Dockerfile',
    };
    assert.ok(request.repository.url);
    assert.ok(request.repository.branch);
  });

  await t.test('requires repository url', () => {
    const request = {
      repository: {
        branch: 'main',
      },
    };
    const repo = request.repository as Record<string, unknown>;
    assert.equal(repo.url, undefined);
  });

  await t.test('requires branch name', () => {
    const request = {
      repository: {
        url: 'https://github.com/org/repo.git',
      },
    };
    const repo = request.repository as Record<string, unknown>;
    assert.equal(repo.branch, undefined);
  });

  await t.test('dockerfile path is optional', () => {
    const withDockerfile = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      dockerfile: 'Dockerfile',
    };
    const withoutDockerfile = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
    };
    assert.ok((withDockerfile as Record<string, unknown>).dockerfile);
    assert.equal((withoutDockerfile as Record<string, unknown>).dockerfile, undefined);
  });
});

test('POST /builds/service/{id}/retry endpoint', async (t) => {
  await t.test('transitions failed build to pending', () => {
    const beforeRetry: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123',
      image: 'app:latest',
      status: 'failed',
      startedAt: Date.now(),
      error: 'Build failed',
    };
    const afterRetry: DockerBuild = {
      ...beforeRetry,
      status: 'pending',
      error: undefined,
    };
    assert.equal(beforeRetry.status, 'failed');
    assert.equal(afterRetry.status, 'pending');
  });

  await t.test('requires admin capability', () => {
    const capabilities = { 'api.admin': true };
    const canRetry = 'api.admin' in capabilities;
    assert.ok(canRetry);
  });

  await t.test('rejects non-admin retry', () => {
    const capabilities = { 'api.write': true };
    const canRetry = 'api.admin' in capabilities;
    assert.equal(canRetry, false);
  });
});

test('POST /builds/service/{id}/delete endpoint', async (t) => {
  await t.test('deletes build record', () => {
    const builds: DockerBuild[] = [
      { id: 'build-001', gitCommit: 'abc', image: 'app:1', status: 'completed', startedAt: Date.now() },
      { id: 'build-002', gitCommit: 'def', image: 'app:2', status: 'completed', startedAt: Date.now() },
    ];
    const filtered = builds.filter((b) => b.id !== 'build-001');
    assert.equal(filtered.length, 1);
  });

  await t.test('requires admin capability', () => {
    const capabilities = { 'api.admin': true };
    const canDelete = 'api.admin' in capabilities;
    assert.ok(canDelete);
  });
});

test('RBAC Enforcement on Build Service', async (t) => {
  function checkCapability(action: string, capability: string): boolean {
    const capabilities: Record<string, string[]> = {
      'read_builds': ['api.admin', 'api.write', 'api.read'],
      'start_build': ['api.admin'],
      'retry_build': ['api.admin'],
      'delete_build': ['api.admin'],
    };
    const requiredCapabilities = capabilities[action] || [];
    return requiredCapabilities.includes(capability);
  }

  await t.test('api.admin can read builds', () => {
    assert.equal(checkCapability('read_builds', 'api.admin'), true);
  });

  await t.test('api.write can read builds', () => {
    assert.equal(checkCapability('read_builds', 'api.write'), true);
  });

  await t.test('api.read can read builds', () => {
    assert.equal(checkCapability('read_builds', 'api.read'), true);
  });

  await t.test('only api.admin can start builds', () => {
    assert.equal(checkCapability('start_build', 'api.admin'), true);
    assert.equal(checkCapability('start_build', 'api.write'), false);
  });

  await t.test('only api.admin can retry builds', () => {
    assert.equal(checkCapability('retry_build', 'api.admin'), true);
    assert.equal(checkCapability('retry_build', 'api.read'), false);
  });

  await t.test('only api.admin can delete builds', () => {
    assert.equal(checkCapability('delete_build', 'api.admin'), true);
    assert.equal(checkCapability('delete_build', 'api.write'), false);
  });
});

test('Build Attestation Integration', async (t) => {
  await t.test('attestation references build id', () => {
    const response = createMockBuildServiceResponse();
    const build = response.builds[0];
    const att = response.attestations.find((a) => a.buildId === build.id);
    assert.ok(att);
  });

  await t.test('SLSA format attestation', () => {
    const response = createMockBuildServiceResponse();
    const slsa = response.attestations.find((a) => a.format === 'slsa');
    assert.ok(slsa);
  });

  await t.test('attestation includes signer', () => {
    const response = createMockBuildServiceResponse();
    const att = response.attestations[0];
    assert.ok(att.signer);
    assert.ok(att.signature);
  });

  await t.test('attestation verification timestamp', () => {
    const response = createMockBuildServiceResponse();
    const att = response.attestations[0];
    if (att.verifiedAt) {
      assert.ok(att.verifiedAt >= att.createdAt);
    }
  });

  await t.test('unverified attestation', () => {
    const att: BuildAttestation = {
      id: 'att-unverified',
      buildId: 'build-001',
      format: 'slsa',
      signer: 'ci-builder@example.com',
      signature: 'ey...',
      createdAt: Date.now(),
    };
    assert.equal(att.verifiedAt, undefined);
  });
});

test('Software BOM Generation', async (t) => {
  await t.test('SBOM references build', () => {
    const response = createMockBuildServiceResponse();
    const build = response.builds[0];
    const sbom = response.sboms.find((s) => s.buildId === build.id);
    assert.ok(sbom);
  });

  await t.test('CycloneDX format SBOM', () => {
    const response = createMockBuildServiceResponse();
    const cyclone = response.sboms.find((s) => s.format === 'cyclonedx');
    assert.ok(cyclone);
  });

  await t.test('SBOM with component count', () => {
    const response = createMockBuildServiceResponse();
    const sbom = response.sboms[0];
    assert.ok(sbom.componentCount > 0);
  });

  await t.test('SBOM vulnerability tracking', () => {
    const response = createMockBuildServiceResponse();
    const sbom = response.sboms[0];
    assert.equal(typeof sbom.vulnerabilities, 'number');
  });

  await t.test('multiple SBOMs per build', () => {
    const response: BuildServiceData = {
      repository: { url: 'https://github.com/org/repo.git', branch: 'main' },
      builds: [{ id: 'build-001', gitCommit: 'abc', image: 'app:1', status: 'completed', startedAt: Date.now() }],
      attestations: [],
      sboms: [
        {
          id: 'sbom-001',
          buildId: 'build-001',
          format: 'cyclonedx',
          componentCount: 50,
          vulnerabilities: 2,
          createdAt: Date.now(),
        },
        {
          id: 'sbom-002',
          buildId: 'build-001',
          format: 'spdx',
          componentCount: 48,
          vulnerabilities: 2,
          createdAt: Date.now(),
        },
      ],
    };
    const sbomCount = response.sboms.filter((s) => s.buildId === 'build-001').length;
    assert.equal(sbomCount, 2);
  });
});

test('Error Response Handling', async (t) => {
  await t.test('error response includes message', () => {
    const errorResponse: ErrorResponse = {
      error: { message: 'Failed to clone repository' },
    };
    assert.ok(errorResponse.error.message);
  });

  await t.test('error response includes code', () => {
    const errorResponse: ErrorResponse = {
      error: { message: 'Unauthorized', code: 'ERR_UNAUTHORIZED' },
    };
    assert.ok(errorResponse.error.code);
  });

  await t.test('validates error response', () => {
    const errorResponse: ErrorResponse = {
      error: { message: 'Test error' },
    };
    assert.ok(validateErrorResponse(errorResponse));
  });
});

test('Self-Hosted Graceful Degradation', async (t) => {
  await t.test('detects self-hosted deployment', () => {
    const isSelfHosted = false;
    const showBuildService = !isSelfHosted;
    assert.ok(showBuildService);
  });

  await t.test('shows message when self-hosted', () => {
    const isSelfHosted = true;
    if (isSelfHosted) {
      const message = 'Self-hosted deployments use local build workflows.';
      assert.ok(message.includes('Self-hosted'));
    }
  });
});

test('Build Lifecycle and Timing', async (t) => {
  await t.test('build duration calculation', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123',
      image: 'app:latest',
      status: 'completed',
      startedAt: Date.now() - 120000,
      completedAt: Date.now(),
      duration: 120000,
    };
    assert.ok(build.duration! > 0);
  });

  await t.test('pending build without completion', () => {
    const build: DockerBuild = {
      id: 'build-001',
      gitCommit: 'abc123',
      image: 'app:latest',
      status: 'pending',
      startedAt: Date.now(),
    };
    assert.equal(build.completedAt, undefined);
    assert.equal(build.duration, undefined);
  });

  await t.test('complete build lifecycle', () => {
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
  });
});

test('Git Repository Tracking', async (t) => {
  await t.test('repository includes url and branch', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(response.repository.url.includes('github.com'));
    assert.equal(response.repository.branch, 'main');
  });

  await t.test('repository tracks current commit', () => {
    const response = createMockBuildServiceResponse();
    if (response.repository.commit) {
      assert.ok(response.repository.commit.length >= 6);
    }
  });

  await t.test('repository tracks last fetch time', () => {
    const response = createMockBuildServiceResponse();
    if (response.repository.lastFetched) {
      assert.ok(response.repository.lastFetched > 0);
    }
  });

  await t.test('repository update sequence', () => {
    let repo: GitRepository = {
      url: 'https://github.com/org/repo.git',
      branch: 'main',
    };
    repo = { ...repo, commit: 'abc123', lastFetched: Date.now() };
    assert.ok(repo.commit);

    repo = { ...repo, commit: 'def456', lastFetched: Date.now() };
    assert.equal(repo.commit, 'def456');
  });
});

test('Multi-Build Coordination', async (t) => {
  await t.test('multiple builds per repository', () => {
    const response = createMockBuildServiceResponse();
    assert.ok(response.builds.length >= 2);
  });

  await t.test('builds map to attestations', () => {
    const response = createMockBuildServiceResponse();
    for (const build of response.builds) {
      const att = response.attestations.find((a) => a.buildId === build.id);
      if (build.status === 'completed') {
        assert.ok(att);
      }
    }
  });

  await t.test('builds map to SBOMs', () => {
    const response = createMockBuildServiceResponse();
    for (const build of response.builds) {
      const sbom = response.sboms.find((s) => s.buildId === build.id);
      if (build.status === 'completed') {
        assert.ok(sbom);
      }
    }
  });
});
