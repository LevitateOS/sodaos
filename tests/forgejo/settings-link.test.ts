import assert from 'node:assert/strict';
import test from 'node:test';
import path from 'node:path';
import {chromium} from 'playwright';
import type {Page} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
const root = path.resolve(import.meta.dirname, '../..'), origin = 'https://forgejo.example.test';
const files: Record<string, string> = payload;

type SessionMode = 'operator' | 'nonoperator' | 'mismatch' | 'expired';
interface NativeContext {target: string; view: string; actor?: string; nativeHost?: boolean}

// Synthetic native markup isolates the emitted navigation module. The Go host
// tests separately verify which queries/routes may emit this content context.
async function navigationFixture(page: Page, prefix: string, context: NativeContext, mode: SessionMode) {
  let queries = 0;
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const target = origin + prefix + context.target;
  await page.route(origin + '/**', async route => {
    const pathname = new URL(route.request().url()).pathname;
    if (pathname === new URL(target).pathname) {
      const hostClass = context.nativeHost === false ? 'page-content' : 'page-content soda-page soda-native-page';
      const content = context.view ? `<main class="${hostClass}"><div id="soda-native-content" data-view="${context.view}" data-actor="${context.actor || '1'}" data-repository-id="${context.view === 'repository-spaces' ? '7' : ''}"></div></main>` : '<main>Native content</main>';
      await route.fulfill({contentType: 'text/html', body: `<!doctype html><link rel="icon" href="data:,">
        <nav id="navbar"><a href="${prefix}/explore/repos">Explore</a>
          <a id="soda-spaces-link" class="item" href="${prefix}/?soda-view=spaces">Spaces</a>
          <span id="soda-settings-link" hidden data-actor="1" data-sub-url="${prefix}"></span>
          <a href="${prefix}/admin">Native site admin</a></nav>
        ${content}<input id="draft" value="unsaved"><script type="module" src="${prefix}/assets/soda/forgejo/soda-settings-link.js"></script>`});
      return;
    }
    if (pathname === prefix + '/-/soda/api/session') {
      queries++;
      assert.equal(route.request().method(), 'GET');
      assert.equal(route.request().headers()['x-soda-expected-user-id'], '1');
      await route.fulfill({status: mode === 'expired' ? 401 : 200, contentType: 'application/json', body: JSON.stringify({user: {id: mode === 'mismatch' ? '2' : '1', login: 'alice'}, csrf_token: 'fixture', forgejo_url: origin, soda_operator: mode !== 'nonoperator'})});
      return;
    }
    const source = files['public' + pathname.slice(prefix.length)];
    assert(source && source.startsWith('@build/forgejo-js/'), 'navigation must not inspect runner/provider state');
    await route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.join(root, '.artifacts/forgejo-js', path.basename(source))).text()});
  });
  await page.goto(target);
  await page.waitForLoadState('networkidle');
  assert(queries >= 1);
  assert.deepEqual(errors, []);
  return errors;
}

async function assertCurrent(page: Page, name: string, expected: boolean) {
  const link = page.getByRole('link', {name, exact: true});
  assert.equal(await link.evaluate(element => element.classList.contains('active')), expected, name);
  assert.equal(await link.getAttribute('aria-current'), expected ? 'page' : null, name);
}

test('emitted native settings link requires matching Soda operator, not site-admin markup', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  for (const mode of ['operator', 'nonoperator', 'mismatch', 'expired'] as const) {
    const page = await browser.newPage();
    const errors = await navigationFixture(page, '', {target: '/?soda-view=runners', view: 'runners'}, mode);
    const link = page.getByRole('link', {name: 'Runners', exact: true});
    assert.equal(await link.count(), mode === 'operator' ? 1 : 0);
    await assertCurrent(page, 'Spaces', false);
    assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
    assert.equal(await page.getByRole('link', {name: 'Native site admin'}).getAttribute('href'), '/admin');
    if (mode === 'operator') {
      assert.equal(await link.getAttribute('href'), '/?soda-view=runners');
      await assertCurrent(page, 'Runners', true);
      await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})));
      assert.equal(await link.count(), 0);
      await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true})));
      await link.waitFor();
      await assertCurrent(page, 'Runners', true);
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      await page.evaluate(() => window.dispatchEvent(new Event('soda-session-retired')));
      assert.equal(await link.count(), 0);
    }
    assert.deepEqual(errors, []);
    await page.close();
  }
});

test('emitted navigation marks only the matching validated native view', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const cases: readonly (NativeContext & {current: string})[] = [
    {target: '/?soda-view=spaces', view: 'spaces', current: 'Spaces'},
    {target: '/?soda-view=runners', view: 'runners', current: 'Runners'},
    {target: '/?soda-view=repository-spaces&repository_id=7', view: 'repository-spaces', current: ''},
    {target: '/', view: '', current: ''},
    {target: '/?soda-view=spaces&soda-view=runners', view: '', current: ''},
    {target: '/?soda-view=runners&repository_id=1', view: '', current: ''},
    {target: '/alice/repo?soda-view=spaces', view: '', current: ''},
    {target: '/alice/repo?soda-view=spaces', view: 'spaces', nativeHost: false, current: ''},
    {target: '/?soda-view=spaces', view: 'spaces', actor: '2', current: ''},
    {target: '/?soda-view=unknown', view: 'unknown', current: ''},
  ];
  for (const prefix of ['', '/forge']) {
    for (const context of cases) {
      const page = await browser.newPage();
      await navigationFixture(page, prefix, context, 'operator');
      await assertCurrent(page, 'Spaces', context.current === 'Spaces');
      await assertCurrent(page, 'Runners', context.current === 'Runners');
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      assert.equal(await page.getByRole('link', {name: 'Spaces', exact: true}).getAttribute('href'), prefix + '/?soda-view=spaces');
      assert.equal(await page.getByRole('link', {name: 'Runners', exact: true}).getAttribute('href'), prefix + '/?soda-view=runners');
      assert.equal(await page.locator('#navbar [aria-current="page"]').count(), context.current ? 1 : 0);
      await page.close();
    }
  }
  // Spaces is the native page being viewed even when Soda authorization fails.
  const page = await browser.newPage();
  await navigationFixture(page, '', {target: '/?soda-view=spaces', view: 'spaces'}, 'expired');
  await assertCurrent(page, 'Spaces', true);
  assert.equal(await page.getByRole('link', {name: 'Runners', exact: true}).count(), 0);
  await page.close();
});
