import {id} from '../../../frontend/spaces/sodaspaces-api.js';
import {workspaceEntryLocation} from '../../../frontend/spaces/sodaspaces-frame.js';

function startWorkspaceEntry() {
  if (window.frameElement) return;
  const marker = document.getElementById('soda-settings-link');
  if (!marker || !id(marker.dataset.actor)) return;
  const next = workspaceEntryLocation(location.href, marker.dataset.subUrl || '');
  if (next) location.replace(next);
}

startWorkspaceEntry();
