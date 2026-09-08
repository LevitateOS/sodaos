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
    nav.addEventListener('focusout', () => queueMicrotask(() => { if (!nav.contains(document.activeElement)) close(); }));
    mobile.addEventListener('change', close);
  }
  const hasErrors = settings.querySelector('.ui.error.message, .ui.negative.message, .field.error');
  settings.querySelectorAll('[data-settings-editor]').forEach(editor => {
    // Ambiguous server errors keep every potentially affected editor visible.
    editor.open = Boolean(hasErrors);
  });
  const revealHash = () => {
    if (!location.hash) return;
    let target;
    try { target = document.getElementById(decodeURIComponent(location.hash.slice(1))); } catch { return; }
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
