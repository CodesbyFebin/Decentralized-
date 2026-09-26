/**
 * BFF configuration. Adapter selection is explicit: production refuses the
 * demo replay unless ALLOW_DEMO_IN_PRODUCTION=1, and there is no automatic
 * fallback between adapters.
 */
import type { AdapterMode } from '../types/reality';

export interface BffConfig {
  adapter: AdapterMode;
  controlEndpoints: string[];
  timeoutMs: number;
  regionLocations?: Record<string, [number, number]>;
  secureCookies: boolean;
  copilotModel: 'gemini' | null;
  production: boolean;
  /** Directory of sealed validation records (the repository's evidence/). */
  evidenceDir: string | null;
  /** dh CLI used to verify records (`dh evidence verify`). */
  cli: string | null;
}

export class ConfigError extends Error {}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): BffConfig {
  const production = env.NODE_ENV === 'production';
  const raw = (env.PLATFORM_ADAPTER ?? '').trim().toLowerCase();
  if (!raw) {
    throw new ConfigError('PLATFORM_ADAPTER is not set. Use PLATFORM_ADAPTER=controlplane with DH_CONTROL_URL, or PLATFORM_ADAPTER=demo for a labelled replay.');
  }
  if (raw !== 'controlplane' && raw !== 'demo') throw new ConfigError(`PLATFORM_ADAPTER must be "controlplane" or "demo", got "${raw}".`);
  if (raw === 'demo' && production && env.ALLOW_DEMO_IN_PRODUCTION !== '1') {
    throw new ConfigError('Refusing to start: PLATFORM_ADAPTER=demo in production. Set ALLOW_DEMO_IN_PRODUCTION=1 only for a labelled showcase.');
  }

  const endpoints = (env.DH_CONTROL_URL ?? '')
    .split(',')
    .map((s) => s.trim().replace(/\/+$/, ''))
    .filter(Boolean);
  if (raw === 'controlplane') {
    if (!endpoints.length) throw new ConfigError('PLATFORM_ADAPTER=controlplane needs DH_CONTROL_URL (comma-separated member API URLs).');
    for (const e of endpoints) if (!/^https?:\/\//.test(e)) throw new ConfigError(`DH_CONTROL_URL entry "${e}" must start with http:// or https://`);
  }

  let regionLocations: Record<string, [number, number]> | undefined;
  if (env.DH_REGION_LOCATIONS) {
    let parsed: unknown;
    try {
      parsed = JSON.parse(env.DH_REGION_LOCATIONS);
    } catch {
      throw new ConfigError('DH_REGION_LOCATIONS must be JSON: {"region":[lat,lng]}');
    }
    regionLocations = {};
    for (const [k, v] of Object.entries(parsed as Record<string, unknown>)) {
      if (!Array.isArray(v) || v.length !== 2 || !v.every((n) => typeof n === 'number' && Number.isFinite(n))) {
        throw new ConfigError(`DH_REGION_LOCATIONS["${k}"] must be [lat, lng]`);
      }
      regionLocations[k] = [v[0], v[1]];
    }
  }

  const timeoutMs = Number(env.DH_CONTROL_TIMEOUT_MS ?? 5000);
  return {
    adapter: raw,
    controlEndpoints: endpoints,
    timeoutMs: Number.isFinite(timeoutMs) && timeoutMs > 0 ? timeoutMs : 5000,
    regionLocations,
    secureCookies: env.COOKIE_SECURE ? env.COOKIE_SECURE === '1' : production,
    copilotModel: env.COPILOT_MODEL === 'gemini' && !!env.GEMINI_API_KEY ? 'gemini' : null,
    production,
    evidenceDir: env.DH_EVIDENCE_DIR?.trim() || null,
    cli: env.DH_CLI?.trim() || null
  };
}
