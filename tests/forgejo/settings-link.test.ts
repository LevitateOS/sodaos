import assert from 'node:assert/strict';
import test from 'node:test';
import path from 'node:path';
import {chromium} from 'playwright';
import type {Page} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
const root = path.resolve(import.meta.dirname, '../..'), origin = 'https://forgejo.example.test';
const files: Record<string, string> = payload;

type SessionMode = 'operator' | 'nonoperator' | 'mismatch' | 'expired';
interface NativeContext {target: string; view: string; actor?: string; nativeHost?: boolean; signed?: boolean; admin?: boolean}

// Synthetic markup isolates the emitted module. Go template tests separately
// render actual header/admin partials; this is not native route/authentication proof.
async function navigationFixture(page: Page, prefix: string, context: NativeContext, mode: SessionMode) {
  let queries = 0;
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const target = origin + prefix + context.target, signed = context.signed !== false;
  await page.route(origin + '/**', async route => {
    const pathname = new URL(route.request().url()).pathname;
    if (pathname === new URL(target).pathname) {
      const hostClass = context.nativeHost === false ? 'page-content' : 'page-content soda-page soda-native-page';
      const content = context.view ? `<main class="${hostClass}"><div id="soda-native-content" data-view="${context.view}" data-actor="${context.actor || '1'}"></div></main>` : '<main>Native content</main>';
      await route.fulfill({contentType: 'text/html', body: `<!doctype html><link rel="icon" href="data:,">
        <nav id="navbar"><a href="${prefix}/explore/repos">Explore</a>
          <a id="soda-spaces-link" class="item" href="${prefix}${signed ? '/?soda-view=spaces' : '/-/soda/spaces'}">Spaces</a>
          ${signed ? `<span id="soda-settings-link" hidden data-actor="1" data-sub-url="${prefix}"></span>` : ''}
        </nav>
        ${signed && context.admin ? `<div class="flex-container-nav"><div class="ui fluid vertical menu">
          <div class="header item">Soda</div>
          <a id="soda-runners-link" class="item" href="${prefix}/-/soda/settings/runners">Runners</a>
          <a id="soda-tailnet-link" class="item" href="${prefix}/-/soda/settings/tailnet">Tailnet</a></div></div>` : ''}
        ${content}<input id="draft" value="unsaved"><script type="module" src="${prefix}/assets/soda/forgejo/soda-settings-link.js"></script>`});
      return;
    }
    if (pathname === prefix + '/-/soda/api/session') {
      queries++;
      await route.fulfill({status: mode === 'expired' ? 401 : 200, contentType: 'application/json', body: JSON.stringify({user: {id: mode === 'mismatch' ? '2' : '1', login: 'alice'}, csrf_token: 'fixture', forgejo_url: origin, soda_operator: mode !== 'nonoperator'})});
      return;
    }
    const source = files['public' + pathname.slice(prefix.length)];
    assert(source && source.startsWith('@build/forgejo-js/'), 'navigation must not start OAuth or inspect settings/provider state');
    await route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.join(root, '.artifacts/forgejo-js', path.basename(source))).text()});
  });
  await page.goto(target);
  await page.waitForLoadState('networkidle');
  const checkPassive = () => {assert.equal(queries, 0, 'navigation must not depend on Soda login'); assert.deepEqual(errors, []);};
  checkPassive();
  return checkPassive;
}

async function assertCurrentSpaces(page: Page, expected: boolean) {
  const link = page.getByRole('link', {name: 'Spaces', exact: true});
  assert.equal(await link.evaluate(element => element.classList.contains('active')), expected);
  assert.equal(await link.getAttribute('aria-current'), expected ? 'page' : null);
}

test('settings stay in native administration and do not depend on a Soda session', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  for (const mode of ['operator', 'nonoperator', 'mismatch', 'expired'] as const) {
    for (const context of [{signed: false, admin: false}, {signed: true, admin: false}, {signed: true, admin: true}]) {
      const page = await browser.newPage();
      const checkPassive = await navigationFixture(page, '/forge', {target: '/admin', view: '', ...context}, mode);
      for (const name of ['Runners', 'Tailnet']) {
        assert.equal(await page.locator('#navbar').getByRole('link', {name, exact: true}).count(), 0);
        const link = page.locator('.flex-container-nav > .ui.vertical.menu').getByRole('link', {name, exact: true});
        assert.equal(await link.count(), context.admin ? 1 : 0);
        if (context.admin) assert.equal(await link.getAttribute('href'), '/forge/-/soda/settings/' + name.toLowerCase());
      }
      await page.evaluate(() => {
        window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true}));
        window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true}));
        window.dispatchEvent(new Event('soda-session-retired'));
      });
      assert.equal(await page.locator('#soda-admin-settings').count(), 0, 'no separate content toolbar');
      assert.equal(await page.locator('.flex-container-nav > .ui.vertical.menu > a').count(), context.admin ? 2 : 0);
      if (context.admin) {
        await page.locator('#soda-runners-link').focus();
        await page.keyboard.press('Tab');
        assert(await page.locator('#soda-tailnet-link').evaluate(link => document.activeElement === link));
      }
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      checkPassive();
      await page.close();
    }
  }
  const noJS = await browser.newContext({javaScriptEnabled: false});
  const page = await noJS.newPage();
  await navigationFixture(page, '', {target: '/admin', view: '', admin: true}, 'expired');
  assert.equal(await page.locator('.flex-container-nav > .ui.vertical.menu > a').count(), 2, 'sidebar entries work without JavaScript');
  await noJS.close();
});

test('global navigation marks only validated matching Spaces, never operator settings', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const cases: readonly (NativeContext & {current?: boolean})[] = [
    {target: '/?soda-view=spaces', view: 'spaces', current: true},
    {target: '/?soda-view=runners', view: 'runners'},
    {target: '/?soda-view=tailnet', view: 'tailnet'},
    {target: '/?soda-view=tailnet&repository_id=1', view: ''},
    {target: '/?soda-view=repository-spaces&repository_id=7', view: 'repository-spaces'},
    {target: '/', view: ''},
    {target: '/?soda-view=spaces&soda-view=runners', view: ''},
    {target: '/?soda-view=runners&repository_id=1', view: ''},
    {target: '/alice/repo?soda-view=spaces', view: ''},
    {target: '/alice/repo?soda-view=spaces', view: 'spaces', nativeHost: false},
    {target: '/?soda-view=spaces', view: 'spaces', actor: '2'},
    {target: '/?soda-view=unknown', view: 'unknown'},
  ];
  for (const prefix of ['', '/forge']) {
    for (const context of cases) {
      const page = await browser.newPage();
      await navigationFixture(page, prefix, context, 'expired');
      await assertCurrentSpaces(page, context.current === true);
      assert.equal(await page.getByRole('link', {name: /^(Runners|Tailnet)$/}).count(), 0);
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      assert.equal(await page.getByRole('link', {name: 'Spaces', exact: true}).getAttribute('href'), prefix + '/?soda-view=spaces');
      assert.equal(await page.locator('#navbar [aria-current="page"]').count(), context.current ? 1 : 0);
      await page.close();
    }
  }
});
