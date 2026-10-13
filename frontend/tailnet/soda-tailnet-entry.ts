import {prepareExtensionMount} from '../spaces/soda-extension.js';
import type {ExtensionMountContext} from '../spaces/soda-extension.js';
import type {MountHandle} from '../spaces/fountain-browser-contract.js';
import {mountTailnetPage} from './soda-tailnet-page.js';
import type {ContributionPresentation, MountHandle, ResumeValidation} from '../spaces/fountain-browser-contract.js';

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
  };
  // Host lifecycle (NFX-D05): suspend clears secrets and pauses polling,
  // resume revalidates under the admitted generation, retire releases.
  const handle: MountHandle & {dispose: () => void} = {
    update: (presentation: ContributionPresentation) => page.update(presentation),
    suspend: () => page.suspend(),
    resume: (validation: ResumeValidation) => page.resume(validation),
    retire: () => dispose(),
    dispose,
  };
  return handle;
}
