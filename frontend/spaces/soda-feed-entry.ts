import {html, render} from 'lit';
import type {MountHandle} from './fountain-browser-contract.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {prepareExtensionMount} from './soda-extension.js';
import {nativeAppSubUrl} from './soda-native-paths.js';
import './sodaspaces-factory-feed.js';
import './sodaspaces-applications.js';
import './soda-providers.js';

export function mount(root: HTMLElement, context: ExtensionMountContext): MountHandle {
  // The repository page has its own entry (soda-repository-entry.ts); this one
  // serves only the global Feed and Applications pages.
  if (context.pageId !== 'feed' && context.pageId !== 'applications') throw Error('Invalid global Soda page mount');
  const transport = prepareExtensionMount(root, context);
  const styles = ['components.css', 'sodaspaces-page.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  root.append(...styles);
  const content = document.createElement('div');
  content.className = 'soda-feed-page';
  root.append(content);
  const prefix = nativeAppSubUrl(context);
  const applications = context.pageId === 'applications';
  const repository = '';
  let visible = true;
  const show = () =>
    render(
      html`
        ${
          // The providers dialog host stays mounted on every feed page for
          // the navbar button. Apps (M14) has no sub-menu: each launch card
          // links to the App tab of the repository it was built from.
          applications
            ? html`<header class="soda-feed-page-heading">
                <div><h1>Apps</h1><p>Open the apps your repositories build. Each app opens at its own address in a new tab.</p></div>
              </header>`
            : html`<header class="soda-feed-page-heading">
                <div><p class="soda-page-eyebrow">Across your projects</p><h1>Following</h1><p>Agent updates, decisions and results.</p></div>
                <nav aria-label="Project navigation">
                  <a href=${`${prefix}/workspace`}>Projects</a>
                  <a href=${`${prefix}/applications`}>Applications</a>
                </nav>
              </header>`
        }
        <soda-providers .externalTrigger=${true} .transport=${visible ? transport : undefined}></soda-providers>
        ${
          applications
            ? html`<soda-applications
                .request=${async (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => {
                  const response = await transport.request(path.replace(/^\/api\//, ''), {
                    method: body ? 'POST' : 'GET',
                    signal: signal ?? null,
                    ...(body ? {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)} : {}),
                  });
                  const value = await response.json();
                  if (!response.ok) throw Error(value.error?.message ?? 'Application request failed.');
                  return value;
                }}
                .active=${visible}
                .sessionKey=${transport.generation}
                .expanded=${true}
                .layout=${'apps'}
                .prefix=${prefix}
              ></soda-applications>`
            : html`<soda-factory-feed
                .context=${visible ? {transport, repository, global: true} : undefined}
              ></soda-factory-feed>`
        }
      `,
      content
    );
  show();
  return {
    update(presentation) {
      visible = presentation.visible;
      show();
    },
    suspend() {
      visible = false;
      show();
    },
    resume(validation) {
      if (validation.sessionGeneration === transport.generation) {
        visible = true;
        show();
      }
    },
    retire() {
      visible = false;
      show();
    },
    dispose() {
      render(html``, content);
      content.remove();
      styles.forEach((el) => el.remove());
      transport.dispose();
    },
  };
}
