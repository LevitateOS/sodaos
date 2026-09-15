const sodaPath = '/-/soda';
const credentialQuery = new Set(['code', 'state', 'access_token', 'id_token']);

function parseFrameURL(raw: string) {
  if (!raw || raw.length > 2048 || raw[0] !== '/') return;
  if (raw.includes('..') || raw.includes('//') || raw.includes('\\')) return;
  try {
    return new URL(raw, 'https://workspace.invalid');
  } catch {
    return;
  }
}

function sodaFramePath(pathname: string) {
  return pathname === sodaPath || pathname.startsWith(sodaPath + '/');
}

export function workspaceFrameLocator(raw: string): string | undefined {
  const url = parseFrameURL(raw);
  if (!url || url.origin !== 'https://workspace.invalid' || url.hash || url.username) return;
  if (sodaFramePath(url.pathname)) return;
  for (const key of url.searchParams.keys()) if (credentialQuery.has(key)) return;
  return url.pathname + url.search;
}
