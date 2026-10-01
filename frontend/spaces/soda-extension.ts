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

// Prepares the native request transport and lifetime for mounted pages.
export function prepareExtensionMount(root: HTMLElement, context: ExtensionMountContext) {
  if (
    context.extensionId !== 'soda' ||
    !context.sessionGeneration ||
    !(
      (['spaces', 'tailnet'].includes(context.pageId || '') && !context.panelId) ||
      (context.panelId === 'workspace' && !context.pageId)
    )
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
      const url = extensionURL(apiBase, relativePath, true);
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
    websocket(relativePath: string): WebSocket {
      if (disposed) throw Error('Soda mount is disposed');
      const url = extensionURL(apiBase, relativePath, false);
      if (url.search) throw Error('Invalid Soda terminal path');
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      return new WebSocket(url);
    },
    dispose() {
      disposed = true;
      lifetime.abort();
    },
  };
}

function extensionURL(apiBase: URL, relativePath: string, allowQuery: boolean): URL {
  const url = new URL(relativePath, apiBase);
  if (!url.href.startsWith(apiBase.href) || url.hash || (!allowQuery && url.search) || /%2f|%5c/i.test(url.pathname)) {
    throw Error('Invalid Soda operation path');
  }
  return url;
}

export type PreparedExtensionMount = ReturnType<typeof prepareExtensionMount>;
