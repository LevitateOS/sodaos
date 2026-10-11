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
}
export interface IntentTask {
  key: string;
  objective: string;
  outcomes: string[];
  idea_ids: string[];
  depends_on?: string[];
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
  state: 'draft' | 'accepted' | 'declined';
  max_issues: number;
  planning_calls: number;
  approach?: 'complete_first' | 'incremental';
}
export const decisionID = (value: unknown): value is string =>
  typeof value === 'string' && /^d[0-9a-f]{24}$/.test(value);
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
  };
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
export function intentRevision(value: unknown): IntentRevision {
  const raw = object(value);
  check(raw.approach === undefined || raw.approach === 'complete_first' || raw.approach === 'incremental');
  check(decisionID(raw.id) && decisionID(raw.intent) && (raw.repository === '0' || id(raw.repository)));
  check(raw.state === 'draft' || raw.state === 'accepted' || raw.state === 'declined');
  const ideas = array(raw.ideas, 50).map(idea);
  const maxIssues = bound(raw.max_issues, 1, 64);
  const tasks = array(raw.tasks, maxIssues).map(task);
  return {
    id: raw.id,
    intent: raw.intent,
    repository: raw.repository,
    objective: text(raw.objective, 16000),
    state: raw.state,
    max_issues: maxIssues,
    planning_calls: bound(raw.planning_calls, 0, 100),
    ...(raw.approach !== undefined ? {approach: raw.approach} : {}),
    ideas,
    tasks,
    ...references(raw),
  };
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
