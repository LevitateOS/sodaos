import {fountainFeedContext} from './soda-factory-feed-host.js';
import type {FactoryFeedContext} from '@factory/product/frontend/factory/feed/feed.js';
// Feed place (page root): the repository's role cards and comments. Lane A
// (M4, M5) owns the feed's content; this file only mounts it.
import {html, render} from 'lit';
import '@factory/product/frontend/factory/feed/feed.js';
import type {Place} from './soda-places.js';
import type {PlaceMount} from './soda-place-view.js';

export const mountFeedPlace: PlaceMount = (host, context, initial) => {
  let post = initial.place === 'feed' ? initial.post : undefined;
  // New request lands here: a post is how a request starts.
  let compose = initial.place === 'feed' && initial.compose === true;
  let active = true;
  let disposed = false;
  let resolved: FactoryFeedContext | undefined;
  let pending: AbortController | undefined;
  let error = '';
  // The feed polls only while it has a context; a hidden feed has none.
  const draw = () =>
    render(
      html`${error ? html`<p role="alert">${error}</p>` : !resolved ? html`<p role="status">Loading feed…</p>` : ''}
        <soda-factory-feed
          .context=${active ? resolved : undefined}
          .post=${post}
          .compose=${compose}
        ></soda-factory-feed>`,
      host
    );
  const load = () => {
    if (!active || disposed || resolved || pending) return;
    const request = new AbortController();
    pending = request;
    error = '';
    void fountainFeedContext(
      context.transport,
      {
        repository: context.repository,
        repositoryName: context.slug,
        global: false,
        prefix: context.prefix,
      },
      request.signal
    ).then(
      (value) => {
        if (disposed || pending !== request || request.signal.aborted) return;
        resolved = value;
        pending = undefined;
        draw();
      },
      () => {
        if (disposed || pending !== request || request.signal.aborted) return;
        pending = undefined;
        error = 'Repository information is unavailable.';
        draw();
      }
    );
  };
  draw();
  load();
  return {
    update(place: Place) {
      if (place.place !== 'feed') return;
      post = place.post;
      compose = place.compose === true;
      draw();
    },
    setActive(value: boolean) {
      active = value;
      if (!active) {
        pending?.abort();
        pending = undefined;
      } else load();
      draw();
    },
    dispose() {
      disposed = true;
      pending?.abort();
      render(html``, host);
    },
  };
};
