import {object} from './sodaspaces-api.js';
import {factoryOutputCursor, factoryOutputFrame, factoryClosedReason} from './sodaspaces-factory-stream-response.js';
import {factoryRunStatusResponse, factoryStatusFrame} from './sodaspaces-factory-response.js';
import {statusText} from './sodaspaces-factory-display.js';
import type {FactoryWatchState} from './sodaspaces-factory-display.js';
import type {FactoryWatchContext} from './sodaspaces-factory.js';
import type {Renderer, TerminalView} from './sodaspaces-terminal.js';

export interface RequestInput {
  readBinding: () => FactoryWatchContext | undefined;
  readDisposed: () => boolean;
  readState: () => FactoryWatchState;
  setState: (state: FactoryWatchState) => void;
  beginGeneration: () => number;
  isLive: (generation: number) => boolean;
  detach: (message: string, stale?: boolean) => void;
  setMessage: (message: string) => void;
  setNotice: (notice: boolean) => void;
  setStatusLine: (line: string) => void;
  readRetries: () => number;
  setRetries: (retries: number) => void;
  readCursor: () => number;
  setCursor: (cursor: number) => void;
  clearTimer: () => void;
  setTimer: (timer: number) => void;
  takeRequest: () => AbortController;
  json: (path: string, signal: AbortSignal) => Promise<Record<string, unknown>>;
  readTitle: () => string;
  loadRenderer: () => Promise<Renderer>;
  awaitScreen: (n: number) => Promise<HTMLElement | undefined>;
  openScreen: (screen: HTMLElement, Terminal: Renderer['Terminal'], FitAddon: Renderer['FitAddon']) => TerminalView;
  readTerminal: () => TerminalView | undefined;
  setSocket: (socket: WebSocket) => void;
  screenReady: () => void;
  clearScreen: () => void;
}

function blockedWatch(input: RequestInput) {
  return (
    input.readDisposed() ||
    !input.readBinding() ||
    input.readState() === 'stale' ||
    input.readState() === 'opening' ||
    input.readState() === 'ready'
  );
}

function endedRun(
  input: RequestInput,
  detail: {state?: {phase: string; exit_code?: number; output?: string; output_truncated: boolean}}
) {
  const state = detail.state;
  const statusLine = state ? statusText(state.phase, false, true, state.exit_code ?? null) : 'ended';
  input.setStatusLine(statusLine);
  const excerpt = state?.output ? `\nLast output:\n${state.output}` : '';
  const cut = state?.output_truncated ? '\n… output truncated …' : '';
  input.detach(`Run ended (${statusLine}). Reattachment is unavailable; this view keeps its status.${excerpt}${cut}`);
}

async function inspectRun(input: RequestInput, n: number, request: AbortController) {
  const binding = input.readBinding()!;
  const detail = factoryRunStatusResponse(
    await input.json(`factory/runs/${binding.runId}`, request.signal),
    binding.runId,
    binding.repositoryId
  );
  if (!input.isLive(n)) return true;
  if (detail.state?.terminal) {
    endedRun(input, detail);
    return true;
  }
  if (detail.state) input.setStatusLine(statusText(detail.state.phase, detail.state.live, false, null));
  else input.setStatusLine(detail.outcome ? `recorded · ${detail.outcome}` : 'recorded');
  return false;
}

function attachPayload(input: RequestInput, cursor: number) {
  const binding = input.readBinding()!;
  return {
    run_id: binding.runId,
    repository_id: binding.repositoryId,
    session_generation: binding.transport.generation,
    cursor,
  };
}

function onPeerOpen(input: RequestInput, n: number, peer: WebSocket, cursor: number) {
  if (!input.isLive(n)) {
    peer.close();
    return;
  }
  try {
    peer.send(JSON.stringify(attachPayload(input, cursor)));
    input.setMessage('Attaching the recorded run…');
  } catch {
    input.detach('Attachment dispatch was not confirmed. No input was sent.');
  }
}

function acceptStatus(input: RequestInput, frame: ReturnType<typeof factoryStatusFrame>) {
  input.setStatusLine(statusText(frame.phase, frame.live, frame.terminal, frame.exit_code));
  if (input.readState() === 'opening') {
    input.clearTimer();
    input.setState('ready');
    input.setRetries(0);
    input.setNotice(false);
    input.setMessage(`Watching ${input.readTitle()}.`);
    input.screenReady();
  }
}

function acceptOutput(
  input: RequestInput,
  frame: ReturnType<typeof factoryOutputFrame>,
  terminal: TerminalView,
  queued: {bytes: number}
) {
  if (input.readState() !== 'ready') throw Error('frame');
  const next = factoryOutputCursor(input.readCursor(), frame);
  if (frame.truncated) terminal.write('\r\n… earlier output truncated …\r\n');
  if (frame.gap) terminal.write(`\r\n… output gap: skipped to byte ${frame.next} …\r\n`);
  if (queued.bytes + frame.bytes.length > 262144) throw Error('output');
  queued.bytes += frame.bytes.length;
  input.setCursor(next);
  if (frame.bytes.length)
    terminal.write(frame.bytes, () => {
      queued.bytes -= frame.bytes.length;
    });
}

function dispatchFrame(
  input: RequestInput,
  frame: Record<string, unknown>,
  terminal: TerminalView,
  queued: {bytes: number}
) {
  if (frame.type === 'status' && input.readBinding())
    acceptStatus(input, factoryStatusFrame(frame, input.readBinding()!.runId));
  else if (frame.type === 'output') acceptOutput(input, factoryOutputFrame(frame), terminal, queued);
  else if (frame.type === 'closed') detachClosed(input, factoryClosedReason(frame));
  else throw Error('frame');
}

function detachClosed(input: RequestInput, reason: string) {
  const ended = reason === 'eof' ? 'Run ended. This view keeps its rendered output.' : `View ended: ${reason}.`;
  input.detach(ended);
}

function onPeerMessage(
  input: RequestInput,
  event: MessageEvent,
  n: number,
  terminal: TerminalView,
  queued: {bytes: number}
) {
  if (!input.isLive(n) || input.readTerminal() !== terminal) return;
  try {
    if (typeof event.data !== 'string' || event.data.length > 65536) throw Error('frame');
    dispatchFrame(input, object(JSON.parse(event.data)), terminal, queued);
  } catch {
    input.detach('Invalid or overloaded stream. No input was sent.');
  }
}

function onPeerClosed(input: RequestInput, n: number) {
  if (!input.isLive(n)) return;
  input.detach('Connection lost. The run continues without this view.');
  const retries = input.readRetries();
  if (retries < 3) {
    input.setRetries(retries + 1);
    input.setTimer(window.setTimeout(() => void watch(input, true), 1000 * 2 ** retries));
  }
}

function bindPeer(input: RequestInput, n: number, terminal: TerminalView, cursor: number) {
  const binding = input.readBinding()!,
    peer = binding.transport.websocket(`factory/runs/${binding.runId}/output`),
    queued = {bytes: 0};
  input.setSocket(peer);
  peer.onopen = () => onPeerOpen(input, n, peer, cursor);
  peer.onmessage = (event) => onPeerMessage(input, event, n, terminal, queued);
  peer.onerror = peer.onclose = () => onPeerClosed(input, n);
}

export async function watch(input: RequestInput, resume = false) {
  if (blockedWatch(input)) return;
  if (!resume) {
    input.clearScreen();
    input.setCursor(0);
  }
  input.clearTimer();
  input.setState('opening');
  const n = input.beginGeneration(),
    request = input.takeRequest();
  input.setTimer(
    window.setTimeout(() => input.detach('Factory view timed out. Inspect the exact run; nothing was replayed.'), 45000)
  );
  input.setNotice(true);
  input.setMessage('Reading the recorded run…');
  try {
    if (await inspectRun(input, n, request)) return;
    await attachPeer(input, n);
  } catch {
    if (input.isLive(n))
      input.detach('Could not authorize or attach. Sign in again or inspect the exact run; nothing was replayed.');
  }
}

async function attachPeer(input: RequestInput, n: number) {
  const {Terminal, FitAddon} = await input.loadRenderer();
  if (!input.isLive(n)) return;
  const screen = await input.awaitScreen(n);
  if (!screen || !input.isLive(n)) return;
  const terminal = input.readTerminal() || input.openScreen(screen, Terminal, FitAddon);
  bindPeer(input, n, terminal, input.readCursor());
}
