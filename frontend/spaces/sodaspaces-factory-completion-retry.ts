import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {check, id, object, readSodaJSON} from './sodaspaces-api.js';
import {factoryCommandId} from './sodaspaces-factory-response.js';
import {factoryAccess} from './sodaspaces-factory-progress-response.js';
import {factoryCommandRecord, factoryIssueDetail} from './sodaspaces-factory-work-response.js';
import {workJourney} from './sodaspaces-factory-work-journey-response.js';
import type {JourneyGate} from './sodaspaces-factory-work-journey-response.js';

interface RetryContext {
  readonly transport: PreparedExtensionMount;
  readonly repository: string;
  readonly issue: string;
}
interface RetryCommand {
  readonly command: string;
  readonly event: string;
  readonly cycle: string;
  readonly confirmed: boolean;
}

export class SodaFactoryCompletionRetry extends LitElement {
  static properties = {
    gates: {state: true},
    allowed: {state: true},
    busy: {state: true},
    message: {state: true},
    pending: {state: true},
  };
  declare private gates: readonly JourneyGate[];
  declare private allowed: boolean;
  declare private busy: boolean;
  declare private message: string;
  declare private pending: RetryCommand | undefined;
  private current: RetryContext | undefined;
  private serial = 0;

  constructor() {
    super();
    this.gates = [];
    this.allowed = this.busy = false;
    this.message = '';
  }
  protected createRenderRoot() {
    return this;
  }
  set context(context: RetryContext | undefined) {
    if (context) check(id(context.repository) && id(context.issue));
    const prior = this.current;
    if (
      context?.transport === prior?.transport &&
      context?.repository === prior?.repository &&
      context?.issue === prior?.issue
    )
      return;
    this.current = context;
    this.serial++;
    this.gates = [];
    this.allowed = this.busy = false;
    this.message = '';
    this.pending = undefined;
    if (context) void this.load(context, this.serial);
  }
  disconnectedCallback() {
    this.current = undefined;
    this.serial++;
    super.disconnectedCallback();
  }
  private async read(context: RetryContext, path: string) {
    const response = await context.transport.request(path, {method: 'GET', signal: AbortSignal.timeout(15000)});
    check(response.ok);
    return readSodaJSON(response);
  }
  private async load(context: RetryContext, serial: number) {
    try {
      const [accessValue, issueValue] = await Promise.all([
        this.read(context, `repositories/${context.repository}/factory/access`),
        this.read(context, `repositories/${context.repository}/factory/issues/${context.issue}`),
      ]);
      const access = factoryAccess(accessValue, context.repository);
      factoryIssueDetail(issueValue, context.repository, context.issue);
      const journey = object(issueValue).journey;
      const gates = journey === undefined ? [] : workJourney(journey, context.repository, context.issue).gates;
      check(gates.every((gate) => gate.repository === context.repository));
      if (serial !== this.serial) return;
      this.allowed = access.viewer.code_write && access.viewer.session_current;
      this.gates = gates.filter((gate) => gate.state === 'attention' && gate.retryable);
      if (this.message.startsWith('Could not load completion reviews.')) this.message = '';
    } catch {
      if (serial === this.serial) {
        this.allowed = false;
        this.message = 'Could not load completion reviews. Refresh the work detail.';
      }
    }
  }
  private async refresh() {
    const context = this.current,
      serial = this.serial;
    if (!context || this.busy) return;
    this.busy = true;
    await this.load(context, serial);
    if (serial === this.serial) this.busy = false;
  }
  private sameGate(command: RetryCommand, gate: JourneyGate) {
    return command.event === gate.completion_id && command.cycle === gate.cycle_id;
  }
  private verify(value: unknown, context: RetryContext, command: RetryCommand) {
    const row = object(value);
    check(
      row.command_id === command.command &&
        row.event_id === command.event &&
        row.prior_cycle === command.cycle &&
        row.successor_cycle === command.command &&
        row.repository === context.repository &&
        row.queued === true
    );
  }
  private confirmed(context: RetryContext, command: RetryCommand, recovered: boolean) {
    this.pending = {...command, confirmed: true};
    this.message = recovered
      ? 'Completion review retry queued. The original command receipt was recovered.'
      : 'Completion review retry queued. Factory will reassess the recorded result in a new bounded attempt.';
    this.dispatchEvent(new CustomEvent('soda-completion-retry-queued', {bubbles: true, composed: true}));
    void this.load(context, this.serial);
  }
  private async recover(context: RetryContext, command: RetryCommand) {
    const value = await this.read(context, `factory/commands/${command.command}`);
    const receipt = factoryCommandRecord(value, command.command);
    check(
      receipt.type === 'retry-completion' &&
        receipt.target === `repository/${context.repository}/completion/${command.event}` &&
        receipt.outcome !== undefined
    );
    this.verify(JSON.parse(receipt.outcome), context, command);
  }
  private async retry(gate: JourneyGate) {
    const context = this.current,
      serial = this.serial;
    if (!context || !this.allowed || this.busy || !this.gates.includes(gate)) return;
    if (this.pending && (!this.sameGate(this.pending, gate) || this.pending.confirmed)) return;
    const command = this.pending ?? {
      command: factoryCommandId(),
      event: gate.completion_id,
      cycle: gate.cycle_id,
      confirmed: false,
    };
    this.pending = command;
    this.busy = true;
    this.message = 'Requesting a new completion review…';
    try {
      const response = await context.transport.request(`repositories/${context.repository}/factory/actions`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        signal: AbortSignal.timeout(90000),
        body: JSON.stringify({
          command_id: command.command,
          action: 'retry_completion',
          completion: {event_id: command.event, expected_prior_cycle: command.cycle},
        }),
      });
      const value = await readSodaJSON(response);
      check(response.ok);
      this.verify(value, context, command);
      if (serial === this.serial) this.confirmed(context, command, false);
    } catch {
      try {
        await this.recover(context, command);
        if (serial === this.serial) this.confirmed(context, command, true);
      } catch {
        if (serial === this.serial)
          this.message =
            'No retry was confirmed. Retry this same request or refresh the work detail; the recorded decision is unchanged.';
      }
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  protected render() {
    if (!this.gates.length && !this.message) return html``;
    return html`<section aria-label="Completion review needs attention">
      ${this.gates.map(
        (gate) => html`<p>
            Completion review needs attention. A retry requests another bounded assessment of the recorded result.
          </p>
          <button
            type="button"
            class="ui button"
            ?disabled=${!this.allowed || this.busy || (!!this.pending && (!this.sameGate(this.pending, gate) || this.pending.confirmed))}
            @click=${() => this.retry(gate)}
          >
            Retry completion review
          </button>`
      )}
      <p role="status">${this.message}</p>
      ${
        this.message.startsWith('Could not load completion reviews.')
          ? html`<button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.refresh()}>
              Refresh completion reviews
            </button>`
          : ''
      }
    </section>`;
  }
}
customElements.define('soda-factory-completion-retry', SodaFactoryCompletionRetry);
