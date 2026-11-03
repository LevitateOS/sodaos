import {fountainFeedContext, fountainRepositoryNavigation} from '../../../frontend/spaces/soda-factory-feed-host.js';
import {fountainIssueContexts} from '../../../frontend/spaces/soda-factory-issue-host.js';
import {SodaFactoryIssue} from '@factory/product/frontend/factory/issues/detail.js';
import {SodaFactoryIssues} from '@factory/product/frontend/factory/issues/issues.js';
import type {FactoryTransport} from '@factory/product/frontend/factory/api/transport.js';
// Browser-realm fixture only. Production modules are emitted separately, never bundled into this double.
import {mountProjectControls} from '../../../frontend/spaces/sodaspaces-project.js';
import {SodaFactoryFeed} from '@factory/product/frontend/factory/feed/feed.js';
import '../../../frontend/spaces/sodaspaces-project-work.js';
import {mount as mountRepositoryEntry} from '../../../frontend/spaces/soda-repository-entry.js';
import {createIntentModel} from './intent-model.js';
const browserFetch = window.fetch.bind(window);
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
  let forwardedAPI: string | undefined;
  const repositoryLabels = new Map([['7', 'alice/factory']]);
  const fetchFixture = async (input: RequestInfo | URL, init?: RequestInit) => {
    const call: Call = {
      url: String(input),
      method: init?.method || 'GET',
      ...(typeof init?.body === 'string' ? {body: init.body} : {}),
      headers: Object.fromEntries(new Headers(init?.headers).entries()),
    };
    calls.push(call);
    if (forwardedAPI && call.url.includes('/api/'))
      return browserFetch(new URL('api/' + call.url.split('/api/')[1], forwardedAPI), init);
    const signal = init?.signal;
    if (signal?.aborted) throw signal.reason;
    const pending = reply?.(call);
    // Mirror production fetch abort semantics: a hung reply must lose to an
    // aborted signal (e.g. the settings preparation GET deadline) instead of
    // hanging the awaiting UI forever.
    const override = await (signal
      ? Promise.race([
          pending,
          new Promise<null>((_, reject) => signal.addEventListener('abort', () => reject(signal.reason), {once: true})),
        ])
      : pending);
    if (override) {
      if (call.url.includes('/api/repositories?q=')) {
        const page = await override
          .clone()
          .json()
          .catch(() => undefined);
        for (const item of page?.items ?? [])
          if (typeof item.id === 'string' && typeof item.owner === 'string' && typeof item.name === 'string')
            repositoryLabels.set(item.id, `${item.owner}/${item.name}`);
      }
      return override;
    }
    const {url, method, body: encoded} = call;
    if (call.headers['x-extension-session-generation'] !== 'fixture-generation')
      return Response.json({error: {code: 'reauthentication_required'}}, {status: 403});
    if (url.endsWith('/api/factory/identity'))
      return Response.json({host: 'a'.repeat(32), actor: '1', native_actor: '1'});
    if (url.endsWith('/api/factory/native-repositories/resolve')) {
      const native = String(JSON.parse(encoded || '{}').repository);
      return Response.json({
        host: 'a'.repeat(32),
        actor: '1',
        repository: native,
        native_repository: native,
        name: repositoryLabels.get(native) ?? `alice/repository-${native}`,
      });
    }
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
      body = {
        repository: url.match(/repositories\/(\d+)/)?.[1],
        observed_unix: 1,
        viewer: {
          code_write: true,
          owner_or_admin: true,
          environment_owner: true,
          operator: true,
          member: true,
          session_current: true,
        },
        editable: {
          policy: {editable: true},
          activity_settings: {editable: true},
          operator_grant: {editable: true},
          capacity: {editable: true},
          environment_grant: {editable: true},
          sponsorships: [{connection: 'connection-observed', editable: true}],
        },
        effective: {},
        required_actions: [],
        preparation: null,
        usage: [],
        planning: [],
      };
    else if (url.endsWith('/api/identity/connections') || url.endsWith('/api/identity/enrollments')) body = [];
    else if (url.endsWith('/api/repositories/7/factory'))
      body = {
        repository: '7',
        policy: null,
        operator_grant: null,
        capacity: {revision: 4, max_concurrent_runs: 2},
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
    else if (url.endsWith(`/api/environments/${environmentID}/members`)) body = {items: [], can_manage: state.admin};
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
    // Host navigation is recorded, never performed, so the page stays put.
    navigate(path: string) {
      document.body.dataset.navigated = path;
      document.body.dataset.navigateCount = String(Number(document.body.dataset.navigateCount ?? '0') + 1);
    },
    dispose() {},
  };
  const api = mountProjectControls(root, {expectedUserId: '1', repositoryId: '7', settings: true, transport});
  let repositoryHandle: ReturnType<typeof mountRepositoryEntry> | undefined;
  let repositoryRoot: HTMLElement | undefined;
  let followRepositoryNavigation = false;
  window.addEventListener('popstate', () => {
    if (followRepositoryNavigation)
      repositoryHandle?.update?.({
        mode: 'page',
        route: location.pathname + location.search + location.hash,
        params: {},
        visible: true,
      });
  });
  const issueHost: {
    contexts?: Awaited<ReturnType<typeof fountainIssueContexts>>;
    transport?: FactoryTransport & {generation: string};
    lifetime: AbortController;
    list?: SodaFactoryIssues;
    detail?: SodaFactoryIssue;
    opened: {kind: string; url: string}[];
    opening?: AbortSignal;
    release?: () => void;
  } = {lifetime: new AbortController(), opened: []};
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
    intentModel: createIntentModel,
    get repositoryHandle() {
      return repositoryHandle;
    },
    get repositoryRoot() {
      return repositoryRoot;
    },
    mountRepository(native: string, route: string, prefix = '', followNavigation = false) {
      repositoryHandle?.dispose?.();
      repositoryRoot?.remove();
      history.replaceState(null, '', route);
      repositoryRoot = document.createElement('div');
      followRepositoryNavigation = followNavigation;
      const navigate = (path: string) => {
        transport.navigate(path);
        if (followRepositoryNavigation) {
          history.pushState(null, '', path);
          repositoryHandle?.update?.({mode: 'page', route: path, params: {}, visible: true});
        }
      };
      if (followNavigation)
        repositoryRoot.addEventListener('click', (event) => {
          const anchor = event.composedPath().find((node) => node instanceof HTMLAnchorElement);
          if (
            !(anchor instanceof HTMLAnchorElement) ||
            event.defaultPrevented ||
            new URL(anchor.href).origin !== location.origin
          )
            return;
          if (
            !anchor.pathname.startsWith(
              prefix + '/' + route.slice(prefix.length).split('/').filter(Boolean).slice(0, 2).join('/') + '/feed'
            )
          )
            return;
          event.preventDefault();
          navigate(anchor.pathname + anchor.search + anchor.hash);
        });
      root.append(repositoryRoot);
      repositoryHandle = mountRepositoryEntry(repositoryRoot, {
        extensionId: 'soda',
        pageId: 'repository-feed',
        repositoryId: native,
        appSubUrl: prefix,
        apiBase: location.origin + '/-/native/pages/soda/spaces/api/',
        assetBase: '/assets/',
        sessionGeneration: transport.generation,
        contractVersion: 1,
        request: transport.request,
        stream: () => {
          throw Error('Repository fixture has no terminal');
        },
        navigate,
      });
    },
    settingsContext(nativeRepository: string) {
      return {transport, nativeRepository, environment: environmentID, profile: fixtureProfile};
    },
    feedContext: fountainFeedContext,
    repositoryNavigation: fountainRepositoryNavigation,
    issueContexts: fountainIssueContexts,
    issueHost,
    issueElement: () => new SodaFactoryIssue(),
    issuesElement: () => new SodaFactoryIssues(),
    browserFetch,
    mountFeed() {
      const feed = new SodaFactoryFeed();
      root.append(feed);
      void fountainFeedContext(transport, {repository: '7'}).then((context) => {
        if (feed.isConnected) feed.context = context;
      });
      return feed;
    },
    button,
    showButton,
    setReply(value: typeof reply) {
      reply = value;
    },
    forwardAPI(value: string) {
      forwardedAPI = value;
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
