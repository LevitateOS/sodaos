import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';
import type {APIResponse} from 'playwright';

test('real Forgejo consent, reuse, repeat connection and both partial logout outcomes', {
  skip: !process.env.SODA_CONNECTION_ORIGIN, timeout: 120000,
}, async t => {
  const origin = process.env.SODA_CONNECTION_ORIGIN; assert(origin);
  const address = new URL(origin);
  assert(address.origin === origin && address.protocol === 'https:' && address.hostname === '127.0.0.1', 'fixture credentials are restricted to the local TLS test server');
  const browser = await chromium.launch({channel: 'chrome', headless: true, chromiumSandbox: true});
  t.after(() => browser.close());
  // Trust is scoped to this test browser's loopback TLS server, not the host.
  const context = await browser.newContext({ignoreHTTPSErrors: true});
  const page = await context.newPage();
  const saved = await Bun.file('.local/screenshot-fixture/create-output.txt').text();
  const password = saved.match(/generated random password is '([^']+)'/)?.[1]; assert(password);
  let grants = 0; let logouts = 0;
  page.on('request', request => {
    const pathname = new URL(request.url()).pathname;
    if (request.method() === 'POST' && pathname === '/login/oauth/grant') grants++;
    if (request.method() === 'POST' && pathname === '/user/logout') logouts++;
  });
  const login = async () => {
    await page.goto(origin + '/-/soda/spaces');
    try {
      await page.locator('input[name=user_name]').fill('soda-screenshot');
      await page.locator('input[name=password]').fill(password);
    } catch {throw Error('Could not fill fixture native login');}
    await page.locator('form[action="/user/login"] button.ui.primary').click();
  };
  const menuLogout = async () => {
    const menu = page.locator('#navbar details').filter({has: page.locator('a[data-url="/user/logout"]')});
    await menu.locator('summary').click();
    await menu.locator('a[data-url="/user/logout"]').focus();
    await page.keyboard.press('Enter');
  };
  await login();
  await page.locator('#authorize-app').waitFor();
  await page.locator('button[name=granted][value=false]').click();
  await page.getByRole('button', {name: 'Retry connection'}).waitFor();
  assert.match(page.url(), /soda-connect=failed/);
  await page.getByRole('button', {name: 'Retry connection'}).click();
  await page.locator('#authorize-app').click();
  await page.locator('#soda-native-content > soda-spaces').waitFor();
  assert.equal(page.url(), origin + '/?soda-view=spaces');
  const cookie = (await context.cookies()).find(c => c.name === '__Secure-sodaspaces-session'); assert(cookie);
  for (const view of ['runners', 'repository-spaces&repository_id=1', 'spaces']) {
    await page.goto(origin + '/?soda-view=' + view);
    await page.locator('#soda-native-content > ' + (view === 'runners' ? 'soda-runners' : view.startsWith('repository-spaces') ? 'soda-project-controls' : 'soda-spaces')).waitFor();
    if (view === 'runners') {
      await page.waitForFunction(() => {const field = document.querySelector<HTMLInputElement>('soda-runners input[name=id]'); return field && !field.matches(':disabled');});
      assert.equal(await page.getByRole('link', {name: 'Forgejo Actions administration'}).getAttribute('href'), origin + '/admin/actions/runners');
      await page.getByLabel('Registration token', {exact: true}).fill('synthetic-ui-only-token');
      await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})));
      assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
    }
    if (view.startsWith('repository-spaces')) {
      await page.getByRole('link', {name: 'Native repository settings', exact: true}).waitFor();
      const response: APIResponse = await page.request.get(origin + '/-/soda/api/environments?repository_id=1', {headers: {'X-Soda-Expected-User-ID': (await page.locator('#soda-native-content').getAttribute('data-actor')) || ''}});
      assert.equal(response.status(), 200);
      const data: unknown = await response.json();
      assert(data && typeof data === 'object' && 'repository' in data && data.repository && typeof data.repository === 'object' && 'owner' in data.repository && typeof data.repository.owner === 'string' && 'name' in data.repository && typeof data.repository.name === 'string');
      assert.equal(await page.getByRole('link', {name: 'Native repository settings', exact: true}).getAttribute('href'), origin + '/' + encodeURIComponent(data.repository.owner) + '/' + encodeURIComponent(data.repository.name) + '/settings');
    }
    for (const width of [1440, 390]) {
      await page.setViewportSize({width, height: 900});
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), view + ' overflows horizontally');
      assert.equal(await page.locator('#navbar').count(), 1);
    }
    await page.setViewportSize({width: 1440, height: 900});
    assert((await context.cookies()).find(c => c.name === cookie.name)?.value === cookie.value, 'valid session rotated on entry');
  }
  // Full-page workspace remains a single mount below the actual native header.
  for (const width of [1440, 390]) {
    await page.setViewportSize({width, height: 900});
    await page.waitForFunction(() => {
      const root = document.getElementById('soda-native-content');
      return root && Math.abs(root.getBoundingClientRect().bottom + (document.querySelector('footer')?.getBoundingClientRect().height || 0) - innerHeight) < 3;
    });
    assert.equal(await page.locator('soda-spaces').count(), 1);
    assert.equal(await page.locator('#soda-drawer').count(), 0);
    assert.equal(await page.locator('#navbar').count(), 1);
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'native host overflows horizontally');
  }
  await page.setViewportSize({width: 1440, height: 900});
  const granted = grants;
  await menuLogout();
  await page.waitForFunction(() => location.pathname === '/' && !document.getElementById('soda-settings-link'));
  assert.equal(logouts, 1);
  assert.equal((await page.request.get(origin + '/-/soda/api/session')).status(), 401);
  await login();
  await page.getByRole('button', {name: 'Retry connection'}).click();
  await page.locator('#soda-native-content > soda-spaces').waitFor();
  assert.equal(grants, granted, 'confidential repeat grant unexpectedly requested consent');

  // Explicit fault injection only on native logout; OAuth and Soda remain real.
  await page.route(origin + '/user/logout', route => route.fulfill({status: 503, body: 'Fixture native logout failure'}));
  await menuLogout();
  await page.getByRole('button', {name: 'Retry Forgejo sign-out'}).waitFor();
  assert.equal((await page.request.get(origin + '/-/soda/api/session')).status(), 401);
  assert.equal(await page.locator('#soda-settings-link').count(), 1, 'native actor remains after native failure');
  await page.unroute(origin + '/user/logout');
  await page.getByRole('button', {name: 'Retry Forgejo sign-out'}).click();
  await page.waitForFunction(() => location.pathname === '/' && !document.getElementById('soda-settings-link'));

  await login();
  await page.getByRole('button', {name: 'Retry connection'}).click();
  await page.locator('#soda-native-content > soda-spaces').waitFor();
  const beforeEscape = logouts;
  await page.route(origin + '/-/soda/api/login/cancel', route => route.request().method() === 'POST' ? route.fulfill({status: 503, body: 'Fixture Soda cancellation failure'}) : route.continue());
  await menuLogout();
  await page.getByRole('button', {name: 'Sign out of Forgejo only'}).waitFor();
  assert.equal(logouts, beforeEscape, 'native logout ran before cancellation confirmation');
  assert.equal((await page.request.get(origin + '/-/soda/api/session')).status(), 200);
  await page.getByRole('button', {name: 'Sign out of Forgejo only'}).click();
  await page.waitForFunction(() => location.pathname === '/' && !document.getElementById('soda-settings-link'));
  assert.equal((await page.request.get(origin + '/-/soda/api/session')).status(), 200, 'native-only escape must not claim Soda revocation');
});
