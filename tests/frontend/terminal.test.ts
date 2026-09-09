// Emitted Lit components in Chromium; API, renderer and transport doubles, not native proof.
import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import type {TerminalFixtureOptions} from './fixtures/terminal-fixture';
const root = path.resolve(import.meta.dirname, '../..'), env = 'p0123456789abcdef01234567', id = 'a'.repeat(32);
let browser: Browser, server: ReturnType<typeof Bun.serve>;
before(async () => {
  const module = await buildForgejoModule(path.join(root, 'tests/frontend/fixtures/terminal-fixture.ts'), 'public/assets/terminal-fixture.js');
  server = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/tests/frontend/fixtures/terminal-fixture.js') return new Response(module, {headers: {'Content-Type': 'text/javascript'}});
    const target = 'public' + pathname.replace(/^\/appliance\/forgejo\/public/, ''), source = Object.entries(payload).find(([destination]) => destination === target)?.[1];
    if (source) return new Response(Bun.file(source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : path.join(root, source)), {headers: {'Content-Type': pathname.endsWith('.css') ? 'text/css' : 'text/javascript'}});
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><button id="native">Native</button><div id="mount" style="display:flex;height:500px;width:800px"></div><script type="module" src="/tests/frontend/fixtures/terminal-fixture.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {await browser?.close(); server?.stop(true);});
async function fixture(t: TestContext, options: TerminalFixtureOptions = {}) {
  const page = await browser.newPage(), errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  t.after(async () => {await page.close(); assert.deepEqual(errors, []);});
  await page.goto(server.url.href); await page.waitForFunction(() => !!window.createTerminalFixture);
  await page.evaluate(async options => {window.terminalFixture = window.createTerminalFixture(options); await window.terminalFixture.api.ready;}, options);
  return page;
}
async function opening(page: Page) {
  await page.locator('.soda-terminal-toolbar > button').first().click();
  await page.waitForFunction(() => window.terminalFixture.sockets.length === 1);
}
async function ready(page: Page) {await opening(page); await page.evaluate(() => window.terminalFixture.ready());}
const writes = (page: Page) => page.evaluate(() => window.terminalFixture.writes());
const inputFrames = (page: Page) => page.evaluate(() => window.terminalFixture.socket().sent.filter(frame => frame.type !== 'resize'));
const existing = {id, login: 'original-alice', repository_id: '7'};

test('mounting is inert and does not own Forgejo markup or beforeunload', async t => {
  const page = await fixture(t);
  assert.deepEqual(await page.evaluate(() => {const f = window.terminalFixture; window.dispatchEvent(new Event('beforeunload')); f.api.dispose(); return {calls: f.calls.length, sockets: f.sockets.length, before: f.before(), native: !!document.getElementById('native'), children: f.root.children.length};}), {calls: 0, sockets: 0, before: 1, native: true, children: 0});
});
test('explicit auth, local renderer, original login and Unicode input/output', async t => {
  const page = await fixture(t); await ready(page);
  assert.equal(await page.evaluate(() => window.terminalFixture.socket().url), server.url.origin.replace('http:', 'wss:') + `/-/soda/api/environments/${env}/terminal`);
  const first = (await inputFrames(page))[0]; assert(first); assert.match(String(first.request_id), /^[0-9a-f]{32}$/);
  assert.deepEqual({...first, request_id: undefined}, {action: 'create', request_id: undefined, expected_user_id: '1', repository_id: '7', csrf_token: 'synthetic-csrf', cols: 80, rows: 24});
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.term().osc), [0, 1, 2, 8, 52]);
  const calls = await page.evaluate(() => window.terminalFixture.calls); assert.equal(calls.length, 2);
  for (const call of calls) {assert.equal(call.credentials, 'same-origin'); assert.equal(call.redirect, 'error'); assert.equal(call.headers['x-soda-expected-user-id'], '1');}
  await page.evaluate(() => {window.terminalFixture.term().input('héllo\r'); window.terminalFixture.socket().message({type: 'output', data: btoa(String.fromCharCode(...new TextEncoder().encode('世界')))});});
  const data = (await inputFrames(page))[1]?.data; assert.equal(typeof data, 'string'); if (typeof data !== 'string') throw Error('input missing');
  assert.equal(Buffer.from(data, 'base64').toString(), 'héllo\r');
  assert.equal(await page.evaluate(() => {const value = window.terminalFixture.term().writes[0]; return typeof value === 'string' ? value : new TextDecoder().decode(value);}), '世界');
});
for (const event of ['pagehide', 'pageshow']) test(`${event} retires this renderer without End or input replay`, async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(async event => {const f = window.terminalFixture; window.dispatchEvent(event === 'pageshow' ? new PageTransitionEvent(event, {persisted: true}) : new Event(event)); await f.api.ready; f.socket().message({type: 'output', data: 'YWJj'}); window.dispatchEvent(new Event('focus')); f.root.querySelector('button')?.click();}, event);
  assert.deepEqual(await page.evaluate(() => ({disposed: window.terminalFixture.term().disposed, state: window.terminalFixture.socket().readyState, count: window.terminalFixture.sockets.length, output: window.terminalFixture.term().writes.length})), {disposed: 1, state: 3, count: 1, output: 0});
  assert(await page.locator('.soda-terminal-toolbar > button').first().isDisabled()); assert.deepEqual(await writes(page), []);
});
test('app visibility and reactive updates keep the same transport, screen and renderer', async t => {
  const page = await fixture(t); await ready(page);
  assert.deepEqual(await page.evaluate(async () => {
    const f = window.terminalFixture, screen = f.root.querySelector('.soda-terminal-screen'), input = f.term().textarea;
    window.dispatchEvent(new Event('blur')); Object.defineProperty(document, 'visibilityState', {value: 'hidden', configurable: true}); document.dispatchEvent(new Event('visibilitychange'));
    f.socket().message({type: 'output', data: 'YWJj'}); f.root.hidden = true; await f.api.retain(); f.root.hidden = false; await f.api.ready;
    window.dispatchEvent(new Event('focus')); const output = f.term().writes[0];
    return {state: f.socket().readyState, disposed: f.term().disposed, count: f.sockets.length, screen: screen === f.root.querySelector('.soda-terminal-screen'), input: input === f.term().textarea, output: new TextDecoder().decode(output instanceof Uint8Array ? output : undefined)};
  }), {state: 1, disposed: 0, count: 1, screen: true, input: true, output: 'abc'});
});
test('late terminal readiness does not steal native focus', async t => {
  const page = await fixture(t); await opening(page); await page.locator('#native').focus(); await page.evaluate(() => window.terminalFixture.ready());
  assert.equal(await page.evaluate(() => document.activeElement?.id), 'native'); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 0);
});
test('native focus handlers retiring the component cannot leave a late observer', async t => {
  const page = await fixture(t); await opening(page);
  await page.evaluate(async () => {const f = window.terminalFixture; f.term().textarea.addEventListener('focus', () => f.api.dispose(), {once: true}); await f.ready();});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.observers()), {observed: 0, disconnected: 0});
  assert.deepEqual(await writes(page), []);
});

test('hidden terminal view cannot start through a synthetic click', async t => {
  const page = await fixture(t); await page.evaluate(async () => {const f = window.terminalFixture; f.root.hidden = true; f.button('Open terminal').click(); await f.api.ready;});
  assert.equal(await page.evaluate(() => window.terminalFixture.calls.length), 0);
});
test('late session response cannot launch after invalidation', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    let release: ((response: Response) => void) | undefined; const f = window.terminalFixture;
    f.setReply(async () => new Promise(resolve => {release = resolve;})); f.button('Open terminal').click(); f.api.invalidate();
    if (!release) throw Error('request not pending'); release(Response.json({})); await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length + window.terminalFixture.terms.length), 0);
});
for (const options of [{user: '2'}, {login: 'other'}, {responseStatus: 401}, {responseStatus: 403}]) test(`session/membership rejection never opens transport: ${JSON.stringify(options)}`, async t => {
  const page = await fixture(t, options); await page.locator('button', {hasText: 'Open terminal'}).click();
  await page.getByText('Could not authorize or attach.', {exact: false}).waitFor(); assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
});
test('End is a separate authenticated action; Escape and control focus stay local', async t => {
  const page = await fixture(t); await ready(page);
  assert.deepEqual(await page.evaluate(() => {
    const f = window.terminalFixture; let escaped = false; f.root.addEventListener('keydown', () => {escaped = true;});
    const escape = new KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true}); f.term().textarea.dispatchEvent(escape);
    const control = new KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true, shiftKey: true, cancelable: true}); f.term().key(control);
    return {prevented: escape.defaultPrevented, escaped, focus: document.activeElement?.textContent};
  }), {prevented: true, escaped: false, focus: 'End terminal'});
  await page.getByRole('button', {name: 'End terminal', exact: true}).click(); await page.getByText('Native cleanup confirmed', {exact: false}).waitFor();
  const sent = await writes(page); assert.equal(sent.length, 1); assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'end'}); assert(sent[0]?.url.endsWith('/terminal-sessions/' + id));
  assert.equal(sent[0]?.headers['x-csrf-token'], 'synthetic-csrf');
  assert.equal((await inputFrames(page)).length, 1); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.equal(await page.evaluate(() => window.terminalFixture.storage()), null);
});
test('keyboard exit remains available while lifetime controls await authorization', async t => {
  const page = await fixture(t); await ready(page);
  assert.equal(await page.evaluate(async () => {
    const f = window.terminalFixture; let release: ((response: Response) => void) | undefined;
    f.setReply(async () => new Promise(resolve => {release = resolve;}));
    f.button('Keep for two hours').click(); await f.api.ready;
    f.term().key(new KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true, shiftKey: true, cancelable: true}));
    const role = document.activeElement?.getAttribute('role');
    f.api.dispose(); release?.(Response.json({user: {id: '1'}, csrf_token: 'synthetic-csrf', forgejo_url: location.origin}));
    return role;
  }), 'status');
  assert.deepEqual(await writes(page), []);
});

test('input and renderer backlog are bounded', async t => {
  const page = await fixture(t, {slow: true}); await ready(page);
  await page.evaluate(() => {for (let i = 0; i < 65; i++) window.terminalFixture.socket().message({type: 'output', data: btoa('\0'.repeat(4096))});});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1); assert.equal(await page.evaluate(() => window.terminalFixture.socket().readyState), 3);
});
test('oversized paste is refused before encoding or dispatch', async t => {
  const page = await fixture(t); await ready(page); await page.evaluate(() => window.terminalFixture.term().input('x'.repeat(65537)));
  assert.equal((await inputFrames(page)).length, 1); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
});
test('known locator restores attach-only without credentials in storage or automatic Return', async t => {
  const page = await fixture(t, {saved: id, existing}); assert.equal(await page.evaluate(() => window.terminalFixture.calls.length), 0);
  await page.evaluate(() => window.terminalFixture.api.restore()); await page.evaluate(() => window.terminalFixture.socket().open());
  assert.equal((await inputFrames(page))[0]?.action, 'attach'); assert.equal((await inputFrames(page))[0]?.id, id);
  assert.deepEqual(await writes(page), []); assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id);
});
for (const state of ['unconfirmed', 'ending']) test(`${state} cleanup reserves locator without attachment or replacement`, async t => {
  const page = await fixture(t, {saved: id, existing: {...existing, state}}); await page.evaluate(() => window.terminalFixture.api.restore());
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0); assert.deepEqual(await writes(page), []);
  assert.match(await page.locator('#mount').innerText(), /slot is reserved/); assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id);
});
test('missing stored target never falls back to creation', async t => {
  const page = await fixture(t, {saved: id}); await page.evaluate(() => window.terminalFixture.api.restore());
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0); assert.deepEqual(await writes(page), []); assert.match(await page.locator('#mount').innerText(), /outcome remains unknown/); assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id);
});
test('lost creation response is looked up, never retried', async t => {
  const page = await fixture(t, {saved: 'pending:' + 'b'.repeat(32)});
  for (let i = 0; i < 2; i++) {await page.getByRole('button', {name: 'Find pending terminal'}).click(); await page.getByText('no creation was retried', {exact: false}).waitFor();}
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0); assert.deepEqual(await writes(page), []);
});
test('lost create is recovered by exact request correlation, never newest-session lookup', async t => {
  const page = await fixture(t, {saved: 'pending:' + 'b'.repeat(32), existing});
  await opening(page); await page.evaluate(() => window.terminalFixture.socket().open());
  assert.equal((await inputFrames(page))[0]?.action, 'attach'); assert.equal((await inputFrames(page))[0]?.id, id);
  assert(await page.evaluate(() => window.terminalFixture.calls.some(call => call.url.endsWith('/terminal-attempts/' + 'b'.repeat(32)))));
});
test('another attempt cannot be substituted for a lost creation result', async t => {
  const page = await fixture(t, {saved: 'pending:' + 'c'.repeat(32), existing});
  await page.getByRole('button', {name: 'Find pending terminal'}).click();
  await page.getByText('Could not authorize or attach.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => window.terminalFixture.storage()), 'pending:' + 'c'.repeat(32));
});
test('legacy uncorrelated pending locators never select a session', async t => {
  const page = await fixture(t, {saved: 'pending', existing});
  await page.getByRole('button', {name: 'Find pending terminal'}).click();
  await page.getByText('Legacy creation outcome', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => window.terminalFixture.calls.some(call => call.url.includes('/terminal-'))), false);
});
test('End acknowledgment plus absent receipt preserves uncertainty and exact ID', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(() => window.terminalFixture.setReply(async call => call.method === 'GET' && call.url.includes('/terminal-sessions/') ? Response.json({terminal: null}) : null));
  await page.getByRole('button', {name: 'End terminal', exact: true}).click();
  await page.getByText('absent receipt is not cleanup proof', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id);
  await page.getByRole('button', {name: 'Reconnect terminal'}).click(); await page.getByText('outcome remains unknown', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 1);
});
test('attach readiness requires this socket’s new locator and writer generation', async t => {
  const page = await fixture(t, {saved: id, existing});
  await page.evaluate(async () => {const f = window.terminalFixture; await f.api.restore(); f.socket().open(); f.socket().message({type: 'ready'}); await f.api.ready;});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id);
  assert.equal(await page.locator('.is-connected').count(), 0);
});
test('Hide carries this attachment locator and is distinct from Keep', async t => {
  const page = await fixture(t); await ready(page); await page.evaluate(() => window.terminalFixture.api.retain());
  const sent = await writes(page); assert.equal(sent.length, 1);
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'hide', attachment_id: '1'.padStart(32, '0')});
});
test('server cannot mutate page using an unknown frame', async t => {
  const page = await fixture(t); await ready(page); await page.evaluate(() => window.terminalFixture.socket().message({type: 'navigate', url: 'https://elsewhere.test'}));
  assert.equal(new URL(page.url()).origin, server.url.origin); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
});
test('late renderer import after Hide/dispose cannot open a transport', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.terminalFixture; let release: (() => void) | undefined;
    f.waitRenderer(new Promise(resolve => {release = resolve;})); f.button('Open terminal').click(); f.root.hidden = true; f.api.dispose(); release?.(); await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.terms.length + window.terminalFixture.sockets.length), 0);
});
test('late socket open after Hide sends no creation frame', async t => {
  const page = await fixture(t); await opening(page);
  await page.evaluate(() => {const f = window.terminalFixture; f.root.hidden = true; f.socket().open();});
  assert.deepEqual(await inputFrames(page), []); assert.equal(await page.evaluate(() => window.terminalFixture.storage()), null);
});
test('rapid lifetime commands dispatch once; reactive control updates preserve renderer', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(() => {const button = window.terminalFixture.button('Keep for two hours'); button.click(); button.click();});
  await page.getByText('Retained until', {exact: false}).waitFor();
  const sent = await writes(page); assert.equal(sent.length, 1); assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'retain', seconds: 7200});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 0); assert.equal(await page.evaluate(() => window.terminalFixture.terms.length), 1);
  await page.getByRole('button', {name: 'Continue working'}).click(); await page.getByText('Active as original-alice', {exact: false}).waitFor();
  assert.equal((await writes(page)).length, 2); assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 1);
});
test('End acknowledgment/disposal/socket callbacks clean up exactly once', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(async () => {const f = window.terminalFixture; f.button('End terminal').click(); f.api.dispose(); f.socket().onclose?.(); f.socket().message({type: 'ready'}); f.term().input('not replayed'); await f.api.ready;});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1); assert.equal(await page.evaluate(() => window.terminalFixture.socket().closed), 1);
  assert.equal((await inputFrames(page)).length, 1);
});
test('an attachment refused by another writer never creates a replacement', async t => {
  const page = await fixture(t, {saved: id, existing}); await page.evaluate(() => window.terminalFixture.api.restore());
  await page.evaluate(() => {const f = window.terminalFixture; f.socket().open(); f.socket().message({type: 'closed', reason: 'unavailable'});});
  assert.equal((await inputFrames(page))[0]?.action, 'attach'); assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id);
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 1); assert.deepEqual(await writes(page), []);
});
test('failed creation dispatch preserves uncertainty, not an automatic retry', async t => {
  const page = await fixture(t); await opening(page);
  await page.evaluate(() => {const f = window.terminalFixture; f.socket().send = () => {throw Error('synthetic send failure');}; f.socket().open();});
  assert.match(await page.evaluate(() => window.terminalFixture.storage()) || '', /^pending:[0-9a-f]{32}$/);
  await page.getByRole('button', {name: 'Find pending terminal'}).click(); await page.getByText('no creation was retried', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 1);
});
test('unconfirmed End does not forget its ID or claim native cleanup', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(() => window.terminalFixture.setReply(async call => call.method === 'POST' ? Response.json({ending: false}) : null));
  await page.getByRole('button', {name: 'End terminal', exact: true}).click(); await page.getByText('Terminal lifetime action was not confirmed.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.storage()), id); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 0);
});
test('retired renderer input cannot reach a successor attachment', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(async () => {const f = window.terminalFixture; f.api.disconnect(); await f.api.restore(); f.socket().open(); f.socket().message({type: 'session', id: 'a'.repeat(32)}); f.socket().message({type: 'ready'}); await f.api.ready; f.terms[0]?.input('stale input');});
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 2); assert.equal((await inputFrames(page)).length, 1);
});
for (const frame of [null, {type: 'ready', extra: true}, {type: 'output', data: ''}, {type: 'output', data: '!'}, {type: 'output', data: 'YQ==', extra: true}, 'x'.repeat(32769)]) test(`invalid terminal frame: ${JSON.stringify(frame).slice(0, 70)}`, async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(frame => window.terminalFixture.socket().message(frame), frame);
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1); assert.equal((await inputFrames(page)).length, 1);
});
test('overloaded outgoing transport detaches without replay', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(() => {window.terminalFixture.socket().bufferedAmount = 65537; window.terminalFixture.term().input('blocked');});
  assert.equal((await inputFrames(page)).length, 1); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
});
