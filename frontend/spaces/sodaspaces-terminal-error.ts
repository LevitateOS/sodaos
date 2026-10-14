import {object, SodaRequestError} from './sodaspaces-api.js';

const supportMessages: Readonly<Record<string, string>> = {
  terminal_support_mismatch: 'Terminal support needs an update. An appliance administrator must update this Project.',
  terminal_support_unavailable:
    'Terminal support could not be inspected. Retry or ask an appliance administrator to inspect this Project.',
  project_maintenance: 'Project maintenance is in progress. Managed access is temporarily unavailable.',
};

export function terminalFailureMessage(error: unknown): string {
  if (!(error instanceof SodaRequestError))
    return 'The terminal outcome could not be confirmed. Inspect the existing locator; no creation or input was replayed.';
  const support = error.code ? supportMessages[error.code] : undefined;
  if (support) return support;
  if (error.code === 'session_changed' || error.code === 'extension_session_changed')
    return 'The native page session changed. Refresh this page before reconnecting the existing terminal.';
  if (error.status === 401)
    return 'Your native login expired. Sign in through Fountain, then reconnect the existing terminal.';
  if (error.status === 403)
    return 'You do not have the required Project membership or repository write permission. Ask a Project administrator for access.';
  return 'The terminal is unavailable. Retry or ask an appliance administrator to inspect it. No creation or input was replayed.';
}

export type SupportState = 'ready' | 'support_mismatch' | 'inspection_unavailable' | 'maintenance_in_progress';
export function supportState(value: unknown): SupportState {
  if (value === 'ready' || value === 'support_mismatch' || value === 'maintenance_in_progress') return value;
  return 'inspection_unavailable';
}
export function supportMessage(state: SupportState): string {
  if (state === 'ready') return 'Terminal support is ready.';
  const code = {
    support_mismatch: 'terminal_support_mismatch',
    inspection_unavailable: 'terminal_support_unavailable',
    maintenance_in_progress: 'project_maintenance',
  }[state];
  return supportMessages[code] || '';
}

export function terminalClosedMessage(reason: unknown): string {
  return terminalFailureMessage(new SodaRequestError(503, typeof reason === 'string' ? reason : undefined));
}
export function supportObservation(value: unknown): SupportState {
  return supportState(value === undefined ? undefined : object(value).state);
}
