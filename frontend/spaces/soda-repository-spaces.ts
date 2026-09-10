// The repository settings page mounts the same command owner as the drawer.
import {SodaProjectControls} from './sodaspaces-project.js';
import {id} from './sodaspaces-api.js';
export function mountRepositorySpaces(mount: HTMLElement, actor: string, repository: string) {
  if (!id(actor) || !id(repository)) throw Error('Invalid repository context');
  mount.classList.add('soda-repository-spaces');
  const controls = new SodaProjectControls();
  controls.configure({expectedUserId: actor, repositoryId: repository, page: true, settings: true});
  // The entry already cleared its content. Preserve its Lit render boundary so
  // a later history restoration can render connection feedback in the same host.
  mount.append(controls);
  void controls.refresh();
  return controls;
}
