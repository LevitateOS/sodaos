// The repository settings page mounts the same command owner as the drawer.
import {SodaProjectControls} from './sodaspaces-project.js';
import {id} from './sodaspaces-api.js';
const mount = document.getElementById('soda-repository-spaces');
if (mount && id(mount.dataset.actor) && id(mount.dataset.repositoryId)) {
  const controls = new SodaProjectControls();
  controls.configure({expectedUserId: mount.dataset.actor, repositoryId: mount.dataset.repositoryId, page: true, settings: true});
  mount.replaceChildren(controls);
  void controls.refresh();
}
