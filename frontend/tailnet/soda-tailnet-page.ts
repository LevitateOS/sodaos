import {LitElement, html} from 'lit';
import {connectionSuppressed} from '../spaces/soda-connection.js';
import {id, object, readSodaJSON, sessionResponse} from '../spaces/sodaspaces-api.js';
import {settingsView, hostResult, enrollmentResult} from './soda-tailnet-response.js';
import type {Settings} from './soda-tailnet-response.js';

class TailnetRequestError extends Error {
  constructor(readonly status: number) {super('Tailnet request not confirmed');}
}
type Scope = 'host' | 'enrollment';
type Confirmation = {scope: Scope; body: Record<string, unknown>; label: string; warning: string};
const unknownOutcome = 'Operation unconfirmed. It may have completed. Observe native state before explicitly retrying; nothing was replayed or rolled back.';

class SodaTailnet extends LitElement {
  private actor = '';
  private apiBase = '';
  private lifetime: AbortController | null = null;
  private timer: ReturnType<typeof setInterval> | undefined;
  private settings: Settings | null = null;
  private busy = false;
  private stale = true;
  private blocked = false;
  private sent = false;
  private notice = '';
  private message = 'Checking Tailnet authorization…';
  private authURL = '';
  private pending: Confirmation | null = null;
  private trigger: HTMLButtonElement | null = null;
  private exitDirty = false;
  private advertiseDirty = false;
  private get hostDirty() {return this.exitDirty || this.advertiseDirty;}
  private enrollmentDirty = false;
  private exitRevision = '';
  private advertiseRevision = '';
  private policyRevision = '0';
  private exitNode = '';
  private allowLAN = false;
  private advertise = false;
  private network = '';
  private tags = '';
  private preauthorized = false;
  private mode: 'save' | 'rotate' = 'save';
  private readonly hiddenPage = () => this.retire();
  private readonly shownPage = () => this.resume();
  private readonly visibility = () => {if (!document.hidden && !this.hostDirty && !this.enrollmentDirty) void this.refresh();};
  private readonly departure = (event: BeforeUnloadEvent) => {
    if (this.hostDirty || this.enrollmentDirty || this.busy) {event.preventDefault(); event.returnValue = '';}
  };
  protected createRenderRoot() {return this;}
  connectedCallback() {
    super.connectedCallback();
    if (!this.actor) {
      this.actor = this.dataset.actor || '';
      this.apiBase = (document.getElementById('soda-settings-link')?.dataset.subUrl || '') + '/-/soda';
    }
    window.addEventListener('pagehide', this.hiddenPage);
    window.addEventListener('pageshow', this.shownPage);
    window.addEventListener('soda-session-retired', this.hiddenPage);
    window.addEventListener('beforeunload', this.departure);
    document.addEventListener('visibilitychange', this.visibility);
    this.resume();
  }
  disconnectedCallback() {
    this.retire();
    window.removeEventListener('pagehide', this.hiddenPage);
    window.removeEventListener('pageshow', this.shownPage);
    window.removeEventListener('soda-session-retired', this.hiddenPage);
    window.removeEventListener('beforeunload', this.departure);
    document.removeEventListener('visibilitychange', this.visibility);
    super.disconnectedCallback();
  }
  private clearSecrets() {
    this.querySelectorAll<HTMLInputElement>('input[type=password]').forEach(input => {input.value = '';});
    this.authURL = '';
    // Synchronous retirement: a queued Lit render must not retain a usable secret link.
    this.querySelectorAll<HTMLAnchorElement>('a[data-authentication]').forEach(link => {link.removeAttribute('href'); link.textContent = 'Authentication link retired';});
  }
  private retire() {
    this.clearSecrets();
    this.lifetime?.abort(); this.lifetime = null;
    clearInterval(this.timer); this.timer = undefined;
    if (this.sent) this.notice = unknownOutcome;
    this.sent = false; this.busy = false; this.stale = true;
    this.settings = null; this.pending = null; this.trigger = null;
    this.message = 'Tailnet controls retired. Revalidate the original operator to continue.';
    this.requestUpdate();
  }
  private resume() {
    if (this.lifetime || !this.isConnected || connectionSuppressed()) return;
    this.clearSecrets();
    this.lifetime = new AbortController();
    this.timer = setInterval(() => {
      if (!document.hidden && !this.hostDirty && !this.enrollmentDirty && !this.blocked) void this.refresh();
    }, 10000);
    void this.refresh();
  }
  private current(lifetime: AbortController) {
    return this.isConnected && this.lifetime === lifetime && !lifetime.signal.aborted && !connectionSuppressed();
  }
  private requireCurrent(lifetime: AbortController) {if (!this.current(lifetime)) throw Error('Retired Tailnet owner');}
  private loseAuthorization() {
    this.clearSecrets(); this.settings = null; this.pending = null; this.trigger = null; this.blocked = true; this.stale = true;
    this.requestUpdate();
  }
  private async session(lifetime: AbortController) {
    this.requireCurrent(lifetime);
    try {
      if (!id(this.actor)) throw Error('Invalid original actor');
      const response = await fetch(this.apiBase + '/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error',
        headers: {'X-Soda-Expected-User-ID': this.actor}, signal: AbortSignal.any([lifetime.signal, AbortSignal.timeout(15000)])});
      this.requireCurrent(lifetime);
      if (!response.ok) {await response.body?.cancel(); throw Error('Operator unavailable');}
      const raw = await readSodaJSON(response), session = sessionResponse(raw, location.origin);
      this.requireCurrent(lifetime);
      if (session.user.id !== this.actor || object(raw).soda_operator !== true) throw Error('Operator changed');
      return session;
    } catch (error) {if (this.current(lifetime)) this.loseAuthorization(); throw error;}
  }
  private async request(lifetime: AbortController, scope?: Scope, body?: string): Promise<unknown> {
    try {
      const session = await this.session(lifetime);
      this.requireCurrent(lifetime);
      if (body !== undefined) this.sent = true;
      const pending = fetch(this.apiBase + '/api/settings/tailnet' + (scope ? '/' + scope : ''), {
        method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer',
        headers: {'X-Soda-Expected-User-ID': this.actor, ...(body === undefined ? {} : {'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token})},
        ...(body === undefined ? {} : {body}), signal: AbortSignal.any([lifetime.signal, AbortSignal.timeout(30000)]),
      });
      body = undefined;
      const response = await pending;
      this.requireCurrent(lifetime);
      if (!response.ok) {
        if ([401, 403].includes(response.status)) this.loseAuthorization();
        await response.body?.cancel(); throw new TailnetRequestError(response.status);
      }
      const result = await readSodaJSON(response);
      this.requireCurrent(lifetime);
      await this.session(lifetime); // A late response must not publish for a retired operator.
      this.requireCurrent(lifetime);
      return result;
    } finally {body = undefined;}
  }
  private resetHost(discard = false) {
    const host = this.settings?.host;
    if (!host) return;
    if (discard) {this.exitDirty = false; this.advertiseDirty = false;}
    if (!this.exitDirty) {
      this.exitRevision = host.revision;
      this.exitNode = host.preferences.exit_node_ip || host.peers.find(p => p.id === host.preferences.exit_node_id)?.addresses[0] || (host.preferences.exit_node_id ? 'missing:' + host.preferences.exit_node_id : '');
      this.allowLAN = host.preferences.allow_lan;
    }
    if (!this.advertiseDirty) {this.advertiseRevision = host.revision; this.advertise = host.preferences.advertise_exit_node;}
    this.requestUpdate();
  }
  private resetEnrollment() {
    const policy = this.settings?.enrollment;
    if (!policy) return;
    this.clearSecrets(); this.policyRevision = policy.revision;
    this.network = policy.tailnet; this.tags = policy.tags.join(', '); this.preauthorized = policy.preauthorized;
    this.enrollmentDirty = false; this.requestUpdate();
  }
  private async refresh() {
    const lifetime = this.lifetime;
    if (!lifetime || !this.current(lifetime) || this.busy || this.pending) return;
    this.busy = true; this.message = 'Checking authorization and Tailnet observations…'; this.requestUpdate();
    try {
      const next = settingsView(await this.request(lifetime));
      this.requireCurrent(lifetime);
      this.settings = next; this.stale = false; this.blocked = false;
      this.resetHost();
      if (!this.enrollmentDirty) {
        // Passive observations never retain/recover an authentication URL.
        this.policyRevision = next.enrollment.revision; this.network = next.enrollment.tailnet;
        this.tags = next.enrollment.tags.join(', '); this.preauthorized = next.enrollment.preauthorized;
      }
      if (next.host?.state === 'Running' && !next.host.expired) this.clearAuthURL();
      this.message = next.host_unavailable ? 'Appliance observation unavailable. This is not a disconnected state; enrollment policy is shown separately.' : 'Observations refreshed. Addresses and online status do not prove client reachability.';
    } catch {
      if (this.current(lifetime)) {this.clearSecrets(); this.stale = true; this.message = this.blocked ? 'The original Soda operator is unavailable. No private controls are shown.' : 'Tailnet observation unavailable. Previous data and drafts are stale, not an empty network. The native helper may be disabled.';}
    } finally {if (this.current(lifetime)) {this.busy = false; this.requestUpdate();}}
  }
  private clearAuthURL() {
    this.authURL = '';
    this.querySelectorAll<HTMLAnchorElement>('a[data-authentication]').forEach(link => link.removeAttribute('href'));
  }
  private async mutate(scope: Scope, action: string, body: string, revision: string) {
    const lifetime = this.lifetime;
    if (!lifetime || !this.current(lifetime) || this.busy || this.stale || this.blocked) return;
    this.busy = true; this.sent = false; this.pending = null; this.trigger = null;
    this.clearSecrets(); this.notice = 'Checking authorization before dispatch…'; this.requestUpdate();
    try {
      const pending = this.request(lifetime, scope, body); body = '';
      const raw = await pending;
      this.requireCurrent(lifetime);
      if (!this.settings) throw Error('Settings retired');
      if (scope === 'host') {
        const result = hostResult(raw, action);
        this.settings = {...this.settings, host: result.host, host_unavailable: result.readback_unavailable};
        this.authURL = result.authURL;
        this.notice = result.outcome === 'unconfirmed' ? unknownOutcome : result.outcome === 'pending' ? 'Authentication pending. Complete the provider step; closing this page does not cancel native authentication.' : result.outcome === 'observed' ? 'Authentication observed; no login or connection change was requested.' : 'Native operation acknowledged. This is not approval or routed-traffic proof.';
        if (result.readback_unavailable) this.notice += ' Host readback failed independently. Refresh observations; do not replay the operation to repair this observer.';
        if (result.outcome === 'confirmed' && !result.readback_unavailable) {
          if (action === 'exit-node') this.exitDirty = false;
          if (action === 'advertise-exit-node') this.advertiseDirty = false;
        }
        this.resetHost();
      } else {
        const result = enrollmentResult(raw, action, revision);
        this.settings = {...this.settings, enrollment: result.enrollment};
        this.notice = result.saved ? 'Policy saved. Existing devices were not disconnected, revoked or retargeted. Project enrollment is not verified.' : 'Credential check passed; nothing was saved and no auth key or device was created. Network, scope and enrollment remain unverified.';
        if (result.saved) this.resetEnrollment();
      }
    } catch (error) {
      if (this.current(lifetime)) {
        this.clearSecrets(); this.stale = true;
        this.notice = error instanceof TailnetRequestError && [400, 409, 422].includes(error.status)
          ? 'Request rejected before the requested management effect. Review inputs, revision and runtime support, then refresh. No automatic retry occurred.'
          : this.sent ? unknownOutcome : 'Operation was not sent. Check the original operator and refresh before retrying.';
      }
    } finally {
      body = '';
      if (this.current(lifetime)) {this.sent = false; this.busy = false; this.requestUpdate();}
    }
  }
  private async choose(event: Event, confirmation: Confirmation) {
    if (this.busy || this.stale || this.blocked || this.pending) return;
    this.clearSecrets(); this.pending = confirmation;
    this.trigger = event.currentTarget instanceof HTMLButtonElement ? event.currentTarget : null;
    this.requestUpdate(); await this.updateComplete;
    if (this.pending === confirmation) this.querySelector<HTMLButtonElement>('[data-confirm]')?.focus();
  }
  private async cancel() {
    const trigger = this.trigger; this.pending = null; this.trigger = null; this.requestUpdate(); await this.updateComplete;
    if (trigger?.isConnected) trigger.focus();
  }
  private confirm() {
    const pending = this.pending;
    if (pending) void this.mutate(pending.scope, String(pending.body.action), JSON.stringify(pending.body), String(pending.body.revision));
  }
  private hostAction(event: Event, action: string) {
    const host = this.settings?.host; if (!host) return;
    if (action === 'exit-node' && this.exitNode.startsWith('missing:')) {this.notice = 'Select an available exit node or explicitly choose None before applying.'; this.requestUpdate(); return;}
    const body: Record<string, unknown> = {action, revision: host.revision};
    if (action === 'signin' || action === 'authentication') {void this.mutate('host', action, JSON.stringify(body), host.revision); return;}
    body.confirm = action;
    let warning = 'No alternative management path has been verified here. This may interrupt your current management connection and SSH sessions. Use your existing approved private route or console for recovery.';
    if (action === 'exit-node') {Object.assign(body, {revision: this.exitRevision, exit_node: this.exitNode, allow_lan: !!this.exitNode && this.allowLAN});}
    if (action === 'advertise-exit-node') {Object.assign(body, {revision: this.advertiseRevision, advertise: this.advertise}); warning = 'Change only exit-node advertisement, preserving unrelated routes. Tailscale owns approval and routing policy; advertisement alone does not prove usable routed traffic.';}
    if (action === 'refresh-forgejo') warning = 'Refresh the appliance Git SSH advertisement through the native helper. This may restart Forgejo and interrupt its requests. It does not change browser/OAuth origins or confirm Tailnet reachability.';
    void this.choose(event, {scope: 'host', body, label: action, warning});
  }
  private submitEnrollment(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || !(event.submitter instanceof HTMLButtonElement)) return;
    const form = event.currentTarget, action = event.submitter.value;
    if (this.busy || this.stale || this.blocked || !['check', 'save', 'rotate'].includes(action)) {this.clearSecrets(); return;}
    if (!form.reportValidity()) {this.clearSecrets(); return;}
    const data = new FormData(form);
    const field = (key: string) => {const value = data.get(key); return typeof value === 'string' ? value : '';};
    if (action !== 'check' && data.get('reviewed') !== 'on') {
      this.clearSecrets(); data.delete('client_secret'); this.notice = 'Review and confirm the exposure/binding change before saving.'; this.requestUpdate(); return;
    }
    const tags = this.tags.split(',').map(tag => tag.trim()).sort();
    let body = JSON.stringify({action, revision: this.policyRevision, tailnet: this.network, tags, preauthorized: this.preauthorized, client_id: field('client_id'), client_secret: field('client_secret')});
    data.delete('client_secret'); this.clearSecrets();
    void this.mutate('enrollment', action, body, this.policyRevision); body = '';
  }
  protected render() {
    const settings = this.settings, host = settings?.host, policy = settings?.enrollment;
    const disabled = this.busy || this.stale || this.blocked || !!this.pending;
    const options = host?.peers.filter(p => p.exit_node && p.addresses.length > 0).map(peer => ({peer, address: peer.addresses.includes(this.exitNode) ? this.exitNode : peer.addresses[0] || ''})) || [];
    const selectedMissing = host && host.preferences.exit_node_id !== '' && !host.peers.some(p => p.id === host.preferences.exit_node_id);
    return html`
      <div class="settings-actions"><button type="button" ?disabled=${this.busy || !!this.pending} @click=${() => {this.resume(); void this.refresh();}}>Refresh observations</button></div>
      <p role="status" aria-live="polite">${this.message}</p>
      ${this.notice ? html`<p class="tailnet-notice" role="status">${this.notice}</p>` : ''}
      ${this.blocked || !this.lifetime ? html`<p><a href=${this.apiBase + '/login?destination=tailnet&expected_user_id=' + this.actor}>Reconnect Soda explicitly</a>. Native CLI/console recovery remains available.</p>` : ''}
      ${this.pending ? html`<section class="tailnet-notice" role="region" aria-labelledby="tailnet-confirm-title" @keydown=${(event: KeyboardEvent) => {if (event.key === 'Escape') {event.preventDefault(); void this.cancel();}}}>
        <h2 id="tailnet-confirm-title">Confirm ${this.pending.label}</h2><p>${this.pending.warning}</p>
        <div class="settings-actions"><button type="button" data-confirm @click=${() => this.confirm()}>Confirm ${this.pending.label}</button><button type="button" @click=${() => void this.cancel()}>Cancel</button></div>
      </section>` : ''}
      ${settings && !this.blocked ? html`
      <section aria-labelledby="tailnet-appliance"><h2 id="tailnet-appliance">${this.dataset.applianceLabel || 'Appliance'}</h2>
        <p>Persistent native connection. Host identity and ordinary SSH authentication are not shared with projects.</p>
        ${this.stale ? html`<p>Appliance observations are stale. No usable endpoint is asserted.</p>` : host ? html`
          <dl><dt>Native state</dt><dd>${host.state}${host.expired ? ' · expired' : ''}</dd><dt>Network</dt><dd>${host.tailnet || 'Not observed'}</dd><dt>Device</dt><dd>${host.dns_name || 'Not observed'}</dd>
          <dt>Observed addresses (not verified reachable)</dt><dd>${host.addresses.join(', ') || 'None observed'}</dd><dt>MagicDNS</dt><dd>${host.magic_dns_enabled ? 'Enabled' : 'Disabled / not observed'}</dd>
          <dt>Native health issues</dt><dd>${host.health_issues} — raw diagnostics are intentionally hidden.</dd></dl>
          ${host.state === 'NeedsMachineAuth' ? html`<p>Device approval is required in Tailscale. Tailnet Lock signing is not automated; do not weaken Lock or copy signing keys.</p>` : ''}
        ` : html`<p>Host observation unavailable, not disconnected. Verify the native daemon, reviewed version and helper configuration through an approved private path.</p>`}
        ${host ? html`
          <fieldset ?disabled=${disabled}><legend>Appliance connection</legend><div class="settings-actions">
            <button type="button" @click=${(e: Event) => this.hostAction(e, 'signin')}>${host.have_node_key ? 'Resume / reauthenticate' : 'Sign in appliance'}</button>
            <button type="button" @click=${(e: Event) => this.hostAction(e, 'authentication')}>Recover authentication link</button>
            <button type="button" @click=${(e: Event) => this.hostAction(e, 'logout')}>Disconnect appliance</button>
            <button type="button" @click=${(e: Event) => this.hostAction(e, 'refresh-forgejo')}>Refresh Forgejo advertisement</button>
          </div></fieldset>
          ${this.authURL ? html`<p><a data-authentication href=${this.authURL} target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer">Continue appliance sign-in at Tailscale</a>. This is provider authentication, not Soda sign-in. Do not copy this link into logs or evidence.</p>` : ''}
          <fieldset ?disabled=${disabled}><legend>Exit-node preferences</legend>
            ${selectedMissing ? html`<p role="alert">The selected native exit-node ID is missing from the peer observation. It has not been cleared. Select an available replacement or explicitly choose None.</p>` : ''}
            <label>Exit node<select aria-label="Exit node" .value=${this.exitNode} @change=${(e: Event) => {if (e.target instanceof HTMLSelectElement) {this.exitNode = e.target.value; if (!this.exitNode) this.allowLAN = false; this.exitDirty = true; this.requestUpdate();}}}>
              <option value="" ?selected=${this.exitNode === ''}>None (clear only on Apply)</option>
              ${this.exitNode && !options.some(p => p.address === this.exitNode) ? html`<option value=${this.exitNode} selected>Previous selection (unavailable)</option>` : ''}
              ${options.map(({peer, address}) => html`<option value=${address} ?selected=${address === this.exitNode} ?disabled=${!peer.online || peer.expired}>${peer.dns_name || peer.id}${!peer.online || peer.expired ? ' (unavailable)' : ''}</option>`)}
            </select></label>
            <label><input type="checkbox" .checked=${this.allowLAN} ?disabled=${!this.exitNode} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) {this.allowLAN = e.target.checked; this.exitDirty = true; this.requestUpdate();}}}>Allow local LAN while using exit node</label>
            <button type="button" @click=${(e: Event) => this.hostAction(e, 'exit-node')}>Apply exit node</button>
            <label><input type="checkbox" .checked=${this.advertise} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) {this.advertise = e.target.checked; this.advertiseDirty = true; this.requestUpdate();}}}>Advertise appliance as an exit node</label>
            <button type="button" @click=${(e: Event) => this.hostAction(e, 'advertise-exit-node')}>Apply advertisement</button>
            <p>Provider approval is separate from advertisement and online status. Existing subnet routes are not edited.</p>
            <button type="button" @click=${() => this.resetHost(true)}>Discard host draft / use latest observation</button>
            ${this.hostDirty ? html`<p>Unsaved host draft; its original revision is retained across refresh.</p>` : ''}
          </fieldset>
          ${!this.stale ? html`<details><summary>Observed peers (${host.peers.length})</summary><ul>${host.peers.map(p => html`<li>${p.dns_name || p.id}: ${p.online ? 'online' : 'offline'}${p.expired ? ', expired' : ''}${p.exit_node ? ', exit node' : ''} · ${p.addresses.join(', ')}</li>`)}</ul></details>` : ''}
        ` : ''}
      </section>
      ${policy ? html`<section aria-labelledby="tailnet-enrollment"><h2 id="tailnet-enrollment">${this.dataset.enrollmentLabel || 'Automatic project access'}</h2>
        <p>${policy.configured ? 'Configured' : 'Not configured'} · Credential check ${policy.credential_checked ? 'previously passed' : 'not recorded'} · Enrollment not verified.</p>
        <p>Managed network: ${policy.tailnet || 'None'}. Future admission: ${policy.admission ? 'configured open' : 'closed'}. New projects: Off.</p>
        <p>Project runtime unsupported in this source slice. No project is enrolled by this page, and automatic defaults remain unavailable. Devices are intended to be ephemeral while projects run; existing projects remain Off until explicitly selected.</p>
        <p>Tailscale owns network access policy. A project tag alone does not isolate host/peers. Approval-required networks need explicitly permitted preauthorization; automatic Tailnet Lock signing is unsupported.</p>
        <p><a href="https://tailscale.com/kb/1215/oauth-clients" target="_blank" rel="noopener noreferrer">Create a restricted Tailscale OAuth client</a> with auth_keys and only the selected tags. Token acceptance does not prove target, scope, expiry or enrollment; no hidden test device is created.</p>
        <fieldset ?disabled=${disabled}><legend>Enrollment configuration</legend>
          <label>Configuration action<select aria-label="Configuration action" .value=${this.mode} @change=${(e: Event) => {if (e.target instanceof HTMLSelectElement) {this.mode = e.target.value === 'rotate' ? 'rotate' : 'save'; this.resetEnrollment();}}}>
            <option value="save">Save a new / replacement binding</option><option value="rotate" ?disabled=${!policy.configured}>Rotate credential in this binding</option>
          </select></label>
          <form @submit=${(e: SubmitEvent) => this.submitEnrollment(e)} @input=${() => {this.enrollmentDirty = true;}}>
            <label>Managed Tailnet<input required maxlength="253" .value=${this.network} ?readonly=${this.mode === 'rotate'} @input=${(e: Event) => {if (e.target instanceof HTMLInputElement) this.network = e.target.value;}}></label>
            <label>Project tags (comma separated)<input required .value=${this.tags} ?readonly=${this.mode === 'rotate'} @input=${(e: Event) => {if (e.target instanceof HTMLInputElement) this.tags = e.target.value;}}></label>
            <label><input type="checkbox" .checked=${this.preauthorized} ?disabled=${this.mode === 'rotate'} @change=${(e: Event) => {if (e.target instanceof HTMLInputElement) this.preauthorized = e.target.checked;}}>Explicitly preauthorize devices if provider policy permits</label>
            <label>OAuth client ID<input name="client_id" required maxlength="128" autocomplete="off"></label>
            <label>OAuth client secret<input name="client_secret" type="password" required maxlength="525" autocomplete="new-password" spellcheck="false"></label>
            <p>Secret is cleared when submitted, on failure and on departure. It is never displayed or stored in browser storage. Re-enter it for a subsequent Save after Check.</p>
            <label><input type="checkbox" name="reviewed">I reviewed private-network exposure and this binding change. Replacement requires explicit project reselection; rotation preserves tags and existing device identities.</label>
            <div class="settings-actions"><button type="submit" value="check">Check credential only</button><button type="submit" value=${this.mode}>${this.mode === 'rotate' ? 'Rotate credential' : 'Save binding'}</button>
            <button type="button" @click=${() => this.resetEnrollment()}>Discard enrollment draft</button></div>
          </form>
          ${policy.configured ? html`<div class="settings-actions"><button type="button" @click=${(e: Event) => void this.choose(e, {scope: 'enrollment', body: {action: 'disable', revision: policy.revision}, label: 'close admission', warning: 'Close future enrollment and keep the new-project default Off. Existing device connections and project policies are not disconnected, revoked or deleted.'})}>Close future admission</button>
          <button type="button" @click=${(e: Event) => void this.choose(e, {scope: 'enrollment', body: {action: 'default', revision: policy.revision, default: false}, label: 'default Off', warning: 'Save Off for future creation defaults. Existing project settings and connections do not change.'})}>Keep new-project default Off</button></div>` : ''}
        </fieldset>
      </section>` : ''}
      ` : ''}`;
  }
}
customElements.define('soda-tailnet', SodaTailnet);
export function mountTailnetPage(root: HTMLElement, actor: string) {
  const page = document.createElement('soda-tailnet'); page.dataset.actor = actor;
  page.dataset.applianceLabel = root.dataset.applianceLabel || 'Appliance';
  page.dataset.enrollmentLabel = root.dataset.enrollmentLabel || 'Automatic project access';
  root.append(page);
}
