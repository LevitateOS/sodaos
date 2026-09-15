import {connectPage} from './soda-connection.js';
import {id} from './sodaspaces-api.js';
import {applyWorkspaceFrameNavigation} from './sodaspaces-frame.js';
import {bindWorkspaceShellLayout} from './sodaspaces-shell-layout.js';
import {mountSpacesPage} from './sodaspaces-page.js';

function frameLocation(frame: HTMLIFrameElement) {
  const win = frame.ownerDocument.defaultView;
  try {
    const loc = frame.contentWindow?.location;
    if (!win || !loc || loc.origin !== win.location.origin) return '';
    return loc.pathname + loc.search;
  } catch {
    return '';
  }
}

function onFrameLoad(frame: HTMLIFrameElement) {
  const win = frame.ownerDocument.defaultView;
  if (!win) return;
  applyWorkspaceFrameNavigation(win, frameLocation(frame));
}

async function startWorkspaceShell() {
  const root = document.getElementById('soda-workspace-root');
  const frame = document.querySelector<HTMLIFrameElement>('#soda-forgejo-frame');
  const actor = root?.dataset.actor;
  if (!root || !frame || !id(actor)) return;
  bindWorkspaceShellLayout(document);
  frame.addEventListener('load', () => onFrameLoad(frame));
  const session = await connectPage(actor, 'spaces', '', () => root.isConnected);
  if (!session) return;
  mountSpacesPage(root, actor, session);
}

void startWorkspaceShell();
