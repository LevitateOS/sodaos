import {html} from 'lit';
import type {TemplateResult} from 'lit';
import {check, object} from './sodaspaces-api.js';
import {decisionID} from './sodaspaces-intent-response.js';
import type {IntentRevision} from './sodaspaces-intent-response.js';
import {activityTimestamp} from './sodaspaces-factory-activity.js';
import type {FactoryIssueDetail, FactoryRepositorySummary} from './sodaspaces-factory-work-response.js';
import {describeIssueDetail, nativeIssueLink} from './sodaspaces-factory-work-view.js';
import type {DeliveryState} from './sodaspaces-factory-work-view.js';
import type {
  IssuePlan,
  PriorityComparison,
  RecordedPlan,
  RequestAttention,
  RequestOverview,
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
                (item) => html`<li>${item.text} ${item.link ? html`<a href=${item.link}>Open issue</a>` : ''}</li>`
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
    allowanceText: `Planning allowance: ${revision.planning_calls} calls accepted. Consumption record unavailable.`,
    deltaText: 'Revision comparison unavailable.',
  };
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

export type OptionalEffectKind = 'issue_label' | 'review_request' | 'completion_note';
export interface OptionalEffectModel {
  readonly kind: OptionalEffectKind;
  readonly title: string;
  readonly proposal: string;
  readonly stateText: string;
}
const effectCopy: Record<OptionalEffectKind, {title: string; proposal: string}> = {
  issue_label: {
    title: 'Generated issue label or assignee',
    proposal: 'Proposed: apply one fixed label and one eligible assignee to each generated issue.',
  },
  review_request: {
    title: 'Pull request human review',
    proposal: 'Proposed: request up to three selected eligible human reviewers on each generated PR.',
  },
  completion_note: {
    title: 'Original-issue completion note',
    proposal: 'Proposed: post one bounded completion comment after verified merge and original closure.',
  },
};
// describeOptionalEffects gates each effect on its own native
// checkpoint. Unpassed effects stay unavailable; no configuration is
// shown without its checkpoint, and none enrolls credentials.
export function describeOptionalEffects(checkpoints: Record<OptionalEffectKind, boolean>): OptionalEffectModel[] {
  return (Object.keys(effectCopy) as OptionalEffectKind[]).map((kind) => ({
    kind,
    title: effectCopy[kind]?.title ?? kind,
    proposal: effectCopy[kind]?.proposal ?? '',
    stateText: checkpoints[kind]
      ? 'Native capability checkpoint passed; configuration is not yet exposed.'
      : 'Unavailable: its native capability checkpoint has not passed.',
  }));
}

export function renderOptionalEffects(models: readonly OptionalEffectModel[]): TemplateResult {
  return html`
    <section aria-label="Optional effects">
      <p>
        Optional effects stay off unless their own native capability is verified. They never replace mandatory review.
      </p>
      <ul>
        ${models.map(
          (model) => html`<li>
            <p>${model.title}</p>
            <p>${model.proposal}</p>
            <p role="status">${model.stateText}</p>
          </li>`
        )}
      </ul>
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
                (item) => html`<li>${item.text} ${item.link ? html`<a href=${item.link}>Open issue</a>` : ''}</li>`
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
