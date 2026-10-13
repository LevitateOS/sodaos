import {prepareExtensionMount} from './soda-extension.js';
import type {ExtensionMountContext} from './soda-extension.js';
import type {MountHandle} from './fountain-browser-contract.js';
import {mountSodaspaces} from './sodaspaces-workspace.js';
import {nativeAppSubUrl} from './soda-native-paths.js';

// One shared Workspace owner (NFX-D02/H-OWNER): the spaces page and the
// workspace panel declare the same owner and entry, so the host mounts a
// single retained instance (the panel root) and drives it through this
// handle. The page root is a presentation slot the host never mounts as a
// competing owner; this entry therefore has no frame or competing-panel
// detection and no page-entry no-op workaround.
export function mount(root: HTMLElement, context: ExtensionMountContext): MountHandle {
  if (!((context.pageId === 'spaces' && !context.panelId) || (context.panelId === 'workspace' && !context.pageId)))
    throw Error('Invalid workspace mount');
  const transport = prepareExtensionMount(root, context);
  const styles = [
    'components.css',
    'sodaspaces.css',
    'sodaspaces-page.css',
    'sodaspaces-drawer.css',
    'sodaspaces-terminal.css',
  ].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  root.append(...styles);
  const prefix = nativeAppSubUrl(context);
  const workspace = mountSodaspaces(root, {
    kind: context.pageId === 'spaces' ? 'page' : 'native',
    transport,
    forgejoPrefix: prefix,
    ...(context.navigate ? {navigate: context.navigate} : {}),
    ...(context.repositoryId ? {repositoryId: context.repositoryId} : {}),
  });
  void workspace.refresh();
  let disposed = false;
  let retired = false;
  let suspended = false;
  let visible = true;
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    transport.dispose();
    workspace.dispose();
    styles.forEach((style) => style.remove());
    root.classList.remove('soda-spaces-page-mount');
  };
  return {
    dispose,
    update(presentation) {
      if (disposed || retired) return;
      visible = presentation.visible;
      const kind = presentation.mode === 'page' ? 'page' : 'native';
      root.classList.toggle('soda-spaces-page-mount', kind === 'page');
      workspace.present(kind, presentation.repositoryId);
      workspace.setVisible(visible && !suspended);
    },
    suspend() {
      if (disposed || retired || suspended) return;
      suspended = true;
      workspace.setVisible(false);
    },
    resume(validation) {
      if (disposed || retired) return;
      if (validation.sessionGeneration !== transport.generation) {
        workspace.invalidate();
        return;
      }
      suspended = false;
      workspace.setVisible(visible);
    },
    retire() {
      if (retired) return;
      retired = true;
      dispose();
    },
  };
}
