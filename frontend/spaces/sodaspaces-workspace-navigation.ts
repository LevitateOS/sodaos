import {html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {Space} from './sodaspaces-inventory-response.js';
import type {SavedRequest} from './sodaspaces-intent-response.js';
import {attentionReason} from './sodaspaces-attention.js';
import type {LayoutEntry, PaneTree} from './sodaspaces-layout.js';
import {paneFor, sameLocator} from './sodaspaces-layout.js';
import {
  renderProjectDirectoryRow,
  renderProjectNavigation,
  renderProjectTerminalList,
} from './sodaspaces-session-navigation-view.js';
import type {Row, Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';

export interface NavReading {
  stale: boolean;
  available: boolean;
  slots: Slot[];
  observedAt: number;
  now: number;
}

function rowAttentionBlocked(reading: NavReading, space: Space) {
  return reading.stale || !reading.available || !space.login || !space.execution_allowed || space.authority_unavailable;
}
export function rowAttention(reading: NavReading, space: Space, row: Row) {
  if (rowAttentionBlocked(reading, space)) return '';
  if (space.native_unavailable) return 'Native status unavailable; refresh';
  const slot = reading.slots.find((slot) => slot.key === row.key);
  return attentionReason(
    row.metadata,
    slot?.metadata ? slot.observedAt : reading.observedAt,
    reading.now,
    slot?.observation?.state
  );
}
export function attentionRows(reading: NavReading, spaces: Space[], entries: LayoutEntry[]) {
  const seen = new Set<string>();
  return spaces.flatMap((space) =>
    rows(space, entries, reading.slots)
      .filter((row) => {
        const key = row.metadata?.id || row.key;
        if (!rowAttention(reading, space, row) || seen.has(key)) return false;
        seen.add(key);
        return true;
      })
      .map((row) => ({
        space,
        row,
      }))
  );
}
/** H-I saved-request list entries (UX-I03/U03). List supply and parsing stay with U request state. */
export interface RequestGroups {
  drafts: SavedRequest[];
  accepted: SavedRequest[];
  closed: SavedRequest[];
}
function byRecency(a: SavedRequest, b: SavedRequest) {
  return b.created_unix - a.created_unix || (a.intent < b.intent ? -1 : 1);
}
/** Group saved requests by lifecycle state; recent first, stable intent identity breaks ties. */
export function groupSavedRequests(summaries: SavedRequest[]): RequestGroups {
  const groups: RequestGroups = {drafts: [], accepted: [], closed: []};
  for (const summary of [...summaries].sort(byRecency)) {
    if (summary.state === 'accepted') groups.accepted.push(summary);
    else if (summary.state === 'draft') groups.drafts.push(summary);
    else groups.closed.push(summary);
  }
  return groups;
}
/** Display label for a request's repository binding; "0" means not yet bound. */
export function requestRepositoryLabel(repository: string): string {
  return repository === '0' || !repository ? 'No repository yet' : repository;
}
export interface RequestGroupsInput {
  groups: RequestGroups;
  resume: (intent: string) => void;
}
/** Discovery section for grouped saved requests; empty inventories render nothing. */
export function renderRequestGroups(input: RequestGroupsInput) {
  const sections: {title: string; items: SavedRequest[]}[] = [
    {title: 'Drafts', items: input.groups.drafts},
    {title: 'Accepted', items: input.groups.accepted},
    {title: 'Closed', items: input.groups.closed},
  ].filter((section) => section.items.length);
  if (!sections.length) return '';
  return html`<section aria-label="Saved requests">
    ${sections.map(
      (section) => html`<h3>${section.title} (${section.items.length})</h3>
        <ul>
          ${section.items.map(
            (item) => html`<li>
              ${item.objective} · ${requestRepositoryLabel(item.repository)}
              <button class="ui button" aria-label="Resume ${item.objective}" @click=${() => input.resume(item.intent)}>
                Resume
              </button>
            </li>`
          )}
        </ul>`
    )}
  </section>`;
}
export interface FilteredSpacesInput {
  reading: NavReading;
  spaces: Space[];
  entries: LayoutEntry[];
  search: string;
  attentionOnly: boolean;
  thisPage: boolean;
  binding: WorkspaceContext | undefined;
  projectName: (space: Space) => string;
}
export function filteredSpaces(input: FilteredSpacesInput) {
  const query = input.search.toLocaleLowerCase();
  return input.spaces.filter(
    (space) =>
      (!input.attentionOnly ||
        rows(space, input.entries, input.reading.slots).some((row) => rowAttention(input.reading, space, row))) &&
      (!input.thisPage ||
        (input.binding?.kind === 'native' && space.environment.repository_id === input.binding.pageRepositoryId)) &&
      (!query ||
        input.projectName(space).toLocaleLowerCase().includes(query) ||
        rows(space, input.entries, input.reading.slots).some((row) =>
          rowName(row, input.reading.slots).toLocaleLowerCase().includes(query)
        ))
  );
}
export function rows(space: Space, entries: LayoutEntry[], slots: Slot[]): Row[] {
  if (space.authority_unavailable || !space.execution_allowed || !space.login) return [];
  const relevant = entries.filter((e) => e.environmentId === space.environment.id),
    seen = new Set<string>();
  const listed = space.terminals.map((metadata) => {
    const entry = relevant.find((e) => sameLocator(e.locator, {kind: 'existing', id: metadata.id}));
    if (entry) seen.add(entry.key);
    const observed = slots.find((s) => s.key === entry?.key)?.metadata || metadata;
    return {
      key: entry?.key || metadata.id,
      ...(entry
        ? {
            entry,
          }
        : {}),
      metadata: observed,
    };
  });
  return [
    ...listed,
    ...relevant
      .filter((e) => !seen.has(e.key))
      .map((entry) => {
        const metadata = slots.find((s) => s.key === entry.key)?.metadata;
        return {
          key: entry.key,
          entry,
          ...(metadata
            ? {
                metadata,
              }
            : {}),
        };
      }),
  ];
}
function unnamedRow(row: Row) {
  if (row.metadata) return 'Terminal ' + row.metadata.id.slice(0, 8);
  if (row.entry?.locator.kind === 'new') return 'Unsent terminal';
  return 'Saved ' + (row.entry?.locator.kind || 'unknown') + ' terminal';
}
export function rowName(row: Row, slots: Slot[]) {
  if (row.metadata?.name) return row.metadata.name;
  const proposed = slots.find((slot) => slot.key === row.key)?.proposedName;
  if (proposed) return proposed;
  return unnamedRow(row);
}
function rowMatchesQuery(row: Row, slots: Slot[], query: string, spaceName: string) {
  return (
    !query || spaceName.toLocaleLowerCase().includes(query) || rowName(row, slots).toLocaleLowerCase().includes(query)
  );
}
function rowTerminalId(row: Row) {
  if (row.metadata?.id) return row.metadata.id;
  if (row.entry?.locator.kind === 'existing') return row.entry.locator.id;
  return '';
}
function rowDescription(row: Row, tree: PaneTree) {
  if (row.metadata && row.metadata.state !== 'ready') return row.metadata.state;
  if (row.entry && !paneFor(tree, row.entry.key)) return 'Hidden';
  if (row.entry) return 'In this window';
  return '';
}
function rowDisabled(reading: NavReading, space: Space) {
  return reading.stale || !reading.available || !space.login || !space.execution_allowed || space.authority_unavailable;
}
function rowNavItem(
  reading: NavReading,
  space: Space,
  row: Row,
  selected: string,
  tree: PaneTree,
  selectRow: (space: Space, row: Row) => void
) {
  return {
    key: row.key,
    active: row.key === selected,
    terminalId: rowTerminalId(row),
    name: rowName(row, reading.slots),
    unread: !!reading.slots.find((slot) => slot.key === row.key)?.unread,
    attention: rowAttention(reading, space, row),
    description: rowDescription(row, tree),
    disabled: rowDisabled(reading, space),
    select: () => selectRow(space, row),
  };
}
function pageProjectNav(
  space: Space,
  kind: 'native' | 'page' | undefined,
  project: string,
  state: 'running' | 'stopped' | 'unknown',
  selectProject: (space: Space) => void
) {
  if (kind !== 'page') return;
  return {
    active: project === space.environment.repository_id,
    environmentId: space.environment.id,
    state,
    select: () => selectProject(space),
  };
}
export interface ProjectDirectoryInput {
  reading: NavReading;
  spaces: Space[];
  entries: LayoutEntry[];
  project: string;
  projectName: (space: Space) => string;
  projectState: (space: Space) => 'running' | 'stopped' | 'unknown';
  selectProject: (space: Space) => void;
}
/** Project selection changes display context only; execution controls live in the selected Project. */
export function renderProjectDirectory(input: ProjectDirectoryInput) {
  const disabled = input.reading.stale || !input.reading.available;
  return html`<div class="soda-project-directory">
    ${repeat(
      input.spaces,
      (space) => space.environment.id,
      (space) =>
        renderProjectDirectoryRow({
          environmentId: space.environment.id,
          name: input.projectName(space),
          state: disabled ? 'unknown' : input.projectState(space),
          attention: attentionRows(input.reading, [space], input.entries).length,
          active: input.project === space.environment.repository_id,
          disabled,
          select: () => input.selectProject(space),
        })
    )}
  </div>`;
}
export interface ProjectRowsInput {
  reading: NavReading;
  space: Space;
  search: string;
  attentionOnly: boolean;
  selected: string;
  entries: LayoutEntry[];
  tree: PaneTree;
  kind: 'native' | 'page' | undefined;
  project: string;
  state: 'running' | 'stopped' | 'unknown';
  status: string;
  projectName: string;
  factory: unknown;
  selectRow: (space: Space, row: Row) => void;
  showManagement: (repository: string) => void;
  selectProject: (space: Space) => void;
}
function projectSessionRows(input: ProjectRowsInput) {
  const query = input.search.toLocaleLowerCase();
  const matched = rows(input.space, input.entries, input.reading.slots).filter((row) =>
    rowMatchesQuery(row, input.reading.slots, query, input.projectName)
  );
  const shown = input.attentionOnly ? matched.filter((row) => rowAttention(input.reading, input.space, row)) : matched;
  return shown.map((row) => rowNavItem(input.reading, input.space, row, input.selected, input.tree, input.selectRow));
}
export function renderProjectTerminals(input: ProjectRowsInput) {
  return renderProjectTerminalList(projectSessionRows(input));
}
export function projectRows(input: ProjectRowsInput) {
  return html`${renderProjectNavigation(
    input.projectName,
    input.status,
    projectSessionRows(input),
    input.reading.stale || !input.reading.available,
    () => {
      void input.showManagement(input.space.environment.repository_id);
    },
    pageProjectNav(input.space, input.kind, input.project, input.state, input.selectProject)
  )}${input.factory}`;
}
