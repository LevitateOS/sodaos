import {LitElement, html} from 'lit';
import type {PropertyValues} from 'lit';
import {check, object, SodaRequestError} from './sodaspaces-api.js';
import type {ApplicationRequest} from './sodaspaces-applications.js';

// Typed API refusals the setup endpoints already return (see
// internal/web/api/application_setup.go and operator.go). A 403
// operator_required is a permission refusal, never a session change; 404
// means the retained setup is gone, never a state claim; 409 names a
// changed generation. Service and host failures (5xx), a lost reply and a
// mismatched reply all leave the outcome unconfirmed and prove nothing.
function setupErrorStatus(error: unknown): number | undefined {
  if (error instanceof SodaRequestError) return error.status;
  if (typeof error === 'object' && error !== null) {
    const status = (error as {status?: unknown}).status;
    if (typeof status === 'number' && Number.isSafeInteger(status)) return status;
  }
  return undefined;
}

// The exact failure reason for an inspect that never confirmed, naming only
// fixed server contract states. Untrusted reply text is never shown.
function setupInspectFailure(status?: number): string {
  if (status === 401) return 'Your Soda session changed. Reconnect, then inspect the same setup before continuing.';
  if (status === 403) return 'Only the configured Soda operator can inspect this setup.';
  if (status === 404) return 'This owner setup was not found. Refresh owner setups before continuing.';
  if (status === 409)
    return 'This setup changed. Refresh owner setups and inspect its current generation before continuing.';
  return 'This setup could not be inspected. Refresh owner setups and inspect the same setup before continuing.';
}

interface Setup {
  id: string;
  application: {id: string; name: string; origin: string};
  administrator_id: number;
  expires_at: number;
  state: 'pending' | 'complete' | 'aborted';
  observation: {container: string; running: boolean; ready: boolean} | null;
}
const appID = (value: unknown): value is string => typeof value === 'string' && /^a[0-9a-f]{24}$/.test(value);
function setup(value: unknown): Setup {
  const v = object(value),
    a = object(v.application);
  check(
    Object.keys(v).length === 6 &&
      appID(v.id) &&
      appID(a.id) &&
      typeof a.name === 'string' &&
      typeof a.origin === 'string'
  );
  const address = new URL(a.origin);
  check(address.protocol === 'https:' && address.origin === a.origin && !address.username && !address.password);
  check(typeof v.administrator_id === 'number' && Number.isSafeInteger(v.administrator_id) && v.administrator_id > 0);
  check(typeof v.expires_at === 'number' && Number.isSafeInteger(v.expires_at) && v.expires_at > 0);
  check(v.state === 'pending' || v.state === 'complete' || v.state === 'aborted');
  let observation: Setup['observation'] = null;
  if (v.observation !== null) {
    const o = object(v.observation);
    check(
      Object.keys(o).length === 3 &&
        typeof o.container === 'string' &&
        (/^[a-f0-9]{64}$/.test(o.container) || (o.container === '' && o.running === false && o.ready === false)) &&
        typeof o.running === 'boolean' &&
        typeof o.ready === 'boolean' &&
        (!o.ready || o.running)
    );
    observation = {container: o.container, running: o.running, ready: o.ready};
  }
  return {
    id: v.id,
    application: {id: a.id, name: a.name, origin: a.origin},
    administrator_id: v.administrator_id,
    expires_at: v.expires_at,
    state: v.state,
    observation,
  };
}
const identity = () => 'a' + crypto.randomUUID().replaceAll('-', '').slice(0, 24);

// Operator-only provisional installation. The fragment link exists only in
// this mounted component and is discarded when its session/lifetime changes.
export class SodaApplicationSetup extends LitElement {
  static properties = {
    request: {attribute: false},
    active: {type: Boolean},
    sessionKey: {attribute: false},
    records: {state: true},
    busy: {state: true},
    notice: {state: true},
    link: {state: true},
    expanded: {state: true},
  };
  declare request: ApplicationRequest | undefined;
  declare active: boolean;
  declare sessionKey: string;
  declare private records: Setup[];
  declare private busy: boolean;
  declare private notice: string;
  declare private link: {id: string; url: string} | null;
  declare private expanded: boolean;
  constructor() {
    super();
    this.active = false;
    this.sessionKey = '';
    this.records = [];
    this.busy = false;
    this.notice = '';
    this.link = null;
    this.expanded = false;
  }
  private controller: AbortController | undefined;
  private epoch = 0;
  createRenderRoot() {
    return this;
  }
  protected updated(changes: PropertyValues) {
    if (changes.has('sessionKey') || (changes.has('active') && !this.active)) this.clear();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.clear();
  }
  private clear() {
    this.epoch++;
    this.controller?.abort();
    this.controller = undefined;
    this.link = null;
    this.records = [];
    this.busy = false;
  }
  // One guarded setup read or action. True only when the work applied
  // against the current epoch; a failure or a superseded run reports false
  // and never applies. The failure status rides along so callers that must
  // not fall through (beginPreparedSetup) can name the exact reason instead
  // of the generic confirmation notice below.
  private async run(
    work: (request: ApplicationRequest, signal: AbortSignal) => Promise<void>
  ): Promise<{applied: boolean; status?: number | undefined}> {
    if (this.busy || !this.active || !this.request) return {applied: false};
    const epoch = this.epoch,
      controller = new AbortController();
    this.controller = controller;
    this.busy = true;
    this.notice = '';
    const request = this.request;
    try {
      await work(async (...args) => {
        const result = await request(...args);
        if (epoch !== this.epoch || controller.signal.aborted) throw Error('inactive');
        return result;
      }, controller.signal);
    } catch (error) {
      if (epoch === this.epoch) {
        this.notice = 'Setup could not be confirmed. Refresh and inspect the same setup before continuing.';
        return {applied: false, status: setupErrorStatus(error)};
      }
      return {applied: false};
    } finally {
      if (epoch === this.epoch) {
        this.busy = false;
        this.controller = undefined;
      }
    }
    if (epoch !== this.epoch || !this.active) return {applied: false};
    return {applied: true};
  }
  private refresh() {
    this.link = null;
    // A fresh inventory read re-derives every row from current bytes.
    return this.run(async (request, signal) => {
      const v = object(await request('/api/applications/setups', undefined, signal));
      check(Array.isArray(v.setups));
      this.records = v.setups.map(setup);
    });
  }
  // Merges the reply record into the pending inventory, dropping settled
  // rows. A settled reply (complete/aborted) replaces any pending row and
  // reports its exact outcome; a reply naming another setup proves nothing
  // and must not merge.
  private apply(value: unknown, expectedId?: string): void {
    const v = object(value),
      record = setup(v.setup);
    // The reply must concern the setup acted on: a well-formed reply naming
    // another setup proves nothing about ours and must not merge foreign
    // state into this inventory.
    check(!expectedId || record.id === expectedId);
    this.records = [...this.records.filter((row) => row.id !== record.id), record].filter(
      (row) => row.state === 'pending'
    );
    this.link = null;
    if (v.result !== undefined) {
      const result = object(v.result);
      check(result.operation === record.id);
      if (result.setup_url !== undefined) {
        check(typeof result.setup_url === 'string');
        const url = new URL(result.setup_url);
        check(
          url.origin === record.application.origin &&
            url.pathname === '/setup' &&
            !url.search &&
            !url.username &&
            !url.password
        );
        const fragment = new URLSearchParams(url.hash.slice(1));
        check(
          fragment.get('operation') === record.id &&
            /^[a-f0-9]{64}$/.test(fragment.get('token') ?? '') &&
            [...fragment.keys()].length === 2
        );
        this.link = {id: record.id, url: result.setup_url};
      }
    }
    if (record.state === 'complete') {
      // Setup completion stays neutral: the retained setup observation is a
      // stopped provisional generation, never current runtime. The exact-app
      // parent inventory supplies all runtime state (running/stopped/unconfirmed).
      this.notice =
        'Owner confirmed. Setup complete. Refresh applications to reconcile the completed setup before starting it.';
      this.dispatchEvent(
        new CustomEvent('application-setup-complete', {
          bubbles: true,
          composed: true,
          detail: {applicationId: record.application.id},
        })
      );
    } else if (record.state === 'aborted') this.notice = 'Setup stopped. Its data and reserved address are retained.';
  }
  private act(record: Setup, action: string) {
    this.link = null;
    return this.run(async (request, signal) => {
      this.apply(
        await request(
          `/api/applications/setups/${record.id}/actions`,
          {
            action,
            expected_container: record.observation?.container ?? '',
            confirm: action === 'complete' || action === 'abort',
          },
          signal
        ),
        record.id
      );
    });
  }
  // Renewal is not offered: every rotation-adjacent renew is refused while
  // the interim fail-closed gate holds (no inventory record carries renew
  // support), so no expired row offers a renew action and no renew POST is
  // ever sent. Inspect and refresh stay available throughout. When the API
  // serves a real capability signal for a generation, its offer returns for
  // exactly that scope — no remembered refusal stands in its way.
  // Called only after an explicit prepared-build setup action. It reuses the
  // same provisional generation and ephemeral credential handling as setup UI.
  async beginPreparedSetup(id: string) {
    if (!appID(id) || !this.active || this.busy) return;
    this.expanded = true;
    const epoch = this.epoch;
    const refreshed = await this.refresh();
    // A failed or superseded inventory read proves nothing about the retained
    // setup: never retire a live setup on a stale read. The failed read keeps
    // its own confirmation notice; a superseded run stays silent for its owner.
    if (!refreshed.applied || epoch !== this.epoch || !this.active) return;
    const record = this.records.find((row) => row.id === id);
    if (!record) {
      // The retained setup id outlived its pending state (finished or
      // cancelled elsewhere): continuing it is unsupported, and no fresh
      // setup at the same address can be invented while its reservation is
      // retained. Name the retained outcome and reconcile against the current
      // inventory instead of leaving a silent dead button.
      this.notice =
        'This owner setup is no longer pending, so it cannot be continued. Its reserved address and data are retained; refresh applications to reconcile the current inventory before acting again.';
      return;
    }
    if (Date.now() >= record.expires_at * 1000) {
      await this.reconcileExpiredSetup(record);
      return;
    }
    await this.act(record, 'begin');
  }
  // An expired pending setup cannot mint a fresh owner link: the host refuses
  // begin past expires_at ("setup token expired") and the retained
  // reservation still blocks a fresh setup at the same address
  // ("setup_conflict"). Re-inspecting the same retained generation is the only
  // supported read; finish and cancel stay supported only on what the inspect
  // confirms. No restart path exists on the retained address, and none is
  // implied: the inspect outcome is reported truthfully, and expiry ends the
  // setup with its retained outcome unless its owner was already created.
  private async reconcileExpiredSetup(record: Setup) {
    const epoch = this.epoch;
    const inspected = await this.act(record, 'inspect');
    if (epoch !== this.epoch || !this.active) return;
    if (!inspected.applied) {
      this.notice = setupInspectFailure(inspected.status);
      return;
    }
    const current = this.records.find((row) => row.id === record.id);
    // apply() already reported a settled outcome (complete/aborted) by dropping
    // the record; keep that exact outcome instead of expiry copy.
    if (!current) return;
    // A null observation proves nothing: the inspect confirmed no state, so
    // absence cannot be claimed. Report unconfirmed and offer a re-inspect or
    // an operator check before stopping and cancelling; no renewal is
    // promised while the API refuses it.
    if (!current.observation) {
      this.notice =
        'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected but its owner state is unconfirmed, so nothing can be confirmed about it yet. Inspect the setup again or ask the Soda operator to check it before stopping and cancelling it. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state.';
      return;
    }
    // Only an empty-container stopped observation confirms absence (the parser
    // admits '' only with running=false and ready=false): nothing is running,
    // so cancel names the only path. No renewal is promised while the API
    // refuses it.
    if (!current.observation.container && !current.observation.running && !current.observation.ready) {
      this.notice =
        'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected and no owner setup is running. Stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Refresh owner setups to confirm the inventory after cancelling.';
      return;
    }
    this.notice =
      'This setup link has expired, so its owner link cannot be re-opened as-is. The same setup was inspected and its owner setup is ' +
      (current.observation.running ? 'running' : 'stopped') +
      '. If its owner was already created, confirm the owner and finish; otherwise stop and cancel this setup. Its owner credential, data, and reserved address stay retained.';
  }
  private create(event: SubmitEvent) {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement)) return;
    const data = new FormData(event.currentTarget),
      administrator = String(data.get('administrator') ?? '');
    let application: Record<string, unknown>;
    try {
      application = object(JSON.parse(String(data.get('application') ?? '')));
      check(!('owner_team' in application));
    } catch {
      this.notice = 'Enter the admitted application references without an owner team.';
      return;
    }
    application = {...application, id: identity(), data: identity()};
    this.link = null;
    return this.run(async (request, signal) => {
      const record = setup(
        await request('/api/applications/setups', {application, administrator_id: administrator}, signal)
      );
      this.records = [...this.records, record];
      this.apply(
        await request(
          `/api/applications/setups/${record.id}/actions`,
          {action: 'begin', expected_container: '', confirm: false},
          signal
        ),
        record.id
      );
    });
  }
  protected render() {
    return html`<details
      ?open=${this.expanded}
      @toggle=${(event: Event) => {
        this.expanded = (event.currentTarget as HTMLDetailsElement).open;
        if (this.expanded && !this.records.length && !this.busy) void this.refresh();
      }}
    >
      <summary>Set up a new application (operator)</summary>
      <p>
        Create the application's first owner at its own address. SodaOS stores its administrator and confirmed team; the
        owner's password stays with the application.
      </p>
      <form @submit=${(event: SubmitEvent) => void this.create(event)}>
        <label
          >Admitted application references<textarea
            name="application"
            required
            rows="5"
            placeholder='{"name":"MyApp","artifact_sha256":"…","bundle_sha256":"…","tools_sha256":"…","runtime_image":"sha256:…","image":"sha256:…","origin":"https://app.example.com","port":24000}'
          ></textarea>
        </label>
        <label
          >Soda administrator user ID<input name="administrator" inputmode="numeric" pattern="[1-9][0-9]*" required
        /></label>
        <button type="submit" class="ui button" ?disabled=${this.busy || !this.active}>Begin owner setup</button>
      </form>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy || !this.active}
        @click=${() => void this.refresh()}
      >
        Refresh owner setups
      </button>
      ${this.notice ? html`<p role="status">${this.notice}</p>` : ''}
      <ul>
        ${this.records.map(
          (record) => html`<li>
            <strong>${record.application.name}</strong> · ${record.application.origin}
            <p>
              ${record.observation ? (record.observation.running ? 'Owner setup is running.' : 'Owner setup is stopped.') : 'Setup state is unknown.'}
              Setup link expires ${new Date(record.expires_at * 1000).toLocaleString()}.
            </p>
            ${Date.now() >= record.expires_at * 1000 ? html`<p>The link has expired. Inspect this setup, then finish if its owner was already created; if no owner was created, stop and cancel this setup. Its owner credential, data, and reserved address stay retained. Inspect or refresh this same setup to confirm its state; renewal is unavailable until the owner claim can be confirmed.</p>` : ''}
            ${this.link?.id === record.id ? html`<p><a class="ui primary button" href=${this.link.url} target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer">Create application owner ↗</a></p>` : ''}
            <button
              type="button"
              class="ui button"
              ?disabled=${this.busy || !this.active || Date.now() >= record.expires_at * 1000}
              @click=${() => void this.act(record, 'begin')}
            >
              Open owner setup ${record.application.name}
            </button>
            <button
              type="button"
              class="ui button"
              ?disabled=${this.busy || !this.active}
              @click=${() => void this.act(record, 'inspect')}
            >
              Inspect owner setup ${record.application.name}
            </button>
            <button
              type="button"
              class="ui button"
              ?disabled=${this.busy || !this.active || !record.observation?.container}
              @click=${() => void this.act(record, 'complete')}
            >
              Confirm owner and finish ${record.application.name}
            </button>
            <button
              type="button"
              class="ui button"
              ?disabled=${this.busy || !this.active}
              @click=${() => void this.act(record, 'abort')}
            >
              Stop and cancel setup ${record.application.name}
            </button>
          </li>`
        )}
      </ul>
    </details>`;
  }
}
if (!customElements.get('soda-application-setup'))
  customElements.define('soda-application-setup', SodaApplicationSetup);
