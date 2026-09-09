import test, {type TestContext} from 'node:test';
import type {ITerminalOptions, ITerminalInitOnlyOptions} from '@xterm/xterm';
import assert from 'node:assert/strict';
import {mountTerminal} from '../../appliance/forgejo/public/assets/sodaspaces-terminal.ts';
import {JSDOM} from 'jsdom';
const env = 'p0123456789abcdef01234567';
const tick = () => new Promise(resolve => setTimeout(resolve, 0));
function fixture(t: TestContext, options: {slow?: boolean; fetch?: (url: string, init: RequestInit) => Promise<Response>} = {}) {
  const dom = new JSDOM('<button id="native">Native</button><div id="mount"></div>', {url: 'https://forge.test/alice/repo', pretendToBeVisual: true});
  const w = dom.window, root = w.document.getElementById('mount');
  assert(root);
  w.document.hasFocus = () => true;
  let before = 0;
  w.addEventListener('beforeunload', () => before++);
  const calls: Array<{url: string; init: RequestInit}> = [], sockets: FakeSocket[] = [], terms: Terminal[] = [];
  Object.defineProperty(w, 'fetch', {value: async (url: string, init: RequestInit) => {
    calls.push({url, init});
    if (options.fetch) return options.fetch(url, init);
    return new Response(JSON.stringify(url.endsWith('/session') ? {user: {id: '1'}, csrf_token: 'synthetic-csrf', forgejo_url: 'https://forge.test'} : {environment: {id: env, repository_id: '7', provisioned: true}, login: 'original-alice'}), {headers: {'Content-Type': 'application/json'}});
  }});
  Object.defineProperty(w, 'ResizeObserver', {value: class {observe() {} disconnect() {}}});
  class FakeSocket {
    readyState = 0; bufferedAmount = 0; sent: Array<Record<string, unknown>> = [];
    onopen?: () => void; onclose?: () => void; onmessage?: (event: {data: string}) => void;
    url: string;
    constructor(url: string | URL) { this.url = String(url); sockets.push(this); }
    send(body: string) { const frame: unknown = JSON.parse(body); assert(frame && typeof frame === 'object' && !Array.isArray(frame)); this.sent.push(frame as Record<string, unknown>); }
    close() { this.readyState = 3; this.onclose?.(); }
    open() { this.readyState = 1; assert(this.onopen); this.onopen(); }
    message(value: unknown) { assert(this.onmessage); this.onmessage({data: JSON.stringify(value)}); }
  }
  Object.defineProperty(w, 'WebSocket', {value: FakeSocket});
  class Terminal {
    cols = 80; rows = 24; osc: number[] = []; writes: Array<string | Uint8Array> = [];
    disposed = false;
    textarea = w.document.createElement('textarea');
    key: (event: KeyboardEvent) => boolean = () => { throw Error('Keyboard handler not installed'); };
    input: (data: string) => void = () => { throw Error('Input handler not installed'); };
    parser = {
      registerOscHandler: (code: number, fn: (data: string) => boolean | Promise<boolean>) => { assert.equal(fn(''), true); this.osc.push(code); return {dispose() {}}; },
      registerCsiHandler() { return {dispose() {}}; }, registerDcsHandler() { return {dispose() {}}; }, registerEscHandler() { return {dispose() {}}; },
    };
    constructor(public options: ITerminalOptions & ITerminalInitOnlyOptions) {terms.push(this);}
    loadAddon() {} attachCustomKeyEventHandler(fn: (event: KeyboardEvent) => boolean) {this.key = fn;}
    onData(fn: (data: string) => void) {this.input = fn; return {dispose() {}};}
    open(node: HTMLElement) {node.append(this.textarea);}
    focus() {this.textarea.focus();} dispose() {this.disposed = true;}
    write(bytes: string | Uint8Array, done?: () => void) {this.writes.push(bytes); if (!options.slow) done?.();}
  }
  const api = mountTerminal(root, {expectedUserId: '1', repositoryId: '7', environmentId: env, login: 'original-alice'}, async () => ({Terminal, FitAddon: class {fit() {} activate() {} dispose() {}}}));
  t.after(() => {api.dispose(); dom.window.close();});
  const open = root.querySelector('button'), disconnect = root.querySelectorAll('button')[1];
  assert(open && disconnect);
  return {w, root, api, calls, sockets, terms, before: () => before, open, disconnect};
}
async function ready(f: ReturnType<typeof fixture>) {
  f.open.click(); await tick(); await tick();
  assert.equal(f.sockets.length, 1); const socket = f.sockets[0]; assert(socket); socket.open(); socket.message({type: 'ready'}); return socket;
}
test('mounting is inert and does not own Forgejo markup or beforeunload', t => {
  const f = fixture(t); assert.equal(f.calls.length, 0); assert.equal(f.sockets.length, 0);
  f.w.dispatchEvent(new f.w.Event('beforeunload')); assert.equal(f.before(), 1);
  f.api.dispose(); assert(f.w.document.getElementById('native')); assert.equal(f.root.children.length, 0);
});
test('explicit auth, local renderer, original login and Unicode input/output', async t => {
  const f = fixture(t), socket = await ready(f);
  assert.equal(socket.url, `wss://forge.test/-/soda/api/environments/${env}/terminal`);
  assert.deepEqual(socket.sent[0], {expected_user_id: '1', repository_id: '7', csrf_token: 'synthetic-csrf', cols: 80, rows: 24});
  assert(f.terms[0]);
  assert.deepEqual(f.terms[0].osc, [0, 1, 2, 8, 52]);
  assert.equal(f.calls.length, 2);
  for (const {init} of f.calls) {assert.equal(init.credentials, 'same-origin'); assert.equal(init.redirect, 'error'); assert.equal(new Headers(init.headers).get('X-Soda-Expected-User-ID'), '1');}
  assert(f.terms[0]);
  f.terms[0].input('héllo\r'); assert(socket.sent[1] && typeof socket.sent[1].data === 'string'); assert.equal(Buffer.from(socket.sent[1].data, 'base64').toString(), 'héllo\r');
  socket.message({type: 'output', data: Buffer.from('世界').toString('base64')});
  assert(f.terms[0]);
  assert(f.terms[0].writes[0] !== undefined);
  assert.equal(Buffer.from(f.terms[0].writes[0]).toString(), '世界');
});
for (const event of ['pagehide', 'pageshow']) {
  test(`${event} invalidates, clears scrollback and never reconnects`, async t => {
    const f = fixture(t), socket = await ready(f);
    const e = new f.w.Event(event); if (event === 'pageshow') Object.defineProperty(e, 'persisted', {value: true}); f.w.dispatchEvent(e);
  assert(f.terms[0]);
    assert(f.terms[0].disposed); assert.equal(socket.readyState, 3); assert(f.open.disabled);
    socket.message({type: 'output', data: 'YWJj'}); f.w.dispatchEvent(new f.w.Event('focus')); f.open.click(); await tick();
  assert(f.terms[0]);
    assert.equal(f.sockets.length, 1); assert.equal(f.terms[0].writes.length, 0);
  });
}
test('app and browser visibility changes keep the same transport and renderer', async t => {
  const f = fixture(t), socket = await ready(f);
  f.w.dispatchEvent(new f.w.Event('blur'));
  Object.defineProperty(f.w.document, 'visibilityState', {value: 'hidden'});
  f.w.document.dispatchEvent(new f.w.Event('visibilitychange'));
  socket.message({type: 'output', data: 'YWJj'});
  assert(f.terms[0]);
  assert.equal(socket.readyState, 1); assert(!f.terms[0].disposed);
  assert(f.terms[0]);
  assert(f.terms[0].writes[0] !== undefined);
  assert.equal(Buffer.from(f.terms[0].writes[0]).toString(), 'abc');
  f.w.dispatchEvent(new f.w.Event('focus')); assert.equal(f.sockets.length, 1);
});
test('late terminal readiness does not steal focus from native page controls', async t => {
  const f = fixture(t); f.open.click(); await tick(); await tick();
  const native = f.w.document.getElementById('native'); assert(native); native.focus();
  assert(f.sockets[0]);
  f.sockets[0].open(); f.sockets[0].message({type: 'ready'});
  assert(f.terms[0]);
  assert.equal(f.w.document.activeElement, native); assert(!f.terms[0].disposed);
});
test('hidden terminal view cannot start a shell through a synthetic click', async t => {
  const f = fixture(t); f.root.hidden = true; f.open.click(); await tick();
  assert.equal(f.calls.length, 0); assert.equal(f.sockets.length, 0);
});
test('late session response cannot launch after invalidation', async t => {
  let release: ((response: Response) => void) | undefined; const f = fixture(t, {fetch: () => new Promise(resolve => {release = resolve;})});
  f.open.click(); f.api.invalidate(); assert(release); release(new Response('{}')); await tick();
  assert.equal(f.sockets.length, 0); assert.equal(f.terms.length, 0);
});
test('session or membership mismatch never opens transport', async t => {
  const f = fixture(t, {fetch: async () => new Response(JSON.stringify({user: {id: '2'}}), {headers: {'Content-Type': 'application/json'}})});
  f.open.click(); await tick(); assert.equal(f.sockets.length, 0); assert(f.open.disabled);
});
test('Disconnect, keyboard escape and focus shortcut stay component-local', async t => {
  const f = fixture(t), socket = await ready(f); let escaped = false;
  f.root.addEventListener('keydown', () => {escaped = true;});
  const e = new f.w.KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
  assert(f.terms[0]);
  f.terms[0].textarea.dispatchEvent(e); assert(e.defaultPrevented); assert.equal(escaped, false);
  assert(f.terms[0]);
  f.terms[0].key(new f.w.KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true, shiftKey: true}));
  assert.equal(f.w.document.activeElement, f.disconnect);
  assert(f.terms[0]);
  f.disconnect.click(); assert.equal(socket.sent.at(-1)?.type, 'close'); assert(f.terms[0].disposed);
});
test('input and renderer backlog are bounded', async t => {
  const f = fixture(t, {slow: true}), socket = await ready(f);
  for (let i = 0; i < 65; i++) socket.message({type: 'output', data: Buffer.alloc(4096).toString('base64')});
  assert(f.terms[0]);
  assert(f.terms[0].disposed); assert.equal(socket.readyState, 3);
});
test('oversized paste is refused before encoding or dispatch', async t => {
  const f = fixture(t), socket = await ready(f);
  assert(f.terms[0]);
  f.terms[0].input('x'.repeat(65537));
  assert(f.terms[0]);
  assert.equal(socket.sent.length, 1); assert(f.terms[0].disposed);
});
test('server cannot mutate page using an unknown frame', async t => {
  const f = fixture(t), socket = await ready(f);
  socket.message({type: 'navigate', url: 'https://elsewhere.test'});
  assert(f.terms[0]);
  assert.equal(f.w.location.origin, 'https://forge.test'); assert(f.terms[0].disposed);
});
