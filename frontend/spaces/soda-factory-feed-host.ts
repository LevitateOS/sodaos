import type {FactoryFeedContext} from '@factory/product/frontend/factory/feed/feed.js';
import type {FactoryRepositoryCatalog} from '@factory/product/frontend/factory/app/navigation.js';
import type {FactoryTransport} from '@factory/product/frontend/factory/api/transport.js';
import {check, id, object, readSodaJSON} from '@factory/product/frontend/factory/api/models.js';
import {nativePath} from '@factory/product/frontend/factory/shared/format.js';
import {repositoryChoices} from './sodaspaces-repository-response.js';
import {applicationsResponse} from './sodaspaces-application-response.js';
import {repositoryPlaces} from './soda-places.js';
import {fountainIssueLink, fountainPullLink} from './soda-factory-links.js';

interface FountainFeedOptions {
  /** Native Fountain repository locator; resolved before a Factory context exists. */
  repository: string;
  repositoryName?: string;
  repositories?: ReadonlyMap<string, string>;
  account?: string;
  global?: boolean;
  prefix?: string;
}

interface FountainFactoryIdentity {
  host: string;
  actor: string;
}

export interface FountainAccountIdentity extends FountainFactoryIdentity {
  nativeActor: string;
}

function identity(value: Record<string, unknown>): FountainFactoryIdentity {
  check(typeof value.host === 'string' && /^[a-f0-9]{32}$/.test(value.host) && id(value.actor));
  return {host: value.host as string, actor: value.actor as string};
}

async function requestJSON(transport: FactoryTransport, path: string, signal?: AbortSignal, body?: object) {
  signal?.throwIfAborted();
  const response = await transport.request(path, {
    method: body ? 'POST' : 'GET',
    signal: signal ? AbortSignal.any([signal, AbortSignal.timeout(15000)]) : AbortSignal.timeout(15000),
    ...(body ? {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)} : {}),
  });
  if (!response.ok) throw Error('Repository information is unavailable.');
  const value = await readSodaJSON(response);
  signal?.throwIfAborted();
  return value;
}

export async function fountainIdentity(
  transport: FactoryTransport,
  signal?: AbortSignal
): Promise<FountainAccountIdentity> {
  const value = object(await requestJSON(transport, 'factory/identity', signal));
  const scope = identity(value);
  check(id(value.native_actor));
  return {...scope, nativeActor: value.native_actor as string};
}

/** Stable authenticated command custody; transport generations are deliberately excluded. */
export async function fountainCommandScope(transport: FactoryTransport, signal: AbortSignal): Promise<string> {
  const account = await fountainIdentity(transport, signal);
  return `fountain:${account.host}:${account.actor}`;
}

/** Resolve only the native locator supplied by the Fountain host, with current visibility. */
export async function fountainRepository(transport: FactoryTransport, native: string, signal?: AbortSignal) {
  check(id(native));
  const value = object(
    await requestJSON(transport, 'factory/native-repositories/resolve', signal, {repository: native})
  );
  const scope = identity(value);
  check(id(value.repository) && value.native_repository === native && typeof value.name === 'string');
  check(/^[^/]+\/[^/]+$/.test(value.name as string) && nativePath('/' + value.name));
  return {...scope, repository: value.repository as string, name: value.name as string};
}

/** Canonical Factory IDs never enter native inventory APIs to discover a URL. */
export async function fountainRepositoryIdentity(
  transport: FactoryTransport,
  repository: string,
  signal?: AbortSignal
) {
  check(id(repository));
  const value = object(await requestJSON(transport, `factory/repositories/${repository}/identity`, signal));
  return repositoryIdentity(value, repository);
}

function repositoryIdentity(value: Record<string, unknown>, repository: string) {
  const scope = identity(value);
  check(value.repository === repository && id(value.native_repository) && typeof value.name === 'string');
  check(/^[^/]+\/[^/]+$/.test(value.name as string) && nativePath('/' + value.name));
  return {...scope, repository, native: value.native_repository as string, name: value.name as string};
}

/** Current configured branch for native navigation, not ref existence or authority. */
export async function fountainRepositoryNavigation(
  transport: FactoryTransport,
  repository: string,
  signal?: AbortSignal
) {
  check(id(repository));
  const generation = transport.generation;
  const value = object(await requestJSON(transport, `factory/repositories/${repository}/identity`, signal));
  check(generation === transport.generation);
  const current = repositoryIdentity(value, repository);
  check(
    typeof value.default_branch === 'string' &&
      new TextEncoder().encode(value.default_branch).length <= 1024 &&
      !/[\u0000-\u001f\u007f]/.test(value.default_branch)
  );
  return {...current, defaultBranch: value.default_branch};
}

/** Fountain owns discovery and native URLs; the resulting feed contains local IDs only. */
export async function fountainFeedContext(
  transport: FactoryTransport & {navigate?: (path: string) => void},
  options: FountainFeedOptions,
  signal?: AbortSignal
): Promise<FactoryFeedContext> {
  const scope = identity(object(await requestJSON(transport, 'factory/identity', signal)));
  const resolve = async (native: string, requestSignal?: AbortSignal) => {
    const ref = await fountainRepository(transport, native, requestSignal);
    check(ref.host === scope.host && ref.actor === scope.actor);
    return ref;
  };
  const current = options.repository ? await resolve(options.repository, signal) : undefined;
  signal?.throwIfAborted();
  return {
    transport,
    repository: current?.repository ?? '',
    ...(current ? {repositoryName: current.name} : {}),
    account: scope.actor,
    ...(options.global !== undefined ? {global: options.global} : {}),
    host: {
      draftScope: `fountain:${scope.host}:${scope.actor}`,
      async repositories(repositories, requestSignal): Promise<FactoryRepositoryCatalog> {
        const refs: FactoryRepositoryCatalog['refs'] = new Map();
        const wanted = new Set(repositories);
        const native = new Set(options.repositories?.keys() ?? []);
        if (options.repository) native.add(options.repository);
        let listedAll = true;
        const add = (ref: {repository: string; name: string}) => {
          refs.set(ref.repository, {path: '/' + ref.name, label: ref.name});
          wanted.delete(ref.repository);
        };
        try {
          if (options.global) {
            let after = '';
            for (;;) {
              const page = object(
                await requestJSON(
                  transport,
                  'factory/repositories' + (after ? '?after=' + encodeURIComponent(after) : ''),
                  requestSignal
                )
              );
              const pageScope = identity(page);
              check(pageScope.host === scope.host && pageScope.actor === scope.actor);
              check(Array.isArray(page.items) && page.items.length <= 100 && typeof page.complete === 'boolean');
              const next = page.next_after;
              check(typeof next === 'string' && (next === '' || id(next)));
              check(next === '' || (!page.complete && BigInt(next) > BigInt(after || '0')));
              // Validate the entire page before exposing any label or link.
              let previous = after || '0';
              const entries = (page.items as unknown[]).map((value) => {
                const item = object(value);
                check(id(item.repository) && typeof item.name === 'string');
                check(/^[^/]+\/[^/]+$/.test(item.name) && nativePath('/' + item.name));
                check(BigInt(item.repository as string) > BigInt(previous));
                previous = item.repository as string;
                return {repository: previous, name: item.name};
              });
              check(next === '' || BigInt(next) >= BigInt(previous));
              for (const entry of entries) add(entry);
              if (!next) {
                listedAll = page.complete;
                break;
              }
              after = next;
            }
          } else {
            const choices = repositoryChoices(await requestJSON(transport, 'repositories?q=', requestSignal), 1);
            for (const item of choices.items) native.add(item.id);
            listedAll = choices.nextCursor === '';
          }
        } catch {
          requestSignal.throwIfAborted();
          listedAll = false;
        }
        // Refresh mutable labels even for the initially selected repository.
        const pending = [...native];
        let next = 0;
        await Promise.all(
          Array.from({length: Math.min(4, pending.length)}, async () => {
            while (next < pending.length) {
              requestSignal.throwIfAborted();
              const value = pending[next++];
              if (!value) continue;
              try {
                add(await resolve(value, requestSignal));
              } catch {
                requestSignal.throwIfAborted();
                listedAll = false;
              }
            }
          })
        );
        requestSignal.throwIfAborted();
        for (const missing of wanted)
          if (!refs.has(missing)) refs.set(missing, {path: undefined, label: `repository ${missing}`});
        return {refs, listedAll};
      },
      links(_repository, label) {
        if (!/^[^/]+\/[^/]+$/.test(label) || !nativePath('/' + label)) return undefined;
        const prefix = options.prefix ?? '';
        const places = repositoryPlaces(prefix, label);
        const repository = `${prefix}/${label}`;
        return {
          repository,
          feed: places.feed,
          request: places.request,
          factoryStatus: places.factoryStatus,
          app: places.app,
          knowledge: places.knowledge,
          settings: places.settings,
          issue: (issue) => fountainIssueLink(repository, String(issue)),
          pull: (pull) => fountainPullLink(repository, String(pull)),
        };
      },
      async applicationOrigins(signal) {
        const inventory = applicationsResponse(await requestJSON(transport, 'applications', signal));
        const origins = new Map<string, string>();
        for (const launch of inventory.launchable)
          if (launch.ready && launch.delivery_id && launch.application.origin)
            origins.set(launch.delivery_id, launch.application.origin);
        for (const record of inventory.applications)
          if (record.observation?.ready && record.delivery?.delivery_id && record.application.origin)
            origins.set(record.delivery.delivery_id, record.application.origin);
        return origins;
      },
      navigate: (path) => (transport.navigate ? transport.navigate(path) : location.assign(path)),
    },
  };
}
