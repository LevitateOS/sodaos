// Native navigation remains native. Only a same-user workspace locator/layout is
// retained; every document obtains fresh Soda authorization before attachment.
import type { DrawerContext } from './sodaspaces-drawer.js';
import {object} from './sodaspaces-api.js';
const identifier = (v: unknown): v is string => typeof v === 'string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v) <= 9223372036854775807n;

export interface DrawerContent { readonly ready?: Promise<unknown>; refresh(): void | Promise<void>; dispose(): void; retain?(): void | Promise<void> | undefined; returnToWork?(): void | Promise<void> | undefined }
export function mountDrawer(doc: Document, mountContent?: (root: HTMLElement, context: Extract<DrawerContext, {kind: 'native'}>) => DrawerContent) {
  const roots = doc.querySelectorAll<HTMLElement>('#sodaspaces-root'), rows = doc.querySelectorAll('.repo-header .repo-buttons');
  if (roots.length !== 1 || rows.length > 1) return;
  const root = roots[0], win = doc.defaultView;
  if (!root || !win || root.dataset.mounted) return;
  const {repositoryId: nativeRepository, userId, signed, subUrl} = root.dataset;
  if (subUrl !== '' || (nativeRepository !== '' && !identifier(nativeRepository)) || (signed !== 'true' && signed !== 'false') || (signed === 'true' ? !identifier(userId) : userId !== '')) return;
  const storageKey = signed === 'true' ? 'soda-workspace:' + userId : null;
  let saved: {repositoryId: string; open: boolean; width?: number} | undefined;
  try {
    const raw = storageKey ? win.sessionStorage.getItem(storageKey) : null;
    if (raw) {
      const value = object(JSON.parse(raw));
      if (identifier(value.repositoryId) && typeof value.open === 'boolean') saved = {repositoryId: value.repositoryId, open: value.open, ...(typeof value.width === 'number' && Number.isFinite(value.width) ? {width: value.width} : {})};
    }
  } catch { /* untrusted locator, not authority */ }
  if (rows.length === 0 && !saved) return;
  const selectedRepository = saved?.repositoryId || nativeRepository;
  if (!identifier(selectedRepository)) return;
  const repositoryId: string = selectedRepository;
  const context: DrawerContext = {kind: 'native', expectedUserId: signed === 'true' ? userId : undefined, repositoryId: identifier(nativeRepository) ? nativeRepository : repositoryId};
  const button = root.querySelector<HTMLElement>('#sodaspaces-button'), drawer = root.querySelector<HTMLElement>('#sodaspaces-drawer');
  const close = root.querySelector<HTMLElement>('#sodaspaces-close'), content = root.querySelector<HTMLElement>('#sodaspaces-content'), divider = root.querySelector<HTMLElement>('#sodaspaces-divider');
  if (!button || !drawer || !close || !content || !divider) return;
  const lifetime = new (win as Window & typeof globalThis).AbortController(), options = {signal: lifetime.signal};
  root.dataset.mounted = 'true';
  if (rows.length) rows.item(0).append(button); else button.classList.add('sodaspaces-global-resume');
  root.hidden = false;
  let controller: DrawerContent | undefined, departed = false, width = saved?.width !== undefined ? Math.max(35, Math.min(65, saved.width)) : 50;
  const persist = () => { if (storageKey) try { win.sessionStorage.setItem(storageKey, JSON.stringify({repositoryId, open: !drawer.hidden, width})); } catch { /* no credential storage */ } };
  const size = () => { doc.body.style.setProperty('--soda-space-width', width + 'vw'); divider.setAttribute('aria-valuenow', String(width)); };
  let generation = 0;
  let loading: Promise<void> | undefined;
  const mount = () => {
    if (controller) return;
    if (mountContent) {controller = mountContent(content, context); void controller.refresh(); return;}
    if (loading) return loading;
    const epoch = generation;
    loading = import('./sodaspaces-drawer.js').then(module => {
      if (departed || generation !== epoch || drawer.hidden) return;
      controller = module.mountSodaspaces(content, context);
      void controller.refresh();
    }).catch(() => {
      if (!departed && generation === epoch && !drawer.hidden) content.textContent = 'Workspace could not load. Reload the page; no action was sent.';
    }).finally(() => {if (generation === epoch) loading = undefined;});
    return loading;
  };
  const show = (deliberate = false) => {
    if (departed) return;
    drawer.hidden = false; doc.body.classList.add('sodaspaces-open'); size(); button.setAttribute('aria-expanded', 'true');
    const mounted = mount(); persist();
    if (deliberate) {
      const epoch = generation;
      const focus = () => {
        if (departed || generation !== epoch || drawer.hidden) return;
        if (doc.activeElement !== button && doc.activeElement !== close && doc.activeElement !== doc.body && !content.contains(doc.activeElement)) return;
        void controller?.returnToWork?.();
        (content.querySelector<HTMLElement>('[role=tab][aria-selected=true]') || close).focus();
      };
      if (mounted || controller?.ready) void Promise.resolve(mounted).then(() => controller?.ready).then(focus).catch(() => {
        if (!departed && generation === epoch && !drawer.hidden) {release(); content.textContent = 'Workspace could not render. Reload the page; no action was replayed.';}
      });
      else focus();
    }
  };
  const hide = () => {
    void controller?.retain?.(); drawer.hidden = true; doc.body.classList.remove('sodaspaces-open');
    button.setAttribute('aria-expanded', 'false'); persist(); button.focus();
  };
  const release = () => { ++generation; loading = undefined; controller?.dispose(); controller = undefined; content.replaceChildren(); };
  button.addEventListener('click', () => { if (drawer.hidden) show(true); }, options);
  close.addEventListener('click', hide, options);
  divider.addEventListener('pointerdown', e => { if (e.button !== 0) return; divider.setPointerCapture(e.pointerId); e.preventDefault(); }, options);
  divider.addEventListener('pointermove', e => {
    if (!divider.hasPointerCapture(e.pointerId)) return;
    width = Math.round(Math.max(35, Math.min(65, (win.innerWidth - e.clientX) / win.innerWidth * 100))); size(); persist();
  }, options);
  divider.addEventListener('pointerup', e => { if (divider.hasPointerCapture(e.pointerId)) divider.releasePointerCapture(e.pointerId); }, options);
  divider.addEventListener('keydown', e => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) return;
    e.preventDefault(); width = e.key === 'Home' ? 35 : e.key === 'End' ? 65 : Math.max(35, Math.min(65, width + (e.key === 'ArrowLeft' ? 5 : -5))); size(); persist();
  }, options);
  win.addEventListener('pagehide', () => { departed = true; release(); }, options);
  win.addEventListener('pageshow', event => { if (event.persisted) { departed = false; release(); if (!drawer.hidden) show(); } }, options);
  if (saved?.open || win.location.hash === '#sodaspaces') show();
  return {dispose() { release(); lifetime.abort(); drawer.hidden = true; doc.body.classList.remove('sodaspaces-open'); doc.body.style.removeProperty('--soda-space-width'); }};
}
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => mountDrawer(document), {once: true});
  else mountDrawer(document);
}
