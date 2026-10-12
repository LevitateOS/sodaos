import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {id, object} from './sodaspaces-api.js';
import {renderWorkspaceIntro} from './sodaspaces-workspace-view.js';
import './sodaspaces-intent.js';
import type {SodaIntent} from './sodaspaces-intent.js';
import {intentList} from './sodaspaces-intent-response.js';
import type {SavedRequest} from './sodaspaces-intent-response.js';
import './sodaspaces-factory-work.js';
import type {WorkDetailContext, WorkOverviewContext, WorkPlanContext} from './sodaspaces-factory-work.js';
import {
  detailContextFromScope,
  planContextFromScope,
  revisionContextFromScope,
} from './sodaspaces-factory-progress-view.js';
import type {DetailScopeRuns, OverviewScope, OverviewScopeIssue} from './sodaspaces-factory-progress-view.js';
import type {FactoryRun} from './sodaspaces-factory-response.js';
import type {WorkCapabilities} from './sodaspaces-factory-work-response.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import type {PreparedExtensionMount} from './soda-extension.js';

/** Native opaque request id shape shared with the host alias routes. */
export const requestIDPattern = /^d[0-9a-f]{24}$/;

/** Strip the admitted installation prefix at an exact segment boundary; undefined on mismatch. */
function stripRoutePrefix(pathname: string, forgejoPrefix: string): string | undefined {
  if (forgejoPrefix === '') return pathname;
  if (!pathname.startsWith(forgejoPrefix)) return undefined;
  const rest = pathname.slice(forgejoPrefix.length);
  if (!rest.startsWith('/')) return undefined;
  return rest;
}

/** Read the resume request id from a workspace route path; '' means fresh entry. */
export function requestIdFromLocation(pathname: string, forgejoPrefix: string): string {
  const rest = stripRoutePrefix(pathname, forgejoPrefix);
  if (rest === undefined) return '';
  const segments = rest.split('/').filter((segment) => segment !== '');
  if (segments.length !== 3) return '';
  if (segments[0] !== 'workspace' || segments[1] !== 'requests') return '';
  const id = segments[2] || '';
  return requestIDPattern.test(id) ? id : '';
}

/** Canonical request route under the admitted installation prefix. */
export function requestRouteFor(forgejoPrefix: string, intent: string): string {
  return `${forgejoPrefix}/workspace/requests/${intent}`;
}

/** Fresh-entry route under the admitted installation prefix. */
export function newRouteFor(forgejoPrefix: string): string {
  return `${forgejoPrefix}/workspace/new`;
}

/** Exact fresh-entry route match under the admitted installation prefix. */
export function isNewRouteLocation(pathname: string, forgejoPrefix: string): boolean {
  return stripRoutePrefix(pathname, forgejoPrefix) === '/workspace/new';
}

/** Intent identity carried by intent-saved/intent-resumed; '' when absent. */
export function intentIdentity(event: Event): string {
  const detail = (event as CustomEvent).detail;
  if (!detail || typeof detail !== 'object' || Array.isArray(detail)) return '';
  const intent = (detail as Record<string, unknown>).intent;
  return typeof intent === 'string' ? intent : '';
}

/** Saved requests carried by intent-inventory; malformed payloads yield []. */
export function inventoryIntents(event: Event): SavedRequest[] {
  try {
    return intentList((event as CustomEvent).detail).intents;
  } catch {
    return [];
  }
}

/** Authorized counts carried by intent-reviewed; malformed payloads yield undefined. */
export function reviewedAuthorization(event: Event): {ideas: number; tasks: number} | undefined {
  const detail = (event as CustomEvent).detail;
  if (!detail || typeof detail !== 'object' || Array.isArray(detail)) return undefined;
  const record = detail as Record<string, unknown>;
  if (!Number.isInteger(record.authorizedIdeas) || (record.authorizedIdeas as number) < 0) return undefined;
  if (!Number.isInteger(record.tasks) || (record.tasks as number) < 0) return undefined;
  return {ideas: record.authorizedIdeas as number, tasks: record.tasks as number};
}

/** Reviewed intent identity carried by intent-reviewed; malformed payloads yield undefined. */
export function reviewedIntent(event: Event): string | undefined {
  const detail = (event as CustomEvent).detail;
  if (!detail || typeof detail !== 'object' || Array.isArray(detail)) return undefined;
  const intent = (detail as Record<string, unknown>).intent;
  return typeof intent === 'string' && requestIDPattern.test(intent) ? intent : undefined;
}

/** Committed resume identity carried by intent-resumed; undefined unless accepted with a bound repository. */
export function resumedCommittedRequest(event: Event): {intent: string; repository: string} | undefined {
  const detail = (event as CustomEvent).detail;
  if (!detail || typeof detail !== 'object' || Array.isArray(detail)) return undefined;
  const record = detail as Record<string, unknown>;
  const intent = record.intent;
  if (typeof intent !== 'string' || !requestIDPattern.test(intent)) return undefined;
  if (record.state !== 'accepted') return undefined;
  const repository = record.repository;
  if (typeof repository !== 'string' || !id(repository)) return undefined;
  return {intent, repository};
}

function resumedRepositoryName(part: unknown): string | undefined {
  if (typeof part !== 'string' || part === '' || part === '.' || part === '..') return undefined;
  if (/[/\\\p{Cc}\p{Cf}]/u.test(part) || new TextEncoder().encode(part).length > 255) return undefined;
  return part;
}

/** Exact visible owner/name for one repository id; undefined when inaccessible or malformed. */
async function readResumedRepository(
  api: SetupActions['api'],
  repository: string
): Promise<{owner: string; name: string} | undefined> {
  try {
    const data = object(await api('/api/environments?repository_id=' + repository));
    const repo = object((data as Record<string, unknown>).repository);
    if (repo.id !== repository) return undefined;
    const owner = resumedRepositoryName(repo.owner);
    const name = resumedRepositoryName(repo.name);
    if (!owner || !name) return undefined;
    return {owner, name};
  } catch {
    return undefined;
  }
}

function resumeCommittedAdmitted(reading: SetupReading, resumed: {intent: string; repository: string}): boolean {
  if (reading.stale || !reading.canRestore || !reading.activeSurface || !reading.transport) return false;
  if (resumed.intent !== reading.requestId) return false;
  if (
    reading.setupCommitted &&
    reading.reviewedIntent === resumed.intent &&
    reading.repositoryChoice === resumed.repository
  )
    return false;
  return true;
}

/**
 * Restore committed overview context for an accepted resume. One
 * authorized environments GET derives exact visible owner/name; the
 * same existing overview mounts with the original identity. Never
 * forces the creation picker; an inaccessible repository reports
 * honest unavailability and leaves committed state untouched.
 */
export async function resumeCommittedOverview(
  event: Event,
  read: () => SetupReading,
  actions: SetupActions
): Promise<void> {
  const resumed = resumedCommittedRequest(event);
  if (!resumed || !resumeCommittedAdmitted(read(), resumed)) return;
  const metadata = await readResumedRepository(actions.api, resumed.repository);
  if (actions.isStale() || read().requestId !== resumed.intent) return;
  if (!metadata) {
    actions.notifyFactory(
      `Saved repository ${resumed.repository} is unavailable; the request overview cannot be shown.`
    );
    return;
  }
  actions.setReviewedIntent(resumed.intent);
  actions.setRepositoryChoice(resumed.repository);
  actions.setRepositoryResult({
    items: [{id: resumed.repository, owner: metadata.owner, name: metadata.name, canCreate: false, project: null}],
    page: 1,
    nextCursor: '',
  });
  actions.setSetupCommitted(true);
  actions.setSetup('intent');
  actions.notifyFactory('');
}
import {renderRepositoryPicker} from './sodaspaces-repository-picker-view.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import type {RepositoryChoices} from './sodaspaces-repository-response.js';
import type {Space} from './sodaspaces-inventory-response.js';

export function setupIntroKind(busy: boolean) {
  return busy ? 'loading' : 'unavailable';
}
export function setupIntroHeading(busy: boolean, reconnectRequired: boolean) {
  if (busy) return 'Loading projects…';
  if (reconnectRequired) return 'Reconnect to Forgejo';
  return 'Could not load projects';
}
export function setupIntroDescription(busy: boolean, reconnectRequired: boolean) {
  if (busy) return html`Checking the projects you can access.`;
  if (reconnectRequired) return html`Sign in again to restore your Forgejo access.`;
  return html`We couldn’t load your project list. Try again.`;
}
export function setupIntroAction(
  busy: boolean,
  reconnectRequired: boolean,
  stale: boolean,
  connectURL: string,
  blocked: boolean,
  onRefresh: () => void
) {
  if (busy) return html``;
  if (reconnectRequired) return html`<a class="ui primary button" href=${connectURL}>Reconnect to Forgejo</a>`;
  if (stale)
    return html`<button class="ui primary button" @click=${() => window.location.reload()}>Reload Spaces</button>`;
  return html`<button class="ui primary button" ?disabled=${blocked} @click=${onRefresh}>Retry projects</button>`;
}
export function setupIntroHelper(busy: boolean) {
  if (busy) return html`This will not create or start anything.`;
  return html`Your existing projects and terminals are not replaced.`;
}
export function searchAdmitted(
  stale: boolean,
  available: boolean,
  setup: 'intent' | 'repositories' | 'configure' | null,
  activeSurface: boolean
) {
  return !stale && available && setup === 'repositories' && activeSurface;
}
export function repositorySearchPath(query: string, cursor: string) {
  const params = new URLSearchParams({q: query});
  if (cursor) params.set('cursor', cursor);
  return '/api/repositories?' + params;
}

export interface SetupReturn {
  project: string;
  view: 'terminal' | 'sessions' | 'project';
  mode: 'standard' | 'journey' | 'settings';
}

export interface SetupReading {
  setup: 'intent' | 'repositories' | 'configure' | null;
  welcomeScreen: boolean;
  busy: boolean;
  reconnectRequired: boolean;
  stale: boolean;
  connectURL: string;
  available: boolean;
  canRestore: boolean;
  activeSurface: boolean;
  repositoryQuery: string;
  repositoryChoice: string;
  repositoryResult: RepositoryChoices | undefined;
  repositoryError: string;
  repositoryBusy: boolean;
  forgejoPrefix: string;
  project: string;
  view: 'terminal' | 'sessions' | 'project';
  managementMode: 'standard' | 'journey' | 'settings';
  setupReturn: SetupReturn | undefined;
  firstSpace: Space | undefined;
  actorId: string;
  requestId: string;
  reviewAuthorization: {ideas: number; tasks: number} | undefined;
  reviewedIntent: string | undefined;
  setupCommitted: boolean;
  transport: PreparedExtensionMount | undefined;
  suggestedRepository: string | undefined;
  overviewScope: OverviewScope | undefined;
  committedSpace: Space | undefined;
  onBegin: () => void;
  onRefresh: () => void;
}

export interface SetupActions {
  setSetup(setup: 'intent' | 'repositories' | 'configure' | null): void;
  setRepositoryQuery(query: string): void;
  setRepositoryChoice(choice: string): void;
  setRepositoryResult(result: RepositoryChoices | undefined): void;
  setRepositoryCursors(cursors: string[]): void;
  setRepositoryError(error: string): void;
  setRepositoryBusy(busy: boolean): void;
  setSetupReturn(target: SetupReturn | undefined): void;
  setReviewAuthorization(auth: {ideas: number; tasks: number} | undefined): void;
  setReviewedIntent(intent: string | undefined): void;
  setSetupCommitted(committed: boolean): void;
  setProject(project: string): void;
  setView(view: 'terminal' | 'sessions' | 'project'): void;
  setManagementMode(mode: 'standard' | 'journey' | 'settings'): void;
  repositoryCursors(): string[];
  currentRepositoryRequest(): AbortController | undefined;
  admitRepositoryRequest(request: AbortController | undefined): void;
  currentRepositoryQuery(): string;
  hasSelectedSpace(): boolean;
  isStale(): boolean;
  isActiveSurface(): boolean;
  selectProject(space: Space): void;
  rememberFocus(): void;
  restoreFocus(): void;
  showManagement(repositoryId: string, mode: 'standard' | 'journey' | 'settings'): void;
  notifyFactory(message: string): void;
  api(path: string, body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown>;
  syncIntentRoute(intent: string | null): void;
  setSavedRequests(intents: SavedRequest[]): void;
  setOverviewScope(scope: OverviewScope | undefined): void;
  requestFactoryWatch(request: FactoryWatchRequest): void;
}

export interface SetupHost {
  updateComplete: Promise<unknown>;
  querySelector<E extends Element>(selectors: string): E | null;
}

/** Committed-request overview context from retained setup state; undefined until every part is exact. */
export function setupOverviewContext(reading: SetupReading): WorkOverviewContext | undefined {
  if (!reading.setupCommitted || !reading.transport || !reading.reviewedIntent) return undefined;
  const choice = reading.repositoryResult?.items.find((item) => item.id === reading.repositoryChoice);
  if (!choice) return undefined;
  const repositoryPath = `${reading.forgejoPrefix}/${choice.owner}/${choice.name}`;
  if (!nativePath(repositoryPath)) return undefined;
  return {transport: reading.transport, repository: choice.id, repositoryPath, intentID: reading.reviewedIntent};
}

const scopeCapabilityKinds = ['watch', 'stop', 'retry', 'takeover', 'pause', 'resume'] as const;

function scopeEntryCapabilities(value: unknown): WorkCapabilities | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined;
  const record = value as Record<string, unknown>;
  for (const kind of scopeCapabilityKinds) {
    const entry = record[kind];
    if (!entry || typeof entry !== 'object' || Array.isArray(entry)) return undefined;
    if (typeof (entry as Record<string, unknown>).available !== 'boolean') return undefined;
  }
  return value as WorkCapabilities;
}

function scopeIssueEntry(value: unknown): OverviewScopeIssue | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined;
  const record = value as Record<string, unknown>;
  if (typeof record.issue !== 'string' || record.issue === '') return undefined;
  const capabilities = record.capabilities === undefined ? undefined : scopeEntryCapabilities(record.capabilities);
  return {issue: record.issue, ...(capabilities === undefined ? {} : {capabilities})};
}

function scopeIssueList(value: unknown): OverviewScopeIssue[] | undefined {
  if (!Array.isArray(value)) return undefined;
  const issues: OverviewScopeIssue[] = [];
  for (const raw of value) {
    const entry = scopeIssueEntry(raw);
    if (entry) issues.push(entry);
  }
  return issues;
}

/** Work scope carried by soda-factory-overview-scope; malformed payloads yield undefined (retire). */
export function overviewScopeDetail(event: Event): OverviewScope | undefined {
  const detail = (event as CustomEvent).detail;
  if (!detail || typeof detail !== 'object' || Array.isArray(detail)) return undefined;
  const record = detail as Record<string, unknown>;
  if (typeof record.repository !== 'string') return undefined;
  if (record.intentID !== undefined && typeof record.intentID !== 'string') return undefined;
  const issues = scopeIssueList(record.issues);
  if (!issues) return undefined;
  return {
    repository: record.repository,
    ...(record.intentID === undefined ? {} : {intentID: record.intentID as string}),
    issues,
  };
}

/** Value equality for scope publications; refresh re-dispatch must not churn the root. */
export function sameScope(a: OverviewScope | undefined, b: OverviewScope | undefined): boolean {
  if (a === b) return true;
  if (!a || !b || a.repository !== b.repository || a.intentID !== b.intentID) return false;
  return JSON.stringify(a.issues) === JSON.stringify(b.issues);
}

export function applyOverviewScope(event: Event, read: () => SetupReading, actions: SetupActions): void {
  const scope = overviewScopeDetail(event);
  if (!sameScope(read().overviewScope, scope)) actions.setOverviewScope(scope);
}

export interface WorkChildScope {
  readonly scope: OverviewScope;
  readonly transport: PreparedExtensionMount;
  readonly repository: string;
  readonly repositoryPath: string;
}

/**
 * Gated child-view feed: a non-retired scope whose keys match the
 * retained committed identity, on the current intent surface only.
 */
export function childScopeFromReading(reading: SetupReading, context: WorkOverviewContext): WorkChildScope | undefined {
  const scope = reading.overviewScope;
  if (!scope || scope.repository === '') return undefined;
  if (reading.setup !== 'intent' || reading.stale) return undefined;
  if (scope.repository !== context.repository) return undefined;
  if (scope.intentID !== undefined && scope.intentID !== context.intentID) return undefined;
  return {scope, transport: context.transport, repository: context.repository, repositoryPath: context.repositoryPath};
}

/**
 * Runs one issue detail may show and act on. runID selects the live
 * (unreconciled) run iff exactly one exists, so stop/retry/takeover
 * never guess among ambiguous rows; projectID is the owning space's
 * environment id, the same project id the backend lists these runs under.
 */
export function detailRunsForIssue(space: Space | undefined, issue: string): DetailScopeRuns {
  const runs = space?.factory_runs.filter((run) => run.issue === issue) ?? [];
  const live = runs.filter((run) => !run.reconciled);
  return {
    runIDs: runs.map((run) => run.id),
    ...(live.length === 1 && live[0] ? {runID: live[0].id} : {}),
    ...(space === undefined ? {} : {projectID: space.environment.id, environmentID: space.environment.id}),
  };
}

/** Watch target carried by soda-factory-watch-request; environment narrows the owning space. */
export interface FactoryWatchRequest {
  readonly run: string;
  readonly repository: string;
  readonly environment?: string;
}

function watchRequestRecord(event: Event): Record<string, unknown> | undefined {
  const detail = (event as CustomEvent).detail;
  if (!detail || typeof detail !== 'object' || Array.isArray(detail)) return undefined;
  return detail as Record<string, unknown>;
}

function nonEmptyRequestPart(value: unknown): string | undefined {
  return typeof value === 'string' && value !== '' ? value : undefined;
}

/** Read a detail watch request; malformed payloads yield undefined (ignore). */
export function factoryWatchRequestDetail(event: Event): FactoryWatchRequest | undefined {
  const record = watchRequestRecord(event);
  if (!record) return undefined;
  const run = nonEmptyRequestPart(record.run),
    repository = nonEmptyRequestPart(record.repository);
  if (!run || !repository) return undefined;
  if (record.environment === undefined) return {run, repository};
  const environment = nonEmptyRequestPart(record.environment);
  return environment ? {run, repository, environment} : undefined;
}

/** Resolve a watch request to its space run; undefined when the binding is not exact. */
export function resolveWatchTarget(
  spaces: readonly Space[],
  request: FactoryWatchRequest
): {space: Space; run: FactoryRun} | undefined {
  const space =
    request.environment === undefined
      ? spaces.find((entry) => entry.factory_runs.some((run) => run.id === request.run))
      : spaces.find((entry) => entry.environment.id === request.environment);
  if (!space || space.environment.repository_id !== request.repository) return undefined;
  const run = space.factory_runs.find((entry) => entry.id === request.run);
  if (!run) return undefined;
  return {space, run};
}

export function applyWatchRequest(event: Event, actions: SetupActions): void {
  const request = factoryWatchRequestDetail(event);
  if (request) actions.requestFactoryWatch(request);
}

function renderWorkDetail(children: WorkChildScope, space: Space | undefined, issue: string): TemplateResult {
  const context: WorkDetailContext = detailContextFromScope(
    children.transport,
    children.repository,
    children.repositoryPath,
    children.scope,
    issue,
    detailRunsForIssue(space, issue)
  );
  return html`<soda-factory-work-detail .context=${context}></soda-factory-work-detail>`;
}

function renderWorkChildren(reading: SetupReading, context: WorkOverviewContext): TemplateResult {
  const children = childScopeFromReading(reading, context);
  if (!children) return html``;
  const plan: WorkPlanContext = planContextFromScope(
    children.transport,
    children.repository,
    children.repositoryPath,
    children.scope
  );
  const revision = revisionContextFromScope(children.transport, children.scope);
  return html`<soda-factory-work-plan .context=${plan}></soda-factory-work-plan>
    ${children.scope.issues.map((entry) => renderWorkDetail(children, reading.committedSpace, entry.issue))}
    ${revision ? html`<soda-factory-direction-revision .context=${revision}></soda-factory-direction-revision>` : ''}`;
}

export function renderSetupBody(
  reading: SetupReading,
  actions: SetupActions,
  host: SetupHost,
  blocked: boolean,
  read: () => SetupReading
) {
  if (reading.setup === 'repositories') {
    return renderRepositoryPicker(
      {
        query: reading.repositoryQuery,
        result: reading.repositoryResult,
        selected: reading.repositoryChoice,
        busy: reading.repositoryBusy,
        error: reading.repositoryError,
        blocked,
        createURL: reading.forgejoPrefix + '/repo/create',
        ...(reading.reviewAuthorization ? {authorization: reading.reviewAuthorization} : {}),
        ...(reading.suggestedRepository ? {suggested: reading.suggestedRepository} : {}),
      },
      {
        query: (value) => onRepositoryQuery(actions, value),
        search: (page) => {
          void searchRepositories(read(), actions, page);
        },
        select: (value) => {
          actions.setRepositoryChoice(value);
        },
        back: () => backToReview(read(), actions, host),
        continue: () => {
          void configureProject(read(), actions, host);
        },
      }
    );
  }
  if (reading.welcomeScreen) return '';
  if (reading.setup === 'intent') {
    const context = setupOverviewContext(reading);
    if (!context) return '';
    return html`<soda-factory-work-overview .context=${context}></soda-factory-work-overview>${renderWorkChildren(
        reading,
        context
      )}`;
  }
  return renderWorkspaceIntro({
    kind: setupIntroKind(reading.busy),
    heading: setupIntroHeading(reading.busy, reading.reconnectRequired),
    description: setupIntroDescription(reading.busy, reading.reconnectRequired),
    action: setupIntroAction(
      reading.busy,
      reading.reconnectRequired,
      reading.stale,
      reading.connectURL,
      blocked,
      reading.onRefresh
    ),
    helper: setupIntroHelper(reading.busy),
  });
}

export function onRepositoryQuery(actions: SetupActions, value: string) {
  actions.setRepositoryQuery(value);
  actions.setRepositoryCursors(['']);
  actions.setRepositoryChoice('');
  actions.setRepositoryResult(undefined);
  actions.setRepositoryError('');
  actions.currentRepositoryRequest()?.abort();
  actions.admitRepositoryRequest(undefined);
  actions.setRepositoryBusy(false);
}

export function renderSetup(reading: SetupReading, actions: SetupActions, host: SetupHost, read: () => SetupReading) {
  const blocked = reading.stale || !reading.canRestore;
  const formClass = reading.setup === 'repositories' ? 'soda-setup-form' : 'soda-setup-welcome';
  return html`<div
    class="soda-setup-panel"
    ?hidden=${reading.setup === 'configure'}
    @soda-factory-overview-scope=${(event: Event) => applyOverviewScope(event, read, actions)}
    @soda-factory-watch-request=${(event: Event) => applyWatchRequest(event, actions)}
  >
    <div class=${formClass}>
      <soda-intent
        .api=${actions.api}
        .actor=${reading.actorId}
        .request=${reading.requestId}
        .blocked=${blocked || !reading.activeSurface || !reading.available}
        .active=${reading.welcomeScreen || reading.setup === 'intent'}
        ?hidden=${!reading.welcomeScreen && reading.setup !== 'intent'}
        @intent-saved=${(event: Event) => actions.syncIntentRoute(intentIdentity(event))}
        @intent-resumed=${(event: Event) => {
          actions.syncIntentRoute(intentIdentity(event));
          void resumeCommittedOverview(event, read, actions);
        }}
        @intent-inventory=${(event: Event) => actions.setSavedRequests(inventoryIntents(event))}
        @intent-reviewed=${(event: Event) => chooseIntentRepository(event, read(), actions, host)}
        @intent-leave=${() => leaveSetup(read(), actions, host)}
      ></soda-intent>
      ${renderSetupBody(reading, actions, host, blocked, read)}
    </div>
  </div>`;
}

export function focusSetupPanel(host: SetupHost, isStale: () => boolean, isActiveSurface: () => boolean) {
  void host.updateComplete.then(() => {
    if (!isStale() && isActiveSurface())
      host.querySelector<HTMLElement>('.soda-setup-panel:not([hidden]) h2, .soda-project-journey h2')?.focus();
  });
}

export function focusSetupIntent(host: SetupHost, isStale: () => boolean, isActiveSurface: () => boolean) {
  void host.updateComplete.then(() => {
    if (!isStale() && isActiveSurface()) host.querySelector<SodaIntent>('soda-intent')?.focusReview();
  });
}

export function beginSetup(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.available || !reading.canRestore || !reading.activeSurface) return;
  actions.notifyFactory('');
  actions.rememberFocus();
  actions.setSetupReturn({project: reading.project, view: reading.view, mode: reading.managementMode});
  actions.setSetup('intent');
  actions.setRepositoryChoice('');
  actions.setReviewAuthorization(undefined);
  actions.setReviewedIntent(undefined);
  actions.setSetupCommitted(false);
  if (reading.requestId !== '') host.querySelector<SodaIntent>('soda-intent')?.reset();
  actions.syncIntentRoute(null);
  focusSetupIntent(host, actions.isStale, actions.isActiveSurface);
}

export function resumeSetup(intent: string, reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.available || !reading.canRestore || !reading.activeSurface) return;
  if (!requestIDPattern.test(intent)) return;
  actions.notifyFactory('');
  actions.rememberFocus();
  actions.setSetupReturn({project: reading.project, view: reading.view, mode: reading.managementMode});
  actions.setSetup('intent');
  actions.setRepositoryChoice('');
  actions.setReviewAuthorization(undefined);
  actions.setReviewedIntent(undefined);
  actions.setSetupCommitted(false);
  host.querySelector<SodaIntent>('soda-intent')?.reset();
  actions.syncIntentRoute(intent);
  focusSetupIntent(host, actions.isStale, actions.isActiveSurface);
}

function chooseIntentRepository(event: Event, reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.canRestore || !reading.activeSurface) return;
  actions.setReviewAuthorization(reviewedAuthorization(event));
  actions.setReviewedIntent(reviewedIntent(event));
  actions.setSetup('repositories');
  focusSetupPanel(host, actions.isStale, actions.isActiveSurface);
  void searchRepositories({...reading, setup: 'repositories', repositoryChoice: ''}, actions, 1);
}

export function backToReview(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.canRestore || reading.setup !== 'repositories' || !reading.activeSurface) return;
  actions.setSetup('intent');
  focusSetupIntent(host, actions.isStale, actions.isActiveSurface);
}

export function inspectCommittedRequest(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.canRestore || reading.setup !== 'configure' || !reading.activeSurface) return;
  actions.setSetup('intent');
  focusSetupIntent(host, actions.isStale, actions.isActiveSurface);
}

export function leaveSetup(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.setupReturn) {
    cancelSetup(reading, actions, host);
    return;
  }
  if (reading.requestId !== '') actions.syncIntentRoute(null);
}

export function cancelSetup(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.canRestore) return;
  actions.notifyFactory('');
  host.querySelector<SodaIntent>('soda-intent')?.reset();
  actions.currentRepositoryRequest()?.abort();
  actions.admitRepositoryRequest(undefined);
  actions.setRepositoryBusy(false);
  actions.setReviewAuthorization(undefined);
  actions.setReviewedIntent(undefined);
  actions.setSetupCommitted(false);
  actions.setSetup(null);
  if (reading.requestId !== '') actions.syncIntentRoute(null);
  if (reading.setupReturn) {
    actions.setProject(reading.setupReturn.project);
    actions.setView(reading.setupReturn.view);
    actions.setManagementMode(reading.setupReturn.mode);
  }
  actions.setSetupReturn(undefined);
  if (!actions.hasSelectedSpace() && reading.firstSpace) actions.selectProject(reading.firstSpace);
  actions.restoreFocus();
}

export function applyRepositorySearch(
  actions: SetupActions,
  request: AbortController,
  query: string,
  result: RepositoryChoices
) {
  if (actions.isStale() || actions.currentRepositoryRequest() !== request || request.signal.aborted) return;
  if (actions.currentRepositoryQuery() !== query) return;
  const cursors = actions.repositoryCursors().slice(0, result.page);
  if (result.nextCursor) cursors.push(result.nextCursor);
  actions.setRepositoryCursors(cursors);
  actions.setRepositoryResult(result);
}

export function failRepositorySearch(actions: SetupActions, request: AbortController, query: string) {
  if (actions.isStale() || actions.currentRepositoryRequest() !== request) return;
  if (actions.currentRepositoryQuery() === query)
    actions.setRepositoryError('Could not find repositories. Search again; no project was created.');
}

export function searchCursor(actions: SetupActions, page: number) {
  if (page === 1) actions.setRepositoryCursors(['']);
  return actions.repositoryCursors()[page - 1];
}

export async function searchRepositories(reading: SetupReading, actions: SetupActions, page: number) {
  if (!searchAdmitted(reading.stale, reading.available, reading.setup, reading.activeSurface)) return;
  const cursor = searchCursor(actions, page);
  if (cursor === undefined) return;
  actions.currentRepositoryRequest()?.abort();
  const request = new AbortController();
  actions.admitRepositoryRequest(request);
  const query = reading.repositoryQuery;
  const timer = window.setTimeout(() => request.abort(), 15000);
  actions.setRepositoryBusy(true);
  actions.setRepositoryError('');
  actions.setRepositoryChoice('');
  actions.setRepositoryResult(undefined);
  try {
    const result = repositoryChoices(
      await actions.api(repositorySearchPath(query, cursor), undefined, request.signal),
      page
    );
    applyRepositorySearch(actions, request, query, result);
  } catch {
    failRepositorySearch(actions, request, query);
  } finally {
    window.clearTimeout(timer);
    if (actions.currentRepositoryRequest() === request) actions.setRepositoryBusy(false);
  }
}

export async function configureProject(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  const choice = reading.repositoryResult?.items.find((item) => item.id === reading.repositoryChoice);
  if (!choice || !configureAdmitted(reading)) return;
  const intent = host.querySelector<SodaIntent>('soda-intent');
  if (!intent) return;
  actions.setRepositoryBusy(true);
  const accepted = await intent.acceptRepository(choice.id);
  actions.setRepositoryBusy(false);
  if (!accepted) {
    actions.setRepositoryError(
      'Could not save the accepted product direction. Your Project has not been created. Try again.'
    );
    return;
  }
  if (actions.isStale() || !actions.isActiveSurface()) return;
  actions.notifyFactory(intent.generationNotice);
  actions.setSetup('configure');
  actions.setSetupCommitted(true);
  void actions.showManagement(choice.id, 'journey');
}
function configureAdmitted(reading: SetupReading) {
  return !reading.repositoryBusy && !reading.stale && reading.canRestore && reading.setup === 'repositories';
}
