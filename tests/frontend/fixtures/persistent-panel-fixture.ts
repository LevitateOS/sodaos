import {mount} from '../../../frontend/spaces/soda-workspace-panel-entry.js';
import {createWorkspaceModel} from './workspace-model.js';
import {object} from '../../../frontend/spaces/sodaspaces-api.js';

const model = createWorkspaceModel(location.origin, '1');
const originalFetch = window.fetch.bind(window);
Object.defineProperty(window, 'fetch', {
  configurable: true,
  value: (input: RequestInfo | URL, init?: RequestInit) =>
    String(input).includes('/-/native/')
      ? model.request(
          String(input),
          init?.method || 'GET',
          typeof init?.body === 'string' ? object(JSON.parse(init.body)) : null
        )
      : originalFetch(input, init),
});
Object.defineProperty(window, 'WebSocket', {configurable: true, value: model.Socket});
const root = document.querySelector<HTMLElement>('[data-extension-panel]');
if (!root) throw Error('Missing native panel');
const panelApiBase = location.origin + '/-/native/pages/soda/spaces/api/';
const panel = mount(root, {
  extensionId: 'soda',
  panelId: 'workspace',
  ownerId: 'workspace',
  appSubUrl: '',
  apiBase: panelApiBase,
  assetBase: '/assets/',
  sessionGeneration: model.generation,
  contractVersion: 1,
  request: (path: string, init: RequestInit = {}) =>
    fetch(panelApiBase + path, {
      ...init,
      credentials: 'same-origin',
      redirect: 'error',
      headers: new Headers({
        ...Object.fromEntries(new Headers(init.headers)),
        'X-Extension-Session-Generation': model.generation,
      }),
    }),
  stream: (path: string) => {
    const url = new URL(panelApiBase + path, location.origin);
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    const socket = new WebSocket(url);
    return {socket, close: (code?: number, reason?: string) => socket.close(code, reason)};
  },
  navigate: (path: string) => window.location.assign(path),
});

declare global {
  interface Window {
    persistentPanelFixture: {model: typeof model; dispose: () => void};
  }
}
window.persistentPanelFixture = {model, dispose: () => panel.dispose?.()};
