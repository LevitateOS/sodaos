(() => {
  const panel = document.getElementById('soda-notification-preview');
  if (!panel || typeof panel.showPopover !== 'function') return;
  const content = panel.querySelector<HTMLElement>('.soda-notification-preview-content');
  const status = panel.querySelector<HTMLElement>('.soda-notification-preview-status');
  const retry = panel.querySelector<HTMLButtonElement>('.soda-notification-preview-retry');
  const closeButton = panel.querySelector<HTMLButtonElement>('.soda-notification-preview-close');
  const all = panel.querySelector<HTMLAnchorElement>('.soda-notification-preview-all');
  if (!content || !status || !retry || !closeButton || !all) return;
  const destination = all.href;
  const bells = [...document.querySelectorAll<HTMLAnchorElement>('#navbar a')].filter(link =>
    link.href === destination && link.querySelector('.notification_count'));
  if (!bells.length) return;

  let ready = false;
  let activeBell: HTMLAnchorElement | undefined;
  let activeRequest: XMLHttpRequest | undefined;
  let pending = false;
  // HTMX is bundled privately by Forgejo; use its DOM events, not window.htmx.
  document.addEventListener('htmx:load', event => {
    if (event.detail.elt === document.body || event.detail.elt?.contains(content)) ready = true;
  });

  const invalidate = () => {
    activeRequest = undefined;
    pending = false;
    content.dispatchEvent(new CustomEvent('htmx:abort', { bubbles: true, detail: { elt: content } }));
    content.replaceChildren();
    content.removeAttribute('aria-busy');
    status.textContent = '';
    retry.hidden = true;
    for (const bell of bells) {
      bell.setAttribute('aria-expanded', 'false');
      bell._tippy?.enable();
    }
  };

  const close = (restoreFocus: boolean) => {
    if (panel.matches(':popover-open')) panel.hidePopover();
    invalidate();
    if (restoreFocus) activeBell?.focus();
  };

  const position = () => {
    if (!panel.matches(':popover-open') || !activeBell) return;
    const anchor = activeBell.getBoundingClientRect();
    if (!anchor.width || !anchor.height) { close(false); return; }
    panel.style.left = `${Math.max(16, Math.min(anchor.right - panel.offsetWidth, innerWidth - panel.offsetWidth - 16))}px`;
    panel.style.top = `${Math.max(16, Math.min(anchor.bottom + 8, innerHeight - panel.offsetHeight - 16))}px`;
  };

  const load = () => {
    if (pending || !panel.matches(':popover-open')) return;
    pending = true;
    retry.hidden = true;
    content.replaceChildren();
    content.setAttribute('aria-busy', 'true');
    status.textContent = 'Loading notifications…';
    content.dispatchEvent(new CustomEvent('soda-notification-open', { bubbles: true, detail: {} }));
    // Leave a usable fallback if HTMX did not initialize this target.
    if (!activeRequest) fail();
    position();
  };

  const fail = () => {
    pending = false;
    content.removeAttribute('aria-busy');
    content.replaceChildren();
    status.textContent = 'Could not load notifications. Try again or view all notifications.';
    retry.hidden = false;
    position();
  };

  for (const bell of bells) {
    bell.setAttribute('aria-controls', panel.id);
    bell.setAttribute('aria-expanded', 'false');
    bell.addEventListener('click', event => {
      if (!ready || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      event.preventDefault();
      if (panel.matches(':popover-open') && activeBell === bell) { close(true); return; }
      close(false);
      activeBell = bell;
      panel.showPopover();
      bell.setAttribute('aria-expanded', 'true');
      bell._tippy?.hide();
      bell._tippy?.disable();
      position();
      closeButton.focus({ preventScroll: true });
      load();
    });
  }
  closeButton.addEventListener('click', () => close(true));
  retry.addEventListener('click', load);
  panel.addEventListener('keydown', event => {
    if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close(true); }
  });
  panel.addEventListener('toggle', () => {
    if (!panel.matches(':popover-open')) invalidate();
  });
  // Tab may leave the non-modal panel. Dismiss without stealing the new focus.
  document.addEventListener('focusin', event => {
    if (panel.matches(':popover-open') && !panel.contains(event.target instanceof Node ? event.target : null) && !(event.target instanceof HTMLAnchorElement && bells.includes(event.target))) close(false);
  });
  window.addEventListener('resize', position);
  window.addEventListener('scroll', position, { passive: true });
  window.addEventListener('pagehide', () => close(false));
  window.addEventListener('pageshow', event => { if (event.persisted) close(false); });

  content.addEventListener('htmx:beforeRequest', event => {
    if (!panel.matches(':popover-open')) { event.preventDefault(); return; }
    activeRequest = event.detail.xhr;
  });
  content.addEventListener('htmx:beforeSwap', event => {
    if (event.detail.xhr !== activeRequest || !panel.matches(':popover-open')) {
      event.preventDefault();
      return;
    }
    // Respect Forgejo's HX-Redirect (including expired-session login redirects).
    if (event.detail.xhr.getResponseHeader('HX-Redirect')) return;
    const documentFragment = new DOMParser().parseFromString(event.detail.serverResponse, 'text/html');
    const root = documentFragment.body.firstElementChild;
    if (event.detail.xhr.status !== 200 || documentFragment.body.children.length !== 1 ||
        !root?.matches('.soda-notification-preview-list[data-soda-notification-fragment]') ||
        documentFragment.querySelector('script, #notification_div, #notification_table')) {
      event.preventDefault();
      fail();
    }
  });
  content.addEventListener('htmx:afterSwap', () => {
    pending = false;
    content.removeAttribute('aria-busy');
    status.textContent = '';
    position();
  });
  for (const type of ['htmx:responseError', 'htmx:sendError', 'htmx:timeout'] as const) {
    content.addEventListener(type, event => {
      // Keep failures local; do not let the global toast print response bodies.
      event.stopPropagation();
      if (event.detail.xhr === activeRequest && panel.matches(':popover-open')) fail();
    });
  }
})();
