// Factory status place: the issue-number lookup and one issue's page (M20,
// lane C).
import {placeholderPlace} from './soda-place-view.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountFactoryStatusPlace: PlaceMount = (host, _context, place) =>
  place.place === 'factory-status' && place.issue
    ? placeholderPlace(host, 'Factory status', `Factory status for issue #${place.issue} will open here.`)
    : placeholderPlace(host, 'Factory status', 'Look up the Factory status of an issue here.');
