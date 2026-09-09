import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import type {} from './fixtures/workspace-fixture';
import {projectView, newManagedTerminal, terminalMenu} from '../installed/sodaspaces-controls.ts';
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
    if (url.pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><style>main{height:calc(100dvh - 40px)}body{margin:0}</style><input id="native-draft" value="unsaved"><main></main><script type="module" src="/workspace-fixture.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {await browser?.close(); server?.stop(true);});
async function fixture(t: TestContext, mode: 'native' | 'page' = 'page') {
  const page = await browser.newPage({viewport: {width: 1440, height: 1000}}), errors: string[] = [];
  page.setDefaultTimeout(5000); page.on('pageerror', e => errors.push(e.message));
  t.after(async () => {await page.close(); assert.deepEqual(errors, []);});
  await page.goto(server.url.href); await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async mode => {window.workspaceFixture = window.createWorkspaceFixture(mode); await window.workspaceFixture.api.ready;}, mode);
  return page;
}
async function openSession(page: Page, name: string) {
  await page.getByRole('button', {name: 'Sessions', exact: true}).click();
  await page.locator('.soda-session-list button').filter({hasText: name}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
}
async function action(page: Page, name: string) {
  await page.locator('.soda-workspace-terminal:not([hidden]) summary[aria-label="Terminal actions"]').click();
  await page.getByRole('button', {name, exact: true}).click();
}
async function create(page: Page, name = 'New build') {
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.getByLabel('Terminal name', {exact: true}).fill(name);
  await page.getByRole('button', {name: 'Create terminal', exact: true}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
}
async function paneAction(page: Page, name: string) {
  await page.getByLabel('Pane actions', {exact: true}).click();
  await page.getByRole('button', {name, exact: true}).click();
}
for (const mode of ['native', 'page'] as const) test(`${mode}: three exact sessions across two projects retain real xterm owners`, async t => {
  const page = await fixture(t, mode);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 0);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  for (const name of ['Build', 'Edit', 'Other project']) await openSession(page, name);
  assert.equal(await page.locator('.xterm').count(), 3);
  await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).click();
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.locator('.xterm').count(), 3);
  assert.deepEqual(await page.evaluate(() => ({sockets: window.workspaceFixture.sockets.length, closed: window.workspaceFixture.sockets.map(s => s.closed), actions: window.workspaceFixture.sockets.map(s => s.sent[0]?.action), writes: window.workspaceFixture.calls.filter(c => c.method !== 'GET').length})), {sockets: 3, closed: [0, 0, 0], actions: ['attach', 'attach', 'attach'], writes: 0});
  assert.equal(await page.locator('#native-draft').inputValue(), 'unsaved');
  await page.evaluate(() => window.workspaceFixture.api.retain());
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'hide').length), 3);
  await page.evaluate(() => window.workspaceFixture.api.returnToWork());
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length), 1);
});
test('explicit New chooser creates once with name and correlated locator, not singleton storage', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await create(page, 'Named build');
  const result = await page.evaluate(() => ({frames: window.workspaceFixture.sockets.flatMap(s => s.sent.filter(f => f.action === 'create')), saved: sessionStorage.getItem('soda-spaces:v2:1'), legacy: sessionStorage.getItem('soda-terminal:1:p' + '1'.repeat(24))}));
  assert.equal(result.frames.length, 1); assert.equal(result.frames[0]?.name, 'Named build'); assert.match(String(result.frames[0]?.request_id), /^[a-f0-9]{32}$/); assert.equal(result.legacy, null); assert(result.saved); assert(!result.saved.includes('synthetic-only'));
});
test('End confirmation defaults Cancel and unknown cleanup preserves exact locator', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await action(page, 'End terminal…');
  assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Cancel');
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'end').length), 0);
  assert.match(await page.getByRole('dialog', {name: 'End terminal confirmation'}).innerText(), /Build.*alice\/Alpha/s);
  await page.keyboard.press('Escape'); assert.equal(await page.getByRole('dialog').count(), 0);
  await page.evaluate(() => window.workspaceFixture.setUnknownEnd()); await action(page, 'End terminal…');
  await page.getByRole('button', {name: 'End terminal', exact: true}).click();
  await page.getByText('End is pending or unconfirmed.', {exact: false}).waitFor();
  assert.match(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''), /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/);
});
test('changed actor invalidates siblings and clears private navigation observations', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await page.evaluate(async () => {window.workspaceFixture.setUser('2'); await window.workspaceFixture.api.refresh();});
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 1);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).isDisabled(), true);
  assert.equal(await page.locator('.soda-session-list button').count(), 0);
});
test('a terminal admission actor mismatch invalidates every sibling without creating a replacement', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build'); await openSession(page, 'Edit');
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.evaluate(() => window.workspaceFixture.setUser('2'));
  await page.getByRole('button', {name: 'Create terminal', exact: true}).click();
  await page.getByRole('link', {name: 'Connect to Soda', exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(socket => socket.closed)), [1, 1]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.flatMap(socket => socket.sent).filter(frame => frame.action === 'create').length), 0);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
});
test('Hide preserves renderer/socket and showing through the shared navigation does not Return', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  const screen = await page.locator('.xterm').elementHandle(); await action(page, 'Hide session');
  assert.equal(await page.locator('.xterm').count(), 1); assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  await openSession(page, 'Build'); assert(await screen?.evaluate(el => el.isConnected));
  assert.deepEqual(await page.evaluate(() => ({sockets: window.workspaceFixture.sockets.length, closed: window.workspaceFixture.sockets[0]?.closed, returns: window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length})), {sockets: 1, closed: 0, returns: 0});
});
test('project drafts and independent terminal owners survive detail switching', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await action(page, 'Environment / access'); await page.locator('soda-project-controls [aria-busy=false]').waitFor();
  await page.getByRole('tab', {name: 'Access', exact: true}).click(); await page.locator('soda-project-controls textarea').fill('unsent public key draft');
  await page.getByRole('button', {name: 'Environment / access: alice/Beta', exact: true}).click(); await page.locator('soda-project-controls:visible [aria-busy=false]').waitFor();
  await page.getByRole('button', {name: 'Environment / access: alice/Alpha', exact: true}).click();
  assert.equal(await page.locator('soda-project-controls:visible textarea').inputValue(), 'unsent public key draft');
  assert.equal(await page.locator('.xterm').count(), 1); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
test('pending rename keeps original ID while another project is deliberately selected', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await action(page, 'Rename terminal'); await page.getByLabel('Session name', {exact: true}).fill('Original project build');
  await page.evaluate(() => window.workspaceFixture.pause(new Promise<void>(resolve => {window.setTimeout(resolve, 600);})));
  await page.getByRole('button', {name: 'Save name', exact: true}).click();
  await page.getByRole('button', {name: 'Environment / access: alice/Beta', exact: true}).click();
  await page.waitForFunction(() => window.workspaceFixture.calls.some(c => c.body?.action === 'rename'));
  const calls = await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'rename'));
  assert.equal(calls.length, 1); assert(calls[0]?.path.endsWith('/terminal-sessions/' + 'a'.repeat(32))); assert.equal(calls[0]?.body?.name, 'Original project build');
});
test('search and This page change navigation only; New defaults to selected original project', async t => {
  const page = await fixture(t, 'native'); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Other project');
  const calls = await page.evaluate(() => window.workspaceFixture.calls.length);
  await page.getByRole('button', {name: 'Sessions', exact: true}).click(); await page.getByLabel('This page only').check();
  assert.equal(await page.locator('.soda-session-list button').count(), 2);
  await page.getByLabel('Find a session').fill('missing'); await page.getByText('No matches.', {exact: true}).waitFor();
  await page.getByRole('button', {name: 'Back to terminal', exact: true}).click();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), calls);
  assert.equal(await page.locator('.xterm').count(), 1);
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  assert.equal(await page.getByRole('dialog', {name: 'New terminal'}).getByLabel('Project', {exact: true}).inputValue(), 'p' + '2'.repeat(24));
  await page.getByLabel('Terminal name', {exact: true}).fill('a\u200bb'); assert(await page.getByRole('button', {name: 'Create terminal', exact: true}).isDisabled());
  await page.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), calls);
});
test('legacy pending and unknown IDs never select/create a session', async t => {
  const page = await fixture(t);
  await page.evaluate(() => {sessionStorage.setItem('soda-terminal:1:p' + '1'.repeat(24), 'pending'); sessionStorage.setItem('soda-terminal:1:p' + '2'.repeat(24), 'd'.repeat(32));});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0); assert.equal(await page.locator('soda-terminal').count(), 0);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-terminal:1:p' + '1'.repeat(24))), 'pending');
});
test('v1 hidden/unavailable records migrate without attachment; explicit navigation reuses their key', async t => {
  const page = await fixture(t);
  const original = await page.evaluate(() => {const text = JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32), hidden: true}, {environmentId: 'p' + '2'.repeat(24), requestId: 'd'.repeat(32), hidden: true}]}); sessionStorage.setItem('soda-spaces:v1:1', text); window.workspaceFixture.spaces.pop(); window.workspaceFixture.setComplete(false); return text;});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  const raw = await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')); assert(raw); const saved = parseLayout(raw);
  assert.equal(saved.entries.length, 2); assert.equal(focusedPane(saved).selected, null); assert.equal(await page.locator('soda-terminal').count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0); assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v1:1')), original);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), raw);
  await openSession(page, 'Build'); const selected = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert.equal(focusedPane(selected).selected, saved.entries[0]?.key); assert.equal(selected.entries.length, 2);
});
test('pending promotion keeps its owner key and native navigation restores exact saved selection', async t => {
  const page = await fixture(t);
  await page.evaluate(() => sessionStorage.setItem('soda-spaces:v1:1', JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), requestId: 'a'.repeat(32)}, {environmentId: 'p' + '1'.repeat(24), id: 'b'.repeat(32)}]})));
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await page.locator('.is-connected').waitFor();
  const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert.equal(saved.entries[0]?.locator.kind, 'existing'); assert.equal(await page.locator('.soda-workspace-terminal').getAttribute('id'), 'soda-owner-' + saved.entries[0]?.key);
  await openSession(page, 'Edit'); const chosen = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || '')); assert.equal(focusedPane(chosen).selected, saved.entries[1]?.key);
  await page.reload(); await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async () => {window.workspaceFixture = window.createWorkspaceFixture('native'); await window.workspaceFixture.api.ready; await window.workspaceFixture.api.refresh();}); await page.locator('.is-connected').waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal').getAttribute('id'), 'soda-owner-' + saved.entries[1]?.key);
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(s => s.sent[0]?.id)), ['b'.repeat(32)]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length), 0);
});
for (const raw of ['{', JSON.stringify({version: 3}), ' '.repeat(32769)]) test('invalid v2 preserves bytes without v1 fallback: ' + raw.length, async t => {
  const page = await fixture(t); await page.evaluate(raw => {sessionStorage.setItem('soda-spaces:v2:1', raw); sessionStorage.setItem('soda-spaces:v1:1', JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32)}]}));}, raw);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); assert.equal(await page.locator('soda-terminal').count(), 0); await create(page);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), raw); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
});
test('failed v2 write preserves v1 and live terminals', async t => {
  const page = await fixture(t);
  const raw = await page.evaluate(() => {const raw = JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32)}]}); sessionStorage.setItem('soda-spaces:v1:1', raw); const write = Storage.prototype.setItem; Storage.prototype.setItem = function(key, value) {if (key.startsWith('soda-spaces:v2:')) throw new DOMException('Quota exceeded', 'QuotaExceededError'); write.call(this, key, value);}; return raw;});
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await page.locator('.is-connected').waitFor();
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v1:1')), raw); assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), null);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1); await page.getByText('Live workspace remains usable', {exact: false}).waitFor();
});
test('real xterm owners survive pane moves, keyboard divider, maximize, compact and consolidation without effects', async t => {
  const page = await fixture(t); await page.setViewportSize({width: 1920, height: 1440}); await page.evaluate(() => window.workspaceFixture.api.refresh());
  for (const name of ['Build', 'Edit', 'Other project']) await openSession(page, name);
  const screens = await page.locator('.xterm').elementHandles(), hosts = await page.locator('.soda-workspace-terminal').elementHandles();
  const before = await page.evaluate(() => window.workspaceFixture.calls.length); await paneAction(page, 'Split right');
  assert.equal(await page.locator('.soda-pane-chrome').count(), 2); assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), before);
  await page.getByLabel('Move terminal to pane', {exact: true}).click(); await page.getByRole('button', {name: 'Pane 2', exact: true}).click();
  await page.waitForFunction(() => document.querySelectorAll('.soda-workspace-terminal:not([hidden])').length === 2);
  await page.waitForFunction(() => window.workspaceFixture.sockets.every(s => {const resize = s.sent.filter(f => f.type === 'resize').at(-1); return Number(resize?.cols) >= 56 && Number(resize?.rows) >= 12;}));
  await page.getByRole('separator', {name: 'Resize panes', exact: true}).press('ArrowLeft');
  const stored = await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1'));
  await paneAction(page, 'Maximize pane'); assert.equal(await page.locator('.soda-pane-chrome').count(), 1);
  await paneAction(page, 'Restore panes'); assert.equal(await page.locator('.soda-pane-chrome').count(), 2);
  await page.setViewportSize({width: 720, height: 900}); await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 1);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), stored);
  await page.getByLabel('Focused pane', {exact: true}).selectOption({label: 'Pane 1'});
  await page.getByRole('group', {name: 'Pane 1', exact: true}).waitFor();
  const compactSelection = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert.deepEqual(compactSelection.tree, parseLayout(stored || '').tree);
  assert.equal(compactSelection.sidebar, parseLayout(stored || '').sidebar);
  await page.setViewportSize({width: 1920, height: 1440}); await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 2);
  await paneAction(page, 'Consolidate panes'); assert.equal(await page.locator('.soda-pane-chrome').count(), 1);
  for (const handle of [...screens, ...hosts]) assert(await handle.evaluate(el => el.isConnected));
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(s => s.closed)), [0, 0, 0]); assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), before);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.flatMap(s => s.sent).filter(f => f.type === 'resize' && (!f.cols || !f.rows)).length), 0);
});
test('sidebar bounds, tab overflow, pointer reorder and edge split preserve owners without IO', async t => {
  const page = await fixture(t); await page.setViewportSize({width: 1920, height: 1440}); await page.evaluate(() => window.workspaceFixture.api.refresh());
  for (const name of ['Build', 'Edit', 'Other project']) await openSession(page, name);
  const hosts = await page.locator('.soda-workspace-terminal').elementHandles(), before = await page.evaluate(() => window.workspaceFixture.calls.length);
  const sidebar = page.getByRole('separator', {name: 'Resize project sidebar', exact: true});
  await sidebar.press('End'); assert.equal(await sidebar.getAttribute('aria-valuenow'), '360');
  const box = await sidebar.boundingBox(); assert(box); await page.mouse.move(box.x + 3, box.y + 80); await page.mouse.down(); await page.mouse.move(230, box.y + 80); await page.mouse.up();
  const layout = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || ''));
  assert(layout.sidebar !== null && layout.sidebar >= 220 && layout.sidebar < 360);
  await page.getByLabel('Workspace options', {exact: true}).click(); await page.getByRole('button', {name: 'Toggle sidebar', exact: true}).click();
  assert.equal(parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1') || '')).sidebar, null);
  await page.keyboard.press('Escape'); await page.getByLabel('Open tabs', {exact: true}).click(); await page.getByLabel('Find an open tab', {exact: true}).fill('Build');
  assert.equal(await page.locator('.soda-tab-overflow button:visible').count(), 1); await page.locator('.soda-tab-overflow button:visible').click();
  assert.equal(await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).getAttribute('aria-selected'), 'true');
  await page.getByRole('tab', {name: 'Other project · alice/Beta', exact: true}).dragTo(page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}));
  assert.match((await page.getByRole('tab').allTextContents())[0] || '', /Other project/);
  const edit = page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}), origin = await edit.boundingBox(); assert(origin);
  await page.mouse.move(origin.x + origin.width / 2, origin.y + origin.height / 2); await page.mouse.down(); await page.mouse.move(origin.x + 25, origin.y + 65, {steps: 8});
  const edge = await page.locator('.soda-drop-edge.right:visible').boundingBox(); assert(edge);
  await page.mouse.move(edge.x + edge.width / 2, edge.y + edge.height / 2, {steps: 8}); await page.mouse.up();
  await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 2);
  for (const host of hosts) assert(await host.evaluate(node => node.isConnected));
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), before);
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(socket => socket.closed)), [0, 0, 0]);
});
test('installed control paths use the actual emitted project, chooser and original-target End UI', async t => {
  const page = await fixture(t, 'native'); await page.evaluate(() => window.workspaceFixture.api.refresh());
  const controls = await projectView(page, '7', 'Access');
  assert.equal(await controls.getAttribute('data-environment-id'), 'p' + '1'.repeat(24));
  assert.equal(await controls.locator('[data-control=copy]').getAttribute('data-clipboard-target'), '#soda-command-7');
  await assert.rejects(() => newManagedTerminal(page, 'alice/Alpha', 'Wrong target', 'p' + '2'.repeat(24)), /observed original project/);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.getByRole('dialog', {name: 'New terminal', exact: true}).getByRole('button', {name: 'Cancel', exact: true}).click();
  await newManagedTerminal(page, 'alice/Alpha', 'Installed path fixture', 'p' + '1'.repeat(24));
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
  await terminalMenu(page, 'End terminal…');
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'end').length), 0);
  await page.getByRole('dialog', {name: 'End terminal confirmation', exact: true}).getByRole('button', {name: 'End terminal', exact: true}).click();
  await page.waitForFunction(() => !document.querySelector('soda-terminal'));
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'end').length), 1);
});
test('departure during authorization cannot dispatch a late collection read', async t => {
  const page = await fixture(t); await page.evaluate(async () => {const f = window.workspaceFixture; let release: (() => void) | undefined; f.pause(new Promise<void>(resolve => {release = resolve;})); const pending = f.api.refresh(); f.api.dispose(); release?.(); await pending;});
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 1);
});
