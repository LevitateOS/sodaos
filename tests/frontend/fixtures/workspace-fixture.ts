// Real emitted workspace; the shared model supplies synthetic HTTP/socket peers.
import {mountSodaspaces} from '../../../frontend/spaces/sodaspaces-workspace.js';
import {object} from '../../../frontend/spaces/sodaspaces-api.js';
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
      if (!String(input).includes('/-/extensions/')) return nativeFetch(input, init);
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

function createWorkspaceFixture(mode: 'native' | 'page' = 'page', firstUse = false, subpath = '') {
  const model = installWorkspaceModel('1', undefined, firstUse);
  const apiBase =
    subpath + (mode === 'page' ? '/-/extensions/pages/soda/spaces/api/' : '/-/extensions/panels/soda/workspace/api/');
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
  const context =
    mode === 'page'
      ? {kind: 'page' as const, transport}
      : {kind: 'native' as const, transport, repositoryId: '7', pageRepositoryId: '7'};
  let api = mountSodaspaces(model.root, context);
  return {
    get api() {
      return api;
    },
    ...model,
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
    workspaceFixture: ReturnType<typeof createWorkspaceFixture>;
  }
}
window.createWorkspaceFixture = createWorkspaceFixture;
