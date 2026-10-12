import {check, object, id} from './sodaspaces-api.js';

export interface Department {
  id: string;
  name: string;
  description: string;
}
export interface ProductIdea {
  id: string;
  description: string;
  consequence: string;
  assumption: boolean;
  decision: 'pending' | 'accept' | 'decline' | 'edit';
  correction?: string;
  source?: 'requested' | 'supported' | 'autonomous' | 'assumption';
  provenance?: 'human' | 'autonomous';
  actor?: string;
  evidence?: string;
  material?: boolean;
}
export interface IntentTask {
  key: string;
  objective: string;
  outcomes: string[];
  idea_ids: string[];
  depends_on?: string[];
}
export interface ProposalBranch {
  outcome: string;
  result: string;
}
export interface ProposalStep {
  actor: string;
  action: string;
  result: string;
  branches?: ProposalBranch[];
}
export interface ProposalAlternative {
  key: string;
  name: string;
  audience?: string;
  summary?: string;
  steps: ProposalStep[];
  visibility?: string;
  consequences?: string;
  evidence?: string;
  outcomes: string[];
}
export interface Exclusion {
  idea: string;
  description: string;
  consequence: string;
  provenance?: 'human' | 'autonomous';
}
export interface ProposalContent {
  producer: string;
  supported: boolean;
  intent?: string;
  revision?: string;
  objective?: string;
  facts?: string[];
  alternatives?: ProposalAlternative[];
  excluded?: string[];
  exclusions?: Exclusion[];
  unknowns: string[];
}
export interface IntentRevision {
  id: string;
  intent: string;
  predecessor?: string;
  repository: string;
  department?: string;
  objective: string;
  ideas: ProductIdea[];
  tasks: IntentTask[];
  state: 'draft' | 'accepted' | 'declined' | 'rejected' | 'abandoned';
  max_issues: number;
  planning_calls: number;
  approach?: 'complete_first' | 'incremental';
  effective_outcomes?: string[];
}
export interface AssumptionChallenge {
  idea: string;
  assumption: string;
  consequence: string;
  recommendation: string;
}
export interface Milestone {
  name: string;
  result: string;
}
export interface ConsolidatedProposal {
  producer: string;
  intent: string;
  revision: string;
  objective: string;
  recommended: string;
  alternative: ProposalAlternative;
  outcomes: string[];
  constraints: string[];
  excluded?: string[];
  milestones: Milestone[];
  advice: string;
}
export interface ConsistencyMismatch {
  class: 'effect_added' | 'outcome_gap' | 'summary_drift' | 'unverified';
  detail: string;
}
export interface ConsistencyReport {
  producer: string;
  intent: string;
  revision: string;
  mismatches: ConsistencyMismatch[];
  confidence: number;
  advice: string;
}
export interface Exploration {
  content: ProposalContent;
  challenges: AssumptionChallenge[];
  settled: string[];
  recommendation?: ConsolidatedProposal;
  consistency?: ConsistencyReport;
}
export interface PlanningAllowance {
  id: string;
  intent: string;
  amount: number;
  reserved: number;
  consumed: number;
}
export function planningAllowance(value: unknown, intent: string): PlanningAllowance {
  const raw = object(value);
  check(decisionID(raw.id) && raw.intent === intent);
  check(raw.unit === 'planning_calls');
  return {
    id: String(raw.id),
    intent: String(raw.intent),
    amount: bound(raw.amount, 0, 100),
    reserved: bound(raw.reserved, 0, 100),
    consumed: bound(raw.consumed, 0, 100),
  };
}
export interface SavedRequest {
  intent: string;
  revision: string;
  state: IntentRevision['state'];
  objective: string;
  repository: string;
  department?: string;
  created_unix: number;
  effective_count: number;
  task_count: number;
  effective_outcomes?: string[];
}
export const decisionID = (value: unknown): value is string =>
  typeof value === 'string' && /^d[0-9a-f]{24}$/.test(value);
const digest = (value: unknown): value is string => typeof value === 'string' && /^[0-9a-f]{64}$/.test(value);
const authoredProducer = 'soda-authored-v1';
function text(value: unknown, limit: number): string {
  check(typeof value === 'string' && value.trim() && value.length <= limit && !value.includes('\0'));
  return value;
}
export function department(value: unknown): Department {
  const raw = object(value);
  check(decisionID(raw.id));
  return {id: raw.id, name: text(raw.name, 120), description: text(raw.description, 2000)};
}
export function departments(value: unknown): Department[] {
  check(Array.isArray(value) && value.length <= 200);
  return value.map(department);
}
function idea(value: unknown): ProductIdea {
  const raw = object(value);
  check(decisionID(raw.id) && typeof raw.assumption === 'boolean');
  check(
    raw.decision === 'pending' || raw.decision === 'accept' || raw.decision === 'decline' || raw.decision === 'edit'
  );
  return {
    id: raw.id,
    description: text(raw.description, 4000),
    consequence: text(raw.consequence, 4000),
    assumption: raw.assumption,
    decision: raw.decision,
    ...(raw.correction ? {correction: text(raw.correction, 4000)} : {}),
    ...provenance(raw),
  };
}
const sourceKinds = ['requested', 'supported', 'autonomous', 'assumption'];
function provenanceKind(raw: Record<string, unknown>, out: Partial<ProductIdea>) {
  check(raw.source === undefined || (typeof raw.source === 'string' && sourceKinds.includes(raw.source)));
  check(raw.provenance === undefined || raw.provenance === 'human' || raw.provenance === 'autonomous');
  if (raw.source) out.source = raw.source as NonNullable<ProductIdea['source']>;
  if (raw.provenance) out.provenance = raw.provenance as NonNullable<ProductIdea['provenance']>;
}
function provenanceDetail(raw: Record<string, unknown>, out: Partial<ProductIdea>) {
  if (raw.actor !== undefined) check(id(raw.actor));
  if (raw.material !== undefined) check(typeof raw.material === 'boolean');
  if (raw.actor) out.actor = String(raw.actor);
  if (raw.evidence) out.evidence = text(raw.evidence, 4000);
  if (raw.material !== undefined) out.material = Boolean(raw.material);
}
function provenance(raw: Record<string, unknown>): Partial<ProductIdea> {
  const out: Partial<ProductIdea> = {};
  provenanceKind(raw, out);
  provenanceDetail(raw, out);
  return out;
}
function effectiveOutcomes(raw: Record<string, unknown>): Partial<IntentRevision> {
  if (raw.effective_outcomes === undefined) return {};
  const outcomes = array(raw.effective_outcomes, 50);
  for (const outcome of outcomes) check(decisionID(outcome));
  return {effective_outcomes: outcomes.map(String)};
}
function strings(value: unknown, limit: number): string[] {
  check(Array.isArray(value) && value.length <= 64);
  return value.map((item) => text(item, limit));
}
function task(value: unknown): IntentTask {
  const raw = object(value);
  check(decisionID(raw.key));
  return {
    key: raw.key,
    objective: text(raw.objective, 4000),
    outcomes: strings(raw.outcomes, 4000),
    idea_ids: strings(raw.idea_ids, 25),
    ...(raw.depends_on ? {depends_on: strings(raw.depends_on, 25)} : {}),
  };
}
export function intentState(value: unknown): IntentRevision['state'] {
  check(
    value === 'draft' || value === 'accepted' || value === 'declined' || value === 'rejected' || value === 'abandoned'
  );
  return value as IntentRevision['state'];
}
export function intentRevision(value: unknown): IntentRevision {
  const raw = object(value);
  check(raw.approach === undefined || raw.approach === 'complete_first' || raw.approach === 'incremental');
  check(decisionID(raw.id) && decisionID(raw.intent) && (raw.repository === '0' || id(raw.repository)));
  const state = intentState(raw.state);
  const ideas = array(raw.ideas, 50).map(idea);
  const maxIssues = bound(raw.max_issues, 1, 64);
  const tasks = array(raw.tasks, maxIssues).map(task);
  return {
    id: raw.id,
    intent: raw.intent,
    repository: raw.repository,
    objective: text(raw.objective, 16000),
    state,
    max_issues: maxIssues,
    planning_calls: bound(raw.planning_calls, 0, 100),
    ...(raw.approach !== undefined ? {approach: raw.approach} : {}),
    ideas,
    tasks,
    ...references(raw),
    ...effectiveOutcomes(raw),
  };
}
function count(value: unknown): number {
  check(Number.isInteger(value) && Number(value) >= 0);
  return Number(value);
}
function savedRequest(value: unknown): SavedRequest {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.revision) && (raw.repository === '0' || id(raw.repository)));
  if (raw.department !== undefined) check(decisionID(raw.department));
  return {
    intent: raw.intent,
    revision: raw.revision,
    state: intentState(raw.state),
    objective: text(raw.objective, 280),
    repository: raw.repository,
    ...(raw.department ? {department: String(raw.department)} : {}),
    created_unix: count(raw.created_unix),
    effective_count: count(raw.effective_count),
    task_count: count(raw.task_count),
    ...effectiveOutcomes(raw),
  };
}
export interface IntentGenerationStatus {
  state: string;
  revision: string;
  tasks: number;
  authorization?: string;
  native?: number;
}
function authorizationReference(value: unknown): string {
  const raw = text(value, 90);
  const separator = raw.indexOf(':');
  check(separator > 0 && decisionID(raw.slice(0, separator)) && /^[0-9a-f]{64}$/.test(raw.slice(separator + 1)));
  return raw.slice(0, separator);
}
export function intentGeneration(value: unknown): IntentGenerationStatus {
  const raw = object(value);
  check(decisionID(raw.revision_id));
  let tasks = 0;
  if (raw.task_issue_ids !== undefined) {
    const bindings = object(raw.task_issue_ids);
    const keys = Object.keys(bindings);
    check(keys.length <= 100);
    tasks = keys.length;
  }
  const out: IntentGenerationStatus = {
    state: text(raw.state, 64),
    revision: raw.revision_id,
    tasks,
  };
  if (raw.authorization_revision !== undefined) out.authorization = authorizationReference(raw.authorization_revision);
  if (raw.native_revision !== undefined) {
    check(Number.isInteger(raw.native_revision) && Number(raw.native_revision) >= 0);
    out.native = Number(raw.native_revision);
  }
  return out;
}
export function intentList(value: unknown): {intents: SavedRequest[]; next_cursor?: string} {
  const raw = object(value);
  const intents = array(raw.intents, 50).map(savedRequest);
  if (raw.next_cursor !== undefined) check(decisionID(raw.next_cursor));
  return {intents, ...(raw.next_cursor ? {next_cursor: String(raw.next_cursor)} : {})};
}
function branch(value: unknown): ProposalBranch {
  const raw = object(value);
  return {outcome: text(raw.outcome, 4000), result: text(raw.result, 4000)};
}
function step(value: unknown): ProposalStep {
  const raw = object(value);
  const branches = raw.branches === undefined ? [] : array(raw.branches, 4).map(branch);
  return {
    actor: text(raw.actor, 4000),
    action: text(raw.action, 4000),
    result: text(raw.result, 4000),
    ...(branches.length ? {branches} : {}),
  };
}
function alternative(value: unknown): ProposalAlternative {
  const raw = object(value);
  const steps = array(raw.steps, 8).map(step);
  check(steps.length >= 1);
  const out: ProposalAlternative = {
    key: text(raw.key, 64),
    name: text(raw.name, 4000),
    steps,
    outcomes: array(raw.outcomes, 20).map((outcome) => text(outcome, 4000)),
  };
  if (raw.audience) out.audience = text(raw.audience, 4000);
  if (raw.summary) out.summary = text(raw.summary, 4000);
  if (raw.visibility) out.visibility = text(raw.visibility, 4000);
  if (raw.consequences) out.consequences = text(raw.consequences, 4000);
  if (raw.evidence) out.evidence = text(raw.evidence, 4000);
  return out;
}
function proposalAlternatives(raw: Record<string, unknown>, supported: boolean): ProposalAlternative[] | undefined {
  if (!supported) {
    check(raw.alternatives === undefined);
    return undefined;
  }
  const alternatives = array(raw.alternatives, 2).map(alternative);
  check(alternatives.length === 2);
  return alternatives;
}
function proposalBinding(raw: Record<string, unknown>, out: ProposalContent) {
  if (raw.intent !== undefined) check(decisionID(raw.intent));
  if (raw.revision !== undefined) check(decisionID(raw.revision));
  if (raw.intent) out.intent = String(raw.intent);
  if (raw.revision) out.revision = String(raw.revision);
  if (raw.objective) out.objective = text(raw.objective, 16000);
  if (raw.facts !== undefined) {
    const facts = array(raw.facts, 51).map((fact) => text(fact, 16000));
    check(facts.length >= 1);
    out.facts = facts;
  }
}
export function proposalContent(value: unknown): ProposalContent {
  const raw = object(value);
  check(typeof raw.supported === 'boolean');
  const unknowns = array(raw.unknowns, 8).map((unknown) => text(unknown, 4000));
  check(unknowns.length >= 1);
  const out: ProposalContent = {
    producer: text(raw.producer, 200),
    supported: raw.supported,
    unknowns,
  };
  proposalBinding(raw, out);
  const alternatives = proposalAlternatives(raw, raw.supported);
  if (alternatives) out.alternatives = alternatives;
  const excluded = raw.excluded !== undefined ? excludedIDs(raw) : undefined;
  if (excluded) out.excluded = excluded;
  const meanings = proposalExclusions(raw, excluded);
  if (meanings) out.exclusions = meanings;
  return out;
}
function exclusion(value: unknown): Exclusion {
  const raw = object(value);
  check(decisionID(raw.idea));
  const out: Exclusion = {
    idea: raw.idea,
    description: text(raw.description, 4000),
    consequence: text(raw.consequence, 4000),
  };
  if (raw.provenance !== undefined) {
    check(raw.provenance === 'human' || raw.provenance === 'autonomous');
    out.provenance = raw.provenance as 'human' | 'autonomous';
  }
  return out;
}
function proposalExclusions(raw: Record<string, unknown>, excluded: string[] | undefined): Exclusion[] | undefined {
  if (raw.exclusions === undefined) {
    check(excluded === undefined || excluded.length === 0);
    return undefined;
  }
  const meanings = array(raw.exclusions, 50).map(exclusion);
  check(excluded !== undefined && meanings.length === excluded.length);
  meanings.forEach((meaning, index) => check(meaning.idea === excluded[index]));
  return meanings;
}
function challenge(value: unknown): AssumptionChallenge {
  const raw = object(value);
  check(decisionID(raw.idea));
  return {
    idea: raw.idea,
    assumption: text(raw.assumption, 4000),
    consequence: text(raw.consequence, 4000),
    recommendation: text(raw.recommendation, 4000),
  };
}
function milestone(value: unknown): Milestone {
  const raw = object(value);
  return {name: text(raw.name, 4000), result: text(raw.result, 4000)};
}
function recommendation(value: unknown): ConsolidatedProposal {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.revision));
  const other = alternative(raw.alternative);
  check(raw.recommended === other.key);
  const outcomes = array(raw.outcomes, 20).map((outcome) => text(outcome, 4000));
  check(outcomes.length >= 1);
  const milestones = array(raw.milestones, 40).map(milestone);
  check(milestones.length >= 1);
  return {
    producer: text(raw.producer, 200),
    intent: raw.intent,
    revision: raw.revision,
    objective: text(raw.objective, 16000),
    recommended: other.key,
    alternative: other,
    outcomes,
    constraints: array(raw.constraints, 50).map((constraint) => text(constraint, 4000)),
    ...(raw.excluded !== undefined ? {excluded: excludedIDs(raw)} : {}),
    milestones,
    advice: text(raw.advice, 200),
  };
}
function excludedIDs(raw: Record<string, unknown>): string[] {
  const excluded = array(raw.excluded, 50);
  for (const id of excluded) check(decisionID(id));
  return excluded.map(String);
}
function mismatch(value: unknown): ConsistencyMismatch {
  const raw = object(value);
  check(
    raw.class === 'effect_added' ||
      raw.class === 'outcome_gap' ||
      raw.class === 'summary_drift' ||
      raw.class === 'unverified'
  );
  return {class: raw.class as ConsistencyMismatch['class'], detail: text(raw.detail, 4000)};
}
function consistency(value: unknown): ConsistencyReport {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.revision));
  check(typeof raw.confidence === 'number' && Number.isFinite(raw.confidence));
  check(Number(raw.confidence) >= 0 && Number(raw.confidence) <= 1);
  return {
    producer: text(raw.producer, 200),
    intent: raw.intent,
    revision: raw.revision,
    mismatches: array(raw.mismatches, 16).map(mismatch),
    confidence: Number(raw.confidence),
    advice: text(raw.advice, 200),
  };
}
export function exploration(value: unknown): Exploration {
  const raw = object(value);
  const settled = array(raw.settled ?? [], 50);
  for (const id of settled) check(decisionID(id));
  const out: Exploration = {
    content: proposalContent(raw.content),
    challenges: array(raw.challenges, 50).map(challenge),
    settled: settled.map(String),
  };
  if (raw.recommendation !== undefined && raw.recommendation !== null)
    out.recommendation = recommendation(raw.recommendation);
  if (raw.consistency !== undefined && raw.consistency !== null) out.consistency = consistency(raw.consistency);
  return out;
}
export interface SavedReview extends Exploration {
  intent: string;
  basis_revision: string;
  result_revision: string;
  saved_unix: number;
}
export interface SavedReviewRead {
  intent: string;
  head_revision: string;
  status: 'current' | 'stale' | 'missing';
  review?: SavedReview;
}
function reviewBindings(raw: Record<string, unknown>, payload: Exploration) {
  check(payload.content.intent === raw.intent && payload.content.revision === raw.basis_revision);
  if (payload.recommendation) {
    check(payload.recommendation.producer === authoredProducer);
    check(payload.recommendation.intent === raw.intent && payload.recommendation.revision === raw.basis_revision);
    check(digest(payload.recommendation.advice));
  }
  if (payload.consistency) {
    check(payload.consistency.producer === authoredProducer);
    check(payload.consistency.intent === raw.intent && payload.consistency.revision === raw.basis_revision);
    check(digest(payload.consistency.advice));
  }
}
function reviewContentShape(payload: Exploration) {
  check(payload.content.producer === authoredProducer);
  const objective = payload.content.objective;
  const facts = payload.content.facts;
  check(objective !== undefined && facts !== undefined && facts.length > 0 && facts[0] === objective);
}
function reviewVerdicts(payload: Exploration) {
  const verdicts = new Set<string>();
  for (const entry of payload.challenges) {
    check(!verdicts.has(entry.idea));
    verdicts.add(entry.idea);
  }
  for (const id of payload.settled) {
    check(!verdicts.has(id));
    verdicts.add(id);
  }
}
export function savedReview(value: unknown): SavedReview {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.basis_revision) && decisionID(raw.result_revision));
  check(raw.basis_revision !== raw.result_revision);
  const payload = exploration(value);
  reviewBindings(raw, payload);
  reviewContentShape(payload);
  reviewVerdicts(payload);
  const saved = count(raw.saved_unix);
  check(saved > 0);
  return {
    intent: raw.intent,
    basis_revision: raw.basis_revision,
    result_revision: raw.result_revision,
    saved_unix: saved,
    ...payload,
  };
}
export function savedReviewRead(value: unknown): SavedReviewRead {
  const raw = object(value);
  check(decisionID(raw.intent) && decisionID(raw.head_revision));
  check(raw.status === 'current' || raw.status === 'stale' || raw.status === 'missing');
  if (raw.status === 'missing') {
    check(raw.review === undefined || raw.review === null);
    return {intent: raw.intent, head_revision: raw.head_revision, status: 'missing'};
  }
  const review = savedReview(raw.review);
  check(review.intent === raw.intent);
  const status = raw.status as SavedReviewRead['status'];
  check(status === (raw.head_revision === review.result_revision ? 'current' : 'stale'));
  return {intent: raw.intent, head_revision: raw.head_revision, status, review};
}
function references(raw: Record<string, unknown>) {
  if (raw.predecessor !== undefined) check(decisionID(raw.predecessor));
  if (raw.department !== undefined) check(decisionID(raw.department));
  return {
    ...(raw.predecessor ? {predecessor: String(raw.predecessor)} : {}),
    ...(raw.department ? {department: String(raw.department)} : {}),
  };
}
function bound(value: unknown, minimum: number, maximum: number) {
  check(Number.isInteger(value) && Number(value) >= minimum && Number(value) <= maximum);
  return Number(value);
}
function array(value: unknown, maximum: number): unknown[] {
  check(Array.isArray(value) && value.length <= maximum);
  return value;
}
export function newDecisionID(): string {
  return (
    'd' + Array.from(crypto.getRandomValues(new Uint8Array(12)), (byte) => byte.toString(16).padStart(2, '0')).join('')
  );
}
