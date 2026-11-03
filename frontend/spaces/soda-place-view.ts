// The contract between the repository entry (soda-repository-entry.ts) and
// each place's view. Every place mounts from its own soda-place-*.ts file, so
// lanes can build a view without touching the entry.
import {html, render} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {Place, SodaPlaces} from './soda-places.js';
import {fountainRepository} from './soda-factory-feed-host.js';

/** The viewer's standing on this repository, read once at mount. */
export interface PlaceViewer {
  /** False until the permission read finished; every power below is false until then. */
  loaded: boolean;
  /** Code write: Terminals. */
  codeWrite: boolean;
  /** Repository admin or environment administrator: New request, SodaOS settings. */
  admin: boolean;
  /** The configured SodaOS operator. */
  operator: boolean;
}

export interface PlaceContext {
  transport: PreparedExtensionMount;
  /** Installation prefix (appSubUrl) for native links. */
  prefix: string;
  /** Native Fountain locator for inventory, Projects and native navigation. */
  repository: string;
  /** owner/repo */
  slug: string;
  /** The only builder of SodaOS addresses for this repository. */
  places: SodaPlaces;
  viewer: PlaceViewer;
  /** Host navigation to a same-origin path (an in-shell swap for SodaOS places). */
  navigate(path: string, options?: {replace?: boolean}): void;
}

/**
 * What a mounted place view returns. The entry calls update when the route
 * stays on the same view with new parameters, setActive(false) while another
 * place shows or the host conceals the page, and dispose exactly once.
 */
export interface PlaceView {
  update?(place: Place): void;
  setActive?(active: boolean): void;
  dispose(): void;
}

export type PlaceMount = (host: HTMLElement, context: PlaceContext, place: Place) => PlaceView;

export interface FactoryPlaceContext extends PlaceContext {
  /** Local Factory identity, resolved separately from the native locator. */
  factoryRepository: string;
  factoryActor: string;
  factoryHost: string;
}

/** Resolve the native host locator before mounting a view that reads Factory records. */
export function withFactoryRepository(
  mount: (host: HTMLElement, context: FactoryPlaceContext, place: Place) => PlaceView
): PlaceMount {
  return (host, context, initial) => {
    let place = initial;
    let active = true;
    let disposed = false;
    let view: PlaceView | undefined;
    let pending: AbortController | undefined;
    const load = () => {
      if (!active || disposed || view || pending) return;
      const read = new AbortController();
      pending = read;
      noticePlace(host, 'SodaOS', 'Loading repository information…');
      void fountainRepository(context.transport, context.repository, read.signal).then(
        (resolved) => {
          if (disposed || pending !== read || read.signal.aborted) return;
          pending = undefined;
          render(html``, host);
          try {
            if (resolved.name !== context.slug) throw Error('Repository name changed. Refresh to reopen it.');
            view = mount(
              host,
              {
                ...context,
                factoryRepository: resolved.repository,
                factoryActor: resolved.actor,
                factoryHost: resolved.host,
              },
              place
            );
          } catch (error) {
            console.error(error);
            noticePlace(host, 'SodaOS', 'This SodaOS place could not load.');
          }
        },
        () => {
          if (disposed || pending !== read || read.signal.aborted) return;
          pending = undefined;
          noticePlace(host, 'SodaOS', 'Repository information is unavailable.');
        }
      );
    };
    load();
    return {
      update(next) {
        place = next;
        view?.update?.(next);
      },
      setActive(value) {
        active = value;
        if (!value) {
          pending?.abort();
          pending = undefined;
        }
        view?.setActive?.(value);
        if (value) load();
      },
      dispose() {
        disposed = true;
        pending?.abort();
        view?.dispose();
        render(html``, host);
      },
    };
  };
}

/** A labelled placeholder for a place another lane fills in. */
export function placeholderPlace(host: HTMLElement, title: string, detail: string): PlaceView {
  render(
    html`<section class="soda-place-placeholder" aria-label=${title}>
      <h2>${title}</h2>
      <p>${detail}</p>
    </section>`,
    host
  );
  return {dispose: () => render(html``, host)};
}

/** A refusal shown instead of a view the viewer may not use. */
export function noticePlace(host: HTMLElement, title: string, notice: string): PlaceView {
  render(
    html`<section class="soda-place-notice" aria-label=${title}>
      <h2>${title}</h2>
      <p role="status">${notice}</p>
    </section>`,
    host
  );
  return {dispose: () => render(html``, host)};
}
