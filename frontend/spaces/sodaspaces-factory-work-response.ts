import {check, id, object, projectId} from './sodaspaces-api.js';
import {nativePath} from './sodaspaces-factory-activity.js';

// Parsers for the Factory work-progress reads V01/V03/V04 consume
// independently: one issue's acceptance/readiness/check/merge facts, one
// repository's queue/setup/authority summary, and the durable outcomes of
// pause/resume/stop/retry/takeover commands. Each parser binds its exact
// repository/issue/run/command identity; absence of a record leaves the
// view without a verdict, never a guessed state.

const commitId = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{40}$/.test(value);
const commandId = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{32}$/.test(value);
const digest = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);

function text(value: unknown, limit: number): string {
  check(typeof value === 'string' && value.length > 0 && value.length <= limit);
  return value;
}
function counter(value: unknown): number {
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= 0);
  return value;
}
function reasonCode(value: unknown): string {
  check(typeof value === 'string' && /^[a-z_]{1,64}$/.test(value));
  return value;
}
function boundedCodes(value: unknown, limit: number): string[] {
  check(Array.isArray(value) && value.length <= limit);
  return value.map(reasonCode);
}
function boundedLines(value: unknown, limit: number, width: number): string[] {
  check(Array.isArray(value) && value.length <= limit);
  return value.map((line) => text(line, width));
}

export interface IssueAcceptanceRef {
  id: string;
  issue_index: string;
  approver: string;
  native_revision: number;
  content_version: number;
}
export interface IssueAcceptanceStatus {
  valid: boolean;
  withdrawn: boolean;
  revision: number;
  reasons: string[];
  acceptance?: IssueAcceptanceRef;
}
export interface IssueBlocker {
  code: string;
  resolution: string;
  detail?: string;
  endpoint_repository?: string;
  endpoint_issue?: string;
}
export interface IssueReadiness {
  readiness: string;
  reason: string;
  revision: number;
  assessed_unix: number;
  acceptance?: string;
  blockers: IssueBlocker[];
}
export interface IssueCheckResult {
  context: string;
  passed: boolean;
  state?: string;
}

export interface IssueChecks {
  verdict: string;
  reason: string;
  resolution: string;
  head_oid: string;
  base_oid: string;
  pr_number: string;
  pr_id: string;
  policy_revision: number;
  native_revision: number;
  revision: number;
  assessed_unix: number;
  stale: boolean;
  results: IssueCheckResult[];
}
export interface IssueMerge {
  stage: string;
  resolution: string;
  head_oid: string;
  base_oid: string;
  pr_number: string;
  pr_id: string;
  revision: number;
  stale: boolean;
  reason?: string;
  merged_commit?: string;
  finished_unix?: number;
}
export interface WorkAttempt {
  id: string;
  role: string;
  stage: string;
  run: string;
  attempts: number;
  created_unix: number;
  outcome?: string;
  reason?: string;
  finished_unix?: number;
}
export interface WorkCorrection {
  updated_unix: number;
  candidate?: string;
  effect?: string;
  completion?: string;
  reason?: string;
}
export interface WorkReview {
  run: string;
  head_oid: string;
  event: string;
  effect?: string;
  completion?: string;
  reason?: string;
}
export interface WorkPublication {
  candidate: string;
  base_sha: string;
  target_branch: string;
  id: string;
  run: string;
  publish_effect: string;
  pr_create_effect: string;
  stage: string;
  corrections_total: number;
  reviews_total: number;
  created_unix: number;
  pr_number?: string;
  pr_id?: string;
  reason?: string;
  finished_unix?: number;
  corrections: WorkCorrection[];
  reviews: WorkReview[];
}
export interface WorkClosure {
  state: string;
  occurrence?: string;
  closed_unix?: number;
}
export interface WorkCapability {
  available: boolean;
  reason?: string;
}
export interface WorkCapabilities {
  watch: WorkCapability;
  stop: WorkCapability;
  retry: WorkCapability;
  takeover: WorkCapability;
  pause: WorkCapability;
  resume: WorkCapability;
}
export interface RequestedRetry {
  command_id: string;
  reason: string;
  prior_run: string;
  queued: boolean;
}
export interface WorkLinks {
  issue_url: string;
  pull_url?: string;
}
export interface FactoryIssueDetail {
  repository: string;
  issue: string;
  status: IssueAcceptanceStatus;
  closure: WorkClosure;
  capabilities: WorkCapabilities;
  observed_unix: number;
  readiness?: IssueReadiness;
  checks?: IssueChecks;
  merge?: IssueMerge;
  attempt?: WorkAttempt;
  publication?: WorkPublication;
  requested_retry?: RequestedRetry;
  links?: WorkLinks;
}

const assignmentStages = ['assigned', 'finished'];
const publicationStages = ['open', 'published', 'failed', 'withdrawn', 'fenced'];
const opEffects = ['pending', 'committed', 'not_committed', 'indeterminate'];
const opCompletions = ['pending', 'complete', 'needs_intervention'];
const closureStates = ['none', 'pending', 'confirmed'];
const capabilityReasons = [
  'repository_write_required',
  'repository_not_visible',
  'native_authority_unavailable',
  'session_changed',
  'no_run',
  'run_unsettled',
  'run_settled',
  'retry_queued',
  'membership_required',
  'dispatch_open',
  'dispatch_closed',
];
const roleName = (value: unknown): value is string => typeof value === 'string' && /^[a-z][a-z0-9-]{0,63}$/.test(value);
const runRef = (value: unknown): value is string =>
  typeof value === 'string' && (value === '' || /^[a-f0-9]{32}$/.test(value));
const readinessStates = [
  'not_authorized',
  'blocked',
  'queued',
  'active',
  'needs_intervention',
  'cancelled',
  'completed',
];
const checkVerdicts = ['pass', 'pending', 'failed', 'refused'];
const mergeStages = ['open', 'merged', 'failed', 'withdrawn', 'fenced'];
const mergeReasons = [
  'merge_completed',
  'native_refused',
  'evidence_invalid',
  'merge_withdrawn',
  'effect_indeterminate',
  'acceptance_superseded',
  'effect_unattributed',
];
const runOutcomes = ['succeeded', 'failed', 'cancelled', 'needs-human'];

function closedSet(value: unknown, allowed: readonly string[]): string {
  check(typeof value === 'string' && allowed.includes(value));
  return value;
}
function blockerDetail(value: unknown): string {
  check(
    typeof value === 'string' && value.length <= 256 && !value.split('').some((char) => char < ' ' || char === '\u007f')
  );
  return value;
}
function endpoint(value: unknown): string {
  check(id(value));
  return value;
}
function issueBlocker(value: unknown): IssueBlocker {
  const raw = object(value);
  check(typeof raw.code === 'string' && raw.code.length > 0 && raw.code.length <= 64);
  return {
    code: raw.code,
    resolution: text(raw.resolution, 1024),
    ...(raw.detail === undefined ? {} : {detail: blockerDetail(raw.detail)}),
    ...(raw.endpoint_repository === undefined ? {} : {endpoint_repository: endpoint(raw.endpoint_repository)}),
    ...(raw.endpoint_issue === undefined ? {} : {endpoint_issue: endpoint(raw.endpoint_issue)}),
  };
}
function issueReadiness(value: unknown): IssueReadiness {
  const raw = object(value);
  check(Array.isArray(raw.blockers) && raw.blockers.length <= 64);
  return {
    readiness: closedSet(raw.readiness, readinessStates),
    reason: text(raw.reason, 256),
    revision: counter(raw.revision),
    assessed_unix: counter(raw.assessed_unix),
    ...(raw.acceptance === undefined ? {} : {acceptance: text(raw.acceptance, 256)}),
    blockers: raw.blockers.map(issueBlocker),
  };
}
const decimalID = (value: unknown): value is string =>
  typeof value === 'string' && /^(0|[1-9][0-9]{0,18})$/.test(value) && BigInt(value) <= 9223372036854775807n;
function checkResult(value: unknown): IssueCheckResult {
  const raw = object(value);
  check(typeof raw.passed === 'boolean');
  return {
    context: text(raw.context, 256),
    passed: raw.passed,
    ...(raw.state === undefined ? {} : {state: text(raw.state, 64)}),
  };
}
function staleFlag(value: unknown): boolean {
  check(value === undefined || typeof value === 'boolean');
  return value === true;
}
function issueChecks(value: unknown): IssueChecks {
  const raw = object(value);
  check(Array.isArray(raw.results) && raw.results.length <= 64);
  check(decimalID(raw.pr_number) && decimalID(raw.pr_id) && commitId(raw.head_oid) && commitId(raw.base_oid));
  return {
    verdict: closedSet(raw.verdict, checkVerdicts),
    reason: text(raw.reason, 256),
    resolution: text(raw.resolution, 1024),
    head_oid: raw.head_oid,
    base_oid: raw.base_oid,
    pr_number: raw.pr_number,
    pr_id: raw.pr_id,
    policy_revision: counter(raw.policy_revision),
    native_revision: counter(raw.native_revision),
    revision: counter(raw.revision),
    assessed_unix: counter(raw.assessed_unix),
    stale: staleFlag(raw.stale),
    results: raw.results.map(checkResult),
  };
}
function mergeReasonText(value: unknown): string | undefined {
  if (value === undefined) return undefined;
  if (value === '') return undefined;
  check(mergeReasons.includes(value as string));
  return value as string;
}
function mergeCommitText(value: unknown): string | undefined {
  if (value === undefined) return undefined;
  check(commitId(value));
  return value;
}
function mergeFinishedUnix(value: unknown): number | undefined {
  if (value === undefined) return undefined;
  return counter(value);
}
function issueMerge(value: unknown): IssueMerge {
  const raw = object(value);
  check(decimalID(raw.pr_number));
  check(decimalID(raw.pr_id));
  check(commitId(raw.head_oid));
  check(commitId(raw.base_oid));
  const reason = mergeReasonText(raw.reason);
  const mergedCommit = mergeCommitText(raw.merged_commit);
  const finished = mergeFinishedUnix(raw.finished_unix);
  return {
    stage: closedSet(raw.stage, mergeStages),
    resolution: text(raw.resolution, 1024),
    head_oid: raw.head_oid,
    base_oid: raw.base_oid,
    pr_number: raw.pr_number,
    pr_id: raw.pr_id,
    revision: counter(raw.revision),
    stale: staleFlag(raw.stale),
    ...(reason === undefined ? {} : {reason}),
    ...(mergedCommit === undefined ? {} : {merged_commit: mergedCommit}),
    ...(finished === undefined ? {} : {finished_unix: finished}),
  };
}
function acceptanceRef(value: unknown): IssueAcceptanceRef {
  const raw = object(value);
  check(id(raw.issue_index) && id(raw.approver));
  return {
    id: text(raw.id, 128),
    issue_index: raw.issue_index,
    approver: raw.approver,
    native_revision: counter(raw.native_revision),
    content_version: counter(raw.content_version),
  };
}
function acceptanceStatus(value: unknown): IssueAcceptanceStatus {
  const raw = object(value);
  check(typeof raw.valid === 'boolean');
  check(raw.withdrawn === undefined || typeof raw.withdrawn === 'boolean');
  check(raw.reasons === undefined || Array.isArray(raw.reasons));
  return {
    valid: raw.valid,
    withdrawn: raw.withdrawn === true,
    revision: counter(raw.revision),
    reasons: raw.reasons === undefined ? [] : boundedLines(raw.reasons, 16, 256),
    ...(raw.acceptance === undefined ? {} : {acceptance: acceptanceRef(raw.acceptance)}),
  };
}
function nonEmpty(value: unknown): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(typeof value === 'string');
  return value;
}
function workOutcome(value: unknown): string | undefined {
  const outcome = nonEmpty(value);
  if (outcome === undefined) return undefined;
  check(runOutcomes.includes(outcome));
  return outcome;
}
function opEffect(value: unknown): string | undefined {
  const effect = nonEmpty(value);
  if (effect === undefined) return undefined;
  check(opEffects.includes(effect));
  return effect;
}
function opCompletion(value: unknown): string | undefined {
  const completion = nonEmpty(value);
  if (completion === undefined) return undefined;
  check(opCompletions.includes(completion));
  return completion;
}
function workAttempt(value: unknown): WorkAttempt {
  const raw = object(value);
  check(commandId(raw.id) && roleName(raw.role) && commandId(raw.run));
  check(typeof raw.attempts === 'number' && Number.isSafeInteger(raw.attempts) && raw.attempts >= 0);
  const outcome = workOutcome(raw.outcome);
  return {
    id: raw.id,
    role: raw.role,
    stage: closedSet(raw.stage, assignmentStages),
    run: raw.run,
    attempts: raw.attempts,
    created_unix: counter(raw.created_unix),
    ...(outcome === undefined ? {} : {outcome}),
    ...(nonEmpty(raw.reason) === undefined ? {} : {reason: text(raw.reason, 256)}),
    ...(typeof raw.finished_unix === 'number' ? {finished_unix: counter(raw.finished_unix)} : {}),
  };
}
function workCorrection(value: unknown): WorkCorrection {
  const raw = object(value);
  const candidate = nonEmpty(raw.candidate);
  if (candidate !== undefined) check(commitId(candidate));
  const effect = opEffect(raw.effect);
  const completion = opCompletion(raw.completion);
  return {
    updated_unix: counter(raw.updated_unix),
    ...(candidate === undefined ? {} : {candidate}),
    ...(effect === undefined ? {} : {effect}),
    ...(completion === undefined ? {} : {completion}),
    ...(nonEmpty(raw.reason) === undefined ? {} : {reason: text(raw.reason, 256)}),
  };
}
function workReview(value: unknown): WorkReview {
  const raw = object(value);
  check(runRef(raw.run) && commitId(raw.head_oid));
  const effect = opEffect(raw.effect);
  const completion = opCompletion(raw.completion);
  return {
    run: raw.run,
    head_oid: raw.head_oid,
    event: text(raw.event, 64),
    ...(effect === undefined ? {} : {effect}),
    ...(completion === undefined ? {} : {completion}),
    ...(nonEmpty(raw.reason) === undefined ? {} : {reason: text(raw.reason, 256)}),
  };
}
function publicationIdentity(raw: Record<string, unknown>) {
  check(commitId(raw.candidate));
  check(commitId(raw.base_sha));
  check(commandId(raw.id));
  check(runRef(raw.run));
  check(opEffects.includes(raw.publish_effect as string));
  check(opEffects.includes(raw.pr_create_effect as string));
  return {
    candidate: raw.candidate as string,
    base_sha: raw.base_sha as string,
    id: raw.id as string,
    run: raw.run as string,
    publish_effect: raw.publish_effect as string,
    pr_create_effect: raw.pr_create_effect as string,
  };
}
function historyEntries(value: unknown, limit: number): unknown[] {
  if (value === undefined) return [];
  check(Array.isArray(value) && value.length <= limit);
  return value;
}
function publicationHistories(raw: Record<string, unknown>) {
  return {
    corrections: historyEntries(raw.corrections, 20).map(workCorrection),
    reviews: historyEntries(raw.reviews, 20).map(workReview),
  };
}
function workPublication(value: unknown): WorkPublication {
  const raw = object(value);
  const identity = publicationIdentity(raw);
  const histories = publicationHistories(raw);
  if (raw.pr_number !== undefined) check(id(raw.pr_number));
  if (raw.pr_id !== undefined) check(id(raw.pr_id));
  return {
    ...identity,
    target_branch: text(raw.target_branch, 256),
    stage: closedSet(raw.stage, publicationStages),
    corrections_total: counter(raw.corrections_total),
    reviews_total: counter(raw.reviews_total),
    created_unix: counter(raw.created_unix),
    ...(raw.pr_number === undefined ? {} : {pr_number: raw.pr_number as string}),
    ...(raw.pr_id === undefined ? {} : {pr_id: raw.pr_id as string}),
    ...(nonEmpty(raw.reason) === undefined ? {} : {reason: text(raw.reason, 256)}),
    ...(typeof raw.finished_unix === 'number' ? {finished_unix: counter(raw.finished_unix)} : {}),
    corrections: histories.corrections,
    reviews: histories.reviews,
  };
}
function workClosure(value: unknown): WorkClosure {
  const raw = object(value);
  const state = closedSet(raw.state, closureStates);
  if (state === 'confirmed') check(id(raw.occurrence) && typeof raw.closed_unix === 'number' && raw.closed_unix > 0);
  if (raw.occurrence !== undefined) check(id(raw.occurrence));
  return {
    state,
    ...(raw.occurrence === undefined ? {} : {occurrence: raw.occurrence as string}),
    ...(typeof raw.closed_unix === 'number' ? {closed_unix: counter(raw.closed_unix)} : {}),
  };
}
function workCapability(value: unknown): WorkCapability {
  const raw = object(value);
  check(typeof raw.available === 'boolean');
  const reason = nonEmpty(raw.reason);
  if (reason !== undefined) check(capabilityReasons.includes(reason));
  return {available: raw.available, ...(reason === undefined ? {} : {reason})};
}
function workCapabilities(value: unknown): WorkCapabilities {
  const raw = object(value);
  return {
    watch: workCapability(raw.watch),
    stop: workCapability(raw.stop),
    retry: workCapability(raw.retry),
    takeover: workCapability(raw.takeover),
    pause: workCapability(raw.pause),
    resume: workCapability(raw.resume),
  };
}
function requestedRetry(value: unknown): RequestedRetry | undefined {
  if (value === undefined) return undefined;
  const raw = object(value);
  if (raw.retry === undefined) return undefined;
  const retry = object(raw.retry);
  check(commandId(retry.command_id) && commandId(retry.prior_run) && typeof retry.queued === 'boolean');
  return {
    command_id: retry.command_id,
    reason: text(retry.reason, 256),
    prior_run: retry.prior_run,
    queued: retry.queued,
  };
}
function workLinks(value: unknown): WorkLinks | undefined {
  if (value === undefined) return undefined;
  const raw = object(value);
  check(nativePath(raw.issue_url));
  if (raw.pull_url !== undefined) check(nativePath(raw.pull_url));
  return {
    issue_url: raw.issue_url as string,
    ...(raw.pull_url === undefined ? {} : {pull_url: raw.pull_url as string}),
  };
}
// factoryIssueDetail binds one native issue's recorded acceptance,
// readiness, attempt, publication, review, check, merge and closure
// facts plus the viewer's current capabilities to its exact repository
// and issue.
export function factoryIssueDetail(value: unknown, repository: string, issue: string): FactoryIssueDetail {
  const data = object(value);
  check(data.repository === repository && data.issue === issue);
  const requested = requestedRetry(data.requested);
  const links = workLinks(data.links);
  return {
    repository,
    issue,
    status: acceptanceStatus(data.status),
    closure: workClosure(data.closure),
    capabilities: workCapabilities(data.capabilities),
    observed_unix: counter(data.observed_unix),
    ...(data.readiness === undefined ? {} : {readiness: issueReadiness(data.readiness)}),
    ...(data.checks === undefined ? {} : {checks: issueChecks(data.checks)}),
    ...(data.merge === undefined ? {} : {merge: issueMerge(data.merge)}),
    ...(data.attempt === undefined ? {} : {attempt: workAttempt(data.attempt)}),
    ...(data.publication === undefined ? {} : {publication: workPublication(data.publication)}),
    ...(requested === undefined ? {} : {requested_retry: requested}),
    ...(links === undefined ? {} : {links}),
  };
}

export interface FactoryQueueSummary {
  queued: number;
  active: number;
}
export interface FactoryPreparationSummary {
  project: string;
  ready: boolean;
}
export interface FactoryEffectiveSummary {
  effective: boolean;
  dispatch_open: boolean;
  missing: string[];
}
export interface FactoryRepositorySummary {
  repository: string;
  queue: FactoryQueueSummary;
  effective: FactoryEffectiveSummary;
  preparation?: FactoryPreparationSummary;
}
function effectiveSummary(value: unknown): FactoryEffectiveSummary {
  const raw = object(value);
  check(typeof raw.effective === 'boolean' && typeof raw.dispatch_open === 'boolean');
  return {effective: raw.effective, dispatch_open: raw.dispatch_open, missing: boundedCodes(raw.missing, 16)};
}
// factoryRepositorySummary binds one repository's recorded queue counts,
// setup preparation and effective authority to its exact repository.
export function factoryRepositorySummary(value: unknown, repository: string): FactoryRepositorySummary {
  const data = object(value);
  check(data.repository === repository);
  const queue = object(data.queue);
  const preparation =
    data.preparation === undefined || data.preparation === null ? undefined : object(data.preparation);
  if (preparation !== undefined) check(projectId(preparation.project) && typeof preparation.ready === 'boolean');
  return {
    repository,
    queue: {queued: counter(queue.queued), active: counter(queue.active)},
    effective: effectiveSummary(data.effective),
    ...(preparation === undefined
      ? {}
      : {preparation: {project: preparation.project as string, ready: preparation.ready as boolean}}),
  };
}

export interface PauseRunOutcome {
  id: string;
  confirmed: boolean;
  uncertain: boolean;
  outcome?: string;
  reason?: string;
}
export interface PauseReceipt {
  command_id: string;
  paused: boolean;
  repository: string;
  revision: number;
  cause: string;
  closed_by: string;
  runs: PauseRunOutcome[];
}
export interface ResumeReceipt {
  command_id: string;
  revision: number;
  reopened: boolean;
  effective: FactoryEffectiveSummary;
}
export interface StopReceipt {
  run_id: string;
  confirmed: boolean;
  uncertain: boolean;
  outcome?: string;
  reason?: string;
}
export interface RetryDecision {
  prior_run: string;
  command_id: string;
  reason: string;
  queued: boolean;
}
export interface TakeoverRecord {
  run: string;
  member: string;
  project: string;
  destination: string;
  command_id: string;
  copied: string;
}
export interface FactoryCommandRecord {
  id: string;
  type: string;
  principal: string;
  digest: string;
  target?: string;
  outcome?: string;
}
// runStopOutcome admits one recorded run-stop outcome from a pause or
// Project-stop receipt. Confirmed and uncertain stay distinct.
export function runStopOutcome(value: unknown): PauseRunOutcome {
  const raw = object(value);
  check(commandId(raw.id) && typeof raw.confirmed === 'boolean' && typeof raw.uncertain === 'boolean');
  if (raw.outcome !== undefined) check(raw.outcome === '' || runOutcomes.includes(raw.outcome as string));
  return {
    id: raw.id,
    confirmed: raw.confirmed,
    uncertain: raw.uncertain,
    ...(typeof raw.outcome === 'string' && raw.outcome !== '' ? {outcome: raw.outcome} : {}),
    ...(typeof raw.reason === 'string' && raw.reason !== '' ? {reason: text(raw.reason, 256)} : {}),
  };
}
// pauseReceipt binds one pause outcome to its exact command and
// repository. Uncertain runs stay fenced; they never read as cancelled.
export function pauseReceipt(value: unknown, commandID: string, repository: string): PauseReceipt {
  const data = object(value);
  check(data.command_id === commandID && data.paused === true);
  const withdrawal = object(data.withdrawal);
  check(withdrawal.repository === repository);
  check(Array.isArray(data.runs) && data.runs.length <= 64);
  return {
    command_id: commandID,
    paused: true,
    repository,
    revision: counter(withdrawal.revision),
    cause: text(withdrawal.cause, 256),
    closed_by: text(withdrawal.closed_by, 128),
    runs: data.runs.map(runStopOutcome),
  };
}
// resumeReceipt binds one resume outcome to its exact command.
export function resumeReceipt(value: unknown, commandID: string): ResumeReceipt {
  const data = object(value);
  check(data.command_id === commandID && typeof data.reopened === 'boolean');
  return {
    command_id: commandID,
    revision: counter(data.revision),
    reopened: data.reopened,
    effective: effectiveSummary(data.effective),
  };
}
// stopReceipt binds one run-stop outcome to its exact run. Confirmed and
// uncertain stay distinct; neither is inferred from the request alone.
export function stopReceipt(value: unknown, runID: string): StopReceipt {
  const data = object(value);
  check(data.run_id === runID && typeof data.confirmed === 'boolean' && typeof data.uncertain === 'boolean');
  if (data.outcome !== undefined) check(data.outcome === '' || runOutcomes.includes(data.outcome as string));
  return {
    run_id: runID,
    confirmed: data.confirmed,
    uncertain: data.uncertain,
    ...(typeof data.outcome === 'string' && data.outcome !== '' ? {outcome: data.outcome} : {}),
    ...(typeof data.reason === 'string' && data.reason !== '' ? {reason: text(data.reason, 256)} : {}),
  };
}
// retryDecision binds one explicit retry to its exact command and prior run.
export function retryDecision(value: unknown, commandID: string, priorRun: string): RetryDecision {
  const data = object(value);
  check(data.command_id === commandID && data.prior_run === priorRun && data.queued === true);
  return {prior_run: priorRun, command_id: commandID, reason: text(data.reason, 256), queued: true};
}
// takeoverRecord binds one takeover to its exact command, run and member checkout.
export function takeoverRecord(value: unknown, commandID: string, runID: string): TakeoverRecord {
  const data = object(value);
  check(data.command_id === commandID && data.run === runID && projectId(data.project));
  check(typeof data.member === 'string' && data.member.length > 0 && data.member.length <= 32);
  check(data.member === (data.member as string).trim());
  check(typeof data.destination === 'string' && (data.destination as string).startsWith(`/home/${data.member}/`));
  return {
    run: runID,
    member: data.member as string,
    project: data.project as string,
    destination: text(data.destination, 256),
    command_id: commandID,
    copied: text(data.copied, 64),
  };
}
// factoryCommandRecord binds one durable command read-back to its exact identity.
export function factoryCommandRecord(value: unknown, commandID: string): FactoryCommandRecord {
  const data = object(value);
  check(data.id === commandID && digest(data.digest));
  return {
    id: commandID,
    type: text(data.type, 64),
    principal: text(data.principal, 256),
    digest: data.digest as string,
    ...(data.target === undefined ? {} : {target: text(data.target, 256)}),
    ...(data.outcome === undefined ? {} : {outcome: text(data.outcome, 1024)}),
  };
}
export interface ProjectStopResult {
  project: string;
  running: boolean;
  hold: boolean;
  hold_synced: boolean;
  withdrawal_revision: number;
  runs: PauseRunOutcome[];
}
// projectStopResult binds one explicit Project-stop outcome to its exact
// project: the observed running state, maintenance-hold sync and every
// outstanding run's stop outcome.
export function projectStopResult(value: unknown, projectID: string): ProjectStopResult {
  const data = object(value);
  const environment = object(data.environment);
  check(environment.id === projectID && typeof environment.running === 'boolean');
  const control = object(data.control);
  check(typeof control.hold === 'boolean' && typeof control.hold_synced === 'boolean');
  const withdrawal = object(control.withdrawal);
  check(Array.isArray(control.runs) && control.runs.length <= 64);
  return {
    project: projectID,
    running: environment.running,
    hold: control.hold,
    hold_synced: control.hold_synced,
    withdrawal_revision: counter(withdrawal.revision),
    runs: control.runs.map(runStopOutcome),
  };
}
export interface SodaErrorDetail {
  code?: string;
  message?: string;
}
function errorRecord(value: unknown): Record<string, unknown> | undefined {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return undefined;
  return value as Record<string, unknown>;
}
function errorText(value: unknown, limit: number): string | undefined {
  if (typeof value !== 'string' || value === '') return undefined;
  return value.slice(0, limit);
}
// sodaErrorDetail reads the shared `{error: {code, message}}` refusal
// envelope. Unknown shapes yield no detail, never a guessed reason.
export function sodaErrorDetail(value: unknown): SodaErrorDetail {
  const error = errorRecord(errorRecord(value)?.error);
  const code = errorText(error?.code, 128);
  const message = errorText(error?.message, 1024);
  return {
    ...(code === undefined ? {} : {code}),
    ...(message === undefined ? {} : {message}),
  };
}
export interface PendingRuns {
  runs: string[];
  message: string;
}
// pendingRuns admits a 409 unsettled-runs refusal: the blocking run
// identities and the retained message, never a success verdict.
export function pendingRuns(value: unknown): PendingRuns {
  const data = object(value);
  check(data.error === 'pending_runs' && Array.isArray(data.runs) && data.runs.length <= 64);
  for (const run of data.runs) check(commandId(run));
  return {runs: data.runs as string[], message: text(data.message, 1024)};
}
