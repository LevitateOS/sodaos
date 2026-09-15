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
export interface Session {
  user: {id: string; login: string};
  csrf_token: string;
  forgejo_url: string;
}
export function sessionResponse(value: unknown, origin: string): Session {
  const data = object(value),
    user = object(data.user);
  check(
    id(user.id) &&
      typeof user.login === 'string' &&
      typeof data.csrf_token === 'string' &&
      /^[A-Za-z0-9_-]{1,128}$/.test(data.csrf_token) &&
      data.forgejo_url === origin
  );
  return {user: {id: user.id, login: user.login}, csrf_token: data.csrf_token, forgejo_url: origin};
}
export interface CreationProfile {
  id: string;
  distribution: string;
  version: string;
  interface: string;
  architecture: string;
  image: string;
  revision: string;
}
function rockyHeadless(
  p: Record<string, unknown>
): Pick<CreationProfile, 'id' | 'distribution' | 'interface' | 'version'> {
  check(
    p.id === 'rocky-headless' &&
      p.distribution === 'rocky' &&
      p.interface === 'headless' &&
      typeof p.version === 'string' &&
      /^[0-9]{1,3}(\.[0-9]{1,3}){0,2}$/.test(p.version)
  );
  return {id: p.id, distribution: p.distribution, version: p.version, interface: p.interface};
}
function profileImage(p: Record<string, unknown>): Pick<CreationProfile, 'architecture' | 'image' | 'revision'> {
  check(p.architecture === 'amd64' || p.architecture === 'arm64');
  check(
    typeof p.image === 'string' &&
      /^sha256:[0-9a-f]{64}$/.test(p.image) &&
      typeof p.revision === 'string' &&
      /^[0-9a-f]{40}$/.test(p.revision)
  );
  return {architecture: p.architecture, image: p.image, revision: p.revision};
}
export function creationProfile(value: unknown): CreationProfile {
  const p = object(value);
  return {...rockyHeadless(p), ...profileImage(p)};
}
export interface Environment {
  id: string;
  repository_id: string;
  profile?: CreationProfile | null;
}
export function environmentResponse(value: unknown, repositoryId: string): Environment {
  const data = object(value);
  check(projectId(data.id) && data.repository_id === repositoryId);
  return {
    id: data.id,
    repository_id: repositoryId,
    profile: data.profile == null ? null : creationProfile(data.profile),
  };
}
export interface Detail {
  environment: Environment & {provisioned: boolean};
  login: string;
  environment_administrator: boolean;
  native_unavailable: boolean;
  authority_unavailable: boolean;
  observed: {id: string; running: boolean} | null;
}
export function detailResponse(value: unknown, environment: Environment): Detail {
  const data = object(value),
    env = object(data.environment);
  check(
    env.id === environment.id &&
      env.repository_id === environment.repository_id &&
      typeof env.provisioned === 'boolean' &&
      typeof data.login === 'string' &&
      typeof data.environment_administrator === 'boolean' &&
      typeof data.native_unavailable === 'boolean' &&
      typeof data.authority_unavailable === 'boolean'
  );
  let observed: Detail['observed'] = null;
  if (data.observed !== null) {
    const state = object(data.observed);
    check(state.id === environment.id && typeof state.running === 'boolean');
    observed = {id: environment.id, running: state.running};
  }
  return {
    environment: {...environmentResponse(env, environment.repository_id), provisioned: env.provisioned},
    login: data.login,
    environment_administrator: data.environment_administrator,
    native_unavailable: data.native_unavailable,
    authority_unavailable: data.authority_unavailable,
    observed,
  };
}
export interface OSObservation {
  running: boolean;
  image: string | null;
  release: {id: string; version: string; name: string} | null;
}
function osImage(env: Record<string, unknown>) {
  const image = env.image === undefined ? null : env.image;
  check(image === null || (typeof image === 'string' && /^sha256:[0-9a-f]{64}$/.test(image)));
  return image as string | null;
}
function osReleaseIdentity(
  r: Record<string, unknown>,
  env: Record<string, unknown>,
  data: Record<string, unknown>
): {id: string; version: string} {
  check(
    env.running && !data.os_release_unavailable && typeof r.id === 'string' && /^[a-z0-9][a-z0-9._-]{0,63}$/.test(r.id)
  );
  check(typeof r.version === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/.test(r.version));
  return {id: r.id, version: r.version};
}
function osReleaseName(r: Record<string, unknown>): string {
  check(
    typeof r.name === 'string' &&
      r.name.length > 0 &&
      new TextEncoder().encode(r.name).length <= 256 &&
      !/[\p{Cc}\p{Cf}]/u.test(r.name)
  );
  return r.name;
}
function osRelease(data: Record<string, unknown>, env: Record<string, unknown>): OSObservation['release'] {
  if (data.os_release === null) {
    check(data.os_release_unavailable);
    return null;
  }
  const r = object(data.os_release);
  return {...osReleaseIdentity(r, env, data), name: osReleaseName(r)};
}
export function osObservation(value: unknown, environmentID: string): OSObservation {
  const data = object(value),
    env = object(data.environment);
  check(
    env.id === environmentID && typeof env.running === 'boolean' && typeof data.os_release_unavailable === 'boolean'
  );
  return {running: env.running, image: osImage(env), release: osRelease(data, env)};
}
export interface SavedKey {
  id: string;
  fingerprint: string;
  public_key?: string;
}
export function savedKeysResponse(value: unknown): SavedKey[] {
  const data = object(value);
  check(Array.isArray(data.items));
  return data.items.map((value: unknown) => {
    const key = object(value);
    check(id(key.id) && fingerprint(key.fingerprint));
    return {
      id: key.id,
      fingerprint: key.fingerprint,
      ...(typeof key.public_key === 'string' ? {public_key: key.public_key} : {}),
    };
  });
}
export interface ProfileKeys {
  items: (SavedKey & {public_key: string; title: string})[];
  page: number;
  more: boolean;
}
export function profileKeysResponse(value: unknown, page: number): ProfileKeys {
  const data = object(value);
  check(data.page === page && typeof data.more === 'boolean' && Array.isArray(data.items) && data.items.length <= 10);
  const rawItems: unknown[] = data.items,
    keys = savedKeysResponse(data);
  const items = keys.map((key, index) => {
    const row = object(rawItems[index]);
    check(typeof key.public_key === 'string' && typeof row.title === 'string');
    return {...key, public_key: key.public_key, title: row.title};
  });
  return {items, page, more: data.more};
}
export interface KeyPreview {
  login: string;
  revision: string;
  installed_fingerprints: string[];
  saved_fingerprints: string[];
}
export function keyPreviewResponse(value: unknown, login: string): KeyPreview {
  const data = object(value);
  check(
    data.login === login &&
      typeof data.revision === 'string' &&
      /^[0-9a-f]{64}$/.test(data.revision) &&
      Array.isArray(data.installed_fingerprints) &&
      Array.isArray(data.saved_fingerprints)
  );
  const installed: unknown[] = data.installed_fingerprints,
    saved: unknown[] = data.saved_fingerprints;
  check(installed.every(fingerprint) && saved.every(fingerprint));
  return {login, revision: data.revision, installed_fingerprints: installed, saved_fingerprints: saved};
}
export const terminalID = (value: unknown): value is string =>
  typeof value === 'string' && /^[0-9a-f]{32}$/.test(value);
export interface TerminalIdentity {
  expectedUserId: string;
  repositoryId: string;
  environmentId: string;
  login: string;
}
export interface TerminalMetadata {
  id: string;
  environment_id: string;
  repository_id: string;
  user_id: string;
  login: string;
  name: string;
  created_at: number;
  ready: boolean;
  attached: boolean;
  state: 'opening' | 'ready' | 'ending' | 'ended';
}
function admitTerminalBinding(data: Record<string, unknown>, binding: TerminalIdentity): string {
  check(
    terminalID(data.id) &&
      data.environment_id === binding.environmentId &&
      data.repository_id === binding.repositoryId &&
      data.user_id === binding.expectedUserId &&
      data.login === binding.login
  );
  return data.id;
}
function admitTerminalName(data: Record<string, unknown>): string {
  check(typeof data.name === 'string' && Array.from(data.name).length <= 80 && !/[\p{Cc}\p{Cf}]/u.test(data.name));
  return data.name;
}
function admitTerminalClock(data: Record<string, unknown>): number {
  check(typeof data.created_at === 'number' && Number.isSafeInteger(data.created_at) && data.created_at > 0);
  return data.created_at;
}
function knownTerminalState(state: unknown): state is TerminalMetadata['state'] {
  return state === 'opening' || state === 'ready' || state === 'ending' || state === 'ended';
}
function admitTerminalState(data: Record<string, unknown>): Pick<TerminalMetadata, 'ready' | 'attached' | 'state'> {
  check(typeof data.ready === 'boolean' && typeof data.attached === 'boolean' && knownTerminalState(data.state));
  check(data.ready === (data.state === 'ready') && (!data.attached || data.ready));
  return {ready: data.ready, attached: data.attached, state: data.state};
}
export function terminalMetadata(value: unknown, binding: TerminalIdentity): TerminalMetadata {
  const data = object(value);
  return {
    id: admitTerminalBinding(data, binding),
    environment_id: binding.environmentId,
    repository_id: binding.repositoryId,
    user_id: binding.expectedUserId,
    login: binding.login,
    name: admitTerminalName(data),
    created_at: admitTerminalClock(data),
    ...admitTerminalState(data),
  };
}
export function terminalResponse(value: unknown, binding: TerminalIdentity): TerminalMetadata | null {
  const data = object(value);
  return data.terminal === null ? null : terminalMetadata(data.terminal, binding);
}
export interface Space {
  tailnet_state?: string;
  environment: Environment & {name: string; repository: string; owner_id: string; provisioned: boolean};
  login: string;
  environment_administrator: boolean;
  authority_unavailable: boolean;
  native_unavailable: boolean;
  observed: Detail['observed'];
  terminals: TerminalMetadata[];
}
function spaceNetwork(data: Record<string, unknown>) {
  const network = data.tailnet_state;
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
      (!detail.authority_unavailable || (!detail.environment_administrator && row.terminals.length === 0))
  );
  const terminals = row.terminals.map((value: unknown) =>
    admitSpaceTerminal(value, expectedUserId, repositoryId, environmentId, detail.login, sessions)
  );
  check(sessions.size <= 64);
  return terminals;
}
function spaceItem(
  value: unknown,
  expectedUserId: string,
  data: Record<string, unknown>,
  seen: Set<string>,
  sessions: Set<string>
): Space {
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
  const network = spaceNetwork(data);
  return {
    ...detail,
    ...(typeof network === 'string' ? {tailnet_state: network} : {}),
    environment: {...detail.environment, name: env.name, repository: env.repository, owner_id: env.owner_id},
    terminals,
  };
}
export function spacesResponse(value: unknown, expectedUserId: string): {items: Space[]; complete: boolean} {
  const data = object(value);
  check(
    id(expectedUserId) && typeof data.complete === 'boolean' && Array.isArray(data.items) && data.items.length <= 32
  );
  const seen = new Set<string>(),
    sessions = new Set<string>();
  const items = data.items.map((row: unknown) => spaceItem(row, expectedUserId, data, seen, sessions));
  return {items, complete: data.complete};
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
  more: boolean;
  limited: boolean;
}
function admitChoicesPage(data: Record<string, unknown>, page: number) {
  check(Number.isInteger(page) && page >= 1 && page <= 100 && data.page === page);
}
function admitChoicesFlags(data: Record<string, unknown>, page: number): {more: boolean; limited: boolean} {
  check(
    typeof data.more === 'boolean' &&
      typeof data.limited === 'boolean' &&
      !(data.more && data.limited) &&
      (!data.more || page < 100) &&
      (!data.limited || page === 100)
  );
  return {more: data.more, limited: data.limited};
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
  admitChoicesPage(data, page);
  const flags = admitChoicesFlags(data, page);
  check(Array.isArray(data.items) && data.items.length <= 12);
  const seen = new Set<string>();
  const items = data.items.map((raw: unknown) => repositoryChoice(raw, seen));
  return {items, page, ...flags};
}
export class SodaRequestError extends Error {
  constructor(
    readonly status: number,
    readonly code?: string
  ) {
    super('Soda request failed');
  }
}
