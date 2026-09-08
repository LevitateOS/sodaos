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
const absent = {repository: {id: '42', owner: 'alice', name: 'repo'}, items: [], can_create: true};
const key = {id: '1', public_key: `ssh-ed25519 ${'A'.repeat(68)}`, fingerprint: `SHA256:${'A'.repeat(43)}`};
const ownConnection = {login: 'alice-original', routing_verified: false, connection: {
  environment: {id: project, ip: '10.89.0.2', running: true}, host_key: key.public_key + '\n', fingerprint: key.fingerprint,
}};
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
    if (url.endsWith('/development-keys')) return json(options.keys || {items: [key]});
    if (url.endsWith('/connection')) return json(options.connection || ownConnection);
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
  assert.equal(f.$('root').querySelectorAll('form,input[type="file"]').length, 0);
  assert(f.$('keys').hidden && f.$('connection').hidden);
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
  assert.equal(f.calls.length, ['running', 'authority error'].includes(name) ? 5 : 4);
  assert.equal(f.$('warning').textContent !== '', Boolean(changes.authority_unavailable));
});

test('fresh provider rename updates the label without remapping the project login', async t => {
  const f = await fixture(t, {provider: {id: '1', login: 'alice-renamed'}, collection: {...absent, items: [reservation]}});
  await f.open();
  assert.equal(f.$('actor').textContent, 'Soda account: alice-renamed (ID 1)');
  assert.equal(f.$('login').textContent, 'Your project login: alice-original');
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

// Mutation doubles exercise dispatch and persistence observations, not native accounts.
async function accessFixture(t, options = {}) {
  let created = Boolean(options.created), joined = false, registered = options.registered ? [key] : [];
  return fixture(t, {fetch: async (url, init, count) => {
    if (init.method === 'POST') {
      assert.equal(init.headers['X-Soda-Expected-User-ID'], '1');
      assert.equal(init.headers['X-CSRF-Token'], 'synthetic-csrf');
      assert.equal(init.headers['Content-Type'], 'application/json');
      if (options.post) return options.post(url, init);
      if (url.endsWith('/environments')) {
        assert.deepEqual(JSON.parse(init.body), {repository_id: '42'});
        created = true;
        return json(reservation, 201);
      }
      if (url.endsWith('/development-keys')) { registered = [key]; return json({items: registered}); }
      if (url.endsWith('/join')) { assert.equal(init.body, '{}'); joined = true; return json({login: 'alice-original'}); }
      throw new Error('Unexpected write');
    }
    if (options.read) {
      const response = options.read(url, init, count);
      if (response) return response;
    }
    if (url.endsWith('/session')) return json(session);
    if (url.endsWith('/forgejo/me')) return json({id: '1', login: 'alice'});
    if (url.includes('?repository_id=')) return json({...absent, can_create: !created, items: created ? [reservation] : []});
    if (url.endsWith(project)) return json({...detail, login: joined ? 'alice-original' : ''});
    if (url.endsWith('/development-keys')) return json({items: registered});
    if (url.endsWith('/connection')) return json(ownConnection);
    throw new Error('Unexpected read');
  }});
}
const settle = async () => { await tick(); await tick(); };
const writes = f => f.calls.filter(c => c.method === 'POST');

test('explicit stable-ID create, key save and join are separate, then own native Copy target', async t => {
  const f = await accessFixture(t);
  await f.open();
  assert(!f.$('create').hidden);
  f.$('create').click(); f.$('create').click(); await settle();
  assert.equal(writes(f).length, 1);
  assert(f.$('create').hidden && !f.$('keys').hidden && f.$('join').hidden);
  assert.match(f.$('result').textContent, /does not join/);
  f.$('public-key').value = key.public_key;
  f.$('save-key').click(); f.$('save-key').click(); await settle();
  assert.equal(writes(f).length, 2);
  assert.equal(f.$('key-list').textContent, key.fingerprint);
  assert(!f.$('join').hidden && f.$('connection').hidden);
  f.$('join').click(); f.$('join').click(); await settle();
  assert.equal(writes(f).length, 3);
  assert(!f.$('connection').hidden && f.$('keys').hidden && f.$('join').hidden);
  assert.equal(f.$('command').value, 'ssh alice-original@10.89.0.2');
  assert(f.$('command').readOnly);
  assert.equal(f.$('copy').getAttribute('data-clipboard-target'), '#sodaspaces-command');
  assert.match(f.$('fingerprint').textContent, /SHA256:/);
  await f.refresh(); f.$('close').click(); await f.open();
  assert.equal(writes(f).length, 3);
});

for (const value of ['', '-----BEGIN OPENSSH PRIVATE KEY----- secret', 'command="evil" ssh-ed25519 AAAA', `${key.public_key}\n${key.public_key}`]) {
  test('key form refuses empty/private/options/multiple input before transmission: ' + value.slice(0, 12), async t => {
    const f = await accessFixture(t, {created: true}); await f.open();
    const before = f.calls.length;
    f.$('public-key').value = value; f.$('save-key').click(); await settle();
    assert.equal(f.calls.length, before);
    assert.match(f.$('result').textContent, /public SSH key/);
  });
}

test('nonowner cannot dispatch hidden create; malformed advisory fails closed', async t => {
  for (const can_create of [false, 'true', undefined]) {
    const f = await fixture(t, {collection: {...absent, can_create}}); await f.open();
    f.$('create').click(); await settle();
    assert.equal(writes(f).length, 0);
    assert(f.$('create').hidden);
  }
});

for (const changed of ['session', 'provider', 'blur', 'close']) test(`action-time ${changed} change prevents dispatch`, async t => {
  let enabled = false, finish;
  const f = await accessFixture(t, {read: url => {
    if (!enabled) return;
    if (url.endsWith('/session') && changed === 'session') return json({...session, user: {id: '2', login: 'bob'}});
    if (url.endsWith('/forgejo/me') && changed === 'provider') return json({id: '2'});
    if (url.endsWith('/forgejo/me') && ['blur', 'close'].includes(changed)) return new Promise(resolve => { finish = resolve; });
  }});
  await f.open(); enabled = true;
  f.$('create').click(); await settle();
  if (changed === 'blur') f.w.dispatchEvent(new f.w.Event('blur'));
  if (changed === 'close') f.$('close').click();
  if (finish) finish(json({id: '1', login: 'alice'}));
  await settle();
  assert.equal(writes(f).length, 0);
});

for (const transition of ['close', 'blur', 'pageshow']) test(`pending mutation survives ${transition}, no late rendering or replay`, async t => {
  let finish;
  const f = await accessFixture(t, {post: () => new Promise(resolve => { finish = resolve; })});
  await f.open(); f.$('create').click(); await settle();
  assert.equal(writes(f).length, 1);
  if (transition === 'close') { f.$('close').click(); await f.open(); }
  else if (transition === 'blur') f.w.dispatchEvent(new f.w.Event('blur'));
  else f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow', {persisted: true}));
  f.$('create').click(); f.$('sign-out').click(); await f.refresh();
  assert.equal(writes(f).length, 1);
  assert.equal(writes(f)[0].signal.aborted, false);
  finish(json(reservation, 201)); await settle();
  assert.equal(f.$('command').value, '');
  assert(!f.$('result').textContent.includes('Environment created'));
  assert.equal(writes(f).length, 1);
});

for (const mode of ['lost', 'HTML', 'wrong association', 'native failure', 'persistence failure']) test(`uncertain creation ${mode} is observed, never replayed`, async t => {
  const f = await accessFixture(t, {post: () => {
    if (mode === 'lost') throw new Error('secret transport detail');
    if (mode === 'HTML') return new Response('<html>secret</html>');
    if (mode === 'wrong association') return json({...reservation, repository_id: '99'}, 201);
    return json({error: {code: mode === 'native failure' ? 'provisioning_incomplete' : 'result_not_saved', message: 'secret native output'}}, mode === 'native failure' ? 502 : 503);
  }});
  await f.open(); f.$('create').click(); await settle();
  assert.match(f.$('result').textContent, /not confirmed/);
  assert(!f.$('root').textContent.includes('secret'));
  await f.refresh(); f.$('close').click(); await f.open(); f.$('create').click();
  assert.equal(writes(f).length, 1);
  assert(f.$('create').disabled);
});

for (const code of ['account_incomplete', 'membership_not_saved']) test(`join ${code} never claims membership or repeats the helper`, async t => {
  const f = await accessFixture(t, {created: true, registered: true, post: () => json({error: {code}}, 503)});
  await f.open(); f.$('join').click(); await settle();
  assert.equal(writes(f).length, 1);
  assert(f.$('connection').hidden && f.$('join').disabled);
  assert.match(f.$('result').textContent, /account may exist/);
  await f.refresh(); f.$('join').click();
  assert.equal(writes(f).length, 1);
});

for (const [code, status] of [['owner_required', 403], ['invalid_csrf', 403], ['reservation_failed', 409], ['unsupported_linux_login', 422], ['development_key_required', 422], ['invalid_public_key', 400]]) test(`confirmed rejection ${code} has no false success or automatic retry`, async t => {
  const f = await accessFixture(t, {post: () => json({error: {code, message: 'secret provider detail'}}, status)});
  await f.open(); f.$('create').click(); await settle();
  assert.equal(writes(f).length, 1);
  assert(f.$('connection').hidden);
  assert(!f.$('result').textContent.includes('secret'));
  await f.refresh(); assert.equal(writes(f).length, 1);
});

for (const changes of [
  {login: 'bob'}, {login: 'root'}, {routing_verified: true},
  {connection: {...ownConnection.connection, environment: {id: project, running: false}}},
  {connection: {...ownConnection.connection, environment: {id: project, running: true, ip: '127.0.0.1'}}},
  {connection: {...ownConnection.connection, environment: {id: project, running: true, ip: '10.0.0.2 -oProxyCommand=evil'}}},
  {connection: {...ownConnection.connection, fingerprint: 'bad'}},
]) test('invalid/unavailable own connection cannot expose a Copy command ' + JSON.stringify(changes), async t => {
  const f = await fixture(t, {collection: {...absent, items: [reservation]}, connection: {...ownConnection, ...changes}});
  await f.open();
  assert(f.$('connection').hidden && f.$('copy').disabled);
  assert.equal(f.$('command').value, '');
  assert.equal(f.$('copy').getAttribute('data-clipboard-target'), null);
});

test('post-action reread with a changed Soda actor cannot display the old result', async t => {
  let finish, changed = false;
  const f = await accessFixture(t, {
    post: () => new Promise(resolve => { finish = resolve; }),
    read: url => changed && url.endsWith('/session') ? json({...session, user: {id: '2', login: 'bob'}}) : undefined,
  });
  await f.open(); f.$('create').click(); await settle();
  changed = true; finish(json(reservation, 201)); await settle();
  assert.equal(writes(f).length, 1);
  assert.equal(f.$('result').textContent, '');
  assert.match(f.$('status').textContent, /identities do not match/);
});

test('queued close cannot let a pending mutation render into a reopened drawer', async t => {
  let finish;
  const f = await accessFixture(t, {post: () => new Promise(resolve => { finish = resolve; })});
  await f.open(); f.$('create').click(); await settle();
  f.$('drawer').close = function () { this.open = false; }; // Native close event queues.
  f.$('close').click(); await f.open();
  f.$('drawer').dispatchEvent(new f.w.Event('close'));
  finish(json(reservation, 201)); await settle();
  assert.equal(writes(f).length, 1);
  assert.equal(f.$('environment').textContent, '');
  assert.match(f.$('status').textContent, /Refresh to inspect actual state/);
});

test('stable-ID create preserves the full signed-int64 decimal body', async t => {
  const huge = '9223372036854775807';
  const f = await fixture(t, {context: {repositoryId: huge}, fetch: (url, init) => {
    if (init.method === 'POST') {
      assert.deepEqual(JSON.parse(init.body), {repository_id: huge});
      return json({...reservation, repository_id: huge}, 201);
    }
    if (url.endsWith('/session')) return json(session);
    if (url.endsWith('/forgejo/me')) return json({id: '1', login: 'alice'});
    return json({...absent, repository: {...absent.repository, id: huge}});
  }});
  await f.open(); f.$('create').click(); await settle();
  assert.equal(writes(f).length, 1);
});

test('empty successful key result is uncertain, not a confirmed saved key or join', async t => {
  const f = await accessFixture(t, {created: true, post: () => json({items: []})});
  await f.open(); f.$('public-key').value = key.public_key;
  f.$('save-key').click(); await settle();
  assert.match(f.$('result').textContent, /not confirmed/);
  assert(f.$('join').hidden);
  assert.equal(writes(f).length, 1);
});

test('connection actor mismatch clears actor and never retains a Copy target', async t => {
  const f = await accessFixture(t, {created: true, registered: true, read: url => {
    if (url.endsWith('/connection')) return json({error: {code: 'identity_mismatch'}}, 403);
  }});
  await f.open(); f.$('join').click(); await settle();
  assert.equal(f.$('actor').textContent, '');
  assert(f.$('connection').hidden && f.$('sign-out').hidden);
  assert(!f.$('sign-in').hidden);
});

for (const registered of [{items: Array.from({length: 33}, (_, i) => ({...key, id: String(i + 1)}))}, {items: [{...key, id: 1}]}, {items: [{...key, fingerprint: '<script>bad</script>'}]}]) test('excess or malformed key summary cannot enable join', async t => {
  const f = await fixture(t, {collection: {...absent, items: [reservation]}, detail: {...detail, login: ''}, keys: registered});
  await f.open(); f.$('join').click();
  assert(f.$('join').hidden);
  assert.equal(writes(f).length, 0);
  if (registered.items.length > 32) assert.match(f.$('warning').textContent, /at most 32/);
});

test('ambiguous document-wide native clipboard selector fails closed', async t => {
  const f = await fixture(t, {collection: {...absent, items: [reservation]}});
  const extra = f.w.document.createElement('input');
  extra.id = 'sodaspaces-command'; extra.value = 'not our command';
  f.w.document.body.prepend(extra);
  await f.open();
  assert(f.$('connection').hidden && f.$('copy').disabled);
  assert.equal(f.$('copy').getAttribute('data-clipboard-target'), null);
});

test('stale member clears Copy target and public data without changing native forms', async t => {
  const f = await fixture(t, {collection: {...absent, items: [reservation]}});
  const form = f.w.document.createElement('form');
  const input = f.w.document.createElement('input');
  input.value = 'Unsaved native form edit'; form.append(input); f.w.document.body.append(form);
  await f.open();
  assert(!f.$('connection').hidden);
  f.w.dispatchEvent(new f.w.Event('blur'));
  assert.equal(f.$('command').value, '');
  assert.equal(f.$('copy').getAttribute('data-clipboard-target'), null);
  assert(f.$('connection').hidden && f.$('copy').disabled);
  assert.equal(input.value, 'Unsaved native form edit');
});

test('unconfirmed logout loses credentials and requires explicit fresh read, not replay', async t => {
  const f = await fixture(t, {fetch: url => url.endsWith('/logout') ? Promise.reject(new Error('lost response')) : url.endsWith('/session') ? json(session) : url.endsWith('/forgejo/me') ? json({id: '1', login: 'alice'}) : json(absent)});
  await f.open(); f.$('sign-out').click(); await tick();
  assert.match(f.$('status').textContent, /not confirmed/);
  f.$('sign-out').click(); await tick();
  assert.equal(f.calls.length, 4);
  assert.equal(f.$('actor').textContent, '');
});
