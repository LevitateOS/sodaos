import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {check, id, object, readSodaJSON} from './sodaspaces-api.js';
import {workJourney} from './sodaspaces-factory-work-journey-response.js';
import {describeWorkJourney, renderWorkJourney} from './sodaspaces-factory-work-journey-view.js';
import type {WorkJourneyModel} from './sodaspaces-factory-work-journey-view.js';

interface JourneyContext {
  readonly transport: PreparedExtensionMount;
  readonly repository: string;
  readonly repositoryPath: string;
  readonly issue: string;
}

export class SodaFactoryWorkJourney extends LitElement {
  static properties = {model: {state: true}, message: {state: true}, busy: {state: true}};
  declare private model: WorkJourneyModel | undefined;
  declare private message: string;
  declare private busy: boolean;
  private current: JourneyContext | undefined;
  private serial = 0;

  constructor() {
    super();
    this.message = '';
    this.busy = false;
  }
  protected createRenderRoot() {
    return this;
  }
  set context(context: JourneyContext | undefined) {
    if (context) check(id(context.repository) && id(context.issue));
    const prior = this.current;
    if (
      context?.transport === prior?.transport &&
      context?.repository === prior?.repository &&
      context?.repositoryPath === prior?.repositoryPath &&
      context?.issue === prior?.issue
    )
      return;
    this.current = context;
    this.serial++;
    this.model = undefined;
    this.message = '';
    this.busy = false;
    if (context) void this.load();
  }
  disconnectedCallback() {
    this.current = undefined;
    this.serial++;
    super.disconnectedCallback();
  }
  private async load() {
    const context = this.current,
      serial = this.serial;
    if (!context || this.busy) return;
    this.busy = true;
    this.message = this.model ? 'Refreshing work history…' : 'Loading work history…';
    try {
      const response = await context.transport.request(
        `repositories/${context.repository}/factory/issues/${context.issue}`,
        {method: 'GET', signal: AbortSignal.timeout(15000)}
      );
      check(response.ok);
      const raw = object(await readSodaJSON(response));
      check(raw.repository === context.repository && raw.issue === context.issue);
      const model =
        raw.journey === undefined
          ? undefined
          : describeWorkJourney(workJourney(raw.journey, context.repository, context.issue), context.repositoryPath);
      if (serial !== this.serial) return;
      this.model = model;
      this.message = model ? '' : 'No work journey has been recorded.';
    } catch {
      if (serial === this.serial)
        this.message = this.model
          ? 'Could not refresh. The last loaded work history is still shown.'
          : 'Could not load the work journey. Refresh to try again.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  protected render() {
    if (!this.current) return html``;
    return html`<p role="status">${this.message}</p>
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>
        ${this.busy ? 'Loading…' : 'Refresh journey'}
      </button>
      ${this.model ? renderWorkJourney(this.model) : ''}`;
  }
}
customElements.define('soda-factory-work-journey', SodaFactoryWorkJourney);
