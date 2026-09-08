// Native shell only. The content module owns API actions; Forgejo owns its page.
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
  const button = root.querySelector('#sodaspaces-button'), dialog = root.querySelector('#sodaspaces-drawer');
  const close = root.querySelector('#sodaspaces-close'), content = root.querySelector('#sodaspaces-content');
  if (!button || !dialog || !close || !content || typeof dialog.showModal !== 'function') return;
  root.dataset.mounted = 'true'; rows[0].append(button); root.hidden = false;
  let controller, ended = false;
  const retire = () => {
    if (ended) return;
    ended = true; controller?.dispose(); controller = undefined; content.replaceChildren();
    const message = doc.createElement('p'); message.id = 'sodaspaces-status'; message.setAttribute('role', 'status');
    message.textContent = 'This drawer session ended. Reload the repository page to continue. Closing or switching away does not undo an action already sent.';
    const reload = doc.createElement('button'); reload.type = 'button'; reload.className = 'ui primary button';
    reload.id = 'sodaspaces-reload'; reload.textContent = 'Reload repository page'; reload.addEventListener('click', () => win.location.reload());
    const actor = doc.createElement('p'); actor.id = 'sodaspaces-actor';
    content.setAttribute('aria-busy', 'false'); content.append(message, actor, reload);
  };
  const shut = () => { retire(); dialog.close(); };
  button.addEventListener('click', () => {
    if (dialog.open) return;
    dialog.showModal(); button.setAttribute('aria-expanded', 'true'); close.focus();
    if (ended) return;
    controller = mountContent(content, {expectedUserId: signed === 'true' ? userId : undefined, repositoryId});
    void controller.refresh();
  });
  close.addEventListener('click', shut);
  dialog.addEventListener('click', event => { if (event.target === dialog) shut(); });
  dialog.addEventListener('cancel', () => retire());
  dialog.addEventListener('close', () => {
    if (dialog.open) return;
    retire(); button.setAttribute('aria-expanded', 'false'); button.focus();
  });
  win.addEventListener('blur', retire);
  win.addEventListener('pagehide', retire);
  doc.addEventListener('visibilitychange', () => { if (doc.hidden) retire(); });
  win.addEventListener('pageshow', event => { if (event.persisted) retire(); });
  if (doc.hidden) retire();
  if (win.location.hash === '#sodaspaces') button.click();
  return {dispose: retire};
}
if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => mountDrawer(document), {once: true});
  else mountDrawer(document);
}
