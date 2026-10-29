// The contract between the repository entry (soda-repository-entry.ts) and
// each place's view. Every place mounts from its own soda-place-*.ts file, so
// lanes can build a view without touching the entry.
import {html, render} from 'lit';
import type {PreparedExtensionMount} from './soda-extension.js';
import type {Place, SodaPlaces} from './soda-places.js';

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
  /** Native repository id (display only, never authority). */
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
  /**
   * True when this live view already shows `place` under a different key
   * (New request after its first save becomes that request's page); the
   * entry then keeps the same instance under the new key.
   */
  claim?(place: Place): boolean;
  dispose(): void;
}

export type PlaceMount = (host: HTMLElement, context: PlaceContext, place: Place) => PlaceView;

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
