import {object, readSodaJSON} from './sodaspaces-api.js';
import {terminalResponse} from './sodaspaces-terminal-response.js';
import type {ConnectionState, TerminalObservation} from './sodaspaces-attention.js';
import type {TerminalState} from './sodaspaces-terminal-attachment.js';
import type {TerminalContext} from './sodaspaces-terminal.js';

export interface ActionsInput {
  readBinding: () => TerminalContext | undefined;
  readDisposed: () => boolean;
  readState: () => TerminalState;
  setState: (state: TerminalState) => void;
  readGeneration: () => number;
  bumpGeneration: () => void;
  readSessionID: () => string | undefined;
  setSessionID: (id: string | undefined) => void;
  setMessage: (message: string) => void;
  setNotice: (notice: boolean) => void;
  setConfirming: (confirming: string | undefined) => void;
  readActionBusy: () => boolean;
  setActionBusy: (busy: boolean) => void;
  readActionRequest: () => AbortController | undefined;
  takeActionRequest: () => AbortController;
  clearActionRequest: () => void;
  clearTimer: () => void;
  abortRequest: () => void;
  disconnectObserver: () => void;
  readSocket: () => WebSocket | undefined;
  clearSocket: () => void;
  clearScreen: () => void;
  remember: (value: string | null) => void;
  dispatchObservation: (detail: TerminalObservation) => void;
  dispatchAuthorityLost: () => void;
}

export function live(input: ActionsInput, generation: number) {
  return !input.readDisposed() && input.readState() !== 'stale' && input.readGeneration() === generation;
}

export function publish(input: ActionsInput, kind: TerminalObservation['kind'], state: ConnectionState) {
  if (input.readDisposed()) return;
  input.dispatchObservation({
    kind,
    state,
    generation: input.readGeneration(),
    id: input.readSessionID() || null,
  });
}

function abortActionIfStale(input: ActionsInput, stale: boolean) {
  if (!stale) return;
  input.readActionRequest()?.abort();
  input.clearActionRequest();
  input.setActionBusy(false);
}

function closeSocket(input: ActionsInput) {
  const old = input.readSocket();
  input.clearSocket();
  if (old && old.readyState < 2) old.close();
}

export function detach(
  input: ActionsInput,
  message: string,
  stale = false,
  reason: ConnectionState = 'connection-lost'
) {
  input.bumpGeneration();
  input.setState(stale ? 'stale' : 'closed');
  input.setNotice(true);
  input.setConfirming(undefined);
  input.clearTimer();
  input.abortRequest();
  abortActionIfStale(input, stale);
  input.disconnectObserver();
  closeSocket(input);
  input.clearScreen();
  input.setMessage(message);
  publish(input, 'state', reason);
}

export async function json(
  input: ActionsInput,
  path: string,
  body: Record<string, unknown> | null,
  signal: AbortSignal
) {
  const binding = input.readBinding();
  if (!binding) throw Error('Missing terminal binding');
  const headers: Record<string, string> = {};
  if (body) {
    headers['Content-Type'] = 'application/json';
  }
  const response = await binding.transport.request(path.slice('/api/'.length), {
    method: body ? 'POST' : 'GET',
    headers,
    ...(body
      ? {
          body: JSON.stringify(body),
        }
      : {}),
    signal,
  });
  if (!response.ok) {
    if (response.status === 401 || response.status === 403) authorityLost(input);
    throw Error('Soda request refused');
  }
  return object(await readSodaJSON(response));
}

function authorityLost(input: ActionsInput) {
  if (!input.readDisposed()) input.dispatchAuthorityLost();
}

function actionCurrent(input: ActionsInput, control: AbortController, target: string) {
  return (
    !input.readDisposed() &&
    input.readState() !== 'stale' &&
    input.readActionRequest() === control &&
    input.readSessionID() === target
  );
}

function confirmNativeEnd(input: ActionsInput, result: Record<string, unknown>, target: string) {
  const observed = terminalResponse(result, input.readBinding()!);
  if (observed && (observed.id !== target || observed.state !== 'ended')) throw Error('outcome');
}

function endUnconfirmed(input: ActionsInput) {
  input.setNotice(true);
  input.setMessage('End was not confirmed. Inspect this exact terminal; no retry or replacement was made.');
  publish(input, 'state', 'unconfirmed');
}

function finishAction(input: ActionsInput, control: AbortController, timeout: number) {
  window.clearTimeout(timeout);
  if (input.readActionRequest() === control) {
    input.clearActionRequest();
    input.setActionBusy(false);
  }
}

function endBlocked(input: ActionsInput) {
  return (
    input.readDisposed() ||
    input.readState() === 'stale' ||
    !input.readSessionID() ||
    !input.readBinding() ||
    input.readActionBusy()
  );
}

export async function endTerminal(input: ActionsInput) {
  if (endBlocked(input)) return;
  const target = input.readSessionID()!,
    control = input.takeActionRequest();
  input.setActionBusy(true);
  const timeout = window.setTimeout(() => control.abort(), 30000);
  try {
    const result = await json(
      input,
      `/api/environments/${input.readBinding()!.environmentId}/terminal-sessions/${target}`,
      {action: 'end'},
      control.signal
    );
    if (!actionCurrent(input, control, target)) return;
    confirmNativeEnd(input, result, target);
    detach(input, 'Native cleanup confirmed. Files and independent services remain.', false, 'ended');
    input.setSessionID(undefined);
    input.remember(null);
  } catch {
    if (actionCurrent(input, control, target)) endUnconfirmed(input);
  } finally {
    finishAction(input, control, timeout);
  }
}
