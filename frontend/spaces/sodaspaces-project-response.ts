import {check, object, projectId} from './sodaspaces-api.js';

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
  execution_allowed: boolean;
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
  check(typeof data.execution_allowed === 'boolean');
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
    execution_allowed: data.execution_allowed,
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
