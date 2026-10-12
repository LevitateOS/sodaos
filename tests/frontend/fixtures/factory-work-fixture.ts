import {
  mountFactoryDirectionRevision,
  mountFactoryOptionalEffects,
  mountFactoryWorkDetail,
  mountFactoryWorkOverview,
  mountFactoryWorkPlan,
} from '../../../frontend/spaces/sodaspaces-factory-work.js';
import type {PreparedExtensionMount} from '../../../frontend/spaces/soda-extension.js';

// Browser stub for the Factory work-progress elements (V01–V07).
// Routes match exact transport paths; replies with echoCommand inject
// the request's command_id so receipts bind the issued command.

export interface WorkCall {
  readonly method: string;
  readonly path: string;
  readonly body: unknown;
}
export interface WorkRoute {
  readonly method: string;
  readonly path: string;
  readonly action?: string;
  readonly status: number;
  readonly json: unknown;
  readonly echoCommand?: boolean;
}

export const INTENT = 'd' + 'e'.repeat(24);
export const REVISION = 'd' + 'c'.repeat(24);
export const TASK_A = 'd' + 'a'.repeat(24);
export const TASK_B = 'd' + 'b'.repeat(24);
export const RUN = 'd'.repeat(32);
export const PROJECT = 'p0123456789abcdef01234567';
export const ENVIRONMENT = 'p' + 'f'.repeat(24);

let routes: WorkRoute[] = [];
const calls: WorkCall[] = [];

function matchBody(route: WorkRoute, body: unknown): boolean {
  if (route.action === undefined) return true;
  if (typeof body !== 'object' || body === null || !('action' in body)) return false;
  return (body as Record<string, unknown>).action === route.action;
}

function withCommand(json: unknown, body: unknown): unknown {
  if (typeof body !== 'object' || body === null || !('command_id' in body)) return json;
  if (typeof json !== 'object' || json === null) return json;
  return {...json, command_id: (body as Record<string, unknown>).command_id};
}

function stubTransport(): PreparedExtensionMount {
  return {
    request: (async (path: string, init: RequestInit = {}) => {
      const method = init.method ?? 'GET';
      let body: unknown;
      if (typeof init.body === 'string' && init.body !== '') {
        try {
          body = JSON.parse(init.body);
        } catch {
          body = init.body;
        }
      }
      calls.push({method, path, body});
      const hit = routes.find((route) => route.method === method && route.path === path && matchBody(route, body));
      if (!hit) return Response.json({error: {code: 'not_found', message: 'No stub reply.'}}, {status: 404});
      const json = hit.echoCommand ? withCommand(hit.json, body) : hit.json;
      return Response.json(json, {status: hit.status});
    }) as PreparedExtensionMount['request'],
  } as unknown as PreparedExtensionMount;
}

export type WorkView = 'overview' | 'detail' | 'plan' | 'direction' | 'effects';

const allCapabilities = {pause: true, resume: true, stop: true, retry: true, takeover: true, project_stop: true};

function mount(view: WorkView): void {
  const root = document.createElement('section');
  root.setAttribute('data-view', view);
  document.querySelector('main')?.append(root);
  const transport = stubTransport();
  if (view === 'overview')
    mountFactoryWorkOverview(root, {transport, repository: '7', repositoryPath: '/soda-tester/demo', intentID: INTENT});
  else if (view === 'detail')
    mountFactoryWorkDetail(root, {
      transport,
      repository: '7',
      repositoryPath: '/soda-tester/demo',
      issue: '12',
      runIDs: [],
      capabilities: {...allCapabilities},
      runID: RUN,
      projectID: PROJECT,
      environmentID: ENVIRONMENT,
    });
  else if (view === 'plan')
    mountFactoryWorkPlan(root, {transport, repository: '7', repositoryPath: '/soda-tester/demo', issues: ['12']});
  else if (view === 'direction') mountFactoryDirectionRevision(root, {transport, intentID: INTENT});
  else mountFactoryOptionalEffects(root, {transport, repository: '7', repositoryPath: '/soda-tester/demo', issue: '12'});
}

declare global {
  interface Window {
    factoryWork: {
      mount: (view: WorkView) => void;
      setRoutes: (next: WorkRoute[]) => void;
      calls: () => WorkCall[];
    };
  }
}

window.factoryWork = {
  mount,
  setRoutes: (next: WorkRoute[]) => {
    routes = next;
    calls.length = 0;
  },
  calls: () => [...calls],
};
