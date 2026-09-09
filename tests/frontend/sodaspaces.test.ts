// Native shell + actual component; DOM/HTTP doubles, not installed browser proof.
import test, {type TestContext} from 'node:test';
import type { DrawerContext } from '../../appliance/forgejo/public/assets/sodaspaces-drawer.ts';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {mountDrawer} from '../../appliance/forgejo/public/assets/sodaspaces.ts';
import {JSDOM} from 'jsdom';
const footer = readFileSync(new URL('../../appliance/forgejo/templates/custom/footer.tmpl', import.meta.url), 'utf8');
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});
const tick = () => new Promise(resolve => setTimeout(resolve, 0));
async function fixture(t: TestContext, options: {saved?: {repositoryId: string; open: boolean; width?: number}; hash?: string; context?: Record<string,string>; noRow?: boolean; duplicate?: boolean; fetch?: (url: string, init: RequestInit) => Promise<Response>; mount?: Parameters<typeof mountDrawer>[1]} = {}) {
  const dom = new JSDOM(`<div class="repo-header"><div class="repo-buttons"><button id="native">Native</button></div></div><form><input id="native-edit" value="dirty"></form>${footer}`, {url: `https://forge.test/alice/repo${options.hash || ''}`, pretendToBeVisual: true});
  const w = dom.window, doc = w.document;
  const $ = (n: string) => { const element = doc.getElementById('sodaspaces-' + n); assert(element); return element; };
  Object.assign($('root').dataset, {subUrl: '', signed: 'true', userId: '1', repositoryId: '7', ...options.context});
  const calls: Array<RequestInit & {url: string}> = [];
  Object.defineProperty(w, 'fetch', {value: async (url: string, init: RequestInit) => {
    calls.push({url, ...init});
    if (options.fetch) return options.fetch(url, init);
    if (url.endsWith('/api/session')) return json({user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: w.location.origin});
    if (url.endsWith('/api/forgejo/me')) return json({id: '1', login: 'alice'});
    return json({items: [], repository: {id: '7', owner: 'alice', name: 'repo'}, can_create: true});
  }});
  if (options.saved) w.sessionStorage.setItem('soda-workspace:1', JSON.stringify(options.saved));
  if (options.noRow) doc.querySelector('.repo-buttons')?.remove();
  if (options.duplicate) doc.body.append($('root').cloneNode(true));
  const api = mountDrawer(doc, options.mount);
  t.after(() => {api?.dispose(); w.close();});
  return {w, doc, $, calls, api, button: (text: string) => { const button = [...$('content').querySelectorAll<HTMLElement>('button,a')].find(b => b.textContent === text); assert(button); return button; }, async open() {$('button').click(); await tick(); await tick();}};
}
test('native mount is inert, unique, and preserves native forms/actions', async t => {
  const f = await fixture(t); assert.equal(f.calls.length, 0); mountDrawer(f.doc);
  assert.equal(f.doc.querySelectorAll('.repo-buttons #sodaspaces-button').length, 1);
  await f.open(); assert.equal(f.calls.length, 3); assert(f.calls.every(c => c.method === 'GET'));
  assert.equal(f.doc.querySelector<HTMLInputElement>('#native-edit')?.value, 'dirty'); assert(f.doc.getElementById('native'));
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
  const contexts: DrawerContext[] = [];
  const f = await fixture(t, {hash: '#sodaspaces', context: {userId: '9007199254740993'}, mount: (_root, context) => {contexts.push(context); return {refresh() {}, dispose() {}};}});
  assert(!f.$('drawer').hidden); assert.equal(contexts[0]?.expectedUserId, '9007199254740993');
});
for (const event of ['pagehide', 'bfcache']) {
  test(event + ' detaches and fresh-mounts only after document restoration', async t => {
    let disposed = 0, mounted = 0;
    const f = await fixture(t, {mount: () => {mounted++; return {refresh() {}, dispose() {disposed++;}};}}); await f.open();
    if (event === 'bfcache') f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow', {persisted: true}));
    else f.w.dispatchEvent(new f.w.Event(event));
    assert.equal(disposed, 1);
    assert.equal(mounted, event === 'bfcache' ? 2 : 1);
    if (event === 'pagehide') f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow', {persisted: true}));
    assert.equal(mounted, 2); assert(![...f.$('content').querySelectorAll('button')].some(b => b.textContent === 'Reload repository page'));
    assert.equal(f.doc.querySelector<HTMLInputElement>('#native-edit')?.value, 'dirty');
  });
}
test('late fetch after document departure cannot repaint or dispatch another request', async t => {
  let release: ((response: Response) => void) | undefined;
  const f = await fixture(t, {fetch: () => new Promise(resolve => {release = resolve;})});
  f.$('button').click(); f.w.dispatchEvent(new f.w.Event('pagehide')); f.$('close').click(); await f.open();
  assert(release); release(json({user: {id: '1'}, csrf_token: 'synthetic-csrf', forgejo_url: f.w.location.origin})); await tick();
  assert.equal(f.calls.length, 1); assert(![...f.$('content').querySelectorAll('button')].some(b => b.textContent === 'Create environment'));
});
test('non-modal native interaction, blur and hide/reopen preserve one mounted component', async t => {
  let mounted = 0, disposed = 0, refreshed = 0;
  const f = await fixture(t, {mount: () => {mounted++; return {refresh() {refreshed++;}, dispose() {disposed++;}};}});
  await f.open();
  assert.equal(f.$('drawer').tagName, 'ASIDE'); assert(!f.doc.querySelector('[inert], [aria-modal=true]'));
  f.doc.getElementById('native-edit')?.focus(); const edit = f.doc.querySelector<HTMLInputElement>('#native-edit'); assert(edit); edit.value = 'still editable';
  f.doc.getElementById('native')?.click(); f.$('drawer').click();
  f.w.dispatchEvent(new f.w.Event('blur')); f.doc.dispatchEvent(new f.w.Event('visibilitychange'));
  assert(!f.$('drawer').hidden); assert(f.doc.body.classList.contains('sodaspaces-open'));
  f.$('close').click(); assert(f.$('drawer').hidden); assert(!f.doc.body.classList.contains('sodaspaces-open'));
  await f.open(); assert.equal(mounted, 1); assert.equal(refreshed, 1); assert.equal(disposed, 0);
  assert.equal(f.doc.querySelector<HTMLInputElement>('#native-edit')?.value, 'still editable');
});
test('workspace separator supports keyboard resizing without native navigation', async t => {
  const f = await fixture(t); await f.open();
  f.$('divider').dispatchEvent(new f.w.KeyboardEvent('keydown', {key: 'ArrowLeft', bubbles: true, cancelable: true}));
  assert.equal(f.$('divider').getAttribute('aria-valuenow'), '55');
  assert.equal(f.doc.body.style.getPropertyValue('--soda-space-width'), '55vw');
});
test('navigation restores selected repository instead of retargeting to the left page', async t => {
  const contexts: DrawerContext[] = []; let disposed = 0;
  const f = await fixture(t, {context: {repositoryId: '8'}, saved: {repositoryId: '7', open: true, width: 55}, mount: (_root, context) => {contexts.push(context); return {refresh() {}, dispose() {disposed++;}};}});
  assert(!f.$('drawer').hidden); assert.equal(contexts[0]?.repositoryId, '7');
  assert.equal(f.doc.body.style.getPropertyValue('--soda-space-width'), '55vw');
  const select = [...f.doc.querySelectorAll('button')].find(b => b.textContent === 'Use repository on the left');
  assert(select); select.click(); assert.equal(disposed, 1); assert.equal(contexts[1]?.repositoryId, '8');
});
test('signed non-repository page resumes only a saved workspace locator', async t => {
  const contexts: DrawerContext[] = [];
  const f = await fixture(t, {noRow: true, context: {repositoryId: ''}, saved: {repositoryId: '7', open: true}, mount: (_root, context) => {contexts.push(context); return {refresh() {}, dispose() {}};}});
  assert.equal(contexts[0]?.repositoryId, '7'); assert(!f.$('drawer').hidden);
  assert(f.$('button').classList.contains('sodaspaces-global-resume'));
});
test('another native actor does not restore a previous user workspace', async t => {
  const f = await fixture(t, {context: {userId: '2'}, saved: {repositoryId: '8', open: true}});
  assert(f.$('drawer').hidden); assert.equal(f.calls.length, 0);
});
test('interior clicks, initial pageshow and focus preserve the current drawer', async t => {
  const f = await fixture(t); await f.open(); const count = f.calls.length;
  f.$('content').click(); f.w.dispatchEvent(new f.w.Event('focus')); f.w.dispatchEvent(new f.w.PageTransitionEvent('pageshow'));
  assert(!f.$('drawer').hidden); assert.equal(f.calls.length, count); assert(f.button('Create environment'));
});
