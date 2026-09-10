import {id, object, readSodaJSON, sessionResponse} from './sodaspaces-api.js';

const suppressionKey = 'soda-signout';
let retired = false;
let signingOut = false;
let replay: HTMLAnchorElement | null = null;
const marker = document.getElementById('soda-settings-link');
const sub = marker?.dataset.subUrl || '';
const api = sub + '/-/soda';

export function connectionSuppressed(): boolean {
  try {return retired || localStorage.getItem(suppressionKey) !== null;} catch {return retired;}
}
function retire() {
  retired = true;
  window.dispatchEvent(new Event('soda-session-retired'));
}
window.addEventListener('storage', event => {
  if (event.key === suppressionKey && event.newValue !== null) retire();
});
window.addEventListener('pageshow', () => {if (connectionSuppressed()) retire();});

function allowExplicitConnection() {
  retired = false;
  try {localStorage.removeItem(suppressionKey);} catch { /* UI signal only. */ }
}

function suppress() {
  retire();
  try {localStorage.setItem(suppressionKey, String(Date.now()));} catch { /* UI signal only. */ }
}

async function request(path: string, headers: Record<string, string>, post = false, signal?: AbortSignal) {
  return fetch(api + path, {
    credentials: 'same-origin', cache: 'no-store', redirect: 'error',
    signal: signal ? AbortSignal.any([signal, AbortSignal.timeout(15000)]) : AbortSignal.timeout(15000), headers,
    ...(post ? {method: 'POST', body: '{}'} : {}),
  });
}

// Cancel pending OAuth first. Its cookie survives a callback claim in the store;
// a late callback cookie cannot restore the deleted context. Then end any session
// which predates this transaction through the existing authenticated logout API.
async function cancel(actor: string, signal: AbortSignal) {
  // Capture the actor/session before asynchronous cancellation; never bootstrap
  // a later session and accidentally adopt it while finishing this sign-out.
  const response = await request('/api/session', {'X-Soda-Expected-User-ID': actor}, false, signal);
  let session = null;
  if (response.ok) {
    session = sessionResponse(await readSodaJSON(response), location.origin);
    if (session.user.id !== actor) throw Error('Actor changed');
  } else {
    await response.body?.cancel();
    if (response.status !== 401) throw Error('Session unavailable or changed');
  }
  const headers = {'X-Soda-Logout': '1', 'X-Soda-Expected-User-ID': actor};
  const pending = await request('/api/login/cancel', headers, false, signal);
  if (pending.status === 200) {
    const data = object(await readSodaJSON(pending));
    if (typeof data.csrf_token !== 'string' || !/^[A-Za-z0-9_-]{43}$/.test(data.csrf_token)) throw Error('Invalid cancellation response');
    const ended = await request('/api/login/cancel', {...headers, 'Content-Type': 'application/json', 'X-CSRF-Token': data.csrf_token}, true, signal);
    if (ended.status !== 204) {await ended.body?.cancel(); throw Error('Cancellation unconfirmed');}
    return;
  }
  if (pending.status !== 204) {await pending.body?.cancel(); throw Error('Cancellation unavailable');}
  if (!session) return;
  const ended = await request('/api/session/logout', {'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token, 'X-Soda-Expected-User-ID': actor}, true, signal);
  if (ended.status !== 204) {await ended.body?.cancel(); throw Error('Logout unconfirmed');}

}

function nativeLogout() {
  return document.querySelector<HTMLAnchorElement>('#navbar a.link-action[data-url="' + sub + '/user/logout"]');
}
function activate(link: HTMLAnchorElement) {
  replay = link;
  link.click(); // Exactly one original native activation; Forgejo owns its POST.
  replay = null;
}
function notice(message: string, actor: string, link: HTMLAnchorElement | null, failed: boolean) {
  document.getElementById('soda-signout-status')?.remove();
  const box = document.createElement('div'); box.id = 'soda-signout-status'; box.className = 'ui warning message'; box.setAttribute('role', 'alert');
  const text = document.createElement('p'); text.textContent = message; box.append(text);
  if (failed) {
    const retry = document.createElement('button'); retry.type = 'button'; retry.className = 'ui button'; retry.textContent = 'Retry sign-out';
    retry.addEventListener('click', () => void signOut(actor)); box.append(retry);
  }
  if (link) {
    const escape = document.createElement('button'); escape.type = 'button'; escape.className = 'ui button'; escape.textContent = failed ? 'Sign out of Forgejo only' : 'Retry Forgejo sign-out';
    escape.addEventListener('click', () => {escape.disabled = true; activate(link);}); box.append(escape);
  } else {
    const back = document.createElement('a'); back.href = sub + '/'; back.textContent = 'Continue to Forgejo to sign out'; box.append(back);
  }
  (document.querySelector('main, .page-content') || document.body).prepend(box);
}
export async function signOut(actor: string) {
  if (signingOut || !id(actor)) return;
  signingOut = true;
  suppress();
  const link = nativeLogout();
  const controller = new AbortController();
  window.addEventListener('pagehide', () => controller.abort(), {once: true, signal: controller.signal});
  try {
    await cancel(actor, controller.signal);
    notice('Soda sign-out is complete. Forgejo sign-out is still pending.', actor, link, false);
    if (link) activate(link);
  } catch {
    notice('Soda sign-out could not be confirmed. Retry, or explicitly sign out of Forgejo only.', actor, link, true);
    signingOut = false;
  } finally {controller.abort();}
}

// Capture synchronously before Forgejo's delegated bubble listener, including the
// click generated by keyboard activation. Other link-actions are untouched.
document.addEventListener('click', event => {
  const target = event.target instanceof Element ? event.target.closest('a') : null;
  if (target instanceof HTMLAnchorElement && new URL(target.href).origin === location.origin && new URL(target.href).pathname === api + '/login' && !signingOut) allowExplicitConnection();
  const link = target?.matches('a.link-action') ? target : null;
  if (!(link instanceof HTMLAnchorElement) || link !== nativeLogout()) return;
  if (replay === link) {replay = null; return;}
  const actor = marker?.dataset.actor;
  if (!id(actor)) return;
  event.preventDefault(); event.stopImmediatePropagation();
  void signOut(actor);
}, true);

// Entry-only bootstrap: callers invoke this before exposing editable content.
// No focus/polling/error handler is permitted to restart OAuth.
export async function connectPage(actor: string, destination: string, repository: string, isCurrent: () => boolean, retry = false, restoring = false): Promise<boolean> {
  if (!id(actor) || !['spaces', 'runners', 'repository-spaces'].includes(destination) || (destination === 'repository-spaces' ? !id(repository) : repository !== '')) throw Error('Invalid page context');
  const key = 'soda-entry:' + destination + ':' + repository + ':' + actor;
  if (!isCurrent()) return false;
  if (retry) {
    allowExplicitConnection();
    try {sessionStorage.removeItem(key);} catch { /* explicit navigation still works */ }
  }
  if (connectionSuppressed()) throw Error('Sign-out requested. Retry connection explicitly.');
  if (!retry && new URL(location.href).searchParams.has('soda-connect')) throw Error('Forgejo connection did not complete.');
  const response = await request('/api/session', {'X-Soda-Expected-User-ID': actor});
  if (!isCurrent() || connectionSuppressed()) {await response.body?.cancel(); throw Error('Entry retired.');}
  if (response.ok) {
    const session = sessionResponse(await readSodaJSON(response), location.origin);
    if (!isCurrent() || connectionSuppressed() || session.user.id !== actor) throw Error('Entry or Forgejo account changed.');
    try {sessionStorage.removeItem(key);} catch { /* No authority stored here. */ }
    return true;
  }
  const missing = response.status === 401; await response.body?.cancel();
  if (!isCurrent() || connectionSuppressed()) return false;
  // History restoration may validate a session, never bootstrap a new login.
  if (!missing || restoring) throw Error('Soda connection is unavailable or belongs to a different account.');
  try {
    if (sessionStorage.getItem(key)) throw Error('Forgejo connection did not complete.');
    sessionStorage.setItem(key, 'attempt');
  } catch (error) {
    // Without a durable per-tab attempt marker, fail closed rather than loop.
    if (!retry) throw error;
  }
  const query = new URLSearchParams({destination, expected_user_id: actor});
  if (repository) query.set('repository_id', repository);
  location.assign(api + '/login?' + query);
  return false;
}
