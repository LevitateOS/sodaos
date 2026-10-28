// Request place: New request (the composer, M9) and one request's page (M11),
// lane B. The entry keeps this view alive per request id while other places
// show (SPEC M2), so the composer and its polls survive a place switch.
import {newRequestID} from './soda-places.js';
import {placeholderPlace} from './soda-place-view.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountRequestPlace: PlaceMount = (host, _context, place) => {
  const request = place.place === 'request' ? place.request : '';
  return request === newRequestID
    ? placeholderPlace(host, 'New request', 'The composer for this repository will open here.')
    : placeholderPlace(host, 'Request', `Request ${request} will open here.`);
};
