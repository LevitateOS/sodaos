// Browser-realm fixture only. Production modules are emitted separately, never bundled into this double.
import {mountProjectControls} from '../../../frontend/spaces/sodaspaces-project.js';
import {SodaFactoryFeed} from '../../../frontend/spaces/sodaspaces-factory-feed.js';
export const environmentID = 'p0123456789abcdef01234567';
export const fixtureProfile = {
  id: 'rocky-headless',
  distribution: 'rocky',
  version: '10.2',
  interface: 'headless',
  architecture: 'amd64',
  image: 'sha256:' + 'a'.repeat(64),
  revision: 'b'.repeat(40),
};
export const fixtureFingerprint = 'SHA256:' + 'A'.repeat(43);
export interface Call {
  url: string;
  method: string;
  body?: string;
  headers: Record<string, string>;
}
export interface State {
  tailnetCreateOutcome: 'queued' | 'unconfirmed';
  tailnetAvailable: boolean;
  tailnetEnabled: boolean;
  tailnetDefault: boolean;
  tailnetState: string;
  tailnetRevision: string;
  running: boolean;
  member: boolean;
  admin: boolean;
  absent: boolean;
  saved: string[];
  installed: string[];
  provisioned: boolean;
  unavailable: boolean;
}
function createFixture(extra: Partial<State> = {}) {
  const root = document.querySelector('main');
  if (!root) throw Error('fixture mount missing');
  const calls: Call[] = [];
  const state: State = {
    tailnetCreateOutcome: 'queued',
    tailnetAvailable: false,
    tailnetEnabled: false,
    tailnetDefault: false,
    tailnetState: 'unconfirmed',
    tailnetRevision: '0',
    running: true,
    member: true,
    admin: true,
    absent: false,
    saved: [fixtureFingerprint],
    installed: [fixtureFingerprint],
    provisioned: true,
    unavailable: false,
    ...extra,
  };
  const network = (saved = false) => ({
    project: environmentID,
    revision: state.tailnetRevision,
    binding: state.tailnetEnabled || state.tailnetRevision !== '0' ? 'a'.repeat(32) : '',
    enabled: state.tailnetEnabled,
    saved,
    state: state.tailnetAvailable ? state.tailnetState : 'runtime-unsupported',
    outcome: saved ? 'queued' : 'observed',
    ...(state.tailnetAvailable ? {available_binding: 'a'.repeat(32), available_network: 'soda.example.test'} : {}),
    ...(state.tailnetState === 'connected' ? {addresses: ['100.64.0.2'], dns_name: 'project.soda.ts.net'} : {}),
  });
  let reply: ((call: Call) => Promise<Response | null>) | undefined;
  const fetchFixture = async (input: RequestInfo | URL, init?: RequestInit) => {
    const call: Call = {
      url: String(input),
      method: init?.method || 'GET',
      ...(typeof init?.body === 'string' ? {body: init.body} : {}),
      headers: Object.fromEntries(new Headers(init?.headers).entries()),
    };
    calls.push(call);
    const override = await reply?.(call);
    if (override) return override;
    const {url, method, body: encoded} = call;
    if (call.headers['x-extension-session-generation'] !== 'fixture-generation')
      return Response.json({error: {code: 'reauthentication_required'}}, {status: 403});
    let body: unknown;
    if (method !== 'GET') {
      const input: unknown = JSON.parse(encoded || '{}');
      if (!input || typeof input !== 'object') throw Error('invalid fixture action');
      if (url.endsWith('/api/environments')) {
        const selection = 'tailnet' in input ? input.tailnet : null;
        const enabled =
          selection && typeof selection === 'object' && 'enabled' in selection && selection.enabled === true;
        state.absent = false;
        body = {
          id: environmentID,
          repository_id: '7',
          provisioned: true,
          profile: fixtureProfile,
          ...(enabled ? {tailnet_outcome: state.tailnetCreateOutcome} : {}),
        };
      } else if (url.endsWith('/tailnet')) {
        state.tailnetEnabled = 'action' in input && input.action !== 'disable';
        state.tailnetRevision = state.tailnetRevision === '0' ? 'b'.repeat(32) : 'c'.repeat(32);
        body = network(true);
      } else if (url.endsWith('/join')) body = {login: 'alice'};
      else if (url.endsWith('/lifecycle'))
        body = {
          environment: {id: environmentID, running: 'action' in input && input.action === 'start'},
          boot_enabled: 'action' in input && input.action === 'start',
        };
      else if (url.endsWith('/access-keys'))
        body = {
          applied: true,
          login: 'alice',
          revision: 'a'.repeat(64),
          installed_fingerprints: 'saved_fingerprints' in input ? input.saved_fingerprints : null,
        };
      else if (method === 'DELETE') body = {removed: true, existing_project_access_changed: false};
      else body = {items: []};
    } else if (url.endsWith('/tailnet-options'))
      body = {
        revision: state.tailnetAvailable ? 'b'.repeat(32) : '0',
        binding: state.tailnetAvailable ? 'a'.repeat(32) : '',
        tailnet: state.tailnetAvailable ? 'soda.example.test' : '',
        available: state.tailnetAvailable,
        default: state.tailnetAvailable && state.tailnetDefault,
      };
    else if (url.endsWith('/tailnet')) body = network();
    else if (/\/api\/repositories\/\d+\/factory\/access$/.test(url))
      body = {repository: url.match(/repositories\/(\d+)/)?.[1], observed_unix: 1,
        viewer: {code_write: true, owner_or_admin: true, environment_owner: true, operator: true, member: true, session_current: true},
        editable: {policy: {editable:true}, activity_settings: {editable:true}, operator_grant: {editable:true}, capacity: {editable:true}, environment_grant: {editable:true},
          sponsorships: [{connection:'connection-observed',editable:true}], usage_budgets: [{connection:'connection-observed',editable:true}]},
        effective: {}, required_actions: [], preparation: null, usage: [], planning: []};
    else if (url.endsWith('/api/identity/connections') || url.endsWith('/api/identity/enrollments')) body = [];
    else if (url.endsWith('/api/repositories/7/factory'))
      body = {
        repository: '7',
        policy: null,
        operator_grant: null,
        capacity: {revision: 4, max_concurrent_runs: 2, max_queued: 8},
        sponsorships: [],
        environment_grant: null,
        preparation: {project: environmentID, requirements: '', approval: '', ready: false},
        queue: {queued: 1, active: 0},
        effective: {effective: false, dispatch_open: false, missing: ['policy_missing'], authority: {}},
      };
    else if (url.endsWith(`/api/environments/${environmentID}/preparation`))
      body = {project: environmentID, hold: false, hold_revision: 3, preparations: []};
    else if (url.endsWith('/api/repositories/7/profiles')) body = {items: [fixtureProfile]};
    else if (url.includes('/api/environments?'))
      body = {
        repository: {id: '7', owner: 'alice', name: 'demo'},
        can_create: state.absent,
        items: state.absent ? [] : [{id: environmentID, repository_id: '7'}],
      };
    else if (url.endsWith('/api/me/development-keys'))
      body = {items: state.saved.map((fingerprint, i) => ({id: String(i + 1), fingerprint}))};
    else if (url.endsWith('/os'))
      body = {
        environment: {id: environmentID, running: state.running, image: 'sha256:' + 'a'.repeat(64)},
        os_release: state.running ? {id: 'rocky', version: '9.7', name: 'Rocky Linux 9.7'} : null,
        os_release_unavailable: !state.running,
      };
    else if (url.endsWith('/lifecycle'))
      body = {environment: {id: environmentID, running: state.running}, boot_enabled: state.running};
    else if (url.endsWith('/access-keys'))
      body = {
        login: 'alice',
        revision: 'a'.repeat(64),
        installed_fingerprints: state.installed,
        saved_fingerprints: state.saved,
      };
    else if (url.endsWith('/connection'))
      body = {
        login: 'alice',
        connection: {environment: {id: environmentID, running: true, ip: '10.89.0.2'}, fingerprint: fixtureFingerprint},
      };
    else
      body = {
        environment: {id: environmentID, repository_id: '7', provisioned: state.provisioned},
        observed: state.unavailable ? null : {id: environmentID, running: state.running},
        login: state.member ? 'alice' : '',
        environment_administrator: state.admin,
        execution_allowed: true,
        native_unavailable: state.unavailable,
        authority_unavailable: false,
      };
    return Response.json(body);
  };
  Object.defineProperty(window, 'fetch', {value: fetchFixture, configurable: true});
  const transport = {
    root,
    generation: 'fixture-generation',
    assetBase: '/assets/',
    request: (path: string, init: RequestInit = {}) =>
      fetchFixture('/-/native/pages/soda/spaces/api/' + path, {
        ...init,
        headers: new Headers({
          ...Object.fromEntries(new Headers(init.headers)),
          'X-Extension-Session-Generation': 'fixture-generation',
        }),
      }),
    websocket: () => {
      throw Error('Project fixture has no terminal');
    },
    dispose() {},
  };
  const api = mountProjectControls(root, {expectedUserId: '1', repositoryId: '7', settings: true, transport});
  const button = (text: string) => {
    const b = [...root.querySelectorAll('button')].find((b) => b.textContent?.trim() === text);
    if (!b) throw Error('Missing button: ' + text);
    return b;
  };
  const showButton = async (text: string) => {
    const b = button(text),
      panel = b.closest('[role=tabpanel]');
    if (panel) root.querySelector<HTMLElement>('[aria-controls="' + panel.id + '"]')?.click();
    await api.ready;
    return b;
  };
  return {
    api,
    state,
    calls,
    get terminals() {
      return root.querySelectorAll('soda-terminal');
    },
    root,
    mountFeed() {
      const feed = new SodaFactoryFeed();
      root.append(feed);
      feed.context = {transport, repository: '7'};
      return feed;
    },
    button,
    showButton,
    setReply(value: typeof reply) {
      reply = value;
    },
  };
}
declare global {
  interface Window {
    createDrawerFixture: typeof createFixture;
    drawerFixture: ReturnType<typeof createFixture>;
  }
}
window.createDrawerFixture = createFixture;
