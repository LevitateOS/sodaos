(() => {
  for (const details of document.querySelectorAll<HTMLDetailsElement>('.soda-repository-actions')) {
    if (details.dataset.repositoryActionsEnhanced) continue;
    const summary = details.querySelector<HTMLElement>(':scope > summary');
    const panel = details.querySelector<HTMLElement>(':scope > .soda-repository-action-menu');
    if (!summary || !panel) continue;
    details.dataset.repositoryActionsEnhanced = 'true';

    const firstAvailableAction = () => [...panel.querySelectorAll<HTMLElement>('a, button')].find(action =>
      !action.matches(':disabled, [aria-disabled="true"]') &&
      !action.closest('.disabled, .ui.modal') && !action.hidden,
    );
    const compactMode = () => getComputedStyle(summary).display !== 'none';
    let compact = false;
    // Hiding summary can blur it before ResizeObserver delivers the new mode.
    let summaryHadFocus = false;
    summary.addEventListener('focus', () => { summaryHadFocus = true; });
    summary.addEventListener('blur', () => {
      if (compactMode()) summaryHadFocus = false;
    });
    const syncMode = (initial = false) => {
      const nextCompact = compactMode();
      const enteredCompact = !compact && nextCompact;
      compact = nextCompact;
      if (!compact) {
        const transferFocus = summaryHadFocus || document.activeElement === summary;
        summaryHadFocus = false;
        details.open = true;
        if (transferFocus) firstAvailableAction()?.focus({preventScroll: true});
      } else if (initial || (enteredCompact && !panel.contains(document.activeElement))) {
        details.open = false;
      }
    };
    syncMode(true);

    details.addEventListener('keydown', event => {
      if (!details.open || event.key !== 'Escape' || event.defaultPrevented || !compactMode()) return;
      if (event.target instanceof Element && event.target.closest('.ui.modal')) return;
      event.preventDefault();
      event.stopPropagation();
      details.open = false;
      summary.focus({preventScroll: true});
    });

    document.addEventListener('click', event => {
      const target = event.target;
      if (!details.open || !(target instanceof Node) || details.contains(target) || !compactMode()) return;
      details.open = false;
    });

    details.addEventListener('focusout', event => {
      const next = event.relatedTarget;
      if (!details.open || next === null || (next instanceof Node && details.contains(next)) || !compactMode()) return;
      details.open = false;
    });

    const header = details.closest('.soda-repository-header');
    if (header && typeof ResizeObserver === 'function') new ResizeObserver(() => syncMode()).observe(header);
  }
})();
