import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import type {State} from './fixtures/drawer-fixture';

const root = path.resolve(import.meta.dirname, '../..');
let browser: Browser;
let server: ReturnType<typeof Bun.serve>;
before(async () => {
  const fixture = await buildForgejoModule(path.join(root, 'tests/frontend/fixtures/drawer-fixture.ts'), 'public/assets/drawer-fixture.js');
  const footer = (await Bun.file(path.join(root, 'appliance/forgejo/templates/custom/footer.tmpl')).text()).split('{{if .IsSigned}}\n<div id="soda-notification-preview"')[0];
  assert(footer);
  server = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(req) {
    const url = new URL(req.url);
    if (url.pathname === '/assets/drawer-fixture.js') return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
    const target = 'public' + url.pathname;
    const source = Object.entries(payload).find(([dest]) => dest === target)?.[1];
    if (source?.startsWith('@build/forgejo-js/')) return new Response(Bun.file(path.join(root, '.artifacts/forgejo-js', path.basename(source))), {headers: {'Content-Type': 'text/javascript'}});
    if (source && !source.startsWith('@build/')) return new Response(Bun.file(path.join(root, source)));
    if (url.pathname === '/native') {
      const signed = url.searchParams.get('anonymous') !== '1';
      const markup = footer.replace(/{{AssetUrlPrefix}}/g, '/assets').replace(/{{AppSubUrl}}/g, '').replace(/{{\.Repository.ID}}/g, '7')
        .replace(/{{if \.IsSigned}}true{{else}}false{{end}}/g, String(signed)).replace(/{{if \.IsSigned}}{{\.SignedUserID}}{{end}}/g, signed ? '1' : '').replace(/{{[\s\S]*?}}/g, '');
      return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/sodaspaces.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><div class="repo-header"><div class="repo-buttons"><button id="native">Native</button></div></div><form><input id="native-edit" value="dirty"></form>' + markup, {headers: {'Content-Type': 'text/html'}});
    }
    if (url.pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><button id="native">Native action</button><input id="native-input" value="unsaved"><main></main><script type="module" src="/assets/drawer-fixture.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {await browser?.close(); server?.stop(true);});
async function fixture(t: TestContext, extra: Partial<State> = {}) {
  const page = await browser.newPage(); const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  t.after(async () => {await page.close(); assert.deepEqual(errors, []);});
  await page.goto(server.url.href);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async extra => {window.drawerFixture = window.createDrawerFixture(extra); await window.drawerFixture.api.ready;}, extra);
  return page;
}
async function refresh(page: Page) {await page.evaluate(() => window.drawerFixture.api.refresh());}
async function click(page: Page, text: string) {
  await page.evaluate(async text => {(await window.drawerFixture.showButton(text)).click();}, text);
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
}
const writes = (page: Page) => page.evaluate(() => window.drawerFixture.calls.filter(call => call.method !== 'GET'));

async function nativeFixture(t: TestContext, options: {anonymous?: boolean; paused?: boolean} = {}) {
  const page = await browser.newPage(), calls: string[] = [], modules: string[] = [], errors: string[] = [];
  let release: (() => void) | undefined;
  const wait = new Promise<void>(resolve => {release = resolve;});
  page.on('pageerror', error => errors.push(error.message));
  page.on('request', request => {if (new URL(request.url()).pathname.endsWith('.js')) modules.push(new URL(request.url()).pathname);});
  t.after(async () => {release?.(); await page.close(); assert.deepEqual(errors, []);});
  await page.route('**/-/soda/api/**', async route => {
    const pathname = new URL(route.request().url()).pathname;
    calls.push(pathname); assert.equal(route.request().method(), 'GET');
    if (options.paused) await wait;
    const body = pathname.endsWith('/session') ? {user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: server.url.origin}
      : pathname.endsWith('/spaces') ? {items: [], complete: true} : pathname.endsWith('/forgejo/me') ? {id: '1'} : {items: [], repository: {id: '7', owner: 'alice', name: 'repo'}, can_create: true};
    await route.fulfill({json: body});
  });
  await page.goto(new URL('/native' + (options.anonymous ? '?anonymous=1' : ''), server.url).href);
  await page.locator('#sodaspaces-root[data-mounted=true]').waitFor({state: 'attached'});
  return {page, calls, modules, release: () => release?.()};
}

test('native mount is inert and lazy, unique, and preserves native forms/actions', async t => {
  const f = await nativeFixture(t); assert.deepEqual(f.calls, []);
  assert(!f.modules.some(module => module.endsWith('/lit.js') || module.endsWith('/sodaspaces-drawer.js')));
  assert.equal(await f.page.locator('.repo-buttons #sodaspaces-button').count(), 1);
  await f.page.locator('#sodaspaces-button').click(); await f.page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
  assert.equal(f.calls.length, 2); assert.equal(await f.page.locator('#native-edit').inputValue(), 'dirty');
  assert.equal(await f.page.locator('#native').count(), 1); assert.equal(await f.page.locator('#sodaspaces-status').innerText(), '');
  assert.equal(await f.page.locator('#sodaspaces-content .soda-page').count(), 0);
  assert.equal(f.modules.filter(module => module.endsWith('/lit.js')).length, 1);
});
test('anonymous native page offers contextual OAuth without substituting Soda identity', async t => {
  const f = await nativeFixture(t, {anonymous: true}); await f.page.locator('#sodaspaces-button').click(); await f.page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
  assert.equal(f.calls.length, 0); assert(await f.page.getByRole('link', {name: 'Connect to Soda'}).isVisible());
  assert.equal(await f.page.getByRole('link', {name: 'Connect to Soda'}).getAttribute('href'), '/-/soda/login?repository_id=7');
  assert(await f.page.getByRole('button', {name: 'New terminal', exact: true}).isDisabled());
  await f.page.getByLabel('Workspace options', {exact: true}).click();
  assert(await f.page.getByRole('button', {name: 'Repository environment / access', exact: true}).isDisabled());
  assert.equal(f.calls.length, 0);
});
test('native late fetch after document departure cannot repaint or dispatch another request', async t => {
  const f = await nativeFixture(t, {paused: true});
  const pending = f.page.waitForRequest('**/-/soda/api/session'); await f.page.locator('#sodaspaces-button').click(); await pending;
  await f.page.evaluate(() => window.dispatchEvent(new Event('pagehide'))); f.release();
  await f.page.locator('#sodaspaces-close').click(); await f.page.locator('#sodaspaces-button').click();
  assert.equal(f.calls.length, 1); assert.equal(await f.page.locator('#sodaspaces-create').count(), 0);
});
test('hiding during lazy module loading cannot mount or dispatch a late refresh', async t => {
  const f = await nativeFixture(t);
  let release: (() => void) | undefined;
  const paused = new Promise<void>(resolve => {release = resolve;});
  await f.page.route('**/assets/sodaspaces-drawer.js?*', async route => {await paused; await route.continue();});
  const requested = f.page.waitForRequest('**/assets/sodaspaces-drawer.js?*');
  await f.page.locator('#sodaspaces-button').click(); await requested;
  await f.page.locator('#sodaspaces-close').click(); release?.();
  await f.page.waitForFunction(() => !!customElements.get('soda-spaces'));
  assert.equal(await f.page.locator('soda-spaces').count(), 0); assert.deepEqual(f.calls, []);
  await f.page.locator('#sodaspaces-button').click(); await f.page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
  assert.equal(await f.page.locator('soda-spaces').count(), 1); assert.equal(f.calls.length, 2);
});

test('interior clicks, initial pageshow and focus preserve the actual Lit drawer', async t => {
  const f = await nativeFixture(t); await f.page.locator('#sodaspaces-button').click(); await f.page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
  const count = f.calls.length;
  await f.page.evaluate(() => {document.getElementById('sodaspaces-content')?.click(); window.dispatchEvent(new Event('focus')); window.dispatchEvent(new PageTransitionEvent('pageshow'));});
  assert(await f.page.locator('#sodaspaces-drawer').isVisible()); assert.equal(f.calls.length, count); assert.equal(await f.page.locator('#sodaspaces-status').innerText(), '');
});

test('project control mount is inert; refresh only reads and never owns a terminal', async t => {
  const page = await fixture(t);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), 0);
  await refresh(page); await refresh(page);
  assert.deepEqual(await writes(page), []);
  assert.deepEqual(await page.evaluate(() => ({count: window.drawerFixture.terminals.length,
    native: document.getElementById('native')?.textContent, input: document.querySelector<HTMLInputElement>('#native-input')?.value})),
  {count: 0, native: 'Native action', input: 'unsaved'});
});
test('project view tabs and app switches dispatch no reads or writes', async t => {
  const page = await fixture(t); await refresh(page);
  const count = await page.evaluate(() => window.drawerFixture.calls.length);
  assert.equal(await page.locator('[role=tab][aria-selected=true]').innerText(), 'Environment');
  await click(page, 'Access'); assert.equal(await page.getByRole('tabpanel', {name: 'Access'}).getAttribute('hidden'), null);
  await click(page, 'Environment');
  await page.evaluate(() => {window.dispatchEvent(new Event('blur')); document.dispatchEvent(new Event('visibilitychange'));});
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), count);
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  await page.getByRole('tab', {name: 'Environment', exact: true}).press('ArrowRight');
  assert.equal(await page.locator('[role=tab][aria-selected=true]').innerText(), 'Access');
  assert.equal(await page.evaluate(() => document.activeElement?.textContent?.trim()), 'Access');
});
test('hidden access view cannot remove a key through a synthetic click', async t => {
  const page = await fixture(t); await refresh(page);
  await page.evaluate(() => window.drawerFixture.button('Remove saved key').click());
  assert.deepEqual(await writes(page), []);
});
test('compact-surface inert project controls cannot dispatch through a synthetic click', async t => {
  const page = await fixture(t); await refresh(page);
  await click(page, 'Access'); await page.locator('textarea').fill('ssh-ed25519 YWJj');
  const changed = await page.evaluate(async () => {
    const f = window.drawerFixture, button = await window.drawerFixture.showButton('Save public key');
    button.closest('soda-project-controls')?.setAttribute('inert', '');
    const before = f.calls.length; button.click(); return f.calls.length - before;
  });
  assert.equal(changed, 0); assert.deepEqual(await writes(page), []);
});
test('create never implicitly joins, saves keys or starts; rapid clicks dispatch once', async t => {
  const page = await fixture(t, {absent: true}); await refresh(page);
  await page.evaluate(async () => {const b = await window.drawerFixture.showButton('Create environment'); b.click(); b.click();});
  await page.locator('[data-project-controls][aria-busy=false]').waitFor();
  const sent = await writes(page); assert.equal(sent.length, 1);
  assert.equal(sent[0]?.url, '/-/soda/api/environments'); assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {repository_id: '7', profile_id: 'rocky-headless'});
  assert.equal(sent[0]?.headers['x-soda-expected-user-id'], '1'); assert.equal(sent[0]?.headers['x-csrf-token'], 'synthetic-csrf');
});
test('legacy OS observation is explicit, read-only and never becomes a creation profile', async t => {
  const page = await fixture(t); await refresh(page);
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.filter(c => c.url.endsWith('/os')).length), 0);
  await click(page, 'Inspect current OS');
  await page.getByText('Rocky Linux 9.7 · rocky 9.7', {exact: true}).waitFor();
  await page.getByText(/Legacy \/ unknown creation profile/).waitFor();
  assert.deepEqual(await writes(page), []);
  await page.evaluate(() => {window.drawerFixture.state.running = false;}); await refresh(page);
  assert.equal(await page.getByText('Rocky Linux 9.7 · rocky 9.7', {exact: true}).count(), 0);
  await click(page, 'Inspect current OS');
  await page.getByText('OS release not read: environment is stopped.', {exact: true}).waitFor();
  assert.deepEqual(await writes(page), []);
});

test('OS text stays text, malformed receipts clear observations, and denial invalidates context', async t => {
  const page = await fixture(t); await refresh(page);
  await page.evaluate(() => window.drawerFixture.setReply(async call => call.url.endsWith('/os') ? Response.json({environment: {id: 'p0123456789abcdef01234567', running: true}, os_release: {id: 'rocky', version: '9.7', name: '<img src=x onerror=alert(1)>'}, os_release_unavailable: false}) : null));
  await click(page, 'Inspect current OS');
  await page.getByText('<img src=x onerror=alert(1)> · rocky 9.7', {exact: true}).waitFor();
  assert.equal(await page.locator('[aria-label="Observed project userspace"] img').count(), 0);
  await page.evaluate(() => window.drawerFixture.setReply(async call => call.url.endsWith('/os') ? Response.json({environment: {id: 'wrong-target', running: true}, os_release: null, os_release_unavailable: true}) : null));
  await click(page, 'Inspect current OS');
  await page.getByText('OS observation unavailable. Nothing was started or repaired.', {exact: true}).waitFor();
  assert.equal(await page.getByText('<img src=x onerror=alert(1)> · rocky 9.7', {exact: true}).count(), 0);
  await page.evaluate(() => window.drawerFixture.setReply(async call => call.url.endsWith('/os') ? Response.json({error: {code: 'forbidden'}}, {status: 403}) : null));
  await click(page, 'Inspect current OS');
  await page.getByText(/Page context changed/).waitFor();
  assert.deepEqual(await writes(page), []);
});

test('Stop requires explicit shared-impact confirmation and Start is separate', async t => {
  const page = await fixture(t); await refresh(page); await click(page, 'Stop'); assert.deepEqual(await writes(page), []);
  await page.locator('input[type=checkbox]').first().check(); await click(page, 'Stop');
  const sent = await writes(page); assert.equal(sent.length, 1); assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {action: 'stop', confirm_stop: true});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
});
test('stopped environment has explicit Start only, and no terminal', async t => {
  const page = await fixture(t, {running: false}); await refresh(page); await click(page, 'Start');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {action: 'start'});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
});
test('saved-key removal truthfully does not dispatch a native apply', async t => {
  const page = await fixture(t); await refresh(page); await click(page, 'Remove saved key');
  const sent = await writes(page); assert.equal(sent.length, 1); assert.equal(sent[0]?.method, 'DELETE'); assert(!sent[0]?.url.endsWith('/access-keys'));
  assert.match(await page.locator('[data-control=result]').innerText(), /Existing project SSH access is unchanged/);
});
test('review then explicit Apply confirms last-key removal', async t => {
  const page = await fixture(t, {saved: []}); await refresh(page); await click(page, 'Review this project’s SSH keys');
  await click(page, 'Apply reviewed saved keys to this project'); assert.deepEqual(await writes(page), []);
  await page.locator('input[type=checkbox]').nth(1).check(); await click(page, 'Apply reviewed saved keys to this project');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {revision: 'a'.repeat(64), saved_fingerprints: [], confirm_empty: true});
});
test('own Forgejo key selection is explicit and does not install or silently save a profile key', async t => {
  const page = await fixture(t, {saved: []}); await refresh(page);
  await page.evaluate(() => window.drawerFixture.setReply(async call => {
    if (call.url.endsWith('/api/me/forgejo-keys?page=1')) return Response.json({items: [{id: '7', title: '<script>not markup</script>', fingerprint: 'SHA256:' + 'A'.repeat(43), public_key: 'ssh-ed25519 YWJj\n'}], page: 1, more: false});
    if (call.method === 'POST' && call.url.endsWith('/api/me/development-keys')) return Response.json({items: [{id: '1', fingerprint: 'SHA256:' + 'A'.repeat(43), public_key: 'ssh-ed25519 YWJj\n'}]});
    return null;
  }));
  await click(page, 'Review my Forgejo public keys');
  assert.deepEqual(await writes(page), []); assert.equal(await page.locator('soda-project-controls script').count(), 0);
  await click(page, 'Select for review');
  assert.equal(await page.locator('textarea').inputValue(), 'ssh-ed25519 YWJj\n'); assert.deepEqual(await writes(page), []);
  await click(page, 'Save public key');
  const sent = await writes(page); assert.equal(sent.length, 1); assert.equal(sent[0]?.url, '/-/soda/api/me/development-keys');
  assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {public_key: 'ssh-ed25519 YWJj'});
});
test('private-key paste is refused before request dispatch', async t => {
  const page = await fixture(t); await refresh(page); await click(page, 'Access');
  await page.locator('textarea').fill('-----BEGIN OPENSSH PRIVATE KEY-----'); await click(page, 'Save public key');
  assert.deepEqual(await writes(page), []); assert.match(await page.locator('[data-control=result]').innerText(), /Never upload a private key/);
});
test('nonadministrator has no lifecycle controls; nonmember joins separately', async t => {
  const page = await fixture(t, {admin: false, member: false}); await refresh(page);
  assert.equal(await page.evaluate(() => window.drawerFixture.button('Start').closest('fieldset')?.hidden), true);
  await click(page, 'Join environment'); const sent = await writes(page); assert.equal(sent.length, 1); assert(sent[0]?.url.endsWith('/join'));
});
test('browser-only Join is available without a public key and never imports saved keys implicitly', async t => {
  for (const saved of [[], ['SHA256:' + 'A'.repeat(43)]]) {
    const page = await fixture(t, {admin: false, member: false, saved}); await refresh(page);
    assert(await page.getByRole('button', {name: 'Join environment', exact: true}).isVisible());
    await click(page, 'Join environment');
    const sent = await writes(page); assert.equal(sent.length, 1); assert.deepEqual(JSON.parse(sent[0]?.body || '{}'), {ssh_keys: 'none'});
  }
});
test('external SSH at Join requires explicit saved-key selection', async t => {
  const page = await fixture(t, {admin: false, member: false}); await refresh(page);
  await page.getByLabel('Also install my saved public keys for external SSH').check();
  await click(page, 'Join environment');
  assert.deepEqual(JSON.parse((await writes(page))[0]?.body || '{}'), {ssh_keys: 'saved'});
});
test('unknown mutation outcome blocks replay, not safe refresh or logout', async t => {
  const page = await fixture(t, {absent: true});
  await page.evaluate(() => window.drawerFixture.setReply(async call => call.method === 'POST' && call.url.endsWith('/api/environments') ? new Response(null, {status: 502}) : null));
  await refresh(page); await click(page, 'Create environment'); await refresh(page);
  assert.equal(await page.locator('[data-control=create]').isDisabled(), true); assert.match(await page.locator('[data-control=result]').innerText(), /Outcome unconfirmed/);
  await click(page, 'Sign out'); assert((await writes(page)).some(call => call.url.endsWith('/api/session/logout')));
});
test('stale project controls cannot refresh/replay on focus', async t => {
  const page = await fixture(t); await refresh(page);
  const count = await page.evaluate(() => window.drawerFixture.calls.length);
  await page.evaluate(async () => {window.dispatchEvent(new Event('pagehide')); window.dispatchEvent(new Event('focus')); await window.drawerFixture.api.refresh();});
  assert.equal(await page.evaluate(() => window.drawerFixture.calls.length), count);
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.equal(await page.locator('[data-control=refresh]').isDisabled(), true); assert.equal(await page.locator('#native').count(), 1);
});
for (const authority of ['user', 'provider'] as const) test(`${authority} mismatch at action time dispatches no mutation`, async t => {
  const page = await fixture(t, {absent: true}); await refresh(page);
  await page.evaluate(authority => {window.drawerFixture.state[authority] = '2';}, authority);
  await click(page, 'Create environment'); assert.deepEqual(await writes(page), []);
});
for (const kind of ['HTML', 'oversized', '401', '403'] as const) test(`${kind} response cannot expose actions`, async t => {
  const page = await fixture(t);
  await page.evaluate(kind => window.drawerFixture.setReply(async () => kind === 'HTML' ? new Response('<html>login</html>', {headers: {'Content-Type': 'text/html'}}) : kind === 'oversized' ? Response.json({padding: 'x'.repeat(65537)}) : new Response(null, {status: Number(kind)})), kind);
  await refresh(page);
  assert.equal(await page.locator('[data-control=create]').getAttribute('hidden'), ''); assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
});
test('repeated project refresh never acquires terminal ownership', async t => {
  const page = await fixture(t); await refresh(page); await refresh(page);
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.doesNotMatch(await page.locator('main').innerText(), /terminal session ended/);
});
test('closing while authorization is pending never dispatches the mutation', async t => {
  const page = await fixture(t, {absent: true}); await refresh(page);
  await page.evaluate(async () => {
    let release: ((value: Response) => void) | undefined;
    const f = window.drawerFixture;
    f.setReply(async call => call.url.endsWith('/api/session') ? new Promise(resolve => {release = resolve;}) : null);
    (await f.showButton('Create environment')).click(); f.api.dispose();
    if (!release) throw Error('authorization was not pending');
    release(Response.json({user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: location.origin}));
  });
  assert.deepEqual(await writes(page), []);
});
test('unknown key-save response cannot claim a confirmed key', async t => {
  const page = await fixture(t); await refresh(page); await click(page, 'Access');
  await page.locator('textarea').fill('ssh-ed25519 YWJj'); await click(page, 'Save public key');
  assert.match(await page.locator('[data-control=result]').innerText(), /Outcome unconfirmed/);
});
test('hidden create and lifecycle actions cannot dispatch through their handlers', async t => {
  const page = await fixture(t, {admin: false}); await refresh(page);
  await click(page, 'Create environment'); await click(page, 'Start'); assert.deepEqual(await writes(page), []);
});
test('copy uses own displayed login/IP without changing native access', async t => {
  const page = await fixture(t); await refresh(page); await click(page, 'Copy SSH connection');
  assert.equal(await page.locator('[data-control=copy]').getAttribute('data-clipboard-target'), '#soda-command-7');
  assert.equal(await page.locator('[data-control=command]').inputValue(), 'ssh alice@10.89.0.2'); assert.deepEqual(await writes(page), []);
});
test('reactive project view/Hide updates preserve draft identity and selection', async t => {
  const page = await fixture(t); await refresh(page); await click(page, 'Access');
  await page.locator('textarea').fill('unsent public key');
  assert.deepEqual(await page.evaluate(async () => {
    const f = window.drawerFixture, input = f.root.querySelector('textarea'), controls = f.root.firstElementChild;
    if (!input || !controls) throw Error('missing fixture nodes'); input.setSelectionRange(2, 5);
    f.button('Environment').click(); await f.api.ready; f.root.hidden = true; f.root.hidden = false;
    f.button('Access').click(); await f.api.ready;
    return {input: input === f.root.querySelector('textarea'), value: input.value, start: input.selectionStart, end: input.selectionEnd,
      controls: controls === f.root.firstElementChild, terminals: f.terminals.length};
  }), {input: true, value: 'unsent public key', start: 2, end: 5, controls: true, terminals: 0});
});
test('hidden completed project reads never attach a terminal on showing details', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {const f = window.drawerFixture; f.root.hidden = true; await f.api.refresh();});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  await page.evaluate(async () => {const f = window.drawerFixture; f.root.hidden = false; await f.api.ready;});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0);
  assert.deepEqual(await writes(page), []);
});

test('render readiness after disposal cannot call terminal factory; reconnect remains retired', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {const f = window.drawerFixture; const pending = f.api.refresh(); const element = f.root.firstElementChild; f.api.dispose(); if (element) f.root.append(element); await pending; await f.api.ready;});
  assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0); assert.deepEqual(await writes(page), []);
});
for (const state of [{provisioned: false}, {unavailable: true}, {user: '2'}]) test(`unavailable/incomplete/mismatched state refuses terminal: ${JSON.stringify(state)}`, async t => {
  const page = await fixture(t, state); await refresh(page); assert.equal(await page.evaluate(() => window.drawerFixture.terminals.length), 0); assert.deepEqual(await writes(page), []);
});
