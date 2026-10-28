// App place: launch rows and verified builds for this repository (M13).
import {html, render} from 'lit';
import './sodaspaces-applications.js';
import type {Place} from './soda-places.js';
import type {PlaceContext, PlaceMount} from './soda-place-view.js';

function applicationRequest(context: PlaceContext) {
  return async (path: string, body?: Record<string, unknown>, signal?: AbortSignal) => {
    const response = await context.transport.request(path.replace(/^\/api\//, ''), {
      method: body ? 'POST' : 'GET',
      signal: signal ?? null,
      ...(body ? {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)} : {}),
    });
    const value = await response.json();
    if (!response.ok) throw Error(value.error?.message ?? 'Application request failed.');
    return value;
  };
}

export const mountAppPlace: PlaceMount = (host, context, initial) => {
  const request = applicationRequest(context);
  let delivery = initial.place === 'app' ? (initial.delivery ?? '') : '';
  let active = true;
  const draw = () =>
    render(
      html`<soda-applications
        .request=${request}
        .active=${active}
        .sessionKey=${context.transport.generation}
        .expanded=${true}
        .repository=${context.repository}
        .requestedDelivery=${delivery}
      ></soda-applications>`,
      host
    );
  draw();
  return {
    update(place: Place) {
      if (place.place !== 'app') return;
      delivery = place.delivery ?? '';
      draw();
    },
    setActive(value: boolean) {
      active = value;
      draw();
    },
    dispose: () => render(html``, host),
  };
};
