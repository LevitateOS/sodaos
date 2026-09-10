// Soda response contracts used by its two browser components. No copied Forgejo authority.

// Never accept HTML, invalid UTF-8 or an unbounded body as Soda JSON.
export async function readSodaJSON(response: Response): Promise<unknown> {
  if (!/^application\/json(?:;|$)/i.test(response.headers.get('Content-Type') || '') || Number(response.headers.get('Content-Length')) > 65536) {
    await response.body?.cancel(); throw Error('Invalid Soda response');
  }
  if (!response.body) throw Error('Missing Soda response body');
  const reader = response.body.getReader(), decoder = new TextDecoder('utf-8', {fatal: true});
  let size = 0, text = '';
  try {
    for (;;) {
      const {done, value} = await reader.read(); if (done) break;
      size += value.byteLength; if (size > 65536) throw Error('Oversized Soda response');
      text += decoder.decode(value, {stream: true});
    }
    return JSON.parse(text + decoder.decode());
  } catch (error) {await reader.cancel(); throw error;}
  finally {reader.releaseLock();}
}
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
export interface CreationProfile {id: string; distribution: string; version: string; interface: string; architecture: string; image: string; revision: string}
export function creationProfile(value: unknown): CreationProfile {
  const p = object(value);
  check(p.id === 'rocky-headless' && p.distribution === 'rocky' && p.interface === 'headless' && typeof p.version === 'string' && /^[0-9]{1,3}(\.[0-9]{1,3}){0,2}$/.test(p.version));
  check((p.architecture === 'amd64' || p.architecture === 'arm64') && typeof p.image === 'string' && /^sha256:[0-9a-f]{64}$/.test(p.image) && typeof p.revision === 'string' && /^[0-9a-f]{40}$/.test(p.revision));
  return {id: p.id, distribution: p.distribution, version: p.version, interface: p.interface, architecture: p.architecture, image: p.image, revision: p.revision};
}
export interface Environment { id: string; repository_id: string; profile?: CreationProfile | null }
export function environmentResponse(value: unknown, repositoryId: string): Environment {
  const data = object(value); check(projectId(data.id) && data.repository_id === repositoryId);
  return { id: data.id, repository_id: repositoryId, profile: data.profile == null ? null : creationProfile(data.profile) };
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
  return { environment: { ...environmentResponse(env, environment.repository_id), provisioned: env.provisioned }, login: data.login, environment_administrator: data.environment_administrator, native_unavailable: data.native_unavailable, authority_unavailable: data.authority_unavailable, observed };
}
export interface SavedKey { id: string; fingerprint: string; public_key?: string }
export function savedKeysResponse(value: unknown): SavedKey[] {
  const data = object(value); check(Array.isArray(data.items));
  return data.items.map((value: unknown) => {
    const key = object(value); check(id(key.id) && fingerprint(key.fingerprint));
    return { id: key.id, fingerprint: key.fingerprint, ...(typeof key.public_key === 'string' ? { public_key: key.public_key } : {}) };
  });
}
export interface ProfileKeys {items: (SavedKey & {public_key: string; title: string})[]; page: number; more: boolean}
export function profileKeysResponse(value: unknown, page: number): ProfileKeys {
  const data = object(value); check(data.page === page && typeof data.more === 'boolean' && Array.isArray(data.items) && data.items.length <= 10);
  const rawItems: unknown[] = data.items, keys = savedKeysResponse(data);
  const items = keys.map((key, index) => {
    const row = object(rawItems[index]);
    check(typeof key.public_key === 'string' && typeof row.title === 'string');
    return {...key, public_key: key.public_key, title: row.title};
  });
  return {items, page, more: data.more};
}
export interface KeyPreview { login: string; revision: string; installed_fingerprints: string[]; saved_fingerprints: string[] }
export function keyPreviewResponse(value: unknown, login: string): KeyPreview {
  const data = object(value);
  check(data.login === login && typeof data.revision === 'string' && /^[0-9a-f]{64}$/.test(data.revision) && Array.isArray(data.installed_fingerprints) && Array.isArray(data.saved_fingerprints));
  const installed: unknown[] = data.installed_fingerprints, saved: unknown[] = data.saved_fingerprints;
  check(installed.every(fingerprint) && saved.every(fingerprint));
  return { login, revision: data.revision, installed_fingerprints: installed, saved_fingerprints: saved };
}
export const terminalID = (value: unknown): value is string => typeof value === 'string' && /^[0-9a-f]{32}$/.test(value);
export interface TerminalIdentity {expectedUserId: string; repositoryId: string; environmentId: string; login: string}
export interface TerminalMetadata {
  id: string; request_id: string; environment_id: string; repository_id: string; user_id: string; login: string; name: string;
  created_at: number; hard_until: number; retain_until: number; effective_until: number;
  ready: boolean; attached: boolean; state: 'opening' | 'ready' | 'ending' | 'unconfirmed' | 'ended';
}
export function terminalMetadata(value: unknown, binding: TerminalIdentity): TerminalMetadata {
  const data = object(value);
  check(terminalID(data.id) && terminalID(data.request_id) && data.environment_id === binding.environmentId && data.repository_id === binding.repositoryId && data.user_id === binding.expectedUserId && data.login === binding.login);
  check(typeof data.name === 'string' && Array.from(data.name).length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(data.name));
  const timestamp = (value: unknown): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
  check(timestamp(data.created_at) && timestamp(data.hard_until) && timestamp(data.retain_until) && timestamp(data.effective_until));
  check(data.created_at > 0 && data.hard_until >= data.created_at && data.hard_until <= data.created_at + 43200 && (data.retain_until === 0 || data.retain_until <= data.hard_until) && data.effective_until === (data.retain_until || data.hard_until));
  check(typeof data.ready === 'boolean' && typeof data.attached === 'boolean' && (data.state === 'opening' || data.state === 'ready' || data.state === 'ending' || data.state === 'unconfirmed' || data.state === 'ended'));
  check(data.state === 'ready' || !data.ready); check(!['ending', 'unconfirmed', 'ended'].includes(data.state) || !data.attached);
  return {id: data.id, request_id: data.request_id, environment_id: binding.environmentId, repository_id: binding.repositoryId, user_id: binding.expectedUserId, login: binding.login, name: data.name,
    created_at: data.created_at, hard_until: data.hard_until, retain_until: data.retain_until, effective_until: data.effective_until, ready: data.ready, attached: data.attached, state: data.state};
}
export function terminalResponse(value: unknown, binding: TerminalIdentity): TerminalMetadata | null {
  const data = object(value); return data.terminal === null ? null : terminalMetadata(data.terminal, binding);
}
export interface Space {
  environment: Environment & {name: string; repository: string; owner_id: string; provisioned: boolean};
  login: string; environment_administrator: boolean; authority_unavailable: boolean; native_unavailable: boolean;
  observed: Detail['observed']; terminals: TerminalMetadata[];
}
export function spacesResponse(value: unknown, expectedUserId: string): {items: Space[]; complete: boolean} {
  const data = object(value); check(id(expectedUserId) && typeof data.complete === 'boolean' && Array.isArray(data.items) && data.items.length <= 32);
  const seen = new Set<string>(), sessions = new Set<string>();
  const items = data.items.map((value: unknown): Space => {
    const row = object(value), env = object(row.environment);
    check(projectId(env.id) && id(env.repository_id) && id(env.owner_id) && typeof env.name === 'string' && typeof env.repository === 'string' && !seen.has(env.id)); seen.add(env.id);
    const environmentId = env.id, repositoryId = env.repository_id;
    const detail = detailResponse(row, {id: environmentId, repository_id: repositoryId});
    check(Array.isArray(row.terminals) && row.terminals.length <= 64 && (!detail.authority_unavailable || (!detail.environment_administrator && row.terminals.length === 0)));
    const terminals = row.terminals.map((value: unknown) => {
      const terminal = terminalMetadata(value, {expectedUserId, repositoryId, environmentId, login: detail.login});
      check(!sessions.has(terminal.id)); sessions.add(terminal.id); return terminal;
    });
    check(sessions.size <= 64);
    return {...detail, environment: {...detail.environment, name: env.name, repository: env.repository, owner_id: env.owner_id}, terminals};
  });
  return {items, complete: data.complete};
}
export class SodaRequestError extends Error {
  constructor(readonly status: number, readonly code?: string) { super('Soda request failed'); }
}
