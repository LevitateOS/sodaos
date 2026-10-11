import {prepareExtensionMount} from './soda-extension.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {mountSodaspaces} from './sodaspaces-workspace.js';
import {nativeAppSubUrl} from './soda-native-paths.js';

export function mount(root: HTMLElement, context: ExtensionMountContext) {
  if (context.panelId !== 'workspace') throw Error('Invalid workspace panel');
  const transport = prepareExtensionMount(root, context);
  const styles = ['components.css', 'sodaspaces-page.css', 'sodaspaces-drawer.css', 'sodaspaces-terminal.css'].map(
    (name) => {
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = transport.assetBase + name;
      return link;
    }
  );
  root.append(...styles);
  const prefix = nativeAppSubUrl(context);
  const host = root.closest<HTMLElement>('[data-extension-workspace]');
  const section = root.closest<HTMLElement>('.extension-workspace-panel');
  const aside = section?.closest<HTMLElement>('.extension-workspace-panels');
  const present = (visible: boolean) => {
    if (!section || !aside) return;
    section.hidden = !visible;
    aside.hidden = !Array.from(aside.querySelectorAll<HTMLElement>('.extension-workspace-panel')).some(
      (panel) => !panel.hidden
    );
  };
  let workspace: ReturnType<typeof mountSodaspaces> | undefined;
  const retire = () => {
    workspace?.dispose();
    workspace = undefined;
    present(false);
  };
  const sync = () => {
    const full = host?.hasAttribute('data-workspace-direct') === true;
    const repositoryId = host?.hasAttribute('data-extension-repository-id')
      ? host.dataset.extensionRepositoryId || undefined
      : context.repositoryId;
    present(true);
    root.classList.toggle('soda-spaces-page-mount', full);
    if (!workspace) {
      workspace = mountSodaspaces(root, {
        kind: full ? 'page' : 'native',
        transport,
        forgejoPrefix: prefix,
        ...(repositoryId ? {repositoryId} : {}),
      });
      void workspace.refresh();
    } else workspace.present(full ? 'page' : 'native', repositoryId);
    measure();
  };
  const measure = () => {
    if (!host?.hasAttribute('data-workspace-direct')) {
      root.style.height = '';
      return;
    }
    const footer = document.querySelector('footer');
    root.style.height =
      Math.max(
        300,
        window.innerHeight - root.getBoundingClientRect().top - (footer?.getBoundingClientRect().height || 0)
      ) + 'px';
  };
  host?.addEventListener('extension-workspace-navigation', sync);
  window.addEventListener('resize', measure);
  sync();
  return {
    dispose() {
      transport.dispose();
      host?.removeEventListener('extension-workspace-navigation', sync);
      window.removeEventListener('resize', measure);
      retire();
      styles.forEach((style) => style.remove());
    },
  };
}
