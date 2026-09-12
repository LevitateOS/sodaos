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
  for (const mode of ['valid', 'restored', 'retired', 'detached', 'missing', 'mismatch', 'unavailable', 'failed-return', 'failed-valid', 'suppressed', 'late-restored', 'late-detached', 'late-retry', 'history-valid', 'history-missing', 'history-changed', 'history-signout']) {
    const page = await browser.newPage(); let logins = 0, sessionRequests = 0, history = false, release: (() => void) | undefined;
    const pause = (['restored', 'retired', 'detached'].includes(mode) || mode.startsWith('late-')) ? new Promise<void>(resolve => {release = resolve;}) : Promise.resolve();
    await page.route(origin + '/**', async route => {
      const pathname = new URL(route.request().url()).pathname;
      if (pathname === '/') return route.fulfill({contentType: 'text/html', body: shell + '<script type="module" src="/assets/soda/forgejo/soda-native-page.js"></script>'});
      if (pathname === '/-/soda/api/session') {
        const late = mode.startsWith('late-') && ++sessionRequests === 1;
        if (late) await pause;
        return route.fulfill({status: history && mode === 'history-changed' ? 403 : mode === 'unavailable' ? 503 : (history && mode === 'history-missing') || late || ['missing','failed-return'].includes(mode) ? 401 : 200, contentType: 'application/json', body: JSON.stringify({...session, user: {...session.user, id: mode === 'mismatch' ? '2' : '1'}})});
      }
      if (pathname === '/-/soda/api/spaces') return route.fulfill({json: {items: [], complete: true}});
      if (pathname === '/-/soda/api/forgejo/me') return route.fulfill({json: {id: '1'}});
      if (pathname === '/-/soda/login') {logins++; return route.fulfill({contentType: 'text/html', body: 'Consent fixture'});}
      if (pathname === '/assets/sodaspaces-page.js' && !mode.startsWith('late-')) await pause;
      const source = files['public' + pathname]; assert(source && source.startsWith('@build/forgejo-js/'), pathname);
      await route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.resolve('.artifacts/forgejo-js', path.basename(source))).text()});
    });
    if (mode === 'suppressed') await page.addInitScript(() => localStorage.setItem('soda-signout', 'fixture'));
    const importing = mode === 'detached' ? page.waitForRequest(request => new URL(request.url()).pathname === '/assets/sodaspaces-page.js') : null;
    await page.goto(origin + '/?soda-view=spaces' + (mode.startsWith('failed-') ? '&soda-connect=failed' : ''));
    if (mode.startsWith('history-')) {
      await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
      history = true;
      await page.evaluate(signedOut => {
        document.querySelector('soda-spaces')?.setAttribute('data-old-owner', 'true');
        if (signedOut) localStorage.setItem('soda-signout', 'fixture');
        window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true}));
        window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true}));
      }, mode === 'history-signout');
      if (mode === 'history-valid') await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
      else await page.getByRole('button', {name: 'Retry connection'}).waitFor();
      await page.waitForLoadState('networkidle');
      assert.equal(await page.locator('[data-old-owner]').count(), 0);
      assert.equal(await page.locator('soda-spaces').count(), mode === 'history-valid' ? 1 : 0);
      assert.equal(logins, 0, 'history must never bootstrap OAuth');
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      await page.close(); continue;
    }
    if (mode.startsWith('late-')) {
      await page.getByRole('status').filter({hasText: 'Connecting to Soda'}).waitFor();
      if (mode === 'late-detached') await page.evaluate(() => document.getElementById('soda-native-content')?.remove());
      else {
        await page.evaluate(() => {
          window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true}));
          window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true}));
        });
        await page.getByRole('button', {name: 'Retry connection'}).waitFor();
        if (mode === 'late-retry') {
          await page.getByRole('button', {name: 'Retry connection'}).click();
          await page.locator('soda-spaces').waitFor();
        }
      }
      release?.(); await page.waitForLoadState('networkidle');
      assert.equal(logins, 0, 'retired session reply initiated OAuth');
      assert.equal(await page.locator('soda-spaces').count(), mode === 'late-retry' ? 1 : 0);
      assert.equal(await page.evaluate(() => sessionStorage.getItem('soda-entry:spaces::1')), null, 'retired reply consumed the attempt');
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      if (mode === 'late-restored') {
        await page.getByRole('button', {name: 'Retry connection'}).click();
        await page.locator('soda-spaces').waitFor();
      }
      await page.close(); continue;
    }
    if (mode === 'detached') {
      await importing;
      await page.evaluate(() => document.getElementById('soda-native-content')?.remove());
      release?.();
      await page.waitForLoadState('networkidle');
      assert.equal(await page.locator('soda-spaces').count(), 0);
      assert.equal(logins, 0);
      assert.equal(await page.locator('#draft').inputValue(), 'unsaved');
      await page.close();
      continue;
    }
    if (mode === 'restored' || mode === 'retired') {
      await page.getByRole('status').filter({hasText: 'Connecting to Soda'}).waitFor();
      await page.evaluate(retired => {
        if (retired) window.dispatchEvent(new Event('soda-session-retired'));
        else {window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})); window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true}));}
      }, mode === 'retired');
      await page.getByRole('button', {name: 'Retry connection'}).waitFor();
      release?.();
      await page.waitForLoadState('networkidle');
      assert.equal(await page.locator('soda-spaces').count(), 0, 'late completion mounted after restoration');
      await page.getByRole('button', {name: 'Retry connection'}).click();
      await page.locator('soda-spaces').waitFor();
    }
    await page.waitForLoadState('networkidle');
    if (mode === 'valid' || mode === 'restored' || mode === 'retired') assert.equal(await page.locator('#soda-native-content > soda-spaces').count(), 1);
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

test('Tailnet entry uses the shared fixed one-attempt OAuth return, not focus reconnect', {skip: process.env.SODA_LIT_BROWSER !== '1'}, async t => {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const page = await browser.newPage(); let logins = 0;
  await page.route(origin + '/**', async route => {
    const url = new URL(route.request().url());
    if (url.pathname === '/') return route.fulfill({contentType: 'text/html', body: shell.replace('data-view="spaces"', 'data-view="tailnet"') + '<script type="module" src="/assets/soda/forgejo/soda-native-page.js"></script>'});
    if (url.pathname === '/-/soda/api/session') return route.fulfill({status: 401, json: {error: {code: 'unauthorized'}}});
    if (url.pathname === '/-/soda/login') {
      logins++; assert.deepEqual([...url.searchParams], [['destination', 'tailnet'], ['expected_user_id', '1']]);
      return route.fulfill({contentType: 'text/html', body: 'Consent fixture'});
    }
    const source = files['public' + url.pathname]; assert(source && source.startsWith('@build/forgejo-js/'));
    return route.fulfill({contentType: 'text/javascript', body: await Bun.file(path.resolve('.artifacts/forgejo-js', path.basename(source))).text()});
  });
  await page.goto(origin + '/?soda-view=tailnet'); await page.waitForURL('**/-/soda/login?**'); assert.equal(logins, 1);
  await page.goto(origin + '/?soda-view=tailnet'); await page.getByRole('button', {name: 'Retry connection'}).waitFor();
  await page.evaluate(() => {window.dispatchEvent(new Event('focus')); document.dispatchEvent(new Event('visibilitychange'));});
  assert.equal(logins, 1); assert.equal(await page.locator('soda-tailnet').count(), 0);
  await page.getByRole('button', {name: 'Retry connection'}).click(); await page.waitForURL('**/-/soda/login?**'); assert.equal(logins, 2);
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
