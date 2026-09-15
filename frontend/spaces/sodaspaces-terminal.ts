import {LitElement} from 'lit';
import {renderTerminal} from './sodaspaces-terminal-view.js';
import type {ConnectionState, TerminalObservation} from './sodaspaces-attention.js';
import {object, readSodaJSON, terminalID, terminalResponse, id as identifier} from './sodaspaces-api.js';
import type {Terminal, ITerminalOptions, ITerminalInitOnlyOptions, ITerminalAddon} from '@xterm/xterm';
import type {FitAddon} from '@xterm/addon-fit';
import type {TerminalMetadata} from './sodaspaces-api.js';
export type TerminalView = Pick<
  Terminal,
  | 'cols'
  | 'rows'
  | 'options'
  | 'parser'
  | 'loadAddon'
  | 'attachCustomKeyEventHandler'
  | 'onData'
  | 'onRender'
  | 'open'
  | 'focus'
  | 'dispose'
  | 'write'
>;
export interface Renderer {
  Terminal: new (options: ITerminalOptions & ITerminalInitOnlyOptions) => TerminalView;
  FitAddon: new () => Pick<FitAddon, 'fit'> & ITerminalAddon;
}
export interface TerminalContext {
  expectedUserId: string;
  csrfToken: string;
  repositoryId: string;
  environmentId: string;
  login: string;
  projectName?: string;
}
export type TerminalLocator =
  | {
      kind: 'new';
    }
  | {
      kind: 'existing';
      id: string;
    };
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
// One immutable original account and imperative attachment owner. Rendering never
// creates a shell, opens a socket, or touches xterm's screen descendants.
export class SodaTerminal extends LitElement {
  static properties = {
    state: {
      state: true,
    },
    sessionID: {
      state: true,
    },
    message: {
      state: true,
    },
    screenVisible: {
      state: true,
    },
    actionBusy: {
      state: true,
    },
    confirming: {
      state: true,
    },
    sessionName: {
      state: true,
    },
    notice: {
      state: true,
    },
  };
  declare private state: 'idle' | 'opening' | 'ready' | 'closed' | 'stale';
  declare private sessionID: string | undefined;
  declare private message: string;
  declare private screenVisible: boolean;
  declare private actionBusy: boolean;
  declare private confirming: string | undefined;
  declare private sessionName: string;
  declare private notice: boolean;
  private createName = '';
  private managedEnded = false;
  private binding: TerminalContext | undefined;
  private loadRenderer: () => Promise<Renderer> = renderer;
  private lifetime = new AbortController();
  private disposed = false;
  private generation = 0;
  private retries = 0;
  private socket: WebSocket | undefined;
  private terminal: TerminalView | undefined;
  private fit: (Pick<FitAddon, 'fit'> & ITerminalAddon) | undefined;
  private observer: ResizeObserver | undefined;
  private timer: number | undefined;
  private request: AbortController | undefined;
  private actionRequest: AbortController | undefined;
  private viewVisible = true;
  private lastSize = '';
  private minimumSize:
    | {
        width: number;
        height: number;
      }
    | undefined;
  constructor() {
    super();
    this.confirming = undefined;
    this.sessionName = '';
    this.notice = true;
    this.state = 'idle';
    this.sessionID = undefined;
    this.screenVisible = this.actionBusy = false;
    this.message = 'Not connected.';
  }
  protected createRenderRoot() {
    return this;
  }
  configure(context: TerminalContext, load: () => Promise<Renderer>, locator: TerminalLocator) {
    if (this.binding || this.disposed) throw Error('Terminal binding is immutable');
    this.binding = {
      ...context,
    };
    this.loadRenderer = load;
    if (locator.kind === 'existing') this.sessionID = locator.id;
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
    return renderTerminal(this.terminalPresentation(), this.terminalCommands());
  }
  private viewDisabled() {
    return this.disposed || this.state === 'stale';
  }
  private canConnectView(disabled: boolean) {
    return !(disabled || this.managedEnded || this.state === 'opening' || this.state === 'ready' || this.actionBusy);
  }
  private canEndView(disabled: boolean) {
    return !disabled && !!this.sessionID && !this.actionBusy;
  }
  private contextLabels() {
    return {
      name: this.sessionName || 'Terminal',
      login: this.binding?.login || '',
      project: this.binding?.projectName || this.binding?.environmentId || '',
    };
  }
  private terminalPresentation() {
    const disabled = this.viewDisabled(),
      labels = this.contextLabels();
    return {
      ready: this.state === 'ready',
      disabled,
      canConnect: this.canConnectView(disabled),
      canEnd: this.canEndView(disabled),
      connectLabel: this.sessionID ? 'Reconnect terminal' : 'Open terminal',
      name: labels.name,
      login: labels.login,
      project: labels.project,
      message: this.message,
      notice: this.notice,
      screenVisible: this.screenVisible,
      confirmingName: this.confirming ? this.sessionName || this.confirming : null,
      canConfirm: !disabled && !this.actionBusy && this.confirming === this.sessionID,
    };
  }
  private menuKey = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      this.closeMenu();
      this.querySelector<HTMLElement>('summary')?.focus();
    }
  };
  private terminalCommands() {
    return {
      connect: () => this.connectFromControls(),
      end: () => this.confirmEnd(),
      confirmEnd: () => this.endConfirmedTerminal(),
      cancelEnd: () => this.cancelEnd(),
      project: () => this.workspaceCommand('project'),
      rename: () => this.workspaceCommand('rename'),
      hide: () => this.workspaceCommand('hide'),
      menuKey: this.menuKey,
    };
  }
  private connectFromControls() {
    this.closeMenu();
    this.retries = 0;
    void this.connect();
  }
  private endConfirmedTerminal() {
    // Admission must see the exact confirmed ID before the dialog is cleared.
    if (this.confirming === this.sessionID && !this.closest('[hidden], [inert]')) void this.endTerminal();
    this.confirming = undefined;
  }
  private closeMenu() {
    const menu = this.querySelector('details');
    if (menu) menu.open = false;
  }
  private workspaceCommand(command: 'project' | 'rename' | 'hide') {
    if (this.disposed || this.state === 'stale' || this.closest('[hidden], [inert]')) return;
    this.closeMenu();
    this.dispatchEvent(
      new CustomEvent('soda-terminal-command', {
        bubbles: true,
        detail: command,
      })
    );
  }
  private confirmEnd() {
    if (
      !this.sessionID ||
      this.disposed ||
      this.state === 'stale' ||
      this.actionBusy ||
      this.closest('[hidden], [inert]')
    )
      return;
    this.closeMenu();
    this.confirming = this.sessionID;
    const target = this.confirming,
      active = document.activeElement;
    void this.updateComplete.then(() => {
      if (
        this.confirming === target &&
        !this.disposed &&
        (document.activeElement === active || document.activeElement === document.body)
      )
        this.querySelector<HTMLElement>('[data-action=cancel-end]')?.focus();
    });
  }
  private cancelEnd() {
    this.confirming = undefined;
    this.querySelector<HTMLElement>('[data-action=controls]')?.focus();
  }
  private remember(value: string | null) {
    if (value === null) this.managedEnded = true;
    this.dispatchEvent(
      new CustomEvent('soda-terminal-locator', {
        bubbles: true,
        detail: value,
      })
    );
  }
  private live(generation: number) {
    return !this.disposed && this.state !== 'stale' && this.generation === generation;
  }
  private publish(kind: TerminalObservation['kind'], state: ConnectionState) {
    if (this.disposed) return;
    const detail: TerminalObservation = {
      kind,
      state,
      generation: this.generation,
      id: this.sessionID || null,
    };
    this.dispatchEvent(
      new CustomEvent<TerminalObservation>('soda-terminal-observation', {
        bubbles: true,
        detail,
      })
    );
  }
  private abortActionIfStale(stale: boolean) {
    if (!stale) return;
    this.actionRequest?.abort();
    this.actionRequest = undefined;
    this.actionBusy = false;
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
    this.lastSize = '';
    this.querySelector('.soda-terminal-screen')?.replaceChildren();
    this.screenVisible = false;
  }
  private detach(message: string, stale = false, reason: ConnectionState = 'connection-lost') {
    ++this.generation;
    this.state = stale ? 'stale' : 'closed';
    this.notice = true;
    this.confirming = undefined;
    window.clearTimeout(this.timer);
    this.request?.abort();
    this.request = undefined;
    this.abortActionIfStale(stale);
    this.observer?.disconnect();
    this.observer = undefined;
    this.closeSocket();
    this.clearScreen();
    this.message = message;
    this.publish('state', reason);
  }
  invalidate() {
    this.detach('Page context changed. No action was replayed; reconnect through a fresh authorized page.', true);
  }
  private async json(path: string, body: Record<string, unknown> | null, signal: AbortSignal) {
    if (!this.binding) throw Error('Missing terminal binding');
    const headers: Record<string, string> = {
      'X-Soda-Expected-User-ID': this.binding.expectedUserId,
    };
    if (body) {
      headers['Content-Type'] = 'application/json';
      headers['X-CSRF-Token'] = this.binding.csrfToken;
    }
    const response = await fetch('/-/soda' + path, {
      method: body ? 'POST' : 'GET',
      credentials: 'same-origin',
      cache: 'no-store',
      redirect: 'error',
      headers,
      ...(body
        ? {
            body: JSON.stringify(body),
          }
        : {}),
      signal,
    });
    if (!response.ok) {
      if (response.status === 401 || response.status === 403) this.authorityLost();
      throw Error('Soda request refused');
    }
    return object(await readSodaJSON(response));
  }
  private authorityLost() {
    if (!this.disposed)
      this.dispatchEvent(
        new CustomEvent('soda-terminal-authority-lost', {
          bubbles: true,
        })
      );
  }
  private actionCurrent(control: AbortController, target: string) {
    return !this.disposed && this.state !== 'stale' && this.actionRequest === control && this.sessionID === target;
  }
  private confirmNativeEnd(result: Record<string, unknown>, target: string) {
    const observed = terminalResponse(result, this.binding!);
    if (observed && (observed.id !== target || observed.state !== 'ended')) throw Error('outcome');
  }
  private endUnconfirmed() {
    this.notice = true;
    this.message = 'End was not confirmed. Inspect this exact terminal; no retry or replacement was made.';
    this.publish('state', 'unconfirmed');
  }
  private finishAction(control: AbortController, timeout: number) {
    window.clearTimeout(timeout);
    if (this.actionRequest === control) {
      this.actionRequest = undefined;
      this.actionBusy = false;
    }
  }
  private endBlocked() {
    return this.disposed || this.state === 'stale' || !this.sessionID || !this.binding || this.actionBusy;
  }
  private async endTerminal() {
    if (this.endBlocked()) return;
    const target = this.sessionID!,
      control = (this.actionRequest = new AbortController());
    this.actionBusy = true;
    const timeout = window.setTimeout(() => control.abort(), 30000);
    try {
      const result = await this.json(
        `/api/environments/${this.binding!.environmentId}/terminal-sessions/${target}`,
        {action: 'end'},
        control.signal
      );
      if (!this.actionCurrent(control, target)) return;
      this.confirmNativeEnd(result, target);
      this.detach('Native cleanup confirmed. Files and independent services remain.', false, 'ended');
      this.sessionID = undefined;
      this.remember(null);
    } catch {
      if (this.actionCurrent(control, target)) this.endUnconfirmed();
    } finally {
      this.finishAction(control, timeout);
    }
  }
  private send(
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
    if (this.state !== 'ready' || !this.socket || this.socket.readyState !== 1 || this.socket.bufferedAmount > 65536) {
      this.detach('Connection lost or overloaded. Reconnect the existing terminal; input was not replayed.');
      return false;
    }
    try {
      this.socket.send(JSON.stringify(frame));
      return true;
    } catch {
      this.detach('Connection lost. No input was replayed.');
      return false;
    }
  }
  private canFit(screen: HTMLElement | null): screen is HTMLElement {
    return (
      !!this.terminal &&
      !!this.fit &&
      this.viewVisible &&
      !this.closest('[hidden]') &&
      this.state === 'ready' &&
      !!screen?.isConnected &&
      !!screen.clientWidth &&
      !!screen.clientHeight
    );
  }
  private resize = () => {
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!this.canFit(screen)) return;
    this.fit!.fit();
    const {cols, rows} = this.terminal!,
      fitted = cols + ':' + rows;
    if (cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300 && this.lastSize !== fitted) {
      this.lastSize = fitted;
      this.send({
        type: 'resize',
        cols,
        rows,
      });
    }
  };
  private geometryScreen() {
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (
      !this.terminal ||
      this.state !== 'ready' ||
      !this.viewVisible ||
      !screen?.isConnected ||
      this.closest('[hidden]')
    )
      return;
    return screen;
  }
  private paneMinimum(screen: HTMLElement) {
    const {cols, rows} = this.terminal!;
    const grid = screen.querySelector<HTMLElement>('.xterm-screen')?.getBoundingClientRect();
    if (!grid?.width || !grid.height || !cols || !rows) return;
    const css = getComputedStyle(screen),
      scrollbar = screen.querySelector<HTMLElement>('.scrollbar.vertical')?.getBoundingClientRect().width || 0;
    return {
      width:
        Math.ceil((grid.width / cols) * 56 + scrollbar + parseFloat(css.paddingLeft) + parseFloat(css.paddingRight)) +
        2,
      height:
        Math.ceil(
          (grid.height / rows) * 12 +
            this.offsetHeight -
            screen.offsetHeight +
            parseFloat(css.paddingTop) +
            parseFloat(css.paddingBottom)
        ) + 2,
    };
  }
  private publishMinimum(minimum: {width: number; height: number}) {
    if (
      minimum.width <= 0 ||
      minimum.height <= 0 ||
      (minimum.width === this.minimumSize?.width && minimum.height === this.minimumSize?.height)
    )
      return;
    this.minimumSize = minimum;
    this.dispatchEvent(
      new CustomEvent('soda-terminal-geometry', {
        bubbles: true,
        detail: minimum,
      })
    );
  }
  private measureMinimum() {
    const screen = this.geometryScreen();
    if (!screen) return;
    const minimum = this.paneMinimum(screen);
    if (minimum) this.publishMinimum(minimum);
  }
  private blockedConnect() {
    return (
      this.disposed ||
      !this.binding ||
      this.state === 'stale' ||
      this.state === 'opening' ||
      this.state === 'ready' ||
      this.actionBusy
    );
  }
  private hiddenConnect(automatic: boolean) {
    return (
      !this.viewVisible ||
      !!this.closest('[hidden]') ||
      (!automatic && (document.visibilityState === 'hidden' || !document.hasFocus()))
    );
  }
  private canStartConnect(automatic: boolean) {
    if (this.blockedConnect()) return false;
    if (this.managedEnded) {
      this.message = 'This exact session ended. Use New terminal for a different shell.';
      return false;
    }
    if (automatic && !this.sessionID) return false;
    return !this.hiddenConnect(automatic);
  }
  private detachUnready(state: TerminalMetadata['state']) {
    this.detach(
      'Native transition is still in progress. No replacement was made.',
      false,
      state === 'ending' ? 'ending' : 'unavailable'
    );
  }
  private hiddenWhileAttaching() {
    return !this.viewVisible || !this.isConnected || !!this.closest('[hidden]');
  }
  private cancelIfHidden() {
    if (!this.hiddenWhileAttaching()) return false;
    this.detach('Attachment cancelled while hidden. No creation was sent.');
    return true;
  }
  private async inspectExisting(n: number, request: AbortController) {
    if (!this.sessionID) return false;
    const metadata = await this.json(
      `/api/environments/${this.binding!.environmentId}/terminal-sessions/${this.sessionID}`,
      null,
      request.signal
    );
    if (!this.live(n)) return true;
    const existing = terminalResponse(metadata, this.binding!);
    if (existing && existing.id !== this.sessionID) throw Error('terminal metadata');
    if (!existing || existing.state === 'ended') {
      this.sessionID = undefined;
      this.remember(null);
      this.detach('Native terminal is absent or ended. Use New terminal for a different shell.', false, 'ended');
      return true;
    }
    this.observe(existing);
    if (!existing.ready) {
      this.detachUnready(existing.state);
      return true;
    }
    if (existing.attached) {
      this.detach(
        'An existing writer is attached. No takeover or replacement was requested.',
        false,
        'attached-elsewhere'
      );
      return true;
    }
    return false;
  }
  private async awaitScreen(n: number) {
    this.screenVisible = true;
    await this.updateComplete;
    if (!this.live(n) || this.cancelIfHidden()) return;
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!screen) throw Error('Missing terminal screen');
    await document.fonts.ready;
    if (!this.live(n) || this.cancelIfHidden()) return;
    return screen;
  }
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
  private interceptControlFocus = (event: KeyboardEvent) => {
    if (event.ctrlKey && event.shiftKey && event.key === 'Enter') {
      event.preventDefault();
      event.stopPropagation();
      this.querySelector<HTMLElement>('[data-action=controls]')?.focus();
      return false;
    }
    return true;
  };
  private sendInput(n: number, screen: HTMLElement, data: string) {
    if (!this.live(n) || !this.viewVisible || this.closest('[hidden]') || !screen.contains(document.activeElement))
      return;
    if (data.length > 65536) {
      this.detach('Input too large. Nothing was replayed.');
      return;
    }
    const bytes = new TextEncoder().encode(data);
    for (let i = 0; i < bytes.length; i += 16384)
      if (
        !this.send({
          type: 'input',
          data: btoa(String.fromCharCode(...bytes.subarray(i, i + 16384))),
        })
      )
        break;
  }
  private openTerminal(n: number, screen: HTMLElement, Terminal: Renderer['Terminal'], FitAddon: Renderer['FitAddon']) {
    const styles = getComputedStyle(this);
    const fontSize = Number.parseFloat(this.requiredToken(styles, '--soda-font-mono-size')),
      lineHeight = Number(this.requiredToken(styles, '--soda-font-mono-line'));
    if (!Number.isFinite(fontSize) || fontSize <= 0 || !Number.isFinite(lineHeight) || lineHeight < 1)
      throw Error('Invalid terminal typography');
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
    terminal.attachCustomKeyEventHandler(this.interceptControlFocus);
    terminal.onData((data) => this.sendInput(n, screen, data));
    terminal.onRender(() => this.measureIfCurrent(n, terminal));
    terminal.open(screen);
    if (screen.clientWidth && screen.clientHeight) fit.fit();
    return terminal;
  }
  private measureIfCurrent(n: number, terminal: TerminalView) {
    if (this.live(n) && this.terminal === terminal) this.measureMinimum();
  }
  private async reserveCreate(n: number, request: AbortController, cols: number, rows: number) {
    const reserved = await this.json(
      `/api/environments/${this.binding!.environmentId}/terminal-sessions`,
      {cols, rows, name: this.createName},
      request.signal
    );
    if (!this.live(n)) return true;
    if (!terminalID(reserved.id)) throw Error('terminal reservation');
    this.sessionID = reserved.id;
    this.setName(this.createName);
    this.remember(reserved.id); // published BEFORE any native Create can be sent
    return !this.live(n);
  }
  private attachPayload(action: 'attach' | 'create', cols: number, rows: number) {
    const {expectedUserId, repositoryId} = this.binding!;
    return {
      action,
      id: this.sessionID,
      ...(action === 'create' ? {name: this.createName} : {}),
      expected_user_id: expectedUserId,
      repository_id: repositoryId,
      csrf_token: this.binding?.csrfToken,
      cols,
      rows,
    };
  }
  private onPeerOpen(n: number, peer: WebSocket, action: 'attach' | 'create', cols: number, rows: number) {
    if (!this.live(n)) {
      peer.close();
      return;
    }
    if (this.cancelIfHidden()) return;
    this.publish('state', 'opening');
    try {
      peer.send(JSON.stringify(this.attachPayload(action, cols, rows)));
      this.message = action === 'create' ? 'Opening terminal…' : 'Attaching the existing terminal…';
    } catch {
      this.detach('Attachment dispatch was not confirmed. No creation or input was retried.');
    }
  }
  private refreshAttachedMetadata(n: number, request: AbortController) {
    const target = this.sessionID,
      environmentId = this.binding!.environmentId;
    void this.json(
      `/api/environments/${environmentId}/terminal-sessions/${target}`,
      null,
      AbortSignal.any([request.signal, AbortSignal.timeout(10000)])
    )
      .then((result) => {
        if (!this.live(n) || this.sessionID !== target || !this.binding) return;
        const metadata = terminalResponse(result, this.binding);
        if (metadata && metadata.id === target) this.observe(metadata);
      })
      .catch(() => {});
  }
  private acceptReady(
    n: number,
    terminal: TerminalView,
    screen: HTMLElement,
    action: 'attach' | 'create',
    request: AbortController
  ) {
    window.clearTimeout(this.timer);
    this.state = 'ready';
    this.retries = 0;
    this.notice = false;
    terminal.options.disableStdin = !this.viewVisible;
    this.message =
      action === 'create' ? `Connected as ${this.binding!.login}.` : `Reconnected as ${this.binding!.login}.`;
    this.publish('state', 'ready');
    void this.screenReady(n, terminal, screen);
    this.refreshAttachedMetadata(n, request);
  }
  private acceptOutput(frame: Record<string, unknown>, terminal: TerminalView, queued: {bytes: number}) {
    if (this.state !== 'ready' || typeof frame.data !== 'string') throw Error('frame');
    const decoded = atob(frame.data);
    if (!decoded.length || decoded.length > 4096 || queued.bytes + decoded.length > 262144) throw Error('output');
    queued.bytes += decoded.length;
    terminal.write(
      Uint8Array.from(decoded, (c) => c.charCodeAt(0)),
      () => {
        queued.bytes -= decoded.length;
      }
    );
    this.publish('output', 'ready');
  }
  private dispatchFrame(
    frame: Record<string, unknown>,
    n: number,
    terminal: TerminalView,
    screen: HTMLElement,
    action: 'attach' | 'create',
    request: AbortController,
    queued: {bytes: number}
  ) {
    const keys = Object.keys(frame).sort().join(',');
    if (frame.type === 'ready' && keys === 'type' && this.state === 'opening' && this.sessionID)
      this.acceptReady(n, terminal, screen, action, request);
    else if (frame.type === 'output' && keys === 'data,type') this.acceptOutput(frame, terminal, queued);
    else if (frame.type === 'closed' && keys === 'reason,type')
      this.detach(
        'Attachment ended or unavailable. Reconnect only the existing terminal; no replacement was launched.',
        false,
        'unavailable'
      );
    else throw Error('frame');
  }
  private onPeerMessage(
    event: MessageEvent,
    n: number,
    terminal: TerminalView,
    screen: HTMLElement,
    action: 'attach' | 'create',
    request: AbortController,
    queued: {bytes: number}
  ) {
    if (!this.live(n) || this.terminal !== terminal) return;
    try {
      if (typeof event.data !== 'string' || event.data.length > 32768) throw Error('frame');
      this.dispatchFrame(object(JSON.parse(event.data)), n, terminal, screen, action, request, queued);
    } catch {
      this.detach('Invalid or overloaded stream. No input or creation was replayed.');
    }
  }
  private onPeerClosed(n: number) {
    if (!this.live(n)) return;
    this.detach('Connection lost. No End or input replay was requested.');
    if (this.sessionID && this.retries < 3) {
      const wait = 1000 * 2 ** this.retries++;
      this.timer = window.setTimeout(() => this.connect(true), wait);
    }
  }
  private bindPeer(
    n: number,
    terminal: TerminalView,
    screen: HTMLElement,
    action: 'attach' | 'create',
    cols: number,
    rows: number,
    request: AbortController
  ) {
    const url = new URL(`/-/soda/api/environments/${this.binding!.environmentId}/terminal`, location.origin);
    url.protocol = 'wss:';
    const peer = (this.socket = new WebSocket(url)),
      queued = {bytes: 0};
    peer.onopen = () => this.onPeerOpen(n, peer, action, cols, rows);
    peer.onmessage = (event) => this.onPeerMessage(event, n, terminal, screen, action, request, queued);
    peer.onerror = peer.onclose = () => this.onPeerClosed(n);
  }
  private async attachPeer(n: number, request: AbortController, automatic: boolean) {
    if (automatic && !this.sessionID) {
      this.detach('Terminal absent. Nothing was created.');
      return;
    }
    const action = this.sessionID ? 'attach' : 'create';
    const {Terminal, FitAddon} = await this.loadRenderer();
    if (!this.live(n)) return;
    const screen = await this.awaitScreen(n);
    if (!screen || !this.live(n)) return;
    const terminal = this.openTerminal(n, screen, Terminal, FitAddon);
    const cols = Math.max(2, Math.min(500, terminal.cols)),
      rows = Math.max(2, Math.min(300, terminal.rows));
    if (action === 'create' && (await this.reserveCreate(n, request, cols, rows))) return;
    this.bindPeer(n, terminal, screen, action, cols, rows, request);
  }
  private async connect(automatic = false) {
    if (!this.canStartConnect(automatic)) return;
    window.clearTimeout(this.timer);
    this.state = 'opening';
    const n = ++this.generation,
      request = (this.request = new AbortController());
    this.timer = window.setTimeout(
      () => this.detach('Terminal connection timed out. Inspect the exact locator; creation was not retried.'),
      45000
    );
    this.notice = true;
    this.message = 'Inspecting the original terminal…';
    try {
      if (await this.inspectExisting(n, request)) return;
      await this.attachPeer(n, request, automatic);
    } catch {
      if (this.live(n))
        this.detach(
          'Could not authorize or attach. Sign in again or inspect the existing terminal; nothing was replayed.'
        );
    }
  }
  private shouldFocusScreen() {
    return (
      document.hasFocus() &&
      document.visibilityState !== 'hidden' &&
      this.viewVisible &&
      !this.closest('[hidden], [inert]') &&
      (document.activeElement === document.body || this.contains(document.activeElement))
    );
  }
  private async screenReady(n: number, terminal: TerminalView, screen: HTMLElement) {
    try {
      await this.updateComplete;
      if (!this.live(n) || this.terminal !== terminal) return;
      if (this.shouldFocusScreen()) terminal.focus();
      if (!this.live(n) || this.terminal !== terminal || !screen.isConnected) return;
      this.observer = new ResizeObserver(this.resize);
      this.observer.observe(screen);
      this.resize();
    } catch {
      if (this.live(n)) this.detach('Terminal rendering failed. No input or creation was replayed.');
    }
  }
  setVisible(visible: boolean) {
    this.viewVisible = visible;
    if (!visible) {
      this.confirming = undefined;
      this.closeMenu();
    }
    if (this.terminal) this.terminal.options.disableStdin = !visible || this.state !== 'ready';
    if (visible) this.resize(); // Presentation only: never connect or focus.
  }
  focus() {
    if (this.viewVisible && !this.closest('[hidden], [inert]') && this.state === 'ready') this.terminal?.focus();
  }
  private observe(metadata: TerminalMetadata) {
    if (metadata.id !== this.sessionID) throw Error('Terminal observation changed');
    this.setName(metadata.name);
    this.dispatchEvent(
      new CustomEvent('soda-terminal-metadata', {
        bubbles: true,
        detail: metadata,
      })
    );
  }
  setName(name: string) {
    if ([...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name)) this.sessionName = name;
  }
  open(name = '') {
    if ([...name].length > 80 || /[\p{Cc}\p{Cf}]/u.test(name)) return;
    this.createName = name;
    return this.connect();
  }
  get started() {
    return this.state !== 'idle';
  }
  restore() {
    if (this.sessionID) return this.connect(true);
  }
  disconnect() {
    this.detach('Detached. Native work is not ended by disconnection.');
  }
  dispose() {
    if (this.disposed) return;
    this.detach('Detached.', true);
    this.disposed = true;
    this.lifetime.abort();
    this.remove();
  }
}
customElements.define('soda-terminal', SodaTerminal);
function admitMountContext(root: HTMLElement, context: TerminalContext) {
  const {expectedUserId, repositoryId, environmentId, login} = context;
  if (
    root.ownerDocument !== document ||
    !identifier(expectedUserId) ||
    !identifier(repositoryId) ||
    !/^[A-Za-z0-9_-]{1,128}$/.test(context.csrfToken) ||
    !/^p[0-9a-f]{24}$/.test(environmentId) ||
    !/^[a-z][a-z0-9_-]{0,30}$/.test(login) ||
    login === 'root'
  )
    throw Error('Invalid terminal mounting context');
}
function admitLocator(locator: TerminalLocator) {
  if (
    !locator ||
    !['new', 'existing'].includes(locator.kind) ||
    (locator.kind === 'existing' && !terminalID(locator.id))
  )
    throw Error('Invalid terminal locator');
}
export function mountTerminal(
  root: HTMLElement,
  context: TerminalContext,
  locator: TerminalLocator,
  loadRenderer: () => Promise<Renderer> = renderer
) {
  admitMountContext(root, context);
  admitLocator(locator);
  const terminal = new SodaTerminal();
  terminal.configure(context, loadRenderer, locator);
  root.append(terminal);
  return {
    setVisible: (visible: boolean) => terminal.setVisible(visible),
    setName: (name: string) => terminal.setName(name),
    focus: () => terminal.focus(),
    open: (name?: string) => terminal.open(name),
    get started() {
      return terminal.started;
    },
    get ready() {
      return terminal.updateComplete;
    },
    restore: () => terminal.restore(),
    invalidate: () => terminal.invalidate(),
    disconnect: () => terminal.disconnect(),
    dispose: () => terminal.dispose(),
  };
}
