// Selected native metadata only. Transport belongs to the existing installed
// reader; this module never authenticates, enrolls, funds or launches a run.
import assert from 'node:assert/strict';
import {object, validID} from '../sodaspaces-input';
import type {
  ApplicationReference,
  ConnectionReference,
  FactoryCase,
  FixtureInventory,
  ProjectReference,
  PublicationReference,
  TerminalReference,
} from './scenario';

export interface FactoryBinding {
  readonly repository_id: string;
  readonly issue_number: number;
  readonly native_issue_id: string | null;
  readonly project_id: string | null;
  readonly connection: ConnectionReference | null;
  // Discovery does not read execution/publication history. Null is unknown,
  // including when the selected repository retains preceding paid attempts.
  readonly run: string | null;
  readonly candidate_head: string | null;
  readonly pr_number: number | null;
  readonly publication: PublicationReference | null;
}

export interface FixtureObservation {
  readonly observed_at: string;
  readonly inventory: FixtureInventory;
  readonly factory: readonly FactoryBinding[];
}

/** Fixed projections from canonical Store/Identity columns, never data blobs. */
export const FIXTURE_IDENTITIES_SQL = `BEGIN READ ONLY;
SELECT json_build_object(
  'projects', (SELECT coalesce(json_agg(row_to_json(p)), '[]') FROM
    (SELECT id, repository_id::text, owner_id::text FROM projects ORDER BY id) p),
  'applications', (SELECT coalesce(json_agg(row_to_json(a)), '[]') FROM
    (SELECT id, data_id, owner_team, origin FROM applications ORDER BY id) a),
  'connections', (SELECT coalesce(json_agg(row_to_json(c)), '[]') FROM
    (SELECT id, owner_id::text, generation, state FROM identity_connections ORDER BY id) c)
); COMMIT;`;

/** Forgejo issue identities only; no issue text, run DTOs or provider output. */
export function issueIdentitiesSQL(cases: readonly FactoryCase[]): string {
  const selections = cases.map(({subject}) => {
    assert(validID(subject.repository_id), 'Invalid selected repository identity');
    positiveInteger(subject.issue_number);
    return `(repo_id=${subject.repository_id} AND index=${subject.issue_number})`;
  });
  return `BEGIN READ ONLY; SELECT coalesce(json_agg(row_to_json(i)), '[]') FROM
    (SELECT id::text, repo_id::text AS repository_id, index AS issue_number
     FROM issue WHERE NOT is_pull AND (${selections.join(' OR ') || 'false'}) ORDER BY id) i;
    COMMIT;`;
}

function text(value: unknown): string {
  assert(typeof value === 'string' && value.length > 0, 'Missing selected metadata string');
  return value;
}

function decimalID(value: unknown): string {
  // Legacy saved observations may contain small JSON numbers. Never recover a
  // rounded int64 by stringifying it; current projections return decimal strings.
  const id = typeof value === 'number' ? String(positiveInteger(value)) : value;
  assert(validID(id), 'Invalid selected numeric identity');
  return id;
}

function positiveInteger(value: unknown): number {
  assert(typeof value === 'number' && Number.isSafeInteger(value) && value > 0, 'Invalid selected integer');
  return value;
}

function rows<T>(value: unknown, project: (row: Record<string, unknown>) => T): T[] {
  assert(Array.isArray(value), 'Missing selected metadata rows');
  return value.map((row) => project(object(row)));
}

function unique<T>(values: readonly T[], identity: (value: T) => string): void {
  assert(new Set(values.map(identity)).size === values.length, 'Ambiguous selected identities');
}

/**
 * Read once through an existing pinned native transport. Its returned projection
 * includes observed_at, the three SQL inventory arrays, selected Forgejo issues,
 * and terminals (null until a genuine native terminal inventory has been read).
 * Neither requested IDs nor cached generations are copied into returned facts.
 */
export async function discoverFixture(
  cases: readonly FactoryCase[],
  read: () => Promise<unknown>
): Promise<FixtureObservation> {
  const selected = object(await read());
  const observed_at = text(selected.observed_at);
  assert(Number.isFinite(Date.parse(observed_at)), 'Missing observation time');
  const projects = rows<ProjectReference>(selected.projects, (p) => ({
    id: text(p.id),
    repository_id: decimalID(p.repository_id),
    owner_id: decimalID(p.owner_id),
  }));
  const applications = rows<ApplicationReference>(selected.applications, (a) => ({
    id: text(a.id),
    data_id: text(a.data_id),
    owner_team: text(a.owner_team),
    origin: text(a.origin),
  }));
  const connections = rows<ConnectionReference>(selected.connections, (c) => ({
    id: text(c.id),
    owner_id: decimalID(c.owner_id),
    generation: positiveInteger(c.generation),
    state: text(c.state),
  }));
  const terminals =
    selected.terminals === null
      ? null
      : rows<TerminalReference>(selected.terminals, (t) => ({
          id: text(t.id),
          project_id: text(t.project_id),
          actor_id: decimalID(t.actor_id),
        }));
  const issues = rows(selected.issues, (i) => ({
    id: decimalID(i.id),
    repository_id: decimalID(i.repository_id),
    issue_number: positiveInteger(i.issue_number),
  }));
  unique(projects, (p) => p.id);
  unique(projects, (p) => p.repository_id);
  unique(applications, (a) => a.id);
  unique(connections, (c) => c.id);
  if (terminals !== null) unique(terminals, (t) => `${t.project_id}/${t.id}`);
  unique(issues, (i) => `${i.repository_id}/${i.issue_number}`);
  const factory = cases.map(({subject}): FactoryBinding => {
    assert(validID(subject.repository_id), 'Invalid selected repository identity');
    positiveInteger(subject.issue_number);
    return {
      repository_id: subject.repository_id,
      issue_number: subject.issue_number,
      native_issue_id:
        issues.find((i) => i.repository_id === subject.repository_id && i.issue_number === subject.issue_number)?.id ??
        null,
      project_id: projects.find((p) => p.repository_id === subject.repository_id)?.id ?? null,
      connection: connections.find((c) => c.id === subject.connection_id) ?? null,
      run: null,
      candidate_head: null,
      pr_number: null,
      publication: null,
    };
  });
  return {observed_at, inventory: {projects, applications, terminals, connections}, factory};
}
