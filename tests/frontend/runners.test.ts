import assert from 'node:assert/strict';
import test from 'node:test';
import type {TestContext} from 'node:test';
import {chromium} from 'playwright';
import type {Page} from 'playwright';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response';
import type {Runner} from '../../frontend/runners/soda-runner-types';

const origin = process.env.SODA_PAGE_ORIGIN || 'https://forgejo.example.test';
const actor = process.env.SODA_PAGE_ACTOR || '1';
const browserCase = {skip: !process.env.SODA_PAGE_ORIGIN};
const exampleRunner = (): Runner => ({
  id: 'one', provider: 'forgejo', registration_url: origin,
  account: 'soda-runner-one', architecture: 'x86-64', version: 'fixture', capacity: 1,
  service: {load: 'loaded', active: 'active', sub: 'running', enabled: 'enabled'},
});

async function runnersPage(t: TestContext, mode = 'html') {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true, ignoreDefaultArgs: ['--disable-back-forward-cache']});
  t.after(() => browser.close());
  const page = await browser.newPage({ignoreHTTPSErrors: true, storageState: process.env.SODA_PAGE_STATE || ''});
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  const state = {runners: [] as unknown[], failList: false, mutationStatus: 200, sessionStatus: 200, actor, operator: mode !== 'denied', logoutStatus: 204, logoutRequests: 0, sessionReads: 0, page: mode};
  const mutations: {path: string; body: Record<string, unknown>}[] = [];
  await page.route(origin + '/**', async route => {
    const request = route.request(), pathname = new URL(request.url()).pathname;
    if (pathname === '/user/logout') return route.fulfill({status: 503, body: 'Synthetic native logout failure'});
    if (pathname === '/-/soda/api/session') {
      state.sessionReads++;
      assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      await route.fulfill({status: state.sessionStatus, json: {user: {id: state.actor, login: 'alice'}, csrf_token: 'csrf-alice', soda_operator: state.operator, forgejo_url: origin}});
      return;
    }
    if (pathname === '/-/soda/api/login/cancel') {await route.fulfill({status: 204}); return;}
    if (pathname === '/-/soda/api/session/logout') {
      state.logoutRequests++;
      assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
      if (state.logoutStatus === 204) state.page = 'anonymous';
      await route.fulfill({status: state.logoutStatus});
      return;
    }
    if (pathname.startsWith('/-/soda/api/settings/runners')) {
      assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      if (request.method() === 'POST') {
        assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
        const body: unknown = request.postDataJSON();
        assert(body && typeof body === 'object' && !Array.isArray(body));
        mutations.push({path: pathname, body: {...body}});
        if (state.mutationStatus === 200) state.runners = pathname.endsWith('/remove') ? [] : [exampleRunner()];
        await route.fulfill({status: state.mutationStatus, json: state.mutationStatus === 200 ? {ok: true} : {error: {message: 'must-not-render-native-secret'}}});
        return;
      }
      await route.fulfill({status: state.failList ? 503 : 200, json: {forgejo_url: origin, runners: state.runners, runner_count: state.runners.length, active_listeners: state.runners.length, total_capacity: state.runners.length}});
      return;
    }
    await route.continue();
  });
  await page.goto(origin + '/?soda-view=runners');
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
  await page.waitForFunction(() => !document.querySelector<HTMLButtonElement>('soda-runners button')?.disabled);
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
        ? {user: {id: document.getElementById('soda-native-content')?.dataset.actor, login: 'alice'}, csrf_token: 'old-csrf', soda_operator: true, forgejo_url: location.origin}
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

test('native HTML and emitted Lit register Forgejo and confirm exact lifecycle targets', browserCase, async t => {
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
  assert.equal(await page.getByRole('combobox').count(), 0);
  assert.equal(await page.locator('input[name=registration_url]').count(), 0);
  assert(state.sessionReads >= 10);
});

test('Forgejo links use only the configured public origin and unsupported observations fail closed', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  assert.equal(await page.getByRole('link', {name: 'Forgejo Actions administration', exact: true}).getAttribute('href'), origin + '/admin/actions/runners');
  for (const address of ['http://internal-only:3000', 'javascript:alert(1)', 'https://external.example.test/repo']) {
    state.runners = [{...exampleRunner(), registration_url: address}];
    await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
    assert.equal(await page.getByRole('link', {name: 'Open one in Forgejo', exact: true}).getAttribute('href'), origin);
    assert(!(await page.locator('body').innerText()).includes(address));
  }
  state.runners = [{...exampleRunner(), provider: 'github'}];
  await page.getByRole('button', {name: 'Refresh status'}).click(); await settled(page);
  await page.getByRole('heading', {name: 'Stale observations'}).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Register and start listener'}).isDisabled(), true);
  assert.equal(await page.getByRole('link', {name: /GitHub/}).count(), 0);
  assert.equal(mutations.length, 0);
});

test('Forgejo field guidance and changed actors prevent invalid registration', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await registerDraft(page);
  await page.getByLabel('Labels', {exact: true}).fill('soda-linux');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  assert.match(await page.getByLabel('Labels', {exact: true}).evaluate(node => node instanceof HTMLInputElement ? node.validationMessage : ''), /name:host/);
  assert.equal(mutations.length, 0);
  await page.getByLabel('Labels', {exact: true}).fill('soda-linux:host');
  await page.getByLabel('Forgejo runner UUID').fill('invalid');
  await page.getByRole('button', {name: 'Register and start listener'}).click();
  assert.equal(mutations.length, 0);
  await page.getByLabel('Forgejo runner UUID').fill('33834eef-e758-48c4-a676-1745426747aa');
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
  await page.getByRole('button', {name: 'Sign out', exact: true}).click();
  await hide(page);
  await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true})));
  await releaseFetch(page);
  assert.equal(state.logoutRequests, 0);
  await page.getByText(/Soda sign-out could not be confirmed/).waitFor();
});

test('late inventory and mutation replies cannot replace a restored page or replay effects', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  await pauseNextFetch(page, 'list');
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await page.waitForFunction(() => document.documentElement.dataset.pausedRunnerFetch === 'list');
  await hide(page); state.runners = [exampleRunner()]; await restore(page);
  await releaseFetch(page);
  await page.getByRole('link', {name: 'Open one in Forgejo', exact: true}).waitFor();
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
  await page.getByRole('button', {name: 'Sign out', exact: true}).click();
  await page.getByText(/Soda sign-out could not be confirmed/).waitFor();
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  state.logoutStatus = 204;
  await page.getByRole('button', {name: 'Retry sign-out'}).click();
  await page.getByRole('button', {name: 'Retry Forgejo sign-out'}).waitFor();
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
});

test('native runner controls refuse non-operator access without listing or mutation', browserCase, async t => {
 const {page, mutations} = await runnersPage(t, 'denied');
 await page.getByText('Operator authorization unavailable.', {exact: false}).waitFor();
 assert.equal(await page.getByLabel('Registration token', {exact: true}).isDisabled(), true);
 assert.equal(mutations.length, 0);
});

test('keyboard confirmation, native Back navigation and both-theme responsive presentation', browserCase, async t => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [exampleRunner()];
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
    }
  }
  await page.getByLabel('Registration token', {exact: true}).fill('unsent-before-back');
  page.on('dialog', dialog => dialog.accept());
  await page.setViewportSize({width: 1280, height: 900});
  await page.locator('#navbar a[href="/issues"]').click();
  await page.waitForURL('**/issues');
  await page.goBack(); await settled(page);
  assert.equal(await page.getByLabel('Registration token', {exact: true}).inputValue(), '');
  assert.equal(mutations.length, 0);
});
