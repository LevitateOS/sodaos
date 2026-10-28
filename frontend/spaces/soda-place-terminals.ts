// Terminals place: shells and agent sessions for code writers (M16, M18,
// lane D). The entry keeps this view alive while other places show (SPEC M2).
import {placeholderPlace} from './soda-place-view.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountTerminalsPlace: PlaceMount = (host) =>
  placeholderPlace(host, 'Terminals', 'Terminals and agent sessions for this repository will open here.');
