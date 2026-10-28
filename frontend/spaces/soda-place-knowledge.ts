// Knowledge place: decisions and lessons (M15).
import {html, render} from 'lit';
import './sodaspaces-factory-library.js';
import type {Place} from './soda-places.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountKnowledgePlace: PlaceMount = (host, context, initial) => {
  let item = initial.place === 'knowledge' ? initial.item : undefined;
  const draw = () =>
    render(
      html`<soda-factory-library
        .repository=${context.repository}
        .itemID=${item}
        .request=${context.transport.request}
      ></soda-factory-library>`,
      host
    );
  draw();
  return {
    update(place: Place) {
      if (place.place !== 'knowledge') return;
      item = place.item;
      draw();
    },
    dispose: () => render(html``, host),
  };
};
