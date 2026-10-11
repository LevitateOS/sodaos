import {check, object} from './sodaspaces-api.js';

/** H-R permission/usage DTO v1 (revision f06-20261011-p1). Planning allowance follows H-A. */
export interface FactoryAccessViewer {
  code_write: boolean;
  owner_or_admin: boolean;
  environment_owner: boolean;
  operator: boolean;
  member: boolean;
  session_current: boolean;
}
export interface FactoryAccessEditable {
  editable: boolean;
  reason?: string;
}
export interface FactoryAccessConnectionEditable extends FactoryAccessEditable {
  connection: string;
}
export interface FactoryAccessRequiredAction {
  code: string;
  resolve_by: string;
  setting: string;
}
export interface FactoryAccessUsage {
  connection: string;
  status?: string;
  unit?: string;
  budget_minutes?: number;
  used_minutes?: number;
}
export interface FactoryAccess {
  viewer: FactoryAccessViewer;
  editable: {
    policy: FactoryAccessEditable;
    activity_settings: FactoryAccessEditable;
    operator_grant: FactoryAccessEditable;
    capacity: FactoryAccessEditable;
    environment_grant: FactoryAccessEditable;
    sponsorships: FactoryAccessConnectionEditable[];
    usage_budgets: FactoryAccessConnectionEditable[];
  };
  effective: Record<string, unknown>;
  required_actions: FactoryAccessRequiredAction[];
  preparation: {project?: string; ready?: boolean; requirements?: unknown; approval?: unknown} | null;
  usage: FactoryAccessUsage[];
  observed_unix: number;
  repository: string;
}

function reason(value: unknown): string | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'string' && /^[a-z_]{1,64}$/.test(value));
  return value;
}
function editable(value: unknown): FactoryAccessEditable {
  const row = object(value);
  check(typeof row.editable === 'boolean');
  const parsed: FactoryAccessEditable = {editable: row.editable};
  const code = reason(row.reason);
  if (code !== undefined) parsed.reason = code;
  return parsed;
}
function connectionEditable(value: unknown): FactoryAccessConnectionEditable {
  const row = object(value);
  check(typeof row.connection === 'string' && row.connection.length > 0);
  return {connection: row.connection, ...editable(value)};
}
function minutes(value: unknown): number | undefined {
  if (value === undefined) return undefined;
  check(typeof value === 'number' && Number.isFinite(value) && value >= 0);
  return value;
}
function usageEntry(value: unknown): FactoryAccessUsage {
  const row = object(value);
  check(typeof row.connection === 'string');
  const status = reason(row.status);
  if (status !== undefined) return {connection: row.connection, status};
  check(row.unit === 'minutes');
  const entry: FactoryAccessUsage = {connection: row.connection, unit: 'minutes'};
  const budget = minutes(row.budget_minutes),
    used = minutes(row.used_minutes);
  if (budget !== undefined) entry.budget_minutes = budget;
  if (used !== undefined) entry.used_minutes = used;
  return entry;
}
function requiredAction(value: unknown): FactoryAccessRequiredAction {
  const row = object(value);
  check(
    typeof row.code === 'string' &&
      row.code.length > 0 &&
      typeof row.resolve_by === 'string' &&
      row.resolve_by.length > 0 &&
      typeof row.setting === 'string' &&
      row.setting.length > 0
  );
  return {code: row.code, resolve_by: row.resolve_by, setting: row.setting};
}
function preparation(value: unknown): FactoryAccess['preparation'] {
  if (value === null) return null;
  const row = object(value);
  check(row.project === undefined || typeof row.project === 'string');
  check(row.ready === undefined || typeof row.ready === 'boolean');
  return {
    ...(typeof row.project === 'string' ? {project: row.project} : {}),
    ...(typeof row.ready === 'boolean' ? {ready: row.ready} : {}),
    ...(row.requirements !== undefined ? {requirements: row.requirements} : {}),
    ...(row.approval !== undefined ? {approval: row.approval} : {}),
  };
}
export function factoryAccessResponse(value: unknown): FactoryAccess {
  const data = object(value);
  const viewer = object(data.viewer);
  for (const key of ['code_write', 'owner_or_admin', 'environment_owner', 'operator', 'member', 'session_current'])
    check(typeof viewer[key] === 'boolean');
  const edit = object(data.editable);
  check(Array.isArray(edit.sponsorships) && Array.isArray(edit.usage_budgets));
  check(Array.isArray(data.required_actions) && Array.isArray(data.usage));
  check(typeof data.observed_unix === 'number' && Number.isFinite(data.observed_unix));
  check(typeof data.repository === 'string' && data.repository.length > 0);
  return {
    viewer: {
      code_write: viewer.code_write as boolean,
      owner_or_admin: viewer.owner_or_admin as boolean,
      environment_owner: viewer.environment_owner as boolean,
      operator: viewer.operator as boolean,
      member: viewer.member as boolean,
      session_current: viewer.session_current as boolean,
    },
    editable: {
      policy: editable(edit.policy),
      activity_settings: editable(edit.activity_settings),
      operator_grant: editable(edit.operator_grant),
      capacity: editable(edit.capacity),
      environment_grant: editable(edit.environment_grant),
      sponsorships: edit.sponsorships.map(connectionEditable),
      usage_budgets: edit.usage_budgets.map(connectionEditable),
    },
    effective: object(data.effective),
    required_actions: (data.required_actions as unknown[]).map(requiredAction),
    preparation: preparation(data.preparation),
    usage: (data.usage as unknown[]).map(usageEntry),
    observed_unix: data.observed_unix,
    repository: data.repository,
  };
}
