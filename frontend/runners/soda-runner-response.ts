import type {Action, Response, Runner} from './soda-runner-types.js';

export function decodeRunnerResponse<A extends Action>(action: A, value: unknown): Response<A>;
export function decodeRunnerResponse(action: Action, value: unknown): unknown {
  assertObject(value);
  if (action !== 'list') {
    if (value.ok !== true) throw Error('Runner mutation was not confirmed');
    return value;
  }
  for (const field of ['runner_count', 'active_listeners', 'total_capacity']) {
    const count = value[field];
    if (typeof count !== 'number' || !Number.isSafeInteger(count) || count < 0) throw Error(`runner list has invalid ${field}`);
  }
  if (!Array.isArray(value.runners) || value.runner_count !== value.runners.length) throw Error('runner list has inconsistent local capacity data');
  value.runners.forEach(assertRunner);
  const rows = value.runners as Runner[];
  if (!Array.isArray(value.unavailable) || rows.length + value.unavailable.length > 64 || value.unavailable.some(id => typeof id !== 'string' || !/^[a-z][a-z0-9-]{0,15}$/.test(id))) throw Error('Invalid unavailable runner locators');
  const ids = [...rows.map(row => row.id), ...value.unavailable];
  if (new Set(ids).size !== ids.length) throw Error('Duplicate runner locators');
  const complete = value.unavailable.length === 0 && rows.every(row => row.version !== '' && row.service !== null);
  if (value.complete !== complete || value.total_capacity !== rows.length || value.active_listeners !== rows.filter(row => row.service?.active === 'active' && row.service.sub === 'running').length) throw Error('Invalid partial inventory counts');
  if (value.forgejo_url !== undefined && typeof value.forgejo_url !== 'string') throw Error('Invalid Forgejo origin');
  return value;
}
function assertObject(value: unknown): asserts value is Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid runner response');
}
function assertRunner(value: unknown): asserts value is Runner {
  assertObject(value);
  for (const field of ['id', 'provider', 'registration_url', 'account', 'architecture', 'version']) {
    if (typeof value[field] !== 'string') throw Error('Invalid runner observation');
  }
  if (typeof value.id !== 'string' || !/^[a-z][a-z0-9-]{0,15}$/.test(value.id) || value.account !== 'soda-runner-' + value.id) throw Error('Invalid runner binding');
  if (value.provider !== 'forgejo') throw Error('Unsupported runner provider');
  if (value.capacity !== 1) throw Error('Runner does not report its one native slot');
  if (value.service === null) return;
  assertObject(value.service);
  for (const field of ['load', 'active', 'sub', 'enabled']) {
    if (typeof value.service[field] !== 'string' || !value.service[field].trim()) throw Error('Invalid runner service observation');
  }
}
