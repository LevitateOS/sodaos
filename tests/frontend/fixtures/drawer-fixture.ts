// Browser-realm fixture only. Production modules are emitted separately, never bundled into this double.
import {mountProjectControls} from '../../../frontend/spaces/sodaspaces-project.js';
export const environmentID = 'p0123456789abcdef01234567';
export const fixtureProfile = {id: 'rocky-headless', distribution: 'rocky', version: '10.2', interface: 'headless', architecture: 'amd64', image: 'sha256:' + 'a'.repeat(64), revision: 'b'.repeat(40)};
export const fixtureFingerprint = 'SHA256:' + 'A'.repeat(43);
export interface Call {url: string; method: string; body?: string; headers: Record<string, string>}
export interface State {running: boolean; member: boolean; admin: boolean; absent: boolean; saved: string[]; installed: string[]; user: string; provider: string; provisioned: boolean; unavailable: boolean}
function createFixture(extra: Partial<State> = {}) {
  const root = document.querySelector('main'); if (!root) throw Error('fixture mount missing');
  const calls: Call[] = [];
  const state: State = {running: true, member: true, admin: true, absent: false, saved: [fixtureFingerprint], installed: [fixtureFingerprint], user: '1', provider: '1', provisioned: true, unavailable: false, ...extra};
  let reply: ((call: Call) => Promise<Response | null>) | undefined;
  const fetchFixture = async (input: RequestInfo | URL, init?: RequestInit) => {
    const call: Call = {url: String(input), method: init?.method || 'GET', ...(typeof init?.body === 'string' ? {body: init.body} : {}), headers: Object.fromEntries(new Headers(init?.headers).entries())};
    calls.push(call);
    const override = await reply?.(call); if (override) return override;
    const {url, method, body: encoded} = call;
    let body: unknown;
    if (method !== 'GET') {
      const input: unknown = JSON.parse(encoded || '{}');
      if (!input || typeof input !== 'object') throw Error('invalid fixture action');
      if (url.endsWith('/api/session/logout')) return new Response(null, {status: 204});
      if (url.endsWith('/api/environments')) body = {id: environmentID, repository_id: '7', provisioned: true, profile: fixtureProfile};
      else if (url.endsWith('/join')) body = {login: 'alice'};
      else if (url.endsWith('/lifecycle')) body = {environment: {id: environmentID, running: 'action' in input && input.action === 'start'}, boot_enabled: 'action' in input && input.action === 'start'};
      else if (url.endsWith('/access-keys')) body = {applied: true, login: 'alice', revision: 'a'.repeat(64), installed_fingerprints: 'saved_fingerprints' in input ? input.saved_fingerprints : null};
      else if (method === 'DELETE') body = {removed: true, existing_project_access_changed: false};
      else body = {items: []};
    } else if (url.endsWith('/api/session')) body = {user: {id: state.user, login: 'alice'}, csrf_token: 'synthetic-csrf', forgejo_url: location.origin};
    else if (url.endsWith('/api/forgejo/me')) body = {id: state.provider};
    else if (url.endsWith('/api/repositories/7/profiles')) body = {items: [fixtureProfile]};
    else if (url.includes('/api/environments?')) body = {repository: {id: '7'}, can_create: state.absent, items: state.absent ? [] : [{id: environmentID, repository_id: '7'}]};
    else if (url.endsWith('/api/me/development-keys')) body = {items: state.saved.map((fingerprint, i) => ({id: String(i + 1), fingerprint}))};
    else if (url.endsWith('/os')) body = {environment: {id: environmentID, running: state.running, image: 'sha256:' + 'a'.repeat(64)}, os_release: state.running ? {id: 'rocky', version: '9.7', name: 'Rocky Linux 9.7'} : null, os_release_unavailable: !state.running};
    else if (url.endsWith('/lifecycle')) body = {environment: {id: environmentID, running: state.running}, boot_enabled: state.running};
    else if (url.endsWith('/access-keys')) body = {login: 'alice', revision: 'a'.repeat(64), installed_fingerprints: state.installed, saved_fingerprints: state.saved};
    else if (url.endsWith('/connection')) body = {login: 'alice', connection: {environment: {id: environmentID, running: true, ip: '10.89.0.2'}, fingerprint: fixtureFingerprint}};
    else body = {environment: {id: environmentID, repository_id: '7', provisioned: state.provisioned}, observed: state.unavailable ? null : {id: environmentID, running: state.running}, login: state.member ? 'alice' : '', environment_administrator: state.admin, native_unavailable: state.unavailable, authority_unavailable: false};
    return Response.json(body);
  };
  Object.defineProperty(window, 'fetch', {value: fetchFixture, configurable: true});
  const api = mountProjectControls(root, {expectedUserId: '1', repositoryId: '7'});
  const button = (text: string) => {
    const b = [...root.querySelectorAll('button')].find(b => b.textContent?.trim() === text);
    if (!b) throw Error('Missing button: ' + text); return b;
  };
  const showButton = async (text: string) => {
    const b = button(text), panel = b.closest('[role=tabpanel]');
    if (panel) root.querySelector<HTMLElement>('[aria-controls="' + panel.id + '"]')?.click();
    await api.ready; return b;
  };
  return {api, state, calls, get terminals() {return root.querySelectorAll('soda-terminal');}, root, button, showButton, setReply(value: typeof reply) {reply = value;}};
}
declare global {interface Window {createDrawerFixture: typeof createFixture; drawerFixture: ReturnType<typeof createFixture>}}
window.createDrawerFixture = createFixture;
