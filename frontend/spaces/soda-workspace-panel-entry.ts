import {prepareExtensionMount} from './soda-extension.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {mountSodaspaces} from './sodaspaces-workspace.js';
import {forgejoPrefix} from './soda-native-paths.js';

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
  const prefix = forgejoPrefix(context);
  const host = root.closest<HTMLElement>('[data-extension-workspace]');
  const frame = host?.querySelector<HTMLIFrameElement>('[data-extension-workspace-frame]');
  let workspace: ReturnType<typeof mountSodaspaces> | undefined;
  const sync = () => {
    // The full Spaces page restores the same actor layout. It must be the sole
    // workspace owner while selected in the native frame.
    const location = frame?.contentWindow?.location;
    const path =
      location?.protocol === 'about:'
        ? new URL(host?.dataset.workspacePath || '/', window.location.href).pathname
        : location?.pathname;
    if (frame && path === prefix + '/-/extensions/pages/soda/spaces') {
      workspace?.dispose();
      workspace = undefined;
    } else if (!workspace) {
      workspace = mountSodaspaces(root, {kind: 'native', transport, forgejoPrefix: prefix});
      void workspace.refresh();
    }
  };
  host?.addEventListener('soda-spaces-opening', sync);
  frame?.addEventListener('load', sync);
  sync();
  return {
    dispose() {
      transport.dispose();
      host?.removeEventListener('soda-spaces-opening', sync);
      frame?.removeEventListener('load', sync);
      workspace?.dispose();
      styles.forEach((style) => style.remove());
    },
  };
}
