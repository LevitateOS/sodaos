import {mountFactoryWatch as mountSharedFactoryWatch} from '@factory/product/frontend/factory/watch/watch.js';
import type {Renderer} from './sodaspaces-terminal.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {ResumeValidation} from './fountain-browser-contract.js';
import {fountainRepository} from './soda-factory-feed-host.js';
export interface FountainFactoryWatchContext {
  /** Native viewer and Project custody, owned by the Fountain host. */
  expectedUserId: string;
  transport: PreparedExtensionMount;
  nativeRepositoryId: string;
  environmentId: string;
  projectName?: string;
  runId: string;
  role: string;
  issue?: string;
  attempt?: string;
}

const renderer = async (): Promise<Renderer> => {
  const [{Terminal}, {FitAddon}] = await Promise.all([
    import('./soda-terminal/xterm.mjs'),
    import('./soda-terminal/addon-fit.mjs'),
  ]);
  return {
    Terminal,
    FitAddon,
  };
};

function optionalMatches(value: string | undefined, pattern: RegExp) {
  return value === undefined || pattern.test(value);
}
function admitWatchIds(
  expectedUserId: string,
  nativeRepositoryId: string,
  environmentId: string,
  runId: string,
  role: string
) {
  if (
    !/^[1-9][0-9]{0,18}$/.test(expectedUserId) ||
    !/^[1-9][0-9]{0,18}$/.test(nativeRepositoryId) ||
    !/^p[0-9a-f]{24}$/.test(environmentId) ||
    !/^[0-9a-f]{32}$/.test(runId) ||
    !/^[a-z][a-z0-9-]{0,63}$/.test(role)
  )
    throw Error('Invalid factory watch context');
}
function admitWatchContext(root: HTMLElement, context: FountainFactoryWatchContext) {
  const {expectedUserId, nativeRepositoryId, environmentId, runId, role} = context;
  admitWatchIds(expectedUserId, nativeRepositoryId, environmentId, runId, role);
  if (
    root.ownerDocument !== document ||
    !optionalMatches(context.issue, /^[1-9][0-9]{0,18}$/) ||
    !optionalMatches(context.attempt, /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/)
  )
    throw Error('Invalid factory watch context');
}
export function mountFactoryWatch(
  root: HTMLElement,
  supplied: FountainFactoryWatchContext,
  loadRenderer: () => Promise<Renderer> = renderer
) {
  admitWatchContext(root, supplied);
  const context = {...supplied};
  const view = mountSharedFactoryWatch(
    root,
    {
      transport: context.transport,
      runId: context.runId,
      role: context.role,
      ...(context.issue === undefined ? {} : {issue: context.issue}),
      ...(context.attempt === undefined ? {} : {attempt: context.attempt}),
      source: {
        repository: async (signal) =>
          (await fountainRepository(context.transport, context.nativeRepositoryId, signal)).repository,
        connect: () => context.transport.websocket('factory/runs/' + context.runId + '/output'),
        attachment: (cursor, generation) => ({
          run_id: context.runId,
          repository_id: context.nativeRepositoryId,
          session_generation: generation,
          cursor,
        }),
      },
    },
    loadRenderer
  );
  return {
    setVisible: view.setVisible,
    get connected() {
      return view.connected;
    },
    watch: view.watch,
    get started() {
      return view.started;
    },
    get ready() {
      return view.ready;
    },
    stop: view.stop,
    suspend: view.suspend,
    resume: (validation?: ResumeValidation) =>
      view.resume(validation === undefined ? undefined : {generation: validation.sessionGeneration}),
    retire: view.retire,
    invalidate: view.invalidate,
    dispose: view.dispose,
  };
}
