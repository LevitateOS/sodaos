// Soda response contracts used by its two browser components. No copied Forgejo authority.

// Never accept HTML, invalid UTF-8 or an unbounded body as Soda JSON.
async function sodaJSONReader(response: Response) {
  const json = /^application\/json(?:;|$)/i.test(response.headers.get('Content-Type') || '');
  if (!json || Number(response.headers.get('Content-Length')) > 65536) {
    await response.body?.cancel();
    throw Error('Invalid Soda response');
  }
  if (!response.body) throw Error('Missing Soda response body');
  return response.body.getReader();
}

export async function readSodaJSON(response: Response): Promise<unknown> {
  const reader = await sodaJSONReader(response),
    decoder = new TextDecoder('utf-8', {fatal: true});
  let size = 0,
    text = '';
  try {
    for (;;) {
      const {done, value} = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > 65536) throw Error('Oversized Soda response');
      text += decoder.decode(value, {stream: true});
    }
    return JSON.parse(text + decoder.decode());
  } catch (error) {
    await reader.cancel();
    throw error;
  } finally {
    reader.releaseLock();
  }
}
export function object(value: unknown): Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid Soda object');
  return value as Record<string, unknown>; // The object shape is checked; properties remain unknown.
}
export function check(condition: unknown): asserts condition {
  if (!condition) throw Error('Invalid or mismatched Soda response');
}
export const id = (value: unknown): value is string =>
  typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) && BigInt(value) <= 9223372036854775807n;
export const projectId = (value: unknown): value is string =>
  typeof value === 'string' && /^p[0-9a-f]{24}$/.test(value);
export const fingerprint = (value: unknown): value is string =>
  typeof value === 'string' && /^SHA256:[A-Za-z0-9+/]{43}$/.test(value);
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
export class SodaRequestError extends Error {
  constructor(
    readonly status: number,
    readonly code?: string
  ) {
    super('Soda request failed');
  }
}
