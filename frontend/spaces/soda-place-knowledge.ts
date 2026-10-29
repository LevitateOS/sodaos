// Knowledge place: decisions and lessons (M15).
import {html, render} from 'lit';
import './sodaspaces-factory-library.js';
import type {Place} from './soda-places.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountKnowledgePlace: PlaceMount = (host, context, initial) => {
  let item = initial.place === 'knowledge' ? initial.item : undefined;
  // "Open source conversation" goes back to the post's card in Feed.
  const openSource = (event: Event) => {
    const detail = (event as CustomEvent<{repository?: unknown; id?: unknown}>).detail;
    if (detail?.repository !== context.repository || typeof detail.id !== 'string' || !/^[a-f0-9]{32}$/.test(detail.id))
      return;
    context.navigate(context.places.feed(detail.id));
  };
  const draw = () =>
    render(
      html`<soda-factory-library
        .repository=${context.repository}
        .itemID=${item}
        .request=${context.transport.request}
        @library-source-open=${openSource}
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
