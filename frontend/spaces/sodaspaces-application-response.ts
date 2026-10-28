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
  bundle_file_sha256?: string;
}
export type ApplicationEnvironment = 'company' | 'preview' | 'unknown';
export interface ApplicationFactoryDelivery {
  environment: 'company' | 'preview';
  state: 'verified';
  intent: string;
  revision: string;
  delivery_id: string;
  run: string;
  source_commit: string;
  bundle_file_sha256: string;
}
export interface ApplicationRecord {
  environment: ApplicationEnvironment;
  project?: {repository: string; name: string; path: string};
  application: ApplicationSpec;
  administrator_id: string;
  observation: ApplicationObservation | null;
  operator: boolean;
  delivery?: ApplicationFactoryDelivery;
  restore?: ApplicationRestoreOperation;
}
export interface ApplicationRestoreOperation {
  id: string;
  snapshot: string;
  previous: ApplicationSpec;
  next: ApplicationSpec;
  expected_container: string;
  state: 'pending';
}
export interface ApplicationSnapshot {
  id: string;
  created_at: number;
  application: ApplicationSpec;
}
export interface ApplicationSnapshotInventory {
  snapshots: ApplicationSnapshot[];
  complete: boolean;
}
function applicationEnvironment(value: unknown): ApplicationEnvironment {
  check(value === undefined || value === 'company' || value === 'preview' || value === 'unknown');
  return value === undefined ? 'unknown' : value;
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
    Object.keys(o).every((key) =>
      ['id', 'revision', 'container', 'image', 'running', 'ready', 'origin', 'bundle_file_sha256'].includes(key)
    ) &&
    (o.bundle_file_sha256 === undefined || sha(o.bundle_file_sha256)) &&
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
  check(
    (sha(o.container) || (o.container === '' && o.running === false && o.ready === false)) &&
      typeof o.running === 'boolean' &&
      typeof o.ready === 'boolean' &&
      (!o.ready || o.running)
  );
  return {
    id: application.id,
    revision: application.revision,
    container: o.container,
    image: application.image,
    running: o.running,
    ready: o.ready,
    origin: application.origin,
    ...(typeof o.bundle_file_sha256 === 'string' ? {bundle_file_sha256: o.bundle_file_sha256} : {}),
  };
}
export function applicationResponse(value: unknown, expected?: ApplicationSpec): ApplicationRecord {
  const v = object(value),
    application = spec(v.application);
  check(
    Object.keys(v).every((key) =>
      [
        'application',
        'administrator_id',
        'observation',
        'operator',
        'project',
        'restore',
        'delivery',
        'environment',
      ].includes(key)
    ) &&
      id(v.administrator_id) &&
      typeof v.operator === 'boolean'
  );
  let project: ApplicationRecord['project'];
  if (v.project !== undefined) {
    const p = object(v.project);
    check(
      id(p.repository) &&
        typeof p.name === 'string' &&
        typeof p.path === 'string' &&
        p.path === '/' + p.name &&
        !/[?#\\\s]/.test(p.path)
    );
    project = {repository: p.repository, name: p.name, path: p.path};
  }
  if (expected) check(JSON.stringify(application) === JSON.stringify(expected));
  let restore: ApplicationRestoreOperation | undefined;
  if (v.restore !== undefined) {
    const row = object(v.restore),
      previous = spec(row.previous),
      next = spec(row.next);
    check(
      Object.keys(row).length === 6 &&
        appID(row.id) &&
        appID(row.snapshot) &&
        sha(row.expected_container) &&
        row.state === 'pending'
    );
    check(
      JSON.stringify(previous) === JSON.stringify(application) &&
        next.revision === previous.revision + 1 &&
        next.data !== previous.data
    );
    check(JSON.stringify({...next, revision: previous.revision, data: previous.data}) === JSON.stringify(previous));
    restore = {
      id: row.id,
      snapshot: row.snapshot,
      previous,
      next,
      expected_container: row.expected_container,
      state: 'pending',
    };
  }
  let delivery: ApplicationFactoryDelivery | undefined;
  if (v.delivery !== undefined) {
    const row = object(v.delivery);
    const reference = (value: unknown): value is string =>
      typeof value === 'string' && /^[a-zA-Z0-9_.:-]{1,160}$/.test(value);
    check(
      Object.keys(row).length === 8 &&
        (row.environment === 'company' || row.environment === 'preview') &&
        row.state === 'verified' &&
        reference(row.intent) &&
        reference(row.revision) &&
        reference(row.delivery_id) &&
        reference(row.run) &&
        typeof row.source_commit === 'string' &&
        /^[a-f0-9]{40}$/.test(row.source_commit) &&
        sha(row.bundle_file_sha256)
    );
    delivery = {
      environment: row.environment as 'company' | 'preview',
      state: 'verified',
      intent: row.intent,
      revision: row.revision,
      delivery_id: row.delivery_id,
      run: row.run,
      source_commit: row.source_commit,
      bundle_file_sha256: row.bundle_file_sha256,
    };
    check(object(v.observation).bundle_file_sha256 === delivery.bundle_file_sha256);
  }
  return {
    application,
    environment: applicationEnvironment(v.environment),
    ...(delivery ? {delivery} : {}),
    administrator_id: v.administrator_id,
    operator: v.operator,
    observation: observation(v.observation, application),
    ...(project ? {project} : {}),
    ...(restore ? {restore} : {}),
  };
}
export interface ApplicationLaunchRecord {
  environment: ApplicationEnvironment;
  delivery_id?: string;
  project?: {repository: string; name: string; path: string};
  application: Pick<ApplicationSpec, 'id' | 'revision' | 'name' | 'origin'>;
  running: boolean;
  ready: boolean;
  state: 'ready' | 'running' | 'stopped' | 'not-installed' | 'unknown';
  launch_only: true;
}
export function applicationLaunch(value: unknown): ApplicationLaunchRecord {
  const v = object(value),
    a = object(v.application);
  check(
    Object.keys(a).length === 4 &&
      appID(a.id) &&
      revision(a.revision) &&
      typeof a.name === 'string' &&
      /^[a-zA-Z0-9_.-]{1,128}$/.test(a.name) &&
      origin(a.origin)
  );
  const application = {id: a.id, revision: a.revision, name: a.name, origin: a.origin};
  check(
    Object.keys(v).every((key) =>
      ['application', 'running', 'ready', 'project', 'launch_only', 'state', 'environment', 'delivery_id'].includes(key)
    ) &&
      typeof v.running === 'boolean' &&
      typeof v.ready === 'boolean' &&
      (!v.ready || v.running) &&
      v.launch_only === true &&
      (v.state === undefined || ['ready', 'running', 'stopped', 'not-installed', 'unknown'].includes(String(v.state)))
  );
  check(v.delivery_id === undefined || (typeof v.delivery_id === 'string' && /^[a-f0-9]{32}$/.test(v.delivery_id)));
  let project: ApplicationLaunchRecord['project'];
  if (v.project !== undefined) {
    const p = object(v.project);
    check(
      id(p.repository) &&
        typeof p.name === 'string' &&
        typeof p.path === 'string' &&
        p.path === '/' + p.name &&
        !/[?#\\\s]/.test(p.path)
    );
    project = {repository: p.repository, name: p.name, path: p.path};
  }
  // The launch projection never carries the administrator grant, the native
  // container generation or lifecycle controls; only open/authenticate.
  const state = v.state ?? (v.ready ? 'ready' : v.running ? 'running' : 'unknown');
  check((state === 'ready') === v.ready && ['ready', 'running'].includes(String(state)) === v.running);
  return {
    application,
    environment: applicationEnvironment(v.environment),
    ...(typeof v.delivery_id === 'string' ? {delivery_id: v.delivery_id} : {}),
    running: v.running,
    ready: v.ready,
    state: state as ApplicationLaunchRecord['state'],
    ...(project ? {project} : {}),
    launch_only: true,
  };
}
export function applicationsResponse(value: unknown): {
  applications: ApplicationRecord[];
  launchable: ApplicationLaunchRecord[];
  complete: boolean;
  can_setup: boolean;
} {
  const v = object(value);
  check(
    Object.keys(v).every((key) => ['applications', 'launchable', 'complete', 'can_setup'].includes(key)) &&
      Array.isArray(v.applications) &&
      typeof v.complete === 'boolean' &&
      (v.can_setup === undefined || typeof v.can_setup === 'boolean')
  );
  const applications = v.applications.map((item) => applicationResponse(item));
  check(new Set(applications.map((item) => item.application.id)).size === applications.length);
  // Older servers omit the launch projection; launch-only employees need it.
  const launchable = Array.isArray(v.launchable) ? v.launchable.map((item) => applicationLaunch(item)) : [];
  check(new Set(launchable.map((item) => item.application.id)).size === launchable.length);
  check(!launchable.some((item) => applications.some((admin) => admin.application.id === item.application.id)));
  return {
    applications,
    launchable,
    complete: v.complete,
    can_setup: v.can_setup === true || (v.can_setup === undefined && applications.some((record) => record.operator)),
  };
}

export function applicationSnapshotsResponse(value: unknown, expected: ApplicationSpec): ApplicationSnapshotInventory {
  const v = object(value);
  check(
    Object.keys(v).length === 2 &&
      Array.isArray(v.snapshots) &&
      v.snapshots.length <= 64 &&
      typeof v.complete === 'boolean'
  );
  const snapshots = v.snapshots.map((item) => {
    const row = object(item),
      application = spec(row.application);
    check(
      Object.keys(row).length === 3 &&
        appID(row.id) &&
        typeof row.created_at === 'number' &&
        Number.isSafeInteger(row.created_at) &&
        row.created_at > 0
    );
    check(
      application.id === expected.id &&
        application.name === expected.name &&
        application.owner_team === expected.owner_team &&
        application.origin === expected.origin &&
        application.port === expected.port
    );
    return {id: row.id, created_at: row.created_at, application};
  });
  check(new Set(snapshots.map((row) => row.id)).size === snapshots.length);
  return {snapshots, complete: v.complete};
}
