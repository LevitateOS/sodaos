import {prepareExtensionMount} from '../spaces/soda-extension.js';
import type {ExtensionMountContext} from '../spaces/soda-extension.js';
import type {MountHandle} from '../spaces/fountain-browser-contract.js';
import {mountTailnetPage} from './soda-tailnet-page.js';

export function mount(root: HTMLElement, context: ExtensionMountContext): MountHandle {
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
  let disposed = false;
  let retired = false;
  let suspended = false;
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    transport.dispose();
    page.dispose();
    styles.forEach((link) => link.remove());
    root.classList.remove('soda-settings', 'soda-tailnet-settings');
  };
  return {
    dispose,
    update(presentation) {
      if (disposed || retired) return;
      if (!presentation.visible && !suspended) {
        suspended = true;
        page.suspend();
      } else if (presentation.visible && suspended) {
        suspended = false;
        page.resume();
      }
    },
    suspend() {
      if (disposed || retired || suspended) return;
      suspended = true;
      page.suspend();
    },
    resume(validation) {
      if (disposed || retired) return;
      if (validation.sessionGeneration !== transport.generation) {
        page.retire();
        return;
      }
      suspended = false;
      page.resume();
    },
    retire() {
      if (retired) return;
      retired = true;
      page.retire();
      dispose();
    },
  };
}
