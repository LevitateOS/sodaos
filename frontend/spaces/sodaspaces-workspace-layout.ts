import {html} from 'lit';
import type {Area, DividerArea, Minimum, Pane, Projection, Split, WorkspaceLayout} from './sodaspaces-layout.js';
import {
  consolidate,
  emptyLayout,
  focusedPane,
  layoutLimit,
  moveTab,
  panes,
  parseLayout,
  projectLayout,
  resizeSplit,
  serializeLayout,
  splitPane,
} from './sodaspaces-layout.js';
import type {Slot, WorkspaceContext} from './sodaspaces-workspace-types.js';

export interface LayoutInput {
  binding: WorkspaceContext | undefined;
  readLayout: () => WorkspaceLayout;
  readMaximized: () => string | undefined;
  readWorkspaceWidth: () => number;
  readCanvasSize: () => {width: number; height: number};
  readSlots: () => Slot[];
  readCell: () => {width: number; height: number};
  readTabHeight: () => number;
  readView: () => 'terminal' | 'sessions' | 'project';
  isStale: () => boolean;
  isDisposed: () => boolean;
  readStorageLoaded: () => boolean;
  readStorageKey: () => string;
  hostRect: () => DOMRect;
  canvasRect: () => DOMRect | undefined;
  setLayout: (layout: WorkspaceLayout) => void;
  setMaximized: (key: string | undefined) => void;
  setStorageNotice: (notice: string) => void;
  setStatus: (status: string) => void;
  setStorageLoaded: (loaded: boolean) => void;
  closeMenus: () => void;
  requestUpdate: () => void;
  rectangle: (area: Area) => string;
  capture: (event: PointerEvent) => void;
  releasePointer: (event: PointerEvent) => void;
}

export interface MinimumInput {
  slots: Slot[];
  cell: {width: number; height: number};
  tabHeight: number;
}

export function paneMinimum(input: MinimumInput, pane: Pane): Minimum {
  const min = input.slots.find((s) => s.key === pane.selected)?.minimum;
  return {
    width: min?.width || Math.ceil(input.cell.width * 56 + 24),
    height: (min?.height || Math.ceil(input.cell.height * 12 + 40)) + input.tabHeight,
  };
}

function minimumInput(input: LayoutInput): MinimumInput {
  return {slots: input.readSlots(), cell: input.readCell(), tabHeight: input.readTabHeight()};
}

export function compact(input: LayoutInput) {
  return (
    input.binding?.kind !== 'page' ||
    input.readWorkspaceWidth() <
      (input.readLayout().sidebar || 220) + paneMinimum(minimumInput(input), focusedPane(input.readLayout())).width + 6
  );
}

export function projection(input: LayoutInput): Projection {
  const canvasSize = input.readCanvasSize();
  return projectLayout(
    input.readLayout(),
    {
      x: 0,
      y: 0,
      ...canvasSize,
    },
    (pane) => paneMinimum(minimumInput(input), pane),
    compact(input),
    input.readMaximized()
  );
}

export function toggleSidebar(input: LayoutInput) {
  const layout = input.readLayout();
  input.setLayout({...layout, sidebar: layout.sidebar === null ? 256 : null});
  persist(input);
}

export function loadLayout(input: LayoutInput) {
  input.setStorageLoaded(true);
  try {
    const current = sessionStorage.getItem(input.readStorageKey());
    if (current !== null) input.setLayout(parseLayout(current));
  } catch {
    input.setLayout(emptyLayout(crypto.randomUUID()));
    input.setStorageNotice(
      'Stored layout was reset. Native terminals remain discoverable; no work was created or ended.'
    );
    persist(input);
  }
}

export function persist(input: LayoutInput) {
  if (!input.readStorageLoaded() || input.isStale() || input.isDisposed()) return false;
  try {
    sessionStorage.setItem(input.readStorageKey(), serializeLayout(input.readLayout()));
    return true;
  } catch {
    input.setStorageNotice('Live workspace remains usable, but reload restoration could not be saved.');
  }
  return false;
}

export function arrange(input: LayoutInput, layout: WorkspaceLayout) {
  if (input.isStale() || input.isDisposed()) return;
  input.setLayout(layout);
  if (!panes(layout.tree).some((p) => p.key === input.readMaximized())) input.setMaximized(undefined);
  persist(input);
  input.requestUpdate();
}

export function focusPane(input: LayoutInput, key: string) {
  const layout = input.readLayout();
  if (panes(layout.tree).some((p) => p.key === key)) {
    if (input.readMaximized()) input.setMaximized(key);
    arrange(input, {
      ...layout,
      focused: key,
    });
  }
}

export function splitBlocked(input: LayoutInput) {
  return (
    input.isStale() ||
    input.readView() !== 'terminal' ||
    input.binding?.kind !== 'page' ||
    !!input.readMaximized() ||
    panes(input.readLayout().tree).length >= layoutLimit
  );
}

export function emptyPaneMinimum(input: LayoutInput) {
  return paneMinimum(minimumInput(input), {kind: 'pane', key: '', tabs: [], selected: null});
}

export function splitFits(input: LayoutInput, axis: Split['axis'], area: Area & {pane: Pane}) {
  const min = paneMinimum(minimumInput(input), area.pane),
    empty = emptyPaneMinimum(input);
  if (axis === 'right')
    return area.width >= min.width + empty.width + 6 && area.height >= Math.max(min.height, empty.height);
  return area.height >= min.height + empty.height + 6 && area.width >= Math.max(min.width, empty.width);
}

export function canSplit(input: LayoutInput, axis: Split['axis'], key: string) {
  if (splitBlocked(input)) return false;
  const found = projection(input);
  const area = found.panes.find((a) => a.pane.key === key);
  if (!area || found.compact) return false;
  return splitFits(input, axis, area);
}

export function split(input: LayoutInput, axis: Split['axis']) {
  if (!canSplit(input, axis, input.readLayout().focused)) return;
  input.closeMenus();
  const layout = input.readLayout();
  arrange(input, splitPane(layout, layout.focused, axis, crypto.randomUUID(), crypto.randomUUID()));
}

export function move(input: LayoutInput, key: string, destination: string, before?: string) {
  try {
    arrange(input, moveTab(input.readLayout(), key, destination, before));
  } catch {
    input.setStatus('The pane destination changed; no terminal was replaced.');
  }
}

export interface MaximizedInput {
  closeMenus: () => void;
  readMaximized: () => string | undefined;
  setMaximized: (key: string | undefined) => void;
  readLayout: () => WorkspaceLayout;
  requestUpdate: () => void;
}

export function toggleMaximizedPane(input: MaximizedInput) {
  input.closeMenus();
  input.setMaximized(input.readMaximized() ? undefined : input.readLayout().focused);
  input.requestUpdate();
}

export interface ConsolidateInput {
  closeMenus: () => void;
  arrange: (layout: WorkspaceLayout) => void;
  readLayout: () => WorkspaceLayout;
}

export function consolidatePanes(input: ConsolidateInput) {
  input.closeMenus();
  input.arrange(consolidate(input.readLayout()));
}

export function dragDivider(input: LayoutInput, event: PointerEvent, divider: DividerArea) {
  if (!(event.currentTarget instanceof HTMLElement) || !event.currentTarget.hasPointerCapture(event.pointerId)) return;
  const canvas = input.canvasRect();
  if (!canvas) return;
  adjustDivider(
    input,
    divider,
    ((divider.axis === 'right' ? event.clientX - canvas.x : event.clientY - canvas.y) - divider.origin) / divider.extent
  );
}

export function keyDivider(input: LayoutInput, event: KeyboardEvent, divider: DividerArea) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return;
  event.preventDefault();
  adjustDivider(
    input,
    divider,
    event.key === 'Home'
      ? divider.minimum
      : event.key === 'End'
        ? divider.maximum
        : divider.ratio + (event.key === 'ArrowLeft' || event.key === 'ArrowUp' ? -0.05 : 0.05)
  );
}

export function adjustDivider(input: LayoutInput, divider: DividerArea, ratio: number) {
  const layout = input.readLayout();
  arrange(input, {
    ...layout,
    tree: resizeSplit(layout.tree, divider.key, Math.max(divider.minimum, Math.min(divider.maximum, ratio))),
  });
}

export function dividerPointerMove(input: LayoutInput, event: PointerEvent) {
  const key = (event.currentTarget as HTMLElement).dataset.dividerKey;
  const divider = projection(input).dividers.find((item) => item.key === key);
  if (divider) dragDivider(input, event, divider);
}

export function dividerKey(input: LayoutInput, event: KeyboardEvent) {
  const key = (event.currentTarget as HTMLElement).dataset.dividerKey;
  const divider = projection(input).dividers.find((item) => item.key === key);
  if (divider) keyDivider(input, event, divider);
}

export function renderPaneDivider(
  input: LayoutInput,
  divider: {key: string; axis: string; minimum: number; maximum: number; ratio: number} & Area
) {
  return html`<div
    class="soda-pane-divider"
    role="separator"
    tabindex="0"
    data-divider-key=${divider.key}
    aria-label="Resize panes"
    aria-orientation=${divider.axis === 'right' ? 'vertical' : 'horizontal'}
    aria-valuemin=${Math.ceil(divider.minimum * 100)}
    aria-valuemax=${Math.floor(divider.maximum * 100)}
    aria-valuenow=${Math.round(divider.ratio * 100)}
    style=${input.rectangle(divider)}
    @pointerdown=${(e: PointerEvent) => input.capture(e)}
    @pointermove=${(e: PointerEvent) => dividerPointerMove(input, e)}
    @pointerup=${(e: PointerEvent) => input.releasePointer(e)}
    @keydown=${(e: KeyboardEvent) => dividerKey(input, e)}
  ></div>`;
}

export function resizeSidebar(input: LayoutInput, event: PointerEvent) {
  if (event.currentTarget instanceof HTMLElement && event.currentTarget.hasPointerCapture(event.pointerId))
    setSidebar(input, event.clientX - input.hostRect().left);
}

export function sidebarKey(input: LayoutInput, event: KeyboardEvent) {
  if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) {
    event.preventDefault();
    setSidebar(
      input,
      event.key === 'Home'
        ? 220
        : event.key === 'End'
          ? 360
          : (input.readLayout().sidebar || 256) + (event.key === 'ArrowLeft' ? -10 : 10)
    );
  }
}

export function setSidebar(input: LayoutInput, width: number) {
  const layout = input.readLayout();
  arrange(input, {
    ...layout,
    sidebar: Math.max(220, Math.min(360, Math.round(width))),
  });
}
