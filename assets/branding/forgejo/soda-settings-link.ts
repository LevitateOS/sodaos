// Retain coordinated native logout on ordinary Forgejo pages. This import does
// not probe a session or begin OAuth just to render navigation.
import '../../../frontend/spaces/soda-connection.js';
import {id} from '../../../frontend/spaces/sodaspaces-api.js';

// Runners/Tailnet entry links are rendered by Forgejo's administration layout,
// not the global header. Their protected APIs still own operator authorization.
const marker = document.getElementById('soda-settings-link');
if (marker && id(marker.dataset.actor)) {
  const host = document.querySelector<HTMLElement>('main.soda-native-page #soda-native-content');
  const spaces = document.getElementById('soda-spaces-link');
  // The native host validated the selector. A query hint or another actor must
  // not claim that an ordinary dashboard/repository page is Spaces.
  if (host?.dataset.actor === marker.dataset.actor && host.dataset.view === 'spaces' && spaces instanceof HTMLAnchorElement) {
    spaces.classList.add('active');
    spaces.setAttribute('aria-current', 'page');
  }
}
