import {check, id, object} from './sodaspaces-api.js';

export type ProviderID = 'codex' | 'muse';
export interface Connection {
  provider_id: ProviderID;
  id: string;
  owner_id: string;
  label: string;
  email: string;
  plan: string;
  state: string;
}
export interface Enrollment {
  provider_id: ProviderID;
  id: string;
  verification_url: string;
  user_code: string;
  state: string;
}
export interface Grant {
  id: string;
  connection_id: string;
  user_id: string;
  project_id: string;
  revoked: boolean;
}
export interface Lease {
  id: string;
  connection_id: string;
  actor_id: string;
  project_id: string;
  kind: string;
  execution_id: string;
}

function text(value: unknown): string {
  check(typeof value === 'string' && value.length <= 2048);
  return value;
}
function identityID(value: unknown): string {
  const result = text(value);
  check(/^[A-Za-z0-9_-]{1,128}$/.test(result));
  return result;
}
function provider(value: unknown): ProviderID {
  check(value === 'codex' || value === 'muse');
  return value;
}
export function connectionView(value: unknown, actor: string): Connection {
  const data = object(value);
  check(data.owner_id === actor && id(actor));
  return availableConnectionView(data);
}
export function availableConnectionView(value: unknown): Connection {
  const data = object(value);
  check(id(data.owner_id));
  return {
    provider_id: provider(data.provider_id),
    id: identityID(data.id),
    owner_id: data.owner_id,
    label: text(data.label),
    email: text(data.email),
    plan: text(data.plan),
    state: text(data.state),
  };
}
export function enrollmentView(value: unknown): Enrollment {
  const data = object(value),
    url = text(data.verification_url),
    providerID = provider(data.provider_id);
  if (url) {
    const parsed = new URL(url);
    check(parsed.origin === enrollmentOrigin(providerID) && !parsed.username && !parsed.password);
  }
  return {
    provider_id: providerID,
    id: identityID(data.id),
    verification_url: url,
    user_code: text(data.user_code),
    state: text(data.state),
  };
}
function enrollmentOrigin(providerID: ProviderID): string {
  if (providerID === 'muse') return 'https://auth.meta.com';
  return 'https://auth.openai.com';
}
export function grantView(value: unknown): Grant {
  const data = object(value);
  check(id(data.user_id) && typeof data.revoked === 'boolean');
  return {
    id: identityID(data.id),
    connection_id: identityID(data.connection_id),
    user_id: data.user_id,
    project_id: text(data.project_id),
    revoked: data.revoked,
  };
}
export function leaseView(value: unknown): Lease {
  const data = object(value);
  check(id(data.actor_id));
  check(data.kind === 'terminal' || data.kind === 'factory');
  return {
    id: identityID(data.id),
    connection_id: identityID(data.connection_id),
    actor_id: data.actor_id,
    project_id: text(data.project_id),
    kind: data.kind,
    execution_id: text(data.execution_id),
  };
}
export function items<T>(value: unknown, parse: (item: unknown) => T): T[] {
  check(Array.isArray(value) && value.length <= 128);
  return value.map(parse);
}
