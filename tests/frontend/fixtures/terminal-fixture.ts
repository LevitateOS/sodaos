import {mountTerminal} from '../../../frontend/spaces/sodaspaces-terminal.js';
import type {TerminalLocator} from '../../../frontend/spaces/sodaspaces-terminal.js';
import type {ITerminalOptions, ITerminalInitOnlyOptions} from '@xterm/xterm';
export const environmentID = 'p0123456789abcdef01234567';
interface Existing {id: string; login: string; repository_id: string; name?: string; state?: string; attached?: boolean}
export interface TerminalFixtureOptions {existing?: Existing; locator?: TerminalLocator; slow?: boolean; user?: string; login?: string; responseStatus?: number}
interface Call {url: string; method: string; body?: string; credentials?: RequestCredentials; redirect?: RequestRedirect; headers: Record<string, string>}
function createFixture(options: TerminalFixtureOptions = {}) {
  const root = document.getElementById('mount'); if (!root) throw Error('Missing fixture mount');
  const calls: Call[] = [], sockets: FakeSocket[] = [], terms: Terminal[] = [];
  let existing: Existing | null = options.existing || null, before = 0, fitCalls = 0, observed = 0, disconnected = 0;
  const created = Math.floor(Date.now()/1000);
  const metadata = () => existing ? {...existing, environment_id: environmentID, user_id: '1', name: existing.name || '', created_at: created,
    ready: !existing.state || existing.state === 'ready', attached: (!existing.state || existing.state === 'ready') && (existing.attached === true || sockets.some(socket => socket.readyState === 1)), state: existing.state || 'ready'} : null;
  // Observe actual browser geometry while recording external-resource ownership.
  const NativeObserver = ResizeObserver;
  class ObservedResize extends NativeObserver {
    observe(target: Element, options?: ResizeObserverOptions) {observed++; super.observe(target, options);}
    disconnect() {disconnected++; super.disconnect();}
  }
  Object.defineProperty(window, 'ResizeObserver', {value: ObservedResize, configurable: true});
  let reply: ((call: Call) => Promise<Response | null>) | undefined;
  let rendererWait: Promise<void> | undefined;
  window.addEventListener('beforeunload', () => before++);
  const locators: (string | null)[] = [];
  root.addEventListener('soda-terminal-locator', event => {
    if (!(event instanceof CustomEvent) || !(event.detail === null || typeof event.detail === 'string')) throw Error('Invalid locator event');
    locators.push(event.detail);
  });
  const fetchFixture = async (url: RequestInfo | URL, init?: RequestInit) => {
    const call: Call = {url: String(url), method: init?.method || 'GET', headers: Object.fromEntries(new Headers(init?.headers).entries()),
      ...(typeof init?.body === 'string' ? {body: init.body} : {}), ...(init?.credentials ? {credentials: init.credentials} : {}), ...(init?.redirect ? {redirect: init.redirect} : {})};
    calls.push(call);
    const override = await reply?.(call); if (override) return override;
    if (options.responseStatus || options.user && options.user !== '1' || options.login && options.login !== 'original-alice') return new Response(null, {status: options.responseStatus || 403});
    if (call.url.endsWith('/terminal-sessions') && call.method === 'POST') return Response.json({id: 'a'.repeat(32)}, {status: 201});
    if (call.url.includes('/terminal-sessions/')) {
      if (call.method === 'POST') {
        const input: unknown = JSON.parse(call.body || '{}');
        if (!input || typeof input !== 'object' || !('action' in input)) throw Error('Missing action');
        if (input.action === 'end') {existing = null; for (const peer of sockets) if (peer.readyState === 1) peer.close();}
      }
      return Response.json({terminal: metadata()});
    }
    return Response.json(call.url.endsWith('/session') ? {user: {id: options.user || '1'}, csrf_token: 'synthetic-csrf', forgejo_url: location.origin}
      : {environment: {id: environmentID, repository_id: '7', provisioned: true}, login: options.login || 'original-alice'});
  };
  Object.defineProperty(window, 'fetch', {value: fetchFixture, configurable: true});
  class FakeSocket {
    readyState = 0; bufferedAmount = 0; sent: Record<string, unknown>[] = []; closed = 0;
    onopen?: () => void; onclose?: () => void; onerror?: () => void; onmessage?: (event: {data: string}) => void;
    url: string;
    constructor(url: string | URL) {this.url = String(url); sockets.push(this);}
    send(body: string) {
      const frame: unknown = JSON.parse(body);
      if (!frame || typeof frame !== 'object' || Array.isArray(frame)) throw Error('Invalid outgoing frame');
      this.sent.push({...frame});
      if ('action' in frame && frame.action === 'create' && 'id' in frame && typeof frame.id === 'string')
        existing = {id: frame.id, login: 'original-alice', repository_id: '7'};
    }
    close() {this.closed++; this.readyState = 3; this.onclose?.();}
    open() {this.readyState = 1; if (!this.onopen) throw Error('Missing handler'); this.onopen();}
    message(value: unknown) {
      this.raw(JSON.stringify(value));
    }
    raw(data: string) {if (!this.onmessage) throw Error('Missing handler'); this.onmessage({data});}
  }
  Object.defineProperty(window, 'WebSocket', {value: FakeSocket, configurable: true});
  class Terminal {
    cols = 80; rows = 24; osc: number[] = []; writes: (string | Uint8Array)[] = []; disposed = 0;
    textarea = document.createElement('textarea');
    key: (event: KeyboardEvent) => boolean = () => {throw Error('Missing keyboard handler');};
    input: (data: string) => void = () => {throw Error('Missing input handler');};
    parser = {registerOscHandler: (code: number, fn: (data: string) => boolean | Promise<boolean>) => {if (fn('') !== true) throw Error('Unconsumed OSC'); this.osc.push(code); return {dispose() {}};},
      registerCsiHandler() {return {dispose() {}};}, registerDcsHandler() {return {dispose() {}};}, registerEscHandler() {return {dispose() {}};}};
    constructor(public options: ITerminalOptions & ITerminalInitOnlyOptions) {terms.push(this);}
    loadAddon() {} attachCustomKeyEventHandler(fn: (event: KeyboardEvent) => boolean) {this.key = fn;}
    onRender() {return {dispose() {}};}
    onData(fn: (data: string) => void) {this.input = fn; return {dispose() {}};}
    open(node: HTMLElement) {node.append(this.textarea);}
    focus() {this.textarea.focus();} dispose() {this.disposed++;}
    write(bytes: string | Uint8Array, done?: () => void) {this.writes.push(bytes); if (!options.slow) done?.();}
  }
  const binding = {expectedUserId: '1', csrfToken: 'synthetic-csrf', repositoryId: '7', environmentId: environmentID, login: 'original-alice'};
  const api = mountTerminal(root, binding, options.locator || {kind: 'new'}, async () => {
    await rendererWait;
    return {Terminal, FitAddon: class {fit() {fitCalls++;} activate() {} dispose() {}}};
  });
  const button = (text: string) => {const found = [...root.querySelectorAll('button')].find(b => b.textContent === text); if (!found) throw Error('Missing button: ' + text); return found;};
  const socket = () => {const result = sockets.at(-1); if (!result) throw Error('Missing socket'); return result;};
  const term = () => {const result = terms.at(-1); if (!result) throw Error('Missing renderer'); return result;};
  return {api, root, calls, sockets, terms, button, socket, term, before: () => before, fits: () => fitCalls, observers: () => ({observed, disconnected}),
    setReply(value: typeof reply) {reply = value;}, waitRenderer(value: Promise<void>) {rendererWait = value;},
    ready() {const peer = socket(); peer.open(); peer.message({type: 'ready'}); return api.ready;},
    writes: () => calls.filter(call => call.method !== 'GET'),
    locator: () => locators.at(-1), locators,
    // Exercise the exported JavaScript boundary with invalid, untyped callers.
    mountInvalidLocator(value: unknown) {Reflect.apply(mountTerminal, undefined, [root, binding, value]);}};
}
declare global {interface Window {createTerminalFixture: typeof createFixture; terminalFixture: ReturnType<typeof createFixture>}}
window.createTerminalFixture = createFixture;
