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
// Track 2 (after Fountain F1-F4): /{owner}/{repo}/requests…, /feed, /app,
// /terminals, /knowledge, /issues/{n}/factory, /settings/sodaos/{section}.

/**
 * THE SWITCH (install B). While false, request and newRequest links point to
 * the global Workspace composer (/workspace/requests/{id}, /workspace/new), so
 * no link opens a place whose composer is not mounted yet (install A). Flip to
 * true in install B, when the repository entry hosts the composer.
 */
export const REQUESTS_IN_REPOSITORY = false;

/** Repository SodaOS settings sections, in place-bar order. */
export const settingsSections = ['environment', 'access', 'network', 'factory'] as const;
export type SettingsSection = (typeof settingsSections)[number];

/** The value of the request subroute that opens the composer. */
export const newRequestID = 'new';

/** One SodaOS place on a repository page, as parsed from or built into an address. */
export type Place =
  | {place: 'feed'}
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
  feed(): string;
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
  const root = `${repository}/workspace`;
  const places: SodaPlaces = {
    slug,
    feed: () => root,
    requests: () => `${root}/requests`,
    request: (id) =>
      REQUESTS_IN_REPOSITORY ? `${root}/requests/${encodeURIComponent(id)}` : requestRouteFor(prefix, id),
    newRequest: () => (REQUESTS_IN_REPOSITORY ? `${root}/requests/${newRequestID}` : newRouteFor(prefix)),
    app: (delivery) => `${root}/app${query('delivery', delivery)}`,
    knowledge: (item) => `${root}/knowledge${query('item', item)}`,
    terminals: (run) => `${root}/terminals${query('run', run)}`,
    settings: (section) => `${root}/settings/${encodeURIComponent(section)}`,
    factoryStatus: (issue) =>
      issue === undefined ? `${root}/issues` : `${root}/issues/${encodeURIComponent(String(issue))}`,
    to(place) {
      switch (place.place) {
        case 'feed':
          return places.feed();
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
  const [owner, repo, fixed, first, second, ...rest] = decoded;
  if (!owner || !repo || fixed !== 'workspace' || rest.length) return undefined;
  const slug = `${owner}/${repo}`;
  const param = (name: string) => url.searchParams.get(name) || undefined;
  const optional = <K extends string>(key: K, value: string | undefined) =>
    (value === undefined ? {} : {[key]: value}) as {[P in K]?: string};
  let place: Place | undefined;
  if (first === undefined) place = {place: 'feed'};
  else if (first === 'requests')
    place = second === undefined ? {place: 'requests'} : {place: 'request', request: second};
  else if (second !== undefined) {
    if (first === 'settings') place = {place: 'settings', section: second};
    else if (first === 'issues') place = {place: 'factory-status', issue: second};
  } else if (first === 'app') place = {place: 'app', ...optional('delivery', param('delivery'))};
  else if (first === 'knowledge') place = {place: 'knowledge', ...optional('item', param('item'))};
  else if (first === 'terminals') place = {place: 'terminals', ...optional('run', param('run'))};
  else if (first === 'issues') place = {place: 'factory-status'};
  return place && {slug, place};
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
