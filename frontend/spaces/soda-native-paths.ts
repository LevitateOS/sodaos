import type {ExtensionMountContext} from './soda-extension.js';

// Only native Forgejo navigation needs its installation prefix. API authority
// remains entirely with the mount transport.
export function nativeAppSubUrl(context: ExtensionMountContext): string {
  const prefix = context.appSubUrl;
  if (
    typeof prefix !== 'string' ||
    (prefix !== '' &&
      (!/^\/(?:[^/\\?#\s]+)(?:\/[^/\\?#\s]+)*$/u.test(prefix) ||
        /%2f|%5c/i.test(prefix) ||
        prefix.split('/').some((part) => part === '.' || part === '..')))
  )
    throw Error('Invalid native app subpath');
  return prefix;
}
