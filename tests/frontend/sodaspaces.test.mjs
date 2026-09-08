// Native shell + actual component; DOM/HTTP doubles, not installed browser proof.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {mountDrawer} from '../../appliance/forgejo/public/assets/sodaspaces.js';
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
const {JSDOM} = require('jsdom');
const footer = readFileSync(new URL('../../appliance/forgejo/templates/custom/footer.tmpl', import.meta.url), 'utf8');
const json = (body, status = 200) => new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});
const tick = () => new Promise(resolve => setTimeout(resolve, 0));
async function fixture(t, options = {}) {
  const dom = new JSDOM(`<div class="repo-header"><div class="repo-buttons"><button id="native">Native</button></div></div><form><input id="native-edit" value="dirty"></form>${footer}`, {url: `https://forge.test/alice/repo${options.hash || ''}`, pretendToBeVisual: true});
  const w = dom.window, doc = w.document;
  const $ = n => doc.getElementById('sodaspaces-' + n);
  Object.assign($('root').dataset, {subUrl: '', signed: 'true', userId: '1', repositoryId: '7', ...options.context});
  w.HTMLDialogElement.prototype.showModal = function () {this.open = true;};
  w.HTMLDialogElement.prototype.close = function () {this.open = false; this.dispatchEvent(new w.Event('close'));};
  const calls = [];
  w.fetch = async (url, init) => {
    calls.push({url, ...init});
    if (options.fetch) return options.fetch(url, init);
    if (url.endsWith('/api/session')) return json({user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: w.location.origin});
    if (url.endsWith('/api/forgejo/me')) return json({id: '1', login: 'alice'});
    return json({items: [], repository: {id: '7', owner: 'alice', name: 'repo'}, can_create: true});
  };
  if (options.noRow) doc.querySelector('.repo-buttons').remove();
  if (options.duplicate) doc.body.append($('root').cloneNode(true));
  const api = mountDrawer(doc, options.mount);
  t.after(() => {api?.dispose(); w.close();});
  return {w, doc, $, calls, api, button: text => [...$('content').querySelectorAll('button,a')].find(b => b.textContent === text), async open() {$('button').click(); await tick(); await tick();}};
}
test('native mount is inert, unique, and preserves native forms/actions', async t => {
  const f = await fixture(t); assert.equal(f.calls.length, 0); mountDrawer(f.doc);
  assert.equal(f.doc.querySelectorAll('.repo-buttons #sodaspaces-button').length, 1);
  await f.open(); assert.equal(f.calls.length, 3); assert(f.calls.every(c => c.method === 'GET'));
  assert.equal(f.doc.getElementById('native-edit').value, 'dirty'); assert(f.doc.getElementById('native'));
  assert(!f.button('Create environment').hidden); assert(!f.$('content').querySelector('.soda-page'));
});
for (const options of [{noRow: true}, {duplicate: true}, {context: {subUrl: '/unsupported'}}, {context: {userId: '0'}}, {context: {repositoryId: '07'}}, {context: {signed: 'false', userId: '1'}}]) {
  test('invalid/ambiguous native context does not mount ' + JSON.stringify(options), async t => {
    const f = await fixture(t, options); assert.equal(f.api, undefined); assert.equal(f.calls.length, 0);
  });
}
test('anonymous page offers explicit contextual OAuth without substituting Soda identity', async t => {
  const f = await fixture(t, {context: {signed: 'false', userId: ''}}); await f.open();
  assert.equal(f.calls.length, 1); assert(!f.button('Connect to Soda').hidden);
  assert.equal(f.button('Connect to Soda').getAttribute('href'), '/-/soda/login?repository_id=7');
  assert(f.button('Create environment').hidden);
});
test('only exact OAuth fragment opens, and large IDs stay strings', async t => {
  const contexts = [];
  const f = await fixture(t, {hash: '#sodaspaces', context: {userId: '9007199254740993'}, mount: (_root, context) => {contexts.push(context); return {refresh() {}, dispose() {}};}});
  assert(f.$('drawer').open); assert.equal(contexts[0].expectedUserId, '9007199254740993');
});
for (const event of ['blur', 'pagehide', 'bfcache', 'close', 'cancel', 'backdrop']) {
  test(event + ' disposes before reopening, without bypassing full-page reload', async t => {
    let disposed = 0, mounted = 0;
    const f = await fixture(t, {mount: () => {mounted++; return {refresh() {}, dispose() {disposed++;}};}}); await f.open();
    if (event === 'close') f.$('close').click();
    else if (event === 'cancel') f.$('drawer').dispatchEvent(new f.w.Event('cancel'));
    else if (event === 'backdrop') f.$('drawer').click();
    else if (event === 'bfcache') f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow', {persisted: true}));
    else f.w.dispatchEvent(new f.w.Event(event));
    assert.equal(disposed, 1); f.$('drawer').close(); await f.open();
    assert.equal(mounted, 1); assert(f.button('Reload repository page'));
    assert.equal(f.doc.getElementById('native-edit').value, 'dirty');
  });
}
test('late fetch after close cannot repaint or dispatch another request', async t => {
  let release;
  const f = await fixture(t, {fetch: () => new Promise(resolve => {release = resolve;})});
  f.$('button').click(); f.$('close').click(); await f.open();
  release(json({user: {id: '1'}, csrf_token: 'synthetic-csrf', forgejo_url: f.w.location.origin})); await tick();
  assert.equal(f.calls.length, 1); assert(f.button('Reload repository page')); assert(!f.button('Create environment'));
});
test('interior clicks, initial pageshow and focus preserve the current drawer', async t => {
  const f = await fixture(t); await f.open(); const count = f.calls.length;
  f.$('content').click(); f.w.dispatchEvent(new f.w.Event('focus')); f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow'));
  assert(f.$('drawer').open); assert.equal(f.calls.length, count); assert(f.button('Create environment'));
});
