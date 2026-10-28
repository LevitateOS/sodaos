// Feed place (page root): the repository's role cards and comments. Lane A
// (M4, M5) owns the feed's content; this file only mounts it.
import {html, render} from 'lit';
import './sodaspaces-factory-feed.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountFeedPlace: PlaceMount = (host, context) => {
  // The feed polls only while it has a context; a hidden feed has none.
  const draw = (active: boolean) =>
    render(
      html`<soda-factory-feed
        .context=${
          active
            ? {
                transport: context.transport,
                repository: context.repository,
                repositoryName: context.slug,
                global: false,
              }
            : undefined
        }
      ></soda-factory-feed>`,
      host
    );
  draw(true);
  return {setActive: draw, dispose: () => render(html``, host)};
};
