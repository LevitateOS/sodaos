// SPDX-License-Identifier: Apache-2.0
// Execute the probe's actual request guard with CDP/page doubles. Native redirect
// delivery remains owned by the separately authorized browser journey.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';
import {mkdtemp, lstat, rm} from 'node:fs/promises';
import {createServer} from 'node:http';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {launchNativeBrowser} from '../installed/native-browser.mjs';

const probe = readFileSync(new URL('../installed/sodaspaces.mjs', import.meta.url), 'utf8');
const start = probe.indexOf('  async function guardedPage() {');
const end = probe.indexOf('  await context.addInitScript(', start);
assert(start > 0 && end > start);

test('private repository declaration is read-only and cannot combine with access mode', () => {
  const from = probe.indexOf('  const [inputFile, home, permission, ...extra]');
  const to = probe.indexOf('  const input = JSON.parse', from);
  assert(from > 0 && to > from);
  for (const [args, access, privateRepo] of [[[], false, false], [['--private-repository'], false, true], [['--allow-environment-access'], true, false]]) {
    const scope = {assert, accessMode: false, process: {argv: ['node', 'probe', 'input', 'home', '--allow-auth-transitions', ...args]}};
    const mode = runInNewContext(probe.slice(from, to) + '\n({accessMode, privateRepository})', scope);
    assert.equal(mode.accessMode, access);
    assert.equal(mode.privateRepository, privateRepo);
  }
  assert.throws(() => runInNewContext(probe.slice(from, to), {assert, process: {argv: ['node', 'probe', 'input', 'home', '--allow-auth-transitions', '--private-repository', '--allow-environment-access']}}));
});

test('private anonymous probe requires native denial without Soda repository reads', async () => {
  const from = probe.indexOf("  stage = 'anonymous and native-cookie-only contexts';");
  const to = probe.indexOf('  await nativeLogin(page, 0);', from);
  assert(from > 0 && to > from);
  for (const [status, visible, reads, valid] of [[404, false, 0, true], [200, false, 0, false], [404, true, 0, false], [404, false, 1, false]]) {
    const scope = {assert, privateRepository: true, repoURL: 'https://fixture.invalid/owner/private', environmentReads: reads, result: {},
      page: {async goto() { return {status() { return status; }}; }, locator() { return {async isVisible() { return visible; }}; }}};
    const attempt = runInNewContext('(async () => {\n' + probe.slice(from, to) + '\n})()', scope);
    if (valid) { await attempt; assert.equal(scope.result.anonymous_repository, 'native 404; no Soda repository reads'); }
    else await assert.rejects(attempt);
  }
});

test('read-only probe records only a usable displayed own connection', async () => {
  const from = probe.indexOf("    if (await page.locator('#sodaspaces-connection').isVisible()) {");
  const to = probe.indexOf('    const cookies = await context.cookies', from);
  assert(from > 0 && to > from);
  for (const disabled of [false, true]) {
    const scope = {assert, input: {users: [{id: '2'}], repository_id: '1'}, index: 0, result: {}, page: {
      locator() { return {async isVisible() { return true; }, async inputValue() { return 'ssh alice@10.89.0.2'; },
        async innerText() { return 'Ed25519 host-key fingerprint: SHA256:' + 'A'.repeat(43); },
        async getAttribute() { return '#sodaspaces-command'; }, async isDisabled() { return disabled; }}; },
    }};
    const run = runInNewContext('(async () => {\n'+probe.slice(from, to)+'\n})()', scope);
    if (disabled) await assert.rejects(run);
    else { await run; assert.equal(scope.result.own_connections[0].user_id, '2'); assert.equal(scope.result.own_connections[0].command, 'ssh alice@10.89.0.2'); }
  }
});

test('native probe guards every paused redirect before transmission', async () => {
  let paused;
  const calls = [];
  const page = {async setViewportSize(size) { assert.equal(size.width, 1280); assert.equal(size.height, 900); }};
  const cdp = {
    on(name, handler) { assert.equal(name, 'Fetch.requestPaused'); paused = handler; },
    async send(method, params) { calls.push({method, params}); },
  };
  const scope = {URL, origin: new URL('https://fixture.invalid'), input: {oauth_client_id: 'synthetic-client'},
    writes: new Set(['/user/login']), accessWrite: null, result: {}, refusedRequest: false, interrupted: false,
    authorizations: 0, environmentReads: 0,
    context: {async newPage() { return page; }, async newCDPSession(p) { assert.equal(p, page); return cdp; }}};
  assert.equal(await runInNewContext(probe.slice(start, end) + '\nguardedPage()', scope), page);
  assert.equal(calls[0].method, 'Fetch.enable');
  assert.equal(calls[0].params.patterns[0].requestStage, 'Request');
  await paused({requestId: 'allowed', redirectedRequestId: 'previous',
    request: {url: 'https://fixture.invalid/login/oauth/authorize?client_id=synthetic-client', method: 'GET'}});
  assert.equal(scope.authorizations, 1);
  assert.equal(calls.at(-1).method, 'Fetch.continueRequest');
  await paused({requestId: 'native-logout-redirect', request: {url: 'https://fixture.invalid/-/fetch-redirect', method: 'POST', postData: 'redirect=%2F'}});
  assert.equal(calls.at(-1).method, 'Fetch.continueRequest');
  for (const [url, method, postData] of [['https://outside.invalid/', 'GET'],
    ['https://fixture.invalid/-/fetch-redirect', 'POST'],
    ['https://fixture.invalid/-/fetch-redirect', 'POST', 'redirect=https%3A%2F%2Foutside.invalid'],
    ['https://fixture.invalid/-/fetch-redirect', 'POST', 'redirect=%2F&extra=value'],
    ['https://fixture.invalid/-/soda/api/environments', 'POST'],
    ['https://fixture.invalid/login/oauth/authorize?client_id=wrong', 'GET']]) {
    scope.refusedRequest = false;
    await paused({requestId: 'denied', redirectedRequestId: 'previous', request: {url, method, postData}});
    assert(scope.refusedRequest);
    assert.equal(calls.at(-1).method, 'Fetch.failRequest');
    assert.equal(calls.at(-1).params.errorReason, 'BlockedByClient');
    assert.equal(scope.authorizations, 1);
    assert.equal(scope.environmentReads, 0);
  }
  const intent = {path: '/-/soda/api/environments', body: '{"repository_id":"42"}', actor: '1'};
  const allowed = {url: 'https://fixture.invalid' + intent.path, method: 'POST', postData: intent.body, headers: {'X-Soda-Expected-User-ID': '1'}};
  for (const request of [{...allowed, postData: '{"repository_id":"43"}'}, {...allowed, headers: {'X-Soda-Expected-User-ID': '2'}}, {...allowed, url: allowed.url + '?extra=1'}]) {
    scope.accessWrite = intent;
    await paused({requestId: 'wrong-access', request});
    assert.equal(calls.at(-1).method, 'Fetch.failRequest');
    assert.equal(scope.accessWrite, intent);
  }
  scope.accessWrite = intent;
  await paused({requestId: 'one-access', request: allowed});
  assert.equal(calls.at(-1).method, 'Fetch.continueRequest');
  assert.equal(scope.accessWrite, null);
  await paused({requestId: 'replay-access', request: allowed});
  assert.equal(calls.at(-1).method, 'Fetch.failRequest');
});

test('native browser refuses long private socket paths before starting a process', async () => {
  await assert.rejects(launchNativeBrowser({executablePath() { assert.fail('must not launch'); }}, '/'+ 'x'.repeat(110), '/unused'), /socket path too long/);
});

test('native browser refuses an occupied socket without removing its owner', async () => {
  const run = await mkdtemp(path.join(tmpdir(), 'soda-cdp-'));
  const socket = path.join(run, 'cdp.sock');
  const owner = createServer();
  try {
    await new Promise(resolve => owner.listen(socket, resolve));
    await assert.rejects(launchNativeBrowser({executablePath() { assert.fail('must not launch'); }}, run, run), /attachment failed/);
    assert((await lstat(socket)).isSocket());
    assert(owner.listening);
  } finally {
    await new Promise(resolve => owner.close(resolve));
    await rm(run, {recursive: true}); // Exact temporary test-owned directory only.
  }
});
