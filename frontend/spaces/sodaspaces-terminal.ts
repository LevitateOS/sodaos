import {LitElement} from 'lit';
import {renderTerminal} from './sodaspaces-terminal-view.js';
import type {ConnectionState, TerminalObservation} from './sodaspaces-attention.js';
import {object, readSodaJSON, terminalID, terminalResponse, id as identifier} from './sodaspaces-api.js';
import type {Terminal, ITerminalOptions, ITerminalInitOnlyOptions, ITerminalAddon} from '@xterm/xterm';
import type {FitAddon} from '@xterm/addon-fit';
import type {TerminalMetadata} from './sodaspaces-api.js';
export type TerminalView = Pick<Terminal, 'cols' | 'rows' | 'options' | 'parser' | 'loadAddon' | 'attachCustomKeyEventHandler' | 'onData' | 'open' | 'focus' | 'dispose' | 'write'>;
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
export type TerminalLocator = {
  kind: 'new';
} | {
  kind: 'existing';
  id: string;
};
const renderer = async (): Promise<Renderer> => {
  const [{Terminal}, {FitAddon}] = await Promise.all([import('./soda-terminal/xterm.mjs'), import('./soda-terminal/addon-fit.mjs')]);
  return {
    Terminal, FitAddon
  };
};
// One immutable original account and imperative attachment owner. Rendering never
// creates a shell, opens a socket, or touches xterm's screen descendants.
export class SodaTerminal extends LitElement {
  static properties = {
    state: {
      state: true
    }, sessionID: {
      state: true
    }, message: {
      state: true
    }, screenVisible: {
      state: true
    }, actionBusy: {
      state: true
    }, confirming: {
      state: true
    }, sessionName: {
      state: true
    }, notice: {
      state: true
    }
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
  private minimumSize: {
    width: number;
    height: number;
  } | undefined;
  private geometryFrame: number | undefined;
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
    if (this.binding || this.disposed)
      throw Error('Terminal binding is immutable');
    this.binding = {
      ...context
    };
    this.loadRenderer = load;
    if (locator.kind === 'existing')
      this.sessionID = locator.id;
    document.fonts.addEventListener('loadingdone', this.resize, {
      signal: this.lifetime.signal
    });
    window.visualViewport?.addEventListener('resize', this.resize, {
      signal: this.lifetime.signal
    });
    window.addEventListener('pagehide', () => this.invalidate(), {
      signal: this.lifetime.signal
    });
    window.addEventListener('pageshow', event => {
      if (event.persisted)
        this.invalidate();
    }, {
      signal: this.lifetime.signal
    });
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.dispose();
  }
  protected render() {
    const disabled = this.disposed || this.state === 'stale';
    return renderTerminal({
      ready: this.state === 'ready', disabled,
      canConnect: !(disabled || this.managedEnded || this.state === 'opening' || this.state === 'ready' || this.actionBusy),
      canEnd: !disabled && !!this.sessionID && !this.actionBusy,
      connectLabel: this.sessionID ? 'Reconnect terminal' : 'Open terminal',
      name: this.sessionName || 'Terminal', login: this.binding?.login || '', project: this.binding?.projectName || this.binding?.environmentId || '',
      message: this.message, notice: this.notice, screenVisible: this.screenVisible,
      confirmingName: this.confirming ? this.sessionName || this.confirming : null,
      canConfirm: !disabled && !this.actionBusy && this.confirming === this.sessionID,
    }, {
      connect: () => this.connectFromControls(),
      end: () => this.confirmEnd(),
      confirmEnd: () => this.endConfirmedTerminal(), cancelEnd: () => this.cancelEnd(),
      project: () => this.workspaceCommand('project'), rename: () => this.workspaceCommand('rename'), hide: () => this.workspaceCommand('hide'),
      menuKey: event => {
        if (event.key === 'Escape') {
          event.preventDefault();
          event.stopPropagation();
          this.closeMenu();
          this.querySelector<HTMLElement>('summary')?.focus();
        }
      },
    });
  }
  private connectFromControls() {
    this.closeMenu();
    this.retries = 0;
    void this.connect();
  }
  private endConfirmedTerminal() {
    // Admission must see the exact confirmed ID before the dialog is cleared.
    if (this.confirming === this.sessionID && !this.closest('[hidden], [inert]'))
      void this.endTerminal();
    this.confirming = undefined;
  }
  private closeMenu() {
    const menu = this.querySelector('details');
    if (menu)
      menu.open = false;
  }
  private workspaceCommand(command: 'project' | 'rename' | 'hide') {
    if (this.disposed || this.state === 'stale' || this.closest('[hidden], [inert]'))
      return;
    this.closeMenu();
    this.dispatchEvent(new CustomEvent('soda-terminal-command', {
      bubbles: true, detail: command
    }));
  }
  private confirmEnd() {
    if (!this.sessionID || this.disposed || this.state === 'stale' || this.actionBusy || this.closest('[hidden], [inert]'))
      return;
    this.closeMenu();
    this.confirming = this.sessionID;
    const target = this.confirming, active = document.activeElement;
    void this.updateComplete.then(() => {
      if (this.confirming === target && !this.disposed && (document.activeElement === active || document.activeElement === document.body))
        this.querySelector<HTMLElement>('[data-action=cancel-end]')?.focus();
    });
  }
  private cancelEnd() {
    this.confirming = undefined;
    this.querySelector<HTMLElement>('[data-action=controls]')?.focus();
  }
  private remember(value: string | null) {
    if (value === null)
      this.managedEnded = true;
    this.dispatchEvent(new CustomEvent('soda-terminal-locator', {
      bubbles: true, detail: value
    }));
  }
  private live(generation: number) {
    return !this.disposed && this.state !== 'stale' && this.generation === generation;
  }
  private publish(kind: TerminalObservation['kind'], state: ConnectionState) {
    if (this.disposed)
      return;
    const detail: TerminalObservation = {
      kind, state, generation: this.generation, id: this.sessionID || null
    };
    this.dispatchEvent(new CustomEvent<TerminalObservation>('soda-terminal-observation', {
      bubbles: true, detail
    }));
  }
  private detach(message: string, stale = false, reason: ConnectionState = 'connection-lost') {
    ++this.generation;
    this.state = stale ? 'stale' : 'closed';
    this.notice = true;
    this.confirming = undefined;
    window.clearTimeout(this.timer);
    this.request?.abort();
    this.request = undefined;
    if (stale) {
      this.actionRequest?.abort();
      this.actionRequest = undefined;
      this.actionBusy = false;
    }
    this.observer?.disconnect();
    this.observer = undefined;
    if (this.geometryFrame !== undefined)
      cancelAnimationFrame(this.geometryFrame);
    this.geometryFrame = undefined;
    const old = this.socket;
    this.socket = undefined;
    if (old && old.readyState < 2)
      old.close();
    this.terminal?.dispose();
    this.terminal = undefined;
    this.fit = undefined;
    this.lastSize = '';
    this.querySelector('.soda-terminal-screen')?.replaceChildren();
    this.screenVisible = false;
    this.message = message;
    this.publish('state', reason);
  }
  invalidate() {
    this.detach('Page context changed. No action was replayed; reconnect through a fresh authorized page.', true);
  }
  private async json(path: string, body: Record<string, unknown> | null, signal: AbortSignal) {
    if (!this.binding)
      throw Error('Missing terminal binding');
    const headers: Record<string, string> = {
      'X-Soda-Expected-User-ID': this.binding.expectedUserId
    };
    if (body) {
      headers['Content-Type'] = 'application/json';
      headers['X-CSRF-Token'] = this.binding.csrfToken;
    }
    const response = await fetch('/-/soda' + path, {
      method: body ? 'POST' : 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, ...(body ? {
        body: JSON.stringify(body)
      } : {}), signal
    });
    if (!response.ok) {
      if (response.status === 401 || response.status === 403)
        this.authorityLost();
      throw Error('Soda request refused');
    }
    return object(await readSodaJSON(response));
  }
  private authorityLost() {
    if (!this.disposed)
      this.dispatchEvent(new CustomEvent('soda-terminal-authority-lost', {
        bubbles: true
      }));
  }
  private async endTerminal() {
    if (this.disposed || this.state === 'stale' || !this.sessionID || !this.binding || this.actionBusy)
      return;
    const target = this.sessionID, control = this.actionRequest = new AbortController();
    const current = () => !this.disposed && this.state !== 'stale' && this.actionRequest === control && this.sessionID === target;
    this.actionBusy = true;
    const timeout = window.setTimeout(() => control.abort(), 30000);
    try {
      const result = await this.json(`/api/environments/${this.binding.environmentId}/terminal-sessions/${target}`, {action: 'end'}, control.signal);
      if (!current())
        return;
      const observed = terminalResponse(result, this.binding);
      if (observed && (observed.id !== target || observed.state !== 'ended'))
        throw Error('outcome');
      this.detach('Native cleanup confirmed. Files and independent services remain.', false, 'ended');
      this.sessionID = undefined;
      this.remember(null);
    }
    catch {
      if (current()) {
        this.notice = true;
        this.message = 'End was not confirmed. Inspect this exact terminal; no retry or replacement was made.';
        this.publish('state', 'unconfirmed');
      }
    }
    finally {
      window.clearTimeout(timeout);
      if (this.actionRequest === control) {
        this.actionRequest = undefined;
        this.actionBusy = false;
      }
    }
  }
  private send(frame: {
    type: 'resize';
    cols: number;
    rows: number;
  } | {
    type: 'input';
    data: string;
  }) {
    if (this.state !== 'ready' || !this.socket || this.socket.readyState !== 1 || this.socket.bufferedAmount > 65536) {
      this.detach('Connection lost or overloaded. Reconnect the existing terminal; input was not replayed.');
      return false;
    }
    try {
      this.socket.send(JSON.stringify(frame));
      return true;
    }
    catch {
      this.detach('Connection lost. No input was replayed.');
      return false;
    }
  }
  private resize = () => {
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!this.terminal || !this.fit || !this.viewVisible || this.closest('[hidden]') || this.state !== 'ready' || !screen?.isConnected || !screen.clientWidth || !screen.clientHeight)
      return;
    this.fit.fit();
    const {cols, rows} = this.terminal, fitted = cols + ':' + rows;
    if (cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300 && this.lastSize !== fitted) {
      this.lastSize = fitted;
      this.send({
        type: 'resize', cols, rows
      });
    }
    // Xterm changes cols/rows before its rendered grid catches up. Measuring in
    // this same callback would divide the old grid by the new column count.
    if (this.geometryFrame === undefined)
      this.geometryFrame = requestAnimationFrame(() => {
        this.geometryFrame = undefined;
        this.measureMinimum();
      });
  };
  private measureMinimum() {
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!this.terminal || this.state !== 'ready' || !this.viewVisible || !screen?.isConnected || this.closest('[hidden]'))
      return;
    const {cols, rows} = this.terminal;
    // Read only geometry of the locked xterm-owned descendants.
    const grid = screen.querySelector<HTMLElement>('.xterm-screen')?.getBoundingClientRect();
    if (grid?.width && grid.height && cols && rows) {
      const css = getComputedStyle(screen), scrollbar = screen.querySelector<HTMLElement>('.scrollbar.vertical')?.getBoundingClientRect().width || 0;
      const minimum = {
        width: Math.ceil(grid.width / cols * 56 + scrollbar + parseFloat(css.paddingLeft) + parseFloat(css.paddingRight)) + 2,
        height: Math.ceil(grid.height / rows * 12 + this.offsetHeight - screen.offsetHeight + parseFloat(css.paddingTop) + parseFloat(css.paddingBottom)) + 2
      };
      if (minimum.width > 0 && minimum.height > 0 && (minimum.width !== this.minimumSize?.width || minimum.height !== this.minimumSize?.height)) {
        this.minimumSize = minimum;
        this.dispatchEvent(new CustomEvent('soda-terminal-geometry', {
          bubbles: true, detail: minimum
        }));
      }
    }
  }
  private async connect(automatic = false) {
    if (this.disposed || !this.binding || this.state === 'stale' || this.state === 'opening' || this.state === 'ready' || this.actionBusy)
      return;
    if (this.managedEnded) {
      this.message = 'This exact session ended. Use New terminal for a different shell.';
      return;
    }
    if (automatic && !this.sessionID)
      return;
    if (!this.viewVisible || this.closest('[hidden]') || (!automatic && (document.visibilityState === 'hidden' || !document.hasFocus())))
      return;
    const {expectedUserId, repositoryId, environmentId, login} = this.binding;
    window.clearTimeout(this.timer);
    this.state = 'opening';
    const n = ++this.generation, request = this.request = new AbortController();
    this.timer = window.setTimeout(() => this.detach('Terminal connection timed out. Inspect the exact locator; creation was not retried.'), 45000);
    this.notice = true;
    this.message = 'Inspecting the original terminal…';
    try {
      if (this.sessionID) {
        const metadata = await this.json(`/api/environments/${environmentId}/terminal-sessions/${this.sessionID}`, null, request.signal);
        if (!this.live(n))
          return;
        const existing = terminalResponse(metadata, this.binding);
        if (existing && existing.id !== this.sessionID)
          throw Error('terminal metadata');
        if (!existing || existing.state === 'ended') {
          this.sessionID = undefined;
          this.remember(null);
          this.detach('Native terminal is absent or ended. Use New terminal for a different shell.', false, 'ended');
          return;
        }
        this.observe(existing);
        if (!existing.ready) {
          this.detach('Native transition is still in progress. No replacement was made.', false, existing.state === 'ending' ? 'ending' : 'unavailable');
          return;
        }
        if (existing.attached) {
          this.detach('An existing writer is attached. No takeover or replacement was requested.', false, 'attached-elsewhere');
          return;
        }
      }
      if (automatic && !this.sessionID) {
        this.detach('Terminal absent. Nothing was created.');
        return;
      }
      const action = this.sessionID ? 'attach' : 'create';
      const {Terminal, FitAddon} = await this.loadRenderer();
      if (!this.live(n))
        return;
      this.screenVisible = true;
      await this.updateComplete;
      if (!this.live(n))
        return;
      if (!this.viewVisible || !this.isConnected || this.closest('[hidden]')) {
        this.detach('Attachment cancelled while hidden. No creation was sent.');
        return;
      }
      const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
      if (!screen)
        throw Error('Missing terminal screen');
      await document.fonts.ready;
      if (!this.live(n))
        return;
      if (!this.viewVisible || !this.isConnected || this.closest('[hidden]')) {
        this.detach('Attachment cancelled while hidden. No creation was sent.');
        return;
      }
      const styles = getComputedStyle(this);
      const token = (name: string) => {
        const value = styles.getPropertyValue(name).trim();
        if (!value)
          throw Error(`Missing terminal token ${name}`);
        return value;
      };
      const fontSize = Number.parseFloat(token('--soda-font-mono-size')), lineHeight = Number(token('--soda-font-mono-line'));
      if (!Number.isFinite(fontSize) || fontSize <= 0 || !Number.isFinite(lineHeight) || lineHeight < 1)
        throw Error('Invalid terminal typography');
      const terminal = this.terminal = new Terminal({
        allowProposedApi: true, disableStdin: true, scrollback: 1000, windowOptions: {}, convertEol: false, cols: 80, rows: 24,
        fontFamily: token('--soda-font-mono-family'), fontSize, lineHeight,
        theme: {
          background: token('--soda-terminal-screen'), foreground: token('--soda-terminal-text'), cursor: token('--soda-terminal-text')
        }
      });
      const fit = this.fit = new FitAddon();
      terminal.loadAddon(fit);
      for (const code of [0, 1, 2, 8, 52])
        terminal.parser.registerOscHandler(code, () => true);
      terminal.attachCustomKeyEventHandler(event => {
        if (event.ctrlKey && event.shiftKey && event.key === 'Enter') {
          event.preventDefault();
          event.stopPropagation();
          this.querySelector<HTMLElement>('[data-action=controls]')?.focus();
          return false;
        }
        return true;
      });
      terminal.onData(data => {
        if (!this.live(n) || !this.viewVisible || this.closest('[hidden]') || !screen.contains(document.activeElement))
          return;
        if (data.length > 65536) {
          this.detach('Input too large. Nothing was replayed.');
          return;
        }
        const bytes = new TextEncoder().encode(data);
        for (let i = 0; i < bytes.length; i += 16384)
          if (!this.send({
            type: 'input', data: btoa(String.fromCharCode(...bytes.subarray(i, i + 16384)))
          }))
            break;
      });
      terminal.open(screen);
      if (screen.clientWidth && screen.clientHeight)
        fit.fit();
      const cols = Math.max(2, Math.min(500, terminal.cols)), rows = Math.max(2, Math.min(300, terminal.rows));
      if (action === 'create') {
        const reserved = await this.json(`/api/environments/${environmentId}/terminal-sessions`, {cols, rows, name: this.createName}, request.signal);
        if (!this.live(n))
          return;
        if (!terminalID(reserved.id))
          throw Error('terminal reservation');
        this.sessionID = reserved.id;
        this.setName(this.createName);
        this.remember(reserved.id); // published BEFORE any native Create can be sent
        if (!this.live(n))
          return;
      }
      const url = new URL(`/-/soda/api/environments/${environmentId}/terminal`, location.origin);
      url.protocol = 'wss:';
      const peer = this.socket = new WebSocket(url);
      peer.onopen = () => {
        if (!this.live(n)) {
          peer.close();
          return;
        }
        if (!this.viewVisible || this.closest('[hidden]')) {
          this.detach('Attachment cancelled while hidden. No creation was sent.');
          return;
        }
        this.publish('state', 'opening');
        try {
          peer.send(JSON.stringify({
            action, id: this.sessionID, ...(action === 'create' ? {name: this.createName} : {}),
            expected_user_id: expectedUserId, repository_id: repositoryId, csrf_token: this.binding?.csrfToken, cols, rows
          }));
          this.message = action === 'create' ? 'Opening terminal…' : 'Attaching the existing terminal…';
        }
        catch {
          this.detach('Attachment dispatch was not confirmed. No creation or input was retried.');
        }
      };
      let queuedOutput = 0;
      peer.onmessage = event => {
        if (!this.live(n) || this.terminal !== terminal)
          return;
        try {
          if (typeof event.data !== 'string' || event.data.length > 32768)
            throw Error('frame');
          const frame = object(JSON.parse(event.data)), keys = Object.keys(frame).sort().join(',');
          if (frame.type === 'ready' && keys === 'type' && this.state === 'opening' && this.sessionID) {
            window.clearTimeout(this.timer);
            this.state = 'ready';
            this.retries = 0;
            this.notice = false;
            terminal.options.disableStdin = !this.viewVisible;
            this.message = action === 'create' ? `Connected as ${login}.` : `Reconnected as ${login}.`;
            this.publish('state', 'ready');
            void this.screenReady(n, terminal, screen);
            const target = this.sessionID;
            void this.json(`/api/environments/${environmentId}/terminal-sessions/${target}`, null, AbortSignal.any([request.signal, AbortSignal.timeout(10000)])).then(result => {
              if (!this.live(n) || this.sessionID !== target || !this.binding)
                return;
              const metadata = terminalResponse(result, this.binding);
              if (metadata?.id === target)
                this.observe(metadata);
            }).catch(() => {
            });
          }
          else if (frame.type === 'output' && this.state === 'ready' && keys === 'data,type' && typeof frame.data === 'string') {
            const decoded = atob(frame.data);
            if (!decoded.length || decoded.length > 4096 || queuedOutput + decoded.length > 262144)
              throw Error('output');
            queuedOutput += decoded.length;
            terminal.write(Uint8Array.from(decoded, c => c.charCodeAt(0)), () => {
              queuedOutput -= decoded.length;
            });
            this.publish('output', 'ready');
          }
          else if (frame.type === 'closed' && keys === 'reason,type')
            this.detach('Attachment ended or unavailable. Reconnect only the existing terminal; no replacement was launched.', false, 'unavailable');
          else
            throw Error('frame');
        }
        catch {
          this.detach('Invalid or overloaded stream. No input or creation was replayed.');
        }
      };
      peer.onerror = peer.onclose = () => {
        if (!this.live(n))
          return;
        this.detach('Connection lost. No End or input replay was requested.');
        if (this.sessionID && this.retries < 3) {
          const wait = 1000 * 2 ** this.retries++;
          this.timer = window.setTimeout(() => this.connect(true), wait);
        }
      };
    }
    catch {
      if (this.live(n))
        this.detach('Could not authorize or attach. Sign in again or inspect the existing terminal; nothing was replayed.');
    }
  }
  private async screenReady(n: number, terminal: TerminalView, screen: HTMLElement) {
    try {
      await this.updateComplete;
      if (!this.live(n) || this.terminal !== terminal)
        return;
      if (document.hasFocus() && document.visibilityState !== 'hidden' && this.viewVisible && !this.closest('[hidden], [inert]') && (document.activeElement === document.body || this.contains(document.activeElement)))
        terminal.focus();
      // Native focus handlers can synchronously retire this component.
      if (!this.live(n) || this.terminal !== terminal || !screen.isConnected)
        return;
      this.observer = new ResizeObserver(this.resize);
      this.observer.observe(screen);
      this.resize();
    }
    catch {
      if (this.live(n))
        this.detach('Terminal rendering failed. No input or creation was replayed.');
    }
  }
  setVisible(visible: boolean) {
    this.viewVisible = visible;
    if (!visible) {
      this.confirming = undefined;
      this.closeMenu();
    }
    if (this.terminal)
      this.terminal.options.disableStdin = !visible || this.state !== 'ready';
    if (visible)
      this.resize(); // Presentation only: never connect or focus.
  }
  focus() {
    if (this.viewVisible && !this.closest('[hidden], [inert]') && this.state === 'ready')
      this.terminal?.focus();
  }
  private observe(metadata: TerminalMetadata) {
    if (metadata.id !== this.sessionID)
      throw Error('Terminal observation changed');
    this.setName(metadata.name);
    this.dispatchEvent(new CustomEvent('soda-terminal-metadata', {
      bubbles: true, detail: metadata
    }));
  }
  setName(name: string) {
    if ([...name].length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(name))
      this.sessionName = name;
  }
  open(name = '') {
    if ([...name].length > 80 || /[\p{Cc}\p{Cf}]/u.test(name))
      return;
    this.createName = name;
    return this.connect();
  }
  get started() {
    return this.state !== 'idle';
  }
  restore() {
    if (this.sessionID)
      return this.connect(true);
  }
  disconnect() {
    this.detach('Detached. Native work is not ended by disconnection.');
  }
  dispose() {
    if (this.disposed)
      return;
    this.detach('Detached.', true);
    this.disposed = true;
    this.lifetime.abort();
    this.remove();
  }
}
customElements.define('soda-terminal', SodaTerminal);
export function mountTerminal(root: HTMLElement, context: TerminalContext, locator: TerminalLocator, loadRenderer: () => Promise<Renderer> = renderer) {
  const {expectedUserId, repositoryId, environmentId, login} = context;
  if (root.ownerDocument !== document || !identifier(expectedUserId) || !identifier(repositoryId) || !/^[A-Za-z0-9_-]{1,128}$/.test(context.csrfToken) || !/^p[0-9a-f]{24}$/.test(environmentId) || !/^[a-z][a-z0-9_-]{0,30}$/.test(login) || login === 'root')
    throw Error('Invalid terminal mounting context');
  if (!locator || !['new', 'existing'].includes(locator.kind) || locator.kind === 'existing' && !terminalID(locator.id))
    throw Error('Invalid terminal locator');
  const terminal = new SodaTerminal();
  terminal.configure(context, loadRenderer, locator);
  root.append(terminal);
  return {
    setVisible: (visible: boolean) => terminal.setVisible(visible), setName: (name: string) => terminal.setName(name), focus: () => terminal.focus(), open: (name?: string) => terminal.open(name), get started() {
      return terminal.started;
    }, get ready() {
      return terminal.updateComplete;
    }, restore: () => terminal.restore(),
    invalidate: () => terminal.invalidate(), disconnect: () => terminal.disconnect(), dispose: () => terminal.dispose()
  };
}
