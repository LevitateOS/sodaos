import {SodaApplications} from '../../../frontend/spaces/sodaspaces-applications';
import type {ApplicationRecord} from '../../../frontend/spaces/sodaspaces-application-response';
import {mountSodaspaces} from '../../../frontend/spaces/sodaspaces-workspace';
import type {PreparedExtensionMount} from '../../../frontend/spaces/soda-extension';

interface ApplicationFixture {
  element: SodaApplications;
  calls: {path: string; body?: Record<string, unknown>}[];
  fail: boolean;
  delay: boolean;
  release: (() => void) | undefined;
  failRestore: boolean;
  setupExpiresAt: number | undefined;
}
interface WorkspaceDeliveryFixture {
  calls: {path: string; body?: Record<string, unknown>}[];
  deliveryId: string;
  present: (kind: 'page' | 'native', repositoryId?: string) => void;
}
declare global {
  interface Window {
    applicationFixture: ApplicationFixture;
    createApplicationFixture: (operator?: boolean) => Promise<void>;
    createApplicationDeliveryFixture: (update?: boolean) => Promise<void>;
    workspaceDeliveryFixture: WorkspaceDeliveryFixture;
    createWorkspaceDeliveryFixture: () => Promise<void>;
  }
}
const initial = (): ApplicationRecord => ({
  environment: 'company',
  application: {
    id: 'a' + '1'.repeat(24),
    revision: 1,
    name: 'Example',
    artifact_sha256: '2'.repeat(64),
    bundle_sha256: '3'.repeat(64),
    tools_sha256: '4'.repeat(64),
    runtime_image: 'sha256:' + '5'.repeat(64),
    image: 'sha256:' + '6'.repeat(64),
    data: 'a' + '7'.repeat(24),
    owner_team: '12345678-1234-1234-1234-123456789abc',
    origin: 'https://app.example.test',
    port: 20001,
  },
  administrator_id: '41',
  operator: false,
  observation: {
    id: 'a' + '1'.repeat(24),
    revision: 1,
    container: 'a'.repeat(64),
    image: 'sha256:' + '6'.repeat(64),
    origin: 'https://app.example.test',
    running: false,
    ready: false,
  },
});
window.createApplicationFixture = async (operator = false) => {
  const element = new SodaApplications(),
    record = initial();
  record.operator = operator;
  const snapshots = [{id: 'a' + '8'.repeat(24), created_at: 1792713600, application: {...record.application}}];
  const fixture: ApplicationFixture = {
    element,
    calls: [],
    fail: false,
    delay: false,
    release: undefined,
    failRestore: false,
    setupExpiresAt: undefined,
  };
  const setups: {
    id: string;
    application: Record<string, unknown>;
    administrator_id: number;
    expires_at: number;
    state: string;
    observation: {container: string; running: boolean; ready: boolean} | null;
  }[] = [];
  element.active = true;
  element.sessionKey = 'generation-1';
  element.request = async (path, body) => {
    fixture.calls.push({path, ...(body ? {body} : {})});
    if (fixture.delay)
      await new Promise<void>((resolve) => {
        fixture.release = resolve;
      });
    if (fixture.fail) throw Error('private diagnostic');
    if (path === '/api/applications/setups') {
      if (!operator) throw Error('operator required');
      if (!body)
        return {
          setups: setups
            .filter((row) => row.state === 'pending')
            .map((row) => ({...row, expires_at: fixture.setupExpiresAt ?? row.expires_at})),
        };
      const row = {
        id: 'a' + 'c'.repeat(24),
        application: body.application as Record<string, unknown>,
        administrator_id: Number(body.administrator_id),
        expires_at: Math.floor(Date.now() / 1000) + 600,
        state: 'pending',
        observation: null,
      };
      setups.push(row);
      return row;
    }
    if (path.startsWith('/api/applications/setups/') && body) {
      const row = setups.find((row) => row.id === path.split('/')[4]);
      if (!row) throw Error('missing setup');
      row.observation = {container: 'c'.repeat(64), running: body.action === 'begin', ready: body.action === 'begin'};
      if (body.action === 'complete') row.state = 'complete';
      if (body.action === 'abort') row.state = 'aborted';
      return {
        setup: {...row, expires_at: fixture.setupExpiresAt ?? row.expires_at},
        result: {
          operation: row.id,
          observation: row.observation,
          ...(body.action === 'begin'
            ? {setup_url: String(row.application.origin) + '/setup#operation=' + row.id + '&token=' + 'd'.repeat(64)}
            : {}),
        },
      };
    }

    if (path.endsWith('/snapshots')) {
      if (body && !snapshots.some((row) => row.id === body.id))
        snapshots.unshift({id: body.id as string, created_at: 1792713601, application: {...record.application}});
      return {snapshots: body ? snapshots.filter((row) => row.id === body.id) : snapshots, complete: true};
    }
    if (path.endsWith('/restores') && body) {
      record.restore ??= {
        id: body.id as string,
        snapshot: body.snapshot as string,
        previous: {...record.application},
        next: {...record.application, revision: record.application.revision + 1, data: 'a' + '9'.repeat(24)},
        expected_container: body.expected_container as string,
        state: 'pending',
      };
      if (fixture.failRestore) {
        fixture.failRestore = false;
        record.observation = null;
        throw Error('lost reply');
      }
      if (body.action !== 'abort') record.application = record.restore.next;
      delete record.restore;
      record.observation = {
        id: record.application.id,
        revision: record.application.revision,
        container: 'b'.repeat(64),
        image: record.application.image,
        origin: record.application.origin,
        running: false,
        ready: false,
      };
      return record;
    }
    if (!body) return {applications: [record], complete: true};
    if (record.observation) {
      record.observation.running = body.action === 'start';
      record.observation.ready = record.observation.running;
    }
    return record;
  };
  document.querySelector('main')?.append(element);
  window.applicationFixture = fixture;
  await element.updateComplete;
};

window.createApplicationDeliveryFixture = async (update = false) => {
  await window.createApplicationFixture(true);
  const fixture = window.applicationFixture,
    element = fixture.element;
  const current = initial();
  current.operator = true;
  current.project = {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'};
  current.environment = 'company';
  const artifact = {
    name: 'Example',
    origin: current.application.origin,
    image: 'sha256:' + '9'.repeat(64),
    environment: 'company',
  };
  const delivery = {delivery_id: 'b'.repeat(32), source_commit: 'c'.repeat(40), environment: 'company'};
  const builds: Record<string, unknown>[] = [];
  let setup: Record<string, unknown> | undefined;
  element.request = async (path, body) => {
    fixture.calls.push({path, ...(body ? {body} : {})});
    if (path === '/api/applications') return {applications: update ? [current] : [], complete: true, can_setup: true};
    if (path.startsWith('/api/repositories/7/application-builds')) {
      if (!body) return {builds, administrator: {id: '1', name: 'Maya'}};
      if (path.endsWith('/setup')) {
        setup ??= {
          id: 'a' + 'c'.repeat(24),
          application: {id: 'a' + 'd'.repeat(24), name: 'Example', origin: artifact.origin},
          administrator_id: 1,
          expires_at: Math.floor(Date.now() / 1000) + 600,
          state: 'pending',
          observation: null,
        };
        return setup;
      }
      const environment = body.environment;
      if (environment !== 'company' && environment !== 'preview') throw Error('missing environment');
      const build = {
        id: (environment === 'company' ? 'e' : 'f').repeat(32),
        artifact: {
          ...artifact,
          environment,
          origin: environment === 'company' ? artifact.origin : 'https://preview.example.test',
        },
        delivery: {...delivery, environment},
      };
      builds.push(build);
      return build;
    }
    if (path === '/api/applications/setups') return {setups: setup ? [setup] : []};
    if (path.startsWith('/api/applications/setups/') && body) {
      if (!setup) throw Error('missing setup');
      setup.observation = {container: 'c'.repeat(64), running: true, ready: true};
      return {
        setup,
        result: {
          operation: setup.id,
          setup_url: artifact.origin + '/setup#operation=' + setup.id + '&token=' + 'd'.repeat(64),
        },
      };
    }
    if (path.endsWith('/actions') && body) {
      if (body.action === 'replace') {
        checkBuild(body.replacement_build);
        current.application = {...current.application, revision: 2, image: artifact.image};
        current.observation = {
          ...current.observation!,
          revision: 2,
          image: artifact.image,
          container: 'b'.repeat(64),
          running: false,
          ready: false,
        };
      }
      return current;
    }
    throw Error('unexpected delivery request');
  };
  element.repository = '7';
  element.requestedDelivery = delivery.delivery_id;
  element.expanded = true;
  await element.updateComplete;
  function checkBuild(value: unknown) {
    if (value !== 'e'.repeat(32)) throw Error('wrong build');
  }
};

// A real SodaSpaces page mount with a synthetic host transport, used to lock
// the delivery-version retention path: the production soda-application-open /
// application-delivery-cleared handlers, the localStorage round-trip and the
// .requestedDelivery binding, across a real browser reload. The transport only
// serves one verified company record and never mutates product state.
window.createWorkspaceDeliveryFixture = async () => {
  const deliveryId = 'b'.repeat(32);
  const record = initial();
  record.operator = false;
  record.project = {repository: '7', name: 'northstar/equipment', path: '/northstar/equipment'};
  record.observation = {...record.observation!, bundle_file_sha256: 'b'.repeat(64)};
  record.delivery = {
    environment: 'company',
    state: 'verified',
    intent: 'intent-current',
    revision: 'revision-current',
    delivery_id: deliveryId,
    run: 'run-current',
    source_commit: 'c'.repeat(40),
    bundle_file_sha256: 'b'.repeat(64),
  };
  const fixture: WorkspaceDeliveryFixture = {
    calls: [],
    deliveryId,
    present: () => {},
  };
  const transport: PreparedExtensionMount = {
    root: document.querySelector('main') as HTMLElement,
    generation: 'delivery-fixture-generation',
    assetBase: '/assets/',
    request: async (relativePath, init = {}) => {
      const body = init.body ? (JSON.parse(String(init.body)) as Record<string, unknown>) : undefined;
      fixture.calls.push({path: relativePath, ...(body ? {body} : {})});
      if (!relativePath.startsWith('applications')) throw Error('unexpected delivery fixture request: ' + relativePath);
      if (!body) return Response.json({applications: [record], complete: true, can_setup: false});
      return Response.json(record);
    },
    websocket: () => {
      throw Error('no sockets in delivery fixture');
    },
    dispose: () => {},
  };
  const mount = mountSodaspaces(document.querySelector('main') as HTMLElement, {
    kind: 'page',
    repositoryId: '7',
    transport,
  });
  fixture.present = (kind, repositoryId) => mount.present(kind, repositoryId);
  const workspace = document.querySelector('soda-spaces') as unknown as {
    actor: {id: string; login: string};
    spaces: unknown[];
  };
  workspace.actor = {id: '1', login: 'alice'};
  workspace.spaces = [
    {
      environment: {id: 'e' + '1'.repeat(24), repository_id: '7', repository: 'northstar/equipment'},
      terminals: [],
      factory_runs: [],
    },
  ];
  window.workspaceDeliveryFixture = fixture;
  await mount.ready;
};
