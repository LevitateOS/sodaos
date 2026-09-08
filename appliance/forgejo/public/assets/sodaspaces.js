// Native page and non-modal workspace share the viewport. Never intercept navigation.
import {mountSodaspaces} from './sodaspaces-drawer.js';
const identifier = v => typeof v === 'string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v) <= 9223372036854775807n;

export function mountDrawer(doc, mountContent = mountSodaspaces) {
  const roots = doc.querySelectorAll('#sodaspaces-root');
  const rows = doc.querySelectorAll('.repo-header .repo-buttons');
  if (roots.length !== 1 || rows.length !== 1) return;
  const root = roots[0], win = doc.defaultView;
  if (root.dataset.mounted) return;
  const {repositoryId, userId, signed, subUrl} = root.dataset;
  if (subUrl !== '' || !identifier(repositoryId) || !['true', 'false'].includes(signed) ||
      (signed === 'true' ? !identifier(userId) : userId !== '')) return;
  const button = root.querySelector('#sodaspaces-button'), drawer = root.querySelector('#sodaspaces-drawer');
  const close = root.querySelector('#sodaspaces-close'), content = root.querySelector('#sodaspaces-content');
  const divider = root.querySelector('#sodaspaces-divider');
  if (!button || !drawer || !close || !content || !divider) return;
  const lifetime = new win.AbortController(), options = {signal: lifetime.signal};
  root.dataset.mounted = 'true'; rows[0].append(button); root.hidden = false;
  let controller, ended = false, width = 50;
  const size = () => {
    doc.body.style.setProperty('--soda-space-width', width + 'vw');
    divider.setAttribute('aria-valuenow', String(width));
  };
  const hide = () => {
    drawer.hidden = true; doc.body.classList.remove('sodaspaces-open');
    button.setAttribute('aria-expanded', 'false'); button.focus();
  };
  const retire = () => {
    if (ended) return;
    ended = true; controller?.dispose(); controller = undefined; content.replaceChildren();
    const message = doc.createElement('p'); message.id = 'sodaspaces-status'; message.setAttribute('role', 'status');
    message.textContent = 'This page context ended. Reload to continue. Reconnection across navigation is not available yet; no action was replayed or undone.';
    const reload = doc.createElement('button'); reload.type = 'button'; reload.className = 'ui primary button';
    reload.id = 'sodaspaces-reload'; reload.textContent = 'Reload repository page'; reload.addEventListener('click', () => win.location.reload(), options);
    content.setAttribute('aria-busy', 'false'); content.append(message, reload);
  };
  button.addEventListener('click', () => {
    if (!drawer.hidden) return;
    drawer.hidden = false; doc.body.classList.add('sodaspaces-open'); size();
    button.setAttribute('aria-expanded', 'true');
    if (!ended && !controller) {
      controller = mountContent(content, {expectedUserId: signed === 'true' ? userId : undefined, repositoryId});
      void controller.refresh();
    }
    (content.querySelector('[role=tab][aria-selected=true]') || close).focus();
  }, options);
  close.addEventListener('click', hide, options);
  // Pointer capture keeps native text/forms untouched; the separator is also keyboard operable.
  divider.addEventListener('pointerdown', e => {
    if (e.button !== 0) return;
    divider.setPointerCapture(e.pointerId); e.preventDefault();
  }, options);
  divider.addEventListener('pointermove', e => {
    if (!divider.hasPointerCapture(e.pointerId)) return;
    width = Math.round(Math.max(35, Math.min(65, (win.innerWidth - e.clientX) / win.innerWidth * 100))); size();
  }, options);
  divider.addEventListener('pointerup', e => { if (divider.hasPointerCapture(e.pointerId)) divider.releasePointerCapture(e.pointerId); }, options);
  divider.addEventListener('keydown', e => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) return;
    e.preventDefault();
    width = e.key === 'Home' ? 35 : e.key === 'End' ? 65 : Math.max(35, Math.min(65, width + (e.key === 'ArrowLeft' ? 5 : -5))); size();
  }, options);
  // Until server-owned reattachment lands, real document departure still retires.
  // Focus, visibility and hiding the workspace are not document/authentication loss.
  win.addEventListener('pagehide', retire, options);
  win.addEventListener('pageshow', event => { if (event.persisted) retire(); }, options);
  if (win.location.hash === '#sodaspaces') button.click();
  return {dispose() {retire(); lifetime.abort(); drawer.hidden = true; doc.body.classList.remove('sodaspaces-open'); doc.body.style.removeProperty('--soda-space-width');}};
}
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => mountDrawer(document), {once: true});
  else mountDrawer(document);
}
