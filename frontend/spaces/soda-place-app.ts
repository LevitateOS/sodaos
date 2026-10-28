// App place (M13): Company and Preview launch rows for this repository, the
// build a Feed card points at (?delivery) with Release this build, builds
// that need attention, and the app administrator's and operator's controls.
// Open app ↗ always opens the app at its own address in a new tab.
import {html, render} from 'lit';
import './sodaspaces-applications.js';
// Registers <soda-factory-verification-retry> (Retry verification, Request
// repair); the Workspace Work view still uses it until install E.
import './sodaspaces-project-work.js';
import {spacesResponse} from './sodaspaces-inventory-response.js';
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

// This repository's recorded issue-zero (application verification) runs, read
// from the spaces collection exactly as the Workspace did: only an account that
// may execute reads verification state, so reviewers see no refusals.
async function verificationRuns(context: PlaceContext, signal: AbortSignal): Promise<string[]> {
  const seen = new Set<string>();
  let after = '';
  for (;;) {
    const response = await context.transport.request('spaces' + (after ? '?after=' + encodeURIComponent(after) : ''), {
      method: 'GET',
      signal,
    });
    if (!response.ok) return [];
    const page = spacesResponse(await response.json());
    const space = page.items.find((item) => item.environment.repository_id === context.repository);
    if (space)
      return space.execution_allowed && !space.authority_unavailable
        ? space.factory_runs.filter((run) => run.reconciled && run.issue === undefined).map((run) => run.id)
        : [];
    if (page.complete || !page.nextAfter || seen.has(page.nextAfter)) return [];
    seen.add(page.nextAfter);
    after = page.nextAfter;
  }
}

export const mountAppPlace: PlaceMount = (host, context, initial) => {
  const request = applicationRequest(context);
  let delivery = initial.place === 'app' ? (initial.delivery ?? '') : '';
  let active = true;
  let runs: string[] = [];
  const reads = new AbortController();
  // The delivery panel's "Follow this delivery in Feed" asks for the Feed.
  const openFeed = () => context.navigate(context.places.feed());
  const draw = () =>
    render(
      html`<soda-applications
          .request=${request}
          .active=${active}
          .sessionKey=${context.transport.generation}
          .expanded=${true}
          .layout=${'repository'}
          .prefix=${context.prefix}
          .places=${context.places}
          .repository=${context.repository}
          .requestedDelivery=${delivery}
          @soda-project-feed-open=${openFeed}
        ></soda-applications>
        <section class="soda-app-attention" aria-label="Builds that need attention">
          <h2>Builds that need attention</h2>
          <soda-factory-verification-retry
            .context=${active && runs.length ? {transport: context.transport, repository: context.repository, runs} : undefined}
          ></soda-factory-verification-retry>
        </section>`,
      host
    );
  draw();
  verificationRuns(context, reads.signal).then(
    (found) => {
      if (reads.signal.aborted) return;
      runs = found.slice(0, 64);
      draw();
    },
    () => undefined
  );
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
    dispose: () => {
      reads.abort();
      render(html``, host);
    },
  };
};
