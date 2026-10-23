import {LitElement, html} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import {object, check, readSodaJSON} from './sodaspaces-api.js';
import {factoryCommandId, factoryRunStatusResponse} from './sodaspaces-factory-response.js';
import type {FactoryRunDetail} from './sodaspaces-factory-response.js';

type ResolutionContext = {
  transport: PreparedExtensionMount;
  run?: FactoryRunDetail;
  repository?: string;
  issue?: string;
};

export class SodaFactoryContextResolution extends LitElement {
  static properties = {
    message: {state: true},
    busy: {state: true},
    context: {attribute: false},
    selectedRun: {state: true},
  };
  declare context: ResolutionContext | undefined;
  declare private message: string;
  declare private busy: boolean;
  private commandID = '';
  private commandScope = '';
  declare private selectedRun: FactoryRunDetail | undefined;
  private readScope = '';
  constructor() {
    super();
    this.message = '';
    this.busy = false;
  }
  protected createRenderRoot() {
    return this;
  }
  protected updated() {
    const context = this.context;
    const scope = context?.run ? `run:${context.run.id}` : `${context?.repository}/${context?.issue}`;
    if (scope === this.readScope) return;
    this.readScope = scope;
    this.selectedRun = context?.run;
    if (!context?.run && context?.repository && context.issue) void this.readRun(context, scope);
  }
  private async readRun(context: ResolutionContext, scope: string) {
    try {
      const response = await context.transport.request(
        `repositories/${context.repository}/factory/issues/${context.issue}`
      );
      const issue = object(await readSodaJSON(response));
      check(response.ok);
      const lineage = object(object(issue.journey).lineage);
      check(typeof lineage.latest_run === 'string' && /^[a-f0-9]{32}$/.test(lineage.latest_run));
      const runResponse = await context.transport.request(`factory/runs/${lineage.latest_run}`);
      check(runResponse.ok);
      const run = factoryRunStatusResponse(await readSodaJSON(runResponse), lineage.latest_run, context.repository!);
      if (this.readScope === scope) this.selectedRun = run;
    } catch {
      if (this.readScope === scope) this.selectedRun = undefined;
    }
  }
  private verify(
    value: unknown,
    command: string,
    run: string,
    selection: NonNullable<FactoryRunDetail['context_selection']>
  ) {
    const row = object(value);
    check(
      row.command_id === command &&
        row.run_id === run &&
        row.bundle === 'current' &&
        row.scope_digest === selection.scope_digest &&
        row.fingerprint === selection.fingerprint &&
        row.required_digest === selection.required_digest
    );
  }
  private async resolve() {
    const context =
        this.context && this.selectedRun ? {transport: this.context.transport, run: this.selectedRun} : undefined,
      selection = context?.run.context_selection;
    if (!context || !selection?.can_use_current || this.busy) return;
    const scope = [context.run.id, selection.scope_digest, selection.fingerprint, selection.required_digest].join('|');
    if (scope !== this.commandScope) {
      this.commandScope = scope;
      this.commandID = factoryCommandId();
    }
    const command = this.commandID;
    this.busy = true;
    this.message = 'Recording your current-context choice…';
    try {
      const response = await context.transport.request(`factory/runs/${context.run.id}/actions`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          command_id: command,
          action: 'use_current_context',
          context: {
            fingerprint: selection.fingerprint,
            scope_digest: selection.scope_digest,
            required_digest: selection.required_digest,
          },
        }),
      });
      const body = await readSodaJSON(response);
      if (!response.ok) {
        this.message = 'This context choice is no longer available. Refresh the work detail.';
        return;
      }
      this.verify(body, command, context.run.id, selection);
      this.message = 'Current required context selected. Factory will recheck this run before starting.';
      this.dispatchEvent(new CustomEvent('soda-context-resolved', {bubbles: true, composed: true}));
    } catch {
      try {
        const response = await context.transport.request(`factory/commands/${command}`);
        const record = object(await readSodaJSON(response));
        check(response.ok && record.id === command && typeof record.outcome === 'string');
        this.verify(JSON.parse(record.outcome), command, context.run.id, selection);
        this.message = 'Current required context selected. The original receipt was recovered.';
        this.dispatchEvent(new CustomEvent('soda-context-resolved', {bubbles: true, composed: true}));
      } catch {
        this.message = 'No result was confirmed. Retry this same choice or refresh the work detail.';
      }
    } finally {
      this.busy = false;
    }
  }
  protected render() {
    const selection = this.selectedRun?.context_selection;
    if (!selection || selection.action !== 'hold') return html``;
    return html`<section aria-label="Context choice needs attention">
      <p>Factory could not confidently choose which context to foreground.</p>
      <p>
        ${selection.availability === 'complete' ? 'The required current context is complete. You can use it for this admitted run.' : 'Required current context is unavailable. Resolve its missing inputs before continuing.'}
      </p>
      <button
        type="button"
        class="ui button"
        ?disabled=${this.busy || !selection.can_use_current}
        @click=${() => this.resolve()}
      >
        Use current required context
      </button>
      <p role="status">${this.message}</p>
    </section>`;
  }
}
customElements.define('soda-factory-context-resolution', SodaFactoryContextResolution);
