const sodaPath = '/-/soda';
const credentialQuery = new Set(['code', 'state', 'access_token', 'id_token']);
const nativeSodaHosts = new Set(['spaces', 'repository-spaces', 'runners', 'tailnet']);

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

function credentialFrame(url: URL) {
  for (const key of url.searchParams.keys()) if (credentialQuery.has(key)) return true;
}

function nativeSodaView(url: URL) {
  const values = url.searchParams.getAll('soda-view');
  if (values.length !== 1) return '';
  return values[0];
}

function staysOutsideWorkspace(url: URL, path: string) {
  if (url.searchParams.has('soda-connect') || nativeSodaHosts.has(nativeSodaView(url))) return true;
  return (
    path.startsWith('/user/login') ||
    path.startsWith('/login/oauth') ||
    path === '/install' ||
    path.startsWith('/install/')
  );
}

function pathAfterSubUrl(pathname: string, subUrl: string) {
  if (subUrl && pathname !== subUrl && !pathname.startsWith(subUrl + '/')) return;
  return (subUrl ? pathname.slice(subUrl.length) : pathname) || '/';
}

export function workspaceFrameLocator(raw: string): string | undefined {
  const url = parseFrameURL(raw);
  if (!url || url.origin !== 'https://workspace.invalid' || url.hash || url.username) return;
  if (sodaFramePath(url.pathname) || credentialFrame(url) || staysOutsideWorkspace(url, url.pathname)) return;
  return url.pathname + url.search;
}

export function workspaceAuthLocation(raw: string): string | undefined {
  const url = parseFrameURL(raw);
  if (!url || url.origin !== 'https://workspace.invalid') return;
  if (sodaFramePath(url.pathname) || credentialFrame(url) || staysOutsideWorkspace(url, url.pathname)) {
    return url.pathname + url.search;
  }
}

export function workspaceFrameAction(raw: string) {
  const leave = workspaceAuthLocation(raw);
  if (leave) return {leave};
  const to = workspaceFrameLocator(raw);
  return to === undefined ? {} : {to};
}

function syncWorkspaceTo(win: Window, to: string) {
  const url = new URL(win.location.href);
  url.search = '';
  if (to !== '/') url.searchParams.set('to', to);
  const href = url.pathname + url.search;
  if (href !== win.location.pathname + win.location.search) win.history.replaceState(null, '', href);
}

export function applyWorkspaceFrameNavigation(win: Window, raw: string) {
  const next = workspaceFrameAction(raw);
  if (next.leave) {
    win.location.replace(next.leave);
    return;
  }
  if (next.to === undefined) return;
  syncWorkspaceTo(win, next.to);
}

export function workspaceEntryLocation(href: string, subUrl: string): string | undefined {
  try {
    const url = new URL(href);
    const path = pathAfterSubUrl(url.pathname, subUrl);
    if (!path || staysOutsideWorkspace(url, path)) return;
    const to = workspaceFrameLocator(path + url.search);
    if (to === undefined) return;
    return (subUrl || '') + '/-/soda/workspace' + (to === '/' ? '' : '?to=' + encodeURIComponent(to));
  } catch {
    return;
  }
}
