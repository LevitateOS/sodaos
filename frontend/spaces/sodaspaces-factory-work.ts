import {LitElement, html} from 'lit';
import './sodaspaces-factory-acceptance.js';
import './sodaspaces-factory-context-resolution.js';
import './sodaspaces-factory-completion-retry.js';
import './sodaspaces-factory-work-journey.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import {id, projectId, readSodaJSON} from './sodaspaces-api.js';
import {decisionID, intentRevision} from './sodaspaces-intent-response.js';
import type {IntentRevision} from './sodaspaces-intent-response.js';
import {terminalID} from './sodaspaces-terminal-response.js';
import {factoryCommandId, factoryRunStatusResponse, factoryRunText} from './sodaspaces-factory-response.js';
import type {FactoryRunDetail} from './sodaspaces-factory-response.js';
import {statusText} from './sodaspaces-factory-display.js';
import {nativePath} from './sodaspaces-factory-activity.js';
import {
  factoryCommandRecord,
  factoryIssueDetail,
  factoryRepositorySummary,
  pendingRuns,
  pauseReceipt,
  projectStopResult,
  resumeReceipt,
  retryDecision,
  sodaErrorDetail,
  stopReceipt,
  takeoverRecord,
} from './sodaspaces-factory-work-response.js';
import type {
  FactoryIssueDetail,
  FactoryRepositorySummary,
  SodaErrorDetail,
} from './sodaspaces-factory-work-response.js';
import {
  capabilityGate,
  capabilityReasonText,
  describeIntervention,
  describeIssueDetail,
  interventionResultText,
  renderInterventions,
  renderWorkDetail,
} from './sodaspaces-factory-work-view.js';
import type {InterventionKind, InterventionResult} from './sodaspaces-factory-work-view.js';
import {
  describeDirectionRevision,
  describePrerequisites,
  describeRecordedChoice,
  describeRecordedPlan,
  describeRequestOverview,
  describeRevisionHistory,
  describeWhyNext,
  describeWorkOverview,
  overviewScopeFromBound,
  overviewScopeFromMappings,
  retiredOverviewScope,
  renderDirectionRevision,
  renderRecordedPlan,
  renderRequestOverview,
  renderRevisionHistory,
  renderWorkOverview,
  renderWorkPlan,
} from './sodaspaces-factory-progress-view.js';
import type {
  OverviewScope,
  RecordedChoiceModel,
  RecordedPlanModel,
  RequestOverviewModel,
  SourceState,
  WorkOverviewModel,
} from './sodaspaces-factory-progress-view.js';
import {issuePlan, requestOverview, revisionHistory} from './sodaspaces-factory-progress-response.js';
import type {RevisionHistory} from './sodaspaces-factory-progress-response.js';

// Mountable Factory work-progress elements for V01–V07. Each element
// reads only its own contracts and renders its own model; composition,
// routing and watch mounting stay with C. Watching observes only: these
// views never launch, resume or input to a run.

export class WorkReadError extends Error {
  constructor(
    readonly status: number,
    readonly detail: SodaErrorDetail
  ) {
    super(detail.message ?? `Factory read failed (${status}).`);
  }
}

async function getJSON(transport: PreparedExtensionMount, path: string): Promise<unknown> {
  const response = await transport.request(path, {method: 'GET', signal: AbortSignal.timeout(15000)});
  if (!response.ok)
    throw new WorkReadError(response.status, sodaErrorDetail(await readSodaJSON(response).catch(() => undefined)));
  return readSodaJSON(response);
}

interface Posted {
  readonly status: number;
  readonly value: unknown;
}

async function writeJSON(
  transport: PreparedExtensionMount,
  path: string,
  method: 'POST' | 'PUT',
  body: unknown
): Promise<Posted> {
  const response = await transport.request(path, {
    method,
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(90000),
  });
  return {status: response.status, value: await readSodaJSON(response).catch(() => undefined)};
}

async function postJSON(transport: PreparedExtensionMount, path: string, body: unknown): Promise<Posted> {
  return writeJSON(transport, path, 'POST', body);
}

// readFailureText maps one failed read to missing-or-unavailable display
// state. Access loss names itself; anything else reports unavailable.
export function readFailureText(
  status: number,
  detail: SodaErrorDetail,
  missingText: string,
  failedText: string
): {missing: boolean; text: string} {
  if (status === 404) return {missing: true, text: missingText};
  if (status === 401 || status === 403)
    return {missing: false, text: detail.message ?? 'Access changed. Sign in again or ask for access.'};
  return {missing: false, text: detail.message ?? failedText};
}

function failureInput(error: unknown): {status: number; detail: SodaErrorDetail} {
  if (error instanceof WorkReadError) return {status: error.status, detail: error.detail};
  return {status: 0, detail: {}};
}

export interface WorkOverviewContext {
  transport: PreparedExtensionMount;
  repository: string;
  repositoryPath: string;
  intentID?: string;
}

function overviewSignature(context: WorkOverviewContext): string {
  return [context.repository, context.repositoryPath, context.intentID ?? ''].join('|');
}
function sameOverview(a: WorkOverviewContext | undefined, b: WorkOverviewContext | undefined) {
  if (!a || !b || a.transport !== b.transport) return false;
  return overviewSignature(a) === overviewSignature(b);
}

export class SodaFactoryWorkOverview extends LitElement {
  static properties = {model: {state: true}, request: {state: true}, message: {state: true}, busy: {state: true}};
  declare private model: WorkOverviewModel | undefined;
  declare private request: RequestOverviewModel | undefined;
  declare private message: string;
  declare private busy: boolean;
  private current: WorkOverviewContext | undefined;
  private scope: OverviewScope;
  private serial = 0;
  protected createRenderRoot() {
    return this;
  }
  constructor() {
    super();
    this.message = '';
    this.busy = false;
    this.scope = retiredOverviewScope();
  }
  // boundScope is the last published child-view feed: the read keys
  // in use plus the bound work items. Composers derive plan, detail
  // and revision contexts from it; soda-factory-overview-scope
  // notifies every publication, including retirement.
  get boundScope(): OverviewScope {
    return this.scope;
  }
  set context(context: WorkOverviewContext | undefined) {
    if (sameOverview(context, this.current)) return;
    this.current = context;
    this.serial++;
    this.model = undefined;
    this.request = undefined;
    this.message = '';
    this.busy = false;
    this.publishScope(retiredOverviewScope());
    if (context) void this.load();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  refresh() {
    return this.load();
  }
  private publishScope(scope: OverviewScope) {
    this.scope = scope;
    this.dispatchEvent(new CustomEvent('soda-factory-overview-scope', {bubbles: true, composed: true, detail: scope}));
  }
  private async applyRepositoryView(context: WorkOverviewContext, serial: number): Promise<void> {
    const overview = await readOverview(context);
    if (serial !== this.serial) return;
    this.request = undefined;
    this.model = overview.model;
    this.message = '';
    this.publishScope(overview.scope);
  }
  private async load() {
    const context = this.current;
    if (!context || this.busy) return;
    const serial = this.serial;
    this.busy = true;
    try {
      const intentID = context.intentID;
      if (intentID) {
        const request = await readRequestOverview(context, intentID);
        if (serial !== this.serial) return;
        this.request = request;
        this.model = undefined;
        this.message = '';
        this.publishScope(overviewScopeFromMappings(request.mappings, context.repository, context.intentID));
      } else {
        await this.applyRepositoryView(context, serial);
      }
    } catch (error) {
      if (serial === this.serial) {
        this.message = error instanceof Error ? error.message : 'Work overview is unavailable.';
        this.publishScope(overviewScopeFromBound([], context.repository, context.intentID));
      }
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  protected render() {
    return html`<section aria-label="Factory work overview">
      <p data-work-message aria-live="polite">${this.message}</p>
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>
        Refresh overview</button
      >${
        this.request
          ? renderRequestOverview(this.request)
          : this.model
            ? renderWorkOverview(this.model)
            : this.busy
              ? html`<p>Loading work overview…</p>`
              : ''
      }
    </section>`;
  }
}
customElements.define('soda-factory-work-overview', SodaFactoryWorkOverview);

async function readOverviewSummary(
  context: WorkOverviewContext
): Promise<{summary?: FactoryRepositorySummary; state: SourceState}> {
  try {
    const summary = factoryRepositorySummary(
      await getJSON(context.transport, `repositories/${context.repository}/factory`),
      context.repository
    );
    return {summary, state: 'ready'};
  } catch (error) {
    const input = failureInput(error);
    return {state: input.status === 404 ? 'missing' : 'unavailable'};
  }
}

// readRequestOverview reads the authoritative H-R request overview.
// Endpoint and parser failures propagate to the honest unavailable
// boundary; a failed intent read never falls back to composed reads.
async function readRequestOverview(context: WorkOverviewContext, intent: string): Promise<RequestOverviewModel> {
  const view = requestOverview(await getJSON(context.transport, `factory/intents/${intent}/overview`), intent);
  return describeRequestOverview(view, context.repositoryPath);
}

// readOverview reads the repository-only view: queue and setup
// summary without intent scope. It serves contexts without an
// intentID only; intent contexts read the H-R overview above.
async function readOverview(context: WorkOverviewContext): Promise<{model: WorkOverviewModel; scope: OverviewScope}> {
  const {summary, state: summaryState} = await readOverviewSummary(context);
  return {
    model: describeWorkOverview({
      ...(summary === undefined ? {} : {summary}),
      issues: [],
      unreadIssues: [],
      repositoryPath: context.repositoryPath,
      intentState: 'missing' as SourceState,
      generationState: 'missing' as SourceState,
      summaryState,
    }),
    scope: overviewScopeFromBound([], context.repository, context.intentID),
  };
}

export interface WorkPlanContext {
  transport: PreparedExtensionMount;
  repository: string;
  repositoryPath: string;
  issues: string[];
}

function planSignature(context: WorkPlanContext): string {
  return [context.repository, context.repositoryPath, context.issues.join(',')].join('|');
}
function samePlan(a: WorkPlanContext | undefined, b: WorkPlanContext | undefined) {
  if (!a || !b || a.transport !== b.transport) return false;
  return planSignature(a) === planSignature(b);
}

interface RecordedPlanEntry {
  readonly issue: string;
  readonly plan: RecordedPlanModel;
  readonly choice: RecordedChoiceModel;
}

export class SodaFactoryWorkPlan extends LitElement {
  static properties = {
    details: {state: true},
    recorded: {state: true},
    unread: {state: true},
    message: {state: true},
    busy: {state: true},
  };
  declare private details: FactoryIssueDetail[];
  declare private recorded: RecordedPlanEntry[];
  declare private unread: string[];
  declare private message: string;
  declare private busy: boolean;
  private current: WorkPlanContext | undefined;
  private serial = 0;
  protected createRenderRoot() {
    return this;
  }
  constructor() {
    super();
    this.details = [];
    this.recorded = [];
    this.unread = [];
    this.message = '';
    this.busy = false;
  }
  set context(context: WorkPlanContext | undefined) {
    if (samePlan(context, this.current)) return;
    this.current = context;
    this.serial++;
    this.details = [];
    this.recorded = [];
    this.unread = [];
    this.message = '';
    this.busy = false;
    if (context) void this.load();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  refresh() {
    return this.load();
  }
  private async load() {
    const context = this.current;
    if (!context || this.busy) return;
    const serial = this.serial;
    this.busy = true;
    try {
      const details = await readPlanIssues(context);
      const recorded = await readRecordedPlans(context);
      if (serial === this.serial) {
        this.details = details.issues;
        this.recorded = recorded;
        this.unread = details.unread;
        this.message = '';
      }
    } catch (error) {
      if (serial === this.serial) this.message = error instanceof Error ? error.message : 'Work plan is unavailable.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private fallbackDetails(): FactoryIssueDetail[] {
    const recorded = new Set(this.recorded.map((entry) => entry.issue));
    return this.details.filter((detail) => !recorded.has(detail.issue));
  }
  protected render() {
    const context = this.current;
    if (!context) return html``;
    return html`<section aria-label="Factory work plan">
      <p data-work-message aria-live="polite">${this.message}</p>
      ${this.unread.length > 0 ? html`<p>${this.unread.length} work item${this.unread.length === 1 ? '' : 's'} could not be read.</p>` : ''}
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>Refresh plan</button>
      ${this.recorded.map((entry) => renderRecordedPlan(entry.plan, entry.choice))}
      ${renderWorkPlan(
        describeWhyNext(this.details, context.repositoryPath, Math.floor(Date.now() / 1000)),
        describePrerequisites(this.fallbackDetails(), context.repositoryPath)
      )}
    </section>`;
  }
}
customElements.define('soda-factory-work-plan', SodaFactoryWorkPlan);

// readRecordedPlans reads each issue's H-R plan independently. A
// missing route or unreadable plan leaves that issue to the
// readiness-based fallback; it never blocks the other issues.
async function readRecordedPlans(context: WorkPlanContext): Promise<RecordedPlanEntry[]> {
  const settled = await Promise.allSettled(
    context.issues.map((issue) =>
      getJSON(context.transport, `repositories/${context.repository}/factory/issues/${issue}/plan?limit=5`)
    )
  );
  const entries: RecordedPlanEntry[] = [];
  settled.forEach((result, index) => {
    const issue = context.issues[index] ?? '';
    if (result.status !== 'fulfilled' || issue === '') return;
    try {
      const plan = issuePlan(result.value, context.repository, issue);
      entries.push({
        issue,
        plan: describeRecordedPlan(plan.plan, plan.freshness),
        choice: describeRecordedChoice(plan, context.repositoryPath),
      });
    } catch {
      return;
    }
  });
  return entries;
}

async function readPlanIssues(context: WorkPlanContext): Promise<{issues: FactoryIssueDetail[]; unread: string[]}> {
  const issues: FactoryIssueDetail[] = [];
  const unread: string[] = [];
  const settled = await Promise.allSettled(
    context.issues.map((issue) =>
      getJSON(context.transport, `repositories/${context.repository}/factory/issues/${issue}`)
    )
  );
  settled.forEach((result, index) => {
    const issue = context.issues[index] ?? '';
    if (result.status === 'fulfilled' && issue !== '') {
      try {
        issues.push(factoryIssueDetail(result.value, context.repository, issue));
      } catch {
        unread.push(issue);
      }
    } else if (issue !== '') {
      unread.push(issue);
    }
  });
  return {issues, unread};
}

export interface DirectionRevisionContext {
  transport: PreparedExtensionMount;
  intentID: string;
}

export class SodaFactoryDirectionRevision extends LitElement {
  static properties = {
    revision: {state: true},
    state: {state: true},
    history: {state: true},
    historyState: {state: true},
    message: {state: true},
    busy: {state: true},
  };
  declare private revision: IntentRevision | undefined;
  declare private state: SourceState;
  declare private history: RevisionHistory | undefined;
  declare private historyState: SourceState;
  declare private message: string;
  declare private busy: boolean;
  private current: DirectionRevisionContext | undefined;
  private serial = 0;
  protected createRenderRoot() {
    return this;
  }
  constructor() {
    super();
    this.state = 'missing';
    this.historyState = 'missing';
    this.message = '';
    this.busy = false;
  }
  set context(context: DirectionRevisionContext | undefined) {
    if (context?.transport === this.current?.transport && context?.intentID === this.current?.intentID) return;
    this.current = context;
    this.serial++;
    this.revision = undefined;
    this.state = 'missing';
    this.history = undefined;
    this.historyState = 'missing';
    this.message = '';
    this.busy = false;
    if (context) void this.load();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  refresh() {
    return this.load();
  }
  private async load() {
    const context = this.current;
    if (!context || this.busy) return;
    const serial = this.serial;
    this.busy = true;
    try {
      await this.loadReads(context, serial);
    } catch (error) {
      if (serial === this.serial) this.message = error instanceof Error ? error.message : 'Direction is unavailable.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private async loadReads(context: DirectionRevisionContext, serial: number) {
    const revision = await readDirectionRevision(context);
    const history = await readRevisionHistory(context);
    if (serial === this.serial) {
      this.revision = revision.revision;
      this.state = revision.state;
      this.history = history.history;
      this.historyState = history.state;
      this.message = '';
    }
  }
  protected render() {
    return html`<section aria-label="Factory direction revision">
      <p data-work-message aria-live="polite">${this.message}</p>
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>
        Refresh direction</button
      >${renderDirectionRevision(describeDirectionRevision(this.revision, this.state))}
      ${renderRevisionHistory(describeRevisionHistory(this.history, this.historyState))}
    </section>`;
  }
}
customElements.define('soda-factory-direction-revision', SodaFactoryDirectionRevision);

async function readDirectionRevision(
  context: DirectionRevisionContext
): Promise<{revision?: IntentRevision; state: SourceState}> {
  try {
    const revision = intentRevision(await getJSON(context.transport, `factory/intents/${context.intentID}`));
    return {revision, state: 'ready'};
  } catch (error) {
    const input = failureInput(error);
    return {state: input.status === 404 ? 'missing' : 'unavailable'};
  }
}

async function readRevisionHistory(
  context: DirectionRevisionContext
): Promise<{history?: RevisionHistory; state: SourceState}> {
  try {
    const history = revisionHistory(
      await getJSON(context.transport, `factory/intents/${context.intentID}/revisions?limit=2`),
      context.intentID
    );
    return {history, state: 'ready'};
  } catch (error) {
    const input = failureInput(error);
    return {state: input.status === 404 ? 'missing' : 'unavailable'};
  }
}

export interface WorkDetailContext {
  transport: PreparedExtensionMount;
  repository: string;
  repositoryPath: string;
  issue: string;
  runIDs: string[];
  capabilities: Record<InterventionKind, boolean>;
  runID?: string;
  projectID?: string;
  environmentID?: string;
}

interface RunEntry {
  readonly id: string;
  readonly detail: FactoryRunDetail | undefined;
  readonly state: SourceState;
}

type InterventionMarks = Record<InterventionKind, {result: InterventionResult; text: string}>;

function idleMarks(): InterventionMarks {
  const mark = {result: 'idle' as InterventionResult, text: 'Not requested.'};
  return {pause: mark, resume: mark, stop: mark, retry: mark, takeover: mark, project_stop: mark};
}

function detailSignature(context: WorkDetailContext): string {
  const kinds = Object.keys(context.capabilities) as InterventionKind[];
  return [
    context.repository,
    context.repositoryPath,
    context.issue,
    context.runID ?? '',
    context.runIDs.join(','),
    context.projectID ?? '',
    context.environmentID ?? '',
    ...kinds.map((kind) => (context.capabilities[kind] ? '1' : '0')),
  ].join('|');
}
function sameDetail(a: WorkDetailContext | undefined, b: WorkDetailContext | undefined) {
  if (!a || !b || a.transport !== b.transport) return false;
  return detailSignature(a) === detailSignature(b);
}

function runRowText(entry: RunEntry): string {
  if (!entry.detail) return `run ${entry.id.slice(0, 8)}: unavailable.`;
  const row = factoryRunText({
    id: entry.detail.id,
    role: entry.detail.role,
    reconciled: entry.detail.reconciled,
    ...(entry.detail.outcome === undefined ? {} : {outcome: entry.detail.outcome}),
    ...(entry.detail.view?.issue === undefined ? {} : {issue: entry.detail.view.issue}),
    ...(entry.detail.view?.attempt === undefined ? {} : {attempt: entry.detail.view.attempt}),
  });
  const state = entry.detail.state
    ? statusText(
        entry.detail.state.phase,
        entry.detail.state.live,
        entry.detail.state.terminal,
        entry.detail.state.exit_code ?? null
      )
    : 'recorded';
  return `${row} · ${state}`;
}

export class SodaFactoryWorkDetail extends LitElement {
  static properties = {
    detail: {state: true},
    runs: {state: true},
    marks: {state: true},
    message: {state: true},
    busy: {state: true},
    acting: {state: true},
  };
  declare private detail: FactoryIssueDetail | undefined;
  declare private runs: RunEntry[];
  declare private marks: InterventionMarks;
  declare private message: string;
  declare private busy: boolean;
  declare private acting: InterventionKind | undefined;
  private current: WorkDetailContext | undefined;
  private armedStop = false;
  private serial = 0;
  protected createRenderRoot() {
    return this;
  }
  constructor() {
    super();
    this.runs = [];
    this.marks = idleMarks();
    this.message = '';
    this.busy = false;
  }
  set context(context: WorkDetailContext | undefined) {
    if (sameDetail(context, this.current)) return;
    this.current = context;
    this.serial++;
    this.detail = undefined;
    this.runs = [];
    this.marks = idleMarks();
    this.message = '';
    this.busy = false;
    this.acting = undefined;
    this.armedStop = false;
    if (context) void this.load();
  }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.serial++;
    this.current = undefined;
  }
  refresh() {
    return this.load();
  }
  private setMark(kind: InterventionKind, result: InterventionResult, text: string) {
    this.marks = {...this.marks, [kind]: {result, text}};
  }
  private async load() {
    const context = this.current;
    if (!context || this.busy) return;
    const serial = this.serial;
    this.busy = true;
    try {
      const detail = await readDetailIssue(context);
      const runs = await readDetailRuns(context);
      if (serial === this.serial) {
        this.detail = detail.detail;
        this.runs = runs;
        this.message = detail.message;
      }
    } catch (error) {
      if (serial === this.serial) this.message = error instanceof Error ? error.message : 'Work detail is unavailable.';
    } finally {
      if (serial === this.serial) this.busy = false;
    }
  }
  private act = (kind: InterventionKind) => {
    void this.runIntervention(kind);
  };
  private armStop(kind: InterventionKind): boolean {
    if (kind !== 'project_stop' || this.armedStop) return false;
    this.armedStop = true;
    this.setMark(kind, 'idle', 'Click again to confirm stopping the project for everyone.');
    return true;
  }
  private backendGate(kind: InterventionKind): {available: boolean; reasonText: string} {
    if (!this.detail) return {available: true, reasonText: ''};
    return capabilityGate(this.detail.capabilities, kind);
  }
  private async runIntervention(kind: InterventionKind) {
    const context = this.current;
    if (!context || this.acting) return;
    if (!context.capabilities[kind] || this.armStop(kind)) return;
    const gate = this.backendGate(kind);
    if (!gate.available) {
      this.setMark(kind, 'failed', `Unavailable: ${gate.reasonText}`);
      return;
    }
    const serial = this.serial;
    const commandID = factoryCommandId();
    this.acting = kind;
    this.setMark(kind, 'pending', 'Requested; awaiting confirmation.');
    await this.executeIntervention(context, kind, commandID, serial);
    if (serial === this.serial) this.acting = undefined;
  }
  private async executeIntervention(
    context: WorkDetailContext,
    kind: InterventionKind,
    commandID: string,
    serial: number
  ) {
    try {
      const posted = await postIntervention(context, kind, commandID);
      if (serial !== this.serial) return;
      this.applyPosted(kind, context, posted, commandID);
      if (refreshesDetail(kind)) void this.load();
    } catch {
      if (serial !== this.serial) return;
      await this.readBack(context, kind, commandID, serial);
    }
  }
  private applyPosted(kind: InterventionKind, context: WorkDetailContext, posted: Posted, commandID: string) {
    try {
      const outcome = parsePostedOutcome(kind, context, posted, commandID);
      const rendered = interventionResultText(outcome);
      this.setMark(kind, rendered.result, rendered.text);
    } catch {
      this.setMark(kind, 'failed', 'The reply could not be understood; refresh before acting again.');
    }
  }
  private async readBack(context: WorkDetailContext, kind: InterventionKind, commandID: string, serial: number) {
    if (kind === 'project_stop') {
      this.setMark(kind, 'pending', 'No reply was confirmed. Inspect the project before acting again.');
      return;
    }
    try {
      const record = factoryCommandRecord(await getJSON(context.transport, `factory/commands/${commandID}`), commandID);
      if (serial !== this.serial) return;
      const rendered = interventionResultText({kind: 'command', record});
      this.setMark(kind, rendered.result, rendered.text);
    } catch {
      if (serial !== this.serial) return;
      const rendered = interventionResultText({kind: 'unconfirmed', commandID});
      this.setMark(kind, rendered.result, rendered.text);
    }
  }
  private requestWatch(runID: string) {
    this.dispatchEvent(
      new CustomEvent('soda-factory-watch-request', {
        bubbles: true,
        composed: true,
        detail: {run: runID, repository: this.current?.repository, environment: this.current?.environmentID},
      })
    );
  }
  private interventionMarks(kind: InterventionKind) {
    const context = this.current;
    const gate = this.backendGate(kind);
    const mark = this.marks[kind] ?? {result: 'idle' as InterventionResult, text: 'Not requested.'};
    return describeIntervention({
      kind,
      enabled: (context?.capabilities[kind] ?? false) && gate.available && this.acting === undefined,
      result: mark.result,
      resultText: !gate.available && mark.result === 'idle' ? `Unavailable: ${gate.reasonText}` : mark.text,
      target: interventionTarget(context, kind),
    });
  }
  private watchNote(): string {
    const watch = this.detail?.capabilities.watch;
    if (!watch || watch.available) return '';
    return `Watch unavailable: ${capabilityReasonText(watch.reason)}`;
  }
  protected render() {
    const context = this.current;
    if (!context) return html``;
    const watchable = this.detail?.capabilities.watch.available ?? true;
    return html`<section aria-label="Factory work detail">
      <p data-work-message aria-live="polite">${this.message}</p>
      <button type="button" class="ui button" ?disabled=${this.busy} @click=${() => this.load()}>Refresh detail</button>
      ${this.detail ? renderWorkDetail(describeIssueDetail(this.detail, context.repositoryPath)) : this.busy ? html`<p>Loading work detail…</p>` : ''}
      <soda-factory-work-journey
        .context=${{transport: context.transport, repository: context.repository, repositoryPath: context.repositoryPath, issue: context.issue}}
      ></soda-factory-work-journey>
      <soda-factory-requirements-acceptance
        .context=${{transport: context.transport, repository: context.repository, repositoryPath: context.repositoryPath, issue: context.issue}}
        @soda-factory-requirements-accepted=${() => this.load()}
      ></soda-factory-requirements-acceptance>
      <soda-factory-completion-retry
        .context=${{transport: context.transport, repository: context.repository, issue: context.issue}}
        @soda-completion-retry-queued=${() => this.load()}
      ></soda-factory-completion-retry>
      ${this.watchNote() ? html`<p>${this.watchNote()}</p>` : ''}
      <ul>
        ${this.runs.map(
          (entry) => html`<li>
            ${runRowText(entry)}
            ${entry.detail ? html`<soda-factory-context-resolution .context=${{transport: context.transport, run: entry.detail}} @soda-context-resolved=${() => this.load()}></soda-factory-context-resolution>` : ''}
            <button type="button" class="ui button" ?disabled=${!watchable} @click=${() => this.requestWatch(entry.id)}>
              Watch output
            </button>
          </li>`
        )}
      </ul>
      ${renderInterventions(
        (Object.keys(this.marks) as InterventionKind[]).map((kind) => this.interventionMarks(kind)),
        {act: this.act}
      )}
    </section>`;
  }
}
customElements.define('soda-factory-work-detail', SodaFactoryWorkDetail);

function refreshesDetail(kind: InterventionKind): boolean {
  return kind === 'pause' || kind === 'resume' || kind === 'stop' || kind === 'project_stop';
}

function interventionTarget(context: WorkDetailContext | undefined, kind: InterventionKind): string {
  if (!context) return 'this work';
  if (kind === 'pause' || kind === 'resume') return `repository ${context.repository}`;
  if (kind === 'project_stop') return context.projectID ? `project ${context.projectID}` : 'this project';
  return context.runID ? `run ${context.runID.slice(0, 8)}` : 'no run selected';
}

async function readDetailIssue(context: WorkDetailContext): Promise<{detail?: FactoryIssueDetail; message: string}> {
  try {
    const detail = factoryIssueDetail(
      await getJSON(context.transport, `repositories/${context.repository}/factory/issues/${context.issue}`),
      context.repository,
      context.issue
    );
    return {detail, message: ''};
  } catch (error) {
    const input = failureInput(error);
    const failure = readFailureText(input.status, input.detail, 'Work item not found.', 'Work detail is unavailable.');
    return {message: failure.text};
  }
}

async function readDetailRuns(context: WorkDetailContext): Promise<RunEntry[]> {
  const runIDs = [...new Set([...context.runIDs, ...(context.runID ? [context.runID] : [])])];
  const settled = await Promise.allSettled(runIDs.map((runID) => getJSON(context.transport, `factory/runs/${runID}`)));
  return settled.map((result, index) => {
    const runID = runIDs[index] ?? '';
    if (result.status !== 'fulfilled' || runID === '')
      return {id: runID, detail: undefined, state: 'unavailable' as SourceState};
    try {
      return {
        id: runID,
        detail: factoryRunStatusResponse(result.value, runID, context.repository),
        state: 'ready' as SourceState,
      };
    } catch {
      return {id: runID, detail: undefined, state: 'unavailable' as SourceState};
    }
  });
}

async function postIntervention(
  context: WorkDetailContext,
  kind: InterventionKind,
  commandID: string
): Promise<Posted> {
  if (kind === 'pause' || kind === 'resume')
    return postJSON(context.transport, `repositories/${context.repository}/factory/actions`, {
      command_id: commandID,
      action: kind,
    });
  if (kind === 'project_stop' && context.environmentID)
    return postJSON(context.transport, `environments/${context.environmentID}/lifecycle`, {
      action: 'stop',
      confirm_stop: true,
    });
  if ((kind === 'stop' || kind === 'retry' || kind === 'takeover') && context.runID)
    return postJSON(context.transport, `factory/runs/${context.runID}/actions`, {command_id: commandID, action: kind});
  throw Error('Intervention target is not available in this context.');
}

function parseRepositoryReceipt(
  kind: 'pause' | 'resume',
  context: WorkDetailContext,
  posted: Posted,
  commandID: string
) {
  if (kind === 'pause')
    return {kind: 'pause' as const, receipt: pauseReceipt(posted.value, commandID, context.repository)};
  return {kind: 'resume' as const, receipt: resumeReceipt(posted.value, commandID)};
}
function parseRunReceipt(
  kind: 'stop' | 'retry' | 'takeover',
  context: WorkDetailContext,
  posted: Posted,
  commandID: string
) {
  const runID = context.runID ?? '';
  if (kind === 'stop') return {kind: 'stop' as const, receipt: stopReceipt(posted.value, runID)};
  if (kind === 'retry') return {kind: 'retry' as const, receipt: retryDecision(posted.value, commandID, runID)};
  return {kind: 'takeover' as const, receipt: takeoverRecord(posted.value, commandID, runID)};
}
function parsePostedReceipt(kind: InterventionKind, context: WorkDetailContext, posted: Posted, commandID: string) {
  if (kind === 'pause' || kind === 'resume') return parseRepositoryReceipt(kind, context, posted, commandID);
  if (kind === 'project_stop')
    return {kind: 'project_stop' as const, receipt: projectStopResult(posted.value, context.projectID ?? '')};
  return parseRunReceipt(kind, context, posted, commandID);
}

function parsePostedOutcome(kind: InterventionKind, context: WorkDetailContext, posted: Posted, commandID: string) {
  if (posted.status === 200) return parsePostedReceipt(kind, context, posted, commandID);
  if (posted.status === 409) {
    try {
      return {kind: 'pending_runs' as const, pending: pendingRuns(posted.value)};
    } catch {
      return {kind: 'refused' as const, status: posted.status, message: 'Request refused while recording runs.'};
    }
  }
  const detail = sodaErrorDetail(posted.value);
  return {
    kind: 'refused' as const,
    status: posted.status,
    message: detail.message ?? `Request refused (${posted.status}).`,
  };
}

function admitWorkRoot(root: HTMLElement) {
  if (root.ownerDocument !== document) throw Error('Invalid work view root');
}
function admitWorkRepository(repository: string) {
  if (!id(repository)) throw Error('Invalid work repository');
}
function admitWorkIssue(issue: string) {
  if (!id(issue)) throw Error('Invalid work issue');
}
function admitWorkPath(path: string) {
  if (!nativePath(path)) throw Error('Invalid work repository path');
}
function admitWorkIntent(intentID: string | undefined) {
  if (intentID !== undefined && !decisionID(intentID)) throw Error('Invalid work intent');
}
function admitWorkRuns(runIDs: string[], runID: string | undefined) {
  if (runIDs.length > 64 || runIDs.some((run) => !terminalID(run))) throw Error('Invalid work runs');
  if (runID !== undefined && !terminalID(runID)) throw Error('Invalid work run');
}
function admitWorkProject(projectID: string | undefined, environmentID: string | undefined) {
  if (projectID !== undefined && !projectId(projectID)) throw Error('Invalid work project');
  if (environmentID !== undefined && !projectId(environmentID)) throw Error('Invalid work environment');
}

export function mountFactoryWorkOverview(root: HTMLElement, context: WorkOverviewContext) {
  admitWorkRoot(root);
  admitWorkRepository(context.repository);
  admitWorkPath(context.repositoryPath);
  admitWorkIntent(context.intentID);
  const view = new SodaFactoryWorkOverview();
  view.context = context;
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}

export function mountFactoryWorkDetail(root: HTMLElement, context: WorkDetailContext) {
  admitWorkRoot(root);
  admitWorkRepository(context.repository);
  admitWorkPath(context.repositoryPath);
  admitWorkIssue(context.issue);
  admitWorkRuns(context.runIDs, context.runID);
  admitWorkProject(context.projectID, context.environmentID);
  const view = new SodaFactoryWorkDetail();
  view.context = context;
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}

export function mountFactoryWorkPlan(root: HTMLElement, context: WorkPlanContext) {
  admitWorkRoot(root);
  admitWorkRepository(context.repository);
  admitWorkPath(context.repositoryPath);
  if (context.issues.length > 64 || context.issues.some((issue) => !id(issue))) throw Error('Invalid plan issues');
  const view = new SodaFactoryWorkPlan();
  view.context = context;
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}

export function mountFactoryDirectionRevision(root: HTMLElement, context: DirectionRevisionContext) {
  admitWorkRoot(root);
  if (!decisionID(context.intentID)) throw Error('Invalid work intent');
  const view = new SodaFactoryDirectionRevision();
  view.context = context;
  root.append(view);
  return {refresh: () => view.refresh(), dispose: () => view.remove()};
}
