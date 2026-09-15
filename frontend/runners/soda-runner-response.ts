import type {Action, Response, Runner} from './soda-runner-types.js';

export function decodeRunnerResponse<A extends Action>(action: A, value: unknown): Response<A>;
export function decodeRunnerResponse(action: Action, value: unknown): unknown {
  assertObject(value);
  if (action !== 'list') return admitMutationResponse(value);
  admitListCounts(value);
  const rows = admitRunners(value);
  const unavailable = admitUnavailable(value, rows);
  admitUniqueLocators(rows, unavailable);
  admitInventoryCounts(value, rows, unavailable);
  admitForgejoOrigin(value);
  return value;
}

function assertObject(value: unknown): asserts value is Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid runner response');
}

function admitMutationResponse(value: Record<string, unknown>): Record<string, unknown> {
  if (value.ok !== true) throw Error('Runner mutation was not confirmed');
  return value;
}

function admitNonNegativeCount(value: Record<string, unknown>, field: string): void {
  const count = value[field];
  if (typeof count !== 'number' || !Number.isSafeInteger(count) || count < 0)
    throw Error(`runner list has invalid ${field}`);
}

function admitListCounts(value: Record<string, unknown>): void {
  for (const field of ['runner_count', 'active_listeners', 'total_capacity']) admitNonNegativeCount(value, field);
}

function admitRunners(value: Record<string, unknown>): Runner[] {
  if (!Array.isArray(value.runners) || value.runner_count !== value.runners.length)
    throw Error('runner list has inconsistent local capacity data');
  value.runners.forEach(assertRunner);
  return value.runners as Runner[];
}

function isUnavailableLocator(id: unknown): id is string {
  return typeof id === 'string' && /^[a-z][a-z0-9-]{0,15}$/.test(id);
}

function admitUnavailable(value: Record<string, unknown>, rows: Runner[]): string[] {
  if (!Array.isArray(value.unavailable) || rows.length + value.unavailable.length > 64)
    throw Error('Invalid unavailable runner locators');
  for (const id of value.unavailable) {
    if (!isUnavailableLocator(id)) throw Error('Invalid unavailable runner locators');
  }
  return value.unavailable;
}

function admitUniqueLocators(rows: Runner[], unavailable: string[]): void {
  const ids = [...rows.map((row) => row.id), ...unavailable];
  if (new Set(ids).size !== ids.length) throw Error('Duplicate runner locators');
}

function runnerObservationComplete(row: Runner): boolean {
  return row.version !== '' && row.service !== null;
}

function runnerListening(row: Runner): boolean {
  return row.service?.active === 'active' && row.service.sub === 'running';
}

function expectedComplete(rows: Runner[], unavailable: string[]): boolean {
  return unavailable.length === 0 && rows.every(runnerObservationComplete);
}

function admitInventoryCounts(value: Record<string, unknown>, rows: Runner[], unavailable: string[]): void {
  const complete = expectedComplete(rows, unavailable);
  if (
    value.complete !== complete ||
    value.total_capacity !== rows.length ||
    value.active_listeners !== rows.filter(runnerListening).length
  )
    throw Error('Invalid partial inventory counts');
}

function admitForgejoOrigin(value: Record<string, unknown>): void {
  if (value.forgejo_url !== undefined && typeof value.forgejo_url !== 'string') throw Error('Invalid Forgejo origin');
}

function assertStringFields(value: Record<string, unknown>, fields: string[], message: string): void {
  for (const field of fields) {
    if (typeof value[field] !== 'string') throw Error(message);
  }
}

function assertRunnerBinding(value: Record<string, unknown>): void {
  if (
    typeof value.id !== 'string' ||
    !/^[a-z][a-z0-9-]{0,15}$/.test(value.id) ||
    value.account !== 'soda-runner-' + value.id
  )
    throw Error('Invalid runner binding');
}

function assertRunnerService(service: unknown): void {
  assertObject(service);
  for (const field of ['load', 'active', 'sub', 'enabled']) {
    if (typeof service[field] !== 'string' || !service[field].trim()) throw Error('Invalid runner service observation');
  }
}

function assertRunner(value: unknown): asserts value is Runner {
  assertObject(value);
  assertStringFields(
    value,
    ['id', 'provider', 'registration_url', 'account', 'architecture', 'version'],
    'Invalid runner observation'
  );
  assertRunnerBinding(value);
  if (value.provider !== 'forgejo') throw Error('Unsupported runner provider');
  if (value.capacity !== 1) throw Error('Runner does not report its one native slot');
  if (value.service === null) return;
  assertRunnerService(value.service);
}
