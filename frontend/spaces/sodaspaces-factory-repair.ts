import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {flag, list, num, object, readSodaJSON, record, text} from './sodaspaces-api.js';
import {factoryCommandId} from './sodaspaces-factory-response.js';

type Context = {transport: PreparedExtensionMount; repository: string};
type Limits = {investigator_runs: number; proposal_judgments: number; active_minutes: number};
type Grant = Limits & {id: string; active: boolean};
type Charge = {id: string; kind: string; episode_id: string; cycle: number; active_minutes: number; failed: boolean};
type Cycle = {id: string; number: number; state: string; reason: string; run_id: string};
type Episode = {episode_id: string; reason: string; cycles: Cycle[]};
type View = {revision: number; grant?: Grant; charges: Charge[]; episodes: Episode[]};
type Change = Limits & {command_id: string; expected_revision: number; active: boolean; effect: string};

function limits(value: Record<string, unknown>): Limits {
  return {
    investigator_runs: num(value.investigator_runs),
    proposal_judgments: num(value.proposal_judgments),
    active_minutes: num(value.active_minutes),
  };
}

function parseView(value: unknown): View {
  const data = object(value);
  let grant: Grant | undefined;
  if (data.grant !== null && data.grant !== undefined) {
    const row = record(data.grant);
    grant = {...limits(row), id: text(row.id), active: flag(row.active)};
  }
  const charges = list(data.charges).map((item) => {
    const row = record(item);
    return {
      id: text(row.id),
      kind: text(row.kind),
      episode_id: text(row.episode_id),
      cycle: num(row.cycle),
      active_minutes: num(row.active_minutes),
      failed: flag(row.failed),
    };
  });
  const episodes = list(data.episodes).map((item) => {
    const row = record(item);
    return {
      episode_id: text(row.episode_id),
      reason: text(row.reason),
      cycles: list(row.cycles).map((entry) => {
        const cycle = record(entry);
        return {
          id: text(cycle.id),
          number: num(cycle.number),
          state: text(cycle.state),
          reason: text(cycle.reason),
          run_id: text(cycle.run_id),
        };
      }),
    };
  });
  return {revision: num(data.revision), ...(grant ? {grant} : {}), charges, episodes};
}

class RepairRequestError extends Error {
  constructor(readonly status: number) {
    super(
      status === 403
        ? 'The current repository owner or administrator is required.'
        : status === 409 || status === 412
          ? 'Repair authority changed. Refresh before saving again.'
          : 'Repair funding could not be confirmed.'
    );
  }
}

export class SodaFactoryRepair extends LitElement {
  static properties = {view: {state: true}, busy: {state: true}, message: {state: true}, pending: {state: true}};
  declare private view: View | undefined;
  declare private busy: boolean;
  declare private message: string;
  declare private pending: Change | undefined;
  private current: Context | undefined;
  private serial = 0;

  constructor() {
    super();
    this.busy = false;
    this.message = '';
  }
  protected createRenderRoot() {
    return this;
  }
  set context(context: Context | undefined) {
    if (context?.transport === this.current?.transport && context?.repository === this.current?.repository) return;
    this.current = context;
    this.serial++;
    this.view = undefined;
    this.pending = undefined;
    this.message = '';
    this.busy = false;
    if (context) void this.load();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  private async request(context: Context, change?: Change) {
    const response = await context.transport.request(`repositories/${context.repository}/factory/repair`, {
      method: change ? 'PUT' : 'GET',
      signal: AbortSignal.timeout(15000),
      ...(change ? {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(change)} : {}),
    });
    if (response.status === 401)
      this.dispatchEvent(new CustomEvent('soda-session-invalid', {bubbles: true, composed: true}));
    if (!response.ok) throw new RepairRequestError(response.status);
    return readSodaJSON(response);
  }
  private async load(change?: Change) {
    const context = this.current;
    if (!context || this.busy) return;
    const serial = this.serial;
    this.busy = true;
    try {
      if (change) {
        await this.request(context, change);
        if (serial !== this.serial) return;
        this.pending = undefined;
      }
      const view = parseView(await this.request(context));
      if (serial !== this.serial) return;
      if (view.grant?.id === this.pending?.command_id) this.pending = undefined;
      this.view = view;
      this.message = change
        ? 'Repair grant confirmed. Recorded consumption is preserved.'
        : this.pending
          ? 'A repair change is unconfirmed. Retry the same change before editing.'
          : '';
    } catch (error) {
      if (serial !== this.serial) return;
      if (error instanceof RepairRequestError && [400, 403, 409, 412].includes(error.status)) {
        this.pending = undefined;
        this.view = undefined;
      }
      this.message = this.pending
        ? 'A repair change is unconfirmed. Retry the same change before editing.'
        : error instanceof Error
          ? error.message
          : 'Repair funding could not be confirmed.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private save = (event: SubmitEvent) => {
    event.preventDefault();
    if (!(event.currentTarget instanceof HTMLFormElement) || !this.view || this.busy || this.pending) return;
    const form = event.currentTarget;
    if (!form.reportValidity()) return;
    const data = new FormData(form);
    const effect = String(data.get('effect') ?? '').trim();
    const change: Change = {
      command_id: factoryCommandId(),
      expected_revision: this.view.revision,
      investigator_runs: Number(data.get('investigator_runs')),
      proposal_judgments: Number(data.get('proposal_judgments')),
      active_minutes: Number(data.get('active_minutes')),
      active: data.has('active'),
      effect,
    };
    try {
      limits(change);
    } catch {
      this.message = 'Enter whole numbers within the displayed repair limits.';
      return;
    }
    const prior = this.view.grant;
    if (
      prior &&
      (change.investigator_runs > prior.investigator_runs ||
        change.proposal_judgments > prior.proposal_judgments ||
        change.active_minutes > prior.active_minutes) &&
      !effect
    ) {
      this.message = 'Describe the effect of increasing a repair limit.';
      return;
    }
    this.pending = change;
    void this.load(change);
  };
  protected render() {
    return html`<section aria-label="Repository repair funding">
      <h3>Repository repair funding</h3>
      <p>
        Authorize a finite allowance for Strategist stall investigations and proposal judgments. Existing charges remain
        spent after failures, retries and grant changes. Each episode permits at most two cycles; a second cycle
        requires new factual evidence.
      </p>
      <p data-repair-message aria-live="polite">${this.message}</p>
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>
        Refresh repair funding
      </button>
      ${this.pending ? html`<button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load(this.pending)}>Retry same repair change</button>` : ''}
      ${this.view ? this.renderView(this.view) : ''}
    </section>`;
  }
  private renderView(view: View) {
    const grant = view.grant;
    const runs = view.charges.filter((charge) => charge.kind === 'run').length;
    const judgments = view.charges.filter((charge) => charge.kind === 'judgment').length;
    const minutes = view.charges.reduce((total, charge) => total + charge.active_minutes, 0);
    return html`<p>
        ${grant ? (grant.active ? 'Repair grant active.' : 'Repair grant withdrawn.') : 'No repair grant has been authorized.'}
        Recorded usage: ${runs} investigator runs, ${judgments} proposal judgments, ${minutes} active minutes reserved.
      </p>
      <form @submit=${this.save}>
        <fieldset ?disabled=${this.busy || !!this.pending}>
          <label
            >Investigator run limit
            <input
              type="number"
              name="investigator_runs"
              min="0"
              max="4"
              step="1"
              required
              .value=${String(grant?.investigator_runs ?? 0)}
          /></label>
          <label
            >Proposal judgment limit
            <input
              type="number"
              name="proposal_judgments"
              min="0"
              max="4"
              step="1"
              required
              .value=${String(grant?.proposal_judgments ?? 0)}
          /></label>
          <label
            >Active minute limit
            <input
              type="number"
              name="active_minutes"
              min="0"
              max="120"
              step="1"
              required
              .value=${String(grant?.active_minutes ?? 0)}
          /></label>
          <label
            ><input type="checkbox" name="active" .checked=${grant?.active ?? false} /> Authorize repository repair
            spending</label
          >
          <label>Effect of this change <textarea name="effect" maxlength="4000"></textarea></label>
          <button type="submit" class="ui button">Save repair grant</button>
        </fieldset>
      </form>
      <details>
        <summary>Recorded repair charges (${view.charges.length})</summary>
        <ul>
          ${view.charges.map((charge) => html`<li>${charge.kind}, episode ${charge.episode_id}, cycle ${charge.cycle}: ${charge.active_minutes} active minutes reserved${charge.failed ? ', failed' : ''}. Charge ${charge.id}.</li>`)}
        </ul>
      </details>
      <details>
        <summary>Repair episodes (${view.episodes.length})</summary>
        ${view.episodes.map(
          (episode) =>
            html`<p>Episode ${episode.episode_id}: ${episode.reason}</p>
              <ul>
                ${episode.cycles.map((cycle) => html`<li>Cycle ${cycle.number}: ${cycle.state}${cycle.reason ? ` · ${cycle.reason}` : ''}${cycle.run_id ? ` · Run ${cycle.run_id}` : ''}</li>`)}
              </ul>`
        )}
      </details>`;
  }
}
customElements.define('soda-factory-repair', SodaFactoryRepair);
