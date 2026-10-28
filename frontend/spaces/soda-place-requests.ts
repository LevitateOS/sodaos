// Requests place: the viewer's requests in this repository (M8, lane B).
import {placeholderPlace} from './soda-place-view.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountRequestsPlace: PlaceMount = (host) =>
  placeholderPlace(host, 'Requests', 'Your requests for this repository will be listed here.');
