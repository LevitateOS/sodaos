import {check, id, object} from './sodaspaces-api.js';
import {decisionID} from './sodaspaces-intent-response.js';

// Parsers for the work-journey read: one issue's recorded phases, tasks
// with lanes, commit/PR evidence, structured findings with corrections,
// Jev judgments with effective waits, D13 completion gates with proposals
// and adoption, delivered-task evidence behind each adoption binding, and
// the verified application revision behind each adopted revision. Each
// parser binds its exact repository/issue identity; absent sections read
// as empty, never as guessed work. Evidence identities stay text here:
// native navigation links are built only in the view layer.

const commitID = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{40}$/.test(value);
const digest = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);

function counter(value: unknown): number {
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= 0);
  return value;
}

function fraction(value: unknown): number {
  check(typeof value === 'number' && !Number.isNaN(value) && value >= 0 && value <= 1);
  return value;
}

function boundedText(value: unknown, limit: number): string {
  check(typeof value === 'string' && value.length > 0 && value.length <= limit);
  check(!value.split('').some((char) => char < ' ' || char === '\u007f'));
  return value;
}

function optionalText(value: unknown, limit: number): string | undefined {
  if (value === undefined || value === '') return undefined;
  return boundedText(value, limit);
}

function closedSet(value: unknown, allowed: readonly string[]): string {
  check(typeof value === 'string' && allowed.includes(value));
  return value;
}

export interface JourneyPhase {
  readonly phase: string;
  readonly state: string;
}

export interface JourneyTask {
  readonly key: string;
  readonly lane: string;
  readonly state: string;
  readonly issue?: string;
}

export interface JourneyCommit {
  readonly commit: string;
  readonly base: string;
  readonly pr_number: string;
  readonly pr_id: string;
}

export interface JourneyFinding {
  readonly key: string;
  readonly severity: string;
  readonly defect: string;
  readonly fix_kind: string;
}

export interface JourneyCorrection {
  readonly candidate?: string;
  readonly effect?: string;
  readonly completion?: string;
  readonly reason?: string;
}

export interface JourneyJudgment {
  readonly cycle_id?: string;
  readonly cycle_state?: string;
  readonly kind: string;
  readonly selected: string;
  readonly wait: boolean;
  readonly model?: string;
  readonly confidence?: number;
  readonly probability?: number;
  readonly wait_reason?: string;
}

export interface JourneyGate {
  readonly completion_id: string;
  readonly cycle_id: string;
  readonly state: string;
  readonly repository: string;
  readonly generation: number;
  readonly retryable: boolean;
  readonly decision?: string;
  readonly proposal?: string;
  readonly wait_reason?: string;
  readonly proposal_digest?: string;
  readonly inventory_digest?: string;
  readonly receipt_digests: readonly string[];
}

export interface JourneyAdoption {
  readonly intent: string;
  readonly revision: string;
  readonly state: string;
  readonly task: string;
  readonly issue_id: string;
  readonly acceptance: string;
}

export interface JourneyDeliveredTask {
  readonly task: string;
  readonly issue: string;
  readonly acceptance: string;
  readonly completion: string;
  readonly publication: string;
  readonly generation: number;
}

export interface JourneyVerifiedApp {
  readonly intent: string;
  readonly revision: string;
  readonly source_commit: string;
  readonly manifest: string;
}

export interface WorkJourney {
  readonly repository: string;
  readonly issue: string;
  readonly observed_unix: number;
  readonly phases: readonly JourneyPhase[];
  readonly tasks: readonly JourneyTask[];
  readonly commits: readonly JourneyCommit[];
  readonly findings: readonly JourneyFinding[];
  readonly corrections: readonly JourneyCorrection[];
  readonly judgments: readonly JourneyJudgment[];
  readonly gates: readonly JourneyGate[];
  readonly adoptions: readonly JourneyAdoption[];
  readonly delivered_tasks: readonly JourneyDeliveredTask[];
  readonly verifications: readonly JourneyVerifiedApp[];
}

const executionPhases = ['decompose', 'investigate', 'implement', 'review', 'correct', 'reassess'];
const journeyStates = ['pending', 'active', 'complete', 'attention'];
const journeyLanes = ['a', 'b', 'c', 'unassigned'];
const findingSeverities = ['critical', 'major', 'minor'];
const findingFixKinds = ['code', 'verification'];
const opEffects = ['pending', 'committed', 'not_committed', 'indeterminate'];
const opCompletions = ['pending', 'complete', 'needs_intervention'];
const judgmentKinds = ['priority', 'delivery', 'reassessment-decision', 'reassessment-proposal'];
const deliveryChoices = ['review_again', 'ready_to_merge'];
const reassessDecisions = ['reassess', 'continue'];
const reassessProposals = ['accept', 'needs_attention'];
const gateStates = [
  'collecting',
  'deciding',
  'reassessing',
  'assessing',
  'applying',
  'released',
  'attention',
  'superseded',
];
const adoptionStates = ['proposed', 'adopted', 'released'];

function journeyPhase(value: unknown): JourneyPhase {
  const raw = object(value);
  return {phase: closedSet(raw.phase, executionPhases), state: closedSet(raw.state, journeyStates)};
}

function journeyTask(value: unknown): JourneyTask {
  const raw = object(value);
  const issue = optionalText(raw.issue, 19);
  if (issue !== undefined) check(id(issue));
  return {
    key: boundedText(raw.key, 128),
    lane: closedSet(raw.lane, journeyLanes),
    state: closedSet(raw.state, journeyStates),
    ...(issue === undefined ? {} : {issue}),
  };
}

function journeyCommit(value: unknown): JourneyCommit {
  const raw = object(value);
  check(commitID(raw.commit) && commitID(raw.base) && raw.commit !== raw.base);
  check(id(raw.pr_number) && id(raw.pr_id));
  return {
    commit: raw.commit as string,
    base: raw.base as string,
    pr_number: raw.pr_number as string,
    pr_id: raw.pr_id as string,
  };
}

function journeyFinding(value: unknown): JourneyFinding {
  const raw = object(value);
  check(typeof raw.key === 'string' && /^[a-f0-9]{32}$/.test(raw.key));
  return {
    key: raw.key,
    severity: closedSet(raw.severity, findingSeverities),
    defect: boundedText(raw.defect, 4096),
    fix_kind: closedSet(raw.fix_kind, findingFixKinds),
  };
}

function journeyCorrection(value: unknown): JourneyCorrection {
  const raw = object(value);
  const candidate = optionalText(raw.candidate, 40);
  if (candidate !== undefined) check(commitID(candidate));
  const effect = optionalText(raw.effect, 16);
  if (effect !== undefined) check(opEffects.includes(effect));
  const completion = optionalText(raw.completion, 32);
  if (completion !== undefined) check(opCompletions.includes(completion));
  return {
    ...(candidate === undefined ? {} : {candidate}),
    ...(effect === undefined ? {} : {effect}),
    ...(completion === undefined ? {} : {completion}),
    ...(optionalText(raw.reason, 256) === undefined ? {} : {reason: raw.reason as string}),
  };
}

function judgmentChoice(kind: string, selected: unknown): string {
  if (kind === 'priority') return boundedText(selected, 128);
  if (kind === 'delivery') return closedSet(selected, deliveryChoices);
  if (kind === 'reassessment-decision') return closedSet(selected, reassessDecisions);
  return closedSet(selected, reassessProposals);
}

function judgmentThreshold(kind: string, confidence: number, probability: number): void {
  if (kind === 'reassessment-decision') check(confidence >= 0.5);
  if (kind === 'reassessment-proposal') check(probability >= 0.8 && confidence >= 0.8);
}

function judgmentScores(kind: string, raw: Record<string, unknown>, closed: boolean): {confidence?: number; probability?: number} {
  const confidence = raw.confidence === undefined ? undefined : fraction(raw.confidence);
  const probability = raw.probability === undefined ? undefined : fraction(raw.probability);
  if (raw.wait !== true && !closed) judgmentThreshold(kind, confidence ?? 0, probability ?? 0);
  return {
    ...(confidence === undefined ? {} : {confidence}),
    ...(probability === undefined ? {} : {probability}),
  };
}

function journeyJudgment(value: unknown): JourneyJudgment {
  const raw = object(value);
  const kind = closedSet(raw.kind, judgmentKinds);
  const selected = judgmentChoice(kind, raw.selected);
  const cycleID = optionalText(raw.cycle_id, 32), cycleState = optionalText(raw.cycle_state, 32);
  check((cycleID === undefined) === (cycleState === undefined));
  if (cycleID !== undefined) {
    check(/^[a-f0-9]{32}$/.test(cycleID) && kind.startsWith('reassessment-'));
    closedSet(cycleState, gateStates);
  }
  const closed = cycleState === 'superseded' || cycleState === 'released';
  const scores = judgmentScores(kind, raw, closed);
  const model = optionalText(raw.model, 128);
  check(typeof raw.wait === 'boolean');
  if (closed) check(raw.wait === false);
  const waitReason = optionalText(raw.wait_reason, 256);
  if (raw.wait === true) check(waitReason !== undefined);
  else check(waitReason === undefined);
  return {
    kind,
    selected,
    wait: raw.wait as boolean,
    ...(cycleID === undefined ? {} : {cycle_id: cycleID, cycle_state: cycleState!}),
    ...(model === undefined ? {} : {model}),
    ...scores,
    ...(waitReason === undefined ? {} : {wait_reason: waitReason}),
  };
}

function gateDigest(value: unknown): string | undefined {
  if (value === undefined || value === '') return undefined;
  check(digest(value));
  return value as string;
}

interface GateScope {
  readonly completion_id: string;
  readonly cycle_id: string;
  readonly state: string;
  readonly repository: string;
  readonly generation: number;
  readonly retryable: boolean;
}

function gateScope(raw: Record<string, unknown>): GateScope {
  check(typeof raw.completion_id === 'string' && /^[a-f0-9]{32}$/.test(raw.completion_id));
  check(typeof raw.cycle_id === 'string' && /^[a-f0-9]{32}$/.test(raw.cycle_id));
  const generation = counter(raw.generation);
  check(generation >= 1 && id(raw.repository) && typeof raw.retryable === 'boolean');
  return {
    completion_id: raw.completion_id as string,
    cycle_id: raw.cycle_id as string,
    state: closedSet(raw.state, gateStates),
    repository: raw.repository as string,
    generation,
    retryable: raw.retryable as boolean,
  };
}

interface GateVerdict {
  readonly decision?: string;
  readonly proposal?: string;
  readonly wait_reason?: string;
}

function gateVerdict(raw: Record<string, unknown>, state: string, retryable: boolean): GateVerdict {
  const decision = optionalText(raw.decision, 32);
  if (decision !== undefined) check(reassessDecisions.includes(decision));
  const proposal = optionalText(raw.proposal, 32);
  if (proposal !== undefined) check(reassessProposals.includes(proposal));
  const waitReason = optionalText(raw.wait_reason, 256);
  if (state === 'attention') check(waitReason !== undefined);
  if (retryable) check(state === 'attention');
  return {
    ...(decision === undefined ? {} : {decision}),
    ...(proposal === undefined ? {} : {proposal}),
    ...(waitReason === undefined ? {} : {wait_reason: waitReason}),
  };
}

interface GateEvidence {
  readonly proposal_digest?: string;
  readonly inventory_digest?: string;
  readonly receipt_digests: readonly string[];
}

function gateEvidence(raw: Record<string, unknown>): GateEvidence {
  const receipts = raw.receipt_digests === undefined ? [] : raw.receipt_digests;
  check(Array.isArray(receipts) && receipts.length <= 8);
  for (const receipt of receipts) check(digest(receipt));
  return {
    ...(gateDigest(raw.proposal_digest) === undefined ? {} : {proposal_digest: raw.proposal_digest as string}),
    ...(gateDigest(raw.inventory_digest) === undefined ? {} : {inventory_digest: raw.inventory_digest as string}),
    receipt_digests: receipts as string[],
  };
}

function journeyGate(value: unknown): JourneyGate {
  const raw = object(value);
  const scope = gateScope(raw);
  return {...scope, ...gateVerdict(raw, scope.state, scope.retryable), ...gateEvidence(raw)};
}

function journeyAdoption(value: unknown): JourneyAdoption {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.revision) && decisionID(raw.acceptance));
  check(id(raw.issue_id));
  return {
    intent: raw.intent as string,
    revision: raw.revision as string,
    state: closedSet(raw.state, adoptionStates),
    task: boundedText(raw.task, 128),
    issue_id: raw.issue_id as string,
    acceptance: raw.acceptance as string,
  };
}

const evidenceID = (value: unknown): value is string => typeof value === 'string' && /^[a-f0-9]{32}$/.test(value);

function journeyDeliveredTask(value: unknown): JourneyDeliveredTask {
  const raw = object(value);
  check(decisionID(raw.acceptance) && evidenceID(raw.completion) && evidenceID(raw.publication));
  check(id(raw.issue));
  const generation = counter(raw.generation);
  check(generation >= 1);
  return {
    task: boundedText(raw.task, 128),
    issue: raw.issue as string,
    acceptance: raw.acceptance as string,
    completion: raw.completion as string,
    publication: raw.publication as string,
    generation,
  };
}

function journeyVerifiedApp(value: unknown): JourneyVerifiedApp {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.revision));
  check(commitID(raw.source_commit) && digest(raw.manifest));
  return {
    intent: raw.intent as string,
    revision: raw.revision as string,
    source_commit: raw.source_commit as string,
    manifest: raw.manifest as string,
  };
}

function boundedList<T>(value: unknown, limit: number, parse: (entry: unknown) => T): readonly T[] {
  check(Array.isArray(value) && value.length <= limit);
  return (value as unknown[]).map(parse);
}

// workJourney binds one issue's recorded work journey to its exact
// repository and issue. Section bounds match the API surface: at most
// one delivered-task row and one verified application row per adoption,
// so the adoption bound covers them. Absent newer sections read as
// empty for older reads, never as guessed work.
export function workJourney(value: unknown, repository: string, issue: string): WorkJourney {
  const data = object(value);
  check(data.repository === repository && data.issue === issue && id(repository) && id(issue));
  return {
    repository,
    issue,
    observed_unix: counter(data.observed_unix),
    phases: boundedList(data.phases, 8, journeyPhase),
    tasks: boundedList(data.tasks, 32, journeyTask),
    commits: boundedList(data.commits, 16, journeyCommit),
    findings: boundedList(data.findings, 64, journeyFinding),
    corrections: boundedList(data.corrections, 20, journeyCorrection),
    judgments: boundedList(data.judgments, 8, journeyJudgment),
    gates: boundedList(data.gates, 8, journeyGate),
    adoptions: boundedList(data.adoptions, 16, journeyAdoption),
    delivered_tasks: data.delivered_tasks === undefined ? [] : boundedList(data.delivered_tasks, 16, journeyDeliveredTask),
    verifications: data.verifications === undefined ? [] : boundedList(data.verifications, 16, journeyVerifiedApp),
  };
}
