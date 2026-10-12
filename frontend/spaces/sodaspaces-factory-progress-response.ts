import {check, id, object, projectId} from './sodaspaces-api.js';
import {decisionID} from './sodaspaces-intent-response.js';
import {issueBlocker} from './sodaspaces-factory-work-response.js';
import type {IssueBlocker} from './sodaspaces-factory-work-response.js';

// Parsers for the H-R request-overview (F01), issue-plan (F03),
// repository-access (F06) and scope-revision (F04) reads plus H-A
// allowance and L06 lifecycle shapes. Each parser binds its exact
// intent, issue or repository identity; absent records leave the view
// without a verdict, never a guessed state.

const decimalID = (value: unknown): value is string =>
  typeof value === 'string' && /^(0|[1-9][0-9]{0,18})$/.test(value) && BigInt(value) <= 9223372036854775807n;
const digest = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);

function text(value: unknown, limit: number): string {
  check(typeof value === 'string' && value.length > 0 && value.length <= limit);
  return value;
}
function counter(value: unknown): number {
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= 0);
  return value;
}
function fraction(value: unknown): number {
  check(typeof value === 'number' && !Number.isNaN(value) && value >= 0 && value <= 1);
  return value;
}
function optionalLabel(value: unknown, limit: number): string | undefined {
  if (value === undefined || value === '') return undefined;
  return text(value, limit);
}
function judgmentNumbers(raw: Record<string, unknown>) {
  return {
    ...(typeof raw.confidence === 'number' ? {confidence: fraction(raw.confidence)} : {}),
    ...(typeof raw.input_tokens === 'number' ? {input_tokens: counter(raw.input_tokens)} : {}),
    ...(typeof raw.output_tokens === 'number' ? {output_tokens: counter(raw.output_tokens)} : {}),
  };
}
function countMap(value: unknown, allowed: readonly string[]): Record<string, number> {
  const raw = object(value);
  const keys = Object.keys(raw);
  check(keys.length <= allowed.length);
  const out: Record<string, number> = {};
  for (const key of keys) {
    check(allowed.includes(key));
    out[key] = counter(raw[key]);
  }
  return out;
}
function probabilityMap(value: unknown): Record<string, number> {
  const raw = object(value);
  const keys = Object.keys(raw);
  check(keys.length <= 256);
  const out: Record<string, number> = {};
  for (const key of keys) {
    check(key.length >= 1 && key.length <= 128);
    out[key] = fraction(raw[key]);
  }
  return out;
}

export interface RequestDirection {
  readonly objective: string;
  readonly ideas_by_decision: Record<string, number>;
  readonly ideas_by_source: Record<string, number>;
  readonly effective_outcomes: readonly string[];
  readonly ideas_total: number;
  readonly tasks_total: number;
  readonly unresolved_assumptions: number;
  readonly approach?: string;
}
export interface RequestContext {
  readonly repository: string;
  readonly department?: string;
}
export interface RequestMapping {
  readonly task_key: string;
  readonly issue_id: string;
  readonly issue_index: string;
  readonly acceptance_id: string;
  readonly created_unix: number;
  readonly effective: boolean;
}
export interface RequestCounts {
  readonly delivered: number;
  readonly working: number;
  readonly waiting: number;
  readonly attention: number;
  readonly total: number;
  readonly outcomes_total: number;
  readonly outcomes_evidenced: number;
}
export interface RequestAttention {
  readonly code: string;
  readonly issue?: string;
  readonly idea?: string;
}
export interface RequestOverview {
  readonly direction: RequestDirection;
  readonly context: RequestContext;
  readonly generation: string;
  readonly setup: {state: string; project?: string};
  readonly work_items: RequestCounts;
  readonly mappings: readonly RequestMapping[];
  readonly attention: readonly RequestAttention[];
  readonly intent: string;
  readonly revision: string;
  readonly state: string;
  readonly observed_unix: number;
}

const intentStates = ['draft', 'accepted', 'declined', 'rejected', 'abandoned'];
const attentionCodes = ['generation_refused', 'work_attention', 'unresolved_assumption'];
const setupStates = ['repository_unassigned', 'project_missing', 'preparing', 'ready'];
const decisionKinds = ['pending', 'accept', 'decline', 'edit'];
const sourceKinds = ['requested', 'supported', 'autonomous', 'assumption', 'unclassified'];

function requestDirection(value: unknown): RequestDirection {
  const raw = object(value);
  check(Array.isArray(raw.effective_outcomes) && raw.effective_outcomes.length <= 64);
  for (const outcome of raw.effective_outcomes) check(decisionID(outcome));
  if (raw.approach !== undefined) check(raw.approach === 'incremental' || raw.approach === 'complete_first');
  return {
    objective: text(raw.objective, 4096),
    ideas_by_decision: countMap(raw.ideas_by_decision, decisionKinds),
    ideas_by_source: countMap(raw.ideas_by_source, sourceKinds),
    effective_outcomes: raw.effective_outcomes as string[],
    ideas_total: counter(raw.ideas_total),
    tasks_total: counter(raw.tasks_total),
    unresolved_assumptions: counter(raw.unresolved_assumptions),
    ...(raw.approach === undefined ? {} : {approach: raw.approach as string}),
  };
}
function requestContext(value: unknown): RequestContext {
  const raw = object(value);
  check(decimalID(raw.repository));
  if (raw.department !== undefined) check(decisionID(raw.department));
  return {
    repository: raw.repository,
    ...(raw.department === undefined ? {} : {department: raw.department as string}),
  };
}
function requestMapping(value: unknown): RequestMapping {
  const raw = object(value);
  check(decisionID(raw.task_key) && decimalID(raw.issue_id) && id(raw.issue_index));
  check(typeof raw.effective === 'boolean');
  return {
    task_key: raw.task_key,
    issue_id: raw.issue_id,
    issue_index: raw.issue_index,
    acceptance_id: text(raw.acceptance_id, 128),
    created_unix: counter(raw.created_unix),
    effective: raw.effective,
  };
}
function requestCounts(value: unknown): RequestCounts {
  const raw = object(value);
  const by = countMap(raw.by_state, ['delivered', 'working', 'waiting', 'attention']);
  check(
    by.delivered !== undefined && by.working !== undefined && by.waiting !== undefined && by.attention !== undefined
  );
  return {
    delivered: by.delivered as number,
    working: by.working as number,
    waiting: by.waiting as number,
    attention: by.attention as number,
    total: counter(raw.total),
    outcomes_total: counter(raw.outcomes_total),
    outcomes_evidenced: counter(raw.outcomes_evidenced),
  };
}
function requestAttention(value: unknown): RequestAttention {
  const raw = object(value);
  check(typeof raw.code === 'string' && attentionCodes.includes(raw.code));
  if (raw.issue !== undefined) check(id(raw.issue));
  if (raw.idea !== undefined) check(decisionID(raw.idea));
  return {
    code: raw.code,
    ...(raw.issue === undefined ? {} : {issue: raw.issue as string}),
    ...(raw.idea === undefined ? {} : {idea: raw.idea as string}),
  };
}
function requestSetup(value: unknown): {state: string; project?: string} {
  const raw = object(value);
  check(typeof raw.state === 'string' && setupStates.includes(raw.state));
  if (raw.project !== undefined) check(projectId(raw.project));
  return {state: raw.state, ...(raw.project === undefined ? {} : {project: raw.project as string})};
}
// requestOverview binds one owner-private request overview to its exact
// intent. Generation states pass through verbatim; counts are evidenced
// work-item facts, never percentages.
export function requestOverview(value: unknown, intent: string): RequestOverview {
  const data = object(value);
  check(data.intent === intent && decisionID(data.revision));
  check(typeof data.state === 'string' && intentStates.includes(data.state));
  check(Array.isArray(data.mappings) && data.mappings.length <= 64);
  check(Array.isArray(data.attention) && data.attention.length <= 64);
  return {
    direction: requestDirection(data.direction),
    context: requestContext(data.context),
    generation: generationState(data.generation),
    setup: requestSetup(data.setup),
    work_items: requestCounts(data.work_items),
    mappings: data.mappings.map(requestMapping),
    attention: data.attention.map(requestAttention),
    intent,
    revision: data.revision,
    state: data.state,
    observed_unix: counter(data.observed_unix),
  };
}

function generationState(value: unknown): string {
  if (typeof value === 'string') return text(value, 64);
  const raw = object(value);
  return text(raw.state, 64);
}

export interface PlanCandidate {
  readonly id: string;
  readonly repository: string;
  readonly issue: string;
  readonly readiness: string;
  readonly first_seen_unix: number;
  readonly acceptance?: string;
}
export interface PlanJudgment {
  readonly probability_count: number;
  readonly probabilities?: Record<string, number>;
  readonly selected?: string;
  readonly model?: string;
  readonly confidence?: number;
  readonly input_tokens?: number;
  readonly output_tokens?: number;
}
export interface RecordedPlan {
  readonly candidate: PlanCandidate;
  readonly missing: readonly IssueBlocker[];
  readonly steps: readonly string[];
  readonly strategy: string;
  readonly snapshot: string;
  readonly outcome: string;
  readonly created_unix: number;
  readonly judgment?: PlanJudgment;
}
export interface PlanFreshness {
  readonly state: string;
  readonly provisional: boolean;
  readonly reasons: readonly string[];
}
export interface PriorityCandidate {
  readonly id: string;
  readonly repository: string;
  readonly issue: string;
  readonly readiness: string;
  readonly first_seen_unix: number;
  readonly assessed_unix: number;
  readonly age_seconds: number;
  readonly impact_known: boolean;
  readonly risk_known: boolean;
  readonly effort_known: boolean;
  readonly reason?: string;
  readonly acceptance?: string;
  readonly blocker_codes: readonly string[];
}
export interface PriorityCall {
  readonly selected?: string;
  readonly model?: string;
  readonly failure?: string;
  readonly confidence?: number;
  readonly input_tokens?: number;
  readonly output_tokens?: number;
}
export interface PriorityComparison {
  readonly candidates: readonly PriorityCandidate[];
  readonly calls: readonly PriorityCall[];
  readonly snapshot: string;
  readonly outcome: string;
  readonly created_unix: number;
  readonly candidates_total: number;
  readonly other_count: number;
  readonly probability_count: number;
  readonly selected_visible: boolean;
  readonly probabilities_full: boolean;
  readonly probabilities?: Record<string, number>;
  readonly selected?: string;
}
export interface IssuePlan {
  readonly freshness: PlanFreshness;
  readonly comparisons: readonly PriorityComparison[];
  readonly repository: string;
  readonly issue: string;
  readonly observed_unix: number;
  readonly plan?: RecordedPlan;
  readonly next_before_unix?: number;
  readonly next_before_snapshot?: string;
}

const readinessStates = [
  'not_authorized',
  'blocked',
  'queued',
  'active',
  'needs_intervention',
  'cancelled',
  'completed',
];
const freshnessStates = ['missing', 'stale', 'provisional', 'current'];
const freshnessReasons = ['current_readiness_missing', 'acceptance_changed', 'inputs_changed'];

function planCandidate(value: unknown): PlanCandidate {
  const raw = object(value);
  check(id(raw.repository) && id(raw.issue));
  check(typeof raw.readiness === 'string' && readinessStates.includes(raw.readiness));
  if (raw.acceptance !== undefined) check(typeof raw.acceptance === 'string' && raw.acceptance.length <= 256);
  return {
    id: text(raw.id, 128),
    repository: raw.repository,
    issue: raw.issue,
    readiness: raw.readiness,
    first_seen_unix: counter(raw.first_seen_unix),
    ...(typeof raw.acceptance === 'string' && raw.acceptance !== '' ? {acceptance: raw.acceptance} : {}),
  };
}
function planJudgment(value: unknown): PlanJudgment | undefined {
  if (value === undefined) return undefined;
  const raw = object(value);
  const probabilities = raw.probabilities === undefined ? undefined : probabilityMap(raw.probabilities);
  const selected = optionalLabel(raw.selected, 128);
  const model = optionalLabel(raw.model, 128);
  return {
    probability_count: counter(raw.probability_count),
    ...(probabilities === undefined ? {} : {probabilities}),
    ...(selected === undefined ? {} : {selected}),
    ...(model === undefined ? {} : {model}),
    ...judgmentNumbers(raw),
  };
}
function recordedPlan(value: unknown): RecordedPlan {
  const raw = object(value);
  check(Array.isArray(raw.missing) && raw.missing.length <= 64);
  check(Array.isArray(raw.steps) && raw.steps.length <= 32);
  for (const step of raw.steps) check(typeof step === 'string' && step.length <= 256);
  check(digest(raw.snapshot));
  const judgment = planJudgment(raw.judgment);
  return {
    candidate: planCandidate(raw.candidate),
    missing: raw.missing.map(issueBlocker),
    steps: raw.steps as string[],
    strategy: text(raw.strategy, 128),
    snapshot: raw.snapshot,
    outcome: text(raw.outcome, 64),
    created_unix: counter(raw.created_unix),
    ...(judgment === undefined ? {} : {judgment}),
  };
}
function planFreshness(value: unknown): PlanFreshness {
  const raw = object(value);
  check(typeof raw.state === 'string' && freshnessStates.includes(raw.state));
  check(typeof raw.provisional === 'boolean');
  const reasons = raw.reasons === undefined ? [] : raw.reasons;
  check(Array.isArray(reasons) && reasons.length <= 3);
  for (const reason of reasons) check(freshnessReasons.includes(reason as string));
  return {state: raw.state, provisional: raw.provisional, reasons: reasons as string[]};
}
function priorityCandidate(value: unknown): PriorityCandidate {
  const raw = object(value);
  check(id(raw.repository) && id(raw.issue));
  check(typeof raw.readiness === 'string' && readinessStates.includes(raw.readiness));
  const reason = optionalLabel(raw.reason, 256);
  const acceptance = optionalLabel(raw.acceptance, 256);
  return {
    id: text(raw.id, 128),
    repository: raw.repository,
    issue: raw.issue,
    readiness: raw.readiness,
    first_seen_unix: counter(raw.first_seen_unix),
    assessed_unix: counter(raw.assessed_unix),
    age_seconds: counter(raw.age_seconds),
    ...candidateFlags(raw),
    ...(reason === undefined ? {} : {reason}),
    ...(acceptance === undefined ? {} : {acceptance}),
    blocker_codes: blockerCodes(raw.blocker_codes),
  };
}
function candidateFlags(raw: Record<string, unknown>) {
  check(
    typeof raw.impact_known === 'boolean' &&
      typeof raw.risk_known === 'boolean' &&
      typeof raw.effort_known === 'boolean'
  );
  return {impact_known: raw.impact_known, risk_known: raw.risk_known, effort_known: raw.effort_known};
}
function blockerCodes(value: unknown): string[] {
  if (value === undefined) return [];
  check(Array.isArray(value) && value.length <= 16);
  for (const code of value) check(typeof code === 'string' && /^[a-z_]{1,64}$/.test(code));
  return value as string[];
}
function priorityCall(value: unknown): PriorityCall {
  const raw = object(value);
  const selected = optionalLabel(raw.selected, 128);
  const model = optionalLabel(raw.model, 128);
  const failure = optionalLabel(raw.failure, 256);
  return {
    ...(selected === undefined ? {} : {selected}),
    ...(model === undefined ? {} : {model}),
    ...(failure === undefined ? {} : {failure}),
    ...judgmentNumbers(raw),
  };
}
function priorityComparison(value: unknown): PriorityComparison {
  const raw = object(value);
  check(Array.isArray(raw.candidates) && raw.candidates.length <= 64);
  check(Array.isArray(raw.calls) && raw.calls.length <= 16);
  check(digest(raw.snapshot));
  check(typeof raw.selected_visible === 'boolean' && typeof raw.probabilities_full === 'boolean');
  const probabilities = raw.probabilities === undefined ? undefined : probabilityMap(raw.probabilities);
  return {
    candidates: raw.candidates.map(priorityCandidate),
    calls: raw.calls.map(priorityCall),
    snapshot: raw.snapshot,
    outcome: text(raw.outcome, 64),
    created_unix: counter(raw.created_unix),
    candidates_total: counter(raw.candidates_total),
    other_count: counter(raw.other_count),
    probability_count: counter(raw.probability_count),
    selected_visible: raw.selected_visible,
    probabilities_full: raw.probabilities_full,
    ...(probabilities === undefined ? {} : {probabilities}),
    ...(typeof raw.selected === 'string' && raw.selected !== '' ? {selected: raw.selected} : {}),
  };
}
// issuePlan binds one issue's recorded plan, freshness and recent
// priority comparisons to its exact repository and issue. Paging
// cursors travel as a pair; a half cursor is rejected.
export interface PlanningAllowanceView {
  readonly id: string;
  readonly intent: string;
  readonly owner: string;
  readonly amount: number;
  readonly unit: string;
  readonly reserved: number;
  readonly consumed: number;
  readonly ceiling: number;
  readonly created_unix: number;
  readonly predecessor?: string;
  readonly effect?: string;
}
function allowanceAmount(raw: Record<string, unknown>) {
  check(typeof raw.amount === 'number' && Number.isSafeInteger(raw.amount) && raw.amount >= 0 && raw.amount <= 100);
  check(typeof raw.ceiling === 'number' && Number.isSafeInteger(raw.ceiling) && raw.ceiling >= 0 && raw.ceiling <= 100);
  check(raw.amount <= raw.ceiling);
  return {amount: raw.amount, ceiling: raw.ceiling};
}
function allowanceConsumption(raw: Record<string, unknown>) {
  check(typeof raw.reserved === 'number' && Number.isSafeInteger(raw.reserved) && raw.reserved >= 0);
  check(typeof raw.consumed === 'number' && Number.isSafeInteger(raw.consumed) && raw.consumed >= 0);
  return {reserved: raw.reserved, consumed: raw.consumed};
}
function allowanceBounds(raw: Record<string, unknown>) {
  const amount = allowanceAmount(raw);
  const consumption = allowanceConsumption(raw);
  check(consumption.reserved + consumption.consumed <= amount.amount);
  return {...amount, ...consumption};
}
function allowanceEffect(value: unknown): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(typeof value === 'string' && value.trim() !== '' && value.length <= 4000);
  return value;
}
// planningAllowance binds one planning allowance record to its exact
// intent: the bounded amount in real planning-call units with reserved
// and consumed consumption.
export function planningAllowance(value: unknown, intent: string): PlanningAllowanceView {
  const data = object(value);
  check(decisionID(data.id) && data.intent === intent && id(data.owner));
  check(data.unit === 'planning_calls');
  check(typeof data.created_unix === 'number' && data.created_unix > 0);
  if (data.predecessor !== undefined && data.predecessor !== '') check(decisionID(data.predecessor));
  const bounds = allowanceBounds(data);
  const effect = allowanceEffect(data.effect);
  return {
    id: data.id,
    intent,
    owner: data.owner,
    ...bounds,
    unit: 'planning_calls',
    created_unix: data.created_unix,
    ...(typeof data.predecessor === 'string' && data.predecessor !== '' ? {predecessor: data.predecessor} : {}),
    ...(effect === undefined ? {} : {effect}),
  };
}

export interface LifecycleEffectConfig {
  readonly event: string;
  readonly revision: string;
  readonly enabled: boolean;
  readonly capable: boolean;
  readonly actor_id: string;
  readonly repository_id: string;
  readonly reviewer_ids: readonly string[];
  readonly label_id?: string;
  readonly assignee_id?: string;
  readonly note?: string;
}
export interface LifecycleEffectAction {
  readonly id: string;
  readonly event: string;
  readonly action: string;
  readonly actor_id: string;
  readonly issue_id: string;
  readonly attempts: number;
  readonly max_attempts: number;
  readonly revoked: boolean;
  readonly settled: boolean;
  readonly results: Record<string, string>;
}

const lifecycleEvents = ['issue_created', 'pr_created', 'verified_merge'];
const lifecycleActions: Record<string, string> = {
  issue_created: 'issue_label_assignee',
  pr_created: 'pr_reviewer_request',
  verified_merge: 'original_completion_note',
};
const lifecycleResults = ['pending', 'delivered', 'failed', 'uncertain', 'denied'];

function lifecycleNote(value: unknown): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(typeof value === 'string' && [...value].length <= 1000);
  return value;
}
function lifecycleActor(value: unknown): string {
  check(typeof value === 'string' && (value === '' || decimalID(value)));
  return value;
}
function lifecycleReviewers(value: unknown): string[] {
  if (value === undefined) return [];
  check(Array.isArray(value) && value.length <= 3);
  for (const reviewer of value) check(id(reviewer));
  return value as string[];
}
// lifecycleEffectConfig binds one event's optional-effect
// configuration to its exact event and repository: enablement,
// capability proof, actor, targets and revision.
export function lifecycleEffectConfig(value: unknown, event: string, repository: string): LifecycleEffectConfig {
  const data = object(value);
  check(data.event === event && lifecycleEvents.includes(event));
  check(decimalID(data.revision));
  check(typeof data.enabled === 'boolean' && typeof data.capable === 'boolean');
  check(data.repository_id === repository);
  if (data.label_id !== undefined) check(id(data.label_id));
  if (data.assignee_id !== undefined) check(id(data.assignee_id));
  const note = lifecycleNote(data.note);
  return {
    event,
    revision: data.revision,
    enabled: data.enabled,
    capable: data.capable,
    actor_id: lifecycleActor(data.actor_id),
    repository_id: repository,
    reviewer_ids: lifecycleReviewers(data.reviewer_ids),
    ...(data.label_id === undefined ? {} : {label_id: data.label_id as string}),
    ...(data.assignee_id === undefined ? {} : {assignee_id: data.assignee_id as string}),
    ...(note === undefined ? {} : {note}),
  };
}
function lifecycleResultsMap(value: unknown): Record<string, string> {
  if (value === undefined) return {};
  const raw = object(value);
  const keys = Object.keys(raw);
  check(keys.length <= 8);
  const out: Record<string, string> = {};
  for (const key of keys) {
    check(key.length >= 1 && key.length <= 128);
    check(typeof raw[key] === 'string' && lifecycleResults.includes(raw[key] as string));
    out[key] = raw[key] as string;
  }
  return out;
}
function lifecycleAction(value: unknown, issue: string, event: string): LifecycleEffectAction {
  const raw = object(value);
  check(typeof raw.id === 'string' && raw.id.length >= 1 && raw.id.length <= 128);
  check(raw.event === event && lifecycleEvents.includes(event));
  check(typeof raw.action === 'string' && raw.action === lifecycleActions[event]);
  check(raw.issue_id === issue);
  check(typeof raw.attempts === 'number' && Number.isSafeInteger(raw.attempts) && raw.attempts >= 0);
  // Unclaimed rows (no attempt yet) carry actor_id "0": the drain
  // claims the enrolled submitter before the first attempt. Claimed
  // rows always carry the enrolled actor.
  const actor = raw.actor_id;
  if (raw.attempts === 0) check(decimalID(actor));
  else check(id(actor));
  check(typeof raw.revoked === 'boolean' && typeof raw.settled === 'boolean');
  return {
    id: raw.id,
    event,
    action: raw.action,
    actor_id: actor,
    issue_id: issue,
    attempts: raw.attempts,
    max_attempts: counter(raw.max_attempts),
    revoked: raw.revoked,
    settled: raw.settled,
    results: lifecycleResultsMap(raw.results),
  };
}
// lifecycleEffectActions binds one issue/event's durable action results
// to the exact issue and event. Attempts stay within the recorded
// budget; revoked and unsettled actions read distinctly.
export function lifecycleEffectActions(value: unknown, issue: string, event: string): LifecycleEffectAction[] {
  const data = object(value);
  check(Array.isArray(data.actions) && data.actions.length <= 64);
  const actions = data.actions.map((entry) => lifecycleAction(entry, issue, event));
  for (const action of actions) check(action.attempts <= action.max_attempts);
  return actions;
}

function planCursor(data: Record<string, unknown>): {next_before_unix?: number; next_before_snapshot?: string} {
  if (data.next_before_unix === undefined && data.next_before_snapshot === undefined) return {};
  check(typeof data.next_before_unix === 'number' && Number.isSafeInteger(data.next_before_unix));
  check(data.next_before_unix > 0 && digest(data.next_before_snapshot));
  return {next_before_unix: data.next_before_unix, next_before_snapshot: data.next_before_snapshot};
}
// issuePlan binds one issue's recorded plan, freshness and recent
// priority comparisons to its exact repository and issue. Paging
// cursors travel as a pair; a half cursor is rejected.
export function issuePlan(value: unknown, repository: string, issue: string): IssuePlan {
  const data = object(value);
  check(data.repository === repository && data.issue === issue);
  check(Array.isArray(data.comparisons) && data.comparisons.length <= 20);
  const plan = data.plan === undefined ? undefined : recordedPlan(data.plan);
  return {
    freshness: planFreshness(data.freshness),
    comparisons: data.comparisons.map(priorityComparison),
    repository,
    issue,
    observed_unix: counter(data.observed_unix),
    ...(plan === undefined ? {} : {plan}),
    ...planCursor(data),
  };
}

export interface AccessViewer {
  readonly code_write: boolean;
  readonly owner_or_admin: boolean;
  readonly environment_owner: boolean;
  readonly operator: boolean;
  readonly member: boolean;
  readonly session_current: boolean;
}
export interface AccessEntry {
  readonly editable: boolean;
  readonly reason?: string;
}
export interface AccessRequiredAction {
  readonly code: string;
  readonly resolve_by: string;
  readonly setting: string;
}
export interface AccessEffective {
  readonly effective: boolean;
  readonly dispatch_open: boolean;
  readonly missing: readonly string[];
}
export interface AccessPlanningEntry {
  readonly intent: string;
  readonly id: string;
  readonly unit: 'planning_calls';
  readonly status: 'ok' | 'exhausted';
  readonly amount: number;
  readonly reserved: number;
  readonly consumed: number;
  readonly available: number;
  readonly ceiling: number;
  readonly editable: AccessEntry;
  readonly effect?: string;
}
export interface FactoryAccess {
  readonly repository: string;
  readonly observed_unix: number;
  readonly viewer: AccessViewer;
  readonly effective: AccessEffective;
  readonly required_actions: readonly AccessRequiredAction[];
  readonly planning: readonly AccessPlanningEntry[];
  readonly planning_truncated: boolean;
}

function accessViewer(value: unknown): AccessViewer {
  const raw = object(value);
  for (const key of ['code_write', 'owner_or_admin', 'environment_owner', 'operator', 'member', 'session_current'])
    check(typeof raw[key] === 'boolean');
  return {
    code_write: raw.code_write as boolean,
    owner_or_admin: raw.owner_or_admin as boolean,
    environment_owner: raw.environment_owner as boolean,
    operator: raw.operator as boolean,
    member: raw.member as boolean,
    session_current: raw.session_current as boolean,
  };
}
function accessEntry(value: unknown): AccessEntry {
  const raw = object(value);
  check(typeof raw.editable === 'boolean');
  const reason = raw.reason === undefined || raw.reason === '' ? undefined : text(raw.reason, 64);
  return {editable: raw.editable, ...(reason === undefined ? {} : {reason})};
}
function accessRequiredAction(value: unknown): AccessRequiredAction {
  const raw = object(value);
  check(typeof raw.code === 'string' && raw.code.length >= 1 && raw.code.length <= 128);
  check(typeof raw.resolve_by === 'string' && raw.resolve_by.length <= 128);
  check(typeof raw.setting === 'string' && raw.setting.length <= 128);
  return {code: raw.code, resolve_by: raw.resolve_by, setting: raw.setting};
}
function accessEffective(value: unknown): AccessEffective {
  const raw = object(value);
  check(typeof raw.effective === 'boolean' && typeof raw.dispatch_open === 'boolean');
  check(Array.isArray(raw.missing) && raw.missing.length <= 32);
  for (const code of raw.missing) check(typeof code === 'string' && code.length >= 1 && code.length <= 128);
  return {effective: raw.effective, dispatch_open: raw.dispatch_open, missing: raw.missing as string[]};
}
function accessPlanningEntry(value: unknown): AccessPlanningEntry {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.id));
  check(raw.unit === 'planning_calls');
  const bounds = allowanceBounds(raw);
  check(typeof raw.available === 'number' && Number.isSafeInteger(raw.available) && raw.available >= 0);
  const available = Math.max(0, bounds.amount - bounds.reserved - bounds.consumed);
  check(raw.available === available);
  check(raw.status === 'ok' || raw.status === 'exhausted');
  check(raw.status === (available <= 0 ? 'exhausted' : 'ok'));
  const effect = allowanceEffect(raw.effect);
  return {
    intent: raw.intent,
    id: raw.id,
    unit: 'planning_calls',
    status: raw.status,
    amount: bounds.amount,
    reserved: bounds.reserved,
    consumed: bounds.consumed,
    available,
    ceiling: bounds.ceiling,
    editable: accessEntry(raw.editable),
    ...(effect === undefined ? {} : {effect}),
  };
}
// factoryAccess binds one repository's permission, authority and
// planning-allowance listing to its exact repository. Only the
// sections V consumes are validated: viewer standing, the effective
// verdict with required actions, and the bounded planning listing.
export function factoryAccess(value: unknown, repository: string): FactoryAccess {
  const data = object(value);
  check(data.repository === repository && decimalID(repository));
  check(Array.isArray(data.required_actions) && data.required_actions.length <= 32);
  check(Array.isArray(data.planning) && data.planning.length <= 20);
  const truncated = data.planning_truncated === undefined ? false : data.planning_truncated;
  check(typeof truncated === 'boolean');
  return {
    repository,
    observed_unix: counter(data.observed_unix),
    viewer: accessViewer(data.viewer),
    effective: accessEffective(data.effective),
    required_actions: data.required_actions.map(accessRequiredAction),
    planning: data.planning.map(accessPlanningEntry),
    planning_truncated: truncated,
  };
}

export interface RevisionTask {
  readonly key: string;
  readonly objective: string;
  readonly outcomes: readonly string[];
  readonly idea_ids: readonly string[];
  readonly depends_on: readonly string[];
}
export interface RevisionTaskChange {
  readonly key: string;
  readonly changed_fields: readonly string[];
  readonly old_objective?: string;
  readonly new_objective?: string;
  readonly old_outcomes?: readonly string[];
  readonly new_outcomes?: readonly string[];
}
export interface RevisionTaskDiff {
  readonly added: readonly RevisionTask[];
  readonly removed: readonly RevisionTask[];
  readonly changed: readonly RevisionTaskChange[];
  readonly reused: readonly string[];
  readonly unchanged_count: number;
}
export interface RevisionIdeaBody {
  readonly id: string;
  readonly description: string;
  readonly decision: string;
}
export interface RevisionIdeaChange {
  readonly id: string;
  readonly from_decision: string;
  readonly to_decision: string;
  readonly correction_changed: boolean;
  readonly content_changed: boolean;
}
export interface RevisionIdeaDiff {
  readonly added: readonly RevisionIdeaBody[];
  readonly removed: readonly RevisionIdeaBody[];
  readonly changed: readonly RevisionIdeaChange[];
}
export interface RevisionScalarChange {
  readonly from: string;
  readonly to: string;
}
export interface RevisionComparison {
  readonly tasks: RevisionTaskDiff;
  readonly ideas: RevisionIdeaDiff;
  readonly effective_added: readonly string[];
  readonly effective_removed: readonly string[];
  readonly repository_change?: RevisionScalarChange;
  readonly state_change?: RevisionScalarChange;
  readonly initial: boolean;
  readonly edges_changed: boolean;
}
export interface RevisionWork {
  readonly task_key: string;
  readonly issue_index: string;
  readonly state: string;
  readonly authority: string;
  readonly content_version: number;
}
export interface RevisionGeneration {
  readonly state: string;
  readonly submitted_keys: readonly string[];
  readonly submitted_differs: boolean;
}
export interface RevisionEntry {
  readonly revision: string;
  readonly state: string;
  readonly repository: string;
  readonly created_unix: number;
  readonly generation: RevisionGeneration;
  readonly comparison: RevisionComparison;
  readonly work: readonly RevisionWork[];
  readonly generated_from?: string;
}
export interface RevisionHistory {
  readonly intent: string;
  readonly observed_unix: number;
  readonly revisions: readonly RevisionEntry[];
  readonly truncated: boolean;
}

const revisionStates: readonly unknown[] = ['draft', 'accepted', 'declined'];
const revisionTaskFields: readonly unknown[] = ['objective', 'outcomes', 'idea_ids', 'depends_on'];
const revisionDecisions: readonly unknown[] = ['pending', 'accept', 'decline', 'edit'];
const revisionGeneration: readonly unknown[] = ['pending', 'committed', 'refused', 'completed', 'not_requested'];
const revisionWorkStates: readonly unknown[] = ['delivered', 'working', 'waiting', 'attention'];
const revisionAuthority: readonly unknown[] = ['current', 'superseded', 'unknown'];

function revisionOutcomes(value: unknown): string[] {
  check(Array.isArray(value) && value.length >= 1 && value.length <= 20);
  return value.map((outcome) => text(outcome, 4000));
}
function revisionIDList(value: unknown, limit: number, min = 0): string[] {
  check(Array.isArray(value) && value.length >= min && value.length <= limit);
  for (const entry of value) check(decisionID(entry));
  return value as string[];
}
function revisionTask(value: unknown): RevisionTask {
  const raw = object(value);
  check(decisionID(raw.key));
  const depends = raw.depends_on === undefined ? [] : revisionIDList(raw.depends_on, 64);
  return {
    key: raw.key,
    objective: text(raw.objective, 4000),
    outcomes: revisionOutcomes(raw.outcomes),
    idea_ids: revisionIDList(raw.idea_ids, 50, 1),
    depends_on: depends,
  };
}
function revisionChangedTexts(
  raw: Record<string, unknown>,
  fields: readonly string[],
  field: string
): {old?: string; current?: string} {
  const oldKey = `old_${field}`;
  const newKey = `new_${field}`;
  if (!fields.includes(field)) {
    check(raw[oldKey] === undefined && raw[newKey] === undefined);
    return {};
  }
  return {old: text(raw[oldKey], 4000), current: text(raw[newKey], 4000)};
}
function revisionChangedOutcomes(
  raw: Record<string, unknown>,
  fields: readonly string[]
): {old?: string[]; current?: string[]} {
  if (!fields.includes('outcomes')) {
    check(raw.old_outcomes === undefined && raw.new_outcomes === undefined);
    return {};
  }
  return {old: revisionOutcomes(raw.old_outcomes), current: revisionOutcomes(raw.new_outcomes)};
}
function revisionTaskChange(value: unknown): RevisionTaskChange {
  const raw = object(value);
  check(decisionID(raw.key));
  check(Array.isArray(raw.changed_fields) && raw.changed_fields.length >= 1 && raw.changed_fields.length <= 4);
  for (const field of raw.changed_fields) check(revisionTaskFields.includes(field));
  const fields = raw.changed_fields as string[];
  const objective = revisionChangedTexts(raw, fields, 'objective');
  const outcomes = revisionChangedOutcomes(raw, fields);
  return {
    key: raw.key,
    changed_fields: fields,
    ...(objective.old === undefined ? {} : {old_objective: objective.old, new_objective: objective.current ?? ''}),
    ...(outcomes.old === undefined ? {} : {old_outcomes: outcomes.old, new_outcomes: outcomes.current ?? []}),
  };
}
function revisionTaskList(value: unknown): RevisionTask[] {
  check(Array.isArray(value) && value.length <= 64);
  return value.map(revisionTask);
}
function revisionTaskDiff(value: unknown): RevisionTaskDiff {
  const raw = object(value);
  check(Array.isArray(raw.changed) && raw.changed.length <= 64);
  const reused = revisionIDList(raw.reused, 64);
  const unchanged = counter(raw.unchanged_count);
  check(unchanged === reused.length);
  return {
    added: revisionTaskList(raw.added),
    removed: revisionTaskList(raw.removed),
    changed: raw.changed.map(revisionTaskChange),
    reused,
    unchanged_count: unchanged,
  };
}
function revisionIdeaBody(value: unknown): RevisionIdeaBody {
  const raw = object(value);
  check(decisionID(raw.id));
  check(revisionDecisions.includes(raw.decision));
  return {id: raw.id, description: text(raw.description, 4000), decision: raw.decision as string};
}
function revisionIdeaBodyList(value: unknown): RevisionIdeaBody[] {
  check(Array.isArray(value) && value.length <= 50);
  return value.map(revisionIdeaBody);
}
function revisionIdeaChange(value: unknown): RevisionIdeaChange {
  const raw = object(value);
  check(decisionID(raw.id));
  check(revisionDecisions.includes(raw.from_decision) && revisionDecisions.includes(raw.to_decision));
  check(typeof raw.correction_changed === 'boolean' && typeof raw.content_changed === 'boolean');
  return {
    id: raw.id,
    from_decision: raw.from_decision as string,
    to_decision: raw.to_decision as string,
    correction_changed: raw.correction_changed,
    content_changed: raw.content_changed,
  };
}
function revisionIdeaDiff(value: unknown): RevisionIdeaDiff {
  const raw = object(value);
  check(Array.isArray(raw.changed) && raw.changed.length <= 50);
  return {
    added: revisionIdeaBodyList(raw.added),
    removed: revisionIdeaBodyList(raw.removed),
    changed: raw.changed.map(revisionIdeaChange),
  };
}
function revisionScalarChange(
  value: unknown,
  allowed: readonly unknown[],
  numeric: boolean
): RevisionScalarChange | undefined {
  if (value === undefined) return undefined;
  const raw = object(value);
  if (numeric) check(decimalID(raw.from) && decimalID(raw.to));
  else check(allowed.includes(raw.from) && allowed.includes(raw.to));
  return {from: raw.from as string, to: raw.to as string};
}
function revisionEffectiveDiff(value: unknown): {added: string[]; removed: string[]} {
  const raw = object(value);
  return {added: revisionIDList(raw.added, 50), removed: revisionIDList(raw.removed, 50)};
}
function checkEmptyInitial(
  tasks: RevisionTaskDiff,
  ideas: RevisionIdeaDiff,
  effective: {added: string[]; removed: string[]},
  edgesChanged: boolean
) {
  check(
    tasks.added.length === 0 && tasks.removed.length === 0 && tasks.changed.length === 0 && tasks.reused.length === 0
  );
  check(
    ideas.added.length === 0 &&
      ideas.removed.length === 0 &&
      ideas.changed.length === 0 &&
      effective.added.length === 0 &&
      effective.removed.length === 0 &&
      !edgesChanged
  );
}
function revisionComparison(value: unknown): RevisionComparison {
  const raw = object(value);
  const initial = raw.initial === undefined ? false : raw.initial;
  check(typeof initial === 'boolean');
  check(typeof raw.edges_changed === 'boolean');
  const tasks = revisionTaskDiff(raw.tasks);
  const ideas = revisionIdeaDiff(raw.ideas);
  const effective = revisionEffectiveDiff(raw.effective_outcomes);
  const repository = revisionScalarChange(raw.repository_change, [], true);
  const state = revisionScalarChange(raw.state_change, revisionStates, false);
  if (initial) checkEmptyInitial(tasks, ideas, effective, raw.edges_changed);
  return {
    tasks,
    ideas,
    effective_added: effective.added,
    effective_removed: effective.removed,
    ...(repository === undefined ? {} : {repository_change: repository}),
    ...(state === undefined ? {} : {state_change: state}),
    initial,
    edges_changed: raw.edges_changed,
  };
}
function revisionWork(value: unknown): RevisionWork {
  const raw = object(value);
  check(decisionID(raw.task_key) && id(raw.issue_index));
  check(revisionWorkStates.includes(raw.state) && revisionAuthority.includes(raw.authority));
  return {
    task_key: raw.task_key,
    issue_index: raw.issue_index,
    state: raw.state as string,
    authority: raw.authority as string,
    content_version: counter(raw.content_version),
  };
}
function entryGeneration(value: unknown): RevisionGeneration {
  const raw = object(value);
  check(revisionGeneration.includes(raw.state));
  const keys = raw.submitted_keys === undefined ? [] : revisionIDList(raw.submitted_keys, 64);
  const differs = raw.submitted_differs === undefined ? false : raw.submitted_differs;
  check(typeof differs === 'boolean');
  const state = raw.state as string;
  if (state === 'not_requested') check(keys.length === 0 && !differs);
  else check(keys.length >= 1);
  if (differs) check(keys.length >= 1);
  return {state, submitted_keys: keys, submitted_differs: differs};
}
function revisionEntry(value: unknown): RevisionEntry {
  const raw = object(value);
  check(decisionID(raw.revision) && decimalID(raw.repository));
  check(revisionStates.includes(raw.state));
  check(typeof raw.created_unix === 'number' && Number.isSafeInteger(raw.created_unix) && raw.created_unix > 0);
  const generation = entryGeneration(raw.generation);
  check(Array.isArray(raw.work) && raw.work.length <= 64);
  const from = raw.generated_from === undefined || raw.generated_from === '' ? undefined : raw.generated_from;
  if (from !== undefined) check(decisionID(from));
  return {
    revision: raw.revision,
    state: raw.state as string,
    repository: raw.repository,
    created_unix: raw.created_unix,
    generation,
    comparison: revisionComparison(raw.comparison),
    work: raw.work.map(revisionWork),
    ...(from === undefined ? {} : {generated_from: from}),
  };
}
// revisionHistory binds one request's scope-revision page to its
// exact intent: newest-first revisions with structural before/after
// and affected work. The first revision renders initial with empty
// diffs; old/new texts travel only with their changed field. The
// confirmed submitted graph stays distinct from accepted direction:
// keys travel only with a generation record, and a differs flag
// requires submitted keys.
export function revisionHistory(value: unknown, intent: string): RevisionHistory {
  const data = object(value);
  check(data.intent === intent && decisionID(intent));
  check(Array.isArray(data.revisions) && data.revisions.length >= 1 && data.revisions.length <= 5);
  const truncated = data.truncated === undefined ? false : data.truncated;
  check(typeof truncated === 'boolean');
  return {
    intent,
    observed_unix: counter(data.observed_unix),
    revisions: data.revisions.map(revisionEntry),
    truncated,
  };
}
