import {check, id, object, projectId} from './sodaspaces-api.js';

export interface RepositoryChoice {
  id: string;
  owner: string;
  name: string;
  canCreate: boolean;
  project: {id: string; provisioned: boolean} | null;
}
export interface RepositoryChoices {
  items: RepositoryChoice[];
  page: number;
  nextCursor: string;
}
function repositoryPathPart(v: unknown): v is string {
  return (
    typeof v === 'string' &&
    v !== '' &&
    v !== '.' &&
    v !== '..' &&
    new TextEncoder().encode(v).length <= 255 &&
    !/[\/\\\\\p{Cc}\p{Cf}]/u.test(v)
  );
}
function repositoryProject(row: Record<string, unknown>): RepositoryChoice['project'] {
  if (row.project !== null) {
    const p = object(row.project);
    check(projectId(p.id) && typeof p.provisioned === 'boolean' && !row.can_create);
    return {id: p.id, provisioned: p.provisioned};
  }
  check(row.can_create);
  return null;
}
function repositoryChoice(raw: unknown, seen: Set<string>): RepositoryChoice {
  const row = object(raw);
  check(id(row.id) && !seen.has(row.id) && typeof row.can_create === 'boolean');
  seen.add(row.id);
  check(repositoryPathPart(row.owner) && repositoryPathPart(row.name));
  return {id: row.id, owner: row.owner, name: row.name, canCreate: row.can_create, project: repositoryProject(row)};
}
export function repositoryChoices(value: unknown, page: number): RepositoryChoices {
  const data = object(value);
  check(Number.isSafeInteger(page) && page >= 1);
  check(data.page === undefined && data.more === undefined && data.limited === undefined);
  check(data.next_cursor === undefined || (typeof data.next_cursor === 'string' && data.next_cursor.length <= 4096));
  check(Array.isArray(data.items) && data.items.length <= 12);
  const seen = new Set<string>();
  const items = data.items.map((raw: unknown) => repositoryChoice(raw, seen));
  return {items, page, nextCursor: typeof data.next_cursor === 'string' ? data.next_cursor : ''};
}
