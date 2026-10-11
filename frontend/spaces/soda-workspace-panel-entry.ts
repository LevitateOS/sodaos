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
  const frame = host?.querySelector<HTMLIFrameElement>('[data-extension-workspace-frame]');
  let workspace: ReturnType<typeof mountSodaspaces> | undefined;
  const retire = () => {
    workspace?.dispose();
    workspace = undefined;
  };
  const sync = () => {
    // The full Spaces page restores the same actor layout. It must be the sole
    // workspace owner while selected in the native frame.
    if (frame?.contentWindow?.location.protocol === 'about:') return;
    const page = frame?.contentDocument?.querySelector(
      '[data-extension-page][data-extension-id="soda"][data-extension-page-id="spaces"]'
    );
    if (page) {
      retire();
    } else if (!workspace) {
      workspace = mountSodaspaces(root, {
        kind: 'native',
        transport,
        forgejoPrefix: prefix,
        ...(context.repositoryId ? {repositoryId: context.repositoryId, pageRepositoryId: context.repositoryId} : {}),
      });
      void workspace.refresh();
    }
  };
  host?.addEventListener('soda-spaces-opening', retire);
  frame?.addEventListener('load', sync);
  sync();
  return {
    dispose() {
      transport.dispose();
      host?.removeEventListener('soda-spaces-opening', retire);
      frame?.removeEventListener('load', sync);
      retire();
      styles.forEach((style) => style.remove());
    },
  };
}
