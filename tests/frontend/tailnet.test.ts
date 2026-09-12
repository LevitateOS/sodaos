import assert from 'node:assert/strict';
import test from 'node:test';
import type {TestContext} from 'node:test';
import {chromium} from 'playwright';
import type {Page} from 'playwright';
import {authenticationURL, hostResult, settingsView} from '../../frontend/tailnet/soda-tailnet-response';
import type {Settings} from '../../frontend/tailnet/soda-tailnet-response';
import payload from '../../internal/nativebuild/forgejo-payload.json';

const origin = process.env.SODA_PAGE_ORIGIN || 'https://forgejo.example.test';
const actor = process.env.SODA_PAGE_ACTOR || '1';
const componentOnly = process.env.SODA_TAILNET_COMPONENT === '1';
assert(!(componentOnly && process.env.SODA_PAGE_ORIGIN), 'Component fixtures must never substitute for native-page consumers');
const nativeCase = {skip: !process.env.SODA_PAGE_ORIGIN};
const behaviorCase = {skip: !process.env.SODA_PAGE_ORIGIN && !componentOnly};
const files: Record<string, string> = payload;
const snapshot = (): Settings => ({host_unavailable: false, host: {
  revision: 'a'.repeat(64), state: 'Running', have_node_key: true, expired: false,
  tailnet: 'soda.example.test', magic_dns_enabled: true, dns_name: 'appliance.soda.ts.net', addresses: ['100.64.0.1'], health_issues: 1,
  preferences: {want_running: true, exit_node_id: '', exit_node_ip: '', allow_lan: false, advertise_exit_node: false},
  peers: [{id: 'exit', dns_name: 'exit.soda.ts.net', addresses: ['100.64.0.2'], online: true, exit_node: true, expired: false}],
}, enrollment: {revision: '0', binding: '', tailnet: '', tags: [], configured: false, admission: false, default: false,
  preauthorized: false, credential_checked: false, enrollment_verified: false, runtime_supported: false}});

test('Tailnet projections reject malformed, unsafe and mixed-version responses', () => {
  const value = snapshot(); assert.deepEqual(settingsView(value), value);
  assert.throws(() => settingsView({host: null, enrollment: value.enrollment}));
  assert.throws(() => settingsView({...value, enrollment: {...value.enrollment, runtime_supported: true}}));
  assert.throws(() => settingsView({...value, host: {...value.host, state: 'invented'}}));
  for (const url of ['http://login.tailscale.com/a/test', 'https://evil.test/a/test', 'https://user@login.tailscale.com/a/test', 'https://login.tailscale.com/a/test?secret=x', 'https://login.tailscale.com/a/test#secret']) assert.throws(() => authenticationURL(url));
  assert.throws(() => hostResult({outcome: 'pending', host: value.host, readback_unavailable: false}, 'signin'));
  assert.throws(() => hostResult({outcome: 'confirmed', host: value.host, readback_unavailable: false, auth_url: 'https://login.tailscale.com/a/test'}, 'logout'));
  const safe = settingsView({...value, host: {...value.host, auth_url: 'must-not-project', Health: ['private diagnostic']}});
  assert(!JSON.stringify(safe).includes('must-not-project')); assert(!JSON.stringify(safe).includes('private diagnostic'));
});

async function tailnetPage(t: TestContext, operator = true) {
  const browser = await chromium.launch({headless: true, chromiumSandbox: true, ignoreDefaultArgs: ['--disable-back-forward-cache']});
  t.after(() => browser.close());
  const page = await browser.newPage({ignoreHTTPSErrors: true, storageState: process.env.SODA_PAGE_STATE || ''});
  const errors: string[] = []; page.on('pageerror', e => errors.push(e.message)); t.after(() => assert.deepEqual(errors, []));
  const state = {data: snapshot(), actor, operator, readStatus: 200, postStatus: 200, readbackUnavailable: false, reads: 0};
  const posts: {path: string; body: Record<string, unknown>}[] = [];
  await page.route(origin + '/**', async route => {
    const request = route.request(), path = new URL(request.url()).pathname;
    if (path === '/-/soda/api/session') {
      assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      return route.fulfill({json: {user: {id: state.actor, login: 'operator'}, csrf_token: 'fixture-csrf', soda_operator: state.operator, forgejo_url: origin}});
    }
    if (path.startsWith('/-/soda/api/settings/tailnet')) {
      assert.equal(request.headers()['x-soda-expected-user-id'], actor);
      if (request.method() === 'GET') {state.reads++; return route.fulfill({status: state.readStatus, json: state.readStatus === 200 ? state.data : {error: {message: 'private-native-diagnostic'}}});}
      assert.equal(request.headers()['x-csrf-token'], 'fixture-csrf');
      const body: unknown = request.postDataJSON(); assert(body && typeof body === 'object' && !Array.isArray(body));
      const data: Record<string, unknown> = {...body}; posts.push({path, body: data});
      if (state.postStatus !== 200) return route.fulfill({status: state.postStatus, json: {error: {message: 'private-native-diagnostic'}}});
      if (path.endsWith('/host')) {
        const h = state.data.host; assert(h);
        if (data.action === 'exit-node') {h.preferences.exit_node_id = data.exit_node ? 'exit' : ''; h.preferences.exit_node_ip = ''; h.preferences.allow_lan = data.allow_lan === true;}
        if (data.action === 'advertise-exit-node') h.preferences.advertise_exit_node = data.advertise === true;
        if (data.action === 'logout') {h.state = 'NeedsLogin'; h.have_node_key = false;}
        h.revision = 'b'.repeat(64);
        const authenticating = data.action === 'signin' || data.action === 'authentication';
        if (authenticating) h.state = 'NeedsLogin';
        return route.fulfill({json: {outcome: authenticating ? 'pending' : 'confirmed', host: state.readbackUnavailable ? null : h, readback_unavailable: state.readbackUnavailable,
          ...(authenticating ? {auth_url: 'https://login.tailscale.com/a/synthetic-ui-probe'} : {})}});
      }
      const saved = data.action !== 'check';
      if (saved) {
        const previous = state.data.enrollment;
        state.data.enrollment = {...previous, configured: true, credential_checked: true, revision: (previous.revision === 'c'.repeat(32) ? 'd' : 'c').repeat(32), binding: 'e'.repeat(32),
          tailnet: typeof data.tailnet === 'string' ? data.tailnet : previous.tailnet, tags: Array.isArray(data.tags) ? data.tags.filter((v): v is string => typeof v === 'string') : previous.tags,
          preauthorized: data.preauthorized === true, admission: data.action !== 'disable'};
      }
      return route.fulfill({json: {saved, outcome: 'confirmed', credential_checked: true, enrollment: state.data.enrollment}});
    }
    if (componentOnly) {
      // Explicit component fixture, never a fallback or substitute for native HTML.
      if (path === '/') return route.fulfill({contentType: 'text/html', body: `<!doctype html><meta name="viewport" content="width=device-width"><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda-settings.css"><link rel="stylesheet" href="/assets/soda-tailnet.css"><div id="soda-native-content" class="soda-settings soda-tailnet-settings" data-actor="${actor}" data-view="tailnet" data-document-title="Tailnet component fixture"></div><script type="module" src="/assets/soda/forgejo/soda-native-page.js"></script>`});
      const source = files['public' + path]; assert(source, 'Unmapped component asset');
      const file = source.startsWith('@build/forgejo-js/') ? '.artifacts/forgejo-js/' + source.split('/').at(-1) : source;
      return route.fulfill({path: new URL('../../' + file, import.meta.url).pathname});
    }
    return route.continue(); // Native mode never replaces HTML/CSP/chrome/assets/login.
  });
  await page.goto(origin + '/?soda-view=tailnet');
  await page.locator('soda-tailnet').waitFor();
  if (operator) await page.getByRole('heading', {name: 'Automatic project access', exact: true}).waitFor();
  else await page.getByText('The original Soda operator is unavailable.', {exact: false}).waitFor();
  return {page, state, posts};
}
async function settled(page: Page) {await page.waitForFunction(() => !document.querySelector<HTMLButtonElement>('soda-tailnet button')?.disabled);}
async function credential(page: Page) {
  await page.getByLabel('Managed Tailnet', {exact: true}).fill('soda.example.test');
  await page.getByLabel('Project tags (comma separated)', {exact: true}).fill('tag:soda-project');
  await page.getByLabel('OAuth client ID', {exact: true}).fill('synthetic-client');
  await page.getByLabel('OAuth client secret', {exact: true}).fill('tskey-client-synthetic-ui-only');
}
async function frame(page: Page) {await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));}
async function hide(page: Page) {await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true})));}
async function restore(page: Page) {await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true}))); await settled(page);}
async function pause(page: Page, kind: 'session' | 'mutation') {
  await page.evaluate(({kind, actor}) => {
    const original = window.fetch; let armed = true;
    window.fetch = Object.assign(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      const match = kind === 'session' ? url.endsWith('/api/session') : url.includes('/api/settings/tailnet/') && init?.method === 'POST';
      if (!armed || !match) return original(input, init);
      armed = false; document.documentElement.dataset.tailnetPaused = kind;
      await new Promise<void>(resolve => window.addEventListener('release-tailnet', () => resolve(), {once: true}));
      return new Response(JSON.stringify(kind === 'session' ? {user: {id: actor, login: 'operator'}, csrf_token: 'old', soda_operator: true, forgejo_url: location.origin} : {outcome: 'confirmed', host: null, readback_unavailable: true}), {headers: {'Content-Type': 'application/json'}});
    }, original);
  }, {kind, actor});
}

test('native Tailnet host retains native chrome, scopes, theme and narrow keyboard layout', nativeCase, async t => {
  const {page, posts} = await tailnetPage(t);
  assert.equal(await page.locator('main,[role=main]').count(), 1);
  assert.equal(await page.locator('#sodaspaces-root').count(), 0);
  assert(await page.locator('#navbar').isVisible()); assert((await page.title()).startsWith('Tailnet - '));
  assert.equal(await page.getByRole('link', {name: 'Tailnet', exact: true}).count(), 1);
  for (const theme of ['soda-light', 'soda-dark']) for (const width of [390, 1440]) {
    await page.setViewportSize({width, height: 1000});
    await page.evaluate(theme => document.documentElement.setAttribute('data-theme', theme), theme);
    const box = await page.locator('soda-tailnet').boundingBox(); assert(box && box.width > 0 && box.x >= 0 && box.x + box.width <= width);
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    assert.notEqual(await page.locator('soda-tailnet button').first().evaluate(el => getComputedStyle(el).minHeight), '0px');
  }
  await page.getByRole('button', {name: 'Disconnect appliance', exact: true}).focus(); await page.keyboard.press('Enter');
  await page.getByRole('button', {name: 'Confirm logout', exact: true}).waitFor();
  assert(await page.getByRole('button', {name: 'Confirm logout', exact: true}).evaluate(el => el === document.activeElement));
  assert.match(await page.getByRole('region', {name: 'Confirm logout'}).innerText(), /No alternative management path.*verified/);
  await page.keyboard.press('Escape'); assert(await page.getByRole('button', {name: 'Disconnect appliance', exact: true}).evaluate(el => el === document.activeElement));
  assert.equal(posts.length, 0); // Entry/read/theme/focus/confirmation never refresh Forgejo or enroll.
});

test('emitted component keyboard confirmation and token-based light/dark narrow/wide layout', behaviorCase, async t => {
  const {page, posts} = await tailnetPage(t);
  const backgrounds: string[] = [];
  for (const scheme of ['light', 'dark']) for (const width of [390, 1440]) {
    await page.setViewportSize({width, height: 1000});
    await page.evaluate(scheme => document.documentElement.style.colorScheme = scheme, scheme);
    const box = await page.locator('soda-tailnet').boundingBox(); assert(box && box.x >= 0 && box.width > 0 && box.x + box.width <= width);
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    backgrounds.push(await page.locator('soda-tailnet fieldset').first().evaluate(el => getComputedStyle(el).backgroundColor));
    assert.equal(await page.locator('soda-tailnet button').first().evaluate(el => getComputedStyle(el).minHeight), '44px');
  }
  assert.notEqual(backgrounds[0], backgrounds[2]);
  await page.getByRole('button', {name: 'Disconnect appliance', exact: true}).focus(); await page.keyboard.press('Enter');
  await page.getByRole('button', {name: 'Confirm logout', exact: true}).waitFor();
  assert(await page.getByRole('button', {name: 'Confirm logout', exact: true}).evaluate(el => el === document.activeElement));
  await page.keyboard.press('Escape'); assert(await page.getByRole('button', {name: 'Disconnect appliance', exact: true}).evaluate(el => el === document.activeElement));
  assert.equal(posts.length, 0);
});

test('host component operations use independent explicit confirmation and safe transient authentication', behaviorCase, async t => {
  const {page, posts, state} = await tailnetPage(t);
  await page.getByRole('button', {name: 'Resume / reauthenticate', exact: true}).click();
  const link = page.getByRole('link', {name: 'Continue appliance sign-in at Tailscale'}); await link.waitFor();
  assert.equal(await link.getAttribute('referrerpolicy'), 'no-referrer'); assert.equal(await link.getAttribute('rel'), 'noopener noreferrer');
  assert(!await page.evaluate(() => JSON.stringify(localStorage).includes('synthetic-ui-probe') || JSON.stringify(sessionStorage).includes('synthetic-ui-probe')));
  await hide(page); assert.equal(await page.locator('[data-authentication][href]').count(), 0); await restore(page);
  assert.equal(await link.count(), 0); assert.equal(posts.length, 1);
  await page.getByRole('button', {name: 'Recover authentication link'}).click(); await link.waitFor();
  await page.getByRole('button', {name: 'Refresh Forgejo advertisement', exact: true}).click();
  assert.match(await page.getByRole('region', {name: 'Confirm refresh-forgejo'}).innerText(), /restart Forgejo/);
  state.readbackUnavailable = true;
  await page.getByRole('button', {name: 'Confirm refresh-forgejo', exact: true}).click();
  await page.getByText('Host readback failed independently.', {exact: false}).waitFor();
  assert.match(await page.locator('.tailnet-notice').innerText(), /acknowledged/);
  await page.getByRole('button', {name: 'Refresh observations'}).click(); await settled(page);
  assert.match(await page.locator('.tailnet-notice').innerText(), /readback failed/);
  assert.deepEqual(posts.map(p => p.body.action), ['signin', 'authentication', 'refresh-forgejo']);
});

test('drafts and revisions survive refresh, targeted writes and failed native results', behaviorCase, async t => {
  const {page, state, posts} = await tailnetPage(t);
  await page.getByLabel('Exit node', {exact: true}).selectOption('100.64.0.2');
  await page.getByLabel('Allow local LAN while using exit node').check();
  await page.getByLabel('Advertise appliance as an exit node').check();
  await page.getByRole('button', {name: 'Refresh observations'}).click(); await settled(page);
  assert.equal(await page.getByLabel('Exit node', {exact: true}).inputValue(), '100.64.0.2');
  await page.getByRole('button', {name: 'Apply exit node', exact: true}).click();
  await page.getByRole('button', {name: 'Confirm exit-node', exact: true}).click(); await settled(page);
  assert.deepEqual(posts[0]?.body, {action: 'exit-node', revision: 'a'.repeat(64), confirm: 'exit-node', exit_node: '100.64.0.2', allow_lan: true});
  assert(await page.getByLabel('Advertise appliance as an exit node').isChecked(), 'unsubmitted advertisement draft must survive the exit write');
  state.postStatus = 409;
  await page.getByRole('button', {name: 'Apply advertisement', exact: true}).click();
  await page.getByRole('button', {name: 'Confirm advertise-exit-node', exact: true}).click(); await settled(page);
  assert.equal(posts[1]?.body.revision, 'a'.repeat(64), 'refresh must not silently rebase an edited scope');
  assert(await page.getByLabel('Advertise appliance as an exit node').isChecked());
  assert(!await page.locator('soda-tailnet').innerText().then(text => text.includes('private-native-diagnostic')));
  await page.getByRole('button', {name: 'Refresh observations'}).click(); await settled(page);
  assert.match(await page.locator('.tailnet-notice').innerText(), /Request rejected/); assert.equal(posts.length, 2);
});

test('credential check/save/rotate/admission are explicit, redacted and do not enable runtime', behaviorCase, async t => {
  const {page, state, posts} = await tailnetPage(t);
  await credential(page); await page.getByRole('button', {name: 'Check credential only'}).click();
  assert.equal(await page.getByLabel('OAuth client secret', {exact: true}).inputValue(), '');
  await page.getByText('Credential check passed;', {exact: false}).waitFor();
  assert.equal(state.data.enrollment.configured, false);
  await credential(page); await page.getByLabel('I reviewed private-network exposure', {exact: false}).check();
  await page.getByRole('button', {name: 'Save binding', exact: true}).click(); await page.getByText('Policy saved.', {exact: false}).waitFor();
  assert.equal(posts[1]?.body.client_secret, 'tskey-client-synthetic-ui-only');
  assert(!await page.evaluate(() => document.documentElement.outerHTML.includes('tskey-client-synthetic-ui-only') || JSON.stringify(localStorage).includes('tskey-client-synthetic-ui-only') || JSON.stringify(sessionStorage).includes('tskey-client-synthetic-ui-only')));
  await page.getByLabel('Configuration action', {exact: true}).selectOption('rotate');
  assert(await page.getByLabel('Managed Tailnet', {exact: true}).getAttribute('readonly') !== null);
  await page.getByLabel('OAuth client secret', {exact: true}).fill('tskey-client-synthetic-rotated');
  await page.getByRole('button', {name: 'Rotate credential', exact: true}).click(); await settled(page);
  assert.equal(posts[2]?.body.action, 'rotate'); assert.equal(posts[2]?.body.tailnet, 'soda.example.test');
  await page.getByRole('button', {name: 'Close future admission'}).click();
  await page.getByRole('button', {name: 'Confirm close admission'}).click(); await settled(page);
  assert.equal(state.data.enrollment.admission, false); assert.equal(state.data.enrollment.default, false);
  assert.equal(posts.filter(p => p.body.action === 'enable' || p.body.action === 'retry' || p.body.default === true).length, 0);
});

test('denied operator and actor loss expose no private controls or credentials', behaviorCase, async t => {
  const denied = await tailnetPage(t, false); assert.equal(denied.state.reads, 0); assert.equal(await denied.page.locator('soda-tailnet fieldset').count(), 0);
  const {page, state, posts} = await tailnetPage(t);
  await credential(page); state.actor = String(BigInt(actor) + 1n);
  await page.getByRole('button', {name: 'Check credential only'}).click(); await settled(page);
  assert.equal(posts.length, 0); assert.equal(await page.locator('soda-tailnet input').count(), 0);
  assert(!await page.locator('soda-tailnet').innerText().then(text => text.includes('appliance.soda.ts.net')));
});

test('late reads/writes ignoring abort cannot dispatch or publish after BFCache retirement', behaviorCase, async t => {
  for (const kind of ['session', 'mutation'] as const) {
    const {page, posts} = await tailnetPage(t);
    await pause(page, kind); await page.getByRole('button', {name: 'Resume / reauthenticate'}).click();
    await page.locator(`html[data-tailnet-paused=${kind}]`).waitFor();
    await hide(page); await restore(page);
    await page.evaluate(() => window.dispatchEvent(new Event('release-tailnet'))); await frame(page);
    assert.equal(posts.length, 0); assert.equal(await page.getByRole('link', {name: 'Continue appliance sign-in at Tailscale'}).count(), 0);
    if (kind === 'mutation') assert.match(await page.locator('.tailnet-notice').innerText(), /unconfirmed/);
  }
});

test('duplicate activation is single-dispatch and post-response operator loss hides the result', behaviorCase, async t => {
  const {page, state, posts} = await tailnetPage(t);
  await page.getByRole('button', {name: 'Disconnect appliance', exact: true}).click();
  await page.getByRole('button', {name: 'Confirm logout', exact: true}).evaluate(el => {if (el instanceof HTMLButtonElement) {el.click(); el.click();}});
  await settled(page); assert.equal(posts.length, 1); assert.equal(posts[0]?.body.action, 'logout');
  await page.getByRole('button', {name: 'Refresh observations'}).click(); await settled(page);
  await pause(page, 'mutation'); await page.getByRole('button', {name: 'Sign in appliance'}).click();
  await page.locator('html[data-tailnet-paused=mutation]').waitFor(); state.operator = false;
  await page.evaluate(() => window.dispatchEvent(new Event('release-tailnet'))); await settled(page);
  assert.equal(await page.locator('soda-tailnet fieldset').count(), 0);
  assert.match(await page.locator('.tailnet-notice').innerText(), /unconfirmed/);
});

test('missing selected node, expired/approval state and read failure never claim usable endpoints', behaviorCase, async t => {
  const {page, state, posts} = await tailnetPage(t); assert(state.data.host);
  state.data.host.preferences.exit_node_id = 'missing'; state.data.host.state = 'NeedsMachineAuth'; state.data.host.expired = true;
  await page.getByRole('button', {name: 'Refresh observations'}).click(); await settled(page);
  await page.getByText('selected native exit-node ID is missing', {exact: false}).waitFor();
  await page.getByText('Device approval is required', {exact: false}).waitFor();
  await page.getByRole('button', {name: 'Apply exit node', exact: true}).click(); assert.equal(posts.length, 0);
  state.readStatus = 503; await page.getByRole('button', {name: 'Refresh observations'}).click(); await settled(page);
  assert.equal(await page.locator('soda-tailnet dl').count(), 0);
  assert(!await page.locator('soda-tailnet').innerText().then(text => text.includes('private-native-diagnostic')));
});
