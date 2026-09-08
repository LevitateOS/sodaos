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

test('native probe guards every paused redirect before transmission', async () => {
  let paused;
  const calls = [];
  const page = {async setViewportSize(size) { assert.equal(size.width, 1280); assert.equal(size.height, 900); }};
  const cdp = {
    on(name, handler) { assert.equal(name, 'Fetch.requestPaused'); paused = handler; },
    async send(method, params) { calls.push({method, params}); },
  };
  const scope = {URL, origin: new URL('https://fixture.invalid'), input: {oauth_client_id: 'synthetic-client'},
    writes: new Set(['/user/login']), result: {}, refusedRequest: false, interrupted: false,
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
