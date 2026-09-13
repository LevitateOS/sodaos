// Retain coordinated native logout on ordinary Forgejo pages. This import does
// not probe a session or begin OAuth just to render navigation.
import '../../../frontend/spaces/soda-connection.js';
import {id} from '../../../frontend/spaces/sodaspaces-api.js';

// Runners/Tailnet entry links are rendered by Forgejo's administration layout,
// not the global header. Their protected APIs still own operator authorization.
const marker = document.getElementById('soda-settings-link');
if (marker && id(marker.dataset.actor)) {
  // The administration host validates the selector before emitting the mount,
  // so a matching view plus the original actor marks the current entry. A query
  // hint or another actor must never claim the current page.
  const content = document.querySelector<HTMLElement>('.soda-admin[role="main"] #soda-native-content');
  const adminView = content?.dataset.view;
  if (content && content.dataset.actor === marker.dataset.actor && (adminView === 'runners' || adminView === 'tailnet')) {
    const current = document.getElementById(adminView === 'runners' ? 'soda-runners-link' : 'soda-tailnet-link');
    if (current instanceof HTMLAnchorElement) {
      // Forgejo still serves its Dashboard handler for these presentations.
      // Replace that native cue, rather than highlighting two destinations.
      const menu = current.closest('.flex-container-nav > .ui.vertical.menu');
      for (const link of menu?.querySelectorAll<HTMLAnchorElement>('a.active') ?? []) {
        if (link.getAttribute('href') === (marker.dataset.subUrl || '') + '/admin') {
          link.classList.remove('active');
          link.removeAttribute('aria-current');
        }
      }
      current.classList.add('active');
      current.setAttribute('aria-current', 'page');
    }
  }
  const host = document.querySelector<HTMLElement>('main.soda-native-page #soda-native-content');
  const spaces = document.getElementById('soda-spaces-link');
  // The native host validated the selector. A query hint or another actor must
  // not claim that an ordinary dashboard/repository page is Spaces.
  if (host?.dataset.actor === marker.dataset.actor && host.dataset.view === 'spaces' && spaces instanceof HTMLAnchorElement) {
    spaces.classList.add('active');
    spaces.setAttribute('aria-current', 'page');
  }
}
