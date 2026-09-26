import express from 'express';
import type { AddressInfo } from 'node:net';
import { createBff } from '../src/server/bff';
import type { PlatformAdapter } from '../src/server/adapters/types';
import type { BffConfig } from '../src/server/config';

export async function startBff(adapter: PlatformAdapter, overrides: Partial<BffConfig> = {}) {
  const config: BffConfig = {
    adapter: adapter.mode,
    controlEndpoints: [],
    timeoutMs: 3000,
    secureCookies: false,
    copilotModel: null,
    production: false,
    ...overrides
  };
  const app = express();
  app.use('/api/v1', createBff({ adapter, config, docs: null }));
  const server = app.listen(0, '127.0.0.1');
  await new Promise((r) => server.once('listening', r));
  const base = `http://127.0.0.1:${(server.address() as AddressInfo).port}/api/v1`;
  const call = async (method: string, path: string, opts: { token?: string; body?: unknown; csrf?: boolean } = {}) => {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (opts.token) headers.Authorization = `Bearer ${opts.token}`;
    if (opts.body !== undefined) headers['Content-Type'] = 'application/json';
    if (method !== 'GET' && opts.csrf !== false) headers['X-DH-Console'] = '1';
    const res = await fetch(base + path, { method, headers, body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined });
    const text = await res.text();
    let json: any = null;
    try {
      json = JSON.parse(text);
    } catch {
      json = null;
    }
    return { status: res.status, json, text, headers: res.headers };
  };
  return { base, call, close: () => new Promise<void>((r) => server.close(() => r())) };
}

export async function waitFor<T>(fn: () => Promise<T | null | undefined | false>, timeoutMs: number, stepMs = 1000): Promise<T> {
  const end = Date.now() + timeoutMs;
  let last: unknown;
  while (Date.now() < end) {
    try {
      const v = await fn();
      if (v) return v;
    } catch (e) {
      last = e;
    }
    await new Promise((r) => setTimeout(r, stepMs));
  }
  throw new Error(`condition not met within ${timeoutMs}ms${last ? `: ${String(last)}` : ''}`);
}
