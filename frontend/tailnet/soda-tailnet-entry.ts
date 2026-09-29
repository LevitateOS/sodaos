import {prepareExtensionMount} from '../spaces/soda-extension.js';
import type {ExtensionMountContext} from '../spaces/soda-extension.js';
import {mountTailnetPage} from './soda-tailnet-page.js';

export function mount(root: HTMLElement, context: ExtensionMountContext) {
  if (context.pageId !== 'tailnet') throw Error('Invalid Tailnet page');
  const transport = prepareExtensionMount(root, context);
  root.classList.add('soda-settings', 'soda-tailnet-settings');
  const styles = ['components.css', 'soda-settings.css', 'soda-tailnet.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  const page = mountTailnetPage(root, transport);
  root.append(...styles);
  return {
    dispose() {
      transport.dispose();
      page.dispose();
      styles.forEach((link) => link.remove());
      root.classList.remove('soda-settings', 'soda-tailnet-settings');
    },
  };
}
