import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import type {TestContext} from 'node:test';
import {chromium} from 'playwright';
import type {Page} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response';
import type {Runner} from '../../frontend/runners/soda-runner-types';

const root = path.resolve(import.meta.dirname, '../..');
const files: Record<string, string> = payload;
const origin = 'https://forgejo.example.test';
const browserCase = {skip: !process.env.SODA_RUNNERS_PAGE_HTML};
const exampleRunner = (provider = 'forgejo'): Runner => ({
  id: 'one', provider, registration_url: provider === 'forgejo' ? origin : 'https://github.com/team/repo',
  account: 'soda-runner-one', architecture: 'x86-64', version: 'fixture', capacity: 1,
  service: {load: 'loaded', active: 'active', sub: 'running', enabled: 'enabled'},
});

async function runnersPage(t: TestContext, mode = 'html') {
  const fixture: unknown = await Bun.file(process.env.SODA_RUNNERS_PAGE_HTML || '').json();
  assert(fixture && typeof fixture === 'object' && 'html' in fixture && typeof fixture.html === 'string' && 'csp' in fixture && typeof fixture.csp === 'string');
  const pages = new Map(Object.entries(fixture));
  const csp = fixture.csp;
  const browser = await chromium.launch({headless: true, chromiumSandbox: true, ignoreDefaultArgs: ['--disable-back-forward-cache']});
  t.after(() => browser.close());
  const page = await browser.newPage();
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const state = {runners: [] as Runner[], failList: false, mutationStatus: 200, sessionStatus: 200, actor: '1', operator: true, logoutStatus: 204, logoutRequests: 0, sessionReads: 0, page: mode};
  const mutations: {path: string; body: Record<string, unknown>}[] = [];
  await page.route(origin + '/**', async route => {
    const request = route.request(), pathname = new URL(request.url()).pathname;
    if (pathname === '/-/soda/settings/runners') {
      const html = pages.get(state.page); assert.equal(typeof html, 'string');
      await route.fulfill({body: String(html), contentType: 'text/html', status: state.page === 'denied' ? 403 : 200, headers: {'Content-Security-Policy': csp, 'Cache-Control': 'private, no-store'}});
      return;
    }
    if (pathname === '/-/soda/api/session') {
      state.sessionReads++;
      assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      await route.fulfill({status: state.sessionStatus, json: {user: {id: state.actor, login: 'alice'}, csrf_token: 'csrf-alice', soda_operator: state.operator, forgejo_url: origin}});
      return;
    }
    if (pathname === '/-/soda/api/session/logout') {
      state.logoutRequests++;
      assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
      if (state.logoutStatus === 204) state.page = 'anonymous';
      await route.fulfill({status: state.logoutStatus});
      return;
    }
    if (pathname.startsWith('/-/soda/api/settings/runners')) {
      assert.equal(request.headers()['x-soda-expected-user-id'], '1');
      if (request.method() === 'POST') {
        assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
        const body: unknown = request.postDataJSON();
        assert(body && typeof body === 'object' && !Array.isArray(body));
        mutations.push({path: pathname, body: {...body}});
        if (state.mutationStatus === 200) state.runners = pathname.endsWith('/remove') ? [] : [exampleRunner('provider' in body && typeof body.provider === 'string' ? body.provider : 'forgejo')];
        await route.fulfill({status: state.mutationStatus, json: state.mutationStatus === 200 ? {ok: true} : {error: {message: 'must-not-render-native-secret'}}});
        return;
      }
      await route.fulfill({status: state.failList ? 503 : 200, json: {forgejo_url: origin, runners: state.runners, runner_count: state.runners.length, active_listeners: state.runners.length, total_capacity: state.runners.length}});
      return;
    }
    const source = files['public' + pathname];
    if (source) {
      const file = source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : path.join(root, source);
      const contentType = pathname.endsWith('.js') ? 'text/javascript' : pathname.endsWith('.css') ? 'text/css' : pathname.endsWith('.svg') ? 'image/svg+xml' : pathname.endsWith('.woff2') ? 'font/woff2' : 'image/png';
      await route.fulfill({body: Buffer.from(await Bun.file(file).arrayBuffer()), contentType});
      return;
    }
    // Native destinations are owned by the provider, never runner API calls.
    await route.fulfill({contentType: 'text/html', body: '<!doctype html><title>Native destination</title><p>Native provider page</p>'});
  });
  await page.goto(origin + '/-/soda/settings/runners');
  if (mode === 'html') await page.getByText('No local runners registered.').waitFor();
  t.after(() => assert.deepEqual(errors, []));
  return {page, state, mutations};
}
async function registerDraft(page: Page) {
  await page.getByLabel('Local runner ID', {exact: true}).fill('one');
  await page.getByLabel('Forgejo runner UUID').fill('33834eef-e758-48c4-a676-1745426747aa');
  await page.getByLabel('Labels', {exact: true}).fill('native:host');
  await page.getByLabel('Registration token', {exact: true}).fill('synthetic-secret-never-store');
}
async function settled(page: Page) {
  await page.waitForFunction(() => !document.querySelector<HTMLButtonElement>('button')?.disabled);
}
async function hide(page: Page) {
  await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})));
}
async function restore(page: Page) {
  await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true})));
  await settled(page);
}

// Deliberately ignore abort for one response. A browser/transport abort alone must
// not be the implementation's guard against publishing or dispatching old work.
async function pauseNextFetch(page: Page, kind: 'session' | 'list' | 'mutation') {
  await page.evaluate(kind => {
    const original = window.fetch;
    let armed = true;
    window.fetch = Object.assign(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      const matches = kind === 'session' ? url.endsWith('/api/session') : url.includes('/api/settings/runners') && (kind === 'mutation' ? init?.method === 'POST' : init?.method === 'GET');
      if (!armed || !matches) return original(input, init);
      armed = false;
      document.documentElement.dataset.pausedRunnerFetch = kind;
      await new Promise<void>(resolve => window.addEventListener('release-runner-fetch', () => resolve(), {once: true}));
      const json = kind === 'session'
        ? {user: {id: '1', login: 'alice'}, csrf_token: 'old-csrf', soda_operator: true, forgejo_url: location.origin}
        : kind === 'mutation' ? {ok: true} : {forgejo_url: location.origin, runners: [], runner_count: 0, active_listeners: 0, total_capacity: 0};
      return new Response(JSON.stringify(json), {headers: {'Content-Type': 'application/json'}});
    }, original);
  }, kind);
}
async function releaseFetch(page: Page) {
  await page.evaluate(() => window.dispatchEvent(new Event('release-runner-fetch')));
  // Let chained async continuations and Lit updates settle without a wall-clock sleep.
  await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
}

test('runner response shares Cockpit one-slot validation', () => {
  assert.throws(() => decodeRunnerResponse('list', {runners: [], runner_count: 1, active_listeners: 0, total_capacity: 0}));
  assert.throws(() => decodeRunnerResponse('remove', {ok: false}));
  assert.throws(() => decodeRunnerResponse('list', {runners: [], runner_count: 0, active_listeners: -1, total_capacity: 0}));
  assert.equal(decodeRunnerResponse('start', {ok: true}).ok, true);
});

test('Go HTML and emitted Lit register both providers and confirm exact lifecycle targets', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  assert.equal(mutations.length, 0);
  await registerDraft(page);
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  await page.getByRole('button', {name: 'start one', exact: true}).waitFor();
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(mutations[0]?.body.registration_token, 'synthetic-secret-never-store');
  assert.equal(mutations[0]?.body.registration_url, '');
  assert(!await page.evaluate(() => document.documentElement.outerHTML.includes('synthetic-secret-never-store') || JSON.stringify(localStorage).includes('synthetic-secret-never-store') || JSON.stringify(sessionStorage).includes('synthetic-secret-never-store')));
  for (const action of ['start', 'stop', 'restart', 'remove']) {
    await page.getByRole('button', {name: `${action} one`, exact: true}).click();
    await page.getByLabel('Exact runner ID').fill('wrong');
    await page.getByRole('button', {name: `Confirm ${action}`, exact: true}).click();
    await page.getByText('Type the exact runner ID to confirm.').waitFor();
    await page.getByLabel('Exact runner ID').fill('one');
    await page.getByRole('button', {name: `Confirm ${action}`, exact: true}).click();
    await settled(page);
  }
  assert.equal(mutations.length, 5);
  await page.getByLabel('Provider', {exact: true}).selectOption('github');
  await page.getByLabel('GitHub registration URL').fill('https://github.com/team/repo');
  await page.getByLabel('Labels', {exact: true}).fill('soda-linux');
  await page.getByLabel('Registration token', {exact: true}).fill('github-synthetic-token');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  await settled(page);
  assert.equal(mutations[5]?.body.provider, 'github');
  assert.equal(mutations[5]?.body.registration_id, '');
  assert.equal(mutations[5]?.body.registration_token, 'github-synthetic-token');
  assert(state.sessionReads >= 12);
});

test('provider switch scrubs credentials and provider drafts, with safe native links', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await registerDraft(page);
  await page.getByLabel('Provider', {exact: true}).selectOption('github');
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  await page.getByLabel('GitHub registration URL').fill('https://github.com/team/repo');
  await page.getByLabel('Registration token', {exact: true}).fill('github-only');
  await page.getByLabel('Provider', {exact: true}).selectOption('forgejo');
  assert.equal(await page.getByLabel('Forgejo runner UUID').inputValue(), '');
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  await page.getByLabel('Provider', {exact: true}).selectOption('github');
  assert.equal(await page.getByLabel('GitHub registration URL').inputValue(), '');
  assert.equal(await page.getByLabel('Local runner ID', {exact: true}).inputValue(), 'one');
  assert.equal(await page.getByRole('link', {name: 'Forgejo Actions administration', exact: true}).getAttribute('href'), origin + '/admin/actions/runners');
  for (const address of ['https://github.com/team/repo', 'javascript:alert(1)', 'https://github.com.attacker.test/repo', 'https://user:pass@github.com/repo', 'https://github.com:443/repo', 'https://github.com/repo?', 'https://github.com/repo#', 'https://github.com/']) {
    state.runners = [{...exampleRunner('github'), registration_url: address}];
    await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
    const link = page.getByRole('link', {name: 'Open one in GitHub', exact: true});
    assert.equal(await link.count(), address === 'https://github.com/team/repo' ? 1 : 0);
  }
  state.runners = [{...exampleRunner(), registration_url: 'http://internal-only:3000'}];
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  assert.equal(await page.getByRole('link', {name: 'Open one in Forgejo', exact: true}).getAttribute('href'), origin);
  assert(!await page.locator('body').innerText().then(text => text.includes('internal-only')));
  assert.equal(mutations.length, 0);
});

test('registration guidance rejects unsafe fields and changed actors before any mutation', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await registerDraft(page);
  await page.getByLabel('Labels', {exact: true}).fill('soda-linux');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  assert.match(await page.getByLabel('Labels', {exact: true}).evaluate(node => node instanceof HTMLInputElement ? node.validationMessage : ''), /name:host/);
  await page.getByLabel('Provider', {exact: true}).selectOption('github');
  await page.getByLabel('GitHub registration URL').fill('https://github.com.attacker.test/repo');
  await page.getByLabel('Registration token', {exact: true}).fill('github-synthetic-token');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  assert.match(await page.getByLabel('GitHub registration URL').evaluate(node => node instanceof HTMLInputElement ? node.validationMessage : ''), /HTTPS github.com/);
  assert.equal(mutations.length, 0);
  await page.getByLabel('GitHub registration URL').fill('https://github.com/team/repo');
  state.actor = '2';
  await page.getByRole('button', {name: 'Register and start listener'}).click(); await settled(page);
  await page.getByText('Reconnect explicitly').waitFor();
  assert.equal(mutations.length, 0);
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
});

test('pagehide scrubs tokens immediately and restoration requires fresh authority', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [exampleRunner()];
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  await registerDraft(page);
  await page.getByRole('button', {name: 'remove one', exact: true}).click();
  await hide(page);
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(await page.getByRole('button', {name: 'Register and start listener'}).isDisabled(), true);
  assert.equal(await page.getByRole('region', {name: 'Confirm runner operation'}).count(), 0);
  state.sessionStatus = 401;
  await restore(page);
  await page.getByText('Reconnect explicitly').waitFor();
  assert.equal(await page.getByRole('region', {name: 'Local runner inventory'}).count(), 0);
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(mutations.length, 0);
});

test('retired session continuations cannot dispatch runner or logout requests after reconnect', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await registerDraft(page);
  await pauseNextFetch(page, 'session');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  await page.waitForFunction(() => document.documentElement.dataset.pausedRunnerFetch === 'session');
  await page.evaluate(() => {
    const element = document.querySelector('soda-runners'); assertElement(element);
    const parent = element.parentElement; assertElement(parent);
    element.remove(); parent.append(element);
    function assertElement(value: Element | null): asserts value is Element {if (!value) throw Error('Missing fixture element');}
  });
  await settled(page);
  await releaseFetch(page);
  assert.equal(mutations.length, 0);
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.match(await page.locator('.settings-notice').innerText(), /Operation was not sent/);
  await pauseNextFetch(page, 'session');
  await page.getByRole('button', {name: 'Sign out of Soda'}).click();
  await hide(page);
  await restore(page);
  await releaseFetch(page);
  assert.equal(state.logoutRequests, 0);
  assert.match(await page.locator('.settings-notice').innerText(), /Soda sign-out was not sent/);
});

test('late inventory and mutation replies cannot replace a restored page or replay effects', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await pauseNextFetch(page, 'list');
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.waitForFunction(() => document.documentElement.dataset.pausedRunnerFetch === 'list');
  await hide(page); state.runners = [exampleRunner('github')]; await restore(page);
  await releaseFetch(page);
  await page.getByRole('link', {name: 'Open one in GitHub', exact: true}).waitFor();
  await pauseNextFetch(page, 'mutation');
  await page.getByRole('button', {name: 'stop one', exact: true}).click();
  await page.getByLabel('Exact runner ID').fill('one');
  await page.getByRole('button', {name: 'Confirm stop', exact: true}).click();
  await page.waitForFunction(() => document.documentElement.dataset.pausedRunnerFetch === 'mutation');
  await hide(page); await restore(page); await releaseFetch(page);
  await page.getByText(/Operation unconfirmed:/).waitFor();
  assert.equal(mutations.length, 0, 'intercepted request is never replayed to the HTTP peer');
  assert(!await page.getByText(/Native operation confirmed/).count());
  assert.equal(await page.getByRole('region', {name: 'Confirm runner operation'}).count(), 0);
});

test('unconfirmed operations survive refresh; auth loss scrubs drafts and hides inventory', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await registerDraft(page); state.mutationStatus = 502;
  await page.getByRole('button', {name: 'Register and start listener'}).click(); await settled(page);
  await page.getByText(/Operation unconfirmed:/).waitFor();
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  await page.getByText(/Operation unconfirmed:/).waitFor();
  assert(!(await page.locator('body').innerText()).includes('must-not-render-native-secret'));
  state.failList = true;
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  await page.getByRole('heading', {name: 'Stale observations'}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Register and start listener'}).isDisabled(), true);
  state.failList = false;
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  await page.getByLabel('Registration token', {exact: true}).fill('unsent-after-operation');
  state.operator = false;
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  await page.getByText('Reconnect explicitly').waitFor();
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(await page.getByRole('region', {name: 'Local runner inventory'}).count(), 0);
  assert.equal(mutations.length, 1);
});

test('Soda logout clears credentials on success and unconfirmed failure', browserCase, async t => {
  const {page, state} = await runnersPage(t);
  await registerDraft(page); state.logoutStatus = 503;
  await page.getByRole('button', {name: 'Sign out of Soda'}).click(); await settled(page);
  await page.getByText(/Soda sign-out unconfirmed/).waitFor();
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  state.logoutStatus = 204;
  await page.getByRole('button', {name: 'Sign out of Soda'}).click();
  await page.getByRole('link', {name: 'Connect to Soda', exact: true}).waitFor();
  assert.equal(await page.locator('input[type=password]').count(), 0);
});

test('first-use and denied Go shells expose explicit connection without runner metadata', browserCase, async t => {
  for (const mode of ['anonymous', 'denied']) {
    const {page, mutations} = await runnersPage(t, mode);
    assert.equal(await page.locator('soda-runners').count(), 0);
    assert.equal(await page.getByRole('link', {name: 'Connect to Soda', exact: true}).getAttribute('href'), '/-/soda/login?destination=runners');
    assert.equal(await page.getByRole('link', {name: 'Spaces', exact: true}).getAttribute('href'), origin + '/-/soda/spaces');
    if (mode === 'denied') await page.getByText(/configured Soda operator/).first().waitFor();
    assert.equal(mutations.length, 0);
  }
});

test('keyboard confirmation, native Back navigation and both-theme responsive presentation', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [exampleRunner('github')];
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  const remove = page.getByRole('button', {name: 'remove one', exact: true});
  await remove.focus(); await page.keyboard.press('Enter');
  assert.equal(await page.getByLabel('Exact runner ID').evaluate(node => node === document.activeElement), true);
  await page.keyboard.press('Escape');
  assert.equal(await remove.evaluate(node => node === document.activeElement), true);
  for (const colorScheme of ['light', 'dark'] as const) {
    await page.emulateMedia({colorScheme});
    for (const width of [1280, 390]) {
      await page.setViewportSize({width, height: 900});
      await page.evaluate(() => document.fonts.ready);
      const metrics = await page.evaluate(() => ({width: innerWidth, scroll: document.documentElement.scrollWidth, background: getComputedStyle(document.body).backgroundColor, text: getComputedStyle(document.body).color, font: getComputedStyle(document.body).fontFamily}));
      assert.equal(metrics.scroll, metrics.width);
      assert.match(metrics.font, /Barlow/); assert.notEqual(metrics.background, metrics.text);
      await page.screenshot({path: path.join(path.dirname(process.env.SODA_RUNNERS_PAGE_HTML || ''), `runners-${colorScheme}-${width}.png`), fullPage: true});
    }
  }
  await page.getByLabel('Registration token', {exact: true}).fill('unsent-before-back');
  page.on('dialog', dialog => dialog.accept());
  await page.getByRole('link', {name: 'Explore', exact: true}).click();
  await page.getByText('Native provider page').waitFor();
  await page.goBack(); await settled(page);
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(mutations.length, 0);
});
