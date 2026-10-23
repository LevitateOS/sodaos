// Real emitted workspace; the shared model supplies synthetic HTTP/socket peers.
import {mountSodaspaces} from '../../../frontend/spaces/sodaspaces-workspace.js';
import {mount as mountWorkspacePanel} from '../../../frontend/spaces/soda-spaces-entry.js';
import {mount as mountSpacesEntry} from '../../../frontend/spaces/soda-spaces-entry.js';
import {object} from '../../../frontend/spaces/sodaspaces-api.js';
import type {ContributionNavigationOptions} from '../../../frontend/spaces/fountain-browser-contract.js';
import {createWorkspaceModel} from './workspace-model.js';
export {projectA, projectB} from './workspace-model.js';
export function installWorkspaceModel(
  actor = '1',
  repository?: {id: string; owner: string; name: string},
  firstUse = false
) {
  const root = document.querySelector<HTMLElement>('main') || document.querySelector<HTMLElement>('.page-content');
  if (!root) throw Error('Missing workspace mount');
  const model = createWorkspaceModel(location.origin, actor, repository, firstUse, sessionStorage);
  const nativeFetch = window.fetch.bind(window);
  Object.defineProperty(window, 'fetch', {
    configurable: true,
    value: async (input: RequestInfo | URL, init?: RequestInit) => {
      if (!String(input).includes('/-/native/')) return nativeFetch(input, init);
      return model.request(
        String(input),
        init?.method || 'GET',
        typeof init?.body === 'string' ? object(JSON.parse(init.body)) : null
      );
    },
  });
  Object.defineProperty(window, 'WebSocket', {configurable: true, value: model.Socket});
  return {root, ...model};
}

function createWorkspaceFixture(
  mode: 'native' | 'page' = 'page',
  firstUse = false,
  subpath = '',
  repositoryId?: string
) {
  const model = installWorkspaceModel('1', undefined, firstUse);
  const apiBase =
    subpath + (mode === 'page' ? '/-/native/pages/soda/spaces/api/' : '/-/native/panels/soda/workspace/api/');
  const transport = {
    root: model.root,
    generation: model.generation,
    assetBase: '/assets/',
    request: (path: string, init: RequestInit = {}) =>
      fetch(apiBase + path, {
        ...init,
        credentials: 'same-origin',
        redirect: 'error',
        headers: new Headers({
          ...Object.fromEntries(new Headers(init.headers)),
          'X-Extension-Session-Generation': model.generation,
        }),
      }),
    websocket: (path: string) => {
      const url = new URL(apiBase + path, location.origin);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      return new WebSocket(url);
    },
    dispose() {},
  };
  const navigations: Array<{path: string; replace: boolean}> = [];
  let navigationDeferred = false;
  const presentRoute = (route = location.pathname + location.search + location.hash) => {
    const pathname = route.split(/[?#]/)[0] || '';
    const workspace = pathname === `${subpath}/workspace` || pathname.startsWith(`${subpath}/workspace/`) || /^\/[^/]+\/[^/]+\/workspace\/?$/.test(pathname.slice(subpath.length));
    api.update({mode: workspace ? 'page' : 'panel', route, params: {}, visible: true});
  };
  // Authored fixture host: history is owned here, and the consumer receives
  // the same public presentation callback that the real native host supplies.
  const navigate = (path: string, {replace = false}: ContributionNavigationOptions = {}) => {
    navigations.push({path, replace});
    if (navigationDeferred) return;
    if (replace) window.history.replaceState(null, '', path);
    else window.history.pushState(null, '', path);
    presentRoute(path);
  };
  const context =
    mode === 'page'
      ? {kind: 'page' as const, transport, navigate, forgejoPrefix: subpath, ...(repositoryId ? {repositoryId} : {})}
      : {
          kind: 'native' as const,
          transport,
          navigate,
          forgejoPrefix: subpath,
          repositoryId: '7',
          pageRepositoryId: '7',
        };
  let api = mountSodaspaces(model.root, context);
  // Match the production entry's explicit initial inventory read.
  void api.refresh();
  window.addEventListener('popstate', () => presentRoute());
  return {
    get api() {
      return api;
    },
    ...model,
    navigations,
    presentRoute,
    deferNavigation(value: boolean) {
      navigationDeferred = value;
    },
    async remount() {
      api.dispose();
      api = mountSodaspaces(model.root, context);
      await api.ready;
      await api.refresh();
    },
  };
}
declare global {
  interface Window {
    createWorkspaceFixture: typeof createWorkspaceFixture;
    mountWorkspacePanel: typeof mountWorkspacePanel;
    workspaceFixture: ReturnType<typeof createWorkspaceFixture>;
    workspacePageModel: ReturnType<typeof installWorkspaceModel>;
  }
}
window.createWorkspaceFixture = createWorkspaceFixture;
window.mountWorkspacePanel = mountWorkspacePanel;
const pageRepository = new Map([
  ['/alice/Beta/workspace', '8'],
  ['/alice/Missing/workspace', '9'],
]);
if (location.pathname === '/workspace' || pageRepository.has(location.pathname)) {
  const model = installWorkspaceModel();
  window.workspacePageModel = model;
  model.root.dataset.extensionPage = '';
  model.root.dataset.extensionId = 'soda';
  model.root.dataset.extensionPageId = 'spaces';
  const pageApiBase = location.origin + '/-/native/pages/soda/spaces/api/';
  mountSpacesEntry(model.root, {
    extensionId: 'soda',
    pageId: 'spaces',
    ownerId: 'workspace',
    appSubUrl: '',
    ...(pageRepository.has(location.pathname) ? {repositoryId: pageRepository.get(location.pathname)!} : {}),
    apiBase: pageApiBase,
    assetBase: location.origin + '/assets/',
    sessionGeneration: model.generation,
    contractVersion: 1,
    request: (path: string, init: RequestInit = {}) =>
      fetch(pageApiBase + path, {
        ...init,
        credentials: 'same-origin',
        redirect: 'error',
        headers: new Headers({
          ...Object.fromEntries(new Headers(init.headers)),
          'X-Extension-Session-Generation': model.generation,
        }),
      }),
    stream: (path: string) => {
      const url = new URL(pageApiBase + path, location.origin);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      const socket = new WebSocket(url);
      return {socket, close: (code?: number, reason?: string) => socket.close(code, reason)};
    },
    navigate: (path: string) => window.location.assign(path),
  });
}
