import {connectPage} from './soda-connection.js';
import {id} from './sodaspaces-api.js';
import {mountSpacesPage} from './sodaspaces-page.js';

async function startWorkspaceShell() {
  const root = document.getElementById('soda-workspace-root');
  const actor = root?.dataset.actor;
  if (!root || !id(actor)) return;
  const session = await connectPage(actor, 'spaces', '', () => root.isConnected);
  if (!session) return;
  mountSpacesPage(root, actor, session);
}

void startWorkspaceShell();
