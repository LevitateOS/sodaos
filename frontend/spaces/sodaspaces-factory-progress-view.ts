import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {check, object} from './sodaspaces-api.js';
import {decisionID} from './sodaspaces-intent-response.js';
import type {IntentRevision} from './sodaspaces-intent-response.js';
import {activityTimestamp} from './sodaspaces-factory-activity.js';
import type {
  FactoryIssueDetail,
  FactoryRepositorySummary,
  WorkCapabilities,
} from './sodaspaces-factory-work-response.js';
import {
  describeIssueDetail,
  nativeIssueLink,
  nativePullLink,
  upfrontCapabilities,
} from './sodaspaces-factory-work-view.js';
import type {DeliveryState} from './sodaspaces-factory-work-view.js';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {DirectionRevisionContext, WorkDetailContext, WorkPlanContext} from './sodaspaces-factory-work.js';
import type {
  AccessPlanningEntry,
  FactoryAccess,
  IssuePlan,
  LifecycleEffectAction,
  LifecycleEffectConfig,
  PlanningAllowanceView,
  PriorityComparison,
  RecordedPlan,
  RequestAttention,
  RequestOverview,
  RevisionEntry,
  RevisionGeneration,
  RevisionHistory,
  RevisionIdeaChange,
  RevisionTaskChange,
} from './sodaspaces-factory-progress-response.js';
import type {PlanFreshness} from './sodaspaces-factory-progress-response.js';

// Pure accepted-request overview, planning/why-next, direction-revision
// and optional-effect models for V01/V02/V05/V07. Each model composes
// only its own recorded reads; missing, waiting and unavailable states
// stay distinct and no selection, plan or effect is invented.

export interface IntentTaskBinding {
  readonly task: string;
  readonly issue: string;
}
export interface IntentGeneration {
  readonly state: 'pending' | 'committed' | 'refused' | 'completed';
  readonly revision_id: string;
  readonly intent_id: string;
  readonly bindings: readonly IntentTaskBinding[];
}
function generationBinding(task: string, value: unknown): IntentTaskBinding {
  check(decisionID(task));
  check(typeof value === 'number' && Number.isSafeInteger(value) && value > 0);
  return {task, issue: String(value)};
}
// intentGeneration binds one native-work generation record to its exact
// intent and revision, with task-to-issue mappings as decimal strings.
export function intentGeneration(value: unknown, intent: string, revision: string): IntentGeneration {
  const data = object(value);
  check(data.intent_id === intent && data.revision_id === revision);
  check(
    data.state === 'pending' || data.state === 'committed' || data.state === 'refused' || data.state === 'completed'
  );
  const raw = data.task_issue_ids === undefined ? {} : object(data.task_issue_ids);
  const tasks = Object.keys(raw);
  check(tasks.length <= 64);
  return {
    state: data.state,
    revision_id: revision,
    intent_id: intent,
    bindings: tasks.map((task) => generationBinding(task, raw[task])),
  };
}

export type SourceState = 'ready' | 'missing' | 'unavailable';
export interface WorkOverviewInput {
  readonly revision?: IntentRevision;
  readonly generation?: IntentGeneration;
  readonly summary?: FactoryRepositorySummary;
  readonly issues: readonly FactoryIssueDetail[];
  readonly unreadIssues: readonly string[];
  readonly repositoryPath: string;
  readonly intentState: SourceState;
  readonly generationState: SourceState;
  readonly summaryState: SourceState;
}
export interface OverviewCounts {
  readonly delivered: number;
  readonly working: number;
  readonly waiting: number;
  readonly attention: number;
  readonly other: number;
}
export interface AttentionItem {
  readonly issue: string;
  readonly text: string;
  readonly link: string | undefined;
}
export interface DependencyRow {
  readonly task: string;
  readonly objective: string;
  readonly dependsOn: readonly string[];
  readonly issue: string | undefined;
  readonly issueLink: string | undefined;
}
export interface WorkOverviewModel {
  readonly direction: string;
  readonly counts: OverviewCounts;
  readonly countsText: string;
  readonly attention: readonly AttentionItem[];
  readonly dependencies: readonly DependencyRow[];
  readonly setupText: string;
  readonly outcomeNote: string;
  readonly availabilityNote: string;
}

const deliveryBuckets: Record<DeliveryState, keyof OverviewCounts> = {
  delivered: 'delivered',
  working: 'working',
  queued: 'working',
  waiting: 'waiting',
  checks_pending: 'waiting',
  merge_confirmed_closure_pending: 'waiting',
  recorded_complete_without_merge: 'waiting',
  needs_attention: 'attention',
  checks_failed: 'attention',
  evidence_refused: 'attention',
  not_authorized: 'attention',
  cancelled: 'other',
  unrecorded: 'other',
};
function bucketDelivery(delivery: DeliveryState): keyof OverviewCounts {
  return deliveryBuckets[delivery];
}
function countIssues(input: WorkOverviewInput): {counts: OverviewCounts; attention: AttentionItem[]} {
  const counts = {delivered: 0, working: 0, waiting: 0, attention: 0, other: 0};
  const attention: AttentionItem[] = [];
  for (const issue of input.issues) {
    const model = describeIssueDetail(issue, input.repositoryPath);
    counts[bucketDelivery(model.delivery)] += 1;
    if (bucketDelivery(model.delivery) === 'attention')
      attention.push({issue: issue.issue, text: `Issue #${issue.issue}: ${model.deliveryText}`, link: model.issueLink});
  }
  for (const issue of input.unreadIssues)
    attention.push({
      issue,
      text: `Issue #${issue} could not be read; it is not counted above.`,
      link: nativeIssueLink(input.repositoryPath, issue),
    });
  return {counts, attention};
}
function countsText(counts: OverviewCounts): string {
  const parts: string[] = [];
  if (counts.delivered > 0) parts.push(`${counts.delivered} delivered`);
  if (counts.working > 0) parts.push(`${counts.working} working`);
  if (counts.waiting > 0) parts.push(`${counts.waiting} waiting`);
  if (counts.attention > 0) parts.push(`${counts.attention} needs your decision`);
  if (counts.other > 0) parts.push(`${counts.other} other`);
  return parts.length > 0 ? parts.join(' · ') : 'No counted work.';
}
function directionText(input: WorkOverviewInput): string {
  if (input.intentState === 'unavailable') return 'Accepted direction could not be read.';
  if (input.intentState === 'missing' || !input.revision) return 'No accepted request recorded.';
  return input.revision.objective;
}
function issueForTask(generation: IntentGeneration | undefined, task: string): string | undefined {
  return generation?.bindings.find((binding) => binding.task === task)?.issue;
}
function dependencyRows(input: WorkOverviewInput): DependencyRow[] {
  if (!input.revision) return [];
  return input.revision.tasks.map((task) => {
    const issue = issueForTask(input.generation, task.key);
    const dependsOn = (task.depends_on ?? []).map((key) => {
      const dep = issueForTask(input.generation, key);
      return dep ? `issue #${dep}` : `task ${key.slice(0, 9)}`;
    });
    return {
      task: task.key,
      objective: task.objective,
      dependsOn,
      issue,
      issueLink: issue === undefined ? undefined : nativeIssueLink(input.repositoryPath, issue),
    };
  });
}
function setupText(input: WorkOverviewInput): string {
  if (input.summaryState === 'unavailable') return 'Project setup could not be read.';
  if (input.summaryState === 'missing' || !input.summary) return 'No Factory setup recorded for this repository.';
  if (!input.summary.preparation) return 'Project setup missing: no preparation recorded.';
  return input.summary.preparation.ready ? 'Project setup ready.' : 'Project setup missing: preparation incomplete.';
}
function outcomeNote(input: WorkOverviewInput, counts: OverviewCounts): string {
  if (input.issues.length === 0) return 'No work items recorded, so no outcome evidence exists yet.';
  if (counts.delivered === input.issues.length && input.unreadIssues.length === 0)
    return 'Every counted work item is delivered. Product-outcome evidence is not yet recorded separately.';
  return 'Overall completion needs evidence for the accepted product outcomes, not closed work items alone.';
}
function availabilityNote(input: WorkOverviewInput): string {
  const missing: string[] = [];
  if (input.intentState === 'unavailable') missing.push('accepted direction');
  if (input.generationState === 'unavailable') missing.push('native work mappings');
  if (input.summaryState === 'unavailable') missing.push('queue and setup');
  if (input.unreadIssues.length > 0)
    missing.push(`${input.unreadIssues.length} work item${input.unreadIssues.length === 1 ? '' : 's'}`);
  return missing.length > 0 ? `Unavailable right now: ${missing.join(', ')}.` : '';
}
// describeWorkOverview projects the accepted direction, factual
// work-item counts, attention, dependency rows and setup state. Each
// section reflects only its own read; unavailable reads are named.
export function describeWorkOverview(input: WorkOverviewInput): WorkOverviewModel {
  const {counts, attention} = countIssues(input);
  return {
    direction: directionText(input),
    counts,
    countsText: countsText(counts),
    attention,
    dependencies: dependencyRows(input),
    setupText: setupText(input),
    outcomeNote: outcomeNote(input, counts),
    availabilityNote: availabilityNote(input),
  };
}

export function renderWorkOverview(model: WorkOverviewModel): TemplateResult {
  return html`
    <section aria-label="Work overview">
      <p>${model.direction}</p>
      <p role="status">${model.countsText}</p>
      ${model.availabilityNote ? html`<p>${model.availabilityNote}</p>` : ''}
      ${
        model.attention.length > 0
          ? html`<ul>
              ${model.attention.map(
                (item) =>
                  html`<li>
                    ${item.text} ${item.link ? html`<a href=${item.link}>Open issue #${item.issue}</a>` : ''}
                  </li>`
              )}
            </ul>`
          : ''
      }
      <p>${model.setupText}</p>
      <p>${model.outcomeNote}</p>
      <ul>
        ${model.dependencies.map(
          (row) => html`<li>
            ${row.objective}
            ${row.dependsOn.length > 0 ? html`<span>Depends on: ${row.dependsOn.join(', ')}.</span>` : ''}
            ${row.issueLink ? html`<a href=${row.issueLink}>Issue #${row.issue}</a>` : 'No native issue mapped yet.'}
          </li>`
        )}
      </ul>
    </section>
  `;
}

export interface EligibleWork {
  readonly issue: string;
  readonly assessed_unix: number;
  readonly ageText: string;
  readonly link: string | undefined;
}
export interface WhyNextModel {
  readonly eligible: readonly EligibleWork[];
  readonly selection: string;
}
function ageText(assessedUnix: number, nowUnix: number): string {
  if (assessedUnix <= 0 || nowUnix < assessedUnix) return 'assessment time unknown';
  const minutes = Math.floor((nowUnix - assessedUnix) / 60);
  if (minutes < 1) return 'assessed just now';
  if (minutes < 60) return `assessed ${minutes} minute${minutes === 1 ? '' : 's'} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return `assessed ${hours} hour${hours === 1 ? '' : 's'} ago`;
  return `assessed ${Math.floor(hours / 24)} days ago`;
}
// describeWhyNext explains next-work selection from recorded facts
// only: a lone eligible issue needs no consultation, while several
// eligible issues await a recorded priority comparison.
export function describeWhyNext(
  issues: readonly FactoryIssueDetail[],
  repositoryPath: string,
  nowUnix: number
): WhyNextModel {
  const eligible = issues
    .filter((issue) => issue.readiness?.readiness === 'queued')
    .map((issue) => ({
      issue: issue.issue,
      assessed_unix: issue.readiness?.assessed_unix ?? 0,
      ageText: ageText(issue.readiness?.assessed_unix ?? 0, nowUnix),
      link: nativeIssueLink(repositoryPath, issue.issue),
    }))
    .sort((a, b) => (a.assessed_unix === 0 ? 1 : b.assessed_unix === 0 ? -1 : a.assessed_unix - b.assessed_unix));
  if (eligible.length === 0) return {eligible, selection: 'No work is currently eligible to start.'};
  if (eligible.length === 1)
    return {eligible, selection: `Issue #${eligible[0]?.issue} is next: it is the only eligible work.`};
  return {
    eligible,
    selection:
      `${eligible.length} issues are eligible. No recorded priority comparison selected one, so none is marked next. ` +
      'Oldest assessment first.',
  };
}

export interface PrerequisiteModel {
  readonly issue: string;
  readonly link: string | undefined;
  readonly waitingText: string;
  readonly planNote: string;
}
// describePrerequisites names each issue's waiting reasons from its
// recorded readiness. Planning records do not exist in this read, so
// the plan note says exactly that instead of inventing freshness.
export function describePrerequisites(
  issues: readonly FactoryIssueDetail[],
  repositoryPath: string
): PrerequisiteModel[] {
  return issues.map((issue) => {
    const blockers = issue.readiness?.blockers ?? [];
    return {
      issue: issue.issue,
      link: nativeIssueLink(repositoryPath, issue.issue),
      waitingText:
        blockers.length === 0
          ? `Issue #${issue.issue}: ${issue.readiness ? `readiness ${issue.readiness.readiness}` : 'no readiness record'}.`
          : `Issue #${issue.issue} waits on: ${blockers.map((blocker) => blocker.resolution).join(' ')}`,
      planNote: 'No recorded plan for this work.',
    };
  });
}

export function renderWorkPlan(whyNext: WhyNextModel, prerequisites: readonly PrerequisiteModel[]): TemplateResult {
  return html`
    <section aria-label="Planning and next work">
      <p role="status">${whyNext.selection}</p>
      <ul>
        ${whyNext.eligible.map(
          (item) => html`<li>
            ${item.link ? html`<a href=${item.link}>Issue #${item.issue}</a>` : `Issue #${item.issue}`}:
            ${item.ageText}.
          </li>`
        )}
      </ul>
      <ul>
        ${prerequisites.map(
          (row) => html`<li>
            ${row.link ? html`<a href=${row.link}>Issue #${row.issue}</a>` : `Issue #${row.issue}`}: ${row.waitingText}
            ${row.planNote}
          </li>`
        )}
      </ul>
    </section>
  `;
}

export interface DirectionRevisionModel {
  readonly headline: string;
  readonly attribution: string;
  readonly decisions: readonly {description: string; decision: string; correction?: string}[];
  readonly taskCount: number;
  readonly allowanceText: string;
  readonly deltaText: string;
}
// describeDirectionRevision presents an accepted revision's
// attributable facts. Revision comparison and allowance consumption
// have no read here, so both say exactly that.
export function describeDirectionRevision(
  revision: IntentRevision | undefined,
  state: SourceState
): DirectionRevisionModel {
  if (state === 'unavailable') {
    return {
      headline: 'Accepted direction could not be read.',
      attribution: '',
      decisions: [],
      taskCount: 0,
      allowanceText: 'Planning allowance could not be read.',
      deltaText: 'Revision comparison unavailable.',
    };
  }
  if (state === 'missing' || !revision) {
    return {
      headline: 'No accepted direction recorded.',
      attribution: '',
      decisions: [],
      taskCount: 0,
      allowanceText: 'No planning allowance recorded.',
      deltaText: 'Revision comparison unavailable.',
    };
  }
  return {
    headline: revision.objective,
    attribution: `Revision ${revision.id}${revision.predecessor ? `, follows ${revision.predecessor}` : ''}. State: ${revision.state}.`,
    decisions: revision.ideas.map((idea) => ({
      description: idea.description,
      decision: idea.decision,
      ...(idea.correction === undefined ? {} : {correction: idea.correction}),
    })),
    taskCount: revision.tasks.length,
    allowanceText: `Accepted planning calls on this revision: ${revision.planning_calls}. Live allowance reads separately.`,
    deltaText: 'Revision comparison unavailable.',
  };
}

export interface AllowanceModel {
  readonly summaryText: string;
  readonly attributionText: string;
  readonly effectText: string;
}
// describeAllowance projects one planning allowance: the bounded
// amount in real planning-call units with reserved, consumed and
// remaining consumption plus its attribution chain.
export function describeAllowance(allowance: PlanningAllowanceView | undefined, state: SourceState): AllowanceModel {
  if (state === 'unavailable')
    return {summaryText: 'Planning allowance could not be read.', attributionText: '', effectText: ''};
  if (state === 'missing' || !allowance)
    return {summaryText: 'No planning allowance recorded.', attributionText: '', effectText: ''};
  const remaining = Math.max(0, allowance.amount - allowance.reserved - allowance.consumed);
  return {
    summaryText:
      `Planning allowance: ${allowance.amount} planning calls; ` +
      `${allowance.reserved} reserved, ${allowance.consumed} consumed, ${remaining} remaining (ceiling ${allowance.ceiling}).`,
    attributionText:
      `Allowance ${allowance.id.slice(0, 9)}, set by owner ${allowance.owner}` +
      `${allowance.predecessor ? `, adjusts ${allowance.predecessor.slice(0, 9)}` : ''}.`,
    effectText: allowance.effect ? `Recorded effect: ${allowance.effect}` : '',
  };
}

export function renderAllowance(model: AllowanceModel): TemplateResult {
  return html`
    <section aria-label="Planning allowance">
      <p role="status">${model.summaryText}</p>
      ${model.attributionText ? html`<p>${model.attributionText}</p>` : ''}
      ${model.effectText ? html`<p>${model.effectText}</p>` : ''}
    </section>
  `;
}

export function renderDirectionRevision(model: DirectionRevisionModel): TemplateResult {
  return html`
    <section aria-label="Direction revision">
      <p>${model.headline}</p>
      ${model.attribution ? html`<p>${model.attribution}</p>` : ''}
      <ul>
        ${model.decisions.map(
          (idea) => html`<li>
            ${idea.description} Decision: ${idea.decision}.${idea.correction ? ` Correction: ${idea.correction}.` : ''}
          </li>`
        )}
      </ul>
      <p>${model.taskCount} planned work items.</p>
      <p>${model.allowanceText}</p>
      <p>${model.deltaText}</p>
    </section>
  `;
}

export interface RequestMappingRow {
  readonly task: string;
  readonly issue: string;
  readonly effective: boolean;
  readonly issueLink: string | undefined;
}
export interface RequestOverviewModel {
  readonly direction: string;
  readonly stateText: string;
  readonly counts: OverviewCounts;
  readonly countsText: string;
  readonly attention: readonly AttentionItem[];
  readonly mappings: readonly RequestMappingRow[];
  readonly setupText: string;
  readonly generationText: string;
  readonly outcomeNote: string;
  readonly observedText: string;
}

function requestAttentionItem(item: RequestAttention, repositoryPath: string): AttentionItem {
  if (item.code === 'work_attention' && item.issue !== undefined)
    return {
      issue: item.issue,
      text: `Issue #${item.issue} needs attention.`,
      link: nativeIssueLink(repositoryPath, item.issue),
    };
  if (item.code === 'unresolved_assumption')
    return {issue: '', text: 'One consequential assumption still needs your decision.', link: undefined};
  return {issue: '', text: 'Native work generation was refused.', link: undefined};
}
function requestSetupText(state: string): string {
  if (state === 'repository_unassigned') return 'No repository assigned to this request.';
  if (state === 'project_missing') return 'Project setup missing: no project prepared.';
  if (state === 'preparing') return 'Project setup preparing.';
  return 'Project setup ready.';
}
function requestGenerationText(state: string): string {
  if (state === 'refused') return 'Native work generation was refused.';
  if (state === 'pending') return 'Native work generation pending.';
  if (state === 'committed' || state === 'completed') return 'Native work generated.';
  if (state === 'not_requested') return 'Native work not requested yet.';
  return `Generation state: ${state}.`;
}
function requestDirectionText(view: RequestOverview): string {
  if (view.state === 'accepted') return view.direction.objective;
  return `Request is ${view.state}: ${view.direction.objective}`;
}
// describeRequestOverview projects the H-R request overview: the exact
// direction, evidenced work-item counts, attention, native mappings and
// setup. Outcome evidence counts render as counts, never percentages.
export function describeRequestOverview(view: RequestOverview, repositoryPath: string): RequestOverviewModel {
  const counted =
    view.work_items.delivered + view.work_items.working + view.work_items.waiting + view.work_items.attention;
  const counts: OverviewCounts = {
    delivered: view.work_items.delivered,
    working: view.work_items.working,
    waiting: view.work_items.waiting,
    attention: view.work_items.attention,
    other: Math.max(0, view.work_items.total - counted),
  };
  const observed = activityTimestamp(view.observed_unix);
  return {
    direction: requestDirectionText(view),
    stateText:
      view.state === 'accepted'
        ? `${view.direction.tasks_total} work items planned, ${view.direction.unresolved_assumptions} unresolved assumptions.`
        : `Revision ${view.revision.slice(0, 9)}.`,
    counts,
    countsText: countsText(counts),
    attention: view.attention.map((item) => requestAttentionItem(item, repositoryPath)),
    mappings: view.mappings.map((mapping) => ({
      task: mapping.task_key.slice(0, 9),
      issue: mapping.issue_index,
      effective: mapping.effective,
      issueLink: nativeIssueLink(repositoryPath, mapping.issue_index),
    })),
    setupText: requestSetupText(view.setup.state),
    generationText: requestGenerationText(view.generation),
    outcomeNote:
      view.work_items.outcomes_total === 0
        ? 'No accepted product outcomes recorded.'
        : `${view.work_items.outcomes_evidenced} of ${view.work_items.outcomes_total} accepted product outcomes evidenced.`,
    observedText: observed ? `Observed ${observed.text}.` : 'Observation time unknown.',
  };
}

// OverviewScope is the child-view feed the reached overview
// publishes: the read keys it used plus the bound work items with
// the capabilities from its own issue reads, when it read them.
// A retired scope carries repository '' and no issues.
export interface OverviewScopeIssue {
  readonly issue: string;
  readonly capabilities?: WorkCapabilities;
}
export interface OverviewScope {
  readonly repository: string;
  readonly intentID?: string;
  readonly issues: readonly OverviewScopeIssue[];
}
// overviewScopeFromMappings feeds scope from H-R request mappings:
// effective mappings only; capabilities stay unknown (the overview
// route reads no per-issue capabilities).
export function overviewScopeFromMappings(
  mappings: readonly RequestMappingRow[],
  repository: string,
  intentID?: string
): OverviewScope {
  return {
    repository,
    ...(intentID === undefined ? {} : {intentID}),
    issues: mappings.filter((row) => row.effective).map((row) => ({issue: row.issue})),
  };
}
// overviewScopeFromBound feeds scope from composed fallback reads:
// every issue the overview read, with the capabilities it saw.
export function overviewScopeFromBound(
  details: readonly FactoryIssueDetail[],
  repository: string,
  intentID?: string
): OverviewScope {
  return {
    repository,
    ...(intentID === undefined ? {} : {intentID}),
    issues: details.map((detail) => ({issue: detail.issue, capabilities: detail.capabilities})),
  };
}
export function retiredOverviewScope(): OverviewScope {
  return {repository: '', issues: []};
}
// planContextFromScope builds the plan context over every scoped
// issue. An empty scope yields an empty plan, never a guess.
export function planContextFromScope(
  transport: PreparedExtensionMount,
  repository: string,
  repositoryPath: string,
  scope: OverviewScope
): WorkPlanContext {
  return {transport, repository, repositoryPath, issues: scope.issues.map((entry) => entry.issue)};
}
export interface DetailScopeRuns {
  readonly runIDs: string[];
  readonly runID?: string;
  readonly projectID?: string;
  readonly environmentID?: string;
}
// detailContextFromScope builds one issue's detail context. Issue
// selection stays with the composer; capabilities come from the
// scope when the overview read that issue, else stay permissive
// until the detail's own read lands.
export function detailContextFromScope(
  transport: PreparedExtensionMount,
  repository: string,
  repositoryPath: string,
  scope: OverviewScope,
  issue: string,
  runs: DetailScopeRuns
): WorkDetailContext {
  const known = scope.issues.find((entry) => entry.issue === issue)?.capabilities;
  return {
    transport,
    repository,
    repositoryPath,
    issue,
    runIDs: runs.runIDs,
    capabilities: upfrontCapabilities(known, runs.environmentID !== undefined),
    ...(runs.runID === undefined ? {} : {runID: runs.runID}),
    ...(runs.projectID === undefined ? {} : {projectID: runs.projectID}),
    ...(runs.environmentID === undefined ? {} : {environmentID: runs.environmentID}),
  };
}
// revisionContextFromScope builds the direction-revision context.
// Without an intent the revision has no read key: undefined means
// do not mount.
export function revisionContextFromScope(
  transport: PreparedExtensionMount,
  scope: OverviewScope
): DirectionRevisionContext | undefined {
  if (scope.intentID === undefined) return undefined;
  return {transport, intentID: scope.intentID};
}

export function renderRequestOverview(model: RequestOverviewModel): TemplateResult {
  return html`
    <section aria-label="Request overview">
      <p>${model.direction}</p>
      <p>${model.stateText}</p>
      <p role="status">${model.countsText}</p>
      ${
        model.attention.length > 0
          ? html`<ul>
              ${model.attention.map(
                (item) =>
                  html`<li>
                    ${item.text} ${item.link ? html`<a href=${item.link}>Open issue #${item.issue}</a>` : ''}
                  </li>`
              )}
            </ul>`
          : ''
      }
      <p>${model.setupText}</p>
      <p>${model.generationText}</p>
      <p>${model.outcomeNote}</p>
      <p>${model.observedText}</p>
      <ul>
        ${model.mappings.map(
          (row) => html`<li>
            Task ${row.task} →
            ${row.issueLink ? html`<a href=${row.issueLink}>issue #${row.issue}</a>` : `issue #${row.issue}`}
            ${row.effective ? '' : '(outside the authorized subset)'}
          </li>`
        )}
      </ul>
    </section>
  `;
}

export interface RecordedPlanModel {
  readonly freshnessText: string;
  readonly strategyText: string;
  readonly steps: readonly string[];
  readonly missingText: string;
}
const freshnessReasonText: Record<string, string> = {
  current_readiness_missing: 'current readiness is missing',
  acceptance_changed: 'acceptance changed',
  inputs_changed: 'inputs changed',
};
function freshnessBase(freshness: PlanFreshness): string {
  if (freshness.state === 'missing') return 'No recorded plan.';
  if (freshness.state === 'stale')
    return `Plan stale: ${freshness.reasons.map((reason) => freshnessReasonText[reason] ?? reason).join('; ') || 'recorded inputs moved'}.`;
  if (freshness.state === 'provisional') return 'Provisional plan: conveys no permission to execute.';
  return 'Current plan.';
}
// describeRecordedPlan projects one issue's recorded plan and
// freshness. Stale reasons name the factual comparison behind them;
// the provisional flag stays visible next to any state.
export function describeRecordedPlan(plan: RecordedPlan | undefined, freshness: PlanFreshness): RecordedPlanModel {
  const base = freshnessBase(freshness);
  return {
    freshnessText:
      freshness.provisional && freshness.state !== 'provisional' && freshness.state !== 'missing'
        ? `${base} Recorded as provisional.`
        : base,
    strategyText: plan ? `Strategy: ${plan.strategy}. Outcome: ${plan.outcome}.` : '',
    steps: plan?.steps ?? [],
    missingText:
      plan && plan.missing.length > 0
        ? `Missing prerequisites: ${plan.missing.map((blocker) => blocker.resolution).join(' ')}`
        : 'No missing prerequisites recorded.',
  };
}

export interface ChoiceCandidateRow {
  readonly issue: string;
  readonly ageText: string;
  readonly estimatesText: string;
  readonly link: string | undefined;
}
export interface RecordedChoiceModel {
  readonly selection: string;
  readonly judgmentText: string;
  readonly candidates: readonly ChoiceCandidateRow[];
  readonly moreText: string;
}
function durationText(seconds: number): string {
  if (seconds < 60) return `${seconds} second${seconds === 1 ? '' : 's'}`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'}`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return `${hours} hour${hours === 1 ? '' : 's'}`;
  return `${Math.floor(hours / 24)} days`;
}
function estimatesText(impact: boolean, risk: boolean, effort: boolean): string {
  const known = [...(impact ? ['impact'] : []), ...(risk ? ['risk'] : []), ...(effort ? ['effort'] : [])];
  return known.length > 0 ? `${known.join(', ')} recorded` : 'no estimates recorded';
}
function choiceSelection(comparison: PriorityComparison): string {
  if (comparison.selected !== undefined && comparison.selected_visible) {
    const winner = comparison.candidates.find((candidate) => candidate.id === comparison.selected);
    const probability = comparison.probabilities?.[comparison.selected];
    const odds = probability === undefined ? '' : ` (recorded probability ${probability}).`;
    if (winner) return `Recorded choice: issue #${winner.issue}.${odds}`;
    return `Recorded choice: ${comparison.selected}.${odds}`;
  }
  if (!comparison.selected_visible) return "Another repository's work was selected.";
  return 'No choice recorded in the latest comparison.';
}
function judgmentText(comparison: PriorityComparison): string {
  const latest = comparison.calls[comparison.calls.length - 1];
  if (!latest) return 'No judgment call recorded.';
  if (latest.failure) return `Latest judgment call failed: ${latest.failure}.`;
  const confidence = latest.confidence === undefined ? '' : ` Confidence ${latest.confidence}.`;
  return `Latest judgment${latest.model ? ` by ${latest.model}` : ''}.${confidence}`;
}
// describeRecordedChoice explains the newest recorded next-work
// comparison: the recorded choice with its uncertainty, the eligible
// candidates with fairness inputs, and masked foreign winners.
export function describeRecordedChoice(plan: IssuePlan, repositoryPath: string): RecordedChoiceModel {
  const comparison = plan.comparisons[0];
  if (!comparison)
    return {
      selection: 'No recorded priority comparison names this issue.',
      judgmentText: '',
      candidates: [],
      moreText: '',
    };
  return {
    selection: choiceSelection(comparison),
    judgmentText: judgmentText(comparison),
    candidates: comparison.candidates.map((candidate) => ({
      issue: candidate.issue,
      ageText: `waiting ${durationText(candidate.age_seconds)}`,
      estimatesText: estimatesText(candidate.impact_known, candidate.risk_known, candidate.effort_known),
      link: nativeIssueLink(repositoryPath, candidate.issue),
    })),
    moreText: plan.next_before_unix !== undefined ? 'Older comparisons exist beyond this page.' : '',
  };
}

export type LifecycleEventKind = 'issue_created' | 'pr_created' | 'verified_merge';
export interface LifecycleEffectModel {
  readonly event: LifecycleEventKind;
  readonly title: string;
  readonly stateText: string;
  readonly configText: string;
  readonly resultsText: readonly string[];
  readonly resultsLinks: readonly (string | undefined)[];
  readonly canConfigure: boolean;
  readonly config: LifecycleEffectConfig | undefined;
}
const lifecycleTitles: Record<LifecycleEventKind, string> = {
  issue_created: 'Generated issue label and assignee',
  pr_created: 'Pull request human review',
  verified_merge: 'Original-issue completion note',
};
function lifecycleTargets(event: LifecycleEventKind, config: LifecycleEffectConfig): string {
  if (event === 'issue_created')
    return `Label ${config.label_id ?? 'unset'}, assignee ${config.assignee_id ?? 'unset'}.`;
  if (event === 'pr_created')
    return config.reviewer_ids.length > 0 ? `Reviewers ${config.reviewer_ids.join(', ')}.` : 'No reviewers set.';
  return config.note ? `Note: ${config.note}` : 'No note set.';
}
// lifecycleResultLink builds a pull link only for pr_created
// actions carrying a journaled pull number, and only when the
// native repository path is known. Work-issue indexes and native
// target IDs never link as PRs.
function lifecycleResultLink(
  event: LifecycleEventKind,
  action: LifecycleEffectAction,
  repositoryPath: string
): string | undefined {
  if (event !== 'pr_created' || action.pr_number === undefined) return undefined;
  return nativePullLink(repositoryPath, action.pr_number);
}
function lifecycleResultLine(action: LifecycleEffectAction): string {
  const entries = Object.entries(action.results);
  const outcomes =
    entries.length > 0 ? entries.map(([target, result]) => `${target}: ${result}`).join('; ') : 'no results recorded';
  const revoked = action.revoked ? ', revoked' : '';
  const settled = action.settled ? ', settled' : '';
  const submitter = action.actor_id === '0' ? 'Submitter unclaimed' : `Submitted by ${action.actor_id}`;
  const authorized = action.config_owner ? `, authorized by ${action.config_owner}` : '';
  const target = action.target_id ? `Native target ${action.target_id}.` : 'Native target unresolved.';
  return (
    `Action ${action.id.slice(0, 8)} on issue #${action.issue_id}: ` +
    `${action.attempts}/${action.max_attempts} attempts${revoked}${settled}. ` +
    `${submitter}${authorized}. ${target} ${outcomes}.`
  );
}
// lifecycleSubmitterText names the enrolled submitter that will act
// ("Unknown" while the producer has not resolved one) and the
// executor state when the producer reports it.
function lifecycleSubmitterText(config: LifecycleEffectConfig): string {
  const submitter = config.submitter_id === '' ? 'Unknown' : config.submitter_id;
  if (config.executor_ready === undefined) return `Submitter ${submitter}.`;
  return `Submitter ${submitter}, executor ${config.executor_ready ? 'ready' : 'not ready'}.`;
}
// describeLifecycleEffect gates one optional effect on its own runnable
// capability (native proof plus a configured executor). Unrunnable
// effects stay unavailable with no configuration shown; configured
// effects name their exact authorizer, submitter, native target and
// per-target results. PR rows link their journaled pull number when
// the native repository path is known.
export function describeLifecycleEffect(
  event: LifecycleEventKind,
  config: LifecycleEffectConfig | undefined,
  configState: SourceState,
  actions: readonly LifecycleEffectAction[],
  repositoryPath = ''
): LifecycleEffectModel {
  const resultsLinks = actions.map((action) => lifecycleResultLink(event, action, repositoryPath));
  if (configState === 'unavailable' || !config)
    return {
      event,
      title: lifecycleTitles[event],
      stateText: configState === 'unavailable' ? 'Configuration could not be read.' : 'No configuration recorded.',
      configText: '',
      resultsText: actions.map(lifecycleResultLine),
      resultsLinks,
      canConfigure: false,
      config: undefined,
    };
  if (!config.capable)
    return {
      event,
      title: lifecycleTitles[event],
      stateText: 'Unavailable: its native capability is not runnable yet.',
      configText: '',
      resultsText: actions.map(lifecycleResultLine),
      resultsLinks,
      canConfigure: false,
      config,
    };
  return {
    event,
    title: lifecycleTitles[event],
    stateText: config.revision === '0' ? 'Not configured yet.' : config.enabled ? 'Active.' : 'Disabled.',
    configText:
      `Authorized by ${config.actor_id || 'unset'}, revision ${config.revision}. ` +
      `${lifecycleTargets(event, config)} ${lifecycleSubmitterText(config)}`,
    resultsText: actions.map(lifecycleResultLine),
    resultsLinks,
    canConfigure: true,
    config,
  };
}

export interface LifecycleEffectInput {
  readonly enabled: boolean;
  readonly label_id: string;
  readonly assignee_id: string;
  readonly reviewer_ids: string;
  readonly note: string;
}
export interface LifecycleEffectCommands {
  readonly save: (event: LifecycleEventKind, input: LifecycleEffectInput) => void;
}

function renderIssueCreatedFields(model: LifecycleEffectModel): TemplateResult {
  const label = model.config?.label_id ?? '';
  const assignee = model.config?.assignee_id ?? '';
  return html`<label>Label ID <input name="label_id" inputmode="numeric" .value=${label} /></label>
    <label>Assignee ID <input name="assignee_id" inputmode="numeric" .value=${assignee} /></label>`;
}

function renderPrCreatedFields(model: LifecycleEffectModel): TemplateResult {
  const reviewers = (model.config?.reviewer_ids ?? []).join(', ');
  return html`<label
    >Reviewer IDs, comma separated <input name="reviewer_ids" inputmode="text" .value=${reviewers}
  /></label>`;
}

function renderVerifiedMergeFields(model: LifecycleEffectModel): TemplateResult {
  const note = model.config?.note ?? '';
  return html`<label>Completion note <input name="note" maxlength="1000" .value=${note} /></label>`;
}

function renderEffectFields(model: LifecycleEffectModel): TemplateResult {
  if (model.event === 'issue_created') return renderIssueCreatedFields(model);
  if (model.event === 'pr_created') return renderPrCreatedFields(model);
  return renderVerifiedMergeFields(model);
}

export function renderLifecycleEffects(
  models: readonly LifecycleEffectModel[],
  commands: LifecycleEffectCommands
): TemplateResult {
  return html`
    <section aria-label="Optional effects">
      <p>
        Optional effects stay off unless their own native capability is verified. They never replace mandatory review.
      </p>
      <ul>
        ${models.map(
          (model) => html`<li>
            <p>${model.title}</p>
            <p role="status">${model.stateText}</p>
            ${model.configText ? html`<p>${model.configText}</p>` : ''}
            <ul>
              ${model.resultsText.map(
                (line, index) => html`<li>
                  ${line}${model.resultsLinks[index] ? html` <a href=${model.resultsLinks[index]}>Native target</a>` : ''}
                </li>`
              )}
            </ul>
            ${
              model.canConfigure
                ? html`<form
                    @submit=${(submit: SubmitEvent) => {
                      submit.preventDefault();
                      const form = submit.target;
                      if (!(form instanceof HTMLFormElement)) return;
                      const data = new FormData(form);
                      commands.save(model.event, {
                        enabled: data.get('enabled') === 'on',
                        label_id: String(data.get('label_id') ?? ''),
                        assignee_id: String(data.get('assignee_id') ?? ''),
                        reviewer_ids: String(data.get('reviewer_ids') ?? ''),
                        note: String(data.get('note') ?? ''),
                      });
                    }}
                  >
                    <label
                      ><input type="checkbox" name="enabled" .checked=${model.config?.enabled ?? false} />Enabled</label
                    >
                    ${renderEffectFields(model)}
                    <button class="ui button" type="submit">Save effect configuration</button>
                  </form>`
                : ''
            }
          </li>`
        )}
      </ul>
    </section>
  `;
}

export function renderRecordedPlan(plan: RecordedPlanModel, choice: RecordedChoiceModel): TemplateResult {
  return html`
    <section aria-label="Recorded plan">
      <p role="status">${plan.freshnessText}</p>
      ${plan.strategyText ? html`<p>${plan.strategyText}</p>` : ''}
      <p>${plan.missingText}</p>
      <ul>
        ${plan.steps.map((step) => html`<li>${step}</li>`)}
      </ul>
      <p>${choice.selection}</p>
      ${choice.judgmentText ? html`<p>${choice.judgmentText}</p>` : ''}
      ${choice.moreText ? html`<p>${choice.moreText}</p>` : ''}
      <ul>
        ${choice.candidates.map(
          (row) => html`<li>
            ${row.link ? html`<a href=${row.link}>Issue #${row.issue}</a>` : `Issue #${row.issue}`}: ${row.ageText},
            ${row.estimatesText}.
          </li>`
        )}
      </ul>
    </section>
  `;
}

export interface PlanningReferenceModel {
  readonly found: boolean;
  readonly referenceText: string;
  readonly truncatedNote: string;
}

function planningEntryText(entry: AccessPlanningEntry): string {
  const editable = entry.editable.editable
    ? 'You may adjust it.'
    : `Adjustment refused: ${entry.editable.reason ?? 'no reason given'}.`;
  const effect = entry.effect === undefined ? '' : ` Effect: ${entry.effect}.`;
  return (
    `Repository allowance reference: ${entry.amount} planning calls, ` +
    `${entry.reserved} reserved, ${entry.consumed} consumed, ${entry.available} available ` +
    `(ceiling ${entry.ceiling}). Status ${entry.status}.${effect} ${editable}`
  );
}

// describePlanningReference finds this request's entry in the
// repository allowance listing. A truncated listing may hide the
// entry; that stays a stated gap, never a no-allowance verdict.
export function describePlanningReference(
  access: FactoryAccess | undefined,
  state: SourceState,
  intent: string
): PlanningReferenceModel {
  const truncatedNote = access?.planning_truncated ? 'The listing shows only the first 20 allowances.' : '';
  if (state !== 'ready' || !access)
    return {
      found: false,
      referenceText:
        state === 'unavailable'
          ? 'Repository allowance listing could not be read.'
          : 'No allowance listing recorded for this repository.',
      truncatedNote: '',
    };
  const entry = access.planning.find((candidate) => candidate.intent === intent);
  if (!entry)
    return {
      found: false,
      referenceText: access.planning_truncated
        ? 'This request is not among the first 20 allowances shown.'
        : 'No bound allowance for this request in this repository.',
      truncatedNote,
    };
  return {found: true, referenceText: planningEntryText(entry), truncatedNote};
}

export function renderPlanningReference(model: PlanningReferenceModel): TemplateResult {
  return html`
    <section aria-label="Repository allowance reference">
      <p role="status">${model.referenceText}</p>
      ${model.truncatedNote ? html`<p>${model.truncatedNote}</p>` : ''}
    </section>
  `;
}

export interface AccessStandingModel {
  readonly powersText: string;
  readonly verdictText: string;
  readonly actionsText: readonly string[];
}

function standingPowers(access: FactoryAccess): string {
  const held: string[] = [];
  if (access.viewer.code_write) held.push('code write');
  if (access.viewer.owner_or_admin) held.push('owner or admin');
  if (access.viewer.environment_owner) held.push('environment owner');
  if (access.viewer.operator) held.push('appliance operator');
  if (access.viewer.member) held.push('project member');
  const session = access.viewer.session_current
    ? 'Session current.'
    : 'Session changed; saves are refused until it is current.';
  return held.length === 0 ? `You hold no repository powers. ${session}` : `You hold: ${held.join(', ')}. ${session}`;
}

function standingVerdict(access: FactoryAccess): string {
  if (access.effective.effective && access.effective.dispatch_open) return 'Factory dispatch is effective and open.';
  if (access.effective.effective) return 'Authority is effective but dispatch is closed.';
  const missing = access.effective.missing.length === 0 ? 'no reason given' : access.effective.missing.join(', ');
  return `Factory authority is not effective: ${missing}.`;
}

function standingAction(code: string, resolveBy: string, setting: string): string {
  const resolver = resolveBy === '' ? 'unassigned' : resolveBy;
  const section = setting === '' ? 'no setting named' : setting;
  return `${code}: resolve by ${resolver} via ${section}.`;
}

// describeAccessStanding states the viewer's exact powers and the
// authority verdict behind permission-gated controls. Unknown
// resolvers stay visible as unassigned, never guessed.
export function describeAccessStanding(access: FactoryAccess | undefined, state: SourceState): AccessStandingModel {
  if (state !== 'ready' || !access)
    return {
      powersText:
        state === 'unavailable' ? 'Your permissions could not be read.' : 'No permission record for this repository.',
      verdictText: 'Factory authority is unknown.',
      actionsText: [],
    };
  return {
    powersText: standingPowers(access),
    verdictText: standingVerdict(access),
    actionsText: access.required_actions.map((action) =>
      standingAction(action.code, action.resolve_by, action.setting)
    ),
  };
}

export function renderAccessStanding(model: AccessStandingModel): TemplateResult {
  return html`
    <section aria-label="Access standing">
      <p>${model.powersText}</p>
      <p role="status">${model.verdictText}</p>
      ${
        model.actionsText.length === 0
          ? ''
          : html`<ul>
              ${model.actionsText.map((line) => html`<li>${line}</li>`)}
            </ul>`
      }
    </section>
  `;
}

export interface RevisionEntryModel {
  readonly headerText: string;
  readonly lineageText: string;
  readonly tasksText: string;
  readonly taskLines: readonly string[];
  readonly ideasText: string;
  readonly ideaLines: readonly string[];
  readonly scopeLines: readonly string[];
  readonly workLines: readonly string[];
}
export interface RevisionHistoryModel {
  readonly stateText: string;
  readonly truncatedNote: string;
  readonly entries: readonly RevisionEntryModel[];
}

function revisionTaskChangedLine(change: RevisionTaskChange): string {
  const parts = [`${change.key} (${change.changed_fields.join(', ')})`];
  if (change.old_objective !== undefined)
    parts.push(`objective: ${change.old_objective} → ${change.new_objective ?? ''}`);
  if (change.old_outcomes !== undefined)
    parts.push(`outcomes: ${(change.old_outcomes ?? []).join('; ')} → ${(change.new_outcomes ?? []).join('; ')}`);
  return parts.join('; ');
}

function revisionTaskLines(entry: RevisionEntry): string[] {
  const lines = entry.comparison.tasks.added.map((task) => `+ ${task.key}: ${task.objective}`);
  for (const task of entry.comparison.tasks.removed) lines.push(`- ${task.key}: ${task.objective}`);
  for (const change of entry.comparison.tasks.changed) lines.push(`~ ${revisionTaskChangedLine(change)}`);
  return lines;
}

function revisionIdeaChangedLine(change: RevisionIdeaChange): string {
  const flags: string[] = [];
  if (change.correction_changed) flags.push('correction changed');
  if (change.content_changed) flags.push('content changed');
  const note = flags.length === 0 ? 'decision only' : flags.join(', ');
  return `${change.id}: ${change.from_decision} → ${change.to_decision} (${note})`;
}

function revisionIdeaLines(entry: RevisionEntry): string[] {
  const lines = entry.comparison.ideas.added.map((idea) => `+ ${idea.id} (${idea.decision}): ${idea.description}`);
  for (const idea of entry.comparison.ideas.removed) lines.push(`- ${idea.id} (${idea.decision}): ${idea.description}`);
  for (const change of entry.comparison.ideas.changed) lines.push(`~ ${revisionIdeaChangedLine(change)}`);
  return lines;
}

function revisionScopeLines(entry: RevisionEntry): string[] {
  const lines: string[] = [];
  if (entry.comparison.effective_added.length > 0)
    lines.push(`Effective outcomes added: ${entry.comparison.effective_added.join(', ')}.`);
  if (entry.comparison.effective_removed.length > 0)
    lines.push(`Effective outcomes removed: ${entry.comparison.effective_removed.join(', ')}.`);
  if (entry.comparison.repository_change)
    lines.push(`Repository ${entry.comparison.repository_change.from} → ${entry.comparison.repository_change.to}.`);
  if (entry.comparison.state_change)
    lines.push(`State ${entry.comparison.state_change.from} → ${entry.comparison.state_change.to}.`);
  if (entry.comparison.edges_changed) lines.push('Dependency edges changed.');
  return lines;
}

function revisionSubmittedNote(generation: RevisionGeneration): string {
  if (generation.state === 'not_requested') return '';
  const count = `${generation.submitted_keys.length} submitted key${generation.submitted_keys.length === 1 ? '' : 's'}`;
  const differs = generation.submitted_differs ? ' Differs from accepted direction.' : '';
  return ` Submitted graph: ${count}.${differs}`;
}

function revisionEntryLineage(entry: RevisionEntry): string {
  const from = entry.generated_from === undefined ? '' : ` Generated from ${entry.generated_from}.`;
  const first = entry.comparison.initial ? ' First revision.' : '';
  const stamp = activityTimestamp(entry.created_unix);
  const when = stamp === undefined ? 'time unknown' : stamp.text;
  return `Repository ${entry.repository}, recorded ${when}.${from}${first} Generation ${entry.generation.state}.${revisionSubmittedNote(entry.generation)}`;
}

function describeRevisionEntry(entry: RevisionEntry): RevisionEntryModel {
  const tasks = entry.comparison.tasks;
  const ideas = entry.comparison.ideas;
  return {
    headerText: `Revision ${entry.revision} (${entry.state})`,
    lineageText: revisionEntryLineage(entry),
    tasksText: `Tasks: ${tasks.added.length} added, ${tasks.removed.length} removed, ${tasks.changed.length} changed, ${tasks.reused.length} reused, ${tasks.unchanged_count} unchanged.`,
    taskLines: revisionTaskLines(entry),
    ideasText: `Ideas: ${ideas.added.length} added, ${ideas.removed.length} removed, ${ideas.changed.length} changed.`,
    ideaLines: revisionIdeaLines(entry),
    scopeLines: revisionScopeLines(entry),
    workLines: entry.work.map(
      (row) =>
        `${row.task_key} → issue ${row.issue_index}: ${row.state} (${row.authority}), content version ${row.content_version}`
    ),
  };
}

// describeRevisionHistory composes the before/after and affected
// work of each shown scope revision from the authoritative v3
// classification, keeping the submitted graph distinct from the
// accepted direction it was generated against.
export function describeRevisionHistory(
  history: RevisionHistory | undefined,
  state: SourceState
): RevisionHistoryModel {
  if (state !== 'ready' || !history)
    return {
      stateText: state === 'unavailable' ? 'Revision history could not be read.' : 'No revision history recorded yet.',
      truncatedNote: '',
      entries: [],
    };
  return {
    stateText: `${history.revisions.length} revision${history.revisions.length === 1 ? '' : 's'} shown, newest first.`,
    truncatedNote: history.truncated ? 'Older revisions exist beyond this page.' : '',
    entries: history.revisions.map(describeRevisionEntry),
  };
}

export function renderRevisionHistory(model: RevisionHistoryModel): TemplateResult {
  return html`
    <section aria-label="Revision history">
      <p role="status">${model.stateText}</p>
      ${model.truncatedNote ? html`<p>${model.truncatedNote}</p>` : ''}
      ${model.entries.map(
        (entry) => html`<article>
          <h4>${entry.headerText}</h4>
          <p>${entry.lineageText}</p>
          <p>${entry.tasksText}</p>
          ${
            entry.taskLines.length === 0
              ? ''
              : html`<ul>
                  ${entry.taskLines.map((line) => html`<li>${line}</li>`)}
                </ul>`
          }
          <p>${entry.ideasText}</p>
          ${
            entry.ideaLines.length === 0
              ? ''
              : html`<ul>
                  ${entry.ideaLines.map((line) => html`<li>${line}</li>`)}
                </ul>`
          }
          ${entry.scopeLines.map((line) => html`<p>${line}</p>`)}
          ${
            entry.workLines.length === 0
              ? html`<p>No mapped work for this revision.</p>`
              : html`<ul>
                  ${entry.workLines.map((line) => html`<li>${line}</li>`)}
                </ul>`
          }
        </article>`
      )}
    </section>
  `;
}
