// Native navigation remains native. Only a same-user workspace locator/layout is
// retained; every document obtains fresh Soda authorization before attachment.
import {mountSodaspaces} from './sodaspaces-drawer.js';
const identifier = v => typeof v === 'string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v) <= 9223372036854775807n;

export function mountDrawer(doc, mountContent = mountSodaspaces) {
  const roots = doc.querySelectorAll('#sodaspaces-root'), rows = doc.querySelectorAll('.repo-header .repo-buttons');
  if (roots.length !== 1 || rows.length > 1) return;
  const root = roots[0], win = doc.defaultView;
  if (root.dataset.mounted) return;
  const {repositoryId: nativeRepository, userId, signed, subUrl} = root.dataset;
  if (subUrl !== '' || (nativeRepository !== '' && !identifier(nativeRepository)) || !['true', 'false'].includes(signed) || (signed === 'true' ? !identifier(userId) : userId !== '')) return;
  const storageKey = signed === 'true' ? 'soda-workspace:' + userId : null;
  let saved;
  try { saved = JSON.parse(win.sessionStorage.getItem(storageKey)); } catch { /* untrusted locator, not authority */ }
  if (!storageKey || !saved || !identifier(saved.repositoryId) || typeof saved.open !== 'boolean') saved = null;
  if (rows.length === 0 && !saved) return;
  let repositoryId = saved?.repositoryId || nativeRepository;
  if (!identifier(repositoryId)) return;
  const button = root.querySelector('#sodaspaces-button'), drawer = root.querySelector('#sodaspaces-drawer');
  const close = root.querySelector('#sodaspaces-close'), content = root.querySelector('#sodaspaces-content'), divider = root.querySelector('#sodaspaces-divider');
  if (!button || !drawer || !close || !content || !divider) return;
  const lifetime = new win.AbortController(), options = {signal: lifetime.signal};
  root.dataset.mounted = 'true';
  if (rows.length) rows[0].append(button); else button.classList.add('sodaspaces-global-resume');
  root.hidden = false;
  let controller, departed = false, width = Number.isFinite(saved?.width) ? Math.max(35, Math.min(65, saved.width)) : 50;
  const persist = () => { if (storageKey) try { win.sessionStorage.setItem(storageKey, JSON.stringify({repositoryId, open: !drawer.hidden, width})); } catch { /* no credential storage */ } };
  const size = () => { doc.body.style.setProperty('--soda-space-width', width + 'vw'); divider.setAttribute('aria-valuenow', String(width)); };
  const mount = () => {
    if (!controller) { controller = mountContent(content, {expectedUserId: signed === 'true' ? userId : undefined, repositoryId}); void controller.refresh(); }
  };
  const show = (deliberate = false) => {
    if (departed) return;
    drawer.hidden = false; doc.body.classList.add('sodaspaces-open'); size(); button.setAttribute('aria-expanded', 'true');
    mount(); persist();
    if (deliberate) { void controller?.returnToWork?.(); (content.querySelector('[role=tab][aria-selected=true]') || close).focus(); }
  };
  const hide = () => {
    void controller?.retain?.(); drawer.hidden = true; doc.body.classList.remove('sodaspaces-open');
    button.setAttribute('aria-expanded', 'false'); persist(); button.focus();
  };
  const release = () => { controller?.dispose(); controller = undefined; content.replaceChildren(); };
  button.addEventListener('click', () => { if (drawer.hidden) show(true); }, options);
  close.addEventListener('click', hide, options);
  // Repository navigation alone must not retarget a shell. Switching workspaces
  // is explicit, detaches the old view and never creates/joins/starts a project.
  if (identifier(nativeRepository) && repositoryId !== nativeRepository) {
    const select = doc.createElement('button'); select.type = 'button'; select.className = 'ui basic button'; select.textContent = 'Use repository on the left';
    rows[0]?.append(select);
    select.addEventListener('click', () => {
      void controller?.retain?.(); release(); repositoryId = nativeRepository; select.remove(); show(true);
    }, options);
  }
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
