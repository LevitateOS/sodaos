import {shellPaneHidden, workspaceWidths, type ShellSurface} from './sodaspaces-widths.js';

export type ShellLayout = {
  body: HTMLElement;
  frame: HTMLElement;
  workspace: HTMLElement;
  divider: HTMLElement;
  switcher: HTMLElement;
  forge: HTMLButtonElement;
  terminal: HTMLButtonElement;
  measure: HTMLElement;
  width: number;
  surface: ShellSurface;
};

function cellWidth(measure: HTMLElement) {
  return measure.getBoundingClientRect().width / 16 || 9;
}

function viewportWidth() {
  return window.visualViewport?.width || window.innerWidth;
}

export function applyShellLayout(layout: ShellLayout) {
  const geometry = workspaceWidths(viewportWidth(), Math.max(cellWidth(layout.measure) * 56 + 28, 0), layout.width);
  layout.body.style.setProperty('--soda-space-width', geometry.actual + 'vw');
  layout.body.classList.toggle('sodaspaces-compact', geometry.compact);
  layout.switcher.hidden = !geometry.compact;
  layout.frame.toggleAttribute('data-surface-hidden', shellPaneHidden(geometry.compact, layout.surface, 'forge'));
  layout.workspace.toggleAttribute(
    'data-surface-hidden',
    shellPaneHidden(geometry.compact, layout.surface, 'terminal')
  );
  layout.forge.setAttribute('aria-pressed', String(layout.surface === 'forge'));
  layout.terminal.setAttribute('aria-pressed', String(layout.surface === 'terminal'));
  layout.divider.setAttribute('aria-valuemin', String(Math.round(geometry.minimum * 10) / 10));
  layout.divider.setAttribute('aria-valuemax', String(Math.round(geometry.maximum * 10) / 10));
  layout.divider.setAttribute('aria-valuenow', String(Math.round(geometry.actual * 10) / 10));
}

export function resizeShellWidth(layout: ShellLayout, desired: number) {
  const geometry = workspaceWidths(viewportWidth(), Math.max(cellWidth(layout.measure) * 56 + 28, 0), layout.width);
  if (geometry.compact) return;
  layout.width = Math.max(geometry.minimum, Math.min(geometry.maximum, desired));
  applyShellLayout(layout);
}

export function dividerKeyWidth(current: number, key: string) {
  if (key === 'Home') return 35;
  if (key === 'End') return 65;
  if (key === 'ArrowLeft') return current + 5;
  if (key === 'ArrowRight') return current - 5;
}

function onDividerKey(layout: ShellLayout, event: KeyboardEvent) {
  const next = dividerKeyWidth(layout.width, event.key);
  if (next === undefined) return;
  event.preventDefault();
  resizeShellWidth(layout, next);
}

function onDividerPointer(layout: ShellLayout, event: PointerEvent) {
  if (!layout.divider.hasPointerCapture(event.pointerId)) return;
  const viewport = viewportWidth();
  resizeShellWidth(layout, ((viewport - event.clientX) / Math.max(1, viewport)) * 100);
}

export function bindWorkspaceShellLayout(doc: Document) {
  const layout = readShellLayout(doc);
  if (!layout) return;
  const size = () => applyShellLayout(layout);
  layout.forge.addEventListener('click', () => {
    layout.surface = 'forge';
    size();
  });
  layout.terminal.addEventListener('click', () => {
    layout.surface = 'terminal';
    size();
  });
  layout.divider.addEventListener('keydown', (event) => onDividerKey(layout, event));
  layout.divider.addEventListener('pointerdown', (event) => {
    if (event.button === 0) layout.divider.setPointerCapture(event.pointerId);
  });
  layout.divider.addEventListener('pointermove', (event) => onDividerPointer(layout, event));
  window.addEventListener('resize', size);
  window.visualViewport?.addEventListener('resize', size);
  size();
}

function readShellLayout(doc: Document): ShellLayout | undefined {
  const frame = doc.querySelector<HTMLIFrameElement>('#soda-forgejo-frame');
  const workspace = doc.getElementById('soda-workspace-root');
  const divider = doc.getElementById('soda-workspace-divider');
  const switcher = doc.getElementById('sodaspaces-surfaces');
  const forge = doc.querySelector<HTMLButtonElement>('#soda-surface-forge');
  const terminal = doc.querySelector<HTMLButtonElement>('#soda-surface-terminal');
  const measure = doc.querySelector<HTMLElement>('.sodaspaces-measure');
  if (!frame || !workspace || !divider || !switcher || !forge || !terminal || !measure) return;
  return {body: doc.body, frame, workspace, divider, switcher, forge, terminal, measure, width: 50, surface: 'forge'};
}
