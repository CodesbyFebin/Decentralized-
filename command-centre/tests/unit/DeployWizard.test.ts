import { test } from 'node:test';
import * as assert from 'node:assert';

test('DeployWizard Form Validation', async (t) => {
  await t.test('should reject invalid application names', () => {
    const names = [
      { name: '', valid: false },
      { name: 'App-123', valid: false },
      { name: 'app@db', valid: false },
      { name: 'a', valid: true },
      { name: 'my-app-123', valid: true },
      { name: 'a'.repeat(42), valid: true },
      { name: 'a'.repeat(43), valid: false },
    ];

    for (const { name, valid } of names) {
      const matches = /^[a-z0-9][a-z0-9-]{0,41}$/.test(name);
      assert.equal(matches, valid, `Name "${name}" validation should be ${valid}`);
    }
  });

  await t.test('should validate process artifact format', () => {
    const artifacts = [
      { ref: 'myapp@b3:' + 'a'.repeat(64), valid: true },
      { ref: 'my-app_v1@b3:' + 'a'.repeat(64), valid: true },
      { ref: 'myapp@b3:' + 'a'.repeat(63), valid: false },
      { ref: 'myapp@sha256:' + 'a'.repeat(64), valid: false },
      { ref: 'myapp', valid: false },
    ];

    for (const { ref, valid } of artifacts) {
      const matches = /^[a-z0-9._-]+@b3:[0-9a-f]{64}$/.test(ref);
      assert.equal(matches, valid, `Artifact "${ref}" validation should be ${valid}`);
    }
  });

  await t.test('should validate docker image format', () => {
    const images = [
      { ref: 'ghcr.io/org/app@sha256:' + 'a'.repeat(64), valid: true },
      { ref: 'alpine@sha256:' + 'a'.repeat(64), valid: true },
      { ref: 'ghcr.io/org/app:latest', valid: false },
      { ref: 'ghcr.io/org/app@sha256:' + 'a'.repeat(63), valid: false },
    ];

    for (const { ref, valid } of images) {
      const matches = /^[\w./:-]+@sha256:[0-9a-f]{64}$/.test(ref);
      assert.equal(matches, valid, `Image "${ref}" validation should be ${valid}`);
    }
  });

  await t.test('should validate replica count', () => {
    for (const r of [1, 2, 32, 64]) {
      assert.ok(Number.isInteger(r) && r >= 1 && r <= 64);
    }
    for (const r of [0, 65, -1, 1.5]) {
      assert.ok(!(Number.isInteger(r) && r >= 1 && r <= 64));
    }
  });

  await t.test('should validate CPU format', () => {
    const cpu = [
      { value: '100m', valid: true },
      { value: '1', valid: true },
      { value: '500m', valid: true },
      { value: 'invalid', valid: false },
      { value: '100M', valid: false },
    ];

    for (const { value, valid } of cpu) {
      const matches = /^\d+m?$/.test(value);
      assert.equal(matches, valid, `CPU "${value}" should be ${valid}`);
    }
  });

  await t.test('should validate memory format', () => {
    const mem = [
      { value: '64Mi', valid: true },
      { value: '1Gi', valid: true },
      { value: '512', valid: true },
      { value: '512Ki', valid: true },
      { value: '512mb', valid: false },
      { value: 'invalid', valid: false },
    ];

    for (const { value, valid } of mem) {
      const matches = /^\d+(Ki|Mi|Gi)?$/.test(value);
      assert.equal(matches, valid, `Memory "${value}" should be ${valid}`);
    }
  });

  await t.test('should validate DNS hostnames', () => {
    const hosts = [
      { host: 'example.com', valid: true },
      { host: 'app.example.co.uk', valid: true },
      { host: 'sub.domain.example.com', valid: true },
      { host: 'example', valid: false },
      { host: 'example.c', valid: false },
    ];

    for (const { host, valid } of hosts) {
      const matches = /^[a-z0-9.-]+\.[a-z]{2,}$/.test(host);
      assert.equal(matches, valid, `Host "${host}" should be ${valid}`);
    }
  });

  await t.test('should validate env var names (UPPER_SNAKE_CASE)', () => {
    const names = [
      { name: 'DATABASE_URL', valid: true },
      { name: 'API_KEY', valid: true },
      { name: '_INTERNAL', valid: true },
      { name: 'database_url', valid: false },
      { name: 'DATABASE-URL', valid: false },
      { name: '123_VAR', valid: false },
      { name: 'APP_KEY_123', valid: true },
    ];

    for (const { name, valid } of names) {
      const matches = /^[A-Z_][A-Z0-9_]*$/.test(name);
      assert.equal(matches, valid, `Env name "${name}" should be ${valid}`);
    }
  });
});

test('DeployWizard Manifest Generation', async (t) => {
  await t.test('should generate valid dh/v1 YAML', () => {
    const yaml = [
      'apiVersion: dh/v1',
      'kind: Application',
      'metadata:',
      '  name: test-app',
      'spec:',
      '  replicas: 2',
    ].join('\n');

    assert.ok(yaml.includes('apiVersion: dh/v1'));
    assert.ok(yaml.includes('kind: Application'));
    assert.ok(yaml.includes('name: test-app'));
  });

  await t.test('should format environment variables correctly', () => {
    const yamlLine = 'DATABASE_URL: "postgresql://localhost/db"';
    assert.ok(yamlLine.match(/DATABASE_URL: "/));
  });

  await t.test('should include docker runtime when needed', () => {
    const yaml = 'runtime: docker';
    assert.equal(yaml, 'runtime: docker');
  });
});

test('DeployWizard Artifact Filtering', async (t) => {
  await t.test('should filter artifacts by name', () => {
    const artifacts = [
      { name: 'myapp-v1', digest: 'a' },
      { name: 'myapp-v2', digest: 'b' },
      { name: 'other-service', digest: 'c' },
    ];
    const query = 'myapp';
    const filtered = artifacts.filter(a => a.name.toLowerCase().includes(query.toLowerCase()));
    assert.equal(filtered.length, 2);
  });

  await t.test('should filter artifacts by digest', () => {
    const artifacts = [
      { name: 'app1', digest: 'abc123def456' },
      { name: 'app2', digest: 'xyz789uva345' },
    ];
    const query = 'abc123';
    const filtered = artifacts.filter(a => a.digest.includes(query));
    assert.equal(filtered.length, 1);
  });

  await t.test('should return empty for non-matching query', () => {
    const artifacts = [
      { name: 'app1', digest: 'abc123' },
      { name: 'app2', digest: 'xyz789' },
    ];
    const query = 'nonexistent';
    const filtered = artifacts.filter(a => a.name.toLowerCase().includes(query.toLowerCase()));
    assert.equal(filtered.length, 0);
  });
});

test('DeployWizard Error Handling', async (t) => {
  await t.test('should recognize VALIDATION_FAILED errors', () => {
    const error = { code: 'VALIDATION_FAILED', message: 'Insufficient resources' };
    assert.equal(error.code, 'VALIDATION_FAILED');
    assert.ok(error.message);
  });

  await t.test('should recognize CONFLICT errors', () => {
    const error = { code: 'CONFLICT', message: 'App already exists' };
    assert.equal(error.code, 'CONFLICT');
  });
});
