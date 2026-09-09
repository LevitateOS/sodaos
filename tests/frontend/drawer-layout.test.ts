import {chromium} from 'playwright';
// Opt-in real Chromium layout of source components with synthetic API responses.
// Not a native Forgejo/OAuth/helper journey. No retained server or credential used.
import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import terminalLock from '../../appliance/terminal-assets.lock.json';
import path from 'node:path';
declare global { interface Window { fixtureSocketCount: number; fixtureClosedCount: number } }
const root = path.resolve(import.meta.dirname, '../..');

test('integrated drawer source: desktop/mobile themes and native-page coexistence', {skip: process.env.SODA_DRAWER_LAYOUT !== '1', timeout: 60000}, async t => {

  const assets = new Map<string, Uint8Array<ArrayBuffer>>();
  for (const item of terminalLock) for (const file of item.files) {
    const bytes = await Bun.file(path.join(root, '.artifacts/browser-terminal/vendor', file.file)).bytes();
    assert.equal(new Bun.CryptoHasher('sha256').update(bytes).digest('hex'), file.sha256);
    assets.set('@build/terminal-assets/' + file.file, bytes);
  }
  for (const source of Object.values(payload)) {
    if (source.startsWith('@build/forgejo-js/')) assets.set(source, await Bun.file(path.join(root, '.artifacts/forgejo-js', path.basename(source))).bytes());
  }
  const env = 'p0123456789abcdef01234567', fingerprint = 'SHA256:' + 'A'.repeat(43);
  let running = true; const writes: string[] = [], errors: string[] = [];
  const server = Bun.serve({hostname: '127.0.0.1', port: 0, async fetch(req) {
    try {
      const url = new URL(req.url);
      if (req.method !== 'GET') {
        if (req.method !== 'POST' || !url.pathname.includes('/terminal-sessions/')) { writes.push(req.method); return new Response(null, {status: 405}); }
        const input: unknown = await req.json();
        assert(input && typeof input === 'object' && 'action' in input && typeof input.action === 'string');
        const action = input.action; writes.push(action);
        const now = Math.floor(Date.now()/1000), until = action === 'return' ? 0 : now+1800;
        return Response.json({terminal: {id: 'a'.repeat(32), request_id: 'b'.repeat(32), environment_id: env, repository_id: '7', user_id: '1', login: 'fixture', name: '', created_at: now, hard_until: now+43200, retain_until: until, effective_until: until || now+43200, ready: true, attached: true, state: 'ready'}});
      }
      if (url.pathname.startsWith('/-/soda/api/')) {
        let body;
        if (url.pathname.endsWith('/session')) body = {user: {id: '1', login: 'component-fixture'}, csrf_token: 'synthetic-csrf', forgejo_url: url.origin};
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
        if (!source || (source.startsWith('@build/') && !assets.has(source))) {return new Response(null, {status: 404});}
        const contentType = /\.m?js$/.test(source) ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : 'application/octet-stream';
        return new Response(assets.get(source) ?? Bun.file(path.join(root, source)), {headers: {'Content-Type': contentType}});
      }
      const footerPart = (await Bun.file(path.join(root, 'appliance/forgejo/templates/custom/footer.tmpl')).text()).split('{{if .IsSigned}}\n<div id="soda-notification-preview"')[0];
      assert(footerPart);
      const footer = footerPart.replace(/{{AssetUrlPrefix}}/g, '/assets').replace(/{{AppSubUrl}}/g, '').replace(/{{\.Repository.ID}}/g, '7').replace(/{{if \.IsSigned}}true{{else}}false{{end}}/g, 'true').replace(/{{if \.IsSigned}}{{\.SignedUserID}}{{end}}/g, '1').replace(/{{[\s\S]*?}}/g, '');
      return new Response(`<!doctype html><title>Soda drawer component fixture — not native proof</title><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><link rel="stylesheet" href="/assets/sodaspaces.css"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><body><main class="soda-page" data-signed="true"><div class="repo-header"><div class="repo-buttons"><button id="native">Native fixture action</button></div></div><input id="native-edit" value="unsaved fixture"></main>${footer}`, {headers: {'Content-Type': 'text/html'}});
    } catch {return new Response(null, {status: 500});}
  }});
  const origin = server.url.origin;
  t.after(() => server.stop(true));
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const out = path.join(root, '.artifacts/merge-5c845a7-560265b/layout-' + Date.now()); await mkdir(out, {recursive: true, mode: 0o700});
  for (const width of [1440, 900, 390, 320]) for (const theme of ['light', 'dark'] as const) for (const state of [true, false]) {
    running = state;
    const page = await browser.newPage({viewport: {width, height: 900}, colorScheme: theme});
    page.on('pageerror', e => errors.push(e.message));
    await page.addInitScript(() => {
      // Renderer transport double only. No native terminal or service is contacted.
      window.fixtureSocketCount = 0;
      window.fixtureClosedCount = 0;
      Object.defineProperty(window, 'WebSocket', {value: class {
        readyState = 0; bufferedAmount = 0;
        onopen?: () => void; onclose?: () => void; onmessage?: (event: {data: string}) => void;
        constructor() { window.fixtureSocketCount++; setTimeout(() => { this.readyState = 1; this.onopen?.(); }, 0); }
        send(data: string) { if (JSON.parse(data).csrf_token) setTimeout(() => { this.onmessage?.({data: JSON.stringify({type: 'session', id: 'a'.repeat(32), request_id: JSON.parse(data).request_id, attachment_id: 'c'.repeat(32)})}); this.onmessage?.({data: '{"type":"ready"}'}); this.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('Renderer fixture only — no native shell'.replace('—', '-'))})}); }, 0); }
        close() { window.fixtureClosedCount++; this.readyState = 3; this.onclose?.(); }
      }});
    });
    await page.goto(origin); await page.evaluate(theme => document.documentElement.style.colorScheme = theme, theme);
    await page.locator('#sodaspaces-button').click();
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.match(await page.locator('#sodaspaces-status').innerText(), state ? /running/ : /stopped/);
    const metrics = await page.locator('#sodaspaces-drawer').evaluate(dialog => ({width: dialog.getBoundingClientRect().width, overflow: dialog.scrollWidth > dialog.clientWidth, heights: [...dialog.querySelectorAll('button')].filter(b => b.getClientRects().length).map(b => b.getBoundingClientRect().height)}));
    assert.equal(metrics.width, width > 800 ? width / 2 : width); assert(!metrics.overflow); assert(metrics.heights.every(h => h === 44), JSON.stringify(metrics));
    const tabbar = await page.getByRole('tablist').boundingBox(), hide = await page.locator('#sodaspaces-close').boundingBox();
    assert(tabbar && hide);
    assert(tabbar.x + tabbar.width <= hide.x, 'view tabs must not overlap Hide, including at 320px');
    assert.equal(await page.locator('#native-edit').inputValue(), 'unsaved fixture');
    assert.equal(await page.locator('#sodaspaces-drawer .soda-page').count(), 0);
    if (state) {
      await page.getByRole('button', {name: 'Open terminal', exact: true}).click();
      await page.getByText('Connected as fixture.', {exact: true}).waitFor();
      assert.equal(await page.locator('.xterm').count(), 1);
      assert.equal(await page.locator('.xterm-helper-textarea').evaluate(el => getComputedStyle(el).padding), '0px');
      await page.keyboard.press('Escape'); assert(await page.locator('#sodaspaces-drawer').isVisible());
      await page.keyboard.press('Control+Shift+Enter');
      assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'End terminal');
      const canvas = await page.locator('.soda-terminal-screen').boundingBox();
      assert(canvas);
      assert(canvas.height > 650, JSON.stringify(canvas));
      if (width > 800) {
        await page.locator('#native-edit').fill('editing native page while developing');
        await page.locator('#native').click();
        assert(await page.locator('#sodaspaces-drawer').isVisible());
        await page.locator('#sodaspaces-divider').focus(); await page.keyboard.press('ArrowLeft');
        assert.equal(await page.locator('#sodaspaces-divider').getAttribute('aria-valuenow'), '55');
        await page.keyboard.press('ArrowRight');
      }
      await page.getByRole('tab', {name: 'Access', exact: true}).click();
      await page.locator('#sodaspaces-command').waitFor();
      await page.getByRole('tab', {name: 'Terminal', exact: true}).click();
      assert.equal(await page.evaluate(() => window.fixtureSocketCount), 1);
      assert.equal(await page.evaluate(() => window.fixtureClosedCount), 0);
    }
    await page.screenshot({path: path.join(out, `${width}-${theme}-${state ? 'running' : 'stopped'}.png`)});
    const away = state ? page.waitForResponse(r => r.request().method() === 'POST' && r.url().includes('/terminal-sessions/')) : Promise.resolve();
    await page.locator('#sodaspaces-close').click(); await away;
    const returned = state ? page.waitForResponse(r => r.request().method() === 'POST' && r.url().includes('/terminal-sessions/')) : Promise.resolve();
    await page.locator('#sodaspaces-button').click(); await returned;
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.equal(await page.locator('.soda-terminal').count(), state ? 1 : 0);
    assert.equal(await page.evaluate(() => window.fixtureClosedCount), 0);
    if (state) assert.equal(await page.evaluate(() => window.fixtureSocketCount), 1);
    await page.close();
  }
  assert.deepEqual(writes, Array.from({length: 8}, () => ['hide', 'return']).flat()); assert.deepEqual(errors, []);
});
