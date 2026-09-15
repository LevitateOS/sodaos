import {connectPage} from './soda-connection.js';
import {id} from './sodaspaces-api.js';
import {workspaceFrameLocator} from './sodaspaces-frame.js';
import {mountSpacesPage} from './sodaspaces-page.js';

function frameLocation(frame: HTMLIFrameElement) {
  try {
    const loc = frame.contentWindow?.location;
    if (!loc || loc.origin !== location.origin) return '';
    return loc.pathname + loc.search;
  } catch {
    return '';
  }
}

function syncWorkspaceLocator(frame: HTMLIFrameElement) {
  const to = workspaceFrameLocator(frameLocation(frame));
  if (to === undefined) return;
  const url = new URL(location.href);
  url.search = '';
  if (to !== '/') url.searchParams.set('to', to);
  const next = url.pathname + url.search;
  if (next !== location.pathname + location.search) history.replaceState(null, '', next);
}

async function startWorkspaceShell() {
  const root = document.getElementById('soda-workspace-root');
  const frame = document.querySelector<HTMLIFrameElement>('#soda-forgejo-frame');
  const actor = root?.dataset.actor;
  if (!root || !frame || !id(actor)) return;
  frame.addEventListener('load', () => syncWorkspaceLocator(frame));
  const session = await connectPage(actor, 'spaces', '', () => root.isConnected);
  if (!session) return;
  mountSpacesPage(root, actor, session);
}

void startWorkspaceShell();
