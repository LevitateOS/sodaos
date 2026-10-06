import {hostResult, enrollmentResult} from './soda-tailnet-response.js';
import type {Host, Settings} from './soda-tailnet-response.js';
import {TailnetRequestError} from './soda-tailnet-observation.js';
import type {Scope} from './soda-tailnet-observation.js';

export const unknownOutcome =
  'Operation unconfirmed. It may have completed. Observe native state before explicitly retrying; nothing was replayed or rolled back.';

export type Confirmation = {scope: Scope; body: Record<string, unknown>; label: string; warning: string};

type HostResult = ReturnType<typeof hostResult>;

function hostOutcomeNotice(outcome: string) {
  if (outcome === 'unconfirmed') return unknownOutcome;
  if (outcome === 'pending')
    return 'Authentication pending. Complete the provider step; closing this page does not cancel native authentication.';
  if (outcome === 'observed') return 'Authentication observed; no login or connection change was requested.';
  return 'Native operation acknowledged. This is not approval or routed-traffic proof.';
}

function mutationFailureNotice(error: unknown, sent: boolean) {
  if (error instanceof TailnetRequestError && [400, 409, 422].includes(error.status)) {
    return 'Request rejected before the requested management effect. Review inputs, revision and runtime support, then refresh. No automatic retry occurred.';
  }
  return sent ? unknownOutcome : 'Operation was not sent. Check Forgejo authorization and refresh before retrying.';
}

function hostWarning(action: string) {
  if (action === 'advertise-exit-node')
    return 'Change only exit-node advertisement, preserving unrelated routes. Tailscale owns approval and routing policy; advertisement alone does not prove usable routed traffic.';
  if (action === 'refresh-forgejo')
    return 'Refresh the appliance Git SSH advertisement through the native helper. This may restart Forgejo and interrupt its requests. It does not change browser/OAuth origins or confirm Tailnet reachability.';
  return 'No alternative management path has been verified here. This may interrupt your current management connection and SSH sessions. Use your existing approved private route or console for recovery.';
}

function formText(data: FormData, key: string) {
  const value = data.get(key);
  return typeof value === 'string' ? value : '';
}

export interface ActionsInput {
  requireLive: (lifetime: AbortController) => void;
  isLive: (lifetime: AbortController) => boolean;
  readLifetime: () => AbortController | null;
  sendRequest: (lifetime: AbortController, scope: Scope, body: string) => Promise<unknown>;
  readSettings: () => Settings | null;
  writeSettings: (settings: Settings) => void;
  isBusy: () => boolean;
  setBusy: (busy: boolean) => void;
  isStale: () => boolean;
  setStale: (stale: boolean) => void;
  isBlocked: () => boolean;
  setNotice: (notice: string) => void;
  appendNotice: (suffix: string) => void;
  readSent: () => boolean;
  writeSent: (sent: boolean) => void;
  readPending: () => Confirmation | null;
  writePending: (pending: Confirmation | null) => void;
  readTrigger: () => HTMLButtonElement | null;
  writeTrigger: (trigger: HTMLButtonElement | null) => void;
  writeExitDirty: (dirty: boolean) => void;
  writeAdvertiseDirty: (dirty: boolean) => void;
  readEnrollmentDirty: () => boolean;
  readHostDrafts: () => {
    exitRevision: string;
    exitNode: string;
    allowLAN: boolean;
    advertiseRevision: string;
    advertise: boolean;
  };
  readEnrollmentDrafts: () => {policyRevision: string; network: string; tags: string; preauthorized: boolean};
  syncHostDraft: (discard: boolean) => void;
  syncEnrollmentDraft: () => void;
  clearSecrets: () => void;
  setAuthURL: (url: string) => void;
  updated: () => Promise<boolean>;
  focusConfirm: () => void;
  requestUpdate: () => void;
}

function clearHostDraft(input: ActionsInput, action: string) {
  if (action === 'exit-node') input.writeExitDirty(false);
  if (action === 'advertise-exit-node') input.writeAdvertiseDirty(false);
}

function applyHostMutation(input: ActionsInput, action: string, result: HostResult) {
  const settings = input.readSettings();
  if (!settings) throw Error('Settings retired');
  input.writeSettings({...settings, host: result.host, host_unavailable: result.readback_unavailable});
  input.setAuthURL(result.authURL);
  input.setNotice(hostOutcomeNotice(result.outcome));
  if (result.readback_unavailable)
    input.appendNotice(
      ' Host readback failed independently. Refresh observations; do not replay the operation to repair this observer.'
    );
  if (result.outcome === 'confirmed' && !result.readback_unavailable) clearHostDraft(input, action);
  input.syncHostDraft(false);
}

function applyEnrollmentMutation(input: ActionsInput, action: string, revision: string, raw: unknown) {
  const settings = input.readSettings();
  if (!settings) throw Error('Settings retired');
  const result = enrollmentResult(raw, action, revision);
  input.writeSettings({...settings, enrollment: result.enrollment});
  input.setNotice(
    result.saved
      ? 'Policy saved. Existing devices were not disconnected, revoked or retargeted. Project enrollment is not verified.'
      : 'Credential check passed; nothing was saved and no auth key or device was created. Network, scope and enrollment remain unverified.'
  );
  // Admission/default writes do not submit the credential-binding draft.
  // Keep its original CAS revision until explicit discard or save/rotation.
  if (result.saved && (action === 'save' || action === 'rotate' || !input.readEnrollmentDirty()))
    input.syncEnrollmentDraft();
}

function applyMutation(input: ActionsInput, scope: Scope, action: string, revision: string, raw: unknown) {
  if (!input.readSettings()) throw Error('Settings retired');
  if (scope === 'host') applyHostMutation(input, action, hostResult(raw, action));
  else applyEnrollmentMutation(input, action, revision, raw);
}

function mutationFailed(input: ActionsInput, lifetime: AbortController, error: unknown) {
  if (!input.isLive(lifetime)) return;
  input.clearSecrets();
  input.setStale(true);
  input.setNotice(mutationFailureNotice(error, input.readSent()));
}

async function mutate(input: ActionsInput, scope: Scope, action: string, body: string, revision: string) {
  const lifetime = input.readLifetime();
  if (!lifetime || !input.isLive(lifetime) || input.isBusy() || input.isStale() || input.isBlocked()) return;
  input.setBusy(true);
  input.writeSent(false);
  input.writePending(null);
  input.writeTrigger(null);
  input.clearSecrets();
  input.setNotice('Checking authorization before dispatch…');
  input.requestUpdate();
  try {
    const pending = input.sendRequest(lifetime, scope, body);
    body = '';
    const raw = await pending;
    input.requireLive(lifetime);
    applyMutation(input, scope, action, revision, raw);
  } catch (error) {
    mutationFailed(input, lifetime, error);
  } finally {
    body = '';
    if (input.isLive(lifetime)) {
      input.writeSent(false);
      input.setBusy(false);
      input.requestUpdate();
    }
  }
}

export async function choose(input: ActionsInput, event: Event, confirmation: Confirmation) {
  if (input.isBusy() || input.isStale() || input.isBlocked() || input.readPending()) return;
  input.clearSecrets();
  input.writePending(confirmation);
  input.writeTrigger(event.currentTarget instanceof HTMLButtonElement ? event.currentTarget : null);
  input.requestUpdate();
  await input.updated();
  if (input.readPending() === confirmation) input.focusConfirm();
}

export async function cancel(input: ActionsInput) {
  const trigger = input.readTrigger();
  input.writePending(null);
  input.writeTrigger(null);
  input.requestUpdate();
  await input.updated();
  if (trigger?.isConnected) trigger.focus();
}

export function confirm(input: ActionsInput) {
  const pending = input.readPending();
  if (pending)
    void mutate(
      input,
      pending.scope,
      String(pending.body.action),
      JSON.stringify(pending.body),
      String(pending.body.revision)
    );
}

function hostActionBody(input: ActionsInput, action: string, host: Host): Record<string, unknown> {
  const body: Record<string, unknown> = {action, revision: host.revision, confirm: action};
  if (action === 'exit-node') {
    const drafts = input.readHostDrafts();
    Object.assign(body, {
      revision: drafts.exitRevision,
      exit_node: drafts.exitNode,
      allow_lan: !!drafts.exitNode && drafts.allowLAN,
    });
  }
  if (action === 'advertise-exit-node') {
    const drafts = input.readHostDrafts();
    Object.assign(body, {revision: drafts.advertiseRevision, advertise: drafts.advertise});
  }
  return body;
}

export function hostAction(input: ActionsInput, event: Event, action: string) {
  const host = input.readSettings()?.host;
  if (!host) return;
  if (action === 'exit-node' && input.readHostDrafts().exitNode.startsWith('missing:')) {
    input.setNotice('Select an available exit node or explicitly choose None before applying.');
    input.requestUpdate();
    return;
  }
  if (action === 'signin' || action === 'authentication') {
    void mutate(input, 'host', action, JSON.stringify({action, revision: host.revision}), host.revision);
    return;
  }
  void choose(input, event, {
    scope: 'host',
    body: hostActionBody(input, action, host),
    label: action,
    warning: hostWarning(action),
  });
}

function enrollmentSubmitBlocked(input: ActionsInput, action: string, form: HTMLFormElement) {
  if (input.isBusy() || input.isStale() || input.isBlocked() || !['check', 'save', 'rotate'].includes(action))
    return true;
  return !form.reportValidity();
}

function enrollmentNeedsReview(action: string, data: FormData) {
  return action !== 'check' && data.get('reviewed') !== 'on';
}

function enrollmentPayload(input: ActionsInput, action: string, data: FormData) {
  const drafts = input.readEnrollmentDrafts();
  const tags = drafts.tags
    .split(',')
    .map((tag) => tag.trim())
    .sort();
  return JSON.stringify({
    action,
    revision: drafts.policyRevision,
    tailnet: drafts.network,
    tags,
    preauthorized: drafts.preauthorized,
    client_id: formText(data, 'client_id'),
    client_secret: formText(data, 'client_secret'),
  });
}

export function submitEnrollment(input: ActionsInput, event: SubmitEvent) {
  event.preventDefault();
  if (!(event.currentTarget instanceof HTMLFormElement) || !(event.submitter instanceof HTMLButtonElement)) return;
  const form = event.currentTarget,
    action = event.submitter.value;
  if (enrollmentSubmitBlocked(input, action, form)) {
    input.clearSecrets();
    return;
  }
  const data = new FormData(form);
  if (enrollmentNeedsReview(action, data)) {
    input.clearSecrets();
    data.delete('client_secret');
    input.setNotice('Review and confirm the exposure/binding change before saving.');
    input.requestUpdate();
    return;
  }
  let body = enrollmentPayload(input, action, data);
  data.delete('client_secret');
  input.clearSecrets();
  void mutate(input, 'enrollment', action, body, input.readEnrollmentDrafts().policyRevision);
  body = '';
}
