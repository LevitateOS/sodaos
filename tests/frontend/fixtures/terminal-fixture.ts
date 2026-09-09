import {mountTerminal} from '../../../frontend/spaces/sodaspaces-terminal.js';
import type {ITerminalOptions, ITerminalInitOnlyOptions} from '@xterm/xterm';
export const environmentID = 'p0123456789abcdef01234567';
interface Existing {id: string; login: string; repository_id: string; request_id?: string; state?: string; retain_until?: number}
export interface TerminalFixtureOptions {existing?: Existing; saved?: string; slow?: boolean; user?: string; login?: string; responseStatus?: number}
interface Call {url: string; method: string; body?: string; credentials?: RequestCredentials; redirect?: RequestRedirect; headers: Record<string, string>}
function createFixture(options: TerminalFixtureOptions = {}) {
  const root = document.getElementById('mount'); if (!root) throw Error('Missing fixture mount');
  const calls: Call[] = [], sockets: FakeSocket[] = [], terms: Terminal[] = [];
  let existing: Existing | null = options.existing || null, before = 0, fitCalls = 0, observed = 0, disconnected = 0;
  const created = Math.floor(Date.now()/1000), hard = created + 43200;
  const metadata = () => existing ? {...existing, request_id: existing.request_id || 'b'.repeat(32), environment_id: environmentID, user_id: '1', name: '', created_at: created, hard_until: hard,
    retain_until: existing.retain_until || 0, effective_until: existing.retain_until || hard, ready: !existing.state || existing.state === 'ready', attached: !existing.state || existing.state === 'ready', state: existing.state || 'ready'} : null;
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
  if (options.saved) sessionStorage.setItem(`soda-terminal:1:${environmentID}`, options.saved);
  const fetchFixture = async (url: RequestInfo | URL, init?: RequestInit) => {
    const call: Call = {url: String(url), method: init?.method || 'GET', headers: Object.fromEntries(new Headers(init?.headers).entries()),
      ...(typeof init?.body === 'string' ? {body: init.body} : {}), ...(init?.credentials ? {credentials: init.credentials} : {}), ...(init?.redirect ? {redirect: init.redirect} : {})};
    calls.push(call);
    const override = await reply?.(call); if (override) return override;
    if (options.responseStatus) return new Response(null, {status: options.responseStatus});
    if (call.url.includes('/terminal-sessions/') || call.url.includes('/terminal-attempts/')) {
      if (call.method === 'POST') {
        const input: unknown = JSON.parse(call.body || '{}');
        if (!input || typeof input !== 'object' || !('action' in input)) throw Error('Missing action');
        if (input.action === 'end') {if (existing) existing = {...existing, state: 'ended'}; return Response.json({ending: true});}
        if (existing) existing = {...existing, retain_until: input.action === 'return' ? 0 : Math.floor(Date.now()/1000) + 7200};
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
    }
    close() {this.closed++; this.readyState = 3; this.onclose?.();}
    open() {this.readyState = 1; if (!this.onopen) throw Error('Missing handler'); this.onopen();}
    message(value: unknown) {
      if (value && typeof value === 'object' && 'type' in value && value.type === 'session' && 'id' in value && typeof value.id === 'string') {
        const request = this.sent[0]?.request_id;
        const request_id = typeof request === 'string' ? request : existing?.request_id || 'b'.repeat(32);
        existing = {id: value.id, login: 'original-alice', repository_id: '7', request_id};
        value = {...value, request_id, attachment_id: sockets.length.toString(16).padStart(32, '0')};
      }
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
    onData(fn: (data: string) => void) {this.input = fn; return {dispose() {}};}
    open(node: HTMLElement) {node.append(this.textarea);}
    focus() {this.textarea.focus();} dispose() {this.disposed++;}
    write(bytes: string | Uint8Array, done?: () => void) {this.writes.push(bytes); if (!options.slow) done?.();}
  }
  const api = mountTerminal(root, {expectedUserId: '1', repositoryId: '7', environmentId: environmentID, login: 'original-alice'}, async () => {
    await rendererWait;
    return {Terminal, FitAddon: class {fit() {fitCalls++;} activate() {} dispose() {}}};
  });
  const button = (text: string) => {const found = [...root.querySelectorAll('button')].find(b => b.textContent === text); if (!found) throw Error('Missing button: ' + text); return found;};
  const socket = () => {const result = sockets.at(-1); if (!result) throw Error('Missing socket'); return result;};
  const term = () => {const result = terms.at(-1); if (!result) throw Error('Missing renderer'); return result;};
  return {api, root, calls, sockets, terms, button, socket, term, before: () => before, fits: () => fitCalls, observers: () => ({observed, disconnected}),
    setReply(value: typeof reply) {reply = value;}, waitRenderer(value: Promise<void>) {rendererWait = value;},
    ready() {const peer = socket(); peer.open(); peer.message({type: 'session', id: 'a'.repeat(32)}); peer.message({type: 'ready'}); return api.ready;},
    writes: () => calls.filter(call => call.method !== 'GET'),
    storage: () => sessionStorage.getItem(`soda-terminal:1:${environmentID}`)};
}
declare global {interface Window {createTerminalFixture: typeof createFixture; terminalFixture: ReturnType<typeof createFixture>}}
window.createTerminalFixture = createFixture;
