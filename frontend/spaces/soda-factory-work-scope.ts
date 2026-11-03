// Fountain workspace composition; shared presentation never imports mount context.
import type {PreparedExtensionMount} from './soda-extension.js';
import type {WorkDetailContext} from '@factory/product/frontend/factory/work/detail.js';
import type {
  DirectionRevisionContext,
  WorkOverviewContext,
  WorkPlanContext,
} from '@factory/product/frontend/factory/work/progress.js';
import {fountainAcceptanceLinks, fountainWorkLinks} from './soda-factory-links.js';
import type {OverviewScope} from '@factory/product/frontend/factory/work/progress-view.js';
import {upfrontCapabilities} from '@factory/product/frontend/factory/work/work-view.js';
import {fountainCommandScope} from './soda-factory-feed-host.js';

// planContextFromScope builds the plan context over every scoped
// issue. An empty scope yields an empty plan, never a guess.
export function planContextFromScope(
  transport: PreparedExtensionMount,
  repository: string,
  repositoryPath: string,
  scope: OverviewScope
): WorkPlanContext {
  return {
    transport,
    repository,
    links: fountainWorkLinks(repositoryPath),
    issues: scope.issues.map((entry) => entry.issue),
  };
}

export interface DetailScopeRuns {
  readonly runIDs: string[];
  readonly runID?: string;
  readonly projectID?: string;
  readonly environmentID?: string;
}

// detailContextFromScope builds one issue's detail context. Issue
// selection stays with the composer; capabilities come from the
// scope when the overview read that issue, else stay permissive
// until the detail's own read lands.
export function detailContextFromScope(
  transport: PreparedExtensionMount,
  repository: string,
  repositoryPath: string,
  scope: OverviewScope,
  issue: string,
  runs: DetailScopeRuns
): WorkDetailContext {
  const known = scope.issues.find((entry) => entry.issue === issue)?.capabilities;
  return fountainDetailContext({
    transport,
    repository,
    repositoryPath,
    issue,
    runIDs: runs.runIDs,
    capabilities: upfrontCapabilities(known, runs.environmentID !== undefined),
    ...(runs.runID === undefined ? {} : {runID: runs.runID}),
    ...(runs.projectID === undefined ? {} : {projectID: runs.projectID}),
    ...(runs.environmentID === undefined ? {} : {environmentID: runs.environmentID}),
  });
}

// revisionContextFromScope builds the direction-revision context.
// Without an intent the revision has no read key: undefined means
// do not mount.
export function revisionContextFromScope(
  transport: PreparedExtensionMount,
  scope: OverviewScope
): DirectionRevisionContext | undefined {
  if (scope.intentID === undefined) return undefined;
  return {transport, intentID: scope.intentID};
}

export interface FountainWorkOverviewContext extends Omit<WorkOverviewContext, 'links'> {
  readonly transport: PreparedExtensionMount;
  readonly repositoryPath: string;
}
export interface FountainWorkPlanContext extends Omit<WorkPlanContext, 'links'> {
  readonly transport: PreparedExtensionMount;
  readonly repositoryPath: string;
}
export function fountainOverviewContext(context: FountainWorkOverviewContext): WorkOverviewContext {
  const {repositoryPath, ...resource} = context;
  return {...resource, links: fountainWorkLinks(repositoryPath)};
}
export function fountainPlanContext(context: FountainWorkPlanContext): WorkPlanContext {
  const {repositoryPath, ...resource} = context;
  return {...resource, links: fountainWorkLinks(repositoryPath)};
}

export interface FountainWorkDetailContext extends Omit<
  WorkDetailContext,
  'links' | 'acceptanceLinks' | 'detailLinks'
> {
  readonly transport: PreparedExtensionMount;
  readonly repositoryPath: string;
}
export function fountainDetailContext(context: FountainWorkDetailContext): WorkDetailContext {
  const {repositoryPath, ...resource} = context;
  return {
    ...resource,
    commandScope: (signal) => fountainCommandScope(context.transport, signal),
    links: fountainWorkLinks(repositoryPath),
    acceptanceLinks: fountainAcceptanceLinks(repositoryPath),
    detailLinks: (detail) => fountainWorkLinks(repositoryPath, detail.links),
  };
}
