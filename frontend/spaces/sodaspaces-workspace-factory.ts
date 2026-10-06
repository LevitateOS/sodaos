import type {TemplateResult} from 'lit';
import {factoryRunText} from './sodaspaces-api.js';
import type {FactoryRun, Space} from './sodaspaces-api.js';
import {mountFactoryWatch} from './sodaspaces-factory.js';
import type {FactoryWatchContext} from './sodaspaces-factory.js';
import {renderFactoryRuns} from './sodaspaces-workspace-view.js';
import type {FactoryWatch, WorkspaceContext} from './sodaspaces-workspace-types.js';

export const factoryWatchLimit = 8;

export interface FactoryAdmission {
  stale: boolean;
  available: boolean;
}

export function factoryWatching(watches: FactoryWatch[], runId: string) {
  return watches.some((watch) => watch.run === runId);
}

export function factoryRowDisabled(
  admission: FactoryAdmission,
  space: Space,
  run: FactoryRun,
  watches: FactoryWatch[]
) {
  return (
    admission.stale ||
    !admission.available ||
    !space.execution_allowed ||
    space.authority_unavailable ||
    (watches.length >= factoryWatchLimit && !factoryWatching(watches, run.id))
  );
}

export interface FactoryWatchMutation {
  readWatches: () => FactoryWatch[];
  isStale: () => boolean;
  pushWatch: (watch: FactoryWatch) => void;
  replaceWatches: (watches: FactoryWatch[]) => void;
  updated: () => void;
}

export function toggleFactoryWatch(mutation: FactoryWatchMutation, space: Space, run: FactoryRun) {
  if (factoryWatching(mutation.readWatches(), run.id)) unwatchRun(mutation, run.id);
  else watchRun(mutation, space, run);
}

export function watchRun(mutation: FactoryWatchMutation, space: Space, run: FactoryRun) {
  if (
    mutation.isStale() ||
    factoryWatching(mutation.readWatches(), run.id) ||
    mutation.readWatches().length >= factoryWatchLimit ||
    !space.execution_allowed ||
    space.authority_unavailable
  )
    return;
  mutation.pushWatch({
    run: run.id,
    environmentId: space.environment.id,
    repositoryId: space.environment.repository_id,
    role: run.role,
    ...(run.issue === undefined ? {} : {issue: run.issue}),
    ...(run.attempt === undefined ? {} : {attempt: run.attempt}),
    hidden: false,
  });
  mutation.updated();
}

export function unwatchRun(mutation: FactoryWatchMutation, runId: string) {
  const watches = mutation.readWatches();
  const watch = watches.find((entry) => entry.run === runId);
  mutation.replaceWatches(watches.filter((entry) => entry.run !== runId));
  watch?.view?.dispose();
  mutation.updated();
}

export interface FactorySectionInput extends FactoryWatchMutation {
  available: boolean;
  space: Space;
  query: string;
}

export function factorySection(input: FactorySectionInput): TemplateResult | string {
  const current = input.readWatches();
  const rows = input.space.factory_runs.filter(
    (run) => !input.query || factoryRunText(run).toLocaleLowerCase().includes(input.query)
  );
  const watches = current.filter((watch) => watch.environmentId === input.space.environment.id);
  if (!rows.length && !watches.length) return '';
  return renderFactoryRuns(
    rows.map((run) => ({
      key: run.id,
      text: factoryRunText(run),
      watching: factoryWatching(current, run.id),
      disabled: factoryRowDisabled({stale: input.isStale(), available: input.available}, input.space, run, current),
      toggle: () => toggleFactoryWatch(input, input.space, run),
    })),
    watches.map((watch) => ({
      run: watch.run,
      hidden: watch.hidden,
      close: () => unwatchRun(input, watch.run),
    })),
    current.length >= factoryWatchLimit
  );
}

export interface FactoryWatchContextInput {
  binding: WorkspaceContext | undefined;
  actor: {id: string; login: string} | undefined;
  watch: FactoryWatch;
  space: Space;
  projectName: (space: Space) => string;
}

export function factoryWatchContext(input: FactoryWatchContextInput): FactoryWatchContext | undefined {
  if (!input.binding || !input.actor) return undefined;
  return {
    expectedUserId: input.actor.id,
    transport: input.binding.transport,
    repositoryId: input.watch.repositoryId,
    environmentId: input.watch.environmentId,
    projectName: input.projectName(input.space),
    runId: input.watch.run,
    role: input.watch.role,
    ...(input.watch.issue === undefined ? {} : {issue: input.watch.issue}),
    ...(input.watch.attempt === undefined ? {} : {attempt: input.watch.attempt}),
  };
}

export interface FactoryCommandInput {
  isStale: () => boolean;
  isDisposed: () => boolean;
  isSurfaceVisible: () => boolean;
  ownerConcealed: () => boolean;
  requestUpdate: () => void;
}

export function onFactoryCommand(input: FactoryCommandInput, watch: FactoryWatch, event: Event) {
  if (event instanceof CustomEvent && event.detail === 'hide' && !input.isStale() && !input.isDisposed()) {
    watch.hidden = !watch.hidden;
    watch.view?.setVisible(input.isSurfaceVisible() && !watch.hidden && !input.ownerConcealed());
    input.requestUpdate();
  }
}

export interface FactoryDisplayInput extends FactoryCommandInput {
  binding: WorkspaceContext | undefined;
  actor: {id: string; login: string} | undefined;
  spaces: Space[];
  watches: FactoryWatch[];
  projectName: (space: Space) => string;
  findOwner: (runId: string) => HTMLElement | null;
  invalidate: () => void;
}

export function displayFactoryWatch(input: FactoryDisplayInput, watch: FactoryWatch) {
  const space = input.spaces.find((entry) => entry.environment.id === watch.environmentId);
  const owner = input.findOwner(watch.run);
  if (!space || !owner?.isConnected) return;
  if (!watch.view) {
    const context = factoryWatchContext({
      binding: input.binding,
      actor: input.actor,
      watch,
      space,
      projectName: input.projectName,
    });
    if (!context) return;
    owner.addEventListener('soda-factory-command', (event) => onFactoryCommand(input, watch, event));
    owner.addEventListener('soda-factory-authority-lost', () => {
      if (!input.isDisposed()) input.invalidate();
    });
    watch.view = mountFactoryWatch(owner, context);
    watch.view.watch();
  }
  watch.view.setVisible(input.isSurfaceVisible() && !watch.hidden && !input.ownerConcealed());
}

export function displayFactoryWatches(input: FactoryDisplayInput) {
  for (const watch of input.watches) displayFactoryWatch(input, watch);
}
