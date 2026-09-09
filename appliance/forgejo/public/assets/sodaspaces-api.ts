// Soda response contracts used by its two browser components. No copied Forgejo authority.
export function object(value: unknown): Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw Error('Invalid Soda object');
  return value as Record<string, unknown>; // The object shape is checked; properties remain unknown.
}
export function check(condition: unknown): asserts condition {
  if (!condition) throw Error('Invalid or mismatched Soda response');
}
export const id = (value: unknown): value is string => typeof value === 'string' && /^[1-9][0-9]{0,18}$/.test(value) && BigInt(value) <= 9223372036854775807n;
export const projectId = (value: unknown): value is string => typeof value === 'string' && /^p[0-9a-f]{24}$/.test(value);
export const fingerprint = (value: unknown): value is string => typeof value === 'string' && /^SHA256:[A-Za-z0-9+/]{43}$/.test(value);
export interface Session { user: { id: string; login: string }; csrf_token: string; forgejo_url: string }
export function sessionResponse(value: unknown, origin: string): Session {
  const data = object(value), user = object(data.user);
  check(id(user.id) && typeof user.login === 'string' && typeof data.csrf_token === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(data.csrf_token) && data.forgejo_url === origin);
  return { user: { id: user.id, login: user.login }, csrf_token: data.csrf_token, forgejo_url: origin };
}
export interface Environment { id: string; repository_id: string }
export function environmentResponse(value: unknown, repositoryId: string): Environment {
  const data = object(value); check(projectId(data.id) && data.repository_id === repositoryId);
  return { id: data.id, repository_id: repositoryId };
}
export interface Detail {
  environment: Environment & { provisioned: boolean };
  login: string; environment_administrator: boolean; native_unavailable: boolean; authority_unavailable: boolean;
  observed: { id: string; running: boolean } | null;
}
export function detailResponse(value: unknown, environment: Environment): Detail {
  const data = object(value), env = object(data.environment);
  check(env.id === environment.id && env.repository_id === environment.repository_id && typeof env.provisioned === 'boolean' && typeof data.login === 'string' && typeof data.environment_administrator === 'boolean' && typeof data.native_unavailable === 'boolean' && typeof data.authority_unavailable === 'boolean');
  let observed: Detail['observed'] = null;
  if (data.observed !== null) {
    const state = object(data.observed); check(state.id === environment.id && typeof state.running === 'boolean');
    observed = { id: environment.id, running: state.running };
  }
  return { environment: { ...environment, provisioned: env.provisioned }, login: data.login, environment_administrator: data.environment_administrator, native_unavailable: data.native_unavailable, authority_unavailable: data.authority_unavailable, observed };
}
export interface SavedKey { id: string; fingerprint: string; public_key?: string }
export function savedKeysResponse(value: unknown): SavedKey[] {
  const data = object(value); check(Array.isArray(data.items));
  return data.items.map((value: unknown) => {
    const key = object(value); check(id(key.id) && fingerprint(key.fingerprint));
    return { id: key.id, fingerprint: key.fingerprint, ...(typeof key.public_key === 'string' ? { public_key: key.public_key } : {}) };
  });
}
export interface KeyPreview { login: string; revision: string; installed_fingerprints: string[]; saved_fingerprints: string[] }
export function keyPreviewResponse(value: unknown, login: string): KeyPreview {
  const data = object(value);
  check(data.login === login && typeof data.revision === 'string' && /^[0-9a-f]{64}$/.test(data.revision) && Array.isArray(data.installed_fingerprints) && Array.isArray(data.saved_fingerprints));
  const installed: unknown[] = data.installed_fingerprints, saved: unknown[] = data.saved_fingerprints;
  check(installed.every(fingerprint) && saved.every(fingerprint));
  return { login, revision: data.revision, installed_fingerprints: installed, saved_fingerprints: saved };
}
export class SodaRequestError extends Error {
  constructor(readonly status: number, readonly code?: string) { super('Soda request failed'); }
}
