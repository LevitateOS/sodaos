/* Guest homepage/login appearance only; Forgejo owns authenticated account themes. */
(() => {
  const root = document.documentElement;
  const script = document.currentScript;
  const key = `soda.login.theme:${script?.dataset.appSubUrl || '/'}`;
  const system = window.matchMedia('(prefers-color-scheme: dark)');
  const valid = (value: unknown): value is 'light' | 'dark' => value === 'light' || value === 'dark';
  let choice: string | null = null;
  try { choice = localStorage.getItem(key); } catch { /* Storage can be blocked. */ }
  if (!valid(choice)) choice = null;
  function apply() {
    const theme = choice || (system.matches ? 'dark' : 'light');
    root.dataset.sodaLoginTheme = theme;
    const button = document.getElementById('soda-theme-toggle');
    if (button) {
      const label = theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme';
      button.setAttribute('aria-label', label);
      button.title = label;
      button.hidden = false;
    }
  }
  apply(); // Runs in the head before the login paints.
  document.addEventListener('DOMContentLoaded', () => {
    const button = document.getElementById('soda-theme-toggle');
    if (!button) return;
    button.addEventListener('click', () => {
      choice = root.dataset.sodaLoginTheme === 'dark' ? 'light' : 'dark';
      try { localStorage.setItem(key, choice); } catch { /* Keep an in-memory choice. */ }
      apply();
    });
    apply();
  }, {once: true});
  system.addEventListener('change', () => { if (!choice) apply(); });
  window.addEventListener('storage', event => {
    if (event.key !== key && event.key !== null) return;
    choice = valid(event.newValue) ? event.newValue : null;
    apply();
  });
})();
