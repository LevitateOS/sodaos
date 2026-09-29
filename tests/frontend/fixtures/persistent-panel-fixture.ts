import {mount} from '../../../frontend/spaces/soda-workspace-panel-entry.js';
import {createWorkspaceModel} from './workspace-model.js';
import {object} from '../../../frontend/spaces/sodaspaces-api.js';

const model = createWorkspaceModel(location.origin, '1');
const originalFetch = window.fetch.bind(window);
Object.defineProperty(window, 'fetch', {
  configurable: true,
  value: (input: RequestInfo | URL, init?: RequestInit) =>
    String(input).includes('/-/extensions/panels/soda/workspace/api/')
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
const panel = mount(root, {
  extensionId: 'soda',
  panelId: 'workspace',
  apiBase: '/-/extensions/panels/soda/workspace/api/',
  assetBase: '/assets/',
  sessionGeneration: model.generation,
});

declare global {
  interface Window {
    persistentPanelFixture: {model: typeof model; dispose: () => void};
  }
}
window.persistentPanelFixture = {model, dispose: () => panel.dispose()};
