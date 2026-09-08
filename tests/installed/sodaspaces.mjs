#!/usr/bin/env node
// SPDX-License-Identifier: Apache-2.0
// Opt-in real stock Forgejo/Caddy journey. Existing repository and two users;
// public by default, with an explicit read-only private-repository variant.
// Authentication-only by default; explicit access mode permits narrowly bound
// create/key/join requests. No response fakes, cookie seeding or private-key upload.
import assert from 'node:assert/strict';
import {lstat, mkdir, readFile, writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import {createRequire} from 'node:module';
import https from 'node:https';
import path from 'node:path';

let stage = 'private input validation';
let context, run, result, nativeBrowser;
let interrupted = false;
let refusedRequest = false;
let failure = false;
let accessMode = false;
process.umask(0o077);
const interrupt = async () => {
  interrupted = true;
  try { await nativeBrowser?.close(); } catch { /* Report only the safe stage below. */ }
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
  assert(permission === '--allow-auth-transitions' && (extra.length === 0 ||
    (extra.length === 1 && ['--allow-environment-access', '--private-repository'].includes(extra[0]))));
  accessMode = extra[0] === '--allow-environment-access';
  const privateRepository = extra[0] === '--private-repository';
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
    assert.deepEqual(Object.keys(user).sort(), accessMode ? ['id', 'login', 'password_file', 'public_key_file'] : ['id', 'login', 'password_file']);
    if (accessMode) assert(/^[a-z][a-z0-9_-]{0,30}$/.test(user.login) && user.login !== 'root');
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
  const publicKeys = accessMode ? await Promise.all(input.users.map(async user => {
    const key = (await privateFile(user.public_key_file, 16384)).trim();
    assert(/^ssh-ed25519 [A-Za-z0-9+/]{68}(?: [^\r\n]*)?$/.test(key));
    return key;
  })) : [];
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
  const {launchNativeBrowser} = await import('./native-browser.mjs');
  nativeBrowser = await launchNativeBrowser(chromium, run, home);
  context = nativeBrowser.context;
  assert(!interrupted);
  context.setDefaultTimeout(30000);
  let environmentReads = 0;
  let authorizations = 0;
  const writes = new Set(['/user/login', '/user/logout', '/login/oauth/grant', '/-/soda/api/session/logout']);
  let accessWrite = null; // One exact expected request, consumed before transmission.
  async function guardedPage() {
    const p = await context.newPage();
    await p.setViewportSize({width: 1280, height: 900});
    const cdp = await context.newCDPSession(p);
    // Playwright route handlers omit redirect hops. CDP Fetch pauses each hop
    // before transmission, including the authorization redirect from Soda.
    cdp.on('Fetch.requestPaused', async ({requestId, request}) => {
      try {
        const url = new URL(request.url);
        // Native logout's link action posts this fixed navigation-only form.
        const logoutRedirect = request.method === 'POST' && url.pathname === '/-/fetch-redirect' && request.postData === 'redirect=%2F';
        const expectedAccess = accessWrite && url.origin === origin.origin && !url.search &&
          request.method === 'POST' && url.pathname === accessWrite.path && request.postData === accessWrite.body &&
          Object.entries(request.headers || {}).some(([name, value]) => name.toLowerCase() === 'x-soda-expected-user-id' && value === accessWrite.actor);
        if (expectedAccess) accessWrite = null;
        const denied = url.origin !== origin.origin ||
          (!['GET', 'HEAD'].includes(request.method) && !writes.has(url.pathname) && !logoutRedirect && !expectedAccess) ||
          (url.pathname === '/login/oauth/authorize' && url.searchParams.get('client_id') !== input.oauth_client_id);
        if (denied) {
          refusedRequest = true;
          result.refused_hop = {same_origin: url.origin === origin.origin,
            method: ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'].includes(request.method) ? request.method : 'other',
            authorization: url.pathname === '/login/oauth/authorize'};
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
  const observedRoutes = new Set(['/', '/-/fetch-redirect', '/user/login', '/user/logout', '/login/oauth/authorize', '/login/oauth/grant',
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
    await p.waitForFunction(() => document.getElementById('sodaspaces-data')?.getAttribute('aria-busy') === 'false' || document.getElementById('sodaspaces-content')?.getAttribute('aria-busy') === 'false');
  }
  async function open(p = page) {
    if (!(await p.locator('#sodaspaces-drawer').isVisible())) await p.locator('#sodaspaces-button').click();
    if (await p.locator('#sodaspaces-reload').isVisible()) {
      // Closing now ends the whole drawer/terminal context. Exercise its explicit
      // full-page reload rather than remounting a stale component in the probe.
      await Promise.all([p.waitForEvent('domcontentloaded'), p.locator('#sodaspaces-reload').click()]);
      await p.locator('#sodaspaces-button').waitFor({state: 'visible'});
      if (!(await p.locator('#sodaspaces-drawer').isVisible())) await p.locator('#sodaspaces-button').click();
    }
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
    assert.equal(await p.locator('#sodaspaces-root').getAttribute('data-repository-id'), input.repository_id);
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
    stage = 'native anonymous landing after logout';
    try {
      await p.waitForFunction(() => document.readyState === 'complete' && !!document.querySelector('#navbar a[href^="/user/login"]'), null, {polling: 100});
    } catch (error) {
      result.logout_landing = await p.evaluate(() => ({atHome: location.pathname === '/', ready: document.readyState,
        focused: document.hasFocus(), hidden: document.hidden, protocol: location.protocol,
        blocked: document.body.innerText.includes('ERR_BLOCKED_BY_CLIENT'), aborted: document.body.innerText.includes('ERR_ABORTED'),
        login: !!document.querySelector('a[href*="/user/login"]'), logout: !!document.querySelector('a[data-url="/user/logout"]')}));
      throw error;
    }
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
    assert(/^(No shared environment\.|Environment (running\.|stopped\.)|Provisioning incomplete\.|Native state unavailable;)/.test(state));
    result.states.push(state.startsWith('No shared') ? 'absent' : state.startsWith('Environment running') ? 'running'
      : state.startsWith('Environment stopped') ? 'stopped' : state.startsWith('Provisioning incomplete') ? 'incomplete' : 'live-status-unavailable');
    if (await page.locator('#sodaspaces-connection').isVisible()) {
      const command = await page.locator('#sodaspaces-command').inputValue();
      const fingerprint = await page.locator('#sodaspaces-fingerprint').innerText();
      assert.match(command, /^ssh [a-z][a-z0-9_-]{0,30}@[0-9a-fA-F:.]+$/);
      assert.match(fingerprint, /^Ed25519 host-key fingerprint: SHA256:[A-Za-z0-9+/]{43}$/);
      assert.equal(await page.locator('#sodaspaces-copy').getAttribute('data-clipboard-target'), '#sodaspaces-command');
      assert(!(await page.locator('#sodaspaces-copy').isDisabled()));
      (result.own_connections ||= []).push({user_id: input.users[index].id, repository_id: input.repository_id, command, fingerprint});
    }
    const cookies = await context.cookies(origin.origin + '/-/soda/api/session');
    const soda = cookies.find(cookie => cookie.name === '__Secure-sodaspaces-session');
    assert(soda && soda.domain === origin.hostname && soda.path === '/-/soda/' && soda.secure && soda.httpOnly && soda.sameSite === 'Lax');
    assert(!(await context.cookies(repoURL)).some(cookie => cookie.name === '__Secure-sodaspaces-session'));
  }

  stage = 'anonymous and native-cookie-only contexts';
  const anonymous = await page.goto(repoURL);
  if (privateRepository) {
    // Stock 15.0.7 denies anonymous private repository access before supplying
    // template repository context. Never make a retained repository public to
    // satisfy the public-fixture journey, or treat denial as Soda absence.
    assert.equal(anonymous.status(), 404);
    assert(!(await page.locator('#sodaspaces-button').isVisible()));
    assert.equal(environmentReads, 0);
    result.anonymous_repository = 'native 404; no Soda repository reads';
  } else {
    assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-signed'), 'false');
    await open();
    assert.equal(environmentReads, 0);
    await page.locator('#sodaspaces-sign-in').waitFor({state: 'visible'});
  }
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
      chrome: document.activeElement === document.body,
      stale: !document.getElementById('sodaspaces-reload').hidden,
      cleared: document.getElementById('sodaspaces-actor').textContent === ''}));
    // Chromium's chrome/body focus transition can briefly report hasFocus true.
    // Require actual blur clearing there, and never allow background form focus.
    assert(focus.inside || focus.chrome);
    if (focus.chrome) { chromeFocus = true; assert(focus.stale && focus.cleared); }
  }
  if (await page.evaluate(() => document.activeElement === document.body)) await page.keyboard.press('Tab');
  assert(await drawer.evaluate(node => node.contains(document.activeElement)));
  stage = 'Escape and asynchronous focus return';
  await page.keyboard.press('Escape');
  await drawer.waitFor({state: 'hidden'});
  await page.waitForFunction(() => document.activeElement?.id === 'sodaspaces-button');
  await open();
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
  assert.match(await page.locator('#sodaspaces-status').innerText(), /identities differ/);
  assert.equal(environmentReads, beforeSwitch);
  stage = 'second real OAuth repository return';
  await oauth(1);

  stage = 'Soda-only logout in another tab';
  await other.bringToFront();
  await other.goto(repoURL);
  await open(other);
  await other.locator('#sodaspaces-sign-out').click();
  await other.waitForFunction(() => document.getElementById('sodaspaces-status')?.textContent.includes('Signed out of Soda, not Forgejo or Linux'));
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
  await page.goBack({waitUntil: 'commit'});
  result.bfcache_restored = await page.evaluate(() => window.sodaspacesProbeRestored === true);
  if (result.bfcache_restored) {
    // An open dialog is retained by BFCache, but its private data must not be.
    await page.locator('#sodaspaces-reload').waitFor({state: 'visible'});
    assert.equal(await page.locator('#sodaspaces-actor').innerText(), '');
  }
  if (accessMode) {
    assert(result.bfcache_restored, 'Access writes require the read-only journey to complete first');
    await other.close();
    await page.goto(repoURL);
    await open();
    stage = 'access fixture must initially have no reservation';
    assert.match(await page.locator('#sodaspaces-status').innerText(), /^No shared environment\./);
    result.access = {reservation_id: null, users: [], copy_native_paste: []};
    const permit = (index, route, body) => {
      assert(!accessWrite && !interrupted && !refusedRequest);
      accessWrite = {actor: input.users[index].id, path: '/-/soda' + route, body: JSON.stringify(body)};
    };
    stage = 'nonowner native create denial';
    permit(1, '/api/environments', {repository_id: input.repository_id});
    const denial = await page.evaluate(async ({actor, repository}) => {
      const s = await (await fetch('/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error'})).json();
      if (s.user.id !== actor) throw new Error('actor mismatch');
      const response = await fetch('/-/soda/api/environments', {method: 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
        headers: {'Content-Type': 'application/json', 'X-Soda-Expected-User-ID': actor, 'X-CSRF-Token': s.csrf_token}, body: JSON.stringify({repository_id: repository})});
      return {status: response.status, code: (await response.json()).error.code};
    }, {actor: input.users[1].id, repository: input.repository_id});
    assert.deepEqual(denial, {status: 403, code: 'owner_required'});
    assert.equal(accessWrite, null);
    result.access.nonowner_create = 'denied';
    for (const index of [0, 1]) {
      stage = `access user ${index}: native login and OAuth`;
      if (await drawer.isVisible()) await page.locator('#sodaspaces-close').click();
      await nativeLogout(page);
      await nativeLogin(page, index);
      await open();
      await oauth(index);
      if (index === 0) {
        stage = 'explicit native shared-environment creation';
        permit(index, '/api/environments', {repository_id: input.repository_id});
        const completed = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/environments' && r.request().method() === 'POST', {timeout: 300000});
        await page.locator('#sodaspaces-create').click();
        const response = await completed;
        const created = await response.json();
        const id = created.id || created.environment?.id;
        if (typeof id === 'string' && /^p[0-9a-f]{24}$/.test(id)) result.access.reservation_id = id;
        assert.equal(response.status(), 201);
        assert.equal(created.repository_id, input.repository_id);
        assert(result.access.reservation_id && created.provisioned === true && accessWrite === null);
        await settled();
        assert.equal(await page.locator('#sodaspaces-login').innerText(), '');
        assert(await page.locator('#sodaspaces-connection').isHidden());
      }
      stage = `access user ${index}: explicit public key registration`;
      await page.locator('#sodaspaces-public-key').fill(publicKeys[index]);
      permit(index, '/api/me/development-keys', {public_key: publicKeys[index]});
      const saved = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/me/development-keys' && r.request().method() === 'POST');
      await page.locator('#sodaspaces-save-key').click();
      assert.equal((await saved).status(), 200);
      await settled();
      assert.equal(accessWrite, null);
      assert.equal(await page.locator('#sodaspaces-login').innerText(), '');
      stage = `access user ${index}: explicit native account join`;
      permit(index, `/api/environments/${result.access.reservation_id}/join`, {});
      const joined = page.waitForResponse(r => new URL(r.url()).pathname === `/-/soda/api/environments/${result.access.reservation_id}/join` && r.request().method() === 'POST', {timeout: 300000});
      await page.locator('#sodaspaces-join').click();
      const joinedResponse = await joined;
      assert.equal(joinedResponse.status(), 200);
      assert.equal((await joinedResponse.json()).login, input.users[index].login);
      await settled();
      assert.equal(accessWrite, null);
      await page.locator('#sodaspaces-connection').waitFor({state: 'visible'});
      const command = await page.locator('#sodaspaces-command').inputValue();
      stage = `access user ${index}: current own connection`;
      const own = await page.evaluate(async ({actor, id}) => {
        const response = await fetch(`/-/soda/api/environments/${id}/connection`, {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': actor}});
        if (response.status !== 200) throw new Error('connection unavailable');
        return response.json(); // Public connection fields only; no bootstrap credentials.
      }, {actor: input.users[index].id, id: result.access.reservation_id});
      assert.equal(own.login, input.users[index].login);
      assert.equal(own.connection.environment.id, result.access.reservation_id);
      assert.equal(own.connection.environment.running, true);
      assert.equal(own.routing_verified, false);
      assert.equal(command, `ssh ${own.login}@${own.connection.environment.ip}`);
      assert.match(own.connection.host_key.trim(), /^ssh-ed25519 [A-Za-z0-9+/]{68}$/);
      assert.match(own.connection.fingerprint, /^SHA256:[A-Za-z0-9+/]{43}$/);
      result.access.users.push({id: input.users[index].id, ...own});
      stage = `access user ${index}: native Copy and real paste`;
      const copied = await page.evaluate(() => window.config.i18n.copy_success);
      assert(typeof copied === 'string' && copied.length > 0);
      await page.locator('#sodaspaces-copy').click();
      await page.getByRole('tooltip').filter({hasText: copied}).waitFor({state: 'visible'});
      await page.locator('#sodaspaces-close').click();
      await page.goto(repoURL + '/issues/new');
      await page.locator('#issue_title').focus();
      await page.keyboard.press('Control+V');
      assert.equal(await page.locator('#issue_title').inputValue(), command);
      result.access.copy_native_paste.push(input.users[index].id);
      // Discard only the command pasted by this run; never submit the issue.
      page.on('dialog', discardFixture);
      try { await page.goto(repoURL); } finally { page.off('dialog', discardFixture); }
      await open();
      assert.equal(await page.locator('#sodaspaces-command').inputValue(), command);
      assert(await page.locator('#sodaspaces-join').isHidden());
    }
    result.access.native_join_confirmed = true;
  }
  assert(!refusedRequest && !interrupted);
} catch (error) {
  if (result) result.failure_kind = error?.name === 'TimeoutError' ? 'timeout'
    : error?.name === 'AssertionError' ? 'assertion'
      : error?.message?.includes('strict mode violation') ? 'ambiguous_locator' : 'other';
  failure = true; // Never print Playwright errors/URLs/input bodies or credentials.
} finally {
  try { await nativeBrowser?.close(); } catch { failure = true; }
  failure ||= interrupted || refusedRequest;
  const outcome = failure ? 'failed' : result?.bfcache_restored ? 'passed-scoped-journey' : 'incomplete-bfcache-not-observed';
  if (run) {
    try {
      await writeFile(path.join(run, 'result.json'), JSON.stringify({...result, outcome, stage, environment_mutations: accessMode ? 'explicit single-use actor/path/body-bound create/key/join requests only' : 'not permitted; unexpected writes aborted', provisioning_ssh_proof: false}, null, 2) + '\n', {flag: 'wx', mode: 0o600});
    } catch { failure = true; }
  }
  console.log(failure ? `Sodaspaces journey failed at ${stage}; private profile/evidence retained if created.` : `Sodaspaces journey: ${outcome}; not whole-product or backend-artifact acceptance.`);
  process.exitCode = failure ? 1 : result?.bfcache_restored ? 0 : 2;
}
