import {nativeAppSubUrl} from './soda-native-paths.js';
import {id} from './sodaspaces-api.js';
import type {MountContext} from './fountain-browser-contract.js';

// Native host inputs for the existing Lit page/workspace adapters. No actor or
// repository displayed by the browser is accepted as operation authority.
export interface ExtensionMountContext {
  extensionId: string;
  pageId?: string;
  panelId?: string;
  appSubUrl: string;
  repositoryId?: string;
  apiBase: string;
  assetBase: string;
  sessionGeneration: string;
  /** Canonical host contract version; entries refuse any other value before effects. */
  contractVersion?: number;
  /** Host-scoped request helper (Fountain MountContext); preferred over the local fallback. */
  request?: MountContext['request'];
  /** Host-scoped stream helper (Fountain MountContext); preferred over the local fallback. */
  stream?: MountContext['stream'];
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
    throw Error('Invalid native app URL');
  }
  return base;
}

// Prepares the native request transport and lifetime for mounted pages.
export function prepareExtensionMount(root: HTMLElement, context: ExtensionMountContext) {
  // Refuse an unexpected canonical contract version before any effects.
  if (context.contractVersion !== undefined && context.contractVersion !== 1) {
    throw Error('Unsupported native contract version');
  }
  if (
    context.extensionId !== 'soda' ||
    !context.sessionGeneration ||
    !(
      (['spaces', 'tailnet'].includes(context.pageId || '') && !context.panelId) ||
      (context.panelId === 'workspace' && !context.pageId)
    )
  ) {
    throw Error('Invalid workspace mount context');
  }
  nativeAppSubUrl(context);
  validateRepository(context.repositoryId);
  const apiBase = nativeBase(context.apiBase, location.origin);
  const assetBase = nativeBase(context.assetBase, location.origin);
  const lifetime = new AbortController();
  const generation = context.sessionGeneration;
  let disposed = false;
  const hostRequest = context.request;
  const hostStream = context.stream;
  return {
    root,
    generation,
    assetBase: assetBase.href,
    async request(relativePath: string, init: RequestInit = {}): Promise<Response> {
      if (disposed) throw Error('Native app mount is disposed');
      // Host-scoped transport (H-N): Fountain binds apiBase, session generation,
      // credentials and lifetime; Soda keeps only its disposed guard here and
      // all protocol/domain parsing in its callers.
      if (hostRequest) return hostRequest(relativePath, init);
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
      if (disposed) throw Error('Native app mount is disposed');
      // Host-scoped stream: Fountain rejects query/hash targets and tracks the
      // socket across update/suspend/resume, closing it only on retire.
      if (hostStream) return hostStream(relativePath).socket;
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

function validateRepository(repositoryId: string | undefined) {
  if (repositoryId !== undefined && !id(repositoryId)) throw Error('Invalid native repository');
}

function extensionURL(apiBase: URL, relativePath: string, allowQuery: boolean): URL {
  const url = new URL(relativePath, apiBase);
  if (!url.href.startsWith(apiBase.href) || url.hash || (!allowQuery && url.search) || /%2f|%5c/i.test(url.pathname)) {
    throw Error('Invalid Soda operation path');
  }
  return url;
}

export type PreparedExtensionMount = ReturnType<typeof prepareExtensionMount>;
