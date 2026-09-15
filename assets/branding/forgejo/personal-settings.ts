/* Progressive navigation/disclosures only. Forgejo owns all submissions. */

function setNavOpen(trigger: Element | null, open: boolean): void {
  trigger?.setAttribute('aria-expanded', String(open));
}

function closeSettingsNav(current: HTMLButtonElement, groups: Element[]): void {
  setNavOpen(current, false);
  groups.forEach((group) => setNavOpen(group.querySelector('button'), false));
}

function navTriggerForEscape(
  current: HTMLButtonElement,
  mobile: MediaQueryList,
  event: KeyboardEvent
): HTMLElement | null | undefined {
  if (mobile.matches) return current;
  const section = event.target instanceof Element ? event.target.closest('section') : null;
  return section?.querySelector('button');
}

function navTriggerForPointer(current: HTMLButtonElement, mobile: MediaQueryList): HTMLElement | null | undefined {
  if (mobile.matches) return current;
  return document.activeElement?.closest('section')?.querySelector('button');
}

function eventTargetNode(target: EventTarget | null): Node | null {
  return target instanceof Node ? target : null;
}

function enhanceSettingsNav(settings: HTMLElement): void {
  const nav = settings.querySelector<HTMLElement>('.soda-settings-nav');
  const current = nav?.querySelector<HTMLButtonElement>('.soda-settings-current');
  if (!nav || !current) return;
  const mobile = matchMedia('(max-width: 899px)');
  const groups = [...nav.querySelectorAll('.soda-settings-nav-group')];
  const active = nav.querySelector('[aria-current="page"]');
  if (active) {
    if (current.firstChild) current.firstChild.textContent = (active.textContent || '').trim() + ' ';
    active.closest('section')?.classList.add('is-current');
  }
  const close = () => closeSettingsNav(current, groups);
  close();
  nav.classList.add('is-enhanced');
  current.addEventListener('click', () => setNavOpen(current, current.getAttribute('aria-expanded') !== 'true'));
  groups.forEach((group) => {
    const trigger = group.querySelector('button');
    if (!trigger) return;
    trigger.addEventListener('click', () => {
      const open = trigger.getAttribute('aria-expanded') !== 'true';
      close();
      setNavOpen(trigger, open);
    });
    trigger.addEventListener('keydown', (event) => {
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        close();
        setNavOpen(trigger, true);
        group.querySelector('a')?.focus();
      }
    });
  });
  nav.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape') return;
    const trigger = navTriggerForEscape(current, mobile, event);
    close();
    trigger?.focus();
    event.preventDefault();
  });
  document.addEventListener('pointerdown', (event) => {
    if (nav.contains(eventTargetNode(event.target))) return;
    const focused = nav.contains(document.activeElement);
    const trigger = navTriggerForPointer(current, mobile);
    close();
    if (focused) trigger?.focus();
  });
  // During focusout activeElement can still be body. Use the destination:
  // hiding the menu before the link receives focus cancels its pointer click.
  nav.addEventListener('focusout', (event) => {
    if (!nav.contains(eventTargetNode(event.relatedTarget))) close();
  });
  mobile.addEventListener('change', close);
}

function clickClosesAvatarDialog(dialog: HTMLDialogElement, event: MouseEvent): boolean {
  if (event.target !== dialog) return false;
  const rect = dialog.getBoundingClientRect();
  return (
    event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom
  );
}

function trapAvatarDialogTab(dialog: HTMLDialogElement, event: KeyboardEvent): void {
  if (event.key !== 'Tab') return;
  const controls = [
    ...dialog.querySelectorAll<HTMLElement>('button, input:not([type=hidden]), a[href], select, textarea'),
  ].filter((el) => !el.matches(':disabled') && el.tabIndex >= 0 && el.checkVisibility());
  const first = controls[0],
    last = controls.at(-1);
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last?.focus();
    return;
  }
  if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first?.focus();
  }
}

function enhanceAvatarDialog(
  avatarEditor: HTMLDetailsElement,
  avatarDialog: HTMLDialogElement,
  avatarTrigger: HTMLElement,
  avatarBody: HTMLElement
): () => void {
  avatarDialog.append(avatarBody);
  avatarEditor.hidden = true;
  avatarTrigger.setAttribute('role', 'button');
  avatarTrigger.setAttribute('aria-haspopup', 'dialog');
  avatarTrigger.setAttribute('aria-controls', 'avatar-dialog');
  const openAvatar = () => {
    if (!avatarDialog.open) avatarDialog.showModal();
  };
  avatarTrigger.addEventListener('click', (event) => {
    event.preventDefault();
    openAvatar();
  });
  avatarTrigger.addEventListener('keydown', (event) => {
    if (event.key === ' ') {
      event.preventDefault();
      openAvatar();
    }
  });
  avatarDialog.querySelector('.soda-avatar-close')?.addEventListener('click', () => avatarDialog.close());
  avatarDialog.addEventListener('click', (event) => {
    if (clickClosesAvatarDialog(avatarDialog, event)) avatarDialog.close();
  });
  avatarDialog.addEventListener('keydown', (event) => {
    trapAvatarDialogTab(avatarDialog, event);
  });
  avatarDialog.addEventListener('close', () => avatarTrigger.focus({preventScroll: true}));
  return openAvatar;
}

function elementFromLocationHash(): HTMLElement | null | undefined {
  if (!location.hash) return undefined;
  try {
    return document.getElementById(decodeURIComponent(location.hash.slice(1)));
  } catch {
    return undefined;
  }
}

function shouldOpenAvatarFromHash(
  openAvatar: (() => void) | undefined,
  avatarEditor: HTMLDetailsElement | null,
  avatarDialog: HTMLDialogElement | null,
  target: HTMLElement
): boolean {
  if (!openAvatar) return false;
  return target === avatarEditor || Boolean(avatarDialog?.contains(target));
}

function revealSettingsEditor(target: HTMLElement): void {
  const editor = target.closest<HTMLDetailsElement>('[data-settings-editor]');
  if (!editor) return;
  editor.open = true;
  editor.scrollIntoView({block: 'start'});
  editor.querySelector<HTMLElement>('input:not([type=hidden]), textarea')?.focus({preventScroll: true});
}

function revealSettingsHash(
  openAvatar: (() => void) | undefined,
  avatarEditor: HTMLDetailsElement | null,
  avatarDialog: HTMLDialogElement | null
): void {
  const target = elementFromLocationHash();
  if (!target) return;
  if (openAvatar && shouldOpenAvatarFromHash(openAvatar, avatarEditor, avatarDialog, target)) {
    openAvatar();
    return;
  }
  revealSettingsEditor(target);
}

function focusVisiblePanel(settings: HTMLElement, button: HTMLButtonElement, selector: string): void {
  const panel = settings.querySelector(selector);
  if (button.classList.contains('show-panel') && panel?.checkVisibility()) {
    panel.querySelector<HTMLElement>('input:not([type=hidden]), textarea')?.focus();
    return;
  }
  if (button.classList.contains('hide-panel')) {
    [...settings.querySelectorAll<HTMLButtonElement>('button.show-panel[data-panel]')]
      .find((trigger) => trigger.dataset.panel === button.dataset.panel)
      ?.focus();
  }
}

function enhanceSettingsPanels(settings: HTMLElement): void {
  settings.addEventListener('click', (event) => {
    const button =
      event.target instanceof Element ? event.target.closest<HTMLButtonElement>('button[data-panel]') : null;
    if (!button?.dataset.panel) return;
    const selector = button.dataset.panel;
    requestAnimationFrame(() => {
      focusVisiblePanel(settings, button, selector);
    });
  });
}

const settings = document.querySelector<HTMLElement>('.soda-settings-shell');
if (settings) {
  enhanceSettingsNav(settings);
  const hasErrors = settings.querySelector('.ui.error.message, .ui.negative.message, .field.error');
  settings.querySelectorAll<HTMLDetailsElement>('[data-settings-editor]').forEach((editor) => {
    // Ambiguous server errors keep every potentially affected editor visible.
    editor.open = Boolean(hasErrors);
  });
  const avatarEditor = settings.querySelector<HTMLDetailsElement>('#avatar-settings');
  const avatarDialog = settings.querySelector<HTMLDialogElement>('#avatar-dialog');
  const avatarTrigger = settings.querySelector<HTMLElement>('.soda-avatar-trigger');
  // Server errors retain the expanded inline editor and authoritative page alerts.
  // Without JS (or dialog support), the avatar link reaches the same native form.
  let openAvatar: (() => void) | undefined;
  const avatarBody = avatarEditor?.querySelector<HTMLElement>('.soda-settings-disclosure-body');
  if (
    avatarEditor &&
    avatarDialog &&
    typeof avatarDialog.showModal === 'function' &&
    avatarTrigger &&
    avatarBody &&
    !hasErrors
  ) {
    openAvatar = enhanceAvatarDialog(avatarEditor, avatarDialog, avatarTrigger, avatarBody);
  }
  const revealHash = () => revealSettingsHash(openAvatar, avatarEditor, avatarDialog);
  revealHash();
  window.addEventListener('hashchange', revealHash);
  if (hasErrors) {
    const field = settings.querySelector<HTMLElement>('.field.error input:not([type=hidden]), .field.error textarea');
    field?.focus();
  }
}
// Native show/hide-panel handlers retain ownership; supplement keyboard focus.
if (settings) enhanceSettingsPanels(settings);
