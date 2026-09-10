import {connectionSuppressed} from '../../../frontend/spaces/soda-connection.js';
import {id, object, readSodaJSON, sessionResponse} from '../../../frontend/spaces/sodaspaces-api.js';

// Native templates cannot know Soda's configured operator. This hint only reveals
// navigation; every settings page/API independently authorizes the real session.
const marker = document.getElementById('soda-settings-link');
if (marker && id(marker.dataset.actor)) {
  const actor = marker.dataset.actor, sub = marker.dataset.subUrl || '';
  const link = document.createElement('a'); link.className = 'item'; link.textContent = 'SodaOS settings';
  link.href = sub + '/?soda-view=runners';
  let generation = 0;
  const hide = () => {generation++; link.remove();};
  const check = async () => {
    hide(); if (connectionSuppressed()) return; const current = generation;
    try {
      const response = await fetch(sub + '/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': actor}, signal: AbortSignal.timeout(10000)});
      if (!response.ok) {await response.body?.cancel(); return;}
      const raw = await readSodaJSON(response), session = sessionResponse(raw, location.origin);
      if (current === generation && !connectionSuppressed() && marker.isConnected && session.user.id === actor && object(raw).soda_operator === true) marker.after(link);
    } catch {if (current === generation) hide();}
  };
  window.addEventListener('pagehide', hide);
  window.addEventListener('soda-session-retired', hide);
  window.addEventListener('pageshow', () => void check());
  document.addEventListener('visibilitychange', () => {if (document.hidden) hide(); else void check();});
  void check();
}
