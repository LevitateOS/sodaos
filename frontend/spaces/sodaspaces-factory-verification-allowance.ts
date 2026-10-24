import {LitElement, html} from 'lit';
import {check, object, readSodaJSON} from './sodaspaces-api.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {factoryCommandId} from './sodaspaces-factory-response.js';
import {terminalID} from './sodaspaces-terminal-response.js';

type VerificationAllowanceContext = {transport: PreparedExtensionMount; repository: string; intent: string};
type VerificationLimits = {runs: number; active_minutes: number};
type VerificationAllowance = {
  intent: string;
  revision: string;
  limits: VerificationLimits;
  consumption: {runs: number; active_seconds: number};
  remaining: {runs: number; active_seconds: number};
  can_increase: boolean;
};
type VerificationAllowanceChange = {command_id: string; expected_revision: string; limits: VerificationLimits};

function verificationAllowance(value: unknown, intent: string): VerificationAllowance {
  const body = object(value),
    limits = object(body.limits),
    used = object(body.consumption),
    left = object(body.remaining);
  check(
    body.intent === intent && typeof body.revision === 'string' && (body.revision === '' || terminalID(body.revision))
  );
  check(typeof body.can_increase === 'boolean');
  for (const amount of [
    limits.runs,
    limits.active_minutes,
    used.runs,
    used.active_seconds,
    left.runs,
    left.active_seconds,
  ])
    check(typeof amount === 'number' && Number.isSafeInteger(amount) && amount >= 0);
  check(Number(limits.runs) > 0 && Number(limits.active_minutes) > 0);
  return {
    intent,
    revision: body.revision,
    limits: {runs: Number(limits.runs), active_minutes: Number(limits.active_minutes)},
    consumption: {runs: Number(used.runs), active_seconds: Number(used.active_seconds)},
    remaining: {runs: Number(left.runs), active_seconds: Number(left.active_seconds)},
    can_increase: body.can_increase,
  };
}

/** An explicit owner limit change never refunds usage or starts verification. */
export class SodaFactoryVerificationAllowance extends LitElement {
  static properties = {
    allowance: {state: true},
    message: {state: true},
    busy: {state: true},
    runs: {state: true},
    minutes: {state: true},
  };
  declare private allowance: VerificationAllowance | undefined;
  declare private message: string;
  declare private busy: boolean;
  declare private runs: number;
  declare private minutes: number;
  private current: VerificationAllowanceContext | undefined;
  private pending: VerificationAllowanceChange | undefined;
  private serial = 0;
  constructor() {
    super();
    this.message = '';
    this.busy = false;
    this.runs = 2;
    this.minutes = 60;
  }
  protected createRenderRoot() {
    return this;
  }
  set context(context: VerificationAllowanceContext | undefined) {
    if (
      context?.transport === this.current?.transport &&
      context?.repository === this.current?.repository &&
      context?.intent === this.current?.intent
    )
      return;
    this.current = context;
    this.serial++;
    this.allowance = undefined;
    this.pending = undefined;
    this.message = '';
    this.busy = false;
    if (context) void this.refresh();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.current = undefined;
    this.serial++;
  }
  private live(serial: number) {
    return serial === this.serial && this.current !== undefined && this.isConnected;
  }
  async refresh() {
    const context = this.current,
      serial = this.serial;
    if (!context || this.busy) return;
    this.busy = true;
    try {
      const response = await context.transport.request(`factory/intents/${context.intent}/verification-allowance`, {
        signal: AbortSignal.timeout(15000),
      });
      check(response.ok);
      const found = verificationAllowance(await readSodaJSON(response), context.intent);
      if (!this.live(serial)) return;
      this.allowance = found;
      this.runs = found.limits.runs;
      this.minutes = found.limits.active_minutes;
      if (this.pending && found.revision === this.pending.command_id) {
        check(
          found.limits.runs === this.pending.limits.runs &&
            found.limits.active_minutes === this.pending.limits.active_minutes
        );
        this.pending = undefined;
      }
      this.message = '';
    } catch {
      if (this.live(serial)) this.message = 'Could not read verification limits. Refresh to try again.';
    } finally {
      if (this.live(serial)) this.busy = false;
    }
  }
  private async save(event: SubmitEvent) {
    event.preventDefault();
    const context = this.current,
      current = this.allowance,
      serial = this.serial;
    if (!context || !current?.can_increase || this.busy) return;
    const limits = {runs: this.runs, active_minutes: this.minutes};
    if (
      !this.pending &&
      (!Number.isSafeInteger(limits.runs) ||
        !Number.isSafeInteger(limits.active_minutes) ||
        limits.runs < current.limits.runs ||
        limits.active_minutes < current.limits.active_minutes ||
        (limits.runs === current.limits.runs && limits.active_minutes === current.limits.active_minutes))
    ) {
      this.message = 'Increase at least one limit; previous usage stays recorded.';
      return;
    }
    const pending = this.pending ?? {command_id: factoryCommandId(), expected_revision: current.revision, limits};
    this.pending = pending;
    this.busy = true;
    try {
      const response = await context.transport.request(`factory/intents/${context.intent}/verification-allowance`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(pending),
        signal: AbortSignal.timeout(15000),
      });
      if (response.status === 409) {
        if (this.live(serial)) {
          this.pending = undefined;
          this.allowance = undefined;
          this.message = 'Verification limits changed. Refresh before retrying.';
        }
        return;
      }
      if (response.status === 401 || response.status === 403) {
        if (this.live(serial)) {
          this.pending = undefined;
          this.allowance = undefined;
          this.message = 'The current account cannot increase these verification limits. Refresh after reconnecting.';
        }
        return;
      }
      if (response.status === 400) {
        if (this.live(serial)) {
          this.pending = undefined;
          this.message = 'The limits were refused. Check the values or refresh.';
        }
        return;
      }
      check(response.ok);
      const saved = verificationAllowance(await readSodaJSON(response), context.intent);
      check(
        saved.revision === pending.command_id &&
          saved.limits.runs === pending.limits.runs &&
          saved.limits.active_minutes === pending.limits.active_minutes
      );
      // A replay acknowledges its original receipt; another owner tab may have
      // already increased the current head. Display that head, never the receipt.
      const latest = await context.transport.request(`factory/intents/${context.intent}/verification-allowance`, {
        signal: AbortSignal.timeout(15000),
      });
      check(latest.ok);
      const confirmed = verificationAllowance(await readSodaJSON(latest), context.intent);
      if (this.live(serial)) {
        this.allowance = confirmed;
        this.pending = undefined;
        this.runs = confirmed.limits.runs;
        this.minutes = confirmed.limits.active_minutes;
        this.message = 'Verification limits saved. Previous usage is retained.';
      }
    } catch {
      if (this.live(serial)) this.message = 'The current limits were not confirmed. Retry the same change or refresh.';
    } finally {
      if (this.live(serial)) this.busy = false;
    }
  }
  protected render() {
    const current = this.allowance,
      disabled = this.busy || this.pending !== undefined;
    return html`<section aria-label="Verification allowance">
      ${current ? html`<p>${current.consumption.runs} of ${current.limits.runs} verification runs used; ${current.consumption.active_seconds} active seconds used of ${current.limits.active_minutes} minutes.</p>` : ''}
      ${
        current?.can_increase
          ? html`<form @submit=${this.save}>
              <label
                >Total verification runs
                <input
                  name="verification-runs"
                  type="number"
                  min=${current.limits.runs}
                  step="1"
                  .value=${String(this.pending?.limits.runs ?? this.runs)}
                  ?disabled=${disabled}
                  @input=${(event: Event) => {
                    this.runs = Number((event.target as HTMLInputElement).value);
                  }}
              /></label>
              <label
                >Total active minutes
                <input
                  name="verification-minutes"
                  type="number"
                  min=${current.limits.active_minutes}
                  step="1"
                  .value=${String(this.pending?.limits.active_minutes ?? this.minutes)}
                  ?disabled=${disabled}
                  @input=${(event: Event) => {
                    this.minutes = Number((event.target as HTMLInputElement).value);
                  }}
              /></label>
              <p>
                Previous usage stays recorded. Each run is still capped at 30 minutes; repository and connection funding
                also apply.
              </p>
              <button type="submit" class="ui button" ?disabled=${this.busy}>
                ${this.pending ? 'Retry same limit change' : 'Increase verification limits'}
              </button>
            </form>`
          : ''
      }
      <p role="status">${this.message}</p>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy}
        @click=${() => {
          void this.refresh();
        }}
      >
        Refresh verification limits
      </button>
    </section>`;
  }
}
customElements.define('soda-factory-verification-allowance', SodaFactoryVerificationAllowance);
