import assert from 'node:assert/strict';
import test from 'node:test';
import path from 'node:path';
import {chromium} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
const root = path.resolve(import.meta.dirname, '../..'), origin = 'https://forgejo.example.test';
const files: Record<string, string> = payload;

test('emitted native settings link requires matching Soda operator, not site-admin markup', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  for (const mode of ['operator', 'nonoperator', 'mismatch', 'expired']) {
    const page = await browser.newPage(); const errors: string[] = []; page.on('pageerror', error => errors.push(error.message));
    let queries = 0;
    await page.route(origin + '/**', async route => {
      const pathname = new URL(route.request().url()).pathname;
      if (pathname === '/') {
        await route.fulfill({contentType: 'text/html', body: '<!doctype html><link rel="icon" href="data:,"><nav><a href="/explore/repos">Explore</a><span id="soda-settings-link" hidden data-actor="1" data-sub-url=""></span><a href="/admin">Native site admin</a></nav><input id="draft" value="unsaved"><script type="module" src="/assets/soda/forgejo/soda-settings-link.js"></script>'}); return;
      }
      if (pathname === '/-/soda/api/session') {
        queries++; assert.equal(route.request().method(), 'GET'); assert.equal(route.request().headers()['x-soda-expected-user-id'], '1');
        await route.fulfill({status: mode === 'expired' ? 401 : 200, contentType: 'application/json', body: JSON.stringify({user: {id: mode === 'mismatch' ? '2' : '1', login: 'alice'}, csrf_token: 'fixture', forgejo_url: origin, soda_operator: mode !== 'nonoperator'})}); return;
      }
      const source = files['public' + pathname]; assert(source && source.startsWith('@build/forgejo-js/'), 'navigation must not inspect runner/provider state');
      await route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.join(root, '.artifacts/forgejo-js', path.basename(source))).text()});
    });
    await page.goto(origin); await page.waitForLoadState('networkidle');
    const link = page.getByRole('link', {name: 'SodaOS settings'});
    assert.equal(await link.count(), mode === 'operator' ? 1 : 0);
    assert.equal(await page.locator('#draft').inputValue(), 'unsaved'); assert(queries >= 1);
    if (mode === 'operator') {
      assert.equal(await link.getAttribute('href'), '/?soda-view=runners');
      await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})));
      assert.equal(await link.count(), 0);
      await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true})));
      await link.waitFor(); assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
    }
    assert.deepEqual(errors, []); await page.close();
  }
});
