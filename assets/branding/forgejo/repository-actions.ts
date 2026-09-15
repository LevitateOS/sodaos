function actionIsAvailable(action: HTMLElement): boolean {
  return (
    !action.matches(':disabled, [aria-disabled="true"]') && !action.closest('.disabled, .ui.modal') && !action.hidden
  );
}

function firstAvailableAction(panel: HTMLElement): HTMLElement | undefined {
  return [...panel.querySelectorAll<HTMLElement>('a, button')].find(actionIsAvailable);
}

function compactActionsMode(summary: HTMLElement): boolean {
  return getComputedStyle(summary).display !== 'none';
}

function expandActionsWide(
  details: HTMLDetailsElement,
  summary: HTMLElement,
  panel: HTMLElement,
  summaryHadFocus: {value: boolean}
): void {
  const transferFocus = summaryHadFocus.value || document.activeElement === summary;
  summaryHadFocus.value = false;
  details.open = true;
  if (transferFocus) firstAvailableAction(panel)?.focus({preventScroll: true});
}

function collapseActionsCompact(
  details: HTMLDetailsElement,
  panel: HTMLElement,
  initial: boolean,
  enteredCompact: boolean
): void {
  if (initial || (enteredCompact && !panel.contains(document.activeElement))) details.open = false;
}

function syncActionsMode(
  details: HTMLDetailsElement,
  summary: HTMLElement,
  panel: HTMLElement,
  state: {compact: boolean; summaryHadFocus: {value: boolean}},
  initial = false
): void {
  const nextCompact = compactActionsMode(summary);
  const enteredCompact = !state.compact && nextCompact;
  state.compact = nextCompact;
  if (!state.compact) {
    expandActionsWide(details, summary, panel, state.summaryHadFocus);
    return;
  }
  collapseActionsCompact(details, panel, initial, enteredCompact);
}

function onActionsEscape(details: HTMLDetailsElement, summary: HTMLElement, event: KeyboardEvent): void {
  if (!details.open || event.key !== 'Escape' || event.defaultPrevented || !compactActionsMode(summary)) return;
  if (event.target instanceof Element && event.target.closest('.ui.modal')) return;
  event.preventDefault();
  event.stopPropagation();
  details.open = false;
  summary.focus({preventScroll: true});
}

function dismissOpenActions(details: HTMLDetailsElement, summary: HTMLElement, target: EventTarget | null): void {
  if (!details.open || !(target instanceof Node) || details.contains(target) || !compactActionsMode(summary)) return;
  details.open = false;
}

function dismissActionsOnFocusOut(details: HTMLDetailsElement, summary: HTMLElement, next: EventTarget | null): void {
  if (
    !details.open ||
    next === null ||
    (next instanceof Node && details.contains(next)) ||
    !compactActionsMode(summary)
  ) {
    return;
  }
  details.open = false;
}

function enhanceRepositoryActions(details: HTMLDetailsElement): void {
  if (details.dataset.repositoryActionsEnhanced) return;
  const summary = details.querySelector<HTMLElement>(':scope > summary');
  const panel = details.querySelector<HTMLElement>(':scope > .soda-repository-action-menu');
  if (!summary || !panel) return;
  details.dataset.repositoryActionsEnhanced = 'true';
  const state = {compact: false, summaryHadFocus: {value: false}};
  // Hiding summary can blur it before ResizeObserver delivers the new mode.
  summary.addEventListener('focus', () => {
    state.summaryHadFocus.value = true;
  });
  summary.addEventListener('blur', () => {
    if (compactActionsMode(summary)) state.summaryHadFocus.value = false;
  });
  syncActionsMode(details, summary, panel, state, true);
  details.addEventListener('keydown', (event) => {
    onActionsEscape(details, summary, event);
  });
  document.addEventListener('click', (event) => {
    dismissOpenActions(details, summary, event.target);
  });
  details.addEventListener('focusout', (event) => {
    dismissActionsOnFocusOut(details, summary, event.relatedTarget);
  });
  const header = details.closest('.soda-repository-header');
  if (header && typeof ResizeObserver === 'function') {
    new ResizeObserver(() => syncActionsMode(details, summary, panel, state)).observe(header);
  }
}

for (const details of document.querySelectorAll<HTMLDetailsElement>('.soda-repository-actions')) {
  enhanceRepositoryActions(details);
}
