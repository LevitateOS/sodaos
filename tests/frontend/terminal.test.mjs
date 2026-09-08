import test from 'node:test';
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mountTerminal} from '../../appliance/forgejo/public/assets/sodaspaces-terminal.js';
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
const {JSDOM} = require('jsdom');
const env = 'p0123456789abcdef01234567';
const tick = () => new Promise(resolve => setTimeout(resolve, 0));
function fixture(t, options = {}) {
  const dom = new JSDOM('<button id="native">Native</button><div id="mount"></div>', {url: 'https://forge.test/alice/repo', pretendToBeVisual: true});
  const w = dom.window, root = w.document.getElementById('mount');
  w.document.hasFocus = () => true;
  let before = 0;
  w.addEventListener('beforeunload', () => before++);
  const calls = [], sockets = [], terms = [];
  w.fetch = async (url, init) => {
    calls.push({url, init});
    if (options.fetch) return options.fetch(url, init);
    return new Response(JSON.stringify(url.endsWith('/session') ? {user: {id: '1'}, csrf_token: 'synthetic-csrf', forgejo_url: 'https://forge.test'} : {environment: {id: env, repository_id: '7', provisioned: true}, login: 'original-alice'}), {headers: {'Content-Type': 'application/json'}});
  };
  w.ResizeObserver = class {observe() {} disconnect() {}};
  w.WebSocket = class {
    readyState = 0; bufferedAmount = 0; sent = [];
    constructor(url) { this.url = String(url); sockets.push(this); }
    send(body) { this.sent.push(JSON.parse(body)); }
    close() { this.readyState = 3; this.onclose?.(); }
    open() { this.readyState = 1; this.onopen(); }
    message(value) { this.onmessage({data: JSON.stringify(value)}); }
  };
  class Terminal {
    cols = 80; rows = 24; osc = []; writes = []; parser = {registerOscHandler: (code, fn) => { assert.equal(fn(), true); this.osc.push(code); }};
    constructor(options) {this.options = options; terms.push(this);}
    loadAddon() {} attachCustomKeyEventHandler(fn) {this.key = fn;} onData(fn) {this.input = fn;}
    open(node) {this.textarea = w.document.createElement('textarea'); node.append(this.textarea);}
    focus() {this.textarea.focus();} dispose() {this.disposed = true;}
    write(bytes, done) {this.writes.push(bytes); if (!options.slow) done();}
  }
  const api = mountTerminal(root, {expectedUserId: '1', repositoryId: '7', environmentId: env, login: 'original-alice'}, async () => ({Terminal, FitAddon: class {fit() {}}}));
  t.after(() => {api.dispose(); dom.window.close();});
  return {w, root, api, calls, sockets, terms, before: () => before, open: root.querySelector('button'), disconnect: root.querySelectorAll('button')[1]};
}
async function ready(f) {
  f.open.click(); await tick(); await tick();
  assert.equal(f.sockets.length, 1); const socket = f.sockets[0]; socket.open(); socket.message({type: 'ready'}); return socket;
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
  assert.deepEqual(f.terms[0].osc, [0, 1, 2, 8, 52]);
  assert.equal(f.calls.length, 2);
  for (const {init} of f.calls) {assert.equal(init.credentials, 'same-origin'); assert.equal(init.redirect, 'error'); assert.equal(init.headers['X-Soda-Expected-User-ID'], '1');}
  f.terms[0].input('héllo\r'); assert.equal(Buffer.from(socket.sent[1].data, 'base64').toString(), 'héllo\r');
  socket.message({type: 'output', data: Buffer.from('世界').toString('base64')});
  assert.equal(Buffer.from(f.terms[0].writes[0]).toString(), '世界');
});
for (const event of ['blur', 'pagehide', 'pageshow', 'visibilitychange']) {
  test(`${event} invalidates, clears scrollback and never reconnects`, async t => {
    const f = fixture(t), socket = await ready(f);
    if (event === 'visibilitychange') {Object.defineProperty(f.w.document, 'visibilityState', {value: 'hidden'}); f.w.document.dispatchEvent(new f.w.Event(event));}
    else {const e = new f.w.Event(event); if (event === 'pageshow') Object.defineProperty(e, 'persisted', {value: true}); f.w.dispatchEvent(e);}
    assert(f.terms[0].disposed); assert.equal(socket.readyState, 3); assert(f.open.disabled);
    socket.message({type: 'output', data: 'YWJj'}); f.w.dispatchEvent(new f.w.Event('focus')); f.open.click(); await tick();
    assert.equal(f.sockets.length, 1); assert.equal(f.terms[0].writes.length, 0);
  });
}
test('late session response cannot launch after invalidation', async t => {
  let release; const f = fixture(t, {fetch: () => new Promise(resolve => {release = resolve;})});
  f.open.click(); f.api.invalidate(); release({ok: true, json: async () => ({})}); await tick();
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
  f.terms[0].textarea.dispatchEvent(e); assert(e.defaultPrevented); assert.equal(escaped, false);
  f.terms[0].key(new f.w.KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true, shiftKey: true}));
  assert.equal(f.w.document.activeElement, f.disconnect);
  f.disconnect.click(); assert.equal(socket.sent.at(-1).type, 'close'); assert(f.terms[0].disposed);
});
test('input and renderer backlog are bounded', async t => {
  const f = fixture(t, {slow: true}), socket = await ready(f);
  for (let i = 0; i < 65; i++) socket.message({type: 'output', data: Buffer.alloc(4096).toString('base64')});
  assert(f.terms[0].disposed); assert.equal(socket.readyState, 3);
});
test('oversized paste is refused before encoding or dispatch', async t => {
  const f = fixture(t), socket = await ready(f);
  f.terms[0].input('x'.repeat(65537));
  assert.equal(socket.sent.length, 1); assert(f.terms[0].disposed);
});
test('server cannot mutate page using an unknown frame', async t => {
  const f = fixture(t), socket = await ready(f);
  socket.message({type: 'navigate', url: 'https://elsewhere.test'});
  assert.equal(f.w.location.origin, 'https://forge.test'); assert(f.terms[0].disposed);
});
