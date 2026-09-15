// Reuse Forgejo's patched Fomantic dropdown: it owns menu selection, keyboard
// navigation and outside-click dismissal. Only repository data loading is Soda-owned.

type Dropdown = {dropdown: (setting: string | Record<string, unknown>, ...args: unknown[]) => unknown};
type Repository = {full_name: string; link: string; private?: boolean; fork?: boolean};
type Picker = {
  element: HTMLElement;
  menu: HTMLElement;
  input: HTMLInputElement;
  status: HTMLElement;
  plugin: Dropdown;
  request: AbortController | undefined;
  timer: number;
};
type Switcher = {
  root: HTMLElement;
  prefix: string;
  currentOwner: {id: string; name: string};
  currentRepo: string | undefined;
  currentName: string;
  picker: Picker;
  itemID: number;
};

function jqueryDropdown(): ((element: HTMLElement) => Dropdown) | undefined {
  return (window as Window & {jQuery?: (element: HTMLElement) => Dropdown}).jQuery;
}

function cloneSymbol(root: HTMLElement, name: string): Node {
  return root.querySelector<HTMLTemplateElement>(`template[data-symbol="${name}"]`)!.content.cloneNode(true);
}

function cancelPicker(p: Picker): void {
  clearTimeout(p.timer);
  p.request?.abort();
  p.request = undefined;
}

function clearPicker(p: Picker): void {
  for (const el of p.menu.querySelectorAll(
    ':scope > .item, :scope > .soda-switcher-more,  :scope > .message:not(.soda-switcher-status)'
  )) {
    el.remove();
  }
  p.input.removeAttribute('aria-activedescendant');
}

function setPickerStatus(p: Picker, text: string): void {
  // Fomantic removes its message node when displaying results. Reattach the
  // same live region for the next asynchronous loading/empty/error state.
  if (!p.menu.contains(p.status)) p.menu.append(p.status);
  p.status.textContent = text;
}

function positionPicker(p: Picker): void {
  const anchor = p.element.getBoundingClientRect();
  p.menu.style.left = `${Math.max(8, Math.min(anchor.right - p.menu.offsetWidth, innerWidth - p.menu.offsetWidth - 8))}px`;
  p.menu.style.top = `${Math.max(8, Math.min(anchor.bottom + 8, innerHeight - 160))}px`;
  p.menu.style.maxHeight = `${Math.min(560, Math.max(120, innerHeight - parseFloat(p.menu.style.top) - 8))}px`;
}

function renderSwitcherRow(
  switcher: Switcher,
  label: string,
  value: string,
  selected: boolean,
  symbol: string,
  href?: string
): HTMLElement {
  const el = document.createElement(href ? 'a' : 'button');
  el.className = 'item js-aria-clickable';
  el.id = `soda-switcher-option-${++switcher.itemID}`;
  el.setAttribute('role', 'option');
  el.setAttribute('tabindex', '-1');
  el.setAttribute('aria-selected', 'false');
  el.dataset.value = value;
  el.dataset.text = label;
  if (el instanceof HTMLAnchorElement && href) el.href = href;
  if (el instanceof HTMLButtonElement) el.type = 'button';
  const name = document.createElement('span');
  name.className = 'soda-switcher-name';
  name.textContent = label;
  el.append(cloneSymbol(switcher.root, symbol), name);
  if (selected) {
    el.append(cloneSymbol(switcher.root, 'check'));
    if (href) el.setAttribute('aria-current', 'page');
  }
  switcher.picker.menu.append(el);
  return el;
}

async function fetchSwitcherText(p: Picker, url: string): Promise<string | undefined> {
  p.request?.abort();
  const controller = new AbortController();
  p.request = controller;
  const timeout = window.setTimeout(() => controller.abort(new Error('Request timed out')), 10000);
  try {
    const response = await fetch(url, {
      credentials: 'same-origin',
      cache: 'no-store',
      signal: controller.signal,
      headers: {'X-Requested-With': 'XMLHttpRequest'},
    });
    if (!response.ok || response.redirected) throw new Error('Could not load');
    const text = await response.text();
    if (controller !== p.request || controller.signal.aborted) return undefined;
    return text;
  } finally {
    clearTimeout(timeout);
  }
}

function parseSearchPayload(text: string): unknown[] {
  const data: unknown = JSON.parse(text);
  if (data === null || typeof data !== 'object') throw new Error('Invalid results');
  if (!('ok' in data) || data.ok !== true || !('data' in data) || !Array.isArray(data.data)) {
    throw new Error('Invalid results');
  }
  return data.data;
}

function admitRepository(entry: unknown): Repository {
  const repo = (entry as {repository?: Repository}).repository;
  if (!repo || typeof repo.full_name !== 'string' || typeof repo.link !== 'string')
    throw new Error('Invalid repository');
  return repo;
}

function admitRepositoryHref(prefix: string, link: string): string {
  const url = new URL(link, location.origin);
  if (url.origin !== location.origin || !url.pathname.startsWith(`${prefix}/`) || url.search || url.hash) {
    throw new Error('Invalid destination');
  }
  return url.href;
}

function repositorySymbol(repo: Repository): string {
  if (repo.private) return 'private';
  if (repo.fork) return 'fork';
  return 'repo';
}

function currentRepoSymbol(root: HTMLElement): string {
  if (root.dataset.repoPrivate === 'true') return 'private';
  if (root.dataset.repoFork === 'true') return 'fork';
  return 'repo';
}

function prepareRepoPage(p: Picker, page: number): void {
  if (page === 1) clearPicker(p);
  else p.menu.querySelector('.soda-switcher-more')?.remove();
}

function renderCurrentRepoIfNeeded(switcher: Switcher, page: number): void {
  if (page !== 1 || switcher.picker.input.value.trim() || !switcher.currentRepo) return;
  renderSwitcherRow(
    switcher,
    switcher.currentName,
    switcher.currentRepo,
    true,
    currentRepoSymbol(switcher.root),
    switcher.currentRepo
  );
}

function labelPrivateOrFork(el: HTMLElement, repo: Repository): void {
  if (!repo.private && !repo.fork) return;
  el.setAttribute('aria-label', `${repo.full_name}, ${repo.private ? 'private repository' : 'fork'}`);
}

function renderRepositoryRow(switcher: Switcher, repo: Repository, href: string, seen: Set<string | undefined>): void {
  if (seen.has(repo.link)) return;
  seen.add(repo.link);
  const el = renderSwitcherRow(
    switcher,
    repo.full_name,
    repo.link,
    repo.link === switcher.currentRepo,
    repositorySymbol(repo),
    href
  );
  labelPrivateOrFork(el, repo);
}

function appendAdmittedRepository(switcher: Switcher, entry: unknown, seen: Set<string | undefined>): void {
  const repo = admitRepository(entry);
  renderRepositoryRow(switcher, repo, admitRepositoryHref(switcher.prefix, repo.link), seen);
}

function renderLoadMore(switcher: Switcher, page: number, count: number): void {
  if (count !== 15) return;
  const more = document.createElement('button');
  more.type = 'button';
  more.className = 'ui basic button soda-switcher-more';
  more.textContent = 'Load more';
  more.addEventListener('click', (event) => {
    event.stopPropagation();
    void loadRepos(switcher, page + 1);
  });
  switcher.picker.menu.append(more);
}

function renderRepoResults(switcher: Switcher, page: number, entries: unknown[]): void {
  const p = switcher.picker;
  renderCurrentRepoIfNeeded(switcher, page);
  const seen = new Set([...p.menu.querySelectorAll<HTMLAnchorElement>('a.item')].map((a) => a.dataset.value));
  for (const entry of entries) appendAdmittedRepository(switcher, entry, seen);
  setPickerStatus(p, p.menu.querySelector('.item') ? '' : 'No matching repositories.');
  renderLoadMore(switcher, page, entries.length);
  p.plugin.dropdown('refresh');
  positionPicker(p);
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError';
}

function failure(switcher: Switcher): void {
  const p = switcher.picker;
  if (!p.element.classList.contains('active')) return;
  clearPicker(p);
  setPickerStatus(p, 'Could not load. Try again.');
  const retry = renderSwitcherRow(switcher, 'Retry', 'retry', false, 'repo');
  retry.addEventListener('click', (event) => {
    event.stopPropagation();
    void loadSwitcher(switcher);
  });
  p.plugin.dropdown('refresh');
}

async function loadRepos(switcher: Switcher, page = 1): Promise<void> {
  const p = switcher.picker;
  prepareRepoPage(p, page);
  setPickerStatus(p, 'Loading repositories…');
  const params = new URLSearchParams({
    uid: switcher.currentOwner.id,
    exclusive: 'true',
    q: p.input.value.trim(),
    limit: '15',
    page: String(page),
    sort: 'alpha',
  });
  try {
    const text = await fetchSwitcherText(p, `${switcher.prefix}/repo/search?${params}`);
    if (text === undefined) return;
    renderRepoResults(switcher, page, parseSearchPayload(text));
  } catch (error) {
    if (!isAbortError(error)) failure(switcher);
  }
}

function loadSwitcher(switcher: Switcher): Promise<void> {
  cancelPicker(switcher.picker);
  return loadRepos(switcher);
}

function repositionIfActive(p: Picker): void {
  if (p.element.classList.contains('active')) positionPicker(p);
}

function bindSwitcherEvents(switcher: Switcher): void {
  const p = switcher.picker;
  p.element.setAttribute('role', 'button');
  p.element.setAttribute('aria-haspopup', 'listbox');
  p.element.setAttribute('aria-controls', p.menu.id);
  p.element.setAttribute('aria-expanded', 'false');
  p.plugin.dropdown({
    action: 'hide',
    selectOnKeydown: false,
    forceSelection: false,
    showOnFocus: false,
    fullTextSearch: true,
    transition: 'none',
    direction: 'downward',
    onShow: () => {
      p.element.setAttribute('aria-expanded', 'true');
      void loadSwitcher(switcher);
      requestAnimationFrame(() => {
        positionPicker(p);
        p.input.focus({preventScroll: true});
      });
    },
    onHide: () => {
      p.element.setAttribute('aria-expanded', 'false');
      cancelPicker(p);
      p.input.removeAttribute('aria-activedescendant');
    },
  });
  p.element.hidden = false;
  // Native keyboard/ARIA stay in charge; remote repository search replaces
  // only native local text filtering, which cannot see paginated results.
  p.input.addEventListener(
    'input',
    (event) => {
      event.stopImmediatePropagation();
      cancelPicker(p);
      clearPicker(p);
      setPickerStatus(p, 'Searching…');
      p.timer = window.setTimeout(() => {
        void loadRepos(switcher);
      }, 200);
    },
    true
  );
  p.element.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      p.plugin.dropdown('hide');
      p.element.focus({preventScroll: true});
    }
  });
  p.element.addEventListener('focusout', (event) => {
    if (event.relatedTarget instanceof Node && !p.element.contains(event.relatedTarget)) p.plugin.dropdown('hide');
  });
  new ResizeObserver(() => {
    repositionIfActive(p);
  }).observe(p.menu);
  window.addEventListener('resize', () => {
    repositionIfActive(p);
  });
  window.addEventListener(
    'scroll',
    () => {
      repositionIfActive(p);
    },
    true
  );
}

function enhanceRepositorySwitcher(): void {
  const root = document.querySelector<HTMLElement>('.soda-repository-breadcrumb');
  const jq = jqueryDropdown();
  if (!root || !jq || !root.dataset.userId) return;
  const element = root.querySelector<HTMLElement>('.soda-repository-switcher[data-kind="repository"]');
  if (!element) return;
  const menu = element.querySelector<HTMLElement>(':scope > .menu')!;
  const input = element.querySelector<HTMLInputElement>('input.search')!;
  const status = element.querySelector<HTMLElement>('[role="status"]')!;
  const currentOwner = {id: root.dataset.ownerId ?? '', name: root.dataset.ownerName ?? ''};
  bindSwitcherEvents({
    root,
    prefix: root.dataset.subUrl ?? '',
    currentOwner,
    currentRepo: root.dataset.repoLink,
    currentName: `${currentOwner.name}/${root.dataset.repoName}`,
    picker: {element, menu, input, status, plugin: jq(element), request: undefined, timer: 0},
    itemID: 0,
  });
}

if (document.readyState === 'complete') enhanceRepositorySwitcher();
else window.addEventListener('load', enhanceRepositorySwitcher, {once: true});
