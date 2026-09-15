import {LitElement, html} from 'lit';
import {connectionSuppressed} from '../spaces/soda-connection.js';
import {id, readSodaJSON} from '../spaces/sodaspaces-api.js';
import type {Session} from '../spaces/sodaspaces-api.js';
import {settingsView, hostResult, enrollmentResult} from './soda-tailnet-response.js';
import type {Enrollment, Host, Settings} from './soda-tailnet-response.js';

class TailnetRequestError extends Error {
  constructor(readonly status: number) {
    super('Tailnet request not confirmed');
  }
}
type Scope = 'host' | 'enrollment';
type Confirmation = {scope: Scope; body: Record<string, unknown>; label: string; warning: string};
type HostResult = ReturnType<typeof hostResult>;
type ExitChoice = {peer: Host['peers'][number]; address: string};
const unknownOutcome =
  'Operation unconfirmed. It may have completed. Observe native state before explicitly retrying; nothing was replayed or rolled back.';

function observedExitNode(host: Host): string {
  const prefs = host.preferences;
  if (prefs.exit_node_ip) return prefs.exit_node_ip;
  const match = host.peers.find((peer) => peer.id === prefs.exit_node_id);
  if (match?.addresses[0]) return match.addresses[0];
  return prefs.exit_node_id ? 'missing:' + prefs.exit_node_id : '';
}
function advertisedExitPeer(peer: Host['peers'][number]) {
  return peer.exit_node && peer.addresses.length > 0;
}
function exitPeerChoice(exitNode: string, peer: Host['peers'][number]): ExitChoice {
  return {peer, address: peer.addresses.includes(exitNode) ? exitNode : peer.addresses[0] || ''};
}
function exitPeerOptions(host: Host, exitNode: string) {
  const options: ExitChoice[] = [];
  for (const peer of host.peers) {
    if (advertisedExitPeer(peer)) options.push(exitPeerChoice(exitNode, peer));
  }
  return options;
}
function exitNodeMissing(host: Host) {
  const selected = host.preferences.exit_node_id;
  if (selected === '') return false;
  return !host.peers.some((peer) => peer.id === selected);
}
function hostOutcomeNotice(outcome: string) {
  if (outcome === 'unconfirmed') return unknownOutcome;
  if (outcome === 'pending')
    return 'Authentication pending. Complete the provider step; closing this page does not cancel native authentication.';
  if (outcome === 'observed') return 'Authentication observed; no login or connection change was requested.';
  return 'Native operation acknowledged. This is not approval or routed-traffic proof.';
}
function mutationFailureNotice(error: unknown, sent: boolean) {
  if (error instanceof TailnetRequestError && [400, 409, 422].includes(error.status)) {
    return 'Request rejected before the requested management effect. Review inputs, revision and runtime support, then refresh. No automatic retry occurred.';
  }
  return sent ? unknownOutcome : 'Operation was not sent. Check the original operator and refresh before retrying.';
}
function hostWarning(action: string) {
  if (action === 'advertise-exit-node')
    return 'Change only exit-node advertisement, preserving unrelated routes. Tailscale owns approval and routing policy; advertisement alone does not prove usable routed traffic.';
  if (action === 'refresh-forgejo')
    return 'Refresh the appliance Git SSH advertisement through the native helper. This may restart Forgejo and interrupt its requests. It does not change browser/OAuth origins or confirm Tailnet reachability.';
  return 'No alternative management path has been verified here. This may interrupt your current management connection and SSH sessions. Use your existing approved private route or console for recovery.';
}
function formText(data: FormData, key: string) {
  const value = data.get(key);
  return typeof value === 'string' ? value : '';
}
function renderPeerItem(peer: Host['peers'][number]) {
  return html`<li>
    ${peer.dns_name || peer.id}:
    ${peer.online ? 'online' : 'offline'}${peer.expired ? ', expired' : ''}${peer.exit_node ? ', exit node' : ''} ·
    ${peer.addresses.join(', ')}
  </li>`;
}

class SodaTailnet extends LitElement {
  private actor = '';
  private csrf = '';
  configure(actor: string, session: Session) {
    if (this.actor || !id(actor) || session.user.id !== actor) throw Error('Invalid original Tailnet actor');
    this.actor = actor;
    this.csrf = session.csrf_token;
  }
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
  private get hostDirty() {
    return this.exitDirty || this.advertiseDirty;
  }
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
  private readonly visibility = () => {
    if (!document.hidden && !this.hostDirty && !this.enrollmentDirty) void this.refresh();
  };
  private readonly departure = (event: BeforeUnloadEvent) => {
    if (this.hostDirty || this.enrollmentDirty || this.busy) {
      event.preventDefault();
      event.returnValue = '';
    }
  };
  protected createRenderRoot() {
    return this;
  }
  connectedCallback() {
    super.connectedCallback();
    this.apiBase = (document.getElementById('soda-settings-link')?.dataset.subUrl || '') + '/-/soda';
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
    this.querySelectorAll<HTMLInputElement>('input[type=password]').forEach((input) => {
      input.value = '';
    });
    this.authURL = '';
    // Synchronous retirement: a queued Lit render must not retain a usable secret link.
    this.querySelectorAll<HTMLAnchorElement>('a[data-authentication]').forEach((link) => {
      link.removeAttribute('href');
      link.textContent = 'Authentication link retired';
    });
  }
  private retire() {
    this.clearSecrets();
    this.lifetime?.abort();
    this.lifetime = null;
    clearInterval(this.timer);
    this.timer = undefined;
    if (this.sent) this.notice = unknownOutcome;
    this.sent = false;
    this.busy = false;
    this.stale = true;
    this.settings = null;
    this.pending = null;
    this.trigger = null;
    this.message = 'Tailnet controls retired. Revalidate the original operator to continue.';
    this.requestUpdate();
  }
  private resume() {
    if (!this.actor || this.lifetime || !this.isConnected || connectionSuppressed()) return;
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
  private requireCurrent(lifetime: AbortController) {
    if (!this.current(lifetime)) throw Error('Retired Tailnet owner');
  }
  private loseAuthorization() {
    this.clearSecrets();
    this.settings = null;
    this.pending = null;
    this.trigger = null;
    this.blocked = true;
    this.stale = true;
    this.requestUpdate();
  }
  private async request(lifetime: AbortController, scope?: Scope, body?: string): Promise<unknown> {
    try {
      this.requireCurrent(lifetime);
      if (body !== undefined) this.sent = true;
      const pending = fetch(this.apiBase + '/api/settings/tailnet' + (scope ? '/' + scope : ''), {
        method: body === undefined ? 'GET' : 'POST',
        credentials: 'same-origin',
        cache: 'no-store',
        redirect: 'error',
        referrerPolicy: 'no-referrer',
        headers: {
          'X-Soda-Expected-User-ID': this.actor,
          ...(body === undefined ? {} : {'Content-Type': 'application/json', 'X-CSRF-Token': this.csrf}),
        },
        ...(body === undefined ? {} : {body}),
        signal: AbortSignal.any([lifetime.signal, AbortSignal.timeout(30000)]),
      });
      body = undefined;
      const response = await pending;
      this.requireCurrent(lifetime);
      if (!response.ok) {
        if ([401, 403].includes(response.status)) this.loseAuthorization();
        await response.body?.cancel();
        throw new TailnetRequestError(response.status);
      }
      const result = await readSodaJSON(response);
      this.requireCurrent(lifetime);
      return result;
    } finally {
      body = undefined;
    }
  }
  private resetHost(discard = false) {
    const host = this.settings?.host;
    if (!host) return;
    if (discard) {
      this.exitDirty = false;
      this.advertiseDirty = false;
    }
    if (!this.exitDirty) {
      this.exitRevision = host.revision;
      this.exitNode = observedExitNode(host);
      this.allowLAN = host.preferences.allow_lan;
    }
    if (!this.advertiseDirty) {
      this.advertiseRevision = host.revision;
      this.advertise = host.preferences.advertise_exit_node;
    }
    this.requestUpdate();
  }
  private copyEnrollmentDraft(policy: Enrollment) {
    this.policyRevision = policy.revision;
    this.network = policy.tailnet;
    this.tags = policy.tags.join(', ');
    this.preauthorized = policy.preauthorized;
  }
  private resetEnrollment() {
    const policy = this.settings?.enrollment;
    if (!policy) return;
    this.clearSecrets();
    this.copyEnrollmentDraft(policy);
    this.enrollmentDirty = false;
    this.requestUpdate();
  }
  private applyObservedSettings(next: Settings) {
    this.settings = next;
    this.stale = false;
    this.blocked = false;
    this.resetHost();
    if (!this.enrollmentDirty) this.copyEnrollmentDraft(next.enrollment);
    if (next.host?.state === 'Running' && !next.host.expired) this.clearAuthURL();
    this.message = next.host_unavailable
      ? 'Appliance observation unavailable. This is not a disconnected state; enrollment policy is shown separately.'
      : 'Observations refreshed. Addresses and online status do not prove client reachability.';
  }
  private refreshFailed(lifetime: AbortController) {
    if (!this.current(lifetime)) return;
    this.clearSecrets();
    this.stale = true;
    this.message = this.blocked
      ? 'The original Soda operator is unavailable. No private controls are shown.'
      : 'Tailnet observation unavailable. Previous data and drafts are stale, not an empty network. The native helper may be disabled.';
  }
  private async refresh() {
    const lifetime = this.lifetime;
    if (!lifetime || !this.current(lifetime) || this.busy || this.pending) return;
    this.busy = true;
    this.message = 'Checking authorization and Tailnet observations…';
    this.requestUpdate();
    try {
      const next = settingsView(await this.request(lifetime));
      this.requireCurrent(lifetime);
      this.applyObservedSettings(next);
    } catch {
      this.refreshFailed(lifetime);
    } finally {
      if (this.current(lifetime)) {
        this.busy = false;
        this.requestUpdate();
      }
    }
  }
  private clearAuthURL() {
    this.authURL = '';
    this.querySelectorAll<HTMLAnchorElement>('a[data-authentication]').forEach((link) => link.removeAttribute('href'));
  }
  private clearHostDraft(action: string) {
    if (action === 'exit-node') this.exitDirty = false;
    if (action === 'advertise-exit-node') this.advertiseDirty = false;
  }
  private applyHostMutation(action: string, result: HostResult) {
    if (!this.settings) throw Error('Settings retired');
    this.settings = {...this.settings, host: result.host, host_unavailable: result.readback_unavailable};
    this.authURL = result.authURL;
    this.notice = hostOutcomeNotice(result.outcome);
    if (result.readback_unavailable)
      this.notice +=
        ' Host readback failed independently. Refresh observations; do not replay the operation to repair this observer.';
    if (result.outcome === 'confirmed' && !result.readback_unavailable) this.clearHostDraft(action);
    this.resetHost();
  }
  private applyEnrollmentMutation(action: string, revision: string, raw: unknown) {
    if (!this.settings) throw Error('Settings retired');
    const result = enrollmentResult(raw, action, revision);
    this.settings = {...this.settings, enrollment: result.enrollment};
    this.notice = result.saved
      ? 'Policy saved. Existing devices were not disconnected, revoked or retargeted. Project enrollment is not verified.'
      : 'Credential check passed; nothing was saved and no auth key or device was created. Network, scope and enrollment remain unverified.';
    // Admission/default writes do not submit the credential-binding draft.
    // Keep its original CAS revision until explicit discard or save/rotation.
    if (result.saved && (action === 'save' || action === 'rotate' || !this.enrollmentDirty)) this.resetEnrollment();
  }
  private applyMutation(scope: Scope, action: string, revision: string, raw: unknown) {
    if (!this.settings) throw Error('Settings retired');
    if (scope === 'host') this.applyHostMutation(action, hostResult(raw, action));
    else this.applyEnrollmentMutation(action, revision, raw);
  }
  private mutationFailed(lifetime: AbortController, error: unknown) {
    if (!this.current(lifetime)) return;
    this.clearSecrets();
    this.stale = true;
    this.notice = mutationFailureNotice(error, this.sent);
  }
  private async mutate(scope: Scope, action: string, body: string, revision: string) {
    const lifetime = this.lifetime;
    if (!lifetime || !this.current(lifetime) || this.busy || this.stale || this.blocked) return;
    this.busy = true;
    this.sent = false;
    this.pending = null;
    this.trigger = null;
    this.clearSecrets();
    this.notice = 'Checking authorization before dispatch…';
    this.requestUpdate();
    try {
      const pending = this.request(lifetime, scope, body);
      body = '';
      const raw = await pending;
      this.requireCurrent(lifetime);
      this.applyMutation(scope, action, revision, raw);
    } catch (error) {
      this.mutationFailed(lifetime, error);
    } finally {
      body = '';
      if (this.current(lifetime)) {
        this.sent = false;
        this.busy = false;
        this.requestUpdate();
      }
    }
  }
  private async choose(event: Event, confirmation: Confirmation) {
    if (this.busy || this.stale || this.blocked || this.pending) return;
    this.clearSecrets();
    this.pending = confirmation;
    this.trigger = event.currentTarget instanceof HTMLButtonElement ? event.currentTarget : null;
    this.requestUpdate();
    await this.updateComplete;
    if (this.pending === confirmation) this.querySelector<HTMLButtonElement>('[data-confirm]')?.focus();
  }
  private async cancel() {
    const trigger = this.trigger;
    this.pending = null;
    this.trigger = null;
    this.requestUpdate();
    await this.updateComplete;
    if (trigger?.isConnected) trigger.focus();
  }
  private confirm() {
    const pending = this.pending;
    if (pending)
      void this.mutate(
        pending.scope,
        String(pending.body.action),
        JSON.stringify(pending.body),
        String(pending.body.revision)
      );
  }
  private hostActionBody(action: string, host: Host): Record<string, unknown> {
    const body: Record<string, unknown> = {action, revision: host.revision, confirm: action};
    if (action === 'exit-node')
      Object.assign(body, {
        revision: this.exitRevision,
        exit_node: this.exitNode,
        allow_lan: !!this.exitNode && this.allowLAN,
      });
    if (action === 'advertise-exit-node')
      Object.assign(body, {revision: this.advertiseRevision, advertise: this.advertise});
    return body;
  }
  private hostAction(event: Event, action: string) {
    const host = this.settings?.host;
    if (!host) return;
    if (action === 'exit-node' && this.exitNode.startsWith('missing:')) {
      this.notice = 'Select an available exit node or explicitly choose None before applying.';
      this.requestUpdate();
      return;
    }
    if (action === 'signin' || action === 'authentication') {
      void this.mutate('host', action, JSON.stringify({action, revision: host.revision}), host.revision);
      return;
    }
    void this.choose(event, {
      scope: 'host',
      body: this.hostActionBody(action, host),
      label: action,
      warning: hostWarning(action),
    });
  }
  private enrollmentSubmitBlocked(action: string, form: HTMLFormElement) {
    if (this.busy || this.stale || this.blocked || !['check', 'save', 'rotate'].includes(action)) return true;
    return !form.reportValidity();
  }
  private enrollmentNeedsReview(action: string, data: FormData) {
    return action !== 'check' && data.get('reviewed') !== 'on';
  }
  private enrollmentPayload(action: string, data: FormData) {
    const tags = this.tags
      .split(',')
      .map((tag) => tag.trim())
      .sort();
    return JSON.stringify({
      action,
      revision: this.policyRevision,
      tailnet: this.network,
      tags,
      preauthorized: this.preauthorized,
      client_id: formText(data, 'client_id'),
      client_secret: formText(data, 'client_secret'),
    });
  }
  private submitEnrollment(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || !(event.submitter instanceof HTMLButtonElement)) return;
    const form = event.currentTarget,
      action = event.submitter.value;
    if (this.enrollmentSubmitBlocked(action, form)) {
      this.clearSecrets();
      return;
    }
    const data = new FormData(form);
    if (this.enrollmentNeedsReview(action, data)) {
      this.clearSecrets();
      data.delete('client_secret');
      this.notice = 'Review and confirm the exposure/binding change before saving.';
      this.requestUpdate();
      return;
    }
    let body = this.enrollmentPayload(action, data);
    data.delete('client_secret');
    this.clearSecrets();
    void this.mutate('enrollment', action, body, this.policyRevision);
    body = '';
  }
  private controlsDisabled() {
    return this.busy || this.stale || this.blocked || !!this.pending;
  }
  private refreshObservations() {
    this.resume();
    void this.refresh();
  }
  private discardHostDraft() {
    this.resetHost(true);
  }
  private markEnrollmentDirty() {
    this.enrollmentDirty = true;
  }
  private onConfirmKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      event.preventDefault();
      void this.cancel();
    }
  }
  private onSignin(event: Event) {
    this.hostAction(event, 'signin');
  }
  private onAuthentication(event: Event) {
    this.hostAction(event, 'authentication');
  }
  private onLogout(event: Event) {
    this.hostAction(event, 'logout');
  }
  private onRefreshForgejo(event: Event) {
    this.hostAction(event, 'refresh-forgejo');
  }
  private onApplyExitNode(event: Event) {
    this.hostAction(event, 'exit-node');
  }
  private onApplyAdvertise(event: Event) {
    this.hostAction(event, 'advertise-exit-node');
  }
  private onExitNodeChange(event: Event) {
    if (!(event.target instanceof HTMLSelectElement)) return;
    this.exitNode = event.target.value;
    if (!this.exitNode) this.allowLAN = false;
    this.exitDirty = true;
    this.requestUpdate();
  }
  private onAllowLANChange(event: Event) {
    if (event.target instanceof HTMLInputElement) {
      this.allowLAN = event.target.checked;
      this.exitDirty = true;
      this.requestUpdate();
    }
  }
  private onAdvertiseChange(event: Event) {
    if (event.target instanceof HTMLInputElement) {
      this.advertise = event.target.checked;
      this.advertiseDirty = true;
      this.requestUpdate();
    }
  }
  private onModeChange(event: Event) {
    if (!(event.target instanceof HTMLSelectElement)) return;
    this.mode = event.target.value === 'rotate' ? 'rotate' : 'save';
    this.resetEnrollment();
  }
  private onNetworkInput(event: Event) {
    if (event.target instanceof HTMLInputElement) this.network = event.target.value;
  }
  private onTagsInput(event: Event) {
    if (event.target instanceof HTMLInputElement) this.tags = event.target.value;
  }
  private onPreauthorizedChange(event: Event) {
    if (event.target instanceof HTMLInputElement) this.preauthorized = event.target.checked;
  }
  private enrollmentRevision() {
    return this.settings?.enrollment.revision || '0';
  }
  private onCloseAdmission(event: Event) {
    void this.choose(event, {
      scope: 'enrollment',
      body: {action: 'disable', revision: this.enrollmentRevision()},
      label: 'close admission',
      warning:
        'Close future enrollment and keep the new-project default Off. Existing device connections and project policies are not disconnected, revoked or deleted.',
    });
  }
  private onOfferManagedDefault(event: Event) {
    void this.choose(event, {
      scope: 'enrollment',
      body: {action: 'default', revision: this.enrollmentRevision(), default: true},
      label: 'managed creation default',
      warning:
        'Offer managed access preselected in future Create forms. The creator must submit the reviewed policy binding and revision. Existing projects and legacy requests stay unchanged.',
    });
  }
  private onKeepDefaultOff(event: Event) {
    void this.choose(event, {
      scope: 'enrollment',
      body: {action: 'default', revision: this.enrollmentRevision(), default: false},
      label: 'default Off',
      warning: 'Save Off for future creation defaults. Existing project settings and connections do not change.',
    });
  }
  private renderReconnect() {
    if (!(this.blocked || !this.lifetime)) return '';
    return html`<p>
      <a href=${this.apiBase + '/login?destination=tailnet&expected_user_id=' + this.actor}>Reconnect Soda explicitly</a
      >. Native CLI/console recovery remains available.
    </p>`;
  }
  private renderPending() {
    const pending = this.pending;
    if (!pending) return '';
    return html`<section
      class="tailnet-notice"
      role="region"
      aria-labelledby="tailnet-confirm-title"
      @keydown=${this.onConfirmKeydown}
    >
      <h2 id="tailnet-confirm-title">Confirm ${pending.label}</h2>
      <p>${pending.warning}</p>
      <div class="settings-actions">
        <button type="button" data-confirm @click=${this.confirm}>Confirm ${pending.label}</button
        ><button type="button" @click=${this.cancel}>Cancel</button>
      </div>
    </section>`;
  }
  private renderHostStatus(host: Host | null | undefined) {
    if (this.stale) return html`<p>Appliance observations are stale. No usable endpoint is asserted.</p>`;
    if (!host)
      return html`<p>
        Host observation unavailable, not disconnected. Verify the native daemon, reviewed version and helper
        configuration through an approved private path.
      </p>`;
    return html`
      <dl>
        <dt>Native state</dt>
        <dd>${host.state}${host.expired ? ' · expired' : ''}</dd>
        <dt>Network</dt>
        <dd>${host.tailnet || 'Not observed'}</dd>
        <dt>Device</dt>
        <dd>${host.dns_name || 'Not observed'}</dd>
        <dt>Observed addresses (not verified reachable)</dt>
        <dd>${host.addresses.join(', ') || 'None observed'}</dd>
        <dt>MagicDNS</dt>
        <dd>${host.magic_dns_enabled ? 'Enabled' : 'Disabled / not observed'}</dd>
        <dt>Native health issues</dt>
        <dd>${host.health_issues} — raw diagnostics are intentionally hidden.</dd>
      </dl>
      ${host.state === 'NeedsMachineAuth' ? html`<p>Device approval is required in Tailscale. Tailnet Lock signing is not automated; do not weaken Lock or copy signing keys.</p>` : ''}
    `;
  }
  private renderAuthLink() {
    if (!this.authURL) return '';
    return html`<p>
      <a data-authentication href=${this.authURL} target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer"
        >Continue appliance sign-in at Tailscale</a
      >. This is provider authentication, not Soda sign-in. Do not copy this link into logs or evidence.
    </p>`;
  }
  private renderUnavailableExitOption(options: ExitChoice[]) {
    if (!(this.exitNode && !options.some((choice) => choice.address === this.exitNode))) return '';
    return html`<option value=${this.exitNode} selected>Previous selection (unavailable)</option>`;
  }
  private readonly renderExitOption = (choice: ExitChoice) => {
    const {peer, address} = choice;
    return html`<option
      value=${address}
      ?selected=${address === this.exitNode}
      ?disabled=${!peer.online || peer.expired}
    >
      ${peer.dns_name || peer.id}${!peer.online || peer.expired ? ' (unavailable)' : ''}
    </option>`;
  };
  private renderExitPreferences() {
    const host = this.settings?.host;
    if (!host) return '';
    const options = exitPeerOptions(host, this.exitNode);
    return html`
      <fieldset ?disabled=${this.controlsDisabled()}>
        <legend>Exit-node preferences</legend>
        ${exitNodeMissing(host) ? html`<p role="alert">The selected native exit-node ID is missing from the peer observation. It has not been cleared. Select an available replacement or explicitly choose None.</p>` : ''}
        <label
          >Exit node<select aria-label="Exit node" .value=${this.exitNode} @change=${this.onExitNodeChange}>
            <option value="" ?selected=${this.exitNode === ''}>None (clear only on Apply)</option>
            ${this.renderUnavailableExitOption(options)} ${options.map(this.renderExitOption)}
          </select></label
        >
        <label
          ><input
            type="checkbox"
            .checked=${this.allowLAN}
            ?disabled=${!this.exitNode}
            @change=${this.onAllowLANChange}
          />Allow local LAN while using exit node</label
        >
        <button type="button" @click=${this.onApplyExitNode}>Apply exit node</button>
        <label
          ><input type="checkbox" .checked=${this.advertise} @change=${this.onAdvertiseChange} />Advertise appliance as
          an exit node</label
        >
        <button type="button" @click=${this.onApplyAdvertise}>Apply advertisement</button>
        <p>
          Provider approval is separate from advertisement and online status. Existing subnet routes are not edited.
        </p>
        <button type="button" @click=${this.discardHostDraft}>Discard host draft / use latest observation</button>
        ${this.hostDirty ? html`<p>Unsaved host draft; its original revision is retained across refresh.</p>` : ''}
      </fieldset>
    `;
  }
  private renderPeerList(host: Host) {
    if (this.stale) return '';
    return html`<details>
      <summary>Observed peers (${host.peers.length})</summary>
      <ul>
        ${host.peers.map(renderPeerItem)}
      </ul>
    </details>`;
  }
  private renderHostControls(host: Host) {
    return html`
      <fieldset ?disabled=${this.controlsDisabled()}>
        <legend>Appliance connection</legend>
        <div class="settings-actions">
          <button type="button" @click=${this.onSignin}>
            ${host.have_node_key ? 'Resume / reauthenticate' : 'Sign in appliance'}
          </button>
          <button type="button" @click=${this.onAuthentication}>Recover authentication link</button>
          <button type="button" @click=${this.onLogout}>Disconnect appliance</button>
          <button type="button" @click=${this.onRefreshForgejo}>Refresh Forgejo advertisement</button>
        </div>
      </fieldset>
      ${this.renderAuthLink()} ${this.renderExitPreferences()} ${this.renderPeerList(host)}
    `;
  }
  private renderHostSection(settings: Settings) {
    const host = settings.host;
    return html`<section aria-labelledby="tailnet-appliance">
      <h2 id="tailnet-appliance">${this.dataset.applianceLabel || 'Appliance'}</h2>
      <p>Persistent native connection. Host identity and ordinary SSH authentication are not shared with projects.</p>
      ${this.renderHostStatus(host)} ${host ? this.renderHostControls(host) : ''}
    </section>`;
  }
  private renderEnrollmentSummary(policy: Enrollment) {
    return html`
      <p>
        ${policy.configured ? 'Configured' : 'Not configured'} · Credential check
        ${policy.credential_checked ? 'previously passed' : 'not recorded'} · Enrollment not verified.
      </p>
      <p>
        Managed network: ${policy.tailnet || 'None'}. Future admission:
        ${policy.admission ? 'configured open' : 'closed'}. New-project default:
        ${policy.default ? 'managed, explicitly reviewed in Create' : 'Off'}.
      </p>
      <p>
        ${policy.runtime_supported ? 'Native project supervision is configured. Each managed project gets a separate ephemeral identity when it runs; token acceptance alone is not enrollment or connectivity proof.' : 'Project runtime is not configured. Automatic defaults remain unavailable.'}
        Existing projects remain Off until explicitly selected. Host login never enrolls projects.
      </p>
      <p>
        Tailscale owns network access policy. A project tag alone does not isolate host/peers. Approval-required
        networks need explicitly permitted preauthorization; automatic Tailnet Lock signing is unsupported.
      </p>
      <p>
        <a href="https://tailscale.com/kb/1215/oauth-clients" target="_blank" rel="noopener noreferrer"
          >Create a restricted Tailscale OAuth client</a
        >
        with auth_keys and only the selected tags. Token acceptance does not prove target, scope, expiry or enrollment;
        no hidden test device is created.
      </p>
    `;
  }
  private renderEnrollmentForm() {
    return html`
      <form @submit=${this.submitEnrollment} @input=${this.markEnrollmentDirty}>
        <label
          >Managed Tailnet<input
            required
            maxlength="253"
            .value=${this.network}
            ?readonly=${this.mode === 'rotate'}
            @input=${this.onNetworkInput}
        /></label>
        <label
          >Project tags (comma separated)<input
            required
            .value=${this.tags}
            ?readonly=${this.mode === 'rotate'}
            @input=${this.onTagsInput}
        /></label>
        <label
          ><input
            type="checkbox"
            .checked=${this.preauthorized}
            ?disabled=${this.mode === 'rotate'}
            @change=${this.onPreauthorizedChange}
          />Explicitly preauthorize devices if provider policy permits</label
        >
        <label>OAuth client ID<input name="client_id" required maxlength="128" autocomplete="off" /></label>
        <label
          >OAuth client secret<input
            name="client_secret"
            type="password"
            required
            maxlength="525"
            autocomplete="new-password"
            spellcheck="false"
        /></label>
        <p>
          Secret is cleared when submitted, on failure and on departure. It is never displayed or stored in browser
          storage. Re-enter it for a subsequent Save after Check.
        </p>
        <label
          ><input type="checkbox" name="reviewed" />I reviewed private-network exposure and this binding change.
          Replacement requires explicit project reselection; rotation preserves tags and existing device
          identities.</label
        >
        <div class="settings-actions">
          <button type="submit" value="check">Check credential only</button
          ><button type="submit" value=${this.mode}>
            ${this.mode === 'rotate' ? 'Rotate credential' : 'Save binding'}
          </button>
          <button type="button" @click=${this.resetEnrollment}>Discard enrollment draft</button>
        </div>
      </form>
    `;
  }
  private renderEnrollmentAdmission(policy: Enrollment) {
    if (!policy.configured) return '';
    return html`<div class="settings-actions">
      <button type="button" @click=${this.onCloseAdmission}>Close future admission</button>
      ${policy.runtime_supported && policy.admission ? html`<button type="button" @click=${this.onOfferManagedDefault}>Offer managed access by default</button>` : ''}
      <button type="button" @click=${this.onKeepDefaultOff}>Keep new-project default Off</button>
    </div>`;
  }
  private renderEnrollmentSection(policy: Enrollment | undefined) {
    if (!policy) return '';
    return html`<section aria-labelledby="tailnet-enrollment">
      <h2 id="tailnet-enrollment">${this.dataset.enrollmentLabel || 'Automatic project access'}</h2>
      ${this.renderEnrollmentSummary(policy)}
      <fieldset ?disabled=${this.controlsDisabled()}>
        <legend>Enrollment configuration</legend>
        <label
          >Configuration action<select
            aria-label="Configuration action"
            .value=${this.mode}
            @change=${this.onModeChange}
          >
            <option value="save">Save a new / replacement binding</option>
            <option value="rotate" ?disabled=${!policy.configured}>Rotate credential in this binding</option>
          </select></label
        >
        ${this.renderEnrollmentForm()} ${this.renderEnrollmentAdmission(policy)}
      </fieldset>
    </section>`;
  }
  private renderAuthorized() {
    const settings = this.settings;
    if (!settings || this.blocked) return '';
    return html` ${this.renderHostSection(settings)} ${this.renderEnrollmentSection(settings.enrollment)} `;
  }
  protected render() {
    return html`
      <div class="settings-actions">
        <button type="button" ?disabled=${this.busy || !!this.pending} @click=${this.refreshObservations}>
          Refresh observations
        </button>
      </div>
      <p role="status" aria-live="polite">${this.message}</p>
      ${this.notice ? html`<p class="tailnet-notice" role="status">${this.notice}</p>` : ''} ${this.renderReconnect()}
      ${this.renderPending()} ${this.renderAuthorized()}
    `;
  }
}
customElements.define('soda-tailnet', SodaTailnet);
export function mountTailnetPage(root: HTMLElement, actor: string, session: Session) {
  const page = new SodaTailnet();
  page.configure(actor, session);
  page.dataset.applianceLabel = root.dataset.applianceLabel || 'Appliance';
  page.dataset.enrollmentLabel = root.dataset.enrollmentLabel || 'Automatic project access';
  root.append(page);
}
