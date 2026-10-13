import {mountSodaspaces} from './sodaspaces-workspace.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {ContributionPresentation, ResumeValidation} from './fountain-browser-contract.js';

export function mountSpacesPage(
  root: HTMLElement,
  transport: PreparedExtensionMount,
  forgejoPrefix = '',
  repositoryId?: string,
  navigate?: (path: string) => void
) {
  const workspace = mountSodaspaces(root, {
    kind: 'page',
    transport,
    forgejoPrefix,
    ...(navigate ? {navigate} : {}),
    ...(repositoryId ? {repositoryId} : {}),
  });
  let disposed = false;
  void workspace.refresh();
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    workspace.dispose();
  };
  return {
    get canRestore() {
      return workspace.canRestore;
    },
    update: (presentation: ContributionPresentation) => workspace.update(presentation),
    suspend: () => workspace.suspend(),
    resume: (validation?: ResumeValidation) => workspace.resume(validation),
    retire: () => dispose(),
    dispose,
  };
}
