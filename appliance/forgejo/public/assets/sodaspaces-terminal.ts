import {LitElement, html} from 'lit';
import {object, readSodaJSON, terminalID, terminalResponse, id as identifier} from './sodaspaces-api.js';
import type {Terminal, ITerminalOptions, ITerminalInitOnlyOptions, ITerminalAddon} from '@xterm/xterm';
import type {FitAddon} from '@xterm/addon-fit';

export type TerminalView = Pick<Terminal, 'cols' | 'rows' | 'options' | 'parser' | 'loadAddon' | 'attachCustomKeyEventHandler' | 'onData' | 'open' | 'focus' | 'dispose' | 'write'>;
export interface Renderer {Terminal: new (options: ITerminalOptions & ITerminalInitOnlyOptions) => TerminalView; FitAddon: new () => Pick<FitAddon, 'fit'> & ITerminalAddon}
export interface TerminalContext {expectedUserId: string; repositoryId: string; environmentId: string; login: string; projectName?: string}
export type TerminalLocator = {kind: 'new'} | {kind: 'existing'; id: string} | {kind: 'pending'; requestId: string};
const renderer = async (): Promise<Renderer> => {
  const [{Terminal}, {FitAddon}] = await Promise.all([import('./soda-terminal/xterm.mjs'), import('./soda-terminal/addon-fit.mjs')]);
  return {Terminal, FitAddon};
};

// One immutable original account and imperative attachment owner. Rendering never
// creates a shell, opens a socket, Returns, or touches xterm's screen descendants.
export class SodaTerminal extends LitElement {
  static properties = {state: {state: true}, sessionID: {state: true}, uncertainCreate: {state: true}, message: {state: true}, screenVisible: {state: true}, actionBusy: {state: true}, endConfirmed: {state: true}};
  declare private state: 'idle' | 'opening' | 'ready' | 'closed' | 'stale';
  declare private sessionID: string | undefined;
  declare private uncertainCreate: boolean;
  declare private message: string;
  declare private screenVisible: boolean;
  declare private actionBusy: boolean;
  declare private endConfirmed: boolean;
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
  constructor() {
    super(); this.endConfirmed = false; this.state = 'idle'; this.sessionID = undefined; this.uncertainCreate = this.screenVisible = this.actionBusy = false; this.message = 'Not connected.';
  }
  protected createRenderRoot() {return this;}
  configure(context: TerminalContext, load: () => Promise<Renderer>, locator?: TerminalLocator) {
    if (this.binding || this.disposed) throw Error('Terminal binding is immutable');
    this.binding = {...context}; this.loadRenderer = load;
    this.storageKey = `soda-terminal:${context.expectedUserId}:${context.environmentId}`;
    this.managed = locator !== undefined;
    if (locator?.kind === 'existing') this.sessionID = locator.id;
    if (locator?.kind === 'pending') {this.requestID = locator.requestId; this.uncertainCreate = true;}
    try {
      const saved = this.managed ? null : sessionStorage.getItem(this.storageKey);
      if (saved?.startsWith('pending:') && terminalID(saved.slice(8))) {this.requestID = saved.slice(8); this.uncertainCreate = true;}
      else if (terminalID(saved)) this.sessionID = saved;
      else if (saved === 'pending') this.uncertainCreate = true; // Old ambiguity must never select another session.
    } catch { /* locator only */ }
    window.addEventListener('pagehide', () => this.invalidate(), {signal: this.lifetime.signal});
    window.addEventListener('pageshow', event => {if (event.persisted) this.invalidate();}, {signal: this.lifetime.signal});
  }
  disconnectedCallback() {super.disconnectedCallback(); this.dispose();}
  protected render() {
    const disabled = this.disposed || this.state === 'stale';
    return html`<section class=${'soda-terminal' + (this.state === 'ready' ? ' is-connected' : '')}>
      <h3>Project terminal as ${this.binding?.login} in ${this.binding?.projectName || this.binding?.environmentId}</h3>
      ${this.managed ? html`<label><input type="checkbox" .checked=${this.endConfirmed} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) this.endConfirmed = e.target.checked;}}> Confirm End for ${this.binding?.login} in ${this.binding?.projectName || this.binding?.environmentId} (${this.binding?.environmentId}); files and independent services remain.</label>` : ''}
      <p>The same tmux session survives navigation and connection loss. Detached or hidden work is retained for 30 minutes, within Soda authentication. Reconnecting alone does not extend retention. Tmux copy-mode holds history; no commands are replayed.</p>
      <div class="soda-terminal-toolbar">
        <button type="button" class="ui primary button" ?disabled=${disabled || this.managedEnded || this.state === 'opening' || this.state === 'ready' || this.actionBusy}
          @click=${() => {this.retries = 0; void this.connect();}}>${this.sessionID ? 'Reconnect terminal' : this.uncertainCreate ? 'Find pending terminal' : 'Open terminal'}</button>
        <button type="button" class="ui basic button" data-action="end" ?disabled=${disabled || !this.sessionID || this.actionBusy} @click=${() => this.control('end')}>End terminal</button>
        <button type="button" class="ui basic button" ?disabled=${disabled || !this.sessionID || this.actionBusy} @click=${() => this.control('return')}>Continue working</button>
        <button type="button" class="ui basic button" ?disabled=${disabled || !this.sessionID || this.actionBusy} @click=${() => this.control('retain', 7200)}>Keep for two hours</button>
        <p role="status" tabindex="-1">${this.message}</p>
      </div>
      <div class="soda-terminal-screen" ?hidden=${!this.screenVisible} aria-label=${`Terminal for ${this.binding?.login}; Ctrl+Shift+Enter focuses End terminal`}
        @keydown=${(event: KeyboardEvent) => {if (event.key === 'Escape') {event.preventDefault(); event.stopPropagation();}}}></div>
    </section>`;
  }
  private remember(value: string | null) {
    if (this.managed) {if (value === null) this.managedEnded = true; this.dispatchEvent(new CustomEvent('soda-terminal-locator', {bubbles: true, detail: value})); return;}
    try {if (value) sessionStorage.setItem(this.storageKey, value); else sessionStorage.removeItem(this.storageKey);} catch { /* no credentials or transcript */ }
  }
  private live(generation: number) {return !this.disposed && this.state !== 'stale' && this.generation === generation;}
  private detach(message: string, stale = false) {
    ++this.generation; this.state = stale ? 'stale' : 'closed';
    window.clearTimeout(this.timer); this.request?.abort(); this.request = undefined;
    this.actionRequest?.abort(); this.actionRequest = undefined; this.actionBusy = false;
    this.observer?.disconnect(); this.observer = undefined;
    const old = this.socket; this.socket = undefined; if (old && old.readyState < 2) old.close();
    this.terminal?.dispose(); this.terminal = undefined; this.fit = undefined;
    this.querySelector('.soda-terminal-screen')?.replaceChildren(); this.screenVisible = false; this.message = message;
  }
  invalidate() {this.detach('Page context changed. No action was replayed; reconnect through a fresh authorized page.', true);}
  private async json(path: string, session: {csrf_token: string} | null, body: Record<string, unknown> | null, signal: AbortSignal) {
    if (!this.binding) throw Error('Missing terminal binding');
    const headers: Record<string, string> = {'X-Soda-Expected-User-ID': this.binding.expectedUserId};
    if (body) {if (!session) throw Error('session'); headers['Content-Type'] = 'application/json'; headers['X-CSRF-Token'] = session.csrf_token;}
    const response = await fetch('/-/soda' + path, {method: body ? 'POST' : 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, ...(body ? {body: JSON.stringify(body)} : {}), signal});
    if (!response.ok) throw Error('Soda request refused');
    return object(await readSodaJSON(response));
  }
  private async session(signal: AbortSignal) {
    const value = await this.json('/api/session', null, null, signal);
    if (object(value.user).id !== this.binding?.expectedUserId || typeof value.csrf_token !== 'string' || !value.csrf_token || value.forgejo_url !== location.origin) throw Error('session');
    return {csrf_token: value.csrf_token};
  }
  private control(action: 'end' | 'return' | 'retain', seconds?: 1800 | 7200) {
    if (action === 'end' && this.managed && !this.endConfirmed) {this.message = 'Confirm the original account/project before End.'; return;}
    if (!this.closest('[hidden]')) void this.retention(action, seconds);
  }
  private async retention(action: 'end' | 'return' | 'retain' | 'hide', seconds?: 1800 | 7200) {
    if (this.disposed || this.state === 'stale' || !this.sessionID || !this.binding || this.actionBusy) return;
    if (action === 'hide' && !this.attachmentID) return;
    const attachment = this.attachmentID;
    const target = this.sessionID, n = this.generation, control = this.actionRequest = new AbortController();
    this.actionBusy = true; // Prevent duplicate actions before Lit updates disabled controls.
    const timeout = window.setTimeout(() => control.abort(), 20000);
    try {
      const current = await this.session(control.signal);
      if (!this.live(n) || this.sessionID !== target) return;
      const result = await this.json(`/api/environments/${this.binding.environmentId}/terminal-sessions/${target}`, current,
        {action, ...(seconds ? {seconds} : {}), ...((action === 'hide' || action === 'return') && attachment ? {attachment_id: attachment} : {})}, control.signal);
      if (!this.live(n)) return;
      if (action === 'end') {
        if (result.ending !== true) throw Error('outcome');
        this.detach('End requested. Native cleanup continues; files and independent services are not undone.');
        await this.inspectOutcome(target);
      } else {
        const retained = terminalResponse(result, this.binding);
        if (!retained || retained.id !== target) throw Error('outcome');
        if (this.managed) this.dispatchEvent(new CustomEvent('soda-terminal-metadata', {bubbles: true, detail: retained}));
        this.message = retained.retain_until ? `Retained until ${new Date(retained.retain_until * 1000).toLocaleTimeString()}, or authentication expiry.` : `Active as ${this.binding.login}; authentication and native safety leases still apply.`;
      }
    } catch {if (this.live(n)) this.message = 'Terminal lifetime action was not confirmed. No retry or replacement was made.';}
    finally {window.clearTimeout(timeout); if (this.actionRequest === control) {this.actionRequest = undefined; this.actionBusy = false;}}
  }
  private async inspectOutcome(target: string) {
    if (!this.binding) return;
    const n = this.generation, request = this.request = new AbortController();
    const timeout = window.setTimeout(() => request.abort(), 10000);
    try {
      const result = await this.json(`/api/environments/${this.binding.environmentId}/terminal-sessions/${target}`, null, null, request.signal);
      if (!this.live(n) || this.sessionID !== target) return;
      const observed = terminalResponse(result, this.binding);
      if (observed && observed.id !== target) throw Error('outcome');
      if (observed?.state === 'ended') {
        this.sessionID = undefined; this.requestID = undefined; this.uncertainCreate = false; this.remember(null);
        this.message = 'Native cleanup confirmed for that terminal. Files and independent services are not undone.';
      } else this.message = 'End is pending or unconfirmed. Reconnect checks that exact ID; an absent receipt is not cleanup proof.';
    } catch {if (this.live(n)) this.message = 'End outcome was not confirmed. The exact locator was retained; no replacement was created.';}
    finally {window.clearTimeout(timeout); if (this.request === request) this.request = undefined;}
  }
  private send(frame: {type: 'resize'; cols: number; rows: number} | {type: 'input'; data: string}) {
    if (this.state !== 'ready' || !this.socket || this.socket.readyState !== 1 || this.socket.bufferedAmount > 65536) {this.detach('Connection lost or overloaded. Reconnect the existing terminal; input was not replayed.'); return false;}
    try {this.socket.send(JSON.stringify(frame)); return true;} catch {this.detach('Connection lost. No input was replayed.'); return false;}
  }
  private resize = () => {
    const screen = this.querySelector<HTMLElement>('.soda-terminal-screen');
    if (!this.terminal || !this.fit || this.state !== 'ready' || !screen?.isConnected || !screen.clientWidth || !screen.clientHeight) return;
    this.fit.fit();
    if (this.terminal.cols >= 2 && this.terminal.cols <= 500 && this.terminal.rows >= 2 && this.terminal.rows <= 300) this.send({type: 'resize', cols: this.terminal.cols, rows: this.terminal.rows});
  };
  private async connect(automatic = false) {
    if (this.disposed || !this.binding || this.state === 'stale' || this.state === 'opening' || this.state === 'ready' || this.actionBusy) return;
    if (this.managedEnded) {this.message = 'This exact session ended. Use New terminal for a different shell.'; return;}
    if (automatic && !this.sessionID && !this.requestID) return;
    if (this.closest('[hidden]') || (!automatic && (document.visibilityState === 'hidden' || !document.hasFocus()))) return;
    const {expectedUserId, repositoryId, environmentId, login} = this.binding;
    window.clearTimeout(this.timer); this.state = 'opening';
    const n = ++this.generation, request = this.request = new AbortController();
    this.timer = window.setTimeout(() => this.detach('Terminal connection timed out. Use the existing locator or find the pending terminal; creation was not retried.'), 45000);
    this.message = 'Checking your Soda session and original account…';
    try {
      const current = await this.session(request.signal); if (!this.live(n)) return;
      const own = await this.json(`/api/environments/${environmentId}`, null, null, request.signal); if (!this.live(n)) return;
      const env = object(own.environment);
      if (env.id !== environmentId || env.repository_id !== repositoryId || env.provisioned !== true || own.login !== login) throw Error('membership');
      if (this.sessionID || this.requestID) {
        const path = this.sessionID ? `terminal-sessions/${this.sessionID}` : `terminal-attempts/${this.requestID}`;
        const metadata = await this.json(`/api/environments/${environmentId}/${path}`, null, null, request.signal); if (!this.live(n)) return;
        const existing = terminalResponse(metadata, this.binding);
        if (existing && ((this.sessionID && existing.id !== this.sessionID) || (!this.sessionID && existing.request_id !== this.requestID))) throw Error('terminal metadata');
        if (!existing) {this.detach('Terminal outcome remains unknown; no creation was retried. An absent record is not cleanup proof.'); return;}
        this.sessionID = existing.id; this.requestID = existing.request_id; this.uncertainCreate = false; this.remember(existing.id);
        if (existing.state === 'ended') {this.sessionID = undefined; this.requestID = undefined; this.remember(null); this.detach('Native cleanup confirmed. Nothing was created; Open terminal explicitly for a new shell.'); return;}
        if (existing.state === 'ending' || existing.state === 'unconfirmed') {this.detach('Native cleanup is pending or unconfirmed. This slot is reserved; no attachment or replacement was made. Ask the operator to inspect an unconfirmed outcome.'); return;}
      }
      if (!this.sessionID && this.uncertainCreate) {this.detach('Legacy creation outcome remains unconfirmed. Reload cannot select another terminal; ask the operator to inspect. No creation was retried.'); return;}
      if (automatic && !this.sessionID) {this.detach('Terminal absent. Nothing was created.'); return;}
      const action = this.sessionID ? 'attach' : 'create';
      const {Terminal, FitAddon} = await this.loadRenderer(); if (!this.live(n)) return;
      this.screenVisible = true;
      await this.updateComplete; if (!this.live(n)) return;
      if (!this.isConnected || this.closest('[hidden]')) {this.detach('Attachment cancelled while hidden. No creation was sent.'); return;}
      const screen = this.querySelector<HTMLElement>('.soda-terminal-screen'); if (!screen) throw Error('Missing terminal screen');
      const terminal = this.terminal = new Terminal({allowProposedApi: true, disableStdin: true, scrollback: 1000, windowOptions: {}, convertEol: false, cols: 80, rows: 24});
      const fit = this.fit = new FitAddon(); terminal.loadAddon(fit);
      for (const code of [0, 1, 2, 8, 52]) terminal.parser.registerOscHandler(code, () => true);
      terminal.attachCustomKeyEventHandler(event => {
        if (event.ctrlKey && event.shiftKey && event.key === 'Enter') {
          event.preventDefault(); event.stopPropagation();
          const end = this.querySelector<HTMLButtonElement>('[data-action=end]');
          (end && !end.disabled ? end : this.querySelector<HTMLElement>('[role=status]'))?.focus();
          return false;
        }
        return true;
      });
      terminal.onData(data => {
        if (!this.live(n)) return;
        if (data.length > 65536) {this.detach('Input too large. Nothing was replayed.'); return;}
        const bytes = new TextEncoder().encode(data);
        for (let i = 0; i < bytes.length; i += 16384) if (!this.send({type: 'input', data: btoa(String.fromCharCode(...bytes.subarray(i, i + 16384)))})) break;
      });
      terminal.open(screen); if (screen.clientWidth && screen.clientHeight) fit.fit();
      const cols = Math.max(2, Math.min(500, terminal.cols)), rows = Math.max(2, Math.min(300, terminal.rows));
      const url = new URL(`/-/soda/api/environments/${environmentId}/terminal`, location.origin); url.protocol = 'wss:';
      const peer = this.socket = new WebSocket(url);
      peer.onopen = () => {
        if (!this.live(n)) {peer.close(); return;}
        if (this.closest('[hidden]')) {this.detach('Attachment cancelled while hidden. No creation was sent.'); return;}
        if (action === 'create') {this.requestID = Array.from(crypto.getRandomValues(new Uint8Array(16)), b => b.toString(16).padStart(2, '0')).join(''); this.uncertainCreate = true; this.remember(`pending:${this.requestID}`);}
        try {
          peer.send(JSON.stringify({action, ...(this.sessionID ? {id: this.sessionID} : {request_id: this.requestID}), expected_user_id: expectedUserId, repository_id: repositoryId, csrf_token: current.csrf_token, cols, rows}));
          this.message = action === 'create' ? 'Starting the managed terminal…' : 'Attaching the existing terminal…';
        } catch {this.detach('Attachment dispatch was not confirmed. No creation or input was retried.');}
      };
      let queuedOutput = 0, located = false;
      peer.onmessage = event => {
        if (!this.live(n) || this.terminal !== terminal) return;
        try {
          if (typeof event.data !== 'string' || event.data.length > 32768) throw Error('frame');
          const frame = object(JSON.parse(event.data)), keys = Object.keys(frame).sort().join(',');
          if (frame.type === 'session' && keys === 'attachment_id,id,request_id,type' && this.state === 'opening' && !located && terminalID(frame.id) && terminalID(frame.request_id) && terminalID(frame.attachment_id) && (!this.sessionID || frame.id === this.sessionID) && (!this.requestID || frame.request_id === this.requestID)) {
            located = true; this.sessionID = frame.id; this.requestID = frame.request_id; this.attachmentID = frame.attachment_id; this.uncertainCreate = false; this.remember(frame.id);
          } else if (frame.type === 'ready' && keys === 'type' && this.state === 'opening' && this.sessionID && located) {
            window.clearTimeout(this.timer); this.state = 'ready'; this.retries = 0; terminal.options.disableStdin = false;
            this.message = action === 'create' ? `Connected as ${login}.` : `Reconnected as ${login}. Choose Continue working to renew a detached deadline.`;
            void this.screenReady(n, terminal, screen);
            if (this.managed) {
              const target = this.sessionID;
              void this.json(`/api/environments/${environmentId}/terminal-sessions/${target}`, null, null, AbortSignal.any([request.signal, AbortSignal.timeout(10000)])).then(result => {
                if (!this.live(n) || this.sessionID !== target || !this.binding) return;
                const metadata = terminalResponse(result, this.binding);
                if (metadata?.id === target) this.dispatchEvent(new CustomEvent('soda-terminal-metadata', {bubbles: true, detail: metadata}));
              }).catch(() => { /* A failed observation neither changes lifetime nor replaces a shell. */ });
            }
          } else if (frame.type === 'output' && this.state === 'ready' && keys === 'data,type' && typeof frame.data === 'string') {
            const decoded = atob(frame.data);
            if (!decoded.length || decoded.length > 4096 || queuedOutput + decoded.length > 262144) throw Error('output');
            queuedOutput += decoded.length; terminal.write(Uint8Array.from(decoded, c => c.charCodeAt(0)), () => {queuedOutput -= decoded.length;});
          } else if (frame.type === 'closed' && keys === 'reason,type') this.detach('Attachment ended or unavailable. Reconnect only the existing terminal; no replacement was launched.');
          else throw Error('frame');
        } catch {this.detach('Invalid or overloaded stream. No input or creation was replayed.');}
      };
      peer.onerror = peer.onclose = () => {
        if (!this.live(n)) return;
        this.detach('Connection lost. The terminal has bounded retention; no input was replayed.');
        if (this.sessionID && this.retries < 3) {const wait = 1000 * 2 ** this.retries++; this.timer = window.setTimeout(() => this.connect(true), wait);}
      };
    } catch {if (this.live(n)) this.detach('Could not authorize or attach. Sign in again or inspect the existing terminal; nothing was replayed.');}
  }
  private async screenReady(n: number, terminal: TerminalView, screen: HTMLElement) {
    try {
      await this.updateComplete;
      if (!this.live(n) || this.terminal !== terminal) return;
      if (document.hasFocus() && document.visibilityState !== 'hidden' && !this.closest('[hidden]') && (document.activeElement === document.body || this.contains(document.activeElement))) terminal.focus();
      // Native focus handlers can synchronously retire this component.
      if (!this.live(n) || this.terminal !== terminal || !screen.isConnected) return;
      this.observer = new ResizeObserver(this.resize); this.observer.observe(screen); this.resize();
    } catch {if (this.live(n)) this.detach('Terminal rendering failed. No input or creation was replayed.');}
  }
  open() {return this.connect();}
  get started() {return this.state !== 'idle';}
  restore() {if (this.sessionID || this.requestID) return this.connect(true);}
  retain() {return this.retention('hide');}
  returnToWork() {return this.retention('return');}
  disconnect() {this.detach('Detached. The terminal has bounded retention.');}
  dispose() {if (this.disposed) return; this.detach('Detached.', true); this.disposed = true; this.lifetime.abort(); this.remove();}
}
customElements.define('soda-terminal', SodaTerminal);

export function mountTerminal(root: HTMLElement, context: TerminalContext, loadRenderer: () => Promise<Renderer> = renderer, locator?: TerminalLocator) {
  const {expectedUserId, repositoryId, environmentId, login} = context;
  if (root.ownerDocument !== document || !identifier(expectedUserId) || !identifier(repositoryId) || !/^p[0-9a-f]{24}$/.test(environmentId) || !/^[a-z][a-z0-9_-]{0,30}$/.test(login) || login === 'root') throw Error('Invalid terminal mounting context');
  if (locator?.kind === 'existing' && !terminalID(locator.id) || locator?.kind === 'pending' && !terminalID(locator.requestId)) throw Error('Invalid terminal locator');
  const terminal = new SodaTerminal(); terminal.configure(context, loadRenderer, locator); root.append(terminal);
  return {open: () => terminal.open(), get started() {return terminal.started;}, get ready() {return terminal.updateComplete;}, restore: () => terminal.restore(), retain: () => terminal.retain(), returnToWork: () => terminal.returnToWork(),
    invalidate: () => terminal.invalidate(), disconnect: () => terminal.disconnect(), dispose: () => terminal.dispose()};
}
