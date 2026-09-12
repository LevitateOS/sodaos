// Reuse Forgejo's patched Fomantic dropdown: it owns menu selection, keyboard
// navigation and outside-click dismissal. Only repository data loading is Soda-owned.
(() => {
  type Dropdown = {dropdown: (setting: string | Record<string, unknown>, ...args: unknown[]) => unknown};
  type Repository = {full_name: string; link: string; private?: boolean; fork?: boolean};
  const init = () => {
    const root = document.querySelector<HTMLElement>('.soda-repository-breadcrumb');
    const jq = (window as Window & {jQuery?: (element: HTMLElement) => Dropdown}).jQuery;
    if (!root || !jq || !root.dataset.userId) return;
    const prefix = root.dataset.subUrl ?? '';
    const currentOwner = {id: root.dataset.ownerId ?? '', name: root.dataset.ownerName ?? ''};
    let itemID = 0;
    const element = root.querySelector<HTMLElement>('.soda-repository-switcher[data-kind="repository"]');
    if (!element) return;
    const menu = element.querySelector<HTMLElement>(':scope > .menu')!;
    const input = element.querySelector<HTMLInputElement>('input.search')!;
    const status = element.querySelector<HTMLElement>('[role="status"]')!;
    const repoPicker = {element, menu, input, status, plugin: jq(element), request: undefined as AbortController | undefined, timer: 0};
    type Picker = typeof repoPicker;
    const currentRepo = root.dataset.repoLink;
    const currentName = `${currentOwner.name}/${root.dataset.repoName}`;
    const icon = (name: string) => root.querySelector<HTMLTemplateElement>(`template[data-symbol="${name}"]`)!.content.cloneNode(true);
    const cancel = (p: Picker) => { clearTimeout(p.timer); p.request?.abort(); p.request = undefined; };
    const clear = (p: Picker) => {
      for (const el of p.menu.querySelectorAll(':scope > .item, :scope > .soda-switcher-more,  :scope > .message:not(.soda-switcher-status)')) el.remove();
      p.input.removeAttribute('aria-activedescendant');
    };
    const setStatus = (p: Picker, text: string) => {
      // Fomantic removes its message node when displaying results. Reattach the
      // same live region for the next asynchronous loading/empty/error state.
      if (!p.menu.contains(p.status)) p.menu.append(p.status);
      p.status.textContent = text;
    };
    const position = (p: Picker) => {
      const anchor = p.element.getBoundingClientRect();
      p.menu.style.left = `${Math.max(8, Math.min(anchor.right - p.menu.offsetWidth, innerWidth - p.menu.offsetWidth - 8))}px`;
      p.menu.style.top = `${Math.max(8, Math.min(anchor.bottom + 8, innerHeight - 160))}px`;
      p.menu.style.maxHeight = `${Math.min(560, Math.max(120, innerHeight - parseFloat(p.menu.style.top) - 8))}px`;
    };
    const row = (p: Picker, label: string, value: string, selected: boolean, symbol: string, href?: string) => {
      const el = document.createElement(href ? 'a' : 'button');
      el.className = 'item js-aria-clickable';
      el.id = `soda-switcher-option-${++itemID}`;
      el.setAttribute('role', 'option');
      el.setAttribute('tabindex', '-1');
      el.setAttribute('aria-selected', 'false');
      el.dataset.value = value;
      el.dataset.text = label;
      if (el instanceof HTMLAnchorElement && href) el.href = href;
      if (el instanceof HTMLButtonElement) el.type = 'button';
      const name = document.createElement('span'); name.className = 'soda-switcher-name'; name.textContent = label;
      el.append(icon(symbol), name);
      if (selected) {
        el.append(icon('check'));
        if (href) el.setAttribute('aria-current', 'page');
      }
      p.menu.append(el);
      return el;
    };
    const get = async (p: Picker, url: string) => {
      p.request?.abort();
      const controller = new AbortController(); p.request = controller;
      const timeout = window.setTimeout(() => controller.abort(new Error('Request timed out')), 10000);
      try {
        const response = await fetch(url, {credentials: 'same-origin', cache: 'no-store', signal: controller.signal, headers: {'X-Requested-With': 'XMLHttpRequest'}});
        if (!response.ok || response.redirected) throw new Error('Could not load');
        const text = await response.text();
        if (controller !== p.request || controller.signal.aborted) return undefined;
        return text;
      } finally { clearTimeout(timeout); }
    };
    const failure = (p: Picker) => {
      if (!p.element.classList.contains('active')) return;
      clear(p); setStatus(p, 'Could not load. Try again.');
      const retry = row(p, 'Retry', 'retry', false, 'repo');
      retry.addEventListener('click', event => {event.stopPropagation(); void load(p);});
      p.plugin.dropdown('refresh');
    };
    const loadRepos = async (page = 1) => {
      const p = repoPicker;
      if (page === 1) clear(p);
      else p.menu.querySelector('.soda-switcher-more')?.remove();
      setStatus(p, 'Loading repositories…');
      const params = new URLSearchParams({uid: currentOwner.id, exclusive: 'true', q: p.input.value.trim(), limit: '15', page: String(page), sort: 'alpha'});
      try {
        const text = await get(p, `${prefix}/repo/search?${params}`);
        if (text === undefined) return;
        const data: unknown = JSON.parse(text);
        if (!data || typeof data !== 'object' || !('ok' in data) || data.ok !== true || !('data' in data) || !Array.isArray(data.data)) throw new Error('Invalid results');
        if (page === 1 && !p.input.value.trim() && currentRepo) {
          row(p, currentName, currentRepo, true, root.dataset.repoPrivate === 'true' ? 'private' : root.dataset.repoFork === 'true' ? 'fork' : 'repo', currentRepo);
        }
        const seen = new Set([...p.menu.querySelectorAll<HTMLAnchorElement>('a.item')].map(a => a.dataset.value));
        for (const entry of data.data) {
          const repo = (entry as {repository?: Repository}).repository;
          if (!repo || typeof repo.full_name !== 'string' || typeof repo.link !== 'string') throw new Error('Invalid repository');
          const url = new URL(repo.link, location.origin);
          if (url.origin !== location.origin || !url.pathname.startsWith(`${prefix}/`) || url.search || url.hash) throw new Error('Invalid destination');
          if (seen.has(repo.link)) continue;
          seen.add(repo.link);
          const symbol = repo.private ? 'private' : repo.fork ? 'fork' : 'repo';
          const el = row(p, repo.full_name, repo.link, repo.link === currentRepo, symbol, url.href);
          if (repo.private || repo.fork) el.setAttribute('aria-label', `${repo.full_name}, ${repo.private ? 'private repository' : 'fork'}`);
        }
        setStatus(p, p.menu.querySelector('.item') ? '' : 'No matching repositories.');
        if (data.data.length === 15) {
          const more = document.createElement('button'); more.type = 'button'; more.className = 'ui basic button soda-switcher-more'; more.textContent = 'Load more';
          more.addEventListener('click', event => {event.stopPropagation(); void loadRepos(page + 1);}); p.menu.append(more);
        }
        p.plugin.dropdown('refresh'); position(p);
      } catch (error) { if (!(error instanceof DOMException && error.name === 'AbortError')) failure(p); }
    };
    const load = (p: Picker) => { cancel(p); return loadRepos(); };
    const p = repoPicker;
    {
      p.element.setAttribute('role', 'button');
      p.element.setAttribute('aria-haspopup', 'listbox');
      p.element.setAttribute('aria-controls', p.menu.id);
      p.element.setAttribute('aria-expanded', 'false');
      p.plugin.dropdown({action: 'hide', selectOnKeydown: false, forceSelection: false, showOnFocus: false, fullTextSearch: true, transition: 'none', direction: 'downward',
        onShow: () => {
          p.element.setAttribute('aria-expanded', 'true');
          void load(p); requestAnimationFrame(() => {position(p); p.input.focus({preventScroll: true});});
        },
        onHide: () => { p.element.setAttribute('aria-expanded', 'false'); cancel(p); p.input.removeAttribute('aria-activedescendant'); },
      });
      p.element.hidden = false;
      // Native keyboard/ARIA stay in charge; remote repository search replaces
      // only native local text filtering, which cannot see paginated results.
      p.input.addEventListener('input', event => {
        event.stopImmediatePropagation();
        cancel(p); clear(p); setStatus(p, 'Searching…');
        p.timer = window.setTimeout(() => {void loadRepos();}, 200);
      }, true);
      p.element.addEventListener('keydown', event => {
        if (event.key === 'Escape') {p.plugin.dropdown('hide'); p.element.focus({preventScroll: true});}
      });
      p.element.addEventListener('focusout', event => {
        if (event.relatedTarget instanceof Node && !p.element.contains(event.relatedTarget)) p.plugin.dropdown('hide');
      });
      new ResizeObserver(() => {if (p.element.classList.contains('active')) position(p);}).observe(p.menu);
    }
    window.addEventListener('resize', () => {if (p.element.classList.contains('active')) position(p);});
    window.addEventListener('scroll', () => {if (p.element.classList.contains('active')) position(p);}, true);
  };
  if (document.readyState === 'complete') init(); else window.addEventListener('load', init, {once: true});
})();
