// Fountain's identity and navigation join for the shared issue views. Product
// components consume these ports without importing the native mount contract.
import type {FactoryTransport} from '@factory/product/frontend/factory/api/transport.js';
import type {IssueListContext} from '@factory/product/frontend/factory/issues/issues.js';
import type {IssueDetailContext} from '@factory/product/frontend/factory/issues/detail.js';
import {check, id} from '@factory/product/frontend/factory/api/models.js';
import {markdownLink} from '@factory/product/frontend/factory/shared/markdown.js';
import {nativeAppSubUrl} from './soda-native-paths.js';
import {fountainRepositoryNavigation} from './soda-factory-feed-host.js';

export interface FountainIssueHostOptions {
  /** Captured, authenticated place binding, not a repository name used as ID. */
  readonly host: string;
  readonly actor: string;
  readonly repository: string;
  readonly nativeRepository: string;
  readonly name: string;
  readonly prefix: string;
  readonly origin: string;
  /** The current host router owns the destination and fragment/history behavior. */
  issuePath(number: string): string;
  navigate(path: string, signal: AbortSignal): void | Promise<void>;
  openExternal(url: string, signal: AbortSignal): void | Promise<void>;
}

/** Recheck live native navigation against the captured canonical place before
 * supplying any read keys. The signal belongs to that place's whole lifetime. */
export async function fountainIssueContexts(
  transport: FactoryTransport,
  options: FountainIssueHostOptions,
  signal: AbortSignal
): Promise<{
  list: IssueListContext;
  detail(number: string): IssueDetailContext;
}> {
  signal.throwIfAborted();
  const generation = transport.generation;
  check(!!generation && /^[a-f0-9]{32}$/.test(options.host));
  check(id(options.actor) && id(options.repository) && id(options.nativeRepository));
  check(typeof options.navigate === 'function' && typeof options.openExternal === 'function');
  check(typeof options.issuePath === 'function');
  // Copy the binding and ports; mutating a caller's object cannot rebind an
  // admitted context while its reads or navigation are outstanding.
  const {host, actor, repository, nativeRepository, name, prefix, origin, issuePath, navigate, openExternal} = options;
  const base = new URL(origin);
  check(['http:', 'https:'].includes(base.protocol) && base.origin === origin);
  nativeAppSubUrl({appSubUrl: prefix});
  const current = await fountainRepositoryNavigation(transport, repository, signal);
  signal.throwIfAborted();
  check(
    current.host === host &&
      current.actor === actor &&
      current.native === nativeRepository &&
      current.name === name &&
      transport.generation === generation
  );
  const repositoryPath = `${prefix}/${current.name.split('/').map(encodeURIComponent).join('/')}`;
  const repositoryURL = new URL(repositoryPath + '/', base);
  const scope = `fountain:${host}:${actor}`;
  const live = () => {
    signal.throwIfAborted();
    check(transport.generation === generation);
  };
  const issueURL = (number: string) => {
    live();
    check(id(number));
    const path = issuePath(number);
    check(typeof path === 'string' && path.startsWith(repositoryPath + '/') && !/[\\\u0000-\u0020\u007f]/.test(path));
    const url = new URL(path, base);
    check(url.origin === origin && !url.username && !url.password && !url.search && !url.hash);
    check(url.pathname.startsWith(repositoryPath + '/'));
    return url;
  };
  const open = async (url: string, opening: AbortSignal) => {
    const lifetime = AbortSignal.any([signal, opening]);
    lifetime.throwIfAborted();
    live();
    const safe = markdownLink(url);
    check(safe !== undefined);
    const target = new URL(safe);
    // The native mount's navigate port admits same-origin paths only. External
    // web/mail destinations have a separate browser/desktop opener.
    if (target.origin === origin) await navigate(target.pathname + target.search + target.hash, lifetime);
    else await openExternal(target.href, lifetime);
    lifetime.throwIfAborted();
    live();
  };
  // Both a component's own read lifetime and the host place must still be live.
  // Consumers composing a frame/journey use this same transport for every child.
  const scopedTransport: FactoryTransport = {
    get generation() {
      return transport.generation;
    },
    get assetBase() {
      return transport.assetBase;
    },
    async request(path, init = {}) {
      live();
      const response = await transport.request(path, {
        ...init,
        signal: init.signal ? AbortSignal.any([signal, init.signal]) : signal,
      });
      live();
      return response;
    },
  };
  const shared = {transport: scopedTransport, host, actor, repository, scope};
  return {
    list: {
      ...shared,
      async openIssue(ref, opening) {
        opening.throwIfAborted();
        live();
        check(ref.kind === 'issue' && ref.repository === repository && id(ref.number));
        check(typeof ref.external_id === 'string' && !!ref.external_id.trim());
        await open(issueURL(ref.number).href, opening);
      },
    },
    detail(number) {
      const page = issueURL(number);
      return {
        ...shared,
        issue: number,
        resolveLink(href) {
          live();
          if (!href || /[\\\u0000-\u001f\u007f]/.test(href)) return;
          // Match native issue/comment Markdown's repository-relative base
          // (modules/markup/markdown/transform_link.go), not a guessed branch.
          // Explicit src/branch/... links retain their exact recorded branch.
          const target = new URL(href, href.startsWith('#') || href.startsWith('?') ? page : repositoryURL);
          return markdownLink(target.href);
        },
        openLink: open,
      };
    },
  };
}
