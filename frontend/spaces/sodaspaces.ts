// Native navigation remains native. Repository/width/open are hints, never
// session authority. Compact Forge/Terminal visibility belongs to this document.
import type {WorkspaceContext} from './sodaspaces-workspace.js';
import {object} from './sodaspaces-api.js';
const identifier = (v: unknown): v is string => typeof v === 'string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v) <= 9223372036854775807n;
export function workspaceWidths(viewport: number, terminalMinimum: number, desired: number) {
  const minimum = Math.max(35, terminalMinimum / Math.max(1, viewport) * 100), maximum = Math.min(65, (viewport - 480) / Math.max(1, viewport) * 100);
  return {compact: minimum > maximum, minimum, maximum, actual: Math.max(minimum, Math.min(maximum, desired))};
}
export interface DrawerContent {readonly ready?: Promise<unknown>; refresh(): void | Promise<void>; dispose(): void; setVisible?(visible: boolean): void; retain?(): void | Promise<void> | undefined; returnToWork?(): void | Promise<void> | undefined}
export function mountDrawer(doc: Document, mountContent?: (root: HTMLElement, context: Extract<WorkspaceContext, {kind: 'native'}>) => DrawerContent) {
  const roots = doc.querySelectorAll<HTMLElement>('#sodaspaces-root'), rows = doc.querySelectorAll('.repo-header .repo-buttons');
  if (roots.length !== 1 || rows.length > 1) return;
  const root = roots[0], win = doc.defaultView; if (!root || !win || root.dataset.mounted) return;
  const browser = win as Window & typeof globalThis;
  const {repositoryId: nativeRepository, userId, signed, subUrl} = root.dataset;
  if (subUrl !== '' || (nativeRepository !== '' && !identifier(nativeRepository)) || (signed !== 'true' && signed !== 'false') || (signed === 'true' ? !identifier(userId) : userId !== '')) return;
  const storageKey = signed === 'true' ? 'soda-workspace:' + userId : null;
  let saved: {repositoryId: string; open: boolean; width?: number} | undefined;
  try {
    const raw = storageKey ? win.sessionStorage.getItem(storageKey) : null;
    if (raw && raw.length <= 512) {const value = object(JSON.parse(raw)); if (identifier(value.repositoryId) && typeof value.open === 'boolean') saved = {repositoryId: value.repositoryId, open: value.open, ...(typeof value.width === 'number' && Number.isFinite(value.width) ? {width: value.width} : {})};}
  } catch { /* Untrusted locator, not authority. */ }
  if (rows.length === 0 && !saved) return;
  const selectedRepository = saved?.repositoryId || nativeRepository; if (!identifier(selectedRepository)) return;
  const repositoryId = selectedRepository;
  const context: WorkspaceContext = {kind: 'native', expectedUserId: signed === 'true' ? userId : undefined, repositoryId: identifier(nativeRepository) ? nativeRepository : repositoryId, ...(identifier(nativeRepository) ? {pageRepositoryId: nativeRepository} : {})};
  const button = root.querySelector<HTMLElement>('#sodaspaces-button'), drawer = root.querySelector<HTMLElement>('#sodaspaces-drawer');
  const close = root.querySelector<HTMLElement>('#sodaspaces-close'), content = root.querySelector<HTMLElement>('#sodaspaces-content'), divider = root.querySelector<HTMLElement>('#sodaspaces-divider');
  if (!button || !drawer || !close || !content || !divider) return;
  const lifetime = new browser.AbortController(), options = {signal: lifetime.signal};
  root.dataset.mounted = 'true';
  if (rows.length) rows.item(0).append(button); else button.classList.add('sodaspaces-global-resume');
  root.hidden = false;
  const switcher = doc.createElement('nav'); switcher.id = 'sodaspaces-surfaces'; switcher.setAttribute('aria-label', 'Workspace surface'); switcher.hidden = true;
  const forge = doc.createElement('button'), terminal = doc.createElement('button'); forge.textContent = 'Forge'; terminal.textContent = 'Terminal'; forge.type = terminal.type = 'button'; forge.className = terminal.className = 'ui button'; switcher.append(forge, terminal); root.append(switcher);
  const measure = doc.createElement('span'); measure.className = 'sodaspaces-measure'; measure.setAttribute('aria-hidden', 'true'); measure.textContent = 'MMMMMMMMMMMMMMMM'; root.append(measure);
  let controller: DrawerContent | undefined, departed = false, width = saved?.width !== undefined ? Math.max(35, Math.min(65, saved.width)) : 50;
  let surface: 'forge' | 'terminal' = 'forge', compact = false, terminalMinimum = 0, nativeFocus: HTMLElement | undefined;
  const covered = new Map<HTMLElement, boolean>();
  const persist = () => {if (storageKey) try {win.sessionStorage.setItem(storageKey, JSON.stringify({repositoryId, open: !drawer.hidden, width}));} catch { /* No credentials or compact visibility. */ }};
  const exposeNative = () => {for (const [element, inert] of covered) {element.inert = inert; delete element.dataset.sodaNativeCovered;} covered.clear();};
  const coverNative = (parent: Element) => {
    for (const element of parent.children) {
      if (!(element instanceof browser.HTMLElement) || element === root || ['SCRIPT', 'STYLE', 'LINK'].includes(element.tagName)) continue;
      if (element.contains(root)) {coverNative(element); continue;}
      if (!covered.has(element)) covered.set(element, element.inert);
      element.inert = true; element.dataset.sodaNativeCovered = '';
    }
  };
  const size = () => {
    const available = win.visualViewport?.width || win.innerWidth, cell = measure.getBoundingClientRect().width / 16 || 9;
    const geometry = workspaceWidths(available, Math.max(cell * 56 + 28, terminalMinimum), width), wasCompact = compact;
    compact = geometry.compact;
    if (compact && !wasCompact && doc.activeElement instanceof browser.HTMLElement && doc.activeElement !== doc.body && !root.contains(doc.activeElement)) surface = 'forge';
    doc.body.style.setProperty('--soda-space-width', geometry.actual + 'vw');
    doc.body.style.setProperty('--soda-viewport-height', (win.visualViewport?.height || win.innerHeight) + 'px');
    doc.body.style.setProperty('--soda-viewport-top', (win.visualViewport?.offsetTop || 0) + 'px');
    doc.body.classList.toggle('sodaspaces-compact', !drawer.hidden && compact);
    switcher.hidden = drawer.hidden || !compact;
    const visible = !drawer.hidden && (!compact || surface === 'terminal');
    drawer.inert = !visible; drawer.toggleAttribute('data-surface-hidden', !visible); drawer.setAttribute('aria-hidden', String(!visible));
    forge.setAttribute('aria-pressed', String(surface === 'forge')); terminal.setAttribute('aria-pressed', String(surface === 'terminal'));
    if (!drawer.hidden && compact && visible) coverNative(doc.body); else exposeNative();
    controller?.setVisible?.(visible);
    divider.setAttribute('aria-valuemin', String(Math.round(geometry.minimum * 10) / 10)); divider.setAttribute('aria-valuemax', String(Math.round(geometry.maximum * 10) / 10)); divider.setAttribute('aria-valuenow', String(Math.round(geometry.actual * 10) / 10));
  };
  let generation = 0, loading: Promise<void> | undefined;
  const mount = () => {
    if (controller) return;
    const visible = !compact || surface === 'terminal';
    if (mountContent) {controller = mountContent(content, context); controller.setVisible?.(visible); void controller.refresh(); return;}
    if (loading) return loading;
    const epoch = generation;
    loading = import('./sodaspaces-workspace.js').then(module => {
      if (departed || generation !== epoch || drawer.hidden) return;
      controller = module.mountSodaspaces(content, context); controller.setVisible?.(!compact || surface === 'terminal'); void controller.refresh();
    }).catch(() => {if (!departed && generation === epoch && !drawer.hidden) content.textContent = 'Workspace could not load. Reload the page; no action was sent.';})
      .finally(() => {if (generation === epoch) loading = undefined;});
    return loading;
  };
  const show = (deliberate = false, intent = false) => {
    if (departed) return;
    if (deliberate || intent) surface = 'terminal';
    drawer.hidden = false; doc.body.classList.add('sodaspaces-open'); size(); button.setAttribute('aria-expanded', 'true');
    // Explicit drawer intent wins over the focus-sensitive compact transition.
    if (deliberate || intent) {surface = 'terminal'; size();}
    const mounted = mount(); persist();
    if (deliberate) {
      const epoch = generation;
      const focus = () => {
        if (departed || generation !== epoch || drawer.hidden || compact && surface !== 'terminal') return;
        if (doc.activeElement !== button && doc.activeElement !== close && doc.activeElement !== doc.body && !content.contains(doc.activeElement)) return;
        void controller?.returnToWork?.(); (content.querySelector<HTMLElement>('[role=tab][aria-selected=true]') || close).focus();
      };
      if (mounted || controller?.ready) void Promise.resolve(mounted).then(() => controller?.ready).then(focus).catch(() => {if (!departed && generation === epoch && !drawer.hidden) {release(); content.textContent = 'Workspace could not render. Reload; no action was replayed.';}}); else focus();
    }
  };
  const hide = () => {void controller?.retain?.(); drawer.hidden = true; doc.body.classList.remove('sodaspaces-open'); size(); button.setAttribute('aria-expanded', 'false'); persist(); button.focus();};
  const release = () => {++generation; loading = undefined; controller?.dispose(); controller = undefined; content.replaceChildren();};
  button.addEventListener('click', () => {if (drawer.hidden) show(true); else if (!departed) {surface = 'terminal'; size();}}, options); close.addEventListener('click', hide, options);
  forge.addEventListener('click', () => {surface = 'forge'; size(); if (nativeFocus?.isConnected && !nativeFocus.inert) nativeFocus.focus({preventScroll: true});}, options);
  terminal.addEventListener('click', () => {surface = 'terminal'; size();}, options);
  doc.addEventListener('focusin', event => {if (event.target instanceof browser.HTMLElement && !root.contains(event.target)) nativeFocus = event.target;}, options);
  root.addEventListener('soda-workspace-minimum', event => {if (event instanceof browser.CustomEvent && typeof event.detail === 'number' && Number.isFinite(event.detail) && event.detail > 0 && event.detail < 10000) {terminalMinimum = event.detail; size();}}, options);
  const resize = (desired: number) => {const geometry = workspaceWidths(win.visualViewport?.width || win.innerWidth, Math.max(terminalMinimum, (measure.getBoundingClientRect().width / 16 || 9) * 56 + 28), width); if (!geometry.compact) {width = Math.max(geometry.minimum, Math.min(geometry.maximum, desired)); size(); persist();}};
  divider.addEventListener('pointerdown', e => {if (e.button === 0) {divider.setPointerCapture(e.pointerId); e.preventDefault();}}, options);
  divider.addEventListener('pointermove', e => {if (divider.hasPointerCapture(e.pointerId)) resize((win.innerWidth - e.clientX) / win.innerWidth * 100);}, options);
  divider.addEventListener('pointerup', e => {if (divider.hasPointerCapture(e.pointerId)) divider.releasePointerCapture(e.pointerId);}, options);
  divider.addEventListener('keydown', e => {if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) {e.preventDefault(); resize(e.key === 'Home' ? 35 : e.key === 'End' ? 65 : width + (e.key === 'ArrowLeft' ? 5 : -5));}}, options);
  const observer = new browser.MutationObserver(() => {if (!drawer.hidden && compact && surface === 'terminal') coverNative(doc.body);}); observer.observe(doc.body, {childList: true});
  win.addEventListener('resize', size, options); win.visualViewport?.addEventListener('resize', size, options); win.visualViewport?.addEventListener('scroll', size, options);
  doc.fonts?.addEventListener('loadingdone', size, options); void doc.fonts?.ready.then(size);
  win.addEventListener('pagehide', () => {departed = true; release(); surface = 'forge'; size();}, options);
  win.addEventListener('pageshow', event => {if (event.persisted) {departed = false; release(); surface = 'forge'; if (!drawer.hidden) show();}}, options);
  if (saved?.open || win.location.hash === '#sodaspaces') show(false, win.location.hash === '#sodaspaces');
  return {dispose() {release(); lifetime.abort(); observer.disconnect(); exposeNative(); switcher.remove(); measure.remove(); drawer.hidden = true; drawer.inert = false; doc.body.classList.remove('sodaspaces-open', 'sodaspaces-compact'); for (const prop of ['--soda-space-width', '--soda-viewport-height', '--soda-viewport-top']) doc.body.style.removeProperty(prop);}};
}
if (typeof document !== 'undefined') {if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => mountDrawer(document), {once: true}); else mountDrawer(document);}
