import {mount as mountWorkspace} from '../../../frontend/spaces/soda-spaces-entry.js';
import type {
  ContributionPresentation,
  MountContext,
  MountHandle,
  ResumeValidation,
  RetireReason,
} from '../../../frontend/spaces/fountain-browser-contract.js';
import {createWorkspaceModel} from './workspace-model.js';
import {object} from '../../../frontend/spaces/sodaspaces-api.js';

// Authored fixture model; the native host supplies the actual mount transport,
// navigation and lifecycle. The probe only records calls to the real Soda entry.
const model = createWorkspaceModel(location.origin, '1', {id: '7', owner: 'team', name: 'repo'});
await model.request('/api/factory/intents', 'POST', {
  id: 'd' + '2'.repeat(24),
  intent: 'd' + '1'.repeat(24),
  repository: '7',
  objective: 'Retained fixture request',
  ideas: [],
  tasks: [],
  state: 'draft',
  max_issues: 5,
  planning_calls: 3,
});
const initialWrites = model.calls.filter((call) => call.method !== 'GET').length;
type LifecycleCall = {
  method: 'mount' | 'update' | 'suspend' | 'resume' | 'retire' | 'dispose';
  presentation?: ContributionPresentation;
  validation?: ResumeValidation;
  reason?: RetireReason;
};
const lifecycle: LifecycleCall[] = [];
const originalFetch = window.fetch.bind(window);
Object.defineProperty(window, 'fetch', {
  configurable: true,
  value: (input: RequestInfo | URL, init?: RequestInit) => {
    const url = input instanceof Request ? input.url : String(input);
    return url.includes('/-/native/pages/soda/spaces/api/')
      ? model.request(url, init?.method || 'GET', typeof init?.body === 'string' ? object(JSON.parse(init.body)) : null)
      : originalFetch(input, init);
  },
});
Object.defineProperty(window, 'WebSocket', {configurable: true, value: model.Socket});
Object.assign(window, {config: {appSubUrl: '/forge', pageData: {}, csrfToken: 'fixture'}});

export function mount(root: HTMLElement, context: MountContext): MountHandle {
  lifecycle.push({method: 'mount'});
  const handle = mountWorkspace(root, context);
  return {
    update(presentation) {
      lifecycle.push({method: 'update', presentation});
      handle.update?.(presentation);
    },
    suspend() {
      lifecycle.push({method: 'suspend'});
      handle.suspend?.();
    },
    resume(validation) {
      lifecycle.push({method: 'resume', validation});
      handle.resume?.(validation);
    },
    retire(reason) {
      lifecycle.push({method: 'retire', reason});
      handle.retire?.(reason);
    },
    dispose() {
      lifecycle.push({method: 'dispose'});
      handle.dispose?.();
    },
  };
}

declare global {
  interface Window {
    persistentPanelFixture: {model: typeof model; lifecycle: LifecycleCall[]; initialWrites: number};
  }
}
window.persistentPanelFixture = {model, lifecycle, initialWrites};
// Load the bundled current Fountain host, without injecting a navigator or
// recreating hostMountHandle/scoped request logic in this fixture.
const native = object(await import(location.origin + '/assets/native-host.js'));
if (typeof native.initExtensionPages !== 'function' || typeof native.initExtensionWorkspace !== 'function')
  throw Error('Missing current native host');
native.initExtensionPages();
native.initExtensionWorkspace();
