import {check, object} from './sodaspaces-api.js';
import {decisionID} from './sodaspaces-intent-response.js';

// Combined subscribed-feed wire contract (Lane B release, R42.1 preparation).
// Backend owns identity, visibility and cursors: entries arrive newest-first
// with a stable order key (seq) and an opaque resume cursor. The subscription
// filter applies inside already permitted activity and never grants access.
// This module only parses and builds that contract; it performs no requests.

export const feedEntryKinds = ['record', 'post'] as const;
export type FeedEntryKind = (typeof feedEntryKinds)[number];

// A served page contains at most the requested number of entries.
export const feedPageLimit = 256;
export const feedPageMaxEntries = feedPageLimit;

// feedPayloadCap mirrors the serving preview bound: one typed-payload
// list previews at most 24 recorded members beside its honest total.
export const feedPayloadCap = 24;

// Feed milestone activities (Lane B3): consequential Factory/Jev
// events served as their own record entries instead of status
// suffixes. A milestone entry names its bound assignment in
// `assignment` when available and receives its own order key when recorded.
export const feedCommunicationKinds = [
  'ask',
  'answer',
  'reply',
  'flag',
  'learned',
  'suggest',
  'partial',
  'blocked',
  'stuck',
  'record_decision',
  'finish_incomplete',
  'system',
  'library_maintenance',
  'artifact',
  'journey',
  'diagnosis',
  'repair_draft',
] as const;
export type FeedCommunicationKind = (typeof feedCommunicationKinds)[number];
export const feedActivities = ['merged', 'completed', 'judgment', ...feedCommunicationKinds] as const;
export type FeedActivity = (typeof feedActivities)[number];

// feedSummaryLimit mirrors the serving bound on one record's summary
// preview, in runes (Unicode code points), not UTF-16 code units.
export const feedSummaryLimit = 512;

export interface FeedEntry {
  kind: FeedEntryKind;
  id: string;
  repository: number;
  seq: number;
  // Optional server-derived enrichment (Lane B committed contract, all
  // omitempty): filled from the canonical owner, never caller input.
  // Posts always carry them; records may arrive unenriched.
  actor?: string;
  author?: string;
  project?: string;
  at?: number;
  status?: string;
  // Additive recorded content (Lane B2, all omitempty, records only):
  // the bound native issue and pull request, the candidate commit the
  // record produced, and a bounded preview of its recorded result
  // summary. An unsuppliable field stays absent, never invented.
  issue?: number;
  pr?: number;
  ref?: string;
  summary?: string;
  // Additive milestone identity and typed component content (Lane B3,
  // all omitempty): empty on assignment records and posts. A payload
  // that is present but malformed parses to {invalid: true} so the
  // entry degrades honestly instead of failing the whole page or
  // coercing the unreadable detail into a success.
  activity?: FeedActivity;
  assignment?: string;
  eventIdentity?: string;
  payload?: FeedPayload;
}

export interface FeedPlanTask {
  id: string;
  objective?: string;
  lane?: string;
  dependsOn: string[];
  outputs: string[];
  checks: string[];
}

export interface FeedPlanLane {
  id: string;
  tasks: string[];
}

export interface FeedPlanCheck {
  id: string;
  argv: string[];
  scope?: string;
  required?: boolean;
}

export interface FeedInspected {
  path: string;
  commit?: string;
  evidence?: string;
}

export interface FeedRejected {
  approach?: string;
  reason?: string;
}

export interface FeedPlan {
  approach?: string;
  sourceCommit?: string;
  generation?: number;
  findings: string[];
  findingsTotal: number;
  unresolved: string[];
  rejected: FeedRejected[];
  inspected: FeedInspected[];
  inspectedTotal: number;
  acceptanceCoverage: string[];
  tasks: FeedPlanTask[];
  tasksTotal: number;
  lanes: FeedPlanLane[];
  checks: FeedPlanCheck[];
  checksTotal: number;
}

export interface FeedTaskState {
  id: string;
  state?: string;
}

export interface FeedTaskStates {
  tasks: FeedTaskState[];
  total: number;
}

export interface FeedImplementation {
  candidate?: string;
  diagnosticRefs: string[];
}

export interface FeedFindingLocation {
  path: string;
  side?: string;
  line?: number;
}

export interface FeedFinding {
  key: string;
  severity?: string;
  defect?: string;
  evidence?: string;
  requiredOutcome?: string;
  fixKind?: string;
  actionable?: boolean;
  location?: FeedFindingLocation;
}

export interface FeedAssessment {
  key: string;
  status?: string;
  evidence?: string;
}

export interface FeedClearReview {
  head?: string;
  base?: string;
  coverageDigest?: string;
  assessments: FeedAssessment[];
  assessmentsTotal: number;
}

export interface FeedReview {
  verdict?: string;
  summary?: string;
  cycleId?: string;
  findings: string[];
  findingsTotal: number;
  newFindings: FeedFinding[];
  newFindingsTotal: number;
  assessments: FeedAssessment[];
  assessmentsTotal: number;
  clear?: FeedClearReview;
}

export interface FeedDisposition {
  key: string;
  disposition?: string;
  changedPaths: string[];
  commits: string[];
}

export interface FeedCorrection {
  newCandidate?: string;
  setDigest?: string;
  findings: FeedFinding[];
  findingsTotal: number;
  dispositions: FeedDisposition[];
  dispositionsTotal: number;
}

export interface FeedIntentTask {
  key: string;
  objective?: string;
  dependsOn: string[];
  outcomes: string[];
}

export interface FeedIntentChange {
  intentId?: string;
  reason?: string;
  tasks: FeedIntentTask[];
  tasksTotal: number;
}

export interface FeedAttention {
  issue?: number;
  reason?: string;
}

export interface FeedCoverage {
  intentId?: string;
  outcomeId?: string;
  taskKeys: string[];
  deliveredAcceptanceIds: string[];
}

export interface FeedReassessment {
  summary?: string;
  retainedIssues: number[];
  changes: FeedIntentChange[];
  changesTotal: number;
  attention: FeedAttention[];
  coverage: FeedCoverage[];
  coverageTotal: number;
  inspectedTotal: number;
}

export interface FeedCheckFailure {
  check?: string;
  detail?: string;
}

export interface FeedRepair {
  outcome?: string;
  causes: FeedCheckFailure[];
  retainedIssues: number[];
  changes: FeedIntentChange[];
  changesTotal: number;
  attention: FeedAttention[];
  coverage: FeedCoverage[];
  coverageTotal: number;
  inspectedTotal: number;
}

export interface FeedCheckReceipt {
  id: string;
  outcome?: string;
  artifactDigest?: string;
}

export interface FeedVerification {
  outcome?: string;
  deliveryId?: string;
  checks: FeedCheckReceipt[];
  checksTotal: number;
  failures: FeedCheckFailure[];
  coverage: FeedCoverage[];
  coverageTotal: number;
  artifactsTotal: number;
  evidenceTotal: number;
}

export interface FeedDecompCandidate {
  key?: string;
  rationale?: string;
  tasks: FeedIntentTask[];
  tasksTotal: number;
}

export interface FeedDecomposition {
  operation?: string;
  attention?: string;
  candidates: FeedDecompCandidate[];
  candidatesTotal: number;
}

export interface FeedJudgmentCandidate {
  id: string;
  repository?: number;
  issue?: number;
  objective?: string;
}

export interface FeedJudgment {
  kind: 'priority' | 'delivery';
  selected?: string;
  outcome?: string;
  effective?: string;
  model?: string;
  probability?: number;
  confidence?: number;
  probabilities: Record<string, number>;
  candidates: FeedJudgmentCandidate[];
  candidatesTotal: number;
}

export interface FeedMerge {
  stage?: string;
  publication?: string;
  mergedCommit?: string;
  mergedUnix?: number;
  originalClosedUnix?: number;
}

export interface FeedCompletion {
  event?: string;
  publication?: string;
  occurrence?: string;
  mergedCommit?: string;
  generation?: number;
}

export interface FeedArtifact {
  id: string;
  name: string;
  mediaType: string;
  href: string;
}

export interface FeedGraphItem {
  id?: string;
  label: string;
  detail?: string;
  state?: string;
  dependsOn?: string[];
}

export interface FeedCommunication {
  intentSource?: {intent: string; revision: string; predecessor?: string; state: string; owner: string};
  planSource?: {attemptRoot: string; plan: string; digest: string; source: string; generation: number};
  planningSource?: {
    snapshot: string;
    recordDigest: string;
    strategy: string;
    outcome: string;
    provisional: boolean;
    recordedUnix: number;
    judgment: Record<string, unknown>;
  };
  commits?: {revision: string; summary: string}[];
  source?: FeedSourceExcerpt;
  kind: FeedCommunicationKind;
  componentId?: string;
  title: string;
  summary: string;
  question?: string;
  questionState?: 'open' | 'answered' | 'replied' | 'incorporated';
  disposition?: 'applied' | 'refused' | 'incorporated';
  dispositionReason?: string;
  resolution?: string;
  scope?: string;
  severity?: 'info' | 'warning' | 'critical';
  evidenceRefs: string[];
  options: string[];
  completed: string[];
  remaining: string[];
  attempts: string[];
  decision?: string;
  rationale?: string;
  alternatives: string[];
  replyTo?: string;
  items: FeedGraphItem[];
  artifacts?: FeedArtifact[];
  libraryItemId?: string;
  libraryRevision?: number;
  before?: string;
  after?: string;
}

export interface FeedSourceExcerpt {
  path: string;
  revision?: string;
  startLine: number;
  code: string;
  after?: string;
  highlightLines: number[];
}

// FeedPayload is one record's typed component content. At most one
// role family is set; `tasks` may accompany an implementation or a
// correction. `invalid` marks a present-but-unreadable payload.
export interface FeedPayload {
  invalid?: boolean;
  componentId?: string;
  communication?: FeedCommunication;
  plan?: FeedPlan;
  tasks?: FeedTaskStates;
  implementation?: FeedImplementation;
  review?: FeedReview;
  correction?: FeedCorrection;
  reassessment?: FeedReassessment;
  repair?: FeedRepair;
  verification?: FeedVerification;
  decomposition?: FeedDecomposition;
  judgment?: FeedJudgment;
  merge?: FeedMerge;
  completion?: FeedCompletion;
}

export interface FeedPageView {
  entries: FeedEntry[];
  next: string;
  truncated: boolean;
}

export interface FeedSubscriptionView {
  subscriber: string;
  repositories: number[];
  kinds: FeedEntryKind[];
  revision: number;
}

export function feedEntryKey(entry: FeedEntry): string {
  return `${entry.kind}:${entry.id}`;
}

function positiveInt(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value > 0;
}

function boundedID(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= 256;
}

// parseFeedCursor admits one opaque resume cursor: empty reads latest,
// otherwise the versioned positive order bound behind the page tail.
export function parseFeedCursor(value: unknown): string {
  check(typeof value === 'string' && value.length <= 64);
  if (value === '') return '';
  const head: string[] = value.split(':');
  check(head.length === 2 && head[0] === 'v1');
  const bound = Number(head[1]);
  check(Number.isSafeInteger(bound) && bound > 0);
  return value;
}

function optionalText(value: unknown, limit: number): string | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'string' && value.length > 0 && value.length <= limit);
  return value;
}

function optionalStamp(value: unknown): number | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= 0);
  return value;
}

function optionalPositiveInt(value: unknown): number | undefined {
  if (value === undefined) return undefined;
  check(positiveInt(value));
  return value;
}

function optionalSummary(value: unknown): string | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'string' && value.length > 0 && [...value].length <= feedSummaryLimit);
  return value;
}

// Typed-payload parsing (Lane B3). Every helper throws through `check`
// on a present-but-malformed value; parseFeedPayload converts that to
// the {invalid: true} degradation, never a partial good-looking parse.
// Absent and empty optional fields stay absent, unknown fields are
// ignored, and a declared total must cover its previewed members.
function payloadText(value: unknown): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(typeof value === 'string' && [...value].length <= feedSummaryLimit);
  return value;
}

function payloadID(value: unknown): string {
  check(boundedID(value));
  return value;
}

function payloadOptionalID(value: unknown): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(boundedID(value));
  return value;
}

function payloadCount(value: unknown): number {
  if (value === undefined) return 0;
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= 0);
  return value;
}

function payloadTotal(value: unknown, shown: number): number {
  if (value === undefined) return shown;
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= shown);
  return value;
}

function payloadPositiveInt(value: unknown): number | undefined {
  if (value === undefined || value === 0) return undefined;
  check(positiveInt(value));
  return value;
}

function payloadBool(value: unknown): boolean | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'boolean');
  return value;
}

function payloadProbability(value: unknown): number | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 1);
  return value;
}

function payloadStrings(value: unknown): string[] {
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length <= feedPayloadCap);
  return value.map((text) => {
    check(typeof text === 'string' && [...text].length <= feedSummaryLimit);
    return text;
  });
}

function payloadIDs(value: unknown): string[] {
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length <= 256);
  return value.map((text) => {
    check(typeof text === 'string' && text.length > 0 && text.length <= feedSummaryLimit);
    return text;
  });
}

// CheckSpec.Validate owns argv's 64-argument/16 KiB UTF-8 bound. An
// argument is command data, not a 512-character feed summary or ID.
function payloadCheckArgv(value: unknown): string[] {
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length > 0 && value.length <= 64);
  let bytes = 0;
  return value.map((argument) => {
    check(
      typeof argument === 'string' && argument.length > 0 && argument.length <= 16 * 1024 && !argument.includes('\0')
    );
    bytes += new TextEncoder().encode(argument).length;
    check(bytes <= 16 * 1024);
    return argument;
  });
}

function payloadIssueList(value: unknown): number[] {
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length <= 1024);
  return value.map((issue) => {
    check(typeof issue === 'number' && Number.isSafeInteger(issue) && issue >= 0);
    return issue;
  });
}

function payloadList<T>(value: unknown, parse: (row: Record<string, unknown>) => T): T[] {
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length <= feedPayloadCap);
  return value.map((row) => parse(object(row)));
}

function parsePlanPayload(value: unknown): FeedPlan {
  const plan = object(value);
  const findings = payloadStrings(plan.findings);
  const inspected = payloadList(plan.inspected, (row) => {
    const entry: FeedInspected = {path: payloadID(row.path)};
    const commit = payloadOptionalID(row.commit);
    const evidence = payloadText(row.evidence);
    if (commit !== undefined) entry.commit = commit;
    if (evidence !== undefined) entry.evidence = evidence;
    return entry;
  });
  const tasks = payloadList(plan.tasks, (row) => {
    const task: FeedPlanTask = {
      id: payloadID(row.id),
      dependsOn: payloadIDs(row.depends_on),
      outputs: payloadIDs(row.outputs),
      checks: payloadIDs(row.checks),
    };
    const objective = payloadText(row.objective);
    const lane = payloadOptionalID(row.lane);
    if (objective !== undefined) task.objective = objective;
    if (lane !== undefined) task.lane = lane;
    return task;
  });
  const checks = payloadList(plan.checks, (row) => {
    const planCheck: FeedPlanCheck = {id: payloadID(row.id), argv: payloadCheckArgv(row.argv)};
    const scope = payloadText(row.scope);
    const required = payloadBool(row.required);
    if (scope !== undefined) planCheck.scope = scope;
    if (required !== undefined) planCheck.required = required;
    return planCheck;
  });
  const parsed: FeedPlan = {
    findings,
    findingsTotal: payloadTotal(plan.findings_total, findings.length),
    unresolved: payloadStrings(plan.unresolved),
    rejected: payloadList(plan.rejected, (row) => {
      const entry: FeedRejected = {};
      const approach = payloadText(row.approach);
      const reason = payloadText(row.reason);
      if (approach !== undefined) entry.approach = approach;
      if (reason !== undefined) entry.reason = reason;
      return entry;
    }),
    inspected,
    inspectedTotal: payloadTotal(plan.inspected_total, inspected.length),
    acceptanceCoverage: payloadIDs(plan.acceptance_coverage),
    tasks,
    tasksTotal: payloadTotal(plan.tasks_total, tasks.length),
    lanes: payloadList(plan.lanes, (row) => ({id: payloadID(row.id), tasks: payloadIDs(row.tasks)})),
    checks,
    checksTotal: payloadTotal(plan.checks_total, checks.length),
  };
  const approach = payloadText(plan.approach);
  const sourceCommit = payloadOptionalID(plan.source_commit);
  const generation = payloadCount(plan.generation);
  if (approach !== undefined) parsed.approach = approach;
  if (sourceCommit !== undefined) parsed.sourceCommit = sourceCommit;
  if (generation > 0) parsed.generation = generation;
  return parsed;
}

function parseTaskStatesPayload(value: unknown): FeedTaskStates {
  const data = object(value);
  const tasks = payloadList(data.tasks, (row) => {
    const task: FeedTaskState = {id: payloadID(row.id)};
    const state = payloadOptionalID(row.state);
    if (state !== undefined) task.state = state;
    return task;
  });
  return {tasks, total: payloadTotal(data.total, tasks.length)};
}

function parseImplementationPayload(value: unknown): FeedImplementation {
  const data = object(value);
  const parsed: FeedImplementation = {diagnosticRefs: payloadIDs(data.diagnostic_refs)};
  const candidate = payloadOptionalID(data.candidate);
  if (candidate !== undefined) parsed.candidate = candidate;
  return parsed;
}

function parseFinding(row: Record<string, unknown>): FeedFinding {
  const finding: FeedFinding = {key: payloadID(row.key)};
  const severity = payloadOptionalID(row.severity);
  const defect = payloadText(row.defect);
  const evidence = payloadText(row.evidence);
  const requiredOutcome = payloadText(row.required_outcome);
  const fixKind = payloadOptionalID(row.fix_kind);
  const actionable = payloadBool(row.actionable);
  if (severity !== undefined) finding.severity = severity;
  if (defect !== undefined) finding.defect = defect;
  if (evidence !== undefined) finding.evidence = evidence;
  if (requiredOutcome !== undefined) finding.requiredOutcome = requiredOutcome;
  if (fixKind !== undefined) finding.fixKind = fixKind;
  if (actionable !== undefined) finding.actionable = actionable;
  if (row.location !== undefined) {
    const location = object(row.location);
    const parsed: FeedFindingLocation = {path: payloadID(location.path)};
    const side = payloadOptionalID(location.side);
    const line = payloadCount(location.line);
    if (side !== undefined) parsed.side = side;
    if (line > 0) parsed.line = line;
    finding.location = parsed;
  }
  return finding;
}

function parseAssessment(row: Record<string, unknown>): FeedAssessment {
  const assessment: FeedAssessment = {key: payloadID(row.key)};
  const status = payloadOptionalID(row.status);
  const evidence = payloadText(row.evidence);
  if (status !== undefined) assessment.status = status;
  if (evidence !== undefined) assessment.evidence = evidence;
  return assessment;
}

function parseReviewPayload(value: unknown): FeedReview {
  const data = object(value);
  const findings = payloadStrings(data.findings);
  const newFindings = payloadList(data.new_findings, parseFinding);
  const assessments = payloadList(data.assessments, parseAssessment);
  const parsed: FeedReview = {
    findings,
    findingsTotal: payloadTotal(data.findings_total, findings.length),
    newFindings,
    newFindingsTotal: payloadTotal(data.new_findings_total, newFindings.length),
    assessments,
    assessmentsTotal: payloadTotal(data.assessments_total, assessments.length),
  };
  const verdict = payloadOptionalID(data.verdict);
  const summary = payloadText(data.summary);
  const cycleId = payloadOptionalID(data.cycle_id);
  if (verdict !== undefined) parsed.verdict = verdict;
  if (summary !== undefined) parsed.summary = summary;
  if (cycleId !== undefined) parsed.cycleId = cycleId;
  if (data.clear !== undefined) {
    const clear = object(data.clear);
    const clearAssessments = payloadList(clear.assessments, parseAssessment);
    const review: FeedClearReview = {
      assessments: clearAssessments,
      assessmentsTotal: payloadTotal(clear.assessments_total, clearAssessments.length),
    };
    const head = payloadOptionalID(clear.head);
    const base = payloadOptionalID(clear.base);
    const coverageDigest = payloadOptionalID(clear.coverage_digest);
    if (head !== undefined) review.head = head;
    if (base !== undefined) review.base = base;
    if (coverageDigest !== undefined) review.coverageDigest = coverageDigest;
    parsed.clear = review;
  }
  return parsed;
}

function parseCorrectionPayload(value: unknown): FeedCorrection {
  const data = object(value);
  const findings = payloadList(data.findings, parseFinding);
  const dispositions = payloadList(data.dispositions, (row) => {
    const entry: FeedDisposition = {
      key: payloadID(row.key),
      changedPaths: payloadIDs(row.changed_paths),
      commits: payloadIDs(row.commits),
    };
    const disposition = payloadOptionalID(row.disposition);
    if (disposition !== undefined) entry.disposition = disposition;
    return entry;
  });
  const parsed: FeedCorrection = {
    findings,
    findingsTotal: payloadTotal(data.findings_total, findings.length),
    dispositions,
    dispositionsTotal: payloadTotal(data.dispositions_total, dispositions.length),
  };
  const newCandidate = payloadOptionalID(data.new_candidate);
  const setDigest = payloadOptionalID(data.set_digest);
  if (newCandidate !== undefined) parsed.newCandidate = newCandidate;
  if (setDigest !== undefined) parsed.setDigest = setDigest;
  return parsed;
}

function parseIntentTask(row: Record<string, unknown>): FeedIntentTask {
  const task: FeedIntentTask = {
    key: payloadID(row.key),
    dependsOn: payloadIDs(row.depends_on),
    outcomes: payloadStrings(row.outcomes),
  };
  const objective = payloadText(row.objective);
  if (objective !== undefined) task.objective = objective;
  return task;
}

function parseIntentChange(row: Record<string, unknown>): FeedIntentChange {
  const tasks = payloadList(row.tasks, parseIntentTask);
  const change: FeedIntentChange = {tasks, tasksTotal: payloadTotal(row.tasks_total, tasks.length)};
  const intentId = payloadOptionalID(row.intent_id);
  const reason = payloadText(row.reason);
  if (intentId !== undefined) change.intentId = intentId;
  if (reason !== undefined) change.reason = reason;
  return change;
}

function parseAttentionRow(row: Record<string, unknown>): FeedAttention {
  const entry: FeedAttention = {};
  const issue = payloadPositiveInt(row.issue);
  const reason = payloadText(row.reason);
  if (issue !== undefined) entry.issue = issue;
  if (reason !== undefined) entry.reason = reason;
  return entry;
}

function parseCoverageRow(row: Record<string, unknown>): FeedCoverage {
  const entry: FeedCoverage = {
    taskKeys: payloadIDs(row.task_keys),
    deliveredAcceptanceIds: payloadIDs(row.delivered_acceptance_ids),
  };
  const intentId = payloadOptionalID(row.intent_id);
  const outcomeId = payloadOptionalID(row.outcome_id);
  if (intentId !== undefined) entry.intentId = intentId;
  if (outcomeId !== undefined) entry.outcomeId = outcomeId;
  return entry;
}

function parseCheckFailure(row: Record<string, unknown>): FeedCheckFailure {
  const entry: FeedCheckFailure = {};
  const checkRef = payloadOptionalID(row.check);
  const detail = payloadText(row.detail);
  if (checkRef !== undefined) entry.check = checkRef;
  if (detail !== undefined) entry.detail = detail;
  return entry;
}

function parseGraphDelta(data: Record<string, unknown>): {
  retainedIssues: number[];
  changes: FeedIntentChange[];
  changesTotal: number;
  attention: FeedAttention[];
  coverage: FeedCoverage[];
  coverageTotal: number;
  inspectedTotal: number;
} {
  const changes = payloadList(data.changes, parseIntentChange);
  const coverage = payloadList(data.coverage, parseCoverageRow);
  return {
    retainedIssues: payloadIssueList(data.retained_issues),
    changes,
    changesTotal: payloadTotal(data.changes_total, changes.length),
    attention: payloadList(data.attention, parseAttentionRow),
    coverage,
    coverageTotal: payloadTotal(data.coverage_total, coverage.length),
    inspectedTotal: payloadCount(data.inspected_total),
  };
}

function parseReassessmentPayload(value: unknown): FeedReassessment {
  const data = object(value);
  const parsed: FeedReassessment = {...parseGraphDelta(data)};
  const summary = payloadText(data.summary);
  if (summary !== undefined) parsed.summary = summary;
  return parsed;
}

function parseRepairPayload(value: unknown): FeedRepair {
  const data = object(value);
  const parsed: FeedRepair = {...parseGraphDelta(data), causes: payloadList(data.causes, parseCheckFailure)};
  const outcome = payloadOptionalID(data.outcome);
  if (outcome !== undefined) parsed.outcome = outcome;
  return parsed;
}

function parseVerificationPayload(value: unknown): FeedVerification {
  const data = object(value);
  const checks = payloadList(data.checks, (row) => {
    const receipt: FeedCheckReceipt = {id: payloadID(row.id)};
    const outcome = payloadOptionalID(row.outcome);
    const artifactDigest = payloadOptionalID(row.artifact_digest);
    if (outcome !== undefined) receipt.outcome = outcome;
    if (artifactDigest !== undefined) receipt.artifactDigest = artifactDigest;
    return receipt;
  });
  const coverage = payloadList(data.coverage, parseCoverageRow);
  const parsed: FeedVerification = {
    checks,
    checksTotal: payloadTotal(data.checks_total, checks.length),
    failures: payloadList(data.failures, parseCheckFailure),
    coverage,
    coverageTotal: payloadTotal(data.coverage_total, coverage.length),
    artifactsTotal: payloadCount(data.artifacts_total),
    evidenceTotal: payloadCount(data.evidence_total),
  };
  const outcome = payloadOptionalID(data.outcome);
  const deliveryId = payloadOptionalID(data.delivery_id);
  if (outcome !== undefined) parsed.outcome = outcome;
  if (deliveryId !== undefined) parsed.deliveryId = deliveryId;
  return parsed;
}

function parseDecompositionPayload(value: unknown): FeedDecomposition {
  const data = object(value);
  const candidates = payloadList(data.candidates, (row) => {
    const tasks = payloadList(row.tasks, parseIntentTask);
    const candidate: FeedDecompCandidate = {tasks, tasksTotal: payloadTotal(row.tasks_total, tasks.length)};
    const key = payloadOptionalID(row.key);
    const rationale = payloadText(row.rationale);
    if (key !== undefined) candidate.key = key;
    if (rationale !== undefined) candidate.rationale = rationale;
    return candidate;
  });
  const parsed: FeedDecomposition = {
    candidates,
    candidatesTotal: payloadTotal(data.candidates_total, candidates.length),
  };
  const operation = payloadOptionalID(data.operation);
  const attention = payloadText(data.attention);
  if (operation !== undefined) parsed.operation = operation;
  if (attention !== undefined) parsed.attention = attention;
  return parsed;
}

function parseJudgmentPayload(value: unknown): FeedJudgment {
  const data = object(value);
  check(data.kind === 'priority' || data.kind === 'delivery');
  const probabilities: Record<string, number> = {};
  if (data.probabilities !== undefined) {
    const distribution = object(data.probabilities);
    check(Object.keys(distribution).length <= 64);
    for (const [key, raw] of Object.entries(distribution)) {
      check(key.length > 0 && key.length <= 256);
      const weight = payloadProbability(raw);
      check(weight !== undefined);
      probabilities[key] = weight;
    }
  }
  const candidates = payloadList(data.candidates, (row) => {
    const candidate: FeedJudgmentCandidate = {id: payloadID(row.id)};
    const repository = payloadPositiveInt(row.repository);
    const issue = payloadPositiveInt(row.issue);
    const objective = payloadText(row.objective);
    if (repository !== undefined) candidate.repository = repository;
    if (issue !== undefined) candidate.issue = issue;
    if (objective !== undefined) candidate.objective = objective;
    return candidate;
  });
  const parsed: FeedJudgment = {
    kind: data.kind,
    probabilities,
    candidates,
    candidatesTotal: payloadTotal(data.candidates_total, candidates.length),
  };
  const selected = payloadOptionalID(data.selected);
  const outcome = payloadOptionalID(data.outcome);
  const effective = payloadOptionalID(data.effective);
  const model = payloadOptionalID(data.model);
  const probability = payloadProbability(data.probability);
  const confidence = payloadProbability(data.confidence);
  if (selected !== undefined) parsed.selected = selected;
  if (outcome !== undefined) parsed.outcome = outcome;
  if (effective !== undefined) parsed.effective = effective;
  if (model !== undefined) parsed.model = model;
  if (probability !== undefined) parsed.probability = probability;
  if (confidence !== undefined) parsed.confidence = confidence;
  return parsed;
}

function parseMergePayload(value: unknown): FeedMerge {
  const data = object(value);
  const parsed: FeedMerge = {};
  const stage = payloadOptionalID(data.stage);
  const publication = payloadOptionalID(data.publication);
  const mergedCommit = payloadOptionalID(data.merged_commit);
  const mergedUnix = payloadCount(data.merged_unix);
  const originalClosedUnix = payloadCount(data.original_closed_unix);
  if (stage !== undefined) parsed.stage = stage;
  if (publication !== undefined) parsed.publication = publication;
  if (mergedCommit !== undefined) parsed.mergedCommit = mergedCommit;
  if (mergedUnix > 0) parsed.mergedUnix = mergedUnix;
  if (originalClosedUnix > 0) parsed.originalClosedUnix = originalClosedUnix;
  return parsed;
}

function parseCompletionPayload(value: unknown): FeedCompletion {
  const data = object(value);
  const parsed: FeedCompletion = {};
  const event = payloadOptionalID(data.event);
  const publication = payloadOptionalID(data.publication);
  const occurrence = payloadOptionalID(data.occurrence);
  const mergedCommit = payloadOptionalID(data.merged_commit);
  const generation = payloadCount(data.generation);
  if (event !== undefined) parsed.event = event;
  if (publication !== undefined) parsed.publication = publication;
  if (occurrence !== undefined) parsed.occurrence = occurrence;
  if (mergedCommit !== undefined) parsed.mergedCommit = mergedCommit;
  if (generation > 0) parsed.generation = generation;
  return parsed;
}

function communicationText(value: unknown, limit = 2048): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(typeof value === 'string' && [...value].length <= limit);
  return value;
}

function parseCommunicationPayload(value: unknown): FeedCommunication {
  const data = object(value);
  const kind = feedCommunicationKinds.find((candidate) => candidate === data.kind);
  const title = communicationText(data.title, 160),
    summary = communicationText(data.summary);
  check(kind !== undefined && title !== undefined && summary !== undefined);
  const parsed: FeedCommunication = {
    kind,
    title,
    summary,
    evidenceRefs: payloadStrings(data.evidence_refs),
    options: payloadStrings(data.options),
    completed: payloadStrings(data.completed),
    remaining: payloadStrings(data.remaining),
    attempts: payloadStrings(data.attempts),
    alternatives: payloadStrings(data.alternatives),
    items: payloadList(data.items, (item) => {
      const label = payloadText(item.label);
      check(label !== undefined);
      const row: FeedGraphItem = {label};
      const itemID = payloadOptionalID(item.id);
      if (itemID !== undefined) row.id = itemID;
      if (item.depends_on !== undefined) row.dependsOn = payloadStrings(item.depends_on);
      const detail = communicationText(item.detail),
        state = payloadText(item.state);
      if (detail !== undefined) row.detail = detail;
      if (state !== undefined) row.state = state;
      return row;
    }),
  };
  if (data.commits !== undefined) {
    check(Array.isArray(data.commits) && data.commits.length <= 8);
    parsed.commits = payloadList(data.commits, (item) => {
      const revision = payloadText(item.revision),
        summary = communicationText(item.summary, 160);
      check(revision !== undefined && /^[a-f0-9]{40}$/.test(revision) && summary !== undefined);
      return {revision, summary};
    });
  }
  if (data.artifacts !== undefined)
    parsed.artifacts = payloadList(data.artifacts, (artifact) => {
      const artifactID = payloadOptionalID(artifact.id),
        name = payloadText(artifact.name);
      const mediaType = payloadText(artifact.media_type),
        href = payloadText(artifact.href);
      check(artifactID !== undefined && name !== undefined && mediaType !== undefined && href !== undefined);
      check(/^\/api\/factory\/artifacts\/[a-f0-9]{32}\?repository=[1-9][0-9]*$/.test(href));
      check(href.startsWith(`/api/factory/artifacts/${artifactID}?`));
      return {id: artifactID, name, mediaType, href};
    });
  if (data.source !== undefined) {
    const source = object(data.source),
      path = payloadText(source.path),
      code = communicationText(source.code, 3072);
    check(
      path !== undefined &&
        code !== undefined &&
        typeof source.start_line === 'number' &&
        Number.isInteger(source.start_line) &&
        source.start_line > 0
    );
    check(path.length <= 512 && code.length <= 3072 && code.split('\n').length <= 32);
    const startLine = source.start_line;
    const highlightLines = source.highlight_lines === undefined ? [] : source.highlight_lines;
    check(
      Array.isArray(highlightLines) &&
        highlightLines.length <= 32 &&
        highlightLines.every(
          (line) =>
            typeof line === 'number' &&
            Number.isInteger(line) &&
            line >= startLine &&
            line < startLine + code.split('\n').length
        )
    );
    parsed.source = {path, code, startLine: source.start_line, highlightLines};
    const revision = communicationText(source.revision, 80),
      after = communicationText(source.after, 3072);
    if (revision !== undefined) parsed.source.revision = revision;
    if (after !== undefined) {
      check(after.length <= 3072 && after.split('\n').length <= 32);
      parsed.source.after = after;
    }
  }
  for (const field of ['question', 'resolution', 'scope', 'decision', 'rationale', 'before', 'after'] as const) {
    const text = communicationText(data[field]);
    if (text !== undefined) parsed[field] = text;
  }
  const componentId = payloadOptionalID(data.component_id),
    replyTo = payloadOptionalID(data.reply_to);
  if (data.intent_source !== undefined) {
    const source = object(data.intent_source);
    const intent = payloadID(source.intent),
      revision = payloadID(source.revision),
      state = payloadText(source.state);
    check(state !== undefined && typeof source.owner === 'string');
    parsed.intentSource = {
      intent,
      revision,
      state,
      owner: source.owner,
      ...(typeof source.predecessor === 'string' ? {predecessor: source.predecessor} : {}),
    };
  }
  if (data.plan_source !== undefined) {
    const source = object(data.plan_source);
    if (
      typeof source.attempt_root !== 'string' ||
      !/^[a-f0-9]{32}$/.test(source.attempt_root) ||
      typeof source.plan !== 'string' ||
      !/^[a-f0-9]{32}$/.test(source.plan) ||
      typeof source.digest !== 'string' ||
      !/^[a-f0-9]{64}$/.test(source.digest) ||
      typeof source.source_commit !== 'string' ||
      !/^(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(source.source_commit) ||
      !Number.isInteger(source.generation) ||
      (source.generation as number) < 0
    )
      throw new Error('Invalid plan source.');
    parsed.planSource = {
      attemptRoot: source.attempt_root,
      plan: source.plan,
      digest: source.digest,
      source: source.source_commit,
      generation: source.generation as number,
    };
  }
  if (data.planning_source !== undefined) {
    const source = object(data.planning_source);
    if (
      typeof source.snapshot !== 'string' ||
      !/^[a-f0-9]{64}$/.test(source.snapshot) ||
      typeof source.record_digest !== 'string' ||
      !/^[a-f0-9]{64}$/.test(source.record_digest) ||
      typeof source.strategy !== 'string' ||
      typeof source.outcome !== 'string' ||
      !['planned', 'waiting', 'unavailable'].includes(source.outcome) ||
      typeof source.provisional !== 'boolean' ||
      typeof source.recorded_unix !== 'number' ||
      !Number.isSafeInteger(source.recorded_unix) ||
      source.recorded_unix <= 0
    )
      throw new Error('Invalid planning source.');
    parsed.planningSource = {
      snapshot: source.snapshot,
      recordDigest: source.record_digest,
      strategy: source.strategy,
      outcome: source.outcome,
      provisional: source.provisional,
      recordedUnix: source.recorded_unix,
      judgment: object(source.judgment),
    };
  }
  const libraryItemId = payloadOptionalID(data.library_item_id);
  if (libraryItemId !== undefined && /^[a-f0-9]{32}$/.test(libraryItemId)) parsed.libraryItemId = libraryItemId;
  if (typeof data.library_revision === 'number' && data.library_revision > 0)
    parsed.libraryRevision = data.library_revision;
  if (componentId !== undefined) parsed.componentId = componentId;
  if (replyTo !== undefined) parsed.replyTo = replyTo;
  if (data.question_state !== undefined) {
    check(
      data.question_state === 'open' ||
        data.question_state === 'answered' ||
        data.question_state === 'replied' ||
        data.question_state === 'incorporated'
    );
    parsed.questionState = data.question_state;
  }
  if (data.disposition !== undefined) {
    check(
      (kind === 'suggest' && (data.disposition === 'applied' || data.disposition === 'refused')) ||
        (kind === 'answer' && data.disposition === 'incorporated')
    );
    parsed.disposition = data.disposition;
    const reason = communicationText(data.disposition_reason);
    if (reason !== undefined) parsed.dispositionReason = reason;
  }
  if (data.severity !== undefined) {
    check(data.severity === 'info' || data.severity === 'warning' || data.severity === 'critical');
    parsed.severity = data.severity;
  }
  return parsed;
}

const feedPayloadFamilies = [
  'communication',
  'plan',
  'implementation',
  'review',
  'correction',
  'reassessment',
  'repair',
  'verification',
  'decomposition',
  'judgment',
  'merge',
  'completion',
] as const;

// parseFeedPayload admits one typed payload: absent stays absent, an
// empty object carries no typed content, and anything present but
// malformed degrades to {invalid: true}. At most one role family may
// be set; `tasks` may accompany only an implementation or correction.
export function parseFeedPayload(value: unknown): FeedPayload | undefined {
  if (value === undefined) return undefined;
  try {
    const data = object(value);
    if (Object.keys(data).length === 0) return undefined;
    const present = feedPayloadFamilies.filter((family) => data[family] !== undefined);
    check(present.length <= 1);
    const tasks = data.tasks === undefined ? undefined : parseTaskStatesPayload(data.tasks);
    if (present.length === 0 && tasks === undefined) return {invalid: true};
    const parsed: FeedPayload = {};
    const componentId = payloadOptionalID(data.component_id);
    if (componentId !== undefined) parsed.componentId = componentId;
    if (data.communication !== undefined) parsed.communication = parseCommunicationPayload(data.communication);
    if (data.plan !== undefined) parsed.plan = parsePlanPayload(data.plan);
    if (data.implementation !== undefined) parsed.implementation = parseImplementationPayload(data.implementation);
    if (data.review !== undefined) parsed.review = parseReviewPayload(data.review);
    if (data.correction !== undefined) parsed.correction = parseCorrectionPayload(data.correction);
    if (data.reassessment !== undefined) parsed.reassessment = parseReassessmentPayload(data.reassessment);
    if (data.repair !== undefined) parsed.repair = parseRepairPayload(data.repair);
    if (data.verification !== undefined) parsed.verification = parseVerificationPayload(data.verification);
    if (data.decomposition !== undefined) parsed.decomposition = parseDecompositionPayload(data.decomposition);
    if (data.judgment !== undefined) parsed.judgment = parseJudgmentPayload(data.judgment);
    if (data.merge !== undefined) parsed.merge = parseMergePayload(data.merge);
    if (data.completion !== undefined) parsed.completion = parseCompletionPayload(data.completion);
    if (tasks !== undefined) {
      check(parsed.implementation !== undefined || parsed.correction !== undefined);
      parsed.tasks = tasks;
    }
    return parsed;
  } catch {
    return {invalid: true};
  }
}

export function parseFeedEntry(value: unknown): FeedEntry {
  const entry = object(value);
  const kind = feedEntryKinds.find((candidate) => candidate === entry.kind);
  check(kind !== undefined && boundedID(entry.id) && positiveInt(entry.repository) && positiveInt(entry.seq));
  const parsed: FeedEntry = {kind, id: entry.id, repository: entry.repository, seq: entry.seq};
  const actor = optionalText(entry.actor, 128);
  const author = optionalText(entry.author, 128);
  const project = optionalText(entry.project, 256);
  const at = optionalStamp(entry.at);
  const status = optionalText(entry.status, 64);
  const issue = optionalPositiveInt(entry.issue);
  const pr = optionalPositiveInt(entry.pr);
  const ref = optionalText(entry.ref, 256);
  const summary = optionalSummary(entry.summary);
  const activity = entry.activity === undefined ? undefined : feedActivities.find((known) => known === entry.activity);
  check(entry.activity === undefined || activity !== undefined);
  const assignment = optionalText(entry.assignment, 256);
  const eventIdentity = optionalText(entry.event_identity, 160);
  const payload = parseFeedPayload(entry.payload);
  if (actor !== undefined) parsed.actor = actor;
  if (author !== undefined) parsed.author = author;
  if (project !== undefined) parsed.project = project;
  if (at !== undefined) parsed.at = at;
  if (status !== undefined) parsed.status = status;
  if (issue !== undefined) parsed.issue = issue;
  if (pr !== undefined) parsed.pr = pr;
  if (ref !== undefined) parsed.ref = ref;
  if (summary !== undefined) parsed.summary = summary;
  if (activity !== undefined) parsed.activity = activity;
  if (assignment !== undefined) parsed.assignment = assignment;
  if (eventIdentity !== undefined) parsed.eventIdentity = eventIdentity;
  if (payload !== undefined) parsed.payload = payload;
  return parsed;
}

function feedKindList(value: unknown): FeedEntryKind[] {
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length <= 8);
  return value.map((kind) => {
    const known = feedEntryKinds.find((candidate) => candidate === kind);
    check(known !== undefined);
    return known;
  });
}

function repositoryScope(value: unknown): number[] {
  // A fresh subscription reports no scope as null, never the widest scope.
  if (value === undefined || value === null) return [];
  check(Array.isArray(value) && value.length > 0 && value.length <= 1024);
  const seen = new Set<number>();
  for (const repository of value) {
    check(positiveInt(repository) && !seen.has(repository));
    seen.add(repository);
  }
  return [...seen];
}

export function parseFeedPageView(value: unknown): FeedPageView {
  const data = object(value);
  check(Array.isArray(data.entries) && data.entries.length <= feedPageMaxEntries);
  const entries = data.entries.map(parseFeedEntry);
  check(data.next === undefined || typeof data.next === 'string');
  const next = data.next === undefined ? '' : parseFeedCursor(data.next);
  check(typeof data.truncated === 'boolean');
  return {entries, next, truncated: data.truncated};
}

export function parseFeedSubscriptionView(value: unknown): FeedSubscriptionView {
  const data = object(value);
  check(typeof data.subscriber === 'string' && data.subscriber.length > 0 && data.subscriber.length <= 128);
  const repositories = repositoryScope(data.repositories);
  const kinds = feedKindList(data.kinds);
  check(typeof data.revision === 'number' && Number.isSafeInteger(data.revision) && data.revision >= 0);
  return {subscriber: data.subscriber, repositories, kinds, revision: data.revision};
}

export interface FeedQueryOptions {
  repositories: number[];
  kinds: FeedEntryKind[];
  limit: number;
  cursor: string;
  view?: string;
  search?: string;
  role?: string;
  current?: boolean;
}

// feedQueryString renders one feed query against the released route:
// comma-separated repository scope, optional kinds, a bounded window and
// the resume cursor. Empty cursor reads from the latest.
export function feedQueryString(query: FeedQueryOptions): string {
  check(query.repositories.length > 0 && query.repositories.length <= 1024);
  check(query.repositories.every(positiveInt));
  check(query.limit > 0 && query.limit <= feedPageLimit);
  const params = new URLSearchParams();
  params.set('repositories', [...new Set(query.repositories)].sort((a, b) => a - b).join(','));
  if (query.kinds.length > 0) params.set('kinds', query.kinds.join(','));
  params.set('limit', String(query.limit));
  if (query.cursor !== '') params.set('cursor', parseFeedCursor(query.cursor));
  if (query.view && query.view !== 'all') params.set('view', query.view);
  if (query.search) params.set('search', query.search);
  if (query.role) params.set('role', query.role);
  if (query.current) params.set('current', 'true');
  return `factory/feed?${params.toString()}`;
}

// Comment pipeline states behind one Lane B comment POST/GET: intake,
// judgment, route, draft, effect and the linked native issue. The POST
// answers as soon as the comment and its intake are durable — judgment,
// draft and effect normally still read pending — and progress is re-read
// through the comment GET. A retried submission reuses its idempotency
// key and recovers the original instead of duplicating it.
// CommentSource is the server-resolved snapshot of the feed entry one
// comment was submitted against. The server resolves the named entry
// inside the caller's visible repository at submission time; the
// frontend never supplies source facts beyond kind and identity.
export interface CommentSource {
  kind: FeedEntryKind;
  id: string;
  issue?: number;
  pr?: number;
  ref?: string;
  status?: string;
  summary?: string;
  // The exact accepted build-request revision this comment was discussed
  // against, as resolved by the canonical feed source resolver. Absent on
  // older comments recorded before the resolver supplied it.
  product?: {intent: string; acceptedRevision: string};
}

export interface CommentProductTaskEvidence {
  taskKey: string;
  issue: number;
  acceptance: string;
  completion?: string;
  // Explicit pending marker for a bound matching task with no canonical
  // completion row yet: the native binding is exact, but no delivery or
  // completion is inferred. Absent reads as not pending.
  pending?: boolean;
}

export interface CommentProductChange {
  intent: string;
  acceptedRevision: string;
  draftRevision?: string;
  // Review handoff states plus the explicit unknown: 'unavailable' is the
  // product-lane projection miss (lane known, detail unreadable). It never
  // asserts absence and never reads as success; a missing product_change on
  // the product lane renders the same way, never as detached progress.
  state: 'review_required' | 'draft' | 'accepted' | 'superseded' | 'unavailable';
  canReview: boolean;
  // Adopted lineage: the accepted descendant revision carrying the
  // deterministic comment idea, with its scope digest. Absent before any
  // adoption; preserved as history when a later head supersedes it.
  adoptedRevision?: string;
  scopeDigest?: string;
  // Controller-owned F10 fact for the adopted revision and scope digest.
  // False while unknown; never inferred from closed issues or worker exits.
  verified: boolean;
  // Canonical work-evidence read state, reusing the linked-work
  // vocabulary: recorded when the adopted delivered graph read back,
  // unavailable while unknown. Evidence and native issue identity stay
  // separate fields on each task row.
  evidenceStatus: 'recorded' | 'unavailable';
  evidence: CommentProductTaskEvidence[];
}

export interface CommentPipelineState {
  productChange?: CommentProductChange;
  linkedWork?: {
    status: string;
    issue: number;
    adoption?: {state: string; actor: string; assignment: string; reason: string};
    work: {assignment: string; phase: string; stage: string; outcome: string; sourceRevision: string}[];
    completion: string;
  };
  detail?: string;
  canRetry?: boolean;
  attempts?: number;
  attemptLimit?: number;
  resume?: boolean;
  commentId: string;
  revision: number;
  intake: string;
  judgment: string;
  route: string;
  draft: string;
  effect: string;
  issue: number;
  // The already-tracked issue an existing_work judgment selected, as
  // its own served field; 0 until that link is persisted.
  existingIssue: number;
  source?: CommentSource;
}

function commentID(value: unknown): value is string {
  return typeof value === 'string' && /^[a-f0-9]{32}$/.test(value);
}

function parseCommentSource(value: unknown): CommentSource | undefined {
  if (value === undefined || value === null) return undefined;
  const data = object(value);
  const kind = feedEntryKinds.find((candidate) => candidate === data.kind);
  check(kind !== undefined && commentID(data.id));
  const parsed: CommentSource = {kind, id: data.id};
  const issue = payloadPositiveInt(data.issue);
  const pr = payloadPositiveInt(data.pr);
  const ref = data.ref === undefined || data.ref === '' ? undefined : optionalText(data.ref, 256);
  const status = data.status === undefined || data.status === '' ? undefined : optionalText(data.status, 128);
  const summary = data.summary === undefined || data.summary === '' ? undefined : optionalSummary(data.summary);
  if (issue !== undefined) parsed.issue = issue;
  if (pr !== undefined) parsed.pr = pr;
  if (ref !== undefined) parsed.ref = ref;
  if (status !== undefined) parsed.status = status;
  if (summary !== undefined) parsed.summary = summary;
  // The canonical resolver names the exact accepted product discussed; a
  // caller-supplied value never reaches here because the server resolves it.
  if (data.product !== undefined) {
    const product = object(data.product);
    check(decisionID(product.intent) && decisionID(product.accepted_revision));
    parsed.product = {intent: product.intent, acceptedRevision: product.accepted_revision};
  }
  return parsed;
}

// commentSourceForEntry names the feed entry one reply answers: a post
// by its comment identity, a record by its assignment identity —
// milestone posts name their bound assignment, since the server
// resolves records through the assignment owner. Entries without a
// resolvable 32-hex identity carry no source.
export function commentSourceForEntry(entry: FeedEntry): {kind: FeedEntryKind; id: string} | undefined {
  if (entry.kind === 'post') return commentID(entry.id) ? {kind: 'post', id: entry.id} : undefined;
  const assignment = entry.assignment ?? entry.id;
  return commentID(assignment) ? {kind: 'record', id: assignment} : undefined;
}

export function parseCommentPipelineState(value: unknown): CommentPipelineState {
  const data = object(value);
  const comment = typeof data.comment_id === 'string' ? data.comment_id : data.comment;
  const {revision, intake, judgment, route, draft, effect, issue} = data;
  check(commentID(comment));
  check(typeof revision === 'number' && Number.isSafeInteger(revision) && revision >= 0);
  check(typeof intake === 'string' && intake.length > 0 && intake.length <= 32);
  check(typeof judgment === 'string' && judgment.length > 0 && judgment.length <= 32);
  check(typeof draft === 'string' && draft.length > 0 && draft.length <= 32);
  check(typeof effect === 'string' && effect.length > 0 && effect.length <= 32);
  check(route === undefined || (typeof route === 'string' && route.length <= 32));
  check(issue === undefined || (typeof issue === 'number' && Number.isSafeInteger(issue) && issue >= 0));
  // Additive persisted selected-issue link for the existing_work route:
  // the backend may serve it as its own field before folding it into
  // `issue`. Tolerate either spelling; never invent one.
  const selected = data.existing_issue ?? data.selected_issue;
  check(selected === undefined || (typeof selected === 'number' && Number.isSafeInteger(selected) && selected >= 0));
  const linkedIssue = typeof issue === 'number' && issue > 0 ? issue : typeof selected === 'number' ? selected : 0;
  // The same-product lane never creates a native issue: a product change
  // row with a linked issue is not the atomic comment-to-draft handoff.
  check(data.product_change === undefined || linkedIssue === 0);
  const source = parseCommentSource(data.source);
  const attempts = data.attempts;
  const attemptLimit = data.attempt_limit;
  check(attempts === undefined || (typeof attempts === 'number' && Number.isSafeInteger(attempts) && attempts >= 0));
  check(
    attemptLimit === undefined ||
      (typeof attemptLimit === 'number' && Number.isSafeInteger(attemptLimit) && attemptLimit > 0)
  );
  check(attempts === undefined || attemptLimit === undefined || attempts <= attemptLimit);
  check(data.resume === undefined || typeof data.resume === 'boolean');
  const parsed: CommentPipelineState = {
    commentId: comment,
    revision,
    intake,
    judgment,
    route: typeof route === 'string' ? route : '',
    draft,
    effect,
    issue: linkedIssue,
    existingIssue: typeof selected === 'number' ? selected : 0,
    ...(typeof data.detail === 'string' && data.detail ? {detail: data.detail} : {}),
    ...(typeof data.can_retry === 'boolean' ? {canRetry: data.can_retry} : {}),
    ...(typeof data.resume === 'boolean' ? {resume: data.resume} : {}),
  };
  if (source !== undefined) parsed.source = source;
  if (typeof attempts === 'number') parsed.attempts = attempts;
  if (typeof attemptLimit === 'number') parsed.attemptLimit = attemptLimit;
  if (data.product_change !== undefined) parsed.productChange = parseCommentProductChange(data.product_change);
  if (data.linked_work !== undefined) {
    const trace = object(data.linked_work);
    check(trace.status === 'recorded' || trace.status === 'unavailable');
    const work = payloadList(trace.work, (entry) => ({
      assignment: payloadID(entry.assignment),
      phase: payloadText(entry.phase) ?? '',
      stage: payloadText(entry.stage) ?? '',
      outcome: payloadText(entry.outcome) ?? '',
      sourceRevision: payloadText(entry.source_revision) ?? '',
    }));
    const adoption = object(trace.adoption ?? {});
    parsed.linkedWork = {
      status: trace.status,
      issue: typeof trace.issue === 'number' ? trace.issue : 0,
      work,
      completion: payloadText(trace.completion) ?? '',
      ...(typeof adoption.state === 'string'
        ? {
            adoption: {
              state: adoption.state,
              actor: payloadText(adoption.actor) ?? '',
              assignment: payloadText(adoption.assignment) ?? '',
              reason: payloadText(adoption.reason) ?? '',
            },
          }
        : {}),
    };
  }
  return parsed;
}

const completionEventID = (value: unknown): value is string =>
  typeof value === 'string' && /^[a-f0-9]{32}$/.test(value);
const scopeDigest = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);

function parseCommentProductEvidence(value: unknown): CommentProductTaskEvidence {
  const data = object(value);
  check(decisionID(data.task_key));
  check(positiveInt(data.issue));
  check(decisionID(data.acceptance));
  check(data.completion === undefined || completionEventID(data.completion));
  check(data.pending === undefined || typeof data.pending === 'boolean');
  return {
    taskKey: data.task_key,
    issue: data.issue,
    acceptance: data.acceptance,
    ...(typeof data.completion === 'string' ? {completion: data.completion} : {}),
    ...(typeof data.pending === 'boolean' ? {pending: data.pending} : {}),
  };
}

// commentProductLaneWithoutProjection names a same-product comment whose
// detail did not arrive: an accepted source behind an applied new-issue
// pipeline with no linked native issue, but no product_change row. Missing
// data here is an unavailable projection, never an absent change.
export function commentProductLaneWithoutProjection(state: CommentPipelineState): boolean {
  return (
    state.productChange === undefined &&
    state.source?.product !== undefined &&
    state.judgment === 'applied' &&
    state.route === 'new_issue' &&
    state.issue === 0 &&
    state.existingIssue === 0
  );
}

function parseCommentProductChange(value: unknown): CommentProductChange {
  const data = object(value);
  check(decisionID(data.intent) && decisionID(data.accepted_revision));
  check(data.draft_revision === undefined || decisionID(data.draft_revision));
  check(
    data.state === 'review_required' ||
      data.state === 'draft' ||
      data.state === 'accepted' ||
      data.state === 'superseded' ||
      data.state === 'unavailable'
  );
  check(typeof data.can_review === 'boolean');
  check(
    data.state === 'review_required' ||
      data.state === 'superseded' ||
      data.state === 'unavailable' ||
      decisionID(data.draft_revision)
  );
  // Adopted lineage is separate from the review handoff: an adopted
  // revision and scope digest arrive together, verified is always an
  // explicit boolean, and evidence status reuses the linked-work
  // recorded/unavailable vocabulary. Unknown work never parses as success.
  const adopted = data.adopted_revision === undefined ? undefined : data.adopted_revision;
  const digest = data.scope_digest === undefined ? undefined : data.scope_digest;
  check(adopted === undefined || decisionID(adopted));
  check(digest === undefined || scopeDigest(digest));
  check(digest === undefined || adopted !== undefined);
  check(typeof data.verified === 'boolean');
  check(data.evidence_status === 'recorded' || data.evidence_status === 'unavailable');
  check(data.evidence_status === 'unavailable' || adopted !== undefined);
  // Canonical delivered tasks are bounded like the serving graph; an
  // absent list reads as no recorded task, never as verified work.
  const evidence = data.evidence === undefined ? [] : data.evidence;
  check(Array.isArray(evidence) && evidence.length <= 64);
  return {
    intent: data.intent,
    acceptedRevision: data.accepted_revision,
    state: data.state,
    canReview: data.can_review,
    verified: data.verified,
    evidenceStatus: data.evidence_status,
    evidence: evidence.map(parseCommentProductEvidence),
    ...(typeof data.draft_revision === 'string' ? {draftRevision: data.draft_revision} : {}),
    ...(typeof adopted === 'string' ? {adoptedRevision: adopted} : {}),
    ...(typeof digest === 'string' ? {scopeDigest: digest} : {}),
  };
}

export interface CommentPostBody {
  repository: number;
  body: string;
  issue?: number;
  reply_to?: string;
  source_kind?: FeedEntryKind;
  source_id?: string;
  idempotency_key: string;
}

export interface CommentPostOptions {
  repository: number;
  body: string;
  idempotencyKey: string;
  issue?: number;
  replyTo?: string;
  source?: {kind: FeedEntryKind; id: string};
}

// commentPostBody builds one comment POST body against the released Lane B
// contract: repository scope, bounded text, optional bound issue and reply
// parent, and the caller-chosen idempotency key. The author is the
// host-admitted caller, never a body field.
export function commentPostBody(options: CommentPostOptions): CommentPostBody {
  check(positiveInt(options.repository));
  check(options.body.trim() !== '' && options.body.length <= 4096);
  check(/^[A-Za-z0-9_-]{1,64}$/.test(options.idempotencyKey));
  const post: CommentPostBody = {
    repository: options.repository,
    body: options.body,
    idempotency_key: options.idempotencyKey,
  };
  if (options.issue !== undefined) {
    check(Number.isSafeInteger(options.issue) && options.issue >= 0);
    post.issue = options.issue;
  }
  if (options.replyTo !== undefined) {
    check(/^[a-f0-9]{32}$/.test(options.replyTo));
    post.reply_to = options.replyTo;
  }
  if (options.source !== undefined) {
    // Source kind and identity travel together or not at all; the
    // server resolves the named entry and refuses invisibles.
    check(feedEntryKinds.includes(options.source.kind) && commentID(options.source.id));
    post.source_kind = options.source.kind;
    post.source_id = options.source.id;
  }
  return post;
}

export interface CommentDetail {
  commentId: string;
  pipeline?: CommentPipelineState;
  body: string;
  authorName: string;
  at: number;
  edited: boolean;
  removed: boolean;
  actor?: string;
  assignment?: string;
  replyTo?: string;
  source?: CommentPipelineState['source'];
  communication?: FeedCommunication;
  consultation?: FeedConsultation;
  contribution?: FeedContribution;
}

export interface FeedContribution {
  source: string;
  revision: number;
  state: string;
  profile?: string;
  reply?: string;
  body: string;
  context?: string;
  roles: unknown[];
  replies: unknown[];
  calls?: {
    assignment: string;
    question: string;
    answer: {selected: string; probabilities: Record<string, number>; confidence: number; model: string};
  }[];
  minimum_probability: number;
  minimum_confidence: number;
  probability_threshold?: number;
  confidence_threshold?: number;
  policy_version?: string;
}

function parseContribution(value: unknown): FeedContribution {
  const data = object(value);
  check(
    commentID(data.source) &&
      typeof data.revision === 'number' &&
      typeof data.state === 'string' &&
      typeof data.body === 'string'
  );
  check(Array.isArray(data.roles) && Array.isArray(data.replies));
  for (const role of data.roles) object(role);
  if (data.calls !== undefined) {
    check(Array.isArray(data.calls) && data.calls.length <= 12);
    for (const value of data.calls) {
      const call = object(value),
        answer = object(call.answer);
      check(
        typeof call.assignment === 'string' &&
          typeof call.question === 'string' &&
          typeof answer.selected === 'string' &&
          typeof answer.confidence === 'number' &&
          typeof answer.model === 'string'
      );
      object(answer.probabilities);
    }
  }
  return data as unknown as FeedContribution;
}

export interface FeedConsultation {
  snapshot: string;
  group_id: string;
  assignment: string;
  source_revision?: string;
  tool: string;
  kind: string;
  state?: unknown;
  instructions?: unknown;
  criteria?: unknown;
  questions?: Record<string, unknown>;
  model?: string;
  status: 'pending' | 'returned' | 'failed' | 'unavailable' | 'uncertain';
  answer?: Record<string, unknown>;
  created_unix: number;
  updated_unix: number;
}

function parseConsultation(value: unknown): FeedConsultation {
  const data = object(value);
  check(typeof data.snapshot === 'string' && /^[a-f0-9]{64}$/.test(data.snapshot));
  check(typeof data.group_id === 'string' && typeof data.assignment === 'string');
  check(typeof data.tool === 'string' && typeof data.kind === 'string');
  check(['pending', 'returned', 'failed', 'unavailable', 'uncertain'].includes(String(data.status)));
  check(typeof data.created_unix === 'number' && typeof data.updated_unix === 'number');
  check(data.status !== 'returned' || (data.answer !== null && typeof data.answer === 'object'));
  return data as unknown as FeedConsultation;
}

// commentTombstoneOf names an authoritative tombstone marker on one direct
// comment GET response: the explicit deleted/revoked flags only. A bodyless
// payload without those flags proves nothing — it may be malformed or
// truncated — and a missing page row alone never counts. Pipeline-only rows
// (body absent but judgment present) are live progress, never tombstones.
export function commentTombstoneOf(value: unknown): {commentId: string} | undefined {
  if (typeof value !== 'object' || value === null) return undefined;
  const data = value as Record<string, unknown>;
  const id = data.comment_id;
  if (!commentID(id)) return undefined;
  if (data.deleted === true || data.revoked === true) return {commentId: id};
  return undefined;
}

// parseCommentDetail admits one comment GET response: the stored pipeline
// plus the display facts. Deleted or revoked sources arrive bodyless; the
// tombstone still resolves through the pipeline states.
export function parseCommentDetail(value: unknown): CommentDetail {
  const data = object(value);
  check(commentID(data.comment_id));
  // Bodyless tombstones carry no body and no pipeline, but only with an
  // explicit deleted/revoked flag do they resolve to a removed marker. A
  // bodyless payload without those flags is malformed and throws instead of
  // reading as a deletion. Missing-row absence alone never reaches here.
  if (data.body === undefined && data.judgment === undefined) {
    check(data.deleted === true || data.revoked === true);
    return {
      commentId: data.comment_id,
      body: '',
      authorName: typeof data.author_name === 'string' ? data.author_name : '',
      at: typeof data.at === 'number' ? data.at : 0,
      edited: false,
      removed: true,
    };
  }
  const pipeline = data.judgment === undefined ? undefined : parseCommentPipelineState(value);
  const {body, author_name, at, edited, deleted, revoked} = data;
  const removed = deleted === true || revoked === true;
  check(removed || (typeof body === 'string' && body.length <= 4096));
  check(body === undefined || (typeof body === 'string' && body.length <= 4096));
  check(author_name === undefined || (typeof author_name === 'string' && author_name.length <= 128));
  check(at === undefined || (typeof at === 'number' && Number.isSafeInteger(at) && at >= 0));
  check(edited === undefined || typeof edited === 'boolean');
  check(deleted === undefined || typeof deleted === 'boolean');
  check(revoked === undefined || typeof revoked === 'boolean');
  return {
    commentId: data.comment_id,
    ...(pipeline ? {pipeline} : {}),
    body: typeof body === 'string' ? body : '',
    authorName: typeof author_name === 'string' ? author_name : '',
    at: typeof at === 'number' ? at : 0,
    edited: edited === true,
    removed,
    ...(typeof data.actor === 'string' ? {actor: data.actor} : {}),
    ...(typeof data.assignment === 'string' && data.assignment !== '' ? {assignment: payloadID(data.assignment)} : {}),
    ...(typeof data.reply_to === 'string' && data.reply_to !== '' ? {replyTo: payloadID(data.reply_to)} : {}),
    ...(data.source ? {source: parseCommentSource(data.source)} : {}),
    ...(data.communication ? {communication: parseCommunicationPayload(data.communication)} : {}),
    ...(data.consultation ? {consultation: parseConsultation(data.consultation)} : {}),
    ...(data.contribution ? {contribution: parseContribution(data.contribution)} : {}),
  };
}

// newIdempotencyKey mints one caller-chosen key per draft. Keystrokes reuse
// it so every submit recovers the same comment; starting a fresh reply mints
// a fresh key.
export function newIdempotencyKey(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return [...bytes]
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .slice(0, 32)
    .join('');
}

// commentUnsettled reports whether one pipeline still advances: judgment,
// draft or native effect pending behind a live route, or an existing_work
// route whose persisted selected-issue link has not landed yet. Settled
// outcomes (verified/applied/blocked/failed/discussion/clarify) need no
// polling; reads never drive them further.
export function commentAttemptsExhausted(state: CommentPipelineState): boolean {
  return (
    !state.resume &&
    state.attempts !== undefined &&
    state.attemptLimit !== undefined &&
    state.attempts >= state.attemptLimit
  );
}

export function commentUnsettled(state: CommentPipelineState): boolean {
  if (state.productChange) return false;
  if (commentAttemptsExhausted(state)) return false;
  if (state.judgment === 'pending') return true;
  if (state.judgment === 'failed') return false;
  if (state.route === 'new_issue') {
    if (state.draft === 'pending') return true;
    if (state.draft === 'invalid') return false;
    return state.effect === 'pending';
  }
  if (state.route === 'existing_work') return state.effect === 'pending';
  return false;
}

// Recorded status segments behind one feed record, as composed by the
// canonical record producer: `phase:stage[:outcome[:verdict]]` followed
// by `>`-joined extras (`findings:N`, `checks:N`, `failures:N`, the
// publication stage, the merge stage, `completed`). Every segment is a
// recorded value; parsing only splits them back apart for structured
// rendering. Unrecognised shapes keep the raw status and no segments,
// so nothing is inferred beyond what was recorded.
export interface FeedStatusSegments {
  readonly raw: string;
  readonly phase: string | undefined;
  readonly stage: string | undefined;
  readonly outcome: string | undefined;
  readonly verdict: string | undefined;
  readonly findings: number | undefined;
  readonly checks: number | undefined;
  readonly failures: number | undefined;
  // Remaining extras in recorded order: publication/merge stages and
  // the verified-completion marker, e.g. ['published', 'merged', 'completed'].
  readonly chain: readonly string[];
}

function segmentCount(extras: string[], prefix: string): number | undefined {
  const hit = extras.find((extra) => extra.startsWith(prefix));
  if (hit === undefined) return undefined;
  const value = Number(hit.slice(prefix.length));
  return Number.isSafeInteger(value) && value >= 0 ? value : undefined;
}

export function parseFeedStatusSegments(status: string | undefined): FeedStatusSegments {
  const raw = status ?? '';
  const empty: FeedStatusSegments = {
    raw,
    phase: undefined,
    stage: undefined,
    outcome: undefined,
    verdict: undefined,
    findings: undefined,
    checks: undefined,
    failures: undefined,
    chain: [],
  };
  if (raw.trim() === '') return empty;
  const [base = '', ...extras] = raw.split('>').map((part) => part.trim());
  const parts = base.split(':').map((part) => part.trim());
  if (parts.length < 2 || parts.some((part) => part === '')) return empty;
  const counted = new Set(['findings:', 'checks:', 'failures:']);
  return {
    raw,
    phase: parts[0],
    stage: parts[1],
    outcome: parts[2],
    verdict: parts[3],
    findings: segmentCount(extras, 'findings:'),
    checks: segmentCount(extras, 'checks:'),
    failures: segmentCount(extras, 'failures:'),
    chain: extras.filter((extra) => extra !== '' && ![...counted].some((prefix) => extra.startsWith(prefix))),
  };
}

// mergeFeedEntries folds one older page behind the confirmed entries:
// newest-first by each entry's global order key, stable identities
// kept once and fresh recorded fields replacing their previous values.
export function mergeFeedEntries(confirmed: FeedEntry[], page: FeedEntry[]): FeedEntry[] {
  const byKey = new Map(confirmed.map((entry) => [feedEntryKey(entry), entry]));
  for (const entry of page) byKey.set(feedEntryKey(entry), entry);
  const merged = [...byKey.values()];
  merged.sort((a, b) => {
    if (a.seq !== b.seq) return b.seq - a.seq;
    return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
  });
  return merged;
}
