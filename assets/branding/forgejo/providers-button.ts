/* Navbar "AI accounts" robot. On a SodaOS place, the page already hosts
   <soda-providers>, so the dialog opens in place. Anywhere else it goes to a
   page that hosts it and asks it to open on arrival through a one-shot
   session flag: the current repository's SodaOS Feed when the header
   template passed one (data-repository-feed), otherwise the navbar Feed. */
(() => {
  const script = document.currentScript;
  const base = script?.dataset.appSubUrl || '';
  const repositoryFeed = script?.dataset.repositoryFeed || '';
  document.addEventListener(
    'DOMContentLoaded',
    () => {
      const button = document.getElementById('soda-providers-button');
      if (!button) return;
      button.addEventListener('click', () => {
        const host = document.querySelector('soda-providers');
        const open = (host as unknown as {open?: unknown} | null)?.open;
        if (typeof open === 'function') {
          void (open as () => Promise<void>).call(host);
          return;
        }
        try {
          sessionStorage.setItem('soda-open-providers', '1');
        } catch {
          /* Storage can be blocked; the page still opens without the dialog. */
        }
        location.assign(repositoryFeed || `${base}/feed`);
      });
    },
    {once: true}
  );
})();
