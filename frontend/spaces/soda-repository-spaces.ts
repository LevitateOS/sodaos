// The repository settings page mounts the same command owner as the drawer.
import {SodaProjectControls} from './sodaspaces-project.js';
import {id} from './sodaspaces-api.js';
export function mountRepositorySpaces(mount: HTMLElement, actor: string, repository: string) {
  if (!id(actor) || !id(repository)) throw Error('Invalid repository context');
  mount.classList.add('soda-repository-spaces');
  const controls = new SodaProjectControls();
  controls.configure({expectedUserId: actor, repositoryId: repository, page: true, settings: true});
  mount.replaceChildren(controls);
  void controls.refresh();
  return controls;
}
const mount = document.getElementById('soda-repository-spaces');
if (mount && id(mount.dataset.actor) && id(mount.dataset.repositoryId)) mountRepositorySpaces(mount, mount.dataset.actor, mount.dataset.repositoryId);
