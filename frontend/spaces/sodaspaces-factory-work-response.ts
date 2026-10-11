import {check, id, object, projectId} from './sodaspaces-api.js';

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
  reason?: string;
  merged_commit?: string;
  finished_unix?: number;
}
export interface FactoryIssueDetail {
  repository: string;
  issue: string;
  status: IssueAcceptanceStatus;
  readiness?: IssueReadiness;
  checks?: IssueChecks;
  merge?: IssueMerge;
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
// factoryIssueDetail binds one native issue's recorded acceptance,
// readiness, check and merge facts to its exact repository and issue.
export function factoryIssueDetail(value: unknown, repository: string, issue: string): FactoryIssueDetail {
  const data = object(value);
  check(data.repository === repository && data.issue === issue);
  return {
    repository,
    issue,
    status: acceptanceStatus(data.status),
    ...(data.readiness === undefined ? {} : {readiness: issueReadiness(data.readiness)}),
    ...(data.checks === undefined ? {} : {checks: issueChecks(data.checks)}),
    ...(data.merge === undefined ? {} : {merge: issueMerge(data.merge)}),
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
function pauseRunOutcome(value: unknown): PauseRunOutcome {
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
    runs: data.runs.map(pauseRunOutcome),
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
