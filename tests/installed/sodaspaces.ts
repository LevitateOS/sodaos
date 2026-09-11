#!/usr/bin/env bun
import {loadRunnerInput, exerciseRunners, type RunnerEvidence} from './runners.ts';
import {chromium, type Page, type BrowserContext, type Dialog, type WebSocket} from 'playwright';
import {journeyInput, managementInput, object} from './sodaspaces-input.ts';
import type {ManagementEvidence} from './sodaspaces-management.ts';
import {projectView, newManagedTerminal, terminalMenu} from './sodaspaces-controls.ts';
import {matrixInput} from './sodaspaces-matrix-input.ts';
import {exerciseWorkspaceMatrix} from './sodaspaces-workspace-journey.ts';
import {observeMatrixShell, inspectMatrixProcess} from './sodaspaces-matrix-native.ts';
import {exerciseSelectedCLIs} from './sodaspaces-cli.ts';
import {terminalID} from '../../frontend/spaces/sodaspaces-api.ts';
import forgejoPayload from '../../internal/nativebuild/forgejo-payload.json';
import type {launchNativeBrowser} from './native-browser.ts';
// SPDX-License-Identifier: Apache-2.0
// Opt-in real stock Forgejo/Caddy journey. Existing repository and two users;
// public by default, with an explicit read-only private-repository variant.
// Authentication-only by default; explicit access mode permits narrowly bound
// create/key/join requests. No response fakes, cookie seeding or private-key upload.
import assert from 'node:assert/strict';
import {lstat, mkdir, writeFile} from 'node:fs/promises';
import {readProbeHTTPS} from './sodaspaces-http';
import path from 'node:path';

let stage = 'private input validation';
interface ProbeEvidence extends Record<string, unknown> {
  states: string[]; bfcache_restored: boolean; http: Array<{route: string; status: number}>;
  own_connections?: Array<{user_id: string; repository_id: string; command: string; fingerprint: string}>;
}
interface AccessEvidence {reservation_id: string | null; users: unknown[]; copy_native_paste: string[]; nonowner_create?: string; native_join_confirmed?: boolean}
interface ShellFacts {login: string; uid: number; gid: number; groups: number[]; home: string; cwd: string; tty: boolean; shell_pid: number; shell_start: string}
declare global { interface Window { sodaspacesProbeRestored?: boolean; config: {i18n: {copy_success: string}} } }
let run: string | undefined, evidence: ProbeEvidence | undefined, nativeBrowser: Awaited<ReturnType<typeof launchNativeBrowser>> | undefined;
let interrupted = false;
let refusedRequest = false;
let failure = false;
let accessMode = false;
let terminalMode = false;
let managementMode = false;
let matrixMode = false;
let runnerMode = false;
let runnerConfirmed = false;
process.umask(0o077);
const interrupt = async () => {
  interrupted = true;
  try { await nativeBrowser?.close(); } catch { /* Report only the safe stage below. */ }
};
process.once('SIGINT', interrupt);
process.once('SIGTERM', interrupt);
async function privateFile(file: string, limit: number) {
  assert(path.isAbsolute(file));
  const info = await lstat(file);
  assert(info.isFile() && !(info.mode & 0o077) && info.size <= limit);
  return Bun.file(file).text();
}
try {
  const [inputFile, home, permission, ...extra] = Bun.argv.slice(2);
  assert(permission === '--allow-auth-transitions' && (extra.length === 0 ||
    (extra.length === 1 && ['--allow-environment-access', '--private-repository', '--allow-existing-terminal'].includes(extra[0] ?? '')) ||
    (extra.length === 2 && ['--allow-existing-management', '--allow-workspace-matrix'].includes(extra[0] || '')) ||
    (extra.length === 3 && extra[0] === '--runner-phase' && /^--allow-runner-(list|register|start|stop|restart|remove|dispatch|job|overlap|departure)$/.test(extra[2] || ''))));
  runnerMode = extra[0] === '--runner-phase';
  managementMode = extra[0] === '--allow-existing-management';
  matrixMode = extra[0] === '--allow-workspace-matrix';
  accessMode = extra[0] === '--allow-environment-access';
  terminalMode = extra[0] === '--allow-existing-terminal' || managementMode || matrixMode;
  const privateRepository = extra[0] === '--private-repository';
  assert(inputFile && home);
  // Runner inputs are checked before proxy, authentication or native effects.
  const runnerRequest = runnerMode && extra[1] && extra[2] ? await loadRunnerInput(extra[1],extra[2]) : null;
  assert(!runnerMode || runnerRequest);
  const input = journeyInput(JSON.parse(await privateFile(inputFile, 16384)), accessMode, terminalMode);
  if (runnerRequest) {
    assert(input.target === runnerRequest.target && input.origin === runnerRequest.origin && input.revision === runnerRequest.revision && input.ca_file === runnerRequest.ca_file);
    assert(input.users[0]?.id === runnerRequest.operator_id && input.users[1]?.id === runnerRequest.denied_id, 'Declare operator then denied actor in separate authenticated contexts');
  }
  assert(!managementMode || extra[1]);
  const managementRequest = managementMode && extra[1] ? managementInput(JSON.parse(await privateFile(extra[1], 16384))) : null;
  const matrixRequest = matrixMode && extra[1] ? matrixInput(JSON.parse(await privateFile(extra[1], 16384)), input) : null;
  if (matrixRequest) {
    await privateFile(matrixRequest.ssh_config, 16384);
    for (const scenario of matrixRequest.cli) {
      const prompt = (await privateFile(scenario.prompt_file, 4096)).trim();
      assert(prompt && !/[\x00-\x08\x0b-\x1f\x7f]/.test(prompt) && !prompt.includes(scenario.expected_text));
    }
  }
  const origin = new URL(input.origin);
  assert(origin.protocol === 'https:' && origin.pathname === '/' && !origin.username && !origin.password && !origin.search && !origin.hash);
  assert(process.env.SODA_NATIVE_VALIDATE === input.target);
  assert(path.isAbsolute(home));
  const homeInfo = await lstat(home);
  assert(homeInfo.isDirectory() && !(homeInfo.mode & 0o077));
  await privateFile(input.ca_file, 65536);
  const passwords = await Promise.all(input.users.map(async user => {
    const password = (await privateFile(user.password_file, 4096)).replace(/\r?\n$/, '');
    assert(password && !/[\r\n]/.test(password));
    return password;
  }));
  const publicKeys = accessMode ? await Promise.all(input.users.map(async user => {
    assert(user.public_key_file);
    const key = (await privateFile(user.public_key_file, 16384)).trim();
    assert(/^ssh-ed25519 [A-Za-z0-9+/]{68}(?: [^\r\n]*)?$/.test(key));
    return key;
  })) : [];
  const root = path.resolve(import.meta.dir, '../..');
  const git = (...args: string[]) => {const child = Bun.spawnSync(['git', ...args], {cwd: root, stdout: 'pipe', stderr: 'pipe'}); assert(child.exitCode === 0, 'Git observation failed'); return child.stdout.toString().trim();};
  assert.equal(git('rev-parse', 'HEAD'), input.revision);
  assert.equal(git('status', '--porcelain'), '');
  assert(!interrupted);
  const newRun = path.join(home, 'sodaspaces-run');
  await mkdir(newRun, {mode: 0o700}); // Exclusive; never finalize an occupied attempt.
  run = newRun;
  const result: ProbeEvidence = {http: [], revision: input.revision, target: input.target, backend_revision: 'requires separate installed artifact verification', states: [], bfcache_restored: false};
  evidence = result;

  // Raw paths must not be normalized by URL/browser clients before reaching Caddy.
  async function rawRead(rawPath: string, headers: Record<string, string> = {}) {
    assert(!interrupted);
    return readProbeHTTPS(origin, input.ca_file, rawPath, headers);
  }
  stage = 'trusted proxy and asset paths';
  assert.equal((await rawRead('/-/soda/api/session')).status, 401);
  for (const route of ['/api/session', '/-/soda', '/-/soda/healthz', '/-/soda/api//session', '/-/soda/api/../api/session', '/-/soda/%61pi/session', '/-/soda/api/session/']) {
    assert.equal((await rawRead(route)).status, 404);
  }
  const version = await rawRead('/api/v1/version');
  assert.equal(version.status, 200);
  // Stock Makefile can include its Gitea compatibility suffix in setting.AppVer.
  const forgejoVersion = object(JSON.parse(version.body.toString())).version;
  assert(typeof forgejoVersion === 'string');
  assert.match(forgejoVersion, /^15\.0\.7(?:\+gitea-1\.22\.0)?$/);
  const presentationVersion=(await Bun.file(path.join(root,'appliance/forgejo/templates/custom/header.tmpl')).text()).match(/name="soda-presentation-revision" content="([a-zA-Z0-9.-]+)"/)?.[1];
  assert(presentationVersion, 'Packaged presentation epoch missing');
  for (const [destination, source] of Object.entries(forgejoPayload).filter(([target, source]) => source.startsWith('@build/forgejo-js/') || /^public\/assets\/sodaspaces[^/]*\.css$/.test(target) || target === 'public/assets/soda-settings.css')) {
    const asset = destination.slice('public'.length);
    const response = await rawRead(asset);
    assert.equal(response.status, 200);
    assert.match(response.headers['cache-control'] || '', /max-age=0/);
    assert.match(response.headers['cache-control'] || '', /must-revalidate/);
    const expected = await Bun.file(source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : path.join(root, source)).bytes();
    assert.equal(new Bun.CryptoHasher('sha256').update(response.body).digest('hex'), new Bun.CryptoHasher('sha256').update(expected).digest('hex'));
    const validator: Record<string, string> = {};
    if (response.headers.etag) validator['If-None-Match'] = response.headers.etag;
    else {assert(response.headers['last-modified']); validator['If-Modified-Since'] = response.headers['last-modified'];}
    assert.equal((await rawRead(asset, validator)).status, 304);
    const versioned=await rawRead(asset+'?v='+presentationVersion);
    assert.equal(versioned.status,200);
    assert.match(versioned.headers['cache-control'] || '', /max-age=0/);
    assert.match(versioned.headers['cache-control'] || '', /must-revalidate/);
    assert.equal(new Bun.CryptoHasher('sha256').update(versioned.body).digest('hex'),new Bun.CryptoHasher('sha256').update(expected).digest('hex'));
    assert.equal((await rawRead(asset+'?v='+presentationVersion,validator)).status,304);
  }
  result.asset_revalidation = 'conditional reads of current bytes; not an update/cutover proof';

  stage = 'trusted browser startup';

  assert(!interrupted);
  const {launchNativeBrowser} = await import('./native-browser.ts');
  nativeBrowser = await launchNativeBrowser(chromium, run, home);
  const context = nativeBrowser.context;
  assert(!interrupted);
  context.setDefaultTimeout(30000);
  let environmentReads = 0;
  let authorizations = 0;
  const writes = new Set(['/user/login', '/user/logout', '/login/oauth/grant', '/-/soda/api/session/logout']);
  let accessWrite: {actor: string; path: string; body: string; method?: string; page?: Page} | null = null; // One exact expected request, consumed before transmission.
  async function guardedPage(pageContext: BrowserContext = context) {
    const p = await pageContext.newPage();
    p.on('close', () => {accessWrite=null;});
    p.on('framenavigated', frame => {if(frame === p.mainFrame()) accessWrite=null;});
    await p.setViewportSize({width: 1280, height: 900});
    const cdp = await pageContext.newCDPSession(p);
    // Playwright route handlers omit redirect hops. CDP Fetch pauses each hop
    // before transmission, including the authorization redirect from Soda.
    cdp.on('Fetch.requestPaused', async ({requestId, request}) => {
      try {
        const url = new URL(request.url);
        // Native logout's link action posts this fixed navigation-only form.
        const logoutRedirect = request.method === 'POST' && url.pathname === '/-/fetch-redirect' && request.postData === 'redirect=%2F';
        const header = (name: string) => Object.entries(request.headers || {}).find(([key]) => key.toLowerCase() === name)?.[1];
        const cancellation = url.pathname === '/-/soda/api/login/cancel' && !url.search &&
          header('x-soda-logout') === '1' && input.users.some(user => user.id === header('x-soda-expected-user-id')) &&
          (request.method === 'GET' || (request.method === 'POST' && request.postData === '{}' && /^[A-Za-z0-9_-]{43}$/.test(header('x-csrf-token') || '')));
        const expectedAccess = !interrupted && !refusedRequest && accessWrite && (!accessWrite.page || accessWrite.page === p) && url.origin === origin.origin && !url.search &&
          request.method === (accessWrite.method || 'POST') && url.pathname === accessWrite.path && request.postData === accessWrite.body &&
          Object.entries(request.headers || {}).some(([name, value]) => name.toLowerCase() === 'x-soda-expected-user-id' && value === accessWrite?.actor);
        if (expectedAccess) accessWrite = null;
        const denied = url.origin !== origin.origin ||
          (url.pathname === '/-/soda/api/login/cancel' && !cancellation) ||
          (!['GET', 'HEAD'].includes(request.method) && ((interrupted || refusedRequest) || (!writes.has(url.pathname) && !logoutRedirect && !cancellation && !expectedAccess))) ||
          (url.pathname === '/login/oauth/authorize' && url.searchParams.get('client_id') !== input.oauth_client_id);
        if (denied) {
          accessWrite=null;
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
      } catch (error) {
        accessWrite=null;
        // Fixed categories only: protocol errors can embed private request data.
        result.interception_failure = error instanceof Error && error.message.includes('Invalid InterceptionId')
          ? 'request-no-longer-intercepted' : 'continuation-failed';
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
  const observedRoutes = new Set(['/-/soda/api/login/cancel', '/-/soda/api/settings/runners', '/', '/-/fetch-redirect', '/user/login', '/user/logout', '/login/oauth/authorize', '/login/oauth/grant', '/-/soda/api/session/logout',
    '/-/soda/login', '/-/soda/oauth/callback', '/-/soda/api/session', '/-/soda/api/forgejo/me', '/-/soda/api/environments']);
  context.on('response', response => {
    const pathname = new URL(response.url()).pathname;
    if (observedRoutes.has(pathname) && result.http.length < 128) result.http.push({route: pathname, status: response.status()});
  });
  const page = await guardedPage();
  const repoURL = origin.origin + input.repository_path;
  const drawer = page.locator('#sodaspaces-drawer');
  const control = (name: string, p = page) => p.locator(`[data-project-controls][data-repository-id="${input.repository_id}"] [data-control="${name}"]`);
  const permitTerminalEnd = (actor: string, environment: string, id: string) => {
    assert(terminalMode && input.terminal_actions?.includes('end') && !accessWrite && !interrupted && !refusedRequest);
    assert(/^p[0-9a-f]{24}$/.test(environment) && terminalID(id));
    accessWrite = {actor, path: `/-/soda/api/environments/${environment}/terminal-sessions/${id}`, body: JSON.stringify({action: 'end'})};
  };
  async function settled(p = page) {
    await p.locator('#sodaspaces-drawer').waitFor({state: 'visible'});
    await p.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    const project = p.locator('[data-project-controls]:visible');
    if (await project.count()) await p.locator('[data-project-controls]:visible[aria-busy=false]').waitFor();
  }
  async function open(p = page) {
    if (!(await p.locator('#sodaspaces-drawer').isVisible())) await p.locator('#sodaspaces-button').click();
    await settled(p);
  }
  async function nativeLogin(p: Page, index: number) {
    const user = input.users[index], password = passwords[index]; assert(user && password);
    assert(!interrupted);
    await p.goto(origin.origin + '/user/login');
    await p.locator('#user_name').fill(user.login);
    await p.locator('#password').fill(password);
    assert(!interrupted);
    await Promise.all([
      p.waitForURL(url => url.pathname !== '/user/login'),
      p.locator('form:has(#user_name) button').click(),
    ]);
    if (runnerMode) return; // Native-view connection below, not the drawer journey.
    await p.goto(repoURL);
    assert.equal(await p.locator('#sodaspaces-root').getAttribute('data-user-id'), user.id);
    assert.equal(await p.locator('#sodaspaces-root').getAttribute('data-repository-id'), input.repository_id);
  }
  async function nativeLogout(p: Page) {
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
  async function oauth(index: number) {
    const user = input.users[index]; assert(user);
    assert(!interrupted);
    const before = authorizations;
    const journey = stage;
    stage = journey + ': authorization navigation';
    await page.getByRole('link', {name: 'Connect to Soda', exact: true}).click();
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
    assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-user-id'), user.id);
    stage = journey + ': displayed Soda actor';
    await projectView(page, input.repository_id);
    assert.match(await control('actor').innerText(), new RegExp(`ID ${user.id}\\)`));
    stage = journey + ': environment state';
    const state = await control('status').innerText();
    assert(/^(No shared environment\.|Environment (running\.|stopped\.)|Provisioning incomplete\.|Native state unavailable;)/.test(state));
    result.states.push(state.startsWith('No shared') ? 'absent' : state.startsWith('Environment running') ? 'running'
      : state.startsWith('Environment stopped') ? 'stopped' : state.startsWith('Provisioning incomplete') ? 'incomplete' : 'live-status-unavailable');
    await projectView(page, input.repository_id, 'Access');
    if (await control('connection').isVisible()) {
      const command = await control('command').inputValue();
      const fingerprint = await control('fingerprint').innerText();
      assert.match(command, /^ssh [a-z][a-z0-9_-]{0,30}@[0-9a-fA-F:.]+$/);
      assert.match(fingerprint, /^Ed25519 host-key fingerprint: SHA256:[A-Za-z0-9+/]{43}$/);
      assert.equal(await control('copy').getAttribute('data-clipboard-target'), '#soda-command-' + input.repository_id);
      assert(!(await control('copy').isDisabled()));
      (result.own_connections ||= []).push({user_id: user.id, repository_id: input.repository_id, command, fingerprint});
    }
    const cookies = await context.cookies(origin.origin + '/-/soda/api/session');
    const soda = cookies.find(cookie => cookie.name === '__Secure-sodaspaces-session');
    assert(soda && soda.domain === origin.hostname && soda.path === '/-/soda/' && soda.secure && soda.httpOnly && soda.sameSite === 'Lax');
    assert(!(await context.cookies(repoURL)).some(cookie => cookie.name === '__Secure-sodaspaces-session'));
  }

  if (runnerRequest) {
    const browser=context.browser(); assert(browser);
    const deniedContext=await browser.newContext({serviceWorkers:'block'});
    deniedContext.setDefaultTimeout(30000);
    const runnerEvidence: RunnerEvidence={}; result.runner=runnerEvidence;
    try {
      const deniedPage=await guardedPage(deniedContext);
      for (const [actorPage,index] of [[page,0],[deniedPage,1]] as const) {
        stage='runner native authentication '+index;
        await nativeLogin(actorPage,index);
        await actorPage.goto(origin.origin+'/?soda-view=runners');
        await Promise.race([
          actorPage.locator('soda-runners').waitFor({state:'attached'}),
          actorPage.locator('#authorize-app').waitFor({state:'visible'}),
        ]);
        if(await actorPage.locator('#authorize-app').isVisible()) {
          assert.equal(await actorPage.locator('input[name="client_id"]').inputValue(),input.oauth_client_id);
          assert(!interrupted && !refusedRequest);
          await actorPage.locator('#authorize-app').click();
        }
        await actorPage.locator('soda-runners').waitFor({state:'attached'});
        assert.equal(await actorPage.locator('meta[name="soda-presentation-revision"]').getAttribute('content'),presentationVersion);
        const user=input.users[index]; assert(user);
        // Fresh session/provider reads prove the cookie identities, not stale DOM
        // markers. Keep only the selected role facts, never provider metadata.
        const identity=await actorPage.evaluate(async actor => {
          const headers={'X-Soda-Expected-User-ID':actor};
          const session=await fetch('/-/soda/api/session',{headers,credentials:'same-origin',cache:'no-store',redirect:'error'});
          const me=await fetch('/-/soda/api/forgejo/me',{headers,credentials:'same-origin',cache:'no-store',redirect:'error'});
          if(!session.ok || !me.ok) throw Error('Actor unavailable');
          const s: unknown=await session.json(), m: unknown=await me.json();
          if(!s || typeof s !== 'object' || !('user' in s) || !s.user || typeof s.user !== 'object' || !('id' in s.user) ||
            !('soda_operator' in s) || !m || typeof m !== 'object' || !('id' in m) || !('is_admin' in m)) throw Error('Invalid actor response');
          if(typeof s.user.id !== 'string' || typeof m.id !== 'string' || typeof s.soda_operator !== 'boolean' || typeof m.is_admin !== 'boolean') throw Error('Invalid actor facts');
          return {id:s.user.id,provider_id:m.id,operator:s.soda_operator,admin:m.is_admin};
        },user.id);
        assert(identity.id === user.id && identity.provider_id === user.id);
        assert(identity.operator === (index === 0) && identity.admin === (index === 1), 'Required nonadmin operator/nonoperator administrator not established');
      }
      assert(context !== deniedContext && page.context() !== deniedPage.context());
      stage='runner phase '+runnerRequest.phase;
      const permit=(actor: string, route: string, body: string) => {
        assert(!accessWrite && !interrupted && !refusedRequest && actor === runnerRequest.operator_id);
        const action=['overlap','departure'].includes(runnerRequest.phase) ? 'restart' : runnerRequest.phase;
        const expected='/api/settings/runners'+(action === 'register' ? '' : '/'+runnerRequest.runner_id+'/'+action);
        assert(['register','start','stop','restart','remove'].includes(action) && route === expected);
        accessWrite={actor,path:'/-/soda'+route,body,page}; // Preserve the exact serialized secret body; never log or reserialize it.
      };
      assert(extra[2]);
      await exerciseRunners(page,deniedPage,runnerRequest,extra[2],permit,runnerEvidence);
      assert(accessWrite === null && runnerEvidence.outcome === 'confirmed' && !interrupted && !refusedRequest);
      runnerConfirmed=true;
    } finally {
      accessWrite=null;
      await deniedContext.close();
    }
  } else {
  stage = 'anonymous and native-cookie-only contexts';
  const anonymous = await page.goto(repoURL);
  if (privateRepository) {
    // Stock 15.0.7 denies anonymous private repository access before supplying
    // template repository context. Never make a retained repository public to
    // satisfy the public-fixture journey, or treat denial as Soda absence.
    assert(anonymous);
    assert.equal(anonymous.status(), 404);
    assert(!(await page.locator('#sodaspaces-button').isVisible()));
    assert.equal(environmentReads, 0);
    result.anonymous_repository = 'native 404; no Soda repository reads';
  } else {
    assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-signed'), 'false');
    await open();
    assert.equal(environmentReads, 0);
    await page.getByRole('link', {name: 'Connect to Soda', exact: true}).waitFor({state: 'visible'});
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
    const session: unknown = await (await fetch(base, {credentials: 'same-origin', cache: 'no-store', signal: AbortSignal.timeout(15000)})).json();
    if (!session || typeof session !== 'object' || !('user' in session) || !session.user || typeof session.user !== 'object' || !('id' in session.user) || typeof session.user.id !== 'string' || !('csrf_token' in session) || typeof session.csrf_token !== 'string') throw Error('Invalid session');
    const outcomes = [];
    for (const [actor, csrf] of [[other, session.csrf_token], [session.user.id, 'intentionally-invalid']] as const) {
      const response = await fetch(base + '/logout', {method: 'POST', credentials: 'same-origin', signal: AbortSignal.timeout(15000),
        headers: {'Content-Type': 'application/json', 'X-Soda-Expected-User-ID': actor, 'X-CSRF-Token': csrf}, body: '{}'});
      const body: unknown = await response.json();
      if (!body || typeof body !== 'object' || !('error' in body) || !body.error || typeof body.error !== 'object' || !('code' in body.error) || typeof body.error.code !== 'string') throw Error('Invalid denial response');
      outcomes.push({status: response.status, code: body.error.code});
    }
    return outcomes; // No credentials leave the browser evaluation.
  }, input.users[1].id);
  assert.deepEqual(guards, [{status: 403, code: 'identity_mismatch'}, {status: 403, code: 'invalid_csrf'}]);

  stage = 'non-modal native keyboard and outside-click coexistence';
  const mountedWorkspace = await page.locator('soda-spaces').elementHandle();
  const beforeFocus = environmentReads;
  for (let i = 0; i < 8; i++) await page.keyboard.press('Tab');
  await page.getByRole('button', {name: 'Sessions', exact: true}).focus();
  await page.keyboard.press('Escape'); assert(await drawer.isVisible());
  await page.mouse.click(4, 800); assert(await drawer.isVisible());
  assert(await mountedWorkspace?.evaluate(node => node.isConnected));
  assert.equal(environmentReads, beforeFocus);
  result.native_focus_preserves_workspace = true;
  stage = 'automatic theme and measured compact selection';
  const theme = await page.locator('html').getAttribute('data-theme'); assert(theme); assert.match(theme, /auto/);
  const backgrounds = [];
  for (const [width, colorScheme] of [[360, 'light'], [1280, 'dark']] as const) {
    stage = `native ${width}-pixel ${colorScheme} layout`;
    await page.setViewportSize({width, height: 900});
    await page.emulateMedia({colorScheme});
    await open();
    const box = await drawer.boundingBox();
    const viewport = await page.evaluate(() => ({width: window.visualViewport?.width || window.innerWidth, height: window.visualViewport?.height || window.innerHeight,
      top: window.visualViewport?.offsetTop || 0, control_height: getComputedStyle(document.body).getPropertyValue('--soda-control-height')}));
    result.native_layout = {width, colorScheme, box, viewport,
      terminal_pressed: await page.getByRole('button', {name: 'Terminal', exact: true, includeHidden: true}).getAttribute('aria-pressed')};
    // The stock browser reserves native scrollbar space; Playwright's requested
    // width is not necessarily the actual visual viewport used by the adapter.
    assert(viewport.width > 0 && viewport.width <= width);
    assert(box && box.x >= -1 && box.width <= viewport.width + 1 && Math.abs(box.x + box.width - viewport.width) < 2);
    if (width === 360) {assert(box.height >= 854); assert.equal(await page.getByRole('button', {name: 'Terminal', exact: true}).getAttribute('aria-pressed'), 'true');}
    else {assert(box.height >= 898 && box.x >= 480);}
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
  assert(await mountedWorkspace?.evaluate(node => node.isConnected));
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
  await page.goto(repoURL + '#sodaspaces'); await settled();
  assert.equal(await page.locator('#sodaspaces-root').getAttribute('data-user-id'), input.users[1].id);
  assert(await page.getByRole('link', {name: 'Connect to Soda', exact: true}).isVisible());
  assert.equal(environmentReads, beforeSwitch);
  stage = 'second real OAuth repository return';
  await oauth(1);

  stage = 'coordinated native logout in another tab';
  await other.bringToFront();
  await other.goto(repoURL);
  await open(other);
  await projectView(other, input.repository_id);
  // The selected drawer Sign out delegates to the native coordinator, not the
  // historical Soda-only logout endpoint. Verify both real authority boundaries.
  const sodaEnded = other.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/login/cancel' && r.request().method() === 'POST');
  const forgejoEnded = other.waitForResponse(r => new URL(r.url()).pathname === '/user/logout' && r.request().method() === 'POST');
  await control('sign-out', other).click({noWaitAfter: true});
  assert.equal((await sodaEnded).status(), 204);
  assert.equal((await forgejoEnded).status(), 200);
  for (const p of [other, page]) {
    await p.bringToFront();
    // Revalidate the peer through stock HTML; do not assume upstream promises
    // an immediate navbar replacement in every background document.
    if (p === page) await p.goto(repoURL);
    await p.waitForFunction(() => document.readyState === 'complete' && !!document.querySelector('#navbar a[href^="/user/login"]'));
    assert.equal(await p.evaluate(async () => (await fetch('/-/soda/api/session', {cache: 'no-store'})).status), 401);
    assert.equal(await p.locator('.soda-workspace-terminal:visible').count(), 0);
  }
  result.coordinated_peer_logout = true;
  await nativeLogin(page, 1);
  await open();
  await oauth(1);

  stage = 'native unsaved form coexistence';
  await page.goto(repoURL + '/issues/new');
  await page.locator('#issue_title').fill('Unsaved Sodaspaces fixture text');
  await open();
  await other.bringToFront();
  await page.bringToFront();
  assert(await drawer.isVisible());
  const draft = await page.locator('#issue_title').elementHandle();
  assert.equal(await page.locator('#issue_title').inputValue(), 'Unsaved Sodaspaces fixture text');
  await page.locator('#issue_title').focus();
  await page.setViewportSize({width: 390, height: 900});
  await page.getByRole('button', {name: 'Forge', exact: true}).waitFor();
  await page.getByRole('button', {name: 'Terminal', exact: true}).click();
  assert(await page.locator('#issue_title').isHidden());
  assert(await draft?.evaluate(node => node.isConnected));
  await page.getByRole('button', {name: 'Forge', exact: true}).click();
  assert.equal(await page.locator('#issue_title').inputValue(), 'Unsaved Sodaspaces fixture text');
  await page.getByRole('button', {name: 'Terminal', exact: true}).click();
  assert.equal(new URL(page.url()).pathname, input.repository_path + '/issues/new');
  await page.locator('#sodaspaces-close').click();
  assert.equal(await page.locator('#issue_title').inputValue(), 'Unsaved Sodaspaces fixture text');
  stage = 'cancel ordinary Open in Spaces navigation with native beforeunload';
  await open();
  const cancelled = page.waitForEvent('dialog').then(async dialog => {assert.equal(dialog.type(), 'beforeunload'); await dialog.dismiss();});
  await page.getByRole('link', {name: 'Open in Spaces', exact: true}).click(); await cancelled;
  assert(await draft?.evaluate(node => node.isConnected));
  assert.equal(new URL(page.url()).pathname, input.repository_path + '/issues/new');
  result.beforeunload_cancelled_without_detach = true;
  stage = 'discard only synthetic form text';
  const discardFixture = async (dialog: Dialog) => {
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
    await page.getByRole('button', {name: 'Forge', exact: true}).waitFor();
    assert.equal(await page.getByRole('button', {name: 'Forge', exact: true}).getAttribute('aria-pressed'), 'true');
    assert(await drawer.isHidden());
  }
  await page.setViewportSize({width: 1280, height: 900});
  stage = 'authenticated native Spaces host and native links';
  await open(); await page.getByRole('link', {name: 'Open in Spaces', exact: true}).click();
  await page.waitForURL(url => url.pathname === '/' && url.searchParams.get('soda-view') === 'spaces');
  await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
  assert.equal(await page.locator('#soda-native-content').getAttribute('data-actor'), input.users[1].id);
  assert.equal(await page.locator('soda-spaces').count(), 1);
  const issuesHref = await page.locator('#navbar').getByRole('link', {name: 'Issues', exact: true}).getAttribute('href');
  assert(issuesHref);
  assert.equal(new URL(issuesHref, origin).href, origin.origin + '/issues');
  await page.goto(repoURL); await open();
  if (accessMode) {
    assert(result.bfcache_restored, 'Access writes require the read-only journey to complete first');
    await other.close();
    await page.goto(repoURL);
    await open();
    stage = 'access fixture must initially have no reservation';
    await projectView(page, input.repository_id);
    assert.match(await control('status').innerText(), /^No shared environment\./);
    const access: AccessEvidence = {reservation_id: null, users: [], copy_native_paste: []};
    result.access = access;
    const permit = (index: number, route: string, body: unknown) => {
      const user = input.users[index]; assert(user);
      assert(!accessWrite && !interrupted && !refusedRequest);
      accessWrite = {actor: user.id, path: '/-/soda' + route, body: JSON.stringify(body)};
    };
    stage = 'nonowner native create denial';
    permit(1, '/api/environments', {repository_id: input.repository_id});
    const denial = await page.evaluate(async ({actor, repository}) => {
      const s: unknown = await (await fetch('/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error'})).json();
      if (!s || typeof s !== 'object' || !('user' in s) || !s.user || typeof s.user !== 'object' || !('id' in s.user) || !('csrf_token' in s) || typeof s.csrf_token !== 'string') throw Error('Invalid session');
      if (s.user.id !== actor) throw new Error('actor mismatch');
      const response = await fetch('/-/soda/api/environments', {method: 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
        headers: {'Content-Type': 'application/json', 'X-Soda-Expected-User-ID': actor, 'X-CSRF-Token': s.csrf_token}, body: JSON.stringify({repository_id: repository})});
      const body: unknown = await response.json();
      if (!body || typeof body !== 'object' || !('error' in body) || !body.error || typeof body.error !== 'object' || !('code' in body.error) || typeof body.error.code !== 'string') throw Error('Invalid denial response');
      return {status: response.status, code: body.error.code};
    }, {actor: input.users[1].id, repository: input.repository_id});
    assert.deepEqual(denial, {status: 403, code: 'owner_required'});
    assert.equal(accessWrite, null);
    access.nonowner_create = 'denied';
    for (const index of [0, 1] as const) {
      const user = input.users[index], publicKey = publicKeys[index]; assert(publicKey);
      stage = `access user ${index}: native login and OAuth`;
      if (await drawer.isVisible()) await page.locator('#sodaspaces-close').click();
      await nativeLogout(page);
      await nativeLogin(page, index);
      await open();
      await oauth(index);
      await projectView(page, input.repository_id);
      if (index === 0) {
        stage = 'explicit native shared-environment creation';
        permit(index, '/api/environments', {repository_id: input.repository_id});
        const completed = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/environments' && r.request().method() === 'POST', {timeout: 300000});
        await control('create').click();
        const response = await completed;
        const created = object(await response.json());
        const id = created.id || (created.environment ? object(created.environment).id : undefined);
        if (typeof id === 'string' && /^p[0-9a-f]{24}$/.test(id)) access.reservation_id = id;
        assert.equal(response.status(), 201);
        assert.equal(created.repository_id, input.repository_id);
        assert(access.reservation_id && created.provisioned === true && accessWrite === null);
        await settled();
        assert.equal(await control('login').innerText(), '');
        assert(await control('connection').isHidden());
      }
      stage = `access user ${index}: explicit public key registration`;
      await projectView(page, input.repository_id, 'Access');
      await control('public-key').fill(publicKey);
      permit(index, '/api/me/development-keys', {public_key: publicKey});
      const saved = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/me/development-keys' && r.request().method() === 'POST');
      await control('save-key').click();
      assert.equal((await saved).status(), 200);
      await settled();
      assert.equal(accessWrite, null);
      assert.equal(await control('login').innerText(), '');
      stage = `access user ${index}: explicit native account join`;
      await projectView(page, input.repository_id);
      permit(index, `/api/environments/${access.reservation_id}/join`, {});
      const joined = page.waitForResponse(r => new URL(r.url()).pathname === `/-/soda/api/environments/${access.reservation_id}/join` && r.request().method() === 'POST', {timeout: 300000});
      await control('join').click();
      const joinedResponse = await joined;
      assert.equal(joinedResponse.status(), 200);
      assert.equal(object(await joinedResponse.json()).login, user.login);
      await settled();
      assert.equal(accessWrite, null);
      await projectView(page, input.repository_id, 'Access');
      await control('connection').waitFor({state: 'visible'});
      const command = await control('command').inputValue();
      stage = `access user ${index}: current own connection`;
      const own = object(await page.evaluate(async ({actor, id}) => {
        const response = await fetch(`/-/soda/api/environments/${id}/connection`, {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': actor}});
        if (response.status !== 200) throw new Error('connection unavailable');
        return await response.json() as unknown; // Public connection fields only; no bootstrap credentials.
      }, {actor: user.id, id: access.reservation_id}));
      const connection = object(own.connection), environment = object(connection.environment);
      assert.equal(own.login, user.login);
      assert.equal(environment.id, access.reservation_id);
      assert.equal(environment.running, true);
      assert.equal(own.routing_verified, false);
      assert.equal(command, `ssh ${own.login}@${environment.ip}`);
      assert(typeof connection.host_key === 'string' && typeof connection.fingerprint === 'string');
      assert.match(connection.host_key.trim(), /^ssh-ed25519 [A-Za-z0-9+/]{68}$/);
      assert.match(connection.fingerprint, /^SHA256:[A-Za-z0-9+/]{43}$/);
      access.users.push({id: user.id, ...own});
      stage = `access user ${index}: native Copy and real paste`;
      const copied = await page.evaluate(() => window.config.i18n.copy_success);
      assert(typeof copied === 'string' && copied.length > 0);
      await control('copy').click();
      await page.getByRole('tooltip').filter({hasText: copied}).waitFor({state: 'visible'});
      await page.locator('#sodaspaces-close').click();
      await page.goto(repoURL + '/issues/new');
      await page.locator('#issue_title').focus();
      await page.keyboard.press('Control+V');
      assert.equal(await page.locator('#issue_title').inputValue(), command);
      access.copy_native_paste.push(user.id);
      // Discard only the command pasted by this run; never submit the issue.
      page.on('dialog', discardFixture);
      try { await page.goto(repoURL); } finally { page.off('dialog', discardFixture); }
      await open();
      await projectView(page, input.repository_id, 'Access');
      assert.equal(await control('command').inputValue(), command);
      await projectView(page, input.repository_id);
      assert(await control('join').isHidden());
    }
    access.native_join_confirmed = true;
  }
  async function authenticateExisting(index: number) {
    await page.goto(repoURL);
    if (await page.locator('a[data-url="/user/logout"]').count()) await nativeLogout(page);
    await nativeLogin(page, index);
    await page.goto(repoURL);
    await open();
    if (await page.getByRole('button', {name: 'New terminal', exact: true}).isEnabled()) {
      await projectView(page, input.repository_id);
      const response = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda/api/session/logout' && r.request().method() === 'POST');
      await control('sign-out').click(); assert.equal((await response).status(), 200);
      await settled();
      await page.reload();
      await open();
    }
    await oauth(index);
    await settled();
  }
  if (matrixMode) {
    assert(matrixRequest && result.bfcache_restored, 'Matrix requires explicit input and the read-only journey first');
    const matrix: unknown[] = []; result.workspace_matrix = matrix;
    for (let index = 0; index < input.users.length; index++) {
      await authenticateExisting(index); stage = `workspace matrix actor ${index}`;
      const user = input.users[index]; assert(user);
      await page.goto(origin.origin + '/?soda-view=spaces'); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
      const outcome: import('./sodaspaces-workspace-journey').MatrixEvidence = {sessions: []}; matrix.push(outcome);
      const observer = observeMatrixShell(page);
      const active = () => {assert(!interrupted && !refusedRequest);};
      try {
        await exerciseWorkspaceMatrix(page, matrixRequest, index, (session, action) => {
          active(); assert(!accessWrite && session.actor === user.id && terminalID(session.id));
          assert(matrixRequest.actions.includes(action) && matrixRequest.projects.some(project => project.environment === session.environment));
          assert(outcome.sessions.some(owned => owned.id === session.id && owned.environment === session.environment));
          accessWrite = {actor: user.id, path: `/-/soda/api/environments/${session.environment}/terminal-sessions/${session.id}`,
            body: JSON.stringify({action, ...(action === 'end' ? {} : {attachment_id: session.attachment})})};
        }, {
          shell: async (_page, session, initialize) => {active(); return observer.shell(session, initialize);},
          inspect: async (project, actor, session, facts, ended) => {active(); await inspectMatrixProcess(matrixRequest, project, actor, session, facts, ended);},
          cli: (_page, sessions) => exerciseSelectedCLIs(page, matrixRequest, index, sessions, observer, active),
        }, outcome, guardedPage);
        assert.equal(accessWrite, null);
      } finally {observer.dispose();}
    }
  }
  if (terminalMode && !matrixMode) {
    assert(result.bfcache_restored, 'Existing-account effects require the read-only journey first');
    const terminals: Array<ShellFacts & {actor: string; socket_closed: boolean; refresh_did_not_reconnect: boolean}> = [];
    result.terminals = terminals;
    for (let index = 0; index < input.users.length; index++) {
      const user = input.users[index]; assert(user);
      stage = `terminal user ${index}: real native login and consent`;
      await authenticateExisting(index);
      assert(await control('connection').isVisible());
      const controls = await projectView(page, input.repository_id, 'Access'), environment = await controls.getAttribute('data-environment-id');
      assert(environment && /^p[0-9a-f]{24}$/.test(environment));
      const marker = `SODA_E2E_${index}_${Date.now()}`;
      let wire = '', facts: ShellFacts | undefined, sockets = 0, closed = false;
      let target: {environment: string; id: string} | undefined;
      const observe = (ws: WebSocket) => {
        const url = new URL(ws.url());
        assert.equal(url.origin, origin.origin.replace('https:', 'wss:'));
        assert.match(url.pathname, /^\/-\/soda\/api\/environments\/p[0-9a-f]{24}\/terminal$/);
        assert(!url.search);
        sockets++;
        ws.on('close', () => { closed = true; });
        ws.on('framereceived', ({payload}) => {
          // Inspect only a bounded transient buffer for our own structured facts.
          // Never retain terminal transcript, authentication frames or input.
          try {
            const frame = object(JSON.parse(typeof payload === 'string' ? payload : payload.toString()));
            if (frame.type === 'session') {assert(!target && url.pathname === `/-/soda/api/environments/${environment}/terminal` && terminalID(frame.id)); target = {environment, id: frame.id}; return;}
            if (frame.type !== 'output' || typeof frame.data !== 'string') return;
            wire = (wire + Buffer.from(frame.data, 'base64').toString('utf8')).slice(-16384);
            const normalized = wire.replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '').replaceAll('\r', '');
            const line = normalized.split('\n').find(s => s.startsWith(marker + ':'));
            if (line) {
              const value = object(JSON.parse(line.slice(marker.length + 1).trim()));
              const {login, uid, gid, groups, home, cwd, tty, shell_pid, shell_start} = value;
              assert(typeof login === 'string' && typeof uid === 'number' && typeof gid === 'number');
              assert(Array.isArray(groups) && groups.every((group: unknown) => typeof group === 'number'));
              assert(typeof home === 'string' && typeof cwd === 'string' && typeof tty === 'boolean');
              assert(typeof shell_pid === 'number' && typeof shell_start === 'string');
              facts = {login, uid, gid, groups, home, cwd, tty, shell_pid, shell_start};
            }
          } catch { /* A frame can end partway through the fact line. */ }
        });
      };
      page.on('websocket', observe);
      stage = `terminal user ${index}: explicit real shell`;
      assert(input.terminal_actions?.includes('create'));
      await newManagedTerminal(page, input.repository_path.slice(1), marker, environment);
      const code = "import os,pwd,json; print(" + JSON.stringify(marker + ':') + "+json.dumps(dict(login=pwd.getpwuid(os.getuid()).pw_name,uid=os.getuid(),gid=os.getgid(),groups=os.getgroups(),home=os.environ['HOME'],cwd=os.getcwd(),tty=os.isatty(0),shell_pid=os.getppid(),shell_start=open('/proc/%d/stat'%os.getppid()).read().split()[21])))";
      const command = "python3 -c '" + code.replaceAll("'", "'\\''") + "'";
      const screen = page.locator('.soda-terminal .xterm-helper-textarea');
      await screen.focus();
      await page.keyboard.insertText(command);
      await page.keyboard.press('Enter');
      stage = `terminal user ${index}: structured native shell facts`;
      const deadline = Date.now() + 15000;
      while (!facts && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 100));
      assert(facts && facts.uid > 0 && facts.login === user.login && facts.tty);
      assert.equal(facts.home, `/home/${user.login}`);
      assert.equal(facts.cwd, facts.home);
      assert.equal(sockets, 1);
      terminals.push({actor: user.id, ...facts, socket_closed: false, refresh_did_not_reconnect: false});
      stage = `terminal user ${index}: native Escape and focus escape`;
      await page.keyboard.press('Escape');
      assert(await drawer.isVisible());
      await page.keyboard.press('Control+Shift+Enter');
      assert(await page.getByLabel('Terminal actions', {exact: true}).evaluate(e => e === document.activeElement));
      stage = `terminal user ${index}: explicit original-target End`;
      assert(target);
      await terminalMenu(page, 'End terminal…');
      assert.equal(await page.evaluate(() => document.activeElement?.textContent), 'Cancel');
      const endingTarget = target;
      permitTerminalEnd(user.id, endingTarget.environment, endingTarget.id);
      const ended = page.waitForResponse(r => new URL(r.url()).pathname === `/-/soda/api/environments/${endingTarget.environment}/terminal-sessions/${endingTarget.id}` && r.request().method() === 'POST');
      await page.getByRole('dialog', {name: 'End terminal confirmation', exact: true}).getByRole('button', {name: 'End terminal', exact: true}).click();
      const endResponse = await ended; assert.equal(endResponse.status(), 200); assert.equal(object(await endResponse.json()).ending, true);
      result[`terminal_${index}_locator`] = target; // Locator only, not native cleanup proof.
      const closeDeadline = Date.now() + 10000;
      while (!closed && Date.now() < closeDeadline) await new Promise(resolve => setTimeout(resolve, 100));
      assert(closed);
      stage = `terminal user ${index}: no reconnect after Refresh`;
      await page.getByLabel('Workspace options', {exact: true}).click(); await page.getByRole('button', {name: 'Refresh Spaces', exact: true}).click();
      await settled();
      assert.equal(sockets, 1);
      assert.equal(await page.locator('.soda-workspace-terminal:visible .is-connected').count(), 0);
      page.off('websocket', observe);
      wire = '';
      const terminal = terminals.at(-1); assert(terminal);
      Object.assign(terminal, {socket_closed: true, refresh_did_not_reconnect: true});
      // Host-side process disappearance must be independently checked, not inferred
      // from this socket closure or from shell facts alone.
    }
  }
  if (managementMode) {
    const {exerciseManagement} = await import('./sodaspaces-management.ts');
    assert(managementRequest);
    const management: ManagementEvidence = {};
    result.management = management;
    await exerciseManagement({page, input, request:managementRequest,
      authenticate:authenticateExisting, settled, evidence:management,
      stage: value => {stage=value;},
      permit: (index: number, route: string, body: unknown, method='POST') => {
        const user = input.users[index]; assert(user);
        assert(!accessWrite && !interrupted && !refusedRequest);
        accessWrite={actor:user.id,path:'/-/soda'+route,body:JSON.stringify(body),method};
      }});
    assert.equal(accessWrite,null);
  }
  }
  assert(!refusedRequest && !interrupted);
} catch (error) {
  if (evidence) evidence.failure_kind = (error instanceof Error ? error.name : undefined) === 'TimeoutError' ? 'timeout'
    : (error instanceof Error ? error.name : undefined) === 'AssertionError' ? 'assertion'
      : (error instanceof Error && error.message.includes('strict mode violation')) ? 'ambiguous_locator' : 'other';
  failure = true; // Never print Playwright errors/URLs/input bodies or credentials.
} finally {
  try { await nativeBrowser?.close(); } catch { failure = true; }
  failure ||= interrupted || refusedRequest;
  const outcome = failure ? 'failed' : runnerMode ? (runnerConfirmed ? 'confirmed-runner-phase' : 'incomplete-runner-phase') : evidence?.bfcache_restored ? 'passed-scoped-journey' : 'incomplete-bfcache-not-observed';
  if (run) {
    try {
      await writeFile(path.join(run, 'result.json'), JSON.stringify({...evidence, outcome, stage, environment_mutations: matrixMode ? 'explicit six-session/two-project per-actor matrix only; exact single-use lifetime actions; CLI observations require separately declared provider scope, not acceptance' : managementMode ? 'explicit existing-project lifecycle and temporary own-key rotation only; single-use actor/path/body/method-bound' : accessMode ? 'explicit single-use actor/path/body-bound create/key/join requests only' : 'not permitted; unexpected writes aborted', provisioning_ssh_proof: false}, null, 2) + '\n', {flag: 'wx', mode: 0o600});
    } catch { failure = true; }
  }
  console.log(failure ? `Sodaspaces journey failed at ${stage}; private profile/evidence retained if created.` : `Sodaspaces journey: ${outcome}; not whole-product or backend-artifact acceptance.`);
  process.exitCode = failure ? 1 : (runnerMode ? runnerConfirmed : evidence?.bfcache_restored) ? 0 : 2;
}
