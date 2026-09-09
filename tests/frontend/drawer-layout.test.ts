import {chromium} from 'playwright';
// Real emitted native adapter/Lit/xterm, synthetic API/IO and a fixture form.
// Not Forgejo OAuth, native process continuity or physical-device keyboard proof.
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import terminalLock from '../../appliance/terminal-assets.lock.json';
import type {TerminalMetadata} from '../../appliance/forgejo/public/assets/sodaspaces-api';
import {newManagedTerminal, projectView, terminalMenu} from '../installed/sodaspaces-controls';
import path from 'node:path';
declare global {interface Window {fixtureSocketCount: number; fixtureClosedCount: number; fixtureFrames: Record<string, unknown>[]; fixtureActions: string[]; fixtureMetadata?: TerminalMetadata; fixtureDirty: boolean}}
const root = path.resolve(import.meta.dirname, '../..');

test('integrated drawer source: measured cells, themes, compact/native focus and navigation', {skip: process.env.SODA_DRAWER_LAYOUT !== '1', timeout: 90000}, async t => {
  const assets = new Map<string, Uint8Array<ArrayBuffer>>();
  for (const item of terminalLock) for (const file of item.files) {
    const bytes = await Bun.file(path.join(root, '.artifacts/browser-terminal/vendor', file.file)).bytes();
    assert.equal(new Bun.CryptoHasher('sha256').update(bytes).digest('hex'), file.sha256); assets.set('@build/terminal-assets/' + file.file, bytes);
  }
  for (const source of Object.values(payload)) if (source.startsWith('@build/forgejo-js/')) assets.set(source, await Bun.file(path.join(root, '.artifacts/forgejo-js', path.basename(source))).bytes());
  const env = 'p0123456789abcdef01234567', fingerprint = 'SHA256:' + 'A'.repeat(43);
  let running = true; const writes: string[] = [], errors: string[] = [];
  const server = Bun.serve({hostname: '127.0.0.1', port: 0, async fetch(req) {
    try {
      const url = new URL(req.url);
      if (req.method !== 'GET') {writes.push(req.method); return new Response(null, {status: 405});}
      if (url.pathname.startsWith('/-/soda/api/')) {
        let body;
        if (url.pathname.endsWith('/session')) body = {user: {id: '1', login: 'component-fixture'}, csrf_token: 'synthetic-csrf', forgejo_url: url.origin};
        else if (url.pathname.endsWith('/spaces')) body = {complete: true, items: [{environment: {id: env, repository_id: '7', owner_id: '1', name: 'Shared work', repository: 'fixture/shared-work', provisioned: true}, login: 'fixture', environment_administrator: true, native_unavailable: false, authority_unavailable: false, observed: {id: env, running}, terminals: []}]};
        else if (url.pathname.includes('/terminal-sessions/')) body = {terminal: null};
        else if (url.pathname.endsWith('/forgejo/me')) body = {id: '1'};
        else if (url.search) body = {repository: {id: '7', owner: 'fixture', name: 'shared-work'}, can_create: false, items: [{id: env, repository_id: '7'}]};
        else if (url.pathname.endsWith('/development-keys')) body = {items: [{id: '1', fingerprint}]};
        else if (url.pathname.endsWith('/lifecycle')) body = {environment: {id: env, running}, boot_enabled: running};
        else if (url.pathname.endsWith('/connection')) body = {login: 'fixture', connection: {environment: {id: env, running, ip: '10.89.0.2'}, fingerprint}};
        else body = {environment: {id: env, repository_id: '7', provisioned: true}, login: 'fixture', environment_administrator: true, native_unavailable: false, authority_unavailable: false, observed: {id: env, running}};
        return Response.json(body);
      }
      if (url.pathname.startsWith('/assets/')) {
        const source = Object.entries(payload).find(([target]) => target === 'public' + url.pathname)?.[1];
        if (!source || source.startsWith('@build/') && !assets.has(source)) return new Response(null, {status: 404});
        const contentType = /\.m?js$/.test(source) ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : 'application/octet-stream';
        return new Response(assets.get(source) ?? Bun.file(path.join(root, source)), {headers: {'Content-Type': contentType}});
      }
      const footerPart = (await Bun.file(path.join(root, 'appliance/forgejo/templates/custom/footer.tmpl')).text()).split('{{if .IsSigned}}\n<div id="soda-notification-preview"')[0]; assert(footerPart);
      const footer = footerPart.replace(/{{AssetUrlPrefix}}/g, '/assets').replace(/{{AppSubUrl}}/g, '').replace(/{{\.Repository.ID}}/g, '7').replace(/{{if \.IsSigned}}true{{else}}false{{end}}/g, 'true').replace(/{{if \.IsSigned}}{{\.SignedUserID}}{{end}}/g, '1').replace(/{{[\s\S]*?}}/g, '');
      return new Response(`<!doctype html><title>Drawer component fixture — not native proof</title><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/fonts/fonts.css"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><link rel="stylesheet" href="/assets/sodaspaces.css"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><body><main data-signed="true"><div class="repo-header"><div class="repo-buttons"><button id="native">Native fixture action</button></div></div><form><input id="native-edit" value="unsaved fixture"></form></main>${footer}`, {headers: {'Content-Type': 'text/html'}});
    } catch {return new Response(null, {status: 500});}
  }});
  t.after(() => server.stop(true));
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const out = path.join(root, '.artifacts/spaces-step5-f28f86e/layout-' + Date.now()); await mkdir(out, {recursive: true, mode: 0o700});
  const matrix: unknown[] = [];
  for (const width of [1440, 1024, 900, 390, 320]) for (const theme of ['light', 'dark'] as const) for (const state of [true, false]) {
    running = state;
    const page = await browser.newPage({viewport: {width, height: 900}, colorScheme: theme}); page.setDefaultTimeout(5000);
    page.on('pageerror', error => errors.push(error.message));
    await page.addInitScript(() => {
      window.fixtureSocketCount = window.fixtureClosedCount = 0; window.fixtureFrames = []; window.fixtureActions = []; window.fixtureDirty = false;
      const object = (value: unknown): Record<string, unknown> => {if (!value || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid fixture frame'); return value as Record<string, unknown>;};
      const nativeFetch = window.fetch;
      const fixtureFetch = async (url: RequestInfo | URL, init?: RequestInit) => {
        const metadata = window.fixtureMetadata;
        if (!String(url).includes('/terminal-sessions/') || !metadata) return nativeFetch(url, init);
        if (init?.method === 'POST') {
          const body = object(JSON.parse(typeof init.body === 'string' ? init.body : '{}'));
          if (!['hide', 'return', 'retain'].includes(String(body.action))) throw Error('Unapproved fixture lifetime action');
          window.fixtureActions.push(String(body.action));
          if (body.action === 'return') metadata.retain_until = 0;
          else if (body.action === 'retain' || !metadata.retain_until) metadata.retain_until = Math.min(metadata.hard_until, Math.floor(Date.now() / 1000) + (body.seconds === 7200 ? 7200 : 1800));
          metadata.effective_until = metadata.retain_until || metadata.hard_until;
        }
        return Response.json({terminal: metadata});
      };
      Object.defineProperty(window, 'fetch', {value: fixtureFetch});
      document.addEventListener('input', event => {if (event.target instanceof HTMLInputElement && event.target.id === 'native-edit') window.fixtureDirty = event.target.value !== 'unsaved fixture';});
      window.addEventListener('beforeunload', event => {if (window.fixtureDirty) {event.preventDefault(); event.returnValue = '';}});
      Object.defineProperty(window, 'WebSocket', {value: class {
        readyState = 0; bufferedAmount = 0;
        onopen?: () => void; onclose?: () => void; onmessage?: (event: {data: string}) => void;
        constructor() {window.fixtureSocketCount++; setTimeout(() => {this.readyState = 1; this.onopen?.();}, 0);}
        send(data: string) {
          const frame = object(JSON.parse(data)); window.fixtureFrames.push(frame);
          if (frame.action !== 'create') return;
          if (typeof frame.request_id !== 'string') throw Error('Missing fixture correlation');
          const now = Math.floor(Date.now() / 1000);
          window.fixtureMetadata = {id: 'a'.repeat(32), request_id: frame.request_id, environment_id: 'p0123456789abcdef01234567', repository_id: '7', user_id: '1', login: 'fixture', name: typeof frame.name === 'string' ? frame.name : '', created_at: now, hard_until: now + 43200, retain_until: 0, effective_until: now + 43200, ready: true, attached: true, state: 'ready'};
          this.onmessage?.({data: JSON.stringify({type: 'session', id: 'a'.repeat(32), request_id: frame.request_id, attachment_id: 'c'.repeat(32)})});
          this.onmessage?.({data: '{"type":"ready"}'}); this.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('Renderer fixture only; no native shell\r\n$ ')})});
        }
        close() {window.fixtureClosedCount++; this.readyState = 3; this.onclose?.();}
      }});
    });
    await page.goto(server.url.href); await page.evaluate(theme => document.documentElement.style.colorScheme = theme, theme);
    await page.locator('#sodaspaces-button').click(); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    const drawer = page.locator('#sodaspaces-drawer');
    const presentation = await drawer.evaluate(node => {
      const style = getComputedStyle(node), probe = document.createElement('span');
      probe.style.color = 'var(--soda-page-canvas)'; node.append(probe);
      const canvas = getComputedStyle(probe).color; probe.remove();
      return {background: style.backgroundColor, canvas, scheme: style.colorScheme, font: style.fontFamily,
        nativeBackground: getComputedStyle(document.body).backgroundColor,
        nativeFont: getComputedStyle(document.querySelector('main')!).fontFamily};
    });
    assert.equal(presentation.background, presentation.canvas);
    assert.equal(presentation.scheme, theme); assert.match(presentation.font, /Barlow/);
    assert.equal(presentation.nativeBackground, 'rgba(0, 0, 0, 0)', 'island tokens must not activate full-page styling');
    assert.doesNotMatch(presentation.nativeFont, /Barlow/);
    const metrics = await drawer.evaluate(node => ({width: node.getBoundingClientRect().width, left: node.getBoundingClientRect().left, overflow: node.scrollWidth > node.clientWidth, compact: document.body.classList.contains('sodaspaces-compact')}));
    assert(!metrics.overflow); assert.equal(metrics.compact, width <= 900);
    if (metrics.compact) assert(Math.abs(metrics.width - width) <= 1); else assert(metrics.left >= 480);
    assert.equal(await page.locator('#native-edit').inputValue(), 'unsaved fixture');
    assert.equal(await drawer.locator('.soda-page').count(), 0);
    if (state) {
      await newManagedTerminal(page, 'fixture/shared-work', 'Layout terminal', env);
      await page.waitForFunction(() => window.fixtureFrames.some(f => f.type === 'resize'));
      const cells = await page.evaluate(() => window.fixtureFrames.filter(f => f.type === 'resize').at(-1));
      assert(cells && Number(cells.rows) >= 12 && Number(cells.cols) > 0); if (!metrics.compact) assert(Number(cells.cols) >= 56, JSON.stringify({metrics, cells}));
      matrix.push({viewport: width, theme, running: state, ...metrics, cols: cells.cols, rows: cells.rows});
      const host = await page.locator('.soda-workspace-terminal').elementHandle(), screen = await page.locator('.xterm').elementHandle();
      assert.equal(await page.locator('.xterm-helper-textarea').evaluate(el => getComputedStyle(el).padding), '0px');
      await page.locator('.xterm-helper-textarea').focus(); await page.keyboard.press('Escape'); assert(await drawer.isVisible());
      await page.keyboard.press('Control+Shift+Enter'); assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('aria-label')), 'Terminal actions');
      const tabbar = await page.getByRole('tablist', {name: 'Terminal sessions', exact: true}).boundingBox(), hide = await page.locator('#sodaspaces-close').boundingBox();
      assert(tabbar && hide && tabbar.y >= hide.y + hide.height, 'local tabs cannot overlap Hide');
      if (!metrics.compact) {
        await page.locator('#native-edit').focus(); await page.locator('#native').click(); assert(await drawer.isVisible());
        await page.locator('#sodaspaces-divider').press('ArrowLeft');
        const resized = await drawer.boundingBox(); assert(resized && resized.x >= 480); await page.locator('#sodaspaces-divider').press('ArrowRight');
      } else {
        await page.getByRole('button', {name: 'Forge', exact: true}).click(); assert(await drawer.isHidden());
        await page.locator('#native-edit').fill('a native draft');
        await page.locator('#native-edit').evaluate(node => {if (node instanceof HTMLInputElement) node.setSelectionRange(2, 7);});
        const form = await page.locator('#native-edit').elementHandle();
        await page.getByRole('button', {name: 'Terminal', exact: true}).click(); assert(await page.locator('#native-edit').isHidden());
        assert(await form?.evaluate(node => node.isConnected));
        const dismissed = page.waitForEvent('dialog').then(async dialog => {assert.equal(dialog.type(), 'beforeunload'); await dialog.dismiss();});
        await page.getByRole('link', {name: 'Open in Spaces', exact: true}).click(); await dismissed;
        assert.equal(page.url(), server.url.href); assert(await screen?.evaluate(node => node.isConnected));
        await page.getByRole('button', {name: 'Forge', exact: true}).click();
        assert.deepEqual(await page.locator('#native-edit').evaluate(node => node instanceof HTMLInputElement ? [node.value, node.selectionStart, node.selectionEnd] : []), ['a native draft', 2, 7]);
        await page.locator('#native-edit').fill('unsaved fixture');
        await page.getByRole('button', {name: 'Terminal', exact: true}).click();
        await page.setViewportSize({width, height: 480});
        await page.waitForFunction(() => {const f = window.fixtureFrames.filter(f => f.type === 'resize').at(-1); return Number(f?.rows) > 0 && Number(f?.rows) < 24;});
        await page.getByLabel('Terminal actions', {exact: true}).click();
        const menu = page.locator('soda-terminal .soda-menu > div'), menuBox = await menu.boundingBox(), ownerBox = await page.locator('.soda-workspace-terminal:visible').boundingBox();
        assert(menuBox && ownerBox && menuBox.y + menuBox.height <= ownerBox.y + ownerBox.height + 1, 'terminal menu must scroll inside its reduced-height owner');
        await menu.getByRole('button', {name: 'End terminal…', exact: true}).scrollIntoViewIfNeeded();
        await page.getByLabel('Terminal actions', {exact: true}).press('Escape');
        await page.setViewportSize({width, height: 900});
      }
      await terminalMenu(page, 'Environment / access'); await projectView(page, '7', 'Access'); await page.locator('[data-control=command]').waitFor();
      await page.getByRole('button', {name: 'Back to terminal', exact: true}).click();
      assert(await screen?.evaluate(node => node.isConnected)); assert(await host?.evaluate(node => node.isConnected));
      assert.deepEqual(await page.evaluate(() => window.fixtureActions), []);
    } else {
      await page.getByRole('button', {name: 'New terminal', exact: true}).click();
      assert(await page.getByRole('button', {name: 'Create terminal', exact: true}).isDisabled());
      await page.getByRole('button', {name: 'Cancel', exact: true}).click(); matrix.push({viewport: width, theme, running: state, ...metrics});
    }
    await page.screenshot({path: path.join(out, `${width}-${theme}-${state ? 'running' : 'stopped'}.png`)});
    await page.locator('#sodaspaces-close').click(); if (state) await page.waitForFunction(() => window.fixtureActions.includes('hide'));
    await page.locator('#sodaspaces-button').click(); if (state) await page.waitForFunction(() => window.fixtureActions.includes('return'));
    assert.equal(await page.locator('soda-terminal').count(), state ? 1 : 0); assert.equal(await page.evaluate(() => window.fixtureClosedCount), 0);
    assert.equal(await page.evaluate(() => window.fixtureSocketCount), state ? 1 : 0);
    assert.deepEqual(await page.evaluate(() => window.fixtureActions), state ? ['hide', 'return'] : []);
    if (metrics.compact) {await page.reload(); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor({state: 'attached'}); assert(await drawer.isHidden()); assert.equal(await page.getByRole('button', {name: 'Forge', exact: true}).getAttribute('aria-pressed'), 'true'); assert.equal(await page.evaluate(() => window.fixtureSocketCount), 0);}
    await page.close();
  }
  await Bun.write(path.join(out, 'metrics.json'), JSON.stringify(matrix, null, 2) + '\n');
  assert.deepEqual(writes, []); assert.deepEqual(errors, []); console.log(`Layout fixture evidence: ${out}`);
});
