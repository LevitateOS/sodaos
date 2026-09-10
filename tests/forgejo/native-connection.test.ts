import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';
import type {APIResponse} from 'playwright';
import {presentationVersion} from '../../scripts/build-forgejo.ts';

test('real Forgejo consent, reuse, repeat connection and both partial logout outcomes', {
  skip: !process.env.SODA_CONNECTION_ORIGIN, timeout: 240000,
}, async t => {
  const origin = process.env.SODA_CONNECTION_ORIGIN; assert(origin);
  const address = new URL(origin);
  assert(address.origin === origin && address.protocol === 'https:' && address.hostname === '127.0.0.1', 'fixture credentials are restricted to the local TLS test server');
  const spki = process.env.SODA_CONNECTION_SPKI; assert(spki && /^[A-Za-z0-9+/]{43}=$/.test(spki));
  const browser = await chromium.launch({channel: 'chrome', headless: true, chromiumSandbox: true, ignoreDefaultArgs: ['--disable-back-forward-cache'], args: [`--ignore-certificate-errors-spki-list=${spki}`]});
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
  if (process.env.SODA_PAGE_CONSUMERS === '1') {
    const state = process.env.SODA_PAGE_STATE; assert(state);
    await context.storageState({path: state});
    const actor = await page.locator('#soda-native-content').getAttribute('data-actor'); assert(actor);
    const repositoryResponse: APIResponse = await page.request.get(origin + '/-/soda/api/environments?repository_id=1', {headers: {'X-Soda-Expected-User-ID': actor}});
    assert.equal(repositoryResponse.status(), 200);
    const repositoryData: unknown = await repositoryResponse.json();
    assert(repositoryData && typeof repositoryData === 'object' && 'repository' in repositoryData);
    const child = Bun.spawn(['bun', 'test', '--timeout', '90000', 'tests/frontend/spaces-page.test.ts', 'tests/frontend/runners.test.ts', 'tests/frontend/repository-settings.test.ts'], {
      env: {...process.env, SODA_PAGE_ORIGIN: origin, SODA_PAGE_ACTOR: actor, SODA_PAGE_REPOSITORY: JSON.stringify(repositoryData.repository)}, stdout: 'inherit', stderr: 'inherit',
    });
    assert.equal(await child.exited, 0, 'Native page consumers failed');
  }
  const cookie = (await context.cookies()).find(c => c.name === '__Secure-sodaspaces-session'); assert(cookie);
  for (const view of ['runners', 'repository-spaces&repository_id=1', 'spaces']) {
    const legacy = view === 'runners' ? '/settings/runners' : view.startsWith('repository-spaces') ? '/repositories/1/settings/spaces' : '/spaces';
    await page.goto(origin + '/-/soda' + legacy);
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
    assert.equal(page.url(), origin + '/?soda-view=' + view, 'bookmark did not reach fixed native host');
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
  // Real HTTP cache remains enabled in this context (no route interception yet).
  // Warm the predecessor's entry/import identities, then require the new native
  // document to load only the current epoch throughout its executed Soda graph.
  const cached = await page.evaluate(async () => {
    const urls = ['/assets/soda/forgejo/soda-native-page.js?v=1', '/assets/soda-connection.js', '/assets/sodaspaces-api.js'];
    for (const url of urls) {
      for (let attempt = 0; attempt < 2; attempt++) {
        const response = await fetch(url); if (!response.ok) throw Error('Cache warm failed');
        if (response.headers.get('cache-control') !== 'private, max-age=21600') throw Error('Fixture asset cache policy drifted');
        await response.text();
      }
    }
    await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
    return urls.map(url => (performance.getEntriesByName(new URL(url, location.href).href).at(-1) as PerformanceResourceTiming).transferSize);
  });
  assert.deepEqual(cached, [0, 0, 0], 'Predecessor module URLs were not actually cached');
  await page.goto(origin + '/?soda-view=runners');
  await page.getByLabel('Registration token', {exact: true}).waitFor();
  const modules = await page.evaluate(() => performance.getEntriesByType('resource')
    .filter(entry => entry.name.includes('/assets/') && new URL(entry.name).pathname.endsWith('.js'))
    .map(entry => ({url: entry.name, bytes: (entry as PerformanceResourceTiming).transferSize})));
  for (const name of ['soda-native-page.js', 'soda-settings-link.js', 'soda-connection.js', 'soda-runners-page.js', 'sodaspaces-api.js', 'lit.js']) {
    const loaded = modules.find(entry => new URL(entry.url).pathname.endsWith('/' + name));
    assert(loaded, `Missing real module ${name}`);
    assert.equal(new URL(loaded.url).search, `?v=${presentationVersion}`, name);
  }
  // No request routing here: Playwright interception disables BFCache in its
  // delegate. The synthetic-operation consumers separately cover late replies.
  // Keep a real second native tab attached to Forgejo's notification worker;
  // Chromium otherwise evicts the last client's document with
  // SharedWorkerWithNoActiveClient. Do not replace the upstream worker.
  const peer = await context.newPage();
  await peer.goto(origin + '/issues');
  await page.bringToFront();
  await page.getByLabel('Registration token', {exact: true}).fill('unsent-history-fixture');
  const duplicate = await page.evaluate(async () => {
    const original = document.querySelector('soda-runners');
    const entry = document.querySelector<HTMLScriptElement>('script[src*="soda-native-page.js"]');
    if (!entry) throw Error('Missing native entry');
    await import(entry.src + '&duplicate-entry-probe=1');
    return original === document.querySelector('soda-runners');
  });
  assert(duplicate, 'Duplicate module evaluation replaced the live owner');
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), 'unsent-history-fixture');
  const historyCDP = await context.newCDPSession(page);
  await historyCDP.send('Page.enable');
  historyCDP.on('Page.backForwardCacheNotUsed', event => t.diagnostic(JSON.stringify(event.notRestoredExplanations)));
  await page.evaluate(() => {
    document.documentElement.dataset.historyDocument = 'runner-before-navigation';
    window.addEventListener('pageshow', event => {document.documentElement.dataset.historyPersisted = String(event.persisted);});
  });
  await page.locator('#navbar a[href="/issues"]').click();
  await page.waitForURL('**/issues');
  await page.goBack({waitUntil: 'commit'});
  await page.getByLabel('Registration token', {exact: true}).waitFor();
  const history = await page.evaluate(() => ({document: document.documentElement.dataset.historyDocument, persisted: document.documentElement.dataset.historyPersisted}));
  assert.equal(history.document, 'runner-before-navigation', 'Back replaced the document rather than restoring BFCache');
  assert.equal(history.persisted, 'true', 'A real persisted pageshow is required');
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(await page.locator('soda-runners').count(), 1);
  await peer.close();
  await page.goto(origin + '/?soda-view=spaces');
  await page.locator('#soda-native-content > soda-spaces').waitFor();
  assert.equal(await page.locator('soda-spaces').count(), 1);
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
