// The SodaOS repository page (SPEC M2): one host page, `repository-feed`, with
// every repository place as a subroute. The host re-adopts this root on every
// move between places and calls update({route}); the entry swaps the view
// itself. Views that must survive a place switch (a request's composer, the
// terminals) stay mounted here, hidden, until this entry is disposed or
// retired. Each place's view lives in its own soda-place-*.ts file.
//
// On track 2 (Fountain named routes) the same entry also mounts the separate
// `repository-settings` page at /{owner}/{repo}/settings/sodaos/{section}:
// settings only, with no place bar, because the native settings layout is
// around it. Old /workspace/… addresses are forwarded in the browser.
import {html, render} from 'lit';
import type {TemplateResult} from 'lit';
import type {ContributionPresentation, MountHandle, ResumeValidation} from './fountain-browser-contract.js';
import type {ExtensionMountContext} from './soda-extension.js';
import {prepareExtensionMount} from './soda-extension.js';
import {nativeAppSubUrl} from './soda-native-paths.js';
import {forwardedAddress, newRequestID, placeFromPath, repositoryPlaces} from './soda-places.js';
import type {Place, PlaceName, SodaPlaces} from './soda-places.js';
import {noticePlace} from './soda-place-view.js';
import type {PlaceContext, PlaceMount, PlaceView, PlaceViewer} from './soda-place-view.js';
import {mountFeedPlace} from './soda-place-feed.js';
import {mountRequestsPlace} from './soda-place-requests.js';
import {mountRequestPlace} from './soda-place-request.js';
import {mountAppPlace} from './soda-place-app.js';
import {mountKnowledgePlace} from './soda-place-knowledge.js';
import {mountTerminalsPlace} from './soda-place-terminals.js';
import {mountSettingsPlace} from './soda-place-settings.js';
import {mountFactoryStatusPlace} from './soda-place-factory-status.js';
import './soda-providers.js';

interface PlaceDefinition {
  title: string;
  mount: PlaceMount;
  /** Retained views get one live instance per key, kept hidden while another place shows. */
  retainKey?: (place: Place) => string;
  /** A refusal instead of the view, when the viewer would get a 403 there. */
  refusal?: (viewer: PlaceViewer, place: Place) => string | undefined;
}

const notAdmin = 'Only repository administrators manage SodaOS settings.';
const placeDefinitions: Record<PlaceName, PlaceDefinition> = {
  feed: {title: 'Feed', mount: mountFeedPlace},
  requests: {title: 'Requests', mount: mountRequestsPlace},
  request: {
    title: 'Request',
    mount: mountRequestPlace,
    retainKey: (place) => `request:${place.place === 'request' ? place.request : ''}`,
    refusal: (viewer, place) =>
      place.place === 'request' && place.request === newRequestID && !viewer.admin
        ? 'Only repository administrators can start Factory work here. Follow the work in Feed.'
        : undefined,
  },
  app: {title: 'App', mount: mountAppPlace},
  knowledge: {title: 'Knowledge', mount: mountKnowledgePlace},
  terminals: {
    title: 'Terminals',
    mount: mountTerminalsPlace,
    retainKey: () => 'terminals',
    refusal: (viewer) => (viewer.codeWrite ? undefined : 'Terminals need write access to this repository.'),
  },
  settings: {
    title: 'SodaOS settings',
    mount: mountSettingsPlace,
    refusal: (viewer) => (viewer.admin || viewer.operator ? undefined : notAdmin),
  },
  'factory-status': {title: 'Factory status', mount: mountFactoryStatusPlace},
};

/** At most this many request composers stay alive; the oldest hidden one goes first. */
const retainedRequestLimit = 4;

/**
 * Soda's header tabs (M21, in the Forgejo image) mark each tab with
 * data-soda-place. When they are there, the page draws no place bar.
 */
function headerHasSodaTabs(root: HTMLElement): boolean {
  return [...document.querySelectorAll('[data-soda-place]')].some((tab) => !root.contains(tab));
}

/** Read the viewer's standing once from the repository's Factory access view. */
async function readViewer(context: PlaceContext): Promise<PlaceViewer> {
  const response = await context.transport.request(
    `repositories/${encodeURIComponent(context.repository)}/factory/access`,
    {method: 'GET'}
  );
  if (!response.ok) throw Error('Viewer permission unavailable');
  const value = (await response.json()) as {viewer?: Record<string, unknown>};
  const viewer = value.viewer ?? {};
  return {
    loaded: true,
    codeWrite: viewer.code_write === true,
    admin: viewer.owner_or_admin === true,
    operator: viewer.operator === true,
  };
}

interface BarItem {
  label: string;
  href: string;
  active: boolean;
}

function placeBarItems(places: SodaPlaces, prefix: string, viewer: PlaceViewer, place: Place | undefined): BarItem[] {
  const name = place?.place;
  const repository = `${prefix}/${places.slug.split('/').map(encodeURIComponent).join('/')}`;
  const items: (BarItem | false)[] = [
    {label: 'Code', href: repository, active: false},
    {label: 'Requests', href: places.requests(), active: name === 'requests' || name === 'request'},
    {label: 'Feed', href: places.feed(), active: name === 'feed'},
    {label: 'App', href: places.app(), active: name === 'app'},
    {label: 'Issues', href: places.factoryStatus(), active: name === 'factory-status'},
    {label: 'Pull requests', href: `${repository}/pulls`, active: false},
    viewer.codeWrite && {label: 'Terminals', href: places.terminals(), active: name === 'terminals'},
    {label: 'Knowledge', href: places.knowledge(), active: name === 'knowledge'},
    (viewer.admin || viewer.operator) && {
      label: 'SodaOS settings',
      href: places.settings('environment'),
      active: name === 'settings',
    },
  ];
  return items.filter((item): item is BarItem => item !== false);
}

function placeBar(items: BarItem[]): TemplateResult {
  return html`<nav class="soda-place-bar" aria-label="SodaOS places">
    ${items.map(
      (item) =>
        html`<a class=${item.active ? 'active' : ''} href=${item.href} aria-current=${item.active ? 'page' : 'false'}
          >${item.label}</a
        >`
    )}
  </nav>`;
}

interface LiveView {
  key: string;
  host: HTMLElement;
  view: PlaceView;
  retained: boolean;
}

function slugFromPath(pathname: string, prefix: string): string {
  return pathname.slice(prefix.length).split('/').filter(Boolean).slice(0, 2).map(decodeURIComponent).join('/');
}

export function mount(root: HTMLElement, context: ExtensionMountContext): MountHandle {
  const settingsOnly = context.pageId === 'repository-settings';
  if ((context.pageId !== 'repository-feed' && !settingsOnly) || context.panelId || !context.repositoryId)
    throw Error('Invalid SodaOS repository mount');
  const transport = prepareExtensionMount(root, context);
  const styles = ['components.css', 'sodaspaces.css', 'sodaspaces-page.css', 'sodaspaces-drawer.css'].map((name) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = transport.assetBase + name;
    return link;
  });
  root.append(...styles);
  const prefix = nativeAppSubUrl(context);
  const slug = placeFromPath(location.pathname, prefix)?.slug ?? slugFromPath(location.pathname, prefix);
  const places = repositoryPlaces(prefix, slug);
  const viewer: PlaceViewer = {loaded: false, codeWrite: false, admin: false, operator: false};
  const placeContext: PlaceContext = {
    transport,
    prefix,
    repository: context.repositoryId,
    slug,
    places,
    viewer,
    navigate: (path, options) => context.navigate(path, options),
  };

  const page = document.createElement('div');
  page.className = 'soda-feed-page soda-repository-page';
  const bar = document.createElement('div');
  const providers = document.createElement('div');
  const viewsLayer = document.createElement('div');
  viewsLayer.className = 'soda-place-views';
  page.append(bar, providers, viewsLayer);
  root.append(page);

  let place: Place | undefined;
  let visible = true;
  let retired = false;
  let current: LiveView | undefined;
  const retained = new Map<string, LiveView>();

  const drawChrome = () => {
    render(
      settingsOnly || headerHasSodaTabs(root) ? html`` : placeBar(placeBarItems(places, prefix, viewer, place)),
      bar
    );
    render(
      html`<soda-providers .externalTrigger=${true} .transport=${visible ? transport : undefined}></soda-providers>`,
      providers
    );
  };

  // An error in one view stays in that view.
  const guarded = (host: HTMLElement, title: string, action: () => PlaceView): PlaceView => {
    try {
      return action();
    } catch (error) {
      console.error(error);
      host.replaceChildren();
      return noticePlace(host, title, 'This SodaOS place could not load.');
    }
  };

  const release = (live: LiveView) => {
    try {
      live.view.dispose();
    } catch (error) {
      console.error(error);
    }
    live.host.remove();
    if (live.retained) retained.delete(live.key);
  };

  const hide = (live: LiveView) => {
    if (!live.retained) {
      release(live);
      return;
    }
    live.host.style.display = 'none';
    live.view.setActive?.(false);
  };

  const trimRequests = () => {
    const requests = [...retained.values()].filter((live) => live.key.startsWith('request:') && live !== current);
    while (requests.length >= retainedRequestLimit) release(requests.shift() as LiveView);
  };

  const showPlace = (next: Place | undefined) => {
    place = next;
    drawChrome();
    if (!next) {
      if (current) hide(current);
      const host = document.createElement('div');
      viewsLayer.append(host);
      current = {
        key: 'unknown',
        host,
        view: noticePlace(host, 'SodaOS', 'This SodaOS page could not load.'),
        retained: false,
      };
      return;
    }
    const definition = placeDefinitions[next.place];
    const refusal = definition.refusal?.(viewer, next);
    const pending = refusal !== undefined && !viewer.loaded;
    const key = refusal === undefined ? (definition.retainKey?.(next) ?? next.place) : `refused:${next.place}`;
    if (current?.key === key && !pending) {
      const live = current;
      live.view = guarded(live.host, definition.title, () => {
        live.view.update?.(next);
        return live.view;
      });
      return;
    }
    if (current) hide(current);
    const kept = retained.get(key);
    if (kept) {
      current = kept;
      kept.host.style.display = '';
      kept.view.setActive?.(visible);
      kept.view = guarded(kept.host, definition.title, () => {
        kept.view.update?.(next);
        return kept.view;
      });
      return;
    }
    const host = document.createElement('div');
    host.className = 'soda-place';
    host.dataset.place = next.place;
    viewsLayer.append(host);
    const isRetained = refusal === undefined && definition.retainKey !== undefined;
    if (isRetained && key.startsWith('request:')) trimRequests();
    const view = guarded(host, definition.title, () => {
      if (pending) return noticePlace(host, definition.title, 'Checking your access…');
      if (refusal !== undefined) return noticePlace(host, definition.title, refusal);
      return definition.mount(host, placeContext, next);
    });
    current = {key: pending ? `pending:${next.place}` : key, host, view, retained: isRetained};
    if (isRetained) retained.set(key, current);
    if (!visible) view.setActive?.(false);
  };

  const route = (address: string) => {
    const forward = settingsOnly ? undefined : forwardedAddress(address, prefix);
    if (forward) {
      // Settings → SodaOS is another host page (inside the settings layout).
      if (placeFromPath(forward, prefix)?.place.place === 'settings') {
        location.replace(forward);
        return;
      }
      history.replaceState(history.state, '', forward);
      address = forward;
    }
    const parsed = placeFromPath(address, prefix);
    if (parsed && parsed.slug !== slug) {
      // A different repository's place: nothing of this one may stay alive.
      for (const live of retained.values()) release(live);
      if (current && !current.retained) release(current);
      current = undefined;
      showPlace(undefined);
      return;
    }
    // The settings page shows only settings.
    showPlace(settingsOnly && parsed?.place.place !== 'settings' ? undefined : parsed?.place);
  };

  const setVisible = (value: boolean) => {
    visible = value;
    current?.view.setActive?.(value);
    drawChrome();
  };

  const releaseAll = () => {
    if (current && !current.retained) release(current);
    for (const live of retained.values()) release(live);
    current = undefined;
  };

  route(location.pathname + location.search + location.hash);
  readViewer(placeContext)
    .then((read) => Object.assign(viewer, read))
    .catch(() => Object.assign(viewer, {loaded: true}))
    .finally(() => {
      if (retired || !page.isConnected) return;
      // A place that waited for the viewer's standing mounts now.
      if (current?.key.startsWith('pending:')) {
        release(current);
        current = undefined;
      }
      showPlace(place);
    });

  return {
    update(presentation: ContributionPresentation) {
      if (typeof presentation.route === 'string') route(presentation.route);
      setVisible(presentation.visible);
    },
    suspend() {
      setVisible(false);
    },
    resume(validation: ResumeValidation) {
      if (validation.sessionGeneration === transport.generation) setVisible(true);
    },
    retire() {
      retired = true;
      visible = false;
      releaseAll();
      drawChrome();
    },
    dispose() {
      retired = true;
      releaseAll();
      render(html``, bar);
      render(html``, providers);
      page.remove();
      styles.forEach((el) => el.remove());
      transport.dispose();
    },
  };
}
