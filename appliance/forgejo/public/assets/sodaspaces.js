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
    const create = element('create');
    const keys = element('keys');
    const keyList = element('key-list');
    const publicKey = element('public-key');
    const saveKey = element('save-key');
    const join = element('join');
    const connection = element('connection');
    const address = element('address');
    const command = element('command');
    const copy = element('copy');
    const fingerprint = element('fingerprint');
    const result = element('result');
    if (![button, dialog, close, status, loading, data, actor, repository, environment,
      login, warning, signIn, signOut, refresh, reload, create, keys, keyList, publicKey,
      saveKey, join, connection, address, command, copy, fingerprint, result].every(Boolean) ||
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
    let pending = false;
    let selectedID;
    // Memory-only uncertainty, not a recovery journal. Reads never repair native state.
    let uncertainCreate = false;
    let uncertainJoin = false;

    function clear() {
      session = undefined;
      result.textContent = '';
      for (const node of [actor, repository, environment, login, warning]) node.textContent = '';
      signIn.hidden = signOut.hidden = true;
      selectedID = undefined;
      create.hidden = keys.hidden = join.hidden = connection.hidden = true;
      publicKey.value = command.value = '';
      keyList.replaceChildren();
      address.textContent = fingerprint.textContent = '';
      copy.disabled = true;
      copy.removeAttribute('data-clipboard-target');
    }
    function busy(value) {
      loading.hidden = !value;
      data.setAttribute('aria-busy', String(value));
      refresh.disabled = value || stale || logoutPending || pending;
      signOut.disabled = value || stale || logoutPending || pending;
      create.disabled = value || stale || pending || uncertainCreate;
      saveKey.disabled = publicKey.disabled = value || stale || pending;
      join.disabled = value || stale || pending || uncertainJoin;
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
    async function readJSON(path, signal, expected = '', input, csrf) {
      const headers = {Accept: 'application/json'};
      if (expected) headers['X-Soda-Expected-User-ID'] = expected;
      const options = {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, signal};
      if (input !== undefined) {
        headers['Content-Type'] = 'application/json';
        headers['X-CSRF-Token'] = csrf;
        options.method = 'POST';
        options.body = JSON.stringify(input);
      }
      const response = await fetch(base + path, options);
      if (response.headers.get('content-type')?.split(';')[0].trim() !== 'application/json' || !response.body) {
        await response.body?.cancel();
        throw new Error('Invalid Soda response');
      }
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
      if (input !== undefined) requireValue(response.status === (path === '/api/environments' ? 201 : 200));
      return result;
    }
    function failure(error) {
      create.hidden = keys.hidden = join.hidden = connection.hidden = true;
      command.value = '';
      copy.disabled = true;
      copy.removeAttribute('data-clipboard-target');
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
      if (stale || logoutPending || pending || !dialog.open) return;
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
        actor.textContent = `Soda account: ${provider.login} (ID ${provider.id})`;
        const collection = await readJSON(`/api/environments?repository_id=${repositoryId}`, signal, userId);
        if (!active()) return;
        requireValue(collection.repository?.id === repositoryId && text(collection.repository.owner) &&
          text(collection.repository.name) && typeof collection.can_create === 'boolean' &&
          Array.isArray(collection.items) && collection.items.length <= 1);
        repository.textContent = `Repository: ${collection.repository.owner}/${collection.repository.name} (ID ${repositoryId})`;
        if (collection.items.length === 0) {
          status.textContent = 'No shared environment.';
          create.hidden = !collection.can_create;
          if (!collection.can_create) warning.textContent = 'Only the current human repository owner can create an environment; organization-owned creation is not supported.';
          if (uncertainCreate) warning.textContent = 'An earlier creation was not confirmed. Ask the operator to inspect the retained state; do not repeat it.';
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
        selectedID = selected.id;
        environment.textContent = `Environment: ${selected.id}`;
        if (detail.login) login.textContent = `Your project login: ${detail.login}`;
        if (detail.authority_unavailable) warning.textContent = 'Current repository ownership could not be verified; only permitted access is shown.';
        status.textContent = !detail.environment.provisioned ? 'Environment provisioning is incomplete.'
          : detail.native_unavailable || detail.observed === null ? 'Environment reserved; live status unavailable.'
          : detail.observed.running ? 'Environment running. Client reachability is not verified.'
          : 'Environment stopped. Refresh does not start it.';
        if (detail.login) {
          uncertainJoin = false;
          if (detail.environment.provisioned && !detail.native_unavailable && detail.observed?.running) {
            try {
              const own = await readJSON(`/api/environments/${selected.id}/connection`, signal, userId);
              if (!active()) return;
              const c = own.connection;
              requireValue(own.login === detail.login && /^[a-z][a-z0-9_-]{0,30}$/.test(own.login) && own.login !== 'root' &&
                own.routing_verified === false && c?.environment?.id === selected.id &&
                typeof c.environment.running === 'boolean');
              if (c.environment.running) {
                requireValue(validIP(c.environment.ip) && typeof c.host_key === 'string' &&
                  /^ssh-ed25519 [A-Za-z0-9+/]{68}$/.test(c.host_key.trim()) &&
                  /^SHA256:[A-Za-z0-9+/]{43}$/.test(c.fingerprint));
                // Native clipboard delegation resolves this selector document-wide.
                requireValue(document.querySelectorAll('#sodaspaces-command').length === 1);
                address.textContent = `Current project address: ${c.environment.ip}`;
                command.value = `ssh ${own.login}@${c.environment.ip}`;
                fingerprint.textContent = `Ed25519 host-key fingerprint: ${c.fingerprint}`;
                connection.hidden = false;
                copy.disabled = false;
                copy.setAttribute('data-clipboard-target', '#sodaspaces-command');
              } else warning.textContent = 'Environment stopped; no usable connection is advertised.';
            } catch (error) {
              if (active()) {
                if (error.status === 401 || error.code === 'identity_mismatch') failure(error);
                else warning.textContent = 'Current connection details are unavailable. Do not use a cached address.';
              }
            }
          }
        } else if (detail.environment.provisioned && !detail.native_unavailable && detail.observed?.running) {
          const registered = await readJSON('/api/me/development-keys', signal, userId);
          if (!active()) return;
          validateKeys(registered);
          for (const key of registered.items) {
            const item = document.createElement('li');
            item.textContent = key.fingerprint;
            keyList.append(item);
          }
          keys.hidden = false;
          join.hidden = registered.items.length === 0 || registered.items.length > 32;
          if (!registered.items.length) warning.textContent = 'Save a public development-access key, then explicitly add yourself.';
          if (registered.items.length > 32) warning.textContent = 'Native onboarding supports at most 32 development keys; joining is unavailable.';
          if (uncertainJoin) warning.textContent = 'An earlier account operation was not confirmed. Missing membership does not mean no account exists; ask the operator to inspect it.';
        }
      } catch (error) {
        if (active()) failure(error);
      } finally {
        clearTimeout(timeout);
        if (active()) busy(false);
      }
    }
    function validIP(value) {
      if (typeof value !== 'string') return false;
      if (/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(value)) {
        const octets = value.split('.');
        return octets.every(n => String(Number(n)) === n && Number(n) <= 255) &&
          Number(octets[0]) > 0 && Number(octets[0]) < 224 && octets[0] !== '127';
      }
      if (!/^[0-9a-f:]{2,39}$/i.test(value)) return false;
      try {
        const host = new URL(`http://[${value}]/`).hostname;
        return host !== '[::]' && host !== '[::1]' && !host.startsWith('[ff') && !host.startsWith('[::ffff:');
      } catch { return false; }
    }
    function validateKeys(value) {
      requireValue(Array.isArray(value.items) && value.items.every(k => id(k?.id) &&
        text(k.public_key) && /^SHA256:[A-Za-z0-9+/]{43}$/.test(k.fingerprint)));
    }
    async function act(kind) {
      const control = kind === 'create' ? create : kind === 'key' ? saveKey : join;
      if (pending || logoutPending || stale || !dialog.open || !session || control.hidden || control.disabled ||
          (kind === 'key' && keys.hidden)) return;
      const target = selectedID;
      const publicValue = publicKey.value.trim();
      if (kind === 'key' && (!publicValue || publicValue.length > 16384 || /PRIVATE KEY|[\r\n]/.test(publicValue) ||
          !/^(ssh-|ecdsa-|sk-)[A-Za-z0-9@._+-]+ [A-Za-z0-9+/]+={0,2}(?:[ \t].*)?$/.test(publicValue))) {
        result.textContent = 'Provide one public SSH key without options. Never submit a private key.';
        return;
      }
      pending = true;
      const current = generation;
      const active = () => current === generation && !stale && dialog.open;
      const readController = new AbortController();
      controller = readController;
      const timeout = setTimeout(() => readController.abort(), 15000);
      let dispatched = false;
      let message = '';
      result.textContent = 'Checking the acting account before submission…';
      busy(true);
      try {
        const found = await readJSON('/api/session', readController.signal);
        if (!active()) return;
        requireValue(found.user?.id === userId && signed === 'true' &&
          typeof found.csrf_token === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(found.csrf_token));
        const provider = await readJSON('/api/forgejo/me', readController.signal, userId);
        if (!active()) return;
        requireValue(provider.id === userId);
        clearTimeout(timeout);
        // The mutation has its own bounded lifetime: closing only invalidates rendering.
        const path = kind === 'create' ? '/api/environments' : kind === 'key' ? '/api/me/development-keys'
          : `/api/environments/${target}/join`;
        const input = kind === 'create' ? {repository_id: repositoryId} : kind === 'key' ? {public_key: publicValue} : {};
        result.textContent = kind === 'create' ? 'Creating shared environment…' : kind === 'key' ? 'Saving public key…' : 'Adding you to this project…';
        dispatched = true;
        const response = await readJSON(path, AbortSignal.timeout(255000), userId, input, found.csrf_token);
        if (kind === 'create') {
          requireValue(/^p[0-9a-f]{24}$/.test(response.id) && response.repository_id === repositoryId && response.provisioned === true);
          message = 'Environment created. Creation does not join you; save a key and explicitly add yourself if needed.';
        } else if (kind === 'key') {
          validateKeys(response);
          const submitted = publicValue.split(/\s+/).slice(0, 2).join(' ');
          requireValue(response.items.some(k => k.public_key.trim().split(/\s+/).slice(0, 2).join(' ') === submitted));
          message = 'Public key registered. This does not join or update existing project accounts.';
        } else {
          requireValue(typeof response.login === 'string' && /^[a-z][a-z0-9_-]{0,30}$/.test(response.login) && response.login !== 'root');
          message = 'Membership confirmed. Checking your current connection details.';
        }
      } catch (error) {
        const rejected = [400, 401, 403, 404, 409, 413, 415, 422].includes(error.status);
        if (dispatched && !rejected) {
          if (kind === 'create') uncertainCreate = true;
          if (kind === 'join') uncertainJoin = true;
          message = kind === 'key' ? 'Key-save outcome was not confirmed. Refresh registered keys before another explicit save.'
            : 'Native outcome was not confirmed. A reservation or account may exist. Ask the operator to inspect it; do not repeat or repair this action.';
        } else {
          const messages = {
            owner_required: 'Only the current human repository owner can create this environment.',
            reservation_failed: 'An environment may already be reserved. Inspect the refreshed state; do not recreate it.',
            invalid_public_key: 'Provide one public SSH key without options. Never submit a private key.',
            development_key_required: 'Save a development public key before joining.',
            too_many_keys: 'Native onboarding supports at most 32 development keys.',
            unsupported_linux_login: 'Your Forgejo username is not supported as a Linux login. No automatic rename is performed.',
            not_provisioned: 'Provisioning is incomplete. Ask the operator to inspect the retained environment.',
          };
          message = messages[error.code] || 'Action was not authorized or its account check failed. Refresh and review your identity and access before another explicit action.';
        }
      } finally {
        clearTimeout(timeout);
        pending = false;
        if (active()) {
          // This is observation only, never another POST. Identity is checked again.
          const next = generation + 1;
          await check();
          if (generation === next && !stale && dialog.open && session?.user.id === userId && signIn.hidden) result.textContent = message;
        } else if (dialog.open && !stale) {
          status.textContent = 'The earlier action has finished waiting. Refresh to inspect actual state; no action was repeated.';
          busy(false);
        }
      }
    }
    create.addEventListener('click', () => { void act('create'); });
    saveKey.addEventListener('click', () => { void act('key'); });
    join.addEventListener('click', () => { void act('join'); });
    button.addEventListener('click', () => {
      if (dialog.open) return;
      dialog.showModal();
      close.focus();
      if (pending) status.textContent = 'An explicit action is still pending. Refresh after it finishes; do not repeat it.';
      else if (logoutPending) status.textContent = 'Soda sign-out is still pending.';
      else void check();
    });
    close.addEventListener('click', () => { discard(); dialog.close(); });
    dialog.addEventListener('click', (event) => {
      if (event.target === dialog) { discard(); dialog.close(); }
    });
    dialog.addEventListener('close', () => {
      if (dialog.open) return; // A queued close from an earlier opening.
      discard();
      button.focus();
    });
    // Clear synchronously on Escape too; the browser emits close asynchronously.
    dialog.addEventListener('cancel', () => discard());
    refresh.addEventListener('click', () => { void check(); });
    reload.addEventListener('click', () => location.reload());
    signIn.addEventListener('click', (event) => { if (stale || logoutPending || pending) event.preventDefault(); });
    signOut.addEventListener('click', async () => {
      if (!session || stale || logoutPending || pending) return;
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
