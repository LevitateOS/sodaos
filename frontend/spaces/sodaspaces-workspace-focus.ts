export interface WorkspaceKeyInput {
  isEditing: () => boolean;
  clearEditing: () => void;
  restoreFocus: () => void;
}

export function workspaceKey(input: WorkspaceKeyInput, event: KeyboardEvent) {
  if (event.key !== 'Escape' || !(event.target instanceof HTMLElement)) return;
  const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
  if (menu) {
    event.preventDefault();
    event.stopPropagation();
    menu.open = false;
    menu.querySelector<HTMLElement>('summary')?.focus();
  } else if (input.isEditing() && event.target.closest('.soda-workspace-dialog')) {
    event.preventDefault();
    event.stopPropagation();
    input.clearEditing();
    input.restoreFocus();
  }
}

export function workspaceClick(root: HTMLElement, event: MouseEvent) {
  if (!(event.target instanceof Element)) return;
  const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
  if (!menu) {
    closeMenus(root);
    return;
  }
  if (!event.target.closest('summary')) return;
  for (const other of root.querySelectorAll<HTMLDetailsElement>('.soda-menu[open]'))
    if (other !== menu) other.open = false;
  const summary = menu.querySelector('summary');
  const boundary = menu.closest('.soda-workspace-terminal, .soda-workspace-canvas') || root;
  if (summary)
    menu.style.setProperty(
      '--soda-menu-available-height',
      Math.max(
        0,
        Math.min(window.innerHeight, boundary.getBoundingClientRect().bottom) -
          summary.getBoundingClientRect().bottom -
          2
      ) + 'px'
    );
}

export function menuFocusOut(event: FocusEvent) {
  if (!(event.target instanceof Element) || !(event.relatedTarget instanceof Node)) return;
  const menu = event.target.closest<HTMLDetailsElement>('.soda-menu');
  if (menu && !menu.contains(event.relatedTarget)) menu.open = false;
}

export function closeMenus(root: HTMLElement) {
  for (const menu of root.querySelectorAll<HTMLDetailsElement>('.soda-menu')) menu.open = false;
}

export interface RememberFocusInput {
  root: HTMLElement;
  setInvoker: (invoker: HTMLElement | undefined) => void;
}

export function rememberFocus(input: RememberFocusInput) {
  input.setInvoker(document.activeElement instanceof HTMLElement ? document.activeElement : undefined);
  closeMenus(input.root);
}

export interface RestoreFocusInput {
  invoker: HTMLElement | undefined;
  updateComplete: Promise<boolean>;
  isStale: () => boolean;
  isActiveSurface: () => boolean;
  root: HTMLElement;
}

export function restoreFocus(input: RestoreFocusInput) {
  const target = input.invoker,
    active = document.activeElement;
  void input.updateComplete.then(() => {
    if (
      !input.isStale() &&
      input.isActiveSurface() &&
      (document.activeElement === active || document.activeElement === document.body)
    ) {
      const visible = (node: HTMLElement) =>
        node.isConnected && !node.closest('[hidden], [inert]') && node.getClientRects().length > 0;
      const next =
        target && visible(target)
          ? target
          : [
              ...input.root.querySelectorAll<HTMLElement>(
                '.soda-setup-welcome .primary, .soda-workspace-intro h2, .soda-workspace-toolbar button'
              ),
            ].find(visible);
      next?.focus();
    }
  });
}
