import {check, fingerprint, id, object} from './sodaspaces-api.js';

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
