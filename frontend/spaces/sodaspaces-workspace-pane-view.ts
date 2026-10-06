import {html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {Space} from './sodaspaces-api.js';
import type {Area, LayoutEntry, Pane, Split, WorkspaceLayout} from './sodaspaces-layout.js';
import {consolidate, moveTab, panes, splitPane} from './sodaspaces-layout.js';
import type {NavReading} from './sodaspaces-workspace-navigation.js';
import {rowAttention, rowName} from './sodaspaces-workspace-navigation.js';
import type {PaneSession, Row, Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';
import {renderMenu, renderSessionTab} from './sodaspaces-workspace-view.js';

export interface PaneChromeInput {
  readLayout: () => WorkspaceLayout;
  readMaximized: () => string | undefined;
  setMaximized: (key: string | undefined) => void;
  readDragged: () => string | undefined;
  setDragged: (key: string | undefined) => void;
  binding: WorkspaceContext | undefined;
  spaces: Space[];
  slots: Slot[];
  available: boolean;
  stale: boolean;
  creating: boolean;
  tabHeight: number;
  projectionCompact: boolean;
  reading: NavReading;
  requestUpdate: () => void;
  closeMenus: () => void;
  arrange: (layout: WorkspaceLayout) => void;
  focusPane: (key: string) => void;
  canSplit: (axis: Split['axis'], key?: string) => boolean;
  split: (axis: Split['axis']) => void;
  move: (key: string, destination: string, before?: string) => void;
  openSaved: (entry: LayoutEntry) => void;
  tabKey: (event: KeyboardEvent, key: string, keys: string[]) => void;
  slotName: (slot: Slot) => string;
  projectName: (space: Space) => string;
  rectangle: (area: Area) => string;
  showSessions: (pane: string) => void;
  newTerminal: (pane: string) => void;
}

export function toggleMaximizedPane(input: PaneChromeInput) {
  input.closeMenus();
  input.setMaximized(input.readMaximized() ? undefined : input.readLayout().focused);
  input.requestUpdate();
}

export function consolidatePanes(input: PaneChromeInput) {
  input.closeMenus();
  input.arrange(consolidate(input.readLayout()));
}

export function showPaneSwitcher(input: PaneChromeInput) {
  if (input.binding?.kind !== 'page' || panes(input.readLayout().tree).length <= 1) return false;
  return input.projectionCompact || !!input.readMaximized();
}

export function onFocusPaneChange(input: PaneChromeInput, e: Event) {
  if (e.target instanceof HTMLSelectElement) input.focusPane(e.target.value);
}

export function renderPaneSwitcher(input: PaneChromeInput) {
  if (!showPaneSwitcher(input)) return '';
  const layout = input.readLayout();
  return html`<label
    >Panes (${panes(layout.tree).length})
    <select aria-label="Focused pane" .value=${layout.focused} @change=${(e: Event) => onFocusPaneChange(input, e)}>
      ${panes(layout.tree).map((pane, index) => html`<option value=${pane.key} ?selected=${pane.key === layout.focused}>Pane ${index + 1}</option>`)}
    </select></label
  >`;
}

export function renderPaneLayoutExtras(input: PaneChromeInput) {
  if (panes(input.readLayout().tree).length <= 1) return '';
  const maximize = input.readMaximized() ? 'Restore panes' : 'Maximize pane';
  return html`<div class="soda-menu-separator"></div>
    <button class="ui button" @click=${() => toggleMaximizedPane(input)}>${maximize}</button>
    <button class="ui button" @click=${() => consolidatePanes(input)}>Consolidate panes</button>`;
}

export function renderPaneSplitHelp(input: PaneChromeInput) {
  if (input.canSplit('right') || input.canSplit('below') || input.readMaximized()) return '';
  return html`<p class="soda-menu-help">Make the workspace larger to split this pane.</p>`;
}

export function renderPaneMenu(input: PaneChromeInput) {
  if (input.binding?.kind !== 'page') return '';
  // Button labels stay inline: wrapping whitespace text nodes around them
  // breaks exact-text assertions.
  // oxfmt-ignore
  return renderMenu(
    'Pane actions',
    'Pane ⌄',
    html`
      <p class="soda-menu-heading">Pane layout</p>
      <button class="ui button" ?disabled=${!input.canSplit('right')} @click=${() => input.split('right')}>Split right</button
      ><button class="ui button" ?disabled=${!input.canSplit('below')} @click=${() => input.split('below')}>Split below</button>
      ${renderPaneLayoutExtras(input)} ${renderPaneSplitHelp(input)}
    `
  );
}

export function paneActions(input: PaneChromeInput) {
  return html`${renderPaneSwitcher(input)} ${renderPaneMenu(input)}`;
}

export function paneTabKeys(input: PaneChromeInput, pane: Pane) {
  return input.binding?.kind === 'native' ? panes(input.readLayout().tree).flatMap((p) => p.tabs) : pane.tabs;
}

export function paneSession(input: PaneChromeInput, key: string): PaneSession[] {
  const entry = input.readLayout().entries.find((e) => e.key === key);
  const space = input.spaces.find((s) => s.environment.id === entry?.environmentId);
  if (!entry || !space || !space.login || !space.execution_allowed || space.authority_unavailable) return [];
  return [{entry, space, slot: input.slots.find((s) => s.key === key)}];
}

export function paneAriaOwns(input: PaneChromeInput, pane: Pane, navigation: boolean) {
  if (navigation || !input.slots.some((s) => s.key === pane.selected && !s.unavailable)) return '';
  return 'soda-owner-' + pane.selected;
}

export function onTabListDragOver(input: PaneChromeInput, e: DragEvent) {
  if (input.readDragged()) e.preventDefault();
}

export function onTabListDrop(input: PaneChromeInput, e: DragEvent, paneKey: string) {
  const dragged = input.readDragged();
  if (!dragged) return;
  e.preventDefault();
  input.move(dragged, paneKey);
  input.setDragged(undefined);
}

export function sessionRow(entry: LayoutEntry, slot: Slot | undefined): Row {
  return {key: entry.key, entry, ...(slot?.metadata ? {metadata: slot.metadata} : {})};
}

export function onTabDragStart(input: PaneChromeInput, event: DragEvent, key: string) {
  if (input.binding?.kind !== 'page') return;
  input.setDragged(key);
  event.dataTransfer?.setData('application/x-soda-tab', key);
  input.requestUpdate();
}

export function onTabDragEnd(input: PaneChromeInput) {
  input.setDragged(undefined);
  input.requestUpdate();
}

export function onTabDrop(input: PaneChromeInput, event: DragEvent, paneKey: string, before: string) {
  const dragged = input.readDragged();
  if (!dragged) return;
  event.preventDefault();
  event.stopPropagation();
  input.move(dragged, paneKey, before);
  input.setDragged(undefined);
}

export function sessionTabProps(input: PaneChromeInput, item: PaneSession, keys: string[], pane: Pane) {
  const {entry, space, slot} = item;
  return {
    key: entry.key,
    name: slot ? input.slotName(slot) : rowName({key: entry.key, entry}, input.slots),
    project: input.projectName(space),
    selected: entry.key === pane.selected,
    unread: !!slot?.unread,
    attention: rowAttention(input.reading, space, sessionRow(entry, slot)),
    select: () => {
      void input.openSaved(entry);
    },
    keydown: (event: KeyboardEvent) => input.tabKey(event, entry.key, keys),
    dragstart: (event: DragEvent) => onTabDragStart(input, event, entry.key),
    dragend: () => onTabDragEnd(input),
    drop: (event: DragEvent) => onTabDrop(input, event, pane.key, entry.key),
  };
}

export function renderFocusedPaneActions(input: PaneChromeInput, pane: Pane, navigation: boolean) {
  if (pane.key !== input.readLayout().focused || !pane.selected || navigation) return html``;
  return paneActions(input);
}

export function filterOverflowTabs(e: Event) {
  if (!(e.target instanceof HTMLInputElement) || !(e.currentTarget instanceof HTMLElement)) return;
  const text = e.target.value.toLocaleLowerCase();
  for (const button of e.currentTarget.parentElement?.querySelectorAll('button') || [])
    button.hidden = !button.textContent?.toLocaleLowerCase().includes(text);
}

export function overflowTabLabel(input: PaneChromeInput, item: PaneSession) {
  return (item.slot ? input.slotName(item.slot) : 'Saved terminal') + ' · ' + input.projectName(item.space);
}

export function renderTabOverflow(input: PaneChromeInput, entries: PaneSession[]) {
  return html`<details class="soda-menu soda-tab-overflow" ?hidden=${entries.length < 2}>
    <summary aria-label="Open tabs" title="Open tabs">Tabs ⌄</summary>
    <div>
      <input
        type="search"
        aria-label="Find an open tab"
        @input=${(e: Event) => filterOverflowTabs(e)}
      />${entries.map((item) => html`<button class="ui button" @click=${() => input.openSaved(item.entry)}>${overflowTabLabel(input, item)}</button>`)}
    </div>
  </details>`;
}

export function showMoveMenu(input: PaneChromeInput, pane: Pane) {
  return (
    input.binding?.kind === 'page' &&
    !!pane.selected &&
    (panes(input.readLayout().tree).length > 1 || pane.tabs.length > 1)
  );
}

export function moveTargetName(entries: PaneSession[], pane: Pane) {
  return entries.find((item) => item.entry.key === pane.selected)?.slot?.metadata?.name || 'terminal';
}

export function moveToPane(input: PaneChromeInput, pane: Pane, destination: string) {
  input.closeMenus();
  if (pane.selected) input.move(pane.selected, destination);
}

export function moveBeforeTab(input: PaneChromeInput, pane: Pane, before: string) {
  input.closeMenus();
  if (pane.selected) input.move(pane.selected, pane.key, before);
}

export function beforeTabName(entries: PaneSession[], before: string) {
  return entries.find((item) => item.entry.key === before)?.slot?.metadata?.name || 'saved tab';
}

export function renderMoveMenu(input: PaneChromeInput, pane: Pane, entries: PaneSession[]) {
  if (!showMoveMenu(input, pane)) return html``;
  return html`<details class="soda-menu soda-move-menu">
    <summary aria-label="Move terminal to pane" title="Move terminal to pane">Move ⌄</summary>
    <div>
      <p class="soda-menu-heading"><span>Move ${moveTargetName(entries, pane)}</span></p>
      ${panes(input.readLayout().tree).map((destination, i) => html`<button class="ui button" @click=${() => moveToPane(input, pane, destination.key)}>Pane ${i + 1}${destination.key === pane.key ? ' — move to end' : ''}</button>`)}${pane.tabs.filter((key) => key !== pane.selected).map((before) => html`<button class="ui button" @click=${() => moveBeforeTab(input, pane, before)}>Move before ${beforeTabName(entries, before)}</button>`)}
    </div>
  </details>`;
}

export function emptyPaneAttached(pane: Pane, entries: PaneSession[]) {
  return entries.some((e) => e.entry.key === pane.selected && e.slot && !e.slot.unavailable);
}

export function renderEmptyPane(
  input: PaneChromeInput,
  pane: Pane,
  area: Area,
  navigation: boolean,
  entries: PaneSession[]
) {
  if (navigation || (pane.selected && emptyPaneAttached(pane, entries))) return html``;
  const message = pane.selected
    ? 'Saved terminal is not attached. Select it after current authorization.'
    : 'No terminal in this pane. Splitting creates no shell.';
  return html`<div
    class="soda-empty-pane"
    style=${`width:${area.width}px;height:${Math.max(0, area.height - input.tabHeight)}px;top:${input.tabHeight}px`}
  >
    <p>${message}</p>
    <button class="ui button" ?disabled=${!input.available || input.stale} @click=${() => input.showSessions(pane.key)}>
      Use existing terminal</button
    ><button
      class="ui button"
      ?disabled=${!input.available || input.stale || input.creating}
      @click=${() => input.newTerminal(pane.key)}
    >
      New terminal here
    </button>
  </div>`;
}

export function dropEdgeStyle(axis: 'right' | 'below', area: Area) {
  return axis === 'right'
    ? `left:${area.width - 32}px;height:${area.height}px`
    : `top:${area.height - 32}px;width:${area.width}px`;
}

export function onDropEdgeOver(input: PaneChromeInput, e: DragEvent, axis: 'right' | 'below', paneKey: string) {
  if (input.readDragged() && input.canSplit(axis, paneKey)) e.preventDefault();
}

export function onDropEdgeDrop(input: PaneChromeInput, e: DragEvent, axis: 'right' | 'below', paneKey: string) {
  const dragged = input.readDragged();
  if (!dragged || !input.canSplit(axis, paneKey)) return;
  e.preventDefault();
  const key = dragged,
    next = splitPane(input.readLayout(), paneKey, axis, crypto.randomUUID(), crypto.randomUUID());
  input.arrange(moveTab(next, key, next.focused));
  input.setDragged(undefined);
}

export function renderDropEdges(input: PaneChromeInput, pane: Pane, area: Area) {
  if (!input.readDragged() || input.binding?.kind !== 'page') return html``;
  return (['right', 'below'] as const).map(
    (axis) =>
      html`<div
        class=${'soda-drop-edge ' + axis}
        ?hidden=${!input.canSplit(axis, pane.key)}
        style=${dropEdgeStyle(axis, area)}
        @dragover=${(e: DragEvent) => onDropEdgeOver(input, e, axis, pane.key)}
        @drop=${(e: DragEvent) => onDropEdgeDrop(input, e, axis, pane.key)}
      >
        Split ${axis}
      </div>`
  );
}

export function paneChrome(input: PaneChromeInput, pane: Pane, area: Area, navigation = false) {
  const keys = paneTabKeys(input, pane);
  const entries = keys.flatMap((key) => paneSession(input, key));
  const layout = input.readLayout();
  return html`<section
    class="soda-pane-chrome"
    role="group"
    aria-label=${'Pane ' + (panes(layout.tree).findIndex((p) => p.key === pane.key) + 1)}
    aria-owns=${paneAriaOwns(input, pane, navigation)}
    data-pane=${pane.key}
    style=${input.rectangle({...area, height: input.tabHeight})}
  >
    <div
      class="soda-workspace-tabs"
      role="tablist"
      aria-label="Terminal sessions"
      @dragover=${(e: DragEvent) => onTabListDragOver(input, e)}
      @drop=${(e: DragEvent) => onTabListDrop(input, e, pane.key)}
    >
      ${repeat(
        entries,
        ({entry}) => entry.key,
        (item) =>
          renderSessionTab(
            sessionTabProps(input, item, keys, pane),
            navigation,
            input.binding?.kind === 'page',
            (event) => onTabListDragOver(input, event)
          )
      )}
    </div>
    ${renderFocusedPaneActions(input, pane, navigation)} ${renderTabOverflow(input, entries)}
    ${renderMoveMenu(input, pane, entries)} ${renderEmptyPane(input, pane, area, navigation, entries)}
    ${renderDropEdges(input, pane, area)}
  </section>`;
}
