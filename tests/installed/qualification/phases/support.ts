import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {gunzipSync} from 'node:zlib';

export const SHA = /^[a-f0-9]{64}$/;
export const COMMIT = /^[a-f0-9]{40}$/;
export function sha(bytes: Uint8Array | string) {
  return createHash('sha256').update(bytes).digest('hex');
}
export function check(value: unknown, reason: string): asserts value {
  assert(value, reason);
}
export function rawRun(url: string) {
  return /factory(?:%2f|\/)runs(?:%2f|\/)/i.test(url);
}
export function packageAssetName(address: string, origin: string, expected: Record<string, string>) {
  const url = new URL(address),
    prefix = '/-/native/assets/soda/';
  if (url.origin !== origin || !url.pathname.startsWith(prefix)) return undefined;
  const relative = url.pathname.slice(prefix.length);
  return Object.hasOwn(expected, relative) ? relative : undefined;
}
export function digestRef(value: string) {
  check(/^sha256:[a-f0-9]{64}$/.test(value), 'image_digest_required');
  return value.slice(7);
}

// The candidate package is public and independently hash-bound by its payload.
// Read only plain regular USTAR asset entries, with the same entry layout used
// by the existing qualified component guards. No archive extraction occurs.
export function packageAssetHashes(compressed: Uint8Array, expectedSHA: string) {
  check(sha(compressed) === expectedSHA, 'exact_ui_package');
  const tar = gunzipSync(compressed, {maxOutputLength: 256 << 20}),
    assets: Record<string, string> = {};
  for (let at = 0; at + 512 <= tar.length;) {
    const header = tar.subarray(at, at + 512);
    if (header.every((x) => x === 0)) break;
    const field = (a: number, b: number) => header.subarray(a, b).toString('utf8').replace(/\0.*$/s, '');
    const leaf = field(0, 100),
      prefix = field(345, 500),
      name = (prefix ? prefix + '/' : '') + leaf,
      type = field(156, 157) || '0';
    check(type === '0' || type === '5', 'regular_ui_entries_only');
    check(name !== '' && !name.startsWith('/') && !name.split('/').includes('..'), 'safe_ui_entry');
    const size = parseInt(field(124, 136).trim() || '0', 8);
    check(Number.isSafeInteger(size) && size >= 0 && at + 512 + size <= tar.length, 'bounded_ui_entry');
    if (type === '0' && name.startsWith('assets/')) {
      const relative = name.slice(7);
      check(relative !== '' && !Object.hasOwn(assets, relative), 'unique_ui_asset');
      assets[relative] = sha(tar.subarray(at + 512, at + 512 + size));
    }
    at += 512 + Math.ceil(size / 512) * 512;
  }
  check(Object.hasOwn(assets, 'soda-spaces-entry.js'), 'workspace_entry_in_candidate_package');
  return assets;
}

export function networkGuard(
  origin: string,
  audit: any,
  loginPhase: () => boolean,
  permit?: (request: any) => Promise<boolean>
) {
  return async (route: any) => {
    const request = route.request(),
      url = new URL(request.url()),
      method = request.method();
    const row = {method, path: url.pathname, blocked: false};
    audit.requests.push(row);
    if (rawRun(url.href)) {
      row.blocked = true;
      audit.blocked_raw_runs++;
      return route.abort('blockedbyclient');
    }
    const login = loginPhase() && method === 'POST' && url.origin === origin && url.pathname === '/user/login';
    let permitted = false;
    if (url.origin === origin && method !== 'GET' && !login && permit) {
      try {
        permitted = await permit(request);
      } catch {
        permitted = false;
      }
    }
    if (url.origin !== origin || (method !== 'GET' && !login && !permitted)) {
      row.blocked = true;
      audit.blocked_other++;
      return route.abort('blockedbyclient');
    }
    if (login) {
      audit.login_posts++;
      check(audit.login_posts === 1, 'one_owner_login_only');
    }
    return route.continue();
  };
}
