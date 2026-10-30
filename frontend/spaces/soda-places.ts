// Canonical SodaOS repository routes. Links and parsing share this owner.

/** Repository SodaOS settings sections, in place-bar order. */
export const settingsSections = ['environment', 'access', 'network', 'factory'] as const;
export type SettingsSection = (typeof settingsSections)[number];

/** One SodaOS place on a repository page, as parsed from or built into an address. */
export type Place =
  | {place: 'feed'; post?: string; compose?: true}
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
  /** A saved request within its repository feed. */
  request(id: string): string;
  /** The Feed with its composer open: a post is how a request starts. */
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
  const root = `${repository}/feed`;
  const places: SodaPlaces = {
    slug,
    feed: (post) => `${repository}/feed${query('post', post)}`,
    request: (id) => `${root}/requests/${encodeURIComponent(id)}`,
    newRequest: () => `${places.feed()}?compose=1`,
    app: (delivery) => `${root}/app${query('delivery', delivery)}`,
    knowledge: (item) => `${root}/knowledge${query('item', item)}`,
    terminals: (run) => `${root}/terminals${query('run', run)}`,
    settings: (section) =>
      `${root}/settings/${encodeURIComponent(section)}`,
    factoryStatus: (issue) => `${root}/issues${issue === undefined ? '' : '/' + encodeURIComponent(String(issue))}`,
    to(place) {
      switch (place.place) {
        case 'feed':
          return place.compose ? places.newRequest() : places.feed(place.post);
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
  const place = fixed === 'feed' ? feedRoutePlace(path, param) : undefined;
  return place && {slug, place};
}

type Param = (name: string) => string | undefined;

function optional<K extends string>(key: K, value: string | undefined) {
  return (value === undefined ? {} : {[key]: value}) as {[P in K]?: string};
}

function feedPlace(param: Param): Place {
  return param('compose') ? {place: 'feed', compose: true} : {place: 'feed', ...optional('post', param('post'))};
}

/** A place under the repository feed route. */
function feedRoutePlace([first, second, ...rest]: string[], param: Param): Place | undefined {
  if (rest.length) return undefined;
  if (first === undefined) return feedPlace(param);
  if (first === 'requests') return second && /^d[0-9a-f]{24}$/.test(second) ? {place: 'request', request: second} : undefined;
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
