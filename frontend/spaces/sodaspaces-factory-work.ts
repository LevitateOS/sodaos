import {SodaFactoryWorkDetail} from '@factory/product/frontend/factory/work/detail.js';
import {
  SodaFactoryWorkOverview,
  SodaFactoryWorkPlan,
  SodaFactoryDirectionRevision,
} from '@factory/product/frontend/factory/work/progress.js';
import type {DirectionRevisionContext} from '@factory/product/frontend/factory/work/progress.js';
import {fountainDetailContext, fountainOverviewContext, fountainPlanContext} from './soda-factory-work-scope.js';
import type {
  FountainWorkDetailContext,
  FountainWorkOverviewContext,
  FountainWorkPlanContext,
} from './soda-factory-work-scope.js';
import {id, projectId} from '@factory/product/frontend/factory/api/models.js';
import {decisionID} from '@factory/product/frontend/factory/api/request-response.js';
import {factoryRunID} from '@factory/product/frontend/factory/api/run-response.js';
import {nativePath} from '@factory/product/frontend/factory/shared/format.js';
function admitWorkRoot(root: HTMLElement) {
  if (root.ownerDocument !== document) throw Error('Invalid work view root');
}
function admitWorkRepository(repository: string) {
  if (!id(repository)) throw Error('Invalid work repository');
}
function admitWorkIssue(issue: string) {
  if (!id(issue)) throw Error('Invalid work issue');
}
function admitWorkPath(path: string) {
  if (!nativePath(path)) throw Error('Invalid work repository path');
}
function admitWorkIntent(intentID: string | undefined) {
  if (intentID !== undefined && !decisionID(intentID)) throw Error('Invalid work intent');
}
function admitWorkRuns(runIDs: string[], runID: string | undefined) {
  if (runIDs.length > 64 || runIDs.some((run) => !factoryRunID(run))) throw Error('Invalid work runs');
  if (runID !== undefined && !factoryRunID(runID)) throw Error('Invalid work run');
}
function admitWorkProject(projectID: string | undefined, environmentID: string | undefined) {
  if (projectID !== undefined && !projectId(projectID)) throw Error('Invalid work project');
  if (environmentID !== undefined && !projectId(environmentID)) throw Error('Invalid work environment');
}

export function mountFactoryWorkOverview(root: HTMLElement, context: FountainWorkOverviewContext) {
  admitWorkRoot(root);
  admitWorkRepository(context.repository);
  admitWorkPath(context.repositoryPath);
  admitWorkIntent(context.intentID);
  const view = new SodaFactoryWorkOverview();
  view.context = fountainOverviewContext(context);
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}

export function mountFactoryWorkDetail(root: HTMLElement, context: FountainWorkDetailContext) {
  admitWorkRoot(root);
  admitWorkRepository(context.repository);
  admitWorkPath(context.repositoryPath);
  admitWorkIssue(context.issue);
  admitWorkRuns(context.runIDs, context.runID);
  admitWorkProject(context.projectID, context.environmentID);
  const view = new SodaFactoryWorkDetail();
  view.context = fountainDetailContext(context);
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}

export function mountFactoryWorkPlan(root: HTMLElement, context: FountainWorkPlanContext) {
  admitWorkRoot(root);
  admitWorkRepository(context.repository);
  admitWorkPath(context.repositoryPath);
  if (context.issues.length > 64 || context.issues.some((issue) => !id(issue))) throw Error('Invalid plan issues');
  const view = new SodaFactoryWorkPlan();
  view.context = fountainPlanContext(context);
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}

export function mountFactoryDirectionRevision(root: HTMLElement, context: DirectionRevisionContext) {
  admitWorkRoot(root);
  if (!decisionID(context.intentID)) throw Error('Invalid work intent');
  const view = new SodaFactoryDirectionRevision();
  view.context = context;
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}
