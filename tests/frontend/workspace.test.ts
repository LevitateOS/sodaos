import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {chromium, type Browser, type Page} from 'playwright';
import path from 'node:path';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import type {} from './fixtures/workspace-fixture';
import {installMeasurementProbe} from './fixtures/workspace-measurement-probe';
import {projectView, newManagedTerminal, terminalMenu} from '../installed/sodaspaces-controls.ts';
import {captureSpacesComponent} from '../../scripts/screenshot';
import {parseLayout, focusedPane} from '../../frontend/spaces/sodaspaces-layout';

const root = path.resolve(import.meta.dirname, '../..');
let browser: Browser, server: ReturnType<typeof Bun.serve>;
before(async () => {
  const fixture = await buildForgejoModule(path.join(root, 'tests/frontend/fixtures/workspace-fixture.ts'), 'public/assets/workspace-fixture.js');
  server = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(req) {
    const url = new URL(req.url);
    if (url.pathname === '/assets/workspace-fixture.js') return new Response(fixture, {headers: {'Content-Type': 'text/javascript'}});
    const target = 'public' + url.pathname;
    const source = Object.entries(payload).find(([dest]) => dest === target)?.[1];
    if (source) {const file = source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : source.startsWith('@build/terminal-assets/') ? path.join(root, '.artifacts/browser-terminal/vendor', path.basename(source)) : path.join(root, source); return new Response(Bun.file(file), {headers: {'Content-Type': /\.m?js$/.test(source) ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : source.endsWith('.svg') ? 'image/svg+xml' : 'application/octet-stream'}});}
    if (url.pathname !== '/') return new Response(null, {status: 404});
    return new Response('<!doctype html><meta name="soda-component-fixture" content="spaces"><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><style>main{height:calc(100dvh - 40px)}body{margin:0}</style><input id="native-draft" value="unsaved"><main></main><script type="module" src="/assets/workspace-fixture.js"></script>', {headers: {'Content-Type': 'text/html'}});
  }});
  browser = await chromium.launch({headless: true, chromiumSandbox: true});
});
after(async () => {await browser?.close(); server?.stop(true);});
async function fixture(t: TestContext, mode: 'native' | 'page' = 'page', beforeMount?: () => void, firstUse = false) {
  const page = await browser.newPage({viewport: {width: 1440, height: 1000}}), errors: string[] = [];
  page.setDefaultTimeout(5000); page.on('pageerror', e => errors.push(e.message));
  t.after(async () => {await page.close(); assert.deepEqual(errors, []);});
  await page.goto(server.url.href); await page.waitForFunction(() => !!window.createWorkspaceFixture);
  if (beforeMount) await page.evaluate(beforeMount);
  await page.evaluate(async ({mode, firstUse}) => {window.workspaceFixture = window.createWorkspaceFixture(mode, firstUse); await window.workspaceFixture.api.ready;}, {mode, firstUse});
  return page;
}
async function openSession(page: Page, name: string) {
  await page.getByRole('button', {name: /^(Sessions|Projects)$/}).click();
  await page.locator('.soda-session-list button').filter({hasText: name}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
}
async function action(page: Page, name: string) {
  await page.locator('.soda-workspace-terminal:not([hidden]) summary[aria-label="Terminal actions"]').click();
  await page.locator('.soda-workspace-terminal:not([hidden])').getByRole('button', {name, exact: true}).click();
}
async function create(page: Page, name = 'New build') {
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
  await action(page, 'Rename terminal');
  await page.getByLabel('Session name', {exact: true}).fill(name);
  await page.getByRole('button', {name: 'Save name', exact: true}).click();
}
async function paneAction(page: Page, name: string) {
  await page.getByLabel('Pane actions', {exact: true}).click();
  await page.getByRole('button', {name, exact: true}).click();
}
for (const failure of ['unavailable', 'unsafe-path', 'wrong-repository', 'storage']) test(`Open in drawer preserves the terminal when ${failure} prevents a safe handoff`, async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  const screen = await page.locator('.xterm').elementHandle(), original = page.url();
  await page.evaluate(failure => {
    if (failure === 'unavailable') window.workspaceFixture.setStatus(503);
    else if (failure === 'storage') {
      const save = Storage.prototype.setItem;
      Storage.prototype.setItem = function(key, value) {if (key.startsWith('soda-spaces:v3:')) throw Error('Synthetic storage failure'); save.call(this, key, value);};
    } else {
      const fetch = window.fetch;
      Object.defineProperty(window, 'fetch', {configurable: true, value: async (input: RequestInfo | URL, init?: RequestInit) => String(input).includes('/api/environments?')
        ? Response.json({repository: {id: failure === 'wrong-repository' ? '8' : '7', owner: 'alice', name: failure === 'unsafe-path' ? '../elsewhere' : 'Alpha'}})
        : fetch(input, init)});
    }
  }, failure);
  await page.getByLabel('Workspace options', {exact: true}).click();
  await page.getByRole('button', {name: 'Open in drawer', exact: true}).click();
  await page.getByText(failure === 'storage' ? 'Save the workspace before opening it in the drawer; restoration is unavailable.' : 'Could not open the repository drawer. Your terminal remains here; refresh and try again.', {exact: true}).waitFor();
  assert.equal(page.url(), original); assert(await screen?.evaluate(node => node.isConnected));
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(call => call.method !== 'GET').length), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
});
for (const mode of ['native', 'page'] as const) for (const theme of ['light', 'dark'] as const) for (const width of [1440, 390]) test(`${mode}/${theme}/${width}: resolved tokens, focus, menu and warning preserve native draft and terminal`, async t => {
  const page = await fixture(t, mode);
  await page.setViewportSize({width, height: 900});
  await page.evaluate(theme => {document.documentElement.style.colorScheme = theme;}, theme);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  assert(await page.locator('.soda-workspace-toolbar').evaluate(node => node.scrollWidth <= node.clientWidth), 'workspace controls overflow');
  const screen = await page.locator('.xterm').elementHandle();
  await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).focus();
  await page.keyboard.press('Tab'); await page.keyboard.press('Shift+Tab');
  const selected = await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).evaluate(node => {
    const style = getComputedStyle(node), probe = document.createElement('span');
    node.append(probe); probe.style.color = 'var(--soda-page-focus)'; const focus = getComputedStyle(probe).color;
    probe.style.color = 'var(--soda-button-primary-bg)'; const background = getComputedStyle(probe).color; probe.remove();
    return {background: style.backgroundColor, expected: background, focus: style.outlineColor, expectedFocus: focus, height: node.getBoundingClientRect().height};
  });
  assert.equal(selected.background, selected.expected); assert.equal(selected.focus, selected.expectedFocus);
  assert.equal(selected.height, mode === 'native' || width === 390 ? 44 : 36);
  await action(page, 'End terminal…');
  const warning = await page.getByRole('dialog', {name: 'End terminal confirmation'}).evaluate(node => {
    const style = getComputedStyle(node), probe = document.createElement('span'); probe.style.color = 'var(--soda-page-warning-bg)'; node.append(probe);
    const expected = getComputedStyle(probe).color; probe.remove(); return {actual: style.backgroundColor, expected};
  });
  assert.equal(warning.actual, warning.expected); await page.keyboard.press('Escape');
  assert(await screen?.evaluate(node => node.isConnected));
  assert.equal(await page.locator('#native-draft').inputValue(), 'unsaved');
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
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
  await page.evaluate(() => {const f = window.workspaceFixture; f.api.setVisible(false); f.api.setVisible(true);});
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.path.endsWith('/api/session')).length), 1);
});
test('observed unread coalesces noisy hidden output, survives refresh, and clears only deliberate viewing', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build'); await openSession(page, 'Edit');
  const screen = await page.locator('.xterm').first().elementHandle();
  await page.locator('#native-draft').focus();
  await page.evaluate(() => {
    const sockets = window.workspaceFixture.sockets;
    for (let i = 0; i < 100; i++) sockets[0]?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('log\r\n')})});
    sockets[1]?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('visible\r\n')})});
  });
  const tab = page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true});
  await tab.locator('.soda-unread').waitFor();
  assert.equal(await page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}).locator('.soda-unread').count(), 0);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); assert.equal(await tab.locator('.soda-unread').count(), 1);
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  assert.equal(await page.locator('.soda-attention-filters:visible').count(), 0);
  await openSession(page, 'Build'); await page.waitForFunction(() => !document.querySelector('[aria-selected=true] .soda-unread'));
  assert(await screen?.evaluate(node => node.isConnected));
  await page.evaluate(() => {
    window.workspaceFixture.api.setVisible(false);
    window.workspaceFixture.sockets[0]?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('behind Forge')})});
    window.workspaceFixture.api.setVisible(true);
  });
  await tab.locator('.soda-unread').waitFor();
  await page.evaluate(() => window.workspaceFixture.api.markViewed());
  await page.waitForFunction(() => !document.querySelector('[aria-selected=true] .soda-unread'));
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
test('attention has stable authorized counts/order and exact navigation without creation or lifetime actions', async t => {
  const page = await fixture(t); await page.evaluate(() => {
    const f = window.workspaceFixture, a = f.spaces[0]?.terminals[0], b = f.spaces[0]?.terminals[1], c = f.spaces[1]?.terminals[0];
    if (!a || !b || !c) throw Error('missing metadata');
    a.attached = true;
    b.state = 'ending'; b.ready = false;
    c.state = 'ending'; c.ready = false;
    return f.api.refresh();
  });
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.getByRole('button', {name: 'Attention (3)', exact: true}).click();
  assert.deepEqual(await page.locator('.soda-session-list > button > span').allTextContents(), ['Build', 'Edit', 'Other project']);
  await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByRole('tab', {name: 'Build · alice/Alpha', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Projects', exact: true}).click(); await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByRole('tab', {name: 'Edit · alice/Alpha', exact: true}).waitFor();
  await page.getByText('Native transition is still in progress.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await page.evaluate(() => {const b = window.workspaceFixture.spaces[1]; if (b) {b.authority_unavailable = true; b.environment_administrator = false; b.terminals = []; b.login = ''; } return window.workspaceFixture.api.refresh();});
  await page.getByRole('button', {name: 'Attention (2)', exact: true}).waitFor();
  await page.evaluate(() => {window.workspaceFixture.setUser('2'); return window.workspaceFixture.api.refresh();});
  assert.equal(await page.locator('.soda-session-list button').count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.flatMap(s => s.sent).filter(f => f.action === 'create').length), 0);
});
test('stale metadata and late transport generations cannot claim cleanup or replay unread', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  const peer = await page.evaluateHandle(() => window.workspaceFixture.sockets[0]);
  await page.evaluate(() => window.workspaceFixture.sockets[0]?.onmessage?.({data: JSON.stringify({type: 'closed', reason: 'unavailable'})}));
  await page.getByText('Attachment ended or unavailable.', {exact: false}).waitFor();
  await page.getByRole('button', {name: 'Projects', exact: true}).click();
  await peer.evaluate(socket => socket?.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('late')})}));
  assert.equal(await page.locator('.soda-unread').count(), 0);
  await page.getByRole('button', {name: 'Attention (1)', exact: true}).waitFor();
  assert.match(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''), /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/);
});

test('direct New reserves one default-named locator before creating; Rename is separate', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await create(page, 'Named build');
  const result = await page.evaluate(() => ({frames: window.workspaceFixture.sockets.flatMap(s => s.sent.filter(f => f.action === 'create')), saved: sessionStorage.getItem('soda-spaces:v3:1'), legacy: sessionStorage.getItem('soda-terminal:1:p' + '1'.repeat(24))}));
  assert.equal(result.frames.length, 1); assert.equal(result.frames[0]?.name, 'Terminal 3'); assert.match(String(result.frames[0]?.id), /^[a-f0-9]{32}$/); assert(!('request_id' in (result.frames[0] || {}))); assert.equal(result.legacy, null); assert(result.saved); assert(!result.saved.includes('synthetic-only'));
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
  await page.getByText('End was not confirmed.', {exact: false}).waitFor();
  assert.match(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''), /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/);
});
test('changed actor invalidates siblings and clears private navigation observations', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await page.evaluate(async () => {window.workspaceFixture.setUser('2'); await window.workspaceFixture.api.refresh();});
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 1);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 0);
  assert.equal(await page.locator('.soda-session-list button').count(), 0);
});
test('a terminal admission actor mismatch invalidates every sibling without creating a replacement', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build'); await openSession(page, 'Edit');
  await page.evaluate(() => window.workspaceFixture.setUser('2'));
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.getByRole('button', {name: 'Reload Spaces', exact: true}).waitFor();
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(socket => socket.closed)), [1, 1]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.flatMap(socket => socket.sent).filter(frame => frame.action === 'create').length), 0);
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
});
test('Hide preserves renderer/socket and showing through the shared navigation does not Return', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  const screen = await page.locator('.xterm').elementHandle(); await action(page, 'Hide terminal');
  assert.equal(await page.locator('.xterm').count(), 1); assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  await openSession(page, 'Build'); assert(await screen?.evaluate(el => el.isConnected));
  assert.deepEqual(await page.evaluate(() => ({sockets: window.workspaceFixture.sockets.length, closed: window.workspaceFixture.sockets[0]?.closed, returns: window.workspaceFixture.calls.filter(c => c.body?.action === 'return').length})), {sockets: 1, closed: 0, returns: 0});
});
test('project drafts and independent terminal owners survive detail switching', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await action(page, 'Project settings'); await page.locator('soda-project-controls [aria-busy=false]').waitFor();
  await page.getByRole('tab', {name: 'Access', exact: true}).click(); await page.locator('soda-project-controls textarea').fill('unsent public key draft');
  await page.locator('.soda-project-select').filter({hasText: 'alice/Beta'}).click();
  await page.getByRole('button', {name: 'Project settings', exact: true}).click();
  await page.locator('soda-project-controls:visible [aria-busy=false]').waitFor();
  await page.locator('.soda-project-select').filter({hasText: 'alice/Alpha'}).click();
  await page.getByRole('button', {name: 'Project settings', exact: true}).click();
  assert.equal(await page.locator('soda-project-controls:visible textarea').inputValue(), 'unsent public key draft');
  assert.equal(await page.locator('.xterm').count(), 1); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets[0]?.closed), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
test('pending rename keeps original ID while another project is deliberately selected', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  await action(page, 'Rename terminal'); await page.getByLabel('Session name', {exact: true}).fill('Original project build');
  await page.evaluate(() => window.workspaceFixture.pause(new Promise<void>(resolve => {window.setTimeout(resolve, 600);})));
  await page.getByRole('button', {name: 'Save name', exact: true}).click();
  await page.locator('.soda-project-select').filter({hasText: 'alice/Beta'}).click();
  await page.waitForFunction(() => window.workspaceFixture.calls.some(c => c.body?.action === 'rename'));
  const calls = await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.body?.action === 'rename'));
  assert.equal(calls.length, 1); assert(calls[0]?.path.endsWith('/terminal-sessions/' + 'a'.repeat(32))); assert.equal(calls[0]?.body?.name, 'Original project build');
});
test('search and This page change navigation only; New defaults to selected original project', async t => {
  const page = await fixture(t, 'native'); await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Other project');
  const calls = await page.evaluate(() => window.workspaceFixture.calls.length);
  await page.getByRole('button', {name: 'Sessions', exact: true}).click(); await page.getByLabel('This page only').check();
  assert.equal(await page.locator('.soda-session-list button').count(), 2);
  await page.getByLabel('Find a terminal').fill('missing'); await page.getByText('No matches.', {exact: true}).waitFor();
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
test('obsolete caches are ignored; native inventory still permits deliberate discovery', async t => {
  const page = await fixture(t);
  const original = await page.evaluate(() => {const text = JSON.stringify({version: 1, entries: [{environmentId: 'p' + '1'.repeat(24), id: 'a'.repeat(32)}, {requestId: 'd'.repeat(32)}]}); sessionStorage.setItem('soda-spaces:v1:1', text); sessionStorage.setItem('soda-spaces:v2:1', text); return text;});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  assert.equal(await page.locator('soda-terminal').count(), 0); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await openSession(page, 'Build'); const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  assert.equal(saved.entries.length, 1); assert.equal(saved.entries[0]?.locator.kind, 'existing');
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v1:1')), original); assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v2:1')), original);
});
test('current arrangement restores the exact selected ID and owner key, never a replacement', async t => {
  const page = await fixture(t); await page.evaluate(() => window.workspaceFixture.api.refresh());
  await openSession(page, 'Build'); await openSession(page, 'Edit');
  const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  const selected = focusedPane(saved).selected; assert(selected);
  await page.reload(); await page.waitForFunction(() => !!window.createWorkspaceFixture);
  await page.evaluate(async () => {window.workspaceFixture = window.createWorkspaceFixture('native'); await window.workspaceFixture.api.ready; await window.workspaceFixture.api.refresh();}); await page.locator('.is-connected').waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal').getAttribute('id'), 'soda-owner-' + selected);
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(s => s.sent[0]?.id)), ['b'.repeat(32)]);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
for (const raw of ['{', JSON.stringify({version: 2}), ' '.repeat(32769)]) test('corrupt/obsolete current cache resets without native effects: ' + raw.length, async t => {
  const page = await fixture(t); await page.evaluate(raw => sessionStorage.setItem('soda-spaces:v3:1', raw), raw);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); assert.equal(await page.locator('soda-terminal').count(), 0);
  const empty = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || '')); assert.equal(empty.entries.length, 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
  await create(page); const saved = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  assert.equal(saved.entries.length, 1); assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 1);
});
test('storage write failure does not disable the live workspace', async t => {
  const page = await fixture(t);
  await page.evaluate(() => {const write = Storage.prototype.setItem; Storage.prototype.setItem = function(key, value) {if (key.startsWith('soda-spaces:v3:')) throw new DOMException('Quota exceeded', 'QuotaExceededError'); write.call(this, key, value);};});
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await openSession(page, 'Build');
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1')), null);
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
  await page.evaluate(() => {
    const f = window.workspaceFixture, names = [...document.querySelectorAll('.soda-pane-chrome [aria-selected=true]')].map(tab => tab.textContent?.trim());
    const visible = f.spaces.flatMap(space => space.terminals).filter(terminal => names.includes(terminal.name)).map(terminal => terminal.id);
    for (const socket of f.sockets.filter(socket => visible.includes(String(socket.sent[0]?.id)))) socket.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('visible pane')})});
  });
  assert.equal(await page.locator('.soda-workspace-tabs .soda-unread').count(), 0);
  await page.waitForFunction(() => window.workspaceFixture.sockets.every(s => {const resize = s.sent.filter(f => f.type === 'resize').at(-1); return Number(resize?.cols) >= 56 && Number(resize?.rows) >= 12;}));
  await page.getByRole('separator', {name: 'Resize panes', exact: true}).press('ArrowLeft');
  const stored = await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1'));
  await paneAction(page, 'Maximize pane'); assert.equal(await page.locator('.soda-pane-chrome').count(), 1);
  await paneAction(page, 'Restore panes'); assert.equal(await page.locator('.soda-pane-chrome').count(), 2);
  await page.setViewportSize({width: 720, height: 900}); await page.waitForFunction(() => document.querySelectorAll('.soda-pane-chrome').length === 1);
  assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1')), stored);
  await page.getByLabel('Focused pane', {exact: true}).selectOption({label: 'Pane 1'});
  await page.getByRole('group', {name: 'Pane 1', exact: true}).waitFor();
  const compactSelection = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
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
  const layout = parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || ''));
  assert(layout.sidebar !== null && layout.sidebar >= 220 && layout.sidebar < 360);
  await page.getByLabel('Workspace options', {exact: true}).click(); await page.getByRole('button', {name: 'Toggle sidebar', exact: true}).click();
  assert.equal(parseLayout(await page.evaluate(() => sessionStorage.getItem('soda-spaces:v3:1') || '')).sidebar, null);
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
test('authorized collection reports an observed other writer without needing a failed attachment', async t => {
  const page = await fixture(t);
  await page.evaluate(async () => {const f = window.workspaceFixture, terminal = f.spaces[0]?.terminals[0]; if (!terminal) throw Error('fixture'); terminal.attached = true; await f.api.refresh();});
  await page.getByRole('button', {name: 'Attention (1)', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Next attention', exact: true}).click();
  await page.getByText('An existing writer is attached.', {exact: false}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  await page.evaluate(async () => {const f = window.workspaceFixture, terminal = f.spaces[0]?.terminals[0]; if (!terminal) throw Error('fixture'); terminal.attached = false; await f.api.refresh();});
  assert.equal(await page.locator('.soda-attention-filters:visible').count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(call => call.body).length), 0);
});
for (const retirement of ['invalidate', 'dispose', 'disconnect'] as const) test(`measurement subscriptions retire on ${retirement}, including queued callbacks`, async t => {
  const page = await fixture(t, 'page', installMeasurementProbe);
  await page.waitForFunction(() => window.measurementProbe.minimumNotifications > 0);
  assert.deepEqual(await page.evaluate(() => {
    const p = window.measurementProbe;
    return {
      beforeRender: p.beforeFirstRender,
      observers: p.observers.length,
      targets: p.observers[0]?.targets.map(target => target.matches('.soda-workspace-canvas') ? 'canvas' : target.localName),
      activeSignals: p.signals.filter(signal => !signal.aborted).length,
    };
  }), {beforeRender: 0, observers: 1, targets: ['canvas', 'soda-spaces'], activeSignals: 2});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.setViewportSize({width: 1100, height: 800});
  assert.equal(await page.evaluate(() => window.measurementProbe.observers.length), 1, 'updates resubscribed');
  await page.evaluate(retirement => {
    const f = window.workspaceFixture;
    if (retirement === 'disconnect') f.root.querySelector('soda-spaces')?.remove();
    else f.api[retirement]();
  }, retirement);
  // Let the caller's own invalidation render settle before delivering stale work.
  await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
  const before = await page.evaluate(() => ({html: window.workspaceFixture.root.innerHTML, notifications: window.measurementProbe.minimumNotifications, calls: window.workspaceFixture.calls.length}));
  await page.evaluate(async () => {
    const p = window.measurementProbe;
    for (const observer of p.observers) observer.deliver();
    p.releaseFonts();
    document.fonts.dispatchEvent(new Event('loadingdone'));
    window.visualViewport?.dispatchEvent(new Event('resize'));
    await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
  });
  assert.deepEqual(await page.evaluate(() => ({html: window.workspaceFixture.root.innerHTML, notifications: window.measurementProbe.minimumNotifications, calls: window.workspaceFixture.calls.length})), before);
  assert.deepEqual(await page.evaluate(() => ({disconnects: window.measurementProbe.observers.map(o => o.disconnected), aborted: window.measurementProbe.signals.every(s => s.aborted), sockets: window.workspaceFixture.sockets.length})), {disconnects: [1], aborted: true, sockets: 0});
  // A mount disposed before its first render must never install subscriptions.
  assert.equal(await page.evaluate(async () => {
    const f = window.createWorkspaceFixture();
    f.api.dispose();
    await f.api.ready;
    return window.measurementProbe.observers.length;
  }), 1);
});

async function chooseFirstRepository(page: Page, capture?: string) {
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  if (capture) await captureSpacesComponent(page, 'picker-' + capture);
  await page.getByRole('radio', {name: /alice\/Alpha/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
  await page.waitForFunction(() => !!document.querySelector('.soda-project-journey button.primary:not([disabled])'));
}
for (const theme of ['light', 'dark']) for (const width of [1440, 800, 640, 390]) test(`first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry`, async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.setViewportSize({width, height: 844});
  await page.evaluate(theme => {document.documentElement.style.colorScheme = theme;}, theme);
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'Create your first project'}).waitFor();
  assert.equal(await page.locator('.soda-workspace-navigation:visible').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 0);
  await captureSpacesComponent(page, `welcome-${theme}-${width}`);
  await chooseFirstRepository(page, `${theme}-${width}`);
  await captureSpacesComponent(page, `configure-${theme}-${width}`);
  assert.equal(await page.getByRole('checkbox').count(), 0, 'unavailable Tailnet must stay out of setup');
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 1);
  await captureSpacesComponent(page, `join-${theme}-${width}`);
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.some(c => c.path.endsWith('/development-keys'))), false);
  assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 1);
  await captureSpacesComponent(page, `first-terminal-${theme}-${width}`);
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  assert.equal(await page.getByRole('dialog', {name: 'New terminal', exact: true}).count(), 0);
  await page.waitForFunction(() => document.activeElement?.classList.contains('xterm-helper-textarea'));
  await page.keyboard.type('spaces-ready');
  await page.waitForFunction(() => document.querySelector('.xterm-rows')?.textContent?.includes('spaces-ready'));
  assert(await page.locator('.soda-workspace').evaluate(node => node.scrollWidth <= node.clientWidth), 'page overflow');
  await captureSpacesComponent(page, `working-${theme}-${width}`);
  const before = await page.evaluate(() => {
    const f = window.workspaceFixture;
    return {id: f.spaces[0]?.terminals[0]?.id, writes: f.calls.filter(c => c.method !== 'GET'), creates: f.sockets.flatMap(s => s.sent).filter(c => c.action === 'create').length};
  });
  assert.equal(before.creates, 1); assert.equal(before.writes.length, 3);
  assert.deepEqual(before.writes[1]?.body, {ssh_keys: 'none'});
  await page.evaluate(() => window.workspaceFixture.remount());
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  assert.deepEqual(await page.evaluate(() => {
    const f = window.workspaceFixture;
    return {id: f.spaces[0]?.terminals[0]?.id, writes: f.calls.filter(c => c.method !== 'GET'), creates: f.sockets.flatMap(s => s.sent).filter(c => c.action === 'create').length};
  }), before);
});
for (const outcome of ['uncertain', 'incomplete', 'rejected'] as const) test(`first use: ${outcome} creation is inspected, not replayed`, async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(outcome => {window.workspaceFixture.setCreateOutcome(outcome); return window.workspaceFixture.api.refresh();}, outcome);
  await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByText(outcome === 'rejected' ? /Installed Project OS unavailable\. No reservation/ : /Outcome unconfirmed/).waitFor();
  await page.getByRole('button', {name: 'Refresh status', exact: true}).click();
  if (outcome === 'uncertain') await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
  else if (outcome === 'incomplete') await page.getByRole('heading', {name: 'Project needs inspection'}).waitFor();
  else await page.getByRole('button', {name: 'Create project', exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 1);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
});
test('first use: keyboard selection, Back and Change retain the repository without writes', async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const marker = document.createElement('span'); marker.id = 'soda-settings-link'; marker.hidden = true; marker.dataset.subUrl = '/forgejo.test'; document.body.append(marker);
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: 'Create project', exact: true}).focus(); await page.keyboard.press('Enter');
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  assert(await page.getByRole('button', {name: 'Continue', exact: true}).isDisabled());
  assert.equal(await page.getByRole('link', {name: 'Create a new repository'}).getAttribute('href'), '/forgejo.test/repo/create');
  await page.getByRole('radio', {name: /alice\/Alpha/}).focus(); await page.keyboard.press('Space');
  await page.getByRole('button', {name: 'Continue', exact: true}).focus(); await page.keyboard.press('Enter');
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  await page.getByRole('button', {name: 'Change repository', exact: true}).click();
  assert(await page.getByRole('radio', {name: /alice\/Alpha/}).isChecked());
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.waitForFunction(() => !!document.querySelector('.soda-project-journey button.primary:not([disabled])'));
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.getByRole('heading', {name: 'Create your first project'}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
test('first use: superseded search and retired actor cannot publish stale private choices', async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => {
    const original = window.fetch;
    Object.defineProperty(window, 'fetch', {configurable: true, value: (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input), location.origin);
      if (url.pathname.endsWith('/api/repositories') && url.searchParams.get('q') === 'Alpha') return new Promise<Response>(resolve => window.addEventListener('release-old-search', () => resolve(Response.json({items: [{id: '7', owner: 'alice', name: 'Alpha', can_create: true, project: null}], page: 1, more: false, limited: false})), {once: true}));
      return original(input, init);
    }});
    return window.workspaceFixture.api.refresh();
  });
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Alpha/}).waitFor();
  const search = page.getByRole('searchbox', {name: 'Search repositories'});
  await search.fill('Alpha'); await search.press('Enter');
  await search.fill('Beta'); await search.press('Enter');
  await page.getByRole('radio', {name: /alice\/Beta/}).check();
  await page.evaluate(() => window.dispatchEvent(new Event('release-old-search')));
  assert.equal(await page.getByRole('radio', {name: /alice\/Alpha/}).count(), 0);
  assert(await page.getByRole('radio', {name: /alice\/Beta/}).isChecked());
  await search.fill('Alpha'); await search.press('Enter');
  await page.evaluate(async () => {window.workspaceFixture.setUser('2'); await window.workspaceFixture.api.refresh(); window.dispatchEvent(new Event('release-old-search'));});
  await page.getByRole('button', {name: 'Reload Spaces', exact: true}).waitFor();
  assert.equal(await page.getByRole('radio').count(), 0);
  assert.equal(await page.getByRole('heading', {name: 'Create your first project'}).count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 0);
});
for (const administrator of [true, false]) test(`first use: stopped project Start is explicit and authorized (${administrator})`, async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
  await page.evaluate(async administrator => {
    const f = window.workspaceFixture, space = f.spaces[0]; if (!space?.observed) throw Error('fixture');
    space.observed.running = false; space.environment_administrator = administrator; await f.api.refresh();
  }, administrator);
  await page.getByRole('heading', {name: 'Project not ready'}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 1);
  if (administrator) {
    await page.getByRole('button', {name: 'Start project', exact: true}).click();
    await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
    assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 2);
  } else assert.equal(await page.getByRole('button', {name: 'Start project', exact: true}).count(), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
});
test('first use: failed Join stays scoped; later explicit keyless Join opens no terminal', async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
  await page.evaluate(() => window.workspaceFixture.setJoinFailure(true));
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByText(/Outcome unconfirmed/).waitFor();
  assert.equal(await page.getByRole('heading', {name: 'Open your first terminal'}).count(), 0);
  await page.evaluate(() => window.workspaceFixture.setJoinFailure(false));
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  assert.equal(await page.evaluate(() => window.workspaceFixture.sockets.length), 0);
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.filter(c => c.method !== 'GET').length), 3);
});
test('first use: second-project setup cancellation preserves the live renderer, input target and layout', async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(() => window.workspaceFixture.api.refresh()); await chooseFirstRepository(page);
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('button', {name: 'Join project', exact: true}).click();
  await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  await page.locator('.is-connected').waitFor();
  const screen = await page.locator('.xterm').elementHandle();
  const before = await page.evaluate(() => ({layout: sessionStorage.getItem('soda-spaces:v3:1'), writes: window.workspaceFixture.calls.filter(c => c.method !== 'GET').length}));
  await page.getByRole('button', {name: 'Create project', exact: true}).click();
  await page.getByRole('radio', {name: /alice\/Beta/}).check();
  await page.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.getByRole('heading', {name: 'Configure project'}).waitFor();
  assert.equal(await page.locator('.soda-workspace-terminal:visible').count(), 0);
  await page.getByRole('button', {name: 'Cancel setup', exact: true}).click();
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
  assert(await screen?.evaluate(node => node.isConnected));
  assert.deepEqual(await page.evaluate(() => ({layout: sessionStorage.getItem('soda-spaces:v3:1'), writes: window.workspaceFixture.calls.filter(c => c.method !== 'GET').length})), before);
  assert.deepEqual(await page.evaluate(() => window.workspaceFixture.sockets.map(s => s.closed)), [0]);
});
test('first use: incomplete and failed inventories never render guessed welcome', async t => {
  const page = await fixture(t, 'page', undefined, true);
  await page.evaluate(async () => {window.workspaceFixture.setComplete(false); await window.workspaceFixture.api.refresh();});
  assert.equal(await page.getByRole('heading', {name: 'Create your first project'}).count(), 0);
  await page.getByRole('heading', {name: 'Could not load projects'}).waitFor();
  await page.evaluate(async () => {window.workspaceFixture.setStatus(503); await window.workspaceFixture.api.refresh();});
  assert.equal(await page.getByRole('heading', {name: 'Create your first project'}).count(), 0);
});

test('departure during authorization cannot dispatch a late collection read', async t => {
  const page = await fixture(t); await page.evaluate(async () => {const f = window.workspaceFixture; let release: (() => void) | undefined; f.pause(new Promise<void>(resolve => {release = resolve;})); const pending = f.api.refresh(); f.api.dispose(); release?.(); await pending;});
  assert.equal(await page.evaluate(() => window.workspaceFixture.calls.length), 1);
});
