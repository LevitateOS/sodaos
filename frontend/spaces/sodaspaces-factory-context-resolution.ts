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

function runSelectionKey(run: FactoryRunDetail): string {
  const selection = run.context_selection;
  return [
    run.id,
    selection?.action ?? '',
    selection?.availability ?? '',
    selection?.can_use_current === true ? '1' : '0',
    selection?.fingerprint ?? '',
    selection?.scope_digest ?? '',
    selection?.required_digest ?? '',
  ].join('|');
}

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
  private readTransport: PreparedExtensionMount | undefined;
  private readToken = 0;
  private uiScope = '';
  private uiTransport: PreparedExtensionMount | undefined;
  private lastRunKey = '';
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
    // A direct run input always wins: consume refreshed runs even when the
    // run ID is unchanged, and retire any in-flight issue/run lookup so a
    // stale read can never overwrite the newer input.
    if (context?.run) {
      this.readToken++;
      this.readScope = `run:${context.run.id}`;
      this.readTransport = context.transport;
      const key = runSelectionKey(context.run);
      // Scoped command UI (FJ-447): a new run/transport/selection never
      // inherits the prior scope's confirmation or busy. The in-flight
      // server op is preserved; its rendering is ignored by resolve's guard.
      const uiScope = `run:${context.run.id}|${key}`;
      if (uiScope !== this.uiScope || context.transport !== this.uiTransport) {
        this.uiScope = uiScope;
        this.uiTransport = context.transport;
        this.message = '';
        this.busy = false;
      }
      if (key !== this.lastRunKey || this.selectedRun !== context.run) {
        this.lastRunKey = key;
        this.selectedRun = context.run;
      }
      return;
    }
    const scope = `${context?.repository}/${context?.issue}`;
    if (scope === this.readScope && context?.transport === this.readTransport) return;
    this.readScope = scope;
    this.readTransport = context?.transport;
    // Same scoped reset for issue entry: opening another issue never shows
    // the prior issue's confirmation and is never blocked by its busy.
    this.uiScope = scope;
    this.uiTransport = context?.transport;
    this.message = '';
    this.busy = false;
    this.lastRunKey = '';
    const token = ++this.readToken;
    this.selectedRun = undefined;
    if (context && context.repository && context.issue) void this.readRun(context, scope, token);
  }
  private async readRun(context: ResolutionContext, scope: string, token: number) {
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
      if (token === this.readToken && this.readScope === scope) {
        // Same-issue unbind (FJ-450): the loaded run/selection changed
        // under identical issue/transport while an older Use-current POST
        // may still be pending. Drop only the command UI binding
        // (busy/message); the original server op is preserved and its
        // stale reply stays ignored by resolve's liveness guard.
        if (runSelectionKey(run) !== this.lastRunKey) {
          this.lastRunKey = runSelectionKey(run);
          this.message = '';
          this.busy = false;
        }
        this.selectedRun = run;
      }
    } catch {
      if (token === this.readToken && this.readScope === scope) {
        // A failed re-read must not inherit the superseded action's busy.
        if (this.selectedRun !== undefined) {
          this.lastRunKey = '';
          this.message = '';
          this.busy = false;
        }
        this.selectedRun = undefined;
      }
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
  refresh() {
    const context = this.context;
    if (!context || context.run || !context.repository || !context.issue) return;
    // Explicit same-scope re-read (FJ-446): the parent re-invokes the child
    // on genuine Work refresh/reopen, which the scope+transport skip would
    // otherwise ignore forever. A fresh token supersedes in-flight reads;
    // their landings never write (see readRun guards). No mutation.
    const scope = `${context.repository}/${context.issue}`;
    const token = ++this.readToken;
    this.readScope = scope;
    this.readTransport = context.transport;
    void this.readRun(context, scope, token);
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
    // Scoped completion guard (FJ-447): only the scope that issued this
    // choice may render its outcome. A delayed reply landing after the user
    // opened another issue/run/selection is ignored — the original server
    // command is preserved, never retried, just not rendered or emitted.
    // A dedicated capture (not readToken, which every direct-run render
    // bumps) keeps this resolve's own busy/message writes live.
    const transport = context.transport;
    const runId = context.run.id;
    const selectionKey = runSelectionKey(context.run);
    const live = () =>
      this.context?.transport === transport &&
      this.selectedRun?.id === runId &&
      this.selectedRun !== undefined &&
      runSelectionKey(this.selectedRun) === selectionKey;
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
        if (!live()) return;
        this.message = 'This context choice is no longer available. Refresh the work detail.';
        return;
      }
      this.verify(body, command, context.run.id, selection);
      if (!live()) return;
      this.message = 'Current required context selected. Factory will recheck this run before starting.';
      this.dispatchEvent(new CustomEvent('soda-context-resolved', {bubbles: true, composed: true}));
    } catch {
      try {
        const response = await context.transport.request(`factory/commands/${command}`);
        const record = object(await readSodaJSON(response));
        check(response.ok && record.id === command && typeof record.outcome === 'string');
        this.verify(JSON.parse(record.outcome), command, context.run.id, selection);
        if (!live()) return;
        this.message = 'Current required context selected. The original receipt was recovered.';
        this.dispatchEvent(new CustomEvent('soda-context-resolved', {bubbles: true, composed: true}));
      } catch {
        if (!live()) return;
        this.message = 'No result was confirmed. Retry this same choice or refresh the work detail.';
      }
    } finally {
      if (live()) this.busy = false;
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
