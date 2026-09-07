// SPDX-License-Identifier: Apache-2.0
// Actual Soda markup/script, synthetic API and dialog calls; not a native browser.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
const {JSDOM} = require('jsdom');
const footer = readFileSync(new URL('../../appliance/forgejo/templates/custom/footer.tmpl', import.meta.url), 'utf8');
const script = readFileSync(new URL('../../appliance/forgejo/public/assets/sodaspaces.js', import.meta.url), 'utf8');
const project = 'p0123456789abcdef01234567';
const session = {user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: 'https://forge.test'};
const absent = {repository: {id: '42', owner: 'alice', name: 'repo'}, items: []};
const reservation = {id: project, repository_id: '42', provisioned: true};
const detail = {environment: reservation, observed: {id: project, running: true}, native_unavailable: false, authority_unavailable: false, login: 'alice-original'};
const json = (body, status = 200) => new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json; charset=utf-8'}});
const tick = () => new Promise(resolve => setTimeout(resolve, 0));
async function fixture(t, options = {}) {
  const dom = new JSDOM(`<div class="repo-header"><div class="repo-buttons"><button id="native">Native action</button></div></div>${footer}`, {
    url: `https://forge.test/alice/repo${options.hash || ''}`, runScripts: 'outside-only', pretendToBeVisual: true,
  });
  t.after(() => dom.window.close());
  const w = dom.window;
  Object.assign(w, {TextDecoder, AbortController, AbortSignal});
  const $ = name => w.document.getElementById(`sodaspaces-${name}`);
  Object.assign($('root').dataset, {subUrl: '', repositoryId: '42', signed: 'true', userId: '1', ...options.context});
  if (options.noRow) w.document.querySelector('.repo-buttons').remove();
  if (options.duplicate) w.document.body.append($('root').cloneNode(true));
  w.HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  w.HTMLDialogElement.prototype.close = function () { this.open = false; this.dispatchEvent(new w.Event('close')); };
  const calls = [];
  w.fetch = async (url, init = {}) => {
    calls.push({url, ...init});
    assert(url.startsWith('/-/soda/'));
    assert.equal(init.credentials, 'same-origin');
    assert.equal(init.cache, 'no-store');
    assert.equal(init.redirect, 'error');
    if (options.fetch) return options.fetch(url, init, calls.length);
    if (url.endsWith('/logout')) return new Response(null, {status: 204});
    if (url.endsWith('/session')) return json(options.session || session);
    assert.equal(init.headers['X-Soda-Expected-User-ID'], options.context?.userId || '1');
    if (url.endsWith('/forgejo/me')) return json(options.provider || {id: '1', login: 'alice'});
    if (url.includes('?repository_id=')) return json(options.collection || absent);
    if (url.endsWith(project)) return json(options.detail || detail);
    throw new Error('Unexpected fixture request');
  };
  w.eval(script);
  await tick();
  return {w, $, calls, async open() { $('button').click(); await tick(); await tick(); }, async refresh() { $('refresh').click(); await tick(); await tick(); }};
}

test('mounts once, leaves native action intact, reads only on open/refresh/reopen', async t => {
  const f = await fixture(t);
  assert.equal(f.calls.length, 0);
  f.w.eval(script);
  assert.equal(f.w.document.querySelectorAll('.repo-buttons #sodaspaces-button').length, 1);
  assert(f.w.document.getElementById('native'));
  await f.open();
  assert.equal(f.calls.length, 3);
  assert.equal(f.$('status').textContent, 'No shared environment.');
  assert.equal(f.$('data').getAttribute('aria-busy'), 'false');
  await f.refresh();
  f.$('close').click();
  assert.equal(f.$('actor').textContent, '');
  await f.open();
  assert.equal(f.calls.length, 9);
  assert(f.calls.every(c => !c.method));
  assert.equal(f.$('root').querySelectorAll('input,textarea,form').length, 0);
});

for (const options of [{noRow: true}, {duplicate: true}, {context: {repositoryId: '042'}}, {context: {userId: '0'}}, {context: {userId: '9223372036854775808'}}, {context: {subUrl: '/native'}}]) {
  test(`refuses unavailable/invalid mount ${JSON.stringify(options)}`, async t => {
    const f = await fixture(t, options);
    assert(f.$('root').hidden);
    assert.equal(f.calls.length, 0);
  });
}

test('large IDs remain strings through provider/header/query; ignores unrelated fragment', async t => {
  const huge = '9223372036854775807';
  const f = await fixture(t, {hash: '#native-comment', context: {repositoryId: huge, userId: huge}, session: {...session, user: {id: huge, login: 'alice'}}, provider: {id: huge, login: 'alice'}, collection: {...absent, repository: {...absent.repository, id: huge}}});
  assert(!f.$('drawer').open);
  await f.open();
  assert.equal(f.calls.at(-1).url, `/-/soda/api/environments?repository_id=${huge}`);
  assert.equal(f.calls.at(-1).headers['X-Soda-Expected-User-ID'], huge);
});

test('only exact OAuth fragment opens; still checks identity', async t => {
  const f = await fixture(t, {hash: '#sodaspaces'});
  await tick(); await tick();
  assert(f.$('drawer').open);
  assert.equal(f.calls.length, 3);
});

for (const context of [{signed: 'false', userId: ''}, {userId: '2'}]) {
  test(`bootstrap does not adopt another actor ${context.signed || 'mismatch'}`, async t => {
    const f = await fixture(t, {context});
    await f.open();
    assert.equal(f.calls.length, 1);
    assert(!f.$('sign-in').hidden);
    const href = new URL(f.$('sign-in').href);
    assert.equal(href.pathname, '/-/soda/login');
    assert.equal(href.searchParams.get('repository_id'), '42');
    assert.equal(href.searchParams.get('expected_user_id'), context.signed === 'false' ? null : '2');
    f.$('sign-out').click(); await tick();
    const call = f.calls.at(-1);
    assert.equal(call.url, '/-/soda/api/session/logout');
    assert.equal(call.headers['X-Soda-Expected-User-ID'], '1');
    assert.equal(call.headers['X-CSRF-Token'], 'synthetic-csrf');
    assert.equal(call.body, '{}');
    assert.match(f.$('status').textContent, /Signed out of Soda only/);
  });
}

test('fresh provider disagreement blocks repository reads', async t => {
  const f = await fixture(t, {provider: {id: '2', login: 'bob'}});
  await f.open();
  assert.equal(f.calls.length, 2);
  assert(!f.$('sign-in').hidden);
  assert(f.$('sign-out').hidden);
});

for (const [code, status, expected] of [['unauthenticated', 401, /Sign in/], ['consent_required', 403, /consent/], ['identity_mismatch', 403, /Sign in/], ['provider_forbidden', 403, /denied/], ['provider_not_found', 404, /not visible/], ['provider_conflict', 409, /current state/], ['provider_unavailable', 503, /unavailable/]]) {
  test(`sanitized ${status} ${code}`, async t => {
    const f = await fixture(t, {fetch: () => json({error: {code, message: '<script>private provider body</script>'}}, status)});
    await f.open();
    assert.match(f.$('status').textContent, expected);
    assert(!f.$('root').textContent.includes('private provider body'));
    assert.equal(f.calls.length, 1);
  });
}

for (const [name, response] of [
  ['HTML', () => new Response('<html>native login</html>', {headers: {'Content-Type': 'text/html'}})],
  ['invalid JSON', () => new Response('{', {headers: {'Content-Type': 'application/json'}})],
  ['oversized actual stream', () => new Response(' '.repeat(65537), {headers: {'Content-Type': 'application/json', 'Content-Length': '1'}})],
  ['wrong origin', () => json({...session, forgejo_url: 'https://other.test'})],
  ['wrong fields', () => json({user: {id: 1}})],
  ['network', () => { throw new Error('secret network detail'); }],
]) test(`rejects ${name}`, async t => {
  const f = await fixture(t, {fetch: response});
  await f.open();
  assert.match(f.$('status').textContent, /unavailable/);
  assert.equal(f.$('actor').textContent, '');
  assert.equal(f.calls.length, 1);
});

for (const [name, changes, expected] of [
  ['running', {}, /running/], ['stopped', {observed: {id: project, running: false}}, /stopped/],
  ['incomplete', {environment: {...reservation, provisioned: false}}, /incomplete/],
  ['unknown', {observed: null}, /unavailable/], ['native error', {native_unavailable: true}, /unavailable/],
  ['authority error', {authority_unavailable: true}, /running/],
]) test(`existing ${name}, original login and text-only repository labels`, async t => {
  const f = await fixture(t, {collection: {...absent, repository: {...absent.repository, name: '<img src=x onerror=evil>'}, items: [reservation]}, detail: {...detail, ...changes}});
  await f.open();
  assert.match(f.$('status').textContent, expected);
  assert.equal(f.$('login').textContent, 'Your project login: alice-original');
  assert.equal(f.$('repository').querySelector('img'), null);
  assert.equal(f.calls.length, 4);
  assert.equal(f.$('warning').textContent !== '', Boolean(changes.authority_unavailable));
});

test('cross-repository collection and detail never render', async t => {
  for (const options of [{collection: {...absent, repository: {...absent.repository, id: '43'}}}, {collection: {...absent, items: [reservation]}, detail: {...detail, environment: {...reservation, repository_id: '43'}}}]) {
    const f = await fixture(t, options); await f.open();
    assert.match(f.$('status').textContent, /unavailable/);
    assert.equal(f.$('environment').textContent, '');
  }
});

for (const event of ['blur', 'pagehide', 'pageshow', 'visibilitychange']) test(`stale ${event} clears data, blocks reopen/refresh without forced navigation`, async t => {
  const f = await fixture(t); await f.open();
  if (event === 'visibilitychange') {
    Object.defineProperty(f.w.document, 'hidden', {value: true});
    f.w.document.dispatchEvent(new f.w.Event(event));
  } else f.w.dispatchEvent(event === 'pageshow' ? new f.w.PageTransitionEvent(event, {persisted: true}) : new f.w.Event(event));
  assert.equal(f.$('actor').textContent, '');
  assert(!f.$('reload').hidden);
  assert(f.$('refresh').disabled);
  const count = f.calls.length;
  f.$('close').click(); await f.open(); await f.refresh();
  assert.equal(f.calls.length, count);
  assert.match(f.$('status').textContent, /Reload/);
  assert.equal(f.w.location.pathname, '/alice/repo');
});

test('normal pageshow, window focus and interior clicks do not invalidate or close', async t => {
  const f = await fixture(t); await f.open();
  f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow', {persisted: false}));
  f.w.dispatchEvent(new f.w.Event('focus'));
  f.$('data').click();
  assert(f.$('drawer').open);
  assert(f.$('reload').hidden);
  f.$('drawer').click();
  assert(!f.$('drawer').open);
});

test('late closed read cannot replace reopened result even if fetch ignores abort', async t => {
  let complete;
  const f = await fixture(t, {fetch: (_url, _init, count) => count === 1 ? new Promise(resolve => { complete = resolve; }) : json({error: {code: 'unauthenticated'}}, 401)});
  await f.open(); f.$('close').click(); await f.open();
  assert(f.calls[0].signal.aborted);
  complete(json(session)); await tick(); await tick();
  assert.equal(f.calls.length, 2);
  assert.equal(f.$('actor').textContent, '');
  assert.match(f.$('status').textContent, /Sign in/);
});

test('closing during logout does not replay or abort its mutation; reopen waits', async t => {
  let complete;
  const f = await fixture(t, {fetch: (url) => url.endsWith('/logout') ? new Promise(resolve => { complete = resolve; }) : url.endsWith('/session') ? json(session) : url.endsWith('/forgejo/me') ? json({id: '1', login: 'alice'}) : json(absent)});
  await f.open(); f.$('sign-out').click(); f.$('close').click(); await f.open();
  assert.equal(f.calls.length, 4);
  assert.equal(f.calls[3].signal.aborted, false);
  complete(new Response(null, {status: 204})); await tick();
  assert.match(f.$('status').textContent, /Signed out of Soda only/);
  assert(f.$('sign-out').hidden);
});

test('queued close from a previous opening cannot clear a new result', async t => {
  const f = await fixture(t); await f.open();
  f.$('drawer').open = false; // Native close event is queued, unlike the fixture double.
  await f.open();
  f.$('drawer').dispatchEvent(new f.w.Event('close'));
  assert.match(f.$('status').textContent, /No shared environment/);
  assert.match(f.$('actor').textContent, /alice/);
});

test('Escape discards pending reads synchronously', async t => {
  let complete;
  const f = await fixture(t, {fetch: () => new Promise(resolve => { complete = resolve; })});
  await f.open();
  f.$('drawer').dispatchEvent(new f.w.Event('cancel'));
  assert(f.calls[0].signal.aborted);
  complete(json(session)); await tick();
  assert.equal(f.$('actor').textContent, '');
  assert.equal(f.calls.length, 1);
});

test('unconfirmed logout loses credentials and requires explicit fresh read, not replay', async t => {
  const f = await fixture(t, {fetch: url => url.endsWith('/logout') ? Promise.reject(new Error('lost response')) : url.endsWith('/session') ? json(session) : url.endsWith('/forgejo/me') ? json({id: '1', login: 'alice'}) : json(absent)});
  await f.open(); f.$('sign-out').click(); await tick();
  assert.match(f.$('status').textContent, /not confirmed/);
  f.$('sign-out').click(); await tick();
  assert.equal(f.calls.length, 4);
  assert.equal(f.$('actor').textContent, '');
});
