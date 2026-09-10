import assert from 'node:assert/strict';
import test from 'node:test';
import path from 'node:path';
import {chromium} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';

const origin = 'https://forgejo.example.test';
const files: Record<string, string> = payload;
const session = {user: {id: '1', login: 'alice'}, csrf_token: 'fixture', forgejo_url: origin};
const shell = `<!doctype html><nav id="navbar"><span id="soda-settings-link" data-actor="1"></span><a class="link-action" href="" data-url="/user/logout">Sign out</a><a class="link-action" data-url="/unrelated">Unrelated</a></nav><main><input id="draft" value="unsaved"><div id="soda-native-content" data-view="spaces" data-actor="1" data-title="Spaces" data-document-title="Spaces" data-destination="/-/soda/spaces"></div></main>`;

test('entry connection is bounded, actor checked, and never restarts on focus or failure', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  for (const mode of ['valid', 'missing', 'mismatch', 'unavailable', 'failed-return', 'failed-valid', 'suppressed']) {
    const page = await browser.newPage(); let logins = 0;
    await page.route(origin + '/**', async route => {
      const pathname = new URL(route.request().url()).pathname;
      if (pathname === '/') return route.fulfill({contentType: 'text/html', body: shell + '<script type="module" src="/assets/soda/forgejo/soda-native-page.js"></script>'});
      if (pathname === '/-/soda/api/session') return route.fulfill({status: mode === 'unavailable' ? 503 : ['missing','failed-return'].includes(mode) ? 401 : 200, contentType: 'application/json', body: JSON.stringify({...session, user: {...session.user, id: mode === 'mismatch' ? '2' : '1'}})});
      if (pathname === '/-/soda/login') {logins++; return route.fulfill({contentType: 'text/html', body: 'Consent fixture'});}
      const source = files['public' + pathname]; assert(source && source.startsWith('@build/forgejo-js/'), pathname);
      await route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.resolve('.artifacts/forgejo-js', path.basename(source))).text()});
    });
    if (mode === 'suppressed') await page.addInitScript(() => localStorage.setItem('soda-signout', 'fixture'));
    await page.goto(origin + '/?soda-view=spaces' + (mode.startsWith('failed-') ? '&soda-connect=failed' : ''));
    await page.waitForLoadState('networkidle');
    if (mode === 'valid') assert.equal(await page.getByRole('link', {name: 'Open Spaces'}).count(), 1);
    else if (mode !== 'missing') assert.equal(await page.getByRole('button', {name: 'Retry connection'}).count(), 1);
    assert.equal(logins, mode === 'missing' ? 1 : 0);
    await page.evaluate(() => {window.dispatchEvent(new Event('focus')); document.dispatchEvent(new Event('visibilitychange'));});
    assert.equal(logins, mode === 'missing' ? 1 : 0);
    if (mode === 'missing') {
      await page.goto(origin + '/?soda-view=spaces'); await page.getByRole('button', {name: 'Retry connection'}).waitFor();
      assert.equal(logins, 1, 'failed callback must not loop');
      await page.getByRole('button', {name: 'Retry connection'}).click(); await page.waitForURL('**/-/soda/login?**'); assert.equal(logins, 2);
    } else assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
    await page.close();
  }
});

test('coordinated logout captures keyboard activation and retires other tabs before native POST', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  for (const mode of ['session', 'pending', 'soda-failure', 'native-failure', 'actor-change']) {
    const context = await browser.newContext(); const page = await context.newPage(); const other = await context.newPage();
    const calls: string[] = [];
    await context.route(origin + '/**', async route => {
      const pathname = new URL(route.request().url()).pathname;
      if (pathname === '/') return route.fulfill({contentType: 'text/html', body: shell.replace(/<div id="soda-native-content"[^>]*><\/div>/, '') + '<script type="module" src="/assets/soda-connection.js"></script>'});
      if (pathname.startsWith('/-/soda/api/')) {
        calls.push(route.request().method() + ' ' + pathname);
        if (pathname.endsWith('/session')) return route.fulfill({status: mode === 'pending' ? 401 : mode === 'actor-change' ? 403 : 200, contentType: 'application/json', body: JSON.stringify(session)});
        if (pathname.endsWith('/login/cancel') && route.request().method() === 'GET') return route.fulfill({status: mode === 'pending' ? 200 : 204, ...(mode === 'pending' ? {contentType: 'application/json', body: JSON.stringify({csrf_token: 'a'.repeat(43)})} : {})});
        return route.fulfill({status: mode === 'soda-failure' ? 503 : 204});
      }
      const source = files['public' + pathname]; assert(source && source.startsWith('@build/forgejo-js/'), pathname);
      await route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.resolve('.artifacts/forgejo-js', path.basename(source))).text()});
    });
    for (const tab of [page, other]) {
      await tab.goto(origin); await tab.waitForLoadState('networkidle');
      await tab.evaluate(() => {
        window.addEventListener('soda-session-retired', () => document.body.dataset.retired = 'true');
        document.addEventListener('click', event => {
          if (!(event.target instanceof Element) || !event.target.closest('[data-url="/user/logout"]')) return;
          event.preventDefault(); document.body.dataset.nativeCalls = String(Number(document.body.dataset.nativeCalls || '0') + 1);
        });
      });
    }
    await page.getByRole('link', {name: 'Sign out', exact: true}).focus(); await page.keyboard.press('Enter');
    await page.getByRole('alert').waitFor();
    assert.equal(await page.locator('body').getAttribute('data-retired'), 'true');
    await other.waitForFunction(() => document.body.dataset.retired === 'true');
    const failed = ['soda-failure', 'actor-change'].includes(mode);
    assert.equal(await page.locator('body').getAttribute('data-native-calls'), failed ? null : '1');
    assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
    if (failed) {
      await page.getByRole('button', {name: 'Sign out of Forgejo only'}).click();
      assert.equal(await page.locator('body').getAttribute('data-native-calls'), '1');
    } else {
      await page.getByRole('link', {name: 'Sign out', exact: true}).click();
      assert.equal(await page.locator('body').getAttribute('data-native-calls'), '1', 'duplicate activation must not replay');
      assert.match(await page.getByRole('alert').innerText(), /Forgejo sign-out is still pending/);
    }
    if (mode === 'pending') assert(!calls.some(call => call.endsWith('/session/logout')));
    if (mode === 'actor-change') assert.equal(calls.length, 1, 'must not adopt another actor for cancellation');
    await context.close();
  }
});
