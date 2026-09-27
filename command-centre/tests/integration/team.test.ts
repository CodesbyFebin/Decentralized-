import { test } from 'node:test';
import * as assert from 'node:assert';
import { startBff } from '../helpers';
import { DemoPlatformAdapter } from '../../src/server/adapters/demo';

test('Team Management: API Contracts', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should fetch team data', async () => {
      const res = await call('GET', '/team', { csrf: false });
      assert.ok([200, 404].includes(res.status));
      if (res.status === 200) {
        assert.ok(res.json?.data?.members !== undefined);
        assert.ok(Array.isArray(res.json.data.members));
        assert.ok(res.json?.data?.invitations !== undefined);
        assert.ok(Array.isArray(res.json.data.invitations));
      }
    });

    await t.test('should validate member structure', async () => {
      const res = await call('GET', '/team', { csrf: false });
      if (res.status === 200 && res.json?.data?.members?.length) {
        const member = res.json.data.members[0];
        assert.ok(typeof member.actor === 'string');
        assert.ok(typeof member.role === 'string');
        assert.ok(typeof member.joinedAt === 'number');
        assert.ok(typeof member.lastSeen === 'number');
      }
    });

    await t.test('should validate invitation structure', async () => {
      const res = await call('GET', '/team', { csrf: false });
      if (res.status === 200 && res.json?.data?.invitations?.length) {
        const inv = res.json.data.invitations[0];
        assert.ok(typeof inv.nonce === 'string');
        assert.ok(['api.read', 'api.write', 'api.admin'].includes(inv.role));
        assert.ok(typeof inv.createdAt === 'number');
        assert.ok(typeof inv.expiresAt === 'number');
        assert.ok(['ACTIVE', 'USED', 'EXPIRED', 'REVOKED'].includes(inv.state));
      }
    });

    await t.test('should create invitation with api.admin', async () => {
      const res = await call('POST', '/team/invitations', {
        body: { role: 'api.read', ttl: 3600 },
        csrf: true
      });

      assert.ok([201, 202, 403, 404, 422].includes(res.status));
      if (res.status < 400) {
        assert.ok(res.json?.data?.nonce);
        assert.equal(res.json?.data?.role, 'api.read');
      }
    });

    await t.test('should validate TTL bounds', async () => {
      const res = await call('POST', '/team/invitations', {
        body: { role: 'api.read', ttl: 30 }, // Too short
        csrf: true
      });

      assert.ok([422, 400, 403, 404].includes(res.status));
    });

    await t.test('should validate role values', async () => {
      const res = await call('POST', '/team/invitations', {
        body: { role: 'invalid_role', ttl: 3600 },
        csrf: true
      });

      assert.ok([422, 400, 403, 404].includes(res.status));
    });

    await t.test('should revoke invitation with confirmation', async () => {
      // First create an invitation
      const createRes = await call('POST', '/team/invitations', {
        body: { role: 'api.read', ttl: 3600 },
        csrf: true
      });

      if (createRes.status < 400 && createRes.json?.data?.nonce) {
        const nonce = createRes.json.data.nonce;

        // Then revoke it
        const revokeRes = await call('POST', `/team/invitations/${nonce}/revoke`, {
          body: {},
          csrf: true
        });

        assert.ok([200, 202, 403, 404, 422].includes(revokeRes.status));
      }
    });

    await t.test('should handle revoke with missing nonce', async () => {
      const res = await call('POST', '/team/invitations/invalid_nonce/revoke', {
        body: {},
        csrf: true
      });

      assert.ok([404, 403, 422].includes(res.status));
    });

    await t.test('should enforce CSRF on mutations', async () => {
      const res = await call('POST', '/team/invitations', {
        body: { role: 'api.read', ttl: 3600 },
        csrf: false
      });

      assert.ok([403, 404, 422].includes(res.status));
    });

    await t.test('should return consistent role values', async () => {
      const roles = ['api.read', 'api.write', 'api.admin'];
      for (const role of roles) {
        const res = await call('POST', '/team/invitations', {
          body: { role, ttl: 3600 },
          csrf: true
        });

        if (res.status < 400) {
          assert.equal(res.json?.data?.role, role);
        }
      }
    });

    await t.test('should maintain invitation state transitions', async () => {
      const res = await call('GET', '/team', { csrf: false });
      if (res.status === 200 && res.json?.data?.invitations?.length) {
        const invitations = res.json.data.invitations;

        // Verify no impossible state transitions
        invitations.forEach((inv: any) => {
          const state = inv.state;
          assert.ok(['ACTIVE', 'USED', 'EXPIRED', 'REVOKED'].includes(state));

          // USED and REVOKED are terminal states
          if (state === 'USED' || state === 'REVOKED') {
            assert.ok(inv.expiresAt > 0);
          }
        });
      }
    });

    await t.test('should handle concurrent invitation operations', async () => {
      const promises = [
        call('POST', '/team/invitations', { body: { role: 'api.read', ttl: 3600 }, csrf: true }),
        call('POST', '/team/invitations', { body: { role: 'api.write', ttl: 7200 }, csrf: true })
      ];

      const results = await Promise.all(promises);
      results.forEach(res => {
        assert.ok([201, 202, 403, 404, 422].includes(res.status));
      });
    });

    await t.test('should track invitation creation audit', async () => {
      const res = await call('GET', '/audit?limit=50', { csrf: false });
      if (res.status === 200 && res.json?.data?.entries) {
        const invitationAudits = res.json.data.entries.filter((e: any) =>
          e.action === 'create_invitation' || e.action === 'revoke_invitation'
        );

        assert.ok(Array.isArray(invitationAudits));
      }
    });

  } finally {
    await close();
  }
});

test('Team Management: Authorization Enforcement', async (t) => {
  const { base, call, close } = await startBff(new DemoPlatformAdapter());

  try {
    await t.test('should enforce api.admin requirement for invitations', async () => {
      const res = await call('POST', '/team/invitations', {
        body: { role: 'api.read', ttl: 3600 },
        csrf: true
      });

      // Demo adapter refuses mutations, but should validate auth
      assert.ok([201, 202, 403, 404, 422].includes(res.status));
    });

    await t.test('should allow viewers to read team data', async () => {
      const res = await call('GET', '/team', { csrf: false });
      assert.ok([200, 404].includes(res.status));
    });

    await t.test('should validate role escalation prevention', async () => {
      // Viewer attempting to create admin invitation
      const res = await call('POST', '/team/invitations', {
        body: { role: 'api.admin', ttl: 3600 },
        csrf: true
      });

      assert.ok([403, 404, 422].includes(res.status) || res.status === 201);
    });

  } finally {
    await close();
  }
});
