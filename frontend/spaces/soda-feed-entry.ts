import {html, render} from 'lit';
import type {MountHandle} from './fountain-browser-contract.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {prepareExtensionMount} from './soda-extension.js';
import {nativeAppSubUrl} from './soda-native-paths.js';
import './sodaspaces-factory-feed.js';
import './sodaspaces-applications.js';
import './soda-providers.js';

export function mount(root: HTMLElement, context: ExtensionMountContext): MountHandle {
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
  let repository =
    context.repositoryId ?? (applications ? (new URLSearchParams(location.search).get('repository') ?? '') : '');
  const global = context.pageId === 'feed' || applications;
  const parts = location.pathname.slice(prefix.length).split('/').filter(Boolean);
  const name = global ? undefined : parts.slice(0, 2).map(decodeURIComponent).join('/');
  let visible = true;
  const show = () =>
    render(
      html`
        <header class="soda-feed-page-heading">
          <div><h1>${applications ? 'Applications' : global ? 'Feed' : name}</h1></div>
          <nav aria-label="Project navigation">
            ${name ? html`<a href=${`${prefix}/${name}`}>Repository ↗</a><a href=${`${prefix}/${name}/issues`}>Issues</a><a href=${`${prefix}/workspace/new?repository=${repository}`}>New work</a>` : ''}
            <a href=${`${prefix}/workspace`}>Projects</a>
            <a href=${`${prefix}/applications${!global && repository ? '?repository=' + repository : ''}`}
              >${applications && repository ? 'All applications' : 'Applications'}</a
            >
            ${!global ? html`<a href=${`${prefix}/feed`}>Following</a>` : ''}
            <soda-providers .transport=${visible ? transport : undefined}></soda-providers>
          </nav>
        </header>
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
                .repository=${repository}
              ></soda-applications>`
            : html`<soda-factory-feed
                .context=${visible ? {transport, repository, repositoryName: name, global} : undefined}
              ></soda-factory-feed>`
        }
      `,
      content
    );
  show();
  return {
    update(presentation) {
      if (applications) repository = new URLSearchParams(location.search).get('repository') ?? '';
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
