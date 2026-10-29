import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, object, SodaRequestError} from './sodaspaces-api.js';
import type {ApplicationRequest} from './sodaspaces-applications.js';
import {applicationsResponse, type ApplicationRecord} from './sodaspaces-application-response.js';

// Typed API refusals the builds endpoints already return (see
// internal/web/api/application_builds.go and application_actions.go). A 409
// delivery_unavailable means the delivery has no current verified bundle or
// its head changed: retrying or refreshing traps the user when there may be
// no setup at all, so the refusal names the condition and points at Feed
// instead of refresh. Service and read failures (503) never assert absence:
// the outcome is unconfirmed and the refresh path stays safe. A 403
// operator_required is a permission refusal, never a session change.
// Packaging outcomes the server did not confirm keep the refresh path so the
// same delivery can be reconciled; a replace whose reply was lost is
// reconciled against the same application instead. Pending packaging names
// the in-progress operation and never claims nothing was prepared; 401 keeps
// the reconnect path.
const unavailableDeliveryCodes = new Set(['delivery_unavailable', 'build_changed', 'build_replacement_conflict']);
const serviceDeliveryCodes = new Set(['builds_unavailable', 'native_unavailable', 'store_unavailable']);
const unconfirmedDeliveryCodes = new Set(['build_unconfirmed', 'native_outcome_unconfirmed']);
const pendingDeliveryCodes = new Set(['build_pending', 'application_pending', 'lifecycle_pending']);

// The operation a delivery request was running when it failed. Pending and
// unconfirmed failures name the operation that was attempted and the one
// that is already in progress, so the user can reconcile the same operation
// instead of retrying blindly. Refusal copy scopes every absence claim to
// the affected environment; other environments are never described.
interface DeliveryOperation {
  kind: 'refresh' | 'prepare' | 'setup' | 'apply';
  environment?: 'company' | 'preview';
  target?: string;
}

function deliveryErrorStatus(error: unknown): number | undefined {
  if (error instanceof SodaRequestError) return error.status;
  if (typeof error === 'object' && error !== null) {
    const status = (error as {status?: unknown}).status;
    if (typeof status === 'number' && Number.isSafeInteger(status)) return status;
  }
  return undefined;
}

function deliveryErrorCode(error: unknown): string | undefined {
  if (error instanceof SodaRequestError) return error.code;
  if (typeof error === 'object' && error !== null) {
    const code = (error as {code?: unknown}).code;
    if (typeof code === 'string') return code;
  }
  return undefined;
}

function unknownDeliveryOutcome(error: unknown): boolean {
  const code = deliveryErrorCode(error);
  if (code) return unconfirmedDeliveryCodes.has(code);
  const status = deliveryErrorStatus(error);
  // A lost reply carries no status at all; any other codeless 5xx likewise
  // leaves the native outcome unknown.
  return status === undefined || status >= 500;
}

interface PreparedBuild {
  id: string;
  delivery: string;
  source: string;
  name: string;
  origin: string;
  image: string;
  setup: string;
  environment: 'company' | 'preview';
}
const buildID = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{32}$/.test(value);
function preparedBuild(value: unknown): PreparedBuild {
  const row = object(value),
    artifact = object(row.artifact),
    delivery = object(row.delivery);
  check(
    (artifact.environment === 'company' || artifact.environment === 'preview') &&
      artifact.environment === delivery.environment
  );
  check(
    buildID(row.id) &&
      buildID(delivery.delivery_id) &&
      typeof delivery.source_commit === 'string' &&
      /^[a-f0-9]{40}$/.test(delivery.source_commit)
  );
  check(
    typeof artifact.name === 'string' &&
      typeof artifact.origin === 'string' &&
      typeof artifact.image === 'string' &&
      /^sha256:[a-f0-9]{64}$/.test(artifact.image)
  );
  const origin = new URL(artifact.origin);
  check(origin.protocol === 'https:' && origin.origin === artifact.origin && !origin.username && !origin.password);
  check(row.setup_id === undefined || (typeof row.setup_id === 'string' && /^a[a-f0-9]{24}$/.test(row.setup_id)));
  return {
    id: row.id,
    delivery: delivery.delivery_id,
    source: delivery.source_commit,
    name: artifact.name,
    origin: artifact.origin,
    image: artifact.image,
    setup: typeof row.setup_id === 'string' ? row.setup_id : '',
    environment: artifact.environment,
  };
}

// An explicit bridge from verified work to existing app lifecycle owners.
// Browsing reads receipts. Only a person's action packages, sets up or updates.
export class SodaApplicationDelivery extends LitElement {
  static properties = {
    request: {attribute: false},
    active: {type: Boolean},
    sessionKey: {attribute: false},
    repository: {type: String},
    requestedDelivery: {type: String},
    applications: {attribute: false},
    builds: {state: true},
    busy: {state: true},
    notice: {state: true},
    deliveryRefused: {state: true},
    administrator: {state: true},
    confirmations: {state: true},
  };
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
  declare sessionKey: string;
  declare repository: string;
  declare requestedDelivery: string;
  declare applications: ApplicationRecord[];
  declare private builds: PreparedBuild[];
  declare private busy: boolean;
  declare private notice: string;
  declare private deliveryRefused: boolean;
  declare private administrator: string;
  declare private confirmations: Set<string>;
  private controller: AbortController | undefined;
  private epoch = 0;
  constructor() {
    super();
    this.active = false;
    this.sessionKey = '';
    this.repository = '';
    this.requestedDelivery = '';
    this.applications = [];
    this.builds = [];
    this.busy = false;
    this.notice = '';
    this.deliveryRefused = false;
    this.administrator = '';
    this.confirmations = new Set();
  }
  createRenderRoot() {
    return this;
  }
  protected updated(changes: PropertyValues<this>) {
    if (
      changes.has('sessionKey') ||
      changes.has('repository') ||
      changes.has('requestedDelivery') ||
      (changes.has('active') && !this.active)
    )
      this.clear();
    if (
      this.active &&
      this.repository &&
      (changes.has('active') ||
        changes.has('sessionKey') ||
        changes.has('repository') ||
        changes.has('requestedDelivery') ||
        changes.has('request'))
    )
      void this.refresh();
  }
  disconnectedCallback() {
    this.clear();
    super.disconnectedCallback();
  }
  private clear() {
    this.epoch++;
    this.controller?.abort();
    this.builds = [];
    this.administrator = '';
    this.confirmations = new Set();
    this.notice = '';
    this.deliveryRefused = false;
    this.busy = false;
  }
  private get path() {
    return `/api/repositories/${this.repository}/application-builds`;
  }
  private async run(
    work: (request: ApplicationRequest, signal: AbortSignal) => Promise<void>,
    operation?: DeliveryOperation
  ) {
    if (!this.active || this.busy || !this.request || !/^[1-9][0-9]*$/.test(this.repository)) return;
    const epoch = this.epoch,
      controller = new AbortController(),
      request = this.request;
    this.controller = controller;
    this.busy = true;
    this.notice = '';
    this.deliveryRefused = false;
    try {
      await work(async (...args) => {
        const result = await request(...args);
        if (this.epoch !== epoch || controller.signal.aborted) throw Error('inactive');
        return result;
      }, controller.signal);
    } catch (error) {
      if (this.epoch === epoch) this.failDelivery(error, operation);
    } finally {
      if (this.epoch === epoch) {
        this.busy = false;
        this.controller = undefined;
      }
    }
  }
  private applyUnconfirmedNotice(target?: string) {
    return target
      ? `The apply outcome for ${target} could not be confirmed. Refresh applications and inspect ${target} to reconcile the same apply before applying again.`
      : 'The apply outcome could not be confirmed. Refresh applications and inspect the application to reconcile the same apply before applying again.';
  }
  private failDelivery(error: unknown, operation?: DeliveryOperation) {
    // The loaded builds and environment stay exactly as received: a refusal
    // never claims retained setup, and nothing is retried automatically.
    const status = deliveryErrorStatus(error),
      code = deliveryErrorCode(error);
    if (status === 401) {
      this.deliveryRefused = false;
      this.notice =
        'Your Soda session changed. Reconnect to Soda Forge, then retry this delivery; nothing was prepared or changed.';
      return;
    }
    if (status === 403) {
      this.deliveryRefused = false;
      this.notice =
        'Preparing a delivery needs an operator. Ask an application operator to prepare it; ordinary accounts cannot prepare versions.';
      return;
    }
    if ((code && serviceDeliveryCodes.has(code)) || (!code && status === 503)) {
      this.deliveryRefused = false;
      this.notice =
        'Application packaging is unavailable right now. The outcome was not confirmed; refresh prepared versions before acting again.';
      return;
    }
    if ((code && unavailableDeliveryCodes.has(code)) || (!code && status === 409)) {
      this.deliveryRefused = true;
      const scope = operation?.environment;
      if (code === 'build_changed' || code === 'build_replacement_conflict')
        this.notice = scope
          ? `This ${scope} delivery changed: the prepared version no longer matches the current verified delivery. Follow its checks in Feed; nothing was prepared for this ${scope} delivery and no ${scope} setup is retained.`
          : 'This delivery changed: the prepared version no longer matches the current verified delivery. Follow its checks in Feed; nothing was prepared and no setup is retained.';
      else
        this.notice = scope
          ? `This delivery has no current verified ${scope} bundle. Follow its checks in Feed; nothing was prepared for this ${scope} delivery and no ${scope} setup is retained.`
          : 'This delivery is unavailable: it has no current verified application bundle. Follow its checks in Feed; nothing was prepared and no setup is retained.';
      return;
    }
    if (code === 'build_setup_conflict') {
      this.deliveryRefused = false;
      this.notice =
        'This version cannot reserve a new app at its address. An existing setup may already be retained; refresh prepared versions and inspect it before acting again.';
      return;
    }
    if (code === 'build_origin_conflict') {
      this.deliveryRefused = false;
      this.notice =
        'This delivery cannot be prepared at its address: use an application address separate from Soda Forge. Refresh prepared versions before acting again.';
      return;
    }
    if (code && pendingDeliveryCodes.has(code)) {
      this.deliveryRefused = false;
      if (operation?.kind === 'apply') {
        this.notice = operation.target
          ? `An operation on ${operation.target} is already in progress. This apply was not submitted again; Refresh applications and inspect ${operation.target} to reconcile the same apply before acting again.`
          : 'An application operation is already in progress. This apply was not submitted again; Refresh applications and inspect the application to reconcile the same apply before acting again.';
        return;
      }
      if (code === 'build_pending') {
        const scope = operation?.environment ? `this ${operation.environment} delivery` : 'this delivery';
        this.notice = `Packaging ${scope} is already in progress. This preparation was not submitted again; wait for the in-progress packaging, then Refresh prepared versions to reconcile the same delivery.`;
        return;
      }
      this.notice =
        'An application operation is already in progress. Nothing was submitted again; Refresh prepared versions to reconcile before retrying.';
      return;
    }
    if (code && unconfirmedDeliveryCodes.has(code)) {
      this.deliveryRefused = false;
      if (operation?.kind === 'apply') {
        this.notice = this.applyUnconfirmedNotice(operation.target);
        return;
      }
      this.notice =
        'The packaging outcome could not be confirmed. Refresh prepared versions to reconcile this same delivery before preparing again.';
      return;
    }
    this.deliveryRefused = false;
    this.notice =
      operation?.kind === 'apply'
        ? this.applyUnconfirmedNotice(operation.target)
        : 'The outcome could not be confirmed. Refresh prepared versions to reconcile this delivery before acting again.';
  }
  private openProjectFeed() {
    if (!this.active || !/^[1-9][0-9]*$/.test(this.repository)) return;
    this.dispatchEvent(
      new CustomEvent('soda-project-feed-open', {
        bubbles: true,
        composed: true,
        detail: {repository: this.repository},
      })
    );
  }
  private refresh() {
    return this.run(async (request, signal) => {
      const value = object(
        await request(
          this.path + (buildID(this.requestedDelivery) ? '?delivery_id=' + this.requestedDelivery : ''),
          undefined,
          signal
        )
      );
      check(Array.isArray(value.builds));
      const administrator = object(value.administrator);
      check(typeof administrator.name === 'string');
      this.builds = value.builds.map(preparedBuild);
      this.administrator = administrator.name;
    });
  }
  private prepare(environment: 'company' | 'preview') {
    if (!buildID(this.requestedDelivery)) return;
    const delivery = this.requestedDelivery;
    return this.run(
      async (request, signal) => {
        try {
          const build = preparedBuild(await request(this.path, {delivery_id: delivery, environment}, signal));
          this.builds = [build, ...this.builds.filter((row) => row.id !== build.id)];
          this.notice =
            'Application bytes prepared from this verified delivery. Owner setup and company release are separate actions.';
        } catch (error) {
          // An unconfirmed packaging reply may still have completed server-side.
          // Reconcile with a status read of the SAME delivery id — never a
          // second POST — and adopt the prepared version when the server
          // reports it.
          if (signal.aborted || !unknownDeliveryOutcome(error)) throw error;
          const reconciled = await this.readDeliveryBuild(request, signal, delivery, environment).catch(
            () => undefined
          );
          if (!reconciled) throw error;
          this.builds = [reconciled, ...this.builds.filter((row) => row.id !== reconciled.id)];
          this.notice =
            'The packaging reply was unconfirmed, so the same delivery was re-read: its prepared version is shown below. Owner setup and company release are separate actions.';
        }
      },
      {kind: 'prepare', environment}
    );
  }
  private async readDeliveryBuild(
    request: ApplicationRequest,
    signal: AbortSignal,
    delivery: string,
    environment: 'company' | 'preview'
  ): Promise<PreparedBuild | undefined> {
    const value = object(await request(this.path + '?delivery_id=' + delivery, undefined, signal));
    check(Array.isArray(value.builds));
    const administrator = object(value.administrator);
    check(typeof administrator.name === 'string');
    const match = value.builds
      .map((row) => {
        try {
          return preparedBuild(row);
        } catch {
          return undefined;
        }
      })
      .find((row) => row !== undefined && row.delivery === delivery && row.environment === environment);
    if (match) this.administrator = administrator.name;
    return match;
  }
  private setup(build: PreparedBuild) {
    return this.run(
      async (request, signal) => {
        const record = object(await request(this.path + '/' + build.id + '/setup', {}, signal));
        check(typeof record.id === 'string' && /^a[a-f0-9]{24}$/.test(record.id));
        this.builds = this.builds.map((row) => (row.id === build.id ? {...row, setup: record.id as string} : row));
        if (record.state === 'pending')
          this.dispatchEvent(
            new CustomEvent('application-build-setup-ready', {
              bubbles: true,
              composed: true,
              detail: {setupId: record.id},
            })
          );
        else if (record.state === 'complete') {
          // A completed setup names its application: the parent reconciles
          // that exact record. Older replies without an application stay
          // compatible and resolve as unconfirmed, never a guess.
          let applicationId: string | undefined;
          const application = record.application;
          if (application !== null && typeof application === 'object') {
            const id = (application as {id?: unknown}).id;
            if (typeof id === 'string' && /^a[a-f0-9]{24}$/.test(id)) applicationId = id;
          }
          this.dispatchEvent(
            new CustomEvent('application-setup-complete', {
              bubbles: true,
              composed: true,
              ...(applicationId ? {detail: {applicationId}} : {}),
            })
          );
        } else
          this.notice =
            'This setup was cancelled. Its reserved address and data are retained; inspect its existing operation before preparing another version.';
      },
      {kind: 'setup', environment: build.environment}
    );
  }
  private company(build: PreparedBuild) {
    return this.applications.find(
      (row) =>
        row.operator &&
        row.environment === build.environment &&
        row.project?.repository === this.repository &&
        row.application.name === build.name &&
        row.application.origin === build.origin
    );
  }
  private apply(build: PreparedBuild, company: ApplicationRecord) {
    if (!company.observation?.container || !this.confirmations.has(build.id)) return;
    const applicationId = company.application.id,
      target = company.application.name,
      environment = build.environment;
    return this.run(
      async (request, signal) => {
        try {
          await request(
            `/api/applications/${applicationId}/actions`,
            {
              action: 'replace',
              revision: company.application.revision,
              expected_container: company.observation?.container ?? '',
              confirm_stop: true,
              replacement_build: build.id,
            },
            signal
          );
        } catch (error) {
          // An unconfirmed replace reply may still have completed server-side.
          // Reconcile with a status read of the SAME application — never a
          // second replace POST — and adopt the replacement when the app now
          // runs the prepared image.
          if (signal.aborted || !unknownDeliveryOutcome(error)) throw error;
          const reconciled = await this.readApplyState(request, signal, applicationId).catch(() => undefined);
          // A stored spec reference alone never proves runtime state: the
          // replace can update the spec then fail its observation (leaving
          // the new image with observation=null), and a concurrent Start can
          // leave running=true. Only a confirmed usable observation proves
          // installed+stopped; anything else is reconciled, never asserted.
          if (!reconciled || reconciled.application.image !== build.image) throw error;
          const observation = reconciled.observation;
          if (!observation) {
            this.notice = `The apply reply was unconfirmed, so the same application was re-read: the new version reference matched but runtime is unconfirmed with no observation. Refresh applications and inspect ${target} to reconcile the same apply before acting again.`;
            return;
          }
          if (observation.running) {
            this.notice = `The apply reply was unconfirmed, so the same application was re-read: the new version reference matched but the application is running. Refresh applications and inspect ${target} to reconcile the same apply before acting again.`;
            return;
          }
          // An empty container with running=false is the canonical
          // not-installed observation (see appState in
          // sodaspaces-applications.ts), not a usable stopped runtime: a
          // matching image alone never proves installed/stopped.
          if (!observation.container) {
            this.notice = `The apply reply was unconfirmed, so the same application was re-read: the new version reference matched but the application is not installed. Refresh applications and inspect ${target} to reconcile the same apply before acting again.`;
            return;
          }
          this.confirmations = new Set();
          this.notice = `The apply reply was unconfirmed, so the same application was re-read: the new version is installed and stopped. Start the ${environment === 'preview' ? 'preview' : 'company application'}, then check its retained records.`;
          this.dispatchEvent(new CustomEvent('application-build-applied', {bubbles: true, composed: true}));
          return;
        }
        this.confirmations = new Set();
        this.notice = `The version is installed and stopped. Start the ${environment === 'preview' ? 'preview' : 'company application'}, then check its retained records.`;
        this.dispatchEvent(new CustomEvent('application-build-applied', {bubbles: true, composed: true}));
      },
      {kind: 'apply', environment, target}
    );
  }
  private async readApplyState(
    request: ApplicationRequest,
    signal: AbortSignal,
    applicationId: string
  ): Promise<ApplicationRecord | undefined> {
    const value = applicationsResponse(await request('/api/applications', undefined, signal));
    return value.applications.find((row) => row.application.id === applicationId);
  }
  private buildView(build: PreparedBuild) {
    const company = this.company(build);
    return html`<li>
      <strong>${build.name}</strong> ·
      ${build.environment === 'preview' ? 'Preview · separate data' : 'Company application'} · source
      <code>${build.source.slice(0, 12)}</code>
      <p>Prepared for ${build.origin}. Preview and company accounts and records remain separate.</p>
      ${
        company
          ? company.application.image === build.image
            ? html`<p>This version is already admitted in this environment.</p>`
            : html`<label
                  ><input
                    type="checkbox"
                    .checked=${this.confirmations.has(build.id)}
                    ?disabled=${this.busy || !this.active}
                    @change=${(event: Event) => {
                      if (!(event.target instanceof HTMLInputElement)) return;
                      const next = new Set(this.confirmations);
                      if (event.target.checked) next.add(build.id);
                      else next.delete(build.id);
                      this.confirmations = next;
                    }}
                  />Stop ${build.name} for everyone and apply this version</label
                >
                <button
                  type="button"
                  class="ui primary button"
                  ?disabled=${this.busy || !this.active || !company.observation?.container || !!company.restore || !this.confirmations.has(build.id)}
                  @click=${() => void this.apply(build, company)}
                >
                  Apply version to ${build.name}
                </button>`
          : html`<p>
                Application administrator: ${this.administrator || 'your current Soda account'}. Create the app owner at
                its own address, then explicitly start the installed app.
              </p>
              <button
                type="button"
                class="ui primary button"
                ?disabled=${this.busy || !this.active}
                @click=${() => void this.setup(build)}
              >
                ${build.setup ? 'Continue owner setup' : build.environment === 'preview' ? 'Set up preview' : 'Set up company application'}
              </button>`
      }
    </li>`;
  }
  protected render() {
    return html`<section
      class="soda-application-delivery"
      aria-label="Application delivery"
      aria-busy=${this.busy ? 'true' : 'false'}
    >
      <h3>From verified work to application</h3>
      ${
        buildID(this.requestedDelivery)
          ? html`<p>Delivery <code>${this.requestedDelivery}</code></p>
              ${!this.builds.some((build) => build.environment === 'preview') ? html`<button type="button" class="ui primary button" ?disabled=${this.busy || !this.active} @click=${() => void this.prepare('preview')}>Prepare preview</button>` : ''}
              ${!this.builds.some((build) => build.environment === 'company') ? html`<button type="button" class="ui button" ?disabled=${this.busy || !this.active} @click=${() => void this.prepare('company')}>Prepare company version</button>` : ''}`
          : html`<p>Open a verified application result in Feed to prepare its exact version.</p>`
      }
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy || !this.active}
        @click=${() => void this.refresh()}
      >
        Refresh prepared versions
      </button>
      ${this.notice ? html`<p role="status">${this.notice}</p>` : ''}
      ${
        this.deliveryRefused && !this.busy
          ? html`<button
              type="button"
              class="ui button"
              ?disabled=${!this.active}
              @click=${() => this.openProjectFeed()}
            >
              Follow this delivery in Feed
            </button>`
          : ''
      }
      <ul>
        ${this.builds.map((build) => this.buildView(build))}
      </ul>
    </section>`;
  }
}
if (!customElements.get('soda-application-delivery'))
  customElements.define('soda-application-delivery', SodaApplicationDelivery);
