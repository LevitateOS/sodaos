import {html} from 'lit';
import type {Space} from './sodaspaces-api.js';
import {factoryAuthorityText, factoryControlText} from './sodaspaces-factory-response.js';
import {attentionReason} from './sodaspaces-attention.js';
import type {LayoutEntry, PaneTree} from './sodaspaces-layout.js';
import {paneFor, sameLocator} from './sodaspaces-layout.js';
import {renderProjectNavigation} from './sodaspaces-workspace-view.js';
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
function projectSubtitle(space: Space, status: string) {
  return (
    status +
    (space.tailnet_state ? ' · Tailnet policy: ' + space.tailnet_state : '') +
    factoryAuthorityText(space.factory_authority) +
    factoryControlText(space.factory_control)
  );
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
export function projectRows(input: ProjectRowsInput) {
  const query = input.search.toLocaleLowerCase();
  const matched = rows(input.space, input.entries, input.reading.slots).filter((row) =>
    rowMatchesQuery(row, input.reading.slots, query, input.projectName)
  );
  const shown = input.attentionOnly ? matched.filter((row) => rowAttention(input.reading, input.space, row)) : matched;
  return html`${renderProjectNavigation(
    input.projectName,
    projectSubtitle(input.space, input.status),
    shown.map((row) => rowNavItem(input.reading, input.space, row, input.selected, input.tree, input.selectRow)),
    input.reading.stale || !input.reading.available,
    () => {
      void input.showManagement(input.space.environment.repository_id);
    },
    pageProjectNav(input.space, input.kind, input.project, input.state, input.selectProject)
  )}${input.factory}`;
}
