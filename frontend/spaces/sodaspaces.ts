// Native navigation remains native. Repository/width/open are hints, never
// session authority. Compact Forge/Terminal visibility belongs to this document.
import type {WorkspaceContext} from './sodaspaces-workspace.js';
import {object} from './sodaspaces-api.js';
const identifier = (v: unknown): v is string =>
  typeof v === 'string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v) <= 9223372036854775807n;
export function workspaceWidths(viewport: number, terminalMinimum: number, desired: number) {
  const minimum = Math.max(35, (terminalMinimum / Math.max(1, viewport)) * 100),
    maximum = Math.min(65, ((viewport - 480) / Math.max(1, viewport)) * 100);
  return {compact: minimum > maximum, minimum, maximum, actual: Math.max(minimum, Math.min(maximum, desired))};
}
export interface DrawerContent {
  readonly ready?: Promise<unknown>;
  refresh(): void | Promise<void>;
  dispose(): void;
  markViewed?(): void;
  setVisible?(visible: boolean): void;
}

type BrowserWindow = Window & typeof globalThis;
type DrawerChrome = {
  button: HTMLElement;
  drawer: HTMLElement;
  close: HTMLElement;
  content: HTMLElement;
  divider: HTMLElement;
};
type SavedWorkspace = {repositoryId: string; open: boolean; width?: number};

function drawerRoots(doc: Document) {
  const roots = doc.querySelectorAll<HTMLElement>('#sodaspaces-root'),
    rows = doc.querySelectorAll('.repo-header .repo-buttons');
  if (roots.length !== 1 || rows.length > 1) return;
  const root = roots[0],
    win = doc.defaultView;
  if (!root || !win || root.dataset.mounted) return;
  return {root, win: win as BrowserWindow, rows};
}

function admitDrawerDataset(root: HTMLElement) {
  const {repositoryId: nativeRepository, userId, signed, subUrl} = root.dataset;
  if (
    subUrl !== '' ||
    (nativeRepository !== '' && !identifier(nativeRepository)) ||
    (signed !== 'true' && signed !== 'false') ||
    (signed === 'true' ? !identifier(userId) : userId !== '')
  )
    return;
  return {nativeRepository, userId, signed, subUrl};
}

function savedWidth(value: Record<string, unknown>) {
  return typeof value.width === 'number' && Number.isFinite(value.width) ? {width: value.width} : {};
}

function readSavedWorkspace(win: BrowserWindow, storageKey: string | null): SavedWorkspace | undefined {
  try {
    const raw = storageKey ? win.sessionStorage.getItem(storageKey) : null;
    if (!raw || raw.length > 512) return;
    const value = object(JSON.parse(raw));
    if (identifier(value.repositoryId) && typeof value.open === 'boolean')
      return {repositoryId: value.repositoryId, open: value.open, ...savedWidth(value)};
  } catch {
    /* Untrusted locator, not authority. */
  }
}

function drawerChrome(root: HTMLElement): DrawerChrome | undefined {
  const button = root.querySelector<HTMLElement>('#sodaspaces-button'),
    drawer = root.querySelector<HTMLElement>('#sodaspaces-drawer');
  const close = root.querySelector<HTMLElement>('#sodaspaces-close'),
    content = root.querySelector<HTMLElement>('#sodaspaces-content'),
    divider = root.querySelector<HTMLElement>('#sodaspaces-divider');
  if (!button || !drawer || !close || !content || !divider) return;
  return {button, drawer, close, content, divider};
}

type NativeWorkspace = Extract<WorkspaceContext, {kind: 'native'}>;

function nativeWorkspace(
  nativeRepository: string | undefined,
  signed: string | undefined,
  userId: string | undefined,
  repositoryId: string
): NativeWorkspace {
  return {
    kind: 'native',
    expectedUserId: signed === 'true' ? userId : undefined,
    repositoryId: identifier(nativeRepository) ? nativeRepository : repositoryId,
    ...(identifier(nativeRepository) ? {pageRepositoryId: nativeRepository} : {}),
  };
}

class DrawerHost {
  private controller: DrawerContent | undefined;
  private departed = false;
  private width: number;
  private surface: 'forge' | 'terminal' = 'forge';
  private compact = false;
  private terminalMinimum = 0;
  private nativeFocus: HTMLElement | undefined;
  private readonly covered = new Map<HTMLElement, boolean>();
  private generation = 0;
  private loading: Promise<void> | undefined;
  private readonly lifetime: AbortController;
  private readonly observer: MutationObserver;
  private readonly switcher: HTMLElement;
  private readonly forge: HTMLButtonElement;
  private readonly terminalBtn: HTMLButtonElement;
  private readonly measure: HTMLElement;

  constructor(
    private readonly doc: Document,
    private readonly win: BrowserWindow,
    private readonly root: HTMLElement,
    rows: NodeListOf<Element>,
    private readonly chrome: DrawerChrome,
    private readonly context: NativeWorkspace,
    private readonly repositoryId: string,
    private readonly storageKey: string | null,
    saved: SavedWorkspace | undefined,
    private readonly mountContent?: (root: HTMLElement, context: NativeWorkspace) => DrawerContent
  ) {
    this.width = saved?.width !== undefined ? Math.max(35, Math.min(65, saved.width)) : 50;
    this.lifetime = new win.AbortController();
    root.dataset.mounted = 'true';
    if (rows.length) rows.item(0).append(chrome.button);
    else chrome.button.classList.add('sodaspaces-global-resume');
    root.hidden = false;
    this.switcher = doc.createElement('nav');
    this.switcher.id = 'sodaspaces-surfaces';
    this.switcher.setAttribute('aria-label', 'Workspace surface');
    this.switcher.hidden = true;
    this.forge = doc.createElement('button');
    this.terminalBtn = doc.createElement('button');
    this.forge.textContent = 'Forge';
    this.terminalBtn.textContent = 'Terminal';
    this.forge.type = this.terminalBtn.type = 'button';
    this.forge.className = this.terminalBtn.className = 'ui button';
    this.switcher.append(this.forge, this.terminalBtn);
    root.append(this.switcher);
    this.measure = doc.createElement('span');
    this.measure.className = 'sodaspaces-measure';
    this.measure.setAttribute('aria-hidden', 'true');
    this.measure.textContent = 'MMMMMMMMMMMMMMMM';
    root.append(this.measure);
    this.observer = new win.MutationObserver(() => this.observeNative());
  }

  private observeNative() {
    if (!this.chrome.drawer.hidden && this.compact && this.surface === 'terminal') this.coverNative(this.doc.body);
  }

  private persist() {
    if (this.storageKey)
      try {
        this.win.sessionStorage.setItem(
          this.storageKey,
          JSON.stringify({repositoryId: this.repositoryId, open: !this.chrome.drawer.hidden, width: this.width})
        );
      } catch {
        /* No credentials or compact visibility. */
      }
  }

  private exposeNative() {
    for (const [element, inert] of this.covered) {
      element.inert = inert;
      delete element.dataset.sodaNativeCovered;
    }
    this.covered.clear();
  }

  private isCoverable(element: Element): element is HTMLElement {
    return (
      element instanceof this.win.HTMLElement &&
      element !== this.root &&
      !['SCRIPT', 'STYLE', 'LINK'].includes(element.tagName)
    );
  }

  private coverNative(parent: Element) {
    for (const element of parent.children) {
      if (!this.isCoverable(element)) continue;
      if (element.contains(this.root)) {
        this.coverNative(element);
        continue;
      }
      if (!this.covered.has(element)) this.covered.set(element, element.inert);
      element.inert = true;
      element.dataset.sodaNativeCovered = '';
    }
  }

  private cellWidth() {
    return this.measure.getBoundingClientRect().width / 16 || 9;
  }

  private availableWidth() {
    return this.win.visualViewport?.width || this.win.innerWidth;
  }

  private captureForgeFocus() {
    const active = this.doc.activeElement;
    if (active instanceof this.win.HTMLElement && active !== this.doc.body && !this.root.contains(active))
      this.surface = 'forge';
  }

  private applyCover() {
    const {drawer} = this.chrome,
      visible = !drawer.hidden && (!this.compact || this.surface === 'terminal');
    drawer.inert = !visible;
    drawer.toggleAttribute('data-surface-hidden', !visible);
    drawer.setAttribute('aria-hidden', String(!visible));
    this.forge.setAttribute('aria-pressed', String(this.surface === 'forge'));
    this.terminalBtn.setAttribute('aria-pressed', String(this.surface === 'terminal'));
    if (!drawer.hidden && this.compact && visible) this.coverNative(this.doc.body);
    else this.exposeNative();
    this.controller?.setVisible?.(visible);
  }

  private applyGeometry(geometry: ReturnType<typeof workspaceWidths>) {
    const {drawer, divider} = this.chrome;
    this.doc.body.style.setProperty('--soda-space-width', geometry.actual + 'vw');
    this.doc.body.style.setProperty(
      '--soda-viewport-height',
      (this.win.visualViewport?.height || this.win.innerHeight) + 'px'
    );
    this.doc.body.style.setProperty('--soda-viewport-top', (this.win.visualViewport?.offsetTop || 0) + 'px');
    this.doc.body.classList.toggle('sodaspaces-compact', !drawer.hidden && this.compact);
    this.switcher.hidden = drawer.hidden || !this.compact;
    divider.setAttribute('aria-valuemin', String(Math.round(geometry.minimum * 10) / 10));
    divider.setAttribute('aria-valuemax', String(Math.round(geometry.maximum * 10) / 10));
    divider.setAttribute('aria-valuenow', String(Math.round(geometry.actual * 10) / 10));
  }

  private size = () => {
    const geometry = workspaceWidths(
        this.availableWidth(),
        Math.max(this.cellWidth() * 56 + 28, this.terminalMinimum),
        this.width
      ),
      wasCompact = this.compact;
    this.compact = geometry.compact;
    if (this.compact && !wasCompact) this.captureForgeFocus();
    this.applyGeometry(geometry);
    this.applyCover();
  };

  private mountWorkspace(module: typeof import('./sodaspaces-workspace.js'), epoch: number) {
    if (this.departed || this.generation !== epoch || this.chrome.drawer.hidden) return;
    this.controller = module.mountSodaspaces(this.chrome.content, this.context);
    this.controller.setVisible?.(!this.compact || this.surface === 'terminal');
    void this.controller.refresh();
  }

  private workspaceLoadFailed(epoch: number) {
    if (!this.departed && this.generation === epoch && !this.chrome.drawer.hidden)
      this.chrome.content.textContent = 'Workspace could not load. Reload the page; no action was sent.';
  }

  private finishLoad(epoch: number) {
    if (this.generation === epoch) this.loading = undefined;
  }

  private mount() {
    if (this.controller) return;
    const visible = !this.compact || this.surface === 'terminal';
    if (this.mountContent) {
      this.controller = this.mountContent(this.chrome.content, this.context);
      this.controller.setVisible?.(visible);
      void this.controller.refresh();
      return;
    }
    if (this.loading) return this.loading;
    const epoch = this.generation;
    this.loading = import('./sodaspaces-workspace.js')
      .then((module) => this.mountWorkspace(module, epoch))
      .catch(() => this.workspaceLoadFailed(epoch))
      .finally(() => this.finishLoad(epoch));
    return this.loading;
  }

  private drawerFocusable(epoch: number) {
    const {drawer} = this.chrome;
    return (
      !this.departed && this.generation === epoch && !drawer.hidden && (!this.compact || this.surface === 'terminal')
    );
  }

  private focusBelongsToDrawer() {
    const {button, close, content} = this.chrome,
      active = this.doc.activeElement;
    return active === button || active === close || active === this.doc.body || content.contains(active);
  }

  private focusDrawer(epoch: number) {
    if (!this.drawerFocusable(epoch) || !this.focusBelongsToDrawer()) return;
    this.controller?.markViewed?.();
    const {close, content} = this.chrome;
    (content.querySelector<HTMLElement>('[role=tab][aria-selected=true]') || close).focus();
  }

  private renderFailed(epoch: number) {
    if (!this.departed && this.generation === epoch && !this.chrome.drawer.hidden) {
      this.release();
      this.chrome.content.textContent = 'Workspace could not render. Reload; no action was replayed.';
    }
  }

  private focusAfterMount(mounted: Promise<void> | undefined) {
    const epoch = this.generation;
    const focus = () => this.focusDrawer(epoch);
    if (mounted || this.controller?.ready)
      void Promise.resolve(mounted)
        .then(() => this.controller?.ready)
        .then(focus)
        .catch(() => this.renderFailed(epoch));
    else focus();
  }

  show(deliberate = false, intent = false) {
    if (this.departed) return;
    if (deliberate || intent) this.surface = 'terminal';
    this.chrome.drawer.hidden = false;
    this.doc.body.classList.add('sodaspaces-open');
    this.size();
    this.chrome.button.setAttribute('aria-expanded', 'true');
    // Explicit drawer intent wins over the focus-sensitive compact transition.
    if (deliberate || intent) {
      this.surface = 'terminal';
      this.size();
    }
    const mounted = this.mount();
    this.persist();
    if (deliberate) this.focusAfterMount(mounted);
  }

  private hide = () => {
    this.chrome.drawer.hidden = true;
    this.doc.body.classList.remove('sodaspaces-open');
    this.size();
    this.chrome.button.setAttribute('aria-expanded', 'false');
    this.persist();
    this.chrome.button.focus();
  };

  private release() {
    ++this.generation;
    this.loading = undefined;
    this.controller?.dispose();
    this.controller = undefined;
    this.chrome.content.replaceChildren();
  }

  private resizeDivider(desired: number) {
    const geometry = workspaceWidths(
      this.availableWidth(),
      Math.max(this.terminalMinimum, this.cellWidth() * 56 + 28),
      this.width
    );
    if (!geometry.compact) {
      this.width = Math.max(geometry.minimum, Math.min(geometry.maximum, desired));
      this.size();
      this.persist();
    }
  }

  private onDividerKey(event: KeyboardEvent) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    this.resizeDivider(
      event.key === 'Home' ? 35 : event.key === 'End' ? 65 : this.width + (event.key === 'ArrowLeft' ? 5 : -5)
    );
  }

  private onButtonClick() {
    if (this.chrome.drawer.hidden) this.show(true);
    else if (!this.departed) {
      this.surface = 'terminal';
      this.size();
      this.controller?.markViewed?.();
    }
  }

  private onForgeClick() {
    this.surface = 'forge';
    this.size();
    if (this.nativeFocus?.isConnected && !this.nativeFocus.inert) this.nativeFocus.focus({preventScroll: true});
  }

  private onTerminalClick() {
    this.surface = 'terminal';
    this.size();
    this.controller?.markViewed?.();
  }

  private onFocusIn(event: FocusEvent) {
    if (event.target instanceof this.win.HTMLElement && !this.root.contains(event.target))
      this.nativeFocus = event.target;
  }

  private onWorkspaceMinimum(event: Event) {
    if (
      !(event instanceof this.win.CustomEvent) ||
      typeof event.detail !== 'number' ||
      !Number.isFinite(event.detail) ||
      event.detail <= 0 ||
      event.detail >= 10000
    )
      return;
    this.terminalMinimum = event.detail;
    this.size();
  }

  private onPointerDown(event: PointerEvent) {
    if (event.button === 0) {
      this.chrome.divider.setPointerCapture(event.pointerId);
      event.preventDefault();
    }
  }

  private onPointerMove(event: PointerEvent) {
    if (this.chrome.divider.hasPointerCapture(event.pointerId))
      this.resizeDivider(((this.win.innerWidth - event.clientX) / this.win.innerWidth) * 100);
  }

  private onPointerUp(event: PointerEvent) {
    if (this.chrome.divider.hasPointerCapture(event.pointerId))
      this.chrome.divider.releasePointerCapture(event.pointerId);
  }

  private onPageHide() {
    this.departed = true;
    this.release();
    this.surface = 'forge';
    this.size();
  }

  private onPageShow(event: PageTransitionEvent) {
    if (!event.persisted) return;
    this.departed = false;
    this.release();
    this.surface = 'forge';
    if (!this.chrome.drawer.hidden) this.show();
  }

  bind() {
    const {button, close, divider} = this.chrome,
      options = {signal: this.lifetime.signal},
      win = this.win,
      doc = this.doc;
    button.addEventListener('click', () => this.onButtonClick(), options);
    close.addEventListener('click', this.hide, options);
    this.forge.addEventListener('click', () => this.onForgeClick(), options);
    this.terminalBtn.addEventListener('click', () => this.onTerminalClick(), options);
    doc.addEventListener('focusin', (event) => this.onFocusIn(event), options);
    this.root.addEventListener('soda-workspace-minimum', (event) => this.onWorkspaceMinimum(event), options);
    divider.addEventListener('pointerdown', (e) => this.onPointerDown(e), options);
    divider.addEventListener('pointermove', (e) => this.onPointerMove(e), options);
    divider.addEventListener('pointerup', (e) => this.onPointerUp(e), options);
    divider.addEventListener('keydown', (e) => this.onDividerKey(e), options);
    this.observer.observe(doc.body, {childList: true});
    win.addEventListener('resize', this.size, options);
    win.visualViewport?.addEventListener('resize', this.size, options);
    win.visualViewport?.addEventListener('scroll', this.size, options);
    doc.fonts?.addEventListener('loadingdone', this.size, options);
    void doc.fonts?.ready.then(this.size);
    win.addEventListener('pagehide', () => this.onPageHide(), options);
    win.addEventListener('pageshow', (event) => this.onPageShow(event), options);
  }

  dispose() {
    this.release();
    this.lifetime.abort();
    this.observer.disconnect();
    this.exposeNative();
    this.switcher.remove();
    this.measure.remove();
    this.chrome.drawer.hidden = true;
    this.chrome.drawer.inert = false;
    this.doc.body.classList.remove('sodaspaces-open', 'sodaspaces-compact');
    for (const prop of ['--soda-space-width', '--soda-viewport-height', '--soda-viewport-top'])
      this.doc.body.style.removeProperty(prop);
  }
}

function resumeWorkspace(
  found: {root: HTMLElement; win: BrowserWindow; rows: NodeListOf<Element>},
  data: NonNullable<ReturnType<typeof admitDrawerDataset>>
) {
  const storageKey = data.signed === 'true' ? 'soda-workspace:' + data.userId : null;
  const saved = readSavedWorkspace(found.win, storageKey);
  if (found.rows.length === 0 && !saved) return;
  const selectedRepository = saved?.repositoryId || data.nativeRepository;
  if (!identifier(selectedRepository)) return;
  const chrome = drawerChrome(found.root);
  if (!chrome) return;
  return {storageKey, saved, selectedRepository, chrome};
}

export function mountDrawer(
  doc: Document,
  mountContent?: (root: HTMLElement, context: NativeWorkspace) => DrawerContent
) {
  const found = drawerRoots(doc);
  if (!found) return;
  const data = admitDrawerDataset(found.root);
  if (!data) return;
  const resume = resumeWorkspace(found, data);
  if (!resume) return;
  const host = new DrawerHost(
    doc,
    found.win,
    found.root,
    found.rows,
    resume.chrome,
    nativeWorkspace(data.nativeRepository, data.signed, data.userId, resume.selectedRepository),
    resume.selectedRepository,
    resume.storageKey,
    resume.saved,
    mountContent
  );
  host.bind();
  if (resume.saved?.open || found.win.location.hash === '#sodaspaces')
    host.show(false, found.win.location.hash === '#sodaspaces');
  return {
    dispose() {
      host.dispose();
    },
  };
}
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading')
    document.addEventListener('DOMContentLoaded', () => mountDrawer(document), {once: true});
  else mountDrawer(document);
}
