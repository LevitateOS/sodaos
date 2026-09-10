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
  if (value.capacity !== 1) throw Error('Runner does not report its one native slot');
  assertObject(value.service);
  for (const field of ['load', 'active', 'sub', 'enabled']) {
    if (typeof value.service[field] !== 'string') throw Error('Invalid runner service observation');
  }
}
