// Emitted Lit in Chromium; synthetic API/renderer/transport, not native proof.
import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import payload from '../../internal/release/build/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import type {TerminalFixtureOptions} from './fixtures/terminal-fixture';
const root = path.resolve(import.meta.dirname, '../..'), env = 'p0123456789abcdef01234567', id = 'a'.repeat(32);
let browser: Browser, server: ReturnType<typeof Bun.serve>;
before(async () => {
  const module = await buildForgejoModule(path.join(root, 'tests/frontend/fixtures/terminal-fixture.ts'), 'public/assets/terminal-fixture.js');
  server = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const pathname = new URL(request.url).pathname;
    if (pathname === '/assets/terminal-fixture.js') return new Response(module, {headers: {'Content-Type': 'text/javascript'}});
    const target = 'public' + pathname, source = Object.entries(payload).find(([destination]) => destination === target)?.[1];
    if (source) return new Response(Bun.file(source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : path.join(root, source)), {headers: {'Content-Type': pathname.endsWith('.css') ? 'text/css' : 'text/javascript'}});
    if (pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><button id="native">Native</button><div id="mount" style="display:flex;height:500px;width:800px"></div><script type="module" src="/assets/terminal-fixture.js"></script>', {headers: {'Content-Type': 'text/html'}});
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
async function action(page: Page, name: string | RegExp) {
  if (await page.locator('.soda-menu').getAttribute('open') === null) await page.getByLabel('Terminal actions', {exact: true}).click();
  await page.getByRole('button', {name, exact: true}).click();
}
async function opening(page: Page) {await action(page, /^(Open|Reconnect) terminal$/); await page.waitForFunction(() => window.terminalFixture.sockets.length === 1);}
async function ready(page: Page) {await opening(page); await page.evaluate(() => window.terminalFixture.ready());}
async function end(page: Page) {await action(page, 'End terminal…'); await page.getByRole('dialog').getByRole('button', {name: 'End terminal', exact: true}).click();}
// HTTP native actions are distinct from the new non-starting reservation POST.
const actions = (page: Page) => page.evaluate(() => window.terminalFixture.writes().filter(call => call.url.includes('/terminal-sessions/')));
const inputFrames = (page: Page) => page.evaluate(() => window.terminalFixture.socket().sent.filter(frame => frame.type !== 'resize'));
const existing = {id, login: 'original-alice', repository_id: '7'};

test('mounting is inert and does not own Forgejo markup or beforeunload', async t => {
  const page = await fixture(t);
  assert.deepEqual(await page.evaluate(() => {const f = window.terminalFixture; window.dispatchEvent(new Event('beforeunload')); f.api.dispose(); return {calls: f.calls.length, sockets: f.sockets.length, before: f.before(), native: !!document.getElementById('native'), children: f.root.children.length};}), {calls: 0, sockets: 0, before: 1, native: true, children: 0});
});
test('server locator is published before Create; original bootstrap CSRF and bounded Unicode IO', async t => {
  const page = await fixture(t); await opening(page);
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), id);
  assert.deepEqual(await inputFrames(page), []);
  const reserved = await page.evaluate(() => window.terminalFixture.writes());
  assert.equal(reserved.length, 1); assert(reserved[0]?.url.endsWith('/terminal-sessions'));
  assert.deepEqual(JSON.parse(reserved[0]?.body || '{}'), {cols: 80, rows: 24, name: ''});
  await page.evaluate(() => window.terminalFixture.ready());
  assert.equal(await page.evaluate(() => window.terminalFixture.socket().url), server.url.origin.replace('http:', 'wss:') + `/-/soda/api/environments/${env}/terminal`);
  assert.deepEqual((await inputFrames(page))[0], {action: 'create', id, name: '', expected_user_id: '1', repository_id: '7', csrf_token: 'synthetic-csrf', cols: 80, rows: 24});
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.term().osc), [0, 1, 2, 8, 52]);
  const calls = await page.evaluate(() => window.terminalFixture.calls);
  assert.equal(calls.length, 2); assert(!calls.some(call => call.url.endsWith('/session')));
  for (const call of calls) {assert.equal(call.credentials, 'same-origin'); assert.equal(call.redirect, 'error'); assert.equal(call.headers['x-soda-expected-user-id'], '1');}
  await page.evaluate(() => {const f = window.terminalFixture; f.term().input('héllo\r'); f.socket().message({type: 'output', data: btoa(String.fromCharCode(...new TextEncoder().encode('世界')))});});
  const data = (await inputFrames(page))[1]?.data; assert.equal(typeof data, 'string'); if (typeof data !== 'string') throw Error('input missing');
  assert.equal(Buffer.from(data, 'base64').toString(), 'héllo\r');
  assert.equal(await page.evaluate(() => {const value = window.terminalFixture.term().writes[0]; return typeof value === 'string' ? value : new TextDecoder().decode(value);}), '世界');
});
for (const event of ['pagehide', 'pageshow']) test(`${event} retires access without End or replay`, async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(async event => {const f = window.terminalFixture; window.dispatchEvent(event === 'pageshow' ? new PageTransitionEvent(event, {persisted: true}) : new Event(event)); await f.api.ready; f.socket().message({type: 'output', data: 'YWJj'}); window.dispatchEvent(new Event('focus')); f.root.querySelector('button')?.click();}, event);
  assert.deepEqual(await page.evaluate(() => ({disposed: window.terminalFixture.term().disposed, state: window.terminalFixture.socket().readyState, count: window.terminalFixture.sockets.length, output: window.terminalFixture.term().writes.length})), {disposed: 1, state: 3, count: 1, output: 0});
  assert.equal(await page.getByRole('button', {name: 'Reconnect terminal', includeHidden: true}).count(), 0); assert.deepEqual(await actions(page), []);
});
test('visibility changes preserve renderer and perform no lifetime operation', async t => {
  const page = await fixture(t); await ready(page);
  assert.deepEqual(await page.evaluate(async () => {
    const f = window.terminalFixture, screen = f.root.querySelector('.soda-terminal-screen'), input = f.term().textarea;
    window.dispatchEvent(new Event('blur')); f.api.setVisible(false); f.root.hidden = true;
    f.socket().message({type: 'output', data: 'YWJj'}); f.root.hidden = false; f.api.setVisible(true); await f.api.ready;
    return {state: f.socket().readyState, disposed: f.term().disposed, count: f.sockets.length, screen: screen === f.root.querySelector('.soda-terminal-screen'), input: input === f.term().textarea};
  }), {state: 1, disposed: 0, count: 1, screen: true, input: true});
  assert.deepEqual(await actions(page), []);
  assert.equal(await page.getByText('Keep for two hours', {exact: true}).count(), 0);
  assert.equal(await page.getByText('Continue working', {exact: true}).count(), 0);
});
test('late readiness does not steal native focus', async t => {
  const page = await fixture(t); await opening(page); await page.locator('#native').focus(); await page.evaluate(() => window.terminalFixture.ready());
  assert.equal(await page.evaluate(() => document.activeElement?.id), 'native');
});
test('focus-triggered retirement cannot leave a late observer', async t => {
  const page = await fixture(t); await opening(page);
  await page.evaluate(async () => {const f = window.terminalFixture; f.term().textarea.addEventListener('focus', () => f.api.dispose(), {once: true}); await f.ready();});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.observers()), {observed: 0, disconnected: 0});
  assert.deepEqual(await actions(page), []);
});
test('hidden controls and late renderer import cannot dispatch', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {const f = window.terminalFixture; f.root.hidden = true; f.button('Open terminal').click(); await f.api.ready;});
  assert.equal(await page.evaluate(() => window.terminalFixture.calls.length), 0);
  await page.evaluate(async () => {const f = window.terminalFixture; f.root.hidden = false; let release: (() => void) | undefined;
    f.waitRenderer(new Promise(resolve => {release = resolve;})); f.button('Open terminal').click(); f.api.dispose(); release?.(); await f.api.ready;});
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length + window.terminalFixture.terms.length), 0);
});
test('late reservation result cannot create a socket after retirement', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const f = window.terminalFixture; let release: ((value: Response) => void) | undefined;
    await new Promise<void>(entered => {f.setReply(async () => new Promise(resolve => {release = resolve; entered();})); f.button('Open terminal').click();});
    f.api.invalidate(); release?.(Response.json({id: 'a'.repeat(32)})); await f.api.ready;
  });
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
});
for (const options of [{user: '2'}, {login: 'other'}, {responseStatus: 401}, {responseStatus: 403}]) test(`actual reservation authorization rejects before native transport: ${JSON.stringify(options)}`, async t => {
  const page = await fixture(t, options); await action(page, 'Open terminal');
  await page.getByText('Could not authorize or attach.', {exact: false}).waitFor(); assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
});
test('End is separately confirmed, uses original CSRF and survives attachment closing first', async t => {
  const page = await fixture(t); await ready(page);
  assert.deepEqual(await page.evaluate(() => {const f = window.terminalFixture; let escaped = false; f.root.addEventListener('keydown', () => {escaped = true;});
    const escape = new KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true}); f.term().textarea.dispatchEvent(escape);
    f.term().key(new KeyboardEvent('keydown', {key: 'Enter', ctrlKey: true, shiftKey: true, cancelable: true}));
    return {prevented: escape.defaultPrevented, escaped, focus: document.activeElement?.getAttribute('aria-label')};
  }), {prevented: true, escaped: false, focus: 'Terminal actions'});
  await action(page, 'End terminal…'); assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Cancel');
  assert.deepEqual(await actions(page), []);
  await page.evaluate(() => {const button = window.terminalFixture.button('End terminal'); button.click(); button.click();});
  await page.getByText('Native cleanup confirmed', {exact: false}).waitFor();
  const sent = await actions(page); assert.equal(sent.length, 1); assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'end'});
  assert(sent[0]?.url.endsWith('/terminal-sessions/' + id)); assert.equal(sent[0]?.headers['x-csrf-token'], 'synthetic-csrf');
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1); assert.equal(await page.evaluate(() => window.terminalFixture.locator()), null);
});
test('unconfirmed End keeps the locator, but does not prevent a later explicit action', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(() => window.terminalFixture.setReply(async call => call.method === 'POST' ? Response.json({}) : null));
  await end(page); await page.getByText('End was not confirmed.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), id);
  await page.evaluate(() => window.terminalFixture.setReply(undefined)); await end(page);
  await page.getByText('Native cleanup confirmed', {exact: false}).waitFor(); assert.equal((await actions(page)).length, 2);
});
test('native focus and hidden views cannot forward input', async t => {
  const page = await fixture(t); await ready(page); await page.locator('#native').focus();
  await page.evaluate(() => window.terminalFixture.term().input('native field')); assert.equal((await inputFrames(page)).length, 1);
  await page.evaluate(() => {const f = window.terminalFixture; f.api.focus(); f.api.setVisible(false); f.term().input('hidden');}); assert.equal((await inputFrames(page)).length, 1);
  await page.evaluate(() => {const f = window.terminalFixture; f.api.setVisible(true); f.api.focus(); f.term().input('focused');}); assert.equal((await inputFrames(page)).length, 2);
});
for (const state of ['ended', undefined]) test(`confirmed native absence/exit cannot reopen the same ID (${state})`, async t => {
  const page = await fixture(t, {locator: {kind: 'existing', id}, ...(state ? {existing: {...existing, state}} : {})});
  await page.evaluate(() => window.terminalFixture.api.restore());
  const count = await page.evaluate(() => window.terminalFixture.calls.length);
  await page.evaluate(async () => {const f = window.terminalFixture; await f.api.open(); await f.api.restore();});
  assert.equal(await page.evaluate(() => window.terminalFixture.calls.length), count); assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => window.terminalFixture.locator()), null); assert.deepEqual(await actions(page), []);
});
test('unavailable lookup is not absence; an explicit refresh may recover', async t => {
  const page = await fixture(t, {locator: {kind: 'existing', id}, existing});
  await page.evaluate(async () => {const f = window.terminalFixture; f.setReply(async () => new Response(null, {status: 503})); await f.api.restore();});
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.locators), []); assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
  await page.evaluate(async () => {const f = window.terminalFixture; f.setReply(undefined); await f.api.restore(); f.socket().open();});
  assert.deepEqual((await inputFrames(page))[0], {action: 'attach', id, expected_user_id: '1', repository_id: '7', csrf_token: 'synthetic-csrf', cols: 80, rows: 24});
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.writes()), []);
});
test('lost creation reply uses the issued ID, not another Create or request namespace', async t => {
  const page = await fixture(t); await opening(page);
  await page.evaluate(async () => {const f = window.terminalFixture; f.socket().open(); f.api.disconnect(); await f.api.restore(); f.socket().open();});
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 2);
  assert.deepEqual(await page.evaluate(() => window.terminalFixture.sockets.map(s => s.sent[0]?.action)), ['create', 'attach']);
  assert.equal((await inputFrames(page))[0]?.id, id);
  assert(!await page.evaluate(() => window.terminalFixture.calls.some(c => c.url.includes('terminal-attempts'))));
});
test('late open after hiding sends no Create, but keeps the pre-issued locator', async t => {
  const page = await fixture(t); await opening(page);
  await page.evaluate(() => {const f = window.terminalFixture; f.root.hidden = true; f.socket().open();});
  assert.deepEqual(await inputFrames(page), []); assert.equal(await page.evaluate(() => window.terminalFixture.locator()), id);
});
test('missing, retired pending and malformed locators refuse before mounting', async t => {
  const page = await fixture(t);
  assert.deepEqual(await page.evaluate(() => {const f = window.terminalFixture; return [undefined, null, 'pending', {}, {kind: 'other'}, {kind: 'existing', id: 'bad'}, {kind: 'pending', requestId: 'b'.repeat(32)}].map(value => {try {f.mountInvalidLocator(value); return false;} catch {return true;}});}), Array(7).fill(true));
  assert.equal(await page.locator('soda-terminal').count(), 1);
});
for (const state of ['opening', 'ending']) test(`${state} is an observation, not a permanent admission flag`, async t => {
  const page = await fixture(t, {locator: {kind: 'existing', id}, existing: {...existing, state}}); await page.evaluate(() => window.terminalFixture.api.restore());
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0); assert.deepEqual(await page.evaluate(() => window.terminalFixture.writes()), []);
  assert.match(await page.locator('#mount').innerText(), /transition is still in progress/);
});
test('observed writer refuses takeover without creating work', async t => {
  const page = await fixture(t, {locator: {kind: 'existing', id}, existing: {...existing, attached: true}}); await page.evaluate(() => window.terminalFixture.api.restore());
  await page.getByText('An existing writer is attached.', {exact: false}).waitFor(); assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 0);
});
test('retired renderer cannot send through a successor attachment', async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(async () => {const f = window.terminalFixture; f.api.disconnect(); await f.api.restore(); await f.ready(); f.terms[0]?.input('stale input');});
  assert.equal(await page.evaluate(() => window.terminalFixture.sockets.length), 2); assert.equal((await inputFrames(page)).length, 1);
});
test('disposal and late socket callbacks close exactly once', async t => {
  const page = await fixture(t); await ready(page); await action(page, 'End terminal…');
  await page.evaluate(async () => {const f = window.terminalFixture; f.button('End terminal').click(); f.api.dispose(); f.socket().onclose?.(); f.socket().message({type: 'ready'}); f.term().input('not replayed'); await f.api.ready;});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1); assert.equal(await page.evaluate(() => window.terminalFixture.socket().closed), 1); assert.equal((await inputFrames(page)).length, 1);
});
test('input and output queues stay bounded', async t => {
  const page = await fixture(t, {slow: true}); await ready(page);
  await page.evaluate(() => {for (let i = 0; i < 65; i++) window.terminalFixture.socket().message({type: 'output', data: btoa('\0'.repeat(4096))});});
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
});
for (const overload of ['paste', 'transport']) test(`${overload} overload detaches without replay`, async t => {
  const page = await fixture(t); await ready(page);
  await page.evaluate(kind => {const f = window.terminalFixture; if (kind === 'transport') f.socket().bufferedAmount = 65537; f.term().input(kind === 'paste' ? 'x'.repeat(65537) : 'blocked');}, overload);
  assert.equal((await inputFrames(page)).length, 1); assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1);
});
for (const frame of [null, {type: 'ready', extra: true}, {type: 'output', data: ''}, {type: 'output', data: '!'}, {type: 'output', data: 'YQ==', extra: true}, {type: 'session', id}, {type: 'navigate', url: 'https://elsewhere.test'}, 'x'.repeat(32769)]) test(`invalid frame: ${JSON.stringify(frame).slice(0,70)}`, async t => {
  const page = await fixture(t); await ready(page); await page.evaluate(frame => window.terminalFixture.socket().message(frame), frame);
  assert.equal(await page.evaluate(() => window.terminalFixture.term().disposed), 1); assert.equal((await inputFrames(page)).length, 1); assert.equal(new URL(page.url()).origin, server.url.origin);
});
