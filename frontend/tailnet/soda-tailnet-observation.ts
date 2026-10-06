import {readSodaJSON} from '../spaces/sodaspaces-api.js';
import {settingsView} from './soda-tailnet-response.js';
import type {Enrollment, Host, Settings} from './soda-tailnet-response.js';

export class TailnetRequestError extends Error {
  constructor(readonly status: number) {
    super('Tailnet request not confirmed');
  }
}

export type Scope = 'host' | 'enrollment';

function observedExitNode(host: Host): string {
  const prefs = host.preferences;
  if (prefs.exit_node_ip) return prefs.exit_node_ip;
  const match = host.peers.find((peer) => peer.id === prefs.exit_node_id);
  if (match?.addresses[0]) return match.addresses[0];
  return prefs.exit_node_id ? 'missing:' + prefs.exit_node_id : '';
}

export interface ObservationInput {
  requireLive: (lifetime: AbortController) => void;
  isLive: (lifetime: AbortController) => boolean;
  transportRequest: (path: string, init: RequestInit) => Promise<Response>;
  noteSent: () => void;
  loseAuthorization: () => void;
  readSettings: () => Settings | null;
  writeSettings: (settings: Settings) => void;
  readLifetime: () => AbortController | null;
  hasPending: () => boolean;
  isBusy: () => boolean;
  setBusy: (busy: boolean) => void;
  readBlocked: () => boolean;
  setBlocked: (blocked: boolean) => void;
  setStale: (stale: boolean) => void;
  setMessage: (message: string) => void;
  clearSecrets: () => void;
  clearAuthLink: () => void;
  readExitDirty: () => boolean;
  readAdvertiseDirty: () => boolean;
  writeExitDirty: (dirty: boolean) => void;
  writeAdvertiseDirty: (dirty: boolean) => void;
  setExitDraft: (revision: string, node: string, allowLAN: boolean) => void;
  setAdvertiseDraft: (revision: string, advertise: boolean) => void;
  setEnrollmentDraft: (revision: string, network: string, tags: string, preauthorized: boolean) => void;
  readEnrollmentDirty: () => boolean;
  writeEnrollmentDirty: (dirty: boolean) => void;
  requestUpdate: () => void;
}

export async function request(
  input: ObservationInput,
  lifetime: AbortController,
  scope?: Scope,
  body?: string
): Promise<unknown> {
  try {
    input.requireLive(lifetime);
    if (body !== undefined) input.noteSent();
    const pending = input.transportRequest('settings/tailnet' + (scope ? '/' + scope : ''), {
      method: body === undefined ? 'GET' : 'POST',
      referrerPolicy: 'no-referrer',
      ...(body === undefined ? {} : {headers: {'Content-Type': 'application/json'}}),
      ...(body === undefined ? {} : {body}),
      signal: AbortSignal.any([lifetime.signal, AbortSignal.timeout(30000)]),
    });
    body = undefined;
    const response = await pending;
    input.requireLive(lifetime);
    if (!response.ok) {
      if ([401, 403].includes(response.status)) input.loseAuthorization();
      await response.body?.cancel();
      throw new TailnetRequestError(response.status);
    }
    const result = await readSodaJSON(response);
    input.requireLive(lifetime);
    return result;
  } finally {
    body = undefined;
  }
}

export function resetHost(input: ObservationInput, discard = false) {
  const host = input.readSettings()?.host;
  if (!host) return;
  if (discard) {
    input.writeExitDirty(false);
    input.writeAdvertiseDirty(false);
  }
  if (!input.readExitDirty()) {
    input.setExitDraft(host.revision, observedExitNode(host), host.preferences.allow_lan);
  }
  if (!input.readAdvertiseDirty()) {
    input.setAdvertiseDraft(host.revision, host.preferences.advertise_exit_node);
  }
  input.requestUpdate();
}

function copyEnrollmentDraft(input: ObservationInput, policy: Enrollment) {
  input.setEnrollmentDraft(policy.revision, policy.tailnet, policy.tags.join(', '), policy.preauthorized);
}

export function resetEnrollment(input: ObservationInput) {
  const policy = input.readSettings()?.enrollment;
  if (!policy) return;
  input.clearSecrets();
  copyEnrollmentDraft(input, policy);
  input.writeEnrollmentDirty(false);
  input.requestUpdate();
}

function applyObservedSettings(input: ObservationInput, next: Settings) {
  input.writeSettings(next);
  input.setStale(false);
  input.setBlocked(false);
  resetHost(input);
  if (!input.readEnrollmentDirty()) copyEnrollmentDraft(input, next.enrollment);
  if (next.host?.state === 'Running' && !next.host.expired) input.clearAuthLink();
  input.setMessage(
    next.host_unavailable
      ? 'Appliance observation unavailable. This is not a disconnected state; enrollment policy is shown separately.'
      : 'Observations refreshed. Addresses and online status do not prove client reachability.'
  );
}

function refreshFailed(input: ObservationInput, lifetime: AbortController) {
  if (!input.isLive(lifetime)) return;
  input.clearSecrets();
  input.setStale(true);
  input.setMessage(
    input.readBlocked()
      ? 'Forgejo operator authorization is unavailable. No private controls are shown.'
      : 'Tailnet observation unavailable. Previous data and drafts are stale, not an empty network. The native helper may be disabled.'
  );
}

export async function refresh(input: ObservationInput) {
  const lifetime = input.readLifetime();
  if (!lifetime || !input.isLive(lifetime) || input.isBusy() || input.hasPending()) return;
  input.setBusy(true);
  input.setMessage('Checking authorization and Tailnet observations…');
  input.requestUpdate();
  try {
    const next = settingsView(await request(input, lifetime));
    input.requireLive(lifetime);
    applyObservedSettings(input, next);
  } catch {
    refreshFailed(input, lifetime);
  } finally {
    if (input.isLive(lifetime)) {
      input.setBusy(false);
      input.requestUpdate();
    }
  }
}
