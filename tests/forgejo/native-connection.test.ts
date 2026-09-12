import assert from 'node:assert/strict';
import {rename, writeFile} from 'node:fs/promises';
import {dirname, join} from 'node:path';
import {test} from 'node:test';
import {chromium} from 'playwright';
import type {APIResponse, Page} from 'playwright';
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
  const errors: string[] = []; page.on('pageerror', error => errors.push(error.message));
  const saved = await Bun.file('.local/screenshot-fixture/create-output.txt').text();
  const password = saved.match(/generated random password is '([^']+)'/)?.[1]; assert(password);
  let grants = 0; let logouts = 0;
  page.on('request', request => {
    const pathname = new URL(request.url()).pathname;
    if (request.method() === 'POST' && pathname === '/login/oauth/grant') grants++;
    if (request.method() === 'POST' && pathname === '/user/logout') logouts++;
  });
  const fillNativeLogin = async (loginPage: Page) => {
    try {
      await loginPage.locator('input[name=user_name]').fill('soda-screenshot');
      await loginPage.locator('input[name=password]').fill(password);
    } catch {throw Error('Could not fill fixture native login');}
    await loginPage.locator('form[action="/user/login"] button.ui.primary').click();
  };
  const login = async () => {
    await page.goto(origin + '/-/soda/spaces');
    await fillNativeLogin(page);
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
    const child = Bun.spawn(['bun', 'test', '--timeout', '90000', 'tests/frontend/spaces-page.test.ts', 'tests/frontend/runners.test.ts', 'tests/frontend/tailnet.test.ts', 'tests/frontend/repository-settings.test.ts'], {
      env: {...process.env, SODA_TAILNET_COMPONENT: '0', SODA_PAGE_ORIGIN: origin, SODA_PAGE_ACTOR: actor, SODA_PAGE_REPOSITORY: JSON.stringify(repositoryData.repository)}, stdout: 'inherit', stderr: 'inherit',
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
  // No interception: retain the browser's real cache and native history behavior.
  const assetPhase = process.env.SODA_CONNECTION_ASSET_PHASE;
  let graphPage = page;
  if (assetPhase) {
    const cacheContext = await browser.newContext({storageState: await context.storageState(), ignoreHTTPSErrors: true});
    graphPage = await cacheContext.newPage();
    graphPage.on('pageerror', error => errors.push(error.message));
    const neutral = await graphPage.goto(origin + '/robots.txt'); assert(neutral);
    assert.equal(neutral.status(), 200);
    assert.match(neutral.headers()['content-type'] || '', /text\/plain/);
    assert.equal(await graphPage.locator('script').count(), 0, 'Cache client must not execute candidate modules before the transition');
    const hashes: unknown = JSON.parse(process.env.SODA_CONNECTION_OLD_HASHES || 'null');
    assert(hashes && typeof hashes === 'object' && !Array.isArray(hashes));
    const entries = Object.entries(hashes).map(([url, hash]) => {
      assert(url.startsWith('/assets/') && typeof hash === 'string' && /^[a-f0-9]{64}$/.test(hash));
      return [url, hash] as const;
    });
    assert.equal(entries.length, 3);
    // Only this fixture's public file responder changes; no HTML/API substitution,
    // service cutover or claim that an already-open predecessor owner is upgraded.
    await writeFile(assetPhase, 'predecessor', {mode: 0o600});
    const observed = await graphPage.evaluate(async entries => {
      const rows = [];
      for (const [url, expected] of entries) {
        for (let attempt = 0; attempt < 2; attempt++) {
          const response = await fetch(url);
          if (!response.ok || response.headers.get('cache-control') !== 'private, max-age=0, must-revalidate') throw Error('Predecessor asset response drifted');
          const bytes = await response.arrayBuffer();
          const hash = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', bytes)), byte => byte.toString(16).padStart(2, '0')).join('');
          if (hash !== expected) throw Error('Browser did not receive the bound predecessor bytes');
          if (!response.headers.get('last-modified')) throw Error('Predecessor cache validator missing');
          await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
          const timing = performance.getEntriesByName(new URL(url, location.href).href).at(-1);
          if (!(timing instanceof PerformanceResourceTiming) || timing.transferSize <= 0) throw Error('Zero-age request did not contact the responder');
          rows.push({url, hash, transferred: timing.transferSize});
        }
      }
      return rows;
    }, entries);
    assert.equal(observed.length, 6);
    await writeFile(assetPhase, 'candidate', {mode: 0o600});
    const currentHash = await graphPage.evaluate(async () => {
      const response = await fetch('/assets/sodaspaces-api.js');
      if (!response.ok || response.headers.get('cache-control') !== 'private, max-age=0, must-revalidate') throw Error('Candidate revalidation failed');
      return Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', await response.arrayBuffer())), byte => byte.toString(16).padStart(2, '0')).join('');
    });
    const priorAPI = entries.find(([url]) => url === '/assets/sodaspaces-api.js'); assert(priorAPI);
    assert.notEqual(currentHash, priorAPI[1], 'Unversioned request retained predecessor bytes');
    const candidateAPI = await Bun.file(join(dirname(assetPhase), 'public/assets/sodaspaces-api.js')).arrayBuffer();
    assert.equal(currentHash, new Bun.CryptoHasher('sha256').update(candidateAPI).digest('hex'), 'Browser candidate bytes did not match the emitted payload');
    t.diagnostic('Bound predecessor asset hashes and zero-age revalidation passed; predecessor document retirement is separate');
  } else {
    // Candidate bytes under legacy URL identities: useful cache-hit coverage,
    // deliberately not called a predecessor/upgrade test.
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
    assert.deepEqual(cached, [0, 0, 0], 'Legacy module URLs were not actually cached');
  }
  await graphPage.goto(origin + '/?soda-view=runners');
  await graphPage.getByLabel('Registration token', {exact: true}).waitFor();
  const modules = await graphPage.evaluate(() => performance.getEntriesByType('resource')
    .filter(entry => entry.name.includes('/assets/') && new URL(entry.name).pathname.endsWith('.js'))
    .map(entry => ({url: entry.name, bytes: (entry as PerformanceResourceTiming).transferSize})));
  for (const name of ['soda-native-page.js', 'soda-settings-link.js', 'soda-connection.js', 'soda-runners-page.js', 'sodaspaces-api.js', 'lit.js']) {
    const loaded = modules.find(entry => new URL(entry.url).pathname.endsWith('/' + name));
    assert(loaded, `Missing real module ${name}`);
    assert.equal(new URL(loaded.url).search, `?v=${presentationVersion}`, name);
  }
  if (graphPage !== page) {
    await graphPage.goBack({waitUntil: 'commit'});
    // A restored plaintext document need not emit a new load event. Back has
    // committed; check its exact URL rather than waiting for another load.
    assert(graphPage.url() === origin + '/robots.txt', 'Back did not reach the original neutral document');
    await graphPage.goForward({waitUntil: 'commit'});
    await graphPage.getByLabel('Registration token', {exact: true}).waitFor();
    assert.equal(await graphPage.locator('soda-runners').count(), 1);
    await graphPage.context().close();
    await page.goto(origin + '/?soda-view=runners');
    await page.getByLabel('Registration token', {exact: true}).waitFor();
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
  for (const view of ['spaces', 'repository-spaces&repository_id=1']) {
    await page.goto(origin + '/?soda-view=' + view);
    const refresh = page.getByRole('button', {name: view === 'spaces' ? 'Refresh Spaces' : 'Refresh status', exact: true});
    const ready = page.locator(view === 'spaces' ? '#sodaspaces-data[aria-busy=false]' : '[data-project-controls][aria-busy=false]');
    await ready.waitFor();
    await page.evaluate(view => {
      document.documentElement.dataset.historyDocument = view;
      window.addEventListener('pageshow', event => {document.documentElement.dataset.historyPersisted = String(event.persisted);});
    }, view);
    await page.locator('#navbar a[href="/issues"]').click(); await page.waitForURL('**/issues');
    await page.goBack({waitUntil: 'commit'});
    await page.waitForFunction(() => document.documentElement.dataset.historyPersisted === 'true');
    await ready.waitFor();
    if (view === 'spaces') await page.getByLabel('Workspace options', {exact: true}).click();
    await refresh.waitFor();
    assert.equal(await page.locator('html').getAttribute('data-history-document'), view);
    assert.equal(await page.locator('html').getAttribute('data-history-persisted'), 'true');
    assert(await refresh.isEnabled(), view + ' restored permanently stale');
    await refresh.click();
    await page.waitForFunction(() => !document.querySelector('[aria-busy="true"]'));
    assert.equal(await page.locator('#soda-native-content > soda-spaces, #soda-native-content > soda-project-controls').count(), 1);
    assert.equal(await page.getByRole('button', {name: 'Retry connection'}).count(), 0);
  }
  // Native Forgejo owns this form and its unsaved data. Entering a Soda view
  // must neither submit it nor replace the browser's history restoration.
  const nativeProfileWrites: string[]=[];
  const profileRequest=(request: import('playwright').Request)=>{
    if(request.method() === 'POST' && new URL(request.url()).pathname === '/user/settings') nativeProfileWrites.push('profile');
  };
  page.on('request',profileRequest);
  const leaveDraft=async (dialog: import('playwright').Dialog)=>{
    assert.equal(dialog.type(),'beforeunload'); await dialog.accept();
  };
  page.on('dialog',leaveDraft);
  try {
    await page.goto(origin+'/user/settings');
    const fullName=page.locator('form[action="/user/settings"] input[name="full_name"]');
    await fullName.fill('Unsaved local fixture draft');
    await page.evaluate(()=>{
      document.documentElement.dataset.historyDocument='native-profile-draft';
      window.addEventListener('pageshow',event=>{document.documentElement.dataset.historyPersisted=String(event.persisted);});
    });
    const beforeDraftGrants=grants;
    await page.goto(origin+'/?soda-view=spaces');
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    await page.goBack({waitUntil:'commit'});
    await fullName.waitFor();
    assert.equal(await fullName.inputValue(),'Unsaved local fixture draft','Native draft lost across Soda navigation');
    assert.equal(await page.locator('html').getAttribute('data-history-document'),'native-profile-draft');
    assert.equal(await page.locator('html').getAttribute('data-history-persisted'),'true');
    await page.goForward({waitUntil:'commit'});
    await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    assert.equal(await page.locator('#soda-native-content > soda-spaces').count(),1);
    assert.equal(await page.getByRole('button',{name:'Retry connection'}).count(),0);
    assert.equal(grants,beforeDraftGrants,'History unexpectedly requested fresh consent');
    assert.deepEqual(nativeProfileWrites,[],'Draft navigation submitted the native profile');
  } finally {
    page.off('request',profileRequest); page.off('dialog',leaveDraft);
  }
  assert.deepEqual(errors, [], 'Native entry/history raised a browser error');
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

  const backendPhase = process.env.SODA_CONNECTION_BACKEND_PHASE;
  if (backendPhase) {
    // Run after every mandatory consumer and the normal native parent. Only this
    // new context uses the genuine old executable/document and fresh old DB.
    await context.close();
    const setBackendPhase = async (phase: 'predecessor' | 'candidate') => {
      // The old owner's read-only timer can run concurrently with the switch.
      await writeFile(backendPhase + '.next', phase, {mode: 0o600});
      await rename(backendPhase + '.next', backendPhase);
    };
    await setBackendPhase('predecessor');
    const oldContext = await browser.newContext({ignoreHTTPSErrors: true});
    const oldPage = await oldContext.newPage();
    const oldErrors: string[] = [];
    oldPage.on('pageerror', error => oldErrors.push(error.message));
    const entryResponse = oldPage.waitForResponse(response => new URL(response.url()).pathname === '/assets/sodaspaces-page.js').catch(() => null);
    await oldPage.goto(origin + '/-/soda/spaces');
    await oldPage.getByRole('link', {name: 'Connect to Soda'}).click();
    await fillNativeLogin(oldPage);
    await oldPage.waitForURL(url => url.pathname === '/login/oauth/authorize' || url.pathname === '/-/soda/spaces');
    if (new URL(oldPage.url()).pathname === '/login/oauth/authorize') await oldPage.locator('#authorize-app').click();
    await oldPage.locator('#spaces-page > soda-spaces').waitFor();
    await oldPage.locator('summary[aria-label="Workspace options"]').click();
    const refresh = oldPage.getByRole('button', {name: 'Refresh Spaces', exact: true});
    await refresh.waitFor();
    await oldPage.waitForFunction(() => document.querySelector('#sodaspaces-data')?.getAttribute('aria-busy') === 'false');
    const hashes: unknown = JSON.parse(process.env.SODA_CONNECTION_OLD_HASHES || 'null');
    assert(hashes && typeof hashes === 'object' && '/assets/sodaspaces-page.js' in hashes);
    const entry = await entryResponse; assert(entry, 'Predecessor entry response unavailable');
    const loadedHash = new Bun.CryptoHasher('sha256').update(await entry.body()).digest('hex');
    assert.equal(loadedHash, hashes['/assets/sodaspaces-page.js']);
    const actor = await oldPage.locator('#spaces-page').getAttribute('data-soda-actor'); assert(actor);
    const mutations: string[] = [];
    oldPage.on('request', request => {
      const path = new URL(request.url()).pathname;
      if (path.startsWith('/-/soda/api/') && !['GET', 'HEAD', 'OPTIONS'].includes(request.method())) mutations.push(request.method() + ' ' + path);
    });
    // Keep the old document/JS realm open while its backend stops and the exact
    // same fresh v6 DB is opened by current handlers. Refresh is the OLD control.
    await setBackendPhase('candidate');
    await refresh.click();
    await oldPage.waitForFunction(() => document.querySelector('#sodaspaces-data')?.getAttribute('aria-busy') === 'false');
    assert.equal(await oldPage.locator('#spaces-page > soda-spaces').count(), 1);
    assert.equal((await oldPage.request.get(origin + '/-/soda/api/session', {headers: {'X-Soda-Expected-User-ID': actor}})).status(), 200);
    assert.deepEqual(mutations, [], 'Old document replayed a mutation during backend transition');
    await oldPage.evaluate(() => {
      const owner = document.querySelector('#spaces-page > soda-spaces');
      window.addEventListener('pagehide', () => {
        // Registered after the real old owner's listener; observe its synchronous
        // retirement, rather than dispatching a synthetic history event.
        sessionStorage.setItem('soda-fixture-predecessor-retired', owner && 'stale' in owner && owner.stale === true ? 'yes' : 'no');
      }, {once: true});
    });
    await oldPage.goto(origin + '/robots.txt');
    await oldPage.goBack();
    assert.equal(await oldPage.evaluate(() => sessionStorage.getItem('soda-fixture-predecessor-retired')), 'yes');
    if (await oldPage.locator('#spaces-page > soda-spaces').count()) {
      await oldPage.getByText('Page or Soda identity changed. Reload; no action was replayed.', {exact: true}).waitFor();
      assert.equal(await refresh.isDisabled(), true);
      t.diagnostic('Predecessor Back restored a retired owner from BFCache');
    } else {
      // Preserve the genuine old response/cache policy. Do not force BFCache by
      // rewriting headers or call a network reload a restored predecessor realm.
      await oldPage.locator('#soda-native-content > soda-spaces').waitFor();
      assert.equal(oldPage.url(), origin + '/?soda-view=spaces');
      assert.equal(await oldPage.locator('soda-spaces').count(), 1);
      t.diagnostic('Predecessor Back performed a network reload into the current owner; old pagehide retirement was observed separately, not BFCache restoration');
    }
    assert.deepEqual(mutations, [], 'History replayed an old mutation');
    await oldPage.goto(origin + '/?soda-view=runners');
    await oldPage.locator('#soda-native-content > soda-runners').waitFor();
    assert.equal(await oldPage.locator('soda-spaces').count(), 0);
    assert.deepEqual(oldErrors, [], 'Predecessor execution raised browser errors');
    await oldContext.close();
    t.diagnostic('Genuine predecessor executable/document, same-DB transition, observed old-owner retirement and explicit current-page entry passed; see actual history mode above; no native project operations');
  }
});
