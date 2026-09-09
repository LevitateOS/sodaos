import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser} from 'playwright';
import path from 'node:path';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import type {} from './fixtures/workspace-fixture';
import {parseLayout, focusedPane} from '../../appliance/forgejo/public/assets/sodaspaces-layout';

const root = path.resolve(import.meta.dirname, '../..');
let browser: Browser, server: ReturnType<typeof Bun.serve>;
before(async () => {
  const fixture = await buildForgejoModule(path.join(root, 'tests/frontend/fixtures/workspace-fixture.ts'), 'public/assets/workspace-fixture.js');
  server = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(req) {
    const url = new URL(req.url);
    if (url.pathname === '/workspace-fixture.js') return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
    const target = 'public' + url.pathname.replace(/^\/appliance\/forgejo\/public/, '');
    const source = Object.entries(payload).find(([dest]) => dest === target)?.[1];
    if (source) {const file = source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : source.startsWith('@build/terminal-assets/') ? path.join(root, '.artifacts/browser-terminal/vendor', path.basename(source)) : path.join(root, source); return new Response(Bun.file(file), {headers: {'Content-Type': /\.m?js$/.test(source) ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : 'application/octet-stream'}});}
    if (url.pathname.startsWith('/assets/soda-terminal/')) return new Response(Bun.file(path.join(root, '.artifacts/browser-terminal/vendor', path.basename(url.pathname))), {headers: {'Content-Type': url.pathname.endsWith('.css') ? 'text/css' : 'text/javascript'}});
    if (url.pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><style>main{height:850px}body{margin:0}</style><input id="native-draft" value="unsaved"><main></main><script type="module" src="/workspace-fixture.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {await browser?.close(); server?.stop(true);});
async function fixture(t: TestContext, mode: 'native' | 'page' = 'page') {
  const page = await browser.newPage({viewport: {width: 1440, height: 1000}}), errors: string[] = [];
  page.setDefaultTimeout(5000);
  page.on('pageerror', e => errors.push(e.message)); t.after(async () => {await page.close(); assert.deepEqual(errors, []);});
  await page.goto(server.url.href); await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async mode => {window.workspaceFixture = window.createWorkspaceFixture(mode); await window.workspaceFixture.api.ready;}, mode);
  return page;
}
for (const mode of ['native', 'page'] as const) test(`${mode}: three exact sessions across two projects retain real xterm owners`, async t => {
  const page = await fixture(t, mode);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 0);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  for (const name of ['Build', 'Edit', 'Other project']) {
    await page.locator('.soda-workspace-existing button').filter({hasText: name}).click();
    try {await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor({timeout: 5000});}
    catch (error) {console.error(await page.locator('main').innerText(), await page.evaluate(() => window.workspaceFixture.calls.map(c => c.path))); throw error;}
  }
  assert.equal(await page.locator('.xterm').count(), 3);
  await page.getByRole('tab', {name: 'Build', exact: true}).click();
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.locator('.xterm').count(), 3);
  assert.deepEqual(await page.evaluate(() => ({sockets: window.workspaceFixture.sockets.length, closed: window.workspaceFixture.sockets.map(s => s.closed), actions: window.workspaceFixture.sockets.map(s => s.sent[0]?.action), writes: window.workspaceFixture.calls.filter(c => c.method !== 'GET').length})), {sockets: 3, closed: [0, 0, 0], actions: ['attach', 'attach', 'attach'], writes: 0});
  assert.equal(await page.locator('#native-draft').inputValue(), 'unsaved');
  await page.evaluate(() => window.workspaceFixture.api.retain());
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'hide').length), 3);
  await page.evaluate(() => window.workspaceFixture.api.returnToWork());
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length), 1);
});
test('explicit New creates once and uses a correlated locator instead of singleton storage', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.is-connected').waitFor();
  const result = await page.evaluate(() => ({frames: window.workspaceFixture.sockets.flatMap(s => s.sent.filter(f => f.action === 'create')), saved: sessionStorage.getItem('soda-spaces:v2:1'), legacy: sessionStorage.getItem('soda-terminal:1:' + 'p' + '1'.repeat(24))}));
  assert.equal(result.frames.length, 1); assert.match(String(result.frames[0]?.request_id), /^[a-f0-9]{32}$/); assert.equal(result.legacy, null); assert(result.saved); assert(!result.saved.includes('synthetic-only'));
});
test('End requires confirmation and unknown cleanup preserves the exact locator', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Build'}).click(); await page.locator('.is-connected').waitFor();
  await page.getByRole('button', {name: 'End terminal', exact: true}).click();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'end').length), 0);
  await page.evaluate(() => window.workspaceFixture.setUnknownEnd());
  await page.locator('.soda-terminal input[type=checkbox]').check(); await page.getByRole('button', {name: 'End terminal', exact: true}).click();
  await page.getByText('End is pending or unconfirmed.', {exact: false}).waitFor();
  assert.match(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''), /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/);
});
test('changed actor invalidates siblings rather than authorizing through the current cookie', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Build'}).click(); await page.locator('.is-connected').waitFor();
  await page.evaluate(() => {window.workspaceFixture.setUser('2');}); await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 1);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).isDisabled(), true);
});
test('Hide preserves the renderer/socket and Show does not Return', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Build'}).click(); await page.locator('.is-connected').waitFor();
  await page.getByRole('button', {name: 'Hide session', exact: true}).click();
  await page.getByRole('button', {name: 'Show Build', exact: true}).waitFor();
  assert.equal(await page.locator('.xterm').count(), 1);
  await page.getByRole('button', {name: 'Show Build', exact: true}).click();
  assert.equal(await page.locator('.xterm').count(), 1);
  assert.deepEqual(await page.evaluate(() => ({sockets: window.workspaceFixture.sockets.length, closed: window.workspaceFixture.sockets[0]?.closed, returns: window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length})), {sockets: 1, closed: 0, returns: 0});
});
test('project drafts and independent terminal owners survive detail switching', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Build'}).click(); await page.locator('.is-connected').waitFor();
  await page.getByRole('button', {name: 'Alpha', exact: true}).click();
  await page.locator('soda-project-controls [aria-busy=false]').waitFor();
  await page.getByRole('tab', {name: 'Access', exact: true}).click();
  await page.locator('soda-project-controls textarea').fill('unsent public key draft');
  await page.getByRole('button', {name: 'Beta', exact: true}).click();
  await page.locator('soda-project-controls:visible [aria-busy=false]').waitFor();
  await page.getByRole('button', {name: 'Alpha', exact: true}).click();
  assert.equal(await page.locator('soda-project-controls:visible textarea').inputValue(), 'unsent public key draft');
  assert.equal(await page.locator('.xterm').count(), 1); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
test('rename remains bound to the original ID when project selection changes', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Build'}).click(); await page.locator('.is-connected').waitFor();
  await page.getByLabel('Session name').fill('Original project build');
  await page.getByRole('button', {name: 'Beta', exact: true}).click();
  await page.waitForFunction(() => window.workspaceFixture.calls.some(c => c.body?.action === 'rename'));
  const calls = await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'rename'));
  assert.equal(calls.length, 1); assert(calls[0]?.path.endsWith('/terminal-sessions/' + 'a'.repeat(32)));
  assert.equal(calls[0]?.body?.name, 'Original project build');
});
test('legacy pending and unknown legacy IDs never select or create a session', async t => {
  const page = await fixture(t);
  await page.evaluate(() => {sessionStorage.setItem('soda-terminal:1:p' + '1'.repeat(24), 'pending'); sessionStorage.setItem('soda-terminal:1:p' + '2'.repeat(24), 'd'.repeat(32));});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await page.locator('soda-terminal').count(), 0);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-terminal:1:p' + '1'.repeat(24))), 'pending');
});
test('v1 hidden and unavailable records migrate without mounting or attaching', async t => {
  const page = await fixture(t);
  const original = await page.evaluate(() => {
    const text = JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32), hidden: true}, {environmentId: 'p' + '2'.repeat(24), requestId: 'd'.repeat(32), hidden: true}]});
    sessionStorage.setItem('soda-spaces:v1:1', text); window.workspaceFixture.spaces.pop(); window.workspaceFixture.setComplete(false); return text;
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  const raw = await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1'));
  assert(raw); const saved = parseLayout(raw);
  assert.equal(saved.entries.length, 2); assert.equal(focusedPane(saved).selected, null);
  assert.equal(await page.locator('soda-terminal').count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v1:1')), original);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), raw);
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Build'}).click();
  await page.locator('.is-connected').waitFor();
  const selected = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert.equal(focusedPane(selected).selected, saved.entries[0]?.key);
  assert.equal(selected.entries.length, 2);
});
test('stored pending promotion keeps its key and native navigation restores exact selection', async t => {
  const page = await fixture(t);
  await page.evaluate(() => sessionStorage.setItem('soda-spaces:v1:1', JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), requestId: 'a'.repeat(32)}, {environmentId: 'p' + '1'.repeat(24), id: 'b'.repeat(32)}]})));
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await page.locator('.is-connected').waitFor();
  const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert.equal(saved.entries[0]?.locator.kind, 'existing');
  assert.equal(await page.locator('.soda-workspace-terminal').getAttribute('id'), 'soda-owner-' + saved.entries[0]?.key);
  await page.locator('.soda-workspace-existing button').filter({hasText: 'Edit'}).click(); await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
  const chosen = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert.equal(focusedPane(chosen).selected, saved.entries[1]?.key);
  await page.reload(); await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async () => {window.workspaceFixture = window.createWorkspaceFixture('native'); await window.workspaceFixture.api.ready; await window.workspaceFixture.api.refresh();});
  await page.locator('.is-connected').waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal').getAttribute('id'), 'soda-owner-' + saved.entries[1]?.key);
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(s => s.sent[0]?.id)), ['b'.repeat(32)]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length), 0);
});
for (const raw of ['{', JSON.stringify({version: 3}), ' '.repeat(32769)]) test('invalid v2 preserves bytes and cannot fall back to v1: ' + raw.length, async t => {
  const page = await fixture(t);
  await page.evaluate(raw => {sessionStorage.setItem('soda-spaces:v2:1', raw); sessionStorage.setItem('soda-spaces:v1:1', JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32)}]}));}, raw);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.locator('soda-terminal').count(), 0);
  await page.getByRole('button', {name: 'New terminal', exact: true}).click(); await page.locator('.is-connected').waitFor();
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), raw);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
});
test('failed v2 write preserves v1 and leaves live terminals usable', async t => {
  const page = await fixture(t);
  const raw = await page.evaluate(() => {
    const raw = JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32)}]}); sessionStorage.setItem('soda-spaces:v1:1', raw);
    const write = Storage.prototype.setItem; Storage.prototype.setItem = function(key, value) {if (key.startsWith('soda-spaces:v2:')) throw new DOMException('Quota exceeded', 'QuotaExceededError'); write.call(this, key, value);}; return raw;
  });
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await page.locator('.is-connected').waitFor();
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v1:1')), raw);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), null);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  await page.getByText('Live workspace remains usable', {exact: false}).waitFor();
});

test('departure during authorization cannot dispatch a later collection read', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {const f = window.workspaceFixture; let release: (() => void) | undefined; f.pause(new Promise<void>(resolve => {release = resolve;})); const pending = f.api.refresh(); f.api.dispose(); release?.(); await pending;});
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 1);
});
