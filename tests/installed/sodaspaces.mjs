#!/usr/bin/env node
// SPDX-License-Identifier: Apache-2.0
// Opt-in real stock Forgejo/Caddy journey. Existing public repository and two users;
// authentication mutations only. No response fakes, cookie seeding or provisioning.
import assert from 'node:assert/strict';
import {lstat, mkdir, readFile, writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {createRequire} from 'node:module';
import https from 'node:https';
import path from 'node:path';

let stage = 'private input validation';
let context, run, result;
let interrupted = false;
let refusedRequest = false;
let failure = false;
process.umask(0o077);
const interrupt = async () => {
  interrupted = true;
  try { await context?.close(); } catch { /* Report only the safe stage below. */ }
};
process.once('SIGINT', interrupt);
process.once('SIGTERM', interrupt);
const validID = value => typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) &&
  (value.length < 19 || value <= '9223372036854775807');
async function privateFile(file, limit) {
  assert(path.isAbsolute(file));
  const info = await lstat(file);
  assert(info.isFile() && !(info.mode & 0o077) && info.size <= limit);
  return readFile(file, 'utf8');
}
try {
  const [inputFile, home, permission, ...extra] = process.argv.slice(2);
  assert(permission === '--allow-auth-transitions' && extra.length === 0);
  const input = JSON.parse(await privateFile(inputFile, 16384));
  assert.deepEqual(Object.keys(input).sort(), ['ca_file', 'oauth_client_id', 'origin', 'repository_id', 'repository_path', 'revision', 'target', 'users']);
  const origin = new URL(input.origin);
  assert(origin.protocol === 'https:' && origin.pathname === '/' && !origin.username && !origin.password && !origin.search && !origin.hash);
  assert(typeof input.target === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,252}$/.test(input.target) && process.env.SODA_NATIVE_VALIDATE === input.target);
  assert(/^[0-9a-f]{40}$/.test(input.revision));
  assert(validID(input.repository_id));
  assert(/^\/[A-Za-z0-9][A-Za-z0-9_.-]*\/[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(input.repository_path));
  assert(typeof input.oauth_client_id === 'string' && input.oauth_client_id.length > 0 && input.oauth_client_id.length <= 256);
  assert(Array.isArray(input.users) && input.users.length === 2);
  for (const user of input.users) {
    assert.deepEqual(Object.keys(user).sort(), ['id', 'login', 'password_file']);
    assert(validID(user.id) && /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(user.login));
  }
  assert(input.users[0].id !== input.users[1].id && input.users[0].login !== input.users[1].login);
  assert(path.isAbsolute(home));
  const homeInfo = await lstat(home);
  assert(homeInfo.isDirectory() && !(homeInfo.mode & 0o077));
  const ca = await privateFile(input.ca_file, 65536);
  const passwords = await Promise.all(input.users.map(async user => {
    const password = (await privateFile(user.password_file, 4096)).replace(/\r?\n$/, '');
    assert(password && !/[\r\n]/.test(password));
    return password;
  }));
  const root = new URL('../../', import.meta.url);
  assert.equal(execFileSync('git', ['rev-parse', 'HEAD'], {cwd: root, encoding: 'utf8'}).trim(), input.revision);
  assert.equal(execFileSync('git', ['status', '--porcelain'], {cwd: root, encoding: 'utf8'}).trim(), '');
  assert(!interrupted);
  const newRun = path.join(home, 'sodaspaces-run');
  await mkdir(newRun, {mode: 0o700}); // Exclusive; never finalize an occupied attempt.
  run = newRun;
  result = {revision: input.revision, target: input.target, backend_revision: 'requires separate installed artifact verification', states: [], bfcache_restored: false};

  // Raw paths must not be normalized by URL/browser clients before reaching Caddy.
  async function rawRead(rawPath, headers = {}) {
    assert(!interrupted);
    return new Promise((resolve, reject) => {
      const request = https.request({hostname: origin.hostname, port: origin.port || 443,
        path: rawPath, method: 'GET', headers, ca, timeout: 15000, signal: AbortSignal.timeout(15000)}, response => {
        let size = 0;
        const chunks = [];
        response.on('data', chunk => {
          size += chunk.length;
          if (size > 65536) response.destroy(new Error('response limit'));
          else chunks.push(chunk);
        });
        response.on('error', reject);
        response.on('end', () => resolve({status: response.statusCode, headers: response.headers, body: Buffer.concat(chunks)}));
      });
      request.on('timeout', () => request.destroy(new Error('request timeout')));
      request.on('error', reject);
      request.end();
    });
  }
  stage = 'trusted proxy and asset paths';
  assert.equal((await rawRead('/-/soda/api/session')).status, 401);
  for (const route of ['/api/session', '/-/soda', '/-/soda/healthz', '/-/soda/api//session', '/-/soda/api/../api/session', '/-/soda/%61pi/session', '/-/soda/api/session/']) {
    assert.equal((await rawRead(route)).status, 404);
  }
  const version = await rawRead('/api/v1/version');
  assert.equal(version.status, 200);
  // Stock Makefile can include its Gitea compatibility suffix in setting.AppVer.
  assert.match(JSON.parse(version.body).version, /^15\.0\.7(?:\+gitea-1\.22\.0)?$/);
  for (const file of ['sodaspaces.css', 'sodaspaces.js']) {
    const response = await rawRead(`/assets/${file}`);
    assert.equal(response.status, 200);
    assert.match(response.headers['cache-control'] || '', /max-age=0/);
    assert.match(response.headers['cache-control'] || '', /must-revalidate/);
    const expected = await readFile(new URL(`../../appliance/forgejo/public/assets/${file}`, import.meta.url));
    assert.equal(createHash('sha256').update(response.body).digest('hex'), createHash('sha256').update(expected).digest('hex'));
    const validator = response.headers.etag
      ? {'If-None-Match': response.headers.etag}
      : {'If-Modified-Since': response.headers['last-modified']};
    assert(Object.values(validator)[0]);
    assert.equal((await rawRead(`/assets/${file}`, validator)).status, 304);
  }
  result.asset_revalidation = 'conditional reads of current bytes; not an update/cutover proof';

  stage = 'trusted browser startup';
  const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
  const {chromium} = require('playwright');
  assert(!interrupted);
  context = await chromium.launchPersistentContext(path.join(run, 'profile'), {
    headless: true, chromiumSandbox: true,
    // Inspected Playwright 1.63.0 otherwise disables the behavior being tested.
    ignoreDefaultArgs: ['--disable-back-forward-cache'],
    env: {...process.env, HOME: home, XDG_CONFIG_HOME: path.join(home, '.config'), XDG_DATA_HOME: path.join(home, '.local/share')},
    viewport: {width: 1280, height: 900},
  });
  assert(!interrupted);
  context.setDefaultTimeout(30000);
  let environmentReads = 0;
  let authorizations = 0;
  const writes = new Set(['/user/login', '/user/logout', '/login/oauth/grant', '/-/soda/api/session/logout']);
  async function guardedPage() {
    const p = await context.newPage();
    const cdp = await context.newCDPSession(p);
    // Playwright route handlers omit redirect hops. CDP Fetch pauses each hop
    // before transmission, including the authorization redirect from Soda.
    cdp.on('Fetch.requestPaused', async ({requestId, request}) => {
      try {
        const url = new URL(request.url);
        const denied = url.origin !== origin.origin ||
          (!['GET', 'HEAD'].includes(request.method) && !writes.has(url.pathname)) ||
          (url.pathname === '/login/oauth/authorize' && url.searchParams.get('client_id') !== input.oauth_client_id);
        if (denied) {
          refusedRequest = true;
          await cdp.send('Fetch.failRequest', {requestId, errorReason: 'BlockedByClient'});
          return;
        }
        if (url.pathname === '/login/oauth/authorize') authorizations++;
        if (url.pathname.startsWith('/-/soda/api/environments')) environmentReads++;
        await cdp.send('Fetch.continueRequest', {requestId}); // No response substitution.
      } catch {
        if (!interrupted) refusedRequest = true;
      }
    });
    await cdp.send('Fetch.enable', {patterns: [{urlPattern: '*', requestStage: 'Request'}]});
    return p;
  }
  await context.addInitScript(() => {
    window.addEventListener('pageshow', event => { window.sodaspacesProbeRestored = event.persisted; });
  });
  // Retain only fixed route labels/status codes, never URLs, queries or bodies.
  result.http = [];
  const observedRoutes = new Set(['/user/login', '/login/oauth/authorize', '/login/oauth/grant',
    '/-/soda/login', '/-/soda/oauth/callback', '/-/soda/api/session', '/-/soda/api/forgejo/me', '/-/soda/api/environments']);
  context.on('response', response => {
    const pathname = new URL(response.url()).pathname;
    if (observedRoutes.has(pathname) && result.http.length < 128) result.http.push({route: pathname, status: response.status()});
  });
  const page = await guardedPage();
  const repoURL = origin.origin + input.repository_path;
  const drawer = page.locator('#sodaspaces-drawer');
  async function settled(p = page) {
    await p.locator('#sodaspaces-drawer').waitFor({state: 'visible'});
    // An unauthenticated/empty data region has zero height in native CSS.
    // Completion is its ARIA state, not whether that empty region has a box.
    await p.waitForFunction(() => document.getElementById('sodaspaces-data')?.getAttribute('aria-busy') === 'false');
  }
  async function open(p = page) {
    await p.locator('#sodaspaces-button').click();
    await settled(p);
  }
  async function nativeLogin(p, index) {
    assert(!interrupted);
    await p.goto(origin.origin + '/user/login');
    await p.locator('#user_name').fill(input.users[index].login);
    await p.locator('#password').fill(passwords[index]);
    assert(!interrupted);
    await Promise.all([
      p.waitForURL(url => url.pathname !== '/user/login'),
      p.locator('form:has(#user_name) button').click(),
    ]);
    await p.goto(repoURL);
    assert.equal(await p.locator('#sodaspaces-root').getAttribute('data-user-id'), input.users[index].id);
  }
  async function nativeLogout(p) {
    const menu = p.locator('details').filter({has: p.locator('a[data-url="/user/logout"]')});
    await menu.locator('summary').click();
    assert(!interrupted);
    const response = p.waitForResponse(res => new URL(res.url()).pathname === '/user/logout' && res.request().method() === 'POST');
    // Native SSE and the link action both navigate on logout. Do not attach a
    // click navigation waiter to one of those competing native navigations.
    await menu.locator('a[data-url="/user/logout"]').click({noWaitAfter: true});
    assert.equal((await response).status(), 200);
    await p.waitForFunction(() => document.readyState === 'complete' && !!document.querySelector('a[href="/user/login"]'));
  }
  async function oauth(index) {
    assert(!interrupted);
    const before = authorizations;
    const journey = stage;
    stage = journey + ': authorization navigation';
    await page.locator('#sodaspaces-sign-in').click();
    await Promise.race([
      page.waitForURL(url => url.pathname === input.repository_path && url.hash === '#sodaspaces'),
      page.locator('#authorize-app').waitFor({state: 'visible'}),
    ]);
    stage = journey + ': native consent';
    if (await page.locator('#authorize-app').isVisible()) {
      assert.equal(await page.locator('input[name="client_id"]').inputValue(), input.oauth_client_id);
      assert(!interrupted);
      await page.locator('#authorize-app').click();
    }
    stage = journey + ': safe repository return';
    await page.waitForURL(url => url.origin === origin.origin && url.pathname === input.repository_path && url.hash === '#sodaspaces');
    stage = journey + ': drawer identity and state';
    await settled();
    stage = journey + ': observed authorization';
    assert(authorizations > before);
    stage = journey + ': native actor';
    assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-user-id'), input.users[index].id);
    stage = journey + ': displayed Soda actor';
    assert.match(await page.locator('#sodaspaces-actor').innerText(), new RegExp(`ID ${input.users[index].id}\\)`));
    stage = journey + ': environment state';
    const state = await page.locator('#sodaspaces-status').innerText();
    assert(/^(No shared environment\.|Environment (running\.|stopped\.|provisioning is incomplete\.|reserved;))/.test(state));
    result.states.push(state.startsWith('No shared') ? 'absent' : state.startsWith('Environment running') ? 'running'
      : state.startsWith('Environment stopped') ? 'stopped' : state.startsWith('Environment provisioning') ? 'incomplete' : 'live-status-unavailable');
    const cookies = await context.cookies(origin.origin + '/-/soda/api/session');
    const soda = cookies.find(cookie => cookie.name === '__Secure-sodaspaces-session');
    assert(soda && soda.domain === origin.hostname && soda.path === '/-/soda/' && soda.secure && soda.httpOnly && soda.sameSite === 'Lax');
    assert(!(await context.cookies(repoURL)).some(cookie => cookie.name === '__Secure-sodaspaces-session'));
  }

  stage = 'anonymous and native-cookie-only contexts';
  await page.goto(repoURL);
  assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-signed'), 'false');
  await open();
  assert.equal(environmentReads, 0);
  await page.locator('#sodaspaces-sign-in').waitFor({state: 'visible'});
  await nativeLogin(page, 0);
  await open();
  assert.equal(environmentReads, 0);
  assert(!(await context.cookies(origin.origin + '/-/soda/')).some(cookie => cookie.name === '__Secure-sodaspaces-session'));
  stage = 'first real OAuth repository return';
  await oauth(0);

  stage = 'protected logout negative guards';
  const guards = await page.evaluate(async other => {
    const base = '/-/soda/api/session';
    const session = await (await fetch(base, {credentials: 'same-origin', cache: 'no-store', signal: AbortSignal.timeout(15000)})).json();
    const outcomes = [];
    for (const [actor, csrf] of [[other, session.csrf_token], [session.user.id, 'intentionally-invalid']]) {
      const response = await fetch(base + '/logout', {method: 'POST', credentials: 'same-origin', signal: AbortSignal.timeout(15000),
        headers: {'Content-Type': 'application/json', 'X-Soda-Expected-User-ID': actor, 'X-CSRF-Token': csrf}, body: '{}'});
      outcomes.push({status: response.status, code: (await response.json()).error.code});
    }
    return outcomes; // No credentials leave the browser evaluation.
  }, input.users[1].id);
  assert.deepEqual(guards, [{status: 403, code: 'identity_mismatch'}, {status: 403, code: 'invalid_csrf'}]);

  stage = 'native keyboard focus and browser chrome';
  let chromeFocus = false;
  for (let i = 0; i < 8; i++) {
    await page.keyboard.press('Tab');
    const focus = await drawer.evaluate(node => ({inside: node.contains(document.activeElement),
      chrome: !document.hasFocus() && document.activeElement === document.body,
      stale: !document.getElementById('sodaspaces-reload').hidden,
      cleared: document.getElementById('sodaspaces-actor').textContent === ''}));
    // Native Chromium permits browser-chrome focus, never background form focus.
    assert(focus.inside || focus.chrome);
    if (focus.chrome) { chromeFocus = true; assert(focus.stale && focus.cleared); }
  }
  if (!await page.evaluate(() => document.hasFocus())) await page.keyboard.press('Tab');
  assert(await drawer.evaluate(node => node.contains(document.activeElement)));
  stage = 'Escape and asynchronous focus return';
  await page.keyboard.press('Escape');
  await drawer.waitFor({state: 'hidden'});
  await page.waitForFunction(() => document.activeElement?.id === 'sodaspaces-button');
  await open();
  if (chromeFocus) {
    await page.locator('#sodaspaces-reload').click();
    await settled();
  }
  result.keyboard_chrome_invalidation = chromeFocus;
  stage = 'backdrop and automatic theme selection';
  await page.mouse.click(10, 400);
  await drawer.waitFor({state: 'hidden'});
  await page.waitForFunction(() => document.activeElement?.id === 'sodaspaces-button');
  assert.match(await page.locator('html').getAttribute('data-theme'), /auto/);
  const backgrounds = [];
  for (const [width, colorScheme] of [[360, 'light'], [1280, 'dark']]) {
    stage = `native ${width}-pixel ${colorScheme} layout`;
    await page.setViewportSize({width, height: 900});
    await page.emulateMedia({colorScheme});
    await open();
    const box = await drawer.boundingBox();
    assert(box && box.x >= -1 && box.width <= width + 1 && Math.abs(box.x + box.width - width) < 2 && box.height >= 898);
    backgrounds.push(await drawer.evaluate(node => getComputedStyle(node).backgroundColor));
    await page.locator('#sodaspaces-close').click();
  }
  assert.notEqual(backgrounds[0], backgrounds[1]);

  stage = 'native-only account switch and stale tab';
  await open();
  const beforeSwitch = environmentReads;
  stage = 'second guarded native page';
  const other = await guardedPage();
  await other.goto(repoURL);
  await other.bringToFront();
  await page.locator('#sodaspaces-reload').waitFor({state: 'visible'});
  assert.equal(await page.locator('#sodaspaces-actor').innerText(), '');
  assert.equal(environmentReads, beforeSwitch);
  stage = 'native logout in second page';
  await nativeLogout(other);
  stage = 'native second-account login';
  await nativeLogin(other, 1);
  stage = 'return to stale native page';
  await page.bringToFront();
  assert.equal(environmentReads, beforeSwitch);
  // Forgejo's native logout broadcast may already have navigated this tab home.
  // Do not suppress its worker/events or pretend it retained the old document.
  result.native_logout_navigation = new URL(page.url()).pathname !== input.repository_path;
  if (result.native_logout_navigation) await page.goto(repoURL + '#sodaspaces');
  else await page.locator('#sodaspaces-reload').click();
  await settled();
  assert.match(await page.locator('#sodaspaces-status').innerText(), /identities do not match/);
  assert.equal(environmentReads, beforeSwitch);
  stage = 'second real OAuth repository return';
  await oauth(1);

  stage = 'Soda-only logout in another tab';
  await other.bringToFront();
  await other.goto(repoURL);
  await open(other);
  await other.locator('#sodaspaces-sign-out').click();
  await other.waitForFunction(() => document.getElementById('sodaspaces-status')?.textContent.includes('Signed out of Soda only'));
  await page.bringToFront();
  await page.locator('#sodaspaces-reload').click();
  await page.locator('#sodaspaces-sign-in').waitFor({state: 'visible'});
  assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-user-id'), input.users[1].id);
  await oauth(1);

  stage = 'native unsaved form coexistence';
  await page.goto(repoURL + '/issues/new');
  await page.locator('#issue_title').fill('Unsaved Sodaspaces fixture text');
  await open();
  await other.bringToFront();
  await page.bringToFront();
  await page.locator('#sodaspaces-reload').waitFor({state: 'visible'});
  assert.equal(await page.locator('#issue_title').inputValue(), 'Unsaved Sodaspaces fixture text');
  assert.equal(new URL(page.url()).pathname, input.repository_path + '/issues/new');
  await page.locator('#sodaspaces-close').click();
  assert.equal(await page.locator('#issue_title').inputValue(), 'Unsaved Sodaspaces fixture text');
  stage = 'discard only synthetic form text';
  const discardFixture = async dialog => {
    if (dialog.type() === 'beforeunload') await dialog.accept();
    else { refusedRequest = true; await dialog.dismiss(); }
  };
  page.on('dialog', discardFixture);
  try { await page.goto(repoURL); } finally { page.off('dialog', discardFixture); }

  stage = 'real back-forward restoration';
  await open();
  await page.goto(origin.origin + '/');
  await page.goBack();
  result.bfcache_restored = await page.evaluate(() => window.sodaspacesProbeRestored === true);
  if (result.bfcache_restored) {
    // An open dialog is retained by BFCache, but its private data must not be.
    await page.locator('#sodaspaces-reload').waitFor({state: 'visible'});
    assert.equal(await page.locator('#sodaspaces-actor').innerText(), '');
  }
  assert(!refusedRequest && !interrupted);
} catch (error) {
  if (result) result.failure_kind = error?.name === 'TimeoutError' ? 'timeout'
    : error?.name === 'AssertionError' ? 'assertion'
      : error?.message?.includes('strict mode violation') ? 'ambiguous_locator' : 'other';
  failure = true; // Never print Playwright errors/URLs/input bodies or credentials.
} finally {
  try { await context?.close(); } catch { failure = true; }
  failure ||= interrupted || refusedRequest;
  const outcome = failure ? 'failed' : result?.bfcache_restored ? 'passed-scoped-journey' : 'incomplete-bfcache-not-observed';
  if (run) {
    try {
      await writeFile(path.join(run, 'result.json'), JSON.stringify({...result, outcome, stage, environment_mutations: 'not permitted; unexpected writes aborted', provisioning_ssh_proof: false}, null, 2) + '\n', {flag: 'wx', mode: 0o600});
    } catch { failure = true; }
  }
  console.log(failure ? `Sodaspaces journey failed at ${stage}; private profile/evidence retained if created.` : `Sodaspaces journey: ${outcome}; not whole-product or backend-artifact acceptance.`);
  process.exitCode = failure ? 1 : result?.bfcache_restored ? 0 : 2;
}
