// Feed place (page root): the repository's role cards and comments. Lane A
// (M4, M5) owns the feed's content; this file only mounts it.
import {html, render} from 'lit';
import './sodaspaces-factory-feed.js';
import type {Place} from './soda-places.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountFeedPlace: PlaceMount = (host, context, initial) => {
  let post = initial.place === 'feed' ? initial.post : undefined;
  let active = true;
  // The feed polls only while it has a context; a hidden feed has none.
  const draw = () =>
    render(
      html`<soda-factory-feed
        .context=${
          active
            ? {
                transport: context.transport,
                repository: context.repository,
                repositoryName: context.slug,
                global: false,
                prefix: context.prefix,
              }
            : undefined
        }
        .post=${post}
      ></soda-factory-feed>`,
      host
    );
  draw();
  return {
    update(place: Place) {
      if (place.place !== 'feed') return;
      post = place.post;
      draw();
    },
    setActive(value: boolean) {
      active = value;
      draw();
    },
    dispose: () => render(html``, host),
  };
};
