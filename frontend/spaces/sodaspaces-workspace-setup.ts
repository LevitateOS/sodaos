import {html} from 'lit';
import {renderWelcome, renderWorkspaceIntro} from './sodaspaces-workspace-view.js';
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
  setup: 'repositories' | 'configure' | null,
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
  setup: 'repositories' | 'configure' | null;
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
  hasSelectedSpace: boolean;
  firstSpace: Space | undefined;
  onBegin: () => void;
  onRefresh: () => void;
}

export interface SetupActions {
  setSetup(setup: 'repositories' | 'configure' | null): void;
  setRepositoryQuery(query: string): void;
  setRepositoryChoice(choice: string): void;
  setRepositoryResult(result: RepositoryChoices | undefined): void;
  setRepositoryCursors(cursors: string[]): void;
  setRepositoryError(error: string): void;
  setRepositoryBusy(busy: boolean): void;
  setSetupReturn(target: SetupReturn | undefined): void;
  setProject(project: string): void;
  setView(view: 'terminal' | 'sessions' | 'project'): void;
  setManagementMode(mode: 'standard' | 'journey' | 'settings'): void;
  repositoryCursors(): string[];
  currentRepositoryRequest(): AbortController | undefined;
  admitRepositoryRequest(request: AbortController | undefined): void;
  currentRepositoryQuery(): string;
  isStale(): boolean;
  isActiveSurface(): boolean;
  selectProject(space: Space): void;
  rememberFocus(): void;
  restoreFocus(): void;
  showManagement(repositoryId: string, mode: 'standard' | 'journey' | 'settings'): void;
  api(path: string, body?: Record<string, unknown>, signal?: AbortSignal): Promise<unknown>;
}

export interface SetupHost {
  updateComplete: Promise<unknown>;
  querySelector<E extends Element>(selectors: string): E | null;
}

export function clearRepositorySearch(actions: SetupActions) {
  actions.setRepositoryQuery('');
  actions.setRepositoryError('');
  actions.setRepositoryChoice('');
  actions.setRepositoryResult(undefined);
  actions.setRepositoryCursors(['']);
  actions.currentRepositoryRequest()?.abort();
  actions.admitRepositoryRequest(undefined);
  actions.setRepositoryBusy(false);
}

export function setupPanelClass(setup: 'repositories' | 'configure' | null) {
  return setup === 'repositories' ? 'soda-setup-form' : 'soda-setup-welcome';
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
      },
      {
        query: (value) => onRepositoryQuery(actions, value),
        search: (page) => {
          void searchRepositories(read(), actions, page);
        },
        select: (value) => {
          actions.setRepositoryChoice(value);
        },
        back: () => cancelSetup(read(), actions),
        continue: () => configureProject(read(), actions),
      }
    );
  }
  if (reading.welcomeScreen) return renderWelcome(blocked, reading.onBegin);
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

export function setupUnavailableHeading(reading: SetupReading) {
  if (reading.busy) return 'Loading projects…';
  if (reading.reconnectRequired) return 'Reconnect to Forgejo';
  return 'Could not load projects';
}

export function setupUnavailableDescription(reading: SetupReading) {
  if (reading.busy) return html`Checking the projects you can access.`;
  if (reading.reconnectRequired) return html`Sign in again to restore your Forgejo access.`;
  return html`We couldn’t load your project list. Try again.`;
}

export function setupUnavailableAction(reading: SetupReading, blocked: boolean) {
  if (reading.busy) return html``;
  if (reading.reconnectRequired)
    return html`<a class="ui primary button" href=${reading.connectURL}>Reconnect to Forgejo</a>`;
  if (reading.stale)
    return html`<button class="ui primary button" @click=${() => window.location.reload()}>Reload Spaces</button>`;
  return html`<button class="ui primary button" ?disabled=${blocked} @click=${() => reading.onRefresh()}>
    Retry projects
  </button>`;
}

export function setupUnavailableHelper(reading: SetupReading) {
  if (reading.busy) return html`This will not create or start anything.`;
  return html`Your existing projects and terminals are not replaced.`;
}

export function renderSetupUnavailable(reading: SetupReading, blocked: boolean) {
  return renderWorkspaceIntro({
    kind: reading.busy ? 'loading' : 'unavailable',
    heading: setupUnavailableHeading(reading),
    description: setupUnavailableDescription(reading),
    action: setupUnavailableAction(reading, blocked),
    helper: setupUnavailableHelper(reading),
  });
}

export function renderSetup(reading: SetupReading, actions: SetupActions, host: SetupHost, read: () => SetupReading) {
  const blocked = reading.stale || !reading.canRestore;
  const formClass = reading.setup === 'repositories' ? 'soda-setup-form' : 'soda-setup-welcome';
  return html`<div class="soda-setup-panel" ?hidden=${reading.setup === 'configure'}>
    <div class=${formClass}>${renderSetupBody(reading, actions, host, blocked, read)}</div>
  </div>`;
}

export function focusSetupPanel(host: SetupHost, isStale: () => boolean, isActiveSurface: () => boolean) {
  void host.updateComplete.then(() => {
    if (!isStale() && isActiveSurface())
      host.querySelector<HTMLElement>('.soda-setup-panel:not([hidden]) h2, .soda-project-journey h2')?.focus();
  });
}

export function beginSetup(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.available || !reading.canRestore || !reading.activeSurface) return;
  actions.rememberFocus();
  actions.setSetupReturn({project: reading.project, view: reading.view, mode: reading.managementMode});
  actions.setSetup('repositories');
  actions.setRepositoryChoice('');
  focusSetupPanel(host, actions.isStale, actions.isActiveSurface);
  void searchRepositories({...reading, setup: 'repositories', repositoryChoice: ''}, actions, 1);
}

export function changeRepository(reading: SetupReading, actions: SetupActions, host: SetupHost) {
  if (reading.stale || !reading.canRestore || reading.setup !== 'configure' || !reading.activeSurface) return;
  actions.setSetup('repositories');
  focusSetupPanel(host, actions.isStale, actions.isActiveSurface);
}

export function cancelSetup(reading: SetupReading, actions: SetupActions) {
  if (reading.stale || !reading.canRestore) return;
  actions.currentRepositoryRequest()?.abort();
  actions.admitRepositoryRequest(undefined);
  actions.setRepositoryBusy(false);
  actions.setSetup(null);
  if (reading.setupReturn) {
    actions.setProject(reading.setupReturn.project);
    actions.setView(reading.setupReturn.view);
    actions.setManagementMode(reading.setupReturn.mode);
  }
  actions.setSetupReturn(undefined);
  if (!reading.hasSelectedSpace && reading.firstSpace) actions.selectProject(reading.firstSpace);
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

export function configureProject(reading: SetupReading, actions: SetupActions) {
  const choice = reading.repositoryResult?.items.find((item) => item.id === reading.repositoryChoice);
  if (!choice || reading.repositoryBusy || reading.stale || !reading.canRestore || reading.setup !== 'repositories')
    return;
  actions.setSetup('configure');
  void actions.showManagement(choice.id, 'journey');
}
