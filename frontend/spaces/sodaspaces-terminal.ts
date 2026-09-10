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
} | {
  kind: 'pending';
  requestId: string;
};
const renderer = async (): Promise<Renderer> => {
  const [{Terminal}, {FitAddon}] = await Promise.all([import('./soda-terminal/xterm.mjs'), import('./soda-terminal/addon-fit.mjs')]);
  return {
    Terminal, FitAddon
  };
};
// One immutable original account and imperative attachment owner. Rendering never
// creates a shell, opens a socket, Returns, or touches xterm's screen descendants.
export class SodaTerminal extends LitElement {
  static properties = {
    state: {
      state: true
    }, sessionID: {
      state: true
    }, uncertainCreate: {
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
    }, retainUntil: {
      state: true
    }
  };
  declare private state: 'idle' | 'opening' | 'ready' | 'closed' | 'stale';
  declare private sessionID: string | undefined;
  declare private uncertainCreate: boolean;
  declare private message: string;
  declare private screenVisible: boolean;
  declare private actionBusy: boolean;
  declare private confirming: string | undefined;
  declare private sessionName: string;
  declare private notice: boolean;
  declare private retainUntil: number | undefined;
  private createName = '';
  private managed = false;
  private managedEnded = false;
  private binding: TerminalContext | undefined;
  private loadRenderer: () => Promise<Renderer> = renderer;
  private storageKey = '';
  private requestID: string | undefined;
  private attachmentID: string | undefined;
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
    this.retainUntil = undefined;
    this.confirming = undefined;
    this.sessionName = '';
    this.notice = true;
    this.state = 'idle';
    this.sessionID = undefined;
    this.uncertainCreate = this.screenVisible = this.actionBusy = false;
    this.message = 'Not connected.';
  }
  protected createRenderRoot() {
    return this;
  }
  configure(context: TerminalContext, load: () => Promise<Renderer>, locator?: TerminalLocator) {
    if (this.binding || this.disposed)
      throw Error('Terminal binding is immutable');
    this.binding = {
      ...context
    };
    this.loadRenderer = load;
    this.storageKey = `soda-terminal:${context.expectedUserId}:${context.environmentId}`;
    this.managed = locator !== undefined;
    if (locator?.kind === 'existing')
      this.sessionID = locator.id;
    if (locator?.kind === 'pending') {
      this.requestID = locator.requestId;
      this.uncertainCreate = true;
    }
    try {
      const saved = this.managed ? null : sessionStorage.getItem(this.storageKey);
      if (saved?.startsWith('pending:') && terminalID(saved.slice(8))) {
        this.requestID = saved.slice(8);
        this.uncertainCreate = true;
      }
      else if (terminalID(saved))
        this.sessionID = saved;
      else if (saved === 'pending')
        this.uncertainCreate = true; // Old ambiguity must never select another session.
    }
    catch { /* locator only */
    }
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
      managed: this.managed, ready: this.state === 'ready', disabled, busy: this.actionBusy,
      canConnect: !(disabled || this.managedEnded || this.state === 'opening' || this.state === 'ready' || this.actionBusy),
      canEnd: !disabled && !!this.sessionID && !this.actionBusy,
      canReturn: !disabled && !!this.sessionID && !this.actionBusy && (!this.managed || !!this.retainUntil),
      connectLabel: this.sessionID ? 'Reconnect terminal' : this.uncertainCreate ? 'Find pending terminal' : 'Open terminal',
      login: this.binding?.login || '', project: this.binding?.projectName || this.binding?.environmentId || '',
      message: this.message, notice: this.notice, screenVisible: this.screenVisible,
      confirmingName: this.confirming ? this.sessionName || this.confirming : null,
      canConfirm: !disabled && !this.actionBusy && this.confirming === this.sessionID,
    }, {
      connect: () => {
        this.closeMenu();
        this.retries = 0;
        void this.connect();
      },
      end: () => this.managed ? this.confirmEnd() : this.control('end'),
      confirmEnd: () => {
        this.control('end');
        this.confirming = undefined;
      }, cancelEnd: () => this.cancelEnd(),
      return: () => {
        this.closeMenu();
        this.control('return');
      }, keep: () => {
        this.closeMenu();
        this.control('retain', 7200);
      },
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
    if (this.managed) {
      if (value === null)
        this.managedEnded = true;
      this.dispatchEvent(new CustomEvent('soda-terminal-locator', {
        bubbles: true, detail: value
      }));
      return;
    }
    try {
      if (value)
        sessionStorage.setItem(this.storageKey, value);
      else
        sessionStorage.removeItem(this.storageKey);
    }
    catch { /* no credentials or transcript */
    }
  }
  private live(generation: number) {
    return !this.disposed && this.state !== 'stale' && this.generation === generation;
  }
  private publish(kind: TerminalObservation['kind'], state: ConnectionState) {
    if (!this.managed || this.disposed)
      return;
    const detail: TerminalObservation = {
      kind, state, generation: this.generation, id: this.sessionID || null, requestId: this.requestID || null
    };
    this.dispatchEvent(new CustomEvent<TerminalObservation>('soda-terminal-observation', {
      bubbles: true, detail
    }));
  }
  private detach(message: string, stale = false, reason: ConnectionState = this.uncertainCreate ? 'unconfirmed' : 'connection-lost') {
    ++this.generation;
    this.state = stale ? 'stale' : 'closed';
    this.notice = true;
    this.confirming = undefined;
    window.clearTimeout(this.timer);
    this.request?.abort();
    this.request = undefined;
    this.actionRequest?.abort();
    this.actionRequest = undefined;
    this.actionBusy = false;
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
  private async json(path: string, session: {
    csrf_token: string;
  } | null, body: Record<string, unknown> | null, signal: AbortSignal) {
    if (!this.binding)
      throw Error('Missing terminal binding');
    const headers: Record<string, string> = {
      'X-Soda-Expected-User-ID': this.binding.expectedUserId
    };
    if (body) {
      if (!session)
        throw Error('session');
      headers['Content-Type'] = 'application/json';
      headers['X-CSRF-Token'] = session.csrf_token;
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
  private async session(signal: AbortSignal) {
    const value = await this.json('/api/session', null, null, signal);
    const actor = object(value.user).id;
    if (identifier(actor) && actor !== this.binding?.expectedUserId)
      this.authorityLost();
    if (actor !== this.binding?.expectedUserId || typeof value.csrf_token !== 'string' || !value.csrf_token || value.forgejo_url !== location.origin)
      throw Error('session');
    return {
      csrf_token: value.csrf_token
    };
  }
  private authorityLost() {
    if (this.managed && !this.disposed)
      this.dispatchEvent(new CustomEvent('soda-terminal-authority-lost', {
        bubbles: true
      }));
  }
  private control(action: 'end' | 'return' | 'retain', seconds?: 1800 | 7200) {
    if (action === 'end' && this.managed && this.confirming !== this.sessionID)
      return;
    if (!this.closest('[hidden], [inert]'))
      void this.retention(action, seconds);
  }
  private async retention(action: 'end' | 'return' | 'retain' | 'hide', seconds?: 1800 | 7200) {
    if (this.disposed || this.state === 'stale' || !this.sessionID || !this.binding || this.actionBusy)
      return;
    if (action === 'hide' && !this.attachmentID)
      return;
    const attachment = this.attachmentID;
    const target = this.sessionID, n = this.generation, control = this.actionRequest = new AbortController();
    this.actionBusy = true; // Prevent duplicate actions before Lit updates disabled controls.
    const timeout = window.setTimeout(() => control.abort(), 20000);
    try {
      const current = await this.session(control.signal);
      if (!this.live(n) || this.sessionID !== target)
        return;
      const result = await this.json(`/api/environments/${this.binding.environmentId}/terminal-sessions/${target}`, current, {
        action, ...(seconds ? {
          seconds
        } : {}), ...((action === 'hide' || action === 'return') && attachment ? {
          attachment_id: attachment
        } : {})
      }, control.signal);
      if (!this.live(n))
        return;
      if (action === 'end') {
        if (result.ending !== true)
          throw Error('outcome');
        this.detach('End requested. Native cleanup continues; files and independent services are not undone.', false, 'ending');
        await this.inspectOutcome(target);
      }
      else {
        const retained = terminalResponse(result, this.binding);
        if (!retained || retained.id !== target)
          throw Error('outcome');
        this.observe(retained);
        this.notice = retained.retain_until > 0;
        this.message = retained.retain_until ? `Retained until ${new Date(retained.effective_until * 1000).toLocaleTimeString()}, or authentication expiry.` : `Active as ${this.binding.login}; authentication and native safety leases still apply.`;
      }
    }
    catch {
      if (this.live(n)) {
        this.notice = true;
        this.message = 'Terminal lifetime action was not confirmed. No retry or replacement was made.';
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
  private async inspectOutcome(target: string) {
    if (!this.binding)
      return;
    const n = this.generation, request = this.request = new AbortController();
    const timeout = window.setTimeout(() => request.abort(), 10000);
    try {
      const result = await this.json(`/api/environments/${this.binding.environmentId}/terminal-sessions/${target}`, null, null, request.signal);
      if (!this.live(n) || this.sessionID !== target)
        return;
      const observed = terminalResponse(result, this.binding);
      if (observed && observed.id !== target)
        throw Error('outcome');
      if (observed?.state === 'ended') {
        this.sessionID = undefined;
        this.requestID = undefined;
        this.uncertainCreate = false;
        this.remember(null);
        this.message = 'Native cleanup confirmed for that terminal. Files and independent services are not undone.';
      }
      else {
        this.message = 'End is pending or unconfirmed. Reconnect checks that exact ID; an absent receipt is not cleanup proof.';
        this.publish('state', observed?.state === 'ending' ? 'ending' : 'unconfirmed');
      }
    }
    catch {
      if (this.live(n)) {
        this.message = 'End outcome was not confirmed. The exact locator was retained; no replacement was created.';
        this.publish('state', 'unconfirmed');
      }
    }
    finally {
      window.clearTimeout(timeout);
      if (this.request === request)
        this.request = undefined;
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
    if (automatic && !this.sessionID && !this.requestID)
      return;
    if (!this.viewVisible || this.closest('[hidden]') || (!automatic && (document.visibilityState === 'hidden' || !document.hasFocus())))
      return;
    const {expectedUserId, repositoryId, environmentId, login} = this.binding;
    window.clearTimeout(this.timer);
    this.state = 'opening';
    const n = ++this.generation, request = this.request = new AbortController();
    this.timer = window.setTimeout(() => this.detach('Terminal connection timed out. Use the existing locator or find the pending terminal; creation was not retried.'), 45000);
    this.notice = true;
    this.message = 'Checking your Soda session and original account…';
    try {
      const current = await this.session(request.signal);
      if (!this.live(n))
        return;
      const own = await this.json(`/api/environments/${environmentId}`, null, null, request.signal);
      if (!this.live(n))
        return;
      const env = object(own.environment);
      if (env.id !== environmentId || env.repository_id !== repositoryId || env.provisioned !== true || own.login !== login)
        throw Error('membership');
      if (this.sessionID || this.requestID) {
        const path = this.sessionID ? `terminal-sessions/${this.sessionID}` : `terminal-attempts/${this.requestID}`;
        const metadata = await this.json(`/api/environments/${environmentId}/${path}`, null, null, request.signal);
        if (!this.live(n))
          return;
        const existing = terminalResponse(metadata, this.binding);
        if (existing && ((this.sessionID && existing.id !== this.sessionID) || (!this.sessionID && existing.request_id !== this.requestID)))
          throw Error('terminal metadata');
        if (!existing) {
          this.detach('Terminal outcome remains unknown; no creation was retried. An absent record is not cleanup proof.', false, 'unconfirmed');
          return;
        }
        this.sessionID = existing.id;
        this.requestID = existing.request_id;
        this.uncertainCreate = false;
        this.remember(existing.id);
        this.observe(existing);
        if (existing.state === 'ended') {
          this.sessionID = undefined;
          this.requestID = undefined;
          this.remember(null);
          this.detach('Native cleanup confirmed. Nothing was created; Open terminal explicitly for a new shell.');
          return;
        }
        if (existing.state === 'ending' || existing.state === 'unconfirmed') {
          this.detach('Native cleanup is pending or unconfirmed. This slot is reserved; no attachment or replacement was made. Ask the operator to inspect an unconfirmed outcome.', false, existing.state);
          return;
        }
        if (existing.attached) {
          this.detach('An existing writer is attached. No takeover or replacement was requested.', false, 'attached-elsewhere');
          return;
        }
      }
      if (!this.sessionID && this.uncertainCreate) {
        this.detach('Legacy creation outcome remains unconfirmed. Reload cannot select another terminal; ask the operator to inspect. No creation was retried.', false, 'unconfirmed');
        return;
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
          if (this.managed) {
            this.querySelector<HTMLElement>('[data-action=controls]')?.focus();
            return false;
          }
          const end = this.querySelector<HTMLButtonElement>('[data-action=end]');
          (end && !end.disabled ? end : this.querySelector<HTMLElement>('[role=status]'))?.focus();
          return false;
        }
        return true;
      });
      terminal.onData(data => {
        if (!this.live(n) || !this.viewVisible || this.closest('[hidden]') || this.managed && !screen.contains(document.activeElement))
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
        if (action === 'create') {
          this.requestID = Array.from(crypto.getRandomValues(new Uint8Array(16)), b => b.toString(16).padStart(2, '0')).join('');
          this.uncertainCreate = true;
          this.remember(`pending:${this.requestID}`);
        }
        this.publish('state', 'opening');
        try {
          peer.send(JSON.stringify({
            action, ...(this.sessionID ? {
              id: this.sessionID
            } : {
              request_id: this.requestID, ...(this.createName ? {
                name: this.createName
              } : {})
            }), expected_user_id: expectedUserId, repository_id: repositoryId, csrf_token: current.csrf_token, cols, rows
          }));
          this.message = action === 'create' ? 'Starting the managed terminal…' : 'Attaching the existing terminal…';
        }
        catch {
          this.detach('Attachment dispatch was not confirmed. No creation or input was retried.');
        }
      };
      let queuedOutput = 0, located = false;
      peer.onmessage = event => {
        if (!this.live(n) || this.terminal !== terminal)
          return;
        try {
          if (typeof event.data !== 'string' || event.data.length > 32768)
            throw Error('frame');
          const frame = object(JSON.parse(event.data)), keys = Object.keys(frame).sort().join(',');
          if (frame.type === 'session' && keys === 'attachment_id,id,request_id,type' && this.state === 'opening' && !located && terminalID(frame.id) && terminalID(frame.request_id) && terminalID(frame.attachment_id) && (!this.sessionID || frame.id === this.sessionID) && (!this.requestID || frame.request_id === this.requestID)) {
            located = true;
            this.sessionID = frame.id;
            this.requestID = frame.request_id;
            this.attachmentID = frame.attachment_id;
            this.uncertainCreate = false;
            this.remember(frame.id);
          }
          else if (frame.type === 'ready' && keys === 'type' && this.state === 'opening' && this.sessionID && located) {
            window.clearTimeout(this.timer);
            this.state = 'ready';
            this.retries = 0;
            this.notice = false;
            terminal.options.disableStdin = !this.viewVisible;
            this.message = action === 'create' ? `Connected as ${login}.` : `Reconnected as ${login}. Choose Continue working to renew a detached deadline.`;
            this.publish('state', 'ready');
            void this.screenReady(n, terminal, screen);
            if (this.managed) {
              const target = this.sessionID;
              void this.json(`/api/environments/${environmentId}/terminal-sessions/${target}`, null, null, AbortSignal.any([request.signal, AbortSignal.timeout(10000)])).then(result => {
                if (!this.live(n) || this.sessionID !== target || !this.binding)
                  return;
                const metadata = terminalResponse(result, this.binding);
                if (metadata?.id === target)
                  this.observe(metadata);
              }).catch(() => {
              });
            }
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
        this.detach('Connection lost. The terminal has bounded retention; no input was replayed.');
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
      this.resize(); // Presentation only: never connect, focus or Return.
  }
  focus() {
    if (this.viewVisible && !this.closest('[hidden], [inert]') && this.state === 'ready')
      this.terminal?.focus();
  }
  private observe(metadata: TerminalMetadata) {
    if (metadata.id !== this.sessionID || this.requestID && metadata.request_id !== this.requestID)
      throw Error('Terminal observation changed');
    this.retainUntil = metadata.state === 'opening' || metadata.state === 'ready' ? metadata.retain_until : undefined;
    this.setName(metadata.name);
    if (this.managed)
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
    if (this.sessionID || this.requestID)
      return this.connect(true);
  }
  retain() {
    return this.retention('hide');
  }
  returnToWork() {
    return this.retention('return');
  }
  disconnect() {
    this.detach('Detached. The terminal has bounded retention.');
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
export function mountTerminal(root: HTMLElement, context: TerminalContext, loadRenderer: () => Promise<Renderer> = renderer, locator?: TerminalLocator) {
  const {expectedUserId, repositoryId, environmentId, login} = context;
  if (root.ownerDocument !== document || !identifier(expectedUserId) || !identifier(repositoryId) || !/^p[0-9a-f]{24}$/.test(environmentId) || !/^[a-z][a-z0-9_-]{0,30}$/.test(login) || login === 'root')
    throw Error('Invalid terminal mounting context');
  if (locator?.kind === 'existing' && !terminalID(locator.id) || locator?.kind === 'pending' && !terminalID(locator.requestId))
    throw Error('Invalid terminal locator');
  const terminal = new SodaTerminal();
  terminal.configure(context, loadRenderer, locator);
  root.append(terminal);
  return {
    setVisible: (visible: boolean) => terminal.setVisible(visible), setName: (name: string) => terminal.setName(name), focus: () => terminal.focus(), open: (name?: string) => terminal.open(name), get started() {
      return terminal.started;
    }, get ready() {
      return terminal.updateComplete;
    }, restore: () => terminal.restore(), retain: () => terminal.retain(), returnToWork: () => terminal.returnToWork(),
    invalidate: () => terminal.invalidate(), disconnect: () => terminal.disconnect(), dispose: () => terminal.dispose()
  };
}
