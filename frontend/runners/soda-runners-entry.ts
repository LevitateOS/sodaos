import {prepareExtensionMount} from '../spaces/soda-extension.js';
import type {ExtensionMountContext} from '../spaces/soda-extension.js';
import {mountRunnersPage} from './soda-runners-page.js';

export function mount(root: HTMLElement, context: ExtensionMountContext) {
  if (context.pageId !== 'runners') throw Error('Invalid runners page');
  const transport = prepareExtensionMount(root, context);
  root.classList.add('soda-settings', 'soda-runner-settings');
  const styles = ['components.css', 'soda-settings.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  const page = mountRunnersPage(root, transport);
  root.append(...styles);
  return {
    dispose() {
      transport.dispose();
      page.dispose();
      styles.forEach((link) => link.remove());
      root.classList.remove('soda-settings', 'soda-runner-settings');
    },
  };
}
