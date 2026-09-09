(() => {
  for (const details of document.querySelectorAll<HTMLDetailsElement>('.soda-repository-actions')) {
    if (details.dataset.repositoryActionsEnhanced) continue;
    const summary = details.querySelector<HTMLElement>(':scope > summary');
    if (!summary) continue;
    details.dataset.repositoryActionsEnhanced = 'true';

    details.addEventListener('keydown', event => {
      if (!details.open || event.key !== 'Escape' || event.defaultPrevented) return;
      if (event.target instanceof Element && event.target.closest('.ui.modal')) return;
      event.preventDefault();
      event.stopPropagation();
      details.open = false;
      summary.focus({preventScroll: true});
    });

    document.addEventListener('click', event => {
      const target = event.target;
      if (!details.open || !(target instanceof Node) || details.contains(target)) return;
      details.open = false;
    });

    details.addEventListener('focusout', event => {
      const next = event.relatedTarget;
      if (!details.open || next === null || (next instanceof Node && details.contains(next))) return;
      details.open = false;
    });
  }
})();
