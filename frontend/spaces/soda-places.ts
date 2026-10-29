// The only builder of SodaOS addresses (SPEC M3). Every link to a SodaOS place
// comes from here, and the repository entry routes with placeFromPath, so a
// different address set (track 2: Fountain's named repository routes) is
// swapped in by changing only the function bodies in this file.
//
// Track 1 (pinned Fountain, today): every repository place is a subroute of the
// one `repository-feed` page under the host's fixed `workspace` segment.
//   /{owner}/{repo}/workspace                      Feed
//   /{owner}/{repo}/workspace/requests             Requests
//   /{owner}/{repo}/workspace/requests/{id|new}    Request / New request
//   /{owner}/{repo}/workspace/app?delivery=        App
//   /{owner}/{repo}/workspace/knowledge?item=      Knowledge
//   /{owner}/{repo}/workspace/terminals?run=       Terminals
//   /{owner}/{repo}/workspace/settings/{section}   SodaOS settings
//   /{owner}/{repo}/workspace/issues[/{issue}]     Factory status
// Track 2 (Fountain F1-F4 named repository routes, PLACES_TRACK === 2):
//   /{owner}/{repo}/feed                           Feed
//   /{owner}/{repo}/requests                       Requests
//   /{owner}/{repo}/requests/{id|new}              Request / New request
//   /{owner}/{repo}/app?delivery=                  App
//   /{owner}/{repo}/knowledge?item=                Knowledge
//   /{owner}/{repo}/terminals?run=                 Terminals
//   /{owner}/{repo}/settings/sodaos/{section}      SodaOS settings (own page)
//   /{owner}/{repo}/issues/{issue}/factory         Factory status, one issue
//   /{owner}/{repo}/workspace/issues               Factory status lookup
// On track 2 the hidden `workspace` route still serves every track-1 address;
// the entry forwards it in the browser (forwardedAddress), except the lookup,
// which has no named route.

declare const SODA_PLACES_TRACK: string | undefined;
/**
 * The address set, chosen when the package is built
 * (scripts/build-soda-extension.ts --track=2 defines SODA_PLACES_TRACK). The
 * manifest built with it (extension.json or extension.track2.json) must match.
 */
export const PLACES_TRACK: 1 | 2 =
  typeof SODA_PLACES_TRACK === 'string' && SODA_PLACES_TRACK === '2' ? 2 : 1;

/**
 * THE SWITCH (install B). While false, request and newRequest links point to
 * the global Workspace composer (/workspace/requests/{id}, /workspace/new), so
 * no link opens a place whose composer is not mounted yet (install A). Flip to
 * true in install B, when the repository entry hosts the composer.
 */
export const REQUESTS_IN_REPOSITORY = true;

/** Repository SodaOS settings sections, in place-bar order. */
export const settingsSections = ['environment', 'access', 'network', 'factory'] as const;
export type SettingsSection = (typeof settingsSections)[number];

/** The value of the request subroute that opens the composer. */
export const newRequestID = 'new';

/** One SodaOS place on a repository page, as parsed from or built into an address. */
export type Place =
  | {place: 'feed'; post?: string}
  | {place: 'requests'}
  | {place: 'request'; request: string}
  | {place: 'app'; delivery?: string}
  | {place: 'knowledge'; item?: string}
  | {place: 'terminals'; run?: string}
  | {place: 'settings'; section: string}
  | {place: 'factory-status'; issue?: string};
export type PlaceName = Place['place'];

/** Address builders for one repository's SodaOS places. */
export interface SodaPlaces {
  readonly slug: string;
  /** The Feed, or one post's card in it. */
  feed(post?: string): string;
  requests(): string;
  /** A saved request's page. Points at the global Workspace until REQUESTS_IN_REPOSITORY. */
  request(id: string): string;
  /** The composer. Points at the global Workspace until REQUESTS_IN_REPOSITORY. */
  newRequest(): string;
  app(delivery?: string): string;
  knowledge(item?: string): string;
  terminals(run?: string): string;
  settings(section: SettingsSection | string): string;
  factoryStatus(issue?: string | number): string;
  /** The address of a parsed place (the inverse of placeFromPath). */
  to(place: Place): string;
}

function query(name: string, value: string | undefined): string {
  return value ? `?${new URLSearchParams({[name]: value})}` : '';
}

function encodeSlug(slug: string): string {
  const parts = slug.split('/');
  if (parts.length !== 2 || parts.some((part) => part === '')) throw Error('Invalid repository slug');
  return parts.map(encodeURIComponent).join('/');
}

/** Address builders for the repository `owner/repo` under the installation prefix (appSubUrl). */
export function repositoryPlaces(prefix: string, slug: string): SodaPlaces {
  const repository = `${prefix}/${encodeSlug(slug)}`;
  const workspace = `${repository}/workspace`;
  const named = PLACES_TRACK === 2;
  const root = named ? repository : workspace;
  const places: SodaPlaces = {
    slug,
    feed: (post) => `${named ? `${repository}/feed` : root}${query('post', post)}`,
    requests: () => `${root}/requests`,
    request: (id) =>
      REQUESTS_IN_REPOSITORY ? `${root}/requests/${encodeURIComponent(id)}` : requestRouteFor(prefix, id),
    newRequest: () => (REQUESTS_IN_REPOSITORY ? `${root}/requests/${newRequestID}` : newRouteFor(prefix)),
    app: (delivery) => `${root}/app${query('delivery', delivery)}`,
    knowledge: (item) => `${root}/knowledge${query('item', item)}`,
    terminals: (run) => `${root}/terminals${query('run', run)}`,
    settings: (section) =>
      `${named ? `${repository}/settings/sodaos` : `${root}/settings`}/${encodeURIComponent(section)}`,
    // The lookup has no named route: it stays on the (hidden) workspace route.
    factoryStatus: (issue) =>
      issue === undefined
        ? `${workspace}/issues`
        : named
          ? `${repository}/issues/${encodeURIComponent(String(issue))}/factory`
          : `${root}/issues/${encodeURIComponent(String(issue))}`,
    to(place) {
      switch (place.place) {
        case 'feed':
          return places.feed(place.post);
        case 'requests':
          return places.requests();
        // A parsed request place is on this repository page already, so its
        // own address never goes through the Workspace switch.
        case 'request':
          return `${root}/requests/${encodeURIComponent(place.request)}`;
        case 'app':
          return places.app(place.delivery);
        case 'knowledge':
          return places.knowledge(place.item);
        case 'terminals':
          return places.terminals(place.run);
        case 'settings':
          return places.settings(place.section);
        case 'factory-status':
          return places.factoryStatus(place.issue);
      }
    },
  };
  return places;
}

/**
 * Parse a repository page address (pathname plus optional search) under the
 * installation prefix into its repository slug and place. Undefined when the
 * address is not a SodaOS repository place.
 */
export function placeFromPath(address: string, prefix: string): {slug: string; place: Place} | undefined {
  const url = new URL(address, 'http://soda.invalid');
  if (prefix !== '' && !(url.pathname === prefix || url.pathname.startsWith(`${prefix}/`))) return undefined;
  const segments = url.pathname.slice(prefix.length).split('/').filter(Boolean);
  let decoded: string[];
  try {
    decoded = segments.map(decodeURIComponent);
  } catch {
    return undefined;
  }
  const [owner, repo, fixed, ...path] = decoded;
  if (!owner || !repo) return undefined;
  const slug = `${owner}/${repo}`;
  const param = (name: string) => url.searchParams.get(name) || undefined;
  const place =
    fixed === 'workspace'
      ? workspacePlace(path, param)
      : PLACES_TRACK === 2 && fixed !== undefined
        ? namedPlace(fixed, path, param)
        : undefined;
  return place && {slug, place};
}

type Param = (name: string) => string | undefined;

function optional<K extends string>(key: K, value: string | undefined) {
  return (value === undefined ? {} : {[key]: value}) as {[P in K]?: string};
}

/** A place under the `workspace` segment (track 1; the hidden route on track 2). */
function workspacePlace([first, second, ...rest]: string[], param: Param): Place | undefined {
  if (rest.length) return undefined;
  if (first === undefined) return {place: 'feed', ...optional('post', param('post'))};
  if (first === 'requests') return second === undefined ? {place: 'requests'} : {place: 'request', request: second};
  if (second !== undefined) {
    if (first === 'settings') return {place: 'settings', section: second};
    if (first === 'issues') return {place: 'factory-status', issue: second};
    return undefined;
  }
  if (first === 'app') return {place: 'app', ...optional('delivery', param('delivery'))};
  if (first === 'knowledge') return {place: 'knowledge', ...optional('item', param('item'))};
  if (first === 'terminals') return {place: 'terminals', ...optional('run', param('run'))};
  if (first === 'issues') return {place: 'factory-status'};
  return undefined;
}

/** A place on one of Fountain's named repository routes (track 2). */
function namedPlace(route: string, path: string[], param: Param): Place | undefined {
  const [first, second, ...rest] = path;
  switch (route) {
    case 'feed':
      return path.length ? undefined : {place: 'feed', ...optional('post', param('post'))};
    case 'requests':
      if (second !== undefined) return undefined;
      return first === undefined ? {place: 'requests'} : {place: 'request', request: first};
    case 'app':
      return path.length ? undefined : {place: 'app', ...optional('delivery', param('delivery'))};
    case 'knowledge':
      return path.length ? undefined : {place: 'knowledge', ...optional('item', param('item'))};
    case 'terminals':
      return path.length ? undefined : {place: 'terminals', ...optional('run', param('run'))};
    case 'settings':
      if (first !== 'sodaos' || rest.length) return undefined;
      return {place: 'settings', section: second ?? settingsSections[0]};
    case 'issues':
      return first !== undefined && second === 'factory' && !rest.length
        ? {place: 'factory-status', issue: first}
        : undefined;
  }
  return undefined;
}

/**
 * Track 2: the named address an old /{owner}/{repo}/workspace/… address
 * forwards to, keeping its query and hash. Undefined on track 1, for any other
 * address, and for the Factory status lookup, which has no named route.
 */
export function forwardedAddress(address: string, prefix: string): string | undefined {
  if (PLACES_TRACK !== 2) return undefined;
  const parsed = placeFromPath(address, prefix);
  if (!parsed) return undefined;
  const url = new URL(address, 'http://soda.invalid');
  const fixed = url.pathname.slice(prefix.length).split('/').filter(Boolean)[2];
  if (fixed !== 'workspace') return undefined;
  const target = new URL(repositoryPlaces(prefix, parsed.slug).to(parsed.place), url);
  if (target.pathname === url.pathname) return undefined;
  for (const [name, value] of url.searchParams) {
    if (!target.searchParams.has(name)) target.searchParams.append(name, value);
  }
  return `${target.pathname}${target.search}${url.hash}`;
}

// The global Workspace page's own addresses. They live here so the Workspace
// keeps routing itself while it exists, and go away with it in install E.

/** Canonical global Workspace request route under the installation prefix. */
export function requestRouteFor(prefix: string, intent: string): string {
  return `${prefix}/workspace/requests/${intent}`;
}

/** Global Workspace fresh-entry route under the installation prefix. */
export function newRouteFor(prefix: string): string {
  return `${prefix}/workspace/new`;
}
