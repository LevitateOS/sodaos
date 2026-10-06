// Soda response contracts used by its two browser components. No copied Forgejo authority.
import {detailResponse} from './sodaspaces-project-response.js';
import type {Detail, Environment} from './sodaspaces-project-response.js';
import {terminalMetadata} from './sodaspaces-terminal-response.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import {spaceFactoryAuthority, spaceFactoryControl, spaceFactoryRuns} from './sodaspaces-factory-response.js';
import type {FactoryAuthority, FactoryControl, FactoryRun} from './sodaspaces-factory-response.js';

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
export interface Space {
  tailnet_state?: string;
  factory_authority?: FactoryAuthority;
  factory_control?: FactoryControl;
  factory_runs: FactoryRun[];
  environment: Environment & {name: string; repository: string; owner_id: string; provisioned: boolean};
  login: string;
  environment_administrator: boolean;
  execution_allowed: boolean;
  authority_unavailable: boolean;
  native_unavailable: boolean;
  observed: Detail['observed'];
  terminals: TerminalMetadata[];
}
function spaceNetwork(row: Record<string, unknown>) {
  const network = row.tailnet_state;
  check(network === undefined || (typeof network === 'string' && ['unavailable', 'off', 'managed'].includes(network)));
  return typeof network === 'string' ? network : undefined;
}
function admitSpaceTerminal(
  value: unknown,
  expectedUserId: string,
  repositoryId: string,
  environmentId: string,
  login: string,
  sessions: Set<string>
) {
  const terminal = terminalMetadata(value, {expectedUserId, repositoryId, environmentId, login});
  check(!sessions.has(terminal.id));
  sessions.add(terminal.id);
  return terminal;
}
function spaceTerminals(
  row: Record<string, unknown>,
  detail: Detail,
  expectedUserId: string,
  repositoryId: string,
  environmentId: string,
  sessions: Set<string>
) {
  check(
    Array.isArray(row.terminals) &&
      row.terminals.length <= 64 &&
      (detail.execution_allowed || row.terminals.length === 0) &&
      (!detail.authority_unavailable || (!detail.environment_administrator && row.terminals.length === 0))
  );
  const terminals = row.terminals.map((value: unknown) =>
    admitSpaceTerminal(value, expectedUserId, repositoryId, environmentId, detail.login, sessions)
  );
  check(sessions.size <= 64);
  return terminals;
}
function spaceItem(value: unknown, expectedUserId: string, seen: Set<string>, sessions: Set<string>): Space {
  const row = object(value),
    env = object(row.environment);
  check(
    projectId(env.id) &&
      id(env.repository_id) &&
      id(env.owner_id) &&
      typeof env.name === 'string' &&
      typeof env.repository === 'string' &&
      !seen.has(env.id)
  );
  seen.add(env.id);
  const environmentId = env.id,
    repositoryId = env.repository_id;
  const detail = detailResponse(row, {id: environmentId, repository_id: repositoryId});
  const terminals = spaceTerminals(row, detail, expectedUserId, repositoryId, environmentId, sessions);
  const network = spaceNetwork(row);
  const authority = spaceFactoryAuthority(row);
  const control = spaceFactoryControl(row);
  const runs = spaceFactoryRuns(row);
  return {
    ...detail,
    ...(typeof network === 'string' ? {tailnet_state: network} : {}),
    ...(authority ? {factory_authority: authority} : {}),
    ...(control ? {factory_control: control} : {}),
    factory_runs: runs,
    environment: {...detail.environment, name: env.name, repository: env.repository, owner_id: env.owner_id},
    terminals,
  };
}
export function spacesResponse(value: unknown): {
  actor: {id: string; login: string};
  items: Space[];
  complete: boolean;
  nextAfter: string;
  factoryIncomplete: boolean;
} {
  const data = object(value);
  const actor = object(data.actor);
  check(
    id(actor.id) &&
      typeof actor.login === 'string' &&
      actor.login.length > 0 &&
      typeof data.complete === 'boolean' &&
      Array.isArray(data.items) &&
      data.items.length <= 32
  );
  check(data.next_after === undefined || projectId(data.next_after));
  check(data.factory_incomplete === undefined || typeof data.factory_incomplete === 'boolean');
  const seen = new Set<string>(),
    sessions = new Set<string>();
  const items = data.items.map((row: unknown) => spaceItem(row, actor.id as string, seen, sessions));
  return {
    actor: {id: actor.id as string, login: actor.login as string},
    items,
    complete: data.complete,
    nextAfter: typeof data.next_after === 'string' ? data.next_after : '',
    factoryIncomplete: data.factory_incomplete === true,
  };
}
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
