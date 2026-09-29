import {prepareExtensionMount} from './soda-extension.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {mountSpacesPage} from './sodaspaces-page.js';
import {forgejoPrefix} from './soda-native-paths.js';

export function mount(root: HTMLElement, context: ExtensionMountContext) {
  if (context.pageId !== 'spaces') throw Error('Invalid Spaces page');
  const transport = prepareExtensionMount(root, context);
  const styles = ['components.css', 'sodaspaces.css', 'sodaspaces-page.css', 'sodaspaces-terminal.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  root.append(...styles);
  const page = mountSpacesPage(root, transport, forgejoPrefix(context));
  return {
    dispose() {
      transport.dispose();
      page.dispose();
      styles.forEach((style) => style.remove());
      root.classList.remove('soda-spaces-page-mount');
    },
  };
}
