// Private journey inputs are validated before any authentication or native effect.
import assert from 'node:assert/strict';

export interface JourneyUser { id: string; login: string; password_file: string; public_key_file?: string }
export interface JourneyInput {
  ca_file: string; oauth_client_id: string; origin: string; repository_id: string;
  repository_path: string; revision: string; target: string; users: [JourneyUser, JourneyUser];
}
export function object(value: unknown): Record<string, unknown> {
  assert(value && typeof value === 'object' && !Array.isArray(value), 'Expected an object');
  return value as Record<string, unknown>;
}
export const validID = (value: unknown): value is string => typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) &&
  (value.length < 19 || value <= '9223372036854775807');
export function journeyInput(value: unknown, accessMode: boolean): JourneyInput {
  const input = object(value);
  assert.deepEqual(Object.keys(input).sort(), ['ca_file', 'oauth_client_id', 'origin', 'repository_id', 'repository_path', 'revision', 'target', 'users']);
  const {ca_file, oauth_client_id, origin, repository_id, repository_path, revision, target} = input;
  assert(typeof ca_file === 'string' && typeof origin === 'string');
  assert(typeof target === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,252}$/.test(target));
  assert(typeof revision === 'string' && /^[0-9a-f]{40}$/.test(revision));
  assert(validID(repository_id));
  assert(typeof repository_path === 'string' && /^\/[A-Za-z0-9][A-Za-z0-9_.-]*\/[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(repository_path));
  assert(typeof oauth_client_id === 'string' && oauth_client_id.length > 0 && oauth_client_id.length <= 256);
  assert(Array.isArray(input.users) && input.users.length === 2);
  const users = input.users.map((value: unknown): JourneyUser => {
    const user = object(value);
    assert.deepEqual(Object.keys(user).sort(), accessMode ? ['id', 'login', 'password_file', 'public_key_file'] : ['id', 'login', 'password_file']);
    const {id, login, password_file, public_key_file} = user;
    assert(validID(id) && typeof login === 'string' && /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(login));
    assert(typeof password_file === 'string');
    if (accessMode) {
      assert(/^[a-z][a-z0-9_-]{0,30}$/.test(login) && login !== 'root' && typeof public_key_file === 'string');
      return {id, login, password_file, public_key_file};
    }
    return {id, login, password_file};
  });
  const [first, second] = users;
  assert(first && second && first.id !== second.id && first.login !== second.login);
  return {ca_file, oauth_client_id, origin, repository_id, repository_path, revision, target, users: [first, second]};
}
export interface ManagementRequest {
  cid: string; key_a: string; key_a_public: string; key_b: string; key_b_public: string;
  original_alice: string; original_bob: string; project: string; ssh_config: string; target: string;
}
export function managementInput(value: unknown): ManagementRequest {
  const input = object(value);
  assert.deepEqual(Object.keys(input).sort(), ['cid','key_a','key_a_public','key_b','key_b_public','original_alice','original_bob','project','ssh_config','target']);
  const {cid, key_a, key_a_public, key_b, key_b_public, original_alice, original_bob, project, ssh_config, target} = input;
  assert(typeof cid === 'string' && /^[0-9a-f]{64}$/.test(cid));
  assert(typeof project === 'string' && /^p[0-9a-f]{24}$/.test(project));
  assert(typeof key_a === 'string' && typeof key_a_public === 'string' && typeof key_b === 'string' && typeof key_b_public === 'string');
  assert(typeof original_alice === 'string' && typeof original_bob === 'string' && typeof ssh_config === 'string' && typeof target === 'string');
  return {cid, key_a, key_a_public, key_b, key_b_public, original_alice, original_bob, project, ssh_config, target};
}
