// SPDX-License-Identifier: Apache-2.0
// Execute the probe's actual request guard with CDP/page doubles. Native redirect
// delivery remains owned by the separately authorized browser journey.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';

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
  for (const [url, method] of [['https://outside.invalid/', 'GET'],
    ['https://fixture.invalid/-/soda/api/environments', 'POST'],
    ['https://fixture.invalid/login/oauth/authorize?client_id=wrong', 'GET']]) {
    scope.refusedRequest = false;
    await paused({requestId: 'denied', redirectedRequestId: 'previous', request: {url, method}});
    assert(scope.refusedRequest);
    assert.equal(calls.at(-1).method, 'Fetch.failRequest');
    assert.equal(calls.at(-1).params.errorReason, 'BlockedByClient');
    assert.equal(scope.authorizations, 1);
    assert.equal(scope.environmentReads, 0);
  }
});
