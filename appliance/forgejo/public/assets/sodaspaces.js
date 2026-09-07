// SPDX-License-Identifier: Apache-2.0
(() => {
  'use strict';
  function mount() {
    const roots = document.querySelectorAll('#sodaspaces-root');
    const rows = document.querySelectorAll('.repo-header .repo-buttons');
    if (roots.length !== 1 || rows.length !== 1) return;
    const root = roots[0];
    const id = (value) => typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) &&
      (value.length < 19 || value <= '9223372036854775807');
    const text = (value) => typeof value === 'string' && value.length <= 4096;
    const {repositoryId, userId, signed, subUrl} = root.dataset;
    if (root.dataset.mounted || subUrl !== '' || !id(repositoryId) ||
        !['true', 'false'].includes(signed) || (signed === 'true' ? !id(userId) : userId !== '')) return;
    const element = (name) => root.querySelector(`#sodaspaces-${name}`);
    const button = element('button');
    const dialog = element('drawer');
    const close = element('close');
    const status = element('status');
    const loading = element('loading');
    const data = element('data');
    const actor = element('actor');
    const repository = element('repository');
    const environment = element('environment');
    const login = element('login');
    const warning = element('warning');
    const signIn = element('sign-in');
    const signOut = element('sign-out');
    const refresh = element('refresh');
    const reload = element('reload');
    if (![button, dialog, close, status, loading, data, actor, repository, environment,
      login, warning, signIn, signOut, refresh, reload].every(Boolean) ||
      typeof dialog.showModal !== 'function' || typeof dialog.close !== 'function') return;
    root.dataset.mounted = 'true';
    rows[0].append(button);
    root.hidden = false;
    const base = '/-/soda';
    const intent = new URLSearchParams({repository_id: repositoryId});
    if (signed === 'true') intent.set('expected_user_id', userId);
    signIn.href = `${base}/login?${intent}`;
    let generation = 0;
    let controller;
    let session;
    let stale = false;
    let logoutPending = false;

    function clear() {
      session = undefined;
      for (const node of [actor, repository, environment, login, warning]) node.textContent = '';
      signIn.hidden = signOut.hidden = true;
    }
    function busy(value) {
      loading.hidden = !value;
      data.setAttribute('aria-busy', String(value));
      refresh.disabled = value || stale || logoutPending;
      signOut.disabled = value || stale || logoutPending;
    }
    function discard() {
      generation++;
      controller?.abort();
      controller = undefined;
      clear();
      busy(false);
    }
    function invalidate() {
      stale = true;
      discard();
      reload.hidden = false;
      status.textContent = 'Page context may have changed. Reload the repository page before checking Sodaspaces.';
    }
    function requireValue(valid) {
      if (!valid) throw new Error('Invalid Soda response');
    }
    // Errors never carry response bodies, provider text or credentials into the UI/log.
    async function readJSON(path, signal, expected = '') {
      const headers = {Accept: 'application/json'};
      if (expected) headers['X-Soda-Expected-User-ID'] = expected;
      const response = await fetch(base + path, {
        credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, signal,
      });
      requireValue(response.headers.get('content-type')?.split(';')[0].trim() === 'application/json' && response.body);
      const reader = response.body.getReader();
      let size = 0;
      let body = '';
      const decoder = new TextDecoder('utf-8', {fatal: true});
      try {
        for (;;) {
          const {value, done} = await reader.read();
          if (done) break;
          size += value.byteLength;
          requireValue(size <= 65536);
          body += decoder.decode(value, {stream: true});
        }
        body += decoder.decode();
      } finally {
        await reader.cancel().catch(() => {});
        reader.releaseLock();
      }
      const result = JSON.parse(body);
      requireValue(result && typeof result === 'object' && !Array.isArray(result));
      if (!response.ok) {
        const error = new Error('Soda request failed');
        error.status = response.status;
        error.code = typeof result.error?.code === 'string' ? result.error.code : '';
        throw error;
      }
      return result;
    }
    function failure(error) {
      environment.textContent = login.textContent = warning.textContent = '';
      if (error.status === 401 || error.code === 'identity_mismatch' || error.code === 'consent_required') {
        status.textContent = error.code === 'consent_required'
          ? 'Forgejo consent is required. Review the Soda grant in native Applications settings, then sign in again.'
          : 'Sign in to Soda again with the account shown by this repository page.';
        signIn.hidden = false;
        signIn.textContent = 'Sign in to Soda again';
        // Never leave an old logout credential available after a session disagreement.
        if (error.status === 401 || error.code === 'identity_mismatch') {
          session = undefined;
          signOut.hidden = true;
          actor.textContent = '';
        }
      } else if (error.status === 403 || error.status === 404) {
        status.textContent = 'Repository access denied or repository not visible. This does not mean no environment exists.';
      } else if (error.status === 409) {
        status.textContent = 'Request could not complete in the current state. Refresh before trying again.';
      } else {
        status.textContent = 'Sodaspaces is unavailable. Refresh to check again.';
      }
    }
    async function check() {
      if (stale || logoutPending || !dialog.open) return;
      discard();
      const current = generation;
      controller = new AbortController();
      const signal = controller.signal;
      const timeout = setTimeout(() => controller?.signal === signal && controller.abort(), 15000);
      const active = () => current === generation && !stale && dialog.open;
      status.textContent = 'Checking environment…';
      busy(true);
      try {
        const found = await readJSON('/api/session', signal);
        if (!active()) return;
        requireValue(id(found.user?.id) && text(found.user.login) && text(found.csrf_token) &&
          /^[A-Za-z0-9_-]{1,128}$/.test(found.csrf_token) &&
          typeof found.forgejo_url === 'string' && new URL(found.forgejo_url).origin === location.origin);
        session = found;
        actor.textContent = `Soda account: ${found.user.login} (ID ${found.user.id})`;
        signOut.hidden = false;
        if (signed !== 'true' || found.user.id !== userId) {
          status.textContent = 'Native page and Soda identities do not match. Reload if the native account changed, or sign in to Soda explicitly.';
          signIn.hidden = false;
          signIn.textContent = 'Sign in to Soda again';
          return;
        }
        const provider = await readJSON('/api/forgejo/me', signal, userId);
        if (!active()) return;
        requireValue(id(provider.id) && text(provider.login));
        if (provider.id !== userId) {
          failure({status: 401});
          return;
        }
        const collection = await readJSON(`/api/environments?repository_id=${repositoryId}`, signal, userId);
        if (!active()) return;
        requireValue(collection.repository?.id === repositoryId && text(collection.repository.owner) &&
          text(collection.repository.name) && Array.isArray(collection.items) && collection.items.length <= 1);
        repository.textContent = `Repository: ${collection.repository.owner}/${collection.repository.name} (ID ${repositoryId})`;
        if (collection.items.length === 0) {
          status.textContent = 'No shared environment.';
          return;
        }
        const selected = collection.items[0];
        requireValue(selected && /^p[0-9a-f]{24}$/.test(selected.id) && selected.repository_id === repositoryId);
        const detail = await readJSON(`/api/environments/${selected.id}`, signal, userId);
        if (!active()) return;
        requireValue(detail.environment?.id === selected.id && detail.environment.repository_id === repositoryId &&
          typeof detail.environment.provisioned === 'boolean' && text(detail.login) &&
          typeof detail.native_unavailable === 'boolean' && typeof detail.authority_unavailable === 'boolean' &&
          (detail.observed === null || (detail.observed?.id === selected.id && typeof detail.observed.running === 'boolean')));
        environment.textContent = `Environment: ${selected.id}`;
        if (detail.login) login.textContent = `Your project login: ${detail.login}`;
        if (detail.authority_unavailable) warning.textContent = 'Current repository ownership could not be verified; only permitted access is shown.';
        status.textContent = !detail.environment.provisioned ? 'Environment provisioning is incomplete.'
          : detail.native_unavailable || detail.observed === null ? 'Environment reserved; live status unavailable.'
          : detail.observed.running ? 'Environment running. Client reachability is not verified.'
          : 'Environment stopped. Refresh does not start it.';
      } catch (error) {
        if (active()) failure(error);
      } finally {
        clearTimeout(timeout);
        if (active()) busy(false);
      }
    }
    button.addEventListener('click', () => {
      if (dialog.open) return;
      dialog.showModal();
      close.focus();
      if (logoutPending) status.textContent = 'Soda sign-out is still pending.';
      else void check();
    });
    close.addEventListener('click', () => dialog.close());
    dialog.addEventListener('click', (event) => { if (event.target === dialog) dialog.close(); });
    dialog.addEventListener('close', () => {
      if (dialog.open) return; // A queued close from an earlier opening.
      discard();
      button.focus();
    });
    // Clear synchronously on Escape too; the browser emits close asynchronously.
    dialog.addEventListener('cancel', () => discard());
    refresh.addEventListener('click', () => { void check(); });
    reload.addEventListener('click', () => location.reload());
    signIn.addEventListener('click', (event) => { if (stale || logoutPending) event.preventDefault(); });
    signOut.addEventListener('click', async () => {
      if (!session || stale || logoutPending) return;
      const {user, csrf_token: csrf} = session;
      logoutPending = true;
      discard();
      status.textContent = 'Signing out of Soda…';
      let confirmed = false;
      try {
        const response = await fetch(`${base}/api/session/logout`, {
          method: 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
          headers: {'Content-Type': 'application/json', 'X-Soda-Expected-User-ID': user.id, 'X-CSRF-Token': csrf},
          body: '{}', signal: AbortSignal.timeout(15000),
        });
        confirmed = response.status === 204;
        await response.body?.cancel();
      } catch { /* A lost response is not proof that logout did not commit. */ }
      finally {
        logoutPending = false;
        if (!stale && dialog.open) {
          status.textContent = confirmed ? 'Signed out of Soda only. Forgejo and project access are unchanged.'
            : 'Soda sign-out was not confirmed. Refresh before trying again.';
          busy(false);
          signIn.hidden = !confirmed;
        }
      }
    });
    window.addEventListener('pagehide', invalidate);
    window.addEventListener('blur', invalidate);
    document.addEventListener('visibilitychange', () => { if (document.hidden) invalidate(); });
    window.addEventListener('pageshow', (event) => { if (event.persisted) invalidate(); });
    if (document.hidden) invalidate();
    if (location.hash === '#sodaspaces') button.click();
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount, {once: true});
  else mount();
})();
