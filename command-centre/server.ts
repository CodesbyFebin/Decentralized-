/**
 * Command Centre server: static/Vite front end + the BFF under /api/v1.
 *
 * The control plane (dh-control) is authoritative. The adapter is chosen
 * explicitly by PLATFORM_ADAPTER; there is no fallback between adapters.
 */
import express from 'express';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import dotenv from 'dotenv';
import { loadConfig, ConfigError } from './src/server/config';
import { ControlPlaneAdapter } from './src/server/adapters/controlPlane';
import { DemoPlatformAdapter } from './src/server/adapters/demo';
import type { PlatformAdapter } from './src/server/adapters/types';
import { createBff } from './src/server/bff';
import { DocsIndex } from './src/server/copilot';

dotenv.config({ path: ['.env.local', '.env'] });

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PORT = Number(process.env.PORT) || 3000;
const HOST = process.env.HOST || '127.0.0.1';

async function main() {
  let config;
  try {
    config = loadConfig();
  } catch (e) {
    if (e instanceof ConfigError) {
      console.error(`[command-centre] configuration error: ${e.message}`);
      process.exit(2);
    }
    throw e;
  }

  const adapter: PlatformAdapter =
    config.adapter === 'controlplane'
      ? new ControlPlaneAdapter({ endpoints: config.controlEndpoints, timeoutMs: config.timeoutMs, regionLocations: config.regionLocations })
      : new DemoPlatformAdapter(config.regionLocations);

  const docsRoot = process.env.DH_DOCS_ROOT || path.resolve(__dirname, '..');
  const docs = new DocsIndex(docsRoot, ['docs', 'README.md']);

  const app = express();
  app.disable('x-powered-by');
  app.use((_req, res, next) => {
    res.setHeader('X-Content-Type-Options', 'nosniff');
    res.setHeader('Referrer-Policy', 'no-referrer');
    res.setHeader('X-Frame-Options', 'DENY');
    next();
  });
  app.use('/api/v1', createBff({ adapter, config, docs }));

  if (config.production) {
    app.use(
      (_req, res, next) => {
        res.setHeader(
          'Content-Security-Policy',
          "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
        );
        next();
      },
      express.static(path.resolve(__dirname, 'dist'), { index: false })
    );
    app.get('*', (_req, res) => res.sendFile(path.resolve(__dirname, 'dist', 'index.html')));
  } else {
    const { createServer } = await import('vite');
    const vite = await createServer({ server: { middlewareMode: true }, appType: 'spa' });
    app.use(vite.middlewares);
  }

  const health = await adapter.health();
  app.listen(PORT, HOST, () => {
    console.log(`[command-centre] http://${HOST}:${PORT} · adapter=${adapter.mode} · ${adapter.source}`);
    console.log(`[command-centre] backend: ${health.reachable ? 'reachable' : 'UNREACHABLE'} (${health.detail}) · docs indexed: ${docs.size} section(s)`);
    if (adapter.mode === 'demo') console.log('[command-centre] DEMO REPLAY: every value is SIMULATED and mutations are refused.');
  });
}

main().catch((err) => {
  console.error('[command-centre] failed to start:', err);
  process.exit(1);
});
