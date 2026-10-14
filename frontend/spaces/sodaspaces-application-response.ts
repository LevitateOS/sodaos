import {check, id, object} from './sodaspaces-api.js';

export interface ApplicationSpec {
  id: string;
  revision: number;
  name: string;
  artifact_sha256: string;
  bundle_sha256: string;
  tools_sha256: string;
  runtime_image: string;
  image: string;
  data: string;
  owner_team: string;
  origin: string;
  port: number;
}
export interface ApplicationObservation {
  id: string;
  revision: number;
  container: string;
  image: string;
  running: boolean;
  ready: boolean;
  origin: string;
}
export interface ApplicationRecord {
  application: ApplicationSpec;
  administrator_id: string;
  observation: ApplicationObservation | null;
  operator: boolean;
}
const appID = (v: unknown): v is string => typeof v === 'string' && /^a[0-9a-f]{24}$/.test(v);
const sha = (v: unknown): v is string => typeof v === 'string' && /^[0-9a-f]{64}$/.test(v);
const image = (v: unknown): v is string => typeof v === 'string' && /^sha256:[0-9a-f]{64}$/.test(v);
function revision(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value > 0;
}
function origin(value: unknown): value is string {
  if (typeof value !== 'string') return false;
  try {
    const url = new URL(value);
    return (
      url.protocol === 'https:' &&
      url.origin === value &&
      !url.username &&
      !url.password &&
      url.hostname.includes('.') &&
      url.hostname !== 'localhost' &&
      !/^\d+(?:\.\d+){3}$/.test(url.hostname)
    );
  } catch {
    return false;
  }
}
function identityFields(s: Record<string, unknown>) {
  check(
    Object.keys(s).length === 12 &&
      appID(s.id) &&
      revision(s.revision) &&
      typeof s.name === 'string' &&
      /^[a-zA-Z0-9_.-]{1,128}$/.test(s.name)
  );
  return {id: s.id, revision: s.revision, name: s.name};
}
function byteFields(s: Record<string, unknown>) {
  check(
    sha(s.artifact_sha256) && sha(s.bundle_sha256) && sha(s.tools_sha256) && image(s.image) && image(s.runtime_image)
  );
  return {
    artifact_sha256: s.artifact_sha256,
    bundle_sha256: s.bundle_sha256,
    tools_sha256: s.tools_sha256,
    runtime_image: s.runtime_image,
    image: s.image,
  };
}
function teamFields(s: Record<string, unknown>) {
  check(
    appID(s.data) &&
      typeof s.owner_team === 'string' &&
      /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(s.owner_team)
  );
  return {data: s.data, owner_team: s.owner_team};
}
function originFields(s: Record<string, unknown>) {
  check(
    origin(s.origin) && typeof s.port === 'number' && Number.isInteger(s.port) && s.port >= 20000 && s.port < 30000
  );
  return {origin: s.origin, port: s.port};
}
function spec(value: unknown): ApplicationSpec {
  const s = object(value);
  return {...identityFields(s), ...byteFields(s), ...teamFields(s), ...originFields(s)};
}
function observationReferences(o: Record<string, unknown>, application: ApplicationSpec) {
  return (
    Object.keys(o).length === 7 &&
    o.id === application.id &&
    o.revision === application.revision &&
    o.image === application.image &&
    o.origin === application.origin
  );
}
function observation(value: unknown, application: ApplicationSpec): ApplicationObservation | null {
  if (value === null) return null;
  const o = object(value);
  check(observationReferences(o, application));
  check(sha(o.container) && typeof o.running === 'boolean' && typeof o.ready === 'boolean' && (!o.ready || o.running));
  return {
    id: application.id,
    revision: application.revision,
    container: o.container,
    image: application.image,
    running: o.running,
    ready: o.ready,
    origin: application.origin,
  };
}
export function applicationResponse(value: unknown, expected?: ApplicationSpec): ApplicationRecord {
  const v = object(value),
    application = spec(v.application);
  check(Object.keys(v).length === 4 && id(v.administrator_id) && typeof v.operator === 'boolean');
  if (expected) check(JSON.stringify(application) === JSON.stringify(expected));
  return {
    application,
    administrator_id: v.administrator_id,
    operator: v.operator,
    observation: observation(v.observation, application),
  };
}
export function applicationsResponse(value: unknown): {applications: ApplicationRecord[]; complete: boolean} {
  const v = object(value);
  check(
    Object.keys(v).length === 2 &&
      Array.isArray(v.applications) &&
      v.applications.length <= 32 &&
      typeof v.complete === 'boolean'
  );
  const applications = v.applications.map((item) => applicationResponse(item));
  check(new Set(applications.map((item) => item.application.id)).size === applications.length);
  return {applications, complete: v.complete};
}
