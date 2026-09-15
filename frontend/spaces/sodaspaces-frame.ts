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

const nativeSodaHosts = new Set(['spaces', 'repository-spaces', 'runners', 'tailnet']);

function pathAfterSubUrl(pathname: string, subUrl: string) {
  if (subUrl && pathname !== subUrl && !pathname.startsWith(subUrl + '/')) return;
  return (subUrl ? pathname.slice(subUrl.length) : pathname) || '/';
}

function nativeSodaView(url: URL) {
  const values = url.searchParams.getAll('soda-view');
  if (values.length !== 1) return '';
  return values[0];
}

function keepWorkspaceEntry(url: URL, path: string) {
  if (url.searchParams.has('soda-connect') || nativeSodaHosts.has(nativeSodaView(url))) return true;
  return (
    path.startsWith('/user/login') ||
    path.startsWith('/login/oauth') ||
    path === '/install' ||
    path.startsWith('/install/')
  );
}

export function workspaceEntryLocation(href: string, subUrl: string): string | undefined {
  try {
    const url = new URL(href);
    const path = pathAfterSubUrl(url.pathname, subUrl);
    if (!path || keepWorkspaceEntry(url, path)) return;
    const to = workspaceFrameLocator(path + url.search);
    if (to === undefined) return;
    return (subUrl || '') + '/-/soda/workspace' + (to === '/' ? '' : '?to=' + encodeURIComponent(to));
  } catch {
    return;
  }
}
