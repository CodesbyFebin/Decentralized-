/**
 * Capability discovery. States are derived from what the connected backend
 * actually reported, never declared statically.
 */
import type { AdapterMode, Capabilities, CapabilityEntry, CapabilityKey, RealitySnapshot, TruthState } from '../../types/reality';

const UNAVAILABLE = (detail: string): CapabilityEntry => ({ state: 'UNAVAILABLE', source: 'none', detail });

export function deriveCapabilities(input: {
  mode: AdapterMode;
  now: number;
  snapshot: RealitySnapshot | null;
  reachable: boolean;
  detail: string;
  hasRegionLocations: boolean;
  copilotModel: boolean;
  /** Description of the configured validation-record store, or null when none is configured. */
  validationRecords?: string | null;
}): Capabilities {
  const { mode, snapshot: s, reachable, now } = input;
  const live: TruthState = mode === 'demo' ? 'SIMULATED' : 'LIVE';
  const src = mode === 'demo' ? 'demo-replay' : 'control-plane';
  // Without a backend nothing operational can be claimed.
  const down = (detail: string): CapabilityEntry => ({ state: 'UNKNOWN', source: src, detail });
  const on = (detail: string, state: TruthState = live): CapabilityEntry => ({ state, source: src, detail });
  const ok = reachable && s !== null;

  const edgeHosts = s ? s.nodes.filter((n) => n.edgeObs).length : 0;

  const items: Record<CapabilityKey, CapabilityEntry> = {
    nodes: ok ? on(`${s!.nodes.length} host(s) from signed observations`) : down(input.detail),
    nodeOperations: ok ? on('approve, drain, undrain and revoke go through the control plane', mode === 'demo' ? 'SIMULATED' : 'LIVE') : down(input.detail),
    nodeLogs: ok ? on('host logs relayed by the control plane') : down(input.detail),
    applications: ok ? on(`${s!.apps.length} application(s) with desired/admitted/observed replicas`) : down(input.detail),
    deployments: ok ? on('manifests applied through POST /api/v1/apply; progress observed from replica rows') : down(input.detail),
    storage: ok ? on(`${s!.volumes.length} replicated volume(s), host storage observations`) : down(input.detail),
    domains: ok ? on('desired ingress from manifests; routing observed at edge hosts; DNS is not observed') : down(input.detail),
    certificates: ok ? (edgeHosts ? on(`X.509 state observed by ${edgeHosts} edge host(s)`) : UNAVAILABLE('no edge host is reporting')) : down(input.detail),
    edgeTraffic: ok ? (edgeHosts ? on('request / error counters and per-endpoint probe latency from edge hosts') : UNAVAILABLE('no edge host is reporting')) : down(input.detail),
    visitorAnalytics: UNAVAILABLE('no visitor analytics subsystem exists'),
    waf: UNAVAILABLE('no web application firewall exists'),
    ddosTelemetry: UNAVAILABLE('no DDoS telemetry feed exists'),
    evidence: ok ? on('hash-chained audit ledger, signed host evidence, artifact attestations, milestone derivations') : down(input.detail),
    audit: ok ? on(`audit ledger head ${s!.evidence.head}, verification ${s!.evidence.verification.state}`) : down(input.detail),
    federation: ok ? on('root-signed agreements with peer clusters') : down(input.detail),
    depin: UNAVAILABLE('no DePIN network adapter is implemented'),
    billing: UNAVAILABLE('self-hosted deployment; external billing is not configured'),
    teamDirectory: UNAVAILABLE('operators are identified by signed capabilities, not a user directory'),
    copilot: ok
      ? on(input.copilotModel ? 'answers grounded in the view you are authorized to read (model-assisted)' : 'answers grounded in the view you are authorized to read (no model configured; deterministic retrieval)', 'DERIVED')
      : down('cannot ground answers without the control plane'),
    validationRecords: input.validationRecords
      ? { state: 'LIVE', source: 'DH_EVIDENCE_DIR', detail: input.validationRecords }
      : UNAVAILABLE('no evidence directory configured (DH_EVIDENCE_DIR)'),
    geolocation: input.hasRegionLocations
      ? { state: 'CONFIGURED', source: 'DH_REGION_LOCATIONS', detail: 'map positions configured per region by the operator' }
      : UNAVAILABLE('hosts do not report geographic coordinates; set DH_REGION_LOCATIONS to place regions on the map')
  };

  return {
    mode,
    backend: {
      reachable,
      state: reachable ? live : 'UNAVAILABLE',
      detail: input.detail,
      checkedAt: now,
      servedBy: s?.overview.cluster.servedBy.member,
      leader: s?.overview.cluster.servedBy.leader
    },
    items
  };
}
