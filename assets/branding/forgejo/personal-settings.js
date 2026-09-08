/* Progressive navigation/disclosures only. Forgejo owns all submissions. */
const settings = document.querySelector('.soda-settings');
if (settings) {
  const nav = settings.querySelector('.soda-settings-nav');
  if (nav) {
    const mobile = matchMedia('(max-width: 899px)');
    const current = nav.querySelector('.soda-settings-current');
    const groups = [...nav.querySelectorAll('.soda-settings-nav-group')];
    const active = nav.querySelector('[aria-current="page"]');
    if (active) {
      current.firstChild.textContent = active.textContent.trim() + ' ';
      active.closest('section').classList.add('is-current');
    }
    const setOpen = (trigger, open) => trigger.setAttribute('aria-expanded', String(open));
    const close = () => {
      setOpen(current, false);
      groups.forEach(group => setOpen(group.querySelector('button'), false));
    };
    close();
    nav.classList.add('is-enhanced');
    current.addEventListener('click', () => setOpen(current, current.getAttribute('aria-expanded') !== 'true'));
    groups.forEach(group => {
      const trigger = group.querySelector('button');
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
      const trigger = mobile.matches ? current : event.target.closest('section')?.querySelector('button');
      close(); trigger?.focus(); event.preventDefault();
    });
    document.addEventListener('pointerdown', event => {
      if (!nav.contains(event.target)) {
        const focused = nav.contains(document.activeElement);
        const trigger = mobile.matches ? current : document.activeElement.closest('section')?.querySelector('button');
        close(); if (focused) trigger?.focus();
      }
    });
    // During focusout activeElement can still be body. Use the destination:
    // hiding the menu before the link receives focus cancels its pointer click.
    nav.addEventListener('focusout', event => {
      if (!nav.contains(event.relatedTarget)) close();
    });
    mobile.addEventListener('change', close);
  }
  const hasErrors = settings.querySelector('.ui.error.message, .ui.negative.message, .field.error');
  settings.querySelectorAll('[data-settings-editor]').forEach(editor => {
    // Ambiguous server errors keep every potentially affected editor visible.
    editor.open = Boolean(hasErrors);
  });
  const avatarEditor = settings.querySelector('#avatar-settings');
  const avatarDialog = settings.querySelector('#avatar-dialog');
  const avatarTrigger = settings.querySelector('.soda-avatar-trigger');
  // Server errors retain the expanded inline editor and authoritative page alerts.
  // Without JS (or dialog support), the avatar link reaches the same native form.
  let openAvatar;
  if (avatarEditor && avatarDialog?.showModal && avatarTrigger && !hasErrors) {
    avatarDialog.append(avatarEditor.querySelector('.soda-settings-disclosure-body'));
    avatarEditor.hidden = true;
    avatarTrigger.setAttribute('role', 'button');
    avatarTrigger.setAttribute('aria-haspopup', 'dialog');
    avatarTrigger.setAttribute('aria-controls', 'avatar-dialog');
    openAvatar = () => { if (!avatarDialog.open) avatarDialog.showModal(); };
    avatarTrigger.addEventListener('click', event => { event.preventDefault(); openAvatar(); });
    avatarTrigger.addEventListener('keydown', event => {
      if (event.key === ' ') { event.preventDefault(); openAvatar(); }
    });
    avatarDialog.querySelector('.soda-avatar-close').addEventListener('click', () => avatarDialog.close());
    avatarDialog.addEventListener('click', event => {
      const rect = avatarDialog.getBoundingClientRect();
      if (event.target === avatarDialog && (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom)) avatarDialog.close();
    });
    avatarDialog.addEventListener('keydown', event => {
      if (event.key !== 'Tab') return;
      const controls = [...avatarDialog.querySelectorAll('button, input:not([type=hidden]), a[href], select, textarea')].filter(el => !el.disabled && el.tabIndex >= 0 && el.checkVisibility());
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
    if (openAvatar && (target === avatarEditor || avatarDialog.contains(target))) { openAvatar(); return; }
    const editor = target?.closest('[data-settings-editor]');
    if (editor) {
      editor.open = true; editor.scrollIntoView({block: 'start'});
      editor.querySelector('input:not([type=hidden]), textarea')?.focus({preventScroll: true});
    }
  };
  revealHash();
  window.addEventListener('hashchange', revealHash);
  if (hasErrors) {
    const field = settings.querySelector('.field.error input:not([type=hidden]), .field.error textarea');
    field?.focus();
  }
}
// Native show/hide-panel handlers retain ownership; supplement keyboard focus.
if (settings) {
  settings.addEventListener('click', event => {
    const button = event.target.closest('button[data-panel]');
    if (!button) return;
    requestAnimationFrame(() => {
      const panel = settings.querySelector(button.dataset.panel);
      if (button.classList.contains('show-panel') && panel?.checkVisibility()) {
        panel.querySelector('input:not([type=hidden]), textarea')?.focus();
      } else if (button.classList.contains('hide-panel')) {
        [...settings.querySelectorAll('button.show-panel[data-panel]')].find(trigger => trigger.dataset.panel === button.dataset.panel)?.focus();
      }
    });
  });
}
