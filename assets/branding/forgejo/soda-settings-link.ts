import {connectionSuppressed} from '../../../frontend/spaces/soda-connection.js';
import {id, object, readSodaJSON, sessionResponse} from '../../../frontend/spaces/sodaspaces-api.js';

// Native templates cannot know Soda's configured operator. This hint only reveals
// navigation; every settings page/API independently authorizes the real session.
const marker = document.getElementById('soda-settings-link');
if (marker && id(marker.dataset.actor)) {
  const actor = marker.dataset.actor, sub = marker.dataset.subUrl || '';
  // The native host already validated the route/selector. URL hints alone must
  // not mark ordinary dashboards, repository pages or invalid destinations.
  const host = document.querySelector<HTMLElement>('main.soda-native-page #soda-native-content');
  const currentView = host?.dataset.actor === actor ? host.dataset.view : undefined;
  const spaces = document.getElementById('soda-spaces-link');
  if (currentView === 'spaces' && spaces instanceof HTMLAnchorElement) {
    spaces.classList.add('active');
    spaces.setAttribute('aria-current', 'page');
  }
  const link = document.createElement('a'); link.className = 'item'; link.textContent = 'Runners';
  link.href = sub + '/?soda-view=runners';
  const tailnet = document.createElement('a'); tailnet.className = 'item';
  tailnet.textContent = marker.dataset.tailnetLabel || 'Tailnet';
  tailnet.href = sub + '/?soda-view=tailnet';
  if (currentView === 'runners') {
    link.classList.add('active');
    link.setAttribute('aria-current', 'page');
  }
  if (currentView === 'tailnet') {
    tailnet.classList.add('active');
    tailnet.setAttribute('aria-current', 'page');
  }
  let generation = 0;
  const hide = () => {generation++; link.remove(); tailnet.remove();};
  const check = async () => {
    hide(); if (connectionSuppressed()) return; const current = generation;
    try {
      const response = await fetch(sub + '/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': actor}, signal: AbortSignal.timeout(10000)});
      if (!response.ok) {await response.body?.cancel(); return;}
      const raw = await readSodaJSON(response), session = sessionResponse(raw, location.origin);
      if (current === generation && !connectionSuppressed() && marker.isConnected && session.user.id === actor && object(raw).soda_operator === true) marker.after(link, tailnet);
    } catch {if (current === generation) hide();}
  };
  window.addEventListener('pagehide', hide);
  window.addEventListener('soda-session-retired', hide);
  window.addEventListener('pageshow', () => void check());
  document.addEventListener('visibilitychange', () => {if (document.hidden) hide(); else void check();});
  void check();
}
