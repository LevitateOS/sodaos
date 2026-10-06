import {LitElement} from 'lit';
import {renderTerminal} from './sodaspaces-terminal-view.js';
import {id as identifier} from './sodaspaces-api.js';
import {terminalID} from './sodaspaces-terminal-response.js';
import type {Terminal, ITerminalOptions, ITerminalInitOnlyOptions, ITerminalAddon} from '@xterm/xterm';
import type {FitAddon} from '@xterm/addon-fit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {connect} from './sodaspaces-terminal-attachment.js';
import type {AttachmentInput} from './sodaspaces-terminal-attachment.js';
import {awaitScreen, clearScreen, openTerminal, resize, screenReady} from './sodaspaces-terminal-screen.js';
import type {ScreenInput} from './sodaspaces-terminal-screen.js';
import {detach, endTerminal, json, live, publish} from './sodaspaces-terminal-actions.js';
import type {ActionsInput} from './sodaspaces-terminal-actions.js';
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
    document.fonts.addEventListener('loadingdone', () => resize(this.screenInput()), {
      signal: this.lifetime.signal,
    });
    window.visualViewport?.addEventListener('resize', () => resize(this.screenInput()), {
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
    if (this.confirming === this.sessionID && !this.closest('[hidden], [inert]')) void endTerminal(this.actionsInput());
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
  invalidate() {
    detach(
      this.actionsInput(),
      'Page context changed. No action was replayed; reconnect through a fresh authorized page.',
      true
    );
  }
  private actionsInput(): ActionsInput {
    return {
      readBinding: () => this.binding,
      readDisposed: () => this.disposed,
      readState: () => this.state,
      setState: (state) => {
        this.state = state;
      },
      readGeneration: () => this.generation,
      bumpGeneration: () => {
        ++this.generation;
      },
      readSessionID: () => this.sessionID,
      setSessionID: (id) => {
        this.sessionID = id;
      },
      setMessage: (message) => {
        this.message = message;
      },
      setNotice: (notice) => {
        this.notice = notice;
      },
      setConfirming: (confirming) => {
        this.confirming = confirming;
      },
      readActionBusy: () => this.actionBusy,
      setActionBusy: (busy) => {
        this.actionBusy = busy;
      },
      readActionRequest: () => this.actionRequest,
      takeActionRequest: () => (this.actionRequest = new AbortController()),
      clearActionRequest: () => {
        this.actionRequest = undefined;
      },
      clearTimer: () => {
        window.clearTimeout(this.timer);
      },
      abortRequest: () => {
        this.request?.abort();
        this.request = undefined;
      },
      disconnectObserver: () => {
        this.observer?.disconnect();
        this.observer = undefined;
      },
      readSocket: () => this.socket,
      clearSocket: () => {
        this.socket = undefined;
      },
      clearScreen: () => {
        clearScreen(this.screenInput());
      },
      remember: (value) => this.remember(value),
      dispatchObservation: (detail) => {
        this.dispatchEvent(
          new CustomEvent('soda-terminal-observation', {
            bubbles: true,
            detail,
          })
        );
      },
      dispatchAuthorityLost: () => {
        this.dispatchEvent(
          new CustomEvent('soda-terminal-authority-lost', {
            bubbles: true,
          })
        );
      },
    };
  }
  private screenInput(): ScreenInput {
    return {
      attachment: this.attachmentInput(),
      readTerminal: () => this.terminal,
      setTerminal: (terminal) => {
        this.terminal = terminal;
      },
      readFit: () => this.fit,
      setFit: (fit) => {
        this.fit = fit;
      },
      readState: () => this.state,
      isViewVisible: () => this.viewVisible,
      isHidden: () => !!this.closest('[hidden]'),
      isHiddenOrInert: () => !!this.closest('[hidden], [inert]'),
      contains: (element) => this.contains(element),
      queryScreen: () => this.querySelector<HTMLElement>('.soda-terminal-screen'),
      focusControls: () => {
        this.querySelector<HTMLElement>('[data-action=controls]')?.focus();
      },
      readHostHeight: () => this.offsetHeight,
      readLastSize: () => this.lastSize,
      setLastSize: (size) => {
        this.lastSize = size;
      },
      setScreenVisible: (visible) => {
        this.screenVisible = visible;
      },
      readMinimumSize: () => this.minimumSize,
      setMinimumSize: (minimum) => {
        this.minimumSize = minimum;
      },
      dispatchGeometry: (minimum) => {
        this.dispatchEvent(
          new CustomEvent('soda-terminal-geometry', {
            bubbles: true,
            detail: minimum,
          })
        );
      },
      setObserver: (observer) => {
        this.observer = observer;
      },
      updateComplete: () => this.updateComplete,
      isLive: (generation) => live(this.actionsInput(), generation),
      detach: (message, stale, reason) => detach(this.actionsInput(), message, stale, reason),
      hostStyles: () => getComputedStyle(this),
    };
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
      publish: (kind, state) => publish(this.actionsInput(), kind, state),
      isLive: (generation) => live(this.actionsInput(), generation),
      detach: (message, stale, reason) => detach(this.actionsInput(), message, stale, reason),
      dispatchMetadata: (metadata) => {
        this.dispatchEvent(
          new CustomEvent('soda-terminal-metadata', {
            bubbles: true,
            detail: metadata,
          })
        );
      },
      json: (path, body, signal) => json(this.actionsInput(), path, body, signal),
      readTerminal: () => this.terminal,
      readSocket: () => this.socket,
      setSocket: (socket) => {
        this.socket = socket;
      },
      loadRenderer: () => this.loadRenderer(),
      awaitScreen: (n) => awaitScreen(this.screenInput(), n),
      openTerminal: (n, screen, Terminal, FitAddon) => openTerminal(this.screenInput(), n, screen, Terminal, FitAddon),
      screenReady: (n, terminal, screen) => {
        void screenReady(this.screenInput(), n, terminal, screen);
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
  setVisible(visible: boolean) {
    this.viewVisible = visible;
    if (!visible) {
      this.confirming = undefined;
      this.closeMenu();
    }
    if (this.terminal) this.terminal.options.disableStdin = !visible || this.state !== 'ready';
    if (visible) resize(this.screenInput()); // Presentation only: never connect or focus.
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
    detach(this.actionsInput(), 'Detached. Native work is not ended by disconnection.');
  }
  dispose() {
    if (this.disposed) return;
    detach(this.actionsInput(), 'Detached.', true);
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
