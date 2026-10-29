import type {MountHandle} from './fountain-browser-contract.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {prepareExtensionMount} from './soda-extension.js';
import {openProvidersEvent, providersReadyAttribute} from './soda-providers-events.js';
import './soda-providers.js';
import type {SodaProviders} from './soda-providers.js';

/** One-shot flag from older navbar code that navigated here to open the dialog. */
const legacyOpenFlag = 'soda-open-providers';

// The AI accounts dialog on every Forge page. Fountain mounts this panel on
// each signed-in document and retains it across navigation, so the dialog and
// its status are ready before the navbar robot is clicked. The panel region
// is hidden outside the retired Workspace, so the dialog lives in a
// body-level host instead of the panel root.
export function mount(root: HTMLElement, context: ExtensionMountContext): MountHandle {
  if (context.panelId !== 'providers' || context.pageId) throw Error('Invalid AI accounts panel mount');
  const transport = prepareExtensionMount(root, context);
  const host = document.createElement('div');
  host.className = 'soda-providers-host';
  const styles = ['components.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  const providers = document.createElement('soda-providers') as SodaProviders;
  host.append(...styles, providers);
  document.body.append(host);
  providers.transport = transport;
  const open = () => void providers.open();
  document.addEventListener(openProvidersEvent, open);
  document.documentElement.setAttribute(providersReadyAttribute, '');
  try {
    if (sessionStorage.getItem(legacyOpenFlag) === '1') {
      sessionStorage.removeItem(legacyOpenFlag);
      open();
    }
  } catch {
    /* Storage can be blocked. */
  }
  return {
    update() {},
    suspend() {},
    resume(validation) {
      // A different session must not reuse this account's transport.
      if (validation.sessionGeneration !== transport.generation) providers.transport = undefined;
    },
    retire() {
      providers.transport = undefined;
    },
    dispose() {
      document.removeEventListener(openProvidersEvent, open);
      document.documentElement.removeAttribute(providersReadyAttribute);
      providers.transport = undefined;
      host.remove();
      transport.dispose();
    },
  };
}
