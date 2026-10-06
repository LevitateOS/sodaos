import {object} from './sodaspaces-api.js';
import {terminalID, terminalResponse} from './sodaspaces-terminal-response.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import type {ConnectionState, TerminalObservation} from './sodaspaces-attention.js';
import type {Renderer, TerminalContext, TerminalView} from './sodaspaces-terminal.js';

export type TerminalState = 'idle' | 'opening' | 'ready' | 'closed' | 'stale';

export interface AttachmentInput {
  readBinding: () => TerminalContext | undefined;
  readSessionID: () => string | undefined;
  setSessionID: (id: string | undefined) => void;
  readState: () => TerminalState;
  setState: (state: TerminalState) => void;
  setMessage: (message: string) => void;
  setNotice: (notice: boolean) => void;
  readRetries: () => number;
  setRetries: (retries: number) => void;
  readCreateName: () => string;
  setName: (name: string) => void;
  remember: (value: string | null) => void;
  publish: (kind: TerminalObservation['kind'], state: ConnectionState) => void;
  isLive: (generation: number) => boolean;
  detach: (message: string, stale?: boolean, reason?: ConnectionState) => void;
  dispatchMetadata: (metadata: TerminalMetadata) => void;
  json: (path: string, body: Record<string, unknown> | null, signal: AbortSignal) => Promise<Record<string, unknown>>;
  readTerminal: () => TerminalView | undefined;
  readSocket: () => WebSocket | undefined;
  setSocket: (socket: WebSocket) => void;
  loadRenderer: () => Promise<Renderer>;
  awaitScreen: (n: number) => Promise<HTMLElement | undefined>;
  openTerminal: (
    n: number,
    screen: HTMLElement,
    Terminal: Renderer['Terminal'],
    FitAddon: Renderer['FitAddon']
  ) => TerminalView;
  screenReady: (n: number, terminal: TerminalView, screen: HTMLElement) => void;
  isViewVisible: () => boolean;
  isConnected: () => boolean;
  isConcealed: () => boolean;
  readDisposed: () => boolean;
  readActionBusy: () => boolean;
  readManagedEnded: () => boolean;
  beginGeneration: () => number;
  takeRequest: () => AbortController;
  clearTimer: () => void;
  setTimer: (timer: number) => void;
}

export function send(
  input: AttachmentInput,
  frame:
    | {
        type: 'resize';
        cols: number;
        rows: number;
      }
    | {
        type: 'input';
        data: string;
      }
) {
  const socket = input.readSocket();
  if (input.readState() !== 'ready' || !socket || socket.readyState !== 1 || socket.bufferedAmount > 65536) {
    input.detach('Connection lost or overloaded. Reconnect the existing terminal; input was not replayed.');
    return false;
  }
  try {
    socket.send(JSON.stringify(frame));
    return true;
  } catch {
    input.detach('Connection lost. No input was replayed.');
    return false;
  }
}

function blockedConnect(input: AttachmentInput) {
  return (
    input.readDisposed() ||
    !input.readBinding() ||
    input.readState() === 'stale' ||
    input.readState() === 'opening' ||
    input.readState() === 'ready' ||
    input.readActionBusy()
  );
}

function hiddenConnect(input: AttachmentInput, automatic: boolean) {
  return (
    !input.isViewVisible() ||
    input.isConcealed() ||
    (!automatic && (document.visibilityState === 'hidden' || !document.hasFocus()))
  );
}

function canStartConnect(input: AttachmentInput, automatic: boolean) {
  if (blockedConnect(input)) return false;
  if (input.readManagedEnded()) {
    input.setMessage('This exact session ended. Use New terminal for a different shell.');
    return false;
  }
  if (automatic && !input.readSessionID()) return false;
  return !hiddenConnect(input, automatic);
}

function detachUnready(input: AttachmentInput, state: TerminalMetadata['state']) {
  input.detach(
    'Native transition is still in progress. No replacement was made.',
    false,
    state === 'ending' ? 'ending' : 'unavailable'
  );
}

function hiddenWhileAttaching(input: AttachmentInput) {
  return !input.isViewVisible() || !input.isConnected() || input.isConcealed();
}

export function cancelIfHidden(input: AttachmentInput) {
  if (!hiddenWhileAttaching(input)) return false;
  input.detach('Attachment cancelled while hidden. No creation was sent.');
  return true;
}

function observe(input: AttachmentInput, metadata: TerminalMetadata) {
  if (metadata.id !== input.readSessionID()) throw Error('Terminal observation changed');
  input.setName(metadata.name);
  input.dispatchMetadata(metadata);
}

async function inspectExisting(input: AttachmentInput, n: number, request: AbortController) {
  if (!input.readSessionID()) return false;
  const metadata = await input.json(
    `/api/environments/${input.readBinding()!.environmentId}/terminal-sessions/${input.readSessionID()}`,
    null,
    request.signal
  );
  if (!input.isLive(n)) return true;
  const existing = terminalResponse(metadata, input.readBinding()!);
  if (existing && existing.id !== input.readSessionID()) throw Error('terminal metadata');
  if (!existing || existing.state === 'ended') {
    input.setSessionID(undefined);
    input.remember(null);
    input.detach('Native terminal is absent or ended. Use New terminal for a different shell.', false, 'ended');
    return true;
  }
  observe(input, existing);
  if (!existing.ready) {
    detachUnready(input, existing.state);
    return true;
  }
  if (existing.attached) {
    input.detach(
      'An existing writer is attached. No takeover or replacement was requested.',
      false,
      'attached-elsewhere'
    );
    return true;
  }
  return false;
}

async function reserveCreate(input: AttachmentInput, n: number, request: AbortController, cols: number, rows: number) {
  const reserved = await input.json(
    `/api/environments/${input.readBinding()!.environmentId}/terminal-sessions`,
    {cols, rows, name: input.readCreateName()},
    request.signal
  );
  if (!input.isLive(n)) return true;
  if (!terminalID(reserved.id)) throw Error('terminal reservation');
  input.setSessionID(reserved.id);
  input.setName(input.readCreateName());
  input.remember(reserved.id); // published BEFORE any native Create can be sent
  return !input.isLive(n);
}

function attachPayload(input: AttachmentInput, action: 'attach' | 'create', cols: number, rows: number) {
  const {repositoryId} = input.readBinding()!;
  return {
    action,
    id: input.readSessionID(),
    ...(action === 'create' ? {name: input.readCreateName()} : {}),
    repository_id: repositoryId,
    session_generation: input.readBinding()?.transport.generation,
    cols,
    rows,
  };
}

function onPeerOpen(
  input: AttachmentInput,
  n: number,
  peer: WebSocket,
  action: 'attach' | 'create',
  cols: number,
  rows: number
) {
  if (!input.isLive(n)) {
    peer.close();
    return;
  }
  if (cancelIfHidden(input)) return;
  input.publish('state', 'opening');
  try {
    peer.send(JSON.stringify(attachPayload(input, action, cols, rows)));
    input.setMessage(action === 'create' ? 'Opening terminal…' : 'Attaching the existing terminal…');
  } catch {
    input.detach('Attachment dispatch was not confirmed. No creation or input was retried.');
  }
}

function refreshAttachedMetadata(input: AttachmentInput, n: number, request: AbortController) {
  const target = input.readSessionID(),
    environmentId = input.readBinding()!.environmentId;
  void input
    .json(
      `/api/environments/${environmentId}/terminal-sessions/${target}`,
      null,
      AbortSignal.any([request.signal, AbortSignal.timeout(10000)])
    )
    .then((result) => {
      if (!input.isLive(n) || input.readSessionID() !== target || !input.readBinding()) return;
      const metadata = terminalResponse(result, input.readBinding()!);
      if (metadata && metadata.id === target) observe(input, metadata);
    })
    .catch(() => {});
}

function acceptReady(
  input: AttachmentInput,
  n: number,
  terminal: TerminalView,
  screen: HTMLElement,
  action: 'attach' | 'create',
  request: AbortController
) {
  input.clearTimer();
  input.setState('ready');
  input.setRetries(0);
  input.setNotice(false);
  terminal.options.disableStdin = !input.isViewVisible();
  input.setMessage(
    action === 'create'
      ? `Connected as ${input.readBinding()!.login}.`
      : `Reconnected as ${input.readBinding()!.login}.`
  );
  input.publish('state', 'ready');
  input.screenReady(n, terminal, screen);
  refreshAttachedMetadata(input, n, request);
}

function acceptOutput(
  input: AttachmentInput,
  frame: Record<string, unknown>,
  terminal: TerminalView,
  queued: {bytes: number}
) {
  if (input.readState() !== 'ready' || typeof frame.data !== 'string') throw Error('frame');
  const decoded = atob(frame.data);
  if (!decoded.length || decoded.length > 4096 || queued.bytes + decoded.length > 262144) throw Error('output');
  queued.bytes += decoded.length;
  terminal.write(
    Uint8Array.from(decoded, (c) => c.charCodeAt(0)),
    () => {
      queued.bytes -= decoded.length;
    }
  );
  input.publish('output', 'ready');
}

function dispatchFrame(
  input: AttachmentInput,
  frame: Record<string, unknown>,
  n: number,
  terminal: TerminalView,
  screen: HTMLElement,
  action: 'attach' | 'create',
  request: AbortController,
  queued: {bytes: number}
) {
  const keys = Object.keys(frame).sort().join(',');
  if (frame.type === 'ready' && keys === 'type' && input.readState() === 'opening' && input.readSessionID())
    acceptReady(input, n, terminal, screen, action, request);
  else if (frame.type === 'output' && keys === 'data,type') acceptOutput(input, frame, terminal, queued);
  else if (frame.type === 'closed' && keys === 'reason,type')
    input.detach(
      'Attachment ended or unavailable. Reconnect only the existing terminal; no replacement was launched.',
      false,
      'unavailable'
    );
  else throw Error('frame');
}

function onPeerMessage(
  input: AttachmentInput,
  event: MessageEvent,
  n: number,
  terminal: TerminalView,
  screen: HTMLElement,
  action: 'attach' | 'create',
  request: AbortController,
  queued: {bytes: number}
) {
  if (!input.isLive(n) || input.readTerminal() !== terminal) return;
  try {
    if (typeof event.data !== 'string' || event.data.length > 32768) throw Error('frame');
    dispatchFrame(input, object(JSON.parse(event.data)), n, terminal, screen, action, request, queued);
  } catch {
    input.detach('Invalid or overloaded stream. No input or creation was replayed.');
  }
}

function onPeerClosed(input: AttachmentInput, n: number) {
  if (!input.isLive(n)) return;
  input.detach('Connection lost. No End or input replay was requested.');
  const retries = input.readRetries();
  if (input.readSessionID() && retries < 3) {
    input.setRetries(retries + 1);
    input.setTimer(window.setTimeout(() => void connect(input, true), 1000 * 2 ** retries));
  }
}

function bindPeer(
  input: AttachmentInput,
  n: number,
  terminal: TerminalView,
  screen: HTMLElement,
  action: 'attach' | 'create',
  cols: number,
  rows: number,
  request: AbortController
) {
  const binding = input.readBinding()!,
    peer = binding.transport.websocket(`environments/${binding.environmentId}/terminal`),
    queued = {bytes: 0};
  input.setSocket(peer);
  peer.onopen = () => onPeerOpen(input, n, peer, action, cols, rows);
  peer.onmessage = (event) => onPeerMessage(input, event, n, terminal, screen, action, request, queued);
  peer.onerror = peer.onclose = () => onPeerClosed(input, n);
}

async function attachPeer(input: AttachmentInput, n: number, request: AbortController, automatic: boolean) {
  if (automatic && !input.readSessionID()) {
    input.detach('Terminal absent. Nothing was created.');
    return;
  }
  const action = input.readSessionID() ? 'attach' : 'create';
  const {Terminal, FitAddon} = await input.loadRenderer();
  if (!input.isLive(n)) return;
  const screen = await input.awaitScreen(n);
  if (!screen || !input.isLive(n)) return;
  const terminal = input.openTerminal(n, screen, Terminal, FitAddon);
  const cols = Math.max(2, Math.min(500, terminal.cols)),
    rows = Math.max(2, Math.min(300, terminal.rows));
  if (action === 'create' && (await reserveCreate(input, n, request, cols, rows))) return;
  bindPeer(input, n, terminal, screen, action, cols, rows, request);
}

export async function connect(input: AttachmentInput, automatic = false) {
  if (!canStartConnect(input, automatic)) return;
  input.clearTimer();
  input.setState('opening');
  const n = input.beginGeneration(),
    request = input.takeRequest();
  input.setTimer(
    window.setTimeout(
      () => input.detach('Terminal connection timed out. Inspect the exact locator; creation was not retried.'),
      45000
    )
  );
  input.setNotice(true);
  input.setMessage('Inspecting the original terminal…');
  try {
    if (await inspectExisting(input, n, request)) return;
    await attachPeer(input, n, request, automatic);
  } catch {
    if (input.isLive(n))
      input.detach(
        'Could not authorize or attach. Sign in again or inspect the existing terminal; nothing was replayed.'
      );
  }
}
