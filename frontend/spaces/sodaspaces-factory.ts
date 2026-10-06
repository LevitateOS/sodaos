import {LitElement} from 'lit';
import {renderFactoryWatch} from './sodaspaces-factory-view.js';
import {statusText, watchCommands, watchPresentation, watchTitle} from './sodaspaces-factory-display.js';
import type {DisplayInput} from './sodaspaces-factory-display.js';
import {object, readSodaJSON} from './sodaspaces-api.js';
import {factoryOutputFrame, factoryOutputCursor, factoryClosedReason} from './sodaspaces-factory-stream-response.js';
import {factoryRunStatusResponse, factoryStatusFrame} from './sodaspaces-factory-response.js';
import type {TerminalView, Renderer} from './sodaspaces-terminal.js';
import type {FitAddon} from '@xterm/addon-fit';
import type {ITerminalAddon} from '@xterm/xterm';
import type {PreparedExtensionMount} from './soda-extension.js';

export interface FactoryWatchContext {
  expectedUserId: string;
  transport: PreparedExtensionMount;
  repositoryId: string;
  environmentId: string;
  projectName?: string;
  runId: string;
  role: string;
  issue?: string;
  attempt?: string;
}

const renderer = async (): Promise<Renderer> => {
  const [{Terminal}, {FitAddon}] = await Promise.all([
    import('./soda-terminal/xterm.mjs'),
    import('./soda-terminal/addon-fit.mjs'),
  ]);
  return {
    Terminal,
    FitAddon,
  };
};

// One immutable run binding and imperative attachment owner. Rendering never
// launches, resumes or inputs to the run; the socket only reads. Detaching or
// hiding the view never ends execution or returns its lease.
export class SodaFactoryWatch extends LitElement {
  static properties = {
    state: {
      state: true,
    },
    message: {
      state: true,
    },
    screenVisible: {
      state: true,
    },
    statusLine: {
      state: true,
    },
    notice: {
      state: true,
    },
  };
  declare private state: 'idle' | 'opening' | 'ready' | 'closed' | 'stale';
  declare private message: string;
  declare private screenVisible: boolean;
  declare private statusLine: string;
  declare private notice: boolean;
  private binding: FactoryWatchContext | undefined;
  private loadRenderer: () => Promise<Renderer> = renderer;
  private lifetime = new AbortController();
  private disposed = false;
  private generation = 0;
  private retries = 0;
  private cursor = 0;
  private socket: WebSocket | undefined;
  private terminal: TerminalView | undefined;
  private fit: (Pick<FitAddon, 'fit'> & ITerminalAddon) | undefined;
  private observer: ResizeObserver | undefined;
  private timer: number | undefined;
  private request: AbortController | undefined;
  private viewVisible = true;
  constructor() {
    super();
    this.notice = true;
    this.state = 'idle';
    this.screenVisible = false;
    this.statusLine = '';
    this.message = 'Not watching.';
  }
  protected createRenderRoot() {
    return this;
  }
  configure(context: FactoryWatchContext, load: () => Promise<Renderer>) {
    if (this.binding || this.disposed) throw Error('Factory binding is immutable');
    this.binding = {
      ...context,
    };
    this.loadRenderer = load;
    document.fonts.addEventListener('loadingdone', this.resize, {
      signal: this.lifetime.signal,
    });
    window.visualViewport?.addEventListener('resize', this.resize, {
      signal: this.lifetime.signal,
    });
    window.addEventListener('pagehide', () => this.invalidate(), {
      signal: this.lifetime.signal,
    });
    window.addEventListener(
      'pageshow',
      (event) => {
        if (event.persisted) this.invalidate();
      },
      {
        signal: this.lifetime.signal,
      }
    );
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  protected render() {
    return renderFactoryWatch(watchPresentation(this.displayInput()), watchCommands(this.displayInput()));
  }
  private displayInput(): DisplayInput {
    return {
      readDisposed: () => this.disposed,
      readState: () => this.state,
      readMessage: () => this.message,
      readNotice: () => this.notice,
      readScreenVisible: () => this.screenVisible,
      readStatusLine: () => this.statusLine,
      readIssue: () => this.binding?.issue,
      readAttempt: () => this.binding?.attempt,
      readRunId: () => this.binding?.runId,
      readRole: () => this.binding?.role,
      setRetries: (retries) => {
        this.retries = retries;
      },
      setCursor: (cursor) => {
        this.cursor = cursor;
      },
      isHiddenOrInert: () => !!this.closest('[hidden], [inert]'),
      queryDetails: () => this.querySelector('details'),
      focusSummary: () => {
        this.querySelector<HTMLElement>('summary')?.focus();
      },
      dispatchHide: () => {
        this.dispatchEvent(
          new CustomEvent('soda-factory-command', {
            bubbles: true,
            detail: 'hide',
          })
        );
      },
      watchNow: () => {
        void this.watch();
      },
      detach: (message) => this.detach(message),
    };
  }
  private live(generation: number) {
    return !this.disposed && this.state !== 'stale' && this.generation === generation;
  }
  private closeSocket() {
    const old = this.socket;
    this.socket = undefined;
    if (old && old.readyState < 2) old.close();
  }
  private clearScreen() {
    this.terminal?.dispose();
    this.terminal = undefined;
    this.fit = undefined;
    this.querySelector('.soda-terminal-screen')?.replaceChildren();
    this.screenVisible = false;
  }
  private detach(message: string, stale = false) {
    ++this.generation;
    this.state = stale ? 'stale' : 'closed';
    this.notice = true;
    window.clearTimeout(this.timer);
    this.request?.abort();
    this.request = undefined;
    this.observer?.disconnect();
    this.observer = undefined;
    this.closeSocket();
    this.message = message;
  }
  invalidate() {
    this.detach('Page context changed. No action was replayed; watch through a fresh authorized page.', true);
  }
  private async json(path: string, signal: AbortSignal) {
    if (!this.binding) throw Error('Missing factory binding');
    const response = await this.binding.transport.request(path, {signal});
    if (!response.ok) {
      if (response.status === 401 || response.status === 403) this.authorityLost();
      throw Error('Soda request refused');
    }
    return object(await readSodaJSON(response));
  }
  private authorityLost() {
    if (!this.disposed)
      this.dispatchEvent(
        new CustomEvent('soda-factory-authority-lost', {
          bubbles: true,
        })
      );
  }
  private fitReady() {
    return !!this.terminal && !!this.fit && this.viewVisible && !this.closest('[hidden]');
  }
  private screenPresent(screen: HTMLElement | null): screen is HTMLElement {
    return !!screen?.isConnected && !!screen.clientWidth && !!screen.clientHeight;
  }
  private canFit(screen: HTMLElement | null): screen is HTMLElement {
    return this.fitReady() && this.screenPresent(screen) && (this.state === 'ready' || this.state === 'opening');
  }
  // Presentation only: fitting never sends a resize frame. The read-only
  // stream rejects every client frame past the handshake.
  private resize = () => {
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!this.canFit(screen)) return;
    this.fit!.fit();
  };
  private requiredToken(styles: CSSStyleDeclaration, name: string) {
    const value = styles.getPropertyValue(name).trim();
    if (!value) throw Error(`Missing terminal token ${name}`);
    return value;
  }
  private terminalTheme(styles: CSSStyleDeclaration) {
    return {
      background: this.requiredToken(styles, '--soda-terminal-screen'),
      foreground: this.requiredToken(styles, '--soda-terminal-text'),
      cursor: this.requiredToken(styles, '--soda-terminal-text'),
    };
  }
  private async awaitScreen(n: number) {
    this.screenVisible = true;
    await this.updateComplete;
    if (!this.live(n) || !this.viewVisible || !this.isConnected || this.closest('[hidden]')) return;
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!screen) throw Error('Missing factory screen');
    await document.fonts.ready;
    if (!this.live(n) || !this.isConnected || this.closest('[hidden]')) return;
    return screen;
  }
  private openScreen(screen: HTMLElement, Terminal: Renderer['Terminal'], FitAddon: Renderer['FitAddon']) {
    const styles = getComputedStyle(this);
    const fontSize = Number.parseFloat(this.requiredToken(styles, '--soda-font-mono-size')),
      lineHeight = Number(this.requiredToken(styles, '--soda-font-mono-line'));
    if (!Number.isFinite(fontSize) || fontSize <= 0 || !Number.isFinite(lineHeight) || lineHeight < 1)
      throw Error('Invalid factory typography');
    const terminal = (this.terminal = new Terminal({
      allowProposedApi: true,
      disableStdin: true,
      scrollback: 1000,
      windowOptions: {},
      convertEol: false,
      cols: 80,
      rows: 24,
      fontFamily: this.requiredToken(styles, '--soda-font-mono-family'),
      fontSize,
      lineHeight,
      theme: this.terminalTheme(styles),
    }));
    const fit = (this.fit = new FitAddon());
    terminal.loadAddon(fit);
    for (const code of [0, 1, 2, 8, 52]) terminal.parser.registerOscHandler(code, () => true);
    terminal.open(screen);
    if (screen.clientWidth && screen.clientHeight) fit.fit();
    return terminal;
  }
  private blockedWatch() {
    return (
      this.disposed || !this.binding || this.state === 'stale' || this.state === 'opening' || this.state === 'ready'
    );
  }
  private endedRun(detail: {state?: {phase: string; exit_code?: number; output?: string; output_truncated: boolean}}) {
    const state = detail.state;
    this.statusLine = state ? statusText(state.phase, false, true, state.exit_code ?? null) : 'ended';
    const excerpt = state?.output ? `\nLast output:\n${state.output}` : '';
    const cut = state?.output_truncated ? '\n… output truncated …' : '';
    this.detach(
      `Run ended (${this.statusLine}). Reattachment is unavailable; this view keeps its status.${excerpt}${cut}`
    );
  }
  private async inspectRun(n: number, request: AbortController) {
    const detail = factoryRunStatusResponse(
      await this.json(`factory/runs/${this.binding!.runId}`, request.signal),
      this.binding!.runId,
      this.binding!.repositoryId
    );
    if (!this.live(n)) return true;
    if (detail.state?.terminal) {
      this.endedRun(detail);
      return true;
    }
    if (detail.state) this.statusLine = statusText(detail.state.phase, detail.state.live, false, null);
    else this.statusLine = detail.outcome ? `recorded · ${detail.outcome}` : 'recorded';
    return false;
  }
  private attachPayload(cursor: number) {
    return {
      run_id: this.binding!.runId,
      repository_id: this.binding!.repositoryId,
      session_generation: this.binding!.transport.generation,
      cursor,
    };
  }
  private onPeerOpen(n: number, peer: WebSocket, cursor: number) {
    if (!this.live(n)) {
      peer.close();
      return;
    }
    try {
      peer.send(JSON.stringify(this.attachPayload(cursor)));
      this.message = 'Attaching the recorded run…';
    } catch {
      this.detach('Attachment dispatch was not confirmed. No input was sent.');
    }
  }
  private acceptStatus(frame: ReturnType<typeof factoryStatusFrame>) {
    this.statusLine = statusText(frame.phase, frame.live, frame.terminal, frame.exit_code);
    if (this.state === 'opening') {
      window.clearTimeout(this.timer);
      this.state = 'ready';
      this.retries = 0;
      this.notice = false;
      this.message = `Watching ${watchTitle(this.displayInput())}.`;
      void this.screenReady();
    }
  }
  private acceptOutput(frame: ReturnType<typeof factoryOutputFrame>, terminal: TerminalView, queued: {bytes: number}) {
    if (this.state !== 'ready') throw Error('frame');
    const next = factoryOutputCursor(this.cursor, frame);
    if (frame.truncated) terminal.write('\r\n… earlier output truncated …\r\n');
    if (frame.gap) terminal.write(`\r\n… output gap: skipped to byte ${frame.next} …\r\n`);
    if (queued.bytes + frame.bytes.length > 262144) throw Error('output');
    queued.bytes += frame.bytes.length;
    this.cursor = next;
    if (frame.bytes.length)
      terminal.write(frame.bytes, () => {
        queued.bytes -= frame.bytes.length;
      });
  }
  private dispatchFrame(frame: Record<string, unknown>, terminal: TerminalView, queued: {bytes: number}) {
    if (frame.type === 'status' && this.binding) this.acceptStatus(factoryStatusFrame(frame, this.binding.runId));
    else if (frame.type === 'output') this.acceptOutput(factoryOutputFrame(frame), terminal, queued);
    else if (frame.type === 'closed') this.detachClosed(factoryClosedReason(frame));
    else throw Error('frame');
  }
  private detachClosed(reason: string) {
    const ended = reason === 'eof' ? 'Run ended. This view keeps its rendered output.' : `View ended: ${reason}.`;
    this.detach(ended);
  }
  private onPeerMessage(event: MessageEvent, n: number, terminal: TerminalView, queued: {bytes: number}) {
    if (!this.live(n) || this.terminal !== terminal) return;
    try {
      if (typeof event.data !== 'string' || event.data.length > 65536) throw Error('frame');
      this.dispatchFrame(object(JSON.parse(event.data)), terminal, queued);
    } catch {
      this.detach('Invalid or overloaded stream. No input was sent.');
    }
  }
  private onPeerClosed(n: number) {
    if (!this.live(n)) return;
    this.detach('Connection lost. The run continues without this view.');
    if (this.retries < 3) {
      const wait = 1000 * 2 ** this.retries++;
      this.timer = window.setTimeout(() => this.watch(true), wait);
    }
  }
  private bindPeer(n: number, terminal: TerminalView, cursor: number) {
    const peer = (this.socket = this.binding!.transport.websocket(`factory/runs/${this.binding!.runId}/output`)),
      queued = {bytes: 0};
    peer.onopen = () => this.onPeerOpen(n, peer, cursor);
    peer.onmessage = (event) => this.onPeerMessage(event, n, terminal, queued);
    peer.onerror = peer.onclose = () => this.onPeerClosed(n);
  }
  async watch(resume = false) {
    if (this.blockedWatch()) return;
    if (!resume) {
      this.clearScreen();
      this.cursor = 0;
    }
    window.clearTimeout(this.timer);
    this.state = 'opening';
    const n = ++this.generation,
      request = (this.request = new AbortController());
    this.timer = window.setTimeout(
      () => this.detach('Factory view timed out. Inspect the exact run; nothing was replayed.'),
      45000
    );
    this.notice = true;
    this.message = 'Reading the recorded run…';
    try {
      if (await this.inspectRun(n, request)) return;
      await this.attachPeer(n);
    } catch {
      if (this.live(n))
        this.detach('Could not authorize or attach. Sign in again or inspect the exact run; nothing was replayed.');
    }
  }
  private async attachPeer(n: number) {
    const {Terminal, FitAddon} = await this.loadRenderer();
    if (!this.live(n)) return;
    const screen = await this.awaitScreen(n);
    if (!screen || !this.live(n)) return;
    const terminal = this.terminal || this.openScreen(screen, Terminal, FitAddon);
    this.bindPeer(n, terminal, this.cursor);
  }
  private async screenReady() {
    const n = this.generation;
    try {
      await this.updateComplete;
      if (!this.live(n) || !this.terminal) return;
      const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
      if (!screen?.isConnected) return;
      this.observer = new ResizeObserver(this.resize);
      this.observer.observe(screen);
      this.resize();
    } catch {
      if (this.live(n)) this.detach('Factory rendering failed. No input was sent.');
    }
  }
  setVisible(visible: boolean) {
    this.viewVisible = visible;
    if (visible) this.resize(); // Presentation only: never connect.
  }
  stop() {
    this.detach('Stopped watching. The run continues without this view.');
  }
  get started() {
    return this.state !== 'idle';
  }
  dispose() {
    if (this.disposed) return;
    this.detach('Detached.', true);
    this.clearScreen();
    this.disposed = true;
    this.lifetime.abort();
    this.remove();
  }
}
customElements.define('soda-factory-watch', SodaFactoryWatch);
function optionalMatches(value: string | undefined, pattern: RegExp) {
  return value === undefined || pattern.test(value);
}
function admitWatchIds(
  expectedUserId: string,
  repositoryId: string,
  environmentId: string,
  runId: string,
  role: string
) {
  if (
    !/^[1-9][0-9]{0,18}$/.test(expectedUserId) ||
    !/^[1-9][0-9]{0,18}$/.test(repositoryId) ||
    !/^p[0-9a-f]{24}$/.test(environmentId) ||
    !/^[0-9a-f]{32}$/.test(runId) ||
    !/^[a-z][a-z0-9-]{0,63}$/.test(role)
  )
    throw Error('Invalid factory watch context');
}
function admitWatchContext(root: HTMLElement, context: FactoryWatchContext) {
  const {expectedUserId, repositoryId, environmentId, runId, role} = context;
  admitWatchIds(expectedUserId, repositoryId, environmentId, runId, role);
  if (
    root.ownerDocument !== document ||
    !optionalMatches(context.issue, /^[1-9][0-9]{0,18}$/) ||
    !optionalMatches(context.attempt, /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/)
  )
    throw Error('Invalid factory watch context');
}
export function mountFactoryWatch(
  root: HTMLElement,
  context: FactoryWatchContext,
  loadRenderer: () => Promise<Renderer> = renderer
) {
  admitWatchContext(root, context);
  const view = new SodaFactoryWatch();
  view.configure(context, loadRenderer);
  root.append(view);
  return {
    setVisible: (visible: boolean) => view.setVisible(visible),
    watch: () => view.watch(),
    get started() {
      return view.started;
    },
    get ready() {
      return view.updateComplete;
    },
    stop: () => view.stop(),
    invalidate: () => view.invalidate(),
    dispose: () => view.dispose(),
  };
}
