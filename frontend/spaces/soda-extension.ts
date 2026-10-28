import {nativeAppSubUrl} from './soda-native-paths.js';
import {id} from './sodaspaces-api.js';
import type {MountContext, ResumeValidation} from './fountain-browser-contract.js';

/**
 * Host lifecycle resumes only under the admitted session generation
 * (NFX-D05/C05). A mismatched resume validation is a retirement signal:
 * the caller must release its resources instead of resuming stale state.
 * The host normally retires on mismatch without calling resume; this
 * check is the consumer-side refusal for a directly supplied validation.
 */
export function resumeGenerationRefused(generation: string, validation?: ResumeValidation): boolean {
  return validation !== undefined && validation.sessionGeneration !== generation;
}

// Soda consumer mount context is the canonical Fountain MountContext
// (NFX-D01/H-N). The host supplies scoped request/stream/navigate helpers;
// Soda keeps only its product/domain validation and disposed guard, and
// never re-implements generic URL/session transport rules.
export type ExtensionMountContext = MountContext;

function isPageMount(context: ExtensionMountContext): boolean {
  return ['spaces', 'feed', 'repository-feed', 'applications'].includes(context.pageId ?? '') && !context.panelId;
}

function isPanelMount(context: ExtensionMountContext): boolean {
  return context.panelId === 'workspace' && !context.pageId;
}

function validateMountContext(context: ExtensionMountContext): void {
  // Refuse an unexpected canonical contract version before any effects.
  // The host dataset already refuses mismatches; this binds the consumer
  // build to a compatible host runtime.
  if (context.contractVersion !== 1) throw Error('Unsupported native contract version');
  if (!context.sessionGeneration) throw Error('This SodaOS page could not load');
  if (context.extensionId === 'tailnet') {
    if (
      context.pageId !== 'tailnet' ||
      context.panelId !== undefined ||
      context.ownerId !== undefined ||
      context.repositoryId !== undefined
    ) {
      throw Error('Invalid Tailnet mount context');
    }
    nativeAppSubUrl(context);
    return;
  }
  if (context.extensionId !== 'soda') throw Error('This SodaOS page could not load');
  if (!isPageMount(context) && !isPanelMount(context)) throw Error('This SodaOS page could not load');
  validateOwner(context);
  nativeAppSubUrl(context);
  validateRepository(context.repositoryId);
}

function validateOwner(context: ExtensionMountContext): void {
  // Shared-owner identity (NFX-C04/H-OWNER): the pair (extensionId, ownerId)
  // names one retained owner. Soda declares exactly one, "workspace",
  // linking the spaces page and the workspace panel on one entry.
  if (context.ownerId === undefined) return;
  if (context.ownerId !== 'workspace') throw Error('This SodaOS page could not load');
  if (context.pageId !== 'spaces' && context.panelId !== 'workspace') throw Error('This SodaOS page could not load');
}

function requireHostTransport(context: ExtensionMountContext) {
  // Host-scoped transport is required (H-N): Fountain binds apiBase,
  // session generation, credentials and lifetime, and tracks sockets
  // across update/suspend/resume, closing them only on retire. There is
  // no local URL/session fallback; generic scoping stays with the host.
  if (typeof context.request !== 'function') throw Error('Native host transport unavailable');
  if (typeof context.stream !== 'function') throw Error('Native host transport unavailable');
  if (typeof context.navigate !== 'function') throw Error('Native host transport unavailable');
  return {hostRequest: context.request, hostStream: context.stream, hostNavigate: context.navigate};
}

// Prepares the native request transport and lifetime for mounted pages.
export function prepareExtensionMount(root: HTMLElement, context: ExtensionMountContext): PreparedExtensionMount {
  validateMountContext(context);
  const {hostRequest, hostStream, hostNavigate} = requireHostTransport(context);
  const generation = context.sessionGeneration;
  let disposed = false;
  const assertLive = () => {
    if (disposed) throw Error('Native app mount is disposed');
  };
  return {
    root,
    generation,
    ownerId: context.ownerId,
    assetBase: context.assetBase,
    async request(relativePath: string, init: RequestInit = {}): Promise<Response> {
      assertLive();
      return hostRequest(relativePath, init);
    },
    websocket(relativePath: string): WebSocket {
      assertLive();
      return hostStream(relativePath).socket;
    },
    navigate(path: string): void {
      assertLive();
      hostNavigate(path);
    },
    dispose() {
      disposed = true;
    },
  };
}

function validateRepository(repositoryId: string | undefined) {
  if (repositoryId !== undefined && !id(repositoryId)) throw Error('Invalid native repository');
}

export interface PreparedExtensionMount {
  root: HTMLElement;
  generation: string;
  ownerId?: string | undefined;
  assetBase: string;
  request(relativePath: string, init?: RequestInit): Promise<Response>;
  websocket(relativePath: string): WebSocket;
  navigate?: (path: string) => void;
  dispose(): void;
}
