import {html} from 'lit';
import type {Space} from './sodaspaces-api.js';
import type {WorkspaceContext} from './sodaspaces-workspace-types.js';

export function sessionsButtonHidden(
  kind: 'native' | 'page' | undefined,
  compact: boolean,
  sidebar: number | null,
  view: 'terminal' | 'sessions' | 'project'
) {
  if (kind !== 'page') return false;
  return !compact && sidebar !== null && view !== 'sessions';
}
export function sessionsButtonLabel(kind: 'native' | 'page' | undefined) {
  return kind === 'page' ? 'Projects' : 'Sessions';
}
function toolbarProjectTitle(space: Space | undefined, projectName: string) {
  return space ? projectName : 'Spaces';
}
export function renderToolbarProject(
  kind: 'native' | 'page' | undefined,
  space: Space | undefined,
  state: string,
  status: string,
  projectName: string
) {
  if (kind !== 'page') return '';
  const stateBadge = space ? html`<span class="soda-project-state" data-state=${state}>${status}</span>` : '';
  const profile = space?.environment.profile
    ? html`<p class="soda-project-profile">Rocky Linux ${space.environment.profile.version} · Terminal</p>`
    : '';
  const title = toolbarProjectTitle(space, projectName);
  return html`<div class="soda-selected-project">
    <div class="soda-project-heading">
      <h1 title=${title}>${title}</h1>
      ${stateBadge}
    </div>
    ${profile}
  </div>`;
}
export function hideNewTerminal(firstTerminal: boolean, kind: 'native' | 'page' | undefined, space: Space | undefined) {
  if (firstTerminal) return true;
  if (kind !== 'page') return false;
  return !space?.login || space.observed?.running !== true;
}
export function disableNewTerminal(
  blocked: boolean,
  creating: boolean,
  kind: 'native' | 'page' | undefined,
  space: Space | undefined
) {
  if (blocked || creating) return true;
  if (kind !== 'page') return false;
  return !space?.environment.provisioned || !!space.authority_unavailable || !!space.native_unavailable;
}
export function newTerminalButtonClass(kind: 'native' | 'page' | undefined) {
  return kind === 'page' ? 'ui primary button' : 'ui button';
}
export function newTerminalButtonLabel(kind: 'native' | 'page' | undefined) {
  return kind === 'page' ? html`<span aria-hidden="true">＋</span> New terminal` : '＋';
}
export function renderProjectSettingsButton(
  kind: 'native' | 'page' | undefined,
  blocked: boolean,
  space: Space | undefined,
  onShow: () => void
) {
  if (kind !== 'page') return '';
  return html`<button class="ui button" ?disabled=${blocked || !space} @click=${onShow}>Project settings</button>`;
}
function hideBackButton(
  view: 'terminal' | 'sessions' | 'project',
  kind: 'native' | 'page' | undefined,
  managementMode: 'standard' | 'journey' | 'settings'
) {
  if (view === 'terminal') return true;
  return kind === 'page' && view === 'project' && managementMode === 'journey';
}
function backButtonLabel(kind: 'native' | 'page' | undefined) {
  return kind === 'page' ? 'Back to workspace' : 'Back to terminal';
}
export function renderBackButton(
  view: 'terminal' | 'sessions' | 'project',
  kind: 'native' | 'page' | undefined,
  managementMode: 'standard' | 'journey' | 'settings',
  onBack: () => void
) {
  if (hideBackButton(view, kind, managementMode)) return '';
  return html`<button class="ui button" @click=${onBack}>${backButtonLabel(kind)}</button>`;
}
export function renderOpenInDrawerOption(
  kind: 'native' | 'page' | undefined,
  blocked: boolean,
  openingDrawer: boolean,
  selected: string,
  onOpen: () => void
) {
  if (kind !== 'page') return '';
  return html`<button class="ui button" ?disabled=${blocked || openingDrawer || !selected} @click=${onOpen}>
    Open in drawer
  </button>`;
}
export function renderToggleSidebarOption(kind: 'native' | 'page' | undefined, onToggle: () => void) {
  if (kind !== 'page') return '';
  return html`<button class="ui button" @click=${onToggle}>Toggle sidebar</button>`;
}
function nativeManagementTarget(binding: WorkspaceContext | undefined) {
  return binding?.kind === 'native' ? binding.repositoryId : '';
}
export function renderNativeManagementOption(
  binding: WorkspaceContext | undefined,
  blocked: boolean,
  showManagement: (repository: string) => void
) {
  if (binding?.kind !== 'native' || !binding.repositoryId) return '';
  return html`<button
    class="ui button"
    ?disabled=${blocked}
    @click=${() => {
      const repository = nativeManagementTarget(binding);
      if (repository) void showManagement(repository);
    }}
  >
    Repository environment / access
  </button>`;
}
export function renderSpacesLink() {
  return '';
}
