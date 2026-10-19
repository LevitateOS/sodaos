/* Navbar AI-providers button: opens the Soda providers dialog hosted by the
   current extension page, or navigates to the feed (which hosts it) and
   asks it to open on arrival through a one-shot session flag. */
(() => {
  const script = document.currentScript;
  const base = script?.dataset.appSubUrl || '';
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
          /* Storage can be blocked; the feed still opens without the dialog. */
        }
        location.assign(`${base}/feed`);
      });
    },
    {once: true}
  );
})();
