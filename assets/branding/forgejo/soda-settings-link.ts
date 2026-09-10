import {id, object, readSodaJSON, sessionResponse} from '../../../frontend/spaces/sodaspaces-api.js';

// Native templates cannot know Soda's configured operator. This hint only reveals
// navigation; every settings page/API independently authorizes the real session.
const marker = document.getElementById('soda-settings-link');
if (marker && id(marker.dataset.actor)) {
  const actor = marker.dataset.actor, sub = marker.dataset.subUrl || '';
  const link = document.createElement('a'); link.className = 'item'; link.textContent = 'SodaOS settings';
  link.href = sub + '/-/soda/settings/runners';
  let generation = 0;
  const hide = () => {generation++; link.remove();};
  const check = async () => {
    hide(); const current = generation;
    try {
      const response = await fetch(sub + '/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', headers: {'X-Soda-Expected-User-ID': actor}, signal: AbortSignal.timeout(10000)});
      if (!response.ok) {await response.body?.cancel(); return;}
      const raw = await readSodaJSON(response), session = sessionResponse(raw, location.origin);
      if (current === generation && marker.isConnected && session.user.id === actor && object(raw).soda_operator === true) marker.after(link);
    } catch {hide();}
  };
  window.addEventListener('pagehide', hide);
  window.addEventListener('pageshow', () => void check());
  document.addEventListener('visibilitychange', () => {if (document.hidden) hide(); else void check();});
  void check();
}
