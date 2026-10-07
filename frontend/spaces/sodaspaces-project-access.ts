import {keyPreviewResponse, profileKeysResponse} from './sodaspaces-keys-response.js';
import type {KeyPreview, ProfileKeys, SavedKey} from './sodaspaces-keys-response.js';
import type {Detail, Environment} from './sodaspaces-project-response.js';
import {mutate} from './sodaspaces-project-mutations.js';
import type {MutationsInput} from './sodaspaces-project-mutations.js';

export interface AccessInput {
  readEnvironment: () => Environment | undefined;
  readDetail: () => Detail | undefined;
  isRunning: () => boolean;
  readPresentation: () => 'standard' | 'journey' | 'settings';
  readUseSavedKeys: () => boolean;
  setUseSavedKeys: (checked: boolean) => void;
  readDraft: () => string;
  setDraft: (value: string) => void;
  readEmptyConfirmed: () => boolean;
  setEmptyConfirmed: (confirmed: boolean) => void;
  readKeyPreview: () => KeyPreview | undefined;
  setKeyPreview: (preview: KeyPreview | undefined) => void;
  setProfileKeys: (keys: ProfileKeys | undefined) => void;
  isBlocked: () => boolean;
  isAccessSelected: () => boolean;
  isConcealed: () => boolean;
  readEpoch: () => number;
  isActive: (generation: number) => boolean;
  setBusy: (busy: boolean) => void;
  setOutcome: (outcome: string) => void;
  runCommand: (event: Event, action: () => void | Promise<void>) => void;
  api: (
    path: string,
    method: string,
    body: Record<string, unknown> | undefined,
    signal: AbortSignal
  ) => Promise<unknown>;
  mutations: MutationsInput;
}

export function setUseSavedKeys(input: AccessInput, checked: boolean) {
  input.setUseSavedKeys(checked);
}

export function setDraft(input: AccessInput, value: string) {
  input.setDraft(value);
}

export function setConfirmEmpty(input: AccessInput, checked: boolean) {
  input.setEmptyConfirmed(checked);
}

export function joinEnvironment(input: AccessInput) {
  const environment = input.readEnvironment(),
    detail = input.readDetail();
  if (!environment || !input.isRunning() || !detail?.execution_allowed || detail.login || detail.authority_unavailable)
    return;
  return mutate(
    input.mutations,
    '/api/environments/' + environment.id + '/join',
    {
      ssh_keys: input.readPresentation() !== 'journey' && input.readUseSavedKeys() ? 'saved' : 'none',
    },
    'Native join confirmed. Your browser terminal uses this account, not SSH. Later SSH-key changes require a separate explicit Apply.'
  );
}

export function removeSavedKey(input: AccessInput, event: Event, key: SavedKey) {
  input.runCommand(event, () => {
    if (
      !window.confirm(
        'Remove this saved public key? This may remove your final saved development key. Previously installed Project SSH keys remain unchanged.'
      )
    )
      return;
    return mutate(
      input.mutations,
      '/api/me/development-keys/' + key.id,
      {confirm_last: true},
      'Saved key removed. Existing project SSH access is unchanged until explicitly applied.',
      'DELETE'
    );
  });
}

export function selectForgejoKey(input: AccessInput, key: string) {
  if (input.isBlocked() || !input.isAccessSelected() || input.isConcealed()) return;
  input.setDraft(key);
  input.setOutcome(
    'Review the selected public key above, then explicitly Save public key. Joining/applying to a project remains a separate action.'
  );
}

export function saveKey(input: AccessInput) {
  const value = input.readDraft().trim();
  if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) {
    input.setOutcome('Provide one public SSH key. Never upload a private key.');
    return;
  }
  input.setDraft('');
  return mutate(
    input.mutations,
    '/api/me/development-keys',
    {
      public_key: value,
    },
    'Public key saved for future joins. Existing project access is unchanged until explicitly applied.'
  );
}

function profileKeysPageAdmitted(input: AccessInput, page: number) {
  return (
    !input.isBlocked() &&
    input.isAccessSelected() &&
    !input.isConcealed() &&
    Number.isInteger(page) &&
    page >= 1 &&
    page <= 8
  );
}

export async function reviewProfileKeys(input: AccessInput, page: number) {
  if (!profileKeysPageAdmitted(input, page)) return;
  const epoch = input.readEpoch();
  input.setBusy(true);
  const controller = new AbortController(),
    timeout = window.setTimeout(() => controller.abort(), 20000);
  try {
    const keys = profileKeysResponse(
      await input.api('/api/me/forgejo-keys?page=' + page, 'GET', undefined, controller.signal),
      page
    );
    if (input.isActive(epoch)) input.setProfileKeys(keys);
  } catch {
    profileKeysFailed(input, epoch);
  } finally {
    window.clearTimeout(timeout);
    if (input.isActive(epoch)) input.setBusy(false);
  }
}

function profileKeysFailed(input: AccessInput, epoch: number) {
  if (!input.isActive(epoch)) return;
  input.setProfileKeys(undefined);
  input.setOutcome(
    'Own Forgejo keys are unavailable. Nothing was imported; use native profile settings or explicitly paste a public key.'
  );
}

export async function reviewKeys(input: AccessInput) {
  const environment = input.readEnvironment(),
    detail = input.readDetail();
  if (input.isBlocked() || !environment || !detail) return;
  const n = input.readEpoch();
  input.setBusy(true);
  const controller = new AbortController(),
    timeout = window.setTimeout(() => controller.abort(), 15000);
  try {
    const preview = keyPreviewResponse(
      await input.api(`/api/environments/${environment.id}/access-keys`, 'GET', undefined, controller.signal),
      detail.login
    );
    if (!input.isActive(n)) return;
    input.setKeyPreview(preview);
    input.setEmptyConfirmed(false);
  } catch {
    if (input.isActive(n)) {
      input.setKeyPreview(undefined);
      input.setOutcome('Key preview unavailable or changed. No update was requested; refresh and inspect.');
    }
  } finally {
    window.clearTimeout(timeout);
    if (input.isActive(n)) input.setBusy(false);
  }
}

export function applyKeys(input: AccessInput) {
  const preview = input.readKeyPreview(),
    environment = input.readEnvironment();
  if (!preview || !environment || !input.readDetail()?.execution_allowed) return;
  if (!preview.saved_fingerprints.length && !input.readEmptyConfirmed()) {
    input.setOutcome('Explicitly confirm removal of the last managed key.');
    return;
  }
  input.setKeyPreview(undefined);
  input.setEmptyConfirmed(false);
  return mutate(
    input.mutations,
    `/api/environments/${environment.id}/access-keys`,
    {
      revision: preview.revision,
      saved_fingerprints: preview.saved_fingerprints,
      confirm_empty: !preview.saved_fingerprints.length,
    },
    'Managed SSH key file updated for this project. Verify new-key login and old-key refusal from your SSH client. Existing sessions and browser access are not revoked.'
  );
}
