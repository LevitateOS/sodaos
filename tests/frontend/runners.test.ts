import assert from 'node:assert/strict';
import test from 'node:test';
import type {TestContext} from 'node:test';
import {chromium} from 'playwright';
import type {Page} from 'playwright';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response';
import type {Runner} from '../../frontend/runners/soda-runner-types';
import payload from '../../internal/release/build/forgejo-payload.json';
import {object} from '../../frontend/spaces/sodaspaces-api';

const origin = process.env.SODA_PAGE_ORIGIN || 'https://forgejo.example.test';
const actor = process.env.SODA_PAGE_ACTOR || '1';
const componentOnly = process.env.SODA_RUNNERS_COMPONENT === '1';
assert(
  !(componentOnly && process.env.SODA_PAGE_ORIGIN),
  'Component fixtures never substitute for native-page consumers'
);
const browserCase = {skip: !process.env.SODA_PAGE_ORIGIN && !componentOnly};
const files: Record<string, string> = payload;
const exampleRunner = (): Runner => ({
  id: 'one',
  provider: 'forgejo',
  registration_url: origin,
  account: 'soda-runner-one',
  architecture: 'x86-64',
  version: 'fixture',
  capacity: 1,
  service: {load: 'loaded', active: 'active', sub: 'running', enabled: 'enabled'},
});

async function runnersPage(t: TestContext, mode = 'html') {
  const browser = await chromium.launch({
    headless: true,
    chromiumSandbox: true,
    ignoreDefaultArgs: ['--disable-back-forward-cache'],
  });
  t.after(() => browser.close());
  const page = await browser.newPage({ignoreHTTPSErrors: true, storageState: process.env.SODA_PAGE_STATE || ''});
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  const state = {
    runners: [] as unknown[],
    unavailable: [] as string[],
    csrf: 'csrf-alice',
    failList: false,
    omitForgejo: false,
    mutationStatus: 200,
    sessionStatus: 200,
    actor,
    operator: mode !== 'denied',
    logoutStatus: 204,
    logoutRequests: 0,
    sessionReads: 0,
    page: mode,
  };
  const mutations: {path: string; body: Record<string, unknown>}[] = [];
  await page.route(origin + '/**', async (route) => {
    const request = route.request(),
      pathname = new URL(request.url()).pathname;
    if (pathname === '/user/logout') return route.fulfill({status: 503, body: 'Synthetic native logout failure'});
    if (pathname === '/-/soda/api/session') {
      state.sessionReads++;
      assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      await route.fulfill({
        status: state.sessionStatus,
        json: {
          user: {id: state.actor, login: 'alice'},
          csrf_token: state.csrf,
          soda_operator: state.operator,
          forgejo_url: origin,
        },
      });
      return;
    }
    if (pathname === '/-/soda/api/login/cancel') {
      await route.fulfill({status: 204});
      return;
    }
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
      if (
        state.sessionStatus !== 200 ||
        !state.operator ||
        state.actor !== actor ||
        (request.method() === 'POST' && request.headers()['x-csrf-token'] !== state.csrf)
      )
        return route.fulfill({status: 403, json: {error: {code: 'reauthentication_required'}}});
      if (request.method() === 'POST') {
        assert.equal(request.headers()['x-csrf-token'], 'csrf-alice');
        const body: unknown = request.postDataJSON();
        assert(body && typeof body === 'object' && !Array.isArray(body));
        mutations.push({path: pathname, body: {...body}});
        if (state.mutationStatus === 200) state.runners = pathname.endsWith('/remove') ? [] : [exampleRunner()];
        await route.fulfill({
          status: state.mutationStatus,
          json: state.mutationStatus === 200 ? {ok: true} : {error: {message: 'must-not-render-native-secret'}},
        });
        return;
      }
      const complete =
        state.unavailable.length === 0 &&
        state.runners.every((row) => object(row).service !== null && object(row).version !== '');
      const active = state.runners.filter((row) => {
        const service = object(row).service;
        return service !== null && object(service).active === 'active' && object(service).sub === 'running';
      }).length;
      await route.fulfill({
        status: state.failList ? 503 : 200,
        json: {
          complete,
          unavailable: state.unavailable,
          runners: state.runners,
          runner_count: state.runners.length,
          active_listeners: active,
          total_capacity: state.runners.length,
          ...(state.omitForgejo ? {} : {forgejo_url: origin}),
        },
      });
      return;
    }
    if (componentOnly) {
      if (pathname === '/admin')
        return route.fulfill({
          contentType: 'text/html',
          body: `<!doctype html><meta name="viewport" content="width=device-width"><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda-settings.css"><nav id="navbar"><a href="#" class="link-action" data-url="/user/logout">Native logout fixture</a></nav><div id="soda-settings-link" data-actor="${actor}" data-sub-url=""></div><div id="soda-native-content" class="soda-settings soda-runner-settings" data-actor="${actor}" data-view="runners" data-document-title="Runners component fixture"></div><script type="module" src="/assets/soda/forgejo/soda-native-page.js"></script>`,
        });
      const source = files['public' + pathname];
      assert(source, 'Unmapped component asset');
      const file = source.startsWith('@build/forgejo-js/')
        ? '.artifacts/forgejo-js/' + source.split('/').at(-1)
        : source;
      return route.fulfill({path: new URL('../../' + file, import.meta.url).pathname});
    }
    await route.continue();
  });
  await page.goto(origin + '/admin?soda-view=runners');
  if (mode === 'html') await page.getByText('No local runners registered.').waitFor();
  t.after(() => assert.deepEqual(errors, []));
  return {page, state, mutations};
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
  await page.evaluate((kind) => {
    const original = window.fetch;
    let armed = true;
    window.fetch = Object.assign(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      const matches =
        kind === 'session'
          ? url.endsWith('/api/session')
          : url.includes('/api/settings/runners') &&
            (kind === 'mutation' ? init?.method === 'POST' : init?.method === 'GET');
      if (!armed || !matches) return original(input, init);
      armed = false;
      document.documentElement.dataset.pausedRunnerFetch = kind;
      await new Promise<void>((resolve) =>
        window.addEventListener('release-runner-fetch', () => resolve(), {once: true})
      );
      const json =
        kind === 'session'
          ? {
              user: {id: document.getElementById('soda-native-content')?.dataset.actor, login: 'alice'},
              csrf_token: 'old-csrf',
              soda_operator: true,
              forgejo_url: location.origin,
            }
          : kind === 'mutation'
            ? {ok: true}
            : {
                forgejo_url: location.origin,
                complete: true,
                unavailable: [],
                runners: [],
                runner_count: 0,
                active_listeners: 0,
                total_capacity: 0,
              };
      return new Response(JSON.stringify(json), {headers: {'Content-Type': 'application/json'}});
    }, original);
  }, kind);
}
async function releaseFetch(page: Page) {
  await page.evaluate(() => window.dispatchEvent(new Event('release-runner-fetch')));
  // Let chained async continuations and Lit updates settle without a wall-clock sleep.
  await page.evaluate(
    () => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
  );
}

test('runner response enforces native one-slot validation', () => {
  assert.throws(() =>
    decodeRunnerResponse('list', {runners: [], runner_count: 1, active_listeners: 0, total_capacity: 0})
  );
  assert.throws(() => decodeRunnerResponse('remove', {ok: false}));
  assert.throws(() =>
    decodeRunnerResponse('list', {runners: [], runner_count: 0, active_listeners: -1, total_capacity: 0})
  );
  assert.equal(decodeRunnerResponse('stop', {ok: true}).ok, true);
  const omittedOrigin = {
    complete: true,
    unavailable: [],
    runners: [],
    runner_count: 0,
    active_listeners: 0,
    total_capacity: 0,
  };
  assert.deepEqual(decodeRunnerResponse('list', omittedOrigin), omittedOrigin);
});

test('partial runner response retains known rows and refuses false completeness or counts', () => {
  const row = exampleRunner();
  const partial = {
    forgejo_url: origin,
    complete: false,
    unavailable: ['broken'],
    runners: [row, {...row, id: 'two', account: 'soda-runner-two', service: null, version: ''}],
    runner_count: 2,
    active_listeners: 1,
    total_capacity: 2,
  };
  assert.deepEqual(decodeRunnerResponse('list', partial), partial);
  for (const change of [
    {complete: true},
    {total_capacity: 3},
    {active_listeners: 2},
    {unavailable: ['one']},
    {unavailable: ['../bad']},
    {unavailable: ['broken', 'broken']},
  ])
    assert.throws(() => decodeRunnerResponse('list', {...partial, ...change}));
  assert.throws(() =>
    decodeRunnerResponse('list', {...partial, runners: [{...row, account: 'root'}, partial.runners[1]]})
  );
});

test('omitted Forgejo origin keeps local inventory available', browserCase, async (t) => {
  const {page, state} = await runnersPage(t);
  state.omitForgejo = true;
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await settled(page);
  await page.getByText('No local runners registered.').waitFor();
});

test('partial inventory shows unknown capacity without locking readable runner actions', browserCase, async (t) => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [
    exampleRunner(),
    {...exampleRunner(), id: 'two', account: 'soda-runner-two', service: null, version: ''},
  ];
  state.unavailable = ['broken'];
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await settled(page);
  await page.getByRole('heading', {name: 'Partial local observations'}).waitFor();
  await page.getByText(/not complete or available capacity/).waitFor();
  await page.getByText(/Service and boot policy unavailable/).waitFor();
  assert.equal(await page.getByRole('button', {name: 'start broken', exact: true}).count(), 0);
  assert(!(await page.getByRole('button', {name: 'stop one', exact: true}).isDisabled()));
  await page.getByRole('button', {name: 'stop one', exact: true}).click();
  await page.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(mutations.length, 0);
});

test('cleanup retains exact-target confirmation while execution controls are unavailable', browserCase, async (t) => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [exampleRunner()];
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await settled(page);
  await page.getByText(/Local CI execution is unavailable/).waitFor();
  assert.equal(await page.getByRole('button', {name: 'Register and start listener'}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'start one', exact: true}).count(), 0);
  assert.equal(await page.getByRole('button', {name: 'restart one', exact: true}).count(), 0);
  assert.equal(await page.locator('input[type=password]').count(), 0);
  await page.getByRole('button', {name: 'remove one', exact: true}).click();
  await page.getByLabel('Exact runner ID').fill('wrong');
  await page.getByRole('button', {name: 'Confirm remove', exact: true}).click();
  await page.getByText('Type the exact runner ID to confirm.').waitFor();
  assert.equal(mutations.length, 0);
  await page.getByLabel('Exact runner ID').fill('one');
  await page.getByRole('button', {name: 'Confirm remove', exact: true}).click();
  await settled(page);
  assert.deepEqual(
    mutations.map((value) => value.body),
    [{confirm_id: 'one'}]
  );
});

test('cleanup rechecks authority and preserves uncertain outcomes without replay', browserCase, async (t) => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [exampleRunner()];
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await settled(page);
  state.mutationStatus = 502;
  await page.getByRole('button', {name: 'stop one', exact: true}).click();
  await page.getByRole('button', {name: 'Confirm stop', exact: true}).click();
  await settled(page);
  await page.getByText(/Operation unconfirmed:/).waitFor();
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await settled(page);
  assert.equal(mutations.length, 1);
  assert(!(await page.locator('body').innerText()).includes('must-not-render-native-secret'));
  state.csrf = 'rotated-csrf';
  await page.getByRole('button', {name: 'stop one', exact: true}).click();
  await page.getByRole('button', {name: 'Confirm stop', exact: true}).click();
  await settled(page);
  await page.getByRole('link', {name: 'Reconnect explicitly', exact: true}).waitFor();
  assert.equal(mutations.length, 1);
  assert.equal(await page.getByRole('region', {name: 'Local runner inventory'}).count(), 0);
});

test('retired cleanup response is not shown as confirmed or replayed', browserCase, async (t) => {
  const {page, state, mutations} = await runnersPage(t);
  state.runners = [exampleRunner()];
  await page.getByRole('button', {name: 'Refresh status'}).click();
  await settled(page);
  await page.getByRole('button', {name: 'stop one', exact: true}).click();
  await pauseNextFetch(page, 'mutation');
  await page.getByRole('button', {name: 'Confirm stop', exact: true}).click();
  await page.waitForFunction(() => document.documentElement.dataset.pausedRunnerFetch === 'mutation');
  await hide(page);
  await restore(page);
  await releaseFetch(page);
  await page.getByText(/Operation unconfirmed:/).waitFor();
  assert.equal(mutations.length, 0);
  assert.equal(await page.getByText(/Native operation confirmed/).count(), 0);
});

test('runner cleanup refuses non-operator access without mutations', browserCase, async (t) => {
  const {page, mutations} = await runnersPage(t, 'denied');
  await page.getByText('Operator authorization unavailable.', {exact: false}).waitFor();
  assert.equal(await page.getByRole('region', {name: 'Local runner inventory'}).count(), 0);
  assert.equal(mutations.length, 0);
});

test(
  'populated runner pages keep long inventory, drafts and keyboard controls usable without writes',
  browserCase,
  async (t) => {
    const {page, state, mutations} = await runnersPage(t);
    state.runners = Array.from({length: 24}, (_, index) => ({
      ...exampleRunner(),
      id: 'runner-' + index,
      account: 'soda-runner-runner-' + index,
      version: 'fixture-version-' + 'long-native-version-'.repeat(12),
    }));
    for (const colorScheme of ['light', 'dark'] as const) {
      await page.emulateMedia({colorScheme});
      for (const width of [360, 768, 1440]) {
        await page.setViewportSize({width, height: 720});
        await page.getByRole('button', {name: 'Refresh status', exact: true}).click();
        await settled(page);
        await page.evaluate(() => document.fonts.ready);
        assert.equal(await page.locator('soda-runners .settings-runner').count(), 24);
        const metrics = await page.evaluate(() => ({
          width: innerWidth,
          scroll: document.documentElement.scrollWidth,
          height: innerHeight,
          content: document.documentElement.scrollHeight,
        }));
        if (metrics.scroll !== metrics.width)
          t.diagnostic(
            JSON.stringify(
              await page.evaluate(() =>
                [...document.querySelectorAll('body *')]
                  .filter((node) => node.getBoundingClientRect().right > innerWidth + 1)
                  .map((node) => ({
                    tag: node.tagName,
                    class: node.className,
                    right: node.getBoundingClientRect().right,
                    width: node.getBoundingClientRect().width,
                  }))
                  .slice(0, 25)
              )
            )
          );
        assert.equal(metrics.scroll, metrics.width, 'Long inventory caused document horizontal overflow');
        assert(metrics.content > metrics.height, 'Fixture must exercise vertical scrolling');
        const remove = page.getByRole('button', {name: 'remove runner-23', exact: true});
        await remove.scrollIntoViewIfNeeded();
        await remove.focus();
        await page.keyboard.press('Enter');
        const confirmation = page.getByLabel('Exact runner ID', {exact: true});
        assert(await confirmation.evaluate((node) => node === document.activeElement));
        await confirmation.scrollIntoViewIfNeeded();
        const box = await confirmation.boundingBox();
        // Chromium scroll offsets are quantized; DOM rectangles retain subpixels.
        // Permit only the same one-CSS-pixel rounding used by layout checks.
        assert(
          box && box.x >= -1 && box.x + box.width <= width + 1 && box.y >= -1 && box.y + box.height <= 721,
          JSON.stringify({colorScheme, width, confirmation: box})
        );
        await page.keyboard.press('Escape');
        assert(await remove.evaluate((node) => node === document.activeElement));
        assert.equal(await confirmation.count(), 0);
        assert.equal(await page.locator('#soda-native-content > soda-runners').count(), 1);
        assert.equal(mutations.length, 0, 'Layout/refresh/confirmation cancellation dispatched a mutation');
      }
    }
  }
);
