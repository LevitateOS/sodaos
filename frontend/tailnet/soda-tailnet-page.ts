import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from '../spaces/soda-extension.js';
import type {Enrollment, Settings} from './soda-tailnet-response.js';
import {refresh, request, resetEnrollment, resetHost} from './soda-tailnet-observation.js';
import type {ObservationInput} from './soda-tailnet-observation.js';
import {cancel, choose, confirm, hostAction, submitEnrollment, unknownOutcome} from './soda-tailnet-actions.js';
import type {ActionsInput, Confirmation} from './soda-tailnet-actions.js';
import {renderHostSection} from './soda-tailnet-host-view.js';
import type {HostViewInput} from './soda-tailnet-host-view.js';

class SodaTailnet extends LitElement {
  private transport: PreparedExtensionMount | null = null;
  configure(transport: PreparedExtensionMount) {
    if (this.transport) throw Error('Tailnet page already configured');
    this.transport = transport;
  }
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
    if (!document.hidden && !this.hostDirty && !this.enrollmentDirty) void refresh(this.observationInput());
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
    window.addEventListener('pagehide', this.hiddenPage);
    window.addEventListener('pageshow', this.shownPage);
    window.addEventListener('beforeunload', this.departure);
    document.addEventListener('visibilitychange', this.visibility);
    this.resume();
  }
  disconnectedCallback() {
    this.retire();
    window.removeEventListener('pagehide', this.hiddenPage);
    window.removeEventListener('pageshow', this.shownPage);
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
    this.message = 'Tailnet controls retired. Refresh this Forgejo page to continue.';
    this.requestUpdate();
  }
  private resume() {
    if (!this.transport || this.lifetime || !this.isConnected) return;
    this.clearSecrets();
    this.lifetime = new AbortController();
    this.timer = setInterval(() => {
      if (!document.hidden && !this.hostDirty && !this.enrollmentDirty && !this.blocked)
        void refresh(this.observationInput());
    }, 10000);
    void refresh(this.observationInput());
  }
  private current(lifetime: AbortController) {
    return this.isConnected && this.lifetime === lifetime && !lifetime.signal.aborted;
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
  private observationInput(): ObservationInput {
    return {
      requireLive: (lifetime) => this.requireCurrent(lifetime),
      isLive: (lifetime) => this.current(lifetime),
      transportRequest: (path, init) => this.transport!.request(path, init),
      noteSent: () => {
        this.sent = true;
      },
      loseAuthorization: () => this.loseAuthorization(),
      readSettings: () => this.settings,
      writeSettings: (settings) => {
        this.settings = settings;
      },
      readLifetime: () => this.lifetime,
      hasPending: () => !!this.pending,
      isBusy: () => this.busy,
      setBusy: (busy) => {
        this.busy = busy;
      },
      readBlocked: () => this.blocked,
      setBlocked: (blocked) => {
        this.blocked = blocked;
      },
      setStale: (stale) => {
        this.stale = stale;
      },
      setMessage: (message) => {
        this.message = message;
      },
      clearSecrets: () => this.clearSecrets(),
      clearAuthLink: () => this.clearAuthURL(),
      readExitDirty: () => this.exitDirty,
      readAdvertiseDirty: () => this.advertiseDirty,
      writeExitDirty: (dirty) => {
        this.exitDirty = dirty;
      },
      writeAdvertiseDirty: (dirty) => {
        this.advertiseDirty = dirty;
      },
      setExitDraft: (revision, node, allowLAN) => {
        this.exitRevision = revision;
        this.exitNode = node;
        this.allowLAN = allowLAN;
      },
      setAdvertiseDraft: (revision, advertise) => {
        this.advertiseRevision = revision;
        this.advertise = advertise;
      },
      setEnrollmentDraft: (revision, network, tags, preauthorized) => {
        this.policyRevision = revision;
        this.network = network;
        this.tags = tags;
        this.preauthorized = preauthorized;
      },
      readEnrollmentDirty: () => this.enrollmentDirty,
      writeEnrollmentDirty: (dirty) => {
        this.enrollmentDirty = dirty;
      },
      requestUpdate: () => this.requestUpdate(),
    };
  }
  private clearAuthURL() {
    this.authURL = '';
    this.querySelectorAll<HTMLAnchorElement>('a[data-authentication]').forEach((link) => link.removeAttribute('href'));
  }
  private actionsInput(): ActionsInput {
    return {
      requireLive: (lifetime) => this.requireCurrent(lifetime),
      isLive: (lifetime) => this.current(lifetime),
      readLifetime: () => this.lifetime,
      sendRequest: (lifetime, scope, body) => request(this.observationInput(), lifetime, scope, body),
      readSettings: () => this.settings,
      writeSettings: (settings) => {
        this.settings = settings;
      },
      isBusy: () => this.busy,
      setBusy: (busy) => {
        this.busy = busy;
      },
      isStale: () => this.stale,
      setStale: (stale) => {
        this.stale = stale;
      },
      isBlocked: () => this.blocked,
      setNotice: (notice) => {
        this.notice = notice;
      },
      appendNotice: (suffix) => {
        this.notice += suffix;
      },
      readSent: () => this.sent,
      writeSent: (sent) => {
        this.sent = sent;
      },
      readPending: () => this.pending,
      writePending: (pending) => {
        this.pending = pending;
      },
      readTrigger: () => this.trigger,
      writeTrigger: (trigger) => {
        this.trigger = trigger;
      },
      writeExitDirty: (dirty) => {
        this.exitDirty = dirty;
      },
      writeAdvertiseDirty: (dirty) => {
        this.advertiseDirty = dirty;
      },
      readEnrollmentDirty: () => this.enrollmentDirty,
      readHostDrafts: () => ({
        exitRevision: this.exitRevision,
        exitNode: this.exitNode,
        allowLAN: this.allowLAN,
        advertiseRevision: this.advertiseRevision,
        advertise: this.advertise,
      }),
      readEnrollmentDrafts: () => ({
        policyRevision: this.policyRevision,
        network: this.network,
        tags: this.tags,
        preauthorized: this.preauthorized,
      }),
      syncHostDraft: (discard) => resetHost(this.observationInput(), discard),
      syncEnrollmentDraft: () => resetEnrollment(this.observationInput()),
      clearSecrets: () => this.clearSecrets(),
      setAuthURL: (url) => {
        this.authURL = url;
      },
      updated: () => this.updateComplete,
      focusConfirm: () => this.querySelector<HTMLButtonElement>('[data-confirm]')?.focus(),
      requestUpdate: () => this.requestUpdate(),
    };
  }
  private controlsDisabled() {
    return this.busy || this.stale || this.blocked || !!this.pending;
  }
  private refreshObservations() {
    this.resume();
    void refresh(this.observationInput());
  }
  private discardHostDraft() {
    resetHost(this.observationInput(), true);
  }
  private markEnrollmentDirty() {
    this.enrollmentDirty = true;
  }
  private onConfirmKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      event.preventDefault();
      void cancel(this.actionsInput());
    }
  }
  private onSignin(event: Event) {
    hostAction(this.actionsInput(), event, 'signin');
  }
  private onAuthentication(event: Event) {
    hostAction(this.actionsInput(), event, 'authentication');
  }
  private onLogout(event: Event) {
    hostAction(this.actionsInput(), event, 'logout');
  }
  private onRefreshForgejo(event: Event) {
    hostAction(this.actionsInput(), event, 'refresh-forgejo');
  }
  private onApplyExitNode(event: Event) {
    hostAction(this.actionsInput(), event, 'exit-node');
  }
  private onApplyAdvertise(event: Event) {
    hostAction(this.actionsInput(), event, 'advertise-exit-node');
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
    resetEnrollment(this.observationInput());
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
    void choose(this.actionsInput(), event, {
      scope: 'enrollment',
      body: {action: 'disable', revision: this.enrollmentRevision()},
      label: 'close admission',
      warning:
        'Close future enrollment and keep the new-project default Off. Existing device connections and project policies are not disconnected, revoked or deleted.',
    });
  }
  private onOfferManagedDefault(event: Event) {
    void choose(this.actionsInput(), event, {
      scope: 'enrollment',
      body: {action: 'default', revision: this.enrollmentRevision(), default: true},
      label: 'managed creation default',
      warning:
        'Offer managed access preselected in future Create forms. The creator must submit the reviewed policy binding and revision. Existing projects and legacy requests stay unchanged.',
    });
  }
  private onKeepDefaultOff(event: Event) {
    void choose(this.actionsInput(), event, {
      scope: 'enrollment',
      body: {action: 'default', revision: this.enrollmentRevision(), default: false},
      label: 'default Off',
      warning: 'Save Off for future creation defaults. Existing project settings and connections do not change.',
    });
  }
  private renderReconnect() {
    if (!(this.blocked || !this.lifetime)) return '';
    return html`<p>
      Refresh this Forgejo page to restore operator authorization. Native CLI/console recovery remains available.
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
        <button type="button" data-confirm @click=${() => confirm(this.actionsInput())}>Confirm ${pending.label}</button
        ><button type="button" @click=${() => cancel(this.actionsInput())}>Cancel</button>
      </div>
    </section>`;
  }
  private hostViewInput(): HostViewInput {
    return {
      isStale: () => this.stale,
      readAuthURL: () => this.authURL,
      readExitNode: () => this.exitNode,
      readHost: () => this.settings?.host,
      controlsDisabled: () => this.controlsDisabled(),
      readAllowLAN: () => this.allowLAN,
      readAdvertise: () => this.advertise,
      isHostDirty: () => this.hostDirty,
      readApplianceLabel: () => this.dataset.applianceLabel || 'Appliance',
      onExitNodeChange: (event) => this.onExitNodeChange(event),
      onAllowLANChange: (event) => this.onAllowLANChange(event),
      onApplyExitNode: (event) => this.onApplyExitNode(event),
      onAdvertiseChange: (event) => this.onAdvertiseChange(event),
      onApplyAdvertise: (event) => this.onApplyAdvertise(event),
      onDiscardHostDraft: () => this.discardHostDraft(),
      onSignin: (event) => this.onSignin(event),
      onAuthentication: (event) => this.onAuthentication(event),
      onLogout: (event) => this.onLogout(event),
      onRefreshForgejo: (event) => this.onRefreshForgejo(event),
    };
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
      <form
        @submit=${(event: SubmitEvent) => submitEnrollment(this.actionsInput(), event)}
        @input=${this.markEnrollmentDirty}
      >
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
          <button type="button" @click=${() => resetEnrollment(this.observationInput())}>
            Discard enrollment draft
          </button>
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
    return html`
      ${renderHostSection(this.hostViewInput(), settings)} ${this.renderEnrollmentSection(settings.enrollment)}
    `;
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
export function mountTailnetPage(root: HTMLElement, transport: PreparedExtensionMount) {
  const page = new SodaTailnet();
  page.configure(transport);
  page.dataset.applianceLabel = root.dataset.applianceLabel || 'Appliance';
  page.dataset.enrollmentLabel = root.dataset.enrollmentLabel || 'Automatic project access';
  root.append(page);
  return {dispose: () => page.remove()};
}
