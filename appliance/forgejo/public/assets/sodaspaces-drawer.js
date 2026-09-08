// All content is owned by one supplied mount. No Forgejo selectors/templates,
// navigation replacement, hidden commands, mutation replay or lifecycle repair.
import {mountTerminal} from './sodaspaces-terminal.js';
const id = v => typeof v === 'string' && /^[1-9][0-9]{0,18}$/.test(v) && BigInt(v) <= 9223372036854775807n;
const projectId = v => typeof v === 'string' && /^p[0-9a-f]{24}$/.test(v);
const fingerprint = v => typeof v === 'string' && /^SHA256:[A-Za-z0-9+/]{43}$/.test(v);
const rejected = new Set([400, 401, 403, 404, 409, 413, 415, 422]);

export function mountSodaspaces(root, {expectedUserId, repositoryId}, terminalFactory = mountTerminal) {
  if (!root || !id(expectedUserId) || !id(repositoryId)) throw Error('Invalid native page context');
  const doc = root.ownerDocument, win = doc.defaultView;
  const lifetime = new win.AbortController();
  const box = doc.createElement('section'); box.className = 'soda-spaces-controls'; root.append(box);
  const node = (tag, text, parent = box) => { const n = doc.createElement(tag); if (text) n.textContent = text; parent.append(n); return n; };
  const status = node('p', 'Refresh to inspect your shared environment.'); status.setAttribute('role', 'status');
  const outcome = node('p'); outcome.setAttribute('role', 'status');
  node('p', 'Actions are separate. Closing or switching away does not undo native work. Stop affects everyone; saved keys and installed access are different.');
  const actor = node('p');
  const connect = node('a', 'Connect to Soda');
  connect.href = '/-/soda/login?' + new URLSearchParams({expected_user_id: expectedUserId, repository_id: repositoryId});
  const buttons = node('div');
  const button = (text, parent = buttons) => { const b = node('button', text, parent); b.type = 'button'; return b; };
  const refreshButton = button('Refresh status'), reload = button('Reload repository page'), signOut = button('Sign out of Soda');
  const create = button('Create environment'), join = button('Join environment');
  const life = node('fieldset'); node('legend', 'Shared environment', life);
  const runtime = node('p', '', life), start = button('Start', life), stop = button('Stop', life);
  const stopLabel = node('label', '', life), stopConfirm = doc.createElement('input'); stopConfirm.type = 'checkbox'; stopLabel.append(stopConfirm, ' I understand Stop interrupts everyone’s SSH, terminals and workloads, and disables next-boot start.');
  const keySection = node('fieldset'); node('legend', 'My development SSH keys (not Forgejo Git keys)', keySection);
  const keyList = node('ul', '', keySection);
  const keyLabel = node('label', 'Public SSH key ', keySection), publicKey = doc.createElement('textarea'); publicKey.rows = 3; publicKey.maxLength = 48000; keyLabel.append(publicKey);
  const save = button('Save public key', keySection);
  node('p', 'Removing a saved key changes future joins only. Use Review → Apply below for this project. Verify a replacement over SSH before revoking the old key.', keySection);
  const review = button('Review this project’s SSH keys', keySection);
  const previewArea = node('div', '', keySection);
  const emptyLabel = node('label', '', keySection), emptyConfirm = doc.createElement('input'); emptyConfirm.type = 'checkbox'; emptyLabel.append(emptyConfirm, ' Remove all managed keys from this account: new SSH logins using them will be denied.');
  const apply = button('Apply reviewed saved keys to this project', keySection);
  const connection = node('fieldset'); node('legend', 'SSH / editor connection', connection);
  const sshCommand = node('code', '', connection), hostFingerprint = node('p', '', connection), copy = button('Copy SSH connection', connection);
  node('p', 'Use ordinary SSH or your editor’s Remote SSH with this account/IP. An observed IP is not proof of laptop routing.', connection);
  const terminalMount = node('div');
  let session, environment, detail, keyPreview, terminal, readController;
  let busy = false, stale = false, disposed = false, uncertain = false, epoch = 0;
  const commandButtons = [create, join, start, stop, save, review, apply, copy];
  const reset = () => {
    terminal?.dispose(); terminal = undefined;
    environment = detail = keyPreview = undefined;
    publicKey.value = ''; keyList.replaceChildren(); previewArea.replaceChildren();
    sshCommand.textContent = hostFingerprint.textContent = runtime.textContent = '';
    create.hidden = join.hidden = life.hidden = keySection.hidden = connection.hidden = true;
    review.hidden = apply.hidden = emptyLabel.hidden = true; stopConfirm.checked = emptyConfirm.checked = false;
  };
  const updateBusy = () => {
    for (const b of commandButtons) b.disabled = busy || stale || uncertain;
    for (const b of keyList.querySelectorAll('button')) b.disabled = busy || stale || uncertain;
    refreshButton.disabled = busy || stale;
    signOut.disabled = busy || stale || !session;
  };
  const invalidate = () => {
    ++epoch; stale = true; readController?.abort(); reset(); session = undefined;
    actor.textContent = ''; signOut.hidden = true; connect.hidden = true;
    status.textContent = 'Page context changed. Reload the full repository page; no action was replayed or undone.'; updateBusy();
  };
  const active = n => !disposed && !stale && epoch === n;
  const api = async (path, method = 'GET', body, signal) => {
    const headers = {'X-Soda-Expected-User-ID': method === 'POST' && path === '/api/session/logout' ? session.user.id : expectedUserId};
    if (method !== 'GET') Object.assign(headers, {'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token});
    // Bootstrap may reveal a different Soda actor solely for explicit local logout.
    if (path === '/api/session') delete headers['X-Soda-Expected-User-ID'];
    const response = await win.fetch('/-/soda' + path, {method, headers, body: body === undefined ? undefined : JSON.stringify(body), credentials: 'same-origin', cache: 'no-store', redirect: 'error', signal});
    if (!response.ok) { const e = Error('Soda request failed'); e.status = response.status; throw e; }
    return response.status === 204 ? null : response.json();
  };
  const check = condition => { if (!condition) throw Error('Invalid or mismatched Soda response'); };
  async function refresh() {
    if (busy || stale || disposed) return;
    ++epoch; const n = epoch; readController?.abort(); reset();
    readController = new win.AbortController(); const timeout = win.setTimeout(() => readController?.abort(), 15000);
    busy = true; session = undefined; connect.hidden = signOut.hidden = true; updateBusy(); status.textContent = 'Checking your account and environment…';
    try {
      const found = await api('/api/session', 'GET', undefined, readController.signal); if (!active(n)) return;
      check(id(found.user?.id) && typeof found.user.login === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(found.csrf_token) && found.forgejo_url === win.location.origin);
      session = found; actor.textContent = `Soda account: ${found.user.login} (ID ${found.user.id})`; signOut.hidden = false;
      if (found.user.id !== expectedUserId) { connect.hidden = false; status.textContent = 'Soda and native page identities differ. Sign out of Soda or reload and connect explicitly.'; return; }
      const provider = await api('/api/forgejo/me', 'GET', undefined, readController.signal); if (!active(n)) return; check(provider.id === expectedUserId);
      const collection = await api('/api/environments?repository_id=' + repositoryId, 'GET', undefined, readController.signal); if (!active(n)) return;
      check(collection.repository?.id === repositoryId && Array.isArray(collection.items) && collection.items.length <= 1 && typeof collection.can_create === 'boolean');
      if (collection.items.length === 0) { create.hidden = !collection.can_create; status.textContent = 'No shared environment. Creation is owner-only and does not join you.'; return; }
      environment = collection.items[0]; check(projectId(environment.id) && environment.repository_id === repositoryId);
      detail = await api(`/api/environments/${environment.id}`, 'GET', undefined, readController.signal); if (!active(n)) return;
      check(detail.environment?.id === environment.id && detail.environment.repository_id === repositoryId && typeof detail.environment.provisioned === 'boolean' && typeof detail.login === 'string' && typeof detail.environment_administrator === 'boolean');
      const running = detail.observed?.running === true;
      status.textContent = !detail.environment.provisioned ? 'Provisioning incomplete. Ask the operator to inspect; do not recreate it.' : detail.native_unavailable ? 'Native state unavailable; refresh or ask the operator to inspect.' : running ? 'Environment running.' : 'Environment stopped.';
      if (detail.environment.provisioned && !running && !detail.native_unavailable && !detail.environment_administrator) status.textContent += ' Ask the project administrator or Soda operator to Start it; nothing is started automatically.';
      const saved = await api('/api/me/development-keys', 'GET', undefined, readController.signal); if (!active(n)) return;
      check(Array.isArray(saved.items) && saved.items.every(k => id(k.id) && fingerprint(k.fingerprint)));
      keySection.hidden = false;
      for (const k of saved.items) {
        const li = node('li', k.fingerprint + ' ', keyList), remove = button('Remove saved key', li);
        remove.addEventListener('click', () => mutate(`/api/me/development-keys/${k.id}`, {}, 'Saved key removed. Existing project SSH access is unchanged until explicitly applied.', 'DELETE'), {signal: lifetime.signal});
      }
      join.hidden = !detail.environment.provisioned || !!detail.login || !running || saved.items.length === 0;
      if (!detail.login) node('p', saved.items.length ? 'Start must be requested from the project administrator when stopped; then explicitly Join.' : 'Save your public key, then explicitly Join when the environment is running.', keyList);
      if (detail.environment.provisioned && detail.environment_administrator) {
        const state = await api(`/api/environments/${environment.id}/lifecycle`, 'GET', undefined, readController.signal); if (!active(n)) return;
        check(state.environment?.id === environment.id && typeof state.environment.running === 'boolean' && typeof state.boot_enabled === 'boolean');
        life.hidden = false; start.hidden = state.environment.running && state.boot_enabled; stop.hidden = !state.environment.running && !state.boot_enabled;
        stopLabel.hidden = stop.hidden; runtime.textContent = `Running: ${state.environment.running ? 'yes' : 'no'}; starts on host boot: ${state.boot_enabled ? 'yes' : 'no'}. Start restores boot start; Stop disables it.`;
      }
      if (detail.login && running) {
        check(/^[a-z][a-z0-9_-]{0,30}$/.test(detail.login) && detail.login !== 'root');
        review.hidden = false;
        const own = await api(`/api/environments/${environment.id}/connection`, 'GET', undefined, readController.signal); if (!active(n)) return;
        const c = own.connection;
        check(own.login === detail.login && c?.environment?.id === environment.id && c.environment.running && typeof c.environment.ip === 'string' && /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(c.environment.ip) && c.environment.ip.split('.').every(v => Number(v) <= 255) && fingerprint(c.fingerprint));
        connection.hidden = false; sshCommand.textContent = `ssh ${detail.login}@${c.environment.ip}`; hostFingerprint.textContent = c.fingerprint;
        terminal = terminalFactory(terminalMount, {expectedUserId, repositoryId, environmentId: environment.id, login: detail.login});
      }
    } catch (e) {
      if (!active(n)) return;
      reset(); actor.textContent = ''; session = undefined; signOut.hidden = true; connect.hidden = !(e.status === 401 || e.status === 403);
      status.textContent = connect.hidden ? 'Could not confirm state. Refresh; do not infer absence or retry an uncertain action.' : 'Connect through Forgejo to authorize Soda access.';
    } finally { win.clearTimeout(timeout); if (active(n)) { busy = false; updateBusy(); } }
  }
  async function mutate(path, body, message, method = 'POST') {
    if (busy || stale || disposed || (uncertain && path !== '/api/session/logout') || !session) return;
    const n = epoch; busy = true; updateBusy(); outcome.textContent = 'Request dispatched. Closing does not cancel or undo native work.';
    const controller = new win.AbortController(), timeout = win.setTimeout(() => controller.abort(), 255000);
    try {
      const result = await api(path, method, body, controller.signal);
      if (!active(n)) return;
      if (path === '/api/environments') check(projectId(result?.id) && result.repository_id === repositoryId && result.provisioned === true);
      else if (path.endsWith('/join')) check(typeof result?.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(result.login) && result.login !== 'root');
      else if (path.endsWith('/lifecycle')) check(result?.environment?.id === environment.id && result.environment.running === (body.action === 'start') && result.boot_enabled === (body.action === 'start'));
      else if (path.endsWith('/access-keys')) check(result?.applied === true && result.login === detail.login && /^[0-9a-f]{64}$/.test(result.revision) && JSON.stringify(result.installed_fingerprints) === JSON.stringify(body.saved_fingerprints));
      else if (method === 'DELETE') check(result?.removed === true && result.existing_project_access_changed === false);
      else if (path === '/api/me/development-keys') check(Array.isArray(result?.items) && result.items.every(k => id(k.id) && fingerprint(k.fingerprint)));
      else if (path === '/api/session/logout') check(result === null);
      outcome.textContent = message;
      if (path === '/api/session/logout') { invalidate(); status.textContent = 'Signed out of Soda, not Forgejo or Linux. Reload to connect again.'; return; }
      busy = false; await refresh();
    } catch (e) {
      if (!active(n)) return;
      uncertain = !rejected.has(e.status);
      outcome.textContent = uncertain ? 'Outcome unconfirmed. Ask the operator to inspect; do not repeat, recreate or repair. Refresh reads state only.' : 'Request rejected. Refresh and review current state before another explicit action.';
    } finally { win.clearTimeout(timeout); if (active(n)) { busy = false; updateBusy(); } }
  }
  const on = (button, fn) => button.addEventListener('click', fn, {signal: lifetime.signal});
  on(refreshButton, refresh); on(reload, () => win.location.reload());
  on(signOut, () => { terminal?.invalidate(); mutate('/api/session/logout', {}, 'Signed out of Soda.'); });
  on(create, () => mutate('/api/environments', {repository_id: repositoryId}, 'Environment created. Save a public key and explicitly Join; creation does not join you.'));
  on(join, () => mutate(`/api/environments/${environment.id}/join`, {}, 'Native join confirmed. Later saved-key changes require a separate explicit Apply.'));
  on(start, () => mutate(`/api/environments/${environment.id}/lifecycle`, {action: 'start'}, 'Start confirmed and next-boot start enabled. Refresh connection status while services initialize.'));
  on(stop, () => { if (!stopConfirm.checked) { outcome.textContent = 'Confirm the shared impact before Stop.'; return; } terminal?.invalidate(); mutate(`/api/environments/${environment.id}/lifecycle`, {action: 'stop', confirm_stop: true}, 'Stop confirmed; next-boot start disabled. Existing data was not recreated or deleted.'); });
  on(save, () => {
    const value = publicKey.value.trim();
    if (!/^(ssh-|ecdsa-|sk-)/.test(value) || /PRIVATE KEY/.test(value) || /[\r\n]/.test(value)) { outcome.textContent = 'Provide one public SSH key. Never upload a private key.'; return; }
    publicKey.value = ''; mutate('/api/me/development-keys', {public_key: value}, 'Public key saved for future joins. Existing project access is unchanged until explicitly applied.');
  });
  on(review, async () => {
    if (busy || stale || disposed || uncertain) return;
    const n = epoch; busy = true; updateBusy();
    const controller = new win.AbortController(), timeout = win.setTimeout(() => controller.abort(), 15000);
    try {
      const preview = await api(`/api/environments/${environment.id}/access-keys`, 'GET', undefined, controller.signal);
      if (!active(n)) return;
      check(preview.login === detail.login && /^[0-9a-f]{64}$/.test(preview.revision) && Array.isArray(preview.installed_fingerprints) && Array.isArray(preview.saved_fingerprints) && [...preview.installed_fingerprints, ...preview.saved_fingerprints].every(fingerprint));
      keyPreview = preview; previewArea.replaceChildren();
      node('p', 'The dedicated Soda-managed key file for this account will match the saved set. Review every removal; other accounts/files/projects and authenticated SSH sessions are unchanged.', previewArea);
      const removed = preview.installed_fingerprints.filter(k => !preview.saved_fingerprints.includes(k)), added = preview.saved_fingerprints.filter(k => !preview.installed_fingerprints.includes(k));
      node('p', `Add: ${added.join(', ') || 'none'}`, previewArea); node('p', `Remove: ${removed.join(', ') || 'none'}`, previewArea);
      emptyConfirm.checked = false; emptyLabel.hidden = preview.saved_fingerprints.length !== 0; apply.hidden = false;
    } catch { if (active(n)) {keyPreview = undefined; previewArea.replaceChildren(); apply.hidden = emptyLabel.hidden = true; outcome.textContent = 'Key preview unavailable or changed. No update was requested; refresh and inspect.';} }
    finally {win.clearTimeout(timeout); if (active(n)) {busy = false; updateBusy();}}
  });
  on(apply, () => {
    if (!keyPreview) return;
    if (!keyPreview.saved_fingerprints.length && !emptyConfirm.checked) {outcome.textContent = 'Explicitly confirm removal of the last managed key.'; return;}
    mutate(`/api/environments/${environment.id}/access-keys`, {revision: keyPreview.revision, saved_fingerprints: keyPreview.saved_fingerprints, confirm_empty: keyPreview.saved_fingerprints.length === 0}, 'Managed SSH key file updated for this project. Verify new-key login and old-key refusal from your SSH client. Existing sessions and browser access are not revoked.');
  });
  on(copy, async () => {const n = epoch; try {await win.navigator.clipboard.writeText(sshCommand.textContent); if (active(n)) outcome.textContent = 'SSH command copied; this does not prove your client route.';} catch {if (active(n)) outcome.textContent = 'Clipboard unavailable. Select the displayed SSH command manually.';}});
  win.addEventListener('blur', invalidate, {signal: lifetime.signal}); win.addEventListener('pagehide', invalidate, {signal: lifetime.signal});
  doc.addEventListener('visibilitychange', () => {if (doc.visibilityState === 'hidden') invalidate();}, {signal: lifetime.signal});
  win.addEventListener('pageshow', e => {if (e.persisted) invalidate();}, {signal: lifetime.signal});
  reset(); signOut.hidden = true; updateBusy();
  return {refresh, invalidate, dispose() {if (disposed) return; invalidate(); disposed = true; lifetime.abort(); box.remove();}};
}
