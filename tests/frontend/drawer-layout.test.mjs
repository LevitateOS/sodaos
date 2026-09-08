// Opt-in real Chromium layout of source components with synthetic API responses.
// Not a native Forgejo/OAuth/helper journey. No retained server or credential used.
import test from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {readFile, mkdir} from 'node:fs/promises';
import {createRequire} from 'node:module';
import path from 'node:path';
import {createHash} from 'node:crypto';
const root = path.resolve(import.meta.dirname, '../..');

test('integrated drawer source: desktop/mobile themes and native-page coexistence', {skip: process.env.SODA_DRAWER_LAYOUT !== '1'}, async t => {
  const require = createRequire(path.join(root, 'cockpit/package.json'));
  const {chromium} = require('playwright');
  const payload = JSON.parse(await readFile(path.join(root, 'internal/nativebuild/forgejo-payload.json'), 'utf8'));
  const vendor = new Map();
  for (const item of JSON.parse(await readFile(path.join(root, 'appliance/terminal-assets.lock.json'), 'utf8'))) for (const file of item.files) {
    const bytes = await readFile(path.join(root, '.artifacts/browser-terminal/vendor', file.file));
    assert.equal(createHash('sha256').update(bytes).digest('hex'), file.sha256);
    vendor.set('@build/terminal-assets/' + file.file, bytes);
  }
  const env = 'p0123456789abcdef01234567', fingerprint = 'SHA256:' + 'A'.repeat(43);
  let origin, running = true; const writes = [], errors = [];
  const server = createServer(async (req, res) => {
    try {
      const url = new URL(req.url, origin);
      if (req.method !== 'GET') { writes.push(req.method); res.writeHead(405).end(); return; }
      if (url.pathname.startsWith('/-/soda/api/')) {
        let body;
        if (url.pathname.endsWith('/session')) body = {user: {id: '1', login: 'component-fixture'}, csrf_token: 'synthetic-csrf', forgejo_url: origin};
        else if (url.pathname.endsWith('/forgejo/me')) body = {id: '1'};
        else if (url.search) body = {repository: {id: '7', owner: 'fixture', name: 'shared-work'}, can_create: false, items: [{id: env, repository_id: '7'}]};
        else if (url.pathname.endsWith('/development-keys')) body = {items: [{id: '1', fingerprint}]};
        else if (url.pathname.endsWith('/lifecycle')) body = {environment: {id: env, running}, boot_enabled: running};
        else if (url.pathname.endsWith('/connection')) body = {login: 'fixture', connection: {environment: {id: env, running, ip: '10.89.0.2'}, fingerprint}};
        else body = {environment: {id: env, repository_id: '7', provisioned: true}, login: 'fixture', environment_administrator: true, native_unavailable: false, authority_unavailable: false, observed: {id: env, running}};
        res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(body)); return;
      }
      if (url.pathname.startsWith('/assets/')) {
        const source = payload['public' + url.pathname];
        if (!source || (source.startsWith('@build/') && !vendor.has(source))) {res.writeHead(404).end(); return;}
        res.setHeader('Content-Type', /\.m?js$/.test(source) ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : 'application/octet-stream');
        res.end(vendor.get(source) || await readFile(path.join(root, source))); return;
      }
      const footer = (await readFile(path.join(root, 'appliance/forgejo/templates/custom/footer.tmpl'), 'utf8')).split('{{if .IsSigned}}\n<div id="soda-notification-preview"')[0]
        .replace(/{{AppSubUrl}}/g, '').replace(/{{\.Repository.ID}}/g, '7').replace(/{{if \.IsSigned}}true{{else}}false{{end}}/g, 'true').replace(/{{if \.IsSigned}}{{\.SignedUserID}}{{end}}/g, '1').replace(/{{[\s\S]*?}}/g, '');
      res.setHeader('Content-Type', 'text/html');
      res.end(`<!doctype html><title>Soda drawer component fixture — not native proof</title><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css"><link rel="stylesheet" href="/assets/sodaspaces.css"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><body><main class="soda-page" data-signed="true"><div class="repo-header"><div class="repo-buttons"><button id="native">Native fixture action</button></div></div><input id="native-edit" value="unsaved fixture"></main>${footer}`);
    } catch {res.writeHead(500).end();}
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve)); origin = `http://127.0.0.1:${server.address().port}`;
  t.after(() => new Promise(resolve => server.close(resolve)));
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const out = path.join(root, '.artifacts/workspace-d57f128/layout-' + Date.now()); await mkdir(out, {recursive: true, mode: 0o700});
  for (const width of [1440, 900, 390, 320]) for (const theme of ['light', 'dark']) for (const state of [true, false]) {
    running = state;
    const page = await browser.newPage({viewport: {width, height: 900}, colorScheme: theme});
    page.on('pageerror', e => errors.push(e.message));
    await page.addInitScript(() => {
      // Renderer transport double only. No native terminal or service is contacted.
      window.fixtureSocketCount = 0;
      window.fixtureClosedCount = 0;
      window.WebSocket = class {
        readyState = 0; bufferedAmount = 0;
        constructor() { window.fixtureSocketCount++; setTimeout(() => { this.readyState = 1; this.onopen?.(); }, 0); }
        send(data) { if (JSON.parse(data).csrf_token) setTimeout(() => { this.onmessage?.({data: '{"type":"ready"}'}); this.onmessage?.({data: JSON.stringify({type: 'output', data: btoa('Renderer fixture only — no native shell'.replace('—', '-'))})}); }, 0); }
        close() { window.fixtureClosedCount++; this.readyState = 3; this.onclose?.(); }
      };
    });
    await page.goto(origin); await page.evaluate(theme => document.documentElement.style.colorScheme = theme, theme);
    await page.locator('#sodaspaces-button').click();
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.match(await page.locator('#sodaspaces-status').innerText(), state ? /running/ : /stopped/);
    const metrics = await page.locator('#sodaspaces-drawer').evaluate(dialog => ({width: dialog.getBoundingClientRect().width, overflow: dialog.scrollWidth > dialog.clientWidth, heights: [...dialog.querySelectorAll('button')].filter(b => b.getClientRects().length).map(b => b.getBoundingClientRect().height)}));
    assert.equal(metrics.width, width > 800 ? width / 2 : width); assert(!metrics.overflow); assert(metrics.heights.every(h => h === 44), JSON.stringify(metrics));
    const tabbar = await page.getByRole('tablist').boundingBox(), hide = await page.locator('#sodaspaces-close').boundingBox();
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
      assert.equal(await page.evaluate(() => document.activeElement.textContent), 'End terminal');
      const canvas = await page.locator('.soda-terminal-screen').boundingBox();
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
    await page.locator('#sodaspaces-close').click(); await page.locator('#sodaspaces-button').click();
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.equal(await page.locator('.soda-terminal').count(), state ? 1 : 0);
    assert.equal(await page.evaluate(() => window.fixtureClosedCount), 0);
    if (state) assert.equal(await page.evaluate(() => window.fixtureSocketCount), 1);
    await page.close();
  }
  assert.deepEqual(writes, []); assert.deepEqual(errors, []);
});
