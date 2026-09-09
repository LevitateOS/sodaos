/* Progressive navigation/disclosures only. Forgejo owns all submissions. */
const settings = document.querySelector<HTMLElement>('.soda-settings-shell');
if (settings) {
  const nav = settings.querySelector<HTMLElement>('.soda-settings-nav');
  const current = nav?.querySelector<HTMLButtonElement>('.soda-settings-current');
  if (nav && current) {
    const mobile = matchMedia('(max-width: 899px)');
    const groups = [...nav.querySelectorAll('.soda-settings-nav-group')];
    const active = nav.querySelector('[aria-current="page"]');
    if (active) {
      if (current.firstChild) current.firstChild.textContent = (active.textContent || '').trim() + ' ';
      active.closest('section')?.classList.add('is-current');
    }
    const setOpen = (trigger: Element | null, open: boolean) => trigger?.setAttribute('aria-expanded', String(open));
    const close = () => {
      setOpen(current, false);
      groups.forEach(group => setOpen(group.querySelector('button'), false));
    };
    close();
    nav.classList.add('is-enhanced');
    current.addEventListener('click', () => setOpen(current, current.getAttribute('aria-expanded') !== 'true'));
    groups.forEach(group => {
      const trigger = group.querySelector('button');
      if (!trigger) return;
      trigger.addEventListener('click', () => {
        const open = trigger.getAttribute('aria-expanded') !== 'true';
        close();
        setOpen(trigger, open);
      });
      trigger.addEventListener('keydown', event => {
        if (event.key === 'ArrowDown') {
          event.preventDefault(); close(); setOpen(trigger, true);
          group.querySelector('a')?.focus();
        }
      });
    });
    nav.addEventListener('keydown', event => {
      if (event.key !== 'Escape') return;
      const trigger = mobile.matches ? current : (event.target instanceof Element ? event.target.closest('section') : null)?.querySelector('button');
      close(); trigger?.focus(); event.preventDefault();
    });
    document.addEventListener('pointerdown', event => {
      if (!nav.contains(event.target instanceof Node ? event.target : null)) {
        const focused = nav.contains(document.activeElement);
        const trigger = mobile.matches ? current : document.activeElement?.closest('section')?.querySelector('button');
        close(); if (focused) trigger?.focus();
      }
    });
    // During focusout activeElement can still be body. Use the destination:
    // hiding the menu before the link receives focus cancels its pointer click.
    nav.addEventListener('focusout', event => {
      if (!nav.contains(event.relatedTarget instanceof Node ? event.relatedTarget : null)) close();
    });
    mobile.addEventListener('change', close);
  }
  const hasErrors = settings.querySelector('.ui.error.message, .ui.negative.message, .field.error');
  settings.querySelectorAll<HTMLDetailsElement>('[data-settings-editor]').forEach(editor => {
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
  if (avatarEditor && avatarDialog && typeof avatarDialog.showModal === "function" && avatarTrigger && avatarBody && !hasErrors) {
    avatarDialog.append(avatarBody);
    avatarEditor.hidden = true;
    avatarTrigger.setAttribute('role', 'button');
    avatarTrigger.setAttribute('aria-haspopup', 'dialog');
    avatarTrigger.setAttribute('aria-controls', 'avatar-dialog');
    openAvatar = () => { if (!avatarDialog.open) avatarDialog.showModal(); };
    avatarTrigger.addEventListener('click', event => { event.preventDefault(); openAvatar?.(); });
    avatarTrigger.addEventListener('keydown', event => {
      if (event.key === ' ') { event.preventDefault(); openAvatar?.(); }
    });
    avatarDialog.querySelector('.soda-avatar-close')?.addEventListener('click', () => avatarDialog.close());
    avatarDialog.addEventListener('click', event => {
      const rect = avatarDialog.getBoundingClientRect();
      if (event.target === avatarDialog && (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom)) avatarDialog.close();
    });
    avatarDialog.addEventListener('keydown', event => {
      if (event.key !== 'Tab') return;
      const controls = [...avatarDialog.querySelectorAll<HTMLElement>('button, input:not([type=hidden]), a[href], select, textarea')].filter(el => !el.matches(':disabled') && el.tabIndex >= 0 && el.checkVisibility());
      const first = controls[0], last = controls.at(-1);
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
    });
    avatarDialog.addEventListener('close', () => avatarTrigger.focus({preventScroll: true}));
  }
  const revealHash = () => {
    if (!location.hash) return;
    let target;
    try { target = document.getElementById(decodeURIComponent(location.hash.slice(1))); } catch { return; }
    if (openAvatar && (target === avatarEditor || avatarDialog?.contains(target))) { openAvatar(); return; }
    const editor = target?.closest<HTMLDetailsElement>('[data-settings-editor]');
    if (editor) {
      editor.open = true; editor.scrollIntoView({block: 'start'});
      editor.querySelector<HTMLElement>('input:not([type=hidden]), textarea')?.focus({preventScroll: true});
    }
  };
  revealHash();
  window.addEventListener('hashchange', revealHash);
  if (hasErrors) {
    const field = settings.querySelector<HTMLElement>('.field.error input:not([type=hidden]), .field.error textarea');
    field?.focus();
  }
}
// Native show/hide-panel handlers retain ownership; supplement keyboard focus.
if (settings) {
  settings.addEventListener('click', event => {
    const button = event.target instanceof Element ? event.target.closest<HTMLButtonElement>('button[data-panel]') : null;
    if (!button?.dataset.panel) return;
    const selector = button.dataset.panel;
    requestAnimationFrame(() => {
      const panel = settings.querySelector(selector);
      if (button.classList.contains('show-panel') && panel?.checkVisibility()) {
        panel.querySelector<HTMLElement>('input:not([type=hidden]), textarea')?.focus();
      } else if (button.classList.contains('hide-panel')) {
        [...settings.querySelectorAll<HTMLButtonElement>('button.show-panel[data-panel]')].find(trigger => trigger.dataset.panel === button.dataset.panel)?.focus();
      }
    });
  });
}
