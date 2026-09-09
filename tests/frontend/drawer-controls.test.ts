import test, {type TestContext} from 'node:test';
import type { TerminalContext } from '../../appliance/forgejo/public/assets/sodaspaces-terminal.ts';
import assert from 'node:assert/strict';
import {mountSodaspaces} from '../../appliance/forgejo/public/assets/sodaspaces-drawer.ts';
import {JSDOM} from 'jsdom';
const env = 'p0123456789abcdef01234567', fp = 'SHA256:' + 'A'.repeat(43);
const tick = () => new Promise(resolve => setTimeout(resolve, 0));
interface Call { url: string; method: string; body?: string }
interface State { running: boolean; member: boolean; admin: boolean; absent: boolean; saved: string[]; installed: string[] }
interface FakeTerminal { ctx: TerminalContext; started: boolean; disposed: boolean; stale: boolean; dispose(): void; invalidate(): void }
function fixture(t: TestContext, extra: Partial<State> & { fetch?: (url: string, init: Omit<Call,'url'>) => Promise<Response | null> } = {}) {
  const dom = new JSDOM('<button id="native">Native action</button><main></main>', {url: 'https://forge.test/alice/repo', pretendToBeVisual: true});
  const w = dom.window, root = w.document.querySelector('main'); assert(root);
  const calls: Call[] = [], terminals: FakeTerminal[] = [];
  const state = {running: true, member: true, admin: true, absent: false, saved: [fp], installed: [fp], ...extra};
  Object.defineProperty(w, 'fetch', {value: async (url: string, init: Omit<Call,'url'>) => {
    calls.push({url, ...init});
    if (extra.fetch) {const value = await extra.fetch(url, init); if (value) return value;}
    let body;
    if (init.method !== 'GET') {
      const input = JSON.parse(init.body || '{}');
      if (url.endsWith('/api/session/logout')) return new Response(null, {status: 204});
      if (url.endsWith('/api/environments')) body = {id: env, repository_id: '7', provisioned: true};
      else if (url.endsWith('/join')) body = {login: 'alice'};
      else if (url.endsWith('/lifecycle')) body = {environment: {id: env, running: input.action === 'start'}, boot_enabled: input.action === 'start'};
      else if (url.endsWith('/access-keys')) body = {applied: true, login: 'alice', revision: 'a'.repeat(64), installed_fingerprints: input.saved_fingerprints};
      else if (init.method === 'DELETE') body = {removed: true, existing_project_access_changed: false};
      else body = {items: []};
    }
    else if (url.endsWith('/api/session')) body = {user: {id: '1', login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: 'https://forge.test'};
    else if (url.endsWith('/api/forgejo/me')) body = {id: '1'};
    else if (url.includes('/api/environments?')) body = {repository: {id: '7'}, can_create: state.absent, items: state.absent ? [] : [{id: env, repository_id: '7'}]};
    else if (url.endsWith('/api/me/development-keys')) body = {items: state.saved.map((f, i) => ({id: String(i + 1), fingerprint: f}))};
    else if (url.endsWith('/lifecycle')) body = {environment: {id: env, running: state.running}, boot_enabled: state.running};
    else if (url.endsWith('/access-keys')) body = {login: 'alice', revision: 'a'.repeat(64), installed_fingerprints: state.installed, saved_fingerprints: state.saved};
    else if (url.endsWith('/connection')) body = {login: 'alice', connection: {environment: {id: env, running: true, ip: '10.89.0.2'}, fingerprint: fp}};
    else body = {environment: {id: env, repository_id: '7', provisioned: true}, observed: {id: env, running: state.running}, login: state.member ? 'alice' : '', environment_administrator: state.admin, native_unavailable: false, authority_unavailable: false};
    return new Response(JSON.stringify(body), {headers: {'Content-Type': 'application/json'}});
  }});

  const api = mountSodaspaces(root, {expectedUserId: '1', repositoryId: '7'}, (_mount, ctx) => {const t = {ctx, started: false, disposed: false, stale: false, dispose() {this.disposed = true;}, invalidate() {this.stale = true;}}; terminals.push(t); return t;});
  const button = (text: string) => {
    const b = [...root.querySelectorAll('button')].find(b => b.textContent === text);
    assert(b);
    const panel = b.closest('[role=tabpanel]');
    if (panel) root.querySelector<HTMLElement>('#' + panel.getAttribute('aria-labelledby'))?.click();
    return b;
  };
  t.after(() => {api.dispose(); dom.window.close();});
  return {w, root, api, calls, state, terminals, button};
}
test('standalone mount is inert; refresh only reads and preserves native nodes', async t => {
  const f = fixture(t); assert.equal(f.calls.length, 0); await f.api.refresh();
  assert(f.calls.every(c => c.method === 'GET')); assert.equal(f.terminals.length, 1); assert.equal(f.terminals[0]?.ctx.environmentId, env);
  await f.api.refresh(); assert(f.terminals[0]?.disposed); assert(f.w.document.getElementById('native'));
});
test('view tabs and app switches retain terminal and dispatch no reads or writes', async t => {
  const f = fixture(t); await f.api.refresh(); const count = f.calls.length;
  assert.equal(f.root.querySelector('[role=tab][aria-selected=true]')?.textContent, 'Terminal');
  f.button('Access').click(); assert(!f.root.querySelector<HTMLElement>('#sodaspaces-view-access')?.hidden);
  f.button('Environment').click(); f.button('Terminal').click();
  f.w.dispatchEvent(new f.w.Event('blur')); f.w.document.dispatchEvent(new f.w.Event('visibilitychange'));
  assert.equal(f.calls.length, count); assert.equal(f.terminals.length, 1); assert(!f.terminals[0]?.disposed);
  f.button('Terminal').dispatchEvent(new f.w.KeyboardEvent('keydown', {key: 'ArrowRight', cancelable: true}));
  assert.equal(f.root.querySelector('[role=tab][aria-selected=true]')?.textContent, 'Environment');
  assert.equal(f.w.document.activeElement?.textContent, 'Environment');
});
test('hidden access view cannot remove a key through a synthetic click', async t => {
  const f = fixture(t); await f.api.refresh();
  assert(f.root.querySelector<HTMLElement>('#sodaspaces-view-access')?.hidden);
  f.root.querySelector<HTMLButtonElement>('#sodaspaces-key-list button')?.click(); await tick();
  assert(f.calls.every(c => c.method === 'GET'));
});
test('create never implicitly joins, saves keys or starts', async t => {
  const f = fixture(t, {absent: true}); await f.api.refresh(); f.button('Create environment').click(); await tick();
  const writes = f.calls.filter(c => c.method !== 'GET'); assert.equal(writes.length, 1); assert(writes[0]); assert(writes[0].url.endsWith('/api/environments'));
  assert.deepEqual(JSON.parse(writes[0].body || '{}'), {repository_id: '7'});
});
test('Stop requires explicit shared-impact confirmation and Start is separate', async t => {
  const f = fixture(t); await f.api.refresh(); f.button('Stop').click(); assert(f.calls.every(c => c.method === 'GET'));
  const checkbox = f.root.querySelector<HTMLInputElement>('input[type=checkbox]'); assert(checkbox); checkbox.checked = true; f.button('Stop').click(); await tick();
  const write = f.calls.find(c => c.method === 'POST'); assert(write && typeof write.body === 'string'); assert.deepEqual(JSON.parse(write.body), {action: 'stop', confirm_stop: true}); assert(f.terminals[0]?.stale);
});
test('saved-key removal truthfully does not dispatch a native apply', async t => {
  const f = fixture(t); await f.api.refresh(); f.button('Remove saved key').click(); await tick();
  const writes = f.calls.filter(c => c.method !== 'GET'); assert.equal(writes.length, 1); assert(writes[0]); assert.equal(writes[0].method, 'DELETE'); assert(!writes[0].url.endsWith('/access-keys'));
  assert.match(f.root.textContent || '', /Existing project SSH access is unchanged/);
});
test('review then explicit Apply confirms last-key removal', async t => {
  const f = fixture(t, {saved: []}); await f.api.refresh(); f.button('Review this project’s SSH keys').click(); await tick();
  f.button('Apply reviewed saved keys to this project').click(); assert(f.calls.every(c => c.method === 'GET'));
  const checkbox = f.root.querySelectorAll<HTMLInputElement>('input[type=checkbox]')[1]; assert(checkbox); checkbox.checked = true; f.button('Apply reviewed saved keys to this project').click(); await tick();
  const write = f.calls.find(c => c.method === 'POST'); assert(write && typeof write.body === 'string'); assert.deepEqual(JSON.parse(write.body), {revision: 'a'.repeat(64), saved_fingerprints: [], confirm_empty: true});
});
test('private-key paste is refused before request dispatch', async t => {
  const f = fixture(t); await f.api.refresh(); const textarea = f.root.querySelector('textarea'); assert(textarea); textarea.value = '-----BEGIN OPENSSH PRIVATE KEY-----'; f.button('Save public key').click();
  assert(f.calls.every(c => c.method === 'GET')); assert.match(f.root.textContent || '', /Never upload a private key/);
});
test('nonadministrator has no lifecycle controls; nonmember joins separately', async t => {
  const f = fixture(t, {admin: false, member: false}); await f.api.refresh(); assert(f.button('Start').closest('fieldset')?.hidden); assert(!f.button('Join environment').hidden);
  f.button('Join environment').click(); await tick(); assert.equal(f.calls.filter(c => c.method !== 'GET').length, 1); assert(f.calls.find(c => c.method === 'POST')?.url.endsWith('/join'));
});
test('unknown mutation outcome blocks replay, not safe refresh or logout', async t => {
  const f = fixture(t, {absent: true, fetch: async (url, init) => init.method === 'POST' && url.endsWith('/api/environments') ? new Response(null, {status: 502}) : null});
  await f.api.refresh(); f.button('Create environment').click(); await tick(); await f.api.refresh();
  assert(f.button('Create environment').disabled); assert.match(f.root.textContent || '', /Outcome unconfirmed/);
  f.button('Sign out of Soda').click(); await tick(); assert(f.calls.some(c => c.url.endsWith('/api/session/logout')));
});
test('stale page closes terminal and cannot refresh/replay on focus', async t => {
  const f = fixture(t); await f.api.refresh(); const count = f.calls.length;
  f.w.dispatchEvent(new f.w.Event('pagehide')); f.w.dispatchEvent(new f.w.Event('focus')); await f.api.refresh();
  assert.equal(f.calls.length, count); assert(f.terminals[0]?.disposed); assert(f.button('Refresh status').disabled);
  assert(f.w.document.getElementById('native'));
});
test('mutation rechecks current session and refuses a switched actor before dispatch', async t => {
  let switched = false;
  const f = fixture(t, {absent: true, fetch: async (url) => switched && url.endsWith('/api/session') ? new Response(JSON.stringify({user: {id: '2'}, csrf_token: 'synthetic-csrf', forgejo_url: 'https://forge.test'}), {headers: {'Content-Type': 'application/json'}}) : null});
  await f.api.refresh(); switched = true; f.button('Create environment').click(); await tick();
  assert(f.calls.every(c => c.method === 'GET'));
});
for (const body of [new Response('<html>login</html>', {headers: {'Content-Type': 'text/html'}}), new Response(JSON.stringify({padding: 'x'.repeat(65537)}), {headers: {'Content-Type': 'application/json'}})]) {
  test('HTML/oversized streamed response cannot expose actions', async t => {
    const f = fixture(t, {fetch: async () => body}); await f.api.refresh();
    assert(f.button('Create environment').hidden); assert.equal(f.terminals.length, 0);
  });
}
test('refresh never remounts an ended or started terminal', async t => {
  const f = fixture(t); await f.api.refresh(); const terminal = f.terminals[0]; assert(terminal); terminal.started = true;
  await f.api.refresh(); assert.equal(f.terminals.length, 1); assert(f.terminals[0]?.disposed);
  assert.match(f.root.textContent || '', /terminal session ended/);
});
test('a provider mismatch at action time dispatches no mutation', async t => {
  let switched = false;
  const f = fixture(t, {absent: true, fetch: async url => switched && url.endsWith('/api/forgejo/me') ? new Response('{"id":"2"}', {headers: {'Content-Type': 'application/json'}}) : null});
  await f.api.refresh(); switched = true; f.button('Create environment').click(); await tick();
  assert(f.calls.every(c => c.method === 'GET'));
});
test('closing while authorization is pending never dispatches the mutation', async t => {
  let paused = false; let release: ((response: Response) => void) | undefined;
  const f = fixture(t, {absent: true, fetch: async url => paused && url.endsWith('/api/session') ? new Promise(resolve => { release = resolve; }) : null});
  await f.api.refresh(); paused = true; f.button('Create environment').click(); f.api.dispose();
  assert(release); release(new Response(JSON.stringify({user: {id: '1'}, csrf_token: 'synthetic-csrf', forgejo_url: 'https://forge.test'}), {headers: {'Content-Type': 'application/json'}})); await tick();
  assert(f.calls.every(c => c.method === 'GET'));
});
test('unknown key-save response cannot claim a confirmed key', async t => {
  const f = fixture(t, {fetch: async (url, init) => init.method === 'POST' ? new Response('{"items":[]}', {headers: {'Content-Type': 'application/json'}}) : null});
  await f.api.refresh(); const textarea = f.root.querySelector('textarea'); assert(textarea); textarea.value = 'ssh-ed25519 YWJj'; f.button('Save public key').click(); await tick();
  assert.match(f.root.textContent || '', /Outcome unconfirmed/);
});
test('hidden create and lifecycle actions cannot dispatch through their handlers', async t => {
  const f = fixture(t, {admin: false}); await f.api.refresh(); f.button('Create environment').click(); f.button('Start').click(); await tick();
  assert(f.calls.every(c => c.method === 'GET'));
});
test('copy uses own displayed login/IP without changing native access', async t => {
  const f = fixture(t); await f.api.refresh(); f.button('Copy SSH connection').click(); await tick(); assert.equal(f.button('Copy SSH connection').getAttribute('data-clipboard-target'), '#sodaspaces-command'); assert.equal(f.root.querySelector<HTMLInputElement>('#sodaspaces-command')?.value, 'ssh alice@10.89.0.2'); assert(f.calls.every(c => c.method === 'GET'));
});
