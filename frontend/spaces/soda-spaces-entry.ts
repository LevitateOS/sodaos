import {prepareExtensionMount} from './soda-extension.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {mountSpacesPage} from './sodaspaces-page.js';
import {nativeAppSubUrl} from './soda-native-paths.js';
import type {ContributionPresentation, MountHandle, ResumeValidation} from './fountain-browser-contract.js';

export function mount(root: HTMLElement, context: ExtensionMountContext) {
  if (context.pageId !== 'spaces') throw Error('Invalid Spaces page');
  // Workspace navigation is handled by the enclosing native host. Do not start
  // a temporary second content owner in its ordinary-page browsing frame.
  if (window.frameElement?.hasAttribute('data-extension-workspace-frame')) return {dispose() {}};
  // The native host retains the single panel/content owner on direct Workspace
  // pages. This page mount supplies the native content slot, not another owner.
  if (
    root
      .closest('[data-workspace-direct]')
      ?.querySelector('[data-extension-panel-id="workspace"][data-extension-id="soda"]')
  )
    return {dispose() {}};
  const transport = prepareExtensionMount(root, context);
  const styles = ['components.css', 'sodaspaces.css', 'sodaspaces-page.css', 'sodaspaces-terminal.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  root.append(...styles);
  const page = mountSpacesPage(root, transport, nativeAppSubUrl(context), context.repositoryId);
  let disposed = false;
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    transport.dispose();
    page.dispose();
    styles.forEach((style) => style.remove());
    root.classList.remove('soda-spaces-page-mount');
  };
  // Host lifecycle (NFX-D05): ordinary navigation updates/suspends/resumes
  // with full retention; only retire/dispose releases browser resources.
  const handle: MountHandle & {dispose: () => void} = {
    update: (presentation: ContributionPresentation) => page.update(presentation),
    suspend: () => page.suspend(),
    resume: (validation: ResumeValidation) => page.resume(validation),
    retire: () => dispose(),
    dispose,
  };
  return handle;
}
