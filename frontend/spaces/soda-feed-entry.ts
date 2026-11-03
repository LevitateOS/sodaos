import {fountainFeedContext} from './soda-factory-feed-host.js';
import type {FactoryFeedContext} from '@factory/product/frontend/factory/feed/feed.js';
import {html, render} from 'lit';
import type {MountHandle} from './fountain-browser-contract.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {prepareExtensionMount} from './soda-extension.js';
import {nativeAppSubUrl} from './soda-native-paths.js';
import '@factory/product/frontend/factory/feed/feed.js';
import './sodaspaces-applications.js';

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
  let disposed = false;
  let factoryContext: FactoryFeedContext | undefined;
  let pending: AbortController | undefined;
  let error = '';
  const show = () =>
    render(
      html`
        ${
          // Apps (M14) has no sub-menu: each launch card links to the App
          // tab of the repository it was built from.
          applications
            ? html`<header class="soda-feed-page-heading">
                <div>
                  <h1>Apps</h1>
                  <p>Open the apps your repositories build. Each app opens at its own address in a new tab.</p>
                </div>
              </header>`
            : html`<header class="soda-feed-page-heading">
                <div>
                  <h1>Feed</h1>
                  <p>Role updates, choices and results from the repositories you follow.</p>
                </div>
              </header>`
        }
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
            : html`${error ? html`<p role="alert">${error}</p>` : !factoryContext ? html`<p role="status">Loading feed…</p>` : ''}
                <soda-factory-feed .context=${visible ? factoryContext : undefined}></soda-factory-feed>`
        }
      `,
      content
    );
  const load = () => {
    if (applications || !visible || disposed || factoryContext || pending) return;
    const request = new AbortController();
    pending = request;
    error = '';
    void fountainFeedContext(transport, {repository, global: true, prefix}, request.signal).then(
      (value) => {
        if (disposed || pending !== request || request.signal.aborted) return;
        pending = undefined;
        factoryContext = value;
        show();
      },
      () => {
        if (disposed || pending !== request || request.signal.aborted) return;
        pending = undefined;
        error = 'Repository information is unavailable.';
        show();
      }
    );
  };
  const present = (value: boolean) => {
    visible = value;
    if (!visible) {
      pending?.abort();
      pending = undefined;
    } else load();
    show();
  };
  show();
  load();
  return {
    update(presentation) {
      present(presentation.visible);
    },
    suspend() {
      present(false);
    },
    resume(validation) {
      if (validation.sessionGeneration === transport.generation) {
        present(true);
      }
    },
    retire() {
      disposed = true;
      present(false);
    },
    dispose() {
      disposed = true;
      pending?.abort();
      render(html``, content);
      content.remove();
      styles.forEach((el) => el.remove());
      transport.dispose();
    },
  };
}
