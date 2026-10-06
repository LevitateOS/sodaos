import {check, id, object, projectId} from './sodaspaces-api.js';
import {detailResponse} from './sodaspaces-project-response.js';
import type {Detail, Environment} from './sodaspaces-project-response.js';
import {terminalMetadata} from './sodaspaces-terminal-response.js';
import type {TerminalMetadata} from './sodaspaces-terminal-response.js';
import {spaceFactoryAuthority, spaceFactoryControl, spaceFactoryRuns} from './sodaspaces-factory-response.js';
import type {FactoryAuthority, FactoryControl, FactoryRun} from './sodaspaces-factory-response.js';

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
