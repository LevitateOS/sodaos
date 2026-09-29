// Native host inputs for the existing Lit page/workspace adapters. No actor or
// repository displayed by the browser is accepted as operation authority.
export interface ExtensionMountContext {
  extensionId: string;
  pageId?: string;
  panelId?: string;
  apiBase: string;
  assetBase: string;
  sessionGeneration: string;
}

function nativeBase(value: string, origin: string): URL {
  const base = new URL(value, origin);
  if (
    base.origin !== origin ||
    base.username ||
    base.password ||
    base.search ||
    base.hash ||
    !base.pathname.endsWith('/')
  ) {
    throw Error('Invalid native extension URL');
  }
  return base;
}

// Prepares transport and lifetime for mountSpacesPage/mountSodaspaces. Their
// existing browser-session transport must be ported before they are connected to
// this adapter; preparation alone does not mount a working native workspace.
export function prepareExtensionMount(root: HTMLElement, context: ExtensionMountContext) {
  if (
    context.extensionId !== 'soda' ||
    !context.sessionGeneration ||
    !((context.pageId === 'spaces' && !context.panelId) || (context.panelId === 'workspace' && !context.pageId))
  ) {
    throw Error('Invalid Soda mount context');
  }
  const apiBase = nativeBase(context.apiBase, location.origin);
  const assetBase = nativeBase(context.assetBase, location.origin);
  const lifetime = new AbortController();
  const generation = context.sessionGeneration;
  let disposed = false;
  return {
    root,
    generation,
    assetBase: assetBase.href,
    async request(relativePath: string, init: RequestInit = {}): Promise<Response> {
      if (disposed) throw Error('Soda mount is disposed');
      const url = new URL(relativePath, apiBase);
      if (!url.href.startsWith(apiBase.href) || url.search || url.hash || /%2f|%5c/i.test(url.pathname)) {
        throw Error('Invalid Soda operation path');
      }
      const headers = new Headers(init.headers);
      headers.set('X-Extension-Session-Generation', generation);
      return fetch(url, {
        ...init,
        headers,
        credentials: 'same-origin',
        redirect: 'error',
        cache: 'no-store',
        signal: init.signal ? AbortSignal.any([init.signal, lifetime.signal]) : lifetime.signal,
      });
    },
    dispose() {
      disposed = true;
      lifetime.abort();
    },
  };
}
