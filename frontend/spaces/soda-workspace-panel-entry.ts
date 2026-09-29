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
  const workspace = mountSodaspaces(root, {kind: 'native', transport, forgejoPrefix: forgejoPrefix(context)});
  void workspace.refresh();
  return {
    dispose() {
      transport.dispose();
      workspace.dispose();
      styles.forEach((style) => style.remove());
    },
  };
}
