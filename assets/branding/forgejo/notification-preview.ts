type NotificationPreview = {
  panel: HTMLElement;
  content: HTMLElement;
  status: HTMLElement;
  retry: HTMLButtonElement;
  closeButton: HTMLButtonElement;
  bells: HTMLAnchorElement[];
  ready: boolean;
  activeBell: HTMLAnchorElement | undefined;
  activeRequest: XMLHttpRequest | undefined;
  pending: boolean;
};

function isNotificationBell(destination: string, link: HTMLAnchorElement): boolean {
  return link.href === destination && Boolean(link.querySelector('.notification_count'));
}

function notificationBells(destination: string): HTMLAnchorElement[] {
  return [...document.querySelectorAll<HTMLAnchorElement>('#navbar a')].filter((link) =>
    isNotificationBell(destination, link)
  );
}

function queryPreviewParts(panel: HTMLElement) {
  const content = panel.querySelector<HTMLElement>('.soda-notification-preview-content');
  const status = panel.querySelector<HTMLElement>('.soda-notification-preview-status');
  const retry = panel.querySelector<HTMLButtonElement>('.soda-notification-preview-retry');
  const closeButton = panel.querySelector<HTMLButtonElement>('.soda-notification-preview-close');
  const all = panel.querySelector<HTMLAnchorElement>('.soda-notification-preview-all');
  if (!content || !status || !retry || !closeButton || !all) return undefined;
  return {content, status, retry, closeButton, all};
}

function invalidatePreview(preview: NotificationPreview): void {
  preview.activeRequest = undefined;
  preview.pending = false;
  preview.content.dispatchEvent(new CustomEvent('htmx:abort', {bubbles: true, detail: {elt: preview.content}}));
  preview.content.replaceChildren();
  preview.content.removeAttribute('aria-busy');
  preview.status.textContent = '';
  preview.retry.hidden = true;
  for (const bell of preview.bells) {
    bell.setAttribute('aria-expanded', 'false');
    bell._tippy?.enable();
  }
}

function closePreview(preview: NotificationPreview, restoreFocus: boolean): void {
  if (preview.panel.matches(':popover-open')) preview.panel.hidePopover();
  invalidatePreview(preview);
  if (restoreFocus) preview.activeBell?.focus();
}

function positionPreview(preview: NotificationPreview): void {
  if (!preview.panel.matches(':popover-open') || !preview.activeBell) return;
  const anchor = preview.activeBell.getBoundingClientRect();
  if (!anchor.width || !anchor.height) {
    closePreview(preview, false);
    return;
  }
  preview.panel.style.left = `${Math.max(16, Math.min(anchor.right - preview.panel.offsetWidth, innerWidth - preview.panel.offsetWidth - 16))}px`;
  preview.panel.style.top = `${Math.max(16, Math.min(anchor.bottom + 8, innerHeight - preview.panel.offsetHeight - 16))}px`;
}

function failPreview(preview: NotificationPreview): void {
  preview.pending = false;
  preview.content.removeAttribute('aria-busy');
  preview.content.replaceChildren();
  preview.status.textContent = 'Could not load notifications. Try again or view all notifications.';
  preview.retry.hidden = false;
  positionPreview(preview);
}

function loadPreview(preview: NotificationPreview): void {
  if (preview.pending || !preview.panel.matches(':popover-open')) return;
  preview.pending = true;
  preview.retry.hidden = true;
  preview.content.replaceChildren();
  preview.content.setAttribute('aria-busy', 'true');
  preview.status.textContent = 'Loading notifications…';
  preview.content.dispatchEvent(new CustomEvent('soda-notification-open', {bubbles: true, detail: {}}));
  // Leave a usable fallback if HTMX did not initialize this target.
  if (!preview.activeRequest) failPreview(preview);
  positionPreview(preview);
}

function isPlainBellActivation(event: MouseEvent): boolean {
  return (
    !event.defaultPrevented &&
    event.button === 0 &&
    !event.metaKey &&
    !event.ctrlKey &&
    !event.shiftKey &&
    !event.altKey
  );
}

function hideBellTooltip(bell: HTMLAnchorElement): void {
  bell._tippy?.hide();
  bell._tippy?.disable();
}

function openBellPreview(preview: NotificationPreview, bell: HTMLAnchorElement): void {
  closePreview(preview, false);
  preview.activeBell = bell;
  preview.panel.showPopover();
  bell.setAttribute('aria-expanded', 'true');
  hideBellTooltip(bell);
  positionPreview(preview);
  preview.closeButton.focus({preventScroll: true});
  loadPreview(preview);
}

function onBellClick(preview: NotificationPreview, bell: HTMLAnchorElement, event: MouseEvent): void {
  if (!preview.ready || !isPlainBellActivation(event)) return;
  event.preventDefault();
  if (preview.panel.matches(':popover-open') && preview.activeBell === bell) {
    closePreview(preview, true);
    return;
  }
  openBellPreview(preview, bell);
}

function parseNotificationFragment(serverResponse: string): boolean {
  const documentFragment = new DOMParser().parseFromString(serverResponse, 'text/html');
  const root = documentFragment.body.firstElementChild;
  if (documentFragment.body.children.length !== 1) return false;
  if (!root?.matches('.soda-notification-preview-list[data-soda-notification-fragment]')) return false;
  return !documentFragment.querySelector('script, #notification_div, #notification_table');
}

function admitNotificationSwap(
  preview: NotificationPreview,
  event: CustomEvent<{xhr: XMLHttpRequest; serverResponse: string}>
): void {
  if (event.detail.xhr !== preview.activeRequest || !preview.panel.matches(':popover-open')) {
    event.preventDefault();
    return;
  }
  // Respect Forgejo's HX-Redirect (including expired-session login redirects).
  if (event.detail.xhr.getResponseHeader('HX-Redirect')) return;
  if (event.detail.xhr.status !== 200 || !parseNotificationFragment(event.detail.serverResponse)) {
    event.preventDefault();
    failPreview(preview);
  }
}

function markPreviewReady(preview: NotificationPreview, elt: Element | undefined): void {
  if (elt === document.body || elt?.contains(preview.content)) preview.ready = true;
}

function focusIsOutsidePreview(preview: NotificationPreview, target: EventTarget | null): boolean {
  if (!(target instanceof Node) || preview.panel.contains(target)) return false;
  return !(target instanceof HTMLAnchorElement && preview.bells.includes(target));
}

function onPreviewHtmxError(preview: NotificationPreview, event: CustomEvent<{xhr: XMLHttpRequest}>): void {
  // Keep failures local; do not let the global toast print response bodies.
  event.stopPropagation();
  if (event.detail.xhr === preview.activeRequest && preview.panel.matches(':popover-open')) failPreview(preview);
}

function bindBellClicks(preview: NotificationPreview): void {
  for (const bell of preview.bells) {
    bell.setAttribute('aria-controls', preview.panel.id);
    bell.setAttribute('aria-expanded', 'false');
    bell.addEventListener('click', (event) => {
      onBellClick(preview, bell, event);
    });
  }
}

function onPreviewEscape(preview: NotificationPreview, event: KeyboardEvent): void {
  if (event.key !== 'Escape') return;
  event.preventDefault();
  event.stopPropagation();
  closePreview(preview, true);
}

function onPreviewFocusIn(preview: NotificationPreview, event: FocusEvent): void {
  if (preview.panel.matches(':popover-open') && focusIsOutsidePreview(preview, event.target)) {
    closePreview(preview, false);
  }
}

function onPreviewBeforeRequest(preview: NotificationPreview, event: CustomEvent<{xhr: XMLHttpRequest}>): void {
  if (!preview.panel.matches(':popover-open')) {
    event.preventDefault();
    return;
  }
  preview.activeRequest = event.detail.xhr;
}

function bindPreviewWindow(preview: NotificationPreview): void {
  window.addEventListener('resize', () => positionPreview(preview));
  window.addEventListener('scroll', () => positionPreview(preview), {passive: true});
  window.addEventListener('pagehide', () => closePreview(preview, false));
  window.addEventListener('pageshow', (event) => {
    if (event.persisted) closePreview(preview, false);
  });
}

function bindPreviewHtmx(preview: NotificationPreview): void {
  preview.content.addEventListener('htmx:beforeRequest', (event) => {
    onPreviewBeforeRequest(preview, event);
  });
  preview.content.addEventListener('htmx:beforeSwap', (event) => {
    admitNotificationSwap(preview, event);
  });
  preview.content.addEventListener('htmx:afterSwap', () => {
    preview.pending = false;
    preview.content.removeAttribute('aria-busy');
    preview.status.textContent = '';
    positionPreview(preview);
  });
  for (const type of ['htmx:responseError', 'htmx:sendError', 'htmx:timeout'] as const) {
    preview.content.addEventListener(type, (event) => {
      onPreviewHtmxError(preview, event);
    });
  }
}

function bindPreviewEvents(preview: NotificationPreview): void {
  document.addEventListener('htmx:load', (event) => {
    markPreviewReady(preview, event.detail.elt);
  });
  bindBellClicks(preview);
  preview.closeButton.addEventListener('click', () => closePreview(preview, true));
  preview.retry.addEventListener('click', () => loadPreview(preview));
  preview.panel.addEventListener('keydown', (event) => {
    onPreviewEscape(preview, event);
  });
  preview.panel.addEventListener('toggle', () => {
    if (!preview.panel.matches(':popover-open')) invalidatePreview(preview);
  });
  // Tab may leave the non-modal panel. Dismiss without stealing the new focus.
  document.addEventListener('focusin', (event) => {
    onPreviewFocusIn(preview, event);
  });
  bindPreviewWindow(preview);
  bindPreviewHtmx(preview);
}

function enhanceNotificationPreview(): void {
  const panel = document.getElementById('soda-notification-preview');
  if (!panel || typeof panel.showPopover !== 'function') return;
  const parts = queryPreviewParts(panel);
  if (!parts) return;
  const bells = notificationBells(parts.all.href);
  if (!bells.length) return;
  bindPreviewEvents({
    panel,
    content: parts.content,
    status: parts.status,
    retry: parts.retry,
    closeButton: parts.closeButton,
    bells,
    ready: false,
    activeBell: undefined,
    activeRequest: undefined,
    pending: false,
  });
}

enhanceNotificationPreview();
