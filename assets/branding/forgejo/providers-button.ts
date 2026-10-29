/* Navbar "AI accounts" robot. The Soda providers panel
   (frontend/spaces/soda-providers-entry.ts) hosts the dialog on every
   signed-in page and marks <html data-soda-providers>; the robot opens it in
   place with a document event. Only when no panel is mounted does it fall
   back to the Feed, asking the dialog to open on arrival through a one-shot
   session flag. Keep the names in sync with soda-providers-events.ts. */
(() => {
  const script = document.currentScript;
  const base = script?.dataset.appSubUrl || '';
  document.addEventListener(
    'DOMContentLoaded',
    () => {
      const button = document.getElementById('soda-providers-button');
      if (!button) return;
      button.addEventListener('click', () => {
        if (document.documentElement.hasAttribute('data-soda-providers')) {
          document.dispatchEvent(new CustomEvent('soda-open-providers'));
          return;
        }
        try {
          sessionStorage.setItem('soda-open-providers', '1');
        } catch {
          /* Storage can be blocked; the page still opens without the dialog. */
        }
        location.assign(`${base}/feed`);
      });
    },
    {once: true}
  );
})();
