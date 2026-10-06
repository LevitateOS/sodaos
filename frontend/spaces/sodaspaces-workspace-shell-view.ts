import {html} from 'lit';
import type {Space} from './sodaspaces-inventory-response.js';
import {renderWorkspaceIntro} from './sodaspaces-workspace-view.js';

export function workspaceBlocked(stale: boolean, available: boolean) {
  return stale || !available;
}
function navigationVisible(view: 'terminal' | 'sessions' | 'project', compact: boolean, sidebar: number | null) {
  return view === 'sessions' || (!compact && sidebar !== null);
}
export function busyAttr(busy: boolean) {
  return busy ? 'true' : 'false';
}
export function workspaceClasses(
  compact: boolean,
  setupScreen: boolean,
  welcomeScreen: boolean,
  setup: 'repositories' | 'configure' | null,
  workspaceIntro: boolean
) {
  let cls = 'soda-workspace ' + (compact ? 'is-compact' : 'is-wide');
  if (setupScreen) cls += ' is-setup';
  if (welcomeScreen) cls += ' is-welcome';
  if (setup) cls += ' is-project-setup';
  if (workspaceIntro) cls += ' is-workspace-intro';
  return cls;
}
export function renderSetupHeading(setupScreen: boolean) {
  if (!setupScreen) return '';
  return html`<div class="soda-setup-heading">
    <h1>Spaces</h1>
    <p>Your projects and terminals, together.</p>
  </div>`;
}
function hideListStatus(status: string, setupScreen: boolean, setup: 'repositories' | 'configure' | null) {
  return !status || (setupScreen && !setup);
}
function retryLoading(complete: boolean, stale: boolean, busy: boolean, onRefresh: () => void) {
  if (complete || stale) return '';
  const label = busy ? 'Loading…' : 'Retry loading';
  return html`<button class="ui button soda-quiet-action" ?disabled=${busy} @click=${onRefresh}>${label}</button>`;
}
export function renderStatusBanners(
  status: string,
  setupScreen: boolean,
  setup: 'repositories' | 'configure' | null,
  storageNotice: string,
  complete: boolean,
  stale: boolean,
  busy: boolean,
  onRefresh: () => void
) {
  // The status element must contain no whitespace text so its empty
  // state reads empty; the formatter would reintroduce it.
  // oxfmt-ignore
  return html`<div
      id="sodaspaces-status"
      class="soda-feedback soda-list-feedback"
      data-tone="warning"
      role="status"
      ?hidden=${hideListStatus(status, setupScreen, setup)}><span>${status}</span>${retryLoading(complete, stale, busy, onRefresh)}</div>
      <p class="soda-feedback" data-tone="warning" role="status" ?hidden=${!storageNotice}>
        ${storageNotice}
      </p>`;
}
function setupBackDisabled(stale: boolean, canRestore: boolean) {
  return stale || !canRestore;
}
export function renderSetupTopbar(
  setup: 'repositories' | 'configure' | null,
  stale: boolean,
  canRestore: boolean,
  onCancelSetup: () => void,
  onSetupBack: () => void
) {
  if (!setup) return '';
  const cancel =
    setup === 'configure'
      ? html`<button
          class="ui button soda-quiet-action"
          ?disabled=${setupBackDisabled(stale, canRestore)}
          @click=${onCancelSetup}
        >
          Cancel setup
        </button>`
      : '';
  return html`<div class="soda-setup-topbar">
    <button class="ui button soda-quiet-action" ?disabled=${setupBackDisabled(stale, canRestore)} @click=${onSetupBack}>
      <span aria-hidden="true">←</span> Back</button
    >${cancel}
  </div>`;
}
export function hideWorkspaceBody(setupScreen: boolean, setup: 'repositories' | 'configure' | null) {
  return setupScreen && setup !== 'configure';
}
export function workspaceBodyClass(view: 'terminal' | 'sessions' | 'project') {
  return 'soda-workspace-body' + (view === 'sessions' ? ' is-navigating' : '');
}
export function workspaceBodyStyle(
  setupScreen: boolean,
  compact: boolean,
  sidebar: number | null,
  view: 'terminal' | 'sessions' | 'project'
) {
  if (!setupScreen && !compact && sidebar !== null && view !== 'sessions') {
    return `grid-template-columns:${sidebar}px 6px minmax(0,1fr)`;
  }
  return 'grid-template-columns:minmax(0,1fr)';
}
export function hideNavigation(
  setupScreen: boolean,
  view: 'terminal' | 'sessions' | 'project',
  compact: boolean,
  sidebar: number | null,
  stale: boolean
) {
  return setupScreen || !navigationVisible(view, compact, sidebar) || stale;
}
export function hideSidebarDivider(
  setupScreen: boolean,
  compact: boolean,
  view: 'terminal' | 'sessions' | 'project',
  sidebar: number | null
) {
  if (setupScreen || compact || view === 'sessions') return true;
  return sidebar === null;
}
export function hideCanvas(view: 'terminal' | 'sessions' | 'project', stale: boolean) {
  return view !== 'terminal' || stale;
}
function inventoryHelper(spaces: Space[], connectURL: string) {
  const reconnect = spaces.some((space) => space.authority_unavailable)
    ? html`<a href=${connectURL}>Reconnect to Forgejo</a>`
    : '';
  return html`Your existing projects are preserved. ${reconnect}`;
}
export function renderInventoryRecovery(
  recovery: boolean,
  busy: boolean,
  onRefresh: () => void,
  spaces: Space[],
  connectURL: string
) {
  if (!recovery) return '';
  return renderWorkspaceIntro({
    kind: 'unavailable',
    heading: 'Project status unavailable',
    description: html`We couldn’t check your projects right now. Try loading the list again.`,
    action: html`<button class="ui primary button" ?disabled=${busy} @click=${onRefresh}>Refresh projects</button>`,
    helper: inventoryHelper(spaces, connectURL),
  });
}
export function renderFirstTerminalIntro(
  firstTerminal: boolean,
  space: Space | undefined,
  projectName: string,
  defaultName: string,
  blocked: boolean,
  creating: boolean,
  onNew: () => void
) {
  if (!firstTerminal) return '';
  return renderWorkspaceIntro({
    kind: 'terminal',
    heading: 'Open your first terminal',
    description: html`<span>Your account is ready in ${space ? projectName : ''}.</span
      ><span>Open a browser terminal to start working.</span>`,
    action: html`<button
      class="ui primary button"
      data-environment-id=${space?.environment.id || ''}
      data-terminal-name=${defaultName}
      ?disabled=${blocked || creating}
      @click=${onNew}
    >
      <span aria-hidden="true">＋</span> New terminal
    </button>`,
    helper: html`Project account: ${space?.login}`,
  });
}
export function hidePaneChrome(firstTerminal: boolean, inventoryRecovery: boolean) {
  return firstTerminal || inventoryRecovery;
}
export function hideManagement(view: 'terminal' | 'sessions' | 'project', stale: boolean) {
  return view !== 'project' || stale;
}
