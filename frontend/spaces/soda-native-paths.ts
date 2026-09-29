import type {ExtensionMountContext} from './soda-extension.js';

// Only native Forgejo navigation needs its installation prefix. API authority
// remains entirely with the mount transport.
export function forgejoPrefix(context: ExtensionMountContext): string {
  const base = new URL(context.apiBase, location.origin);
  const suffix = context.pageId
    ? `/-/extensions/pages/soda/${context.pageId}/api/`
    : `/-/extensions/panels/soda/${context.panelId}/api/`;
  validateBase(base, suffix);
  const prefix = base.pathname.slice(0, -suffix.length);
  validatePrefix(prefix);
  return prefix;
}

function validateBase(base: URL, suffix: string) {
  if (
    base.origin !== location.origin ||
    base.username ||
    base.password ||
    !base.pathname.endsWith(suffix) ||
    base.search ||
    base.hash
  )
    throw Error('Invalid native extension API base');
}

function validatePrefix(prefix: string) {
  if (
    prefix &&
    (!/^\/(?:[^/\\?#\s]+)(?:\/[^/\\?#\s]+)*$/u.test(prefix) ||
      /%2f|%5c/i.test(prefix) ||
      prefix.split('/').some((part) => part === '.' || part === '..'))
  )
    throw Error('Invalid Forgejo installation prefix');
}
