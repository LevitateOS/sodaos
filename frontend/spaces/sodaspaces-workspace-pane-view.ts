import {html} from 'lit';
import {repeat} from 'lit/directives/repeat.js';
import type {Space} from './sodaspaces-inventory-response.js';
import type {Area, LayoutEntry, Pane, Split, WorkspaceLayout} from './sodaspaces-layout.js';
import {moveTab, panes, splitPane} from './sodaspaces-layout.js';
import {consolidatePanes, toggleMaximizedPane} from './sodaspaces-workspace-layout.js';
import type {NavReading} from './sodaspaces-workspace-navigation.js';
import {rowAttention, rowName} from './sodaspaces-workspace-navigation.js';
import type {PaneSession, Row, Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';
import {renderMenu} from './sodaspaces-workspace-view.js';
import {renderSessionTab} from './sodaspaces-session-navigation-view.js';

/** Strip action icons: each trigger keeps its text label for tooltips and assistive tech. */
const icons = {
  layout: html`<svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    aria-hidden="true"
    fill="none"
    stroke="currentColor"
    stroke-width="1.4"
  >
    <rect x="1.5" y="2.5" width="13" height="11" />
    <path d="M8 2.5v11" />
  </svg>`,
  tabs: html`<svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    aria-hidden="true"
    fill="none"
    stroke="currentColor"
    stroke-width="1.4"
  >
    <path d="M2.5 4h11M2.5 8h11M2.5 12h7" />
  </svg>`,
  move: html`<svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    aria-hidden="true"
    fill="none"
    stroke="currentColor"
    stroke-width="1.4"
  >
    <path d="M2 8h10M9 4.5 12.5 8 9 11.5M14.5 3v10" />
  </svg>`,
};
const unnamed = 'Unnamed terminal';

export interface PaneChromeInput {
  readLayout: () => WorkspaceLayout;
  readMaximized: () => string | undefined;
  setMaximized: (key: string | undefined) => void;
  readDragged: () => string | undefined;
  setDragged: (key: string | undefined) => void;
  readDropHint: () => {pane: string; before: string | undefined} | undefined;
  setDropHint: (hint: {pane: string; before: string | undefined} | undefined) => void;
  readClosing: () => string | undefined;
  requestClose: (key: string) => void;
  cancelClose: () => void;
  confirmClose: () => void;
  rename: (key: string) => void;
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
    icons.layout,
    html`
      <p class="soda-menu-heading">Layout</p>
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

export function onTabListDragOver(input: PaneChromeInput, e: DragEvent, paneKey?: string) {
  if (!input.readDragged()) return;
  e.preventDefault();
  // Over the empty part of the strip: the tab lands at the end.
  if (paneKey && e.target === e.currentTarget) input.setDropHint({pane: paneKey, before: undefined});
}

export function onTabListDrop(input: PaneChromeInput, e: DragEvent, paneKey: string) {
  const dragged = input.readDragged();
  if (!dragged) return;
  e.preventDefault();
  const hint = input.readDropHint();
  input.move(dragged, paneKey, hint?.pane === paneKey ? hint.before : undefined);
  input.setDragged(undefined);
}

/** Over a tab: land before it on its left half, after it on its right half. */
export function onTabDragOver(input: PaneChromeInput, e: DragEvent, pane: Pane, key: string) {
  if (!input.readDragged()) return;
  e.preventDefault();
  e.stopPropagation();
  const target = e.currentTarget as HTMLElement,
    box = target.getBoundingClientRect(),
    after = e.clientX > box.left + box.width / 2,
    index = pane.tabs.indexOf(key);
  input.setDropHint({pane: pane.key, before: after ? pane.tabs[index + 1] : key});
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
  const hint = input.readDropHint();
  const target = hint?.pane === paneKey ? hint.before : before;
  if (target !== dragged) input.move(dragged, paneKey, target);
  input.setDragged(undefined);
}

/** Which side of a tab the insertion line shows on while dragging. */
function dropSide(input: PaneChromeInput, pane: Pane, key: string): 'before' | 'after' | '' {
  const hint = input.readDropHint();
  if (!input.readDragged() || hint?.pane !== pane.key) return '';
  if (hint.before === key) return 'before';
  const index = pane.tabs.indexOf(key);
  return hint.before === pane.tabs[index + 1] && (hint.before !== undefined || index === pane.tabs.length - 1)
    ? 'after'
    : '';
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
    dragover: (event: DragEvent) => onTabDragOver(input, event, pane, entry.key),
    drop: (event: DragEvent) => onTabDrop(input, event, pane.key, entry.key),
    // Page tabs carry their own close button; docked (native) tabs keep the menu.
    ...(input.binding?.kind === 'page' ? {close: () => input.requestClose(entry.key)} : {}),
    rename: () => input.rename(entry.key),
    dragging: input.readDragged() === entry.key,
    dropSide: dropSide(input, pane, entry.key),
  };
}

/** The "+" after the last tab: everybody reads it as "new tab". */
function renderNewTab(input: PaneChromeInput, pane: Pane, navigation: boolean) {
  if (navigation || input.binding?.kind !== 'page') return '';
  return html`<button
    type="button"
    class="soda-tab-new"
    aria-label="New terminal"
    title="New terminal"
    ?disabled=${!input.available || input.stale || input.creating}
    @click=${() => input.newTerminal(pane.key)}
  >
    <svg width="14" height="14" viewBox="0 0 14 14" aria-hidden="true">
      <path d="M7 1v12M1 7h12" stroke="currentColor" stroke-width="1.8" />
    </svg>
  </button>`;
}

/** Ending a terminal stops its processes, so the tab's × asks once, right under the strip. */
function renderCloseConfirm(input: PaneChromeInput, entries: PaneSession[], navigation: boolean) {
  const key = input.readClosing();
  const item = entries.find((e) => e.entry.key === key);
  if (navigation || !key || !item) return '';
  const name = item.slot ? input.slotName(item.slot) : 'this terminal';
  return html`<div
    class="soda-tab-confirm"
    role="alertdialog"
    aria-label=${'End ' + name}
    @keydown=${(event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        input.cancelClose();
      }
    }}
  >
    <p><strong>End “${name}”?</strong> Commands still running in it will stop. Files are kept.</p>
    <button type="button" class="ui basic button" data-action="cancel" @click=${() => input.cancelClose()}>
      Cancel</button
    ><button type="button" class="ui red button" data-action="end" @click=${() => input.confirmClose()}>
      End terminal
    </button>
  </div>`;
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
  return (item.slot ? input.slotName(item.slot) : unnamed) + ' · ' + input.projectName(item.space);
}

export function renderTabOverflow(input: PaneChromeInput, entries: PaneSession[]) {
  const selected = input.readLayout().entries.find((e) => e.key === focusedSelected(input))?.key;
  return html`<details class="soda-menu soda-tab-overflow" ?hidden=${entries.length < 2}>
    <summary aria-label="Open tabs" title="Open tabs">${icons.tabs}</summary>
    <div>
      <input
        type="search"
        placeholder="Search tabs"
        aria-label="Find an open tab"
        @input=${(e: Event) => filterOverflowTabs(e)}
      />${entries.map(
        (item) =>
          html`<button
            class="ui button soda-menu-item"
            aria-current=${item.entry.key === selected ? 'true' : 'false'}
            title=${overflowTabLabel(input, item)}
            @click=${() => input.openSaved(item.entry)}
          >
            <span class="soda-menu-item-label">${item.slot ? input.slotName(item.slot) : unnamed}</span
            ><span class="soda-menu-item-meta">${input.projectName(item.space)}</span>
          </button>`
      )}
    </div>
  </details>`;
}

function focusedSelected(input: PaneChromeInput) {
  const layout = input.readLayout();
  return panes(layout.tree).find((p) => p.key === layout.focused)?.selected;
}

export function showMoveMenu(input: PaneChromeInput, pane: Pane) {
  return (
    input.binding?.kind === 'page' &&
    !!pane.selected &&
    (panes(input.readLayout().tree).length > 1 || pane.tabs.length > 1)
  );
}

export function moveTargetName(entries: PaneSession[], pane: Pane) {
  return entries.find((item) => item.entry.key === pane.selected)?.slot?.metadata?.name || 'this terminal';
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
  return entries.find((item) => item.entry.key === before)?.slot?.metadata?.name || unnamed;
}

export function renderMoveMenu(input: PaneChromeInput, pane: Pane, entries: PaneSession[]) {
  if (!showMoveMenu(input, pane)) return html``;
  const all = panes(input.readLayout().tree),
    others = all.map((destination, i) => ({destination, i})).filter(({destination}) => destination.key !== pane.key),
    before = pane.tabs.filter((key) => key !== pane.selected);
  return html`<details class="soda-menu soda-move-menu">
    <summary aria-label="Move terminal to pane" title="Move terminal to pane">${icons.move}</summary>
    <div>
      <p class="soda-menu-heading"><span>Move “${moveTargetName(entries, pane)}”</span></p>
      ${others.length ? html`<p class="soda-menu-label">To another pane</p>` : ''}
      ${others.map(({destination, i}) => html`<button class="ui button soda-menu-item" @click=${() => moveToPane(input, pane, destination.key)}>Pane ${i + 1}</button>`)}
      ${before.length ? html`<p class="soda-menu-label">Within this pane <span>Ctrl+Shift+← →</span></p>` : ''}
      ${before.map((key) => html`<button class="ui button soda-menu-item" @click=${() => moveBeforeTab(input, pane, key)}>Before “${beforeTabName(entries, key)}”</button>`)}
      ${before.length ? html`<button class="ui button soda-menu-item" @click=${() => moveToPane(input, pane, pane.key)}>To the end</button>` : ''}
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
      data-drop-end=${input.readDropHint()?.pane === pane.key && input.readDropHint()?.before === undefined && !!input.readDragged() ? 'true' : 'false'}
      @dragover=${(e: DragEvent) => onTabListDragOver(input, e, pane.key)}
      @dragleave=${(e: DragEvent) => {
        if (e.target === e.currentTarget && !(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node))
          input.setDropHint(undefined);
      }}
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
      ${renderNewTab(input, pane, navigation)}
    </div>
    ${renderCloseConfirm(input, entries, navigation)} ${renderFocusedPaneActions(input, pane, navigation)}
    ${renderTabOverflow(input, entries)} ${renderMoveMenu(input, pane, entries)}
    ${renderEmptyPane(input, pane, area, navigation, entries)} ${renderDropEdges(input, pane, area)}
  </section>`;
}
