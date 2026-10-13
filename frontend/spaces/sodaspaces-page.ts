import {mountSodaspaces} from './sodaspaces-workspace.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {ContributionPresentation, ResumeValidation} from './fountain-browser-contract.js';

export function mountSpacesPage(
  root: HTMLElement,
  transport: PreparedExtensionMount,
  forgejoPrefix = '',
  repositoryId?: string
) {
  const workspace = mountSodaspaces(root, {
    kind: 'page',
    transport,
    forgejoPrefix,
    ...(repositoryId ? {repositoryId} : {}),
  });
  let disposed = false;
  const measure = () => {
    if (disposed || !root.isConnected) return;
    const footer = document.querySelector('footer');
    const top = root.getBoundingClientRect().top + window.scrollY;
    const height = Math.max(
      0,
      (window.visualViewport?.height || window.innerHeight) - top - (footer?.getBoundingClientRect().height || 0)
    );
    root.style.height = Math.floor(height) + 'px';
  };
  const observer = new ResizeObserver(measure);
  for (const element of [document.getElementById('navbar'), document.querySelector('footer')])
    if (element) observer.observe(element);
  window.addEventListener('resize', measure);
  window.visualViewport?.addEventListener('resize', measure);
  void document.fonts.ready.then(measure);
  measure();
  void workspace.refresh();
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    observer.disconnect();
    window.removeEventListener('resize', measure);
    window.visualViewport?.removeEventListener('resize', measure);
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
