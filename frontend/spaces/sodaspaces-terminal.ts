import {LitElement} from 'lit';
import {renderTerminal} from './sodaspaces-terminal-view.js';
import type {ConnectionState, TerminalObservation} from './sodaspaces-attention.js';
import {object, readSodaJSON, id as identifier} from './sodaspaces-api.js';
import {terminalID, terminalResponse} from './sodaspaces-terminal-response.js';
import type {Terminal, ITerminalOptions, ITerminalInitOnlyOptions, ITerminalAddon} from '@xterm/xterm';
import type {FitAddon} from '@xterm/addon-fit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {cancelIfHidden, connect, send} from './sodaspaces-terminal-attachment.js';
import type {AttachmentInput} from './sodaspaces-terminal-attachment.js';
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
  transport: PreparedExtensionMount;
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
    void connect(this.attachmentInput());
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
  private attachmentInput(): AttachmentInput {
    return {
      readBinding: () => this.binding,
      readSessionID: () => this.sessionID,
      setSessionID: (id) => {
        this.sessionID = id;
      },
      readState: () => this.state,
      setState: (state) => {
        this.state = state;
      },
      setMessage: (message) => {
        this.message = message;
      },
      setNotice: (notice) => {
        this.notice = notice;
      },
      readRetries: () => this.retries,
      setRetries: (retries) => {
        this.retries = retries;
      },
      readCreateName: () => this.createName,
      setName: (name) => this.setName(name),
      remember: (value) => this.remember(value),
      publish: (kind, state) => this.publish(kind, state),
      isLive: (generation) => this.live(generation),
      detach: (message, stale, reason) => this.detach(message, stale, reason),
      dispatchMetadata: (metadata) => {
        this.dispatchEvent(
          new CustomEvent('soda-terminal-metadata', {
            bubbles: true,
            detail: metadata,
          })
        );
      },
      json: (path, body, signal) => this.json(path, body, signal),
      readTerminal: () => this.terminal,
      readSocket: () => this.socket,
      setSocket: (socket) => {
        this.socket = socket;
      },
      loadRenderer: () => this.loadRenderer(),
      awaitScreen: (n) => this.awaitScreen(n),
      openTerminal: (n, screen, Terminal, FitAddon) => this.openTerminal(n, screen, Terminal, FitAddon),
      screenReady: (n, terminal, screen) => {
        void this.screenReady(n, terminal, screen);
      },
      isViewVisible: () => this.viewVisible,
      isConnected: () => this.isConnected,
      isConcealed: () => !!this.closest('[hidden]'),
      readDisposed: () => this.disposed,
      readActionBusy: () => this.actionBusy,
      readManagedEnded: () => this.managedEnded,
      beginGeneration: () => ++this.generation,
      takeRequest: () => (this.request = new AbortController()),
      clearTimer: () => {
        window.clearTimeout(this.timer);
      },
      setTimer: (timer) => {
        this.timer = timer;
      },
    };
  }
  private async json(path: string, body: Record<string, unknown> | null, signal: AbortSignal) {
    if (!this.binding) throw Error('Missing terminal binding');
    const headers: Record<string, string> = {};
    if (body) {
      headers['Content-Type'] = 'application/json';
    }
    const response = await this.binding.transport.request(path.slice('/api/'.length), {
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
      send(this.attachmentInput(), {
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
  private async awaitScreen(n: number) {
    this.screenVisible = true;
    await this.updateComplete;
    if (!this.live(n) || cancelIfHidden(this.attachmentInput())) return;
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!screen) throw Error('Missing terminal screen');
    await document.fonts.ready;
    if (!this.live(n) || cancelIfHidden(this.attachmentInput())) return;
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
        !send(this.attachmentInput(), {
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
  setName(name: string) {
    if ([...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name)) this.sessionName = name;
  }
  open(name = '') {
    if ([...name].length > 80 || /[\p{Cc}\p{Cf}]/u.test(name)) return;
    this.createName = name;
    return connect(this.attachmentInput());
  }
  get started() {
    return this.state !== 'idle';
  }
  restore() {
    if (this.sessionID) return connect(this.attachmentInput(), true);
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
